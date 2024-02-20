test:
	go test ./... -count=1 -parallel=2 -v -coverprofile cover.out \
		&& go tool cover -func=cover.out 

lint:
	golangci-lint run

