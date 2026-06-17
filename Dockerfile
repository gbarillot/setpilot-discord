FROM python:3.12-slim

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /usr/local/bin/

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_LINK_MODE=copy

ARG USERNAME=python
ARG USER_UID=1001
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

WORKDIR /home/python

COPY pyproject.toml uv.lock ./
COPY app ./app

RUN uv sync --frozen --no-dev \
    && mkdir -p /home/python/logs \
    && chown -R $USERNAME:$USERNAME /home/python

USER $USERNAME

CMD ["uv", "run", "bot"]
