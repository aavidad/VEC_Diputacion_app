#!/usr/bin/env bash
# PG18 efímero con V3 sintética estructural. No acredita COSE ni SMTP.
set -Eeuo pipefail
raiz=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-usuarios-correos-$$"
datos="/tmp/$contenedor"
socket_dir="$datos/socket"
primera_salida=''
segunda_salida=''
limpiar() {
 docker rm -f "$contenedor" >/dev/null 2>&1 || true
 docker run --rm --pull never --network none -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 /d/socket >/dev/null 2>&1 || true
 rmdir "$datos" 2>/dev/null || true
 if [[ -n $primera_salida ]]; then rm -f "$primera_salida"; fi
 if [[ -n $segunda_salida ]]; then rm -f "$segunda_salida"; fi
}
trap limpiar EXIT
mkdir -p "$socket_dir"
docker run -d --rm --pull never --network none --name "$contenedor" \
 -v "$datos:/var/lib/postgresql" -v "$socket_dir:/var/run/postgresql" \
 -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" psql -X -q -U postgres -c 'SELECT 1' >/dev/null 2>&1 && break
 sleep 0.5
done
sleep 2
docker exec -u root "$contenedor" chmod 0777 /var/run/postgresql
if ! docker exec "$contenedor" psql -X -q -U postgres -c 'SELECT 1' >/dev/null 2>&1; then
 docker logs "$contenedor" >&2 || true
 exit 1
