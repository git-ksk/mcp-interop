package capability

import "testing"

func TestV1CandidateCapabilityStateAndEvidenceKindsRemainStable(t *testing.T) {
	if StatePass != "pass" || StateFail != "fail" || StateUnknown != "unknown" || StateUnsupported != "unsupported" || StateUntested != "untested" {
		t.Fatalf("capability states changed: %q %q %q %q %q", StatePass, StateFail, StateUnknown, StateUnsupported, StateUntested)
	}
	if EvidenceClientProtocol != "client_protocol" || EvidenceClientControlSurface != "client_control_surface" || EvidenceClientObservedState != "client_observed_state" || EvidenceAdapterPolicy != "adapter_policy" || EvidenceNone != "none" {
		t.Fatal("capability evidence-kind identity changed")
	}
	if err := validateObservation(Observation{CapabilityID: "resources", State: StatePass, EvidenceKind: EvidenceAdapterPolicy, EvidenceID: "policy"}); err == nil {
		t.Fatal("capability PASS accepted indirect adapter-policy evidence")
	}
	if err := validateObservation(Observation{CapabilityID: "resources", State: StateUntested, EvidenceKind: EvidenceNone}); err != nil {
		t.Fatalf("untested/no-evidence contract changed: %v", err)
	}
}
