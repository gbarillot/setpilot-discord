FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bot .

FROM alpine:3.22

ARG USERNAME=bot
ARG USER_UID=1001
ARG USER_GID=$USER_UID

RUN addgroup -g $USER_GID $USERNAME \
    && adduser -D -u $USER_UID -G $USERNAME $USERNAME \
    && mkdir -p /home/bot/logs /home/bot/data \
    && chown -R $USERNAME:$USERNAME /home/bot

COPY --from=build /bot /home/bot/bot

USER $USERNAME
WORKDIR /home/bot

CMD ["/home/bot/bot"]
