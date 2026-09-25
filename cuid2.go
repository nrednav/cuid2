package cuid2

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

const (
	DefaultIdLength int = 24
	MinIdLength     int = 2
	MaxIdLength     int = 32

	// ~22k hosts before 50% chance of initial counter collision
	MaxSessionCount int64 = 476782367

	Base36 = 36

	alphabet     = "abcdefghijklmnopqrstuvwxyz"
	AlphabetSize = len(alphabet)

	// entropyBatchValues is the worst case: one first letter plus a full-length salt.
	entropyBatchValues = MaxIdLength + 1
)

type Config struct {
	// A custom function that can generate a floating-point value between 0 and 1
	RandomFunc func() float64

	// A counter that will be used to affect the entropy of successive id
	// generation calls
	SessionCounter Counter

	// Length of the generated Cuid, min = 2, max = 32
	Length int

	// A unique string that will be used by the Cuid generator to help prevent
	// collisions when generating Cuids in a distributed system.
	Fingerprint string
}

type Counter interface {
	Increment() int64
}

type SessionCounter struct {
	value int64
}

func NewSessionCounter(initialCount int64) *SessionCounter {
	return &SessionCounter{value: initialCount}
}

func (sc *SessionCounter) Increment() int64 {
	return atomic.AddInt64(&sc.value, 1)
}

type cuidGenerator struct {
	length      int
	counter     Counter
	fingerprint string
}

type Option func(*Config) error

// Initializes the Cuid generator with default or user-defined config options
//
// Returns a function that can be called to generate Cuids using the initialized config
func Init(options ...Option) (func() string, error) {
	entropy, err := newBatchedSource()

	if err != nil {
		return nil, err
	}

	initialSessionCount := int64(
		math.Floor(entropy.Float64() * float64(MaxSessionCount)),
	)

	config := &Config{
		SessionCounter: NewSessionCounter(initialSessionCount),
		Length:         DefaultIdLength,
		Fingerprint:    createFingerprint(entropy.Float64, getEnvironmentKeyString()),
	}

	for _, option := range options {
		if option != nil {
			if applyErr := option(config); applyErr != nil {
				return nil, applyErr
			}
		}
	}

	g := &cuidGenerator{
		length:      config.Length,
		counter:     config.SessionCounter,
		fingerprint: config.Fingerprint,
	}

	return func() string {
		randomFunc := config.RandomFunc

		if randomFunc == nil {
			source, err := newBatchedSource()

			if err != nil {
				panic(err)
			}

			randomFunc = source.Float64
		}

		return g.generate(time.Now().UnixMilli(), randomFunc)
	}, nil
}

func (g *cuidGenerator) generate(timeMs int64, randomFunc func() float64) string {
	firstLetter := getRandomAlphabet(randomFunc)
	timeStr := strconv.FormatInt(timeMs, Base36)
	countStr := strconv.FormatInt(g.counter.Increment(), Base36)
	salt := createEntropy(g.length, randomFunc)
	hashInput := timeStr + salt + countStr + g.fingerprint

	return firstLetter + hash(hashInput)[1:g.length]
}

var (
	defaultGenerator atomic.Value // func() string
	initMu           sync.Mutex
)

// Generate returns a CUID using the default configuration.
// If initialization fails, it fails closed and a later call retries it.
func Generate() string {
	if g, ok := defaultGenerator.Load().(func() string); ok {
		return g()
	}

	initMu.Lock()
	defer initMu.Unlock()

	if defaultGenerator.Load() == nil {
		g, err := Init()

		if err != nil {
			panic(err)
		}

		defaultGenerator.Store(g)
	}

	return defaultGenerator.Load().(func() string)()
}

// Checks whether a given Cuid has a valid form and length
var cuidRegex = regexp.MustCompile("^[a-z][0-9a-z]+$")

func IsCuid(cuid string) bool {
	length := len(cuid)

	return cuidRegex.MatchString(cuid) && length >= MinIdLength && length <= MaxIdLength
}

