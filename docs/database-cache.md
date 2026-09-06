# Database and cache configuration

RustDesk API uses an SQL database for persistent data and a separate cache. SQLite and file cache are the defaults, so Redis, DragonflyDB, MySQL, and PostgreSQL are only required when explicitly selected.

Environment variables use the `RUSTDESK_API_` prefix. Dots and hyphens in YAML keys become underscores. The variables below are explicitly bound, work when their YAML sections are absent, and override YAML values.

## Cache

| Environment variable | Meaning | Default |
| --- | --- | --- |
| `RUSTDESK_API_CACHE_TYPE` | `file`, `memory`, or `redis` | `file` |
| `RUSTDESK_API_CACHE_FILE_DIR` | File-cache directory | `./runtime/cache` |
| `RUSTDESK_API_CACHE_REDIS_ADDR` | Redis-protocol address | `127.0.0.1:6379` |
| `RUSTDESK_API_CACHE_REDIS_PWD` | Redis password | empty |
| `RUSTDESK_API_CACHE_REDIS_DB` | Redis logical database | `0` |
| `RUSTDESK_API_CACHE_CONNECT_TIMEOUT` | Startup Redis connectivity timeout | `5s` |

DragonflyDB is Redis-protocol compatible and can use the `redis` cache type. Startup performs a bounded `PING` and exits on connection, authentication, or database-selection failure.

```yaml
environment:
  RUSTDESK_API_CACHE_TYPE: redis
  RUSTDESK_API_CACHE_REDIS_ADDR: dragonfly:6379
  RUSTDESK_API_CACHE_REDIS_PWD: ${REDIS_PASSWORD}
  RUSTDESK_API_CACHE_REDIS_DB: 1
```

## SQL databases

| Environment variable | Meaning | Default |
| --- | --- | --- |
| `RUSTDESK_API_GORM_TYPE` | `sqlite`, `mysql`, `postgresql`; `postgres` is an alias | `sqlite` |
| `RUSTDESK_API_GORM_MAX_IDLE_CONNS` | Maximum idle connections | `10` |
| `RUSTDESK_API_GORM_MAX_OPEN_CONNS` | Maximum open connections | `100` |
| `RUSTDESK_API_GORM_CONN_MAX_LIFETIME` | Maximum connection lifetime | `1h` |
| `RUSTDESK_API_GORM_CONN_MAX_IDLE_TIME` | Maximum idle connection time | `10m` |
| `RUSTDESK_API_GORM_CONNECT_TIMEOUT` | Startup connectivity timeout | `10s` |

PostgreSQL uses `RUSTDESK_API_POSTGRESQL_HOST`, `RUSTDESK_API_POSTGRESQL_PORT`, `RUSTDESK_API_POSTGRESQL_USER`, `RUSTDESK_API_POSTGRESQL_PASSWORD`, `RUSTDESK_API_POSTGRESQL_DBNAME`, `RUSTDESK_API_POSTGRESQL_SSLMODE`, and `RUSTDESK_API_POSTGRESQL_TIME_ZONE`. SSL mode accepts `disable`, `allow`, `prefer`, `require`, `verify-ca`, or `verify-full`.

```yaml
environment:
  RUSTDESK_API_GORM_TYPE: postgresql
  RUSTDESK_API_POSTGRESQL_HOST: postgres
  RUSTDESK_API_POSTGRESQL_PORT: 5432
  RUSTDESK_API_POSTGRESQL_USER: rustdesk
  RUSTDESK_API_POSTGRESQL_PASSWORD: ${POSTGRES_PASSWORD}
  RUSTDESK_API_POSTGRESQL_DBNAME: rustdesk
  RUSTDESK_API_POSTGRESQL_SSLMODE: disable
  RUSTDESK_API_POSTGRESQL_TIME_ZONE: UTC
```

For production TLS, use `require` at minimum, or `verify-ca`/`verify-full` when PostgreSQL client trust material and hostname verification are configured. PostgreSQL URLs are built with standard URL encoding, so special characters in passwords cannot alter connection fields. Logs omit passwords and full DSNs.

MySQL retains `RUSTDESK_API_MYSQL_ADDR`, `RUSTDESK_API_MYSQL_USERNAME`, `RUSTDESK_API_MYSQL_PASSWORD`, `RUSTDESK_API_MYSQL_DBNAME`, and `RUSTDESK_API_MYSQL_TLS`.

## PostgreSQL and DragonflyDB together

The SQL database and Redis logical database are independent. `RUSTDESK_API_CACHE_REDIS_DB=1` does not affect the PostgreSQL database name.

```yaml
services:
  rustdesk-api:
    image: ymg2006/rustdesk-api
    restart: unless-stopped
    environment:
      RUSTDESK_API_GORM_TYPE: postgresql
      RUSTDESK_API_POSTGRESQL_HOST: postgres
      RUSTDESK_API_POSTGRESQL_PORT: 5432
      RUSTDESK_API_POSTGRESQL_USER: rustdesk
      RUSTDESK_API_POSTGRESQL_PASSWORD: ${POSTGRES_PASSWORD}
      RUSTDESK_API_POSTGRESQL_DBNAME: rustdesk
      RUSTDESK_API_POSTGRESQL_SSLMODE: disable
      RUSTDESK_API_POSTGRESQL_TIME_ZONE: UTC
      RUSTDESK_API_CACHE_TYPE: redis
      RUSTDESK_API_CACHE_REDIS_ADDR: dragonfly:6379
      RUSTDESK_API_CACHE_REDIS_PWD: ${REDIS_PASSWORD}
      RUSTDESK_API_CACHE_REDIS_DB: 1
    depends_on:
      - postgres
      - dragonfly
    ports:
      - "21114:21114"

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: rustdesk
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: rustdesk
    volumes:
      - postgres-data:/var/lib/postgresql/data

  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly
    restart: unless-stopped
    command: ["--requirepass=${REDIS_PASSWORD}"]
    volumes:
      - dragonfly-data:/data

volumes:
  postgres-data:
  dragonfly-data:
```

Keep `POSTGRES_PASSWORD` and `REDIS_PASSWORD` in the deployment environment or a Compose `.env`/secret source; do not commit their values.
