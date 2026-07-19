#!/usr/bin/env bash
set -Eeuo pipefail

directorio="${1:-}"
if [[ -z "$directorio" || ! -d "$directorio" ]]; then
  echo "Uso: $0 DIRECTORIO_LEAFLET_1.9.4" >&2
  exit 2
fi

cd -- "$directorio"
sha256sum -c <<'HUELLAS'
a7837102824184820dfa198d1ebcd109ff6d0ff9a2672a074b9a1b4d147d04c6  leaflet.css
85d455b4522415f6badc42a0e7d17c919d100347d6b8958bd0dc738fdecd6d50  leaflet.js
1dbbe9d028e292f36fcba8f8b3a28d5e8932754fc2215b9ac69e4cdecf5107c6  images/layers.png
066daca850d8ffbef007af00b06eac0015728dee279c51f3cb6c716df7c42edf  images/layers-2x.png
574c3a5cca85f4114085b6841596d62f00d7c892c7b03f28cbfa301deb1dc437  images/marker-icon.png
00179c4c1ee830d3a108412ae0d294f55776cfeb085c60129a39aa6fc4ae2528  images/marker-icon-2x.png
264f5c640339f042dd729062cfc04c17f8ea0f29882b538e3848ed8f10edb4da  images/marker-shadow.png
53e8dc25862014e4324741ca18fbe3611e11d42ef69f59f86ea8c5389647d4cb  LICENSE
HUELLAS

# La distribución oficial contiene enlaces explicativos dentro de comentarios
# (incidencias de navegadores y página del proyecto). Se eliminan únicamente
# los comentarios para comprobar las instrucciones que sí se ejecutan; las
# huellas anteriores siguen garantizando que los artefactos están intactos.
sin_comentarios="$(mktemp)"
urls_no_permitidas="$(mktemp)"
trap 'rm -f "$sin_comentarios" "$urls_no_permitidas"' EXIT
perl -0777 -pe 's{/\*.*?\*/}{}gs' leaflet.js leaflet.css >"$sin_comentarios"
perl -ne 'while (m{https?://[^"'"'"'\s,)]+}g) { print "$&\n" }' "$sin_comentarios" \
  | sort -u \
  | grep -Ev '^(http://www\.w3\.org/2000/svg|https://leafletjs\.com)$' \
  >"$urls_no_permitidas" || true
if [[ -s "$urls_no_permitidas" ]] \
  || rg -n '//unpkg|//cdn|//cdnjs|//jsdelivr|tile\.openstreetmap\.org' "$sin_comentarios"; then
  cat "$urls_no_permitidas" >&2
  echo "ERROR: la distribucion contiene una dependencia remota ejecutable inesperada." >&2
  exit 1
fi

echo "OK: Leaflet 1.9.4 local coincide con los artefactos upstream aprobados."
