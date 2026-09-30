\set ON_ERROR_STOP on
-- Ejecutar después de AD3-136 y de las cinco altas nominales B2.
-- Solo verifica catálogo, configuración y ACL; no crea consumos.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
DO $prueba$
DECLARE
 firma text;
 beneficiario text;
 f oid;
 p record;
 nucleo oid;
 fuente text;
 config text[];
BEGIN
 FOR firma,beneficiario IN
  SELECT * FROM (VALUES
   ('vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_contratacion_temporal_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_bolsa_llamamientos_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_bolsa_llamamientos_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_personal_propietario'),
   ('vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'vec_contratacion_temporal_propietario')
  ) AS esperadas(firma,beneficiario)
 LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL THEN RAISE EXCEPTION 'B2: fachada ausente %',firma; END IF;
  SELECT proowner,prosecdef,proconfig,proacl INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR p.prosecdef IS NOT TRUE
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
     OR NOT has_function_privilege(beneficiario,f,'EXECUTE')
     OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE a.grantee NOT IN (p.proowner,beneficiario::regrole::oid)
          OR a.privilege_type<>'EXECUTE')
  THEN RAISE EXCEPTION 'B2: configuración o ACL divergente %',firma; END IF;
 END LOOP;

 nucleo:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF nucleo IS NULL THEN RAISE EXCEPTION 'B2: núcleo ausente'; END IF;
 SELECT prosrc,proconfig INTO STRICT fuente,config FROM pg_proc WHERE oid=nucleo;
 IF config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR strpos(fuente,'documentos.firmado.custodiar')=0
    OR strpos(fuente,'vinculo_categoria_rpt_ct')=0
    OR strpos(fuente,'consulta_persona_aceptacion_ct_bolsa')=0
    OR strpos(fuente,'consulta_anclaje_aceptacion_ct_bolsa')=0
    OR strpos(fuente,'plan_incorporacion_personal_ct')=0
    OR strpos(fuente,'incorporacion_personal_ct')=0
 THEN RAISE EXCEPTION 'B2: núcleo incompleto'; END IF;
END $prueba$;
ROLLBACK;
