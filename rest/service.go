package rest

import (
	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/barque/model"
	"github.com/evergreen-ci/gimlet"
	"github.com/evergreen-ci/gimlet/cached"
	"github.com/evergreen-ci/gimlet/ldap"
	"github.com/evergreen-ci/gimlet/usercache"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

type Service struct {
	Environment barque.Environment
	UserManager gimlet.UserManager
	Conf        *model.Configuration
	umconf      gimlet.UserMiddlewareConfiguration
}

func New(env barque.Environment) (*gimlet.APIApp, error) {
	s := &Service{Environment: env}
	if err := s.setup(); err != nil {
		return nil, errors.WithStack(err)
	}
	app := gimlet.NewApp()

	app.SetPrefix("rest")

	s.addMiddleware(app)
	s.addRoutes(app)

	return app, nil
}

func (s *Service) setup() error {
	ctx, cancel := s.Environment.Context()
	defer cancel()
	conf, err := model.FindConfiguration(ctx, s.Environment)
	if err != nil {
		return errors.WithStack(err)
	}
	s.Conf = conf

	s.umconf = gimlet.UserMiddlewareConfiguration{
		HeaderKeyName:  barque.APIKeyHeader,
		HeaderUserName: barque.APIUserHeader,
		CookieName:     barque.AuthTokenCookie,
		CookiePath:     "/",
		CookieTTL:      barque.TokenExpireAfter,
	}
	if err = s.umconf.Validate(); err != nil {
		return errors.New("programmer error; invalid user manager configuration")
	}

	if err := s.setupUserAuth(); err != nil {
		return errors.Wrap(err, "setting up auth")
	}

	return nil
}

func (s *Service) setupUserAuth() error {
	var readOnly []gimlet.UserManager
	var readWrite []gimlet.UserManager
	if s.Conf.ServiceAuth.Enabled {
		usrMngr, err := s.setupServiceAuth()
		if err != nil {
			return errors.Wrap(err, "setting up service user auth")
		}
		readOnly = append(readOnly, usrMngr)
	}
	if s.Conf.LDAP.URL != "" {
		usrMngr, err := s.setupLDAPAuth()
		if err != nil {
			return errors.Wrap(err, "setting up LDAP user auth")
		}
		readWrite = append(readWrite, usrMngr)
	}
	if s.Conf.NaiveAuth.AppAuth {
		usrMngr, err := s.setupNaiveAuth()
		if err != nil {
			return errors.Wrap(err, "setting up naive user auth")
		}
		readOnly = append(readOnly, usrMngr)
	}

	if len(readOnly)+len(readWrite) == 0 {
		return errors.New("no user authentication method could be set up")
	}

	s.UserManager = gimlet.NewMultiUserManager(readWrite, readOnly)

	return nil
}

func (s *Service) setupServiceAuth() (gimlet.UserManager, error) {
	opts := usercache.ExternalOptions{
		PutUserGetToken: func(gimlet.User) (string, error) {
			grip.Debug(message.Fields{
				"op":      "PutUserGetToken",
				"context": "service user manager",
			})
			return "", errors.New("cannot put new users in DB")
		},
		GetUserByToken: func(string) (gimlet.User, bool, error) {
			grip.Debug(message.Fields{
				"op":      "GetUserByToken",
				"context": "service user manager",
			})
			return nil, false, errors.New("cannot get user by login token")
		},
		ClearUserToken: func(gimlet.User, bool) error {
			grip.Debug(message.Fields{
				"op":      "ClearUserToken",
				"context": "service user manager",
			})
			return errors.New("cannot clear user login token")
		},
		GetUserByID: func(id string) (gimlet.User, bool, error) {
			msg := message.Fields{
				"username": id,
				"op":       "GetUserByID",
				"context":  "service user manager",
			}
			var user gimlet.User
			user, _, err := model.GetUser(id)
			if err != nil {
				msg["message"] = "failed to find user by ID"
				grip.Debug(message.WrapError(err, msg))
				return nil, false, errors.Errorf("finding user")
			}
			msg["message"] = "successfully found user by ID"
			grip.Debug(msg)
			return user, true, nil
		},
		GetOrCreateUser: func(u gimlet.User) (gimlet.User, error) {
			msg := message.Fields{
				"username": u.Username(),
				"op":       "GetOrCreateUser",
				"context":  "service user manager",
			}
			var user gimlet.User
			user, _, err := model.GetUser(u.Username())
			if err != nil {
				msg["message"] = "failed to find existing user"
				grip.Debug(message.WrapError(err, msg))
				return nil, errors.Wrap(err, "failed to find user and cannot create new one")
			}
			msg["message"] = "successfully found existing user"
			grip.Debug(msg)
			return user, nil
		},
	}

	cache, err := usercache.NewExternal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "setting up user cache backed by DB")
	}
	usrMngr, err := cached.NewUserManager(cache)
	if err != nil {
		return nil, errors.Wrap(err, "creating user manager backed by DB")
	}

	return usrMngr, nil
}

