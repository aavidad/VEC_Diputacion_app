-- CA6 solo puede retirarse sin uso ni dependencias de CT109.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $preimagen$
DECLARE cantidad integer;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA6: retirada con dependencias' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO cantidad FROM pg_namespace n,
  LATERAL aclexplode(n.nspacl) a
 WHERE n.nspname='vec_contexto_actor_v1'
   AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                     'vec_contratacion_temporal_propietario'::regrole)
   AND a.grantor='vec_contexto_actor_v1_propietario'::regrole
   AND a.privilege_type='USAGE' AND NOT a.is_grantable;
 IF cantidad<>2 OR EXISTS (SELECT 1 FROM pg_namespace n,
   LATERAL aclexplode(n.nspacl) a
   WHERE n.nspname='vec_contexto_actor_v1'
     AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                       'vec_contratacion_temporal_propietario'::regrole)
     AND (a.grantor<>'vec_contexto_actor_v1_propietario'::regrole
       OR a.privilege_type<>'USAGE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'CA6: ACL previa a retirada incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(jsonb,text,text,text,text,text);
DROP FUNCTION vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(text,numeric,text,numeric,text,numeric,text,numeric);
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM
 vec_contexto_actor_corporativo_rrhh_selector,
 vec_contratacion_temporal_propietario;
DO $postimagen$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_namespace n,
   LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
   WHERE n.nspname='vec_contexto_actor_v1'
     AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                       'vec_contratacion_temporal_propietario'::regrole))
   OR has_schema_privilege('vec_contexto_actor_corporativo_rrhh_selector',
         'vec_contexto_actor_v1','USAGE')
   OR has_schema_privilege('vec_contratacion_temporal_propietario',
         'vec_contexto_actor_v1','USAGE')
 THEN RAISE EXCEPTION 'CA6: ACL residual tras retirada' USING ERRCODE='55000'; END IF;
END $postimagen$;
COMMIT;
