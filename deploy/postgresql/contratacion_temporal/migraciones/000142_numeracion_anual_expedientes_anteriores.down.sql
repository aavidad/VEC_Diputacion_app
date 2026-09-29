\set ON_ERROR_STOP on
-- Retira CT-000142 solo sin historia: si algún expediente ya recibió su número
-- anual, retirarla borraría esa actuación y sus eventos quedarían sin rastro
-- en la tabla que los explica, y se rechaza. Sin asignaciones devuelve las
-- consultas a su cuerpo anterior (huellas fijadas) y retira los objetos nuevos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000142', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.numeracion_anual_asignada'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000142 no instalada';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_contratacion_temporal.numeracion_anual_asignada
    IN ACCESS EXCLUSIVE MODE;

DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.numeracion_anual_asignada) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000142: reversión denegada, hay números anuales asignados';
    END IF;
END
$historia$;

DO $consultas$
DECLARE
    v_cambio record;
    v_antes record;
    v_despues record;
    v_definicion text;
    v_apariciones integer;
BEGIN
    FOR v_cambio IN
        SELECT *
          FROM (VALUES
            ('vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1)',
             'b5a878d579c79b5550cd2d42108c9a25d4c8db44efc1c536398a14418788951f',
             '45a6f552b1544a74f717c6dbd424a348305f037b51efc19db075a0e226e879bb',
             $a1$               vec_contratacion_temporal.numero_visible_vigente_v1(
                   publicada.expediente_ref, publicada.numero_visible
               ) AS numero_visible,
               publicada.version,$a1$,
             $n1$               publicada.numero_visible,
               publicada.version,$n1$),
            ('vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)',
             'cf50cae432619bdbfa45a249639559374f9c2cc88771e4c1908e897a808464b0',
             'ead046ef1e96d3ca2fe39171bbc94496ca10dda894402866de87ca19d12ef72f',
             $a2$               vec_contratacion_temporal.numero_visible_vigente_v1(
                   publicada.expediente_ref, publicada.numero_visible
               ) AS numero_visible, publicada.fase_clave,$a2$,
             $n2$               publicada.numero_visible, publicada.fase_clave,$n2$),
            ('vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)',
             '6ee713225c5fbaa9052f451d6e0212cd44752aedb6c824269ceca52e80e3f77d',
             '0826469968ccba7c35bd6d3e7ca7e493aa871b1b753a7af929bb9c33f0893a2f',
             $a3$    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        vec_contratacion_temporal.numero_visible_vigente_v1(
            v_fila.expediente_ref, v_fila.numero_visible
        ),$a3$,
             $n3$    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        v_fila.numero_visible,$n3$),
            ('vec_contratacion_temporal.expedientes_centro_ct124(jsonb,text)',
             'f698ff4755a5b0b2c6f6070e6dc28cf7174dad2d061ce807942c7d0e85d8954b',
             '8e42e37d9f3a3aa7637d6177b5206dd4d1f3bb8d54b626200f5606e18c7519cb',
             $a4$c.expediente_ref, vec_contratacion_temporal.numero_visible_vigente_v1(c.expediente_ref, c.numero_visible) AS numero_visible, ac.version$a4$,
             $n4$c.expediente_ref, c.numero_visible, ac.version$n4$)
          ) AS cambio(firma, huella_antes, huella_despues, anterior, nuevo)
    LOOP
        SELECT p.oid, p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor
          INTO v_antes
          FROM pg_catalog.pg_proc p
         WHERE p.oid = pg_catalog.to_regprocedure(v_cambio.firma)
           AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
        IF NOT FOUND
           OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                  v_antes.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_cambio.huella_antes THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: consulta incompatible ' || v_cambio.firma;
        END IF;
        v_apariciones := (pg_catalog.length(v_antes.definicion)
            - pg_catalog.length(pg_catalog.replace(
                v_antes.definicion, v_cambio.anterior, '')))
            / pg_catalog.length(v_cambio.anterior);
        IF v_apariciones <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: fragmento ambiguo en ' || v_cambio.firma;
        END IF;
        v_definicion := pg_catalog.replace(
            v_antes.definicion, v_cambio.anterior, v_cambio.nuevo);
        EXECUTE v_definicion;
        SELECT p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor
          INTO STRICT v_despues
          FROM pg_catalog.pg_proc p
         WHERE p.oid = v_antes.oid;
        IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               v_despues.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_cambio.huella_despues
           OR v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS DISTINCT FROM v_antes.definidor THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: definición o permisos alterados en ' || v_cambio.firma;
        END IF;
    END LOOP;
END
$consultas$;

DROP FUNCTION vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1();
DROP FUNCTION vec_contratacion_temporal.numero_visible_vigente_v1(text, text);
DROP TABLE vec_contratacion_temporal.numeracion_anual_asignada;
COMMIT;
