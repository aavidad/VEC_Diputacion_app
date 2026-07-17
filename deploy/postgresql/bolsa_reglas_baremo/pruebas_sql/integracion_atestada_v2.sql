-- Vectores mecanicos de composicion. El broker, COSE y el verificador de
-- evidencia no se simulan como productivos: se reutiliza el catalogo de
-- confianza de prueba de VEC-AD-2 y toda la transaccion termina en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $retirar_checks_nominales$
DECLARE
    restriccion record;
BEGIN
    FOR restriccion IN
        SELECT conname FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
           AND contype IN ('c', 'f')
    LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 DROP CONSTRAINT %I',
            restriccion.conname
        );
    END LOOP;
END
$retirar_checks_nominales$;
DROP TRIGGER decision_autorizacion_v2_inmutable ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2;
CREATE SEQUENCE vec_autorizacion.prueba_reglas_v2_nominal_seq;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_reglas_consumidor_atestado_prueba;
GRANT USAGE, SELECT ON SEQUENCE
    vec_autorizacion.prueba_reglas_v2_nominal_seq
    TO vec_autorizacion_propietario,
       vec_reglas_consumidor_atestado_prueba;

CREATE OR REPLACE FUNCTION
vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
    p_decision_canonica bytea,
    p_motivo_canonico bytea
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $sustituto$
DECLARE
    documento jsonb;
BEGIN
    documento := convert_from(p_decision_canonica, 'UTF8')::jsonb;
    PERFORM nextval(
        'vec_autorizacion.prueba_reglas_v2_nominal_seq'::regclass
    );
    INSERT INTO vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
        decision_ref, huella_decision_sha256, decision_canonica,
        documento_v2, documento_comun, principal_id, perfil_activo_ref,
        accion, recurso_ref, modulo_id, tipo_recurso,
        contexto_recurso_huella_sha256, finalidad, correlacion_ref,
        solicitud_huella_sha256, motivo_huella_sha256, motivo_canonico,
        motivo_catalogo_id, motivo_catalogo_version,
        motivo_catalogo_huella_sha256, motivo_entrada_clave,
        asignacion_ref, version_rol_ref, control_vigencia_version_rol_ref,
        control_vigencia_version_rol_revision,
        emitida_en, valida_hasta, registrada_en
    ) VALUES (
        documento ->> 'decision_ref',
        encode(sha256(p_decision_canonica), 'hex'), p_decision_canonica,
        documento, '{}'::jsonb, documento ->> 'principal_id',
        'perfil:reglas:v2:prueba', documento ->> 'accion',
        documento ->> 'recurso_ref', documento ->> 'modulo_id',
        documento ->> 'tipo_recurso',
        documento ->> 'contexto_recurso_huella_sha256',
        documento ->> 'finalidad', documento ->> 'correlacion_ref',
        repeat('1', 64), documento ->> 'motivo_huella_sha256',
        p_motivo_canonico, 'motivos.reglas.v2.prueba', 1,
        repeat('2', 64), 'motivo_11111111111111111111111111111111',
        'asignacion:reglas:v2:prueba', 'rol:reglas:v2:prueba',
        'rol:reglas:v2:prueba', 1,
        (documento ->> 'emitida_en')::timestamptz,
        (documento ->> 'valida_hasta')::timestamptz,
        date_trunc('microseconds', clock_timestamp())
    );
    RETURN true;
END
$sustituto$;

CREATE TEMP TABLE vectores_reglas_v2 (
    nombre text PRIMARY KEY,
    plan bytea NOT NULL,
    decision bytea NOT NULL,
    payload bytea NOT NULL,
    sobre bytea NOT NULL,
    evidencia bytea NOT NULL,
    raiz bytea NOT NULL,
    capacidad jsonb NOT NULL,
    huella_plan text NOT NULL,
    huella_estado text NOT NULL,
    recibo_ref text
);

