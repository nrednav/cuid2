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

func TestDeterminismOfGeneration(t *testing.T) {
	testCases := []struct {
		name string
		length int
		fingerprint string
		counterStart int64
		timeMs int64
		randomFloat float64
		expectedID string
		expectedNextID string
	}{
		{
			name: "Short ID with low random value",
			length: 10,
			fingerprint: "test-fingerprint",
			counterStart: 0,
			timeMs: 1751850060928,
			randomFloat: 0.1,
			expectedID: "c79ab4qwd8",
			expectedNextID: "ctfxvev2em",
		},
		{
			name:          "Long ID with high random value",
			length:        32,
			fingerprint:   "fruit-salad",
			counterStart:  476782360,
			timeMs:        1751850806018,
			randomFloat:   0.8,
			expectedID:    "uhqvhs8l0q5ub01c37pgwfqak5az4l2n",
			expectedNextID: "u2c7rvhbd6evwn1vj69bye7tj8e7ou2m",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			g := &cuidGenerator {
				length: testCase.length,
				counter: NewSessionCounter(testCase.counterStart),
				fingerprint: testCase.fingerprint,
			}

			mockRandomFunc := func() float64 {
				return testCase.randomFloat
			}

			firstID := g.generate(testCase.timeMs, mockRandomFunc)

			if firstID != testCase.expectedID {
				t.Errorf("First ID generated did not match expected.\nGot: %s, Expected: %s", firstID, testCase.expectedID)
			}

			secondID := g.generate(testCase.timeMs, mockRandomFunc)

			if secondID != testCase.expectedNextID {
				t.Errorf("Second ID generated did not match expected.\nGot: %s, Expected: %s", secondID, testCase.expectedNextID)
			}
		})
	}
}
