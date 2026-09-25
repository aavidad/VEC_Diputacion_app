#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../../../.."
name=vec-f1-probe-$$
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
pw=$(od -An -N18 -tx1 /dev/urandom | tr -d '[:space:]')
image=postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
docker run --detach --rm --name "$name" -e POSTGRES_PASSWORD="$pw" -e POSTGRES_DB=vec_f1_probe "$image" >/dev/null
for _ in $(seq 1 100); do
  if docker exec "$name" psql -XAt -U postgres -d vec_f1_probe -c 'select 1' 2>/dev/null | rg -q '^1$'; then break; fi
  sleep 0.2
done
psql_admin() { docker exec -i "$name" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d vec_f1_probe; }
psql_admin <<'SQL'
DO $x$
BEGIN
  CREATE ROLE vec_f1_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO vec_f1_dueno_base', current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END $x$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
for f in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/roles_historicos_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000005_lectura_contexto_historico_v2.up.sql \
  deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql; do
  psql_admin < "$f"
done
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES ('prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones SELECT cuenta_ref,2,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada',estado,vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE version=1;
UPDATE vec_contexto_actor_v1.proyeccion_cuenta_actual SET version=2;
INSERT INTO vec_contexto_actor_v1.persona_versiones SELECT persona_ref,2,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada',estado,vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.persona_versiones WHERE version=1;
UPDATE vec_contexto_actor_v1.persona_actual SET version=2;
INSERT INTO vec_contexto_actor_v1.perfil_versiones SELECT perfil_ref,2,persona_ref,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada',estado,vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.perfil_versiones WHERE version=1;
UPDATE vec_contexto_actor_v1.perfil_actual SET version=2;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones SELECT vinculo_ref,2,cuenta_ref,perfil_ref,persona_ref,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada',estado,vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE version=1;
UPDATE vec_contexto_actor_v1.vinculo_contexto_actual SET version=2;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones SELECT vinculo_ref,2,persona_ref,tipo,referencia,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada',estado,vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE version=1;
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=2;
COMMIT;
CREATE ROLE vec_f1_probe_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO vec_f1_probe_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
probe() { docker exec "$name" psql -XAt -v ON_ERROR_STOP=1 -U vec_f1_probe_runtime -d vec_f1_probe -c "$1"; }
probe "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa','rca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc','certificado','alto',clock_timestamp()); COMMIT;"
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones SELECT vinculo_ref,3,persona_ref,tipo,referencia,'prc_maestra_f1_probe_000001',1,repeat('a',64),'autoridad_maestra_acreditada','revocado',vigente_desde,vigente_hasta FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE version=2 AND tipo='candidato';
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=3 WHERE vinculo_ref='vin_sintetico_eeeeeeeeeeeeeeeeeeeeeeee';
COMMIT;
SELECT v.tipo,v.estado,v.persona_ref,a.version FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version) ORDER BY v.tipo;
SQL
# Centinela de la preimagen: los dos punteros pertenecen a la misma persona;
# únicamente candidato está revocado y el perfil activo es exacto.
psql_admin <<'SQL'
DO $x$
DECLARE c record; e record; p record;
BEGIN
  SELECT v.* INTO STRICT c FROM vec_contexto_actor_v1.vinculo_referencia_actual a
  JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
  WHERE v.tipo='candidato';
  SELECT v.* INTO STRICT e FROM vec_contexto_actor_v1.vinculo_referencia_actual a
  JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
  WHERE v.tipo='empleado';
  SELECT v.* INTO STRICT p FROM vec_contexto_actor_v1.perfil_actual a
  JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version);
  IF c.estado <> 'revocado' OR c.version <> 3 OR e.estado <> 'activo'
     OR e.version <> 2 OR c.persona_ref <> e.persona_ref
     OR p.persona_ref <> e.persona_ref
     OR p.perfil_ref <> 'prf_sintetico_cccccccccccccccccccccccc' THEN
    RAISE EXCEPTION 'preimagen candidata/empleado/perfil divergente';
  END IF;
END $x$;
SQL
set +e
fallo_preimagen=$(probe "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb','rca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc','certificado','alto',clock_timestamp()); COMMIT;" 2>&1)
preimagen_codigo=$?
set -e
if [[ $preimagen_codigo -eq 0 || $fallo_preimagen != *'referencias de contexto actor V2 no vigentes'* ]]; then
  echo 'la preimagen no reprodujo el bloqueo P0002' >&2
  exit 1
fi
up_f1=deploy/postgresql/contexto_actor_v1/migraciones/000006_vinculos_efectivos_temporales.up.sql
rechazar_preimagen() {
  local descripcion=$1 esperado=$2 salida codigo
  set +e
  salida=$(psql_admin < "$up_f1" 2>&1)
  codigo=$?
  set -e
  if [[ $codigo -eq 0 || $salida != *"$esperado"* ]]; then
    echo "000006 aceptó preimagen inválida: $descripcion" >&2
    exit 1
  fi
}
rechazar_preimagen 'rol de Autorización ausente' 'falta postimagen V2 historica de ContextoActor'
psql_admin <<'SQL'
CREATE ROLE vec_autorizacion_propietario NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
SQL
rechazar_preimagen 'EXECUTE de acreditación ausente' 'preimagen owner/configuracion/ACL ContextoActor divergente'
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,
  text,text,timestamptz,timestamptz) TO vec_autorizacion_propietario;
