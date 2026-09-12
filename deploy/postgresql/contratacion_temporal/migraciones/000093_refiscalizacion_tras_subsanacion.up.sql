-- Fuente candidata: segunda fiscalización tras subsanación CT92; no aplicada.
-- Conserva API, material HMAC, recibo y consumidor AD3-10. Las funciones CT52
-- iniciales permanecen intactas. Instalación sólo por integrador tras doble revisión.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal.dependencias.subsanacion_reparos.v1',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:refiscalizacion_tras_subsanacion:v1',0));

DO $prevalidacion$
DECLARE v_def text; v_sha text;
BEGIN
    IF pg_catalog.to_regclass('vec_contratacion_temporal.reserva_subsanacion_reparos') IS NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_fiscalizacion_tras_subsanacion_v1(jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT pg_catalog.has_function_privilege(current_user,
          'vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'dependencias de nueva fiscalización incompatibles' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.pg_get_constraintdef(oid) INTO STRICT v_def
      FROM pg_catalog.pg_constraint
     WHERE conrelid='vec_contratacion_temporal.reserva_fiscalizacion'::regclass
       AND conname='reserva_fiscalizacion_version_expediente_check' AND contype='c' AND convalidated;
    IF v_def IS DISTINCT FROM 'CHECK ((version_expediente = (5)::numeric))' THEN
        RAISE EXCEPTION 'preimagen de reserva fiscalización incompatible' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.pg_get_constraintdef(oid) INTO STRICT v_def
      FROM pg_catalog.pg_constraint
     WHERE conrelid='vec_contratacion_temporal.retorno_fiscalizacion_unidad'::regclass
       AND conname='retorno_fiscalizacion_unidad_version_expediente_check' AND contype='c' AND convalidated;
    IF v_def IS DISTINCT FROM 'CHECK ((version_expediente = (6)::numeric))' THEN
        RAISE EXCEPTION 'preimagen de retorno fiscalización incompatible' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid='vec_contratacion_temporal.reserva_subsanacion_reparos'::regclass
           AND confrelid='vec_contratacion_temporal.retorno_fiscalizacion_unidad'::regclass
           AND conname='reserva_subsanacion_reparos_retorno_ref_fkey'
           AND contype='f' AND convalidated
           AND pg_catalog.pg_get_constraintdef(oid)='FOREIGN KEY (retorno_ref) REFERENCES vec_contratacion_temporal.retorno_fiscalizacion_unidad(retorno_ref)') THEN
        RAISE EXCEPTION 'ligadura CT92 de retorno incompatible' USING ERRCODE='55000';
    END IF;
    -- Preimagen de las funciones reutilizadas: no adaptar a ciegas una CT52 distinta.
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') INTO STRICT v_sha
      FROM pg_catalog.pg_proc WHERE oid='vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb)'::regprocedure;
    IF v_sha IS DISTINCT FROM '6e1ca30ad221cd836918bde23da8341283e3ed0586156b911cef0220af4262f9' THEN
        RAISE EXCEPTION 'preimagen preparar_fiscalizacion_v1 incompatible' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') INTO STRICT v_sha
      FROM pg_catalog.pg_proc WHERE oid='vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    IF v_sha IS DISTINCT FROM 'bb21608248394f30ef120618c26c5a753d93d0ba12fd98f69a6953ea9154aecf' THEN
        RAISE EXCEPTION 'preimagen confirmar_fiscalizacion_v1 incompatible' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') INTO STRICT v_sha
      FROM pg_catalog.pg_proc WHERE oid='vec_contratacion_temporal.recibo_fiscalizacion_v1(text)'::regprocedure;
    IF v_sha IS DISTINCT FROM 'e1c091fd3f04752d3e361580faa0d1507ebd995e61fbfefdb4cf010c73ededbb' THEN
        RAISE EXCEPTION 'preimagen recibo_fiscalizacion_v1 incompatible' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.reserva_fiscalizacion
    DROP CONSTRAINT reserva_fiscalizacion_version_expediente_check,
    ADD CONSTRAINT reserva_fiscalizacion_version_expediente_check
        CHECK (version_expediente=5 OR version_expediente BETWEEN 7 AND 9007199254740990);
ALTER TABLE vec_contratacion_temporal.retorno_fiscalizacion_unidad
    DROP CONSTRAINT retorno_fiscalizacion_unidad_version_expediente_check,
    ADD CONSTRAINT retorno_fiscalizacion_unidad_version_expediente_check
        CHECK (version_expediente=6 OR version_expediente BETWEEN 8 AND 9007199254740991);

-- Lectura interna del antecedente por su retorno exacto: fiscalizador y
-- subsanador pueden ser actores distintos. Sin permiso directo sobre la tabla.
CREATE POLICY lectura_antecedente_refiscalizacion
ON vec_contratacion_temporal.reserva_subsanacion_reparos
FOR SELECT TO vec_contratacion_temporal_propietario USING (
    organizacion_ref=pg_catalog.current_setting('vec.ct_refiscalizacion.organizacion_ref',true)
    AND expediente_ref=pg_catalog.current_setting('vec.ct_refiscalizacion.expediente_ref',true)
    AND retorno_ref=pg_catalog.current_setting('vec.ct_refiscalizacion.retorno_ref',true)
    AND pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));

