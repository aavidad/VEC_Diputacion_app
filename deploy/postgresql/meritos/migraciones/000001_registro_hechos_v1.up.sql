\set ON_ERROR_STOP on
-- RUM03: historia del hecho, operación y acceso de recuperación. No acredita.
BEGIN;
SET LOCAL ROLE vec_meritos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000001',0));
DO $pre$ BEGIN
 IF current_user<>'vec_meritos_propietario' OR to_regclass('vec_meritos.hecho_identidad') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'meritos.error.preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_meritos.referencia_valida_v1(p text) RETURNS boolean
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog,pg_temp
AS $$ SELECT p IS NOT NULL AND octet_length(p) BETWEEN 1 AND 256 AND p ~ '^[A-Za-z0-9:._-]+$' $$;
CREATE FUNCTION vec_meritos.claves_exactas_v1(p jsonb, requeridas text[], opcionales text[] DEFAULT '{}'::text[]) RETURNS boolean
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog,pg_temp
AS $$ SELECT jsonb_typeof(p)='object' AND p ?& requeridas AND NOT EXISTS (
 SELECT 1 FROM jsonb_object_keys(p) k WHERE NOT k=ANY(requeridas||opcionales)) $$;

CREATE TABLE vec_meritos.hecho_identidad (
 hecho_ref text PRIMARY KEY CHECK (vec_meritos.referencia_valida_v1(hecho_ref)),
 persona_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(persona_ref)),
 fuente_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(fuente_ref)),
 origen_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(origen_ref)),
 tipo text NOT NULL CHECK (tipo IN ('titulacion','curso_asistencia','curso_superacion','experiencia','idioma','otro')),
 declarante_ref text NOT NULL CHECK (declarante_ref=persona_ref),
 UNIQUE(persona_ref,fuente_ref,origen_ref,tipo), UNIQUE(hecho_ref,persona_ref)
);
CREATE TABLE vec_meritos.hecho_version (
 hecho_ref text NOT NULL,
 persona_ref text NOT NULL,
 version integer NOT NULL CHECK (version BETWEEN 1 AND 1073741824),
 registro jsonb NOT NULL CHECK (jsonb_typeof(registro)='object'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(hecho_ref,version),
 FOREIGN KEY(hecho_ref,persona_ref) REFERENCES vec_meritos.hecho_identidad(hecho_ref,persona_ref),
 CHECK (registro->'hecho'->>'referencia'=hecho_ref AND registro->'hecho'->>'persona_ref'=persona_ref
 AND (registro->'hecho'->>'version')::integer=version)
);
CREATE TABLE vec_meritos.operacion (
 clave text NOT NULL CHECK (vec_meritos.referencia_valida_v1(clave)),
 huella_comando text NOT NULL CHECK (huella_comando ~ '^[0-9a-f]{64}$'),
 comando bytea NOT NULL CHECK (octet_length(comando) BETWEEN 1 AND 65536),
 hecho_ref text NOT NULL,
 persona_ref text NOT NULL,
 version integer NOT NULL,
 actor_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(actor_ref)),
 accion text NOT NULL CHECK (accion IN ('meritos.hecho.declarar','meritos.hecho.rectificar','meritos.hecho.rechazar')),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (vec_meritos.referencia_valida_v1(recibo_ref)),
 recibo jsonb NOT NULL CHECK (jsonb_typeof(recibo)='object'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(actor_ref,clave), UNIQUE(hecho_ref,version),
 FOREIGN KEY(hecho_ref,persona_ref) REFERENCES vec_meritos.hecho_identidad(hecho_ref,persona_ref),
 FOREIGN KEY(hecho_ref,version) REFERENCES vec_meritos.hecho_version(hecho_ref,version)
);
CREATE TABLE vec_meritos.auditoria_operacion (
 auditoria_ref text PRIMARY KEY CHECK (vec_meritos.referencia_valida_v1(auditoria_ref)),
 clave text NOT NULL CHECK (vec_meritos.referencia_valida_v1(clave)),
 persona_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(persona_ref)),
 entrada jsonb NOT NULL CHECK (jsonb_typeof(entrada)='object'),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_meritos.outbox (
 evento_ref text PRIMARY KEY CHECK (vec_meritos.referencia_valida_v1(evento_ref)),
 clave text NOT NULL CHECK (vec_meritos.referencia_valida_v1(clave)),
 actor_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(actor_ref)),
 persona_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(persona_ref)),
 tipo text NOT NULL CHECK (tipo='meritos.hecho.registrado.v1'),
 carga jsonb NOT NULL CHECK (jsonb_typeof(carga)='object'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(actor_ref,clave), FOREIGN KEY(actor_ref,clave) REFERENCES vec_meritos.operacion(actor_ref,clave)
);
CREATE TABLE vec_meritos.acceso_operacion (
 acceso_ref text PRIMARY KEY CHECK (vec_meritos.referencia_valida_v1(acceso_ref)),
 clave text NOT NULL CHECK (vec_meritos.referencia_valida_v1(clave)),
 persona_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(persona_ref)),
 actor_ref text NOT NULL CHECK (vec_meritos.referencia_valida_v1(actor_ref)),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(actor_ref,clave) REFERENCES vec_meritos.operacion(actor_ref,clave)
);

