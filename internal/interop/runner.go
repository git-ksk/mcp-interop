package interop

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Target identifies the Remote MCP deployment under test.
type Target struct {
	Endpoint string `json:"endpoint"`
}

func (t Target) Validate() error {
	u, err := url.Parse(t.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("endpoint must use http or https")
	}
	if u.Host == "" {
		return errors.New("endpoint must include a host")
	}
	if u.User != nil {
		return errors.New("endpoint must not embed user info")
	}
	if u.Fragment != "" {
		return errors.New("endpoint must not include a URL fragment")
	}
	return nil
}

// Adapter executes client-specific black-box observations inside an isolated
// session owned by the runner.
type Adapter interface {
	Run(ctx context.Context, target Target, session *Session) (Result, error)
}

type sessionFactory func() (*Session, error)

// Runner owns temporary-session lifecycle, independent auth-failure diagnostic
// enrichment, and final secret redaction.
type Runner struct {
	newSession sessionFactory
}

func NewRunner() *Runner {
	return &Runner{newSession: NewSession}
}

func newRunner(factory sessionFactory) *Runner {
	return &Runner{newSession: factory}
}

func (r *Runner) Run(ctx context.Context, adapter Adapter, target Target) (Result, error) {
	if adapter == nil {
		return Result{}, errors.New("adapter is required")
	}
	if err := target.Validate(); err != nil {
		return Result{}, err
	}

	session, err := r.newSession()
	if err != nil {
		return Result{}, fmt.Errorf("create isolated session: %w", err)
	}

	result, runErr := adapter.Run(ctx, target, session)
	cleanupErr := session.Cleanup()
	if result.Endpoint == "" {
		result.Endpoint = target.Endpoint
	}
	EnrichAuthFailure(ctx, target.Endpoint, &result, nil)
	result = RedactResult(result)

	if runErr != nil {
		runErr = errors.New(Redact(runErr.Error()))
	}
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("cleanup isolated session: %w", cleanupErr)
	}

	return result, errors.Join(runErr, cleanupErr)
}
