#!/usr/bin/env bash
# PostgreSQL 18 efímero; doble V3 estructural, sin COSE real ni datos reales.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-pref-$$"
datos="/tmp/$contenedor"
primera_salida=''
segunda_salida=''
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --pull never --network none -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
 rmdir "$datos" 2>/dev/null || true
 if [[ -n $primera_salida ]]; then rm -f "$primera_salida"; fi
 if [[ -n $segunda_salida ]]; then rm -f "$segunda_salida"; fi
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
if ! docker exec "$contenedor" pg_isready -q -U postgres; then
 docker logs "$contenedor" >&2 || true
 exit 1
fi
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
sql="$raiz/deploy/postgresql/usuarios_vec"
ad3="$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000106_consumidor_preferencias_propias.up.sql"
psql_pg < "$sql/roles_up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/preimagen_ad3_sintetica.sql" >/dev/null
psql_pg < "$ad3" >/dev/null
psql_pg < "$sql/migraciones/000001_preferencias_base.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000002_operaciones_preferencias.up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/vector_contexto_go.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/operaciones_sinteticas.sql" >/dev/null
psql_login() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba -d postgres "$@"; }
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
    WHERE p.oid='vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND a.grantee=0)
    OR NOT has_function_privilege('vec_usuarios_ejecutor','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_usuarios_ejecutor','vec_usuarios.preferencias_actual','SELECT')
    OR has_function_privilege('vec_usuarios_ejecutor','vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_usuarios_propietario','vec_usuarios_migrador','vec_usuarios_ejecutor')
      AND (rolcanlogin OR rolbypassrls OR rolsuper))
    OR EXISTS (SELECT 1 FROM pg_auth_members m
      WHERE m.member='vec_usuarios_prueba'::regrole
      AND (m.roleid<>'vec_usuarios_ejecutor'::regrole OR m.set_option OR m.admin_option OR NOT m.inherit_option))
    OR (SELECT count(*) FROM pg_auth_members WHERE member='vec_usuarios_prueba'::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios'
      AND roles<>ARRAY['vec_usuarios_propietario']::name[])
 THEN RAISE EXCEPTION 'ACL Usuarios abierta o ejecutor sin fachada'; END IF;
END $prueba$;
SQL
psql_login <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $prueba$
DECLARE v jsonb; g jsonb; r jsonb; repetido jsonb; valores jsonb;
 clave constant text:='preferencias-prueba-0001';
BEGIN
 valores:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 g:=public.probar_preferencias('get',0,'',valores);
 IF g->>'existe'<>'false' OR (g->>'version')::integer<>0 OR g->'valores' IS DISTINCT FROM valores
 THEN RAISE EXCEPTION 'GET ausente no devuelve v0/defaults'; END IF;
 IF public.probar_preferencias('rec',0,clave,valores) IS NOT NULL
 THEN RAISE EXCEPTION 'REC ausente produjo recibo'; END IF;
 BEGIN
  PERFORM public.probar_preferencias('rec',0,'preferencias-prueba-falta',valores,true);
  RAISE EXCEPTION 'V3 falsa reveló clave ausente';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 r:=public.probar_preferencias('put',0,clave,valores);
 IF (r->>'version')::integer<>1 OR r->>'recibo_ref' !~ '^pref_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'PUT inicial incorrecto'; END IF;
 repetido:=public.probar_preferencias('rec',0,clave,valores);
 IF r-'replay' IS DISTINCT FROM repetido-'replay' OR repetido->>'replay'<>'true'
 THEN RAISE EXCEPTION 'replay modificó recibo'; END IF;
 BEGIN
  PERFORM public.probar_preferencias('rec',0,clave,valores,true);
  RAISE EXCEPTION 'V3 falsa reveló clave existente';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('put',0,'preferencias-prueba-0003',valores,true);
  RAISE EXCEPTION 'V3 falsa reveló CAS';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('put',0,'preferencias-prueba-0002',valores);
  RAISE EXCEPTION 'CAS obsoleto aceptado';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  v:=jsonb_set(valores,'{filas}','50'::jsonb);
  PERFORM public.probar_preferencias('rec',0,clave,v);
  RAISE EXCEPTION 'clave divergente aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 g:=public.probar_preferencias('get',0,'',valores);
 IF g->>'existe'<>'true' OR (g->>'version')::integer<>1 OR g->'valores' IS DISTINCT FROM valores
 THEN RAISE EXCEPTION 'GET posterior incorrecto'; END IF;
