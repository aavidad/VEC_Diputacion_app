\set ON_ERROR_STOP on
-- S2 BC8: preparación incompleta, sin aprobación, publicación o activación.
-- Sólo bytes y controles durables. Los pendientes se derivan exclusivamente en Go.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000008',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR to_regnamespace('vec_bolsa_convocatorias') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.preparacion_bases_version_v3') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.rechazar_mutacion_inmutable()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_guardar_preparacion_bases_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consultar_preparacion_bases_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regrole('vec_bolsa_convocatorias_ejecutor_preparacion_bases') IS NULL
 OR to_regrole('vec_bolsa_convocatorias_lector_preparacion_bases') IS NULL
 THEN RAISE EXCEPTION 'BC8: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_version_v3 (
 preparacion_ref text COLLATE "C" NOT NULL CHECK(preparacion_ref~'^[A-Za-z0-9:_./-]{1,180}$'),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 1000000),
 organizacion_ref text COLLATE "C" NOT NULL CHECK(organizacion_ref~'^org_[a-z0-9]{16,80}$'),
 unidad_gestion_ref text COLLATE "C" NOT NULL CHECK(unidad_gestion_ref='' OR unidad_gestion_ref~'^uni_[a-z0-9]{16,80}$'),
 huella_material_sha256 text NOT NULL CHECK(huella_material_sha256~'^[0-9a-f]{64}$'),
 material_canonico bytea NOT NULL CHECK(octet_length(material_canonico) BETWEEN 2 AND 262272
   AND encode(sha256(material_canonico),'hex')=huella_material_sha256),
 confirmada_en timestamptz(6) NOT NULL CHECK(isfinite(confirmada_en)),
 PRIMARY KEY(preparacion_ref,revision),
 UNIQUE(preparacion_ref,revision,huella_material_sha256,organizacion_ref,unidad_gestion_ref)
);
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_actual_v3 (
 preparacion_ref text COLLATE "C" PRIMARY KEY,
 organizacion_ref text COLLATE "C" NOT NULL,
 unidad_gestion_ref text COLLATE "C" NOT NULL,
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 1000000),
 huella_material_sha256 text NOT NULL,
 actualizada_en timestamptz(6) NOT NULL CHECK(isfinite(actualizada_en)),
 FOREIGN KEY(preparacion_ref,revision,huella_material_sha256,organizacion_ref,unidad_gestion_ref)
 REFERENCES vec_bolsa_convocatorias.preparacion_bases_version_v3(preparacion_ref,revision,huella_material_sha256,organizacion_ref,unidad_gestion_ref)
);
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_historia_v3 (
 historia_ref text PRIMARY KEY CHECK(historia_ref~'^historia_preparacion_bases_[0-9a-f]{64}$'),
 preparacion_ref text COLLATE "C" NOT NULL,
 revision bigint NOT NULL,
 actor_ref text COLLATE "C" NOT NULL,
 auditoria_efecto_ref text NOT NULL UNIQUE,
 confirmada_en timestamptz(6) NOT NULL CHECK(isfinite(confirmada_en)),
 UNIQUE(preparacion_ref,revision), UNIQUE(historia_ref,preparacion_ref,revision),
 FOREIGN KEY(preparacion_ref,revision) REFERENCES vec_bolsa_convocatorias.preparacion_bases_version_v3(preparacion_ref,revision)
);
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_outbox_v3 (
 evento_ref text PRIMARY KEY CHECK(evento_ref~'^evento_preparacion_bases_[0-9a-f]{64}$'),
 preparacion_ref text COLLATE "C" NOT NULL,
 revision bigint NOT NULL,
 historia_ref text NOT NULL,
 esquema text NOT NULL CHECK(esquema='vec.bolsa.preparacion-bases.guardada.v3'),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en)),
 UNIQUE(preparacion_ref,revision), UNIQUE(evento_ref,preparacion_ref,revision),
 FOREIGN KEY(historia_ref,preparacion_ref,revision) REFERENCES vec_bolsa_convocatorias.preparacion_bases_historia_v3(historia_ref,preparacion_ref,revision)
);
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_recibo_v3 (
 actor_ref text COLLATE "C" NOT NULL,
 clave_operacion text COLLATE "C" NOT NULL CHECK(clave_operacion~'^[A-Za-z0-9:_./-]{1,128}$'),
 huella_intencion_sha256 text NOT NULL CHECK(huella_intencion_sha256~'^[0-9a-f]{64}$'),
 preparacion_ref text COLLATE "C" NOT NULL,
 revision bigint NOT NULL,
 huella_material_sha256 text NOT NULL,
 organizacion_ref text COLLATE "C" NOT NULL,
 unidad_gestion_ref text COLLATE "C" NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^recibo_preparacion_bases_[0-9a-f]{64}$'),
 historia_ref text NOT NULL UNIQUE,
 auditoria_efecto_ref text NOT NULL UNIQUE,
 evento_ref text NOT NULL UNIQUE,
 confirmada_en timestamptz(6) NOT NULL CHECK(isfinite(confirmada_en)),
 PRIMARY KEY(actor_ref,clave_operacion), UNIQUE(preparacion_ref,revision),
 FOREIGN KEY(preparacion_ref,revision,huella_material_sha256,organizacion_ref,unidad_gestion_ref)
 REFERENCES vec_bolsa_convocatorias.preparacion_bases_version_v3(preparacion_ref,revision,huella_material_sha256,organizacion_ref,unidad_gestion_ref),
 FOREIGN KEY(historia_ref,preparacion_ref,revision) REFERENCES vec_bolsa_convocatorias.preparacion_bases_historia_v3(historia_ref,preparacion_ref,revision),
 FOREIGN KEY(evento_ref,preparacion_ref,revision) REFERENCES vec_bolsa_convocatorias.preparacion_bases_outbox_v3(evento_ref,preparacion_ref,revision)
);
CREATE TABLE vec_bolsa_convocatorias.preparacion_bases_acceso_v3 (
 recibo_acceso_ref text PRIMARY KEY CHECK(recibo_acceso_ref~'^acceso_preparacion_bases_[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_acceso_ref text NOT NULL UNIQUE,
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 organizacion_ref text NOT NULL,
 unidad_gestion_ref text NOT NULL,
 preparacion_solicitada_ref text NOT NULL,
 operacion text NOT NULL CHECK(operacion IN('guardar','consultar')),
 resultado text NOT NULL CHECK(resultado IN('guardada','recuperada','version_en_conflicto','clave_reutilizada','obtenida','no_encontrada')),
 material_envelope_sha256 text NOT NULL CHECK(material_envelope_sha256~'^[0-9a-f]{64}$'),
 decision_sha256 text NOT NULL CHECK(decision_sha256~'^[0-9a-f]{64}$'),
 correlacion_ref text NOT NULL,
 accedida_en timestamptz(6) NOT NULL CHECK(isfinite(accedida_en))
);
CREATE FUNCTION vec_bolsa_convocatorias.avanzar_preparacion_bases_actual_v3()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF TG_OP<>'UPDATE' OR current_user<>'vec_bolsa_convocatorias_propietario'
 OR NEW.preparacion_ref IS DISTINCT FROM OLD.preparacion_ref
 OR NEW.organizacion_ref IS DISTINCT FROM OLD.organizacion_ref
 OR NEW.unidad_gestion_ref IS DISTINCT FROM OLD.unidad_gestion_ref
 OR NEW.revision IS DISTINCT FROM OLD.revision+1
 OR NEW.actualizada_en<OLD.actualizada_en
 THEN RAISE EXCEPTION 'BC8: avance de preparación incompatible' USING ERRCODE='55000'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.avanzar_preparacion_bases_actual_v3() FROM PUBLIC;
DO $cerrar$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['preparacion_bases_version_v3','preparacion_bases_actual_v3','preparacion_bases_historia_v3','preparacion_bases_outbox_v3','preparacion_bases_recibo_v3','preparacion_bases_acceso_v3'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_convocatorias.%I FROM PUBLIC,vec_bolsa_convocatorias_ejecutor_preparacion_bases,vec_bolsa_convocatorias_lector_preparacion_bases',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_convocatorias.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE vec_bolsa_convocatorias.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_convocatorias.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_bolsa_convocatorias.%I FOR ALL TO vec_bolsa_convocatorias_propietario USING(current_user=''vec_bolsa_convocatorias_propietario'') WITH CHECK(current_user=''vec_bolsa_convocatorias_propietario'')',t);
  IF t='preparacion_bases_actual_v3' THEN
   EXECUTE format('CREATE TRIGGER avance_exacto BEFORE UPDATE OR DELETE ON vec_bolsa_convocatorias.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.avanzar_preparacion_bases_actual_v3()',t);
  ELSE
   EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_convocatorias.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',t);
  END IF;
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',t);
 END LOOP;
END $cerrar$;

CREATE FUNCTION vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(p_material text,p_contenido bytea,p_guardar boolean)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; c jsonb; canon text; ambito text; intencion text; rev numeric;
BEGIN
 IF p_guardar IS NULL OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'BC8: material fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'BC8: envelope inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR jsonb_typeof(m->'unidad_gestion_ref') IS DISTINCT FROM 'string'
 OR coalesce(m->>'preparacion_ref','') !~ '^[A-Za-z0-9:_./-]{1,180}$'
 OR coalesce(m->>'organizacion_ref','') !~ '^org_[a-z0-9]{16,80}$'
 OR (coalesce(m->>'unidad_gestion_ref','')<>'' AND m->>'unidad_gestion_ref' !~ '^uni_[a-z0-9]{16,80}$')
 OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(m->>'contexto_actor_ref','') !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR coalesce(m->>'correlacion_ref','') !~ '^correlacion_[0-9a-f]{32}$'
 OR jsonb_typeof(m->'contexto_version') IS DISTINCT FROM 'number'
 OR jsonb_typeof(m->'persona_version') IS DISTINCT FROM 'number'
 OR jsonb_typeof(m->'perfil_version') IS DISTINCT FROM 'number'
 OR coalesce(m->>'contexto_version','') !~ '^[1-9][0-9]{0,19}$'
 OR coalesce(m->>'persona_version','') !~ '^[1-9][0-9]{0,15}$'
 OR coalesce(m->>'perfil_version','') !~ '^[1-9][0-9]{0,15}$'
 THEN RAISE EXCEPTION 'BC8: identidades o ámbito inválidos' USING ERRCODE='22023'; END IF;
 IF (m->>'contexto_version')::numeric>18446744073709551615
 OR (m->>'persona_version')::numeric>9007199254740991
 OR (m->>'perfil_version')::numeric>9007199254740991
 THEN RAISE EXCEPTION 'BC8: versiones fuera de límites' USING ERRCODE='22023'; END IF;
 canon:=',"actor_ref":'||to_json(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_json(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"persona_version":'||(m->>'persona_version')||
  ',"perfil_ref":'||to_json(m->>'perfil_ref')::text||',"perfil_version":'||(m->>'perfil_version')||
  ',"correlacion_ref":'||to_json(m->>'correlacion_ref')::text||'}';
 IF p_guardar THEN
  IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
    'actor_ref','clave_operacion','contexto_actor_ref','contexto_version','correlacion_ref','esquema',
    'huella_esperada_sha256','huella_intencion_sha256','huella_material_sha256','organizacion_ref',
    'perfil_ref','perfil_version','persona_version','preparacion_ref','revision_esperada','unidad_gestion_ref']
  OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.preparacion-bases.guardar.v3'
  OR jsonb_typeof(m->'revision_esperada') IS DISTINCT FROM 'number'
  OR coalesce(m->>'revision_esperada','') !~ '^(0|[1-9][0-9]{0,5})$'
  OR coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9:_./-]{1,128}$'
  OR coalesce(m->>'huella_material_sha256','') !~ '^[0-9a-f]{64}$'
  OR coalesce(m->>'huella_intencion_sha256','') !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(m->'huella_esperada_sha256') IS DISTINCT FROM 'string'
  OR (m->>'revision_esperada'='0' AND m->>'huella_esperada_sha256'<>'')
  OR (m->>'revision_esperada'<>'0' AND coalesce(m->>'huella_esperada_sha256','') !~ '^[0-9a-f]{64}$')
  OR p_contenido IS NULL OR octet_length(p_contenido) NOT BETWEEN 2 AND 262272
  OR encode(sha256(p_contenido),'hex') IS DISTINCT FROM m->>'huella_material_sha256'
  THEN RAISE EXCEPTION 'BC8: guardado inválido' USING ERRCODE='22023'; END IF;
  BEGIN c:=convert_from(p_contenido,'UTF8')::jsonb;
  EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'BC8: contenido inválido' USING ERRCODE='22023'; END;
  -- Forma del wrapper; ninguna política de completitud ni aprobación vive aquí.
  IF jsonb_typeof(c) IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(c) ORDER BY 1) IS DISTINCT FROM ARRAY['esquema','material']
  OR c->>'esquema' IS DISTINCT FROM 'bolsa.preparacion_bases.material.v1'
  OR jsonb_typeof(c->'material') IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(c->'material') ORDER BY 1) IS DISTINCT FROM ARRAY['contenido','referencias']
  OR jsonb_typeof(c#>'{material,contenido}') IS DISTINCT FROM 'object'
  OR jsonb_typeof(c#>'{material,referencias}') IS DISTINCT FROM 'array'
  THEN RAISE EXCEPTION 'BC8: wrapper de material inválido' USING ERRCODE='22023'; END IF;
  rev:=(m->>'revision_esperada')::numeric;
  ambito:='{"organizacion_ref":'||to_json(m->>'organizacion_ref')::text||
    CASE WHEN m->>'unidad_gestion_ref'='' THEN '' ELSE ',"unidad_gestion_ref":'||to_json(m->>'unidad_gestion_ref')::text END||'}';
  intencion:='{"ambito":'||ambito||',"esquema":"bolsa.preparacion_bases.intencion.v1","esperada":{"preparacion_ref":'||to_json(m->>'preparacion_ref')::text||
    ',"revision":'||rev::text||',"huella_material_sha256":'||to_json(m->>'huella_esperada_sha256')::text||'},"huella_material":'||to_json(m->>'huella_material_sha256')::text||'}';
  IF encode(sha256(convert_to(intencion,'UTF8')),'hex') IS DISTINCT FROM m->>'huella_intencion_sha256'
  THEN RAISE EXCEPTION 'BC8: intención divergente' USING ERRCODE='22023'; END IF;
  canon:='{"esquema":"vec.bolsa.preparacion-bases.guardar.v3","preparacion_ref":'||to_json(m->>'preparacion_ref')::text||
    ',"organizacion_ref":'||to_json(m->>'organizacion_ref')::text||',"unidad_gestion_ref":'||to_json(m->>'unidad_gestion_ref')::text||
    ',"revision_esperada":'||rev::text||',"huella_esperada_sha256":'||to_json(m->>'huella_esperada_sha256')::text||
    ',"huella_material_sha256":'||to_json(m->>'huella_material_sha256')::text||',"huella_intencion_sha256":'||to_json(m->>'huella_intencion_sha256')::text||
    ',"clave_operacion":'||to_json(m->>'clave_operacion')::text||canon;
 ELSE
  IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
    'actor_ref','contexto_actor_ref','contexto_version','correlacion_ref','esquema','huella_material_sha256',
    'modo','organizacion_ref','perfil_ref','perfil_version','persona_version','preparacion_ref','revision','unidad_gestion_ref']
  OR m->>'esquema' IS DISTINCT FROM 'vec.bolsa.preparacion-bases.consultar.v3'
  OR coalesce(m->>'modo','') NOT IN('actual','exacta')
  OR jsonb_typeof(m->'revision') IS DISTINCT FROM 'number'
  OR coalesce(m->>'revision','') !~ '^(0|[1-9][0-9]{0,6})$'
  OR jsonb_typeof(m->'huella_material_sha256') IS DISTINCT FROM 'string'
  OR (m->>'modo'='actual' AND (m->>'revision'<>'0' OR m->>'huella_material_sha256'<>''))
  OR (m->>'modo'='exacta' AND ((m->>'revision')::numeric NOT BETWEEN 1 AND 1000000 OR coalesce(m->>'huella_material_sha256','') !~ '^[0-9a-f]{64}$'))
  OR p_contenido IS NOT NULL
  THEN RAISE EXCEPTION 'BC8: consulta inválida' USING ERRCODE='22023'; END IF;
  canon:='{"esquema":"vec.bolsa.preparacion-bases.consultar.v3","preparacion_ref":'||to_json(m->>'preparacion_ref')::text||
    ',"organizacion_ref":'||to_json(m->>'organizacion_ref')::text||',"unidad_gestion_ref":'||to_json(m->>'unidad_gestion_ref')::text||
    ',"modo":'||to_json(m->>'modo')::text||',"revision":'||(m->>'revision')||',"huella_material_sha256":'||to_json(m->>'huella_material_sha256')::text||canon;
 END IF;
 IF canon IS DISTINCT FROM p_material THEN RAISE EXCEPTION 'BC8: envelope no canónico' USING ERRCODE='22023'; END IF;
 RETURN m;
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'BC8: material fuera del contrato' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(text,bytea,boolean) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_convocatorias.consumir_preparacion_bases_v3_interna(
 m jsonb,p_material text,p_guardar boolean,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb;d jsonb;ctx jsonb;x record;grupo text;accion text;audiencia text;campos jsonb;recurso text;h text;
BEGIN
 grupo:=CASE WHEN p_guardar THEN 'vec_bolsa_convocatorias_ejecutor_preparacion_bases' ELSE 'vec_bolsa_convocatorias_lector_preparacion_bases' END;
 accion:=CASE WHEN p_guardar THEN 'bolsa.preparacion_bases.guardar' ELSE 'bolsa.preparacion_bases.consultar' END;
 audiencia:=CASE WHEN p_guardar THEN 'vec_bolsa_convocatorias.preparacion_bases.guardar.v1' ELSE 'vec_bolsa_convocatorias.preparacion_bases.consultar.v1' END;
 campos:=CASE WHEN p_guardar THEN '["auditoria","evento_outbox","historia","material_preparacion","recibo_preparacion"]'::jsonb ELSE '["material_preparacion","recibo_preparacion"]'::jsonb END;
 IF current_user<>'vec_bolsa_convocatorias_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR p_guardar IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid=grupo::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=grupo::regrole)
 THEN RAISE EXCEPTION 'BC8: sesión o transacción incompatible' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'BC8: material V3 inválido' USING ERRCODE='22023'; END;
 recurso:='{"ambitos":{"organizacion_ref":'||to_json(m->>'organizacion_ref')::text||
  CASE WHEN m->>'unidad_gestion_ref'='' THEN '' ELSE ',"unidad_gestion_ref":'||to_json(m->>'unidad_gestion_ref')::text END||
  '},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}';
 h:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
 OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
 OR m->>'contexto_actor_ref' IS DISTINCT FROM ctx->>'contexto_actor_ref'
 OR m->>'contexto_version' IS DISTINCT FROM ctx->>'contexto_version'
 OR m->>'actor_ref' IS DISTINCT FROM ctx->>'principal_ref'
 OR m->>'persona_version' IS DISTINCT FROM ctx->>'persona_version'
 OR m->>'perfil_version' IS DISTINCT FROM ctx->>'perfil_version'
 OR m->>'perfil_ref' IS DISTINCT FROM ctx->>'perfil_activo_ref'
 OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
 OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 OR m->>'correlacion_ref' IS DISTINCT FROM d->>'correlacion_ref'
 OR d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
 OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'preparacion_bases'
 OR d->>'finalidad' IS DISTINCT FROM 'preparacion_bases'
 OR d->>'recurso_ref' IS DISTINCT FROM m->>'preparacion_ref' OR c->>'efecto_ref' IS DISTINCT FROM m->>'preparacion_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
 OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'false'
 OR d->>'concedida' IS DISTINCT FROM 'true'
 THEN RAISE EXCEPTION 'BC8: material y concesión divergentes' USING ERRCODE='42501'; END IF;
 IF p_guardar THEN
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_guardar_preparacion_bases_bolsa_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_consultar_preparacion_bases_bolsa_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM m->>'preparacion_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM h
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 OR clock_timestamp()>=(d->>'valida_hasta')::timestamptz
 THEN RAISE EXCEPTION 'BC8: consumo fresco incompatible' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.consumir_preparacion_bases_v3_interna(jsonb,text,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_convocatorias.registrar_acceso_preparacion_bases_v3_interna(
 m jsonb,p_material text,p_decision bytea,p_operacion text,p_resultado text,p_decision_ref text,p_consumo text,p_auditoria text,p_instante timestamptz)
RETURNS text LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r text;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario' THEN RAISE EXCEPTION 'BC8: acceso interno denegado' USING ERRCODE='42501'; END IF;
 r:='acceso_preparacion_bases_'||encode(sha256(convert_to(p_decision_ref||':'||p_consumo,'UTF8')),'hex');
 INSERT INTO vec_bolsa_convocatorias.preparacion_bases_acceso_v3 VALUES(r,p_decision_ref,p_consumo,p_auditoria,
  m->>'actor_ref',m->>'perfil_ref',m->>'organizacion_ref',m->>'unidad_gestion_ref',m->>'preparacion_ref',p_operacion,p_resultado,
  encode(sha256(convert_to(p_material,'UTF8')),'hex'),encode(sha256(p_decision),'hex'),m->>'correlacion_ref',p_instante);
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.registrar_acceso_preparacion_bases_v3_interna(jsonb,text,bytea,text,text,text,text,text,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_convocatorias.guardar_preparacion_bases_v3(
 p_material text,p_contenido bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(resultado text,preparacion_ref text,revision bigint,huella_material_sha256 text,material_canonico bytea,
 recibo_ref text,historia_ref text,auditoria_efecto_ref text,evento_ref text,huella_intencion_sha256 text,confirmada_en timestamptz(6),
 decision_ref text,consumo_huella_sha256 text,auditoria_acceso_ref text,recibo_acceso_ref text,correlacion_ref text,accedida_en timestamptz(6))
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE m jsonb;consumo record;codigo text;acceso text;actor text;clave text;prep text;org text;uni text;
 esperado bigint;nueva bigint;rr text;hh text;ee text;encontrada boolean;
 cabeza vec_bolsa_convocatorias.preparacion_bases_actual_v3%ROWTYPE;
 original vec_bolsa_convocatorias.preparacion_bases_recibo_v3%ROWTYPE;
 fuente vec_bolsa_convocatorias.preparacion_bases_version_v3%ROWTYPE;
BEGIN
 m:=vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(p_material,p_contenido,true);
 actor:=m->>'actor_ref';clave:=m->>'clave_operacion';prep:=m->>'preparacion_ref';org:=m->>'organizacion_ref';uni:=m->>'unidad_gestion_ref';
 esperado:=(m->>'revision_esperada')::bigint;
 -- Orden único actor+clave -> agregado -> autoridad. Se recupera antes del CAS.
 PERFORM pg_advisory_xact_lock(hashtextextended('BC8:clave:'||jsonb_build_array(actor,clave)::text,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('BC8:preparacion:'||prep,0));
 SELECT * INTO STRICT consumo FROM vec_bolsa_convocatorias.consumir_preparacion_bases_v3_interna(
  m,p_material,true,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT r.* INTO original FROM vec_bolsa_convocatorias.preparacion_bases_recibo_v3 r
 WHERE r.actor_ref=actor AND r.clave_operacion=clave;
 encontrada:=FOUND;
 IF encontrada THEN
  IF original.preparacion_ref IS DISTINCT FROM prep OR original.organizacion_ref IS DISTINCT FROM org
   OR original.unidad_gestion_ref IS DISTINCT FROM uni
   OR original.huella_intencion_sha256 IS DISTINCT FROM m->>'huella_intencion_sha256'
   OR original.huella_material_sha256 IS DISTINCT FROM m->>'huella_material_sha256'
  THEN codigo:='clave_reutilizada';
  ELSE
   SELECT v.* INTO STRICT fuente FROM vec_bolsa_convocatorias.preparacion_bases_version_v3 v
   WHERE v.preparacion_ref=original.preparacion_ref AND v.revision=original.revision
   AND v.huella_material_sha256=original.huella_material_sha256
   AND v.organizacion_ref=org AND v.unidad_gestion_ref=uni;
   codigo:='recuperada';
  END IF;
 ELSE
  SELECT a.* INTO cabeza FROM vec_bolsa_convocatorias.preparacion_bases_actual_v3 a WHERE a.preparacion_ref=prep FOR UPDATE;
  encontrada:=FOUND;
  IF encontrada AND (cabeza.organizacion_ref IS DISTINCT FROM org OR cabeza.unidad_gestion_ref IS DISTINCT FROM uni)
  THEN RAISE EXCEPTION 'BC8: ámbito almacenado denegado' USING ERRCODE='42501'; END IF;
  IF (NOT encontrada AND esperado<>0) OR (encontrada AND
    (cabeza.revision IS DISTINCT FROM esperado OR cabeza.huella_material_sha256 IS DISTINCT FROM m->>'huella_esperada_sha256'))
  THEN codigo:='version_en_conflicto';
  ELSE
   nueva:=esperado+1;
   rr:='recibo_preparacion_bases_'||consumo.consumo_huella_sha256;
   hh:='historia_preparacion_bases_'||consumo.consumo_huella_sha256;
   ee:='evento_preparacion_bases_'||consumo.consumo_huella_sha256;
   INSERT INTO vec_bolsa_convocatorias.preparacion_bases_version_v3 VALUES(prep,nueva,org,uni,m->>'huella_material_sha256',p_contenido,consumo.consumida_en);
   INSERT INTO vec_bolsa_convocatorias.preparacion_bases_historia_v3 VALUES(hh,prep,nueva,actor,consumo.auditoria_ref,consumo.consumida_en);
   INSERT INTO vec_bolsa_convocatorias.preparacion_bases_outbox_v3 VALUES(ee,prep,nueva,hh,'vec.bolsa.preparacion-bases.guardada.v3',
    jsonb_build_object('esquema','vec.bolsa.preparacion-bases.guardada.v3','preparacion_ref',prep,'revision',nueva,'huella_material_sha256',m->>'huella_material_sha256','recibo_ref',rr),consumo.consumida_en);
   INSERT INTO vec_bolsa_convocatorias.preparacion_bases_recibo_v3 VALUES(actor,clave,m->>'huella_intencion_sha256',prep,nueva,m->>'huella_material_sha256',org,uni,rr,hh,consumo.auditoria_ref,ee,consumo.consumida_en);
   IF encontrada THEN
    UPDATE vec_bolsa_convocatorias.preparacion_bases_actual_v3 a SET revision=nueva,huella_material_sha256=m->>'huella_material_sha256',actualizada_en=consumo.consumida_en
    WHERE a.preparacion_ref=prep AND a.revision=esperado AND a.huella_material_sha256=m->>'huella_esperada_sha256';
    IF NOT FOUND THEN RAISE EXCEPTION 'BC8: CAS perdió el cercado' USING ERRCODE='40001'; END IF;
   ELSE
    INSERT INTO vec_bolsa_convocatorias.preparacion_bases_actual_v3 VALUES(prep,org,uni,nueva,m->>'huella_material_sha256',consumo.consumida_en);
   END IF;
   SELECT r.* INTO STRICT original FROM vec_bolsa_convocatorias.preparacion_bases_recibo_v3 r WHERE r.actor_ref=actor AND r.clave_operacion=clave;
   SELECT v.* INTO STRICT fuente FROM vec_bolsa_convocatorias.preparacion_bases_version_v3 v WHERE v.preparacion_ref=prep AND v.revision=nueva;
   codigo:='guardada';
  END IF;
 END IF;
 IF codigo IN('guardada','recuperada') AND
  (encode(sha256(fuente.material_canonico),'hex') IS DISTINCT FROM fuente.huella_material_sha256
   OR fuente.material_canonico IS DISTINCT FROM p_contenido)
 THEN RAISE EXCEPTION 'BC8: versión original no íntegra' USING ERRCODE='55000'; END IF;
 acceso:=vec_bolsa_convocatorias.registrar_acceso_preparacion_bases_v3_interna(m,p_material,p_decision,'guardar',codigo,
  consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en);
 IF codigo IN('guardada','recuperada') THEN
  RETURN QUERY SELECT codigo,fuente.preparacion_ref,fuente.revision,fuente.huella_material_sha256,fuente.material_canonico,
   original.recibo_ref,original.historia_ref,original.auditoria_efecto_ref,original.evento_ref,original.huella_intencion_sha256,original.confirmada_en,
   consumo.decision_ref::text,consumo.consumo_huella_sha256::text,consumo.auditoria_ref::text,acceso,m->>'correlacion_ref',consumo.consumida_en;
 ELSE
  -- Conflictos autorizados conservan auditoría de acceso y no exponen historial.
  RETURN QUERY SELECT codigo,NULL::text,NULL::bigint,NULL::text,NULL::bytea,NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::timestamptz,
   consumo.decision_ref::text,consumo.consumo_huella_sha256::text,consumo.auditoria_ref::text,acceso,m->>'correlacion_ref',consumo.consumida_en;
 END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.guardar_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_convocatorias.obtener_preparacion_bases_v3(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(resultado text,preparacion_ref text,revision bigint,huella_material_sha256 text,material_canonico bytea,
 recibo_ref text,historia_ref text,auditoria_efecto_ref text,evento_ref text,huella_intencion_sha256 text,confirmada_en timestamptz(6),
 decision_ref text,consumo_huella_sha256 text,auditoria_acceso_ref text,recibo_acceso_ref text,correlacion_ref text,accedida_en timestamptz(6))
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE m jsonb;consumo record;codigo text;acceso text;prep text;org text;uni text;encontrada boolean;rev bigint;
 cabeza vec_bolsa_convocatorias.preparacion_bases_actual_v3%ROWTYPE;
 fuente vec_bolsa_convocatorias.preparacion_bases_version_v3%ROWTYPE;
 original vec_bolsa_convocatorias.preparacion_bases_recibo_v3%ROWTYPE;
BEGIN
 m:=vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(p_material,NULL,false);
 prep:=m->>'preparacion_ref';org:=m->>'organizacion_ref';uni:=m->>'unidad_gestion_ref';
 SELECT * INTO STRICT consumo FROM vec_bolsa_convocatorias.consumir_preparacion_bases_v3_interna(
  m,p_material,false,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- Cabeza y revisión se leen en la misma instantánea SERIALIZABLE del acceso.
 SELECT a.* INTO cabeza FROM vec_bolsa_convocatorias.preparacion_bases_actual_v3 a WHERE a.preparacion_ref=prep;
 encontrada:=FOUND;
 IF encontrada AND (cabeza.organizacion_ref IS DISTINCT FROM org OR cabeza.unidad_gestion_ref IS DISTINCT FROM uni)
 THEN RAISE EXCEPTION 'BC8: ámbito almacenado denegado' USING ERRCODE='42501'; END IF;
 IF encontrada THEN
  rev:=CASE WHEN m->>'modo'='actual' THEN cabeza.revision ELSE (m->>'revision')::bigint END;
  SELECT v.* INTO fuente FROM vec_bolsa_convocatorias.preparacion_bases_version_v3 v
  WHERE v.preparacion_ref=prep AND v.revision=rev AND v.organizacion_ref=org AND v.unidad_gestion_ref=uni
  AND (m->>'modo'='actual' OR v.huella_material_sha256=m->>'huella_material_sha256');
  encontrada:=FOUND;
 END IF;
 codigo:=CASE WHEN encontrada THEN 'obtenida' ELSE 'no_encontrada' END;
 IF encontrada THEN
  IF encode(sha256(fuente.material_canonico),'hex') IS DISTINCT FROM fuente.huella_material_sha256
  THEN RAISE EXCEPTION 'BC8: material no íntegro' USING ERRCODE='55000'; END IF;
  SELECT r.* INTO STRICT original FROM vec_bolsa_convocatorias.preparacion_bases_recibo_v3 r
  WHERE r.preparacion_ref=prep AND r.revision=fuente.revision AND r.organizacion_ref=org AND r.unidad_gestion_ref=uni;
 END IF;
 acceso:=vec_bolsa_convocatorias.registrar_acceso_preparacion_bases_v3_interna(m,p_material,p_decision,'consultar',codigo,
  consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,consumo.consumida_en);
 IF encontrada THEN
  RETURN QUERY SELECT codigo,fuente.preparacion_ref,fuente.revision,fuente.huella_material_sha256,fuente.material_canonico,
   original.recibo_ref,original.historia_ref,original.auditoria_efecto_ref,original.evento_ref,original.huella_intencion_sha256,original.confirmada_en,
   consumo.decision_ref::text,consumo.consumo_huella_sha256::text,consumo.auditoria_ref::text,acceso,m->>'correlacion_ref',consumo.consumida_en;
 ELSE
  RETURN QUERY SELECT codigo,NULL::text,NULL::bigint,NULL::text,NULL::bytea,NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::timestamptz,
   consumo.decision_ref::text,consumo.consumo_huella_sha256::text,consumo.auditoria_ref::text,acceso,m->>'correlacion_ref',consumo.consumida_en;
 END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.obtener_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_convocatorias TO vec_bolsa_convocatorias_ejecutor_preparacion_bases,vec_bolsa_convocatorias_lector_preparacion_bases;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.guardar_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_convocatorias_ejecutor_preparacion_bases;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.obtener_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_convocatorias_lector_preparacion_bases;
DO $acl$
DECLARE x record;f oid;owner oid:='vec_bolsa_convocatorias_propietario'::regrole;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('guardar_preparacion_bases_v3','text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea','vec_bolsa_convocatorias_ejecutor_preparacion_bases'),
  ('obtener_preparacion_bases_v3','text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea','vec_bolsa_convocatorias_lector_preparacion_bases')) q(nombre,firma,rol) LOOP
  f:=to_regprocedure('vec_bolsa_convocatorias.'||x.nombre||'('||x.firma||')');
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=owner AND p.prosecdef AND p.provolatile='v'
      AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=15s'])
  OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
      AND (a.grantee NOT IN(owner,x.rol::regrole) OR a.grantor<>owner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'BC8: ACL de fachada incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
DO $acl_datos$
DECLARE t text;f text;obj oid;
BEGIN
 FOREACH t IN ARRAY ARRAY['preparacion_bases_version_v3','preparacion_bases_actual_v3','preparacion_bases_historia_v3','preparacion_bases_outbox_v3','preparacion_bases_recibo_v3','preparacion_bases_acceso_v3'] LOOP
  obj:=to_regclass('vec_bolsa_convocatorias.'||t);
  IF EXISTS(SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
      WHERE c.oid=obj AND a.grantee<>c.relowner)
  OR EXISTS(SELECT 1 FROM pg_attribute p,LATERAL aclexplode(p.attacl) a WHERE p.attrelid=obj AND a.grantee<>'vec_bolsa_convocatorias_propietario'::regrole)
  OR EXISTS(SELECT 1 FROM pg_type y,LATERAL aclexplode(coalesce(y.typacl,acldefault('T',y.typowner))) a WHERE y.typrelid=obj AND a.grantee<>y.typowner)
  THEN RAISE EXCEPTION 'BC8: datos accesibles fuera del propietario' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'avanzar_preparacion_bases_actual_v3()',
  'validar_material_preparacion_bases_v3(text,bytea,boolean)',
  'consumir_preparacion_bases_v3_interna(jsonb,text,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'registrar_acceso_preparacion_bases_v3_interna(jsonb,text,bytea,text,text,text,text,text,timestamptz)'] LOOP
  obj:=to_regprocedure('vec_bolsa_convocatorias.'||f);
  IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=obj AND a.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'BC8: helper expuesto a runtime' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl_datos$;
COMMIT;
