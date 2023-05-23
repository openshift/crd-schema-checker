all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps.mk \
	targets/openshift/images.mk \
)

# Exclude e2e tests from unit testing
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

IMAGE_REGISTRY?=registry.svc.ci.openshift.org

# This will call a macro called "build-image" which will generate image specific targets based on the parameters:
# $0 - macro name
# $1 - target suffix
# $2 - Dockerfile path
# $3 - context directory for image build
# It will generate target "image-$(1)" for builing the image an binding it as a prerequisite to target "images".
$(call build-image,crd-schema-checker,$(IMAGE_REGISTRY)/ocp/4.2:crd-schema-checker,./Dockerfile.rhel7,.)

$(call verify-golang-versions,Dockerfile.rhel7)

clean:
	$(RM) ./crd-schema-checker
	$(RM) -rf ./_output
.PHONY: clean