fi
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
sql="$raiz/deploy/postgresql/usuarios_vec"
ad3="$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones"
pref_raiz=${VEC_USUARIOS_PREF_WORKTREE:-$raiz}
pref_sql="$pref_raiz/deploy/postgresql/usuarios_vec"
pref_ad3="$pref_raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones"
psql_pg < "$pref_sql/roles_up.sql" >/dev/null
psql_pg < "$sql/roles_correos_up.sql" >/dev/null
psql_pg < "$pref_sql/pruebas_sql/preimagen_ad3_sintetica.sql" >/dev/null
psql_pg < "$pref_ad3/000106_consumidor_preferencias_propias.up.sql" >/dev/null
psql_pg < "$pref_sql/migraciones/000001_preferencias_base.up.sql" >/dev/null
psql_pg < "$pref_sql/migraciones/000002_operaciones_preferencias.up.sql" >/dev/null
psql_pg < "$ad3/000107_consumidor_correos_usuarios.up.sql" >/dev/null
psql_pg < "$sql/migraciones/000004_correos_propios.up.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/correos_vector_material.sql" >/dev/null
psql_pg < "$sql/pruebas_sql/correos_operaciones_sinteticas.sql" >/dev/null
psql_pg <<'SQL' >/dev/null
DO $test$ BEGIN
 IF has_table_privilege('vec_usuarios_prueba_interna','vec_usuarios.correos_direccion','SELECT')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.validar_material_correos(text,bytea,bytea,numeric,numeric)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.reservar_despacho_correo_v1(text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.consultar_correo_activo_verificado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_despachador','vec_usuarios.reservar_despacho_correo_v1(text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_despachador','vec_usuarios.leer_sobre_despacho_correo_v1(text,text,text)','EXECUTE')
    OR has_function_privilege('vec_usuarios_despachador','vec_usuarios.reconciliar_aceptacion_correo_v1(text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_prueba_interna','vec_usuarios.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_usuarios' AND tablename LIKE 'correos_%' AND roles<>ARRAY['vec_usuarios_propietario']::name[])
 THEN RAISE EXCEPTION 'ACL/RLS correos incompatible'; END IF;
END $test$;
SQL
psql_pg <<'SQL' >/dev/null
DO $test$ BEGIN
 BEGIN
  INSERT INTO vec_usuarios.correos_outbox(outbox_ref,persona_ref,correo_ref,desafio_ref,clave_ref,vence_en,tipo,creado_en)
  VALUES('correo_outbox:'||replace(gen_random_uuid()::text,'-',''),
   'per_ABCDEFGHIJKLMNOPQRSTUV','correo:11111111111111111111111111111111',
   'desafio:sintetico-0000000001','codigo-v1',clock_timestamp()+interval '1 day','verificacion',clock_timestamp());
  RAISE EXCEPTION 'outbox sin desafio admitido';
 EXCEPTION WHEN check_violation THEN NULL; END;
END $test$;
SQL
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$
DECLARE r jsonb; g jsonb;
 a constant text:='correo:11111111111111111111111111111111';
 b constant text:='correo:22222222222222222222222222222222';
BEGIN
 g:=public.probar_correos('vec.correos.consultar',0,'','','');
 IF (g->>'version')::integer<>0 OR g->'correos'<>'[]'::jsonb THEN RAISE EXCEPTION 'GET inicial'; END IF;
 r:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',a,'');
 IF (r->>'version')::integer<>1 OR r->>'correo_ref'<>a THEN RAISE EXCEPTION 'alta inicial'; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',1,'correo-prueba-clave-igualdad-v2',b,'',
   'aplicar',NULL,false,'interna_corporativa','igualdad-v2');
  RAISE EXCEPTION 'rotación sin reindexado aceptada';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',1,'correo-prueba-misma-igualdad',b,'',
   'aplicar',NULL,false,'interna_corporativa','igualdad-v1',a);
  RAISE EXCEPTION 'misma huella de dirección admitida';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 IF public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',a,'','recuperar')->>'recibo_ref'<>r->>'recibo_ref'
 THEN RAISE EXCEPTION 'replay alta'; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',b,'','recuperar');
  RAISE EXCEPTION 'clave divergente';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 BEGIN
  PERFORM public.probar_correos('vec.correos.reenviar',1,'correo-prueba-v3-falsa',a,'','aplicar',NULL,true);
  RAISE EXCEPTION 'V3 falsa';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verificar-fallo',a,'','aplicar',false);
 IF r->>'valido'<>'false' THEN RAISE EXCEPTION 'intento falso no persistente'; END IF;
 r:=public.probar_correos('vec.correos.verificar',1,'correo-prueba-verificar-0001',a,'','aplicar',true);
 IF (r->>'version')::integer<>2 THEN RAISE EXCEPTION 'verificación'; END IF;
 r:=public.probar_correos('vec.correos.activar',2,'correo-prueba-activar-0001',a,'');
 IF (r->>'version')::integer<>3 THEN RAISE EXCEPTION 'activar'; END IF;
 r:=public.probar_correos('vec.correos.anadir',3,'correo-prueba-anadir-0002',b,'');
 IF (r->>'version')::integer<>4 THEN RAISE EXCEPTION 'segunda alta'; END IF;
 r:=public.probar_correos('vec.correos.verificar',4,'correo-prueba-verificar-0002',b,'','aplicar',true);
 IF (r->>'version')::integer<>5 THEN RAISE EXCEPTION 'segunda verificación'; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.retirar',5,'correo-prueba-retirar-sin-sustituto',a,'');
  RAISE EXCEPTION 'retirada activa sin sustituto';
 EXCEPTION WHEN SQLSTATE 'P1409' THEN NULL; END;
 r:=public.probar_correos('vec.correos.retirar',5,'correo-prueba-retirar-0001',a,b);
 IF (r->>'version')::integer<>6 THEN RAISE EXCEPTION 'retirada con sustituto'; END IF;
 g:=public.probar_correos('vec.correos.consultar',0,'','','');
 IF (g->>'version')::integer<>6 OR jsonb_array_length(g->'correos')<>1 OR g#>>'{correos,0,correo_ref}'<>b
 THEN RAISE EXCEPTION 'consulta posterior'; END IF;
END $test$;
COMMIT;
SQL
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$
DECLARE r jsonb; i integer;
 a constant text:='correo:11111111111111111111111111111111';
 c constant text:='correo:33333333333333333333333333333333';
