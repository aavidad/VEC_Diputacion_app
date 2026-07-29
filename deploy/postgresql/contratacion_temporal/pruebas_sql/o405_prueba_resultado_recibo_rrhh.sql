\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
SET search_path = pg_catalog;

CREATE FUNCTION vec_contratacion_temporal.canon_recibo_desde_prueba_ct43(
    p_prueba vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
)
RETURNS bytea
LANGUAGE sql
STABLE
STRICT
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
        ROW(
            'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2',
            p_prueba.acceso_ref, p_prueba.secuencia,
            p_prueba.anterior_sha256, p_prueba.huella_sha256,
            p_prueba.vinculo_identidad_huella_sha256,
            COALESCE(p_prueba.alcance_huella_sha256, ''),
            p_prueba.registrada_en, p_prueba.auditoria_vec_ref,
            p_prueba.auditoria_vec_huella_sha256,
            p_prueba.consumo_vec_huella_sha256,
            p_prueba.decision_ref, p_prueba.decision_huella_sha256,
            p_prueba.capacidad_huella_sha256,
            p_prueba.material_huella_sha256,
            p_prueba.consulta_huella_sha256,
            p_prueba.correlacion_ref, p_prueba.autenticacion_ref,
            p_prueba.autenticacion_huella_sha256,
            p_prueba.sesion_ref, p_prueba.control_sesion_ref,
            p_prueba.control_sesion_revision,
            p_prueba.control_sesion_huella_sha256,
            p_prueba.actor_ref, p_prueba.perfil_ref,
            p_prueba.perfil_version, p_prueba.organizacion_ref,
            p_prueba.clase_ambito, p_prueba.ambito_ref,
            p_prueba.accion, p_prueba.finalidad,
            COALESCE(p_prueba.expediente_ref, ''),
            COALESCE(p_prueba.version_expediente, 0),
            p_prueba.total, p_prueba.contenido_huella_sha256,
            p_prueba.resultado_huella_sha256,
            COALESCE(p_prueba.cursor_huella_sha256, ''),
            p_prueba.generada_en
        )::vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
    );
END;

DO $prueba_durable$
DECLARE
    v_cuadros integer;
    v_detalles integer;
BEGIN
    SELECT pg_catalog.count(*) FILTER (
               WHERE tipo_consulta = 'cuadro'
           ),
           pg_catalog.count(*) FILTER (
               WHERE tipo_consulta = 'detalle'
           )
      INTO v_cuadros, v_detalles
      FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2;
    IF v_cuadros < 2 OR v_detalles < 1
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .prueba_resultado_recibo_rrhh_v2 prueba
            WHERE prueba.recibo_sello_sha256 <>
                  pg_catalog.encode(
                      pg_catalog.sha256(prueba.recibo_canonico), 'hex'
                  )
               OR prueba.contenido_huella_sha256 <>
                  pg_catalog.encode(
                      pg_catalog.sha256(prueba.contenido_canonico), 'hex'
                  )
               OR prueba.resultado_huella_sha256 <>
                  pg_catalog.encode(
                      pg_catalog.sha256(prueba.resultado_canonico), 'hex'
                  )
               OR prueba.recibo_canonico <>
                  vec_contratacion_temporal
                  .canon_recibo_desde_prueba_ct43(prueba)
               OR (
                   prueba.tipo_consulta = 'cuadro'
                   AND prueba.total <>
                       pg_catalog.cardinality(prueba.resumenes)
               )
               OR (
                   prueba.tipo_consulta = 'detalle'
                   AND (
                       prueba.total <> 1
                       OR prueba.expediente_ref <>
                          (prueba.detalle).resumen.expediente_ref
                       OR prueba.version_expediente <>
                          (prueba.detalle).resumen.version
                   )
               )
       ) THEN
        RAISE EXCEPTION 'prueba durable CT43 divergente';
    END IF;
END
$prueba_durable$;

ALTER TABLE
vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
DISABLE TRIGGER prueba_resultado_recibo_rrhh_v2_inmutable;

