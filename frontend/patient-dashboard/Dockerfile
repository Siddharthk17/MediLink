FROM node:22-alpine AS base
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /app

# --- Dependencies ---
FROM base AS deps
COPY pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend/shared/package.json frontend/shared/
COPY frontend/patient-dashboard/package.json frontend/patient-dashboard/
RUN pnpm install --frozen-lockfile

# --- Builder ---
FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/frontend/shared/node_modules ./frontend/shared/node_modules
COPY --from=deps /app/frontend/patient-dashboard/node_modules ./frontend/patient-dashboard/node_modules
COPY pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend/shared/ frontend/shared/
COPY frontend/patient-dashboard/ frontend/patient-dashboard/

ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PUBLIC_API_URL=http://api:8080
ENV NEXT_PUBLIC_PHYSICIAN_PORTAL_URL=/
ENV NEXT_PUBLIC_BASE_PATH=/patient

RUN pnpm --filter @medilink/patient-dashboard build

# --- Runner ---
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3002
ENV HOSTNAME=0.0.0.0

RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs

COPY --from=builder /app/frontend/patient-dashboard/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/frontend/patient-dashboard/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/frontend/patient-dashboard/.next/static ./frontend/patient-dashboard/.next/static

USER nextjs
EXPOSE 3002

CMD ["node", "frontend/patient-dashboard/server.js"]
