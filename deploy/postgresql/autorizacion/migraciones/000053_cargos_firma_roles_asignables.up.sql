\set ON_ERROR_STOP on
-- AUT53: publicación gobernada de las versiones de rol de los cargos del plan
-- nominal de firma de Contratación temporal y su registro como perfiles
-- asignables (clase ordinario) en una sola operación técnica.
--
-- Un operador con LOGIN propio, miembro único del grupo NOLOGIN
-- vec_admin_cargos_firma_ejecutor, presenta un plan privado cuya huella aprobó
-- dirección y que el DBA ligó a ese LOGIN en config_cargos_firma_admin_v1.
-- Por cada cargo se publica una versión de rol ordinaria (versión 1, o la
-- siguiente con CAS sobre la huella de la anterior), su control de vigencia
-- habilitado y su fila en rol_administrable_exacto_v1, validada con la misma
-- defensa de AUT49 (validar_perfil_asignable_admin_v1). Las concesiones son
-- cerradas: la competencial de cada paso (AUT32 la exige sin campos ni
-- obligaciones) y, si el plan las pide, las dos de la firma V2 tal como las
-- consume AD162. Nada se concede a administración, Sistemas ni Intervención.
--
-- La auditoría común reutiliza la familia de AD196 (registro e intento de
-- perfiles asignables) sin tocar el núcleo ni el CHECK de autorizacion_atestada_v3,
-- y cada registro común de esta operación se anota además en
-- operacion_cargos_firma_admin_v1 por auditoria_ref (patrón de AD198).
-- Instalar esta estructura no publica ningún rol ni concede permisos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN RAISE EXCEPTION 'AUT53: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regclass('vec_autorizacion.rol_administrable_exacto_v1') IS NULL OR to_regclass('vec_autorizacion.config_perfiles_asignables_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb,boolean)') IS NULL
 OR to_regprocedure('vec_autorizacion.canon_version_rol_admin_v1(jsonb)') IS NULL OR to_regprocedure('vec_autorizacion.canon_control_rol_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.rechazar_mutacion_inmutable()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR to_regclass('vec_autorizacion.config_cargos_firma_admin_v1') IS NOT NULL OR to_regrole('vec_admin_cargos_firma_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT53: PARO clave=dependencias actual=divergente esperado=AUT24_AUT49_AD196_sin_AUT53' USING ERRCODE='55000'; END IF;
END $pre$;
-- La defensa de clases que se reutiliza debe ser exactamente la de AUT49.
DO $guarda$ DECLARE actual text;BEGIN
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb,boolean)') AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef;
 IF actual IS DISTINCT FROM '4ca98d16e7d1b7cc154871a939879ee24f26385cf7d234f15681c24169eccd96' THEN RAISE EXCEPTION 'AUT53: PARO clave=validar_perfil_asignable_admin_v1.prosrc actual=% esperado=4ca98d16e7d1b7cc154871a939879ee24f26385cf7d234f15681c24169eccd96',COALESCE(actual,'ausente') USING ERRCODE='55000'; END IF;
