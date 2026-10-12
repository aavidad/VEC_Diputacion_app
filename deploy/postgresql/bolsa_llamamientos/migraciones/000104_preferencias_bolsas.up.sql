\set ON_ERROR_STOP on
-- B104. Base de historia propia para preferencias ENTRE bolsas. No modifica
-- puntuaciones, orden de bolsa, disponibilidad, ofertas ni llamamientos.
-- Se instala después de AD241, AUT76 y AD246; la lectura nominal se audita
-- en la misma transacción sin consumir una decisión V3.
-- Las fachadas V3 de escritura se completan en este mismo
-- corte antes de instalar la migración en cualquier base conservada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000104',0));

DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(bytea,bytea,jsonb)') IS NULL
    OR to_regrole('vec_bolsa_llamamientos_portal_externo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.preferencia_bolsas_version') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.preferencia_bolsas_outbox') IS NOT NULL
 THEN RAISE EXCEPTION 'B104: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La frontera externa conserva la denegación nominal también en la ruta
-- nueva. Se amplían únicamente su CHECK y su guarda de rutas, desde la
-- definición B77 instalada; propietario, ACL y configuración son idénticos.
DO $frontera$
DECLARE ruta_nueva text:='/api/vec/bolsa/mi-bolsa/preferencias';
 tabla regclass:='vec_bolsa_llamamientos.denegacion_frontera_portal_externo'::regclass;
 firma regprocedure:='vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::regprocedure;
 marca text:='''/api/vec/bolsa/mi-bolsa/solicitudes-documentales''';
 nombre text; expresion text; expresion_nueva text; total integer;
 anterior text; nuevo text; actual text; acl aclitem[]; propietario oid; config text[]; definidora boolean;
BEGIN
 SELECT count(*),min(c.conname),min(pg_get_expr(c.conbin,c.conrelid))
 INTO total,nombre,expresion FROM pg_constraint c WHERE c.conrelid=tabla AND c.contype='c'
  AND c.convalidated AND pg_get_constraintdef(c.oid) LIKE '%/api/vec/bolsa/mi-bolsa/solicitudes-documentales%';
 IF total<>1 OR expresion IS NULL OR strpos(expresion,ruta_nueva)<>0 THEN
  RAISE EXCEPTION 'PARO clave=B104.frontera_check, actual=%/%, esperado=1/false',
   total,coalesce(strpos(expresion,ruta_nueva)<>0,false) USING ERRCODE='55000'; END IF;
 EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo DROP CONSTRAINT %I',nombre);
 EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo ADD CONSTRAINT %I CHECK ((%s) OR ruta=%L)',
  nombre,expresion,ruta_nueva);
 SELECT pg_get_expr(c.conbin,c.conrelid) INTO STRICT expresion_nueva FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname=nombre AND c.convalidated;
 IF strpos(expresion_nueva,ruta_nueva)=0 OR strpos(expresion_nueva,'/api/vec/bolsa/mi-bolsa/solicitudes-documentales')=0
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity) THEN
  RAISE EXCEPTION 'PARO clave=B104.frontera_postimagen, actual=%, esperado=ambas rutas/RLS',
   expresion_nueva USING ERRCODE='55000'; END IF;
 SELECT p.prosrc,p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT anterior,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=firma;
 IF (length(anterior)-length(replace(anterior,marca,'')))/length(marca)<>1
    OR strpos(anterior,ruta_nueva)<>0 OR NOT definidora
    OR propietario<>'vec_bolsa_llamamientos_propietario'::regrole
    OR encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
     '0ceeb298f5505da8cfac40cad1f9733359d904813cfe4156db4122bc601618e6'
    OR ('search_path=pg_catalog, pg_temp'=ANY(config)) IS NOT TRUE THEN
  RAISE EXCEPTION 'PARO clave=B104.frontera_funcion, actual=%, esperado=%',
   encode(sha256(convert_to(anterior,'UTF8')),'hex'),
   '0ceeb298f5505da8cfac40cad1f9733359d904813cfe4156db4122bc601618e6'
   USING ERRCODE='55000'; END IF;
END $frontera$;