CREATE FUNCTION vec_contratacion_temporal.antecedente_refiscalizacion_v1(p_expediente jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
DECLARE v_retorno text; v_subsanacion record; v_actuacion jsonb; v_version numeric;
BEGIN
    IF pg_catalog.jsonb_typeof(p_expediente) IS DISTINCT FROM 'object'
       OR p_expediente->>'fase_actual' IS DISTINCT FROM 'subsanacion_unidad'
       OR p_expediente->>'estado_actual' IS DISTINCT FROM 'incidencia'
       OR p_expediente#>>'{fiscalizacion,resultado}' IS DISTINCT FROM 'desfavorable'
       OR p_expediente#>>'{fiscalizacion,retorno,estado}' IS DISTINCT FROM 'pendiente'
       OR pg_catalog.jsonb_typeof(p_expediente->'actuaciones') IS DISTINCT FROM 'array'
       OR coalesce(p_expediente->>'version','') !~ '^[1-9][0-9]{0,15}$' THEN
        RAISE EXCEPTION 'origen de nueva fiscalización incompatible' USING ERRCODE='40001';
    END IF;
    v_version:=(p_expediente->>'version')::numeric;
    v_retorno:=p_expediente#>>'{fiscalizacion,retorno,retorno_ref}';
    IF v_version NOT BETWEEN 7 AND 9007199254740990
       OR coalesce(v_retorno,'') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(p_expediente->'actuaciones') x
            WHERE x->>'accion_clave'='contratacion_temporal.subsanacion_reparos.registrar'
              AND x->>'retorno_ref'=v_retorno)<>1 THEN
        RAISE EXCEPTION 'subsanación vigente ausente o ambigua' USING ERRCODE='40001';
    END IF;
    PERFORM pg_catalog.set_config('vec.ct_refiscalizacion.organizacion_ref',p_expediente->>'organizacion_ref',true);
    PERFORM pg_catalog.set_config('vec.ct_refiscalizacion.expediente_ref',p_expediente->>'referencia',true);
    PERFORM pg_catalog.set_config('vec.ct_refiscalizacion.retorno_ref',v_retorno,true);
    SELECT s.*, u.version_expediente AS version_retorno,
           f.fiscalizacion_ref AS fiscalizacion_previa_ref
      INTO v_subsanacion
      FROM vec_contratacion_temporal.reserva_subsanacion_reparos s
      JOIN vec_contratacion_temporal.retorno_fiscalizacion_unidad u USING(retorno_ref)
      JOIN vec_contratacion_temporal.reserva_fiscalizacion f ON f.ambito_hmac=u.ambito_hmac
     WHERE s.retorno_ref=v_retorno AND s.estado='confirmada'
       AND s.organizacion_ref=p_expediente->>'organizacion_ref'
       AND s.expediente_ref=p_expediente->>'referencia'
       AND u.expediente_ref=s.expediente_ref AND u.estado='pendiente'
       AND f.expediente_ref=s.expediente_ref AND f.organizacion_ref=s.organizacion_ref
       AND f.estado='confirmada' AND f.resultado='desfavorable'
       AND f.retorno_ref=u.retorno_ref AND f.version_expediente+1=u.version_expediente
       AND f.fiscalizacion_ref=p_expediente#>>'{fiscalizacion,fiscalizacion_ref}'
       AND u.unidad_ref=p_expediente#>>'{asignacion,unidad_ref}'
       AND u.responsable_ref=p_expediente#>>'{asignacion,responsable_ref}'
       AND u.unidad_ref=p_expediente#>>'{fiscalizacion,retorno,unidad_ref}'
       AND u.responsable_ref=p_expediente#>>'{fiscalizacion,retorno,responsable_ref}'
       AND u.creada_en=(p_expediente#>>'{fiscalizacion,retorno,creado_en}')::timestamptz;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'antecedente durable de nueva fiscalización ausente' USING ERRCODE='40001';
    END IF;
    SELECT x INTO STRICT v_actuacion FROM pg_catalog.jsonb_array_elements(p_expediente->'actuaciones') x
     WHERE x->>'accion_clave'='contratacion_temporal.subsanacion_reparos.registrar'
       AND x->>'retorno_ref'=v_retorno;
    IF v_subsanacion.version_esperada<v_subsanacion.version_retorno
       OR v_subsanacion.version_esperada+1>v_version
       OR v_actuacion->>'recibo_ref' IS DISTINCT FROM v_subsanacion.recibo_ref
       OR (v_actuacion->>'version_expediente')::numeric IS DISTINCT FROM v_subsanacion.version_esperada+1
       OR (v_actuacion->>'secuencia')::numeric <= (p_expediente#>>'{fiscalizacion,actuacion_registro,secuencia}')::numeric
       OR (v_actuacion->>'realizada_en')::timestamptz < (p_expediente#>>'{fiscalizacion,fiscalizada_en}')::timestamptz
       OR v_subsanacion.expediente_siguiente_json->'fiscalizacion' IS DISTINCT FROM p_expediente->'fiscalizacion'
       OR (v_subsanacion.expediente_siguiente_json->'actuaciones')->-1 IS DISTINCT FROM v_actuacion
       OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral v
            JOIN vec_contratacion_temporal.actuacion_expediente_integral a
              ON a.expediente_ref=v.expediente_ref AND a.version_expediente=v.version
           WHERE v.expediente_ref=v_subsanacion.expediente_ref
             AND v.version=v_subsanacion.version_esperada+1
             AND v.origen_version='subsanacion_reparos_v1'
             AND v.operacion_ref=v_subsanacion.reserva_ref
             AND v.agregado_json=v_subsanacion.expediente_siguiente_json
             AND a.recibo_ref=v_subsanacion.recibo_ref AND a.actuacion_json=v_actuacion) THEN
        RAISE EXCEPTION 'ligadura de subsanación divergente' USING ERRCODE='40001';
    END IF;
    RETURN pg_catalog.jsonb_build_object('retorno_previo_ref',v_retorno,
        'subsanacion_recibo_ref',v_subsanacion.recibo_ref);
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.preparar_fiscalizacion_tras_subsanacion_v1(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text, expediente_json text, reserva_ref text,
    fiscalizacion_ref text, recibo_ref text, evento_ref text,
    retorno_ref text, ambito_hmac text, huella_peticion_hmac text,
    organizacion_ref text, expediente_ref text, version_expediente bigint,
    actor_ref text, perfil_ref text, resultado_fiscalizacion text,
    observaciones text, estado text, recibo_json jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_activo jsonb := p_operacion #> '{sellos_hmac,activo}';
    v_retenido jsonb;
    v_huella_buscada text;
    v_reserva vec_contratacion_temporal.reserva_fiscalizacion%ROWTYPE;
    v_actual record;
    v_encontrada boolean := false;
    v_par jsonb;
    v_generaciones text[] := ARRAY[]::text[];
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR p_operacion IS NULL
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR pg_catalog.pg_column_size(p_operacion) > 65536
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion, ARRAY[
               'actor_ref','esquema','expediente_ref','observaciones',
               'operacion','organizacion_ref','perfil_ref',
               'referencias_candidatas','resultado','sellos_hmac',
               'version_expediente'
           ]) IS NOT TRUE
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion) AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.preparar-fiscalizacion.v1'
       OR p_operacion ->> 'operacion' <> 'registrar_resultado'
       OR p_operacion ->> 'version_expediente' !~ '^[1-9][0-9]{0,15}$'
       OR (p_operacion ->> 'version_expediente')::numeric NOT BETWEEN 7 AND 9007199254740990
       OR p_operacion ->> 'resultado' NOT IN (
           'favorable', 'favorable_con_observaciones', 'desfavorable'
       )
       OR p_operacion ->> 'observaciones' <>
          pg_catalog.btrim(p_operacion->>'observaciones',E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))
       OR p_operacion->>'observaciones'<>normalize(p_operacion->>'observaciones',NFC)
       OR pg_catalog.translate(p_operacion->>'observaciones',E'\t\n','') ~ '[[:cntrl:]]'
	   OR pg_catalog.char_length(p_operacion ->> 'observaciones') > 2000
       OR pg_catalog.octet_length(p_operacion ->> 'observaciones') > 8192
       OR (
           (p_operacion ->> 'resultado' = 'favorable' AND
            p_operacion ->> 'observaciones' <> '')
           OR
           (p_operacion ->> 'resultado' IN (
                'favorable_con_observaciones', 'desfavorable'
            ) AND p_operacion ->> 'observaciones' = '')
       )
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'referencias_candidatas',
           ARRAY['evento_ref','fiscalizacion_ref','recibo_ref',
                 'reserva_ref','retorno_ref']) IS NOT TRUE
       OR (
           (p_operacion ->> 'resultado' = 'desfavorable' AND
            coalesce(
                p_operacion #>> '{referencias_candidatas,retorno_ref}', ''
            ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
           OR
           (p_operacion ->> 'resultado' <> 'desfavorable' AND
            p_operacion #>> '{referencias_candidatas,retorno_ref}' <> '')
       )
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'sellos_hmac', ARRAY['activo','retenidos']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           v_activo, ARRAY['ambito_hmac','generacion','huella_peticion_hmac']) IS NOT TRUE
       OR pg_catalog.jsonb_typeof(p_operacion #> '{sellos_hmac,retenidos}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           p_operacion #> '{sellos_hmac,retenidos}') > 16
       OR coalesce(v_activo ->> 'ambito_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR coalesce(v_activo ->> 'huella_peticion_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                 p_operacion #> '{sellos_hmac,retenidos}') AS e(valor)
            WHERE vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
                      e.valor,
                      ARRAY['ambito_hmac','generacion','huella_peticion_hmac']) IS NOT TRUE
               OR coalesce(e.valor ->> 'ambito_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
               OR coalesce(e.valor ->> 'huella_peticion_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(
                 p_operacion -> 'referencias_candidatas') c
            WHERE c.key <> 'retorno_ref'
	      AND coalesce(c.value, '') !~
	          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       )
       OR p_operacion ->> 'organizacion_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'expediente_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'actor_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'perfil_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de fiscalización no autorizada';
    END IF;

    IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion) c
         WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM
            CASE WHEN c.key IN ('referencias_candidatas','sellos_hmac') THEN 'object'
                 WHEN c.key='version_expediente' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion->'referencias_candidatas') c
                   WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM 'string') THEN
        RAISE EXCEPTION 'tipos de nueva fiscalización inválidos' USING ERRCODE='22023';
    END IF;

    FOR v_par IN SELECT value FROM pg_catalog.jsonb_array_elements(
        pg_catalog.jsonb_build_array(v_activo)||(p_operacion#>'{sellos_hmac,retenidos}')) LOOP
        IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(v_par) c WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM
                    CASE WHEN c.key='generacion' THEN 'number' ELSE 'string' END)
           OR v_par->>'generacion' !~ '^[1-9][0-9]{0,8}$'
           OR v_par->>'generacion'=ANY(v_generaciones)
           OR v_par->>'ambito_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v'||(v_par->>'generacion')||':[0-9a-f]{64}$')
           OR v_par->>'huella_peticion_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v'||(v_par->>'generacion')||':[0-9a-f]{64}$') THEN
            RAISE EXCEPTION 'par HMAC de nueva fiscalización inválido' USING ERRCODE='22023';
        END IF;
        v_generaciones:=pg_catalog.array_append(v_generaciones,v_par->>'generacion');
    END LOOP;

    v_huella_buscada := v_activo ->> 'huella_peticion_hmac';
    SELECT r.*
      INTO v_reserva
      FROM vec_contratacion_temporal.reserva_fiscalizacion r
     WHERE r.ambito_hmac = v_activo ->> 'ambito_hmac';
    v_encontrada := FOUND;
    IF NOT v_encontrada THEN
        FOR v_retenido IN
            SELECT valor
              FROM pg_catalog.jsonb_array_elements(
                  p_operacion #> '{sellos_hmac,retenidos}'
              ) AS e(valor)
        LOOP
            v_huella_buscada := v_retenido ->> 'huella_peticion_hmac';
            SELECT r.*
              INTO v_reserva
              FROM vec_contratacion_temporal.reserva_fiscalizacion r
             WHERE r.ambito_hmac = v_retenido ->> 'ambito_hmac';
            IF FOUND THEN
                v_encontrada := true;
                EXIT;
            END IF;
        END LOOP;
    END IF;

    IF v_encontrada THEN
        IF v_reserva.huella_peticion_hmac IS DISTINCT FROM v_huella_buscada
           OR v_reserva.operacion IS DISTINCT FROM p_operacion ->> 'operacion'
           OR v_reserva.organizacion_ref IS DISTINCT FROM
              p_operacion ->> 'organizacion_ref'
           OR v_reserva.expediente_ref IS DISTINCT FROM
              p_operacion ->> 'expediente_ref'
           OR v_reserva.version_expediente IS DISTINCT FROM
              (p_operacion ->> 'version_expediente')::numeric
           OR v_reserva.actor_ref IS DISTINCT FROM p_operacion ->> 'actor_ref'
           OR v_reserva.perfil_ref IS DISTINCT FROM p_operacion ->> 'perfil_ref'
           OR v_reserva.resultado IS DISTINCT FROM p_operacion ->> 'resultado'
           OR v_reserva.observaciones IS DISTINCT FROM
              p_operacion ->> 'observaciones' THEN
            resultado := 'idempotencia_reutilizada';
        ELSIF v_reserva.estado = 'confirmada' THEN
            resultado := 'confirmada';
        ELSE
            resultado := 'reutilizada';
        END IF;
    ELSE
        SELECT a.version, v.agregado_json
          INTO STRICT v_actual
          FROM vec_contratacion_temporal.expediente_integral_actual a
          JOIN vec_contratacion_temporal.expediente_version_integral v
            USING (expediente_ref, version)
         WHERE a.expediente_ref = p_operacion ->> 'expediente_ref';
        IF v_actual.version <> (p_operacion ->> 'version_expediente')::numeric
           OR v_actual.agregado_json ->> 'organizacion_ref' <>
              p_operacion ->> 'organizacion_ref'
           OR v_actual.agregado_json ->> 'fase_actual' IS DISTINCT FROM 'subsanacion_unidad'
           OR v_actual.agregado_json ->> 'estado_actual' IS DISTINCT FROM 'incidencia'
           OR NOT (v_actual.agregado_json ? 'asignacion')
           OR NOT (v_actual.agregado_json ? 'informe_juridico')
           OR v_actual.agregado_json #>> '{fiscalizacion,resultado}' IS DISTINCT FROM 'desfavorable' THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'expediente no disponible para fiscalización';
        END IF;
        v_reserva.ambito_hmac := v_activo ->> 'ambito_hmac';
        v_reserva.huella_peticion_hmac :=
            v_activo ->> 'huella_peticion_hmac';
        v_reserva.operacion := 'registrar_resultado';
        v_reserva.organizacion_ref := p_operacion ->> 'organizacion_ref';
        v_reserva.expediente_ref := p_operacion ->> 'expediente_ref';
        v_reserva.version_expediente := (p_operacion ->> 'version_expediente')::numeric;
        v_reserva.actor_ref := p_operacion ->> 'actor_ref';
        v_reserva.perfil_ref := p_operacion ->> 'perfil_ref';
        v_reserva.resultado := p_operacion ->> 'resultado';
        v_reserva.observaciones := p_operacion ->> 'observaciones';
        v_reserva.reserva_ref :=
            p_operacion #>> '{referencias_candidatas,reserva_ref}';
        v_reserva.fiscalizacion_ref :=
            p_operacion #>> '{referencias_candidatas,fiscalizacion_ref}';
        v_reserva.recibo_ref :=
            p_operacion #>> '{referencias_candidatas,recibo_ref}';
        v_reserva.evento_ref :=
            p_operacion #>> '{referencias_candidatas,evento_ref}';
        v_reserva.retorno_ref := nullif(
            p_operacion #>> '{referencias_candidatas,retorno_ref}', '');
        v_reserva.expediente_anterior_json := v_actual.agregado_json;
        v_reserva.estado := 'preparada';
        resultado := 'preparada';
    END IF;

    IF resultado <> 'idempotencia_reutilizada' THEN
        PERFORM vec_contratacion_temporal.antecedente_refiscalizacion_v1(v_reserva.expediente_anterior_json);
    END IF;
    RETURN QUERY SELECT resultado, v_reserva.expediente_anterior_json::text,
        v_reserva.reserva_ref, v_reserva.fiscalizacion_ref,
        v_reserva.recibo_ref, v_reserva.evento_ref,
        coalesce(v_reserva.retorno_ref, ''),
        v_reserva.ambito_hmac, v_reserva.huella_peticion_hmac,
        v_reserva.organizacion_ref, v_reserva.expediente_ref,
        v_reserva.version_expediente::bigint, v_reserva.actor_ref,
        v_reserva.perfil_ref, v_reserva.resultado,
        v_reserva.observaciones, v_reserva.estado,
        CASE WHEN v_reserva.estado = 'confirmada'
             THEN vec_contratacion_temporal.recibo_fiscalizacion_v1(
                      v_reserva.ambito_hmac)
        END;
EXCEPTION
    WHEN invalid_text_representation OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'preparación de fiscalización inválida';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(
    p_operacion jsonb,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_fiscalizacion%ROWTYPE;
    v_actual record;
    v_consumo record;
    v_decision jsonb;
    v_ahora timestamptz(6);
    v_carga_huella text;
    v_agregado_huella text;
    v_prueba bytea;
    v_payload_evento bytea;
    v_anterior text;
    v_secuencia numeric;
    v_actuacion jsonb;
    v_vinculo jsonb;
    v_fiscalizacion jsonb;
    v_retorno jsonb;
    v_expediente_esperado jsonb;
    v_fase_destino text;
    v_estado_destino text;
    v_unidad_retorno text;
    v_responsable_retorno text;
    v_informe_ref text;
    v_documento_ref text;
    v_contexto_canonico bytea;
    v_contexto_huella text;
    v_observaciones_huella text;
    v_reserva_persistida boolean := false;
    v_antecedente jsonb;
    v_secuencia_actuacion numeric;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR p_operacion IS NULL
       OR pg_catalog.pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.pg_column_size(p_operacion) > 3145728
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion, ARRAY[
               'actor_ref','actuacion','ambito_idempotencia_hmac',
               'autorizacion','esquema','expediente_anterior',
               'expediente_ref','expediente_siguiente',
               'huella_peticion_hmac','instante_efecto','observaciones',
               'operacion','organizacion_ref','perfil_ref','politica',
               'referencias','reserva_ref','resultado','version_anterior'
           ]) IS NOT TRUE
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion) AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.confirmar-fiscalizacion.v1'
       OR p_operacion ->> 'operacion' <> 'registrar_resultado'
       OR p_operacion ->> 'version_anterior' !~ '^[1-9][0-9]{0,15}$'
       OR (p_operacion ->> 'version_anterior')::numeric NOT BETWEEN 7 AND 9007199254740990
       OR p_operacion ->> 'resultado' NOT IN (
           'favorable', 'favorable_con_observaciones', 'desfavorable'
       )
       OR p_operacion ->> 'observaciones' <>
          pg_catalog.btrim(p_operacion->>'observaciones',E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))
       OR p_operacion->>'observaciones'<>normalize(p_operacion->>'observaciones',NFC)
       OR pg_catalog.translate(p_operacion->>'observaciones',E'\t\n','') ~ '[[:cntrl:]]'
       OR pg_catalog.char_length(p_operacion ->> 'observaciones') > 2000
       OR pg_catalog.octet_length(p_operacion ->> 'observaciones') > 8192
       OR (
           (p_operacion ->> 'resultado' = 'favorable' AND
            p_operacion ->> 'observaciones' <> '')
           OR
           (p_operacion ->> 'resultado' IN (
                'favorable_con_observaciones', 'desfavorable'
            ) AND p_operacion ->> 'observaciones' = '')
       )
       OR p_operacion ->> 'ambito_idempotencia_hmac' !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR p_operacion ->> 'huella_peticion_hmac' !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'referencias',
           ARRAY['evento_ref','fiscalizacion_ref','recibo_ref',
                 'reserva_ref','retorno_ref']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'politica',
           ARRAY['accion','definicion_huella_sha256','definicion_ref',
                 'definicion_version','evaluada_en','finalidad',
                 'unidad_fiscalizadora_ref','valida_hasta']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'autorizacion',
           ARRAY['accion','contexto_recurso_huella_sha256',
                 'decision_canonica_hex','decision_huella_sha256',
                 'decision_ref','finalidad','motivo_canonico_hex',
                 'perfil_activo_ref','perfil_version','persona_version',
                 'principal_id','recurso_ref']) IS NOT TRUE
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'referencias')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'politica')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'autorizacion')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'reserva_ref' <>
          p_operacion #>> '{referencias,reserva_ref}'
       OR (
           (p_operacion ->> 'resultado' = 'desfavorable' AND
            coalesce(
                p_operacion #>> '{referencias,retorno_ref}', ''
            ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
           OR
           (p_operacion ->> 'resultado' <> 'desfavorable' AND
            p_operacion #>> '{referencias,retorno_ref}' <> '')
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(
                 p_operacion -> 'referencias') c
            WHERE c.key <> 'retorno_ref'
	      AND coalesce(c.value, '') !~
	          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       )
       OR p_operacion ->> 'organizacion_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'expediente_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'actor_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'perfil_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion #>> '{politica,unidad_fiscalizadora_ref}'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion #>> '{politica,definicion_ref}'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_operacion #>> '{politica,definicion_version}')::numeric
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_operacion #>> '{politica,definicion_huella_sha256}'
          !~ '^[0-9a-f]{64}$'
       OR p_operacion #>> '{politica,accion}' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR p_operacion #>> '{politica,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,accion}' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR p_operacion #>> '{autorizacion,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,recurso_ref}' <>
          p_operacion ->> 'expediente_ref'
       OR p_operacion #>> '{autorizacion,principal_id}' <>
          p_operacion ->> 'actor_ref'
       OR p_operacion #>> '{autorizacion,perfil_activo_ref}' <>
          p_operacion ->> 'perfil_ref'
       OR p_operacion #>> '{autorizacion,decision_canonica_hex}' <>
          pg_catalog.encode(p_decision, 'hex')
       OR p_operacion #>> '{autorizacion,motivo_canonico_hex}' <>
          pg_catalog.encode(p_motivo, 'hex')
       OR (p_operacion #>> '{autorizacion,persona_version}')::numeric <>
          p_persona_version
       OR (p_operacion #>> '{autorizacion,perfil_version}')::numeric <>
          p_perfil_version
       OR p_operacion #>> '{autorizacion,decision_huella_sha256}' <>
          pg_catalog.encode(pg_catalog.sha256(p_decision), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmación de fiscalización no autorizada';
    END IF;

    IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion) c
         WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM
            CASE WHEN c.key IN ('actuacion','autorizacion','expediente_anterior','expediente_siguiente','politica','referencias') THEN 'object'
                 WHEN c.key='version_anterior' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion->'referencias') c
                   WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM 'string') THEN
        RAISE EXCEPTION 'tipos de nueva fiscalización inválidos' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion->'politica') c
                   WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM
                      CASE WHEN c.key='definicion_version' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_each(p_operacion->'autorizacion') c
                   WHERE pg_catalog.jsonb_typeof(c.value) IS DISTINCT FROM
                      CASE WHEN c.key IN ('persona_version','perfil_version') THEN 'number' ELSE 'string' END) THEN
        RAISE EXCEPTION 'tipos de política o autorización inválidos' USING ERRCODE='22023';
    END IF;
    IF pg_catalog.split_part(pg_catalog.split_part(p_operacion->>'ambito_idempotencia_hmac','/v',2),':',1)
       IS DISTINCT FROM pg_catalog.split_part(pg_catalog.split_part(p_operacion->>'huella_peticion_hmac','/v',2),':',1) THEN
        RAISE EXCEPTION 'generaciones HMAC divergentes' USING ERRCODE='22023';
    END IF;

    v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    v_carga_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_operacion::text, 'UTF8')), 'hex');

    SELECT reserva.* INTO r
      FROM vec_contratacion_temporal.reserva_fiscalizacion reserva
     WHERE reserva.ambito_hmac =
           p_operacion ->> 'ambito_idempotencia_hmac'
     FOR UPDATE;
    v_reserva_persistida := FOUND;

    IF v_reserva_persistida THEN
        IF r.huella_peticion_hmac <>
              p_operacion ->> 'huella_peticion_hmac'
           OR r.operacion <> p_operacion ->> 'operacion'
           OR r.organizacion_ref <> p_operacion ->> 'organizacion_ref'
           OR r.expediente_ref <> p_operacion ->> 'expediente_ref'
           OR r.version_expediente <>
              (p_operacion ->> 'version_anterior')::numeric
           OR r.actor_ref <> p_operacion ->> 'actor_ref'
           OR r.perfil_ref <> p_operacion ->> 'perfil_ref'
           OR r.resultado <> p_operacion ->> 'resultado'
           OR r.observaciones <> p_operacion ->> 'observaciones'
           OR r.reserva_ref <> p_operacion ->> 'reserva_ref'
           OR r.fiscalizacion_ref <>
              p_operacion #>> '{referencias,fiscalizacion_ref}'
           OR r.recibo_ref <> p_operacion #>> '{referencias,recibo_ref}'
           OR r.evento_ref <> p_operacion #>> '{referencias,evento_ref}'
           OR coalesce(r.retorno_ref, '') <>
              p_operacion #>> '{referencias,retorno_ref}'
           OR r.expediente_anterior_json <>
              p_operacion -> 'expediente_anterior' THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'clave de fiscalización reutilizada';
        END IF;
        IF r.estado = 'confirmada' THEN
            -- La recuperación HTTP pasa por preparación y autorización fresca.
            -- Nunca devolver el terminal desde una evidencia SQL vieja.
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'recuperar preparación de fiscalización confirmada';
        END IF;
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva de fiscalización no terminal';
    END IF;

    r.ambito_hmac := p_operacion ->> 'ambito_idempotencia_hmac';
    r.huella_peticion_hmac := p_operacion ->> 'huella_peticion_hmac';
    r.operacion := 'registrar_resultado';
    r.organizacion_ref := p_operacion ->> 'organizacion_ref';
    r.expediente_ref := p_operacion ->> 'expediente_ref';
    r.version_expediente := (p_operacion ->> 'version_anterior')::numeric;
    r.actor_ref := p_operacion ->> 'actor_ref';
    r.perfil_ref := p_operacion ->> 'perfil_ref';
    r.resultado := p_operacion ->> 'resultado';
    r.observaciones := p_operacion ->> 'observaciones';
    r.unidad_fiscalizadora_ref :=
        p_operacion #>> '{politica,unidad_fiscalizadora_ref}';
    r.reserva_ref := p_operacion #>> '{referencias,reserva_ref}';
    r.fiscalizacion_ref :=
        p_operacion #>> '{referencias,fiscalizacion_ref}';
    r.recibo_ref := p_operacion #>> '{referencias,recibo_ref}';
    r.evento_ref := p_operacion #>> '{referencias,evento_ref}';
    r.retorno_ref := nullif(
        p_operacion #>> '{referencias,retorno_ref}', '');
    r.expediente_anterior_json := p_operacion -> 'expediente_anterior';

    SELECT v.* INTO STRICT v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        USING (expediente_ref, version)
     WHERE a.expediente_ref = r.expediente_ref
     FOR UPDATE OF a, v;

    IF v_actual.version <> (p_operacion ->> 'version_anterior')::numeric
       OR v_actual.agregado_json <> r.expediente_anterior_json
       OR v_actual.agregado_json ->> 'organizacion_ref' <>
          r.organizacion_ref
       OR v_actual.agregado_json ->> 'fase_actual' IS DISTINCT FROM 'subsanacion_unidad'
       OR v_actual.agregado_json ->> 'estado_actual' IS DISTINCT FROM 'incidencia'
       OR NOT (v_actual.agregado_json ? 'asignacion')
       OR NOT (v_actual.agregado_json ? 'informe_juridico')
       OR v_actual.agregado_json #>> '{fiscalizacion,resultado}' IS DISTINCT FROM 'desfavorable' THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de fiscalización perdido';
    END IF;

    v_antecedente := vec_contratacion_temporal.antecedente_refiscalizacion_v1(v_actual.agregado_json);
    v_secuencia_actuacion := pg_catalog.jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    IF r.retorno_ref IS NOT NULL AND r.retorno_ref=v_antecedente->>'retorno_previo_ref' THEN
        RAISE EXCEPTION 'el nuevo reparo exige otro retorno' USING ERRCODE='22023';
    END IF;
    v_unidad_retorno :=
        v_actual.agregado_json #>> '{asignacion,unidad_ref}';
    v_responsable_retorno :=
        v_actual.agregado_json #>> '{asignacion,responsable_ref}';
    v_informe_ref :=
        v_actual.agregado_json #>> '{informe_juridico,informe_ref}';
    v_documento_ref :=
        v_actual.agregado_json #>> '{informe_juridico,documento_ref}';
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());

    IF v_unidad_retorno
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_responsable_retorno
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_informe_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_documento_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_operacion #>> '{politica,evaluada_en}')::timestamptz >
          (p_operacion ->> 'instante_efecto')::timestamptz
       OR (p_operacion ->> 'instante_efecto')::timestamptz < (v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR (p_operacion ->> 'instante_efecto')::timestamptz IS DISTINCT FROM pg_catalog.date_trunc('microseconds',(p_operacion ->> 'instante_efecto')::timestamptz)
       OR v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >=
          (p_operacion #>> '{politica,valida_hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'coordenadas o vigencia de fiscalización inválidas';
    END IF;

    IF r.resultado = 'desfavorable' THEN
        v_fase_destino := 'subsanacion_unidad';
        v_estado_destino := 'incidencia';
        v_retorno := pg_catalog.jsonb_build_object(
            'retorno_ref', r.retorno_ref,
            'unidad_ref', v_unidad_retorno,
            'responsable_ref', v_responsable_retorno,
            'estado', 'pendiente',
            'creado_en', p_operacion -> 'instante_efecto'
        );
    ELSE
        v_fase_destino := 'fiscalizacion';
        v_estado_destino := 'en_curso';
        v_retorno := NULL;
    END IF;

    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', v_secuencia_actuacion,
        'version_expediente', r.version_expediente+1,
        'accion_clave', 'contratacion_temporal.fiscalizacion.registrar',
        'actor_ref', r.actor_ref,
        'unidad_ref', r.unidad_fiscalizadora_ref,
        'recibo_ref', r.recibo_ref,
        'realizada_en', p_operacion -> 'instante_efecto',
        'fase_origen', v_actual.agregado_json -> 'fase_actual',
        'fase_destino', v_fase_destino,
        'estado_origen', v_actual.agregado_json -> 'estado_actual',
        'estado_destino', v_estado_destino,
        'observaciones', r.observaciones,
        'documentos_ref', pg_catalog.jsonb_build_array(v_documento_ref),
        'retorno_ref', v_antecedente->>'retorno_previo_ref'
    );
    IF r.observaciones = '' THEN
        v_actuacion := v_actuacion - 'observaciones';
    END IF;

    v_vinculo := pg_catalog.jsonb_build_object(
        'secuencia', v_secuencia_actuacion,
        'version_expediente', r.version_expediente+1,
        'accion_clave', 'contratacion_temporal.fiscalizacion.registrar',
        'fase_destino', v_fase_destino,
        'estado_destino', v_estado_destino,
        'recibo_ref', r.recibo_ref,
        'fiscalizacion_ref', r.fiscalizacion_ref,
        'resultado', r.resultado,
        'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
        'informe_juridico_ref', v_informe_ref,
        'documento_informe_ref', v_documento_ref
    );
    IF r.resultado = 'desfavorable' THEN
        v_vinculo := v_vinculo || pg_catalog.jsonb_build_object(
            'retorno_ref', r.retorno_ref,
            'unidad_retorno_ref', v_unidad_retorno,
            'responsable_retorno_ref', v_responsable_retorno
        );
    END IF;

    v_fiscalizacion := pg_catalog.jsonb_build_object(
        'fiscalizacion_ref', r.fiscalizacion_ref,
        'resultado', r.resultado,
        'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
        'informe_juridico_ref', v_informe_ref,
        'documento_informe_ref', v_documento_ref,
        'observaciones', r.observaciones,
        'fiscalizada_en', p_operacion -> 'instante_efecto',
        'actuacion_registro', v_vinculo
    );
    IF r.observaciones = '' THEN
        v_fiscalizacion := v_fiscalizacion - 'observaciones';
    END IF;
    IF v_retorno IS NOT NULL THEN
        v_fiscalizacion := v_fiscalizacion ||
            pg_catalog.jsonb_build_object('retorno', v_retorno);
    END IF;

    v_expediente_esperado := v_actual.agregado_json ||
        pg_catalog.jsonb_build_object(
            'version', r.version_expediente+1,
            'fase_actual', v_fase_destino,
            'estado_actual', v_estado_destino,
            'actualizado_en', p_operacion -> 'instante_efecto',
            'actuaciones', (v_actual.agregado_json -> 'actuaciones') ||
                pg_catalog.jsonb_build_array(v_actuacion),
            'fiscalizacion', v_fiscalizacion
        );
    IF v_secuencia_actuacion > 9007199254740991
       OR p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
       OR p_operacion -> 'expediente_siguiente' IS DISTINCT FROM
          v_expediente_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyección de fiscalización divergente';
    END IF;
    v_observaciones_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(r.observaciones, 'UTF8')), 'hex');
    v_contexto_canonico := pg_catalog.convert_to(
        '{"ambitos":{"estado_previo":"' ||
        (v_actual.agregado_json ->> 'estado_actual') ||
        '","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||
        (v_actual.agregado_json ->> 'fase_actual') ||
        '","organizacion_ref":"' || r.organizacion_ref ||
        '"},"atributos":{"ambito_idempotencia_hmac":"' || r.ambito_hmac ||
        '","documento_informe_ref":"' || v_documento_ref ||
        '","huella_peticion_hmac":"' || r.huella_peticion_hmac ||
        '","informe_juridico_ref":"' || v_informe_ref ||
        '","observaciones_huella_sha256":"' || v_observaciones_huella ||
        '","politica_huella_sha256":"' ||
        (p_operacion #>> '{politica,definicion_huella_sha256}') ||
        '","politica_ref":"' ||
        (p_operacion #>> '{politica,definicion_ref}') ||
        '","politica_version":"' ||
        (p_operacion #>> '{politica,definicion_version}') ||
        '","responsable_asignado_ref":"' || v_responsable_retorno ||
        '","resultado":"' || r.resultado ||
        '","retorno_previo_ref":"' || (v_antecedente->>'retorno_previo_ref') ||
        '","subsanacion_recibo_ref":"' || (v_antecedente->>'subsanacion_recibo_ref') ||
        '","unidad_asignada_ref":"' || v_unidad_retorno ||
        '","unidad_fiscalizadora_ref":"' || r.unidad_fiscalizadora_ref ||
        '","version_expediente":"' || r.version_expediente::text || '"}}',
        'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_canonico), 'hex');

    IF p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}' <>
          v_contexto_huella
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          v_contexto_huella
       OR v_decision ->> 'principal_id' <> r.actor_ref
       OR v_decision ->> 'perfil_activo_ref' <> r.perfil_ref
       OR v_decision ->> 'recurso_ref' <> r.expediente_ref
       OR v_decision ->> 'accion' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR v_decision ->> 'modulo_id' <> 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' <>
          'fiscalizacion_contratacion_temporal'
       OR v_decision ->> 'finalidad' <> 'gestionar_contratacion_temporal'
       OR v_decision ->> 'decision_ref' <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR p_operacion #>> '{politica,unidad_fiscalizadora_ref}' <>
          r.unidad_fiscalizadora_ref THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto autorizado de fiscalización divergente';
    END IF;

    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3
           .registrar_y_consumir_fiscalizacion_v3_atestada(
          p_capacidad, p_decision, p_motivo, p_contexto,
          p_persona_version, p_perfil_version, p_payload, p_sobre,
          p_evidencia, p_raiz
      );
    IF v_consumo.decision_ref <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_consumo.efecto_ref <> r.expediente_ref
       OR v_consumo.huella_efecto_sha256 <> v_contexto_huella
       OR coalesce(v_consumo.auditoria_ref, '') !~
          '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consumo de autorización de fiscalización divergente';
    END IF;

    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());
    IF v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >=
          (p_operacion #>> '{politica,valida_hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vigencia de fiscalización agotada';
    END IF;

    INSERT INTO vec_contratacion_temporal.reserva_fiscalizacion (
        ambito_hmac, huella_peticion_hmac, operacion, organizacion_ref,
        expediente_ref, version_expediente, actor_ref, perfil_ref,
        resultado, observaciones, unidad_fiscalizadora_ref,
        reserva_ref, fiscalizacion_ref, recibo_ref, evento_ref, retorno_ref,
        expediente_anterior_json, reservada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, r.operacion,
        r.organizacion_ref, r.expediente_ref, r.version_expediente,
        r.actor_ref, r.perfil_ref, r.resultado, r.observaciones,
        r.unidad_fiscalizadora_ref, r.reserva_ref, r.fiscalizacion_ref,
        r.recibo_ref, r.evento_ref, r.retorno_ref,
        r.expediente_anterior_json, v_ahora
    );

    v_agregado_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            (p_operacion -> 'expediente_siguiente')::text, 'UTF8')), 'hex');
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-EXPEDIENTE-FISCALIZACION-V1' || chr(10) ||
        r.expediente_ref || chr(10) || (r.version_expediente+1)::text || chr(10) ||
        v_agregado_huella || chr(10) || r.reserva_ref || chr(10) ||
        r.recibo_ref || chr(10) || v_consumo.decision_ref || chr(10) ||
        v_ahora::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json, agregado_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        r.expediente_ref, r.version_expediente+1, p_operacion -> 'expediente_siguiente',
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_actual.flujo_ref, v_actual.flujo_version,
        v_actual.flujo_huella_sha256, v_fase_destino, v_estado_destino,
        'fiscalizacion_o5', r.reserva_ref, v_ahora
    );

    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version = r.version_expediente+1,
           actualizada_en = v_ahora,
           operacion_ref = r.reserva_ref
     WHERE expediente_ref = r.expediente_ref
       AND version = r.version_expediente;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS final de fiscalización perdido';
    END IF;

    v_prueba := pg_catalog.convert_to(
        'VEC-CT-ACTUACION-FISCALIZACION-V1' || chr(10) ||
        (p_operacion -> 'actuacion')::text || chr(10) || r.recibo_ref ||
        chr(10) || v_ahora::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref, secuencia, version_expediente, operacion_ref,
        recibo_ref, actuacion_json, actuacion_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, registrada_en
    ) VALUES (
        r.expediente_ref, v_secuencia_actuacion, r.version_expediente+1, r.reserva_ref, r.recibo_ref,
        p_operacion -> 'actuacion',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            (p_operacion -> 'actuacion')::text, 'UTF8')), 'hex'),
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_ahora
    );

    IF r.resultado = 'desfavorable' THEN
        INSERT INTO vec_contratacion_temporal.retorno_fiscalizacion_unidad (
            retorno_ref, ambito_hmac, expediente_ref, version_expediente,
            unidad_ref, responsable_ref, estado, creada_en
        ) VALUES (
            r.retorno_ref, r.ambito_hmac, r.expediente_ref, r.version_expediente+1,
            v_unidad_retorno, v_responsable_retorno, 'pendiente',
            (p_operacion ->> 'instante_efecto')::timestamptz
        );
    END IF;

    SELECT secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22003',
            MESSAGE = 'límite de outbox alcanzado';
    END IF;
    v_secuencia := v_secuencia + 1;
    v_payload_evento := pg_catalog.convert_to(
        (
            pg_catalog.jsonb_build_object(
                'esquema',
                    'vec.contratacion-temporal.fiscalizacion-registrada.v1',
                'expediente_ref', r.expediente_ref,
                'version_resultante', r.version_expediente+1,
                'fiscalizacion_ref', r.fiscalizacion_ref,
                'resultado', r.resultado,
                'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
                'recibo_ref', r.recibo_ref
            ) ||
            CASE WHEN r.resultado = 'desfavorable'
                 THEN pg_catalog.jsonb_build_object(
                     'retorno_ref', r.retorno_ref,
                     'unidad_retorno_ref', v_unidad_retorno,
                     'responsable_retorno_ref', v_responsable_retorno,
                     'estado_retorno', 'pendiente'
                 )
                 ELSE '{}'::jsonb
            END
        )::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref, secuencia, operacion_ref, expediente_ref,
        version_expediente, tipo_evento, payload_canonico,
        payload_huella_sha256, anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        r.evento_ref, v_secuencia, r.reserva_ref, r.expediente_ref, r.version_expediente+1,
        'contratacion_temporal.fiscalizacion_registrada', v_payload_evento,
        pg_catalog.encode(pg_catalog.sha256(v_payload_evento), 'hex'),
        v_anterior,
        pg_catalog.encode(pg_catalog.sha256(
            v_anterior::bytea || v_payload_evento), 'hex'),
        v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox = v_secuencia,
           cabeza_outbox_sha256 = pg_catalog.encode(pg_catalog.sha256(
               v_anterior::bytea || v_payload_evento), 'hex'),
           actualizada_en = v_ahora
     WHERE control_id;

    INSERT INTO vec_contratacion_temporal.terminal_fiscalizacion (
        ambito_hmac, huella_peticion_hmac, decision_ref,
        decision_huella_sha256, consumo_huella_sha256, auditoria_ref,
        politica_ref, politica_version, politica_huella_sha256,
        carga_huella_sha256, confirmada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, v_consumo.decision_ref,
        p_operacion #>> '{autorizacion,decision_huella_sha256}',
        v_consumo.consumo_huella_sha256, v_consumo.auditoria_ref,
        p_operacion #>> '{politica,definicion_ref}',
        (p_operacion #>> '{politica,definicion_version}')::numeric,
        p_operacion #>> '{politica,definicion_huella_sha256}',
        v_carga_huella, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_fiscalizacion
       SET estado = 'confirmada',
           confirmada_en = v_ahora
     WHERE ambito_hmac = r.ambito_hmac
       AND estado = 'reservada';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva de fiscalización perdida';
    END IF;

    RETURN QUERY SELECT
        vec_contratacion_temporal.recibo_fiscalizacion_v1(r.ambito_hmac);
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow OR
         numeric_value_out_of_range OR character_not_in_repertoire THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada de fiscalización inválida';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.antecedente_refiscalizacion_v1(jsonb),
    vec_contratacion_temporal.preparar_fiscalizacion_tras_subsanacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_fiscalizacion_tras_subsanacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
TO vec_contratacion_temporal_ejecutor;
COMMIT;
