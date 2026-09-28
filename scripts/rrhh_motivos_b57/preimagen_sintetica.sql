\set ON_ERROR_STOP on
-- Solo contenedor desechable. AD3-101/102 se aplican desde sus SQL reales;
-- B12/B16/B35 y el consumidor de auditoría son dobles declarados.
CREATE ROLE vec_bolsa_llamamientos_migrador NOLOGIN;
CREATE ROLE vec_b57_rrhh LOGIN INHERIT;
CREATE ROLE vec_b57_revisor LOGIN INHERIT;
CREATE ROLE vec_b57_auditor LOGIN;
CREATE ROLE vec_b57_sin_permiso LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b57_rrhh;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b57_revisor;
CREATE SCHEMA vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_b57_auditor;
GRANT CONNECT ON DATABASE postgres TO vec_b57_rrhh,vec_b57_revisor,vec_b57_auditor,vec_b57_sin_permiso;
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$ BEGIN RAISE EXCEPTION 'inmutable'; END $$;
CREATE TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (
 intento_ref text PRIMARY KEY,correlacion_ref text NOT NULL,
 accion text NOT NULL CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK (accion IN
  ('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento')),
 ruta_clase text NOT NULL CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK (ruta_clase IN
  ('coleccion','detalle','situacion','contactos','datos_contacto','emisiones')),
 actor_ref text,resultado text NOT NULL,registrada_en timestamptz NOT NULL
);
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FORCE ROW LEVEL SECURITY;
CREATE POLICY bitacora_b57_propietario ON vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
 TO vec_bolsa_llamamientos_propietario USING (current_user='vec_bolsa_llamamientos_propietario') WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user<>'vec_b57_auditor' THEN
  RAISE EXCEPTION 'registrador de ensayo no autorizado' USING ERRCODE='42501';
 END IF;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
 p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento')
    OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos','datos_contacto','emisiones')
    OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$')
    OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado','correcto')
 THEN RAISE EXCEPTION 'intento de frontera inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento VALUES
  (v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;
CREATE TABLE vec_bolsa_llamamientos.situacion_participacion(
 participacion_ref text NOT NULL,desde timestamptz NOT NULL,situacion text NOT NULL,fecha_disponible timestamptz,
 motivo text NOT NULL,actor text NOT NULL,clave_idempotencia text NOT NULL,recibo_ref text NOT NULL,registrada_en timestamptz NOT NULL,
 PRIMARY KEY(participacion_ref,desde),UNIQUE(participacion_ref,clave_idempotencia),UNIQUE(recibo_ref));
CREATE TABLE vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref text,desde timestamptz,operacion text,justificante_tipo text,justificante_ref text,justificante_sha256 text,actor text,validador text,validada_en timestamptz,PRIMARY KEY(participacion_ref,desde));
CREATE TABLE vec_bolsa_llamamientos.datos_contacto_participacion(
 participacion_ref text NOT NULL,version bigint NOT NULL,clave_ref text NOT NULL,nonce bytea NOT NULL,cifrado bytea NOT NULL,
 motivo text NOT NULL,actor text NOT NULL,registrada_en timestamptz NOT NULL,clave_idempotencia text NOT NULL,recibo_ref text NOT NULL,
 PRIMARY KEY(participacion_ref,version),UNIQUE(participacion_ref,clave_idempotencia),UNIQUE(recibo_ref));
CREATE TABLE vec_bolsa_llamamientos.traza_valor_participacion(participacion_ref text,recibo_ref text,campo text,valor_anterior text,valor_nuevo text,actor text,registrada_en timestamptz,PRIMARY KEY(participacion_ref,recibo_ref,campo));
CREATE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
 text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(reutilizada boolean,recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 DECLARE x record; BEGIN SELECT * INTO x FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=$2 AND clave_idempotencia=$8;
 IF FOUND THEN RETURN QUERY SELECT true,x.recibo_ref,x.situacion,x.desde,x.fecha_disponible; RETURN; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion VALUES($2,$4,$3,$5,$6,$7,$8,$9,$10);
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion) VALUES($2,$4,'cambiar');
 INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion VALUES($2,$9,'situacion','anterior',$3,$7,$10);
 RETURN QUERY SELECT false,$9,$3,$4,$5; END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(
 text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,registrada_en timestamptz)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 DECLARE x record; BEGIN SELECT * INTO x FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE participacion_ref=$2 AND clave_idempotencia=$10;
 IF FOUND THEN RETURN QUERY SELECT true,x.recibo_ref,x.version,x.registrada_en; RETURN; END IF;
 INSERT INTO vec_bolsa_llamamientos.datos_contacto_participacion VALUES($2,$3,$4,$5,$6,$7,$8,$9,$10,$11);
 INSERT INTO vec_bolsa_llamamientos.traza_valor_participacion VALUES($2,$11,'datos_contacto','version:0','version:'||$3,$8,$9);
 RETURN QUERY SELECT false,$11,$3,$9; END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(
 text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text)
 RETURNS TABLE(o_reutilizada boolean,o_recibo_ref text,o_version bigint,o_registrada_en timestamptz)
 LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT reutilizada,recibo_ref,version,registrada_en FROM vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
 $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
 text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(reutilizada boolean,recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz)
 LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT reutilizada,recibo_ref,situacion,desde,fecha_disponible FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
 $f$;
CREATE FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(
 text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(desde timestamptz,operacion text,situacion text,justificante_tipo text,justificante_ref text,
 justificante_sha256 text,actor text,validador text,validada_en timestamptz,motivo text)
 LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT s.desde,o.operacion,s.situacion,o.justificante_tipo,o.justificante_ref,o.justificante_sha256,
 o.actor,o.validador,o.validada_en,s.motivo
 FROM vec_bolsa_llamamientos.operacion_situacion_participacion o
 JOIN vec_bolsa_llamamientos.situacion_participacion s USING(participacion_ref,desde)
 WHERE o.participacion_ref=$1
 $f$;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_llamamientos FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) TO vec_b57_auditor;
RESET ROLE;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 RETURNS TABLE(efecto_ref text,huella_efecto_sha256 text,consumo_nuevo boolean) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$ SELECT convert_from($1,'UTF8')::jsonb->>'efecto_ref',convert_from($1,'UTF8')::jsonb->>'huella_efecto_sha256',true $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
