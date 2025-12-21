VERSION		:=$(shell cat .version)
YAML_FILES 	:=$(shell find . ! -path "./vendor/*" -type f -regex ".*y*ml" -print)
REG_URI    	?= example/repo
REPO_NAME  	:=$(shell basename $(PWD))
DB_URI     	?= 
MGRT_NAME  	?=
MGRT_DIR   	:= ./sql/migrations/
MGRT_DIRECTION	?=
PROJ_BIN_PATH	:= ./bin/

all: help

.PHONY: init
init: ## Init tools that used in the project
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/spf13/cobra-cli@latest
	go install go.uber.org/mock/mockgen@latest
	GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@v1.54.0
.PHONY: version
version: ## Prints the current version
	@echo $(VERSION)

.PHONY: tidy
tidy: ## Updates the go modules and vendors all dependancies 
	go mod tidy

.PHONY: upgrade
upgrade: ## Upgrades all dependancies 
	go get -d -u ./...
	go mod tidy
	go mod vendor

.PHONY: test
test: tidy ## Runs unit tests, GO_TEST_ARGS for extra args
	go test -count=1 -race -covermode=atomic -coverprofile=cover.out $$GO_TEST_ARGS ./...

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project 
	@echo "Completed Go and YAML lints"

.PHONY: lint-go
lint-go: ## Lints the entire project using go 
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Runs yamllint on all yaml files (brew install yamllint)
	yamllint -c .yamllint $(YAML_FILES)

.PHONY: vulncheck
vulncheck: ## Checks for soource vulnerabilities
	govulncheck -test ./...

.PHONY: grpc-server
grpc-server: ## Runs uncpiled version of the server, needs env [DB_URI]
	go run main.go server grpc -a localhost:8080 -e dev -n "grpc-server" -l DEBUG -c $(DB_URI)

.PHONY: http-server
http-server: ## Runs uncpiled version of the server, needs env [DB_URI]
	go run main.go server http -a localhost:8080 -e local -n "http-server" -v -l DEBUG -c $(DB_URI)

.PHONY: image
image: ## Builds the server images
	@echo "Building server image..."
	KO_DOCKER_REPO=$(REG_URI)/$(REPO_NAME)-server \
    GOFLAGS="-ldflags=-X=main.version=$(VERSION)" \
    ko build cmd/server/main.go --image-refs .digest --bare --tags $(VERSION),latest

.PHONY: tag
tag: ## Creates release tag 
	git tag -s -m "version bump to $(VERSION)" $(VERSION)
	git push origin $(VERSION)

.PHONY: tagless
tagless: ## Delete the current release tag 
	git tag -d $(VERSION)
	git push --delete origin $(VERSION)

.PHONY: clean
clean: ## Cleans bin and temp directories
	go clean
	rm -fr ./vendor
	rm -fr ./bin

.PHONY: pblint
pblint: ## Lint and format protobuf files
	@echo "Formating protobuf files..."
#	@docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf format
	buf format
	@echo "Linting protobuf files..."
#	@docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf lint
	buf lint
	@echo "Finished"

.PHONY: pbgen
pbgen: ## Generate protobuf
#       docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf generate
	buf generate

.PHONY: dbgen
dbgen: ## Compile sql to type-safe code
	@sqlc generate

.PHONY: mgrt-prep
mgrt-prep: ## Prepare migration files, needs env [MGRT_NAME="init schema"]
	migrate create -ext sql -dir $(MGRT_DIR) -seq $(MGRT_NAME)

.PHONY: mgrt
mgrt: ## Migrate schema, needs env [DB_URI="db connect uri"] [MGRT_DIRECTION=up|down]
	migrate -database $(DB_URI) -path $(MGRT_DIR) $(MGRT_DIRECTION)

.PHONY: build-prod
build-prod: ## Compile binary by disable CGO and omits DWARF symbol table and debug info
	@echo Buidling binary to $(PROJ_BIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags='-w -s -extldflags "-static"' -o $(PROJ_BIN_PATH)$${APP_NAME} .

.PHONY: build build-dev
build: build-dev
build-dev: ## Compile binary by disable CGO and omits DWARF symbol table and debug info
	@echo Buidling binary to $(PROJ_BIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags='-extldflags "-static"' -o $(PROJ_BIN_PATH)$${APP_NAME} .

.PHONY: run
run: ## run quick
	@echo Buidling binary to $(PROJ_BIN_PATH)
	go run ./main.go

.PHONY: mock
mock: ## Generate test mock files for interfaces
	go generate ./...

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
