package rest

import (
	"net/http"

	"github.com/evergreen-ci/barque/model"
	"github.com/evergreen-ci/gimlet"
	"github.com/pkg/errors"
)

type userCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userAPIKeyResponse struct {
	Username string `json:"username"`
	Key      string `json:"key"`
}

func (s *Service) fetchUserToken(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	creds := &userCredentials{}
	if err := gimlet.GetJSON(r.Body, creds); err != nil {
		gimlet.WriteResponse(rw, gimlet.MakeJSONErrorResponder(errors.Wrap(err, "problem reading request body")))
		return
	}

	if creds.Username == "" {
		gimlet.WriteJSONResponse(rw, http.StatusUnauthorized, gimlet.ErrorResponse{
			Message:    "no username specified",
			StatusCode: http.StatusUnauthorized,
		})
		return
	}

	resp := &userAPIKeyResponse{Username: creds.Username}

	token, err := s.UserManager.CreateUserToken(creds.Username, creds.Password)
	if err != nil {
		gimlet.WriteResponse(rw, gimlet.MakeJSONErrorResponder(errors.Wrap(err, "problem creating user token")))
		return
	}

	user, err := s.UserManager.GetUserByToken(ctx, token)
	if err != nil {
		gimlet.WriteResponse(rw, gimlet.MakeJSONInternalErrorResponder(errors.Wrap(err, "problem finding user")))
		return
	}

	key := user.GetAPIKey()
	if key != "" {
		resp.Key = key
		gimlet.WriteJSON(rw, resp)
		return
	}

	dbuser, ok := user.(*model.User)
	if !ok {
		gimlet.WriteJSONResponse(rw, http.StatusInternalServerError, gimlet.ErrorResponse{
			Message:    "cannot generate key for user",
			StatusCode: http.StatusInternalServerError,
		})

		return
	}
	if err = dbuser.Save(ctx, s.Environment); err != nil {
		gimlet.WriteResponse(rw, gimlet.MakeJSONInternalErrorResponder(errors.Wrap(err, "problem generating key")))
		return
	}
	key, err = dbuser.CreateAPIKey(ctx, s.Environment)
	if err != nil {
		gimlet.WriteResponse(rw, gimlet.MakeJSONInternalErrorResponder(errors.Wrap(err, "problem generating key")))
		return
	}

	resp.Key = key
	gimlet.WriteJSON(rw, resp)
}
