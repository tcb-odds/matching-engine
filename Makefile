BUILD=./build/app
#GOOS?=linux
DOCKER_TAG?=matching-engine

build: vendor clean
	CGO_ENABLED=0 GOOS=${GOOS} go build -a -installsuffix cgo \
		-o ${BUILD} ./cmd/main.go

clean:
	@[ -f ${BUILD} ] && rm -f ${BUILD} || true

docker-build:
	docker build --build-arg ssh_prv_key="$(cat ~/.ssh/id_ed25519)" --build-arg ssh_pub_key="$(cat ~/.ssh/id_ed25519.pub)"  . -t ${DOCKER_TAG}

network:
	sudo docker network create tcb-network

start:
	sudo docker-compose up -d

start-win:
	./build/app

dev-api:
	go run ./cmd/main.go development

stop:
	sudo docker-compose down

rebuild: build
	sudo docker-compose up -d --force-recreate --build

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

lint:
	golangci-lint run

format:
	go fmt ./...

coverage:
	go test -cover -coverprofile=coverage.out ./... -p 1 && go tool cover -html=coverage.out -o coverage.html

gen-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/proto/matching-engine.proto
