\set ON_ERROR_STOP on
-- CC7: gobierno del plan nominal de firma CT por CatalogoConfigurable.
-- Depende de CC1 y del consumo AD177 en la misma transacción. Sin acceso LOGIN.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SET LOCAL idle_in_transaction_session_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000007',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
    OR pg_catalog.to_regclass('vec_catalogos_configurables.publicacion') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class p
      WHERE p.oid='vec_catalogos_configurables.publicacion'::pg_catalog.regclass
        AND p.relowner=current_user::pg_catalog.regrole AND p.relrowsecurity AND p.relforcerowsecurity)
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.rechazar_cambio_inmutable()') IS NULL
    OR pg_catalog.to_regclass('vec_catalogos_configurables.plan_firma_control') IS NOT NULL
    OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_catalogos_configurables_propietario',
      'vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_catalogos_configurables_propietario',
      'vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb)','EXECUTE')
    OR NOT pg_catalog.has_schema_privilege('vec_catalogos_configurables_propietario',
      'vec_autorizacion_atestada_v3','USAGE')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
      WHERE n.nspname='vec_catalogos_configurables' AND n.nspowner=current_user::pg_catalog.regrole)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname IN
      ('vec_catalogos_configurables_propietario','vec_autorizacion_atestada_v3_propietario','vec_contratacion_temporal_propietario')
      AND (r.rolcanlogin OR r.rolsuper OR r.rolbypassrls)) THEN
  RAISE EXCEPTION 'CC7: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La cabeza mutable conserva revisión/estado actuales. El SHA de publicación
-- apunta a bytes inmutables y no cambia al retirarse la versión.
CREATE TABLE vec_catalogos_configurables.plan_firma_control (
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 modulo_id text NOT NULL DEFAULT 'contratacion_temporal' CHECK(modulo_id='contratacion_temporal'),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 2147483647),
 estado text NOT NULL CHECK(estado IN ('borrador','publicado','retirado')),
 canonico_actual bytea NOT NULL CHECK(pg_catalog.octet_length(canonico_actual) BETWEEN 2 AND 2097152),
 huella_actual text NOT NULL CHECK(huella_actual ~ '^[0-9a-f]{64}$'),
 publicacion_sha256 text CHECK(publicacion_sha256 ~ '^[0-9a-f]{64}$'),
 publicacion_revision bigint CHECK(publicacion_revision BETWEEN 1 AND 2147483647),
 creado_por text NOT NULL CHECK(pg_catalog.octet_length(creado_por) BETWEEN 3 AND 512),
 ultimo_editor text NOT NULL CHECK(pg_catalog.octet_length(ultimo_editor) BETWEEN 3 AND 512),
 publicado_por text,
 retirado_por text,
 PRIMARY KEY(catalogo_id,version),
 CHECK(pg_catalog.encode(pg_catalog.sha256(canonico_actual),'hex')=huella_actual),
 CHECK((estado='borrador' AND publicacion_sha256 IS NULL AND publicacion_revision IS NULL AND publicado_por IS NULL AND retirado_por IS NULL)
    OR (estado='publicado' AND publicacion_sha256 IS NOT NULL AND publicacion_revision IS NOT NULL AND publicado_por IS NOT NULL AND retirado_por IS NULL)
    OR (estado='retirado' AND publicacion_sha256 IS NOT NULL AND publicacion_revision IS NOT NULL AND publicado_por IS NOT NULL AND retirado_por IS NOT NULL))
);
CREATE TABLE vec_catalogos_configurables.plan_firma_historia (
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 2147483647),
 operacion text NOT NULL CHECK(operacion IN ('crear','actualizar','publicar','retirar')),
 estado text NOT NULL CHECK(estado IN ('borrador','publicado','retirado')),
 canonico_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(canonico_exacto) BETWEEN 2 AND 2097152),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK(pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 512),
 decision_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 512),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:[0-9a-f-]{36}$'),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(catalogo_id,version,revision,estado),
 UNIQUE(catalogo_id,version,revision,estado,huella_sha256),
 FOREIGN KEY(catalogo_id,version) REFERENCES vec_catalogos_configurables.plan_firma_control(catalogo_id,version),
 CHECK(pg_catalog.encode(pg_catalog.sha256(canonico_exacto),'hex')=huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.plan_firma_publicacion (
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$'),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 2147483647),
 estado text NOT NULL DEFAULT 'publicado' CHECK(estado='publicado'),
 canonico_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(canonico_exacto) BETWEEN 2 AND 2097152),
 publicacion_sha256 text NOT NULL CHECK(publicacion_sha256 ~ '^[0-9a-f]{64}$'),
 publicado_por text NOT NULL CHECK(pg_catalog.octet_length(publicado_por) BETWEEN 3 AND 512),
 publicada_en timestamptz(6) NOT NULL,
 aprobacion_ref text NOT NULL CHECK(pg_catalog.octet_length(aprobacion_ref) BETWEEN 3 AND 512),
 PRIMARY KEY(catalogo_id,version),
 FOREIGN KEY(catalogo_id,version,revision,estado,publicacion_sha256)
  REFERENCES vec_catalogos_configurables.plan_firma_historia(catalogo_id,version,revision,estado,huella_sha256),
 CHECK(pg_catalog.encode(pg_catalog.sha256(canonico_exacto),'hex')=publicacion_sha256)
);
CREATE TABLE vec_catalogos_configurables.plan_firma_efecto (
 actor_ref text NOT NULL CHECK(pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 512),
 catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$'),
 clave_operacion text NOT NULL CHECK(pg_catalog.octet_length(clave_operacion) BETWEEN 16 AND 128),
 operacion text NOT NULL CHECK(operacion IN ('crear','actualizar','publicar','retirar')),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 modulo_id text NOT NULL DEFAULT 'contratacion_temporal' CHECK(modulo_id='contratacion_temporal'),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 2147483647),
 estado text NOT NULL CHECK(estado IN ('borrador','publicado','retirado')),
 huella_resultado text NOT NULL CHECK(huella_resultado ~ '^[0-9a-f]{64}$'),
 publicacion_sha256_resultado text CHECK(publicacion_sha256_resultado ~ '^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:[0-9a-f-]{36}$'),
 decision_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 512),
 auditoria_ref text NOT NULL CHECK(pg_catalog.octet_length(auditoria_ref) BETWEEN 3 AND 512),
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 traza_sha256 text NOT NULL CHECK(traza_sha256 ~ '^[0-9a-f]{64}$'),
 evento_sha256 text NOT NULL CHECK(evento_sha256 ~ '^[0-9a-f]{64}$'),
 confirmado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(actor_ref,catalogo_id,clave_operacion),
 FOREIGN KEY(catalogo_id,version,revision,estado,huella_resultado)
  REFERENCES vec_catalogos_configurables.plan_firma_historia(catalogo_id,version,revision,estado,huella_sha256),
 CHECK((estado='borrador' AND publicacion_sha256_resultado IS NULL)
    OR (estado IN ('publicado','retirado') AND publicacion_sha256_resultado IS NOT NULL))
);
CREATE TABLE vec_catalogos_configurables.plan_firma_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.plan_firma_efecto(recibo_ref),
 evento_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(evento_exacto) BETWEEN 2 AND 65536),
 evento_sha256 text NOT NULL CHECK(evento_sha256 ~ '^[0-9a-f]{64}$'),
 creado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK(pg_catalog.encode(pg_catalog.sha256(evento_exacto),'hex')=evento_sha256)
);
DO $acl$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_firma_control','plan_firma_historia','plan_firma_publicacion','plan_firma_efecto','plan_firma_outbox'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY %I ON vec_catalogos_configurables.%I TO vec_catalogos_configurables_propietario USING(true) WITH CHECK(true)',t||'_propietario',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM PUBLIC',t);
  IF t='plan_firma_control' THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',t||'_no_borrar',t);
  ELSE
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',t||'_inmutable',t);
  END IF;
 END LOOP;
