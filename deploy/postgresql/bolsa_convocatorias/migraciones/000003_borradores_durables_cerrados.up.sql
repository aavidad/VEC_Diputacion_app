-- Vertical durable de borradores. Separa expresamente consulta del diario,
-- reserva post-PDP, nucleo legado de confirmacion no expuesto y recuperacion.
--
-- Ninguna funcion recibe ni conserva la clave de idempotencia. L y F son HMAC
-- nominales distintos. El diario, sus auditorias y su outbox no conservan
-- principal ni motivo en claro. El agregado se guarda como sobre cifrado y la
-- lectura tabular usa una proyeccion acotada, nunca el documento grande.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- El propietario fija tambien el minimo privilegio de los objetos futuros.
-- No se confia en los valores por defecto de PostgreSQL (EXECUTE/USAGE para
-- PUBLIC en funciones y tipos) ni en que una migracion posterior recuerde
-- revocarlos al final.
REVOKE CREATE ON SCHEMA vec_bolsa_convocatorias FROM PUBLIC,
    vec_bolsa_convocatorias_ejecutor_consulta,
    vec_bolsa_convocatorias_proyector_gobierno,
    vec_bolsa_convocatorias_registrador_atestacion;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias
    REVOKE ALL ON TABLES FROM PUBLIC,
        vec_bolsa_convocatorias_ejecutor_consulta,
        vec_bolsa_convocatorias_proyector_gobierno,
        vec_bolsa_convocatorias_registrador_atestacion;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias
    REVOKE ALL ON SEQUENCES FROM PUBLIC,
        vec_bolsa_convocatorias_ejecutor_consulta,
        vec_bolsa_convocatorias_proyector_gobierno,
        vec_bolsa_convocatorias_registrador_atestacion;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias
    REVOKE ALL ON FUNCTIONS FROM PUBLIC,
        vec_bolsa_convocatorias_ejecutor_consulta,
        vec_bolsa_convocatorias_proyector_gobierno,
        vec_bolsa_convocatorias_registrador_atestacion;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias
    REVOKE ALL ON TYPES FROM PUBLIC,
        vec_bolsa_convocatorias_ejecutor_consulta,
        vec_bolsa_convocatorias_proyector_gobierno,
        vec_bolsa_convocatorias_registrador_atestacion;

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_bolsa_convocatorias.version_convocatoria') IS NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(jsonb,bytea,bytea,text,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL
       OR to_regclass('vec_bolsa_convocatorias.diario_borrador_version')
          IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar borradores durables';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_convocatorias.referencia_clave_hmac_valida(
    p_valor text, p_dominio text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND octet_length(p_valor) BETWEEN 1 AND 128
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[[:space:][:cntrl:]*]'
       AND CASE p_dominio
           WHEN 'localizador' THEN
               p_valor ~ '^clave:hmac:convocatorias:localizador:[a-z0-9][a-z0-9._:-]{0,88}$'
           WHEN 'huella_solicitud' THEN
               p_valor ~ '^clave:hmac:convocatorias:huella:[a-z0-9][a-z0-9._:-]{0,88}$'
           WHEN 'motivo' THEN
               p_valor ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$'
           ELSE false
       END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.hmac_sha256_valido(p_valor bytea)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND octet_length(p_valor) = 32
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.objeto_json_exacto(
    p_objeto jsonb, p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_objeto IS NOT NULL AND jsonb_typeof(p_objeto) = 'object'
       AND (SELECT array_agg(clave ORDER BY clave)
              FROM jsonb_object_keys(p_objeto) AS clave)
           IS NOT DISTINCT FROM
           (SELECT array_agg(clave ORDER BY clave) FROM unnest(p_claves) AS clave)
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.lista_texto_canonica(
    p_valores text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valores IS NOT NULL
       AND cardinality(p_valores) > 0
       AND NOT EXISTS (
           SELECT 1 FROM unnest(p_valores) WITH ORDINALITY AS actual(valor, posicion)
            WHERE valor IS NULL OR valor !~ '^[a-z0-9][a-z0-9._-]{0,79}$'
               OR (posicion > 1 AND valor <= p_valores[posicion - 1])
       )
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.referencia_estado_valida(
    p_estado jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT vec_bolsa_convocatorias.objeto_json_exacto(
               p_estado,
               ARRAY['huella_estado_sha256','referencia','revision']
           ) IS TRUE
       AND vec_bolsa_convocatorias.texto_opaco_valido(
               p_estado ->> 'referencia', 512
           ) IS TRUE
       AND (p_estado ->> 'referencia') ~ '^.+#[1-9][0-9]{0,18}$'
       AND (p_estado ->> 'revision') ~ '^[1-9][0-9]{0,18}$'
       AND vec_bolsa_convocatorias.huella_sha256_valida(
               p_estado ->> 'huella_estado_sha256'
           ) IS TRUE
$funcion$;

ALTER TABLE vec_bolsa_convocatorias.atestacion_autorizacion_version
    ADD CONSTRAINT atestacion_borrador_vinculo_exacto UNIQUE (
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_evidencia_sha256
    );

-- Enlace local exacto de la atestacion PDP. La fila base contiene el sobre
-- COSE; esta proyeccion fija tambien el verificador que vio el caso de uso.
CREATE TABLE vec_bolsa_convocatorias.atestacion_pdp_borrador (
    decision_ref text NOT NULL,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    verificador_ref text NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_atestacion_sha256
    ),
    FOREIGN KEY (
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_atestacion_sha256
    ) REFERENCES vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_evidencia_sha256
    ),
    CONSTRAINT atestacion_pdp_borrador_valida CHECK (
        estado = 'activa' AND version > 0
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            verificador_ref, 512
        ) IS TRUE
        AND verificador_ref <> atestacion_ref
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_decision_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            decision_ref, 512
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_atestacion_sha256
        ) IS TRUE
        AND verificada_en <= registrada_en
    )
);

-- Una atestacion proyectada solo es autoridad mientras la version COSE
-- inmutable sigue siendo exactamente la seleccionada por el puntero actual.
-- Todas las mutaciones la releen con el reloj de PostgreSQL; no se confia en
-- un instante suministrado por el proceso llamador.
CREATE FUNCTION vec_bolsa_convocatorias.atestacion_pdp_borrador_vigente(
    p_decision_ref text,
    p_atestacion_ref text,
    p_version bigint,
    p_estado text,
    p_huella_decision_sha256 text,
    p_huella_atestacion_sha256 text,
    p_verificador_ref text,
    p_verificada_en timestamptz,
    p_instante_bd timestamptz
)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
    SELECT p_instante_bd IS NOT NULL
       AND EXISTS (
           SELECT 1
             FROM vec_bolsa_convocatorias.atestacion_pdp_borrador AS p
             JOIN vec_bolsa_convocatorias.atestacion_autorizacion_version AS v
               ON v.decision_ref = p.decision_ref
              AND v.atestacion_ref = p.atestacion_ref
              AND v.version = p.version
              AND v.estado = p.estado
              AND v.huella_decision_sha256 = p.huella_decision_sha256
              AND v.huella_evidencia_sha256 =
                  p.huella_atestacion_sha256
             JOIN vec_bolsa_convocatorias.atestacion_autorizacion_actual AS a
               ON a.decision_ref = v.decision_ref
              AND a.atestacion_ref = v.atestacion_ref
              AND a.version = v.version
              AND a.estado = v.estado
            WHERE p.decision_ref = p_decision_ref
              AND p.atestacion_ref = p_atestacion_ref
              AND p.version = p_version
              AND p.estado = p_estado
              AND p.estado = 'activa'
              AND p.huella_decision_sha256 = p_huella_decision_sha256
              AND p.huella_atestacion_sha256 =
                  p_huella_atestacion_sha256
              AND p.verificador_ref = p_verificador_ref
              AND p.verificada_en = p_verificada_en
              AND v.valida_desde <= p_instante_bd
              AND p_instante_bd < v.valida_hasta
       )
$funcion$;

-- Material V2 exacto. Es seguro persistirlo porque solo contiene referencias
-- de estado y el compromiso HMAC del motivo, nunca su texto o su SHA desnuda.
CREATE TABLE vec_bolsa_convocatorias.material_borrador (
    huella_material_sha256 text PRIMARY KEY,
    material_canonico bytea NOT NULL,
    esquema text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    revision_nueva bigint NOT NULL,
    huella_estado_nuevo_sha256 text NOT NULL,
    revision_esperada bigint,
    huella_estado_esperado_sha256 text,
    dominio_criptografico_motivo text NOT NULL,
    generacion_clave_motivo bigint NOT NULL,
    clave_hmac_motivo_ref text NOT NULL,
    valor_hmac_motivo bytea NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (
        huella_material_sha256, accion, recurso_ref, revision_nueva,
        huella_estado_nuevo_sha256, dominio_criptografico_motivo,
        generacion_clave_motivo, clave_hmac_motivo_ref,
        valor_hmac_motivo
    ),
    CONSTRAINT material_borrador_integro CHECK (
        esquema = 'bolsa.convocatoria.intencion.v2'
        AND accion IN (
            'bolsa.convocatoria.borrador.crear',
            'bolsa.convocatoria.borrador.actualizar'
        )
        AND octet_length(material_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(material_canonico), 'hex') =
            huella_material_sha256
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_material_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            recurso_ref, 512
        ) IS TRUE
        AND recurso_ref ~ '^.+#[1-9][0-9]{0,18}$'
        AND revision_nueva > 0
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_estado_nuevo_sha256
        ) IS TRUE
        AND ((accion = 'bolsa.convocatoria.borrador.crear'
              AND revision_nueva = 1
              AND revision_esperada IS NULL
              AND huella_estado_esperado_sha256 IS NULL)
             OR
             (accion = 'bolsa.convocatoria.borrador.actualizar'
              AND revision_esperada > 0
              AND revision_nueva = revision_esperada + 1
              AND vec_bolsa_convocatorias.huella_sha256_valida(
                  huella_estado_esperado_sha256
              ) IS TRUE
              AND huella_estado_esperado_sha256 <>
                  huella_estado_nuevo_sha256))
        AND dominio_criptografico_motivo =
            'bolsa.convocatoria.motivo.v1'
        AND generacion_clave_motivo BETWEEN 1 AND 4294967295
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            clave_hmac_motivo_ref, 'motivo'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            valor_hmac_motivo
        ) IS TRUE
    )
);

-- Historia del diario. Cada fila contiene una proyeccion completa; el puntero
-- actual solo referencia una revision, por lo que no puede divergir de ella.
CREATE TABLE vec_bolsa_convocatorias.diario_borrador_version (
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision bigint NOT NULL,
    huella_esquema_version integer NOT NULL,
    huella_clave_ref text NOT NULL,
    huella_generacion_clave bigint NOT NULL,
    huella_hmac bytea NOT NULL,
    estado text NOT NULL,
    cercado bigint NOT NULL,
    arrendamiento_inicia_en timestamptz(6) NOT NULL,
    arrendamiento_vence_en timestamptz(6) NOT NULL,
    accion text NOT NULL,
    huella_material_sha256 text NOT NULL,
    recurso_ref text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    esquema_huella_decision text NOT NULL,
    decision_ref text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    modulo_id text NOT NULL,
    tipo_recurso text NOT NULL,
    finalidad text NOT NULL,
    version_rol_ref text NOT NULL,
    version_rol_huella_sha256 text NOT NULL,
    control_vigencia_rol_revision bigint NOT NULL,
    revision_catalogo_politicas bigint NOT NULL,
    catalogo_politicas_huella_sha256 text NOT NULL,
    decision_verificada_en timestamptz(6) NOT NULL,
    decision_valida_hasta timestamptz(6) NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    atestacion_estado text NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    verificador_ref text NOT NULL,
    atestacion_verificada_en timestamptz(6) NOT NULL,
    transaccion_ref text,
    estado_principal_ref text,
    estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    evento_outbox_ref text,
    huella_evento_outbox_sha256 text,
    confirmada_en timestamptz(6),
    registrada_en timestamptz(6) NOT NULL,
    recibo_ref text,
    recibo_canonico bytea,
    huella_recibo_sha256 text,
    prueba_desenlace_ref text,
    huella_prueba_desenlace_sha256 text,
    asignacion_ref text NOT NULL,
    asignacion_huella_sha256 text NOT NULL,
    control_vigencia_version_rol_ref text NOT NULL,
    control_vigencia_version_rol_huella_sha256 text NOT NULL,
    decision_emitida_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    ),
    FOREIGN KEY (huella_material_sha256)
        REFERENCES vec_bolsa_convocatorias.material_borrador(
            huella_material_sha256
        ),
    FOREIGN KEY (
        decision_ref, atestacion_ref, atestacion_version,
        atestacion_estado, huella_decision_sha256,
        huella_atestacion_sha256
    ) REFERENCES vec_bolsa_convocatorias.atestacion_pdp_borrador(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_atestacion_sha256
    ),
    CONSTRAINT diario_borrador_identidad_nominal CHECK (
        localizador_esquema_version BETWEEN 1 AND 65535
        AND localizador_generacion_clave BETWEEN 1 AND 4294967295
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            localizador_clave_ref, 'localizador'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            localizador_hmac
        ) IS TRUE
        AND huella_esquema_version BETWEEN 1 AND 65535
        AND huella_generacion_clave BETWEEN 1 AND 4294967295
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            huella_clave_ref, 'huella_solicitud'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(huella_hmac) IS TRUE
        AND localizador_esquema_version = huella_esquema_version
        AND localizador_generacion_clave = huella_generacion_clave
    ),
    CONSTRAINT diario_borrador_estado_control CHECK (
        revision > 0 AND cercado > 0
        AND estado IN (
            'reservado', 'en_curso', 'indeterminado',
            'confirmado', 'no_aplicado'
        )
        AND arrendamiento_inicia_en < arrendamiento_vence_en
        AND arrendamiento_vence_en - arrendamiento_inicia_en <=
            interval '5 minutes'
    ),
    CONSTRAINT diario_borrador_decision_exacta CHECK (
        esquema_huella_decision =
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
        AND accion IN (
            'bolsa.convocatoria.borrador.crear',
            'bolsa.convocatoria.borrador.actualizar'
        )
        AND modulo_id = 'bolsa'
        AND tipo_recurso = 'version_convocatoria_gobernada'
        AND finalidad = 'gobierno_convocatorias'
        AND recurso_ref ~ '^.+#[1-9][0-9]{0,18}$'
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            contexto_recurso_huella_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_decision_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            version_rol_huella_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            asignacion_ref, 512
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            asignacion_huella_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            version_rol_ref, 512
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            control_vigencia_version_rol_ref, 512
        ) IS TRUE
        AND control_vigencia_version_rol_ref = version_rol_ref
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            control_vigencia_version_rol_huella_sha256
        ) IS TRUE
        AND control_vigencia_rol_revision > 0
        AND revision_catalogo_politicas > 0
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            catalogo_politicas_huella_sha256
        ) IS TRUE
        AND atestacion_estado = 'activa'
        AND atestacion_version > 0
        AND decision_ref <> version_rol_ref
        AND decision_ref <> atestacion_ref
        AND version_rol_ref <> atestacion_ref
        AND asignacion_ref <> decision_ref
        AND decision_emitida_en <= atestacion_verificada_en
        AND atestacion_verificada_en <= decision_verificada_en
        AND decision_verificada_en < decision_valida_hasta
        AND arrendamiento_inicia_en >= decision_verificada_en
        AND arrendamiento_vence_en <= decision_valida_hasta
    ),
    CONSTRAINT diario_borrador_recibo_exacto CHECK (
        (estado = 'confirmado'
         AND transaccion_ref IS NOT NULL
         AND estado_principal_ref = recurso_ref
         AND estado_principal_revision > 0
         AND vec_bolsa_convocatorias.huella_sha256_valida(
             estado_principal_huella_sha256
         ) IS TRUE
         AND auditoria_ref IS NOT NULL
         AND vec_bolsa_convocatorias.huella_sha256_valida(
             huella_auditoria_sha256
         ) IS TRUE
         AND evento_outbox_ref IS NOT NULL
         AND vec_bolsa_convocatorias.huella_sha256_valida(
             huella_evento_outbox_sha256
         ) IS TRUE
         AND confirmada_en IS NOT NULL
         AND vec_bolsa_convocatorias.texto_opaco_valido(
             recibo_ref, 512
         ) IS TRUE
         AND recibo_ref ~ '^recibo-borrador-[0-9a-f]{64}$'
         AND octet_length(recibo_canonico) BETWEEN 2 AND 1048576
         AND vec_bolsa_convocatorias.huella_sha256_valida(
             huella_recibo_sha256
         ) IS TRUE
         AND encode(sha256(recibo_canonico), 'hex') =
             huella_recibo_sha256)
        OR
        (estado <> 'confirmado'
         AND transaccion_ref IS NULL AND estado_principal_ref IS NULL
         AND estado_principal_revision IS NULL
         AND estado_principal_huella_sha256 IS NULL
         AND auditoria_ref IS NULL AND huella_auditoria_sha256 IS NULL
         AND evento_outbox_ref IS NULL
         AND huella_evento_outbox_sha256 IS NULL
         AND confirmada_en IS NULL
         AND recibo_ref IS NULL AND recibo_canonico IS NULL
         AND huella_recibo_sha256 IS NULL
         AND ((estado = 'no_aplicado'
               AND vec_bolsa_convocatorias.texto_opaco_valido(
                   prueba_desenlace_ref, 512
               ) IS TRUE
               AND prueba_desenlace_ref ~
                   '^prueba-desenlace-borrador-[0-9a-f]{64}$'
               AND vec_bolsa_convocatorias.huella_sha256_valida(
                   huella_prueba_desenlace_sha256
               ) IS TRUE)
              OR
              (estado <> 'no_aplicado'
               AND prueba_desenlace_ref IS NULL
               AND huella_prueba_desenlace_sha256 IS NULL)))
    )
);
CREATE UNIQUE INDEX diario_borrador_recibo_ref_unico
    ON vec_bolsa_convocatorias.diario_borrador_version(recibo_ref)
    WHERE recibo_ref IS NOT NULL;

CREATE TABLE vec_bolsa_convocatorias.diario_borrador_actual (
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision bigint NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac
    ),
    FOREIGN KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    ) REFERENCES vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    )
);

-- Todas las parejas L/F presentadas durante la reserva se conservan como
-- alias HMAC opacos. Una rotacion solapada encuentra siempre la misma
-- operacion primaria, aunque ninguna de las dos listas comparta su primer
-- elemento. Las filas son historia de idempotencia y no se purgan antes que
-- el diario al que apuntan.
ALTER TABLE vec_bolsa_convocatorias.diario_borrador_version
    ADD CONSTRAINT diario_borrador_identidad_revision_exacta UNIQUE (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision,
        huella_esquema_version, huella_clave_ref,
        huella_generacion_clave, huella_hmac
    );

CREATE TABLE vec_bolsa_convocatorias.identidad_alias_borrador (
    alias_localizador_esquema_version integer NOT NULL,
    alias_localizador_clave_ref text NOT NULL,
    alias_localizador_generacion_clave bigint NOT NULL,
    alias_localizador_hmac bytea NOT NULL,
    alias_huella_esquema_version integer NOT NULL,
    alias_huella_clave_ref text NOT NULL,
    alias_huella_generacion_clave bigint NOT NULL,
    alias_huella_hmac bytea NOT NULL,
    primario_localizador_esquema_version integer NOT NULL,
    primario_localizador_clave_ref text NOT NULL,
    primario_localizador_generacion_clave bigint NOT NULL,
    primario_localizador_hmac bytea NOT NULL,
    primario_huella_esquema_version integer NOT NULL,
    primario_huella_clave_ref text NOT NULL,
    primario_huella_generacion_clave bigint NOT NULL,
    primario_huella_hmac bytea NOT NULL,
    revision_origen bigint NOT NULL DEFAULT 1,
    ordinalidad bigint NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        alias_localizador_esquema_version, alias_localizador_clave_ref,
        alias_localizador_generacion_clave, alias_localizador_hmac
    ),
    UNIQUE (
        alias_huella_esquema_version, alias_huella_clave_ref,
        alias_huella_generacion_clave, alias_huella_hmac
    ),
    UNIQUE (
        primario_localizador_esquema_version,
        primario_localizador_clave_ref,
        primario_localizador_generacion_clave,
        primario_localizador_hmac, ordinalidad
    ),
    CONSTRAINT identidad_alias_generacion_primaria_unica UNIQUE (
        primario_localizador_esquema_version,
        primario_localizador_clave_ref,
        primario_localizador_generacion_clave,
        primario_localizador_hmac,
        alias_localizador_generacion_clave
    ),
    FOREIGN KEY (
        primario_localizador_esquema_version,
        primario_localizador_clave_ref,
        primario_localizador_generacion_clave,
        primario_localizador_hmac, revision_origen,
        primario_huella_esquema_version, primario_huella_clave_ref,
        primario_huella_generacion_clave, primario_huella_hmac
    ) REFERENCES vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision,
        huella_esquema_version, huella_clave_ref,
        huella_generacion_clave, huella_hmac
    ),
    CONSTRAINT identidad_alias_borrador_nominal CHECK (
        alias_localizador_esquema_version BETWEEN 1 AND 65535
        AND alias_huella_esquema_version =
            alias_localizador_esquema_version
        AND alias_localizador_generacion_clave BETWEEN 1 AND 4294967295
        AND alias_huella_generacion_clave =
            alias_localizador_generacion_clave
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            alias_localizador_clave_ref, 'localizador'
        ) IS TRUE
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            alias_huella_clave_ref, 'huella_solicitud'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            alias_localizador_hmac
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            alias_huella_hmac
        ) IS TRUE
        AND primario_localizador_esquema_version BETWEEN 1 AND 65535
        AND primario_huella_esquema_version =
            primario_localizador_esquema_version
        AND primario_localizador_generacion_clave BETWEEN 1 AND 4294967295
        AND primario_huella_generacion_clave =
            primario_localizador_generacion_clave
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            primario_localizador_clave_ref, 'localizador'
        ) IS TRUE
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            primario_huella_clave_ref, 'huella_solicitud'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            primario_localizador_hmac
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(
            primario_huella_hmac
        ) IS TRUE
        AND revision_origen = 1
        AND ordinalidad > 0
        AND (ordinalidad <> 1 OR ROW(
                alias_localizador_esquema_version,
                alias_localizador_clave_ref,
                alias_localizador_generacion_clave,
                alias_localizador_hmac,
                alias_huella_esquema_version,
                alias_huella_clave_ref,
                alias_huella_generacion_clave,
                alias_huella_hmac
            ) IS NOT DISTINCT FROM ROW(
                primario_localizador_esquema_version,
                primario_localizador_clave_ref,
                primario_localizador_generacion_clave,
                primario_localizador_hmac,
                primario_huella_esquema_version,
                primario_huella_clave_ref,
                primario_huella_generacion_clave,
                primario_huella_hmac
            ))
    )
);

