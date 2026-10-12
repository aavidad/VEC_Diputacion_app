\set ON_ERROR_STOP on
-- AD241: unicamente la escritura propia de preferencias entre bolsas.
-- Preimagen externa literal AD228, conservada por AD238/AD240. Los GET no
-- incorporan consumidores V3: esperan sesion certificada y auditoria nominal.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000241',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 p pg_proc%ROWTYPE; v_def text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR session_user=current_user
    OR f IS NULL OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD241: preimagen ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(v_def,'UTF8')),'hex') IS DISTINCT FROM '648c57689742532689fea2b1d6d3e4e13d2c5431c6f04f8b78adf97ad21af081'
    OR p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'AD241: nucleo externo divergente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Definicion completa literal: conserva la preimagen externa y añade solo
-- una rama de accion/audiencia estrictamente propia.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    c jsonb;
    d jsonb;
    x jsonb;
    v_clave record;
    v_puntero_clave record;
    v_config record;
    v_raiz record;
    v_registro record;
    v_replay record;
    v_ahora timestamptz(6);
    v_huella_capacidad text;
    v_huella_consumo text;
    v_preimagen_auditoria bytea;
    v_anterior text;
    v_secuencia numeric(20, 0);
    v_auditoria_ref text;
    v_statement numeric;
    v_idle numeric;
    v_revalidada_en timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR session_user <> 'vec_externo_bolsa_desarrollo'
       OR p_perfil_mutacion IS NULL
       OR p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa','presentacion_inscripcion')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user
          AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb
          AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
           WHERE m.member=session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
          WHERE m.member=session_user::regrole
            AND m.roleid='vec_bolsa_llamamientos_portal_externo'::regrole
            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR EXISTS (WITH RECURSIVE roles(rol_id) AS (
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             WHERE m.member=session_user::regrole
            UNION
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             JOIN roles r ON r.rol_id=m.member)
           SELECT 1 FROM roles
            WHERE rol_id<>'vec_bolsa_llamamientos_portal_externo'::regrole)
    THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consumo externo VEC-AD-3 rechazado';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'límites VEC-AD-3 ausentes';
    END IF;
    IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
           p_capacidad_canonica) IS NOT TRUE
       OR pg_catalog.octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_sobre_cose_sign1)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR pg_catalog.scale(p_persona_version) <> 0
       OR pg_catalog.scale(p_perfil_version) <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada VEC-AD-3 inválida';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        x := pg_catalog.convert_from(p_contexto_actor_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada VEC-AD-3 inválida';
    END;
    IF vec_autorizacion_atestada_v3.capacidad_tipos_validos(c) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.capacidad_canonica(c)
          IS DISTINCT FROM p_capacidad_canonica
       OR c ->> 'esquema' <>
          'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
       OR c ->> 'version' <> '3'
       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.solicitud.crear'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.solicitud.crear'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'asignacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.unidad.asignar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.unidad.asignar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'asignacion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'informe_juridico'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.informe_juridico.generar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.informe_juridico.generar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'informe_juridico_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'fiscalizacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.fiscalizacion.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.fiscalizacion.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'fiscalizacion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.renuncia_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.siguiente.abrir'
               )
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'integracion_llamamientos_bolsa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'comunicacion_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'comunicacion_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_seleccion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'reanudacion_seleccion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'respuesta_recibida_rrhh'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'respuesta_recibida_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_justificante_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_justificante'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_justificante'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'justificante_respuesta_recibida_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resolucion_manual_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_manual_respuesta_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'continuacion_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.siguiente.continuar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.siguiente.continuar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'continuacion_llamamiento_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'propuesta_formalizacion_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.formalizacion.propuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.formalizacion.propuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_formalizacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'organizacion_preparacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'estructura_organizativa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_estructura_organizativa'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.presentar',
                   'contratacion_temporal.peticion_centro.ratificar',
                   'contratacion_temporal.peticion_centro.consultar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'entrega_peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.rrhh.consultar',
                   'contratacion_temporal.peticion_centro.rrhh.entregar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'entrega_peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitar_peticion_centro_rrhh'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resolucion_formalizacion_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.formalizacion.resolucion.manual_ejercicio.registrar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_formalizacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_personal.alta_ejercicio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_relacion_ocupacion_sinteticas'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_ejercicio_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.incorporacion_ejercicio.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.incorporacion.confirmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_incorporacion_ejercicio_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_incorporacion_confirmada_por_personal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'anotacion_administrativa_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.anotacion_administrativa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.anotacion_administrativa.registrar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'anotacion_administrativa_incorporacion_v1'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_anotacion_administrativa_incorporacion'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'cierre_administrativo_sin_cese_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.cerrar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'seguimiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'cerrar_expediente_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'subsanacion_reparos_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'subsanacion_reparo_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'despacho_correo_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.correo.despachar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'despacho_correo_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resultado_correo_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.resultado_correo_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.correo.registrar_resultado'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resultado_correo_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.mi-bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'consulta_participaciones_propias'
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_borradores_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_contactos_participacion')
 OR (p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_contactos_participacion')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_datos_contacto_participacion')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'emision_llamamiento_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.llamamiento.emitir.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'accion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_llamamientos_bolsa')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_relacion_propia_dietas_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.relacion_propia.consultar_dietas.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'relacion_empleado_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'crear_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'crear_borrador_propio')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consultar_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borrador_propio')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'acceso_rutas_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas_rutas_v1.acceso.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_itinerario_dietas'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IS NOT DISTINCT FROM 'dietas.ruta.catalogo.consultar' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_rutas_dietas')
   OR (d->>'accion' IS NOT DISTINCT FROM 'dietas.ruta.calculo.solicitar' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'calculo_rutas_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_organizacion_historica_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.organizacion_historica.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.organizacion_historica.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'organizacion_historica'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_organizacion_historica'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'importacion_organizacion_historica_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.organizacion_historica.importar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'importacion_organizacion_historica'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND (
   (d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.preparar'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_organizacion_historica'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["hechos","manifiesto","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.conciliar'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'conciliar_organizacion_historica'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["decisiones","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar')
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (p_perfil_mutacion='cronos_marcaje_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_propio.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.propio.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_marcaje_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_marcaje_remoto_disponibilidad'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_remoto_disponibilidad.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.remoto.disponibilidad.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_remoto_disponibilidad'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_disponibilidad_marcaje_remoto'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["autorizado","continuidad_confirmada","motivo","movimientos_permitidos","periodo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_marcaje_remoto_recibo'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_remoto_recibo.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.remoto.recibo.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_remoto_recibo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'recuperar_recibo_marcaje_remoto'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_saldo_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.saldo_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.saldo.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'saldo_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_saldo_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["detalle","periodo","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_movimientos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.movimientos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.movimientos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'movimientos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_movimientos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["absentismos","calendario","correcciones","periodo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion_propia.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correccion_marcaje'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_correccion_marcaje'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permisos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permisos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permisos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'permisos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_permisos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","pendientes","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permiso_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permiso_propio.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permiso.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_permiso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_permiso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_mutacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.editar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.editar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'editar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.borrar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.borrar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'borrar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.enviar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.enviar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'enviar_borrador_propio')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_consulta'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.propio.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento_propio.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_documento_propio_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correccion_asignacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_asignacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.grupo_corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.grupo_corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_grupo_dieta')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'alta_inicial_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.registrar_inicial'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.registrar_inicial.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_asignacion_dietas_inicial')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'prelectura_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.circuito.preleer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.circuito.preleer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preleer_competencia_circuito_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'circuito_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.revisar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.revisar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.autorizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.autorizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'autorizar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.liquidar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.liquidar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'liquidar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.fiscalizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.fiscalizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'fiscalizar_documento_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.revision.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.revision.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_revision_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.autorizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.autorizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_autorizacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.liquidacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.liquidacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_liquidacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.fiscalizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.fiscalizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_fiscalizacion_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'competencias_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.competencias_consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.competencias.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas_competencias'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'tramitar_dietas_asignadas'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'rectificacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.solicitar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.propia.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificacion_dietas_propia')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.resolver' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.resolver.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.competente.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificaciones_competentes_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificaciones_dietas_competentes')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'revisor_documento_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.circuito.documento.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.circuito.documento.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_documento_circuito_dietas'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.alta.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.alta.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'alta_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.hecho.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.hecho.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_hecho_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'registro_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_ficha_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.vacantes.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.vacantes.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vacantes_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_vacantes'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","cursor","cursor_siguiente","evidencia","limite","organismo_ref","vacantes"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.publicar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.publicar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["entrada","recibo"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.retirar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.retirar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["entrada","recibo"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.consultar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cursor_siguiente","entradas","evidencia","organismo_ref"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.empleados.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.empleados.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'empleados_rrhh'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_empleados'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","cursor","cursor_siguiente","empleados","evidencia","limite","organismo_ref"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cese_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cese.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.cesar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cese_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_cese_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cierre_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cierre_expediente.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.cerrar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cierre_expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'cerrar_expediente_tras_cese'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'modificacion_tras_nombramiento_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.modificacion_tras_nombramiento.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.modificar_tras_nombramiento'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'modificacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'modificar_expediente_tras_nombramiento'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND ((((c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_pausa'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_reactivacion'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.responder_llamamiento'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'))
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.manifestar_disposicion'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'oferta_bolsa'))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'firma_documento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_documento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'firma_documento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
 AND c->>'efecto_ref' ~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.confirmar_contacto'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (d->>'accion' IS NOT DISTINCT FROM 'documentos.generado.alta'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_generado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'alta_documento_generado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.expediente.listar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_documental'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'listar_documentos_expediente'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["items","siguiente_cursor"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original.descargar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_original'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'descargar_documento_original'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["contenido","documento"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.notificacion.preparar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'notificacion_preparada'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_notificacion'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["preparacion","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cancelacion_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cancelacion_expediente.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cancelacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'confirmacion_ginpix_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmacion_ginpix.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_centro_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'no_incorporacion_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.no_incorporacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.incorporacion.no_incorporacion'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'no_incorporacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_no_incorporacion_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.historial_propio.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.mi-bolsa.historial.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_historial_propio'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_auditoria_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.auditoria.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'auditoria'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historial_auditoria'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' ~ '^[a-z][a-z0-9_]{0,127}$'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'reincorporacion_titular_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.reincorporacion_titular.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.registrar_reincorporacion_titular'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'reincorporacion_titular_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'politica_ofertas_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_ofertas.publicar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_ofertas.publicar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_politica_ofertas_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.consultar','contratacion_temporal.plantillas_documentos.editar','contratacion_temporal.plantillas_documentos.publicar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.plantillas_documentos.consultar'
      THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_cese.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_cese.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'politica_cese_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'politica:bolsa:cese:vigente'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_politica_cese_rrhh'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["politica_cese"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.documental_listar','contratacion_temporal.plantillas_documentos.documental_descargar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borradores_expediente'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_ofertas.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_ofertas.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_politica_ofertas_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["politica_ofertas"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.lectura_reincorporacion_titular.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.consultar_reincorporacion_titular'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'verificar_antecedente_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cese_evento_ref","cese_recibo_ref","documento_ref","documento_sha256","existe_cese","fecha_efectiva","relacion_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reincorporacion_titular.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reincorporaciones_titular"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_recibo_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_recibo'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'respuesta_recibida_comunicacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM
                   '["auditoria_ref","comunicacion_ref","estado","expediente_ref","justificante_ref","organizacion_ref","recibo_ref","registrada_en","respuesta"]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_comunicaciones_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.comunicaciones_expediente.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.comunicaciones.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["antecedente_tipo","comunicacion_ref","estado","estado_respuesta","expediente_ref","llamamiento_ref","organizacion_ref","recibo_antecedente_ref","recibo_comunicacion_ref","registrada_en","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_consulta_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_actualizacion_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.actualizar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_consultar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","direccion","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_anadir_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.anadir.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.anadir.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.anadir'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","direccion","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_reenviar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.reenviar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.reenviar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.reenviar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_verificar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.verificar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.verificar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.verificar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_activar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.activar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.activar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.activar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_retirar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.retirar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.retirar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.retirar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'imagen_consultar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.imagen.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'imagen_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["foto","icono","modo","paleta","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'imagen_actualizar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.actualizar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.actualizar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.imagen.actualizar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'imagen_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["foto","icono","modo","paleta","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correo_avisos_llamamiento_usuarios'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_llamamientos_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.consultar.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_alta'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.alta.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.alta'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_rectificar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.rectificar.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.rectificar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["codigo_postal","domicilio","movil","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'ajustes_reglas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.ajustes_reglas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.reglas.consultar_ajustes','contratacion_temporal.reglas.ajustar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_reglas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_reglas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.reglas'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.reglas.consultar_ajustes'
      THEN '["historial","vigente"]'::jsonb ELSE '["ajustes","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'presentacion_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.presentar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.presentar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'inscripcion_convocatoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
       OR c ->> 'nonce' !~ '^[0-9a-f]{64}$'
       OR c ->> 'nonce' = pg_catalog.repeat('0', 64)
       OR c ->> 'clave_version' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'revision_gobierno' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'configuracion_secuencia' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'raiz_version' !~ '^[1-9][0-9]{0,19}$'
       OR (c ->> 'clave_version')::numeric >
          9007199254740991::numeric
       OR (c ->> 'revision_gobierno')::numeric >
          9007199254740991::numeric
       OR (c ->> 'configuracion_secuencia')::numeric >
          9007199254740991::numeric
       OR (c ->> 'raiz_version')::numeric >
          9007199254740991::numeric
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 c ->> 'huella_gobierno_sha256',
                 c ->> 'huella_decision_sha256',
                 c ->> 'huella_motivo_sha256',
                 c ->> 'huella_payload_vec_ad_3_sha256',
                 c ->> 'huella_sobre_cose_sign1_sha256',
                 c ->> 'huella_prueba_confianza_sha256',
                 c ->> 'huella_contexto_sha256',
                 c ->> 'huella_efecto_sha256',
                 c ->> 'huella_configuracion_sha256',
                 c ->> 'huella_raiz_spki_sha256',
                 c ->> 'mac_sha256'
             ]) AS h(valor)
            WHERE vec_autorizacion_atestada_v3.huella_sha256_valida(
                      h.valor) IS NOT TRUE)
       OR x ->> 'esquema' <> 'vec.contexto-actor.vinculado.v2'
       OR x ->> 'persona_version' <> p_persona_version::text
       OR x ->> 'perfil_version' <> p_perfil_version::text
       OR pg_catalog.encode(
           pg_catalog.sha256(p_contexto_actor_canonico), 'hex') <> c ->> 'huella_contexto_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_decision_canonica), 'hex') <> c ->> 'huella_decision_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_motivo_canonico), 'hex') <> c ->> 'huella_motivo_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_payload_vec_ad_3), 'hex') <> c ->> 'huella_payload_vec_ad_3_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_sobre_cose_sign1), 'hex') <> c ->> 'huella_sobre_cose_sign1_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_evidencia_verificacion), 'hex') <> c ->> 'huella_prueba_confianza_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_raiz_publica_spki), 'hex') <> c ->> 'huella_raiz_spki_sha256'
       OR d ->> 'decision_ref' <> c ->> 'decision_ref'
       OR d ->> 'motivo_huella_sha256' <> c ->> 'huella_motivo_sha256'
       OR d ->> 'accion' <> c ->> 'operacion'
       OR d ->> 'recurso_ref' <> c ->> 'efecto_ref'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          c ->> 'huella_efecto_sha256'
       OR (d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz
       OR d #>> '{vinculo_autenticacion_actor,registro_contexto_ref}' <>
          c ->> 'contexto_ref'
       OR d #>> '{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' <>
          c ->> 'huella_contexto_sha256'
       OR d ->> 'principal_id' <> x ->> 'principal_ref'
       OR d ->> 'perfil_activo_ref' <> x ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'ligadura VEC-AD-3 inválida';
    END IF;
    IF p_perfil_mutacion IN ('preferencias_consulta_usuarios','preferencias_actualizacion_usuarios','correos_consultar_usuarios','correos_anadir_usuarios','correos_reenviar_usuarios','correos_verificar_usuarios','correos_activar_usuarios','correos_retirar_usuarios','imagen_consultar_usuarios','imagen_actualizar_usuarios','correo_avisos_llamamiento_usuarios')
       AND NOT (
         (d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
          AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
          AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
          AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
         OR
         (d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
          AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER')
          AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
          AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
       ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Usuarios rechazado';
    END IF;
    IF p_perfil_mutacion IN ('aspirantes_ficha_consultar','aspirantes_ficha_alta','aspirantes_ficha_rectificar')
       AND NOT (
         d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
         AND pg_catalog.pg_has_role(session_user,'vec_aspirantes_ejecutor_externo','MEMBER')
         AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_aspirantes_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
         AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
       ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Aspirantes rechazado';
    END IF;
    v_huella_capacidad := pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex');
    PERFORM 1
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo
     WHERE control_id
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'gobierno VEC-AD-3 no disponible';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v3:decision:' ||
            (c ->> 'decision_ref'), 0));
    SELECT a.decision_ref, a.efecto_ref, a.huella_efecto_sha256,
           a.consumo_huella_sha256, u.auditoria_ref, a.consumida_en,
           t.capacidad_canonica, t.decision_canonica, t.motivo_canonico,
           t.contexto_actor_canonico, t.payload_vec_ad_3,
           t.sobre_cose_sign1, t.evidencia_verificacion,
           t.raiz_publica_spki
      INTO v_replay
      FROM vec_autorizacion_atestada_v3.consumo_decision_v3_externa a
      JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3_externa t
        USING (decision_ref)
      JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa u
        USING (decision_ref)
     WHERE a.decision_ref = c ->> 'decision_ref'
        OR a.nonce = c ->> 'nonce';
    IF FOUND THEN
        IF v_replay.decision_ref <> c ->> 'decision_ref'
           OR v_replay.efecto_ref <> c ->> 'efecto_ref'
           OR v_replay.huella_efecto_sha256 <>
              c ->> 'huella_efecto_sha256'
           OR v_replay.capacidad_canonica <> p_capacidad_canonica
           OR v_replay.decision_canonica <> p_decision_canonica
           OR v_replay.motivo_canonico <> p_motivo_canonico
           OR v_replay.contexto_actor_canonico <>
              p_contexto_actor_canonico
           OR v_replay.payload_vec_ad_3 <> p_payload_vec_ad_3
           OR v_replay.sobre_cose_sign1 <> p_sobre_cose_sign1
           OR v_replay.evidencia_verificacion <>
              p_evidencia_verificacion
           OR v_replay.raiz_publica_spki <> p_raiz_publica_spki THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'conflicto VEC-AD-3';
        END IF;
        v_revalidada_en := vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(
            p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);
        IF v_revalidada_en IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay externo VEC-AD-3 rechazado';
        END IF;
        RETURN QUERY SELECT
            v_replay.decision_ref, v_replay.efecto_ref,
            v_replay.huella_efecto_sha256,
            v_replay.consumo_huella_sha256, v_replay.auditoria_ref,
            v_replay.consumida_en, false;
        RETURN;
    END IF;
    v_ahora := clock_timestamp();
    SELECT k.* INTO v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
     WHERE k.clave_id = c ->> 'clave_id'
       AND k.version = (c ->> 'clave_version')::numeric
     FOR SHARE;
    SELECT p.* INTO v_puntero_clave
      FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p
     WHERE p.clave_id = c ->> 'clave_id'
       AND p.version = (c ->> 'clave_version')::numeric
       AND p.establecida_en <= v_ahora
     ORDER BY p.orden DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND
       OR v_puntero_clave.clave_id IS NULL
       OR v_clave.revision_gobierno <>
          (c ->> 'revision_gobierno')::numeric
       OR v_clave.huella_gobierno_sha256 <>
          c ->> 'huella_gobierno_sha256'
       OR v_clave.emisor_id <> c ->> 'emisor_id'
       OR v_clave.audiencia_consumo <> c ->> 'audiencia_consumo'
       OR (c ->> 'emitida_en')::timestamptz < v_clave.valida_desde
       OR (c ->> 'expira_en')::timestamptz > v_clave.valida_hasta
       OR v_ahora < (c ->> 'emitida_en')::timestamptz
       OR v_ahora >= (c ->> 'expira_en')::timestamptz
       OR (c ->> 'expira_en')::timestamptz <=
          (c ->> 'emitida_en')::timestamptz
       OR (c ->> 'expira_en')::timestamptz >
          (c ->> 'emitida_en')::timestamptz + interval '5 seconds'
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora)
       OR vec_autorizacion_atestada_v3.bytea_igual_constante(
           public.hmac(
               vec_autorizacion_atestada_v3.preimagen_mac(c),
               v_clave.secreto_hmac,
               'sha256'),
           pg_catalog.decode(c ->> 'mac_sha256', 'hex')) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'capacidad VEC-AD-3 rechazada';
    END IF;
    SELECT cfg.*, cp.configuracion_secuencia_minima,
           cp.raiz_version_minima
      INTO v_config
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cfg
        ON cfg.revision = p.configuracion_revision
      JOIN vec_autorizacion_atestada_v3.checkpoint_gobierno_externo cp
        ON cp.control_id
     WHERE p.establecida_en <= v_ahora
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE OF p, cfg;
    SELECT r.* INTO v_raiz
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON r.clave_id = cr.raiz_clave_id
       AND r.version = cr.raiz_version
     WHERE cr.configuracion_revision = v_config.revision
       AND r.clave_id = c ->> 'raiz_clave_id'
       AND r.version = (c ->> 'raiz_version')::numeric
     FOR SHARE OF r;
    IF v_config.revision IS NULL OR v_raiz.clave_id IS NULL
       OR v_config.revision <> c ->> 'revision_confianza'
       OR v_config.secuencia <> (c ->> 'configuracion_secuencia')::numeric
       OR v_config.secuencia < v_config.configuracion_secuencia_minima
       OR v_config.huella_configuracion_sha256 <>
          c ->> 'huella_configuracion_sha256'
       OR v_config.publicada_en <>
          (c ->> 'configuracion_publicada_en')::timestamptz
       OR v_config.expira_en <>
          (c ->> 'configuracion_expira_en')::timestamptz
       OR v_raiz.version < v_config.raiz_version_minima
       OR v_raiz.huella_spki_sha256 <> c ->> 'huella_raiz_spki_sha256'
       OR v_raiz.clave_publica_spki <> p_raiz_publica_spki
       OR v_raiz.valida_desde <> (c ->> 'raiz_valida_desde')::timestamptz
       OR v_raiz.valida_hasta <> (c ->> 'raiz_valida_hasta')::timestamptz
       OR v_raiz.suite <> c ->> 'suite'
       OR v_raiz.audiencia_despliegue <> c ->> 'audiencia_despliegue'
       OR (c ->> 'verificada_en')::timestamptz <
          v_config.publicada_en
       OR (c ->> 'verificada_en')::timestamptz >= v_config.expira_en
       OR (c ->> 'verificada_en')::timestamptz < v_raiz.valida_desde
       OR (c ->> 'verificada_en')::timestamptz >= v_raiz.valida_hasta
       OR v_ahora >= v_config.expira_en OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confianza VEC-AD-3 rechazada';
    END IF;
    SELECT * INTO v_registro
      FROM vec_autorizacion.
           registrar_y_revalidar_decision_contexto_actor_externa_v3(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
    IF NOT FOUND OR v_registro.concedida IS NOT TRUE
       OR v_registro.decision_huella_sha256 <>
          c ->> 'huella_decision_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'decisión VEC-AD-3 rechazada';
    END IF;
    v_ahora := clock_timestamp();
    IF v_ahora >= (c ->> 'expira_en')::timestamptz
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR v_ahora >= v_config.expira_en OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1 FROM
             vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'vigencia VEC-AD-3 agotada';
    END IF;
    v_revalidada_en :=
      vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
    IF v_revalidada_en IS NULL
       OR v_revalidada_en < v_registro.revalidada_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'revalidación viva VEC-AD-3 rechazada';
    END IF;
    v_huella_consumo := pg_catalog.encode(pg_catalog.sha256(
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_capacidad_canonica, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_decision_canonica, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_contexto_actor_canonico, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256')), 'hex');
    INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3_externa (
        decision_ref, huella_decision_sha256, decision_canonica,
        motivo_canonico, contexto_actor_canonico, payload_vec_ad_3,
        sobre_cose_sign1, evidencia_verificacion, raiz_publica_spki,
        capacidad_canonica, huella_capacidad_sha256, efecto_ref,
        huella_efecto_sha256, registrada_en) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        p_decision_canonica, p_motivo_canonico,
        p_contexto_actor_canonico, p_payload_vec_ad_3,
        p_sobre_cose_sign1, p_evidencia_verificacion,
        p_raiz_publica_spki, p_capacidad_canonica,
        v_huella_capacidad, c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_ahora);
    INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3_externa (
        decision_ref, huella_decision_sha256, nonce, efecto_ref,
        huella_efecto_sha256, consumo_huella_sha256, consumida_en) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        c ->> 'nonce', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo, v_ahora);
    SELECT asiento_ad207.secuencia_previa, asiento_ad207.anterior_sha256 INTO STRICT v_secuencia, v_anterior FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_externa_v5() asiento_ad207;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22003',
            MESSAGE = 'límite de secuencia VEC-AD-3 alcanzado';
    END IF;
    v_secuencia := v_secuencia + 1;
    v_auditoria_ref := 'aud_v3_' ||
        pg_catalog.substr(v_huella_consumo, 1, 32);
    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'decision_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_huella_consumo);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora);
    NULL; /* AD207: la cabeza la avanza el sellado diferido */
    RETURN QUERY SELECT
        c ->> 'decision_ref', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo,
        v_auditoria_ref, v_ahora, true;
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada VEC-AD-3 inválida';
END
$function$;

DO $post$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 p pg_proc%ROWTYPE; v_def text; v_extension text:=$extension$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
 AND c->>'efecto_ref' ~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 v_def:=pg_get_functiondef(f);
 IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension)
    OR encode(sha256(convert_to(replace(v_def,v_extension,''),'UTF8')),'hex') IS DISTINCT FROM '648c57689742532689fea2b1d6d3e4e13d2c5431c6f04f8b78adf97ad21af081'
    OR p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
 THEN RAISE EXCEPTION 'AD241: nucleo externo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $post$;

-- La CHECK se amplía desde la postimagen exacta de AD240. No se publica
-- ninguna audiencia de lectura de preferencias.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE anterior text; siguiente text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT anterior FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
   '0c4b261a3822b6432592ab94b3b964f2ea409a56178a4ef4586037d3d4d3879b'
    OR right(anterior,1)<>')' OR strpos(anterior,'vec.bolsa.preferencias-bolsas.ordenar.v1')<>0
 THEN RAISE EXCEPTION 'AD241: audiencia preimagen AD240 divergente' USING ERRCODE='55000'; END IF;
 siguiente:=left(anterior,length(anterior)-1)||
  ' OR audiencia_consumo = ''vec.bolsa.preferencias-bolsas.ordenar.v1''::text)';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||siguiente;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD241: material invalido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.v1'
    OR c->>'efecto_ref' IS NULL OR c->>'efecto_ref' !~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD241: material denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'portal_candidato_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
 THEN RAISE EXCEPTION 'AD241: decision ya consumida' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
DO $cerrar_acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
END $cerrar_acl$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile<>'v'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD241: ACL de consumidor divergente' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
