\set ON_ERROR_STOP on
-- CT236: propuesta ordinaria y propuesta del hijo CT234 en las mismas dos
-- funciones CT211. Las referencias de resolución antigua solo viven en la
-- rama ordinaria; una aceptación derivada conserva su recibo y origen reales.
-- CT237 se instala antes: la lectura de detalle autorizada publica el recibo.
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
    OR (SELECT count(*) FROM pg_attribute a
       WHERE a.attrelid='vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regclass
         AND a.attnum>0 AND NOT a.attisdropped)<>17
    OR (SELECT count(*) FROM pg_type t
       WHERE t.oid='vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regtype
         AND t.typowner='vec_contratacion_temporal_propietario'::regrole
         AND t.typacl::text='{vec_contratacion_temporal_propietario=U/vec_contratacion_temporal_propietario}')<>1
    OR NOT EXISTS(SELECT 1 FROM pg_proc p
       WHERE p.oid=to_regprocedure('vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)')
         AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='aced84f976a15715df9fc0606111cb5359ccc1d5dd9cea646b1181689a7069c4')
    OR NOT EXISTS(SELECT 1 FROM pg_proc p
       WHERE p.oid=to_regprocedure('vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)')
         AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='f0685b3d58f9d448c57a806ae4079b47ed2f74703177f08680088ce9ced27b66')
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

ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
 ADD ATTRIBUTE propuesta_formalizacion jsonb;

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
-- Fuente: CT211 instalado y postimágenes explícitas CT234/CT237; huellas cotejadas.
CREATE TEMP TABLE pg_temp.ct236_pre(
 firma text PRIMARY KEY,oid oid NOT NULL,propietario oid NOT NULL,acl aclitem[],
 posthash text NOT NULL,config_final text[] NOT NULL,
 volatilidad "char" NOT NULL,definidor boolean NOT NULL,paralelo "char" NOT NULL
) ON COMMIT DROP;
DO $pre_funciones$
DECLARE x record;p pg_proc%ROWTYPE;f regprocedure;
BEGIN
 FOR x IN SELECT * FROM (VALUES
 ('vec_contratacion_temporal.leer_antecedente_propuesta_formalizacion_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','ea35ab5bba36c8bcce5c7e1f26af9089f47fe926dc677e68238f83328da13712','1a0665d48d931983b55f9720d5432c3ce66418ad3d7a543a8bb0786e0cc0c701','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[],'v',true,'u'),
 ('vec_contratacion_temporal.registrar_propuesta_formalizacion_nativa_ct211_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','b2414d394ce0e0389a69a9b0f9f7cb792aa02e32bea7884486adebce1e394138','bcb2ec7320e5aecbd0e1a13736542e9c3815b3c318c16276499d83efba49babf','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[],'v',true,'u'),
 ('vec_contratacion_temporal.proteger_version_circuito_ct163()','08bfd3363ae587062d72083aed4fa81821e272a1b81d73f62aa754710b8a5973','6f5572ec3a51512801b289e42bfeab124e6fa32bdf97c1a6553b5cbe915b56fb','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['row_security=on','search_path=pg_catalog, pg_temp']::text[],'v',true,'u'),
 ('vec_contratacion_temporal.proteger_version_circuito_ct164()','e3ac1b9724108eaf92b2e55131bb1c8ca980718932dd9823ced2ab7b939e4f0c','24fc0173622809f6380207d9c789af0c781e320e88542141b579283bc31476e5','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['row_security=on','search_path=pg_catalog, pg_temp']::text[],'v',true,'u'),
 ('vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)','aced84f976a15715df9fc0606111cb5359ccc1d5dd9cea646b1181689a7069c4','e79ba196d514b630bc16f5bc1d138592fa900712c2014c9a02d6fb23fe19eb81','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[],'s',true,'u'),
 ('vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)','f0685b3d58f9d448c57a806ae4079b47ed2f74703177f08680088ce9ced27b66','5f30f7540c9ce8b2d9a2023006be49c6f08d616f0bef3df68fab13b2b3be02e4','{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}',ARRAY['search_path=pg_catalog, pg_temp']::text[],'i',false,'s')
 ) AS fuente(firma,prehash,posthash,acl_original,config_original,volatilidad,definidor,paralelo) LOOP
  f:=to_regprocedure(x.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'CT236: función fuente ausente' USING ERRCODE='55000'; END IF;
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR p.prosecdef IS DISTINCT FROM x.definidor
     OR p.provolatile IS DISTINCT FROM x.volatilidad
     OR p.proparallel IS DISTINCT FROM x.paralelo
     OR p.proacl::text IS DISTINCT FROM x.acl_original
     OR cardinality(p.proconfig) IS DISTINCT FROM cardinality(x.config_original)
     OR EXISTS(SELECT 1 FROM unnest(x.config_original) cfg WHERE cfg<>ALL(p.proconfig))
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM x.prehash
  THEN RAISE EXCEPTION 'CT236: preimagen divergente: %',f USING ERRCODE='55000'; END IF;
  INSERT INTO pg_temp.ct236_pre VALUES(x.firma,f,p.proowner,p.proacl,x.posthash,x.config_original,
    x.volatilidad,x.definidor,x.paralelo);
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

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.materializar_detalle_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_corte_global numeric)
 RETURNS vec_contratacion_temporal.materializacion_detalle_rrhh_v1
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_fila record;
    v_agregado jsonb;
    v_padre jsonb;
    v_instruccion jsonb;
    v_version_instruccion numeric(20,0);
    v_expediente_instruccion_ref text;
    v_json_seguro boolean;
    v_derivacion record;
    v_propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
    v_hito_propuesta jsonb; v_propuesta_publica jsonb;
    v_hay_propuesta boolean; v_hay_hito boolean;
    v_nodo jsonb;
    v_indice integer;
    v_total integer;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud
        vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis
        vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura
        vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion
        vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion
        vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_subsanacion_nodo jsonb;
    v_subsanaciones integer;
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_hitos
        vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_analisis_presente boolean;
    v_cobertura_presente boolean;
    v_asignacion_presente boolean;
    v_fiscalizacion_presente boolean;
    v_subsanacion_presente boolean;
    v_decision_gobernada boolean;
    v_coste_presente boolean;
    v_referencia_analisis numeric(20, 0) := 0;
    v_referencia_cobertura numeric(20, 0) := 0;
    v_referencia_asignacion numeric(20, 0) := 0;
    v_referencia_fiscalizacion numeric(20, 0) := 0;
    v_referencia_subsanacion numeric(20, 0) := 0;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        p_consulta
    );

    -- MATERIALIZED fija primero la versión actual relativa al corte. El
    -- predicado exterior aplica después el ámbito y el control de versión.
    WITH elegida AS MATERIALIZED (
        SELECT publicacion.*, historia.agregado_json
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicacion
          JOIN vec_contratacion_temporal.expediente_version_integral historia
            ON historia.expediente_ref = publicacion.expediente_ref
           AND historia.version = publicacion.version
           AND historia.flujo_ref = publicacion.flujo_ref
           AND historia.flujo_version = publicacion.flujo_version
           AND historia.flujo_huella_sha256 =
               publicacion.flujo_huella_sha256
           AND historia.fase_clave = publicacion.fase_clave
           AND historia.estado = publicacion.estado_clave
           AND historia.agregado_json_huella_sha256 =
               publicacion.agregado_huella_sha256
           AND historia.registrada_en = publicacion.registrada_en
         WHERE publicacion.expediente_ref = p_consulta.expediente_ref
           AND publicacion.corte_global <= p_corte_global
         ORDER BY publicacion.corte_global DESC
         LIMIT 1
    )
    SELECT elegida.*
      INTO STRICT v_fila
      FROM elegida
     WHERE elegida.organizacion_ref = p_alcance.organizacion_ref
       AND (
           p_consulta.version_observada = 0
           OR p_consulta.version_observada = elegida.version
       )
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN
               elegida.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN
               elegida.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN
               elegida.unidad_ref = p_alcance.ambito_ref
           ELSE false
       END;

    v_agregado := v_fila.agregado_json;
    v_version_instruccion := v_fila.version;
    v_expediente_instruccion_ref := v_fila.expediente_ref;
    IF v_agregado ? 'instruccion_compartida' THEN
        SELECT derivacion.*, alta.derivacion_ref AS alta_derivacion_ref
          INTO STRICT v_derivacion
          FROM vec_contratacion_temporal.nombramiento_derivado_v1 derivacion
          JOIN vec_contratacion_temporal.expediente_alta alta
            ON alta.expediente_ref=derivacion.hijo_ref
           AND alta.derivacion_ref=derivacion.derivacion_ref
         WHERE derivacion.hijo_ref=v_fila.expediente_ref;
        -- CT234 fija snapshot y preimagen desde el mismo agregado JSONB.
        -- El recibo y la versión v6 conservan esa fuente sin leer el padre actual.
        v_padre := v_agregado #> '{instruccion_compartida,expediente_padre}';
        IF pg_catalog.jsonb_typeof(v_padre)<>'object'
           OR pg_catalog.jsonb_typeof(v_padre->'actuaciones')<>'array'
           OR v_agregado #>> '{instruccion_compartida,oferta_ref}' IS DISTINCT FROM v_derivacion.oferta_ref
           OR v_agregado #>> '{instruccion_compartida,numero_plaza}' IS DISTINCT FROM v_derivacion.numero_plaza::text
           OR v_agregado #>> '{instruccion_compartida,recibo_derivacion_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR pg_catalog.jsonb_typeof(v_agregado#>'{instruccion_compartida,preimagen_padre_json}') IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(v_agregado#>>'{instruccion_compartida,preimagen_padre_json}') NOT BETWEEN 2 AND 262144
           OR (v_agregado#>>'{instruccion_compartida,preimagen_padre_json}')::jsonb IS DISTINCT FROM v_padre
           OR v_agregado#>>'{instruccion_compartida,preimagen_padre_json}' IS DISTINCT FROM v_padre::text
           OR v_agregado #>> '{instruccion_compartida,huella_padre_sha256}' IS DISTINCT FROM
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    v_agregado#>>'{instruccion_compartida,preimagen_padre_json}','UTF8')),'hex')
           OR v_derivacion.recibo_json#>'{expediente,instruccion_compartida}' IS DISTINCT FROM
                v_agregado->'instruccion_compartida'
           OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral inicial
                WHERE inicial.expediente_ref=v_derivacion.hijo_ref AND inicial.version=6
                  AND inicial.agregado_json=v_derivacion.recibo_json->'expediente')
           OR v_padre->>'referencia' IS DISTINCT FROM v_derivacion.padre_ref
           OR v_padre->>'version' IS DISTINCT FROM v_derivacion.version_padre::text
           OR v_agregado #>> '{actuaciones,0,recibo_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR v_agregado #>> '{actuaciones,0,actor_ref}' IS DISTINCT FROM v_derivacion.actor_ref
           OR v_agregado #>> '{actuaciones,0,accion_clave}' IS DISTINCT FROM 'contratacion_temporal.nombramiento.derivar'
           OR v_agregado->'analisis' IS DISTINCT FROM v_padre->'analisis'
           OR v_agregado->'via_cobertura' IS DISTINCT FROM v_padre->'via_cobertura'
           OR v_agregado->'asignacion' IS DISTINCT FROM v_padre->'asignacion'
           OR v_agregado->'informe_juridico' IS DISTINCT FROM v_padre->'informe_juridico'
           OR v_agregado->'fiscalizacion' IS DISTINCT FROM v_padre->'fiscalizacion'
           OR v_agregado->>'numero_visible' IS DISTINCT FROM
                (v_padre->>'numero_visible')||'-P'||v_derivacion.numero_plaza::text
        THEN RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_version_instruccion := (v_padre->>'version')::numeric(20,0);
        v_expediente_instruccion_ref := v_derivacion.padre_ref;
        IF v_version_instruccion<6 OR
           pg_catalog.jsonb_array_length(v_padre->'actuaciones')<>v_version_instruccion THEN
           RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_instruccion := pg_catalog.jsonb_build_object(
          'derivacion_ref',v_derivacion.derivacion_ref,
          'padre_ref',v_derivacion.padre_ref,
          'numero_visible_padre',v_padre->>'numero_visible',
          'oferta_ref',v_derivacion.oferta_ref,
          'numero_plaza',v_derivacion.numero_plaza,
          'recibo_derivacion_ref',v_derivacion.recibo_ref,
          'huella_padre_sha256',v_agregado#>>'{instruccion_compartida,huella_padre_sha256}',
          'creado_en_padre',v_padre->>'creado_en',
          'actualizado_en_padre',v_padre->>'actualizado_en',
          'hitos_padre',(
             SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
               'secuencia',nodo->'secuencia','version_expediente',nodo->'version_expediente',
               'accion_clave',nodo->'accion_clave','realizada_en',nodo->'realizada_en',
               'fase_origen',COALESCE(nodo->>'fase_origen',''),
               'fase_destino',nodo->'fase_destino','estado_origen',nodo->'estado_origen',
               'estado_destino',nodo->'estado_destino') ORDER BY ord)
             FROM pg_catalog.jsonb_array_elements(v_padre->'actuaciones')
               WITH ORDINALITY AS original(nodo,ord)));
    END IF;
    IF v_instruccion IS NULL THEN
        v_json_seguro := pg_catalog.octet_length(v_agregado::text)<=262144
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(v_agregado);
    ELSE
        -- El hijo contiene una copia tipada y otra textual del padre. Cada
        -- fragmento estructural conserva exactamente los controles legados.
        v_json_seguro := pg_catalog.octet_length(v_agregado::text)<=2097152
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(
            v_agregado - 'instruccion_compartida')
          AND vec_contratacion_temporal.json_rrhh_seguro_v1(v_padre);
    END IF;
    IF v_json_seguro IS NOT TRUE
       OR pg_catalog.jsonb_typeof(v_agregado -> 'solicitud') <> 'object'
       OR pg_catalog.jsonb_typeof(
           v_agregado #> '{solicitud,periodo}'
       ) <> 'object'
       OR pg_catalog.jsonb_typeof(v_agregado -> 'actuaciones') <> 'array'
       OR v_agregado ->> 'referencia' IS DISTINCT FROM
          v_fila.expediente_ref
       OR v_agregado ->> 'organizacion_ref' IS DISTINCT FROM
          v_fila.organizacion_ref
       OR v_agregado ->> 'numero_visible' IS DISTINCT FROM
          v_fila.numero_visible
       OR v_agregado ->> 'version' IS DISTINCT FROM
          v_fila.version::text
       OR v_agregado #>> '{flujo,definicion_ref}' IS DISTINCT FROM
          v_fila.flujo_ref
       OR v_agregado #>> '{flujo,version}' IS DISTINCT FROM
          v_fila.flujo_version::text
       OR v_agregado #>> '{flujo,huella_sha256}' IS DISTINCT FROM
          v_fila.flujo_huella_sha256
       OR v_agregado ->> 'fase_actual' IS DISTINCT FROM
          v_fila.fase_clave
       OR v_agregado ->> 'estado_actual' IS DISTINCT FROM
          v_fila.estado_clave THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        vec_contratacion_temporal.numero_visible_vigente_v1(
            v_fila.expediente_ref, v_fila.numero_visible
        ),
        v_fila.version,
        v_fila.flujo_ref,
        v_fila.flujo_version,
        v_fila.flujo_huella_sha256,
        v_fila.fase_clave,
        v_fila.estado_clave,
        v_fila.centro_ref,
        v_fila.categoria_ref,
        COALESCE(v_fila.modalidad_clave, ''),
        COALESCE(v_fila.unidad_ref, ''),
        v_fila.creado_en,
        v_fila.actualizado_en
    );
    v_solicitud := ROW(
        v_agregado #>> '{solicitud,grupo_subgrupo}',
        v_agregado #>> '{solicitud,motivo_clave}',
        (v_agregado #>> '{solicitud,periodo,inicio}')::timestamptz,
        (v_agregado #>> '{solicitud,periodo,fin}')::timestamptz,
        v_agregado #>> '{solicitud,periodo,causa_fin}'
    );

    v_analisis_presente := v_agregado ? 'analisis';
    IF v_analisis_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'analisis') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,periodo}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,validacion_rc}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        -- Sin coste previsto Go tampoco guarda la fuente del coste: sólo
        -- entonces «hay coste» es false. Con fuente y sin coste sigue NULL y
        -- el canon lo rechaza, como antes de CT180.
        v_coste_presente := COALESCE(
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object',
            CASE WHEN v_agregado #> '{analisis,fuente_coste_ref}' IS NULL
                 THEN false END
        );
        v_referencia_analisis := (
            v_agregado #>>
            '{analisis,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_analisis := ROW(
            v_agregado #>> '{analisis,modalidad_clave}',
            v_agregado #>> '{analisis,categoria_ref}',
            v_agregado #>> '{analisis,causa_clave}',
            (v_agregado #>> '{analisis,periodo,inicio}')::timestamptz,
            (v_agregado #>> '{analisis,periodo,fin}')::timestamptz,
            (v_agregado #>> '{analisis,porcentaje_jornada}')::smallint,
            v_agregado #>> '{analisis,validacion_rc,resultado}',
            v_coste_presente,
            CASE WHEN v_coste_presente THEN
                (v_agregado #>>
                 '{analisis,coste_previsto,centimos}')::bigint
            ELSE 0::bigint END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,coste_previsto,moneda}'
            ELSE '' END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,fuente_coste_ref}'
            ELSE '' END,
            COALESCE(v_agregado #>> '{analisis,observaciones}', ''),
            v_agregado #>> '{analisis,periodo,causa_fin}'
        );
    ELSE
        v_analisis := NULL;
    END IF;

    v_cobertura_presente := v_agregado ? 'via_cobertura';
    IF v_cobertura_presente THEN
        IF pg_catalog.jsonb_typeof(
               v_agregado -> 'via_cobertura'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_decision_gobernada :=
            pg_catalog.jsonb_typeof(
                v_agregado #>
                '{via_cobertura,decision_gobernada}'
            ) = 'object';
        v_comprobaciones := ARRAY[]::vec_contratacion_temporal
            .comprobacion_operativa_rrhh_v1[];
        IF v_decision_gobernada THEN
            IF NOT v_analisis_presente THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := (
                v_agregado #>>
                '{via_cobertura,decision_gobernada,actuacion,version_expediente}'
            )::numeric(20, 0);
            SELECT ARRAY(
                SELECT ROW(
                    evidencia.comprobacion_clave,
                    evidencia.comprobacion_resultado
                )::vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
                  FROM vec_contratacion_temporal.consumo_cobertura_lote lote
                  JOIN vec_contratacion_temporal.consumo_cobertura_evidencia evidencia
                    ON evidencia.lote_ref = lote.lote_ref
                 WHERE lote.preparacion_c1_ref = v_agregado #>>
                       '{via_cobertura,decision_gobernada,preparacion_evidencias_ref}'
                   AND lote.expediente_ref = v_expediente_instruccion_ref
                   AND lote.version_expediente = v_referencia_analisis
                   AND evidencia.expediente_ref = lote.expediente_ref
                   AND evidencia.version_expediente = lote.version_expediente
                 ORDER BY evidencia.posicion
            ) INTO v_comprobaciones;
            IF pg_catalog.cardinality(v_comprobaciones) = 0
               OR pg_catalog.cardinality(v_comprobaciones) > 32 THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
        ELSE
            IF NOT v_analisis_presente
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{via_cobertura,comprobaciones}'
               ) <> 'array' THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := v_referencia_analisis + 1;
            v_total := pg_catalog.jsonb_array_length(
                v_agregado #> '{via_cobertura,comprobaciones}'
            );
            FOR v_indice IN 0..v_total - 1 LOOP
                v_nodo := (
                    v_agregado #> '{via_cobertura,comprobaciones}'
                ) -> v_indice;
                v_comprobaciones := pg_catalog.array_append(
                    v_comprobaciones,
                    ROW(
                        v_nodo ->> 'clave',
                        v_nodo ->> 'resultado'
                    )::vec_contratacion_temporal
                        .comprobacion_operativa_rrhh_v1
                );
            END LOOP;
        END IF;
        v_cobertura := ROW(
            v_agregado #>> '{via_cobertura,via_clave}',
            v_decision_gobernada,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                v_agregado #>> '{via_cobertura,procedimiento_ref}' END,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                COALESCE(
                    v_agregado #>> '{via_cobertura,bolsa_ref}', ''
                ) END,
            v_comprobaciones
        );
    ELSE
        v_cobertura := NULL;
    END IF;

    v_asignacion_presente := v_agregado ? 'asignacion';
    IF v_asignacion_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'asignacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{asignacion,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_asignacion := (
            v_agregado #>>
            '{asignacion,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_asignacion := ROW(
            v_agregado #>> '{asignacion,unidad_ref}',
            (v_agregado #>> '{asignacion,asignada_en}')::timestamptz,
            COALESCE(v_agregado #>> '{asignacion,motivo_clave}', '')
        );
    ELSE
        v_asignacion := NULL;
    END IF;

    v_fiscalizacion_presente := v_agregado ? 'fiscalizacion';
    v_subsanacion_presente := false;
    IF v_fiscalizacion_presente THEN
        IF NOT v_asignacion_presente
           OR pg_catalog.jsonb_typeof(v_agregado -> 'fiscalizacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,actuacion_registro}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,resultado}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,fiscalizada_en}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,observaciones}'
           ) NOT IN ('string', 'null') THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_fiscalizacion := (v_agregado #>>
            '{fiscalizacion,actuacion_registro,secuencia}')::numeric(20, 0);
        IF v_referencia_fiscalizacion <= v_referencia_asignacion
           OR v_referencia_fiscalizacion > v_version_instruccion
           OR v_agregado #>> '{fiscalizacion,resultado}' NOT IN (
                'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
              ~ '[\x00-\x1F]'
           OR pg_catalog.char_length(
                COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
           ) > 2000
           OR (v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
               AND COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '') <> '')
           OR (v_agregado #>> '{fiscalizacion,resultado}' <> 'favorable'
               AND pg_catalog.char_length(
                   COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
               ) = 0) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_fiscalizacion := ROW(
            v_agregado #>> '{fiscalizacion,resultado}',
            CASE WHEN v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
                 THEN '' ELSE 'observaciones_fiscalizacion' END,
            COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', ''),
            (v_agregado #>> '{fiscalizacion,fiscalizada_en}')::timestamptz,
            NULL::timestamptz, NULL::text
        );
        IF v_agregado #>> '{fiscalizacion,resultado}' = 'desfavorable' THEN
            IF pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno}'
               ) <> 'object'
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno,retorno_ref}'
               ) <> 'string' THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            SELECT pg_catalog.count(*)
              INTO v_subsanaciones
              FROM pg_catalog.jsonb_array_elements(
                   v_agregado -> 'actuaciones'
              ) nodo
             WHERE nodo ->> 'accion_clave' =
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND nodo ->> 'retorno_ref' =
                   v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
            IF v_subsanaciones > 1 THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            IF v_subsanaciones = 1 THEN
                SELECT nodo INTO STRICT v_subsanacion_nodo
                  FROM pg_catalog.jsonb_array_elements(
                       v_agregado -> 'actuaciones'
                  ) nodo
                 WHERE nodo ->> 'accion_clave' =
                       'contratacion_temporal.subsanacion_reparos.registrar'
                   AND nodo ->> 'retorno_ref' =
                       v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
                IF pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'realizada_en'
                   ) <> 'string'
                   OR pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'observaciones'
                   ) <> 'string'
                   OR pg_catalog.char_length(
                       v_subsanacion_nodo ->> 'observaciones'
                   ) NOT BETWEEN 1 AND 2000
                   OR v_subsanacion_nodo ->> 'observaciones' ~ '[\x00-\x1F]' THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_referencia_subsanacion := (v_subsanacion_nodo ->>
                    'secuencia')::numeric(20, 0);
                IF v_referencia_subsanacion <= v_referencia_fiscalizacion
                   OR v_referencia_subsanacion > v_version_instruccion THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_subsanacion_presente := true;
                v_fiscalizacion.subsanacion_registrada_en :=
                    (v_subsanacion_nodo ->> 'realizada_en')::timestamptz;
                v_fiscalizacion.subsanacion_texto :=
                    v_subsanacion_nodo ->> 'observaciones';
            END IF;
        END IF;
    ELSE
        v_fiscalizacion := NULL;
    END IF;

    v_hitos := ARRAY[]::vec_contratacion_temporal
        .hito_expediente_rrhh_v1[];
    v_total := pg_catalog.jsonb_array_length(
        v_agregado -> 'actuaciones'
    );
    IF v_total < 1 OR v_total::numeric + (CASE WHEN v_instruccion IS NULL THEN 0 ELSE 5 END) <> v_fila.version THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;
    FOR v_indice IN 0..v_total - 1 LOOP
        v_nodo := (v_agregado -> 'actuaciones') -> v_indice;
        v_hitos := pg_catalog.array_append(
            v_hitos,
            ROW(
                (v_nodo ->> 'secuencia')::numeric(20, 0),
                (v_nodo ->> 'version_expediente')::numeric(20, 0),
                v_nodo ->> 'accion_clave',
                (v_nodo ->> 'realizada_en')::timestamptz,
                COALESCE(v_nodo ->> 'fase_origen', ''),
                v_nodo ->> 'fase_destino',
                v_nodo ->> 'estado_origen',
                v_nodo ->> 'estado_destino'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        );
    END LOOP;

    -- La propuesta se lee dentro del detalle ya autorizado; nunca desde CT91.
    IF v_instruccion IS NOT NULL AND v_fila.version>=7 THEN
        v_hito_propuesta:=v_agregado->'actuaciones'->1;
        v_hay_hito:=v_hito_propuesta->>'secuencia'='7'
          AND v_hito_propuesta->>'version_expediente'='7'
          AND v_hito_propuesta->>'accion_clave'='registrar_propuesta_formalizacion';
        SELECT * INTO v_propuesta FROM vec_contratacion_temporal.propuesta_formalizacion p
         WHERE p.derivacion_ref=v_derivacion.derivacion_ref
           AND p.organizacion_ref=v_fila.organizacion_ref
           AND p.expediente_ref=v_fila.expediente_ref;
        v_hay_propuesta:=FOUND;
        IF v_hay_propuesta AND EXISTS(SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
            WHERE x.propuesta_anterior_ref=v_propuesta.propuesta_ref) THEN
            v_hay_propuesta:=false; v_hay_hito:=false;
        END IF;
        IF v_hay_propuesta IS DISTINCT FROM v_hay_hito THEN
            RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        IF v_hay_propuesta THEN
            IF v_propuesta.version_previa<>6 OR v_propuesta.version_resultante<>7
               OR v_propuesta.llamamiento_ref IS NOT NULL OR v_propuesta.resolucion_ref IS NOT NULL
               OR v_propuesta.recibo_aceptacion_ref IS NOT NULL
               OR v_propuesta.solicitud_json->>'DerivacionRef' IS DISTINCT FROM v_derivacion.derivacion_ref
               OR v_propuesta.material_json->'AceptacionDerivada' IS DISTINCT FROM v_propuesta.aceptacion_bolsa_json
               OR v_propuesta.material_sha256 IS DISTINCT FROM
                    encode(sha256(convert_to(v_propuesta.material,'UTF8')),'hex')
               OR v_propuesta.recibo_json->>'Estado' IS DISTINCT FROM 'confirmado'
               OR v_propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM v_propuesta.propuesta_ref
               OR v_propuesta.recibo_json->>'ReciboLocalRef' IS DISTINCT FROM v_propuesta.recibo_ref
               OR v_propuesta.recibo_json->'Solicitud' IS DISTINCT FROM v_propuesta.solicitud_json
               OR v_propuesta.recibo_json->'VersionResultante' IS DISTINCT FROM '7'::jsonb
               OR v_propuesta.recibo_json->>'ConfirmadaEn' IS DISTINCT FROM
                    to_char(v_propuesta.confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
               OR v_propuesta.confirmada_en IS DISTINCT FROM
                    (v_hito_propuesta->>'realizada_en')::timestamptz
               OR v_propuesta.recibo_ref IS DISTINCT FROM v_hito_propuesta->>'recibo_ref'
               OR v_propuesta.aceptacion_bolsa_json IS DISTINCT FROM jsonb_build_object(
                    'DerivacionRef',v_derivacion.derivacion_ref,'PadreRef',v_derivacion.padre_ref,
                    'HijoRef',v_derivacion.hijo_ref,'OfertaRef',v_derivacion.oferta_ref,
                    'NumeroPlaza',v_derivacion.numero_plaza,'AceptacionRef',v_derivacion.aceptacion_ref,
                    'ReciboDerivacionRef',v_derivacion.recibo_ref)
               OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral v7
                    JOIN vec_contratacion_temporal.actuacion_expediente_integral acto7
                      ON acto7.expediente_ref=v7.expediente_ref AND acto7.secuencia=7
                    WHERE v7.expediente_ref=v_fila.expediente_ref AND v7.version=7
                      AND v7.origen_version='propuesta_formalizacion_o6'
                      AND v7.operacion_ref=v_propuesta.propuesta_ref
                      AND v7.agregado_json_huella_sha256=encode(sha256(convert_to(v7.agregado_json::text,'UTF8')),'hex')
                      AND v7.agregado_json->'actuaciones'->1 IS NOT DISTINCT FROM v_hito_propuesta
                      AND acto7.actuacion_json IS NOT DISTINCT FROM v_hito_propuesta
                      AND acto7.operacion_ref=v_propuesta.propuesta_ref
                      AND acto7.actuacion_json_huella_sha256=encode(sha256(convert_to(acto7.actuacion_json::text,'UTF8')),'hex')
                      AND acto7.recibo_ref=v_propuesta.recibo_ref)
               OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral evento
                    WHERE evento.evento_ref=v_propuesta.evento_ref
                      AND evento.operacion_ref=v_propuesta.propuesta_ref
                      AND evento.expediente_ref=v_fila.expediente_ref
                      AND evento.version_expediente=7
                      AND evento.tipo_evento='contratacion_temporal.propuesta_formalizacion_registrada'
                      AND evento.payload_huella_sha256=encode(sha256(evento.payload_canonico),'hex')
                      AND (convert_from(evento.payload_canonico,'UTF8')::jsonb)->>'propuesta_ref'=v_propuesta.propuesta_ref
                      AND (convert_from(evento.payload_canonico,'UTF8')::jsonb)->>'derivacion_ref'=v_derivacion.derivacion_ref
                      AND (convert_from(evento.payload_canonico,'UTF8')::jsonb)->>'recibo_ref'=v_propuesta.recibo_ref)
            THEN RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
            v_propuesta_publica:=jsonb_build_object('propuesta_ref',v_propuesta.propuesta_ref,
              'recibo_ref',v_propuesta.recibo_ref,'version',7,
              'confirmada_en',to_char(v_propuesta.confirmada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
        END IF;
    END IF;

    v_detalle := ROW(
        v_resumen,
        v_solicitud,
        v_analisis_presente,
        v_analisis,
        v_referencia_analisis,
        v_cobertura_presente,
        v_cobertura,
        v_referencia_cobertura,
        v_asignacion_presente,
        v_asignacion,
        v_referencia_asignacion,
        v_hitos,
        v_fiscalizacion_presente,
        v_fiscalizacion,
        v_referencia_fiscalizacion,
        v_referencia_subsanacion,
        v_instruccion,
        v_propuesta_publica
    );

    -- Reutiliza la validación del canon nominal sin volver a consultar ni
    -- conservar el agregado. El instante de la versión basta para esta
    -- validación estructural; el motor canoniza después su instante real.
    PERFORM
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resumen.actualizado_en,
            v_detalle
        );
    RETURN ROW(v_detalle)::
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
END
$function$

;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(p_generada_en timestamp with time zone, p_entrada vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)
 RETURNS bytea
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path=pg_catalog,pg_temp
AS $function$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            (CASE WHEN p_entrada.propuesta_formalizacion IS NOT NULL THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V6'
                  WHEN p_entrada.instruccion_compartida IS NOT NULL THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V5'
                  WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                       OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V4'
                ELSE 'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' END)
                || pg_catalog.chr(10)
        );
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_hito vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_anterior vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_comprobacion
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1;
    v_campos text[];
    v_campo text;
    v_claves_vistas text[] := ARRAY[]::text[];
    v_indice integer;
    v_total_hitos integer;
    v_total_instruccion integer;
    v_version_instruccion numeric(20,0);
    v_hitos_instruccion vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_instruccion jsonb := p_entrada.instruccion_compartida;
    v_derivado boolean := p_entrada.instruccion_compartida IS NOT NULL;
    v_total_comprobaciones integer;
    v_mascara smallint := 0;
    v_analisis_nulo boolean;
    v_cobertura_nula boolean;
    v_asignacion_nula boolean;
    v_fiscalizacion_nula boolean;
    v_subsanacion_presente boolean;
    v_utc timestamp;
