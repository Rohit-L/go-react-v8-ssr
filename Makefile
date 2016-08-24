TARGET        = bin/server
BUNDLE        = server/data/static/build/bundle.js
NODE_BIN      = $(shell npm bin)
GO_FILES      = $(shell find ./server -type f -name "*.go")
APP_FILES     = $(shell find client -type f)
GIT_HASH      = $(shell git rev-parse HEAD)
LDFLAGS       = -w -X main.commitHash=$(GIT_HASH)

build: clean $(TARGET)

$(TARGET): $(BUNDLE) $(GO_FILES)
	@(cd server; rice embed -v)
	@go build -i -ldflags '$(LDFLAGS)' -o $@ $(GO_FILES)
	@go build -ldflags '$(LDFLAGS)' -o $@ $(GO_FILES)

$(BUNDLE): $(APP_FILES) $(NODE_BIN)/webpack
	@$(NODE_BIN)/webpack --progress --colors --bail

$(NODE_BIN)/webpack:
	npm install

clean:
	@rm -rf server/data/static/build
	@rm -f $(TARGET)

lint:
	@eslint client || true
	@golint $(GO_FILES) || true
