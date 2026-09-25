# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.1.1] - 2026-09-25

### Added

- Hardening tests for the random-function contract, lazy initialization,
  entropy-read count, environment parsing, and the collision histogram bound

### Changed

- Security
  - `WithRandomFunc` rejects NaN and any value outside `[0, 1)` at
    configuration time and validates every subsequent call, failing closed
    instead of silently collapsing or crashing
  - `Init` now returns a nil generator on error, instead of a closure that
    yields an empty string
  - A transient entropy failure during lazy initialization is retried on the
    next `Generate()` call instead of permanently disabling the generator
- Performance
  - The default random source reads all required entropy in a single
    `crypto/rand` read per id, replacing per-value `big.Int` and `big.Float`
    allocations
  - `IsCuid` uses a precompiled regular expression instead of recompiling on
    every call
- Parity
  - The fingerprint is truncated to 32 characters, matching the JavaScript
    reference

### Fixed

- `getEnvironmentKeyString` no longer panics on an environment entry without
  an `=` separator
- The collision-test histogram clamps the top of the range into the last bin
- `AlphabetSize` is derived from a single alphabet definition
- The README custom-counter example now compiles

## [v1.1.0] - 2025-07-07

### Added

- Added deterministic tests to verify the correctness of the core ID generation
  algorithm
- Added an aggregate progress reporter to the collision test suite for better
  feedback during long test runs

### Changed

- Security
  - The default RNG has been updated to use the cryptographically secure
    `crypto/rand`, instead of the insecure `math/rand`
  - The library is now secure by default
- Performance
  - Optimized entropy generation by replacing inefficient string concatenation
    with usage of `strings.Builder`. This reduces memory allocations.
  - Refactored collision test suite to be faster and use orders of magnitude
    less memory.
- Other
  - The `Generate()` function is now initialized lazily and safely on first use
    via `sync.Once`
  - The long-running collision test has been separated from the default test
    suite via a build tag, to speed up testing during development

### Fixed

- The default fingerprint generation is now fully deterministic, as it sorts
  environment variable keys, ensuring the generated fingerprint is stable across
  different platforms

## [v1.0.1] - 2024-10-26

### Fixed

- Fixed validation of non-CUID strings by `IsCuid()`
  - Added new criteria (starts with a letter) to the validation regex
- Fixed security alert for `crypto` package
  - Updated from v0.10.0 to v0.17.0 via Dependabot

## [v1.0.0] - 2023-09-17

- First release
