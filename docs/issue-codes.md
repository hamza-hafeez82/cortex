# Issue code registry

All Cortex findings follow the format `CX-{CATEGORY}-{NUMBER}`.

## Security — CX-SEC

| Code | Name | Severity | Languages |
|------|------|----------|-----------|
| CX-SEC-001 | Hardcoded secrets | Critical | All |
| CX-SEC-002 | SQL injection | High | JS, TS, Python, Go, Java |
| CX-SEC-003 | Command injection | High | All |
| CX-SEC-004 | Path traversal | High | All |
| CX-SEC-005 | Insecure random | Medium | All |
| CX-SEC-006 | JWT misconfiguration | High | JS, TS, Python, Go |
| CX-SEC-007 | CORS misconfiguration | Medium | JS, TS, Python, Go |
| CX-SEC-008 | Sensitive data in logs | Medium | All |

## Dependencies — CX-DEP

| Code | Name | Severity | Languages |
|------|------|----------|-----------|
| CX-DEP-001 | Known vulnerable dependency | Critical/High | All |
| CX-DEP-002 | Lockfile drift | Medium | JS, TS |

## Architecture — CX-ARCH

| Code | Name | Severity | Languages |
|------|------|----------|-----------|
| CX-ARCH-001 | God file | Medium | All |
| CX-ARCH-002 | Circular dependency | High | All |
| CX-ARCH-003 | Deep nesting | Medium | All |
| CX-ARCH-004 | Missing error handling | High | Go, JS, TS |
| CX-ARCH-005 | Magic numbers | Low | All |
| CX-ARCH-006 | Dead code | Low | All |
| CX-ARCH-007 | Missing tests | Medium | All |
| CX-ARCH-008 | Inconsistent naming | Low | All |
