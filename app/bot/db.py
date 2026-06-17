from collections import defaultdict
from typing import Any

import asyncpg

from app.bot.config import Config


SCHEMA_SQL = """
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public'
AND table_name LIKE 'setpilot_%'
ORDER BY table_name, ordinal_position
"""


class PostgresClient:
    def __init__(self, config: Config) -> None:
        self.config = config
        self.pool: asyncpg.Pool | None = None

    async def connect(self) -> None:
        self.pool = await asyncpg.create_pool(
            host=self.config.postgres_host,
            port=self.config.postgres_port,
            database=self.config.postgres_db,
            user=self.config.postgres_user,
            password=self.config.postgres_password,
            min_size=1,
            max_size=3,
        )

    async def fetch_schema_summary(self) -> str:
        if self.pool is None:
            raise RuntimeError("Postgres pool is not connected")

        async with self.pool.acquire() as connection:
            rows = await connection.fetch(SCHEMA_SQL)

        tables: dict[str, list[str]] = defaultdict(list)
        for row in rows:
            tables[row["table_name"]].append(f"{row['column_name']} {row['data_type']}")

        return "\n".join(f"{table}({', '.join(columns)})" for table, columns in tables.items())

    async def fetch_rows(self, sql: str) -> list[dict[str, Any]]:
        if self.pool is None:
            raise RuntimeError("Postgres pool is not connected")

        async with self.pool.acquire() as connection:
            async with connection.transaction():
                await connection.execute("SET TRANSACTION READ ONLY")
                rows = await connection.fetch(sql)

        return [dict(row) for row in rows]
