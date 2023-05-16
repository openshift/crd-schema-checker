# crd-schema-checker
Tools to check CRD schemas for compatibility and best practices

`crd-schema-checker check-manifests [--existing-crd-filename=] --new-crd-filename=`


```bash
[deads@fedora crd-schema-checker]$ make && ./crd-schema-checker check-manifests --existing-crd-filename=pkg/manifestcomparators/testdata/no_bools/bool-already-existed-and-another/existing.yaml --new-crd-filename=pkg/manifestcomparators/testdata/no_bools/bool-already-existed-and-another/new.yaml
go build -mod=vendor -trimpath -ldflags "-X github.com/openshift/crd-schema-checker/pkg/version.versionFromGit="v0.0.0-unknown-6df7258" -X github.com/openshift/crd-schema-checker/pkg/version.commitFromGit="6df7258" -X github.com/openshift/crd-schema-checker/pkg/version.gitTreeState="dirty" -X github.com/openshift/crd-schema-checker/pkg/version.buildDate="2023-05-16T21:12:30Z" " github.com/openshift/crd-schema-checker/cmd/crd-schema-checker
ERROR: "NoBools": crd/schedulers.config.openshift.io version/v1 field/^.spec.newIllegalField may not be a boolean
```

## Goals

1. Create a CLI command to compare an old and new CRD manifest for violations
2. Create a validating admission webhook to check CRD updates for violations
3. Easy to extend the ruleset, potentially by vendoring the tool and inserting into the command.
4. Easy to select a ruleset, both defaults (if vendored out) and via the CLI.
5. Easy to unit and CLI test.
6. Probable goal: runtime check against all existing instances to locate failures


### What is a violation?
This area is still evolving and likely has multiple levels of opinion about violations
1. Changes that break deserializations.  We may still want to allow these given an override. 
   For instance, in early experimentation there may not be any existent clients.
2. Changes that break certain clients.
   For instance, changes that tighten or loosen a regex.
3. Changes that violate best practices.
   For instance, don't allow bools.
4. Changes that break round-tripping.
   For instance, removing a field.
5. Changes that violate guidance.
   For instance, don't use maps.  Usually.

### Selecting rules
There must be a mechanism for selecting which rules to apply and whether they are fatal or informative.

### Ignoring rules
There must be a way to identify a rule,field,value tuple that is an allowed violation.
It must be trackable to the person who allowed that violation.
Some of these will be unnecessary beyond a certain point (once a field was removed, there's no need to keep the exception).

