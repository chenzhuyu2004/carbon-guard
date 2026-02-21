#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

slugify() {
  echo "$1" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[`*_~]//g; s/[^a-z0-9 _-]//g; s/[[:space:]]+/-/g; s/-+/-/g; s/^-|-$//g'
}

fail=0

check_local_links() {
  while IFS= read -r file; do
    while IFS= read -r raw; do
      link="${raw%% *}"

      case "$link" in
        http://*|https://*|mailto:*|tel:*|data:* )
          continue
          ;;
        \#* )
          continue
          ;;
      esac

      path="${link%%#*}"
      [ -z "$path" ] && continue

      target="$(realpath -m "$(dirname "$file")/$path")"
      if [ ! -e "$target" ]; then
        echo "BROKEN_LOCAL: $file -> $link"
        fail=1
      fi
    done < <(grep -oP '\[[^\]]+\]\(\K[^)]+' "$file" || true)
  done < <(rg --files -g '*.md' | sort)
}

check_anchor_links() {
  while IFS= read -r file; do
    while IFS= read -r raw; do
      link="${raw%% *}"

      case "$link" in
        http://*|https://*|mailto:*|tel:*|data:* )
          continue
          ;;
      esac

      case "$link" in
        *\#* )
          ;;
        * )
          continue
          ;;
      esac

      path="${link%%#*}"
      frag="${link#*#}"
      [ -z "$frag" ] && continue

      if [ -z "$path" ]; then
        target="$file"
      else
        target="$(realpath -m "$(dirname "$file")/$path")"
      fi

      if [ ! -f "$target" ] || [[ "$target" != *.md ]]; then
        continue
      fi

      found=1
      while IFS= read -r h; do
        text="${h#\#}"
        text="${text#\#}"
        text="${text#\#}"
        text="${text#\#}"
        text="${text#\#}"
        text="${text#\#}"
        text="${text# }"
        if [ "$(slugify "$text")" = "$frag" ]; then
          found=0
          break
        fi
      done < <(grep -E '^#{1,6} ' "$target" || true)

      if [ "$found" -ne 0 ]; then
        echo "BROKEN_ANCHOR: $file -> $link"
        fail=1
      fi
    done < <(grep -oP '\[[^\]]+\]\(\K[^)]+' "$file" || true)
  done < <(rg --files -g '*.md' | sort)
}

check_external_links() {
  mapfile -t links < <(rg --files -g '*.md' | xargs -r grep -h -oP '\[[^\]]+\]\(\Khttps?://[^) ]+' | sort -u)

  for url in "${links[@]:-}"; do
    [ -z "$url" ] && continue

    code=$(curl -L -s -o /dev/null -w '%{http_code}' --max-time 15 --connect-timeout 8 -I "$url" || true)
    if [ "$code" = "000" ] || [ "$code" -ge 400 ]; then
      code2=$(curl -L -s -o /dev/null -w '%{http_code}' --max-time 15 --connect-timeout 8 "$url" || true)
      if [ "$code2" = "000" ] || [ "$code2" -ge 400 ]; then
        echo "BROKEN_EXTERNAL: $code/$code2 $url"
        fail=1
      fi
    fi
  done
}

check_local_links
check_anchor_links
check_external_links

if [ "$fail" -ne 0 ]; then
  echo "Link check failed"
  exit 1
fi

echo "All documentation links are valid"
