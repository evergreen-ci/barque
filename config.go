package barque

import (
	"errors"
	"time"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/grip"
)

type Configuration struct {
	MongoDBURI         string
	DatabaseName       string
	QueueName          string
	MongoDBDialTimeout time.Duration
	SocketTimeout      time.Duration
	DisableQueues      bool
	NumWorkers         int
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
		URI:          c.MongoDBURI,
		DB:           c.DatabaseName,
		Priority:     true,
		Format:       amboy.BSON2,
		WaitInterval: time.Second,
	}
}

func (c *Configuration) GetQueueGroupOptions() queue.MongoDBOptions {
	opts := c.GetQueueOptions()
	opts.UseGroups = true
	opts.GroupName = c.QueueName
	return opts
}
