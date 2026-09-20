-- Comprobación focal posterior a instalación: no inserta ni modifica datos.
DO $prueba$
DECLARE
    v_oid oid := 'vec_bolsa_llamamientos.consultar_participaciones_propias_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    v_definicion text;
    v_preimagen text;
    v_posicion_consumo integer;
    v_posicion_consumo_nuevo integer;
    v_posicion_revalidacion integer;
    v_posicion_lectura integer;
BEGIN
    v_preimagen := '{"ambitos":{"candidato_ref":' || pg_catalog.to_json('can_aaaaaaaaaaaaaaaaaaaaaa'::text)::text
        || '},"atributos":{"finalidad":"consulta_posicion_propia_bolsa"}}';
    IF v_preimagen IS DISTINCT FROM '{"ambitos":{"candidato_ref":"can_aaaaaaaaaaaaaaaaaaaaaa"},"atributos":{"finalidad":"consulta_posicion_propia_bolsa"}}'
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_preimagen,'UTF8')),'hex') !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'preimagen canónica B11 divergente';
    END IF;
    SELECT pg_catalog.pg_get_functiondef(v_oid) INTO v_definicion;
    v_posicion_consumo := pg_catalog.strpos(
        v_definicion,
        'registrar_consumo_participaciones_propias_b11_v3'
    );
    v_posicion_consumo_nuevo := pg_catalog.strpos(
        v_definicion,
        'IF v_consumo.consumo_nuevo IS NOT TRUE'
    );
    v_posicion_revalidacion := pg_catalog.strpos(
        v_definicion,
        'revalidar_consumo_participaciones_propias_b11_v3'
    );
    v_posicion_lectura := pg_catalog.strpos(
        v_definicion,
        'listar_participaciones_candidato_v1(v_candidato_ref)'
    );
    IF NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid = v_oid)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
                    pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
                   WHERE p.oid=v_oid AND a.grantee=0 AND a.privilege_type='EXECUTE')
       OR NOT pg_catalog.has_function_privilege(
          'vec_bolsa_llamamientos_consultor_participaciones_propias', v_oid, 'EXECUTE')
       OR NOT pg_catalog.has_schema_privilege(
          'vec_bolsa_llamamientos_consultor_participaciones_propias',
          'vec_bolsa_llamamientos', 'USAGE')
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid = v_oid)
          IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s']::text[]
       OR v_posicion_consumo = 0
       OR v_posicion_consumo_nuevo = 0
       OR v_posicion_revalidacion = 0
       OR v_posicion_lectura = 0
       OR NOT (v_posicion_consumo < v_posicion_consumo_nuevo
               AND v_posicion_consumo_nuevo < v_posicion_revalidacion
               AND v_posicion_revalidacion < v_posicion_lectura)
       OR strpos(v_definicion, 'ERRCODE = ''P0663''') = 0
       OR strpos(v_definicion, 'revalidar_consumo_participaciones_propias_b11_v3') = 0
       OR strpos(v_definicion, 'v_principal_ref IS DISTINCT FROM v_contexto ->> ''persona_ref''') = 0
       OR strpos(v_definicion, 'v_numero_candidatos <> 1') = 0
       OR strpos(v_definicion, 'campos_permitidos') = 0
       OR strpos(v_definicion, 'obligaciones') = 0
       OR strpos(v_definicion, '''candidato:'' || v_candidato_ref') = 0
       OR strpos(v_definicion, 'v_consumo.efecto_ref') = 0
       OR strpos(v_definicion, 'v_consumo.huella_efecto_sha256') = 0
       OR strpos(v_definicion, 'v_preimagen_recurso') = 0
       OR strpos(v_definicion, 'v_huella_recurso') = 0
       OR strpos(v_definicion, '"ambitos":{"candidato_ref":') = 0
       OR strpos(v_definicion, 'e ->> ''tipo'' = ''candidato''') = 0
       OR strpos(v_definicion, 'e ->> ''estado'' = ''activo''') = 0
       OR strpos(v_definicion, 'listar_participaciones_candidato_v1(v_candidato_ref)') = 0
       OR strpos(v_definicion, 'auditoria_ref') = 0
       OR strpos(v_definicion, '''esquema'', ''vec.bolsa.mi-bolsa.v1''') = 0
       OR strpos(v_definicion, '''consultada_en''') = 0
       OR strpos(v_definicion, 'date_trunc(''microseconds'', pg_catalog.clock_timestamp())') = 0
       OR strpos(v_definicion, '''YYYY-MM-DD"T"HH24:MI:SS.US"Z"''') = 0
       OR strpos(v_definicion, '''bolsa_ref''') = 0
       OR strpos(v_definicion, '''categoria_ref''') = 0
       OR strpos(v_definicion, '''version_bolsa''') = 0
       OR strpos(v_definicion, '''orden''') = 0
       OR strpos(v_definicion, '''total_participaciones''') = 0
       OR strpos(v_definicion, '''estado_bolsa'', p.estado') = 0
       OR strpos(v_definicion, '''vigente_desde''') = 0
       OR strpos(v_definicion, '''vigente_hasta''') = 0
       OR strpos(v_definicion, '''participacion_ref''') <> 0
       OR strpos(v_definicion, '''acta_ref''') <> 0
       OR strpos(v_definicion, 'contacto') <> 0
       OR strpos(v_definicion, 'puntuacion') <> 0
       OR strpos(v_definicion, '''llamamiento_ref''') <> 0
       OR strpos(v_definicion, 'ultimo_llamamiento') <> 0
       OR strpos(v_definicion, 'contrato') <> 0
       OR strpos(v_definicion, 'situacion') <> 0 THEN
        RAISE EXCEPTION 'fachada B11 estática incompleta o expone campos prohibidos';
    END IF;
END
$prueba$;

-- Vectores negativos de la selección de vínculo: 0 o 2 candidatos no pueden
-- convertirse en un can_*; el principal persona tampoco se confunde con él.
DO $negativos$
DECLARE
    v_contexto jsonb;
    v_numero integer;
    v_referencia text;
BEGIN
    FOREACH v_contexto IN ARRAY ARRAY[
        '{"principal_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","vinculos":[]}'::jsonb,
        '{"principal_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","vinculos":[{"tipo":"candidato","estado":"activo","referencia":"can_aaaaaaaaaaaaaaaaaaaaaa"},{"tipo":"candidato","estado":"activo","referencia":"can_bbbbbbbbbbbbbbbbbbbbbb"}]}'::jsonb
    ] LOOP
        SELECT pg_catalog.count(*), pg_catalog.min(e ->> 'referencia')
          INTO v_numero, v_referencia
          FROM pg_catalog.jsonb_array_elements(v_contexto -> 'vinculos') e
         WHERE e ->> 'tipo' = 'candidato' AND e ->> 'estado' = 'activo'
           AND e ->> 'referencia' ~ '^can_[A-Za-z0-9_-]{22,128}$';
        IF v_contexto ->> 'principal_ref' <> v_contexto ->> 'persona_ref'
           OR v_contexto ->> 'principal_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
           OR v_numero = 1 OR v_referencia IS NOT NULL AND v_numero = 1 THEN
            RAISE EXCEPTION 'vector negativo B11 aceptado indebidamente';
        END IF;
    END LOOP;
END
$negativos$;
