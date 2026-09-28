#!/usr/bin/env bash
# Ensayo aislado de Usuarios 000003. VEC_USUARIOS_BASE_SQL permite leer el WIP
# 000001/000002 de su escritor sin copiarlo ni editarlo.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
sql="$raiz/deploy/postgresql/usuarios_vec"
base=${VEC_USUARIOS_BASE_SQL:-$sql}
base_raiz=$(realpath "$base/../../..")
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-frontera-$$"
datos="/tmp/$contenedor"
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --pull never --network none -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
 rmdir "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$contenedor" \
 -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break
 sleep 0.5
done
sleep 2
docker exec "$contenedor" pg_isready -q -U postgres
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
psql_login() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_registrador_prueba -d postgres "$@"; }
psql_pg < "$base/roles_up.sql" >/dev/null
psql_pg < "$base/pruebas_sql/preimagen_ad3_sintetica.sql" >/dev/null
psql_pg < "$base_raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000106_consumidor_preferencias_propias.up.sql" >/dev/null
psql_pg < "$base/migraciones/000001_preferencias_base.up.sql" >/dev/null
psql_pg < "$base/migraciones/000002_operaciones_preferencias.up.sql" >/dev/null
psql_pg < "$sql/roles_000003_up.sql" >/dev/null
psql_pg < "$sql/migraciones/000003_auditoria_frontera_preferencias.up.sql" >/dev/null
psql_pg <<'SQL' >/dev/null
CREATE ROLE vec_usuarios_registrador_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
 NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_registrador_frontera TO vec_usuarios_registrador_prueba
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
DO $test$ BEGIN
 IF NOT has_function_privilege('vec_usuarios_registrador_prueba',
      'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor',
      'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba',
      'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
      WHERE p.oid='vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::regprocedure
        AND a.grantee=0 AND a.privilege_type='EXECUTE')
    OR has_table_privilege('vec_usuarios_registrador_prueba',
      'vec_usuarios.denegacion_frontera_preferencias','SELECT')
    OR has_table_privilege('vec_usuarios_registrador_prueba',
      'vec_usuarios.denegacion_frontera_preferencias','INSERT')
    OR has_table_privilege('vec_usuarios_registrador_prueba',
      'vec_usuarios.preferencias_actual','SELECT')
    OR has_function_privilege('vec_usuarios_registrador_prueba',
      'vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_registrador_prueba',
      'vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Usuarios 000003: ACL abierta'; END IF;
END $test$;
SQL
for rol in vec_usuarios_ejecutor vec_contratacion_temporal_ejecutor; do
 if psql_pg >/dev/null 2>&1 <<SQL
SET ROLE $rol;
SELECT vec_usuarios.registrar_denegacion_preferencias_v1(
 'corr_no_disponible','autenticacion_requerida',
 'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',NULL);
SQL
 then
  echo "Usuarios 000003: llamada no autorizada por $rol" >&2
  exit 1
 fi
done
probar_grant_directo() {
 psql_pg >/dev/null <<SQL
$1
SQL
 psql_login >/dev/null <<'SQL'
DO $test$ BEGIN
 BEGIN
  PERFORM vec_usuarios.registrar_denegacion_preferencias_v1(
   'corr_no_disponible','autenticacion_requerida',
   'api.usuarios.preferencias.ruta_exacta','/api/vec/usuarios/mis-preferencias',NULL);
  RAISE EXCEPTION 'Usuarios 000003: LOGIN con GRANT directo aceptado';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
SQL
 psql_pg >/dev/null <<SQL
$2
SQL
}
probar_grant_directo \
 'GRANT CREATE ON SCHEMA vec_usuarios TO vec_usuarios_registrador_prueba;' \
 'REVOKE CREATE ON SCHEMA vec_usuarios FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT SELECT ON vec_usuarios.preferencias_actual TO vec_usuarios_registrador_prueba;' \
 'REVOKE SELECT ON vec_usuarios.preferencias_actual FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT SELECT (persona_ref) ON vec_usuarios.preferencias_actual TO vec_usuarios_registrador_prueba;' \
 'REVOKE SELECT (persona_ref) ON vec_usuarios.preferencias_actual FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT USAGE ON SEQUENCE vec_usuarios.denegacion_frontera_preferencias_evento_id_seq TO vec_usuarios_registrador_prueba;' \
 'REVOKE USAGE ON SEQUENCE vec_usuarios.denegacion_frontera_preferencias_evento_id_seq FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT EXECUTE ON FUNCTION vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_registrador_prueba;' \
 'REVOKE EXECUTE ON FUNCTION vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT EXECUTE ON FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text) TO vec_usuarios_registrador_prueba;' \
 'REVOKE EXECUTE ON FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text) FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_usuarios_registrador_prueba;' \
 'REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_usuarios_registrador_prueba;'
probar_grant_directo \
 'GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_registrador_frontera;' \
 'REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_usuarios_registrador_frontera;'
psql_login < "$sql/pruebas_sql/frontera_operaciones.sql" >/dev/null
psql_pg <<'SQL' >/dev/null
SET ROLE vec_usuarios_propietario;
DO $test$ DECLARE n integer; BEGIN
 IF (SELECT count(*) FROM vec_usuarios.denegacion_frontera_preferencias)<>2
 THEN RAISE EXCEPTION 'Usuarios 000003: historia inesperada'; END IF;
 BEGIN
  UPDATE vec_usuarios.denegacion_frontera_preferencias SET motivo='acceso_denegado';
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>0 THEN RAISE EXCEPTION 'Usuarios 000003: UPDATE permitido'; END IF;
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  DELETE FROM vec_usuarios.denegacion_frontera_preferencias;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>0 THEN RAISE EXCEPTION 'Usuarios 000003: DELETE permitido'; END IF;
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  TRUNCATE vec_usuarios.denegacion_frontera_preferencias;
  RAISE EXCEPTION 'Usuarios 000003: TRUNCATE permitido';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $test$;
SQL
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break
 sleep 0.5
done
psql_pg <<'SQL' >/dev/null
SET ROLE vec_usuarios_propietario;
DO $test$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.denegacion_frontera_preferencias)<>2
    OR (SELECT count(*) FROM vec_usuarios.denegacion_frontera_preferencias
        WHERE motivo='autenticacion_requerida' AND actor_ref IS NULL)<>1
 THEN RAISE EXCEPTION 'Usuarios 000003: historia tras reinicio incorrecta'; END IF;
END $test$;
SQL
echo 'Usuarios 000003 PG18: ACL y GRANT directo, TEMP, entradas cerradas, append-only, rollback y reinicio OK (base sintética)'
