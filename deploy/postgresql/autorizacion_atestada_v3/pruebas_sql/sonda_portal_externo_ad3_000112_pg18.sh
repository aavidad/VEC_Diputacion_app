#!/usr/bin/env bash
# Ensayo PG18.4 desechable de AD3-112 (sonda y lectura de gobierno V3 del
# proceso del portal externo, con consumidores cerrados 'usuarios_preferencias',
# 'usuarios_correos', 'usuarios_imagen', 'mi_bolsa' y 'portal_candidato').
#
# Base: la misma cadena canónica que el ensayo de AD3-69 (ContextoActor 1/2 +
# Autorización 1–7 + AD3 1/2/50a/53a/69). La restricción de audiencias de
# AD3-002 se amplía como DBA a las audiencias externas, sin ejecutar las
# migraciones de consumidores. Gobierno, raíz y claves son sintéticos.
#
# Comprueba: ROLLBACK sin rastro, UP, segunda aplicación rechazada, ACL exacta
# (solo el rol de preflight externo ejecuta; el interno no; el externo no
# ejecuta nada más ni lee tablas), positivos de los cinco consumidores,
# negativos (consumidores internos 'ct' y 'personal_b2', desconocidos, mezcla,
# duplicado, orden, ausencia, exceso, huellas falsas, revisión falsificada,
# login incorrecto, LOGIN interno, revocación, rotación, caducidad, retroceso
# por mínimos), funciones internas intactas (md5 antes/después), UP/DOWN/UP,
# DOWN rechazado con un LOGIN miembro, y reinicio.
# No acredita firma COSE, decisión PDP ni la cadena AD3 íntegra.
set -Eeuo pipefail
trap 'printf "AD3-112: orden fallida en la línea %s\n" "$LINENO" >&2' ERR
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-ad3-112-sonda-${BASHPID}"
mig="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones"
up="$mig/000112_preflight_gobierno_portal_externo.up.sql"
down="$mig/000112_preflight_gobierno_portal_externo.down.sql"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
sql() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres; }
sql_como() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { sql < "$repo_dir/$1" >/dev/null; }
fallo() { printf 'AD3-112: %s\n' "$*" >&2; exit 1; }
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
CREATE ROLE vec_ad3_112_migrador LOGIN NOINHERIT;
GRANT CONNECT ON DATABASE postgres TO vec_ad3_112_migrador;
GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_112_migrador
  WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL
a3=deploy/postgresql/autorizacion_atestada_v3/migraciones
sql_como vec_ad3_112_migrador < "$repo_dir/$a3/000001_gobierno_y_registro_v3.up.sql" >/dev/null
sql_como vec_ad3_112_migrador < "$repo_dir/$a3/000002_consumidor_capacidad_v3.up.sql" >/dev/null
archivo "$a3/000050a_preflight_material_interno.up.sql"
archivo "$a3/000053a_lectura_configuracion_interna.up.sql"
archivo "$a3/000069_sonda_lectura_gobierno_por_consumidor.up.sql"
ok 'cadena canónica: AD3 1/2/50a/53a/69 instaladas'

# LOGIN nominal y dos LOGIN de contraste. Ninguno es superusuario.
sql >/dev/null <<'SQL'
CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_ad3_112_rol_extra NOLOGIN;
GRANT CONNECT ON DATABASE postgres TO vec_interno_preflight_v3_desarrollo;
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
  ADD CONSTRAINT prueba_ad3_112_audiencias CHECK (audiencia_consumo = ANY (ARRAY[
   'vec_contratacion_temporal.confirmar_alta_atestada.v1',
   'vec_personal.alta_ejercicio.v1','vec_personal.lectura_incorporacion.v2',
   'vec_contratacion_temporal.incorporacion_ejercicio.v2',
   'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
   'vec_usuarios.preferencias.consultar.externa_personal.v1',
   'vec_usuarios.preferencias.actualizar.externa_personal.v1',
   'vec_usuarios.correos.consultar.externa_personal.v1',
   'vec_usuarios.correos.anadir.externa_personal.v1',
   'vec_usuarios.correos.reenviar.externa_personal.v1',
   'vec_usuarios.correos.verificar.externa_personal.v1',
   'vec_usuarios.correos.activar.externa_personal.v1',
   'vec_usuarios.correos.retirar.externa_personal.v1',
   'vec_usuarios.imagen.consultar.externa_personal.v1',
   'vec_usuarios.imagen.actualizar.externa_personal.v1',
   'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
   'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',
   'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1',
   'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',
   'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1',
   'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1']));
