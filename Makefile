test-golang:
	go test ./...

test-e2e:
	docker-compose -f test/docker-compose.yml build passage
	docker-compose -f test/docker-compose.yml up -d
	./test/test.sh
	docker-compose -f test/docker-compose.yml down