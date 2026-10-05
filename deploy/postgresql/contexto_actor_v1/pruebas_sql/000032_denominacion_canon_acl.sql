\set ON_ERROR_STOP on
-- Sólo tras instalar el árbol nominal revisado, dentro del clon sintético.
-- Sin fuentes favorables ficticias, personas sembradas ni permisos por petición.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $canon$
DECLARE s jsonb;can text;f oid;owner oid:='vec_contexto_actor_v1_propietario'::regrole;runtime oid:='vec_persona_denominacion_ejecutor'::regrole;
BEGIN
 s:=jsonb_build_object('Esquema','vec.persona.denominacion.aead.v1','PersonaRef','per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','ClaveRef','kms:denominacion:cifrado:1','Version',1,'Nonce',vec_contexto_actor_v1.base64_denominacion_v1(decode(repeat('00',12),'hex')),'Cifrado',vec_contexto_actor_v1.base64_denominacion_v1(decode(repeat('00',17),'hex')),'Indice',jsonb_build_object('AmbitoRef','ambito:denominacion:unidad:1','NormaRef','norma:denominacion:1','NormaSHA256',repeat('a',64),'ClaveRef','kms:denominacion:indice:1','Tokens',jsonb_build_array(vec_contexto_actor_v1.base64_denominacion_v1(decode(repeat('00',32),'hex')))));
 -- Este vector sólo prueba transporte y canon; no se afirma un cifrado real.
 can:=vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(s);
 IF can::jsonb IS DISTINCT FROM s OR strpos(can,'"Esquema":')<>2 THEN RAISE EXCEPTION 'canon incompatible';END IF;
 BEGIN PERFORM vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(s||'{"Version":9007199254740992}'::jsonb);RAISE EXCEPTION 'versión inexacta admitida';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM vec_contexto_actor_v1.canon_sobre_denominacion_persona_v1(s||'{"nombre":"Ana Valdivia"}'::jsonb);RAISE EXCEPTION 'nombre claro admitido';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contexto_actor_v1' AND p.proname IN('canon_sobre_denominacion_persona_v1','validar_material_denominacion_persona_v1','bloquear_persona_denominacion_v1','configurar_sobre_denominacion_persona_v1','cotejar_lectura_denominacion_persona_v1')LOOP
  IF has_function_privilege(runtime,f,'EXECUTE') OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner)))a WHERE p.oid=f AND (a.grantee<>owner OR a.privilege_type<>'EXECUTE'OR a.is_grantable))THEN RAISE EXCEPTION 'helper privado publicado';END IF;
 END LOOP;
 IF has_table_privilege(runtime,'vec_contexto_actor_v1.denominacion_persona_version_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN RAISE EXCEPTION 'runtime con tabla';END IF;
END $canon$;
-- Negativas de ACL sobre los MISMOS objetos: el número de dependencias no
-- cambia con CREATE/TEMP ni WITH GRANT OPTION. No se fabrican fuentes V3.
CREATE ROLE vec_prueba_ca32_acl LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_persona_denominacion_ejecutor TO vec_prueba_ca32_acl WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
SELECT vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
RESET SESSION AUTHORIZATION;
SAVEPOINT esquema_create;
GRANT CREATE ON SCHEMA vec_contexto_actor_v1 TO vec_persona_denominacion_ejecutor;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT esquema_create;
SAVEPOINT esquema_grant_option;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_persona_denominacion_ejecutor WITH GRANT OPTION;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT esquema_grant_option;
SAVEPOINT db_create;
DO $grant$BEGIN EXECUTE format('GRANT CREATE ON DATABASE %I TO vec_persona_denominacion_ejecutor',current_database());END $grant$;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT db_create;
SAVEPOINT db_temp;
DO $grant$BEGIN EXECUTE format('GRANT TEMP ON DATABASE %I TO vec_persona_denominacion_ejecutor',current_database());END $grant$;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT db_temp;
SAVEPOINT db_grant_option;
DO $grant$BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_persona_denominacion_ejecutor WITH GRANT OPTION',current_database());END $grant$;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT db_grant_option;
SAVEPOINT funcion_grant_option;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1()TO vec_persona_denominacion_ejecutor WITH GRANT OPTION;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT funcion_grant_option;
SAVEPOINT login_acl_directa;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_prueba_ca32_acl;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
DO $rechazo$BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
  RAISE EXCEPTION 'CA32: vector_acl_admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $rechazo$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT login_acl_directa;
SET SESSION AUTHORIZATION vec_prueba_ca32_acl;
SELECT vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1();
RESET SESSION AUTHORIZATION;
ROLLBACK;
