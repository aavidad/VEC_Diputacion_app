\set ON_ERROR_STOP on
-- B97: incorporación administrativa de una solicitud admitida a una
-- participación y orden B7/B79 PREEXISTENTES. El acta es una selección RRHH
-- expresa; no se deduce de categoría, fecha ni material enviado por navegador.
-- La asociación convocatoria publicada→acta y el recibo v3 nacen en el mismo
-- TopXID SERIALIZABLE que el consumo nominal V3. Sin DOWN sobre historia.
-- incorporar_inscripcion_v1 recibe 13 argumentos, como presentar y revisar:
-- el tercero es el recurso canónico que el Go liga a la decisión V3.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000097',0));

DO $pre$
DECLARE f oid;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_version') IS NULL
 OR (SELECT count(*) FROM pg_attribute a
   WHERE a.attrelid=to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_version')
    AND a.attname IN('perfil_ref','cuenta_ref') AND a.attnotnull AND NOT a.attisdropped)<>2
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_historia') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_outbox') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_recibo') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.solicitud_inscripcion_acceso') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.recibo_carga_convoca') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.sustitucion_bolsa') IS NULL
 OR to_regprocedure('vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.inscripcion_acta_asociada') IS NOT NULL
 OR to_regclass('vec_bolsa_llamamientos.inscripcion_incorporacion') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.incorporar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B97: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(text,text,text)'::regprocedure::oid,
  'vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(text,integer,text)'::regprocedure::oid,
  'vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(text)'::regprocedure::oid,
  'vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)'::regprocedure::oid,
  'vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure::oid
 ] LOOP
  IF NOT has_function_privilege(current_user,f,'EXECUTE') THEN
   RAISE EXCEPTION 'B97: dependencia sin EXECUTE' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;

-- B96 fija las tres familias al crear sus tablas. Se amplían únicamente esas
-- restricciones; filas anteriores conservan estado, recibo y significado.
DO $checks$
DECLARE h text; o text; a text;
BEGIN
 SELECT pg_get_constraintdef(c.oid) INTO STRICT h FROM pg_constraint c
 WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion_historia'::regclass
 AND c.conname='solicitud_inscripcion_historia_accion_check' AND c.contype='c' AND c.convalidated;
 SELECT pg_get_constraintdef(c.oid) INTO STRICT o FROM pg_constraint c
 WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion_outbox'::regclass
 AND c.conname='solicitud_inscripcion_outbox_esquema_check' AND c.contype='c' AND c.convalidated;
 SELECT pg_get_constraintdef(c.oid) INTO STRICT a FROM pg_constraint c
 WHERE c.conrelid='vec_bolsa_llamamientos.solicitud_inscripcion_acceso'::regclass
 AND c.conname='solicitud_inscripcion_acceso_accion_check' AND c.contype='c' AND c.convalidated;
 IF h IS DISTINCT FROM $literal$CHECK ((accion = ANY (ARRAY['presentar'::text, 'revisar'::text])))$literal$
 OR o IS DISTINCT FROM $literal$CHECK ((esquema = ANY (ARRAY['vec.bolsa.inscripcion.presentada.v1'::text, 'vec.bolsa.inscripcion.revisada.v1'::text])))$literal$
 OR a IS DISTINCT FROM $literal$CHECK ((accion = ANY (ARRAY['presentar'::text, 'revisar'::text])))$literal$
 THEN RAISE EXCEPTION 'B97: restricciones B96 divergentes' USING ERRCODE='55000'; END IF;
END $checks$;
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_historia
 DROP CONSTRAINT solicitud_inscripcion_historia_accion_check,
 ADD CONSTRAINT solicitud_inscripcion_historia_accion_check CHECK(accion IN('presentar','revisar','incorporar'));
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_outbox
 DROP CONSTRAINT solicitud_inscripcion_outbox_esquema_check,
 ADD CONSTRAINT solicitud_inscripcion_outbox_esquema_check CHECK(esquema IN(
  'vec.bolsa.inscripcion.presentada.v1','vec.bolsa.inscripcion.revisada.v1',
  'vec.bolsa.inscripcion.incorporada.v1'));
ALTER TABLE vec_bolsa_llamamientos.solicitud_inscripcion_acceso
 DROP CONSTRAINT solicitud_inscripcion_acceso_accion_check,
 ADD CONSTRAINT solicitud_inscripcion_acceso_accion_check CHECK(accion IN('presentar','revisar','incorporar'));

