\set ON_ERROR_STOP on
-- P51: limita el recibo de lectura histórica CRN11 de Personal26 a las
-- operaciones SELECT/INSERT y a la pareja persona-empleado ya acreditada.
-- Personal26 queda intacta en Git; esta sucesora conserva su OID y ACL.
-- Requiere AD269 y Personal26 recién instalada. Si hubiera recibos previos,
-- para sin modificarlos: ese estado necesita una migración de historia aparte.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000051',0));
DO $pre$
DECLARE politica record; f oid:='vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_personal.recibo_vinculo_propio_crn11') IS NULL
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL
    OR to_regprocedure('vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND a.attname IN ('persona_ref','proyeccion_ref','proyeccion_version') AND NOT a.attisdropped)
    OR EXISTS (SELECT 1 FROM vec_personal.recibo_vinculo_propio_crn11)
 THEN RAISE EXCEPTION 'P51: preimagen incompatible o historia previa' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR (SELECT array_agg(a.attname::text ORDER BY a.attnum) FROM pg_attribute a
         WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass AND a.attnum>0 AND NOT a.attisdropped)
       IS DISTINCT FROM ARRAY['recibo_ref','empleado_ref','material_sha256','decision_ref','auditoria_ref','consumo_huella_sha256','consultada_en']
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
         AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attnotnull)
    OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
       WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass AND a.grantee<>c.relowner)
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.proyeccion_empleado_persona_historia'::regclass
       AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_personal.proyeccion_empleado_persona_historia'::regclass
       AND c.conname='proyeccion_empleado_persona_historia_pkey' AND c.contype='p'
       AND pg_get_constraintdef(c.oid,true)='PRIMARY KEY (proyeccion_ref, version)')
 THEN RAISE EXCEPTION 'P51: tabla o fuente incompatible' USING ERRCODE='55000'; END IF;
 SELECT pol.polname AS nombre,pol.polcmd AS comando,pol.polroles AS roles,
   pg_get_expr(pol.polqual,pol.polrelid) AS condicion,pg_get_expr(pol.polwithcheck,pol.polrelid) AS condicion_insercion
   INTO STRICT politica FROM pg_policy pol WHERE pol.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass;
 IF politica.nombre<>'propietario_interno' OR politica.comando<>'*'
    OR politica.roles<>ARRAY['vec_personal_propietario'::regrole::oid]
    OR politica.condicion<>'true' OR politica.condicion_insercion<>'true' THEN
   RAISE EXCEPTION 'P51: política Personal26 incompatible' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND t.tgname IN ('historia_inmutable','no_truncar') AND t.tgenabled='O'
       AND t.tgfoid='vec_personal.rechazar_mutacion_proyeccion_empleado_v1()'::regprocedure)<>2 THEN
   RAISE EXCEPTION 'P51: historia sin protección' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
       AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
       AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='ab9dc0b7838bfd0fa580239cee7f6ac1e8506ccfcb98d4af5201cf0720dab8d7')
    OR NOT has_function_privilege('vec_personal_ejecutor','vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee NOT IN (p.proowner,'vec_personal_ejecutor'::regrole))
 THEN RAISE EXCEPTION 'P51: función Personal26 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_personal.recibo_vinculo_propio_crn11
 ADD COLUMN persona_ref text NOT NULL CHECK (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 ADD COLUMN proyeccion_ref text NOT NULL CHECK (proyeccion_ref ~ '^pep_[A-Za-z0-9_-]{22,128}$'),
 ADD COLUMN proyeccion_version bigint NOT NULL CHECK (proyeccion_version>0),
 ADD CONSTRAINT recibo_vinculo_crn11_fuente_fk FOREIGN KEY (proyeccion_ref,proyeccion_version)
   REFERENCES vec_personal.proyeccion_empleado_persona_historia(proyeccion_ref,version);

CREATE FUNCTION vec_personal.recibo_crn11_fuente_valida_v1(p_proyeccion text,p_version bigint,p_persona text,p_empleado text,p_consultada timestamptz)
RETURNS boolean LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref=p_proyeccion AND h.version=p_version
    AND h.persona_ref=p_persona AND h.empleado_ref=p_empleado AND h.estado='activa'
    AND h.vigente_desde<=p_consultada AND p_consultada<h.vigente_hasta)
$f$;
REVOKE ALL ON FUNCTION vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

DO $funcion_pre$
DECLARE f oid:='vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=$x$ INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=f),'UTF8')),'hex')<>'ab9dc0b7838bfd0fa580239cee7f6ac1e8506ccfcb98d4af5201cf0720dab8d7'
    OR propietario<>'vec_personal_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR (length(original)-length(replace(original,marca,'')))<>length(marca)
 THEN RAISE EXCEPTION 'P51: punto de inserción incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec_p51.funcion_pre',jsonb_build_object('meta',meta,'deps',deps,
   'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
    FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))::text,true);
