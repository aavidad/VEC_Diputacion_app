\set ON_ERROR_STOP on
-- CT244 / F1-98. Desde F1-87 el centro puede declarar la RC y el alta v3
-- (CT150, CT239) guarda su fecha como fecha civil «2006-01-02», igual que
-- los periodos. Go la lee como medianoche UTC y la devuelve en RFC3339 dentro
-- del expediente anterior del análisis. La normalización de CT202 solo
-- igualaba los periodos, así que la confirmación del análisis de todo
-- expediente nuevo con RC declarada fallaba con «transición O3 v2 en
-- conflicto» (503 en la pantalla). Se añade la misma igualación para la
-- fecha de la RC existente. Solo cambia la comparación: el agregado guardado
-- conserva sus bytes. Propietario, ACL, volatilidad y configuración, iguales.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000244',0));
DO $pre$
DECLARE p record;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'CT244: dependencias incompatibles' USING ERRCODE='55000'; END IF;
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.provolatile,q.proacl::text AS acl,q.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
 INTO p FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)');
 -- Preimagen exacta de CT202; la postimagen de CT244 tiene otra huella, así
 -- que una segunda aplicación también se detiene aquí.
 IF NOT FOUND OR p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT FALSE
    OR p.provolatile IS DISTINCT FROM 'i'
    OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR p.huella IS DISTINCT FROM 'ac6c9964bee22ed4cc04d7ddd852c8372a75c9b5bf970f77f7534de053031d86' THEN
  RAISE EXCEPTION 'CT244: preimagen incompatible o ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(p_agregado jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    resultado jsonb := p_agregado;
    actuacion jsonb;
    analisis jsonb;
    indice integer;
    cantidad integer;
BEGIN
    resultado := pg_catalog.jsonb_set(
        resultado, '{creado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'creado_en'
            )
        ), false
    );
    resultado := pg_catalog.jsonb_set(
        resultado, '{actualizado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'actualizado_en'
            )
        ), false
    );
    -- O2-07 usa una forma canónica de transporte para RC ausente.
    -- Solo su valor neutro exacto se convierte a la forma del dominio Go.
    IF resultado #>> '{solicitud,rc,existe}' = 'false'
       AND resultado #>> '{solicitud,rc,fecha}' = ''
       AND resultado #>> '{solicitud,rc,importe,centimos}' = '0'
       AND resultado #>> '{solicitud,rc,importe,moneda}' = 'EUR'
       AND coalesce(resultado #>> '{solicitud,rc,numero}', '') = ''
       AND coalesce(
           resultado #>> '{solicitud,rc,documento_ref}', ''
       ) = '' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,fecha}',
            pg_catalog.to_jsonb('0001-01-01T00:00:00Z'::text), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,importe,moneda}',
            pg_catalog.to_jsonb(''::text), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,inicio}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,inicio}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,inicio}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,fin}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,fin}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,fin}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    -- CT193 guarda la misma fecha civil también en la necesidad sellada.
    -- La comparación con la proyección Go usa UTC a medianoche en ambos
    -- lugares; el agregado persistido conserva sus bytes originales.
    IF resultado #>> '{solicitud,necesidad,periodo,inicio}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,necesidad,periodo,inicio}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,necesidad,periodo,inicio}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,necesidad,periodo,fin}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,necesidad,periodo,fin}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,necesidad,periodo,fin}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    -- CT244 / F1-98: con RC declarada por el centro (F1-87) el alta guarda
    -- también su fecha como fecha civil; se compara igual que los periodos.
    IF resultado #>> '{solicitud,rc,existe}' = 'true'
       AND resultado #>> '{solicitud,rc,fecha}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,fecha}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,rc,fecha}') || 'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,observaciones}' = '' THEN
        resultado := resultado #- '{solicitud,observaciones}';
    END IF;
    IF resultado #>> '{solicitud,rc,numero}' = '' THEN
        resultado := resultado #- '{solicitud,rc,numero}';
    END IF;
    IF resultado #>> '{solicitud,rc,documento_ref}' = '' THEN
        resultado := resultado #- '{solicitud,rc,documento_ref}';
    END IF;
    IF resultado #> '{solicitud,documentos_adjuntos}' = '[]'::jsonb THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,documentos_adjuntos}',
            'null'::jsonb, false
        );
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'actuaciones') = 'array' THEN
        cantidad := pg_catalog.jsonb_array_length(
            resultado -> 'actuaciones'
        );
        IF cantidad > 0 THEN
            FOR indice IN 0..cantidad - 1 LOOP
                actuacion := resultado #> ARRAY[
                    'actuaciones', indice::text
                ];
                IF actuacion ->> 'observaciones' = '' THEN
                    actuacion := actuacion - 'observaciones';
                END IF;
                IF actuacion -> 'documentos_ref' = '[]'::jsonb THEN
                    actuacion := actuacion - 'documentos_ref';
                END IF;
                actuacion := pg_catalog.jsonb_set(
                    actuacion, '{realizada_en}',
                    pg_catalog.to_jsonb(
                        vec_contratacion_temporal.texto_instante_utc_go_v2(
                            actuacion ->> 'realizada_en'
                        )
                    ), false
                );
                resultado := pg_catalog.jsonb_set(
                    resultado,
                    ARRAY['actuaciones', indice::text],
                    actuacion,
                    false
                );
            END LOOP;
        END IF;
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'analisis') = 'object' THEN
        analisis := resultado -> 'analisis';
        analisis := pg_catalog.jsonb_set(
            analisis, '{validacion_rc,validada_en}',
            pg_catalog.to_jsonb(
                vec_contratacion_temporal.texto_instante_utc_go_v2(
                    analisis #>> '{validacion_rc,validada_en}'
                )
            ), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{analisis}', analisis, false
        );
    END IF;
    RETURN resultado;
