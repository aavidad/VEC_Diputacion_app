\set ON_ERROR_STOP on
-- CT229: anexo durable de los selectores de firma histórica en el detalle
-- ya autorizado. El recibo base CT43 y la ficha legacy permanecen intactos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000229',0));
DO $pre$
DECLARE base oid:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 bolsa oid:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 aut oid:=to_regprocedure('vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(text,text,text,text,text,text,text)');
 original oid:=to_regprocedure('vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_is_in_recovery() OR base IS NULL OR original IS NULL OR bolsa IS NULL OR aut IS NULL
  OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)') IS NOT NULL
  OR to_regclass('vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1') IS NOT NULL
  OR to_regclass('vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1') IS NOT NULL
  OR to_regprocedure('vec_contratacion_temporal.confirmar_vigencia_selectores_rrhh_v1()') IS NOT NULL
  OR to_regclass('vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2') IS NULL
  OR to_regclass('vec_contratacion_temporal.firma_documento_revision_pdf_v2') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=base
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='c5fb42d90fa4aab807a583d6d797db7bb42f50f6cd2a603bae2c6ed0c1ef06b8')
  OR to_regprocedure('vec_contratacion_temporal.consultar_original_propuesta_rrhh_con_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=original
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='9ef32619a08eb16bb2a5129cf0478fd2a60354fd49319a778c363a92c1d4b985')
  OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=bolsa
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d8a56e04b0239a2c92ff13de105cf8e7d6eefc572c1719fbfbd1336c1bdc38ea')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=aut
   AND p.proowner=to_regrole('vec_autorizacion_propietario') AND p.prosecdef
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario',
    'vec_contratacion_temporal_propietario=X/vec_autorizacion_propietario']::aclitem[])
 THEN RAISE EXCEPTION 'CT229: preimagen o dependencias divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1(
 acceso_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(acceso_ref)
  ON UPDATE RESTRICT ON DELETE RESTRICT,
 sello_detalle_sha256 text NOT NULL REFERENCES vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(recibo_sello_sha256)
  ON UPDATE RESTRICT ON DELETE RESTRICT,
 decision_ref text NOT NULL,
 version_rol_ref text NOT NULL,
 asignacion_ref text,
 asignacion_huella_sha256 text,
 perfil_aux_ref text,
 version_rol_aux_ref text,
 unidad_ref text,
 selectores_canon bytea NOT NULL,
 selectores_huella_sha256 text NOT NULL,
 recibo_canon bytea NOT NULL,
 sello_selectores_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 CONSTRAINT prueba_selectores_limites CHECK(
  sello_detalle_sha256 ~ '^[0-9a-f]{64}$'
  AND selectores_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND sello_selectores_sha256 ~ '^[0-9a-f]{64}$'
  AND octet_length(selectores_canon) BETWEEN 2 AND 131072
  AND octet_length(recibo_canon) BETWEEN 32 AND 262144
  AND selectores_huella_sha256=encode(sha256(selectores_canon),'hex')
  AND sello_selectores_sha256=encode(sha256(recibo_canon),'hex')
  AND ((asignacion_ref IS NULL AND asignacion_huella_sha256 IS NULL AND perfil_aux_ref IS NULL
    AND version_rol_aux_ref IS NULL AND unidad_ref IS NULL AND selectores_canon=convert_to('[]','UTF8'))
   OR (asignacion_ref IS NOT NULL AND asignacion_huella_sha256 ~ '^[0-9a-f]{64}$'
    AND perfil_aux_ref IS NOT NULL AND version_rol_aux_ref IS NOT NULL AND unidad_ref IS NOT NULL))
 )
);
CREATE TRIGGER prueba_selectores_recuperacion_inmutable BEFORE UPDATE OR DELETE
 ON vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER prueba_selectores_recuperacion_no_truncar BEFORE TRUNCATE
 ON vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1
 FOR ALL TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 FROM PUBLIC;

-- El cierre se comprueba al COMMIT, después de la lectura de Bolsa o de
-- cualquier demora del cliente. Una fila por acceso CT43 y ninguna concesión
-- directa a LOGIN; el original reducido usa modo original sin anexo.
CREATE TABLE vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1(
 acceso_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(acceso_ref)
  ON UPDATE RESTRICT ON DELETE RESTRICT,
 sello_detalle_sha256 text NOT NULL REFERENCES vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2(recibo_sello_sha256)
  ON UPDATE RESTRICT ON DELETE RESTRICT,
 decision_ref text NOT NULL,decision_huella_sha256 text NOT NULL,
 principal_ref text NOT NULL,organizacion_ref text NOT NULL,
 valida_hasta timestamptz(6) NOT NULL,modo text NOT NULL CHECK(modo IN('selectores','original')),
 transaccion_origen xid8 NOT NULL,sesion_origen text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 CHECK(decision_huella_sha256 ~ '^[0-9a-f]{64}$' AND sello_detalle_sha256 ~ '^[0-9a-f]{64}$')
);
CREATE TRIGGER cierre_vigencia_selectores_inmutable BEFORE UPDATE OR DELETE
 ON vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER cierre_vigencia_selectores_no_truncar BEFORE TRUNCATE
 ON vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total ON vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1
 FOR ALL TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1 FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.confirmar_vigencia_selectores_rrhh_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' AS $cierre$
