package interop

import (
	"fmt"
	"sort"
	"unicode"
	"unicode/utf8"
)

const MaxExpectedToolNames = 64

// ValidateExpectedToolName accepts bounded, printable identifiers with Unicode
// letters/marks/numbers and three ASCII separators. It excludes whitespace,
// bidi/control/format characters and arbitrary URL/token syntax. Never put
// credentials in operator-supplied names; flags may be visible to the OS.
func ValidateExpectedToolName(name string) error {
	const message = "expected tool name must be 1-128 bytes, start with a letter or underscore, and contain only Unicode letters/marks/numbers or '.', '_' and '-'"
	if len(name) == 0 || len(name) > 128 || !utf8.ValidString(name) {
		return fmt.Errorf("%s", message)
	}
	for index, letter := range name {
		if index == 0 {
			if !unicode.IsLetter(letter) && letter != '_' {
				return fmt.Errorf("%s", message)
			}
			continue
		}
		if !unicode.IsLetter(letter) && !unicode.IsNumber(letter) && !unicode.IsMark(letter) && letter != '.' && letter != '_' && letter != '-' {
			return fmt.Errorf("%s", message)
		}
	}
	return nil
}

// ToolExpectation is additive command output. It does NOT change the four core
// interop stage statuses or belong to the strict live-result v1/v2 artifact.
// Only operator-supplied expected names are echoed, never observed extra names.
type ToolExpectation struct {
	Status        Status   `json:"status"`
	ReasonCode    string   `json:"reason_code"`
	ExpectedNames []string `json:"expected_names,omitempty"`
	MissingNames  []string `json:"missing_names,omitempty"`
	ExpectedCount *int     `json:"expected_count,omitempty"`
	ObservedCount *int     `json:"observed_count,omitempty"`
}

const (
	ToolReasonMatched       = "expected_tools_matched"
	ToolReasonMissing       = "expected_tools_missing"
	ToolReasonCountMismatch = "expected_tool_count_mismatch"
	ToolReasonUnobservable  = "tool_inventory_unobservable"
)

// CheckToolExpectation evaluates a supported direct-client observation. It
// cannot infer name matches from generic stage PASS or fixture-only data.
func CheckToolExpectation(result Result, names []string, expectedCount *int) ToolExpectation {
	report := ToolExpectation{
		Status:        StatusUnknown,
		ReasonCode:    ToolReasonUnobservable,
		ExpectedNames: append([]string(nil), names...),
	}
	sort.Strings(report.ExpectedNames)
	if expectedCount != nil {
		n := *expectedCount
		report.ExpectedCount = &n
	}
	actual, known := result.ObservedToolNames()
	if !result.Passed() || !known {
		return report
	}
	count := len(actual)
	report.ObservedCount = &count
	observed := make(map[string]struct{}, len(actual))
	for _, tool := range actual {
		observed[tool] = struct{}{}
	}
	for _, expected := range report.ExpectedNames {
		if _, ok := observed[expected]; !ok {
			report.MissingNames = append(report.MissingNames, expected)
		}
	}
	if len(report.MissingNames) > 0 {
		report.Status = StatusFail
		report.ReasonCode = ToolReasonMissing
		return report
	}
	if expectedCount != nil && count != *expectedCount {
		report.Status = StatusFail
		report.ReasonCode = ToolReasonCountMismatch
		return report
	}
	report.Status = StatusPass
	report.ReasonCode = ToolReasonMatched
	return report
}
