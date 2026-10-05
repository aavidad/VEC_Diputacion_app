\set ON_ERROR_STOP on
-- No configura una aprobación positiva ni atribuye un actor humano.
-- Vector estructural en clon, después de UP189 nueva, nunca reapply/DOWN.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $estructura$
DECLARE f oid;c jsonb;
BEGIN
 SELECT vec_autorizacion_atestada_v3.codigos_frontera_admin_tecnica_v1() INTO c;
 IF jsonb_array_length(c)<>8 OR EXISTS(SELECT 1 FROM jsonb_array_elements(c) x WHERE x->>'resultado' NOT IN('denegado','error')) THEN RAISE EXCEPTION 'AD189: catalogo_no_cerrado';END IF;
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname IN('acreditar_frontera_admin_tecnica_v1','registrar_frontera_admin_tecnica_v1') LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
  OR NOT has_function_privilege('vec_admin_frontera_tecnica_ejecutor',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND(a.grantee NOT IN(p.proowner,'vec_admin_frontera_tecnica_ejecutor'::regrole) OR a.is_grantable OR a.privilege_type<>'EXECUTE')) THEN RAISE EXCEPTION 'AD189: ACL_no_privada';END IF;
 END LOOP;
 IF(SELECT count(*) FROM vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1)<>0 THEN RAISE EXCEPTION 'AD189: config_favorable_en_instalador';END IF;
 BEGIN PERFORM vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1('{}');RAISE EXCEPTION 'AD189: superusuario_admitido';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $estructura$;
ROLLBACK;
