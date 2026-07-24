-- Regresion del reloj previo a espera. Una conexion conserva el lock del
-- puntero de sesion; otra inicia la lectura y se acredita en pg_stat_activity
-- que espera ese lock. Se libera solo cuando la decision alcanza exactamente
-- el limite superior half-open. No se sincroniza mediante sleeps arbitrarios.
\getenv clave_ejecutor CLAVE_EJECUTOR

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL ROLE vec_autorizacion_propietario;

DO $preparar_decision$
DECLARE
    contexto bytea := convert_to(
        '{"ambitos":{"organizacion_ref":"org_0123456789abcdef"},"atributos":{}}',
        'UTF8'
    );
BEGIN
    PERFORM public.crear_decision_borrador_runtime_prueba(
        'decision-runtime-listar-concurrente',
        'bolsa.convocatoria.borrador.listar',
        'coleccion_versiones_convocatoria_gobernada',
        'borradores:org_0123456789abcdef',
        'consulta_interna_convocatorias',
        '["version_convocatoria"]'::jsonb, contexto, interval '8 seconds'
    );
    PERFORM public.convertir_decision_borrador_runtime_v2(
        'decision-runtime-listar-concurrente'
    );
END
$preparar_decision$;

RESET ROLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $atestacion$
DECLARE
    fila record;
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    evidencia bytea := convert_to('{}', 'UTF8');
    sobre bytea;
BEGIN
    SELECT * INTO STRICT fila
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-concurrente';
    sobre := convert_to('cose-prueba:' || fila.decision_ref, 'UTF8');
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        fila.decision_ref, fila.atestacion_ref, 1, 'activa',
        fila.huella_decision_sha256, evidencia,
        fila.huella_atestacion_sha256, sobre,
        encode(sha256(sobre), 'hex'), 'clave-pdp-runtime',
        'confianza-runtime', ahora, ahora - interval '1 minute',
        ahora + interval '2 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (fila.decision_ref, fila.atestacion_ref, 1, 'activa', ahora);
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador VALUES (
        fila.decision_ref, fila.atestacion_ref, 1, 'activa',
        fila.huella_decision_sha256, fila.huella_atestacion_sha256,
        'verificador:' || fila.decision_ref, ahora, ahora
    );
END
$atestacion$;
RESET ROLE;

CREATE FUNCTION public.probar_lectura_borrador_concurrente_runtime()
RETURNS boolean
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    fila record;
    contexto bytea := convert_to(
        '{"ambitos":{"organizacion_ref":"org_0123456789abcdef"},"atributos":{}}',
        'UTF8'
    );
    lectura jsonb;
BEGIN
    SELECT * INTO STRICT fila
      FROM public.fixture_decision_borrador_runtime
     WHERE decision_ref = 'decision-runtime-listar-concurrente';
    lectura := jsonb_build_object(
        'decision_ref', fila.decision_ref,
        'huella_decision_sha256', fila.huella_decision_sha256,
        'atestacion_ref', fila.atestacion_ref, 'atestacion_version', 1,
        'estado_atestacion', 'activa',
        'huella_atestacion_sha256', fila.huella_atestacion_sha256,
        'accion', 'bolsa.convocatoria.borrador.listar',
        'recurso_ref', 'borradores:org_0123456789abcdef',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', ''
    );
    BEGIN
        PERFORM vec_bolsa_convocatorias.listar_borradores_v1(
            jsonb_build_object(
                'limite',10,'cursor','','texto','','categoria',''
            ), lectura, fila.prueba, fila.decision_canonica, contexto
        );
        RETURN false;
    EXCEPTION WHEN insufficient_privilege THEN
        RETURN true;
    END;
END
$funcion$;
GRANT EXECUTE ON FUNCTION
    public.probar_lectura_borrador_concurrente_runtime()
    TO vec_convocatorias_ejecutor_prueba;
COMMIT;

SELECT dblink_connect(
    'bloqueador_lectura',
    'dbname=' || current_database() ||
    ' user=postgres application_name=vec_bloqueador_lectura_temporal'
);
SELECT dblink_exec('bloqueador_lectura', 'BEGIN');
SELECT dblink_exec(
    'bloqueador_lectura', 'SET ROLE vec_autorizacion_propietario'
);
SELECT *
  FROM dblink(
      'bloqueador_lectura',
      $consulta$
      SELECT sesion_ref
        FROM vec_autorizacion.control_sesion_actual_v1
       WHERE sesion_ref = 'ses_convocatorias_runtime_prueba_000001'
       FOR UPDATE
      $consulta$
  ) AS bloqueo(sesion_ref text);

SELECT dblink_connect(
    'lector_temporal',
    'host=127.0.0.1 dbname=' || current_database() ||
    ' user=vec_convocatorias_ejecutor_prueba password=' || :'clave_ejecutor' ||
    ' application_name=vec_lector_temporal' ||
    ' options=-cdefault_transaction_isolation=serializable'
);
SELECT dblink_send_query(
    'lector_temporal',
    'SELECT public.probar_lectura_borrador_concurrente_runtime()'
);

DO $esperar_lock_real$
DECLARE
    limite timestamptz := clock_timestamp() + interval '5 seconds';
BEGIN
    LOOP
        EXIT WHEN EXISTS (
            SELECT 1
              FROM pg_catalog.pg_stat_activity
             WHERE application_name = 'vec_lector_temporal'
               AND state = 'active' AND wait_event_type = 'Lock'
        );
        IF clock_timestamp() >= limite THEN
            RAISE EXCEPTION
                'la lectura concurrente no alcanzo el lock de sesion';
        END IF;
    END LOOP;
END
$esperar_lock_real$;

-- Espera contra el limite autoritativo almacenado, no durante un intervalo
-- supuesto. Al salir, ahora == valida_hasta o lo ha sobrepasado: [desde,hasta)
-- exige denegar.
DO $alcanzar_limite_half_open$
DECLARE
    limite timestamptz;
BEGIN
    SELECT valida_hasta INTO STRICT limite
      FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
     WHERE decision_ref = 'decision-runtime-listar-concurrente';
    WHILE clock_timestamp() < limite LOOP
        PERFORM 1;
    END LOOP;
END
$alcanzar_limite_half_open$;

SELECT dblink_exec('bloqueador_lectura', 'COMMIT');
CREATE TEMP TABLE resultado_lectura_temporal AS
SELECT * FROM dblink_get_result('lector_temporal') AS r(denegada boolean);

DO $verificar_denegacion$
BEGIN
    IF (SELECT denegada FROM resultado_lectura_temporal) IS NOT TRUE THEN
        RAISE EXCEPTION
            'una decision en el limite half-open fue aceptada tras la espera';
    END IF;
END
$verificar_denegacion$;

SELECT dblink_disconnect('lector_temporal');
SELECT dblink_disconnect('bloqueador_lectura');
