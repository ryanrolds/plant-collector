.PHONY: build test deploy login

build: 
	go build -o ./bin/ ./bridge/... 

test:
	go test ./bridge/... -v -coverprofile=coverage.out

deploy: test build
	balena deploy --build PlantMetrics

login:
	balena login
