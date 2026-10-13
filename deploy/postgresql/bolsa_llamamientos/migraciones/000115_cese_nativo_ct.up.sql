\set ON_ERROR_STOP on
-- B115: alta y cese CT/Personal B2 nativos. B13 y B12 acreditan trabajo
-- efectivo antes de B45; cada efecto lleva auditoría V3 y outbox en su TX.
-- B90 deriva la elegibilidad; no se crea situación manual ni se altera el
-- ledger histórico integracion_desarrollo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000115',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.origen_llamamiento_ct_v1') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.auditoria_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.candidatos_cese_pendiente_b90') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)') IS NULL
    OR to_regprocedure('vec_personal.probar_cierre_cese_ct_v1(bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)') IS NULL
    OR NOT has_function_privilege(current_user,
       'vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(text)','EXECUTE')
    OR to_regprocedure('vec_contratacion_temporal.probar_incorporacion_personal_b2_para_bolsa_v1(text)') IS NULL
    OR NOT has_function_privilege(current_user,
       'vec_contratacion_temporal.probar_incorporacion_personal_b2_para_bolsa_v1(text)','EXECUTE')
    OR NOT has_function_privilege(current_user,'vec_personal.probar_cierre_cese_ct_v1(bytea)','EXECUTE')
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.cursor_contratos_participacion_v1()') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()') IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
      'vec_bolsa_llamamientos.cursor_contratos_participacion_v1()')),'UTF8')),'hex')
      IS DISTINCT FROM '30ce996fc322b2e06943ae58e8ee116139211068c327449f4c460097941315fa'
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
      'vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()')),'UTF8')),'hex')
      IS DISTINCT FROM '049dd363f185a8607cc4b3e2812d2326297d38c2c42e44509f95fb26277f9afa'
    OR to_regclass('vec_bolsa_llamamientos.cese_nativo_ct_v1') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.alta_nativa_ct_v1') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.aplicar_alta_nativa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B115: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.alta_nativa_ct_v1(
 origen_ref text PRIMARY KEY,
 evento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.contrato_participacion(evento_ref),
 recibo_situacion_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.situacion_participacion(recibo_ref),
 recibo_ct_ref text NOT NULL UNIQUE,
 participacion_ref text NOT NULL,
 bolsa_ref text NOT NULL,
 llamamiento_ref text NOT NULL,
 relacion_ref text NOT NULL,
 ocupacion_ref text NOT NULL,
 actor_ref text NOT NULL,
 material_canonico text NOT NULL CHECK(octet_length(material_canonico) BETWEEN 2 AND 16384),
 material_json jsonb NOT NULL CHECK(jsonb_typeof(material_json)='object' AND material_json=material_canonico::jsonb),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(convert_to(material_canonico,'UTF8')),'hex')),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_v3_ref text NOT NULL UNIQUE,
 trabajando_desde timestamptz(6) NOT NULL CHECK(isfinite(trabajando_desde)),
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 CHECK(evento_ref='evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
  'incorporacion'||chr(31)||origen_ref,'UTF8')),'hex')),
 CHECK(recibo_ct_ref=material_json#>>'{personal,recibo_ct_ref}'
   AND participacion_ref=material_json->>'participacion_ref'
   AND bolsa_ref=material_json#>>'{selector,bolsa_ref}'
   AND llamamiento_ref=material_json#>>'{selector,llamamiento_ref}'
   AND relacion_ref=material_json#>>'{personal,relacion_ref}'
   AND ocupacion_ref=material_json#>>'{personal,ocupacion_ref}'
   AND actor_ref=material_json#>>'{personal,actor_ref}')
);
CREATE TABLE vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1(
 evento_ref text PRIMARY KEY,
 origen_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.alta_nativa_ct_v1(origen_ref),
 tipo text NOT NULL CHECK(tipo='bolsa.alta.ct.personal_confirmado'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex')),
 estado text NOT NULL CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en)),
 CHECK(evento_ref='evento:bolsa:alta-nativa:'||encode(sha256(convert_to(origen_ref,'UTF8')),'hex'))
);

