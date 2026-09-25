\set ON_ERROR_STOP on
-- CT-000110 sobre una base con la cadena CT instalada hasta 000110 (p. ej. una
-- restauración desechable de la base de desarrollo). Todo termina en ROLLBACK.
-- 1) relleno idéntico a un cálculo independiente; 2) el disparador encadena
-- tramos de la misma fase y abre uno nuevo al cambiar; 3) la tabla es de solo
-- adición; 4) el lector del canon V1 devuelve refs y versiones en su orden y
-- rechaza bytes ajenos. Aplicar dos veces el UP ya lo rechaza su prevalidación.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

DO $$
BEGIN
    IF EXISTS (
        SELECT e.expediente_ref, e.version
          FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh e
          JOIN vec_contratacion_temporal.publicacion_version_rrhh p USING (expediente_ref, version)
         WHERE e.fase_clave <> p.fase_clave
            OR e.fase_desde <> (
               SELECT q.actualizado_en FROM vec_contratacion_temporal.publicacion_version_rrhh q
                WHERE q.expediente_ref = p.expediente_ref AND q.version <= p.version
                  AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh r
                                   WHERE r.expediente_ref = p.expediente_ref
                                     AND r.version BETWEEN q.version AND p.version
                                     AND r.fase_clave <> p.fase_clave)
                ORDER BY q.version LIMIT 1)
    )
       OR (SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh)
          <> (SELECT count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh) THEN
        RAISE EXCEPTION 'CT110: relleno incorrecto';
    END IF;
END $$;

-- Cuatro versiones nuevas sobre el expediente con la última versión más alta.
DO $$
DECLARE
  v_ref text; v_ultima numeric; base record; agregado jsonb; prueba bytea;
  pasos text[] := ARRAY['misma', 'fiscalizacion', 'fiscalizacion', 'asignacion_unidad'];
  v_fase text; v_instante timestamptz; v_primera_fisc timestamptz; v_fila record; i int;
