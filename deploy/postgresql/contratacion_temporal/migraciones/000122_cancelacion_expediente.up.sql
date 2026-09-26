\set ON_ERROR_STOP on
-- CT122: cancelación del expediente antes de la fiscalización (duda 12 de
-- RRHH). El centro solicitante sobre su petición o RRHH sobre cualquier
-- expediente de su ámbito lo dan por terminado con un motivo del catálogo
-- gobernado y una observación opcional.
--  * Solo desde las fases que fija la regla c20 del catálogo (llegan en el
--    material y quedan ligadas a la decisión autorizada) y siempre antes de
--    la fiscalización: un expediente con fiscalización en su historia no se
--    cancela aquí, porque tras ella hay llamamientos y efectos en Bolsa que
--    esta operación no deshace.
--  * Añade una versión del expediente en estado `cancelado` (terminal) en su
--    misma fase, con su actuación; consume la autorización AD3-87 y publica
--    `ct.expediente_cancelado.v1` en el outbox del expediente, todo en una
--    transacción. La publicación al cuadro RRHH la hace el disparador de
--    versiones ya existente.
--  * Bolsa: antes de la fiscalización no existe llamamiento ni candidato
--    llamado, así que no hay a quién liberar ni penalizar. No se publica
--    ningún evento hacia Bolsa.
-- Historia de solo adición: ninguna fila anterior se modifica. La migración
-- es autónoma: no usa auxiliares de otras migraciones de CT.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000122',0));

