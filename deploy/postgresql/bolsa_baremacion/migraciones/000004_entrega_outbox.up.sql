-- La entrega es un agregado separado del evento de dominio: el evento queda
-- inmutable y siempre en estado "pendiente". Los intentos, arrendamientos y
-- cursores se conservan como historia append-only por consumidor.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

ALTER TABLE vec_bolsa_baremacion.evento_outbox
    ADD CONSTRAINT evento_outbox_referencia_secuencia_unica
        UNIQUE (referencia, secuencia);

-- Se configura por una migracion operativa ejecutada como propietario. No se
-- ofrece alta al runtime: si no existe un enlace exacto consumidor/login, las
-- funciones de entrega fallan cerradas.
CREATE TABLE vec_bolsa_baremacion.consumidor_outbox_version (
    consumidor_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    rol_sesion text NOT NULL,
    secuencia_inicial numeric(20, 0) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    PRIMARY KEY (consumidor_ref, version),
    UNIQUE (consumidor_ref, version, estado),
    CONSTRAINT consumidor_outbox_version_rango CHECK (
        version BETWEEN 1 AND 18446744073709551615
        AND secuencia_inicial BETWEEN 0 AND 18446744073709551615
    ),
    CONSTRAINT consumidor_outbox_version_perfil CHECK (
        estado IN ('activo', 'revocado')
        AND vec_bolsa_baremacion.texto_opaco_valido(consumidor_ref, 256)
        AND vec_bolsa_baremacion.texto_opaco_valido(rol_sesion, 63)
        AND vec_bolsa_baremacion.texto_opaco_valido(acto_ref, 512)
    )
);

CREATE TABLE vec_bolsa_baremacion.consumidor_outbox_actual (
    consumidor_ref text PRIMARY KEY,
    version numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (consumidor_ref, version, estado)
        REFERENCES vec_bolsa_baremacion.consumidor_outbox_version(
            consumidor_ref, version, estado
        )
);

CREATE TABLE vec_bolsa_baremacion.entrega_outbox_version (
    consumidor_ref text NOT NULL,
    evento_ref text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    intento numeric(20, 0) NOT NULL,
    consumidor_version numeric(20, 0) NOT NULL,
    huella_arrendamiento_sha256 text NOT NULL,
    reclamada_en timestamptz(6) NOT NULL,
    arrendamiento_expira_en timestamptz(6) NOT NULL,
    finalizada_en timestamptz(6),
    error_clave text,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (consumidor_ref, evento_ref, revision),
    UNIQUE (consumidor_ref, evento_ref, revision, estado),
    UNIQUE (consumidor_ref, evento_ref, revision, estado, intento),
    FOREIGN KEY (evento_ref)
        REFERENCES vec_bolsa_baremacion.evento_outbox(referencia),
    FOREIGN KEY (consumidor_ref, consumidor_version)
        REFERENCES vec_bolsa_baremacion.consumidor_outbox_version(
            consumidor_ref, version
        ),
    CONSTRAINT entrega_outbox_revision_rango CHECK (
        revision BETWEEN 1 AND 18446744073709551615
        AND intento BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT entrega_outbox_perfil CHECK (
        estado IN ('reclamada', 'expirada', 'entregada', 'fallida')
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_arrendamiento_sha256
        )
        AND arrendamiento_expira_en > reclamada_en
        AND arrendamiento_expira_en <= reclamada_en + interval '5 minutes'
        AND registrada_en >= reclamada_en
        AND (
            (estado = 'reclamada' AND finalizada_en IS NULL
             AND error_clave IS NULL)
            OR
            (estado = 'entregada' AND finalizada_en IS NOT NULL
             AND error_clave IS NULL)
            OR
            (estado = 'expirada' AND finalizada_en IS NOT NULL
             AND error_clave = 'arrendamiento_expirado')
            OR
            (estado = 'fallida' AND finalizada_en IS NOT NULL
             AND vec_bolsa_baremacion.texto_opaco_valido(
                 error_clave, 128
             ))
        )
        AND (finalizada_en IS NULL OR (
            finalizada_en >= reclamada_en
            AND registrada_en >= finalizada_en
        ))
    )
);

