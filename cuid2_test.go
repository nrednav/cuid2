package cuid2

import (
	"math/rand"
	"testing"
)

// External Tests
func TestIsCuid(t *testing.T) {
	testCases := map[string]bool{
		Generate():                           true,  // Default
		Generate() + Generate() + Generate(): false, // Too Long
		"":                                   false, // Too Short
		"42":                                 false, // Non-CUID
		"aaaaDLL":                            false, // Capital letters
		"yi7rqj1trke":                        true,  // Valid
		"-x!ha":                              false, // Invalid characters
		"ab*%@#x":                            false, // Invalid characters
	}

	for testCase, expected := range testCases {
		if IsCuid(testCase) != expected {
			t.Fatalf("Expected IsCuid(%v) to be %v, but got %v", testCase, expected, !expected)
		}
	}
}

func TestGeneratingInvalidCuid(t *testing.T) {
	_, err := Init(WithLength(64))
	if err == nil {
		t.Fatalf(
			"Expected to receive an error for Init(WithLength(64)), but got nothing",
		)
	}
}

func TestDefaultCuidLength(t *testing.T) {
	cuid := Generate()
	if len(cuid) != DefaultIdLength {
		t.Fatalf("Expected default Cuid length to be %v, but got %v", DefaultIdLength, len(cuid))
	}
}

func TestGeneratingCuidWithCustomLength(t *testing.T) {
	customLength := 16
	generate, err := Init(WithLength(customLength))
	if err != nil {
		t.Fatalf("Expected to initialize cuid2 generator but received error = %v", err.Error())
	}

	cuid := generate()

	if len(cuid) != customLength {
		t.Fatalf("Expected to generate Cuid with a custom length of %v, but got %v", customLength, len(cuid))
	}
}

func TestGeneratingCuidWithMaxLength(t *testing.T) {
	generate, err := Init(WithLength(MaxIdLength))
	if err != nil {
		t.Fatalf("Expected to initialize cuid2 generator but received error = %v", err.Error())
	}

	cuid := generate()

	if len(cuid) != MaxIdLength {
		t.Fatalf("Expected to generate Cuid with a max length of %v, but got %v", MaxIdLength, cuid)
	}
}

// Internal Tests
func TestSessionCounter(t *testing.T) {
	var initialSessionCount int64 = 10
	sessionCounter := NewSessionCounter(initialSessionCount)
	expectedCounts := []int64{11, 12, 13, 14}
	actualCounts := []int64{
		sessionCounter.Increment(),
		sessionCounter.Increment(),
		sessionCounter.Increment(),
		sessionCounter.Increment(),
	}

	for index, actualCount := range actualCounts {
		expectedCount := expectedCounts[index]
		if actualCount != expectedCount {
			t.Error("Expected session counts to increment by one for each successive call")
			t.Errorf("For an initial session count of %v, expected %v", initialSessionCount, expectedCounts)
			t.Fatalf("Got %v", actualCounts)
		}
	}
}

func TestUnsafeSessionCounter(t *testing.T) {
	var initialSessionCount int64 = 10
	sessionCounter := NewUnsafeSessionCounter(initialSessionCount)
	expectedCounts := []int64{11, 12, 13, 14}
	actualCounts := []int64{
		sessionCounter.Increment(),
		sessionCounter.Increment(),
		sessionCounter.Increment(),
		sessionCounter.Increment(),
	}

	for index, actualCount := range actualCounts {
		expectedCount := expectedCounts[index]
		if actualCount != expectedCount {
			t.Error("Expected session counts to increment by one for each successive call")
			t.Errorf("For an initial session count of %v, expected %v", initialSessionCount, expectedCounts)
			t.Fatalf("Got %v", actualCounts)
		}
	}
}

func TestCreatingFingerprintWithEnvKeyString(t *testing.T) {
	fingerprint := createFingerprint(rand.Float64, getEnvironmentKeyString())
	if len(fingerprint) < MinIdLength {
		t.Error("Could not generate fingerprint of adequate length")
		t.Fatalf("Expected length to be at least %v, but got %v", MinIdLength, len(fingerprint))
	}
}

func TestCreatingFingerprintWithoutEnvKeyString(t *testing.T) {
	fingerprint := createFingerprint(rand.Float64, "")
	if len(fingerprint) < MinIdLength {
		t.Error("Could not generate fingerprint of adequate length")
		t.Fatalf("Expected length to be at least %v, but got %v", MinIdLength, len(fingerprint))
	}
}

func TestGeneratingDefaultFromChan(t *testing.T) {
	ch, _, _ := InitChan()

	for i := 0; i < 100; i++ {
		cuid := <-ch

		if len(cuid) != DefaultIdLength {
			t.Fatalf("Expected default Cuid length to be %v, but got %v", DefaultIdLength, len(cuid))
		}
	}
}

func TestGeneratingCustomLengthFromChan(t *testing.T) {
	customLength := 16
	ch, _, err := InitChan(WithLength(customLength))

	if err != nil {
		t.Fatalf("Expected to initialize cuid2 chan generator but received error = %v", err.Error())
	}

	for i := 0; i < 100; i++ {
		cuid := <-ch

		if len(cuid) != customLength {
			t.Fatalf("Expected to generate Cuid with a custom length of %v, but got %v", customLength, len(cuid))
		}
	}
}