CREATE FUNCTION vec_meritos.historia_inmutable_v1() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN
 RAISE EXCEPTION 'meritos.error.historia_inmutable' USING ERRCODE='55000'; END $$;
DO $seguridad$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['hecho_identidad','hecho_version','operacion','auditoria_operacion','outbox','acceso_operacion'] LOOP
 EXECUTE format('ALTER TABLE vec_meritos.%I ENABLE ROW LEVEL SECURITY',t);
 EXECUTE format('ALTER TABLE vec_meritos.%I FORCE ROW LEVEL SECURITY',t);
 EXECUTE format('CREATE POLICY lectura_nominal ON vec_meritos.%I FOR SELECT TO vec_meritos_propietario USING (persona_ref=current_setting(''vec_meritos.persona_ref'',true))',t);
 EXECUTE format('CREATE POLICY alta_nominal ON vec_meritos.%I FOR INSERT TO vec_meritos_propietario WITH CHECK (persona_ref=current_setting(''vec_meritos.persona_ref'',true))',t);
 EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_meritos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_meritos.historia_inmutable_v1()',t);
 EXECUTE format('REVOKE ALL ON TABLE vec_meritos.%I FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_externo,vec_meritos_interno,vec_meritos_migrador',t);
 END LOOP;
END $seguridad$;

