\set ON_ERROR_STOP on
-- CC12. Gobierno de una categoria de ejercicio por organizacion. AD232 consume
-- V3 antes de invocar el efecto, en la misma transaccion. Sin LOGIN ni ruta.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000012',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_catalogos_configurables_propietario'
    OR pg_catalog.to_regrole('vec_catalogos_configurables_rpt_consumidor') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)') IS NULL
    OR pg_catalog.to_regclass('vec_catalogos_configurables.categoria_control') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.rechazar_cambio_inmutable()') IS NULL
    OR pg_catalog.to_regclass('vec_catalogos_configurables.rpt_propuesta') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid='vec_catalogos_configurables.publicacion'::pg_catalog.regclass
        AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname IN
       ('vec_catalogos_configurables_propietario','vec_catalogos_configurables_rpt_consumidor','vec_autorizacion_atestada_v3_propietario')
       AND (r.rolcanlogin OR r.rolsuper OR r.rolbypassrls)) THEN
  RAISE EXCEPTION 'CC12: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

-- Autoridad de perfiles externa: la primera fila y cada revision requieren
-- un wrapper administrativo futuro que compruebe acto, fuente, CAS y V3.
-- Ausencia de fila = denegacion. Esta migracion no provee perfiles de negocio.
CREATE TABLE vec_catalogos_configurables.rpt_perfiles_competencia (
 clave boolean PRIMARY KEY DEFAULT true CHECK(clave),
 preparacion_ref text NOT NULL CHECK(pg_catalog.octet_length(preparacion_ref) BETWEEN 3 AND 160),
 revision_ref text NOT NULL CHECK(pg_catalog.octet_length(revision_ref) BETWEEN 3 AND 160),
 version bigint NOT NULL CHECK(version > 0),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(pg_catalog.octet_length(fuente_ref) BETWEEN 3 AND 320),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 custodia_ref text NOT NULL CHECK(pg_catalog.octet_length(custodia_ref) BETWEEN 3 AND 320),
 acto_ref text NOT NULL CHECK(pg_catalog.octet_length(acto_ref) BETWEEN 3 AND 320),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 activa boolean NOT NULL,
 CHECK(preparacion_ref <> revision_ref),
 CHECK(vigente_desde < vigente_hasta),
 CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   pg_catalog.jsonb_build_object('preparacion_ref',preparacion_ref,'revision_ref',revision_ref,
      'version',version,'fuente_ref',fuente_ref,'fuente_sha256',fuente_sha256,
      'custodia_ref',custodia_ref,'acto_ref',acto_ref,
      'vigente_desde',vigente_desde,'vigente_hasta',vigente_hasta,'activa',activa)::text,'UTF8')),'hex'))
);
CREATE TABLE vec_catalogos_configurables.rpt_perfiles_historia (
 version bigint PRIMARY KEY CHECK(version > 0),
 preimagen_version bigint NOT NULL CHECK(preimagen_version >= 0),
 preimagen_sha256 text,
 huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 preparacion_ref text NOT NULL,
 revision_ref text NOT NULL,
 fuente_ref text NOT NULL,
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 custodia_ref text NOT NULL,
 acto_ref text NOT NULL,
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 activa boolean NOT NULL,
 actor_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 decision_sha256 text NOT NULL UNIQUE CHECK(decision_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK(version=preimagen_version+1),
 CHECK((preimagen_version=0 AND preimagen_sha256 IS NULL)
    OR (preimagen_version>0 AND preimagen_sha256 ~ '^[0-9a-f]{64}$'))
);
CREATE TABLE vec_catalogos_configurables.rpt_perfiles_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.rpt_perfiles_historia(recibo_ref),
 version bigint NOT NULL,
 evento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(evento)='object'),
 evento_sha256 text NOT NULL CHECK(evento_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(evento::text,'UTF8')),'hex')),
 creado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
