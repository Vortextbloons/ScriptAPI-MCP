package version

import "testing"

func TestDisplayName(t *testing.T) {
	got := DisplayName()
	want := Name + " v" + Current
	if got != want {
		t.Errorf("DisplayName() = %q, want %q", got, want)
	}
}
