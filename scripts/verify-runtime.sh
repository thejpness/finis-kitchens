#!/usr/bin/env bash

# Read-only staging runtime attestation. It deliberately uses Docker metadata
# only: it never prints environment values or reads secret-file contents.

set -uo pipefail

expected_environment="staging"
expected_checkout="/opt/finis-kitchens-staging"
expected_project="finis-staging"
legacy_project="finis-kitchens"
staging_hostname="finiskitchens.southcoastapps.co.uk"

pass_count=0
fail_count=0
warn_count=0

pass() {
  printf 'PASS: %s\n' "$1"
  ((pass_count++))
}

fail() {
  printf 'FAIL: %s\n' "$1"
  ((fail_count++))
}

warn() {
  printf 'WARN: %s\n' "$1"
  ((warn_count++))
}

summary() {
  printf 'SUMMARY: PASS=%d FAIL=%d WARN=%d\n' "$pass_count" "$fail_count" "$warn_count"
}

usage() {
  printf 'Usage: %s staging\n' "$0" >&2
}

container_field() {
  docker inspect --format "$2" "$1" 2>/dev/null
}

container_label() {
  docker inspect --format "{{ index .Config.Labels \"$2\" }}" "$1" 2>/dev/null
}

container_networks() {
  docker inspect --format '{{range $network, $_ := .NetworkSettings.Networks}}{{println $network}}{{end}}' "$1" 2>/dev/null
}

contains() {
  local needle=$1
  shift
  local item
  for item in "$@"; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

lookup_service() {
  local project=$1
  local service=$2
  local id
  local -a ids=()

  while IFS= read -r id; do
    [[ -n "$id" ]] && ids+=("$id")
  done < <(docker ps -q \
    --filter "label=com.docker.compose.project=$project" \
    --filter "label=com.docker.compose.service=$service")

  resolved_count=${#ids[@]}
  if [[ "$resolved_count" -eq 1 ]]; then
    resolved_container=${ids[0]}
    return 0
  fi

  return 1
}

resolve_service() {
  if ! lookup_service "$1" "$2"; then
    fail "expected one running $1/$2 container; found $resolved_count"
    return 1
  fi
}

check_equal() {
  local description=$1
  local actual=$2
  local expected=$3

  if [[ "$actual" == "$expected" ]]; then
    pass "$description"
  else
    fail "$description (expected $expected; observed $actual)"
  fi
}

check_non_empty() {
  local description=$1
  local actual=$2

  if [[ -n "$actual" && "$actual" != "<no value>" ]]; then
    pass "$description"
  else
    fail "$description"
  fi
}

check_health() {
  local service=$1
  local container=$2
  local health

  health=$(container_field "$container" '{{if .State.Health}}{{.State.Health.Status}}{{else}}absent{{end}}')
  check_equal "$service health check is healthy" "$health" "healthy"
}

check_no_published_ports() {
  local service=$1
  local container=$2
  local published

  published=$(container_field "$container" '{{range $port, $bindings := .NetworkSettings.Ports}}{{range $bindings}}{{if .HostPort}}{{println .HostIp .HostPort}}{{end}}{{end}}{{end}}')
  if [[ -z "$published" ]]; then
    pass "$service has no published host ports"
  else
    fail "$service has published host ports"
  fi
}

check_hardening() {
  local service=$1
  local container=$2
  local readonly_rootfs cap_drop security_opt restart_policy runtime_user

  readonly_rootfs=$(container_field "$container" '{{.HostConfig.ReadonlyRootfs}}')
  cap_drop=$(container_field "$container" '{{json .HostConfig.CapDrop}}')
  security_opt=$(container_field "$container" '{{json .HostConfig.SecurityOpt}}')
  restart_policy=$(container_field "$container" '{{.HostConfig.RestartPolicy.Name}}')
  runtime_user=$(container_field "$container" '{{.Config.User}}')

  check_equal "$service root filesystem is read-only" "$readonly_rootfs" "true"
  if [[ "$cap_drop" == *'"ALL"'* ]]; then
    pass "$service drops all Linux capabilities"
  else
    fail "$service drops all Linux capabilities"
  fi
  if [[ "$security_opt" == *'no-new-privileges:true'* ]]; then
    pass "$service enables no-new-privileges"
  else
    fail "$service enables no-new-privileges"
  fi
  check_equal "$service restart policy is unless-stopped" "$restart_policy" "unless-stopped"

  case "$runtime_user" in
    ""|root|0|0:*|root:*)
      fail "$service runtime user is non-root"
      ;;
    *)
      pass "$service runtime user is non-root"
      ;;
  esac
}