BEGIN
    IF p_entrada.resumen IS NOT DISTINCT FROM NULL
       OR p_entrada.solicitud IS NOT DISTINCT FROM NULL
       OR p_entrada.analisis_presente IS NULL
       OR p_entrada.referencia_analisis IS NULL
       OR p_entrada.cobertura_presente IS NULL
       OR p_entrada.referencia_cobertura IS NULL
       OR p_entrada.asignacion_presente IS NULL
       OR p_entrada.referencia_asignacion IS NULL
       OR p_entrada.hitos IS NULL
       OR p_entrada.fiscalizacion_presente IS NULL
       OR p_entrada.referencia_fiscalizacion IS NULL
       OR p_entrada.referencia_subsanacion IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_resumen := p_entrada.resumen;
    v_solicitud := p_entrada.solicitud;
    v_analisis_nulo := p_entrada.analisis IS NOT DISTINCT FROM NULL;
    v_cobertura_nula := p_entrada.cobertura IS NOT DISTINCT FROM NULL;
    v_asignacion_nula := p_entrada.asignacion IS NOT DISTINCT FROM NULL;
    v_fiscalizacion_nula := p_entrada.fiscalizacion IS NOT DISTINCT FROM NULL;

    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        p_generada_en
    );
    v_canon := v_canon
        || vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
            v_resumen
        );
    IF p_generada_en < v_resumen.actualizado_en
       OR v_solicitud.grupo_subgrupo IS NULL
       OR v_solicitud.grupo_subgrupo !~
          '^[A-Z][A-Z0-9/+.-]{0,19}$'
       OR v_solicitud.motivo_clave IS NULL
       OR v_solicitud.motivo_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR v_solicitud.periodo_inicio IS NULL
       OR (v_solicitud.periodo_fin IS NULL
           AND coalesce(v_solicitud.periodo_causa_fin,'') !~
               '^[a-z][a-z0-9._-]{1,79}$')
       OR (v_solicitud.periodo_fin IS NOT NULL
           AND v_solicitud.periodo_causa_fin IS NOT NULL)
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_inicio
    );
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_fin
    );
    v_utc := v_solicitud.periodo_inicio AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_utc := v_solicitud.periodo_fin AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    v_total_hitos := pg_catalog.cardinality(p_entrada.hitos);
    IF v_total_hitos < 1
       OR v_total_hitos::numeric + (CASE WHEN v_derivado THEN 5 ELSE 0 END) <> v_resumen.version
       OR pg_catalog.array_ndims(p_entrada.hitos) <> 1
       OR pg_catalog.array_lower(p_entrada.hitos, 1) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF pg_catalog.array_position(p_entrada.hitos, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_hitos_instruccion := p_entrada.hitos;
    v_total_instruccion := v_total_hitos;
    v_version_instruccion := v_resumen.version;
    IF v_derivado THEN
        IF pg_catalog.jsonb_typeof(v_instruccion)<>'object'
           OR NOT (p_entrada.analisis_presente AND p_entrada.cobertura_presente
                   AND p_entrada.asignacion_presente AND p_entrada.fiscalizacion_presente)
           OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(v_instruccion))<>10
           OR pg_catalog.jsonb_typeof(v_instruccion->'hitos_padre')<>'array'
           OR COALESCE(v_instruccion->>'padre_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'recibo_derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'oferta_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'huella_padre_sha256','') !~ '^[0-9a-f]{64}$'
           OR v_instruccion->>'huella_padre_sha256'=pg_catalog.repeat('0',64)
           OR COALESCE(v_instruccion->>'numero_plaza','') !~ '^(100|[1-9][0-9]?)$'
           OR v_resumen.numero_visible IS DISTINCT FROM
              (v_instruccion->>'numero_visible_padre')||'-P'||(v_instruccion->>'numero_plaza')
           OR v_instruccion->>'padre_ref' = v_resumen.expediente_ref
           OR v_instruccion->>'creado_en_padre' IS NULL
           OR v_instruccion->>'actualizado_en_padre' IS NULL
           OR (v_instruccion->>'creado_en_padre')::timestamptz >
              (v_instruccion->>'actualizado_en_padre')::timestamptz
           OR (v_instruccion->>'actualizado_en_padre')::timestamptz > v_resumen.creado_en
        THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
        SELECT pg_catalog.array_agg(ROW(
          (nodo->>'secuencia')::numeric(20,0),
          (nodo->>'version_expediente')::numeric(20,0),
          nodo->>'accion_clave',(nodo->>'realizada_en')::timestamptz,
          COALESCE(nodo->>'fase_origen',''),nodo->>'fase_destino',
          nodo->>'estado_origen',nodo->>'estado_destino')::
          vec_contratacion_temporal.hito_expediente_rrhh_v1 ORDER BY ord)
          INTO v_hitos_instruccion
          FROM pg_catalog.jsonb_array_elements(v_instruccion->'hitos_padre')
            WITH ORDINALITY AS original(nodo,ord);
        v_total_instruccion := pg_catalog.cardinality(v_hitos_instruccion);
        v_version_instruccion := v_total_instruccion;
        IF v_total_instruccion<6 OR v_total_instruccion>1500 OR
           pg_catalog.array_position(v_hitos_instruccion,NULL) IS NOT NULL THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis_nulo
           OR p_entrada.referencia_analisis
              NOT BETWEEN 2 AND v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 1;
        v_analisis := p_entrada.analisis;
    ELSIF NOT v_analisis_nulo OR p_entrada.referencia_analisis <> 0
          OR v_resumen.modalidad_clave <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.cobertura_presente THEN
        IF v_cobertura_nula OR NOT p_entrada.analisis_presente
           OR p_entrada.referencia_cobertura
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_cobertura
              <= p_entrada.referencia_analisis THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 2;
        v_cobertura := p_entrada.cobertura;
    ELSIF NOT v_cobertura_nula OR p_entrada.referencia_cobertura <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.asignacion_presente THEN
        IF v_asignacion_nula OR NOT p_entrada.cobertura_presente
           OR p_entrada.referencia_asignacion
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_asignacion
              <= p_entrada.referencia_cobertura THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 4;
        v_asignacion := p_entrada.asignacion;
    ELSIF NOT v_asignacion_nula OR p_entrada.referencia_asignacion <> 0
          OR v_resumen.unidad_ref <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.fiscalizacion_presente THEN
        IF v_fiscalizacion_nula OR NOT p_entrada.asignacion_presente
           OR p_entrada.referencia_fiscalizacion
              NOT BETWEEN 1 AND v_version_instruccion
           OR p_entrada.referencia_fiscalizacion
              <= p_entrada.referencia_asignacion
           OR p_entrada.referencia_subsanacion < 0
           OR p_entrada.referencia_subsanacion > v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 8;
        v_fiscalizacion := p_entrada.fiscalizacion;
    ELSIF NOT v_fiscalizacion_nula
          OR p_entrada.referencia_fiscalizacion <> 0
          OR p_entrada.referencia_subsanacion <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF v_mascara NOT IN (0, 1, 3, 7, 15) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis.modalidad_clave IS NULL
           OR v_analisis.modalidad_clave !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.modalidad_clave <> v_resumen.modalidad_clave
           OR v_analisis.categoria_ref IS NULL
           OR v_analisis.categoria_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_analisis.categoria_ref <> v_resumen.categoria_ref
           OR v_analisis.causa_clave IS NULL
           OR v_analisis.causa_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.periodo_inicio IS NULL
           OR (v_analisis.periodo_fin IS NULL
               AND coalesce(v_analisis.periodo_causa_fin,'') !~
                   '^[a-z][a-z0-9._-]{1,79}$')
           OR (v_analisis.periodo_fin IS NOT NULL
               AND v_analisis.periodo_causa_fin IS NOT NULL)
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio
           OR v_analisis.porcentaje_jornada IS NULL
           OR v_analisis.porcentaje_jornada NOT BETWEEN 1 AND 10000
           OR v_analisis.resultado_rc IS NULL
           OR v_analisis.resultado_rc NOT IN (
               'validada', 'no_requerida', 'rechazada'
           )
           OR v_analisis.coste_presente IS NULL
           OR v_analisis.coste_centimos IS NULL
           OR v_analisis.coste_moneda IS NULL
           OR v_analisis.fuente_coste_ref IS NULL
           OR v_analisis.observaciones IS NULL
           OR pg_catalog.length(v_analisis.observaciones) > 4000 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_inicio
        );
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_fin
        );
        v_utc := v_analisis.periodo_inicio AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_utc := v_analisis.periodo_fin AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        IF v_analisis.coste_presente THEN
            IF v_analisis.coste_centimos <= 0
               OR v_analisis.coste_moneda <> 'EUR'
               OR v_analisis.fuente_coste_ref !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_analisis.coste_centimos <> 0
              OR v_analisis.coste_moneda <> ''
              OR v_analisis.fuente_coste_ref <> '' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END IF;

    IF p_entrada.cobertura_presente THEN
        IF v_cobertura.via_clave IS NULL
           OR v_cobertura.via_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_cobertura.decision_gobernada IS NULL
           OR v_cobertura.procedimiento_ref IS NULL
           OR v_cobertura.bolsa_ref IS NULL
           OR v_cobertura.comprobaciones IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_total_comprobaciones :=
            pg_catalog.cardinality(v_cobertura.comprobaciones);
        IF v_cobertura.decision_gobernada THEN
            IF v_cobertura.procedimiento_ref <> ''
               OR v_cobertura.bolsa_ref <> '' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            IF v_total_comprobaciones = 0 THEN
                IF pg_catalog.array_ndims(v_cobertura.comprobaciones)
                   IS NOT NULL THEN
                    RAISE EXCEPTION USING ERRCODE = '22023',
                        MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
            ELSIF v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(v_cobertura.comprobaciones) <> 1
               OR pg_catalog.array_lower(v_cobertura.comprobaciones, 1) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSE
            IF v_cobertura.procedimiento_ref !~
               '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_cobertura.bolsa_ref !~
                  '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(
                   v_cobertura.comprobaciones
               ) <> 1
               OR pg_catalog.array_lower(
                   v_cobertura.comprobaciones, 1
               ) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_total_comprobaciones > 0 THEN
            IF pg_catalog.array_position(v_cobertura.comprobaciones, NULL) IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            FOR v_indice IN 1..v_total_comprobaciones LOOP
                v_comprobacion := v_cobertura.comprobaciones[v_indice];
                IF v_comprobacion.clave IS NULL OR v_comprobacion.clave !~ '^[a-z][a-z0-9._-]{1,79}$'
                   OR v_comprobacion.resultado IS NULL OR v_comprobacion.resultado NOT IN ('afirmativa', 'negativa', 'no_aplica', 'no_consta')
                   OR v_comprobacion.clave = ANY(v_claves_vistas) THEN
                    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
                v_claves_vistas := pg_catalog.array_append(v_claves_vistas, v_comprobacion.clave);
            END LOOP;
        END IF;
    END IF;

    IF p_entrada.asignacion_presente THEN
        IF v_asignacion.unidad_ref IS NULL
           OR v_asignacion.unidad_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_asignacion.unidad_ref <> v_resumen.unidad_ref
           OR v_asignacion.asignada_en IS NULL
           OR v_asignacion.asignada_en < (CASE WHEN v_derivado THEN (v_instruccion->>'creado_en_padre')::timestamptz ELSE v_resumen.creado_en END)
           OR v_asignacion.asignada_en > v_resumen.actualizado_en
           OR v_asignacion.motivo_clave IS NULL
           OR v_asignacion.motivo_clave !~
              '^$|^[a-z][a-z0-9._-]{1,79}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_asignacion.asignada_en
        );
    END IF;

    FOR v_indice IN 1..v_total_instruccion LOOP
        v_hito := v_hitos_instruccion[v_indice];
        IF v_hito.secuencia IS NULL
           OR v_hito.secuencia <> v_indice
           OR v_hito.version_expediente IS NULL
           OR v_hito.version_expediente <> v_indice
           OR v_hito.accion_clave IS NULL
           OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.realizada_en IS NULL
           OR v_hito.realizada_en > v_resumen.actualizado_en
           OR v_hito.fase_origen IS NULL
           OR v_hito.fase_origen !~
              '^$|^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.fase_destino IS NULL
           OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.estado_origen IS NULL
           OR v_hito.estado_origen NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           )
           OR v_hito.estado_destino IS NULL
           OR v_hito.estado_destino NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_hito.realizada_en
        );
        IF p_entrada.fiscalizacion_presente
           AND v_indice = p_entrada.referencia_fiscalizacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.fiscalizacion.registrar'
               OR v_hito.realizada_en <> v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF p_entrada.fiscalizacion_presente
           AND p_entrada.referencia_subsanacion <> 0
           AND v_indice = p_entrada.referencia_subsanacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.subsanacion_reparos.registrar'
               OR v_hito.realizada_en <>
                  v_fiscalizacion.subsanacion_registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_indice = 1 THEN
            IF v_hito.fase_origen <> ''
               OR v_hito.estado_origen <> 'pendiente' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_hito.fase_origen <> v_anterior.fase_destino
              OR v_hito.estado_origen <> v_anterior.estado_destino
              OR v_hito.realizada_en < v_anterior.realizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_anterior := v_hito;
    END LOOP;
    IF (NOT v_derivado AND (v_anterior.fase_destino <> v_resumen.fase_clave
       OR v_anterior.estado_destino <> v_resumen.estado_clave
       OR v_anterior.realizada_en <> v_resumen.actualizado_en))
       OR (v_derivado AND (v_anterior.fase_destino <> 'fiscalizacion'
       OR v_anterior.estado_destino <> 'en_curso'
       OR v_anterior.realizada_en <>
           (v_instruccion->>'actualizado_en_padre')::timestamptz
       OR v_hitos_instruccion[1].realizada_en <>
           (v_instruccion->>'creado_en_padre')::timestamptz))
       OR (p_entrada.asignacion_presente AND
           (v_hitos_instruccion[p_entrada.referencia_asignacion::integer]).realizada_en
           <> v_asignacion.asignada_en) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF v_derivado THEN
        v_anterior := NULL;
        FOR v_indice IN 1..v_total_hitos LOOP
            v_hito := p_entrada.hitos[v_indice];
            IF v_hito.secuencia IS DISTINCT FROM v_indice+5
               OR v_hito.version_expediente IS DISTINCT FROM v_indice+5
               OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.realizada_en IS NULL
               OR v_hito.realizada_en > v_resumen.actualizado_en
               OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.estado_origen NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
               OR v_hito.estado_destino NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
            THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en);
            IF v_indice=1 THEN
                IF v_hito.accion_clave<>'contratacion_temporal.nombramiento.derivar'
                   OR v_hito.fase_origen<>'fiscalizacion' OR v_hito.estado_origen<>'en_curso'
                   OR v_hito.fase_destino<>'fiscalizacion' OR v_hito.estado_destino<>'en_curso'
                   OR v_hito.realizada_en<>v_resumen.creado_en
                THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            ELSIF v_hito.fase_origen<>v_anterior.fase_destino
                OR v_hito.estado_origen<>v_anterior.estado_destino
                OR v_hito.realizada_en<v_anterior.realizada_en THEN
                RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023';
            END IF;
            v_anterior:=v_hito;
        END LOOP;
        IF v_anterior.fase_destino<>v_resumen.fase_clave
           OR v_anterior.estado_destino<>v_resumen.estado_clave
           OR v_anterior.realizada_en<>v_resumen.actualizado_en THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    v_campos := ARRAY[
        v_solicitud.grupo_subgrupo,
        v_solicitud.motivo_clave,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_inicio
        ),
        CASE WHEN v_solicitud.periodo_fin IS NOT NULL THEN
            (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                    OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'fecha:' ELSE '' END) ||
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_solicitud.periodo_fin
            )
        ELSE 'causa:' || v_solicitud.periodo_causa_fin END,
        v_mascara::text
    ]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.analisis_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_analisis::text
    ]::text[];
    IF p_entrada.analisis_presente THEN
        v_campos := v_campos || ARRAY[
            v_analisis.modalidad_clave, v_analisis.categoria_ref,
            v_analisis.causa_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_inicio
            ),
            CASE WHEN v_analisis.periodo_fin IS NOT NULL THEN
                (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                        OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                    THEN 'fecha:' ELSE '' END) ||
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_analisis.periodo_fin
                )
            ELSE 'causa:' || v_analisis.periodo_causa_fin END,
            v_analisis.porcentaje_jornada::text,
            v_analisis.resultado_rc,
            CASE WHEN v_analisis.coste_presente THEN '1' ELSE '0' END
        ]::text[];
        IF v_analisis.coste_presente THEN
            v_campos := v_campos || ARRAY[
                v_analisis.coste_centimos::text,
                v_analisis.coste_moneda
            ]::text[];
        END IF;
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.fuente_coste_ref
        );
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.observaciones
        );
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.cobertura_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_cobertura::text
    ]::text[];
    IF p_entrada.cobertura_presente THEN
        v_campos := v_campos || ARRAY[
            v_cobertura.via_clave,
            CASE WHEN v_cobertura.decision_gobernada THEN '1' ELSE '0' END,
            v_cobertura.procedimiento_ref, v_cobertura.bolsa_ref,
            v_total_comprobaciones::text
        ]::text[];
        FOR v_indice IN 1..v_total_comprobaciones LOOP
            v_comprobacion := v_cobertura.comprobaciones[v_indice];
            v_campos := v_campos || ARRAY[
                v_comprobacion.clave, v_comprobacion.resultado
            ]::text[];
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.asignacion_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_asignacion::text
    ]::text[];
    IF p_entrada.asignacion_presente THEN
        v_campos := v_campos || ARRAY[
            v_asignacion.unidad_ref,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_asignacion.asignada_en
            ),
            v_asignacion.motivo_clave
        ]::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_campos := ARRAY[
            '1', p_entrada.referencia_fiscalizacion::text,
            p_entrada.referencia_subsanacion::text,
            v_fiscalizacion.resultado_clave,
            CASE WHEN v_fiscalizacion.reparo_clave = '' THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.reparo_clave <> '' THEN
            v_campos := v_campos || ARRAY[
                v_fiscalizacion.reparo_clave, v_fiscalizacion.reparo_texto
            ]::text[];
        END IF;
        v_campos := v_campos || ARRAY[
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.registrada_en
            ),
            CASE WHEN v_fiscalizacion.subsanacion_registrada_en IS NULL
                 THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.subsanacion_registrada_en IS NOT NULL THEN
            v_campos := v_campos || ARRAY[
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_fiscalizacion.subsanacion_registrada_en
                ), v_fiscalizacion.subsanacion_texto
            ]::text[];
        END IF;
    ELSE
        v_campos := ARRAY['0', '0', '0']::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[v_total_hitos::text]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_subsanacion_presente :=
            v_fiscalizacion.subsanacion_registrada_en IS NOT NULL
            OR v_fiscalizacion.subsanacion_texto IS NOT NULL;
        IF v_fiscalizacion.resultado_clave NOT IN (
               'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR v_fiscalizacion.reparo_clave IS NULL
           OR v_fiscalizacion.reparo_texto IS NULL
           OR v_fiscalizacion.registrada_en IS NULL
           OR (v_fiscalizacion.resultado_clave = 'favorable'
               AND (v_fiscalizacion.reparo_clave <> ''
                    OR v_fiscalizacion.reparo_texto <> ''))
           OR (v_fiscalizacion.resultado_clave <> 'favorable'
               AND (v_fiscalizacion.reparo_clave <> 'observaciones_fiscalizacion'
                    OR pg_catalog.char_length(v_fiscalizacion.reparo_texto)
                       NOT BETWEEN 1 AND 2000))
           OR (v_subsanacion_presente
               AND (v_fiscalizacion.resultado_clave <> 'desfavorable'
                    OR v_fiscalizacion.subsanacion_registrada_en IS NULL
                    OR v_fiscalizacion.subsanacion_texto IS NULL
                    OR pg_catalog.char_length(v_fiscalizacion.subsanacion_texto)
                       NOT BETWEEN 1 AND 2000
                    OR p_entrada.referencia_subsanacion
                       <= p_entrada.referencia_fiscalizacion))
           OR (NOT v_subsanacion_presente
               AND p_entrada.referencia_subsanacion <> 0) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_fiscalizacion.registrada_en
        );
        IF v_subsanacion_presente THEN
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.subsanacion_registrada_en
            );
            IF v_fiscalizacion.subsanacion_registrada_en
               <= v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
    END IF;

    FOR v_indice IN 1..v_total_hitos LOOP
        v_hito := p_entrada.hitos[v_indice];
        v_campos := ARRAY[
            v_hito.secuencia::text,
            v_hito.version_expediente::text,
            v_hito.accion_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_hito.realizada_en
            ),
            v_hito.fase_origen, v_hito.fase_destino,
            v_hito.estado_origen, v_hito.estado_destino
        ]::text[];
        FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon := v_canon
                || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                    vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                        v_campo
                    )
                );
        END LOOP;
        IF pg_catalog.octet_length(v_canon) > 262144 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END LOOP;
    IF v_derivado THEN
        v_campos := ARRAY[
          v_instruccion->>'derivacion_ref',v_instruccion->>'padre_ref',
          v_instruccion->>'numero_visible_padre',v_instruccion->>'oferta_ref',
          v_instruccion->>'numero_plaza',v_instruccion->>'recibo_derivacion_ref',
          v_instruccion->>'huella_padre_sha256',
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'creado_en_padre')::timestamptz),
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'actualizado_en_padre')::timestamptz),
          v_total_instruccion::text
        ];
        FOREACH v_campo IN ARRAY v_campos LOOP
          v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
        END LOOP;
        FOR v_indice IN 1..v_total_instruccion LOOP
          v_hito:=v_hitos_instruccion[v_indice];
          v_campos:=ARRAY[v_hito.secuencia::text,v_hito.version_expediente::text,
            v_hito.accion_clave,vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en),
            v_hito.fase_origen,v_hito.fase_destino,v_hito.estado_origen,v_hito.estado_destino];
          FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
              vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
          END LOOP;
        END LOOP;
    END IF;
    IF p_entrada.propuesta_formalizacion IS NOT NULL THEN
        IF NOT v_derivado OR v_total_hitos<2
           OR jsonb_typeof(p_entrada.propuesta_formalizacion)<>'object'
           OR (SELECT count(*) FROM jsonb_object_keys(p_entrada.propuesta_formalizacion))<>4
           OR NOT (p_entrada.propuesta_formalizacion ?&
             ARRAY['propuesta_ref','recibo_ref','version','confirmada_en'])
           OR p_entrada.propuesta_formalizacion->>'propuesta_ref' !~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR p_entrada.propuesta_formalizacion->>'recibo_ref' !~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR p_entrada.propuesta_formalizacion->'version' IS DISTINCT FROM '7'::jsonb
           OR p_entrada.propuesta_formalizacion->>'confirmada_en' IS DISTINCT FROM
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                  (p_entrada.propuesta_formalizacion->>'confirmada_en')::timestamptz)
        THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
        v_hito:=p_entrada.hitos[2];
        IF v_hito.secuencia<>7 OR v_hito.version_expediente<>7
           OR v_hito.accion_clave<>'registrar_propuesta_formalizacion'
           OR v_hito.realizada_en IS DISTINCT FROM
                (p_entrada.propuesta_formalizacion->>'confirmada_en')::timestamptz
           OR p_generada_en<v_hito.realizada_en
        THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
        v_campos:=ARRAY[p_entrada.propuesta_formalizacion->>'propuesta_ref',
          p_entrada.propuesta_formalizacion->>'recibo_ref',
          p_entrada.propuesta_formalizacion->>'version',
          p_entrada.propuesta_formalizacion->>'confirmada_en'];
        FOREACH v_campo IN ARRAY v_campos LOOP
          v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
        END LOOP;
    END IF;
    IF pg_catalog.octet_length(v_canon)>262144 THEN
      RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'contenido de detalle RRHH inválido';
END
$function$

;

DO $post_funciones$
BEGIN
 IF (SELECT count(*) FROM pg_temp.ct236_pre)<>6
    OR EXISTS(SELECT 1 FROM pg_temp.ct236_pre antes
      LEFT JOIN pg_proc despues ON despues.oid=antes.oid
      WHERE despues.oid IS NULL OR to_regprocedure(antes.firma) IS DISTINCT FROM antes.oid
        OR despues.proowner IS DISTINCT FROM antes.propietario
        OR despues.proacl IS DISTINCT FROM antes.acl
        OR despues.prosecdef IS DISTINCT FROM antes.definidor
        OR despues.provolatile IS DISTINCT FROM antes.volatilidad
        OR despues.proparallel IS DISTINCT FROM antes.paralelo
        OR encode(sha256(convert_to(despues.prosrc,'UTF8')),'hex') IS DISTINCT FROM antes.posthash
        OR cardinality(despues.proconfig) IS DISTINCT FROM cardinality(antes.config_final)
        OR EXISTS(SELECT 1 FROM unnest(antes.config_final) cfg WHERE cfg<>ALL(despues.proconfig)))
 THEN RAISE EXCEPTION 'CT236: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post_funciones$;
-- CT236-GENERADO-FIN
COMMIT;
