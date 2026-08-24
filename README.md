# Fini's Kitchens

Website and enquiry platform for **Fini's Kitchens**.

The application is built with Astro and deployed through Docker, with nginx serving the generated frontend, Traefik providing external routing, and separate Go services handling contact-form enquiries and Turnstile validation.

## Technology

* Astro
* TypeScript
* Docker / Docker Compose
* nginx
* Traefik
* Cloudflare Turnstile
* Go enquiry service
* Go enquiry proxy

## Repository structure

```text
/
├── docker/
│   ├── Dockerfile.prod
│   └── nginx.conf
├── docs/
│   └── deployment.md
├── public/
├── services/
│   ├── enquiry/
│   └── enquiry-proxy/
├── src/
│   ├── assets/
│   ├── components/
│   ├── content/
│   ├── layouts/
│   ├── pages/
│   └── styles/
├── docker-compose.yml
├── docker-compose.local.yml
├── docker-compose.production.yml
├── astro.config.mjs
├── package.json
└── README.md
```

## Local development

### Frontend only

For normal Astro development:

```bash
npm install
npm run dev
```

The Astro development server is available by default at:

```text
http://localhost:4321
```

Build the static site locally with:

```bash
npm run build
```

Preview a completed build with:

```bash
npm run preview
```

### Full local stack

To run the frontend, Traefik, enquiry service and enquiry proxy together:

```bash
docker compose -f docker-compose.local.yml up --build
```

The local Compose environment uses:

```text
.env.local
```

and expects the local secret files configured by `docker-compose.local.yml`.

To stop the local stack:

```bash
docker compose -f docker-compose.local.yml down
```

## Validation

Before committing application changes, run:

```bash
npm run build
git diff --check
```

There are currently no separate project-level lint, typecheck or automated test scripts defined in `package.json`.

## Deployment model

Production and staging are intentionally isolated using:

* separate Git checkouts
* separate Docker Compose project names
* separate environment files
* separate image names
* separate Docker networks
* separate secret directories

The only intentionally shared infrastructure is the VPS, Docker engine and external Traefik network.

### Important deployment rule

> **Do not run bare `docker compose` lifecycle commands in staging or production.**

Always provide the correct:

* Compose project name
* environment file
* Compose file(s)

For example, this is **not** a valid staging deployment command:

```bash
docker compose up -d
```

Without the staging environment file, variables such as `DEPLOYMENT_ENV`, `PUBLIC_SITE_ORIGIN`, `PUBLIC_TURNSTILE_SITE_KEY`, `SITE_DOMAIN`, `TRAEFIK_NETWORK` and secret paths may resolve to blank values.

That can cause invalid builds or incorrect Docker/Traefik resources.

---

# Staging

Staging checkout:

```text
/opt/finis-kitchens-staging
```

Compose project:

```text
finis-staging
```

Environment file:

```text
.env.staging
```

Secrets are normally stored in:

```text
./secrets-staging/
```

## Update staging source

```bash
cd /opt/finis-kitchens-staging

git status --short
git switch main
git pull --ff-only origin main

git log -1 --oneline
```

Do not delete `secrets-staging/` if it appears as an untracked runtime directory.

## Validate staging Compose configuration

Before a build or lifecycle operation:

```bash
cd /opt/finis-kitchens-staging

docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  config --quiet
```

## Full staging deployment

Use this when application services or deployment configuration have changed:

```bash
cd /opt/finis-kitchens-staging

docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  up -d --build
```

## Web-only staging deployment

For changes limited to the Astro frontend, the web service can be rebuilt without restarting the enquiry services.

Build:

```bash
cd /opt/finis-kitchens-staging

docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  build --no-cache web
```

Replace only the web container:

```bash
docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  up -d --no-deps web
```

## Check staging health

```bash
docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  ps
```

Web logs:

```bash
docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  logs --tail=100 web
```

Enquiry logs:

```bash
docker compose \
  --project-name finis-staging \
  --env-file .env.staging \
  -f docker-compose.yml \
  logs --tail=100 enquiry enquiry-proxy
```

All three services should normally report healthy:

```text
finis-staging-web-1
finis-staging-enquiry-1
finis-staging-enquiry-proxy-1
```

## Verify staging routing

A read-only request can be used to confirm that the staging Traefik router is serving the site:

```bash
curl --silent --show-error --head \
  https://finiskitchens.southcoastapps.co.uk/ \
  | grep -Ei '^(HTTP/|x-robots-tag:|content-type:)'
```

