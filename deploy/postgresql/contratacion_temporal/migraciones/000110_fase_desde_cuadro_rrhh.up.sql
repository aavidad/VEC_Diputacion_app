\set ON_ERROR_STOP on
-- CT-000110: fecha objetiva de entrada en la fase actual para la bandeja RRHH.
-- Solo guarda el hecho (instante en que el expediente entró en la fase); el
-- plazo que corresponda lo resuelve la aplicación con el catálogo de reglas.
-- La publicación CT37 es historia inmutable: el dato vive en una tabla de solo
-- adición enlazada por (expediente_ref, version) y el canon V1 del cuadro, su
-- recibo y las fachadas v1/v2 quedan intactos. La fachada v3 añade dos
-- columnas alineadas con el orden del contenido canónico devuelto por v2.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000110', 0)
);

-- SHARE impide publicar versiones nuevas hasta que el disparador y el relleno
-- confirmen juntos; los escritores que esperan se registran después con él.
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE MODE;
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh IN SHARE MODE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.fase_entrada_publicacion_rrhh'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL
    -- El relleno exige versiones contiguas desde 1 en cada expediente.
    OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.publicacion_version_rrhh p
         GROUP BY p.expediente_ref
        HAVING pg_catalog.min(p.version) <> 1
            OR pg_catalog.max(p.version) <> pg_catalog.count(*)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para fase de entrada del cuadro RRHH';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh (
    expediente_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    fase_clave text NOT NULL,
    fase_desde timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version),
    FOREIGN KEY (expediente_ref, version)
        REFERENCES vec_contratacion_temporal.publicacion_version_rrhh (
            expediente_ref, version
        ),
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (fase_clave ~ '^[a-z][a-z0-9._-]{1,79}$'),
    CHECK (fase_desde = pg_catalog.date_trunc('microseconds', fase_desde))
);

COMMENT ON TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh IS
'Instante en que cada versión publicada entró en su fase: el actualizado_en de la primera versión del último tramo continuo con esa fase. Solo adición.';

-- Se ejecuta tras cada publicación CT37. La versión anterior ya tiene su fila
-- (relleno o disparador previo); si faltara, la escritura falla cerrada.
CREATE FUNCTION vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_desde timestamptz;
    v_anterior record;
BEGIN
    IF TG_OP <> 'INSERT'
       OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
       OR TG_TABLE_NAME <> 'publicacion_version_rrhh'
       OR current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'registro de fase de entrada RRHH no autorizado';
    END IF;
    v_desde := NEW.actualizado_en;
    IF NEW.version > 1 THEN
        SELECT entrada.fase_clave, entrada.fase_desde
          INTO STRICT v_anterior
          FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
         WHERE entrada.expediente_ref = NEW.expediente_ref
           AND entrada.version = NEW.version - 1;
        IF v_anterior.fase_clave = NEW.fase_clave THEN
            v_desde := v_anterior.fase_desde;
        END IF;
    END IF;
    INSERT INTO vec_contratacion_temporal.fase_entrada_publicacion_rrhh (
        expediente_ref, version, fase_clave, fase_desde
    ) VALUES (NEW.expediente_ref, NEW.version, NEW.fase_clave, v_desde);
    RETURN NULL;
END
$funcion$;

-- Relleno: cada cambio de fase abre un tramo; fase_desde es el actualizado_en
-- de la primera versión del tramo al que pertenece la fila.
WITH marcadas AS (
    SELECT p.expediente_ref, p.version, p.fase_clave, p.actualizado_en,
           CASE WHEN pg_catalog.lag(p.fase_clave) OVER por_expediente
                     IS DISTINCT FROM p.fase_clave
                THEN 1 ELSE 0 END AS abre_tramo
      FROM vec_contratacion_temporal.publicacion_version_rrhh p
    WINDOW por_expediente AS (
        PARTITION BY p.expediente_ref ORDER BY p.version
    )
), tramos AS (
    SELECT marcada.*,
           pg_catalog.sum(marcada.abre_tramo) OVER (
               PARTITION BY marcada.expediente_ref ORDER BY marcada.version
           ) AS tramo
      FROM marcadas marcada
)
INSERT INTO vec_contratacion_temporal.fase_entrada_publicacion_rrhh (
    expediente_ref, version, fase_clave, fase_desde
)
SELECT tramo.expediente_ref, tramo.version, tramo.fase_clave,
       pg_catalog.first_value(tramo.actualizado_en) OVER (
           PARTITION BY tramo.expediente_ref, tramo.tramo
           ORDER BY tramo.version
       )
  FROM tramos tramo
 ORDER BY tramo.expediente_ref, tramo.version;

