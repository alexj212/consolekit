-include .env
export $(shell [ -f ".env" ] && sed 's/=.*//' .env)



export DATE := $(shell date +%Y.%m.%d-%H%M)
export LATEST_COMMIT := $(shell git log --pretty=format:'%h' -n 1)
export BRANCH := $(shell git branch |grep -v "no branch"| grep \*|cut -d ' ' -f2)
export GIT_REPO := $(shell git config --get remote.origin.url  2> /dev/null)

export BUILT_ON_OS=$(shell uname -a)
ifeq ($(BRANCH),)
BRANCH := master
endif

export VERSION_FILE   := version.txt
export TAG     := $(shell [ -f "$(VERSION_FILE)" ] && cat "$(VERSION_FILE)" || echo '0.5.46')
export VERMAJMIN      := $(subst ., ,$(TAG))
export VERSION        := $(word 1,$(VERMAJMIN))
export MAJOR          := $(word 2,$(VERMAJMIN))
export MINOR          := $(word 3,$(VERMAJMIN))
export NEW_MINOR      := $(shell expr "$(MINOR)" + 1)
export NEW_TAG := $(VERSION).$(MAJOR).$(NEW_MINOR)




export COMPILE_LDFLAGS=-s -X "main.BuildDate=${DATE}" \
                          -X "main.LatestCommit=${LATEST_COMMIT}" \
						  -X "main.Version=${TAG}"\
						  -X "main.GitRepo=${GIT_REPO}" \
                          -X "main.GitBranch=${BRANCH}"



build_info: check_prereq ## Build the container
	@echo ''
	@echo '---------------------------------------------------------'
	@echo 'VERSION           $(TAG)'
	@echo 'DATE              $(DATE)'
	@echo 'LATEST_COMMIT     $(LATEST_COMMIT)'
	@echo 'BRANCH            $(BRANCH)'
	@echo 'COMPILE_LDFLAGS   $(COMPILE_LDFLAGS)'
	@echo 'PATH              $(PATH)'
	@echo '---------------------------------------------------------'
	@echo ''
	# Delete the go.sum file
	@rm -f go.sum
    # Regenerate it by downloading modules
	@go mod tidy



####################################################################################################################
##
## help for each task - https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
##
####################################################################################################################
.PHONY: help

help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help



####################################################################################################################
##
## Code vetting tools
##
####################################################################################################################


test: ## run tests
	go test -v ./...

fmt: ## run fmt on project	
	@test ! -e ./.history || rm -rf ./.history
	gofmt -s -d -w -l .

doc: ## launch godoc on port 6060
	godoc -http=:6060

deps: ## display deps for project
	go list -f '{{ join .Deps  "\n"}}' . |grep "/" | grep "\." | sort |uniq

lint: ## run lint on the project
	golint ./...

staticcheck: ## run staticcheck on the project
	staticcheck -ignore "$(shell cat .checkignore)" .

vet: ## run go vet on the project
	go vet .

reportcard: ## run goreportcard-cli
	goreportcard-cli -v

tools: ## install dependent tools for code analysis
	go install github.com/gordonklaus/ineffassign@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	go install github.com/gojp/goreportcard/cmd/goreportcard-cli@latest
	go install github.com/goreleaser/goreleaser@latest







####################################################################################################################
##
## Build of binaries
##
####################################################################################################################

all: simple ssh_server multi_transport production_server rest_api tailscale_http ## build all example binaries

binaries: simple ssh_server multi_transport production_server rest_api tailscale_http ## build all example binaries

create_dir:
	@mkdir -p ./build


check_prereq: create_dir


###################################################################################################################
##
## Cleanup of binaries
##
####################################################################################################################

clean: ## clean all binaries in bin dir
	@rm -rf ./build/

simple: build_info create_dir ## build simple example binary
	@echo "build simple"
	CGO_ENABLED=0 go build -o ./build/simple -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/simple
	@echo ''

ssh_server: build_info create_dir ## build ssh_server example binary
	@echo "build ssh_server"
	CGO_ENABLED=0 go build -o ./build/ssh_server -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/ssh_server
	@echo ''

multi_transport: build_info create_dir ## build multi_transport example binary
	@echo "build multi_transport"
	CGO_ENABLED=0 go build -o ./build/multi_transport -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/multi_transport
	@echo ''

production_server: build_info create_dir ## build production_server example binary
	@echo "build production_server"
	CGO_ENABLED=0 go build -o ./build/production_server -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/production_server
	@echo ''

rest_api: build_info create_dir ## build rest_api example binary
	@echo "build rest_api"
	CGO_ENABLED=0 go build -o ./build/rest_api -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/rest_api
	@echo ''

tailscale_http: build_info create_dir ## build tailscale_http example binary
	@echo "build tailscale_http"
	CGO_ENABLED=0 go build -o ./build/tailscale_http -a -ldflags '$(COMPILE_LDFLAGS)' github.com/alexj212/consolekit/examples/tailscale_http
	@echo ''




publish: ## tag & push to git repo
	echo "$(NEW_TAG)" > "$(VERSION_FILE)"
	@echo "\n\n\n\nRunning git add\n"
	git add -A
	@echo "\n\n\nRunning git commit v$(NEW_TAG)\n"
	git commit -m "latest version: v$(NEW_TAG)"
	@echo "\n\n\nRunning git tag $(NEW_TAG)\n"
	git tag -s "v$(NEW_TAG)" -m "publishing version: v$(NEW_TAG)"
	@echo "\n\n\nRunning git push $(NEW_TAG)\n"
	git push -f origin "v$(NEW_TAG)"
	git push


upgrade:
	go get -u ./...
	go mod tidy


