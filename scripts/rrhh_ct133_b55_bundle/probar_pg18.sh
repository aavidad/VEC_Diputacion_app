#!/usr/bin/env bash
# RRHH 4.08 + 2.05: ensayos complementarios sobre bases efímeras, no E2E V3.
set -Eeuo pipefail

directorio=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$directorio" rev-parse --show-toplevel)
origen_b55=$repo
imagen=postgres:18.4-alpine
if [[ $# != 1 || $1 != --synthetic ]]; then
  echo 'Uso: scripts/rrhh_ct133_b55_bundle/probar_pg18.sh --synthetic' >&2
  echo 'El modo conjunto con núcleo V3 real requiere una preimagen de producto autorizada.' >&2
  exit 2
fi
for programa in docker go python3 sha256sum; do command -v "$programa" >/dev/null || exit 2; done
git -C "$repo" merge-base --is-ancestor bb67ac3ac6535cda5acd27a62677d9e09a531725 HEAD || {
  echo 'El candidato debe descender del ensamblado bb67ac3ac' >&2; exit 2;
}
docker info >/dev/null 2>&1 || { echo 'Docker no disponible' >&2; exit 2; }
temporal=$(mktemp -d /dev/shm/vec-rrhh-ct133-b55.XXXXXXXX)
nombre=vec-ct133-b55-${temporal##*.}
contenedor=
mkdir -p "$temporal/data" "$temporal/socket" "$temporal/go-cache" "$temporal/go-tmp"
chmod 1777 "$temporal/socket"
limpiar() {
  if [[ -n $contenedor ]]; then docker rm -f "$contenedor" >/dev/null 2>&1 || true; fi
  docker run --rm --network none --pull never -v "$temporal:/limpiar" --entrypoint /bin/sh "$imagen" \
    -c 'rm -rf /limpiar/data /limpiar/socket' >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT
contenedor=$(docker run -d --rm --pull never --network none --name "$nombre" \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$temporal/data:/var/lib/postgresql" -v "$temporal/socket:/var/run/postgresql" "$imagen")
for _ in $(seq 1 120); do
  if docker logs "$contenedor" 2>&1 | grep -q 'PostgreSQL init process complete' &&
     docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres || exit 1
admin() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
scalar() { docker exec "$contenedor" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
ejecutor() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_plantillas_ejecutor_ensayo -d postgres; }
[[ $(scalar 'SHOW server_version') == 18.4* ]] || exit 1
[[ -z $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$contenedor") ]] || exit 1

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
archivos=(
  "$ct/000131_catalogo_plantillas_documentos.up.sql"
  "$ct/000133_obtener_catalogo_plantillas_publicado_documental.up.sql"
  "$ad3/000099_ambito_organizacion_plantillas_ct.up.sql"
  "$ct/000135_ambito_organizacion_plantillas.up.sql"
  "$ad3/000100_documental_tres_ambitos_ct.up.sql"
  "$ct/000137_documental_tres_ambitos.up.sql"
  "$origen_b55/deploy/postgresql/autorizacion_atestada_v3/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql"
  "$origen_b55/deploy/postgresql/bolsa_llamamientos/migraciones/000055_lectura_reincorporacion_titular_v3.up.sql"
)
sha256sum "${archivos[@]}"
admin <"$directorio/preimagen_plantillas_sintetica.sql" >/dev/null
instalar() {
  local archivo=$1 sonda=$2
  [[ $(scalar "SELECT $sonda") == f ]] || { echo "Preimagen no vacía: $archivo" >&2; exit 1; }
  sed 's/^COMMIT;$/ROLLBACK;/' "$archivo" | admin >/dev/null
  [[ $(scalar "SELECT $sonda") == f ]] || { echo "ROLLBACK dejó objetos: $archivo" >&2; exit 1; }
  admin <"$archivo" >/dev/null
  [[ $(scalar "SELECT $sonda") == t ]] || { echo "UP no instaló: $archivo" >&2; exit 1; }
  if admin <"$archivo" >/dev/null 2>&1; then echo "Doble UP aceptado: $archivo" >&2; exit 1; fi
  echo "OK $(basename "$archivo"): ROLLBACK, UP, doble UP denegado"
}
instalar "${archivos[0]}" "to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NOT NULL"

# Copia temporal del catálogo de demostración. Se añade una clave configurada
# cuyo contenido es sintético; el árbol de producto permanece intacto.
python3 - "$repo/data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json" "$temporal/catalogo.json" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f: data=json.load(f)
entrada=next(x for x in data['catalogo']['entradas'] if x['clave']=='cese').copy()
entrada['clave']='prueba_tipo_rrhh'
data['catalogo']['entradas'].append(entrada)
with open(sys.argv[2], 'w', encoding='utf-8') as f: json.dump(data, f, ensure_ascii=False)
PY
aprobacion=$(python3 - "$temporal/catalogo.json" <<'PY'
import json,sys
print(json.load(open(sys.argv[1],encoding='utf-8'))['catalogo']['aprobacion_ref'])
PY
)
export GOCACHE="$temporal/go-cache" GOTMPDIR="$temporal/go-tmp" GOPROXY=off GOFLAGS=-mod=readonly
export VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL="host=$temporal/socket user=vec_plantillas_migrador_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_DSN="host=$temporal/socket user=vec_plantillas_ejecutor_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_CATALOGO="$temporal/catalogo.json"
docker exec "$contenedor" psql -X -At -v ON_ERROR_STOP=1 -U vec_plantillas_migrador_ensayo -d postgres -c 'SELECT session_user' >/dev/null
python3 - "$repo" "$directorio/_provision_overlay_test.go.txt" "$temporal/overlay-provision.json" <<'PY'
import json,pathlib,sys
repo,source,output=sys.argv[1:]
virtual=str(pathlib.Path(repo)/'cmd/vec-provision-plantillas/zz_rrhh_bundle_pg18_test.go')
pathlib.Path(output).write_text(json.dumps({'Replace':{virtual:source}}),encoding='utf-8')
PY
(cd "$repo" && go test -v -overlay="$temporal/overlay-provision.json" -count=1 -run '^TestProvisionPreflightPG18Bundle$' ./cmd/vec-provision-plantillas)
provisionar() { (cd "$repo" && go run ./cmd/vec-provision-plantillas -catalogo "$temporal/catalogo.json" -aprobacion-ref "$aprobacion"); }
provisionar >"$temporal/recibo1.json"
provisionar >"$temporal/recibo2.json"
python3 - "$temporal/recibo1.json" "$temporal/recibo2.json" <<'PY'
import json,sys
a,b=(json.load(open(x,encoding='utf-8')) for x in sys.argv[1:])
assert (a['resultado'],b['resultado'])==('registrado','replay')
assert all(a[k]==b[k] for k in ('recibo_ref','registrada_en','version','revision','catalogo_huella_sha256','contenido_json_sha256'))
print('OK provisión CLI y replay: mismo recibo, fecha, versión y huellas')
PY
conteo_catalogo=$(scalar "SELECT (SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1)||':'||(SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1)||':'||(SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1)")
[[ $conteo_catalogo == 1:1:0 ]] || { echo "Historia inicial inesperada: $conteo_catalogo" >&2; exit 1; }
instalar "${archivos[1]}" "to_regprocedure('vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
instalar "${archivos[2]}" "to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
instalar "${archivos[3]}" "EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.catalogo_plantillas_historia_v1'::regclass AND attname='organizacion_ref' AND NOT attisdropped)"
admin <"$directorio/autoridades_preflight_sinteticas.sql" >/dev/null
python3 - "$repo" "$directorio/_preflight_overlay_test.go.txt" "$temporal/overlay.json" <<'PY'
import json,pathlib,sys
repo,source,output=sys.argv[1:]
virtual=str(pathlib.Path(repo)/'internal/app/bootstrap/zz_rrhh_bundle_pg18_test.go')
pathlib.Path(output).write_text(json.dumps({'Replace':{virtual:source}}),encoding='utf-8')
PY
export VEC_PLANTILLAS_PG18_FUENTE_DSN="host=$temporal/socket user=vec_plantillas_fuente_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_MOTIVOS_DSN="host=$temporal/socket user=vec_plantillas_motivos_ensayo dbname=postgres sslmode=disable"
export VEC_PLANTILLAS_PG18_STAGE=aceptado VEC_PLANTILLAS_PG18_AUTORIDADES_STAGE=aceptado
(cd "$repo" && go test -overlay="$temporal/overlay.json" -count=1 -run '^TestPlantillasPG18(PreflightAislado|AutoridadesAisladas)$' ./internal/app/bootstrap)
echo 'OK preflight Go con LOGIN CT, fuente y motivos nominales (fachadas fuente/motivos sintéticas)'
[[ $(scalar "SELECT NOT has_schema_privilege('vec_plantillas_ejecutor_ensayo','vec_autorizacion_atestada_v3','USAGE') AND NOT has_table_privilege('vec_plantillas_ejecutor_ensayo','vec_contratacion_temporal.catalogo_plantillas_historia_v1','SELECT')") == t ]] || {
  echo 'ACL: ejecutor CT ve esquema AD3 o historia directa' >&2; exit 1;
}
admin >/dev/null <<'SQL'
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;
SQL
(cd "$repo" && VEC_PLANTILLAS_PG18_STAGE=rechazo go test -overlay="$temporal/overlay.json" -count=1 -run '^TestPlantillasPG18PreflightAislado$' ./internal/app/bootstrap)
admin >/dev/null <<'SQL'
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
REVOKE vec_autorizacion_fuente FROM vec_plantillas_fuente_ensayo;
SQL
(cd "$repo" && VEC_PLANTILLAS_PG18_AUTORIDADES_STAGE=rechazo go test -overlay="$temporal/overlay.json" -count=1 -run '^TestPlantillasPG18AutoridadesAisladas$' ./internal/app/bootstrap)
admin >/dev/null <<'SQL'
GRANT vec_autorizacion_fuente TO vec_plantillas_fuente_ensayo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SQL
echo 'OK preflight: revocación CT/AD3 y fuente separada denegadas; ACL directa cerrada'
admin >/dev/null <<'SQL'
SET ROLE vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.expediente_version_integral(expediente_ref text,version bigint,agregado_json jsonb);
INSERT INTO vec_contratacion_temporal.expediente_version_integral VALUES
 ('expediente:ensayo-ct137',1,'{"organizacion_ref":"organizacion:desarrollo:dipgra"}');
RESET ROLE;
SQL
instalar "${archivos[4]}" "to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
instalar "${archivos[5]}" "to_regprocedure('vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
[[ $(scalar "SELECT NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || exit 1
ejecutor <"$directorio/documental_sintetico.sql" >/dev/null

docker restart "$contenedor" >/dev/null
for _ in $(seq 1 120); do docker exec "$contenedor" pg_isready -q -U postgres -d postgres && break; sleep 0.5; done
provisionar >"$temporal/recibo3.json"
python3 - "$temporal/recibo1.json" "$temporal/recibo3.json" <<'PY'
import json,sys
a,b=(json.load(open(x,encoding='utf-8')) for x in sys.argv[1:])
assert b['resultado']=='replay'
assert all(a[k]==b[k] for k in ('recibo_ref','registrada_en','version','revision','catalogo_huella_sha256','contenido_json_sha256'))
print('OK reinicio PG18: mismo recibo y catálogo')
PY
[[ $(scalar "SELECT (SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1)||':'||(SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1)||':'||(SELECT count(*) FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1)") == "$conteo_catalogo" ]] || {
  echo 'Reinicio/provisión alteró historia, auditoría u outbox' >&2; exit 1;
}
echo "OK historia CT tras reinicio: $conteo_catalogo (historia:auditoría:outbox)"
ejecutor <"$directorio/documental_sintetico.sql" >/dev/null
echo 'CT133/CT137/AD3-100: ensayo sintético SQL y Go completo.'

# AD3-101 requiere una definición exacta del núcleo V3 AD3-97 que no ofrece
# este fixture CT. Sus dos sondas oficiales corren en bases efímeras separadas.
(cd "$origen_b55" && VEC_POSTGRES_TEST_IMAGE="$imagen" bash "$directorio/ad3_101_focal_seguro.sh")
(cd "$origen_b55" && VEC_POSTGRES_TEST_IMAGE="$imagen" bash "$directorio/b55_focal_seguro.sh")
echo 'B55/AD3-101: ensayos SQL complementarios terminados; sin criptografía ni HTTP.'