CREATE TABLE vec_bolsa_llamamientos.inscripcion_acta_asociada (
 convocatoria_ref text COLLATE "C" NOT NULL,
 convocatoria_id text COLLATE "C" NOT NULL,
 secuencia bigint NOT NULL CHECK(secuencia>0),
 version_sha256 text NOT NULL CHECK(version_sha256~'^[0-9a-f]{64}$'),
 categoria_ref text COLLATE "C" NOT NULL,
 acta_ref text COLLATE "C" NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
 bolsa_ref text COLLATE "C" NOT NULL,
 version_bolsa bigint NOT NULL,
 huella_bolsa_sha256 text NOT NULL,
 instantanea_ref text COLLATE "C" NOT NULL,
 version_instantanea bigint NOT NULL,
 huella_instantanea_sha256 text NOT NULL,
 solicitud_origen_ref text COLLATE "C" NOT NULL REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 politica_ref text COLLATE "C" NOT NULL CHECK(politica_ref='asociacion.manual.acta'),
 politica_catalogo_ref text COLLATE "C" NOT NULL,
 politica_catalogo_version integer NOT NULL CHECK(politica_catalogo_version>0),
 politica_catalogo_sha256 text NOT NULL CHECK(politica_catalogo_sha256~'^[0-9a-f]{64}$'),
 motivo_ref text COLLATE "C" NOT NULL,
 alcance_ref text COLLATE "C" NOT NULL,
 actor_ref text COLLATE "C" NOT NULL,
 decision_ref text COLLATE "C" NOT NULL UNIQUE,
 auditoria_ref text COLLATE "C" NOT NULL UNIQUE,
 asociada_en timestamptz(6) NOT NULL CHECK(isfinite(asociada_en)),
 PRIMARY KEY(convocatoria_ref,categoria_ref),
 UNIQUE(convocatoria_ref,categoria_ref,acta_ref,bolsa_ref,instantanea_ref,version_instantanea),
 FOREIGN KEY(bolsa_ref,version_bolsa,huella_bolsa_sha256)
  REFERENCES vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref,version,huella_bolsa_sha256),
 FOREIGN KEY(instantanea_ref,version_instantanea,huella_instantanea_sha256)
  REFERENCES vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256),
 CHECK(alcance_ref=convocatoria_ref)
);
CREATE TABLE vec_bolsa_llamamientos.inscripcion_incorporacion (
 solicitud_ref text COLLATE "C" PRIMARY KEY REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion(solicitud_ref),
 version bigint NOT NULL CHECK(version=3),
 convocatoria_ref text COLLATE "C" NOT NULL,
 categoria_ref text COLLATE "C" NOT NULL,
 acta_ref text COLLATE "C" NOT NULL REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
 bolsa_ref text COLLATE "C" NOT NULL,
 instantanea_ref text COLLATE "C" NOT NULL,
 version_instantanea bigint NOT NULL,
 participacion_ref text COLLATE "C" NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.vinculo_candidato(participacion_ref),
 candidato_ref text COLLATE "C" NOT NULL CHECK(candidato_ref~'^can_[A-Za-z0-9_-]{22,128}$'),
 orden_acta bigint NOT NULL CHECK(orden_acta>0),
 persona_ref text COLLATE "C" NOT NULL,
 persona_version bigint NOT NULL CHECK(persona_version>0),
 vinculo_ref text COLLATE "C" NOT NULL,
 vinculo_version bigint NOT NULL CHECK(vinculo_version>0),
 procedencia_ref text COLLATE "C" NOT NULL,
 procedencia_version bigint NOT NULL CHECK(procedencia_version>0),
 procedencia_sha256 text NOT NULL CHECK(procedencia_sha256~'^[0-9a-f]{64}$'),
 decision_ref text COLLATE "C" NOT NULL UNIQUE,
 auditoria_ref text COLLATE "C" NOT NULL UNIQUE,
 recibo_ref text COLLATE "C" NOT NULL UNIQUE,
 incorporada_en timestamptz(6) NOT NULL CHECK(isfinite(incorporada_en)),
 FOREIGN KEY(instantanea_ref,version_instantanea,participacion_ref)
  REFERENCES vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,participacion_ref),
 FOREIGN KEY(convocatoria_ref,categoria_ref,acta_ref,bolsa_ref,instantanea_ref,version_instantanea)
  REFERENCES vec_bolsa_llamamientos.inscripcion_acta_asociada(
   convocatoria_ref,categoria_ref,acta_ref,bolsa_ref,instantanea_ref,version_instantanea),
 FOREIGN KEY(recibo_ref) REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_recibo(recibo_ref),
 FOREIGN KEY(solicitud_ref,version) REFERENCES vec_bolsa_llamamientos.solicitud_inscripcion_version(solicitud_ref,version)
);

