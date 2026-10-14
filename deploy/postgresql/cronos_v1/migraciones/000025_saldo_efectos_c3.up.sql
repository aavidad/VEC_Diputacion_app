\set ON_ERROR_STOP on
-- CRN25: política de efectos y fuente duradera del saldo C3b.
-- Después de CRN24 y del consumidor de publicación nominal en AD273.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:migracion:000025:saldo-efectos',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regrole('vec_cronos_v1_publicador_politicas') IS NOT NULL
  OR to_regclass('vec_cronos_v1.politica_efectos_saldo_c3') IS NOT NULL
  OR to_regclass('vec_cronos_v1.snapshot_saldo_efectos_c3') IS NOT NULL
  OR to_regprocedure('vec_cronos_v1.consultar_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR to_regprocedure('vec_cronos_v1.publicar_politica_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR to_regprocedure('vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regclass('vec_cronos_v1.programacion_contexto_c4') IS NULL
  OR to_regclass('vec_cronos_v1.permiso_resolucion') IS NULL
  OR to_regclass('vec_cronos_v1.permiso_estado') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_politica_efectos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR NOT coalesce(has_function_privilege('vec_cronos_v1_propietario',
      to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_politica_efectos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
      'EXECUTE'),false)
 THEN RAISE EXCEPTION 'CRN25: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_cronos_v1_publicador_politicas NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_cronos_v1_publicador_politicas;
DO $conectar$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_cronos_v1_publicador_politicas',current_database()); END $conectar$;
SET LOCAL ROLE vec_cronos_v1_propietario;

CREATE TABLE vec_cronos_v1.politica_efectos_saldo_c3 (
 politica_ref text NOT NULL CHECK(politica_ref ~ '^politica:cronos:efectos:[a-z0-9_:-]{3,128}$'),
 version integer NOT NULL CHECK(version BETWEEN 1 AND 2147483647),
 centro_ref text NOT NULL CHECK(centro_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 estado text NOT NULL CHECK(estado IN ('publicada','publicada_sintetica')),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 acto_evidencia_sha256 text CHECK(acto_evidencia_sha256 IS NULL OR acto_evidencia_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_url text NOT NULL CHECK(length(fuente_url) BETWEEN 9 AND 512 AND left(fuente_url,8)='https://'),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 hasta_exclusivo date NOT NULL CHECK(isfinite(hasta_exclusivo) AND hasta_exclusivo>vigente_desde),
 reglas jsonb NOT NULL CHECK(jsonb_typeof(reglas)='array' AND jsonb_array_length(reglas) BETWEEN 1 AND 100),
 sha256 text NOT NULL CHECK(sha256=encode(pg_catalog.sha256(convert_to(reglas::text,'UTF8')),'hex')),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:cronos:politica:[0-9a-f-]{36}$'),
 decision_ref text UNIQUE, auditoria_ref text UNIQUE, consumo_huella_sha256 text UNIQUE,
 PRIMARY KEY(politica_ref,version),
 CHECK((estado='publicada_sintetica' AND acto_evidencia_sha256 IS NULL AND decision_ref IS NULL AND auditoria_ref IS NULL AND consumo_huella_sha256 IS NULL)
    OR (estado='publicada' AND acto_evidencia_sha256 IS NOT NULL AND decision_ref IS NOT NULL AND auditoria_ref IS NOT NULL AND consumo_huella_sha256 IS NOT NULL))
);
CREATE INDEX politica_efectos_c3_centro_fecha_idx ON vec_cronos_v1.politica_efectos_saldo_c3(centro_ref,vigente_desde,hasta_exclusivo,version DESC);
ALTER TABLE vec_cronos_v1.politica_efectos_saldo_c3 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.politica_efectos_saldo_c3 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propietario ON vec_cronos_v1.politica_efectos_saldo_c3 FOR SELECT TO vec_cronos_v1_propietario USING(true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.politica_efectos_saldo_c3 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK(true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.politica_efectos_saldo_c3 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.politica_efectos_saldo_c3 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_publicador_politicas;
REVOKE ALL ON TYPE vec_cronos_v1.politica_efectos_saldo_c3 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_publicador_politicas;

CREATE TABLE vec_cronos_v1.snapshot_saldo_efectos_c3 (
 snapshot_ref text PRIMARY KEY CHECK(snapshot_ref ~ '^saldo:cronos:snapshot:[0-9a-f]{64}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 desde date NOT NULL CHECK(isfinite(desde)),
 hasta date NOT NULL CHECK(isfinite(hasta) AND hasta>=desde AND hasta-desde<=366),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 fuente jsonb NOT NULL CHECK(jsonb_typeof(fuente)='object' AND fuente_sha256=encode(pg_catalog.sha256(convert_to(fuente::text,'UTF8')),'hex')),
 primer_acceso_decision_ref text NOT NULL REFERENCES vec_cronos_v1.saldo_acceso(decision_ref),
 creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en)),
 UNIQUE(empleado_ref,desde,hasta,fuente_sha256)
);
CREATE INDEX snapshot_saldo_efectos_c3_empleado_idx ON vec_cronos_v1.snapshot_saldo_efectos_c3(empleado_ref,creada_en DESC);
ALTER TABLE vec_cronos_v1.snapshot_saldo_efectos_c3 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.snapshot_saldo_efectos_c3 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propia ON vec_cronos_v1.snapshot_saldo_efectos_c3 FOR SELECT TO vec_cronos_v1_propietario USING(empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.snapshot_saldo_efectos_c3 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK(empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.snapshot_saldo_efectos_c3 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.snapshot_saldo_efectos_c3 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_publicador_politicas;
REVOKE ALL ON TYPE vec_cronos_v1.snapshot_saldo_efectos_c3 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_publicador_politicas;

CREATE FUNCTION vec_cronos_v1.publicar_politica_saldo_efectos_c3_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb; d jsonb; consumo record; v integer; previo integer; ahora timestamptz(6);
 sha text; contexto_sha text; ref text; recibo text; regla jsonb;
BEGIN
 IF current_user<>'vec_cronos_v1_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members a WHERE a.member=session_user::regrole
       AND a.roleid='vec_cronos_v1_publicador_politicas'::regrole AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option)
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members a WHERE a.member=session_user::regrole)<>1
  OR has_table_privilege(session_user,'vec_cronos_v1.politica_efectos_saldo_c3','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  OR has_table_privilege(session_user,'vec_cronos_v1.snapshot_saldo_efectos_c3','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 100 AND 65536
  OR p_material IS NOT JSON OBJECT WITH UNIQUE KEYS
 THEN RAISE EXCEPTION 'CRN25: publicación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;v:=(m->>'version')::integer;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CRN25: material inválido' USING ERRCODE='22023'; END;
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
    ARRAY['acto_evidencia_sha256','acto_ref','centro_ref','estado','fuente_ref','fuente_url','hasta_exclusivo','politica_ref','reglas','sha256','version','vigente_desde']
  OR m->>'estado' IS DISTINCT FROM 'publicada'
  OR coalesce(m->>'politica_ref','') !~ '^politica:cronos:efectos:[a-z0-9_:-]{3,128}$'
  OR coalesce(m->>'centro_ref','') !~ '^[a-z][a-z0-9_:-]{2,127}$'
  OR coalesce(m->>'acto_ref','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
  OR coalesce(m->>'acto_evidencia_sha256','') !~ '^[0-9a-f]{64}$'
  OR coalesce(m->>'fuente_ref','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
  OR left(coalesce(m->>'fuente_url',''),8)<>'https://' OR length(m->>'fuente_url')>512
  OR v NOT BETWEEN 1 AND 2147483647 OR jsonb_typeof(m->'reglas') IS DISTINCT FROM 'array'
  OR jsonb_array_length(m->'reglas') NOT BETWEEN 1 AND 100
  OR coalesce(m->>'vigente_desde','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
  OR coalesce(m->>'hasta_exclusivo','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
  OR (m->>'vigente_desde')::date >= (m->>'hasta_exclusivo')::date
 THEN RAISE EXCEPTION 'CRN25: política inválida' USING ERRCODE='22023'; END IF;
 sha:=encode(pg_catalog.sha256(convert_to((m->'reglas')::text,'UTF8')),'hex');
 IF m->>'sha256' IS DISTINCT FROM sha THEN RAISE EXCEPTION 'CRN25: huella divergente' USING ERRCODE='22023'; END IF;
 FOR regla IN SELECT value FROM jsonb_array_elements(m->'reglas') LOOP
  IF jsonb_typeof(regla) IS DISTINCT FROM 'object'
   OR ARRAY(SELECT jsonb_object_keys(regla) ORDER BY 1) IS DISTINCT FROM
      ARRAY['catalogo_version_ref','centro_ref','colectivo_ref','efecto','fuente_consultada_en','fuente_publicada_en','fuente_ref','fuente_url','fuente_version_ref','hasta_exclusivo','permiso_ref','referencia','vigente_desde']
   OR regla->>'centro_ref' IS DISTINCT FROM m->>'centro_ref'
   OR regla->>'efecto' IS DISTINCT FROM 'credito_tiempo_autorizado'
   OR coalesce(regla->>'referencia','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR coalesce(regla->>'colectivo_ref','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR coalesce(regla->>'fuente_ref','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR coalesce(regla->>'fuente_version_ref','') !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR left(coalesce(regla->>'fuente_url',''),8)<>'https://'
   OR coalesce(regla->>'fuente_publicada_en','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
   OR coalesce(regla->>'fuente_consultada_en','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
   OR (regla->>'fuente_publicada_en')::date>(regla->>'fuente_consultada_en')::date
   OR (regla->>'fuente_consultada_en')::date>current_date
   OR coalesce(regla->>'vigente_desde','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
   OR coalesce(regla->>'hasta_exclusivo','') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
   OR (regla->>'vigente_desde')::date >= (regla->>'hasta_exclusivo')::date
   OR (regla->>'vigente_desde')::date < (m->>'vigente_desde')::date
   OR (regla->>'hasta_exclusivo')::date > (m->>'hasta_exclusivo')::date
  THEN RAISE EXCEPTION 'CRN25: regla incompatible' USING ERRCODE='22023'; END IF;
 END LOOP;
 ref:=m->>'politica_ref'||':v'||v::text;
 contexto_sha:=encode(pg_catalog.sha256(convert_to('{"ambitos":{"centro_ref":'||to_jsonb(m->>'centro_ref')::text||
  '},"atributos":{"acto_evidencia_sha256":"'||(m->>'acto_evidencia_sha256')||'","material_sha256":"'||
  encode(pg_catalog.sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'cronos'
  OR d->>'accion' IS DISTINCT FROM 'cronos.politica_efectos.publicar'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'politica_efectos'
  OR d->>'finalidad' IS DISTINCT FROM 'publicar_politica_efectos'
  OR d->>'recurso_ref' IS DISTINCT FROM ref
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
  OR d->'campos_permitidos' IS DISTINCT FROM
     '["acto_evidencia_sha256","acto_ref","centro_ref","estado","fuente_ref","fuente_url","hasta_exclusivo","politica_ref","reglas","sha256","version","vigente_desde"]'::jsonb
  OR d->'obligaciones' IS DISTINCT FROM '["acto_doble_control"]'::jsonb
 THEN RAISE EXCEPTION 'CRN25: decisión incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_politica_efectos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM ref OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 THEN RAISE EXCEPTION 'CRN25: consumo divergente' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(m->'reglas') AS regla(valor)
   WHERE NOT EXISTS(SELECT 1 FROM vec_cronos_v1.permiso_catalogo c
     WHERE c.version_ref=regla.valor->>'catalogo_version_ref' AND c.permiso_ref=regla.valor->>'permiso_ref'))
 THEN RAISE EXCEPTION 'CRN25: permiso de catálogo no publicado' USING ERRCODE='22023'; END IF;
 SELECT coalesce(max(p.version),0) INTO previo FROM vec_cronos_v1.politica_efectos_saldo_c3 p WHERE p.politica_ref=m->>'politica_ref';
 IF v<>previo+1 THEN RAISE EXCEPTION 'CRN25: versión incompatible' USING ERRCODE='40001'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());recibo:='recibo:cronos:politica:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.politica_efectos_saldo_c3
  (politica_ref,version,centro_ref,estado,acto_ref,acto_evidencia_sha256,fuente_ref,fuente_url,vigente_desde,hasta_exclusivo,
   reglas,sha256,publicada_en,recibo_ref,decision_ref,auditoria_ref,consumo_huella_sha256)
 VALUES(m->>'politica_ref',v,m->>'centro_ref','publicada',m->>'acto_ref',m->>'acto_evidencia_sha256',m->>'fuente_ref',m->>'fuente_url',
   (m->>'vigente_desde')::date,(m->>'hasta_exclusivo')::date,m->'reglas',sha,ahora,recibo,
   consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256);
 RETURN jsonb_build_object('recibo_ref',recibo,'politica_ref',m->>'politica_ref','version',v,
  'sha256',sha,'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,'publicada_en',ahora);
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.publicar_politica_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.publicar_politica_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_publicador_politicas;

CREATE FUNCTION vec_cronos_v1.consultar_saldo_efectos_c3_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb; d jsonb; saldo jsonb; programas jsonb; permisos jsonb; politicas jsonb; fuente jsonb;
 acceso vec_cronos_v1.saldo_acceso%ROWTYPE; empleado text; desde date; hasta date;
 suma text; snapshot text; previo jsonb;
BEGIN
 -- La fachada C3 consume exactamente una decisión de saldo, registra su
 -- auditoría y deja la RLS de esta transacción ligada al empleado.
 saldo:=vec_cronos_v1.consultar_saldo_propio_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 empleado:=m->>'empleado_ref';desde:=(m->>'desde')::date;hasta:=(m->>'hasta')::date;
 IF empleado IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'')
  OR saldo->>'empleado_ref' IS DISTINCT FROM empleado
  OR saldo->>'desde' IS DISTINCT FROM desde::text OR saldo->>'hasta' IS DISTINCT FROM hasta::text
  OR jsonb_typeof(saldo->'jornadas') IS DISTINCT FROM 'array'
  OR d->>'decision_ref' IS NULL
 THEN RAISE EXCEPTION 'CRN25: saldo divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT acceso FROM vec_cronos_v1.saldo_acceso a WHERE a.decision_ref=d->>'decision_ref';
 IF acceso.empleado_ref IS DISTINCT FROM empleado OR acceso.desde IS DISTINCT FROM desde OR acceso.hasta IS DISTINCT FROM hasta
  OR acceso.material_sha256 IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(p_material,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CRN25: acceso divergente' USING ERRCODE='42501'; END IF;

 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'fecha',j->>'fecha','referencia',c.programacion_ref,'fuente_ref',j->>'fuente_ref',
  'version',c.version,'minutos',(j->>'minutos_previstos')::integer,
  'inicio_utc',c.inicio_utc,'fin_utc',c.fin_utc,'centro_ref',c.centro_ref,'colectivo_ref',c.regimen_ref)
  ORDER BY j->>'fecha'),'[]'::jsonb) INTO programas
 FROM jsonb_array_elements(saldo->'jornadas') j
 JOIN vec_cronos_v1.programacion_contexto_c4 c ON c.programacion_ref=j->>'programacion_ref'
  AND c.empleado_ref=empleado AND c.fecha=(j->>'fecha')::date AND c.version=(j->>'version')::integer
  AND c.fuente_ref=j->>'fuente_ref' AND c.publicada_en<=acceso.consultada_en;

 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'solicitud_ref',s.solicitud_ref,'resolucion_ref',coalesce(r.resolucion_ref,''),
   'resolucion_version',coalesce(r.version_resultante,0),'permiso_ref',s.permiso_ref,
   'catalogo_version_ref',s.catalogo_version_ref,'desde',s.desde,'hasta',s.hasta,
   'hora_inicio',coalesce(s.hora_inicio,''),'hora_fin',coalesce(s.hora_fin,''),
   'cantidad_minutos',s.cantidad,'resolucion_decision_ref',coalesce(r.decision_ref,''),
   'resolucion_auditoria_ref',coalesce(r.auditoria_ref,''),'resuelta_en',r.registrada_en,
   'catalogo_fuente_ref',pc.fuente_ref,'catalogo_publicada_en',pc.publicada_en,
   'pendiente_justificar',e.pendiente_justificar)
   ORDER BY s.desde,s.solicitud_ref),'[]'::jsonb)
 INTO permisos FROM vec_cronos_v1.permiso_solicitud s
 JOIN vec_cronos_v1.permiso_catalogo pc ON pc.version_ref=s.catalogo_version_ref AND pc.permiso_ref=s.permiso_ref
 JOIN LATERAL (SELECT e.* FROM vec_cronos_v1.permiso_estado e
   WHERE e.solicitud_ref=s.solicitud_ref ORDER BY e.version DESC LIMIT 1) e ON true
 LEFT JOIN vec_cronos_v1.permiso_resolucion r ON r.solicitud_ref=s.solicitud_ref AND r.version_resultante=e.version
  AND r.estado_resultante='concedido' AND r.empleado_ref=s.empleado_ref
 WHERE s.empleado_ref=empleado AND e.empleado_ref=empleado AND e.estado='concedido'
  AND s.desde<=hasta AND s.hasta>=desde;
 IF EXISTS(SELECT 1 FROM vec_cronos_v1.permiso_solicitud s
   WHERE s.empleado_ref=empleado AND s.desde<=hasta AND s.hasta>=desde
     AND NOT EXISTS(SELECT 1 FROM vec_cronos_v1.permiso_estado e
       WHERE e.solicitud_ref=s.solicitud_ref AND e.empleado_ref=empleado))
 THEN RAISE EXCEPTION 'CRN25: historia de permiso incompleta' USING ERRCODE='55000'; END IF;

 SELECT coalesce(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,'version',p.version,'sha256',p.sha256,
   'acto_ref',p.acto_ref,'publicada_en',p.publicada_en,'reglas',p.reglas)
   ORDER BY p.politica_ref,p.version),'[]'::jsonb) INTO politicas
 FROM vec_cronos_v1.politica_efectos_saldo_c3 p
 WHERE p.publicada_en<=acceso.consultada_en AND p.vigente_desde<=hasta AND p.hasta_exclusivo>desde
  AND p.centro_ref IN (SELECT DISTINCT x.valor->>'centro_ref' FROM jsonb_array_elements(programas) AS x(valor));

 fuente:=jsonb_build_object('saldo',saldo,'programaciones',programas,'permisos',permisos,
  'politicas',politicas,'cobertura_permisos',true);
 IF octet_length(fuente::text)>4194304 THEN
  RAISE EXCEPTION 'CRN25: fuente excede el límite' USING ERRCODE='54000'; END IF;
 suma:=encode(pg_catalog.sha256(convert_to(fuente::text,'UTF8')),'hex');
 snapshot:='saldo:cronos:snapshot:'||suma;
 INSERT INTO vec_cronos_v1.snapshot_saldo_efectos_c3
  (snapshot_ref,empleado_ref,desde,hasta,fuente_sha256,fuente,primer_acceso_decision_ref,creada_en)
 VALUES(snapshot,empleado,desde,hasta,suma,fuente,acceso.decision_ref,acceso.consultada_en)
 ON CONFLICT(snapshot_ref) DO NOTHING;
 SELECT s.fuente INTO STRICT previo FROM vec_cronos_v1.snapshot_saldo_efectos_c3 s WHERE s.snapshot_ref=snapshot AND s.empleado_ref=empleado;
 IF previo IS DISTINCT FROM fuente THEN RAISE EXCEPTION 'CRN25: snapshot divergente' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('fuente_canonica',fuente::text,'snapshot_ref',snapshot,'fuente_sha256',suma,'corte_utc',acceso.consultada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_publicador_politicas;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.consultar_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
COMMIT;
