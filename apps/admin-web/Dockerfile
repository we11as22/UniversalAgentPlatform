FROM node:20-bookworm-slim AS builder
WORKDIR /app
COPY package.json pnpm-workspace.yaml turbo.json tsconfig.base.json ./
COPY apps/admin-web/package.json apps/admin-web/package.json
COPY packages/config/package.json packages/config/package.json
COPY packages/ts-sdk/package.json packages/ts-sdk/package.json
COPY packages/ui/package.json packages/ui/package.json
RUN corepack enable && pnpm install
COPY . .
RUN pnpm --filter @uap/admin-web build

FROM node:20-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/apps/admin-web/.next/standalone ./
COPY --from=builder /app/apps/admin-web/.next/static ./apps/admin-web/.next/static
COPY --from=builder /app/apps/admin-web/public ./apps/admin-web/public
EXPOSE 3300
CMD ["node", "apps/admin-web/server.js"]

