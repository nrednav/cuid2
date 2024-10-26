# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.1] - 2024-10-26

### Fixed

- Fixed validation of non-CUID strings by `IsCuid()`
  - Added new criteria (starts with a letter) to the validation regex
- Fixed security alert for `crypto` package
  - Updated from v0.10.0 to v0.17.0 via Dependabot

## [v1.0.0] - 2023-09-17

- First release