END $acl$;

-- JSONB colapsa claves repetidas. Se comprueba la representación JSON original
-- en cada objeto antes de usar JSONB para las reglas de negocio.
CREATE FUNCTION vec_catalogos_configurables.json_plan_sin_duplicados_v1(p_json json,p_profundidad integer)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' AS $f$
DECLARE elemento json; tipo text;
BEGIN
 IF p_json IS NULL OR p_profundidad IS NULL OR p_profundidad NOT BETWEEN 0 AND 32 THEN RETURN false; END IF;
 tipo:=pg_catalog.json_typeof(p_json);
 IF tipo='object' THEN
  IF (SELECT count(*) FROM pg_catalog.json_object_keys(p_json))<>
     (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_json::jsonb)) THEN RETURN false; END IF;
  FOR elemento IN SELECT value FROM pg_catalog.json_each(p_json) LOOP
   IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(elemento,p_profundidad+1) THEN RETURN false; END IF;
  END LOOP;
 ELSIF tipo='array' THEN
  FOR elemento IN SELECT value FROM pg_catalog.json_array_elements(p_json) LOOP
   IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(elemento,p_profundidad+1) THEN RETURN false; END IF;
  END LOOP;
 ELSIF tipo NOT IN ('string','number','boolean','null') THEN RETURN false;
 END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer) FROM PUBLIC;