CREATE TABLE vec_catalogos_configurables.rpt_propuesta (
 propuesta_ref text PRIMARY KEY CHECK(pg_catalog.octet_length(propuesta_ref) BETWEEN 3 AND 160),
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
 modulo_id text NOT NULL CHECK(modulo_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
 categoria_id text NOT NULL CHECK(categoria_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
 organizacion_ref text NOT NULL CHECK(pg_catalog.octet_length(organizacion_ref) BETWEEN 3 AND 320),
 version integer NOT NULL CHECK(version > 0),
 accion text NOT NULL CHECK(accion='publicar'),
 contenido jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(contenido)='object'),
 contenido_sha256 text NOT NULL CHECK(contenido_sha256 ~ '^[0-9a-f]{64}$'),
 documento_canonico text NOT NULL CHECK(pg_catalog.octet_length(documento_canonico) BETWEEN 2 AND 16777216),
 documento_sha256 text NOT NULL CHECK(documento_sha256 ~ '^[0-9a-f]{64}$'),
 preimagenes jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(preimagenes)='object'),
 preimagenes_sha256 text NOT NULL CHECK(preimagenes_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(pg_catalog.octet_length(fuente_ref) BETWEEN 3 AND 320),
 fuente_clase text NOT NULL CHECK(fuente_clase ~ '^[a-z][a-z0-9_]{2,63}$'),
 fuente_procedencia_ref text NOT NULL CHECK(pg_catalog.octet_length(fuente_procedencia_ref) BETWEEN 3 AND 320),
 fuente_custodia_ref text NOT NULL CHECK(pg_catalog.octet_length(fuente_custodia_ref) BETWEEN 3 AND 320),
 fuente_bytes bytea NOT NULL CHECK(pg_catalog.octet_length(fuente_bytes) BETWEEN 1 AND 16777216),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_vigente_desde timestamptz(6) NOT NULL,
 fuente_vigente_hasta timestamptz(6) NOT NULL,
 perfil_config_version bigint NOT NULL CHECK(perfil_config_version > 0),
 perfil_config_sha256 text NOT NULL CHECK(perfil_config_sha256 ~ '^[0-9a-f]{64}$'),
 creado_por text NOT NULL CHECK(pg_catalog.octet_length(creado_por) BETWEEN 3 AND 160),
 creado_perfil text NOT NULL CHECK(pg_catalog.octet_length(creado_perfil) BETWEEN 3 AND 160),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 3),
 estado text NOT NULL CHECK(estado IN ('propuesta','aprobada','confirmada')),
 aprobado_por text,
 aprobado_perfil text,
 publicado_por text,
 propuesta_decision_ref text NOT NULL UNIQUE,
 propuesta_decision_sha256 text NOT NULL CHECK(propuesta_decision_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_decision_ref text UNIQUE,
 aprobacion_decision_sha256 text,
 confirmacion_decision_ref text UNIQUE,
 confirmacion_decision_sha256 text,
 publicacion_recibo_ref text UNIQUE,
 propuesta_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 aprobada_en timestamptz(6),
 confirmada_en timestamptz(6),
 UNIQUE(catalogo_id,version),
 CHECK(fuente_vigente_desde < fuente_vigente_hasta),
 CHECK(contenido_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(contenido::text,'UTF8')),'hex')),
 CHECK(documento_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento_canonico,'UTF8')),'hex')),
 CHECK(preimagenes_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(preimagenes::text,'UTF8')),'hex')),
 CHECK(fuente_sha256=pg_catalog.encode(pg_catalog.sha256(fuente_bytes),'hex')),
 CHECK(aprobacion_decision_ref IS NULL OR aprobacion_decision_ref<>propuesta_decision_ref),
 CHECK(aprobacion_decision_sha256 IS NULL OR aprobacion_decision_sha256<>propuesta_decision_sha256),
 CHECK(confirmacion_decision_ref IS NULL OR confirmacion_decision_ref NOT IN (propuesta_decision_ref,aprobacion_decision_ref)),
 CHECK(confirmacion_decision_sha256 IS NULL OR confirmacion_decision_sha256 NOT IN (propuesta_decision_sha256,aprobacion_decision_sha256)),
 CHECK((estado='propuesta' AND revision=1 AND aprobado_por IS NULL AND publicado_por IS NULL AND aprobada_en IS NULL AND confirmada_en IS NULL)
    OR (estado='aprobada' AND revision=2 AND aprobado_por IS NOT NULL AND aprobado_perfil IS NOT NULL
      AND aprobacion_decision_ref IS NOT NULL AND aprobacion_decision_sha256 IS NOT NULL AND aprobada_en IS NOT NULL
      AND publicado_por IS NULL AND confirmada_en IS NULL)
    OR (estado='confirmada' AND revision=3 AND aprobado_por IS NOT NULL AND publicado_por=aprobado_por
      AND confirmacion_decision_ref IS NOT NULL AND confirmacion_decision_sha256 IS NOT NULL AND confirmada_en IS NOT NULL))
);
CREATE TABLE vec_catalogos_configurables.rpt_gobierno_historia (
 propuesta_ref text NOT NULL REFERENCES vec_catalogos_configurables.rpt_propuesta(propuesta_ref),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 3),
 accion text NOT NULL CHECK(accion IN ('proponer','aprobar','confirmar')),
 estado text NOT NULL CHECK(estado IN ('propuesta','aprobada','confirmada')),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 decision_ref text NOT NULL UNIQUE,
 decision_sha256 text NOT NULL UNIQUE CHECK(decision_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 motivo_ref text NOT NULL CHECK(pg_catalog.octet_length(motivo_ref) BETWEEN 3 AND 320),
 contenido_sha256 text NOT NULL CHECK(contenido_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 revision_categoria bigint NOT NULL CHECK((accion='confirmar' AND revision_categoria>0)
    OR (accion<>'confirmar' AND revision_categoria=0)),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(propuesta_ref,revision),
 UNIQUE(propuesta_ref,accion)
);
CREATE TABLE vec_catalogos_configurables.rpt_gobierno_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.rpt_gobierno_historia(recibo_ref),
 propuesta_ref text NOT NULL,
 accion text NOT NULL,
 evento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(evento)='object'),
 evento_sha256 text NOT NULL CHECK(evento_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(evento::text,'UTF8')),'hex')),
 creado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);

DO $proteccion$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['rpt_perfiles_competencia','rpt_perfiles_historia','rpt_perfiles_outbox','rpt_propuesta','rpt_gobierno_historia','rpt_gobierno_outbox'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('CREATE POLICY %I ON vec_catalogos_configurables.%I TO vec_catalogos_configurables_propietario USING(true) WITH CHECK(true)',tabla||'_owner',tabla);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM PUBLIC',tabla);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM PUBLIC',tabla);
  IF tabla IN ('rpt_perfiles_historia','rpt_perfiles_outbox','rpt_gobierno_historia','rpt_gobierno_outbox') THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',tabla||'_inmutable',tabla);
  ELSE
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',tabla||'_no_borrar',tabla);
  END IF;
 END LOOP;
END $proteccion$;

-- La autoridad administrativa aprobada entrega referencias nominales ya
-- verificadas; esta funcion solo calcula la huella que AD3 vinculara a V3.
CREATE FUNCTION vec_catalogos_configurables.preparar_provision_perfiles_rpt(
 p_preparacion_ref text,p_revision_ref text,p_version bigint,p_fuente_ref text,
 p_fuente_sha256 text,p_custodia_ref text,p_acto_ref text,
 p_vigente_desde timestamptz,p_vigente_hasta timestamptz,p_activa boolean)
