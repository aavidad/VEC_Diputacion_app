\set ON_ERROR_STOP on
-- B146 (F1-41): solicitud conjunta con su recibo y reintento, y solicitud
-- separada por categoría (contenido de B105, que nunca se instaló solo).
-- * Retira la unicidad persona+convocatoria: la clave primaria ya es
--   persona+convocatoria+categoría. Conserva filas, recibos e historia.
-- * Dos tablas nuevas e inmutables: la presentación conjunta (recibo único,
--   una por persona y convocatoria) y sus categorías, cada una ligada a su
--   solicitud por categoría.
-- * solicitar_inscripcion_v1 y el proyector del aspirante se reconstruyen
--   desde su definición instalada (B96 y B142) con pre y posimagen fijas.
-- * solicitar_inscripcion_conjunta_v1: una autorización V3 por categoría,
--   consumidas en la misma transacción; exige BC13.
-- Sin DOWN ni reaplicación.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000146',0));
DO $causal$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_conjunta') IS NOT NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(text,jsonb,bytea[],bytea[],bytea[],bytea[],bytea[],numeric[],numeric[],bytea[],bytea[],bytea[],bytea[])') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT has_function_privilege(current_user,
   'vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb,text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text,text)') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_portal_externo') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
 -- BC13 instalada: la única comprobación que admite la modalidad conjunta.
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)') IS NULL
 OR NOT has_function_privilege(current_user,
   'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)','EXECUTE')
 OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(text,jsonb)')),'UTF8')),'hex')
    <>'ec7bd7bcd6fbca1b320001088cee5ce6f320b861185c60623ddf2c4668efced4'
 THEN RAISE EXCEPTION 'B146: falta BC13 o la preimagen causal es incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion'::regclass
    AND c.conname='solicitud_inscripcion_persona_ref_convocatoria_ref_key'
    AND c.contype='u' AND c.convalidated
    AND pg_catalog.pg_get_constraintdef(c.oid)='UNIQUE (persona_ref, convocatoria_ref)')
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion'::regclass
    AND c.contype='p' AND pg_catalog.pg_get_constraintdef(c.oid)='PRIMARY KEY (solicitud_ref)')
 THEN RAISE EXCEPTION 'B146: claves de B96 divergentes' USING ERRCODE='55000'; END IF;