BEGIN
 r:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',a,'');
 IF r->>'replay'<>'true' OR (r->>'version')::integer<>1 THEN RAISE EXCEPTION 'replay directo'; END IF;
 r:=public.probar_correos('vec.correos.anadir',6,'correo-prueba-anadir-0003',c,'');
 IF (r->>'version')::integer<>7 THEN RAISE EXCEPTION 'tercera alta'; END IF;
 FOR i IN 1..3 LOOP
  r:=public.probar_correos('vec.correos.reenviar',6+i,'correo-prueba-reenviar-000'||i,c,'');
  IF (r->>'version')::integer<>7+i THEN RAISE EXCEPTION 'reenvío'; END IF;
 END LOOP;
 BEGIN
  PERFORM public.probar_correos('vec.correos.reenviar',10,'correo-prueba-reenviar-0004',c,'');
  RAISE EXCEPTION 'cuarto reenvío aceptado';
 EXCEPTION WHEN SQLSTATE 'P1429' THEN NULL; END;
 FOR i IN 1..5 LOOP
  r:=public.probar_correos('vec.correos.verificar',10,'correo-prueba-intento-000'||i,c,'','aplicar',false);
  IF r->>'valido'<>'false' THEN RAISE EXCEPTION 'intento inválido'; END IF;
 END LOOP;
 BEGIN
  PERFORM public.probar_correos('vec.correos.verificar',10,'correo-prueba-intento-0006',c,'','aplicar',false);
  RAISE EXCEPTION 'sexto intento aceptado';
 EXCEPTION WHEN SQLSTATE 'P1429' THEN NULL; END;
END $test$;
COMMIT;
SQL
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 80); do
 docker exec "$contenedor" psql -X -q -U postgres -c 'SELECT 1' >/dev/null 2>&1 && break
 sleep 0.5
done
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$ DECLARE r jsonb; BEGIN
 r:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',
  'correo:11111111111111111111111111111111','','recuperar');
 IF (r->>'version')::integer<>1 OR r->>'replay'<>'true' OR r->>'recibo_ref' !~ '^correo_recibo:[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'recibo original perdido tras reinicio'; END IF;
END $test$;
COMMIT;
SQL
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_cruzada -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$ BEGIN
 BEGIN
  PERFORM public.probar_correos('vec.correos.consultar',0,'','','','aplicar',NULL,false,'interna_corporativa');
  RAISE EXCEPTION 'login con dos roles aceptó consulta';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
COMMIT;
SQL
psql_pg <<'SQL' >/dev/null
DO $test$ BEGIN
 IF (SELECT count(*) FROM vec_usuarios.correos_conjunto)<>1
    OR (SELECT version FROM vec_usuarios.correos_conjunto)<>10
    OR (SELECT count(*) FROM vec_usuarios.correos_direccion WHERE activo)<>1
    OR (SELECT count(*) FROM vec_usuarios.correos_historia)<>10
    OR (SELECT count(*) FROM vec_usuarios.correos_recibo)<>10
    OR (SELECT count(*) FROM vec_usuarios.correos_desafio WHERE intentos=1)<>1
    OR (SELECT count(*) FROM vec_usuarios.correos_desafio WHERE intentos=5 AND estado='agotado')<>1
    OR (SELECT count(*) FROM vec_usuarios.correos_intento_fallido)<>6
    OR (SELECT count(*) FROM vec_usuarios.correos_reenvio)<>3
    OR (SELECT clave_igualdad_ref FROM vec_usuarios.correos_conjunto)<>'igualdad-v1'
    OR NOT EXISTS(SELECT 1 FROM vec_usuarios.correos_historia WHERE version=6
       AND anterior_activo_ref='correo:11111111111111111111111111111111'
       AND sustituto_ref='correo:22222222222222222222222222222222')
    OR (SELECT count(*) FROM vec_usuarios.correos_contexto)<>0
 THEN RAISE EXCEPTION 'estado/historia correos divergente'; END IF;
END $test$;
SQL
psql_pg <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
DO $rls$ DECLARE n integer; BEGIN
 SELECT count(*) INTO n FROM vec_usuarios.correos_direccion;
 IF n<>0 THEN RAISE EXCEPTION 'propietario leyó correos sin contexto'; END IF;
 BEGIN
  INSERT INTO vec_usuarios.correos_conjunto(persona_ref,version,actualizado_en)
  VALUES('per_0123456789abcdefghijkl',1,clock_timestamp());
  RAISE EXCEPTION 'propietario escribió correos sin contexto';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $rls$;
ROLLBACK;
SQL
primera_salida=$(mktemp /tmp/vec-correos-race-1.XXXXXX)
segunda_salida=$(mktemp /tmp/vec-correos-race-2.XXXXXX)
{
 docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT public.probar_correos('vec.correos.anadir',10,'correo-prueba-carrera-0001',
 'correo:44444444444444444444444444444444','');
SELECT pg_sleep(2);
COMMIT;
SQL
} >"$primera_salida" 2>&1 &
primero=$!
sleep 0.4
if docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >"$segunda_salida" 2>&1; then
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT public.probar_correos('vec.correos.anadir',10,'correo-prueba-carrera-0001',
 'correo:44444444444444444444444444444444','');
