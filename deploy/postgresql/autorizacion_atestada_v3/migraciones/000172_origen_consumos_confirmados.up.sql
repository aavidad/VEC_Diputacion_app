\set ON_ERROR_STOP on
-- AD172: origen del eslabón de consumos nuevos; material V3 y ABI intactos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000172',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NOT NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND c.conname='auditoria_tipo_disjunto_v2' AND c.contype='c' AND c.convalidated)
 THEN RAISE EXCEPTION 'AD172: PARO clave=preimagen actual=incompatible esperado=AD171_sin_AD172' USING ERRCODE='55000'; END IF;
END $pre$;

-- Esta tabla sólo identifica procesos técnicos. No concede acciones ni perfiles.
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 (
 login_nombre name NOT NULL CHECK (login_nombre::text ~ '^[a-zA-Z_][a-zA-Z0-9_-]{0,62}$'),
 audiencia_consumo text NOT NULL CHECK (audiencia_consumo ~ '^[a-z][a-z0-9._:-]*$' AND length(audiencia_consumo)<=512),
 operacion text NOT NULL CHECK (operacion ~ '^[a-z][a-z0-9._:-]{0,159}$'),
 proceso text NOT NULL CHECK (proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 canal_permitido text NOT NULL CHECK (canal_permitido IN (
   'interna_corporativa','administracion_privilegiada','externa_personal')),
 configurada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(login_nombre,audiencia_consumo,operacion)
);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON
 vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON
 vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
ALTER TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 FOR ALL TO vec_autorizacion_atestada_v3_propietario
 USING(current_user='vec_autorizacion_atestada_v3_propietario')
 WITH CHECK(current_user='vec_autorizacion_atestada_v3_propietario');
REVOKE ALL ON vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
 p_audiencia text,p_operacion text,p_canal text) RETURNS text
LANGUAGE sql STABLE SECURITY INVOKER PARALLEL UNSAFE SET search_path=pg_catalog
AS $resolver$
 SELECT c.proceso
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c
 JOIN pg_catalog.pg_roles l ON l.rolname=c.login_nombre
 WHERE current_user='vec_autorizacion_atestada_v3_propietario'
  AND pg_catalog.current_setting('role')='none'
  AND l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
  AND NOT l.rolsuper AND NOT l.rolcreaterole AND NOT l.rolcreatedb
  AND NOT l.rolreplication AND NOT l.rolbypassrls
  AND c.audiencia_consumo=p_audiencia AND c.operacion=p_operacion
  AND c.canal_permitido=p_canal
$resolver$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text) FROM PUBLIC;

LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD COLUMN version_consumo smallint;
-- Conserva literalmente la condición AD171; ninguna fila histórica se actualiza.
DO $familias$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_tipo_disjunto_v2' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior,'UTF8')),'hex')
      IS DISTINCT FROM 'f31b31dc0ec40bdd2d7a6930346210e919ab4aed225352235723525a27e7dc29'
 OR pg_catalog.left(anterior,7)<>'CHECK (' OR pg_catalog.right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD172: PARO clave=CHECK actual=incompatible esperado=CHECK_validado' USING ERRCODE='55000'; END IF;
 nueva:='CHECK ((version_consumo IS NULL AND ('||pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||')) OR ('||
 $v2$tipo_registro='consumo_confirmado_v2' AND version_consumo IS NOT NULL AND version_consumo=2
 AND decision_ref IS NOT NULL AND efecto_ref IS NOT NULL AND huella_efecto_sha256 IS NOT NULL
 AND proceso IS NOT NULL AND canal IS NOT NULL
 AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$'
 AND canal IN ('interna_corporativa','administracion_privilegiada','externa_personal')
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL
 AND actor_ref IS NULL AND perfil_activo_ref IS NULL AND registro_contexto_ref IS NULL
 AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL AND autenticacion_ref IS NULL
 AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND accion IS NULL
 AND modulo_id IS NULL AND recurso_ref IS NULL AND finalidad_ref IS NULL
 AND resultado IS NULL AND motivo_ref IS NULL AND correlacion_ref IS NULL AND vinculo_sha256 IS NULL
 AND evento_ref IS NULL AND evento_material_sha256 IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL
 AND operador_login IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL$v2$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v2;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v3 '||nueva;
END $familias$;

