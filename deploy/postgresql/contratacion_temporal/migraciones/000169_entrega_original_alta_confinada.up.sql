\set ON_ERROR_STOP on
-- CT169: preparar una entrega con autorización vigente y recuperar sólo
-- el material mínimo del alta original. El llamador debe cotejar su HMAC y
-- recibo ANTES de COMMIT; un fallo revierte también el autoenlace de CT150.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:000169:original_alta_entrega',0));
DO $preimagen$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb)') IS NULL THEN
    RAISE EXCEPTION 'CT169 preimagen incompatible' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT p FROM pg_catalog.pg_proc
  WHERE oid='vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.provolatile<>'v' OR p.prorettype<>'jsonb'::regtype
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')<>'1faa7d805e292180a5d16f41f05afe18e1c4fd1f05a966d64b44dd8aa80be02a'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']::text[]
    OR p.proacl IS DISTINCT FROM ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[] THEN
    RAISE EXCEPTION 'CT169 requiere CT151 intacta' USING ERRCODE='55000';
 END IF;
END
$preimagen$;

CREATE FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(entrega jsonb,original_alta jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' SET idle_in_transaction_session_timeout='20s'
AS $funcion$
DECLARE m jsonb; e jsonb; r record; a jsonb; v_recibo jsonb; v_politica jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN
    RAISE EXCEPTION 'identidad o transacción de recuperación rechazada' USING ERRCODE='P0683';
 END IF;
 IF p_material IS NULL OR pg_catalog.octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
    RAISE EXCEPTION 'material de recuperación inválido' USING ERRCODE='P0680';
 END IF;
 BEGIN m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'material de recuperación inválido' USING ERRCODE='P0680';
 END;
 IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR m->>'modo' IS DISTINCT FROM 'preparar' THEN
    RAISE EXCEPTION 'modo de recuperación inválido' USING ERRCODE='P0680';
 END IF;
 -- Autoridad nominal existente: coteja los ámbitos de la revisión ratificada,
 -- consume decisión CURRENT y registra el acceso dentro de esta transacción.
 -- No existe consulta por perfil histórico o ámbito suministrado libremente.
 e:=vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1(
    p_material,p_capacidad,p_decision,p_motivo,p_contexto,
    p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF (e->>'estado_entrega' IS DISTINCT FROM 'preparada'
     AND e->>'estado_entrega' IS DISTINCT FROM 'confirmada')
    OR e->'peticion'->>'referencia' IS DISTINCT FROM m->>'peticion_ref'
    OR e->>'actor_ref' IS DISTINCT FROM m->>'actor_ref' THEN
    RAISE EXCEPTION 'preparación recuperada incoherente' USING ERRCODE='55000';
 END IF;
 -- Las coordenadas usadas proceden de la reserva devuelta por la operación
 -- autorizada. Sólo se admite el alias HMAC vinculado a esa misma clave UUID.
 SELECT s.clave_alta,s.ambito_alta_hmac,s.actor_ref,s.perfil_ref,
        i.ambito_hmac AS raiz,i.organizacion_ref,i.expediente_ref,
        i.numero_visible,i.recibo_ref,i.reserva_ref,
        i.actor_ref AS identidad_actor,i.perfil_ref AS identidad_perfil,
        h.alias_hmac AS huella_peticion_hmac,
        c.confirmacion_ref,c.expediente_ref AS confirmado_expediente,
        c.numero_visible AS confirmado_numero,c.recibo_ref AS confirmado_recibo,
        c.reserva_ref AS confirmado_reserva,c.version_expediente,
        c.auditoria_ref,c.evento_ref,c.confirmada_en,c.huella_alta_sha256,
        b.organizacion_ref AS alta_organizacion,b.actor_ref AS alta_actor,
        b.perfil_ref AS alta_perfil,b.confirmacion_ref AS alta_confirmacion,
        v.alta_canonica,v.confirmacion_ref AS version_confirmacion,
        v.flujo_ref,v.flujo_version,v.flujo_huella_sha256,
        v.huella_alta_sha256 AS version_huella
   INTO STRICT r
   FROM vec_contratacion_temporal.entrega_peticion_centro_reserva s
   LEFT JOIN vec_contratacion_temporal.alias_ambito_alta x ON x.alias_hmac=s.ambito_alta_hmac
   LEFT JOIN vec_contratacion_temporal.identidad_reserva_alta i ON i.ambito_hmac=x.ambito_raiz_hmac
   LEFT JOIN vec_contratacion_temporal.alias_huella_alta h ON h.ambito_raiz_hmac=i.ambito_hmac AND h.generacion=x.generacion
   LEFT JOIN vec_contratacion_temporal.confirmacion_agregado_alta c ON c.ambito_hmac=i.ambito_hmac
   LEFT JOIN vec_contratacion_temporal.expediente_alta b ON b.expediente_ref=i.expediente_ref
   LEFT JOIN vec_contratacion_temporal.expediente_alta_version v ON v.expediente_ref=i.expediente_ref AND v.version=1
  WHERE s.peticion_ref=m->>'peticion_ref';
 IF r.clave_alta::text IS DISTINCT FROM e->>'clave_alta'
    OR r.ambito_alta_hmac IS DISTINCT FROM e->>'ambito_alta_hmac'
    OR r.actor_ref IS DISTINCT FROM e->>'actor_ref'
    OR r.perfil_ref IS DISTINCT FROM e->>'perfil_ref' THEN
    RAISE EXCEPTION 'reserva original divergente' USING ERRCODE='55000';
 END IF;
 IF r.confirmacion_ref IS NULL THEN
    IF e->>'estado_entrega'='confirmada' OR r.alta_confirmacion IS NOT NULL
       OR r.alta_canonica IS NOT NULL THEN
       RAISE EXCEPTION 'alta original parcialmente confirmada' USING ERRCODE='55000';
    END IF;
    RETURN QUERY SELECT e,NULL::jsonb;
    RETURN;
 END IF;
 IF r.organizacion_ref IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
    OR r.identidad_actor IS DISTINCT FROM r.actor_ref
    OR r.identidad_perfil IS DISTINCT FROM r.perfil_ref
    OR r.alta_organizacion IS DISTINCT FROM r.organizacion_ref
    OR r.alta_actor IS DISTINCT FROM r.actor_ref OR r.alta_perfil IS DISTINCT FROM r.perfil_ref
    OR r.confirmado_expediente IS DISTINCT FROM r.expediente_ref
    OR r.confirmado_numero IS DISTINCT FROM r.numero_visible
    OR r.confirmado_recibo IS DISTINCT FROM r.recibo_ref
    OR r.confirmado_reserva IS DISTINCT FROM r.reserva_ref
    OR r.version_expediente IS DISTINCT FROM 1::numeric
    OR r.alta_confirmacion IS DISTINCT FROM r.confirmacion_ref
    OR r.version_confirmacion IS DISTINCT FROM r.confirmacion_ref
    OR r.version_huella IS DISTINCT FROM r.huella_alta_sha256
    OR r.alta_canonica IS NULL OR r.huella_peticion_hmac IS NULL
    OR pg_catalog.encode(pg_catalog.sha256(r.alta_canonica),'hex') IS DISTINCT FROM r.huella_alta_sha256 THEN
    RAISE EXCEPTION 'alta original incoherente' USING ERRCODE='55000';
 END IF;
 BEGIN a:=pg_catalog.convert_from(r.alta_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'canon de alta original inválido' USING ERRCODE='55000';
 END;
 IF a->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.efecto-alta.v2'
    OR a->>'organizacion_ref' IS DISTINCT FROM r.organizacion_ref
    OR a->>'actor_ref' IS DISTINCT FROM r.actor_ref OR a->>'perfil_ref' IS DISTINCT FROM r.perfil_ref
    OR a->>'expediente_ref' IS DISTINCT FROM r.expediente_ref
    OR a->>'numero_visible' IS DISTINCT FROM r.numero_visible
    OR a->>'recibo_ref' IS DISTINCT FROM r.recibo_ref
    OR a->>'reserva_ref' IS DISTINCT FROM r.reserva_ref OR a->>'version' IS DISTINCT FROM '1'
    OR a#>>'{flujo,definicion_ref}' IS DISTINCT FROM r.flujo_ref
    OR a#>>'{flujo,version}' IS DISTINCT FROM r.flujo_version::text
    OR a#>>'{flujo,huella_sha256}' IS DISTINCT FROM r.flujo_huella_sha256 THEN
    RAISE EXCEPTION 'canon y alta original divergentes' USING ERRCODE='55000';
 END IF;
 -- El canon de alta conserva fechas civiles; el validador de análisis usa
 -- instantes UTC y no se aplica a este material ya sellado.
 v_politica:=a#>'{solicitud,periodo,politica_fin}';
 IF v_politica IS NOT NULL AND
    vec_contratacion_temporal.periodo_previsto_estructural_v1(a#>'{solicitud,periodo}') IS NOT TRUE THEN
    RAISE EXCEPTION 'política original inválida' USING ERRCODE='55000';
 END IF;
 v_recibo:=pg_catalog.jsonb_build_object('expediente_ref',r.expediente_ref,
    'numero_visible',r.numero_visible,'version',1,'recibo_ref',r.recibo_ref,
    'auditoria_ref',r.auditoria_ref,'evento_ref',r.evento_ref,
    'confirmada_en',CASE
        WHEN r.confirmada_en=pg_catalog.date_trunc('second',r.confirmada_en)
          THEN pg_catalog.to_char(r.confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
        ELSE pg_catalog.regexp_replace(pg_catalog.to_char(r.confirmada_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'0+Z$','Z') END);
 IF e->>'estado_entrega'='confirmada' AND e->'recibo_alta' IS DISTINCT FROM v_recibo THEN
    RAISE EXCEPTION 'recibo de entrega y alta original divergentes' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT e,pg_catalog.jsonb_build_object(
    'esquema','vec.contratacion-temporal.original-alta-entrega.v1',
    'organizacion_ref',r.organizacion_ref,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,
    'flujo',a->'flujo','politica_fin',v_politica,'ambito_hmac',r.ambito_alta_hmac,
    'huella_peticion_hmac',r.huella_peticion_hmac,'recibo_alta',v_recibo);
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_contratacion_temporal_migrador,vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
COMMIT;