-- Validación estructural del hecho de comando: declarado, sin revisión.
-- Las evidencias son referencias a versiones, nunca prueba de acreditación.
CREATE FUNCTION vec_meritos.validar_hecho_comando_v1(h jsonb) RETURNS void
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE p jsonb; v jsonb; e jsonb; anterior_id text; anterior_version integer;
BEGIN
 IF NOT coalesce(vec_meritos.claves_exactas_v1(h,ARRAY['referencia','persona_ref','version','tipo','concepto_ref','denominacion','procedencia','vigencia','estado','evidencias'],ARRAY['horas','revision']),false)
 OR EXISTS (SELECT 1 FROM jsonb_each(h) kv WHERE kv.key=ANY(ARRAY['referencia','persona_ref','tipo','concepto_ref','denominacion','estado']) AND jsonb_typeof(kv.value)<>'string')
 OR h->>'estado' IS DISTINCT FROM 'declarado' OR coalesce(h->'revision','null'::jsonb)<>'null'::jsonb
 OR NOT coalesce(vec_meritos.referencia_valida_v1(h->>'referencia') AND vec_meritos.referencia_valida_v1(h->>'persona_ref') AND vec_meritos.referencia_valida_v1(h->>'concepto_ref'),false)
 OR jsonb_typeof(h->'version') IS DISTINCT FROM 'number' OR h->>'version' !~ '^[1-9][0-9]*$' OR (h->>'version')::numeric>1073741824
 OR jsonb_typeof(h->'tipo') IS DISTINCT FROM 'string' OR h->>'tipo' NOT IN ('titulacion','curso_asistencia','curso_superacion','experiencia','idioma','otro')
 OR jsonb_typeof(h->'denominacion') IS DISTINCT FROM 'string' OR octet_length(h->>'denominacion') NOT BETWEEN 1 AND 512
 OR h->>'denominacion'<>btrim(h->>'denominacion') OR h->>'denominacion' ~ '[[:cntrl:]]'
 OR jsonb_typeof(h->'evidencias') IS DISTINCT FROM 'array' OR jsonb_array_length(h->'evidencias')>32
 THEN RAISE EXCEPTION 'meritos.error.hecho_invalido' USING ERRCODE='22023'; END IF;
 IF h ? 'horas' AND (jsonb_typeof(h->'horas') IS DISTINCT FROM 'number' OR h->>'horas' !~ '^(0|[1-9][0-9]*)$' OR (h->>'horas')::numeric>2147483647 OR jsonb_typeof(h->'tipo') IS DISTINCT FROM 'string' OR h->>'tipo' NOT IN ('curso_asistencia','curso_superacion'))
 THEN RAISE EXCEPTION 'meritos.error.horas_invalidas' USING ERRCODE='22023'; END IF;
 p:=h->'procedencia'; v:=h->'vigencia';
 IF NOT coalesce(vec_meritos.claves_exactas_v1(p,ARRAY['fuente_ref','version','hecho_origen_ref','capturada_en']),false)
 OR EXISTS (SELECT 1 FROM jsonb_each(p) kv WHERE jsonb_typeof(kv.value)<>'string')
 OR EXISTS (SELECT 1 FROM jsonb_each(v) kv WHERE jsonb_typeof(kv.value)<>'string')
 OR NOT coalesce(vec_meritos.referencia_valida_v1(p->>'fuente_ref') AND vec_meritos.referencia_valida_v1(p->>'version') AND vec_meritos.referencia_valida_v1(p->>'hecho_origen_ref'),false)
 OR coalesce(p->>'capturada_en','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T(0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{1,9})?(Z|[+-](0[0-9]|1[0-9]|2[0-3]):[0-5][0-9])$'
 OR NOT coalesce(vec_meritos.claves_exactas_v1(v,ARRAY['desde'],ARRAY['hasta']),false)
 OR coalesce(v->>'desde','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR (v ? 'hasta' AND (coalesce(v->>'hasta','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR v->>'hasta'<v->>'desde'))
 THEN RAISE EXCEPTION 'meritos.error.procedencia_vigencia_invalida' USING ERRCODE='22023'; END IF;
 PERFORM (p->>'capturada_en')::timestamptz, (v->>'desde')::date;
 IF v ? 'hasta' THEN PERFORM (v->>'hasta')::date; END IF;
 FOR e IN SELECT value FROM jsonb_array_elements(h->'evidencias') LOOP
 IF NOT coalesce(vec_meritos.claves_exactas_v1(e,ARRAY['id','version']),false)
 OR jsonb_typeof(e->'id') IS DISTINCT FROM 'string'
 OR NOT coalesce(vec_meritos.referencia_valida_v1(e->>'id'),false)
 OR jsonb_typeof(e->'version') IS DISTINCT FROM 'number' OR coalesce(e->>'version','') !~ '^[1-9][0-9]*$' OR (e->>'version')::numeric>2147483647
 THEN RAISE EXCEPTION 'meritos.error.evidencia_invalida' USING ERRCODE='22023'; END IF;
 -- Orden por ID y versión numérica como el serializador Go.
 IF anterior_id IS NOT NULL AND ((e->>'id') COLLATE "C"<anterior_id COLLATE "C" OR ((e->>'id')=anterior_id AND (e->>'version')::integer<=anterior_version)) THEN RAISE EXCEPTION 'meritos.error.evidencia_duplicada_desordenada' USING ERRCODE='22023'; END IF;
 anterior_id:=e->>'id'; anterior_version:=(e->>'version')::integer;
 END LOOP;
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'meritos.error.hecho_invalido' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_meritos.validar_comando_v1(p_comando bytea) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE o jsonb; h jsonb; m jsonb;
BEGIN
 IF p_comando IS NULL OR octet_length(p_comando) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'meritos.error.comando_invalido' USING ERRCODE='22023'; END IF;
 o:=convert_from(p_comando,'UTF8')::jsonb; h:=o->'hecho'; m:=o->'motivo';
 IF NOT coalesce(vec_meritos.claves_exactas_v1(o,ARRAY['esquema','accion','actor_ref','clave_idempotencia','version_esperada','hecho','motivo','fecha_corte']),false)
 OR EXISTS (SELECT 1 FROM jsonb_each(o) kv WHERE kv.key=ANY(ARRAY['esquema','accion','actor_ref','clave_idempotencia','fecha_corte']) AND jsonb_typeof(kv.value)<>'string')
 OR o->>'esquema' IS DISTINCT FROM 'vec.meritos.hecho.operacion.v1'
 OR o->>'accion' NOT IN ('meritos.hecho.declarar','meritos.hecho.rectificar','meritos.hecho.rechazar')
 OR NOT coalesce(vec_meritos.referencia_valida_v1(o->>'actor_ref') AND vec_meritos.referencia_valida_v1(o->>'clave_idempotencia'),false)
 OR jsonb_typeof(o->'version_esperada') IS DISTINCT FROM 'number' OR coalesce(o->>'version_esperada','') !~ '^(0|[1-9][0-9]*)$' OR (o->>'version_esperada')::numeric>=1073741824
 OR coalesce(o->>'fecha_corte','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR NOT coalesce(vec_meritos.claves_exactas_v1(m,ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']),false)
 OR EXISTS (SELECT 1 FROM jsonb_each(m) kv WHERE kv.key<>'catalogo_version' AND jsonb_typeof(kv.value)<>'string')
 OR NOT coalesce(vec_meritos.referencia_valida_v1(m->>'catalogo_id') AND vec_meritos.referencia_valida_v1(m->>'entrada_clave'),false)
 OR jsonb_typeof(m->'catalogo_version') IS DISTINCT FROM 'number' OR coalesce(m->>'catalogo_version','') !~ '^[1-9][0-9]*$' OR (m->>'catalogo_version')::numeric>2147483647
 OR coalesce(m->>'catalogo_huella_sha256','') !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'meritos.error.comando_invalido' USING ERRCODE='22023'; END IF;
 PERFORM (o->>'fecha_corte')::date;
 PERFORM vec_meritos.validar_hecho_comando_v1(h);
 IF (h->>'version')::integer<>(o->>'version_esperada')::integer+1
 THEN RAISE EXCEPTION 'meritos.error.version_invalida' USING ERRCODE='22023'; END IF;
 RETURN o;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'meritos.error.comando_invalido' USING ERRCODE='22023';
