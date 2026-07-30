package healthcheck

import "testing"

func TestFeatureID(t *testing.T) {
	if (&Feature{}).ID() != "healthcheck" {
		t.Fatal("unexpected feature ID")
	}
}
