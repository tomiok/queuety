.PHONY: build run stop clean logs

# Variables
IMAGE_NAME=queuety
CONTAINER_NAME=queuety
VERSION=latest

build:
	@docker build -t $(IMAGE_NAME):$(VERSION) .

run-queuety:
	 @docker run -d \
	  --name $(CONTAINER_NAME) \
	  $(IMAGE_NAME):$(VERSION)

stop:
	docker compose down

logs:
	docker compose logs -f queuety

clean:
	docker compose down -v
	docker rmi $(IMAGE_NAME):$(VERSION) || true
	docker system prune -f

restart: stop build run

shell:
	docker exec -it $(CONTAINER_NAME) /bin/sh