CREATE TABLE vec_bolsa_llamamientos.cese_nativo_ct_v1(
 origen_ref text PRIMARY KEY,
 evento_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.contrato_participacion(evento_ref),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.restriccion_cese_bolsa(recibo_ref),
 auditoria_bolsa_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.auditoria_cese_bolsa(auditoria_ref),
 recibo_ct_ref text NOT NULL UNIQUE,
 relacion_recibo_ref text NOT NULL,
 ocupacion_recibo_ref text NOT NULL,
 participacion_ref text NOT NULL,
 bolsa_ref text NOT NULL,
 actor_ref text NOT NULL,
 material_canonico text NOT NULL CHECK(octet_length(material_canonico) BETWEEN 2 AND 16384),
 material_json jsonb NOT NULL CHECK(jsonb_typeof(material_json)='object' AND material_json=material_canonico::jsonb),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(convert_to(material_canonico,'UTF8')),'hex')),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_v3_ref text NOT NULL UNIQUE,
 politica_version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_cese_bolsa(version),
 fecha_efecto date NOT NULL CHECK(isfinite(fecha_efecto)),
 disponible_desde date NOT NULL CHECK(isfinite(disponible_desde) AND disponible_desde>=fecha_efecto),
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en)),
 CHECK(evento_ref='evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||origen_ref,'UTF8')),'hex')),
 CHECK(recibo_ct_ref=material_json#>>'{personal,recibo_ct_ref}'
   AND relacion_recibo_ref=material_json#>>'{personal,relacion_recibo_ref}'
   AND ocupacion_recibo_ref=material_json#>>'{personal,ocupacion_recibo_ref}'
   AND participacion_ref=material_json->>'participacion_ref'
   AND bolsa_ref=material_json#>>'{selector,bolsa_ref}'
   AND actor_ref=material_json#>>'{personal,actor_ref}')
);
CREATE TABLE vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1(
 evento_ref text PRIMARY KEY,
 origen_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.cese_nativo_ct_v1(origen_ref),
 tipo text NOT NULL CHECK(tipo='bolsa.cese.ct.personal_confirmado'),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_sha256 text NOT NULL CHECK(payload_sha256=encode(sha256(convert_to(payload::text,'UTF8')),'hex')),
 estado text NOT NULL CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en)),
 CHECK(evento_ref='evento:bolsa:cese-nativo:'||encode(sha256(convert_to(origen_ref,'UTF8')),'hex'))
);
ALTER TABLE vec_bolsa_llamamientos.alta_nativa_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alta_nativa_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.alta_nativa_ct_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alta_nativa_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.alta_nativa_ct_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.alta_nativa_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

ALTER TABLE vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

ALTER TABLE vec_bolsa_llamamientos.cese_nativo_ct_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.cese_nativo_ct_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.cese_nativo_ct_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.cese_nativo_ct_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.cese_nativo_ct_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.cese_nativo_ct_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

ALTER TABLE vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1 FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- El relevo legado pagina sólo su propia historia. Una inserción nativa
-- posterior no puede adelantar el cursor y ocultarle orígenes pendientes.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.cursor_contratos_participacion_v1()
RETURNS TABLE(origen_posicion bigint,origen_ref text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT c.origen_posicion,c.origen_ref
 FROM vec_bolsa_llamamientos.contrato_participacion c
 WHERE NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cese_nativo_ct_v1 n
  WHERE n.evento_ref=c.evento_ref)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.alta_nativa_ct_v1 n
  WHERE n.evento_ref=c.evento_ref)
 ORDER BY c.origen_posicion DESC,c.origen_ref DESC LIMIT 1
