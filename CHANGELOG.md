# Changelog

## [2.0.0](https://github.com/meigma/github-token-broker/compare/v1.1.0...v2.0.0) (2026-04-30)


### ⚠ BREAKING CHANGES

* **security:** harden broker deployment boundaries ([#31](https://github.com/meigma/github-token-broker/issues/31))

### Bug Fixes

* **security:** harden broker deployment boundaries ([#31](https://github.com/meigma/github-token-broker/issues/31)) ([3285df2](https://github.com/meigma/github-token-broker/commit/3285df2b36cbb1aeb7f910bf231488852bb51a8d))
* **terraform:** support AWS provider v5 region data ([#33](https://github.com/meigma/github-token-broker/issues/33)) ([a6cc7dd](https://github.com/meigma/github-token-broker/commit/a6cc7dd4e9b804260530d20440b4a08d852a6a01))

## [1.1.0](https://github.com/meigma/github-token-broker/compare/v1.0.0...v1.1.0) (2026-04-23)


### Features

* **terraform:** reusable module for deploying the broker Lambda ([#25](https://github.com/meigma/github-token-broker/issues/25)) ([8409a8a](https://github.com/meigma/github-token-broker/commit/8409a8a806ac1e6c23af40ad29208473c7640224))

## 1.0.0 (2026-04-23)


### Features

* port broker core from reference implementation (phase 1a-1f) ([#4](https://github.com/meigma/github-token-broker/issues/4)) ([30e1f70](https://github.com/meigma/github-token-broker/commit/30e1f7016f940a8f12e4e0dc7ec05c262ef41d10))
* tag-triggered publish pipeline with attestations ([#11](https://github.com/meigma/github-token-broker/issues/11)) ([c51372f](https://github.com/meigma/github-token-broker/commit/c51372f8df3180b01512d068fcb25229d79140ad))


### Bug Fixes

* enforce repository owner before minting tokens ([#7](https://github.com/meigma/github-token-broker/issues/7)) ([6682886](https://github.com/meigma/github-token-broker/commit/66828864904e049b27b1e4e16bb8c6df13d122a1))
