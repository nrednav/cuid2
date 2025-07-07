package cuid2

import (
	"fmt"
	"math"
	"math/big"
	"crypto/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"sync"
	"time"

	"golang.org/x/crypto/sha3"
)

const (
	DefaultIdLength int = 24
	MinIdLength     int = 2
	MaxIdLength     int = 32

	// ~22k hosts before 50% chance of initial counter collision
	MaxSessionCount int64 = 476782367
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
	length int
	counter Counter
	fingerprint string
}

type Option func(*Config) error

// Initializes the Cuid generator with default or user-defined config options
//
// Returns a function that can be called to generate Cuids using the initialized config
func Init(options ...Option) (func() string, error) {
	defaultRandomFunc := newRandomFunc()

	initialSessionCount := int64(
		math.Floor(defaultRandomFunc() * float64(MaxSessionCount)),
	)

	config := &Config{
		RandomFunc:     defaultRandomFunc,
		SessionCounter: NewSessionCounter(initialSessionCount),
		Length:         DefaultIdLength,
		Fingerprint:    createFingerprint(defaultRandomFunc, getEnvironmentKeyString()),
	}

	for _, option := range options {
		if option != nil {
			if applyErr := option(config); applyErr != nil {
				return func() string { return "" }, applyErr
			}
		}
	}

	g := &cuidGenerator{
		length: config.Length,
		counter: config.SessionCounter,
		fingerprint: config.Fingerprint,
	}

	return func() string {
		return g.generate(time.Now().UnixMilli(), config.RandomFunc)
	}, nil
}

func (g *cuidGenerator) generate(timeMs int64, randomFunc func() float64) string {
	firstLetter := getRandomAlphabet(randomFunc)
	timeStr := strconv.FormatInt(timeMs, 36)
	countStr := strconv.FormatInt(g.counter.Increment(), 36)
	salt := createEntropy(g.length, randomFunc)
	hashInput := timeStr + salt + countStr + g.fingerprint

	return firstLetter + hash(hashInput)[1:g.length]
}

var (
	defaultGenerator func() string
	initOnce sync.Once
)

// Generate returns a CUID using the default configuration.
// The default generator is initialized lazily and safely on the first call.
func Generate() string {
	initOnce.Do(func() {
		defaultGenerator, _ = Init()
	})

	return defaultGenerator()
}

// Checks whether a given Cuid has a valid form and length
func IsCuid(cuid string) bool {
	length := len(cuid)
	hasValidForm, _ := regexp.MatchString("^[a-z][0-9a-z]+$", cuid)

	if hasValidForm && length >= MinIdLength && length <= MaxIdLength {
		return true
	}

	return false
}

// A custom function that will generate a random floating-point value between 0 and 1
func WithRandomFunc(randomFunc func() float64) Option {
	return func(config *Config) error {
		randomness := randomFunc()
		if randomness < 0 || randomness > 1 {
			return fmt.Errorf("Error: the provided random function does not generate a value between 0 and 1")
		}
		config.RandomFunc = randomFunc
		return nil
	}
}

// A custom counter that will be used to affect the entropy of successive id
// generation calls
func WithSessionCounter(sessionCounter Counter) Option {
	return func(config *Config) error {
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

// Returns a function that provides a cryptographically secure random float64
// value between 0.0 and 1.0.
// It panics if the OS's source of entropy is unavailable.
func newRandomFunc() func() float64 {
	// max is 2^53 - 1, the largest integer that can be represented exactly by a float64
	maxInt := new(big.Int).Lsh(big.NewInt(1), 53)
	maxFloat := new(big.Float).SetInt(maxInt)

	return func() float64 {
		randomInt, err := rand.Int(rand.Reader, maxInt)

		if err != nil {
			panic(fmt.Errorf("Error: Failed to read from crypto/rand: %w", err))
		}

		randomFloat := new(big.Float).SetInt(randomInt)
		randomFloat.Quo(randomFloat, maxFloat)
		randomFloatValue, _ := randomFloat.Float64()

		return randomFloatValue
	}
}

func createFingerprint(randomFunc func() float64, envKeyString string) string {
	sourceString := createEntropy(MaxIdLength, randomFunc)

	if len(envKeyString) > 0 {
		sourceString += envKeyString
	}

	sourceStringHash := hash(sourceString)

	return sourceStringHash[1:]
}

func createEntropy(length int, randomFunc func() float64) string {
	entropy := ""

	for len(entropy) < length {
		randomness := int64(math.Floor(randomFunc() * 36))
		entropy += strconv.FormatInt(randomness, 36)
	}

	return entropy
}

func getEnvironmentKeyString() string {
	env := os.Environ()

	keys := []string{}

	// Discard values of environment variables
	for _, variable := range env {
		key := variable[:strings.IndexByte(variable, '=')]
		keys = append(keys, key)
	}

	return strings.Join(keys, "")
}

func hash(input string) string {
	hash := sha3.New512()
	hash.Write([]byte(input))
	hashDigest := hash.Sum(nil)
	return new(big.Int).SetBytes(hashDigest).Text(36)[1:]
}

func getRandomAlphabet(randomFunc func() float64) string {
	alphabets := "abcdefghijklmnopqrstuvwxyz"
	randomIndex := int64(math.Floor(randomFunc() * 26))
	randomAlphabet := string(alphabets[randomIndex])
	return randomAlphabet
}