$f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1()
RETURNS TABLE(origen_posicion bigint,origen_ref text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'cursor de cese no autorizado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.posicion,x.ref FROM (
  SELECT r.origen_posicion AS posicion,r.origen_ref AS ref
  FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  WHERE NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cese_nativo_ct_v1 n
   WHERE n.evento_ref=r.evento_ref)
  UNION ALL
  SELECT a.origen_posicion,a.origen_ref FROM vec_bolsa_llamamientos.cese_ajeno_bolsa a
  UNION ALL
  SELECT s.origen_posicion,s.origen_ref FROM vec_bolsa_llamamientos.cese_sin_candidato_bolsa s
 ) x ORDER BY x.posicion DESC,x.ref DESC LIMIT 1;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; sel jsonb; personal jsonb; capacidad jsonb; decision jsonb; c jsonb; anclaje jsonb;
 previo vec_bolsa_llamamientos.cese_nativo_ct_v1%ROWTYPE;
 alta vec_bolsa_llamamientos.alta_nativa_ct_v1%ROWTYPE;
 politica vec_bolsa_llamamientos.politica_cese_bolsa%ROWTYPE;
 consumo record; b13 record; restriccion record; auditoria record;
 material_sha text; contexto_sha text; v_evento_ref text; recibo_ref text; auditoria_ref text;
 outbox_ref text; clase text; meses integer; disponible date; fecha date; ahora timestamptz(6);
 auditoria_json jsonb; outbox_json jsonb; total bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'B115: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 16384
 THEN RAISE EXCEPTION 'B115: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN m:=p_material::jsonb; capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
  decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B115: material o autorización inválida' USING ERRCODE='42501'; END;
 sel:=m->'selector'; personal:=m->'personal';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_material::json))<>6
    OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
    OR NOT m ?& ARRAY['esquema','selector','participacion_ref','unidad_autorizacion_ref','ambito_ref','personal']
    OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.cese-nativo-ct.acto.v1'
    OR jsonb_typeof(sel) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each((p_material::json)->'selector'))<>10
    OR (SELECT count(*) FROM jsonb_object_keys(sel))<>10
    OR NOT sel ?& ARRAY['organizacion_ref','expediente_ref','clave_ct','unidad_ref',
      'categoria_ref','necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref']
    OR jsonb_typeof(personal) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each((p_material::json)->'personal'))<>15
    OR (SELECT count(*) FROM jsonb_object_keys(personal))<>15
    OR jsonb_typeof(m->'participacion_ref') IS DISTINCT FROM 'string'
    OR m->>'participacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
    OR jsonb_typeof(m->'unidad_autorizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'ambito_ref') IS DISTINCT FROM 'string'
    OR m->>'unidad_autorizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
    OR m->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
 THEN RAISE EXCEPTION 'B115: intención incompleta' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_json(m->>'ambito_ref')::text||
  ',"unidad_ref":'||to_json(m->>'unidad_autorizacion_ref')::text||
  '},"atributos":{"material_sha256":"'||material_sha||'"}}','UTF8')),'hex');
 IF capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
    OR capacidad->>'operacion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR capacidad->>'efecto_ref' IS DISTINCT FROM m->>'participacion_ref'
    OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
    OR capacidad->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR decision->>'accion' IS DISTINCT FROM capacidad->>'operacion'
    OR decision->>'principal_id' IS DISTINCT FROM personal->>'actor_ref'
    OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR decision->>'recurso_ref' IS DISTINCT FROM m->>'participacion_ref'
    OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B115: ACT divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM m->>'participacion_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 THEN RAISE EXCEPTION 'B115: consumo divergente' USING ERRCODE='42501'; END IF;

 -- Desde aquí sólo fuentes propietarias; ningún campo de Personal se acepta
 -- como hecho hasta que P48 y el cese CT publicado coincidan exactamente.
 SELECT vec_contratacion_temporal.probar_cese_personal_b2_para_bolsa_v1(
  personal->>'recibo_ct_ref') INTO c;
 IF c IS NULL OR c->>'recibo_ct_ref' IS DISTINCT FROM personal->>'recibo_ct_ref'
    OR c->>'organizacion_ref' IS DISTINCT FROM sel->>'organizacion_ref'
    OR c->>'expediente_ref' IS DISTINCT FROM sel->>'expediente_ref'
    OR c->>'llamamiento_ref' IS DISTINCT FROM sel->>'llamamiento_ref'
    OR c->>'actor_ref' IS DISTINCT FROM personal->>'actor_ref'
    OR c->>'version_ct' IS DISTINCT FROM personal->>'version_ct'
    OR c->>'relacion_ref' IS DISTINCT FROM personal->>'relacion_ref'
    OR c->>'ocupacion_ref' IS DISTINCT FROM personal->>'ocupacion_ref'
    OR c->>'empleado_ref' IS DISTINCT FROM personal->>'empleado_ref'
    OR c->>'relacion_recibo_ref' IS DISTINCT FROM personal->>'relacion_recibo_ref'
    OR c->>'ocupacion_recibo_ref' IS DISTINCT FROM personal->>'ocupacion_recibo_ref'
    OR c->>'fecha_efecto' IS DISTINCT FROM personal->>'vigente_hasta'
 THEN RAISE EXCEPTION 'B115: cese CT no acreditado' USING ERRCODE='23503'; END IF;
 IF jsonb_typeof(c->'evento') IS DISTINCT FROM 'object'
    OR c->>'origen_ref' IS NULL OR c->>'origen_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR c->>'origen_posicion' !~ '^(0|[1-9][0-9]{0,18})$'
    OR (c->>'origen_posicion')::numeric>9223372036854775807
    OR c->>'origen_creada_en' IS NULL
    OR c->'evento'->>'origen_ref' IS DISTINCT FROM c->>'origen_ref'
    OR c->'evento'->>'tipo' IS DISTINCT FROM 'cese'
    OR c->'evento'->>'llamamiento_ref' IS DISTINCT FROM sel->>'llamamiento_ref'
    OR c->>'origen_huella_sha256' IS DISTINCT FROM
       encode(sha256(convert_to((c->'evento')::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B115: publicación CT divergente' USING ERRCODE='23503'; END IF;
 v_evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
  'cese'||chr(31)||c->>'origen_ref','UTF8')),'hex');
 IF c->'evento'->>'evento_ref' IS DISTINCT FROM v_evento_ref
    OR c->>'modalidad_clave' IS NULL OR c->>'causa_contrato_clave' IS NULL
    OR c->>'fecha_efecto' IS NULL OR c->>'fuente_tipo' IS NULL
    OR c->>'fuente_ref' IS NULL OR c->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B115: fuente de cese incompleta' USING ERRCODE='23503'; END IF;
 BEGIN fecha:=(c->>'fecha_efecto')::date;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'B115: fecha CT inválida' USING ERRCODE='23503'; END;
 IF NOT isfinite(fecha) OR to_char(fecha,'YYYY-MM-DD') IS DISTINCT FROM c->>'fecha_efecto'
    OR c->'evento'->>'fin_previsto' IS NULL
    OR vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'inicio',true) IS NOT TRUE
    OR vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'fin_previsto',true) IS NOT TRUE
    OR vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'ocurrido_en',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'B115: evento CT inválido' USING ERRCODE='23503'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:contrato-participacion:'||v_evento_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:restriccion-cese:'||v_evento_ref,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.cese_nativo_ct_v1
  WHERE origen_ref=c->>'origen_ref';
 IF FOUND THEN
  IF previo.material_sha256 IS DISTINCT FROM material_sha OR previo.material_canonico IS DISTINCT FROM p_material
     OR previo.recibo_ct_ref IS DISTINCT FROM personal->>'recibo_ct_ref'
     OR previo.relacion_recibo_ref IS DISTINCT FROM personal->>'relacion_recibo_ref'
     OR previo.ocupacion_recibo_ref IS DISTINCT FROM personal->>'ocupacion_recibo_ref'
     OR previo.participacion_ref IS DISTINCT FROM m->>'participacion_ref'
     OR previo.bolsa_ref IS DISTINCT FROM sel->>'bolsa_ref'
     OR previo.actor_ref IS DISTINCT FROM personal->>'actor_ref'
     OR previo.evento_ref IS DISTINCT FROM v_evento_ref
  THEN RAISE EXCEPTION 'B115: replay divergente' USING ERRCODE='VBC01'; END IF;
  SELECT * INTO b13 FROM vec_bolsa_llamamientos.contrato_participacion x WHERE x.evento_ref=v_evento_ref;
  SELECT * INTO restriccion FROM vec_bolsa_llamamientos.restriccion_cese_bolsa x WHERE x.evento_ref=v_evento_ref;
  SELECT * INTO auditoria FROM vec_bolsa_llamamientos.auditoria_cese_bolsa a
   WHERE a.auditoria_ref=previo.auditoria_bolsa_ref AND a.evento_ref=v_evento_ref;
  IF b13.evento_ref IS NULL OR b13.participacion_ref IS DISTINCT FROM previo.participacion_ref
     OR restriccion.recibo_ref IS DISTINCT FROM previo.recibo_ref
     OR restriccion.politica_version IS DISTINCT FROM previo.politica_version
     OR auditoria.recibo_ref IS DISTINCT FROM previo.recibo_ref
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1 ev
       WHERE ev.origen_ref=previo.origen_ref)
  THEN RAISE EXCEPTION 'B115: replay histórico incompleto' USING ERRCODE='VBC01'; END IF;
  RETURN jsonb_build_object('reutilizada',true,'recibo_ref',previo.recibo_ref,
   'auditoria_ref',previo.auditoria_bolsa_ref,'evento_ref',previo.evento_ref,
   'participacion_ref',previo.participacion_ref,'bolsa_ref',previo.bolsa_ref,
   'fecha_efecto',to_char(previo.fecha_efecto,'YYYY-MM-DD'),
   'disponible_desde',to_char(previo.disponible_desde,'YYYY-MM-DD'),
   'politica_version',previo.politica_version);
 END IF;
 -- El replay revalida ACT y el recibo original sin depender de revisiones
 -- posteriores de Personal. La primera escritura sí exige ambos cierres.
 IF vec_personal.probar_cierre_cese_ct_v1(convert_to(
   ((p_material::json)->'personal')::text,'UTF8')) IS NOT TRUE
 THEN RAISE EXCEPTION 'B115: ambos cierres Personal pendientes' USING ERRCODE='23503'; END IF;
 anclaje:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(sel);
 IF anclaje->>'estado' NOT IN ('pendiente','acreditado')
    OR anclaje#>>'{anclaje,participacion_ref}' IS DISTINCT FROM m->>'participacion_ref'
    OR anclaje#>>'{anclaje,bolsa_ref}' IS DISTINCT FROM sel->>'bolsa_ref'
    OR anclaje#>>'{anclaje,llamamiento_ref}' IS DISTINCT FROM c->>'llamamiento_ref'
    OR anclaje#>>'{anclaje,organizacion_ref}' IS DISTINCT FROM c->>'organizacion_ref'
    OR anclaje#>>'{anclaje,expediente_ref}' IS DISTINCT FROM c->>'expediente_ref'
 THEN RAISE EXCEPTION 'B115: anclaje Bolsa divergente' USING ERRCODE='23503'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion x WHERE x.evento_ref=v_evento_ref)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa x WHERE x.evento_ref=v_evento_ref)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa x WHERE x.origen_ref=c->>'origen_ref')
 THEN RAISE EXCEPTION 'B115: cese ya tratado por otra vía' USING ERRCODE='VBC01'; END IF;
 SELECT * INTO alta FROM vec_bolsa_llamamientos.alta_nativa_ct_v1 a
  WHERE a.participacion_ref=m->>'participacion_ref'
    AND a.bolsa_ref=sel->>'bolsa_ref'
    AND a.llamamiento_ref=sel->>'llamamiento_ref'
    AND a.recibo_ct_ref=c->>'incorporacion_recibo_ref'
    AND a.relacion_ref=personal->>'relacion_ref'
    AND a.ocupacion_ref=personal->>'ocupacion_ref';
 IF NOT FOUND OR NOT EXISTS (
    SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion b
    WHERE b.evento_ref=alta.evento_ref AND b.tipo='incorporacion'
      AND b.origen_ref=alta.origen_ref AND b.participacion_ref=alta.participacion_ref
      AND b.llamamiento_ref=alta.llamamiento_ref
 ) OR NOT EXISTS (
    SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=alta.participacion_ref
      AND s.recibo_ref=alta.recibo_situacion_ref AND s.situacion='trabajando'
      AND s.desde=alta.trabajando_desde
 ) OR (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=alta.participacion_ref ORDER BY s.desde DESC LIMIT 1)
      IS DISTINCT FROM 'trabajando'
 THEN RAISE EXCEPTION 'B115: incorporación nativa previa no vigente' USING ERRCODE='23503'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('bolsa:politica-cese',0));
 SELECT * INTO politica FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR (politica.estado='ejemplo_sintetico' AND c->>'organizacion_ref' !~ '^organizacion:desarrollo:')
    OR politica.catalogo_sha256 IS DISTINCT FROM encode(sha256(convert_to(jsonb_build_object(
      'mapeo',politica.mapeo,'meses_general',politica.meses_general,
      'meses_acumulacion',politica.meses_acumulacion,'computo',politica.computo,
      'estado',politica.estado)::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B115: política de cese no aplicable' USING ERRCODE='55000'; END IF;
 clase:=coalesce(politica.mapeo->>(c->>'modalidad_clave'||'|'||c->>'causa_contrato_clave'),
  politica.mapeo->>c->>'modalidad_clave');
 IF clase NOT IN ('general','acumulacion_tareas') OR clase IS NULL
 THEN RAISE EXCEPTION 'B115: modalidad CT sin mapeo Bolsa' USING ERRCODE='55000'; END IF;
 meses:=CASE clase WHEN 'acumulacion_tareas' THEN politica.meses_acumulacion ELSE politica.meses_general END;
 disponible:=(fecha+make_interval(months=>meses))::date;
 recibo_ref:='recibo:bolsa:cese:'||encode(sha256(convert_to(v_evento_ref,'UTF8')),'hex');
 auditoria_ref:='auditoria:bolsa:cese:'||encode(sha256(convert_to(v_evento_ref,'UTF8')),'hex');
 outbox_ref:='evento:bolsa:cese-nativo:'||encode(sha256(convert_to(c->>'origen_ref','UTF8')),'hex');
 ahora:=date_trunc('microseconds',clock_timestamp());

 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
  evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,tipo,
  organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,
  inicio,fin_previsto,modalidad_clave,categoria_ref,causa_clave,ocurrido_en,recibido_en)
 VALUES(v_evento_ref,c->>'origen_huella_sha256',c->'evento',c->>'origen_ref',
  (c->>'origen_creada_en')::timestamptz,(c->>'origen_posicion')::bigint,'cese',
  c->>'organizacion_ref',c->>'expediente_ref',c->>'llamamiento_ref',
  m->>'participacion_ref',sel->>'bolsa_ref',
  (c->'evento'->>'inicio')::timestamptz,(c->'evento'->>'fin_previsto')::timestamptz,
  c->'evento'->>'modalidad_clave',c->'evento'->>'categoria_ref',
  c->'evento'->>'causa_clave',(c->'evento'->>'ocurrido_en')::timestamptz,ahora);
 INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa(
  evento_ref,origen_ref,origen_huella_sha256,origen_posicion,candidato_ref,
  llamamiento_ref,relacion_ref,expediente_ref,organizacion_ref,modalidad_clave,
  causa_contrato_clave,clase_bolsa,fecha_efecto,disponible_desde,politica_version,
  fuente_tipo,fuente_ref,fuente_sha256,recibo_ct_ref,recibo_ref,recibida_en)
 VALUES(v_evento_ref,c->>'origen_ref',c->>'origen_huella_sha256',
  (c->>'origen_posicion')::bigint,anclaje#>>'{anclaje,candidato_ref}',
  c->>'llamamiento_ref',c->>'relacion_ref',c->>'expediente_ref',c->>'organizacion_ref',
  c->>'modalidad_clave',c->>'causa_contrato_clave',clase,fecha,disponible,
  politica.version,c->>'fuente_tipo',c->>'fuente_ref',c->>'fuente_sha256',
  personal->>'recibo_ct_ref',recibo_ref,ahora);
 auditoria_json:=jsonb_build_object('esquema','vec.bolsa.cese.auditoria.v1',
  'evento_ref',v_evento_ref,'accion','restriccion_cese_aplicada','resultado','aplicada',
  'actor',session_user,'correlacion_ref',c->>'origen_ref','politica_version',politica.version,
  'recibo_ref',recibo_ref,'registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_bolsa_llamamientos.auditoria_cese_bolsa(
  auditoria_ref,evento_ref,accion,resultado,actor,correlacion_ref,politica_version,
  recibo_ref,registro,registro_sha256,registrada_en)
 VALUES(auditoria_ref,v_evento_ref,'restriccion_cese_aplicada','aplicada',session_user,
  c->>'origen_ref',politica.version,recibo_ref,auditoria_json,
  encode(sha256(convert_to(auditoria_json::text,'UTF8')),'hex'),ahora);
 INSERT INTO vec_bolsa_llamamientos.cese_nativo_ct_v1(
  origen_ref,evento_ref,recibo_ref,auditoria_bolsa_ref,recibo_ct_ref,
  relacion_recibo_ref,ocupacion_recibo_ref,participacion_ref,bolsa_ref,actor_ref,
  material_canonico,material_json,material_sha256,decision_ref,consumo_huella_sha256,
  auditoria_v3_ref,politica_version,fecha_efecto,disponible_desde,registrada_en)
 VALUES(c->>'origen_ref',v_evento_ref,recibo_ref,auditoria_ref,
  personal->>'recibo_ct_ref',personal->>'relacion_recibo_ref',personal->>'ocupacion_recibo_ref',
  m->>'participacion_ref',sel->>'bolsa_ref',personal->>'actor_ref',
  p_material,m,material_sha,consumo.decision_ref,consumo.consumo_huella_sha256,
  consumo.auditoria_ref,politica.version,fecha,disponible,ahora);
 outbox_json:=jsonb_build_object('esquema','vec.bolsa.cese-nativo-ct.confirmado.v1',
  'origen_ref',c->>'origen_ref','evento_ref',v_evento_ref,'recibo_ref',recibo_ref,
  'participacion_ref',m->>'participacion_ref','politica_version',politica.version,
  'fecha_efecto',to_char(fecha,'YYYY-MM-DD'),
  'disponible_desde',to_char(disponible,'YYYY-MM-DD'));
 INSERT INTO vec_bolsa_llamamientos.cese_nativo_ct_outbox_v1(
  evento_ref,origen_ref,tipo,payload,payload_sha256,estado,creada_en)
 VALUES(outbox_ref,c->>'origen_ref','bolsa.cese.ct.personal_confirmado',outbox_json,
  encode(sha256(convert_to(outbox_json::text,'UTF8')),'hex'),'pendiente',ahora);
 RETURN jsonb_build_object('reutilizada',false,'recibo_ref',recibo_ref,
  'auditoria_ref',auditoria_ref,'evento_ref',v_evento_ref,
  'participacion_ref',m->>'participacion_ref','bolsa_ref',sel->>'bolsa_ref',
  'fecha_efecto',to_char(fecha,'YYYY-MM-DD'),
  'disponible_desde',to_char(disponible,'YYYY-MM-DD'),
  'politica_version',politica.version);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'B115: conflicto concurrente' USING ERRCODE='VBC01';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.aplicar_alta_nativa_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; sel jsonb; personal jsonb; capacidad jsonb; decision jsonb;
 c jsonb; anclaje jsonb; previo vec_bolsa_llamamientos.alta_nativa_ct_v1%ROWTYPE;
 consumo record; situacion record; contrato record;
 material_sha text; contexto_sha text; evento_ref text; recibo_situacion text;
 outbox_ref text; outbox_json jsonb; ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'B115: sesión de alta denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 16384
 THEN RAISE EXCEPTION 'B115: material de alta inválido' USING ERRCODE='22023'; END IF;
 BEGIN m:=p_material::jsonb; capacidad:=convert_from(p_capacidad,'UTF8')::jsonb;
  decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B115: autorización de alta inválida' USING ERRCODE='42501'; END;
 sel:=m->'selector'; personal:=m->'personal';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_material::json))<>6
    OR (SELECT count(*) FROM jsonb_object_keys(m))<>6
    OR NOT m ?& ARRAY['esquema','selector','participacion_ref','unidad_autorizacion_ref','ambito_ref','personal']
    OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.alta-nativa-ct.acto.v1'
    OR jsonb_typeof(sel) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each((p_material::json)->'selector'))<>10
    OR (SELECT count(*) FROM jsonb_object_keys(sel))<>10
    OR NOT sel ?& ARRAY['organizacion_ref','expediente_ref','clave_ct','unidad_ref',
      'categoria_ref','necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref']
    OR jsonb_typeof(personal) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each((p_material::json)->'personal'))<>12
    OR (SELECT count(*) FROM jsonb_object_keys(personal))<>12
    OR NOT personal ?& ARRAY['recibo_ct_ref','version_expediente','actor_ref','empleado_ref',
      'relacion_ref','relacion_version','relacion_recibo_ref','ocupacion_ref',
      'ocupacion_version','ocupacion_recibo_ref','plan_ref','plan_sha256']
    OR coalesce(m->>'participacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
    OR coalesce(m->>'unidad_autorizacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
    OR coalesce(m->>'ambito_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$'
    OR coalesce(personal->>'plan_sha256','') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B115: intención de alta incompleta' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_json(m->>'ambito_ref')::text||
  ',"unidad_ref":'||to_json(m->>'unidad_autorizacion_ref')::text||
  '},"atributos":{"material_sha256":"'||material_sha||'"}}','UTF8')),'hex');
 IF capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
    OR capacidad->>'operacion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
    OR capacidad->>'efecto_ref' IS DISTINCT FROM m->>'participacion_ref'
    OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
    OR capacidad->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR decision->>'accion' IS DISTINCT FROM capacidad->>'operacion'
    OR decision->>'principal_id' IS DISTINCT FROM personal->>'actor_ref'
    OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion'
    OR decision->>'recurso_ref' IS DISTINCT FROM m->>'participacion_ref'
    OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B115: ACT de alta divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM m->>'participacion_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 THEN RAISE EXCEPTION 'B115: consumo de alta divergente' USING ERRCODE='42501'; END IF;

 -- La recuperación usa la prueba persistida y el ACT nuevo. No consulta
 -- revisiones Personal actuales ni reabre trabajo después de un cese.
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:alta-nativa:'||personal->>'recibo_ct_ref',0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.alta_nativa_ct_v1 a
  WHERE a.recibo_ct_ref=personal->>'recibo_ct_ref';
 IF FOUND THEN
  IF previo.material_canonico IS DISTINCT FROM p_material
     OR previo.material_sha256 IS DISTINCT FROM material_sha
     OR previo.participacion_ref IS DISTINCT FROM m->>'participacion_ref'
     OR previo.bolsa_ref IS DISTINCT FROM sel->>'bolsa_ref'
     OR previo.llamamiento_ref IS DISTINCT FROM sel->>'llamamiento_ref'
     OR previo.actor_ref IS DISTINCT FROM personal->>'actor_ref'
     OR previo.relacion_ref IS DISTINCT FROM personal->>'relacion_ref'
     OR previo.ocupacion_ref IS DISTINCT FROM personal->>'ocupacion_ref'
  THEN RAISE EXCEPTION 'B115: replay de alta divergente' USING ERRCODE='VBC01'; END IF;
  IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion b
       WHERE b.evento_ref=previo.evento_ref AND b.tipo='incorporacion'
         AND b.participacion_ref=previo.participacion_ref AND b.origen_ref=previo.origen_ref)
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s
       WHERE s.recibo_ref=previo.recibo_situacion_ref AND s.participacion_ref=previo.participacion_ref
         AND s.situacion='trabajando' AND s.desde=previo.trabajando_desde)
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1 ev
       WHERE ev.origen_ref=previo.origen_ref)
  THEN RAISE EXCEPTION 'B115: historia de alta incompleta' USING ERRCODE='VBC01'; END IF;
  RETURN jsonb_build_object('reutilizada',true,'evento_ref',previo.evento_ref,
   'recibo_situacion_ref',previo.recibo_situacion_ref,'auditoria_ref',previo.auditoria_v3_ref,
   'outbox_ref','evento:bolsa:alta-nativa:'||encode(sha256(convert_to(previo.origen_ref,'UTF8')),'hex'),
   'bolsa_ref',previo.bolsa_ref,'participacion_ref',previo.participacion_ref,
   'trabajando_desde',to_char(previo.trabajando_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 END IF;

 SELECT vec_contratacion_temporal.probar_incorporacion_personal_b2_para_bolsa_v1(
  personal->>'recibo_ct_ref') INTO c;
 IF c IS NULL OR c->>'recibo_ct_ref' IS DISTINCT FROM personal->>'recibo_ct_ref'
    OR c->>'organizacion_ref' IS DISTINCT FROM sel->>'organizacion_ref'
    OR c->>'expediente_ref' IS DISTINCT FROM sel->>'expediente_ref'
    OR c->>'llamamiento_ref' IS DISTINCT FROM sel->>'llamamiento_ref'
    OR c->'selector' IS DISTINCT FROM sel
    OR c->>'version_expediente' IS DISTINCT FROM personal->>'version_expediente'
    OR c->>'actor_ref' IS DISTINCT FROM personal->>'actor_ref'
    OR c->>'empleado_ref' IS DISTINCT FROM personal->>'empleado_ref'
    OR c->>'relacion_ref' IS DISTINCT FROM personal->>'relacion_ref'
    OR c->>'relacion_version' IS DISTINCT FROM personal->>'relacion_version'
    OR c->>'relacion_recibo_ref' IS DISTINCT FROM personal->>'relacion_recibo_ref'
    OR c->>'ocupacion_ref' IS DISTINCT FROM personal->>'ocupacion_ref'
    OR c->>'ocupacion_version' IS DISTINCT FROM personal->>'ocupacion_version'
    OR c->>'ocupacion_recibo_ref' IS DISTINCT FROM personal->>'ocupacion_recibo_ref'
    OR c->>'plan_ref' IS DISTINCT FROM personal->>'plan_ref'
    OR c->>'plan_sha256' IS DISTINCT FROM personal->>'plan_sha256'
 THEN RAISE EXCEPTION 'B115: CT/P49 no acredita alta' USING ERRCODE='23503'; END IF;
 anclaje:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(sel);
 IF anclaje->>'estado' IS DISTINCT FROM 'acreditado'
    OR anclaje#>>'{anclaje,participacion_ref}' IS DISTINCT FROM m->>'participacion_ref'
    OR anclaje#>>'{anclaje,bolsa_ref}' IS DISTINCT FROM sel->>'bolsa_ref'
    OR anclaje#>>'{anclaje,llamamiento_ref}' IS DISTINCT FROM sel->>'llamamiento_ref'
    OR anclaje#>>'{anclaje,organizacion_ref}' IS DISTINCT FROM sel->>'organizacion_ref'
    OR anclaje#>>'{anclaje,expediente_ref}' IS DISTINCT FROM sel->>'expediente_ref'
 THEN RAISE EXCEPTION 'B115: anclaje de alta divergente' USING ERRCODE='23503'; END IF;
 IF jsonb_typeof(c->'evento') IS DISTINCT FROM 'object'
    OR coalesce(c->>'origen_huella_sha256','') !~ '^[0-9a-f]{64}$'
    OR coalesce(c->>'origen_posicion','') !~ '^(0|[1-9][0-9]{0,18})$'
    OR (c->>'origen_posicion')::numeric>9223372036854775807
    OR c->'evento'->>'tipo' IS DISTINCT FROM 'incorporacion'
    OR c->'evento'->>'origen_ref' IS DISTINCT FROM c->>'origen_ref'
    OR c->>'origen_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to((c->'evento')::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B115: evento de alta divergente' USING ERRCODE='23503'; END IF;
 evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
  'incorporacion'||chr(31)||c->>'origen_ref','UTF8')),'hex');
 IF c->'evento'->>'evento_ref' IS DISTINCT FROM evento_ref
    OR vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'inicio',true) IS NOT TRUE
    OR (c->'evento'->>'inicio')::timestamptz>clock_timestamp()
    OR (c->'evento'->>'fin_previsto' IS NOT NULL AND
      vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'fin_previsto',true) IS NOT TRUE)
    OR vec_bolsa_llamamientos.instante_contrato_valido(c->'evento'->'ocurrido_en',false) IS NOT TRUE
 THEN RAISE EXCEPTION 'B115: fechas de alta divergentes' USING ERRCODE='23503'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:contrato-participacion:'||evento_ref,0));
 SELECT * INTO situacion FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref=m->>'participacion_ref'
  ORDER BY s.desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND OR situacion.situacion IS DISTINCT FROM 'disponible'
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion b
      WHERE b.evento_ref=evento_ref OR
        (b.participacion_ref=m->>'participacion_ref' AND b.llamamiento_ref=sel->>'llamamiento_ref'
         AND b.tipo='incorporacion'))
 THEN RAISE EXCEPTION 'B115: participación no disponible para alta' USING ERRCODE='VBC01'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF ahora<=situacion.desde
 THEN RAISE EXCEPTION 'B115: instante de alta anterior a situación' USING ERRCODE='VBC01'; END IF;
 recibo_situacion:='recibo:situacion:alta-nativa:'||encode(sha256(convert_to(c->>'origen_ref','UTF8')),'hex');
 outbox_ref:='evento:bolsa:alta-nativa:'||encode(sha256(convert_to(c->>'origen_ref','UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
  evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,tipo,
  organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,
  inicio,fin_previsto,modalidad_clave,categoria_ref,causa_clave,ocurrido_en,recibido_en)
 VALUES(evento_ref,c->>'origen_huella_sha256',c->'evento',c->>'origen_ref',
  (c->>'origen_creada_en')::timestamptz,(c->>'origen_posicion')::bigint,
  'incorporacion',c->>'organizacion_ref',c->>'expediente_ref',c->>'llamamiento_ref',
  m->>'participacion_ref',sel->>'bolsa_ref',
  (c->'evento'->>'inicio')::timestamptz,(c->'evento'->>'fin_previsto')::timestamptz,
  c->'evento'->>'modalidad_clave',c->'evento'->>'categoria_ref',
  c->'evento'->>'causa_clave',(c->'evento'->>'ocurrido_en')::timestamptz,ahora);
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
  participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,
  registrada_en,clave_idempotencia,recibo_ref)
 VALUES(m->>'participacion_ref','trabajando',ahora,NULL,NULL,
  'incorporacion_personal_b2_confirmada',personal->>'actor_ref',ahora,
  'alta-nativa:'||encode(sha256(convert_to(c->>'origen_ref','UTF8')),'hex'),recibo_situacion);
 INSERT INTO vec_bolsa_llamamientos.alta_nativa_ct_v1(
  origen_ref,evento_ref,recibo_situacion_ref,recibo_ct_ref,participacion_ref,
  bolsa_ref,llamamiento_ref,relacion_ref,ocupacion_ref,actor_ref,
  material_canonico,material_json,material_sha256,decision_ref,
  consumo_huella_sha256,auditoria_v3_ref,trabajando_desde,registrada_en)
 VALUES(c->>'origen_ref',evento_ref,recibo_situacion,personal->>'recibo_ct_ref',
  m->>'participacion_ref',sel->>'bolsa_ref',sel->>'llamamiento_ref',
  personal->>'relacion_ref',personal->>'ocupacion_ref',personal->>'actor_ref',
  p_material,m,material_sha,consumo.decision_ref,consumo.consumo_huella_sha256,
  consumo.auditoria_ref,ahora,ahora);
 outbox_json:=jsonb_build_object('esquema','vec.bolsa.alta-nativa-ct.confirmada.v1',
  'origen_ref',c->>'origen_ref','evento_ref',evento_ref,
  'recibo_situacion_ref',recibo_situacion,'participacion_ref',m->>'participacion_ref',
  'bolsa_ref',sel->>'bolsa_ref','trabajando_desde',
  to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_bolsa_llamamientos.alta_nativa_ct_outbox_v1(
  evento_ref,origen_ref,tipo,payload,payload_sha256,estado,creada_en)
 VALUES(outbox_ref,c->>'origen_ref','bolsa.alta.ct.personal_confirmado',outbox_json,
  encode(sha256(convert_to(outbox_json::text,'UTF8')),'hex'),'pendiente',ahora);
 RETURN jsonb_build_object('reutilizada',false,'evento_ref',evento_ref,
  'recibo_situacion_ref',recibo_situacion,'auditoria_ref',consumo.auditoria_ref,
  'outbox_ref',outbox_ref,'bolsa_ref',sel->>'bolsa_ref',
  'participacion_ref',m->>'participacion_ref',
  'trabajando_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'B115: conflicto concurrente de alta' USING ERRCODE='VBC01';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.aplicar_alta_nativa_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.aplicar_alta_nativa_ct_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
