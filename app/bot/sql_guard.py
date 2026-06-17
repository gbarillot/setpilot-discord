import re


FORBIDDEN_KEYWORDS = {
    "alter",
    "call",
    "copy",
    "create",
    "delete",
    "do",
    "drop",
    "grant",
    "insert",
    "revoke",
    "truncate",
    "update",
}


def validate_select_query(sql: str) -> str:
    sql = strip_code_fence(sql).strip()
    sql = sql[:-1].strip() if sql.endswith(";") else sql

    if ";" in sql:
        raise ValueError("Multiple SQL statements are not allowed")

    lowered = sql.casefold()
    if not (lowered.startswith("select ") or lowered.startswith("with ")):
        raise ValueError("Only SELECT queries are allowed")

    tokens = set(re.findall(r"\b[a-z_]+\b", lowered))
    forbidden = tokens & FORBIDDEN_KEYWORDS
    if forbidden:
        raise ValueError(f"Forbidden SQL keyword: {sorted(forbidden)[0]}")

    if not re.search(r"\blimit\s+\d+\b", lowered):
        sql = f"{sql}\nLIMIT 50"

    return sql


def strip_code_fence(text: str) -> str:
    text = text.strip()
    if text.startswith("```"):
        lines = text.splitlines()
        if lines and lines[0].startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        return "\n".join(lines)
    return text
