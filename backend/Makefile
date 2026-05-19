GOOSE = goose -dir migrations postgres "$(DATABASE_URL)"

.PHONY: migrate-up migrate-down migrate-status migrate-create

migrate-up:
	$(GOOSE) up

migrate-down:
	$(GOOSE) down

migrate-status:
	$(GOOSE) status

migrate-create:
	goose -dir migrations create $(NAME) sql
