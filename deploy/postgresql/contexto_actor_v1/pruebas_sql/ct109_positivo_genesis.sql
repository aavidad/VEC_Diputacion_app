-- Solo restauración de preimagen vacía, antes de las dos guardas de INSERT.
-- Réplica de génesis auth000003 (líneas 185–189) y auth000008 (498–505).
-- El runner instala después las dos guardas originales y coteja todo el esquema.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL row_security=on;
DO $fase_restauracion$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_checkpoint_origen)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1)
 OR EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid IN (
 'vec_autorizacion.motivo_v2_checkpoint_origen'::regclass,
 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass)
 AND tgname IN ('bloquear_insercion_checkpoint','vinculacion_motivo_rrhh_checkpoint_inmutable'))
 THEN RAISE EXCEPTION 'CT109: génesis fuera de la fase de restauración vacía'; END IF;
END $fase_restauracion$;
INSERT INTO vec_autorizacion.motivo_v2_checkpoint_origen
 (control_id,ultima_secuencia,actualizado_en)
 VALUES(true,0,clock_timestamp());
INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
 (clase_consulta,ultima_publicacion_version,actualizado_en)
 VALUES('cuadro',0,clock_timestamp()),('detalle',0,clock_timestamp());
COMMIT;