CREATE FUNCTION pg_temp.crear_vector_reglas_v2(
    p_nombre text,
    p_operacion text,
    p_contenido_sufijo text,
    p_intencion_semilla text,
    p_decision_semilla text,
    p_nonce text,
    p_cas_huella text DEFAULT NULL,
    p_cas_revision numeric DEFAULT NULL,
    p_sujeto_capacidad text DEFAULT NULL,
    p_vigencia_evidencia interval DEFAULT interval '10 minutes'
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $generar$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    verificada timestamptz(6) := ahora - interval '200 milliseconds';
    emitida timestamptz(6) := ahora - interval '100 milliseconds';
    expira timestamptz(6) := ahora + interval '4 seconds';
    decision_hasta timestamptz(6) := ahora + interval '5 minutes';
    configuracion record;
    raiz record;
    clave record;
    contenido_ref text := 'rgl_' || p_contenido_sufijo;
    convocatoria_ref text := 'con_11111111111111111111111111111111';
    expediente_ref text := 'exp_22222222222222222222222222222222';
    huella_contexto text :=
        '5a2626eed065e5d317177a1ea312b393e523690c24c852f6de8c2abbc2bca2fb';
    huella_contenido text := repeat('a', 64);
    principal text := 'per_0123456789abcdef0123456789abcdef';
    sujeto_plan text := 'hmac-sha256:reglas_baremo_v2:' || repeat('8', 64);
    accion text;
    estado text;
    revision numeric;
    intencion_huella text := encode(sha256(convert_to(
        p_intencion_semilla, 'UTF8'
    )), 'hex');
    evidencia_huella text := encode(sha256(convert_to(
        'evidencia:' || p_intencion_semilla, 'UTF8'
    )), 'hex');
    evidencia_ref text;
    motivo jsonb;
    motivo_bytes bytea;
    motivo_huella text;
    motivo_version jsonb;
    contenido jsonb;
    version jsonb;
    version_bytes bytea;
    huella_estado text;
    vinculo_resultado jsonb;
    cas jsonb;
    vinculo_evidencia jsonb;
    plan jsonb;
    plan_bytes bytea;
    huella_plan text;
    decision jsonb;
    decision_bytes bytea;
    payload bytea := convert_to('payload-reglas-v2-' || p_nombre, 'UTF8');
    sobre bytea := convert_to('cose-sign1-reglas-v2-' || p_nombre, 'UTF8');
    evidencia bytea := convert_to(
        'evidencia-verificacion-reglas-v2-' || p_nombre, 'UTF8'
    );
    capacidad jsonb;
BEGIN
    IF p_contenido_sufijo !~ '^[0-9a-f]{32}$'
       OR p_nonce !~ '^[0-9a-f]{64}$'
       OR p_vigencia_evidencia <= interval '0 seconds'
       OR p_vigencia_evidencia > interval '10 minutes' THEN
        RAISE EXCEPTION 'semilla de vector invalida';
    END IF;
    CASE p_operacion
    WHEN 'alta_borrador' THEN
        accion := 'bolsa.reglas_baremo.borrador.crear';
        estado := 'borrador'; revision := 1;
    WHEN 'publicar' THEN
        accion := 'bolsa.reglas_baremo.publicar';
        estado := 'publicada'; revision := 2;
        evidencia_ref :=
            'atestacion:reglas-baremo:v2:' || evidencia_huella;
    ELSE
        RAISE EXCEPTION 'operacion de vector no soportada';
    END CASE;

    motivo := jsonb_build_object(
        'esquema',
            'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
        'referencia', jsonb_build_object(
            'catalogo_id', 'motivos.reglas.v2.prueba',
            'catalogo_version', 1,
            'catalogo_huella_sha256', repeat('2', 64),
            'entrada_clave', 'motivo_11111111111111111111111111111111'
        )
    );
    motivo_bytes := convert_to(motivo::text, 'UTF8');
    motivo_huella := encode(sha256(motivo_bytes), 'hex');
    motivo_version := jsonb_build_object(
        'catalogo', jsonb_build_object(
            'referencia', 'motivos.reglas.v2.prueba',
            'version', 1, 'huella_sha256', repeat('2', 64)
        ),
        'clave', 'motivo_11111111111111111111111111111111'
    );
    contenido := jsonb_build_object(
        'identidad', jsonb_build_object(
            'referencia', contenido_ref, 'version', 1,
            'convocatoria_ref', convocatoria_ref,
            'expediente_ref', expediente_ref
        )
    );
    version := jsonb_build_object(
        'esquema', 'vec.bolsa.gobierno-reglas-baremo.v1',
        'contenido', contenido,
        'referencia_contenido', jsonb_build_object(
            'referencia', contenido_ref, 'version', 1,
            'huella_sha256', huella_contenido
        ),
        'revision', revision, 'estado', estado,
        'creada_por', principal, 'creada_en', to_char(
            CASE WHEN p_operacion = 'alta_borrador' THEN ahora
                 ELSE ahora - interval '1 second' END,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'motivo_creacion', motivo_version
    );
    IF p_operacion = 'publicar' THEN
        version := version || jsonb_build_object(
            'publicacion', jsonb_build_object(
                'actor_ref', principal, 'motivo', motivo_version,
                'instante', to_char(
                    ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                ),
                'aprobacion', jsonb_build_object(
                    'atestacion', jsonb_build_object(
                        'referencia', evidencia_ref, 'version', 2,
                        'huella_sha256', evidencia_huella
                    ),
                    'vinculo', jsonb_build_object(
                        'contenido', jsonb_build_object(
                            'referencia', contenido_ref, 'version', 1,
                            'huella_sha256', huella_contenido
                        ),
                        'revision', 1,
                        'huella_estado_sha256', p_cas_huella
                    ),
                    'valida_hasta', to_char(
                        ahora + p_vigencia_evidencia,
                        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                    )
                )
            )
        );
    END IF;
    version_bytes := convert_to(version::text, 'UTF8');
    huella_estado := encode(sha256(version_bytes), 'hex');
    vinculo_resultado := jsonb_build_object(
        'contenido', jsonb_build_object(
            'referencia', contenido_ref, 'version', 1,
            'huella_sha256', huella_contenido
        ),
        'revision', revision, 'huella_estado_sha256', huella_estado
    );
    IF p_operacion = 'alta_borrador' THEN
        cas := 'null'::jsonb;
        vinculo_evidencia := 'null'::jsonb;
    ELSE
        cas := jsonb_build_object(
            'contenido', vinculo_resultado -> 'contenido',
            'revision', p_cas_revision,
            'huella_estado_sha256', p_cas_huella
        );
        vinculo_evidencia := jsonb_build_object(
            'referencia', evidencia_ref, 'version', 2,
            'huella_sha256', evidencia_huella
        );
    END IF;
    plan := jsonb_build_object(
        'esquema', 'vec.bolsa.gobierno-reglas-baremo.plan-cambio.v2',
        'operacion', p_operacion,
        'intencion', jsonb_build_object(
            'referencia',
                'intencion:reglas-baremo:v2:' || intencion_huella,
            'version', 2, 'huella_sha256', intencion_huella
        ),
        'cas_esperado', cas,
        'version_resultado_canonica',
            replace(encode(version_bytes, 'base64'), E'\n', ''),
        'huella_version_resultado_sha256', huella_estado,
        'vinculo_resultado', vinculo_resultado,
        'vinculo_evidencia', vinculo_evidencia,
        'sujeto_seudonimo_hmac', sujeto_plan,
        'principal_ref', principal,
        'motivo_canonico',
            replace(encode(motivo_bytes, 'base64'), E'\n', ''),
        'huella_motivo_sha256', motivo_huella,
        'correlacion_ref',
            'correlacion_' || substr(encode(sha256(convert_to(
                'correlacion:' || p_nombre, 'UTF8'
            )), 'hex'), 1, 32),
        'instante_transicion', to_char(
            ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'accion', accion, 'modulo_id', 'bolsa',
        'tipo_recurso', 'version_reglas_baremo_gobernada',
        'perfil_proteccion', 'interno_alto',
        'recurso_ref', 'reglas-baremo:' || huella_estado,
        'convocatoria_ref', convocatoria_ref,
        'expediente_ref', expediente_ref,
        'huella_contexto_recurso_sha256', huella_contexto,
        'finalidad', 'gobierno_reglas_baremo',
        'campos',
            '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb,
        'requisitos_ejecucion',
            '["alcance_resuelto_servidor","cotejo_evidencia_verificador_confiable","decision_vec_v2_consumible","consumo_atestado_vec_ad2_mismo_commit","commit_serializable_atomico","reloj_autoritativo_frescura_cotejada","recibo_durable_reconciliable"]'::jsonb,
        'componentes',
            '["contenido","version","puntero_cas","vinculo_evidencia","vec","auditoria","outbox","recibo"]'::jsonb
    );
    plan_bytes := convert_to(plan::text, 'UTF8');
    huella_plan := encode(sha256(plan_bytes), 'hex');
    decision := jsonb_build_object(
        'decision_ref', 'decision:reglas:v2:' || p_decision_semilla,
        'motivo_huella_sha256', motivo_huella,
        'principal_id', principal, 'accion', accion,
        'modulo_id', 'bolsa',
        'tipo_recurso', 'version_reglas_baremo_gobernada',
        'recurso_ref', 'reglas-baremo:' || huella_estado,
        'contexto_recurso_huella_sha256', huella_contexto,
        'finalidad', 'gobierno_reglas_baremo',
        'correlacion_ref', plan ->> 'correlacion_ref',
        'emitida_en', to_char(
            ahora - interval '1 second',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'valida_hasta', to_char(
            decision_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'garantia_minima', 'alto',
        'campos_permitidos', plan -> 'campos',
        'obligaciones', '[]'::jsonb,
        'vinculo_autenticacion_actor', jsonb_build_object(
            'garantia_observada', 'alto',
            'metodo_observado', 'kerberos_certificado',
            'superficie', 'interna_corporativa',
            'cuenta_privilegiada', false
        )
    );
    decision_bytes := convert_to(decision::text, 'UTF8');

    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
      JOIN vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
        ON puntero.revision = version.revision
     ORDER BY puntero.orden DESC LIMIT 1;
    SELECT version.clave_id, version.version, version.clave_publica_spki,
           version.huella_clave_spki_sha256, version.valida_desde,
           version.valida_hasta, version.suite, version.audiencia_despliegue
      INTO STRICT raiz
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
        ON version.clave_id = enlace.clave_id
       AND version.version = enlace.version
     WHERE enlace.configuracion_revision = configuracion.revision
     ORDER BY version.clave_id COLLATE "C" LIMIT 1;
    SELECT version.* INTO STRICT clave
      FROM vec_autorizacion_atestada_v2.clave_capacidad_version AS version
      JOIN vec_autorizacion_atestada_v2.puntero_clave_capacidad AS puntero
        ON puntero.clave_id = version.clave_id
       AND puntero.version = version.version
     ORDER BY puntero.orden DESC LIMIT 1;

    capacidad := jsonb_build_object(
        'esquema',
            'vec.autorizacion.capacidad-registro-consumo-atestado.v2',
        'clave_id', clave.clave_id, 'clave_version', clave.version,
        'emisor_id', clave.emisor_id, 'audiencia', clave.audiencia,
        'nonce', p_nonce,
        'emitida_en', to_char(emitida, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'expira_en', to_char(expira, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'registro_ref', 'registro:reglas:v2:' || p_decision_semilla,
        'consumo_ref', 'consumo:reglas:v2:' || p_decision_semilla,
        'decision_ref', decision ->> 'decision_ref',
        'huella_decision_sha256', encode(sha256(decision_bytes), 'hex'),
        'huella_motivo_sha256', motivo_huella,
        'huella_payload_vec_ad_2_sha256', encode(sha256(payload), 'hex'),
        'huella_sobre_cose_sign1_sha256', encode(sha256(sobre), 'hex'),
        'huella_evidencia_verificacion_sha256',
            encode(sha256(evidencia), 'hex'),
        'principal_id', principal, 'accion', accion,
        'finalidad', 'gobierno_reglas_baremo',
        'sujeto_ref', COALESCE(p_sujeto_capacidad, sujeto_plan),
        'recurso_ref', decision ->> 'recurso_ref',
        'contexto_recurso_huella_sha256', huella_contexto,
        'correlacion_ref', decision ->> 'correlacion_ref',
        'decision_valida_hasta', decision ->> 'valida_hasta',
        'efecto_ref', 'efecto:reglas-baremo:v2:' || huella_plan,
        'huella_efecto_sha256', huella_plan,
        'verificada_en', to_char(
            verificada, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'revision_confianza', configuracion.revision::text,
        'huella_configuracion_sha256',
            configuracion.huella_configuracion_sha256,
        'configuracion_publicada_en', to_char(
            configuracion.publicada_en,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'configuracion_expira_en', to_char(
            configuracion.expira_en,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_clave_id', raiz.clave_id, 'raiz_version', raiz.version,
        'huella_raiz_spki_sha256', raiz.huella_clave_spki_sha256,
        'raiz_valida_desde', to_char(
            raiz.valida_desde, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_valida_hasta', to_char(
            raiz.valida_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'suite', raiz.suite,
        'audiencia_despliegue', raiz.audiencia_despliegue
    );
    capacidad := capacidad || jsonb_build_object(
        'mac_sha256', encode(public.hmac(
            vec_autorizacion_atestada_v2.preimagen_capacidad(capacidad),
            clave.secreto_hmac, 'sha256'
        ), 'hex')
    );
    INSERT INTO pg_temp.vectores_reglas_v2(
        nombre, plan, decision, payload, sobre, evidencia, raiz,
        capacidad, huella_plan, huella_estado
    ) VALUES (
        p_nombre, plan_bytes, decision_bytes, payload, sobre, evidencia,
        raiz.clave_publica_spki, capacidad, huella_plan, huella_estado
    );
END
$generar$;

SELECT pg_temp.crear_vector_reglas_v2(
    'alta', 'alta_borrador', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'intencion-alta', 'alta', repeat('3', 64)
);
SELECT pg_temp.crear_vector_reglas_v2(
    'publicar', 'publicar', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'intencion-publicar', 'publicar', repeat('4', 64),
    (SELECT huella_estado FROM pg_temp.vectores_reglas_v2
      WHERE nombre = 'alta'), 1
);
SELECT pg_temp.crear_vector_reglas_v2(
    'cas_obsoleto', 'publicar', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'intencion-cas-obsoleto', 'cas-obsoleto', repeat('5', 64),
    repeat('f', 64), 1
);
SELECT pg_temp.crear_vector_reglas_v2(
    'atomico', 'alta_borrador', 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    'intencion-atomica', 'atomico', repeat('6', 64)
);
SELECT pg_temp.crear_vector_reglas_v2(
    'sujeto_cruzado', 'alta_borrador',
    'cccccccccccccccccccccccccccccccc', 'intencion-sujeto',
    'sujeto-cruzado', repeat('7', 64), NULL, NULL,
    'hmac-sha256:pagos:' || repeat('9', 64)
);
SELECT pg_temp.crear_vector_reglas_v2(
    'caduca_durante_vec', 'publicar',
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'intencion-caducidad',
    'caducidad', repeat('8', 64),
    (SELECT huella_estado FROM pg_temp.vectores_reglas_v2
      WHERE nombre = 'alta'), 1, NULL, interval '2 seconds'
);

GRANT SELECT, UPDATE ON TABLE pg_temp.vectores_reglas_v2
    TO vec_reglas_consumidor_atestado_prueba;

-- Demora exclusivamente el vector señalado despues de que VEC-AD-2 haya
-- fijado su instante nominal. Asi se demuestra que la frontera modular toma
-- una hora nueva al volver de VEC y no reutiliza una instantanea anterior.
CREATE FUNCTION pg_temp.demorar_post_instante_vec_reglas_v2()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $demora$
BEGIN
    IF NEW.efecto_ref IS NOT DISTINCT FROM current_setting(
           'vec.prueba_efecto_demora_reglas_v2', true
       ) THEN
        PERFORM pg_sleep(2.2);
    END IF;
    RETURN NEW;
END
$demora$;
CREATE TRIGGER prueba_caducidad_post_instante_vec
    AFTER INSERT ON vec_autorizacion_atestada_v2.auditoria_consumo_v2
    FOR EACH ROW EXECUTE FUNCTION
        pg_temp.demorar_post_instante_vec_reglas_v2();

SET SESSION AUTHORIZATION vec_reglas_consumidor_atestado_prueba;

WITH vector AS (
    SELECT * FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'alta'
), recibo AS (
    SELECT resultado.*
      FROM vector
      CROSS JOIN LATERAL
        vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
            vector.plan, vector.decision, vector.payload, vector.sobre,
            vector.evidencia, vector.raiz, vector.capacidad
        ) AS resultado
)
UPDATE pg_temp.vectores_reglas_v2 AS vector
   SET recibo_ref = recibo.recibo_ref
  FROM recibo WHERE vector.nombre = 'alta';

DO $adversarial_precentral$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
    antes bigint;
    despues bigint;
BEGIN
    SELECT last_value INTO antes
      FROM vec_autorizacion.prueba_reglas_v2_nominal_seq;
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'alta';
    BEGIN
        PERFORM * FROM
            vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
                vector.plan, vector.decision, vector.payload,
                vector.sobre, vector.evidencia, vector.raiz,
                vector.capacidad
            );
        RAISE EXCEPTION 'segundo intento devolvio recibo';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'cas_obsoleto';
    BEGIN
        PERFORM * FROM
            vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
                vector.plan, vector.decision, vector.payload,
                vector.sobre, vector.evidencia, vector.raiz,
                vector.capacidad
            );
        RAISE EXCEPTION 'CAS obsoleto aceptado';
    EXCEPTION WHEN serialization_failure THEN
        NULL;
    END;
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'sujeto_cruzado';
    BEGIN
        PERFORM * FROM
            vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
                vector.plan, vector.decision, vector.payload,
                vector.sobre, vector.evidencia, vector.raiz,
                vector.capacidad
            );
        RAISE EXCEPTION 'sujeto de otro dominio aceptado';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
    SELECT last_value INTO despues
      FROM vec_autorizacion.prueba_reglas_v2_nominal_seq;
    IF despues <> antes THEN
        RAISE EXCEPTION 'un rechazo precentral alcanzo el registrador VEC';
    END IF;
END
$adversarial_precentral$;

DO $caducidad_durante_vec$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
    antes bigint;
    despues bigint;
    mensaje text;
BEGIN
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2
     WHERE nombre = 'caduca_durante_vec';
    SELECT last_value INTO antes
      FROM vec_autorizacion.prueba_reglas_v2_nominal_seq;
    PERFORM set_config(
        'vec.prueba_efecto_demora_reglas_v2',
        vector.capacidad ->> 'efecto_ref', true
    );
    BEGIN
        PERFORM * FROM
            vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
                vector.plan, vector.decision, vector.payload,
                vector.sobre, vector.evidencia, vector.raiz,
                vector.capacidad
            );
        RAISE EXCEPTION 'evidencia caducada durante VEC fue aceptada';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        GET STACKED DIAGNOSTICS mensaje = MESSAGE_TEXT;
        IF mensaje <> 'reloj o evidencia V2 vencieron durante VEC-AD-2' THEN
            RAISE;
        END IF;
    END;
    PERFORM set_config(
        'vec.prueba_efecto_demora_reglas_v2', '', true
    );
    SELECT last_value INTO despues
      FROM vec_autorizacion.prueba_reglas_v2_nominal_seq;
    IF despues <> antes + 1 THEN
        RAISE EXCEPTION 'la caducidad no ocurrio despues del registro nominal';
    END IF;
END
$caducidad_durante_vec$;

RESET SESSION AUTHORIZATION;
DO $comprobar_caducidad_post_vec$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
BEGIN
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2
     WHERE nombre = 'caduca_durante_vec';
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion_atestada_v2.consumo_decision_v2
         WHERE decision_ref = vector.capacidad ->> 'decision_ref'
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
         WHERE decision_ref = vector.capacidad ->> 'decision_ref'
    ) THEN
        RAISE EXCEPTION 'caducidad post-VEC no revirtio consumo y efecto';
    END IF;
END
$comprobar_caducidad_post_vec$;
SET SESSION AUTHORIZATION vec_reglas_consumidor_atestado_prueba;

WITH vector AS (
    SELECT * FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'publicar'
), recibo AS (
    SELECT resultado.*
      FROM vector
      CROSS JOIN LATERAL
        vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
            vector.plan, vector.decision, vector.payload, vector.sobre,
            vector.evidencia, vector.raiz, vector.capacidad
        ) AS resultado
)
UPDATE pg_temp.vectores_reglas_v2 AS vector
   SET recibo_ref = recibo.recibo_ref
  FROM recibo WHERE vector.nombre = 'publicar';

RESET SESSION AUTHORIZATION;

CREATE FUNCTION pg_temp.fallar_version_reglas_v2()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $fallo$
BEGIN
    RAISE EXCEPTION USING ERRCODE = 'PZ001',
        MESSAGE = 'fallo local posterior a VEC para probar atomicidad';
END
$fallo$;
CREATE TRIGGER prueba_fallo_post_vec
    BEFORE INSERT ON vec_bolsa_reglas_baremo.version_reglas_baremo
    FOR EACH ROW EXECUTE FUNCTION pg_temp.fallar_version_reglas_v2();

SET SESSION AUTHORIZATION vec_reglas_consumidor_atestado_prueba;
DO $atomicidad$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
BEGIN
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'atomico';
    BEGIN
        PERFORM * FROM
            vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
                vector.plan, vector.decision, vector.payload,
                vector.sobre, vector.evidencia, vector.raiz,
                vector.capacidad
            );
        RAISE EXCEPTION 'fallo local no interrumpio la confirmacion';
    EXCEPTION WHEN SQLSTATE 'PZ001' THEN
        NULL;
    END;
END
$atomicidad$;
RESET SESSION AUTHORIZATION;
DROP TRIGGER prueba_fallo_post_vec ON
    vec_bolsa_reglas_baremo.version_reglas_baremo;

DO $comprobar_atomicidad$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
BEGIN
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'atomico';
    IF EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v2.consumo_decision_v2
            WHERE decision_ref = vector.capacidad ->> 'decision_ref'
       ) OR EXISTS (
           SELECT 1 FROM vec_bolsa_reglas_baremo.version_reglas_baremo
            WHERE contenido_ref = 'rgl_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
       ) OR EXISTS (
           SELECT 1 FROM vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2
            WHERE decision_ref = vector.capacidad ->> 'decision_ref'
       ) THEN
        RAISE EXCEPTION 'rollback no revirtio consumo y efecto conjuntamente';
    END IF;
END
$comprobar_atomicidad$;

SET SESSION AUTHORIZATION vec_reglas_consumidor_atestado_prueba;

-- Statement separado: recuperacion de COMMIT ambiguo, nunca reintento de la
-- decision single-use. Debe devolver exactamente el mismo recibo durable.
DO $reconciliacion$
DECLARE
    vector pg_temp.vectores_reglas_v2%ROWTYPE;
    recibo record;
BEGIN
    FOR vector IN
        SELECT * FROM pg_temp.vectores_reglas_v2
         WHERE nombre IN ('alta', 'publicar') ORDER BY nombre
    LOOP
        SELECT * INTO recibo
          FROM vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
              vector.capacidad ->> 'decision_ref',
              vector.capacidad ->> 'huella_decision_sha256',
              vector.capacidad ->> 'efecto_ref',
              vector.capacidad ->> 'huella_efecto_sha256',
              vector.capacidad ->> 'nonce', vector.huella_plan
          );
        IF NOT FOUND OR recibo.recibo_ref IS DISTINCT FROM
           vector.recibo_ref THEN
            RAISE EXCEPTION 'reconciliacion no devolvio el recibo exacto';
        END IF;
    END LOOP;
    SELECT * INTO STRICT vector
      FROM pg_temp.vectores_reglas_v2 WHERE nombre = 'alta';
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
              vector.capacidad ->> 'decision_ref',
              vector.capacidad ->> 'huella_decision_sha256',
              vector.capacidad ->> 'efecto_ref',
              vector.capacidad ->> 'huella_efecto_sha256',
              repeat('f', 64), vector.huella_plan
          )
    ) THEN
        RAISE EXCEPTION 'reconciliacion acepto nonce distinto';
    END IF;
END
$reconciliacion$;

RESET SESSION AUTHORIZATION;

DO $grafo_completo$
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2) <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.version_reglas_baremo
            WHERE contenido_ref =
                  'rgl_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa') <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.uso_prueba_transicion
            WHERE prueba_ref LIKE
                  'atestacion:reglas-baremo:v2:%') <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.outbox
            WHERE ruta =
                  'bolsa.reglas_baremo.estado_confirmado.v2') <> 2
       OR (SELECT count(*)
             FROM vec_autorizacion_atestada_v2.consumo_decision_v2
            WHERE efecto_ref LIKE
                  'efecto:reglas-baremo:v2:%') <> 2 THEN
        RAISE EXCEPTION 'grafo atestado V2 incompleto';
    END IF;
END
$grafo_completo$;

\if :{?vec_reglas_probar_down_con_historia}
-- El runner concatena a continuacion el down V2 en esta misma sesion. La
-- historia sigue visible y el down debe abortar; el cierre de psql revierte
-- tambien todos los sustitutos mecanicos de esta prueba.
\else
ROLLBACK;
\endif
