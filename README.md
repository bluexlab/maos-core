# Multi-Agent Operating System Core (maos-core)

## Introduction

MAOS (Multi-Agent Operating System) is an innovative project designed to provide robust infrastructure for AI agents. It focuses on:

1. Managing shared infrastructure and resources
2. Regulating agent execution models

The core component of MAOS serves as the foundation for building complex multi-agent systems.

## Features

- Resource allocation and management
- Agent execution regulation
- Scalable infrastructure for AI agents
- [more key features as they are developed]

## Getting Started

### Prerequisites

Go 1.22 or later is required

## Installation

### Kubernates Bootstraping

After the maos-core service is up and running, you need to create an agent and token for the Admin UI:

1. Set up Kubernetes port forwarding:

```
kubectl -n devbxpay port-forward service/devbxpay-portals-maos-core 5001:5000
```

2. Create an agent via the API:

```
curl -X POST --location 'http://127.0.0.1:5001/v1/admin/agents' \
  -H 'Authorization: Bearer bootstrap-token' \
  -H 'Content-Type: application/json' \
  --data '{"name": "admin-portal"}'
```

Note: Save the id from the response for use in step 3.

3. Create a token for the Admin UI:

```
curl -X POST --location 'http://127.0.0.1:5001/v1/admin/api_tokens' \
  -H 'Authorization: Bearer bootstrap-token' \
  -H 'Content-Type: application/json' \
  --data '{"agent_id": <id_from_step_2>, "expire_at":2037780304, "created_by":"kevin@bluextrade.com", "permissions":["admin"]}'
```

Note: Replace <id_from_step_2> with the actual agent ID returned in step 2.

4. Configure the Admin UI:
   After the token is created, the bootstrap token will become invalid. Assign the newly created token to the Admin UI configuration.

### Create invocation manaully

make sure you have a token with create:invocation permission:

```
curl -v -X POST --location 'http://127.0.0.1:5001/v1/invocations/sync' \
  -H 'Authorization: Bearer {token}' \
  -H 'Content-Type: application/json' \
  --data '{"agent":"<agent-name>","meta":{"kind": "<method name>"},"payload":{.....}}'
```

## Development

### Setting Up the Development Environment

Install dependencies:

```shell
go mod download
```

### Running Tests

To execute tests, follow these steps:

1. Create testing databases:

```shell
createdb maos-test
```

This command creates a testing databases under system account

2. Set up the database connection:

If you created the test database with default settings, you can skip this step.

If you used a specific username or different database name, you need to set the TEST_DATABASE_URL environment variable:
Option A: Set it directly in your shell:

```shell
export TEST_DATABASE_URL=postgres://username:password@localhost/maos-test?sslmode=disable
```

Option B: Add it to a .env file in the project root:

```
TEST_DATABASE_URL=postgres://username:password@localhost/maos-test?sslmode=disable
```

Replace username, password, and maos-test with your specific database credentials and name.

3. Run the tests using the following methods:

```shell
go test ./...
```

To run tests without using the cache:

```shell
go test -count=1 ./...
```

For verbose output:

```shell
go test -v ./...
```
