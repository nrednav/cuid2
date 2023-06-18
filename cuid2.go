package cuid2

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	randomFunc     func() float64
	sessionCounter func() int64
	length         int
	fingerprint    string
}

type Option func(*Config) error

// Initializes the Cuid2 generator with default or user-defined config options
//
// Returns a function that can be called to generate Cuids using the initialized config
func Init(options ...Option) (func() string, error) {
	config := &Config{
		randomFunc:     rand.Float64,
		sessionCounter: createCounter(int64(math.Floor(rand.Float64() * float64(MaxSessionCount)))),
		length:         DefaultIdLength,
		fingerprint:    createFingerprint(rand.Float64, getEnvironmentKeyString()),
	}

	for _, option := range options {
		if option != nil {
			if applyErr := option(config); applyErr != nil {
				return func() string { return "" }, applyErr
			}
		}
	}

	return func() string {
		firstLetter := getRandomAlphabet(config.randomFunc)
		time := strconv.FormatInt(time.Now().UnixMilli(), 36)
		count := strconv.FormatInt(config.sessionCounter(), 36)
		salt := createEntropy(config.length, config.randomFunc)
		hashInput := time + salt + count + config.fingerprint
		hashDigest := firstLetter + hash(hashInput)[1:config.length]

		return hashDigest
	}, nil
}

// Generates Cuids using default config options
var Generate, _ = Init()

// Checks whether a given Cuid has a valid form and length
func IsCuid(id string) bool {
	length := len(id)
	idMatchesRegex, _ := regexp.MatchString("^[0-9a-z]+$", id)

	if idMatchesRegex && length >= MinIdLength && length <= MaxIdLength {
		return true
	}

	return false
}

// A custom function that will generate a random floating-point value between 0 and 1
func WithRandomFunc(randomFunc func() float64) Option {
	return func(c *Config) error {
		randomness := randomFunc()
		if randomness < 0 || randomness > 1 {
			return fmt.Errorf("Error: the provided random function does not generate a value between 0 and 1")
		}
		c.randomFunc = randomFunc
		return nil
	}
}

// A custom function that will be used to start a session
// counter which affects the entropy of successive id generation calls
func WithSessionCounter(sessionCounter func() int64) Option {
	return func(c *Config) error {
		c.sessionCounter = sessionCounter
		return nil
	}
}

// Configures the length of the generated Cuid
//
// Max Length = 32
// Min Length = 2
func WithLength(length int) Option {
	return func(c *Config) error {
		if length > MaxIdLength {
			return fmt.Errorf("Error: Can only generate id's with a max length of %v", length)
		}
		c.length = length
		return nil
	}
}

// A unique fingerprint that will be used by the id generator to help prevent
// collisions when generating id's in a distributed system.
func WithFingerprint(fingerprint string) Option {
	return func(c *Config) error {
		c.fingerprint = fingerprint
		return nil
	}
}

func createCounter(initialCount int64) func() int64 {
	return func() int64 {
		initialCount++
		return initialCount - 1
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
	bigInt := big.Int{}
	bigInt.SetBytes(hashDigest)
	return bigInt.Text(36)[1:]
}

func getRandomAlphabet(randomFunc func() float64) string {
	alphabets := "abcdefghijklmnopqrstuvwxyz"
	randomIndex := int64(math.Floor(randomFunc() * 26))
	randomAlphabet := string(alphabets[randomIndex])
	return randomAlphabet
}
