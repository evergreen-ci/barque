package operations

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/barque/rest"
	"github.com/evergreen-ci/barque/units"
	"github.com/evergreen-ci/gimlet"
	"github.com/mongodb/amboy"
	amboyRest "github.com/mongodb/amboy/rest"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/recovery"
	"github.com/mongodb/jasper/remote"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func Service() cli.Command {
	return cli.Command{
		Name:  "service",
		Usage: "run the barque service",
		Subcommands: []cli.Command{
			startWebServer(),
			startWorkers(),
		},
	}
}

func startWorkers() cli.Command {
	return cli.Command{
		Name:  "workers",
		Usage: "start a service that only processes background jobs",
		Flags: mergeFlags(baseFlags(), dbFlags(), adminFlags()),
		Action: func(c *cli.Context) error {
			conf := &barque.Configuration{
				MongoDBURI:    c.String(dbURIFlag),
				DatabaseName:  c.String(dbNameFlag),
				NumWorkers:    c.Int(numWorkersFlag),
				DisableQueues: c.Bool(disableWorkersFlag),
				QueueName:     barque.QueueName,
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go signalListener(ctx, cancel)

			env, err := barque.NewEnvironment(ctx, conf)
			if err != nil {
				return errors.WithStack(err)
			}
			barque.SetEnvironment(env)

			if err = setupLogging(ctx, env); err != nil {
				return errors.WithStack(err)
			}

			var adminWait gimlet.WaitFunc

			if !c.Bool(disableAdminFlagName) {
				adminWait, err = runAdminService(ctx, env, c.Int(adminPortFlagName))
				if err != nil {
					return errors.WithStack(err)
				}
			}

			if !c.Bool(disableBackgroundJobCreation) {
				if err := units.StartCrons(ctx, env); err != nil {
					return errors.WithStack(err)
				}
			}

			amboy.WaitInterval(ctx, env.RemoteQueue(), time.Second)
			doWait(ctx, adminWait)

			return env.Close(ctx)
		},
	}

}

func startWebServer() cli.Command {
	const (
		servicePortFlag = "port"
	)

	return cli.Command{
		Name:  "web",
		Usage: "run the barque service",
		Flags: mergeFlags(baseFlags(), adminFlags(), dbFlags(
			cli.IntFlag{
				Name:   joinFlagNames(servicePortFlag, "p"),
				Usage:  "specify a port to run the REST service on",
				Value:  3000,
				EnvVar: envVarRESTPort,
			},
		)),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go signalListener(ctx, cancel)

			conf := &barque.Configuration{
				MongoDBURI:    c.String(dbURIFlag),
				DatabaseName:  c.String(dbNameFlag),
				NumWorkers:    c.Int(numWorkersFlag),
				DisableQueues: c.Bool(disableWorkersFlag),
				QueueName:     barque.QueueName,
			}

			env, err := barque.NewEnvironment(ctx, conf)
			if err != nil {
				return errors.WithStack(err)
			}
			barque.SetEnvironment(env)

			if err = setupLogging(ctx, env); err != nil {
				return errors.WithStack(err)
			}

			var adminWait gimlet.WaitFunc

			if !c.Bool(disableAdminFlagName) {
				adminWait, err = runAdminService(ctx, env, c.Int(adminPortFlagName))
				if err != nil {
					return errors.Wrap(err, "failed to start admin service")
				}
			}
			if !c.Bool(disableBackgroundJobCreation) {
				if err = units.StartCrons(ctx, env); err != nil {
					return errors.Wrap(err, "failed to start crons")
				}
			}

			appWait, err := runRestService(ctx, env, c.Int(servicePortFlag))
			if err != nil {
				return errors.Wrap(err, "failed to start rest service")
			}

			appWait(ctx)
			doWait(ctx, adminWait)

			return env.Close(ctx)
		},
	}
}

func signalListener(ctx context.Context, trigger context.CancelFunc) {
	defer recovery.LogStackTraceAndContinue("graceful shutdown")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	select {
	case <-sigChan:
		grip.Debug("received signal")
	case <-ctx.Done():
		grip.Debug("context canceled")
	}

	trigger()
}

func doWait(ctx context.Context, wait gimlet.WaitFunc) {
	if wait == nil {
		return
	}

	wait(ctx)
}

func runRestService(ctx context.Context, env barque.Environment, port int) (gimlet.WaitFunc, error) {
	app, err := rest.New(env)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := app.SetPort(port); err != nil {
		return nil, errors.WithStack(err)
	}

	return app.BackgroundRun(ctx)
}

func runAdminService(ctx context.Context, env barque.Environment, port int) (gimlet.WaitFunc, error) {
	localPool, ok := env.LocalQueue().Runner().(amboy.AbortableRunner)
	if !ok {
		return nil, errors.New("local pool is not configured with an abortable pool")
	}
	remotePool, ok := env.RemoteQueue().Runner().(amboy.AbortableRunner)
	if !ok {
		return nil, errors.New("remote pool is not configured with an abortable pool")
	}

	app := gimlet.NewApp()

	if err := app.SetHost("localhost"); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := app.SetPort(port); err != nil {
		return nil, errors.WithStack(err)
	}
	app.NoVersions = true

	app.AddMiddleware(gimlet.MakeRecoveryLogger())

	localReporting := amboyRest.NewReportingService(env.LocalReporter()).App()
	localReporting.SetPrefix("/amboy/local/reporting")
	groupReporting := amboyRest.NewReportingService(env.GroupReporter()).App()
	groupReporting.SetPrefix("/amboy/group/reporting")
	remoteReporting := amboyRest.NewReportingService(env.RemoteReporter()).App()
	remoteReporting.SetPrefix("/amboy/remote/reporting")

	localAbort := amboyRest.NewManagementService(localPool).App()
	localAbort.SetPrefix("/amboy/local/pool")
	remoteAbort := amboyRest.NewManagementService(remotePool).App()
	remoteAbort.SetPrefix("/amboy/remote/pool")
	groupAbort := amboyRest.NewManagementGroupService(env.QueueGroup()).App()
	groupAbort.SetPrefix("/amboy/group/pool")

	jpm := remote.NewRestService(env.Jasper())
	jpm.SetDisableCachePruning(true)
	jpmapp := jpm.App(ctx)
	jpmapp.SetPrefix("/jasper")

	err := app.Merge(gimlet.GetPProfApp(), jpmapp, localReporting, groupReporting, remoteReporting, localAbort, remoteAbort, groupAbort)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return app.BackgroundRun(ctx)
}