// A custom function that will generate a random floating-point value between 0 and 1
func WithRandomFunc(randomFunc func() float64) Option {
	return func(config *Config) error {
		if r := randomFunc(); math.IsNaN(r) || r < 0 || r >= 1 {
			return fmt.Errorf("Error: the provided random function does not generate a value between 0 (inclusive) and 1 (exclusive)")
		}

		config.RandomFunc = func() float64 {
			v := randomFunc()

			if math.IsNaN(v) || v < 0 || v >= 1 {
				panic("Error: the provided random function returned a value outside the range [0, 1)")
			}

			return v
		}

		return nil
	}
}

// A custom counter that will be used to affect the entropy of successive id
// generation calls
func WithSessionCounter(sessionCounter Counter) Option {
	return func(config *Config) error {
		if sessionCounter == nil {
			return fmt.Errorf("Error: the session counter cannot be nil")
		}

		config.SessionCounter = sessionCounter

		return nil
	}
}

// Configures the length of the generated Cuid
//
// Min Length = 2, Max Length = 32
func WithLength(length int) Option {
	return func(config *Config) error {
		if length < MinIdLength || length > MaxIdLength {
			return fmt.Errorf("Error: Can only generate Cuid's with a length between %v and %v", MinIdLength, MaxIdLength)
		}

		config.Length = length

		return nil
	}
}

// A unique string that will be used by the id generator to help prevent
// collisions when generating Cuids in a distributed system.
func WithFingerprint(fingerprint string) Option {
	return func(config *Config) error {
		config.Fingerprint = fingerprint
		return nil
	}
}

// batchedSource yields uniform float64 values in [0, 1) from a single read of
// the OS entropy source.
type batchedSource struct {
	buf [entropyBatchValues * 8]byte
	off int
}

func newBatchedSource() (*batchedSource, error) {
	s := &batchedSource{}

	if _, err := rand.Read(s.buf[:]); err != nil {
		return nil, fmt.Errorf("Error: Failed to read from crypto/rand: %w", err)
	}

	return s, nil
}

// Float64 returns the next value. The top 53 bits of each 64-bit window are
// used, matching the precision of the previous 2^53 division.
func (s *batchedSource) Float64() float64 {
	v := binary.BigEndian.Uint64(s.buf[s.off : s.off+8])
	s.off += 8

	return float64(v>>11) / float64(1<<53)
}

func createFingerprint(randomFunc func() float64, envKeyString string) string {
	sourceString := createEntropy(MaxIdLength, randomFunc)

	if len(envKeyString) > 0 {
		sourceString += envKeyString
	}

	return hash(sourceString)[:MaxIdLength]
}

func createEntropy(length int, randomFunc func() float64) string {
	var builder strings.Builder

	builder.Grow(length)

	for builder.Len() < length {
		builder.WriteString(strconv.FormatInt(getRandomInt(randomFunc, Base36), Base36))
	}

	return builder.String()[:length]
}

func getEnvironmentKeyString() string {
	return getEnvironmentKeyStringFrom(os.Environ())
}

func getEnvironmentKeyStringFrom(env []string) string {
	keys := make([]string, 0, len(env))

	// Discard values of environment variables
	for _, variable := range env {
		if idx := strings.IndexByte(variable, '='); idx >= 0 {
			keys = append(keys, variable[:idx])
		}
	}

	sort.Strings(keys)

	return strings.Join(keys, "")
}

func hash(input string) string {
	hash := sha3.New512()
	hash.Write([]byte(input))
	hashDigest := hash.Sum(nil)

	return new(big.Int).SetBytes(hashDigest).Text(Base36)[1:]
}

func getRandomAlphabet(randomFunc func() float64) string {
	return string(alphabet[getRandomInt(randomFunc, int64(AlphabetSize))])
}

// getRandomInt converts a random float64 between 0 and 1 into an integer in the range [0, max-1].
func getRandomInt(randomFunc func() float64, max int64) int64 {
	return int64(math.Floor(randomFunc() * float64(max)))
}