RETURNS text LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
BEGIN
 IF p_version IS NULL OR p_version<1
    OR p_preparacion_ref IS NULL OR pg_catalog.octet_length(p_preparacion_ref) NOT BETWEEN 3 AND 160
    OR p_revision_ref IS NULL OR pg_catalog.octet_length(p_revision_ref) NOT BETWEEN 3 AND 160
    OR p_preparacion_ref=p_revision_ref
    OR p_fuente_ref IS NULL OR pg_catalog.octet_length(p_fuente_ref) NOT BETWEEN 3 AND 320
    OR p_fuente_sha256 IS NULL OR p_fuente_sha256 !~ '^[0-9a-f]{64}$'
    OR p_custodia_ref IS NULL OR pg_catalog.octet_length(p_custodia_ref) NOT BETWEEN 3 AND 320
    OR p_acto_ref IS NULL OR pg_catalog.octet_length(p_acto_ref) NOT BETWEEN 3 AND 320
    OR p_vigente_desde IS NULL OR p_vigente_hasta IS NULL OR p_vigente_desde>=p_vigente_hasta
    OR p_activa IS NULL THEN
  RAISE EXCEPTION 'CC12: provision de perfiles invalida' USING ERRCODE='22023'; END IF;
 RETURN pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_object(
   'preparacion_ref',p_preparacion_ref,'revision_ref',p_revision_ref,'version',p_version,
   'fuente_ref',p_fuente_ref,'fuente_sha256',p_fuente_sha256,'custodia_ref',p_custodia_ref,
   'acto_ref',p_acto_ref,'vigente_desde',p_vigente_desde,'vigente_hasta',p_vigente_hasta,
   'activa',p_activa)::text,'UTF8')),'hex');
END $f$;

