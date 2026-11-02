package updater

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	if compareVersions("v2.1.0", "v2.0.9") <= 0 {
		t.Fatal("newer version was not detected")
	}
	if compareVersions("v2.0.0", "v2.0.0") != 0 {
		t.Fatal("equal versions differ")
	}
	if compareVersions("v1.9.9", "v2.0.0") >= 0 {
		t.Fatal("older version was not detected")
	}
}

func TestChecksumFor(t *testing.T) {
	t.Parallel()
	sum := sha256.Sum256([]byte("binary"))
	want := fmt.Sprintf("%x", sum)
	got, err := checksumFor([]byte(want+"  plexlinker-linux-amd64\n"), "plexlinker-linux-amd64")
	if err != nil || got != want {
		t.Fatalf("checksum=%q err=%v", got, err)
	}
}
