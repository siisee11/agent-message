# syntax=docker/dockerfile:1

FROM node:20-alpine AS build
WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/. ./
RUN npm run build

FROM node:20-alpine
WORKDIR /app

COPY deploy/agent_gateway.mjs /app/deploy/agent_gateway.mjs
COPY --from=build /app/web/dist /app/web/dist

ENV AGENT_GATEWAY_HOST=0.0.0.0
ENV AGENT_GATEWAY_PORT=8788
ENV AGENT_API_ORIGIN=http://server:8080
ENV AGENT_WEB_DIST=/app/web/dist

EXPOSE 8788

CMD ["node", "/app/deploy/agent_gateway.mjs"]
