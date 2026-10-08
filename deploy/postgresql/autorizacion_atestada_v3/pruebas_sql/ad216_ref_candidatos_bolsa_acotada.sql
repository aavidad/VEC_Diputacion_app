\set ON_ERROR_STOP on
-- Sólo en clon PG18 POST214→AD216. La función de prueba y sus GRANT terminan
-- en ROLLBACK; no publica capacidad, decisión ni actor sintético como real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='5s';
SET LOCAL idle_in_transaction_session_timeout='5s';
DO $contrato$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fuente text; patron text; i integer; ref text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD216 prueba: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT p.prosrc INTO STRICT fuente FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
     '96d92f060ddfd4970bf47167c5b23ee140bc67bd521166363c942f6f83203bc3'
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM
     '03621298761d25e5e5decb2a8bee13cb2193f9fc07035a6dd7145ea5e5eec1cd'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex')
        FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
          AND c.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM
       '8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e'
    OR to_regprocedure('vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada()') IS NOT NULL
 THEN RAISE EXCEPTION 'AD216 prueba: postimagen o fixture incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
     AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
     AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
     AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
       aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 THEN RAISE EXCEPTION 'AD216 prueba: ACL/entorno del núcleo abiertos' USING ERRCODE='55000'; END IF;
 i:=strpos(fuente,'d->>''recurso_ref'' ~ ''^bolsa:');
 IF i=0 OR strpos(fuente,'octet_length(d->>''recurso_ref'') BETWEEN 79 AND 200')=0
 THEN RAISE EXCEPTION 'AD216 prueba: límite o expresión ausentes' USING ERRCODE='55000'; END IF;
 patron:=split_part(substr(fuente,i+length('d->>''recurso_ref'' ~ ''')),'''',1);
 IF patron IS DISTINCT FROM '^bolsa:[A-Za-z0-9_-]+(:[A-Za-z0-9_-]+)*:filtro:[a-f0-9]{64}$'
 THEN RAISE EXCEPTION 'AD216 prueba: clase de bolsa_ref divergente' USING ERRCODE='55000'; END IF;
 FOREACH ref IN ARRAY ARRAY[
  'bolsa:auxiliar:2026:filtro:'||repeat('a',64),
  'bolsa:of:1:filtro:'||repeat('b',64),
  'bolsa:'||repeat('c',64)||':filtro:'||repeat('d',64),
  'bolsa:'||repeat('e',122)||':filtro:'||repeat('f',64)] LOOP
  IF NOT (octet_length(ref) BETWEEN 79 AND 200 AND ref ~ patron)
  THEN RAISE EXCEPTION 'AD216 prueba: referencia válida denegada' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH ref IN ARRAY ARRAY[
  'bolsa:a::b:filtro:'||repeat('a',64),
  'bolsa::a:filtro:'||repeat('a',64),
  'bolsa:'||repeat('e',123)||':filtro:'||repeat('f',64),
  'bolsa:a:filtro:'||repeat('A',64),
  'bolsa:a:filtro:'||repeat('a',63),
  'bolsa:a:filtro:'||repeat('a',64)||':extra',
  'bolsa:a/b:filtro:'||repeat('a',64)] LOOP
  IF octet_length(ref) BETWEEN 79 AND 200 AND ref ~ patron
  THEN RAISE EXCEPTION 'AD216 prueba: referencia fuera de contrato aceptada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $contrato$;

-- La sesión conserva session_user del LOGIN técnico real al ejecutar el
-- wrapper privado. SET ROLE desde postgres no probaría esa guarda del núcleo.
CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada() RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE codigo text;
BEGIN
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'consulta_rrhh_bolsa',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  RETURN codigo;
 END;
 RETURN 'aceptado';
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada()
 OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada()
 TO vec_bolsa_llamamientos_ejecutor;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $login$
BEGIN
 IF session_user<>'vec_bolsa_llamamientos_desarrollo' OR current_user<>session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
       AND r.rolcanlogin AND r.rolinherit
       AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
       AND r.rolconfig IS NULL)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
       AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
       AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR vec_autorizacion_atestada_v3.prueba_ad216_material_invalido_privada()<>'22023'
 THEN RAISE EXCEPTION 'AD216 prueba: LOGIN o rechazo de material inválido divergente' USING ERRCODE='55000'; END IF;
END $login$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