END $causal$;
DO $pre$
DECLARE r record; f regprocedure; p record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('solicitar','vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','a3d2e011104a3c5be0e678c12b86abfd691580598283ed5e97e5eb57a3d38063','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}'),
   ('proyectar','vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean,text)','4b0d847f7451f4097ffe8e82e07b4ac4144fb7c5d193b839f59ff87a6dcc63ce','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}')
  ) AS esperado(clave,firma,huella,acl) LOOP
  f:=to_regprocedure(r.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'B146: funcion causal ausente' USING ERRCODE='55000'; END IF;
  SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>'v' OR p.proparallel::text<>'u'
  OR p.proacl::text IS DISTINCT FROM r.acl
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>r.huella
  THEN RAISE EXCEPTION 'B146: definicion o metadatos divergentes: %',r.clave USING ERRCODE='55000'; END IF;
  PERFORM set_config('vec.b146.oid_'||r.clave,f::oid::text,true);
 END LOOP;
END $pre$;

-- Una fila por categoría: la clave primaria (huella de persona, convocatoria y
-- categoría) ya impide repetir categoría; la conjunta añade su propia unicidad.
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion
 DROP CONSTRAINT solicitud_inscripcion_persona_ref_convocatoria_ref_key;

CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta (
 presentacion_ref text COLLATE "C" PRIMARY KEY CHECK(presentacion_ref~'^presentacion_inscripcion_[0-9a-f]{64}$'),
 persona_ref text COLLATE "C" NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 convocatoria_ref text COLLATE "C" NOT NULL CHECK(convocatoria_ref~'^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'),
 categorias jsonb NOT NULL CHECK(jsonb_typeof(categorias)='array' AND jsonb_array_length(categorias) BETWEEN 1 AND 128),
 canal text NOT NULL CHECK(canal='externa_personal'),
 clave_sha256 text NOT NULL CHECK(clave_sha256~'^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^recibo_inscripcion_conjunta_[0-9a-f]{64}$'),
 presentada_en timestamptz(6) NOT NULL CHECK(isfinite(presentada_en)),
 UNIQUE(persona_ref,convocatoria_ref)
);
CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria (
 presentacion_ref text COLLATE "C" NOT NULL
  REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_conjunta(presentacion_ref),
 orden integer NOT NULL CHECK(orden BETWEEN 1 AND 128),
 categoria_ref text COLLATE "C" NOT NULL CHECK(categoria_ref~'^[a-z][a-z0-9_.:-]{2,127}$'),
 solicitud_ref text COLLATE "C" NOT NULL UNIQUE
  REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 PRIMARY KEY(presentacion_ref,orden),
 UNIQUE(presentacion_ref,categoria_ref)
);
REVOKE ALL ON TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta FROM PUBLIC;
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_inscripcion ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta FOR ALL TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
REVOKE ALL ON TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria FROM PUBLIC;
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_inscripcion ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria FOR ALL TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(p_material text, p_captura_actor jsonb, p_recurso_canonico bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; recurso jsonb; publicacion jsonb; etiquetas jsonb;
 v_evaluacion jsonb; v_categorias_validas jsonb; consumo record;
 v_solicitud_ref text; v_clave_sha text; v_material_sha text; v_recurso_sha text;
 v_persona text; v_categoria text; v_categoria_etiqueta text; v_convocatoria text;
 v_canal text; v_tipo_recurso text; v_audiencia text; v_ambito_clave text; v_vinculo_tipo text;
 v_existente vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
 v_recibo vec_bolsa_llamamientos.solicitud_inscripcion_recibo%ROWTYPE;
 v_historia text; v_evento text; v_recibo_ref text; v_declaracion_ref text;
 v_declaracion jsonb;
 v_recibo_json jsonb; v_instante timestamptz(6);
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_externo_bolsa_desarrollo'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'externa_personal'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 16384
 OR p_recurso_canonico IS NULL OR octet_length(p_recurso_canonico) NOT BETWEEN 2 AND 4096
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
 THEN RAISE EXCEPTION 'B96: presentación no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  contexto:=convert_from(p_contexto,'UTF8')::jsonb;
  recurso:=convert_from(p_recurso_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: material de solicitud inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'catalogo_version','categoria_ref','clave_idempotencia','convocatoria_ref','declaraciones','esquema']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.presentar.v1'
 OR coalesce(m->>'convocatoria_ref','') !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
 OR octet_length(m->>'convocatoria_ref') NOT BETWEEN 9 AND 255
 OR coalesce(m->>'categoria_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]*$'
 OR octet_length(m->>'categoria_ref') NOT BETWEEN 1 AND 200
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9_-][A-Za-z0-9._:-]{15,127}$'
 OR jsonb_typeof(m->'catalogo_version') IS DISTINCT FROM 'number'
 OR coalesce(m->>'catalogo_version','') !~ '^[1-9][0-9]{0,9}$'
 OR jsonb_typeof(m->'declaraciones') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'declaraciones')>32
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'declaraciones') AS x(valor)
   WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(x.valor) ORDER BY 1)
       NOT IN(ARRAY['requisito_codigo'],ARRAY['evidencia_ref','requisito_codigo'])
    OR coalesce(x.valor->>'requisito_codigo','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}$'
    OR (x.valor ? 'evidencia_ref' AND
        (coalesce(x.valor->>'evidencia_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,254}$')))
 OR (SELECT count(DISTINCT x.valor->>'requisito_codigo') FROM jsonb_array_elements(m->'declaraciones') AS x(valor))
    <>jsonb_array_length(m->'declaraciones')
 THEN RAISE EXCEPTION 'B96: formulario de solicitud inválido' USING ERRCODE='B9605'; END IF;
 v_convocatoria:=m->>'convocatoria_ref';
 v_categoria:=m->>'categoria_ref';
 v_persona:=d->>'principal_id';
 v_canal:=p_captura_actor->>'canal';
 -- Primera versión: sólo aspirantes externos (sin canal empleado).
 v_tipo_recurso:='inscripcion_convocatoria';
 v_audiencia:='vec_bolsa_llamamientos.inscripcion.presentar.v1';
 v_ambito_clave:='candidato_ref';
 v_vinculo_tipo:='candidato';
 IF coalesce(v_persona,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR v_canal IS DISTINCT FROM 'externa_personal'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_persona
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM v_canal
 OR p_captura_actor->>'idioma' IS NULL
 OR p_captura_actor->>'idioma' NOT IN ('es','en')
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM contexto->>'principal_ref'
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM contexto->>'perfil_activo_ref'
 OR coalesce(contexto->>'contexto_actor_ref','') !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(contexto->>'contexto_version','') !~ '^[1-9][0-9]{0,15}$'
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM v_tipo_recurso
 OR d->>'finalidad' IS DISTINCT FROM 'presentar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM v_canal
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM v_audiencia
 THEN RAISE EXCEPTION 'B96: actor o concesión divergente' USING ERRCODE='42501'; END IF;
 v_solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
   v_persona||chr(31)||v_convocatoria||chr(31)||v_categoria,'UTF8')),'hex');
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 v_recurso_sha:=encode(sha256(p_recurso_canonico),'hex');
 IF jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso) ORDER BY 1) IS DISTINCT FROM ARRAY['ambitos','atributos']
 OR jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'ambitos') ORDER BY 1) IS DISTINCT FROM ARRAY[v_ambito_clave]
 OR coalesce(recurso#>>'{ambitos,candidato_ref}','') !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'atributos') ORDER BY 1) IS DISTINCT FROM
   ARRAY['catalogo_version','categoria_ref','convocatoria_ref','material_sha256']
 OR recurso#>>'{atributos,convocatoria_ref}' IS DISTINCT FROM v_convocatoria
 OR recurso#>>'{atributos,categoria_ref}' IS DISTINCT FROM v_categoria
 OR jsonb_typeof(recurso#>'{atributos,catalogo_version}') IS DISTINCT FROM 'string'
 OR recurso#>>'{atributos,catalogo_version}' IS DISTINCT FROM m->>'catalogo_version'
 OR recurso#>>'{atributos,material_sha256}' IS DISTINCT FROM v_material_sha
 THEN RAISE EXCEPTION 'B96: recurso de presentación incompatible' USING ERRCODE='42501'; END IF;
 IF d->>'recurso_ref' IS DISTINCT FROM v_solicitud_ref
 OR c->>'efecto_ref' IS DISTINCT FROM v_solicitud_ref
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B96: intención y decisión divergentes' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_bolsa_llamamientos:inscripcion:'||v_persona||':'||v_convocatoria,0));
 -- El consumidor revalida firma, identidad, sesión, concesión y origen. Cada
 -- replay exige una nueva decisión vigente; el recibo histórico no da acceso.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_solicitud_ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B96: consumo de presentación incompatible' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B96: candidato de recurso no vinculado' USING ERRCODE='42501'; END IF;
 IF (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') AS x(valor)
     WHERE x.valor->>'tipo'=v_vinculo_tipo)<>1
 OR (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') AS x(valor)
     WHERE x.valor->>'tipo'=v_vinculo_tipo
      AND x.valor->>'referencia'=recurso->'ambitos'->>v_ambito_clave)<>1
 THEN RAISE EXCEPTION 'B96: identidad de recurso no vinculada' USING ERRCODE='42501'; END IF;

 SELECT * INTO v_existente FROM vec_bolsa_llamamientos.solicitud_inscripcion
 WHERE solicitud_ref=v_solicitud_ref;
 IF FOUND THEN
  IF v_existente.persona_ref IS DISTINCT FROM v_persona
  OR v_existente.convocatoria_ref IS DISTINCT FROM v_convocatoria
  OR v_existente.categoria_ref IS DISTINCT FROM v_categoria
  OR v_existente.clave_sha256 IS DISTINCT FROM v_clave_sha
  OR v_existente.material_sha256 IS DISTINCT FROM v_material_sha
  OR v_existente.canal IS DISTINCT FROM v_canal
  THEN
   IF v_existente.clave_sha256 IS DISTINCT FROM v_clave_sha THEN
    RAISE EXCEPTION 'B96: ya existe solicitud para esta persona, convocatoria y categoría' USING ERRCODE='B9604';
   END IF;
   RAISE EXCEPTION 'B96: clave de inscripción reutilizada con otro material' USING ERRCODE='B9603';
  END IF;
  SELECT * INTO STRICT v_recibo FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo
   WHERE solicitud_ref=v_solicitud_ref AND version=1 AND persona_ref=v_persona;
  etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   v_existente.catalogo_ref,v_existente.catalogo_version,v_existente.catalogo_sha256,
   ARRAY[v_categoria],p_captura_actor->>'idioma');
  v_categoria_etiqueta:=etiquetas#>>'{0,categoria}';
  IF v_categoria_etiqueta IS NULL THEN
   RAISE EXCEPTION 'B96: etiqueta de categoría no disponible' USING ERRCODE='B9601';
  END IF;
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
   acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
  VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
   v_solicitud_ref,v_persona,'presentar',consumo.decision_ref,consumo.auditoria_ref,'repetida',consumo.consumida_en);
  RETURN jsonb_build_object('solicitud_ref',v_solicitud_ref,'recibo_ref',v_recibo.recibo_ref,
   'convocatoria_ref',v_convocatoria,'categoria_ref',v_categoria,'categoria',v_categoria_etiqueta,
   'bases_ref',v_existente.bases_ref,'declaracion_ref',v_existente.declaracion_ref,
   'estado','pendiente','version',1,'repetida',true,'registrada_en',v_existente.presentada_en,
   'auditoria_ref',consumo.auditoria_ref);
 END IF;
 -- B146 (B105): la modalidad separada admite una solicitud por categoría; la
 -- clave primaria (persona, convocatoria, categoría) impide la segunda.

 -- Sólo la fuente gobernada de Bolsa determina versión/plazo/catálogo.
 publicacion:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(v_convocatoria,v_categoria);
 IF publicacion->>'catalogo_version' IS DISTINCT FROM m->>'catalogo_version'
 OR publicacion->>'categoria_ref' IS DISTINCT FROM v_categoria
 OR publicacion->>'convocatoria_ref' IS DISTINCT FROM v_convocatoria
 OR jsonb_typeof(publicacion->'requisitos') IS DISTINCT FROM 'array'
 OR jsonb_typeof(publicacion->'categorias') IS DISTINCT FROM 'array'
 OR jsonb_array_length(publicacion->'categorias') NOT BETWEEN 1 AND 128
 OR (SELECT count(*) FROM jsonb_array_elements(publicacion->'categorias') AS categoria(valor)
     WHERE categoria.valor->>'categoria_ref'=v_categoria)<>1
 OR publicacion->>'numero_categorias' IS DISTINCT FROM
    jsonb_array_length(publicacion->'categorias')::text
 OR coalesce(publicacion->>'unidad_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR coalesce(publicacion->>'ambito_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR publicacion->>'fuente_ref' IS DISTINCT FROM
   'ambito.gestion.'||substring(v_convocatoria FROM 5 FOR 64)||'.v'||split_part(v_convocatoria,'_v',2)
 OR publicacion->>'fuente_version' IS DISTINCT FROM publicacion->>'politica_catalogo_version'
 OR publicacion->>'fuente_sha256' IS DISTINCT FROM publicacion->>'politica_catalogo_sha256'
 THEN RAISE EXCEPTION 'B96: catálogo o publicación divergente' USING ERRCODE='B9601'; END IF;
 v_categorias_validas:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object(
   'catalogo_ref',publicacion->>'catalogo_ref',
   'catalogo_version',(publicacion->>'catalogo_version')::integer,
   'catalogo_sha256',publicacion->>'catalogo_sha256',
   'categorias_refs',(SELECT jsonb_agg(x.valor->>'categoria_ref' ORDER BY x.orden)
    FROM jsonb_array_elements(publicacion->'categorias') WITH ORDINALITY AS x(valor,orden)))),
  p_captura_actor->>'idioma');
 IF jsonb_typeof(v_categorias_validas) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_categorias_validas)<>1
 OR v_categorias_validas#>>'{0,catalogo_completo}' IS DISTINCT FROM 'true'
 THEN RAISE EXCEPTION 'B96: formulario de categoría no disponible' USING ERRCODE='B9601'; END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  publicacion->>'catalogo_ref',(publicacion->>'catalogo_version')::integer,
  publicacion->>'catalogo_sha256',ARRAY[v_categoria],p_captura_actor->>'idioma');
 v_categoria_etiqueta:=etiquetas#>>'{0,categoria}';
 IF v_categoria_etiqueta IS NULL THEN
  RAISE EXCEPTION 'B96: etiqueta de categoría no disponible' USING ERRCODE='B9601';
 END IF;
 IF vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
  publicacion->>'politica_catalogo_ref',
  (publicacion->>'politica_catalogo_version')::integer,
  publicacion->>'politica_catalogo_sha256',p_captura_actor->>'canal') IS NOT TRUE
 THEN RAISE EXCEPTION 'B96: política de presentación no vigente' USING ERRCODE='B9601'; END IF;
 FOR v_declaracion IN SELECT valor FROM jsonb_array_elements(m->'declaraciones') AS x(valor) LOOP
  IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(publicacion->'requisitos') AS r(valor)
      WHERE r.valor->>'referencia'=v_declaracion->>'requisito_codigo') THEN
   RAISE EXCEPTION 'B96: declaración ajena a las bases' USING ERRCODE='B9605';
  END IF;
 END LOOP;
 -- El único requisito que este corte puede comprobar por sí mismo es la
 -- identidad certificada de la misma sesión consumida en V3. Todo requisito
 -- descriptivo restante conserva pendiente, aunque exista declaración.
 SELECT coalesce(jsonb_agg(CASE
  WHEN r.valor->>'referencia'='identidad_certificada'
   AND contexto->>'metodo' IN('certificado','dnie')
   AND contexto->>'garantia'='alto'
  THEN r.valor||jsonb_build_object('estado','cumple','motivo_codigo','',
    'procedencia_ref',contexto->>'contexto_actor_ref')
  ELSE r.valor END ORDER BY r.orden),'[]'::jsonb)
 INTO v_evaluacion FROM jsonb_array_elements(publicacion->'requisitos') WITH ORDINALITY AS r(valor,orden);
 v_instante:=consumo.consumida_en;
 IF v_instante< (publicacion->>'plazo_abre_en')::timestamptz
 OR v_instante>= (publicacion->>'plazo_cierra_en')::timestamptz
 THEN RAISE EXCEPTION 'B96: plazo cerrado al consumir autorización' USING ERRCODE='B9602'; END IF;
 v_historia:='historia_inscripcion_'||encode(sha256(convert_to(v_solicitud_ref||':1','UTF8')),'hex');
 v_evento:='evento_inscripcion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
 v_recibo_ref:='recibo_inscripcion_'||encode(sha256(convert_to(v_solicitud_ref||chr(31)||v_material_sha,'UTF8')),'hex');
 v_declaracion_ref:='declaracion_inscripcion_'||encode(sha256(convert_to(
  v_solicitud_ref||chr(31)||v_material_sha,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion(
  solicitud_ref,persona_ref,contexto_actor_ref,contexto_actor_version,
  convocatoria_ref,convocatoria_id,secuencia,version_sha256,
  identificador_publico,categoria_ref,bases_ref,catalogo_ref,catalogo_version,catalogo_sha256,
  politica_catalogo_ref,politica_catalogo_version,politica_catalogo_sha256,
  unidad_ref,ambito_ref,ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,
  formulario_ref,formulario_version,formulario_sha256,plazo_ref,plazo_abre_en,plazo_cierra_en,
  requisitos,requisitos_sha256,declaraciones,declaracion_ref,
  clave_sha256,material_sha256,canal,presentada_en)
 VALUES(v_solicitud_ref,v_persona,contexto->>'contexto_actor_ref',
  (contexto->>'contexto_version')::bigint,v_convocatoria,publicacion->>'convocatoria_id',
  (publicacion->>'secuencia')::bigint,publicacion->>'version_sha256',
  publicacion->>'identificador_publico',v_categoria,publicacion->>'bases_ref',
  publicacion->>'catalogo_ref',(publicacion->>'catalogo_version')::integer,
  publicacion->>'catalogo_sha256',publicacion->>'politica_catalogo_ref',
  (publicacion->>'politica_catalogo_version')::integer,publicacion->>'politica_catalogo_sha256',
  publicacion->>'unidad_ref',publicacion->>'ambito_ref',publicacion->>'fuente_ref',
  (publicacion->>'fuente_version')::integer,publicacion->>'fuente_sha256',
  publicacion->>'formulario_ref',
  (publicacion->>'formulario_version')::integer,publicacion->>'formulario_sha256',
  publicacion->>'plazo_ref',(publicacion->>'plazo_abre_en')::timestamptz,
  (publicacion->>'plazo_cierra_en')::timestamptz,v_evaluacion,
  publicacion->>'requisitos_sha256',m->'declaraciones',v_declaracion_ref,
  v_clave_sha,v_material_sha,
  v_canal,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(v_solicitud_ref,1,'pendiente',v_persona,d->>'perfil_activo_ref',contexto->>'cuenta_ref',
  v_evaluacion,
  consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
  historia_ref,solicitud_ref,version,accion,actor_ref,estado,
  unidad_ref,ambito_ref,ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,
  registrada_en)
 VALUES(v_historia,v_solicitud_ref,1,'presentar',v_persona,'pendiente',
  publicacion->>'unidad_ref',publicacion->>'ambito_ref',publicacion->>'fuente_ref',
  (publicacion->>'fuente_version')::integer,publicacion->>'fuente_sha256',v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
  evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_solicitud_ref,1,'vec.bolsa.inscripcion.presentada.v1',
  jsonb_build_object('solicitud_ref',v_solicitud_ref,'version',1,'convocatoria_ref',v_convocatoria,
    'categoria_ref',v_categoria,'estado','pendiente'),v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
  recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
  historia_ref,evento_ref,auditoria_ref,emitida_en)
 VALUES(v_recibo_ref,v_solicitud_ref,1,v_persona,v_canal,v_clave_sha,v_material_sha,
  v_historia,v_evento,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
  acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
 VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
  v_solicitud_ref,v_persona,'presentar',consumo.decision_ref,consumo.auditoria_ref,'confirmada',v_instante);
 RETURN jsonb_build_object('solicitud_ref',v_solicitud_ref,'recibo_ref',v_recibo_ref,
  'estado','pendiente','version',1,'repetida',false,'registrada_en',v_instante,
  'auditoria_ref',consumo.auditoria_ref,'categoria_ref',v_categoria,
  'categoria',v_categoria_etiqueta,
  'convocatoria_ref',v_convocatoria,'bases_ref',publicacion->>'bases_ref',
  'declaracion_ref',v_declaracion_ref);
END $function$
;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(p_fuente jsonb, p_persona_ref text, p_idioma text, p_metodo text, p_garantia text, p_lista boolean, p_canal text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 v_items jsonb; v_item jsonb; v_categoria jsonb; v_requisito jsonb;
 v_selectores jsonb:='[]'::jsonb; v_etiquetas jsonb;
 v_categorias_pedidos jsonb; v_categorias_validas jsonb;
 v_gestion_pedidos jsonb; v_gestion_validos jsonb;
 v_propias jsonb; v_categorias jsonb; v_requisitos jsonb; v_bolsas jsonb:='[]'::jsonb;
 v_etiqueta jsonb; v_requisitos_resumen text; v_num_categorias integer; v_carta jsonb;
 v_estado text; v_puede boolean; v_catalogo_completo boolean; v_impedimento text;
 v_pos integer:=0; v_gestion_pos integer:=0;
 v_conv text; v_propia jsonb; v_salida jsonb;
 v_propias_conv jsonb; v_propias_total integer; v_conjunta boolean;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en') OR p_lista IS NULL
 OR p_canal IS DISTINCT FROM 'externa_personal'
 OR jsonb_typeof(p_fuente) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B96: fuente abierta inválida' USING ERRCODE='55000'; END IF;
 v_items:=CASE WHEN p_lista THEN p_fuente->'items' ELSE jsonb_build_array(p_fuente) END;
 IF jsonb_typeof(v_items) IS DISTINCT FROM 'array' OR jsonb_array_length(v_items)>100
 THEN RAISE EXCEPTION 'B96: página abierta fuera de límite' USING ERRCODE='54000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(v_items) AS x(valor)
  WHERE (p_lista AND (
   coalesce(x.valor->>'numero_categorias','') !~ '^[1-9][0-9]{0,2}$'
   OR (x.valor->>'numero_categorias')::integer NOT BETWEEN 1 AND 128
   OR coalesce(x.valor->>'categoria_ref_comprobacion','')=''
   OR jsonb_typeof(x.valor->'categorias_refs_comprobacion') IS DISTINCT FROM 'array'
   OR jsonb_array_length(x.valor->'categorias_refs_comprobacion')
      IS DISTINCT FROM (x.valor->>'numero_categorias')::integer
   OR x.valor ? 'categorias'))
  OR (NOT p_lista AND (
   jsonb_typeof(x.valor->'categorias') IS DISTINCT FROM 'array'
   OR jsonb_array_length(x.valor->'categorias') NOT BETWEEN 1 AND 128)))
 THEN RAISE EXCEPTION 'B96: categorías publicadas incompatibles' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',item.valor->>'catalogo_ref',
   'catalogo_version',(item.valor->>'catalogo_version')::integer,
   'catalogo_sha256',item.valor->>'catalogo_sha256',
   'categoria_ref',categoria.valor->>'categoria_ref',
   'politica_catalogo_ref',item.valor->>'politica_catalogo_ref',
   'politica_catalogo_version',(item.valor->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',item.valor->>'politica_catalogo_sha256')
   ORDER BY item.orden,categoria.orden),'[]'::jsonb)
 INTO v_selectores
 FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden)
 CROSS JOIN LATERAL jsonb_array_elements(
   CASE WHEN p_lista THEN jsonb_build_array(jsonb_build_object(
     'categoria_ref',item.valor->>'categoria_ref_comprobacion'))
   ELSE item.valor->'categorias' END
 ) WITH ORDINALITY AS categoria(valor,orden);
 v_etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  v_selectores,p_idioma,p_canal);
 IF jsonb_typeof(v_etiquetas) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_etiquetas)<>jsonb_array_length(v_selectores)
 THEN RAISE EXCEPTION 'B96: lote de etiquetas incompleto' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'politica_catalogo_ref',item.valor->>'politica_catalogo_ref',
  'politica_catalogo_version',(item.valor->>'politica_catalogo_version')::integer,
  'politica_catalogo_sha256',item.valor->>'politica_catalogo_sha256',
  'convocatoria_ref',item.valor->>'convocatoria_ref') ORDER BY item.orden),'[]'::jsonb)
 INTO v_gestion_pedidos
 FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden);
 v_gestion_validos:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  v_gestion_pedidos);
 IF jsonb_typeof(v_gestion_validos) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_gestion_validos)<>jsonb_array_length(v_items)
 THEN RAISE EXCEPTION 'B96: cotejo de gestión incompleto' USING ERRCODE='55000'; END IF;
 IF p_lista THEN
  SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',item.valor->>'catalogo_ref',
   'catalogo_version',(item.valor->>'catalogo_version')::integer,
   'catalogo_sha256',item.valor->>'catalogo_sha256',
   'categorias_refs',item.valor->'categorias_refs_comprobacion')
   ORDER BY item.orden),'[]'::jsonb)
  INTO v_categorias_pedidos
  FROM jsonb_array_elements(v_items) WITH ORDINALITY AS item(valor,orden);
  v_categorias_validas:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
   v_categorias_pedidos,p_idioma);
  IF jsonb_typeof(v_categorias_validas) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_categorias_validas)<>jsonb_array_length(v_items)
  THEN RAISE EXCEPTION 'B96: cotejo de categorías incompleto' USING ERRCODE='55000'; END IF;
 END IF;
 -- B146: una convocatoria puede tener varias solicitudes propias, una por categoría.
 SELECT coalesce(jsonb_object_agg(q.convocatoria_ref,q.solicitudes),'{}'::jsonb)
 INTO v_propias FROM (
  SELECT s.convocatoria_ref,jsonb_agg(jsonb_build_object(
   'categoria_ref',s.categoria_ref,'solicitud_ref',s.solicitud_ref,'estado',v.estado)
   ORDER BY s.categoria_ref COLLATE "C",s.solicitud_ref COLLATE "C") AS solicitudes
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT estado FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY version DESC LIMIT 1) v ON true
  WHERE s.persona_ref=p_persona_ref
   AND s.convocatoria_ref IN (SELECT x.valor->>'convocatoria_ref' FROM jsonb_array_elements(v_items) AS x(valor))
  GROUP BY s.convocatoria_ref
 ) q;
 FOR v_item IN SELECT valor FROM jsonb_array_elements(v_items) AS x(valor) LOOP
  v_conv:=v_item->>'convocatoria_ref';
 v_categorias:='[]'::jsonb;
 v_requisitos:='[]'::jsonb;
  v_requisitos_resumen:=v_item->>'resumen';
  v_puede:=true;
  IF p_lista THEN
   v_num_categorias:=(v_item->>'numero_categorias')::integer;
   IF jsonb_typeof(v_categorias_validas->v_pos->'catalogo_completo') IS DISTINCT FROM 'boolean'
    OR v_categorias_validas->v_pos->>'numero_categorias'
       IS DISTINCT FROM v_num_categorias::text
   THEN RAISE EXCEPTION 'B96: cotejo de categorías divergente' USING ERRCODE='55000'; END IF;
   v_catalogo_completo:=(v_categorias_validas->v_pos->>'catalogo_completo')::boolean;
   v_etiqueta:=v_etiquetas->v_pos;
   v_pos:=v_pos+1;
   IF v_etiqueta->>'categoria_ref' IS DISTINCT FROM v_item->>'categoria_ref_comprobacion'
    OR coalesce(v_etiqueta->>'categoria','')=''
    OR jsonb_typeof(v_etiqueta->'politica_valida') IS DISTINCT FROM 'boolean'
    OR coalesce(v_etiqueta->>'motivo_etiqueta_pendiente','')=''
    OR coalesce(v_etiqueta->>'motivo_etiqueta_cumple','')=''
   THEN RAISE EXCEPTION 'B96: categoría del lote divergente' USING ERRCODE='55000'; END IF;
   v_puede:=(v_etiqueta->>'politica_valida')::boolean AND v_catalogo_completo;
  ELSE
   v_num_categorias:=jsonb_array_length(v_item->'categorias');
   IF v_item->>'numero_categorias' IS DISTINCT FROM v_num_categorias::text THEN
    RAISE EXCEPTION 'B96: total de categorías divergente' USING ERRCODE='55000';
   END IF;
   FOR v_categoria IN SELECT valor FROM jsonb_array_elements(v_item->'categorias') AS x(valor) LOOP
    v_etiqueta:=v_etiquetas->v_pos;
    v_pos:=v_pos+1;
    IF v_etiqueta->>'categoria_ref' IS DISTINCT FROM v_categoria->>'categoria_ref'
     OR coalesce(v_etiqueta->>'categoria','')=''
     OR jsonb_typeof(v_etiqueta->'politica_valida') IS DISTINCT FROM 'boolean'
     OR coalesce(v_etiqueta->>'motivo_etiqueta_pendiente','')=''
     OR coalesce(v_etiqueta->>'motivo_etiqueta_cumple','')=''
    THEN RAISE EXCEPTION 'B96: categoría del lote divergente' USING ERRCODE='55000'; END IF;
    v_categorias:=v_categorias||jsonb_build_array(jsonb_build_object(
     'categoria_ref',v_categoria->>'categoria_ref','categoria',v_etiqueta->>'categoria'));
    v_puede:=v_puede AND (v_etiqueta->>'politica_valida')::boolean;
   END LOOP;
   -- El catálogo de política es común a la versión, y el detalle sí presenta
   -- cada requisito con su estado y motivo desde la fuente publicada.
   v_etiqueta:=v_etiquetas->(v_pos-1);
   v_requisitos_resumen:=NULL;
   FOR v_requisito IN SELECT valor FROM jsonb_array_elements(v_item->'requisitos') AS x(valor) LOOP
   IF coalesce(v_requisito->>'descripcion','')='' THEN
    RAISE EXCEPTION 'B96: descripción de requisito ausente' USING ERRCODE='55000';
   END IF;
   v_requisitos_resumen:=concat_ws('; ',v_requisitos_resumen,v_requisito->>'descripcion');
   v_estado:='pendiente';
   IF v_requisito->>'referencia'='identidad_certificada'
    AND p_metodo IN('certificado','dnie') AND p_garantia='alto' THEN
    v_estado:='cumple';
   END IF;
   v_requisitos:=v_requisitos||jsonb_build_array(jsonb_build_object(
    'codigo',coalesce(v_requisito->>'codigo',v_requisito->>'referencia'),
    'descripcion',v_requisito->>'descripcion',
    'obligatorio',(v_requisito->>'obligatorio')::boolean,
    'estado',v_estado,
    'motivo_codigo',CASE WHEN v_estado='pendiente' THEN 'requisito.pendiente' ELSE '' END,
    'motivo_etiqueta',CASE WHEN v_estado='pendiente'
      THEN v_etiqueta->>'motivo_etiqueta_pendiente' ELSE v_etiqueta->>'motivo_etiqueta_cumple' END,
    'hito_cumplimiento',NULL,'hito_etiqueta',NULL));
  END LOOP;
  END IF;
  IF jsonb_typeof(v_gestion_validos->v_gestion_pos->'ambito_gestion_disponible')
     IS DISTINCT FROM 'boolean'
  THEN RAISE EXCEPTION 'B96: gestión de convocatoria divergente' USING ERRCODE='55000'; END IF;
  v_puede:=v_puede AND (v_gestion_validos->v_gestion_pos->>'ambito_gestion_disponible')::boolean;
  v_gestion_pos:=v_gestion_pos+1;
  -- B142: la modalidad sellada viaja a lista y ficha; otro valor es incoherente.
  IF jsonb_typeof(v_item->'modalidad_inscripcion') IS DISTINCT FROM 'string'
   OR v_item->>'modalidad_inscripcion' NOT IN('separada','conjunta')
  THEN RAISE EXCEPTION 'B142: modalidad de inscripcion invalida' USING ERRCODE='55000'; END IF;
  -- B146: separada admite una solicitud por categoría; conjunta, una sola
  -- presentación para las categorías elegidas.
  v_conjunta:=v_item->>'modalidad_inscripcion'='conjunta';
  v_propias_conv:=coalesce(v_propias->v_conv,'[]'::jsonb);
  v_propias_total:=jsonb_array_length(v_propias_conv);
  IF v_propias_total>v_num_categorias THEN
   RAISE EXCEPTION 'B146: solicitudes propias exceden categorías publicadas' USING ERRCODE='55000';
  END IF;
  IF NOT p_lista AND EXISTS (
   SELECT 1 FROM jsonb_array_elements(v_propias_conv) AS propia(valor)
   WHERE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(v_categorias) AS categoria(valor)
    WHERE categoria.valor->>'categoria_ref'=propia.valor->>'categoria_ref')
  ) THEN RAISE EXCEPTION 'B146: solicitud propia ajena a la publicación' USING ERRCODE='55000'; END IF;
  v_propia:=CASE WHEN v_propias_total=1 THEN v_propias_conv->0 ELSE NULL END;
  v_puede:=v_puede AND CASE WHEN v_conjunta THEN v_propias_total=0
    ELSE v_propias_total<v_num_categorias END;
  v_impedimento:=CASE WHEN v_propias_total=v_num_categorias OR (v_conjunta AND v_propias_total>0)
    THEN v_etiqueta->>'impedimento_etiqueta_existente'
    WHEN NOT v_puede THEN v_etiqueta->>'impedimento_etiqueta_politica'
    ELSE NULL END;
  IF NOT v_puede AND coalesce(v_impedimento,'')='' THEN
   RAISE EXCEPTION 'B96: impedimento sin catálogo' USING ERRCODE='B9601';
  END IF;
  v_carta:=jsonb_build_object(
   'convocatoria_ref',v_conv,'titulo',v_item->>'titulo',
   'modalidad_inscripcion',v_item->>'modalidad_inscripcion',
   'numero_categorias',v_num_categorias,
   'plazo_inicio',(v_item->>'plazo_abre_en')::timestamptz,
   'plazo_fin',(v_item->>'plazo_cierra_en')::timestamptz,
   'catalogo_version',(v_item->>'catalogo_version')::bigint,
   'requisitos_resumen',coalesce(v_requisitos_resumen,v_item->>'resumen'),
   'puede_iniciar',v_puede,
   'impedimento_etiqueta',v_impedimento,
   'estado_solicitud_propia',v_propia->>'estado',
   'solicitud_ref',v_propia->>'solicitud_ref');
  IF NOT p_lista THEN
   v_carta:=v_carta||jsonb_build_object('categorias',v_categorias,'requisitos',v_requisitos,
    'solicitudes_propias',v_propias_conv);
  END IF;
  v_bolsas:=v_bolsas||jsonb_build_array(v_carta);
 END LOOP;
 IF p_lista THEN
  v_salida:=jsonb_build_object('convocatorias',v_bolsas,'total',(p_fuente->>'total')::bigint,
   'cursor_siguiente',p_fuente->>'siguiente_cursor');
 ELSE v_salida:=v_bolsas->0; END IF;
 RETURN v_salida;
