\set ON_ERROR_STOP on
-- P52: unidad de trabajo propia en una fecha y corte de conocimiento.
-- Dependencias: Personal16/17/51 y fachada nominal AD273 instalada antes.
-- Lectura, consumo V3, auditoría común y recibo se confirman en una TX.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000052:unidad-trabajo-cronos',0));
DO $pre$
DECLARE c oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_personal_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR c IS NULL
  OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
  OR to_regclass('vec_personal.recibo_vinculo_propio_crn11') IS NULL
  OR to_regprocedure('vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)') IS NULL
  OR to_regprocedure('vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)') IS NULL
  OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NULL
  OR to_regprocedure('vec_personal.rechazar_mutacion_registro_empleado_v1()') IS NULL
  OR to_regclass('vec_personal.recibo_unidad_trabajo_cronos') IS NOT NULL
  OR to_regprocedure('vec_personal.consultar_unidad_trabajo_propia_cronos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_class t WHERE t.oid='vec_personal.relacion_servicio_historia'::regclass
       AND t.relowner='vec_personal_propietario'::regrole AND t.relrowsecurity AND t.relforcerowsecurity)
  OR NOT has_function_privilege('vec_personal_propietario',c,'EXECUTE')
  OR has_function_privilege('vec_personal_ejecutor',c,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=c AND (p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT p.prosecdef
         OR NOT p.proretset OR p.prorettype<>'record'::regtype OR p.pronargs<>10
         OR p.proargnames[11:17] IS DISTINCT FROM ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']
         OR a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'P52: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_personal.recibo_unidad_trabajo_cronos (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^unidadcronos:[0-9a-f-]{36}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 fecha date NOT NULL CHECK(isfinite(fecha)),
 conocido_en timestamptz(6) NOT NULL CHECK(isfinite(conocido_en)),
 cobertura text NOT NULL CHECK(cobertura IN ('completa','sin_datos','no_acreditada')),
 relacion_ref text CHECK(relacion_ref IS NULL OR relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 relacion_revision integer CHECK(relacion_revision IS NULL OR relacion_revision>0),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL, auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consultada_en timestamptz(6) NOT NULL CHECK(isfinite(consultada_en)),
 CHECK ((cobertura='completa')=(relacion_ref IS NOT NULL AND relacion_revision IS NOT NULL)),
 FOREIGN KEY(relacion_ref,relacion_revision) REFERENCES vec_personal.relacion_servicio_historia(relacion_ref,revision)
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_unidad_trabajo_cronos
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_unidad_trabajo_cronos
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_unidad_trabajo_cronos ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_unidad_trabajo_cronos FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propietario ON vec_personal.recibo_unidad_trabajo_cronos FOR SELECT TO vec_personal_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_personal.recibo_unidad_trabajo_cronos FOR INSERT TO vec_personal_propietario WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.recibo_unidad_trabajo_cronos FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;
REVOKE ALL ON TYPE vec_personal.recibo_unidad_trabajo_cronos FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

CREATE FUNCTION vec_personal.consultar_unidad_trabajo_propia_cronos_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET datestyle='ISO, YMD'
 SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; ctx jsonb; vinculo jsonb; proyeccion record; consumo record;
 fuente vec_personal.relacion_servicio_historia%ROWTYPE;
 empleado text; persona text; fecha date; conocido timestamptz(6); ahora timestamptz(6);
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 candidatos integer; cobertura text; unidad jsonb; recibo text; k text;
 campos constant jsonb:='["desde","empleado_ref","fuente_ref","fuente_version_ref","hasta_exclusivo","organismo_ref","regimen_ref","relacion_ref","unidad_clave","version"]';
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members r WHERE r.member=session_user::regrole AND r.roleid='vec_personal_ejecutor'::regrole
       AND r.inherit_option AND NOT r.set_option AND NOT r.admin_option)
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members r WHERE r.member=session_user::regrole)<>1
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members r WHERE r.member='vec_personal_ejecutor'::regrole)
  OR has_table_privilege(session_user,'vec_personal.relacion_servicio_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
  OR has_table_privilege(session_user,'vec_personal.recibo_unidad_trabajo_cronos','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
  OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536
  OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 65536
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
  OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
  OR p_persona_version IS NULL OR p_perfil_version IS NULL
  OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
  OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
  OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
  OR p_raiz IS NULL OR octet_length(p_raiz)<>44
 THEN RAISE EXCEPTION 'unidad propia denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
  fecha:=(m->>'fecha')::date; conocido:=(m->>'conocido_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de unidad inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
  OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(ctx) IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['conocido_en','contexto_actor','empleado_ref','esquema','fecha']
  OR m->>'esquema' IS DISTINCT FROM 'vec.personal.unidad-trabajo-propia-cronos.consulta.v1'
  OR m->'contexto_actor' IS DISTINCT FROM ctx
  OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
  OR coalesce(m->>'fecha','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
  OR fecha IS NULL OR NOT isfinite(fecha) OR fecha::text IS DISTINCT FROM m->>'fecha'
  OR conocido IS NULL OR NOT isfinite(conocido) OR conocido>clock_timestamp()
  OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
  OR ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
  OR ctx->>'estado' IS DISTINCT FROM 'activo'
  OR jsonb_typeof(ctx->'vinculos') IS DISTINCT FROM 'array'
  OR coalesce(ctx->>'persona_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR ctx->>'principal_ref' IS DISTINCT FROM ctx->>'persona_ref'
  OR coalesce(ctx->>'cuenta_ref','') !~ '^cta_[A-Za-z0-9_-]{22,128}$'
  OR coalesce(ctx->>'perfil_activo_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'material de unidad incompatible' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','persona_version','perfil_version'] LOOP
  IF jsonb_typeof(ctx->k) IS DISTINCT FROM 'number' OR coalesce(ctx->>k,'') !~ '^[1-9][0-9]{0,18}$'
   OR (ctx->>k)::numeric>9223372036854775807 THEN
   RAISE EXCEPTION 'versión de contexto inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 empleado:=m->>'empleado_ref'; persona:=ctx->>'persona_ref';
 IF ctx->>'persona_version' IS DISTINCT FROM p_persona_version::text
  OR ctx->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
  OR d->>'principal_id' IS DISTINCT FROM persona
  OR d->>'perfil_activo_ref' IS DISTINCT FROM ctx->>'perfil_activo_ref'
 THEN RAISE EXCEPTION 'actor de unidad divergente' USING ERRCODE='42501'; END IF;
 material_canon:='{"esquema":"vec.personal.unidad-trabajo-propia-cronos.consulta.v1","empleado_ref":'||to_jsonb(empleado)::text||
  ',"fecha":'||to_jsonb(m->>'fecha')::text||',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"contexto_actor":'||convert_from(p_contexto,'UTF8')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN RAISE EXCEPTION 'material de unidad no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||
  '},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"fecha":'||to_jsonb(m->>'fecha')::text||',"material_sha256":"'||material_sha||
  '","operacion":"unidad_trabajo_propia_cronos"}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
  OR d->>'accion' IS DISTINCT FROM 'personal.unidad_trabajo.propia.consultar'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'unidad_trabajo_propia_cronos'
  OR d->>'finalidad' IS DISTINCT FROM 'consultar_unidad_trabajo_propia_cronos'
  OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
  OR d->>'recurso_ref' IS DISTINCT FROM empleado OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
  OR c->>'operacion' IS DISTINCT FROM d->>'accion'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.unidad_trabajo.propia.cronos.v1'
  OR c->>'efecto_ref' IS DISTINCT FROM empleado OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
 THEN RAISE EXCEPTION 'concesión de unidad divergente' USING ERRCODE='42501'; END IF;
 IF (SELECT count(*) FROM jsonb_array_elements(ctx->'vinculos') v(e) WHERE v.e->>'tipo'='empleado')<>1 THEN
  RAISE EXCEPTION 'empleado ambiguo' USING ERRCODE='42501'; END IF;
 SELECT v.e INTO STRICT vinculo FROM jsonb_array_elements(ctx->'vinculos') v(e) WHERE v.e->>'tipo'='empleado';
 IF vinculo->>'referencia' IS DISTINCT FROM empleado OR vinculo->>'estado' IS DISTINCT FROM 'activo'
  OR coalesce(vinculo->>'vinculo_ref','') !~ '^pep_[A-Za-z0-9_-]{22,128}$'
  OR jsonb_typeof(vinculo->'version') IS DISTINCT FROM 'number'
  OR coalesce(vinculo->>'version','') !~ '^[1-9][0-9]{0,18}$'
  OR (vinculo->>'version')::numeric>9223372036854775807
 THEN RAISE EXCEPTION 'vínculo de empleado divergente' USING ERRCODE='42501'; END IF;
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,clock_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado' OR proyeccion.persona_ref IS DISTINCT FROM persona
  OR proyeccion.empleado_ref IS DISTINCT FROM empleado OR proyeccion.proyeccion_ref IS DISTINCT FROM vinculo->>'vinculo_ref'
  OR proyeccion.version::text IS DISTINCT FROM vinculo->>'version'
 THEN RAISE EXCEPTION 'empleado propio no vigente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
  OR consumo.efecto_ref IS DISTINCT FROM empleado OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
  OR coalesce(consumo.consumo_huella_sha256,'') !~ '^[0-9a-f]{64}$'
  OR consumo.auditoria_ref IS DISTINCT FROM 'aud_v3_'||left(consumo.consumo_huella_sha256,32)
 THEN RAISE EXCEPTION 'consumo de unidad divergente' USING ERRCODE='42501'; END IF;
 IF EXISTS(
  SELECT 1 FROM (SELECT DISTINCT ON (h.relacion_ref) h.persona_ref
   FROM vec_personal.relacion_servicio_historia h
   WHERE h.empleado_ref=empleado AND h.conocido_desde<=conocido
   ORDER BY h.relacion_ref,h.conocido_desde DESC,h.revision DESC) actual
  WHERE actual.persona_ref<>persona
 ) THEN RAISE EXCEPTION 'relación propia incoherente' USING ERRCODE='55000'; END IF;
 -- Una revisión por relación, conocida en el corte. Relaciones concurrentes
 -- cuentan todas; ninguna prioridad oculta selecciona una unidad arbitraria.
 WITH ultimas AS (
  SELECT DISTINCT ON (h.relacion_ref) h.* FROM vec_personal.relacion_servicio_historia h
  WHERE h.empleado_ref=empleado AND h.persona_ref=persona AND h.conocido_desde<=conocido
  ORDER BY h.relacion_ref,h.conocido_desde DESC,h.revision DESC
 ), candidatas AS (
  SELECT * FROM ultimas h WHERE h.vigente_desde<=fecha AND (h.vigente_hasta IS NULL OR fecha<h.vigente_hasta)
 )
 SELECT count(*)::integer INTO candidatos FROM (SELECT 1 FROM candidatas LIMIT 2) dos;
 IF candidatos=0 THEN cobertura:='sin_datos'; unidad:=NULL;
 ELSIF candidatos>1 THEN cobertura:='no_acreditada'; unidad:=NULL;
 ELSE
  WITH ultimas AS (
   SELECT DISTINCT ON (h.relacion_ref) h.* FROM vec_personal.relacion_servicio_historia h
   WHERE h.empleado_ref=empleado AND h.persona_ref=persona AND h.conocido_desde<=conocido
   ORDER BY h.relacion_ref,h.conocido_desde DESC,h.revision DESC
  )
  SELECT h.* INTO STRICT fuente FROM ultimas h
   WHERE h.vigente_desde<=fecha AND (h.vigente_hasta IS NULL OR fecha<h.vigente_hasta);
  IF fuente.estado<>'vigente' THEN cobertura:='no_acreditada'; unidad:=NULL;
  ELSE
   cobertura:='completa';
   unidad:=jsonb_build_object('empleado_ref',empleado,'relacion_ref',fuente.relacion_ref,
    'organismo_ref',fuente.organismo_ref,'unidad_clave',fuente.unidad_ref,'regimen_ref',fuente.regimen_ref,
    'desde',fuente.vigente_desde::text,'hasta_exclusivo',coalesce(fuente.vigente_hasta::text,''),
    'fuente_ref',fuente.fuente_ref,'fuente_version_ref',fuente.fuente_version::text,'version',fuente.revision);
  END IF;
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz
  OR ahora<proyeccion.vigente_desde OR ahora>=proyeccion.vigente_hasta
 THEN RAISE EXCEPTION 'lectura de unidad caducada' USING ERRCODE='42501'; END IF;
 recibo:='unidadcronos:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_unidad_trabajo_cronos(
  recibo_ref,empleado_ref,fecha,conocido_en,cobertura,relacion_ref,relacion_revision,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,fecha,conocido,cobertura,
  CASE WHEN cobertura='completa' THEN fuente.relacion_ref ELSE NULL END,
  CASE WHEN cobertura='completa' THEN fuente.revision ELSE NULL END,
  material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 RETURN jsonb_build_object('unidad',unidad,'cobertura',cobertura,
  'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
   'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_unidad_trabajo_propia_cronos_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_migrador;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_unidad_trabajo_propia_cronos_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
DO $final$
DECLARE f oid:='vec_personal.consultar_unidad_trabajo_propia_cronos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class t WHERE t.oid='vec_personal.recibo_unidad_trabajo_cronos'::regclass
      AND t.relowner='vec_personal_propietario'::regrole AND t.relrowsecurity AND t.relforcerowsecurity)
  OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_unidad_trabajo_cronos'::regclass
      AND p.polname IN ('lectura_propietario','adicion_propietario')
      AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid])<>2
  OR EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_unidad_trabajo_cronos'::regclass
      AND p.polname NOT IN ('lectura_propietario','adicion_propietario'))
  OR EXISTS(SELECT 1 FROM pg_class t,LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
      WHERE t.oid='vec_personal.recibo_unidad_trabajo_cronos'::regclass AND a.grantee<>t.relowner)
  OR EXISTS(SELECT 1 FROM pg_type t,LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
      WHERE t.oid='vec_personal.recibo_unidad_trabajo_cronos'::regtype AND a.grantee<>t.typowner)
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v')
  OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole)
       OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'P52: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $final$;
COMMIT;
