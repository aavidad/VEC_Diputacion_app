\set ON_ERROR_STOP on
-- B130. El registrador B119 ya admite la ruta del justificante; su CHECK
-- físico B61/B104 debe admitir la misma ruta para conservar el rechazo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000130',0));

DO $b130$
DECLARE
 tabla regclass:=to_regclass('vec_bolsa_llamamientos.denegacion_frontera_portal_externo');
 firma regprocedure:=to_regprocedure('vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)');
 nombre constant text:='denegacion_frontera_portal_externo_ruta_check';
 ruta_nueva constant text:='/api/vec/bolsa/mi-bolsa/justificantes';
 expresion text; definicion text; tabla_actual record; funcion_actual record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_bolsa_llamamientos_propietario' OR tabla IS NULL OR firma IS NULL
 THEN RAISE EXCEPTION 'B130: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity,c.relkind
 INTO STRICT tabla_actual FROM pg_class c WHERE c.oid=tabla;
 SELECT pg_get_constraintdef(c.oid),pg_get_expr(c.conbin,c.conrelid)
 INTO definicion,expresion FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname=nombre AND c.contype='c'
  AND c.convalidated AND c.conislocal AND c.coninhcount=0;
 SELECT p.proowner,p.proacl,p.prosecdef,p.provolatile,p.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 INTO STRICT funcion_actual FROM pg_proc p WHERE p.oid=firma;
 IF tabla_actual.relowner<>'vec_bolsa_llamamientos_propietario'::regrole
    OR tabla_actual.relacl IS NOT NULL OR tabla_actual.relkind<>'r'
    OR NOT tabla_actual.relrowsecurity OR NOT tabla_actual.relforcerowsecurity
    OR definicion IS NULL OR expresion IS NULL
    OR encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
      '1dcefc307d6780c1e76a09fc9256da48780185dda99330429c1b233d9c6c724e'
    OR strpos(expresion,ruta_nueva)<>0
    OR funcion_actual.proowner<>'vec_bolsa_llamamientos_propietario'::regrole
    OR NOT funcion_actual.prosecdef OR funcion_actual.provolatile<>'v'
    OR funcion_actual.proacl::text IS DISTINCT FROM
      '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
    OR funcion_actual.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC',
      'lock_timeout=1s','statement_timeout=2s']::text[]
    OR funcion_actual.huella IS DISTINCT FROM
      '3379b34b44f2412bff1f001c6fd863937daae15e3f65752e67880af09504fd3c'
 THEN RAISE EXCEPTION 'B130: CHECK, tabla o registrador B119 divergente' USING ERRCODE='55000'; END IF;

 EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo DROP CONSTRAINT %I',nombre);
 EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo ADD CONSTRAINT %I CHECK ((%s) OR ruta=%L)',
  nombre,expresion,ruta_nueva);

 SELECT pg_get_constraintdef(c.oid) INTO definicion FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname=nombre AND c.contype='c'
  AND c.convalidated AND c.conislocal AND c.coninhcount=0;
 SELECT c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity,c.relkind
 INTO STRICT tabla_actual FROM pg_class c WHERE c.oid=tabla;
 SELECT p.proowner,p.proacl,p.prosecdef,p.provolatile,p.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 INTO STRICT funcion_actual FROM pg_proc p WHERE p.oid=firma;
 IF definicion IS NULL OR
    encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
      '72f6260369587777c08531ac627925026f150a4682805ca19c7ca2c48e0d4ffa'
    OR tabla_actual.relowner<>'vec_bolsa_llamamientos_propietario'::regrole
    OR tabla_actual.relacl IS NOT NULL OR tabla_actual.relkind<>'r'
    OR NOT tabla_actual.relrowsecurity OR NOT tabla_actual.relforcerowsecurity
    OR funcion_actual.proowner<>'vec_bolsa_llamamientos_propietario'::regrole
    OR NOT funcion_actual.prosecdef OR funcion_actual.provolatile<>'v'
    OR funcion_actual.proacl::text IS DISTINCT FROM
      '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_registrador_portal_externo=X/vec_bolsa_llamamientos_propietario}'
    OR funcion_actual.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC',
      'lock_timeout=1s','statement_timeout=2s']::text[]
    OR funcion_actual.huella IS DISTINCT FROM
      '3379b34b44f2412bff1f001c6fd863937daae15e3f65752e67880af09504fd3c'
 THEN RAISE EXCEPTION 'B130: postimagen divergente' USING ERRCODE='55000'; END IF;
END $b130$;
COMMIT;
