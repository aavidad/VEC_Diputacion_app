\set ON_ERROR_STOP on
-- Fixture privado genuino: AUT42/AD184/CA32 instalados, publicación y lectura
-- V3 con COMMIT real. Variables psql del operador por stdin/archivo privado:
-- runtime_login, material, cap_hex, decision_hex, motivo_hex, contexto_hex,
-- persona_version, perfil_version, payload_hex, sobre_hex, evidencia_hex,
-- raiz_hex, acuse, demora_segundos. No valores sensibles en argv ni Git.
-- Preparar CADA CASO con configuración, raíz o clave vigentes al primer cotejo
-- pero con vencimiento real DURANTE demora; capacidad/decisión siguen vigentes.
-- No actualizar historia instalada, no fakegate ni firmas o permiso favorables.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
SET SESSION AUTHORIZATION :"runtime_login";
SELECT 1 / CASE WHEN vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1(
 :'material',decode(:'cap_hex','hex'),decode(:'decision_hex','hex'),decode(:'motivo_hex','hex'),decode(:'contexto_hex','hex'),
 :'persona_version'::numeric,:'perfil_version'::numeric,decode(:'payload_hex','hex'),decode(:'sobre_hex','hex'),decode(:'evidencia_hex','hex'),decode(:'raiz_hex','hex'),:'acuse') IS TRUE THEN 1 ELSE 0 END;
RESET SESSION AUTHORIZATION;
-- Sólo introduce demora en la función original, tras todas sus comprobaciones;
-- el resultado procede del cuerpo real. El cambio desaparece con ROLLBACK.
SELECT set_config('vec.ensayo.ad184.demora',:'demora_segundos',true);
DO $demora$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.cotejar_consumo_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)');
 cuerpo text;marca text:=' RETURN clock_timestamp()<(c->>''expira_en'')::timestamptz';segundos numeric;
BEGIN
 segundos:=current_setting('vec.ensayo.ad184.demora')::numeric;
 IF segundos<=0 OR segundos>4 THEN RAISE EXCEPTION 'AD184: vector_demora_fuera_limite';END IF;
 SELECT pg_get_functiondef(f)INTO STRICT cuerpo;
 IF length(cuerpo)-length(replace(cuerpo,marca,''))<>length(marca) THEN RAISE EXCEPTION 'AD184: vector_marca_no_unica';END IF;
 EXECUTE replace(cuerpo,marca,' PERFORM pg_catalog.pg_sleep('||segundos::text||');'||chr(10)||marca);
END $demora$;
SET SESSION AUTHORIZATION :"runtime_login";
SELECT 1 / CASE WHEN vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1(
 :'material',decode(:'cap_hex','hex'),decode(:'decision_hex','hex'),decode(:'motivo_hex','hex'),decode(:'contexto_hex','hex'),
 :'persona_version'::numeric,:'perfil_version'::numeric,decode(:'payload_hex','hex'),decode(:'sobre_hex','hex'),decode(:'evidencia_hex','hex'),decode(:'raiz_hex','hex'),:'acuse') IS FALSE THEN 1 ELSE 0 END;
-- El rechazo no puede atribuirse a la propia caducidad de la capacidad.
SELECT 1 / CASE WHEN clock_timestamp()<(convert_from(decode(:'cap_hex','hex'),'UTF8')::jsonb->>'expira_en')::timestamptz
 AND clock_timestamp()<(convert_from(decode(:'cap_hex','hex'),'UTF8')::jsonb->>'decision_valida_hasta')::timestamptz THEN 1 ELSE 0 END;
RESET SESSION AUTHORIZATION;
ROLLBACK;
