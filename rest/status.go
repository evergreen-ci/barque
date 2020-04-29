package rest

import (
	"net/http"

	"github.com/evergreen-ci/barque"
	"github.com/evergreen-ci/gimlet"
	"github.com/mongodb/amboy"
)

////////////////////////////////////////////////////////////////////////
//
// GET /status

type StatusResponse struct {
	Revision     string           `json:"revision"`
	QueueStats   amboy.QueueStats `json:"queue,omitempty"`
	QueueRunning bool             `json:"running"`
}

// statusHandler processes the GET request for
func (s *Service) statusHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.Environment.Context()
	defer cancel()

	resp := &StatusResponse{
		Revision: barque.BuildRevision,
	}

	if queue := s.Environment.RemoteQueue(); queue != nil {
		resp.QueueRunning = queue.Info().Started
		resp.QueueStats = queue.Stats(ctx)
	}

	gimlet.WriteJSON(w, resp)
}
