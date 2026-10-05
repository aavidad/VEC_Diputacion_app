\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $vector$
DECLARE persona text:='per_'||replace(gen_random_uuid()::text,'-','');perfil text:='prf_'||replace(gen_random_uuid()::text,'-','');x jsonb;
BEGIN
 x:=vec_contexto_actor_v1.metadatos_persona_administrable_v1(persona);
 IF x IS NOT NULL THEN RAISE EXCEPTION 'CA34: ausencia_no_comprobada';END IF;
 x:=vec_contexto_actor_v1.metadatos_perfil_administrable_v1(persona,perfil);
 IF x IS NOT NULL THEN RAISE EXCEPTION 'CA34: perfil_ajeno';END IF;
 BEGIN PERFORM vec_contexto_actor_v1.metadatos_persona_administrable_v1(NULL);RAISE EXCEPTION 'CA34: null_admitido';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM vec_contexto_actor_v1.metadatos_perfil_administrable_v1(persona,'cuenta_inventada');RAISE EXCEPTION 'CA34: referencia_abierta';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
END $vector$;
RESET ROLE;
DO $acl$
DECLARE f oid;
BEGIN
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contexto_actor_v1' AND p.proname IN('metadatos_persona_administrable_v1','metadatos_perfil_administrable_v1') LOOP
  IF has_function_privilege('vec_admin_perfiles_ejecutor',f,'EXECUTE') THEN RAISE EXCEPTION 'CA34: runtime_con_metadata';END IF;
 END LOOP;
END $acl$;
ROLLBACK;
