FROM node:22-alpine AS frontend-build

WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend ./
ARG VITE_DEMO_MODE=false
ENV VITE_DEMO_MODE=$VITE_DEMO_MODE
RUN pnpm build

FROM golang:1.22 AS build

WORKDIR /src
COPY go.mod ./
COPY backend ./backend
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sharex-uploader ./backend
RUN mkdir -p /out/data/uploads

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=build /out/sharex-uploader ./sharex-uploader
COPY --from=frontend-build /src/frontend/dist ./frontend/dist
COPY --chown=65532:65532 --from=build /out/data ./data

ENV FRONTEND_DIR=/app/frontend/dist
ENV DATA_DIR=/app/data
ENV PORT=8080
EXPOSE 8080
VOLUME ["/app/data"]

ENTRYPOINT ["/app/sharex-uploader"]
USER nonroot:nonroot

