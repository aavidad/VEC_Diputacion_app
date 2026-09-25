\set ON_ERROR_STOP on
-- B2: lista RRHH de empleados del organismo, para elegir la ficha sin teclear
-- referencias. Personal no guarda datos civiles: cada empleado se identifica
-- por sus relaciones vigentes (unidad, puesto, régimen y modalidad). Lectura
-- nominal con consumo V3 (AD3-56) en la misma transacción y recibo propio.
-- Instalar tras Personal 000020 y AD3-56. Sin DOWN tras historia.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000021:lista-empleados-b2',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.entrada_catalogo_registro_empleado_historia') IS NULL
    OR to_regclass('vec_personal.org_nodo_historia') IS NULL
    OR to_regprocedure('vec_personal.traza_registro_empleado_b2_v1(date,date,timestamptz,integer,text,text,bigint)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regclass('vec_personal.recibo_lista_empleados_b2') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 000021: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El recibo conserva selector opaco, huellas y cardinalidad; nunca la lista.
CREATE TABLE vec_personal.recibo_lista_empleados_b2 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^registroconsulta:[0-9a-f-]{36}$'),
 organismo_ref text NOT NULL CHECK(organismo_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 100),
 consultada_en timestamptz(6) NOT NULL
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_lista_empleados_b2
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_lista_empleados_b2
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_lista_empleados_b2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_lista_empleados_b2 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_lista_empleados_b2
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.recibo_lista_empleados_b2 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_empleados_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; consumo record;
 fecha date; conocido timestamptz(6); limite integer; cursor text; offset_p integer:=0;
 organismo text; material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 base_sha text; empleados jsonb; cardinalidad integer; hay_mas boolean; cursor_siguiente text;
 ahora timestamptz(6); recibo text;
 campos constant jsonb:='["corte","cursor","cursor_siguiente","empleados","evidencia","limite","organismo_ref"]';
BEGIN
 IF session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR length(p_material)>4096 OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL
    OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'consulta B2 denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
  limite:=(m->>'limite')::integer; cursor:=m->>'cursor';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B2 inválido' USING ERRCODE='22023'; END;
 organismo:=m->>'organismo_ref';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
      'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
      'cursor','empleado_ref','esquema','limite','operacion','organismo_ref','perfil_ref','perfil_version',
      'persona_ref','persona_version','vigente_en']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.registro-empleado-b2.consulta.v1'
    OR m->>'operacion' IS DISTINCT FROM 'empleados'
    OR m->>'empleado_ref' IS DISTINCT FROM ''
    OR fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido)
    OR conocido>transaction_timestamp()
    OR fecha::text IS DISTINCT FROM m->>'vigente_en'
    OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR organismo IS NULL OR organismo !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR limite IS NULL OR limite NOT BETWEEN 1 AND 100
    OR cursor IS NULL OR cursor !~ '^(|p_[1-9][0-9]{0,6}_[0-9a-f]{64})$'
    OR m->>'actor_ref' IS NULL OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' IS NULL OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'cuenta_ref' IS NULL OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' IS NULL OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' IS NULL OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
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
 -- json.Marshal de la estructura Go, en orden de campos y bytes UTF-8 exactos.
 material_canon:='{"esquema":'||to_jsonb(m->>'esquema')::text||',"operacion":"empleados"'||
  ',"empleado_ref":"","organismo_ref":'||to_jsonb(organismo)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"limite":'||limite::text||',"cursor":'||to_jsonb(cursor)::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(m->>'persona_ref')::text||
  ',"persona_version":'||(m->>'persona_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material B2 no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"organismo_ref":'||to_jsonb(organismo)::text||
  '},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"material_sha256":"'||material_sha||'","operacion":"empleados"'||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.registro_empleado.empleados.consultar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'empleados_rrhh'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_empleados' OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->>'recurso_ref' IS DISTINCT FROM organismo
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM 'personal.registro_empleado.empleados.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.empleados.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM organismo OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión B2 divergente' USING ERRCODE='42501'; END IF;
 base_sha:=encode(sha256(convert_to((m-'cursor')::text,'UTF8')),'hex');
 IF cursor<>'' THEN
  offset_p:=split_part(cursor,'_',2)::integer;
  IF offset_p>1000000 OR split_part(cursor,'_',3)<>base_sha THEN
   RAISE EXCEPTION 'cursor B2 divergente' USING ERRCODE='22023'; END IF;
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM organismo
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'consumo B2 divergente' USING ERRCODE='42501'; END IF;
 -- La última revisión conocida de cada hecho manda; la vigencia se filtra
 -- después. Un empleado sin relación vigente en la fecha se lista sin ellas.
 WITH rel AS (SELECT DISTINCT ON (relacion_ref) * FROM vec_personal.relacion_servicio_historia
     WHERE organismo_ref=organismo AND conocido_desde<=conocido
     ORDER BY relacion_ref,conocido_desde DESC,revision DESC),
 ocu AS (SELECT DISTINCT ON (ocupacion_ref) * FROM vec_personal.ocupacion_empleado_historia
     WHERE organismo_ref=organismo AND conocido_desde<=conocido
     ORDER BY ocupacion_ref,conocido_desde DESC,revision DESC),
 nodos AS (SELECT DISTINCT ON (nodo_ref) * FROM vec_personal.org_nodo_historia
     WHERE organismo_ref=organismo AND conocido_desde<=conocido
     ORDER BY nodo_ref,conocido_desde DESC,revision DESC),
 puestos AS (SELECT DISTINCT ON (puesto_ref) * FROM vec_personal.puesto_rpt_historia
     WHERE organismo_ref=organismo AND conocido_desde<=conocido
     ORDER BY puesto_ref,conocido_desde DESC,revision DESC),
 todos AS (SELECT DISTINCT empleado_ref FROM rel),
 pagina_e AS (SELECT empleado_ref,row_number() OVER (ORDER BY empleado_ref) AS num
     FROM (SELECT empleado_ref FROM todos ORDER BY empleado_ref OFFSET offset_p LIMIT limite+1) t),
 vigentes AS (
  SELECT r.empleado_ref,jsonb_build_object(
    'relacion_ref',r.relacion_ref,'estado',r.estado,'unidad_ref',r.unidad_ref,
    'unidad_denominacion',coalesce((SELECT n.denominacion FROM nodos n
       WHERE n.unidad_ref=r.unidad_ref AND NOT n.retirado AND n.vigente_desde<=fecha
         AND (n.vigente_hasta IS NULL OR fecha<n.vigente_hasta) ORDER BY n.nodo_ref LIMIT 1),''),
    'puesto_denominacion',coalesce((SELECT pt.denominacion FROM ocu o
       JOIN puestos pu ON pu.puesto_ref=o.puesto_ref
       JOIN vec_personal.puesto_tipo_historia pt ON pt.tipo_ref=pu.tipo_ref AND pt.revision=pu.tipo_revision
       WHERE o.relacion_ref=r.relacion_ref AND o.estado='vigente' AND o.vigente_desde<=fecha
         AND (o.vigente_hasta IS NULL OR fecha<o.vigente_hasta) ORDER BY o.ocupacion_ref LIMIT 1),''),
    'regimen_denominacion',coalesce(r.catalogo_snapshot #>> '{regimen,denominacion}',''),
    'modalidad_denominacion',coalesce(r.catalogo_snapshot #>> '{modalidad,denominacion}',''),
    'traza',vec_personal.traza_registro_empleado_b2_v1(r.vigente_desde,r.vigente_hasta,
       r.conocido_desde,r.revision,r.acto_ref,r.fuente_ref,r.fuente_version)) AS relacion,
    r.relacion_ref
  FROM rel r JOIN pagina_e p ON p.empleado_ref=r.empleado_ref AND p.num<=limite
  WHERE r.estado<>'finalizada' AND r.vigente_desde<=fecha AND (r.vigente_hasta IS NULL OR fecha<r.vigente_hasta))
 SELECT coalesce(jsonb_agg(jsonb_build_object('empleado_ref',p.empleado_ref,
    'relaciones',coalesce((SELECT jsonb_agg(v.relacion ORDER BY v.relacion_ref) FROM vigentes v
       WHERE v.empleado_ref=p.empleado_ref),'[]'::jsonb)) ORDER BY p.empleado_ref)
    FILTER (WHERE p.num<=limite),'[]'::jsonb),
   count(*) FILTER (WHERE p.num<=limite),count(*)>limite
  INTO empleados,cardinalidad,hay_mas
  FROM pagina_e p;
 IF EXISTS (SELECT 1 FROM jsonb_array_elements(empleados) e WHERE jsonb_array_length(e->'relaciones')>20) THEN
  RAISE EXCEPTION 'lista B2 excede límite' USING ERRCODE='54000'; END IF;
 IF hay_mas THEN cursor_siguiente:='p_'||(offset_p+limite)::text||'_'||base_sha;
 ELSE cursor_siguiente:=''; END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'consulta B2 caducada' USING ERRCODE='42501'; END IF;
 recibo:='registroconsulta:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_lista_empleados_b2
  (recibo_ref,organismo_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,organismo,material_sha,consumo.decision_ref,consumo.auditoria_ref,
   consumo.consumo_huella_sha256,cardinalidad,ahora);
 RETURN jsonb_build_object('pagina',jsonb_build_object('organismo_ref',organismo,
   'corte',jsonb_build_object('vigente_en',fecha::text,'conocido_en',m->>'conocido_en'),
   'limite',limite,'cursor',cursor,'cursor_siguiente',cursor_siguiente,'empleados',empleados),
   'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
   'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_empleados_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
