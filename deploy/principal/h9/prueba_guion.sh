#!/usr/bin/env bash
# Dobles de Podman/psql; solo archivos sintéticos, sin servicios ni red.
set -euo pipefail
SCRIPT=$(realpath -- "$(dirname -- "$0")/instalar.sh")
T=$(mktemp -d "${TMPDIR:-/tmp}/h9-mock-XXXXXXXX")
trap 'rm -rf -- "$T"' EXIT
trap 'printf "FALLO %s\n" "${mode:-preparacion}"; if [[ -f ${F:-}/result ]]; then cat "$F/result"; fi' ERR
mkdir -p "$T/mock-bin"
MODOS=(sql_fail startup_fail success altered_manifest wrong_preimage app_active maintenance_fail recovery_db_fail recovery_art_fail recovery_db_error config_file mounts_match mounts_fail)
if [[ ${1:-} == --web-config ]]; then MODOS=(success config_file mounts_match mounts_fail); fi
cat > "$T/mock-bin/id" <<'MOCK'
#!/usr/bin/env bash
if [[ "$1" == -un ]]; then echo openclaw; else /usr/bin/id "$@"; fi
MOCK
cat > "$T/mock-bin/podman" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == --remote=false ]] && shift
case "$1" in
unshare) shift; exec "$@" ;;
inspect)
  if [[ "$3" == '{{json .Mounts}}' ]]; then
    printf '[{"Source":"/fixture/pgdata","Destination":"/db","Type":"bind"}]\n[{"Source":"/fixture/art","Destination":"/app","Type":"bind"}]\n'
  else cat "$FIXTURE/$4.running"; fi
  ;;
stop) echo false > "$FIXTURE/$4.running" ;;
start)
  [[ "$2" != app || $(cat "$FIXTURE/gate") == closed ]]
  echo true > "$FIXTURE/$2.running"
  if [[ "$2" == app && ( "$MODE" == startup_fail || "$MODE" == recovery_* ) ]]; then echo false > "$FIXTURE/app.running"; fi
  if [[ "$2" == pg ]]; then
    starts=$(cat "$FIXTURE/pg.starts"); starts=$((starts+1)); echo "$starts" > "$FIXTURE/pg.starts"
    if [[ "$starts" == 2 && "$MODE" == recovery_db_fail ]]; then echo changed >> "$FIXTURE/pgdata/rows"; fi
    if [[ "$starts" == 2 && "$MODE" == recovery_art_fail ]]; then echo changed >> "$FIXTURE/art/vec-server"; fi
  fi
  ;;
logs) [[ "$MODE" == startup_fail || "$MODE" == recovery_* ]] || echo 'vec server listening' ;;
exec)
  if [[ "$*" == *pg_isready* ]]; then exit 0; fi
  input=$(cat)
  if [[ "$input" == *PREIMAGEN_MOCK* ]]; then
    [[ "$MODE" != recovery_db_error || $(cat "$FIXTURE/pg.starts") != 2 ]] || exit 1
    cat "$FIXTURE/pgdata/rows"; exit 0
  fi
  [[ "$input" != *TEST_TWO* ]] || echo executed > "$FIXTURE/sql-two.executed"
  if [[ "$input" == *TEST_TWO* && "$MODE" == sql_fail ]]; then echo 'error sintético SQL' >&2; exit 1; fi
  if [[ "$input" == *COMMIT* ]]; then
    echo migration >> "$FIXTURE/pgdata/rows"
    [[ "$MODE" != app_active ]] || echo true > "$FIXTURE/app.running"
    [[ "$MODE" != maintenance_fail ]] || echo broken > "$FIXTURE/gate"
  fi
  ;;
*) echo 'comando mock inesperado' >&2; exit 2 ;;
esac
MOCK
cat > "$T/mock-bin/sleep" <<'MOCK'
#!/usr/bin/env bash
exit 0
MOCK
chmod +x "$T/mock-bin/"*
for mode in "${MODOS[@]}"; do
  F="$T/$mode"; K="$F/kit"; mkdir -p "$K/bin" "$K/web/static" "$K/locales" "$F/pgdata/pg_tblspc" "$F/pgdata/pg_wal" "$F/pgconf" "$F/art/web/static" "$F/art/locales" "$F/conf" "$F/backups"
  echo seed > "$F/pgdata/rows"; echo pgconfig > "$F/pgconf/postgresql.conf"; echo hba > "$F/hba"
  echo old > "$F/art/vec-server"; echo old-web > "$F/art/web/static/app.js"; echo old-i18n > "$F/art/locales/test.json"; echo private > "$F/conf/config"
  if [[ "$mode" == config_file ]]; then mv "$F/conf" "$F/conf-directory"; echo private > "$F/conf"; fi
  echo new > "$K/bin/vec-server"; echo new-web > "$K/web/static/app.js"; echo new-i18n > "$K/locales/test.json"
  echo old-map > "$F/art/web/cartografia.json"; echo new-map > "$K/web/cartografia.json"
  printf 'one.up.sql\ntwo.up.sql\n' > "$K/sql.list"
  printf 'BEGIN;\nSELECT 1; -- TEST_ONE\nCOMMIT;\n' > "$K/one.up.sql"
  printf 'BEGIN;\nSELECT 1; -- TEST_TWO\nCOMMIT;\n' > "$K/two.up.sql"
  echo 'SELECT 1; -- PREIMAGEN_MOCK' > "$K/consultas_preimagen.sql"
  (cd "$K"; find . -type f -printf '%P\n' | LC_ALL=C sort | xargs sha256sum) > "$F/manifest"
  mv "$F/manifest" "$K/SHA256SUMS"
  sha=$(sha256sum "$K/SHA256SUMS"); sha=${sha%% *}
  for action in close check open; do
    cat > "$F/$action" <<HOOK