-- Prueba durable de que un intento concreto no hizo COMMIT. Siempre enlaza
-- el control no terminal que se comprobo bajo bloqueo; la fila no_aplicado
-- posterior la referencia por ref+huella. Nunca contiene principal ni motivo.
CREATE TABLE vec_bolsa_convocatorias.prueba_desenlace_borrador (
    prueba_ref text PRIMARY KEY,
    huella_prueba_sha256 text NOT NULL,
    tipo_prueba text NOT NULL,
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision_control bigint NOT NULL,
    cercado_control bigint NOT NULL,
    decision_ref text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    prueba_canonica bytea NOT NULL,
    comprobada_en timestamptz(6) NOT NULL,
    UNIQUE (prueba_ref, huella_prueba_sha256),
    UNIQUE (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision_control
    ),
    FOREIGN KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision_control
    ) REFERENCES vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    ),
    CONSTRAINT prueba_desenlace_borrador_integra CHECK (
        prueba_ref ~ '^prueba-desenlace-borrador-[0-9a-f]{64}$'
        AND tipo_prueba IN ('ausencia_atomica','conflicto_cas')
        AND revision_control > 0 AND cercado_control > 0
        AND accion IN (
            'bolsa.convocatoria.borrador.crear',
            'bolsa.convocatoria.borrador.actualizar'
        )
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            recurso_ref, 512
        ) IS TRUE
        AND octet_length(prueba_canonica) BETWEEN 2 AND 1048576
        AND encode(sha256(prueba_canonica), 'hex') =
            huella_prueba_sha256
    )
);
ALTER TABLE vec_bolsa_convocatorias.diario_borrador_version
    ADD CONSTRAINT diario_borrador_prueba_desenlace_fk
    FOREIGN KEY (prueba_desenlace_ref, huella_prueba_desenlace_sha256)
    REFERENCES vec_bolsa_convocatorias.prueba_desenlace_borrador(
        prueba_ref, huella_prueba_sha256
    );

CREATE TABLE vec_bolsa_convocatorias.uso_decision_borrador (
    consumo_ref text PRIMARY KEY,
    decision_ref text NOT NULL UNIQUE,
    huella_decision_sha256 text NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    atestacion_estado text NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision_diario bigint NOT NULL,
    cercado bigint NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    huella_material_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    UNIQUE (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac,
        revision_diario
    ),
    FOREIGN KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac,
        revision_diario
    ) REFERENCES vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    ),
    FOREIGN KEY (
        decision_ref, atestacion_ref, atestacion_version,
        atestacion_estado, huella_decision_sha256,
        huella_atestacion_sha256
    ) REFERENCES vec_bolsa_convocatorias.atestacion_pdp_borrador(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_atestacion_sha256
    ),
    CONSTRAINT uso_decision_borrador_valido CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(
            consumo_ref, 512
        ) IS TRUE
        AND revision_diario > 0 AND cercado > 0
    )
);

CREATE TABLE vec_bolsa_convocatorias.sellado_motivo_borrador (
    token_consumo_ref text PRIMARY KEY,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    atestacion_estado text NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    materializador_ref text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    huella_material_sha256 text NOT NULL,
    dominio_criptografico text NOT NULL,
    generacion_clave bigint NOT NULL,
    clave_hmac_ref text NOT NULL,
    valor_hmac bytea NOT NULL,
    localizador_esquema_version integer NOT NULL,
    localizador_clave_ref text NOT NULL,
    localizador_generacion_clave bigint NOT NULL,
    localizador_hmac bytea NOT NULL,
    revision_diario bigint NOT NULL,
    cercado bigint NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    UNIQUE (atestacion_ref, atestacion_version),
    FOREIGN KEY (huella_material_sha256)
        REFERENCES vec_bolsa_convocatorias.material_borrador(
            huella_material_sha256
        ),
    FOREIGN KEY (
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac,
        revision_diario
    ) REFERENCES vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision
    ),
    CONSTRAINT sellado_motivo_borrador_valido CHECK (
        atestacion_estado = 'verificada' AND atestacion_version > 0
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_atestacion_sha256
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            token_consumo_ref, 512
        ) IS TRUE
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            materializador_ref, 512
        ) IS TRUE
        AND token_consumo_ref <> atestacion_ref
        AND materializador_ref <> atestacion_ref
        AND materializador_ref <> token_consumo_ref
        AND dominio_criptografico =
            'bolsa.convocatoria.motivo.v1'
        AND generacion_clave BETWEEN 1 AND 4294967295
        AND vec_bolsa_convocatorias.referencia_clave_hmac_valida(
            clave_hmac_ref, 'motivo'
        ) IS TRUE
        AND vec_bolsa_convocatorias.hmac_sha256_valido(valor_hmac) IS TRUE
        AND cercado > 0 AND emitida_en <= consumida_en
        AND consumida_en < valida_hasta
        AND valida_hasta - emitida_en <= interval '5 minutes'
    )
);

-- El agregado completo solo cruza como sobre cifrado. La proyeccion contiene
-- los campos necesarios para listado, filtros y CAS, sin documentos ni textos
-- extensos, por lo que no provoca N+1 ni lee blobs de hasta 32 MiB.
CREATE TABLE vec_bolsa_convocatorias.borrador_convocatoria_version (
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    referencia text NOT NULL,
    revision bigint NOT NULL,
    huella_estado_sha256 text NOT NULL,
    sobre_cifrado bytea NOT NULL,
    huella_sobre_cifrado_sha256 text NOT NULL,
    algoritmo_cifrado text NOT NULL,
    clave_cifrado_ref text NOT NULL,
    generacion_clave_cifrado bigint NOT NULL,
    nonce bytea NOT NULL,
    etiqueta_autenticacion bytea NOT NULL,
    atestacion_cifrado_ref text NOT NULL,
    huella_atestacion_cifrado_sha256 text NOT NULL,
    codigo_version_publica text NOT NULL,
    identificador_publico text NOT NULL,
    titulo text NOT NULL,
    tipo text NOT NULL,
    categorias text[] NOT NULL,
    expediente_ref text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text,
    numero_plazos integer NOT NULL,
    numero_requisitos integer NOT NULL,
    numero_documentos integer NOT NULL,
    numero_ayudas integer NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (convocatoria_id, secuencia, revision),
    UNIQUE (
        convocatoria_id, secuencia, revision, huella_estado_sha256
    ),
    CONSTRAINT borrador_version_identidad CHECK (
        secuencia > 0 AND revision > 0
        AND referencia = convocatoria_id || '#' || secuencia::text
        AND convocatoria_id !~ '[#@]'
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            convocatoria_id, 480
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_estado_sha256
        ) IS TRUE
    ),
    CONSTRAINT borrador_version_cifrado CHECK (
        octet_length(sobre_cifrado) BETWEEN 16 AND 33554432
        AND encode(sha256(sobre_cifrado), 'hex') =
            huella_sobre_cifrado_sha256
        AND algoritmo_cifrado = 'A256GCM'
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            clave_cifrado_ref, 512
        ) IS TRUE
        AND generacion_clave_cifrado BETWEEN 1 AND 4294967295
        AND octet_length(nonce) = 12
        AND octet_length(etiqueta_autenticacion) = 16
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            atestacion_cifrado_ref, 512
        ) IS TRUE
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_atestacion_cifrado_sha256
        ) IS TRUE
    ),
    CONSTRAINT borrador_version_proyeccion CHECK (
        codigo_version_publica ~ '^[a-z0-9][a-z0-9._-]{0,79}$'
        AND identificador_publico ~ '^[a-z0-9][a-z0-9-]{2,79}$'
        AND tipo ~ '^[a-z0-9][a-z0-9._-]{0,79}$'
        AND cardinality(categorias) BETWEEN 1 AND 1024
        AND vec_bolsa_convocatorias.lista_texto_canonica(categorias) IS TRUE
        AND titulo = btrim(titulo) AND octet_length(titulo) BETWEEN 1 AND 1000
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            expediente_ref, 512
        ) IS TRUE
        AND organizacion_ref ~ '^org_[a-z0-9]{16,80}$'
        AND (unidad_gestion_ref IS NULL OR
             unidad_gestion_ref ~ '^uni_[a-z0-9]{16,80}$')
        AND numero_plazos > 0 AND numero_requisitos >= 0
        AND numero_documentos > 0 AND numero_ayudas >= 0
        AND creada_en <= actualizada_en
        AND actualizada_en <= registrada_en
    )
);

CREATE TABLE vec_bolsa_convocatorias.borrador_convocatoria_actual (
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    revision bigint NOT NULL,
    huella_estado_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (convocatoria_id, secuencia),
    FOREIGN KEY (
        convocatoria_id, secuencia, revision, huella_estado_sha256
    ) REFERENCES vec_bolsa_convocatorias.borrador_convocatoria_version(
        convocatoria_id, secuencia, revision, huella_estado_sha256
    )
);

CREATE INDEX borrador_actual_orden
    ON vec_bolsa_convocatorias.borrador_convocatoria_actual(
        convocatoria_id, secuencia
    );
CREATE INDEX borrador_version_busqueda_texto
    ON vec_bolsa_convocatorias.borrador_convocatoria_version(
        organizacion_ref, unidad_gestion_ref, lower(titulo) text_pattern_ops,
        convocatoria_id
    );
CREATE INDEX borrador_version_categorias
    ON vec_bolsa_convocatorias.borrador_convocatoria_version
    USING gin(categorias);

CREATE TABLE vec_bolsa_convocatorias.auditoria_borrador (
    auditoria_ref text PRIMARY KEY,
    secuencia bigint NOT NULL UNIQUE,
    decision_ref text NOT NULL,
    transaccion_ref text NOT NULL UNIQUE,
    registro_canonico bytea NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_auditoria_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_borrador_integra CHECK (
        secuencia > 0
        AND octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(registro_canonico), 'hex') =
            huella_auditoria_sha256
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_anterior_sha256
        ) IS TRUE
    )
);

CREATE TABLE vec_bolsa_convocatorias.auditoria_borrador_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultima_huella_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_borrador_actual_huella CHECK (
        vec_bolsa_convocatorias.huella_sha256_valida(
            ultima_huella_sha256
        ) IS TRUE
    )
);
INSERT INTO vec_bolsa_convocatorias.auditoria_borrador_actual
VALUES (true, 0, repeat('0', 64), statement_timestamp());

CREATE TABLE vec_bolsa_convocatorias.outbox_borrador (
    evento_ref text PRIMARY KEY,
    secuencia bigint NOT NULL UNIQUE,
    tipo_evento text NOT NULL,
    transaccion_ref text NOT NULL UNIQUE,
    convocatoria_id text NOT NULL,
    secuencia_convocatoria bigint NOT NULL,
    revision bigint NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    evento_canonico bytea NOT NULL,
    huella_evento_sha256 text NOT NULL UNIQUE,
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (auditoria_ref)
        REFERENCES vec_bolsa_convocatorias.auditoria_borrador(auditoria_ref),
    CONSTRAINT outbox_borrador_integra CHECK (
        tipo_evento IN ('borrador_creado','borrador_actualizado')
        AND secuencia > 0 AND secuencia_convocatoria > 0 AND revision > 0
        AND octet_length(evento_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(evento_canonico), 'hex') = huella_evento_sha256
    )
);

-- Las lecturas internas tambien consumen una decision y dejan auditoria. El
-- cursor es una referencia opaca registrada y ligada a filtros y ambito.
CREATE TABLE vec_bolsa_convocatorias.uso_decision_lectura_borrador (
    consumo_ref text PRIMARY KEY,
    decision_ref text NOT NULL UNIQUE,
    huella_decision_sha256 text NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    atestacion_estado text NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text,
    huella_efecto_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (
        decision_ref, atestacion_ref, atestacion_version,
        atestacion_estado, huella_decision_sha256,
        huella_atestacion_sha256
    ) REFERENCES vec_bolsa_convocatorias.atestacion_pdp_borrador(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, huella_atestacion_sha256
    ),
    CONSTRAINT uso_lectura_borrador_valido CHECK (
        accion IN (
            'bolsa.convocatoria.borrador.listar',
            'bolsa.convocatoria.borrador.consultar'
        )
        AND organizacion_ref ~ '^org_[a-z0-9]{16,80}$'
        AND (unidad_gestion_ref IS NULL OR
             unidad_gestion_ref ~ '^uni_[a-z0-9]{16,80}$')
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_efecto_sha256
        ) IS TRUE
    )
);

CREATE TABLE vec_bolsa_convocatorias.auditoria_lectura_borrador (
    auditoria_ref text PRIMARY KEY,
    consumo_ref text NOT NULL UNIQUE,
    registro_canonico bytea NOT NULL,
    huella_auditoria_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (consumo_ref)
        REFERENCES vec_bolsa_convocatorias.uso_decision_lectura_borrador(
            consumo_ref
        ),
    CONSTRAINT auditoria_lectura_borrador_integra CHECK (
        octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(registro_canonico), 'hex') =
            huella_auditoria_sha256
    )
);

CREATE TABLE vec_bolsa_convocatorias.cursor_listado_borrador (
    cursor_ref text PRIMARY KEY,
    huella_filtros_sha256 text NOT NULL,
    organizacion_ref text NOT NULL,
    unidad_gestion_ref text,
    ultimo_convocatoria_id text NOT NULL,
    ultima_secuencia bigint NOT NULL,
    emitido_en timestamptz(6) NOT NULL,
    valido_hasta timestamptz(6) NOT NULL,
    consumo_origen_ref text NOT NULL,
    FOREIGN KEY (consumo_origen_ref)
        REFERENCES vec_bolsa_convocatorias.uso_decision_lectura_borrador(
            consumo_ref
        ),
    CONSTRAINT cursor_listado_borrador_valido CHECK (
        cursor_ref ~ '^cursor-borrador-[0-9a-f]{64}$'
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_filtros_sha256
        ) IS TRUE
        AND ultima_secuencia > 0 AND emitido_en < valido_hasta
        AND valido_hasta - emitido_en <= interval '15 minutes'
    )
);

-- Los punteros permiten el mismo microsegundo. La revision y la fila de
-- historia, no el timestamp, constituyen el orden total.
CREATE FUNCTION vec_bolsa_convocatorias.validar_avance_puntero_borrador()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.revision <> OLD.revision + 1
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de puntero de borrador invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.validar_avance_diario_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF ROW(NEW.localizador_esquema_version, NEW.localizador_clave_ref,
           NEW.localizador_generacion_clave, NEW.localizador_hmac)
       IS DISTINCT FROM
       ROW(OLD.localizador_esquema_version, OLD.localizador_clave_ref,
           OLD.localizador_generacion_clave, OLD.localizador_hmac)
       OR NEW.revision <> OLD.revision + 1
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de diario invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.validar_avance_auditoria_borrador()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.ultima_secuencia <> OLD.ultima_secuencia + 1
       OR NEW.actualizada_en < OLD.actualizada_en
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_convocatorias.auditoria_borrador AS a
            WHERE a.secuencia = NEW.ultima_secuencia
              AND a.huella_anterior_sha256 = OLD.ultima_huella_sha256
              AND a.huella_auditoria_sha256 = NEW.ultima_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de auditoria de borrador invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER borrador_actual_avance
    BEFORE UPDATE ON vec_bolsa_convocatorias.borrador_convocatoria_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_avance_puntero_borrador();
CREATE TRIGGER diario_borrador_actual_avance
    BEFORE UPDATE ON vec_bolsa_convocatorias.diario_borrador_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_avance_diario_actual();
CREATE TRIGGER auditoria_borrador_actual_avance
    BEFORE UPDATE ON vec_bolsa_convocatorias.auditoria_borrador_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_avance_auditoria_borrador();

-- Los wrappers SECURITY DEFINER no confian en current_user, que pasa a ser
-- el propietario. Reidentifican session_user como LOGIN tecnico endurecido,
-- con una unica membresia directa y sin atributos de evasion.
CREATE FUNCTION vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
    p_rol text, p_exclusiva boolean
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    identidad record;
    objetivo record;
    numero_membresias bigint;
    membresia_exacta boolean;
BEGIN
    IF p_exclusiva IS NOT TRUE OR p_rol NOT IN (
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno'
    ) THEN
        RETURN false;
    END IF;
    SELECT oid, rolcanlogin, rolinherit, rolsuper, rolcreatedb,
           rolcreaterole, rolreplication, rolbypassrls
      INTO identidad
      FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolcanlogin, rolinherit, rolsuper, rolcreatedb,
           rolcreaterole, rolreplication, rolbypassrls
      INTO objetivo
      FROM pg_catalog.pg_roles WHERE rolname = p_rol;
    SELECT count(*), COALESCE(bool_and(
               membresia.roleid = objetivo.oid
               AND membresia.admin_option IS FALSE
               AND membresia.inherit_option IS TRUE
               AND membresia.set_option IS TRUE
           ), false)
      INTO numero_membresias, membresia_exacta
      FROM pg_catalog.pg_auth_members AS membresia
     WHERE membresia.member = identidad.oid;
    RETURN identidad IS NOT NULL AND identidad.rolcanlogin
       AND identidad.rolinherit
       AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
       AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
       AND NOT identidad.rolbypassrls
       AND objetivo IS NOT NULL AND NOT objetivo.rolcanlogin
       AND objetivo.rolinherit
       AND NOT objetivo.rolsuper AND NOT objetivo.rolcreatedb
       AND NOT objetivo.rolcreaterole AND NOT objetivo.rolreplication
       AND NOT objetivo.rolbypassrls
       AND numero_membresias = 1 AND membresia_exacta
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = objetivo.oid
       );
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
    p_contexto bytea, p_lectura jsonb, p_referencia text,
    p_es_coleccion boolean
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    contexto jsonb;
    ambitos jsonb;
    esperado jsonb;
