\set ON_ERROR_STOP on
-- B2: lecturas nominales de Personal. Instalar tras Personal 000017 y AD3 000060.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000018:lectura-b2',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.cobertura_ocupaciones_historia') IS NULL
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regprocedure('vec_personal.consultar_registro_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_vacantes_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 000018: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El recibo local conserva solo selector opaco/huellas y no datos de ficha.
CREATE TABLE vec_personal.recibo_lectura_registro_empleado_b2 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^registroconsulta:[0-9a-f-]{36}$'),
 operacion text NOT NULL CHECK(operacion IN ('ficha','vacantes')),
 selector_ref text NOT NULL CHECK(length(selector_ref) BETWEEN 3 AND 160),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 800),
 consultada_en timestamptz(6) NOT NULL
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_lectura_registro_empleado_b2
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_lectura_registro_empleado_b2
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_lectura_registro_empleado_b2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_lectura_registro_empleado_b2 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_lectura_registro_empleado_b2
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.recibo_lectura_registro_empleado_b2 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.traza_registro_empleado_b2_v1(
 p_desde date,p_hasta date,p_conocido timestamptz,p_revision integer,
 p_acto text,p_fuente text,p_fuente_version bigint) RETURNS jsonb
LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
 SELECT jsonb_strip_nulls(jsonb_build_object(
  'desde',p_desde::text,'hasta',p_hasta::text,
  'registrada_en',to_char(p_conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'version',p_revision,'acto_ref',p_acto,'fuente_ref',p_fuente,'fuente_version',p_fuente_version));
$f$;
REVOKE ALL ON FUNCTION vec_personal.traza_registro_empleado_b2_v1(date,date,timestamptz,integer,text,text,bigint) FROM PUBLIC,vec_personal_ejecutor;

-- Helper invocado únicamente por los dos puntos nominales SECURITY DEFINER.
CREATE FUNCTION vec_personal.consultar_registro_empleado_b2_interna(
 p_operacion text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; consumo record;
 fecha date; conocido timestamptz(6); limite integer; cursor text; offset_p integer:=0;
 empleado text; organismo text; selector text; accion text; audiencia text; tipo text; finalidad text;
 campos jsonb; material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 base_sha text; ficha jsonb; pagina jsonb; relaciones jsonb; ocupaciones jsonb; situaciones jsonb; servicios jsonb;
 vacantes jsonb; cardinalidad integer; hay_mas boolean; cursor_siguiente text; version_ficha integer;
 persona text; v_plantilla_ref text; v_plantilla_revision integer; v_rpt_ref text; v_rpt_revision integer; n integer;
 ahora timestamptz(6); recibo text;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_operacion NOT IN ('ficha','vacantes')
    OR p_material IS NULL OR length(p_material)>4096 OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL
    OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'consulta B2 denegada' USING ERRCODE='42501';
 END IF;
 -- 000018 se instala antes de 000020. La lectura permanece cerrada hasta
 -- que las cuatro historias tengan el snapshot publicado de su acto.
 IF pg_catalog.to_regprocedure('vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)') IS NULL
    OR (SELECT count(*) FROM pg_catalog.pg_attribute a
        WHERE a.attrelid IN ('vec_personal.relacion_servicio_historia'::regclass,
          'vec_personal.ocupacion_empleado_historia'::regclass,
          'vec_personal.situacion_empleado_historia'::regclass,
          'vec_personal.servicio_reconocido_historia'::regclass)
          AND a.attname='catalogo_snapshot' AND NOT a.attisdropped
          AND a.atttypid='jsonb'::regtype AND a.attnotnull)<>4 THEN
  RAISE EXCEPTION 'catálogo de Personal no instalado' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
  limite:=(m->>'limite')::integer; cursor:=m->>'cursor';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B2 inválido' USING ERRCODE='22023'; END;
 empleado:=m->>'empleado_ref'; organismo:=m->>'organismo_ref';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
      'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
      'cursor','empleado_ref','esquema','limite','operacion','organismo_ref','perfil_ref','perfil_version',
      'persona_ref','persona_version','vigente_en']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.registro-empleado-b2.consulta.v1'
    OR m->>'operacion' IS DISTINCT FROM p_operacion
    OR fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido)
    OR conocido>transaction_timestamp()
    OR fecha::text IS DISTINCT FROM m->>'vigente_en'
    OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR cursor IS NULL OR empleado IS NULL OR organismo IS NULL OR limite IS NULL
    OR m->>'actor_ref' IS NULL OR m->>'contexto_actor_ref' IS NULL
    OR m->>'cuenta_ref' IS NULL OR m->>'perfil_ref' IS NULL OR m->>'persona_ref' IS NULL
    OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'cuenta_version' !~ '^[1-9][0-9]{0,19}$'
    OR m->>'perfil_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' !~ '^[1-9][0-9]{0,18}$'
    OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR m->>'actor_ref' IS DISTINCT FROM m->>'persona_ref'
    OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'material B2 incompatible' USING ERRCODE='42501';
 END IF;
 IF p_operacion='ficha' THEN
  IF empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
     OR organismo !~ '^[a-z][a-z0-9_:-]{2,159}$' OR limite<>0 OR cursor<>'' THEN
   RAISE EXCEPTION 'selector de ficha inválido' USING ERRCODE='22023'; END IF;
  selector:=empleado; accion:='personal.registro_empleado.ficha.consultar';
  audiencia:='vec_personal.registro_empleado.ficha.v1';tipo:='registro_empleado_rrhh';
  finalidad:='consultar_ficha_empleado';
  campos:='["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb;
 ELSE
  IF empleado<>'' OR organismo !~ '^[a-z][a-z0-9_:-]{2,159}$' OR limite NOT BETWEEN 1 AND 100
     OR cursor !~ '^(|p_[1-9][0-9]{0,6}_[0-9a-f]{64})$' THEN
   RAISE EXCEPTION 'selector de vacantes inválido' USING ERRCODE='22023'; END IF;
  selector:=organismo; accion:='personal.registro_empleado.vacantes.consultar';
  audiencia:='vec_personal.registro_empleado.vacantes.v1';tipo:='vacantes_rrhh';
  finalidad:='consultar_vacantes';
  campos:='["cobertura","corte","cursor","cursor_siguiente","evidencia","limite","organismo_ref","vacantes"]'::jsonb;
 END IF;
 -- json.Marshal de la estructura Go, en orden de campos y bytes UTF-8 exactos.
 material_canon:='{"esquema":'||to_jsonb(m->>'esquema')::text||',"operacion":'||to_jsonb(p_operacion)::text||
  ',"empleado_ref":'||to_jsonb(empleado)::text||',"organismo_ref":'||to_jsonb(organismo)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"limite":'||limite::text||',"cursor":'||to_jsonb(cursor)::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||m->>'contexto_version'||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||m->>'cuenta_version'||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||m->>'perfil_version'||',"persona_ref":'||to_jsonb(m->>'persona_ref')::text||
  ',"persona_version":'||m->>'persona_version'||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material B2 no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{'||CASE WHEN p_operacion='ficha' THEN
  '"empleado_ref":'||to_jsonb(empleado)::text||',"organismo_ref":'||to_jsonb(organismo)::text
  ELSE '"organismo_ref":'||to_jsonb(organismo)::text END||
  '},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"material_sha256":"'||material_sha||'","operacion":'||to_jsonb(p_operacion)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM accion OR d->>'tipo_recurso' IS DISTINCT FROM tipo
    OR d->>'finalidad' IS DISTINCT FROM finalidad OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->>'recurso_ref' IS DISTINCT FROM selector
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM selector OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión B2 divergente' USING ERRCODE='42501'; END IF;
 IF p_operacion='vacantes' THEN
  base_sha:=encode(sha256(convert_to((m-'cursor')::text,'UTF8')),'hex');
  IF cursor<>'' THEN
   offset_p:=split_part(cursor,'_',2)::integer;
   IF offset_p>1000000 OR split_part(cursor,'_',3)<>base_sha THEN
    RAISE EXCEPTION 'cursor B2 divergente' USING ERRCODE='22023'; END IF;
  END IF;
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM selector
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'consumo B2 divergente' USING ERRCODE='42501'; END IF;
 -- La última revisión conocida manda; la vigencia se filtra después.
 -- Una revisión cerrada jamás resucita una anterior del mismo hecho.
 IF p_operacion='ficha' THEN
  SELECT count(DISTINCT r.persona_ref),min(r.persona_ref)
    INTO n,persona
   FROM (SELECT DISTINCT ON (relacion_ref) * FROM vec_personal.relacion_servicio_historia
         WHERE empleado_ref=empleado AND organismo_ref=organismo AND conocido_desde<=conocido
         ORDER BY relacion_ref,conocido_desde DESC,revision DESC) r;
  IF n=0 THEN RAISE EXCEPTION 'empleado no encontrado' USING ERRCODE='P7404'; END IF;
  IF n<>1 THEN RAISE EXCEPTION 'empleado ambiguo' USING ERRCODE='55000'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('relacion_ref',r.relacion_ref,'unidad_ref',r.unidad_ref,
    'organismo_ref',r.organismo_ref,'regimen_ref',r.regimen_ref,'modalidad_ref',r.modalidad_ref,
    'estado',r.estado,'catalogo_snapshot',r.catalogo_snapshot,
    'traza',vec_personal.traza_registro_empleado_b2_v1(r.vigente_desde,r.vigente_hasta,
       r.conocido_desde,r.revision,r.acto_ref,r.fuente_ref,r.fuente_version))
       ORDER BY r.relacion_ref,r.revision),'[]'::jsonb)
    INTO relaciones FROM vec_personal.relacion_servicio_historia r
    WHERE r.empleado_ref=empleado AND r.organismo_ref=organismo AND r.conocido_desde<=conocido;
  SELECT coalesce(jsonb_agg(jsonb_build_object('ocupacion_ref',o.ocupacion_ref,
    'relacion_ref',o.relacion_ref,'plaza_ref',o.plaza_ref::text,
    'puesto_ref',coalesce(o.puesto_ref::text,''),'unidad_ref',o.unidad_ref,
    'modalidad_ref',o.modalidad_ref,'clase',o.clase,'estado',o.estado,
    'catalogo_snapshot',o.catalogo_snapshot,
    'traza',vec_personal.traza_registro_empleado_b2_v1(o.vigente_desde,o.vigente_hasta,
      o.conocido_desde,o.revision,o.acto_ref,o.fuente_ref,o.fuente_version))
      ORDER BY o.ocupacion_ref,o.revision),'[]'::jsonb)
   INTO ocupaciones FROM vec_personal.ocupacion_empleado_historia o
   WHERE o.empleado_ref=empleado AND o.organismo_ref=organismo AND o.conocido_desde<=conocido;
  SELECT coalesce(jsonb_agg(jsonb_build_object('situacion_ref',s.situacion_ref,'relacion_ref',s.relacion_ref,
    'codigo_ref',s.situacion_codigo,'estado',s.estado,
    'catalogo_snapshot',s.catalogo_snapshot,
    'traza',vec_personal.traza_registro_empleado_b2_v1(s.vigente_desde,s.vigente_hasta,
      s.conocido_desde,s.revision,s.acto_ref,s.fuente_ref,s.fuente_version))
      ORDER BY s.situacion_ref,s.revision),'[]'::jsonb)
   INTO situaciones FROM vec_personal.situacion_empleado_historia s
   WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido;
  SELECT coalesce(jsonb_agg(jsonb_build_object('servicio_ref',s.servicio_ref,'relacion_ref',s.relacion_ref,
    'estado',s.estado,'clase_ref',s.clase_ref,'catalogo_snapshot',s.catalogo_snapshot,
    'periodo_desde',s.periodo_desde::text,
    'periodo_hasta',s.periodo_hasta::text,'dias_reconocidos',s.dias_reconocidos,
    'traza',vec_personal.traza_registro_empleado_b2_v1(s.vigente_desde,s.vigente_hasta,
      s.conocido_desde,s.revision,s.acto_ref,s.fuente_ref,s.fuente_version))
      ORDER BY s.servicio_ref,s.revision),'[]'::jsonb)
   INTO servicios FROM vec_personal.servicio_reconocido_historia s
   WHERE s.empleado_ref=empleado AND s.organismo_ref=organismo AND s.conocido_desde<=conocido;
  IF jsonb_array_length(relaciones)>200 OR jsonb_array_length(ocupaciones)>200
    OR jsonb_array_length(situaciones)>200 OR jsonb_array_length(servicios)>200 THEN
   RAISE EXCEPTION 'ficha B2 excede límite' USING ERRCODE='54000'; END IF;
  cardinalidad:=jsonb_array_length(relaciones)+jsonb_array_length(ocupaciones)+
    jsonb_array_length(situaciones)+jsonb_array_length(servicios);
  -- Versión de foto: cardinalidad de hechos inmutables conocidos, monótona al
  -- añadir otra relación v1 o una revisión. No es versión de un acto jurídico.
  version_ficha:=cardinalidad;
  ficha:=jsonb_build_object('empleado_ref',empleado,'organismo_ref',organismo,'persona_ref',persona,
   'corte',jsonb_build_object('vigente_en',fecha::text,'conocido_en',m->>'conocido_en'),
   'version',version_ficha,'eficacia_administrativa',false,'firma_oficial',false,
   'relaciones',relaciones,'ocupaciones',ocupaciones,
   'situaciones',situaciones,'servicios',servicios);
 ELSE
  -- Solo una versión estructural publicada por organismo y corte. La cobertura
  -- completa se refiere precisamente a esa revisión de plantilla.
  SELECT count(*),min(version_ref::text),min(revision) INTO n,v_plantilla_ref,v_plantilla_revision
  FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_plantilla_historia
    WHERE organismo_ref=organismo AND conocido_desde<=conocido
    ORDER BY version_ref,conocido_desde DESC,revision DESC) v
  WHERE estado='publicada' AND NOT retirado AND vigente_desde<=fecha
    AND (vigente_hasta IS NULL OR fecha<vigente_hasta);
  IF n<>1 THEN RAISE EXCEPTION 'plantilla publicada no unívoca' USING ERRCODE='P7401'; END IF;
  SELECT count(*),min(version_ref::text),min(revision) INTO n,v_rpt_ref,v_rpt_revision
  FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_rpt_historia
    WHERE organismo_ref=organismo AND conocido_desde<=conocido
    ORDER BY version_ref,conocido_desde DESC,revision DESC) v
  WHERE estado='publicada' AND NOT retirado AND vigente_desde<=fecha
    AND (vigente_hasta IS NULL OR fecha<vigente_hasta);
  IF n<>1 THEN RAISE EXCEPTION 'RPT publicada no unívoca' USING ERRCODE='P7401'; END IF;
  SELECT count(*),min(plantilla_version_ref::text),min(h.plantilla_revision) INTO n,v_plantilla_ref,v_plantilla_revision
  FROM (SELECT DISTINCT ON (cobertura_ref) * FROM vec_personal.cobertura_ocupaciones_historia
    WHERE organismo_ref=organismo AND conocido_desde<=conocido
    ORDER BY cobertura_ref,conocido_desde DESC,revision DESC) h
  WHERE estado='completa' AND vigente_desde<=fecha AND fecha<vigente_hasta
    AND plantilla_version_ref::text=v_plantilla_ref
    AND h.plantilla_revision=v_plantilla_revision;
  IF n<>1 THEN RAISE EXCEPTION 'cobertura no acreditada' USING ERRCODE='P7401'; END IF;
  WITH plazas AS (SELECT DISTINCT ON (plaza_ref) * FROM vec_personal.plaza_plantilla_historia
      WHERE organismo_ref=organismo AND conocido_desde<=conocido
      ORDER BY plaza_ref,conocido_desde DESC,revision DESC),
  ocup AS (SELECT DISTINCT ON (ocupacion_ref) * FROM vec_personal.ocupacion_empleado_historia
      WHERE organismo_ref=organismo AND conocido_desde<=conocido
      ORDER BY ocupacion_ref,conocido_desde DESC,revision DESC),
  rel AS (SELECT DISTINCT ON (relacion_ref) * FROM vec_personal.relacion_servicio_historia
      WHERE organismo_ref=organismo AND conocido_desde<=conocido
      ORDER BY relacion_ref,conocido_desde DESC,revision DESC),
  vinc AS (SELECT DISTINCT ON (vinculo_ref) * FROM vec_personal.vinculo_plaza_puesto_historia
      WHERE organismo_ref=organismo AND conocido_desde<=conocido
      ORDER BY vinculo_ref,conocido_desde DESC,revision DESC),
  puestos AS (SELECT DISTINCT ON (puesto_ref) * FROM vec_personal.puesto_rpt_historia
      WHERE organismo_ref=organismo AND conocido_desde<=conocido
      ORDER BY puesto_ref,conocido_desde DESC,revision DESC),
  candidatos AS (
   SELECT p.*,vp.puesto_ref,pt.denominacion AS puesto_denominacion
   FROM plazas p
   LEFT JOIN LATERAL (SELECT v.puesto_ref FROM vinc v JOIN puestos pu ON pu.puesto_ref=v.puesto_ref
      JOIN vec_personal.puesto_tipo_historia pt0 ON pt0.tipo_ref=pu.tipo_ref AND pt0.revision=pu.tipo_revision
      WHERE v.plaza_ref=p.plaza_ref AND v.estado='confirmado' AND NOT v.retirado
       AND v.vigente_desde<=fecha AND (v.vigente_hasta IS NULL OR fecha<v.vigente_hasta)
       AND pu.estado_estructural='vigente' AND NOT pu.retirado
       AND pu.vigente_desde<=fecha AND (pu.vigente_hasta IS NULL OR fecha<pu.vigente_hasta)
       AND pt0.rpt_version_ref::text=v_rpt_ref
       AND pt0.rpt_revision=v_rpt_revision
       AND pt0.conocido_desde<=conocido AND pt0.vigente_desde<=fecha
       AND (pt0.vigente_hasta IS NULL OR fecha<pt0.vigente_hasta) AND NOT pt0.retirado
      ORDER BY v.vinculo_ref LIMIT 1) vp ON true
   LEFT JOIN puestos pu ON pu.puesto_ref=vp.puesto_ref
   LEFT JOIN vec_personal.puesto_tipo_historia pt ON pt.tipo_ref=pu.tipo_ref AND pt.revision=pu.tipo_revision
      AND pt.conocido_desde<=conocido AND pt.vigente_desde<=fecha
      AND (pt.vigente_hasta IS NULL OR fecha<pt.vigente_hasta) AND NOT pt.retirado
   WHERE p.plantilla_version_ref::text=v_plantilla_ref
      AND p.plantilla_revision=v_plantilla_revision
      AND p.estado_estructural='vigente' AND p.dotacion_presupuestaria='acreditada' AND NOT p.retirado
      AND p.vigente_desde<=fecha AND (p.vigente_hasta IS NULL OR fecha<p.vigente_hasta)
      AND NOT EXISTS (SELECT 1 FROM vinc v WHERE v.plaza_ref=p.plaza_ref
        AND v.estado='pendiente_reconciliacion' AND NOT v.retirado
        AND v.vigente_desde<=fecha AND (v.vigente_hasta IS NULL OR fecha<v.vigente_hasta))
      AND NOT EXISTS (SELECT 1 FROM ocup o JOIN rel r ON r.relacion_ref=o.relacion_ref
         WHERE o.plaza_ref=p.plaza_ref AND o.estado='vigente'
          AND o.vigente_desde<=fecha AND (o.vigente_hasta IS NULL OR fecha<o.vigente_hasta)
          AND r.estado<>'finalizada' AND r.vigente_desde<=fecha
          AND (r.vigente_hasta IS NULL OR fecha<r.vigente_hasta))
  ), pagina_c AS (SELECT * FROM candidatos ORDER BY plaza_ref OFFSET offset_p LIMIT limite+1)
  SELECT coalesce(jsonb_agg(jsonb_build_object('plaza_ref',p.plaza_ref::text,
     'puesto_ref',coalesce(p.puesto_ref::text,''),'unidad_ref',p.unidad_ref,
     'puesto_denominacion',coalesce(p.puesto_denominacion,''),
     'codigo_plaza_fuente',p.codigo_plaza_fuente,
     'estado_cobertura','vacante_sin_ocupacion',
     'version_plantilla_ref','plantilla:'||v_plantilla_ref,
     'version_rpt_ref','rpt:'||v_rpt_ref,
     'traza',jsonb_strip_nulls(jsonb_build_object(
       'desde',p.vigente_desde::text,'hasta',p.vigente_hasta::text,
       'registrada_en',to_char(p.conocido_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
       'revision_estructural',p.revision,'version_plantilla_ref','plantilla:'||v_plantilla_ref,
       'acto_ref',p.acto_ref,'fuente_ref',p.fuente_ref,
       'fuente_huella_sha256',p.huella_fuente_sha256))) ORDER BY p.plaza_ref)
       FILTER (WHERE num<=limite),'[]'::jsonb),count(*) FILTER (WHERE num<=limite),count(*)>limite
     INTO vacantes,cardinalidad,hay_mas
   FROM (SELECT *,row_number() OVER (ORDER BY plaza_ref) AS num FROM pagina_c) p;
  IF hay_mas THEN cursor_siguiente:='p_'||(offset_p+limite)::text||'_'||base_sha;
  ELSE cursor_siguiente:=''; END IF;
  pagina:=jsonb_build_object('organismo_ref',organismo,
   'corte',jsonb_build_object('vigente_en',fecha::text,'conocido_en',m->>'conocido_en'),
   'limite',limite,'cursor',cursor,'cursor_siguiente',cursor_siguiente,
   'cobertura','completa','vacantes',vacantes);
 END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'consulta B2 caducada' USING ERRCODE='42501'; END IF;
 recibo:='registroconsulta:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_lectura_registro_empleado_b2
  (recibo_ref,operacion,selector_ref,material_sha256,decision_ref,auditoria_ref,
   consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,p_operacion,selector,material_sha,consumo.decision_ref,consumo.auditoria_ref,
   consumo.consumo_huella_sha256,cardinalidad,ahora);
 RETURN jsonb_build_object(CASE WHEN p_operacion='ficha' THEN 'ficha' ELSE 'pagina' END,
   CASE WHEN p_operacion='ficha' THEN ficha ELSE pagina END,
   'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
   'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_registro_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_registro_empleado_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 RETURN vec_personal.consultar_registro_empleado_b2_interna('ficha',p_material,p_capacidad,p_decision,p_motivo,
   p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_registro_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_registro_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_vacantes_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
BEGIN
 RETURN vec_personal.consultar_registro_empleado_b2_interna('vacantes',p_material,p_capacidad,p_decision,p_motivo,
   p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_vacantes_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_vacantes_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
