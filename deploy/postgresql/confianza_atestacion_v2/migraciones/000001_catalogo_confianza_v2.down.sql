-- Retirada destructiva denegada por defecto. El operador debe aportar solo en
-- esta sesion:
--   -c vec.confirmar_destruccion_confianza_atestacion_v2=DESTRUIR_CONFIANZA_V2_IRREVERSIBLE
-- La confirmacion no sustituye copia, autorizacion ni doble control.
BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:migracion_down:v1', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_confianza_atestacion_v2:gobierno:v1', 0
    )
);

DO $confirmacion$
BEGIN
    IF current_setting(
           'vec.confirmar_destruccion_confianza_atestacion_v2',
           true
       ) IS DISTINCT FROM 'DESTRUIR_CONFIANZA_V2_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de confianza V2 rechazado: falta confirmacion explicita',
            HINT = 'verifique copia y autorizacion antes del opt-in de sesion';
    END IF;
END
$confirmacion$;

-- Bloquea toda tabla presente, incluidas las que incorpore una migracion
-- futura. Un objeto desconocido no se elimina: el DROP SCHEMA RESTRICT final
-- aborta la transaccion completa.
DO $bloquear_tablas$
DECLARE
    tabla record;
BEGIN
    FOR tabla IN
        SELECT espacio.nspname, clase.relname
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_confianza_atestacion_v2'
           AND clase.relkind IN ('r', 'p')
         ORDER BY clase.oid
    LOOP
        EXECUTE format(
            'LOCK TABLE %I.%I IN ACCESS EXCLUSIVE MODE',
            tabla.nspname,
            tabla.relname
        );
    END LOOP;
END
$bloquear_tablas$;

DROP TABLE
    vec_confianza_atestacion_v2.puntero_configuracion_actual,
    vec_confianza_atestacion_v2.puntero_raiz_actual,
    vec_confianza_atestacion_v2.revocacion_configuracion,
    vec_confianza_atestacion_v2.revocacion_raiz,
    vec_confianza_atestacion_v2.configuracion_raiz,
    vec_confianza_atestacion_v2.configuracion_confianza_version,
    vec_confianza_atestacion_v2.raiz_confianza_version,
    vec_confianza_atestacion_v2.acto_gobierno
    RESTRICT;

DROP FUNCTION vec_confianza_atestacion_v2.obtener_confianza_actual()
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.validar_puntero_configuracion()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.validar_puntero_raiz()
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.calcular_huella_configuracion(text)
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.validar_revocacion_raiz()
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.validar_miembro_configuracion()
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.validar_revocacion_configuracion()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.validar_raiz_monotona()
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.validar_configuracion_monotona()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.validar_acto_monotono()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.rechazar_truncado()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.proteger_historia_fila()
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.encuadrar_huella(text)
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.instante_rfc3339nano(timestamptz)
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.instante_go_valido(timestamptz)
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.clave_spki_ed25519_valida(bytea)
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.audiencia_despliegue_valida(text)
    RESTRICT;
DROP FUNCTION vec_confianza_atestacion_v2.huella_sha256_valida(text)
    RESTRICT;
DROP FUNCTION
    vec_confianza_atestacion_v2.texto_tecnico_valido(text, integer)
    RESTRICT;

DROP SCHEMA vec_confianza_atestacion_v2 RESTRICT;
COMMIT;
