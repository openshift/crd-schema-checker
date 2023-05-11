all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps.mk \
)

# Exclude e2e tests from unit testing
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

$(call verify-golang-versions,Dockerfile.rhel7)

clean:
	$(RM) ./crd-schema-checker
	$(RM) -rf ./_output
.PHONY: clean

