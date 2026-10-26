# Deploy the Pageguard backend on Render

Render runs the Go API as a Docker web service, with a managed PostgreSQL
database and the Playwright browser worker included in the image. The Next.js
dashboard can then be hosted on [Netlify](netlify-deploy.md) or on Render.

This guide uses the repository's `Dockerfile` and `render.yaml` blueprint.

## Prerequisites

- A GitHub, GitLab, or Bitbucket account containing this repository.
- A [Render account](https://render.com).
- The repository pushed to the provider (Render deploys from Git, not from your
  local machine).
- Optional: the Netlify site URL from [netlify-deploy.md](netlify-deploy.md),
  used for CORS.

## Option A: Blueprint deployment (recommended)

`render.yaml` creates the database and the web service in one step.

1. In the Render Dashboard, click **New → Blueprint**.
2. Connect your Git provider and select this repository.
3. Give the Blueprint a name, choose the branch to deploy (for example `main`),
   and click **Connect**.
4. Render reads `render.yaml` and lists:
   - `pageguard-db` — PostgreSQL database
   - `pageguard-api` — Docker web service
5. For `PAGEGUARD_CORS_ORIGINS`, enter your Netlify origin, for example
   `https://pageguard.netlify.app`. If you do not have it yet, leave it blank
   and set it later (step 5 of Option B).
6. Click **Apply**. Render provisions PostgreSQL first, then builds the Docker
   image. The first build installs Node, Playwright, and Chromium, so it can take
   5–15 minutes.
7. Wait for the service status to become **Live** and the `/healthz` check to
   pass.
8. Open the service and copy its URL, for example
   `https://pageguard-api.onrender.com`. This is your API base.

## Option B: Manual deployment

Use this if you prefer to create resources one at a time or already have a
database.

### 1. Create PostgreSQL

1. Dashboard → **New → Postgres**.
2. Set **Name** to `pageguard-db`, **Database** to `pageguard`, and **User** to
   `pageguard`.
3. Pick the same **Region** you will use for the web service.
4. Choose a plan and click **Create Database**.
5. When it is ready, copy the **Internal Database URL**. Use the internal URL
   because the web service runs in the same Render account and region. If you
   must connect from outside Render, copy the **External Database URL** and add
   `?sslmode=require` if it is not already present.

### 2. Create the web service

1. Dashboard → **New → Web Service**.
2. Connect the repository containing this project.
3. Configure the service:

   | Setting | Value |
   | --- | --- |
   | Name | `pageguard-api` |
   | Region | Same region as the database |
   | Language / Runtime | `Docker` |
   | Dockerfile path | `./Dockerfile` |
   | Docker build context | `.` (repository root) |
   | Health check path | `/healthz` |
   | Auto-Deploy | `Yes` (deploys on every push) |

4. Choose a plan. Free is fine for HTTP-only scans and demos. For browser
   rendering with Chromium, use at least 2 GB of memory.
5. Click **Create Web Service** (or **Deploy**).

### 3. Add environment variables

Add these under **Environment → Environment Variables** before the first
successful boot:

| Variable | Value |
| --- | --- |
| `PAGEGUARD_ADDR` | `:10000` |
| `PAGEGUARD_PLUGINS_DIR` | `/app/plugins` |
| `PAGEGUARD_MIGRATIONS_DIR` | `/app/migrations` |
| `PAGEGUARD_BROWSER_CMD` | `node /app/tools/browser-renderer/index.mjs` |
| `DATABASE_URL` | Internal Database URL from step 1 |
| `PAGEGUARD_CORS_ORIGINS` | `https://<your-site>.netlify.app` |

Optional:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PAGEGUARD_LOG` | `info` | `debug`, `info`, `warn`, or `error` |
| `PAGEGUARD_MIGRATE` | `true` | Apply migrations on startup |
| `PAGEGUARD_PROXIES` | empty | Rotating proxy list for scans |
| `PAGEGUARD_CAPTURE_HAR` | `false` | Capture HAR archives |
| `PAGEGUARD_CAPTURE_DATALAYER` | `false` | Capture data-layer payloads |

Click **Save Changes**. Render restarts the service. Migrations run on boot when
`DATABASE_URL` is set.

### 4. Verify the API

Replace the host with your Render URL:

```bash
curl -i https://pageguard-api.onrender.com/healthz
curl -s https://pageguard-api.onrender.com/api/v1/projects

curl -s https://pageguard-api.onrender.com/api/v1/projects \
  -H 'content-type: application/json' \
  -d '{"name":"Acme"}'
```

A `200` from `/healthz`, an empty `[]` project list, and a created project mean
the API and database are working.

### 5. Connect the Netlify dashboard

1. In Netlify, set `NEXT_PUBLIC_API_BASE` to
   `https://pageguard-api.onrender.com`.
2. Redeploy the Netlify site so the value is baked into the Next.js build.
3. On Render, set `PAGEGUARD_CORS_ORIGINS` to the exact Netlify origin, with no
   trailing slash:

   ```bash
   PAGEGUARD_CORS_ORIGINS=https://<your-site>.netlify.app
   ```

4. Save the variable and wait for the service to restart.
5. Open the Netlify site, create a scan, and start a run.

## Deploys and updates

With **Auto-Deploy** enabled, pushing to the connected branch rebuilds and
redeploys the service. To deploy manually, use **Manual Deploy → Deploy latest
commit** in the Render Dashboard. Database migrations run automatically on each
boot.

The dashboard does not need to be redeployed when only the backend changes,
unless `NEXT_PUBLIC_API_BASE` changes.

## Browser rendering on Render

The Docker image installs Chromium through Playwright. It is large and
memory-hungry:

- Browser-rendered scans need roughly 1–2 GB of RAM. The free plan may be too
  small; use a paid instance if scans fail with browser or memory errors.
- If you only need HTTP scanning, remove `PAGEGUARD_BROWSER_CMD` and create scans
  with `"renderMode": "http"`. You can also comment out the Playwright install in
  the `Dockerfile` to speed up builds.
- Free web services spin down after inactivity. The first request after a spin
  down is slow, and long scans can be interrupted by restarts. Use a paid
  instance for reliable scheduled or long-running audits.

## Custom domain

1. In Render, open the service → **Settings → Custom Domains** and add your
   domain.
2. Create the DNS records Render shows.
3. Update `NEXT_PUBLIC_API_BASE` in Netlify and redeploy.
4. Add the new origin to `PAGEGUARD_CORS_ORIGINS` and restart the API.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| Build fails with `go.mod requires go >= 1.25.0` | Keep the `golang` image tag in `Dockerfile` aligned with `go.mod` (currently `golang:1.25-alpine`). |
| Deploy stuck at "Port scan timeout" | Ensure `PAGEGUARD_ADDR=:10000`. The API must listen on Render's port, and it binds to `PAGEGUARD_ADDR`, not `$PORT`. |
| `/healthz` returns `502` | Check the service logs for migration or database errors. Confirm `DATABASE_URL` is the internal URL and the database is in the same region. |
| Database SSL or connection errors | Use the internal connection string from the same region. For external connections, append `?sslmode=require`. |
| Dashboard shows CORS errors | `PAGEGUARD_CORS_ORIGINS` must exactly match the browser origin (scheme and host, no trailing slash). Restart after changing it. |
| Browser scans fail with out-of-memory or timeouts | Increase the instance memory or switch scans to `"renderMode": "http"`. |
| WebSocket/SSE progress does not update | The browser connects directly to the Render API URL, not through Netlify. Confirm `NEXT_PUBLIC_API_BASE` points to Render and that the Netlify site was rebuilt. |
| First request after idle is slow | Free services spin down; upgrade or keep the service warm. |
