# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2020-02-06
## Added
- Initial buffered bit `Reader` implementation.
- Add `Reset` method to `Writer` to complement `Reader.Reset`.

## Changed
- `Writer.Write` now returns `ErrInvalidCount` when the provided count is invalid (details in documentation). This matches the newly created `Reader` implementation.

## [0.1.0] - 2020-02-05
### Added
- Initial buffered bit `Writer` implementation.
