\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
DO $prueba$
DECLARE v_j jsonb; v_m bytea;
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_catalogo) <> 2
    OR (SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_evento) <> 2
    OR (SELECT ultima_secuencia FROM vec_contratacion_temporal.gobi_o404b_checkpoint WHERE control) <> 2 THEN
   RAISE EXCEPTION 'CT141 historia no durable tras reinicio';
 END IF;
 FOR v_j,v_m IN SELECT j,m FROM vec_ct141_prueba.v1 LOOP
   IF vec_contratacion_temporal.gobi_o404b_material_catalogo(v_j)
      IS DISTINCT FROM v_m THEN
     RAISE EXCEPTION 'CT141 preimagen cambió tras reinicio';
   END IF;
 END LOOP;
END
$prueba$;
RESET ROLE;
