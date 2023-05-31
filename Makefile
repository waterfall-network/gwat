# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gwat android ios gwat-cross evm all test clean
.PHONY: gwat-linux gwat-linux-386 gwat-linux-amd64 gwat-linux-mips64 gwat-linux-mips64le
.PHONY: gwat-linux-arm gwat-linux-arm-5 gwat-linux-arm-6 gwat-linux-arm-7 gwat-linux-arm64
.PHONY: gwat-darwin gwat-darwin-386 gwat-darwin-amd64
.PHONY: gwat-windows gwat-windows-386 gwat-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

gwat:
	$(GORUN) build/ci.go install ./cmd/gwat
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gwat\" to launch gwat."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gwat.aar\" to use the library."
	@echo "Import \"$(GOBIN)/gwat-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gwat.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/kevinburke/go-bindata/go-bindata@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gwat-cross: gwat-linux gwat-darwin gwat-windows gwat-android gwat-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gwat-*

gwat-linux: gwat-linux-386 gwat-linux-amd64 gwat-linux-arm gwat-linux-mips64 gwat-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-*

gwat-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gwat
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep 386

gwat-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gwat
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep amd64

gwat-linux-arm: gwat-linux-arm-5 gwat-linux-arm-6 gwat-linux-arm-7 gwat-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm

gwat-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gwat
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-5

gwat-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gwat
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-6

gwat-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gwat
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-7

gwat-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gwat
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm64

gwat-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips

gwat-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mipsle

gwat-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips64

gwat-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips64le

gwat-darwin: gwat-darwin-386 gwat-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-*

gwat-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gwat
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-* | grep 386

gwat-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gwat
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-* | grep amd64

gwat-windows: gwat-windows-386 gwat-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-*

gwat-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gwat
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-* | grep 386

gwat-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gwat
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-* | grep amd64
