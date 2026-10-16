\set ON_ERROR_STOP on
-- B133 / F1-09: acto de hechos, sin cambiar situación ni reservar un llamamiento.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000133',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'B133: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
 comprobacion_ref text PRIMARY KEY CHECK(comprobacion_ref ~ '^comprobacion:[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL,expediente_ref text NOT NULL,necesidad_ref text NOT NULL,
 bolsa_ref text NOT NULL,participacion_ref text NOT NULL,
 version bigint NOT NULL CHECK(version>0),version_esperada bigint NOT NULL CHECK(version=version_esperada+1),
 contexto jsonb NOT NULL,politica jsonb NOT NULL,hechos jsonb NOT NULL,resultado jsonb NOT NULL,
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,perfil_ref text NOT NULL,clave_idempotencia text NOT NULL,
 recibo_ref text NOT NULL UNIQUE,evento_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,registrada_en timestamptz(6) NOT NULL,
 UNIQUE(organizacion_ref,expediente_ref,necesidad_ref,participacion_ref,version),
 UNIQUE(organizacion_ref,expediente_ref,necesidad_ref,participacion_ref,clave_idempotencia),
 CHECK(jsonb_typeof(contexto)='object' AND jsonb_typeof(hechos)='object'),
 CHECK(resultado->>'resultado' IN ('pendiente','no_aplicable'))
);
ALTER TABLE vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY comprobacion_propietario ON vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 FROM PUBLIC;
CREATE TRIGGER comprobacion_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER comprobacion_no_truncar BEFORE TRUNCATE
 ON vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Cada valor es una aportación documental de RRHH. Quién/cuándo se verificó
-- lo fija la escritura desde la decisión, nunca el cuerpo enviado por el cliente.
CREATE FUNCTION vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(
 h jsonb,c jsonb,p_corte timestamptz) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE k text;v jsonb;valores jsonb:='{}';condicion jsonb;limite jsonb;
 inicio date;fin date;baja date;borde date;encaja boolean:=false;incompleto boolean:=false;
 motivo text:='antecedentes_incompletos';
