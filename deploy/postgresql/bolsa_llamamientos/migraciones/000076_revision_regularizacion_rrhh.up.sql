\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000076',0));

-- RRHH18: espera documental y regularización tras fin de causa. No crea
-- suspensión temporal ni publica una política; el catálogo la versiona.
-- B73 es requisito: conserva sus guardas de exclusión y sanción viva.
DO $preimagen$
DECLARE n text; h text; v_actual text;
BEGIN
 FOR n,h IN SELECT * FROM (VALUES
 ('registrar_situacion_participacion_interna_b73','816f45f22f667a352820adc3d052e67a'),
 ('registrar_operacion_situacion_participacion_v1','7f1241c406e24de298ff4d413d71011f'),
 ('transiciones_situacion_canonicas','bd5fcddbdef391a3bef7436f1123e351')) AS t(n,h) LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
     WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname=n
       AND pg_get_userbyid(p.proowner)=current_user AND md5(p.prosrc)=h) THEN
   SELECT string_agg(md5(p.prosrc),',' ORDER BY p.oid) INTO v_actual
     FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
    WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname=n;
   RAISE EXCEPTION 'preimagen B76 incompatible: clave=%, actual=%, esperado=%',
     n,coalesce(v_actual,'ausente'),h USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.sancion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NULL
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
      WHERE ns.nspname='vec_bolsa_llamamientos' AND p.proname IN
       ('registrar_situacion_participacion_interna_b76','registrar_operacion_situacion_participacion_v2')) THEN
  RAISE EXCEPTION 'estado incompatible B76' USING ERRCODE='55000';
 END IF;
 -- Una fachada SECURITY DEFINER nunca adopta concesiones ajenas o PUBLIC.
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE ns.nspname='vec_bolsa_llamamientos'
      AND p.proname IN ('registrar_situacion_participacion_v1','registrar_operacion_situacion_participacion_v1')
      AND a.privilege_type='EXECUTE'
      AND a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::regrole)) THEN
  RAISE EXCEPTION 'ACL de fachada B76 incompatible' USING ERRCODE='42501';
 END IF;
