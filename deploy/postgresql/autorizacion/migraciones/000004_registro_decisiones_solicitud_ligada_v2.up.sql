-- Registro nominal de concesiones V2. Se instala despues de la fuente local
-- de sesion/ContextoActor (000002) y de la proyeccion de motivos (000003).
-- Una fila V2 nunca se inserta en decision_autorizacion: esa separacion evita
-- que pueda reconstruirse una representacion V1 y omitir los compromisos de
-- solicitud y motivo exigidos por VEC-AD-2.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:registro_decisiones_v2:000004', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_autorizacion.decision_autorizacion') IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(jsonb,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion.motivo_v2_catalogo_publicado'
       ) IS NULL
       OR to_regclass('vec_autorizacion.motivo_v2_entrada') IS NULL
       OR to_regclass('vec_autorizacion.motivo_v2_retirada') IS NULL
       OR to_regclass(
           'vec_autorizacion.motivo_v2_checkpoint_origen'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para registro de decisiones V2';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.manifiesto_decision_v2_canonico_valido(
    p_manifiesto jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    entrada jsonb;
    referencia text;
    anterior text;
BEGIN
    IF jsonb_typeof(p_manifiesto) IS DISTINCT FROM 'array'
       OR jsonb_array_length(p_manifiesto) > 512 THEN
        RETURN false;
    END IF;
    FOR entrada IN SELECT value FROM jsonb_array_elements(p_manifiesto) LOOP
        IF jsonb_typeof(entrada) IS DISTINCT FROM 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(entrada)) <> 2
           OR NOT (entrada ?& ARRAY['referencia', 'huella_sha256'])
           OR jsonb_typeof(entrada -> 'referencia') <> 'string'
           OR jsonb_typeof(entrada -> 'huella_sha256') <> 'string' THEN
            RETURN false;
        END IF;
        referencia := entrada ->> 'referencia';
        IF vec_autorizacion.texto_positivo_valido(referencia, 512) IS NOT TRUE
           OR (entrada ->> 'huella_sha256') !~ '^[0-9a-f]{64}$'
           OR (anterior IS NOT NULL AND
               referencia COLLATE "C" <= anterior COLLATE "C") THEN
            RETURN false;
        END IF;
        anterior := referencia;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.lista_decision_v2_canonica_valida(
    p_lista jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    elemento jsonb;
    valor text;
    anterior text;
BEGIN
    IF vec_autorizacion.lista_positiva_valida(p_lista, false) IS NOT TRUE THEN
        RETURN false;
    END IF;
    FOR elemento IN SELECT value FROM jsonb_array_elements(p_lista) LOOP
        valor := elemento #>> '{}';
        IF anterior IS NOT NULL AND
           valor COLLATE "C" <= anterior COLLATE "C" THEN
            RETURN false;
        END IF;
        anterior := valor;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Convierte solo la parte comun a la forma durable ya endurecida por 000001 y
-- 000002. NULL significa rechazo. La representacion V2 original se conserva
-- aparte y nunca se expone como documento V1 ejecutable.
CREATE FUNCTION vec_autorizacion.materializar_documento_comun_decision_v2(
    p_documento jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    clave text;
    referencias_evaluadas jsonb;
    huellas_evaluadas jsonb;
    referencias_aplicables jsonb;
    huellas_aplicables jsonb;
    comun jsonb;
BEGIN
    IF jsonb_typeof(p_documento) IS DISTINCT FROM 'object'
       OR pg_column_size(p_documento) > 524288
       OR (SELECT count(*) FROM jsonb_object_keys(p_documento)) <> 34
       OR NOT (p_documento ?& ARRAY[
           'esquema', 'decision_ref', 'concedida', 'codigo', 'principal_id',
           'perfil_activo_ref', 'accion', 'recurso_ref', 'modulo_id',
           'tipo_recurso', 'contexto_recurso_huella_sha256', 'finalidad',
           'correlacion_ref', 'esquema_huella_solicitud',
           'solicitud_huella_sha256', 'esquema_huella_motivo',
           'motivo_huella_sha256', 'vinculo_autenticacion_actor',
           'asignacion_ref', 'asignacion_huella_sha256', 'version_rol_ref',
           'version_rol_huella_sha256',
           'control_vigencia_version_rol_ref',
           'control_vigencia_version_rol_revision',
           'control_vigencia_version_rol_huella_sha256',
           'revision_catalogo_politicas',
           'catalogo_politicas_huella_sha256', 'politicas_evaluadas',
           'politicas_aplicables', 'garantia_minima', 'campos_permitidos',
           'obligaciones', 'emitida_en', 'valida_hasta'
       ])
       OR p_documento ->> 'esquema' <>
          'vec.autorizacion.decision.reforzada.v2.solicitud-ligada'
       OR p_documento ->> 'esquema_huella_solicitud' <>
          'vec.autorizacion.solicitud.v2.efectiva-minimizada'
       OR p_documento ->> 'esquema_huella_motivo' <>
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR jsonb_typeof(p_documento -> 'solicitud_huella_sha256') <> 'string'
       OR (p_documento ->> 'solicitud_huella_sha256') !~ '^[0-9a-f]{64}$'
       OR p_documento ->> 'solicitud_huella_sha256' = repeat('0', 64)
       OR jsonb_typeof(p_documento -> 'motivo_huella_sha256') <> 'string'
       OR (p_documento ->> 'motivo_huella_sha256') !~ '^[0-9a-f]{64}$'
       OR p_documento ->> 'motivo_huella_sha256' = repeat('0', 64)
       OR jsonb_typeof(p_documento -> 'correlacion_ref') <> 'string'
       OR (p_documento ->> 'correlacion_ref') !~
          '^correlacion_[0-9a-f]{32}$'
       OR vec_autorizacion.manifiesto_decision_v2_canonico_valido(
              p_documento -> 'politicas_evaluadas'
          ) IS NOT TRUE
       OR vec_autorizacion.manifiesto_decision_v2_canonico_valido(
              p_documento -> 'politicas_aplicables'
          ) IS NOT TRUE
       OR vec_autorizacion.lista_decision_v2_canonica_valida(
              p_documento -> 'campos_permitidos'
          ) IS NOT TRUE
       OR vec_autorizacion.lista_decision_v2_canonica_valida(
              p_documento -> 'obligaciones'
          ) IS NOT TRUE THEN
        RETURN NULL;
    END IF;

    SELECT COALESCE(jsonb_agg(entrada -> 'referencia'), '[]'::jsonb),
           COALESCE(jsonb_object_agg(
               entrada ->> 'referencia', entrada -> 'huella_sha256'
           ), '{}'::jsonb)
      INTO referencias_evaluadas, huellas_evaluadas
      FROM jsonb_array_elements(
          p_documento -> 'politicas_evaluadas'
      ) AS entrada;
    SELECT COALESCE(jsonb_agg(entrada -> 'referencia'), '[]'::jsonb),
           COALESCE(jsonb_object_agg(
               entrada ->> 'referencia', entrada -> 'huella_sha256'
           ), '{}'::jsonb)
      INTO referencias_aplicables, huellas_aplicables
      FROM jsonb_array_elements(
          p_documento -> 'politicas_aplicables'
      ) AS entrada;

    comun := p_documento - ARRAY[
        'esquema', 'esquema_huella_solicitud',
        'solicitud_huella_sha256', 'esquema_huella_motivo',
        'motivo_huella_sha256', 'politicas_evaluadas',
        'politicas_aplicables'
    ] || jsonb_build_object(
        'politicas_evaluadas_refs', referencias_evaluadas,
        'politicas_evaluadas_huellas_sha256', huellas_evaluadas,
        'politicas_refs', referencias_aplicables,
        'politicas_huellas_sha256', huellas_aplicables
    );
    IF vec_autorizacion.documento_decision_estructura_valida(comun) IS NOT TRUE
       OR NOT (referencias_aplicables <@ referencias_evaluadas)
       OR EXISTS (
           SELECT 1
             FROM jsonb_each(huellas_aplicables) AS aplicable
            WHERE huellas_evaluadas -> aplicable.key IS DISTINCT FROM
                  aplicable.value
       ) THEN
        RETURN NULL;
    END IF;
    RETURN comun;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 (
    decision_ref text PRIMARY KEY,
    huella_decision_sha256 text NOT NULL UNIQUE,
    decision_canonica bytea NOT NULL,
    documento_v2 jsonb NOT NULL,
    documento_comun jsonb NOT NULL,
    principal_id text NOT NULL,
    perfil_activo_ref text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    modulo_id text NOT NULL,
    tipo_recurso text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    finalidad text NOT NULL,
    correlacion_ref text NOT NULL,
    solicitud_huella_sha256 text NOT NULL,
    motivo_huella_sha256 text NOT NULL,
    motivo_canonico bytea NOT NULL,
    motivo_catalogo_id text NOT NULL,
    motivo_catalogo_version integer NOT NULL,
    motivo_catalogo_huella_sha256 text NOT NULL,
    motivo_entrada_clave text NOT NULL,
    asignacion_ref text NOT NULL,
    version_rol_ref text NOT NULL,
    control_vigencia_version_rol_ref text NOT NULL,
    control_vigencia_version_rol_revision numeric(20, 0) NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT decision_v2_referencias CHECK (
        vec_autorizacion.texto_positivo_valido(decision_ref, 512) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(principal_id, 512) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(perfil_activo_ref, 512) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(accion, 256) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(recurso_ref, 512) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(modulo_id, 128) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(tipo_recurso, 128) IS TRUE
        AND vec_autorizacion.texto_positivo_valido(finalidad, 512) IS TRUE
        AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
    ),
    CONSTRAINT decision_v2_huellas CHECK (
        huella_decision_sha256 ~ '^[0-9a-f]{64}$'
        AND solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND solicitud_huella_sha256 <> repeat('0', 64)
        AND motivo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND motivo_huella_sha256 <> repeat('0', 64)
        AND contexto_recurso_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND encode(sha256(decision_canonica), 'hex') =
            huella_decision_sha256
        AND encode(sha256(motivo_canonico), 'hex') =
            motivo_huella_sha256
    ),
    CONSTRAINT decision_v2_tamanos CHECK (
        octet_length(decision_canonica) BETWEEN 1 AND 524288
        AND octet_length(motivo_canonico) BETWEEN 1 AND 65536
    ),
    CONSTRAINT decision_v2_documentos CHECK (
        vec_autorizacion.materializar_documento_comun_decision_v2(
            documento_v2
        ) IS NOT DISTINCT FROM documento_comun
        AND documento_v2 ->> 'decision_ref' = decision_ref
        AND documento_v2 ->> 'principal_id' = principal_id
        AND documento_v2 ->> 'perfil_activo_ref' = perfil_activo_ref
        AND documento_v2 ->> 'accion' = accion
        AND documento_v2 ->> 'recurso_ref' = recurso_ref
        AND documento_v2 ->> 'modulo_id' = modulo_id
        AND documento_v2 ->> 'tipo_recurso' = tipo_recurso
        AND documento_v2 ->> 'contexto_recurso_huella_sha256' =
            contexto_recurso_huella_sha256
        AND documento_v2 ->> 'finalidad' = finalidad
        AND documento_v2 ->> 'correlacion_ref' = correlacion_ref
        AND documento_v2 ->> 'solicitud_huella_sha256' =
            solicitud_huella_sha256
        AND documento_v2 ->> 'motivo_huella_sha256' =
            motivo_huella_sha256
        AND documento_v2 ->> 'asignacion_ref' = asignacion_ref
        AND documento_v2 ->> 'version_rol_ref' = version_rol_ref
        AND documento_v2 ->> 'control_vigencia_version_rol_ref' =
            control_vigencia_version_rol_ref
        AND (documento_v2 ->>
             'control_vigencia_version_rol_revision')::numeric =
            control_vigencia_version_rol_revision
        AND (documento_v2 ->> 'emitida_en')::timestamptz = emitida_en
        AND (documento_v2 ->> 'valida_hasta')::timestamptz = valida_hasta
    ),
    CONSTRAINT decision_v2_motivo CHECK (
        motivo_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$'
        AND motivo_catalogo_version BETWEEN 1 AND 2147483647
        AND motivo_catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND motivo_catalogo_huella_sha256 <> repeat('0', 64)
        AND motivo_entrada_clave ~ '^motivo_[0-9a-f]{32}$'
    ),
    CONSTRAINT decision_v2_vigencia CHECK (
        valida_hasta > emitida_en
        AND valida_hasta <= emitida_en + interval '5 minutes'
        AND registrada_en >= emitida_en
        AND registrada_en < valida_hasta
    ),
    FOREIGN KEY (asignacion_ref)
        REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
    FOREIGN KEY (version_rol_ref)
        REFERENCES vec_autorizacion.version_rol(version_rol_ref),
    FOREIGN KEY (
        control_vigencia_version_rol_ref,
        control_vigencia_version_rol_revision
    ) REFERENCES vec_autorizacion.control_vigencia_version_rol(
        version_rol_ref, revision
    ),
    FOREIGN KEY (motivo_catalogo_id, motivo_catalogo_version)
        REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(
            catalogo_id, catalogo_version
        ),
    FOREIGN KEY (
        motivo_catalogo_id, motivo_catalogo_version, motivo_entrada_clave
    ) REFERENCES vec_autorizacion.motivo_v2_entrada(
        catalogo_id, catalogo_version, entrada_clave
    )
);

CREATE INDEX decision_autorizacion_v2_principal_fecha
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
        principal_id, registrada_en DESC
    );
CREATE INDEX decision_autorizacion_v2_perfil_fecha
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
        perfil_activo_ref, registrada_en DESC
    );
