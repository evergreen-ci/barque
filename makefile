# start project configuration
name := barque
buildDir := build
packages := $(name) model rest units operations
orgPath := github.com/evergreen-ci
projectPath := $(orgPath)/$(name)
# end project configuration


# start environment setup
ifneq (,$(GO_BIN_PATH))
 gobin := $(GO_BIN_PATH)
else
 gobin := $(shell if [ -x /opt/golang/go1.9/bin/go ]; then echo /opt/golang/go1.9/bin/go; fi)
 ifeq (,$(gobin))
   gobin := go
 endif
endif

gopath := $(GOPATH)
ifeq ($(OS),Windows_NT)
  gopath := $(shell cygpath -m $(gopath))
  gobin := $(shell cygpath -m $(gobin))
endif
goEnv := GOPATH=$(gopath)$(if $(GO_BIN_PATH), PATH="$(shell dirname $(GO_BIN_PATH)):$(PATH)")
# end environment setup


# start lint setup targets
lintDeps := $(buildDir)/golangci-lint $(buildDir)/.lintSetup $(buildDir)/run-linter
$(buildDir)/.lintSetup:$(buildDir)/golangci-lint
	@mkdir -p $(buildDir)
	@touch $@
$(buildDir)/golangci-lint:
	@curl  --retry 10 --retry-max-time 60 -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/76a82c6ed19784036bbf2d4c84d0228ca12381a4/install.sh | sh -s -- -b $(buildDir) v1.23.8 >/dev/null 2>&1
$(buildDir)/run-linter:cmd/run-linter/run-linter.go $(buildDir)/.lintSetup
	@mkdir -p $(buildDir)
	$(gobin) build -o $@ $<
# end lint setup targets


# start dependency installation tools
#   implementation details for being able to lazily install dependencies
srcFiles := makefile $(shell find . -name "*.go" -not -path "./$(buildDir)/*" -not -name "*_test.go" -not -path "*\#*")
distContents := $(buildDir)/$(name)
coverageOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage)
coverageHtmlOutput := $(foreach target,$(packages),$(buildDir)/output.$(target).coverage.html)
# end dependency installation tools


# implementation details for building the binary and creating a
# convienent link in the working directory
$(name):$(buildDir)/$(name)
	@[ -e $@ ] || ln -s $<
$(buildDir)/$(name):$(srcFiles)
	GOOS=$(if $(GOOS),$(GOOS),linux) $(goEnv) $(gobin) build -ldflags "-X github.com/evergreen-ci/barque.BuildRevision=`git rev-parse HEAD`" -o $@ cmd/$(name)/$(name).go
$(buildDir)/generate-points:cmd/generate-points/generate-points.go
	$(goEnv) $(gobin) build -o $@ $<
generate-points:$(buildDir)/generate-points
	./$<
$(buildDir)/make-tarball:cmd/make-tarball/make-tarball.go
	@mkdir -p $(buildDir)
	@$(goEnv) $(gobin) build -o $@ $<
# end dependency installation tools


# distribution targets and implementation
dist:$(buildDir)/dist.tar.gz
$(buildDir)/dist.tar.gz:$(buildDir)/make-tarball $(distContents)
	./$< --name $@ --prefix $(name) $(foreach item,$(distContents),--item $(item)) --trim $(buildDir)
	tar -tvf $@
# end deploy and distribution targets


# userfacing targets for basic build and development operations
lint:$(foreach target,$(packages),$(buildDir)/output.$(target).lint)
test:build $(buildDir)/output.test
build:$(buildDir)/$(name)
.PHONY: benchmark
coverage:build $(coverageOutput)
coverage-html:build $(coverageHtmlOutput)
list-tests:
	@echo -e "test targets:" $(foreach target,$(packages),\\n\\ttest-$(target))
phony += lint lint-deps build build-race race test coverage coverage-html list-race list-tests
.PRECIOUS:$(coverageOutput) $(coverageHtmlOutput)
.PRECIOUS:$(foreach target,$(packages),$(buildDir)/output.$(target).test)
.PRECIOUS:$(foreach target,$(packages),$(buildDir)/output.$(target).lint)
.PRECIOUS:$(buildDir)/output.lint
# end front-ends


