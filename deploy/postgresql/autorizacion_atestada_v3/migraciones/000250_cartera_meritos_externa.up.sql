\set ON_ERROR_STOP on
-- AD250. Consumo V3 exclusivo de retirada y auditoría nominal de cartera.
-- Parte de POST-AD249 #1055@cd1e540a3, SHA archivo
-- 07b94d02f7d615211aa7cb02a394c518f8ba44748c9f17cf95ad0aebbc4898d5.
-- No instala permisos personales ni motivos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='90s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;k pg_constraint%ROWTYPE;h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_meritos_cartera_externo') IS NULL
 OR to_regrole('vec_meritos_propietario') IS NULL
 OR to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1()') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(bytea,bytea,jsonb,text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD250: dependencias o firmas incompatibles' USING ERRCODE='55000';END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.oid<>16692 OR p.proowner<>to_regrole('vec_autorizacion_atestada_v3_propietario')
 OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR p.proacl IS DISTINCT FROM ARRAY[
  'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario'::aclitem]
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  '21293bfb02cda7760d3395ebbe2066528577e5eb20ff859e5ea2d652d1c64d7b'
 THEN RAISE EXCEPTION 'AD250: núcleo POST-AD249 divergente' USING ERRCODE='55000';END IF;
 SELECT * INTO k FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR k.oid<>161379
 OR encode(sha256(convert_to(pg_get_constraintdef(k.oid,false),'UTF8')),'hex')
  IS DISTINCT FROM '1a299eefbc2936a06b414d47defbb17b213fec7a29f2fa6eec448a008727d8ba'
 THEN RAISE EXCEPTION 'AD250: CHECK auditor POST-AD249 divergente' USING ERRCODE='55000';END IF;
 SELECT * INTO k FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check'
 AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR k.oid<>161185
 OR encode(sha256(convert_to(pg_get_constraintdef(k.oid,false),'UTF8')),'hex')
  IS DISTINCT FROM '17e41379b0d758caba982dc4fdd9cb93c5c60f4d2df0883b557157afac74a63d'
 THEN RAISE EXCEPTION 'AD250: CHECK audiencias POST-AD249 divergente' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nueva text;
 marca text:='''lectura_origen_datos_contacto''::text';
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
  '1a299eefbc2936a06b414d47defbb17b213fec7a29f2fa6eec448a008727d8ba'
 OR (length(anterior)-length(replace(anterior,marca,'')))/length(marca)<>1
 OR strpos(anterior,'''lectura_cartera_meritos''::text')<>0
 THEN RAISE EXCEPTION 'AD250: familia lectora divergente' USING ERRCODE='55000';END IF;
 nueva:=replace(anterior,marca,marca||', ''lectura_cartera_meritos''::text');
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
  DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
  ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_cartera_meritos_formato_v1 CHECK (
 tipo_registro<>'lectura_cartera_meritos' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$'
 AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$'
 AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$'
 AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='meritos' AND proceso='vec-server' AND canal='externa_personal'
 AND accion='meritos.cartera.consultar_propia'
 AND recurso_ref ~ '^cartera:per_[A-Za-z0-9_-]{22,128}$'
 AND finalidad_ref='gestion_cartera_propia' AND resultado='obtenida'
 AND motivo_ref='lectura_cartera_meritos_correcta'
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));

DO $nucleo$
DECLARE f oid:=to_regprocedure(
 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;nuevo text;meta jsonb;
 runtime text:=$r$
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa_externa'
            AND vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS TRUE)
$r$;
 runtime_nuevo text:=$r$
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa_externa'
            AND vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
             JOIN pg_roles g ON g.oid=m.roleid
             WHERE l.rolname=session_user AND l.rolname='vec_externo_meritos_cartera'
              AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
              AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
               OR g.rolreplication OR g.rolbypassrls)
              AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
            AND NOT EXISTS(SELECT 1 FROM pg_auth_members
             WHERE member='vec_meritos_cartera_externo'::regrole))
