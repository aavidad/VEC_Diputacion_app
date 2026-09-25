#!/usr/bin/env bash
# Ensayo de las migraciones de Bolsa corregidas tras la revisión de la
# integración (000028, 000030, 000032, 000037) en PostgreSQL 18 efímero.
# Instala la cadena real de Bolsa con dobles de los consumidores AD3 (solo
# firmas y retorno: la autorización real se ensaya en su propio esquema) y
# comprueba: ROLLBACK, UP/DOWN/UP de 000032 y 000037, dobles aplicaciones
# rechazadas, DOWN de 000032 rechazado mientras 000037 está instalada, 000037
# rechazada sin 000032, la versión inicial de la política, ACL con roles
# reales y las pruebas funcionales b2 (política), bof (ofertas: decisión de
# Bolsa, finalidad y recurso), b30 (portal, incluido el replay con decisión
# viva) y b37 (efectos de sanciones y readmisión). Los datos viven en
# /dev/shm/vec-pg-revision-bolsa-<pid> (montado con -v, sin volúmenes anónimos) y se
# borran al terminar.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-revision-bolsa-$$
datos=/dev/shm/vec-pg-revision-bolsa-$$
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then docker run --rm -v "$datos":/d --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true; fi
  rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run --detach --rm --network none --name "$contenedor" -v "$datos":/var/lib/postgresql -v "$repo":/repo:ro \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -q -U postgres 2>/dev/null && break; sleep 0.5; done
sleep 2
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
fichero() { psql_pg -f "/repo/$1"; }
m=deploy/postgresql/bolsa_llamamientos/migraciones
for ruta in deploy/postgresql/autorizacion/roles_up.sql deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/bolsa_llamamientos/roles_up.sql deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql; do
  fichero "$ruta" >/dev/null
done
fichero deploy/postgresql/bolsa_llamamientos/pruebas_sql/contacto_origen/dobles_ad3.sql >/dev/null 2>&1
# B2, el portal del candidato y la emisión leen además la huella del efecto: sus dobles
# devuelven la firma completa.
psql_pg >/dev/null <<'SQL'
DO $f$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_y_consumir_situacion_participacion_v3_atestada','registrar_y_consumir_portal_candidato_bolsa_v3_atestada','registrar_y_consumir_emision_llamamiento_v3_atestada'] LOOP
  EXECUTE format('DROP FUNCTION IF EXISTS vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', n);
  EXECUTE format($s$CREATE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $q$SELECT 'decision:doble', convert_from(p_decision,'UTF8')::jsonb->>'recurso_ref', convert_from(p_decision,'UTF8')::jsonb->>'contexto_recurso_huella_sha256', repeat('c',64), 'auditoria:doble', now(), true$q$$s$, n);
  EXECUTE format('ALTER FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario', n);
 END LOOP;
END $f$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
SQL
for f in "$repo"/$m/*.up.sql; do
  n=$(basename "$f")
  [[ $n > 000031_~ ]] && break
  fichero "$m/$n" >/dev/null 2>&1 || { fichero "$m/$n" >&2; exit 1; }
  if [[ $n == 000010_* ]]; then fichero deploy/postgresql/bolsa_llamamientos/roles_registrador_frontera_up.sql >/dev/null; fi
done
rechaza() { if fichero "$1" >/dev/null 2>&1; then echo "aceptado y debía rechazarse: $1" >&2; exit 1; fi; }
# 000037 sin 000032: rechazada.
rechaza $m/000037_efectos_sanciones_participacion.up.sql
# 000032: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP.
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m/000032_politica_transiciones_situacion.up.sql" | psql_pg >/dev/null
psql_pg -tAc "SELECT to_regclass('vec_bolsa_llamamientos.politica_transiciones_situacion') IS NULL" | grep -qx t || { echo 'ROLLBACK de 000032 dejó objetos' >&2; exit 1; }
fichero $m/000032_politica_transiciones_situacion.up.sql >/dev/null
rechaza $m/000032_politica_transiciones_situacion.up.sql
fichero $m/000032_politica_transiciones_situacion.down.sql >/dev/null
rechaza $m/000032_politica_transiciones_situacion.down.sql
fichero $m/000032_politica_transiciones_situacion.up.sql >/dev/null
psql_pg -tAc "SELECT cardinality(transiciones)=18 AND 'disponible>disponible_desde'=ANY(transiciones) AND NOT EXISTS (SELECT 1 FROM unnest(transiciones) t WHERE t LIKE 'excluido>%') FROM vec_bolsa_llamamientos.politica_transiciones_situacion WHERE version=1" | grep -qx t || { echo 'versión 1 de la política inesperada' >&2; exit 1; }
for f in 000033_politica_segregacion 000034_traza_valores_participacion 000035_origen_datos_contacto; do fichero "$m/$f.up.sql" >/dev/null; done
# 000037: ROLLBACK, UP, doble UP, DOWN de 000032 rechazado, DOWN, doble DOWN, UP.
sed 's/^COMMIT;$/ROLLBACK;/' "$repo/$m/000037_efectos_sanciones_participacion.up.sql" | psql_pg >/dev/null
fichero $m/000037_efectos_sanciones_participacion.up.sql >/dev/null
rechaza $m/000037_efectos_sanciones_participacion.up.sql
rechaza $m/000032_politica_transiciones_situacion.down.sql
fichero $m/000037_efectos_sanciones_participacion.down.sql >/dev/null
rechaza $m/000037_efectos_sanciones_participacion.down.sql
fichero $m/000037_efectos_sanciones_participacion.up.sql >/dev/null
echo 'UP/DOWN/UP, dependencias y dobles aplicaciones: OK'
# La readmisión no tiene EXECUTE para nadie salvo su propietario.
psql_pg -tAc "SELECT NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz)','EXECUTE')
  AND NOT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid='vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(text,text,text,text,text,text,text,timestamptz)'::regprocedure AND a.grantee<>p.proowner)" \
  | grep -qx t || { echo 'la readmisión es invocable por otros roles' >&2; exit 1; }
echo 'ACL: OK'
p=deploy/postgresql/bolsa_llamamientos/pruebas_sql
prueba() { # fichero, marca de éxito
  local salida
  salida=$(fichero "$1" 2>&1) || { printf '%s\n' "$salida" >&2; exit 1; }
  [[ -z ${2:-} ]] || grep -q "$2" <<<"$salida" || { printf '%s\n' "$salida" >&2; exit 1; }
  echo "OK $(basename "$1")"
}
prueba $p/b37_efectos_sanciones.sql 'OK B37 efectos de sanciones'
fichero $p/revision/datos.sql >/dev/null
prueba $p/b2_politica_transiciones.sql
prueba $p/bof_ofertas_publicadas.sql 'OK B-OF focal'
prueba $p/b30_portal_candidato.sql
prueba $p/revision/b30_replay_autorizacion.sql 'OK B30 replay con decisión viva'