DECLARE b record;a record;amb jsonb;
BEGIN
 IF TG_OP<>'INSERT' OR current_user<>'vec_contratacion_temporal_propietario'
  OR session_user=current_user OR NEW.sesion_origen IS DISTINCT FROM session_user
  OR NEW.transaccion_origen IS DISTINCT FROM pg_current_xact_id()
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CT229: cierre fuera de la transacción original' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT b FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 x
  WHERE x.acceso_ref=NEW.acceso_ref FOR SHARE;
 IF b.tipo_consulta IS DISTINCT FROM 'detalle' OR b.recibo_sello_sha256 IS DISTINCT FROM NEW.sello_detalle_sha256
  OR b.decision_ref IS DISTINCT FROM NEW.decision_ref
  OR b.decision_huella_sha256 IS DISTINCT FROM NEW.decision_huella_sha256
  OR b.actor_ref IS DISTINCT FROM NEW.principal_ref
  OR b.organizacion_ref IS DISTINCT FROM NEW.organizacion_ref THEN
  RAISE EXCEPTION 'CT229: cierre desligado del recibo CT43' USING ERRCODE='42501'; END IF;
 IF NEW.modo='selectores' THEN
  SELECT * INTO STRICT a FROM vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 x
   WHERE x.acceso_ref=NEW.acceso_ref FOR SHARE;
  IF a.sello_detalle_sha256 IS DISTINCT FROM b.recibo_sello_sha256
   OR a.decision_ref IS DISTINCT FROM b.decision_ref THEN
   RAISE EXCEPTION 'CT229: anexo desligado' USING ERRCODE='42501'; END IF;
  IF a.asignacion_ref IS NOT NULL THEN
   amb:=vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(
    b.actor_ref,b.organizacion_ref,a.unidad_ref,a.perfil_aux_ref,a.version_rol_aux_ref,
    a.asignacion_ref,a.asignacion_huella_sha256);
   IF amb->>'principal_ref' IS DISTINCT FROM b.actor_ref
    OR amb->>'organizacion_ref' IS DISTINCT FROM b.organizacion_ref
    OR amb->>'unidad_ref' IS DISTINCT FROM a.unidad_ref
    OR amb->>'asignacion_ref' IS DISTINCT FROM a.asignacion_ref
    OR amb->>'asignacion_huella_sha256' IS DISTINCT FROM a.asignacion_huella_sha256 THEN
    RAISE EXCEPTION 'CT229: ASG retirada al confirmar' USING ERRCODE='42501'; END IF;
  ELSIF a.selectores_canon IS DISTINCT FROM convert_to('[]','UTF8') THEN
   RAISE EXCEPTION 'CT229: anexo sin ASG no vacío' USING ERRCODE='42501'; END IF;
 ELSIF EXISTS(SELECT 1 FROM vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 x
   WHERE x.acceso_ref=NEW.acceso_ref) THEN
  RAISE EXCEPTION 'CT229: original reducido con anexo' USING ERRCODE='42501'; END IF;
 IF clock_timestamp()>=NEW.valida_hasta THEN
  RAISE EXCEPTION 'CT229: decisión caducada al confirmar' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $cierre$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.confirmar_vigencia_selectores_rrhh_v1() FROM PUBLIC;
CREATE CONSTRAINT TRIGGER cierre_vigencia_selectores_commit
 AFTER INSERT ON vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION vec_contratacion_temporal.confirmar_vigencia_selectores_rrhh_v1();

CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea, p_filtro jsonb)
 RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, selectores_canon bytea, selectores_huella_sha256 text, sello_selectores_sha256 text, recibo_selectores_canon bytea)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_decision_huella text;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_contenido bytea;
    v_filtro jsonb; v_ambito jsonb; v_ambito_final jsonb;
    v_selectores jsonb; v_canon_selectores bytea; v_huella_selectores text;
    v_canon_anexo bytea; v_sello_anexo text; v_numero integer;
