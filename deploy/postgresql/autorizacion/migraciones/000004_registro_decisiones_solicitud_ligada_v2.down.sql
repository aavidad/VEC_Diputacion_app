-- La evidencia V2 no se elimina automaticamente. La retirada solo es valida
-- mientras el registro siga vacio; cualquier fila exige un procedimiento de
-- archivo/retencion aprobado y una migracion distinta.
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
    IF to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.materializar_documento_comun_decision_v2(jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para retirar registro V2';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    IN ACCESS EXCLUSIVE MODE;

DO $conservacion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada rechazada: existen decisiones V2 durables';
    END IF;
END
$conservacion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        bytea, bytea
    ) FROM vec_autorizacion_registro;
DROP FUNCTION
    vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        bytea, bytea
    );
DROP TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2;
DROP FUNCTION
    vec_autorizacion.materializar_documento_comun_decision_v2(jsonb);
DROP FUNCTION vec_autorizacion.lista_decision_v2_canonica_valida(jsonb);
DROP FUNCTION vec_autorizacion.manifiesto_decision_v2_canonico_valido(jsonb);

COMMIT;