END $f$;

-- Cotejo local de negocio. La autoridad criptográfica y revocación las aplica
-- exclusivamente la fachada V3 común antes de leer o añadir filas propias.
CREATE FUNCTION vec_meritos.cotejar_autorizacion_v1(o jsonb,p_comando bytea,p_decision bytea,p_motivo bytea) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE d jsonb; m jsonb; contexto text; finalidad text; interna boolean; externa boolean;
BEGIN
 interna:=pg_has_role(session_user,'vec_meritos_interno','MEMBER'); externa:=pg_has_role(session_user,'vec_meritos_externo','MEMBER');
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user OR interna=externa
 OR NOT pg_has_role(session_user,'vec_meritos_ejecutor','MEMBER')
 OR (SELECT count(*) FROM pg_auth_members a WHERE a.member=(SELECT oid FROM pg_roles WHERE rolname=session_user))<>2
 OR EXISTS (SELECT 1 FROM pg_auth_members a WHERE a.member=(SELECT oid FROM pg_roles WHERE rolname=session_user) AND (a.admin_option OR NOT a.inherit_option OR a.set_option))
 OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls OR NOT r.rolcanlogin))
 OR pg_has_role(session_user,'vec_meritos_propietario','MEMBER') OR pg_has_role(session_user,'vec_meritos_migrador','MEMBER')
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'meritos.error.identidad_transaccion_denegada' USING ERRCODE='42501'; END IF;
 d:=convert_from(p_decision,'UTF8')::jsonb; m:=(convert_from(p_motivo,'UTF8')::jsonb)->'referencia';
 finalidad:=CASE o->>'accion' WHEN 'meritos.hecho.declarar' THEN 'declaracion_hecho_propio'
 WHEN 'meritos.hecho.rectificar' THEN 'rectificacion_hecho_propio' ELSE 'revision_hecho_merito' END;
 contexto:='{"ambitos":{"huella_comando_sha256":"'||encode(sha256(p_comando),'hex')||'","persona_ref":"'||(o->'hecho'->>'persona_ref')||'","version_esperada":"'||(o->>'version_esperada')||'"},"atributos":{}}';
 IF d->>'accion' IS DISTINCT FROM o->>'accion' OR d->>'principal_id' IS DISTINCT FROM o->>'actor_ref'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'recurso_ref' IS DISTINCT FROM o->'hecho'->>'referencia' OR d->>'finalidad' IS DISTINCT FROM finalidad
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(contexto,'UTF8')),'hex')
 OR d->'campos_permitidos' IS DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR m->>'catalogo_id' IS DISTINCT FROM o->'motivo'->>'catalogo_id'
 OR m->>'catalogo_version' IS DISTINCT FROM o->'motivo'->>'catalogo_version'
 OR m->>'catalogo_huella_sha256' IS DISTINCT FROM o->'motivo'->>'catalogo_huella_sha256'
 OR m->>'entrada_clave' IS DISTINCT FROM o->'motivo'->>'entrada_clave'
 THEN RAISE EXCEPTION 'meritos.error.autorizacion_divergente' USING ERRCODE='42501'; END IF;
 RETURN d;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'meritos.error.autorizacion_invalida' USING ERRCODE='42501';
