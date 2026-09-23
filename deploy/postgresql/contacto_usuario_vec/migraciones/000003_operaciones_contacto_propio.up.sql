\set ON_ERROR_STOP on
-- Contacto3: intención propia, confirmación exacta y recuperación sin estado web.
-- AD3 operaciones debe preceder a T13/8 y a esta migración. No contiene
-- correo claro ni crea fuente de identidad, permiso o firma alternativa.
BEGIN;
SET LOCAL ROLE vec_contacto_usuario_owner;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:3',0));
DO $pre$
DECLARE r text;
BEGIN
 IF current_user<>'vec_contacto_usuario_owner' OR getdatabaseencoding()<>'UTF8'
    OR to_regprocedure('vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text)') IS NULL
    OR to_regclass('vec_contacto_usuario_v1.operaciones') IS NOT NULL
    OR NOT has_function_privilege('vec_contacto_usuario_writer',
        'vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE') THEN
    RAISE EXCEPTION 'Contacto3: preimagen o dependencias incompatibles' USING ERRCODE='55000';
 END IF;
 FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer',
      'vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
        AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
       OR (r<>'vec_contacto_usuario_owner' AND
           has_table_privilege(r,'vec_contacto_usuario_v1.versiones','SELECT,INSERT,UPDATE,DELETE')) THEN
       RAISE EXCEPTION 'Contacto3: roles o ACL de contacto incompatibles' USING ERRCODE='55000';
    END IF;
 END LOOP;
END $pre$;

