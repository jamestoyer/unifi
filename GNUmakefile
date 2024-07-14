default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT=5m go test ./... -v $(TESTARGS) -timeout 120m