check_resource_limits() {
  local service=$1
  local container=$2
  local pids memory

  pids=$(container_field "$container" '{{if .HostConfig.PidsLimit}}{{.HostConfig.PidsLimit}}{{else}}0{{end}}')
  memory=$(container_field "$container" '{{if .HostConfig.Memory}}{{.HostConfig.Memory}}{{else}}0{{end}}')

  check_equal "$service PID limit" "$pids" "64"
  check_equal "$service memory limit" "$memory" "134217728"
}

canonical_path() {
  realpath -m -- "$1" 2>/dev/null
}

check_mount() {
  local service=$1
  local container=$2
  local expected_source=$3
  local expected_destination=$4
  local source destination writable extra actual_source
  local found=false

  expected_source=$(canonical_path "$expected_source")
  if [[ -z "$expected_source" ]]; then
    fail "$service expected secret source path is canonical"
    return
  fi

  while IFS=' ' read -r source destination writable extra; do
    if [[ "$destination" == "$expected_destination" ]]; then
      found=true
      actual_source=$(canonical_path "$source")
      check_equal "$service mounts $expected_destination from its staging secret source" \
        "$actual_source" "$expected_source"
      if [[ "$writable" == "false" ]]; then
        pass "$service mounts $expected_destination read-only"
      else
        fail "$service mounts $expected_destination read-only"
      fi
    fi
  done < <(container_field "$container" '{{range .Mounts}}{{println .Source .Destination .RW}}{{end}}')

  if [[ "$found" != true ]]; then
    fail "$service mounts $expected_destination"
  fi
}