The staging response should include:

```text
X-Robots-Tag: noindex, nofollow
```

Do not POST to `/api/enquiry` simply to test routing.

After frontend or contact-form work, perform a deliberate browser smoke test:

1. Confirm the page renders correctly.
2. Check desktop and mobile layouts.
3. Confirm Cloudflare Turnstile loads.
4. Submit one staging enquiry where appropriate.
5. Confirm the browser success state.
6. Confirm the enquiry reaches the staging recipient.

---

# Production

Production checkout:

```text
/opt/finis-kitchens-production
```

Compose project:

```text
finis-production
```

Environment file:

```text
.env.production
```

Production additionally uses:

```text
docker-compose.production.yml
```

Secrets are normally stored in:

```text
./secrets/
```

## Update production source

```bash
cd /opt/finis-kitchens-production

git status --short
git switch main
git pull --ff-only origin main

git log -1 --oneline
```

## Validate production Compose configuration

Always inspect or validate the resolved configuration before deployment:

```bash
cd /opt/finis-kitchens-production

docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  config --quiet
```

## Full production deployment

```bash
cd /opt/finis-kitchens-production

docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  up -d --build
```

## Web-only production deployment

For a frontend-only release:

```bash
cd /opt/finis-kitchens-production

docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  build --no-cache web
```

Then replace only the web container:

```bash
docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  up -d --no-deps web
```

## Check production health

```bash
docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  ps
```

Web logs:

```bash
docker compose \
  --project-name finis-production \
  --env-file .env.production \
  -f docker-compose.yml \
  -f docker-compose.production.yml \
  logs --tail=100 web
```

---

# Environment configuration

Environment-specific configuration must remain separate.

Typical files are:

```text
.env.local
.env.staging
.env.production
```

Example templates are provided where appropriate:

```text
.env.staging.example
.env.production.example
```

Do not commit real credentials, passwords, API secrets or private environment values.

## Docker secrets

The deployed enquiry stack expects four secret files:

```text
turnstile_secret.txt
smtp_user.txt
smtp_pass.txt
internal_enquiry_secret.txt
```

Staging and production must use their own secret directories and deployment credentials.

Do not copy production Turnstile or internal enquiry credentials into staging.

Secret file ownership and permissions matter because the enquiry services run as non-root users. Do not use blanket permission changes when troubleshooting.

See the deployment documentation before modifying secret ownership or modes.

---

# Contact enquiry architecture

The browser submits enquiries to:

```text
POST /api/enquiry
```

Traefik routes the request to the enquiry proxy.

The proxy handles controls including:

* Cloudflare Turnstile validation
* body-size controls
* rate limiting
* concurrent-request limits

Validated requests are then passed internally to the enquiry service, which handles email delivery.

The browser payload currently includes:

```text
name
email
phone
timeline
message
page
source
consent
company
captchaToken
channel
```

Changes to the Contact page should preserve this contract unless the enquiry architecture is being intentionally changed and tested.

---

# Git workflow

Normal feature work should be performed on a branch rather than directly on `main`.

Example:

```bash
git switch main
git pull --ff-only origin main
git switch -c feat/example-change
```

Before committing:

```bash
npm run build
git diff --check
git status --short
```

After review:

```bash
git add .
git commit -m "feat: describe change"
git push -u origin feat/example-change
```

Create a pull request into `main`, validate the change, then merge.

After merge, clean up locally:

```bash
git switch main
git pull --ff-only origin main
git branch -d feat/example-change
git fetch --prune
git status
```

---

# Operational safety

Before staging or production changes:

* confirm you are in the correct checkout
* inspect `git status`
* confirm the intended Git revision
* use the correct Compose project name
* use the correct environment file
* include the correct Compose override files
* validate Compose configuration before deployment
* avoid restarting unrelated services for frontend-only changes
* check container health after deployment
* inspect logs before treating a deployment as successful

Do **not** use destructive commands such as these as routine deployment steps:

```text
git reset --hard
docker system prune
docker compose down -v
rm -rf secrets*
```

If a deployment checkout contains unexpected local changes, investigate them before removing anything.

---

# Further documentation

Detailed deployment architecture, environment isolation, Traefik routing, secret permissions, staging behaviour, migration and rollback procedures are documented in:

```text
docs/deployment.md
```

Treat that document as the authoritative reference for deployment internals and troubleshooting.
