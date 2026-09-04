# syntax=docker/dockerfile:1.7
FROM node:24-alpine

ENV NODE_ENV=production
WORKDIR /app

COPY package.json package-lock.json ./
COPY migrations ./migrations
COPY src ./src

USER node
EXPOSE 8080
CMD ["node", "src/main.ts"]
