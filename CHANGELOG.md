# Changelog

All notable changes to `lgcorzo/mux` will be documented in this file.

## [v1.10.2] - 2026-10-10

### Security & Static Analysis
- **Go Security Remediation**: Updated `go.mod` compatibility configuration to remediate standard library vulnerability alerts and ensure clean `govulncheck` execution.
- **Static Analysis**: Resolved `errcheck` warnings in `aistor_full_bench_test.go` and `aistor_routing_test.go` by explicitly checking/handling `router.Walk` return values.
- **Verification**: Ensured `go test -race ./...`, `golangci-lint`, `gosec`, and `govulncheck` pass cleanly with zero warnings/vulnerabilities.
