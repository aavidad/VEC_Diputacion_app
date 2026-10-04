\set ON_ERROR_STOP on
-- AD193 prospectiva: preimágenes medidas en copia fría POST184/185/192; revisión/ensayo causal pendientes.
-- Las huellas esperadas son literales; no se calculan para aprobar el destino.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000193',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE nombre text;f oid;etiqueta text;actual text;
 nucleo_def text:='536ea653143147e0cfb2d3277948530acb8d4d1643e44f4786462fe5ad1379fe'; -- medida en copia fría POST192, pg_get_functiondef
 nucleo_src text:='b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45'; -- medida en copia fría POST192, pg_proc.prosrc
 helper_def text:='1930a2da7f948cac8e44126c768c256b3d25c9b9e8f9ed39bc32ee4ee5d4c725'; -- comprobador conservado desde POST173, pg_get_functiondef
 helper_src text:='f0d1b453d4f594e14191750c6dcde2b6727aa71559cb9c344b0a1621185f7d6f'; -- comprobador conservado desde POST173, pg_proc.prosrc
 check_sha text:='0f6d15ebdc61ba5ff67903bde824db878a6396fa593029e98498946e2d8d1331'; -- medida POST192, pg_get_constraintdef(false)
 esperado_def text;esperado_src text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 THEN RAISE EXCEPTION 'AD193: PARO clave=esquema_base esperado=PG18_propietario_tablas_consumo_auditoria actual=incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['consumo_decision_v3','auditoria_consumo_v3'] LOOP
  IF EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('vec_autorizacion_atestada_v3.'||nombre)
   AND attname='transaccion_origen' AND NOT attisdropped)
  THEN RAISE EXCEPTION 'AD193: PARO clave=columna_transaccion_origen_% esperado=ausente actual=presente',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH etiqueta IN ARRAY ARRAY['nucleo','helper'] LOOP
  f:=CASE etiqueta WHEN 'nucleo' THEN to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
   ELSE to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)') END;
  IF f IS NULL THEN RAISE EXCEPTION 'AD193: PARO clave=funcion_% esperado=presente actual=ausente',etiqueta USING ERRCODE='55000'; END IF;
  esperado_def:=CASE etiqueta WHEN 'nucleo' THEN nucleo_def ELSE helper_def END;
  esperado_src:=CASE etiqueta WHEN 'nucleo' THEN nucleo_src ELSE helper_src END;
  actual:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
  IF esperado_def !~ '^[0-9a-f]{64}$' OR esperado_def IS DISTINCT FROM actual
  THEN RAISE EXCEPTION 'AD193: PARO clave=def_sha_% esperado=% actual=%',etiqueta,esperado_def,actual USING ERRCODE='55000'; END IF;
  SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO STRICT actual FROM pg_proc WHERE oid=f;
  IF esperado_src !~ '^[0-9a-f]{64}$' OR esperado_src IS DISTINCT FROM actual
  THEN RAISE EXCEPTION 'AD193: PARO clave=src_sha_% esperado=% actual=%',etiqueta,esperado_src,actual USING ERRCODE='55000'; END IF;
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=current_user::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=(CASE etiqueta WHEN 'nucleo' THEN ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    ELSE ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s','TimeZone=UTC'] END))
  OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)
    <> (CASE etiqueta WHEN 'nucleo' THEN 1 ELSE 2 END)
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner
    OR NOT(a.grantee=p.proowner OR (etiqueta='helper' AND a.grantee='vec_autorizacion_propietario'::regrole))))
  THEN RAISE EXCEPTION 'AD193: PARO clave=metadatos_ACL_% esperado=propietario_config_ACL_exactos actual=incompatible',etiqueta USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF check_sha !~ '^[0-9a-f]{64}$' OR check_sha IS DISTINCT FROM actual
 THEN RAISE EXCEPTION 'AD193: PARO clave=CHECK_sha esperado=% actual=%',check_sha,actual USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_autorizacion_atestada_v3.consumo_decision_v3,
 vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.consumo_decision_v3 ADD COLUMN transaccion_origen xid8;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD COLUMN transaccion_origen xid8;
