#!/usr/bin/env bash
# B47 con estructura real de Bolsa hasta B28 y doble aislado del consumidor
# AD3-93. AD3-93 real se prueba por separado con el núcleo completo.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-b47-$$
datos=/dev/shm/vec-pg-b47-$$
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm --pull never -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true
    rmdir "$datos" 2>/dev/null || true
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$contenedor" \
  -v "$datos:/var/lib/postgresql" -v "$repo:/repo:ro" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do
  docker exec "$contenedor" pg_isready -q -U postgres 2>/dev/null && break
  sleep 0.5
done
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
fichero() { psql_pg -f "/repo/$1"; }
for ruta in \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/bolsa_llamamientos/roles_up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql; do
  fichero "$ruta" >/dev/null
done
fichero deploy/postgresql/bolsa_llamamientos/pruebas_sql/disposicion_oferta/dobles_ad3.sql >/dev/null
psql_pg >/dev/null <<'SQL'
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
 repeat('a',64),repeat('b',64),'auditoria:'||gen_random_uuid(),clock_timestamp(),true
$f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_propietario;
SQL
m=deploy/postgresql/bolsa_llamamientos/migraciones
for f in "$repo"/$m/0000{01,02,03,04,05,06,07,08,10,11,12,13,14,16,17,18,19,20,21,22,23,24,25,26,28}_*.up.sql; do
  psql_pg < "$f" >/dev/null
  if [[ $(basename "$f") == 000010_* ]]; then fichero deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql >/dev/null; fi
done
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m/000047_politica_ofertas_ejemplo.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -tAc "SELECT to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL") == t ]] || { echo 'B47 ROLLBACK dejó objetos' >&2; exit 1; }
fichero "$m/000047_politica_ofertas_ejemplo.up.sql" >/dev/null
if fichero "$m/000047_politica_ofertas_ejemplo.up.sql" >/dev/null 2>&1; then echo 'B47 doble UP aceptado' >&2; exit 1; fi
psql_pg >/dev/null <<'SQL'
DO $acl$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.leer_politica_ofertas_v1(text)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.politica_ofertas_version','SELECT,INSERT,UPDATE,DELETE')
    OR has_function_privilege('public','vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'B47 ACL incorrecta'; END IF;
END $acl$;
SET session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:of:1',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion VALUES
 ('acta:of:1','bolsa:of:1',1,encode(sha256('{}'::bytea),'hex'),'inst:of:1',1,repeat('c',64),'categoria:rpt:aux','per_actoractoractoractoractor',now()-interval '10 days',now()-interval '10 days');
SET session_replication_role=origin;
SQL
psql_pg >/dev/null <<'SQL'
DO $prueba$
DECLARE
 p jsonb:='{"plazo":{"unidad":"dias_habiles","cantidad":2,"computo":"administrativo","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}';
 cap bytea:=convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 dec bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.publicar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gobierno_politica_ofertas_bolsa","recurso_ref":"bolsa:of:1"}','UTF8');
 capof bytea:=convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 decof bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:of:1"}','UTF8');
 r record; q record; v jsonb; plazo jsonb; datos jsonb:='{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}';
BEGIN
 IF (vec_bolsa_llamamientos.leer_politica_ofertas_v1('bolsa:of:1')->>'version')::int<>0 THEN RAISE EXCEPTION 'política ausente distinta de 0'; END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:of:1',0,p,'per_actoractoractoractoractor','clave-0001','recibo:politica-ofertas:'||repeat('a',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.politica->>'version'<>'1' OR r.politica->>'ejemplo'<>'true' THEN RAISE EXCEPTION 'publicación B47 incorrecta %',r.politica; END IF;
 SELECT * INTO q FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:of:1',0,p,'per_actoractoractoractoractor','clave-0001','recibo:politica-ofertas:'||repeat('a',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT q.reutilizada OR q.politica->>'recibo_ref' IS DISTINCT FROM r.politica->>'recibo_ref' THEN RAISE EXCEPTION 'replay B47 distinto'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:of:1',0,p,'per_actoractoractoractoractor','clave-0002','recibo:politica-ofertas:'||repeat('b',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'versión obsoleta aceptada';
 EXCEPTION WHEN SQLSTATE 'VBP01' THEN NULL; END;
 v:=r.politica;
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:bolsa:of:1:1','huella_catalogo',v->>'huella_sha256',
  'unidad','dias_habiles','cantidad',2,'computo','administrativo','municipio_sede','18087',
  'ultimo_dia','2026-09-29','ejemplo',true,'politica_version',1,'calendarios',jsonb_build_array('cal:1'));
 SELECT * INTO q FROM vec_bolsa_llamamientos.publicar_oferta_v1('oferta:'||repeat('1',64),'recibo:oferta:'||repeat('1',64),
  'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-0001',datos,plazo,clock_timestamp(),clock_timestamp()+interval '1 day',
  capof,decof,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF q.reutilizada OR q.oferta->'plazo'->>'politica_version'<>'1' THEN RAISE EXCEPTION 'oferta sin versión %',q.oferta; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v1('oferta:'||repeat('2',64),'recibo:oferta:'||repeat('2',64),
   'bolsa:of:1','per_actoractoractoractoractor','clave-oferta-0002',datos,plazo||'{"politica_version":9}'::jsonb,clock_timestamp(),clock_timestamp()+interval '1 day',
   capof,decof,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'oferta con versión falsa aceptada';
 EXCEPTION WHEN SQLSTATE 'VBP02' THEN NULL; END;
END $prueba$;
SQL
if fichero "$m/000047_politica_ofertas_ejemplo.down.sql" >/dev/null 2>&1; then echo 'B47 DOWN con historia aceptado' >&2; exit 1; fi
[[ $(psql_pg -tAc "SELECT count(*) FROM vec_bolsa_llamamientos.politica_ofertas_version") == 1 ]] || { echo 'B47 historia perdida' >&2; exit 1; }
echo 'PG18 B47: ROLLBACK, UP, ACL, versión, replay, oferta ligada y DOWN con historia OK (AD3 doble)'