CREATE TRIGGER publicacion_version_rrhh_fase_entrada
AFTER INSERT
ON vec_contratacion_temporal.publicacion_version_rrhh
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1();

CREATE TRIGGER fase_entrada_publicacion_rrhh_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.fase_entrada_publicacion_rrhh
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER fase_entrada_publicacion_rrhh_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.fase_entrada_publicacion_rrhh
FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.fase_entrada_publicacion_rrhh
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

-- Lee del contenido canónico V1 (CT42) solo la referencia y la versión de
-- cada resumen, en su orden. Rechaza cualquier byte fuera del formato.
CREATE FUNCTION vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(
    p_contenido bytea
)
RETURNS TABLE(orden integer, expediente_ref text, version numeric)
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_cabecera constant bytea := pg_catalog.convert_to(
        'VEC-CT-CONTENIDO-CUADRO-RRHH-V1' || pg_catalog.chr(10), 'UTF8'
    );
    v_largo constant integer := pg_catalog.octet_length(p_contenido);
    v_pos integer;
    v_total integer;
    v_indice integer;
    v_campo integer;
    v_separador integer;
    v_longitud integer;
    v_digitos text;
    v_valor bytea;
    v_ref text;
    v_version numeric;
BEGIN
    IF v_largo > 262144
       OR pg_catalog.substr(p_contenido, 1, pg_catalog.octet_length(v_cabecera))
          IS DISTINCT FROM v_cabecera THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de cuadro RRHH no válido';
    END IF;
    v_pos := pg_catalog.octet_length(v_cabecera) + 1;
    -- Marcos «longitud:valor\n»: generada_en, total y 15 por resumen.
    v_indice := 0;
    v_campo := 0;
    v_total := -1;
    LOOP
        EXIT WHEN v_total >= 0 AND v_indice = v_total;
        v_separador := pg_catalog.position(
            pg_catalog.substr(p_contenido, v_pos, 7), '\x3a'::bytea
        );
        IF v_separador < 2 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de cuadro RRHH no válido';
        END IF;
        v_digitos := pg_catalog.convert_from(
            pg_catalog.substr(p_contenido, v_pos, v_separador - 1), 'UTF8'
        );
        IF v_digitos !~ '^(0|[1-9][0-9]{0,5})$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de cuadro RRHH no válido';
        END IF;
        v_longitud := v_digitos::integer;
        IF v_pos + v_separador + v_longitud > v_largo
           OR pg_catalog.get_byte(
               p_contenido, v_pos + v_separador + v_longitud - 1
           ) <> 10 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de cuadro RRHH no válido';
        END IF;
        v_valor := pg_catalog.substr(
            p_contenido, v_pos + v_separador, v_longitud
        );
        v_pos := v_pos + v_separador + v_longitud + 1;
        IF v_total < 0 THEN
            v_campo := v_campo + 1;
            IF v_campo = 2 THEN
                v_digitos := pg_catalog.convert_from(v_valor, 'UTF8');
                IF v_digitos !~ '^(0|[1-9][0-9]{0,2})$'
                   OR v_digitos::integer > 100 THEN
                    RAISE EXCEPTION USING ERRCODE = '22023',
                        MESSAGE = 'contenido de cuadro RRHH no válido';
                END IF;
                v_total := v_digitos::integer;
                v_campo := 0;
            END IF;
            CONTINUE;
        END IF;
        v_campo := v_campo + 1;
        IF v_campo = 1 THEN
            v_ref := pg_catalog.convert_from(v_valor, 'UTF8');
        ELSIF v_campo = 4 THEN
            v_digitos := pg_catalog.convert_from(v_valor, 'UTF8');
            IF v_digitos !~ '^[1-9][0-9]{0,15}$' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de cuadro RRHH no válido';
            END IF;
            v_version := v_digitos::numeric;
        ELSIF v_campo = 15 THEN
            v_indice := v_indice + 1;
            v_campo := 0;
            orden := v_indice;
            expediente_ref := v_ref;
            version := v_version;
            RETURN NEXT;
        END IF;
    END LOOP;
    RETURN;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    p_capacidad_canonica bytea, p_decision_canonica bytea,
    p_motivo_canonico bytea, p_contexto_actor_canonico bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea, p_raiz_publica_spki bytea
)
RETURNS TABLE(
    contenido_canonico bytea, cursor_siguiente text, esquema text,
    acceso_ref text, secuencia numeric, anterior_sha256 text,
    huella_sha256 text, vinculo_identidad_huella_sha256 text,
    alcance_huella_sha256 text, registrada_en timestamptz,
    auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
    consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
    resultado_huella_sha256 text, cursor_huella_sha256 text,
    generada_en timestamptz, expediente_ref text,
    version_expediente numeric, total smallint, recibo_sello_sha256 text,
    total_filtrado numeric, en_tramitacion numeric,
    con_incidencia numeric, en_llamamiento numeric,
    fase_desde_expedientes text[], fase_desde_instantes timestamptz[]
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_v2 record;
    v_refs text[];
    v_instantes timestamptz[];
    v_filas integer;
    v_con_fase integer;
BEGIN
    SELECT * INTO STRICT v_v2
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    SELECT COALESCE(pg_catalog.array_agg(leido.expediente_ref ORDER BY leido.orden), '{}'),
           COALESCE(pg_catalog.array_agg(entrada.fase_desde ORDER BY leido.orden), '{}'),
           pg_catalog.count(*)::integer,
           pg_catalog.count(entrada.fase_desde)::integer
      INTO v_refs, v_instantes, v_filas, v_con_fase
      FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(
               v_v2.contenido_canonico
           ) leido
      LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
        ON entrada.expediente_ref = leido.expediente_ref
       AND entrada.version = leido.version;
    IF v_filas <> v_v2.total OR v_con_fase <> v_filas THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'fase de entrada de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY SELECT v_v2.contenido_canonico, v_v2.cursor_siguiente,
        v_v2.esquema, v_v2.acceso_ref, v_v2.secuencia,
        v_v2.anterior_sha256, v_v2.huella_sha256,
        v_v2.vinculo_identidad_huella_sha256,
        v_v2.alcance_huella_sha256, v_v2.registrada_en,
        v_v2.auditoria_vec_ref, v_v2.auditoria_vec_huella_sha256,
        v_v2.consumo_vec_huella_sha256, v_v2.contenido_huella_sha256,
        v_v2.resultado_huella_sha256, v_v2.cursor_huella_sha256,
        v_v2.generada_en, v_v2.expediente_ref, v_v2.version_expediente,
        v_v2.total, v_v2.recibo_sello_sha256, v_v2.total_filtrado,
        v_v2.en_tramitacion, v_v2.con_incidencia, v_v2.en_llamamiento,
        v_refs, v_instantes;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1(),
    vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMENT ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) IS 'Cuadro RRHH v2 más la fecha de entrada en la fase de cada expediente de la página, en el orden del contenido canónico.';

