\set ON_ERROR_STOP on
-- Personal reserva las claves antes del primer acto. Los recibos de B2
-- permiten recuperar efectos confirmados aunque cambie el ContextoActor.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000023',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.plan_incorporacion_ct') IS NOT NULL
    OR to_regclass('vec_personal.clases_ocupacion_plan_ct_catalogo') IS NOT NULL
    OR to_regclass('vec_personal.registro_empleado_b2_recibo') IS NULL
    OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
    OR to_regclass('vec_personal.ocupacion_empleado_historia') IS NULL
    OR to_regprocedure('vec_personal.registrar_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_personal.registrar_hecho_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Personal23: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Configuración técnica de presentación. Esta colección no acredita una
-- clasificación jurídica aprobada; cada versión conserva sus datos y textos.
CREATE TABLE vec_personal.clases_ocupacion_plan_ct_catalogo (
 ref text NOT NULL CHECK(ref='personal:incorporacion_ct:clases_ocupacion'),
 version bigint NOT NULL CHECK(version>0),
 datos_canon text NOT NULL CHECK(octet_length(datos_canon) BETWEEN 1 AND 16384),
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object' AND datos=datos_canon::jsonb),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(convert_to(datos_canon,'UTF8')),'hex')),
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 PRIMARY KEY(ref,version), UNIQUE(ref,version,huella_sha256)
);
CREATE FUNCTION vec_personal.validar_version_clases_ocupacion_plan_ct() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE anterior bigint; opcion jsonb;
BEGIN
 SELECT coalesce(max(version),0) INTO anterior FROM vec_personal.clases_ocupacion_plan_ct_catalogo WHERE ref=NEW.ref;
 IF NEW.version IS DISTINCT FROM anterior+1
    OR ARRAY(SELECT jsonb_object_keys(NEW.datos) ORDER BY 1) IS DISTINCT FROM ARRAY['opciones']
    OR jsonb_typeof(NEW.datos->'opciones') IS DISTINCT FROM 'array'
    OR jsonb_array_length(NEW.datos->'opciones') NOT BETWEEN 1 AND 64
    OR (SELECT count(DISTINCT o->>'valor') FROM jsonb_array_elements(NEW.datos->'opciones') o)
       <>jsonb_array_length(NEW.datos->'opciones') THEN
  RAISE EXCEPTION 'Personal23: versión de clases inválida' USING ERRCODE='22023'; END IF;
 FOR opcion IN SELECT value FROM jsonb_array_elements(NEW.datos->'opciones') LOOP
  IF jsonb_typeof(opcion) IS DISTINCT FROM 'object'
     OR ARRAY(SELECT jsonb_object_keys(opcion) ORDER BY 1) IS DISTINCT FROM ARRAY['etiquetas','texto_clave','valor']
     OR opcion->>'valor' IS NULL OR opcion->>'valor' !~ '^[a-z][a-z0-9_]{0,63}$'
     OR opcion->>'valor' = 'reserva'
     OR opcion->>'texto_clave' IS DISTINCT FROM 'rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.'||(opcion->>'valor')
     OR jsonb_typeof(opcion->'etiquetas') IS DISTINCT FROM 'object'
     OR ARRAY(SELECT jsonb_object_keys(opcion->'etiquetas') ORDER BY 1) IS DISTINCT FROM ARRAY['en','es']
     OR jsonb_typeof(opcion->'etiquetas'->'es') IS DISTINCT FROM 'string'
     OR jsonb_typeof(opcion->'etiquetas'->'en') IS DISTINCT FROM 'string'
     OR length(opcion->'etiquetas'->>'es') NOT BETWEEN 1 AND 80
     OR length(opcion->'etiquetas'->>'en') NOT BETWEEN 1 AND 80 THEN
   RAISE EXCEPTION 'Personal23: opción de clase inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_version_clases_ocupacion_plan_ct() FROM PUBLIC,vec_personal_ejecutor;
CREATE TRIGGER version_continua BEFORE INSERT ON vec_personal.clases_ocupacion_plan_ct_catalogo
 FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_version_clases_ocupacion_plan_ct();
DO $datos$
DECLARE canon text:='{"opciones":[{"valor":"titular","texto_clave":"rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.titular","etiquetas":{"es":"Titular","en":"Holder"}},{"valor":"provisional","texto_clave":"rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.provisional","etiquetas":{"es":"Provisional","en":"Provisional"}},{"valor":"temporal","texto_clave":"rrhh.ct.incorporacion.b2.clase_ocupacion.opcion.temporal","etiquetas":{"es":"Temporal","en":"Temporary"}}]}';
BEGIN
 INSERT INTO vec_personal.clases_ocupacion_plan_ct_catalogo VALUES(
  'personal:incorporacion_ct:clases_ocupacion',1,canon,canon::jsonb,
  encode(sha256(convert_to(canon,'UTF8')),'hex'),clock_timestamp());
END $datos$;

CREATE TABLE vec_personal.plan_incorporacion_ct (
 idempotencia_ref uuid PRIMARY KEY,
 plan_ref text NOT NULL UNIQUE CHECK(plan_ref ~ '^perplan_[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^perplanrec_[0-9a-f]{32}$'),
 organismo_ref text NOT NULL,
 datos_canon text NOT NULL CHECK(octet_length(datos_canon) BETWEEN 1 AND 16384),
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object' AND datos=datos_canon::jsonb),
 negocio_sha256 text NOT NULL CHECK(negocio_sha256=encode(sha256(convert_to(datos_canon,'UTF8')),'hex')),
 modo text NOT NULL CHECK(modo IN ('alta_empleado','nueva_relacion')),
 empleado_existente_ref text NOT NULL,
 clave_alta_relacion uuid NOT NULL UNIQUE,
 clave_ocupacion uuid NOT NULL UNIQUE,
 uso_rpt_ref text NOT NULL UNIQUE,
 reserva_rpt_ref text NOT NULL UNIQUE,
 confirmacion_rpt_ref text NOT NULL UNIQUE,
 clases_ocupacion_catalogo_ref text NOT NULL,
 clases_ocupacion_catalogo_version bigint NOT NULL,
 clases_ocupacion_catalogo_huella_sha256 text NOT NULL,
 FOREIGN KEY(clases_ocupacion_catalogo_ref,clases_ocupacion_catalogo_version,clases_ocupacion_catalogo_huella_sha256)
   REFERENCES vec_personal.clases_ocupacion_plan_ct_catalogo(ref,version,huella_sha256),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 CHECK(clave_alta_relacion<>clave_ocupacion),
 CHECK((modo='alta_empleado' AND empleado_existente_ref='') OR
       (modo='nueva_relacion' AND empleado_existente_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$')),
 CHECK(organismo_ref=datos->>'organismo_ref' AND idempotencia_ref::text=datos->>'idempotencia_ref'),
 CHECK(huella_sha256=encode(sha256(convert_to(negocio_sha256||'|'||plan_ref||'|'||recibo_ref||'|'||modo||'|'||empleado_existente_ref||'|'||clave_alta_relacion::text||'|'||clave_ocupacion::text||'|'||uso_rpt_ref||'|'||reserva_rpt_ref||'|'||confirmacion_rpt_ref||'|'||clases_ocupacion_catalogo_ref||'|'||clases_ocupacion_catalogo_version::text||'|'||clases_ocupacion_catalogo_huella_sha256,'UTF8')),'hex'))
);
CREATE UNIQUE INDEX plan_incorporacion_ct_origen_uq ON vec_personal.plan_incorporacion_ct (organismo_ref,(datos->>'origen_ct_ref'));
CREATE TABLE vec_personal.ejecucion_plan_incorporacion_ct (
 plan_ref text PRIMARY KEY REFERENCES vec_personal.plan_incorporacion_ct(plan_ref),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^perplaneje_[0-9a-f]{32}$'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_alta_relacion_ref text NOT NULL REFERENCES vec_personal.registro_empleado_b2_recibo(recibo_ref),
 recibo_ocupacion_ref text NOT NULL REFERENCES vec_personal.registro_empleado_b2_recibo(recibo_ref),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_personal.acceso_plan_incorporacion_ct (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^perplanacc_[0-9a-f]{32}$'),
 operacion text NOT NULL CHECK(operacion IN ('preparar','consultar','ejecutar','confirmar','seleccionar','clases_ocupacion')),
 selector_ref text NOT NULL,
 actor_ref text NOT NULL,
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 consultada_en timestamptz(6) NOT NULL CHECK(isfinite(consultada_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_incorporacion_ct','ejecucion_plan_incorporacion_ct','acceso_plan_incorporacion_ct','clases_ocupacion_plan_ct_catalogo'] LOOP
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_interno ON vec_personal.%I FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true)',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',t);
 END LOOP;
END $tablas$;
REVOKE ALL ON TYPE vec_personal.plan_incorporacion_ct,vec_personal.ejecucion_plan_incorporacion_ct,vec_personal.acceso_plan_incorporacion_ct,vec_personal.clases_ocupacion_plan_ct_catalogo FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.recibo_plan_ct_interno(p vec_personal.registro_empleado_b2_recibo)
RETURNS jsonb LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('recibo_ref',(p).recibo_ref,'empleado_ref',(p).empleado_ref,
 'relacion_ref',(p).relacion_ref,'tipo',(p).tipo,'version',(p).version,'registrado_en',(p).registrado_en,
 'decision_ref',(p).decision_ref,'efecto_ref',(p).efecto_ref,'eficacia_administrativa',false,
 'firma_oficial',false,'consumo_huella_sha256',(p).consumo_huella_sha256,'auditoria_ref',(p).auditoria_ref)
 || CASE WHEN (p).operacion='alta' THEN jsonb_build_object('proyeccion_ref',(p).proyeccion_ref)
 ELSE jsonb_build_object('hecho_ref',(p).hecho_ref) END;
$f$;
REVOKE ALL ON FUNCTION vec_personal.recibo_plan_ct_interno(vec_personal.registro_empleado_b2_recibo) FROM PUBLIC,vec_personal_ejecutor;


CREATE FUNCTION vec_personal.validar_clase_plan_ct_interna(p vec_personal.plan_incorporacion_ct)
RETURNS void LANGUAGE plpgsql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM vec_personal.clases_ocupacion_plan_ct_catalogo c
   WHERE c.ref=p.clases_ocupacion_catalogo_ref AND c.version=p.clases_ocupacion_catalogo_version
    AND c.huella_sha256=p.clases_ocupacion_catalogo_huella_sha256
    AND EXISTS (SELECT 1 FROM jsonb_array_elements(c.datos->'opciones') o WHERE o->>'valor'=p.datos->>'clase_ocupacion')) THEN
  RAISE EXCEPTION 'Personal23: clase ajena al catálogo original' USING ERRCODE='23505'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_clase_plan_ct_interna(vec_personal.plan_incorporacion_ct) FROM PUBLIC,vec_personal_ejecutor;

-- No reproduce material de un actor antiguo. Coteja el negocio con los hechos
-- originales y las claves reservadas; nunca acepta un recibo solo por su ID.
CREATE FUNCTION vec_personal.estado_plan_ct_interno(p vec_personal.plan_incorporacion_ct)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE a vec_personal.registro_empleado_b2_recibo%ROWTYPE;
 o vec_personal.registro_empleado_b2_recibo%ROWTYPE;
 r vec_personal.relacion_servicio_historia%ROWTYPE;
 h vec_personal.ocupacion_empleado_historia%ROWTYPE;
 e vec_personal.ejecucion_plan_incorporacion_ct%ROWTYPE;
 da jsonb:=p.datos; proc jsonb:=p.datos->'procedencia'; ar jsonb; oc jsonb;
 estado text:='preparado'; ejec_sha text:=''; ejec_rec text:='';
BEGIN
 PERFORM vec_personal.validar_clase_plan_ct_interna(p);
 SELECT * INTO a FROM vec_personal.registro_empleado_b2_recibo WHERE idempotencia_ref=p.clave_alta_relacion;
 IF FOUND THEN
  SELECT * INTO r FROM vec_personal.relacion_servicio_historia WHERE relacion_ref=a.relacion_ref AND revision=a.version;
  IF NOT FOUND OR a.version<>1 OR a.tipo IS DISTINCT FROM (CASE p.modo WHEN 'alta_empleado' THEN 'alta' ELSE 'relacion' END)
     OR a.operacion IS DISTINCT FROM (CASE p.modo WHEN 'alta_empleado' THEN 'alta' ELSE 'hecho' END)
     OR a.efecto_ref IS DISTINCT FROM (CASE p.modo WHEN 'alta_empleado' THEN da->>'persona_ref' ELSE p.empleado_existente_ref END)
     OR (p.modo='nueva_relacion' AND a.empleado_ref IS DISTINCT FROM p.empleado_existente_ref)
     OR r.persona_ref IS DISTINCT FROM da->>'persona_ref' OR r.empleado_ref IS DISTINCT FROM a.empleado_ref
     OR r.organismo_ref IS DISTINCT FROM p.organismo_ref OR r.unidad_ref IS DISTINCT FROM da->>'unidad_ref'
     OR r.vigente_desde IS DISTINCT FROM (da->>'desde')::date
     OR r.vigente_hasta IS DISTINCT FROM NULLIF(da->>'hasta','')::date
     OR r.estado IS DISTINCT FROM 'vigente' OR r.regimen_ref IS DISTINCT FROM da->'regimen'->>'ref'
     OR r.modalidad_ref IS DISTINCT FROM da->'modalidad'->>'ref'
     OR r.acto_ref IS DISTINCT FROM proc->>'acto_ref' OR r.fuente_ref IS DISTINCT FROM proc->>'fuente_ref'
     OR r.fuente_version IS DISTINCT FROM (proc->>'fuente_version')::bigint
     OR r.fuente_huella_sha256 IS DISTINCT FROM proc->>'fuente_huella_sha256'
     OR r.catalogo_snapshot->'regimen'->>'version' IS DISTINCT FROM da->'regimen'->>'version'
     OR r.catalogo_snapshot->'modalidad'->>'version' IS DISTINCT FROM da->'modalidad'->>'version'
     OR a.catalogo_snapshot IS DISTINCT FROM r.catalogo_snapshot THEN
   RAISE EXCEPTION 'Personal23: recibo de relación divergente' USING ERRCODE='23505'; END IF;
  ar:=vec_personal.recibo_plan_ct_interno(a); estado:='relacion_registrada';
 END IF;
 SELECT * INTO o FROM vec_personal.registro_empleado_b2_recibo WHERE idempotencia_ref=p.clave_ocupacion;
 IF FOUND THEN
  SELECT * INTO h FROM vec_personal.ocupacion_empleado_historia WHERE ocupacion_ref=o.hecho_ref AND revision=o.version;
  IF ar IS NULL OR NOT FOUND OR o.operacion IS DISTINCT FROM 'hecho' OR o.tipo IS DISTINCT FROM 'ocupacion'
     OR o.version<>1 OR o.empleado_ref IS DISTINCT FROM a.empleado_ref OR o.efecto_ref IS DISTINCT FROM a.empleado_ref
     OR o.relacion_ref IS DISTINCT FROM a.relacion_ref OR h.relacion_ref IS DISTINCT FROM a.relacion_ref
     OR h.relacion_revision IS DISTINCT FROM a.version::integer
     OR h.empleado_ref IS DISTINCT FROM a.empleado_ref OR h.organismo_ref IS DISTINCT FROM p.organismo_ref
     OR h.unidad_ref IS DISTINCT FROM da->>'unidad_ref'
     OR h.vigente_desde IS DISTINCT FROM (da->>'desde')::date
     OR h.vigente_hasta IS DISTINCT FROM NULLIF(da->>'hasta','')::date
     OR h.plaza_ref::text IS DISTINCT FROM replace(da->>'plaza_ref','plaza:','')
     OR h.puesto_ref::text IS DISTINCT FROM replace(da->>'puesto_ref','puesto:','')
     OR h.plaza_revision IS DISTINCT FROM (da->>'revision_plaza')::integer
     OR h.puesto_revision IS DISTINCT FROM (da->>'revision_puesto')::integer
     OR h.clase IS DISTINCT FROM da->>'clase_ocupacion' OR h.estado IS DISTINCT FROM 'vigente'
     OR h.modalidad_ref IS DISTINCT FROM da->'modalidad'->>'ref'
     OR h.catalogo_snapshot->'modalidad'->>'version' IS DISTINCT FROM da->'modalidad'->>'version'
     OR o.catalogo_snapshot IS DISTINCT FROM h.catalogo_snapshot
     OR h.acto_ref IS DISTINCT FROM proc->>'acto_ref' OR h.fuente_ref IS DISTINCT FROM proc->>'fuente_ref'
     OR h.fuente_version IS DISTINCT FROM (proc->>'fuente_version')::bigint
     OR h.fuente_huella_sha256 IS DISTINCT FROM proc->>'fuente_huella_sha256'
     OR NOT EXISTS (SELECT 1 FROM vec_personal.plaza_plantilla_historia pl WHERE pl.plaza_ref=h.plaza_ref
       AND pl.revision=h.plaza_revision AND 'plantilla:'||pl.plantilla_version_ref::text=da->>'version_plantilla_ref')
     OR NOT EXISTS (SELECT 1 FROM vec_personal.puesto_rpt_historia pu JOIN vec_personal.puesto_tipo_historia pt
       ON pt.tipo_ref=pu.tipo_ref AND pt.revision=pu.tipo_revision WHERE pu.puesto_ref=h.puesto_ref
       AND pu.revision=h.puesto_revision AND 'rpt:'||pt.rpt_version_ref::text=da->>'version_rpt_ref') THEN
   RAISE EXCEPTION 'Personal23: recibo de ocupación divergente' USING ERRCODE='23505'; END IF;
  oc:=vec_personal.recibo_plan_ct_interno(o); estado:='ocupacion_registrada';
 END IF;
 SELECT * INTO e FROM vec_personal.ejecucion_plan_incorporacion_ct WHERE plan_ref=p.plan_ref;
 IF FOUND THEN
  IF oc IS NULL OR e.recibo_alta_relacion_ref IS DISTINCT FROM a.recibo_ref
     OR e.recibo_ocupacion_ref IS DISTINCT FROM o.recibo_ref
     OR e.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(p.huella_sha256||'|'||a.recibo_ref||'|'||o.recibo_ref,'UTF8')),'hex') THEN
   RAISE EXCEPTION 'Personal23: ejecución divergente' USING ERRCODE='55000'; END IF;
  estado:='ejecutado'; ejec_rec:=e.recibo_ref; ejec_sha:=e.huella_sha256;
 END IF;
 RETURN jsonb_build_object('plan',jsonb_build_object('plan_ref',p.plan_ref,'recibo_ref',p.recibo_ref,
   'version',1,'huella_sha256',p.huella_sha256,'datos',p.datos,'modo',p.modo,
   'empleado_existente_ref',p.empleado_existente_ref,'clave_alta_relacion',p.clave_alta_relacion,
   'clave_ocupacion',p.clave_ocupacion,'uso_rpt_ref',p.uso_rpt_ref,'reserva_rpt_ref',p.reserva_rpt_ref,
   'confirmacion_rpt_ref',p.confirmacion_rpt_ref,
   'clases_ocupacion_catalogo_ref',p.clases_ocupacion_catalogo_ref,
   'clases_ocupacion_catalogo_version',p.clases_ocupacion_catalogo_version,
   'clases_ocupacion_catalogo_huella_sha256',p.clases_ocupacion_catalogo_huella_sha256),'estado',estado,'recibo_alta_relacion',ar,
   'recibo_ocupacion',oc,'ejecucion_recibo_ref',ejec_rec,'ejecucion_huella_sha256',ejec_sha);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.estado_plan_ct_interno(vec_personal.plan_incorporacion_ct) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.validar_datos_plan_ct_interno(d jsonb,p_canon text) RETURNS void
LANGUAGE plpgsql IMMUTABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE k text; v text; canon text:='{';
BEGIN
 IF jsonb_typeof(d) IS DISTINCT FROM 'object' OR ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM ARRAY['catalogo_rpt_categoria', 'catalogo_rpt_huella_sha256', 'catalogo_rpt_id', 'catalogo_rpt_modulo', 'catalogo_rpt_version', 'clase_ocupacion', 'desde', 'expediente_ref', 'expediente_version', 'fuente_bolsa_huella_sha256', 'fuente_bolsa_recibo_ref', 'fuente_bolsa_ref', 'fuente_bolsa_version', 'fuente_organizacion_huella_sha256', 'fuente_organizacion_ref', 'hasta', 'idempotencia_ref', 'modalidad', 'organismo_ref', 'origen_ct_huella_sha256', 'origen_ct_recibo_ref', 'origen_ct_ref', 'persona_ref', 'persona_version', 'plaza_ref', 'procedencia', 'puesto_ref', 'regimen', 'revision_plaza', 'revision_puesto', 'unidad_ref', 'version_plantilla_ref', 'version_rpt_ref', 'vinculo_ct_recibo_ref'] THEN
 RAISE EXCEPTION 'Personal23: datos inválidos' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(d->'idempotencia_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||'"idempotencia_ref":'||to_jsonb(d->>'idempotencia_ref')::text;
 IF jsonb_typeof(d->'origen_ct_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"origen_ct_ref":'||to_jsonb(d->>'origen_ct_ref')::text;
 IF jsonb_typeof(d->'origen_ct_recibo_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"origen_ct_recibo_ref":'||to_jsonb(d->>'origen_ct_recibo_ref')::text;
 IF jsonb_typeof(d->'origen_ct_huella_sha256') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"origen_ct_huella_sha256":'||to_jsonb(d->>'origen_ct_huella_sha256')::text;
 IF jsonb_typeof(d->'expediente_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"expediente_ref":'||to_jsonb(d->>'expediente_ref')::text;
 IF jsonb_typeof(d->'expediente_version') IS DISTINCT FROM 'number' OR d->>'expediente_version' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"expediente_version":'||(d->>'expediente_version');
 IF jsonb_typeof(d->'organismo_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"organismo_ref":'||to_jsonb(d->>'organismo_ref')::text;
 IF jsonb_typeof(d->'unidad_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"unidad_ref":'||to_jsonb(d->>'unidad_ref')::text;
 IF jsonb_typeof(d->'persona_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"persona_ref":'||to_jsonb(d->>'persona_ref')::text;
 IF jsonb_typeof(d->'persona_version') IS DISTINCT FROM 'number' OR d->>'persona_version' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"persona_version":'||(d->>'persona_version');
 IF jsonb_typeof(d->'fuente_bolsa_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_bolsa_ref":'||to_jsonb(d->>'fuente_bolsa_ref')::text;
 IF jsonb_typeof(d->'fuente_bolsa_version') IS DISTINCT FROM 'number' OR d->>'fuente_bolsa_version' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_bolsa_version":'||(d->>'fuente_bolsa_version');
 IF jsonb_typeof(d->'fuente_bolsa_recibo_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_bolsa_recibo_ref":'||to_jsonb(d->>'fuente_bolsa_recibo_ref')::text;
 IF jsonb_typeof(d->'fuente_bolsa_huella_sha256') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_bolsa_huella_sha256":'||to_jsonb(d->>'fuente_bolsa_huella_sha256')::text;
 IF jsonb_typeof(d->'regimen') IS DISTINCT FROM 'object' OR ARRAY(SELECT jsonb_object_keys(d->'regimen') ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version'] OR jsonb_typeof(d->'regimen'->'version') IS DISTINCT FROM 'number' OR d->'regimen'->>'version' !~ '^[1-9][0-9]{0,9}$' OR d->'regimen'->>'ref' !~ '^[a-z][a-z0-9_:-]{2,159}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"regimen":'||'{"ref":'||to_jsonb(d->'regimen'->>'ref')::text||',"version":'||(d->'regimen'->>'version')||'}';
 IF jsonb_typeof(d->'modalidad') IS DISTINCT FROM 'object' OR ARRAY(SELECT jsonb_object_keys(d->'modalidad') ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version'] OR jsonb_typeof(d->'modalidad'->'version') IS DISTINCT FROM 'number' OR d->'modalidad'->>'version' !~ '^[1-9][0-9]{0,9}$' OR d->'modalidad'->>'ref' !~ '^[a-z][a-z0-9_:-]{2,159}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"modalidad":'||'{"ref":'||to_jsonb(d->'modalidad'->>'ref')::text||',"version":'||(d->'modalidad'->>'version')||'}';
 IF jsonb_typeof(d->'desde') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"desde":'||to_jsonb(d->>'desde')::text;
 IF jsonb_typeof(d->'hasta') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"hasta":'||to_jsonb(d->>'hasta')::text;
 IF jsonb_typeof(d->'plaza_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"plaza_ref":'||to_jsonb(d->>'plaza_ref')::text;
 IF jsonb_typeof(d->'puesto_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"puesto_ref":'||to_jsonb(d->>'puesto_ref')::text;
 IF jsonb_typeof(d->'clase_ocupacion') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"clase_ocupacion":'||to_jsonb(d->>'clase_ocupacion')::text;
 IF jsonb_typeof(d->'version_plantilla_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"version_plantilla_ref":'||to_jsonb(d->>'version_plantilla_ref')::text;
 IF jsonb_typeof(d->'version_rpt_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"version_rpt_ref":'||to_jsonb(d->>'version_rpt_ref')::text;
 IF jsonb_typeof(d->'revision_plaza') IS DISTINCT FROM 'number' OR d->>'revision_plaza' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"revision_plaza":'||(d->>'revision_plaza');
 IF jsonb_typeof(d->'revision_puesto') IS DISTINCT FROM 'number' OR d->>'revision_puesto' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"revision_puesto":'||(d->>'revision_puesto');
 IF jsonb_typeof(d->'fuente_organizacion_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_organizacion_ref":'||to_jsonb(d->>'fuente_organizacion_ref')::text;
 IF jsonb_typeof(d->'fuente_organizacion_huella_sha256') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"fuente_organizacion_huella_sha256":'||to_jsonb(d->>'fuente_organizacion_huella_sha256')::text;
 IF jsonb_typeof(d->'catalogo_rpt_id') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"catalogo_rpt_id":'||to_jsonb(d->>'catalogo_rpt_id')::text;
 IF jsonb_typeof(d->'catalogo_rpt_modulo') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"catalogo_rpt_modulo":'||to_jsonb(d->>'catalogo_rpt_modulo')::text;
 IF jsonb_typeof(d->'catalogo_rpt_categoria') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"catalogo_rpt_categoria":'||to_jsonb(d->>'catalogo_rpt_categoria')::text;
 IF jsonb_typeof(d->'catalogo_rpt_version') IS DISTINCT FROM 'number' OR d->>'catalogo_rpt_version' !~ '^[1-9][0-9]{0,18}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"catalogo_rpt_version":'||(d->>'catalogo_rpt_version');
 IF jsonb_typeof(d->'catalogo_rpt_huella_sha256') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"catalogo_rpt_huella_sha256":'||to_jsonb(d->>'catalogo_rpt_huella_sha256')::text;
 IF jsonb_typeof(d->'vinculo_ct_recibo_ref') IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"vinculo_ct_recibo_ref":'||to_jsonb(d->>'vinculo_ct_recibo_ref')::text;
 IF jsonb_typeof(d->'procedencia') IS DISTINCT FROM 'object' OR ARRAY(SELECT jsonb_object_keys(d->'procedencia') ORDER BY 1) IS DISTINCT FROM ARRAY['acto_ref','fuente_huella_sha256','fuente_ref','fuente_version','idempotencia_ref'] OR d->'procedencia'->>'acto_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$' OR d->'procedencia'->>'fuente_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$' OR d->'procedencia'->>'fuente_version' !~ '^[1-9][0-9]{0,18}$' OR d->'procedencia'->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$' OR d->'procedencia'->>'idempotencia_ref' !~ '^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' THEN RAISE EXCEPTION 'Personal23: campo de datos inválido' USING ERRCODE='22023'; END IF;
 canon:=canon||',"procedencia":'||'{"acto_ref":'||to_jsonb(d->'procedencia'->>'acto_ref')::text||',"fuente_ref":'||to_jsonb(d->'procedencia'->>'fuente_ref')::text||',"fuente_version":'||(d->'procedencia'->>'fuente_version')||',"fuente_huella_sha256":'||to_jsonb(d->'procedencia'->>'fuente_huella_sha256')::text||',"idempotencia_ref":'||to_jsonb(d->'procedencia'->>'idempotencia_ref')::text||'}';
 canon:=canon||'}';
 IF canon IS DISTINCT FROM p_canon THEN RAISE EXCEPTION 'Personal23: datos no canónicos' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['origen_ct_ref','origen_ct_recibo_ref','expediente_ref','organismo_ref','unidad_ref','fuente_bolsa_ref','fuente_bolsa_recibo_ref','version_plantilla_ref','version_rpt_ref','fuente_organizacion_ref','catalogo_rpt_id','catalogo_rpt_modulo','catalogo_rpt_categoria','vinculo_ct_recibo_ref'] LOOP
 IF d->>k !~ '^[a-z][a-z0-9_:-]{2,159}$' THEN RAISE EXCEPTION 'Personal23: referencia inválida' USING ERRCODE='22023'; END IF; END LOOP;
 FOREACH k IN ARRAY ARRAY['origen_ct_huella_sha256','fuente_bolsa_huella_sha256','fuente_organizacion_huella_sha256','catalogo_rpt_huella_sha256'] LOOP
 IF d->>k !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'Personal23: huella inválida' USING ERRCODE='22023'; END IF; END LOOP;
 IF d->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR d->>'idempotencia_ref' !~ '^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' OR d->>'plaza_ref' !~ '^plaza:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' OR d->>'puesto_ref' !~ '^puesto:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' OR d->>'desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR (d->>'hasta'<>'' AND d->>'hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$') OR d->>'clase_ocupacion' !~ '^[a-z][a-z0-9_]{0,63}$' THEN RAISE EXCEPTION 'Personal23: datos de incorporación inválidos' USING ERRCODE='22023'; END IF;
 IF NOT isfinite((d->>'desde')::date) OR (d->>'hasta'<>'' AND (NOT isfinite((d->>'hasta')::date) OR (d->>'hasta')::date<=(d->>'desde')::date)) THEN RAISE EXCEPTION 'Personal23: periodo inválido' USING ERRCODE='22023'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_datos_plan_ct_interno(jsonb,text) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.seleccion_plan_ct_interna(p_org text,s jsonb) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE pl vec_personal.plaza_plantilla_historia%ROWTYPE; pu vec_personal.puesto_rpt_historia%ROWTYPE;
 pt vec_personal.puesto_tipo_historia%ROWTYPE; vp vec_personal.version_plantilla_historia%ROWTYPE;
 vr vec_personal.version_rpt_historia%ROWTYPE; n bigint; fecha date;
BEGIN
 fecha:=(s->>'desde')::date;
 SELECT * INTO pl FROM vec_personal.plaza_plantilla_historia
 WHERE plaza_ref=replace(s->>'plaza_ref','plaza:','')::uuid ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND OR pl.organismo_ref IS DISTINCT FROM p_org OR pl.retirado
    OR pl.estado_estructural<>'vigente' OR pl.dotacion_presupuestaria<>'acreditada'
    OR pl.vigente_desde>fecha OR (pl.vigente_hasta IS NOT NULL AND fecha>=pl.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal23: plaza no acreditada' USING ERRCODE='P7404'; END IF;
 SELECT * INTO pu FROM vec_personal.puesto_rpt_historia
 WHERE puesto_ref=replace(s->>'puesto_ref','puesto:','')::uuid ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND OR pu.organismo_ref IS DISTINCT FROM p_org OR pu.unidad_ref IS DISTINCT FROM pl.unidad_ref
    OR pu.retirado OR pu.estado_estructural<>'vigente' OR pu.vigente_desde>fecha
    OR (pu.vigente_hasta IS NOT NULL AND fecha>=pu.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal23: puesto no acreditado' USING ERRCODE='P7404'; END IF;
 SELECT * INTO pt FROM vec_personal.puesto_tipo_historia WHERE tipo_ref=pu.tipo_ref ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND OR pt.revision IS DISTINCT FROM pu.tipo_revision OR pt.organismo_ref IS DISTINCT FROM p_org
    OR pt.retirado OR pt.vigente_desde>fecha OR (pt.vigente_hasta IS NOT NULL AND fecha>=pt.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal23: tipo de puesto no acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO vp FROM vec_personal.version_plantilla_historia
 WHERE version_ref=pl.plantilla_version_ref ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND OR vp.revision IS DISTINCT FROM pl.plantilla_revision OR vp.organismo_ref IS DISTINCT FROM p_org
    OR vp.estado<>'publicada' OR vp.retirado OR vp.vigente_desde>fecha
    OR (vp.vigente_hasta IS NOT NULL AND fecha>=vp.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal23: plantilla no publicada' USING ERRCODE='42501'; END IF;
 SELECT * INTO vr FROM vec_personal.version_rpt_historia WHERE version_ref=pt.rpt_version_ref ORDER BY revision DESC LIMIT 1;
 IF NOT FOUND OR vr.revision IS DISTINCT FROM pt.rpt_revision OR vr.organismo_ref IS DISTINCT FROM p_org
    OR vr.estado<>'publicada' OR vr.retirado OR vr.vigente_desde>fecha
    OR (vr.vigente_hasta IS NOT NULL AND fecha>=vr.vigente_hasta) THEN
  RAISE EXCEPTION 'Personal23: RPT no publicada' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n FROM (SELECT DISTINCT ON (vinculo_ref) *
  FROM vec_personal.vinculo_plaza_puesto_historia WHERE plaza_ref=pl.plaza_ref
  ORDER BY vinculo_ref,revision DESC) v WHERE NOT v.retirado AND v.estado<>'terminado'
  AND v.vigente_desde<=fecha AND (v.vigente_hasta IS NULL OR fecha<v.vigente_hasta);
 IF n<>1 OR NOT EXISTS (SELECT 1 FROM (SELECT DISTINCT ON (vinculo_ref) *
    FROM vec_personal.vinculo_plaza_puesto_historia WHERE plaza_ref=pl.plaza_ref
    ORDER BY vinculo_ref,revision DESC) v WHERE NOT v.retirado AND v.estado='confirmado'
    AND v.plaza_revision=pl.revision AND v.puesto_ref=pu.puesto_ref AND v.puesto_revision=pu.revision
    AND v.organismo_ref=p_org AND v.vigente_desde<=fecha AND (v.vigente_hasta IS NULL OR fecha<v.vigente_hasta)) THEN
  RAISE EXCEPTION 'Personal23: vínculo estructural no unívoco' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('organismo_ref',p_org,'unidad_ref',pl.unidad_ref,'plaza_ref','plaza:'||pl.plaza_ref,
  'puesto_ref','puesto:'||pu.puesto_ref,'desde',s->>'desde','revision_plaza',pl.revision,
  'revision_puesto',pu.revision,'version_plantilla_ref','plantilla:'||vp.version_ref,
  'version_rpt_ref','rpt:'||vr.version_ref,'plantilla_fuente_ref',vp.fuente_ref,'rpt_fuente_ref',vr.fuente_ref,'revision_plantilla',vp.revision,'revision_rpt',vr.revision,'plantilla_huella_sha256',vp.huella_fuente_sha256,
  'rpt_huella_sha256',vr.huella_fuente_sha256,'fuente_organizacion_ref',pl.fuente_ref,
  'fuente_organizacion_huella_sha256',pl.huella_fuente_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.seleccion_plan_ct_interna(text,jsonb) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.plan_incorporacion_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; a jsonb; datos jsonb; seleccion jsonb; op text; selector text; org text;
 datos_raw text; actor_raw text; canon text; sha text; recurso text; recurso_sha text; clave uuid;
 p vec_personal.plan_incorporacion_ct%ROWTYPE; v_consumo record; catalogo vec_personal.clases_ocupacion_plan_ct_catalogo%ROWTYPE; est jsonb; evidencia jsonb;
 acceso text; k text; n bigint; emp text; modo text; planref text; reciboref text; ka uuid; ko uuid;
 uso text; reserva text; confirmacion text; ph text; dec_hasta timestamptz; cap_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 32768
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal23: plan denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
  a:=m->'actor'; datos:=m->'datos'; op:=m->>'operacion'; selector:=m->>'plan_ref'; org:=m->>'organismo_ref';
  dec_hasta:=(d->>'valida_hasta')::timestamptz; cap_hasta:=(c->>'expira_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal23: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM (CASE WHEN op='seleccionar' THEN ARRAY['actor','datos','esquema','negocio_sha256','operacion','organismo_ref','plan_ref','seleccion'] ELSE ARRAY['actor','datos','esquema','negocio_sha256','operacion','organismo_ref','plan_ref'] END)
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.plan-incorporacion-ct.v1'
    OR op IS NULL OR op NOT IN ('preparar','consultar','ejecutar','confirmar','seleccionar','clases_ocupacion')
    OR org IS NULL OR org !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR selector IS NULL OR jsonb_typeof(a) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(a) ORDER BY 1) IS DISTINCT FROM ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','perfil_ref','perfil_version','persona_ref','persona_version']
    OR a->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR a->>'persona_ref' IS DISTINCT FROM x->>'persona_ref'
    OR a->>'actor_ref' IS DISTINCT FROM a->>'persona_ref'
    OR a->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR a->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR a->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR a->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR a->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR a->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR a->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR dec_hasta IS NULL OR cap_hasta IS NULL OR NOT isfinite(dec_hasta) OR NOT isfinite(cap_hasta)
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta' THEN
  RAISE EXCEPTION 'Personal23: contexto divergente' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','perfil_version','persona_version'] LOOP
  IF jsonb_typeof(a->k) IS DISTINCT FROM 'number' OR a->>k !~ '^[1-9][0-9]{0,19}$' THEN
   RAISE EXCEPTION 'Personal23: versión de actor inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF a->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR a->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR a->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR a->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$' THEN
  RAISE EXCEPTION 'Personal23: actor inválido' USING ERRCODE='22023'; END IF;
 IF op='preparar' THEN
  IF selector !~ '^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$'
     OR m->>'negocio_sha256' IS NULL OR m->>'negocio_sha256' !~ '^[0-9a-f]{64}$'
     OR jsonb_typeof(datos) IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION 'Personal23: preparación inválida' USING ERRCODE='22023'; END IF;
  datos_raw:=substring(p_material FROM ',"datos":(.*),"negocio_sha256":"');
  PERFORM vec_personal.validar_datos_plan_ct_interno(datos,datos_raw);
  IF datos->>'idempotencia_ref' IS DISTINCT FROM selector OR datos->>'organismo_ref' IS DISTINCT FROM org
     OR encode(sha256(convert_to(datos_raw,'UTF8')),'hex') IS DISTINCT FROM m->>'negocio_sha256' THEN
   RAISE EXCEPTION 'Personal23: negocio divergente' USING ERRCODE='22023'; END IF;
 ELSIF op='clases_ocupacion' THEN
  IF selector IS DISTINCT FROM org OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'Personal23: selector de catálogo inválido' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 ELSIF op='seleccionar' THEN
  seleccion:=m->'seleccion';
  IF selector !~ '^plaza:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$' OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' OR jsonb_typeof(seleccion) IS DISTINCT FROM 'object'
     OR ARRAY(SELECT jsonb_object_keys(seleccion) ORDER BY 1) IS DISTINCT FROM ARRAY['desde','plaza_ref','puesto_ref']
     OR seleccion->>'plaza_ref' IS DISTINCT FROM selector
     OR seleccion->>'puesto_ref' IS NULL OR seleccion->>'puesto_ref' !~ '^puesto:[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$'
     OR seleccion->>'desde' IS NULL OR seleccion->>'desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
   RAISE EXCEPTION 'Personal23: selección inválida' USING ERRCODE='22023'; END IF;
  IF NOT isfinite((seleccion->>'desde')::date) THEN RAISE EXCEPTION 'Personal23: fecha inválida' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 ELSE
  IF selector !~ '^perplan_[0-9a-f]{32}$' OR datos IS DISTINCT FROM 'null'::jsonb
     OR m->>'negocio_sha256' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'Personal23: selector inválido' USING ERRCODE='22023'; END IF;
  datos_raw:='null';
 END IF;
 actor_raw:='{"actor_ref":'||to_jsonb(a->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(a->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(a->>'contexto_version')||',"cuenta_ref":'||to_jsonb(a->>'cuenta_ref')::text||
  ',"cuenta_version":'||(a->>'cuenta_version')||',"perfil_ref":'||to_jsonb(a->>'perfil_ref')::text||
  ',"perfil_version":'||(a->>'perfil_version')||',"persona_ref":'||to_jsonb(a->>'persona_ref')::text||
  ',"persona_version":'||(a->>'persona_version')||'}';
 canon:='{"esquema":"vec.personal.plan-incorporacion-ct.v1","operacion":'||to_jsonb(op)::text||
  ',"plan_ref":'||to_jsonb(selector)::text||',"organismo_ref":'||to_jsonb(org)::text||',"datos":'||datos_raw||
  ',"negocio_sha256":'||to_jsonb(m->>'negocio_sha256')::text||',"actor":'||actor_raw||'}';
 IF op='seleccionar' THEN
  canon:=left(canon,length(canon)-1)||',"seleccion":{"plaza_ref":'||to_jsonb(seleccion->>'plaza_ref')::text||',"puesto_ref":'||to_jsonb(seleccion->>'puesto_ref')::text||',"desde":'||to_jsonb(seleccion->>'desde')::text||'}}';
 END IF;
 IF p_material IS DISTINCT FROM canon THEN RAISE EXCEPTION 'Personal23: material no canónico' USING ERRCODE='22023'; END IF;
 sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":'||to_jsonb(selector)::text||',"organismo_ref":'||to_jsonb(org)::text||
  '},"atributos":{"material_sha256":"'||sha||'","operacion":'||to_jsonb(op)::text||'}}';
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM 'personal.plan_incorporacion_ct.'||op
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.plan_incorporacion_ct.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM selector OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'plan_incorporacion_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_incorporacion_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM selector OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN op='seleccionar' THEN '["evidencia","seleccion"]'::jsonb WHEN op='clases_ocupacion' THEN '["catalogo","evidencia"]'::jsonb ELSE '["ejecucion_huella_sha256","ejecucion_recibo_ref","estado","evidencia","plan","recibo_alta_relacion","recibo_ocupacion"]'::jsonb END) THEN
  RAISE EXCEPTION 'Personal23: permiso nominal divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_plan_incorporacion_personal_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM selector
    OR v_consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
    OR v_consumo.consumo_huella_sha256 IS NULL OR v_consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_consumo.consumida_en>=cap_hasta OR v_consumo.consumida_en>=dec_hasta THEN
  RAISE EXCEPTION 'Personal23: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:plan-incorporacion-ct:'||selector,0));
 IF op='clases_ocupacion' THEN
  SELECT * INTO catalogo FROM vec_personal.clases_ocupacion_plan_ct_catalogo
  WHERE ref='personal:incorporacion_ct:clases_ocupacion' ORDER BY version DESC LIMIT 1;
  IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: catálogo no disponible' USING ERRCODE='55000'; END IF;
  est:=jsonb_build_object('catalogo',jsonb_build_object('ref',catalogo.ref,'version',catalogo.version,
    'huella_sha256',catalogo.huella_sha256,'opciones',(SELECT jsonb_agg(jsonb_build_object(
      'valor',o->>'valor','texto_clave',o->>'texto_clave') ORDER BY ord)
     FROM jsonb_array_elements(catalogo.datos->'opciones') WITH ORDINALITY x(o,ord))));
 ELSIF op='seleccionar' THEN
  est:=jsonb_build_object('seleccion',vec_personal.seleccion_plan_ct_interna(org,seleccion));
 ELSIF op='preparar' THEN
  clave:=selector::uuid;
  SELECT * INTO p FROM vec_personal.plan_incorporacion_ct WHERE idempotencia_ref=clave;
  IF FOUND THEN
   IF p.negocio_sha256 IS DISTINCT FROM m->>'negocio_sha256' OR p.datos_canon IS DISTINCT FROM datos_raw
      OR p.organismo_ref IS DISTINCT FROM org THEN
    RAISE EXCEPTION 'Personal23: idempotencia divergente' USING ERRCODE='23505'; END IF;
  ELSE
   seleccion:=vec_personal.seleccion_plan_ct_interna(org,jsonb_build_object(
    'plaza_ref',datos->>'plaza_ref','puesto_ref',datos->>'puesto_ref','desde',datos->>'desde'));
   FOREACH k IN ARRAY ARRAY['version_plantilla_ref','version_rpt_ref','revision_plaza','revision_puesto',
       'fuente_organizacion_ref','fuente_organizacion_huella_sha256','unidad_ref'] LOOP
    IF seleccion->>k IS DISTINCT FROM datos->>k THEN
     RAISE EXCEPTION 'Personal23: fuente estructural del plan divergente' USING ERRCODE='42501'; END IF;
   END LOOP;
   -- La clase debe figurar en la versión vigente antes de reservar claves o actos.
   SELECT * INTO catalogo FROM vec_personal.clases_ocupacion_plan_ct_catalogo
    WHERE ref='personal:incorporacion_ct:clases_ocupacion' ORDER BY version DESC LIMIT 1;
   IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: catálogo no disponible' USING ERRCODE='55000'; END IF;
   IF NOT EXISTS (SELECT 1 FROM jsonb_array_elements(catalogo.datos->'opciones') o
      WHERE o->>'valor'=datos->>'clase_ocupacion') THEN
    RAISE EXCEPTION 'Personal23: clase no publicada' USING ERRCODE='22023'; END IF;
   -- La relación histórica evita una segunda alta tras vencer una proyección.
   -- B2 conserva su control de vigencia al ejecutar la nueva relación.
   PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(datos->>'persona_ref');
   SELECT count(DISTINCT z.empleado_ref),min(z.empleado_ref) INTO n,emp FROM (
     SELECT empleado_ref FROM vec_personal.relacion_servicio_historia WHERE persona_ref=datos->>'persona_ref'
     UNION SELECT empleado_ref FROM vec_personal.proyeccion_empleado_persona_historia WHERE persona_ref=datos->>'persona_ref') z;
   IF n>1 THEN RAISE EXCEPTION 'Personal23: empleado histórico ambiguo' USING ERRCODE='55000'; END IF;
   modo:=CASE WHEN n=0 THEN 'alta_empleado' ELSE 'nueva_relacion' END; emp:=coalesce(emp,'');
   planref:='perplan_'||replace(gen_random_uuid()::text,'-','');
   reciboref:='perplanrec_'||replace(gen_random_uuid()::text,'-','');
   ka:=gen_random_uuid(); ko:=gen_random_uuid();
   uso:='uso:'||gen_random_uuid()::text; reserva:='reserva:'||gen_random_uuid()::text;
   confirmacion:='confirmacion:'||gen_random_uuid()::text;
   ph:=encode(sha256(convert_to((m->>'negocio_sha256')||'|'||planref||'|'||reciboref||'|'||modo||'|'||emp||'|'||ka::text||'|'||ko::text||'|'||uso||'|'||reserva||'|'||confirmacion||'|'||catalogo.ref||'|'||catalogo.version::text||'|'||catalogo.huella_sha256,'UTF8')),'hex');
   INSERT INTO vec_personal.plan_incorporacion_ct VALUES(clave,planref,reciboref,org,datos_raw,datos,
    m->>'negocio_sha256',modo,emp,ka,ko,uso,reserva,confirmacion,catalogo.ref,catalogo.version,catalogo.huella_sha256,ph,v_consumo.decision_ref,
    v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en) RETURNING * INTO p;
  END IF;
 ELSE
  SELECT * INTO p FROM vec_personal.plan_incorporacion_ct WHERE plan_ref=selector AND organismo_ref=org;
  IF NOT FOUND THEN RAISE EXCEPTION 'Personal23: plan no encontrado' USING ERRCODE='P7404'; END IF;
 END IF;
 IF op NOT IN ('seleccionar','clases_ocupacion') THEN est:=vec_personal.estado_plan_ct_interno(p); END IF;
 IF op='ejecutar' AND est->>'estado' IN ('preparado','relacion_registrada') THEN
  BEGIN
   seleccion:=vec_personal.seleccion_plan_ct_interna(org,jsonb_build_object(
    'plaza_ref',p.datos->>'plaza_ref','puesto_ref',p.datos->>'puesto_ref','desde',p.datos->>'desde'));
  -- El helper no consume permisos: estos errores describen cambios de la
  -- estructura reservada. La autorización actual ya se consumió antes.
  EXCEPTION WHEN SQLSTATE 'P7404' OR SQLSTATE '42501' THEN
   RAISE EXCEPTION 'Personal23: estructura reservada cambió' USING ERRCODE='23505';
  END;
  FOREACH k IN ARRAY ARRAY['version_plantilla_ref','version_rpt_ref','revision_plaza','revision_puesto',
      'fuente_organizacion_ref','fuente_organizacion_huella_sha256','unidad_ref'] LOOP
   IF seleccion->>k IS DISTINCT FROM p.datos->>k THEN
    RAISE EXCEPTION 'Personal23: estructura cambió antes de ejecutar' USING ERRCODE='23505'; END IF;
  END LOOP;
 END IF;
 IF op='confirmar' AND est->>'estado'<>'ejecutado' THEN
  IF est->>'estado' IS DISTINCT FROM 'ocupacion_registrada' THEN
   RAISE EXCEPTION 'Personal23: efectos pendientes' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_personal.ejecucion_plan_incorporacion_ct VALUES(p.plan_ref,
    'perplaneje_'||replace(gen_random_uuid()::text,'-',''),
    encode(sha256(convert_to(p.huella_sha256||'|'||(est->'recibo_alta_relacion'->>'recibo_ref')||'|'||(est->'recibo_ocupacion'->>'recibo_ref'),'UTF8')),'hex'),
    est->'recibo_alta_relacion'->>'recibo_ref',est->'recibo_ocupacion'->>'recibo_ref',
    v_consumo.decision_ref,v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en);
  est:=vec_personal.estado_plan_ct_interno(p);
 END IF;
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'Personal23: autorización caducada' USING ERRCODE='42501'; END IF;
 acceso:='perplanacc_'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_personal.acceso_plan_incorporacion_ct VALUES(acceso,op,selector,a->>'actor_ref',sha,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_consumo.consumida_en);
 evidencia:=jsonb_build_object('recibo_ref',acceso,'decision_ref',v_consumo.decision_ref,
  'efecto_ref',selector,'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
  'auditoria_ref',v_consumo.auditoria_ref,'consultada_en',v_consumo.consumida_en);
 RETURN est||jsonb_build_object('evidencia',evidencia);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;


-- B2 conserva su contrato y su único consumo V3. Esta fachada añade el vínculo
-- al plan y coteja recibo/estructura dentro de la misma transacción del acto.
-- SERIALIZABLE acredita una instantánea coherente. No promete observar una
-- publicación de organización que confirme después de fijar esa instantánea.
CREATE FUNCTION vec_personal.registrar_acto_plan_incorporacion_ct_v1(
 p_operacion text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE m jsonb; clave uuid; n bigint; p vec_personal.plan_incorporacion_ct%ROWTYPE;
 resultado jsonb; estado jsonb; seleccion jsonb; k text; esperado jsonb;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_operacion IS NULL OR p_operacion NOT IN ('alta','hecho')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 16384
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal23: acto del plan denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; clave:=(m->'procedencia'->>'idempotencia_ref')::uuid;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal23: material de acto inválido' USING ERRCODE='22023'; END;
 IF clave IS NULL THEN RAISE EXCEPTION 'Personal23: clave de acto ausente' USING ERRCODE='22023'; END IF;
 SELECT count(*) INTO n FROM vec_personal.plan_incorporacion_ct
 WHERE clave_alta_relacion=clave OR clave_ocupacion=clave;
 IF n<>1 THEN RAISE EXCEPTION 'Personal23: acto sin plan exacto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_personal.plan_incorporacion_ct
 WHERE clave_alta_relacion=clave OR clave_ocupacion=clave;
 IF (p_operacion='alta' AND (clave IS DISTINCT FROM p.clave_alta_relacion OR p.modo<>'alta_empleado'))
    OR (p_operacion='hecho' AND clave=p.clave_alta_relacion
      AND (p.modo<>'nueva_relacion' OR m->>'tipo' IS DISTINCT FROM 'relacion'))
    OR (p_operacion='hecho' AND clave=p.clave_ocupacion AND m->>'tipo' IS DISTINCT FROM 'ocupacion') THEN
  RAISE EXCEPTION 'Personal23: modo de acto divergente' USING ERRCODE='42501'; END IF;
 -- El original valida material, ContextoActor, catálogos y autorización actual;
 -- consume la decisión y escribe el efecto. No se consulta Organización antes.
 IF p_operacion='alta' THEN
  resultado:=vec_personal.registrar_empleado_rrhh_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  resultado:=vec_personal.registrar_hecho_empleado_rrhh_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 estado:=vec_personal.estado_plan_ct_interno(p);
 esperado:=CASE WHEN clave=p.clave_alta_relacion THEN estado->'recibo_alta_relacion' ELSE estado->'recibo_ocupacion' END;
 IF esperado IS NULL OR esperado='null'::jsonb OR resultado->'recibo' IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'Personal23: recibo de acto divergente' USING ERRCODE='23505'; END IF;
 BEGIN
  seleccion:=vec_personal.seleccion_plan_ct_interna(p.organismo_ref,jsonb_build_object(
   'plaza_ref',p.datos->>'plaza_ref','puesto_ref',p.datos->>'puesto_ref','desde',p.datos->>'desde'));
 -- Solo se convierten los errores estructurales del helper. Los rechazos de
 -- ContextoActor, V3 o caducidad del acto original conservan su código.
 EXCEPTION WHEN SQLSTATE 'P7404' OR SQLSTATE '42501' THEN
  RAISE EXCEPTION 'Personal23: fuente reservada cambió' USING ERRCODE='23505';
 END;
 FOREACH k IN ARRAY ARRAY['version_plantilla_ref','version_rpt_ref','revision_plaza','revision_puesto',
     'fuente_organizacion_ref','fuente_organizacion_huella_sha256','unidad_ref'] LOOP
  IF seleccion->>k IS DISTINCT FROM p.datos->>k THEN
   RAISE EXCEPTION 'Personal23: fuente de acto divergente' USING ERRCODE='23505'; END IF;
 END LOOP;
 IF clock_timestamp()>=(convert_from(p_capacidad,'UTF8')::jsonb->>'expira_en')::timestamptz
    OR clock_timestamp()>=(convert_from(p_decision,'UTF8')::jsonb->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'Personal23: acto caducado antes de confirmación' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;

-- Puerto nominal entre propietarios. CT conserva su consumo y la evidencia
-- independiente de RPT; esta fachada solo acredita los hechos de Personal.
CREATE FUNCTION vec_personal.probar_origen_incorporacion_plan_v1(
 p_org text,p_exp text,p_ct_plan_ref text,p_ct_plan_version bigint,p_ct_sha text,p_hechos jsonb
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE p vec_personal.plan_incorporacion_ct%ROWTYPE; est jsonb; a jsonb; o jsonb;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
  RAISE EXCEPTION 'Personal23: consumidor de origen denegado' USING ERRCODE='42501'; END IF;
 IF p_org IS NULL OR p_exp IS NULL OR p_ct_plan_ref IS NULL OR p_ct_plan_version IS DISTINCT FROM 1
    OR p_ct_sha IS NULL OR p_ct_sha !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_hechos) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'Personal23: selector de origen inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO p FROM vec_personal.plan_incorporacion_ct
 WHERE plan_ref=p_hechos->>'personal_plan_ref' AND organismo_ref=p_org;
 IF NOT FOUND THEN RETURN NULL; END IF;
 est:=vec_personal.estado_plan_ct_interno(p); a:=est->'recibo_alta_relacion'; o:=est->'recibo_ocupacion';
 IF est->>'estado' IS DISTINCT FROM 'ejecutado'
    OR p.datos->>'expediente_ref' IS DISTINCT FROM p_exp
    OR p.datos->>'origen_ct_ref' IS DISTINCT FROM p_ct_plan_ref
    OR p.datos->>'origen_ct_huella_sha256' IS DISTINCT FROM p_ct_sha
    OR p_hechos->>'personal_plan_version' IS DISTINCT FROM '1'
    OR p_hechos->>'personal_plan_recibo_ref' IS DISTINCT FROM p.recibo_ref
    OR p_hechos->>'personal_plan_sha256' IS DISTINCT FROM p.huella_sha256
    OR p_hechos->>'modo_personal' IS DISTINCT FROM p.modo
    OR p_hechos->>'alta_recibo_ref' IS DISTINCT FROM (CASE WHEN p.modo='alta_empleado' THEN a->>'recibo_ref' ELSE '' END)
    OR p_hechos->>'empleado_ref' IS DISTINCT FROM a->>'empleado_ref'
    OR p_hechos->>'relacion_ref' IS DISTINCT FROM a->>'relacion_ref'
    OR p_hechos->>'relacion_version' IS DISTINCT FROM a->>'version'
    OR p_hechos->>'relacion_recibo_ref' IS DISTINCT FROM a->>'recibo_ref'
    OR p_hechos->>'ocupacion_ref' IS DISTINCT FROM o->>'hecho_ref'
    OR p_hechos->>'ocupacion_version' IS DISTINCT FROM o->>'version'
    OR p_hechos->>'ocupacion_recibo_ref' IS DISTINCT FROM o->>'recibo_ref'
    OR p_hechos->>'rpt_confirmacion_ref' IS DISTINCT FROM p.confirmacion_rpt_ref
    OR p_hechos->>'rpt_recibo_ref' IS NULL OR p_hechos->>'rpt_recibo_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR (SELECT max(revision) FROM vec_personal.relacion_servicio_historia WHERE relacion_ref=a->>'relacion_ref')::text IS DISTINCT FROM a->>'version'
    OR (SELECT max(revision) FROM vec_personal.ocupacion_empleado_historia WHERE ocupacion_ref=o->>'hecho_ref')::text IS DISTINCT FROM o->>'version' THEN
  RAISE EXCEPTION 'Personal23: origen o hechos divergentes' USING ERRCODE='42501'; END IF;
 RETURN p_hechos;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb) FROM PUBLIC,vec_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.probar_origen_incorporacion_plan_v1(text,text,text,bigint,text,jsonb) TO vec_contratacion_temporal_propietario;

-- Los privilegios predeterminados tampoco pueden exponer helpers o tablas.
DO $acl$
DECLARE f record; a record; t record; permitido oid;
BEGIN
 FOR f IN SELECT p.oid,p.proowner,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_personal' AND p.proname IN ('recibo_plan_ct_interno','estado_plan_ct_interno',
   'validar_datos_plan_ct_interno','validar_clase_plan_ct_interna','validar_version_clases_ocupacion_plan_ct','seleccion_plan_ct_interna','plan_incorporacion_ct_v1','registrar_acto_plan_incorporacion_ct_v1','probar_origen_incorporacion_plan_v1') LOOP
  permitido:=CASE WHEN f.proname IN ('plan_incorporacion_ct_v1','registrar_acto_plan_incorporacion_ct_v1') THEN 'vec_personal_ejecutor'::regrole::oid
    WHEN f.proname='probar_origen_incorporacion_plan_v1' THEN 'vec_contratacion_temporal_propietario'::regrole::oid ELSE f.proowner END;
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f.oid AND x.grantee<>f.proowner AND x.grantee<>permitido LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f.oid::regprocedure,
     CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
 END LOOP;
 FOR t IN SELECT c.oid,c.relowner FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_personal' AND c.relname IN ('plan_incorporacion_ct','ejecucion_plan_incorporacion_ct','acceso_plan_incorporacion_ct','clases_ocupacion_plan_ct_catalogo') LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
   WHERE c.oid=t.oid AND x.grantee<>t.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE %s FROM %s',t.oid::regclass,
     CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