DO $material_y_relaciones$
DECLARE
    v_origen
        vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2%ROWTYPE;
    v_destino
        vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2%ROWTYPE;
    v_nuevo
        vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v_origen
      FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
     WHERE tipo_consulta = 'cuadro'
     ORDER BY registrada_en, acceso_ref
     LIMIT 1;
    SELECT * INTO STRICT v_destino
      FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
     WHERE tipo_consulta = 'cuadro'
     ORDER BY registrada_en DESC, acceso_ref DESC
     LIMIT 1;

    BEGIN
        UPDATE vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
           SET total = 0
         WHERE acceso_ref = v_destino.acceso_ref;
        RAISE EXCEPTION 'total/material de cuadro cruzable';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
    BEGIN
        UPDATE vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
           SET expediente_ref = expediente_ref || ':otro'
         WHERE tipo_consulta = 'detalle';
        RAISE EXCEPTION 'identidad material de detalle cruzable';
    EXCEPTION WHEN check_violation THEN NULL;
    END;

    v_nuevo := v_destino;
    v_nuevo.tipo_consulta := v_origen.tipo_consulta;
    v_nuevo.expediente_ref := v_origen.expediente_ref;
    v_nuevo.version_expediente := v_origen.version_expediente;
    v_nuevo.total := v_origen.total;
    v_nuevo.generada_en := v_origen.generada_en;
    v_nuevo.resumenes := v_origen.resumenes;
    v_nuevo.hay_mas := v_origen.hay_mas;
    v_nuevo.cursor_material_huella_sha256 :=
        v_origen.cursor_material_huella_sha256;
    v_nuevo.detalle := v_origen.detalle;
    v_nuevo.contenido_canonico := v_origen.contenido_canonico;
    v_nuevo.contenido_huella_sha256 :=
        v_origen.contenido_huella_sha256;
    v_nuevo.cursor_huella_sha256 := v_origen.cursor_huella_sha256;
    v_nuevo.resultado_canonico := v_origen.resultado_canonico;
    v_nuevo.resultado_huella_sha256 :=
        v_origen.resultado_huella_sha256;
    v_nuevo.recibo_canonico :=
        vec_contratacion_temporal
        .canon_recibo_desde_prueba_ct43(v_nuevo);
    v_nuevo.recibo_sello_sha256 := pg_catalog.encode(
        pg_catalog.sha256(v_nuevo.recibo_canonico), 'hex'
    );
    BEGIN
        UPDATE vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
           SET tipo_consulta = v_nuevo.tipo_consulta,
               expediente_ref = v_nuevo.expediente_ref,
               version_expediente = v_nuevo.version_expediente,
               total = v_nuevo.total,
               generada_en = v_nuevo.generada_en,
               resumenes = v_nuevo.resumenes,
               hay_mas = v_nuevo.hay_mas,
               cursor_material_huella_sha256 =
                   v_nuevo.cursor_material_huella_sha256,
               detalle = v_nuevo.detalle,
               contenido_canonico = v_nuevo.contenido_canonico,
               contenido_huella_sha256 =
                   v_nuevo.contenido_huella_sha256,
               cursor_huella_sha256 = v_nuevo.cursor_huella_sha256,
               resultado_canonico = v_nuevo.resultado_canonico,
               resultado_huella_sha256 =
                   v_nuevo.resultado_huella_sha256,
               recibo_canonico = v_nuevo.recibo_canonico,
               recibo_sello_sha256 = v_nuevo.recibo_sello_sha256
         WHERE acceso_ref = v_destino.acceso_ref;
        RAISE EXCEPTION 'resultado cruzado para el mismo acceso';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;

    v_nuevo := v_destino;
    v_nuevo.auditoria_vec_ref := v_origen.auditoria_vec_ref;
    v_nuevo.auditoria_vec_huella_sha256 :=
        v_origen.auditoria_vec_huella_sha256;
    v_nuevo.decision_ref := v_origen.decision_ref;
    v_nuevo.decision_huella_sha256 :=
        v_origen.decision_huella_sha256;
    v_nuevo.capacidad_huella_sha256 :=
        v_origen.capacidad_huella_sha256;
    v_nuevo.consulta_huella_sha256 :=
        v_origen.consulta_huella_sha256;
    v_nuevo.correlacion_ref := v_origen.correlacion_ref;
    v_nuevo.accion := v_origen.accion;
    v_nuevo.finalidad := v_origen.finalidad;
    v_nuevo.recibo_canonico :=
        vec_contratacion_temporal
        .canon_recibo_desde_prueba_ct43(v_nuevo);
    v_nuevo.recibo_sello_sha256 := pg_catalog.encode(
        pg_catalog.sha256(v_nuevo.recibo_canonico), 'hex'
    );
    BEGIN
        UPDATE vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
           SET auditoria_vec_ref = v_nuevo.auditoria_vec_ref,
               auditoria_vec_huella_sha256 =
                   v_nuevo.auditoria_vec_huella_sha256,
               decision_ref = v_nuevo.decision_ref,
               decision_huella_sha256 =
                   v_nuevo.decision_huella_sha256,
               capacidad_huella_sha256 =
                   v_nuevo.capacidad_huella_sha256,
               consulta_huella_sha256 =
                   v_nuevo.consulta_huella_sha256,
               correlacion_ref = v_nuevo.correlacion_ref,
               accion = v_nuevo.accion,
               finalidad = v_nuevo.finalidad,
               recibo_canonico = v_nuevo.recibo_canonico,
               recibo_sello_sha256 = v_nuevo.recibo_sello_sha256
         WHERE acceso_ref = v_destino.acceso_ref;
        RAISE EXCEPTION 'prueba VEC cruzada para el mismo acceso';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;
END
$material_y_relaciones$;

ALTER TABLE
vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
ENABLE TRIGGER prueba_resultado_recibo_rrhh_v2_inmutable;

DO $inmutabilidad$
BEGIN
    BEGIN
        UPDATE vec_contratacion_temporal
               .prueba_resultado_recibo_rrhh_v2
           SET acceso_ref = acceso_ref;
        RAISE EXCEPTION 'prueba CT43 actualizable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_contratacion_temporal
                    .prueba_resultado_recibo_rrhh_v2;
        RAISE EXCEPTION 'prueba CT43 borrable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contratacion_temporal
                 .prueba_resultado_recibo_rrhh_v2;
        RAISE EXCEPTION 'prueba CT43 truncable';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$inmutabilidad$;

DROP FUNCTION vec_contratacion_temporal.canon_recibo_desde_prueba_ct43(
    vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
);
RESET ROLE;
