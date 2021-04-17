# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: secret android ios secret-cross evm all test clean
.PHONY: secret-linux secret-linux-386 secret-linux-amd64 secret-linux-mips64 secret-linux-mips64le
.PHONY: secret-linux-arm secret-linux-arm-5 secret-linux-arm-6 secret-linux-arm-7 secret-linux-arm64
.PHONY: secret-darwin secret-darwin-386 secret-darwin-amd64
.PHONY: secret-windows secret-windows-386 secret-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on CGO_CFLAGS=-Wno-undef-prefix go run

secret:
	$(GORUN) build/ci.go install ./cmd/secret
	@echo "Done building."
	@echo "Run \"$(GOBIN)/secret\" to launch secret."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/secret.aar\" to use the library."
	@echo "Import \"$(GOBIN)/secret-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/secret.framework\" to use the library."

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
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

secret-cross: secret-linux secret-darwin secret-windows secret-android secret-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/secret-*

secret-linux: secret-linux-386 secret-linux-amd64 secret-linux-arm secret-linux-mips64 secret-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-*

secret-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/secret
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep 386

secret-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/secret
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep amd64

secret-linux-arm: secret-linux-arm-5 secret-linux-arm-6 secret-linux-arm-7 secret-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep arm

secret-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/secret
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep arm-5

secret-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/secret
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep arm-6

secret-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/secret
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep arm-7

secret-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/secret
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep arm64

secret-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/secret
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep mips

secret-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/secret
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep mipsle

secret-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/secret
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep mips64

secret-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/secret
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/secret-linux-* | grep mips64le

secret-darwin: secret-darwin-386 secret-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/secret-darwin-*

secret-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/secret
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/secret-darwin-* | grep 386

secret-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/secret
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/secret-darwin-* | grep amd64

secret-windows: secret-windows-386 secret-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/secret-windows-*

secret-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/secret
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/secret-windows-* | grep 386

secret-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/secret
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/secret-windows-* | grep amd64
