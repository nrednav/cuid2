package cuid2

import (
	"os"
	"strings"
	"testing"
)

func TestDocumentedInvariants(t *testing.T) {
	source, err := os.ReadFile("cuid2.go")
	if err != nil {
		t.Fatalf("could not read cuid2.go: %v", err)
	}

	sourceText := string(source)

	if !strings.Contains(sourceText, "SHA3-512 base36 is longer than MaxIdLength") {
		t.Error("cuid2.go must document the digest slice invariant")
	}

	if strings.Contains(sourceText, "2^53 - 1") {
		t.Error("cuid2.go still carries the stale 2^53 - 1 comment")
	}

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("could not read README.md: %v", err)
	}

	readmeText := string(readme)

	if !strings.Contains(readmeText, "getConstants") || !strings.Contains(readmeText, "DefaultIdLength") {
		t.Error("README must document the getConstants to Go constant mapping")
	}

	if !strings.Contains(readmeText, "approximately") {
		t.Error("README must document that ordering is approximate")
	}

	if strings.Contains(readmeText, "&sc.value") {
		t.Error("README counter example references sc.value, which does not compile")
	}
}
