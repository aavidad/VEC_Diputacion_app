\set ON_ERROR_STOP on
-- Solo en fixture: CT138 requiere la marca de CT124. No simula su circuito.
SET ROLE vec_contratacion_temporal_propietario;
DO $preimagen$
DECLARE v_def text; v_marca text;
BEGIN
    SELECT pg_get_functiondef(to_regprocedure(
      'vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')) INTO v_def;
    v_marca:=$a$    IF v_seleccion.situacion IS DISTINCT FROM 'confirmada'$a$;
    IF v_def IS NULL OR length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca) THEN
        RAISE EXCEPTION 'fixture CT138 sin preimagen CT56';
    END IF;
    v_def:=replace(v_def,v_marca,
      $n$    -- Marca sintética CT124: r.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')
    IF v_seleccion.situacion IS DISTINCT FROM 'confirmada'$n$);
    EXECUTE v_def;
END $preimagen$;
RESET ROLE;