-- Valida el subconjunto finito que DesdeCatalogo interpreta. El SHA identifica
-- los bytes de CatalogoConfigurable; jsonb sirve sólo para cotejar estructura.
CREATE FUNCTION vec_catalogos_configurables.validar_plan_nominal_firma_v1(p_canon bytea,p_sha text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' AS $f$
DECLARE c jsonb; e jsonb; a jsonb;
 go_cero constant text:='0001-01-01T00:00:00Z';
 claves text[]:=ARRAY['esquema','circuito_ref','circuito_version','circuito_sha256',
  'documento','paso_ref','paso_orden','perfil_esperado_ref','rol_id','cargo_ref',
  'organizacion_ref','unidad_ref','accion_competencial','finalidad','tipo_recurso',
  'esquema_contexto','mapeo_version','mapeo_fuente_ref'];
 k text;
BEGIN
 IF p_canon IS NULL OR pg_catalog.octet_length(p_canon) NOT BETWEEN 2 AND 2097152
    OR p_sha IS NULL OR p_sha !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_canon),'hex') IS DISTINCT FROM p_sha THEN
  RAISE EXCEPTION 'CC7: canon o huella inválidos' USING ERRCODE='22023'; END IF;
 BEGIN
  IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(p_canon,'UTF8')::json,0) THEN
   RAISE EXCEPTION 'CC7: canon ambiguo' USING ERRCODE='22023'; END IF;
  c:=pg_catalog.convert_from(p_canon,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CC7: canon no es JSON UTF8' USING ERRCODE='22023'; END;
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR (c->>'id' ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR c->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR (c->>'estado' IN ('borrador','publicado','retirado')) IS NOT TRUE
    OR pg_catalog.jsonb_typeof(c->'version') IS DISTINCT FROM 'number'
    OR pg_catalog.jsonb_typeof(c->'revision') IS DISTINCT FROM 'number'
    OR (c->>'version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR (c->>'revision' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR pg_catalog.jsonb_typeof(c->'entradas') IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_array_length(c->'entradas')>64
    OR (c->>'estado'<>'borrador' AND pg_catalog.jsonb_array_length(c->'entradas')=0)
    OR NOT (c ?& ARRAY['id','version','revision','modulo_id','nombre','fuente_ref',
      'motivo_creacion','entradas','estado','creado_por','creado_en',
      'ultima_modificacion_en','publicado_en','retirado_en'])
    OR pg_catalog.octet_length(coalesce(c->>'nombre','')) NOT BETWEEN 1 AND 2048
    OR pg_catalog.octet_length(coalesce(c->>'motivo_creacion','')) NOT BETWEEN 1 AND 4096
    OR pg_catalog.octet_length(coalesce(c->>'creado_por','')) NOT BETWEEN 3 AND 512
    OR pg_catalog.jsonb_typeof(c->'creado_en') IS DISTINCT FROM 'string'
    OR c->>'creado_en' IS NULL
    OR pg_catalog.octet_length(coalesce(c->>'fuente_ref','')) NOT BETWEEN 3 AND 512
    OR c->>'fuente_ref'='paquete:ejemplo:vec:v1' THEN
  RAISE EXCEPTION 'CC7: catálogo de firma incompatible' USING ERRCODE='22023'; END IF;
 IF (c->>'version')::numeric>2147483647 OR (c->>'revision')::numeric>2147483647 THEN
  RAISE EXCEPTION 'CC7: versión o revisión fuera de rango' USING ERRCODE='22023'; END IF;
 IF ((c->>'version')::bigint=1 AND c ? 'version_anterior_ref')
    OR ((c->>'version')::bigint>1 AND c->>'version_anterior_ref' IS DISTINCT FROM
      (c->>'id')||':'||((c->>'version')::bigint-1)::text)
    OR ((c->>'revision')::bigint=1 AND
      (c ? 'ultima_modificacion_por' OR c ? 'motivo_modificacion' OR c->>'ultima_modificacion_en' IS DISTINCT FROM go_cero))
    OR ((c->>'revision')::bigint>1 AND
      (NOT (c ?& ARRAY['ultima_modificacion_por','motivo_modificacion'])
       OR c->>'ultima_modificacion_en' IS NOT DISTINCT FROM go_cero))
    OR (c->>'estado'='borrador' AND
      ((c ?| ARRAY['publicado_por','aprobacion_ref','motivo_publicacion',
        'retirado_por','retirada_aprobacion_ref','motivo_retirada'])
       OR c->>'publicado_en' IS DISTINCT FROM go_cero OR c->>'retirado_en' IS DISTINCT FROM go_cero))
    OR (c->>'estado' IN ('publicado','retirado') AND
      (NOT (c ?& ARRAY['publicado_por','aprobacion_ref','motivo_publicacion'])
       OR c->>'publicado_en' IS NOT DISTINCT FROM go_cero))
    OR (c->>'estado'='publicado' AND
      ((c ?| ARRAY['retirado_por','retirada_aprobacion_ref','motivo_retirada'])
       OR c->>'retirado_en' IS DISTINCT FROM go_cero))
    OR (c->>'estado'='retirado' AND
      (NOT (c ?& ARRAY['retirado_por','retirada_aprobacion_ref','motivo_retirada'])
       OR c->>'retirado_en' IS NOT DISTINCT FROM go_cero)) THEN
  RAISE EXCEPTION 'CC7: historia de catálogo incompatible' USING ERRCODE='22023'; END IF;
 IF (c->>'creado_en' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
    OR (c->>'ultima_modificacion_en' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
    OR (c->>'publicado_en' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
    OR (c->>'retirado_en' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
    OR (c->>'ultima_modificacion_en'<>go_cero AND
      (c->>'ultima_modificacion_en')::timestamptz<(c->>'creado_en')::timestamptz)
    OR (c->>'publicado_en'<>go_cero AND
      (c->>'publicado_en')::timestamptz<(c->>'creado_en')::timestamptz)
    OR (c->>'retirado_en'<>go_cero AND
      (c->>'retirado_en')::timestamptz<(c->>'publicado_en')::timestamptz) THEN
  RAISE EXCEPTION 'CC7: instantes de gobierno incompatibles' USING ERRCODE='22023'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_object_keys(c) AS x LOOP
  IF k <> ALL(ARRAY['id','version','revision','version_anterior_ref','modulo_id','nombre','descripcion',
     'fuente_ref','motivo_creacion','entradas','estado','creado_por','creado_en',
     'ultima_modificacion_por','ultima_modificacion_en','motivo_modificacion',
     'publicado_por','publicado_en','aprobacion_ref','motivo_publicacion',
     'retirado_por','retirado_en','retirada_aprobacion_ref','motivo_retirada']) THEN
   RAISE EXCEPTION 'CC7: campo de catálogo ajeno' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR e IN SELECT value FROM pg_catalog.jsonb_array_elements(c->'entradas') LOOP
  a:=e->'atributos';
  IF pg_catalog.jsonb_typeof(e) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(a) IS DISTINCT FROM 'object'
     OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(a))<>18
     OR NOT (a ?& claves)
     OR NOT (e ?& ARRAY['clave','etiqueta','orden','vigente_desde','atributos'])
     OR (e->>'clave' ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
     OR pg_catalog.octet_length(coalesce(e->>'etiqueta','')) NOT BETWEEN 1 AND 2048
     OR pg_catalog.jsonb_typeof(e->'orden') IS DISTINCT FROM 'number'
     OR (e->>'orden' ~ '^(0|[1-9][0-9]{0,8})$') IS NOT TRUE
     OR pg_catalog.jsonb_typeof(e->'vigente_desde') IS DISTINCT FROM 'string'
     OR e->>'vigente_desde' IS NULL
     OR pg_catalog.jsonb_typeof(e->'vigente_hasta') IS DISTINCT FROM 'string'
     OR pg_catalog.jsonb_typeof(a->'esquema') IS DISTINCT FROM 'string'
     OR a->>'esquema' IS DISTINCT FROM 'ct.plan-competencia-firma.v2'
     OR pg_catalog.jsonb_typeof(a->'circuito_sha256') IS DISTINCT FROM 'string'
     OR (a->>'circuito_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
   RAISE EXCEPTION 'CC7: entrada de plan incompatible' USING ERRCODE='22023'; END IF;
  FOR k IN SELECT x FROM pg_catalog.jsonb_object_keys(e) AS x LOOP
   IF k <> ALL(ARRAY['clave','etiqueta','descripcion','orden','vigente_desde','vigente_hasta','atributos']) THEN
    RAISE EXCEPTION 'CC7: atributo de entrada ajeno' USING ERRCODE='22023'; END IF;
  END LOOP;
  IF (e->>'vigente_desde' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
     OR (e->>'vigente_hasta' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$') IS NOT TRUE
     OR (e->>'vigente_hasta'<>go_cero AND
       (e->>'vigente_hasta')::timestamptz<=(e->>'vigente_desde')::timestamptz) THEN
   RAISE EXCEPTION 'CC7: vigencia de entrada inválida' USING ERRCODE='22023'; END IF;
  FOREACH k IN ARRAY ARRAY['circuito_ref','documento','paso_ref','perfil_esperado_ref','rol_id',
    'cargo_ref','organizacion_ref','unidad_ref','accion_competencial','finalidad',
    'tipo_recurso','esquema_contexto','mapeo_fuente_ref'] LOOP
   IF pg_catalog.jsonb_typeof(a->k) IS DISTINCT FROM 'string'
      -- PostgreSQL limita {m,n} a 255: la longitud se comprueba aparte.
      OR (a->>k ~ '^[a-z][A-Za-z0-9_.:-]*$') IS NOT TRUE OR length(a->>k) NOT BETWEEN 3 AND 512 THEN
    RAISE EXCEPTION 'CC7: selector de plan inválido' USING ERRCODE='22023'; END IF;
  END LOOP;
  IF (a->>'circuito_version' ~ '^[1-9][0-9]{0,15}$') IS NOT TRUE
     OR pg_catalog.jsonb_typeof(a->'circuito_version') IS DISTINCT FROM 'string'
     OR (a->>'mapeo_version' ~ '^[1-9][0-9]{0,15}$') IS NOT TRUE
     OR pg_catalog.jsonb_typeof(a->'mapeo_version') IS DISTINCT FROM 'string'
     OR pg_catalog.jsonb_typeof(a->'paso_orden') IS DISTINCT FROM 'string'
     OR (a->>'paso_orden' ~ '^([1-9]|1[0-6])$') IS NOT TRUE THEN
   RAISE EXCEPTION 'CC7: versión o paso inválido' USING ERRCODE='22023'; END IF;
  IF (a->>'circuito_version')::numeric>9007199254740991
     OR (a->>'mapeo_version')::numeric>9007199254740991 THEN
   RAISE EXCEPTION 'CC7: versión de mapeo fuera de rango' USING ERRCODE='22023'; END IF;
  IF (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'entradas') x
      WHERE x.value->>'clave'=e->>'clave')<>1
     OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'entradas') x
      WHERE x.value->'atributos'->>'documento'=a->>'documento'
       AND x.value->'atributos'->>'paso_ref'=a->>'paso_ref'
       AND x.value->'atributos'->>'perfil_esperado_ref'=a->>'perfil_esperado_ref'
       AND x.value->'atributos'->>'organizacion_ref'=a->>'organizacion_ref'
       AND x.value->'atributos'->>'unidad_ref'=a->>'unidad_ref')<>1 THEN
   RAISE EXCEPTION 'CC7: mapeo ambiguo o duplicado' USING ERRCODE='22023'; END IF;
 END LOOP;
 RETURN c;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text) FROM PUBLIC;

-- El pin sigue vigente hasta COMMIT de la transacción CT. La prueba de firma
-- V3 ya consumida procede de AD177 y se comprueba antes de leer el plan.
CREATE FUNCTION vec_catalogos_configurables.leer_plan_nominal_firma_v1(
 p_catalogo_id text,p_version bigint,p_publicacion_sha256 text,p_entrada_clave text,p_consumo_firma jsonb)
RETURNS TABLE(documento_exacto bytea,publicacion_sha256 text,entrada jsonb,revision_control bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='10s' SET TimeZone='UTC' AS $f$
DECLARE control vec_catalogos_configurables.plan_firma_control%ROWTYPE;
 publicacion vec_catalogos_configurables.plan_firma_publicacion%ROWTYPE;
 prueba jsonb; documento jsonb; encontrado jsonb; total integer; ahora timestamptz(6);
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR pg_catalog.pg_is_in_recovery()
    OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR p_version IS NULL OR p_version NOT BETWEEN 1 AND 2147483647
    OR (p_publicacion_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (p_entrada_clave ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CC7: pin de plan inválido' USING ERRCODE='42501'; END IF;
 SELECT vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(p_consumo_firma) INTO STRICT prueba;
 IF pg_catalog.jsonb_typeof(prueba) IS DISTINCT FROM 'object'
    OR prueba->>'decision_ref' IS DISTINCT FROM p_consumo_firma->>'decision_ref'
    OR prueba->>'efecto_ref' IS DISTINCT FROM p_consumo_firma->>'efecto_ref'
    OR prueba->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo_firma->>'consumo_huella_sha256'
    OR prueba->>'auditoria_ref' IS DISTINCT FROM p_consumo_firma->>'auditoria_ref' THEN
  RAISE EXCEPTION 'CC7: consumo de firma no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT control FROM vec_catalogos_configurables.plan_firma_control c
  WHERE c.catalogo_id=p_catalogo_id AND c.version=p_version FOR SHARE;
 IF prueba->>'decision_valida_hasta' IS NULL
    OR (prueba->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC7: consumo de firma caducado tras bloqueo' USING ERRCODE='42501'; END IF;
 IF control.estado IS DISTINCT FROM 'publicado'
    OR control.publicacion_sha256 IS DISTINCT FROM p_publicacion_sha256
    OR control.modulo_id IS DISTINCT FROM 'contratacion_temporal' THEN
  RAISE EXCEPTION 'CC7: plan retirado o no publicado' USING ERRCODE='42501'; END IF;
 -- Sólo vale la publicación vigente: ni una versión anterior aún no retirada
 -- ni otro catálogo de plan del módulo. SERIALIZABLE detecta una publicación
 -- concurrente que cambie esta lectura.
 -- Una versión posterior que llegó a publicarse la sustituye para siempre,
 -- aunque después se retire: la anterior no revive sin publicar otra nueva.
 IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
    WHERE x.modulo_id='contratacion_temporal'
      AND ((x.catalogo_id<>p_catalogo_id AND x.estado='publicado')
        OR (x.catalogo_id=p_catalogo_id AND x.version>p_version AND x.estado IN ('publicado','retirado')))) THEN
  RAISE EXCEPTION 'CC7: plan sustituido por otra publicación' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT publicacion FROM vec_catalogos_configurables.plan_firma_publicacion p
  WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version;
 IF publicacion.publicacion_sha256 IS DISTINCT FROM control.publicacion_sha256
    OR publicacion.revision IS DISTINCT FROM control.publicacion_revision
    OR pg_catalog.encode(pg_catalog.sha256(publicacion.canonico_exacto),'hex') IS DISTINCT FROM p_publicacion_sha256
    OR publicacion.publicada_en>pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC7: publicación original incoherente' USING ERRCODE='55000'; END IF;
 documento:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(publicacion.canonico_exacto,p_publicacion_sha256);
 IF documento->>'id' IS DISTINCT FROM p_catalogo_id
    OR documento->>'version' IS DISTINCT FROM p_version::text
    OR documento->>'estado' IS DISTINCT FROM 'publicado'
    OR (documento->>'publicado_en')::timestamptz IS DISTINCT FROM publicacion.publicada_en
    OR documento->>'publicado_por' IS DISTINCT FROM publicacion.publicado_por
    OR documento->>'aprobacion_ref' IS DISTINCT FROM publicacion.aprobacion_ref THEN
  RAISE EXCEPTION 'CC7: publicación ajena' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO total FROM pg_catalog.jsonb_array_elements(documento->'entradas') x
  WHERE x.value->>'clave'=p_entrada_clave;
 SELECT x.value INTO encontrado FROM pg_catalog.jsonb_array_elements(documento->'entradas') x
  WHERE x.value->>'clave'=p_entrada_clave LIMIT 1;
 ahora:=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
 IF total<>1 OR encontrado->'atributos'->>'esquema' IS DISTINCT FROM 'ct.plan-competencia-firma.v2'
    OR (encontrado->>'vigente_desde')::timestamptz>ahora
    OR (encontrado->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND
      (encontrado->>'vigente_hasta')::timestamptz<=ahora) THEN
  RAISE EXCEPTION 'CC7: entrada nominal ausente' USING ERRCODE='42501'; END IF;
 IF encontrado->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND
    (encontrado->>'vigente_hasta')::timestamptz<=pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC7: entrada vencida durante pin' USING ERRCODE='42501'; END IF;
 IF (prueba->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC7: consumo de firma caducado durante pin' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT publicacion.canonico_exacto,publicacion.publicacion_sha256,encontrado,control.revision;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)
 TO vec_contratacion_temporal_propietario;

-- AD177 consume y audita V3 en esta transacción antes de llamar. Su comprobador
-- relee las filas originales y entrega sólo datos de la decisión verificada.
CREATE FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(
 p_material_exacto bytea,p_consumo jsonb)
RETURNS TABLE(recibo_ref text,estado text,revision bigint,huella_comun text,publicacion_sha256 text,
 actor_ref text,confirmado_en timestamptz,auditoria_ref text,outbox_recibo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='30s' SET TimeZone='UTC' AS $f$
DECLARE m jsonb; c jsonb; traza jsonb; evento jsonb; v3 jsonb;
 canon bytea; traza_bytes bytea; evento_bytes bytea;
 op text; cat text; actor text; clave text; accion_evento text;
 ver bigint; rev bigint; rev_esperada bigint; h text; h_esperada text;
 h_material text; h_contexto text; h_publicacion text; recibo text; existe boolean;
 actor_original text; fecha_original timestamptz(6); auditoria_original text; outbox_original text;
 actual vec_catalogos_configurables.plan_firma_control%ROWTYPE;
 previo vec_catalogos_configurables.plan_firma_efecto%ROWTYPE;
 origen jsonb; momento timestamptz(6);
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR pg_catalog.pg_is_in_recovery()
    OR p_material_exacto IS NULL OR pg_catalog.octet_length(p_material_exacto) NOT BETWEEN 2 AND 4194304
    OR pg_catalog.jsonb_typeof(p_consumo) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_consumo))<>7
    OR NOT (p_consumo ?& ARRAY['consumo','actor_ref','perfil_ref','accion','finalidad','proceso','canal'])
    OR pg_catalog.jsonb_typeof(p_consumo->'consumo') IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'CC7: material o consumo inválido' USING ERRCODE='42501'; END IF;
 BEGIN
  IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(p_material_exacto,'UTF8')::json,0) THEN
   RAISE EXCEPTION 'CC7: material ambiguo' USING ERRCODE='22023'; END IF;
  m:=pg_catalog.convert_from(p_material_exacto,'UTF8')::jsonb;
  canon:=pg_catalog.decode(m->>'catalogo_canonico_base64','base64');
  traza_bytes:=pg_catalog.decode(m->>'traza_canonica_base64','base64');
  evento_bytes:=pg_catalog.decode(m->>'evento_canonico_base64','base64');
  IF NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(traza_bytes,'UTF8')::json,0)
     OR NOT vec_catalogos_configurables.json_plan_sin_duplicados_v1(pg_catalog.convert_from(evento_bytes,'UTF8')::json,0) THEN
   RAISE EXCEPTION 'CC7: evidencia ambigua' USING ERRCODE='22023'; END IF;
  traza:=pg_catalog.convert_from(traza_bytes,'UTF8')::jsonb;
  evento:=pg_catalog.convert_from(evento_bytes,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'CC7: bytes de material inválidos' USING ERRCODE='22023'; END;
 IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(m))<>13
    OR NOT (m ?& ARRAY['esquema','operacion','catalogo_id','version','revision_esperada',
      'huella_esperada','clave_operacion','catalogo_canonico_base64','catalogo_sha256',
      'traza_canonica_base64','traza_sha256','evento_canonico_base64','evento_sha256'])
    OR m->>'esquema' IS DISTINCT FROM 'vec.catalogos.plan-firma.gobierno.v1'
    OR (m->>'operacion' IN ('crear','actualizar','publicar','retirar')) IS NOT TRUE
    OR (m->>'version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
    OR (m->>'revision_esperada' ~ '^(0|[1-9][0-9]{0,9})$') IS NOT TRUE
    OR (m->>'clave_operacion' ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,127}$') IS NOT TRUE
    OR (m->>'catalogo_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (m->>'traza_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (m->>'evento_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR pg_catalog.octet_length(traza_bytes) NOT BETWEEN 2 AND 65536
    OR pg_catalog.octet_length(evento_bytes) NOT BETWEEN 2 AND 65536
    OR pg_catalog.encode(pg_catalog.sha256(traza_bytes),'hex') IS DISTINCT FROM m->>'traza_sha256'
    OR pg_catalog.encode(pg_catalog.sha256(evento_bytes),'hex') IS DISTINCT FROM m->>'evento_sha256' THEN
  RAISE EXCEPTION 'CC7: esquema de material inválido' USING ERRCODE='22023'; END IF;
 op:=m->>'operacion'; cat:=m->>'catalogo_id'; clave:=m->>'clave_operacion';
 ver:=(m->>'version')::bigint; rev_esperada:=(m->>'revision_esperada')::bigint;
 h:=m->>'catalogo_sha256'; h_esperada:=m->>'huella_esperada';
 IF ver>2147483647 OR rev_esperada>2147483647
    OR (cat ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR (op='crear' AND m->'huella_esperada' IS DISTINCT FROM 'null'::jsonb)
    OR (op<>'crear' AND (h_esperada ~ '^[0-9a-f]{64}$') IS NOT TRUE) THEN
  RAISE EXCEPTION 'CC7: referencia o CAS inválido' USING ERRCODE='22023'; END IF;
 c:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(canon,h);
 rev:=(c->>'revision')::bigint;
 IF c->>'id' IS DISTINCT FROM cat OR c->>'version' IS DISTINCT FROM ver::text
    OR c->>'estado' IS DISTINCT FROM (CASE op WHEN 'crear' THEN 'borrador' WHEN 'actualizar' THEN 'borrador'
      WHEN 'publicar' THEN 'publicado' ELSE 'retirado' END)
    OR c->>'version_anterior_ref' IS DISTINCT FROM (CASE WHEN ver=1 THEN NULL ELSE cat||':'||(ver-1)::text END) THEN
  RAISE EXCEPTION 'CC7: catálogo no corresponde al efecto' USING ERRCODE='22023'; END IF;
 h_material:=pg_catalog.encode(pg_catalog.sha256(p_material_exacto),'hex');
 h_contexto:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  '{"ambitos":{},"atributos":{"estado":"'||(c->>'estado')||'","material_sha256":"'||h_material||
  '","revision":"'||rev::text||'"}}','UTF8')),'hex');
 -- El comprobador AD177 no confía en este envelope: relee la decisión,
 -- consumo y auditoría originales, comprueba xmin/ventana y devuelve actor.
 SELECT vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(p_consumo->'consumo') INTO STRICT v3;
 actor:=v3->>'registrador_principal_ref';
 IF pg_catalog.jsonb_typeof(v3) IS DISTINCT FROM 'object'
    OR v3->>'decision_ref' IS DISTINCT FROM p_consumo#>>'{consumo,decision_ref}'
    OR v3->>'efecto_ref' IS DISTINCT FROM cat||':'||ver::text
    OR v3->>'huella_efecto_sha256' IS DISTINCT FROM h_contexto
    OR v3->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo#>>'{consumo,consumo_huella_sha256}'
    OR v3->>'auditoria_ref' IS DISTINCT FROM p_consumo#>>'{consumo,auditoria_ref}'
    OR actor IS NULL OR actor IS DISTINCT FROM p_consumo->>'actor_ref'
    OR v3->>'registrador_perfil_ref' IS DISTINCT FROM p_consumo->>'perfil_ref'
    OR v3->>'operacion' IS DISTINCT FROM 'vec.catalogos.'||op
    OR v3->>'operacion' IS DISTINCT FROM p_consumo->>'accion'
    OR v3->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR v3->>'finalidad' IS DISTINCT FROM p_consumo->>'finalidad'
    OR v3->>'proceso' IS NULL OR v3->>'canal' IS NULL
    OR v3->>'proceso' IS DISTINCT FROM p_consumo->>'proceso'
    OR v3->>'canal' IS DISTINCT FROM p_consumo->>'canal'
    OR (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
  RAISE EXCEPTION 'CC7: decisión o consumo no ligado' USING ERRCODE='42501'; END IF;
 accion_evento:=CASE op WHEN 'crear' THEN 'vec.catalogos.borrador.creado'
  WHEN 'actualizar' THEN 'vec.catalogos.borrador.actualizado'
  WHEN 'publicar' THEN 'vec.catalogos.publicado' ELSE 'vec.catalogos.retirado' END;
 IF pg_catalog.jsonb_typeof(traza) IS DISTINCT FROM 'object' OR pg_catalog.jsonb_typeof(evento) IS DISTINCT FROM 'object'
    OR traza->>'actor_id' IS DISTINCT FROM actor OR evento->>'actor_id' IS DISTINCT FROM actor
    OR traza->>'action' IS DISTINCT FROM accion_evento OR evento->>'type' IS DISTINCT FROM accion_evento
    OR traza->>'module_id' IS DISTINCT FROM 'contratacion_temporal'
    OR evento->>'module_id' IS DISTINCT FROM 'contratacion_temporal'
    OR traza->>'subject_ref' IS DISTINCT FROM cat||':'||ver::text
    OR evento->>'subject_ref' IS DISTINCT FROM cat||':'||ver::text
    OR traza->>'after_hash' IS DISTINCT FROM h
    OR evento#>>'{payload,huella_sha256}' IS DISTINCT FROM h
    OR traza->>'result' IS DISTINCT FROM 'correcto'
    OR coalesce(traza->>'before_hash','') IS DISTINCT FROM coalesce(h_esperada,'') THEN
  RAISE EXCEPTION 'CC7: traza o evento no ligado' USING ERRCODE='22023'; END IF;
 IF (op='crear' AND (rev<>1 OR rev_esperada<>0 OR c->>'creado_por' IS DISTINCT FROM actor))
    OR (op='actualizar' AND (rev<>rev_esperada+1 OR rev_esperada<1 OR c->>'ultima_modificacion_por' IS DISTINCT FROM actor))
    OR (op IN ('publicar','retirar') AND (rev<>rev_esperada OR rev_esperada<1))
    OR (op='publicar' AND c->>'publicado_por' IS DISTINCT FROM actor)
    OR (op='retirar' AND c->>'retirado_por' IS DISTINCT FROM actor) THEN
  RAISE EXCEPTION 'CC7: transición o actor inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:plan_firma:'||cat,0));
 SELECT * INTO previo FROM vec_catalogos_configurables.plan_firma_efecto x
  WHERE x.actor_ref=actor AND x.catalogo_id=cat AND x.clave_operacion=clave;
 IF FOUND THEN
  IF previo.material_sha256 IS DISTINCT FROM h_material OR previo.operacion IS DISTINCT FROM op
     OR previo.version IS DISTINCT FROM ver OR previo.huella_resultado IS DISTINCT FROM h THEN
   RAISE EXCEPTION 'CC7: clave reutilizada con otro material' USING ERRCODE='23505'; END IF;
  SELECT x.recibo_ref INTO outbox_original FROM vec_catalogos_configurables.plan_firma_outbox x
   WHERE x.recibo_ref=previo.recibo_ref AND x.evento_sha256=m->>'evento_sha256';
  IF NOT FOUND THEN
   RAISE EXCEPTION 'CC7: recibo histórico sin outbox original' USING ERRCODE='55000'; END IF;
  IF (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
   RAISE EXCEPTION 'CC7: decisión de replay caducada' USING ERRCODE='42501'; END IF;
  RETURN QUERY SELECT previo.recibo_ref,previo.estado,previo.revision,previo.huella_resultado,
   previo.publicacion_sha256_resultado,previo.actor_ref,previo.confirmado_en,previo.auditoria_ref,outbox_original;
  RETURN;
 END IF;
 SELECT * INTO actual FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.catalogo_id=cat AND x.version=ver FOR UPDATE;
 existe:=FOUND;
 IF op='crear' THEN
  IF existe OR (ver>1 AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
       WHERE x.catalogo_id=cat AND x.version=ver-1 AND x.estado IN ('publicado','retirado'))) THEN
   RAISE EXCEPTION 'CC7: versión de borrador en conflicto' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_catalogos_configurables.plan_firma_control
   (catalogo_id,version,revision,estado,canonico_actual,huella_actual,creado_por,ultimo_editor)
  VALUES(cat,ver,rev,'borrador',canon,h,actor,actor);
 ELSE
  IF NOT existe OR actual.revision IS DISTINCT FROM rev_esperada
     OR actual.huella_actual IS DISTINCT FROM h_esperada
     OR (op IN ('actualizar','publicar') AND actual.estado IS DISTINCT FROM 'borrador')
     OR (op='retirar' AND actual.estado IS DISTINCT FROM 'publicado')
     OR c->>'creado_por' IS DISTINCT FROM actual.creado_por THEN
   RAISE EXCEPTION 'CC7: CAS de plan fallido' USING ERRCODE='40001'; END IF;
  origen:=pg_catalog.convert_from(actual.canonico_actual,'UTF8')::jsonb;
  IF op='actualizar' THEN
   IF c->>'publicado_por' IS NOT NULL OR c->>'retirado_por' IS NOT NULL
      OR c->>'creado_en' IS DISTINCT FROM origen->>'creado_en' THEN
    RAISE EXCEPTION 'CC7: edición fuera de borrador' USING ERRCODE='42501'; END IF;
  ELSIF op='publicar' THEN
   -- Un único plan publicado por módulo: no se publica junto a otro catálogo.
   PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:plan_firma:modulo:contratacion_temporal',0));
   IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
      WHERE x.modulo_id='contratacion_temporal' AND x.estado='publicado' AND x.catalogo_id<>cat) THEN
    RAISE EXCEPTION 'CC7: ya hay otro plan publicado en el módulo' USING ERRCODE='42501'; END IF;
   -- Separación de funciones: nadie que haya creado o editado esta versión publica.
   IF actor IN (actual.creado_por,actual.ultimo_editor)
      OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_historia x
        WHERE x.catalogo_id=cat AND x.version=ver AND x.operacion IN ('crear','actualizar') AND x.actor_ref=actor)
      OR c->>'aprobacion_ref' IS NULL
      OR c - 'estado' - 'publicado_por' - 'publicado_en' - 'aprobacion_ref' - 'motivo_publicacion'
         IS DISTINCT FROM origen - 'estado' - 'publicado_en' THEN
    RAISE EXCEPTION 'CC7: publicación sin separación o borrador idéntico' USING ERRCODE='42501'; END IF;
  ELSIF op='retirar' THEN
   IF actor=actual.publicado_por OR c->>'aprobacion_ref' IS DISTINCT FROM origen->>'aprobacion_ref'
      OR c->>'retirada_aprobacion_ref' IS NULL
      OR c->>'retirada_aprobacion_ref' IS NOT DISTINCT FROM origen->>'aprobacion_ref'
      OR c - 'estado' - 'retirado_por' - 'retirado_en' - 'retirada_aprobacion_ref' - 'motivo_retirada'
         IS DISTINCT FROM origen - 'estado' - 'retirado_en' THEN
    RAISE EXCEPTION 'CC7: retirada sin separación o publicación alterada' USING ERRCODE='42501'; END IF;
  END IF;
  UPDATE vec_catalogos_configurables.plan_firma_control SET
   revision=rev,estado=c->>'estado',canonico_actual=canon,huella_actual=h,
   ultimo_editor=CASE WHEN op='actualizar' THEN actor ELSE actual.ultimo_editor END,
   publicado_por=CASE WHEN op='publicar' THEN actor ELSE actual.publicado_por END,
   retirado_por=CASE WHEN op='retirar' THEN actor ELSE actual.retirado_por END,
   publicacion_sha256=CASE WHEN op='publicar' THEN h ELSE actual.publicacion_sha256 END,
   publicacion_revision=CASE WHEN op='publicar' THEN rev ELSE actual.publicacion_revision END
  WHERE catalogo_id=cat AND version=ver;
 END IF;
 recibo:='recibo:'||pg_catalog.gen_random_uuid()::text;
 INSERT INTO vec_catalogos_configurables.plan_firma_historia
  (catalogo_id,version,revision,operacion,estado,canonico_exacto,huella_sha256,actor_ref,decision_ref,recibo_ref)
 VALUES(cat,ver,rev,op,c->>'estado',canon,h,actor,v3->>'decision_ref',recibo);
 IF op='publicar' THEN
  momento:=(c->>'publicado_en')::timestamptz;
  INSERT INTO vec_catalogos_configurables.plan_firma_publicacion
   (catalogo_id,version,revision,canonico_exacto,publicacion_sha256,publicado_por,publicada_en,aprobacion_ref)
  VALUES(cat,ver,rev,canon,h,actor,momento,c->>'aprobacion_ref');
 END IF;
 SELECT x.publicacion_sha256 INTO h_publicacion FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.catalogo_id=cat AND x.version=ver;
 INSERT INTO vec_catalogos_configurables.plan_firma_efecto
  (actor_ref,catalogo_id,clave_operacion,operacion,version,revision,estado,huella_resultado,publicacion_sha256_resultado,
   material_sha256,recibo_ref,decision_ref,auditoria_ref,consumo_huella_sha256,traza_sha256,evento_sha256)
 VALUES(actor,cat,clave,op,ver,rev,c->>'estado',h,h_publicacion,h_material,recibo,v3->>'decision_ref',
  v3->>'auditoria_ref',v3->>'consumo_huella_sha256',m->>'traza_sha256',m->>'evento_sha256');
 INSERT INTO vec_catalogos_configurables.plan_firma_outbox(recibo_ref,evento_exacto,evento_sha256)
 VALUES(recibo,evento_bytes,m->>'evento_sha256');
 SELECT x.actor_ref,x.confirmado_en,x.auditoria_ref,o.recibo_ref
  INTO actor_original,fecha_original,auditoria_original,outbox_original
  FROM vec_catalogos_configurables.plan_firma_efecto x
  JOIN vec_catalogos_configurables.plan_firma_outbox o ON o.recibo_ref=x.recibo_ref
  WHERE x.recibo_ref=recibo AND x.actor_ref=actor AND x.material_sha256=h_material
    AND o.evento_sha256=m->>'evento_sha256';
 IF NOT FOUND OR actor_original IS NULL OR fecha_original IS NULL OR auditoria_original IS NULL OR outbox_original IS NULL THEN
  RAISE EXCEPTION 'CC7: efecto confirmado sin outbox original' USING ERRCODE='55000'; END IF;
 IF (v3->>'decision_valida_hasta' IS NULL OR (v3->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp()) THEN
  RAISE EXCEPTION 'CC7: decisión caducada antes del efecto' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT recibo,c->>'estado',rev,h,h_publicacion,
  actor_original,fecha_original,auditoria_original,outbox_original;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;
-- Los privilegios por defecto heredados de la base pueden contener más roles
-- que PUBLIC. Se sanea y comprueba sólo el conjunto nuevo de CC7.
DO $acl_final$
DECLARE f regprocedure; t text; x record;
 owner_id oid:='vec_catalogos_configurables_propietario'::regrole;
 ad_id oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 ct_id oid:='vec_contratacion_temporal_propietario'::regrole;
 permitido boolean;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_firma_control','plan_firma_historia','plan_firma_publicacion',
  'plan_firma_efecto','plan_firma_outbox'] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t) AND a.grantee<>owner_id LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM %s',t,
    CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_type y
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(y.typacl,pg_catalog.acldefault('T',y.typowner))) a
    WHERE y.typrelid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t) AND a.grantee<>owner_id LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM %s',t,
    CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t) AND a.grantee<>owner_id)
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_type y
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(y.typacl,pg_catalog.acldefault('T',y.typowner))) a
    WHERE y.typrelid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t) AND a.grantee<>owner_id) THEN
   RAISE EXCEPTION 'CC7: ACL de tabla o tipo incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_catalogos_configurables.json_plan_sin_duplicados_v1(json,integer)'::regprocedure,
  'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure,
  'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure,
  'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE p.oid=f LOOP
   permitido:=x.grantee=owner_id OR
    (f='vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure AND x.grantee=ct_id) OR
    (f='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure AND x.grantee=ad_id);
   IF NOT permitido THEN
    EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
     CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
   END IF;
  END LOOP;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee<>owner_id
     AND NOT (f='vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure AND a.grantee=ct_id)
     AND NOT (f='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure AND a.grantee=ad_id)) THEN
   RAISE EXCEPTION 'CC7: ACL de función incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege(ct_id,
   'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)','EXECUTE')
    OR pg_catalog.has_function_privilege(ct_id,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege(ad_id,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'CC7: ACL final de propietarios incompatible' USING ERRCODE='55000'; END IF;
END $acl_final$;
COMMIT;
