\set ON_ERROR_STOP on
-- Reversión protegida de CT187 para un ensayo desechable. No ejecutar sobre
-- una base que ya haya servido accesos con esta versión.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000187', 0)
);
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass('vec_contratacion_temporal.peticion_centro_ratificada_orden_ct187_idx') IS NULL THEN
        RAISE EXCEPTION 'CT187: PARO clave=indice_ct187 esperado=presente actual=ausente'
            USING ERRCODE = '55000';
    END IF;
END
$prevalidacion$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $consultas$
DECLARE
    v_cambio record;
    v_antes record;
    v_despues record;
    v_fragmento record;
    v_definicion text;
    v_observada text;
    v_apariciones integer;
    v_meta_antes jsonb;
    v_meta_despues jsonb;
    v_clave text;
    v_llamada text := E'vec_contratacion_temporal.numero_visible_vigente_v1(\n                   publicada.expediente_ref, publicada.numero_visible\n               )';
    v_join_antes text := E'FROM ultimas ultima\n         WHERE';
    v_join_despues text := E'FROM ultimas ultima\n          LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion\n            ON numeracion.expediente_ref = ultima.expediente_ref\n         WHERE';
    v_urgencia_antes text := E'EXISTS (\n               SELECT 1\n                 FROM vec_contratacion_temporal.urgencia_expediente_analisis u\n                WHERE u.expediente_ref = filtrada.expediente_ref\n                  AND u.version <= filtrada.version\n           )';
    v_entrada_antes text := E'AND entrada.version = filtrada.version\n     WHERE';
    v_entrada_despues text := E'AND entrada.version = filtrada.version\n      LEFT JOIN (\n          SELECT expediente_ref, pg_catalog.min(version) AS primera_version\n            FROM vec_contratacion_temporal.urgencia_expediente_analisis\n           GROUP BY expediente_ref\n      ) urgencia ON urgencia.expediente_ref = filtrada.expediente_ref\n     WHERE';
