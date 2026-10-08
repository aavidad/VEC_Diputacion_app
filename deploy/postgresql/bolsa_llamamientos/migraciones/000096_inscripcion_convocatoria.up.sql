\set ON_ERROR_STOP on
-- B96: solicitud propia y revisión administrativa de una convocatoria publicada.
-- No crea participación, orden ni vínculo B8. La incorporación se gobierna en B97.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000096',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)') IS NULL
 OR NOT has_function_privilege(current_user,
   'vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(text,text)','EXECUTE')
 OR to_regprocedure('vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)') IS NULL
 OR NOT has_function_privilege(current_user,
   'vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(text,integer,text,text[],text)','EXECUTE')
 OR to_regprocedure('vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb)') IS NULL
 OR NOT has_function_privilege(current_user,
   'vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(jsonb)','EXECUTE')
 OR to_regprocedure('vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(text,integer,text,text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(text,integer,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones_empleado') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones_rrhh') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion') IS NOT NULL
 THEN RAISE EXCEPTION 'B96: dependencias o preimagen incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion (
 solicitud_ref text COLLATE "C" PRIMARY KEY CHECK(solicitud_ref~'^solicitud_inscripcion_[0-9a-f]{64}$'),
 persona_ref text COLLATE "C" NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 contexto_actor_ref text COLLATE "C" NOT NULL CHECK(contexto_actor_ref~'^vca_[A-Za-z0-9_-]{22,128}$'),
 contexto_actor_version bigint NOT NULL CHECK(contexto_actor_version BETWEEN 1 AND 9007199254740991),
 convocatoria_ref text COLLATE "C" NOT NULL CHECK(convocatoria_ref~'^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'),
 convocatoria_id text COLLATE "C" NOT NULL CHECK(octet_length(convocatoria_id) BETWEEN 1 AND 480),
 secuencia bigint NOT NULL CHECK(secuencia BETWEEN 1 AND 9007199254740991),
 version_sha256 text NOT NULL CHECK(version_sha256~'^[0-9a-f]{64}$'),
 identificador_publico text COLLATE "C" NOT NULL CHECK(octet_length(identificador_publico) BETWEEN 1 AND 512),
 categoria_ref text COLLATE "C" NOT NULL CHECK(octet_length(categoria_ref) BETWEEN 1 AND 200),
 bases_ref text COLLATE "C" NOT NULL CHECK(octet_length(bases_ref) BETWEEN 1 AND 512),
 catalogo_ref text COLLATE "C" NOT NULL CHECK(octet_length(catalogo_ref) BETWEEN 1 AND 512),
 catalogo_version integer NOT NULL CHECK(catalogo_version>0),
 catalogo_sha256 text NOT NULL CHECK(catalogo_sha256~'^[0-9a-f]{64}$'),
 politica_catalogo_ref text COLLATE "C" NOT NULL CHECK(octet_length(politica_catalogo_ref) BETWEEN 1 AND 512),
 politica_catalogo_version integer NOT NULL CHECK(politica_catalogo_version>0),
 politica_catalogo_sha256 text NOT NULL CHECK(politica_catalogo_sha256~'^[0-9a-f]{64}$'),
 formulario_ref text COLLATE "C" NOT NULL CHECK(octet_length(formulario_ref) BETWEEN 1 AND 512),
 formulario_version integer NOT NULL CHECK(formulario_version>0),
 formulario_sha256 text NOT NULL CHECK(formulario_sha256~'^[0-9a-f]{64}$'),
 plazo_ref text COLLATE "C" NOT NULL CHECK(octet_length(plazo_ref) BETWEEN 1 AND 512),
 plazo_abre_en timestamptz(6) NOT NULL,
 plazo_cierra_en timestamptz(6) NOT NULL,
 requisitos jsonb NOT NULL CHECK(jsonb_typeof(requisitos)='array' AND jsonb_array_length(requisitos)<=256),
 requisitos_sha256 text NOT NULL CHECK(requisitos_sha256~'^[0-9a-f]{64}$'),
 declaraciones jsonb NOT NULL CHECK(jsonb_typeof(declaraciones)='array' AND jsonb_array_length(declaraciones)<=32),
 declaracion_ref text NOT NULL UNIQUE CHECK(declaracion_ref~'^declaracion_inscripcion_[0-9a-f]{64}$'),
 clave_sha256 text NOT NULL CHECK(clave_sha256~'^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 canal text NOT NULL CHECK(canal='externa_personal'),
 presentada_en timestamptz(6) NOT NULL CHECK(isfinite(presentada_en)),
 UNIQUE(persona_ref,convocatoria_ref),
 UNIQUE(persona_ref,convocatoria_ref,categoria_ref,canal,clave_sha256),
 CHECK(plazo_abre_en<=presentada_en AND presentada_en<plazo_cierra_en)
);
CREATE INDEX solicitud_inscripcion_por_convocatoria
 ON vec_bolsa_llamamientos.solicitud_inscripcion(convocatoria_ref,categoria_ref,presentada_en,solicitud_ref);
CREATE INDEX solicitud_inscripcion_por_persona
 ON vec_bolsa_llamamientos.solicitud_inscripcion(persona_ref,presentada_en,solicitud_ref);

CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_version (
 solicitud_ref text COLLATE "C" NOT NULL REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 9007199254740991),
 estado text NOT NULL CHECK(estado IN('pendiente','admitida_a_convocatoria','rechazada','incorporada')),
 actor_ref text COLLATE "C" NOT NULL CHECK(actor_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 perfil_ref text COLLATE "C" NOT NULL CHECK(perfil_ref~'^prf_[A-Za-z0-9_-]{22,128}$'),
 cuenta_ref text COLLATE "C" NOT NULL CHECK(cuenta_ref~'^cta_[A-Za-z0-9_-]{22,128}$'),
 motivo_ref text COLLATE "C",
 motivo_catalogo_ref text COLLATE "C",
 motivo_catalogo_version integer,
 motivo_catalogo_sha256 text,
 evaluacion jsonb NOT NULL CHECK(jsonb_typeof(evaluacion)='array' AND jsonb_array_length(evaluacion)<=256),
 decision_ref text COLLATE "C" NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ref text COLLATE "C" NOT NULL UNIQUE,
 aplicada_en timestamptz(6) NOT NULL CHECK(isfinite(aplicada_en)),
 PRIMARY KEY(solicitud_ref,version),
 CHECK((version=1 AND estado='pendiente' AND motivo_ref IS NULL AND motivo_catalogo_ref IS NULL
        AND motivo_catalogo_version IS NULL AND motivo_catalogo_sha256 IS NULL)
    OR (version>1 AND ((estado='rechazada' AND motivo_ref IS NOT NULL
                      AND motivo_catalogo_ref IS NOT NULL AND motivo_catalogo_version>0
                      AND motivo_catalogo_sha256~'^[0-9a-f]{64}$')
                      OR (estado IN('admitida_a_convocatoria','incorporada') AND motivo_ref IS NULL
                         AND motivo_catalogo_ref IS NULL AND motivo_catalogo_version IS NULL
                         AND motivo_catalogo_sha256 IS NULL))))
);
CREATE INDEX solicitud_inscripcion_version_actual
 ON vec_bolsa_llamamientos.solicitud_inscripcion_version(solicitud_ref,version DESC);

CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_historia (
 historia_ref text PRIMARY KEY CHECK(historia_ref~'^historia_inscripcion_[0-9a-f]{64}$'),
 solicitud_ref text NOT NULL,
 version bigint NOT NULL,
 accion text NOT NULL CHECK(accion IN('presentar','revisar')),
 actor_ref text NOT NULL,
 estado text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(solicitud_ref,version),
 UNIQUE(historia_ref,solicitud_ref,version),
 FOREIGN KEY(solicitud_ref,version) REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_version(solicitud_ref,version)
);
CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_outbox (
 evento_ref text PRIMARY KEY CHECK(evento_ref~'^evento_inscripcion_[0-9a-f]{64}$'),
 historia_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_historia(historia_ref),
 solicitud_ref text NOT NULL,
 version bigint NOT NULL,
 esquema text NOT NULL CHECK(esquema IN('vec.bolsa.inscripcion.presentada.v1','vec.bolsa.inscripcion.revisada.v1')),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(solicitud_ref,version),
 UNIQUE(evento_ref,solicitud_ref,version),
 FOREIGN KEY(historia_ref,solicitud_ref,version)
  REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_historia(historia_ref,solicitud_ref,version)
);
CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_recibo (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref~'^recibo_inscripcion_[0-9a-f]{64}$'),
 solicitud_ref text NOT NULL,
 version bigint NOT NULL,
 persona_ref text NOT NULL,
 canal text NOT NULL,
 clave_sha256 text NOT NULL,
 material_sha256 text NOT NULL,
 historia_ref text NOT NULL UNIQUE,
 evento_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 emitida_en timestamptz(6) NOT NULL,
 UNIQUE(solicitud_ref,version),
 UNIQUE(solicitud_ref,canal,clave_sha256),
 FOREIGN KEY(solicitud_ref,version) REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_version(solicitud_ref,version),
 FOREIGN KEY(historia_ref,solicitud_ref,version)
  REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_historia(historia_ref,solicitud_ref,version),
 FOREIGN KEY(evento_ref,solicitud_ref,version)
  REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_outbox(evento_ref,solicitud_ref,version)
);
CREATE TABLE vec_bolsa_llamamientos.solicitud_inscripcion_acceso (
 acceso_ref text PRIMARY KEY CHECK(acceso_ref~'^acceso_inscripcion_[0-9a-f]{64}$'),
 solicitud_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 persona_ref text NOT NULL,
 accion text NOT NULL CHECK(accion IN('presentar','revisar')),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 resultado text NOT NULL CHECK(resultado IN('confirmada','repetida','conflicto')),
 accedida_en timestamptz(6) NOT NULL
);

DO $cerrar$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['solicitud_inscripcion','solicitud_inscripcion_version',
   'solicitud_inscripcion_historia','solicitud_inscripcion_outbox',
   'solicitud_inscripcion_recibo','solicitud_inscripcion_acceso'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_llamamientos_ejecutor',tabla);
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_llamamientos.%I FROM PUBLIC',tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_inscripcion ON vec_bolsa_llamamientos.%I FOR ALL TO vec_bolsa_llamamientos_propietario USING(current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK(current_user=''vec_bolsa_llamamientos_propietario'')',tabla);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',tabla);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',tabla);
 END LOOP;
END $cerrar$;

