\set ON_ERROR_STOP on
-- B141 / F1-57: RRHH declara el documento que custodia con una referencia
-- legible. La evidencia de un hecho admite la referencia opaca de B77 (sin
-- etiquetas ni números de documento de identidad) o el evento de contrato que
-- el propio sistema recibe de Contratación temporal. El navegador calcula la
-- huella; el documento no sale del equipo ni VEC lo custodia.
-- Añade además la lectura del histórico de contratos con su huella, para que
-- la ficha cite el contrato como evidencia. La lectura v1 no cambia.
-- Secuencia: después de B133 y B137; independiente de B140.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000141',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_bolsa_llamamientos_propietario'
 OR to_regprocedure('vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(text)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(text)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '26baa7aadeed04704fe5c2ec780cb56def83464f47aa9ae5ca4640562c4fa1c1'
 OR to_regprocedure('vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '27b65b67bcb026980948f4d9829fe887beadfa18cbcf664bd77eb5028d8ab2c9'
 OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=
   'vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)'::regprocedure)
   IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR to_regprocedure('vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '69beb556f7daf04edfe7fb53799bd3ea106a42039e5afbda69e4e4ba856da08f'
 OR to_regprocedure('vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR (SELECT count(*) FROM pg_attribute a WHERE a.attrelid='vec_bolsa_llamamientos.contrato_participacion'::regclass
   AND a.attname='huella_sha256' AND NOT a.attisdropped AND a.attnotnull)<>1
 THEN RAISE EXCEPTION 'B141: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Mismo cuerpo instalado por B133; solo cambia la forma admitida de documento_ref.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h jsonb, c jsonb, p_corte timestamp with time zone)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE k text;v jsonb;valores jsonb:='{}';condicion jsonb;limite jsonb;
 inicio date;fin date;baja date;borde date;encaja boolean:=false;incompleto boolean:=false;
 motivo text:='antecedentes_incompletos';
BEGIN
 IF jsonb_typeof(h) IS DISTINCT FROM 'object' OR octet_length(h::text)>32768
 OR p_corte IS NULL OR NOT isfinite(p_corte)
 THEN RAISE EXCEPTION 'B133: hechos inválidos' USING ERRCODE='22023'; END IF;
 FOR k,v IN SELECT key,value FROM jsonb_each(h) LOOP
  IF k NOT IN ('trabaja_en_diputacion','tipo_relacion','modalidad','causa_relacion','inicio','fin_previsto',
   'fin_inicial_programa','fecha_inicial_baja','es_vacante','prelacion','antecedentes_completos')
  OR jsonb_typeof(v) IS DISTINCT FROM 'object'
  OR (SELECT array_agg(x ORDER BY x) FROM jsonb_object_keys(v) x)
     IS DISTINCT FROM ARRAY['documento_ref','huella','valor']
  OR jsonb_typeof(v->'documento_ref') IS DISTINCT FROM 'string'
  OR octet_length(v->>'documento_ref') NOT BETWEEN 3 AND 256
  -- Referencia legible sin datos de identidad (B77) o evento de contrato del sistema.
  OR (vec_bolsa_llamamientos.b77_documento_ref_opaco_v1(v->>'documento_ref') IS NOT TRUE
   AND v->>'documento_ref' !~ '^evento:ct:contrato-bolsa:[0-9a-f]{64}$')
  OR jsonb_typeof(v->'huella') IS DISTINCT FROM 'string' OR v->>'huella' !~ '^[0-9a-f]{64}$'
  OR v->'valor'='null'::jsonb THEN
   RAISE EXCEPTION 'B133: evidencia inválida' USING ERRCODE='22023'; END IF;
  IF (k IN ('trabaja_en_diputacion','es_vacante','antecedentes_completos') AND jsonb_typeof(v->'valor')<>'boolean')
  OR (k='tipo_relacion' AND (jsonb_typeof(v->'valor')<>'string' OR v->>'valor' NOT IN ('acumulacion','sustitucion','programa','otra')))
  OR (k IN ('modalidad','causa_relacion') AND (jsonb_typeof(v->'valor')<>'string' OR v->>'valor' !~ '^[a-z][a-z0-9._-]{1,79}$'))
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
   OR c#>>'{mejora,habilitada}' IS DISTINCT FROM 'true' THEN
   RETURN jsonb_build_object('resultado','no_aplicable','motivo_clave','sin_supuesto_articulo_9'); END IF;
  FOR condicion IN SELECT value FROM jsonb_array_elements(c#>'{mejora,condiciones_actuales}') LOOP
   IF condicion->>'modalidad' IS DISTINCT FROM valores->>'modalidad' THEN CONTINUE; END IF;
   -- Es la causa administrativa del contrato, nunca la causa de una baja.
   IF condicion ? 'causa' THEN
    IF NOT valores ? 'causa_relacion' THEN incompleto:=true;CONTINUE; END IF;
    IF condicion->>'causa' IS DISTINCT FROM valores->>'causa_relacion' THEN CONTINUE; END IF;
   END IF;
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
END $function$;

-- Igual que la v1 de B24 (mismo consumo V3, mismas guardas, mismo orden y
-- límite), con la huella del evento conservado para citarlo como evidencia.
CREATE FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v2(
 p_participacion_ref text,p_actor text,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(evento_ref text, tipo text, inicio timestamptz, fin_previsto timestamptz, modalidad_clave text,
 categoria_ref text, causa_clave text, expediente_ref text, llamamiento_ref text, ocurrido_en timestamptz, huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_consumo record; v_decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL OR p_actor IS NULL THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada';
 END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada'; END;
 IF v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR v_consumo.consumo_nuevo IS NOT TRUE
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de contratos no autorizada';
 END IF;
 RETURN QUERY SELECT c.evento_ref, c.tipo, c.inicio, c.fin_previsto, c.modalidad_clave, c.categoria_ref, c.causa_clave,
   c.expediente_ref, c.llamamiento_ref, c.ocurrido_en, c.huella_sha256
   FROM vec_bolsa_llamamientos.contrato_participacion c
  WHERE c.participacion_ref = p_participacion_ref
  ORDER BY coalesce(c.inicio, c.ocurrido_en) DESC, c.evento_ref DESC
  LIMIT 200;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;

DO $post$ DECLARE x record;p pg_proc%ROWTYPE;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)',NULL::text,false),
  ('vec_bolsa_llamamientos.listar_contratos_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_bolsa_llamamientos_ejecutor',true)
 ) v(firma,ejecutor,definidora) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(x.firma);
  IF p.proowner IS DISTINCT FROM to_regrole('vec_bolsa_llamamientos_propietario') OR p.prosecdef IS DISTINCT FROM x.definidora
  OR p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'] IS NOT TRUE
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee<>p.proowner AND (a.grantee IS DISTINCT FROM to_regrole(x.ejecutor)
      OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR (x.ejecutor IS NOT NULL AND NOT has_function_privilege(x.ejecutor,p.oid,'EXECUTE'))
  THEN RAISE EXCEPTION 'B141: función sin cierre de permisos %',x.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF (SELECT q.proacl::text FROM pg_proc q WHERE q.oid=
   'vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(jsonb,jsonb,timestamptz)'::regprocedure)
   IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(
   'vec_bolsa_llamamientos.listar_contratos_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '69beb556f7daf04edfe7fb53799bd3ea106a42039e5afbda69e4e4ba856da08f'
 OR vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(
   jsonb_build_object('es_vacante',jsonb_build_object('valor',true,'documento_ref','registro/2026/0042','huella',repeat('a',64))),
   '{}'::jsonb,'2026-10-19T00:00:00Z')->>'resultado' IS DISTINCT FROM 'pendiente'
 THEN RAISE EXCEPTION 'B141: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
