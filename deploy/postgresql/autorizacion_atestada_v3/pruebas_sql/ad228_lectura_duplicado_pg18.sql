\set ON_ERROR_STOP on
-- Ejecutar SOLO en copia aislada PG18 desechable, con -v AD228_DISPOSABLE_CLONE=1.
-- Crea LOGIN sintético y concede EXECUTE directo para probar el emisor; no es bootstrap.
\if :{?AD228_DISPOSABLE_CLONE}
\else
\echo 'AD228: falta AD228_DISPOSABLE_CLONE; prueba omitida sin efectos'
\quit
\endif
CREATE ROLE vec_bolsa_inscripciones_lector LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_inscripciones_empleado_lector LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_inscripciones_rrhh_lector LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_llamamientos_lector_inscripciones TO vec_bolsa_inscripciones_lector WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- Primera versión sólo externa: el LOGIN empleado heredado no tiene grupo lector
-- y debe quedar denegado aunque tenga EXECUTE directo.
GRANT vec_bolsa_llamamientos_lector_inscripciones_rrhh TO vec_bolsa_inscripciones_rrhh_lector WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_inscripciones_lector,
 vec_bolsa_inscripciones_empleado_lector,vec_bolsa_inscripciones_rrhh_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb),
 vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(text,text)
 TO vec_bolsa_inscripciones_lector,vec_bolsa_inscripciones_empleado_lector,vec_bolsa_inscripciones_rrhh_lector;
CREATE SCHEMA ad228_fixture;
CREATE TABLE ad228_fixture.material(contexto bytea NOT NULL,vinculo bytea NOT NULL,orden jsonb NOT NULL);
GRANT USAGE ON SCHEMA ad228_fixture TO vec_bolsa_inscripciones_lector,
 vec_bolsa_inscripciones_empleado_lector,vec_bolsa_inscripciones_rrhh_lector;
GRANT SELECT ON ad228_fixture.material TO vec_bolsa_inscripciones_lector,
 vec_bolsa_inscripciones_empleado_lector,vec_bolsa_inscripciones_rrhh_lector;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $prepare$
DECLARE r record; v bytea; j jsonb; ref text; encontrado boolean:=false;
BEGIN
 FOR r IN SELECT c.registro_contexto_ref,c.huella_sha256,c.manifiesto_procedencia_huella_sha256,
                 c.representacion_canonica,d.decision_canonica
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3 d
          JOIN vec_contexto_actor_v1.registros_contexto c
            ON c.registro_contexto_ref=convert_from(d.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,registro_contexto_ref}'
          WHERE convert_from(d.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,superficie}'='interna_corporativa'
          ORDER BY c.resuelto_en DESC LIMIT 100
 LOOP
  BEGIN
   j:=convert_from(r.decision_canonica,'UTF8')::jsonb->'vinculo_autenticacion_actor';
   v:=convert_to(j::text,'UTF8');
   IF vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(r.registro_contexto_ref,r.huella_sha256,r.manifiesto_procedencia_huella_sha256,r.representacion_canonica)
      AND vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(v) THEN
    ref:='lectura_'||substr(encode(sha256(convert_to('ad228-p1:'||r.registro_contexto_ref,'UTF8')),'hex'),1,32);
    INSERT INTO ad228_fixture.material VALUES(r.representacion_canonica,v,jsonb_build_object(
     'intento_ref',ref,'registro_contexto_ref',r.registro_contexto_ref,'contexto_sha256',r.huella_sha256,
     'procedencia_sha256',r.manifiesto_procedencia_huella_sha256,'autenticacion_ref',j->>'autenticacion_ref',
     'sesion_ref',j->>'sesion_ref','autenticacion_sha256',j->>'autenticacion_huella_sha256',
     'accion','bolsa.inscripcion.rrhh.listar','modulo_id','bolsa',
     'recurso_ref','inscripciones_rrhh_'||repeat('a',64),'finalidad_ref','consulta_inscripcion_rrhh',
     'resultado','obtenida','motivo_ref','inscripcion_lectura_correcta','proceso','vec-server',
     'canal','interna_corporativa','correlacion_ref','correlacion_ad228_p1',
     'lectura_revision_permisos','1','lectura_instantanea_sha256',repeat('b',64)));
    encontrado:=true;EXIT;
   END IF;
  EXCEPTION WHEN others THEN CONTINUE;
  END;
 END LOOP;
 IF NOT encontrado THEN RAISE EXCEPTION 'AD228: sin material histórico apto';END IF;
