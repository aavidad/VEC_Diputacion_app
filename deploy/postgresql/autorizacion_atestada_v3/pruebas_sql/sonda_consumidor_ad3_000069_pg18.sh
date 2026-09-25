#!/usr/bin/env bash
# Ensayo PG18.4 desechable de AD3-69 (sonda y lectura de gobierno V3 por
# consumidor cerrado 'ct' | 'personal_b2').
#
# Base: cadena canónica ContextoActor 1/2 + Autorización 1–7 + AD3 1/2/50a/53a
# (la misma que deploy/principal/composicion_interna/pruebas_canonicas). La
# restricción de audiencias de AD3-002 solo admite el alta CT; el fixture la
# amplía como DBA a las 13 audiencias que dejan instaladas las migraciones
# posteriores (CT y B2 54–56), sin ejecutar esas migraciones. Gobierno, raíz
# y claves son sintéticos (secretos de relleno, sin valor real).
#
# El fixture NO toca checkpoint_gobierno.revision: solo lo mueven los
# triggers reales de AD3-2, de modo que las revisiones de las claves quedan por
# encima del contador, como tras una publicación real de vec-server. v2 debe
# aceptarlas de inmediato (sin comparación entre escalas); v1 sigue igual y las
# rechaza salvo que el DBA adelante el contador (contraste, en ROLLBACK).
#
# Comprueba: ROLLBACK sin rastro, UP, segunda aplicación rechazada, positivos
# CT5 y B2-8, negativos (consumidor desconocido/NULL, mezcla, duplicado,
# orden, ausencia, exceso, huellas falsas, revisión falsificada, raíz B2
# propia, login incorrecto, revocación, rotación de puntero con salto de
# versión, caducidad, retroceso por mínimos de configuración y raíz del
# checkpoint), aislamiento entre consumidores, sin acceso directo a tablas ni
# secretos, funciones antiguas intactas (md5 antes/después), UP/DOWN/UP y
# reinicio.
# No acredita firma COSE, decisión PDP ni la cadena AD3 íntegra.
set -Eeuo pipefail
trap 'printf "AD3-69: orden fallida en la línea %s\n" "$LINENO" >&2' ERR
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-ad3-69-sonda-${BASHPID}"
mig="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones"
up="$mig/000069_sonda_lectura_gobierno_por_consumidor.up.sql"
down="$mig/000069_sonda_lectura_gobierno_por_consumidor.down.sql"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
sql() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
sql_como() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { sql < "$repo_dir/$1" >/dev/null; }
fallo() { printf 'AD3-69: %s\n' "$*" >&2; exit 1; }
ok() { printf 'OK %s\n' "$*"; }
esperar() {
  for _ in {1..120}; do
    if valor 'SELECT 1' >/dev/null 2>&1; then return 0; fi
    sleep 0.25
  done
  fallo 'PostgreSQL no quedó disponible'
}

"$motor" run -d --rm --network none --cpus 2 --name "$contenedor" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'se requiere PostgreSQL 18.4'

# --- Cadena canónica (igual que pruebas_canonicas/prueba_cadena_contexto_pg18.sh ad3)
sql >/dev/null <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
ca=deploy/postgresql/contexto_actor_v1
archivo "$ca/roles_up.sql"
archivo "$ca/migraciones/000001_contexto_actor_v1.up.sql"
archivo "$ca/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
archivo "$ca/pruebas_sql/fixtures_sinteticos.sql"
archivo deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
sql >/dev/null <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
au=deploy/postgresql/autorizacion
archivo "$au/roles_up.sql"
archivo "$au/roles_v2_up.sql"
archivo "$au/migraciones/000001_autorizacion.up.sql"
archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
archivo "$au/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql"
archivo "$au/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql"
archivo "$au/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql"
archivo "$au/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql"
archivo "$au/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql"
archivo "$au/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql"
archivo deploy/postgresql/contratacion_temporal/roles_up.sql
archivo deploy/postgresql/autorizacion_atestada_v3/roles_up.sql
sql >/dev/null <<'SQL'
CREATE ROLE vec_ad3_69_migrador LOGIN NOINHERIT;
GRANT CONNECT ON DATABASE postgres TO vec_ad3_69_migrador;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_69_migrador
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL
a3=deploy/postgresql/autorizacion_atestada_v3/migraciones
sql_como vec_ad3_69_migrador < "$repo_dir/$a3/000001_gobierno_y_registro_v3.up.sql" >/dev/null
sql_como vec_ad3_69_migrador < "$repo_dir/$a3/000002_consumidor_capacidad_v3.up.sql" >/dev/null
archivo "$a3/000050a_preflight_material_interno.up.sql"
archivo "$a3/000053a_lectura_configuracion_interna.up.sql"
ok 'cadena canónica: AD3 1/2/50a/53a instaladas'

