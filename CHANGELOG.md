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