CREATE INDEX decision_autorizacion_v2_correlacion
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
        correlacion_ref, registrada_en DESC
    );

CREATE TRIGGER decision_autorizacion_v2_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER decision_autorizacion_v2_no_truncar
    BEFORE TRUNCATE
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_autorizacion.rechazar_mutacion_inmutable();

ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FOR ALL TO vec_autorizacion_propietario
    USING (current_user = 'vec_autorizacion_propietario')
    WITH CHECK (current_user = 'vec_autorizacion_propietario');

CREATE FUNCTION
vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
    p_decision_canonica bytea,
    p_motivo_canonico bytea
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    decision_v2 jsonb;
    decision_comun jsonb;
    motivo jsonb;
    referencia_motivo jsonb;
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifiesto_actual jsonb;
    referencias_actuales jsonb;
    referencias_decision jsonb;
    referencias_aplicadas jsonb;
    instante timestamptz(6);
BEGIN
    IF p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536 THEN
        RETURN false;
    END IF;
    BEGIN
        decision_v2 := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        motivo := convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN false;
    END;
    decision_comun :=
        vec_autorizacion.materializar_documento_comun_decision_v2(
            decision_v2
        );
    IF decision_comun IS NULL
       OR jsonb_typeof(motivo) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(motivo)) <> 2
       OR NOT (motivo ?& ARRAY['esquema', 'referencia'])
       OR motivo ->> 'esquema' <>
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR jsonb_typeof(motivo -> 'referencia') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              motivo -> 'referencia'
          )) <> 4
       OR NOT ((motivo -> 'referencia') ?& ARRAY[
           'catalogo_id', 'catalogo_version',
           'catalogo_huella_sha256', 'entrada_clave'
       ])
       OR encode(sha256(p_motivo_canonico), 'hex') IS DISTINCT FROM
          decision_v2 ->> 'motivo_huella_sha256' THEN
        RETURN false;
    END IF;
    referencia_motivo := motivo -> 'referencia';
    IF jsonb_typeof(referencia_motivo -> 'catalogo_id') <> 'string'
       OR (referencia_motivo ->> 'catalogo_id') !~
          '^[a-z][a-z0-9._-]{0,127}$'
       OR jsonb_typeof(referencia_motivo -> 'catalogo_version') <> 'number'
       OR (referencia_motivo ->> 'catalogo_version') !~ '^[1-9][0-9]{0,9}$'
       OR (referencia_motivo ->> 'catalogo_version')::numeric NOT BETWEEN
          1 AND 2147483647
       OR jsonb_typeof(
              referencia_motivo -> 'catalogo_huella_sha256'
          ) <> 'string'
       OR (referencia_motivo ->> 'catalogo_huella_sha256') !~
          '^[0-9a-f]{64}$'
       OR referencia_motivo ->> 'catalogo_huella_sha256' = repeat('0', 64)
       OR jsonb_typeof(referencia_motivo -> 'entrada_clave') <> 'string'
       OR (referencia_motivo ->> 'entrada_clave') !~
          '^motivo_[0-9a-f]{32}$' THEN
        RETURN false;
    END IF;

    -- La barrera se toma antes de cualquier reloj autoritativo. Publicar o
    -- retirar necesita FOR UPDATE sobre el mismo checkpoint; desde aqui el
    -- catalogo de motivos queda estable hasta COMMIT.
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;

    -- Evita que una migracion descendente adquiera ACCESS EXCLUSIVE mientras
    -- se coteja la instantanea y haga esperar al INSERT despues del reloj.
    LOCK TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
        IN ROW EXCLUSIVE MODE;

    SELECT actual.asignacion_ref, asignacion.principal_id,
           asignacion.version_rol_ref, asignacion.huella_sha256
      INTO asignacion_actual
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = actual.perfil_activo_ref
       AND asignacion.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = decision_comun ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion_actual.asignacion_ref IS DISTINCT FROM
          decision_comun ->> 'asignacion_ref'
       OR asignacion_actual.principal_id IS DISTINCT FROM
          decision_comun ->> 'principal_id'
       OR asignacion_actual.version_rol_ref IS DISTINCT FROM
          decision_comun ->> 'version_rol_ref'
       OR asignacion_actual.huella_sha256 IS DISTINCT FROM
          decision_comun ->> 'asignacion_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT rol.huella_sha256, control.version_rol_ref, control.revision,
           control.huella_sha256 AS huella_control
      INTO rol_actual
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = asignacion_actual.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol_actual.huella_sha256 IS DISTINCT FROM
          decision_comun ->> 'version_rol_huella_sha256'
       OR rol_actual.version_rol_ref IS DISTINCT FROM
          decision_comun ->> 'control_vigencia_version_rol_ref'
       OR rol_actual.revision IS DISTINCT FROM
          (decision_comun ->>
           'control_vigencia_version_rol_revision')::numeric
       OR rol_actual.huella_control IS DISTINCT FROM
          decision_comun ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND
       OR catalogo_actual.revision IS DISTINCT FROM
          (decision_comun ->> 'revision_catalogo_politicas')::numeric
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM
          decision_comun ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_object_agg(
               politica.politica_ref, politica.huella_sha256
               ORDER BY politica.politica_ref COLLATE "C"
           ), '{}'::jsonb),
           COALESCE(jsonb_agg(
               politica.politica_ref
               ORDER BY politica.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto_actual, referencias_actuales
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS politica
        ON politica.politica_id = actual.politica_id
       AND politica.politica_ref = actual.politica_ref;
    SELECT COALESCE(jsonb_agg(
               referencia ORDER BY referencia COLLATE "C"
           ), '[]'::jsonb)
      INTO referencias_decision
      FROM jsonb_array_elements_text(
          decision_comun -> 'politicas_evaluadas_refs'
      ) AS referencia;
    SELECT COALESCE(jsonb_agg(
               referencia ORDER BY referencia COLLATE "C"
           ), '[]'::jsonb)
      INTO referencias_aplicadas
      FROM jsonb_array_elements_text(
          decision_comun -> 'politicas_refs'
      ) AS referencia;
    IF manifiesto_actual IS DISTINCT FROM
          decision_comun -> 'politicas_evaluadas_huellas_sha256'
       OR referencias_actuales IS DISTINCT FROM referencias_decision
       OR EXISTS (
           SELECT 1
             FROM jsonb_each(
                 decision_comun -> 'politicas_huellas_sha256'
             ) AS aplicada
            WHERE manifiesto_actual -> aplicada.key IS DISTINCT FROM
                  aplicada.value
       )
       OR jsonb_array_length(referencias_aplicadas) IS DISTINCT FROM
          (SELECT count(*)
             FROM jsonb_object_keys(
                 decision_comun -> 'politicas_huellas_sha256'
             )) THEN
        RETURN false;
    END IF;

    -- Primera pasada: valida la ventana completa y adquiere los locks de los
    -- punteros de sesion y ContextoActor. emitida_en es evidencia, no reloj.
    IF vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
           decision_comun -> 'vinculo_autenticacion_actor',
           decision_comun ->> 'principal_id',
           decision_comun ->> 'perfil_activo_ref',
           decision_comun -> 'vinculo_autenticacion_actor'
               ->> 'contexto_actor_huella_sha256',
           (decision_comun ->> 'emitida_en')::timestamptz,
           (decision_comun ->> 'valida_hasta')::timestamptz,
           (decision_comun ->> 'emitida_en')::timestamptz
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    -- Unico reloj autoritativo. La segunda pasada no espera: esta transaccion
    -- ya posee ambos locks. El motivo se coteja contra exactamente el mismo
    -- instante mientras su checkpoint sigue bloqueado.
    instante := clock_timestamp();
    IF instante < (decision_comun ->> 'emitida_en')::timestamptz
       OR instante >= (decision_comun ->> 'valida_hasta')::timestamptz
       OR vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
              decision_comun -> 'vinculo_autenticacion_actor',
              decision_comun ->> 'principal_id',
              decision_comun ->> 'perfil_activo_ref',
              decision_comun -> 'vinculo_autenticacion_actor'
                  ->> 'contexto_actor_huella_sha256',
              (decision_comun ->> 'emitida_en')::timestamptz,
              (decision_comun ->> 'valida_hasta')::timestamptz,
              instante
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_v2_catalogo_publicado AS catalogo
          JOIN vec_autorizacion.motivo_v2_entrada AS entrada
            ON entrada.catalogo_id = catalogo.catalogo_id
           AND entrada.catalogo_version = catalogo.catalogo_version
         WHERE catalogo.catalogo_id = referencia_motivo ->> 'catalogo_id'
           AND catalogo.catalogo_version =
               (referencia_motivo ->> 'catalogo_version')::integer
           AND catalogo.catalogo_huella_publicada_sha256 =
               referencia_motivo ->> 'catalogo_huella_sha256'
           AND entrada.entrada_clave =
               referencia_motivo ->> 'entrada_clave'
           AND catalogo.publicado_en <= instante
           AND entrada.vigente_desde <= instante
           AND (entrada.vigente_hasta IS NULL
                OR instante < entrada.vigente_hasta)
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion.motivo_v2_retirada AS retirada
                WHERE retirada.catalogo_id = catalogo.catalogo_id
                  AND retirada.catalogo_version = catalogo.catalogo_version
           )
    ) THEN
        RETURN false;
    END IF;

    INSERT INTO vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 (
        decision_ref, huella_decision_sha256, decision_canonica,
        documento_v2, documento_comun, principal_id, perfil_activo_ref,
        accion, recurso_ref, modulo_id, tipo_recurso,
        contexto_recurso_huella_sha256, finalidad, correlacion_ref,
        solicitud_huella_sha256, motivo_huella_sha256, motivo_canonico,
        motivo_catalogo_id, motivo_catalogo_version,
        motivo_catalogo_huella_sha256, motivo_entrada_clave,
        asignacion_ref, version_rol_ref,
        control_vigencia_version_rol_ref,
        control_vigencia_version_rol_revision,
        emitida_en, valida_hasta, registrada_en
    ) VALUES (
        decision_v2 ->> 'decision_ref',
        encode(sha256(p_decision_canonica), 'hex'),
        p_decision_canonica, decision_v2, decision_comun,
        decision_v2 ->> 'principal_id',
        decision_v2 ->> 'perfil_activo_ref',
        decision_v2 ->> 'accion', decision_v2 ->> 'recurso_ref',
        decision_v2 ->> 'modulo_id', decision_v2 ->> 'tipo_recurso',
        decision_v2 ->> 'contexto_recurso_huella_sha256',
        decision_v2 ->> 'finalidad', decision_v2 ->> 'correlacion_ref',
        decision_v2 ->> 'solicitud_huella_sha256',
        decision_v2 ->> 'motivo_huella_sha256', p_motivo_canonico,
        referencia_motivo ->> 'catalogo_id',
        (referencia_motivo ->> 'catalogo_version')::integer,
        referencia_motivo ->> 'catalogo_huella_sha256',
        referencia_motivo ->> 'entrada_clave',
        decision_v2 ->> 'asignacion_ref',
        decision_v2 ->> 'version_rol_ref',
        decision_v2 ->> 'control_vigencia_version_rol_ref',
        (decision_v2 ->>
         'control_vigencia_version_rol_revision')::numeric,
        (decision_v2 ->> 'emitida_en')::timestamptz,
        (decision_v2 ->> 'valida_hasta')::timestamptz,
        instante
    );
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON TABLE
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.manifiesto_decision_v2_canonico_valido(jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.lista_decision_v2_canonica_valida(jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.materializar_documento_comun_decision_v2(jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        bytea, bytea
    ) FROM PUBLIC;

COMMIT;