-- AD3 consume V3/auditoria antes de llamar aqui. Solo su propietario NOLOGIN
-- tiene EXECUTE; el grupo tecnico de preparacion no puede proveer perfiles.
CREATE FUNCTION vec_catalogos_configurables.provisionar_perfiles_rpt(
 p_preparacion_ref text,p_revision_ref text,p_version_esperada bigint,p_huella_esperada text,
 p_fuente_ref text,p_fuente_sha256 text,p_custodia_ref text,p_acto_ref text,
 p_actor_ref text,p_decision_ref text,p_decision_sha256 text,p_recibo_ref text,
 p_vigente_desde timestamptz,p_vigente_hasta timestamptz,p_activa boolean,p_huella_nueva text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE actual vec_catalogos_configurables.rpt_perfiles_competencia%ROWTYPE;
 anterior vec_catalogos_configurables.rpt_perfiles_historia%ROWTYPE; calculada text; evento jsonb;
BEGIN
 IF p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada=9223372036854775807
    OR ((p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
      OR (p_version_esperada>0 AND (p_huella_esperada IS NULL OR p_huella_esperada !~ '^[0-9a-f]{64}$')))
    OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 3 AND 160
    OR p_decision_ref IS NULL OR pg_catalog.octet_length(p_decision_ref) NOT BETWEEN 3 AND 160
    OR p_decision_sha256 IS NULL OR p_decision_sha256 !~ '^[0-9a-f]{64}$'
    OR p_recibo_ref IS NULL OR pg_catalog.octet_length(p_recibo_ref) NOT BETWEEN 3 AND 160
    OR p_huella_nueva IS NULL OR p_huella_nueva !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CC12: provision no autorizable' USING ERRCODE='22023'; END IF;
 calculada:=vec_catalogos_configurables.preparar_provision_perfiles_rpt(
   p_preparacion_ref,p_revision_ref,p_version_esperada+1,p_fuente_ref,p_fuente_sha256,
   p_custodia_ref,p_acto_ref,p_vigente_desde,p_vigente_hasta,p_activa);
 IF calculada<>p_huella_nueva THEN
  RAISE EXCEPTION 'CC12: huella de provision distinta' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:rpt:perfiles',0));
 SELECT * INTO anterior FROM vec_catalogos_configurables.rpt_perfiles_historia WHERE recibo_ref=p_recibo_ref;
 IF FOUND THEN
  IF anterior.preimagen_version<>p_version_esperada OR anterior.preimagen_sha256 IS DISTINCT FROM p_huella_esperada
     OR anterior.huella_sha256<>p_huella_nueva OR anterior.preparacion_ref<>p_preparacion_ref
     OR anterior.revision_ref<>p_revision_ref OR anterior.fuente_ref<>p_fuente_ref
     OR anterior.fuente_sha256<>p_fuente_sha256 OR anterior.custodia_ref<>p_custodia_ref
     OR anterior.acto_ref<>p_acto_ref OR anterior.actor_ref<>p_actor_ref
     OR anterior.vigente_desde<>p_vigente_desde OR anterior.vigente_hasta<>p_vigente_hasta
     OR anterior.activa<>p_activa THEN
   RAISE EXCEPTION 'CC12: recibo de perfiles ocupado' USING ERRCODE='23505'; END IF;
  RETURN pg_catalog.jsonb_build_object('version',anterior.version,'huella_sha256',anterior.huella_sha256,
    'recibo_ref',anterior.recibo_ref,'registrada_en',anterior.registrada_en,
    'decision_ref',anterior.decision_ref,'decision_sha256',anterior.decision_sha256);
 END IF;
 SELECT * INTO actual FROM vec_catalogos_configurables.rpt_perfiles_competencia WHERE clave=true FOR UPDATE;
 IF p_version_esperada=0 THEN
  IF FOUND THEN RAISE EXCEPTION 'CC12: perfiles ya provistos' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_catalogos_configurables.rpt_perfiles_competencia
   (clave,preparacion_ref,revision_ref,version,huella_sha256,fuente_ref,fuente_sha256,
    custodia_ref,acto_ref,vigente_desde,vigente_hasta,activa)
  VALUES(true,p_preparacion_ref,p_revision_ref,1,p_huella_nueva,p_fuente_ref,p_fuente_sha256,
    p_custodia_ref,p_acto_ref,p_vigente_desde,p_vigente_hasta,p_activa);
 ELSE
  IF NOT FOUND OR actual.version<>p_version_esperada OR actual.huella_sha256<>p_huella_esperada
     OR NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.rpt_perfiles_historia
         WHERE version=actual.version AND huella_sha256=actual.huella_sha256) THEN
   RAISE EXCEPTION 'CC12: CAS de perfiles obsoleto' USING ERRCODE='40001'; END IF;
  UPDATE vec_catalogos_configurables.rpt_perfiles_competencia
   SET preparacion_ref=p_preparacion_ref,revision_ref=p_revision_ref,version=p_version_esperada+1,
       huella_sha256=p_huella_nueva,fuente_ref=p_fuente_ref,fuente_sha256=p_fuente_sha256,
       custodia_ref=p_custodia_ref,acto_ref=p_acto_ref,vigente_desde=p_vigente_desde,
       vigente_hasta=p_vigente_hasta,activa=p_activa
   WHERE clave=true AND version=p_version_esperada AND huella_sha256=p_huella_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'CC12: CAS de perfiles fallido' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_catalogos_configurables.rpt_perfiles_historia
  (version,preimagen_version,preimagen_sha256,huella_sha256,preparacion_ref,revision_ref,
   fuente_ref,fuente_sha256,custodia_ref,acto_ref,vigente_desde,vigente_hasta,activa,
   actor_ref,decision_ref,decision_sha256,recibo_ref)
 VALUES(p_version_esperada+1,p_version_esperada,p_huella_esperada,p_huella_nueva,
   p_preparacion_ref,p_revision_ref,p_fuente_ref,p_fuente_sha256,p_custodia_ref,p_acto_ref,
   p_vigente_desde,p_vigente_hasta,p_activa,p_actor_ref,p_decision_ref,p_decision_sha256,p_recibo_ref);
 evento:=pg_catalog.jsonb_build_object('accion','provisionar_perfiles_rpt','version',p_version_esperada+1,
   'huella_sha256',p_huella_nueva,'fuente_ref',p_fuente_ref,'fuente_sha256',p_fuente_sha256,
   'acto_ref',p_acto_ref,'recibo_ref',p_recibo_ref);
 INSERT INTO vec_catalogos_configurables.rpt_perfiles_outbox(recibo_ref,version,evento,evento_sha256)
 VALUES(p_recibo_ref,p_version_esperada+1,evento,
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(evento::text,'UTF8')),'hex'));
 RETURN pg_catalog.jsonb_build_object('version',p_version_esperada+1,'huella_sha256',p_huella_nueva,
    'recibo_ref',p_recibo_ref,'registrada_en',(SELECT registrada_en FROM vec_catalogos_configurables.rpt_perfiles_historia
       WHERE recibo_ref=p_recibo_ref),'decision_ref',p_decision_ref,'decision_sha256',p_decision_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.preparar_provision_perfiles_rpt(text,text,bigint,text,text,text,text,timestamptz,timestamptz,boolean),
 vec_catalogos_configurables.provisionar_perfiles_rpt(text,text,bigint,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,boolean,text)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.preparar_provision_perfiles_rpt(text,text,bigint,text,text,text,text,timestamptz,timestamptz,boolean),
 vec_catalogos_configurables.provisionar_perfiles_rpt(text,text,bigint,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,boolean,text)
 TO vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_catalogos_configurables.rpt_propuesta_guardar_fuente()
RETURNS trigger LANGUAGE plpgsql
SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF (pg_catalog.to_jsonb(NEW) - ARRAY['revision','estado','aprobado_por','aprobado_perfil',
       'publicado_por','aprobacion_decision_ref','aprobacion_decision_sha256',
       'confirmacion_decision_ref','confirmacion_decision_sha256','publicacion_recibo_ref',
       'aprobada_en','confirmada_en'])
    IS DISTINCT FROM
    (pg_catalog.to_jsonb(OLD) - ARRAY['revision','estado','aprobado_por','aprobado_perfil',
       'publicado_por','aprobacion_decision_ref','aprobacion_decision_sha256',
       'confirmacion_decision_ref','confirmacion_decision_sha256','publicacion_recibo_ref',
       'aprobada_en','confirmada_en'])
    OR NEW.revision<>OLD.revision+1
    OR (OLD.estado='propuesta' AND NEW.estado<>'aprobada')
    OR (OLD.estado='aprobada' AND NEW.estado<>'confirmada')
    OR OLD.estado='confirmada' THEN
  RAISE EXCEPTION 'CC12: propuesta o fuente inmutable' USING ERRCODE='55000';
 END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER rpt_propuesta_fuente_inmutable BEFORE UPDATE ON vec_catalogos_configurables.rpt_propuesta
 FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rpt_propuesta_guardar_fuente();
REVOKE ALL ON FUNCTION vec_catalogos_configurables.rpt_propuesta_guardar_fuente() FROM PUBLIC;

-- La preparacion produce la misma representacion JSONB que el efecto. No
-- concede competencia y no lee bytes ajenos: recibe material del servidor.
CREATE FUNCTION vec_catalogos_configurables.preparar_propuesta_rpt(p_contenido jsonb,p_fuente_bytes bytea,p_fuente_meta jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE c jsonb; d jsonb; entrada jsonb; pre jsonb; doc text; h_doc text; h_pre text; h_fuente text;
BEGIN
 IF p_contenido IS NULL OR pg_catalog.jsonb_typeof(p_contenido) IS DISTINCT FROM 'object'
    OR pg_catalog.octet_length(p_contenido::text) > 17825792
    OR p_fuente_bytes IS NULL OR pg_catalog.octet_length(p_fuente_bytes) NOT BETWEEN 1 AND 16777216
    OR p_fuente_meta IS NULL OR pg_catalog.jsonb_typeof(p_fuente_meta) IS DISTINCT FROM 'object'
    OR pg_catalog.octet_length(p_fuente_meta::text) > 4096 THEN
  RAISE EXCEPTION 'CC12: material invalido' USING ERRCODE='22023'; END IF;
 -- Un retiro/deshabilitacion exige otro acto y otra migracion.
 IF p_contenido->>'accion' IS DISTINCT FROM 'publicar'
    OR (p_contenido->>'catalogo_id' ~ '^[a-z][a-z0-9_.:-]{2,127}$') IS NOT TRUE
    OR (p_contenido->>'modulo_id' ~ '^[a-z][a-z0-9_.:-]{2,127}$') IS NOT TRUE
    OR (p_contenido->>'version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR p_contenido->>'fuente_ref' IS DISTINCT FROM p_fuente_meta->>'fuente_ref'
    OR (p_fuente_meta->>'clase' ~ '^[a-z][a-z0-9_]{2,63}$') IS NOT TRUE
    OR p_fuente_meta->>'clase' IS DISTINCT FROM ALL (ARRAY['ejercicio','tecnica'])
    OR (pg_catalog.octet_length(p_fuente_meta->>'procedencia_ref') BETWEEN 3 AND 320) IS NOT TRUE
    OR (pg_catalog.octet_length(p_fuente_meta->>'custodia_ref') BETWEEN 3 AND 320) IS NOT TRUE
    OR (pg_catalog.octet_length(p_fuente_meta->>'organizacion_ref') BETWEEN 3 AND 320) IS NOT TRUE
    OR (pg_catalog.octet_length(p_fuente_meta->>'fuente_ref') BETWEEN 3 AND 320) IS NOT TRUE
    OR p_fuente_meta->>'vigente_desde' IS NULL OR p_fuente_meta->>'vigente_hasta' IS NULL
    OR (p_fuente_meta->>'vigente_desde')::timestamptz >= (p_fuente_meta->>'vigente_hasta')::timestamptz
    THEN
  RAISE EXCEPTION 'CC12: fuente no admitida' USING ERRCODE='22023'; END IF;
 IF (p_contenido->>'version')::bigint > 2147483647 THEN
  RAISE EXCEPTION 'CC12: version incompatible' USING ERRCODE='22023'; END IF;
 h_fuente:=pg_catalog.encode(pg_catalog.sha256(p_fuente_bytes),'hex');
 IF p_fuente_meta->>'sha256' IS DISTINCT FROM h_fuente THEN
  RAISE EXCEPTION 'CC12: huella de fuente distinta' USING ERRCODE='22023'; END IF;
 doc:=p_contenido->>'documento_canonico';
 IF doc IS NULL OR pg_catalog.octet_length(doc) NOT BETWEEN 2 AND 16777216 THEN
  RAISE EXCEPTION 'CC12: documento invalido' USING ERRCODE='22023'; END IF;
 h_doc:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc,'UTF8')),'hex');
 IF p_contenido->>'documento_huella_sha256' IS DISTINCT FROM h_doc THEN
  RAISE EXCEPTION 'CC12: huella documental distinta' USING ERRCODE='22023'; END IF;
 d:=doc::jsonb;
 IF pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'id' IS DISTINCT FROM p_contenido->>'catalogo_id'
    OR d->>'modulo_id' IS DISTINCT FROM p_contenido->>'modulo_id'
    OR d->>'version' IS DISTINCT FROM p_contenido->>'version'
    OR d->>'fuente_ref' IS DISTINCT FROM p_fuente_meta->>'fuente_ref'
    OR d->>'estado' IS DISTINCT FROM 'publicado'
    OR pg_catalog.jsonb_typeof(d->'entradas') IS DISTINCT FROM 'array'
    OR (pg_catalog.octet_length(d->>'creado_por') BETWEEN 3 AND 160) IS NOT TRUE
    OR (pg_catalog.octet_length(d->>'publicado_por') BETWEEN 3 AND 160) IS NOT TRUE
    OR d->>'creado_por'=d->>'publicado_por' THEN
  RAISE EXCEPTION 'CC12: documento de categoria incompatible' USING ERRCODE='22023'; END IF;
 IF pg_catalog.jsonb_array_length(d->'entradas') <> 1 THEN
  RAISE EXCEPTION 'CC12: numero de entradas incompatible' USING ERRCODE='22023'; END IF;
 entrada:=d->'entradas'->0;
 IF pg_catalog.jsonb_typeof(entrada) IS DISTINCT FROM 'object'
    OR (entrada->>'clave' ~ '^[a-z][a-z0-9_.:-]{2,127}$') IS NOT TRUE
    OR (pg_catalog.octet_length(entrada->>'etiqueta') BETWEEN 1 AND 2048) IS NOT TRUE
    OR pg_catalog.jsonb_typeof(entrada->'atributos') IS DISTINCT FROM 'object'
    OR entrada->'atributos'->>'organizacion_ref' IS DISTINCT FROM p_fuente_meta->>'organizacion_ref'
    OR entrada->'atributos'->>'estado' IS DISTINCT FROM 'habilitada' THEN
  RAISE EXCEPTION 'CC12: categoria u organizacion incompatible' USING ERRCODE='22023'; END IF;
 pre:=p_contenido->'preimagenes_control';
 IF pg_catalog.jsonb_typeof(pre) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(pre::text)>1048576
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(pre))>1
    OR ((SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(pre))=1 AND NOT pre ? (entrada->>'clave'))
    OR p_contenido->>'categoria_id' IS NOT NULL OR p_contenido->>'revision_esperada' IS NOT NULL THEN
  RAISE EXCEPTION 'CC12: preimagen incompatible' USING ERRCODE='22023'; END IF;
 h_pre:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex');
 IF p_contenido->>'preimagenes_huella_sha256' IS NULL
    OR p_contenido->>'preimagenes_huella_sha256' NOT IN ('',h_pre) THEN
  RAISE EXCEPTION 'CC12: huella de preimagen distinta' USING ERRCODE='22023'; END IF;
 c:=pg_catalog.jsonb_set(p_contenido,'{preimagenes_huella_sha256}',pg_catalog.to_jsonb(h_pre),true);
 RETURN pg_catalog.jsonb_build_object('contenido',c,'huella_sha256',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(c::text,'UTF8')),'hex'),
   'fuente_sha256',h_fuente,'categoria_id',entrada->>'clave',
   'organizacion_ref',p_fuente_meta->>'organizacion_ref');