CREATE SCHEMA prueba_ad3_112;
REVOKE ALL ON SCHEMA prueba_ad3_112 FROM PUBLIC;
CREATE TABLE prueba_ad3_112.audiencia(consumidor text, i int, nombre text, PRIMARY KEY (consumidor,i));
INSERT INTO prueba_ad3_112.audiencia VALUES
 ('ct',1,'vec_personal.alta_ejercicio.v1'),
 ('ct',2,'vec_personal.lectura_incorporacion.v2'),
 ('ct',3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 ('ct',4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 ('ct',5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'),
 ('usuarios_preferencias',1,'vec_usuarios.preferencias.consultar.externa_personal.v1'),
 ('usuarios_preferencias',2,'vec_usuarios.preferencias.actualizar.externa_personal.v1'),
 ('usuarios_correos',1,'vec_usuarios.correos.consultar.externa_personal.v1'),
 ('usuarios_correos',2,'vec_usuarios.correos.anadir.externa_personal.v1'),
 ('usuarios_correos',3,'vec_usuarios.correos.reenviar.externa_personal.v1'),
 ('usuarios_correos',4,'vec_usuarios.correos.verificar.externa_personal.v1'),
 ('usuarios_correos',5,'vec_usuarios.correos.activar.externa_personal.v1'),
 ('usuarios_correos',6,'vec_usuarios.correos.retirar.externa_personal.v1'),
 ('usuarios_imagen',1,'vec_usuarios.imagen.consultar.externa_personal.v1'),
 ('usuarios_imagen',2,'vec_usuarios.imagen.actualizar.externa_personal.v1'),
 ('mi_bolsa',1,'vec.bolsa.mi-bolsa.v1'),
 ('mi_bolsa',2,'vec.bolsa.mi-bolsa.historial.v1'),
 ('portal_candidato',1,'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1'),
 ('portal_candidato',2,'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1'),
 ('portal_candidato',3,'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'),
 ('portal_candidato',4,'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'),
 ('portal_candidato',5,'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1');
-- Clave sintética n: versión, revisión y orden de puntero = n; secreto de
-- relleno. No ajusta el checkpoint: solo lo mueven los triggers de AD3-2.
CREATE FUNCTION prueba_ad3_112.clave(n int, audiencia text, desde timestamptz, hasta timestamptz)
RETURNS void LANGUAGE sql AS $f$
 INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
  (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,
   emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
 SELECT 'k'||n,n,n,repeat('a',64),s,encode(sha256(s),'hex'),'emisor_sintetico',audiencia,desde,hasta,'acto_clave_'||n
   FROM (SELECT decode(repeat(lpad(to_hex(n),2,'0'),32),'hex') AS s) x;
 INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden,clave_id,version,establecida_en,acto_ref)
 VALUES (n,'k'||n,n,clock_timestamp()-interval '1 second','acto_puntero_clave_'||n);
$f$;
CREATE FUNCTION prueba_ad3_112.configuracion(n int, desde timestamptz, hasta timestamptz)
RETURNS void LANGUAGE sql AS $f$
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
  (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES ('c'||n,n,repeat(lpad(to_hex(n),2,'0'),32),desde,hasta,'acto_configuracion_'||n);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES ('c'||n,'r1',1);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual (orden,configuracion_revision,establecida_en,acto_ref)
 VALUES (n,'c'||n,clock_timestamp()-interval '1 second','acto_puntero_configuracion_'||n);
$f$;
-- Material tal como lo enviaría el llamante: claves vigentes en el orden fijado.
CREATE FUNCTION prueba_ad3_112.material(p_consumidor text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object(
  'claves',(SELECT jsonb_agg(jsonb_build_object('audiencia_consumo',k.audiencia_consumo,'clave_id',k.clave_id,
     'version',k.version,'revision_gobierno',k.revision_gobierno,'huella_gobierno_sha256',k.huella_gobierno_sha256,
     'huella_secreto_sha256',k.huella_secreto_sha256,'emisor_id',k.emisor_id) ORDER BY a.i)
    FROM prueba_ad3_112.audiencia a
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
SELECT prueba_ad3_112.configuracion(1,clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_112.clave(n,a.nombre,clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day')
  FROM (SELECT row_number() OVER (ORDER BY consumidor,i)::int AS n,nombre FROM prueba_ad3_112.audiencia) a;
SQL
ok 'fixture sintético: raíz compartida, configuración c1 y 22 claves (CT5 + 17 externas)'
escalas=$(valor "SELECT c.revision||'/'||m.maxima||':'||(c.revision < m.maxima)
                  FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c,
                       (SELECT max(revision_gobierno) AS maxima FROM vec_autorizacion_atestada_v3.clave_capacidad_version) m
                 WHERE c.control_id")
[[ $escalas == *':true' ]] || fallo "el checkpoint ya alcanza las claves ($escalas); el ensayo no reproduce el desfase"
ok "checkpoint sin ajustar por debajo de las claves (revisión/máxima de clave: ${escalas%:*})"

huella_v1() {
  valor "SELECT string_agg(p.proname||':'||md5(pg_get_functiondef(p.oid))||':'||md5(coalesce(p.proacl::text,'')),'|' ORDER BY p.proname)
           FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_interna_v1','leer_configuracion_interna_v1',
                              'comprobar_material_emision_interna_v2','leer_configuracion_interna_v2')"
}
existe_v2() {
  valor "SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_externa_v1','leer_configuracion_externa_v1')"
}
huella_v2() {
  valor "SELECT string_agg(p.proname||':'||md5(pg_get_functiondef(p.oid))||':'||md5(coalesce(p.proacl::text,'')),'|' ORDER BY p.proname)
           FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname IN ('comprobar_material_emision_externa_v1','leer_configuracion_externa_v1')"
}
v1_antes=$(huella_v1)
[[ $(printf '%s\n' "$v1_antes" | tr '|' '\n' | wc -l) == 4 ]] || fallo 'faltan funciones internas AD3-50a/53a/69'

# --- ROLLBACK sin rastro, UP y segunda aplicación.
rol_externo() { valor "SELECT count(*) FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_preflight_externo'"; }
sed '$s/^COMMIT;$/ROLLBACK;/' "$up" | sql >/dev/null
[[ $(existe_v2) == 0 && $(huella_v1) == "$v1_antes" && $(rol_externo) == 0 ]] || fallo 'ROLLBACK de AD3-112 dejó rastro'
ok 'ROLLBACK de AD3-112 sin rastro (ni funciones ni rol)'
sql < "$up" >/dev/null
[[ $(existe_v2) == 2 && $(huella_v1) == "$v1_antes" && $(rol_externo) == 1 ]] || fallo 'UP de AD3-112'
v2_primera=$(huella_v2)
if sql < "$up" >/dev/null 2>&1; then fallo 'segunda aplicación de AD3-112 aceptada'; fi
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'segunda aplicación alteró funciones'
ok 'UP aplicado; segunda aplicación rechazada; funciones internas con md5 idéntico'

# LOGIN nominal externo y uno de contraste, ahora que existe su grupo.
sql >/dev/null <<'SQL'
CREATE ROLE vec_externo_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_externo TO vec_externo_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_ad3_112_otro_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_externo TO vec_ad3_112_otro_login
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_externo_preflight_v3_desarrollo, vec_ad3_112_otro_login;
SQL

# --- ACL: solo propietario y preflight externo ejecutan; el interno no.
acl=$(valor "SELECT string_agg(r||':'||has_function_privilege(r,f,'EXECUTE'),',' ORDER BY r,f)
  FROM unnest(ARRAY['vec_autorizacion_atestada_v3_preflight_externo','vec_autorizacion_atestada_v3_preflight_interno',
                    'vec_autorizacion_atestada_v3_consumidor','vec_autorizacion_atestada_v3_emisor','vec_ad3_112_rol_extra']) r,
       unnest(ARRAY['vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)',
                    'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)']) f")
[[ $acl == 'vec_ad3_112_rol_extra:false,vec_ad3_112_rol_extra:false,vec_autorizacion_atestada_v3_consumidor:false,vec_autorizacion_atestada_v3_consumidor:false,vec_autorizacion_atestada_v3_emisor:false,vec_autorizacion_atestada_v3_emisor:false,vec_autorizacion_atestada_v3_preflight_externo:true,vec_autorizacion_atestada_v3_preflight_externo:true,vec_autorizacion_atestada_v3_preflight_interno:false,vec_autorizacion_atestada_v3_preflight_interno:false' ]] \
  || fallo "ACL divergente: $acl"
[[ $(valor "SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
            AND p.proname NOT IN ('comprobar_material_emision_externa_v1','leer_configuracion_externa_v1')
            AND has_function_privilege('vec_externo_preflight_v3_desarrollo',p.oid,'EXECUTE')") == 0 ]] \
  || fallo 'el LOGIN externo ejecuta otras funciones del esquema'
[[ $(valor "SELECT count(*) FROM pg_class c WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace AND c.relkind='r'
            AND has_table_privilege('vec_externo_preflight_v3_desarrollo',c.oid,'SELECT,INSERT,UPDATE,DELETE')") == 0 ]] \
  || fallo 'el LOGIN externo tiene acceso directo a tablas'
[[ $(valor "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
            WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
              AND p.proname IN ('comprobar_material_emision_externa_v1','leer_configuracion_externa_v1')
              AND (a.grantee=0 OR a.is_grantable AND a.grantee<>p.proowner)") == 0 ]] || fallo 'ACL con PUBLIC o GRANT OPTION'
[[ $(valor "SELECT bool_and(prosecdef AND provolatile='s' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
                    AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole)
              FROM pg_proc WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
               AND proname IN ('comprobar_material_emision_externa_v1','leer_configuracion_externa_v1')") == t ]] \
  || fallo 'definidora, volatilidad o entorno divergentes'
ok 'ACL exacta: solo preflight_externo ejecuta (el interno no); el LOGIN externo no ejecuta nada más ni lee tablas'

# Arnés: $1 = login; $2 = preparación DBA; $3 = cuerpo plpgsql con a (correos),
# b (portal), p (preferencias), i (imagen), m (mi_bolsa), ct y n disponibles.
arnes() {
  local login=$1 preparacion=$2 cuerpo=${3//%/%%} fin=${4:-ROLLBACK}
  sql <<SQL
BEGIN;
$preparacion
SELECT prueba_ad3_112.material('usuarios_correos')::text AS a, prueba_ad3_112.material('portal_candidato')::text AS b,
       prueba_ad3_112.material('usuarios_preferencias')::text AS p, prueba_ad3_112.material('usuarios_imagen')::text AS i,
       prueba_ad3_112.material('mi_bolsa')::text AS m, prueba_ad3_112.material('ct')::text AS ct \gset
SET SESSION AUTHORIZATION $login;
SELECT format(\$fmt\$DO \$t\$
DECLARE a jsonb := %L; b jsonb := %L; p jsonb := %L; i jsonb := %L; m jsonb := %L; ct jsonb := %L; n int := 0; r record; j jsonb;
BEGIN
$cuerpo
END \$t\$;\$fmt\$, :'a', :'b', :'p', :'i', :'m', :'ct') \gexec
RESET SESSION AUTHORIZATION;
$fin;
SQL
}
rechazos() {
  cat <<'PL'
FOR r IN SELECT * FROM (VALUES
PL
  printf '%s\n' "$1"
  cat <<'PL'
) AS v(nombre, consumidor, material) LOOP
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(r.consumidor, r.material);
    RAISE EXCEPTION 'sonda aceptó: %', r.nombre USING ERRCODE = 'P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1;
  END;
  BEGIN
    PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(r.consumidor, r.material);
    RAISE EXCEPTION 'lectura aceptó: %', r.nombre USING ERRCODE = 'P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1;
  END;
END LOOP;
RAISE NOTICE 'rechazos 42501: %', n;
PL
}
positivos='
IF vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($$usuarios_correos$$, a) IS NOT TRUE
   OR vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($$portal_candidato$$, b) IS NOT TRUE
   OR vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($$usuarios_preferencias$$, p) IS NOT TRUE
   OR vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($$usuarios_imagen$$, i) IS NOT TRUE
   OR vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($$mi_bolsa$$, m) IS NOT TRUE
THEN RAISE EXCEPTION $$positivo sonda$$; END IF;
j := vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($$usuarios_correos$$, a);
IF j->>$$revision$$ IS DISTINCT FROM a->$$configuracion$$->>$$revision$$
   OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(j) k)
      IS DISTINCT FROM ARRAY[$$expira_en$$,$$huella_configuracion_sha256$$,$$publicada_en$$,$$revision$$,$$secuencia$$]
   OR vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($$portal_candidato$$, b) IS DISTINCT FROM j
   OR vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($$usuarios_preferencias$$, p) IS DISTINCT FROM j
   OR vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($$usuarios_imagen$$, i) IS DISTINCT FROM j
   OR vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($$mi_bolsa$$, m) IS DISTINCT FROM j
THEN RAISE EXCEPTION $$positivo lectura$$; END IF;'

arnes vec_externo_preflight_v3_desarrollo '' "$positivos" >/dev/null
ok 'positivos: los cinco consumidores externos en sonda y lectura, sin tocar el checkpoint'

# --- Negativos estáticos de discriminador y material.
casos=$(cat <<'CASOS'
 ('consumidor interno ct con su material','ct',ct),
 ('consumidor interno personal_b2','personal_b2',a),
 ('consumidor desconocido','rrhh',a),
 ('consumidor en mayúsculas','USUARIOS_CORREOS',a),
 ('consumidor vacío','',a),
 ('consumidor con espacio','portal_candidato ',b),
 ('consumidor NULL',NULL::text,a),
 ('material NULL','usuarios_correos',NULL::jsonb),
 ('material null JSON','usuarios_correos','null'::jsonb),
 ('material CT como usuarios_correos','usuarios_correos',ct),
 ('mezcla: material portal como correos','usuarios_correos',b),
 ('mezcla: material correos como portal','portal_candidato',a),
 ('mezcla: clave correos dentro de portal','portal_candidato',jsonb_set(b,'{claves,0}',a->'claves'->0)),
 ('mezcla: clave CT dentro de correos','usuarios_correos',jsonb_set(a,'{claves,5}',ct->'claves'->4)),
 ('duplicado portal','portal_candidato',jsonb_set(b,'{claves,1}',b->'claves'->0)),
 ('duplicado correos','usuarios_correos',jsonb_set(a,'{claves,5}',a->'claves'->3)),
 ('orden alterado portal','portal_candidato',jsonb_set(jsonb_set(b,'{claves,0}',b->'claves'->1),'{claves,1}',b->'claves'->0)),
 ('ausencia portal (4)','portal_candidato',b #- '{claves,4}'),
 ('ausencia correos (5)','usuarios_correos',a #- '{claves,5}'),
 ('exceso portal (6)','portal_candidato',jsonb_set(b,'{claves}',(b->'claves')||jsonb_build_array(b->'claves'->0))),
 ('clave NULL','portal_candidato',jsonb_set(b,'{claves,2}','null'::jsonb)),
 ('lista de audiencias del llamante','portal_candidato',b||jsonb_build_object('audiencias',jsonb_build_array('x'))),
 ('campo extra en clave','usuarios_correos',jsonb_set(a,'{claves,0,audiencias}','["x"]'::jsonb)),
 ('audiencia de clave cambiada','portal_candidato',jsonb_set(b,'{claves,3,audiencia_consumo}','"vec_usuarios.correos.consultar.externa_personal.v1"'::jsonb)),
 ('huella secreto falsa','portal_candidato',jsonb_set(b,'{claves,4,huella_secreto_sha256}',to_jsonb(repeat('e',64)))),
 ('huella gobierno falsa','usuarios_correos',jsonb_set(a,'{claves,2,huella_gobierno_sha256}',to_jsonb(repeat('d',64)))),
 ('huella configuración falsa','portal_candidato',jsonb_set(b,'{configuracion,huella_configuracion_sha256}',to_jsonb(repeat('f',64)))),
 ('huella raíz falsa','usuarios_correos',jsonb_set(a,'{raiz,huella_spki_sha256}',to_jsonb(repeat('c',64)))),
 ('huella no hexadecimal','portal_candidato',jsonb_set(b,'{claves,0,huella_secreto_sha256}','"ZZ"'::jsonb)),
 ('emisor falso','portal_candidato',jsonb_set(b,'{claves,3,emisor_id}','"otro"'::jsonb)),
 ('versión de clave falsa','portal_candidato',jsonb_set(b,'{claves,4,version}','99'::jsonb)),
 ('revisión de gobierno falsa','usuarios_correos',jsonb_set(a,'{claves,1,revision_gobierno}','99'::jsonb)),
 ('secuencia de configuración falsa','usuarios_correos',jsonb_set(a,'{configuracion,secuencia}','2'::jsonb)),
 ('raíz con audiencia propia','portal_candidato',jsonb_set(b,'{raiz,audiencia_despliegue}','"vec:externo:area-personal:atestacion:v3"'::jsonb)),
 ('suite distinta','usuarios_correos',jsonb_set(a,'{raiz,suite}','"OTRA"'::jsonb)),
 ('raíz inexistente','usuarios_correos',jsonb_set(a,'{raiz,version}','2'::jsonb))
CASOS
)
salida=$(arnes vec_externo_preflight_v3_desarrollo '' "$(rechazos "$casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 72'* ]] || fallo "negativos estáticos: $salida"
ok 'negativos estáticos: 36 casos x 2 funciones = 72 rechazos 42501 (incluidos consumidores internos)'

# --- Login incorrecto: superusuario, otro LOGIN con la misma membresía, LOGIN
# nominal con membresía extra o con SET, y el LOGIN interno.
login_casos=" ('positivo correos','usuarios_correos',a), ('positivo portal','portal_candidato',b)"
for login in postgres vec_ad3_112_otro_login vec_interno_preflight_v3_desarrollo; do
  salida=$(arnes "$login" '' "$(rechazos "$login_casos")" 2>&1 || true)
  [[ $salida == *'rechazos 42501: 4'* ]] || fallo "login $login: $salida"
done
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  'GRANT vec_ad3_112_rol_extra TO vec_externo_preflight_v3_desarrollo;' "$(rechazos "$login_casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "membresía adicional: $salida"
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  'REVOKE vec_autorizacion_atestada_v3_preflight_externo FROM vec_externo_preflight_v3_desarrollo;
   GRANT vec_autorizacion_atestada_v3_preflight_externo TO vec_externo_preflight_v3_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;' \
  "$(rechazos "$login_casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "membresía con SET: $salida"
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  'GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_externo_preflight_v3_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;' \
  "$(rechazos "$login_casos")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "LOGIN externo con el grupo interno: $salida"
ok 'login incorrecto: superusuario, otro LOGIN, LOGIN interno, membresía extra, con SET o con el grupo interno rechazados'

# --- Separación cruzada: el LOGIN externo no puede usar las funciones internas.
salida=$(arnes vec_externo_preflight_v3_desarrollo '' "
BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(\$\$ct\$\$, ct);
  RAISE EXCEPTION \$\$externo usó la sonda interna\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN n := n + 1; END;
BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(\$\$ct\$\$, ct);
  RAISE EXCEPTION \$\$externo usó la lectura interna\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN n := n + 1; END;
BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(ct);
  RAISE EXCEPTION \$\$externo usó la sonda interna v1\$\$ USING ERRCODE=\$\$P0001\$\$;
EXCEPTION WHEN SQLSTATE \$\$42501\$\$ THEN n := n + 1; END;
RAISE NOTICE \$\$funciones internas denegadas: %\$\$, n;" 2>&1 || true)
[[ $salida == *'funciones internas denegadas: 3'* ]] || fallo "separación cruzada: $salida"
ok 'separación cruzada: el LOGIN externo no ejecuta las funciones internas'

# --- Sin acceso directo a tablas ni secretos desde el LOGIN nominal.
for consulta in \
  'SELECT secreto_hmac FROM vec_autorizacion_atestada_v3.clave_capacidad_version' \
  'SELECT * FROM vec_autorizacion_atestada_v3.configuracion_confianza_version' \
  'SELECT * FROM vec_autorizacion_atestada_v3.checkpoint_gobierno' \
  'SELECT * FROM prueba_ad3_112.audiencia'; do
  if printf '%s;\n' "$consulta" | sql_como vec_externo_preflight_v3_desarrollo >/dev/null 2>&1; then
    fallo "acceso directo permitido: $consulta"
  fi
done
ok 'LOGIN nominal sin SELECT sobre tablas de gobierno ni secretos'

# --- Revocación de una clave del portal: portal rechazado, correos intacto.
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad (clave_id,version,revocada_en,motivo_catalogado_ref,acto_ref)
   SELECT clave_id,version,clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_portal'
     FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1';" \
  "IF vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(\$\$usuarios_correos\$\$, a) IS NOT TRUE THEN RAISE EXCEPTION \$\$correos afectado\$\$; END IF;
$(rechazos " ('portal con clave revocada','portal_candidato',b)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 2'* ]] || fallo "revocación de clave: $salida"
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion (configuracion_revision,revocada_en,motivo_catalogado_ref,acto_ref)
   VALUES ('c1',clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_c1');" \
  "$(rechazos " ('correos con configuración revocada','usuarios_correos',a), ('portal con configuración revocada','portal_candidato',b)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "revocación de configuración: $salida"
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  "INSERT INTO vec_autorizacion_atestada_v3.revocacion_raiz (raiz_clave_id,raiz_version,revocada_en,motivo_catalogado_ref,acto_ref)
   VALUES ('r1',1,clock_timestamp()-interval '1 second','motivo_sintetico','acto_revocacion_r1');" \
  "$(rechazos " ('correos con raíz revocada','usuarios_correos',a), ('portal con raíz revocada','portal_candidato',b)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "revocación de raíz: $salida"
ok 'revocación de clave del portal (correos intacto), de configuración y de raíz rechazadas'

# --- Rotación de puntero: una clave nueva deja obsoleto el material anterior.
salida=$(sql 2>&1 <<'SQL' || printf 'PSQL_FALLO\n'
BEGIN;
SELECT prueba_ad3_112.material('portal_candidato')::text AS b_viejo \gset
SELECT prueba_ad3_112.clave(40,'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_112.material('portal_candidato')::text AS b_nuevo \gset
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE viejo jsonb := %L; nuevo jsonb := %L;
BEGIN
  IF vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', nuevo) IS NOT TRUE
     OR (vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('portal_candidato', nuevo)->>'revision') <> 'c1'
  THEN RAISE EXCEPTION 'material rotado rechazado'; END IF;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', viejo);
    RAISE EXCEPTION 'puntero anterior aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN RAISE NOTICE 'puntero anterior rechazado'; END;
END $t$;$fmt$, :'b_viejo', :'b_nuevo') \gexec
RESET SESSION AUTHORIZATION;
ROLLBACK;
SQL
)
[[ $salida == *'puntero anterior rechazado'* ]] || fallo "rotación de puntero: $salida"
ok 'rotación de puntero del portal: material nuevo aceptado, anterior rechazado'

# --- Caducidad de clave y de configuración.
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  "SELECT prueba_ad3_112.clave(41,'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',clock_timestamp()-interval '2 day',clock_timestamp()-interval '1 day');" \
  "$(rechazos " ('portal con clave caducada','portal_candidato',b)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 2'* ]] || fallo "caducidad de clave: $salida"
salida=$(arnes vec_externo_preflight_v3_desarrollo \
  "SELECT prueba_ad3_112.configuracion(2,clock_timestamp()-interval '2 day',clock_timestamp()-interval '1 day');" \
  "$(rechazos " ('correos con configuración vigente caducada','usuarios_correos',a), ('portal con configuración vigente caducada','portal_candidato',b)")" 2>&1 || true)
[[ $salida == *'rechazos 42501: 4'* ]] || fallo "caducidad de configuración: $salida"
ok 'caducidad de clave y de configuración vigente rechazadas'

# --- Renovación y retroceso por mínimos del checkpoint.
salida=$(sql 2>&1 <<'SQL' || printf 'PSQL_FALLO\n'
BEGIN;
SELECT prueba_ad3_112.material('portal_candidato')::text AS b_c1 \gset
SELECT prueba_ad3_112.configuracion(3,clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
SELECT prueba_ad3_112.material('portal_candidato')::text AS b_c3 \gset
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE previa jsonb := %L; actual jsonb := %L;
BEGIN
  IF (vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('portal_candidato', previa)->>'revision') <> 'c3'
     OR vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', actual) IS NOT TRUE
  THEN RAISE EXCEPTION 'renovación'; END IF;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', previa);
    RAISE EXCEPTION 'sonda aceptó configuración ya sustituida' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN RAISE NOTICE 'configuración sustituida rechazada en sonda'; END;
END $t$;$fmt$, :'b_c1', :'b_c3') \gexec
RESET SESSION AUTHORIZATION;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=4 WHERE control_id;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE b jsonb := %L; n int := 0;
BEGIN
  BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('portal_candidato', b);
    RAISE EXCEPTION 'retroceso aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', b);
    RAISE EXCEPTION 'retroceso aceptado' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  RAISE NOTICE 'retroceso configuración rechazado: %%', n;
END $t$;$fmt$, :'b_c3') \gexec
RESET SESSION AUTHORIZATION;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=0, raiz_version_minima=2 WHERE control_id;
SET SESSION AUTHORIZATION vec_externo_preflight_v3_desarrollo;
SELECT format($fmt$DO $t$
DECLARE b jsonb := %L; n int := 0;
BEGIN
  BEGIN PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1('portal_candidato', b);
    RAISE EXCEPTION 'raíz retrocedida aceptada' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_externa_v1('portal_candidato', b);
    RAISE EXCEPTION 'raíz retrocedida aceptada' USING ERRCODE='P0001';
  EXCEPTION WHEN SQLSTATE '42501' THEN n := n + 1; END;
  RAISE NOTICE 'retroceso raíz rechazado: %%', n;
END $t$;$fmt$, :'b_c3') \gexec
RESET SESSION AUTHORIZATION;
ROLLBACK;
SQL
)
[[ $salida == *'configuración sustituida rechazada en sonda'* && $salida == *'retroceso configuración rechazado: 2'* \
   && $salida == *'retroceso raíz rechazado: 2'* ]] || fallo "renovación/retroceso: $salida"
ok 'renovación c1→c3 leída; retroceso por mínimos de configuración y raíz rechazado'

[[ $(huella_v1) == "$v1_antes" ]] || fallo 'funciones internas alteradas durante el ensayo'

# --- DOWN rechazado con un LOGIN miembro; UP/DOWN/UP y reinicio.
if sql < "$down" >/dev/null 2>&1; then fallo 'DOWN aceptado con LOGIN miembro del rol'; fi
[[ $(existe_v2) == 2 && $(rol_externo) == 1 ]] || fallo 'DOWN rechazado dejó rastro'
sql >/dev/null <<'SQL'
REVOKE CONNECT ON DATABASE postgres FROM vec_externo_preflight_v3_desarrollo, vec_ad3_112_otro_login;
DROP ROLE vec_externo_preflight_v3_desarrollo;
DROP ROLE vec_ad3_112_otro_login;
SQL
sql < "$down" >/dev/null
[[ $(existe_v2) == 0 && $(rol_externo) == 0 && $(huella_v1) == "$v1_antes" ]] || fallo 'DOWN'
if sql < "$down" >/dev/null 2>&1; then fallo 'segundo DOWN aceptado'; fi
sql < "$up" >/dev/null
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'UP tras DOWN no reproduce la postimagen'
sql >/dev/null <<'SQL'
CREATE ROLE vec_externo_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_externo TO vec_externo_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT CONNECT ON DATABASE postgres TO vec_externo_preflight_v3_desarrollo;
SQL
arnes vec_externo_preflight_v3_desarrollo '' "$positivos" >/dev/null
ok 'DOWN rechazado con LOGIN miembro; UP/DOWN/UP con postimagen idéntica; internas intactas; positivos de nuevo'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(huella_v2) == "$v2_primera" && $(huella_v1) == "$v1_antes" ]] || fallo 'estado perdido tras reinicio'
arnes vec_externo_preflight_v3_desarrollo '' "$positivos" >/dev/null
ok 'funciones, ACL y positivos persisten tras reiniciar PostgreSQL'
printf 'md5 internas (antes=después): %s\n' "$v1_antes"
printf 'PG18.4: AD3-112 sobre cadena canónica AD3 1/2/50a/53a/69 con fixture sintético; NO acredita firma COSE, PDP ni cadena AD3 íntegra.\n'
