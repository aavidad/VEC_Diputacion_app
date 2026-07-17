DO $verificacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_bolsa_panel.atestacion_autorizacion_version
         WHERE decision_ref = 'decision:panel:concurrencia:1'
           AND clock_timestamp() >= valida_hasta
    ) THEN
        RAISE EXCEPTION 'la prueba no cruzo la expiracion';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_bolsa_panel.consulta_confirmada)
       OR EXISTS (SELECT 1 FROM vec_bolsa_panel.auditoria)
       OR (SELECT ultima_secuencia FROM vec_bolsa_panel.auditoria_actual
            WHERE control_id = true) <> 0 THEN
        RAISE EXCEPTION 'una espera caducada produjo efectos';
    END IF;
END
$verificacion$;
