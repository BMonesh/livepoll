# LivePoll

LivePoll lets a creator publish a question, share a link or QR code, and watch votes arrive live. Creators sign up with email and password. Viewers can vote and see results from the shared link.

## Run locally

With Docker and the Compose plugin running:

```sh
docker compose up --build -d
```

Open [LivePoll on localhost](http://localhost:8080). Sign up, create a poll with two to six options, and share its URL. Leave the poll open in one browser and cast a ballot from another browser profile. Both pages should show the new count automatically. Close voting from the creator dashboard to leave a final result at the same URL.

The local URL is accessible from the computer running Docker. To scan the QR code from a phone or share it with other people, use the public HTTPS deployment.

To stop the services while preserving the local database:

```sh
docker compose down
```

### Development

Use a Go toolchain compatible with `backend/go.mod` and Node.js 22.12+ or 24. Start the databases with `docker compose up -d mongo-init redis` and wait for `mongo-init` to exit successfully.

In one terminal, run the Go API with `PUBLIC_ORIGIN` set to `http://localhost:5173`. In PowerShell:

```powershell
cd backend
$env:PUBLIC_ORIGIN = 'http://localhost:5173'
go run ./cmd/server
```

In another terminal, run:

```sh
cd frontend
npm ci
npm run dev
```

Vite proxies API calls to port 8080. Open [the Vite dev server](http://localhost:5173) and use that URL throughout the session. See `.env.example` for the available variables. The Go app reads them from the process environment.

## How a ballot reaches a viewer

1. React uses HTTP for poll reads and votes. Each open poll also connects to a Gin Server-Sent Events (SSE) endpoint.
2. Gin checks the voter cookie and option. A MongoDB transaction records the ballot and increments the matching option, total, and revision. A unique index on `pollId` and `voter` protects against duplicate ballots. When a vote is retried, the API returns the stored choice.
3. Gin publishes a Redis Pub/Sub notification after a successful commit. Each app instance receives it and wakes the local viewers watching that poll. Their SSE streams send a fresh, versioned snapshot from MongoDB.
4. Redis Pub/Sub can miss notifications during an outage. Each stream reads a fresh snapshot on connect and checks again every 12 seconds, so a committed vote still reaches viewers if its notification was missed.

MongoDB holds the accounts, expiring sessions, polls, and ballots. Redis handles the immediate update notifications and short-lived request limits. The frontend generates the QR code using the same URL as **Copy link**.

### Audience rules

- A poll is unlisted. Anyone with the link can vote and watch the results. Only the owner can close it.
- A browser receives a random, HTTP-only voter cookie. The ballot stores a digest of that token. Clearing cookies, switching browser profiles, or starting a new incognito session can enable another ballot.
- IP-based request limits expire after a minute and allow a surge of 30 voters on a shared connection. A limit applies to request frequency. Polls have no headcount cap.
- Poll options remain fixed after publishing. Closing a poll prevents new ballots and leaves final results available.

## Verification

```sh
cd backend
go test ./...
```

With the local replica set running, execute the concurrent ballot integration test. In PowerShell:

```powershell
$env:TEST_MONGO_URI = 'mongodb://localhost:27017/?replicaSet=rs0&directConnection=true'
go test -v ./internal/polls
```

The integration test submits votes for 30 distinct voter IDs with two simultaneous retries, compares the ballot collection with the final count, and checks ownership and closed-poll behavior. Run `npm ci` and `npm run build` in `frontend` for the TypeScript and production build checks.

## Deploy

Deploy the supplied Dockerfile to a container host with a public HTTPS domain, a MongoDB replica set, and a Redis endpoint. Configure these values through the host's environment or secrets settings:

| Variable | Production value |
|---|---|
| `PUBLIC_ORIGIN` | Exact HTTPS origin of the site, for example `https://poll.example.com`. Use the same canonical host for shared links. |
| `MONGO_URI` | Authenticated MongoDB replica-set URI, with transaction support and network access from the app. |
| `MONGO_DATABASE` | Database name, default `livepoll`. Give the app rights to manage indexes in this database. |
| `REDIS_URL` | Redis connection URL. Use `rediss://` for TLS when supported by the provider. |
| `TRUSTED_PROXIES` | Comma-separated proxy IPs or CIDRs when the app sits behind a trusted proxy that supplies client IP headers. Leave empty to use the immediate connection address. |
| `PORT` | HTTP listen port. The image defaults to 8080. |

Use `/healthz` for readiness checks. Configure any reverse proxy to stream `/api/polls/*/events` without response buffering or compression and keep connections open between heartbeats. After deploying, open the public URL and repeat the two-browser voting and QR scan checks. The local Compose configuration is intended for development.

## Submission

The brief calls for a public GitHub repo, a publicly reachable deployment, and a 3–5 minute video on unlisted YouTube or a public Google Drive link. In the video, demonstrate two browsers sharing a live tally, explain the most difficult issue you actually encountered and how you resolved it, and describe how you used AI.
