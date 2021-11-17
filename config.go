package barque

import (
	"errors"
	"os"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	yaml "gopkg.in/yaml.v2"
)

type Configuration struct {
	MongoDBURI         string
	DatabaseName       string
	QueueName          string
	MongoDBDialTimeout time.Duration
	SocketTimeout      time.Duration
	DisableQueues      bool
	NumWorkers         int
	DBAuthFile         string
}

func (c *Configuration) Validate() error {
	catcher := grip.NewBasicCatcher()

	if c.MongoDBURI == "" {
		catcher.Add(errors.New("must specify a mongodb url"))
	}
	if c.NumWorkers < 1 {
		catcher.Add(errors.New("must specify a valid number of amboy workers"))
	}
	if c.MongoDBDialTimeout <= 0 {
		c.MongoDBDialTimeout = 2 * time.Second
	}
	if c.SocketTimeout <= 0 {
		c.SocketTimeout = time.Minute
	}
	if c.QueueName == "" {
		c.QueueName = "barque.service"
	}

	return catcher.Resolve()
}

func (c *Configuration) GetQueueOptions() queue.MongoDBOptions {
	return queue.MongoDBOptions{
		URI:                      c.MongoDBURI,
		DB:                       c.DatabaseName,
		Priority:                 true,
		Format:                   amboy.BSON2,
		WaitInterval:             time.Second,
		SkipQueueIndexBuilds:     true,
		SkipReportingIndexBuilds: true,
	}
}

func (c *Configuration) GetQueueGroupOptions() queue.MongoDBOptions {
	opts := c.GetQueueOptions()
	opts.UseGroups = true
	opts.GroupName = c.QueueName
	return opts
}

func (c *Configuration) HasAuth() bool {
	return c.DBAuthFile != ""
}

type dbCreds struct {
	DBUser string `yaml:"mdb_database_username"`
	DBPwd  string `yaml:"mdb_database_password"`
}

func (c *Configuration) GetAuth() (string, string, error) {
	return getAuthFromYAML(c.DBAuthFile)
}

func getAuthFromYAML(authFile string) (string, string, error) {

	file, err := os.Open(authFile)
	if err != nil {
		grip.Warning(message.Fields{"message": "error opening authfile",
			"authfile": authFile,
			"err":      err})
		return "", "", err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	creds := &dbCreds{}
	if err := decoder.Decode(creds); err != nil {
		return "", "", err
	}

	return creds.DBUser, creds.DBPwd, nil
}
