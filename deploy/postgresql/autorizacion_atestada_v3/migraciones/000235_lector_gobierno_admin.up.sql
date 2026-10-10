\set ON_ERROR_STOP on
-- AD235: lector del gobierno técnico ADMIN sin superusuario.
-- vec-gobierno-usuarios-admin lee con dsn_lectura, en preparar, la instantánea
-- de la configuración vigente y la huella de la preimagen de AD188/AD198, y en
-- verificar, el tramo de la cadena de auditoría del gobierno. Las preimágenes
-- son SECURITY DEFINER con EXECUTE solo para el propietario y las tablas son
-- privadas, así que hasta ahora ese DSN tenía que ser superusuario.
-- Esta migración crea un grupo NOLOGIN sin pertenencias y dos fachadas
-- SECURITY DEFINER, STABLE y de solo lectura, que devuelven exactamente el
-- documento que la CLI leía (mismas expresiones, mismo jsonb):
--  * instantanea_gobierno_admin_lectura_v1(conjunto): la de preparar. Solo
--    lleva la huella SHA256 de la preimagen, no la preimagen.
--  * cadena_gobierno_admin_lectura_v1(esquema): la de verificar.
-- Ninguna devuelve secreto_hmac ni material privado. Antes de llamar a una
-- preimagen, la fachada coteja la huella de su definición: si el núcleo la
-- rehace, la fachada para y hay que publicar otra. Solo el grupo nuevo tiene
-- EXECUTE. Las preimágenes y las tablas siguen solo para su propietario (las
-- guardas de AD198 no cambian). El grupo no tiene TEMP ni CREATE. El LOGIN de
-- cada entorno lo crea el DBA, miembro solo de este grupo. Requiere AD188,
-- AD191, AD198 y AD207. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000235',0));
DO $pre$
DECLARE f record;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD235: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regrole('vec_autorizacion_atestada_v3_lector_gobierno') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD235: PARO clave=preimagen actual=ya_instalada esperado=sin_AD235' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_configuracion_actual') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_confianza_version') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_raiz') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.raiz_confianza_version') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_clave_emision') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.control_cadena_auditoria') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.eslabon_auditoria_v5') IS NULL
 THEN RAISE EXCEPTION 'AD235: PARO clave=AD198_AD207 actual=ausente esperado=instaladas' USING ERRCODE='55000'; END IF;
 -- Huella exacta de las dos preimágenes envueltas (AD188 y AD198), dueño,
 -- SECURITY DEFINER y ACL solo del propietario, como exigen las guardas de AD198.
 FOR f IN SELECT * FROM (VALUES
   ('vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()','117916cefb1a0f8891f9b6ec46f15b2f0231489dd34b036eb15fd7e28f97c222'),
   ('vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)','924fb5a69b691404247420f885abaebd2de659711ab115b21960bfba58fe1366')) v(firma,huella) LOOP
  IF pg_catalog.to_regprocedure(f.firma) IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(f.firma)
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')=f.huella)
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE p.oid=pg_catalog.to_regprocedure(f.firma) AND a.grantee<>p.proowner)
  THEN RAISE EXCEPTION 'AD235: PARO clave=preimagen_envuelta actual=distinta esperado=AD188_AD198 %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 -- Con USAGE en el esquema el grupo podría llamar a cualquier función abierta
 -- a PUBLIC: no debe haber ninguna, ni tablas abiertas a PUBLIC.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault(CASE WHEN c.relkind='S' THEN 's'::"char" ELSE 'r'::"char" END,c.relowner))) a
   WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
   WHERE d.datname=pg_catalog.current_database() AND a.grantee=0 AND a.privilege_type IN('TEMPORARY','CREATE'))
 THEN RAISE EXCEPTION 'AD235: PARO clave=PUBLIC actual=con_permisos esperado=sin_EXECUTE_SELECT_TEMP_CREATE' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_autorizacion_atestada_v3_lector_gobierno NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $connect$ BEGIN EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v3_lector_gobierno',pg_catalog.current_database()); END $connect$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Instantánea de preparar. Conjunto 0: gobierno de usuarios (AD188); 1 o más:
