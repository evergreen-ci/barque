package barque

import (
	"context"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/management"
	"github.com/mongodb/amboy/pool"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/anser/apm"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/mongodb/jasper"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewEnvironment(ctx context.Context, conf *Configuration) (Environment, error) {
	env := &envImpl{name: "primary", conf: conf}

	if err := env.conf.Validate(); err != nil {
		return nil, errors.WithStack(err)
	}

	initFuncs := []CloserFunc{
		// init functions have the same
		// type name
		env.initDB,
		env.initLocalQueue,
		env.initRemoteQueue,
		env.initQueueGroup,
		env.initJasper,
		env.initContext,
	}

	for _, initFn := range initFuncs {
		if err := initFn(ctx); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return env, nil
}

func (e *envImpl) initDB(ctx context.Context) error {
	var err error
	opts := options.Client().ApplyURI(e.conf.MongoDBURI).
		SetConnectTimeout(e.conf.MongoDBDialTimeout).
		SetSocketTimeout(e.conf.SocketTimeout).
		SetServerSelectionTimeout(e.conf.SocketTimeout).
		SetMonitor(apm.NewLoggingMonitor(ctx, time.Minute, apm.NewBasicMonitor(&apm.MonitorConfig{AllTags: true})).DriverAPM())
	if e.conf.HasAuth() {
		ymlUser, ymlPwd, err := e.conf.GetAuth()
		if err != nil {
			grip.Error(errors.Wrap(err, "problem getting auth from yaml authfile, attempting to connect without auth"))
		}
		if err == nil && ymlUser != "" {
			credential := options.Credential{
				Username: ymlUser,
				Password: ymlPwd,
			}
			opts.SetAuth(credential)
		}
	}
	e.client, err = mongo.NewClient(opts)
	if err != nil {
		return errors.Wrap(err, "problem constructing mongodb client")
	}

	if err = e.client.Ping(ctx, nil); err != nil {
		connctx, cancel := context.WithTimeout(ctx, e.conf.MongoDBDialTimeout)
		defer cancel()
		if err := e.client.Connect(connctx); err != nil {
			return errors.Wrap(err, "problem connecting to database")
		}
	}

	return nil
}

func (e *envImpl) initLocalQueue(ctx context.Context) error {
	if e.conf.DisableQueues {
		return nil
	}
	e.localQueue = queue.NewLocalLimitedSize(e.conf.NumWorkers, 1024)

	e.RegisterCloser("local-queue", true, func(ctx context.Context) error {
		if !amboy.WaitInterval(ctx, e.localQueue, 10*time.Millisecond) {
			grip.Critical(message.Fields{
				"message": "pending jobs failed to finish",
				"queue":   "system",
				"status":  e.localQueue.Stats(ctx),
			})
			return errors.New("failed to stop with running jobs")
		}
		e.localQueue.Runner().Close(ctx)
		return nil
	})

	if err := e.localQueue.SetRunner(pool.NewAbortablePool(e.conf.NumWorkers, e.localQueue)); err != nil {
		return errors.Wrap(err, "problem configuring worker pool for local queue")
	}

	if err := e.localQueue.Start(ctx); err != nil {
		return errors.Wrap(err, "problem starting remote queue")
	}

	e.localManager = management.NewQueueManager(e.localQueue)

	return nil
}

func (e *envImpl) initRemoteQueue(ctx context.Context) error {
	if e.conf.DisableQueues {
		return nil
	}

	opts := e.conf.GetQueueOptions()
	args := queue.MongoDBQueueCreationOptions{
		Size:    e.conf.NumWorkers,
		Name:    e.conf.QueueName,
		Ordered: false,
		Client:  e.client,
		MDB:     opts,
	}

	rq, err := queue.NewMongoDBQueue(ctx, args)
	if err != nil {
		return errors.Wrap(err, "problem setting main queue backend")
	}

	if err = rq.SetRunner(pool.NewAbortablePool(e.conf.NumWorkers, rq)); err != nil {
		return errors.Wrap(err, "problem configuring worker pool for main remote queue")
	}
	e.remoteQueue = rq
	e.RegisterCloser("application-queue", false, func(ctx context.Context) error {
		e.remoteQueue.Runner().Close(ctx)
		return nil
	})

	grip.Info(message.Fields{
		"message":  "configured a remote mongodb-backed queue",
		"db":       e.conf.DatabaseName,
		"prefix":   e.conf.QueueName,
		"priority": true})

	if err = e.remoteQueue.Start(ctx); err != nil {
		return errors.Wrap(err, "problem starting remote queue")
	}
	reporterOpts := management.DBQueueManagerOptions{
		Name:    e.conf.QueueName,
		Options: opts,
	}
	e.remoteManager, err = management.MakeDBQueueManager(ctx, reporterOpts, e.client)
	if err != nil {
		return errors.Wrap(err, "problem starting remote reporter")
	}

	return nil
}

func (e *envImpl) initQueueGroup(ctx context.Context) error {
	if e.conf.DisableQueues {
		return nil
	}

	opts := e.conf.GetQueueGroupOptions()
	args := queue.MongoDBQueueGroupOptions{
		Prefix:                    e.conf.QueueName,
		DefaultWorkers:            e.conf.NumWorkers,
		Ordered:                   false,
		BackgroundCreateFrequency: 10 * time.Minute,
		PruneFrequency:            10 * time.Minute,
		TTL:                       time.Minute,
	}

	var err error
	e.queueGroup, err = queue.NewMongoDBSingleQueueGroup(ctx, args, e.client, opts)
	if err != nil {
		return errors.Wrap(err, "problem starting remote queue group")
	}

	e.RegisterCloser("remote-queue-group", false, func(ctx context.Context) error {
		return errors.Wrap(e.queueGroup.Close(ctx), "problem waiting for remote queue group to close")
	})

	reporterOpts := management.DBQueueManagerOptions{
		Name:     e.conf.QueueName,
		Options:  opts,
		ByGroups: true,
	}
	e.groupManager, err = management.MakeDBQueueManager(ctx, reporterOpts, e.client)
	if err != nil {
		return errors.Wrap(err, "problem starting remote reporter")
	}

	return nil
}

func (e *envImpl) initJasper(_ context.Context) error {
	jpm, err := jasper.NewSelfClearingProcessManager(2048, false)
	if err != nil {
		return errors.WithStack(err)
	}
	e.jpm, err = jasper.MakeSynchronizedManager(jpm)
	if err != nil {
		return errors.WithStack(err)
	}

	e.RegisterCloser("jasper-manager", true, func(ctx context.Context) error {
		return errors.WithStack(e.jpm.Close(ctx))
	})

	return nil
}

func (e *envImpl) initContext(ctx context.Context) error {
	var capturedCtxCancel context.CancelFunc
	e.context, capturedCtxCancel = context.WithCancel(ctx)
	e.RegisterCloser("env-captured-context-cancel", true, func(_ context.Context) error {
		capturedCtxCancel()
		return nil
	})

	return nil
}
