\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
\ir 000011_admin_copias_fixture.sql
SET SESSION AUTHORIZATION vec_fixture_is11_login;
DO $contrato$ DECLARE n integer;BEGIN
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.acreditar_runtime_admin_copias_v1() WHERE acreditada AND identidad_login='vec_fixture_is11_login';
 IF n<>1 THEN RAISE EXCEPTION 'IS11: runtime no acreditado'; END IF;
 IF pg_catalog.has_table_privilege(session_user,'vec_identidad_sesiones_v1.sesion_admin_copias_v1','SELECT') OR pg_catalog.has_table_privilege(session_user,'vec_contexto_actor_v1.perfil_admin_copias_actual_v1','SELECT,INSERT,UPDATE,DELETE') OR pg_catalog.has_function_privilege(session_user,'vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text)','EXECUTE') OR pg_catalog.has_function_privilege(session_user,'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])','EXECUTE') THEN RAISE EXCEPTION 'IS11: runtime con privilegio ajeno'; END IF;
END $contrato$;
SELECT * FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('a',64),repeat('b',64),:'autenticada',:'revocada') \gset actor_
SELECT (:'actor_cuenta_id'='admin-cuenta-v1:cta_fixture_is11_priv_Aaaaaaaaaaaa') AS alias_inyectivo \gset
\if :alias_inyectivo
\else
\quit 1
\endif
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('a',64),repeat('b',64),:'autenticada',:'revocada',:'ses_autenticacion_ref',:'ses_sesion_ref') AS binding \gset
\if :binding
\else
\quit 1
\endif
SELECT vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('a',64),repeat('b',64),:'autenticada',:'revocada',:'ses_autenticacion_ref',:'ses_sesion_ref') AS binding_replay \gset
\if :binding_replay
\else
\quit 1
\endif
DO $denegaciones$ DECLARE n integer;BEGIN
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','otro.test.invalid','admin_copias',repeat('a',64),repeat('b',64),pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()); IF n<>0 THEN RAISE EXCEPTION 'IS11: host incorrecto admitido'; END IF;
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','admin.test.invalid','otro_perfil',repeat('a',64),repeat('b',64),pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()); IF n<>0 THEN RAISE EXCEPTION 'IS11: audiencia incorrecta admitida'; END IF;
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('e',64),repeat('b',64),pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()); IF n<>0 THEN RAISE EXCEPTION 'IS11: certificado desconocido admitido'; END IF;
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('a',64),repeat('b',64),pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()-interval '4 minutes'); IF n<>0 THEN RAISE EXCEPTION 'IS11: revocación caducada admitida'; END IF;
 SELECT count(*) INTO n FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1('desarrollo','admin.test.invalid','admin_copias',repeat('a',64),repeat('b',64),pg_catalog.clock_timestamp()+interval '1 hour',pg_catalog.clock_timestamp()); IF n<>0 THEN RAISE EXCEPTION 'IS11: autenticación futura admitida'; END IF;
END $denegaciones$;
RESET SESSION AUTHORIZATION;
SELECT vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(:'ses_autenticacion_ref',:'ses_sesion_ref','cta_fixture_is11_priv_Aaaaaaaaaaaa','cta_fixture_is11_ord_aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','pga_fixture_is11_aaaaaaaaaaaaaaaa',repeat('c',64)) AS efecto \gset
\if :efecto
\else
\quit 1
\endif
SELECT NOT vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(:'ses_autenticacion_ref',:'ses_sesion_ref','cta_fixture_is11_priv_Aaaaaaaaaaaa','cta_fixture_is11_ord_aaaaaaaaaaaa','per_fixture_is11_otra_persona_aaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','pga_fixture_is11_aaaaaaaaaaaaaaaa',repeat('c',64)) AS ajeno_denegado \gset
\if :ajeno_denegado
\else
\quit 1
\endif
SELECT vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1('vca_fixture_is11_certificado_aaaaa',1,'acto_admin:11000000000000000000000000000003');
SELECT NOT vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(:'ses_autenticacion_ref',:'ses_sesion_ref','cta_fixture_is11_priv_Aaaaaaaaaaaa','cta_fixture_is11_ord_aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','pga_fixture_is11_aaaaaaaaaaaaaaaa',repeat('c',64)) AS revocado_denegado \gset
\if :revocado_denegado
\else
\quit 1
\endif
DO $cas$ DECLARE fallo boolean:=false;BEGIN
 BEGIN PERFORM vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1('vca_fixture_is11_certificado_aaaaa',1,'acto_admin:11000000000000000000000000000004'); EXCEPTION WHEN serialization_failure THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS11: CAS repetido aceptado'; END IF;
END $cas$;
ROLLBACK;
