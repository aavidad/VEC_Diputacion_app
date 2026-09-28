\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000057',0));
-- AD3 bloquea primero sus dos registros; así una consulta/publicación V3
-- concurrente no puede quedar confirmada entre la guardia y el DROP Bolsa.
DO $historia_v3$ BEGIN
 IF vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1() THEN
  RAISE EXCEPTION 'B57: DOWN prohibido con historia V3' USING ERRCODE='55000';
 END IF;
END $historia_v3$;
LOCK TABLE vec_bolsa_llamamientos.causa_participacion_catalogo,
           vec_bolsa_llamamientos.propuesta_causa_participacion,
           vec_bolsa_llamamientos.causa_situacion_participacion,
           vec_bolsa_llamamientos.causa_contacto_participacion,
           vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento IN ACCESS EXCLUSIVE MODE;
-- DOWN solo en ensayo sin historia B57. No reescribir recibos confirmados.
DO $guard$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.causa_participacion_catalogo') IS NULL
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.causa_situacion_participacion)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.causa_contacto_participacion)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.propuesta_causa_participacion)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
      WHERE accion IN ('consultar_causas_participacion','proponer_causa_participacion',
                       'consultar_propuesta_causa_participacion','publicar_causa_participacion'))
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.causa_participacion_catalogo
              WHERE actor_ref<>'sistema:migracion:bolsa57' OR version<>1
                 OR codigo NOT IN ('gestion_situacion','actualizacion_contacto'))
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.causa_participacion_catalogo)<>2
 THEN RAISE EXCEPTION 'B57: DOWN prohibido con historia' USING ERRCODE='55000'; END IF;
END $guard$;

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


DROP FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v2(text,text);
DROP FUNCTION vec_bolsa_llamamientos.recuperar_datos_contacto_participacion_v2(text,text);
DROP FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text);
DROP FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text);
DROP FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v2(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text,text,bigint,text);
DROP FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v2(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bigint,text);
DROP FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.exigir_causa_participacion_v1(text,bigint,text,text);
DROP FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.listar_causas_participacion_v1();
DROP FUNCTION vec_bolsa_llamamientos.publicar_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text);
DROP FUNCTION vec_bolsa_llamamientos.proponer_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.consultar_propuesta_causa_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_bolsa_llamamientos.etiqueta_causa_participacion_publicable_v1(text,bigint,text);
DROP TABLE vec_bolsa_llamamientos.causa_situacion_participacion;
DROP TABLE vec_bolsa_llamamientos.causa_contacto_participacion;
DROP TABLE vec_bolsa_llamamientos.causa_participacion_catalogo;
DROP TABLE vec_bolsa_llamamientos.propuesta_causa_participacion;
DROP FUNCTION vec_bolsa_llamamientos.huella_causa_participacion_v1(text,bigint,text,boolean,boolean,boolean,boolean);
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_datos_contacto_origen_participacion_v1(text,text,bigint,text,bytea,bytea,text,text,timestamptz,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,timestamptz,date,text,text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
-- DOWN solo sin intentos B57; restaura el vocabulario B17 exacto.
DO $bitacora_b57$
BEGIN
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK (accion IN ('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento'));
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
 ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK (ruta_clase IN ('coleccion','detalle','situacion','contactos','datos_contacto','emisiones'));
END $bitacora_b57$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
	 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos','datos_contacto','emisiones') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado','correcto') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;

COMMIT;
