#!/bin/sh
# Pulls the prebuilt task images Harbor's task cache names that Apple's
# `container` does not have yet. Harbor otherwise pulls inside the container
# start timeout, which multi-GB images (mteb-*, pytorch-*) overrun. Uses no
# model quota; safe to run at any time and to rerun.
set -u
CACHE=${HARBOR_TASK_CACHE:-$HOME/.cache/harbor/tasks}
[ -d "$CACHE" ] || exit 0
have=$(container image list 2>/dev/null | awk 'NR>1 {print $1 ":" $2}')
find "$CACHE" -name task.toml -exec sed -n 's/^docker_image *= *"\(.*\)"/\1/p' {} + | sort -u |
while read -r image; do
  printf '%s\n' "$have" | grep -qxF "$image" && continue
  for attempt in 1 2 3; do
    echo "pulling $image (attempt $attempt)"
    container image pull --progress none "$image" && break
    sleep $((attempt * 10))
  done
done
