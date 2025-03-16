.PHONY: check-podman compose-up compose-down

NETWORK_NAME = testnetwork
DB_CONTAINER = test-mariadb
APP_CONTAINER = test-axfr
DB_PASSWORD = testpass123


test-deps:
	@which podman >/dev/null 2>&1 || (echo "❌ podman is required but not installed. Aborting." && exit 1)
	@which curl >/dev/null 2>&1 || (echo "❌ curl is required but not installed. Aborting." && exit 1)
	@which jq >/dev/null 2>&1 || (echo "❌ jq is required but not installed. Aborting." && exit 1)
	@which mysql >/dev/null 2>&1 || (echo "❌ mysql is required but not installed. Aborting." && exit 1)

test: test-deps
	@echo "ℹ️ Creating podman network..."
	podman network create $(NETWORK_NAME) || true

	@echo "ℹ️ Starting MariaDB container..."
	podman run -d --name $(DB_CONTAINER) \
		--network $(NETWORK_NAME) \
		-e MYSQL_ROOT_PASSWORD=$(DB_PASSWORD) \
		docker.io/library/mariadb:latest

	@echo "ℹ️ Building application container..."
	podman build -t $(APP_CONTAINER) .

	@echo "ℹ️ Waiting for MariaDB to be ready..."
	sleep 5
	
	@echo "ℹ️ Importing database dumps using podman..."
	# Wait for MariaDB to be fully initialized
	podman exec -i $(DB_CONTAINER) bash -c 'until mariadb -u root -p$(DB_PASSWORD) -e "SELECT 1"; do sleep 1; echo "Waiting for MariaDB to be ready..."; done'
	
	# Create databases
	podman exec -i $(DB_CONTAINER) mariadb -u root -p"$(DB_PASSWORD)" -e "CREATE DATABASE IF NOT EXISTS nudiff;"
	podman exec -i $(DB_CONTAINER) mariadb -u root -p"$(DB_PASSWORD)" -e "CREATE DATABASE IF NOT EXISTS nudump;"
	
	# Copy and import SQL files
	podman cp migrations/nudiff.sql $(DB_CONTAINER):/tmp/nudiff.sql
	podman cp migrations/nudump.sql $(DB_CONTAINER):/tmp/nudump.sql
	podman exec $(DB_CONTAINER) bash -c "mariadb -u root -p'$(DB_PASSWORD)' nudiff < /tmp/nudiff.sql"
	podman exec $(DB_CONTAINER) bash -c "mariadb -u root -p'$(DB_PASSWORD)' nudump < /tmp/nudump.sql"
	
	@echo "✅ Database dumps imported successfully"

	@echo "ℹ️ Starting application container..."
	podman run -d --name $(APP_CONTAINER) \
		-p 8080:8080 \
		--network $(NETWORK_NAME) \
		-e MYSQL_HOSTNAME=$(DB_CONTAINER) \
		-e MYSQL_NUDUMP_DATABASE=nudump \
		-e MYSQL_NUDUMP_USERNAME=root \
		-e MYSQL_NUDUMP_PASSWORD=$(DB_PASSWORD) \
		-e MYSQL_NU_DATABASE=nudiff \
		-e MYSQL_NU_USERNAME=root \
		-e MYSQL_NU_PASSWORD=$(DB_PASSWORD) \
		$(APP_CONTAINER)

	@echo "✅ Test environment is ready!"

	@echo "ℹ️ Running integration tests..."
	./integration_test.sh

clean:
	@echo "ℹ️ Cleaning up containers and volumes..."
	podman stop $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman rm -v $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman network rm $(NETWORK_NAME) || true

compose-up: check-podman
	podman-compose build --no-cache
	podman-compose up

compose-down: check-podman
	podman-compose down