# start vendoring configuration
#    begin with configuration of dependencies
vendor-clean:
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/certdepot/vendor/github.com/mongodb/anser/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/poplar/vendor/github.com/evergreen-ci/pail/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/poplar/vendor/github.com/mongodb/mongo-go-driver/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/poplar/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/timber/vendor/gopkg.in/yaml.v2
	rm -rf vendor/github.com/mongodb/jasper/vendor/go.mongodb.org/mongo-driver/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/urfave/cli/
	rm -rf vendor/github.com/mongodb/jasper/vendor/gopkg.in/mgo.v2/
	rm -rf vendor/github.com/mongodb/curator/vendor/
	rm -rf vendor/github.com/mongodb/curator/operations/
	rm -rf vendor/github.com/mongodb/curator/greenbay/
	rm -rf vendor/github.com/mongodb/curator/barquesubmit/
	rm -rf vendor/github.com/mongodb/curator/cmd/
	rm -rf vendor/github.com/mongodb/curator/*.{go,yaml,rst,lock}
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/davecgh/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/pmezard/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/github.com/stretchr/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/evergreen-ci/gimlet/vendor/go.mongodb.org/
	rm -rf vendor/github.com/evergreen-ci/pail/vendor/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/mongodb/amboy/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/mholt/archiver/
	rm -rf vendor/github.com/evergreen-ci/bond/vendor/github.com/stretchr/testify/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/pkg/errors/
	rm -rf vendor/github.com/mongodb/amboy/vendor/gopkg.in/yaml.v2/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/aws/aws-sdk-go
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/davecgh/go-spew
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/evergreen-ci/gimlet
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/mongodb/grip/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/mongodb/mongo-go-driver
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/stretchr/testify
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/tychoish/gimlet/
	rm -rf vendor/github.com/mongodb/amboy/vendor/github.com/urfave/cli/
	rm -rf vendor/github.com/mongodb/amboy/vendor/go.mongodb.org/mongo-driver
	rm -rf vendor/github.com/mongodb/amboy/vendor/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/amboy/vendor/gopkg.in/mgo.v2
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/mongodb/amboy
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/tychoish/tarjan
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/pkg/errors
	rm -rf vendor/github.com/mongodb/anser/vendor/github.com/stretchr
	rm -rf vendor/github.com/mongodb/anser/vendor/go.mongodb.org/mongo-driver
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/davecgh/go-spew/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/pkg/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/pmezard/go-difflib/
	rm -rf vendor/github.com/mongodb/grip/vendor/github.com/stretchr/
	rm -rf vendor/github.com/mongodb/grip/vendor/golang.org/x/net/
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/evergreen-ci/gimlet
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/amboy
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/mongodb/grip
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/pkg/errors
	rm -rf vendor/github.com/mongodb/jasper/vendor/github.com/stretchr/
	rm -rf vendor/github.com/rs/cors/examples/
	rm -rf vendor/github.com/rs/cors/wrapper
	rm -rf vendor/github.com/mholt/archiver/rar.go
	rm -rf vendor/github.com/mholt/archiver/tarbz2.go
	rm -rf vendor/github.com/mholt/archiver/tarlz4.go
	rm -rf vendor/github.com/mholt/archiver/tarsz.go
	rm -rf vendor/github.com/mholt/archiver/tarxz.go
	rm -rf vendor/go.mongodb.org/mongo-driver/data/
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/davecgh/go-spew/
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/github.com/stretchr/
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/golang.org/x/net
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/golang.org/x/text
	rm -rf vendor/go.mongodb.org/mongo-driver/vendor/gopkg.in/yaml.v2/
	find vendor/ -name "*.gif" -o -name "*.gz" -o -name "*.png" -o -name "*.ico" -o -name "*testdata*" | xargs rm -rf
phony += vendor-clean
# end vendoring tooling configuration


# convenience targets for runing tests and coverage tasks on a
# specific package.
test-%:$(buildDir)/output.%.test
	@grep -s -q -e "^PASS" $<
coverage-%:$(buildDir)/output.%.coverage
	@grep -s -q -e "^PASS" $(buildDir)/output.$*.test
html-coverage-%:$(buildDir)/output.%.coverage.html
	@grep -s -q -e "^PASS" $(buildDir)/output.$*.test
lint-%:$(buildDir)/output.%.lint
	@grep -v -s -q "^--- FAIL" $<
# end convienence targets


# start test and coverage artifacts
#    This varable includes everything that the tests actually need to
#    run. (The "build" target is intentional and makes these targetsb
#    rerun as expected.)
testTimeout := -timeout=20m
testArgs := -v $(testTimeout)
ifneq (,$(RUN_TEST))
testArgs += -run='$(RUN_TEST)'
endif
ifneq (,$(RUN_COUNT))
testArgs += -count=$(RUN_COUNT)
endif
ifneq (,$(SKIP_LONG))
testArgs += -short
endif
ifeq (,$(DISABLE_COVERAGE))
testArgs += -cover
endif
ifneq (,$(RACE_DETECTOR))
testArgs += -race
endif
# extra dependencies
# test execution and output handlers
$(buildDir)/:
	mkdir -p $@
$(buildDir)/output.%.test:$(buildDir)/ .FORCE
	$(goEnv) $(gobin) test $(testArgs) ./$(if $(subst $(name),,$*),$(subst -,/,$*),) | tee $@
	@! grep -s -q -e "^FAIL" $@ && ! grep -s -q "^WARNING: DATA RACE" $@
$(buildDir)/output.test:$(buildDir)/ .FORCE
	$(goEnv) $(gobin) test $(testArgs) ./... | tee $@
	@! grep -s -q -e "^FAIL" $@ && ! grep -s -q "^WARNING: DATA RACE" $@
$(buildDir)/output.%.coverage:$(buildDir)/ .FORCE
	$(goEnv) $(gobin) test $(testArgs) ./$(if $(subst $(name),,$*),$(subst -,/,$*),) -covermode=count -coverprofile $@ | tee $(buildDir)/output.$*.test
	-[ -f $@ ] && $(goEnv) $(gobin) tool cover -func=$@ | sed 's%$(projectPath)/%%' | column -t
$(buildDir)/output.%.coverage.html:$(buildDir)/output.%.coverage
	$(goEnv) $(gobin) tool cover -html=$< -o $@
#  targets to generate gotest output from the linter.
$(buildDir)/output.%.lint:$(buildDir)/run-linter $(buildDir)/ .FORCE
	@$(goEnv) ./$< --output=$@ --lintBin="$(buildDir)/golangci-lint" --packages='$*'
$(buildDir)/output.lint:$(buildDir)/run-linter $(buildDir)/ .FORCE
	@$(goEnv) ./$< --output=$@ --lintBin="$(buildDir)/golangci-lint" --packages='$(packages)'
#  targets to process and generate coverage reports
# end test and coverage artifacts


# mongodb utility targets
ifeq ($(OS),Windows_NT)
  decompress := 7z.exe x
else
  decompress := tar -zxvf
endif
mongodb/.get-mongodb:
	rm -rf mongodb
	mkdir -p mongodb
	cd mongodb && curl "$(MONGODB_URL)" -o mongodb.tgz && $(decompress) mongodb.tgz && chmod +x ./mongodb-*/bin/*
	cd mongodb && mv ./mongodb-*/bin/* . && rm -rf db_files && rm -rf db_logs && mkdir -p db_files && mkdir -p db_logs
get-mongodb:mongodb/.get-mongodb
	@touch $<
start-mongod:mongodb/.get-mongodb
	./mongodb/mongod --dbpath ./mongodb/db_files --port 27017 --replSet evg --smallfiles --oplogSize 10
	@echo "waiting for mongod to start up"
init-rs:mongodb/.get-mongodb
	./mongodb/mongo --eval 'rs.initiate()'
check-mongod:mongodb/.get-mongodb
	./mongodb/mongo --nodb --eval "assert.soon(function(x){try{var d = new Mongo(\"localhost:27017\"); return true}catch(e){return false}}, \"timed out connecting\")"
	@echo "mongod is up"
# end mongodb targets


# clean and other utility targets
clean:
	rm -f *.pb.go
	rm -rf $(lintDeps) $(buildDir)/coverage.* $(name) $(buildDir)/$(name)
clean-results:
	rm -rf $(buildDir)/output.*
phony += clean
# end dependency targets


# configure phony targets
.FORCE:
.PHONY:$(phony) .FORCE
