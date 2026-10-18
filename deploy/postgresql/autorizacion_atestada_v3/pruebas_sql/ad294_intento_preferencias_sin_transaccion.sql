\set ON_ERROR_STOP on
-- Fuente y ACL, en copia desechable tras AD294. No fabrica sesión ni decisión V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $focal$
DECLARE intento text;lectura text;acl text;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl::text INTO STRICT intento,acl
 FROM pg_proc p WHERE p.oid=
  'vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)'::regprocedure;
 SELECT pg_get_functiondef(p.oid) INTO STRICT lectura FROM pg_proc p WHERE p.oid=
  'vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)'::regprocedure;
 IF encode(sha256(convert_to(intento,'UTF8')),'hex') IS DISTINCT FROM
   '819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27'
 OR strpos(intento,'transaccion_origen')<>0 OR strpos(intento,'pg_current_xact_id()')<>0
 OR strpos(intento,'''intento_nominal''')=0
 OR acl IS DISTINCT FROM
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR strpos(lectura,'transaccion_origen')=0 OR strpos(lectura,'pg_current_xact_id()')=0
 OR has_function_privilege('public',
   'vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)',
   'EXECUTE')
 THEN RAISE EXCEPTION 'AD294: intento, lectura positiva o ACL divergentes'; END IF;
END $focal$;
ROLLBACK;