CREATE FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(
 p_material text,p_captura_actor jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; publicacion jsonb; etiquetas jsonb;
 v_evaluacion jsonb; consumo record;
 v_solicitud_ref text; v_clave_sha text; v_material_sha text; v_recurso text; v_recurso_sha text;
 v_persona text; v_categoria text; v_categoria_etiqueta text; v_convocatoria text; v_correlacion text;
 v_existente vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
 v_recibo vec_bolsa_llamamientos.solicitud_inscripcion_recibo%ROWTYPE;
 v_historia text; v_evento text; v_recibo_ref text; v_declaracion_ref text;
 v_declaracion jsonb;
 v_recibo_json jsonb; v_instante timestamptz(6);
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_externo_bolsa_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 16384
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
 THEN RAISE EXCEPTION 'B96: presentación no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  contexto:=convert_from(p_contexto,'UTF8')::jsonb;
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
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$'
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
 v_correlacion:=d->>'correlacion_ref';
 IF coalesce(v_persona,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_persona
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'externa_personal'
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
 OR d->>'tipo_recurso' IS DISTINCT FROM 'inscripcion_convocatoria'
 OR d->>'finalidad' IS DISTINCT FROM 'presentar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.presentar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.presentar.v1'
 THEN RAISE EXCEPTION 'B96: actor o concesión divergente' USING ERRCODE='42501'; END IF;
 v_solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
   v_persona||chr(31)||v_convocatoria||chr(31)||v_categoria,'UTF8')),'hex');
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 v_recurso:='{"ambitos":{"categoria_ref":'||to_json(v_categoria)::text||
  ',"convocatoria_ref":'||to_json(v_convocatoria)::text||
  '},"atributos":{"material_sha256":"'||v_material_sha||'"}}';
 v_recurso_sha:=encode(sha256(convert_to(v_recurso,'UTF8')),'hex');
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

 SELECT * INTO v_existente FROM vec_bolsa_llamamientos.solicitud_inscripcion
 WHERE solicitud_ref=v_solicitud_ref;
 IF FOUND THEN
  IF v_existente.persona_ref IS DISTINCT FROM v_persona
  OR v_existente.convocatoria_ref IS DISTINCT FROM v_convocatoria
  OR v_existente.categoria_ref IS DISTINCT FROM v_categoria
  OR v_existente.clave_sha256 IS DISTINCT FROM v_clave_sha
  OR v_existente.material_sha256 IS DISTINCT FROM v_material_sha
  OR v_existente.canal IS DISTINCT FROM 'externa_personal'
  THEN
   IF v_existente.clave_sha256 IS DISTINCT FROM v_clave_sha THEN
    RAISE EXCEPTION 'B96: ya existe solicitud para esta persona y convocatoria' USING ERRCODE='B9604';
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
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  WHERE s.persona_ref=v_persona AND s.convocatoria_ref=v_convocatoria) THEN
  RAISE EXCEPTION 'B96: ya existe solicitud para esta convocatoria' USING ERRCODE='B9604';
 END IF;

 -- Sólo la fuente gobernada de Bolsa determina versión/plazo/catálogo.
 publicacion:=vec_bolsa_convocatorias.comprobar_publicacion_inscripcion_v1(v_convocatoria,v_categoria);
 IF publicacion->>'catalogo_version' IS DISTINCT FROM m->>'catalogo_version'
 OR publicacion->>'categoria_ref' IS DISTINCT FROM v_categoria
 OR publicacion->>'convocatoria_ref' IS DISTINCT FROM v_convocatoria
 OR jsonb_typeof(publicacion->'requisitos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B96: catálogo o publicación divergente' USING ERRCODE='B9601'; END IF;
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
  publicacion->>'politica_catalogo_sha256') IS NOT TRUE
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
  publicacion->>'formulario_ref',
  (publicacion->>'formulario_version')::integer,publicacion->>'formulario_sha256',
  publicacion->>'plazo_ref',(publicacion->>'plazo_abre_en')::timestamptz,
  (publicacion->>'plazo_cierra_en')::timestamptz,v_evaluacion,
  publicacion->>'requisitos_sha256',m->'declaraciones',v_declaracion_ref,
  v_clave_sha,v_material_sha,
  'externa_personal',v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(v_solicitud_ref,1,'pendiente',v_persona,d->>'perfil_activo_ref',contexto->>'cuenta_ref',
  v_evaluacion,
  consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
  historia_ref,solicitud_ref,version,accion,actor_ref,estado,registrada_en)
 VALUES(v_historia,v_solicitud_ref,1,'presentar',v_persona,'pendiente',v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
  evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_solicitud_ref,1,'vec.bolsa.inscripcion.presentada.v1',
  jsonb_build_object('solicitud_ref',v_solicitud_ref,'version',1,'convocatoria_ref',v_convocatoria,
    'categoria_ref',v_categoria,'estado','pendiente'),v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
  recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
  historia_ref,evento_ref,auditoria_ref,emitida_en)
 VALUES(v_recibo_ref,v_solicitud_ref,1,v_persona,'externa_personal',v_clave_sha,v_material_sha,
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
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_portal_externo;

CREATE FUNCTION vec_bolsa_llamamientos.revisar_inscripcion_v1(
 p_material text,p_captura_actor jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; motivos jsonb; motivo jsonb;
 etiquetas jsonb; consumo record; solicitud vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
 actual vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 v_revision_version vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 previo vec_bolsa_llamamientos.solicitud_inscripcion_recibo%ROWTYPE;
 v_ref text; v_actor text; v_material_sha text; v_clave_sha text;
 v_recurso text; v_recurso_sha text; v_estado text; v_categoria text;
 v_historia text; v_evento text; v_recibo_ref text; v_instante timestamptz(6);
 v_esperada bigint; v_nueva bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_bolsa_llamamientos_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B96: revisión no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: material de revisión inválido' USING ERRCODE='B9605';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
  'clave_idempotencia','decision','esquema','motivo_codigo','solicitud_ref','version_esperada']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.decidir.v1'
 OR coalesce(m->>'solicitud_ref','') !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR jsonb_typeof(m->'decision') IS DISTINCT FROM 'string'
 OR (m->>'decision' IN('admitir','rechazar')) IS NOT TRUE
 OR jsonb_typeof(m->'motivo_codigo') IS DISTINCT FROM 'string'
 OR (m->>'decision'='admitir' AND m->>'motivo_codigo'<>'')
 OR (m->>'decision'='rechazar' AND coalesce(m->>'motivo_codigo','') !~ '^[a-z][a-z0-9_.:-]{2,127}$')
 OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
 OR coalesce(m->>'version_esperada','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$'
 THEN RAISE EXCEPTION 'B96: decisión inválida' USING ERRCODE='B9605'; END IF;
 v_ref:=m->>'solicitud_ref';
 v_actor:=d->>'principal_id';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 v_esperada:=(m->>'version_esperada')::bigint;
 v_recurso:='{"ambitos":{"solicitud_ref":'||to_json(v_ref)::text||
  '},"atributos":{"material_sha256":"'||v_material_sha||'"}}';
 v_recurso_sha:=encode(sha256(convert_to(v_recurso,'UTF8')),'hex');
 IF coalesce(v_actor,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_actor
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_captura_actor->>'idioma' IS NULL
 OR p_captura_actor->>'idioma' NOT IN('es','en')
 OR contexto->>'principal_ref' IS DISTINCT FROM v_actor
 OR contexto->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.decidir'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitud_inscripcion'
 OR d->>'finalidad' IS DISTINCT FROM 'revisar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.decidir'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.revisar.v1'
 OR d->>'recurso_ref' IS DISTINCT FROM v_ref
 OR c->>'efecto_ref' IS DISTINCT FROM v_ref
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B96: revisión sin decisión exacta' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:inscripcion:'||v_ref,0));
 -- Validar y consumir V3 antes de consultar la solicitud impide que un LOGIN
 -- técnico con material sin firma sondee existencia, estado o autoría. Toda
 -- denegación posterior revierte este consumo dentro de la misma transacción.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B96: consumo de revisión incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO solicitud FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_ref FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'B96: solicitud no encontrada' USING ERRCODE='B9604'; END IF;
 IF v_actor IS NOT DISTINCT FROM solicitud.persona_ref THEN
  RAISE EXCEPTION 'B96: el solicitante no puede revisar su solicitud' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT actual FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
  WHERE solicitud_ref=v_ref ORDER BY version DESC LIMIT 1 FOR SHARE;
 IF actual.version>v_esperada THEN
  SELECT * INTO previo FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo
   WHERE solicitud_ref=v_ref AND version=v_esperada+1;
  IF NOT FOUND OR previo.clave_sha256 IS DISTINCT FROM v_clave_sha
   OR previo.material_sha256 IS DISTINCT FROM v_material_sha
   OR previo.canal IS DISTINCT FROM 'interna_corporativa'
  THEN RAISE EXCEPTION 'B96: versión de decisión superada' USING ERRCODE='B9604'; END IF;
  SELECT * INTO STRICT v_revision_version
   FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
   WHERE solicitud_ref=v_ref AND version=v_esperada+1;
  IF v_revision_version.actor_ref IS DISTINCT FROM v_actor
   OR v_revision_version.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref'
   OR v_revision_version.cuenta_ref IS DISTINCT FROM contexto->>'cuenta_ref'
  THEN RAISE EXCEPTION 'B96: recibo de revisión ajeno' USING ERRCODE='B9604'; END IF;
  etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   solicitud.catalogo_ref,solicitud.catalogo_version,solicitud.catalogo_sha256,
   ARRAY[solicitud.categoria_ref],p_captura_actor->>'idioma');
  v_categoria:=etiquetas#>>'{0,categoria}';
  IF v_revision_version.estado='rechazada' THEN
   motivo:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
    v_revision_version.motivo_catalogo_ref,v_revision_version.motivo_catalogo_version,
    v_revision_version.motivo_catalogo_sha256,ARRAY[v_revision_version.motivo_ref],
    p_captura_actor->>'idioma')->0;
  END IF;
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
   acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
  VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
   v_ref,v_actor,'revisar',consumo.decision_ref,consumo.auditoria_ref,'repetida',consumo.consumida_en);
  RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',previo.recibo_ref,
   'convocatoria_ref',solicitud.convocatoria_ref,'categoria_ref',solicitud.categoria_ref,
   'categoria',v_categoria,'bases_ref',solicitud.bases_ref,'estado',v_revision_version.estado,
   'declaracion_ref',solicitud.declaracion_ref,
   'version',v_revision_version.version,'registrada_en',solicitud.presentada_en,
   'decidida_en',v_revision_version.aplicada_en,'repetida',true,'auditoria_ref',consumo.auditoria_ref,
   'motivo_codigo',v_revision_version.motivo_ref,
   'motivo_etiqueta',motivo->>'categoria');
 END IF;
 IF actual.version IS DISTINCT FROM v_esperada OR actual.estado<>'pendiente'
 THEN RAISE EXCEPTION 'B96: versión de decisión superada' USING ERRCODE='B9604'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(actual.evaluacion) AS r(valor)
   WHERE r.valor->>'obligatorio' IS DISTINCT FROM 'false'
     AND r.valor->>'estado' IS DISTINCT FROM 'cumple')
   AND m->>'decision'='admitir'
 THEN RAISE EXCEPTION 'B96: requisito obligatorio pendiente' USING ERRCODE='B9606'; END IF;
 v_estado:=CASE m->>'decision' WHEN 'admitir' THEN 'admitida_a_convocatoria' ELSE 'rechazada' END;
 IF v_estado='rechazada' THEN
  motivos:=vec_catalogos_configurables.listar_motivos_inscripcion_v1(p_captura_actor->>'idioma');
  motivo:=vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(
   m->>'motivo_codigo',(motivos->>'catalogo_version')::integer,
   motivos->>'catalogo_sha256',p_captura_actor->>'idioma');
 END IF;
 v_nueva:=actual.version+1;
 v_instante:=consumo.consumida_en;
 v_historia:='historia_inscripcion_'||encode(sha256(convert_to(v_ref||':'||v_nueva,'UTF8')),'hex');
 v_evento:='evento_inscripcion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
 v_recibo_ref:='recibo_inscripcion_'||encode(sha256(convert_to(v_ref||chr(31)||v_nueva||chr(31)||v_material_sha,'UTF8')),'hex');
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  solicitud.catalogo_ref,solicitud.catalogo_version,solicitud.catalogo_sha256,
  ARRAY[solicitud.categoria_ref],p_captura_actor->>'idioma');
 v_categoria:=etiquetas#>>'{0,categoria}';
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,
  motivo_ref,motivo_catalogo_ref,
  motivo_catalogo_version,motivo_catalogo_sha256,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(v_ref,v_nueva,v_estado,v_actor,d->>'perfil_activo_ref',contexto->>'cuenta_ref',
  CASE WHEN v_estado='rechazada' THEN m->>'motivo_codigo' END,
  CASE WHEN v_estado='rechazada' THEN motivos->>'catalogo_ref' END,
  CASE WHEN v_estado='rechazada' THEN (motivos->>'catalogo_version')::integer END,
  CASE WHEN v_estado='rechazada' THEN motivos->>'catalogo_sha256' END,
  actual.evaluacion,consumo.decision_ref,consumo.consumo_huella_sha256,
  consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
  historia_ref,solicitud_ref,version,accion,actor_ref,estado,registrada_en)
 VALUES(v_historia,v_ref,v_nueva,'revisar',v_actor,v_estado,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
  evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_ref,v_nueva,'vec.bolsa.inscripcion.revisada.v1',
  jsonb_build_object('solicitud_ref',v_ref,'version',v_nueva,'estado',v_estado),v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
  recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
  historia_ref,evento_ref,auditoria_ref,emitida_en)
 VALUES(v_recibo_ref,v_ref,v_nueva,solicitud.persona_ref,'interna_corporativa',
  v_clave_sha,v_material_sha,v_historia,v_evento,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
  acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
 VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
  v_ref,v_actor,'revisar',consumo.decision_ref,consumo.auditoria_ref,'confirmada',v_instante);
 RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',v_recibo_ref,
  'convocatoria_ref',solicitud.convocatoria_ref,'categoria_ref',solicitud.categoria_ref,
  'categoria',v_categoria,'bases_ref',solicitud.bases_ref,'estado',v_estado,
  'declaracion_ref',solicitud.declaracion_ref,
  'version',v_nueva,'registrada_en',solicitud.presentada_en,'decidida_en',v_instante,
  'repetida',false,'auditoria_ref',consumo.auditoria_ref,
  'motivo_codigo',CASE WHEN v_estado='rechazada' THEN m->>'motivo_codigo' END,
  'motivo_etiqueta',CASE WHEN v_estado='rechazada' THEN motivo->>'motivo_etiqueta' END);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.revisar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.revisar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
 p_fila jsonb,p_categoria text,p_motivo_etiqueta text,
 p_pendiente_etiqueta text,p_cumple_etiqueta text
) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET "TimeZone"='UTC' AS $f$
DECLARE v_requisitos jsonb;
BEGIN
 IF jsonb_typeof(p_fila) IS DISTINCT FROM 'object'
 OR p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 1 AND 200
 THEN RAISE EXCEPTION 'B96: proyección incompleta' USING ERRCODE='55000'; END IF;
 IF jsonb_typeof(p_fila->'evaluacion') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B96: evaluación ausente' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_fila->'evaluacion') AS r(valor)
   WHERE coalesce(r.valor->>'codigo',r.valor->>'referencia','')=''
      OR coalesce(r.valor->>'descripcion','')=''
      OR r.valor->>'estado' NOT IN('cumple','no_cumple','pendiente'))
 THEN RAISE EXCEPTION 'B96: requisito sin fuente publicable' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_fila->'evaluacion') AS r(valor)
   WHERE r.valor->>'estado'='pendiente'
     AND r.valor->>'motivo_codigo'='requisito.pendiente')
 AND (p_pendiente_etiqueta IS NULL OR octet_length(p_pendiente_etiqueta) NOT BETWEEN 1 AND 200)
 THEN RAISE EXCEPTION 'B96: motivo pendiente sin catálogo' USING ERRCODE='B9601'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_fila->'evaluacion') AS r(valor)
   WHERE r.valor->>'estado'='cumple')
 AND (p_cumple_etiqueta IS NULL OR octet_length(p_cumple_etiqueta) NOT BETWEEN 1 AND 200)
 THEN RAISE EXCEPTION 'B96: motivo de cumplimiento sin catálogo' USING ERRCODE='B9601'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'codigo',coalesce(r.valor->>'codigo',r.valor->>'referencia'),
  'descripcion',r.valor->>'descripcion',
  'obligatorio',(r.valor->>'obligatorio')::boolean,
  'estado',r.valor->>'estado',
  'motivo_codigo',coalesce(r.valor->>'motivo_codigo',''),
  'motivo_etiqueta',CASE WHEN r.valor->>'estado'='pendiente'
     AND r.valor->>'motivo_codigo'='requisito.pendiente' THEN p_pendiente_etiqueta
    WHEN r.valor->>'estado'='cumple' THEN p_cumple_etiqueta
    ELSE coalesce(r.valor->>'motivo_etiqueta','') END,
  'hito_cumplimiento',NULL,
  'hito_etiqueta',NULL,
  'procedencia_ref',r.valor->>'procedencia_ref') ORDER BY r.orden),'[]'::jsonb)
 INTO v_requisitos FROM jsonb_array_elements(p_fila->'evaluacion') WITH ORDINALITY AS r(valor,orden);
 RETURN jsonb_build_object(
  'solicitud_ref',p_fila->>'solicitud_ref',
  'recibo_ref',p_fila->>'recibo_ref',
  'convocatoria_ref',p_fila->>'convocatoria_ref',
  'categoria_ref',p_fila->>'categoria_ref',
  'categoria',p_categoria,
  'bases_ref',p_fila->>'bases_ref',
  'declaracion_ref',p_fila->>'declaracion_ref',
  'catalogo_version',(p_fila->>'catalogo_version')::bigint,
  'plazo_inicio',(p_fila->>'plazo_abre_en')::timestamptz,
  'plazo_fin',(p_fila->>'plazo_cierra_en')::timestamptz,
  'requisitos',v_requisitos,
  'estado',p_fila->>'estado',
  'version',(p_fila->>'version')::bigint,
  'registrada_en',(p_fila->>'presentada_en')::timestamptz,
  'decidida_en',CASE WHEN p_fila->>'estado'='pendiente' THEN NULL
    ELSE (p_fila->>'aplicada_en')::timestamptz END,
  'motivo_codigo',p_fila->>'motivo_ref',
  'motivo_etiqueta',p_motivo_etiqueta,
  'decision_ref',p_fila->>'decision_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(jsonb,text,text,text,text)
 FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
 p_propia boolean,p_persona_ref text,p_filtro jsonb,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 v_estado text; v_convocatoria text; v_cursor text; v_limite integer;
 v_cursor_en timestamptz(6); v_total bigint; v_raw jsonb; v_labels jsonb;
 v_peticiones jsonb; v_solicitudes jsonb; v_mas boolean; v_siguiente text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en')
 OR jsonb_typeof(p_filtro) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(p_filtro) ORDER BY 1) IS DISTINCT FROM
    ARRAY['convocatoria_ref','cursor','estado','limite']
 OR jsonb_typeof(p_filtro->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_filtro->>'limite','') !~ '^[1-9][0-9]{0,2}$'
 OR (p_filtro->>'limite')::integer NOT BETWEEN 1 AND 100
 OR jsonb_typeof(p_filtro->'cursor') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_filtro->'estado') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_filtro->'convocatoria_ref') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'B96: filtro de solicitudes inválido' USING ERRCODE='B9605'; END IF;
 v_estado:=p_filtro->>'estado';
 v_convocatoria:=p_filtro->>'convocatoria_ref';
 v_cursor:=p_filtro->>'cursor';
 v_limite:=(p_filtro->>'limite')::integer;
 IF v_estado NOT IN('','pendiente','admitida_a_convocatoria','rechazada','incorporada')
 OR (v_convocatoria<>'' AND (v_convocatoria !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
   OR octet_length(v_convocatoria)>200))
 OR (v_cursor<>'' AND v_cursor !~ '^solicitud_inscripcion_[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'B96: filtro de solicitudes inválido' USING ERRCODE='B9605'; END IF;
 IF v_cursor<>'' THEN
  SELECT presentada_en INTO v_cursor_en
  FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_cursor AND (NOT p_propia OR persona_ref=p_persona_ref);
  IF NOT FOUND THEN RAISE EXCEPTION 'B96: cursor no disponible' USING ERRCODE='B9605'; END IF;
 END IF;
 WITH filtradas AS MATERIALIZED (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
    ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  WHERE (NOT p_propia OR s.persona_ref=p_persona_ref)
   AND (v_estado='' OR v.estado=v_estado)
   AND (v_convocatoria='' OR s.convocatoria_ref=v_convocatoria)
 )
 SELECT (SELECT count(*) FROM filtradas),
  (SELECT coalesce(jsonb_agg(to_jsonb(q) ORDER BY q.presentada_en DESC,q.solicitud_ref DESC),'[]'::jsonb)
   FROM (SELECT * FROM filtradas
    WHERE v_cursor='' OR (presentada_en,solicitud_ref)<(v_cursor_en,v_cursor)
    ORDER BY presentada_en DESC,solicitud_ref DESC LIMIT v_limite+1) q)
 INTO v_total,v_raw;
 v_mas:=jsonb_array_length(v_raw)>v_limite;
 IF v_mas THEN
  SELECT coalesce(jsonb_agg(r.valor ORDER BY r.orden),'[]'::jsonb)
  INTO v_raw FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden)
  WHERE r.orden<=v_limite;
  v_siguiente:=v_raw->(jsonb_array_length(v_raw)-1)->>'solicitud_ref';
 END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'catalogo_ref',r.valor->>'catalogo_ref',
   'catalogo_version',(r.valor->>'catalogo_version')::integer,
   'catalogo_sha256',r.valor->>'catalogo_sha256',
   'categoria_ref',r.valor->>'categoria_ref',
   'politica_catalogo_ref',r.valor->>'politica_catalogo_ref',
   'politica_catalogo_version',(r.valor->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',r.valor->>'politica_catalogo_sha256') ORDER BY r.orden),'[]'::jsonb)
 INTO v_peticiones FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden);
 v_labels:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  v_peticiones,p_idioma);
 IF jsonb_array_length(v_labels)<>jsonb_array_length(v_raw) THEN
  RAISE EXCEPTION 'B96: proyección de catálogo incompleta' USING ERRCODE='55000'; END IF;
  SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
   r.valor,l.valor->>'categoria',NULL,l.valor->>'motivo_etiqueta_pendiente',
   l.valor->>'motivo_etiqueta_cumple') ORDER BY r.orden),'[]'::jsonb)
 INTO v_solicitudes
 FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden)
 JOIN jsonb_array_elements(v_labels) WITH ORDINALITY AS l(valor,orden) USING(orden);
 RETURN jsonb_build_object('solicitudes',v_solicitudes,'total',v_total,
  'cursor_siguiente',v_siguiente);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_lector_inscripciones;

