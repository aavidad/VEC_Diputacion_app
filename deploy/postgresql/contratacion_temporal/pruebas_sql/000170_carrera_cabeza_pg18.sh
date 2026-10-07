#!/usr/bin/env bash
# Carrera CT170 sobre un clon PostgreSQL 18 desechable, con datos sintéticos.
# Uso:
#   000170_carrera_cabeza_pg18.sh --clone-archive ESTADO.tgz \
#     --bootstrap-list LISTA.txt [--bootstrap-root RAIZ]
# LISTA contiene, en orden, CT145, stub de AD125, CT152, stubs de AD156,
# AD157, AD159 y AUT30, y CT170. Las rutas pueden ser relativas a RAIZ o al
# directorio de LISTA. Son dependencias explícitas: este archivo no instala
# ninguna SQL sobre la principal. Los stubs solo acreditan el CAS/trigger CT;
# no prueban criptografía V3 ni competencia real de AUT30.
set -Eeuo pipefail

raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4}
archivo=
lista=
while (($#)); do
  case "$1" in
    --clone-archive) archivo=${2-}; shift 2 ;;
    --bootstrap-list) lista=${2-}; shift 2 ;;
    --bootstrap-root) raiz=${2-}; shift 2 ;;
    *) printf 'Argumento desconocido: %s\n' "$1" >&2; exit 2 ;;
  esac
done
[[ -f $archivo && -f $lista && -d $raiz ]] || {
  printf 'Se requieren --clone-archive, --bootstrap-list y una --bootstrap-root válida\n' >&2
  exit 2
}
docker image inspect "$imagen" >/dev/null
scratch=$(mktemp -d /dev/shm/vec-ct170-carrera-XXXXXX)
chmod 700 "$scratch"
contenedor="vec-ct170-carrera-$$"
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run --rm --network none --user 0 -v "$scratch:/d" --entrypoint rm "$imagen" -rf /d >/dev/null 2>&1 || true
  rmdir "$scratch" >/dev/null 2>&1 || true
}
trap limpiar EXIT

docker run --rm --network none --user 0 -v "$scratch:/d" \
  -v "$(dirname -- "$archivo"):/o:ro" --entrypoint tar "$imagen" \
  -C /d -xzf "/o/$(basename -- "$archivo")"
docker run -d --rm --network none --name "$contenedor" \
  -v "$scratch:/var/lib/postgresql" "$imagen" >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super() { docker exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }

lista_dir=$(dirname -- "$lista")
while IFS= read -r sql || [[ -n $sql ]]; do
  [[ -z $sql || $sql == \#* ]] && continue
  if [[ -f $raiz/$sql ]]; then
    ruta=$raiz/$sql
  elif [[ -f $lista_dir/$sql ]]; then
    ruta=$lista_dir/$sql
  else
    printf 'SQL de bootstrap ausente: %s\n' "$sql" >&2
    exit 2
  fi
  printf 'Bootstrap: %s\n' "$sql"
  psql_super -f /dev/stdin < "$ruta" >/dev/null
done < "$lista"

psql_super > "$scratch/fixture.out" <<'SQL'
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer / 10000 <> 18
    OR to_regclass('vec_contratacion_temporal.firma_historia_cabeza_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
      AND tgname='cabeza_global_ai' AND NOT tgisinternal) THEN
   RAISE EXCEPTION 'CT170 o trigger no instalados en PostgreSQL 18 desechable';
 END IF;
END $pre$;
CREATE ROLE vec_ct170_carrera_login LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
CREATE SCHEMA prueba_ct170_carrera;
GRANT USAGE ON SCHEMA prueba_ct170_carrera TO vec_ct170_carrera_login;

-- El doble sustituye exclusivamente AD157 dentro del contenedor aislado.
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_vec_ct_v3_atestada(
 p_solicitud text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:ct170:carrera',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'auditoria:ct170:carrera:'||gen_random_uuid()::text,clock_timestamp(),true;
END $f$;
RESET ROLE;

CREATE TABLE prueba_ct170_carrera.base AS
SELECT a.expediente_ref,a.version,v.agregado_json->>'organizacion_ref' AS organizacion_ref,
 coalesce(h.revision,0) AS revision_inicial,
 coalesce(h.huella_sha256,encode(sha256(convert_to('vec:ct:firma-historia:v1:[]','UTF8')),'hex')) AS huella_inicial
FROM vec_contratacion_temporal.expediente_integral_actual a
JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
LEFT JOIN vec_contratacion_temporal.firma_historia_cabeza_v1 h
 ON h.expediente_ref=a.expediente_ref AND h.organizacion_ref=v.agregado_json->>'organizacion_ref'
WHERE v.agregado_json->>'organizacion_ref' IS NOT NULL
 AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
   WHERE f.expediente_ref=a.expediente_ref AND f.documento IN ('ct170_carrera_a','ct170_carrera_b'))
ORDER BY a.expediente_ref LIMIT 1;
DO $pre$
BEGIN
 IF (SELECT count(*) FROM prueba_ct170_carrera.base) <> 1 THEN
   RAISE EXCEPTION 'Falta expediente sintético apto en clon desechable'; END IF;
END $pre$;
CREATE TABLE prueba_ct170_carrera.material (
 nombre text PRIMARY KEY, solicitud text NOT NULL, decision bytea NOT NULL
);
WITH s AS (
 SELECT nombre,jsonb_build_object(
  'Via','certificado_vec','OrganizacionRef',b.organizacion_ref,
  'ExpedienteRef',b.expediente_ref,'VersionExpediente',b.version,
  'Documento','ct170_carrera_'||nombre,
  'CatalogoRef','vec.contratacion_temporal.circuito_firma:1','CatalogoHuella',repeat('a',64),
  'PasoRef','vec.contratacion_temporal.circuito_firma:1:ct170_carrera_'||nombre||'.p1',
  'PasoOrden',1,'Secuencia',1,'HistoriaRevision',b.revision_inicial,
  'HistoriaHuella',b.huella_inicial,'OriginalRef','documento:ct170:carrera:original:'||nombre,
  'OriginalVersion',1,'OriginalHuella',repeat('b',64),'FirmadoHuella',repeat('c',64),
  'CertificadoHuella',repeat('d',64),'FirmanteRef','ref:'||repeat('d',64),
  'FirmantePrincipalRef','per_ct170_firmante_prueba','PerfilFirmanteRef','prf_ct170_firmante_prueba',
  'CargoFirmante','Jefatura de prueba','UnidadFirmanteRef','unidad:ct170:prueba',
  'PerfilActivoFirmanteRef','prf_ct170_firmante_prueba',
  'PuestoFirmanteRef',NULL,'AmbitoFirmanteRef',NULL,
  'AsignacionFirmanteRef','asignacion:ct170:prueba','AsignacionFirmanteVersion',1,
  'AsignacionFirmanteHuella',repeat('e',64),'VersionRolFirmanteRef','rol:ct170:firmante:prueba',
  'VersionRolFirmanteHuella',repeat('f',64),'ControlVigenciaFirmanteRef','rol:ct170:firmante:prueba',
  'ControlVigenciaFirmanteRevision',1,'ControlVigenciaFirmanteHuella',repeat('1',64),
  'AsignacionVigenteDesde','2026-01-01T00:00:00Z','AsignacionVigenteHasta','2027-01-01T00:00:00Z',
  'ActoCompetenciaRef',NULL,'DelegacionRef',NULL,
  'PoliticaVerificacion','politica:vec:firma:verificacion-autonoma:v1',
  'RevocacionEstado','vigente','SelloTiempoEstado','no_presente',
  'ClaveIdempotencia','clave-ct170-carrera-'||nombre||'-000001',
  'DocumentoCustodiaRef','documento:ct170:carrera:firmado:'||nombre,
  'DocumentoCustodiaVersion',1)::text AS solicitud
 FROM prueba_ct170_carrera.base b CROSS JOIN (VALUES ('a'),('b')) t(nombre)
)
INSERT INTO prueba_ct170_carrera.material
SELECT nombre,solicitud,convert_to(jsonb_build_object(
 'accion','contratacion_temporal.documento.firma_vec.registrar',
 'modulo_id','contratacion_temporal','tipo_recurso','firma_vec_documento_contratacion_temporal',
 'finalidad','gestionar_contratacion_temporal',
 'recurso_ref','operacion-firma-vec-ct:'||(solicitud::jsonb->>'ClaveIdempotencia'),
 'contexto_recurso_huella_sha256',encode(sha256(convert_to(
  '{"ambitos":{"organizacion_ref":"'||(solicitud::jsonb->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
 'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
 'principal_id','per_ct170_firmante_prueba','perfil_activo_ref','prf_ct170_firmante_prueba',
 'version_rol_ref','rol:ct170:firmante:prueba')::text,'UTF8') FROM s;
GRANT SELECT ON prueba_ct170_carrera.material TO vec_ct170_carrera_login;
CREATE FUNCTION prueba_ct170_carrera.cabeza() RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT coalesce(h.revision,b.revision_inicial)::text||':'||coalesce(h.huella_sha256,b.huella_inicial)
 FROM prueba_ct170_carrera.base b LEFT JOIN vec_contratacion_temporal.firma_historia_cabeza_v1 h
 ON h.organizacion_ref=b.organizacion_ref AND h.expediente_ref=b.expediente_ref
$f$;
CREATE FUNCTION prueba_ct170_carrera.registrar(p_nombre text) RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path=pg_catalog AS $f$
 SELECT vec_contratacion_temporal.registrar_firma_verificada_v1(
  m.solicitud,'\x',m.decision,'\x','\x',1,1,'\x','\x','\x','\x')
 FROM prueba_ct170_carrera.material m WHERE m.nombre=p_nombre
$f$;
REVOKE ALL ON FUNCTION prueba_ct170_carrera.cabeza() FROM PUBLIC;
REVOKE ALL ON FUNCTION prueba_ct170_carrera.registrar(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION prueba_ct170_carrera.cabeza() TO vec_ct170_carrera_login;
GRANT EXECUTE ON FUNCTION prueba_ct170_carrera.registrar(text) TO vec_ct170_carrera_login;
SQL

mkfifo "$scratch/a.in" "$scratch/b.in"
docker exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
  -U vec_ct170_carrera_login -d postgres < "$scratch/a.in" > "$scratch/a.out" 2> "$scratch/a.err" &
pid_a=$!
docker exec -i "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
  -U vec_ct170_carrera_login -d postgres < "$scratch/b.in" > "$scratch/b.out" 2> "$scratch/b.err" &
pid_b=$!
exec 3> "$scratch/a.in"
exec 4> "$scratch/b.in"
esperar_marca() {
  local archivo_salida=$1 marca=$2
  for _ in $(seq 1 100); do
    if rg -q "^${marca}$" "$archivo_salida"; then return 0; fi
    sleep 0.1
  done
  printf 'No llegó %s de sesión de prueba\n' "$marca" >&2
  return 1
}
printf '%s\n' 'BEGIN ISOLATION LEVEL SERIALIZABLE;' "SET LOCAL statement_timeout='15s';" \
 "SELECT prueba_ct170_carrera.cabeza();" '\echo lista-a' >&3
printf '%s\n' 'BEGIN ISOLATION LEVEL SERIALIZABLE;' "SET LOCAL statement_timeout='15s';" \
 "SELECT prueba_ct170_carrera.cabeza();" '\echo lista-b' >&4
esperar_marca "$scratch/a.out" lista-a
esperar_marca "$scratch/b.out" lista-b
cabeza_a=$(head -n 1 "$scratch/a.out")
cabeza_b=$(head -n 1 "$scratch/b.out")
[[ $cabeza_a == "$cabeza_b" && $cabeza_a =~ ^[0-9]+:[0-9a-f]{64}$ ]] || {
  printf 'Las sesiones no observaron la misma cabeza inicial\n' >&2; exit 1;
}

printf '%s\n' "SELECT prueba_ct170_carrera.registrar('a');" 'COMMIT;' '\echo confirmada-a' '\q' >&3
exec 3>&-
wait "$pid_a"
esperar_marca "$scratch/a.out" confirmada-a
printf '%s\n' "SELECT prueba_ct170_carrera.registrar('b');" 'COMMIT;' >&4
exec 4>&-
if wait "$pid_b"; then
  printf 'La segunda firma también confirmó: CAS global roto\n' >&2
  exit 1
fi
if ! rg -q 'ERROR:  (P1701|40001|40P01):' "$scratch/b.err"; then
  printf 'Fallo distinto de CAS/serialización en segunda sesión:\n' >&2
  sed -n '1,8p' "$scratch/b.err" >&2
  exit 1
fi

psql_super <<'SQL' >/dev/null
DO $verificar$
DECLARE b record; f record; h record; base_count bigint;
BEGIN
 SELECT * INTO STRICT b FROM prueba_ct170_carrera.base;
 SELECT count(*) INTO base_count FROM vec_contratacion_temporal.firma_documento_v1
  WHERE expediente_ref=b.expediente_ref AND documento IN ('ct170_carrera_a','ct170_carrera_b');
 IF base_count<>1 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1
   WHERE expediente_ref=b.expediente_ref AND documento='ct170_carrera_b') THEN
   RAISE EXCEPTION 'La carrera dejó dos firmas o la equivocada'; END IF;
 SELECT * INTO STRICT f FROM vec_contratacion_temporal.firma_documento_v1
  WHERE expediente_ref=b.expediente_ref AND documento='ct170_carrera_a';
 SELECT * INTO STRICT h FROM vec_contratacion_temporal.firma_historia_cabeza_v1
  WHERE organizacion_ref=b.organizacion_ref AND expediente_ref=b.expediente_ref;
 IF f.historia_revision_observada<>b.revision_inicial
    OR f.historia_huella_observada<>b.huella_inicial
    OR h.revision<>b.revision_inicial+1
    OR h.huella_sha256 IS DISTINCT FROM vec_contratacion_temporal.nueva_huella_firma_historia_v1(
       b.huella_inicial,f.firma_ref,f.recibo_ref,f.documento,f.secuencia,f.solicitud_huella_sha256)
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1 WHERE firma_ref=f.firma_ref)<>1
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1 WHERE firma_ref=f.firma_ref)<>1
    OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1 WHERE firma_ref=f.firma_ref)<>1 THEN
   RAISE EXCEPTION 'Efecto, cabeza, custodia, auditoría u outbox inconsistentes'; END IF;
END $verificar$;
SQL
printf 'CT170 PG18: dos sesiones SERIALIZABLE, misma cabeza inicial, una firma y una revisión confirmadas; segunda rechazada (%s).\n' \
  "$(rg -o 'ERROR:  (P1701|40001|40P01):' "$scratch/b.err" | head -n 1 | cut -d ' ' -f 3 | tr -d :)"
