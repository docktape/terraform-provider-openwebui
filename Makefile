.PHONY: test testacc docs

test:
	go test ./...

testacc:
	TF_ACC=1 go test ./internal/provider/... -v -count=1 -timeout 30m

docs:
	go generate ./...
