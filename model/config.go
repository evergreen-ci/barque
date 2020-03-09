package model

import (
	"context"

	"github.com/evergreen-ci/barque"
	"github.com/mongodb/anser/bsonutil"
	"github.com/mongodb/grip/send"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	configCollection = "configuration"
	configID         = "barque-application-configuration"
)

type Configuration struct {
	ID          string                    `bson:"_id" json:"id" yaml:"id"`
	Splunk      send.SplunkConnectionInfo `bson:"splunk" json:"splunk" yaml:"splunk"`
	Flags       OperationalFlags          `bson:"flags" json:"flags" yaml:"flags"`
	Slack       SlackConfig               `bson:"slack" json:"slack" yaml:"slack"`
	LDAP        LDAPConfig                `bson:"ldap" json:"ldap" yaml:"ldap"`
	NaiveAuth   NaiveAuthConfig           `bson:"naive_auth" json:"naive_auth" yaml:"naive_auth"`
	Repobuilder RepobuilderConfig         `bson:"repobuilder" json:"repobuilder" yaml:"repobuilder"`
}

func FindConfiguration(ctx context.Context, env barque.Environment) (*Configuration, error) {
	conf := &Configuration{}
	res := env.DB().Collection(configCollection).FindOne(ctx, bson.M{"_id": configID})

	if err := res.Decode(conf); err == mongo.ErrNoDocuments {
		if err = conf.Save(ctx, env); err != nil {
			return nil, errors.Wrap(err, "problem saving new config")
		}
		return conf, nil
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	conf.Flags.env = env

	return conf, nil
}

func (conf *Configuration) Save(ctx context.Context, env barque.Environment) error {
	if conf.ID == "" {
		conf.ID = configID
	}

	if conf.ID != configID {
		return errors.Errorf("configuration id='%s' is unexpected", conf.ID)
	}

	res, err := env.DB().Collection(configCollection).ReplaceOne(ctx, bson.M{"_id": conf.ID}, conf,
		options.Replace().SetUpsert(true))

	if err != nil {
		return errors.Wrap(err, "problem saving configuration object")
	}

	if res.ModifiedCount+res.UpsertedCount != 1 {
		return errors.New("did not save configuration object")
	}

	conf.Flags.env = env

	return nil
}

var (
	confIDKey        = bsonutil.MustHaveTag(Configuration{}, "ID")
	confSplunkKey    = bsonutil.MustHaveTag(Configuration{}, "Splunk")
	confFlagsKey     = bsonutil.MustHaveTag(Configuration{}, "Flags")
	confSlackKey     = bsonutil.MustHaveTag(Configuration{}, "Slack")
	confLDAPKey      = bsonutil.MustHaveTag(Configuration{}, "LDAP")
	confNaiveAuthKey = bsonutil.MustHaveTag(Configuration{}, "NaiveAuth")
)

type SlackConfig struct {
	Options *send.SlackOptions `bson:"options" json:"options" yaml:"options"`
	Token   string             `bson:"token" json:"token" yaml:"token"`
	Level   string             `bson:"level" json:"level" yaml:"level"`
}

var (
	slackConfigOptionsKey = bsonutil.MustHaveTag(SlackConfig{}, "Options")
	slackConfigTokenKey   = bsonutil.MustHaveTag(SlackConfig{}, "Token")
	slackConfigLevelKey   = bsonutil.MustHaveTag(SlackConfig{}, "Level")
)

// LDAPConfig contains settings for interacting with an LDAP server.
type LDAPConfig struct {
	URL          string `bson:"url" json:"url" yaml:"url"`
	Port         string `bson:"port" json:"port" yaml:"port"`
	UserPath     string `bson:"path" json:"path" yaml:"path"`
	ServicePath  string `bson:"service_path" json:"service_path" yaml:"service_path"`
	UserGroup    string `bson:"user_group" json:"user_group" yaml:"user_group"`
	ServiceGroup string `bson:"service_group" json:"service_group" yaml:"service_group"`
}

var (
	ldapAuthConfigURLKey          = bsonutil.MustHaveTag(LDAPConfig{}, "URL")
	ldapAuthConfigPortKey         = bsonutil.MustHaveTag(LDAPConfig{}, "Port")
	ldapAuthConfigUserPathKey     = bsonutil.MustHaveTag(LDAPConfig{}, "UserPath")
	ldapAuthConfigServicePathKey  = bsonutil.MustHaveTag(LDAPConfig{}, "ServicePath")
	ldapAuthConfigGroupKey        = bsonutil.MustHaveTag(LDAPConfig{}, "UserGroup")
	ldapAuthConfigServiceGroupKey = bsonutil.MustHaveTag(LDAPConfig{}, "ServiceGroup")
)

type NaiveAuthConfig struct {
	AppAuth bool              `bson:"app_auth" json:"app_auth" yaml:"app_auth"`
	Users   []NaiveUserConfig `bson:"users" json:"users" yaml:"users"`
}

var (
	naiveAuthConfigAppAuthKey = bsonutil.MustHaveTag(NaiveAuthConfig{}, "AppAuth")
	naiveAuthConfigUsersKey   = bsonutil.MustHaveTag(NaiveAuthConfig{}, "Users")
)

type NaiveUserConfig struct {
	ID           string   `bson:"_id" json:"id" yaml:"id"`
	Name         string   `bson:"name" json:"name" yaml:"name"`
	EmailAddress string   `bson:"email" json:"email" yaml:"email"`
	Password     string   `bson:"password" json:"password" yaml:"password"`
	Key          string   `bson:"key" json:"key" yaml:"key"`
	AccessRoles  []string `bson:"roles,omitempty" json:"roles" yaml:"roles"`
	Invalid      bool     `bson:"invalid" json:"invalid" yaml:"invalid"`
}

var (
	naiveUserConfigIDKey           = bsonutil.MustHaveTag(NaiveUserConfig{}, "ID")
	naiveUserConfigNameKey         = bsonutil.MustHaveTag(NaiveUserConfig{}, "Name")
	naiveUserConfigEmailAddressKey = bsonutil.MustHaveTag(NaiveUserConfig{}, "EmailAddress")
	naiveUserConfigPasswordKey     = bsonutil.MustHaveTag(NaiveUserConfig{}, "Password")
	naiveUserConfigKeyKey          = bsonutil.MustHaveTag(NaiveUserConfig{}, "Key")
	naiveUserConfigAccessRolesKey  = bsonutil.MustHaveTag(NaiveUserConfig{}, "AccessRoles")
	naiveUserConfigInvalidKey      = bsonutil.MustHaveTag(NaiveUserConfig{}, "Invalid")
)

type RepobuilderConfig struct {
	Path    string         `bson:"path" json:"path" yaml:"path"`
	Buckets []BucketConfig `bson:"buckets,omitempty" json:"buckets" yaml:"buckets"`
}

var (
	repoBuilderConfPathKey    = bsonutil.MustHaveTag(RepobuilderConfig{}, "Path")
	repoBuilderConfBucketsKey = bsonutil.MustHaveTag(RepobuilderConfig{}, "Buckets")
)

func (c *RepobuilderConfig) GetBucketConfig(name string) (*BucketConfig, error) {
	for idx := range c.Buckets {
		if c.Buckets[idx].Name == name {
			return &c.Buckets[idx], nil
		}
	}

	return nil, errors.Errorf("could not find bucket configuration matching '%s'", name)
}

type BucketConfig struct {
	Name   string `bson:"name" json:"name" yaml:"name"`
	Key    string `bson:"key" json:"key" yaml:"key"`
	Secret string `bson:"secret" json:"secret" yaml:"secret"`
	Token  string `bson:"token" json:"token" yaml:"token"`
}

var (
	bucketConfNameKey   = bsonutil.MustHaveTag(BucketConfig{}, "Name")
	bucketConfKeyKey    = bsonutil.MustHaveTag(BucketConfig{}, "Key")
	bucketConfSecretKey = bsonutil.MustHaveTag(BucketConfig{}, "Secret")
	bucketConfTokenKey  = bsonutil.MustHaveTag(BucketConfig{}, "Token")
)