# LOGIN nominal y dos LOGIN de contraste. Ninguno es superusuario.
sql >/dev/null <<'SQL'
CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_ad3_69_otro_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_ad3_69_otro_login
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_ad3_69_rol_extra NOLOGIN;
GRANT CONNECT ON DATABASE postgres TO vec_interno_preflight_v3_desarrollo, vec_ad3_69_otro_login;
SQL

# --- Fixture sintético de gobierno (como DBA; RLS no aplica a superusuario).
sql >/dev/null <<'SQL'
DO $aud$
DECLARE d text;
BEGIN
  SELECT c.conname INTO STRICT d FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
     AND c.contype='c' AND pg_get_constraintdef(c.oid) LIKE '%audiencia_consumo%';
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT %I', d);
END $aud$;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  ADD CONSTRAINT prueba_ad3_69_audiencias CHECK (audiencia_consumo = ANY (ARRAY[
   'vec_contratacion_temporal.confirmar_alta_atestada.v1',
   'vec_personal.alta_ejercicio.v1','vec_personal.lectura_incorporacion.v2',
   'vec_contratacion_temporal.incorporacion_ejercicio.v2',
   'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
   'vec_personal.registro_empleado.ficha.v1','vec_personal.registro_empleado.vacantes.v1',
   'vec_personal.registro_empleado.alta.v1','vec_personal.registro_empleado.hecho.v1',
   'vec_personal.registro_empleado.catalogo.consultar.v1',
   'vec_personal.registro_empleado.catalogo.publicar.v1',
   'vec_personal.registro_empleado.catalogo.retirar.v1',
   'vec_personal.registro_empleado.empleados.v1']));
