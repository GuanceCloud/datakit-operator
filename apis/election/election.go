package election

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/luci/go-render/render"
)

const (
	jsonContentType = "application/json"
)

var globalTaskManager = newTaskManager()

func HandleElection(w http.ResponseWriter, r *http.Request) {
	handle(w, r)
}

func handle(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	l.Debugf("handling request: %s", body)

	requ := &electionRequest{}
	if err := json.Unmarshal(body, requ); err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		l.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	result, err := processElection(requ)
	if err != nil {
		msg := fmt.Sprintf("Invalid election parameter: %v", err)
		l.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return

	}

	respBytes, err := json.Marshal(result)
	if err != nil {
		l.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", jsonContentType)
	if _, err := w.Write(respBytes); err != nil {
		l.Error(err)
	}
}

type electionRequest struct {
	Namespace         string         `json:"namespace,omitempty"`
	ID                string         `json:"id"`
	Timestamp         int64          `json:"timestamp,omitempty"`
	ApplicationInputs map[string]int `json:"application_inputs"`
	RunningInputs     []string       `json:"running_inputs,omitempty"`
}

type electionResult struct {
	AllowedInputs []string `json:"allowed_inputs"`
}

func processElection(requ *electionRequest) (*electionResult, error) {
	globalTaskManager.mu.Lock()
	defer globalTaskManager.mu.Unlock()

	if err := globalTaskManager.checkList(requ.ApplicationInputs); err != nil {
		return nil, err
	}

	allowed := globalTaskManager.takeTask(requ.ID)
	l.Debugf("ID %s is allowed to run %s", requ.ID, allowed)
	l.Debugf("workers info: %#v", render.Render(globalTaskManager.workers))

	return &electionResult{AllowedInputs: allowed}, nil
}
