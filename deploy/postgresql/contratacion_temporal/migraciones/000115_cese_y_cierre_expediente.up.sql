\set ON_ERROR_STOP on
-- CT115: cese del nombramiento o contrato y cierre del expediente (petición
-- RRHH p. 1, punto 3; dudas 11 y 12).
--  * El cese se registra sobre un expediente en nombramiento con incorporación
--    acreditada (CT75): causa del catálogo gobernado, fecha de efecto y
--    justificante (tipo, referencia y huella; nunca el contenido). Añade una
--    versión del expediente y una actuación, consume la autorización AD3-82 y
--    publica `ct.cese.v1` en el outbox del expediente, todo en una transacción.
--  * El cierre exige el cese registrado y, si la regla del catálogo lo pide,
--    la ficha de GINPIX confirmada con su número. Deja el expediente en
--    `completado` (terminal) con su propia actuación, autorización y evento.
--  * La lectura de CT113 hacia Bolsa (`leer_contratos_bolsa_v1`) publica también
--    el cese con el mismo evento que la incorporación (tipo `cese`).
-- Historia de solo adición: ninguna fila anterior se modifica.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000115',0));

DO $prevalidacion$
DECLARE v_origen text; t text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION 'CT115: propietario incompatible' USING ERRCODE='55000';
    END IF;
    FOREACH t IN ARRAY ARRAY['expediente_version_integral','expediente_integral_actual','actuacion_expediente_integral',
        'outbox_expediente_integral','control_cadenas_expediente_integral','incorporacion_registro_v2','propuesta_formalizacion'] LOOP
        IF to_regclass('vec_contratacion_temporal.'||t) IS NULL THEN
            RAISE EXCEPTION 'CT115: falta %', t USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF to_regprocedure('vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[])') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
       OR to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.cierre_expediente_v1') IS NOT NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_expediente_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'CT115: dependencias incompatibles (AD3-82 requerida)' USING ERRCODE='55000';
    END IF;
    -- CT113 publica a Bolsa las incorporaciones; CT115 añade los ceses a esa
    -- misma lectura. Se exige su versión exacta para no pisar otra.
    IF to_regprocedure('vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz)') IS NULL
       OR (SELECT md5(prosrc) FROM pg_proc WHERE oid=to_regprocedure('vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)'))
          IS DISTINCT FROM 'ecb89f46ebaf84a53bf52ee7ea4e2b93' THEN
        RAISE EXCEPTION 'CT115: dependencias incompatibles (CT113 exacta requerida)' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF strpos(v_origen,'''subsanacion_reparos_v1''::text')=0 OR right(v_origen,4)<>'])))'
       OR strpos(v_origen,'cese_nombramiento_ct115')<>0 OR strpos(v_origen,'cierre_expediente_ct115')<>0 THEN
        RAISE EXCEPTION 'CT115: preimagen de origen de versión incompatible' USING ERRCODE='55000';
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
        ||left(v_origen,length(v_origen)-4)||', ''cese_nombramiento_ct115''::text, ''cierre_expediente_ct115''::text])))';
END
$origen$;

-- Texto libre acotado: sin espacios de borde, NFC y sin controles salvo
-- tabulador y salto de línea.
CREATE FUNCTION vec_contratacion_temporal.texto_valido_ct115(p text, p_maximo integer, p_vacio boolean)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT p IS NOT NULL AND (p='' AND p_vacio OR p<>'' AND char_length(p)<=p_maximo AND octet_length(p)<=p_maximo*4
       AND p=btrim(p,E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(8232)||chr(8233)||chr(12288))
       AND p=normalize(p,NFC) AND translate(p,E'\t\n','') !~ '[[:cntrl:]]')
$$;

-- Serialización de un mapa plano con claves ordenadas, idéntica a la de Go
-- para los valores que admite esta migración (ASCII sin escapes especiales).
CREATE FUNCTION vec_contratacion_temporal.mapa_go_ct115(p jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
DECLARE v text;
BEGIN
    IF jsonb_typeof(p)<>'object' OR EXISTS (SELECT 1 FROM jsonb_each(p) e
        WHERE jsonb_typeof(e.value)<>'string' OR e.key !~ '^[a-z][a-z0-9_]{0,63}$'
           OR e.value #>> '{}' !~ '^[A-Za-z0-9._:/#, _-]*$') THEN
        RAISE EXCEPTION 'CT115: mapa de contexto no admitido' USING ERRCODE='22023';
    END IF;
    SELECT '{'||coalesce(string_agg('"'||e.key||'":"'||(e.value #>> '{}')||'"',',' ORDER BY e.key COLLATE "C"),'')||'}'
      INTO v FROM jsonb_each(p) e;
    RETURN v;
END
$$;

CREATE FUNCTION vec_contratacion_temporal.huella_contexto_go_ct115(p_ambitos jsonb, p_atributos jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
    SELECT encode(sha256(convert_to('{"ambitos":'||vec_contratacion_temporal.mapa_go_ct115(p_ambitos)
        ||',"atributos":'||vec_contratacion_temporal.mapa_go_ct115(p_atributos)||'}','UTF8')),'hex')
$$;

-- Fecha civil de inicio de la incorporación acreditada (hora peninsular).
CREATE FUNCTION vec_contratacion_temporal.inicio_incorporacion_ct115(p_material jsonb)
RETURNS date LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $$
    SELECT ((p_material #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz AT TIME ZONE 'Europe/Madrid')::date
$$;

CREATE TABLE vec_contratacion_temporal.cese_nombramiento_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cese[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cese[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL UNIQUE,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    causa_clave text NOT NULL CHECK (causa_clave ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    fecha_efecto date NOT NULL CHECK (isfinite(fecha_efecto)),
    justificante_tipo text NOT NULL CHECK (justificante_tipo ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    justificante_ref text NOT NULL CHECK (justificante_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    justificante_sha256 text NOT NULL CHECK (justificante_sha256 ~ '^[0-9a-f]{64}$' AND justificante_sha256<>repeat('0',64)),
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct115(observaciones,2000,true)),
    incorporacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref),
    inicio_incorporacion date NOT NULL CHECK (fecha_efecto>=inicio_incorporacion),
    llamamiento_ref text,
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

CREATE TABLE vec_contratacion_temporal.cierre_expediente_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cierre-expediente[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]cierre-expediente[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.cese_nombramiento_v1(expediente_ref),
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    condiciones text[] NOT NULL CHECK (cardinality(condiciones) BETWEEN 1 AND 2 AND 'cese_registrado'=ANY(condiciones)
        AND condiciones <@ ARRAY['cese_registrado','ginpix_confirmado']::text[]),
    ginpix_numero text NOT NULL CHECK (ginpix_numero='' OR ginpix_numero ~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$'),
    ginpix_confirmada_en date,
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct115(observaciones,2000,true)),
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
    CHECK (('ginpix_confirmado'=ANY(condiciones)) = (ginpix_numero<>'' AND ginpix_confirmada_en IS NOT NULL)),
    CHECK (ginpix_numero<>'' OR ginpix_confirmada_en IS NULL),
    CHECK (ginpix_confirmada_en IS NULL OR isfinite(ginpix_confirmada_en)),
    FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral
);

-- RLS: el propietario solo ve las filas de la operación en curso, fijada por
-- la función que la ejecuta; la sesión debe ser el ejecutor de CT.
DO $rls$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','cierre_expediente_v1'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format($p$CREATE POLICY lectura_operacion_ct115 ON vec_contratacion_temporal.%I FOR SELECT TO vec_contratacion_temporal_propietario USING (
            organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
            AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
            AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
            AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
            AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'))$p$,t);
        EXECUTE format($p$CREATE POLICY escritura_operacion_ct115 ON vec_contratacion_temporal.%I FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
            organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
            AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
            AND actor_ref=current_setting('vec.ct115.actor_ref',true)
            AND perfil_ref=current_setting('vec.ct115.perfil_ref',true)
            AND ambito_hmac=current_setting('vec.ct115.ambito_hmac',true)
            AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
            AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
            AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'))$p$,t);
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',t||'_inmutable',t);
    END LOOP;
END
$rls$;
-- La publicación a Bolsa lee todos los ceses, solo desde su función.
CREATE POLICY publicacion_bolsa_ct115 ON vec_contratacion_temporal.cese_nombramiento_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    current_setting('vec.ct115.publicacion_bolsa',true)='activa'
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));

-- Comprobaciones comunes de la operación y de la sesión.
CREATE FUNCTION vec_contratacion_temporal.exigir_sesion_ct115(p_lectura boolean)
RETURNS void LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>(CASE WHEN p_lectura THEN 'on' ELSE 'off' END) THEN
        RAISE EXCEPTION 'CT115: sesión no autorizada' USING ERRCODE='42501';
    END IF;
END
$$;

CREATE FUNCTION vec_contratacion_temporal.referencia_valida_ct115(p text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT p IS NOT NULL AND p ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
$$;

-- Material de cese: forma exacta y tipos. Devuelve la fecha de efecto.
CREATE FUNCTION vec_contratacion_temporal.validar_material_cese_ct115(m jsonb)
RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text; f date;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','causa_clave','fecha_efecto','justificante_tipo','justificante_ref','justificante_sha256','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='version_esperada' THEN 'number' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR m->>'causa_clave' !~ '^[a-z][a-z0-9_.-]{1,79}$' OR m->>'justificante_tipo' !~ '^[a-z][a-z0-9_.-]{1,79}$'
       OR m->>'justificante_sha256' !~ '^[0-9a-f]{64}$' OR m->>'justificante_sha256'=repeat('0',64)
       OR m->>'fecha_efecto' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR NOT vec_contratacion_temporal.texto_valido_ct115(m->>'observaciones',2000,true) THEN
        RAISE EXCEPTION 'CT115: material de cese inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref','justificante_ref'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
            RAISE EXCEPTION 'CT115: referencia de cese inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    f:=(m->>'fecha_efecto')::date;
    IF to_char(f,'YYYY-MM-DD')<>m->>'fecha_efecto' THEN
        RAISE EXCEPTION 'CT115: fecha de efecto inválida' USING ERRCODE='22023';
    END IF;
    RETURN f;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RAISE EXCEPTION 'CT115: fecha de efecto inválida' USING ERRCODE='22023';
END
$$;

CREATE FUNCTION vec_contratacion_temporal.validar_material_cierre_ct115(m jsonb)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text; f date;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','condiciones','ginpix_numero','ginpix_confirmada_en','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
            CASE WHEN c.key='version_esperada' THEN 'number' WHEN c.key='condiciones' THEN 'array' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR jsonb_array_length(m->'condiciones') NOT BETWEEN 1 AND 2
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(m->'condiciones') e WHERE jsonb_typeof(e) IS DISTINCT FROM 'string'
                  OR e #>> '{}' NOT IN ('cese_registrado','ginpix_confirmado'))
       OR NOT m->'condiciones' @> '["cese_registrado"]'::jsonb
       OR (SELECT count(DISTINCT e #>> '{}')<>count(*) FROM jsonb_array_elements(m->'condiciones') e)
       OR (SELECT array_agg(e #>> '{}' ORDER BY ord) IS DISTINCT FROM array_agg(e #>> '{}' ORDER BY e #>> '{}' COLLATE "C")
             FROM jsonb_array_elements(m->'condiciones') WITH ORDINALITY x(e,ord))
       OR NOT (m->>'ginpix_numero'='' OR m->>'ginpix_numero' ~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$')
       OR NOT (m->>'ginpix_confirmada_en'='' OR m->>'ginpix_confirmada_en' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
       OR (m->'condiciones' @> '["ginpix_confirmado"]'::jsonb) <> (m->>'ginpix_numero'<>'' AND m->>'ginpix_confirmada_en'<>'')
       OR (m->>'ginpix_numero'='') <> (m->>'ginpix_confirmada_en'='')
       OR NOT vec_contratacion_temporal.texto_valido_ct115(m->>'observaciones',2000,true) THEN
        RAISE EXCEPTION 'CT115: material de cierre inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
            RAISE EXCEPTION 'CT115: referencia de cierre inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF m->>'ginpix_confirmada_en'<>'' THEN
        f:=(m->>'ginpix_confirmada_en')::date;
        IF to_char(f,'YYYY-MM-DD')<>m->>'ginpix_confirmada_en' THEN
            RAISE EXCEPTION 'CT115: fecha de GINPIX inválida' USING ERRCODE='22023';
        END IF;
    END IF;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RAISE EXCEPTION 'CT115: fecha de GINPIX inválida' USING ERRCODE='22023';
END
$$;

-- Entrada común de preparación: esquema, sellos HMAC (activo y retenidos) y
-- referencias candidatas. Devuelve la lista de pares HMAC en orden.
CREATE FUNCTION vec_contratacion_temporal.validar_preparacion_ct115(p jsonb, p_esquema text, p_operacion text, p_dominio text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE v_par jsonb; v_pares jsonb; v_generaciones text[]:=ARRAY[]::text[];
BEGIN
    IF p IS NULL OR pg_column_size(p)>65536
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['esquema','operacion','material','sellos_hmac','referencias_candidatas']) IS NOT TRUE
       OR p->>'esquema' IS DISTINCT FROM p_esquema OR p->>'operacion' IS DISTINCT FROM p_operacion
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'referencias_candidatas',ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(p->'referencias_candidatas') c WHERE jsonb_typeof(c.value)<>'string'
                  OR NOT vec_contratacion_temporal.referencia_valida_ct115(c.value #>> '{}'))
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'sellos_hmac',ARRAY['activo','retenidos']) IS NOT TRUE
       OR jsonb_typeof(p#>'{sellos_hmac,retenidos}') IS DISTINCT FROM 'array'
       OR jsonb_array_length(p#>'{sellos_hmac,retenidos}')>16 THEN
        RAISE EXCEPTION 'CT115: entrada de preparación inválida' USING ERRCODE='22023';
    END IF;
    v_pares:=jsonb_build_array(p#>'{sellos_hmac,activo}')||(p#>'{sellos_hmac,retenidos}');
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_par,ARRAY['ambito_hmac','generacion','huella_peticion_hmac']) IS NOT TRUE
           OR EXISTS (SELECT 1 FROM jsonb_each(v_par) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='generacion' THEN 'number' ELSE 'string' END)
           OR v_par->>'generacion' !~ '^[1-9][0-9]{0,8}$' OR v_par->>'generacion'=ANY(v_generaciones)
           OR v_par->>'ambito_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]'||p_dominio||'[.]ambito/v'||(v_par->>'generacion')||':[0-9a-f]{64}$')
           OR v_par->>'huella_peticion_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]'||p_dominio||'[.]peticion/v'||(v_par->>'generacion')||':[0-9a-f]{64}$') THEN
            RAISE EXCEPTION 'CT115: dominio o generación HMAC inválidos' USING ERRCODE='22023';
        END IF;
        v_generaciones:=array_append(v_generaciones,v_par->>'generacion');
    END LOOP;
    RETURN v_pares;
END
$$;

-- Entrada común de confirmación: política y autorización.
CREATE FUNCTION vec_contratacion_temporal.validar_confirmacion_ct115(p jsonb, p_esquema text, p_operacion text, p_dominio text,
    p_accion text, p_finalidad text, p_decision bytea, p_motivo bytea, p_persona_version numeric, p_perfil_version numeric)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE pol jsonb:=p->'politica'; a jsonb:=p->'autorizacion'; m jsonb:=p->'material';
BEGIN
    IF p IS NULL OR pg_column_size(p)>3145728
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p,ARRAY['esquema','operacion','material','referencias','ambito_idempotencia_hmac',
            'huella_peticion_hmac','expediente_anterior','expediente_siguiente','actuacion','politica','autorizacion','instante_efecto','contexto']) IS NOT TRUE
       OR p->>'esquema' IS DISTINCT FROM p_esquema OR p->>'operacion' IS DISTINCT FROM p_operacion
       OR EXISTS (SELECT 1 FROM jsonb_each(p) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('material','referencias',
            'expediente_anterior','expediente_siguiente','actuacion','politica','autorizacion','contexto') THEN 'object' ELSE 'string' END)
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'referencias',ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(p->'referencias') c WHERE jsonb_typeof(c.value)<>'string'
                  OR NOT vec_contratacion_temporal.referencia_valida_ct115(c.value #>> '{}'))
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p->'contexto',ARRAY['ambitos','atributos']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(pol,ARRAY['definicion_ref','definicion_version','definicion_huella_sha256',
            'accion','finalidad','evaluada_en','valida_hasta']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(pol) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='definicion_version' THEN 'number' ELSE 'string' END)
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(a,ARRAY['accion','contexto_recurso_huella_sha256','decision_canonica_hex',
            'decision_huella_sha256','decision_ref','finalidad','motivo_canonico_hex','perfil_activo_ref','perfil_version','persona_version','principal_id','recurso_ref']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(a) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('persona_version','perfil_version') THEN 'number' ELSE 'string' END)
       OR p->>'ambito_idempotencia_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]'||p_dominio||'[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
       OR p->>'huella_peticion_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]'||p_dominio||'[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(pol->>'definicion_ref')
       OR pol->>'definicion_version' !~ '^[1-9][0-9]{0,15}$' OR pol->>'definicion_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR pol->>'accion' IS DISTINCT FROM p_accion OR pol->>'finalidad' IS DISTINCT FROM p_finalidad
       OR p->>'instante_efecto' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'evaluada_en' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'valida_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'CT115: entrada de confirmación inválida' USING ERRCODE='22023';
    END IF;
    IF p_decision IS NULL OR p_motivo IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
       OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
       OR a->>'accion' IS DISTINCT FROM p_accion OR a->>'finalidad' IS DISTINCT FROM p_finalidad
       OR a->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR a->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR a->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR a->>'decision_canonica_hex' IS DISTINCT FROM encode(p_decision,'hex')
       OR a->>'motivo_canonico_hex' IS DISTINCT FROM encode(p_motivo,'hex')
       OR (a->>'persona_version')::numeric IS DISTINCT FROM p_persona_version
       OR (a->>'perfil_version')::numeric IS DISTINCT FROM p_perfil_version
       OR a->>'decision_huella_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
        RAISE EXCEPTION 'CT115: autorización divergente' USING ERRCODE='42501';
    END IF;
END
$$;

-- Incorporación acreditada más reciente del expediente y llamamiento de Bolsa
-- de su propuesta, si existe.
CREATE FUNCTION vec_contratacion_temporal.incorporacion_expediente_ct115(p_organizacion text, p_expediente text)
RETURNS TABLE(recibo_ref text, inicio date, llamamiento_ref text)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT r.recibo_ref, vec_contratacion_temporal.inicio_incorporacion_ct115(r.material_json),
           (SELECT pf.llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion pf
             WHERE pf.organizacion_ref=p_organizacion AND pf.expediente_ref=p_expediente
             ORDER BY pf.confirmada_en DESC, pf.propuesta_ref COLLATE "C" DESC LIMIT 1)
      FROM vec_contratacion_temporal.incorporacion_registro_v2 r
     WHERE r.organizacion_ref=p_organizacion AND r.expediente_ref=p_expediente
     ORDER BY r.registrada_en DESC, r.recibo_ref COLLATE "C" DESC LIMIT 1
$$;

CREATE FUNCTION vec_contratacion_temporal.resultado_cese_ct115(r vec_contratacion_temporal.cese_nombramiento_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-cese.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,
            'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'causa_clave',r.causa_clave,
            'fecha_efecto',to_char(r.fecha_efecto,'YYYY-MM-DD'),'justificante_tipo',r.justificante_tipo,'justificante_ref',r.justificante_ref,
            'justificante_sha256',r.justificante_sha256,'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'incorporacion',jsonb_build_object('recibo_ref',r.incorporacion_ref,'inicio',to_char(r.inicio_incorporacion,'YYYY-MM-DD')),
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

CREATE FUNCTION vec_contratacion_temporal.resultado_cierre_ct115(r vec_contratacion_temporal.cierre_expediente_v1, p_cese_recibo text)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-cierre-expediente.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,
            'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'condiciones',to_jsonb(r.condiciones),
            'ginpix_numero',r.ginpix_numero,'ginpix_confirmada_en',coalesce(to_char(r.ginpix_confirmada_en,'YYYY-MM-DD'),''),
            'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'cese_recibo_ref',p_cese_recibo,
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

-- ============================================================ CESE
CREATE FUNCTION vec_contratacion_temporal.preparar_cese_nombramiento_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_fecha date;
    r vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; v_actual record; v_inc record; v_existe boolean;
    e text:='vec.contratacion-temporal.resultado-cese.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,'vec.contratacion-temporal.preparar-cese.v1','registrar_cese','cese');
    v_fecha:=vec_contratacion_temporal.validar_material_cese_ct115(m);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    -- Recuperación: la misma intención con cualquier generación HMAC vigente.
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF r.huella_peticion_hmac IS DISTINCT FROM v_par->>'huella_peticion_hmac' OR r.organizacion_ref<>m->>'organizacion_ref'
               OR r.expediente_ref<>m->>'expediente_ref' OR r.version_esperada<>(m->>'version_esperada')::numeric
               OR r.actor_ref<>m->>'actor_ref' OR r.perfil_ref<>m->>'perfil_ref' OR r.causa_clave<>m->>'causa_clave'
               OR r.fecha_efecto<>v_fecha OR r.justificante_tipo<>m->>'justificante_tipo' OR r.justificante_ref<>m->>'justificante_ref'
               OR r.justificante_sha256<>m->>'justificante_sha256' OR r.observaciones<>m->>'observaciones' THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_cese_ct115(r);
        END IF;
    END LOOP;
    SELECT EXISTS (SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE expediente_ref=m->>'expediente_ref') INTO v_existe;
    IF v_existe THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cese_existente');
    END IF;
    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_actual.version<>(m->>'version_esperada')::numeric
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    SELECT * INTO v_inc FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND OR v_inc.inicio IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_incorporacion');
    END IF;
    IF v_fecha<v_inc.inicio THEN
        RETURN jsonb_build_object('esquema',e,'resultado','fecha_anterior_incorporacion');
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas',
        'incorporacion',jsonb_build_object('recibo_ref',v_inc.recibo_ref,'inicio',to_char(v_inc.inicio,'YYYY-MM-DD')),
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT115: entrada de cese inválida' USING ERRCODE='22023';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_cese_nombramiento_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-cese.v1';
    r vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE;
    v_fecha date; v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_inc record;
    v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
    v_recibo jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,'vec.contratacion-temporal.confirmar-cese.v1','registrar_cese','cese',
        'contratacion_temporal.seguimiento.cesar','registrar_cese_contratacion_temporal',p_decision,p_motivo,p_persona_version,p_perfil_version);
    v_fecha:=vec_contratacion_temporal.validar_material_cese_ct115(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT115: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct115.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct115.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct115.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        -- Una petición nueva recupera por la preparación; aquí solo se admite
        -- la repetición directa con la misma evidencia ya consumida.
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac' OR r.version_esperada<>v_version
           OR r.causa_clave<>m->>'causa_clave' OR r.fecha_efecto<>v_fecha OR r.justificante_ref<>m->>'justificante_ref'
           OR r.justificante_sha256<>m->>'justificante_sha256' OR r.observaciones<>m->>'observaciones' THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT115: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND OR v_actual.version<>v_version
       OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cese_existente');
    END IF;
    SELECT * INTO v_inc FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND OR v_inc.inicio IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_incorporacion');
    END IF;
    IF v_fecha<v_inc.inicio THEN
        RETURN jsonb_build_object('esquema',e,'resultado','fecha_anterior_incorporacion');
    END IF;
    v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.seguimiento.cesar','actor_ref',m->>'actor_ref',
        'unidad_ref',v_actual.agregado_json#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen','nombramiento','fase_destino','nombramiento',
        'estado_origen','en_curso','estado_destino','en_curso','documentos_ref',jsonb_build_array(m->>'justificante_ref'));
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT115: proyección de cese divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa','nombramiento','estado_previo','en_curso');
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'causa_clave',m->>'causa_clave','fecha_efecto',m->>'fecha_efecto',
        'justificante_tipo',m->>'justificante_tipo','justificante_ref',m->>'justificante_ref','justificante_sha256',m->>'justificante_sha256',
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'incorporacion_ref',v_inc.recibo_ref,'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT115: contexto de cese divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.seguimiento.cesar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'cese_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'registrar_cese_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT115: contexto autorizado de cese divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT115: vigencia de cese agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT115: consumo de cese divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT115: vigencia final de cese agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-CESE-CT115'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
        'cese_nombramiento_ct115',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT115: CAS final de cese perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-CESE-CT115'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT115: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento de dominio `ct.cese.v1`: solo referencias opacas, claves y fechas.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.cese.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'incorporacion_ref',v_inc.recibo_ref,
        'llamamiento_ref',v_inc.llamamiento_ref,'causa_clave',m->>'causa_clave','fecha_efecto',m->>'fecha_efecto',
        'recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.cese.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','registrar_cese','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante','nombramiento','estado_resultante','en_curso',
        'causa_clave',m->>'causa_clave','fecha_efecto',m->>'fecha_efecto','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.cese_nombramiento_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,causa_clave,fecha_efecto,
        justificante_tipo,justificante_ref,justificante_sha256,observaciones,incorporacion_ref,inicio_incorporacion,llamamiento_ref,estado,
        reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,decision_ref,decision_huella_sha256,
        consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'causa_clave',v_fecha,m->>'justificante_tipo',m->>'justificante_ref',m->>'justificante_sha256',
        m->>'observaciones',v_inc.recibo_ref,v_inc.inicio,v_inc.llamamiento_ref,'confirmada',refs->>'reserva_ref',refs->>'recibo_ref',
        refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,v_consumo.decision_ref,a->>'decision_huella_sha256',
        v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,pol->>'definicion_ref',(pol->>'definicion_version')::numeric,
        pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='cese_nombramiento_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='cese_nombramiento_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'cese_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT115: entrada de cese inválida' USING ERRCODE='22023';
END
$funcion$;

-- ============================================================ CIERRE
CREATE FUNCTION vec_contratacion_temporal.preparar_cierre_expediente_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_actual record; v_cese record;
    r vec_contratacion_temporal.cierre_expediente_v1%ROWTYPE;
    e text:='vec.contratacion-temporal.resultado-cierre-expediente.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,'vec.contratacion-temporal.preparar-cierre-expediente.v1','cerrar_expediente','cierre-expediente');
    PERFORM vec_contratacion_temporal.validar_material_cierre_ct115(m);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    SELECT c.recibo_ref INTO v_cese FROM vec_contratacion_temporal.cese_nombramiento_v1 c
     WHERE c.organizacion_ref=m->>'organizacion_ref' AND c.expediente_ref=m->>'expediente_ref';
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF r.huella_peticion_hmac IS DISTINCT FROM v_par->>'huella_peticion_hmac' OR r.organizacion_ref<>m->>'organizacion_ref'
               OR r.expediente_ref<>m->>'expediente_ref' OR r.version_esperada<>(m->>'version_esperada')::numeric
               OR r.actor_ref<>m->>'actor_ref' OR r.perfil_ref<>m->>'perfil_ref' OR to_jsonb(r.condiciones)<>m->'condiciones'
               OR r.ginpix_numero<>m->>'ginpix_numero' OR coalesce(to_char(r.ginpix_confirmada_en,'YYYY-MM-DD'),'')<>m->>'ginpix_confirmada_en'
               OR r.observaciones<>m->>'observaciones' THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_cierre_ct115(r,v_cese.recibo_ref);
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cierre_existente');
    END IF;
    IF v_cese.recibo_ref IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_cese');
    END IF;
    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_actual.version<>(m->>'version_esperada')::numeric
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas','cese_recibo_ref',v_cese.recibo_ref,
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT115: entrada de cierre inválida' USING ERRCODE='22023';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_cierre_expediente_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-cierre-expediente.v1';
    r vec_contratacion_temporal.cierre_expediente_v1%ROWTYPE; v_cese record;
    v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_decision jsonb; v_consumo record;
    v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text; v_secuencia_actuacion numeric;
    v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric; v_recibo jsonb; v_condiciones text[];
    v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,'vec.contratacion-temporal.confirmar-cierre-expediente.v1','cerrar_expediente',
        'cierre-expediente','contratacion_temporal.expediente.cerrar','cerrar_expediente_tras_cese',p_decision,p_motivo,p_persona_version,p_perfil_version);
    PERFORM vec_contratacion_temporal.validar_material_cierre_ct115(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT115: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    SELECT array_agg(x #>> '{}' ORDER BY ord) INTO v_condiciones FROM jsonb_array_elements(m->'condiciones') WITH ORDINALITY t(x,ord);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct115.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct115.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct115.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac' OR r.version_esperada<>v_version
           OR r.condiciones<>v_condiciones OR r.ginpix_numero<>m->>'ginpix_numero' OR r.observaciones<>m->>'observaciones' THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT115: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    SELECT c.recibo_ref, c.fecha_efecto INTO v_cese FROM vec_contratacion_temporal.cese_nombramiento_v1 c
     WHERE c.organizacion_ref=m->>'organizacion_ref' AND c.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_cese');
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND OR v_actual.version<>v_version
       OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.expediente.cerrar','actor_ref',m->>'actor_ref',
        'unidad_ref',v_actual.agregado_json#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen','nombramiento','fase_destino','nombramiento',
        'estado_origen','en_curso','estado_destino','completado');
    IF m->>'ginpix_numero'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('documentos_ref',jsonb_build_array('ginpix:'||(m->>'ginpix_numero')));
    END IF;
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,'estado_actual','completado',
        'actualizado_en',p_operacion->'instante_efecto','actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT115: proyección de cierre divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa','nombramiento','estado_previo','en_curso');
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'cese_recibo_ref',v_cese.recibo_ref,
        'condiciones',array_to_string(v_condiciones,','),'ginpix_numero',m->>'ginpix_numero','ginpix_confirmada_en',m->>'ginpix_confirmada_en',
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT115: contexto de cierre divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.cerrar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'cierre_expediente_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'cerrar_expediente_tras_cese'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT115: contexto autorizado de cierre divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT115: vigencia de cierre agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_expediente_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT115: consumo de cierre divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT115: vigencia final de cierre agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-CIERRE-CT115'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','completado',
        'cierre_expediente_ct115',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT115: CAS final de cierre perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-CIERRE-CT115'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT115: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.cierre-expediente.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'cese_recibo_ref',v_cese.recibo_ref,
        'condiciones',to_jsonb(v_condiciones),'recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.cierre_expediente.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','cerrar_expediente','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante','nombramiento','estado_resultante','completado',
        'cese_recibo_ref',v_cese.recibo_ref,'recibo_ref',refs->>'recibo_ref','auditoria_ref',v_consumo.auditoria_ref,
        'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref','registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.cierre_expediente_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,condiciones,ginpix_numero,
        ginpix_confirmada_en,observaciones,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,
        decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',v_condiciones,m->>'ginpix_numero',nullif(m->>'ginpix_confirmada_en','')::date,m->>'observaciones',
        'confirmada',refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,
        v_consumo.decision_ref,a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,
        pol->>'definicion_ref',(pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='cierre_expediente_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='cierre_expediente_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'cierre_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT115: entrada de cierre inválida' USING ERRCODE='22023';
END
$funcion$;

-- ============================================================ CONSULTA
-- Estado del cese y del cierre de un expediente para el detalle. La
-- composición solo la invoca tras acreditar la consulta V3 del mismo detalle.
CREATE FUNCTION vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(p_organizacion text, p_expediente text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
DECLARE c record; k record; i record;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT115: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT115: consulta inválida' USING ERRCODE='22023';
    END IF;
    PERFORM set_config('vec.ct115.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct115.expediente_ref',p_expediente,true);
    SELECT * INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO k FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_expediente_ct115(p_organizacion,p_expediente);
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.cese-cierre-expediente.v1',
        'expediente_ref',p_expediente,
        'incorporacion',CASE WHEN i.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo_ref',i.recibo_ref,'inicio',to_char(i.inicio,'YYYY-MM-DD')) END,
        'cese',CASE WHEN c.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('causa_clave',c.causa_clave,
            'fecha_efecto',to_char(c.fecha_efecto,'YYYY-MM-DD'),'justificante_tipo',c.justificante_tipo,'justificante_ref',c.justificante_ref,
            'justificante_sha256',c.justificante_sha256,'observaciones',c.observaciones,'recibo',c.recibo_json) END,
        'cierre',CASE WHEN k.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('condiciones',to_jsonb(k.condiciones),
            'ginpix_numero',k.ginpix_numero,'ginpix_confirmada_en',coalesce(to_char(k.ginpix_confirmada_en,'YYYY-MM-DD'),''),
            'observaciones',k.observaciones,'recibo',k.recibo_json) END);
END
$funcion$;

-- ============================================================ PUBLICACIÓN A BOLSA
-- CT113 publica las incorporaciones con el evento de integración
-- `vec.contratacion-temporal.contrato-bolsa.v1`. La misma lectura, con el
-- mismo cursor (creada_en, origen_ref) y la misma forma, añade ahora los
-- ceses: tipo `cese`, inicio de la incorporación, fin = fecha de efecto del
-- cese y su causa del catálogo. Bolsa los recibe por su inbox idempotente sin
-- otro relevo. Solo ceses de expedientes cubiertos por un llamamiento.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(
    p_desde_en timestamptz,
    p_desde_ref text,
    p_limite integer
) RETURNS TABLE(
    evento_ref text,
    evento jsonb,
    huella_sha256 text,
    origen_ref text,
    origen_creada_en timestamptz
)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
    IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
       OR (p_desde_en IS NULL) <> (p_desde_ref IS NULL)
       OR (p_desde_en IS NOT NULL AND NOT pg_catalog.isfinite(p_desde_en))
       OR pg_catalog.octet_length(p_desde_ref) > 512 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'lectura de contratos para Bolsa inválida';
    END IF;
    -- La política de lectura de ceses solo se abre dentro de esta función.
    PERFORM pg_catalog.set_config('vec.ct115.publicacion_bolsa', 'activa', true);
    RETURN QUERY
    WITH incorporaciones AS (
        SELECT o.outbox_ref AS origen, o.creada_en AS creada,
               'evento:ct:contrato-bolsa:' || pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to('incorporacion' || pg_catalog.chr(31) || o.outbox_ref, 'UTF8')
               ), 'hex') AS ref,
               pg_catalog.jsonb_build_object(
                   'esquema', 'vec.contratacion-temporal.contrato-bolsa.v1',
                   'tipo', 'incorporacion',
                   'origen_ref', o.outbox_ref,
                   'organizacion_ref', r.organizacion_ref,
                   'expediente_ref', r.expediente_ref,
                   'llamamiento_ref', p.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
                   'fin_previsto', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,hasta}')::timestamptz),
                   'modalidad_clave', e.agregado_json #>> '{analisis,modalidad_clave}',
                   'categoria_ref', e.agregado_json #>> '{analisis,categoria_ref}',
                   'causa_clave', e.agregado_json #>> '{analisis,causa_clave}',
                   'ocurrido_en', vec_contratacion_temporal.instante_contrato_bolsa_v1(r.registrada_en)
               ) AS cuerpo
          FROM vec_contratacion_temporal.incorporacion_outbox_v2 o
          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r
            ON r.recibo_ref = o.recibo_ref AND r.outbox_ref = o.outbox_ref
          JOIN vec_contratacion_temporal.propuesta_formalizacion p
            ON p.organizacion_ref = r.organizacion_ref AND p.expediente_ref = r.expediente_ref
          JOIN vec_contratacion_temporal.expediente_version_integral e
            ON e.expediente_ref = r.expediente_ref AND e.version = r.version_expediente
         WHERE p_desde_en IS NULL OR (o.creada_en, o.outbox_ref) > (p_desde_en, p_desde_ref)
         ORDER BY o.creada_en, o.outbox_ref
         LIMIT p_limite
    ), ceses AS (
        SELECT c.evento_ref AS origen, c.confirmada_en AS creada,
               'evento:ct:contrato-bolsa:' || pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to('cese' || pg_catalog.chr(31) || c.evento_ref, 'UTF8')
               ), 'hex') AS ref,
               pg_catalog.jsonb_build_object(
                   'esquema', 'vec.contratacion-temporal.contrato-bolsa.v1',
                   'tipo', 'cese',
                   'origen_ref', c.evento_ref,
                   'organizacion_ref', c.organizacion_ref,
                   'expediente_ref', c.expediente_ref,
                   'llamamiento_ref', c.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
                   'fin_previsto', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       c.fecha_efecto::timestamp AT TIME ZONE 'UTC'),
                   'modalidad_clave', c.expediente_siguiente_json #>> '{analisis,modalidad_clave}',
                   'categoria_ref', c.expediente_siguiente_json #>> '{analisis,categoria_ref}',
                   'causa_clave', c.causa_clave,
                   'ocurrido_en', vec_contratacion_temporal.instante_contrato_bolsa_v1(c.registrada_en)
               ) AS cuerpo
          FROM vec_contratacion_temporal.cese_nombramiento_v1 c
          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r ON r.recibo_ref = c.incorporacion_ref
         WHERE c.llamamiento_ref IS NOT NULL
           AND (p_desde_en IS NULL OR (c.confirmada_en, c.evento_ref) > (p_desde_en, p_desde_ref))
         ORDER BY c.confirmada_en, c.evento_ref
         LIMIT p_limite
    ), base AS (
        SELECT * FROM incorporaciones UNION ALL SELECT * FROM ceses
         ORDER BY 2, 1
         LIMIT p_limite
    )
    SELECT b.ref, b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               (b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref))::text, 'UTF8')), 'hex'),
           b.origen, b.creada
      FROM base b
     ORDER BY b.creada, b.origen;
END
$f$;
COMMENT ON FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer) IS
    'CT113+CT115: publica a Bolsa incorporaciones y ceses de expedientes cubiertos por llamamiento; solo referencias opacas, fechas y claves.';

-- ACL: solo el ejecutor de CT invoca las fachadas; tablas y auxiliares
-- quedan cerrados, también frente a privilegios por defecto.
DO $acl$
DECLARE v record; f regprocedure; t text; destinatario text;
    fachadas regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.preparar_cese_nombramiento_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_cese_nombramiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.preparar_cierre_expediente_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text)'::regprocedure];
    auxiliares regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.texto_valido_ct115(text,integer,boolean)'::regprocedure,
      'vec_contratacion_temporal.mapa_go_ct115(jsonb)'::regprocedure,
      'vec_contratacion_temporal.huella_contexto_go_ct115(jsonb,jsonb)'::regprocedure,
      'vec_contratacion_temporal.inicio_incorporacion_ct115(jsonb)'::regprocedure,
      'vec_contratacion_temporal.exigir_sesion_ct115(boolean)'::regprocedure,
      'vec_contratacion_temporal.referencia_valida_ct115(text)'::regprocedure,
      'vec_contratacion_temporal.validar_material_cese_ct115(jsonb)'::regprocedure,
      'vec_contratacion_temporal.validar_material_cierre_ct115(jsonb)'::regprocedure,
      'vec_contratacion_temporal.validar_preparacion_ct115(jsonb,text,text,text)'::regprocedure,
      'vec_contratacion_temporal.validar_confirmacion_ct115(jsonb,text,text,text,text,text,bytea,bytea,numeric,numeric)'::regprocedure,
      'vec_contratacion_temporal.incorporacion_expediente_ct115(text,text)'::regprocedure,
      'vec_contratacion_temporal.resultado_cese_ct115(vec_contratacion_temporal.cese_nombramiento_v1)'::regprocedure,
      'vec_contratacion_temporal.resultado_cierre_ct115(vec_contratacion_temporal.cierre_expediente_v1,text)'::regprocedure];
BEGIN
    FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','cierre_expediente_v1'] LOOP
        FOR v IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
                  WHERE c.oid=('vec_contratacion_temporal.'||t)::regclass AND x.grantee<>c.relowner LOOP
            destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
            EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %s',t,destinatario);
        END LOOP;
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',t);
    END LOOP;
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
                WHERE c.oid IN ('vec_contratacion_temporal.cese_nombramiento_v1'::regclass,'vec_contratacion_temporal.cierre_expediente_v1'::regclass)
                  AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                   WHERE p.oid=ANY(fachadas||auxiliares)
                     AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole
                          OR (x.grantee<>p.proowner AND NOT (p.oid=ANY(fachadas) AND x.grantee='vec_contratacion_temporal_ejecutor'::regrole
                              AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))))
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.cese_nombramiento_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.cierre_expediente_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR EXISTS (SELECT 1 FROM unnest(auxiliares) a WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',a,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) a WHERE NOT has_function_privilege('vec_contratacion_temporal_ejecutor',a,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) a WHERE (SELECT NOT prosecdef FROM pg_proc WHERE oid=a))
       OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)','EXECUTE')
       OR has_function_privilege('public','vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)','EXECUTE') THEN
        RAISE EXCEPTION 'CT115: ACL efectiva incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMENT ON TABLE vec_contratacion_temporal.cese_nombramiento_v1 IS
    'CT115: cese registrado de un nombramiento o contrato; causa del catálogo, fecha de efecto y justificante por referencia y huella.';
COMMENT ON TABLE vec_contratacion_temporal.cierre_expediente_v1 IS
    'CT115: cierre del expediente tras el cese, con las condiciones de la regla c10 y la confirmación de GINPIX.';
COMMIT;
