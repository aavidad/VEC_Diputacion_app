#!/usr/bin/env bash
# Ensayo acotado RRHH 4.08 en PostgreSQL 18 sin red ni servicios compartidos.
# --synthetic usa dobles explícitos de la preimagen AD3/CT133, pero instala
# los SQL reales CT131, AD3-99 y CT135. No acredita emisión V3 real.
set -Eeuo pipefail

directorio=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$directorio" rev-parse --show-toplevel)
imagen=postgres:18.4-alpine
modo=${1:-}
if [[ $modo != --synthetic || $# != 1 ]]; then
  echo 'Uso: scripts/rrhh_plantillas/probar_pg18.sh --synthetic' >&2
  echo 'Sin volcado real, este modo acredita solo CT131/AD3-99/CT135 sobre preimagen AD3 sintética.' >&2
  exit 2
fi
for programa in docker go python3 sha256sum; do
  command -v "$programa" >/dev/null || { echo "Falta $programa" >&2; exit 2; }
done
docker info >/dev/null 2>&1 || { echo 'Docker no disponible' >&2; exit 2; }
temporal=$(mktemp -d /dev/shm/vec-rrhh-plantillas-pg18.XXXXXXXX)
nombre="vec-rrhh-plantillas-$$"
mkdir -p "$temporal/data" "$temporal/socket" "$temporal/go-cache" "$temporal/go-tmp"
chmod 1777 "$temporal/socket"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm --network none -v "$temporal:/limpiar" --entrypoint /bin/sh "$imagen" \
    -c 'rm -rf /limpiar/data /limpiar/socket' >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT
docker run -d --rm --network none --name "$nombre" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$temporal/data:/var/lib/postgresql" \
  -v "$temporal/socket:/var/run/postgresql" "$imagen" >/dev/null
for _ in $(seq 1 240); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' &&
     docker exec "$nombre" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$nombre" pg_isready -q -U postgres -d postgres || {
  docker logs "$nombre" >&2 || true
  echo 'PG18 no arrancó' >&2; exit 1;
}
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]] || {
  echo 'Volumen anónimo inesperado' >&2; exit 1;
}
admin() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
consulta() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
runtime() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U vec_plantillas_ejecutor_ensayo -d postgres; }
runtime_scalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U vec_plantillas_ejecutor_ensayo -d postgres -c "$1"; }
[[ $(consulta 'SHOW server_version') == 18.4* ]] || { echo 'Se exige PostgreSQL 18.4' >&2; exit 1; }
echo "PG18 aislado: $(consulta 'SHOW server_version')"
admin <"$directorio/preimagen_sintetica.sql" >/dev/null
ct="$repo/deploy/postgresql/contratacion_temporal/migraciones"
ad3="$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones"
ct131="$ct/000131_catalogo_plantillas_documentos.up.sql"
ad399="$ad3/000099_ambito_organizacion_plantillas_ct.up.sql"
ct135="$ct/000135_ambito_organizacion_plantillas.up.sql"
for f in "$ct131" "$ad399" "$ct135"; do
  [[ -s $f ]] || { echo "Migración ausente: $f" >&2; exit 1; }
  sha256sum "$f"
done
echo 'Fixture: AD3-94/96 y CT133 son dobles; CT131, AD3-99 y CT135 se ejecutan desde migraciones reales.'

instalar() {
  local fichero=$1 sonda=$2
  [[ $(consulta "SELECT $sonda") == f ]] || { echo "Ya instalada: $fichero" >&2; exit 1; }
  sed 's/^COMMIT;$/ROLLBACK;/' "$fichero" | admin >/dev/null
  [[ $(consulta "SELECT $sonda") == f ]] || { echo "ROLLBACK dejó rastro: $fichero" >&2; exit 1; }
  admin <"$fichero" >/dev/null
  [[ $(consulta "SELECT $sonda") == t ]] || { echo "UP no detectable: $fichero" >&2; exit 1; }
  if admin <"$fichero" >/dev/null 2>&1; then echo "Doble UP aceptado: $fichero" >&2; exit 1; fi
  echo "OK $(basename "$fichero"): ROLLBACK, UP, doble UP rechazado"
}
instalar "$ct131" "to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NOT NULL"

