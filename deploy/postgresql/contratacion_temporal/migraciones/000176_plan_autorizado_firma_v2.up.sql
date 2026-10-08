\set ON_ERROR_STOP on
-- CT176: registro de firma V2 con plan nominal fijado. Requiere AD178, AD177 y
-- CC7 instaladas (y CT172). Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000176',0));
DO $pre$
DECLARE nombre text; f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR getdatabaseencoding()<>'UTF8'
    OR to_regclass('vec_contratacion_temporal.firma_documento_plan_v2') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_revision_pdf_v2') IS NULL
    OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
    OR EXISTS(SELECT 1 FROM pg_roles r
      WHERE r.rolname='vec_contratacion_temporal_propietario' AND r.rolcanlogin) THEN
  RAISE EXCEPTION 'CT176 preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea)',
  'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)',
  'vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[])',
  'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'] LOOP
  f:=to_regprocedure(nombre);
  IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN
   RAISE EXCEPTION 'CT176 dependencia no disponible: %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger t
   WHERE t.tgrelid='vec_contratacion_temporal.firma_documento_v1'::regclass
     AND t.tgname='hija_pdf_v2_ai' AND NOT t.tgisinternal) THEN
  RAISE EXCEPTION 'CT176 guarda CT172 no disponible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La hija conserva la primera confirmación. Una autorización nueva para un
