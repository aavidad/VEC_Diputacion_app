\set ON_ERROR_STOP on
-- Vector estructural tras UP190. No emite claves, decisiones ni perfiles.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE f oid;aud text;core text;recurso jsonb;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
 OR has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE') IS NOT TRUE
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND(a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD190: ACL del consumidor ampliada' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT aud FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(aud,'vec_autorizacion.administracion_perfiles.lote_ordinario.v1')=0
 THEN RAISE EXCEPTION 'AD190: audiencia ausente' USING ERRCODE='55000'; END IF;
 SELECT p.prosrc INTO STRICT core FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF strpos(core,'''admin_perfiles_lote_ordinario''')=0
 THEN RAISE EXCEPTION 'AD190: perfil del núcleo ausente' USING ERRCODE='55000'; END IF;
 recurso:=vec_autorizacion_atestada_v3.recurso_lote_ordinario_admin_v1(
  $sol${"Esquema":"administracion_perfiles_lote:v3","OperacionRef":"acto_admin:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","OrganizacionRef":"org_prueba","Cambios":[{"Objetivo":{"UnidadRef":"unidad:prueba","PersonaRef":"per_aaaaaaaaaaaaaaaaaaaaaa"}}]}$sol$);
 IF recurso->>'solicitud_sha256' IS DISTINCT FROM '2309a242dda19bcb4396565a8aae684f35bce0b10abca88dd45ef560f853e492'
 OR recurso->>'contexto_sha256' IS DISTINCT FROM '2fb8a0cc32cce33eeaff3153b8b7be63285f9b0116ea1fff52cf08479fe4cf41'
 THEN RAISE EXCEPTION 'AD190: vector canon Go divergente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_lote_ordinario_admin_v3_atestada('{}',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD190: superusuario admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $prueba$;
ROLLBACK;
