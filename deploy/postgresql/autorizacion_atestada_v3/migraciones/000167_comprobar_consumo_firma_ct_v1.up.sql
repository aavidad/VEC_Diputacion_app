\set ON_ERROR_STOP on
-- AD167: atestación privada del consumo recién creado por el núcleo V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000167',0));
DO $pre$
DECLARE n text;
BEGIN
 IF current_user <> 'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'ad167_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['atestacion_decision_v3','consumo_decision_v3','auditoria_consumo_v3'] LOOP
  IF to_regclass('vec_autorizacion_atestada_v3.'||n) IS NULL THEN
   RAISE EXCEPTION 'ad167_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)') IS NOT NULL THEN
  RAISE EXCEPTION 'ad167_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_verificada_ct_v2_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'ad167_preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET TimeZone='UTC'
AS $f$
DECLARE r record; capacidad jsonb; decision jsonb; ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation') <> 'serializable'
  OR current_setting('transaction_read_only') <> 'off'
  OR current_setting('TimeZone') <> 'UTC' OR pg_is_in_recovery() THEN
  RAISE EXCEPTION 'ad167_consumo_no_disponible' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(p_consumo) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(p_consumo)) <> 7
  OR NOT (p_consumo ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
   'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  OR p_consumo->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb
  OR p_consumo->>'decision_ref' IS NULL
  OR p_consumo->>'efecto_ref' IS NULL
  OR (p_consumo->>'huella_efecto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_consumo->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR p_consumo->>'auditoria_ref' IS NULL
  OR p_consumo->>'consumida_en' IS NULL THEN
  RAISE EXCEPTION 'ad167_consumo_no_disponible' USING ERRCODE='42501'; END IF;
 SELECT a.decision_ref,a.efecto_ref,a.huella_efecto_sha256,a.consumo_huella_sha256,
   a.consumida_en,u.auditoria_ref,u.registrada_en,u.huella_sha256 AS auditoria_huella_sha256,
   t.huella_decision_sha256,t.decision_canonica,t.capacidad_canonica,
   a.xmin AS consumo_xmin,u.xmin AS auditoria_xmin
 INTO STRICT r
 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
   ON u.decision_ref=a.decision_ref AND u.efecto_ref=a.efecto_ref
    AND u.huella_efecto_sha256=a.huella_efecto_sha256
 JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
   ON t.decision_ref=a.decision_ref AND t.efecto_ref=a.efecto_ref
    AND t.huella_efecto_sha256=a.huella_efecto_sha256
 WHERE a.decision_ref=p_consumo->>'decision_ref' FOR SHARE OF a,u,t;
 ahora := clock_timestamp();
 -- Las dos filas nuevas deben haber sido insertadas por esta transacción.
 -- transaction_timestamp por sí solo no prueba eso si BEGIN precede al
 -- primer snapshot SERIALIZABLE y otro COMMIT ocurre entre ambos.
 IF r.efecto_ref IS DISTINCT FROM p_consumo->>'efecto_ref'
  OR r.huella_efecto_sha256 IS DISTINCT FROM p_consumo->>'huella_efecto_sha256'
  OR r.consumo_huella_sha256 IS DISTINCT FROM p_consumo->>'consumo_huella_sha256'
  OR r.auditoria_ref IS DISTINCT FROM p_consumo->>'auditoria_ref'
  OR r.consumida_en IS DISTINCT FROM (p_consumo->>'consumida_en')::timestamptz
  OR r.registrada_en IS DISTINCT FROM r.consumida_en
  OR r.consumo_xmin IS DISTINCT FROM pg_current_xact_id()::xid
  OR r.auditoria_xmin IS DISTINCT FROM pg_current_xact_id()::xid
  OR r.consumida_en > ahora
  OR r.huella_decision_sha256 IS DISTINCT FROM encode(sha256(r.decision_canonica),'hex') THEN
  RAISE EXCEPTION 'ad167_consumo_no_disponible' USING ERRCODE='42501'; END IF;
 decision := convert_from(r.decision_canonica,'UTF8')::jsonb;
 capacidad := convert_from(r.capacidad_canonica,'UTF8')::jsonb;
 IF decision->>'decision_ref' IS DISTINCT FROM r.decision_ref
  OR decision->>'concedida' IS DISTINCT FROM 'true'
  OR decision->>'codigo' IS DISTINCT FROM 'concedida'
  OR decision->>'accion' IS DISTINCT FROM capacidad->>'operacion'
  OR decision->>'recurso_ref' IS DISTINCT FROM r.efecto_ref
  OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM r.huella_efecto_sha256
  OR capacidad->>'efecto_ref' IS DISTINCT FROM r.efecto_ref
  OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM r.huella_efecto_sha256
  OR (capacidad->>'audiencia_consumo' IN
   ('vec_contratacion_temporal.firma_vec.v2','vec_contratacion_temporal.firma_externa.v2')) IS NOT TRUE
  OR (decision->>'accion' IN
   ('contratacion_temporal.documento.firma_vec.registrar',
    'contratacion_temporal.documento.firma_externa.registrar')) IS NOT TRUE
  OR (capacidad->>'audiencia_consumo'='vec_contratacion_temporal.firma_vec.v2'
      AND (decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.documento.firma_vec.registrar'
       OR decision->>'tipo_recurso' IS DISTINCT FROM 'firma_vec_documento_contratacion_temporal'))
  OR (capacidad->>'audiencia_consumo'='vec_contratacion_temporal.firma_externa.v2'
      AND (decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
       OR decision->>'tipo_recurso' IS DISTINCT FROM 'firma_externa_documento_contratacion_temporal'))
  OR decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR decision->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR decision#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
  OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
  OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR decision->>'principal_id' IS NULL
  OR decision->>'perfil_activo_ref' IS NULL
  OR decision->>'valida_hasta' IS NULL
  OR (decision->>'valida_hasta')::timestamptz <= ahora THEN
  RAISE EXCEPTION 'ad167_consumo_no_disponible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('decision_ref',r.decision_ref,'efecto_ref',r.efecto_ref,
  'huella_efecto_sha256',r.huella_efecto_sha256,
  'consumo_huella_sha256',r.consumo_huella_sha256,'auditoria_ref',r.auditoria_ref,
  'consumida_en',r.consumida_en,'registrador_principal_ref',decision->>'principal_id',
  'registrador_perfil_ref',decision->>'perfil_activo_ref',
  'auditoria_huella_sha256',r.auditoria_huella_sha256,
  'operacion',decision->>'accion','audiencia',capacidad->>'audiencia_consumo',
  'decision_valida_hasta',decision->>'valida_hasta');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)'::regprocedure;
 a record; owner_id oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL
  aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid=f AND x.grantee<>owner_id LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)
  TO vec_autorizacion_propietario;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(
  coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
  AND x.grantee NOT IN (owner_id,'vec_autorizacion_propietario'::regrole)) THEN
  RAISE EXCEPTION 'ad167_acl_incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