BEGIN
    IF p_contexto IS NULL OR p_lectura IS NULL
       OR p_es_coleccion IS NULL THEN
        RETURN false;
    END IF;
    contexto := convert_from(p_contexto, 'UTF8')::jsonb;
    ambitos := jsonb_strip_nulls(jsonb_build_object(
        'organizacion_ref', p_lectura ->> 'organizacion_ref',
        'unidad_gestion_ref', NULLIF(
            p_lectura ->> 'unidad_gestion_ref', ''
        )
    ));
    esperado := jsonb_build_object(
        'ambitos', ambitos, 'atributos', '{}'::jsonb
    );
    RETURN contexto = esperado
       AND p_lectura ->> 'recurso_ref' = p_referencia
       AND ((p_es_coleccion
             AND p_lectura ->> 'accion' =
                 'bolsa.convocatoria.borrador.listar'
             AND p_referencia = 'borradores:' ||
                 (p_lectura ->> 'organizacion_ref'))
            OR
            (NOT p_es_coleccion
             AND p_lectura ->> 'accion' =
                 'bolsa.convocatoria.borrador.consultar'));
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Recovery pre-PDP: no consume una decision nueva ni acepta principal. La
-- autoridad es exclusivamente la identidad de workload revalidada y el
-- testigo HMAC L/F exacto de una operacion ya reservada.
CREATE FUNCTION vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(
    p_identidad jsonb, p_estado_esperado text, p_revision_esperada bigint,
    p_cercado_esperado bigint, p_reconciliada_en timestamptz
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    prueba_desenlace_ref text,
    huella_prueba_desenlace_sha256 text,
    comprobada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad runtime de reconciliacion rechazada';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          p_identidad, p_estado_esperado, p_revision_esperada,
          p_cercado_esperado, p_reconciliada_en
      );
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_convocatorias.registrar_lectura_borrador_interna_v1(
    p_lectura jsonb, p_efecto_canonico bytea
)
RETURNS TABLE (
    consumo_ref text, auditoria_ref text,
    huella_auditoria_sha256 text, registrada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_consumo_ref text;
    v_auditoria_ref text;
    v_registro bytea;
    v_huella_efecto text;
    v_huella_auditoria text;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_lectura, ARRAY[
               'accion','atestacion_ref','atestacion_version',
               'decision_ref','estado_atestacion',
               'huella_atestacion_sha256','huella_decision_sha256',
               'organizacion_ref','recurso_ref','unidad_gestion_ref'
           ]
       ) IS NOT TRUE
       OR p_lectura ->> 'accion' NOT IN (
           'bolsa.convocatoria.borrador.listar',
           'bolsa.convocatoria.borrador.consultar'
       )
       OR (p_lectura ->> 'atestacion_version') !~ '^[1-9][0-9]{0,18}$'
       OR p_lectura ->> 'estado_atestacion' <> 'activa'
       OR p_lectura ->> 'organizacion_ref' !~ '^org_[a-z0-9]{16,80}$'
       OR (NULLIF(p_lectura ->> 'unidad_gestion_ref', '') IS NOT NULL
           AND p_lectura ->> 'unidad_gestion_ref' !~
               '^uni_[a-z0-9]{16,80}$')
       OR p_efecto_canonico IS NULL
       OR octet_length(p_efecto_canonico) NOT BETWEEN 2 AND 1048576
       OR NOT EXISTS (
           SELECT 1
             FROM vec_bolsa_convocatorias.atestacion_pdp_borrador AS a
             JOIN vec_bolsa_convocatorias.atestacion_autorizacion_version AS v
               ON v.decision_ref = a.decision_ref
              AND v.atestacion_ref = a.atestacion_ref
              AND v.version = a.version AND v.estado = a.estado
              AND v.huella_decision_sha256 = a.huella_decision_sha256
              AND v.huella_evidencia_sha256 =
                  a.huella_atestacion_sha256
             JOIN vec_bolsa_convocatorias.atestacion_autorizacion_actual AS c
               ON c.decision_ref = a.decision_ref
              AND c.atestacion_ref = a.atestacion_ref
              AND c.version = a.version AND c.estado = a.estado
            WHERE a.decision_ref = p_lectura ->> 'decision_ref'
              AND a.atestacion_ref = p_lectura ->> 'atestacion_ref'
              AND a.version = (p_lectura ->> 'atestacion_version')::bigint
              AND a.estado = p_lectura ->> 'estado_atestacion'
              AND a.huella_decision_sha256 =
                  p_lectura ->> 'huella_decision_sha256'
              AND a.huella_atestacion_sha256 =
                  p_lectura ->> 'huella_atestacion_sha256'
              AND v.valida_desde <= v_ahora
              AND v_ahora < v.valida_hasta
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'lectura de borrador no atestada';
    END IF;
    v_huella_efecto := encode(sha256(p_efecto_canonico), 'hex');
    v_consumo_ref := 'consumo-lectura-borrador-' || encode(sha256(convert_to(
        p_lectura ->> 'decision_ref' || ':' || v_huella_efecto, 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_convocatorias.uso_decision_lectura_borrador
    VALUES (
        v_consumo_ref, p_lectura ->> 'decision_ref',
        p_lectura ->> 'huella_decision_sha256',
        p_lectura ->> 'atestacion_ref',
        (p_lectura ->> 'atestacion_version')::bigint,
        p_lectura ->> 'estado_atestacion',
        p_lectura ->> 'huella_atestacion_sha256',
        p_lectura ->> 'accion', p_lectura ->> 'recurso_ref',
        p_lectura ->> 'organizacion_ref',
        NULLIF(p_lectura ->> 'unidad_gestion_ref', ''),
        v_huella_efecto, v_ahora
    );
    v_auditoria_ref := 'auditoria-lectura-borrador-' ||
        encode(sha256(convert_to(v_consumo_ref || ':' || v_huella_efecto,
        'UTF8')), 'hex');
    v_registro := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.auditoria-lectura.v1',
        'auditoria_ref', v_auditoria_ref,
        'consumo_ref', v_consumo_ref,
        'decision_ref', p_lectura ->> 'decision_ref',
        'accion', p_lectura ->> 'accion',
        'recurso_ref', p_lectura ->> 'recurso_ref',
        'organizacion_ref', p_lectura ->> 'organizacion_ref',
        'unidad_gestion_ref', NULLIF(
            p_lectura ->> 'unidad_gestion_ref', ''
        ),
        'huella_efecto_sha256', v_huella_efecto,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(v_registro), 'hex');
    INSERT INTO vec_bolsa_convocatorias.auditoria_lectura_borrador
    VALUES (
        v_auditoria_ref, v_consumo_ref, v_registro,
        v_huella_auditoria, v_ahora
    );
    RETURN QUERY SELECT v_consumo_ref, v_auditoria_ref,
        v_huella_auditoria, v_ahora;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.listar_borradores_interna_v1(
    p_selector jsonb, p_lectura jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_limite integer;
    v_cursor text;
    v_texto text;
    v_categoria text;
    v_ultimo_id text := '';
    v_ultima_secuencia bigint := 0;
    v_huella_filtros text;
    v_total bigint;
    v_elementos jsonb;
    v_hay_mas boolean;
    v_ultimo record;
    v_efecto bytea;
    v_lectura record;
    v_siguiente_cursor text;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_selector, ARRAY['categoria','cursor','limite','texto']
       ) IS NOT TRUE
       OR (p_selector ->> 'limite') !~ '^[1-9][0-9]?$'
       OR p_lectura ->> 'accion' <>
          'bolsa.convocatoria.borrador.listar' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'selector de borradores invalido';
    END IF;
    v_limite := (p_selector ->> 'limite')::integer;
    v_cursor := NULLIF(p_selector ->> 'cursor', '');
    v_texto := NULLIF(p_selector ->> 'texto', '');
    v_categoria := NULLIF(p_selector ->> 'categoria', '');
    IF v_limite NOT BETWEEN 1 AND 50
       OR (v_texto IS NOT NULL AND
           (v_texto <> btrim(v_texto) OR octet_length(v_texto) > 720))
       OR (v_categoria IS NOT NULL AND
           v_categoria !~ '^[a-z0-9][a-z0-9._-]{0,79}$') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'selector de borradores invalido';
    END IF;
    v_huella_filtros := encode(sha256(convert_to(jsonb_build_object(
        'limite', v_limite, 'texto', v_texto, 'categoria', v_categoria
    )::text, 'UTF8')), 'hex');
    IF v_cursor IS NOT NULL THEN
        SELECT ultimo_convocatoria_id, ultima_secuencia
          INTO STRICT v_ultimo_id, v_ultima_secuencia
          FROM vec_bolsa_convocatorias.cursor_listado_borrador
         WHERE cursor_ref = v_cursor
           AND huella_filtros_sha256 = v_huella_filtros
           AND organizacion_ref = p_lectura ->> 'organizacion_ref'
           AND unidad_gestion_ref IS NOT DISTINCT FROM
               NULLIF(p_lectura ->> 'unidad_gestion_ref', '')
           AND v_ahora < valido_hasta;
    END IF;

    SELECT count(*) INTO v_total
      FROM vec_bolsa_convocatorias.borrador_convocatoria_actual AS a
      JOIN vec_bolsa_convocatorias.borrador_convocatoria_version AS v
        ON v.convocatoria_id = a.convocatoria_id
       AND v.secuencia = a.secuencia AND v.revision = a.revision
       AND v.huella_estado_sha256 = a.huella_estado_sha256
     WHERE v.organizacion_ref = p_lectura ->> 'organizacion_ref'
       AND (NULLIF(p_lectura ->> 'unidad_gestion_ref', '') IS NULL
            OR v.unidad_gestion_ref =
               p_lectura ->> 'unidad_gestion_ref')
       AND (v_texto IS NULL OR lower(v.titulo) LIKE
            '%' || lower(v_texto) || '%' OR lower(v.identificador_publico)
            LIKE '%' || lower(v_texto) || '%')
       AND (v_categoria IS NULL OR v.categorias @> ARRAY[v_categoria]);

    WITH candidatas AS (
        SELECT v.*
          FROM vec_bolsa_convocatorias.borrador_convocatoria_actual AS a
          JOIN vec_bolsa_convocatorias.borrador_convocatoria_version AS v
            ON v.convocatoria_id = a.convocatoria_id
           AND v.secuencia = a.secuencia AND v.revision = a.revision
           AND v.huella_estado_sha256 = a.huella_estado_sha256
         WHERE v.organizacion_ref = p_lectura ->> 'organizacion_ref'
           AND (NULLIF(p_lectura ->> 'unidad_gestion_ref', '') IS NULL
                OR v.unidad_gestion_ref =
                   p_lectura ->> 'unidad_gestion_ref')
           AND ROW(v.convocatoria_id, v.secuencia) >
               ROW(v_ultimo_id, v_ultima_secuencia)
           AND (v_texto IS NULL OR lower(v.titulo) LIKE
                '%' || lower(v_texto) || '%'
                OR lower(v.identificador_publico) LIKE
                   '%' || lower(v_texto) || '%')
           AND (v_categoria IS NULL OR
                v.categorias @> ARRAY[v_categoria])
         ORDER BY v.convocatoria_id, v.secuencia
         LIMIT v_limite + 1
    ), pagina AS (
        SELECT * FROM candidatas
         ORDER BY convocatoria_id, secuencia LIMIT v_limite
    )
    SELECT COALESCE(jsonb_agg(jsonb_build_object(
               'referencia_estado', jsonb_build_object(
                   'referencia', referencia, 'revision', revision,
                   'huella_estado_sha256', huella_estado_sha256
               ),
               'etag', '"' || revision::text || '-' ||
                   huella_estado_sha256 || '"',
               'codigo_version_publica', codigo_version_publica,
               'identificador_publico', identificador_publico,
               'titulo', titulo, 'tipo', tipo,
               'categorias', to_jsonb(categorias),
               'expediente_ref', expediente_ref,
               'creada_en', to_char(creada_en AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
               'actualizada_en', to_char(actualizada_en AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
               'numero_plazos', numero_plazos,
               'numero_requisitos', numero_requisitos,
               'numero_documentos', numero_documentos,
               'numero_ayudas', numero_ayudas,
               'capacidades', jsonb_build_object(
                   'consultar', true, 'actualizar', false
               )
           ) ORDER BY convocatoria_id, secuencia), '[]'::jsonb),
           (SELECT count(*) > v_limite FROM candidatas)
      INTO v_elementos, v_hay_mas
      FROM pagina;
    IF v_hay_mas THEN
        SELECT v.convocatoria_id, v.secuencia INTO STRICT v_ultimo
          FROM vec_bolsa_convocatorias.borrador_convocatoria_actual AS a
          JOIN vec_bolsa_convocatorias.borrador_convocatoria_version AS v
            ON v.convocatoria_id = a.convocatoria_id
           AND v.secuencia = a.secuencia AND v.revision = a.revision
         WHERE v.organizacion_ref = p_lectura ->> 'organizacion_ref'
           AND (NULLIF(p_lectura ->> 'unidad_gestion_ref', '') IS NULL
                OR v.unidad_gestion_ref =
                   p_lectura ->> 'unidad_gestion_ref')
           AND ROW(v.convocatoria_id, v.secuencia) >
               ROW(v_ultimo_id, v_ultima_secuencia)
           AND (v_texto IS NULL OR lower(v.titulo) LIKE
                '%' || lower(v_texto) || '%'
                OR lower(v.identificador_publico) LIKE
                   '%' || lower(v_texto) || '%')
           AND (v_categoria IS NULL OR
                v.categorias @> ARRAY[v_categoria])
         ORDER BY v.convocatoria_id, v.secuencia
         OFFSET v_limite - 1 LIMIT 1;
    END IF;
    v_efecto := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.efecto-listado-borradores.v1',
        'huella_filtros_sha256', v_huella_filtros,
        'total', v_total, 'elementos', v_elementos
    )::text, 'UTF8');
    SELECT * INTO STRICT v_lectura
      FROM vec_bolsa_convocatorias.registrar_lectura_borrador_interna_v1(
          p_lectura, v_efecto
      );
    IF v_hay_mas THEN
        v_siguiente_cursor := 'cursor-borrador-' || encode(sha256(convert_to(
            v_lectura.consumo_ref || ':' || v_huella_filtros || ':' ||
            v_ultimo.convocatoria_id || ':' || v_ultimo.secuencia::text,
            'UTF8'
        )), 'hex');
        INSERT INTO vec_bolsa_convocatorias.cursor_listado_borrador
        VALUES (
            v_siguiente_cursor, v_huella_filtros,
            p_lectura ->> 'organizacion_ref',
            NULLIF(p_lectura ->> 'unidad_gestion_ref', ''),
            v_ultimo.convocatoria_id, v_ultimo.secuencia, v_ahora,
            v_ahora + interval '15 minutes', v_lectura.consumo_ref
        );
    END IF;
    RETURN jsonb_build_object(
        'esquema', 'vec.bolsa.borradores.lista.v1',
        'selector', jsonb_strip_nulls(jsonb_build_object(
            'limite', v_limite, 'cursor', v_cursor,
            'texto', v_texto, 'categoria', v_categoria
        )),
        'paginacion', jsonb_strip_nulls(jsonb_build_object(
            'limite', v_limite, 'total', v_total,
            'siguiente_cursor', v_siguiente_cursor
        )),
        'capacidades', jsonb_build_object(
            'consultar', true, 'crear', false
        ),
        'elementos', v_elementos
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    p_selector jsonb, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura,
           'borradores:' || (p_lectura ->> 'organizacion_ref'), true
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.listar',
           'coleccion_versiones_convocatoria_gobernada',
           p_lectura ->> 'recurso_ref',
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb, clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'listado de borradores no revalidado';
    END IF;
    RETURN vec_bolsa_convocatorias.listar_borradores_interna_v1(
        p_selector, p_lectura
    );
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.obtener_borrador_interna_v1(
    p_referencia text, p_lectura jsonb
)
RETURNS TABLE (
    metadatos jsonb, sobre_cifrado bytea,
    huella_sobre_cifrado_sha256 text, algoritmo_cifrado text,
    clave_cifrado_ref text, generacion_clave_cifrado bigint,
    nonce bytea, etiqueta_autenticacion bytea,
    atestacion_cifrado_ref text,
    huella_atestacion_cifrado_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_fila record;
    v_efecto bytea;
    v_lectura record;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           p_referencia, 512
       ) IS NOT TRUE
       OR p_referencia !~ '^.+#[1-9][0-9]{0,18}$'
       OR p_lectura ->> 'accion' <>
          'bolsa.convocatoria.borrador.consultar'
       OR p_lectura ->> 'recurso_ref' <> p_referencia THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'detalle de borrador invalido';
    END IF;
    SELECT v.* INTO STRICT v_fila
      FROM vec_bolsa_convocatorias.borrador_convocatoria_actual AS a
      JOIN vec_bolsa_convocatorias.borrador_convocatoria_version AS v
        ON v.convocatoria_id = a.convocatoria_id
       AND v.secuencia = a.secuencia AND v.revision = a.revision
       AND v.huella_estado_sha256 = a.huella_estado_sha256
     WHERE v.referencia = p_referencia
       AND v.organizacion_ref = p_lectura ->> 'organizacion_ref'
       AND (NULLIF(p_lectura ->> 'unidad_gestion_ref', '') IS NULL
            OR v.unidad_gestion_ref =
               p_lectura ->> 'unidad_gestion_ref');
    v_efecto := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.efecto-detalle-borrador.v1',
        'referencia', v_fila.referencia,
        'revision', v_fila.revision,
        'huella_estado_sha256', v_fila.huella_estado_sha256,
        'huella_sobre_cifrado_sha256',
            v_fila.huella_sobre_cifrado_sha256
    )::text, 'UTF8');
    SELECT * INTO STRICT v_lectura
      FROM vec_bolsa_convocatorias.registrar_lectura_borrador_interna_v1(
          p_lectura, v_efecto
      );
    RETURN QUERY SELECT jsonb_build_object(
        'referencia_estado', jsonb_build_object(
            'referencia', v_fila.referencia,
            'revision', v_fila.revision,
            'huella_estado_sha256', v_fila.huella_estado_sha256
        ),
        'etag', '"' || v_fila.revision::text || '-' ||
            v_fila.huella_estado_sha256 || '"',
        'codigo_version_publica', v_fila.codigo_version_publica,
        'identificador_publico', v_fila.identificador_publico,
        'ambito_lectura', jsonb_strip_nulls(jsonb_build_object(
            'organizacion_ref', v_fila.organizacion_ref,
            'unidad_gestion_ref', v_fila.unidad_gestion_ref
        )),
        'expediente_ref', v_fila.expediente_ref
    ), v_fila.sobre_cifrado, v_fila.huella_sobre_cifrado_sha256,
       v_fila.algoritmo_cifrado, v_fila.clave_cifrado_ref,
       v_fila.generacion_clave_cifrado, v_fila.nonce,
       v_fila.etiqueta_autenticacion, v_fila.atestacion_cifrado_ref,
       v_fila.huella_atestacion_cifrado_sha256;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    p_referencia text, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    metadatos jsonb, sobre_cifrado bytea,
    huella_sobre_cifrado_sha256 text, algoritmo_cifrado text,
    clave_cifrado_ref text, generacion_clave_cifrado bigint,
    nonce bytea, etiqueta_autenticacion bytea,
    atestacion_cifrado_ref text,
    huella_atestacion_cifrado_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura, p_referencia, false
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.consultar',
           'version_convocatoria_gobernada', p_referencia,
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb, clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'detalle de borrador no revalidado';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.obtener_borrador_interna_v1(
          p_referencia, p_lectura
      );
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
    p_revision_esperada bigint,
    p_cercado_esperado bigint,
    p_reserva jsonb,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    recibo jsonb, identidad jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_identidad jsonb := p_reserva -> 'identidad';
    v_l jsonb := v_identidad -> 'localizador';
    v_f jsonb := v_identidad -> 'huella_solicitud';
    v_d jsonb := p_reserva -> 'decision';
    v_a jsonb := v_d -> 'atestacion_pdp';
    v_actual record;
    v_revision_nueva bigint;
    v_cercado_nuevo bigint;
    v_consumo_ref text;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR p_revision_esperada < 1 OR p_cercado_esperado < 1
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_reserva, ARRAY[
               'accion','arrendamiento_inicia_en',
               'arrendamiento_vence_en','contexto_recurso_huella_sha256',
               'decision','esquema','huella_material_sha256','identidad',
               'identidades_consulta','recurso_ref','solicitada_en'
           ]
       ) IS NOT TRUE
       OR p_reserva ->> 'esquema' <>
          'vec.bolsa.convocatoria.reserva-decision.v2'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           v_identidad
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           p_reserva -> 'identidades_consulta'
       ) IS NOT TRUE
       OR (SELECT count(*)
             FROM jsonb_array_elements(
                      p_reserva -> 'identidades_consulta'
                  ) AS identidad(elemento)
            WHERE elemento = v_identidad) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reclamacion post-PDP no revalidada';
    END IF;
    SELECT h.*,
           x.alias_huella_esquema_version,
           x.alias_huella_clave_ref,
           x.alias_huella_generacion_clave,
           x.alias_huella_hmac
      INTO STRICT v_actual
      FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
      JOIN vec_bolsa_convocatorias.diario_borrador_actual AS c
        ON c.localizador_esquema_version =
           x.primario_localizador_esquema_version
       AND c.localizador_clave_ref = x.primario_localizador_clave_ref
       AND c.localizador_generacion_clave =
           x.primario_localizador_generacion_clave
       AND c.localizador_hmac = x.primario_localizador_hmac
      JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
        ON h.localizador_esquema_version = c.localizador_esquema_version
       AND h.localizador_clave_ref = c.localizador_clave_ref
       AND h.localizador_generacion_clave =
           c.localizador_generacion_clave
       AND h.localizador_hmac = c.localizador_hmac
       AND h.revision = c.revision
     WHERE x.alias_localizador_esquema_version =
           (v_l ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref = v_l ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (v_l ->> 'generacion_clave')::bigint
       AND x.alias_localizador_hmac =
           decode(v_l ->> 'hmac_sha256', 'hex')
     FOR UPDATE OF c;
    IF ROW(v_actual.alias_huella_esquema_version,
           v_actual.alias_huella_clave_ref,
           v_actual.alias_huella_generacion_clave,
           v_actual.alias_huella_hmac)
       IS DISTINCT FROM
       ROW((v_f ->> 'version_esquema')::integer,
           v_f ->> 'clave_ref',
           (v_f ->> 'generacion_clave')::bigint,
           decode(v_f ->> 'hmac_sha256', 'hex'))
       OR v_actual.huella_material_sha256 <>
          p_reserva ->> 'huella_material_sha256'
       OR v_actual.accion <> p_reserva ->> 'accion'
       OR v_actual.recurso_ref <> p_reserva ->> 'recurso_ref'
       OR v_actual.revision <> p_revision_esperada
       OR v_actual.cercado <> p_cercado_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva no reclamable o cercado obsoleto';
    END IF;
    -- Reclamacion en dos pasos: reconciliar debe haber cerrado primero el
    -- intento como no_aplicado con prueba durable y CAS revision+cercado.
    -- Solo una concesion PDP nueva puede abrir otra ventana.
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF v_actual.estado <> 'no_aplicado'
       OR v_ahora < v_actual.arrendamiento_vence_en
       OR NOT EXISTS (
           SELECT 1
             FROM vec_bolsa_convocatorias.prueba_desenlace_borrador AS p
            WHERE p.prueba_ref = v_actual.prueba_desenlace_ref
              AND p.huella_prueba_sha256 =
                  v_actual.huella_prueba_desenlace_sha256
              AND p.localizador_esquema_version =
                  v_actual.localizador_esquema_version
              AND p.localizador_clave_ref = v_actual.localizador_clave_ref
              AND p.localizador_generacion_clave =
                  v_actual.localizador_generacion_clave
              AND p.localizador_hmac = v_actual.localizador_hmac
              AND p.cercado_control < v_actual.cercado
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva requiere reconciliacion previa';
    END IF;
    IF vec_bolsa_convocatorias.validar_reserva_borrador_interna_v1(
           p_reserva, p_material_canonico, p_version_canonica,
           p_decision_canonica, p_contexto_recurso_canonico, v_ahora
       ) IS NOT TRUE
       OR (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz <
          v_actual.arrendamiento_vence_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reclamacion post-PDP no revalidada';
    END IF;
    v_revision_nueva := v_actual.revision + 1;
    v_cercado_nuevo := v_actual.cercado + 1;
    INSERT INTO vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision,
        huella_esquema_version, huella_clave_ref,
        huella_generacion_clave, huella_hmac, estado, cercado,
        arrendamiento_inicia_en, arrendamiento_vence_en, accion,
        huella_material_sha256, recurso_ref,
        contexto_recurso_huella_sha256, esquema_huella_decision,
        decision_ref, huella_decision_sha256, modulo_id, tipo_recurso,
        finalidad, version_rol_ref, version_rol_huella_sha256,
        control_vigencia_rol_revision, revision_catalogo_politicas,
        catalogo_politicas_huella_sha256, decision_verificada_en,
        decision_valida_hasta, atestacion_ref, atestacion_version,
        atestacion_estado, huella_atestacion_sha256, verificador_ref,
        atestacion_verificada_en, registrada_en,
        asignacion_ref, asignacion_huella_sha256,
        control_vigencia_version_rol_ref,
        control_vigencia_version_rol_huella_sha256, decision_emitida_en
    ) VALUES (
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_revision_nueva,
        v_actual.huella_esquema_version, v_actual.huella_clave_ref,
        v_actual.huella_generacion_clave, v_actual.huella_hmac,
        'reservado', v_cercado_nuevo,
        (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz,
        (p_reserva ->> 'arrendamiento_vence_en')::timestamptz,
        p_reserva ->> 'accion', p_reserva ->> 'huella_material_sha256',
        p_reserva ->> 'recurso_ref',
        p_reserva ->> 'contexto_recurso_huella_sha256',
        v_d ->> 'esquema_huella', v_d ->> 'decision_ref',
        v_d ->> 'huella_decision_sha256', v_d ->> 'modulo_id',
        v_d ->> 'tipo_recurso', v_d ->> 'finalidad',
        v_d ->> 'version_rol_ref', v_d ->> 'version_rol_huella_sha256',
        (v_d ->> 'control_vigencia_version_rol_revision')::bigint,
        (v_d ->> 'revision_catalogo_politicas')::bigint,
        v_d ->> 'catalogo_politicas_huella_sha256',
        (v_d ->> 'verificada_en')::timestamptz,
        (v_d ->> 'valida_hasta')::timestamptz,
        v_a ->> 'atestacion_ref', (v_a ->> 'version')::bigint,
        v_a ->> 'estado', v_a ->> 'huella_atestacion_sha256',
        v_a ->> 'verificador_ref',
        (v_a ->> 'verificada_en')::timestamptz, v_ahora,
        v_d ->> 'asignacion_ref', v_d ->> 'asignacion_huella_sha256',
        v_d ->> 'control_vigencia_version_rol_ref',
        v_d ->> 'control_vigencia_version_rol_huella_sha256',
        (v_d ->> 'emitida_en')::timestamptz
    );
    UPDATE vec_bolsa_convocatorias.diario_borrador_actual
       SET revision = v_revision_nueva, actualizada_en = v_ahora
     WHERE localizador_esquema_version =
           v_actual.localizador_esquema_version
       AND localizador_clave_ref = v_actual.localizador_clave_ref
       AND localizador_generacion_clave =
           v_actual.localizador_generacion_clave
       AND localizador_hmac = v_actual.localizador_hmac
       AND revision = v_actual.revision;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS perdido durante reclamacion';
    END IF;
    v_consumo_ref := 'consumo-decision-borrador-' || encode(sha256(convert_to(
        v_d ->> 'decision_ref' || ':' || encode(v_actual.localizador_hmac,
        'hex') || ':' || v_cercado_nuevo::text, 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_convocatorias.uso_decision_borrador VALUES (
        v_consumo_ref, v_d ->> 'decision_ref',
        v_d ->> 'huella_decision_sha256', v_a ->> 'atestacion_ref',
        (v_a ->> 'version')::bigint, v_a ->> 'estado',
        v_a ->> 'huella_atestacion_sha256',
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_revision_nueva, v_cercado_nuevo,
        v_actual.accion, v_actual.recurso_ref,
        v_actual.huella_material_sha256, v_ahora
    );
    RETURN QUERY SELECT 'reservado'::text, v_revision_nueva,
        v_cercado_nuevo,
        (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz,
        (p_reserva ->> 'arrendamiento_vence_en')::timestamptz,
        NULL::jsonb, v_identidad;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(
    p_revision_esperada bigint, p_cercado_esperado bigint,
    p_reserva jsonb, p_prueba jsonb, p_material_canonico bytea,
    p_version_canonica bytea, p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    recibo jsonb, identidad jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           p_reserva ->> 'accion', 'version_convocatoria_gobernada',
           p_reserva ->> 'recurso_ref',
           'gobierno_convocatorias',
           '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
           clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'decision de reclamacion no revalidada';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
          p_revision_esperada, p_cercado_esperado, p_reserva,
          p_material_canonico, p_version_canonica, p_decision_canonica,
          p_contexto_recurso_canonico
      );
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
    p_identidad jsonb, p_estado_esperado text, p_revision_esperada bigint,
    p_cercado_esperado bigint, p_reconciliada_en timestamptz
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    prueba_desenlace_ref text,
    huella_prueba_desenlace_sha256 text,
    comprobada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_l jsonb := p_identidad -> 'localizador';
    v_f jsonb := p_identidad -> 'huella_solicitud';
    v_ahora timestamptz;
    v_actual record;
    v_revision_nueva bigint;
    v_cercado_nuevo bigint;
    v_prueba_ref text;
    v_prueba jsonb;
    v_prueba_canonica bytea;
    v_huella_prueba text;
    v_ausencia_concluyente boolean;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           p_identidad
       ) IS NOT TRUE
       OR p_estado_esperado NOT IN (
           'reservado','en_curso','indeterminado','no_aplicado'
       )
       OR p_revision_esperada < 1 OR p_cercado_esperado < 1
       OR p_reconciliada_en IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'reconciliacion invalida';
    END IF;
    SELECT h.*,
           x.alias_huella_esquema_version,
           x.alias_huella_clave_ref,
           x.alias_huella_generacion_clave,
           x.alias_huella_hmac
      INTO STRICT v_actual
      FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
      JOIN vec_bolsa_convocatorias.diario_borrador_actual AS c
        ON c.localizador_esquema_version =
           x.primario_localizador_esquema_version
       AND c.localizador_clave_ref = x.primario_localizador_clave_ref
       AND c.localizador_generacion_clave =
           x.primario_localizador_generacion_clave
       AND c.localizador_hmac = x.primario_localizador_hmac
      JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
        ON h.localizador_esquema_version = c.localizador_esquema_version
       AND h.localizador_clave_ref = c.localizador_clave_ref
       AND h.localizador_generacion_clave =
           c.localizador_generacion_clave
       AND h.localizador_hmac = c.localizador_hmac
       AND h.revision = c.revision
     WHERE x.alias_localizador_esquema_version =
           (v_l ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref = v_l ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (v_l ->> 'generacion_clave')::bigint
       AND x.alias_localizador_hmac =
           decode(v_l ->> 'hmac_sha256', 'hex')
     FOR UPDATE OF c;
    IF ROW(v_actual.alias_huella_esquema_version,
           v_actual.alias_huella_clave_ref,
           v_actual.alias_huella_generacion_clave,
           v_actual.alias_huella_hmac)
       IS DISTINCT FROM
       ROW((v_f ->> 'version_esquema')::integer,
           v_f ->> 'clave_ref',
           (v_f ->> 'generacion_clave')::bigint,
           decode(v_f ->> 'hmac_sha256', 'hex'))
       THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'huella de solicitud distinta al reconciliar';
    END IF;
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF v_actual.estado = 'confirmado' THEN
        IF p_estado_esperado = 'no_aplicado'
           OR v_actual.revision <= p_revision_esperada
           OR v_actual.cercado <> p_cercado_esperado THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'confirmacion terminal incompatible con control';
        END IF;
        RETURN QUERY SELECT v_actual.estado, v_actual.revision,
            v_actual.cercado, v_actual.arrendamiento_inicia_en,
            v_actual.arrendamiento_vence_en, NULL::text, NULL::text,
            v_ahora, convert_from(v_actual.recibo_canonico, 'UTF8')::jsonb;
        RETURN;
    END IF;
    IF v_actual.estado = 'no_aplicado' THEN
        IF (p_estado_esperado = 'no_aplicado'
            AND (v_actual.revision <> p_revision_esperada
                 OR v_actual.cercado <> p_cercado_esperado))
           OR (p_estado_esperado <> 'no_aplicado'
               AND (v_actual.revision <= p_revision_esperada
                    OR v_actual.cercado <= p_cercado_esperado)) THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'no-aplicacion terminal incompatible con control';
        END IF;
        RETURN QUERY SELECT v_actual.estado, v_actual.revision,
            v_actual.cercado, v_actual.arrendamiento_inicia_en,
            v_actual.arrendamiento_vence_en,
            v_actual.prueba_desenlace_ref,
            v_actual.huella_prueba_desenlace_sha256,
            v_ahora, NULL::jsonb;
        RETURN;
    END IF;
    IF v_actual.estado <> p_estado_esperado
       OR v_actual.revision <> p_revision_esperada
       OR v_actual.cercado <> p_cercado_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'revision o cercado obsoleto al reconciliar';
    END IF;
    IF v_actual.estado NOT IN ('reservado','en_curso','indeterminado')
       OR v_ahora < v_actual.arrendamiento_vence_en THEN
        RETURN QUERY SELECT v_actual.estado, v_actual.revision,
            v_actual.cercado, v_actual.arrendamiento_inicia_en,
            v_actual.arrendamiento_vence_en, NULL::text, NULL::text,
            v_ahora, NULL::jsonb;
        RETURN;
    END IF;

    -- La ausencia solo es concluyente si no existe ninguna pieza que la
    -- transaccion atomica de confirmacion hubiera debido comprometer. Ante
    -- cualquier resto parcial se conserva exactamente el control no terminal.
    SELECT NOT EXISTS (
               SELECT 1
                 FROM vec_bolsa_convocatorias.sellado_motivo_borrador AS s
                WHERE s.localizador_esquema_version =
                      v_actual.localizador_esquema_version
                  AND s.localizador_clave_ref = v_actual.localizador_clave_ref
                  AND s.localizador_generacion_clave =
                      v_actual.localizador_generacion_clave
                  AND s.localizador_hmac = v_actual.localizador_hmac
                  AND s.cercado = v_actual.cercado
           )
       AND NOT EXISTS (
               SELECT 1
                 FROM vec_bolsa_convocatorias.auditoria_borrador AS a
                WHERE a.decision_ref = v_actual.decision_ref
           )
       AND NOT EXISTS (
               SELECT 1
                 FROM vec_bolsa_convocatorias.diario_borrador_version AS h
                WHERE h.localizador_esquema_version =
                      v_actual.localizador_esquema_version
                  AND h.localizador_clave_ref = v_actual.localizador_clave_ref
                  AND h.localizador_generacion_clave =
                      v_actual.localizador_generacion_clave
                  AND h.localizador_hmac = v_actual.localizador_hmac
                  AND h.estado = 'confirmado'
           )
      INTO v_ausencia_concluyente;
    IF v_ausencia_concluyente IS NOT TRUE THEN
        RETURN QUERY SELECT v_actual.estado, v_actual.revision,
            v_actual.cercado, v_actual.arrendamiento_inicia_en,
            v_actual.arrendamiento_vence_en, NULL::text, NULL::text,
            v_ahora, NULL::jsonb;
        RETURN;
    END IF;

    v_revision_nueva := v_actual.revision + 1;
    v_cercado_nuevo := v_actual.cercado + 1;
    v_prueba_ref := 'prueba-desenlace-borrador-' || encode(sha256(
        convert_to(encode(v_actual.localizador_hmac, 'hex') ||
            ':ausencia_atomica:' || v_actual.revision::text || ':' ||
            v_actual.cercado::text, 'UTF8')
    ), 'hex');
    v_prueba := jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.prueba-desenlace.v1',
        'prueba_ref', v_prueba_ref,
        'tipo_prueba', 'ausencia_atomica',
        'decision_ref', v_actual.decision_ref,
        'accion', v_actual.accion,
        'recurso_ref', v_actual.recurso_ref,
        'revision_control', v_actual.revision,
        'cercado_control', v_actual.cercado,
        'estado_resultante', 'no_aplicado',
        'comprobada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    v_prueba_canonica := convert_to(v_prueba::text, 'UTF8');
    v_huella_prueba := encode(sha256(v_prueba_canonica), 'hex');
    INSERT INTO vec_bolsa_convocatorias.prueba_desenlace_borrador
    VALUES (
        v_prueba_ref, v_huella_prueba, 'ausencia_atomica',
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_actual.revision, v_actual.cercado,
        v_actual.decision_ref, v_actual.accion, v_actual.recurso_ref,
        v_prueba_canonica, v_ahora
    );
    INSERT INTO vec_bolsa_convocatorias.diario_borrador_version
    SELECT
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_revision_nueva,
        v_actual.huella_esquema_version, v_actual.huella_clave_ref,
        v_actual.huella_generacion_clave, v_actual.huella_hmac,
        'no_aplicado', v_cercado_nuevo,
        v_actual.arrendamiento_inicia_en,
        v_actual.arrendamiento_vence_en, v_actual.accion,
        v_actual.huella_material_sha256, v_actual.recurso_ref,
        v_actual.contexto_recurso_huella_sha256,
        v_actual.esquema_huella_decision, v_actual.decision_ref,
        v_actual.huella_decision_sha256, v_actual.modulo_id,
        v_actual.tipo_recurso, v_actual.finalidad,
        v_actual.version_rol_ref, v_actual.version_rol_huella_sha256,
        v_actual.control_vigencia_rol_revision,
        v_actual.revision_catalogo_politicas,
        v_actual.catalogo_politicas_huella_sha256,
        v_actual.decision_verificada_en, v_actual.decision_valida_hasta,
        v_actual.atestacion_ref, v_actual.atestacion_version,
        v_actual.atestacion_estado, v_actual.huella_atestacion_sha256,
        v_actual.verificador_ref, v_actual.atestacion_verificada_en,
        NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        v_ahora, NULL, NULL, NULL, v_prueba_ref, v_huella_prueba,
        v_actual.asignacion_ref, v_actual.asignacion_huella_sha256,
        v_actual.control_vigencia_version_rol_ref,
        v_actual.control_vigencia_version_rol_huella_sha256,
        v_actual.decision_emitida_en;
    UPDATE vec_bolsa_convocatorias.diario_borrador_actual
       SET revision = v_revision_nueva,
           actualizada_en = v_ahora
     WHERE localizador_esquema_version =
           v_actual.localizador_esquema_version
       AND localizador_clave_ref = v_actual.localizador_clave_ref
       AND localizador_generacion_clave =
           v_actual.localizador_generacion_clave
       AND localizador_hmac = v_actual.localizador_hmac
       AND revision = v_actual.revision;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS perdido durante reconciliacion';
    END IF;
    RETURN QUERY SELECT 'no_aplicado'::text, v_revision_nueva,
        v_cercado_nuevo, v_actual.arrendamiento_inicia_en,
        v_actual.arrendamiento_vence_en, v_prueba_ref, v_huella_prueba,
        v_ahora, NULL::jsonb;
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
    p_confirmacion jsonb,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_sobre_cifrado bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_identidad jsonb := p_confirmacion -> 'identidad';
    v_l jsonb := v_identidad -> 'localizador';
    v_f jsonb := v_identidad -> 'huella_solicitud';
    v_sellado jsonb := p_confirmacion -> 'sellado_motivo';
    v_hmac_sellado jsonb := p_confirmacion -> 'sellado_motivo' -> 'hmac';
    v_envoltura jsonb := p_confirmacion -> 'envoltura_cifrado';
    v_proyeccion jsonb := p_confirmacion -> 'proyeccion_ligera';
    v_material jsonb;
    v_version jsonb;
    v_actual record;
    v_negocio record;
    v_revision_control bigint;
    v_revision_diario_nueva bigint;
    v_estado_esperado jsonb;
    v_estado_nuevo jsonb;
    v_partes_hmac text[];
    v_secuencia_auditoria bigint;
    v_huella_anterior text;
    v_transaccion_ref text;
    v_auditoria_ref text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
    v_evento_ref text;
    v_evento bytea;
    v_huella_evento text;
    v_tipo_evento text;
    v_confirmacion_solicitada timestamptz;
    v_recibo_ref text;
    v_recibo jsonb;
    v_recibo_canonico bytea;
    v_huella_recibo text;
    v_prueba_ref text;
    v_prueba jsonb;
    v_prueba_canonica bytea;
    v_huella_prueba text;
    v_cercado_terminal bigint;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_confirmacion, ARRAY[
               'cercado','envoltura_cifrado','esquema','identidad',
               'proyeccion_ligera','revision','sellado_motivo',
               'solicitada_en'
           ]
       ) IS NOT TRUE
       OR p_confirmacion ->> 'esquema' <>
          'vec.bolsa.convocatoria.confirmacion-borrador.v2'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           v_identidad
       ) IS NOT TRUE
       OR (p_confirmacion ->> 'revision') !~ '^[1-9][0-9]{0,18}$'
       OR (p_confirmacion ->> 'cercado') !~ '^[1-9][0-9]{0,18}$'
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_confirmacion ->> 'solicitada_en'
       ) IS NOT TRUE
       OR p_material_canonico IS NULL OR p_version_canonica IS NULL
       OR p_sobre_cifrado IS NULL
       OR octet_length(p_material_canonico) NOT BETWEEN 2 AND 1048576
       OR octet_length(p_version_canonica) NOT BETWEEN 2 AND 33554432
       OR octet_length(p_sobre_cifrado) NOT BETWEEN 16 AND 33554432 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'confirmacion de borrador invalida';
    END IF;
    BEGIN
        v_material := convert_from(p_material_canonico, 'UTF8')::jsonb;
        v_version := convert_from(p_version_canonica, 'UTF8')::jsonb;
        v_confirmacion_solicitada :=
            (p_confirmacion ->> 'solicitada_en')::timestamptz;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'confirmacion de borrador invalida';
    END;

    SELECT h.*,
           x.alias_huella_esquema_version,
           x.alias_huella_clave_ref,
           x.alias_huella_generacion_clave,
           x.alias_huella_hmac
      INTO v_actual
      FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
      JOIN vec_bolsa_convocatorias.diario_borrador_actual AS c
        ON c.localizador_esquema_version =
           x.primario_localizador_esquema_version
       AND c.localizador_clave_ref = x.primario_localizador_clave_ref
       AND c.localizador_generacion_clave =
           x.primario_localizador_generacion_clave
       AND c.localizador_hmac = x.primario_localizador_hmac
      JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
        ON h.localizador_esquema_version = c.localizador_esquema_version
       AND h.localizador_clave_ref = c.localizador_clave_ref
       AND h.localizador_generacion_clave =
           c.localizador_generacion_clave
       AND h.localizador_hmac = c.localizador_hmac
       AND h.revision = c.revision
     WHERE x.alias_localizador_esquema_version =
           (v_l ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref = v_l ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (v_l ->> 'generacion_clave')::bigint
       AND x.alias_localizador_hmac =
           decode(v_l ->> 'hmac_sha256', 'hex')
     FOR UPDATE OF c;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002',
            MESSAGE = 'confirmacion sin reserva previa';
    END IF;
    IF ROW(v_actual.alias_huella_esquema_version,
           v_actual.alias_huella_clave_ref,
           v_actual.alias_huella_generacion_clave,
           v_actual.alias_huella_hmac)
       IS DISTINCT FROM
       ROW((v_f ->> 'version_esquema')::integer,
           v_f ->> 'clave_ref',
           (v_f ->> 'generacion_clave')::bigint,
           decode(v_f ->> 'hmac_sha256', 'hex')) THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'localizador reutilizado con otra huella';
    END IF;
    IF v_actual.estado = 'confirmado' THEN
        RETURN QUERY SELECT
            'idempotencia_reutilizada'::text, v_actual.estado::text,
            v_actual.revision::bigint, v_actual.cercado::bigint,
            v_actual.transaccion_ref::text, v_actual.accion::text,
            v_actual.estado_principal_ref::text,
            v_actual.estado_principal_revision::bigint,
            v_actual.estado_principal_huella_sha256::text,
            v_actual.auditoria_ref::text,
            v_actual.huella_auditoria_sha256::text,
            v_actual.evento_outbox_ref::text,
            v_actual.huella_evento_outbox_sha256::text,
            v_actual.confirmada_en::timestamptz,
            convert_from(v_actual.recibo_canonico, 'UTF8')::jsonb;
        RETURN;
    ELSIF v_actual.estado = 'no_aplicado' THEN
        RETURN QUERY SELECT
            'idempotencia_reutilizada'::text, v_actual.estado::text,
            v_actual.revision::bigint, v_actual.cercado::bigint,
            NULL::text, v_actual.accion::text, NULL::text, NULL::bigint,
            NULL::text, NULL::text, NULL::text, NULL::text, NULL::text,
            v_actual.registrada_en::timestamptz, NULL::jsonb;
        RETURN;
    END IF;
    -- El bloqueo anterior puede haber esperado. La vigencia se decide con un
    -- reloj de base de datos nuevo y solo para un intento no terminal; un
    -- replay terminal conserva literalmente el veredicto ya registrado.
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF v_actual.estado <> 'reservado'
       OR v_actual.revision <> (p_confirmacion ->> 'revision')::bigint
       OR v_actual.cercado <> (p_confirmacion ->> 'cercado')::bigint
       OR v_confirmacion_solicitada < v_actual.arrendamiento_inicia_en
       OR v_confirmacion_solicitada >= v_actual.arrendamiento_vence_en
       OR v_ahora >= v_actual.arrendamiento_vence_en
       OR v_ahora >= v_actual.decision_valida_hasta
       OR encode(sha256(p_material_canonico), 'hex') <>
          v_actual.huella_material_sha256
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_convocatorias.material_borrador AS m
            WHERE m.huella_material_sha256 =
                  v_actual.huella_material_sha256
              AND m.material_canonico = p_material_canonico
       )
       OR vec_bolsa_convocatorias.atestacion_pdp_borrador_vigente(
              v_actual.decision_ref, v_actual.atestacion_ref,
              v_actual.atestacion_version, v_actual.atestacion_estado,
              v_actual.huella_decision_sha256,
              v_actual.huella_atestacion_sha256,
              v_actual.verificador_ref, v_actual.atestacion_verificada_en,
              v_ahora
          ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reserva no confirmable';
    END IF;

    v_estado_nuevo := v_material -> 'estado_principal_nuevo';
    v_estado_esperado := v_material -> 'estado_principal_esperado';
    v_partes_hmac := regexp_match(
        v_material ->> 'huella_motivo_hmac_sha256',
        '^hmac-sha256:(.+):([0-9a-f]{64})$'
    );
    IF v_material ->> 'accion' <> v_actual.accion
       OR v_estado_nuevo ->> 'referencia' <> v_actual.recurso_ref
       OR encode(sha256(p_version_canonica), 'hex') <>
          v_estado_nuevo ->> 'huella_estado_sha256'
       OR (v_version ->> 'id') || '#' ||
          (v_version ->> 'secuencia') <> v_actual.recurso_ref
       OR v_version ->> 'revision' <> v_estado_nuevo ->> 'revision'
       OR v_version ->> 'estado_gobierno' <> 'borrador' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material o version no coinciden con la reserva';
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_sellado, ARRAY[
               'accion','atestacion_emitida_en','atestacion_ref',
               'atestacion_valida_hasta','convocatoria_ref',
               'estado_atestacion','hmac','huella_atestacion_sha256',
               'materializador_ref',
               'token_consumo_ref','version_atestacion'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           v_hmac_sellado, ARRAY[
               'clave_hmac_ref','dominio_criptografico',
               'generacion_clave','valor_hmac_sha256'
           ]
       ) IS NOT TRUE
       OR v_sellado ->> 'accion' <> v_actual.accion
       OR v_sellado ->> 'convocatoria_ref' <> v_actual.recurso_ref
       OR v_hmac_sellado ->> 'dominio_criptografico' <>
          v_material ->> 'dominio_criptografico_motivo'
       OR v_hmac_sellado ->> 'generacion_clave' <>
          v_material ->> 'generacion_clave_motivo'
       OR v_hmac_sellado ->> 'clave_hmac_ref' <> v_partes_hmac[1]
       OR v_hmac_sellado ->> 'valor_hmac_sha256' <> v_partes_hmac[2]
       OR vec_bolsa_convocatorias.referencia_clave_hmac_valida(
           v_hmac_sellado ->> 'clave_hmac_ref', 'motivo'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_hmac_sellado ->> 'valor_hmac_sha256'
       ) IS NOT TRUE
       OR v_sellado ->> 'estado_atestacion' <> 'verificada'
       OR (v_sellado ->> 'version_atestacion') !~ '^[1-9][0-9]{0,9}$'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_sellado ->> 'huella_atestacion_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_sellado ->> 'atestacion_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_sellado ->> 'token_consumo_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_sellado ->> 'materializador_ref', 512
       ) IS NOT TRUE
       OR v_sellado ->> 'atestacion_ref' =
          v_sellado ->> 'token_consumo_ref'
       OR v_sellado ->> 'atestacion_ref' =
          v_sellado ->> 'materializador_ref'
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_sellado ->> 'atestacion_emitida_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_sellado ->> 'atestacion_valida_hasta'
       ) IS NOT TRUE
       OR (v_sellado ->> 'atestacion_emitida_en')::timestamptz >
          v_confirmacion_solicitada
       OR v_confirmacion_solicitada >=
          (v_sellado ->> 'atestacion_valida_hasta')::timestamptz
       OR v_ahora >=
          (v_sellado ->> 'atestacion_valida_hasta')::timestamptz
       OR (v_sellado ->> 'atestacion_valida_hasta')::timestamptz -
          (v_sellado ->> 'atestacion_emitida_en')::timestamptz >
          interval '5 minutes' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'sellado HSM/KMS de motivo no valido';
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_envoltura, ARRAY[
               'algoritmo','atestacion_cifrado_ref','clave_cifrado_ref',
               'etiqueta_autenticacion_hex','generacion_clave',
               'huella_atestacion_cifrado_sha256',
               'huella_sobre_cifrado_sha256','nonce_hex'
           ]
       ) IS NOT TRUE
       OR v_envoltura ->> 'algoritmo' <> 'A256GCM'
       OR (v_envoltura ->> 'generacion_clave') !~ '^[1-9][0-9]{0,9}$'
       OR (v_envoltura ->> 'nonce_hex') !~ '^[0-9a-f]{24}$'
       OR (v_envoltura ->> 'etiqueta_autenticacion_hex') !~
          '^[0-9a-f]{32}$'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_envoltura ->> 'huella_sobre_cifrado_sha256'
       ) IS NOT TRUE
       OR encode(sha256(p_sobre_cifrado), 'hex') <>
          v_envoltura ->> 'huella_sobre_cifrado_sha256'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_envoltura ->> 'huella_atestacion_cifrado_sha256'
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'envoltura cifrada de borrador invalida';
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_proyeccion, ARRAY[
               'actualizada_en','categorias','codigo_version_publica',
               'convocatoria_id','creada_en','expediente_ref',
               'huella_estado_sha256','identificador_publico',
               'numero_ayudas','numero_documentos','numero_plazos',
               'numero_requisitos','organizacion_ref','referencia','revision',
               'secuencia','tipo','titulo','unidad_gestion_ref'
           ]
       ) IS NOT TRUE
       OR v_proyeccion ->> 'convocatoria_id' <> v_version ->> 'id'
       OR v_proyeccion ->> 'secuencia' <> v_version ->> 'secuencia'
       OR v_proyeccion ->> 'referencia' <> v_actual.recurso_ref
       OR v_proyeccion ->> 'revision' <> v_version ->> 'revision'
       OR v_proyeccion ->> 'huella_estado_sha256' <>
          v_estado_nuevo ->> 'huella_estado_sha256'
       OR v_proyeccion ->> 'codigo_version_publica' <>
          v_version ->> 'codigo_version_publica'
       OR v_proyeccion ->> 'identificador_publico' <>
          v_version -> 'contenido' ->> 'identificador_publico'
       OR v_proyeccion ->> 'titulo' <>
          v_version -> 'contenido' ->> 'titulo'
       OR v_proyeccion ->> 'tipo' <> v_version -> 'contenido' ->> 'tipo'
       OR v_proyeccion -> 'categorias' <>
          v_version -> 'contenido' -> 'categorias'
       OR v_proyeccion ->> 'expediente_ref' <>
          v_version ->> 'expediente_ref'
       OR v_proyeccion ->> 'organizacion_ref' <>
          v_version -> 'ambito_organizativo' ->> 'organizacion_ref'
       OR COALESCE(v_proyeccion ->> 'unidad_gestion_ref', '') <>
          COALESCE(
              v_version -> 'ambito_organizativo' ->> 'unidad_gestion_ref', ''
          )
       OR (v_proyeccion ->> 'numero_plazos')::integer <>
          jsonb_array_length(v_version -> 'contenido' -> 'plazos')
       OR (v_proyeccion ->> 'numero_requisitos')::integer <>
          jsonb_array_length(v_version -> 'contenido' -> 'requisitos')
       OR (v_proyeccion ->> 'numero_documentos')::integer <>
          jsonb_array_length(v_version -> 'contenido' -> 'documentos')
       OR (v_proyeccion ->> 'numero_ayudas')::integer <>
          jsonb_array_length(v_version -> 'contenido' -> 'ayuda')
       OR v_proyeccion ->> 'creada_en' <> v_version ->> 'creada_en'
       OR v_proyeccion ->> 'actualizada_en' <>
          (CASE WHEN (v_version ->> 'revision')::bigint = 1
                THEN v_version ->> 'creada_en'
                ELSE v_version ->> 'ultima_modificacion_en' END)
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_proyeccion ->> 'creada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_proyeccion ->> 'actualizada_en'
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyeccion ligera no coincide con el agregado';
    END IF;

    -- Primero se hace visible en la historia el estado en_curso. Sigue dentro
    -- de la misma transaccion: un rollback revierte este paso y todos los demas.
    v_revision_control := v_actual.revision + 1;
    INSERT INTO vec_bolsa_convocatorias.diario_borrador_version
    SELECT
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_revision_control,
        v_actual.huella_esquema_version, v_actual.huella_clave_ref,
        v_actual.huella_generacion_clave, v_actual.huella_hmac,
        'en_curso', v_actual.cercado,
        v_actual.arrendamiento_inicia_en,
        v_actual.arrendamiento_vence_en, v_actual.accion,
        v_actual.huella_material_sha256, v_actual.recurso_ref,
        v_actual.contexto_recurso_huella_sha256,
        v_actual.esquema_huella_decision, v_actual.decision_ref,
        v_actual.huella_decision_sha256, v_actual.modulo_id,
        v_actual.tipo_recurso, v_actual.finalidad,
        v_actual.version_rol_ref, v_actual.version_rol_huella_sha256,
        v_actual.control_vigencia_rol_revision,
        v_actual.revision_catalogo_politicas,
        v_actual.catalogo_politicas_huella_sha256,
        v_actual.decision_verificada_en, v_actual.decision_valida_hasta,
        v_actual.atestacion_ref, v_actual.atestacion_version,
        v_actual.atestacion_estado, v_actual.huella_atestacion_sha256,
        v_actual.verificador_ref, v_actual.atestacion_verificada_en,
        NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, v_ahora,
        NULL, NULL, NULL, NULL, NULL,
        v_actual.asignacion_ref, v_actual.asignacion_huella_sha256,
        v_actual.control_vigencia_version_rol_ref,
        v_actual.control_vigencia_version_rol_huella_sha256,
        v_actual.decision_emitida_en;
    UPDATE vec_bolsa_convocatorias.diario_borrador_actual
       SET revision = v_revision_control, actualizada_en = v_ahora
     WHERE localizador_esquema_version =
           v_actual.localizador_esquema_version
       AND localizador_clave_ref = v_actual.localizador_clave_ref
       AND localizador_generacion_clave =
           v_actual.localizador_generacion_clave
       AND localizador_hmac = v_actual.localizador_hmac
       AND revision = v_actual.revision;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de diario perdido al confirmar';
    END IF;

    SELECT a.* INTO v_negocio
      FROM vec_bolsa_convocatorias.borrador_convocatoria_actual AS a
     WHERE a.convocatoria_id = v_proyeccion ->> 'convocatoria_id'
       AND a.secuencia = (v_proyeccion ->> 'secuencia')::bigint
     FOR UPDATE OF a;
    IF v_actual.accion = 'bolsa.convocatoria.borrador.crear' THEN
        IF FOUND THEN
            v_negocio := NULL;
        END IF;
    ELSE
        IF NOT FOUND
           OR v_negocio.revision <>
              (v_estado_esperado ->> 'revision')::bigint
           OR v_negocio.huella_estado_sha256 <>
              v_estado_esperado ->> 'huella_estado_sha256' THEN
            v_negocio := NULL;
        END IF;
    END IF;
    IF v_negocio IS NULL AND
       (v_actual.accion = 'bolsa.convocatoria.borrador.actualizar'
        OR FOUND) THEN
        v_revision_diario_nueva := v_revision_control + 1;
        v_cercado_terminal := v_actual.cercado + 1;
        v_prueba_ref := 'prueba-desenlace-borrador-' || encode(sha256(
            convert_to(v_actual.decision_ref || ':conflicto_cas:' ||
                v_revision_control::text || ':' ||
                v_cercado_terminal::text, 'UTF8')
        ), 'hex');
        v_prueba := jsonb_build_object(
            'esquema', 'vec.bolsa.convocatoria.prueba-desenlace.v1',
            'prueba_ref', v_prueba_ref,
            'tipo_prueba', 'conflicto_cas',
            'decision_ref', v_actual.decision_ref,
            'accion', v_actual.accion,
            'recurso_ref', v_actual.recurso_ref,
            'revision_control', v_revision_control,
            'cercado_control', v_actual.cercado,
            'estado_resultante', 'no_aplicado',
            'comprobada_en', to_char(v_ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        );
        v_prueba_canonica := convert_to(v_prueba::text, 'UTF8');
        v_huella_prueba := encode(sha256(v_prueba_canonica), 'hex');
        INSERT INTO vec_bolsa_convocatorias.prueba_desenlace_borrador
        VALUES (
            v_prueba_ref, v_huella_prueba, 'conflicto_cas',
            v_actual.localizador_esquema_version,
            v_actual.localizador_clave_ref,
            v_actual.localizador_generacion_clave,
            v_actual.localizador_hmac, v_revision_control,
            v_actual.cercado, v_actual.decision_ref, v_actual.accion,
            v_actual.recurso_ref, v_prueba_canonica, v_ahora
        );
        INSERT INTO vec_bolsa_convocatorias.diario_borrador_version
        SELECT
            v_actual.localizador_esquema_version,
            v_actual.localizador_clave_ref,
            v_actual.localizador_generacion_clave,
            v_actual.localizador_hmac, v_revision_diario_nueva,
            v_actual.huella_esquema_version, v_actual.huella_clave_ref,
            v_actual.huella_generacion_clave, v_actual.huella_hmac,
            'no_aplicado', v_cercado_terminal,
            v_actual.arrendamiento_inicia_en,
            v_actual.arrendamiento_vence_en, v_actual.accion,
            v_actual.huella_material_sha256, v_actual.recurso_ref,
            v_actual.contexto_recurso_huella_sha256,
            v_actual.esquema_huella_decision, v_actual.decision_ref,
            v_actual.huella_decision_sha256, v_actual.modulo_id,
            v_actual.tipo_recurso, v_actual.finalidad,
            v_actual.version_rol_ref, v_actual.version_rol_huella_sha256,
            v_actual.control_vigencia_rol_revision,
            v_actual.revision_catalogo_politicas,
            v_actual.catalogo_politicas_huella_sha256,
            v_actual.decision_verificada_en,
            v_actual.decision_valida_hasta, v_actual.atestacion_ref,
            v_actual.atestacion_version, v_actual.atestacion_estado,
            v_actual.huella_atestacion_sha256, v_actual.verificador_ref,
            v_actual.atestacion_verificada_en,
            NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, v_ahora,
            NULL, NULL, NULL, v_prueba_ref, v_huella_prueba,
            v_actual.asignacion_ref, v_actual.asignacion_huella_sha256,
            v_actual.control_vigencia_version_rol_ref,
            v_actual.control_vigencia_version_rol_huella_sha256,
            v_actual.decision_emitida_en;
        UPDATE vec_bolsa_convocatorias.diario_borrador_actual
           SET revision = v_revision_diario_nueva,
               actualizada_en = v_ahora
         WHERE localizador_esquema_version =
               v_actual.localizador_esquema_version
           AND localizador_clave_ref = v_actual.localizador_clave_ref
           AND localizador_generacion_clave =
               v_actual.localizador_generacion_clave
           AND localizador_hmac = v_actual.localizador_hmac
           AND revision = v_revision_control;
        RETURN QUERY SELECT
            'conflicto_cas'::text, 'no_aplicado'::text,
            v_revision_diario_nueva, v_cercado_terminal,
            NULL::text, v_actual.accion, NULL::text, NULL::bigint,
            NULL::text, NULL::text, NULL::text, NULL::text, NULL::text,
            v_ahora, NULL::jsonb;
        RETURN;
    END IF;

    INSERT INTO vec_bolsa_convocatorias.sellado_motivo_borrador VALUES (
        v_sellado ->> 'token_consumo_ref',
        v_sellado ->> 'atestacion_ref',
        (v_sellado ->> 'version_atestacion')::bigint,
        v_sellado ->> 'estado_atestacion',
        v_sellado ->> 'huella_atestacion_sha256',
        v_sellado ->> 'materializador_ref', v_actual.accion,
        v_actual.recurso_ref, v_actual.huella_material_sha256,
        v_hmac_sellado ->> 'dominio_criptografico',
        (v_hmac_sellado ->> 'generacion_clave')::bigint,
        v_hmac_sellado ->> 'clave_hmac_ref',
        decode(v_hmac_sellado ->> 'valor_hmac_sha256', 'hex'),
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_actual.revision, v_actual.cercado,
        (v_sellado ->> 'atestacion_emitida_en')::timestamptz,
        (v_sellado ->> 'atestacion_valida_hasta')::timestamptz,
        v_ahora
    );

    INSERT INTO vec_bolsa_convocatorias.borrador_convocatoria_version(
        convocatoria_id, secuencia, referencia, revision,
        huella_estado_sha256, sobre_cifrado,
        huella_sobre_cifrado_sha256, algoritmo_cifrado,
        clave_cifrado_ref, generacion_clave_cifrado, nonce,
        etiqueta_autenticacion, atestacion_cifrado_ref,
        huella_atestacion_cifrado_sha256, codigo_version_publica,
        identificador_publico, titulo, tipo, categorias, expediente_ref,
        organizacion_ref, unidad_gestion_ref, numero_plazos,
        numero_requisitos, numero_documentos, numero_ayudas, creada_en,
        actualizada_en, registrada_en
    ) VALUES (
        v_proyeccion ->> 'convocatoria_id',
        (v_proyeccion ->> 'secuencia')::bigint,
        v_proyeccion ->> 'referencia',
        (v_proyeccion ->> 'revision')::bigint,
        v_proyeccion ->> 'huella_estado_sha256', p_sobre_cifrado,
        v_envoltura ->> 'huella_sobre_cifrado_sha256',
        v_envoltura ->> 'algoritmo', v_envoltura ->> 'clave_cifrado_ref',
        (v_envoltura ->> 'generacion_clave')::bigint,
        decode(v_envoltura ->> 'nonce_hex', 'hex'),
        decode(v_envoltura ->> 'etiqueta_autenticacion_hex', 'hex'),
        v_envoltura ->> 'atestacion_cifrado_ref',
        v_envoltura ->> 'huella_atestacion_cifrado_sha256',
        v_proyeccion ->> 'codigo_version_publica',
        v_proyeccion ->> 'identificador_publico',
        v_proyeccion ->> 'titulo', v_proyeccion ->> 'tipo',
        ARRAY(SELECT jsonb_array_elements_text(v_proyeccion -> 'categorias')),
        v_proyeccion ->> 'expediente_ref',
        v_proyeccion ->> 'organizacion_ref',
        NULLIF(v_proyeccion ->> 'unidad_gestion_ref', ''),
        (v_proyeccion ->> 'numero_plazos')::integer,
        (v_proyeccion ->> 'numero_requisitos')::integer,
        (v_proyeccion ->> 'numero_documentos')::integer,
        (v_proyeccion ->> 'numero_ayudas')::integer,
        (v_proyeccion ->> 'creada_en')::timestamptz,
        (v_proyeccion ->> 'actualizada_en')::timestamptz, v_ahora
    );
    IF v_actual.accion = 'bolsa.convocatoria.borrador.crear' THEN
        INSERT INTO vec_bolsa_convocatorias.borrador_convocatoria_actual
        VALUES (
            v_proyeccion ->> 'convocatoria_id',
            (v_proyeccion ->> 'secuencia')::bigint,
            (v_proyeccion ->> 'revision')::bigint,
            v_proyeccion ->> 'huella_estado_sha256', v_ahora
        );
        v_tipo_evento := 'borrador_creado';
    ELSE
        UPDATE vec_bolsa_convocatorias.borrador_convocatoria_actual
           SET revision = (v_proyeccion ->> 'revision')::bigint,
               huella_estado_sha256 =
                   v_proyeccion ->> 'huella_estado_sha256',
               actualizada_en = v_ahora
         WHERE convocatoria_id = v_proyeccion ->> 'convocatoria_id'
           AND secuencia = (v_proyeccion ->> 'secuencia')::bigint
           AND revision = (v_estado_esperado ->> 'revision')::bigint
           AND huella_estado_sha256 =
               v_estado_esperado ->> 'huella_estado_sha256';
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS perdido durante la confirmacion';
        END IF;
        v_tipo_evento := 'borrador_actualizado';
    END IF;

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_secuencia_auditoria, v_huella_anterior
      FROM vec_bolsa_convocatorias.auditoria_borrador_actual
     WHERE control_id FOR UPDATE;
    v_transaccion_ref := 'transaccion-borrador-' || encode(sha256(convert_to(
        encode(v_actual.localizador_hmac, 'hex') || ':' ||
        v_actual.cercado::text || ':' ||
        to_char(v_ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), 'UTF8'
    )), 'hex');
    v_auditoria_ref := 'auditoria-borrador-' || encode(sha256(convert_to(
        (v_secuencia_auditoria + 1)::text || ':' || v_transaccion_ref ||
        ':' || v_huella_anterior, 'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.auditoria-borrador.v2',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_secuencia_auditoria + 1,
        'huella_anterior_sha256', v_huella_anterior,
        'transaccion_ref', v_transaccion_ref,
        'decision_ref', v_actual.decision_ref,
        'accion', v_actual.accion,
        'recurso_ref', v_actual.recurso_ref,
        'revision', (v_estado_nuevo ->> 'revision')::bigint,
        'huella_estado_sha256', v_estado_nuevo ->> 'huella_estado_sha256',
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(v_registro_auditoria), 'hex');
    INSERT INTO vec_bolsa_convocatorias.auditoria_borrador VALUES (
        v_auditoria_ref, v_secuencia_auditoria + 1,
        v_actual.decision_ref, v_transaccion_ref, v_registro_auditoria,
        v_huella_anterior, v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_convocatorias.auditoria_borrador_actual
       SET ultima_secuencia = v_secuencia_auditoria + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE control_id;

    v_evento_ref := 'outbox-borrador-' || encode(sha256(convert_to(
        v_transaccion_ref || ':' || v_auditoria_ref, 'UTF8'
    )), 'hex');
    v_evento := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.outbox-borrador.v2',
        'evento_ref', v_evento_ref,
        'secuencia', v_secuencia_auditoria + 1,
        'tipo_evento', v_tipo_evento,
        'transaccion_ref', v_transaccion_ref,
        'recurso_ref', v_actual.recurso_ref,
        'revision', (v_estado_nuevo ->> 'revision')::bigint,
        'huella_estado_sha256', v_estado_nuevo ->> 'huella_estado_sha256',
        'auditoria_ref', v_auditoria_ref,
        'huella_auditoria_sha256', v_huella_auditoria,
        'creada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_evento := encode(sha256(v_evento), 'hex');
    INSERT INTO vec_bolsa_convocatorias.outbox_borrador VALUES (
        v_evento_ref, v_secuencia_auditoria + 1, v_tipo_evento,
        v_transaccion_ref, v_proyeccion ->> 'convocatoria_id',
        (v_proyeccion ->> 'secuencia')::bigint,
        (v_proyeccion ->> 'revision')::bigint,
        v_auditoria_ref, v_evento, v_huella_evento, v_ahora
    );

    v_revision_diario_nueva := v_revision_control + 1;
    v_recibo_ref := 'recibo-borrador-' || encode(sha256(convert_to(
        v_transaccion_ref || ':' || encode(v_actual.localizador_hmac, 'hex')
        || ':' || v_revision_diario_nueva::text || ':' ||
        v_actual.cercado::text, 'UTF8'
    )), 'hex');
    v_recibo := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.borrador.recibo.v2',
        'recibo_ref', v_recibo_ref,
        'transaccion_ref', v_transaccion_ref,
        'accion', v_actual.accion,
        'estado_principal', v_estado_nuevo,
        'identidad', v_identidad,
        'decision', jsonb_build_object(
            'esquema_huella', v_actual.esquema_huella_decision,
            'decision_ref', v_actual.decision_ref,
            'huella_decision_sha256', v_actual.huella_decision_sha256,
            'accion', v_actual.accion,
            'recurso_ref', v_actual.recurso_ref,
            'modulo_id', v_actual.modulo_id,
            'tipo_recurso', v_actual.tipo_recurso,
            'contexto_recurso_huella_sha256',
                v_actual.contexto_recurso_huella_sha256,
            'finalidad', v_actual.finalidad,
            'asignacion_ref', v_actual.asignacion_ref,
            'asignacion_huella_sha256',
                v_actual.asignacion_huella_sha256,
            'version_rol_ref', v_actual.version_rol_ref,
            'version_rol_huella_sha256',
                v_actual.version_rol_huella_sha256,
            'control_vigencia_version_rol_ref',
                v_actual.control_vigencia_version_rol_ref,
            'control_vigencia_version_rol_revision',
                v_actual.control_vigencia_rol_revision,
            'control_vigencia_version_rol_huella_sha256',
                v_actual.control_vigencia_version_rol_huella_sha256,
            'revision_catalogo_politicas',
                v_actual.revision_catalogo_politicas,
            'catalogo_politicas_huella_sha256',
                v_actual.catalogo_politicas_huella_sha256,
            'emitida_en', to_char(
                v_actual.decision_emitida_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'verificada_en', to_char(
                v_actual.decision_verificada_en AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'valida_hasta', to_char(
                v_actual.decision_valida_hasta AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'atestacion_pdp', jsonb_build_object(
                'decision_ref', v_actual.decision_ref,
                'atestacion_ref', v_actual.atestacion_ref,
                'version', v_actual.atestacion_version,
                'estado', v_actual.atestacion_estado,
                'huella_atestacion_sha256',
                    v_actual.huella_atestacion_sha256,
                'verificador_ref', v_actual.verificador_ref,
                'verificada_en', to_char(
                    v_actual.atestacion_verificada_en AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                )
            )
        ),
        'sellado_motivo', v_sellado,
        'revision_confirmada', v_revision_diario_nueva,
        'cercado_confirmado', v_actual.cercado,
        'arrendamiento_inicia_en', to_char(
            v_actual.arrendamiento_inicia_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'arrendamiento_vence_en', to_char(
            v_actual.arrendamiento_vence_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'auditoria_ref', v_auditoria_ref,
        'huella_auditoria_sha256', v_huella_auditoria,
        'evento_outbox_ref', v_evento_ref,
        'huella_evento_outbox_sha256', v_huella_evento,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    v_recibo_canonico := convert_to(v_recibo::text, 'UTF8');
    v_huella_recibo := encode(sha256(v_recibo_canonico), 'hex');
    INSERT INTO vec_bolsa_convocatorias.diario_borrador_version
    SELECT
        v_actual.localizador_esquema_version,
        v_actual.localizador_clave_ref,
        v_actual.localizador_generacion_clave,
        v_actual.localizador_hmac, v_revision_diario_nueva,
        v_actual.huella_esquema_version, v_actual.huella_clave_ref,
        v_actual.huella_generacion_clave, v_actual.huella_hmac,
        'confirmado', v_actual.cercado,
        v_actual.arrendamiento_inicia_en,
        v_actual.arrendamiento_vence_en, v_actual.accion,
        v_actual.huella_material_sha256, v_actual.recurso_ref,
        v_actual.contexto_recurso_huella_sha256,
        v_actual.esquema_huella_decision, v_actual.decision_ref,
        v_actual.huella_decision_sha256, v_actual.modulo_id,
        v_actual.tipo_recurso, v_actual.finalidad,
        v_actual.version_rol_ref, v_actual.version_rol_huella_sha256,
        v_actual.control_vigencia_rol_revision,
        v_actual.revision_catalogo_politicas,
        v_actual.catalogo_politicas_huella_sha256,
        v_actual.decision_verificada_en, v_actual.decision_valida_hasta,
        v_actual.atestacion_ref, v_actual.atestacion_version,
        v_actual.atestacion_estado, v_actual.huella_atestacion_sha256,
        v_actual.verificador_ref, v_actual.atestacion_verificada_en,
        v_transaccion_ref, v_actual.recurso_ref,
        (v_estado_nuevo ->> 'revision')::bigint,
        v_estado_nuevo ->> 'huella_estado_sha256', v_auditoria_ref,
        v_huella_auditoria, v_evento_ref, v_huella_evento, v_ahora,
        v_ahora, v_recibo_ref, v_recibo_canonico, v_huella_recibo,
        NULL, NULL,
        v_actual.asignacion_ref, v_actual.asignacion_huella_sha256,
        v_actual.control_vigencia_version_rol_ref,
        v_actual.control_vigencia_version_rol_huella_sha256,
        v_actual.decision_emitida_en;
    UPDATE vec_bolsa_convocatorias.diario_borrador_actual
       SET revision = v_revision_diario_nueva, actualizada_en = v_ahora
     WHERE localizador_esquema_version =
           v_actual.localizador_esquema_version
       AND localizador_clave_ref = v_actual.localizador_clave_ref
       AND localizador_generacion_clave =
           v_actual.localizador_generacion_clave
       AND localizador_hmac = v_actual.localizador_hmac
       AND revision = v_revision_control;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS perdido al cerrar el diario';
    END IF;
    RETURN QUERY SELECT
        'confirmada'::text, 'confirmado'::text,
        v_revision_diario_nueva, v_actual.cercado,
        v_transaccion_ref, v_actual.accion, v_actual.recurso_ref,
        (v_estado_nuevo ->> 'revision')::bigint,
        v_estado_nuevo ->> 'huella_estado_sha256', v_auditoria_ref,
        v_huella_auditoria, v_evento_ref, v_huella_evento, v_ahora,
        v_recibo;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    p_confirmacion jsonb, p_prueba jsonb, p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea, p_material_canonico bytea,
    p_version_canonica bytea, p_sobre_cifrado bytea
)
RETURNS TABLE (
    resultado text, estado_diario text, revision_diario bigint,
    cercado bigint, transaccion_ref text, accion text,
    estado_principal_ref text, estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    -- Cierre dentro de la propia capacidad: aun si una migracion o un
    -- operador conceden EXECUTE por error, ningun parametro se inspecciona y
    -- no se alcanza el nucleo legado hasta completar el contrato KMS.
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'confirmacion de borrador cerrada: contrato KMS no satisfecho';
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
    p_identidad jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_l jsonb;
    v_f jsonb;
BEGIN
    IF vec_bolsa_convocatorias.objeto_json_exacto(
           p_identidad, ARRAY['huella_solicitud','localizador']
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    v_l := p_identidad -> 'localizador';
    v_f := p_identidad -> 'huella_solicitud';
    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_l, ARRAY[
               'clave_ref','dominio','generacion_clave','hmac_sha256',
               'version_esquema'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           v_f, ARRAY[
               'clave_ref','dominio','generacion_clave','hmac_sha256',
               'version_esquema'
           ]
       ) IS NOT TRUE
       OR v_l ->> 'dominio' <> 'localizador'
       OR v_f ->> 'dominio' <> 'huella_solicitud'
       OR (v_l ->> 'version_esquema') !~ '^[1-9][0-9]{0,4}$'
       OR (v_f ->> 'version_esquema') !~ '^[1-9][0-9]{0,4}$'
       OR (v_l ->> 'generacion_clave') !~ '^[1-9][0-9]{0,9}$'
       OR (v_f ->> 'generacion_clave') !~ '^[1-9][0-9]{0,9}$'
       OR v_l ->> 'version_esquema' <> v_f ->> 'version_esquema'
       OR v_l ->> 'generacion_clave' <> v_f ->> 'generacion_clave'
       OR (v_l ->> 'hmac_sha256') !~ '^[0-9a-f]{64}$'
       OR (v_f ->> 'hmac_sha256') !~ '^[0-9a-f]{64}$'
       OR vec_bolsa_convocatorias.referencia_clave_hmac_valida(
           v_l ->> 'clave_ref', 'localizador'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.referencia_clave_hmac_valida(
           v_f ->> 'clave_ref', 'huella_solicitud'
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    RETURN (v_l ->> 'version_esquema')::integer BETWEEN 1 AND 65535
       AND (v_f ->> 'version_esquema')::integer BETWEEN 1 AND 65535
       AND (v_l ->> 'generacion_clave')::bigint BETWEEN 1 AND 4294967295
       AND (v_f ->> 'generacion_clave')::bigint BETWEEN 1 AND 4294967295;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- La generacion primaria ocupa la primera posicion. La rotacion admite como
-- maximo cuatro pares L/F, todos nominalmente emparejados, en orden de
-- generacion estrictamente decreciente y sin reutilizar L ni F.
CREATE FUNCTION vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
    p_identidades jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_total integer;
    v_invalidas integer;
    v_generaciones_incorrectas integer;
    v_localizadores_unicos integer;
    v_huellas_unicas integer;
BEGIN
    IF jsonb_typeof(p_identidades) <> 'array' THEN
        RETURN false;
    END IF;
    v_total := jsonb_array_length(p_identidades);
    IF v_total NOT BETWEEN 1 AND 4 THEN
        RETURN false;
    END IF;

    SELECT
        count(*) FILTER (
            WHERE vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
                      elemento
                  ) IS NOT TRUE
        ),
        count(DISTINCT elemento -> 'localizador'),
        count(DISTINCT elemento -> 'huella_solicitud')
      INTO v_invalidas, v_localizadores_unicos, v_huellas_unicas
      FROM jsonb_array_elements(p_identidades) AS identidad(elemento);
    IF v_invalidas <> 0 OR v_localizadores_unicos <> v_total
       OR v_huellas_unicas <> v_total THEN
        RETURN false;
    END IF;

    SELECT count(*)
      INTO v_generaciones_incorrectas
      FROM (
          SELECT (elemento -> 'localizador' ->> 'generacion_clave')::bigint
                     AS generacion,
                 lag((elemento -> 'localizador' ->>
                      'generacion_clave')::bigint)
                     OVER (ORDER BY ordinalidad) AS anterior
            FROM jsonb_array_elements(p_identidades)
                 WITH ORDINALITY AS identidad(elemento, ordinalidad)
      ) AS generaciones
     WHERE anterior IS NOT NULL AND anterior <= generacion;
    RETURN v_generaciones_incorrectas = 0;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

-- Comprueba toda la orden efimera de SolicitudReservaDecisionBorrador. No
-- persiste version, decision ni recurso canonicos; solo autoriza su proyeccion.
CREATE FUNCTION vec_bolsa_convocatorias.validar_reserva_borrador_interna_v1(
    p_reserva jsonb,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_material jsonb;
    v_version jsonb;
    v_decision jsonb;
    v_contexto jsonb;
    v_proyeccion_decision jsonb;
    v_atestacion jsonb;
    v_estado_nuevo jsonb;
    v_estado_esperado jsonb;
    v_accion text;
    v_huella_material text;
    v_recurso_ref text;
    v_partes_hmac text[];
    v_solicitada timestamptz;
    v_inicio timestamptz;
    v_fin timestamptz;
    v_emitida timestamptz;
    v_verificada timestamptz;
    v_valida_hasta timestamptz;
    v_atestacion_verificada timestamptz;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR p_instante IS NULL
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_reserva, ARRAY[
               'accion','arrendamiento_inicia_en',
               'arrendamiento_vence_en','contexto_recurso_huella_sha256',
               'decision','esquema','huella_material_sha256','identidad',
               'identidades_consulta','recurso_ref','solicitada_en'
           ]
       ) IS NOT TRUE
       OR p_reserva ->> 'esquema' <>
          'vec.bolsa.convocatoria.reserva-decision.v2'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           p_reserva -> 'identidad'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           p_reserva -> 'identidades_consulta'
       ) IS NOT TRUE
       OR (SELECT count(*)
             FROM jsonb_array_elements(
                      p_reserva -> 'identidades_consulta'
                  ) AS identidad(elemento)
            WHERE elemento = p_reserva -> 'identidad') <> 1
       OR p_material_canonico IS NULL OR p_version_canonica IS NULL
       OR p_decision_canonica IS NULL
       OR p_contexto_recurso_canonico IS NULL
       OR octet_length(p_material_canonico) NOT BETWEEN 2 AND 1048576
       OR octet_length(p_version_canonica) NOT BETWEEN 2 AND 33554432
       OR octet_length(p_decision_canonica) NOT BETWEEN 2 AND 1048576
       OR octet_length(p_contexto_recurso_canonico) NOT BETWEEN 2 AND 65536
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           p_reserva ->> 'huella_material_sha256'
       ) IS NOT TRUE
       OR encode(sha256(p_material_canonico), 'hex') <>
          p_reserva ->> 'huella_material_sha256'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           p_reserva ->> 'contexto_recurso_huella_sha256'
       ) IS NOT TRUE
       OR encode(sha256(p_contexto_recurso_canonico), 'hex') <>
          p_reserva ->> 'contexto_recurso_huella_sha256'
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_reserva ->> 'solicitada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_reserva ->> 'arrendamiento_inicia_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           p_reserva ->> 'arrendamiento_vence_en'
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    BEGIN
        v_material := convert_from(p_material_canonico, 'UTF8')::jsonb;
        v_version := convert_from(p_version_canonica, 'UTF8')::jsonb;
        v_decision := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        v_contexto :=
            convert_from(p_contexto_recurso_canonico, 'UTF8')::jsonb;
        v_solicitada := (p_reserva ->> 'solicitada_en')::timestamptz;
        v_inicio :=
            (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz;
        v_fin := (p_reserva ->> 'arrendamiento_vence_en')::timestamptz;
    EXCEPTION WHEN OTHERS THEN
        RETURN false;
    END;

    v_accion := p_reserva ->> 'accion';
    v_huella_material := p_reserva ->> 'huella_material_sha256';
    v_recurso_ref := p_reserva ->> 'recurso_ref';
    v_proyeccion_decision := p_reserva -> 'decision';
    v_atestacion := v_proyeccion_decision -> 'atestacion_pdp';
    v_estado_nuevo := v_material -> 'estado_principal_nuevo';
    v_estado_esperado := v_material -> 'estado_principal_esperado';

    IF v_accion NOT IN (
           'bolsa.convocatoria.borrador.crear',
           'bolsa.convocatoria.borrador.actualizar'
       )
       OR v_material ->> 'esquema' <>
          'bolsa.convocatoria.intencion.v2'
       OR v_material ->> 'accion' <> v_accion
       OR ((v_accion = 'bolsa.convocatoria.borrador.crear'
            AND vec_bolsa_convocatorias.objeto_json_exacto(
                v_material, ARRAY[
                    'accion','dominio_criptografico_motivo','esquema',
                    'estado_principal_nuevo','generacion_clave_motivo',
                    'huella_motivo_hmac_sha256'
                ]
            ) IS NOT TRUE)
           OR
           (v_accion = 'bolsa.convocatoria.borrador.actualizar'
            AND vec_bolsa_convocatorias.objeto_json_exacto(
                v_material, ARRAY[
                    'accion','dominio_criptografico_motivo','esquema',
                    'estado_principal_esperado','estado_principal_nuevo',
                    'generacion_clave_motivo',
                    'huella_motivo_hmac_sha256'
                ]
            ) IS NOT TRUE))
       OR vec_bolsa_convocatorias.referencia_estado_valida(
           v_estado_nuevo
       ) IS NOT TRUE
       OR v_estado_nuevo ->> 'referencia' <> v_recurso_ref
       OR v_recurso_ref <> p_reserva ->> 'recurso_ref'
       OR (v_accion = 'bolsa.convocatoria.borrador.crear'
           AND (v_estado_nuevo ->> 'revision')::bigint <> 1)
       OR (v_accion = 'bolsa.convocatoria.borrador.actualizar'
           AND (vec_bolsa_convocatorias.referencia_estado_valida(
                    v_estado_esperado
                ) IS NOT TRUE
                OR v_estado_esperado ->> 'referencia' <> v_recurso_ref
                OR (v_estado_nuevo ->> 'revision')::bigint <>
                   (v_estado_esperado ->> 'revision')::bigint + 1
                OR v_estado_nuevo ->> 'huella_estado_sha256' =
                   v_estado_esperado ->> 'huella_estado_sha256'))
       OR v_material ->> 'dominio_criptografico_motivo' <>
          'bolsa.convocatoria.motivo.v1'
       OR (v_material ->> 'generacion_clave_motivo') !~
          '^[1-9][0-9]{0,9}$' THEN
        RETURN false;
    END IF;
    v_partes_hmac := regexp_match(
        v_material ->> 'huella_motivo_hmac_sha256',
        '^hmac-sha256:(.+):([0-9a-f]{64})$'
    );
    IF cardinality(v_partes_hmac) <> 2
       OR vec_bolsa_convocatorias.referencia_clave_hmac_valida(
           v_partes_hmac[1], 'motivo'
       ) IS NOT TRUE
       OR (v_material ->> 'generacion_clave_motivo')::bigint NOT BETWEEN
          1 AND 4294967295 THEN
        RETURN false;
    END IF;

    -- La version es efimera, pero fija estado, ambito y material exactos.
    IF jsonb_typeof(v_version) <> 'object'
       OR v_version ->> 'id' IS NULL
       OR (v_version ->> 'secuencia') !~ '^[1-9][0-9]{0,18}$'
       OR (v_version ->> 'revision') !~ '^[1-9][0-9]{0,18}$'
       OR v_version ->> 'estado_gobierno' <> 'borrador'
       OR (v_version ->> 'id') || '#' ||
          (v_version ->> 'secuencia') <> v_recurso_ref
       OR (v_version ->> 'revision')::bigint <>
          (v_estado_nuevo ->> 'revision')::bigint
       OR encode(sha256(p_version_canonica), 'hex') <>
          v_estado_nuevo ->> 'huella_estado_sha256'
       OR jsonb_typeof(v_version -> 'ambito_organizativo') <> 'object'
       OR jsonb_typeof(v_version -> 'contenido') <> 'object' THEN
        RETURN false;
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_contexto, ARRAY['ambitos','atributos']
       ) IS NOT TRUE
       OR jsonb_typeof(v_contexto -> 'ambitos') <> 'object'
       OR jsonb_typeof(v_contexto -> 'atributos') <> 'object'
       OR v_contexto -> 'atributos' <>
          jsonb_build_object('huella_intencion_sha256', v_huella_material)
       OR v_contexto -> 'ambitos' <>
          v_version -> 'ambito_organizativo' THEN
        RETURN false;
    END IF;

    IF vec_bolsa_convocatorias.objeto_json_exacto(
           v_proyeccion_decision, ARRAY[
               'accion','asignacion_huella_sha256','asignacion_ref',
               'atestacion_pdp','catalogo_politicas_huella_sha256',
               'contexto_recurso_huella_sha256',
               'control_vigencia_version_rol_huella_sha256',
               'control_vigencia_version_rol_ref',
               'control_vigencia_version_rol_revision','decision_ref',
               'emitida_en','esquema_huella','finalidad',
               'huella_decision_sha256',
               'modulo_id','recurso_ref','revision_catalogo_politicas',
               'tipo_recurso','valida_hasta','verificada_en',
               'version_rol_huella_sha256','version_rol_ref'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           v_atestacion, ARRAY[
               'atestacion_ref','decision_ref','estado',
               'huella_atestacion_sha256','verificada_en',
               'verificador_ref','version'
           ]
       ) IS NOT TRUE
       OR v_proyeccion_decision ->> 'esquema_huella' <>
          'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR encode(sha256(p_decision_canonica), 'hex') <>
          v_proyeccion_decision ->> 'huella_decision_sha256'
       OR v_proyeccion_decision ->> 'accion' <> v_accion
       OR v_proyeccion_decision ->> 'recurso_ref' <> v_recurso_ref
       OR v_proyeccion_decision ->> 'modulo_id' <> 'bolsa'
       OR v_proyeccion_decision ->> 'tipo_recurso' <>
          'version_convocatoria_gobernada'
       OR v_proyeccion_decision ->> 'contexto_recurso_huella_sha256' <>
          p_reserva ->> 'contexto_recurso_huella_sha256'
       OR v_proyeccion_decision ->> 'finalidad' <> 'gobierno_convocatorias'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_proyeccion_decision ->> 'decision_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_proyeccion_decision ->> 'asignacion_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_proyeccion_decision ->> 'asignacion_huella_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_proyeccion_decision ->> 'version_rol_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_proyeccion_decision ->> 'version_rol_huella_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_proyeccion_decision ->>
               'control_vigencia_version_rol_ref', 512
       ) IS NOT TRUE
       OR v_proyeccion_decision ->> 'control_vigencia_version_rol_ref' <>
          v_proyeccion_decision ->> 'version_rol_ref'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_proyeccion_decision ->>
               'control_vigencia_version_rol_huella_sha256'
       ) IS NOT TRUE
       OR (v_proyeccion_decision ->>
           'control_vigencia_version_rol_revision') !~
          '^[1-9][0-9]{0,18}$'
       OR (v_proyeccion_decision ->> 'revision_catalogo_politicas') !~
          '^[1-9][0-9]{0,18}$'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_proyeccion_decision ->> 'catalogo_politicas_huella_sha256'
       ) IS NOT TRUE
       OR v_atestacion ->> 'decision_ref' <>
          v_proyeccion_decision ->> 'decision_ref'
       OR v_atestacion ->> 'estado' <> 'activa'
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_atestacion ->> 'atestacion_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.texto_opaco_valido(
           v_atestacion ->> 'verificador_ref', 512
       ) IS NOT TRUE
       OR v_atestacion ->> 'atestacion_ref' =
          v_atestacion ->> 'verificador_ref'
       OR (v_atestacion ->> 'version') !~ '^[1-9][0-9]{0,9}$'
       OR vec_bolsa_convocatorias.huella_sha256_valida(
           v_atestacion ->> 'huella_atestacion_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_proyeccion_decision ->> 'emitida_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_proyeccion_decision ->> 'verificada_en'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_proyeccion_decision ->> 'valida_hasta'
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
           v_atestacion ->> 'verificada_en'
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    v_emitida :=
        (v_proyeccion_decision ->> 'emitida_en')::timestamptz;
    v_verificada :=
        (v_proyeccion_decision ->> 'verificada_en')::timestamptz;
    v_valida_hasta :=
        (v_proyeccion_decision ->> 'valida_hasta')::timestamptz;
    v_atestacion_verificada :=
        (v_atestacion ->> 'verificada_en')::timestamptz;

    IF jsonb_typeof(v_decision) <> 'object'
       OR v_decision ->> 'decision_ref' <>
          v_proyeccion_decision ->> 'decision_ref'
       OR v_decision ->> 'accion' <> v_accion
       OR v_decision ->> 'recurso_ref' <> v_recurso_ref
       OR v_decision ->> 'modulo_id' <> 'bolsa'
       OR v_decision ->> 'tipo_recurso' <>
          'version_convocatoria_gobernada'
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          p_reserva ->> 'contexto_recurso_huella_sha256'
       OR v_decision ->> 'finalidad' <> 'gobierno_convocatorias'
       OR v_decision ->> 'asignacion_ref' <>
          v_proyeccion_decision ->> 'asignacion_ref'
       OR v_decision ->> 'asignacion_huella_sha256' <>
          v_proyeccion_decision ->> 'asignacion_huella_sha256'
       OR v_decision ->> 'version_rol_ref' <>
          v_proyeccion_decision ->> 'version_rol_ref'
       OR v_decision ->> 'version_rol_huella_sha256' <>
          v_proyeccion_decision ->> 'version_rol_huella_sha256'
       OR v_decision ->> 'control_vigencia_version_rol_revision' <>
          v_proyeccion_decision ->> 'control_vigencia_version_rol_revision'
       OR v_decision ->> 'control_vigencia_version_rol_ref' <>
          v_proyeccion_decision ->> 'control_vigencia_version_rol_ref'
       OR v_decision ->> 'control_vigencia_version_rol_huella_sha256' <>
          v_proyeccion_decision ->>
              'control_vigencia_version_rol_huella_sha256'
       OR v_decision ->> 'revision_catalogo_politicas' <>
          v_proyeccion_decision ->> 'revision_catalogo_politicas'
       OR v_decision ->> 'catalogo_politicas_huella_sha256' <>
          v_proyeccion_decision ->> 'catalogo_politicas_huella_sha256'
       OR v_decision -> 'campos_permitidos' <>
          '["auditoria","evento_outbox","version_convocatoria"]'::jsonb
       OR v_decision -> 'obligaciones' <> '[]'::jsonb
       OR v_decision ->> 'garantia_minima' <> 'alto'
       OR v_decision ->> 'valida_hasta' <>
          v_proyeccion_decision ->> 'valida_hasta'
       OR v_decision ->> 'emitida_en' <>
          v_proyeccion_decision ->> 'emitida_en' THEN
        RETURN false;
    END IF;

    IF v_solicitada <> v_inicio OR v_inicio >= v_fin
       OR v_fin - v_inicio > interval '5 minutes'
       OR v_inicio < v_verificada OR v_fin > v_valida_hasta
       OR v_emitida > v_atestacion_verificada
       OR v_atestacion_verificada > v_verificada
       OR p_instante < v_solicitada
       OR p_instante - v_solicitada > interval '30 seconds'
       OR p_instante >= v_fin
       OR vec_bolsa_convocatorias.atestacion_pdp_borrador_vigente(
              v_atestacion ->> 'decision_ref',
              v_atestacion ->> 'atestacion_ref',
              (v_atestacion ->> 'version')::bigint,
              v_atestacion ->> 'estado',
              v_proyeccion_decision ->> 'huella_decision_sha256',
              v_atestacion ->> 'huella_atestacion_sha256',
              v_atestacion ->> 'verificador_ref',
              v_atestacion_verificada, p_instante
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(
    p_identidad jsonb
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    transaccion_ref text, accion text, estado_principal_ref text,
    estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb
)
LANGUAGE plpgsql
STABLE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_l jsonb;
    v_f jsonb;
    v_fila record;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           p_identidad
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'identidad de operacion invalida';
    END IF;
    v_l := p_identidad -> 'localizador';
    v_f := p_identidad -> 'huella_solicitud';
    SELECT h.*,
           x.alias_huella_esquema_version,
           x.alias_huella_clave_ref,
           x.alias_huella_generacion_clave,
           x.alias_huella_hmac
      INTO v_fila
      FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
      JOIN vec_bolsa_convocatorias.diario_borrador_actual AS a
        ON a.localizador_esquema_version =
           x.primario_localizador_esquema_version
       AND a.localizador_clave_ref = x.primario_localizador_clave_ref
       AND a.localizador_generacion_clave =
           x.primario_localizador_generacion_clave
       AND a.localizador_hmac = x.primario_localizador_hmac
      JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
        ON h.localizador_esquema_version = a.localizador_esquema_version
       AND h.localizador_clave_ref = a.localizador_clave_ref
       AND h.localizador_generacion_clave =
           a.localizador_generacion_clave
       AND h.localizador_hmac = a.localizador_hmac
       AND h.revision = a.revision
     WHERE x.alias_localizador_esquema_version =
           (v_l ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref = v_l ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (v_l ->> 'generacion_clave')::bigint
       AND x.alias_localizador_hmac =
           decode(v_l ->> 'hmac_sha256', 'hex');
    IF NOT FOUND THEN
        RETURN QUERY SELECT
            'ausente'::text, 0::bigint, 0::bigint,
            NULL::timestamptz, NULL::timestamptz,
            NULL::text, NULL::text, NULL::text, NULL::bigint,
            NULL::text, NULL::text, NULL::text, NULL::text,
            NULL::text, NULL::timestamptz, NULL::jsonb;
        RETURN;
    END IF;
    IF ROW(v_fila.alias_huella_esquema_version,
           v_fila.alias_huella_clave_ref,
           v_fila.alias_huella_generacion_clave,
           v_fila.alias_huella_hmac)
       IS DISTINCT FROM
       ROW((v_f ->> 'version_esquema')::integer,
           v_f ->> 'clave_ref',
           (v_f ->> 'generacion_clave')::bigint,
           decode(v_f ->> 'hmac_sha256', 'hex')) THEN
        RETURN QUERY SELECT
            'conflicto'::text, 0::bigint, 0::bigint,
            NULL::timestamptz, NULL::timestamptz,
            NULL::text, NULL::text, NULL::text, NULL::bigint,
            NULL::text, NULL::text, NULL::text, NULL::text,
            NULL::text, NULL::timestamptz, NULL::jsonb;
        RETURN;
    END IF;
    RETURN QUERY SELECT
        v_fila.estado::text, v_fila.revision::bigint,
        v_fila.cercado::bigint,
        v_fila.arrendamiento_inicia_en::timestamptz,
        v_fila.arrendamiento_vence_en::timestamptz,
        v_fila.transaccion_ref::text, v_fila.accion::text,
        v_fila.estado_principal_ref::text,
        v_fila.estado_principal_revision::bigint,
        v_fila.estado_principal_huella_sha256::text,
        v_fila.auditoria_ref::text,
        v_fila.huella_auditoria_sha256::text,
        v_fila.evento_outbox_ref::text,
        v_fila.huella_evento_outbox_sha256::text,
        v_fila.confirmada_en::timestamptz,
        CASE WHEN v_fila.recibo_canonico IS NULL THEN NULL::jsonb
             ELSE convert_from(v_fila.recibo_canonico, 'UTF8')::jsonb END;
END
$funcion$;

-- Consulta el conjunto de rotacion completo bajo una unica instantanea
-- repetible. Devuelve cero o una coincidencia; dos localizadores existentes
-- son una ambiguedad de seguridad, aunque alguno presente conflicto de F.
CREATE FUNCTION vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
    p_identidades jsonb
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    transaccion_ref text, accion text, estado_principal_ref text,
    estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb,
    identidades_consultadas jsonb, identidad_primaria jsonb
)
LANGUAGE plpgsql
STABLE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_coincidencias integer;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') NOT IN (
           'repeatable read', 'serializable'
       )
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           p_identidades
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'conjunto de identidades de operacion invalido';
    END IF;

    WITH candidatas AS (
        SELECT elemento AS identidad
          FROM jsonb_array_elements(p_identidades) AS i(elemento)
    )
    SELECT count(DISTINCT (
               x.primario_localizador_esquema_version,
               x.primario_localizador_clave_ref,
               x.primario_localizador_generacion_clave,
               x.primario_localizador_hmac
           )) INTO v_coincidencias
      FROM candidatas AS i
      JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
        ON x.alias_localizador_esquema_version =
           (i.identidad -> 'localizador' ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref =
           i.identidad -> 'localizador' ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (i.identidad -> 'localizador' ->>
            'generacion_clave')::bigint
       AND x.alias_localizador_hmac = decode(
           i.identidad -> 'localizador' ->> 'hmac_sha256', 'hex'
       );
    IF v_coincidencias > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '21000',
            MESSAGE = 'consulta idempotente multigeneracion ambigua';
    END IF;

    RETURN QUERY
    WITH candidatas AS (
        SELECT elemento AS identidad, ordinalidad
          FROM jsonb_array_elements(p_identidades)
               WITH ORDINALITY AS i(elemento, ordinalidad)
    ), coincidencias AS (
        SELECT i.identidad, i.ordinalidad, h.*,
               x.primario_localizador_esquema_version,
               x.primario_localizador_clave_ref,
               x.primario_localizador_generacion_clave,
               x.primario_localizador_hmac,
               x.primario_huella_esquema_version,
               x.primario_huella_clave_ref,
               x.primario_huella_generacion_clave,
               x.primario_huella_hmac,
               ROW(x.alias_huella_esquema_version,
                   x.alias_huella_clave_ref,
                   x.alias_huella_generacion_clave,
                   x.alias_huella_hmac)
               IS NOT DISTINCT FROM ROW(
                   (i.identidad -> 'huella_solicitud' ->>
                    'version_esquema')::integer,
                   i.identidad -> 'huella_solicitud' ->> 'clave_ref',
                   (i.identidad -> 'huella_solicitud' ->>
                    'generacion_clave')::bigint,
                   decode(i.identidad -> 'huella_solicitud' ->>
                          'hmac_sha256', 'hex')
               ) AS huella_coincide
          FROM candidatas AS i
          JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
            ON x.alias_localizador_esquema_version =
               (i.identidad -> 'localizador' ->>
                'version_esquema')::integer
           AND x.alias_localizador_clave_ref =
               i.identidad -> 'localizador' ->> 'clave_ref'
           AND x.alias_localizador_generacion_clave =
               (i.identidad -> 'localizador' ->>
                'generacion_clave')::bigint
           AND x.alias_localizador_hmac = decode(
               i.identidad -> 'localizador' ->> 'hmac_sha256', 'hex'
           )
          JOIN vec_bolsa_convocatorias.diario_borrador_actual AS a
            ON a.localizador_esquema_version =
               x.primario_localizador_esquema_version
           AND a.localizador_clave_ref =
               x.primario_localizador_clave_ref
           AND a.localizador_generacion_clave =
               x.primario_localizador_generacion_clave
           AND a.localizador_hmac = x.primario_localizador_hmac
          JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
            ON h.localizador_esquema_version =
               a.localizador_esquema_version
           AND h.localizador_clave_ref = a.localizador_clave_ref
           AND h.localizador_generacion_clave =
               a.localizador_generacion_clave
           AND h.localizador_hmac = a.localizador_hmac
           AND h.revision = a.revision
    ), resolucion AS (
        SELECT jsonb_agg(c.identidad ORDER BY c.ordinalidad)
                   AS identidades_consultadas,
               bool_and(c.huella_coincide) AS huellas_coinciden,
               min(c.ordinalidad) AS primera_ordinalidad,
               c.localizador_esquema_version,
               c.localizador_clave_ref,
               c.localizador_generacion_clave,
               c.localizador_hmac,
               c.huella_esquema_version,
               c.huella_clave_ref,
               c.huella_generacion_clave,
               c.huella_hmac,
               c.estado, c.revision, c.cercado,
               c.arrendamiento_inicia_en, c.arrendamiento_vence_en,
               c.transaccion_ref, c.accion, c.estado_principal_ref,
               c.estado_principal_revision,
               c.estado_principal_huella_sha256,
               c.auditoria_ref, c.huella_auditoria_sha256,
               c.evento_outbox_ref, c.huella_evento_outbox_sha256,
               c.confirmada_en, c.recibo_canonico
          FROM coincidencias AS c
         GROUP BY c.localizador_esquema_version,
                  c.localizador_clave_ref,
                  c.localizador_generacion_clave,
                  c.localizador_hmac,
                  c.huella_esquema_version,
                  c.huella_clave_ref,
                  c.huella_generacion_clave,
                  c.huella_hmac,
                  c.estado, c.revision, c.cercado,
                  c.arrendamiento_inicia_en, c.arrendamiento_vence_en,
                  c.transaccion_ref, c.accion, c.estado_principal_ref,
                  c.estado_principal_revision,
                  c.estado_principal_huella_sha256,
                  c.auditoria_ref, c.huella_auditoria_sha256,
                  c.evento_outbox_ref, c.huella_evento_outbox_sha256,
                  c.confirmada_en, c.recibo_canonico
    )
    SELECT CASE WHEN r.huellas_coinciden THEN r.estado::text
                ELSE 'conflicto'::text END,
           CASE WHEN r.huellas_coinciden THEN r.revision::bigint
                ELSE 0::bigint END,
           CASE WHEN r.huellas_coinciden THEN r.cercado::bigint
                ELSE 0::bigint END,
           CASE WHEN r.huellas_coinciden
                THEN r.arrendamiento_inicia_en::timestamptz END,
           CASE WHEN r.huellas_coinciden
                THEN r.arrendamiento_vence_en::timestamptz END,
           CASE WHEN r.huellas_coinciden THEN r.transaccion_ref::text END,
           CASE WHEN r.huellas_coinciden THEN r.accion::text END,
           CASE WHEN r.huellas_coinciden
                THEN r.estado_principal_ref::text END,
           CASE WHEN r.huellas_coinciden
                THEN r.estado_principal_revision::bigint END,
           CASE WHEN r.huellas_coinciden
                THEN r.estado_principal_huella_sha256::text END,
           CASE WHEN r.huellas_coinciden THEN r.auditoria_ref::text END,
           CASE WHEN r.huellas_coinciden
                THEN r.huella_auditoria_sha256::text END,
           CASE WHEN r.huellas_coinciden THEN r.evento_outbox_ref::text END,
           CASE WHEN r.huellas_coinciden
                THEN r.huella_evento_outbox_sha256::text END,
           CASE WHEN r.huellas_coinciden THEN r.confirmada_en::timestamptz END,
           CASE WHEN r.huellas_coinciden AND r.recibo_canonico IS NOT NULL
                THEN convert_from(r.recibo_canonico, 'UTF8')::jsonb END,
           r.identidades_consultadas,
           jsonb_build_object(
               'localizador', jsonb_build_object(
                   'version_esquema', r.localizador_esquema_version,
                   'dominio', 'localizador',
                   'clave_ref', r.localizador_clave_ref,
                   'generacion_clave', r.localizador_generacion_clave,
                   'hmac_sha256', encode(r.localizador_hmac, 'hex')
               ),
               'huella_solicitud', jsonb_build_object(
                   'version_esquema', r.huella_esquema_version,
                   'dominio', 'huella_solicitud',
                   'clave_ref', r.huella_clave_ref,
                   'generacion_clave', r.huella_generacion_clave,
                   'hmac_sha256', encode(r.huella_hmac, 'hex')
               )
           )
      FROM resolucion AS r
     ORDER BY r.huellas_coinciden, r.primera_ordinalidad
     LIMIT 1;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.consultar_identidades_borrador_v1(
    p_identidades jsonb
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    transaccion_ref text, accion text, estado_principal_ref text,
    estado_principal_revision bigint,
    estado_principal_huella_sha256 text,
    auditoria_ref text, huella_auditoria_sha256 text,
    evento_outbox_ref text, huella_evento_outbox_sha256 text,
    confirmada_en timestamptz, recibo jsonb,
    identidades_consultadas jsonb, identidad_primaria jsonb
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad runtime de consulta idempotente rechazada';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
          p_identidades
      );
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
    p_reserva jsonb,
    p_material_canonico bytea,
    p_version_canonica bytea,
    p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    recibo jsonb, identidades_consultadas jsonb,
    identidad_primaria jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY INVOKER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_identidad jsonb := p_reserva -> 'identidad';
    v_l jsonb := v_identidad -> 'localizador';
    v_f jsonb := v_identidad -> 'huella_solicitud';
    v_d jsonb := p_reserva -> 'decision';
    v_a jsonb := v_d -> 'atestacion_pdp';
    v_material jsonb;
    v_estado_nuevo jsonb;
    v_estado_esperado jsonb;
    v_partes_hmac text[];
    v_existente vec_bolsa_convocatorias.diario_borrador_version%ROWTYPE;
    v_identidad_coincidente jsonb;
    v_identidades_resueltas jsonb;
    v_identidad_primaria jsonb;
    v_coincidencias integer;
    v_huella_coincide boolean;
    v_consumo_ref text;
BEGIN
    IF current_user <> 'vec_bolsa_convocatorias_propietario'
       OR current_setting('transaction_isolation') <> 'serializable'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           p_reserva, ARRAY[
               'accion','arrendamiento_inicia_en',
               'arrendamiento_vence_en','contexto_recurso_huella_sha256',
               'decision','esquema','huella_material_sha256','identidad',
               'identidades_consulta','recurso_ref','solicitada_en'
           ]
       ) IS NOT TRUE
       OR p_reserva ->> 'esquema' <>
          'vec.bolsa.convocatoria.reserva-decision.v2'
       OR vec_bolsa_convocatorias.identidad_operacion_borrador_valida(
           v_identidad
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           p_reserva -> 'identidades_consulta'
       ) IS NOT TRUE
       OR v_identidad IS DISTINCT FROM
          p_reserva -> 'identidades_consulta' -> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reserva post-PDP no revalidada';
    END IF;

    -- Todas las generaciones se cotejan antes de cualquier alta. El bloqueo
    -- de los punteros coincidentes y SERIALIZABLE convierten consulta+alta en
    -- una sola decision; una rotacion nunca puede elegir la primera fila.
    WITH candidatas AS (
        SELECT elemento AS identidad
          FROM jsonb_array_elements(
                   p_reserva -> 'identidades_consulta'
               ) AS i(elemento)
    )
    SELECT count(DISTINCT (
               x.primario_localizador_esquema_version,
               x.primario_localizador_clave_ref,
               x.primario_localizador_generacion_clave,
               x.primario_localizador_hmac
           )) INTO v_coincidencias
      FROM candidatas AS i
      JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
        ON x.alias_localizador_esquema_version =
           (i.identidad -> 'localizador' ->> 'version_esquema')::integer
       AND x.alias_localizador_clave_ref =
           i.identidad -> 'localizador' ->> 'clave_ref'
       AND x.alias_localizador_generacion_clave =
           (i.identidad -> 'localizador' ->>
            'generacion_clave')::bigint
       AND x.alias_localizador_hmac = decode(
           i.identidad -> 'localizador' ->> 'hmac_sha256', 'hex'
       );
    IF v_coincidencias > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '21000',
            MESSAGE = 'reserva idempotente multigeneracion ambigua';
    ELSIF v_coincidencias = 1 THEN
        WITH candidatas AS (
            SELECT elemento AS identidad, ordinalidad
              FROM jsonb_array_elements(
                       p_reserva -> 'identidades_consulta'
                   ) WITH ORDINALITY AS i(elemento, ordinalidad)
        )
        SELECT i.identidad,
               ROW(x.alias_huella_esquema_version,
                   x.alias_huella_clave_ref,
                   x.alias_huella_generacion_clave,
                   x.alias_huella_hmac)
               IS NOT DISTINCT FROM ROW(
                   (i.identidad -> 'huella_solicitud' ->>
                    'version_esquema')::integer,
                   i.identidad -> 'huella_solicitud' ->> 'clave_ref',
                   (i.identidad -> 'huella_solicitud' ->>
                    'generacion_clave')::bigint,
                   decode(i.identidad -> 'huella_solicitud' ->>
                          'hmac_sha256', 'hex')
               ) AS huella_coincide
          INTO STRICT v_identidad_coincidente, v_huella_coincide
          FROM candidatas AS i
          JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
            ON x.alias_localizador_esquema_version =
               (i.identidad -> 'localizador' ->>
                'version_esquema')::integer
           AND x.alias_localizador_clave_ref =
               i.identidad -> 'localizador' ->> 'clave_ref'
           AND x.alias_localizador_generacion_clave =
               (i.identidad -> 'localizador' ->>
                'generacion_clave')::bigint
           AND x.alias_localizador_hmac = decode(
               i.identidad -> 'localizador' ->> 'hmac_sha256', 'hex'
           )
         ORDER BY huella_coincide, i.ordinalidad
         LIMIT 1;
        SELECT h.* INTO STRICT v_existente
          FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
          JOIN vec_bolsa_convocatorias.diario_borrador_actual AS c
            ON c.localizador_esquema_version =
               x.primario_localizador_esquema_version
           AND c.localizador_clave_ref =
               x.primario_localizador_clave_ref
           AND c.localizador_generacion_clave =
               x.primario_localizador_generacion_clave
           AND c.localizador_hmac = x.primario_localizador_hmac
          JOIN vec_bolsa_convocatorias.diario_borrador_version AS h
            ON h.localizador_esquema_version =
               c.localizador_esquema_version
           AND h.localizador_clave_ref = c.localizador_clave_ref
           AND h.localizador_generacion_clave =
               c.localizador_generacion_clave
           AND h.localizador_hmac = c.localizador_hmac
           AND h.revision = c.revision
         WHERE x.alias_localizador_esquema_version =
               (v_identidad_coincidente -> 'localizador' ->>
                'version_esquema')::integer
           AND x.alias_localizador_clave_ref =
               v_identidad_coincidente -> 'localizador' ->> 'clave_ref'
           AND x.alias_localizador_generacion_clave =
               (v_identidad_coincidente -> 'localizador' ->>
                'generacion_clave')::bigint
           AND x.alias_localizador_hmac = decode(
               v_identidad_coincidente -> 'localizador' ->>
               'hmac_sha256', 'hex'
           )
         FOR UPDATE OF c;
        v_identidad_primaria := jsonb_build_object(
            'localizador', jsonb_build_object(
                'version_esquema',
                    v_existente.localizador_esquema_version,
                'dominio', 'localizador',
                'clave_ref', v_existente.localizador_clave_ref,
                'generacion_clave',
                    v_existente.localizador_generacion_clave,
                'hmac_sha256', encode(
                    v_existente.localizador_hmac, 'hex'
                )
            ),
            'huella_solicitud', jsonb_build_object(
                'version_esquema', v_existente.huella_esquema_version,
                'dominio', 'huella_solicitud',
                'clave_ref', v_existente.huella_clave_ref,
                'generacion_clave',
                    v_existente.huella_generacion_clave,
                'hmac_sha256', encode(v_existente.huella_hmac, 'hex')
            )
        );
        SELECT jsonb_agg(i.elemento ORDER BY i.ordinalidad)
          INTO STRICT v_identidades_resueltas
          FROM jsonb_array_elements(p_reserva -> 'identidades_consulta')
               WITH ORDINALITY AS i(elemento, ordinalidad)
          JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
            ON x.alias_localizador_esquema_version =
               (i.elemento -> 'localizador' ->>
                'version_esquema')::integer
           AND x.alias_localizador_clave_ref =
               i.elemento -> 'localizador' ->> 'clave_ref'
           AND x.alias_localizador_generacion_clave =
               (i.elemento -> 'localizador' ->>
                'generacion_clave')::bigint
           AND x.alias_localizador_hmac = decode(
               i.elemento -> 'localizador' ->> 'hmac_sha256', 'hex'
           )
         WHERE x.primario_localizador_esquema_version =
               v_existente.localizador_esquema_version
           AND x.primario_localizador_clave_ref =
               v_existente.localizador_clave_ref
           AND x.primario_localizador_generacion_clave =
               v_existente.localizador_generacion_clave
           AND x.primario_localizador_hmac =
               v_existente.localizador_hmac;

        -- El derivador HMAC es parte de la base de confianza hasta que una
        -- atestacion autoritativa demuestre la pareja generada. Una segunda
        -- pareja distinta de la misma generacion no se incorpora como alias:
        -- se rechaza antes de escribir y la UNIQUE mantiene la invariante.
        IF EXISTS (
            SELECT 1
              FROM jsonb_array_elements(
                       p_reserva -> 'identidades_consulta'
                   ) AS i(elemento)
              JOIN vec_bolsa_convocatorias.identidad_alias_borrador AS x
                ON x.primario_localizador_esquema_version =
                   v_existente.localizador_esquema_version
               AND x.primario_localizador_clave_ref =
                   v_existente.localizador_clave_ref
               AND x.primario_localizador_generacion_clave =
                   v_existente.localizador_generacion_clave
               AND x.primario_localizador_hmac =
                   v_existente.localizador_hmac
               AND x.alias_localizador_generacion_clave =
                   (i.elemento -> 'localizador' ->>
                    'generacion_clave')::bigint
             WHERE ROW(
                       x.alias_localizador_esquema_version,
                       x.alias_localizador_clave_ref,
                       x.alias_localizador_generacion_clave,
                       x.alias_localizador_hmac,
                       x.alias_huella_esquema_version,
                       x.alias_huella_clave_ref,
                       x.alias_huella_generacion_clave,
                       x.alias_huella_hmac
                   ) IS DISTINCT FROM ROW(
                       (i.elemento -> 'localizador' ->>
                        'version_esquema')::integer,
                       i.elemento -> 'localizador' ->> 'clave_ref',
                       (i.elemento -> 'localizador' ->>
                        'generacion_clave')::bigint,
                       decode(i.elemento -> 'localizador' ->>
                              'hmac_sha256', 'hex'),
                       (i.elemento -> 'huella_solicitud' ->>
                        'version_esquema')::integer,
                       i.elemento -> 'huella_solicitud' ->> 'clave_ref',
                       (i.elemento -> 'huella_solicitud' ->>
                        'generacion_clave')::bigint,
                       decode(i.elemento -> 'huella_solicitud' ->>
                              'hmac_sha256', 'hex')
                   )
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'pareja L/F alternativa para generacion primaria';
        END IF;
        IF v_huella_coincide IS NOT TRUE THEN
            RETURN QUERY SELECT 'conflicto'::text, 0::bigint, 0::bigint,
                NULL::timestamptz, NULL::timestamptz, NULL::jsonb,
                v_identidades_resueltas, v_identidad_primaria;
        ELSE
            -- Una ventana de rotacion deslizante puede aportar generaciones
            -- nuevas junto a un alias historico exacto. Se incorporan todas
            -- como historia del mismo primario antes de devolver el replay.
            WITH candidatas AS (
                SELECT elemento, ordinalidad
                  FROM jsonb_array_elements(
                           p_reserva -> 'identidades_consulta'
                       ) WITH ORDINALITY AS i(elemento, ordinalidad)
            ), faltantes AS (
                SELECT i.*,
                       row_number() OVER (ORDER BY i.ordinalidad) AS nueva
                  FROM candidatas AS i
                  LEFT JOIN vec_bolsa_convocatorias.identidad_alias_borrador
                    AS x
                    ON x.alias_localizador_esquema_version =
                       (i.elemento -> 'localizador' ->>
                        'version_esquema')::integer
                   AND x.alias_localizador_clave_ref =
                       i.elemento -> 'localizador' ->> 'clave_ref'
                   AND x.alias_localizador_generacion_clave =
                       (i.elemento -> 'localizador' ->>
                        'generacion_clave')::bigint
                   AND x.alias_localizador_hmac = decode(
                       i.elemento -> 'localizador' ->>
                       'hmac_sha256', 'hex'
                   )
                 WHERE x.alias_localizador_hmac IS NULL
            ), base AS (
                SELECT COALESCE(max(x.ordinalidad), 0) AS ultima
                  FROM vec_bolsa_convocatorias.identidad_alias_borrador AS x
                 WHERE x.primario_localizador_esquema_version =
                       v_existente.localizador_esquema_version
                   AND x.primario_localizador_clave_ref =
                       v_existente.localizador_clave_ref
                   AND x.primario_localizador_generacion_clave =
                       v_existente.localizador_generacion_clave
                   AND x.primario_localizador_hmac =
                       v_existente.localizador_hmac
            )
            INSERT INTO vec_bolsa_convocatorias.identidad_alias_borrador
            SELECT
                (f.elemento -> 'localizador' ->>
                 'version_esquema')::integer,
                f.elemento -> 'localizador' ->> 'clave_ref',
                (f.elemento -> 'localizador' ->>
                 'generacion_clave')::bigint,
                decode(f.elemento -> 'localizador' ->>
                       'hmac_sha256', 'hex'),
                (f.elemento -> 'huella_solicitud' ->>
                 'version_esquema')::integer,
                f.elemento -> 'huella_solicitud' ->> 'clave_ref',
                (f.elemento -> 'huella_solicitud' ->>
                 'generacion_clave')::bigint,
                decode(f.elemento -> 'huella_solicitud' ->>
                       'hmac_sha256', 'hex'),
                v_existente.localizador_esquema_version,
                v_existente.localizador_clave_ref,
                v_existente.localizador_generacion_clave,
                v_existente.localizador_hmac,
                v_existente.huella_esquema_version,
                v_existente.huella_clave_ref,
                v_existente.huella_generacion_clave,
                v_existente.huella_hmac,
                1, b.ultima + f.nueva, v_ahora
              FROM faltantes AS f CROSS JOIN base AS b;
            -- La rama de coincidencia nunca concede la reserva al intento
            -- actual. Si el ganador sigue reservado se proyecta en_curso:
            -- devolver reservado permitiría al perdedor adoptar una decisión
            -- y un lease que no creó. El estado durable no se modifica.
            RETURN QUERY SELECT
                CASE WHEN v_existente.estado = 'reservado'
                     THEN 'en_curso'::text
                     ELSE v_existente.estado::text END,
                v_existente.revision::bigint, v_existente.cercado::bigint,
                v_existente.arrendamiento_inicia_en::timestamptz,
                v_existente.arrendamiento_vence_en::timestamptz,
                CASE WHEN v_existente.recibo_canonico IS NULL
                     THEN NULL::jsonb
                     ELSE convert_from(
                              v_existente.recibo_canonico, 'UTF8'
                          )::jsonb END,
                p_reserva -> 'identidades_consulta',
                v_identidad_primaria;
        END IF;
        RETURN;
    END IF;

    -- Solo las altas nuevas consumen el veredicto temporal actual. Un replay
    -- L/F exacto se resuelve con el veredicto durable, incluso si entre ambos
    -- intentos vencieron la decision o la atestacion.
    v_ahora := date_trunc('microseconds', clock_timestamp());
    IF vec_bolsa_convocatorias.validar_reserva_borrador_interna_v1(
           p_reserva, p_material_canonico, p_version_canonica,
           p_decision_canonica, p_contexto_recurso_canonico, v_ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reserva post-PDP no revalidada';
    END IF;

    v_material := convert_from(p_material_canonico, 'UTF8')::jsonb;
    v_estado_nuevo := v_material -> 'estado_principal_nuevo';
    v_estado_esperado := v_material -> 'estado_principal_esperado';
    v_partes_hmac := regexp_match(
        v_material ->> 'huella_motivo_hmac_sha256',
        '^hmac-sha256:(.+):([0-9a-f]{64})$'
    );
    INSERT INTO vec_bolsa_convocatorias.material_borrador(
        huella_material_sha256, material_canonico, esquema, accion,
        recurso_ref, revision_nueva, huella_estado_nuevo_sha256,
        revision_esperada, huella_estado_esperado_sha256,
        dominio_criptografico_motivo, generacion_clave_motivo,
        clave_hmac_motivo_ref, valor_hmac_motivo, registrada_en
    ) VALUES (
        p_reserva ->> 'huella_material_sha256', p_material_canonico,
        v_material ->> 'esquema', v_material ->> 'accion',
        v_estado_nuevo ->> 'referencia',
        (v_estado_nuevo ->> 'revision')::bigint,
        v_estado_nuevo ->> 'huella_estado_sha256',
        CASE WHEN v_estado_esperado IS NULL THEN NULL
             ELSE (v_estado_esperado ->> 'revision')::bigint END,
        v_estado_esperado ->> 'huella_estado_sha256',
        v_material ->> 'dominio_criptografico_motivo',
        (v_material ->> 'generacion_clave_motivo')::bigint,
        v_partes_hmac[1], decode(v_partes_hmac[2], 'hex'), v_ahora
    ) ON CONFLICT (huella_material_sha256) DO NOTHING;
    IF NOT EXISTS (
        SELECT 1 FROM vec_bolsa_convocatorias.material_borrador AS m
         WHERE m.huella_material_sha256 =
               p_reserva ->> 'huella_material_sha256'
           AND m.material_canonico = p_material_canonico
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'XX001',
            MESSAGE = 'colision de material de borrador';
    END IF;

    INSERT INTO vec_bolsa_convocatorias.diario_borrador_version(
        localizador_esquema_version, localizador_clave_ref,
        localizador_generacion_clave, localizador_hmac, revision,
        huella_esquema_version, huella_clave_ref,
        huella_generacion_clave, huella_hmac, estado, cercado,
        arrendamiento_inicia_en, arrendamiento_vence_en, accion,
        huella_material_sha256, recurso_ref,
        contexto_recurso_huella_sha256, esquema_huella_decision,
        decision_ref, huella_decision_sha256, modulo_id, tipo_recurso,
        finalidad, version_rol_ref, version_rol_huella_sha256,
        control_vigencia_rol_revision, revision_catalogo_politicas,
        catalogo_politicas_huella_sha256, decision_verificada_en,
        decision_valida_hasta, atestacion_ref, atestacion_version,
        atestacion_estado, huella_atestacion_sha256, verificador_ref,
        atestacion_verificada_en, registrada_en,
        asignacion_ref, asignacion_huella_sha256,
        control_vigencia_version_rol_ref,
        control_vigencia_version_rol_huella_sha256, decision_emitida_en
    ) VALUES (
        (v_l ->> 'version_esquema')::integer, v_l ->> 'clave_ref',
        (v_l ->> 'generacion_clave')::bigint,
        decode(v_l ->> 'hmac_sha256', 'hex'), 1,
        (v_f ->> 'version_esquema')::integer, v_f ->> 'clave_ref',
        (v_f ->> 'generacion_clave')::bigint,
        decode(v_f ->> 'hmac_sha256', 'hex'), 'reservado', 1,
        (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz,
        (p_reserva ->> 'arrendamiento_vence_en')::timestamptz,
        p_reserva ->> 'accion', p_reserva ->> 'huella_material_sha256',
        p_reserva ->> 'recurso_ref',
        p_reserva ->> 'contexto_recurso_huella_sha256',
        v_d ->> 'esquema_huella', v_d ->> 'decision_ref',
        v_d ->> 'huella_decision_sha256', v_d ->> 'modulo_id',
        v_d ->> 'tipo_recurso', v_d ->> 'finalidad',
        v_d ->> 'version_rol_ref', v_d ->> 'version_rol_huella_sha256',
        (v_d ->> 'control_vigencia_version_rol_revision')::bigint,
        (v_d ->> 'revision_catalogo_politicas')::bigint,
        v_d ->> 'catalogo_politicas_huella_sha256',
        (v_d ->> 'verificada_en')::timestamptz,
        (v_d ->> 'valida_hasta')::timestamptz,
        v_a ->> 'atestacion_ref', (v_a ->> 'version')::bigint,
        v_a ->> 'estado', v_a ->> 'huella_atestacion_sha256',
        v_a ->> 'verificador_ref',
        (v_a ->> 'verificada_en')::timestamptz, v_ahora,
        v_d ->> 'asignacion_ref', v_d ->> 'asignacion_huella_sha256',
        v_d ->> 'control_vigencia_version_rol_ref',
        v_d ->> 'control_vigencia_version_rol_huella_sha256',
        (v_d ->> 'emitida_en')::timestamptz
    );
    INSERT INTO vec_bolsa_convocatorias.diario_borrador_actual VALUES (
        (v_l ->> 'version_esquema')::integer, v_l ->> 'clave_ref',
        (v_l ->> 'generacion_clave')::bigint,
        decode(v_l ->> 'hmac_sha256', 'hex'), 1, v_ahora
    );
    INSERT INTO vec_bolsa_convocatorias.identidad_alias_borrador(
        alias_localizador_esquema_version,
        alias_localizador_clave_ref,
        alias_localizador_generacion_clave,
        alias_localizador_hmac,
        alias_huella_esquema_version,
        alias_huella_clave_ref,
        alias_huella_generacion_clave,
        alias_huella_hmac,
        primario_localizador_esquema_version,
        primario_localizador_clave_ref,
        primario_localizador_generacion_clave,
        primario_localizador_hmac,
        primario_huella_esquema_version,
        primario_huella_clave_ref,
        primario_huella_generacion_clave,
        primario_huella_hmac,
        revision_origen, ordinalidad, registrada_en
    )
    SELECT
        (i.elemento -> 'localizador' ->> 'version_esquema')::integer,
        i.elemento -> 'localizador' ->> 'clave_ref',
        (i.elemento -> 'localizador' ->> 'generacion_clave')::bigint,
        decode(i.elemento -> 'localizador' ->> 'hmac_sha256', 'hex'),
        (i.elemento -> 'huella_solicitud' ->>
         'version_esquema')::integer,
        i.elemento -> 'huella_solicitud' ->> 'clave_ref',
        (i.elemento -> 'huella_solicitud' ->>
         'generacion_clave')::bigint,
        decode(i.elemento -> 'huella_solicitud' ->>
               'hmac_sha256', 'hex'),
        (v_l ->> 'version_esquema')::integer,
        v_l ->> 'clave_ref',
        (v_l ->> 'generacion_clave')::bigint,
        decode(v_l ->> 'hmac_sha256', 'hex'),
        (v_f ->> 'version_esquema')::integer,
        v_f ->> 'clave_ref',
        (v_f ->> 'generacion_clave')::bigint,
        decode(v_f ->> 'hmac_sha256', 'hex'),
        1, i.ordinalidad, v_ahora
      FROM jsonb_array_elements(p_reserva -> 'identidades_consulta')
           WITH ORDINALITY AS i(elemento, ordinalidad);
    v_consumo_ref := 'consumo-decision-borrador-' || encode(sha256(convert_to(
        v_d ->> 'decision_ref' || ':' || encode(
            decode(v_l ->> 'hmac_sha256', 'hex'), 'hex'
        ), 'UTF8'
    )), 'hex');
    INSERT INTO vec_bolsa_convocatorias.uso_decision_borrador VALUES (
        v_consumo_ref, v_d ->> 'decision_ref',
        v_d ->> 'huella_decision_sha256', v_a ->> 'atestacion_ref',
        (v_a ->> 'version')::bigint, v_a ->> 'estado',
        v_a ->> 'huella_atestacion_sha256',
        (v_l ->> 'version_esquema')::integer, v_l ->> 'clave_ref',
        (v_l ->> 'generacion_clave')::bigint,
        decode(v_l ->> 'hmac_sha256', 'hex'), 1, 1,
        p_reserva ->> 'accion', p_reserva ->> 'recurso_ref',
        p_reserva ->> 'huella_material_sha256', v_ahora
    );
    RETURN QUERY SELECT 'reservado'::text, 1::bigint, 1::bigint,
        (p_reserva ->> 'arrendamiento_inicia_en')::timestamptz,
        (p_reserva ->> 'arrendamiento_vence_en')::timestamptz,
        NULL::jsonb, p_reserva -> 'identidades_consulta', v_identidad;
END
$funcion$;

-- La reserva y su resolucion de identidad tienen contrato tecnico explicito.
-- Esto no completa el adaptador PostgreSQL ni habilita la confirmacion: faltan
-- composicion runtime, atestacion del derivador HMAC y contrato KMS real.
CREATE FUNCTION vec_bolsa_convocatorias.reservar_decision_borrador_v1(
    p_reserva jsonb, p_prueba jsonb, p_material_canonico bytea,
    p_version_canonica bytea, p_decision_canonica bytea,
    p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    estado text, revision bigint, cercado bigint,
    arrendamiento_inicia_en timestamptz,
    arrendamiento_vence_en timestamptz,
    recibo jsonb, identidades_consultadas jsonb,
    identidad_primaria jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_proyector_gobierno', true
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           p_reserva ->> 'accion', 'version_convocatoria_gobernada',
           p_reserva ->> 'recurso_ref',
           'gobierno_convocatorias',
           '["auditoria","evento_outbox","version_convocatoria"]'::jsonb,
           clock_timestamp()
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'decision de reserva no revalidada',
            DETAIL = 'politica o atestacion productiva no revalidada';
    END IF;
    RETURN QUERY SELECT *
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          p_reserva, p_material_canonico, p_version_canonica,
          p_decision_canonica, p_contexto_recurso_canonico
      );
END
$funcion$;

REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
COMMIT;

BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_pdp_borrador', 'material_borrador',
        'diario_borrador_version', 'identidad_alias_borrador',
        'prueba_desenlace_borrador',
        'uso_decision_borrador',
        'sellado_motivo_borrador', 'borrador_convocatoria_version',
        'auditoria_borrador', 'outbox_borrador',
        'uso_decision_lectura_borrador', 'auditoria_lectura_borrador',
        'cursor_listado_borrador'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_convocatorias.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'diario_borrador_actual', 'borrador_convocatoria_actual',
        'auditoria_borrador_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
END
$protecciones$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_pdp_borrador', 'material_borrador',
        'diario_borrador_version', 'diario_borrador_actual',
        'identidad_alias_borrador',
        'prueba_desenlace_borrador',
        'uso_decision_borrador', 'sellado_motivo_borrador',
        'borrador_convocatoria_version', 'borrador_convocatoria_actual',
        'auditoria_borrador', 'auditoria_borrador_actual',
        'outbox_borrador', 'uso_decision_lectura_borrador',
        'auditoria_lectura_borrador', 'cursor_listado_borrador'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_convocatorias.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_convocatorias.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_convocatorias.%I FOR ALL TO vec_bolsa_convocatorias_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_convocatorias_propietario',
            'vec_bolsa_convocatorias_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;

DO $cerrar_runtime$
DECLARE
    rol text;
    funcion record;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion'
    ] LOOP
        FOR funcion IN
            SELECT p.oid::regprocedure AS firma
              FROM pg_catalog.pg_proc AS p
              JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
        LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',
                           funcion.firma, rol);
        END LOOP;
        EXECUTE format(
            'REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_convocatorias FROM %I',
            rol
        );
        EXECUTE format(
            'REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_convocatorias FROM %I',
            rol
        );
        EXECUTE format(
            'REVOKE ALL ON SCHEMA vec_bolsa_convocatorias FROM %I', rol
        );
    END LOOP;
END
$cerrar_runtime$;

-- PostgreSQL no dispone de REVOKE ... ON ALL TYPES IN SCHEMA. Los tipos de
-- fila implícitos de tablas se gobiernan con la ACL de su relación; para los
-- tipos autónomos se cierra individualmente cualquier definición presente.
DO $cerrar_tipos$
DECLARE
    rol text;
    tipo record;
BEGIN
    FOR tipo IN
        SELECT t.oid::regtype AS firma
          FROM pg_catalog.pg_type AS t
          LEFT JOIN pg_catalog.pg_class AS c ON c.oid = t.typrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND t.typisdefined AND t.typelem = 0
           AND (t.typrelid = 0 OR c.relkind = 'c')
    LOOP
        EXECUTE format('REVOKE ALL ON TYPE %s FROM PUBLIC', tipo.firma);
        FOREACH rol IN ARRAY ARRAY[
            'vec_bolsa_convocatorias_ejecutor_consulta',
            'vec_bolsa_convocatorias_proyector_gobierno',
            'vec_bolsa_convocatorias_registrador_atestacion'
        ] LOOP
            EXECUTE format(
                'REVOKE ALL ON TYPE %s FROM %I', tipo.firma, rol
            );
        END LOOP;
    END LOOP;
END
$cerrar_tipos$;

-- Apertura minima: solo wrappers SECURITY DEFINER nominales. Los grupos no
-- reciben tablas, secuencias, funciones internas ni pertenencia al propietario.
GRANT USAGE ON SCHEMA vec_bolsa_convocatorias
    TO vec_bolsa_convocatorias_ejecutor_consulta,
       vec_bolsa_convocatorias_proyector_gobierno;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.listar_borradores_v1(
        jsonb, jsonb, jsonb, bytea, bytea
    ),
    vec_bolsa_convocatorias.obtener_borrador_v1(
        text, jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_convocatorias_ejecutor_consulta;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb),
    vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(
        jsonb, text, bigint, bigint, timestamptz
    ),
    vec_bolsa_convocatorias.reservar_decision_borrador_v1(
        jsonb, jsonb, bytea, bytea, bytea, bytea
    ),
    vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(
        bigint, bigint, jsonb, jsonb, bytea, bytea, bytea, bytea
    ) TO vec_bolsa_convocatorias_proyector_gobierno;

COMMENT ON FUNCTION
    vec_bolsa_convocatorias.reservar_decision_borrador_v1(
        jsonb, jsonb, bytea, bytea, bytea, bytea
    ) IS
    'Reserva post-PDP multigeneracion. Solo el proyector runtime endurecido puede ejecutarla; el cifrado permanece fail-closed hasta recibir un sobre KMS autentico.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.confirmar_borrador_v1(
        jsonb, jsonb, bytea, bytea, bytea, bytea, bytea
    ) IS
    'NO-GO incondicional: siempre aborta con SQLSTATE 55000 aunque reciba EXECUTE accidental; una migracion posterior debera reemplazar el stub tras cerrar perfil, AAD, DEK envuelta y atestacion KMS autoritativa.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.listar_borradores_v1(
        jsonb, jsonb, jsonb, bytea, bytea
    ) IS
    'Listado maximo 50 sobre proyeccion ligera, filtros y cursor opaco ligados al ambito; lectura atestada, consumida y auditada.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.obtener_borrador_v1(
        text, jsonb, jsonb, bytea, bytea
    ) IS
    'Detalle exacto cifrado, limitado por ambito y con lectura atestada, consumida y auditada.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb) IS
    'Consulta pre-PDP por hasta cuatro testigos L/F HMAC; reidentifica el LOGIN proyector y nunca expone tablas ni principal.';
COMMENT ON FUNCTION
    vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(
        jsonb, text, bigint, bigint, timestamptz
    ) IS
    'Recovery pre-PDP ligado al testigo L/F, revision y cercado exactos; solo para el LOGIN proyector de privilegio minimo.';

COMMIT;