catalogo="$repo/data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json"
aprobacion=$(python3 - "$catalogo" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    print(json.load(f)['catalogo']['aprobacion_ref'])
PY
)
export GOCACHE="$temporal/go-cache" GOTMPDIR="$temporal/go-tmp" GOPROXY=off GOFLAGS=-mod=readonly
export VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL="host=$temporal/socket user=vec_plantillas_migrador_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_DSN="host=$temporal/socket user=vec_plantillas_ejecutor_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_CATALOGO="$catalogo"
python3 - "$repo" "$directorio/_preflight_overlay_test.go" "$temporal/overlay.json" \
  "${VEC_PLANTILLAS_PREFLIGHT_SOURCE:-}" "${VEC_PLANTILLAS_AUTORIDADES_SOURCE:-}" <<'PY'
import json, pathlib, sys
repo, source, output, candidate, authorities = sys.argv[1:]
virtual = str(pathlib.Path(repo) / 'internal/app/bootstrap/zz_rrhh_plantillas_pg18_test.go')
replacements = {virtual: source}
if candidate:
    if not pathlib.Path(candidate).is_file():
        raise SystemExit('VEC_PLANTILLAS_PREFLIGHT_SOURCE no es un fichero')
    original = str(pathlib.Path(repo) / 'internal/app/bootstrap/contratacion_temporal_plantillas_catalogo_preflight.go')
    replacements[original] = candidate
if authorities:
    if not pathlib.Path(authorities).is_file():
        raise SystemExit('VEC_PLANTILLAS_AUTORIDADES_SOURCE no es un fichero')
    original = str(pathlib.Path(repo) / 'internal/app/bootstrap/contratacion_temporal_plantillas_desarrollo.go')
    replacements[original] = authorities
pathlib.Path(output).write_text(json.dumps({'Replace': replacements}), encoding='utf-8')
PY
if [[ -n ${VEC_PLANTILLAS_PREFLIGHT_SOURCE:-} ]]; then
  echo "Preflight Go candidato externo: $(sha256sum "$VEC_PLANTILLAS_PREFLIGHT_SOURCE" | cut -d' ' -f1)"
fi
if [[ -n ${VEC_PLANTILLAS_AUTORIDADES_SOURCE:-} ]]; then
  echo "Autoridades Go candidato externo: $(sha256sum "$VEC_PLANTILLAS_AUTORIDADES_SOURCE" | cut -d' ' -f1)"
fi
preflight() {
  local etapa=$1
  (cd "$repo" && VEC_PLANTILLAS_PG18_STAGE="$etapa" go test -overlay="$temporal/overlay.json" \
    -count=1 -run '^TestPlantillasPG18PreflightAislado$' ./internal/app/bootstrap)
}
autoridades() {
  local etapa=$1
  (cd "$repo" && VEC_PLANTILLAS_PG18_AUTORIDADES_STAGE="$etapa" go test -overlay="$temporal/overlay.json" \
    -count=1 -run '^TestPlantillasPG18AutoridadesAisladas$' ./internal/app/bootstrap)
}
provisionar() { (cd "$repo" && go run ./cmd/vec-provision-plantillas -catalogo "$catalogo" -aprobacion-ref "$aprobacion"); }
if [[ $(consulta "SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1") != 0 ]]; then
  echo 'La preimagen sintética no está vacía' >&2; exit 1
fi
if runtime_scalar "SELECT vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(repeat('0',64),1,'paquete:ejemplo:vec:v1')" >"$temporal/preflight_falta.log" 2>&1; then
  echo 'Preflight aceptó catálogo sin provisión' >&2; exit 1
fi
grep -q 'CT-131: publicación inicial no provisionada o divergente' "$temporal/preflight_falta.log" || {
  echo 'Rechazo inesperado de preflight sin base' >&2; exit 1;
}
echo 'OK preflight de base ausente: 55000'
provisionar >"$temporal/recibo_1.json"
provisionar >"$temporal/recibo_2.json"
python3 - "$temporal/recibo_1.json" "$temporal/recibo_2.json" <<'PY'
import json, sys
a, b = [json.load(open(p, encoding='utf-8')) for p in sys.argv[1:]]
assert a['resultado'] == 'registrado' and b['resultado'] == 'replay', (a, b)
for key in ('recibo_ref', 'registrada_en', 'version', 'revision', 'catalogo_huella_sha256', 'contenido_json_sha256'):
    assert a[key] == b[key], key
