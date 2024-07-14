default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 UNIFI_STDOUT=true go test ./... -v $(TESTARGS) -timeout 120m