END $f$;

-- La búsqueda de idempotencia se restringe al actor autorizado, incluso si
-- la reutilización intenta cambiar la persona del comando. No devuelve datos ajenos.
DROP POLICY lectura_nominal ON vec_meritos.operacion;
CREATE POLICY lectura_nominal ON vec_meritos.operacion FOR SELECT TO vec_meritos_propietario
 USING (actor_ref=current_setting('vec_meritos.actor_ref',true));

CREATE FUNCTION vec_meritos.registrar_auditoria_v1(o jsonb,d jsonb,p_auditoria_ref text,p_fecha timestamptz,p_resultado text) RETURNS void
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 INSERT INTO vec_meritos.auditoria_operacion(auditoria_ref,clave,persona_ref,entrada,registrada_en)
 VALUES(p_auditoria_ref,o->>'clave_idempotencia',o->'hecho'->>'persona_ref',
 jsonb_build_object('actor_id',d->>'principal_id','actor_profile',d->>'perfil_activo_ref',
 'action',d->>'accion','module_id','meritos','purpose',d->>'finalidad',
 'subject_ref',d->>'recurso_ref','result',p_resultado,'correlation_ref',d->>'correlacion_ref',
 'object_version',(o->'hecho'->>'version')::integer,'rule_ref',o->'motivo'->>'entrada_clave',
 'authorization_ref',d->>'decision_ref','occurred_at',to_char(p_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),p_fecha);
END $f$;

CREATE FUNCTION vec_meritos.operar_hecho_v1(
 p_comando bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' SET idle_in_transaction_session_timeout='20s'
AS $f$
#variable_conflict use_variable
DECLARE
 o jsonb; d jsonb; h jsonb; actual jsonb; anterior jsonb; nuevo jsonb; revision jsonb; recibo jsonb;
 u record; op vec_meritos.operacion%ROWTYPE; identidad vec_meritos.hecho_identidad%ROWTYPE;
 codigo text; huella text; clave text; hecho_ref text; persona_ref text; accion text; actor text;
 esperada integer; fecha timestamptz(6); instante text; recibo_ref text; evento_ref text;
BEGIN
 o:=vec_meritos.validar_comando_v1(p_comando);
 d:=vec_meritos.cotejar_autorizacion_v1(o,p_comando,p_decision,p_motivo);
 h:=o->'hecho'; huella:=encode(sha256(p_comando),'hex'); clave:=o->>'clave_idempotencia';
 hecho_ref:=h->>'referencia'; persona_ref:=h->>'persona_ref'; accion:=o->>'accion'; actor:=o->>'actor_ref';
 esperada:=(o->>'version_esperada')::integer;
 -- No se toca ningún hecho antes de la revalidación y consumo del núcleo real.
 SELECT * INTO STRICT u FROM vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF u.consumo_nuevo IS NOT TRUE OR u.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR u.efecto_ref IS DISTINCT FROM hecho_ref OR u.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 OR NOT coalesce(vec_meritos.referencia_valida_v1(u.auditoria_ref),false)
 THEN RAISE EXCEPTION 'meritos.error.consumo_divergente' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec_meritos.persona_ref',persona_ref,true);
 PERFORM set_config('vec_meritos.actor_ref',actor,true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:clave:'||actor||':'||clave,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:hecho:'||hecho_ref,0));
 -- Evita otra alta con distinta referencia del mismo origen, también concurrente.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:origen:'||persona_ref||'|'||(h->'procedencia'->>'fuente_ref')||'|'||(h->'procedencia'->>'hecho_origen_ref')||'|'||(h->>'tipo'),0));
 fecha:=date_trunc('microseconds',clock_timestamp());
 instante:=to_char(fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 -- La separación de funciones limita también la recuperación.
 IF (accion='meritos.hecho.rechazar' AND (NOT pg_has_role(session_user,'vec_meritos_interno','MEMBER') OR actor=persona_ref))
 OR (accion<>'meritos.hecho.rechazar' AND actor<>persona_ref) THEN
 PERFORM vec_meritos.registrar_auditoria_v1(o,d,u.auditoria_ref,fecha,'denegada');
 RETURN jsonb_build_object('codigo','denegada','auditoria_ref',u.auditoria_ref,'anterior',NULL,'recibo',NULL);
 END IF;
 SELECT * INTO op FROM vec_meritos.operacion x WHERE x.clave=clave AND x.actor_ref=actor;
 -- La clave se coteja antes del CAS y siempre con actor, acción y bytes exactos.
 IF FOUND THEN
 IF op.huella_comando IS DISTINCT FROM huella OR op.comando IS DISTINCT FROM p_comando
 OR op.actor_ref IS DISTINCT FROM actor OR op.accion IS DISTINCT FROM accion OR op.persona_ref IS DISTINCT FROM persona_ref
 THEN codigo:='clave_reutilizada';
 ELSE
 IF esperada>0 THEN SELECT v.registro INTO STRICT anterior FROM vec_meritos.hecho_version v WHERE v.hecho_ref=hecho_ref AND v.version=esperada; END IF;
 PERFORM vec_meritos.registrar_auditoria_v1(o,d,u.auditoria_ref,fecha,'recuperada');
 INSERT INTO vec_meritos.acceso_operacion(acceso_ref,clave,persona_ref,actor_ref,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES('acceso:'||gen_random_uuid()::text,clave,persona_ref,actor,u.decision_ref,u.auditoria_ref,u.consumo_huella_sha256,fecha);
 RETURN jsonb_build_object('codigo','confirmada','auditoria_ref',u.auditoria_ref,'anterior',anterior,'recibo',op.recibo);
 END IF;
 ELSE
 SELECT * INTO identidad FROM vec_meritos.hecho_identidad x WHERE x.hecho_ref=hecho_ref;
 IF FOUND THEN
 SELECT v.registro INTO STRICT actual FROM vec_meritos.hecho_version v WHERE v.hecho_ref=hecho_ref ORDER BY v.version DESC LIMIT 1;
 anterior:=actual;
 END IF;
 IF accion='meritos.hecho.declarar' THEN
 IF esperada<>0 OR actual IS NOT NULL OR EXISTS (SELECT 1 FROM vec_meritos.hecho_identidad x
 WHERE x.persona_ref=persona_ref AND x.fuente_ref=h->'procedencia'->>'fuente_ref' AND x.origen_ref=h->'procedencia'->>'hecho_origen_ref' AND x.tipo=h->>'tipo')
 THEN codigo:='conflicto_version';
 ELSE nuevo:=jsonb_build_object('hecho',h,'declarante_ref',actor); END IF;
 ELSE
 IF actual IS NULL OR (actual->'hecho'->>'version')::integer<>esperada
 OR identidad.persona_ref IS DISTINCT FROM persona_ref OR identidad.fuente_ref IS DISTINCT FROM h->'procedencia'->>'fuente_ref'
 OR identidad.origen_ref IS DISTINCT FROM h->'procedencia'->>'hecho_origen_ref' OR identidad.tipo IS DISTINCT FROM h->>'tipo'
 THEN codigo:='conflicto_version';
 ELSIF accion='meritos.hecho.rectificar' THEN
 IF actor IS DISTINCT FROM identidad.declarante_ref THEN codigo:='denegada';
 ELSE nuevo:=jsonb_build_object('hecho',jsonb_set(h,'{estado}','"pendiente"'::jsonb)-'revision','declarante_ref',identidad.declarante_ref); END IF;
 ELSE
 IF actor=identidad.declarante_ref THEN codigo:='denegada';
 ELSIF actual->'hecho'->>'estado' NOT IN ('declarado','pendiente')
 OR ((actual->'hecho')-'estado'-'revision'-'version') IS DISTINCT FROM (h-'estado'-'revision'-'version')
 THEN codigo:='conflicto_version';
 ELSE
 revision:=jsonb_build_object('referencia','revision:'||huella,'actor_ref',actor,
 'motivo_ref',(o->'motivo'->>'catalogo_id')||':'||(o->'motivo'->>'catalogo_version')||':'||(o->'motivo'->>'entrada_clave'),'fecha',instante);
 nuevo:=jsonb_build_object('hecho',jsonb_set(jsonb_set(h,'{estado}','"rechazado"'::jsonb),'{revision}',revision),'declarante_ref',identidad.declarante_ref);
 END IF;
 END IF;
 END IF;
 END IF;
 IF codigo IS NOT NULL THEN
 -- Negación de negocio después de V3: se confirma la auditoría, no una versión.
 PERFORM vec_meritos.registrar_auditoria_v1(o,d,u.auditoria_ref,fecha,codigo);
 RETURN jsonb_build_object('codigo',codigo,'auditoria_ref',u.auditoria_ref,'anterior',NULL,'recibo',NULL);
 END IF;
 IF nuevo IS NULL THEN RAISE EXCEPTION 'meritos.error.transicion_incompleta' USING ERRCODE='55000'; END IF;
 IF accion='meritos.hecho.declarar' THEN
 BEGIN
 INSERT INTO vec_meritos.hecho_identidad(hecho_ref,persona_ref,fuente_ref,origen_ref,tipo,declarante_ref)
 VALUES(hecho_ref,persona_ref,h->'procedencia'->>'fuente_ref',h->'procedencia'->>'hecho_origen_ref',h->>'tipo',actor);
 EXCEPTION WHEN unique_violation THEN
 -- Una referencia global ya ocupada por otra persona tampoco se proyecta.
 PERFORM vec_meritos.registrar_auditoria_v1(o,d,u.auditoria_ref,fecha,'conflicto_version');
 RETURN jsonb_build_object('codigo','conflicto_version','auditoria_ref',u.auditoria_ref,'anterior',NULL,'recibo',NULL);
 END;
 END IF;
 recibo_ref:='recibo:'||gen_random_uuid()::text; evento_ref:='evento:'||gen_random_uuid()::text;
 recibo:=jsonb_build_object('referencia',recibo_ref,'accion',accion,'actor_ref',actor,'clave_idempotencia',clave,
 'huella_comando',huella,'version_esperada',esperada,'registro',nuevo,'registrado_en',instante,'auditoria_ref',u.auditoria_ref,'evento_ref',evento_ref);
 INSERT INTO vec_meritos.hecho_version(hecho_ref,persona_ref,version,registro,registrada_en)
 VALUES(hecho_ref,persona_ref,esperada+1,nuevo,fecha);
 INSERT INTO vec_meritos.operacion(clave,huella_comando,comando,hecho_ref,persona_ref,version,actor_ref,accion,decision_ref,consumo_huella_sha256,recibo_ref,recibo,registrada_en)
 VALUES(clave,huella,p_comando,hecho_ref,persona_ref,esperada+1,actor,accion,u.decision_ref,u.consumo_huella_sha256,recibo_ref,recibo,fecha);
 PERFORM vec_meritos.registrar_auditoria_v1(o,d,u.auditoria_ref,fecha,'confirmada');
 INSERT INTO vec_meritos.outbox(evento_ref,clave,actor_ref,persona_ref,tipo,carga,creada_en)
 VALUES(evento_ref,clave,actor,persona_ref,'meritos.hecho.registrado.v1',jsonb_build_object('hecho_ref',hecho_ref,'version',esperada+1,'recibo_ref',recibo_ref,'accion',accion),fecha);
 RETURN jsonb_build_object('codigo','confirmada','auditoria_ref',u.auditoria_ref,'anterior',anterior,'recibo',recibo);
END $f$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_meritos FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_externo,vec_meritos_interno,vec_meritos_migrador;
GRANT EXECUTE ON FUNCTION vec_meritos.operar_hecho_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_meritos_ejecutor;
-- Los tipos de fila nuevos tampoco quedan publicados.
DO $tipos$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['hecho_identidad','hecho_version','operacion','auditoria_operacion','outbox','acceso_operacion'] LOOP
 EXECUTE format('REVOKE ALL ON TYPE vec_meritos.%I FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_externo,vec_meritos_interno,vec_meritos_migrador',t);
 END LOOP;
END $tipos$;
COMMIT;
