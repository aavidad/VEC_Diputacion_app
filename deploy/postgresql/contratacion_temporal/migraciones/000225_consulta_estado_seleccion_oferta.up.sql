\set ON_ERROR_STOP on
-- CT225. Descubrimiento acotado del ámbito y consulta V3 de la única
-- selección de oferta activa, o del último descarte. No mueve leases ni
-- produce otra selección. Requiere CT220 y AD278 instalados.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000225',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.estado_seleccion_oferta_ct220(vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_class c
      WHERE c.oid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
      AND c.relowner='vec_contratacion_temporal_propietario'::regrole
      AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_index x JOIN pg_class idx ON idx.oid=x.indexrelid
      WHERE x.indrelid='vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
      AND idx.relname='ejecucion_seleccion_oferta_unica_expediente_ct220'
      AND x.indisunique AND x.indisvalid)
 THEN RAISE EXCEPTION 'CT225: preimagen CT220/AD278 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $pre_detalle$
DECLARE f1 oid;f2 oid;
BEGIN
 f1:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 f2:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF f1 IS NULL OR f2 IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(f1),'UTF8')),'hex') IS DISTINCT FROM
       '52970fd98d969acf31bff8f7b53d99a51b01026055833f701be043cce5d280e4'
    OR encode(sha256(convert_to(pg_get_functiondef(f2),'UTF8')),'hex') IS DISTINCT FROM
       '44c8d0e2c3597533300cdd066279a7c4a17526dd2c9eb1f158722a0c36249fcd'
    OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid IN (f1,f2)
       AND (p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
        OR NOT p.prosecdef OR p.provolatile<>'v' OR p.proacl::text IS DISTINCT FROM
         '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'))
    OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT225: fuente nominal del detalle divergente' USING ERRCODE='55000'; END IF;
END $pre_detalle$;

-- Preparación interna: limita la V3 posterior al ámbito y versión actuales.
-- El resultado nunca cruza HTTP sin la consulta nominal siguiente.
CREATE FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(
 p_organizacion text,p_expediente text)