func (s *Service) setupLDAPAuth() (gimlet.UserManager, error) {
	usrMngr, err := ldap.NewUserService(ldap.CreationOpts{
		URL:          s.Conf.LDAP.URL,
		Port:         s.Conf.LDAP.Port,
		UserPath:     s.Conf.LDAP.UserPath,
		ServicePath:  s.Conf.LDAP.ServicePath,
		UserGroup:    s.Conf.LDAP.UserGroup,
		ServiceGroup: s.Conf.LDAP.ServiceGroup,
		ExternalCache: &usercache.ExternalOptions{
			PutUserGetToken: func(u gimlet.User) (string, error) {
				msg := message.Fields{
					"username": u.Username(),
					"op":       "PutLoginCache",
					"context":  "LDAP user manager",
				}
				token, err := model.PutLoginCache(u)
				if err != nil {
					msg["message"] = "failed to update login cache for user"
					grip.Debug(message.WrapError(err, msg))
					return "", errors.WithStack(err)
				}
				msg["message"] = "successfully updated login cache for user"
				msg["token"] = token
				grip.Debug(msg)
				return token, nil
			},
			GetUserByToken: func(token string) (gimlet.User, bool, error) {
				msg := message.Fields{
					"token":   token,
					"op":      "GetUserByToken",
					"context": "LDAP user manager",
				}
				u, valid, err := model.GetLoginCache(token)
				if err != nil {
					msg["message"] = "failed to get user by token"
					grip.Debug(message.WrapError(err, msg))
					return nil, false, errors.WithStack(err)
				}
				msg["message"] = "successfully found user by token"
				msg["username"] = u.Username()
				msg["valid"] = valid
				grip.Debug(msg)
				return u, valid, nil
			},
			ClearUserToken: func(u gimlet.User, all bool) error {
				msg := message.Fields{
					"all":     all,
					"op":      "ClearUserToken",
					"context": "LDAP user manager",
				}
				if err := model.ClearLoginCache(u, all); err != nil {
					msg["message"] = "failed to clear user token"
					grip.Debug(message.WrapError(err, msg))
					return errors.WithStack(err)
				}
				msg["message"] = "successfully cleared user token"
				grip.Debug(msg)
				return nil
			},
			GetUserByID: func(id string) (gimlet.User, bool, error) {
				msg := message.Fields{
					"username": id,
					"op":       "GetUserByID",
					"context":  "LDAP user manager",
				}
				u, valid, err := model.GetUser(id)
				if err != nil {
					msg["message"] = "failed to find user by ID"
					grip.Debug(message.WrapError(err, msg))
					return u, valid, errors.WithStack(err)
				}
				msg["message"] = "successfully found user by ID"
				msg["valid"] = valid
				grip.Debug(msg)
				return u, valid, nil
			},
			GetOrCreateUser: func(u gimlet.User) (gimlet.User, error) {
				msg := message.Fields{
					"username": u.Username(),
					"op":       "GetOrCreateUser",
					"context":  "LDAP user manager",
				}
				user, err := model.GetOrAddUser(u)
				if err != nil {
					msg["message"] = "failed to get existing or create new user"
					grip.Debug(message.WrapError(err, msg))
					return nil, errors.WithStack(err)
				}
				msg["message"] = "successfully found existing user or created new user"
				grip.Debug(msg)
				return user, nil
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "problem setting up ldap user manager")
	}
	return usrMngr, nil
}

func (s *Service) setupNaiveAuth() (gimlet.UserManager, error) {
	users := []gimlet.BasicUser{}
	for _, user := range s.Conf.NaiveAuth.Users {
		users = append(
			users,
			gimlet.BasicUser{
				ID:           user.ID,
				Name:         user.Name,
				EmailAddress: user.EmailAddress,
				Password:     user.Password,
				Key:          user.Key,
				AccessRoles:  user.AccessRoles,
			},
		)
	}
	usrMngr, err := gimlet.NewBasicUserManager(users, nil)
	if err != nil {
		return nil, errors.Wrap(err, "problem setting up basic user manager")
	}
	return usrMngr, nil
}

func (s *Service) addMiddleware(app *gimlet.APIApp) {
	app.AddMiddleware(gimlet.MakeRecoveryLogger())
	app.AddMiddleware(gimlet.UserMiddleware(s.UserManager, s.umconf))
	app.AddMiddleware(gimlet.NewAuthenticationHandler(gimlet.NewBasicAuthenticator(nil, nil), s.UserManager))
}

func (s *Service) addRoutes(app *gimlet.APIApp) {
	checkUser := gimlet.NewRequireAuthHandler()

	app.AddRoute("/admin/login").Version(1).Post().Handler(s.fetchUserToken)
	app.AddRoute("/admin/status").Version(1).Get().Handler(s.statusHandler)
	app.AddRoute("/repobuilder").Version(1).Post().Wrap(checkUser).Handler(s.addRepobuilderJob)
	app.AddRoute("/repobuilder/check/{job_id}").Version(1).Get().Wrap(checkUser).Handler(s.checkRepobuilderJob)
}
