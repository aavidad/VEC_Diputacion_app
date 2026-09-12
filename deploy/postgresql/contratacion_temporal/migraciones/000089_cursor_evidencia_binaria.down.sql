\set ON_ERROR_STOP on
-- La retirada no reinterpreta ni revoca cursores emitidos con CT89.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000089', 0));
LOCK TABLE vec_contratacion_temporal.cursor_cuadro_rrhh IN SHARE ROW EXCLUSIVE MODE;

DO $ct89$
DECLARE
    cambio record;
    antes record;
    despues record;
    definicion text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cursor_cuadro_rrhh) THEN
        RAISE EXCEPTION 'CT89: se conservan cursores emitidos' USING ERRCODE = '55000';
    END IF;
    FOR cambio IN
        SELECT * FROM (VALUES
            (
                'vec_contratacion_temporal.preparar_salida_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.materializacion_cuadro_rrhh_v1)',
                '5ca9172ea709e3cac9fef939980f8be618b663940c0c19641c7c2d22a1339d89',
                $despues_preparar$true, v_token, pg_catalog.sha256(pg_catalog.decode(
            pg_catalog.rpad(pg_catalog.translate(v_token, '-_', '+/'), 44, '='),
            'base64'
        )),$despues_preparar$,
                $antes_preparar$true, v_token, pg_catalog.decode(v_token_huella, 'hex'),$antes_preparar$,
                '9dad01ea5f09085dcb808a29d9e3df370b953603f6ae73851a85c07044503725'
            ),
            (
                'vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2)',
                '0e7d8edd3317eb74092c5a762f80b4054313346d92b61a87f90afbda5d178299',
                $despues_cierre$p_cierre.cursor_huella_sha256 IS DISTINCT FROM
          (CASE WHEN p_salida.hay_mas
                THEN pg_catalog.encode(p_salida.cursor_huella, 'hex') ELSE '' END)$despues_cierre$,
                $antes_cierre$p_cierre.cursor_huella_sha256 IS DISTINCT FROM
          (CASE WHEN p_salida.hay_mas
                THEN p_salida.token_nuevo_huella_sha256 ELSE '' END)$antes_cierre$,
                '3f678b3e18e94596aa501ee5b28a6000ba18ee54e60f724408b9f32bcefee54a'
            ),
            (
                'vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2)',
                '3f678b3e18e94596aa501ee5b28a6000ba18ee54e60f724408b9f32bcefee54a',
                $despues_efectos$p_salida.cursor_huella IS DISTINCT FROM pg_catalog.sha256(
               pg_catalog.decode(pg_catalog.rpad(pg_catalog.translate(
                   p_salida.cursor_siguiente, '-_', '+/'
               ), 44, '='), 'base64')
           )
           OR pg_catalog.rtrim(pg_catalog.translate(pg_catalog.encode(
               pg_catalog.decode(pg_catalog.rpad(pg_catalog.translate(
                   p_salida.cursor_siguiente, '-_', '+/'
               ), 44, '='), 'base64'), 'base64'
           ), '+/', '-_'), E'=\n') IS DISTINCT FROM p_salida.cursor_siguiente$despues_efectos$,
                $antes_efectos$p_salida.cursor_huella IS DISTINCT FROM pg_catalog.decode(
               p_salida.token_nuevo_huella_sha256, 'hex'
           )$antes_efectos$,
                '7f949e4d036b6ecfff72130b80bd29bbc823f8619aaef1f3227fe28aef349660'
            )
        ) AS cambios(firma, huella_antes, fragmento_antes, fragmento_despues, huella_despues)
    LOOP
        SELECT p.oid, p.prosrc AS cuerpo, pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario, p.proconfig AS configuracion,
               p.prosecdef AS definidor
          INTO antes
          FROM pg_proc p
         WHERE p.oid = to_regprocedure(cambio.firma)
           AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole
           AND p.prosecdef
           AND p.prolang = (SELECT oid FROM pg_language WHERE lanname = 'plpgsql');
        IF NOT FOUND
           OR encode(sha256(convert_to(antes.cuerpo, 'UTF8')), 'hex') IS DISTINCT FROM cambio.huella_antes
           OR (length(antes.cuerpo) - length(replace(antes.cuerpo, cambio.fragmento_antes, '')))
                / length(cambio.fragmento_antes) <> 1
           OR strpos(antes.cuerpo, cambio.fragmento_despues) <> 0
           OR (length(antes.definicion) - length(replace(antes.definicion, antes.cuerpo, '')))
                / length(antes.cuerpo) <> 1 THEN
            RAISE EXCEPTION 'CT89: postimagen de cursor incompatible' USING ERRCODE = '55000';
        END IF;
        definicion := replace(antes.definicion, antes.cuerpo,
            replace(antes.cuerpo, cambio.fragmento_antes, cambio.fragmento_despues));
        EXECUTE definicion;
        SELECT p.prosrc AS cuerpo, pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario, p.proconfig AS configuracion,
               p.prosecdef AS definidor
          INTO STRICT despues
          FROM pg_proc p WHERE p.oid = antes.oid;
        IF encode(sha256(convert_to(despues.cuerpo, 'UTF8')), 'hex') IS DISTINCT FROM cambio.huella_despues
           OR despues.definicion IS DISTINCT FROM definicion
           OR despues.acl IS DISTINCT FROM antes.acl
           OR despues.propietario IS DISTINCT FROM antes.propietario
           OR despues.configuracion IS DISTINCT FROM antes.configuracion
           OR despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT89: retirada de cursor incompatible' USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$ct89$;
COMMIT;
