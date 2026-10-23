package paths

import "testing"

func TestNormalizeDestinationPath(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		`W:\shows`:                 `W:\shows`,
		`W:\\shows`:                `W:\shows`,
		`w:/shows//season 1`:       `w:\shows\season 1`,
		`\\server\share\shows`:     `\\server\share\shows`,
		`\\\\server\\share\\shows`: `\\server\share\shows`,
		`/mnt/media/shows`:         `/mnt/media/shows`,
		`auto`:                     `auto`,
	}
	for input, want := range tests {
		if got := NormalizeDestinationPath(input); got != want {
			t.Errorf("NormalizeDestinationPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveAutoPathNormalizesConfiguredWindowsPath(t *testing.T) {
	t.Parallel()
	if got := ResolveAutoPath(`W:\\shows`, "shows"); got != `W:\shows` {
		t.Fatalf("resolved path = %q", got)
	}
}
