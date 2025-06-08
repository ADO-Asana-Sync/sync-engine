# Deploy notes

The supplied docker compose file will start a local instance of the application. The application will be available at `http://localhost:<SERVER_PORT>`. For example http://localhost:8456.

In order to update the port and configure the environment, make a copy of the .env.example file and populate the variables:

```bash
cp .env.example .env
```

## Bring the service online

Using docker compose, you may bring the services online by running one of the following commands:

```bash
# Run normally:
docker compose up --build --remove-orphans

# Development with watch:
docker compose up --watch --build --remove-orphans

# Development for local debug:
docker compose up --build --remove-orphans --scale web-ui=0 --scale sync=0
```

The Mongo Admin UI may be found on the port http://localhost:8581. Authenticate with the credentials found in the .env file.