BEGIN
  SELECT expediente_ref, max(version) INTO STRICT v_ref, v_ultima
    FROM vec_contratacion_temporal.publicacion_version_rrhh
   GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT 1;
  FOR i IN 1..4 LOOP
    SELECT * INTO STRICT base FROM vec_contratacion_temporal.expediente_version_integral
     WHERE expediente_ref = v_ref AND version = v_ultima + i - 1;
    v_fase := CASE WHEN pasos[i] = 'misma' THEN base.fase_clave ELSE pasos[i] END;
    v_instante := date_trunc('second', clock_timestamp()) + make_interval(days => i);
    agregado := base.agregado_json || jsonb_build_object('version', v_ultima + i, 'fase_actual', v_fase,
        'actualizado_en', to_char(v_instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'));
    prueba := decode(repeat(md5((v_ultima + i)::text || 'ct110'), 8), 'hex');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral
      (expediente_ref, version, agregado_json, agregado_json_huella_sha256,
       prueba_canonica, prueba_huella_sha256, flujo_ref, flujo_version,
       flujo_huella_sha256, fase_clave, estado, origen_version, operacion_ref, registrada_en)
    VALUES (v_ref, v_ultima + i, agregado, encode(sha256(convert_to(agregado::text, 'UTF8')), 'hex'),
       prueba, encode(sha256(prueba), 'hex'), base.flujo_ref, base.flujo_version,
       base.flujo_huella_sha256, v_fase, base.estado, base.origen_version,
       'operacion:ct110:' || (v_ultima + i), v_instante + interval '1 second');
    SELECT * INTO STRICT v_fila FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh
     WHERE expediente_ref = v_ref AND version = v_ultima + i;
    IF i = 2 THEN v_primera_fisc := v_instante; END IF;
    IF (i = 1 AND v_fila.fase_desde <> (SELECT fase_desde FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh
                                           WHERE expediente_ref = v_ref AND version = v_ultima))
       OR (i IN (2, 4) AND v_fila.fase_desde <> v_instante)
       OR (i = 3 AND v_fila.fase_desde <> v_primera_fisc) THEN
        RAISE EXCEPTION 'CT110: disparador incorrecto en el paso %', i;
    END IF;
  END LOOP;
  IF EXISTS (
        SELECT e.expediente_ref, e.version
          FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh e
          JOIN vec_contratacion_temporal.publicacion_version_rrhh p USING (expediente_ref, version)
         WHERE e.fase_clave <> p.fase_clave
            OR e.fase_desde <> (
               SELECT q.actualizado_en FROM vec_contratacion_temporal.publicacion_version_rrhh q
                WHERE q.expediente_ref = p.expediente_ref AND q.version <= p.version
                  AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.publicacion_version_rrhh r
                                   WHERE r.expediente_ref = p.expediente_ref
                                     AND r.version BETWEEN q.version AND p.version
                                     AND r.fase_clave <> p.fase_clave)
                ORDER BY q.version LIMIT 1)
    ) THEN
    RAISE EXCEPTION 'CT110: disparador y relleno discrepan';
  END IF;
END $$;

DO $$
BEGIN
  BEGIN
    UPDATE vec_contratacion_temporal.fase_entrada_publicacion_rrhh SET fase_desde = fase_desde;
    RAISE EXCEPTION 'CT110: UPDATE admitido';
  EXCEPTION WHEN OTHERS THEN
    IF SQLERRM LIKE 'CT110:%' THEN RAISE; END IF;
  END;
  BEGIN
    DELETE FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh;
    RAISE EXCEPTION 'CT110: DELETE admitido';
  EXCEPTION WHEN OTHERS THEN
    IF SQLERRM LIKE 'CT110:%' THEN RAISE; END IF;
  END;
END $$;

DO $$
DECLARE
  v_canon bytea; v_esperadas text[]; v_leidas text[];
BEGIN
  WITH ultimas AS (
    SELECT DISTINCT ON (p.expediente_ref COLLATE "C") p.*
      FROM vec_contratacion_temporal.publicacion_version_rrhh p
     ORDER BY p.expediente_ref COLLATE "C", p.version DESC
  ), primeras AS (
    SELECT * FROM ultimas ORDER BY actualizado_en DESC, expediente_ref COLLATE "C" DESC LIMIT 100
  )
  SELECT vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
           clock_timestamp() + interval '10 days',
           array_agg(ROW(u.expediente_ref, u.organizacion_ref, u.numero_visible, u.version,
             u.flujo_ref, u.flujo_version, u.flujo_huella_sha256, u.fase_clave,
             u.estado_clave, u.centro_ref, u.categoria_ref, COALESCE(u.modalidad_clave, ''),
             COALESCE(u.unidad_ref, ''), u.creado_en, u.actualizado_en
           )::vec_contratacion_temporal.resumen_publicacion_rrhh_v1
           ORDER BY u.actualizado_en DESC, u.expediente_ref COLLATE "C" DESC),
           true, decode(repeat('ab', 32), 'hex')),
         array_agg(u.expediente_ref ORDER BY u.actualizado_en DESC, u.expediente_ref COLLATE "C" DESC)
    INTO v_canon, v_esperadas
    FROM primeras u;
  SELECT array_agg(l.expediente_ref ORDER BY l.orden) INTO v_leidas
    FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(v_canon) l
    JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh e USING (expediente_ref, version);
  IF v_leidas IS DISTINCT FROM v_esperadas THEN
    RAISE EXCEPTION 'CT110: lector del canon incorrecto';
  END IF;
  BEGIN
    PERFORM * FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(
      convert_to('VEC-CT-CONTENIDO-CUADRO-RRHH-V1' || chr(10) || '3:abc' || chr(10) || '1:1' || chr(10) || '99:x', 'UTF8'));
    RAISE EXCEPTION 'CT110: canon truncado admitido';
  EXCEPTION WHEN invalid_parameter_value THEN NULL;
  END;
END $$;

SELECT 'CT110 correcto' AS resultado;
ROLLBACK;