END $preimagen$;
-- Se amplían restricciones, sin modificar ninguna fila histórica.
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion
 DROP CONSTRAINT situacion_participacion_situacion_check,
 ADD CONSTRAINT situacion_participacion_situacion_check
 CHECK (situacion IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision'));
ALTER TABLE vec_bolsa_llamamientos.operacion_situacion_participacion
 DROP CONSTRAINT operacion_situacion_participacion_operacion_check,
 ADD CONSTRAINT operacion_situacion_participacion_operacion_check
 CHECK (operacion IN ('pausar','reactivar','excluir','revisar','regularizar')),
 ADD COLUMN situacion_esperada_desde timestamptz(6),
 ADD COLUMN causa_finalizada_en timestamptz(6),
 ADD CONSTRAINT operacion_situacion_participacion_regularizacion_b76
 CHECK ((operacion NOT IN ('revisar','regularizar') OR situacion_esperada_desde IS NOT NULL)
   AND (operacion <> 'regularizar' OR (causa_finalizada_en IS NOT NULL AND isfinite(causa_finalizada_en)
     AND causa_finalizada_en <= validada_en AND causa_finalizada_en <= desde))
   AND (operacion = 'regularizar' OR causa_finalizada_en IS NULL));

-- La traza B34 mantiene su minimización y admite el código nuevo.
ALTER TABLE vec_bolsa_llamamientos.traza_valor_participacion
 DROP CONSTRAINT traza_valor_participacion_check1,
 ADD CONSTRAINT traza_valor_participacion_check1
 CHECK (CASE campo
      WHEN 'situacion' THEN coalesce(valor_anterior,'disponible') IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision')
                        AND valor_nuevo IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision')
      WHEN 'fecha_disponible' THEN coalesce(valor_anterior,'2000-01-01T00:00:00.000000Z') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
                        AND coalesce(valor_nuevo,'2000-01-01T00:00:00.000000Z') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
      ELSE coalesce(valor_anterior,'version:1') ~ '^version:[1-9][0-9]{0,18}$' AND valor_nuevo ~ '^version:[1-9][0-9]{0,18}$'
    END);

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.transiciones_situacion_canonicas(p_transiciones text[])
RETURNS text[] LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $f$
DECLARE
 v_orden constant text[] := ARRAY['disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision'];
 v_canonica text[];
 v_par text;
 v_origen text;
BEGIN
 IF array_ndims(p_transiciones) IS DISTINCT FROM 1 OR array_position(p_transiciones, NULL) IS NOT NULL
    OR cardinality(p_transiciones) > 56 THEN
  RETURN NULL;
 END IF;
 FOREACH v_par IN ARRAY p_transiciones LOOP
  IF split_part(v_par, '>', 1) <> ALL (v_orden) OR split_part(v_par, '>', 2) <> ALL (v_orden)
     OR v_par <> split_part(v_par, '>', 1) || '>' || split_part(v_par, '>', 2)
     OR split_part(v_par, '>', 1) = split_part(v_par, '>', 2)
     OR (split_part(v_par, '>', 1) = 'excluido' AND v_par <> 'excluido>disponible') THEN
   RETURN NULL;
  END IF;
 END LOOP;
 v_canonica := ARRAY(
  SELECT o.s || '>' || d.s
    FROM unnest(v_orden) WITH ORDINALITY AS o(s, n)
    CROSS JOIN unnest(v_orden) WITH ORDINALITY AS d(s, m)
   WHERE (o.s || '>' || d.s) = ANY (p_transiciones)
   ORDER BY o.n, d.m);
 IF cardinality(v_canonica) <> cardinality(p_transiciones) THEN
  RETURN NULL;
 END IF;
 FOREACH v_origen IN ARRAY v_orden LOOP
  IF v_origen <> 'excluido'
     AND (v_origen <> 'en_revision' OR EXISTS (SELECT 1 FROM unnest(v_canonica) par WHERE par LIKE 'en_revision>%' OR par LIKE '%>en_revision'))
     AND NOT ((v_origen || '>excluido') = ANY (v_canonica)) THEN
   RETURN NULL;
  END IF;
 END LOOP;
 RETURN v_canonica;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76(p_bolsa_ref text,p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_reincorporacion_justificada boolean,p_situacion_esperada_desde timestamptz,p_operacion_b76 text,p_causa_finalizada_en timestamptz)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog SET lock_timeout='2s' AS $f$
DECLARE anterior record; consumo record; decision jsonb; v_politica record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_reincorporacion_justificada IS NULL OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_situacion IS NULL OR p_situacion NOT IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde','en_revision') OR p_desde IS NULL OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_clave_idempotencia<>btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256 OR p_recibo_ref IS NULL OR p_recibo_ref<>btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 OR p_registrada_en IS NULL OR (p_situacion='disponible_desde' AND p_fecha_disponible IS NULL) OR (p_situacion<>'disponible_desde' AND p_fecha_disponible IS NOT NULL) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='situacion invalida'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='participacion ajena a la bolsa'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:situacion:' || p_participacion_ref, 0));
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p_participacion_ref ORDER BY desde DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar' OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia AND (s.situacion<>p_situacion OR s.motivo<>p_motivo OR s.fecha_disponible IS DISTINCT FROM p_fecha_disponible OR (p_operacion_b76 IS NOT NULL AND s.actor IS DISTINCT FROM p_actor))) THEN RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro comando'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia) THEN RETURN QUERY SELECT true, s.recibo_ref, s.situacion, s.desde, s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia; RETURN; END IF;
 -- B76: el replay ya se comprobó con una autorización nueva. Las guardas
 -- siguientes rigen efectos nuevos; la historia anterior sigue recuperable.
 IF p_operacion_b76 IS NOT NULL AND
    (p_situacion_esperada_desde IS NULL OR anterior.desde IS DISTINCT FROM p_situacion_esperada_desde) THEN
  RAISE EXCEPTION USING ERRCODE='VBS02', MESSAGE='situacion esperada distinta';
 END IF;
 IF p_situacion = 'en_revision' THEN
  IF p_operacion_b76 IS DISTINCT FROM 'revisar'
     OR (anterior.situacion='renuncia' OR (anterior.situacion='no_disponible'
       AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
         WHERE o.participacion_ref=p_participacion_ref AND o.desde=anterior.desde AND o.operacion='pausar')
       AND (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
         WHERE s.participacion_ref=p_participacion_ref AND s.desde<anterior.desde
         ORDER BY s.desde DESC LIMIT 1)='renuncia')) IS NOT TRUE THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='revision sin antecedente de renuncia';
  END IF;
 END IF;
 IF anterior.situacion='en_revision' AND
    ((p_operacion_b76='regularizar' AND p_situacion='disponible') OR
         (p_operacion_b76='excluir' AND p_situacion='excluido')) IS NOT TRUE THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='revision exige operacion nominal RRHH';
 END IF;
 IF p_situacion='disponible' AND anterior.situacion IN ('renuncia','no_disponible','en_revision','excluido') THEN
  IF p_operacion_b76 IS DISTINCT FROM 'regularizar' OR p_causa_finalizada_en IS NULL
     OR anterior.situacion='no_disponible'
     OR (anterior.situacion='excluido' AND
        (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
         WHERE s.participacion_ref=p_participacion_ref AND s.desde<anterior.desde
         ORDER BY s.desde DESC LIMIT 1) IS DISTINCT FROM 'renuncia') THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='regularizacion exige fin de causa validado';
  END IF;
 END IF;
 IF p_operacion_b76='regularizar' AND anterior.situacion NOT IN ('renuncia','en_revision','excluido') THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='regularizacion sin antecedente';
 END IF;
 -- Una salida de excluido exige la operación B8 de exclusión original y que
 -- ninguna sanción de exclusión siga viva sobre ese mismo efecto.
 IF anterior.situacion = 'excluido' THEN
  IF p_reincorporacion_justificada IS NOT TRUE OR p_situacion <> 'disponible'
     OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
                     WHERE o.participacion_ref = p_participacion_ref AND o.desde = anterior.desde
                       AND o.operacion = 'excluir')
     OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sancion_participacion sa
                 WHERE sa.participacion_ref = p_participacion_ref AND sa.efecto = 'excluir'
                   AND sa.situacion_desde = anterior.desde
                   AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                    WHERE rv.sancion_ref = sa.sancion_ref)) THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='reincorporacion no admitida';
  END IF;
 END IF;
 IF p_desde < anterior.desde THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='desde anterior a la situacion vigente'; END IF;
 IF p_situacion='disponible_desde' AND p_fecha_disponible<=p_registrada_en THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='fecha disponible no futura'; END IF;
 -- 000032: la tabla de transiciones es la política vigente, no un literal.
 -- Cerrojo compartido frente a la publicación, que toma el exclusivo.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion', 0));
 SELECT p.version, p.transiciones INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_transiciones_situacion p ORDER BY p.version DESC LIMIT 1;
 IF NOT ((anterior.situacion || '>' || p_situacion) = ANY (v_politica.transiciones)) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version) VALUES(p_participacion_ref,p_situacion,p_desde,NULL,p_fecha_disponible,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref,v_politica.version);
 RETURN QUERY SELECT false,p_recibo_ref,p_situacion,p_desde,p_fecha_disponible;