END $prepare$;
COMMIT;
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp() WHERE control;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $externo$
BEGIN
 IF vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.propias.listar','externa_personal') IS NOT TRUE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.propias.listar','interna_corporativa') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.listar','interna_corporativa') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.convocatorias.listar','interna_corporativa') IS NOT FALSE
 THEN RAISE EXCEPTION 'AD228: carril exterior divergente';END IF;
 BEGIN
  PERFORM * FROM ad228_fixture.material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden);
  RAISE EXCEPTION 'AD228: exterior aceptó lectura RRHH';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
END $externo$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_empleado_lector;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $empleado$
BEGIN
 IF vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.propias.listar','interna_corporativa') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.propias.listar','externa_personal') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.listar','interna_corporativa') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.convocatorias.listar','interna_corporativa') IS NOT FALSE
 THEN RAISE EXCEPTION 'AD228: carril empleado admitido';END IF;
 BEGIN
  PERFORM * FROM ad228_fixture.material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden);
  RAISE EXCEPTION 'AD228: empleado aceptó lectura RRHH';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
END $empleado$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_rrhh_lector;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $aborted$
DECLARE x record;
BEGIN
 IF vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.listar','interna_corporativa') IS NOT TRUE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.convocatorias.listar','interna_corporativa') IS NOT TRUE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.rrhh.convocatorias.listar','externa_personal') IS NOT FALSE
 OR vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(
   'bolsa.inscripcion.propias.listar','interna_corporativa') IS NOT FALSE
 THEN RAISE EXCEPTION 'AD228: carril RRHH divergente';END IF;
 SELECT * INTO STRICT x FROM ad228_fixture.material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden);
 IF x.auditoria_ref !~ '^aud_v3_li_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD228: acuse abortado inválido';END IF;
END $aborted$;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $invalid_revision$
BEGIN
 BEGIN
  PERFORM * FROM ad228_fixture.material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
    m.contexto,m.vinculo,m.orden||jsonb_build_object('lectura_revision_permisos','0'));
  RAISE EXCEPTION 'AD228: revisión cero aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;END;
 BEGIN
  PERFORM * FROM ad228_fixture.material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
    m.contexto,m.vinculo,m.orden||jsonb_build_object('lectura_revision_permisos','18446744073709551616'));
  RAISE EXCEPTION 'AD228: revisión desbordada aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL;END;
END $invalid_revision$;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $first$
DECLARE x record;
BEGIN
 SELECT * INTO STRICT x FROM ad228_fixture.material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden);
 IF x.auditoria_ref !~ '^aud_v3_li_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD228: primer acuse inválido';END IF;
END $first$;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $duplicate$
DECLARE n bigint; mensaje text;
BEGIN
 SELECT count(*) INTO n FROM ad228_fixture.material;
 IF n<>1 THEN RAISE EXCEPTION 'AD228: fixture divergente';END IF;
 BEGIN
  PERFORM * FROM ad228_fixture.material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden);
  RAISE EXCEPTION 'AD228: lectura confirmada reutilizó acuse';
 EXCEPTION WHEN SQLSTATE '23505' THEN
  GET STACKED DIAGNOSTICS mensaje = MESSAGE_TEXT;
  IF mensaje IS DISTINCT FROM 'AD228: lectura ya asentada' THEN
   RAISE EXCEPTION 'AD228: conflicto ajeno a la guarda de replay';
  END IF;
 END;
END $duplicate$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
DO $asiento$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE tipo_registro='lectura_inscripcion' AND lectura_revision_permisos=1
       AND lectura_instantanea_sha256=repeat('b',64))<>1
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE tipo_registro<>'lectura_inscripcion'
       AND (lectura_revision_permisos IS NOT NULL OR lectura_instantanea_sha256 IS NOT NULL))
 THEN RAISE EXCEPTION 'AD228: revisión/huella no durables o familias mezcladas';END IF;
END $asiento$;
SELECT count(*) AS lecturas_confirmadas FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='lectura_inscripcion';
