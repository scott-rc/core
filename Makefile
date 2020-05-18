lint:
	@echo 'linting...'
	@golangci-lint run
	@echo 'done'

fmt:
	@echo 'formatting...'
	@go fmt ./...
	@echo 'done'