CREATE TABLE vec_bolsa_baremacion.entrega_outbox_actual (
    consumidor_ref text NOT NULL,
    evento_ref text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    intento numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (consumidor_ref, evento_ref),
    FOREIGN KEY (consumidor_ref, evento_ref, revision, estado, intento)
        REFERENCES vec_bolsa_baremacion.entrega_outbox_version(
            consumidor_ref, evento_ref, revision, estado, intento
        )
);

CREATE TABLE vec_bolsa_baremacion.cursor_outbox_version (
    consumidor_ref text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    secuencia numeric(20, 0) NOT NULL,
    evento_ref text NOT NULL,
    entrega_revision numeric(20, 0) NOT NULL,
    estado_entrega text NOT NULL,
    avanzada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (consumidor_ref, revision),
    UNIQUE (consumidor_ref, revision, secuencia, evento_ref),
    UNIQUE (consumidor_ref, secuencia),
    FOREIGN KEY (evento_ref, secuencia)
        REFERENCES vec_bolsa_baremacion.evento_outbox(
            referencia, secuencia
        ),
    FOREIGN KEY (
        consumidor_ref, evento_ref, entrega_revision, estado_entrega
    ) REFERENCES vec_bolsa_baremacion.entrega_outbox_version(
        consumidor_ref, evento_ref, revision, estado
    ),
    CONSTRAINT cursor_outbox_rango CHECK (
        revision BETWEEN 1 AND 18446744073709551615
        AND secuencia BETWEEN 1 AND 18446744073709551615
        AND entrega_revision BETWEEN 1 AND 18446744073709551615
        AND estado_entrega = 'entregada'
    )
);

CREATE TABLE vec_bolsa_baremacion.cursor_outbox_actual (
    consumidor_ref text PRIMARY KEY,
    revision numeric(20, 0) NOT NULL,
    secuencia numeric(20, 0) NOT NULL,
    evento_ref text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (consumidor_ref, revision, secuencia, evento_ref)
        REFERENCES vec_bolsa_baremacion.cursor_outbox_version(
            consumidor_ref, revision, secuencia, evento_ref
        )
);

CREATE INDEX entrega_outbox_actual_estado
    ON vec_bolsa_baremacion.entrega_outbox_actual(
        consumidor_ref, estado, actualizada_en
    );

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_consumidor_outbox()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.consumidor_ref IS DISTINCT FROM OLD.consumidor_ref
       OR NEW.version IS DISTINCT FROM OLD.version + 1
       OR OLD.estado <> 'activo'
       OR NEW.estado NOT IN ('activo', 'revocado')
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de consumidor outbox invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_entrega_outbox()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.consumidor_ref IS DISTINCT FROM OLD.consumidor_ref
       OR NEW.evento_ref IS DISTINCT FROM OLD.evento_ref
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1
       OR NEW.actualizada_en < OLD.actualizada_en
       OR NOT (
           (OLD.estado = 'reclamada' AND NEW.estado IN (
               'expirada', 'entregada', 'fallida'
           ) AND NEW.intento = OLD.intento)
           OR
           (OLD.estado IN ('expirada', 'fallida')
            AND NEW.estado = 'reclamada'
            AND NEW.intento = OLD.intento + 1)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de entrega outbox invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.validar_avance_cursor_outbox()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.consumidor_ref IS DISTINCT FROM OLD.consumidor_ref
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1
       OR NEW.secuencia <= OLD.secuencia
       OR NEW.actualizada_en < OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de cursor outbox invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER consumidor_outbox_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.consumidor_outbox_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_consumidor_outbox();
CREATE TRIGGER entrega_outbox_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.entrega_outbox_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_entrega_outbox();
CREATE TRIGGER cursor_outbox_actual_avance
    BEFORE UPDATE ON vec_bolsa_baremacion.cursor_outbox_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_avance_cursor_outbox();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'consumidor_outbox_version', 'entrega_outbox_version',
        'cursor_outbox_version'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_baremacion.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_baremacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'consumidor_outbox_actual', 'entrega_outbox_actual',
        'cursor_outbox_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_bolsa_baremacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
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
        'consumidor_outbox_version', 'consumidor_outbox_actual',
        'entrega_outbox_version', 'entrega_outbox_actual',
        'cursor_outbox_version', 'cursor_outbox_actual'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_baremacion.%I FOR ALL TO vec_bolsa_baremacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_baremacion_propietario',
            'vec_bolsa_baremacion_propietario'
        );
    END LOOP;