-- replay consume de nuevo AD, pero nunca sustituye estas referencias.
CREATE TABLE vec_contratacion_temporal.firma_documento_plan_v2(
 firma_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.firma_documento_v1(firma_ref),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.firma_documento_v1(recibo_ref),
 solicitud_huella_sha256 text NOT NULL CHECK(solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
 descriptor_huella_sha256 text NOT NULL CHECK(descriptor_huella_sha256 ~ '^[0-9a-f]{64}$'),
 envoltorio_original bytea NOT NULL CHECK(octet_length(envoltorio_original) BETWEEN 2 AND 65536),
 envoltorio_huella_sha256 text NOT NULL CHECK(envoltorio_huella_sha256=encode(sha256(envoltorio_original),'hex')),
 decision_interior_sha256 text NOT NULL CHECK(decision_interior_sha256 ~ '^[0-9a-f]{64}$'),
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$'),
 catalogo_version bigint NOT NULL CHECK(catalogo_version BETWEEN 1 AND 2147483647),
 publicacion_sha256 text NOT NULL CHECK(publicacion_sha256 ~ '^[0-9a-f]{64}$'),
 entrada_clave text NOT NULL CHECK(entrada_clave ~ '^[a-z][a-z0-9._-]{2,127}$'),
 entrada_original jsonb NOT NULL CHECK(jsonb_typeof(entrada_original)='object'),
 revision_control bigint NOT NULL CHECK(revision_control>=1),
 decision_interior_ref text NOT NULL,
 consumo_interior_huella_sha256 text NOT NULL CHECK(consumo_interior_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_interior_ref text NOT NULL,
 decision_exterior_ref text NOT NULL,
 consumo_exterior_huella_sha256 text NOT NULL CHECK(consumo_exterior_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_exterior_ref text NOT NULL,
 confirmada_en timestamptz(6) NOT NULL CHECK(isfinite(confirmada_en))
);
ALTER TABLE vec_contratacion_temporal.firma_documento_plan_v2 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.firma_documento_plan_v2 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.firma_documento_plan_v2
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_plan_v2 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_plan_v2 FROM PUBLIC;
CREATE TRIGGER firma_plan_v2_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_contratacion_temporal.firma_documento_plan_v2 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Prospectiva: sólo los nuevos INSERT V2 quedan bajo esta guarda. La historia
-- CT172 ya existente no se rellena ni se reescribe.
CREATE FUNCTION vec_contratacion_temporal.comprobar_hija_plan_firma_v2() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF TG_OP<>'INSERT' OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal'
    OR TG_TABLE_NAME<>'firma_documento_v1' THEN
  RAISE EXCEPTION 'CT176 guarda de plan denegada' USING ERRCODE='42501'; END IF;
 IF NEW.politica_verificacion='politica:vec:firma:verificacion-autonoma:v2'
    AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.firma_documento_plan_v2 p
      WHERE p.firma_ref=NEW.firma_ref AND p.recibo_ref=NEW.recibo_ref
        AND p.solicitud_huella_sha256=NEW.solicitud_huella_sha256
        AND p.confirmada_en=NEW.registrada_en) THEN
  RAISE EXCEPTION 'CT176 firma V2 sin plan autorizado' USING ERRCODE='23514'; END IF;
 RETURN NULL;
END $f$;
CREATE CONSTRAINT TRIGGER hija_plan_firma_v2_ai AFTER INSERT
 ON vec_contratacion_temporal.firma_documento_v1
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION vec_contratacion_temporal.comprobar_hija_plan_firma_v2();

CREATE FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v2(
 p_solicitud text,p_comprobada_en timestamptz,p_envoltorio bytea,
 p_cap_i bytea,p_dec_i bytea,p_mot_i bytea,p_ctx_i bytea,p_per_i numeric,p_prf_i numeric,
 p_pay_i bytea,p_sobre_i bytea,p_evid_i bytea,p_raiz_i bytea,
 p_cap_e bytea,p_dec_e bytea,p_mot_e bytea,p_ctx_e bytea,p_per_e numeric,p_prf_e numeric,
 p_pay_e bytea,p_sobre_e bytea,p_evid_e bytea,p_raiz_e bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE env_text text; env_json json; env jsonb; plan_json json; plan jsonb;
 desc_text text; descriptor_exacto bytea; d jsonb; s jsonb; a jsonb; decision_i jsonb; decision_e jsonb; pin record;
 exterior jsonb; interior jsonb; recibo jsonb; original record; revision record; hija record;
 h text; eh text; dh text; recurso text; contexto_i text; contexto_e text; esperado text;
 cat text; ver bigint; pub text; clave text; decision_h text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR pg_is_in_recovery() THEN
  RAISE EXCEPTION 'CT176 registro de firma con plan denegado' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
    OR p_envoltorio IS NULL OR octet_length(p_envoltorio) NOT BETWEEN 2 AND 65536
    OR p_dec_i IS NULL OR octet_length(p_dec_i) NOT BETWEEN 2 AND 524288
    OR p_ctx_i IS DISTINCT FROM p_ctx_e
    OR p_per_i IS DISTINCT FROM p_per_e OR p_prf_i IS DISTINCT FROM p_prf_e THEN
  RAISE EXCEPTION 'CT176 materiales ligados inválidos' USING ERRCODE='42501'; END IF;
 BEGIN
  env_text:=convert_from(p_envoltorio,'UTF8');
  env_json:=env_text::json;
  env:=env_text::jsonb;
  plan_json:=env_json->'plan';
  plan:=env->'plan';
  desc_text:=(env_json->'descriptor')::text;
  descriptor_exacto:=convert_to(desc_text,'UTF8');
  d:=desc_text::jsonb;
  s:=p_solicitud::jsonb;
  decision_i:=convert_from(p_dec_i,'UTF8')::jsonb;
  decision_e:=convert_from(p_dec_e,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'CT176 material JSON inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(env) IS DISTINCT FROM 'object'
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(env,ARRAY[
      'esquema','descriptor','plan','decision_interior_sha256']) IS NOT TRUE
    OR (SELECT count(*) FROM json_each(env_json))<>4
    OR env->>'esquema' IS DISTINCT FROM 'ct.plan-autorizado-firma.v2'
    OR jsonb_typeof(plan) IS DISTINCT FROM 'object'
    OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(plan,ARRAY[
      'catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']) IS NOT TRUE
    OR (SELECT count(*) FROM json_each(plan_json))<>4
    OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR octet_length(descriptor_exacto) NOT BETWEEN 2 AND 32768
    OR jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR jsonb_typeof(decision_i) IS DISTINCT FROM 'object'
    OR jsonb_typeof(decision_e) IS DISTINCT FROM 'object'
    OR jsonb_typeof(decision_i->'valida_hasta') IS DISTINCT FROM 'string'
    OR jsonb_typeof(decision_e->'valida_hasta') IS DISTINCT FROM 'string'
    OR jsonb_typeof(plan->'catalogo_id') IS DISTINCT FROM 'string'
    OR (plan->>'catalogo_id' ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR jsonb_typeof(plan->'catalogo_version') IS DISTINCT FROM 'number'
    OR (plan->>'catalogo_version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR (plan->>'catalogo_version')::numeric>2147483647
    OR jsonb_typeof(plan->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
    OR (plan->>'catalogo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR jsonb_typeof(plan->'entrada_clave') IS DISTINCT FROM 'string'
    OR (plan->>'entrada_clave' ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR jsonb_typeof(env->'decision_interior_sha256') IS DISTINCT FROM 'string'
    OR (env->>'decision_interior_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT176 envoltorio de plan inválido' USING ERRCODE='22023'; END IF;
 cat:=plan->>'catalogo_id'; ver:=(plan->>'catalogo_version')::bigint;
 pub:=plan->>'catalogo_huella_sha256'; clave:=plan->>'entrada_clave';
 decision_h:=encode(sha256(p_dec_i),'hex');
 -- La envoltura Go tiene este orden y el descriptor es el valor original.
 -- Las cuatro cadenas interpoladas están restringidas a ASCII seguro.
 esperado:='{"esquema":"ct.plan-autorizado-firma.v2","descriptor":'||desc_text||
  ',"plan":{"catalogo_id":"'||cat||'","catalogo_version":'||ver::text||
  ',"catalogo_huella_sha256":"'||pub||'","entrada_clave":"'||clave||
  '"},"decision_interior_sha256":"'||decision_h||'"}';
 IF env_text IS DISTINCT FROM esperado
    OR env->>'decision_interior_sha256' IS DISTINCT FROM decision_h THEN
  RAISE EXCEPTION 'CT176 envoltorio y decisión interior divergentes' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex');
 eh:=encode(sha256(p_envoltorio),'hex');
 dh:=encode(sha256(descriptor_exacto),'hex');
 IF (s->>'OrganizacionRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
    OR (s->>'ExpedienteRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
    OR (s->>'ClaveIdempotencia' ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$') IS NOT TRUE
    OR s->>'Via' NOT IN ('certificado_vec','portafirmas_registro_rrhh') THEN
  RAISE EXCEPTION 'CT176 solicitud no ligada' USING ERRCODE='22023'; END IF;
 recurso:=CASE WHEN s->>'Via'='certificado_vec' THEN 'operacion-firma-vec-ct:'
               ELSE 'operacion-firma-externa-ct:' END ||(s->>'ClaveIdempotencia');
 contexto_i:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"descriptor_firma_sha256":"'||dh||'","material_sha256":"'||h||'"}}','UTF8')),'hex');
 contexto_e:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||h||'","plan_firma_sha256":"'||eh||'"}}','UTF8')),'hex');

 -- AD consume primero la decisión exterior y coteja actor/perfil/acción/
 -- recurso con la decisión interior exacta, no con campos de este JSON.
 exterior:=vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(
  p_solicitud,p_envoltorio,p_dec_i,p_cap_e,p_dec_e,p_mot_e,p_ctx_e,p_per_e,p_prf_e,
  p_pay_e,p_sobre_e,p_evid_e,p_raiz_e);
 IF jsonb_typeof(exterior) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(exterior))<>7
    OR NOT (exterior ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
      'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
    OR exterior->>'efecto_ref' IS DISTINCT FROM recurso
    OR exterior->>'huella_efecto_sha256' IS DISTINCT FROM contexto_e
    OR exterior->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb THEN
  RAISE EXCEPTION 'CT176 consumo exterior divergente' USING ERRCODE='42501'; END IF;
 recibo:=vec_contratacion_temporal.registrar_firma_verificada_v2(
  p_solicitud,p_comprobada_en,p_cap_i,p_dec_i,p_mot_i,p_ctx_i,p_per_i,p_prf_i,
  p_pay_i,p_sobre_i,p_evid_i,p_raiz_i,descriptor_exacto);
 interior:=vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(p_dec_i);
 IF jsonb_typeof(interior) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(interior))<>7
    OR NOT (interior ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
      'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
    OR interior->>'efecto_ref' IS DISTINCT FROM recurso
    OR interior->>'huella_efecto_sha256' IS DISTINCT FROM contexto_i
    OR interior->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb THEN
  RAISE EXCEPTION 'CT176 consumo interior divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT pin FROM vec_catalogos_configurables.leer_plan_nominal_firma_v1(
  cat,ver,pub,clave,interior);
 a:=pin.entrada->'atributos';
 IF pin.publicacion_sha256 IS DISTINCT FROM pub
    OR pin.documento_exacto IS NULL
    OR encode(sha256(pin.documento_exacto),'hex') IS DISTINCT FROM pub
    OR pin.entrada->>'clave' IS DISTINCT FROM clave
    OR pin.revision_control IS NULL OR pin.revision_control<1
    OR jsonb_typeof(a) IS DISTINCT FROM 'object'
    OR a->>'esquema' IS DISTINCT FROM 'ct.plan-competencia-firma.v2'
    OR a->>'circuito_ref' IS DISTINCT FROM s->>'CatalogoRef'
    OR a->>'circuito_version' IS DISTINCT FROM s->>'CatalogoVersion'
    OR a->>'circuito_sha256' IS DISTINCT FROM s->>'CatalogoHuella'
    OR a->>'documento' IS DISTINCT FROM s->>'Documento'
    OR a->>'paso_ref' IS DISTINCT FROM s->>'PasoRef'
    OR a->>'paso_orden' IS DISTINCT FROM s->>'PasoOrden'
    OR a->>'perfil_esperado_ref' IS DISTINCT FROM d#>>'{seleccion,perfil_esperado_ref}'
    OR a->>'rol_id' IS DISTINCT FROM d#>>'{seleccion,rol_id}'
    OR a->>'cargo_ref' IS DISTINCT FROM d#>>'{seleccion,cargo_ref}'
    OR a->>'organizacion_ref' IS DISTINCT FROM s->>'OrganizacionRef'
    OR a->>'unidad_ref' IS DISTINCT FROM s->>'UnidadFirmanteRef'
    OR a->>'accion_competencial' IS DISTINCT FROM d->>'accion'
    OR a->>'finalidad' IS DISTINCT FROM d->>'finalidad'
    OR a->>'tipo_recurso' IS DISTINCT FROM d#>>'{recurso,tipo_recurso}' THEN
  RAISE EXCEPTION 'CT176 plan publicado divergente' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(recibo) IS DISTINCT FROM 'object'
    OR jsonb_typeof(recibo->'YaRegistrada') IS DISTINCT FROM 'boolean'
    OR (recibo->>'FirmaRef' ~ '^firma-ct:[0-9a-f-]{36}$') IS NOT TRUE
    OR (recibo->>'ReciboRef' ~ '^recibo-firma-ct:[0-9a-f-]{36}$') IS NOT TRUE
    OR recibo->>'SolicitudHuella' IS DISTINCT FROM h THEN
  RAISE EXCEPTION 'CT176 recibo CT172 incoherente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT original FROM vec_contratacion_temporal.firma_documento_v1 f
  WHERE f.firma_ref=recibo->>'FirmaRef' AND f.recibo_ref=recibo->>'ReciboRef'
  FOR KEY SHARE;
 SELECT * INTO STRICT revision FROM vec_contratacion_temporal.firma_documento_revision_pdf_v2 r
  WHERE r.firma_ref=original.firma_ref FOR KEY SHARE;
 IF original.organizacion_ref IS DISTINCT FROM s->>'OrganizacionRef'
    OR original.expediente_ref IS DISTINCT FROM s->>'ExpedienteRef'
    OR original.clave_idempotencia IS DISTINCT FROM s->>'ClaveIdempotencia'
    OR original.solicitud_huella_sha256 IS DISTINCT FROM h
    OR original.politica_verificacion IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
    OR original.actor_ref IS DISTINCT FROM recibo->>'ActorRef'
    OR original.perfil_ref IS DISTINCT FROM recibo->>'PerfilRef'
    OR original.registrada_en IS DISTINCT FROM (recibo->>'RegistradaEn')::timestamptz
    OR revision.descriptor_firma_original IS DISTINCT FROM descriptor_exacto
    OR revision.descriptor_firma_huella_sha256 IS DISTINCT FROM dh
    OR recibo->>'DocumentoCustodiaRef' IS DISTINCT FROM
      (SELECT c.documento_ref FROM vec_contratacion_temporal.firma_documento_custodia_v1 c
        WHERE c.firma_ref=original.firma_ref) THEN
  RAISE EXCEPTION 'CT176 efecto CT172 divergente' USING ERRCODE='55000'; END IF;
 IF recibo->'YaRegistrada'='false'::jsonb THEN
  IF original.decision_ref IS DISTINCT FROM interior->>'decision_ref'
     OR original.consumo_huella_sha256 IS DISTINCT FROM interior->>'consumo_huella_sha256'
     OR original.auditoria_consumo_ref IS DISTINCT FROM interior->>'auditoria_ref'
     OR revision.competencia_consumo_decision_ref IS DISTINCT FROM interior->>'decision_ref'
     OR revision.competencia_consumo_huella_sha256 IS DISTINCT FROM interior->>'consumo_huella_sha256'
     OR revision.competencia_auditoria_consumo_ref IS DISTINCT FROM interior->>'auditoria_ref' THEN
   RAISE EXCEPTION 'CT176 consumo interior no coincide con efecto nuevo' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_contratacion_temporal.firma_documento_plan_v2(
   firma_ref,recibo_ref,solicitud_huella_sha256,descriptor_huella_sha256,
   envoltorio_original,envoltorio_huella_sha256,decision_interior_sha256,
   catalogo_id,catalogo_version,publicacion_sha256,entrada_clave,entrada_original,revision_control,
   decision_interior_ref,consumo_interior_huella_sha256,auditoria_interior_ref,
   decision_exterior_ref,consumo_exterior_huella_sha256,auditoria_exterior_ref,confirmada_en)
  VALUES(original.firma_ref,original.recibo_ref,h,dh,p_envoltorio,eh,decision_h,
   cat,ver,pub,clave,pin.entrada,pin.revision_control,
   interior->>'decision_ref',interior->>'consumo_huella_sha256',interior->>'auditoria_ref',
   exterior->>'decision_ref',exterior->>'consumo_huella_sha256',exterior->>'auditoria_ref',
   original.registrada_en);
 ELSE
  SELECT * INTO hija FROM vec_contratacion_temporal.firma_documento_plan_v2 p
   WHERE p.firma_ref=original.firma_ref FOR KEY SHARE;
  -- Una firma V2 anterior a CT176 no tiene plan: no se le asigna uno ahora.
  IF NOT FOUND THEN
   RAISE EXCEPTION 'CT176 firma registrada sin plan; no admite reintento con plan' USING ERRCODE='55000'; END IF;
  IF hija.recibo_ref IS DISTINCT FROM original.recibo_ref
     OR hija.solicitud_huella_sha256 IS DISTINCT FROM h
     OR hija.descriptor_huella_sha256 IS DISTINCT FROM dh
     OR hija.catalogo_id IS DISTINCT FROM cat OR hija.catalogo_version IS DISTINCT FROM ver
     OR hija.publicacion_sha256 IS DISTINCT FROM pub OR hija.entrada_clave IS DISTINCT FROM clave
     OR hija.entrada_original IS DISTINCT FROM pin.entrada
     OR hija.confirmada_en IS DISTINCT FROM original.registrada_en
     OR hija.decision_interior_ref IS DISTINCT FROM original.decision_ref
     OR hija.consumo_interior_huella_sha256 IS DISTINCT FROM original.consumo_huella_sha256
     OR hija.auditoria_interior_ref IS DISTINCT FROM original.auditoria_consumo_ref
     OR encode(sha256(hija.envoltorio_original),'hex') IS DISTINCT FROM hija.envoltorio_huella_sha256 THEN
   RAISE EXCEPTION 'CT176 replay no coincide con plan histórico' USING ERRCODE='55000'; END IF;
 END IF;
 -- CC7 acaba de bloquear el control. Una espera o la escritura de la hija
 -- también pueden agotar las dos decisiones o la vigencia de la entrada.
 IF pin.entrada->>'vigente_desde' IS NULL
    OR pin.entrada->>'vigente_hasta' IS NULL
    OR (pin.entrada->>'vigente_desde')::timestamptz>clock_timestamp()
    OR (pin.entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND
        (pin.entrada->>'vigente_hasta')::timestamptz<=clock_timestamp())
    OR (decision_i->>'valida_hasta')::timestamptz<=clock_timestamp()
    OR (decision_e->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'CT176 pin o decisión caducados antes de confirmar' USING ERRCODE='42501'; END IF;
 RETURN recibo;
END $f$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_hija_plan_firma_v2() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v2(
 text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure; a record; tabla oid; tipo oid; ejecutor oid;
BEGIN
 tabla:='vec_contratacion_temporal.firma_documento_plan_v2'::regclass;
 SELECT c.reltype INTO STRICT tipo FROM pg_class c WHERE c.oid=tabla;
 ejecutor:='vec_contratacion_temporal_ejecutor'::regrole;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_class c
  CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
  WHERE c.oid=tabla AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.firma_documento_plan_v2 FROM %I',pg_get_userbyid(a.grantee));
 END LOOP;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_type t
  CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) x
  WHERE t.oid=tipo AND x.grantee<>0 AND x.grantee<>t.typowner LOOP
  EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.firma_documento_plan_v2 FROM %I',pg_get_userbyid(a.grantee));
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.comprobar_hija_plan_firma_v2()'::regprocedure,
  'vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
    WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
  END LOOP;
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v2(
  text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,
  bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
  TO vec_contratacion_temporal_ejecutor;
 IF EXISTS(SELECT 1 FROM pg_class c
   CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
   WHERE c.oid=tabla AND x.grantee NOT IN (c.relowner))
    OR EXISTS(SELECT 1 FROM pg_type t
      CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) x
      WHERE t.oid=tipo AND x.grantee NOT IN (t.typowner))
    OR EXISTS(SELECT 1 FROM pg_proc p
      CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid='vec_contratacion_temporal.comprobar_hija_plan_firma_v2()'::regprocedure
       AND x.grantee NOT IN (p.proowner))
    OR EXISTS(SELECT 1 FROM pg_proc p
      CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid='vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND (x.grantee NOT IN (p.proowner,ejecutor) OR x.privilege_type<>'EXECUTE')) THEN
  RAISE EXCEPTION 'CT176 ACL efectiva incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