BEGIN
    SELECT *
      INTO v_login
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = SESSION_USER;
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL
       OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit
       OR v_login.rolsuper
       OR v_login.rolcreatedb
       OR v_login.rolcreaterole
       OR v_login.rolreplication
       OR v_login.rolbypassrls
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = v_login.oid
       ) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
             JOIN pg_catalog.pg_roles grupo
               ON grupo.oid = membresia.roleid
            WHERE membresia.member = v_login.oid
              AND grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT membresia.admin_option
              AND membresia.inherit_option
              AND NOT membresia.set_option
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.roleid = v_login.oid
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles grupo
            WHERE grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT grupo.rolcanlogin
              AND grupo.rolinherit
              AND NOT grupo.rolsuper
              AND NOT grupo.rolcreatedb
              AND NOT grupo.rolcreaterole
              AND NOT grupo.rolreplication
              AND NOT grupo.rolbypassrls
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member =
                  'vec_contratacion_temporal_consultor_rrhh'
                      ::pg_catalog.regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval >
          interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval >
          interval '4 seconds'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       ) = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       )::interval > interval '6 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Los límites O(1) preceden a los cánones CT40 y, por tanto, a cualquier
    -- expresión regular sobre varlena exterior.
    IF p_alcance IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.organizacion_ref, '')
       ) > 160
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.clase_ambito, '')
       ) > 16
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.ambito_ref, '')
       ) > 160
       OR p_consulta IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.expediente_ref, '')
       ) > 160 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Limita antes de decodificar o analizar cualquier pieza controlada por
    -- el llamador. Repite exactamente la frontera privada CT44.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(
           p_capacidad_canonica
       ) NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(
           p_decision_canonica
       ) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(
           p_motivo_canonico
       ) NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(
           p_contexto_actor_canonico
       ) NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(
           p_payload_vec_ad_3
       ) NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(
           p_sobre_cose_sign_1
       ) NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(
           p_evidencia_verificacion
       ) NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            p_consulta
        );
    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.convert_from(
        p_decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_capacidad ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Solo la versión v2 gobernada de los cuatro perfiles consulta CT puede
    -- pedir el complemento. La decisión sigue siendo de detalle ordinario.
    IF v_decision ->> 'garantia_minima' IS DISTINCT FROM 'sustancial'
       OR v_decision ->> 'version_rol_ref' NOT IN (
           'rol:ct_cargo_tecnico_consulta_dev:v2',
           'rol:ct_cargo_jefatura_consulta_dev:v2',
           'rol:ct_cargo_diputacion_consulta_dev:v2',
           'rol:ct_cargo_secretaria_consulta_dev:v2')
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM
          '["selectores_recuperacion_firmas"]'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version,
        p_payload_vec_ad_3, p_sobre_cose_sign_1,
        p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal
       .material_autorizacion_consulta_rrhh_v3;
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
            p_alcance, p_consulta, v_material
        );
    v_cierre := v_resultado.cierre;
    v_detalle := v_resultado.detalle;
    v_resumen := v_detalle.resumen;
    IF v_resultado IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
       OR v_cierre IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
       OR v_detalle IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal
              .entrada_detalle_expediente_rrhh_v1
       OR v_resumen IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resumen_publicacion_rrhh_v1
       OR v_resultado.generada_en IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;
    v_contenido :=
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resultado.generada_en, v_detalle
        );
    IF v_cierre.generada_en IS DISTINCT FROM
           v_resultado.generada_en
       OR v_cierre.total IS DISTINCT FROM 1::smallint
       OR v_cierre.cursor_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.alcance_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.expediente_ref IS DISTINCT FROM
          v_resumen.expediente_ref
       OR v_cierre.version_expediente IS DISTINCT FROM
          v_resumen.version
       OR v_cierre.contenido_huella_sha256 IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(v_contenido), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- La autorización V3, su auditoría y el recibo CT43 ya quedaron dentro de
    -- motor_consultar_detalle_rrhh_v1 en esta misma transacción.
    v_filtro:=COALESCE(p_filtro,'null'::jsonb);
    IF v_filtro IS DISTINCT FROM 'null'::jsonb THEN
      IF pg_catalog.jsonb_typeof(v_filtro) IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'CT229: filtro privado inválido' USING ERRCODE='42501'; END IF;
      IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(v_filtro))<>5
        OR NOT (v_filtro ?& ARRAY['unidad_ref','perfil_aux_ref','version_rol_aux_ref',
           'asignacion_ref','asignacion_huella_sha256'])
        OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_object_keys(v_filtro) AS claves(nombre)
          WHERE pg_catalog.jsonb_typeof(v_filtro->claves.nombre) IS DISTINCT FROM 'string') THEN
        RAISE EXCEPTION 'CT229: filtro privado inválido' USING ERRCODE='42501';
      END IF;
    END IF;
    IF v_filtro IS DISTINCT FROM 'null'::jsonb THEN
        v_ambito:=vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(
          v_decision->>'principal_id',p_alcance.organizacion_ref,v_filtro->>'unidad_ref',
          v_filtro->>'perfil_aux_ref',v_filtro->>'version_rol_aux_ref',
          v_filtro->>'asignacion_ref',v_filtro->>'asignacion_huella_sha256');
        IF v_ambito->>'principal_ref' IS DISTINCT FROM v_decision->>'principal_id'
         OR v_ambito->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
         OR v_ambito->>'unidad_ref' IS DISTINCT FROM v_filtro->>'unidad_ref'
         OR v_ambito->>'asignacion_ref' IS DISTINCT FROM v_filtro->>'asignacion_ref'
         OR v_ambito->>'asignacion_huella_sha256' IS DISTINCT FROM v_filtro->>'asignacion_huella_sha256'
         OR v_ambito->>'perfil_ref' IS DISTINCT FROM v_filtro->>'perfil_aux_ref'
         OR v_ambito->>'version_rol_ref' IS DISTINCT FROM v_filtro->>'version_rol_aux_ref' THEN
          RAISE EXCEPTION 'CT229: ámbito propio divergente' USING ERRCODE='42501'; END IF;
        SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
          'firma_ref',x.firma_ref,'selector',jsonb_build_object(
           'expediente_ref',x.expediente_ref,'version_expediente',x.expediente_version,
           'documento',x.documento,'paso_orden',x.paso_orden,
           'clave_idempotencia',x.clave_idempotencia,'catalogo_huella',x.catalogo_huella_sha256,
           'via',x.via_registro)) ORDER BY x.secuencia),'[]'::jsonb)
          INTO v_numero,v_selectores
        FROM (SELECT f.firma_ref,f.expediente_ref,f.expediente_version,f.documento,
             f.paso_orden,f.clave_idempotencia,f.catalogo_huella_sha256,f.via_registro,f.secuencia
          FROM vec_contratacion_temporal.firma_documento_v1 f
          JOIN vec_contratacion_temporal.firma_documento_revision_pdf_v2 r ON r.firma_ref=f.firma_ref
          WHERE f.organizacion_ref=p_alcance.organizacion_ref
           AND f.expediente_ref=v_cierre.expediente_ref
           AND f.expediente_version<=v_cierre.version_expediente
           AND f.via_registro='certificado_vec' AND f.resultado='firmado'
           AND f.unidad_firmante_ref=v_filtro->>'unidad_ref'
           AND r.competencia_evidencia_ref IS NOT NULL
           AND r.competencia_evidencia_huella_sha256 IS NOT NULL
           AND r.competencia_consumo_decision_ref=f.decision_ref
           AND r.competencia_consumo_huella_sha256=f.consumo_huella_sha256
           AND r.competencia_auditoria_consumo_ref=f.auditoria_consumo_ref
          ORDER BY f.secuencia LIMIT 129) x;
        IF v_numero>128 THEN RAISE EXCEPTION 'CT229: selectores no representables' USING ERRCODE='P1525'; END IF;
        -- Repite AUT102 antes de cerrar el anexo; su helper bloquea el puntero
        -- y la versión actual hasta COMMIT.
        v_ambito_final:=vec_autorizacion.comprobar_ambito_propio_selectores_firma_ct_v1(
          v_decision->>'principal_id',p_alcance.organizacion_ref,v_filtro->>'unidad_ref',
          v_filtro->>'perfil_aux_ref',v_filtro->>'version_rol_aux_ref',
          v_filtro->>'asignacion_ref',v_filtro->>'asignacion_huella_sha256');
        IF v_ambito_final IS DISTINCT FROM v_ambito THEN
          RAISE EXCEPTION 'CT229: ámbito cambiado durante lectura' USING ERRCODE='42501'; END IF;
    ELSE
        v_selectores:='[]'::jsonb;
    END IF;
    v_canon_selectores:=convert_to(v_selectores::text,'UTF8');
    IF octet_length(v_canon_selectores)>131072 THEN
      RAISE EXCEPTION 'CT229: selectores no representables' USING ERRCODE='P1525'; END IF;
    v_huella_selectores:=encode(sha256(v_canon_selectores),'hex');
    -- El recibo complementario liga sin ambigüedad el acceso y sello CT43,
    -- la decisión vigente, la ASG restrictiva y los bytes JSON entregados.
    v_canon_anexo:=convert_to(jsonb_build_object(
      'esquema','vec.ct.detalle.selectores-recuperacion.v1',
      'acceso_ref',v_cierre.acceso_ref,'sello_detalle_sha256',v_cierre.recibo_sello_sha256,
      'decision_ref',v_decision->>'decision_ref','version_rol_ref',v_decision->>'version_rol_ref',
      'asignacion_ref',v_filtro->>'asignacion_ref',
      'asignacion_huella_sha256',v_filtro->>'asignacion_huella_sha256',
      'perfil_aux_ref',v_filtro->>'perfil_aux_ref',
      'version_rol_aux_ref',v_filtro->>'version_rol_aux_ref',
      'unidad_ref',v_filtro->>'unidad_ref',
      'selectores_huella_sha256',v_huella_selectores,'selectores',v_selectores)::text,'UTF8');
    v_sello_anexo:=encode(sha256(v_canon_anexo),'hex');
    INSERT INTO vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1(
      acceso_ref,sello_detalle_sha256,decision_ref,version_rol_ref,
      asignacion_ref,asignacion_huella_sha256,perfil_aux_ref,version_rol_aux_ref,
      unidad_ref,selectores_canon,selectores_huella_sha256,recibo_canon,sello_selectores_sha256,
      registrada_en)
    VALUES(v_cierre.acceso_ref,v_cierre.recibo_sello_sha256,v_decision->>'decision_ref',
      v_decision->>'version_rol_ref',v_filtro->>'asignacion_ref',
      v_filtro->>'asignacion_huella_sha256',v_filtro->>'perfil_aux_ref',
      v_filtro->>'version_rol_aux_ref',v_filtro->>'unidad_ref',v_canon_selectores,
      v_huella_selectores,v_canon_anexo,v_sello_anexo,clock_timestamp());
    IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1 x
      WHERE x.acceso_ref=v_cierre.acceso_ref AND x.sello_detalle_sha256=v_cierre.recibo_sello_sha256
       AND x.sello_selectores_sha256=v_sello_anexo AND x.selectores_canon=v_canon_selectores) THEN
      RAISE EXCEPTION 'CT229: anexo no confirmado' USING ERRCODE='55000'; END IF;
    IF v_decision->>'valida_hasta' IS NULL OR
      (v_decision->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
      RAISE EXCEPTION 'CT229: decisión caducada' USING ERRCODE='42501'; END IF;
    INSERT INTO vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1(
      acceso_ref,sello_detalle_sha256,decision_ref,decision_huella_sha256,
      principal_ref,organizacion_ref,valida_hasta,modo,transaccion_origen,sesion_origen,registrada_en)
    VALUES(v_cierre.acceso_ref,v_cierre.recibo_sello_sha256,v_decision->>'decision_ref',
      v_decision_huella,v_decision->>'principal_id',p_alcance.organizacion_ref,
      (v_decision->>'valida_hasta')::timestamptz,'selectores',pg_current_xact_id(),session_user,clock_timestamp());

    RETURN QUERY SELECT
        v_contenido, v_cierre.esquema, v_cierre.acceso_ref,
        v_cierre.secuencia, v_cierre.anterior_sha256,
        v_cierre.huella_sha256,
        v_cierre.vinculo_identidad_huella_sha256,
        v_cierre.alcance_huella_sha256, v_cierre.registrada_en,
        v_cierre.auditoria_vec_ref,
        v_cierre.auditoria_vec_huella_sha256,
        v_cierre.consumo_vec_huella_sha256,
        v_cierre.contenido_huella_sha256,
        v_cierre.resultado_huella_sha256,
        v_cierre.cursor_huella_sha256, v_cierre.generada_en,
        v_cierre.expediente_ref, v_cierre.version_expediente,
        v_cierre.total, v_cierre.recibo_sello_sha256,
        v_canon_selectores,v_huella_selectores,v_sello_anexo,v_canon_anexo;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$function$;
CREATE FUNCTION vec_contratacion_temporal.consultar_original_propuesta_rrhh_con_selectores_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_decision_huella text;
    v_material
        vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_resultado
        vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1;
    v_cierre
        vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2;
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_contenido bytea;
BEGIN
    SELECT *
      INTO v_login
      FROM pg_catalog.pg_roles rol
     WHERE rol.rolname = SESSION_USER;
    IF CURRENT_USER <>
           'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL
       OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit
       OR v_login.rolsuper
       OR v_login.rolcreatedb
       OR v_login.rolcreaterole
       OR v_login.rolreplication
       OR v_login.rolbypassrls
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = v_login.oid
       ) <> 1
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
             JOIN pg_catalog.pg_roles grupo
               ON grupo.oid = membresia.roleid
            WHERE membresia.member = v_login.oid
              AND grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT membresia.admin_option
              AND membresia.inherit_option
              AND NOT membresia.set_option
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.roleid = v_login.oid
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles grupo
            WHERE grupo.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'
              AND NOT grupo.rolcanlogin
              AND grupo.rolinherit
              AND NOT grupo.rolsuper
              AND NOT grupo.rolcreatedb
              AND NOT grupo.rolcreaterole
              AND NOT grupo.rolreplication
              AND NOT grupo.rolbypassrls
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member =
                  'vec_contratacion_temporal_consultor_rrhh'
                      ::pg_catalog.regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation')
          <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval >
          interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval >
          interval '4 seconds'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       ) = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout'
       )::interval > interval '6 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Los límites O(1) preceden a los cánones CT40 y, por tanto, a cualquier
    -- expresión regular sobre varlena exterior.
    IF p_alcance IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.organizacion_ref, '')
       ) > 160
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.clase_ambito, '')
       ) > 16
       OR pg_catalog.octet_length(
           COALESCE(p_alcance.ambito_ref, '')
       ) > 160
       OR p_consulta IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.consulta_detalle_rrhh_v1
       OR pg_catalog.octet_length(
           COALESCE(p_consulta.expediente_ref, '')
       ) > 160 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- Limita antes de decodificar o analizar cualquier pieza controlada por
    -- el llamador. Repite exactamente la frontera privada CT44.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(
           p_capacidad_canonica
       ) NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(
           p_decision_canonica
       ) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(
           p_motivo_canonico
       ) NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(
           p_contexto_actor_canonico
       ) NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(
           p_payload_vec_ad_3
       ) NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(
           p_sobre_cose_sign_1
       ) NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(
           p_evidencia_verificacion
       ) NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            p_consulta
        );
    v_capacidad := pg_catalog.convert_from(
        p_capacidad_canonica, 'UTF8'
    )::jsonb;
    v_decision := pg_catalog.convert_from(
        p_decision_canonica, 'UTF8'
    )::jsonb;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(
            pg_catalog.sha256(v_consulta_canonica), 'hex'
        ) || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    v_decision_huella := pg_catalog.encode(
        pg_catalog.sha256(p_decision_canonica), 'hex'
    );
    IF v_capacidad ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
          'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          v_decision_huella
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM
          'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256'
          IS DISTINCT FROM v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    -- La capacidad v2 permite el campo agregado, pero esta fachada histórica
    -- devuelve solo el original v7 que necesita el borrador: no lee selectores.
    IF v_decision ->> 'garantia_minima' IS DISTINCT FROM 'sustancial'
       OR v_decision ->> 'version_rol_ref' NOT IN (
           'rol:ct_cargo_tecnico_consulta_dev:v2',
           'rol:ct_cargo_jefatura_consulta_dev:v2',
           'rol:ct_cargo_diputacion_consulta_dev:v2',
           'rol:ct_cargo_secretaria_consulta_dev:v2')
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM
          '["selectores_recuperacion_firmas"]'::jsonb THEN
        RAISE EXCEPTION 'CT229: original con concesión ajena' USING ERRCODE='42501';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version,
        p_payload_vec_ad_3, p_sobre_cose_sign_1,
        p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal
       .material_autorizacion_consulta_rrhh_v3;
    v_resultado :=
        vec_contratacion_temporal.motor_consultar_original_propuesta_rrhh_v1(
            p_alcance, p_consulta, v_material
        );
    v_cierre := v_resultado.cierre;
    v_detalle := v_resultado.detalle;
    v_resumen := v_detalle.resumen;
    IF v_resultado IS NOT DISTINCT FROM
           NULL::vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
       OR v_cierre IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
       OR v_detalle IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal
              .entrada_detalle_expediente_rrhh_v1
       OR v_resumen IS NOT DISTINCT FROM
          NULL::vec_contratacion_temporal.resumen_publicacion_rrhh_v1
       OR v_resultado.generada_en IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;
    v_contenido :=
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resultado.generada_en, v_detalle
        );
    IF v_cierre.generada_en IS DISTINCT FROM
           v_resultado.generada_en
       OR v_cierre.total IS DISTINCT FROM 1::smallint
       OR v_cierre.cursor_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.alcance_huella_sha256 IS DISTINCT FROM ''
       OR v_cierre.expediente_ref IS DISTINCT FROM
          v_resumen.expediente_ref
       OR v_cierre.version_expediente IS DISTINCT FROM
          v_resumen.version
       OR v_cierre.contenido_huella_sha256 IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(v_contenido), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
    END IF;

    IF v_decision->>'valida_hasta' IS NULL OR
      (v_decision->>'valida_hasta')::timestamptz<=clock_timestamp() THEN
      RAISE EXCEPTION 'CT229: original caducado' USING ERRCODE='42501'; END IF;
    INSERT INTO vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1(
      acceso_ref,sello_detalle_sha256,decision_ref,decision_huella_sha256,
      principal_ref,organizacion_ref,valida_hasta,modo,transaccion_origen,sesion_origen,registrada_en)
    VALUES(v_cierre.acceso_ref,v_cierre.recibo_sello_sha256,v_decision->>'decision_ref',
      v_decision_huella,v_decision->>'principal_id',p_alcance.organizacion_ref,
      (v_decision->>'valida_hasta')::timestamptz,'original',pg_current_xact_id(),session_user,clock_timestamp());

    RETURN QUERY SELECT
        v_contenido, v_cierre.esquema, v_cierre.acceso_ref,
        v_cierre.secuencia, v_cierre.anterior_sha256,
        v_cierre.huella_sha256,
        v_cierre.vinculo_identidad_huella_sha256,
        v_cierre.alcance_huella_sha256, v_cierre.registrada_en,
        v_cierre.auditoria_vec_ref,
        v_cierre.auditoria_vec_huella_sha256,
        v_cierre.consumo_vec_huella_sha256,
        v_cierre.contenido_huella_sha256,
        v_cierre.resultado_huella_sha256,
        v_cierre.cursor_huella_sha256, v_cierre.generada_en,
        v_cierre.expediente_ref, v_cierre.version_expediente,
        v_cierre.total, v_cierre.recibo_sello_sha256;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$function$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_original_propuesta_rrhh_con_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_original_propuesta_rrhh_con_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)
 TO vec_contratacion_temporal_consultor_rrhh;
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_cursor text,p_filtro jsonb) RETURNS TABLE(contenido_canonico bytea, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, resultado_bolsa jsonb,selectores_canon bytea,selectores_huella_sha256 text,sello_selectores_sha256 text, recibo_selectores_canon bytea)
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '1s'
    SET statement_timeout TO '4s'
    SET idle_in_transaction_session_timeout TO '6s'
    AS $_$
