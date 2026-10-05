\set ON_ERROR_STOP on
-- AD198: gobierno genérico de las claves de capacidad V3 de Administración.
-- Publica en una sola operación la configuración diaria y una clave nueva por
-- cada audiencia de un conjunto CERRADO y VERSIONADO (tabla
-- conjunto_audiencias_capacidad_admin_v1). El conjunto 1 son las dos
-- audiencias de usuarios (AD188) y la del lote ordinario de perfiles (AD190).
-- Otro conjunto exige otra migración que añada su fila: nunca se edita uno.
-- AD188/AD191 no se reescriben: se reutilizan su cálculo de huella de la
-- configuración y sus registros de auditoría común (mismos tipos, sin tocar el
-- CHECK ni el verificador de la cadena). Como esos registros llevan la acción de
-- AD188, cada operación de AD198 (también los intentos denegados) se anota
-- además en operacion_gobierno_capacidades_admin_v1, de solo adición, en la
-- misma transacción y ligada a la referencia de auditoría. El grupo operador y
-- la configuración aprobada son propios. Requiere AD188, AD191 y AD190 instaladas. Una sola
-- vez; sin DOWN. Instalarla no publica ninguna clave.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000198',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE efecto text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD198: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 -- AD188 con la corrección de AD191 (huella de la definición tras AD191).
 SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') INTO efecto FROM pg_proc p
 WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)');
 IF efecto IS DISTINCT FROM 'aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973'
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1(timestamp with time zone)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(jsonb,vec_autorizacion_atestada_v3.raiz_confianza_version)') IS NULL
 THEN RAISE EXCEPTION 'AD198: PARO clave=AD188_AD191 actual=ausente_o_sin_corregir esperado=instaladas' USING ERRCODE='55000'; END IF;
 -- La clave del lote sólo cabe en la tabla de claves si AD190 amplió su CHECK.
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_autorizacion.administracion_perfiles.lote_ordinario.v1')>0)
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD198: PARO clave=AD190 actual=ausente esperado=audiencia_lote_admitida' USING ERRCODE='55000'; END IF;
 IF to_regrole('vec_gobierno_capacidades_admin_operador') IS NOT NULL
 OR to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AD198: PARO clave=preimagen actual=ya_instalada esperado=sin_AD198' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_gobierno_capacidades_admin_operador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_gobierno_capacidades_admin_operador',current_database()); END $connect$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- Conjuntos cerrados de audiencias. Cada fila es inmutable; el orden de
-- audiencias es el de las claves del material. segmentos fija el tramo de
-- clave_id de cada audiencia (clave:capacidad:admin:<segmento>:...).
CREATE TABLE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(
 version integer PRIMARY KEY CHECK(version BETWEEN 1 AND 1000000),
 audiencias text[] NOT NULL CHECK(cardinality(audiencias) BETWEEN 1 AND 16 AND array_position(audiencias,NULL) IS NULL),
 segmentos text[] NOT NULL CHECK(cardinality(segmentos)=cardinality(audiencias) AND array_position(segmentos,NULL) IS NULL),
 registrado_en timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 FROM PUBLIC;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES
 (1,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1'],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote']);

-- Configuración aprobada por el DBA para un LOGIN técnico: conjunto y huellas
-- exactas de plan, preimagen y material, con una ventana.
CREATE TABLE vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1(
 identidad_login name PRIMARY KEY,
 conjunto_version integer NOT NULL REFERENCES vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL CHECK(vigente_hasta>vigente_desde),
 entorno text NOT NULL CHECK(entorno='desarrollo'));
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 FROM PUBLIC,vec_gobierno_capacidades_admin_operador;