END $f$;

CREATE FUNCTION vec_catalogos_configurables.preparar_avance_rpt(p_propuesta_ref text,p_huella text,p_revision bigint,p_accion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE p vec_catalogos_configurables.rpt_propuesta%ROWTYPE; anterior vec_catalogos_configurables.rpt_gobierno_historia%ROWTYPE;
BEGIN
 IF p_propuesta_ref IS NULL OR pg_catalog.octet_length(p_propuesta_ref) NOT BETWEEN 3 AND 160
    OR p_huella !~ '^[0-9a-f]{64}$' OR p_accion NOT IN ('aprobar','confirmar')
    OR p_revision NOT IN (1,2) THEN
  RAISE EXCEPTION 'CC12: avance invalido' USING ERRCODE='22023'; END IF;
 SELECT * INTO p FROM vec_catalogos_configurables.rpt_propuesta WHERE propuesta_ref=p_propuesta_ref;
 IF NOT FOUND OR p.contenido_sha256<>p_huella
    OR (p_accion='aprobar' AND p_revision<>1)
    OR (p_accion='confirmar' AND p_revision<>2) THEN
  RAISE EXCEPTION 'CC12: avance obsoleto' USING ERRCODE='40001'; END IF;
 SELECT * INTO anterior FROM vec_catalogos_configurables.rpt_gobierno_historia
  WHERE propuesta_ref=p_propuesta_ref AND accion=p_accion;
 IF NOT FOUND AND (p.revision<>p_revision
    OR (p_accion='aprobar' AND p.estado<>'propuesta')
    OR (p_accion='confirmar' AND p.estado<>'aprobada')
    OR pg_catalog.clock_timestamp() NOT BETWEEN p.fuente_vigente_desde AND p.fuente_vigente_hasta) THEN
  RAISE EXCEPTION 'CC12: avance obsoleto' USING ERRCODE='40001'; END IF;
 RETURN pg_catalog.jsonb_build_object('propuesta_ref',p.propuesta_ref,'catalogo_id',p.catalogo_id,
   'modulo_id',p.modulo_id,'huella_sha256',p.contenido_sha256,'revision',p.revision,
   'categoria_id',p.categoria_id,'organizacion_ref',p.organizacion_ref,'fuente_sha256',p.fuente_sha256);
END $f$;

CREATE FUNCTION vec_catalogos_configurables.efectuar_gobierno_rpt(
 p_accion text,p_propuesta_ref text,p_contenido jsonb,p_fuente_bytes bytea,p_fuente_meta jsonb,
 p_huella text,p_revision_esperada bigint,p_actor text,p_perfil text,p_decision_ref text,
 p_decision_sha256 text,p_recibo_ref text,p_motivo_ref text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE p vec_catalogos_configurables.rpt_propuesta%ROWTYPE; perfiles vec_catalogos_configurables.rpt_perfiles_competencia%ROWTYPE;
 previo vec_catalogos_configurables.rpt_gobierno_historia%ROWTYPE; prep jsonb; c jsonb; meta jsonb;
 control vec_catalogos_configurables.categoria_control%ROWTYPE;
 v_revision bigint; v_revision_categoria bigint:=0; v_estado text; v_evento jsonb; v_recibo text;
BEGIN
 IF p_accion NOT IN ('proponer','aprobar','confirmar') OR p_propuesta_ref IS NULL OR pg_catalog.octet_length(p_propuesta_ref) NOT BETWEEN 3 AND 160
    OR p_huella !~ '^[0-9a-f]{64}$'
    OR (p_accion='proponer' AND p_revision_esperada IS DISTINCT FROM 0)
    OR (p_accion='aprobar' AND p_revision_esperada IS DISTINCT FROM 1)
    OR (p_accion='confirmar' AND p_revision_esperada IS DISTINCT FROM 2)
    OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
    OR p_perfil IS NULL OR pg_catalog.octet_length(p_perfil) NOT BETWEEN 3 AND 160
    OR p_decision_ref IS NULL OR pg_catalog.octet_length(p_decision_ref) NOT BETWEEN 3 AND 160
    OR p_decision_sha256 !~ '^[0-9a-f]{64}$'
    OR p_recibo_ref IS NULL OR pg_catalog.octet_length(p_recibo_ref) NOT BETWEEN 3 AND 160
    OR p_motivo_ref IS NULL OR p_motivo_ref !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
  RAISE EXCEPTION 'CC12: efecto invalido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:rpt:'||p_propuesta_ref,0));
 SELECT * INTO perfiles FROM vec_catalogos_configurables.rpt_perfiles_competencia WHERE clave=true FOR SHARE;
 IF NOT FOUND OR NOT perfiles.activa OR pg_catalog.clock_timestamp() NOT BETWEEN perfiles.vigente_desde AND perfiles.vigente_hasta
    OR (p_accion='proponer' AND p_perfil<>perfiles.preparacion_ref)
    OR (p_accion<>'proponer' AND p_perfil<>perfiles.revision_ref) THEN
  RAISE EXCEPTION 'CC12: competencia no vigente' USING ERRCODE='42501'; END IF;
 SELECT * INTO previo FROM vec_catalogos_configurables.rpt_gobierno_historia WHERE recibo_ref=p_recibo_ref;
 IF FOUND AND (previo.propuesta_ref<>p_propuesta_ref OR previo.accion<>p_accion OR previo.actor_ref<>p_actor OR previo.perfil_ref<>p_perfil) THEN
  RAISE EXCEPTION 'CC12: recibo ocupado' USING ERRCODE='23505'; END IF;
 SELECT * INTO p FROM vec_catalogos_configurables.rpt_propuesta WHERE propuesta_ref=p_propuesta_ref FOR UPDATE;
 IF p_accion='proponer' THEN
  IF p_revision_esperada<>0 OR p_contenido IS NULL OR p_fuente_bytes IS NULL OR p_fuente_meta IS NULL THEN
   RAISE EXCEPTION 'CC12: propuesta incompleta' USING ERRCODE='22023'; END IF;
  prep:=vec_catalogos_configurables.preparar_propuesta_rpt(p_contenido,p_fuente_bytes,p_fuente_meta);
  c:=prep->'contenido'; meta:=p_fuente_meta;
  IF c IS DISTINCT FROM p_contenido OR prep->>'huella_sha256' IS DISTINCT FROM p_huella
     OR (c->>'documento_canonico')::jsonb->>'creado_por' IS DISTINCT FROM p_actor
     OR (c->>'documento_canonico')::jsonb->>'publicado_por' IS NULL
     OR (c->>'documento_canonico')::jsonb->>'publicado_por'=p_actor THEN
   RAISE EXCEPTION 'CC12: documento o actor incompatible' USING ERRCODE='22023'; END IF;
  IF p.propuesta_ref IS NOT NULL THEN
   IF p.contenido<>c OR p.contenido_sha256<>p_huella OR p.fuente_bytes<>p_fuente_bytes
      OR p.fuente_sha256<>prep->>'fuente_sha256' OR p.creado_por<>p_actor
      OR p.fuente_clase<>meta->>'clase' OR p.fuente_procedencia_ref<>meta->>'procedencia_ref'
      OR p.fuente_custodia_ref<>meta->>'custodia_ref' OR p.organizacion_ref<>meta->>'organizacion_ref'
      OR p.fuente_vigente_desde<>(meta->>'vigente_desde')::timestamptz
      OR p.fuente_vigente_hasta<>(meta->>'vigente_hasta')::timestamptz
      OR p.perfil_config_sha256<>perfiles.huella_sha256 OR p.perfil_config_version<>perfiles.version THEN
    RAISE EXCEPTION 'CC12: propuesta ocupada' USING ERRCODE='23505'; END IF;
  ELSE
   INSERT INTO vec_catalogos_configurables.rpt_propuesta
    (propuesta_ref,catalogo_id,modulo_id,categoria_id,organizacion_ref,version,accion,contenido,contenido_sha256,
     documento_canonico,documento_sha256,preimagenes,preimagenes_sha256,fuente_ref,fuente_clase,fuente_procedencia_ref,
     fuente_custodia_ref,fuente_bytes,fuente_sha256,fuente_vigente_desde,fuente_vigente_hasta,
     perfil_config_version,perfil_config_sha256,creado_por,creado_perfil,revision,estado,propuesta_decision_ref,propuesta_decision_sha256)
   VALUES (p_propuesta_ref,c->>'catalogo_id',c->>'modulo_id',prep->>'categoria_id',prep->>'organizacion_ref',(c->>'version')::integer,
      c->>'accion',c,p_huella,c->>'documento_canonico',c->>'documento_huella_sha256',
      c->'preimagenes_control',c->>'preimagenes_huella_sha256',meta->>'fuente_ref',meta->>'clase',
      meta->>'procedencia_ref',meta->>'custodia_ref',p_fuente_bytes,prep->>'fuente_sha256',
      (meta->>'vigente_desde')::timestamptz,(meta->>'vigente_hasta')::timestamptz,
      perfiles.version,perfiles.huella_sha256,p_actor,p_perfil,1,'propuesta',p_decision_ref,p_decision_sha256);
   SELECT * INTO p FROM vec_catalogos_configurables.rpt_propuesta WHERE propuesta_ref=p_propuesta_ref FOR UPDATE;
  END IF;
  v_revision:=1; v_estado:='propuesta';
 ELSE
  IF p.propuesta_ref IS NULL OR p_contenido IS NOT NULL OR p_fuente_bytes IS NOT NULL OR p_fuente_meta IS NOT NULL
     OR p.contenido_sha256<>p_huella OR p.perfil_config_sha256<>perfiles.huella_sha256
     OR p.perfil_config_version<>perfiles.version OR p.creado_por=p_actor
     OR (p.contenido->>'documento_canonico')::jsonb->>'publicado_por' IS DISTINCT FROM p_actor
     OR p.fuente_sha256<>pg_catalog.encode(pg_catalog.sha256(p.fuente_bytes),'hex') THEN
   RAISE EXCEPTION 'CC12: material o revision incompatible' USING ERRCODE='40001'; END IF;
  -- El control puede variar mientras se revisa: la publicacion vuelve a
  -- comprobarlo con CAS en CC1 bajo su lock de catalogo.
  IF p_accion='aprobar' THEN
   v_revision:=2; v_estado:='aprobada';
  ELSE
   v_revision:=3; v_estado:='confirmada';
  END IF;
 END IF;
 SELECT * INTO previo FROM vec_catalogos_configurables.rpt_gobierno_historia
  WHERE propuesta_ref=p_propuesta_ref AND accion=p_accion;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor OR previo.perfil_ref<>p_perfil OR previo.contenido_sha256<>p_huella
     OR previo.fuente_sha256<>p.fuente_sha256 THEN
   RAISE EXCEPTION 'CC12: replay incompatible' USING ERRCODE='23505'; END IF;
  RETURN pg_catalog.jsonb_build_object('propuesta_ref',p_propuesta_ref,'huella_sha256',p_huella,
     'revision',previo.revision,'estado',previo.estado,'recibo_ref',previo.recibo_ref,
     'accion',previo.accion,'version',p.version,'revision_categoria',previo.revision_categoria,
     'registrada_en',previo.registrada_en,'decision_ref',previo.decision_ref,'decision_sha256',previo.decision_sha256);
 END IF;
 IF pg_catalog.clock_timestamp() NOT BETWEEN p.fuente_vigente_desde AND p.fuente_vigente_hasta THEN
  RAISE EXCEPTION 'CC12: fuente caducada' USING ERRCODE='42501'; END IF;
 SELECT * INTO control FROM vec_catalogos_configurables.categoria_control WHERE categoria_id=p.categoria_id;
 IF FOUND THEN
  IF control.catalogo_id<>p.catalogo_id OR p.preimagenes->p.categoria_id IS DISTINCT FROM
    pg_catalog.jsonb_build_object('version',control.version,'huella_sha256',control.huella_sha256,
     'revision',control.revision,'estado',control.estado) THEN
   RAISE EXCEPTION 'CC12: preimagen obsoleta' USING ERRCODE='40001'; END IF;
 ELSIF p.preimagenes ? p.categoria_id THEN
  RAISE EXCEPTION 'CC12: preimagen inexistente' USING ERRCODE='40001';
 END IF;
 IF p_accion='aprobar' THEN
  IF p.revision<>1 OR p.estado<>'propuesta' THEN RAISE EXCEPTION 'CC12: revision obsoleta' USING ERRCODE='40001'; END IF;
  UPDATE vec_catalogos_configurables.rpt_propuesta SET revision=2,estado='aprobada',aprobado_por=p_actor,
    aprobado_perfil=p_perfil,aprobacion_decision_ref=p_decision_ref,aprobacion_decision_sha256=p_decision_sha256,
    aprobada_en=pg_catalog.clock_timestamp() WHERE propuesta_ref=p_propuesta_ref AND revision=1;
 ELSIF p_accion='confirmar' THEN
  IF p.revision<>2 OR p.estado<>'aprobada' OR p.aprobado_por<>p_actor OR p.aprobado_perfil<>p_perfil
     OR p.aprobacion_decision_ref=p_decision_ref THEN
   RAISE EXCEPTION 'CC12: confirmacion incompatible' USING ERRCODE='40001'; END IF;
  v_recibo:=vec_catalogos_configurables.publicar(p.catalogo_id,p.version,p.documento_sha256,
    p.documento_canonico,p.preimagenes,p.preimagenes_sha256,p.propuesta_decision_ref,
    p.aprobacion_decision_ref,p_actor,p_decision_ref,p_recibo_ref,p_motivo_ref);
  IF v_recibo<>p_recibo_ref THEN RAISE EXCEPTION 'CC12: recibo de publicacion incompatible' USING ERRCODE='55000'; END IF;
  SELECT h.revision INTO v_revision_categoria
    FROM vec_catalogos_configurables.historia h
    JOIN vec_catalogos_configurables.publicacion pub
      ON pub.recibo_ref=p_recibo_ref AND pub.catalogo_id=p.catalogo_id AND pub.version=p.version
   WHERE h.recibo_ref=p_recibo_ref||':'||p.categoria_id AND h.categoria_id=p.categoria_id
     AND h.accion='publicar' AND h.decision_ref=p_decision_ref AND h.actor_ref=p_actor;
  IF NOT FOUND OR v_revision_categoria IS NULL OR v_revision_categoria<1 THEN
   RAISE EXCEPTION 'CC12: historia de publicacion ausente' USING ERRCODE='55000'; END IF;
  UPDATE vec_catalogos_configurables.rpt_propuesta SET revision=3,estado='confirmada',publicado_por=p_actor,
    confirmacion_decision_ref=p_decision_ref,confirmacion_decision_sha256=p_decision_sha256,
    publicacion_recibo_ref=p_recibo_ref,confirmada_en=pg_catalog.clock_timestamp()
    WHERE propuesta_ref=p_propuesta_ref AND revision=2;
 END IF;
 INSERT INTO vec_catalogos_configurables.rpt_gobierno_historia
  (propuesta_ref,revision,accion,estado,actor_ref,perfil_ref,decision_ref,decision_sha256,
   recibo_ref,motivo_ref,contenido_sha256,fuente_sha256,revision_categoria)
 VALUES(p_propuesta_ref,v_revision,p_accion,v_estado,p_actor,p_perfil,p_decision_ref,p_decision_sha256,
   p_recibo_ref,p_motivo_ref,p_huella,p.fuente_sha256,v_revision_categoria);
 v_evento:=pg_catalog.jsonb_build_object('propuesta_ref',p_propuesta_ref,'revision',v_revision,
    'accion',p_accion,'estado',v_estado,'catalogo_id',p.catalogo_id,'categoria_id',p.categoria_id,
    'organizacion_ref',p.organizacion_ref,'contenido_sha256',p_huella,'fuente_sha256',p.fuente_sha256,
    'recibo_ref',p_recibo_ref);
 INSERT INTO vec_catalogos_configurables.rpt_gobierno_outbox(recibo_ref,propuesta_ref,accion,evento,evento_sha256)
 VALUES(p_recibo_ref,p_propuesta_ref,p_accion,v_evento,
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_evento::text,'UTF8')),'hex'));
 RETURN pg_catalog.jsonb_build_object('propuesta_ref',p_propuesta_ref,'huella_sha256',p_huella,
   'revision',v_revision,'estado',v_estado,'recibo_ref',p_recibo_ref,'accion',p_accion,
   'version',p.version,'revision_categoria',v_revision_categoria,
   'decision_ref',p_decision_ref,'decision_sha256',p_decision_sha256,
   'registrada_en',(SELECT registrada_en FROM vec_catalogos_configurables.rpt_gobierno_historia
      WHERE propuesta_ref=p_propuesta_ref AND accion=p_accion));
END $f$;

REVOKE ALL ON FUNCTION vec_catalogos_configurables.preparar_propuesta_rpt(jsonb,bytea,jsonb),
 vec_catalogos_configurables.preparar_avance_rpt(text,text,bigint,text),
 vec_catalogos_configurables.efectuar_gobierno_rpt(text,text,jsonb,bytea,jsonb,text,bigint,text,text,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_catalogos_configurables_rpt_consumidor;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.preparar_propuesta_rpt(jsonb,bytea,jsonb),
 vec_catalogos_configurables.preparar_avance_rpt(text,text,bigint,text)
 TO vec_catalogos_configurables_rpt_consumidor, vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.efectuar_gobierno_rpt(text,text,jsonb,bytea,jsonb,text,bigint,text,text,text,text,text,text)
 TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
