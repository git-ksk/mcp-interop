package suite

import "testing"

func TestV1CandidateSuiteSchemaIdentitiesRemainStable(t *testing.T) {
	checks := []struct {
		name        string
		gotVersion  int
		wantVersion int
		gotType     string
		wantType    string
	}{
		{"manifest", SchemaVersionV1, 1, "", ""},
		{"result-set", ResultSetSchemaVersion, 1, ResultSetArtifactType, "mcp-interop/suite-results"},
		{"baseline", BaselineSchemaVersion, 1, BaselineArtifactType, "mcp-interop/suite-baseline"},
		{"regression", RegressionReportSchemaVersion, 1, RegressionReportArtifactType, "mcp-interop/suite-regression-report"},
		{"compatibility", CompatibilityEnvelopeSchemaVersion, 1, CompatibilityEnvelopeArtifactType, "mcp-interop/compatibility-envelope"},
	}
	for _, check := range checks {
		if check.gotVersion != check.wantVersion || check.gotType != check.wantType {
			t.Fatalf("%s schema identity changed: version=%d type=%q", check.name, check.gotVersion, check.gotType)
		}
	}
}
