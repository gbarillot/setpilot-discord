from dataclasses import dataclass
from pathlib import Path


ENV_FILE = Path("/home/python/.env")


@dataclass(frozen=True)
class Config:
    discord_token: str
    discord_channel_ids: set[int]
    whitelist_words: set[str]
    openrouter_api_key: str
    openrouter_model: str
    postgres_host: str
    postgres_port: int
    postgres_db: str
    postgres_user: str
    postgres_password: str


def load_config(path: Path = ENV_FILE) -> Config:
    values = read_env_file(path)

    return Config(
        discord_token=require(values, "DISCORD_TOKEN", path),
        discord_channel_ids=parse_int_set(values.get("DISCORD_CHANNEL_IDS", "")),
        whitelist_words=parse_str_set(values.get("WHITE_LIST_WORDS", "")),
        openrouter_api_key=require_any(values, ("OPENROUTER_API_KEY", "LLM_TOKEN"), path),
        openrouter_model=values.get("OPENROUTER_MODEL", "openrouter/auto"),
        postgres_host=require_any(values, ("DB_HOST", "POSTGRES_HOST"), path),
        postgres_port=int(require_any(values, ("DB_PORT", "POSTGRES_PORT"), path)),
        postgres_db=require_any(values, ("DB_NAME", "POSTGRES_DB", "DATABASE_NAME"), path),
        postgres_user=require_any(values, ("POSTGRES_USER", "POSTGRES_USERNAME", "DB_USER"), path),
        postgres_password=require_any(values, ("DB_PASSWORD", "DB_password", "POSTGRES_PASSWORD"), path),
    )


def read_env_file(path: Path) -> dict[str, str]:
    if not path.exists():
        raise RuntimeError(f"Missing env file: {path}")

    values: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue

        name, value = line.split("=", 1)
        values[name.strip()] = value.strip().strip('"').strip("'")

    return values


def require(values: dict[str, str], key: str, path: Path) -> str:
    value = values.get(key)
    if not value:
        raise RuntimeError(f"{key} is required in {path}")
    return value


def require_any(values: dict[str, str], keys: tuple[str, ...], path: Path) -> str:
    for key in keys:
        value = values.get(key)
        if value:
            return value
    raise RuntimeError(f"One of {', '.join(keys)} is required in {path}")


def parse_int_set(value: str) -> set[int]:
    return {int(item.strip()) for item in value.split(",") if item.strip()}


def parse_str_set(value: str) -> set[str]:
    return {item.strip() for item in value.split(",") if item.strip()}
