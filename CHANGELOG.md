## [1.2.0] - 2026-09-20

### Added
- **WebSocket authentication support** — subscription tests can now authenticate
- `Connect()` reads `tester.CurrentToken()` and includes it in:
  - HTTP handshake headers (`Authorization: Bearer <token>`)
  - `connection_init` payload (`{"Authorization": "Bearer <token>"}`)
- **Error message handling** — `readPump()` now processes `type: "error"` messages
- **`connection_error` support** — handles servers that send this message type
- **New assertion methods on `Subscription`:**
  - `ExpectError(contains, timeout)` — asserts an error is received
  - `ExpectNoError(timeout)` — asserts no error arrives
- **New assertion methods on `SubscriptionAssertions`:**
  - `AssertErrorContains(contains)` — fluent error assertion
  - `AssertUnauthenticated()` — asserts "Unauthenticated" error
  - `AssertPermissionDenied()` — asserts permission error
  - `AssertForbidden()` — asserts "Forbidden" error
- **`tester.SignInAndGetToken(email, password)`** — convenience method for logging in via GraphQL
- **`Subscription.Errors` field** — accumulates errors received during the subscription
- `errorMessages()` helper for extracting message strings from `[]*GraphQLError`

### Fixed
- Error messages from server were previously silently ignored during subscriptions
- Subscriptions can now be tested with the full auth flow

### Compatible With
- `github.com/mwangaben/graphql-kit` v0.2.0+
- Any graphql-ws compatible server with connection_init payload auth
- HTTP header auth (Authorization: Bearer <token>) during WS handshake

### Test Coverage
- `TestSubscription_Unauthenticated` — verifies reject without auth
- `TestSubscription_Authenticated` — verifies success with auth


## [1.1.3] - 2026-09-20

### Added
- **Ent ORM adapter** — full database support for Ent-backed GraphQL APIs
    - MySQL / MariaDB, PostgreSQL, SQLite
    - Insert, Update, Delete, Count, GetRecord(s), HasRecord
    - Transaction support (BeginTx, Commit, Rollback)
    - Soft-delete detection
    - Automatic dialect detection from DSN
    - Integration with `graphqltester` assertions
- Comprehensive Ent adapter tests
- Integration test proving EntAdapter works end-to-end with the tester

### Documentation
- Full Ent section in README with usage examples
- GraphQL type mapping guide (int32 for Int fields, *string for nullable)

### Compatible With
- `github.com/mwangaben/graphqltester` v1.0.0+
- All graph-gophers/graphql-go versions
- Ent v0.12+