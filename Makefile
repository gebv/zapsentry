test:
	go test -v -timeout 5s -race -bench=. -run=. -coverprofile=coverage.txt -covermode=atomic