END $function$
;

-- Solicitud conjunta: una sola presentación y un solo recibo para varias
-- categorías. Cada categoría conserva su propia fila de solicitud (B96), con
-- su autorización V3 consumida en esta misma transacción, para que RRHH revise
-- e incorpore por categoría como hasta ahora. El reintento con la misma clave
-- y el mismo material devuelve el mismo recibo; exige decisiones nuevas.
CREATE FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(
 p_material text,p_captura_actor jsonb,p_recursos bytea[],
 p_capacidades bytea[],p_decisiones bytea[],p_motivos bytea[],p_contextos bytea[],
 p_persona_versiones numeric[],p_perfil_versiones numeric[],
 p_payloads bytea[],p_sobres bytea[],p_evidencias bytea[],p_raices bytea[]
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; recurso jsonb; publicacion jsonb; etiquetas jsonb;
 v_evaluacion jsonb; v_categorias_validas jsonb; consumo record; v_declaracion jsonb;
 v_n integer; i integer; v_categorias text[]; v_solicitudes text[]:='{}';
 v_decisiones text[]:='{}'; v_huellas text[]:='{}'; v_consumos text[]:='{}'; v_auditorias text[]:='{}';
 v_persona text; v_convocatoria text; v_canal text; v_candidato text;
 v_presentacion_ref text; v_clave_sha text; v_material_sha text; v_recurso_sha text;
 v_recibo_ref text; v_historia text; v_evento text; v_declaracion_ref text; v_solicitud text;
 v_existente vec_bolsa_llamamientos.solicitud_inscripcion_conjunta%ROWTYPE;
 v_salida jsonb:='[]'::jsonb; v_desde timestamptz(6); v_instante timestamptz(6);
BEGIN
 v_n:=cardinality(p_recursos);
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_externo_bolsa_desarrollo'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'externa_personal'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 16384
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 OR v_n IS NULL OR v_n NOT BETWEEN 1 AND 128
 -- Once listas paralelas, de una dimensión, base 1 y del mismo tamaño.
 OR EXISTS(SELECT 1 FROM (VALUES
   (array_ndims(p_recursos),array_lower(p_recursos,1),cardinality(p_recursos)),
   (array_ndims(p_capacidades),array_lower(p_capacidades,1),cardinality(p_capacidades)),
   (array_ndims(p_decisiones),array_lower(p_decisiones,1),cardinality(p_decisiones)),
   (array_ndims(p_motivos),array_lower(p_motivos,1),cardinality(p_motivos)),
   (array_ndims(p_contextos),array_lower(p_contextos,1),cardinality(p_contextos)),
   (array_ndims(p_persona_versiones),array_lower(p_persona_versiones,1),cardinality(p_persona_versiones)),
   (array_ndims(p_perfil_versiones),array_lower(p_perfil_versiones,1),cardinality(p_perfil_versiones)),
   (array_ndims(p_payloads),array_lower(p_payloads,1),cardinality(p_payloads)),
   (array_ndims(p_sobres),array_lower(p_sobres,1),cardinality(p_sobres)),
   (array_ndims(p_evidencias),array_lower(p_evidencias,1),cardinality(p_evidencias)),
   (array_ndims(p_raices),array_lower(p_raices,1),cardinality(p_raices))) AS x(dims,base,total)
   WHERE x.dims IS DISTINCT FROM 1 OR x.base IS DISTINCT FROM 1 OR x.total IS DISTINCT FROM v_n)
 OR array_position(p_recursos,NULL) IS NOT NULL OR array_position(p_capacidades,NULL) IS NOT NULL
 OR array_position(p_decisiones,NULL) IS NOT NULL OR array_position(p_contextos,NULL) IS NOT NULL
 OR EXISTS(SELECT 1 FROM unnest(p_recursos) AS r(valor) WHERE octet_length(r.valor) NOT BETWEEN 2 AND 4096)
 THEN RAISE EXCEPTION 'B146: presentación conjunta no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;
  contexto:=convert_from(p_contextos[1],'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B146: material de solicitud inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'catalogo_version','categorias_refs','clave_idempotencia','convocatoria_ref','declaraciones','esquema']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.presentar_conjunta.v1'
 OR coalesce(m->>'convocatoria_ref','') !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
 OR octet_length(m->>'convocatoria_ref') NOT BETWEEN 9 AND 255
 OR jsonb_typeof(m->'categorias_refs') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'categorias_refs') NOT BETWEEN 1 AND 128
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'categorias_refs') AS x(valor)
   WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'string'
    OR x.valor#>>'{}' !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 -- Orden C y sin repetidos: una sola forma canónica para la misma elección.
 OR m->'categorias_refs' IS DISTINCT FROM (SELECT jsonb_agg(DISTINCT (x.valor#>>'{}') COLLATE "C"
   ORDER BY (x.valor#>>'{}') COLLATE "C") FROM jsonb_array_elements(m->'categorias_refs') AS x(valor))
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9_-][A-Za-z0-9._:-]{15,127}$'
 OR jsonb_typeof(m->'catalogo_version') IS DISTINCT FROM 'number'
 OR coalesce(m->>'catalogo_version','') !~ '^[1-9][0-9]{0,9}$'
 OR jsonb_typeof(m->'declaraciones') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'declaraciones')>32
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'declaraciones') AS x(valor)
   WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(x.valor) ORDER BY 1)
       NOT IN(ARRAY['requisito_codigo'],ARRAY['evidencia_ref','requisito_codigo'])
    OR coalesce(x.valor->>'requisito_codigo','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}$'
    OR (x.valor ? 'evidencia_ref' AND
        (coalesce(x.valor->>'evidencia_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,254}$')))
 OR (SELECT count(DISTINCT x.valor->>'requisito_codigo') FROM jsonb_array_elements(m->'declaraciones') AS x(valor))
    <>jsonb_array_length(m->'declaraciones')
 THEN RAISE EXCEPTION 'B146: formulario de solicitud inválido' USING ERRCODE='B9605'; END IF;
 v_categorias:=ARRAY(SELECT x.valor FROM jsonb_array_elements_text(m->'categorias_refs')
   WITH ORDINALITY AS x(valor,orden) ORDER BY x.orden);
 IF cardinality(v_categorias)<>v_n
 THEN RAISE EXCEPTION 'B146: una autorización por categoría' USING ERRCODE='42501'; END IF;
 v_convocatoria:=m->>'convocatoria_ref';
 v_canal:=p_captura_actor->>'canal';
 v_persona:=p_captura_actor->>'persona_ref';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 -- Un solo actor y un solo contexto para toda la presentación.
 IF coalesce(v_persona,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'idioma' IS NULL OR p_captura_actor->>'idioma' NOT IN ('es','en')
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM contexto->>'principal_ref'
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM contexto->>'perfil_activo_ref'
 OR coalesce(contexto->>'contexto_actor_ref','') !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(contexto->>'contexto_version','') !~ '^[1-9][0-9]{0,15}$'
 OR EXISTS(SELECT 1 FROM unnest(p_contextos) AS x(valor) WHERE x.valor IS DISTINCT FROM p_contextos[1])
 THEN RAISE EXCEPTION 'B146: actor o concesión divergente' USING ERRCODE='42501'; END IF;
 FOR i IN 1..v_n LOOP
  BEGIN
   d:=convert_from(p_decisiones[i],'UTF8')::jsonb;
   c:=convert_from(p_capacidades[i],'UTF8')::jsonb;
   recurso:=convert_from(p_recursos[i],'UTF8')::jsonb;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'B146: material de solicitud inválido' USING ERRCODE='22023';
  END;
  v_solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
    v_persona||chr(31)||v_convocatoria||chr(31)||v_categorias[i],'UTF8')),'hex');
  v_recurso_sha:=encode(sha256(p_recursos[i]),'hex');
  IF d->>'principal_id' IS DISTINCT FROM v_persona
  OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
  OR d->>'concedida' IS DISTINCT FROM 'true'
  OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
  OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'inscripcion_convocatoria'
  OR d->>'finalidad' IS DISTINCT FROM 'presentar_inscripcion'
  OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM v_canal
  OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.presentar.v1'
  THEN RAISE EXCEPTION 'B146: actor o concesión divergente' USING ERRCODE='42501'; END IF;
  -- Mismo recurso que la solicitud por categoría, ligado al material conjunto.
  IF jsonb_typeof(recurso) IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(recurso) ORDER BY 1) IS DISTINCT FROM ARRAY['ambitos','atributos']
  OR jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(recurso->'ambitos') ORDER BY 1) IS DISTINCT FROM ARRAY['candidato_ref']
  OR coalesce(recurso#>>'{ambitos,candidato_ref}','') !~ '^can_[A-Za-z0-9_-]{22,128}$'
  OR (i>1 AND recurso#>>'{ambitos,candidato_ref}' IS DISTINCT FROM v_candidato)
  OR jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(recurso->'atributos') ORDER BY 1) IS DISTINCT FROM
    ARRAY['catalogo_version','categoria_ref','convocatoria_ref','material_sha256']
  OR recurso#>>'{atributos,convocatoria_ref}' IS DISTINCT FROM v_convocatoria
  OR recurso#>>'{atributos,categoria_ref}' IS DISTINCT FROM v_categorias[i]
  OR jsonb_typeof(recurso#>'{atributos,catalogo_version}') IS DISTINCT FROM 'string'
  OR recurso#>>'{atributos,catalogo_version}' IS DISTINCT FROM m->>'catalogo_version'
  OR recurso#>>'{atributos,material_sha256}' IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'B146: recurso de presentación incompatible' USING ERRCODE='42501'; END IF;
  v_candidato:=recurso#>>'{ambitos,candidato_ref}';
  IF d->>'recurso_ref' IS DISTINCT FROM v_solicitud
  OR c->>'efecto_ref' IS DISTINCT FROM v_solicitud
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
  THEN RAISE EXCEPTION 'B146: intención y decisión divergentes' USING ERRCODE='42501'; END IF;
  v_solicitudes:=v_solicitudes||v_solicitud;
  v_huellas:=v_huellas||v_recurso_sha;
 END LOOP;
 -- Misma clave de bloqueo que la solicitud por categoría (B96).
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_bolsa_llamamientos:inscripcion:'||v_persona||':'||v_convocatoria,0));
 -- Cada decisión se consume aquí; ninguna se reutiliza ni da acceso al recibo.
 FOR i IN 1..v_n LOOP
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(
   p_capacidades[i],p_decisiones[i],p_motivos[i],p_contextos[i],p_persona_versiones[i],p_perfil_versiones[i],
   p_payloads[i],p_sobres[i],p_evidencias[i],p_raices[i]);
  IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_solicitudes[i]
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_huellas[i]
  OR consumo.decision_ref IS NULL OR consumo.decision_ref=ANY(v_decisiones)
  OR consumo.consumida_en IS NULL
  THEN RAISE EXCEPTION 'B146: consumo de presentación incompatible' USING ERRCODE='42501'; END IF;
  v_decisiones:=v_decisiones||consumo.decision_ref;
  v_auditorias:=v_auditorias||consumo.auditoria_ref;
  v_consumos:=v_consumos||consumo.consumo_huella_sha256;
  v_desde:=least(v_desde,consumo.consumida_en);
  v_instante:=greatest(v_instante,consumo.consumida_en);
 END LOOP;
 IF jsonb_typeof(contexto->'vinculos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') AS x(valor)
     WHERE x.valor->>'tipo'='candidato')<>1
 OR (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') AS x(valor)
     WHERE x.valor->>'tipo'='candidato' AND x.valor->>'referencia'=v_candidato)<>1
 THEN RAISE EXCEPTION 'B146: identidad de recurso no vinculada' USING ERRCODE='42501'; END IF;

 v_presentacion_ref:='presentacion_inscripcion_'||encode(sha256(convert_to(
   v_persona||chr(31)||v_convocatoria||chr(31)||'conjunta','UTF8')),'hex');
 SELECT * INTO v_existente FROM vec_bolsa_llamamientos.solicitud_inscripcion_conjunta
  WHERE persona_ref=v_persona AND convocatoria_ref=v_convocatoria;
 IF FOUND THEN
  IF v_existente.clave_sha256 IS DISTINCT FROM v_clave_sha THEN
   RAISE EXCEPTION 'B146: ya existe solicitud conjunta para esta convocatoria' USING ERRCODE='B9604';
  END IF;
  IF v_existente.presentacion_ref IS DISTINCT FROM v_presentacion_ref
  OR v_existente.material_sha256 IS DISTINCT FROM v_material_sha
  OR v_existente.canal IS DISTINCT FROM v_canal
  OR v_existente.categorias IS DISTINCT FROM m->'categorias_refs'
  THEN RAISE EXCEPTION 'B146: clave de inscripción reutilizada con otro material' USING ERRCODE='B9603'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',s.categoria_ref,
    'solicitud_ref',s.solicitud_ref,'recibo_ref',r.recibo_ref,'declaracion_ref',s.declaracion_ref)
    ORDER BY g.orden),'[]'::jsonb)
  INTO v_salida
  FROM vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria g
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion s ON s.solicitud_ref=g.solicitud_ref
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r ON r.solicitud_ref=s.solicitud_ref AND r.version=1
  WHERE g.presentacion_ref=v_existente.presentacion_ref
   AND s.persona_ref=v_persona AND s.convocatoria_ref=v_convocatoria;
  IF jsonb_array_length(v_salida)<>v_n
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(v_salida) WITH ORDINALITY AS x(valor,orden)
    WHERE x.valor->>'solicitud_ref' IS DISTINCT FROM v_solicitudes[x.orden]
     OR x.valor->>'categoria_ref' IS DISTINCT FROM v_categorias[x.orden])
  THEN RAISE EXCEPTION 'B146: presentación conjunta incoherente' USING ERRCODE='55000'; END IF;
  FOR i IN 1..v_n LOOP
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
    acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
   VALUES('acceso_inscripcion_'||encode(sha256(convert_to(v_decisiones[i],'UTF8')),'hex'),
    v_solicitudes[i],v_persona,'presentar',v_decisiones[i],v_auditorias[i],'repetida',v_instante);
  END LOOP;
  publicacion:=jsonb_build_object('bases_ref',(SELECT s.bases_ref FROM vec_bolsa_llamamientos.solicitud_inscripcion s
    WHERE s.solicitud_ref=v_solicitudes[1]),
   'catalogo_ref',(SELECT s.catalogo_ref FROM vec_bolsa_llamamientos.solicitud_inscripcion s WHERE s.solicitud_ref=v_solicitudes[1]),
   'catalogo_version',(SELECT s.catalogo_version FROM vec_bolsa_llamamientos.solicitud_inscripcion s WHERE s.solicitud_ref=v_solicitudes[1]),
   'catalogo_sha256',(SELECT s.catalogo_sha256 FROM vec_bolsa_llamamientos.solicitud_inscripcion s WHERE s.solicitud_ref=v_solicitudes[1]));
  v_instante:=v_existente.presentada_en;
  v_recibo_ref:=v_existente.recibo_ref;
 ELSE
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_inscripcion s
   WHERE s.persona_ref=v_persona AND s.convocatoria_ref=v_convocatoria) THEN
   RAISE EXCEPTION 'B146: ya existe solicitud para esta convocatoria' USING ERRCODE='B9604';
  END IF;
  -- Sólo la fuente gobernada de Bolsa determina modalidad, versión, plazo y catálogo.
  publicacion:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_conjunta_v1(
   v_convocatoria,m->'categorias_refs');
  IF publicacion->>'catalogo_version' IS DISTINCT FROM m->>'catalogo_version'
  OR publicacion->>'convocatoria_ref' IS DISTINCT FROM v_convocatoria
  OR publicacion->>'modalidad_inscripcion' IS DISTINCT FROM 'conjunta'
  OR publicacion->'categorias_seleccionadas' IS DISTINCT FROM m->'categorias_refs'
  OR jsonb_typeof(publicacion->'requisitos') IS DISTINCT FROM 'array'
  OR jsonb_typeof(publicacion->'categorias') IS DISTINCT FROM 'array'
  OR jsonb_array_length(publicacion->'categorias') NOT BETWEEN 1 AND 128
  OR publicacion->>'numero_categorias' IS DISTINCT FROM
     jsonb_array_length(publicacion->'categorias')::text
  OR coalesce(publicacion->>'unidad_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR coalesce(publicacion->>'ambito_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR publicacion->>'fuente_ref' IS DISTINCT FROM
    'ambito.gestion.'||substring(v_convocatoria FROM 5 FOR 64)||'.v'||split_part(v_convocatoria,'_v',2)
  OR publicacion->>'fuente_version' IS DISTINCT FROM publicacion->>'politica_catalogo_version'
  OR publicacion->>'fuente_sha256' IS DISTINCT FROM publicacion->>'politica_catalogo_sha256'
  THEN RAISE EXCEPTION 'B146: catálogo o publicación divergente' USING ERRCODE='B9601'; END IF;
  v_categorias_validas:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
   jsonb_build_array(jsonb_build_object(
    'catalogo_ref',publicacion->>'catalogo_ref',
    'catalogo_version',(publicacion->>'catalogo_version')::integer,
    'catalogo_sha256',publicacion->>'catalogo_sha256',
    'categorias_refs',(SELECT jsonb_agg(x.valor->>'categoria_ref' ORDER BY x.orden)
     FROM jsonb_array_elements(publicacion->'categorias') WITH ORDINALITY AS x(valor,orden)))),
   p_captura_actor->>'idioma');
  IF jsonb_typeof(v_categorias_validas) IS DISTINCT FROM 'array'
  OR jsonb_array_length(v_categorias_validas)<>1
  OR v_categorias_validas#>>'{0,catalogo_completo}' IS DISTINCT FROM 'true'
  THEN RAISE EXCEPTION 'B146: formulario de categoría no disponible' USING ERRCODE='B9601'; END IF;
  IF vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
   publicacion->>'politica_catalogo_ref',
   (publicacion->>'politica_catalogo_version')::integer,
   publicacion->>'politica_catalogo_sha256',v_canal) IS NOT TRUE
  THEN RAISE EXCEPTION 'B146: política de presentación no vigente' USING ERRCODE='B9601'; END IF;
  FOR v_declaracion IN SELECT valor FROM jsonb_array_elements(m->'declaraciones') AS x(valor) LOOP
   IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(publicacion->'requisitos') AS r(valor)
       WHERE r.valor->>'referencia'=v_declaracion->>'requisito_codigo') THEN
    RAISE EXCEPTION 'B146: declaración ajena a las bases' USING ERRCODE='B9605';
   END IF;
  END LOOP;
  -- Como en B96, sólo la identidad certificada de esta sesión se comprueba aquí.
  SELECT coalesce(jsonb_agg(CASE
   WHEN r.valor->>'referencia'='identidad_certificada'
    AND contexto->>'metodo' IN('certificado','dnie')
    AND contexto->>'garantia'='alto'
   THEN r.valor||jsonb_build_object('estado','cumple','motivo_codigo','',
     'procedencia_ref',contexto->>'contexto_actor_ref')
   ELSE r.valor END ORDER BY r.orden),'[]'::jsonb)
  INTO v_evaluacion FROM jsonb_array_elements(publicacion->'requisitos') WITH ORDINALITY AS r(valor,orden);
  IF v_desde<(publicacion->>'plazo_abre_en')::timestamptz
  OR v_instante>=(publicacion->>'plazo_cierra_en')::timestamptz
  THEN RAISE EXCEPTION 'B146: plazo cerrado al consumir autorización' USING ERRCODE='B9602'; END IF;
  v_recibo_ref:='recibo_inscripcion_conjunta_'||encode(sha256(convert_to(
    v_presentacion_ref||chr(31)||v_material_sha,'UTF8')),'hex');
  FOR i IN 1..v_n LOOP
   v_solicitud:=v_solicitudes[i];
   v_historia:='historia_inscripcion_'||encode(sha256(convert_to(v_solicitud||':1','UTF8')),'hex');
   v_evento:='evento_inscripcion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
   v_declaracion_ref:='declaracion_inscripcion_'||encode(sha256(convert_to(
    v_solicitud||chr(31)||v_material_sha,'UTF8')),'hex');
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion(
    solicitud_ref,persona_ref,contexto_actor_ref,contexto_actor_version,
    convocatoria_ref,convocatoria_id,secuencia,version_sha256,
    identificador_publico,categoria_ref,bases_ref,catalogo_ref,catalogo_version,catalogo_sha256,
    politica_catalogo_ref,politica_catalogo_version,politica_catalogo_sha256,
    unidad_ref,ambito_ref,ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,
    formulario_ref,formulario_version,formulario_sha256,plazo_ref,plazo_abre_en,plazo_cierra_en,
    requisitos,requisitos_sha256,declaraciones,declaracion_ref,
    clave_sha256,material_sha256,canal,presentada_en)
   VALUES(v_solicitud,v_persona,contexto->>'contexto_actor_ref',
    (contexto->>'contexto_version')::bigint,v_convocatoria,publicacion->>'convocatoria_id',
    (publicacion->>'secuencia')::bigint,publicacion->>'version_sha256',
    publicacion->>'identificador_publico',v_categorias[i],publicacion->>'bases_ref',
    publicacion->>'catalogo_ref',(publicacion->>'catalogo_version')::integer,
    publicacion->>'catalogo_sha256',publicacion->>'politica_catalogo_ref',
    (publicacion->>'politica_catalogo_version')::integer,publicacion->>'politica_catalogo_sha256',
    publicacion->>'unidad_ref',publicacion->>'ambito_ref',publicacion->>'fuente_ref',
    (publicacion->>'fuente_version')::integer,publicacion->>'fuente_sha256',
    publicacion->>'formulario_ref',
    (publicacion->>'formulario_version')::integer,publicacion->>'formulario_sha256',
    publicacion->>'plazo_ref',(publicacion->>'plazo_abre_en')::timestamptz,
    (publicacion->>'plazo_cierra_en')::timestamptz,v_evaluacion,
    publicacion->>'requisitos_sha256',m->'declaraciones',v_declaracion_ref,
    v_clave_sha,v_material_sha,v_canal,v_instante);
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
    solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
    consumo_huella_sha256,auditoria_ref,aplicada_en)
   VALUES(v_solicitud,1,'pendiente',v_persona,p_captura_actor->>'perfil_ref',contexto->>'cuenta_ref',
    v_evaluacion,v_decisiones[i],v_consumos[i],v_auditorias[i],v_instante);
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
    historia_ref,solicitud_ref,version,accion,actor_ref,estado,
    unidad_ref,ambito_ref,ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,
    registrada_en)
   VALUES(v_historia,v_solicitud,1,'presentar',v_persona,'pendiente',
    publicacion->>'unidad_ref',publicacion->>'ambito_ref',publicacion->>'fuente_ref',
    (publicacion->>'fuente_version')::integer,publicacion->>'fuente_sha256',v_instante);
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
    evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
   VALUES(v_evento,v_historia,v_solicitud,1,'vec.bolsa.inscripcion.presentada.v1',
    jsonb_build_object('solicitud_ref',v_solicitud,'version',1,'convocatoria_ref',v_convocatoria,
      'categoria_ref',v_categorias[i],'estado','pendiente'),v_instante);
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
    recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
    historia_ref,evento_ref,auditoria_ref,emitida_en)
   VALUES('recibo_inscripcion_'||encode(sha256(convert_to(v_solicitud||chr(31)||v_material_sha,'UTF8')),'hex'),
    v_solicitud,1,v_persona,v_canal,v_clave_sha,v_material_sha,
    v_historia,v_evento,v_auditorias[i],v_instante);
   INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
    acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
   VALUES('acceso_inscripcion_'||encode(sha256(convert_to(v_decisiones[i],'UTF8')),'hex'),
    v_solicitud,v_persona,'presentar',v_decisiones[i],v_auditorias[i],'confirmada',v_instante);
   v_salida:=v_salida||jsonb_build_array(jsonb_build_object('categoria_ref',v_categorias[i],
    'solicitud_ref',v_solicitud,
    'recibo_ref','recibo_inscripcion_'||encode(sha256(convert_to(v_solicitud||chr(31)||v_material_sha,'UTF8')),'hex'),
    'declaracion_ref',v_declaracion_ref));
  END LOOP;
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_conjunta(
   presentacion_ref,persona_ref,convocatoria_ref,categorias,canal,clave_sha256,material_sha256,
   recibo_ref,presentada_en)
  VALUES(v_presentacion_ref,v_persona,v_convocatoria,m->'categorias_refs',v_canal,v_clave_sha,
   v_material_sha,v_recibo_ref,v_instante);
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria(
   presentacion_ref,orden,categoria_ref,solicitud_ref)
  SELECT v_presentacion_ref,x.orden::integer,x.categoria,v_solicitudes[x.orden]
  FROM unnest(v_categorias) WITH ORDINALITY AS x(categoria,orden);
 END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  publicacion->>'catalogo_ref',(publicacion->>'catalogo_version')::integer,
  publicacion->>'catalogo_sha256',v_categorias,p_captura_actor->>'idioma');
 IF jsonb_typeof(etiquetas) IS DISTINCT FROM 'array' OR jsonb_array_length(etiquetas)<>v_n
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(etiquetas) WITH ORDINALITY AS x(valor,orden)
   WHERE x.valor->>'categoria_ref' IS DISTINCT FROM v_categorias[x.orden]
    OR coalesce(x.valor->>'categoria','')='')
 THEN RAISE EXCEPTION 'B146: etiqueta de categoría no disponible' USING ERRCODE='B9601'; END IF;
 SELECT jsonb_agg(x.valor||jsonb_build_object('categoria',etiquetas->(x.orden::integer-1)->>'categoria',
   'auditoria_ref',v_auditorias[x.orden]) ORDER BY x.orden)
 INTO v_salida FROM jsonb_array_elements(v_salida) WITH ORDINALITY AS x(valor,orden);
 RETURN jsonb_build_object('presentacion_ref',v_presentacion_ref,'recibo_ref',v_recibo_ref,
  'convocatoria_ref',v_convocatoria,'bases_ref',publicacion->>'bases_ref',
  'estado','pendiente','version',1,'repetida',v_existente.presentacion_ref IS NOT NULL,
  'registrada_en',v_instante,'categorias',v_salida);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(
 text,jsonb,bytea[],bytea[],bytea[],bytea[],bytea[],numeric[],numeric[],bytea[],bytea[],bytea[],bytea[])
 FROM PUBLIC,vec_bolsa_llamamientos_portal_externo,vec_bolsa_llamamientos_ejecutor;
