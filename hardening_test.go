package cuid2

import (
	"crypto/rand"
	"errors"
	"io"
	"math"
	"sync/atomic"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

type countingReader struct {
	inner io.Reader
	reads int
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.reads++

	return c.inner.Read(p)
}

func TestWithRandomFuncRejectsOutOfRangeValues(t *testing.T) {
	invalid := map[string]func() float64{
		"NaN":      func() float64 { return math.NaN() },
		"negative": func() float64 { return -0.001 },
		"one":      func() float64 { return 1 },
		"above one": func() float64 {
			return 1.5
		},
	}

	for name, randomFunc := range invalid {
		if _, err := Init(WithRandomFunc(randomFunc)); err == nil {
			t.Errorf("expected Init to reject a random function returning %s", name)
		}
	}
}

func TestCustomRandomFuncOutOfContractPanics(t *testing.T) {
	calls := 0
	randomFunc := func() float64 {
		calls++

		if calls == 1 {
			return 0.5
		}

		return math.NaN()
	}

	generate, err := Init(WithRandomFunc(randomFunc), WithLength(8))
	if err != nil {
		t.Fatalf("expected Init to accept a valid first sample, got %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic when the custom random function returns NaN after its first sample")
		}
	}()

	generate()
}

func TestWithSessionCounterRejectsNil(t *testing.T) {
	if _, err := Init(WithSessionCounter(nil)); err == nil {
		t.Fatal("expected Init to reject a nil session counter")
	}
}

func TestInitReturnsNilGeneratorOnError(t *testing.T) {
	generate, err := Init(WithLength(MaxIdLength + 1))
	if err == nil {
		t.Fatal("expected Init to fail for an out-of-range length")
	}

	if generate != nil {
		t.Fatal("expected a nil generator when Init returns an error")
	}
}

func TestGenerateRetriesAfterInitFailure(t *testing.T) {
	initMu.Lock()
	defaultGenerator = atomic.Value{}
	initMu.Unlock()

	originalReader := rand.Reader
	rand.Reader = failingReader{}
	defer func() { rand.Reader = originalReader }()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected Generate to fail closed when entropy is unavailable")
			}
		}()

		Generate()
	}()

	rand.Reader = originalReader

	for i := 0; i < 3; i++ {
		if cuid := Generate(); len(cuid) != DefaultIdLength {
			t.Fatalf("expected the retried generator to produce a %d-character id, got %q", DefaultIdLength, cuid)
		}
	}
}

func TestPerIDEntropyFailurePanics(t *testing.T) {
	generate, err := Init()
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	originalReader := rand.Reader
	rand.Reader = failingReader{}
	defer func() { rand.Reader = originalReader }()

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic when the per-id entropy read fails")
		}
	}()

	generate()
}

func TestDefaultSourceReadsEntropyOncePerID(t *testing.T) {
	originalReader := rand.Reader
	counting := &countingReader{inner: originalReader}
	rand.Reader = counting
	defer func() { rand.Reader = originalReader }()

	for _, length := range []int{MinIdLength, DefaultIdLength, MaxIdLength} {
		generate, err := Init(WithLength(length))
		if err != nil {
			t.Fatalf("unexpected init error for length %d: %v", length, err)
		}

		before := counting.reads
		generate()

		if got := counting.reads - before; got != 1 {
			t.Errorf("length %d: expected 1 entropy read per id, got %d", length, got)
		}
	}
}

func TestEnvironmentKeyStringSkipsEntriesWithoutSeparator(t *testing.T) {
	env := []string{"KEY1=value", "MALFORMED", "KEY2=value"}

	if got := getEnvironmentKeyStringFrom(env); got != "KEY1KEY2" {
		t.Fatalf("expected KEY1KEY2, got %q", got)
	}
}

func TestFingerprintLength(t *testing.T) {
	fingerprint := createFingerprint(func() float64 { return 0.5 }, "KEY")

	if len(fingerprint) != MaxIdLength {
		t.Fatalf("expected a %d-character fingerprint, got %d", MaxIdLength, len(fingerprint))
	}
}

func TestAlphabetSizeMatchesAlphabet(t *testing.T) {
	if int64(AlphabetSize) != int64(len(alphabet)) {
		t.Fatalf("AlphabetSize %d does not match the alphabet length %d", AlphabetSize, len(alphabet))
	}
}
