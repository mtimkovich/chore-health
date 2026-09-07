# syntax=docker/dockerfile:1

# ---- Frontend build ----
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Backend build ----
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Overwrite the empty static/ placeholder (see static/.gitkeep) with the
# real build, then embed it into the binary via //go:embed in static.go.
COPY --from=frontend-builder /app/frontend/dist ./static
RUN CGO_ENABLED=0 go build -o /chore-health .

# ---- Final image ----
FROM alpine:3.20

# tzdata lets time.Local resolve a named zone from the TZ env var, so
# "local midnight" (recurring chores reappearing after being marked done)
# matches the operator's actual timezone instead of defaulting to UTC.
#
# The chore-health user/group get a fixed UID/GID (rather than whatever
# addgroup/adduser would pick next) so it's documented and stable across
# rebuilds: a bind-mounted host directory (e.g. docker-compose's ./data)
# is owned by whatever's on the host, not by this image's chown below - that
# only applies to a Docker-managed volume or the container's own layer. To
# let this non-root user write into a host directory, chown it to match:
#   sudo chown -R 10001:10001 ./data
RUN apk add --no-cache tzdata && \
    addgroup -g 10001 -S chore-health && \
    adduser -u 10001 -S chore-health -G chore-health && \
    mkdir -p /data && chown chore-health:chore-health /data

WORKDIR /app
COPY --from=backend-builder /chore-health ./chore-health

ENV CHORE_HEALTH_DB_PATH=/data/chores.db
VOLUME ["/data"]
EXPOSE 8080

USER chore-health
ENTRYPOINT ["./chore-health"]