print('OK CLI migrador: provisión y replay con el mismo recibo, fecha, versión y huellas')
PY
huella=$(python3 - "$temporal/recibo_1.json" <<'PY'
import json, sys
print(json.load(open(sys.argv[1], encoding='utf-8'))['catalogo_huella_sha256'])
PY
)
[[ $huella =~ ^[0-9a-f]{64}$ ]] || { echo 'Huella de recibo inválida' >&2; exit 1; }
[[ $(runtime_scalar "SELECT vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1('$huella',1,'paquete:ejemplo:vec:v1')") == t ]] || {
  echo 'Preflight rechazó base provisionada' >&2; exit 1;
}
if runtime_scalar "SELECT vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(repeat('0',64),1,'paquete:ejemplo:vec:v1')" >"$temporal/preflight_divergente.log" 2>&1; then
  echo 'Preflight aceptó huella divergente' >&2; exit 1
fi
grep -q 'CT-131: publicación inicial no provisionada o divergente' "$temporal/preflight_divergente.log" || {
  echo 'Rechazo inesperado de preflight divergente' >&2; exit 1;
}
echo 'OK preflight SQL CT131: base provisionada; ausente y divergente rechazadas'
preflight rechazo
echo 'OK preflight Go antes de AD3-99: cerrado'
[[ $(consulta 'SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1') == 1 ]] || {
  echo 'Historia de provisión duplicada' >&2; exit 1;
}
[[ $(consulta 'SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1') == 1 ]] || {
  echo 'Auditoría de provisión duplicada' >&2; exit 1;
}
[[ $(consulta "SELECT NOT has_table_privilege('vec_plantillas_ejecutor_ensayo','vec_contratacion_temporal.catalogo_plantillas_historia_v1','SELECT') AND NOT has_function_privilege('vec_plantillas_migrador_ensayo','vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || {
  echo 'ACL nominal de provisión/ejecución incompatible' >&2; exit 1;
}
echo 'OK LOGIN migrador y ejecutor separados; ejecutor sin SELECT de historia'

admin <"$directorio/ct133_doble.sql" >/dev/null
instalar "$ad399" "to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
preflight rechazo
echo 'OK preflight Go sin CT135: cerrado'
instalar "$ct135" "EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.catalogo_plantillas_historia_v1'::regclass AND attname='organizacion_ref' AND NOT attisdropped)"
preflight_ok=1
if preflight aceptado; then
  echo 'OK preflight Go CT131/CT135/AD3-99 y base provisionada'
else
  preflight_ok=0
  echo 'ROJO preflight Go con CT131/CT135/AD3-99; se continúan solo las sondas independientes' >&2
fi

admin <"$directorio/acl_cruces.sql" >/dev/null
if [[ $preflight_ok == 1 ]]; then
  preflight aceptado
  admin >/dev/null <<'SQL'
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) TO vec_contratacion_temporal_ejecutor;
SQL
  preflight rechazo
  admin >/dev/null <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) FROM vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
SQL
  preflight rechazo
  admin >/dev/null <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
SQL
  preflight aceptado
  echo 'OK preflight Go deniega concesiones cruzadas CT108 y CT132 (firmas sintéticas)'
else
  echo 'OMITIDO ACL cruzadas Go: el preflight base ya está rojo' >&2
fi

