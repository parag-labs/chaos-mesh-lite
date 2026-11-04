# Changelog

All notable changes to this project are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this project uses [semantic versioning](https://semver.org/).

## [Unreleased]

### Changed

- Toolchains: Python 3.15, Java 25 LTS, Go 1.27, Rust edition 2024 (1.99+), Node 26 LTS; .NET stays on 10 (pinned via `global.json`).
- Test frameworks: xUnit v3 on Microsoft.Testing.Platform, JUnit 6, TypeScript 7, Vitest 5.
- CI uses the latest `actions/*` majors and reads versions from each port's version file.

## [0.1.0] - 2026-08-27

First public release. deterministic fault injection with SLO pass/fail evaluation, in Python, C#, and Java.

[0.1.0]: https://github.com/parag-labs/chaos-mesh-lite/releases/tag/v0.1.0
