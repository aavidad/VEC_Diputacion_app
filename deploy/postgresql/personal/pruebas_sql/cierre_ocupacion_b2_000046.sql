\set ON_ERROR_STOP on
-- Personal46: prueba focal sobre clon PG18 desechable tras instalar P46.
-- Fixture propio en ROLLBACK. Se llama a la fachada B2 completa, con un doble
-- V3 transaccional explícito: NO acredita firma, V3 real ni eficacia jurídica.
-- Requiere una revisión1 histórica vigente y su proyección activa en el clon.
-- La fuente puede estar cerrada hoy: sólo se copia a nuevas referencias de prueba.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
CREATE TEMP TABLE p46_material(material jsonb NOT NULL,original jsonb NOT NULL);
CREATE TEMP TABLE p46_resultados(nombre text PRIMARY KEY,resultado jsonb NOT NULL);
CREATE TEMP TABLE p46_consumos(decision_ref text NOT NULL,efecto_ref text NOT NULL,auditoria_ref text NOT NULL);
GRANT SELECT ON p46_material TO vec_personal_ejecutor;
GRANT SELECT,INSERT ON p46_resultados TO vec_personal_ejecutor;
GRANT INSERT ON p46_consumos TO vec_personal_propietario;
-- Se conservan los cuerpos/ACL reales por ROLLBACK; sólo este consumidor
-- es sustituido temporalmente, sin modificar la función Personal46 probada.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $doble$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb; token text:=gen_random_uuid()::text;
BEGIN
 decision_ref:='decision:p46:'||token; efecto_ref:=d->>'recurso_ref';
 huella_efecto_sha256:=d->>'contexto_recurso_huella_sha256';
 consumo_huella_sha256:=encode(sha256(convert_to(token,'UTF8')),'hex');
 auditoria_ref:='auditoria:p46:'||token; consumida_en:=clock_timestamp(); consumo_nuevo:=true;
 INSERT INTO pg_temp.p46_consumos VALUES(decision_ref,efecto_ref,auditoria_ref);
 RETURN NEXT;
END $doble$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $fixture$
DECLARE o vec_personal.ocupacion_empleado_historia%ROWTYPE;
 r vec_personal.relacion_servicio_historia%ROWTYPE; h jsonb; m jsonb; fin date; vpla text; vpue text;
 rel text:='rel_'||repeat('p46r',8); ajena text:='rel_'||repeat('p46a',8);
 ocu text:='ocu_'||repeat('p46o',8); ocuajena text:='ocu_'||repeat('p46x',8);
 proc jsonb:=jsonb_build_object('acto_ref','acto:p46:cese','fuente_ref','fuente:p46:cese',
 'fuente_version',2,'fuente_huella_sha256',repeat('e',64),
 'idempotencia_ref','46000000-0000-4000-8000-000000000001');
