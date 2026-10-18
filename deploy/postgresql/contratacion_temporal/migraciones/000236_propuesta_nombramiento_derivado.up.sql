\set ON_ERROR_STOP on
-- CT236: propuesta ordinaria y propuesta del hijo CT234 en las mismas dos
-- funciones CT211. Las referencias de resolución antigua solo viven en la
-- rama ordinaria; una aceptación derivada conserva su recibo y origen reales.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000236',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.nombramiento_derivado_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.propuesta_formalizacion') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(jsonb,jsonb)') IS NOT NULL
    OR EXISTS(SELECT 1 FROM pg_attribute a
       WHERE a.attrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
         AND a.attname='derivacion_ref' AND NOT a.attisdropped)
    OR (SELECT count(*) FROM pg_attribute a
       WHERE a.attrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
         AND a.attname IN ('llamamiento_ref','resolucion_ref','recibo_aceptacion_ref')
         AND a.attnotnull AND NOT a.attisdropped)<>3
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c
       WHERE c.conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
         AND c.conname='propuesta_formalizacion_check3' AND c.convalidated
         AND strpos(pg_get_constraintdef(c.oid),'AceptacionBolsaNativa')>0)
 THEN RAISE EXCEPTION 'CT236: preimagen o dependencias incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
 ADD COLUMN derivacion_ref text UNIQUE REFERENCES vec_contratacion_temporal.nombramiento_derivado_v1(derivacion_ref),
 ALTER COLUMN llamamiento_ref DROP NOT NULL,
 ALTER COLUMN resolucion_ref DROP NOT NULL,
 ALTER COLUMN recibo_aceptacion_ref DROP NOT NULL,
 DROP CONSTRAINT propuesta_formalizacion_check3;
ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
 ADD CONSTRAINT propuesta_formalizacion_check3 CHECK(
  jsonb_typeof(aceptacion_bolsa_json)='object' AND (
   ((material_json ? 'AceptacionBolsa') AND NOT (material_json ?| ARRAY['AceptacionBolsaNativa','AceptacionDerivada'])
     AND aceptacion_bolsa_json=material_json->'AceptacionBolsa')
   OR ((material_json ? 'AceptacionBolsaNativa') AND NOT (material_json ?| ARRAY['AceptacionBolsa','AceptacionDerivada'])
     AND aceptacion_bolsa_json=material_json->'AceptacionBolsaNativa')
   OR ((material_json ? 'AceptacionDerivada') AND NOT (material_json ?| ARRAY['AceptacionBolsa','AceptacionBolsaNativa'])
     AND aceptacion_bolsa_json=material_json->'AceptacionDerivada'))),
 ADD CONSTRAINT propuesta_origen_derivado_ct236 CHECK(
  (derivacion_ref IS NULL AND llamamiento_ref IS NOT NULL AND resolucion_ref IS NOT NULL
    AND recibo_aceptacion_ref IS NOT NULL AND NOT (material_json ? 'AceptacionDerivada'))
  OR (derivacion_ref IS NOT NULL AND llamamiento_ref IS NULL AND resolucion_ref IS NULL
    AND recibo_aceptacion_ref IS NULL AND material_json ? 'AceptacionDerivada'
    AND solicitud_json->>'DerivacionRef' IS NOT DISTINCT FROM derivacion_ref));

-- Solo la función de propuesta, después del consumo V3, puede usar este
-- antecedente. CT234, su aceptación y sus recibos son la fuente histórica.
CREATE FUNCTION vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(
 p_solicitud jsonb,p_presentada jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE d vec_contratacion_temporal.nombramiento_derivado_v1%ROWTYPE; evidencia jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT236: antecedente denegado' USING ERRCODE='P0613'; END IF;
 IF jsonb_typeof(p_solicitud) IS DISTINCT FROM 'object'
    OR p_solicitud->>'DerivacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_solicitud->'VersionEsperada' IS DISTINCT FROM '6'::jsonb
    OR p_solicitud->>'LlamamientoRef' IS DISTINCT FROM ''
    OR p_solicitud->>'ResolucionLlamamientoAceptadaRef' IS DISTINCT FROM ''
    OR p_solicitud->>'ReciboResolucionAceptadaRef' IS DISTINCT FROM ''
 THEN RAISE EXCEPTION 'CT236: selector derivado inválido' USING ERRCODE='P0610'; END IF;
 SELECT * INTO d FROM vec_contratacion_temporal.nombramiento_derivado_v1
 WHERE derivacion_ref=p_solicitud->>'DerivacionRef'
   AND organizacion_ref=p_solicitud->>'OrganizacionRef'
   AND hijo_ref=p_solicitud->>'ExpedienteRef' FOR SHARE;
 IF NOT FOUND OR d.material_sha256 IS DISTINCT FROM encode(sha256(convert_to(d.material,'UTF8')),'hex')
    OR d.material::jsonb->>'organizacion_ref' IS DISTINCT FROM d.organizacion_ref
    OR d.material::jsonb->>'padre_ref' IS DISTINCT FROM d.padre_ref
    OR d.material::jsonb->>'hijo_ref' IS DISTINCT FROM d.hijo_ref
    OR d.material::jsonb->>'oferta_ref' IS DISTINCT FROM d.oferta_ref
    OR d.material::jsonb->>'numero_plaza' IS DISTINCT FROM d.numero_plaza::text
    OR d.recibo_json->>'estado' IS DISTINCT FROM 'confirmado'
    OR d.recibo_json->>'padre_ref' IS DISTINCT FROM d.padre_ref
    OR d.recibo_json->>'hijo_ref' IS DISTINCT FROM d.hijo_ref
    OR d.recibo_json->>'aceptacion_ref' IS DISTINCT FROM d.aceptacion_ref
    OR d.recibo_json->>'recibo_ref' IS DISTINCT FROM d.recibo_ref
    OR d.recibo_json->>'numero_plaza' IS DISTINCT FROM d.numero_plaza::text
    OR d.evidencia_bolsa->>'persona_ref' IS DISTINCT FROM d.persona_ref
    OR d.evidencia_bolsa->>'version_padre' IS DISTINCT FROM d.version_padre::text
    OR d.evidencia_bolsa->>'numero_de_plaza' IS DISTINCT FROM d.numero_plaza::text
    OR d.evidencia_bolsa->>'recibo_adjudicacion_ref' IS DISTINCT FROM d.material::jsonb->>'recibo_adjudicacion_ref'
    OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_alta alta
       JOIN vec_contratacion_temporal.expediente_version_integral v6
         ON v6.expediente_ref=alta.expediente_ref AND v6.version=6
       JOIN vec_contratacion_temporal.actuacion_expediente_integral act
         ON act.expediente_ref=v6.expediente_ref AND act.secuencia=6
       WHERE alta.expediente_ref=d.hijo_ref AND alta.derivacion_ref=d.derivacion_ref
         AND v6.origen_version='derivacion_nombramiento_ct234'
         AND v6.agregado_json=d.recibo_json->'expediente'
         AND v6.agregado_json_huella_sha256=encode(sha256(convert_to(v6.agregado_json::text,'UTF8')),'hex')
         AND act.recibo_ref=d.recibo_ref
         AND act.actuacion_json->>'accion_clave'='contratacion_temporal.nombramiento.derivar')
    OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral padre
       WHERE padre.expediente_ref=d.padre_ref AND padre.version=d.version_padre
         AND padre.agregado_json=d.recibo_json#>'{expediente,instruccion_compartida,expediente_padre}'
         AND padre.agregado_json_huella_sha256=encode(sha256(convert_to(padre.agregado_json::text,'UTF8')),'hex'))
    OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral evento
       WHERE evento.operacion_ref=d.derivacion_ref AND evento.expediente_ref=d.hijo_ref
         AND evento.version_expediente=6 AND evento.tipo_evento='contratacion_temporal.nombramiento_derivado'
         AND evento.payload_huella_sha256=encode(sha256(evento.payload_canonico),'hex')
         AND (convert_from(evento.payload_canonico,'UTF8')::jsonb)->>'recibo_ref'=d.recibo_ref)
 THEN RAISE EXCEPTION 'CT236: derivación no acreditada' USING ERRCODE='P0615'; END IF;
 evidencia:=jsonb_build_object(
  'DerivacionRef',d.derivacion_ref,'PadreRef',d.padre_ref,'HijoRef',d.hijo_ref,
  'OfertaRef',d.oferta_ref,'NumeroPlaza',d.numero_plaza,
  'AceptacionRef',d.aceptacion_ref,'ReciboDerivacionRef',d.recibo_ref);
 IF p_presentada IS NOT NULL AND p_presentada IS DISTINCT FROM evidencia THEN
  RAISE EXCEPTION 'CT236: evidencia derivada divergente' USING ERRCODE='P0615'; END IF;
 RETURN evidencia;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(jsonb,jsonb) FROM PUBLIC;