#!/usr/bin/env bash
set -euo pipefail
case $action in
close) echo closed > '$F/gate' ;;
check) [[ \$(cat '$F/gate') == closed ]] ;;
open) echo open > '$F/gate' ;;
esac
HOOK
    chmod +x "$F/$action"
  done
  echo true > "$F/app.running"; echo true > "$F/pg.running"; echo open > "$F/gate"; echo 0 > "$F/pg.starts"
  db=$(sha256sum "$F/pgdata/rows"); db=${db%% *}
  tree_hash() {
    python3 - "$1" <<'PY'
import hashlib,pathlib,sys
root=pathlib.Path(sys.argv[1]); h=hashlib.sha256()
for p in sorted(root.rglob('*'),key=lambda p:p.relative_to(root).as_posix()):
    h.update(f'{p.relative_to(root).as_posix()}\0{p.stat().st_mode & 0o777:o}\0'.encode())
    h.update(hashlib.sha256(p.read_bytes()).digest() if p.is_file() else b'directory')
print(h.hexdigest())
PY
  }
  cat > "$F/config.sh" <<CFG
APP=app
PG=pg
PGDATA='$F/pgdata'
PGCONF='$F/pgconf'
PGHBA='$F/hba'
ART='$F/art'
CONF='$F/conf'
BACKUP_ROOT='$F/backups'
PREIMAGEN_DB_SHA='$db'
PREIMAGEN_ART_SHA='$(tree_hash "$F/art")'
PREIMAGEN_CONF_SHA='$(tree_hash "$F/conf")'
MANTENIMIENTO_CERRAR='$F/close'
MANTENIMIENTO_COMPROBAR='$F/check'
MANTENIMIENTO_ABRIR='$F/open'
CFG
  if [[ "$mode" == mounts_match || "$mode" == mounts_fail ]]; then
    mounts_sha=$(python3 - <<'PYM'
import hashlib,json
mounts=[[dict(Source='/fixture/pgdata',Destination='/db',Type='bind')],[dict(Source='/fixture/art',Destination='/app',Type='bind')]]
print(hashlib.sha256(json.dumps(mounts,sort_keys=True,separators=(',',':')).encode()).hexdigest())
PYM
)
    [[ "$mode" != mounts_fail ]] || mounts_sha=$(printf '%064d' 0)
    printf "RUNTIME_MOUNTS_SHA='%s'\n" "$mounts_sha" >> "$F/config.sh"
  fi
  chmod 600 "$F/config.sh"
  [[ "$mode" != altered_manifest ]] || echo altered >> "$K/web/static/app.js"
  [[ "$mode" != wrong_preimage ]] || echo unexpected >> "$F/pgdata/rows"
  rc=0
  FIXTURE="$F" MODE="$mode" PATH="$T/mock-bin:$PATH" bash "$SCRIPT" "$K" "$F/config.sh" "$sha" > "$F/result" 2>&1 || rc=$?
  if [[ "$mode" == success || "$mode" == mounts_match ]]; then
    [[ "$rc" == 0 && $(cat "$F/gate") == open && $(cat "$F/app.running") == true ]]
    [[ $(cat "$F/art/vec-server") == new && $(cat "$F/art/web/static/app.js") == new-web && $(cat "$F/art/locales/test.json") == new-i18n && $(cat "$F/art/web/cartografia.json") == new-map ]]
    [[ $(wc -l < "$F/pgdata/rows") == 3 ]]
  elif [[ "$mode" == altered_manifest || "$mode" == config_file || "$mode" == mounts_fail ]]; then
    [[ "$rc" != 0 && $(cat "$F/app.running") == true && $(cat "$F/gate") == open ]]
  elif [[ "$mode" == wrong_preimage ]]; then
    [[ "$rc" != 0 && $(cat "$F/app.running") == false && $(cat "$F/gate") == closed ]]
    [[ $(wc -l < "$F/pgdata/rows") == 2 ]]
  elif [[ "$mode" == recovery_* ]]; then
    [[ "$rc" != 0 && $(cat "$F/gate") == closed && $(cat "$F/app.running") == false ]]
    if rg -q '^RECUPERADA;' "$F/result" "$F/backups"; then exit 1; fi
    [[ $(cat "$F/conf/config") == private ]]
    if [[ "$mode" == recovery_db_fail ]]; then rg -q 'clave=preimagen_db' "$F/result" "$F/backups"; fi
    if [[ "$mode" == recovery_art_fail ]]; then rg -q 'clave=preimagen_artefacto' "$F/result" "$F/backups"; fi
  else
    [[ "$rc" != 0 && $(cat "$F/gate") == closed && $(cat "$F/app.running") == false ]]
    if [[ "$mode" == app_active || "$mode" == maintenance_fail ]]; then [[ ! -e "$F/sql-two.executed" ]]; fi
    rg -q '^RECUPERADA;' "$F/result" "$F/backups"
    [[ $(cat "$F/pgdata/rows") == seed && $(cat "$F/art/vec-server") == old && $(cat "$F/art/web/static/app.js") == old-web && $(cat "$F/art/locales/test.json") == old-i18n ]]
    [[ $(cat "$F/conf/config") == private && $(cat "$F/hba") == hba ]]
    [[ -n $(find "$F/backups" -path '*/postimagen/origen-0/rows' -print -quit) ]]
  fi
  printf 'OK %s\n' "$mode"
done