-- Como la solicitud por categoría: sólo el portal externo presenta.
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(
 text,jsonb,bytea[],bytea[],bytea[],bytea[],bytea[],numeric[],numeric[],bytea[],bytea[],bytea[],bytea[])
 TO vec_bolsa_llamamientos_portal_externo;

DO $post$
DECLARE r record; f regprocedure; p record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('solicitar','vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','21b31e8b6e2786f4c4cc380b593d2a47392d3218985d91c1cb5a4919c85ca395','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}'),
   ('proyectar','vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean,text)','94ddb56c8cf79e9a343648d1b3b4fdc721ebf224eb14b299a2b5929f36dda484','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'),
   ('conjunta','vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(text,jsonb,bytea[],bytea[],bytea[],bytea[],bytea[],numeric[],numeric[],bytea[],bytea[],bytea[],bytea[])','0a319689dfa3632c5bca5fe861d5882b8fb4916b1456769b170ece6e1e8eeec1','{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_portal_externo=X/vec_bolsa_llamamientos_propietario}')
  ) AS esperado(clave,firma,huella,acl) LOOP
  f:=to_regprocedure(r.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'B146: funcion ausente' USING ERRCODE='55000'; END IF;
  SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_llamamientos_propietario'
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile::text<>'v' OR p.proparallel::text<>'u'
  OR p.proacl::text IS DISTINCT FROM r.acl
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>r.huella
  THEN RAISE EXCEPTION 'B146: definicion o metadatos divergentes: %',r.clave USING ERRCODE='55000'; END IF;
  IF r.clave<>'conjunta' AND f::oid::text IS DISTINCT FROM current_setting('vec.b146.oid_'||r.clave,true)
  THEN RAISE EXCEPTION 'B146: OID divergente: %',r.clave USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion'::regclass
    AND c.conname='solicitud_inscripcion_persona_ref_convocatoria_ref_key')
 THEN RAISE EXCEPTION 'B146: claves divergentes' USING ERRCODE='55000'; END IF;
 -- Tablas nuevas: propietario, RLS habilitada y forzada, una sola política y
 -- ningún privilegio fuera del propietario.
 IF (SELECT count(*) FROM pg_catalog.pg_class c
     WHERE c.oid IN('vec_bolsa_llamamientos.solicitud_inscripcion_conjunta'::regclass,
       'vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria'::regclass)
      AND pg_get_userbyid(c.relowner)='vec_bolsa_llamamientos_propietario'
      AND c.relrowsecurity AND c.relforcerowsecurity
      AND (c.relacl IS NULL OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a
        WHERE a.grantee<>c.relowner))
      AND (SELECT count(*) FROM pg_catalog.pg_policy pol WHERE pol.polrelid=c.oid)=1
      AND EXISTS(SELECT 1 FROM pg_catalog.pg_policy pol WHERE pol.polrelid=c.oid
        AND pol.polname='propietario_inscripcion' AND pol.polcmd='*' AND pol.polpermissive
        AND pol.polroles=ARRAY[to_regrole('vec_bolsa_llamamientos_propietario')::oid]))<>2
 OR EXISTS(SELECT 1 FROM (VALUES('public'),('vec_bolsa_llamamientos_portal_externo'),('vec_bolsa_llamamientos_ejecutor')) roles(rol),
   (VALUES('vec_bolsa_llamamientos.solicitud_inscripcion_conjunta'),
    ('vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria')) t(tabla)
   WHERE has_table_privilege(roles.rol,t.tabla,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 THEN RAISE EXCEPTION 'B146: permisos de las tablas nuevas divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
