\set ON_ERROR_STOP on
-- AD173: fecha y coordenadas nominales en el nuevo eslabón común.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000173',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
    AND c.conname='auditoria_tipo_disjunto_v3' AND c.contype='c' AND c.convalidated)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
    AND c.conname='auditoria_tipo_disjunto_v4')
 THEN RAISE EXCEPTION 'AD173: PARO clave=preimagen actual=incompatible esperado=AD172_sin_AD173' USING ERRCODE='55000'; END IF;
END $pre$;

-- Reutiliza actor/perfil/finalidad AD169. consumida_en permanece en su tabla
-- original, ligada por la FK exacta existente; la proyección debe unirla.
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familias$
DECLARE anterior text;nueva text;actual_sha text;
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_tipo_disjunto_v3' AND c.contype='c' AND c.convalidated;
 actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(anterior,'UTF8')),'hex');
 IF actual_sha IS DISTINCT FROM '55603c422c7c752909d558d8dd238e371719c1e9d0626272e955cde21240c282'
 OR pg_catalog.left(anterior,7)<>'CHECK (' OR pg_catalog.right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD173: PARO clave=CHECK_SHA256 actual=% esperado=55603c422c7c752909d558d8dd238e371719c1e9d0626272e955cde21240c282',actual_sha USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||') OR ('||
 $v3$tipo_registro='consumo_confirmado_v3' AND version_consumo IS NOT NULL AND version_consumo=3
 AND decision_ref IS NOT NULL AND efecto_ref IS NOT NULL AND huella_efecto_sha256 IS NOT NULL
 AND proceso IS NOT NULL AND canal IS NOT NULL
 AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$'
 AND canal IN ('interna_corporativa','administracion_privilegiada','externa_personal')
 AND actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL AND finalidad_ref IS NOT NULL
 AND octet_length(actor_ref) BETWEEN 1 AND 512
 AND octet_length(perfil_activo_ref) BETWEEN 1 AND 512
 AND octet_length(finalidad_ref) BETWEEN 1 AND 512
 AND isfinite(registrada_en)
 AND registrada_en >= '0001-01-01T00:00:00Z'::timestamptz
 AND registrada_en < '10000-01-01T00:00:00Z'::timestamptz
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL
 AND accion IS NULL AND modulo_id IS NULL AND recurso_ref IS NULL
 AND resultado IS NULL AND motivo_ref IS NULL AND correlacion_ref IS NULL AND vinculo_sha256 IS NULL
 AND evento_ref IS NULL AND evento_material_sha256 IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL
 AND operador_login IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL$v3$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v3;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familias$;

DO $nucleo$
DECLARE
 f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;revertida text;
 def_sha text;src_sha text;meta jsonb;deps jsonb;compartidas jsonb;
 variante text;post_def text;post_src text;
 antiguo1 text:=$antiguo1$    v_canal_origen text;$antiguo1$;
 nuevo1 text:=$nuevo1$    v_canal_origen text;
    v_actor_nominal text;
    v_perfil_nominal text;
    v_finalidad_nominal text;
    v_fecha_nominal text;$nuevo1$;
 antiguo2 text:=$antiguo2$    v_preimagen_auditoria :=
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
        'consumo_confirmado_v2', 2, v_proceso_origen, v_canal_origen);$antiguo2$;
 nuevo2 text:=$nuevo2$    -- AD173: coordenadas de la decisión/contexto ya autenticados y revalidados.
    -- El instante guardado en ambas tablas procede del mismo v_ahora UTC6.
    v_actor_nominal := d ->> 'principal_id';
    v_perfil_nominal := d ->> 'perfil_activo_ref';
    v_finalidad_nominal := d ->> 'finalidad';
    v_fecha_nominal := pg_catalog.to_char(v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac('consumo_confirmado_v3') ||
        vec_autorizacion_atestada_v3.encuadrar_mac('3') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'decision_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_huella_consumo) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_proceso_origen) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_canal_origen) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_fecha_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_fecha_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_actor_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_perfil_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_finalidad_nominal);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en, tipo_registro, version_consumo, proceso, canal,
        actor_ref, perfil_activo_ref, finalidad_ref) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora,
        'consumo_confirmado_v3', 3, v_proceso_origen, v_canal_origen,
        v_actor_nominal, v_perfil_nominal, v_finalidad_nominal);$nuevo2$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),p.prosrc,pg_catalog.to_jsonb(p)-'prosrc'
 INTO STRICT original,fuente,meta FROM pg_catalog.pg_proc p WHERE p.oid=f;
 def_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex');
 src_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex');
 -- Parejas completas medidas, no aceptación por anclas ni nombre del objeto.
 IF def_sha='92b4245448ffb76aa0792788616e5a779ddd867612e4bd28ab106d95d5356446'
 AND src_sha='54327be7e866b84d0cd1feff3a54ef6fc58ceec92daaa2277b150da984371fe9' THEN
  variante:='POST154_AD172';
  post_def:='777f6a6e94c57cfdd8516d082441c1662e303c4b11aa1f187a542a9a11eeb2d6';
  post_src:='528f35de95885283cc9987fc6f2ca8f09db59e61672b10f3c69134b1d91c23f6';
  IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  THEN RAISE EXCEPTION 'AD173: PARO clave=POST154_fachada_rpt actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 ELSIF def_sha='32939b59240dab122779c10789c8fca3d5bcd4024b6341b477e61d82020d8c10'
 AND src_sha='24c003368a265ab7622e87b0c1949f0f13ba0b585a6624ff0729ecce88de066e' THEN
  variante:='POST168_AD172';
  post_def:='6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9';
  post_src:='bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c';
 ELSE
  RAISE EXCEPTION 'AD173: PARO clave=nucleo_sha256 actual=%/% esperado=POST154_AD172_o_POST168_AD172',def_sha,src_sha USING ERRCODE='55000';
 END IF;
 IF (meta->>'proowner')::oid<>'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (meta->>'prosecdef')::boolean IS NOT TRUE
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD173: PARO clave=nucleo_metadatos actual=incompatible esperado=propietario_config_ACL_exactos' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=original;
 IF length(nueva)-length(replace(nueva,antiguo1,''))<>length(antiguo1)
 THEN RAISE EXCEPTION 'AD173: PARO clave=bloque_1 actual=no_unico esperado=una_coincidencia' USING ERRCODE='55000'; END IF;
 nueva:=replace(nueva,antiguo1,nuevo1);
 IF length(nueva)-length(replace(nueva,antiguo2,''))<>length(antiguo2)
 THEN RAISE EXCEPTION 'AD173: PARO clave=bloque_2 actual=no_unico esperado=una_coincidencia' USING ERRCODE='55000'; END IF;
 nueva:=replace(nueva,antiguo2,nuevo2);
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM post_def
 THEN RAISE EXCEPTION 'AD173: PARO clave=postimagen_sha256 actual=% esperado=%',encode(sha256(convert_to(nueva,'UTF8')),'hex'),post_def USING ERRCODE='55000'; END IF;
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 revertida:=replace(replace(actual,nuevo2,antiguo2),nuevo1,antiguo1);
 IF actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
 OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM post_def
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM post_src
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD173: PARO clave=delta_o_metadatos actual=divergente esperado=solo_dos_bloques' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
