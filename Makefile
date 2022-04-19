define deploy
	sls deploy --stage local
endef

build:
	@ go fmt ./...
	@ go vet ./...
	@ env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/transaction-summary ./internal/app/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	$(call deploy)