-- Registro propio de cada operación de AD198, de solo adición. Los registros de
-- la cadena común llevan la acción de AD188; esta fila, unida por auditoria_ref,
-- dice que fue AD198, con qué conjunto y qué claves publicó.
CREATE TABLE vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1(
 auditoria_ref text PRIMARY KEY CHECK(auditoria_ref ~ '^aud_v3_gui?_[0-9a-f]{32}$'),
 tipo text NOT NULL CHECK(tipo IN('confirmacion','intento')),
 funcion text NOT NULL CHECK(funcion='aprovisionar_gobierno_capacidades_admin_v1'),
 operador_login name NOT NULL,
 solicitud_sha256 text NOT NULL CHECK(solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 conjunto_version integer REFERENCES vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version),
 resultado text NOT NULL CHECK(resultado IN('permitido','denegado','error')),
 clave_ids text[] NOT NULL CHECK(array_position(clave_ids,NULL) IS NULL),
 registrada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(resultado='permitido' OR cardinality(clave_ids)=0),
 CHECK(tipo='intento' OR resultado='permitido'),
 CHECK((tipo='confirmacion')=(auditoria_ref LIKE 'aud\_v3\_gu\_%')));
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1()
RETURNS vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1;l record;g oid;f oid;
BEGIN
 g:='vec_gobierno_capacidades_admin_operador'::regrole;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)');
 SELECT * INTO STRICT l FROM pg_roles WHERE rolname=session_user;
 IF NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls
 OR l.rolconfig IS NOT NULL OR NOT pg_has_role(session_user,g,'MEMBER')
 OR EXISTS(WITH RECURSIVE m(oid) AS(SELECT roleid FROM pg_auth_members WHERE member=l.oid UNION SELECT a.roleid FROM pg_auth_members a JOIN m ON a.member=m.oid) SELECT 1 FROM m WHERE oid<>g)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND (admin_option OR set_option OR NOT inherit_option))
 OR EXISTS(SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=l.oid AND d.deptype IN('a','o'))
 OR EXISTS(SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=g AND d.deptype='a' AND NOT
   ((d.classid='pg_database'::regclass AND d.objid=(SELECT oid FROM pg_database WHERE datname=current_database()))
    OR (d.classid='pg_namespace'::regclass AND d.objid='vec_autorizacion_atestada_v3'::regnamespace)
    OR (d.classid='pg_proc'::regclass AND d.objid=f)))
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname=current_database() AND a.grantee=g AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid='vec_autorizacion_atestada_v3'::regnamespace AND a.grantee=g AND (a.privilege_type<>'USAGE' OR a.is_grantable))
 OR EXISTS(SELECT 1 FROM pg_proc x CROSS JOIN LATERAL aclexplode(coalesce(x.proacl,acldefault('f',x.proowner))) a WHERE x.oid=f AND a.grantee=g AND (a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR EXISTS(SELECT 1 FROM pg_db_role_setting s WHERE s.setrole IN(l.oid,g))
 OR has_database_privilege(session_user,current_database(),'CREATE') OR has_database_privilege(session_user,current_database(),'TEMP')
 THEN RAISE EXCEPTION 'AD198: PARO clave=LOGIN actual=incompatible esperado=tecnico_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 WHERE identidad_login=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<c.vigente_desde OR clock_timestamp()>=c.vigente_hasta
 THEN RAISE EXCEPTION 'AD198: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobada_vigente' USING ERRCODE='42501'; END IF;
 RETURN c;
END $f$;

-- Misma preimagen que AD188 con el conjunto y las claves de sus audiencias.
CREATE FUNCTION vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(p_conjunto integer)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('conjunto_version',s.version,'audiencias',to_jsonb(s.audiencias),
 'configuracion',(SELECT to_jsonb(c) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision ORDER BY p.orden DESC LIMIT 1),
 'raices',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY r.clave_id,r.version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.raiz_confianza_version r JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.raiz_clave_id=r.clave_id AND cr.raiz_version=r.version WHERE cr.configuracion_revision=(SELECT configuracion_revision FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual ORDER BY orden DESC LIMIT 1)),
 'checkpoint',(SELECT to_jsonb(c) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c WHERE control_id),
 'orden_configuracion',(SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual),
 'orden_claves',(SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
 'revocaciones_raiz',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY raiz_clave_id,raiz_version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.revocacion_raiz r),
 'revocaciones_configuracion',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY configuracion_revision),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.revocacion_configuracion r),
 'claves',(SELECT coalesce(jsonb_agg(to_jsonb(k)-'secreto_hmac' ORDER BY clave_id,version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.clave_capacidad_version k WHERE k.audiencia_consumo=ANY(s.audiencias)))
 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 s WHERE s.version=p_conjunto
$f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p jsonb,m jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE k jsonb;g record;r vec_autorizacion_atestada_v3.raiz_confianza_version;checkpoint record;ahora timestamptz;
BEGIN
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 SELECT x.* INTO STRICT g FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision WHERE a.orden=(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual) FOR SHARE OF a,x;
 SELECT x.* INTO STRICT r FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=g.revision FOR SHARE OF cr,x;
 SELECT * INTO STRICT checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id FOR SHARE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 ahora:=clock_timestamp();
 IF (p->>'caduca_en')::timestamptz<=ahora THEN RAISE EXCEPTION 'AD198: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 IF g.revision IS DISTINCT FROM p->'configuracion'->>'revision' OR g.secuencia IS DISTINCT FROM (p->'configuracion'->>'secuencia')::bigint
 OR g.huella_configuracion_sha256 IS DISTINCT FROM p->'configuracion'->>'huella_sha256'
 OR g.publicada_en>ahora OR g.expira_en<=ahora OR r.valida_desde>ahora OR r.valida_hasta<=ahora
 OR g.secuencia<checkpoint.configuracion_secuencia_minima OR r.version<checkpoint.raiz_version_minima
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE x.raiz_clave_id=r.clave_id AND x.raiz_version=r.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=g.revision)
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(p->'configuracion',r) IS DISTINCT FROM g.huella_configuracion_sha256
 THEN RAISE EXCEPTION 'AD198: PARO clave=gobierno_actual actual=incompatible esperado=original_vigente_no_retirado' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(m->'claves') LOOP
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x JOIN vec_autorizacion_atestada_v3.puntero_clave_emision a ON a.clave_id=x.clave_id AND a.version=x.version
   WHERE x.clave_id=k->>'clave_id' AND x.version=(k->>'version')::bigint AND x.audiencia_consumo=k->>'audiencia'
   AND x.revision_gobierno=(k->>'revision_gobierno')::bigint AND x.huella_gobierno_sha256=k->>'huella_gobierno_sha256' AND x.huella_secreto_sha256=k->>'huella_secreto_sha256'
   AND x.emisor_id=k->>'emisor_id' AND x.valida_desde=(k->>'valida_desde')::timestamptz AND x.valida_hasta=(k->>'valida_hasta')::timestamptz
   AND x.valida_desde<=ahora AND x.valida_hasta>ahora
   AND a.orden=(SELECT max(pc.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision pc JOIN vec_autorizacion_atestada_v3.clave_capacidad_version kc ON kc.clave_id=pc.clave_id AND kc.version=pc.version WHERE kc.audiencia_consumo=x.audiencia_consumo AND pc.establecida_en<=ahora)
   AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad rc WHERE rc.clave_id=x.clave_id AND rc.version=x.version))
  THEN RAISE EXCEPTION 'AD198: PARO clave=clave_actual actual=retirada_o_distinta esperado=original_vigente' USING ERRCODE='42501';END IF;
 END LOOP;