COMMIT;
SQL
 echo 'carrera aceptó dos altas de la misma clave' >&2; exit 1
fi
wait "$primero"
if ! rg -q 'could not serialize access|40001' "$segunda_salida"; then
 cat "$segunda_salida" >&2
 echo 'carrera falló fuera de SERIALIZABLE' >&2; exit 1
fi
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$ DECLARE r jsonb; BEGIN
 r:=public.probar_correos('vec.correos.anadir',10,'correo-prueba-carrera-0001',
  'correo:44444444444444444444444444444444','','recuperar');
 IF (r->>'version')::integer<>11 OR r->>'replay'<>'true' THEN RAISE EXCEPTION 'replay tras carrera'; END IF;
END $test$;
COMMIT;
SQL
psql_pg <<'SQL' >/dev/null
DO $test$ BEGIN
 IF (SELECT version FROM vec_usuarios.correos_conjunto)<>11
    OR (SELECT count(*) FROM vec_usuarios.correos_recibo)<>11
    OR (SELECT count(*) FROM vec_usuarios.correos_historia)<>11
 THEN RAISE EXCEPTION 'carrera duplicó historia o recibo'; END IF;
END $test$;
SQL
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_externa -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$ DECLARE r jsonb; BEGIN
 r:=public.probar_correos('vec.correos.consultar',0,'','','','aplicar',NULL,false,'externa_personal');
 IF (r->>'version')::integer<>11 OR jsonb_array_length(r->'correos')<>3
 THEN RAISE EXCEPTION 'lectura exterior del mismo conjunto'; END IF;
 r:=public.probar_correos('vec.correos.anadir',0,'correo-prueba-anadir-0001',
  'correo:11111111111111111111111111111111','','recuperar',NULL,false,'externa_personal');
 IF (r->>'version')::integer<>1 OR r->>'replay'<>'true'
 THEN RAISE EXCEPTION 'replay entre superficies'; END IF;
 BEGIN
  PERFORM public.probar_correos('vec.correos.consultar',0,'','','','aplicar',NULL,false,'interna_corporativa');
  RAISE EXCEPTION 'login exterior aceptó audiencia interna';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 r:=public.probar_correos('vec.correos.anadir',11,'correo-prueba-exterior-0001',
  'correo:55555555555555555555555555555555','','aplicar',NULL,false,'externa_personal');
 IF (r->>'version')::integer<>12 THEN RAISE EXCEPTION 'alta exterior'; END IF;
END $test$;
COMMIT;
SQL
docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U vec_usuarios_prueba_interna -d postgres <<'SQL' >/dev/null
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$ BEGIN
 BEGIN
  PERFORM public.probar_correos('vec.correos.consultar',0,'','','','aplicar',NULL,false,'externa_personal');
  RAISE EXCEPTION 'login interno aceptó audiencia exterior';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
COMMIT;
SQL
psql_pg <<'SQL' >/dev/null
DO $test$ BEGIN
 IF (SELECT version FROM vec_usuarios.correos_conjunto)<>12
    OR (SELECT count(*) FROM vec_usuarios.correos_recibo)<>12
    OR has_function_privilege('vec_usuarios_prueba_externa','vec_usuarios.consultar_correo_activo_verificado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_usuarios_prueba_externa','vec_usuarios.reservar_despacho_correo_v1(text,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'superficies o ACL exterior divergentes'; END IF;
END $test$;
SQL
if [[ -n ${VEC_USUARIOS_CORREOS_GO_WORKTREE:-} ]]; then
  (
    cd "$VEC_USUARIOS_CORREOS_GO_WORKTREE"
    VEC_USUARIOS_CORREOS_PG18_DSN="host=$socket_dir user=vec_usuarios_prueba_interna dbname=postgres sslmode=disable" \
      "${VEC_USUARIOS_CORREOS_GO_BIN:-go}" test ./internal/modules/usuarios/adapters/postgres -run '^TestCorreosPG18PoolNominal$' -count=1
  )
fi
echo 'Usuarios 5.08b PG18: cadena, dos superficies, ACL/RLS, transiciones, cuotas, replay, carrera y reinicio sintéticos OK; sin COSE ni SMTP'
