\set ON_ERROR_STOP on
-- B78: lectura RRHH de los datos de contacto completos (correo y teléfonos) de
-- una participación. Consume la decisión V3 propia de la consulta (AD197) en la
-- misma transacción SERIALIZABLE que lee el sobre cifrado vigente, su marca de
-- origen y su confirmación; el consumo deja el asiento en la auditoría común con
-- la participación como recurso. Si la decisión no vale, la participación no es
-- de esa bolsa o no hay datos, la transacción se revierte y no queda consumo.
-- La base sigue sin ver el claro: devuelve el sobre y Go lo descifra con el KMS.
-- No cambia la lectura sin consumo de B16, que siguen usando el registro de
-- datos y la emisión de llamamientos para trabajar con el sobre.
-- Una sola vez; sin DOWN. Requiere AD197.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000078',0));

DO $precondicion$
DECLARE actual jsonb; esperado jsonb:=pg_catalog.jsonb_build_object(
 'rol',true,'ad197',true,'tablas',true,'libre',true,'ejecutor',true);
BEGIN
 actual:=pg_catalog.jsonb_build_object(
  'rol',current_user='vec_bolsa_llamamientos_propietario',
  'ad197',pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_datos_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
  'tablas',pg_catalog.to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.origen_datos_contacto_participacion') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.confirmacion_contacto_participacion') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.constitucion') IS NOT NULL,
  'libre',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_datos_contacto_participacion_rrhh_v1(text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL,
  'ejecutor',EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls));
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'PARO clave=B78.preimagen, actual=%, esperado=%',actual,esperado USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_datos_contacto_participacion_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_actor_ref text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(version bigint,clave_ref text,nonce bytea,cifrado bytea,registrada_en timestamptz,recibo_ref text,
 origen text,vigente_hasta timestamptz,ultimo_dia date,regla_ref text,regla_huella_sha256 text,confirmada_en timestamptz,
 decision_ref text,auditoria_ref text,consumida_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; decision jsonb; dato record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa_ref IS NULL OR p_bolsa_ref<>pg_catalog.btrim(p_bolsa_ref) OR pg_catalog.octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512
    OR p_participacion_ref IS NULL OR p_participacion_ref<>pg_catalog.btrim(p_participacion_ref)
    OR pg_catalog.octet_length(p_participacion_ref) NOT BETWEEN 1 AND 512
    OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256 THEN
  RAISE EXCEPTION 'B78: consulta de datos de contacto no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B78: consulta de datos de contacto no autorizada' USING ERRCODE='42501'; END;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_datos_contacto_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_participacion_ref
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref
    OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref THEN
  RAISE EXCEPTION 'B78: consulta de datos de contacto no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
   WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B78: consulta de datos de contacto no autorizada' USING ERRCODE='42501'; END IF;
 SELECT d.version,d.clave_ref,d.nonce,d.cifrado,d.registrada_en,d.recibo_ref INTO dato
  FROM vec_bolsa_llamamientos.datos_contacto_participacion d
  WHERE d.participacion_ref=p_participacion_ref ORDER BY d.version DESC LIMIT 1;
 -- Sin datos no hay nada que revelar: se revierte también el consumo.
 IF NOT FOUND THEN
  RAISE EXCEPTION 'B78: participación sin datos de contacto' USING ERRCODE='P0002'; END IF;
 RETURN QUERY SELECT dato.version,dato.clave_ref,dato.nonce,dato.cifrado,dato.registrada_en,dato.recibo_ref,
   o.origen,o.vigente_hasta,o.ultimo_dia,o.regla_ref,o.regla_huella_sha256,cf.confirmada_en,
   consumo.decision_ref,consumo.auditoria_ref,consumo.consumida_en
  FROM (SELECT 1) AS unica
  LEFT JOIN vec_bolsa_llamamientos.origen_datos_contacto_participacion o
    ON o.participacion_ref=p_participacion_ref AND o.version=dato.version
  LEFT JOIN vec_bolsa_llamamientos.confirmacion_contacto_participacion cf
    ON cf.participacion_ref=p_participacion_ref AND cf.version=dato.version;
END $f$;

DO $acl$
DECLARE f pg_catalog.regprocedure:='vec_bolsa_llamamientos.consultar_datos_contacto_participacion_rrhh_v1(text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc q
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
  WHERE q.oid=f AND a.grantee<>q.proowner LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f::text);
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    OR NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f)
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc q
         CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
         WHERE q.oid=f AND a.grantee NOT IN (q.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))<>0 THEN
  RAISE EXCEPTION 'PARO clave=B78.ACL, esperado=propietario_y_ejecutor' USING ERRCODE='55000';
 END IF;
END $acl$;
COMMIT;