CREATE FUNCTION vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
 p_propia boolean,p_persona_ref text,p_solicitud_ref text,p_idioma text
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE fila jsonb; etiquetas jsonb; motivo_etiqueta text; v_estado text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 THEN RAISE EXCEPTION 'B96: selector de solicitud inválido' USING ERRCODE='B9605'; END IF;
 SELECT to_jsonb(q) INTO fila FROM (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
   WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
   ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  WHERE s.solicitud_ref=p_solicitud_ref
   AND (NOT p_propia OR s.persona_ref=p_persona_ref)
 ) q;
 IF NOT FOUND THEN RETURN NULL; END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object(
   'catalogo_ref',fila->>'catalogo_ref',
   'catalogo_version',(fila->>'catalogo_version')::integer,
   'catalogo_sha256',fila->>'catalogo_sha256',
   'categoria_ref',fila->>'categoria_ref',
   'politica_catalogo_ref',fila->>'politica_catalogo_ref',
   'politica_catalogo_version',(fila->>'politica_catalogo_version')::integer,
   'politica_catalogo_sha256',fila->>'politica_catalogo_sha256')),p_idioma);
 IF jsonb_array_length(etiquetas)<>1 OR etiquetas#>>'{0,categoria}' IS NULL
 THEN RAISE EXCEPTION 'B96: etiqueta de categoría incompleta' USING ERRCODE='55000'; END IF;
 v_estado:=fila->>'estado';
 IF v_estado='rechazada' THEN
  motivo_etiqueta:=(vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
   fila->>'motivo_catalogo_ref',(fila->>'motivo_catalogo_version')::integer,
   fila->>'motivo_catalogo_sha256',ARRAY[fila->>'motivo_ref'],p_idioma))#>>'{0,categoria}';
 END IF;
 RETURN vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
  fila,etiquetas#>>'{0,categoria}',motivo_etiqueta,
  etiquetas#>>'{0,motivo_etiqueta_pendiente}',
  etiquetas#>>'{0,motivo_etiqueta_cumple}');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_lector_inscripciones;

