\set ON_ERROR_STOP on
-- AD255: asiento nominal de lectura RRHH Méritos en la cadena común.
-- El destinatario de datos es el propietario Méritos; el login lector RRHH
-- reutiliza el perfil técnico de Bolsa y no recibe EXECUTE de esta fachada.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000255',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_meritos_propietario') IS NULL
 OR to_regrole('vec_bolsa_llamamientos_lector_inscripciones_rrhh') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'AD255: preimagen o roles incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE d text; marca constant text:='''lectura_cartera_meritos''::text';
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM
  '0e6c3c43bdc1c7b958b465f728777571a490ad638032c6799ade4b46bacf58f5'
 OR (length(d)-length(replace(d,marca,'')))/length(marca)<>1
 OR strpos(d,'''lectura_meritos_rrhh''::text')<>0
 THEN RAISE EXCEPTION 'AD255: familia de auditoría divergente' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '
  ||replace(d,marca,marca||', ''lectura_meritos_rrhh''::text');
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_meritos_rrhh_formato_v1 CHECK (
 tipo_registro<>'lectura_meritos_rrhh' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$'
 AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$'
 AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$'
 AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND accion='meritos.hecho.revision_rrhh.consultar'
 AND modulo_id='meritos' AND recurso_ref ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND finalidad_ref='revision_meritos_inscripcion_rrhh'
 AND proceso='vec-server' AND canal='interna_corporativa'
 AND ((resultado='obtenida' AND motivo_ref='revision_meritos_rrhh_obtenida')
   OR (resultado='denegado' AND motivo_ref='revision_meritos_rrhh_denegada'))
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE c jsonb;v jsonb;s record;clave text;material bytea;material_sha text;
 anterior text;numero numeric;instante timestamptz(6);huella text;ref text;
 claves constant text[]:=ARRAY['intento_ref','registro_contexto_ref','contexto_sha256',
  'procedencia_sha256','autenticacion_ref','sesion_ref','autenticacion_sha256',
  'accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref',
  'proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','persona_ref'];
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR session_user<>'vec_bolsa_inscripciones_rrhh_lector'
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND m.roleid='vec_bolsa_llamamientos_lector_inscripciones_rrhh'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536
 OR p_vinculo IS NULL OR octet_length(p_vinculo) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(claves,1)
 OR (p_orden ?& claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD255: lector RRHH no acreditado' USING ERRCODE='42501'; END IF;
 FOREACH clave IN ARRAY claves LOOP
  IF jsonb_typeof(p_orden->clave) IS DISTINCT FROM 'string'
  OR octet_length(p_orden->>clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD255: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'accion' IS DISTINCT FROM 'meritos.hecho.revision_rrhh.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'recurso_ref' !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'revision_meritos_inscripcion_rrhh'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR NOT ((p_orden->>'resultado'='obtenida' AND p_orden->>'motivo_ref'='revision_meritos_rrhh_obtenida')
   OR (p_orden->>'resultado'='denegado' AND p_orden->>'motivo_ref'='revision_meritos_rrhh_denegada'))
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'AD255: contrato de lectura divergente' USING ERRCODE='22023'; END IF;
 BEGIN c:=convert_from(p_contexto,'UTF8')::jsonb;
       v:=convert_from(p_vinculo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD255: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(v) IS DISTINCT FROM 'object'
 OR c->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v->>'bloque_version' IS DISTINCT FROM '2'
 OR v->>'principal_id' IS DISTINCT FROM c->>'principal_ref'
 OR v->>'perfil_activo_ref' IS DISTINCT FROM c->>'perfil_activo_ref'
 OR v->>'cuenta_ref' IS DISTINCT FROM c->>'cuenta_ref'
 OR v->>'metodo_observado' IS DISTINCT FROM c->>'metodo'
 OR v->>'garantia_observada' IS DISTINCT FROM c->>'garantia'
 OR v->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR v->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
   p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
   p_orden->>'procedencia_sha256',p_contexto) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(p_vinculo) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD255: contexto o identidad divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND OR s.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR s.cuenta_ref IS DISTINCT FROM c->>'cuenta_ref'
 OR s.control_sesion_ref IS DISTINCT FROM v->>'control_sesion_ref'
 OR s.control_sesion_revision::text IS DISTINCT FROM v->>'control_sesion_revision'
 OR s.control_sesion_huella_sha256 IS DISTINCT FROM v->>'control_sesion_huella_sha256'
 OR s.superficie IS DISTINCT FROM 'interna_corporativa'
 OR s.cuenta_privilegiada IS NOT FALSE OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.sesion_valida_hasta IS DISTINCT FROM (v->>'sesion_valida_hasta')::timestamptz
 OR s.sesion_revalidada_en IS DISTINCT FROM (v->>'sesion_revalidada_en')::timestamptz
 THEN RAISE EXCEPTION 'AD255: sesión RRHH revocada' USING ERRCODE='42501'; END IF;
 material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_meritos_rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo),'hex'));
 FOREACH clave IN ARRAY claves LOOP
  material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>clave);
 END LOOP;
 material_sha:=encode(sha256(material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.intento_ref=p_orden->>'intento_ref')
 THEN RAISE EXCEPTION 'AD255: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT numero,anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 numero:=numero+1; instante:=clock_timestamp();
 ref:='aud_v3_li_'||substr(p_orden->>'intento_ref',9,32);
 huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_meritos_rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(numero::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(ref,numero,anterior,huella,instante,'lectura_meritos_rrhh',
  p_orden->>'intento_ref',material_sha,c->>'principal_ref',c->>'perfil_activo_ref',
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',p_orden->>'finalidad_ref',
  p_orden->>'resultado',p_orden->>'motivo_ref',p_orden->>'proceso',p_orden->>'canal',
  p_orden->>'correlacion_ref',encode(sha256(p_vinculo),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT ref,numero,huella,p_orden->>'correlacion_ref',instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(bytea,bytea,jsonb)
 TO vec_meritos_propietario;
COMMIT;
