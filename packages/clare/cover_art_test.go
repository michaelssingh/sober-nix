package main

import (
	"testing"
)

func TestCoverArtCmd(t *testing.T) {
	cmd := doFetchCoverArt("", "", 16, 11)
	if cmd != nil {
		t.Fatalf("expected nil cmd for empty inputs, got non-nil")
	}
}
