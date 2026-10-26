# Deploy Pageguard on Netlify

Netlify hosts the **Next.js dashboard** (`web/`). It cannot host the Go backend:
the API is a long-running server that streams SSE/WebSocket progress, launches
Playwright browser workers, and needs PostgreSQL. Run the backend on a container
host (Render, Fly.io, Railway, or a VPS), then point the Netlify dashboard at it.

```
Browser ──> https://<site>.netlify.app   (Next.js dashboard, Netlify)
                    │
                    │ NEXT_PUBLIC_API_BASE
                    ▼
            https://<api-host>           (Go API + PostgreSQL + Playwright)
```

## 1. Deploy the backend

The repository `Dockerfile` already bundles the Go binary, Node, the Playwright
renderer, plugins, and migrations.

### Render (Docker)

See [render-deploy.md](render-deploy.md) for the full step-by-step guide,
including the `render.yaml` blueprint. Short version:

1. Create a **PostgreSQL** instance and copy its internal connection string.
2. **New → Web Service → Build and deploy from a Git repository**, pick this repo.
3. Set **Runtime** to `Docker`, **Dockerfile path** to `./Dockerfile`.
4. Add environment variables:
   - `PAGEGUARD_ADDR=:10000`
   - `PAGEGUARD_PLUGINS_DIR=/app/plugins`
   - `PAGEGUARD_MIGRATIONS_DIR=/app/migrations`
   - `PAGEGUARD_BROWSER_CMD=node /app/tools/browser-renderer/index.mjs`
   - `DATABASE_URL=<Postgres internal URL>`
   - `PAGEGUARD_CORS_ORIGINS=https://<your-site>.netlify.app`
5. Deploy and confirm `https://<api-host>/api/v1/projects` responds.

### Fly.io

```bash
fly launch --dockerfile Dockerfile --no-deploy
fly postgres create
fly postgres attach <postgres-app-name>
fly secrets set \
  PAGEGUARD_ADDR=:8080 \
  PAGEGUARD_PLUGINS_DIR=/app/plugins \
  PAGEGUARD_MIGRATIONS_DIR=/app/migrations \
  PAGEGUARD_BROWSER_CMD="node /app/tools/browser-renderer/index.mjs" \
  PAGEGUARD_CORS_ORIGINS="https://<your-site>.netlify.app"
fly deploy
```

Expose the app on port `8080` (`[http_service] internal_port = 8080`) and use
`https://<app>.fly.dev` as the API base.

### VPS / Docker Compose

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

Put the API behind HTTPS (Caddy, nginx, or a tunnel) before using it from
Netlify, otherwise browsers block mixed content.

> Browser rendering needs Chromium. The provided image includes it; if you run a
> slim image, either install Chromium or unset `PAGEGUARD_BROWSER_CMD` and use
> `"renderMode": "http"` for scans.

## 2. Deploy the dashboard to Netlify

`netlify.toml` at the repository root already configures the build:

```toml
[build]
  base = "web"
  command = "npm run build"
  publish = ".next"

[[plugins]]
  package = "@netlify/plugin-nextjs"
```

### Netlify UI

1. **Add new site → Import an existing project**, connect GitHub, and select the
   repository. Netlify reads `netlify.toml`; leave the detected settings as-is.
2. Before the first deploy, add an environment variable:
   - `NEXT_PUBLIC_API_BASE=https://<api-host>`
3. Click **Deploy site**.

`NEXT_PUBLIC_*` values are inlined during `next build`, so changing the API base
requires a redeploy (**Deploys → Trigger deploy → Clear cache and deploy site**).

### Netlify CLI

```bash
npm install -g netlify-cli
netlify login
netlify init
netlify env:set NEXT_PUBLIC_API_BASE "https://<api-host>"
netlify deploy --build --prod
```

## 3. Allow the Netlify origin on the API

Set the backend's CORS allow-list to the Netlify URL (comma-separate multiple
origins, or use `*` for a public demo) and restart the backend:

```bash
PAGEGUARD_CORS_ORIGINS=https://<your-site>.netlify.app
```

Then open the site and start a scan. If the dashboard shows network errors, check
the browser console for a CORS or mixed-content message and confirm
`NEXT_PUBLIC_API_BASE` was set before the build.

## 4. Custom domain (optional)

In Netlify, **Domain management → Add a domain** and follow the DNS
instructions. Add the final origin to `PAGEGUARD_CORS_ORIGINS` as well.

## Static HTML mockup

`opentagvalidator_dashboard.html` is a self-contained demo, not the real
dashboard. To publish it instead, drop the file into Netlify Drop or publish it as
`index.html` from a folder containing only that file.
