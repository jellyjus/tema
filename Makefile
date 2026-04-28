-include .env
export


.PHONY: run
run:
	go run ./cmd/tema/...

.PHONY: compose
compose:
	docker-compose up -d

.PHONY: goose
goose:
	goose -dir=migrations postgres "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DATABASE)" $(cmd)

buildpush:
	docker build --platform linux/amd64 -t ghcr.io/jellyjus/tema:master .
	docker push ghcr.io/jellyjus/tema:master