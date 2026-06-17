from typing import Any
import re

import httpx

from app.bot.config import Config


OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"


class OpenRouterClient:
    def __init__(self, config: Config) -> None:
        self.config = config

    async def generate_sql(self, user_message: str, schema_summary: str) -> str:
        content = await self._chat(
            [
                {
                    "role": "system",
                    "content": (
                        "You convert user questions into PostgreSQL SQL queries. "
                        "Return only an SQL query, with no markdown or explanation. "
                        "The query should only target the band 'Groove Station', DO NOT Oquery about other band "
                        "The query must be read-only and start with SELECT or WITH."
                    ),
                },
                {
                    "role": "user",
                    "content": (
                        f"Available public schema:\n{schema_summary}\n\n"
                        f"User question:\n{user_message}"
                    ),
                },
            ]
        )
        return content.strip()

    async def generate_answer(self, user_message: str, sql: str, rows: list[dict[str, Any]]) -> str:
        content = await self._chat(
            [
                {
                    "role": "system",
                    "content": (
                        "You are SetPilot's Discord agent. Reply in French in a clear, concise, "
                        "and pleasant way. Format the response for Discord. All dates and times "
                        "must be expressed in Paris time, Europe/Paris, currently CEST if applicable. "
                        "Never suggest any additional action or help. Never say phrases like "
                        "'if you want', 'I can also', 'would you like', or equivalent. Limit yourself strictly "
                        "to answering the question asked. If no result is found, say so simply. "
                        "Never mention SQL, JSON, the database, the provided data, or any "
                        "technical marker like '[provided SQL result]'. Display only the final answer."
                    ),
                },
                {
                    "role": "user",
                    "content": (
                        f"User question:\n{user_message}\n\n"
                        f"Executed SQL:\n{sql}\n\n"
                        f"JSON results:\n{rows}"
                    ),
                },
            ]
        )
        return clean_answer(content)

    async def _chat(self, messages: list[dict[str, str]]) -> str:
        headers = {
            "Authorization": f"Bearer {self.config.openrouter_api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://setpilot.local",
            "X-Title": "SetPilot Discord Agent",
        }
        payload = {
            "model": self.config.openrouter_model,
            "messages": messages,
            "temperature": 0.1,
        }

        async with httpx.AsyncClient(timeout=60) as client:
            response = await client.post(OPENROUTER_URL, headers=headers, json=payload)
            response.raise_for_status()
            data = response.json()

        return data["choices"][0]["message"]["content"]


def clean_answer(content: str) -> str:
    content = re.sub(r"\[[^\]]*sql[^\]]*\]", "", content, flags=re.IGNORECASE)
    content = re.sub(r"\[[^\]]*json[^\]]*\]", "", content, flags=re.IGNORECASE)
    return content.strip()