DO $prevalidacion$
DECLARE v_origen text; t text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION 'CT122: propietario incompatible' USING ERRCODE='55000';
    END IF;
    FOREACH t IN ARRAY ARRAY['expediente_version_integral','expediente_integral_actual','actuacion_expediente_integral',
        'outbox_expediente_integral','control_cadenas_expediente_integral','publicacion_version_rrhh'] LOOP
        IF to_regclass('vec_contratacion_temporal.'||t) IS NULL THEN
            RAISE EXCEPTION 'CT122: falta %', t USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF to_regprocedure('vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[])') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)') IS NULL
       OR to_regclass('vec_contratacion_temporal.cancelacion_expediente_v1') IS NOT NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'CT122: dependencias incompatibles (AD3-87 requerida)' USING ERRCODE='55000';
    END IF;
    -- El estado `cancelado` ya es válido en versiones y en la publicación RRHH.
    IF strpos((SELECT pg_get_constraintdef(oid) FROM pg_constraint
               WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
                 AND conname='expediente_version_integral_estado_check'),'''cancelado''::text')=0
       OR strpos((SELECT pg_get_constraintdef(oid) FROM pg_constraint
               WHERE conrelid='vec_contratacion_temporal.publicacion_version_rrhh'::regclass
                 AND conname='publicacion_version_rrhh_estado_clave_valido'),'''cancelado''::text')=0 THEN
        RAISE EXCEPTION 'CT122: estado cancelado no admitido por la estructura' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF strpos(v_origen,'''alta_o2''::text')=0 OR right(v_origen,4)<>'])))'
       OR strpos(v_origen,'cancelacion_expediente_ct122')<>0 THEN
        RAISE EXCEPTION 'CT122: preimagen de origen de versión incompatible' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

-- La lista de orígenes se amplía sobre la vigente: otras migraciones pueden
-- haberla ampliado antes; se conservan todos sus valores.
DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||left(v_origen,length(v_origen)-4)||', ''cancelacion_expediente_ct122''::text])))';
END
$origen$;

-- Texto libre acotado: sin espacios de borde, NFC y sin controles salvo
-- tabulador y salto de línea (mismo criterio que el dominio en Go).
CREATE FUNCTION vec_contratacion_temporal.texto_valido_ct122(p text, p_maximo integer, p_vacio boolean)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT p IS NOT NULL AND (p='' AND p_vacio OR p<>'' AND char_length(p)<=p_maximo AND octet_length(p)<=p_maximo*4
       AND p=btrim(p,E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(8232)||chr(8233)||chr(12288))
       AND p=normalize(p,NFC) AND translate(p,E'\t\n','') !~ '[[:cntrl:]]')
$$;

CREATE FUNCTION vec_contratacion_temporal.referencia_valida_ct122(p text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT p IS NOT NULL AND p ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
$$;

-- Serialización de un mapa plano con claves ordenadas, idéntica a la de Go
-- para los valores que admite esta migración (ASCII sin escapes especiales).
CREATE FUNCTION vec_contratacion_temporal.mapa_go_ct122(p jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
DECLARE v text;
BEGIN
    IF jsonb_typeof(p)<>'object' OR EXISTS (SELECT 1 FROM jsonb_each(p) e
        WHERE jsonb_typeof(e.value)<>'string' OR e.key !~ '^[a-z][a-z0-9_]{0,63}$'
           OR e.value #>> '{}' !~ '^[A-Za-z0-9._:/#, _-]*$') THEN
        RAISE EXCEPTION 'CT122: mapa de contexto no admitido' USING ERRCODE='22023';
    END IF;
    SELECT '{'||coalesce(string_agg('"'||e.key||'":"'||(e.value #>> '{}')||'"',',' ORDER BY e.key COLLATE "C"),'')||'}'
      INTO v FROM jsonb_each(p) e;
    RETURN v;
END
$$;

CREATE FUNCTION vec_contratacion_temporal.huella_contexto_go_ct122(p_ambitos jsonb, p_atributos jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
    SELECT encode(sha256(convert_to('{"ambitos":'||vec_contratacion_temporal.mapa_go_ct122(p_ambitos)
        ||',"atributos":'||vec_contratacion_temporal.mapa_go_ct122(p_atributos)||'}','UTF8')),'hex')
$$;

-- Fases admitidas en el orden del catálogo, unidas por coma (como en Go).
CREATE FUNCTION vec_contratacion_temporal.fases_texto_ct122(m jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
    SELECT string_agg(e #>> '{}',',' ORDER BY o) FROM jsonb_array_elements(m->'fases_admitidas') WITH ORDINALITY x(e,o)
$$;

-- Forma de dominio del agregado vigente. La versión 1 procede del alta, que
-- guarda su forma canónica de transporte; el análisis la normaliza igual
-- antes de compararla con la proyección de Go (CT13/CT49).
CREATE FUNCTION vec_contratacion_temporal.agregado_dominio_ct122(p_agregado jsonb, p_version numeric)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT CASE WHEN p_version=1 THEN vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(p_agregado) ELSE p_agregado END
$$;

-- Solo el ejecutor de CT, en transacción serializable del tipo esperado.
CREATE FUNCTION vec_contratacion_temporal.exigir_sesion_ct122(p_lectura boolean)
RETURNS void LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>(CASE WHEN p_lectura THEN 'on' ELSE 'off' END) THEN
        RAISE EXCEPTION 'CT122: sesión no autorizada' USING ERRCODE='42501';
    END IF;
END
$$;

-- Material de cancelación: forma exacta y tipos.
CREATE FUNCTION vec_contratacion_temporal.validar_material_cancelacion_ct122(m jsonb)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','canal','motivo_clave','fases_admitidas','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
            CASE WHEN c.key='version_esperada' THEN 'number' WHEN c.key='fases_admitidas' THEN 'array' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR m->>'canal' NOT IN ('rrhh','centro')
       OR m->>'motivo_clave' !~ '^[a-z][a-z0-9_.-]{1,79}$'
       OR jsonb_array_length(m->'fases_admitidas') NOT BETWEEN 1 AND 16
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(m->'fases_admitidas') e WHERE jsonb_typeof(e) IS DISTINCT FROM 'string'
                  OR e #>> '{}' !~ '^[a-z][a-z0-9_.-]{1,79}$')
       OR (SELECT count(DISTINCT e #>> '{}')<>count(*) FROM jsonb_array_elements(m->'fases_admitidas') e)
       OR NOT vec_contratacion_temporal.texto_valido_ct122(m->>'observaciones',2000,true) THEN
        RAISE EXCEPTION 'CT122: material de cancelación inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct122(m->>k) THEN
            RAISE EXCEPTION 'CT122: referencia de cancelación inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
END
$$;

-- Entrada de preparación: esquema, sellos HMAC (activo y retenidos) y
-- referencias candidatas. Devuelve la lista de pares HMAC en orden.
CREATE FUNCTION vec_contratacion_temporal.validar_preparacion_ct122(p jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE v_par jsonb; v_pares jsonb; v_generaciones text[]:=ARRAY[]::text[];
BEGIN
    IF p IS NULL OR pg_column_size(p)>65536
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['esquema','operacion','material','sellos_hmac','referencias_candidatas']) IS NOT TRUE
       OR p->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.preparar-cancelacion.v1'
       OR p->>'operacion' IS DISTINCT FROM 'cancelar_expediente'
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'referencias_candidatas',ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(p->'referencias_candidatas') c WHERE jsonb_typeof(c.value)<>'string'
                  OR NOT vec_contratacion_temporal.referencia_valida_ct122(c.value #>> '{}'))
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'sellos_hmac',ARRAY['activo','retenidos']) IS NOT TRUE
       OR jsonb_typeof(p#>'{sellos_hmac,retenidos}') IS DISTINCT FROM 'array'
       OR jsonb_array_length(p#>'{sellos_hmac,retenidos}')>16 THEN
        RAISE EXCEPTION 'CT122: entrada de preparación inválida' USING ERRCODE='22023';
    END IF;
    v_pares:=jsonb_build_array(p#>'{sellos_hmac,activo}')||(p#>'{sellos_hmac,retenidos}');
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_par,ARRAY['ambito_hmac','generacion','huella_peticion_hmac']) IS NOT TRUE
           OR EXISTS (SELECT 1 FROM jsonb_each(v_par) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='generacion' THEN 'number' ELSE 'string' END)
           OR v_par->>'generacion' !~ '^[1-9][0-9]{0,8}$' OR v_par->>'generacion'=ANY(v_generaciones)
           OR v_par->>'ambito_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]ambito/v'||(v_par->>'generacion')||':[0-9a-f]{64}$')
           OR v_par->>'huella_peticion_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]peticion/v'||(v_par->>'generacion')||':[0-9a-f]{64}$') THEN
            RAISE EXCEPTION 'CT122: dominio o generación HMAC inválidos' USING ERRCODE='22023';
        END IF;
        v_generaciones:=array_append(v_generaciones,v_par->>'generacion');
    END LOOP;
    RETURN v_pares;
END
$$;

-- Entrada de confirmación: política y autorización ligadas al material.
CREATE FUNCTION vec_contratacion_temporal.validar_confirmacion_ct122(p jsonb, p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE pol jsonb:=p->'politica'; a jsonb:=p->'autorizacion'; m jsonb:=p->'material';
BEGIN
    IF p IS NULL OR pg_column_size(p)>3145728
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['esquema','operacion','material','referencias','ambito_idempotencia_hmac',
            'huella_peticion_hmac','expediente_anterior','expediente_siguiente','actuacion','politica','autorizacion','instante_efecto','contexto']) IS NOT TRUE
       OR p->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.confirmar-cancelacion.v1'
       OR p->>'operacion' IS DISTINCT FROM 'cancelar_expediente'
       OR EXISTS (SELECT 1 FROM jsonb_each(p) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('material','referencias',
            'expediente_anterior','expediente_siguiente','actuacion','politica','autorizacion','contexto') THEN 'object' ELSE 'string' END)
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'referencias',ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(p->'referencias') c WHERE jsonb_typeof(c.value)<>'string'
                  OR NOT vec_contratacion_temporal.referencia_valida_ct122(c.value #>> '{}'))
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'contexto',ARRAY['ambitos','atributos']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(pol,ARRAY['definicion_ref','definicion_version','definicion_huella_sha256',
            'accion','finalidad','evaluada_en','valida_hasta']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(pol) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='definicion_version' THEN 'number' ELSE 'string' END)
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(a,ARRAY['accion','contexto_recurso_huella_sha256','decision_canonica_hex',
            'decision_huella_sha256','decision_ref','finalidad','motivo_canonico_hex','perfil_activo_ref','perfil_version','persona_version','principal_id','recurso_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(a) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('persona_version','perfil_version') THEN 'number' ELSE 'string' END)
       OR p->>'ambito_idempotencia_hmac' !~ '^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR p->>'huella_peticion_hmac' !~ '^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR NOT vec_contratacion_temporal.referencia_valida_ct122(pol->>'definicion_ref')
       OR pol->>'definicion_version' !~ '^[1-9][0-9]{0,15}$' OR pol->>'definicion_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR pol->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
       OR pol->>'finalidad' IS DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
       OR p->>'instante_efecto' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'evaluada_en' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'valida_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'CT122: entrada de confirmación inválida' USING ERRCODE='22023';
    END IF;
    IF p_decision IS NULL OR p_motivo IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
       OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
       OR a->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
       OR a->>'finalidad' IS DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
       OR a->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR a->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR a->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR a->>'decision_canonica_hex' IS DISTINCT FROM encode(p_decision,'hex')
       OR a->>'motivo_canonico_hex' IS DISTINCT FROM encode(p_motivo,'hex')
       OR (a->>'persona_version')::numeric IS DISTINCT FROM p_persona_version
       OR (a->>'perfil_version')::numeric IS DISTINCT FROM p_perfil_version
       OR a->>'decision_huella_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
        RAISE EXCEPTION 'CT122: autorización divergente' USING ERRCODE='42501';
    END IF;
END
$$;

-- Admisión del expediente vigente: el orden de las comprobaciones fija el
-- rechazo que se devuelve. Nunca escribe.
CREATE FUNCTION vec_contratacion_temporal.admision_cancelacion_ct122(p_agregado jsonb, p_version numeric, m jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT CASE
        WHEN p_agregado IS NULL OR p_version IS DISTINCT FROM (m->>'version_esperada')::numeric
             OR p_agregado->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
             OR jsonb_typeof(p_agregado->'actuaciones') IS DISTINCT FROM 'array'
             OR jsonb_array_length(p_agregado->'actuaciones')<1 THEN 'version_en_conflicto'
        WHEN p_agregado ? 'fiscalizacion' OR EXISTS (SELECT 1 FROM jsonb_array_elements(p_agregado->'actuaciones') x
             WHERE x->>'accion_clave'='contratacion_temporal.fiscalizacion.registrar') THEN 'tras_fiscalizacion'
        WHEN p_agregado->>'estado_actual' IS DISTINCT FROM 'en_curso'
             OR NOT (m->'fases_admitidas') @> jsonb_build_array(p_agregado->>'fase_actual')
             OR NOT vec_contratacion_temporal.referencia_valida_ct122(p_agregado->'actuaciones'->-1->>'unidad_ref')
             OR (m->>'canal'='centro' AND NOT vec_contratacion_temporal.referencia_valida_ct122(p_agregado#>>'{solicitud,centro_ref}')) THEN 'fase_no_admitida'
        ELSE 'admitida' END
$$;

CREATE TABLE vec_contratacion_temporal.cancelacion_expediente_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cancelacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL UNIQUE,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    canal text NOT NULL CHECK (canal IN ('rrhh','centro')),
    centro_ref text CHECK ((canal='centro')=(centro_ref IS NOT NULL)),
    motivo_clave text NOT NULL CHECK (motivo_clave ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    fases_admitidas text[] NOT NULL CHECK (cardinality(fases_admitidas) BETWEEN 1 AND 16),
    fase_previa text NOT NULL CHECK (fase_previa=ANY(fases_admitidas)),
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct122(observaciones,2000,true)),
    estado text NOT NULL CHECK (estado='confirmada'),
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    expediente_anterior_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_anterior_json)='object'),
    expediente_siguiente_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_siguiente_json)='object'),
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
    politica_ref text NOT NULL,
    politica_version numeric(20,0) NOT NULL CHECK (politica_version BETWEEN 1 AND 9007199254740991),
    politica_huella_sha256 text NOT NULL CHECK (politica_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    confirmada_en timestamptz(6) NOT NULL CHECK (confirmada_en>=registrada_en),
    FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral
);

-- RLS: el propietario solo ve las filas de la operación en curso, fijada por
-- la función que la ejecuta; la sesión debe ser el ejecutor de CT.
ALTER TABLE vec_contratacion_temporal.cancelacion_expediente_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.cancelacion_expediente_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_operacion_ct122 ON vec_contratacion_temporal.cancelacion_expediente_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    organizacion_ref=current_setting('vec.ct122.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct122.expediente_ref',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY escritura_operacion_ct122 ON vec_contratacion_temporal.cancelacion_expediente_v1 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
    organizacion_ref=current_setting('vec.ct122.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct122.expediente_ref',true)
    AND actor_ref=current_setting('vec.ct122.actor_ref',true)
    AND perfil_ref=current_setting('vec.ct122.perfil_ref',true)
    AND ambito_hmac=current_setting('vec.ct122.ambito_hmac',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE TRIGGER cancelacion_expediente_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.cancelacion_expediente_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.resultado_cancelacion_ct122(r vec_contratacion_temporal.cancelacion_expediente_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-cancelacion.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,
            'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'canal',r.canal,
            'motivo_clave',r.motivo_clave,'fases_admitidas',to_jsonb(r.fases_admitidas),'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

-- La misma intención ya confirmada: coincide campo a campo.
CREATE FUNCTION vec_contratacion_temporal.misma_intencion_ct122(r vec_contratacion_temporal.cancelacion_expediente_v1, p_huella text, m jsonb)
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT r.huella_peticion_hmac IS NOT DISTINCT FROM p_huella AND r.organizacion_ref=m->>'organizacion_ref'
       AND r.expediente_ref=m->>'expediente_ref' AND r.version_esperada=(m->>'version_esperada')::numeric
       AND r.actor_ref=m->>'actor_ref' AND r.perfil_ref=m->>'perfil_ref' AND r.canal=m->>'canal'
       AND r.motivo_clave=m->>'motivo_clave' AND to_jsonb(r.fases_admitidas)=m->'fases_admitidas'
       AND r.observaciones=m->>'observaciones'
$$;

-- ============================================================ PREPARACIÓN
CREATE FUNCTION vec_contratacion_temporal.preparar_cancelacion_expediente_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_actual record; v_admision text;
    r vec_contratacion_temporal.cancelacion_expediente_v1%ROWTYPE;
    e text:='vec.contratacion-temporal.resultado-cancelacion.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct122(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct122(p_operacion);
    PERFORM vec_contratacion_temporal.validar_material_cancelacion_ct122(m);
    PERFORM set_config('vec.ct122.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct122.expediente_ref',m->>'expediente_ref',true);
    -- Recuperación: la misma intención con cualquier generación HMAC vigente.
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF NOT vec_contratacion_temporal.misma_intencion_ct122(r,v_par->>'huella_peticion_hmac',m) THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_cancelacion_ct122(r);
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cancelacion_existente');
    END IF;
    SELECT a.version, vec_contratacion_temporal.agregado_dominio_ct122(v.agregado_json,a.version) AS agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    v_admision:=vec_contratacion_temporal.admision_cancelacion_ct122(v_actual.agregado_json,v_actual.version,m);
    IF v_admision<>'admitida' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',v_admision);
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas',
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT122: entrada de cancelación inválida' USING ERRCODE='22023';
END
$funcion$;

-- ============================================================ CONFIRMACIÓN
CREATE FUNCTION vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-cancelacion.v1';
    r vec_contratacion_temporal.cancelacion_expediente_v1%ROWTYPE;
    v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_dominio jsonb; v_admision text; v_fase text; v_centro text;
    v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
    v_recibo jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct122(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct122(p_operacion,p_decision,p_motivo,p_persona_version,p_perfil_version);
    PERFORM vec_contratacion_temporal.validar_material_cancelacion_ct122(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT122: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct122.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct122.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct122.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct122.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct122.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        -- Una petición nueva recupera por la preparación; aquí solo se admite
        -- la repetición directa con la misma evidencia ya consumida.
        IF NOT vec_contratacion_temporal.misma_intencion_ct122(r,p_operacion->>'huella_peticion_hmac',m) THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT122: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cancelacion_existente');
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    v_dominio:=vec_contratacion_temporal.agregado_dominio_ct122(v_actual.agregado_json,v_actual.version);
    v_admision:=vec_contratacion_temporal.admision_cancelacion_ct122(v_dominio,v_actual.version,m);
    IF v_admision<>'admitida' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',v_admision);
    END IF;
    IF v_dominio IS DISTINCT FROM p_operacion->'expediente_anterior' THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    v_fase:=v_dominio->>'fase_actual';
    v_secuencia_actuacion:=jsonb_array_length(v_dominio->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.expediente.cancelar','actor_ref',m->>'actor_ref',
        'unidad_ref',v_dominio->'actuaciones'->-1->>'unidad_ref','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen',v_fase,'fase_destino',v_fase,
        'estado_origen','en_curso','estado_destino','cancelado');
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_dominio||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
        'estado_actual','cancelado','actuaciones',(v_dominio->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_dominio->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT122: proyección de cancelación divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa',v_fase,'estado_previo','en_curso');
    IF m->>'canal'='centro' THEN
        v_centro:=v_dominio#>>'{solicitud,centro_ref}';
        v_ambitos:=v_ambitos||jsonb_build_object('centro_ref',v_centro);
    END IF;
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'canal',m->>'canal','motivo_clave',m->>'motivo_clave',
        'fases_admitidas',vec_contratacion_temporal.fases_texto_ct122(m),
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT122: contexto de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct122(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'cancelacion_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT122: contexto autorizado de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT122: vigencia de cancelación agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT122: consumo de cancelación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT122: vigencia final de cancelación agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-CANCELACION-CT122'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,v_fase,'cancelado',
        'cancelacion_expediente_ct122',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT122: CAS final de cancelación perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-CANCELACION-CT122'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT122: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento de dominio `ct.expediente_cancelado.v1`: solo referencias opacas,
    -- claves y fechas. No hay llamamiento previo a la fiscalización: no lleva
    -- efecto para Bolsa.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.expediente-cancelado.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'fase_previa',v_fase,'canal',m->>'canal',
        'motivo_clave',m->>'motivo_clave','recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.expediente_cancelado.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','cancelar_expediente','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante',v_fase,'estado_resultante','cancelado',
        'motivo_clave',m->>'motivo_clave','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.cancelacion_expediente_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,canal,centro_ref,motivo_clave,
        fases_admitidas,fase_previa,observaciones,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,
        recibo_json,decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,
        registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'canal',v_centro,m->>'motivo_clave',
        ARRAY(SELECT x #>> '{}' FROM jsonb_array_elements(m->'fases_admitidas') WITH ORDINALITY y(x,o) ORDER BY o),v_fase,
        m->>'observaciones','confirmada',refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_dominio,v_siguiente,
        v_recibo,v_consumo.decision_ref,a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,
        pol->>'definicion_ref',(pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='cancelacion_expediente_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='cancelacion_expediente_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'cancelacion_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT122: entrada de cancelación inválida' USING ERRCODE='22023';
END
$funcion$;

-- ============================================================ CONSULTA
-- Cancelación registrada de un expediente: motivo, canal, fase previa,
-- observación y recibo. La composición solo la invoca tras acreditar la
-- lectura del expediente exacto.
CREATE FUNCTION vec_contratacion_temporal.consultar_cancelacion_expediente_v1(p_organizacion text, p_expediente text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE r vec_contratacion_temporal.cancelacion_expediente_v1%ROWTYPE;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct122(true);
    IF NOT vec_contratacion_temporal.referencia_valida_ct122(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct122(p_expediente) THEN
        RAISE EXCEPTION 'CT122: consulta de cancelación inválida' USING ERRCODE='22023';
    END IF;
    PERFORM set_config('vec.ct122.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct122.expediente_ref',p_expediente,true);
    SELECT * INTO r FROM vec_contratacion_temporal.cancelacion_expediente_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.cancelacion-expediente.v1','expediente_ref',p_expediente,
        'cancelacion',CASE WHEN r.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('canal',r.canal,'motivo_clave',r.motivo_clave,
            'fase_previa',r.fase_previa,'observaciones',r.observaciones,'recibo_ref',r.recibo_ref,
            'registrada_en',r.recibo_json->'registrada_en') END);
END
$funcion$;

-- ACL: solo el ejecutor de CT invoca las fachadas; tabla y auxiliares
-- quedan cerrados, también frente a privilegios por defecto.
DO $acl$
DECLARE v record; f regprocedure; destinatario text;
    fachadas regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.preparar_cancelacion_expediente_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_cancelacion_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.consultar_cancelacion_expediente_v1(text,text)'::regprocedure];
    auxiliares regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.texto_valido_ct122(text,integer,boolean)'::regprocedure,
      'vec_contratacion_temporal.referencia_valida_ct122(text)'::regprocedure,
      'vec_contratacion_temporal.mapa_go_ct122(jsonb)'::regprocedure,
      'vec_contratacion_temporal.huella_contexto_go_ct122(jsonb,jsonb)'::regprocedure,
      'vec_contratacion_temporal.fases_texto_ct122(jsonb)'::regprocedure,
      'vec_contratacion_temporal.agregado_dominio_ct122(jsonb,numeric)'::regprocedure,
      'vec_contratacion_temporal.exigir_sesion_ct122(boolean)'::regprocedure,
      'vec_contratacion_temporal.validar_material_cancelacion_ct122(jsonb)'::regprocedure,
      'vec_contratacion_temporal.validar_preparacion_ct122(jsonb)'::regprocedure,
      'vec_contratacion_temporal.validar_confirmacion_ct122(jsonb,bytea,bytea,numeric,numeric)'::regprocedure,
      'vec_contratacion_temporal.admision_cancelacion_ct122(jsonb,numeric,jsonb)'::regprocedure,
      'vec_contratacion_temporal.resultado_cancelacion_ct122(vec_contratacion_temporal.cancelacion_expediente_v1)'::regprocedure,
      'vec_contratacion_temporal.misma_intencion_ct122(vec_contratacion_temporal.cancelacion_expediente_v1,text,jsonb)'::regprocedure];
BEGIN
    FOR v IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
              WHERE c.oid='vec_contratacion_temporal.cancelacion_expediente_v1'::regclass AND x.grantee<>c.relowner LOOP
        destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.cancelacion_expediente_v1 FROM %s',destinatario);
    END LOOP;
    REVOKE ALL ON TABLE vec_contratacion_temporal.cancelacion_expediente_v1 FROM PUBLIC;
    FOREACH f IN ARRAY fachadas||auxiliares LOOP
        FOR v IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                  WHERE p.oid=f AND x.grantee<>p.proowner LOOP
            destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,destinatario);
        END LOOP;
    END LOOP;
    FOREACH f IN ARRAY fachadas LOOP
        EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f::text);
    END LOOP;
    -- Comprobación efectiva.
    IF EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
                WHERE c.oid='vec_contratacion_temporal.cancelacion_expediente_v1'::regclass
                  AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                   WHERE p.oid=ANY(fachadas||auxiliares)
                     AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole
                          OR (x.grantee<>p.proowner AND NOT (p.oid=ANY(fachadas) AND x.grantee='vec_contratacion_temporal_ejecutor'::regrole
                              AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))))
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.cancelacion_expediente_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR EXISTS (SELECT 1 FROM unnest(auxiliares) x WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) x WHERE NOT has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) x WHERE (SELECT NOT prosecdef FROM pg_proc WHERE oid=x)) THEN
        RAISE EXCEPTION 'CT122: ACL efectiva incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMENT ON TABLE vec_contratacion_temporal.cancelacion_expediente_v1 IS
    'CT122: cancelación del expediente antes de la fiscalización; motivo del catálogo, canal (centro o RRHH), fases admitidas por la regla c20 y observación.';
COMMIT;