CREATE TABLE vec_contacto_usuario_v1.operaciones (
    operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    sujeto_ref text NOT NULL CHECK(sujeto_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
    actor_id_hmac text NOT NULL CHECK(actor_id_hmac ~ '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'),
    version_esperada numeric(20,0) NOT NULL CHECK(version_esperada BETWEEN 0 AND 9007199254740990),
    hmac_clave_ref text NOT NULL CHECK(hmac_clave_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    hmac_valor text NOT NULL CHECK(hmac_valor ~ '^[0-9a-f]{64}$' AND hmac_valor<>repeat('0',64)),
    estado text NOT NULL CHECK(estado IN ('preparada','confirmada','cancelada')),
    version_resultante numeric(20,0),
    recibo_ref text,
    consumo_ref text,
    auditoria_preparacion_ref text NOT NULL CHECK(auditoria_preparacion_ref ~ '^acc_[0-9a-f]{40}$'),
    auditoria_cierre_ref text CHECK(auditoria_cierre_ref IS NULL OR auditoria_cierre_ref ~ '^acc_[0-9a-f]{40}$'),
    preparada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    cerrada_en timestamptz(6),
    CONSTRAINT operacion_contacto_instantes CHECK(isfinite(preparada_en)
        AND (cerrada_en IS NULL OR (isfinite(cerrada_en) AND cerrada_en>=preparada_en))),
    CONSTRAINT operacion_contacto_resultado_coherente CHECK(
        (estado='preparada' AND version_resultante IS NULL AND recibo_ref IS NULL AND consumo_ref IS NULL
            AND auditoria_cierre_ref IS NULL AND cerrada_en IS NULL)
        OR (estado='cancelada' AND version_resultante IS NULL AND recibo_ref IS NULL AND consumo_ref IS NULL
            AND auditoria_cierre_ref IS NOT NULL AND cerrada_en IS NOT NULL)
        OR (estado='confirmada' AND version_resultante IS NOT NULL AND version_resultante=version_esperada+1
            AND recibo_ref IS NOT NULL AND recibo_ref ~ '^acc_[0-9a-f]{40}$'
            AND consumo_ref IS NOT NULL AND auditoria_cierre_ref IS NOT NULL AND auditoria_cierre_ref=recibo_ref
            AND cerrada_en IS NOT NULL)),
    CONSTRAINT operacion_contacto_version_fk FOREIGN KEY(sujeto_ref,version_resultante)
        REFERENCES vec_contacto_usuario_v1.versiones(sujeto_ref,version),
    CONSTRAINT operacion_contacto_consumo_fk FOREIGN KEY(consumo_ref)
        REFERENCES vec_contacto_usuario_v1.versiones(consumo_ref),
    CONSTRAINT operacion_contacto_sujeto_unico UNIQUE(operacion_ref,sujeto_ref)
);
CREATE UNIQUE INDEX operaciones_pendiente_unica
    ON vec_contacto_usuario_v1.operaciones(sujeto_ref,version_esperada) WHERE estado='preparada';
CREATE UNIQUE INDEX operaciones_contacto_version_confirmada
    ON vec_contacto_usuario_v1.operaciones(sujeto_ref,version_resultante) WHERE estado='confirmada';
CREATE INDEX operaciones_contacto_indice_propio
    ON vec_contacto_usuario_v1.operaciones(sujeto_ref,preparada_en DESC,operacion_ref DESC);

CREATE TABLE vec_contacto_usuario_v1.operacion_eventos (
    operacion_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    secuencia smallint NOT NULL CHECK(secuencia BETWEEN 1 AND 2),
    estado text NOT NULL CHECK(estado IN ('preparada','confirmada','cancelada')),
    auditoria_ref text NOT NULL CHECK(auditoria_ref ~ '^acc_[0-9a-f]{40}$'),
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(operacion_ref,secuencia),
    UNIQUE(auditoria_ref),
    CONSTRAINT operacion_evento_sujeto_fk FOREIGN KEY(operacion_ref,sujeto_ref)
        REFERENCES vec_contacto_usuario_v1.operaciones(operacion_ref,sujeto_ref),
    CONSTRAINT operacion_evento_inicial CHECK(secuencia<>1 OR estado='preparada'),
    CONSTRAINT operacion_evento_final CHECK(secuencia<>2 OR estado IN ('confirmada','cancelada'))
);

ALTER TABLE vec_contacto_usuario_v1.operaciones ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.operaciones FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.operacion_eventos ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contacto_usuario_v1.operacion_eventos FORCE ROW LEVEL SECURITY;
CREATE POLICY operaciones_lectura ON vec_contacto_usuario_v1.operaciones FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true));
CREATE POLICY operaciones_alta ON vec_contacto_usuario_v1.operaciones FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true)='vec.contacto_usuario.operacion.preparar');
CREATE POLICY operaciones_cambio ON vec_contacto_usuario_v1.operaciones FOR UPDATE TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.operacion.cancelar')) WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true) IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.operacion.cancelar'));
CREATE POLICY operaciones_bloqueo_preparacion ON vec_contacto_usuario_v1.operaciones FOR UPDATE TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true)='vec.contacto_usuario.operacion.preparar') WITH CHECK(false);
CREATE POLICY operacion_eventos_alta ON vec_contacto_usuario_v1.operacion_eventos FOR INSERT TO vec_contacto_usuario_owner WITH CHECK(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true));
CREATE POLICY operacion_eventos_lectura ON vec_contacto_usuario_v1.operacion_eventos FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true));
-- Preparar compara la versión real bajo autorización V3 propia. La política
-- anterior de contacto ocultaría una versión distinta y haría parecer vacío.
CREATE POLICY lectura_actual_preparacion ON vec_contacto_usuario_v1.actual FOR SELECT TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true)='vec.contacto_usuario.operacion.preparar');
CREATE POLICY bloqueo_actual_preparacion ON vec_contacto_usuario_v1.actual FOR UPDATE TO vec_contacto_usuario_owner USING(
    sujeto_ref=current_setting('vec.contacto.operacion_sujeto',true)
    AND current_setting('vec.contacto.operacion_accion',true)='vec.contacto_usuario.operacion.preparar') WITH CHECK(false);
REVOKE ALL ON TABLE vec_contacto_usuario_v1.operaciones,vec_contacto_usuario_v1.operacion_eventos FROM PUBLIC;

