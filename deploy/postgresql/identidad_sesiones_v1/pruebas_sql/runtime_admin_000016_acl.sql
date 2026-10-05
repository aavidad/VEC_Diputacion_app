\set ON_ERROR_STOP on
-- Ensayo de Dirección tras instalar el candidato exacto en su clon privado.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $acl$ DECLARE f oid;g oid;owner_ca oid; BEGIN
 f:=to_regprocedure('vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(text,numeric,text,text,text,text,text,timestamptz,text)');
 g:=to_regrole('vec_identidad_sesiones_v1_admin_perfiles_runtime');
 owner_ca:=to_regrole('vec_contexto_actor_v1_propietario');
 IF f IS NULL OR g IS NULL OR owner_ca IS NULL THEN RAISE EXCEPTION 'IS16: falta instalación'; END IF;
 IF has_function_privilege(g,f,'EXECUTE') OR NOT has_function_privilege(owner_ca,f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
 THEN RAISE EXCEPTION 'IS16: cotejo accesible fuera del propietario CA'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class t WHERE t.relnamespace='vec_identidad_sesiones_v1'::regnamespace AND t.relkind='r'
 AND has_table_privilege(g,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')) THEN RAISE EXCEPTION 'IS16: runtime con tablas'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class t WHERE t.oid IN('vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1'::regclass,'vec_identidad_sesiones_v1.vinculo_sesion_admin_v1'::regclass) AND NOT(t.relrowsecurity AND t.relforcerowsecurity))
 THEN RAISE EXCEPTION 'IS16: RLS no forzada'; END IF;
 IF vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1('vis_'||repeat('a',32),1,repeat('b',64),'aut_'||repeat('c',32),'ses_'||repeat('d',32),'cta_'||repeat('e',32),'prf_'||repeat('f',32),clock_timestamp(),'certificado') IS NOT NULL
 THEN RAISE EXCEPTION 'IS16: vínculo ausente fabricado'; END IF;
END $acl$;
ROLLBACK;
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL timezone='UTC';
DO $ausencia$ BEGIN
 IF vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,clock_timestamp(),'dnie') IS NOT NULL
 THEN RAISE EXCEPTION 'IS16: vínculo nulo fabricado'; END IF;
END $ausencia$;
ROLLBACK;