BEGIN
    FOR v_cambio IN
        SELECT * FROM (VALUES
            ('vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1)',
             '174ee9497abf84536e59d96b8c1331dea84eeb9efde1d3e5777ca314dc975d76',
             'b5a878d579c79b5550cd2d42108c9a25d4c8db44efc1c536398a14418788951f', 2),
            ('vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)',
             '0b293192c96a7f193fac4160f3755d9809596226d16d4636be64b73c011201e3',
             'cf50cae432619bdbfa45a249639559374f9c2cc88771e4c1908e897a808464b0', 1),
            ('vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)',
             'b820203e211aa615dff50d6c49a26dce0e9f43daa2ec0854a9d3036bee401cf7',
             'ed8375f116beda4874848dda79b99d330d64b421284e2850956b079c4f1b0135', 1)
        ) AS cambio(firma, huella_cuerpo, huella_despues, apariciones_numero)
    LOOP
        SELECT p.oid, p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario, p.proconfig AS configuracion,
               p.prosecdef AS definidor, p.provolatile AS volatilidad,
               p.proparallel AS paralelismo, p.prorettype AS retorno, p.proargtypes AS argumentos
          INTO v_antes FROM pg_catalog.pg_proc p
         WHERE p.oid = pg_catalog.to_regprocedure(v_cambio.firma);
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT187: PARO firma=% clave=funcion esperado=presente actual=ausente',
                v_cambio.firma USING ERRCODE = '55000';
        END IF;
        IF v_antes.propietario <> 'vec_contratacion_temporal_propietario'::pg_catalog.regrole THEN
            RAISE EXCEPTION 'CT187: PARO firma=% clave=propietario esperado=% actual=%',
                v_cambio.firma, 'vec_contratacion_temporal_propietario',
                v_antes.propietario::pg_catalog.regrole USING ERRCODE = '55000';
        END IF;
        v_observada := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_antes.cuerpo, 'UTF8')), 'hex');
        IF v_observada <> v_cambio.huella_cuerpo THEN
            RAISE EXCEPTION 'CT187: PARO firma=% clave=huella_cuerpo esperado=% actual=%',
                v_cambio.firma, v_cambio.huella_cuerpo, v_observada USING ERRCODE = '55000';
        END IF;
        FOR v_fragmento IN
            SELECT * FROM (VALUES
                ('union_ultimas', v_join_despues, 1),
                ('numero_visible_filtrado',
                 'COALESCE(numeracion.numero_visible, ultima.numero_visible)',
                 v_cambio.apariciones_numero),
                ('numero_visible_publicado', 'publicada.numero_visible', 1)
            ) AS fragmento(clave, texto, esperadas)
        LOOP
            v_apariciones := (pg_catalog.length(v_antes.definicion) -
                pg_catalog.length(pg_catalog.replace(v_antes.definicion,
                    v_fragmento.texto, ''))) / pg_catalog.length(v_fragmento.texto);
            IF v_apariciones <> v_fragmento.esperadas THEN
                RAISE EXCEPTION 'CT187: PARO firma=% clave=% esperado=% actual=%',
                    v_cambio.firma, v_fragmento.clave, v_fragmento.esperadas,
                    v_apariciones USING ERRCODE = '55000';
            END IF;
        END LOOP;
        v_definicion := pg_catalog.replace(v_antes.definicion,
            'COALESCE(numeracion.numero_visible, ultima.numero_visible)',
            'ultima.numero_visible');
        v_definicion := pg_catalog.replace(v_definicion, v_join_despues, v_join_antes);
        IF v_cambio.firma LIKE '%contar_resumen_cuadro_rrhh_v1%' THEN
            FOR v_fragmento IN
                SELECT * FROM (VALUES
                    ('urgencia_agrupada',
                     'COALESCE(urgencia.primera_version <= filtrada.version, false)'),
                    ('union_entrada', v_entrada_despues)
                ) AS fragmento(clave, texto)
            LOOP
                v_apariciones := (pg_catalog.length(v_definicion) -
                    pg_catalog.length(pg_catalog.replace(v_definicion,
                        v_fragmento.texto, ''))) / pg_catalog.length(v_fragmento.texto);
                IF v_apariciones <> 1 THEN
                    RAISE EXCEPTION 'CT187: PARO firma=% clave=% esperado=1 actual=%',
                        v_cambio.firma, v_fragmento.clave, v_apariciones
                        USING ERRCODE = '55000';
                END IF;
            END LOOP;
            v_definicion := pg_catalog.replace(v_definicion,
                'COALESCE(urgencia.primera_version <= filtrada.version, false)',
                v_urgencia_antes);
            v_definicion := pg_catalog.replace(v_definicion,
                v_entrada_despues, v_entrada_antes);
        END IF;
        v_definicion := pg_catalog.replace(v_definicion, 'publicada.numero_visible',
            v_llamada);
        EXECUTE v_definicion;
        SELECT p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario, p.proconfig AS configuracion,
               p.prosecdef AS definidor, p.provolatile AS volatilidad,
               p.proparallel AS paralelismo, p.prorettype AS retorno, p.proargtypes AS argumentos
          INTO STRICT v_despues FROM pg_catalog.pg_proc p WHERE p.oid = v_antes.oid;
        v_observada := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(v_despues.cuerpo, 'UTF8')), 'hex');
        IF v_observada <> v_cambio.huella_despues THEN
            RAISE EXCEPTION 'CT187: PARO firma=% clave=huella_cuerpo_despues esperado=% actual=%',
                v_cambio.firma, v_cambio.huella_despues, v_observada USING ERRCODE = '55000';
        END IF;
        IF v_despues.definicion IS DISTINCT FROM v_definicion THEN
            RAISE EXCEPTION 'CT187: PARO firma=% clave=huella_definicion esperado=% actual=%',
                v_cambio.firma,
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_definicion, 'UTF8')), 'hex'),
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_despues.definicion, 'UTF8')), 'hex')
                USING ERRCODE = '55000';
        END IF;
        v_meta_antes := pg_catalog.jsonb_build_object(
            'acl', v_antes.acl::text, 'propietario', v_antes.propietario::text,
            'configuracion', v_antes.configuracion::text, 'definidor', v_antes.definidor,
            'volatilidad', v_antes.volatilidad, 'paralelismo', v_antes.paralelismo,
            'retorno', v_antes.retorno::text, 'argumentos', v_antes.argumentos::text);
        v_meta_despues := pg_catalog.jsonb_build_object(
            'acl', v_despues.acl::text, 'propietario', v_despues.propietario::text,
            'configuracion', v_despues.configuracion::text, 'definidor', v_despues.definidor,
            'volatilidad', v_despues.volatilidad, 'paralelismo', v_despues.paralelismo,
            'retorno', v_despues.retorno::text, 'argumentos', v_despues.argumentos::text);
        FOR v_clave IN SELECT pg_catalog.jsonb_object_keys(v_meta_antes) LOOP
            IF v_meta_antes->v_clave IS DISTINCT FROM v_meta_despues->v_clave THEN
                RAISE EXCEPTION 'CT187: PARO firma=% clave=% esperado=% actual=%',
                    v_cambio.firma, v_clave,
                    v_meta_antes->>v_clave, v_meta_despues->>v_clave
                    USING ERRCODE = '55000';
            END IF;
        END LOOP;
    END LOOP;
END
$consultas$;

DROP INDEX vec_contratacion_temporal.peticion_centro_ratificada_orden_ct187_idx;
COMMIT;