-- Postcondición: objetos, propietario, definidor, ACL efectiva, RLS,
-- disparadores y relleno completo, como CT111 y CT115.
DO $postcondicion$
DECLARE
    v_v3 regprocedure := 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
    v_registro regprocedure := 'vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()';
    v_lector regprocedure := 'vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)';
    v_tabla regclass := 'vec_contratacion_temporal.fase_entrada_publicacion_rrhh';
BEGIN
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid IN (v_v3, v_registro, v_lector)
                AND p.proowner <> 'vec_contratacion_temporal_propietario'::regrole)
       OR NOT (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid = v_v3)
       OR NOT (SELECT p.prosecdef FROM pg_catalog.pg_proc p WHERE p.oid = v_registro)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl, pg_catalog.acldefault('f', p.proowner))) x
                  WHERE p.oid IN (v_v3, v_registro, v_lector) AND x.grantee <> p.proowner
                    AND NOT (p.oid = v_v3 AND x.grantee = 'vec_contratacion_temporal_consultor_rrhh'::regrole
                             AND x.privilege_type = 'EXECUTE' AND NOT x.is_grantable))
       OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_consultor_rrhh', v_v3, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor', v_v3, 'EXECUTE')
       OR (SELECT c.relowner <> 'vec_contratacion_temporal_propietario'::regrole
                  OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity
             FROM pg_catalog.pg_class c WHERE c.oid = v_tabla)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c
                   CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl, pg_catalog.acldefault('r', c.relowner))) x
                  WHERE c.oid = v_tabla AND x.grantee <> c.relowner)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger t
            WHERE NOT t.tgisinternal AND t.tgname IN ('publicacion_version_rrhh_fase_entrada',
                  'fase_entrada_publicacion_rrhh_inmutable', 'fase_entrada_publicacion_rrhh_no_truncar')) <> 3
       OR (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh)
          <> (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000110: postcondición incumplida';
    END IF;
END
$postcondicion$;
COMMIT;