BEGIN
 IF jsonb_typeof(h) IS DISTINCT FROM 'object' OR octet_length(h::text)>32768
 OR p_corte IS NULL OR NOT isfinite(p_corte)
 THEN RAISE EXCEPTION 'B133: hechos inválidos' USING ERRCODE='22023'; END IF;
 FOR k,v IN SELECT key,value FROM jsonb_each(h) LOOP
  IF k NOT IN ('trabaja_en_diputacion','tipo_relacion','modalidad','inicio','fin_previsto',
   'fin_inicial_programa','fecha_inicial_baja','es_vacante','prelacion','antecedentes_completos')
  OR jsonb_typeof(v) IS DISTINCT FROM 'object'
  OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(v) x)
     IS DISTINCT FROM ARRAY['documento_ref','huella','valor']
  OR jsonb_typeof(v->'documento_ref') IS DISTINCT FROM 'string'
  OR octet_length(v->>'documento_ref') NOT BETWEEN 3 AND 256
  OR v->>'documento_ref' !~ '^ref:[0-9a-f]{64}$'
  OR jsonb_typeof(v->'huella') IS DISTINCT FROM 'string' OR v->>'huella' !~ '^[0-9a-f]{64}$'
  OR v->'valor'='null'::jsonb THEN
   RAISE EXCEPTION 'B133: evidencia inválida' USING ERRCODE='22023'; END IF;
  IF (k IN ('trabaja_en_diputacion','es_vacante','antecedentes_completos') AND jsonb_typeof(v->'valor')<>'boolean')
  OR (k='tipo_relacion' AND (jsonb_typeof(v->'valor')<>'string' OR v->>'valor' NOT IN ('acumulacion','sustitucion','programa','otra')))
  OR (k='modalidad' AND (jsonb_typeof(v->'valor')<>'string' OR v->>'valor' !~ '^[a-z][a-z0-9._-]{1,79}$'))
  OR (k='prelacion' AND (jsonb_typeof(v->'valor')<>'number' OR v->>'valor' !~ '^[1-9][0-9]{0,8}$'))
  OR (k IN ('inicio','fin_previsto','fin_inicial_programa','fecha_inicial_baja')
   AND (jsonb_typeof(v->'valor')<>'string' OR v->>'valor' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'))
  THEN RAISE EXCEPTION 'B133: valor inválido' USING ERRCODE='22023'; END IF;
  valores:=valores||jsonb_build_object(k,v->'valor');
 END LOOP;
 -- Fechas de calendario, sin inferir una baja del nombre de una modalidad.
 inicio:=(valores->>'inicio')::date;fin:=(valores->>'fin_inicial_programa')::date;
 baja:=(valores->>'fecha_inicial_baja')::date;
 IF (inicio IS NOT NULL AND (NOT isfinite(inicio) OR inicio>p_corte::date))
 OR (fin IS NOT NULL AND (inicio IS NULL OR fin<=inicio))
 OR (baja IS NOT NULL AND (inicio IS NULL OR baja>inicio))
 OR (valores ? 'fin_previsto' AND (inicio IS NULL OR (valores->>'fin_previsto')::date<inicio))
 THEN RAISE EXCEPTION 'B133: fechas incoherentes' USING ERRCODE='22023'; END IF;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(c) IS NOT TRUE OR c->'mejora' IS NULL
 THEN motivo:='criterios_invalidos';
 ELSIF valores ?& ARRAY['trabaja_en_diputacion','tipo_relacion','modalidad','inicio','fin_previsto','es_vacante','prelacion','antecedentes_completos']
  AND valores->>'antecedentes_completos'='true' THEN
  IF valores->>'trabaja_en_diputacion'='false' OR valores->>'es_vacante'='false'
   OR c#>>'{mejora,habilitada}'<>'true' THEN
   RETURN jsonb_build_object('resultado','no_aplicable','motivo_clave','sin_supuesto_articulo_9'); END IF;
  FOR condicion IN SELECT value FROM jsonb_array_elements(c#>'{mejora,condiciones_actuales}') LOOP
   IF condicion->>'modalidad' IS DISTINCT FROM valores->>'modalidad' THEN CONTINUE; END IF;
   IF (valores->>'tipo_relacion'='acumulacion' AND valores->>'modalidad'<>'acumulacion_tareas')
    OR (valores->>'tipo_relacion' IN ('sustitucion','programa') AND valores->>'tipo_relacion'<>valores->>'modalidad')
    OR valores->>'tipo_relacion'='otra' THEN CONTINUE; END IF;
   IF condicion->>'requiere_baja_comprobada'='true' AND baja IS NULL
   THEN incompleto:=true;CONTINUE; END IF;
   limite:=condicion->'meses_transcurridos';
   IF limite IS NOT NULL THEN
    borde:=((CASE WHEN condicion->>'requiere_baja_comprobada'='true' THEN baja ELSE inicio END)
     +make_interval(months=>(limite->>'meses')::int))::date;
    IF p_corte::date>borde OR (p_corte::date=borde AND limite->>'incluye_limite'='false') THEN CONTINUE; END IF;
   END IF;
   limite:=condicion->'meses_duracion_inicial';
   IF limite IS NOT NULL THEN
    IF fin IS NULL THEN incompleto:=true;CONTINUE; END IF;
    borde:=(inicio+make_interval(months=>(limite->>'meses')::int))::date;
    IF fin>borde OR (fin=borde AND limite->>'incluye_limite'='false') THEN CONTINUE; END IF;
   END IF;
   encaja:=true;EXIT;
  END LOOP;
  IF NOT encaja AND NOT incompleto THEN
   RETURN jsonb_build_object('resultado','no_aplicable','motivo_clave','sin_supuesto_articulo_9'); END IF;
  IF encaja THEN motivo:='guarda_contratacion_pendiente'; END IF;
  IF encaja AND (c#>'{mejora,supuestos_reduccion_vacante}') ? (valores->>'modalidad')
  THEN motivo:='descuento_pendiente_criterio_rrhh'; END IF;
 END IF;
 RETURN jsonb_build_object('resultado','pendiente','motivo_clave',motivo);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(
 p jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE d jsonb;c jsonb;consumo record;previa record;politica record;entrada record;
 version_actual bigint;huella text;referencia text;recibo text;evento text;
 instante timestamptz(6);resultado jsonb;hechos jsonb;conjunto jsonb;es_replay boolean;origen record;
BEGIN
 IF session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'B133: registro denegado' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>49152
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p) k)
    IS DISTINCT FROM ARRAY['clave_idempotencia','contexto','hechos','version_esperada']
 OR jsonb_typeof(p->'contexto') IS DISTINCT FROM 'object'
 OR jsonb_typeof(p->'version_esperada') IS DISTINCT FROM 'number'
 OR p->>'version_esperada' !~ '^(0|[1-9][0-9]{0,8})$'
 OR jsonb_typeof(p->'clave_idempotencia') IS DISTINCT FROM 'string'
 OR octet_length(p->>'clave_idempotencia') NOT BETWEEN 8 AND 256
 OR p->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]+$'
 THEN RAISE EXCEPTION 'B133: solicitud inválida' USING ERRCODE='22023'; END IF;
 c:=p->'contexto';
 IF (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(c) k) IS DISTINCT FROM
 ARRAY['ambito_ref','bolsa_ref','categoria_ref','causa_ofrecida','expediente_ref',
  'huella_politica_ofertas_sha256','modalidad_ofrecida','necesidad_ref','orden_acta',
  'organizacion_ref','participacion_ref','unidad_ref','version_bolsa','version_expediente',
  'version_instantanea','version_politica_ofertas']
 OR EXISTS(SELECT 1 FROM jsonb_each(c) x WHERE
  CASE WHEN x.key IN ('orden_acta','version_bolsa','version_expediente','version_instantanea','version_politica_ofertas')
   THEN jsonb_typeof(x.value)<>'number' OR x.value#>>'{}' !~ '^[1-9][0-9]{0,8}$'
   ELSE jsonb_typeof(x.value)<>'string' OR octet_length(x.value#>>'{}')>256
    OR (x.key<>'causa_ofrecida' AND x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]+$') END)
 OR c->>'huella_politica_ofertas_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B133: contexto inválido' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF consumo.efecto_ref IS DISTINCT FROM c->>'participacion_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella
 OR d#>>'{ambitos,unidad_ref}' IS DISTINCT FROM c->>'unidad_ref'
 OR d#>>'{ambitos,ambito_ref}' IS DISTINCT FROM c->>'ambito_ref'
 THEN RAISE EXCEPTION 'B133: decisión ajena al acto' USING ERRCODE='42501'; END IF;
 conjunto:=vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
 IF c->>'ambito_ref' IS DISTINCT FROM conjunto->>'ambito_ref' OR conjunto->>'ambito_ref' IS NULL
 THEN RAISE EXCEPTION 'B133: ámbito ajeno' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:comprobacion:'||
  jsonb_build_array(c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',c->>'participacion_ref')::text,0));
 SELECT * INTO previa FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
 WHERE x.organizacion_ref=c->>'organizacion_ref' AND x.expediente_ref=c->>'expediente_ref'
  AND x.necesidad_ref=c->>'necesidad_ref' AND x.participacion_ref=c->>'participacion_ref'
  AND x.clave_idempotencia=p->>'clave_idempotencia';
 es_replay:=FOUND;
 IF vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(c,NOT es_replay) IS NOT TRUE
 THEN RAISE EXCEPTION 'B133: contexto CT divergente' USING ERRCODE='VBC01'; END IF;
 IF es_replay THEN
  IF previa.material_sha256 IS DISTINCT FROM huella
  OR previa.actor_ref IS DISTINCT FROM d->>'principal_id'
  OR previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
  THEN RAISE EXCEPTION 'B133: clave reutilizada' USING ERRCODE='VBC01'; END IF;
 ELSE
  IF consumo.consumo_nuevo IS NOT TRUE THEN
   RAISE EXCEPTION 'B133: falta efecto del consumo' USING ERRCODE='42501'; END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||(c->>'bolsa_ref'),0));
  SELECT * INTO STRICT politica FROM vec_bolsa_llamamientos.politica_ofertas_version x
   WHERE x.bolsa_ref=c->>'bolsa_ref' ORDER BY x.version DESC LIMIT 1;
  SELECT * INTO STRICT origen FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
   c->>'categoria_ref',c->>'modalidad_ofrecida',c->>'causa_ofrecida');
  SELECT * INTO entrada FROM jsonb_to_recordset(origen.entradas)
   x(participacion_ref text,orden_acta bigint,orden_vigente bigint,razon text)
   WHERE x.orden_vigente=1 OR x.razon='pendiente_comprobacion_rrhh' ORDER BY x.orden_acta LIMIT 1;
  IF NOT FOUND OR origen.bolsa_ref IS DISTINCT FROM c->>'bolsa_ref'
  OR origen.version_bolsa IS DISTINCT FROM (c->>'version_bolsa')::bigint
  OR origen.version_instantanea IS DISTINCT FROM (c->>'version_instantanea')::bigint
  OR entrada.participacion_ref IS DISTINCT FROM c->>'participacion_ref'
  OR entrada.razon IS DISTINCT FROM 'pendiente_comprobacion_rrhh'
  OR entrada.orden_acta IS DISTINCT FROM (c->>'orden_acta')::bigint
  OR politica.version IS DISTINCT FROM (c->>'version_politica_ofertas')::bigint
  OR politica.huella_sha256 IS DISTINCT FROM c->>'huella_politica_ofertas_sha256'
  THEN RAISE EXCEPTION 'B133: origen o política divergentes' USING ERRCODE='VBC01'; END IF;
  SELECT coalesce(max(x.version),0) INTO version_actual FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 x
   WHERE x.organizacion_ref=c->>'organizacion_ref' AND x.expediente_ref=c->>'expediente_ref'
   AND x.necesidad_ref=c->>'necesidad_ref' AND x.participacion_ref=c->>'participacion_ref';
  IF version_actual<>(p->>'version_esperada')::bigint THEN
   RAISE EXCEPTION 'B133: versión obsoleta' USING ERRCODE='VBC01'; END IF;
  instante:=date_trunc('microseconds',clock_timestamp());
  resultado:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(
   p->'hechos',politica.politica->'criterios_ofrecimiento',instante);
  IF p#>>'{hechos,prelacion,valor}' IS NOT NULL
   AND p#>>'{hechos,prelacion,valor}' IS DISTINCT FROM c->>'orden_acta'
  THEN RAISE EXCEPTION 'B133: prelación divergente' USING ERRCODE='VBC01'; END IF;
  SELECT coalesce(jsonb_object_agg(key,value||jsonb_build_object('verificada_por',d->>'principal_id',
   'verificada_en',instante)),'{}') INTO hechos FROM jsonb_each(p->'hechos');
  referencia:='comprobacion:'||encode(sha256(convert_to(
   jsonb_build_array(c,p->>'clave_idempotencia',version_actual+1)::text,'UTF8')),'hex');
  recibo:='recibo:'||referencia;evento:='evento:'||referencia;
  INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 VALUES(
   referencia,c->>'organizacion_ref',c->>'expediente_ref',c->>'necesidad_ref',c->>'bolsa_ref',c->>'participacion_ref',
   version_actual+1,version_actual,c,politica.politica,hechos,resultado,huella,d->>'principal_id',d->>'perfil_activo_ref',
   p->>'clave_idempotencia',recibo,evento,consumo.decision_ref,consumo.auditoria_ref,instante)
  RETURNING * INTO previa;
 END IF;
 -- Recibo mínimo: nunca publica hechos, evidencias ni datos de la relación.
 RETURN jsonb_build_object('comprobacion_ref',previa.comprobacion_ref,'version',previa.version,
  'version_esperada',previa.version_esperada,'resultado',previa.resultado->>'resultado',
  'motivo_clave',previa.resultado->>'motivo_clave','version_politica_ofertas',previa.contexto->'version_politica_ofertas',
  'recibo_ref',previa.recibo_ref,'evento_ref',previa.evento_ref,'auditoria_ref',previa.auditoria_ref,
  'registrada_en',previa.registrada_en,'reutilizada',es_replay);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
