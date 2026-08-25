# Independent production and staging deployments

Use two independent checkouts so the environments can run different Git
revisions at the same time:

```text
/opt/finis-kitchens-production  # production revision
/opt/finis-kitchens-staging     # staging revision
```

They may be ordinary separate clones or separately managed Git worktrees. The
important property is that updating or checking out a revision in one directory
never changes the source used by the other deployment.

The only intentionally shared infrastructure is the VPS, Docker engine,
Traefik, and its external `traefik` network. Each application deployment uses
its own Compose project, images, private networks and Docker-secret objects.
Never run a production command in the staging checkout or vice versa.

## Runtime security verification

Before staging deployment sign-off, run the read-only attestation from the
staging checkout on the Docker host:

```sh
cd /opt/finis-kitchens-staging
bash scripts/verify-runtime.sh staging
```

The script requires both the explicit `staging` argument and the staging
checkout path, so it refuses to run from production. It reads Docker and file
metadata only; it never reads environment or secret values and never performs
Docker lifecycle actions. Investigate any `FAIL` result before deployment or
sign-off. `WARN` results identify metadata that needs operator review without
changing the runtime.

## Immutable container base images

Dockerfiles keep their human-readable tags to show the intended image family
and pin each external base image to a registry manifest digest. The digest is
the immutable build input; update it only by deliberately resolving and
reviewing the replacement digest from the upstream registry.

## Host-file ownership and permissions

Docker Compose project scoping isolates Docker resources; it does **not**
protect a host secret file that is readable by another host user. Keep each
deployment's secret directory restricted (normally `0700`) and each environment
file at `0600`. A secret source file must also be readable by the non-root
runtime UID/GID that consumes its `/run/secrets/...` mount; `0600` owned only by
the deploying operator is not a safe default for these distroless services.

Before changing permissions on existing files, inspect their current owner and
mode *and* inspect the runtime user configured in each image. Do not change
ownership as part of this procedure; if the owner or runtime identity is not
understood, ask that account's administrator to make the change.

```sh
# Run in the relevant checkout. This reads metadata only, not secret contents.
stat -c '%A %a %U:%G %n' .env.production secrets 2>/dev/null || true
find secrets -maxdepth 1 -type f -name '*.txt' -exec stat -c '%A %a %U:%G %n' {} +

# Run after the images have been built. This reads image metadata only.
docker image inspect finis-production-enquiry:latest \
  --format '{{.Config.User}}'
docker image inspect finis-production-enquiry-proxy:latest \
  --format '{{.Config.User}}'
```

Once ownership and the consuming runtime UID/GID are confirmed, set narrowly
restricted owner/group/mode combinations that allow that runtime identity to
read each required secret file. Do not apply a blanket `chmod 0600` to
operator-owned secret files. Staging was successfully configured using narrowly
restricted source files that were readable by the runtime user.

The secret directory may remain `0700`; Docker mounts the selected source files
into the containers. The host source files themselves still need the correct
owner/group/mode for the runtime identity. Set the environment file mode
separately:

```sh
chmod 0700 secrets
chmod 0600 .env.production
```

Use the equivalent staging paths and `.env.staging` in the staging checkout.
The secret values themselves must be provisioned through an approved secure
method; these commands neither print nor create them.

### Troubleshooting: `permission denied reading /run/secrets/...`

This indicates that a Go service started as its non-root runtime user cannot
read a mounted source file. Inspect service status/logs, image runtime users,
and source-file metadata without reading secret contents:

```sh
docker compose --project-name finis-staging --env-file .env.staging \
  -f docker-compose.yml ps
docker compose --project-name finis-staging --env-file .env.staging \
  -f docker-compose.yml logs --tail=50 enquiry enquiry-proxy
docker image inspect finis-staging-enquiry:latest --format '{{.Config.User}}'
docker image inspect finis-staging-enquiry-proxy:latest --format '{{.Config.User}}'
stat -c '%A %a %U:%G %n' .env.staging secrets-staging 2>/dev/null || true
find secrets-staging -maxdepth 1 -type f -name '*.txt' \
  -exec stat -c '%A %a %U:%G %n' {} +
```

Have an authorised operator correct only the affected source file's
owner/group/mode so it remains narrowly restricted while readable by the
documented runtime UID/GID, then recreate the affected services through the
normal deployment procedure. Do not weaken all secret files to world-readable
permissions.

## Production

Checkout: `/opt/finis-kitchens-production`  
Expected Compose project name: `finis-production`.

1. Create `.env.production` from `.env.production.example` and supply the
   production non-secret configuration. Set its mode to `0600`.
2. Provision the four files in the directory named by `SECRETS_DIR` (normally
   `./secrets`): `turnstile_secret.txt`, `smtp_user.txt`, `smtp_pass.txt`, and
   `internal_enquiry_secret.txt`. Keep the directory restricted and provision
   each source file with narrowly restricted permissions readable by the
   service runtime UID/GID.
3. Inspect the rendered configuration before any lifecycle command:

   ```sh
   cd /opt/finis-kitchens-production
   docker compose --project-name finis-production --env-file .env.production \
     -f docker-compose.yml -f docker-compose.production.yml config
   ```