DECLARE d record; actual record; vinculos jsonb; sugeridas jsonb; n integer;
 entradas jsonb; todos jsonb; lote jsonb; pagina_n integer; antes_en timestamptz; antes_ref text;
 ultimo_en timestamptz; ultimo_ref text; mas boolean:=false; siguiente text; bolsa_vigente text;
 personas_text text; personas numeric; aceptaciones integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_consultor_rrhh','MEMBER') IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR (p_cursor IS NOT NULL AND p_cursor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z#llamamiento:[0-9a-f]{64}$') THEN
  RAISE EXCEPTION 'CT201: lectura de resultado denegada' USING ERRCODE='42501';
 END IF;
 IF p_cursor IS NOT NULL THEN
  antes_en:=split_part(p_cursor,'#',1)::timestamptz;
  antes_ref:=split_part(p_cursor,'#',2);
 END IF;
 SELECT * INTO STRICT d FROM vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(
  p_alcance,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,p_filtro);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
  'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)),
  '[]'::jsonb) INTO todos FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
  WHERE x.expediente_ref=d.expediente_ref;
 n:=jsonb_array_length(todos);
 aceptaciones:=vec_bolsa_llamamientos.contar_aceptaciones_firmes_ct_v1(todos);
 SELECT coalesce(jsonb_agg(jsonb_build_object('bolsa_ref',x.bolsa_ref,
   'llamamiento_ref',x.llamamiento_ref,'recibo_emision_ref',x.recibo_emision_ref)
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO entradas
 FROM (SELECT v.bolsa_ref,v.llamamiento_ref,v.recibo_emision_ref,v.vinculado_en
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC LIMIT 20) x;
 pagina_n:=jsonb_array_length(entradas);
 IF pagina_n=20 THEN
  SELECT v.vinculado_en,v.llamamiento_ref INTO ultimo_en,ultimo_ref
   FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
   WHERE v.expediente_ref=d.expediente_ref
     AND (p_cursor IS NULL OR (v.vinculado_en,v.llamamiento_ref)<(antes_en,antes_ref))
   ORDER BY v.vinculado_en DESC,v.llamamiento_ref DESC OFFSET 19 LIMIT 1;
  SELECT EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
    WHERE v.expediente_ref=d.expediente_ref
      AND (v.vinculado_en,v.llamamiento_ref)<(ultimo_en,ultimo_ref)) INTO mas;
  IF mas THEN
   siguiente:=to_char(ultimo_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')||'#'||ultimo_ref;
  END IF;
 END IF;
 lote:=vec_bolsa_llamamientos.leer_resultados_emisiones_ct_v1(entradas);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa_ref',x.bolsa_ref,'llamamiento_ref',x.llamamiento_ref,
   'recibo_emision_ref',x.recibo_emision_ref,
   'recibo_vinculo_ref',x.recibo_ref,'vinculado_en',x.vinculado_en,
   'emitido_en',b.valor->'emitido_en',
   'participaciones',b.valor->'participaciones')
   ORDER BY x.vinculado_en DESC,x.llamamiento_ref DESC),'[]'::jsonb) INTO vinculos
 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
 JOIN LATERAL jsonb_array_elements(lote) b(valor)
   ON b.valor->>'llamamiento_ref'=x.llamamiento_ref
  AND b.valor->>'bolsa_ref'=x.bolsa_ref
  AND b.valor->>'recibo_emision_ref'=x.recibo_emision_ref
 WHERE x.expediente_ref=d.expediente_ref;
 IF jsonb_array_length(vinculos)<>pagina_n THEN
  RAISE EXCEPTION 'CT201: resultado Bolsa no recuperable' USING ERRCODE='55000';
 END IF;
 SELECT e.numero_visible,a.version,v.agregado_json INTO actual
 FROM vec_contratacion_temporal.expediente_alta e
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=e.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE e.expediente_ref=d.expediente_ref;
 IF NOT FOUND OR actual.version IS DISTINCT FROM d.version_expediente THEN
  RAISE EXCEPTION 'CT201: agregado de lectura divergente' USING ERRCODE='55000';
 END IF;
 personas_text:=actual.agregado_json#>>'{solicitud,numero_personas}';
 IF personas_text IS NULL THEN
  personas_text:=actual.agregado_json#>>'{solicitud,necesidad,campos,numero_personas}';
 END IF;
 IF personas_text IS NOT NULL THEN
  IF personas_text !~ '^[1-9][0-9]{0,9}$' THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  IF personas_text::numeric>4294967295 THEN
   RAISE EXCEPTION 'CT201: número de personas inválido' USING ERRCODE='55000';
  END IF;
  personas:=personas_text::numeric;
 END IF;
 sugeridas:='[]'::jsonb;
 IF actual.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
    AND actual.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
    AND actual.agregado_json#>'{via_cobertura,bolsa_ref}' IS NULL
    AND actual.agregado_json#>>'{analisis,categoria_ref}' IS NOT NULL THEN
  bolsa_vigente:=vec_bolsa_llamamientos.resolver_bolsa_vigente_ct_v1(
   actual.agregado_json#>>'{analisis,categoria_ref}');
 END IF;
 IF bolsa_vigente IS NOT NULL THEN
  SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY b.emitido_en DESC,b.llamamiento_ref DESC),'[]'::jsonb)
   INTO sugeridas FROM vec_bolsa_llamamientos.listar_emisiones_ct_v1(
    bolsa_vigente,
    vec_contratacion_temporal.numero_visible_vigente_v1(d.expediente_ref,actual.numero_visible),
    NULL,NULL,20,ARRAY(SELECT x.llamamiento_ref
     FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 x
     WHERE x.expediente_ref=d.expediente_ref)) b;
 END IF;
 RETURN QUERY SELECT d.contenido_canonico,d.esquema,d.acceso_ref,d.secuencia,
  d.anterior_sha256,d.huella_sha256,d.vinculo_identidad_huella_sha256,
  d.alcance_huella_sha256,d.registrada_en,d.auditoria_vec_ref,
  d.auditoria_vec_huella_sha256,d.consumo_vec_huella_sha256,
  d.contenido_huella_sha256,d.resultado_huella_sha256,d.cursor_huella_sha256,
  d.generada_en,d.expediente_ref,d.version_expediente,d.total,d.recibo_sello_sha256,
  jsonb_build_object('vinculos',vinculos,'total_vinculos',n,
   'personas_solicitadas',personas,'aceptaciones_firmes',aceptaciones,
   'emisiones_vinculables',sugeridas,'siguiente_cursor',siguiente),
  d.selectores_canon,d.selectores_huella_sha256,d.sello_selectores_sha256,d.recibo_selectores_canon;
