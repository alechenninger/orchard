OUT?=out/orchard

.PHONY: build
build:
	GOFLAGS= CGO_ENABLED=1 go build -o $(OUT) ./cmd/orchard

.PHONY: sign
sign: build entitlements.plist
	codesign -s - --entitlements entitlements.plist --force $(OUT)
	codesign -dv --entitlements - $(OUT) || true

.PHONY: run
run: sign
	./$(OUT)


.PHONY: install
install: sign
	sudo install -m 0755 $(OUT) /usr/local/bin/orchard