CREATE FUNCTION vec_contacto_usuario_v1.preparar_operacion_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(operacion_ref text,estado text,version_esperada numeric,replay_confirmado boolean,
    version_resultante numeric,recibo_ref text,conflicto_material boolean,conflicto_version boolean,auditoria_central bytea,
    consumo_ref text,consumo_huella_sha256 text,hmac_clave_ref text,hmac_valor text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; s text; op text; esperada numeric; actual numeric; huellas jsonb; anterior record;
    consumo record; revalidada record; auditada jsonb; conflicto text:=''; ref_resultado text; repetida boolean:=false;
    estado_resultado text:='preparada'; version_resultado numeric; recibo_resultado text;
    hmac_clave_resultado text:=''; hmac_valor_resultado text:='';
BEGIN
 IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.operacion.preparar'
    OR current_user<>'vec_contacto_usuario_owner' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
    RAISE EXCEPTION 'Contacto3: preparar denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF consumo.consumo_nuevo IS DISTINCT FROM true THEN
    RAISE EXCEPTION 'Contacto3: preparación exige V3 fresco' USING ERRCODE='P1102';
 END IF;
 b:=vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 s:=b->>'SujetoRef'; op:=b->>'OperacionRef'; esperada:=(b->>'VersionEsperada')::numeric;
 huellas:=b->'HuellasReplay';
 IF s IS NULL OR s !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR op IS NULL OR op !~ '^opr_[A-Za-z0-9_-]{22,128}$'
    OR esperada IS NULL OR esperada<0 OR esperada>9007199254740990
    OR jsonb_typeof(huellas) IS DISTINCT FROM 'array' OR jsonb_array_length(huellas) NOT BETWEEN 1 AND 4 THEN
    RAISE EXCEPTION 'Contacto3: intención inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:contacto:operacion:'||s,0));
 PERFORM set_config('vec.contacto.operacion_sujeto',s,true),set_config('vec.contacto.operacion_accion',p_accion,true);
 SELECT a.version INTO actual FROM vec_contacto_usuario_v1.actual a WHERE a.sujeto_ref=s FOR UPDATE;
 SELECT o.operacion_ref,o.hmac_clave_ref,o.hmac_valor,o.estado,o.version_resultante,o.recibo_ref INTO anterior
   FROM vec_contacto_usuario_v1.operaciones o
  WHERE o.sujeto_ref=s AND o.version_esperada=esperada AND o.estado IN ('preparada','confirmada')
  ORDER BY (o.estado='confirmada') DESC,o.preparada_en DESC,o.operacion_ref DESC LIMIT 1 FOR UPDATE;
 IF FOUND THEN
    ref_resultado:=anterior.operacion_ref; estado_resultado:=anterior.estado;
    version_resultado:=anterior.version_resultante; recibo_resultado:=anterior.recibo_ref;
    hmac_clave_resultado:=anterior.hmac_clave_ref; hmac_valor_resultado:=anterior.hmac_valor;
    IF EXISTS(SELECT 1 FROM jsonb_array_elements(huellas) h
        WHERE h->>'ClaveRef'=anterior.hmac_clave_ref AND h->>'ValorHMACSHA256'=anterior.hmac_valor) THEN
       repetida:=true;
    ELSE conflicto:='material'; END IF;
 ELSIF coalesce(actual,0)<>esperada THEN
    conflicto:='version'; ref_resultado:=''; estado_resultado:='conflicto';
 ELSE
    ref_resultado:=op;
    hmac_clave_resultado:=huellas->0->>'ClaveRef'; hmac_valor_resultado:=huellas->0->>'ValorHMACSHA256';
 END IF;
 auditada:=vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(
    p_accion,p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,
    consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
    CASE WHEN conflicto<>'' THEN 'conflicto' ELSE estado_resultado END,op);
 IF auditada->>'id' IS NULL OR auditada->>'id' !~ '^acc_[0-9a-f]{40}$' THEN
    RAISE EXCEPTION 'Contacto3: auditoría de preparación ausente' USING ERRCODE='55000';
 END IF;
 IF conflicto='' AND NOT repetida THEN
    INSERT INTO vec_contacto_usuario_v1.operaciones(operacion_ref,sujeto_ref,actor_id_hmac,version_esperada,
        hmac_clave_ref,hmac_valor,estado,auditoria_preparacion_ref)
      VALUES(op,s,b#>>'{Auditoria,actor_id}',esperada,huellas->0->>'ClaveRef',huellas->0->>'ValorHMACSHA256','preparada',auditada->>'id');
    INSERT INTO vec_contacto_usuario_v1.operacion_eventos(operacion_ref,sujeto_ref,secuencia,estado,auditoria_ref)
      VALUES(op,s,1,'preparada',auditada->>'id');
 END IF;
 SELECT * INTO STRICT revalidada FROM vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF revalidada.decision_ref IS DISTINCT FROM consumo.decision_ref
    OR revalidada.consumo_huella_sha256 IS DISTINCT FROM consumo.consumo_huella_sha256
    OR revalidada.revalidada_en IS NULL THEN
    RAISE EXCEPTION 'Contacto3: preparación perdió autorización' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT ref_resultado,
    estado_resultado,esperada,repetida,version_resultado,recibo_resultado,
    conflicto='material',conflicto='version',
    convert_to(auditada::text,'UTF8'),consumo.auditoria_ref::text,consumo.consumo_huella_sha256::text,
    hmac_clave_resultado,hmac_valor_resultado;
END $f$;

CREATE FUNCTION vec_contacto_usuario_v1.cancelar_operacion_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(operacion_ref text,estado text,version_esperada numeric,replay_confirmado boolean,
    encontrada boolean,conflicto boolean,auditoria_central bytea,consumo_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; s text; op text; o record; consumo record; revalidada record; auditada jsonb;
    estado_resultado text; hallada boolean:=false; repetida boolean:=false; cerrada boolean:=false;
BEGIN
 IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.operacion.cancelar'
    OR current_user<>'vec_contacto_usuario_owner' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
    RAISE EXCEPTION 'Contacto3: cancelación denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF consumo.consumo_nuevo IS DISTINCT FROM true THEN
    RAISE EXCEPTION 'Contacto3: cancelación exige V3 fresco' USING ERRCODE='P1102';
 END IF;
 b:=vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 s:=b->>'SujetoRef'; op:=b->>'OperacionRef';
 IF s IS NULL OR s !~ '^per_[A-Za-z0-9_-]{22,128}$' OR op IS NULL OR op !~ '^opr_[A-Za-z0-9_-]{22,128}$' THEN
    RAISE EXCEPTION 'Contacto3: selector de cancelación inválido' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:contacto:operacion:'||s,0));
 PERFORM set_config('vec.contacto.operacion_sujeto',s,true),set_config('vec.contacto.operacion_accion',p_accion,true);
 SELECT x.sujeto_ref,x.version_esperada,x.estado INTO o
   FROM vec_contacto_usuario_v1.operaciones x WHERE x.operacion_ref=op FOR UPDATE;
 hallada:=FOUND;
 IF NOT hallada THEN estado_resultado:='ausente';
 ELSIF o.estado='confirmada' THEN estado_resultado:='conflicto';
 ELSE estado_resultado:='cancelada'; repetida:=o.estado='cancelada';
 END IF;
 auditada:=vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(
    p_accion,p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,
    consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,estado_resultado,op);
 IF auditada->>'id' IS NULL OR auditada->>'id' !~ '^acc_[0-9a-f]{40}$' THEN
    RAISE EXCEPTION 'Contacto3: auditoría de cancelación ausente' USING ERRCODE='55000';
 END IF;
 IF hallada AND o.estado='preparada' THEN
    UPDATE vec_contacto_usuario_v1.operaciones x SET estado='cancelada',
        auditoria_cierre_ref=auditada->>'id',cerrada_en=clock_timestamp()
      WHERE x.operacion_ref=op AND x.estado='preparada';
    IF NOT FOUND THEN RAISE EXCEPTION 'Contacto3: cierre concurrente' USING ERRCODE='40001'; END IF;
    INSERT INTO vec_contacto_usuario_v1.operacion_eventos(operacion_ref,sujeto_ref,secuencia,estado,auditoria_ref)
      VALUES(op,s,2,'cancelada',auditada->>'id');
    cerrada:=true;
 END IF;
 SELECT * INTO STRICT revalidada FROM vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF revalidada.decision_ref IS DISTINCT FROM consumo.decision_ref
    OR revalidada.consumo_huella_sha256 IS DISTINCT FROM consumo.consumo_huella_sha256
    OR revalidada.revalidada_en IS NULL THEN
    RAISE EXCEPTION 'Contacto3: cancelación perdió autorización' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT CASE WHEN hallada THEN op ELSE '' END,estado_resultado,
    CASE WHEN hallada THEN o.version_esperada ELSE 0::numeric END,repetida,hallada,
    estado_resultado='conflicto',convert_to(auditada::text,'UTF8'),
    consumo.auditoria_ref::text,consumo.consumo_huella_sha256::text;
END $f$;

CREATE FUNCTION vec_contacto_usuario_v1.listar_operaciones_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(operaciones jsonb,siguiente_desde text,cursor_encontrado boolean,
    auditoria_central bytea,consumo_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; s text; cursor text; limite integer; fecha_cursor timestamptz(6);
    consumo record; revalidada record; auditada jsonb; total integer; salida jsonb; siguiente text:=''; valido boolean:=true;
BEGIN
 IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.operacion.listar'
    OR current_user<>'vec_contacto_usuario_owner' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
    RAISE EXCEPTION 'Contacto3: lista denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF consumo.consumo_nuevo IS DISTINCT FROM true THEN RAISE EXCEPTION 'Contacto3: lista exige V3 fresco' USING ERRCODE='P1102'; END IF;
 b:=vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 s:=b->>'SujetoRef'; cursor:=coalesce(b->>'DespuesDe',''); limite:=(b->>'Limite')::integer;
 IF s IS NULL OR s !~ '^per_[A-Za-z0-9_-]{22,128}$' OR limite NOT BETWEEN 1 AND 50
    OR (cursor<>'' AND cursor !~ '^opr_[A-Za-z0-9_-]{22,128}$') THEN
    RAISE EXCEPTION 'Contacto3: paginación inválida' USING ERRCODE='22023';
 END IF;
 PERFORM set_config('vec.contacto.operacion_sujeto',s,true),set_config('vec.contacto.operacion_accion',p_accion,true);
 IF cursor<>'' THEN
    SELECT o.preparada_en INTO fecha_cursor FROM vec_contacto_usuario_v1.operaciones o WHERE o.operacion_ref=cursor;
    valido:=FOUND;
 END IF;
 salida:='[]'::jsonb;
 IF valido THEN
    WITH pagina AS (
      SELECT o.operacion_ref,o.estado,o.version_esperada,o.version_resultante,o.recibo_ref,o.preparada_en,
             row_number() OVER (ORDER BY o.preparada_en DESC,o.operacion_ref DESC) AS fila
        FROM vec_contacto_usuario_v1.operaciones o
       WHERE o.sujeto_ref=s AND (cursor='' OR (o.preparada_en,o.operacion_ref)<(fecha_cursor,cursor))
       ORDER BY o.preparada_en DESC,o.operacion_ref DESC LIMIT limite+1
    )
    SELECT count(*)::integer,coalesce(jsonb_agg(jsonb_build_object(
             'operacion_ref',p.operacion_ref,'estado',p.estado,'version_esperada',p.version_esperada,
             'version',p.version_resultante,'recibo_ref',p.recibo_ref) ORDER BY p.fila)
             FILTER (WHERE p.fila<=limite),'[]'::jsonb),
           coalesce(max(p.operacion_ref) FILTER (WHERE p.fila=limite),'')
      INTO total,salida,siguiente FROM pagina p;
    IF total<=limite THEN siguiente:=''; END IF;
 END IF;
 auditada:=vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(
    p_accion,p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,
    consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,'consulta',cursor);
 SELECT * INTO STRICT revalidada FROM vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF revalidada.decision_ref IS DISTINCT FROM consumo.decision_ref
    OR revalidada.consumo_huella_sha256 IS DISTINCT FROM consumo.consumo_huella_sha256
    OR revalidada.revalidada_en IS NULL THEN RAISE EXCEPTION 'Contacto3: lista perdió autorización' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT salida,siguiente,valido,convert_to(auditada::text,'UTF8'),
    consumo.auditoria_ref::text,consumo.consumo_huella_sha256::text;
END $f$;

CREATE FUNCTION vec_contacto_usuario_v1.detalle_operacion_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(encontrada boolean,operacion jsonb,auditoria_central bytea,consumo_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; s text; op text; o record; consumo record; revalidada record; auditada jsonb; hallada boolean;
BEGIN
 IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.operacion.detalle'
    OR current_user<>'vec_contacto_usuario_owner' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
    RAISE EXCEPTION 'Contacto3: detalle denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF consumo.consumo_nuevo IS DISTINCT FROM true THEN RAISE EXCEPTION 'Contacto3: detalle exige V3 fresco' USING ERRCODE='P1102'; END IF;
 b:=vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 s:=b->>'SujetoRef'; op:=b->>'OperacionRef';
 IF s IS NULL OR s !~ '^per_[A-Za-z0-9_-]{22,128}$' OR op IS NULL OR op !~ '^opr_[A-Za-z0-9_-]{22,128}$' THEN
    RAISE EXCEPTION 'Contacto3: detalle inválido' USING ERRCODE='22023';
 END IF;
 PERFORM set_config('vec.contacto.operacion_sujeto',s,true),set_config('vec.contacto.operacion_accion',p_accion,true);
 SELECT x.operacion_ref,x.estado,x.version_esperada,x.version_resultante,x.recibo_ref INTO o
   FROM vec_contacto_usuario_v1.operaciones x WHERE x.operacion_ref=op;
 hallada:=FOUND;
 auditada:=vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(
    p_accion,p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,
    consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
    CASE WHEN hallada THEN 'encontrada' ELSE 'ausente' END,op);
 SELECT * INTO STRICT revalidada FROM vec_autorizacion_atestada_v3.revalidar_operacion_contacto_v3_atestada(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 IF revalidada.decision_ref IS DISTINCT FROM consumo.decision_ref
    OR revalidada.consumo_huella_sha256 IS DISTINCT FROM consumo.consumo_huella_sha256
    OR revalidada.revalidada_en IS NULL THEN RAISE EXCEPTION 'Contacto3: detalle perdió autorización' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT hallada,
    CASE WHEN hallada THEN jsonb_build_object('operacion_ref',o.operacion_ref,'estado',o.estado,
        'version_esperada',o.version_esperada,'version',o.version_resultante,'recibo_ref',o.recibo_ref)
        ELSE '{}'::jsonb END,
    convert_to(auditada::text,'UTF8'),consumo.auditoria_ref::text,consumo.consumo_huella_sha256::text;
END $f$;

CREATE FUNCTION vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(
    p_operacion_ref text,p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(sujeto_ref text,version numeric,auditoria_central bytea,consumo_ref text,
    consumo_huella_sha256 text,replay_confirmado boolean,hmac_clave_ref text,hmac_valor text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE b jsonb; r jsonb; s text; anterior numeric; nueva numeric; o record; v record; q record; recibo jsonb;
BEGIN
 IF p_operacion_ref IS NULL OR p_operacion_ref !~ '^opr_[A-Za-z0-9_-]{22,128}$'
    OR p_accion NOT IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar')
    OR p_negocio IS NULL OR p_recurso IS NULL OR current_user<>'vec_contacto_usuario_owner'
    OR session_user=current_user OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
    OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
    RAISE EXCEPTION 'Contacto3: confirmación denegada' USING ERRCODE='42501';
 END IF;
 b:=convert_from(p_negocio,'UTF8')::jsonb;
 r:=convert_from(p_recurso,'UTF8')::jsonb;
 s:=b->>'SujetoRef'; anterior:=(b->>'VersionEsperada')::numeric;
 nueva:=(b->>'VersionNueva')::numeric;
 IF s IS NULL OR s !~ '^per_[A-Za-z0-9_-]{22,128}$' OR anterior IS NULL OR nueva IS NULL
    OR nueva<>anterior+1 OR r#>>'{atributos,contacto_operacion_ref}' IS DISTINCT FROM p_operacion_ref
    OR r#>>'{atributos,contacto_sujeto_ref}' IS DISTINCT FROM s
    OR r#>>'{atributos,contacto_version_esperada}' IS DISTINCT FROM anterior::text
    OR (p_accion='vec.contacto_usuario.alta') IS DISTINCT FROM (anterior=0) THEN
    RAISE EXCEPTION 'Contacto3: intención no ligada al recurso V3' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:contacto:operacion:'||s,0));
 PERFORM set_config('vec.contacto.operacion_sujeto',s,true),set_config('vec.contacto.operacion_accion',p_accion,true);
 SELECT x.sujeto_ref,x.version_esperada,x.hmac_clave_ref,x.hmac_valor,x.estado,
        x.version_resultante,x.recibo_ref,x.consumo_ref INTO o
   FROM vec_contacto_usuario_v1.operaciones x WHERE x.operacion_ref=p_operacion_ref FOR UPDATE;
 IF NOT FOUND OR o.sujeto_ref IS DISTINCT FROM s OR o.version_esperada IS DISTINCT FROM anterior
    OR o.estado='cancelada' OR NOT EXISTS(
        SELECT 1 FROM jsonb_array_elements(b->'HuellasReplay') h
         WHERE h->>'ClaveRef'=o.hmac_clave_ref AND h->>'ValorHMACSHA256'=o.hmac_valor) THEN
    RAISE EXCEPTION 'Contacto3: operación ausente o material distinto' USING ERRCODE='P1103';
 END IF;
 SELECT * INTO STRICT v FROM vec_contacto_usuario_v1.registrar_contacto_v1(
    p_accion,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 recibo:=convert_from(v.auditoria_central,'UTF8')::jsonb;
 IF v.sujeto_ref IS DISTINCT FROM s OR v.version IS DISTINCT FROM nueva
    OR v.hmac_clave_ref IS DISTINCT FROM o.hmac_clave_ref OR v.hmac_valor IS DISTINCT FROM o.hmac_valor
    OR recibo->>'id' IS NULL OR recibo->>'id' !~ '^acc_[0-9a-f]{40}$' THEN
    RAISE EXCEPTION 'Contacto3: recibo no corresponde a intención' USING ERRCODE='55000';
 END IF;
 IF o.estado='confirmada' THEN
    IF v.replay_confirmado IS DISTINCT FROM true OR o.version_resultante IS DISTINCT FROM v.version
       OR o.recibo_ref IS DISTINCT FROM recibo->>'id' OR o.consumo_ref IS DISTINCT FROM v.consumo_ref THEN
       RAISE EXCEPTION 'Contacto3: replay divergente' USING ERRCODE='55000';
    END IF;
 ELSE
    IF v.replay_confirmado IS DISTINCT FROM false THEN
       RAISE EXCEPTION 'Contacto3: recibo previo ajeno a operación' USING ERRCODE='P1103';
    END IF;
    UPDATE vec_contacto_usuario_v1.operaciones x SET estado='confirmada',version_resultante=v.version,
        recibo_ref=recibo->>'id',consumo_ref=v.consumo_ref,auditoria_cierre_ref=recibo->>'id',cerrada_en=clock_timestamp()
      WHERE x.operacion_ref=p_operacion_ref AND x.estado='preparada';
    IF NOT FOUND THEN RAISE EXCEPTION 'Contacto3: cierre concurrente' USING ERRCODE='40001'; END IF;
    INSERT INTO vec_contacto_usuario_v1.operacion_eventos(operacion_ref,sujeto_ref,secuencia,estado,auditoria_ref)
      VALUES(p_operacion_ref,s,2,'confirmada',recibo->>'id');
 END IF;
 IF p_accion='vec.contacto_usuario.alta' THEN
    SELECT * INTO STRICT q FROM vec_autorizacion_atestada_v3.revalidar_alta_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 ELSE
    SELECT * INTO STRICT q FROM vec_autorizacion_atestada_v3.revalidar_actualizar_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
 END IF;
 IF q.decision_ref IS DISTINCT FROM recibo->>'authorization_ref'
    OR q.consumo_huella_sha256 IS DISTINCT FROM v.consumo_huella_sha256
    OR q.revalidada_en IS NULL THEN
    RAISE EXCEPTION 'Contacto3: autorización dejó de estar vigente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT v.sujeto_ref::text,v.version::numeric,v.auditoria_central::bytea,
    v.consumo_ref::text,v.consumo_huella_sha256::text,v.replay_confirmado::boolean,
    v.hmac_clave_ref::text,v.hmac_valor::text;
END $f$;

-- El escritor sólo entra por la operación preparada. El registro V1 permanece
-- invocable por el owner desde confirmar_operacion_contacto_v1, con sus
-- invariantes originales de V3, versión, T13 y outbox.
REVOKE EXECUTE ON FUNCTION vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM vec_contacto_usuario_writer;
REVOKE ALL ON FUNCTION vec_contacto_usuario_v1.preparar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contacto_usuario_v1.cancelar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contacto_usuario_v1.listar_operaciones_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contacto_usuario_v1.detalle_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.preparar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.cancelar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.listar_operaciones_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.detalle_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
GRANT EXECUTE ON FUNCTION vec_contacto_usuario_v1.confirmar_operacion_contacto_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)
    TO vec_contacto_usuario_writer;
DO $post$
DECLARE f regprocedure; nombre text;
BEGIN
 IF current_user<>'vec_contacto_usuario_owner'
    OR has_function_privilege('vec_contacto_usuario_writer',
       'vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.operaciones','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.operacion_eventos','SELECT,INSERT,UPDATE,DELETE') THEN
    RAISE EXCEPTION 'Contacto3: ACL final incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH nombre IN ARRAY ARRAY['preparar_operacion_contacto_v1','cancelar_operacion_contacto_v1',
        'listar_operaciones_contacto_v1','detalle_operacion_contacto_v1','confirmar_operacion_contacto_v1'] LOOP
    SELECT p.oid::regprocedure INTO STRICT f FROM pg_proc p
      WHERE p.pronamespace='vec_contacto_usuario_v1'::regnamespace AND p.proname=nombre;
    IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=current_user::regrole
         AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0)
       OR NOT has_function_privilege('vec_contacto_usuario_writer',f,'EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_reader',f,'EXECUTE') THEN
       RAISE EXCEPTION 'Contacto3: función nominal sin ACL exacta' USING ERRCODE='42501';
    END IF;
 END LOOP;
END $post$;
-- Falta congelar la postimagen AD3-54, completar DOWN y ejecutar E10/PG18.
-- Se deja una guarda explícita y se retira sólo con esas dependencias cerradas.
DO $incompleta$ BEGIN RAISE EXCEPTION 'Contacto3 WIP: falta AD3 post-CT51 y validación' USING ERRCODE='55000'; END $incompleta$;