RETURNS TABLE(version_actual bigint,categoria_ref text,centro_ref text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR p_organizacion IS NULL OR p_expediente IS NULL
    OR length(p_organizacion) NOT BETWEEN 2 AND 160
    OR length(p_expediente) NOT BETWEEN 2 AND 160
 THEN RAISE EXCEPTION 'CT225: contexto denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 SELECT a.version::bigint,v.agregado_json#>>'{solicitud,categoria_ref}',
        v.agregado_json#>>'{solicitud,centro_ref}'
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=p_expediente
   AND v.agregado_json->>'organizacion_ref'=p_organizacion
   AND a.version>=1 AND a.version<=9007199254740991
   AND length(v.agregado_json#>>'{solicitud,categoria_ref}') BETWEEN 2 AND 160
   AND length(v.agregado_json#>>'{solicitud,centro_ref}') BETWEEN 2 AND 160;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)
 TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb;d jsonb;h text;ctx text;c record;e vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6%ROWTYPE;
 consumo record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 128 AND 65536
 THEN RAISE EXCEPTION 'CT225: lectura no autorizable' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'CT225: material ilegible' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m)<>'object'
    OR m->>'esquema' IS DISTINCT FROM 'vec.ct.seleccion-oferta.consulta-expediente.v1'
    OR m-ARRAY['esquema','organizacion_ref','expediente_ref','version_observada',
      'categoria_ref','centro_ref','actor_ref','perfil_ref']<>'{}'::jsonb
    OR NOT (m ?& ARRAY['esquema','organizacion_ref','expediente_ref','version_observada',
      'categoria_ref','centro_ref','actor_ref','perfil_ref'])
    OR jsonb_typeof(m->'organizacion_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'expediente_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'categoria_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'centro_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'actor_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'perfil_ref') IS DISTINCT FROM 'string'
    OR jsonb_typeof(m->'version_observada')<>'number'
    OR m->>'version_observada' !~ '^[1-9][0-9]{0,15}$'
    OR length(m->>'organizacion_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'expediente_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'categoria_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'centro_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'actor_ref') NOT BETWEEN 2 AND 160
    OR length(m->>'perfil_ref') NOT BETWEEN 2 AND 160
 THEN RAISE EXCEPTION 'CT225: consulta inválida' USING ERRCODE='22023'; END IF;
 -- Se bloquea la cabecera antes de leer O6; reserva, confirmación y
 -- cancelación usan el mismo orden de bloqueo.
 SELECT a.version,v.agregado_json INTO c
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE a.expediente_ref=m->>'expediente_ref'
   AND v.agregado_json->>'organizacion_ref'=m->>'organizacion_ref'
 FOR SHARE OF a,v;
 IF c.version IS NULL OR c.version::text IS DISTINCT FROM m->>'version_observada'
    OR c.agregado_json#>>'{solicitud,categoria_ref}' IS DISTINCT FROM m->>'categoria_ref'
    OR c.agregado_json#>>'{solicitud,centro_ref}' IS DISTINCT FROM m->>'centro_ref'
 THEN RAISE EXCEPTION 'CT225: contexto CT cambió' USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(m->>'categoria_ref')::text||',"centro_ref":'||
   to_json(m->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(m->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.bolsa.oferta.vincular'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_oferta_bolsa_expediente'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ctx
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT225: decisión no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM ctx
    OR consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'CT225: consumo V3 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO e FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 x
 WHERE x.tipo_seleccion='oferta'
   AND x.solicitud_json->>'organizacion_ref'=m->>'organizacion_ref'
   AND x.solicitud_json->>'expediente_ref'=m->>'expediente_ref'
 ORDER BY (x.situacion<>'descartada') DESC,x.actualizada_en DESC,x.clave_idempotencia DESC
 LIMIT 1 FOR SHARE;
 IF e.clave_idempotencia IS NULL THEN
  RETURN jsonb_build_object('situacion','ausente');
 END IF;
 IF e.solicitud_json->>'actor_ref' IS DISTINCT FROM m->>'actor_ref'
    OR e.solicitud_json->>'perfil_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR e.solicitud_json->>'categoria_ref' IS DISTINCT FROM m->>'categoria_ref'
    OR e.solicitud_json->>'centro_ref' IS DISTINCT FROM m->>'centro_ref'
    OR vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(e.comando_oferta_json,e.solicitud_json) IS NOT TRUE
    OR (e.situacion='confirmada' AND
      vec_contratacion_temporal.terminal_seleccion_oferta_valido_ct220(e) IS NOT TRUE)
    OR (e.situacion='descartada' AND
      vec_contratacion_temporal.terminal_descarte_oferta_valido_ct220(e) IS NOT TRUE)
 THEN RAISE EXCEPTION 'CT225: ejecución incompatible' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('situacion',e.situacion,
  'intencion_original',e.solicitud_json,
  'estado',vec_contratacion_temporal.estado_seleccion_oferta_ct220(e));
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

-- El recibo original conserva sus bytes y su hash. La cifra se obtiene
-- en la misma transacción, de la versión exacta que devolvió ese recibo.
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 contenido_canonico bytea,
 esquema text,
 acceso_ref text,
 secuencia numeric,
 anterior_sha256 text,
 huella_sha256 text,
 vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,
 registrada_en timestamptz,
 auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,
 resultado_huella_sha256 text,
 cursor_huella_sha256 text,
 generada_en timestamptz,
 expediente_ref text,
 version_expediente numeric,
 total smallint,
 recibo_sello_sha256 text,
 numero_personas smallint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE v_recibo record;v_agregado jsonb;v_numero jsonb;v_cantidad smallint;
BEGIN
 SELECT * INTO STRICT v_recibo FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT v.agregado_json INTO STRICT v_agregado
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=v_recibo.expediente_ref
   AND v.version=v_recibo.version_expediente
   AND v.agregado_json->>'organizacion_ref'=p_alcance.organizacion_ref;
 v_numero:=v_agregado#>'{solicitud,numero_personas}';
 IF v_numero IS NULL OR (jsonb_typeof(v_numero)='number' AND v_numero::text='0') THEN
  -- La solicitud anterior al campo estructurado sigue el circuito singular.
  v_cantidad:=1;
 ELSIF jsonb_typeof(v_numero)='number' AND v_numero::text ~ '^([1-9][0-9]?|100)$'
       AND (v_numero::text)::integer BETWEEN 1 AND 100 THEN
  v_cantidad:=(v_numero::text)::smallint;
 ELSE
  RAISE EXCEPTION 'CT225: cantidad histórica inválida' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT
  v_recibo.contenido_canonico,
  v_recibo.esquema,
  v_recibo.acceso_ref,
  v_recibo.secuencia,
  v_recibo.anterior_sha256,
  v_recibo.huella_sha256,
  v_recibo.vinculo_identidad_huella_sha256,
  v_recibo.alcance_huella_sha256,
  v_recibo.registrada_en,
  v_recibo.auditoria_vec_ref,
  v_recibo.auditoria_vec_huella_sha256,
  v_recibo.consumo_vec_huella_sha256,
  v_recibo.contenido_huella_sha256,
  v_recibo.resultado_huella_sha256,
  v_recibo.cursor_huella_sha256,
  v_recibo.generada_en,
  v_recibo.expediente_ref,
  v_recibo.version_expediente,
  v_recibo.total,
  v_recibo.recibo_sello_sha256,
  v_cantidad;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_consultor_rrhh;

-- El recibo original conserva sus bytes y su hash. La cifra se obtiene
-- en la misma transacción, de la versión exacta que devolvió ese recibo.
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,
 p_cursor text)
RETURNS TABLE(
 contenido_canonico bytea,
 esquema text,
 acceso_ref text,
 secuencia numeric,
 anterior_sha256 text,
 huella_sha256 text,
 vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,
 registrada_en timestamptz,
 auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text,
 contenido_huella_sha256 text,
 resultado_huella_sha256 text,
 cursor_huella_sha256 text,
 generada_en timestamptz,
 expediente_ref text,
 version_expediente numeric,
 total smallint,
 recibo_sello_sha256 text,
 resultado_bolsa jsonb,
 numero_personas smallint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $f$
DECLARE v_recibo record;v_agregado jsonb;v_numero jsonb;v_cantidad smallint;
BEGIN
 SELECT * INTO STRICT v_recibo FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,p_cursor);
 SELECT v.agregado_json INTO STRICT v_agregado
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=v_recibo.expediente_ref
   AND v.version=v_recibo.version_expediente
   AND v.agregado_json->>'organizacion_ref'=p_alcance.organizacion_ref;
 v_numero:=v_agregado#>'{solicitud,numero_personas}';
 IF v_numero IS NULL OR (jsonb_typeof(v_numero)='number' AND v_numero::text='0') THEN
  -- La solicitud anterior al campo estructurado sigue el circuito singular.
  v_cantidad:=1;
 ELSIF jsonb_typeof(v_numero)='number' AND v_numero::text ~ '^([1-9][0-9]?|100)$'
       AND (v_numero::text)::integer BETWEEN 1 AND 100 THEN
  v_cantidad:=(v_numero::text)::smallint;
 ELSE
  RAISE EXCEPTION 'CT225: cantidad histórica inválida' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT
  v_recibo.contenido_canonico,
  v_recibo.esquema,
  v_recibo.acceso_ref,
  v_recibo.secuencia,
  v_recibo.anterior_sha256,
  v_recibo.huella_sha256,
  v_recibo.vinculo_identidad_huella_sha256,
  v_recibo.alcance_huella_sha256,
  v_recibo.registrada_en,
  v_recibo.auditoria_vec_ref,
  v_recibo.auditoria_vec_huella_sha256,
  v_recibo.consumo_vec_huella_sha256,
  v_recibo.contenido_huella_sha256,
  v_recibo.resultado_huella_sha256,
  v_recibo.cursor_huella_sha256,
  v_recibo.generada_en,
  v_recibo.expediente_ref,
  v_recibo.version_expediente,
  v_recibo.total,
  v_recibo.recibo_sello_sha256,
  v_recibo.resultado_bolsa,
  v_cantidad;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text) TO vec_contratacion_temporal_consultor_rrhh;

DO $post$
BEGIN
 IF to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)') IS NULL
    OR NOT has_function_privilege('vec_contratacion_temporal_consultor_rrhh',
      'vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_consultor_rrhh',
      'vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'::regprocedure,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p,
      LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
      WHERE p.oid IN (
        'vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)'::regprocedure,
        'vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'::regprocedure)
        AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
 THEN RAISE EXCEPTION 'CT225: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