END $prueba$;
COMMIT;
SQL
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.preferencias_actual)<>1
    OR (SELECT count(*) FROM vec_usuarios.preferencias_historia)<>1
    OR (SELECT count(*) FROM vec_usuarios.preferencias_recibo)<>1
    OR (SELECT count(*) FROM vec_usuarios.contexto_transaccion)<>0
    OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.prueba_consumos)<5
 THEN RAISE EXCEPTION 'historia o recibo duplicados'; END IF;
 BEGIN
  UPDATE vec_usuarios.preferencias_historia SET version=2;
  RAISE EXCEPTION 'historia mutable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF has_table_privilege('vec_usuarios_prueba','vec_usuarios.preferencias_recibo','SELECT')
    OR has_table_privilege('vec_usuarios_prueba','vec_usuarios.contexto_transaccion','SELECT')
    OR has_function_privilege('vec_usuarios_prueba','vec_usuarios.activar_contexto_preferencias(text,text,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'lectura directa de recibo'; END IF;
END $prueba$;
SQL
psql_pg <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
DO $rls$ DECLARE n integer; BEGIN
 SELECT count(*) INTO n FROM vec_usuarios.preferencias_actual;
 IF n<>0 THEN RAISE EXCEPTION 'owner lee estado sin contexto V3'; END IF;
 BEGIN
  INSERT INTO vec_usuarios.preferencias_actual(persona_ref,version,catalogo_version_ref,valores,recibo_ref,actualizado_en)
  VALUES('per_0123456789abcdefghijkl',1,'usuarios-preferencias-v1',
   '{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb,
   'pref_0123456789abcdef0123456789abcdef',clock_timestamp());
  RAISE EXCEPTION 'owner escribió sin contexto V3';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $rls$;
ROLLBACK;
SQL
psql_login <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
SAVEPOINT prueba_guardado;
DO $rollback$ DECLARE v jsonb; r jsonb; BEGIN
 v:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 r:=public.probar_preferencias('put',1,'preferencias-savepoint-0001',v);
 IF (r->>'version')::integer<>2 THEN RAISE EXCEPTION 'PUT de savepoint'; END IF;
END $rollback$;
ROLLBACK TO SAVEPOINT prueba_guardado;
DO $rollback$ DECLARE v jsonb; g jsonb; BEGIN
 v:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 g:=public.probar_preferencias('get',0,'',v);
 IF (g->>'version')::integer<>1 THEN RAISE EXCEPTION 'rollback no restauró versión'; END IF;
END $rollback$;
COMMIT;
SQL
psql_pg -c "DO \$\$ BEGIN IF (SELECT count(*) FROM vec_usuarios.contexto_transaccion)<>0 OR (SELECT count(*) FROM vec_usuarios.preferencias_historia)<>1 THEN RAISE EXCEPTION 'savepoint dejó marcador o historia'; END IF; END \$\$" >/dev/null
primera_salida=$(mktemp /tmp/vec-pref-concurrencia-1.XXXXXX)
segunda_salida=$(mktemp /tmp/vec-pref-concurrencia-2.XXXXXX)
{
 psql_login <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
SELECT public.probar_preferencias('put',1,'preferencias-concurrencia-1',
 '{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb);
SELECT pg_sleep(2);
COMMIT;
SQL
} >"$primera_salida" 2>&1 &
primero=$!
sleep 0.4
if psql_login <<'SQL' >"$segunda_salida" 2>&1; then
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
SELECT public.probar_preferencias('put',1,'preferencias-concurrencia-2',
 '{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb);
COMMIT;
SQL
 echo 'CAS concurrente aceptó las dos versiones' >&2; exit 1
fi
wait "$primero"
if ! rg -q 'could not serialize access|40001' "$segunda_salida"; then
 cat "$segunda_salida" >&2
 echo 'fallo concurrente distinto de 40001' >&2; exit 1
fi
psql_pg -c "DO \$\$ BEGIN IF (SELECT version FROM vec_usuarios.preferencias_actual)<>2 OR (SELECT count(*) FROM vec_usuarios.preferencias_historia)<>2 OR (SELECT count(*) FROM vec_usuarios.preferencias_recibo)<>2 OR (SELECT count(*) FROM vec_usuarios.contexto_transaccion)<>0 THEN RAISE EXCEPTION 'CAS concurrente duplicó efecto o dejó contexto'; END IF; END \$\$" >/dev/null
echo 'Usuarios 5.08a PG18 sintético: ACL/RLS, GET ausente, CAS concurrente, replay, conflicto, historia, rollback y denegación OK (V3 doble, no COSE)'
