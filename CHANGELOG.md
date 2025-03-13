# Changelog

## [0.0.2] - 2025-03-13
### Changed
- Moved `go.mod` and `go.sum` files to the root of the project.
- Removed `go.mod` and `go.sum` from `core/` and `pg/` directories.
- Centralized dependency management in the root directory.

## [0.0.1] - 2025-03-13
### Added
- Initial release of `go-lockbox` with PostgreSQL backend support.
- Implemented atomic lock acquisition with Postgres.
- Added TTL control and renewal.
- Basic distributed lock acquisition and TTL management.