-- conjunto de capacidades de AD198. Mismo documento que la consulta que la CLI
-- lanzaba antes de AD235; un conjunto inexistente da pre_sha nulo.
CREATE FUNCTION vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(p_conjunto integer)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE envuelta regprocedure;huella text;pre text;doc jsonb;
BEGIN
 IF p_conjunto IS NULL OR p_conjunto NOT BETWEEN 0 AND 1000000
 THEN RAISE EXCEPTION 'AD235: PARO clave=conjunto actual=fuera_de_rango esperado=0_a_1000000' USING ERRCODE='22023'; END IF;
 IF p_conjunto=0 THEN
  envuelta:=to_regprocedure('vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()');
  huella:='117916cefb1a0f8891f9b6ec46f15b2f0231489dd34b036eb15fd7e28f97c222';
 ELSE
  envuelta:=to_regprocedure('vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)');
  huella:='924fb5a69b691404247420f885abaebd2de659711ab115b21960bfba58fe1366';
 END IF;
 IF envuelta IS NULL OR encode(sha256(convert_to(pg_get_functiondef(envuelta),'UTF8')),'hex') IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AD235: PARO clave=preimagen_envuelta actual=distinta esperado=AD188_AD198' USING ERRCODE='55000'; END IF;
 IF p_conjunto=0 THEN
  pre:=encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex');
 ELSE
  pre:=encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(p_conjunto)::text,'UTF8')),'hex');
 END IF;
 SELECT jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,'spki',encode(r.clave_publica_spki,'base64'),'clave_id',r.clave_id,'version',r.version,'audiencia',r.audiencia_despliegue,'desde',r.valida_desde,'hasta',r.valida_hasta,'orden',(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),'max_version',(SELECT max(version) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'max_revision',(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'pre_sha',pre)
 INTO doc
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version ORDER BY p.orden DESC LIMIT 1;
 RETURN doc;
END $f$;

-- Documento de verificar: tramo contiguo de la familia del gobierno de
-- usuarios tras el último evento de otra familia. Mismo documento que la
-- consulta que la CLI lanzaba antes de AD235; p_esquema solo se repite.
CREATE FUNCTION vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(p_esquema text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE d jsonb;
BEGIN
 IF p_esquema IS NULL OR p_esquema !~ '^[a-z0-9][a-z0-9._-]{0,127}$'
 THEN RAISE EXCEPTION 'AD235: PARO clave=esquema actual=invalido esperado=identificador_acotado' USING ERRCODE='22023'; END IF;
 WITH corte AS (SELECT secuencia n FROM vec_autorizacion_atestada_v3.control_cadena_auditoria),
 cadena AS (
  SELECT a.*,a.secuencia posicion,NULL::jsonb eslabon,a.anterior_sha256 enlace_previo,a.huella_sha256 enlace
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a,corte WHERE a.secuencia<=corte.n
  UNION ALL
  SELECT a.*,e.posicion,jsonb_build_object('posicion',e.posicion,'secuencia',e.secuencia,'anterior_sha256',e.anterior_sha256,'eslabon_sha256',e.eslabon_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'sellado_en',to_char(e.sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),e.anterior_sha256,e.eslabon_sha256
  FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING (secuencia)
 ), rows AS (
  SELECT * FROM cadena
  WHERE tipo_registro IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
  AND posicion > COALESCE((SELECT max(posicion) FROM cadena
  WHERE tipo_registro NOT IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')),0)
 ), rango AS(SELECT min(posicion) primero,max(posicion) ultimo,count(*) cuenta FROM rows)
 SELECT jsonb_build_object('esquema',p_esquema,'manifiesto',jsonb_build_object('cadena_id','cadena:comun:interna','primera_secuencia',ra.primero,'ultima_secuencia',ra.ultimo,'registros',ra.cuenta,'anterior_sha256',(SELECT enlace_previo FROM rows ORDER BY posicion LIMIT 1),'cabeza_sha256',(SELECT enlace FROM rows ORDER BY posicion DESC LIMIT 1)),
  'registros',(SELECT jsonb_agg(jsonb_build_object('tipo_registro',a.tipo_registro) || CASE WHEN a.eslabon IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('eslabon',a.eslabon) END ||
  CASE WHEN a.tipo_registro='gobierno_usuarios_admin' THEN jsonb_build_object('gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('plan_sha256',a.plan_sha256,'preimagen_sha256',a.gobierno_usuarios_detalle->>'preimagen_sha256','configuracion_origen_ref',a.gobierno_usuarios_detalle->>'configuracion_origen_ref','configuracion_destino_ref',a.gobierno_usuarios_detalle->>'configuracion_destino_ref','claves_sha256',a.gobierno_usuarios_detalle->>'claves_sha256'))
  ELSE jsonb_build_object('intento_gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref) || jsonb_build_object('solicitud_sha256',a.gobierno_usuarios_solicitud_sha256)) END
  ORDER BY a.posicion) FROM rows a))
 INTO d FROM rango ra;
 RETURN d;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer),
 vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_atestada_v3_lector_gobierno;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer),
 vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text) TO vec_autorizacion_atestada_v3_lector_gobierno;
RESET ROLE;
DO $post$
DECLARE g oid:='vec_autorizacion_atestada_v3_lector_gobierno'::regrole;b oid;
 fachadas oid[]:=ARRAY['vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)'::regprocedure,'vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)'::regprocedure]::oid[];
BEGIN
 SELECT oid INTO STRICT b FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 -- Grupo: sin LOGIN ni atributos, sin configuración, sin pertenencias ni miembros.
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE oid=g AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolcreatedb
   AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g OR roleid=g)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole=g)
 THEN RAISE EXCEPTION 'AD235: PARO clave=grupo actual=incompatible esperado=NOLOGIN_sin_pertenencias' USING ERRCODE='55000'; END IF;
 -- Base: solo CONNECT, sin TEMP ni CREATE (vec-server rechaza un LOGIN vec_* con TEMP).
 IF pg_catalog.has_database_privilege(g,b,'TEMPORARY') OR pg_catalog.has_database_privilege(g,b,'CREATE') OR NOT pg_catalog.has_database_privilege(g,b,'CONNECT')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
   WHERE d.oid=b AND a.grantee=g AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD235: PARO clave=base actual=incompatible esperado=solo_CONNECT' USING ERRCODE='55000'; END IF;
 -- Esquema: solo USAGE; ningún otro esquema le concede nada.
 IF pg_catalog.has_schema_privilege(g,'vec_autorizacion_atestada_v3','CREATE')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
   WHERE a.grantee=g AND (n.oid<>'vec_autorizacion_atestada_v3'::regnamespace OR a.privilege_type<>'USAGE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD235: PARO clave=esquema actual=incompatible esperado=solo_USAGE' USING ERRCODE='55000'; END IF;
 -- Ninguna tabla, vista o secuencia legible o escribible por el grupo: sin
 -- entradas propias en ningún ACL (PUBLIC ya se comprobó vacío en el UP) y sin
 -- privilegio efectivo sobre las tablas del esquema.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace
   AND CASE WHEN c.relkind IN('r','p','v','m','f') THEN pg_catalog.has_table_privilege(g,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') ELSE false END)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a WHERE a.grantee=g)
 THEN RAISE EXCEPTION 'AD235: PARO clave=tablas actual=accesibles esperado=ninguna' USING ERRCODE='55000'; END IF;
 -- Funciones: el grupo solo ejecuta las dos fachadas; ni las preimágenes ni
 -- ninguna otra función del esquema.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
   AND p.oid<>ALL(fachadas) AND pg_catalog.has_function_privilege(g,p.oid,'EXECUTE'))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=g AND p.oid<>ALL(fachadas))
 THEN RAISE EXCEPTION 'AD235: PARO clave=funciones actual=ampliadas esperado=solo_fachadas' USING ERRCODE='55000'; END IF;
 -- Fachadas: propietario, SECURITY DEFINER, STABLE, search_path fijo y EXECUTE
 -- solo para el propietario y el grupo, sin opción de concesión.
 IF (SELECT count(*) FROM pg_catalog.pg_proc p WHERE p.oid=ANY(fachadas) AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='s' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'])<>2
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=ANY(fachadas) AND (a.grantee NOT IN(p.proowner,g) OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 OR NOT pg_catalog.has_function_privilege(g,fachadas[1],'EXECUTE') OR NOT pg_catalog.has_function_privilege(g,fachadas[2],'EXECUTE')
 THEN RAISE EXCEPTION 'AD235: PARO clave=fachadas actual=incompatibles esperado=definer_stable_solo_grupo' USING ERRCODE='55000'; END IF;
 -- Las guardas de AD198 siguen igual: preimágenes solo del propietario.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid IN('vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()'::regprocedure,'vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)'::regprocedure)
   AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'AD235: PARO clave=AD198 actual=preimagen_ampliada esperado=solo_propietario' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