ALTER TABLE vec_autorizacion_atestada_v3.consumo_decision_v3 ADD CONSTRAINT consumo_transaccion_origen_positivo_v4
 CHECK(transaccion_origen IS NULL OR transaccion_origen>'0'::xid8);
DO $familia$
DECLARE anterior text;nueva text;v4 text;marca text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4';
 IF left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD193: PARO clave=CHECK_forma esperado=CHECK_parentesis actual=incompatible' USING ERRCODE='55000'; END IF;
 v4:=substr(anterior,8,length(anterior)-8);
 FOREACH marca IN ARRAY ARRAY['''consumo_confirmado_v3''::text','version_consumo = 3'] LOOP
  IF length(v4)-length(replace(v4,marca,''))<>length(marca)
  THEN RAISE EXCEPTION 'AD193: PARO clave=CHECK_marca_% esperado=1 actual=%',marca,(length(v4)-length(replace(v4,marca,'')))/length(marca) USING ERRCODE='55000'; END IF;
 END LOOP;
 v4:=replace(replace(v4,'''consumo_confirmado_v3''::text','''consumo_confirmado_v4''::text'),'version_consumo = 3','version_consumo = 4');
 nueva:='CHECK ((transaccion_origen IS NULL AND ('||substr(anterior,8,length(anterior)-8)||')) OR ('||
  'transaccion_origen IS NOT NULL AND transaccion_origen > ''0''::xid8 AND tipo_registro IS NOT DISTINCT FROM ''consumo_confirmado_v4'' AND ('||v4||')))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;nueva text;actual text;revertida text;meta jsonb;deps jsonb;compartidas jsonb;
 antiguos text[]:=ARRAY[$a0$    v_fecha_nominal text;$a0$,$a1$    INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3 (
        decision_ref, huella_decision_sha256, nonce, efecto_ref,
        huella_efecto_sha256, consumo_huella_sha256, consumida_en) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        c ->> 'nonce', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo, v_ahora);
$a1$,$a2$    v_preimagen_auditoria :=
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
        v_actor_nominal, v_perfil_nominal, v_finalidad_nominal);
$a2$];
 nuevos text[]:=ARRAY[$b0$    v_fecha_nominal text;
    v_transaccion_origen xid8;$b0$,$b1$    v_transaccion_origen := pg_catalog.pg_current_xact_id();
    INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3 (
        decision_ref, huella_decision_sha256, nonce, efecto_ref,
        huella_efecto_sha256, consumo_huella_sha256, consumida_en, transaccion_origen) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        c ->> 'nonce', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo, v_ahora, v_transaccion_origen);
$b1$,$b2$    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac('consumo_confirmado_v4') ||
        vec_autorizacion_atestada_v3.encuadrar_mac('4') ||
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
        vec_autorizacion_atestada_v3.encuadrar_mac(v_finalidad_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_transaccion_origen::text);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en, tipo_registro, version_consumo, proceso, canal,
        actor_ref, perfil_activo_ref, finalidad_ref, transaccion_origen) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora,
        'consumo_confirmado_v4', 4, v_proceso_origen, v_canal_origen,
        v_actor_nominal, v_perfil_nominal, v_finalidad_nominal, v_transaccion_origen);
