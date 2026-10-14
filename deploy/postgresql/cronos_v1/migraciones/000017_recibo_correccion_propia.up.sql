\set ON_ERROR_STOP on
-- CRN17: lectura auditada del recibo v1 original de un olvido propio.
-- Requiere AD267, CRN16 y AD269; no cambia marcajes ni actuaciones.
-- Preimagen: cuerpo y ruta medidos tras CRN16 en la copia HZ37.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000017',0));
DO $pre$
BEGIN
 IF current_user<>'vec_cronos_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_cronos_v1.correccion_solicitud') IS NULL
    OR to_regclass('vec_cronos_v1.correccion_actuacion') IS NULL
    OR to_regprocedure('vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regclass('vec_cronos_v1.correccion_recibo_acceso') IS NOT NULL
    OR to_regclass('vec_cronos_v1.solicitud_outbox_correccion_recibo_idx') IS NOT NULL
    OR to_regprocedure('vec_cronos_v1.recuperar_recibo_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p
         WHERE p.oid='vec_cronos_v1.solicitar_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       <>'9b0151b5720e3792fbcaa8afa963577cad8b17d6211501cd6574dd74534faa36'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex') FROM pg_constraint c
         WHERE c.conrelid='vec_cronos_v1.solicitud_outbox'::regclass AND c.conname='solicitud_outbox_tipo_check')
       <>'4bb48b1e56c6510a0b14e31cb0363959bfcb031e48655c9e1ad785a16bfda9a8'
    OR EXISTS (SELECT 1 FROM pg_class c WHERE c.oid IN ('vec_cronos_v1.correccion_solicitud'::regclass,
         'vec_cronos_v1.correccion_actuacion'::regclass,'vec_cronos_v1.solicitud_outbox'::regclass)
       AND (c.relowner<>'vec_cronos_v1_propietario'::regrole OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity OR c.relacl IS NOT NULL))
 THEN RAISE EXCEPTION 'CRN17: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La lista cerrada del helper instalado por CRN16 admite esta fachada nueva.
-- Se conserva firma, OID, ACL, dependencias y todos los contratos anteriores.
DO $consumo$
DECLARE f oid:='vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=$x$'consumir_cronos_permisos_propio_v3_atestada','registrar_y_consumir_cronos_permiso_v3_atestada') THEN$x$;
 sustitucion text:=$x$'consumir_cronos_permisos_propio_v3_atestada','registrar_y_consumir_cronos_permiso_v3_atestada',
     'consumir_cronos_correccion_recibo_propio_v3_atestada') THEN$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'29c4b8e3aa24d117a0cccf6bee52e9bb15b41c8de97388a99a189a37c7b5d265'
    OR propietario<>'vec_cronos_v1_propietario'::regrole OR definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
    OR acl::text<>'{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario}'
    OR (length(original)-length(replace(original,marca,'')))<>length(marca)
    OR strpos(original,'consumir_cronos_correccion_recibo_propio_v3_atestada')<>0
 THEN RAISE EXCEPTION 'CRN17: helper de consumo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustitucion);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,sustitucion,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'CRN17: helper alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $consumo$;

CREATE INDEX solicitud_outbox_correccion_recibo_idx ON vec_cronos_v1.solicitud_outbox(agregado_ref,empleado_ref)
 WHERE tipo='cronos.correccion.solicitada';

CREATE TABLE vec_cronos_v1.correccion_recibo_acceso (
 decision_ref text PRIMARY KEY,
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
 actuacion_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_actuacion,
 recibo_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_actuacion(recibo_ref),
 version_recibo integer NOT NULL CHECK (version_recibo=1),
 material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consultada_en timestamptz(6) NOT NULL
);
CREATE INDEX correccion_recibo_acceso_empleado_idx ON vec_cronos_v1.correccion_recibo_acceso(empleado_ref,solicitud_ref);
ALTER TABLE vec_cronos_v1.correccion_recibo_acceso ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_recibo_acceso FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propia ON vec_cronos_v1.correccion_recibo_acceso FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.correccion_recibo_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_recibo_acceso
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_recibo_acceso FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.correccion_recibo_acceso FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

-- Objeto exacto de siete cadenas. Se mira json antes de jsonb para detectar
-- claves repetidas; la huella V3 resume los bytes recibidos, sin reconstruirlos.
CREATE FUNCTION vec_cronos_v1.material_recuperacion_correccion_propia_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE bruto json; m jsonb; cantidad bigint; distintas bigint;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   bruto:=p_material::json;
   IF json_typeof(bruto) IS DISTINCT FROM 'object' THEN
     RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
   END IF;
   SELECT count(*),count(DISTINCT e.key) INTO cantidad,distintas FROM json_each(bruto) e;
   m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 IF cantidad<>7 OR distintas<>7 OR m-ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','solicitud_ref','paso','version_recibo']<>'{}'::jsonb
    OR EXISTS (SELECT 1 FROM jsonb_each(m) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
    OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'clave_operacion','') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
    OR m->>'solicitud_ref' IS DISTINCT FROM 'correccion:cronos:'||(m->>'clave_operacion')
    OR m->>'paso' IS DISTINCT FROM 'solicitud' OR m->>'version_recibo' IS DISTINCT FROM '1'
 THEN RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001'; END IF;
 RETURN m;
END $f$;