-- Definición final literal tomada de la postimagen B104 ensayada en PG18.
-- PostgreSQL conserva propietario y ACL al reemplazar la función.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(p_correlacion text, p_motivo text, p_superficie text, p_ruta text, p_actor text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '2s'
AS $function$
DECLARE login pg_catalog.pg_roles%ROWTYPE; grupo pg_catalog.pg_roles%ROWTYPE;
        fachada oid := 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
BEGIN
 SELECT * INTO login FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO grupo FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_portal_externo';
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR login.oid IS NULL OR NOT login.rolcanlogin OR NOT login.rolinherit
    OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole OR login.rolreplication OR login.rolbypassrls
    OR grupo.oid IS NULL OR grupo.rolcanlogin OR NOT grupo.rolinherit
    OR grupo.rolsuper OR grupo.rolcreatedb OR grupo.rolcreaterole OR grupo.rolreplication OR grupo.rolbypassrls
    OR (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid
      AND m.roleid=grupo.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=login.oid OR m.member=grupo.oid)
    OR NOT pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','USAGE')
    OR pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','CREATE')
    OR NOT pg_catalog.has_function_privilege(login.oid,fachada,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=fachada
       AND a.privilege_type='EXECUTE' AND
         (a.grantee=0 OR (a.grantee=grupo.oid AND a.is_grantable)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND n.nspname<>'vec_bolsa_llamamientos'
      AND pg_catalog.has_schema_privilege(login.oid,n.oid,'USAGE,CREATE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND (n.nspowner IN (login.oid,grupo.oid) OR EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=login.oid
          OR (a.grantee=grupo.oid AND (n.nspname<>'vec_bolsa_llamamientos' OR a.privilege_type<>'USAGE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND (c.relowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'
      AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee IN (login.oid,grupo.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND (p.proowner IN (login.oid,grupo.oid)
       OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=login.oid
         OR (a.grantee=grupo.oid AND (p.oid<>fachada OR a.privilege_type<>'EXECUTE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname LIKE 'vec_%' AND (t.typowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND p.oid<>fachada AND pg_catalog.has_function_privilege(login.oid,p.oid,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind IN ('r','p','v','m','f') THEN
        pg_catalog.has_table_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
        OR pg_catalog.has_any_column_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES') ELSE false END)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind='S' THEN
       pg_catalog.has_sequence_privilege(login.oid,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
    OR pg_catalog.pg_is_in_recovery() OR pg_catalog.current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Bolsa: registrador exterior invalido' USING ERRCODE='42501'; END IF;
 IF p_correlacion IS NULL OR (p_correlacion<>'corr_no_disponible' AND p_correlacion !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado')
    OR p_superficie IS DISTINCT FROM 'api.bolsa.candidato.ruta_exacta'
    OR p_ruta IS NULL OR p_ruta NOT IN (
      '/api/vec/bolsa/mi-bolsa','/api/vec/bolsa/mi-bolsa/historial',
      '/api/vec/bolsa/mi-bolsa/solicitudes','/api/vec/bolsa/mi-bolsa/respuestas',
      '/api/vec/bolsa/mi-bolsa/disposiciones','/api/vec/bolsa/mi-bolsa/contacto','/api/vec/bolsa/mi-bolsa/solicitudes-documentales','/api/vec/bolsa/mi-bolsa/preferencias')
    OR (p_motivo='autenticacion_requerida' AND p_actor IS NOT NULL)
    OR (p_motivo='acceso_denegado' AND (p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'))
 THEN RAISE EXCEPTION 'Bolsa: denegacion exterior invalida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.denegacion_frontera_portal_externo
  (correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
 VALUES (p_correlacion,p_motivo,p_superficie,p_ruta,p_actor,
  pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()));
 RETURN true;
END $function$;
DO $frontera_post$
DECLARE f regprocedure:='vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::regprocedure;
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
 OR NOT p.prosecdef OR p.proconfig IS NULL
 OR NOT ('search_path=pg_catalog, pg_temp'=ANY(p.proconfig))
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  'c566dc322bd2af3326d57675b64430e455a783a58dc454c081dc4423ebb4cac2'
 OR strpos(p.prosrc,'/api/vec/bolsa/mi-bolsa/preferencias')=0
 THEN RAISE EXCEPTION 'PARO clave=B104.frontera_funcion_post, actual=%, esperado=%',
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),
  'c566dc322bd2af3326d57675b64430e455a783a58dc454c081dc4423ebb4cac2'
  USING ERRCODE='55000';END IF;
END $frontera_post$;


-- Solo adición. El último número de versión es la preferencia vigente; la
-- clave y huella conservan la semántica de reintento tras respuesta incierta.
CREATE TABLE vec_bolsa_llamamientos.preferencia_bolsas_version (
 candidato_ref text NOT NULL CHECK (candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK (version BETWEEN 1 AND 2147483647),
 orden text[] NOT NULL CHECK (array_ndims(orden)=1 AND cardinality(orden) BETWEEN 1 AND 256 AND array_position(orden,NULL) IS NULL),
 conjunto_huella_sha256 text NOT NULL CHECK (conjunto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 clave_idempotencia text NOT NULL CHECK (clave_idempotencia ~ '^[A-Za-z0-9._:-]{8,128}$'),
 solicitud_huella_sha256 text NOT NULL CHECK (solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:preferencias-bolsas:[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE CHECK (octet_length(decision_ref) BETWEEN 1 AND 256),
 auditoria_ref text NOT NULL CHECK (octet_length(auditoria_ref) BETWEEN 1 AND 256),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (candidato_ref,version),
 UNIQUE (candidato_ref,clave_idempotencia)
);
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version FORCE ROW LEVEL SECURITY;
CREATE POLICY preferencia_bolsas_version_propietario ON vec_bolsa_llamamientos.preferencia_bolsas_version
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.preferencia_bolsas_version FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.preferencia_bolsas_version FROM PUBLIC;
CREATE TRIGGER preferencia_bolsas_version_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.preferencia_bolsas_version
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- La versión y su evento se insertan en la misma transacción que el consumo
-- autorizado y la auditoría. El replay devuelve el recibo sin otro evento.
CREATE TABLE vec_bolsa_llamamientos.preferencia_bolsas_outbox (
 evento_ref text PRIMARY KEY CHECK(evento_ref ~ '^evento:preferencias-bolsas:[0-9a-f]{64}$'),
 candidato_ref text NOT NULL,
 version bigint NOT NULL,
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 esquema text NOT NULL CHECK(esquema='vec.bolsa.preferencias.ordenadas.v1'),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(candidato_ref,version),
 FOREIGN KEY(candidato_ref,version)
  REFERENCES vec_bolsa_llamamientos.preferencia_bolsas_version(candidato_ref,version),
 FOREIGN KEY(recibo_ref) REFERENCES vec_bolsa_llamamientos.preferencia_bolsas_version(recibo_ref)
);
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY preferencia_bolsas_outbox_propietario ON vec_bolsa_llamamientos.preferencia_bolsas_outbox
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.preferencia_bolsas_outbox FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.preferencia_bolsas_outbox FROM PUBLIC;
CREATE TRIGGER preferencia_bolsas_outbox_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.preferencia_bolsas_outbox
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER preferencia_bolsas_outbox_no_truncar
 BEFORE TRUNCATE ON vec_bolsa_llamamientos.preferencia_bolsas_outbox
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Un candidato puede conservar participaciones históricas de una bolsa.
-- Solo se ordena la constitución más reciente aún vigente o suspendida, en
-- su intervalo temporal; la bolsa extinguida pasa a 'retirada' en la lectura.
-- Esta función es privada al propietario, nunca una fachada de lectura.
CREATE FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(
 p_candidato_ref text,p_corte timestamptz)
RETURNS text[] LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_bolsas text[];
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_corte IS NULL OR NOT isfinite(p_corte)
 THEN RAISE EXCEPTION 'B104: consulta interna inválida' USING ERRCODE='22023'; END IF;
 WITH ultima AS (
  SELECT DISTINCT ON (p.bolsa_ref) p.bolsa_ref,p.estado,p.vigente_desde,p.vigente_hasta
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) p
   ORDER BY p.bolsa_ref,p.confirmada_en DESC,p.participacion_ref DESC
 )
 SELECT coalesce(array_agg(bolsa_ref ORDER BY bolsa_ref),ARRAY[]::text[])
 INTO v_bolsas FROM ultima
 WHERE estado IN ('vigente','suspendida') AND vigente_desde<=p_corte
   AND (vigente_hasta IS NULL OR vigente_hasta>p_corte);
 RETURN v_bolsas;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(text,timestamptz) FROM PUBLIC;

-- Proyección privada de la versión escrita y de las diferencias actuales.
-- Nunca añade una bolsa al orden guardado por inferencia.
CREATE FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(
 p_candidato_ref text,p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_actuales text[]; v_orden text[]; v_nuevas text[]; v_retiradas text[];
 v_version bigint:=0; v_recibo text; v_fecha timestamptz;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'B104: lectura interna denegada' USING ERRCODE='42501';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,p_corte);
 SELECT p.version,p.orden,p.recibo_ref,p.registrada_en
 INTO v_version,v_orden,v_recibo,v_fecha
 FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref ORDER BY p.version DESC LIMIT 1;
 v_version:=coalesce(v_version,0);
 v_orden:=coalesce(v_orden,ARRAY[]::text[]);
 SELECT coalesce(array_agg(u.ref ORDER BY u.ref),ARRAY[]::text[]) INTO v_nuevas
 FROM unnest(v_actuales) AS u(ref) WHERE NOT u.ref=ANY(v_orden);
 SELECT coalesce(array_agg(u.ref ORDER BY u.ref),ARRAY[]::text[]) INTO v_retiradas
 FROM unnest(v_orden) AS u(ref) WHERE NOT u.ref=ANY(v_actuales);
 RETURN jsonb_build_object('version',v_version,'orden_guardado',to_jsonb(v_orden),
  'bolsas_actuales',to_jsonb(v_actuales),'bolsas_nuevas',to_jsonb(v_nuevas),
  'bolsas_retiradas',to_jsonb(v_retiradas),
  'revision_requerida',v_version=0 OR cardinality(v_nuevas)<>0 OR cardinality(v_retiradas)<>0,
  'recibo_ref',v_recibo,'registrado_en',v_fecha);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(text,timestamptz) FROM PUBLIC;

-- La proyección privada no tiene GRANT exterior. Esta fachada exige evidencia
-- nominal de la sesión externa y asienta la lectura positiva antes de devolver
-- datos. El adaptador debe confirmar COMMIT antes de responder por HTTP.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(
 p_candidato_ref text,p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_estado jsonb; v_auditoria record; v_corte timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_externo_bolsa_desarrollo'
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B104: lectura nominal inválida' USING ERRCODE='42501'; END IF;
 v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(p_candidato_ref,v_corte);
 SELECT * INTO STRICT v_auditoria
 FROM vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF v_estado IS NULL OR jsonb_typeof(v_estado) IS DISTINCT FROM 'object'
    OR v_auditoria.auditoria_ref IS NULL OR v_auditoria.huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_auditoria.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
    OR v_auditoria.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B104: lectura no asentada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',v_estado,'auditoria_ref',v_auditoria.auditoria_ref,
  'auditoria_huella_sha256',v_auditoria.huella_sha256,
  'correlacion_ref',v_auditoria.correlacion_ref,'auditada_en',v_auditoria.registrada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_portal_externo;

-- Fallo posterior a la autenticación. La misma sesión LOGIN entra por la
-- fachada Bolsa; AD246 acredita CA/AUT y registra el intento nominal común.
-- No concede al LOGIN acceso directo a AD3 ni consulta la concesión del GET.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(
 p_candidato_ref text,p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_contexto jsonb;v_candidato text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_externo_bolsa_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  'mi-bolsa:'||encode(sha256(convert_to('mi-bolsa:'||p_candidato_ref,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B104: intento nominal inválido' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'B104: contexto inválido' USING ERRCODE='42501'; END;
 SELECT x->>'referencia' INTO v_candidato FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF (SELECT count(*) FROM jsonb_array_elements(v_contexto->'vinculos') x
     WHERE x->>'tipo'='candidato' AND x->>'estado'='activo')<>1
 OR v_candidato IS DISTINCT FROM p_candidato_ref
 THEN RAISE EXCEPTION 'B104: candidato de intento denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en
 FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden) a;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(text,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_portal_externo;

-- Fachada de efecto. AD241 verifica la concesión vigente, el contexto y el
-- material atestado y registra su auditoría dentro de esta transacción. Una
-- autorización nueva se exige también para recuperar un recibo por clave.
-- p_registrada_en es solo una marca de frescura de la petición. La fecha
-- conservada de la versión y del outbox procede de clock_timestamp() en PG.
CREATE FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 p_candidato_ref text,p_orden text[],p_version_esperada bigint,p_clave text,p_registrada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,orden text[],registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_actuales text[]; v_ordenado text[];
 v_previa vec_bolsa_llamamientos.preferencia_bolsas_version%ROWTYPE;
 v_consumo record; v_version bigint; v_huella text; v_conjunto text; v_recibo text;
 v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_orden IS NULL OR array_ndims(p_orden) IS DISTINCT FROM 1 OR cardinality(p_orden) NOT BETWEEN 1 AND 256
    OR array_position(p_orden,NULL) IS NOT NULL
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
    OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_registrada_en < v_ahora-interval '5 minutes' OR p_registrada_en > v_ahora+interval '1 minute'
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B104: orden inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B104: material inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B104: orden denegado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'B104: candidato denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B104: consumo denegado' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias_bolsas:'||p_candidato_ref,0));
 IF EXISTS (SELECT 1 FROM unnest(p_orden) AS u(ref)
  WHERE u.ref IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(u.ref,512)) THEN
  RAISE EXCEPTION 'B104: referencia inválida' USING ERRCODE='22023'; END IF;
 -- La huella usa separador de control prohibido en las referencias canónicas.
 v_huella:=encode(sha256(convert_to(p_candidato_ref||chr(31)||p_version_esperada::text||chr(31)||
  array_to_string(p_orden,chr(31)),'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref AND p.clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previa.solicitud_huella_sha256<>v_huella OR v_previa.version<>p_version_esperada+1
  THEN RAISE EXCEPTION 'B104: clave reutilizada con otro orden' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.orden,v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B104: versión de preferencias desactualizada' USING ERRCODE='VBP01';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,v_ahora);
 IF cardinality(v_actuales)<>cardinality(p_orden) THEN
  RAISE EXCEPTION 'B104: conjunto de bolsas cambiado' USING ERRCODE='VBP03'; END IF;
 SELECT array_agg(u.ref ORDER BY u.ref) INTO v_ordenado FROM unnest(p_orden) AS u(ref);
 IF v_ordenado IS DISTINCT FROM v_actuales THEN
  RAISE EXCEPTION 'B104: orden incompleto, duplicado o ajeno' USING ERRCODE='VBP03'; END IF;
 v_conjunto:=encode(sha256(convert_to(array_to_string(v_actuales,chr(31)),'UTF8')),'hex');
 v_recibo:='recibo:preferencias-bolsas:'||encode(sha256(convert_to(
  p_candidato_ref||chr(31)||p_clave||chr(31)||v_huella,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(p_candidato_ref,v_version+1,p_orden,v_conjunto,p_clave,v_huella,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_outbox(
  evento_ref,candidato_ref,version,recibo_ref,decision_ref,auditoria_ref,esquema,evento,creada_en)
 VALUES('evento:preferencias-bolsas:'||substr(v_recibo,length('recibo:preferencias-bolsas:')+1),
  p_candidato_ref,v_version+1,v_recibo,v_consumo.decision_ref,v_consumo.auditoria_ref,
  'vec.bolsa.preferencias.ordenadas.v1',
  jsonb_build_object('candidato_ref',p_candidato_ref,'version',v_version+1,
   'orden',p_orden,'conjunto_huella_sha256',v_conjunto,'recibo_ref',v_recibo,
   'decision_ref',v_consumo.decision_ref,'auditoria_ref',v_consumo.auditoria_ref),v_ahora);
 RETURN QUERY SELECT false,v_recibo,v_version+1,p_orden,v_ahora;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_portal_externo;
COMMIT;