runtime <"$directorio/organizacion.sql" >/dev/null
[[ $(consulta "SELECT NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || {
  echo 'AD3-99 no sustituyó la ACL antigua' >&2; exit 1;
}
echo 'OK AD3-99: fachada antigua revocada y nueva confinada al propietario CT'
admin >/dev/null <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;
SQL
if [[ $preflight_ok == 1 ]]; then preflight rechazo; fi
admin >/dev/null <<'SQL'
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
SQL
if [[ $preflight_ok == 1 ]]; then
  preflight aceptado
  echo 'OK preflight Go falla cerrado si se revoca AD3-99 al propietario CT'
else
  echo 'OMITIDO preflight Go de revocación AD3-99: la base ya está roja' >&2
fi
if [[ $preflight_ok == 1 ]]; then
  admin >/dev/null <<'SQL'
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SECURITY INVOKER;
SQL
  preflight rechazo
  admin >/dev/null <<'SQL'
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SECURITY DEFINER;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO postgres;
SQL
  preflight rechazo
  admin >/dev/null <<'SQL'
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
SQL
  preflight aceptado
  echo 'OK preflight Go exige SECURITY DEFINER y propietario AD3 nominal'
fi
admin <"$directorio/autoridades_preflight.sql" >/dev/null
export VEC_PLANTILLAS_PG18_FUENTE_DSN="host=$temporal/socket user=vec_plantillas_fuente_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_MOTIVOS_DSN="host=$temporal/socket user=vec_plantillas_motivos_ensayo dbname=postgres sslmode=disable"
autoridades_ok=1
if autoridades aceptado; then
  echo 'OK preflight Go fuente y motivos con LOGIN separados'
else
  autoridades_ok=0
  echo 'ROJO preflight Go fuente/motivos con roles de fixture' >&2
fi
if docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U vec_plantillas_fuente_ensayo -d postgres \
  -c "SELECT to_regprocedure('vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" \
  >"$temporal/fuente_ct_oid.log" 2>&1; then
  echo 'Fuente sin USAGE CT pudo resolver la firma CT' >&2; exit 1
fi
grep -q 'permission denied for schema vec_contratacion_temporal' "$temporal/fuente_ct_oid.log" || {
  echo 'Rechazo inesperado al resolver CT desde fuente nominal' >&2; exit 1;
}
echo 'OK sonda fuente/motivos: to_regprocedure(CT) sin USAGE CT da 42501; revisar preflight de producto'
if [[ $autoridades_ok == 1 ]]; then
  admin >/dev/null <<'SQL'
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) TO vec_autorizacion_fuente;
SQL
  autoridades rechazo
  admin >/dev/null <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) FROM vec_autorizacion_fuente;
SQL
  autoridades aceptado
  echo 'OK fuente sin permiso CT: concesión cruzada CT108 rechazada'
fi
admin >/dev/null <<'SQL'
BEGIN;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;
DO $t$ BEGIN
 IF has_function_privilege('vec_contratacion_temporal_propietario',
  'vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'revocación AD3-99 ineficaz'; END IF;
END $t$;
ROLLBACK;
SQL
admin >/dev/null <<'SQL'
BEGIN;
REVOKE vec_contratacion_temporal_ejecutor FROM vec_plantillas_ejecutor_ensayo;
DO $t$ BEGIN
 IF pg_has_role('vec_plantillas_ejecutor_ensayo','vec_contratacion_temporal_ejecutor','USAGE')
 THEN RAISE EXCEPTION 'revocación ejecutor CT ineficaz'; END IF;
END $t$;
ROLLBACK;
SQL
echo 'OK revocaciones transaccionales de EXECUTE AD3-99 y pertenencia CT (ROLLBACK)'

docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do
  if docker exec "$nombre" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$nombre" pg_isready -q -U postgres -d postgres || { echo 'PG18 no volvió del reinicio' >&2; exit 1; }
provisionar >"$temporal/recibo_3.json"
python3 - "$temporal/recibo_1.json" "$temporal/recibo_3.json" <<'PY'
import json, sys
a, b = [json.load(open(p, encoding='utf-8')) for p in sys.argv[1:]]
assert b['resultado'] == 'replay'
for key in ('recibo_ref', 'registrada_en', 'version', 'revision', 'catalogo_huella_sha256', 'contenido_json_sha256'):
    assert a[key] == b[key], key
print('OK reinicio PG18: misma provisión y recibo')
PY
[[ $(consulta 'SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1') == 1 ]] || {
  echo 'Historia duplicada tras reinicio' >&2; exit 1;
}
[[ $(runtime_scalar "SELECT vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1('$huella',1,'paquete:ejemplo:vec:v1')") == t ]] || {
  echo 'Preflight rechazó base tras reinicio' >&2; exit 1;
}
echo 'ENSAYO SINTÉTICO ACOTADO COMPLETO. No acredita PDP/V3 real, CT132/134, TLS, HTTP ni navegador.'
if [[ $preflight_ok != 1 || $autoridades_ok != 1 ]]; then exit 1; fi
