import asyncio
import json
import os
import re
from datetime import UTC, datetime
from pathlib import Path

import discord

from app.bot.config import Config, load_config
from app.bot.db import PostgresClient
from app.bot.llm import OpenRouterClient
from app.bot.sql_guard import validate_select_query


OUTPUT_FILE = Path(os.getenv("BOT_OUTPUT_FILE", "/home/python/logs/discord_messages.jsonl"))
REFUSAL_MESSAGE = "Désolé mais je ne peux répondre qu'aux questions concernant les concerts et les playlists de Groove Station, le meilleur groupe de Funk du monde."
ERROR_MESSAGE = "Désolé, je n'arrive pas à récupérer l'information pour le moment. Réessaie un peu plus tard."

class WriteDownBot(discord.Client):
    def __init__(self, config: Config, db: PostgresClient, llm: OpenRouterClient, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self.config = config
        self.db = db
        self.llm = llm
        self.schema_summary = ""

    async def setup_hook(self) -> None:
        await self.db.connect()
        self.schema_summary = await self.db.fetch_schema_summary()

    async def on_ready(self) -> None:
        print(f"Logged in as {self.user} ({self.user.id if self.user else 'unknown'})")

    async def on_message(self, message: discord.Message) -> None:
        if message.author == self.user:
            return

        if self.config.discord_channel_ids and message.channel.id not in self.config.discord_channel_ids:
            return

        if self.user not in message.mentions:
            return

        if not is_whitelisted_message(message.content, self.config.whitelist_words):
            await message.reply(REFUSAL_MESSAGE)
            return

        status_message = await message.reply("Je cherche...")

        payload = {
            "received_at": datetime.now(UTC).isoformat(),
            "guild_id": message.guild.id if message.guild else None,
            "channel_id": message.channel.id,
            "author_id": message.author.id,
            "author": str(message.author),
            "content": message.content,
        }

        try:
            await asyncio.to_thread(write_payload, payload)
            sql = await self.llm.generate_sql(message.content, self.schema_summary)
            sql = validate_select_query(sql)
            rows = await self.db.fetch_rows(sql)
            answer = await self.llm.generate_answer(message.content, sql, rows)
        except Exception as error:
            print(f"Failed to process message {message.id}: {error}")
            await reply_or_edit(message, status_message, ERROR_MESSAGE)
            return

        await reply_or_edit(message, status_message, answer[:1900])


def write_payload(payload: dict) -> None:
    OUTPUT_FILE.parent.mkdir(parents=True, exist_ok=True)
    with OUTPUT_FILE.open("a", encoding="utf-8") as file:
        file.write(json.dumps(payload, ensure_ascii=False) + "\n")


async def reply_or_edit(original_message: discord.Message, status_message: discord.Message, content: str) -> None:
    try:
        await status_message.edit(content=content)
    except Exception as error:
        print(f"Failed to edit status message {status_message.id}: {error}")
        await original_message.reply(content)


def main() -> None:
    config = load_config()

    intents = discord.Intents.default()
    intents.message_content = True

    db = PostgresClient(config)
    llm = OpenRouterClient(config)
    client = WriteDownBot(config=config, db=db, llm=llm, intents=intents)
    client.run(config.discord_token)


def is_whitelisted_message(content: str, whitelist_words: set[str]) -> bool:
    allowed_words = {variant for word in whitelist_words for variant in word_variants(word)}
    message_words = {variant for word in re.findall(r"\w+", content, flags=re.UNICODE) for variant in word_variants(word)}
    return bool(allowed_words & message_words)


def word_variants(word: str) -> set[str]:
    word = word.casefold().strip()
    variants = {word}
    if len(word) > 3 and word.endswith("s"):
        variants.add(word[:-1])
    if len(word) > 3 and word.endswith("x"):
        variants.add(word[:-1])
    if len(word) > 3 and word.endswith("es"):
        variants.add(word[:-2])
    return variants


if __name__ == "__main__":
    main()
