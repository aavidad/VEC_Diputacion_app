\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:000006',0));
DO $pre$ BEGIN
 IF current_user<>'vec_identidad_sesiones_v1_propietario'
    OR to_regprocedure('vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_identidad_sesiones_v1.registro_propio_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'identidad 000006: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La referencia institucional es opaca y la equivalencia llega acreditada
-- por un proveedor externo. Este índice impide crear una segunda persona con
-- el mismo sujeto estable; no intenta inferir equivalencia por DNI o correo.
CREATE TABLE vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 (
 sujeto_ref text PRIMARY KEY CHECK(sujeto_ref ~ '^suj_[A-Za-z0-9_-]{22,128}$'),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 prueba_ref text NOT NULL CHECK(prueba_ref ~ '^[a-z]{3}_[A-Za-z0-9_-]{22,128}$'),
 prueba_version numeric(20,0) NOT NULL CHECK(prueba_version BETWEEN 1 AND 18446744073709551615::numeric),
 prueba_sha256 text NOT NULL CHECK(prueba_sha256 ~ '^[0-9a-f]{64}$'),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$')
);
CREATE TABLE vec_identidad_sesiones_v1.registro_propio_v1 (
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^rpr_[A-Za-z0-9_-]{22,128}$'),
 entrada_sha256 text NOT NULL CHECK(entrada_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 sujeto_ref text NOT NULL UNIQUE REFERENCES vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1(sujeto_ref),
 cuenta_ref text NOT NULL UNIQUE REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
 cuenta_version numeric(20,0) NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 persona_version numeric(20,0) NOT NULL,
 perfil_ref text NOT NULL UNIQUE CHECK(perfil_ref ~ '^prf_[A-Za-z0-9_-]{22,128}$'),
 perfil_version numeric(20,0) NOT NULL,
 vinculo_ref text NOT NULL UNIQUE CHECK(vinculo_ref ~ '^vca_[A-Za-z0-9_-]{22,128}$'),
 vinculo_version numeric(20,0) NOT NULL,
 procedencia_ref text NOT NULL CHECK(procedencia_ref ~ '^prc_[A-Za-z0-9_-]{22,128}$'),
 procedencia_version numeric(20,0) NOT NULL,
 procedencia_sha256 text NOT NULL CHECK(procedencia_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 estado text NOT NULL CHECK(estado='pendiente_contacto'),
 registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY sujeto_persona_solo_propietario ON vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
ALTER TABLE vec_identidad_sesiones_v1.registro_propio_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.registro_propio_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY registro_propio_solo_propietario ON vec_identidad_sesiones_v1.registro_propio_v1 TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
CREATE TRIGGER sujeto_persona_inmutable BEFORE UPDATE OR DELETE ON vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TRIGGER registro_propio_inmutable BEFORE UPDATE OR DELETE ON vec_identidad_sesiones_v1.registro_propio_v1 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
REVOKE ALL ON vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1,vec_identidad_sesiones_v1.registro_propio_v1 FROM PUBLIC;

CREATE TABLE vec_identidad_sesiones_v1.registro_propio_outbox_v1 (
 operacion_ref text PRIMARY KEY REFERENCES vec_identidad_sesiones_v1.registro_propio_v1(operacion_ref),
 evento_ref text NOT NULL UNIQUE CHECK(evento_ref ~ '^evt_[A-Za-z0-9_-]{22,128}$'),
 tipo text NOT NULL CHECK(tipo='registro_pendiente_contacto'),
 creada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_identidad_sesiones_v1.registro_propio_outbox_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.registro_propio_outbox_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY registro_propio_outbox_solo_propietario ON vec_identidad_sesiones_v1.registro_propio_outbox_v1 TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
CREATE TRIGGER registro_propio_outbox_inmutable BEFORE UPDATE OR DELETE ON vec_identidad_sesiones_v1.registro_propio_outbox_v1 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
REVOKE ALL ON vec_identidad_sesiones_v1.registro_propio_outbox_v1 FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.registrar_propio_v1(
 p_entrada bytea,p_recurso bytea,p_actor text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_entrada jsonb; v_acred jsonb; v_equiv jsonb; v_recurso jsonb; v_decision jsonb;
 v_operacion text; v_sujeto text; v_persona text; v_cuenta text; v_perfil text; v_vinculo text; v_recibo text;
 v_huella text; v_prev record; v_consumo record; v_version record; v_ahora timestamptz(6);
 v_desde timestamptz; v_hasta timestamptz; v_nueva boolean;
BEGIN
 IF current_user<>'vec_identidad_sesiones_v1_propietario'
    OR NOT pg_has_role(session_user,'vec_identidad_sesiones_v1_provisionador','MEMBER')
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR octet_length(p_entrada) NOT BETWEEN 1 AND 8192
    OR octet_length(p_recurso) NOT BETWEEN 1 AND 2048
 THEN RAISE EXCEPTION 'registro propio denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  v_entrada:=convert_from(p_entrada,'UTF8')::jsonb; v_recurso:=convert_from(p_recurso,'UTF8')::jsonb;
  v_decision:=convert_from(p_decision,'UTF8')::jsonb;
  v_operacion:=v_entrada->>'Operacion'; v_acred:=v_entrada->'Acreditacion'; v_equiv:=v_entrada->'Equivalencia';
  v_sujeto:=v_acred->>'SujetoRef'; v_nueva:=(v_equiv->>'Nueva')::boolean;
  v_desde:=(v_acred->>'VigenteDesde')::timestamptz; v_hasta:=(v_acred->>'VigenteHasta')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'registro propio: entrada inválida' USING ERRCODE='22023'; END;
 v_ahora:=clock_timestamp(); v_huella:=encode(sha256(p_entrada),'hex');
 IF v_operacion !~ '^opr_[A-Za-z0-9_-]{22,128}$' OR v_sujeto !~ '^suj_[A-Za-z0-9_-]{22,128}$'
    OR v_acred->>'CredencialRef' !~ '^[a-z]{3}_[A-Za-z0-9_-]{22,128}$'
    OR v_acred->>'DominioHMACRef' !~ '^idh_[A-Za-z0-9_-]{22,128}$'
    OR v_acred->>'ClaveHMACID' IS NULL OR (v_acred->>'ClaveHMACVersion')::numeric<1
    OR octet_length(decode(v_acred->>'CuentaHuellaHMAC','base64'))<>32
    OR octet_length(decode(v_acred->>'SujetoHuellaHMAC','base64'))<>32
    OR decode(v_acred->>'CuentaHuellaHMAC','base64')=decode(v_acred->>'SujetoHuellaHMAC','base64')
    OR v_acred->>'ProcedenciaRef' !~ '^prc_[A-Za-z0-9_-]{22,128}$'
    OR v_acred->>'ProcedenciaSHA256' !~ '^[0-9a-f]{64}$'
    OR v_acred->>'ProcedenciaAutoridad'<>'autoridad_maestra_acreditada'
    OR v_desde>v_ahora OR v_hasta<=v_ahora
    OR v_equiv->>'SujetoRef' IS DISTINCT FROM v_sujeto
    OR v_equiv->>'PruebaRef' IS DISTINCT FROM v_acred->>'EquivalenciaRef'
    OR v_equiv->>'PruebaSHA256' !~ '^[0-9a-f]{64}$'
    OR (v_equiv->>'PruebaVersion')::numeric<1
    OR (v_nueva AND v_equiv->>'PersonaRef'<>'')
    OR (NOT v_nueva AND v_equiv->>'PersonaRef' !~ '^per_[A-Za-z0-9_-]{22,128}$')
    OR v_recurso#>>'{ambitos,sujeto_ref}' IS DISTINCT FROM v_sujeto
    OR v_recurso#>>'{atributos,entrada_sha256}' IS DISTINCT FROM v_huella
    OR v_recurso#>>'{atributos,equivalencia_ref}' IS DISTINCT FROM v_equiv->>'PruebaRef'
    OR encode(sha256(p_recurso),'hex') IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256'
    OR v_decision->>'principal_id' IS DISTINCT FROM p_actor
    OR v_decision->>'accion' IS DISTINCT FROM 'vec.registro_propio.crear'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'vec'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'registro_propio'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'alta_vec_propia'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM v_operacion
 THEN RAISE EXCEPTION 'registro propio: autoridad divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS DISTINCT FROM true OR v_consumo.efecto_ref IS DISTINCT FROM v_operacion
 THEN RAISE EXCEPTION 'registro propio: consumo denegado' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:registro-propio:sujeto:'||v_sujeto,0));
 SELECT * INTO v_prev FROM vec_identidad_sesiones_v1.registro_propio_v1 WHERE operacion_ref=v_operacion;
 IF FOUND THEN
  IF v_prev.entrada_sha256<>v_huella OR v_prev.actor_ref<>p_actor
  THEN RAISE EXCEPTION 'registro propio: clave divergente' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('OperacionRef',v_prev.operacion_ref,'ReciboRef',v_prev.recibo_ref,'Estado',v_prev.estado,
   'CuentaRef',v_prev.cuenta_ref,'CuentaVersion',v_prev.cuenta_version,'PersonaRef',v_prev.persona_ref,'PersonaVersion',v_prev.persona_version,
   'PerfilRef',v_prev.perfil_ref,'PerfilVersion',v_prev.perfil_version,'VinculoRef',v_prev.vinculo_ref,'VinculoVersion',v_prev.vinculo_version,
   'ProcedenciaRef',v_prev.procedencia_ref,'ProcedenciaVersion',v_prev.procedencia_version,'ProcedenciaSHA256',v_prev.procedencia_sha256);
 END IF;
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1 WHERE sujeto_ref=v_sujeto)
 THEN RAISE EXCEPTION 'registro propio: sujeto ya registrado con otra clave' USING ERRCODE='23505'; END IF;
 SELECT cuenta_ref INTO STRICT v_cuenta FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
  v_operacion,'vec.identidad.hmac-sha256.v1',v_acred->>'DominioHMACRef',v_acred->>'ClaveHMACID',
  (v_acred->>'ClaveHMACVersion')::bigint,decode(v_acred->>'CuentaHuellaHMAC','base64'),
  decode(v_acred->>'SujetoHuellaHMAC','base64'),false,NULL);
 v_persona:=CASE WHEN v_nueva THEN 'per_'||encode(public.gen_random_bytes(18),'hex') ELSE v_equiv->>'PersonaRef' END;
 v_perfil:='prf_'||encode(public.gen_random_bytes(18),'hex');
 v_vinculo:='vca_'||encode(public.gen_random_bytes(18),'hex');
 v_recibo:='rpr_'||encode(public.gen_random_bytes(18),'hex');
 IF v_persona=p_actor THEN RAISE EXCEPTION 'registro propio: actor y solicitante coinciden' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_version FROM vec_contexto_actor_v1.registrar_persona_registro_propio_v1(
  v_cuenta,v_persona,v_perfil,v_vinculo,v_nueva,v_acred->>'ProcedenciaRef',
  (v_acred->>'ProcedenciaVersion')::numeric,v_acred->>'ProcedenciaSHA256',
  v_acred->>'ProcedenciaAutoridad',v_desde,v_hasta);
 INSERT INTO vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1(sujeto_ref,persona_ref,prueba_ref,prueba_version,prueba_sha256,operacion_ref)
  VALUES(v_sujeto,v_persona,v_equiv->>'PruebaRef',(v_equiv->>'PruebaVersion')::numeric,v_equiv->>'PruebaSHA256',v_operacion);
 INSERT INTO vec_identidad_sesiones_v1.registro_propio_v1(operacion_ref,recibo_ref,entrada_sha256,actor_ref,sujeto_ref,cuenta_ref,cuenta_version,persona_ref,persona_version,perfil_ref,perfil_version,vinculo_ref,vinculo_version,procedencia_ref,procedencia_version,procedencia_sha256,decision_ref,estado,registrada_en)
  VALUES(v_operacion,v_recibo,v_huella,p_actor,v_sujeto,v_cuenta,v_version.cuenta_version,v_persona,v_version.persona_version,v_perfil,v_version.perfil_version,v_vinculo,v_version.vinculo_version,v_acred->>'ProcedenciaRef',(v_acred->>'ProcedenciaVersion')::numeric,v_acred->>'ProcedenciaSHA256',v_consumo.decision_ref,'pendiente_contacto',v_ahora);
 INSERT INTO vec_identidad_sesiones_v1.registro_propio_outbox_v1(operacion_ref,evento_ref,tipo,creada_en)
  VALUES(v_operacion,'evt_'||encode(public.gen_random_bytes(18),'hex'),'registro_pendiente_contacto',v_ahora);
 RETURN jsonb_build_object('OperacionRef',v_operacion,'ReciboRef',v_recibo,'Estado','pendiente_contacto',
  'CuentaRef',v_cuenta,'CuentaVersion',v_version.cuenta_version,'PersonaRef',v_persona,'PersonaVersion',v_version.persona_version,
  'PerfilRef',v_perfil,'PerfilVersion',v_version.perfil_version,'VinculoRef',v_vinculo,'VinculoVersion',v_version.vinculo_version,
  'ProcedenciaRef',v_acred->>'ProcedenciaRef','ProcedenciaVersion',(v_acred->>'ProcedenciaVersion')::numeric,
  'ProcedenciaSHA256',v_acred->>'ProcedenciaSHA256');
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.registrar_propio_v1(bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.registrar_propio_v1(bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_identidad_sesiones_v1_provisionador;
COMMIT;