DO $nucleo$
DECLARE
 f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;revertida text;
 def_sha text;src_sha text;post_def text;post_src text;
 meta jsonb;acl aclitem[];propietario oid;config text[];definidora boolean;
 deps jsonb;compartidas jsonb;
 antiguo1 text:=$antiguo1$    v_preimagen_auditoria bytea;$antiguo1$;
 nuevo1 text:=$nuevo1$    v_preimagen_auditoria bytea;
    v_proceso_origen text;
    v_canal_origen text;$nuevo1$;
 antiguo2 text:=$antiguo2$        RETURN;
    END IF;
    v_ahora := clock_timestamp();
    SELECT k.* INTO v_clave$antiguo2$;
 nuevo2 text:=$nuevo2$        RETURN;
    END IF;
    -- AD172: configuración técnica; no amplía la autorización de negocio.
    -- Se exige sólo al consumo nuevo; la autoridad viva original valida después
    -- el vínculo canónico antes de insertar atestación, consumo y auditoría.
    v_canal_origen := d #>> '{vinculo_autenticacion_actor,superficie}';
    v_proceso_origen := vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
        c ->> 'audiencia_consumo', c ->> 'operacion', v_canal_origen);
    IF v_proceso_origen IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='origen de consumo no acreditado';
    END IF;
    v_ahora := clock_timestamp();
    SELECT k.* INTO v_clave$nuevo2$;
 antiguo3 text:=$antiguo3$    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'decision_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_huella_consumo);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora);$antiguo3$;
 nuevo3 text:=$nuevo3$    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac('consumo_confirmado_v2') ||
        vec_autorizacion_atestada_v3.encuadrar_mac('2') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'decision_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_huella_consumo) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_proceso_origen) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_canal_origen);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en, tipo_registro, version_consumo, proceso, canal) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora,
        'consumo_confirmado_v2', 2, v_proceso_origen, v_canal_origen);$nuevo3$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,fuente,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 def_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex');
 src_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex');
 SELECT v.post_def,v.post_src INTO post_def,post_src FROM (VALUES
  ('0659ad9bf01278e8373225c3415fe82c7c69b6058f7a537c8ff47aa589955156','5b9d8a87caec44e4b7abd1273f52c2d397c398cfe204a0cf4d9697bb8aee983d','92b4245448ffb76aa0792788616e5a779ddd867612e4bd28ab106d95d5356446','54327be7e866b84d0cd1feff3a54ef6fc58ceec92daaa2277b150da984371fe9'),
  ('00fdab71ff0477cbe3fb1dcabcde7377da857d20578b9be478339d031ac034ce','1c4a33b316fe58454c76c207db1722504d69a2de21e5c4fb32c73a4cffc20fe4','32939b59240dab122779c10789c8fca3d5bcd4024b6341b477e61d82020d8c10','24c003368a265ab7622e87b0c1949f0f13ba0b585a6624ff0729ecce88de066e')
 ) v(pre_def,pre_src,post_def,post_src) WHERE v.pre_def=def_sha AND v.pre_src=src_sha;
 IF NOT FOUND
 OR (def_sha='0659ad9bf01278e8373225c3415fe82c7c69b6058f7a537c8ff47aa589955156'
     AND pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL)
 OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
 OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_catalog.aclexplode(coalesce(acl,pg_catalog.acldefault('f',propietario))))<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(acl,pg_catalog.acldefault('f',propietario))) a
   WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD172: PARO clave=nucleo_sha256 actual=%/% esperado=POST154_o_POST168_exactos',def_sha,src_sha USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database());
 nueva:=original;
 IF length(nueva)-length(replace(nueva,antiguo1,''))<>length(antiguo1)
 THEN RAISE EXCEPTION 'AD172: PARO clave=bloque_1 actual=no_unico esperado=una_coincidencia' USING ERRCODE='55000'; END IF;
 nueva:=replace(nueva,antiguo1,nuevo1);
 IF length(nueva)-length(replace(nueva,antiguo2,''))<>length(antiguo2)
 THEN RAISE EXCEPTION 'AD172: PARO clave=bloque_2 actual=no_unico esperado=una_coincidencia' USING ERRCODE='55000'; END IF;
 nueva:=replace(nueva,antiguo2,nuevo2);
 IF length(nueva)-length(replace(nueva,antiguo3,''))<>length(antiguo3)
 THEN RAISE EXCEPTION 'AD172: PARO clave=bloque_3 actual=no_unico esperado=una_coincidencia' USING ERRCODE='55000'; END IF;
 nueva:=replace(nueva,antiguo3,nuevo3);
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM post_def
 THEN RAISE EXCEPTION 'AD172: PARO clave=postimagen_sha256 actual=% esperado=%',encode(sha256(convert_to(nueva,'UTF8')),'hex'),post_def USING ERRCODE='55000'; END IF;
 EXECUTE nueva;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 revertida:=actual;
 revertida:=replace(revertida,nuevo3,antiguo3);
 revertida:=replace(revertida,nuevo2,antiguo2);
 revertida:=replace(revertida,nuevo1,antiguo1);
 IF actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
 OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM post_def
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM post_src
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
      AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD172: PARO clave=delta_o_metadatos actual=divergente esperado=solo_tres_bloques' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
