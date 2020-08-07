package rest

import (
	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/barque/model"
	"github.com/evergreen-ci/gimlet"
	"github.com/evergreen-ci/gimlet/cached"
	"github.com/evergreen-ci/gimlet/ldap"
	"github.com/evergreen-ci/gimlet/usercache"
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
			return "", errors.New("cannot put new users in DB")
		},
		GetUserByToken: func(string) (gimlet.User, bool, error) {
			return nil, false, errors.New("cannot get user by login token")
		},
		ClearUserToken: func(gimlet.User, bool) error {
			return errors.New("cannot clear user login token")
		},
		GetUserByID: func(id string) (gimlet.User, bool, error) {
			user, _, err := model.GetUser(id)
			if err != nil {
				return nil, false, errors.Errorf("finding user")
			}
			return user, true, nil
		},
		GetOrCreateUser: func(u gimlet.User) (gimlet.User, error) {
			user, _, err := model.GetUser(u.Username())
			if err != nil {
				return nil, errors.Wrap(err, "failed to find user and cannot create new one")
			}
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
			PutUserGetToken: model.PutLoginCache,
			GetUserByToken:  model.GetLoginCache,
			ClearUserToken:  model.ClearLoginCache,
			GetUserByID:     model.GetUser,
			GetOrCreateUser: model.GetOrAddUser,
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

	app.AddRoute("/admin/status").Version(1).Get().Handler(s.statusHandler)
	app.AddRoute("/repobuilder").Version(1).Post().Wrap(checkUser).Handler(s.addRepobuilderJob)
	app.AddRoute("/repobuilder/check/{job_id}").Version(1).Get().Wrap(checkUser).Handler(s.checkRepobuilderJob)
}
