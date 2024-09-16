## Gwat: A Waterfall Verifier Layer Implementation Written in Go

The Waterfall team forked the project on October 20, 2021, and has since made its own changes.

- [Discord](https://discord.gg/Nwb8aR2XvR)
- [Documentation](https://docs.waterfall.network/)

This is the core repository for Gwat, a [Golang](https://golang.org/) implementation of the [Waterfall Consensus](https://waterfall.network/) specification, developed by Blue Wave Inc.


## Building the source
**We strongly recommend installing go version 1.21.11 or later**

```shell
CGO_CFLAGS="-O2 -D__BLST_PORTABLE__" go run build/ci.go install ./cmd/gwat
```

### Getting Started

A detailed set of installation and usage instructions as well as breakdowns of each individual component are available in the [official documentation portal](https://docs.waterfall.network). If you still have questions, feel free to stop by our [Discord](https://discord.gg/Nwb8aR2XvR).

### Staking on Mainnet

To participate in staking, you see on the [official waterfall site](https://waterfall.network/use-waterfall/stake-water). The launchpad is the only recommended way to become a validator on mainnet.

## Contributing
### Branches
Gwat maintains two permanent branches:

* [main](https://github.com/waterfall-network/gwat/tree/main): This points to the latest stable release. It is ideal for most users.
* [develop](https://github.com/waterfall-network/gwat/tree/develop): This is used for development, it contains the latest PRs. Developers should base their PRs on this branch.

### Guide
Want to get involved? Check out our [Contribution Guide](/CONTRIBUTING.md) to learn more!

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.

The code written for the Waterfall project is distributed under [APACHE LICENSE, VERSION 2.0](https://www.apache.org/licenses/LICENSE-2.0), also
included in our repository in the `LICENSE` file.
