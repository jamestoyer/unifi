default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	# Workaround for https://github.com/testcontainers/testcontainers-go/issues/2621
	TF_ACC=1 TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT=5m go test ./... -v $(TESTARGS) -timeout 120m