END $funcion_pre$;

CREATE OR REPLACE FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
 m jsonb; d jsonb; c jsonb; ctx jsonb; proyeccion record; fuente record; consumo record;
 empleado text; persona text; material_canon text; material_sha text;
 contexto_canon text; contexto_sha text; ahora timestamptz(6); recibo text;
 k text; vinculos integer;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolbypassrls)
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material)>4096
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'vínculo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de vínculo CRN11 inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(ctx) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'material de vínculo CRN11 inválido' USING ERRCODE='22023';
 END IF;
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
   'empleado_ref','esquema','perfil_ref','perfil_version','persona_ref','persona_version',
   'vinculo_ref','vinculo_version']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.vinculo-propio-crn11.consulta.v1'
    OR m->>'empleado_ref' IS NULL OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' IS NULL OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'actor_ref' IS DISTINCT FROM m->>'persona_ref'
    OR m->>'cuenta_ref' IS NULL OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' IS NULL OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' IS NULL OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'vinculo_ref' IS NULL OR m->>'vinculo_ref' !~ '^pep_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'material de vínculo CRN11 incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','perfil_version','persona_version','vinculo_version'] LOOP
  IF jsonb_typeof(m->k) IS DISTINCT FROM 'number'
     OR m->>k IS NULL OR m->>k !~ '^[1-9][0-9]{0,18}$'
     OR (m->>k)::numeric>9223372036854775807 THEN
   RAISE EXCEPTION 'versión de vínculo CRN11 inválida' USING ERRCODE='22023';
  END IF;
 END LOOP;
 empleado:=m->>'empleado_ref'; persona:=m->>'persona_ref';
 IF m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR persona IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref' THEN
  RAISE EXCEPTION 'actor de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- Orden exacto de json.Marshal(MaterialVinculoPropioCRN11).
 material_canon:='{"esquema":"vec.personal.vinculo-propio-crn11.consulta.v1","empleado_ref":'||to_jsonb(empleado)::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(persona)::text||
  ',"persona_version":'||(m->>'persona_version')||',"vinculo_ref":'||to_jsonb(m->>'vinculo_ref')::text||
  ',"vinculo_version":'||(m->>'vinculo_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de vínculo CRN11 no canónico' USING ERRCODE='22023';
 END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||
  '},"atributos":{"material_sha256":"'||material_sha||'","operacion":"vinculo_propio_historico_crn11"}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_historico_propio_crn11'
    OR d->>'finalidad' IS DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
    OR d->'campos_permitidos' IS DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS DISTINCT FROM empleado
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM empleado
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- El canon del contexto y su uso vigente los verifica el núcleo V3/CA7.
 -- El material además debe referirse a la entrada exacta del empleado.
 IF ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR ctx->>'principal_ref' IS DISTINCT FROM m->>'actor_ref'
    OR ctx->>'persona_ref' IS DISTINCT FROM persona
    OR ctx->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR ctx->>'estado' IS DISTINCT FROM 'activo'
    OR jsonb_typeof(ctx->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'contexto de vínculo CRN11 incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH k IN ARRAY ARRAY['contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','persona_version','perfil_version'] LOOP
  IF ctx->k IS DISTINCT FROM m->k THEN
   RAISE EXCEPTION 'contexto de vínculo CRN11 divergente' USING ERRCODE='42501';
  END IF;
 END LOOP;
 SELECT count(*) INTO vinculos FROM jsonb_array_elements(ctx->'vinculos') v(e)
  WHERE v.e->>'tipo'='empleado';
 IF vinculos<>1 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ctx->'vinculos') v(e)
     WHERE v.e->>'tipo'='empleado' AND v.e->>'referencia'=empleado
       AND v.e->>'vinculo_ref'=m->>'vinculo_ref' AND v.e->'version'=m->'vinculo_version'
       AND v.e->>'estado'='activo') THEN
  RAISE EXCEPTION 'contexto de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,clock_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado'
    OR proyeccion.persona_ref IS DISTINCT FROM persona OR proyeccion.empleado_ref IS DISTINCT FROM empleado
    OR proyeccion.proyeccion_ref IS DISTINCT FROM m->>'vinculo_ref'
    OR proyeccion.version::text IS DISTINCT FROM m->>'vinculo_version' THEN
  RAISE EXCEPTION 'vínculo actual CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- Consumo nuevo: V3 revalida permiso/contexto/revocación antes de la lectura.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM empleado OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'consumo de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 SELECT h.* INTO STRICT fuente FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref=m->>'vinculo_ref' AND h.version=(m->>'vinculo_version')::bigint
    AND h.persona_ref=persona AND h.empleado_ref=empleado;
 IF fuente.procedencia_ref IS DISTINCT FROM proyeccion.procedencia_ref
    OR fuente.procedencia_ref IS NULL OR fuente.procedencia_ref !~ '^prc_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'fuente de vínculo CRN11 divergente' USING ERRCODE='55000';
 END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz
    OR ahora<proyeccion.vigente_desde OR ahora>=proyeccion.vigente_hasta THEN
  RAISE EXCEPTION 'vínculo CRN11 caducado' USING ERRCODE='42501';
 END IF;
 recibo:='vinculocrn11:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,persona_ref,proyeccion_ref,proyeccion_version,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,persona,fuente.proyeccion_ref,fuente.version,
  material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 RETURN jsonb_build_object('vinculo',jsonb_build_object(
  'persona_ref',persona,'empleado_ref',empleado,'vinculo_ref',fuente.proyeccion_ref,
  'fuente_ref',fuente.procedencia_ref,'version',fuente.version),
  'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
  'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
  'auditoria_ref',consumo.auditoria_ref,
  'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $function$
;

DO $funcion_post$
DECLARE f oid:='vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 actual text; snapshot jsonb;
 marca text:=$x$ INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);$x$;
 sustitucion text:=$x$ INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,persona_ref,proyeccion_ref,proyeccion_version,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,persona,fuente.proyeccion_ref,fuente.version,
  material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);$x$;
