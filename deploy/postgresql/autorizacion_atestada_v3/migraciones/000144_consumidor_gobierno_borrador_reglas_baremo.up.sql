\set ON_ERROR_STOP on
-- AD3-144: fachada nominal de consumo V3 para el gobierno de borradores de baremo.
-- Debe instalarse después de AD3-143 y del perfil de núcleo GobiernoG.
-- Sólo vec_bolsa_reglas_baremo_propietario puede invocarla; la autorización
-- real sigue en V3 y la operación de Bolsa la consume en su transacción.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000144',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
DECLARE nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        definicion text; audiencia text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_orden_copias_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR nucleo IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_reglas_baremo_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_reglas_baremo_ejecutor_gobierno' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls)
 OR has_schema_privilege('vec_bolsa_reglas_baremo_propietario','vec_autorizacion_atestada_v3','USAGE')
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole)
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=nucleo AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
    OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD3-144: orden causal o roles incompatibles' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(nucleo) INTO STRICT definicion;
 IF strpos(definicion,'p_perfil_mutacion IS NOT DISTINCT FROM ''gobierno_borrador_reglas_baremo''')=0
 OR strpos(definicion,'vec_bolsa_reglas_baremo.gobierno_borrador.v3')=0
 OR strpos(definicion,'vec_bolsa_reglas_baremo_ejecutor_gobierno')=0
 THEN RAISE EXCEPTION 'AD3-144: núcleo GobiernoG no preparado' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT audiencia FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(audiencia,'vec_bolsa_reglas_baremo.gobierno_borrador.v3')=0
 THEN RAISE EXCEPTION 'AD3-144: audiencia GobiernoG no preparada' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,
 consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s'
AS $f$
DECLARE c jsonb; d jsonb; x record; escritura boolean;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD3-144: material V3 fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD3-144: material V3 inválido' USING ERRCODE='22023';
 END;
 escritura:=c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reglas_baremo.borrador.crear';
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_reglas_baremo.gobierno_borrador.v3'
 OR coalesce(c->>'operacion','') NOT IN ('bolsa.reglas_baremo.borrador.crear','bolsa.reglas_baremo.version.consultar','bolsa.reglas_baremo.recibo.consultar')
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM CASE WHEN escritura THEN 'intencion_gobierno_reglas_baremo' ELSE 'version_reglas_baremo_gobernada' END
 OR d->>'finalidad' IS DISTINCT FROM CASE WHEN escritura THEN 'gobierno_reglas_baremo' ELSE 'consulta_gobierno_reglas_baremo' END
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'false'
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR (escritura AND (coalesce(d->>'recurso_ref','') !~ '^intencion-reglas-baremo:[0-9a-f]{64}$'
   OR d->'campos_permitidos' IS DISTINCT FROM '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb))
 OR (NOT escritura AND coalesce(d->>'recurso_ref','') !~ '^reglas-baremo:[0-9a-f]{64}$')
 OR (c->>'operacion'='bolsa.reglas_baremo.version.consultar'
   AND d->'campos_permitidos' IS DISTINCT FROM '["estado_reglas_baremo"]'::jsonb)
 OR (c->>'operacion'='bolsa.reglas_baremo.recibo.consultar'
   AND d->'campos_permitidos' IS DISTINCT FROM '["recibo"]'::jsonb)
 THEN RAISE EXCEPTION 'AD3-144: capacidad GobiernoG denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'gobierno_borrador_reglas_baremo',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD3-144: requiere consumo fresco' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_reglas_baremo_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_reglas_baremo_propietario;
DO $acl$
DECLARE f oid:='vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner
    OR a.grantee NOT IN (p.proowner,'vec_bolsa_reglas_baremo_propietario'::regrole)))
 OR NOT has_function_privilege('vec_bolsa_reglas_baremo_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD3-144: ACL de consumidor incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