CREATE SCHEMA prueba_ad3_69;
REVOKE ALL ON SCHEMA prueba_ad3_69 FROM PUBLIC;
CREATE TABLE prueba_ad3_69.audiencia(consumidor text, i int, nombre text, PRIMARY KEY (consumidor,i));
INSERT INTO prueba_ad3_69.audiencia VALUES
 ('ct',1,'vec_personal.alta_ejercicio.v1'),
 ('ct',2,'vec_personal.lectura_incorporacion.v2'),
 ('ct',3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 ('ct',4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 ('ct',5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'),
 ('personal_b2',1,'vec_personal.registro_empleado.ficha.v1'),
 ('personal_b2',2,'vec_personal.registro_empleado.vacantes.v1'),
 ('personal_b2',3,'vec_personal.registro_empleado.alta.v1'),
 ('personal_b2',4,'vec_personal.registro_empleado.hecho.v1'),
 ('personal_b2',5,'vec_personal.registro_empleado.catalogo.consultar.v1'),
 ('personal_b2',6,'vec_personal.registro_empleado.catalogo.publicar.v1'),
 ('personal_b2',7,'vec_personal.registro_empleado.catalogo.retirar.v1'),
 ('personal_b2',8,'vec_personal.registro_empleado.empleados.v1');
-- Clave sintética n: versión, revisión y orden de puntero = n; secreto de
-- relleno. No ajusta el checkpoint: solo lo mueven los triggers de AD3-2.
CREATE FUNCTION prueba_ad3_69.clave(n int, audiencia text, desde timestamptz, hasta timestamptz)
RETURNS void LANGUAGE sql AS $f$
 INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
  (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,
   emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
 SELECT 'k'||n,n,n,repeat('a',64),s,encode(sha256(s),'hex'),'emisor_sintetico',audiencia,desde,hasta,'acto_clave_'||n
   FROM (SELECT decode(repeat(lpad(to_hex(n),2,'0'),32),'hex') AS s) x;
 INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden,clave_id,version,establecida_en,acto_ref)
 VALUES (n,'k'||n,n,clock_timestamp()-interval '1 second','acto_puntero_clave_'||n);
$f$;
CREATE FUNCTION prueba_ad3_69.configuracion(n int, desde timestamptz, hasta timestamptz)
RETURNS void LANGUAGE sql AS $f$
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
  (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES ('c'||n,n,repeat(lpad(to_hex(n),2,'0'),32),desde,hasta,'acto_configuracion_'||n);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES ('c'||n,'r1',1);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual (orden,configuracion_revision,establecida_en,acto_ref)
 VALUES (n,'c'||n,clock_timestamp()-interval '1 second','acto_puntero_configuracion_'||n);
$f$;
-- Material tal como lo enviaría el llamante: claves vigentes en el orden fijado.
CREATE FUNCTION prueba_ad3_69.material(p_consumidor text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object(
  'claves',(SELECT jsonb_agg(jsonb_build_object('audiencia_consumo',k.audiencia_consumo,'clave_id',k.clave_id,
     'version',k.version,'revision_gobierno',k.revision_gobierno,'huella_gobierno_sha256',k.huella_gobierno_sha256,
     'huella_secreto_sha256',k.huella_secreto_sha256,'emisor_id',k.emisor_id) ORDER BY a.i)
    FROM prueba_ad3_69.audiencia a
    CROSS JOIN LATERAL (SELECT k.* FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
       JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k USING (clave_id,version)
      WHERE k.audiencia_consumo=a.nombre ORDER BY p.orden DESC LIMIT 1) k
   WHERE a.consumidor=p_consumidor),
  'configuracion',(SELECT jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,
     'huella_configuracion_sha256',c.huella_configuracion_sha256)
    FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
    JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision
   ORDER BY p.orden DESC LIMIT 1),
  'raiz',(SELECT jsonb_build_object('clave_id',clave_id,'version',version,'huella_spki_sha256',huella_spki_sha256,
     'audiencia_despliegue',audiencia_despliegue,'suite',suite)
    FROM vec_autorizacion_atestada_v3.raiz_confianza_version WHERE clave_id='r1'));
$f$;
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
 (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
SELECT 'r1',1,s,encode(sha256(s),'hex'),clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day',
       'VEC-AD-3-COSE-EDDSA-1','vec:desarrollo:contratacion-temporal:atestacion:v3','acto_raiz_1'
  FROM (SELECT decode(repeat('01',44),'hex') AS s) x;
SELECT prueba_ad3_69.configuracion(1,clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_69.clave(n,a.nombre,clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day')
  FROM (SELECT row_number() OVER (ORDER BY consumidor,i)::int AS n,nombre FROM prueba_ad3_69.audiencia) a;
SQL
ok 'fixture sintético: raíz compartida, configuración c1 y 13 claves (CT5 + B2-8)'
escalas=$(valor "SELECT c.revision||'/'||m.maxima||':'||(c.revision < m.maxima)
                  FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c,
                       (SELECT max(revision_gobierno) AS maxima FROM vec_autorizacion_atestada_v3.clave_capacidad_version) m
                 WHERE c.control_id")
[[ $escalas == *':true' ]] || fallo "el checkpoint ya alcanza las claves ($escalas); el ensayo no reproduce el desfase"
ok "checkpoint sin ajustar por debajo de las claves (revisión/máxima de clave: ${escalas%:*})"

huella_v1() {
  valor "SELECT string_agg(p.proname||':'||md5(pg_get_functiondef(p.oid))||':'||md5(coalesce(p.proacl::text,'')),'|' ORDER BY p.proname)
           FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_interna_v1','leer_configuracion_interna_v1')"
}
existe_v2() {
  valor "SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_interna_v2','leer_configuracion_interna_v2')"
}
huella_v2() {
  valor "SELECT string_agg(p.proname||':'||md5(pg_get_functiondef(p.oid))||':'||md5(coalesce(p.proacl::text,'')),'|' ORDER BY p.proname)
           FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_interna_v2','leer_configuracion_interna_v2')"
}
v1_antes=$(huella_v1)
[[ $(printf '%s\n' "$v1_antes" | tr '|' '\n' | wc -l) == 2 ]] || fallo 'faltan funciones AD3-50a/53a'

# --- ROLLBACK sin rastro, UP y segunda aplicación.
sed '$s/^COMMIT;$/ROLLBACK;/' "$up" | sql >/dev/null
[[ $(existe_v2) == 0 && $(huella_v1) == "$v1_antes" ]] || fallo 'ROLLBACK de AD3-69 dejó rastro'
ok 'ROLLBACK de AD3-69 sin rastro'
sql < "$up" >/dev/null
[[ $(existe_v2) == 2 && $(huella_v1) == "$v1_antes" ]] || fallo 'UP de AD3-69'
v2_primera=$(huella_v2)
if sql < "$up" >/dev/null 2>&1; then fallo 'segunda aplicación de AD3-69 aceptada'; fi
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'segunda aplicación alteró funciones'
ok 'UP aplicado; segunda aplicación rechazada; funciones antiguas con md5 idéntico'

# --- ACL: solo propietario y rol de preflight ejecutan; nadie lee tablas.
acl=$(valor "SELECT string_agg(r||':'||has_function_privilege(r,f,'EXECUTE'),',' ORDER BY r,f)
  FROM unnest(ARRAY['vec_autorizacion_atestada_v3_preflight_interno','vec_autorizacion_atestada_v3_consumidor',
                    'vec_autorizacion_atestada_v3_emisor','vec_ad3_69_rol_extra']) r,
       unnest(ARRAY['vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)',
                    'vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)']) f")
[[ $acl == 'vec_ad3_69_rol_extra:false,vec_ad3_69_rol_extra:false,vec_autorizacion_atestada_v3_consumidor:false,vec_autorizacion_atestada_v3_consumidor:false,vec_autorizacion_atestada_v3_emisor:false,vec_autorizacion_atestada_v3_emisor:false,vec_autorizacion_atestada_v3_preflight_interno:true,vec_autorizacion_atestada_v3_preflight_interno:true' ]] \
  || fallo "ACL divergente: $acl"
[[ $(valor "SELECT count(*) FROM pg_class c WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace AND c.relkind='r'
            AND has_table_privilege('vec_interno_preflight_v3_desarrollo',c.oid,'SELECT,INSERT,UPDATE,DELETE')") == 0 ]] \
  || fallo 'el LOGIN de preflight tiene acceso directo a tablas'
[[ $(valor "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
            WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
              AND p.proname IN ('comprobar_material_emision_interna_v2','leer_configuracion_interna_v2')
              AND (a.grantee=0 OR a.is_grantable AND a.grantee<>p.proowner)") == 0 ]] || fallo 'ACL con PUBLIC o GRANT OPTION'
[[ $(valor "SELECT bool_and(prosecdef AND provolatile='s' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
                    AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole)
              FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
               AND proname IN ('comprobar_material_emision_interna_v2','leer_configuracion_interna_v2')") == t ]] \
  || fallo 'definidora, volatilidad o entorno divergentes'
ok 'ACL exacta: solo preflight_interno ejecuta; sin PUBLIC; sin acceso a tablas; SECURITY DEFINER STABLE search_path=pg_catalog'

# Arnés: ejecuta como el LOGIN indicado los casos positivos y negativos con el
# material vigente que calcula el DBA. $1 = login; $2 = preparación DBA (SQL,
# dentro de la transacción); $3 = cuerpo plpgsql con ct/b2/n disponibles.
# Todo se revierte salvo que $4 sea COMMIT.
arnes() {
  local login=$1 preparacion=$2 cuerpo=${3//%/%%} fin=${4:-ROLLBACK}
  sql <<SQL
BEGIN;
$preparacion
SELECT prueba_ad3_69.material('ct')::text AS ct, prueba_ad3_69.material('personal_b2')::text AS b2 \gset
SET SESSION AUTHORIZATION $login;
SELECT format(\$fmt\$DO \$t\$
DECLARE ct jsonb := %L; b2 jsonb := %L; n int := 0; r record; j jsonb;
BEGIN
$cuerpo
END \$t\$;\$fmt\$, :'ct', :'b2') \gexec
RESET SESSION AUTHORIZATION;
$fin;
SQL
}
# Cuerpo reutilizable: todos los casos de la tabla deben dar 42501 en ambas funciones.
rechazos() {
  cat <<'PL'
FOR r IN SELECT * FROM (VALUES
PL
  printf '%s\n' "$1"
  cat <<'PL'
) AS v(nombre, consumidor, material) LOOP
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(r.consumidor, r.material);
    RAISE EXCEPTION 'sonda aceptó: %', r.nombre USING ERRCODE = 'P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1;
  END;
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(r.consumidor, r.material);
    RAISE EXCEPTION 'lectura aceptó: %', r.nombre USING ERRCODE = 'P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1;
  END;
END LOOP;
RAISE NOTICE 'rechazos 42501: %', n;
PL
}
positivos='
IF vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2($$ct$$, ct) IS NOT TRUE
   OR vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2($$personal_b2$$, b2) IS NOT TRUE
THEN RAISE EXCEPTION $$positivo sonda$$; END IF;
j := vec_autorizacion_atestada_v3.leer_configuracion_interna_v2($$ct$$, ct);
IF j->>$$revision$$ IS DISTINCT FROM ct->$$configuracion$$->>$$revision$$
   OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(j) k)
      IS DISTINCT FROM ARRAY[$$expira_en$$,$$huella_configuracion_sha256$$,$$publicada_en$$,$$revision$$,$$secuencia$$]
   OR vec_autorizacion_atestada_v3.leer_configuracion_interna_v2($$personal_b2$$, b2) IS DISTINCT FROM j
THEN RAISE EXCEPTION $$positivo lectura$$; END IF;'

# --- Positivos CT5 y B2-8 con el checkpoint sin ajustar. v1, intacta, sigue
# comparando revision_gobierno con el contador y rechaza CT (el fallo que v2
# corrige); con el contador adelantado por el DBA (ROLLBACK) acepta CT como
# antes. v1 nunca acepta B2.
salida=$(arnes vec_interno_preflight_v3_desarrollo '' "$positivos
BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(ct);
  RAISE EXCEPTION \$\$v1 aceptó CT sin checkpoint\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN n := n + 1; END;
BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(ct);
  RAISE EXCEPTION \$\$v1 leyó CT sin checkpoint\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN n := n + 1; END;
RAISE NOTICE \$\$v1 rechaza CT sin checkpoint: %\$\$, n;" 2>&1 || true)
[[ $salida == *'v1 rechaza CT sin checkpoint: 2'* ]] || fallo "positivos v2 / contraste v1: $salida"
arnes vec_interno_preflight_v3_desarrollo \
  "UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
      SET revision=(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version) WHERE control_id;" \
  "IF vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(ct) IS NOT TRUE
   OR (vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(ct)->>\$\$revision\$\$) IS DISTINCT FROM \$\$c1\$\$
THEN RAISE EXCEPTION \$\$v1 CT\$\$; END IF;
BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(b2);
  RAISE EXCEPTION \$\$v1 aceptó B2\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN NULL; END;" >/dev/null
ok 'positivos: CT5 y B2-8 en sonda y lectura v2 sin tocar el checkpoint; v1 intacta rechaza CT sin checkpoint, lo acepta con él adelantado y rechaza B2'

# --- Negativos estáticos de discriminador y material.
casos=$(cat <<'CASOS'
 ('consumidor desconocido','rrhh',ct),
 ('consumidor en mayúsculas','CT',ct),
 ('consumidor vacío','',ct),
 ('consumidor con espacio','personal_b2 ',b2),
 ('consumidor NULL',NULL::text,ct),
 ('material NULL','ct',NULL::jsonb),
 ('material null JSON','ct','null'::jsonb),
 ('mezcla: material B2 como ct','ct',b2),
 ('mezcla: material CT como personal_b2','personal_b2',ct),
 ('mezcla: clave CT dentro de B2','personal_b2',jsonb_set(b2,'{claves,0}',ct->'claves'->0)),
 ('mezcla: clave B2 dentro de CT','ct',jsonb_set(ct,'{claves,4}',b2->'claves'->7)),
 ('duplicado B2','personal_b2',jsonb_set(b2,'{claves,1}',b2->'claves'->0)),
 ('duplicado CT','ct',jsonb_set(ct,'{claves,4}',ct->'claves'->3)),
 ('orden alterado B2','personal_b2',jsonb_set(jsonb_set(b2,'{claves,0}',b2->'claves'->1),'{claves,1}',b2->'claves'->0)),
 ('ausencia B2 (7)','personal_b2',b2 #- '{claves,7}'),
 ('ausencia CT (4)','ct',ct #- '{claves,4}'),
 ('exceso B2 (9)','personal_b2',jsonb_set(b2,'{claves}',(b2->'claves')||jsonb_build_array(b2->'claves'->0))),
 ('clave NULL','personal_b2',jsonb_set(b2,'{claves,2}','null'::jsonb)),
 ('lista de audiencias del llamante','personal_b2',b2||jsonb_build_object('audiencias',jsonb_build_array('x'))),
 ('campo extra en clave','ct',jsonb_set(ct,'{claves,0,audiencias}','["x"]'::jsonb)),
 ('audiencia de clave cambiada','personal_b2',jsonb_set(b2,'{claves,3,audiencia_consumo}','"vec_personal.alta_ejercicio.v1"'::jsonb)),
 ('huella secreto falsa','personal_b2',jsonb_set(b2,'{claves,5,huella_secreto_sha256}',to_jsonb(repeat('e',64)))),
 ('huella gobierno falsa','ct',jsonb_set(ct,'{claves,2,huella_gobierno_sha256}',to_jsonb(repeat('d',64)))),
 ('huella configuración falsa','personal_b2',jsonb_set(b2,'{configuracion,huella_configuracion_sha256}',to_jsonb(repeat('f',64)))),
 ('huella raíz falsa','ct',jsonb_set(ct,'{raiz,huella_spki_sha256}',to_jsonb(repeat('c',64)))),
 ('huella no hexadecimal','personal_b2',jsonb_set(b2,'{claves,0,huella_secreto_sha256}','"ZZ"'::jsonb)),
 ('emisor falso','personal_b2',jsonb_set(b2,'{claves,6,emisor_id}','"otro"'::jsonb)),
 ('versión de clave falsa','personal_b2',jsonb_set(b2,'{claves,4,version}','99'::jsonb)),
 ('revisión de gobierno falsa','ct',jsonb_set(ct,'{claves,1,revision_gobierno}','99'::jsonb)),
 ('secuencia de configuración falsa','ct',jsonb_set(ct,'{configuracion,secuencia}','2'::jsonb)),
 ('raíz con audiencia B2 propia','personal_b2',jsonb_set(b2,'{raiz,audiencia_despliegue}','"vec:interno:personal:registro-empleado:atestacion:v3"'::jsonb)),
 ('suite distinta','ct',jsonb_set(ct,'{raiz,suite}','"OTRA"'::jsonb)),
 ('raíz inexistente','ct',jsonb_set(ct,'{raiz,version}','2'::jsonb))
CASOS
)
salida=$(arnes vec_interno_preflight_v3_desarrollo '' "$(rechazos "$casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 66'* ]] || fallo "negativos estáticos: $salida"
ok 'negativos estáticos: 33 casos x 2 funciones = 66 rechazos 42501'

# --- Login incorrecto: superusuario, otro LOGIN con la misma membresía y el
# LOGIN nominal con una membresía adicional. Positivos esperados como rechazo.
login_casos=" ('positivo CT','ct',ct), ('positivo B2','personal_b2',b2)"
for login in postgres vec_ad3_69_otro_login; do
  salida=$(arnes "$login" '' "$(rechazos "$login_casos")" 2>&1 || true)
  [[ $salida == *'rechazos 42501: 4'* ]] || fallo "login $login: $salida"
done
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  'GRANT vec_ad3_69_rol_extra TO vec_interno_preflight_v3_desarrollo;' "$(rechazos "$login_casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "membresía adicional: $salida"
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  'REVOKE vec_autorizacion_atestada_v3_preflight_interno FROM vec_interno_preflight_v3_desarrollo;
   GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;' \
  "$(rechazos "$login_casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "membresía con SET: $salida"
ok 'login incorrecto: superusuario, otro LOGIN, membresía extra y membresía con SET rechazados'

# --- Sin acceso directo a tablas ni secretos desde el LOGIN nominal.
for consulta in \
  'SELECT secreto_hmac FROM vec_autorizacion_atestada_v3.clave_capacidad_version' \
  'SELECT * FROM vec_autorizacion_atestada_v3.configuracion_confianza_version' \
  'SELECT * FROM vec_autorizacion_atestada_v3.checkpoint_gobierno' \
  'SELECT * FROM prueba_ad3_69.audiencia'; do
  if printf '%s;\n' "$consulta" | sql_como vec_interno_preflight_v3_desarrollo >/dev/null 2>&1; then
    fallo "acceso directo permitido: $consulta"
  fi
done
ok 'LOGIN nominal sin SELECT sobre tablas de gobierno ni secretos'

# --- Revocación de una clave B2: B2 rechazado, CT intacto (aislamiento).
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad (clave_id,version,revocada_en,motivo_catalogado_ref,acto_ref)
   SELECT clave_id,version,clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_b2'
     FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_personal.registro_empleado.catalogo.retirar.v1';" \
  "IF vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(\$\$ct\$\$, ct) IS NOT TRUE THEN RAISE EXCEPTION \$\$CT afectado\$\$; END IF;
$(rechazos " ('B2 con clave revocada','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 2'* ]] || fallo "revocación de clave: $salida"
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion (configuracion_revision,revocada_en,motivo_catalogado_ref,acto_ref)
   VALUES ('c1',clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_c1');" \
  "$(rechazos " ('CT con configuración revocada','ct',ct), ('B2 con configuración revocada','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "revocación de configuración: $salida"
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz (raiz_clave_id,raiz_version,revocada_en,motivo_catalogado_ref,acto_ref)
   VALUES ('r1',1,clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_r1');" \
  "$(rechazos " ('CT con raíz revocada','ct',ct), ('B2 con raíz revocada','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "revocación de raíz: $salida"
ok 'revocación de clave B2 (CT intacto), de configuración y de raíz rechazadas'

# --- Rotación de puntero: una clave B2 nueva deja obsoleto el material anterior.
salida=$(sql 2>&1 <<'SQL' || printf 'PSQL_FALLO\n'
BEGIN;
SELECT prueba_ad3_69.material('personal_b2')::text AS b2_viejo \gset
SELECT prueba_ad3_69.clave(40,'vec_personal.registro_empleado.hecho.v1',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_69.material('personal_b2')::text AS b2_nuevo \gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE viejo jsonb := %L; nuevo jsonb := %L;
BEGIN
  IF vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', nuevo) IS NOT TRUE
     OR (vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2', nuevo)->>'revision') <> 'c1'
  THEN RAISE EXCEPTION 'material rotado rechazado'; END IF;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', viejo);
    RAISE EXCEPTION 'puntero anterior aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN RAISE NOTICE 'puntero anterior rechazado'; END;
END $t$;$fmt$, :'b2_viejo', :'b2_nuevo') \gexec
RESET SESSION AUTHORIZATION;
ROLLBACK;
SQL
)
[[ $salida == *'puntero anterior rechazado'* ]] || fallo "rotación de puntero: $salida"
ok 'rotación de puntero B2: material nuevo aceptado, anterior rechazado'

# --- Caducidad de clave y de configuración.
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "SELECT prueba_ad3_69.clave(41,'vec_personal.registro_empleado.ficha.v1',clock_timestamp()-interval '2 day',clock_timestamp()-interval '1 day');" \
  "$(rechazos " ('B2 con clave caducada','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 2'* ]] || fallo "caducidad de clave: $salida"
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "SELECT prueba_ad3_69.configuracion(2,clock_timestamp()-interval '2 day',clock_timestamp()-interval '1 day');" \
  "$(rechazos " ('CT con configuración vigente caducada','ct',ct), ('B2 con configuración vigente caducada','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "caducidad de configuración: $salida"
# Caducidad durante una misma sentencia: la clave vence a los 2 s y la
# comprobación ocurre tras pg_sleep(3) dentro del mismo DO.
salida=$(arnes vec_interno_preflight_v3_desarrollo \
  "SELECT prueba_ad3_69.clave(41,'vec_personal.registro_empleado.ficha.v1',clock_timestamp()-interval '1 day',clock_timestamp()+interval '2 second');" \
  "PERFORM pg_catalog.pg_sleep(3);
$(rechazos " ('B2 con clave caducada durante la sentencia','personal_b2',b2)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 2'* ]] || fallo "caducidad durante la sentencia: $salida"
ok 'caducidad de clave B2 (también durante la sentencia) y de configuración vigente rechazadas'

# --- Renovación de configuración: lectura desde la previa devuelve la nueva;
# después, retroceso por checkpoint (secuencia y raíz mínimas) rechazado.
salida=$(sql 2>&1 <<'SQL' || printf 'PSQL_FALLO\n'
BEGIN;
SELECT prueba_ad3_69.material('personal_b2')::text AS b2_c1 \gset
SELECT prueba_ad3_69.configuracion(3,clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_69.material('personal_b2')::text AS b2_c3, prueba_ad3_69.material('ct')::text AS ct_c3 \gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE previa jsonb := %L; actual jsonb := %L; ct jsonb := %L;
BEGIN
  IF (vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2', previa)->>'revision') <> 'c3'
     OR (vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2', actual)->>'revision') <> 'c3'
     OR vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', actual) IS NOT TRUE
     OR vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct', ct) IS NOT TRUE
  THEN RAISE EXCEPTION 'renovación'; END IF;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', previa);
    RAISE EXCEPTION 'sonda aceptó configuración ya sustituida' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN RAISE NOTICE 'configuración sustituida rechazada en sonda'; END;
END $t$;$fmt$, :'b2_c1', :'b2_c3', :'ct_c3') \gexec
RESET SESSION AUTHORIZATION;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=4 WHERE control_id;
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE b2 jsonb := %L; ct jsonb := %L; n int := 0;
BEGIN
  BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2', b2);
    RAISE EXCEPTION 'retroceso aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct', ct);
    RAISE EXCEPTION 'retroceso aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  RAISE NOTICE 'retroceso configuración rechazado: %%', n;
END $t$;$fmt$, :'b2_c3', :'ct_c3') \gexec
RESET SESSION AUTHORIZATION;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=0, raiz_version_minima=2 WHERE control_id;
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE b2 jsonb := %L; ct jsonb := %L; n int := 0;
BEGIN
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', b2);
    RAISE EXCEPTION 'raíz retrocedida aceptada' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct', ct);
    RAISE EXCEPTION 'raíz retrocedida aceptada' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  RAISE NOTICE 'retroceso raíz rechazado: %%', n;
END $t$;$fmt$, :'b2_c3', :'ct_c3') \gexec
RESET SESSION AUTHORIZATION;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET raiz_version_minima=0 WHERE control_id;
SELECT (revision < (SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version))::text AS por_debajo
  FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id \gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE b2 jsonb := %L; ct jsonb := %L;
BEGIN
  IF %L::boolean IS NOT TRUE
     OR vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2', b2) IS NOT TRUE
     OR (vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2', b2)->>'revision') <> 'c3'
     OR vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct', ct) IS NOT TRUE
     OR (vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct', ct)->>'revision') <> 'c3'
  THEN RAISE EXCEPTION 'mínimos restituidos'; END IF;
  RAISE NOTICE 'claves por encima del contador aceptadas con mínimos restituidos';
END $t$;$fmt$, :'b2_c3', :'ct_c3', :'por_debajo') \gexec
RESET SESSION AUTHORIZATION;
ROLLBACK;
SQL
)
[[ $salida == *'configuración sustituida rechazada en sonda'* && $salida == *'retroceso configuración rechazado: 2'* \
   && $salida == *'retroceso raíz rechazado: 2'* && $salida == *'claves por encima del contador aceptadas'* ]] \
  || fallo "renovación/retroceso: $salida"
ok 'renovación c1→c3 leída; retroceso por mínimos de configuración y raíz rechazado; con mínimos restituidos, claves por encima del contador aceptadas'

[[ $(huella_v1) == "$v1_antes" ]] || fallo 'funciones antiguas alteradas durante el ensayo'

# --- UP/DOWN/UP y reinicio.
sql < "$down" >/dev/null
[[ $(existe_v2) == 0 && $(huella_v1) == "$v1_antes" ]] || fallo 'DOWN'
if sql < "$down" >/dev/null 2>&1; then fallo 'segundo DOWN aceptado'; fi
sql < "$up" >/dev/null
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'UP tras DOWN no reproduce la postimagen'
arnes vec_interno_preflight_v3_desarrollo '' "$positivos" >/dev/null
ok 'UP/DOWN/UP: postimagen idéntica (md5 de definición y ACL), v1 intacta, positivos de nuevo'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'estado perdido tras reinicio'
arnes vec_interno_preflight_v3_desarrollo '' "$positivos" >/dev/null
ok 'funciones, ACL y positivos persisten tras reiniciar PostgreSQL'
printf 'md5 v1 (antes=después): %s\n' "$v1_antes"
printf 'PG18.4: AD3-69 sobre cadena canónica AD3 1/2/50a/53a con fixture sintético; NO acredita firma COSE, PDP ni cadena AD3 íntegra.\n'