END $_$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)
 TO vec_contratacion_temporal_consultor_rrhh;
DO $post$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)');
 original oid:=to_regprocedure('vec_contratacion_temporal.consultar_original_propuesta_rrhh_con_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 bolsa oid:=to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_y_selectores_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,jsonb)');
 cierre oid:=to_regprocedure('vec_contratacion_temporal.confirmar_vigencia_selectores_rrhh_v1()');
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
  AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
  AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='8695ddb2bf20ae5c25b41182cd164c5974762be8e6dc43f59681f4f43d6c84bb')
  OR original IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=original
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d3ad02aa6e87ff94df3ed961b336757692ca422469c616635f77b9c6f102ff8f')
  OR bolsa IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=bolsa
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
    'vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='853d51a877900a9372a9f1ca446a38dc448a24347543f8625b482a26a3a53e0f')
  OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_contratacion_temporal.prueba_selectores_recuperacion_rrhh_v1')
   AND c.relowner=to_regrole('vec_contratacion_temporal_propietario') AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1')
   AND c.relowner=to_regrole('vec_contratacion_temporal_propietario') AND c.relrowsecurity AND c.relforcerowsecurity)
  OR cierre IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=cierre
   AND p.proowner=to_regrole('vec_contratacion_temporal_propietario') AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s']::text[]
   AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='f1de015e60b2e470c88468ee81165e404a057fa1436b78d69ba1213826a87da9')
  OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid=to_regclass('vec_contratacion_temporal.cierre_vigencia_selectores_rrhh_v1')
   AND t.tgname='cierre_vigencia_selectores_commit' AND t.tgfoid=cierre
   AND t.tgdeferrable AND t.tginitdeferred AND t.tgenabled='O')
 THEN RAISE EXCEPTION 'CT229: postimagen, prueba o ACL divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
