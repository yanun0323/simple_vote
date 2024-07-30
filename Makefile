.PHONY: run

run:
	go run ./main.go

.PHONY: build docker.run docker.up docker.down docker.rm

build:	## Build backend Docker image
	docker build . \
		-t vote \
		--no-cache \

docker.run:
	docker run -d \
	-p 8080:8080 \
	--name vote vote

docker.up:
	docker container start vote

docker.down:
	docker container stop vote

docker.rm:
	docker rm -f vote