FROM caddy:2.6-builder-alpine AS builder

RUN xcaddy build \
    --with github.com/simongregorebner/caddy-gitea@v1.0.4

FROM caddy:2.6.2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
