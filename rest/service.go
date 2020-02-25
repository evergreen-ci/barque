package rest

import (
	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/barque/model"
	"github.com/evergreen-ci/gimlet"
	"github.com/evergreen-ci/gimlet/ldap"
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

	if s.Conf.LDAP.URL != "" {
		s.UserManager, err = ldap.NewUserService(ldap.CreationOpts{
			URL:           s.Conf.LDAP.URL,
			Port:          s.Conf.LDAP.Port,
			UserPath:      s.Conf.LDAP.UserPath,
			ServicePath:   s.Conf.LDAP.ServicePath,
			UserGroup:     s.Conf.LDAP.UserGroup,
			ServiceGroup:  s.Conf.LDAP.ServiceGroup,
			PutCache:      model.PutLoginCache,
			GetCache:      model.GetLoginCache,
			ClearCache:    model.ClearLoginCache,
			GetUser:       model.GetUser,
			GetCreateUser: model.GetOrAddUser,
		})
		if err != nil {
			return errors.Wrap(err, "problem setting up ldap user manager")
		}
	} else if s.Conf.NaiveAuth.AppAuth {
		users := []gimlet.User{}
		for _, user := range s.Conf.NaiveAuth.Users {
			users = append(
				users,
				gimlet.NewBasicUser(
					user.ID,
					user.Name,
					user.EmailAddress,
					user.Password,
					user.Key,
					user.AccessRoles,
					user.Invalid,
				),
			)
		}
		s.UserManager, err = gimlet.NewBasicUserManager(users)
		if err != nil {
			return errors.Wrap(err, "problem setting up basic user manager")
		}
	}

	return nil
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
	app.AddRoute("/repobuilder/check/{job_id}").Version(1).Post().Wrap(checkUser).Handler(s.checkRepobuilderJob)
}
