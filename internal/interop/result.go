package interop

// Stage identifies an observable part of Remote MCP interoperability.
type Stage string

const (
	StageReach Stage = "reach"
	StageAuth  Stage = "auth"
	StageInit  Stage = "init"
	StageTools Stage = "tools"
)

// OrderedStages is the stable display/serialization order used by reports.
var OrderedStages = []Stage{StageReach, StageAuth, StageInit, StageTools}

// Status is the outcome of one interoperability stage.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkip    Status = "skip"
	StatusUnknown Status = "unknown"
)

// StageResult records one observed interoperability stage.
type StageResult struct {
	Stage   Stage  `json:"stage"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// Result is the deterministic report returned by one real-client adapter.
type Result struct {
	ClientID      string        `json:"client_id"`
	ClientName    string        `json:"client_name"`
	ClientVersion string        `json:"client_version,omitempty"`
	Endpoint      string        `json:"endpoint"`
	Stages        []StageResult `json:"stages"`
}

// NewResult creates a report with every stage explicitly unknown. Adapters must
// promote each stage only when they have an observable result.
func NewResult(clientID, clientName, clientVersion, endpoint string) Result {
	result := Result{
		ClientID:      clientID,
		ClientName:    clientName,
		ClientVersion: clientVersion,
		Endpoint:      endpoint,
		Stages:        make([]StageResult, 0, len(OrderedStages)),
	}
	for _, stage := range OrderedStages {
		result.Stages = append(result.Stages, StageResult{Stage: stage, Status: StatusUnknown})
	}
	return result
}

// Set updates exactly one known stage while preserving report ordering.
func (r *Result) Set(stage Stage, status Status, message string) bool {
	for i := range r.Stages {
		if r.Stages[i].Stage == stage {
			r.Stages[i].Status = status
			r.Stages[i].Message = message
			return true
		}
	}
	return false
}

// Get returns one stage result.
func (r Result) Get(stage Stage) (StageResult, bool) {
	for _, item := range r.Stages {
		if item.Stage == stage {
			return item, true
		}
	}
	return StageResult{}, false
}

// Failed reports whether any attempted stage explicitly failed.
func (r Result) Failed() bool {
	for _, item := range r.Stages {
		if item.Status == StatusFail {
			return true
		}
	}
	return false
}
