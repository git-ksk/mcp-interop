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
	Stage      Stage      `json:"stage"`
	Status     Status     `json:"status"`
	ReasonCode ReasonCode `json:"reason_code,omitempty"`
	Message    string     `json:"message,omitempty"`
}

// AuthCapabilities is independent, secret-free authorization-server metadata
// evidence attached to a real-client auth failure. Nil booleans mean the
// capability could not be proven for one unambiguous authorization server.
type AuthCapabilities struct {
	CIMDAdvertised           *bool `json:"cimd_advertised,omitempty"`
	DCRAdvertised            *bool `json:"dcr_advertised,omitempty"`
	AuthorizationServerCount int   `json:"authorization_server_count,omitempty"`
	SelectionAmbiguous       bool  `json:"selection_ambiguous,omitempty"`
}

// Diagnostic adds supporting evidence without changing the four-stage real-
// client verdict. Diagnostics must remain secret-free and are redacted before
// they are emitted.
type Diagnostic struct {
	ID               string            `json:"id"`
	Stage            Stage             `json:"stage,omitempty"`
	ReasonCode       ReasonCode        `json:"reason_code,omitempty"`
	Conclusion       string            `json:"conclusion,omitempty"`
	Message          string            `json:"message,omitempty"`
	AuthCapabilities *AuthCapabilities `json:"auth_capabilities,omitempty"`
}

// Result is the deterministic report returned by one real-client adapter.
type Result struct {
	ClientID      string        `json:"client_id"`
	ClientName    string        `json:"client_name"`
	ClientVersion string        `json:"client_version,omitempty"`
	Endpoint      string        `json:"endpoint"`
	Stages        []StageResult `json:"stages"`
	Diagnostics   []Diagnostic  `json:"diagnostics,omitempty"`
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

// Set updates exactly one known stage while preserving report ordering. Any
// previously attached reason code is cleared because the caller is replacing
// the complete stage observation.
func (r *Result) Set(stage Stage, status Status, message string) bool {
	return r.SetWithReason(stage, status, "", message)
}

// SetWithReason updates one known stage and attaches a stable machine-readable
// explanation in addition to the human-readable message.
func (r *Result) SetWithReason(stage Stage, status Status, reasonCode ReasonCode, message string) bool {
	for i := range r.Stages {
		if r.Stages[i].Stage == stage {
			r.Stages[i].Status = status
			r.Stages[i].ReasonCode = reasonCode
			r.Stages[i].Message = message
			return true
		}
	}
	return false
}

// AddDiagnostic appends supporting evidence without mutating any stage result.
func (r *Result) AddDiagnostic(diagnostic Diagnostic) {
	r.Diagnostics = append(r.Diagnostics, diagnostic)
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

// Passed reports whether the complete interoperability contract was proven.
// Unknown and skipped stages are intentionally not treated as success so CI
// cannot silently pass an inconclusive client test.
func (r Result) Passed() bool {
	if len(r.Stages) != len(OrderedStages) {
		return false
	}
	for _, stage := range OrderedStages {
		item, ok := r.Get(stage)
		if !ok || item.Status != StatusPass {
			return false
		}
	}
	return true
}