END
$function$;

DO $post$
DECLARE p record;
BEGIN
 SELECT pg_get_userbyid(q.proowner) AS dueno,q.prosecdef,q.provolatile,q.proacl::text AS acl,q.proconfig,
  encode(sha256(convert_to(pg_get_functiondef(q.oid),'UTF8')),'hex') AS huella
 INTO STRICT p FROM pg_proc q WHERE q.oid='vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)'::regprocedure;
 IF p.dueno<>'vec_contratacion_temporal_propietario' OR p.prosecdef IS NOT FALSE
    OR p.provolatile IS DISTINCT FROM 'i'
    OR p.acl IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR p.huella IS DISTINCT FROM '5a84a4ca69933b17a8ea35b1afbe8e4b4ffe88d30280097c913e3cc9d216e1af'
    OR has_function_privilege('public','vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)','EXECUTE')
    OR vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
         '{"creado_en":"2026-01-01T00:00:00Z","actualizado_en":"2026-01-01T00:00:00Z","solicitud":{"rc":{"existe":true,"fecha":"2026-10-15"}}}'
       ) #>> '{solicitud,rc,fecha}' IS DISTINCT FROM '2026-10-15T00:00:00Z'
    OR vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
         '{"creado_en":"2026-01-01T00:00:00Z","actualizado_en":"2026-01-01T00:00:00Z","solicitud":{"rc":{"existe":true,"fecha":"2026-10-15T00:00:00Z"}}}'
       ) #>> '{solicitud,rc,fecha}' IS DISTINCT FROM '2026-10-15T00:00:00Z'
    OR vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(
         '{"creado_en":"2026-01-01T00:00:00Z","actualizado_en":"2026-01-01T00:00:00Z","solicitud":{"rc":{"existe":false,"fecha":"2026-10-15"}}}'
       ) #>> '{solicitud,rc,fecha}' IS DISTINCT FROM '2026-10-15' THEN
  RAISE EXCEPTION 'CT244: ACL o postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
