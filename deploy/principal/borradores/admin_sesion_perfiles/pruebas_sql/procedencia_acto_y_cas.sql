-- Solo clon efímero, fuentes sintéticas. La transacción revierte todo.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.clock_timestamp() AS ahora \gset
INSERT INTO vec_contexto_actor_v1.procedencias VALUES('prc_fixture_ca23_fuente_aaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES('cta_fixture_ca23_aaaaaaaaaaaaaaaa',1,'prc_fixture_ca23_fuente_aaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo',:'ahora',:'ahora'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_fixture_ca23_aaaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES('per_fixture_ca23_aaaaaaaaaaaaaaaa',1,'prc_fixture_ca23_fuente_aaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo',:'ahora',:'ahora'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_fixture_ca23_aaaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES('prf_fixture_ca23_aaaaaaaaaaaaaaaa',1,'per_fixture_ca23_aaaaaaaaaaaaaaaa','prc_fixture_ca23_fuente_aaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo',:'ahora',:'ahora'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_fixture_ca23_aaaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES('vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,'cta_fixture_ca23_aaaaaaaaaaaaaaaa','prf_fixture_ca23_aaaaaaaaaaaaaaaa','per_fixture_ca23_aaaaaaaaaaaaaaaa','prc_fixture_ca23_fuente_aaaaaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada','activo',:'ahora',:'ahora'::timestamptz+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_fixture_ca23_aaaaaaaaaaaaaaaa',1);
SET LOCAL ROLE vec_autorizacion_propietario;
DO $rechazos$ DECLARE denegado boolean; BEGIN
 denegado:=false;
 BEGIN PERFORM vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('b',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',0,0,'activo'); EXCEPTION WHEN invalid_parameter_value THEN denegado:=true; END;
 IF NOT denegado THEN RAISE EXCEPTION 'CA23: alta/versión cero admitida como procedencia de revocación'; END IF;
 denegado:=false;
 BEGIN PERFORM vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('b',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',2,1,'revocado'); EXCEPTION WHEN serialization_failure THEN denegado:=true; END;
 IF NOT denegado THEN RAISE EXCEPTION 'CA23: versión previa falsa admitida'; END IF;
END $rechazos$;
SELECT vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('b',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,1,'revocado') AS registrada \gset
SELECT vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('b',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,1,'revocado')=:'registrada'::jsonb AS replay_previo \gset
\if :replay_previo
\else
\quit 1
\endif
DO $colision$ DECLARE denegado boolean:=false; BEGIN
 BEGIN PERFORM vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('c',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,1,'revocado'); EXCEPTION WHEN unique_violation THEN denegado:=true; END;
 IF NOT denegado THEN RAISE EXCEPTION 'CA23: acto reutilizado con material distinto'; END IF;
END $colision$;
SELECT vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1('cta_fixture_ca23_aaaaaaaaaaaaaaaa','per_fixture_ca23_aaaaaaaaaaaaaaaa','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,1,1,1,(:'registrada'::jsonb)->>'procedencia_ref',((:'registrada'::jsonb)->>'procedencia_version')::numeric,(:'registrada'::jsonb)->>'procedencia_huella_sha256');
SELECT vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1('acto_admin:23000000000000000000000000000001','cierre_admin:23000000000000000000000000000001',repeat('b',64),'decision:fixture-ca23','auditoria:fixture-ca23','prf_fixture_ca23_aaaaaaaaaaaaaaaa','vca_fixture_ca23_aaaaaaaaaaaaaaaa',1,1,'revocado')=:'registrada'::jsonb AS replay_efecto \gset
\if :replay_efecto
\else
\quit 1
\endif
RESET ROLE;
DO $historia$ DECLARE denegado boolean:=false; n integer; BEGIN
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.procedencia_acto_admin_v1 WHERE acto_ref='acto_admin:23000000000000000000000000000001';
 IF n<>1 THEN RAISE EXCEPTION 'CA23: ledger duplicado'; END IF;
 IF (SELECT version FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref='cta_fixture_ca23_aaaaaaaaaaaaaaaa')<>1 OR (SELECT version FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref='per_fixture_ca23_aaaaaaaaaaaaaaaa')<>1 THEN RAISE EXCEPTION 'CA23: acto modificó la fuente maestra'; END IF;
 BEGIN UPDATE vec_contexto_actor_v1.procedencia_acto_admin_v1 SET material_sha256=repeat('f',64) WHERE acto_ref='acto_admin:23000000000000000000000000000001'; EXCEPTION WHEN object_not_in_prerequisite_state THEN denegado:=true; END;
 IF NOT denegado THEN RAISE EXCEPTION 'CA23: ledger mutable'; END IF;
 IF pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_admin_perfiles','vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1(text,text,text,text,text,text,text,numeric,numeric,text)','EXECUTE') THEN RAISE EXCEPTION 'CA23: runtime puede registrar actos'; END IF;
END $historia$;
ROLLBACK;