$r$;
 excl text:=$e$p_perfil_mutacion IS DISTINCT FROM 'version_rol_bolsa_externa'$e$;
 excl_nuevo text:=$e$p_perfil_mutacion IS DISTINCT FROM 'version_rol_bolsa_externa'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_cartera_retirada'$e$;
 marca text:=$m$       ) IS NOT TRUE
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$m$;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.cartera.retirar_propio.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.cartera.retirar_propio'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_cartera_propia'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'false'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["disponibilidad","recibo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
$x$;
 post_marca text:=$p$    v_huella_capacidad := pg_catalog.encode($p$;
 post_guard text:=$g$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
       AND vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
        d->'vinculo_autenticacion_actor',
        (c->>'emitida_en')::timestamptz,(c->>'expira_en')::timestamptz,
        clock_timestamp()) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo cartera Méritos denegado';
    END IF;
$g$;
 registro_marca text:=$q$    SELECT * INTO v_registro
      FROM vec_autorizacion.
           registrar_y_revalidar_decision_contexto_actor_v3(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
$q$;
 registro_nuevo text:=$q$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada' THEN
        SELECT * INTO v_registro
          FROM vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
            p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);
    ELSE
        SELECT * INTO v_registro
          FROM vec_autorizacion.
               registrar_y_revalidar_decision_contexto_actor_v3(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;
$q$;
 reval_marca text:=$v$    v_revalidada_en :=
      vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
$v$;
 reval_nuevo text:=$v$    IF p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada' THEN
        v_revalidada_en:=vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(
          p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);
    ELSE
        v_revalidada_en :=
          vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;
$v$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta
 FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)
  IS DISTINCT FROM '21293bfb02cda7760d3395ebbe2066528577e5eb20ff859e5ea2d652d1c64d7b'
 OR (length(original)-length(replace(original,runtime,'')))/length(runtime)<>1
 OR (length(original)-length(replace(original,excl,'')))/length(excl)<>1
 OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 OR (length(original)-length(replace(original,post_marca,'')))/length(post_marca)<>1
 OR (length(original)-length(replace(original,registro_marca,'')))/length(registro_marca)<>1
 OR (length(original)-length(replace(original,reval_marca,'')))/length(reval_marca)<>1
 OR strpos(original,'meritos_cartera_retirada')<>0
 THEN RAISE EXCEPTION 'AD250: núcleo sin marcas POST-AD249 únicas' USING ERRCODE='55000';END IF;
 nuevo:=replace(original,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,excl,excl_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 nuevo:=replace(nuevo,post_marca,post_guard||post_marca);
 nuevo:=replace(nuevo,registro_marca,registro_nuevo);
 nuevo:=replace(nuevo,reval_marca,reval_nuevo);
 EXECUTE nuevo;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR replace(replace(replace(replace(replace(replace(pg_get_functiondef(f),
   reval_nuevo,reval_marca),registro_nuevo,registro_marca),
   post_guard||post_marca,post_marca),extension||marca,marca),
   excl_nuevo,excl),runtime_nuevo,runtime) IS DISTINCT FROM original
 THEN RAISE EXCEPTION 'AD250: núcleo alterado fuera del contrato' USING ERRCODE='55000';END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE d text;nuevo text;oid_check oid;
BEGIN
 SELECT c.oid,pg_get_constraintdef(c.oid,false) INTO STRICT oid_check,d
 FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check'
 AND c.contype='c' AND c.convalidated;
 IF oid_check<>161185
 OR encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM
   '17e41379b0d758caba982dc4fdd9cb93c5c60f4d2df0883b557157afac74a63d'
 OR left(d,7)<>'CHECK (' OR right(d,1)<>')'
 OR (length(d)-length(replace(d,
    '''vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1''::text','')))
   /length('''vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1''::text')<>1
 OR strpos(d,'''vec_meritos.cartera.retirar_propio.v1''::text')<>0
 THEN RAISE EXCEPTION 'AD250: audiencia POST-AD249 divergente' USING ERRCODE='55000';END IF;
 -- La audiencia personal tiene su propio disyunto, fuera de las audiencias ADMIN.
 nuevo:=left(d,length(d)-1)||
  ' OR (audiencia_consumo = ''vec_meritos.cartera.retirar_propio.v1''::text))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
  SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  JOIN pg_roles g ON g.oid=m.roleid
  WHERE l.rolname=session_user AND l.rolname='vec_externo_meritos_cartera'
   AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
   AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb
    OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,
 consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;m jsonb;x record;
BEGIN
 IF vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1() IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD250: ejecutor de retirada denegado' USING ERRCODE='42501';END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD250: material V3 inválido' USING ERRCODE='22023';END;
 IF c->>'operacion' IS DISTINCT FROM 'meritos.cartera.retirar_propio'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_meritos.cartera.retirar_propio.v1'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_cartera_propia'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
 OR d->'campos_permitidos' IS DISTINCT FROM '["disponibilidad","recibo"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR m#>>'{referencia,catalogo_id}' IS DISTINCT FROM 'motivos_autorizacion_meritos_cartera'
 THEN RAISE EXCEPTION 'AD250: retirada V3 divergente' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'meritos_cartera_retirada',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE
 OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM d->>'recurso_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 THEN RAISE EXCEPTION 'AD250: consumo V3 divergente' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_meritos_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_acreditada timestamptz;
 v_claves constant text[]:=ARRAY[
  'esquema','persona_ref','limite','filtro','cursor_hecho_ref','corte_en','idioma',
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'tipo_recurso','recurso_ref','finalidad_ref','campos_permitidos','resultado',
  'motivo_ref','proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','perfil_activo_ref','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref',
  'revision_control_version_rol','control_version_rol_huella_sha256',
  'revision_catalogo_politicas','catalogo_politicas_huella_sha256',
  'politicas_evaluadas','emitida_en','valida_hasta'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1() IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD250: lectura sin emisor nominal' USING ERRCODE='42501';END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD250: lista de lectura inválida' USING ERRCODE='22023';END IF;
  ELSIF v_clave IN ('limite','asignacion_version','version_rol_version',
    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR coalesce(p_orden->>v_clave ~ '^[1-9][0-9]{0,19}$',false) IS NOT TRUE
   THEN RAISE EXCEPTION 'AD250: versión de lectura inválida' USING ERRCODE='22023';END IF;
  ELSIF v_clave='cursor_hecho_ref' THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave)>256
   THEN RAISE EXCEPTION 'AD250: cursor inválido' USING ERRCODE='22023';END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD250: orden de lectura inválida' USING ERRCODE='22023';END IF;
 END LOOP;
 IF p_orden->>'esquema' IS DISTINCT FROM 'vec.meritos.cartera.propia.v1'
 OR p_orden->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR (p_orden->>'limite')::integer NOT BETWEEN 1 AND 50
 OR p_orden->>'filtro' NOT IN ('todos','disponibles','retirados')
 OR p_orden->>'idioma' NOT IN ('es','en')
 OR p_orden->>'accion' IS DISTINCT FROM 'meritos.cartera.consultar_propia'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'cartera_personal'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'cartera:'||(p_orden->>'persona_ref')
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_cartera_meritos_correcta'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
  '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb
 THEN RAISE EXCEPTION 'AD250: contrato positivo inválido' USING ERRCODE='22023';END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD250: evidencia de lectura inválida' USING ERRCODE='22023';END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_contexto->>'principal_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_contexto->>'persona_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 THEN RAISE EXCEPTION 'AD250: contexto externo divergente' USING ERRCODE='42501';END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz);
 IF v_acreditada IS NULL
 OR vec_autorizacion.cotejar_cartera_meritos_actual_v1(p_orden,v_vinculo) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD250: autoridad de lectura revocada' USING ERRCODE='42501';END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_cartera_meritos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD250: lectura repetida' USING ERRCODE='23505';END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_mc_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_cartera_meritos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_cartera_meritos',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
 bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
 bytea,bytea,jsonb) TO vec_meritos_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb,p_accion text)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_contexto jsonb;v_vinculo jsonb;v_acreditada timestamptz;
 v_claves constant text[]:=ARRAY['esquema','persona_ref','intento_ref',
  'registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'tipo_recurso','recurso_ref','finalidad_ref','resultado','motivo_ref',
  'proceso','canal','correlacion_ref'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_emitida timestamptz(6);v_valida timestamptz(6);
 v_huella text;v_ref text;v_existente record;motivo_prefijo text;
BEGIN
 IF vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1() IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_accion NOT IN ('meritos.cartera.consultar_propia','meritos.cartera.retirar_propio')
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD250: intento sin emisor nominal' USING ERRCODE='42501';END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD250: orden de intento inválida' USING ERRCODE='22023';END IF;
 END LOOP;
 motivo_prefijo:=CASE p_accion
  WHEN 'meritos.cartera.consultar_propia' THEN 'lectura_cartera_meritos_'
  ELSE 'intento_retirada_cartera_meritos_' END;
 IF p_orden->>'esquema' IS DISTINCT FROM 'vec.meritos.cartera.propia.v1'
 OR p_orden->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'intento_ref' !~ '^intento_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  'cartera:'||encode(sha256(convert_to(p_orden->>'persona_ref','UTF8')),'hex')
 OR p_orden->>'accion' IS DISTINCT FROM p_accion
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'cartera_personal'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'resultado' NOT IN ('denegado','error')
 OR p_orden->>'motivo_ref' IS DISTINCT FROM motivo_prefijo||(p_orden->>'resultado')
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD250: contrato de intento inválido' USING ERRCODE='22023';END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD250: evidencia de intento inválida' USING ERRCODE='22023';END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_contexto->>'principal_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_contexto->>'persona_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 THEN RAISE EXCEPTION 'AD250: vínculo de intento divergente' USING ERRCODE='42501';END IF;
 v_emitida:=clock_timestamp();
 v_valida:=least(v_emitida+interval '1 second',
  (v_vinculo->>'sesion_valida_hasta')::timestamptz,
  (v_contexto->>'vigente_hasta')::timestamptz);
 IF v_valida<=v_emitida
 OR vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
  v_vinculo,v_emitida,v_valida,clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD250: sesión de intento revocada' USING ERRCODE='42501';END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',v_emitida,v_valida);
 IF v_acreditada IS NULL
 THEN RAISE EXCEPTION 'AD250: contexto de intento no acreditado' USING ERRCODE='42501';END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:intento:'||(p_orden->>'intento_ref'),0));
 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,
  a.registrada_en,a.intento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN
  IF v_existente.intento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD250: intento incompatible' USING ERRCODE='23505';END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,
   v_existente.huella_sha256,v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_i_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_nominal',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id());
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(
 bytea,bytea,jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(
  p_contexto,p_vinculo,p_orden,'meritos.cartera.consultar_propia')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(
 bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(
 bytea,bytea,jsonb) TO vec_meritos_propietario,vec_meritos_cartera_externo;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.registrar_intento_cartera_meritos_privado_v1(
  p_contexto,p_vinculo,p_orden,'meritos.cartera.retirar_propio')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(
 bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(
 bytea,bytea,jsonb) TO vec_meritos_propietario,vec_meritos_cartera_externo;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
 TO vec_meritos_propietario,vec_meritos_cartera_externo;
COMMIT;
