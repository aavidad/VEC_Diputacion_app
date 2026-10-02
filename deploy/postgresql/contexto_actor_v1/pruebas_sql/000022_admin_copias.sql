\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
\ir ../../identidad_sesiones_v1/pruebas_sql/000011_admin_copias_fixture.sql
DO $puntero_inicial$ BEGIN
 IF (SELECT revision FROM vec_contexto_actor_v1.perfil_admin_copias_actual_v1 WHERE cuenta_ref='cta_fixture_is11_priv_Aaaaaaaaaaaa') IS DISTINCT FROM 1
 THEN RAISE EXCEPTION 'CA22: asignación activa sin puntero inicial'; END IF;
END $puntero_inicial$;
DO $cas$ DECLARE fallo boolean:=false;BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa','admin_copias',pg_catalog.clock_timestamp()+interval '1 minute',repeat('0',64),repeat('d',64),'acto_admin:22000000000000000000000000000001'); EXCEPTION WHEN serialization_failure THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'CA22: CAS falso aceptado'; END IF;
END $cas$;
SET SESSION AUTHORIZATION vec_fixture_is11_login;
SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_copias_v1('oca_fixture_ca22_aaaaaaaaaaaaaaaa','rca_fixture_ca22_aaaaaaaaaaaaaaaa','cta_fixture_is11_priv_Aaaaaaaaaaaa',:'autenticada') \gset contexto_
RESET SESSION AUTHORIZATION;
SELECT pg_catalog.encode(:'contexto_representacion_canonica'::bytea,'hex') AS antes \gset
SELECT * FROM vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa') \gset rev_
SELECT vec_contexto_actor_v1.revocar_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa',:'rev_sha256',repeat('e',64),'acto_admin:22000000000000000000000000000002');
DO $revocada$ DECLARE n integer;fallo boolean:=false;pre record;BEGIN
 IF (SELECT revision FROM vec_contexto_actor_v1.perfil_admin_copias_actual_v1 WHERE cuenta_ref='cta_fixture_is11_priv_Aaaaaaaaaaaa') IS DISTINCT FROM 2
 THEN RAISE EXCEPTION 'CA22: revocación sin avance de puntero'; END IF;
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.consultar_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa'); IF n<>0 THEN RAISE EXCEPTION 'CA22: perfil revocado admitido'; END IF;
 SELECT * INTO STRICT pre FROM vec_contexto_actor_v1.preimagen_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa');
 BEGIN PERFORM vec_contexto_actor_v1.provisionar_perfil_admin_copias_v1('cta_fixture_is11_priv_Aaaaaaaaaaaa','per_fixture_is11_aaaaaaaaaaaaaaaa','prf_fixture_is11_aaaaaaaaaaaaaaaa','vca_fixture_is11_contexto_aaaaaaaa','admin_copias',pg_catalog.clock_timestamp()+interval '1 minute',pre.sha256,repeat('e',64),'acto_admin:22000000000000000000000000000003'); EXCEPTION WHEN object_not_in_prerequisite_state THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'CA22: perfil revivido'; END IF;
END $revocada$;
ROLLBACK;
