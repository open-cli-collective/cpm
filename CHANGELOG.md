# Changelog

## 1.0.0 (2026-01-23)


### Features

* **claude:** add client interface and implementation ([2512c5e](https://github.com/open-cli-collective/claude-plugin-manager/commit/2512c5ebe1ac1a90bf1d92f1ddd9a177c90e9ea5))
* **claude:** add type definitions for plugin data structures ([efcd424](https://github.com/open-cli-collective/claude-plugin-manager/commit/efcd424c5261c958107ab8e381fb8dedf8bfa4c8))
* **main:** add entry point with version/help flags and claude check ([b633761](https://github.com/open-cli-collective/claude-plugin-manager/commit/b6337615dab3d5e94874311d9b8d52a797ae3671))
* **tui:** add core model with plugin loading ([6ff9a82](https://github.com/open-cli-collective/claude-plugin-manager/commit/6ff9a82c426535dfb52be5e4ea2df2a84c87ba0f))
* **tui:** add execution flow with confirmation, progress, and error modals ([f286335](https://github.com/open-cli-collective/claude-plugin-manager/commit/f2863356c57ef153e5604ce0f319ec84e8e4315f))
* **tui:** add filter input view and help text updates ([09eb799](https://github.com/open-cli-collective/claude-plugin-manager/commit/09eb799186ad7704053e18422f7d7b266cee3391))
* **tui:** add filter mode, refresh handler, and mouse support ([2ce0d91](https://github.com/open-cli-collective/claude-plugin-manager/commit/2ce0d916e5d39ffc3109ea345be70c07f96f172a))
* **tui:** add filter state fields and quit confirmation state ([d537c1c](https://github.com/open-cli-collective/claude-plugin-manager/commit/d537c1c631cc26f7f2fe574733585083c5eeccbc))
* **tui:** add key binding definitions ([818e9e2](https://github.com/open-cli-collective/claude-plugin-manager/commit/818e9e2012f0aa1a4e278aa9567b8d070a02e356))
* **tui:** add lip gloss styles for two-pane layout ([efe9602](https://github.com/open-cli-collective/claude-plugin-manager/commit/efe96020841c6291b9b1bf2c8f29875bcbafd8b4))
* **tui:** add mouse toggle with 'm' key ([d75a7ac](https://github.com/open-cli-collective/claude-plugin-manager/commit/d75a7accd0029f9840e98976fd276e45e2834750))
* **tui:** add selection key handlers for l/p/Tab/u/Esc ([f54f6f9](https://github.com/open-cli-collective/claude-plugin-manager/commit/f54f6f93e3a45970377ef716922277ee16825ecd))
* **tui:** add two-pane view rendering ([7081672](https://github.com/open-cli-collective/claude-plugin-manager/commit/708167223618618bee496b81c860840e378d31e9))
* **tui:** enable mouse cell motion support ([2ffe050](https://github.com/open-cli-collective/claude-plugin-manager/commit/2ffe050f4f51bb96bdbbb9477901e128d606981a))
* **tui:** integrate styles, keys, view into model ([b68510d](https://github.com/open-cli-collective/claude-plugin-manager/commit/b68510d1fd9b60d354f8e9389c3913625003d7ad))
* **tui:** show components and author for uninstalled plugins ([e4fd564](https://github.com/open-cli-collective/claude-plugin-manager/commit/e4fd564c54664892e52a7ee03dc26efbb3d0b3e4))
* **tui:** show external plugin indicator for URL-based plugins ([0b8870b](https://github.com/open-cli-collective/claude-plugin-manager/commit/0b8870b98505de3a8db2a2fca37ece3a284c5ab0))
* **tui:** show plugin descriptions, components, and author info ([79eba0d](https://github.com/open-cli-collective/claude-plugin-manager/commit/79eba0dd673f26218f95e29dc56bcf707394c277))
* use mise for CI and add branch to version info ([e1b76a5](https://github.com/open-cli-collective/claude-plugin-manager/commit/e1b76a5159ab2be7b9f13ff0659d662fedd6ff87))
* **version:** add version package with build-time injection ([6b3fd3b](https://github.com/open-cli-collective/claude-plugin-manager/commit/6b3fd3b393fe313c011b91de7510c4c56017ca23))


### Bug Fixes

* address code review feedback for Phase 8 ([f10bbf9](https://github.com/open-cli-collective/claude-plugin-manager/commit/f10bbf92322054b10ec95b6dcf0ea25fe3963db7))
* address linting issues ([5917337](https://github.com/open-cli-collective/claude-plugin-manager/commit/5917337a9af6ab00637e369a4524c05333a42d3c))
* **claude:** resolve linting issues - field alignment and gosec suppressions ([b967308](https://github.com/open-cli-collective/claude-plugin-manager/commit/b967308079d81907c8e2dea28817798465535634))
* **config:** add version: \"2\" to golangci.yml ([335bc86](https://github.com/open-cli-collective/claude-plugin-manager/commit/335bc8665deb4dcbdbb4f9f00883f9510a9efefb))
* remove invalid version field from golangci-lint config ([abe1e9e](https://github.com/open-cli-collective/claude-plugin-manager/commit/abe1e9e69bce270871aef2c3103d9c6c70429a07))
* resolve golangci-lint issues ([ed143ed](https://github.com/open-cli-collective/claude-plugin-manager/commit/ed143ed9cfe0cffceb62c157e9fcb8e01a935ed1))
* **tui:** address bugs from manual testing ([9939fe4](https://github.com/open-cli-collective/claude-plugin-manager/commit/9939fe40edde6ea31d9c11dbb70f807b7a970e3d))
* **tui:** address Phase 6 code review feedback ([5ea2589](https://github.com/open-cli-collective/claude-plugin-manager/commit/5ea25898381182bc931b7005ca85c522d6f29ab1))
* **tui:** filter installed plugins by current working directory ([95ea156](https://github.com/open-cli-collective/claude-plugin-manager/commit/95ea156d912c09bc439e63761afab0fa90b86181))
* **tui:** fix mouse click off-by-one error ([239ccf0](https://github.com/open-cli-collective/claude-plugin-manager/commit/239ccf0d3a2b742f52d028de4777132e26c16857))
* **tui:** fix selection highlighting for bottom items ([63ac3ce](https://github.com/open-cli-collective/claude-plugin-manager/commit/63ac3ceda341c4b4286086632a56310a9d14b683))
* **tui:** handle worktrees and prevent duplicate plugins ([de57116](https://github.com/open-cli-collective/claude-plugin-manager/commit/de57116f41e6d15f9d8497dcf39a33938aa456b8))
* **tui:** resolve lint issues - unused fields and parameters ([26279ce](https://github.com/open-cli-collective/claude-plugin-manager/commit/26279ceaae042be4b81ea8484982952bb6a7415a))
* **tui:** show installed plugins not in available list ([4051af3](https://github.com/open-cli-collective/claude-plugin-manager/commit/4051af3495a9b7f84b0cf3cfde399f5dd6587c05))
