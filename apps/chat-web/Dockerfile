FROM node:20-bookworm-slim AS builder
WORKDIR /app
COPY package.json pnpm-workspace.yaml turbo.json tsconfig.base.json ./
COPY apps/chat-web/package.json apps/chat-web/package.json
COPY packages/config/package.json packages/config/package.json
COPY packages/ts-sdk/package.json packages/ts-sdk/package.json
COPY packages/ui/package.json packages/ui/package.json
RUN corepack enable && pnpm install
COPY . .
RUN pnpm --filter @uap/chat-web build

FROM node:20-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/apps/chat-web/.next/standalone ./
COPY --from=builder /app/apps/chat-web/.next/static ./apps/chat-web/.next/static
COPY --from=builder /app/apps/chat-web/public ./apps/chat-web/public
EXPOSE 3200
CMD ["node", "apps/chat-web/server.js"]

