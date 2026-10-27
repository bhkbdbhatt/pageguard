# Pageguard marketing site

A dependency-free static promotional site for **Pageguard**, kept in its own
folder so it does not interfere with the Next.js dashboard in `web/` or the Go
service.

## Goals

- Explain what Pageguard does in the first screen: crawl sites, execute tags in
  a real browser, and prove what fired, what was stored, and what leaked.
- Drive two actions: **self-host quick start** and **view the repository**.
- Reinforce the differentiators: open source (MIT), evidence-first findings,
  self-hosted by default, and privacy-safe cookie handling.
- Send technical readers to the architecture, tag policy, journey, privacy, and
  plugin documentation.

## Site plan

| Section | Purpose | Content |
| --- | --- | --- |
| Header | Persistent navigation and primary CTA | Product, how it works, capabilities, self-host, FAQ, GitHub |
| Hero | Instant positioning | Headline, subhead, CTAs, trust chips, illustrative dashboard preview |
| Vendor band | Recognition | Martech names Pageguard audits |
| Problem | Create urgency | Tag drift, consent leaks, evidence gaps |
| How it works | Explain the engine | Crawl, render, validate, prove |
| Capabilities | Show breadth | Scanning, tag policies, journeys, privacy, plugin SDK, streaming API |
| Evidence | Build trust | Traceable findings, hashed cookies, resumable runs, exports, code examples |
| Self-host | Remove friction | Quick start, Docker Compose, requirements, license badges |
| Stats | Reinforce value | Regulations, render modes, self-hosting, MIT |
| FAQ | Handle objections | Licensing, CMP, cookie storage, SPAs, journeys, cost |
| Final CTA | Convert | Self-host and repository actions |
| Footer | Navigation and context | Product, resources, project, rename note |

## Structure

```
marketing/
  index.html          All page copy and markup
  assets/styles.css   Design tokens, layout, components, responsive rules
  assets/main.js      Mobile nav, copy buttons, tabs, scroll reveal, year
  assets/favicon.svg  Brand mark
  netlify.toml        Static deploy and security headers
  robots.txt          Crawler policy
  README.md           This plan
```

There is no build step, package manager, framework, or external network request.
Fonts fall back to the system stack.

## Run locally

Open `index.html` directly, or serve the folder so clipboard permissions behave
like production:

```powershell
npx serve marketing
```

```powershell
python -m http.server 4173 --directory marketing
```

Then visit `http://localhost:4173`.

## Editing

- **Copy and links:** `index.html`. Repository links currently point at
  `github.com/bhkbdbhatt/openobserve`; update them if the project moves.
- **Design:** custom properties at the top of `assets/styles.css` control color,
  spacing, and type. The palette mirrors the dashboard: slate surfaces, indigo
  primary, emerald accent.
- **Behavior:** `assets/main.js`. Interactive pieces degrade gracefully: tabs
  fall back with native `hidden`, FAQ uses `<details>`, and scroll reveal is
  disabled for reduced-motion users and no-JavaScript visitors.
- **Keep claims factual.** The dashboard preview and report JSON are labeled
  illustrative. Do not add invented customer logos, audit counts, or
  testimonials.

## Deploy

### Netlify

Create a new site from this repository and set:

- **Base directory:** `marketing`
- **Build command:** leave empty
- **Publish directory:** `.`

The `marketing/netlify.toml` supplies security and cache headers. The root
`netlify.toml` belongs to the dashboard deployment and is not used here.

Or deploy from the command line:

```powershell
netlify deploy --dir marketing --prod
```

### Other hosts

Any static host works: GitHub Pages, Cloudflare Pages, S3, nginx, or the Go
binary’s static file server. Asset paths are relative, so the folder can be
mounted at a subpath without changes.

## Accessibility and performance

- Semantic landmarks, a skip link, visible focus states, and an accessible tab
  interface with arrow-key navigation.
- Responsive down to small phones; no horizontal scrolling in code blocks.
- Respects `prefers-reduced-motion`.
- Zero external requests, no web fonts, no analytics, and no JavaScript
  required to read the page.
- Decorative dashboard content is hidden from assistive tech with a single
  descriptive label on the container.

## Next steps

1. Replace the illustrative dashboard with a real screenshot, or embed a
   compressed clip from `demo.webm` (the current file is about 29 MB, so
   transcode to MP4/WebM under a few MB first).
2. Add a 1200×630 social card PNG and reference it from the Open Graph and
   Twitter meta tags.
3. Add a short changelog or “what’s new” strip once releases are tagged.
4. Add a privacy-friendly analytics snippet only if needed.
5. Wire a link checker and Lighthouse check into CI.
