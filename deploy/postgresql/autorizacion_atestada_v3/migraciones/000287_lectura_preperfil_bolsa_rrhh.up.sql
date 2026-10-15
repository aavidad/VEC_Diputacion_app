\set ON_ERROR_STOP on
-- AD287: asiento común preperfil para selección propia RRHH de Bolsa.
-- Conserva literalmente el CHECK instalado y suma una familia cerrada.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000287',0));
DO $check$
DECLARE anterior text; expresion text; nueva text; c pg_constraint%ROWTYPE;
 rama constant text:=$rama$
 (tipo_registro='preperfil_bolsa_rrhh'
  AND auditoria_ref IS NOT NULL AND secuencia IS NOT NULL
  AND anterior_sha256 IS NOT NULL AND huella_sha256 IS NOT NULL
  AND registrada_en IS NOT NULL
  AND actor_ref IS NOT NULL AND accion IS NOT NULL AND modulo_id IS NOT NULL
  AND recurso_ref IS NOT NULL AND finalidad_ref IS NOT NULL
  AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
  AND proceso IS NOT NULL AND canal IS NOT NULL AND correlacion_ref IS NOT NULL
  AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL
  AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
  AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
  AND intento_ref IS NULL AND intento_material_sha256 IS NULL
  AND perfil_activo_ref IS NULL AND registro_contexto_ref IS NULL
  AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
  AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL
  AND vinculo_sha256 IS NULL AND operador_login IS NULL AND plan_sha256 IS NULL
  AND aprobacion_ref IS NULL AND version_consumo IS NULL
  AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL
  AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL
  AND fuentes_solicitud_sha256 IS NULL AND unidad_plan_ref IS NULL
  AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL
  AND unidad_alcance IS NULL AND unidad_recibo_ref IS NULL
  AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL
  AND bootstrap_solicitud_sha256 IS NULL AND mantenimiento_detalle IS NULL
  AND mantenimiento_solicitud_sha256 IS NULL AND periodica_detalle IS NULL
  AND preservacion_detalle IS NULL AND gobierno_usuarios_detalle IS NULL
  AND gobierno_usuarios_solicitud_sha256 IS NULL AND transaccion_origen IS NULL
  AND perfiles_asignables_detalle IS NULL AND perfiles_asignables_solicitud_sha256 IS NULL
  AND identidad_operacion_ref IS NULL AND identidad_plan_ref IS NULL
  AND identidad_preimagen_sha256 IS NULL AND identidad_configuracion_sha256 IS NULL
  AND identidad_alcance IS NULL AND identidad_solicitud_sha256 IS NULL
  AND catalogo_acciones_detalle IS NULL AND catalogo_acciones_solicitud_sha256 IS NULL
  AND gobierno_rol_nuevo_solicitud_sha256 IS NULL
  AND version_rol_bolsa_solicitud_sha256 IS NULL
  AND ratificacion_catalogo_detalle IS NULL
  AND lectura_revision_permisos IS NULL AND lectura_instantanea_sha256 IS NULL
  AND publicacion_vinculo_f1_detalle IS NULL AND intento_vinculo_f1_sha256 IS NULL
  AND accion='bolsa.justificante_renuncia.perfil_propio.consultar'
  AND modulo_id='bolsa' AND finalidad_ref='seleccion_perfil_rrhh'
  AND canal='interna_corporativa' AND resultado IN ('permitido','denegado'))$rama$;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR
  NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL OR
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD287: entorno incompatible' USING ERRCODE='55000'; END IF;
 LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
 SELECT * INTO STRICT c FROM pg_constraint WHERE
  conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND
  conname='auditoria_tipo_disjunto_v4' AND contype='c' AND convalidated;
 anterior:=pg_get_constraintdef(c.oid,false);
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
   '939babe4fcb650d87a774a559db7fdd74f0eb64f133a4b6b227821b5b34c468e' OR
  left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')' OR
  (SELECT relowner FROM pg_class WHERE oid=c.conrelid) IS DISTINCT FROM
   'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD287: CHECK preimagen divergente' USING ERRCODE='55000'; END IF;
 expresion:=substr(anterior,8,length(anterior)-8);
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
  DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 CHECK (('
  ||expresion||') OR '||rama||')';
 SELECT pg_get_constraintdef(oid,false) INTO STRICT nueva FROM pg_constraint WHERE
  conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND
  conname='auditoria_tipo_disjunto_v4' AND contype='c' AND convalidated;
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM
   '182b3fed22360afa10c6a496ca11e36c863ce6c731141d9314735b78504b9326' OR
  (SELECT relowner FROM pg_class WHERE oid=c.conrelid) IS DISTINCT FROM
   'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD287: CHECK postimagen divergente' USING ERRCODE='55000'; END IF;
END $check$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE claves constant text[]:=ARRAY['evento_ref','actor_ref','recurso_ref','resultado',
 'motivo_ref','correlacion_ref','fuente_sha256'];
 k text; material bytea; material_sha text; previa record;
 cabeza record; n numeric; instante timestamptz(6); ref text; huella text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR
  jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_evento))<>array_length(claves,1) OR
  p_evento ?& claves IS NOT TRUE OR
  NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND
    NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND
    m.roleid='vec_autorizacion_fuente'::regrole AND m.inherit_option AND
    NOT m.set_option AND NOT m.admin_option AND
    (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1)
 THEN RAISE EXCEPTION 'AD287: evento denegado' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY claves LOOP
  IF jsonb_typeof(p_evento->k) IS DISTINCT FROM 'string' OR
   octet_length(p_evento->>k) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD287: evento inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$' OR
  p_evento->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR
  p_evento->>'recurso_ref' !~ '^cuenta:[0-9a-f]{64}$' OR
  p_evento->>'resultado' NOT IN ('permitido','denegado') OR
  p_evento->>'motivo_ref' NOT IN ('perfil_unico','perfil_ausente_o_ambiguo') OR
  p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$' OR
  p_evento->>'fuente_sha256' !~ '^[0-9a-f]{64}$' OR
  ((p_evento->>'resultado'='permitido') IS DISTINCT FROM
   (p_evento->>'motivo_ref'='perfil_unico'))
 THEN RAISE EXCEPTION 'AD287: evento divergente' USING ERRCODE='42501'; END IF;
 material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.preperfil-bolsa-rrhh.v1');
 FOREACH k IN ARRAY claves LOOP
  material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>k);
 END LOOP;
 material_sha:=encode(sha256(material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:evento-bolsa:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO previa
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF previa.tipo_registro IS DISTINCT FROM 'preperfil_bolsa_rrhh' OR
   previa.evento_material_sha256 IS DISTINCT FROM material_sha
  THEN RAISE EXCEPTION 'AD287: evento repetido divergente' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT previa.auditoria_ref,previa.secuencia,previa.huella_sha256,
   previa.correlacion_ref,previa.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa AS secuencia,r.anterior_sha256 AS cabeza_sha256 INTO STRICT cabeza
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF cabeza.secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD287: secuencia agotada' USING ERRCODE='22003'; END IF;
 n:=cabeza.secuencia+1; instante:=clock_timestamp();
 ref:='aud_v3_b_'||substr(p_evento->>'evento_ref',8,32);
 huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.preperfil-bolsa-rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(n::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(cabeza.cabeza_sha256)||
  vec_autorizacion_atestada_v3.encuadrar_mac(ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,fuente_ref,fuente_sha256,actor_ref,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(ref,n,cabeza.cabeza_sha256,huella,instante,'preperfil_bolsa_rrhh',
  p_evento->>'evento_ref',material_sha,'identidad_sesiones_v1',p_evento->>'fuente_sha256',
  p_evento->>'actor_ref','bolsa.justificante_renuncia.perfil_propio.consultar','bolsa',
  p_evento->>'recurso_ref','seleccion_perfil_rrhh',p_evento->>'resultado',
  p_evento->>'motivo_ref','vec-server','interna_corporativa',p_evento->>'correlacion_ref');
 RETURN QUERY SELECT ref,n,huella,p_evento->>'correlacion_ref',instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)
 TO vec_autorizacion_propietario;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole OR
  p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v' OR
  p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
   'TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']::text[] OR
  p.proacl::text IS DISTINCT FROM
   '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_autorizacion_propietario=X/vec_autorizacion_atestada_v3_propietario}' OR
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
   '509973a72d9a8e2fa02e5ab9ec641ba39a5e25ad6fe657c1e8ae9ed40791bd37'
 THEN RAISE EXCEPTION 'AD287: función postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