-- Solo la propuesta CT211 pasa el hijo acreditado de v6 a v7. Los demás
-- actos conservan la denegación del circuito hasta su adaptación nominal.
CREATE FUNCTION vec_contratacion_temporal.propuesta_derivada_ct236_v1(
 p_expediente text,p_anterior jsonb,p_nuevo jsonb,p_version numeric)
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT coalesce(
  p_version=7 AND p_anterior->>'version'='6' AND p_nuevo->>'version'='7'
  AND p_anterior->>'fase_actual'='fiscalizacion'
  AND p_nuevo->>'fase_actual'='nombramiento'
  AND p_anterior->>'estado_actual'='en_curso'
  AND p_nuevo->>'estado_actual'='en_curso'
  AND (p_nuevo-ARRAY['version','fase_actual','estado_actual','actualizado_en','actuaciones'])
      =(p_anterior-ARRAY['version','fase_actual','estado_actual','actualizado_en','actuaciones'])
  AND jsonb_typeof(p_anterior->'actuaciones')='array'
  AND jsonb_typeof(p_nuevo->'actuaciones')='array'
  AND jsonb_array_length(p_anterior->'actuaciones')=1
  AND jsonb_array_length(p_nuevo->'actuaciones')=2
  AND p_nuevo->'actuaciones'->0=p_anterior->'actuaciones'->0
  AND p_anterior->'actuaciones'->0->>'accion_clave'='contratacion_temporal.nombramiento.derivar'
  AND p_anterior->'actuaciones'->0->>'secuencia'='6'
  AND p_anterior->'actuaciones'->0->>'version_expediente'='6'
  AND p_nuevo->'actuaciones'->1->>'accion_clave'='registrar_propuesta_formalizacion'
  AND p_nuevo->'actuaciones'->1->>'secuencia'='7'
  AND p_nuevo->'actuaciones'->1->>'version_expediente'='7'
  AND p_nuevo->'actuaciones'->1->>'fase_origen'='fiscalizacion'
  AND p_nuevo->'actuaciones'->1->>'fase_destino'='nombramiento'
  AND p_nuevo->'actuaciones'->1->>'estado_origen'='en_curso'
  AND p_nuevo->'actuaciones'->1->>'estado_destino'='en_curso'
  AND p_nuevo->'actuaciones'->1->>'unidad_ref'=p_anterior#>>'{asignacion,unidad_ref}'
  AND (p_nuevo->'actuaciones'->1->>'realizada_en')::timestamptz
      =(p_nuevo->>'actualizado_en')::timestamptz
  AND (p_nuevo->>'actualizado_en')::timestamptz
      >=(p_anterior->>'actualizado_en')::timestamptz
  AND p_nuevo->>'referencia'=p_expediente
  AND p_anterior->>'referencia'=p_expediente
  AND p_nuevo->'circuito' IS NOT DISTINCT FROM p_anterior->'circuito'
  AND p_nuevo->'instruccion_compartida'=p_anterior->'instruccion_compartida'
  AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral inicial
     JOIN vec_contratacion_temporal.actuacion_expediente_integral acto6
       ON acto6.expediente_ref=inicial.expediente_ref AND acto6.secuencia=6
     WHERE inicial.expediente_ref=p_expediente AND inicial.version=6
       AND inicial.origen_version='derivacion_nombramiento_ct234'
       AND inicial.agregado_json=p_anterior
       AND acto6.recibo_ref=p_anterior#>>'{instruccion_compartida,recibo_derivacion_ref}'
       AND acto6.actuacion_json=p_anterior->'actuaciones'->0
       AND vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
         p_expediente,inicial.agregado_json)),false)
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.propuesta_derivada_ct236_v1(text,jsonb,jsonb,numeric) FROM PUBLIC;

