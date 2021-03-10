package operations

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/barque/model"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

func Admin() cli.Command {
	return cli.Command{
		Name: "admin",
		Subcommands: []cli.Command{
			Config(),
		},
	}
}

func Config() cli.Command {
	return cli.Command{
		Name: "config",
		Subcommands: []cli.Command{
			DumpConf(),
			LoadConf(),
		},
	}

}

func DumpConf() cli.Command {
	return cli.Command{
		Name:  "dump",
		Flags: dbFlags(),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			conf := &barque.Configuration{
				MongoDBURI:    c.String(dbURIFlag),
				DatabaseName:  c.String(dbNameFlag),
				NumWorkers:    1,
				DisableQueues: true,
				DBAuthFile:    c.String(dbCredsFileFlag),
			}

			env, err := barque.NewEnvironment(ctx, conf)
			if err != nil {
				return errors.WithStack(err)
			}
			barque.SetEnvironment(env)
			defer func() { grip.Error(env.Close(ctx)) }()

			cfg, err := model.FindConfiguration(ctx, env)
			if err != nil {
				return errors.Wrap(err, "problem finding configuration")
			}

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return errors.Wrap(err, "problem marshaling config")
			}

			file, err := os.Create(fmt.Sprintf("barque.config.%d.yaml", time.Now().Unix()))
			if err != nil {
				return errors.Wrap(err, "problem creating config file")
			}

			if _, err = file.Write(out); err != nil {
				grip.Error(file.Close())
				return errors.Wrap(err, "problem writing data")
			}

			if err = file.Close(); err != nil {
				return errors.WithStack(err)
			}

			return nil
		},
	}
}

func LoadConf() cli.Command {
	const pathFlagName = "path"
	return cli.Command{
		Name: "load",
		Flags: dbFlags(cli.StringFlag{
			Name:  pathFlagName,
			Usage: "specify the path to upload",
			Value: "barque.config.yaml",
		}),
		Action: func(c *cli.Context) error {
			fn := c.String(pathFlagName)
			if _, err := os.Stat(fn); os.IsNotExist(err) {
				return errors.Errorf("no file named '%s'", fn)
			}

			file, err := os.Open(fn)
			if err != nil {
				return errors.Wrap(err, "problem opening input file")
			}
			defer file.Close()

			data, err := ioutil.ReadAll(file)
			if err != nil {
				return errors.Wrap(err, "problem reading data from file")
			}

			cfg := &model.Configuration{}
			if err = yaml.Unmarshal(data, cfg); err != nil {
				return errors.Wrap(err, "problem marshaling data from file")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			conf := &barque.Configuration{
				MongoDBURI:    c.String(dbURIFlag),
				DatabaseName:  c.String(dbNameFlag),
				NumWorkers:    1,
				DisableQueues: true,
				DBAuthFile:    c.String(dbCredsFileFlag),
			}

			env, err := barque.NewEnvironment(ctx, conf)
			if err != nil {
				return errors.WithStack(err)
			}
			barque.SetEnvironment(env)
			defer func() { grip.Error(env.Close(ctx)) }()

			if err := cfg.Save(ctx, env); err != nil {
				return errors.New("problem saving file")
			}

			return nil
		},
	}

}