BEGIN
 snapshot:=current_setting('vec_p51.funcion_pre',true)::jsonb;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF encode(sha256(convert_to(actual,'UTF8')),'hex')<>'fb1c44d936bda8a8b4ae46125c48a3ad2a2d0b25083b22805bdb459626590792'
    OR encode(sha256(convert_to(replace(actual,sustitucion,marca),'UTF8')),'hex')<>'6ae112230fd23f86ad0fea2a9215848ef3c4736d0012ddce1db9d1913ea8a800'
    OR encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=f),'UTF8')),'hex')<>'5fc35c28e123d01fe134dc9be16b3bddd386c51045d9e803ce01d8ff7ec6f9a1'
    OR snapshot IS NULL
    OR (SELECT jsonb_build_object('meta',to_jsonb(p)-'prosrc',
       'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f),
       'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))
       FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM snapshot
 THEN RAISE EXCEPTION 'P51: función alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $funcion_post$;


DROP POLICY propietario_interno ON vec_personal.recibo_vinculo_propio_crn11;
CREATE POLICY lectura_fuente ON vec_personal.recibo_vinculo_propio_crn11 FOR SELECT TO vec_personal_propietario
 USING (vec_personal.recibo_crn11_fuente_valida_v1(proyeccion_ref,proyeccion_version,persona_ref,empleado_ref,consultada_en));
CREATE POLICY adicion_fuente ON vec_personal.recibo_vinculo_propio_crn11 FOR INSERT TO vec_personal_propietario
 WITH CHECK (vec_personal.recibo_crn11_fuente_valida_v1(proyeccion_ref,proyeccion_version,persona_ref,empleado_ref,consultada_en));
REVOKE ALL ON TYPE vec_personal.recibo_vinculo_propio_crn11 FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

DO $final$
BEGIN
 IF (SELECT count(*) FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND p.polname IN ('lectura_fuente','adicion_fuente')
      AND ((p.polname='lectura_fuente' AND p.polcmd='r') OR (p.polname='adicion_fuente' AND p.polcmd='a'))
      AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid])<>2
    OR EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND p.polname NOT IN ('lectura_fuente','adicion_fuente'))
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR EXISTS (SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
      WHERE t.oid='vec_personal.recibo_vinculo_propio_crn11'::regtype AND a.grantee<>t.typowner)
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid='vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)'::regprocedure
      AND p.proowner='vec_personal_propietario'::regrole AND NOT p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid='vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)'::regprocedure
        AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'P51: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $final$;
COMMIT;