CREATE FUNCTION vec_cronos_v1.recuperar_recibo_correccion_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; s vec_cronos_v1.correccion_solicitud%ROWTYPE; a vec_cronos_v1.correccion_actuacion%ROWTYPE; eventos bigint;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR p_raiz IS NULL OR octet_length(p_raiz)<>44
    OR p_persona_version IS NULL OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
    OR p_perfil_version IS NULL OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric THEN
   RAISE EXCEPTION 'material V3 Cronos inválido' USING ERRCODE='PC003';
 END IF;
 m:=vec_cronos_v1.material_recuperacion_correccion_propia_v1(p_material);
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_propio_v1(
  'consumir_cronos_correccion_recibo_propio_v3_atestada','cronos.correccion.recibo.consultar',
  'recibo_correccion_propia','consultar_recibo_correccion_propia',m->>'solicitud_ref',NULL,m,p_material,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN
   SELECT * INTO STRICT s FROM vec_cronos_v1.correccion_solicitud solicitud
    WHERE solicitud.empleado_ref=m->>'empleado_ref' AND solicitud.actor_ref=m->>'actor_ref'
      AND solicitud.perfil_ref=m->>'perfil_ref'
      AND solicitud.solicitud_ref=m->>'solicitud_ref' AND solicitud.clave_operacion=m->>'clave_operacion';
 EXCEPTION WHEN no_data_found THEN
   RAISE EXCEPTION 'recibo propio ausente' USING ERRCODE='PC004';
 WHEN too_many_rows THEN
   RAISE EXCEPTION 'historia Cronos incompatible' USING ERRCODE='PC003';
 END;
 BEGIN
   SELECT * INTO STRICT a FROM vec_cronos_v1.correccion_actuacion actuacion
    WHERE actuacion.solicitud_ref=s.solicitud_ref AND actuacion.empleado_ref=s.empleado_ref
      AND actuacion.version=1 AND actuacion.paso='solicitud' AND actuacion.estado='pendiente_responsable';
 EXCEPTION WHEN no_data_found OR too_many_rows THEN
   RAISE EXCEPTION 'historia Cronos incompatible' USING ERRCODE='PC003';
 END;
 SELECT count(*) INTO eventos FROM vec_cronos_v1.solicitud_outbox o
  WHERE o.agregado_ref=s.solicitud_ref AND o.empleado_ref=s.empleado_ref
    AND o.tipo='cronos.correccion.solicitada'
    AND o.carga_json->>'solicitud_ref'=s.solicitud_ref
    AND o.carga_json->>'recibo_ref'=a.recibo_ref
    AND o.carga_json->>'version'='1'
    AND o.carga_json->>'estado'='pendiente_responsable';
 IF eventos<>1 OR a.registrada_en IS DISTINCT FROM s.solicitada_en THEN
   RAISE EXCEPTION 'historia Cronos incompleta' USING ERRCODE='PC003';
 END IF;
 INSERT INTO vec_cronos_v1.correccion_recibo_acceso(decision_ref,empleado_ref,solicitud_ref,actuacion_ref,recibo_ref,
   version_recibo,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,s.empleado_ref,s.solicitud_ref,a.actuacion_ref,a.recibo_ref,
   1,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 IF clock_timestamp()>=k.vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 RETURN jsonb_build_object('solicitud_ref',s.solicitud_ref,'actuacion_ref',a.actuacion_ref,
   'recibo_ref',a.recibo_ref,'estado',a.estado,'version',a.version,'instante_utc',a.registrada_en,'replay',true);
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.material_recuperacion_correccion_propia_v1(text) FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.recuperar_recibo_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.recuperar_recibo_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_ejecutor;

-- La frontera conoce la ruta nueva para auditar 401/403 sin recurrir a 'otra'.
DO $ruta$
DECLARE d text; nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.denegacion_frontera'::regclass
    AND c.conname='denegacion_frontera_ruta_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex')<>'a6833e0396c7cd19faf0c076285b4a03209c146c114f3905e8d4d8a8f68661f9'
    OR right(d,3)<>']))' OR strpos(d,'/api/interna/cronos/correcciones/recibo')<>0
 THEN RAISE EXCEPTION 'CRN17: rutas de denegación incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', '||quote_literal('/api/interna/cronos/correcciones/recibo')||'::text]))';
 ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
 EXECUTE 'ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check '||nueva;
END $ruta$;

DO $final$
DECLARE f oid:='vec_cronos_v1.recuperar_recibo_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_cronos_v1_propietario'::regrole AND p.prosecdef
                 AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_cronos_v1_ejecutor'::regrole)
                 OR a.privilege_type<>'EXECUTE' OR (a.grantee='vec_cronos_v1_ejecutor'::regrole AND a.is_grantable)))
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid='vec_cronos_v1.material_recuperacion_correccion_propia_v1(text)'::regprocedure AND a.grantee<>p.proowner)
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_cronos_v1.correccion_recibo_acceso'::regclass AND c.relrowsecurity AND c.relforcerowsecurity AND c.relowner='vec_cronos_v1_propietario'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) acl
               WHERE c.oid='vec_cronos_v1.correccion_recibo_acceso'::regclass AND acl.grantee<>c.relowner)
    OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid='vec_cronos_v1.correccion_recibo_acceso'::regclass
         AND p.polroles=ARRAY[(SELECT oid FROM pg_roles WHERE rolname='vec_cronos_v1_propietario')])<>2
 THEN RAISE EXCEPTION 'CRN17: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $final$;
COMMIT;
