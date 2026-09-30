\set ON_ERROR_STOP on
-- Lectura VOLATILE de un uso y su publicacion para la misma sentencia que
-- reserva o termina el uso. Requiere 000001 y 000002; no concede acceso LOGIN.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000003',0));
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.consultar_uso(text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)') IS NOT NULL
       OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
           'vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text)','EXECUTE') THEN
        RAISE EXCEPTION 'catalogos configurables 000003: preimagen incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;

-- El recibo terminal y la evidencia opaca quedan ligados de modo inmutable.
-- El modulo consumidor comprueba el efecto (o su ausencia) antes de emitir V3;
-- esta tabla no pretende verificar ni interpretar sus hechos privados.
CREATE TABLE vec_catalogos_configurables.evidencia_terminal (
    consumidor text NOT NULL,
    uso_ref text NOT NULL,
    reserva_recibo_ref text NOT NULL CHECK (pg_catalog.octet_length(reserva_recibo_ref) BETWEEN 3 AND 160),
    terminal_recibo_ref text NOT NULL UNIQUE CHECK (pg_catalog.octet_length(terminal_recibo_ref) BETWEEN 3 AND 160),
    estado text NOT NULL CHECK (estado IN ('confirmado','cancelado')),
    categoria_id text NOT NULL,
    catalogo_id text NOT NULL,
    version integer NOT NULL CHECK (version>0),
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    evidencia_ref text NOT NULL CHECK (pg_catalog.octet_length(evidencia_ref) BETWEEN 3 AND 160),
    evidencia_sha256 text NOT NULL CHECK (evidencia_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    PRIMARY KEY (consumidor,uso_ref),
    FOREIGN KEY (consumidor,uso_ref) REFERENCES vec_catalogos_configurables.uso(consumidor,uso_ref),
    FOREIGN KEY (catalogo_id,version,huella_sha256) REFERENCES
        vec_catalogos_configurables.publicacion(catalogo_id,version,huella_sha256),
    FOREIGN KEY (catalogo_id,version,categoria_id) REFERENCES
        vec_catalogos_configurables.entrada_publicada(catalogo_id,version,categoria_id)
);
ALTER TABLE vec_catalogos_configurables.evidencia_terminal ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_catalogos_configurables.evidencia_terminal FORCE ROW LEVEL SECURITY;
CREATE POLICY evidencia_terminal_propietario ON vec_catalogos_configurables.evidencia_terminal
    TO vec_catalogos_configurables_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_catalogos_configurables.evidencia_terminal FROM PUBLIC;
REVOKE ALL ON TYPE vec_catalogos_configurables.evidencia_terminal FROM PUBLIC;
CREATE TRIGGER evidencia_terminal_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
    ON vec_catalogos_configurables.evidencia_terminal
    FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();

-- consultar_uso 000002 es STABLE: su snapshot no garantiza ver una escritura
-- anterior dentro de la misma llamada a la fachada. Esta consulta es VOLATILE.
CREATE FUNCTION vec_catalogos_configurables.obtener_uso_publicacion(
    p_consumidor text,p_uso_ref text,p_reserva_recibo_ref text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE
    r record;
BEGIN
    IF p_consumidor IS NULL OR p_consumidor !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_uso_ref IS NULL OR pg_catalog.octet_length(p_uso_ref) NOT BETWEEN 3 AND 160
       OR p_reserva_recibo_ref IS NULL OR pg_catalog.octet_length(p_reserva_recibo_ref) NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'consulta de uso para efecto invalida' USING ERRCODE='22023';
    END IF;
    SELECT u.*,p.documento_canonico,p.publicada_en
      INTO r
      FROM vec_catalogos_configurables.uso AS u
      JOIN vec_catalogos_configurables.publicacion AS p
        ON p.catalogo_id=u.catalogo_id AND p.version=u.version AND p.huella_sha256=u.huella_sha256
     WHERE u.consumidor=p_consumidor AND u.uso_ref=p_uso_ref
       AND u.reserva_recibo_ref=p_reserva_recibo_ref;
    IF NOT FOUND THEN
        RETURN pg_catalog.jsonb_build_object('encontrado',false,'datos',NULL);
    END IF;
    RETURN pg_catalog.jsonb_build_object('encontrado',true,'datos',pg_catalog.jsonb_build_object(
        'uso',pg_catalog.jsonb_build_object(
            'consumidor',r.consumidor,'uso_ref',r.uso_ref,'categoria_id',r.categoria_id,
            'catalogo_id',r.catalogo_id,'version',r.version,'huella_sha256',r.huella_sha256,
            'estado',r.estado,'revision',r.revision,'reserva_recibo_ref',r.reserva_recibo_ref,
            'terminal_recibo_ref',r.terminal_recibo_ref,
            'reservado_en',pg_catalog.to_char(r.reservado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'terminal_en',CASE WHEN r.terminal_en IS NULL THEN NULL ELSE
                pg_catalog.to_char(r.terminal_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END),
        'publicacion',pg_catalog.jsonb_build_object(
            'catalogo_id',r.catalogo_id,'version',r.version,'huella_sha256',r.huella_sha256,
            'documento_canonico',r.documento_canonico,
            'publicada_en',pg_catalog.to_char(r.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))));
END $f$;

-- Encapsula el terminal del core y fija su evidencia opaca bajo el mismo
-- bloqueo de uso. Un terminal anterior sin vinculo no se completa a posteriori.
CREATE FUNCTION vec_catalogos_configurables.terminar_uso_con_evidencia(
    p_consumidor text,p_uso_ref text,p_reserva_recibo_ref text,p_estado text,
    p_actor_ref text,p_decision_ref text,p_terminal_recibo_ref text,p_motivo_ref text,
    p_evidencia_ref text,p_evidencia_sha256 text
) RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE
    previo vec_catalogos_configurables.uso%ROWTYPE;
    actual vec_catalogos_configurables.uso%ROWTYPE;
    vinculo vec_catalogos_configurables.evidencia_terminal%ROWTYPE;
    recibido text;
BEGIN
    IF p_estado IS NULL OR p_estado NOT IN ('confirmado','cancelado')
       OR p_evidencia_ref IS NULL OR pg_catalog.octet_length(p_evidencia_ref) NOT BETWEEN 3 AND 160
       OR p_evidencia_sha256 IS NULL OR p_evidencia_sha256 !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'evidencia terminal invalida' USING ERRCODE='22023';
    END IF;
    SELECT * INTO previo FROM vec_catalogos_configurables.uso
     WHERE consumidor=p_consumidor AND uso_ref=p_uso_ref;
    IF previo.estado IN ('confirmado','cancelado')
       AND NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.evidencia_terminal
                        WHERE consumidor=p_consumidor AND uso_ref=p_uso_ref) THEN
        RAISE EXCEPTION 'terminal anterior sin vinculo de evidencia' USING ERRCODE='55000';
    END IF;
    recibido := vec_catalogos_configurables.terminar_uso(
        p_consumidor,p_uso_ref,p_reserva_recibo_ref,p_estado,p_actor_ref,
        p_decision_ref,p_terminal_recibo_ref,p_motivo_ref);
    SELECT * INTO actual FROM vec_catalogos_configurables.uso
     WHERE consumidor=p_consumidor AND uso_ref=p_uso_ref;
    IF NOT FOUND OR recibido IS DISTINCT FROM p_terminal_recibo_ref
       OR actual.estado IS DISTINCT FROM p_estado
       OR actual.terminal_recibo_ref IS DISTINCT FROM p_terminal_recibo_ref
       OR actual.reserva_recibo_ref IS DISTINCT FROM p_reserva_recibo_ref THEN
        RAISE EXCEPTION 'terminal de uso incoherente' USING ERRCODE='55000';
    END IF;
    SELECT * INTO vinculo FROM vec_catalogos_configurables.evidencia_terminal
     WHERE consumidor=p_consumidor AND uso_ref=p_uso_ref FOR UPDATE;
    IF FOUND THEN
        IF vinculo.reserva_recibo_ref IS DISTINCT FROM actual.reserva_recibo_ref
           OR vinculo.terminal_recibo_ref IS DISTINCT FROM actual.terminal_recibo_ref
           OR vinculo.estado IS DISTINCT FROM actual.estado
           OR vinculo.categoria_id IS DISTINCT FROM actual.categoria_id
           OR vinculo.catalogo_id IS DISTINCT FROM actual.catalogo_id
           OR vinculo.version IS DISTINCT FROM actual.version
           OR vinculo.huella_sha256 IS DISTINCT FROM actual.huella_sha256
           OR vinculo.evidencia_ref IS DISTINCT FROM p_evidencia_ref
           OR vinculo.evidencia_sha256 IS DISTINCT FROM p_evidencia_sha256 THEN
            RAISE EXCEPTION 'evidencia de terminal en conflicto' USING ERRCODE='23505';
        END IF;
    ELSE
        IF previo.estado IS DISTINCT FROM 'reservado' THEN
            RAISE EXCEPTION 'terminal anterior sin vinculo de evidencia' USING ERRCODE='55000';
        END IF;
        INSERT INTO vec_catalogos_configurables.evidencia_terminal
            (consumidor,uso_ref,reserva_recibo_ref,terminal_recibo_ref,estado,
             categoria_id,catalogo_id,version,huella_sha256,evidencia_ref,evidencia_sha256)
        VALUES (actual.consumidor,actual.uso_ref,actual.reserva_recibo_ref,actual.terminal_recibo_ref,
            actual.estado,actual.categoria_id,actual.catalogo_id,actual.version,actual.huella_sha256,
            p_evidencia_ref,p_evidencia_sha256);
    END IF;
    RETURN recibido;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.obtener_uso_publicacion(text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.terminar_uso_con_evidencia(
    text,text,text,text,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)
    TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.terminar_uso_con_evidencia(
    text,text,text,text,text,text,text,text,text,text)
    TO vec_autorizacion_atestada_v3_propietario;
DO $acl$
DECLARE f oid;
BEGIN
    FOREACH f IN ARRAY ARRAY[
      'vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)'::regprocedure,
      'vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)'::regprocedure
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
            AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef
            AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
               CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
              WHERE p.oid=f AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable
                OR a.grantee NOT IN (p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
            RAISE EXCEPTION 'catalogos configurables 000003: ACL incompatible' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
COMMIT;
