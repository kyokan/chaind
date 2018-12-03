# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). Until a 1.0.0 release is cut, expect breaking changes between releases.

## [Unreleased]
### Added
- Initial set of Prometheus metrics.
- Cache warmer that by default caches transaction receipts and block data for the latest 200 finalized blocks.
- Ability to selectively disable Ethereum APIs from the config file. 

### Changed
- Moved the `eth_path` config variable into a dedicated `eth` stanza.

### Fixed
- Fixed a bug that prevented backends declared before the `main` backend from being selected during failover. 