COMMIT;
SQL
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
REVOKE EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
  text,text,text,text,text,text,timestamptz) FROM vec_contexto_actor_v1_runtime;
COMMIT;
SQL
rechazar_preimagen 'EXECUTE de resolución ausente' 'preimagen owner/configuracion/ACL ContextoActor divergente'
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
  text,text,text,text,text,text,timestamptz) TO vec_contexto_actor_v1_runtime;
COMMIT;
SQL
psql_admin < "$up_f1"
set +e
probe "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb','rca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc','certificado','alto',clock_timestamp()); COMMIT;"
code=$?
set -e
[[ $code -eq 0 ]] || { echo 'resolución nueva con empleado efectivo rechazada' >&2; exit 1; }
psql_admin <<'SQL'
DO $x$
DECLARE r record; d jsonb; m jsonb;
BEGIN
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
  WHERE operacion_ref='oca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb';
  d := convert_from(r.representacion_canonica,'UTF8')::jsonb;
  m := convert_from(r.manifiesto_procedencia_canonico,'UTF8')::jsonb;
  IF d->>'persona_ref' <> 'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb'
     OR d->>'perfil_activo_ref' <> 'prf_sintetico_cccccccccccccccccccccccc'
     OR jsonb_array_length(d->'vinculos') <> 1
     OR d->'vinculos'->0->>'tipo' <> 'empleado'
     OR d->'vinculos'->0->>'referencia' <> 'emp_sintetico_hhhhhhhhhhhhhhhhhhhhhhhh'
     OR jsonb_array_length(m->'vinculos') <> 1
     OR m->'vinculos'->0->>'tipo' <> 'empleado' THEN
    RAISE EXCEPTION 'canon/manifiesto no conservan empleado efectivo exacto';
  END IF;
END $x$;
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $x$
DECLARE r record; acreditada timestamptz; emitida timestamptz := clock_timestamp();
BEGIN
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
  WHERE operacion_ref='oca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb';
  acreditada := vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,
    r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,
    r.cuenta_ref,2,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
    r.perfil_ref,2,'vca_sintetico_dddddddddddddddddddddddd',2,
    r.metodo,r.garantia,emitida,emitida+interval '1 minute');
  IF acreditada IS NULL THEN RAISE EXCEPTION 'nuevo recibo efectivo no acreditado'; END IF;
END $x$;
COMMIT;
SQL
printf 'POST_CODE=%s\n' "$code"
original=$(docker exec "$name" psql -XAt -v ON_ERROR_STOP=1 -U postgres -d vec_f1_probe -c "SELECT solicitado_en FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref='oca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa'")
printf 'REPLAY_ORIGINAL=%s\n' "$original"
set +e
probe "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa','rca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc','certificado','alto','$original'::timestamptz); COMMIT;"
replay_code=$?
set -e
[[ $replay_code -eq 0 ]] || { echo 'replay histórico dejó de ser idempotente' >&2; exit 1; }
printf 'REPLAY_CODE=%s\n' "$replay_code"
psql_admin <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $x$
DECLARE r record; emitida timestamptz := clock_timestamp();
BEGIN
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
  WHERE operacion_ref='oca_f1_pre_aaaaaaaaaaaaaaaaaaaaaaaa';
  IF vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,
    r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,
    r.cuenta_ref,2,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
    r.perfil_ref,2,'vca_sintetico_dddddddddddddddddddddddd',2,
    r.metodo,r.garantia,emitida,emitida+interval '1 minute') IS NOT NULL THEN
    RAISE EXCEPTION 'recibo anterior reutilizado tras revocar candidato';
  END IF;
END $x$;
COMMIT;
SQL

# Una ausencia de vinculo empleado tras revocación propia no concede permiso.
psql_admin <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones
SELECT vinculo_ref,3,persona_ref,tipo,referencia,'prc_maestra_f1_probe_000001',1,
       repeat('a',64),'autoridad_maestra_acreditada','revocado',vigente_desde,vigente_hasta
FROM vec_contexto_actor_v1.vinculo_referencia_versiones WHERE version=2 AND tipo='empleado';
UPDATE vec_contexto_actor_v1.vinculo_referencia_actual SET version=3
WHERE vinculo_ref='vin_sintetico_gggggggggggggggggggggggg';
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $x$
DECLARE r record; emitida timestamptz := clock_timestamp();
BEGIN
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
  WHERE operacion_ref='oca_f1_post_bbbbbbbbbbbbbbbbbbbbbbbb';
  IF vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,
    r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,
    r.cuenta_ref,2,'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',2,
    r.perfil_ref,2,'vca_sintetico_dddddddddddddddddddddddd',2,
    r.metodo,r.garantia,emitida,emitida+interval '1 minute') IS NOT NULL THEN
    RAISE EXCEPTION 'recibo empleado se reutilizó después de revocación';
  END IF;
END $x$;
COMMIT;
SQL
printf '%s\n' 'ContextoActor F1 temporal PG18: empleado efectivo, canon y revocaciones comprobados'
