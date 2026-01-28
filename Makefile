test:
	go test ./... -failfast

bench:
	go test -run=^$ -bench=^.*$ -benchmem