CREATE FUNCTION vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
 p_fuente jsonb,p_persona_ref text,p_idioma text,p_metodo text,p_garantia text,p_lista boolean
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 v_items jsonb; v_item jsonb; v_categoria jsonb; v_requisito jsonb;
 v_selectores jsonb:='[]'::jsonb; v_etiquetas jsonb;
 v_categorias_pedidos jsonb; v_categorias_validas jsonb;
 v_propias jsonb; v_categorias jsonb; v_requisitos jsonb; v_bolsas jsonb:='[]'::jsonb;
 v_etiqueta jsonb; v_requisitos_resumen text; v_num_categorias integer; v_carta jsonb;
 v_estado text; v_puede boolean; v_catalogo_completo boolean; v_impedimento text;
 v_pos integer:=0; v_conv text; v_propia jsonb; v_salida jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en') OR p_lista IS NULL
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
  v_selectores,p_idioma);
 IF jsonb_typeof(v_etiquetas) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_etiquetas)<>jsonb_array_length(v_selectores)
 THEN RAISE EXCEPTION 'B96: lote de etiquetas incompleto' USING ERRCODE='55000'; END IF;
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
   v_categorias_pedidos);
  IF jsonb_typeof(v_categorias_validas) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_categorias_validas)<>jsonb_array_length(v_items)
  THEN RAISE EXCEPTION 'B96: cotejo de categorías incompleto' USING ERRCODE='55000'; END IF;
 END IF;
 SELECT coalesce(jsonb_object_agg(q.convocatoria_ref,
   jsonb_build_object('solicitud_ref',q.solicitud_ref,'estado',q.estado)),'{}'::jsonb)
 INTO v_propias FROM (
  SELECT s.convocatoria_ref,s.solicitud_ref,v.estado
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT estado FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY version DESC LIMIT 1) v ON true
  WHERE s.persona_ref=p_persona_ref
   AND s.convocatoria_ref IN (SELECT x.valor->>'convocatoria_ref' FROM jsonb_array_elements(v_items) AS x(valor))
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
  v_propia:=v_propias->v_conv;
  v_puede:=v_puede AND v_propia IS NULL;
  v_impedimento:=CASE WHEN v_propia IS NOT NULL
    THEN v_etiqueta->>'impedimento_etiqueta_existente'
    WHEN NOT v_puede THEN v_etiqueta->>'impedimento_etiqueta_politica'
    ELSE NULL END;
  IF NOT v_puede AND coalesce(v_impedimento,'')='' THEN
   RAISE EXCEPTION 'B96: impedimento sin catálogo' USING ERRCODE='B9601';
  END IF;
  v_carta:=jsonb_build_object(
   'convocatoria_ref',v_conv,'titulo',v_item->>'titulo',
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
   v_carta:=v_carta||jsonb_build_object('categorias',v_categorias,'requisitos',v_requisitos);
  END IF;
  v_bolsas:=v_bolsas||jsonb_build_array(v_carta);
 END LOOP;
 IF p_lista THEN
  v_salida:=jsonb_build_object('convocatorias',v_bolsas,'total',(p_fuente->>'total')::bigint,
   'cursor_siguiente',p_fuente->>'siguiente_cursor');
 ELSE v_salida:=v_bolsas->0; END IF;
 RETURN v_salida;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(jsonb,text,text,text,text,boolean)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_lector_inscripciones;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_inscripcion_v1(
 p_accion text,p_selector jsonb,p_contexto_canonico bytea,
 p_vinculo_canonico bytea,p_captura jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE
 v_contexto jsonb; v_vinculo jsonb; v_orden jsonb; v_proyeccion jsonb;
 v_resultado text:='obtenida'; v_persona text; v_canal text; v_idioma text;
 v_recurso text; v_recurso_esperado text; v_recurso_canonico text;
 v_filtro jsonb; v_f_estado text; v_f_convocatoria text; v_f_cursor text;
 v_f_limite integer; v_ref_selector text; v_prefijo text;
 v_motivos jsonb; v_motivos_proyeccion jsonb;
 v_ahora timestamptz(6); v_emitida_en timestamptz; v_valida_hasta timestamptz;
 v_auditoria record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user NOT IN('vec_bolsa_inscripciones_lector',
    'vec_bolsa_inscripciones_empleado_lector','vec_bolsa_inscripciones_rrhh_lector')
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_selector) IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_captura) IS DISTINCT FROM 'object'
 OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
 OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 THEN RAISE EXCEPTION 'B96: consulta no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
  v_ahora:=date_trunc('microseconds',clock_timestamp());
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: captura lectora inválida' USING ERRCODE='42501';
 END;
 v_persona:=p_captura->>'persona_ref';
 v_canal:=p_captura->>'canal';
 v_idioma:=p_captura->>'idioma';
 v_recurso:=p_captura->>'recurso_ref';
 IF jsonb_typeof(p_captura->'emitida_en') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p_captura->'valida_hasta') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'B96: ventana de captura ausente' USING ERRCODE='42501'; END IF;
 BEGIN
  v_emitida_en:=(p_captura->>'emitida_en')::timestamptz;
  v_valida_hasta:=(p_captura->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B96: ventana de captura inválida' USING ERRCODE='42501';
 END;
 IF v_persona IS NULL OR v_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura->>'accion' IS DISTINCT FROM p_accion
 OR p_captura->>'perfil_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR p_captura->>'perfil_ref' IS DISTINCT FROM v_vinculo->>'perfil_activo_ref'
 OR p_captura->>'persona_ref' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR p_captura->>'persona_ref' IS DISTINCT FROM v_vinculo->>'principal_id'
 OR p_captura->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR p_captura->>'cuenta_ref' IS DISTINCT FROM v_vinculo->>'cuenta_ref'
 OR p_captura->>'sesion_ref' IS DISTINCT FROM v_vinculo->>'sesion_ref'
 OR p_captura->>'autenticacion_ref' IS DISTINCT FROM v_vinculo->>'autenticacion_ref'
 OR p_captura->>'canal' IS DISTINCT FROM v_vinculo->>'superficie'
 OR v_idioma IS NULL OR v_idioma NOT IN('es','en')
 OR v_canal IS NULL OR v_canal NOT IN('externa_personal','interna_corporativa')
 OR coalesce(p_captura->>'revision_permisos','') !~ '^[1-9][0-9]{0,15}$'
 OR (v_emitida_en<=v_ahora AND v_valida_hasta>v_ahora) IS NOT TRUE
 OR NOT isfinite(v_emitida_en) OR NOT isfinite(v_valida_hasta)
 OR coalesce(p_captura->>'intento_ref','') !~ '^lectura_[0-9a-f]{32}$'
 OR p_captura->>'correlacion_ref' IS NULL
 OR p_captura->>'finalidad' IS NULL
 OR v_recurso IS NULL
 THEN RAISE EXCEPTION 'B96: sesión o captura no ligadas' USING ERRCODE='42501'; END IF;
 IF (
  (session_user='vec_bolsa_inscripciones_lector'
   AND v_canal='externa_personal'
   AND p_accion IN('bolsa.inscripcion.convocatorias.listar',
    'bolsa.inscripcion.convocatoria.consultar',
    'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar'))
  OR (session_user='vec_bolsa_inscripciones_empleado_lector'
   AND v_canal='interna_corporativa'
   AND p_accion IN('bolsa.inscripcion.convocatorias.listar',
    'bolsa.inscripcion.convocatoria.consultar',
    'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar'))
  OR (session_user='vec_bolsa_inscripciones_rrhh_lector'
   AND v_canal='interna_corporativa'
   AND p_accion IN('bolsa.inscripcion.rrhh.listar',
    'bolsa.inscripcion.rrhh.consultar','bolsa.inscripcion.rrhh.motivos'))
 ) IS NOT TRUE THEN RAISE EXCEPTION 'B96: LOGIN lector fuera de competencia' USING ERRCODE='42501'; END IF;

 -- Mismo canon que application/inscripcion.RecursoLectura: el recurso del
 -- permiso y de la auditoría se ata al selector, idioma y persona exactos.
 v_filtro:=p_captura->'filtro';
 IF jsonb_typeof(v_filtro) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(v_filtro) ORDER BY 1) IS DISTINCT FROM
  ARRAY['convocatoria_ref','cursor','estado','limite']
 OR jsonb_typeof(v_filtro->'estado') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'convocatoria_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'cursor') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v_filtro->'limite') IS DISTINCT FROM 'number'
 OR coalesce(v_filtro->>'limite','') !~ '^(0|[1-9][0-9]{0,2})$'
 THEN RAISE EXCEPTION 'B96: filtro de captura inválido' USING ERRCODE='42501'; END IF;
 v_f_estado:=v_filtro->>'estado';
 v_f_convocatoria:=v_filtro->>'convocatoria_ref';
 v_f_cursor:=v_filtro->>'cursor';
 v_f_limite:=(v_filtro->>'limite')::integer;
 IF p_accion IN('bolsa.inscripcion.convocatoria.consultar',
    'bolsa.inscripcion.propia.consultar','bolsa.inscripcion.rrhh.consultar',
    'bolsa.inscripcion.rrhh.motivos') THEN
  IF v_filtro IS DISTINCT FROM jsonb_build_object(
     'estado','','convocatoria_ref','','limite',0,'cursor','')
  THEN RAISE EXCEPTION 'B96: filtro de detalle divergente' USING ERRCODE='42501'; END IF;
 END IF;
 IF p_accion='bolsa.inscripcion.convocatoria.consultar' THEN
  v_ref_selector:=p_selector->>'convocatoria_ref';
  v_recurso_esperado:=v_ref_selector;
 ELSIF p_accion IN('bolsa.inscripcion.propia.consultar',
                   'bolsa.inscripcion.rrhh.consultar') THEN
  v_ref_selector:=p_selector->>'solicitud_ref';
  v_recurso_esperado:=v_ref_selector;
 ELSIF p_accion='bolsa.inscripcion.convocatorias.listar' THEN
  IF v_f_estado<>'' OR v_f_convocatoria<>''
   OR v_f_limite::text IS DISTINCT FROM p_selector->>'limite'
   OR v_f_cursor IS DISTINCT FROM p_selector->>'cursor'
  THEN RAISE EXCEPTION 'B96: lista abierta no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_abiertas_';
 ELSIF p_accion='bolsa.inscripcion.propias.listar' THEN
  IF v_filtro IS DISTINCT FROM p_selector THEN
   RAISE EXCEPTION 'B96: lista propia no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_propias_';
 ELSIF p_accion='bolsa.inscripcion.rrhh.listar' THEN
  IF v_filtro IS DISTINCT FROM p_selector THEN
   RAISE EXCEPTION 'B96: lista RRHH no ligada' USING ERRCODE='42501'; END IF;
  v_ref_selector:='';v_prefijo:='inscripciones_rrhh_';
 ELSIF p_accion='bolsa.inscripcion.rrhh.motivos' THEN
  v_ref_selector:=p_selector->>'decision';v_prefijo:='motivos_inscripcion_';
 ELSE RAISE EXCEPTION 'B96: acción lectora desconocida' USING ERRCODE='42501'; END IF;
 IF v_prefijo IS NOT NULL THEN
  v_recurso_canonico:='{"accion":'||to_json(p_accion)::text||
   ',"persona_ref":'||to_json(v_persona)::text||
   ',"idioma":'||to_json(v_idioma)::text||
   ',"estado":'||to_json(v_f_estado)::text||
   ',"convocatoria_ref":'||to_json(v_f_convocatoria)::text||
   ',"limite":'||v_f_limite::text||
   ',"cursor":'||to_json(v_f_cursor)::text||
   ',"ref":'||to_json(v_ref_selector)::text||'}';
  v_recurso_esperado:=v_prefijo||encode(sha256(convert_to(v_recurso_canonico,'UTF8')),'hex');
 END IF;
 IF v_recurso IS DISTINCT FROM v_recurso_esperado THEN
  RAISE EXCEPTION 'B96: recurso de lectura no ligado al selector' USING ERRCODE='42501';
 END IF;

 IF p_accion='bolsa.inscripcion.convocatorias.listar' THEN
  IF v_canal NOT IN('externa_personal','interna_corporativa')
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['cursor','limite']
   OR jsonb_typeof(p_selector->'limite') IS DISTINCT FROM 'number'
   OR coalesce(p_selector->>'limite','') !~ '^[1-9][0-9]{0,2}$'
   OR (p_selector->>'limite')::integer NOT BETWEEN 1 AND 100
   OR jsonb_typeof(p_selector->'cursor') IS DISTINCT FROM 'string'
  THEN RAISE EXCEPTION 'B96: lista abierta inválida' USING ERRCODE='B9605'; END IF;
  v_proyeccion:=vec_bolsa_convocatorias.listar_abiertas_inscripcion_v1(
   NULLIF(p_selector->>'cursor',''),(p_selector->>'limite')::integer);
  v_proyeccion:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   v_proyeccion,v_persona,v_idioma,v_contexto->>'metodo',v_contexto->>'garantia',true);
 ELSIF p_accion='bolsa.inscripcion.convocatoria.consultar' THEN
  IF v_canal NOT IN('externa_personal','interna_corporativa')
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['convocatoria_ref']
   OR v_recurso IS DISTINCT FROM p_selector->>'convocatoria_ref'
  THEN RAISE EXCEPTION 'B96: detalle abierto inválido' USING ERRCODE='B9605'; END IF;
  v_proyeccion:=vec_bolsa_convocatorias.detalle_abierta_inscripcion_v1(
   p_selector->>'convocatoria_ref');
  IF v_proyeccion IS NULL THEN v_resultado:='no_encontrada';
  ELSE v_proyeccion:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(
   v_proyeccion,v_persona,v_idioma,v_contexto->>'metodo',v_contexto->>'garantia',false);
  END IF;
 ELSIF p_accion IN('bolsa.inscripcion.propias.listar','bolsa.inscripcion.rrhh.listar') THEN
  IF (p_accion='bolsa.inscripcion.propias.listar'
      AND v_canal NOT IN('externa_personal','interna_corporativa'))
   OR (p_accion='bolsa.inscripcion.rrhh.listar' AND v_canal<>'interna_corporativa')
   OR p_captura->'filtro' IS DISTINCT FROM p_selector
  THEN RAISE EXCEPTION 'B96: filtro no ligado' USING ERRCODE='42501'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(
   p_accion='bolsa.inscripcion.propias.listar',v_persona,p_selector,v_idioma);
 ELSIF p_accion IN('bolsa.inscripcion.propia.consultar','bolsa.inscripcion.rrhh.consultar') THEN
  IF (p_accion='bolsa.inscripcion.propia.consultar'
      AND v_canal NOT IN('externa_personal','interna_corporativa'))
   OR (p_accion='bolsa.inscripcion.rrhh.consultar' AND v_canal<>'interna_corporativa')
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['solicitud_ref']
   OR v_recurso IS DISTINCT FROM p_selector->>'solicitud_ref'
  THEN RAISE EXCEPTION 'B96: detalle no ligado' USING ERRCODE='42501'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(
   p_accion='bolsa.inscripcion.propia.consultar',v_persona,
   p_selector->>'solicitud_ref',v_idioma);
  IF v_proyeccion IS NULL THEN v_resultado:='no_encontrada'; END IF;
 ELSIF p_accion='bolsa.inscripcion.rrhh.motivos' THEN
  IF v_canal<>'interna_corporativa'
   OR ARRAY(SELECT jsonb_object_keys(p_selector) ORDER BY 1) IS DISTINCT FROM ARRAY['decision']
   OR jsonb_typeof(p_selector->'decision') IS DISTINCT FROM 'string'
   OR (p_selector->>'decision' IN('admitir','rechazar')) IS NOT TRUE
  THEN RAISE EXCEPTION 'B96: motivos no ligados' USING ERRCODE='42501'; END IF;
  v_motivos:=vec_catalogos_configurables.listar_motivos_inscripcion_v1(v_idioma);
  SELECT coalesce(jsonb_agg(jsonb_build_object('codigo',x.valor->>'motivo_ref',
    'etiqueta',x.valor->>'motivo_etiqueta','obligatorio',true) ORDER BY x.valor->>'motivo_ref'),'[]'::jsonb)
  INTO v_motivos_proyeccion FROM jsonb_array_elements(v_motivos->'motivos') AS x(valor)
  WHERE p_selector->>'decision'='rechazar';
  v_proyeccion:=jsonb_build_object('catalogo_version',(v_motivos->>'catalogo_version')::bigint,
   'motivos',v_motivos_proyeccion);
 ELSE RAISE EXCEPTION 'B96: acción lectora desconocida' USING ERRCODE='42501'; END IF;

 v_orden:=jsonb_build_object(
  'intento_ref',p_captura->>'intento_ref',
  'registro_contexto_ref',v_vinculo->>'registro_contexto_ref',
  'contexto_sha256',v_vinculo->>'contexto_actor_huella_sha256',
  'procedencia_sha256',v_vinculo->>'manifiesto_procedencia_huella_sha256',
  'autenticacion_ref',v_vinculo->>'autenticacion_ref',
  'sesion_ref',v_vinculo->>'sesion_ref',
  'autenticacion_sha256',v_vinculo->>'autenticacion_huella_sha256',
  'accion',p_accion,'modulo_id','bolsa','recurso_ref',v_recurso,
  'finalidad_ref',p_captura->>'finalidad',
  'resultado',v_resultado,
  'motivo_ref',CASE WHEN v_resultado='obtenida' THEN 'inscripcion_lectura_correcta'
                   ELSE 'inscripcion_no_encontrada' END,
  'proceso','vec-server','canal',v_canal,
  'correlacion_ref',p_captura->>'correlacion_ref');
 SELECT * INTO STRICT v_auditoria
  FROM vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(
   p_contexto_canonico,p_vinculo_canonico,v_orden);
 IF v_auditoria.auditoria_ref IS NULL
 OR v_auditoria.correlacion_ref IS DISTINCT FROM p_captura->>'correlacion_ref'
 THEN RAISE EXCEPTION 'B96: asiento de lectura incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('resultado',v_resultado,'proyeccion',v_proyeccion,
  'auditoria_ref',v_auditoria.auditoria_ref,
  'correlacion_ref',v_auditoria.correlacion_ref,'consultada_en',v_auditoria.registrada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO
 vec_bolsa_llamamientos_lector_inscripciones,
 vec_bolsa_llamamientos_lector_inscripciones_empleado,
 vec_bolsa_llamamientos_lector_inscripciones_rrhh;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_inscripcion_v1(text,jsonb,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_lector_inscripciones,
    vec_bolsa_llamamientos_lector_inscripciones_empleado,
    vec_bolsa_llamamientos_lector_inscripciones_rrhh;

COMMIT;