BEGIN
 SELECT h.* INTO STRICT o FROM vec_personal.ocupacion_empleado_historia h
 WHERE h.estado='vigente' AND h.revision=1
 AND EXISTS(SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia p
  WHERE p.empleado_ref=h.empleado_ref AND p.estado='activa'
  AND p.vigente_desde<=clock_timestamp() AND clock_timestamp()<p.vigente_hasta)
 ORDER BY h.ocupacion_ref LIMIT 1;
 SELECT * INTO STRICT r FROM vec_personal.relacion_servicio_historia
 WHERE relacion_ref=o.relacion_ref AND revision=o.relacion_revision;
 fin:=o.vigente_desde+1;
 IF r.vigente_desde>o.vigente_desde
 OR (o.vigente_hasta IS NOT NULL AND fin>o.vigente_hasta)
 OR (r.vigente_hasta IS NOT NULL AND fin>r.vigente_hasta) THEN
  RAISE EXCEPTION 'P46: fixture de recorrido sin periodo válido'; END IF;
 -- Dos relaciones del mismo empleado; la segunda permite comprobar cruce
 -- de ocupación pese a compartir empleado, organismo, unidad y procedencia.
 h:=to_jsonb(r)||jsonb_build_object('relacion_ref',rel,'revision',1,'estado','vigente');
 INSERT INTO vec_personal.relacion_servicio_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.relacion_servicio_historia,h)).*;
 h:=h||jsonb_build_object('revision',2,'estado','finalizada','vigente_hasta',fin,
 'acto_ref',proc->>'acto_ref','fuente_ref',proc->>'fuente_ref','fuente_version',2,
 'fuente_huella_sha256',proc->>'fuente_huella_sha256');
 INSERT INTO vec_personal.relacion_servicio_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.relacion_servicio_historia,h)).*;
 h:=h||jsonb_build_object('relacion_ref',ajena,'revision',1,'estado','vigente');
 INSERT INTO vec_personal.relacion_servicio_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.relacion_servicio_historia,h)).*;
 h:=h||jsonb_build_object('revision',2,'estado','finalizada');
 INSERT INTO vec_personal.relacion_servicio_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.relacion_servicio_historia,h)).*;
 h:=to_jsonb(o)||jsonb_build_object('ocupacion_ref',ocu,'revision',1,
 'relacion_ref',rel,'relacion_revision',1);
 INSERT INTO vec_personal.ocupacion_empleado_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.ocupacion_empleado_historia,h)).*;
 INSERT INTO vec_personal.ocupacion_empleado_historia
 SELECT (jsonb_populate_record(NULL::vec_personal.ocupacion_empleado_historia,
 h||jsonb_build_object('ocupacion_ref',ocuajena,'relacion_ref',ajena))).*;
 SELECT 'plantilla:'||plantilla_version_ref::text INTO STRICT vpla
 FROM vec_personal.plaza_plantilla_historia WHERE plaza_ref=o.plaza_ref AND revision=o.plaza_revision;
 vpue:='';
 IF o.puesto_ref IS NOT NULL THEN
  SELECT 'rpt:'||t.rpt_version_ref::text INTO STRICT vpue
  FROM vec_personal.puesto_rpt_historia p JOIN vec_personal.puesto_tipo_historia t
    ON t.tipo_ref=p.tipo_ref AND t.revision=p.tipo_revision
  WHERE p.puesto_ref=o.puesto_ref AND p.revision=o.puesto_revision;
 END IF;
 m:=jsonb_build_object('actor',jsonb_build_object('actor_ref','actor:p46','contexto_actor_ref','contexto:p46',
 'contexto_version',1,'cuenta_ref','cuenta:p46','cuenta_version',1,'perfil_ref','perfil:p46',
 'perfil_version',1,'persona_ref','persona:p46','persona_version',1),
 'esquema','vec.personal.registro-empleado-b2.hecho-cierre-ocupacion.v1','operacion','hecho',
 'empleado_ref',o.empleado_ref,'organismo_ref',o.organismo_ref,'tipo','ocupacion',
 'relacion_ref',rel,'ocupacion_ref',ocu,'relacion_version_esperada',2,'revision_esperada',2,
 'unidad_ref',o.unidad_ref,'regimen',jsonb_build_object('ref','','version',0),
 'modalidad',jsonb_build_object('ref',o.modalidad_ref,'version',(o.catalogo_snapshot#>>'{modalidad,version}')::bigint),
 'situacion',jsonb_build_object('ref','','version',0),'clase_servicio',jsonb_build_object('ref','','version',0),
 'clase_ocupacion',o.clase,'plaza_ref','plaza:'||o.plaza_ref::text,
 'puesto_ref',coalesce('puesto:'||o.puesto_ref::text,''),'version_plaza_ref',vpla,'version_puesto_ref',vpue,
 'dias_reconocidos',0,'periodo_desde','','periodo_hasta','','estado','finalizada',
 'vigente_desde',o.vigente_desde,'vigente_hasta',fin,'procedencia',proc);
 INSERT INTO pg_temp.p46_material VALUES(m,h);
END $fixture$;
-- Preparador del doble: recalcula huella V3 tras cada mutación del material.
CREATE FUNCTION pg_temp.invocar_p46(m jsonb) RETURNS jsonb
 LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog,pg_temp AS $llamar$
DECLARE canon text:=m::text; sha text; recurso text; rh text; c jsonb; d jsonb; x jsonb;
BEGIN
 sha:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 recurso:='{"ambitos":{"organismo_ref":'||to_jsonb(m->>'organismo_ref')::text||
 '},"atributos":{"material_sha256":"'||sha||'","objetivo_ref":'||to_jsonb(m->>'empleado_ref')::text||
 ',"operacion":"hecho"}}';
 rh:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 c:=jsonb_build_object('emitida_en',clock_timestamp()-interval '1 minute','expira_en',clock_timestamp()+interval '1 hour',
 'decision_valida_hasta','2100-01-01T00:00:00Z','operacion','personal.registro_empleado.hecho.registrar',
 'audiencia_consumo','vec_personal.registro_empleado.hecho.v1','efecto_ref',m->>'empleado_ref','huella_efecto_sha256',rh);
 d:=jsonb_build_object('principal_id','actor:p46','perfil_activo_ref','perfil:p46','concedida',true,
 'modulo_id','personal','obligaciones','[]'::jsonb,'valida_hasta','2100-01-01T00:00:00Z',
 'tipo_recurso','hecho_empleado_rrhh','finalidad','registrar_hecho_empleado',
 'campos_permitidos','["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb,
 'accion','personal.registro_empleado.hecho.registrar','recurso_ref',m->>'empleado_ref','contexto_recurso_huella_sha256',rh);
 x:=((m->'actor') - 'actor_ref'::text - 'perfil_ref'::text)||jsonb_build_object('principal_ref','actor:p46',
 'perfil_activo_ref','perfil:p46','esquema','vec.contexto-actor.vinculado.v2');
 RETURN vec_personal.registrar_hecho_empleado_rrhh_v1(canon,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),
 '{}'::bytea,convert_to(x::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
END $llamar$;
REVOKE ALL ON FUNCTION pg_temp.invocar_p46(jsonb) FROM PUBLIC;
CREATE ROLE vec_p46_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_p46_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION pg_temp.invocar_p46(jsonb) TO vec_p46_prueba;
SET SESSION AUTHORIZATION vec_p46_prueba;
DO $casos$
DECLARE m jsonb; n jsonb; caso record; codigo text; mensaje text; contexto text;
BEGIN
 SELECT material INTO STRICT m FROM pg_temp.p46_material;
 FOR caso IN SELECT * FROM (VALUES
 ('relacion_ajena','relacion_ref',to_jsonb('rel_'||repeat('p46a',8)),'42501'),
 ('ocupacion_ajena','ocupacion_ref',to_jsonb('ocu_'||repeat('p46x',8)),'42501'),
 ('revision_ocupacion','revision_esperada','3'::jsonb,'23505'),
 ('revision_relacion','relacion_version_esperada','3'::jsonb,'23505'),
 ('hasta','vigente_hasta',to_jsonb(((m->>'vigente_hasta')::date+1)::text),'42501'),
 ('unidad','unidad_ref','"unidad:ajena"'::jsonb,'42501'),
 ('plaza','plaza_ref','"plaza:46000000-0000-4000-8000-000000000009"'::jsonb,'42501'),
 ('version_plaza','version_plaza_ref','"plantilla:ajena"'::jsonb,'23505'),
 ('v1_con_ocupacion','esquema','"vec.personal.registro-empleado-b2.hecho.v1"'::jsonb,'22023'),
 ('estado','estado','"vigente"'::jsonb,'22023')) c(nombre,campo,valor,esperado) LOOP
  n:=jsonb_set(m,ARRAY[caso.campo],caso.valor); codigo:='ok';
  BEGIN
   PERFORM pg_temp.invocar_p46(n);
  EXCEPTION WHEN others THEN
   GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT,contexto=PG_EXCEPTION_CONTEXT;
  END;
  IF codigo IS DISTINCT FROM caso.esperado THEN
   RAISE EXCEPTION 'P46 negativo %: actual %, esperado %; mensaje %; contexto %',
    caso.nombre,codigo,caso.esperado,mensaje,contexto; END IF;
 END LOOP;
 n:=jsonb_set(m,'{procedencia,fuente_huella_sha256}',to_jsonb(repeat('0',64))); codigo:='ok';
 BEGIN PERFORM pg_temp.invocar_p46(n);
 EXCEPTION WHEN others THEN
   GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT,contexto=PG_EXCEPTION_CONTEXT;
  END;
 IF codigo IS DISTINCT FROM '42501' THEN RAISE EXCEPTION 'P46: procedencia actual %; mensaje %; contexto %',codigo,mensaje,contexto; END IF;
 INSERT INTO pg_temp.p46_resultados VALUES('registro',pg_temp.invocar_p46(m));
 INSERT INTO pg_temp.p46_resultados VALUES('replay',pg_temp.invocar_p46(m));
 -- Una clave nueva tras el cierre no crea otra revisión, mientras el replay
 -- anterior sí recupera el recibo, porque retorna antes de mirar estado actual.
 n:=jsonb_set(m,'{procedencia,idempotencia_ref}','"46000000-0000-4000-8000-000000000002"'::jsonb);
 codigo:='ok';
 BEGIN PERFORM pg_temp.invocar_p46(n);
 EXCEPTION WHEN others THEN
   GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT,contexto=PG_EXCEPTION_CONTEXT;
  END;
 IF codigo IS DISTINCT FROM '23505' THEN RAISE EXCEPTION 'P46: segundo cierre actual %; mensaje %; contexto %',codigo,mensaje,contexto; END IF;
END $casos$;
RESET SESSION AUTHORIZATION;
DO $resultado$
DECLARE m jsonb; o vec_personal.ocupacion_empleado_historia%ROWTYPE;
 p jsonb; q jsonb; antiguo jsonb; r vec_personal.relacion_servicio_historia%ROWTYPE;
BEGIN
 SELECT material,original INTO STRICT m,antiguo FROM pg_temp.p46_material;
 SELECT resultado INTO STRICT p FROM pg_temp.p46_resultados WHERE nombre='registro';
 SELECT resultado INTO STRICT q FROM pg_temp.p46_resultados WHERE nombre='replay';
 IF p->'recibo' IS DISTINCT FROM q->'recibo'
 OR p#>>'{recibo,hecho_ref}' IS DISTINCT FROM m->>'ocupacion_ref'
 OR p#>>'{recibo,version}' IS DISTINCT FROM '2'
 OR q#>>'{acceso_actual,estado_replay}' IS DISTINCT FROM 'replay'
 OR p#>>'{acceso_actual,auditoria_ref}' IS NOT DISTINCT FROM q#>>'{acceso_actual,auditoria_ref}'
 OR (SELECT count(*) FROM pg_temp.p46_consumos)<>2 THEN
  RAISE EXCEPTION 'P46: recibo, replay o consumo transaccional divergente'; END IF;
 SELECT * INTO STRICT o FROM vec_personal.ocupacion_empleado_historia
 WHERE ocupacion_ref=m->>'ocupacion_ref' AND revision=2;
 SELECT * INTO STRICT r FROM vec_personal.relacion_servicio_historia
 WHERE relacion_ref=m->>'relacion_ref' AND revision=2;
 IF o.estado<>'finalizada' OR r.estado<>'finalizada' OR o.relacion_revision<>2
 OR o.vigente_hasta IS DISTINCT FROM r.vigente_hasta
 OR o.vigente_desde IS DISTINCT FROM (antiguo->>'vigente_desde')::date
 OR o.catalogo_snapshot IS DISTINCT FROM antiguo->'catalogo_snapshot'
 OR o.plaza_ref::text IS DISTINCT FROM antiguo->>'plaza_ref'
 OR o.plaza_revision IS DISTINCT FROM (antiguo->>'plaza_revision')::integer
 OR o.puesto_ref::text IS DISTINCT FROM antiguo->>'puesto_ref'
 OR o.puesto_revision IS DISTINCT FROM (antiguo->>'puesto_revision')::integer
 OR o.clase IS DISTINCT FROM antiguo->>'clase'
 OR o.modalidad_ref IS DISTINCT FROM antiguo->>'modalidad_ref'
 OR o.acto_ref IS DISTINCT FROM r.acto_ref OR o.fuente_ref IS DISTINCT FROM r.fuente_ref
 OR o.fuente_version IS DISTINCT FROM r.fuente_version
 OR o.fuente_huella_sha256 IS DISTINCT FROM r.fuente_huella_sha256
 OR o.decision_ref IS DISTINCT FROM p#>>'{recibo,decision_ref}'
 OR o.auditoria_ref IS DISTINCT FROM p#>>'{recibo,auditoria_ref}'
 OR (SELECT count(*) FROM vec_personal.ocupacion_empleado_historia WHERE ocupacion_ref=m->>'ocupacion_ref')<>2
 OR (SELECT count(*) FROM vec_personal.ocupacion_empleado_historia WHERE ocupacion_ref='ocu_'||repeat('p46x',8))<>1
 OR (SELECT count(*) FROM vec_personal.relacion_servicio_historia WHERE relacion_ref IN
 ('rel_'||repeat('p46r',8),'rel_'||repeat('p46a',8)))<>4
 OR (SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo WHERE idempotencia_ref IN
 ('46000000-0000-4000-8000-000000000001','46000000-0000-4000-8000-000000000002'))<>1 THEN
  RAISE EXCEPTION 'P46: historia final, referencias, versiones o atomicidad divergentes'; END IF;
 -- La revisión anterior queda literal, salvo conocido_desde fijado por su trigger.
 IF (SELECT to_jsonb(h)-'conocido_desde' FROM vec_personal.ocupacion_empleado_historia h
 WHERE ocupacion_ref=m->>'ocupacion_ref' AND revision=1) IS DISTINCT FROM antiguo-'conocido_desde' THEN
  RAISE EXCEPTION 'P46: revisión anterior alterada'; END IF;
 IF has_function_privilege('vec_personal_ejecutor',
 'vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
  RAISE EXCEPTION 'P46: interna expuesta'; END IF;
 RAISE NOTICE 'P46_OK: relación2/ocupación2 finalizadas, recibo/replay, preservación y 12 negativas sin efectos; doble V3 explícito';
END $resultado$;
ROLLBACK;