END
$rls$;

CREATE FUNCTION vec_bolsa_baremacion.autenticar_consumidor_outbox(
    p_consumidor_ref text
)
RETURNS TABLE (consumidor_version numeric, secuencia_inicial numeric)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF vec_bolsa_baremacion.texto_opaco_valido(
        p_consumidor_ref, 256
    ) IS NOT TRUE THEN
        RETURN;
    END IF;
    RETURN QUERY
    SELECT version.version, version.secuencia_inicial
      FROM vec_bolsa_baremacion.consumidor_outbox_actual AS actual
      JOIN vec_bolsa_baremacion.consumidor_outbox_version AS version
        ON version.consumidor_ref = actual.consumidor_ref
       AND version.version = actual.version
       AND version.estado = actual.estado
     WHERE actual.consumidor_ref = p_consumidor_ref
       AND actual.estado = 'activo'
       AND version.rol_sesion = session_user::text
       AND EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles AS rol
            WHERE rol.rolname = session_user::text AND rol.rolcanlogin
       )
     FOR SHARE OF actual;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.reclamar_evento_outbox(
    p_consumidor_ref text,
    p_token_arrendamiento bytea,
    p_duracion_segundos integer
)
RETURNS TABLE (
    resultado text,
    evento_ref text,
    secuencia text,
    tipo text,
    carga jsonb,
    arrendamiento_expira_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
#variable_conflict use_variable
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    autenticacion record;
    curso numeric(20, 0);
    evento record;
    entrega record;
    huella_token text;
    nueva_revision numeric(20, 0);
    nuevo_intento numeric(20, 0);
    revision_anterior numeric(20, 0);
BEGIN
    resultado := 'rechazada';
    evento_ref := '';
    secuencia := '';
    tipo := '';
    IF p_token_arrendamiento IS NULL
       OR octet_length(p_token_arrendamiento) NOT BETWEEN 32 AND 1024
       OR p_duracion_segundos IS NULL
       OR p_duracion_segundos NOT BETWEEN 5 AND 300 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autenticacion
      FROM vec_bolsa_baremacion.autenticar_consumidor_outbox(
          p_consumidor_ref
      );
    IF NOT FOUND THEN
        resultado := 'consumidor_no_autorizado';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:outbox:' || p_consumidor_ref, 0
    ));
    SELECT actual.secuencia INTO curso
      FROM vec_bolsa_baremacion.cursor_outbox_actual AS actual
     WHERE actual.consumidor_ref = p_consumidor_ref
     FOR SHARE;
    IF NOT FOUND THEN
        curso := autenticacion.secuencia_inicial;
    END IF;
    SELECT * INTO evento
      FROM vec_bolsa_baremacion.evento_outbox AS pendiente
     WHERE pendiente.secuencia > curso
     ORDER BY pendiente.secuencia
     LIMIT 1;
    IF NOT FOUND THEN
        resultado := 'sin_evento';
        RETURN NEXT;
        RETURN;
    END IF;
    huella_token := encode(sha256(p_token_arrendamiento), 'hex');
    SELECT version.* INTO entrega
      FROM vec_bolsa_baremacion.entrega_outbox_actual AS actual
      JOIN vec_bolsa_baremacion.entrega_outbox_version AS version
        ON version.consumidor_ref = actual.consumidor_ref
       AND version.evento_ref = actual.evento_ref
       AND version.revision = actual.revision
       AND version.estado = actual.estado
       AND version.intento = actual.intento
     WHERE actual.consumidor_ref = p_consumidor_ref
       AND actual.evento_ref = evento.referencia
     FOR UPDATE OF actual;
    IF NOT FOUND THEN
        nueva_revision := 1;
        nuevo_intento := 1;
        INSERT INTO vec_bolsa_baremacion.entrega_outbox_version (
            consumidor_ref, evento_ref, revision, estado, intento,
            consumidor_version, huella_arrendamiento_sha256,
            reclamada_en, arrendamiento_expira_en, registrada_en
        ) VALUES (
            p_consumidor_ref, evento.referencia, nueva_revision,
            'reclamada', nuevo_intento, autenticacion.consumidor_version,
            huella_token, ahora,
            ahora + make_interval(secs => p_duracion_segundos), ahora
        );
        INSERT INTO vec_bolsa_baremacion.entrega_outbox_actual (
            consumidor_ref, evento_ref, revision, estado, intento,
            actualizada_en
        ) VALUES (
            p_consumidor_ref, evento.referencia, nueva_revision,
            'reclamada', nuevo_intento, ahora
        );
    ELSE
        IF entrega.estado = 'entregada' THEN
            resultado := 'estado_inconsistente';
            RETURN NEXT;
            RETURN;
        END IF;
        IF entrega.estado = 'reclamada'
           AND ahora < entrega.arrendamiento_expira_en THEN
            IF entrega.huella_arrendamiento_sha256 = huella_token THEN
                nueva_revision := entrega.revision;
                nuevo_intento := entrega.intento;
            ELSE
                resultado := 'ocupada';
                RETURN NEXT;
                RETURN;
            END IF;
        ELSE
            IF entrega.estado = 'reclamada' THEN
                INSERT INTO vec_bolsa_baremacion.entrega_outbox_version (
                    consumidor_ref, evento_ref, revision, estado, intento,
                    consumidor_version, huella_arrendamiento_sha256,
                    reclamada_en, arrendamiento_expira_en, finalizada_en,
                    error_clave, registrada_en
                ) VALUES (
                    p_consumidor_ref, evento.referencia,
                    entrega.revision + 1, 'expirada', entrega.intento,
                    entrega.consumidor_version,
                    entrega.huella_arrendamiento_sha256,
                    entrega.reclamada_en, entrega.arrendamiento_expira_en,
                    ahora, 'arrendamiento_expirado', ahora
                );
                UPDATE vec_bolsa_baremacion.entrega_outbox_actual
                   SET revision = entrega.revision + 1,
                       estado = 'expirada', actualizada_en = ahora
                 WHERE consumidor_ref = p_consumidor_ref
                   AND evento_ref = evento.referencia
                   AND revision = entrega.revision;
                IF NOT FOUND THEN
                    RAISE EXCEPTION USING ERRCODE = '40001',
                        MESSAGE = 'CAS de expiracion outbox perdido';
                END IF;
                entrega.revision := entrega.revision + 1;
                entrega.estado := 'expirada';
            END IF;
            nueva_revision := entrega.revision + 1;
            nuevo_intento := entrega.intento + 1;
            revision_anterior := entrega.revision;
            INSERT INTO vec_bolsa_baremacion.entrega_outbox_version (
                consumidor_ref, evento_ref, revision, estado, intento,
                consumidor_version, huella_arrendamiento_sha256,
                reclamada_en, arrendamiento_expira_en, registrada_en
            ) VALUES (
                p_consumidor_ref, evento.referencia, nueva_revision,
                'reclamada', nuevo_intento,
                autenticacion.consumidor_version, huella_token, ahora,
                ahora + make_interval(secs => p_duracion_segundos), ahora
            );
            UPDATE vec_bolsa_baremacion.entrega_outbox_actual AS actual
               SET revision = nueva_revision, estado = 'reclamada',
                   intento = nuevo_intento, actualizada_en = ahora
             WHERE actual.consumidor_ref = p_consumidor_ref
               AND actual.evento_ref = evento.referencia
               AND actual.revision = revision_anterior;
            IF NOT FOUND THEN
                RAISE EXCEPTION USING ERRCODE = '40001',
                    MESSAGE = 'CAS de reclamacion outbox perdido';
            END IF;
        END IF;
    END IF;
    resultado := 'reclamada';
    evento_ref := evento.referencia;
    secuencia := evento.secuencia::text;
    tipo := evento.tipo;
    arrendamiento_expira_en := ahora
        + make_interval(secs => p_duracion_segundos);
    IF entrega.estado = 'reclamada'
       AND entrega.huella_arrendamiento_sha256 = huella_token
       AND ahora < entrega.arrendamiento_expira_en THEN
        arrendamiento_expira_en := entrega.arrendamiento_expira_en;
    END IF;
    carga := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.evento-outbox.v1',
        'referencia', evento.referencia,
        'secuencia', evento.secuencia::text,
        'tipo', evento.tipo,
        'modulo', evento.modulo,
        'proceso_ref', evento.proceso_ref,
        'solicitud_ref', evento.solicitud_ref,
        'baremacion_merito_ref', evento.baremacion_merito_ref,
        'decision_ref', evento.decision_ref,
        'manifiesto_probatorio_ref', evento.manifiesto_probatorio_ref,
        'huella_manifiesto_sha256', evento.huella_manifiesto_sha256,
        'documento_firmado_ref', evento.documento_firmado_ref,
        'evidencia_custodia_firmado_ref',
            evento.evidencia_custodia_firmado_ref,
        'evidencia_retencion_firmado_ref',
            evento.evidencia_retencion_firmado_ref,
        'sujeto_ref', evento.sujeto_ref,
        'principal_ref', evento.principal_ref,
        'version_nueva', evento.version_nueva::text,
        'huella_nueva_sha256', evento.huella_nueva_sha256,
        'auditoria_ref', evento.auditoria_ref,
        'huella_auditoria_sha256', evento.huella_auditoria_sha256,
        'correlacion_ref', evento.correlacion_ref,
        'registrada_en',
            vec_bolsa_baremacion.instante_rfc3339nano(evento.registrada_en),
        'huella_evento_anterior_sha256',
            evento.huella_evento_anterior_sha256,
        'huella_registro_sha256', evento.huella_registro_sha256
    );
    RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.finalizar_entrega_outbox(
    p_consumidor_ref text,
    p_evento_ref text,
    p_token_arrendamiento bytea,
    p_resultado text,
    p_error_clave text
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
#variable_conflict use_variable
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    autenticacion record;
    entrega record;
    evento record;
    cursor_actual record;
    curso numeric(20, 0);
    huella_token text;
    nueva_revision numeric(20, 0);
    revision_cursor numeric(20, 0);
BEGIN
    IF vec_bolsa_baremacion.texto_opaco_valido(
           p_evento_ref, 512
       ) IS NOT TRUE
       OR p_token_arrendamiento IS NULL
       OR octet_length(p_token_arrendamiento) NOT BETWEEN 32 AND 1024
       OR p_resultado IS NULL
       OR p_resultado NOT IN ('entregada', 'fallida')
       OR (p_resultado = 'entregada' AND p_error_clave IS NOT NULL)
       OR (p_resultado = 'fallida' AND
           vec_bolsa_baremacion.texto_opaco_valido(
               p_error_clave, 128
           ) IS NOT TRUE) THEN
        RETURN 'rechazada';
    END IF;
    SELECT * INTO autenticacion
      FROM vec_bolsa_baremacion.autenticar_consumidor_outbox(
          p_consumidor_ref
      );
    IF NOT FOUND THEN
        RETURN 'consumidor_no_autorizado';
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:outbox:' || p_consumidor_ref, 0
    ));
    huella_token := encode(sha256(p_token_arrendamiento), 'hex');
    SELECT version.* INTO entrega
      FROM vec_bolsa_baremacion.entrega_outbox_actual AS actual
      JOIN vec_bolsa_baremacion.entrega_outbox_version AS version
        ON version.consumidor_ref = actual.consumidor_ref
       AND version.evento_ref = actual.evento_ref
       AND version.revision = actual.revision
       AND version.estado = actual.estado
       AND version.intento = actual.intento
     WHERE actual.consumidor_ref = p_consumidor_ref
       AND actual.evento_ref = p_evento_ref
     FOR UPDATE OF actual;
    IF NOT FOUND THEN
        RETURN 'arrendamiento_invalido';
    END IF;
    IF entrega.estado = p_resultado
       AND entrega.huella_arrendamiento_sha256 = huella_token
       AND entrega.error_clave IS NOT DISTINCT FROM p_error_clave THEN
        RETURN p_resultado;
    END IF;
    IF entrega.estado <> 'reclamada'
       OR entrega.huella_arrendamiento_sha256 <> huella_token
       OR entrega.consumidor_version <> autenticacion.consumidor_version
       OR ahora >= entrega.arrendamiento_expira_en THEN
        RETURN 'arrendamiento_obsoleto';
    END IF;
    SELECT * INTO evento
      FROM vec_bolsa_baremacion.evento_outbox
     WHERE referencia = p_evento_ref;
    IF NOT FOUND THEN
        RETURN 'evento_no_encontrado';
    END IF;
    SELECT * INTO cursor_actual
      FROM vec_bolsa_baremacion.cursor_outbox_actual
     WHERE consumidor_ref = p_consumidor_ref
     FOR UPDATE;
    IF FOUND THEN
        curso := cursor_actual.secuencia;
    ELSE
        curso := autenticacion.secuencia_inicial;
    END IF;
    IF evento.secuencia <> (
        SELECT min(pendiente.secuencia)
          FROM vec_bolsa_baremacion.evento_outbox AS pendiente
         WHERE pendiente.secuencia > curso
    ) THEN
        RETURN 'orden_invalido';
    END IF;
    nueva_revision := entrega.revision + 1;
    INSERT INTO vec_bolsa_baremacion.entrega_outbox_version (
        consumidor_ref, evento_ref, revision, estado, intento,
        consumidor_version, huella_arrendamiento_sha256,
        reclamada_en, arrendamiento_expira_en, finalizada_en,
        error_clave, registrada_en
    ) VALUES (
        p_consumidor_ref, p_evento_ref, nueva_revision, p_resultado,
        entrega.intento, autenticacion.consumidor_version, huella_token,
        entrega.reclamada_en, entrega.arrendamiento_expira_en, ahora,
        p_error_clave, ahora
    );
    UPDATE vec_bolsa_baremacion.entrega_outbox_actual
       SET revision = nueva_revision, estado = p_resultado,
           actualizada_en = ahora
     WHERE consumidor_ref = p_consumidor_ref
       AND evento_ref = p_evento_ref AND revision = entrega.revision;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de finalizacion outbox perdido';
    END IF;
    IF p_resultado = 'fallida' THEN
        RETURN 'fallida';
    END IF;
    IF cursor_actual.consumidor_ref IS NULL THEN
        revision_cursor := 1;
        INSERT INTO vec_bolsa_baremacion.cursor_outbox_version (
            consumidor_ref, revision, secuencia, evento_ref,
            entrega_revision, estado_entrega, avanzada_en
        ) VALUES (
            p_consumidor_ref, revision_cursor, evento.secuencia,
            p_evento_ref, nueva_revision, 'entregada', ahora
        );
        INSERT INTO vec_bolsa_baremacion.cursor_outbox_actual (
            consumidor_ref, revision, secuencia, evento_ref, actualizada_en
        ) VALUES (
            p_consumidor_ref, revision_cursor, evento.secuencia,
            p_evento_ref, ahora
        );
    ELSE
        revision_cursor := cursor_actual.revision + 1;
        INSERT INTO vec_bolsa_baremacion.cursor_outbox_version (
            consumidor_ref, revision, secuencia, evento_ref,
            entrega_revision, estado_entrega, avanzada_en
        ) VALUES (
            p_consumidor_ref, revision_cursor, evento.secuencia,
            p_evento_ref, nueva_revision, 'entregada', ahora
        );
        UPDATE vec_bolsa_baremacion.cursor_outbox_actual
           SET revision = revision_cursor, secuencia = evento.secuencia,
               evento_ref = p_evento_ref, actualizada_en = ahora
         WHERE consumidor_ref = p_consumidor_ref
           AND revision = cursor_actual.revision;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS de cursor outbox perdido';
        END IF;
    END IF;
    RETURN 'entregada';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_baremacion.autenticar_consumidor_outbox(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.reclamar_evento_outbox(
    text, bytea, integer
) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_baremacion.finalizar_entrega_outbox(
    text, text, bytea, text, text
) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_baremacion
    TO vec_bolsa_baremacion_lector_outbox;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.reclamar_evento_outbox(
    text, bytea, integer
) TO vec_bolsa_baremacion_lector_outbox;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.finalizar_entrega_outbox(
    text, text, bytea, text, text
) TO vec_bolsa_baremacion_lector_outbox;
COMMIT;
