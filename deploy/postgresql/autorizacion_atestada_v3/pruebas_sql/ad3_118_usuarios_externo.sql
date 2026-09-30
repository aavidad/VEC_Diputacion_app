\set ON_ERROR_STOP on
-- Sonda estructural sin escritura: instalar AUT-17 y AD3-118 antes.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f regprocedure; cuerpo text; despacho text;
BEGIN
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT p.prosrc INTO STRICT cuerpo FROM pg_catalog.pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
 IF cuerpo !~ 'session_user <> ''vec_externo_usuarios_desarrollo'''
    OR cuerpo !~ 'vec_usuarios_ejecutor_externo'
    OR cuerpo !~ 'preferencias_consulta_usuarios'
    OR cuerpo !~ 'imagen_actualizar_usuarios'
    OR cuerpo !~ 'correos_retirar_usuarios'
    OR cuerpo !~ 'registrar_y_revalidar_decision_usuarios_externo_v3'
    OR cuerpo !~ 'revalidar_decision_usuarios_externo_v3_viva'
    OR cuerpo ~ 'registrar_y_revalidar_decision_contexto_actor_externa_v3'
    OR cuerpo ~ 'revalidar_decision_contexto_actor_externa_v3_viva'
    OR cuerpo ~ 'vec_autorizacion_atestada_v3\.(atestacion_decision_v3|consumo_decision_v3|auditoria_consumo_v3|control_cadena_auditoria)([^_a-z]|$)'
 THEN RAISE EXCEPTION 'AD3-118: carril Usuarios no aislado'; END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
     LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=f AND a.privilege_type='EXECUTE'
       AND (a.grantee=0 OR a.grantee='vec_usuarios_ejecutor_externo'::regrole))
 THEN RAISE EXCEPTION 'AD3-118: función interna expuesta'; END IF;
 SELECT p.prosrc INTO STRICT despacho FROM pg_catalog.pg_proc p
  WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF despacho !~ 'session_user = ''vec_externo_usuarios_desarrollo'''
    OR despacho !~ 'consumir_decision_mutacion_v3_usuarios_externa_interna'
    OR despacho !~ 'session_user = ''vec_externo_bolsa_desarrollo'''
    OR despacho !~ 'consumir_decision_mutacion_v3_externa_interna'
 THEN RAISE EXCEPTION 'AD3-118: despacho externo incompatible'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
  WHERE conrelid='vec_autorizacion_atestada_v3.atestacion_decision_v3_externa'::regclass
    AND confrelid='vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass
    AND contype='f')
 THEN RAISE EXCEPTION 'AD3-118: atestación no enlaza decisión externa'; END IF;
END $prueba$;
ROLLBACK;
