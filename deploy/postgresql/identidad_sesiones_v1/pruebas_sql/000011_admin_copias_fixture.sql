-- Solo para PostgreSQL 18 efímero sintético. La transacción llamadora revierte.
\set ON_ERROR_STOP on
SELECT pg_catalog.clock_timestamp() AS fixture_now \gset
INSERT INTO vec_identidad_sesiones_v1.cuenta VALUES
('cta_fixture_is11_ord_aaaaaaaaaaaa',false,NULL,:'fixture_now','opr_fixture_is11_ord_aaaaaaaaaaaa'),
('cta_fixture_is11_priv_Aaaaaaaaaaaa',true,'cta_fixture_is11_ord_aaaaaaaaaaaa',:'fixture_now','opr_fixture_is11_priv_aaaaaaaaaaaa');
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta VALUES
('cta_fixture_is11_ord_aaaaaaaaaaaa',1,'activa',:'fixture_now','opr_fixture_is11_est_ord_aaaaaaaa'),
('cta_fixture_is11_priv_Aaaaaaaaaaaa',1,'activa',:'fixture_now','opr_fixture_is11_est_priv_aaaaaaaa');
INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual VALUES
('cta_fixture_is11_ord_aaaaaaaaaaaa',1,:'fixture_now','opr_fixture_is11_est_ord_aaaaaaaa'),
('cta_fixture_is11_priv_Aaaaaaaaaaaa',1,:'fixture_now','opr_fixture_is11_est_priv_aaaaaaaa');
SELECT vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1('opr_fixture_is11_aliasord_aaaaaaa','cta_fixture_is11_ord_aaaaaaaaaaaa','vec.identidad.hmac-sha256.v1','idh_fixture_is11_aaaaaaaaaaaaaaaa','clave-fixture-is11',1,decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex')) IS NOT NULL AS ord_ok \gset
SELECT vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1('opr_fixture_is11_aliaspriv_aaaaaa','cta_fixture_is11_priv_Aaaaaaaaaaaa','vec.identidad.hmac-sha256.v1','idh_fixture_is11_aaaaaaaaaaaaaaaa','clave-fixture-is11',1,decode(repeat('66',32),'hex'),decode(repeat('22',32),'hex')) IS NOT NULL AS priv_ok \gset
INSERT INTO vec_contexto_actor_v1.procedencias VALUES('prc_fixture_is11_aaaaaaaaaaaaaaaa',1,repeat('1',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES('cta_fixture_is11_priv_Aaaaaaaaaaaa',1,'prc_fixture_is11_aaaaaaaaaaaaaaaa',1,repeat('1',64),'autoridad_maestra_acreditada','activo',:'fixture_now',:'fixture_now'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_fixture_is11_priv_Aaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES('per_fixture_is11_aaaaaaaaaaaaaaaa',1,'prc_fixture_is11_aaaaaaaaaaaaaaaa',1,repeat('1',64),'autoridad_maestra_acreditada','activo',:'fixture_now',:'fixture_now'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_fixture_is11_aaaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES('prf_fixture_is11_aaaaaaaaaaaaaaaa',1,'per_fixture_is11_aaaaaaaaaaaaaaaa','prc_fixture_is11_aaaaaaaaaaaaaaaa',1,repeat('1',64),'autoridad_maestra_acreditada','activo',:'fixture_now',:'fixture_now'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_fixture_is11_aaaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES('vca_fixture_is11_contexto_aaaaaaaa',1,'cta_fixture_is11_priv_Aaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prc_fixture_is11_aaaaaaaaaaaaaaaa',1,repeat('1',64),'autoridad_maestra_acreditada','activo',:'fixture_now',:'fixture_now'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_fixture_is11_contexto_aaaaaaaa',1);
INSERT INTO vec_identidad_sesiones_v1.politica_certificado_admin_v1(politica_ref,entorno,host_admin,ca_sha256,huella_aprobacion_sha256,maxima_edad_revocacion,vigente_hasta) VALUES('pga_fixture_is11_aaaaaaaaaaaaaaaa','desarrollo','admin.test.invalid',repeat('b',64),repeat('c',64),interval '3 minutes',pg_catalog.clock_timestamp()+interval '1 hour');
SELECT vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1('vca_fixture_is11_certificado_aaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','cta_fixture_is11_priv_Aaaaaaaaaaaa',repeat('a',64),repeat('b',64),'pga_fixture_is11_aaaaaaaaaaaaaaaa',pg_catalog.clock_timestamp()+interval '30 minutes','acto_admin:11000000000000000000000000000001');
SELECT * FROM vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa') \gset pre_
SELECT vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa','admin_copias',pg_catalog.clock_timestamp()+interval '20 minutes',:'pre_sha256',repeat('d',64),'acto_admin:11000000000000000000000000000002');
SELECT pg_catalog.clock_timestamp() AS autenticada,pg_catalog.clock_timestamp() AS revocada \gset
SELECT * FROM vec_identidad_sesiones_v1.registrar_sesion_v1('opr_fixture_is11_sesion_aaaaaaaaa','vec.identidad.hmac-sha256.v1','idh_fixture_is11_aaaaaaaaaaaaaaaa','clave-fixture-is11',1,decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),decode(repeat('22',32),'hex'),decode(repeat('66',32),'hex'),decode(repeat('11',32),'hex'),true,'administracion_privilegiada','certificado','alto',repeat('f',64),:'autenticada',pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()+interval '1 minute','pga_fixture_is11_aaaaaaaaaaaaaaaa',repeat('c',64)) \gset ses_
CREATE ROLE vec_fixture_is11_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_copias TO vec_fixture_is11_login WITH INHERIT TRUE,SET FALSE,ADMIN FALSE;