END $f$;

-- Efecto: igual que AD188 (tras AD191) con N claves, una por audiencia del
-- conjunto aprobado y en su orden. La confirmación y los intentos usan los
-- registros comunes de AD188.
CREATE FUNCTION vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(p_plan text,p_aprobado text,p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1;s vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1;
 p jsonb;m jsonb;g jsonb;pre jsonb;sha text;k jsonb;claves jsonb;root vec_autorizacion_atestada_v3.raiz_confianza_version;anterior record;v_confirmacion record;previo record;
 secret bytea;ord bigint;i integer:=0;claves_sha text;ahora timestamptz;n integer;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD198: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000';END IF;
 IF octet_length(p_plan)>16384 OR octet_length(p_material)>32768 THEN RAISE EXCEPTION 'AD198: PARO clave=entrada actual=grande esperado=acotada' USING ERRCODE='22023';END IF;
 SELECT * INTO STRICT s FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=c.conjunto_version;
 n:=cardinality(s.audiencias);
 sha:=encode(sha256(convert_to(p_plan,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_aprobado OR sha IS DISTINCT FROM c.plan_sha256
 OR encode(sha256(convert_to(p_material,'UTF8')),'hex') IS DISTINCT FROM c.material_sha256
 THEN RAISE EXCEPTION 'AD198: PARO clave=huella actual=distinta esperado=aprobada' USING ERRCODE='42501';END IF;
 p:=p_plan::jsonb;m:=p_material::jsonb;g:=p->'configuracion';claves:=m->'claves';
 IF jsonb_typeof(p)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>8
 OR NOT p ?& ARRAY['version','operacion_ref','preparado_en','caduca_en','preimagen_sha256','conjunto_version','configuracion','clave_ordenes']
 OR p->>'version'<>'2' OR p->'conjunto_version' IS DISTINCT FROM to_jsonb(c.conjunto_version)
 OR p->>'operacion_ref' !~ '^gca_[A-Za-z0-9_-]{22,124}$'
 OR p->>'preimagen_sha256' IS DISTINCT FROM c.preimagen_sha256
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp()
 OR jsonb_typeof(g)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(g))<>5 OR NOT g ?& ARRAY['revision','secuencia','huella_sha256','publicada_en','expira_en']
 OR jsonb_typeof(m)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>1
 OR jsonb_typeof(claves)<>'array' OR jsonb_array_length(claves)<>n
 OR jsonb_typeof(p->'clave_ordenes')<>'array' OR jsonb_array_length(p->'clave_ordenes')<>n
 THEN RAISE EXCEPTION 'AD198: PARO clave=plan actual=incompatible esperado=ABI198' USING ERRCODE='22023';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Mismo cerrojo que AD188 y los publicadores del gobierno; además CAS de tablas.
 LOCK TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual,vec_autorizacion_atestada_v3.puntero_clave_emision IN SHARE ROW EXCLUSIVE MODE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1();
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AD198: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 SELECT * INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='gobierno_usuarios_admin' AND plan_sha256=sha;
 IF FOUND THEN
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p,m);
  RETURN jsonb_build_object('auditoria_ref',previo.auditoria_ref,'secuencia',previo.secuencia,'huella_sha256',previo.huella_sha256,'correlacion_ref',previo.correlacion_ref,'registrada_en',previo.registrada_en,'replay',true,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',previo.gobierno_usuarios_detalle->>'claves_sha256','configuracion_ref',g->>'revision');
 END IF;
 pre:=vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(c.conjunto_version);
 IF encode(sha256(convert_to(pre::text,'UTF8')),'hex') IS DISTINCT FROM c.preimagen_sha256 THEN RAISE EXCEPTION 'AD198: PARO clave=preimagen actual=distinta esperado=aprobada' USING ERRCODE='40001';END IF;
 SELECT x.* INTO STRICT anterior FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision ORDER BY a.orden DESC LIMIT 1;
 SELECT x.* INTO STRICT root FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=anterior.revision;
 ahora:=clock_timestamp();
 IF root.valida_desde>ahora OR root.valida_hasta<=ahora OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r WHERE r.raiz_clave_id=root.clave_id AND r.raiz_version=root.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r WHERE r.configuracion_revision=anterior.revision)
 OR (g->>'publicada_en')::timestamptz<>date_trunc('day',ahora) OR (g->>'expira_en')::timestamptz<>(g->>'publicada_en')::timestamptz+interval '1 day'
 OR g->>'revision' !~ '^confianza:atestacion:ct:desarrollo:[0-9]{4}-[0-9]{2}-[0-9]{2}(:r[0-9]+)?$'
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(g,root) IS DISTINCT FROM g->>'huella_sha256'
 OR g->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR (g->>'secuencia')::bigint<=anterior.secuencia
 OR (g->>'secuencia')::bigint<=(pre->>'orden_configuracion')::bigint
 THEN RAISE EXCEPTION 'AD198: PARO clave=renovacion actual=incompatible esperado=misma_raiz_diaria_aprobada' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(claves) LOOP
  i:=i+1;
  IF jsonb_typeof(p->'clave_ordenes'->(i-1))<>'number' THEN RAISE EXCEPTION 'AD198: PARO clave=orden actual=no_numerica esperado=entero' USING ERRCODE='22023';END IF;
  ord:=(p->'clave_ordenes'->>(i-1))::bigint;
  IF i>1 AND ord<=(p->'clave_ordenes'->>(i-2))::bigint THEN RAISE EXCEPTION 'AD198: PARO clave=orden actual=no_creciente esperado=creciente' USING ERRCODE='22023';END IF;
  IF jsonb_typeof(k)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(k))<>10
  OR NOT k ?& ARRAY['audiencia','clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac','huella_secreto_sha256','emisor_id','valida_desde','valida_hasta']
  OR k->>'audiencia' IS DISTINCT FROM s.audiencias[i]
  OR k->>'clave_id' !~ '^clave:capacidad:admin:[a-z0-9:._-]{1,160}$'
  OR left(k->>'clave_id',length('clave:capacidad:admin:'||s.segmentos[i]||':')) IS DISTINCT FROM 'clave:capacidad:admin:'||s.segmentos[i]||':'
  OR k->>'emisor_id' !~ '^emisor:admin:[a-z0-9:._-]{1,120}$'
  OR (k->>'version')::bigint NOT BETWEEN 1 AND 9007199254740991 OR (k->>'revision_gobierno')::bigint NOT BETWEEN 1 AND 9007199254740991
  OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora OR (k->>'valida_hasta')::timestamptz>root.valida_hasta
  OR ord<=(pre->>'orden_claves')::bigint OR ord>9007199254740991
  OR k->>'huella_gobierno_sha256' !~ '^[0-9a-f]{64}$' OR k->>'huella_secreto_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AD198: PARO clave=clave actual=incompatible esperado=audiencia_del_conjunto_aprobada' USING ERRCODE='22023';END IF;
  secret:=decode(k->>'secreto_hmac','base64');
  IF octet_length(secret)<>32 OR encode(sha256(secret),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' OR x.secreto_hmac=secret)
  THEN RAISE EXCEPTION 'AD198: PARO clave=material actual=incompatible esperado=nuevo_dedicado_32bytes' USING ERRCODE='42501';END IF;
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::bigint,(k->>'revision_gobierno')::bigint,k->>'huella_gobierno_sha256',secret,k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia',(k->>'valida_desde')::timestamptz,(k->>'valida_hasta')::timestamptz,'acto_tecnico:admin:capacidades:clave:'||sha||':'||i::text);
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) VALUES(ord,k->>'clave_id',(k->>'version')::bigint,ahora,'acto_tecnico:admin:capacidades:puntero:'||sha||':'||i::text);
 END LOOP;
 -- Sólo configuración y enlace/puntero nuevos; raíz original intacta.
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(g->>'revision',(g->>'secuencia')::bigint,g->>'huella_sha256',(g->>'publicada_en')::timestamptz,(g->>'expira_en')::timestamptz,'acto:ct:desarrollo:configuracion:r'||(g->>'secuencia'));
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES(g->>'revision',root.clave_id,root.version);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)
 VALUES((g->>'secuencia')::bigint,g->>'revision',(g->>'publicada_en')::timestamptz,'acto:ct:desarrollo:puntero-configuracion:r'||(g->>'secuencia'));
 claves_sha:=encode(sha256(convert_to((SELECT jsonb_agg(value-'secreto_hmac') FROM jsonb_array_elements(claves))::text,'UTF8')),'hex');
 SELECT * INTO v_confirmacion FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','gobierno_usuarios_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'configuracion_origen_ref',anterior.revision,'configuracion_destino_ref',g->>'revision','claves_sha256',claves_sha,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||substr(sha,1,32)));
 INSERT INTO vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1(auditoria_ref,tipo,funcion,operador_login,solicitud_sha256,conjunto_version,resultado,clave_ids)
 VALUES(v_confirmacion.auditoria_ref,'confirmacion','aprovisionar_gobierno_capacidades_admin_v1',session_user,sha,c.conjunto_version,'permitido',
  ARRAY(SELECT x->>'clave_id' FROM jsonb_array_elements(claves) x));
 PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p,m);
 RETURN to_jsonb(v_confirmacion)||jsonb_build_object('replay',false,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',claves_sha,'configuracion_ref',g->>'revision');
END $f$;

-- Envoltura: intento permitido o, si el efecto falla, el efecto se revierte y
-- queda sólo el intento denegado o de error. Los intentos usan el registro de
-- AD188 (su acción y recurso son los de ese registro).
CREATE FUNCTION vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(p_plan text,p_aprobado text,p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE recibo jsonb;intento record;estado text:='permitido';motivo text:='gobierno_usuarios_registrado';codigo text:='gobierno_usuarios_registrado';solicitud text;conjunto integer;
BEGIN
 solicitud:=encode(sha256(convert_to(coalesce(p_plan,''),'UTF8')),'hex');
 -- Conjunto aprobado para este LOGIN, si lo hay (también para el intento denegado).
 SELECT x.conjunto_version INTO conjunto FROM vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 x WHERE x.identidad_login=session_user;
 BEGIN
  recibo:=vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(p_plan,p_aprobado,p_material);
  IF recibo->>'replay'='true' THEN motivo:='gobierno_usuarios_replay';codigo:=motivo;END IF;
  SELECT * INTO intento FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','intento_gobierno_usuarios_admin','evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),'operador_login',session_user::text,'solicitud_sha256',solicitud,'accion','aprovisionar_gobierno_usuarios_admin_v1','recurso_ref','solicitud_gobierno_usuarios:'||substr(solicitud,1,32),'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||replace(gen_random_uuid()::text,'-','')));
  INSERT INTO vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1(auditoria_ref,tipo,funcion,operador_login,solicitud_sha256,conjunto_version,resultado,clave_ids)
  VALUES(intento.auditoria_ref,'intento','aprovisionar_gobierno_capacidades_admin_v1',session_user,solicitud,conjunto,'permitido',ARRAY(SELECT x->>'clave_id' FROM jsonb_array_elements(p_material::jsonb->'claves') x));
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(p_plan::jsonb,p_material::jsonb);
 EXCEPTION WHEN insufficient_privilege OR serialization_failure OR invalid_parameter_value OR invalid_text_representation OR datetime_field_overflow OR unique_violation OR no_data_found THEN
  estado:='denegado';motivo:='gobierno_usuarios_denegado';codigo:=motivo;recibo:=NULL;
 WHEN OTHERS THEN estado:='error';motivo:='gobierno_usuarios_error';codigo:=motivo;recibo:=NULL;
 END;
 IF estado<>'permitido' THEN
  SELECT * INTO intento FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','intento_gobierno_usuarios_admin','evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),'operador_login',session_user::text,'solicitud_sha256',solicitud,'accion','aprovisionar_gobierno_usuarios_admin_v1','recurso_ref','solicitud_gobierno_usuarios:'||substr(solicitud,1,32),'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||replace(gen_random_uuid()::text,'-','')));
  -- Sin efecto: el intento denegado o con error queda anotado como de AD198.
  INSERT INTO vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1(auditoria_ref,tipo,funcion,operador_login,solicitud_sha256,conjunto_version,resultado,clave_ids)
  VALUES(intento.auditoria_ref,'intento','aprovisionar_gobierno_capacidades_admin_v1',session_user,solicitud,conjunto,estado,'{}');
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',recibo,'auditoria_intento',to_jsonb(intento)||jsonb_build_object('solicitud_sha256',solicitud));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1(),
 vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer),
 vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(jsonb,jsonb),
 vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text),
 vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_gobierno_capacidades_admin_operador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text) TO vec_gobierno_capacidades_admin_operador;
RESET ROLE;
DO $acl$
DECLARE g oid:='vec_gobierno_capacidades_admin_operador'::regrole;f text;
BEGIN
 FOREACH f IN ARRAY ARRAY['vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1()',
  'vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)',
  'vec_autorizacion_atestada_v3.revalidar_gobierno_capacidades_admin_v1(jsonb,jsonb)',
  'vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'] LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f::regprocedure AND a.grantee<>p.proowner)
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f::regprocedure AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
  THEN RAISE EXCEPTION 'AD198: PARO clave=ACL_privada actual=ampliada esperado=solo_propietario %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid='vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure
   AND (a.grantee NOT IN(p.proowner,g) OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 OR NOT has_function_privilege(g,'vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid IN('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass,'vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1'::regclass,'vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1'::regclass)
   AND a.grantee<>c.relowner)
 THEN RAISE EXCEPTION 'AD198: PARO clave=ACL actual=ampliada esperado=operador_solo_aprovisionar' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
