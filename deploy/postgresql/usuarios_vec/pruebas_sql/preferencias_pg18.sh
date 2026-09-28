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
psql_login() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres "$@"; }
psql_externa() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_externa -d postgres "$@"; }
psql_cruzada() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_cruzada -d postgres "$@"; }
psql_bolsa() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_bolsa_prueba -d postgres "$@"; }
psql_ct() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_ct_prueba -d postgres "$@"; }
psql_pg <<'SQL' >/dev/null
DO $prueba$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
    WHERE p.oid='vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND a.grantee=0)
    OR NOT has_function_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_ejecutor_externo','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.preferencias_actual','SELECT')
    OR has_table_privilege('vec_usuarios_ejecutor_externo','vec_usuarios.preferencias_actual','SELECT')
    OR has_function_privilege('vec_usuarios_ejecutor_interno','vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo','vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
      ('vec_usuarios_propietario','vec_usuarios_migrador','vec_usuarios_ejecutor_interno','vec_usuarios_ejecutor_externo')
      AND (rolcanlogin OR rolbypassrls OR rolsuper))
    OR EXISTS (SELECT 1 FROM pg_auth_members m
      WHERE m.member='vec_usuarios_prueba_interna'::regrole
      AND (m.roleid<>'vec_usuarios_ejecutor_interno'::regrole OR m.set_option OR m.admin_option OR NOT m.inherit_option))
    OR EXISTS (SELECT 1 FROM pg_auth_members m
      WHERE m.member='vec_usuarios_prueba_externa'::regrole
      AND (m.roleid<>'vec_usuarios_ejecutor_externo'::regrole OR m.set_option OR m.admin_option OR NOT m.inherit_option))
    OR (SELECT count(*) FROM pg_auth_members WHERE member='vec_usuarios_prueba_interna'::regrole)<>1
    OR (SELECT count(*) FROM pg_auth_members WHERE member='vec_usuarios_prueba_externa'::regrole)<>1
    OR EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios'
      AND roles<>ARRAY['vec_usuarios_propietario']::name[])
 THEN RAISE EXCEPTION 'ACL Usuarios abierta o ejecutor sin fachada'; END IF;
END $prueba$;
SQL
for perfil_ajeno in bolsa ct; do
 if [[ $perfil_ajeno == bolsa ]]; then prueba_ajena=psql_bolsa; else prueba_ajena=psql_ct; fi
 "$prueba_ajena" <<'SQL' >/dev/null
DO $guarda$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'preferencias_consulta_usuarios','x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea,
   1,1,convert_to('sonda_bolsa','UTF8'),'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'núcleo aceptó perfil Usuarios de sesión ajena';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $guarda$;
SQL
done
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
 IF has_table_privilege('vec_usuarios_prueba_interna','vec_usuarios.preferencias_recibo','SELECT')
    OR has_table_privilege('vec_usuarios_prueba_externa','vec_usuarios.preferencias_recibo','SELECT')
    OR has_table_privilege('vec_usuarios_prueba_interna','vec_usuarios.contexto_transaccion','SELECT')
    OR has_table_privilege('vec_usuarios_prueba_externa','vec_usuarios.contexto_transaccion','SELECT')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.activar_contexto_preferencias(text,text,text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_externa','vec_usuarios.activar_contexto_preferencias(text,text,text,text,text)','EXECUTE')
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
psql_externa <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $externa$ DECLARE v jsonb; g jsonb; r jsonb; BEGIN
 v:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 IF vec_usuarios.catalogo_vigente_preferencias_v1('externa_personal')->>'version_ref'<>'usuarios-preferencias-v1'
 THEN RAISE EXCEPTION 'catálogo exterior incompatible'; END IF;
 g:=public.probar_preferencias('get',0,'',v,false,'externa_personal');
 IF (g->>'version')::integer<>2 THEN RAISE EXCEPTION 'GET exterior no comparte estado de persona'; END IF;
 r:=public.probar_preferencias('put',2,'preferencias-exterior-0001',v,false,'externa_personal');
 IF (r->>'version')::integer<>3 THEN RAISE EXCEPTION 'PUT exterior no avanzó versión'; END IF;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'interna_corporativa');
  RAISE EXCEPTION 'exterior usó superficie interna';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.catalogo_vigente_preferencias_v1('interna_corporativa');
  RAISE EXCEPTION 'exterior consultó catálogo interno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'externa_personal',
   'vec_usuarios.preferencias.consultar.interna_corporativa.v1');
  RAISE EXCEPTION 'exterior usó audiencia interna';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'externa_personal',NULL,'interna_corporativa');
  RAISE EXCEPTION 'exterior usó vínculo interno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $externa$;
COMMIT;
SQL
psql_login <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TimeZone='UTC';
DO $interna$ DECLARE v jsonb; r jsonb; BEGIN
 v:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 IF vec_usuarios.catalogo_vigente_preferencias_v1('interna_corporativa')->>'version_ref'<>'usuarios-preferencias-v1'
 THEN RAISE EXCEPTION 'catálogo interno incompatible'; END IF;
 r:=public.probar_preferencias('rec',2,'preferencias-exterior-0001',v);
 IF (r->>'version')::integer<>3 OR r->>'replay'<>'true'
 THEN RAISE EXCEPTION 'replay interno de recibo exterior divergente'; END IF;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'externa_personal');
  RAISE EXCEPTION 'interno usó superficie exterior';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.catalogo_vigente_preferencias_v1('externa_personal');
  RAISE EXCEPTION 'interno consultó catálogo exterior';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'interna_corporativa',
   'vec_usuarios.preferencias.consultar.externa_personal.v1');
  RAISE EXCEPTION 'interno usó audiencia exterior';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v,false,'interna_corporativa',NULL,'externa_personal');
  RAISE EXCEPTION 'interno usó vínculo exterior';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $interna$;
COMMIT;
SQL
psql_cruzada <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $cruce$ DECLARE v jsonb; BEGIN
 v:='{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}'::jsonb;
 BEGIN
  PERFORM public.probar_preferencias('get',0,'',v);
  RAISE EXCEPTION 'LOGIN con dos roles pudo consultar';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_usuarios.catalogo_vigente_preferencias_v1('interna_corporativa');
  RAISE EXCEPTION 'LOGIN con dos roles pudo leer catálogo';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $cruce$;
ROLLBACK;
SQL
psql_pg -c "DO \$\$ BEGIN IF (SELECT version FROM vec_usuarios.preferencias_actual)<>3 OR (SELECT count(*) FROM vec_usuarios.preferencias_historia)<>3 OR (SELECT count(*) FROM vec_usuarios.preferencias_recibo)<>3 OR (SELECT count(*) FROM vec_usuarios.contexto_transaccion)<>0 THEN RAISE EXCEPTION 'superficies duplicaron estado o dejaron contexto'; END IF; END \$\$" >/dev/null
echo 'Usuarios 5.08a PG18 sintético: cuatro operaciones internas/externas, cruces denegados, ACL/RLS, CAS concurrente, replay, rollback e historia OK (V3 doble, no COSE)'