-- Cierre literal de cada tabla nueva: sin permisos ajenos, RLS forzada sólo
-- para el propietario e historia inmutable (UPDATE, DELETE y TRUNCATE rechazados).
REVOKE ALL ON TABLE vec_bolsa_llamamientos.inscripcion_acta_asociada FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.inscripcion_acta_asociada FROM PUBLIC;
ALTER TABLE vec_bolsa_llamamientos.inscripcion_acta_asociada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.inscripcion_acta_asociada FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_inscripcion ON vec_bolsa_llamamientos.inscripcion_acta_asociada FOR ALL TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.inscripcion_acta_asociada
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.inscripcion_acta_asociada
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
REVOKE ALL ON TABLE vec_bolsa_llamamientos.inscripcion_incorporacion FROM PUBLIC,vec_bolsa_llamamientos_ejecutor;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.inscripcion_incorporacion FROM PUBLIC;
ALTER TABLE vec_bolsa_llamamientos.inscripcion_incorporacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.inscripcion_incorporacion FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_inscripcion ON vec_bolsa_llamamientos.inscripcion_incorporacion FOR ALL TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.inscripcion_incorporacion
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.inscripcion_incorporacion
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.incorporar_inscripcion_v1(
 p_material text,p_captura_actor jsonb,p_recurso_canonico bytea,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
-- B9701 vínculo de identidad pendiente; B9702 acta/participación no apta;
-- B9703 conflicto de estado/publicación/política; B9603 clave reutilizada.
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='20s'
AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; contexto jsonb; recurso jsonb; v_conjunto jsonb; consumo record;
 s vec_bolsa_llamamientos.solicitud_inscripcion%ROWTYPE;
 actual vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 aprobada vec_bolsa_llamamientos.solicitud_inscripcion_version%ROWTYPE;
 asociacion vec_bolsa_llamamientos.inscripcion_acta_asociada%ROWTYPE;
 vinculo vec_bolsa_llamamientos.inscripcion_incorporacion%ROWTYPE;
 previo vec_bolsa_llamamientos.solicitud_inscripcion_recibo%ROWTYPE;
 acta vec_bolsa_llamamientos.constitucion%ROWTYPE;
 carga vec_bolsa_llamamientos.recibo_carga_convoca%ROWTYPE;
 bolsa vec_bolsa_llamamientos.bolsa_constituida%ROWTYPE;
 entrada record;
 publicacion jsonb; politica jsonb; ca jsonb; inversa jsonb; etiquetas jsonb;
 v_ref text; v_acta text; v_actor text; v_material_sha text; v_clave_sha text;
 v_recurso_sha text; v_candidato text; v_categoria text;
 v_historia text; v_evento text; v_recibo_ref text; v_instante timestamptz(6);
 v_esperada bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_bolsa_llamamientos_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 OR p_recurso_canonico IS NULL OR octet_length(p_recurso_canonico) NOT BETWEEN 2 AND 4096
 OR jsonb_typeof(p_captura_actor) IS DISTINCT FROM 'object'
 OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
 THEN RAISE EXCEPTION 'B97: incorporación no disponible' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  contexto:=convert_from(p_contexto,'UTF8')::jsonb;
  recurso:=convert_from(p_recurso_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B97: material inválido' USING ERRCODE='B9605';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
  'clave_idempotencia','esquema','evidencia_ref','solicitud_ref','version_esperada']
 OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.inscripcion.incorporar.v1'
 OR coalesce(m->>'solicitud_ref','') !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR coalesce(m->>'evidencia_ref','') !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 OR coalesce(m->>'clave_idempotencia','') !~ '^[A-Za-z0-9_-][A-Za-z0-9._:-]{15,127}$'
 OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
 OR m->>'version_esperada' IS DISTINCT FROM '2'
 THEN RAISE EXCEPTION 'B97: incorporación inválida' USING ERRCODE='B9605'; END IF;
 v_ref:=m->>'solicitud_ref';
 v_acta:=m->>'evidencia_ref';
 v_esperada:=2;
 v_actor:=d->>'principal_id';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_clave_sha:=encode(sha256(convert_to(m->>'clave_idempotencia','UTF8')),'hex');
 -- Mismo recurso que application/inscripcion.RecursoIncorporacion: ámbito de
 -- gestión RRHH y atributos en cadena. Se coteja con la fuente vigente tras
 -- consumir V3, igual que la revisión B96.
 v_recurso_sha:=encode(sha256(p_recurso_canonico),'hex');
 IF jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso) ORDER BY 1) IS DISTINCT FROM ARRAY['ambitos','atributos']
 OR jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'ambitos') ORDER BY 1) IS DISTINCT FROM
   ARRAY['ambito_ref','unidad_ref']
 OR coalesce(recurso#>>'{ambitos,unidad_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR coalesce(recurso#>>'{ambitos,ambito_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
 OR jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(recurso->'atributos') ORDER BY 1) IS DISTINCT FROM ARRAY[
  'conjunto_fuente_ref','conjunto_fuente_sha256','conjunto_fuente_version','conjunto_ref',
  'evidencia_ref','material_sha256','solicitud_fuente_ref','solicitud_fuente_sha256',
  'solicitud_fuente_version','solicitud_ref','version_esperada']
 OR EXISTS(SELECT 1 FROM jsonb_each(recurso->'atributos') AS a(clave,valor)
   WHERE jsonb_typeof(a.valor) IS DISTINCT FROM 'string')
 OR recurso#>>'{atributos,solicitud_ref}' IS DISTINCT FROM v_ref
 OR recurso#>>'{atributos,evidencia_ref}' IS DISTINCT FROM v_acta
 OR recurso#>>'{atributos,version_esperada}' IS DISTINCT FROM m->>'version_esperada'
 OR recurso#>>'{atributos,material_sha256}' IS DISTINCT FROM v_material_sha
 OR coalesce(recurso#>>'{atributos,conjunto_ref}','')=''
 OR coalesce(recurso#>>'{atributos,conjunto_fuente_ref}','')=''
 OR recurso#>>'{atributos,conjunto_fuente_version}' !~ '^[1-9][0-9]{0,9}$'
 OR recurso#>>'{atributos,conjunto_fuente_sha256}' !~ '^[0-9a-f]{64}$'
 OR coalesce(recurso#>>'{atributos,solicitud_fuente_ref}','')=''
 OR recurso#>>'{atributos,solicitud_fuente_version}' !~ '^[1-9][0-9]{0,9}$'
 OR recurso#>>'{atributos,solicitud_fuente_sha256}' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B97: recurso de incorporación incompatible' USING ERRCODE='42501'; END IF;
 IF coalesce(v_actor,'') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'persona_ref' IS DISTINCT FROM v_actor
 OR coalesce(p_captura_actor->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(p_captura_actor->>'cuenta_ref','') !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR p_captura_actor->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR p_captura_actor->>'cuenta_ref' IS DISTINCT FROM contexto->>'cuenta_ref'
 OR p_captura_actor->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_captura_actor->>'idioma' IS NULL
 OR p_captura_actor->>'idioma' NOT IN('es','en')
 OR contexto->>'principal_ref' IS DISTINCT FROM v_actor
 OR contexto->>'perfil_activo_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR d->>'concedida' IS DISTINCT FROM 'true'
 OR d->>'accion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'solicitud_inscripcion'
 OR d->>'finalidad' IS DISTINCT FROM 'incorporar_inscripcion'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.incorporar.v1'
 OR d->>'recurso_ref' IS DISTINCT FROM v_ref
 OR c->>'efecto_ref' IS DISTINCT FROM v_ref
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B97: decisión no exacta' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:inscripcion:'||v_ref,0));
 -- Cada reintento consume una decisión vigente distinta. Si cualquier fuente
 -- falla después, toda la transacción y su consumo vuelven atrás.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_incorporacion_inscripcion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM v_ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'B97: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO s FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_ref FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'B97: solicitud ausente' USING ERRCODE='B9703'; END IF;
 IF v_actor IS NOT DISTINCT FROM s.persona_ref THEN
  RAISE EXCEPTION 'B97: el solicitante no puede incorporar' USING ERRCODE='42501';
 END IF;
 v_conjunto:=vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
 IF recurso#>>'{ambitos,unidad_ref}' IS DISTINCT FROM s.unidad_ref
 OR recurso#>>'{ambitos,ambito_ref}' IS DISTINCT FROM s.ambito_ref
 OR v_conjunto->>'unidad_ref' IS DISTINCT FROM s.unidad_ref
 OR v_conjunto->>'ambito_ref' IS DISTINCT FROM s.ambito_ref
 OR recurso#>>'{atributos,conjunto_ref}' IS DISTINCT FROM v_conjunto->>'conjunto_ref'
 OR recurso#>>'{atributos,conjunto_fuente_ref}' IS DISTINCT FROM v_conjunto->>'fuente_ref'
 OR recurso#>>'{atributos,conjunto_fuente_version}' IS DISTINCT FROM v_conjunto->>'fuente_version'
 OR recurso#>>'{atributos,conjunto_fuente_sha256}' IS DISTINCT FROM v_conjunto->>'fuente_sha256'
 OR recurso#>>'{atributos,solicitud_fuente_ref}' IS DISTINCT FROM s.ambito_fuente_ref
 OR recurso#>>'{atributos,solicitud_fuente_version}' IS DISTINCT FROM s.ambito_fuente_version::text
 OR recurso#>>'{atributos,solicitud_fuente_sha256}' IS DISTINCT FROM s.ambito_fuente_sha256
 THEN RAISE EXCEPTION 'B97: gestión RRHH no vigente para solicitud' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:inscripcion:convocatoria:'||s.convocatoria_ref||':'||s.categoria_ref,0));
 SELECT * INTO STRICT actual FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
  WHERE solicitud_ref=v_ref ORDER BY version DESC LIMIT 1 FOR SHARE;
 SELECT * INTO aprobada FROM vec_bolsa_llamamientos.solicitud_inscripcion_version
  WHERE solicitud_ref=v_ref AND version=2 FOR SHARE;
 IF NOT FOUND OR aprobada.estado IS DISTINCT FROM 'admitida_a_convocatoria'
 OR aprobada.actor_ref IS NOT DISTINCT FROM s.persona_ref
 OR actual.version NOT IN (2,3)
 OR actual.estado NOT IN ('admitida_a_convocatoria','incorporada')
 THEN RAISE EXCEPTION 'B97: solicitud no admitida' USING ERRCODE='B9703'; END IF;

 -- La versión publicada original puede tener cerrado ya el plazo. BC9
 -- comprueba bytes, huella y categoría históricas sin reabrir inscripciones.
 BEGIN
  publicacion:=vec_bolsa_convocatorias.comprobar_version_publicada_inscripcion_v1(
   s.convocatoria_ref,s.categoria_ref,s.version_sha256);
 EXCEPTION WHEN SQLSTATE 'B9601' OR SQLSTATE 'B9605' THEN
  RAISE EXCEPTION 'B97: publicación histórica no acreditada' USING ERRCODE='B9703';
 END;
 IF publicacion->>'convocatoria_ref' IS DISTINCT FROM s.convocatoria_ref
 OR publicacion->>'convocatoria_id' IS DISTINCT FROM s.convocatoria_id
 OR publicacion->>'secuencia' IS DISTINCT FROM s.secuencia::text
 OR publicacion->>'version_sha256' IS DISTINCT FROM s.version_sha256
 OR publicacion->>'categoria_ref' IS DISTINCT FROM s.categoria_ref
 OR publicacion->>'identificador_publico' IS DISTINCT FROM s.identificador_publico
 OR publicacion->>'bases_ref' IS DISTINCT FROM s.bases_ref
 OR publicacion->>'catalogo_ref' IS DISTINCT FROM s.catalogo_ref
 OR publicacion->>'catalogo_version' IS DISTINCT FROM s.catalogo_version::text
 OR publicacion->>'catalogo_sha256' IS DISTINCT FROM s.catalogo_sha256
 OR publicacion->>'politica_catalogo_ref' IS DISTINCT FROM s.politica_catalogo_ref
 OR publicacion->>'politica_catalogo_version' IS DISTINCT FROM s.politica_catalogo_version::text
 OR publicacion->>'politica_catalogo_sha256' IS DISTINCT FROM s.politica_catalogo_sha256
 OR publicacion->>'formulario_ref' IS DISTINCT FROM s.formulario_ref
 OR publicacion->>'formulario_version' IS DISTINCT FROM s.formulario_version::text
 OR publicacion->>'formulario_sha256' IS DISTINCT FROM s.formulario_sha256
 OR publicacion->>'requisitos_sha256' IS DISTINCT FROM s.requisitos_sha256
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(publicacion->'plazos_inscripcion') AS p(valor)
  WHERE p.valor->>'plazo_ref'=s.plazo_ref
   AND (p.valor->>'plazo_abre_en')::timestamptz=s.plazo_abre_en
   AND (p.valor->>'plazo_cierra_en')::timestamptz=s.plazo_cierra_en)
 THEN RAISE EXCEPTION 'B97: publicación histórica divergente' USING ERRCODE='B9703'; END IF;
 BEGIN
  politica:=vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256);
 EXCEPTION WHEN SQLSTATE 'B9601' THEN
  RAISE EXCEPTION 'B97: política de asociación no acreditada' USING ERRCODE='B9703';
 END;
 IF politica->>'permitida' IS DISTINCT FROM 'true'
 OR politica->>'politica_ref' IS DISTINCT FROM 'asociacion.manual.acta'
 OR politica->>'version' IS DISTINCT FROM s.politica_catalogo_version::text
 OR politica->>'sha256' IS DISTINCT FROM s.politica_catalogo_sha256
 OR politica->>'alcance_ref' IS DISTINCT FROM s.convocatoria_ref
 OR coalesce(politica->>'motivo_ref','')=''
 THEN RAISE EXCEPTION 'B97: política de asociación ausente' USING ERRCODE='B9703'; END IF;

 -- Acta seleccionada por RRHH: B79 y B7 deben ser exactamente el mismo
 -- agregado; la categoría y la vigencia se cotejan desde Bolsa.
 SELECT * INTO acta FROM vec_bolsa_llamamientos.constitucion
  WHERE acta_ref=v_acta FOR SHARE;
 SELECT * INTO carga FROM vec_bolsa_llamamientos.recibo_carga_convoca
  WHERE acta_ref=v_acta FOR SHARE;
 IF acta.acta_ref IS NULL OR carga.acta_ref IS NULL
 OR acta.categoria_ref IS DISTINCT FROM s.categoria_ref
 OR carga.categoria_ref IS DISTINCT FROM s.categoria_ref
 OR carga.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
 OR carga.recibo_constitucion->>'acta_ref' IS DISTINCT FROM acta.acta_ref
 OR carga.recibo_constitucion->>'bolsa_ref' IS DISTINCT FROM acta.bolsa_ref
 OR carga.recibo_constitucion->>'instantanea_ref' IS DISTINCT FROM acta.instantanea_ref
 OR carga.recibo_constitucion->>'version_bolsa' IS DISTINCT FROM acta.version_bolsa::text
 OR carga.recibo_constitucion->>'version_instantanea' IS DISTINCT FROM acta.version_instantanea::text
 THEN RAISE EXCEPTION 'B97: acta no acreditada' USING ERRCODE='B9702'; END IF;
 SELECT * INTO bolsa FROM vec_bolsa_llamamientos.bolsa_constituida
  WHERE bolsa_ref=acta.bolsa_ref AND version=acta.version_bolsa
   AND huella_bolsa_sha256=acta.huella_bolsa_sha256 FOR SHARE;
 IF bolsa.bolsa_ref IS NULL OR bolsa.categoria_ref IS DISTINCT FROM s.categoria_ref
 OR bolsa.estado IS DISTINCT FROM 'vigente'
 OR consumo.consumida_en<bolsa.vigente_desde
 OR (bolsa.vigente_hasta IS NOT NULL AND consumo.consumida_en>=bolsa.vigente_hasta)
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa reemplazo
  WHERE reemplazo.bolsa_ref_sustituida=acta.bolsa_ref
   AND reemplazo.version_sustituida=acta.version_bolsa
   AND reemplazo.huella_sustituida_sha256=acta.huella_bolsa_sha256
   AND reemplazo.sustituida_en<=consumo.consumida_en)
 THEN RAISE EXCEPTION 'B97: bolsa sin vigencia' USING ERRCODE='B9702'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.instantanea_orden_bolsa i
  WHERE i.instantanea_ref=acta.instantanea_ref AND i.version=acta.version_instantanea
   AND i.huella_instantanea_sha256=acta.huella_instantanea_sha256
   AND i.bolsa_ref=acta.bolsa_ref AND i.version_bolsa=acta.version_bolsa
   AND i.huella_bolsa_sha256=acta.huella_bolsa_sha256)
 THEN RAISE EXCEPTION 'B97: instantánea ajena' USING ERRCODE='B9702'; END IF;

 ca:=vec_contexto_actor_v1.resolver_candidato_persona_inscripcion_v1(s.persona_ref);
 IF ca->>'estado' IS DISTINCT FROM 'acreditado'
 OR ca->>'persona_ref' IS DISTINCT FROM s.persona_ref
 OR coalesce(ca->>'candidato_ref','') !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(ca->>'persona_version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'procedencia_version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(ca->>'procedencia_sha256','') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'B97: vínculo de identidad pendiente' USING ERRCODE='B9701'; END IF;
 v_candidato:=ca->>'candidato_ref';
 inversa:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(v_candidato);
 IF inversa->>'estado' IS DISTINCT FROM 'acreditado'
 OR inversa#>>'{persona,ref}' IS DISTINCT FROM s.persona_ref
 OR inversa#>>'{persona,version}' IS DISTINCT FROM ca->>'persona_version'
 OR inversa#>>'{vinculo,ref}' IS DISTINCT FROM ca->>'vinculo_ref'
 OR inversa#>>'{vinculo,version}' IS DISTINCT FROM ca->>'version'
 OR inversa#>>'{vinculo,procedencia_ref}' IS DISTINCT FROM ca->>'procedencia_ref'
 OR inversa#>>'{vinculo,procedencia_version}' IS DISTINCT FROM ca->>'procedencia_version'
 OR inversa#>>'{vinculo,procedencia_sha256}' IS DISTINCT FROM ca->>'procedencia_sha256'
 OR inversa#>>'{vinculo,poblacion}' IS DISTINCT FROM ca->>'poblacion'
 THEN RAISE EXCEPTION 'B97: identidad y candidato divergentes' USING ERRCODE='B9701'; END IF;
 SELECT vc.participacion_ref,vc.acta_ref,vc.instantanea_ref,vc.version_instantanea,
  e.orden, c2.bolsa_ref
 INTO entrada
 FROM vec_bolsa_llamamientos.vinculo_candidato vc
 JOIN vec_bolsa_llamamientos.constitucion c2 ON c2.acta_ref=vc.acta_ref
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
  ON e.instantanea_ref=vc.instantanea_ref AND e.version_instantanea=vc.version_instantanea
  AND e.participacion_ref=vc.participacion_ref
 WHERE vc.candidato_ref=v_candidato AND vc.acta_ref=v_acta
  AND vc.instantanea_ref=acta.instantanea_ref
  AND vc.version_instantanea=acta.version_instantanea
  AND c2.bolsa_ref=acta.bolsa_ref;
 IF NOT FOUND OR entrada.participacion_ref IS NULL OR entrada.orden<1
 THEN RAISE EXCEPTION 'B97: participación ajena o ausente' USING ERRCODE='B9702'; END IF;

 SELECT * INTO asociacion FROM vec_bolsa_llamamientos.inscripcion_acta_asociada
  WHERE convocatoria_ref=s.convocatoria_ref AND categoria_ref=s.categoria_ref FOR SHARE;
 IF FOUND THEN
  IF asociacion.acta_ref IS DISTINCT FROM v_acta
   OR asociacion.convocatoria_id IS DISTINCT FROM s.convocatoria_id
   OR asociacion.secuencia IS DISTINCT FROM s.secuencia
   OR asociacion.version_sha256 IS DISTINCT FROM s.version_sha256
   OR asociacion.categoria_ref IS DISTINCT FROM s.categoria_ref
   OR asociacion.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
   OR asociacion.instantanea_ref IS DISTINCT FROM acta.instantanea_ref
   OR asociacion.version_instantanea IS DISTINCT FROM acta.version_instantanea
   OR asociacion.politica_catalogo_ref IS DISTINCT FROM s.politica_catalogo_ref
   OR asociacion.politica_catalogo_version IS DISTINCT FROM s.politica_catalogo_version
   OR asociacion.politica_catalogo_sha256 IS DISTINCT FROM s.politica_catalogo_sha256
   OR asociacion.alcance_ref IS DISTINCT FROM s.convocatoria_ref
  THEN RAISE EXCEPTION 'B97: convocatoria asociada a otra acta' USING ERRCODE='B9702'; END IF;
 ELSIF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.inscripcion_acta_asociada x WHERE x.acta_ref=v_acta) THEN
  RAISE EXCEPTION 'B97: acta ya asociada a otra convocatoria' USING ERRCODE='B9702';
 END IF;

 etiquetas:=vec_catalogos_configurables.leer_etiquetas_inscripcion_v1(
  s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
  ARRAY[s.categoria_ref],p_captura_actor->>'idioma');
 v_categoria:=etiquetas#>>'{0,categoria}';
 IF v_categoria IS NULL OR octet_length(v_categoria) NOT BETWEEN 1 AND 200
 THEN RAISE EXCEPTION 'B97: categoría no publicable' USING ERRCODE='B9703'; END IF;

 IF actual.version=3 THEN
  IF actual.actor_ref IS DISTINCT FROM v_actor
   OR actual.perfil_ref IS DISTINCT FROM p_captura_actor->>'perfil_ref'
   OR actual.cuenta_ref IS DISTINCT FROM p_captura_actor->>'cuenta_ref'
  THEN RAISE EXCEPTION 'B97: reintento de otra identidad' USING ERRCODE='42501'; END IF;
  SELECT * INTO previo FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo
   WHERE solicitud_ref=v_ref AND version=3;
  SELECT * INTO vinculo FROM vec_bolsa_llamamientos.inscripcion_incorporacion
   WHERE solicitud_ref=v_ref;
  IF previo.recibo_ref IS NULL OR vinculo.solicitud_ref IS NULL
  THEN RAISE EXCEPTION 'B97: recibo histórico incompleto' USING ERRCODE='55000'; END IF;
  IF previo.clave_sha256 IS DISTINCT FROM v_clave_sha
   OR previo.material_sha256 IS DISTINCT FROM v_material_sha
   OR previo.canal IS DISTINCT FROM 'interna_corporativa'
  THEN RAISE EXCEPTION 'B97: clave reutilizada con otro material' USING ERRCODE='B9603'; END IF;
  IF vinculo.acta_ref IS DISTINCT FROM v_acta
   OR vinculo.bolsa_ref IS DISTINCT FROM acta.bolsa_ref
   OR vinculo.participacion_ref IS DISTINCT FROM entrada.participacion_ref
   OR vinculo.candidato_ref IS DISTINCT FROM v_candidato
   OR vinculo.orden_acta IS DISTINCT FROM entrada.orden
   OR vinculo.recibo_ref IS DISTINCT FROM previo.recibo_ref
  THEN RAISE EXCEPTION 'B97: reintento incompatible' USING ERRCODE='B9702'; END IF;
  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
   acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
  VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
   v_ref,v_actor,'incorporar',consumo.decision_ref,consumo.auditoria_ref,'repetida',consumo.consumida_en);
  RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',previo.recibo_ref,
   'convocatoria_ref',s.convocatoria_ref,'categoria_ref',s.categoria_ref,'categoria',v_categoria,
   'bases_ref',s.bases_ref,'declaracion_ref',s.declaracion_ref,
   'estado','incorporada','version',3,'participacion_ref',vinculo.participacion_ref,
   'bolsa_ref',vinculo.bolsa_ref,'registrada_en',s.presentada_en,
   'decidida_en',vinculo.incorporada_en,'repetida',true,'auditoria_ref',consumo.auditoria_ref);
 END IF;
 IF actual.version IS DISTINCT FROM v_esperada OR actual.estado<>'admitida_a_convocatoria'
 THEN RAISE EXCEPTION 'B97: versión superada' USING ERRCODE='B9703'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.inscripcion_incorporacion i
    WHERE i.participacion_ref=entrada.participacion_ref)
 THEN RAISE EXCEPTION 'B97: participación ya incorporada' USING ERRCODE='B9702'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
    WHERE r.solicitud_ref=v_ref AND r.canal='interna_corporativa' AND r.clave_sha256=v_clave_sha)
 THEN RAISE EXCEPTION 'B97: clave ya utilizada' USING ERRCODE='B9603'; END IF;

 v_instante:=consumo.consumida_en;
 v_historia:='historia_inscripcion_'||encode(sha256(convert_to(v_ref||':3','UTF8')),'hex');
 v_evento:='evento_inscripcion_'||encode(sha256(convert_to(v_historia,'UTF8')),'hex');
 v_recibo_ref:='recibo_inscripcion_'||encode(sha256(convert_to(
  v_ref||chr(31)||'3'||chr(31)||v_material_sha,'UTF8')),'hex');
 -- Incorporaciones simultáneas: la segunda choca con una clave única tras la
 -- foto SERIALIZABLE; 40001 para reintentar y responder por la rama de repetición.
 BEGIN
 IF asociacion.convocatoria_ref IS NULL THEN
  INSERT INTO vec_bolsa_llamamientos.inscripcion_acta_asociada(
   convocatoria_ref,convocatoria_id,secuencia,version_sha256,categoria_ref,
   acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
   instantanea_ref,version_instantanea,huella_instantanea_sha256,
   solicitud_origen_ref,politica_ref,politica_catalogo_ref,politica_catalogo_version,
   politica_catalogo_sha256,motivo_ref,alcance_ref,actor_ref,decision_ref,auditoria_ref,asociada_en)
  VALUES(s.convocatoria_ref,s.convocatoria_id,s.secuencia,s.version_sha256,s.categoria_ref,
   v_acta,acta.bolsa_ref,acta.version_bolsa,acta.huella_bolsa_sha256,
   acta.instantanea_ref,acta.version_instantanea,acta.huella_instantanea_sha256,
   v_ref,'asociacion.manual.acta',s.politica_catalogo_ref,s.politica_catalogo_version,
   s.politica_catalogo_sha256,politica->>'motivo_ref',politica->>'alcance_ref',
   v_actor,consumo.decision_ref,consumo.auditoria_ref,v_instante);
 END IF;
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version(
  solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,evaluacion,decision_ref,
  consumo_huella_sha256,auditoria_ref,aplicada_en)
 VALUES(v_ref,3,'incorporada',v_actor,p_captura_actor->>'perfil_ref',
  p_captura_actor->>'cuenta_ref',aprobada.evaluacion,consumo.decision_ref,
  consumo.consumo_huella_sha256,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia(
  historia_ref,solicitud_ref,version,accion,actor_ref,estado,registrada_en)
 VALUES(v_historia,v_ref,3,'incorporar',v_actor,'incorporada',v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox(
  evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en)
 VALUES(v_evento,v_historia,v_ref,3,'vec.bolsa.inscripcion.incorporada.v1',
  jsonb_build_object('solicitud_ref',v_ref,'convocatoria_ref',s.convocatoria_ref,
   'acta_ref',v_acta,'bolsa_ref',acta.bolsa_ref,
   'participacion_ref',entrada.participacion_ref,'version',3,'estado','incorporada'),v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo(
  recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,
  historia_ref,evento_ref,auditoria_ref,emitida_en)
 VALUES(v_recibo_ref,v_ref,3,s.persona_ref,'interna_corporativa',v_clave_sha,v_material_sha,
  v_historia,v_evento,consumo.auditoria_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.inscripcion_incorporacion(
  solicitud_ref,version,convocatoria_ref,categoria_ref,acta_ref,bolsa_ref,instantanea_ref,version_instantanea,
  participacion_ref,candidato_ref,orden_acta,persona_ref,persona_version,
  vinculo_ref,vinculo_version,procedencia_ref,procedencia_version,procedencia_sha256,
  decision_ref,auditoria_ref,recibo_ref,incorporada_en)
 VALUES(v_ref,3,s.convocatoria_ref,s.categoria_ref,v_acta,acta.bolsa_ref,acta.instantanea_ref,acta.version_instantanea,
  entrada.participacion_ref,v_candidato,entrada.orden,s.persona_ref,
  (ca->>'persona_version')::bigint,ca->>'vinculo_ref',(ca->>'version')::bigint,
  ca->>'procedencia_ref',(ca->>'procedencia_version')::bigint,ca->>'procedencia_sha256',
  consumo.decision_ref,consumo.auditoria_ref,v_recibo_ref,v_instante);
 INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_acceso(
  acceso_ref,solicitud_ref,persona_ref,accion,decision_ref,auditoria_ref,resultado,accedida_en)
 VALUES('acceso_inscripcion_'||encode(sha256(convert_to(consumo.decision_ref,'UTF8')),'hex'),
  v_ref,v_actor,'incorporar',consumo.decision_ref,consumo.auditoria_ref,'confirmada',v_instante);
 EXCEPTION WHEN unique_violation THEN
  RAISE EXCEPTION 'B97: incorporación concurrente, reintentar' USING ERRCODE='40001';
 END;
 RETURN jsonb_build_object('solicitud_ref',v_ref,'recibo_ref',v_recibo_ref,
  'convocatoria_ref',s.convocatoria_ref,'categoria_ref',s.categoria_ref,'categoria',v_categoria,
  'bases_ref',s.bases_ref,'declaracion_ref',s.declaracion_ref,
  'estado','incorporada','version',3,'participacion_ref',entrada.participacion_ref,
  'bolsa_ref',acta.bolsa_ref,'registrada_en',s.presentada_en,
  'decidida_en',v_instante,'repetida',false,'auditoria_ref',consumo.auditoria_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.incorporar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.incorporar_inscripcion_v1(
 text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;

-- Las consultas B96 seleccionan ya la última versión y su recibo. Este
-- complemento sólo añade las dos referencias nacidas en B97 cuando el estado
-- es incorporada; no altera filtros, permisos ni historia de otros estados.
CREATE FUNCTION vec_bolsa_llamamientos.proyectar_vinculo_inscripcion_v1(
 p_fila jsonb,p_estado text) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
BEGIN
 IF p_estado<>'incorporada' THEN RETURN '{}'::jsonb; END IF;
 IF p_fila->>'participacion_ref' IS NULL OR p_fila->>'bolsa_ref' IS NULL
 THEN RAISE EXCEPTION 'B97: proyección de incorporación incompleta' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('participacion_ref',p_fila->>'participacion_ref',
  'bolsa_ref',p_fila->>'bolsa_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_vinculo_inscripcion_v1(jsonb,text) FROM PUBLIC;

DO $proyeccion_b96_preimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(jsonb,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=proyeccion_b96_preimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM '78104593be4562a8c2211ce2a11f572b2e27fd3b2b155a6cbb697b667bc1b11f'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '2fa52732451d3016159dc4b0958d9f7dc32cf67b3245d7e15a02b9d07824d5bf'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",TimeZone=UTC}'
 OR p.prosecdef IS DISTINCT FROM false OR p.provolatile<>'s'
 THEN RAISE EXCEPTION 'B97: PARO clave=proyeccion_b96_preimagen esperado=% actual=%','78104593be4562a8c2211ce2a11f572b2e27fd3b2b155a6cbb697b667bc1b11f',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $proyeccion_b96_preimagen$;
-- Definición final literal (postimagen ee405f07…), escrita entera en el fichero.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(p_fila jsonb, p_categoria text, p_motivo_etiqueta text, p_pendiente_etiqueta text, p_cumple_etiqueta text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE v_requisitos jsonb;
BEGIN
 IF jsonb_typeof(p_fila) IS DISTINCT FROM 'object'
 OR p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 1 AND 2048
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
  'decision_ref',p_fila->>'decision_ref') ||
  vec_bolsa_llamamientos.proyectar_vinculo_inscripcion_v1(
   p_fila,p_fila->>'estado');
END $function$;
DO $proyeccion_b96_postimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(jsonb,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=proyeccion_b96_postimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM 'ee405f0773ed4a492ac96aea1e1447897fabcc554aed8ad1f536fa97bfd5dec8'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'fb9e74a277d21604235236e0cd79ef7f97bd747febd14c1497994a2e3f4d1501'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",TimeZone=UTC}'
 OR p.prosecdef IS DISTINCT FROM false OR p.provolatile<>'s'
 THEN RAISE EXCEPTION 'B97: PARO clave=proyeccion_b96_postimagen esperado=% actual=%','ee405f0773ed4a492ac96aea1e1447897fabcc554aed8ad1f536fa97bfd5dec8',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $proyeccion_b96_postimagen$;

-- B96 pagina 100 filas: unir el vínculo una vez en SQL evita una consulta
-- adicional por cada proyección. Ambos lectores conservan permisos/filtros.
DO $listar_b96_preimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=listar_b96_preimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM '744780f17405bd926c6874a8ab35706fb18b2eef50d86fba13a28d9986451b18'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'd880814fc7b73c87ac13951d1259ebc3038e2d574c409a39842fc72a9ed9dabd'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s,statement_timeout=15s}'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'B97: PARO clave=listar_b96_preimagen esperado=% actual=%','744780f17405bd926c6874a8ab35706fb18b2eef50d86fba13a28d9986451b18',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $listar_b96_preimagen$;
-- Definición final literal (postimagen a672e1ea…), escrita entera en el fichero.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(p_propia boolean, p_persona_ref text, p_filtro jsonb, p_idioma text, p_canal text, p_unidad_ref text, p_ambito_ref text)
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
 v_estado text; v_convocatoria text; v_cursor text; v_limite integer;
 v_cursor_en timestamptz(6); v_total bigint; v_raw jsonb; v_labels jsonb;
 v_peticiones jsonb; v_solicitudes jsonb; v_mas boolean; v_siguiente text;
 v_meta_convocatoria jsonb; v_convocatoria_titulo text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_idioma NOT IN('es','en') OR p_canal NOT IN('externa_personal','interna_corporativa')
 OR (p_propia AND p_canal<>'externa_personal')
 OR (NOT p_propia AND p_canal<>'interna_corporativa')
 OR (p_propia AND (p_unidad_ref IS NOT NULL OR p_ambito_ref IS NOT NULL))
 OR (NOT p_propia AND (
  coalesce(p_unidad_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR coalesce(p_ambito_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'))
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
 OR (NOT p_propia AND v_convocatoria='')
 OR (v_convocatoria<>'' AND (v_convocatoria !~ '^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$'
   OR octet_length(v_convocatoria)>200))
 OR (v_cursor<>'' AND v_cursor !~ '^solicitud_inscripcion_[0-9a-f]{64}$')
 THEN RAISE EXCEPTION 'B96: filtro de solicitudes inválido' USING ERRCODE='B9605'; END IF;
 IF NOT p_propia THEN
  v_meta_convocatoria:=vec_bolsa_convocatorias.resolver_resumen_convocatorias_inscripcion_lote_v1(
   ARRAY[v_convocatoria],p_idioma);
  IF jsonb_typeof(v_meta_convocatoria) IS DISTINCT FROM 'array'
   OR jsonb_array_length(v_meta_convocatoria)<>1
   OR v_meta_convocatoria#>>'{0,convocatoria_ref}' IS DISTINCT FROM v_convocatoria
   OR coalesce(v_meta_convocatoria#>>'{0,titulo}','')=''
   OR char_length(v_meta_convocatoria#>>'{0,titulo}')>180
  THEN RAISE EXCEPTION 'B96: título de convocatoria no disponible' USING ERRCODE='B9601'; END IF;
  v_convocatoria_titulo:=v_meta_convocatoria#>>'{0,titulo}';
 END IF;
 IF v_cursor<>'' THEN
  SELECT presentada_en INTO v_cursor_en
  FROM vec_bolsa_llamamientos.solicitud_inscripcion
  WHERE solicitud_ref=v_cursor AND (NOT p_propia OR persona_ref=p_persona_ref)
   AND (v_convocatoria='' OR convocatoria_ref=v_convocatoria)
   AND (p_propia OR (unidad_ref=p_unidad_ref AND ambito_ref=p_ambito_ref));
  IF NOT FOUND THEN RAISE EXCEPTION 'B96: cursor no disponible' USING ERRCODE='B9605'; END IF;
 END IF;
 WITH filtradas AS MATERIALIZED (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref,inc.participacion_ref,inc.bolsa_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
    WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
    ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  LEFT JOIN vec_bolsa_llamamientos.inscripcion_incorporacion inc
    ON inc.solicitud_ref=s.solicitud_ref AND inc.version=v.version
  WHERE (NOT p_propia OR s.persona_ref=p_persona_ref)
   AND (p_propia OR (s.unidad_ref=p_unidad_ref AND s.ambito_ref=p_ambito_ref))
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
  v_peticiones,p_idioma,p_canal);
 IF jsonb_array_length(v_labels)<>jsonb_array_length(v_raw) THEN
  RAISE EXCEPTION 'B96: proyección de catálogo incompleta' USING ERRCODE='55000'; END IF;
  SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_solicitud_inscripcion_v1(
   r.valor,l.valor->>'categoria',NULL,l.valor->>'motivo_etiqueta_pendiente',
   l.valor->>'motivo_etiqueta_cumple') ORDER BY r.orden),'[]'::jsonb)
 INTO v_solicitudes
 FROM jsonb_array_elements(v_raw) WITH ORDINALITY AS r(valor,orden)
 JOIN jsonb_array_elements(v_labels) WITH ORDINALITY AS l(valor,orden) USING(orden);
 RETURN jsonb_build_object('solicitudes',v_solicitudes,'total',v_total,
  'cursor_siguiente',v_siguiente)||CASE WHEN p_propia THEN '{}'::jsonb
  ELSE jsonb_build_object('convocatoria_titulo',v_convocatoria_titulo) END;
END $function$;
DO $listar_b96_postimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.listar_solicitudes_inscripcion_interna_v1(boolean,text,jsonb,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=listar_b96_postimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM 'a672e1eada275323cd7e2eb35d14f09eac0234da7d9c1d2dc09f65fd12ce4d93'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '67f6f6292913594678ff925471d89fe7e8207ef473cbb2b7188ba5a033a7ea6b'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s,statement_timeout=15s}'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'B97: PARO clave=listar_b96_postimagen esperado=% actual=%','a672e1eada275323cd7e2eb35d14f09eac0234da7d9c1d2dc09f65fd12ce4d93',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $listar_b96_postimagen$;
DO $leer_b96_preimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=leer_b96_preimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM 'df4d7cbf85178d2005112e8e73b5f1c1e4d7779c39fe06907fcc5ba5d984c9b7'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '1df4d48c1f07115d32e4a71aa3b32fa0d2589f07299c6479ef3846ca6c3eca57'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s,statement_timeout=10s}'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'B97: PARO clave=leer_b96_preimagen esperado=% actual=%','df4d7cbf85178d2005112e8e73b5f1c1e4d7779c39fe06907fcc5ba5d984c9b7',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $leer_b96_preimagen$;
-- Definición final literal (postimagen d02cd059…), escrita entera en el fichero.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(p_propia boolean, p_persona_ref text, p_solicitud_ref text, p_idioma text, p_canal text, p_unidad_ref text, p_ambito_ref text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE fila jsonb; etiquetas jsonb; motivo_etiqueta text; v_estado text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_propia IS NULL OR p_persona_ref IS NULL
 OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_idioma IS NULL OR p_idioma NOT IN('es','en')
 OR p_canal IS NULL OR p_canal NOT IN('externa_personal','interna_corporativa')
 OR (p_propia AND p_canal<>'externa_personal')
 OR (NOT p_propia AND p_canal<>'interna_corporativa')
 OR (p_propia AND (p_unidad_ref IS NOT NULL OR p_ambito_ref IS NOT NULL))
 OR (NOT p_propia AND (
  coalesce(p_unidad_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'
  OR coalesce(p_ambito_ref,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$'))
 THEN RAISE EXCEPTION 'B96: selector de solicitud inválido' USING ERRCODE='B9605'; END IF;
 SELECT to_jsonb(q) INTO fila FROM (
  SELECT s.solicitud_ref,s.persona_ref,s.convocatoria_ref,s.categoria_ref,
   s.catalogo_ref,s.catalogo_version,s.catalogo_sha256,
   s.politica_catalogo_ref,s.politica_catalogo_version,s.politica_catalogo_sha256,
   s.bases_ref,s.declaracion_ref,s.plazo_abre_en,s.plazo_cierra_en,s.presentada_en,
   v.version,v.estado,v.evaluacion,v.motivo_ref,v.motivo_catalogo_ref,
   v.motivo_catalogo_version,v.motivo_catalogo_sha256,v.decision_ref,v.aplicada_en,
   r.recibo_ref,inc.participacion_ref,inc.bolsa_ref
  FROM vec_bolsa_llamamientos.solicitud_inscripcion s
  JOIN LATERAL (SELECT * FROM vec_bolsa_llamamientos.solicitud_inscripcion_version x
   WHERE x.solicitud_ref=s.solicitud_ref ORDER BY x.version DESC LIMIT 1) v ON true
  JOIN vec_bolsa_llamamientos.solicitud_inscripcion_recibo r
   ON r.solicitud_ref=s.solicitud_ref AND r.version=v.version
  LEFT JOIN vec_bolsa_llamamientos.inscripcion_incorporacion inc
   ON inc.solicitud_ref=s.solicitud_ref AND inc.version=v.version
  WHERE s.solicitud_ref=p_solicitud_ref
   AND (NOT p_propia OR s.persona_ref=p_persona_ref)
   AND (p_propia OR (s.unidad_ref=p_unidad_ref AND s.ambito_ref=p_ambito_ref))
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
   'politica_catalogo_sha256',fila->>'politica_catalogo_sha256')),p_idioma,p_canal);
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
END $function$;
DO $leer_b96_postimagen$
DECLARE f oid:=to_regprocedure('vec_bolsa_llamamientos.leer_solicitud_inscripcion_interna_v1(boolean,text,text,text,text,text,text)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'B97: PARO clave=leer_b96_postimagen actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM 'd02cd0598097a0de74386df92a1ab133c1e7530d68af3b8e9d1a9da02add5d4d'
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '1a90c909e3b1d41495ad3ee9c47dc84787fe200576c71fb2fb0fbd90cecf0b73'
 OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario}'
 OR p.proconfig::text IS DISTINCT FROM '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s,statement_timeout=10s}'
 OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'B97: PARO clave=leer_b96_postimagen esperado=% actual=%','d02cd0598097a0de74386df92a1ab133c1e7530d68af3b8e9d1a9da02add5d4d',
  encode(sha256(convert_to(v_def,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $leer_b96_postimagen$;
COMMIT;
