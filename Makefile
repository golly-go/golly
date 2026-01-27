GO=go
GOCOVER=$(GO) tool cover
GOTEST=TZ=UTC $(GO) test -tags $(TAGS)
GOVET=go vet -tags $(TAGS)
TAGS=test


###
# Development Commands
###

development: dev

dev:
	air -c .air.toml


migrate:
	$(GO) run main.go migration migrate

migration: 
	$(GO) run main.go migration generate $(ARGS)

### 
# Application Development Dependencies
###

install-gocov2lcov:
	$(GO) install github.com/jandelgado/gocov2lcov@latest
	brew install lcov

install-lint:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest


init-dev-deps: install-gocov2lcov install-lint


install-vulnerability-scanner:
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

###
# Tools
###

vulnerability:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

deps:
	go mod download

vet:
	$(GOVET) ./...


###
# Test & Lint Commands
###

test: vet
	$(GOTEST) ./... -cover -coverprofile=coverage.out

bench:
	$(GOTEST) ./... -benchmem -bench ./...

coverage:
	$(GOCOVER) -func=coverage.out
	@unlink coverage.out

lint:
	golangci-lint run -v --build-tags=$(TAGS) --timeout=5m

.PHONY: test/cover
test/cover: coverage

# reevaluate the coverage exclude list
cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	gcov2lcov -infile coverage.out -outfile lcov.info
	genhtml lcov.info \
		--output-directory coverage-site \
		--title "Apext – Go Coverage" \
		--ignore-errors unused \
		--exclude '*/**/_mock.go' \
		--exclude '**/*_mock.go' \
		--exclude '**/*mocks.go' \
		--exclude '*/mocks/*' \
		--exclude '*/**/mock.go' \
		--exclude '**/factory.go' \
		--exclude '*/**/factory.go' \
		--exclude '**/factories.go' \
		--exclude '*/**/factories.go' \
		--exclude 'tools/**' \
		--exclude '**/initializer.go' \
		--exclude '**/routes.go' \
		--exclude '**/testharness/*' \
		--legend --show-details \
		--rc genhtml_med_limit=65 \
  		--rc genhtml_hi_limit=80
	@unlink coverage.out
	@unlink lcov.info
	@echo "\n✅ Coverage report generated at: coverage-site/index.html"


ci-deps: deps install-lint

ci: ci-deps lint test 