check_secret_file_permissions() {
  local path=$1
  local mode mode_number

  if [[ ! -f "$path" ]]; then
    fail "secret source exists: $path"
    return
  fi

  mode=$(stat -c '%a' "$path" 2>/dev/null) || {
    fail "secret source metadata is readable: $path"
    return
  }
  mode_number=$((8#$mode))

  if ((mode_number & 0007)); then
    fail "secret source is not world-accessible: $path"
  else
    pass "secret source is not world-accessible: $path"
  fi
  if ((mode_number & 0020)); then
    warn "secret source is group-writable: $path"
  fi
}

check_environment_file_permissions() {
  local path=$1
  local mode mode_number

  if [[ ! -f "$path" ]]; then
    fail "staging environment file exists: $path"
    return
  fi

  mode=$(stat -c '%a' "$path" 2>/dev/null) || {
    fail "staging environment-file metadata is readable: $path"
    return
  }
  mode_number=$((8#$mode))

  if ((mode_number & 0077)); then
    fail "staging environment file is not group/world-accessible: $path"
  else
    pass "staging environment file is not group/world-accessible: $path"
  fi
}

check_secret_directory_permissions() {
  local path=$1
  local mode mode_number

  if [[ ! -d "$path" ]]; then
    fail "staging secret directory exists"
    return
  fi

  mode=$(stat -c '%a' "$path" 2>/dev/null) || {
    fail "staging secret-directory metadata is readable"
    return
  }
  mode_number=$((8#$mode))

  if ((mode_number & 0007)); then
    fail "staging secret directory is not world-accessible"
  else
    pass "staging secret directory is not world-accessible"
  fi
}

if [[ $# -ne 1 || "$1" != "$expected_environment" ]]; then
  usage
  exit 2
fi

script_path=${BASH_SOURCE[0]:-}
script_dir=${script_path%/*}
if [[ "$script_dir" == "$script_path" ]]; then
  script_dir=.
fi
repository_root=$(cd -- "$script_dir/.." && pwd -P)
current_directory=$(pwd -P)

if [[ "$repository_root" != "$expected_checkout" && "$current_directory" != "$expected_checkout" ]]; then
  fail "refusing to run outside $expected_checkout"
  summary
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  fail "Docker CLI is available"
  summary
  exit 1
fi

declare -A containers
missing_service=false
for service in web enquiry-proxy enquiry; do
  if resolve_service "$expected_project" "$service"; then
    containers[$service]=$resolved_container
  else
    missing_service=true
  fi
done

if [[ "$missing_service" == true ]]; then
  summary
  exit 1
fi

pass "Compose project $expected_project has all expected running services"

for service in web enquiry-proxy enquiry; do
  container=${containers[$service]}
  check_equal "$service belongs to $expected_project" \
    "$(container_label "$container" 'com.docker.compose.project')" "$expected_project"
  check_equal "$service records the staging checkout" \
    "$(container_label "$container" 'com.docker.compose.project.working_dir')" "$expected_checkout"
  check_equal "$service records the staging Compose file" \
    "$(container_label "$container" 'com.docker.compose.project.config_files')" \
    "$expected_checkout/docker-compose.yml"
  check_health "$service" "$container"
  check_no_published_ports "$service" "$container"
  check_hardening "$service" "$container"
done

check_resource_limits "enquiry-proxy" "${containers[enquiry-proxy]}"
check_resource_limits "enquiry" "${containers[enquiry]}"

web_container=${containers[web]}
proxy_container=${containers[enquiry-proxy]}
enquiry_container=${containers[enquiry]}

check_equal "web enables Traefik" \
  "$(container_label "$web_container" 'traefik.enable')" "true"
check_equal "enquiry-proxy enables Traefik" \
  "$(container_label "$proxy_container" 'traefik.enable')" "true"
check_equal "enquiry disables Traefik" \
  "$(container_label "$enquiry_container" 'traefik.enable')" "false"

resource_prefix=$expected_project
web_rule_key="traefik.http.routers.${resource_prefix}-web.rule"
web_priority_key="traefik.http.routers.${resource_prefix}-web.priority"
enquiry_rule_key="traefik.http.routers.${resource_prefix}-enquiry.rule"
enquiry_priority_key="traefik.http.routers.${resource_prefix}-enquiry.priority"
expected_web_rule="Host(\`${staging_hostname}\`)"
expected_enquiry_rule="Host(\`${staging_hostname}\`) && Path(\`/api/enquiry\`)"

check_equal "staging web router rule" \
  "$(container_label "$web_container" "$web_rule_key")" "$expected_web_rule"
check_equal "staging web router priority" \
  "$(container_label "$web_container" "$web_priority_key")" "110"
check_equal "staging enquiry router rule" \
  "$(container_label "$proxy_container" "$enquiry_rule_key")" "$expected_enquiry_rule"
check_equal "staging enquiry router priority" \
  "$(container_label "$proxy_container" "$enquiry_priority_key")" "400"

web_traefik_network=$(container_label "$web_container" 'traefik.docker.network')
proxy_traefik_network=$(container_label "$proxy_container" 'traefik.docker.network')
check_non_empty "web declares its Traefik network" "$web_traefik_network"
check_non_empty "enquiry-proxy declares its Traefik network" "$proxy_traefik_network"
check_equal "web and enquiry-proxy declare the same Traefik network" \
  "$proxy_traefik_network" "$web_traefik_network"
traefik_network=$web_traefik_network

mapfile -t web_networks < <(container_networks "$web_container")
mapfile -t proxy_networks < <(container_networks "$proxy_container")
mapfile -t enquiry_networks < <(container_networks "$enquiry_container")

if contains "$web_traefik_network" "${web_networks[@]}"; then
  pass "web is attached to its Traefik network"
else
  fail "web is attached to its Traefik network"
fi
if contains "$proxy_traefik_network" "${proxy_networks[@]}"; then
  pass "enquiry-proxy is attached to the Traefik network"
else
  fail "enquiry-proxy is attached to the Traefik network"
fi
if contains "$web_traefik_network" "${enquiry_networks[@]}" || contains "$proxy_traefik_network" "${enquiry_networks[@]}"; then
  fail "enquiry is not attached to the Traefik network"
else
  pass "enquiry is not attached to the Traefik network"
fi

internal_network=""
egress_network=""
for network in "${proxy_networks[@]}"; do
  if [[ "$network" == "$traefik_network" ]] || ! contains "$network" "${enquiry_networks[@]}"; then
    continue
  fi

  network_project=$(docker network inspect --format '{{ index .Labels "com.docker.compose.project" }}' "$network" 2>/dev/null)
  network_internal=$(docker network inspect --format '{{.Internal}}' "$network" 2>/dev/null)
  if [[ "$network_project" != "$expected_project" ]]; then
    continue
  fi
  if [[ "$network_internal" == "true" ]]; then
    internal_network=$network
  elif [[ "$network_internal" == "false" ]]; then
    egress_network=$network
  fi
done

if [[ -n "$internal_network" ]]; then
  pass "enquiry and enquiry-proxy share a project-scoped internal network"
else
  fail "enquiry and enquiry-proxy share a project-scoped internal network"
fi
if [[ -n "$egress_network" ]]; then
  pass "enquiry and enquiry-proxy share a project-scoped egress network"
else
  fail "enquiry and enquiry-proxy share a project-scoped egress network"
fi
if [[ -n "$internal_network" ]] && contains "$internal_network" "${web_networks[@]}"; then
  fail "web is not attached to the internal network"
else
  pass "web is not attached to the internal network"
fi

check_mount "enquiry-proxy" "$proxy_container" \
  "$expected_checkout/secrets-staging/turnstile_secret.txt" \
  "/run/secrets/finis_turnstile_secret"
check_mount "enquiry-proxy" "$proxy_container" \
  "$expected_checkout/secrets-staging/internal_enquiry_secret.txt" \
  "/run/secrets/finis_internal_enquiry_secret"
check_mount "enquiry" "$enquiry_container" \
  "$expected_checkout/secrets-staging/smtp_user.txt" \
  "/run/secrets/finis_smtp_user"
check_mount "enquiry" "$enquiry_container" \
  "$expected_checkout/secrets-staging/smtp_pass.txt" \
  "/run/secrets/finis_smtp_pass"
check_mount "enquiry" "$enquiry_container" \
  "$expected_checkout/secrets-staging/internal_enquiry_secret.txt" \
  "/run/secrets/finis_internal_enquiry_secret"

check_environment_file_permissions "$expected_checkout/.env.staging"
check_secret_directory_permissions "$expected_checkout/secrets-staging"
for secret_file in \
  turnstile_secret.txt \
  smtp_user.txt \
  smtp_pass.txt \
  internal_enquiry_secret.txt; do
  check_secret_file_permissions "$expected_checkout/secrets-staging/$secret_file"
done

named_runtime_identity=false
for service in enquiry-proxy enquiry; do
  runtime_user=$(container_field "${containers[$service]}" '{{.Config.User}}')
  if [[ ! "$runtime_user" =~ ^[0-9]+(:[0-9]+)?$ ]]; then
    named_runtime_identity=true
  fi
done
if [[ "$named_runtime_identity" == true ]]; then
  warn "named non-root runtime users prevent a reliable host UID/GID comparison for source secrets"
else
  pass "numeric runtime users allow host UID/GID comparison for source secrets"
fi

legacy_available=true
if lookup_service "$legacy_project" web; then
  legacy_web_container=$resolved_container
else
  legacy_available=false
fi
if lookup_service "$legacy_project" enquiry-proxy; then
  legacy_proxy_container=$resolved_container
else
  legacy_available=false
fi

if [[ "$legacy_available" == true ]]; then
  if [[ "$expected_project" != "$legacy_project" ]]; then
    pass "staging and legacy Compose projects differ"
  else
    fail "staging and legacy Compose projects differ"
  fi
  check_equal "legacy web router priority" \
    "$(container_label "$legacy_web_container" 'traefik.http.routers.finis-web.priority')" "10"
  check_equal "legacy enquiry router priority" \
    "$(container_label "$legacy_proxy_container" 'traefik.http.routers.finis-enquiry.priority')" "300"
  pass "staging router priorities outrank the legacy router priorities"
else
  warn "legacy Compose project is not fully running; router-precedence comparison skipped"
fi

summary
if ((fail_count > 0)); then
  exit 1
fi
