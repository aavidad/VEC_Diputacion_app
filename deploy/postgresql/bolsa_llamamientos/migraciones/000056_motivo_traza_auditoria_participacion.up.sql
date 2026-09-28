\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000056', 0));

-- B56: proyecta cada traza B34 con causa minimizada de su historia B2/B4.
-- La unión es exacta por participación y recibo; no devuelve el motivo libre.
-- La fila B48 de situación solo permanece si carece de traza: la actuación con
-- cambio se consulta en la fila completa, sin depender de otra página.
-- Preserva firma, consumo V3, cursor y ACL de B48. Requiere B48 y B16.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR position('B56: motivo unido' in pg_get_functiondef(to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))) > 0 THEN
  RAISE EXCEPTION 'estado incompatible para consulta de auditoria Bolsa' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(
 p_participacion_ref text, p_actor_filtro text, p_desde timestamptz, p_hasta timestamptz,
 p_antes_instante timestamptz, p_antes_fuente text, p_antes_id text, p_limite integer,
 p_principal text, p_finalidad text, p_motivo_ref text, p_filtro_sha256 text,
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
 p_persona_version numeric, p_perfil_version numeric, p_payload bytea,
 p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(id text, ocurrido_en timestamptz, accion text, actor_ref text,
 resultado text, expediente_ref text, recibo_ref text, motivo text,
 campo text, valor_anterior text, valor_nuevo text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_consumo record; v_decision jsonb; v_capacidad jsonb; v_huella_recurso text;
BEGIN
 -- B56: motivo unido. Marca de preimagen para denegar doble UP.
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user = current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR current_setting('TimeZone') <> 'UTC'
    OR p_participacion_ref IS NULL OR p_participacion_ref = '' OR octet_length(p_participacion_ref) > 512
    OR p_principal IS NULL OR p_principal = '' OR octet_length(p_principal) > 256
    OR p_finalidad IS NULL OR p_finalidad = '' OR octet_length(p_finalidad) > 256
    OR p_motivo_ref IS NULL OR p_motivo_ref = '' OR octet_length(p_motivo_ref) > 256
    OR p_filtro_sha256 IS NULL OR p_filtro_sha256 !~ '^[0-9a-f]{64}$'
    OR (p_actor_filtro IS NOT NULL AND (p_actor_filtro = '' OR octet_length(p_actor_filtro) > 512))
    OR p_desde IS NULL OR p_hasta IS NULL OR p_desde >= p_hasta
    OR p_hasta - p_desde > interval '31 days'
    OR (p_antes_instante IS NULL) <> (p_antes_id IS NULL)
    OR (p_antes_instante IS NULL) <> (p_antes_fuente IS NULL)
    OR (p_antes_fuente IS NOT NULL AND p_antes_fuente <> 'bolsa')
    OR (p_antes_instante IS NOT NULL AND (p_antes_instante < p_desde OR p_antes_instante >= p_hasta))
    OR (p_antes_id IS NOT NULL AND (p_antes_id = '' OR octet_length(p_antes_id) > 512))
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 IF p_filtro_sha256 IS DISTINCT FROM encode(sha256(convert_to(array_to_string(ARRAY[
    'vec.auditoria.filtro.v1','bolsa',p_participacion_ref,coalesce(p_actor_filtro,''),
    coalesce(to_char(p_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    coalesce(to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    p_limite::text,
    coalesce(to_char(p_antes_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
    coalesce(p_antes_fuente,''),coalesce(p_antes_id,''),p_finalidad,p_motivo_ref
 ],E'\n'),'UTF8')),'hex') THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 -- El contexto usa la representación compacta de encoding/json de Go.
 -- Se escapan también los caracteres HTML que ese codificador protege.
 v_huella_recurso := encode(sha256(convert_to(
  '{"ambitos":{"expediente_ref":'||
  replace(replace(replace(replace(replace(to_json(p_participacion_ref)::text,
    '&','\u0026'),'<','\u003c'),'>','\u003e'),chr(8232),'\u2028'),chr(8233),'\u2029')||
  ',"fuente":"bolsa"},"atributos":{"filtro_sha256":'||to_json(p_filtro_sha256)::text||'}}','UTF8')),'hex');
 BEGIN
  v_decision := convert_from(p_decision,'UTF8')::jsonb;
  v_capacidad := convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END;
 IF v_capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
    OR v_capacidad->>'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR v_capacidad->>'efecto_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->>'principal_id' IS DISTINCT FROM p_principal
    OR v_decision->>'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'auditoria'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'historial_auditoria'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR v_decision->>'finalidad' IS DISTINCT FROM p_finalidad
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_huella_recurso
    OR v_decision->'campos_permitidos' IS DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
    OR v_decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_capacidad->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT v_consumo
   FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(
    p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
    p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.efecto_ref IS DISTINCT FROM p_participacion_ref
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_decision->>'contexto_recurso_huella_sha256' THEN
  RAISE EXCEPTION 'consulta de auditoria Bolsa denegada' USING ERRCODE='42501';
 END IF;

 RETURN QUERY
 WITH hechos AS (
  SELECT ('situacion:' || s.recibo_ref)::text AS id, s.registrada_en AS ocurrido_en,
         coalesce(o.operacion, 'situacion:' || s.situacion)::text AS accion,
         s.actor AS actor_ref, 'confirmado'::text AS resultado,
         s.participacion_ref AS expediente_ref, s.recibo_ref,
         CASE WHEN s.recibo_ref='recibo:situacion:constitucion:' || s.participacion_ref
                    AND s.actor='sistema:constitucion' AND s.situacion='disponible'
                    AND s.motivo='Constitución de bolsa' THEN 'Constitución de bolsa'
              ELSE 'Motivo reservado en Bolsa' END::text AS motivo,
         NULL::text AS campo, NULL::text AS valor_anterior, NULL::text AS valor_nuevo
    FROM vec_bolsa_llamamientos.situacion_participacion s
    LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
      ON o.participacion_ref=s.participacion_ref AND o.desde=s.desde
   WHERE s.participacion_ref=p_participacion_ref
     AND NOT EXISTS (
       SELECT 1 FROM vec_bolsa_llamamientos.traza_valor_participacion t
        WHERE t.participacion_ref=s.participacion_ref AND t.recibo_ref=s.recibo_ref)
  UNION ALL
  SELECT ('cambio:' || t.recibo_ref || ':' || t.campo)::text, t.registrada_en,
         CASE WHEN s.recibo_ref IS NOT NULL
                THEN coalesce(o.operacion, 'situacion:' || s.situacion)
              ELSE 'valor:' || t.campo END::text,
         t.actor, 'confirmado'::text,
         t.participacion_ref, t.recibo_ref,
         CASE WHEN t.campo IN ('situacion','fecha_disponible')
                 AND s.recibo_ref='recibo:situacion:constitucion:' || s.participacion_ref
                 AND s.actor='sistema:constitucion' AND s.situacion='disponible'
                 AND s.motivo='Constitución de bolsa'
                THEN 'Constitución de bolsa'
              ELSE 'Motivo reservado en Bolsa' END::text,
         t.campo, t.valor_anterior, t.valor_nuevo
    FROM vec_bolsa_llamamientos.traza_valor_participacion t
    LEFT JOIN vec_bolsa_llamamientos.situacion_participacion s
      ON t.campo IN ('situacion','fecha_disponible')
     AND s.participacion_ref=t.participacion_ref AND s.recibo_ref=t.recibo_ref
    LEFT JOIN vec_bolsa_llamamientos.operacion_situacion_participacion o
      ON s.participacion_ref=o.participacion_ref AND s.desde=o.desde
    LEFT JOIN vec_bolsa_llamamientos.datos_contacto_participacion d
      ON t.campo IN ('datos_contacto','correo','telefono_1','telefono_2')
     AND d.participacion_ref=t.participacion_ref AND d.recibo_ref=t.recibo_ref
   WHERE t.participacion_ref=p_participacion_ref
     AND (s.recibo_ref IS NOT NULL OR d.recibo_ref IS NOT NULL)
 )
 SELECT h.id,h.ocurrido_en,h.accion,h.actor_ref,h.resultado,h.expediente_ref,
        h.recibo_ref,h.motivo,h.campo,h.valor_anterior,h.valor_nuevo
   FROM hechos h
  WHERE (p_actor_filtro IS NULL OR h.actor_ref=p_actor_filtro)
    AND (p_desde IS NULL OR h.ocurrido_en>=p_desde)
    AND h.ocurrido_en<p_hasta
    AND (p_antes_instante IS NULL OR (h.ocurrido_en,'bolsa',h.id)<(p_antes_instante,p_antes_fuente,p_antes_id))
  ORDER BY h.ocurrido_en DESC,h.id DESC
  LIMIT p_limite+1;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
