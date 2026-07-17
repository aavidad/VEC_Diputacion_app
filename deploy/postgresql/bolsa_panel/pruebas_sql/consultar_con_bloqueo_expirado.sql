BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_bolsa_panel_propietario;

SELECT resultado.panel_canonico
  FROM public.vec_panel_concurrencia_expiracion AS fixture
  CROSS JOIN LATERAL vec_bolsa_panel.consultar_panel_interno_v1(
      fixture.operacion,
      fixture.prueba,
      fixture.decision_canonica,
      fixture.motivo_canonico,
      fixture.correlacion_ref
  ) AS resultado
 WHERE fixture.control_id = true;
COMMIT;
