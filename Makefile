TARGET        = bin/server
BUNDLE        = server/data/static/build/bundle.js
EMBEDDED_DATA = server/rice-box.go
NODE_BIN      = $(shell npm bin)
GO_FILES      = $(shell find ./server -type f -name "*.go")
APP_FILES     = $(shell find client -type f)
GIT_HASH      = $(shell git rev-parse HEAD)
LDFLAGS       = -X main.commitHash=$(GIT_HASH)

build: clean $(TARGET)

$(TARGET): $(EMBEDDED_DATA) $(GO_FILES)
	@go build -i -ldflags '$(LDFLAGS)' -o $@ server/*.go
	@go build -ldflags '$(LDFLAGS)' -o $@ server/*.go

$(EMBEDDED_DATA): $(BUNDLE)
	@(cd server; rice embed -v)

$(BUNDLE): $(APP_FILES) $(NODE_BIN)/webpack
	@$(NODE_BIN)/webpack --progress --colors --bail

$(NODE_BIN)/webpack:
	npm install

clean:
	@rm -rf server/data/static/build # includes $(BUNDLE)
	@rm -f $(TARGET) $(EMBEDDED_DATA)

lint:
	@eslint client || true
	@golint $(GO_FILES) || true