END $f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b73(p_bolsa_ref text,p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_reincorporacion_justificada boolean)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76(
  p_bolsa_ref,p_participacion_ref,p_situacion,p_desde,p_fecha_disponible,p_motivo,p_actor,p_clave_idempotencia,
  p_recibo_ref,p_registrada_en,p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,
  p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,p_reincorporacion_justificada,NULL,NULL,NULL);
END $f$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
 p_bolsa_ref text, p_participacion_ref text, p_operacion text, p_desde timestamptz, p_fecha_disponible timestamptz,
 p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,
 p_justificante_tipo text, p_justificante_ref text, p_justificante_sha256 text, p_validador text, p_validada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_situacion text; v_cambio record; v_previa record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 v_situacion := CASE p_operacion WHEN 'pausar' THEN 'no_disponible' WHEN 'reactivar' THEN 'disponible' WHEN 'excluir' THEN 'excluido' END;
 IF v_situacion IS NULL
    OR p_justificante_tipo NOT IN ('solicitud_candidato','informe_medico','resolucion','correo','acta_bolsa','otro')
    OR p_justificante_ref IS NULL OR p_justificante_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$' OR octet_length(p_justificante_ref) NOT BETWEEN 1 AND 256
    OR p_justificante_sha256 IS NULL OR p_justificante_sha256 !~ '^[a-f0-9]{64}$'
    OR p_validador IS NULL OR p_validador <> btrim(p_validador) OR octet_length(p_validador) NOT BETWEEN 1 AND 256
    OR p_validada_en IS NULL OR p_registrada_en IS NULL OR p_validada_en > p_registrada_en
    OR (p_operacion = 'excluir' AND p_validador = p_actor)
    OR p_fecha_disponible IS NOT NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='operacion invalida';
 END IF;

 -- La autorización positiva se consume también en replay; el validador
 -- declarado es solo traza. El actor debe coincidir con la decisión V3.
 SELECT o.*, s.situacion AS situacion_registrada, s.recibo_ref AS recibo_registrado, s.fecha_disponible AS fecha_registrada
   INTO v_previa
   FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
  WHERE o.participacion_ref = p_participacion_ref AND o.clave_idempotencia = p_clave_idempotencia;
 IF v_previa.participacion_ref IS NULL AND p_operacion IN ('pausar','reactivar') THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='operacion historica sin efectos nuevos';
 END IF;
 SELECT * INTO STRICT v_cambio FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b73(
   p_bolsa_ref, p_participacion_ref, v_situacion, p_desde, NULL, p_motivo, p_actor, p_clave_idempotencia,
   p_recibo_ref, p_registrada_en, p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version,
   p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz, p_operacion = 'reactivar');
 IF v_cambio.reutilizada THEN
  -- Los instantes desde/validada_en pertenecen a la primera ejecución;
  -- un reintento autorizado conserva esos valores sin exigir el mismo reloj.
  IF v_previa.participacion_ref IS NULL OR v_previa.operacion <> p_operacion OR v_previa.justificante_tipo <> p_justificante_tipo
     OR v_previa.justificante_ref <> p_justificante_ref OR v_previa.justificante_sha256 <> p_justificante_sha256
     OR v_previa.validador <> p_validador OR v_previa.actor <> p_actor THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra operacion';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_registrado, v_previa.situacion_registrada, v_previa.desde, v_previa.fecha_registrada;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
   participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256,
   actor, validador, validada_en, registrada_en, clave_idempotencia)
 VALUES (p_participacion_ref, v_cambio.desde, p_operacion, p_justificante_tipo, p_justificante_ref,
   p_justificante_sha256, p_actor, p_validador, p_validada_en, p_registrada_en, p_clave_idempotencia);
 RETURN QUERY SELECT false, v_cambio.recibo_ref, v_cambio.situacion, v_cambio.desde, v_cambio.fecha_disponible;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(
 p_bolsa_ref text, p_participacion_ref text, p_operacion text, p_desde timestamptz, p_fecha_disponible timestamptz,
 p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,
 p_justificante_tipo text, p_justificante_ref text, p_justificante_sha256 text, p_validador text, p_validada_en timestamptz,
 p_situacion_esperada_desde timestamptz, p_causa_finalizada_en timestamptz,
 p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric,
 p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_situacion text; v_cambio record; v_previa record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='operacion no autorizada'; END IF;
 v_situacion := CASE p_operacion WHEN 'revisar' THEN 'en_revision' WHEN 'regularizar' THEN 'disponible' WHEN 'excluir' THEN 'excluido' END;
 IF current_setting('transaction_isolation') <> 'serializable'
    OR p_desde>p_registrada_en
    OR p_situacion_esperada_desde IS NULL OR NOT isfinite(p_situacion_esperada_desde)
    OR p_desde IS NULL OR NOT isfinite(p_desde) OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_validada_en IS NULL OR NOT isfinite(p_validada_en)
    OR (p_operacion='regularizar' AND (p_causa_finalizada_en IS NULL OR NOT isfinite(p_causa_finalizada_en) OR p_causa_finalizada_en>p_validada_en OR p_causa_finalizada_en>p_desde))
    OR (p_operacion<>'regularizar' AND p_causa_finalizada_en IS NOT NULL)
    OR v_situacion IS NULL
    OR p_justificante_tipo NOT IN ('solicitud_candidato','informe_medico','resolucion','correo','acta_bolsa','otro')
    OR p_justificante_ref IS NULL OR p_justificante_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$' OR octet_length(p_justificante_ref) NOT BETWEEN 1 AND 256
    OR p_justificante_sha256 IS NULL OR p_justificante_sha256 !~ '^[a-f0-9]{64}$'
    OR p_validador IS NULL OR p_validador <> btrim(p_validador) OR octet_length(p_validador) NOT BETWEEN 1 AND 256
    OR p_validada_en IS NULL OR p_registrada_en IS NULL OR p_validada_en > p_registrada_en
    OR (p_operacion = 'excluir' AND p_validador = p_actor)
    OR p_fecha_disponible IS NOT NULL THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='operacion invalida';
 END IF;

 -- La autorización positiva se consume también en replay; el validador
 -- declarado es solo traza. El actor debe coincidir con la decisión V3.
 SELECT o.*, s.situacion AS situacion_registrada, s.recibo_ref AS recibo_registrado, s.fecha_disponible AS fecha_registrada
   INTO v_previa
   FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
   JOIN vec_bolsa_llamamientos.situacion_participacion s USING (participacion_ref, desde)
  WHERE o.participacion_ref = p_participacion_ref AND o.clave_idempotencia = p_clave_idempotencia;
 SELECT * INTO STRICT v_cambio FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76(
   p_bolsa_ref, p_participacion_ref, v_situacion, p_desde, NULL, p_motivo, p_actor, p_clave_idempotencia,
   p_recibo_ref, p_registrada_en, p_capacidad, p_decision, p_motivo_autorizacion, p_contexto, p_persona_version,
   p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz, p_operacion = 'regularizar',p_situacion_esperada_desde,p_operacion,p_causa_finalizada_en);
 IF v_cambio.reutilizada THEN
  -- Los instantes desde/validada_en pertenecen a la primera ejecución;
  -- un reintento autorizado conserva esos valores sin exigir el mismo reloj.
  IF v_previa.participacion_ref IS NULL OR v_previa.operacion <> p_operacion OR v_previa.justificante_tipo <> p_justificante_tipo
     OR v_previa.justificante_ref <> p_justificante_ref OR v_previa.justificante_sha256 <> p_justificante_sha256
     OR v_previa.validador <> p_validador OR v_previa.actor <> p_actor
     OR v_previa.situacion_esperada_desde IS DISTINCT FROM p_situacion_esperada_desde
     OR v_previa.causa_finalizada_en IS DISTINCT FROM p_causa_finalizada_en THEN
   RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otra operacion';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_registrado, v_previa.situacion_registrada, v_previa.desde, v_previa.fecha_registrada;
  RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(
   participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256,
   actor, validador, validada_en, registrada_en, clave_idempotencia,situacion_esperada_desde,causa_finalizada_en)
 VALUES (p_participacion_ref, v_cambio.desde, p_operacion, p_justificante_tipo, p_justificante_ref,
   p_justificante_sha256, p_actor, p_validador, p_validada_en, p_registrada_en, p_clave_idempotencia,p_situacion_esperada_desde,p_causa_finalizada_en);
 RETURN QUERY SELECT false, v_cambio.recibo_ref, v_cambio.situacion, v_cambio.desde, v_cambio.fecha_disponible;
END $f$;

-- El helper conserva denegación por defecto incluso con default privileges.
DO $acl$
DECLARE r record; f regprocedure;
BEGIN
 SELECT p.oid::regprocedure INTO STRICT f FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_situacion_participacion_interna_b76';
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
 FOR r IN SELECT DISTINCT a.grantee,pg_get_userbyid(a.grantee) rol FROM pg_proc p
   CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>0 AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,r.rol);
 END LOOP;
 SELECT p.oid::regprocedure INTO STRICT f FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_operacion_situacion_participacion_v2';
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
 FOR r IN SELECT DISTINCT a.grantee,pg_get_userbyid(a.grantee) rol FROM pg_proc p
   CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>0 AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,r.rol);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f);
END $acl$;
COMMIT;
