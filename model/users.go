package model

import (
	"context"
	"time"

	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/gimlet"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/anser/bsonutil"
	"github.com/mongodb/anser/db"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const userCollection = "users"

// Stores user information in database, resulting in a cache for the LDAP user manager.
type User struct {
	ID           string     `bson:"_id"`
	Display      string     `bson:"display_name"`
	EmailAddress string     `bson:"email"`
	CreatedAt    time.Time  `bson:"created_at"`
	APIKey       string     `bson:"apikey"`
	SystemRoles  []string   `bson:"roles"`
	LoginCache   LoginCache `bson:"login_cache"`
	populated    bool
}

var (
	dbUserIDKey           = bsonutil.MustHaveTag(User{}, "ID")
	dbUserDisplayNameKey  = bsonutil.MustHaveTag(User{}, "Display")
	dbUserEmailAddressKey = bsonutil.MustHaveTag(User{}, "EmailAddress")
	dbUserAPIKeyKey       = bsonutil.MustHaveTag(User{}, "APIKey")
	dbUserSystemRolesKey  = bsonutil.MustHaveTag(User{}, "SystemRoles")
	dbUserLoginCacheKey   = bsonutil.MustHaveTag(User{}, "LoginCache")
)

type LoginCache struct {
	Token string    `bson:"token"`
	TTL   time.Time `bson:"ttl"`
}

var (
	loginCacheTokenKey = bsonutil.MustHaveTag(LoginCache{}, "Token")
	loginCacheTTLKey   = bsonutil.MustHaveTag(LoginCache{}, "TTL")
)

func FindUser(ctx context.Context, env barque.Environment, id string) (*User, error) {
	u := &User{ID: id}
	if err := u.Find(ctx, env); err != nil {
		return nil, errors.WithStack(err)
	}

	return u, nil
}

func (u *User) idQuery() interface{} { return bson.M{"_id": u.ID} }

func (u *User) IsNil() bool { return !u.populated }
func (u *User) Find(ctx context.Context, env barque.Environment) error {
	res := env.DB().Collection(userCollection).FindOne(ctx, u.idQuery())
	err := res.Err()
	if db.ResultsNotFound(err) {
		return errors.Wrapf(err, "could not find user %s in the database", u.Username())
	} else if err != nil {
		return errors.Wrap(err, "problem finding user")
	}
	u.populated = false
	if err = res.Decode(u); err != nil {
		return errors.Wrap(err, "problem decoding user document")
	}
	u.populated = true
	return nil
}

func (u *User) Save(ctx context.Context, env barque.Environment) error {
	res, err := env.DB().Collection(userCollection).ReplaceOne(ctx, u.idQuery(), u, options.Replace().SetUpsert(true))
	if err != nil {
		return errors.Wrapf(err, "problem saving user document %s", u.Username())
	}

	if res.ModifiedCount != 0 {
		return errors.Errorf("no user document saved or modified for %s", u.Username())
	}

	return nil
}

func (u *User) Email() string     { return u.EmailAddress }
func (u *User) Username() string  { return u.ID }
func (u *User) GetAPIKey() string { return u.APIKey }
func (u *User) Roles() []string   { return u.SystemRoles }

func (u *User) DisplayName() string {
	if u.Display != "" {
		return u.Display
	}
	return u.ID
}

func (u *User) SetAPIKey(ctx context.Context, env barque.Environment) (string, error) {
	k := utility.RandomString()

	res, err := env.DB().Collection(userCollection).UpdateOne(ctx, u.idQuery(), bson.M{
		"$set": bson.M{
			dbUserAPIKeyKey: k,
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}
	if res.ModifiedCount != 1 {
		return "", errors.New("could not find user in the database")
	}

	u.APIKey = k
	return k, nil
}

func (u *User) UpdateLoginCache(ctx context.Context, env barque.Environment) (string, error) {
	var update bson.M

	if u.LoginCache.Token == "" {
		u.LoginCache.Token = utility.RandomString()

		update = bson.M{"$set": bson.M{
			bsonutil.GetDottedKeyName(dbUserLoginCacheKey, loginCacheTokenKey): u.LoginCache.Token,
			bsonutil.GetDottedKeyName(dbUserLoginCacheKey, loginCacheTTLKey):   time.Now(),
		}}
	} else {
		update = bson.M{"$set": bson.M{
			bsonutil.GetDottedKeyName(dbUserLoginCacheKey, loginCacheTTLKey): time.Now(),
		}}
	}

	res, err := env.DB().Collection(userCollection).UpdateOne(ctx, u.idQuery(), update)
	if err != nil {
		return "", errors.Wrap(err, "problem with update operation for cached user")
	}

	if res.ModifiedCount != 1 {
		return "", errors.Wrapf(err, "problem updating cached user document '%s'", u.Username())
	}

	return u.LoginCache.Token, nil
}

// PutLoginCache generates, saves, and returns a new token; the user's TTL is
// updated.
func PutLoginCache(user gimlet.User) (string, error) {
	env := barque.GetEnvironment()
	ctx, cancel := env.Context()
	defer cancel()

	u, err := FindUser(ctx, env, user.Username())
	if db.ResultsNotFound(errors.Cause(err)) {
		return "", errors.Errorf("could not find user %s in the database", user.Username())
	} else if err != nil {
		return "", errors.Wrap(err, "problem finding user")
	}

	token, err := u.UpdateLoginCache(ctx, env)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return token, nil
}

// GetUserLoginCache retrieves cached users by token.
//
// It returns an error if and only if there was an error retrieving the user
// from the cache.
//
// It returns (<user>, true, nil) if the user is present in the cache and is
// valid.
//
// It returns (<user>, false, nil) if the user is present in the cache but has
// expired.
//
// It returns (nil, false, nil) if the user is not present in the cache.
func GetLoginCache(token string) (gimlet.User, bool, error) {
	env := barque.GetEnvironment()
	ctx, cancel := env.Context()
	defer cancel()

	user := &User{}
	query := bson.M{bsonutil.GetDottedKeyName(dbUserLoginCacheKey, loginCacheTokenKey): token}

	err := env.DB().Collection(userCollection).FindOne(ctx, query).Decode(user)
	if db.ResultsNotFound(err) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, errors.Wrap(err, "problem getting user from cache")
	}

	if time.Since(user.LoginCache.TTL) > barque.TokenExpireAfter {
		return user, false, nil
	}
	return user, true, nil
}

// ClearLoginCache removes users' tokens from cache. Passing true will ignore
// the user passed and clear all users.
func ClearLoginCache(user gimlet.User, all bool) error {
	env := barque.GetEnvironment()
	ctx, cancel := env.Context()
	defer cancel()

	update := bson.M{"$unset": bson.M{dbUserLoginCacheKey: 1}}
	if all {
		query := bson.M{}
		_, err := env.DB().Collection(userCollection).UpdateMany(ctx, query, update)
		if err != nil {
			return errors.Wrap(err, "problem clearing user cache")
		}
	} else {
		u := &User{ID: user.Username()}

		res, err := env.DB().Collection(userCollection).UpdateOne(ctx, u.idQuery(), update)
		if err != nil {
			return errors.Wrap(err, "problem updating user cache")
		}

		if res.ModifiedCount != 1 {
			return errors.Errorf("did clear cached user for '%s'", u.Username())
		}
	}

	return nil
}

// GetUser gets a user by id from persistent storage, and returns whether the
// returned user's token is valid or not.
func GetUser(id string) (gimlet.User, bool, error) {
	env := barque.GetEnvironment()
	ctx, cancel := env.Context()
	defer cancel()

	user, err := FindUser(ctx, env, id)
	if err != nil {
		return nil, false, errors.WithStack(err)

	}

	return user, time.Since(user.LoginCache.TTL) < barque.TokenExpireAfter, nil
}

// GetOrAddUser gets a user from persistent storage, or if the user does not
// exist, to create and save it.
func GetOrAddUser(user gimlet.User) (gimlet.User, error) {
	env := barque.GetEnvironment()
	ctx, cancel := env.Context()
	defer cancel()

	u, err := FindUser(ctx, env, user.Username())
	if db.ResultsNotFound(errors.Cause(err)) {
		u = &User{}
		u.ID = user.Username()
		u.Display = user.DisplayName()
		u.EmailAddress = user.Email()
		u.APIKey = user.GetAPIKey()
		u.SystemRoles = user.Roles()
		u.CreatedAt = time.Now()
		u.LoginCache = LoginCache{Token: utility.RandomString(), TTL: time.Now()}
		u.populated = true
		if err = u.Save(ctx, env); err != nil {
			return nil, errors.Wrapf(err, "problem inserting user %s", user.Username())
		}
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem finding user %s by id", user.Username())
	}

	return u, nil
}