4. Build and start only after that review:

   ```sh
   cd /opt/finis-kitchens-production
   docker compose --project-name finis-production --env-file .env.production \
     -f docker-compose.yml -f docker-compose.production.yml up -d --build
   ```

## Staging

Checkout: `/opt/finis-kitchens-staging`  
Expected Compose project name: `finis-staging`.

1. Create `.env.staging` from `.env.staging.example`. It must name the staging
   host, a staging Turnstile site key and a non-production `ENQUIRY_TO`. Set
   its mode to `0600`.
2. Create a separate `SECRETS_DIR` (normally `./secrets-staging`) containing
   the same four filenames as production. Do not reuse production Turnstile
   credentials or the production proxy-to-enquiry secret. The staging enquiry
   recipient must not be the client's production destination.
3. Staging may deliberately use the same SMTP infrastructure as production if
   it has the non-production recipient above. Prefer a staging-specific sender
   identity or credential where the SMTP provider supports it; a separate SMTP
   server is not required for this deployment boundary. Use a clear subject
   prefix such as `[FINIS STAGING]`.
4. The still-running legacy `finis-kitchens` project also advertises the
   staging hostname. Its web router has priority `10`, and its `/api/enquiry`
   router has priority `300`. The staging template deliberately sets `110` and
   `400`, respectively. Traefik therefore sends requests for the staging host
   to `finis-staging` while leaving legacy apex and `www` routers untouched.
5. Inspect the rendered configuration and confirm those rules and priorities:

   ```sh
   cd /opt/finis-kitchens-staging
   docker compose --project-name finis-staging --env-file .env.staging \
     -f docker-compose.yml config
   ```

6. Build and start staging only after that review:

   ```sh
   cd /opt/finis-kitchens-staging
   docker compose --project-name finis-staging --env-file .env.staging \
     -f docker-compose.yml up -d --build
   ```

7. Before using the staging site or testing its form, verify the running
   staging containers and their Traefik labels. The expected labels are the
   staging host plus priorities `110` and `400`:

   ```sh
   cd /opt/finis-kitchens-staging
   docker compose --project-name finis-staging --env-file .env.staging \
     -f docker-compose.yml ps

   docker inspect "$(docker compose --project-name finis-staging --env-file .env.staging \
     -f docker-compose.yml ps -q web)" \
     --format '{{index .Config.Labels "traefik.http.routers.finis-staging-web.rule"}} {{index .Config.Labels "traefik.http.routers.finis-staging-web.priority"}}'

   docker inspect "$(docker compose --project-name finis-staging --env-file .env.staging \
     -f docker-compose.yml ps -q enquiry-proxy)" \
     --format '{{index .Config.Labels "traefik.http.routers.finis-staging-enquiry.rule"}} {{index .Config.Labels "traefik.http.routers.finis-staging-enquiry.priority"}}'
   ```

   A read-only web request should include staging's distinctive noindex header,
   proving that the staging web router, rather than the legacy router, is
   serving the host:

   ```sh
   curl --silent --show-error --head https://finiskitchens.southcoastapps.co.uk/ \
     | grep -Ei '^(HTTP/|x-robots-tag:|content-type:)'
   ```

   Do not POST to `/api/enquiry` while performing this routing verification.
   Its separate staging router wins deterministically because `400` is greater
   than the legacy project's `300`.

Staging enables a `noindex, nofollow` meta tag, a disallow-all `robots.txt`,
and an `X-Robots-Tag` response header. Production uses an apex canonical URL;
the production-only override preserves `www` to apex redirection.

## Migration from the current single project

The running v1.0 project is named `finis-kitchens` in
`/opt/finis-kitchens`. Before changing it, save its current Compose file there
as the untracked `docker-compose.v1-backup.yml` and confirm it can render with
the existing `.env`.

Prepare `/opt/finis-kitchens-production` independently at the selected release
revision. An authorised operator must provision its production `.env.production`
and `secrets/` files with permissions appropriate to the service runtime
UID/GID; do not copy production values into the staging checkout. Build the new
production images first; they have distinct `finis-production-*` tags and do
not overwrite the old `finis-*:prod` images.

At cutover, stop the old project and start `finis-production` immediately from
`/opt/finis-kitchens-production`. Both projects must not run production routers
at the same time because Traefik would have two routers for the same host and
priority. This creates a short, planned handoff window.

To roll production back, stop `finis-production`, then run the saved
`/opt/finis-kitchens/docker-compose.v1-backup.yml` again with project name
`finis-kitchens` and its original `.env`. The old images remain available unless
they are manually removed.

To remove staging only, run:

```sh
cd /opt/finis-kitchens-staging
docker compose --project-name finis-staging --env-file .env.staging \
  -f docker-compose.yml down
```

This stops the higher-priority staging routers, so the legacy staging route
resumes automatically until the legacy project is removed. It does not affect
production because its checkout, project, private networks, images, and secret
objects have different names.

## Consent setting

The browser presents consent as mandatory while the current live service is
configured with `REQUIRE_CONSENT=false`. This is deliberately not changed by
the deployment split. Set and review the value independently in each env file
once the business decision is made.
