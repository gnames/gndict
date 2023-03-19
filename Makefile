PROJ_NAME = gndict

VERSION = $(shell git describe --tags)
VER = $(shell git describe --tags --abbrev=0)
DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S%Z')

NO_C = CGO_ENABLED=0
FLAGS_SHARED = $(NO_C) GOARCH=amd64
FLAGS_LD = -trimpath -ldflags "-w -s \
					 -X github.com/gnames/$(PROJ_NAME)/pkg.Build=$(DATE) \
           -X github.com/gnames/$(PROJ_NAME)/pkg.Version=$(VERSION)"
FLAGS_REL = -trimpath -ldflags "-s -w \
						-X github.com/gnames/$(PROJ_NAME)/pkg.Build=$(DATE)"

GOCMD =go
GOINSTALL = $(GOCMD) install $(FLAGS_LD)
GOBUILD = $(GOCMD) build $(FLAGS_LD)
GORELEASE = $(GOCMD) build $(FLAGS_REL)
GOCLEAN = $(GOCMD) clean
GOGET = $(GOCMD) get

all: install

test: deps install
	go test -shuffle=on -count=1 -race -coverprofile=coverage.txt -covermode=atomic ./...

tools: deps
	@echo Installing tools from tools.go
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

deps:
	@echo Download go.mod dependencies
	$(GOCMD) mod download;

build:
	$(GOCLEAN); \
	$(NO_C) $(GOBUILD);

buildrel:
	$(GOCLEAN); \
	$(NO_C) $(GORELEASE);

install:
	$(FLAGS_SHARED) $(GOINSTALL);