$b2$];
 i integer;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=original;
 FOR i IN 1..array_length(antiguos,1) LOOP
  IF length(nueva)-length(replace(nueva,antiguos[i],''))<>length(antiguos[i])
  THEN RAISE EXCEPTION 'AD193: PARO clave=bloque_% esperado=1 actual=%',i,(length(nueva)-length(replace(nueva,antiguos[i],'')))/length(antiguos[i]) USING ERRCODE='55000'; END IF;
  nueva:=replace(nueva,antiguos[i],nuevos[i]);
 END LOOP;
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 revertida:=actual;
 FOR i IN REVERSE array_length(antiguos,1)..1 LOOP
  IF length(revertida)-length(replace(revertida,nuevos[i],''))<>length(nuevos[i])
  THEN RAISE EXCEPTION 'AD193: PARO clave=bloque_nuevo_% esperado=1 actual=%',i,(length(revertida)-length(replace(revertida,nuevos[i],'')))/length(nuevos[i]) USING ERRCODE='55000'; END IF;
  revertida:=replace(revertida,nuevos[i],antiguos[i]);
 END LOOP;
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM 'f581dbf9aa01d454caa6906ece774f97cf16cca9e9b8910d0eb348e23ef8c34b'
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'bb21afce73af87d532574c99da4f3ea8cd0534edc4910f14018d9ee55eb2a79b'
 OR actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD193: PARO clave=delta_metadatos esperado=postimagenes_reversion_OID_ACL_dependencias_exactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;
DO $helper$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)');
 original text;nueva text;actual text;revertida text;meta jsonb;deps jsonb;compartidas jsonb;
 antiguos text[]:=ARRAY[$a0$   a.xmin AS consumo_xmin,u.xmin AS auditoria_xmin$a0$,$a1$  OR r.consumo_xmin IS DISTINCT FROM pg_current_xact_id()::xid
  OR r.auditoria_xmin IS DISTINCT FROM pg_current_xact_id()::xid$a1$,$a2$ -- Las dos filas nuevas deben haber sido insertadas por esta transacción.
 -- transaction_timestamp por sí solo no prueba eso si BEGIN precede al
 -- primer snapshot SERIALIZABLE y otro COMMIT ocurre entre ambos.$a2$];
 nuevos text[]:=ARRAY[$b0$   a.transaccion_origen AS consumo_transaccion_origen,
   u.transaccion_origen AS auditoria_transaccion_origen$b0$,$b1$  OR r.consumo_transaccion_origen IS DISTINCT FROM pg_current_xact_id()
  OR r.auditoria_transaccion_origen IS DISTINCT FROM pg_current_xact_id()$b1$,$b2$ -- Ambos sellos completos deben pertenecer al TopXID actual.
 -- SAVEPOINT y EXCEPTION pueden insertar con xmin de un SubXID distinto.
 -- Un recibo histórico o de otra transacción no acredita este consumo.$b2$];
 i integer;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=original;
 FOR i IN 1..array_length(antiguos,1) LOOP
  IF length(nueva)-length(replace(nueva,antiguos[i],''))<>length(antiguos[i])
  THEN RAISE EXCEPTION 'AD193: PARO clave=bloque_% esperado=1 actual=%',i,(length(nueva)-length(replace(nueva,antiguos[i],'')))/length(antiguos[i]) USING ERRCODE='55000'; END IF;
  nueva:=replace(nueva,antiguos[i],nuevos[i]);
 END LOOP;
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 revertida:=actual;
 FOR i IN REVERSE array_length(antiguos,1)..1 LOOP
  IF length(revertida)-length(replace(revertida,nuevos[i],''))<>length(nuevos[i])
  THEN RAISE EXCEPTION 'AD193: PARO clave=bloque_nuevo_% esperado=1 actual=%',i,(length(revertida)-length(replace(revertida,nuevos[i],'')))/length(nuevos[i]) USING ERRCODE='55000'; END IF;
  revertida:=replace(revertida,nuevos[i],antiguos[i]);
 END LOOP;
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM 'fe5e93bbece6225e72155029211e77b3d08518a3ac6ef8b350f994658cbcb3fe'
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '3984c45db3ceb2ebbeead7246336294a8a2ffbb2d99f66cf6f6876fc07ab6d1f'
 OR actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD193: PARO clave=delta_metadatos esperado=postimagenes_reversion_OID_ACL_dependencias_exactos actual=divergente' USING ERRCODE='55000'; END IF;
END $helper$;
-- CREATE OR REPLACE conserva las ACL comprobadas; no abre concesiones.
COMMIT;
