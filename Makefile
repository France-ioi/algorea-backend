GOCMD=env GO111MODULE=auto go
GOBUILD=CGO_ENABLED=0 $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOLIST=$(GOCMD) list
BIN_NAME=AlgoreaBackend

TEST_REPORT_DIR=test-results

LOCAL_BIN_DIR=./bin

BIN_PATH=$(LOCAL_BIN_DIR)/$(BIN_NAME)
GOLANGCILINT=$(LOCAL_BIN_DIR)/golangci-lint
GOLANGCILINT_VERSION=1.52.2
MYSQL_CONNECTOR_JAVA=$(LOCAL_BIN_DIR)/mysql-connector-java-8.jar
SCHEMASPY=$(LOCAL_BIN_DIR)/schemaspy-6.0.0.jar
PWD=$(shell pwd)

VERSION_FETCHING_CMD=git describe --always --dirty
GOBUILD_VERSION_INJECTION=-ldflags="-X main.version=$(shell $(VERSION_FETCHING_CMD))"

# Filter for tests
ifdef FILTER
	TEST_FILTER=-run $(FILTER)
endif
ifdef DIRECTORY
	TEST_DIR=$(DIRECTORY)
else
	TEST_DIR=./app/...
endif
ifdef TAGS
	TEST_TAGS=--godog.tags=$(TAGS)
endif

# extract AWS_PROFILE if given
ifdef AWS_PROFILE
	AWS_PARAMS=--profile $(AWS_PROFILE)
endif

# use the NOECHO env var to disable the echo (printing the executed commands) on make test
ifeq ("$(NOECHO)","1")
	Q := @
	vecho = @true
else
	Q :=
	vecho = @echo
endif

# Check that given variables are set and all have non-empty values,
# die with an error otherwise.
#
# Params:
#   1. Variable name(s) to test.
#   2. (optional) Error message to print.
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))

all: build
build: $(BIN_PATH)
$(BIN_PATH): .FORCE # let go decide what need to be rebuilt
	$(GOBUILD) -o $(BIN_PATH) -v -tags=prod $(GOBUILD_VERSION_INJECTION)
gen-keys:
	openssl genpkey -algorithm RSA -out private_key.pem 2>/dev/null | openssl genrsa -out private_key.pem 1024
	openssl rsa -pubout -in private_key.pem -out public_key.pem
db-restore: $(BIN_PATH)
	$(BIN_PATH) db-restore
db-migrate: $(BIN_PATH)
	$(BIN_PATH) db-migrate
db-migrate-undo: $(BIN_PATH)
	$(BIN_PATH) db-migrate-undo
db-recompute: $(BIN_PATH)
	$(BIN_PATH) db-recompute

test: $(TEST_REPORT_DIR)
	$(Q)# TODO: the tests using the db do not currently support parallelism
	$(Q)# add DIRECTORY=./app/api/item to only test a certain directory. Must start with ".".
	$(Q)# Warning: DIRECTORY must be a directory, it will fail if it is a file
	$(Q)# add FILTER=functionToTest to only test a certain function. functionToTest is a Regex.

	$(Q)$(GOTEST) -gcflags=all=-l -race -coverprofile=$(TEST_REPORT_DIR)/coverage.txt -covermode=atomic -v $(TEST_DIR) -p 1 -parallel 1 $(TEST_FILTER)
test-dev:
	$(Q)$(GOTEST) -gcflags=all=-l $(TEST_DIR) -p 1 -parallel 1 $(TEST_FILTER)
test-unit:
	$(GOTEST) -gcflags=all=-l -race -cover -v -tags=unit $(TEST_DIR) $(TEST_FILTER)
test-bdd:
	# to pass args: make TAGS=wip test-bdd
	$(Q)$(GOTEST) -gcflags=all=-l -race -v -tags=!unit -run TestBDD $(TEST_DIR) -p 1 -parallel 1 $(TEST_TAGS)
lint:
	@[ -e $(GOLANGCILINT) ] && \
		($(GOLANGCILINT) --version | grep -F "version $(GOLANGCILINT_VERSION) built" > /dev/null || rm $(GOLANGCILINT)) || true
	$(MAKE) $(GOLANGCILINT)
	$(GOLANGCILINT) run -v --deadline 10m0s

swagger-generate:
	swagger generate spec --nullable-pointers --scan-models -o ./swagger.yaml && \
		swagger validate ./swagger.yaml && \
		swagger2openapi --refSiblings allOf --yaml swagger.yaml | sed 's/x-nullable:/nullable:/g' > openapi3.yaml && \
		mv openapi3.yaml swagger.yaml && \
		redocly lint --skip-rule security-defined --skip-rule spec --skip-rule no-identical-paths swagger.yaml

swagger-serve: swagger-generate
	redocly preview-docs swagger.yaml

dbdoc: $(MYSQL_CONNECTOR_JAVA) $(SCHEMASPY)
	$(call check_defined, DBNAME)
	$(call check_defined, DBHOST)
	$(call check_defined, DBUSER)
	$(call check_defined, DBPASS)
	java -jar $(SCHEMASPY) -t mysql -dp $(MYSQL_CONNECTOR_JAVA) -db $(DBNAME) -host $(DBHOST) -port 3306 -u $(DBUSER) -p $(DBPASS) -o db/doc -s $(DBNAME) -noimplied -nopages
clean:
	$(GOCLEAN)
	$(GOCLEAN) -testcache
	rm -rf $(LOCAL_BIN_DIR)/*

linux-build:
	GOOS=linux $(GOBUILD) -o $(BIN_PATH)-linux $(GOBUILD_VERSION_INJECTION)

awslambda-build:
	GOARCH=amd64 GOOS=linux go build -o $(BIN_PATH)-awslambda -tags lambda.norpc $(GOBUILD_VERSION_INJECTION)

version:
	@echo $(shell $(VERSION_FETCHING_CMD))

$(TEST_REPORT_DIR):
	mkdir -p $(TEST_REPORT_DIR)
$(GOLANGCILINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCAL_BIN_DIR) v$(GOLANGCILINT_VERSION)
$(MYSQL_CONNECTOR_JAVA):
	curl -sfL https://dev.mysql.com/get/Downloads/Connector-J/mysql-connector-java-8.0.16.tar.gz | tar -xzf - mysql-connector-java-8.0.16/mysql-connector-java-8.0.16.jar
	mv mysql-connector-java-8.0.16/mysql-connector-java-8.0.16.jar $(MYSQL_CONNECTOR_JAVA)
	rm -rf mysql-connector-java-8.0.16
$(SCHEMASPY):
	curl -sfL -o $(SCHEMASPY) https://github.com/schemaspy/schemaspy/releases/download/v6.0.0/schemaspy-6.0.0.jar

.FORCE: # force the rule using it to always re-run
.PHONY: all build gen-keys db-restore db-migrate db-migrate-undo db-recompute test test-unit test-bdd lint dbdoc clean linux-build version
