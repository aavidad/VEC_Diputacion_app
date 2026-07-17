\set ON_ERROR_STOP 1

DO $prueba$
BEGIN
    IF (SELECT ultima_secuencia
          FROM vec_autorizacion.motivo_v2_checkpoint_origen
         WHERE control_id) <> 2
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_evento_origen) <> 2
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_retirada) <> 1
       OR (SELECT catalogo_huella_publicada_sha256
             FROM vec_autorizacion.motivo_v2_catalogo_publicado
            WHERE catalogo_id = 'motivos_autorizacion'
              AND catalogo_version = 1) <> repeat('b', 64) THEN
        RAISE EXCEPTION 'la carrera dejo evidencia o huella publicada incorrectas';
    END IF;
END
$prueba$;