END $guarda$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Una fila por LOGIN técnico, escrita por el DBA tras la aprobación externa.
CREATE TABLE vec_autorizacion.config_cargos_firma_admin_v1(
 login_nombre name PRIMARY KEY,
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 entorno text NOT NULL CHECK(entorno='desarrollo'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde AND vigente_hasta-vigente_desde<=interval '1 day')
);
-- Historia de la operación: plan exacto, recibo y auditoría. Sirve al replay.
CREATE TABLE vec_autorizacion.registro_cargos_firma_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^rpa_cf_[A-Za-z0-9_-]{19,121}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 plan bytea NOT NULL CHECK(octet_length(plan) BETWEEN 1 AND 65536 AND encode(sha256(plan),'hex')=plan_sha256),
 login_nombre name NOT NULL,
 auditoria_ref text NOT NULL,
 recibo jsonb NOT NULL CHECK(jsonb_typeof(recibo)='object'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
-- Cada registro común (confirmación e intentos) que deja esta operación con
-- la familia de AD196. Permite distinguirlos de los de AUT49 por auditoria_ref.
CREATE TABLE vec_autorizacion.operacion_cargos_firma_admin_v1(
 auditoria_ref text PRIMARY KEY CHECK(auditoria_ref ~ '^aud_v3_pai?_[0-9a-f]{32}$'),
 tipo text NOT NULL CHECK(tipo IN('confirmacion','intento')),
 login_nombre name NOT NULL,
 solicitud_sha256 text NOT NULL CHECK(solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 resultado text NOT NULL CHECK(resultado IN('permitido','denegado','error')),
 operacion_ref text CHECK(operacion_ref IS NULL OR operacion_ref ~ '^rpa_cf_[A-Za-z0-9_-]{19,121}$'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
-- Regla de quién asigna cada cargo (pregunta 143 abierta). Es configuración:
-- el plan elige una regla por cargo y aquí se traduce a la audiencia que
-- consumirá la asignación. Hoy sólo existe el lote ordinario de Administración
-- (AD190/AUT44). Otra regla (p. ej. doble control) se añade con otra fila
-- cuando exista su circuito; las filas no se cambian ni se borran.
CREATE TABLE vec_autorizacion.regla_asignacion_cargo_firma_v1(
 regla text PRIMARY KEY CHECK(regla ~ '^[a-z][a-z0-9_]{2,63}$'),
 audiencia_administrativa text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(audiencia_administrativa,256) IS TRUE),
 alta_en timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(alta_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_cargos_firma_admin_v1','registro_cargos_firma_admin_v1','operacion_cargos_firma_admin_v1','regla_asignacion_cargo_firma_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;
INSERT INTO vec_autorizacion.regla_asignacion_cargo_firma_v1(regla,audiencia_administrativa) VALUES('lote_ordinario','vec_autorizacion.administracion_perfiles.lote_ordinario.v1');
RESET ROLE;
CREATE ROLE vec_admin_cargos_firma_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_cargos_firma_ejecutor',current_database()); END $conexion$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- El LOGIN debe ser mínimo y exclusivo, como en AUT49: miembro único del
-- grupo con INHERIT y sin SET/ADMIN, sin ajustes propios ni permisos directos.
CREATE FUNCTION vec_autorizacion.exigir_operador_cargos_firma_admin_v1()
RETURNS vec_autorizacion.config_cargos_firma_admin_v1 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_cargos_firma_admin_v1;ns oid;db oid;f oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none' THEN RAISE EXCEPTION 'AUT53: PARO clave=transaccion actual=divergente esperado=SERIALIZABLE_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;SELECT * INTO g FROM pg_roles WHERE rolname='vec_admin_cargos_firma_ejecutor';
 ns:=to_regnamespace('vec_autorizacion');SELECT oid INTO db FROM pg_database WHERE datname=current_database();f:=to_regprocedure('vec_autorizacion.publicar_cargos_firma_admin_v1(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid) OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT((dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a') OR(dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=f AND deptype='a') OR(dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='CONNECT' AND NOT x.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='USAGE' AND NOT x.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(x.privilege_type='EXECUTE' AND NOT x.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) x WHERE d.oid=db AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) x WHERE n.oid=ns AND x.grantee=l.oid)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee=l.oid)
 OR has_schema_privilege(l.oid,ns,'CREATE') OR has_database_privilege(l.oid,db,'CREATE,TEMP') THEN RAISE EXCEPTION 'AUT53: PARO clave=operador actual=no_acreditado esperado=LOGIN_minimo_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_cargos_firma_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN RAISE EXCEPTION 'AUT53: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_cargos_firma_admin_v1() FROM PUBLIC;

-- Concesiones de la firma V2 que puede pedir el plan, exactamente como las
-- comprueba AD162: registrar la firma con certificado VEC y consultar las
-- firmas R5 V2 con sus 44 campos. No hay otra operación V2 admisible.
CREATE FUNCTION vec_autorizacion.concesion_v2_cargo_firma_v1(p_operacion text)
RETURNS jsonb LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT CASE p_operacion
 WHEN 'firmar_vec' THEN '{"accion":"contratacion_temporal.documento.firma_vec.registrar","modulo_id":"contratacion_temporal","tipo_recurso":"firma_vec_documento_contratacion_temporal","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}'::jsonb
 WHEN 'consultar_r5' THEN '{"accion":"contratacion_temporal.documento.firmas_r5_v2.consultar","modulo_id":"contratacion_temporal","tipo_recurso":"expediente_contratacion_temporal","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":["ByteRange","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"],"obligaciones":[]}'::jsonb
 END
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesion_v2_cargo_firma_v1(text) FROM PUBLIC;

-- Documento de rol de un cargo del plan. Valida los doce campos cerrados y
-- construye la versión: nombre, concesiones V2 pedidas (en orden fijo) y las
-- competenciales del paso, todas de Contratación temporal, garantía alta.
-- La competencial sólo admite acciones de documento sobre un tipo
-- documento_…contratacion_temporal: ningún consumidor V3 usa esos tipos, así
-- que no amplía otra operación. Fecha y publicador salen del plan.
CREATE FUNCTION vec_autorizacion.documento_rol_cargo_firma_v1(c jsonb,p jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE concesiones jsonb:='[]';x jsonb;op text;anterior text:='';
BEGIN
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>12
 OR NOT c ?& ARRAY['rol_id','version','nombre','version_anterior_sha256','operaciones_v2','competencias','version_rol_sha256','regla_asignacion','organizacion_ref','vigente_desde','vigente_hasta','duracion_propuesta_segundos']
 OR jsonb_typeof(c->'rol_id') IS DISTINCT FROM 'string' OR c->>'rol_id' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'version') IS DISTINCT FROM 'number' OR c->>'version' !~ '^[1-9][0-9]{0,8}$'
 OR jsonb_typeof(c->'nombre') IS DISTINCT FROM 'string' OR char_length(c->>'nombre') NOT BETWEEN 3 AND 200 OR c->>'nombre'<>btrim(c->>'nombre') OR c->>'nombre' ~ '[[:cntrl:]]' OR c->>'nombre' ~ '^[[:space:]]|[[:space:]]$'
 OR jsonb_typeof(c->'version_anterior_sha256') IS DISTINCT FROM 'string'
 OR ((c->>'version')='1' AND c->>'version_anterior_sha256'<>'') OR ((c->>'version')<>'1' AND c->>'version_anterior_sha256' !~ '^[0-9a-f]{64}$')
 OR jsonb_typeof(c->'operaciones_v2') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'operaciones_v2')>2
 OR jsonb_typeof(c->'competencias') IS DISTINCT FROM 'array' OR jsonb_array_length(c->'competencias') NOT BETWEEN 1 AND 8
 OR jsonb_typeof(c->'version_rol_sha256') IS DISTINCT FROM 'string' OR c->>'version_rol_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(c->'regla_asignacion') IS DISTINCT FROM 'string' OR c->>'regla_asignacion' !~ '^[a-z][a-z0-9_]{2,63}$'
 OR jsonb_typeof(c->'organizacion_ref') IS DISTINCT FROM 'string' OR c->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'AUT53: PARO clave=cargo actual=invalido esperado=campos_exactos' USING ERRCODE='22023'; END IF;
 -- Operaciones V2: subconjunto ordenado y sin repetir de las dos admitidas.
 FOR op IN SELECT value FROM jsonb_array_elements_text(c->'operaciones_v2') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF op NOT IN('consultar_r5','firmar_vec') OR op COLLATE "C"<=anterior COLLATE "C" THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=subconjunto_ordenado' USING ERRCODE='22023'; END IF;
  anterior:=op;
  concesiones:=concesiones||jsonb_build_array(vec_autorizacion.concesion_v2_cargo_firma_v1(op));
 END LOOP;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(c->'operaciones_v2') e WHERE jsonb_typeof(e)<>'string') THEN RAISE EXCEPTION 'AUT53: PARO clave=operaciones_v2 actual=invalido esperado=cadenas' USING ERRCODE='22023'; END IF;
 FOR x IN SELECT value FROM jsonb_array_elements(c->'competencias') WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF jsonb_typeof(x) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(x))<>3 OR NOT x ?& ARRAY['accion','tipo_recurso','finalidad']
  OR jsonb_typeof(x->'accion') IS DISTINCT FROM 'string' OR x->>'accion' !~ '^contratacion_temporal\.documento(\.[a-z][a-z0-9_]{1,63}){1,4}$'
  OR jsonb_typeof(x->'tipo_recurso') IS DISTINCT FROM 'string' OR x->>'tipo_recurso' !~ '^documento_([a-z0-9_]{1,80}_)?contratacion_temporal$'
  OR jsonb_typeof(x->'finalidad') IS DISTINCT FROM 'string' OR x->>'finalidad' !~ '^[a-z][a-z0-9_]{2,127}$' OR x->>'finalidad' ~ '(fiscaliz|intervenc)'
  THEN RAISE EXCEPTION 'AUT53: PARO clave=competencia actual=invalida esperado=documento_contratacion_temporal' USING ERRCODE='22023'; END IF;
  concesiones:=concesiones||jsonb_build_array(jsonb_build_object('accion',x->>'accion','modulo_id','contratacion_temporal','tipo_recurso',x->>'tipo_recurso',
   'finalidades',jsonb_build_array(x->>'finalidad'),'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb));
 END LOOP;
 RETURN jsonb_build_object('rol_id',c->>'rol_id','version',(c->>'version')::bigint,'nombre',c->>'nombre','estado','publicada','concesiones',concesiones,
  'publicada_por','operacion:cargos_firma:'||(p->>'operacion_ref'),'publicada_en',p->>'preparado_en');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_cargos_firma_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE cfg vec_autorizacion.config_cargos_firma_admin_v1;p jsonb;sha text;c jsonb;d jsonb;ctl jsonb;ref text;h text;hc text;maxv bigint;prev text;prev_doc jsonb;
 reg record;lista jsonb:='[]';cargos jsonb:='[]';lista_sha text;e jsonb;aud record;recibo jsonb;previo record;publicador text;instante timestamptz;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_cargos_firma_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=divergente esperado=plan_privado_aprobado' USING ERRCODE='42501'; END IF;
 sha:=encode(sha256(convert_to(plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM cfg.plan_sha256 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan_sha actual=divergente esperado=aprobado' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;
 -- El texto aprobado debe ser la forma canónica de jsonb, como en AUT49.
 IF plan_canonico IS DISTINCT FROM p::text THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=no_canonico esperado=jsonb_text' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>5 OR NOT p ?& ARRAY['esquema','operacion_ref','preparado_en','caduca_en','cargos']
 OR p->>'esquema' IS DISTINCT FROM 'vec.admin.cargos-firma.plan.v1' OR jsonb_typeof(p->'operacion_ref') IS DISTINCT FROM 'string' OR p->>'operacion_ref' !~ '^rpa_cf_[A-Za-z0-9_-]{19,121}$'
 OR jsonb_typeof(p->'cargos') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'cargos') NOT BETWEEN 1 AND 16
 OR (SELECT count(DISTINCT x->>'rol_id') FROM jsonb_array_elements(p->'cargos') x)<>jsonb_array_length(p->'cargos')
 OR jsonb_typeof(p->'preparado_en') IS DISTINCT FROM 'string' OR jsonb_typeof(p->'caduca_en') IS DISTINCT FROM 'string'
 OR p->>'preparado_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' OR p->>'caduca_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR (p->>'caduca_en')::timestamptz>(p->>'preparado_en')::timestamptz+interval '1 day'
 THEN RAISE EXCEPTION 'AUT53: PARO clave=plan actual=invalido esperado=plan_cerrado_1_16' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.registro_cargos_firma_admin_v1 WHERE operacion_ref=p->>'operacion_ref' FOR SHARE;
 IF FOUND THEN
  -- Replay: mismo plan exacto; el recibo original sólo se devuelve si cada
  -- versión publicada sigue con su huella, habilitada y registrada como ordinaria.
  IF previo.plan IS DISTINCT FROM convert_to(plan_canonico,'UTF8') OR previo.plan_sha256 IS DISTINCT FROM sha
  THEN RAISE EXCEPTION 'AUT53: PARO clave=replay actual=material_distinto esperado=plan_original' USING ERRCODE='23505'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(previo.recibo->'cargos') x LEFT JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=x->>'version_rol_ref'
   LEFT JOIN vec_autorizacion.rol_administrable_exacto_v1 a ON a.version_rol_ref=r.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
   LEFT JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=ca.version_rol_ref AND v.revision=ca.revision
   WHERE r.version_rol_ref IS NULL OR r.huella_sha256 IS DISTINCT FROM x->>'version_rol_sha256' OR a.version_rol_ref IS NULL OR a.clase<>'ordinario' OR a.huella_sha256 IS DISTINCT FROM r.huella_sha256
   OR v.estado IS DISTINCT FROM 'habilitada')
  THEN RAISE EXCEPTION 'AUT53: PARO clave=replay_cargos actual=divergente esperado=registro_original' USING ERRCODE='P0V01'; END IF;
  RETURN jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AUT53: PARO clave=plan_caducado actual=caducado esperado=vigente' USING ERRCODE='42501'; END IF;
 publicador:='operacion:cargos_firma:'||(p->>'operacion_ref');
 instante:=(p->>'preparado_en')::timestamptz;
 FOR c IN SELECT value FROM jsonb_array_elements(p->'cargos') ORDER BY value->>'rol_id' LOOP
  d:=vec_autorizacion.documento_rol_cargo_firma_v1(c,p);
  -- CAS de versión: la 1 exige que el rol no exista; la N exige que la última
  -- publicada sea N-1 y tenga la huella que el plan declara.
  SELECT max(version) INTO maxv FROM vec_autorizacion.version_rol WHERE rol_id=c->>'rol_id';
  SELECT huella_sha256,documento INTO prev,prev_doc FROM vec_autorizacion.version_rol WHERE rol_id=c->>'rol_id' AND version=(c->>'version')::bigint-1 FOR SHARE;
  IF ((c->>'version')::bigint=1 AND maxv IS NOT NULL) OR ((c->>'version')::bigint>1 AND (maxv IS DISTINCT FROM (c->>'version')::bigint-1 OR prev IS DISTINCT FROM c->>'version_anterior_sha256'))
  THEN RAISE EXCEPTION 'AUT53: PARO clave=version actual=divergente esperado=CAS_version_anterior' USING ERRCODE='P0V01'; END IF;
  -- Sólo se sucede a un rol que ya tiene forma de cargo de firma: todas sus
  -- concesiones de Contratación temporal y sobre los tipos que publica AUT53.
  -- Así no se saca la versión siguiente de un rol ajeno (Dietas, Bolsa…).
  IF (c->>'version')::bigint>1 AND (jsonb_typeof(prev_doc->'concesiones') IS DISTINCT FROM 'array' OR EXISTS(SELECT 1 FROM jsonb_array_elements(prev_doc->'concesiones') q
   WHERE q->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
   OR (q->>'tipo_recurso' ~ '^(documento_([a-z0-9_]{1,80}_)?contratacion_temporal|firma_vec_documento_contratacion_temporal|expediente_contratacion_temporal)$') IS NOT TRUE))
  THEN RAISE EXCEPTION 'AUT53: PARO clave=version_anterior actual=no_es_cargo esperado=rol_con_forma_de_cargo' USING ERRCODE='P0V01'; END IF;
  SELECT * INTO reg FROM vec_autorizacion.regla_asignacion_cargo_firma_v1 r WHERE r.regla=c->>'regla_asignacion';
  IF NOT FOUND THEN RAISE EXCEPTION 'AUT53: PARO clave=regla_asignacion actual=desconocida esperado=configurada' USING ERRCODE='22023'; END IF;
  IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE THEN RAISE EXCEPTION 'AUT53: PARO clave=concesiones actual=invalidas esperado=positivas_sin_repetir' USING ERRCODE='22023'; END IF;
  h:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex');
  IF h IS DISTINCT FROM c->>'version_rol_sha256' THEN RAISE EXCEPTION 'AUT53: PARO clave=version_rol_sha actual=divergente esperado=huella_del_plan' USING ERRCODE='P0V01'; END IF;
  ref:='rol:'||(c->>'rol_id')||':v'||(c->>'version');
  ctl:=jsonb_build_object('version_rol_ref',ref,'revision',1,'estado','habilitada','actualizado_por',publicador,'actualizado_en',p->>'preparado_en');
  hc:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(ctl),'UTF8')),'hex');
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES(ref,c->>'rol_id',(c->>'version')::bigint,h,instante,d);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES(ref,1,'habilitada',hc,instante,ctl);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) VALUES(ref,1,instante,publicador,publicador);
  -- Defensa de clases de AUT49 sobre la versión recién publicada. La unidad
  -- la aporta cada asignación: AUT32 exige organización y unidad.
  lista:=lista||jsonb_build_array(vec_autorizacion.validar_perfil_asignable_admin_v1(jsonb_build_object('version_rol_ref',ref,'version_rol_sha256',h,
   'control_revision',1,'control_sha256',hc,'unidad_requerida',true,
   'ambitos_fijos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(c->>'organizacion_ref'))),
   'vigente_desde',c->'vigente_desde','vigente_hasta',c->'vigente_hasta','duracion_propuesta_segundos',c->'duracion_propuesta_segundos'),true));
  IF EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=ref) THEN RAISE EXCEPTION 'AUT53: PARO clave=ya_registrado actual=presente esperado=ausente' USING ERRCODE='23505'; END IF;
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1(version_rol_ref,clase,huella_sha256,vigente_desde,vigente_hasta,unidad_requerida,audiencia_administrativa,ambitos_fijos,duracion_propuesta)
  VALUES(ref,'ordinario',h,(c->>'vigente_desde')::timestamptz,(c->>'vigente_hasta')::timestamptz,true,reg.audiencia_administrativa,
   jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array(c->>'organizacion_ref'))),
   make_interval(secs=>(c->>'duracion_propuesta_segundos')::double precision));
  cargos:=cargos||jsonb_build_array(jsonb_build_object('rol_id',c->>'rol_id','version_rol_ref',ref,'version_rol_sha256',h,'control_revision',1,'control_sha256',hc,'regla_asignacion',c->>'regla_asignacion'));
 END LOOP;
 lista_sha:=encode(sha256(convert_to(lista::text,'UTF8')),'hex');
 e:=jsonb_build_object('tipo_registro','perfiles_asignables_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,
  'operacion_ref',p->>'operacion_ref','perfiles_sha256',lista_sha,'perfiles_numero',jsonb_array_length(lista)::text,'aprobacion_sha256',cfg.aprobacion_sha256,
  'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref','correlacion_'||substr(sha,33,32));
 SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(e);
 INSERT INTO vec_autorizacion.operacion_cargos_firma_admin_v1 VALUES(aud.auditoria_ref,'confirmacion',session_user,sha,'permitido',p->>'operacion_ref',aud.registrada_en);
 recibo:=jsonb_build_object('esquema','vec.admin.cargos-firma.recibo.v1','operacion_ref',p->>'operacion_ref','plan_sha256',sha,'aprobacion_ref',cfg.aprobacion_ref,'aprobacion_sha256',cfg.aprobacion_sha256,
  'cargos',cargos,'perfiles_sha256',lista_sha,'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'confirmado_en',aud.registrada_en);
 INSERT INTO vec_autorizacion.registro_cargos_firma_admin_v1 VALUES(p->>'operacion_ref',sha,convert_to(plan_canonico,'UTF8'),session_user,aud.auditoria_ref,recibo,aud.registrada_en);
 -- Revalidación final bajo la misma transacción: configuración aún vigente.
 PERFORM vec_autorizacion.exigir_operador_cargos_firma_admin_v1();
 IF clock_timestamp()>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'AUT53: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recibo',recibo,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_cargos_firma_admin_v1(text,text) FROM PUBLIC;

-- Fachada única del operador. Todo intento, también el replay o el rechazo,
-- deja un registro común y su anotación. El efecto y su intento permitido
-- comparten subbloque; si algo falla se revierten ambos y sólo queda la
-- negativa gestionada.
CREATE FUNCTION vec_autorizacion.publicar_cargos_firma_admin_v1(plan_canonico text,sha_aprobado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='5s' SET statement_timeout='30s' AS $f$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;sol text;evento text;corr text;solsha text;
BEGIN
 sol:='solicitud_perfiles_asignables:'||replace(gen_random_uuid()::text,'-','');evento:='evento_'||replace(gen_random_uuid()::text,'-','');corr:='correlacion_'||replace(gen_random_uuid()::text,'-','');
 -- La huella de la solicitud sólo se calcula sobre un plan dentro del límite.
 solsha:=encode(sha256(convert_to(jsonb_build_object('plan',CASE WHEN octet_length(plan_canonico)<=65536 THEN plan_canonico ELSE '' END,
  'sha_aprobado',CASE WHEN octet_length(sha_aprobado)<=128 THEN sha_aprobado ELSE '' END,'operacion','publicar_cargos_firma_admin_v1')::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_cargos_firma_admin_v1(plan_canonico,sha_aprobado);
  motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'perfiles_asignables_replay' ELSE 'perfiles_asignables_registrado' END;
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb_build_object('tipo_registro','intento_perfiles_asignables_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','registrar_perfiles_asignables_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref',corr));
  INSERT INTO vec_autorizacion.operacion_cargos_firma_admin_v1 VALUES(aud.auditoria_ref,'intento',session_user,solsha,estado,respuesta#>>'{recibo,operacion_ref}',aud.registrada_en);
  PERFORM vec_autorizacion.exigir_operador_cargos_firma_admin_v1();
 EXCEPTION WHEN OTHERS THEN respuesta:=NULL;codigo:=SQLSTATE;estado:=CASE WHEN codigo IN('42501','22023','22P02','22007','22008','P0V01','23505','25000') THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'perfiles_asignables_denegado' ELSE 'perfiles_asignables_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'cargos_firma_rechazado' ELSE 'cargos_firma_no_disponible' END;
 END;
 IF estado<>'permitido' THEN
  SELECT * INTO STRICT aud FROM vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb_build_object('tipo_registro','intento_perfiles_asignables_admin','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solsha,'accion','registrar_perfiles_asignables_admin_v1','recurso_ref',sol,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','perfiles_asignables_admin','correlacion_ref',corr));
  INSERT INTO vec_autorizacion.operacion_cargos_firma_admin_v1 VALUES(aud.auditoria_ref,'intento',session_user,solsha,estado,NULL,aud.registrada_en);
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo','replay',COALESCE((respuesta->>'replay')::boolean,false),
  'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.publicar_cargos_firma_admin_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_cargos_firma_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicar_cargos_firma_admin_v1(text,text) TO vec_admin_cargos_firma_ejecutor;
RESET ROLE;

DO $acl$
DECLARE g oid:=to_regrole('vec_admin_cargos_firma_ejecutor');
 internas regprocedure[]:=ARRAY['vec_autorizacion.exigir_operador_cargos_firma_admin_v1()','vec_autorizacion.concesion_v2_cargo_firma_v1(text)',
  'vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)','vec_autorizacion.aplicar_cargos_firma_admin_v1(text,text)']::regprocedure[];
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=ANY(internas) AND a.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=to_regprocedure('vec_autorizacion.publicar_cargos_firma_admin_v1(text,text)') AND (a.grantee NOT IN(p.proowner,g) OR (a.grantee=g AND a.is_grantable)))
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid IN('vec_autorizacion.config_cargos_firma_admin_v1'::regclass,'vec_autorizacion.registro_cargos_firma_admin_v1'::regclass,
   'vec_autorizacion.operacion_cargos_firma_admin_v1'::regclass,'vec_autorizacion.regla_asignacion_cargo_firma_v1'::regclass) AND a.grantee<>c.relowner)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE (oid=ANY(internas) OR oid=to_regprocedure('vec_autorizacion.publicar_cargos_firma_admin_v1(text,text)'))
  AND proowner<>'vec_autorizacion_propietario'::regrole)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE oid IN(to_regprocedure('vec_autorizacion.exigir_operador_cargos_firma_admin_v1()'),to_regprocedure('vec_autorizacion.aplicar_cargos_firma_admin_v1(text,text)'),
  to_regprocedure('vec_autorizacion.publicar_cargos_firma_admin_v1(text,text)')) AND NOT prosecdef)
 OR (SELECT count(*) FROM vec_autorizacion.regla_asignacion_cargo_firma_v1)<>1
 THEN RAISE EXCEPTION 'AUT53: PARO clave=ACL actual=ampliada esperado=propietario_y_grupo_operador' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