-- CT236-GENERADO-INICIO
-- Fuente: definiciones instaladas CT211; huellas previas y finales cotejadas.
CREATE TEMP TABLE pg_temp.ct236_pre(
 firma text PRIMARY KEY,oid oid NOT NULL,propietario oid NOT NULL,acl aclitem[],
 posthash text NOT NULL,config_final text[] NOT NULL
) ON COMMIT DROP;
DO $pre_funciones$
DECLARE x record;p pg_proc%ROWTYPE;f regprocedure;
BEGIN
 FOR x IN SELECT * FROM (VALUES
 ('vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','ea35ab5bba36c8bcce5c7e1f26af9089f47fe926dc677e68238f83328da13712','1a0665d48d931983b55f9720d5432c3ce66418ad3d7a543a8bb0786e0cc0c701','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]),
 ('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','b2414d394ce0e0389a69a9b0f9f7cb792aa02e32bea7884486adebce1e394138','bcb2ec7320e5aecbd0e1a13736542e9c3815b3c318c16276499d83efba49babf','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]),
 ('vec_contratacion_temporal.proteger_version_circuito_ct163()','08bfd3363ae587062d72083aed4fa81821e272a1b81d73f62aa754710b8a5973','6f5572ec3a51512801b289e42bfeab124e6fa32bdf97c1a6553b5cbe915b56fb','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['row_security=on','search_path=pg_catalog, pg_temp']::text[]),
 ('vec_contratacion_temporal.proteger_version_circuito_ct164()','e3ac1b9724108eaf92b2e55131bb1c8ca980718932dd9823ced2ab7b939e4f0c','24fc0173622809f6380207d9c789af0c781e320e88542141b579283bc31476e5','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['row_security=on','search_path=pg_catalog, pg_temp']::text[])
 ) AS fuente(firma,prehash,posthash,acl_original,config_original) LOOP
  f:=to_regprocedure(x.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'CT236: función CT211 ausente' USING ERRCODE='55000'; END IF;
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
     OR p.proacl::text IS DISTINCT FROM x.acl_original
     OR cardinality(p.proconfig) IS DISTINCT FROM cardinality(x.config_original)
     OR EXISTS(SELECT 1 FROM unnest(x.config_original) cfg WHERE cfg<>ALL(p.proconfig))
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM x.prehash
  THEN RAISE EXCEPTION 'CT236: preimagen CT211 divergente: %',f USING ERRCODE='55000'; END IF;
  INSERT INTO pg_temp.ct236_pre VALUES(x.firma,f,p.proowner,p.proacl,x.posthash,x.config_original);
 END LOOP;
END $pre_funciones$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    v_n bigint; v_siguiente bigint;
    m jsonb; s jsonb; b jsonb; d jsonb; v_campo text; v_snapshot jsonb;
    v_hash text; v_contexto_hash text; v_consumo record;
    v_derivada boolean; v_derivacion jsonb;
    v_resolucion vec_contratacion_temporal.resolucion_manual_respuesta_rrhh%ROWTYPE;
    v_previa vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
    v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
    v_respuesta jsonb; v_seleccion jsonb; v_resultado jsonb; v_continuacion jsonb;
    v_ahora timestamptz(6); v_fecha_bolsa timestamptz(6);
    v_propuesta text; v_recibo text; v_evento text; v_unidad text;
    v_actuacion jsonb; v_agregado jsonb; v_agregado_hash text; v_prueba bytea;
    v_secuencia numeric(20,0); v_anterior text; v_payload bytea; v_outbox_hash text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'propuesta denegada' USING ERRCODE='P0613';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de propuesta inválido' USING ERRCODE='P0610';
    END IF;
    BEGIN m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de propuesta inválido' USING ERRCODE='P0610';
    END;
    IF m->>'Etapa' IS DISTINCT FROM 'consulta'
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['Etapa','Solicitud']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',(CASE WHEN (m->'Solicitud') ? 'DerivacionRef' THEN ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef','VersionEsperada',
           'TipoFormalizacion','Plantilla','Anexos','PoliticaFirma','PlanFirma','DerivacionRef'] ELSE ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef','VersionEsperada',
           'TipoFormalizacion','Plantilla','Anexos','PoliticaFirma','PlanFirma'] END)) IS NOT TRUE
       OR (SELECT count(*) FROM json_each(p_material::json))<>2
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>(CASE WHEN (m->'Solicitud') ? 'DerivacionRef' THEN 13 ELSE 12 END) THEN
        RAISE EXCEPTION 'CT211: consulta inválida' USING ERRCODE='P0610';
    END IF;
    s:=m->'Solicitud'; v_derivada:=s ? 'DerivacionRef';
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s->'VersionEsperada') IS DISTINCT FROM 'number'
       OR coalesce(s->>'VersionEsperada','') !~ '^[1-9][0-9]{0,15}$'
       OR (s->'Anexos') NOT IN ('null'::jsonb,'[]'::jsonb) THEN
        RAISE EXCEPTION 'solicitud de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_n:=(s->>'VersionEsperada')::bigint;
    IF v_n NOT BETWEEN 6 AND 9007199254740990 OR (v_derivada AND v_n<>6) THEN
        RAISE EXCEPTION 'versión de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_siguiente:=v_n+1;
    FOREACH v_campo IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef'] LOOP
        IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
           OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de propuesta inválida' USING ERRCODE='P0610'; END IF;
    END LOOP;
    IF v_derivada THEN
        IF jsonb_typeof(s->'DerivacionRef') IS DISTINCT FROM 'string'
           OR s->>'DerivacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR EXISTS(SELECT 1 FROM unnest(ARRAY['LlamamientoRef',
                'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef']) campo
                WHERE jsonb_typeof(s->campo) IS DISTINCT FROM 'string' OR s->>campo IS DISTINCT FROM '')
        THEN RAISE EXCEPTION 'referencias derivadas inválidas' USING ERRCODE='P0610'; END IF;
    ELSE
        FOREACH v_campo IN ARRAY ARRAY['LlamamientoRef',
            'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef'] LOOP
            IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
               OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION 'referencia de propuesta inválida' USING ERRCODE='P0610'; END IF;
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        v_snapshot:=s->v_campo;
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_snapshot,ARRAY['Referencia','Version','HuellaSHA256']) IS NOT TRUE THEN
            RAISE EXCEPTION 'publicación de propuesta inválida' USING ERRCODE='P0610';
        END IF;
        IF (SELECT count(*) FROM json_each(((p_material::json)->'Solicitud')->v_campo))<>3
           OR jsonb_typeof(v_snapshot->'Referencia') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'Referencia')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_snapshot->'Version' IS DISTINCT FROM '1'::jsonb
           OR jsonb_typeof(v_snapshot->'HuellaSHA256') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'HuellaSHA256')!~'^[0-9a-f]{64}$'
           OR v_snapshot->>'HuellaSHA256'=repeat('0',64) THEN
            RAISE EXCEPTION 'compromiso de publicación inválido' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    v_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_hash:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_hash||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.formalizacion.propuesta.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'propuesta_formalizacion_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'permiso de propuesta divergente' USING ERRCODE='P0613';
    END IF;
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_propuesta_formalizacion_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0613' THEN
        RAISE EXCEPTION 'consumo de propuesta denegado' USING ERRCODE='P0613';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'propuesta requiere consumo nuevo ligado' USING ERRCODE='P0613';
    END IF;

    -- Primera lectura de negocio: permiso fresco ya consumido en esta tx.
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo p
            WHERE p.componente=v_campo AND p.referencia=s->v_campo->>'Referencia'
              AND to_jsonb(p.version)=s->v_campo->'Version' AND p.huella_sha256=s->v_campo->>'HuellaSHA256'
              AND p.huella_sha256=encode(sha256(convert_to(p.contenido,'UTF8')),'hex')) THEN
            RAISE EXCEPTION 'publicación de propuesta no acreditada' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    IF v_derivada THEN
        v_derivacion:=vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(s,NULL);
        RETURN jsonb_build_object('Derivacion',v_derivacion);
    END IF;
    SELECT * INTO v_resolucion FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
     WHERE r.resolucion_ref=s->>'ResolucionLlamamientoAceptadaRef' AND r.recibo_ref=s->>'ReciboResolucionAceptadaRef'
       AND r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef' AND r.estado='confirmado'
       AND r.solicitud_json->>'Respuesta'='aceptacion' AND r.revision_respuesta_rrhh AND r.revision_plazo_rrhh
       AND r.politica_ref='politica:ct:revision-manual-sintetica:20260906' AND r.politica_version=1
       AND r.politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'
       AND r.solicitud_json->>'CriterioValidacionRef'=r.politica_ref
       AND r.solicitud_json->'RevisionRespuestaRRHH'='true'::jsonb
       AND r.solicitud_json->'RevisionPlazoRRHH'='true'::jsonb
       AND r.estado_plazo='vigente' AND r.version_resultante=3
       AND r.recibo_json->>'Estado'='confirmado' AND r.recibo_json->'Solicitud'=r.solicitud_json
       AND r.recibo_json->>'ResolucionRef'=r.resolucion_ref AND r.recibo_json->>'ReciboLocalRef'=r.recibo_ref
       AND r.recibo_json->>'EvaluacionPlazoRef'=r.evaluacion_plazo_ref
       AND r.recibo_json->'Politica'=jsonb_build_object('Referencia',r.politica_ref,'Version',r.politica_version,'HuellaSHA256',r.politica_sha256)
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'aceptación CT no acreditada' USING ERRCODE='P0615';
    END IF;
    SELECT j.recibo_json,e.recibo_json,continuacion.continuacion_recibo
      INTO v_respuesta,v_seleccion,v_continuacion
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh j
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c ON c.comunicacion_ref=j.comunicacion_ref
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=j.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
      -- CT65: confirmación original del primer sucesor, ligada al aviso exacto.
      LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh continuacion
        ON c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
       AND continuacion.seleccion_clave=e.clave_idempotencia
       AND continuacion.organizacion_ref=j.organizacion_ref
       AND continuacion.expediente_ref=j.expediente_ref
       AND continuacion.llamamiento_ref=e.recibo_json->>'llamamiento_ref'
       AND continuacion.llamamiento_ref<>j.llamamiento_ref
       AND continuacion.estado='confirmado'
       AND continuacion.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')
       AND continuacion.continuacion_clave IS NOT NULL
       AND continuacion.continuacion_recibo->>'Estado'='confirmado'
       AND continuacion.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND continuacion.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=j.organizacion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=j.expediente_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ResolucionRef'=continuacion.resolucion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion.continuacion_clave::text
       AND continuacion.continuacion_recibo->>'LlamamientoAnteriorRef'=continuacion.llamamiento_ref
       AND continuacion.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=j.llamamiento_ref
     WHERE j.justificante_ref=v_resolucion.justificante_ref AND j.seleccion_clave=v_resolucion.seleccion_clave
       AND j.organizacion_ref=v_resolucion.organizacion_ref AND j.expediente_ref=v_resolucion.expediente_ref
       AND j.llamamiento_ref=v_resolucion.llamamiento_ref AND j.comunicacion_ref=v_resolucion.comunicacion_ref
       AND j.respuesta='aceptacion' AND j.version_comunicacion=2 AND j.estado='registrada_por_rrhh'
       AND c.organizacion_ref=j.organizacion_ref AND c.expediente_ref=j.expediente_ref AND c.llamamiento_ref=j.llamamiento_ref
       AND c.estado='registrada_localmente' AND c.version_resultante=2
       AND ((e.solicitud_json->'version_expediente'=to_jsonb(v_n)
       AND e.recibo_json->'version_expediente'=to_jsonb(v_n))
        -- CT128: tras una no incorporación la selección es la original y el
        -- expediente sigue en una versión posterior a esa no incorporación.
        OR (continuacion.solicitud_json->>'Respuesta'='aceptacion'
            AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
            AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n128
                         WHERE n128.aceptacion_resolucion_ref=continuacion.resolucion_ref
                           AND n128.intencion_ref=continuacion.continuacion_recibo->'Solicitud'->>'IntencionRef'
                           AND n128.organizacion_ref=j.organizacion_ref AND n128.expediente_ref=j.expediente_ref
                           AND n128.version_esperada+1<=v_n)))
       AND e.situacion='confirmada' AND e.solicitud_json->>'organizacion_ref'=j.organizacion_ref
       AND e.solicitud_json->>'expediente_ref'=j.expediente_ref
       AND e.recibo_json->>'organizacion_ref'=j.organizacion_ref AND e.recibo_json->>'expediente_ref'=j.expediente_ref
       AND (
           (NOT ((c.material_json->'solicitud') ? 'TipoAntecedente')
            AND e.recibo_json->>'llamamiento_ref'=j.llamamiento_ref
            AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef')
           OR (c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
               AND continuacion.resolucion_ref IS NOT NULL)
       )
       AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND j.recibo_json->>'JustificanteRef'=j.justificante_ref AND j.recibo_json->>'ReciboRef'=j.recibo_ref
       AND j.recibo_json->'Solicitud'=j.material_json AND j.recibo_json->>'Estado'='registrada_por_rrhh'
       AND j.registrada_en<=v_resolucion.resuelta_en
     FOR SHARE OF j,c,e;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'antecedentes de aceptación incompatibles' USING ERRCODE='P0615';
    END IF;
    IF m->>'Etapa'='consulta' THEN
        RETURN jsonb_build_object('Resolucion',v_resolucion.recibo_json,
            'Justificante',jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion)||
                CASE WHEN v_continuacion IS NULL THEN '{}'::jsonb
                     ELSE jsonb_build_object('Continuacion',v_continuacion) END,
            'SeleccionClave',v_resolucion.seleccion_clave::text);
    END IF;
    RAISE EXCEPTION 'CT211: consulta sin resultado' USING ERRCODE='P0615';
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    v_n bigint; v_siguiente bigint;
    m jsonb; s jsonb; b jsonb; d jsonb; v_campo text; v_snapshot jsonb;
    v_hash text; v_contexto_hash text; v_consumo record;
    v_derivada boolean; v_derivacion jsonb;
    v_resolucion vec_contratacion_temporal.resolucion_manual_respuesta_rrhh%ROWTYPE;
    v_previa vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
    v_actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
    v_respuesta jsonb; v_seleccion jsonb; v_resultado jsonb; v_continuacion jsonb; v_bolsa_nativa jsonb;
    v_ahora timestamptz(6); v_fecha_bolsa timestamptz(6);
    v_propuesta text; v_recibo text; v_evento text; v_unidad text;
    v_actuacion jsonb; v_agregado jsonb; v_agregado_hash text; v_prueba bytea;
    v_secuencia numeric(20,0); v_anterior text; v_payload bytea; v_outbox_hash text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'propuesta denegada' USING ERRCODE='P0613';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'material de propuesta inválido' USING ERRCODE='P0610';
    END IF;
    BEGIN m:=p_material::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'JSON de propuesta inválido' USING ERRCODE='P0610';
    END;
    IF m->>'Etapa' IS DISTINCT FROM 'confirmacion'
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,
          (CASE WHEN (m->'Solicitud') ? 'DerivacionRef' THEN ARRAY['Etapa','Solicitud','AceptacionDerivada'] ELSE ARRAY['Etapa','Solicitud','AceptacionBolsaNativa'] END)) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m->'Solicitud',(CASE WHEN (m->'Solicitud') ? 'DerivacionRef' THEN ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef','VersionEsperada',
           'TipoFormalizacion','Plantilla','Anexos','PoliticaFirma','PlanFirma','DerivacionRef'] ELSE ARRAY[
           'ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef',
           'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef','VersionEsperada',
           'TipoFormalizacion','Plantilla','Anexos','PoliticaFirma','PlanFirma'] END)) IS NOT TRUE
       OR (SELECT count(*) FROM json_each(p_material::json))<>3
       OR (SELECT count(*) FROM json_each((p_material::json)->'Solicitud'))<>(CASE WHEN (m->'Solicitud') ? 'DerivacionRef' THEN 13 ELSE 12 END) THEN
        RAISE EXCEPTION 'CT211: material nativo inválido' USING ERRCODE='P0610';
    END IF;
    s:=m->'Solicitud'; v_derivada:=s ? 'DerivacionRef';
    b:=CASE WHEN v_derivada THEN m->'AceptacionDerivada' ELSE m->'AceptacionBolsaNativa' END;
    IF jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string'
       OR (s->>'ClaveIdempotencia')!~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
       OR s->>'ClaveIdempotencia'='00000000-0000-4000-8000-000000000000'
       OR jsonb_typeof(s->'VersionEsperada') IS DISTINCT FROM 'number'
       OR coalesce(s->>'VersionEsperada','') !~ '^[1-9][0-9]{0,15}$'
       OR (s->'Anexos') NOT IN ('null'::jsonb,'[]'::jsonb) THEN
        RAISE EXCEPTION 'solicitud de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_n:=(s->>'VersionEsperada')::bigint;
    IF v_n NOT BETWEEN 6 AND 9007199254740990 OR (v_derivada AND v_n<>6) THEN
        RAISE EXCEPTION 'versión de propuesta inválida' USING ERRCODE='P0610';
    END IF;
    v_siguiente:=v_n+1;
    FOREACH v_campo IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef'] LOOP
        IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
           OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de propuesta inválida' USING ERRCODE='P0610'; END IF;
    END LOOP;
    IF v_derivada THEN
        IF jsonb_typeof(s->'DerivacionRef') IS DISTINCT FROM 'string'
           OR s->>'DerivacionRef' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR EXISTS(SELECT 1 FROM unnest(ARRAY['LlamamientoRef',
                'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef']) campo
                WHERE jsonb_typeof(s->campo) IS DISTINCT FROM 'string' OR s->>campo IS DISTINCT FROM '')
        THEN RAISE EXCEPTION 'referencias derivadas inválidas' USING ERRCODE='P0610'; END IF;
    ELSE
        FOREACH v_campo IN ARRAY ARRAY['LlamamientoRef',
            'ResolucionLlamamientoAceptadaRef','ReciboResolucionAceptadaRef'] LOOP
            IF jsonb_typeof(s->v_campo) IS DISTINCT FROM 'string'
               OR (s->>v_campo)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION 'referencia de propuesta inválida' USING ERRCODE='P0610'; END IF;
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        v_snapshot:=s->v_campo;
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_snapshot,ARRAY['Referencia','Version','HuellaSHA256']) IS NOT TRUE THEN
            RAISE EXCEPTION 'publicación de propuesta inválida' USING ERRCODE='P0610';
        END IF;
        IF (SELECT count(*) FROM json_each(((p_material::json)->'Solicitud')->v_campo))<>3
           OR jsonb_typeof(v_snapshot->'Referencia') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'Referencia')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_snapshot->'Version' IS DISTINCT FROM '1'::jsonb
           OR jsonb_typeof(v_snapshot->'HuellaSHA256') IS DISTINCT FROM 'string'
           OR (v_snapshot->>'HuellaSHA256')!~'^[0-9a-f]{64}$'
           OR v_snapshot->>'HuellaSHA256'=repeat('0',64) THEN
            RAISE EXCEPTION 'compromiso de publicación inválido' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    IF v_derivada THEN
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b,ARRAY[
            'DerivacionRef','PadreRef','HijoRef','OfertaRef','NumeroPlaza',
            'AceptacionRef','ReciboDerivacionRef']) IS NOT TRUE
           OR (SELECT count(*) FROM json_each((p_material::json)->'AceptacionDerivada'))<>7
        THEN RAISE EXCEPTION 'CT236: evidencia derivada inválida' USING ERRCODE='P0610'; END IF;
    ELSE
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b,
           ARRAY['Selector','RespuestaRef','RespondidaEn']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(b->'Selector',
           ARRAY['organizacion_ref','expediente_ref','clave_ct','unidad_ref','categoria_ref',
                 'necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref']) IS NOT TRUE
       OR (SELECT count(*) FROM json_each((p_material::json)->'AceptacionBolsaNativa'))<>3
       OR (SELECT count(*) FROM json_each(((p_material::json)->'AceptacionBolsaNativa')->'Selector'))<>10
       OR EXISTS(SELECT 1 FROM jsonb_each(b->'Selector') x
          WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
            OR octet_length(x.value#>>'{}') NOT BETWEEN 3 AND 160
            OR x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
       OR b#>>'{Selector,organizacion_ref}' IS DISTINCT FROM s->>'OrganizacionRef'
       OR b#>>'{Selector,expediente_ref}' IS DISTINCT FROM s->>'ExpedienteRef'
       OR b#>>'{Selector,llamamiento_ref}' IS DISTINCT FROM s->>'LlamamientoRef'
       OR jsonb_typeof(b->'RespuestaRef') IS DISTINCT FROM 'string'
       OR b->>'RespuestaRef' !~ '^respuesta-portal:[0-9a-f]{64}$'
       OR jsonb_typeof(b->'RespondidaEn') IS DISTINCT FROM 'string'
       OR b->>'RespondidaEn' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'CT211: respuesta nativa inválida' USING ERRCODE='P0610';
    END IF;
    BEGIN v_fecha_bolsa:=(b->>'RespondidaEn')::timestamptz;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'CT211: fecha de respuesta inválida' USING ERRCODE='P0610';
    END;
    IF NOT isfinite(v_fecha_bolsa) OR v_fecha_bolsa='0001-01-01T00:00:00Z'::timestamptz
       OR v_fecha_bolsa>clock_timestamp() THEN
        RAISE EXCEPTION 'CT211: fecha de respuesta inválida' USING ERRCODE='P0610';
    END IF;
    END IF;
    v_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_contexto_hash:=encode(sha256(convert_to(
        '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
        '"},"atributos":{"material_sha256":"'||v_hash||'"}}','UTF8')),'hex');
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'decisión de propuesta inválida' USING ERRCODE='P0613';
    END;
    IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.formalizacion.propuesta.registrar'
       OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'propuesta_formalizacion_ct'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'permiso de propuesta divergente' USING ERRCODE='P0613';
    END IF;
    BEGIN
        SELECT * INTO STRICT v_consumo
          FROM vec_autorizacion_atestada_v3.registrar_y_consumir_propuesta_formalizacion_ct_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload,p_sobre,p_evidencia,p_raiz);
    EXCEPTION WHEN insufficient_privilege OR data_exception OR SQLSTATE 'P0613' THEN
        RAISE EXCEPTION 'consumo de propuesta denegado' USING ERRCODE='P0613';
    END;
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_hash THEN
        RAISE EXCEPTION 'propuesta requiere consumo nuevo ligado' USING ERRCODE='P0613';
    END IF;

    -- Primera lectura de negocio: permiso fresco ya consumido en esta tx.
    FOREACH v_campo IN ARRAY ARRAY['TipoFormalizacion','Plantilla','PoliticaFirma','PlanFirma'] LOOP
        IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo p
            WHERE p.componente=v_campo AND p.referencia=s->v_campo->>'Referencia'
              AND to_jsonb(p.version)=s->v_campo->'Version' AND p.huella_sha256=s->v_campo->>'HuellaSHA256'
              AND p.huella_sha256=encode(sha256(convert_to(p.contenido,'UTF8')),'hex')) THEN
            RAISE EXCEPTION 'publicación de propuesta no acreditada' USING ERRCODE='P0610';
        END IF;
    END LOOP;
    IF v_derivada THEN
        v_derivacion:=vec_contratacion_temporal.acreditar_antecedente_derivado_ct236_v1(s,b);
        SELECT creada_en INTO STRICT v_fecha_bolsa
        FROM vec_contratacion_temporal.nombramiento_derivado_v1
        WHERE derivacion_ref=s->>'DerivacionRef';
    ELSE
    SELECT * INTO v_resolucion FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
     WHERE r.resolucion_ref=s->>'ResolucionLlamamientoAceptadaRef' AND r.recibo_ref=s->>'ReciboResolucionAceptadaRef'
       AND r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef' AND r.estado='confirmado'
       AND r.solicitud_json->>'Respuesta'='aceptacion' AND r.revision_respuesta_rrhh AND r.revision_plazo_rrhh
       AND r.politica_ref='politica:ct:revision-manual-sintetica:20260906' AND r.politica_version=1
       AND r.politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'
       AND r.solicitud_json->>'CriterioValidacionRef'=r.politica_ref
       AND r.solicitud_json->'RevisionRespuestaRRHH'='true'::jsonb
       AND r.solicitud_json->'RevisionPlazoRRHH'='true'::jsonb
       AND r.estado_plazo='vigente' AND r.version_resultante=3
       AND r.recibo_json->>'Estado'='confirmado' AND r.recibo_json->'Solicitud'=r.solicitud_json
       AND r.recibo_json->>'ResolucionRef'=r.resolucion_ref AND r.recibo_json->>'ReciboLocalRef'=r.recibo_ref
       AND r.recibo_json->>'EvaluacionPlazoRef'=r.evaluacion_plazo_ref
       AND r.recibo_json->'Politica'=jsonb_build_object('Referencia',r.politica_ref,'Version',r.politica_version,'HuellaSHA256',r.politica_sha256)
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'aceptación CT no acreditada' USING ERRCODE='P0615';
    END IF;
    SELECT j.recibo_json,e.recibo_json,continuacion.continuacion_recibo
      INTO v_respuesta,v_seleccion,v_continuacion
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh j
      JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c ON c.comunicacion_ref=j.comunicacion_ref
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=j.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
      -- CT65: confirmación original del primer sucesor, ligada al aviso exacto.
      LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh continuacion
        ON c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
       AND continuacion.seleccion_clave=e.clave_idempotencia
       AND continuacion.organizacion_ref=j.organizacion_ref
       AND continuacion.expediente_ref=j.expediente_ref
       AND continuacion.llamamiento_ref=e.recibo_json->>'llamamiento_ref'
       AND continuacion.llamamiento_ref<>j.llamamiento_ref
       AND continuacion.estado='confirmado'
       AND continuacion.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')
       AND continuacion.continuacion_clave IS NOT NULL
       AND continuacion.continuacion_recibo->>'Estado'='confirmado'
       AND continuacion.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND continuacion.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=j.organizacion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=j.expediente_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ResolucionRef'=continuacion.resolucion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion.continuacion_clave::text
       AND continuacion.continuacion_recibo->>'LlamamientoAnteriorRef'=continuacion.llamamiento_ref
       AND continuacion.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=j.llamamiento_ref
     WHERE j.justificante_ref=v_resolucion.justificante_ref AND j.seleccion_clave=v_resolucion.seleccion_clave
       AND j.organizacion_ref=v_resolucion.organizacion_ref AND j.expediente_ref=v_resolucion.expediente_ref
       AND j.llamamiento_ref=v_resolucion.llamamiento_ref AND j.comunicacion_ref=v_resolucion.comunicacion_ref
       AND j.respuesta='aceptacion' AND j.version_comunicacion=2 AND j.estado='registrada_por_rrhh'
       AND c.organizacion_ref=j.organizacion_ref AND c.expediente_ref=j.expediente_ref AND c.llamamiento_ref=j.llamamiento_ref
       AND c.estado='registrada_localmente' AND c.version_resultante=2
       AND ((e.solicitud_json->'version_expediente'=to_jsonb(v_n)
       AND e.recibo_json->'version_expediente'=to_jsonb(v_n))
        -- CT128: tras una no incorporación la selección es la original y el
        -- expediente sigue en una versión posterior a esa no incorporación.
        OR (continuacion.solicitud_json->>'Respuesta'='aceptacion'
            AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
            AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n128
                         WHERE n128.aceptacion_resolucion_ref=continuacion.resolucion_ref
                           AND n128.intencion_ref=continuacion.continuacion_recibo->'Solicitud'->>'IntencionRef'
                           AND n128.organizacion_ref=j.organizacion_ref AND n128.expediente_ref=j.expediente_ref
                           AND n128.version_esperada+1<=v_n)))
       AND e.situacion='confirmada' AND e.solicitud_json->>'organizacion_ref'=j.organizacion_ref
       AND e.solicitud_json->>'expediente_ref'=j.expediente_ref
       AND e.recibo_json->>'organizacion_ref'=j.organizacion_ref AND e.recibo_json->>'expediente_ref'=j.expediente_ref
       AND (
           (NOT ((c.material_json->'solicitud') ? 'TipoAntecedente')
            AND e.recibo_json->>'llamamiento_ref'=j.llamamiento_ref
            AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef')
           OR (c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
               AND continuacion.resolucion_ref IS NOT NULL)
       )
       AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND j.recibo_json->>'JustificanteRef'=j.justificante_ref AND j.recibo_json->>'ReciboRef'=j.recibo_ref
       AND j.recibo_json->'Solicitud'=j.material_json AND j.recibo_json->>'Estado'='registrada_por_rrhh'
       AND j.registrada_en<=v_resolucion.resuelta_en
     FOR SHARE OF j,c,e;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'antecedentes de aceptación incompatibles' USING ERRCODE='P0615';
    END IF;
    -- Los hechos CT anteriores siguen siendo propios y se cotejan con la
    -- selección nativa firmada; Bolsa acredita su respuesta en su propietario.
    IF v_continuacion IS NOT NULL
       OR v_seleccion#>>'{procedencia,sin_retencion_declarada}' IS DISTINCT FROM 'true'
       OR v_seleccion->>'llamamiento_ref' IS DISTINCT FROM s->>'LlamamientoRef'
       OR v_seleccion->>'recibo_ref' IS DISTINCT FROM b#>>'{Selector,recibo_emision_ref}'
       OR v_seleccion#>>'{bolsa,referencia}' IS DISTINCT FROM b#>>'{Selector,bolsa_ref}'
       OR v_seleccion#>>'{necesidad,referencia}' IS DISTINCT FROM b#>>'{Selector,necesidad_ref}'
       OR v_resolucion.seleccion_clave::text IS DISTINCT FROM b#>>'{Selector,clave_ct}'
       OR v_fecha_bolsa>v_resolucion.resuelta_en THEN
        RAISE EXCEPTION 'CT211: selección nativa desligada' USING ERRCODE='P0615';
    END IF;
    v_bolsa_nativa:=vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(b->'Selector');
    IF v_bolsa_nativa->>'estado' NOT IN ('pendiente','acreditado')
       OR v_bolsa_nativa->'anclaje' IS NULL
       OR v_bolsa_nativa#>>'{anclaje,version_expediente}' IS DISTINCT FROM v_seleccion->>'version_expediente'
       OR v_bolsa_nativa#>>'{anclaje,respuesta_ref}' IS DISTINCT FROM b->>'RespuestaRef'
       OR (v_bolsa_nativa#>>'{anclaje,respondida_en}')::timestamptz IS DISTINCT FROM v_fecha_bolsa THEN
        RAISE EXCEPTION 'CT211: aceptación Bolsa no acreditada' USING ERRCODE='P0615';
    END IF;
    END IF;
    -- Recuperación antes de exigir la versión N. Misma autoridad fresca, actor, perfil
    -- y material completo; no vuelve a tocar expediente, recibos o cadenas.
    SELECT * INTO v_previa FROM vec_contratacion_temporal.propuesta_formalizacion
     WHERE organizacion_ref=s->>'OrganizacionRef' AND clave_idempotencia=(s->>'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d->>'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
            RAISE EXCEPTION 'recuperación de propuesta denegada' USING ERRCODE='P0613';
        END IF;
        IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM m THEN
            RAISE EXCEPTION 'clave de propuesta divergente' USING ERRCODE='P0611';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('Estado','replay_confirmado');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion
        WHERE derivacion_ref=s->>'DerivacionRef' OR resolucion_ref=v_resolucion.resolucion_ref OR
            (organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
             -- CT128: no cuentan las ya sustituidas ni la que sustituye esta propuesta.
             AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x128
                              WHERE x128.propuesta_anterior_ref=propuesta_formalizacion.propuesta_ref)
             AND propuesta_ref IS DISTINCT FROM (SELECT p128.propuesta_ref FROM vec_contratacion_temporal.propuesta_sustituible_ct128(
                 v_continuacion,s->>'OrganizacionRef',s->>'ExpedienteRef',v_n) p128))) THEN
        RAISE EXCEPTION 'aceptación con propuesta previa' USING ERRCODE='P0611';
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=s->>'ExpedienteRef' FOR UPDATE OF a,v;
    IF NOT FOUND OR v_actual.version<>v_n OR v_actual.fase_clave<>'fiscalizacion' OR v_actual.estado<>'en_curso'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM s->>'OrganizacionRef'
       OR v_actual.agregado_json->>'referencia' IS DISTINCT FROM s->>'ExpedienteRef'
       OR v_actual.agregado_json->'version' IS DISTINCT FROM to_jsonb(v_n)
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'fiscalizacion'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR jsonb_array_length(v_actual.agregado_json->'actuaciones')<>(CASE WHEN v_derivada THEN v_n-5 ELSE v_n END)
       OR coalesce(v_actual.agregado_json#>>'{fiscalizacion,resultado}','') NOT IN ('favorable','favorable_con_observaciones')
       OR v_actual.agregado_json_huella_sha256<>encode(sha256(convert_to(v_actual.agregado_json::text,'UTF8')),'hex') THEN
        RAISE EXCEPTION 'versión de expediente incompatible' USING ERRCODE='P0612';
    END IF;
    v_unidad:=v_actual.agregado_json#>>'{asignacion,unidad_ref}';
    IF coalesce(v_unidad,'')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (NOT v_derivada AND (v_unidad IS DISTINCT FROM b#>>'{Selector,unidad_ref}'
           OR v_actual.agregado_json#>>'{analisis,categoria_ref}' IS DISTINCT FROM b#>>'{Selector,categoria_ref}')) THEN
        RAISE EXCEPTION 'unidad de expediente no disponible' USING ERRCODE='P0614';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_fecha_bolsa OR v_ahora<v_actual.registrada_en THEN
        RAISE EXCEPTION 'reloj anterior a los antecedentes' USING ERRCODE='P0614';
    END IF;
    v_propuesta:='propuesta:'||gen_random_uuid()::text;
    v_recibo:='recibo:'||gen_random_uuid()::text;
    v_evento:='evento:'||gen_random_uuid()::text;
    v_actuacion:=jsonb_build_object(
        'secuencia',v_siguiente,'version_expediente',v_siguiente,'accion_clave','registrar_propuesta_formalizacion',
        'actor_ref',d->>'principal_id','unidad_ref',v_unidad,'recibo_ref',v_recibo,
        'realizada_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'fase_origen',v_actual.agregado_json->'fase_actual','fase_destino','nombramiento',
        'estado_origen',v_actual.agregado_json->'estado_actual','estado_destino','en_curso');
    v_agregado:=v_actual.agregado_json||jsonb_build_object(
        'version',v_siguiente,'fase_actual','nombramiento','estado_actual','en_curso',
        'actualizado_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    v_agregado_hash:=encode(sha256(convert_to(v_agregado::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-PROPUESTA-V1'||chr(10)||
        (s->>'ExpedienteRef')||chr(10)||v_siguiente::text||chr(10)||v_agregado_hash||chr(10)||
        v_propuesta||chr(10)||v_recibo||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',v_siguiente,v_agregado,v_agregado_hash,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
        'propuesta_formalizacion_o6',v_propuesta,v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version=v_siguiente,actualizada_en=v_ahora,operacion_ref=v_propuesta
     WHERE expediente_ref=s->>'ExpedienteRef' AND version=v_n;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'versión de propuesta perdida' USING ERRCODE='P0612';
    END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-PROPUESTA-V1'||chr(10)||v_actuacion::text||chr(10)||v_recibo||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en
    ) VALUES (
        s->>'ExpedienteRef',v_siguiente,v_siguiente,v_propuesta,v_recibo,v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991::numeric THEN
        RAISE EXCEPTION 'límite de outbox alcanzado' USING ERRCODE='P0614';
    END IF;
    v_secuencia:=v_secuencia+1;
    v_payload:=convert_to((jsonb_build_object('esquema','vec.contratacion-temporal.propuesta-formalizacion-registrada.v1',
        'expediente_ref',s->>'ExpedienteRef','version_resultante',v_siguiente,'propuesta_ref',v_propuesta,
        'recibo_ref',v_recibo) || (CASE WHEN v_derivada
          THEN jsonb_build_object('derivacion_ref',s->>'DerivacionRef')
          ELSE jsonb_build_object('resolucion_aceptada_ref',v_resolucion.resolucion_ref) END))::text,'UTF8');
    v_outbox_hash:=encode(sha256(v_anterior::bytea||v_payload),'hex');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,
        payload_canonico,payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en
    ) VALUES (
        v_evento,v_secuencia,v_propuesta,s->>'ExpedienteRef',v_siguiente,'contratacion_temporal.propuesta_formalizacion_registrada',
        v_payload,encode(sha256(v_payload),'hex'),v_anterior,v_outbox_hash,v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=v_outbox_hash,actualizada_en=v_ahora WHERE control_id;
    -- CT128: la propuesta anterior queda sustituida por su no incorporación.
    INSERT INTO vec_contratacion_temporal.propuesta_sustitucion_v1(
        propuesta_ref,propuesta_anterior_ref,no_incorporacion_recibo_ref,organizacion_ref,expediente_ref,ronda,registrada_en)
    SELECT v_propuesta,p128.propuesta_ref,p128.no_incorporacion_recibo_ref,s->>'OrganizacionRef',s->>'ExpedienteRef',p128.ronda,v_ahora
      FROM vec_contratacion_temporal.propuesta_sustituible_ct128(v_continuacion,s->>'OrganizacionRef',s->>'ExpedienteRef',v_n) p128;
    v_resultado:=jsonb_build_object('Solicitud',s,'PropuestaRef',v_propuesta,'ReciboLocalRef',v_recibo,
        'AuditoriaRef',v_consumo.auditoria_ref,'VersionResultante',v_siguiente,
        'ConfirmadaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    INSERT INTO vec_contratacion_temporal.propuesta_formalizacion (
        propuesta_ref,organizacion_ref,expediente_ref,llamamiento_ref,resolucion_ref,recibo_aceptacion_ref,derivacion_ref,
        clave_idempotencia,version_previa,version_resultante,material,material_json,material_sha256,solicitud_json,
        aceptacion_bolsa_json,actor_ref,perfil_ref,auditoria_ref,decision_ref,consumo_huella_sha256,
        evidencia_huella_sha256,recibo_ref,recibo_json,evento_ref,confirmada_en
    ) VALUES (
        v_propuesta,s->>'OrganizacionRef',s->>'ExpedienteRef',NULLIF(s->>'LlamamientoRef',''),
        (CASE WHEN v_derivada THEN NULL ELSE v_resolucion.resolucion_ref END),
        (CASE WHEN v_derivada THEN NULL ELSE v_resolucion.recibo_ref END),
        (CASE WHEN v_derivada THEN s->>'DerivacionRef' ELSE NULL END),
        (s->>'ClaveIdempotencia')::uuid,v_n,v_siguiente,p_material,m,v_hash,s,b,d->>'principal_id',d->>'perfil_activo_ref',
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,
        encode(sha256(p_evidencia),'hex'),v_recibo,v_resultado,v_evento,v_ahora);
    RETURN v_resultado;
EXCEPTION
    WHEN unique_violation THEN
        RAISE EXCEPTION 'conflicto de propuesta' USING ERRCODE='P0611';
    WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
        RAISE EXCEPTION 'propuesta no disponible' USING ERRCODE='P0614';
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct163()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET row_security=on SET search_path=pg_catalog,pg_temp
AS $funcion$
DECLARE
    v_anterior jsonb;
    v_tiene_circuito boolean;
BEGIN
    v_tiene_circuito := NEW.agregado_json ? 'circuito';
    IF NEW.version = 1 THEN
        IF v_tiene_circuito AND (
            NEW.origen_version IS DISTINCT FROM 'alta_o2'
            OR NOT vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json)
            OR NEW.agregado_json ->> 'version' IS DISTINCT FROM NEW.version::text
            OR NEW.agregado_json #>> '{circuito,definicion,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
            OR (NEW.agregado_json #>> '{circuito,definicion,version}')::numeric IS DISTINCT FROM NEW.flujo_version
            OR NEW.agregado_json #>> '{circuito,definicion,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
            OR pg_catalog.jsonb_array_length(NEW.agregado_json #> '{circuito,hitos}') <> 0
        ) THEN
            RAISE EXCEPTION 'CT163: alta de circuito divergente' USING ERRCODE = '55000';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.version=6 AND NEW.origen_version='derivacion_nombramiento_ct234' THEN
        IF vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
            NEW.expediente_ref,NEW.agregado_json) IS NOT TRUE
           OR (v_tiene_circuito AND
               vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json) IS NOT TRUE)
        THEN RAISE EXCEPTION 'CT234: circuito inicial divergente' USING ERRCODE='55000'; END IF;
        RETURN NEW;
    END IF;
    SELECT v.agregado_json INTO v_anterior
      FROM vec_contratacion_temporal.expediente_version_integral v
     WHERE v.expediente_ref = NEW.expediente_ref AND v.version = NEW.version - 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CT163: falta la versión anterior' USING ERRCODE = '55000';
    END IF;
    IF v_tiene_circuito OR v_anterior ? 'circuito' THEN
        IF NOT vec_contratacion_temporal.circuito_vinculado_ct163(NEW.agregado_json)
           OR NEW.agregado_json ->> 'version' IS DISTINCT FROM NEW.version::text
           OR NEW.agregado_json -> 'flujo' IS DISTINCT FROM v_anterior -> 'flujo'
           OR NEW.agregado_json #>> '{circuito,definicion,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
           OR (NEW.agregado_json #>> '{circuito,definicion,version}')::numeric IS DISTINCT FROM NEW.flujo_version
           OR NEW.agregado_json #>> '{circuito,definicion,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
           OR (NEW.agregado_json -> 'circuito' IS DISTINCT FROM v_anterior -> 'circuito'
               AND NOT vec_contratacion_temporal.circuito_siguiente_ct163(v_anterior, NEW.agregado_json)) THEN
            RAISE EXCEPTION 'CT163: transición de circuito divergente' USING ERRCODE = '55000';
        END IF;
    END IF;
    IF (v_anterior ? 'instruccion_compartida' OR NEW.agregado_json ? 'instruccion_compartida')
       AND vec_contratacion_temporal.propuesta_derivada_ct236_v1(
         NEW.expediente_ref,v_anterior,NEW.agregado_json,NEW.version) IS NOT TRUE THEN
        RAISE EXCEPTION 'CT236: historia del hijo divergente' USING ERRCODE='55000'; END IF;
    RETURN NEW;
END
$funcion$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct164()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET row_security=on SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_anterior jsonb; v_anterior_circuito boolean:=false;
BEGIN
 IF NEW.version=6 AND NEW.origen_version='derivacion_nombramiento_ct234' THEN
  IF vec_contratacion_temporal.inicio_derivado_acreditado_ct234_v1(
       NEW.expediente_ref,NEW.agregado_json) IS NOT TRUE
     OR vec_contratacion_temporal.circuito_agregado_valido_ct164(NEW.agregado_json) IS NOT TRUE
  THEN RAISE EXCEPTION 'CT234: terna inicial divergente' USING ERRCODE='42501'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.version>1 THEN
  SELECT agregado_json INTO v_anterior FROM vec_contratacion_temporal.expediente_version_integral
   WHERE expediente_ref=NEW.expediente_ref AND version=NEW.version-1;
  v_anterior_circuito:=coalesce(v_anterior ? 'circuito',false);
 END IF;
 IF v_anterior_circuito OR NEW.flujo_ref='flujo:ct:rrhh:20261002' OR NEW.agregado_json ? 'circuito' THEN
  IF vec_contratacion_temporal.circuito_agregado_valido_ct164(NEW.agregado_json) IS NOT TRUE
   OR NEW.agregado_json #>> '{flujo,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
   OR (NEW.agregado_json #>> '{flujo,version}')::numeric IS DISTINCT FROM NEW.flujo_version
   OR NEW.agregado_json #>> '{flujo,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
   OR NEW.agregado_json->>'version' IS DISTINCT FROM NEW.version::text
  THEN RAISE EXCEPTION 'CT164: terna o circuito inicial divergente' USING ERRCODE='42501'; END IF;
  IF NEW.version>1 THEN
   IF v_anterior IS NULL OR (CASE
       WHEN v_anterior ? 'instruccion_compartida' OR NEW.agregado_json ? 'instruccion_compartida'
       THEN vec_contratacion_temporal.propuesta_derivada_ct236_v1(
         NEW.expediente_ref,v_anterior,NEW.agregado_json,NEW.version) IS NOT TRUE
       ELSE vec_contratacion_temporal.circuito_acto_admitido_ct164(v_anterior,NEW.agregado_json) IS NOT TRUE END)
   THEN RAISE EXCEPTION 'CT164: acto del circuito sin fuente nominal verificada' USING ERRCODE='42501'; END IF;
  END IF;
 END IF;
 RETURN NEW;
END $f$;

DO $post_funciones$
BEGIN
 IF (SELECT count(*) FROM pg_temp.ct236_pre)<>4
    OR EXISTS(SELECT 1 FROM pg_temp.ct236_pre antes
      LEFT JOIN pg_proc despues ON despues.oid=antes.oid
      WHERE despues.oid IS NULL OR to_regprocedure(antes.firma) IS DISTINCT FROM antes.oid
        OR despues.proowner IS DISTINCT FROM antes.propietario
        OR despues.proacl IS DISTINCT FROM antes.acl OR despues.prosecdef IS NOT TRUE
        OR encode(sha256(convert_to(despues.prosrc,'UTF8')),'hex') IS DISTINCT FROM antes.posthash
        OR cardinality(despues.proconfig) IS DISTINCT FROM cardinality(antes.config_final)
        OR EXISTS(SELECT 1 FROM unnest(antes.config_final) cfg WHERE cfg<>ALL(despues.proconfig)))
 THEN RAISE EXCEPTION 'CT236: postimagen CT211 divergente' USING ERRCODE='55000'; END IF;
END $post_funciones$;
-- CT236-GENERADO-FIN
COMMIT;
