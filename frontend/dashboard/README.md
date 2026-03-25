# Sentinel dashboard

Next.js UI for the Sentinel Fraud Engine API gateway: sign-in (JWT), live transaction feed, open alerts with resolve, and aggregate metrics. Data refreshes every five seconds while the page is open.

## Prerequisites

- Node.js 18+
- API gateway reachable from the browser (default `http://localhost:8000`), e.g. after `./start.sh` from the monorepo root.

## Environment

Copy the example file and adjust if your gateway is not on port 8000:

```bash
cp .env.example .env.local
```

| Variable | Description |
|----------|-------------|
| `NEXT_PUBLIC_API_BASE_URL` | API gateway origin (scheme + host + port). Example: `http://localhost:8000`. Must be `NEXT_PUBLIC_*` because the browser calls the API directly. |

If the gateway runs only inside Docker on your machine, `localhost:8000` is correct as long as compose publishes port `8000`.

## Commands

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). Use **Sign in** with the seeded analyst (see monorepo `database/seeds/001_users.sql`), e.g. `analyst@sentinel.com` / `sentinel123`.

```bash
npm run build   # production build
npm run start   # serve production build on port 3000
npm run lint
```

## Notes

- The gateway accepts any email/password for demo JWT issuance; the dashboard still expects a successful `/api/auth/login` response.
- Tokens are stored in `localStorage` under `sentinel_jwt`.
- There is no WebSocket in the gateway yet; polling drives the “live” view.
