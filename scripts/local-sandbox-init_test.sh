#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT
mkdir -p "$temp_dir/bin"

cat >"$temp_dir/bin/stub" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
command_name="$(basename "$0")"
printf '%s %s\n' "$command_name" "$*" >>"$FAKE_LOG"

case "$command_name" in
  k3d)
    if [[ "${1:-}" == "cluster" && "${2:-}" == "list" ]]; then
      printf 'NAME SERVERS AGENTS LOADBALANCER\n'
      for cluster in ${FAKE_K3D_CLUSTERS:-}; do
        printf '%s 1/1 2/2 true\n' "$cluster"
      done
    fi
    ;;
  kubectl)
    case "${1:-} ${2:-}" in
      "create namespace") printf 'apiVersion: v1\nkind: Namespace\n' ;;
      "apply") cat >/dev/null ;;
      "cluster-info") printf 'Kubernetes control plane is running\n' ;;
      "get nodes") printf 'k3d-agent Ready\n' ;;
      "get namespace") exit 0 ;;
      *) exit 0 ;;
    esac
    ;;
  *)
    exit 0
    ;;
esac
STUB
chmod +x "$temp_dir/bin/stub"
for command_name in k3d kubectl helm docker go; do
  ln -s stub "$temp_dir/bin/$command_name"
done

run_initializer() {
  FAKE_LOG="$temp_dir/commands.log" \
  FAKE_K3D_CLUSTERS="$1" \
  KTTM_CLUSTER_NAME=kttm-dev \
  KTTM_SKIP_BREW=1 \
  KTTM_SKIP_DEPS=1 \
  PATH="$temp_dir/bin:/usr/bin:/bin" \
    bash "$repo_root/scripts/local-sandbox-init.sh" >"$temp_dir/output.log" </dev/null
}

cd "$repo_root"
: >"$temp_dir/commands.log"
run_initializer "kttm-dev"
run_initializer "kttm-dev"

if grep -q 'Delete and recreate?' "$temp_dir/output.log"; then
  printf 'initializer prompted to delete an existing cluster\n' >&2
  exit 1
fi
if grep -q '^k3d cluster delete ' "$temp_dir/commands.log"; then
  printf 'initializer deleted an existing cluster\n' >&2
  exit 1
fi
if grep -q '^k3d cluster create ' "$temp_dir/commands.log"; then
  printf 'initializer recreated an existing cluster\n' >&2
  exit 1
fi

: >"$temp_dir/commands.log"
run_initializer "kttm-dev-old"
if ! grep -q '^k3d cluster create --config ' "$temp_dir/commands.log"; then
  printf 'initializer failed to distinguish a similarly named cluster\n' >&2
  exit 1
fi