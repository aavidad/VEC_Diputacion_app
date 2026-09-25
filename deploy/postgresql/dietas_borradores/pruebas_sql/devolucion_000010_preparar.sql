\set ON_ERROR_STOP on
-- Sólo para el arnés desechable de Dietas 000010. Historia sintética de una
-- comisión enviada (v2) y devuelta por el responsable (v3) con su motivo, y de
-- otra comisión que nunca salió de borrador. Se inserta sin disparadores: el
-- circuito real ya está ensayado en 000008; aquí se ensaya lo que 000010 lee.
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.borrador_comision(referencia,persona_ref,empleado_ref,relacion_ref,unidad_ref,relacion_version,procedencia_acto_ref,fuente_ref,fuente_version,fecha_inicio,fecha_fin,motivo,codigos_ruta,clave_idempotencia,huella_semantica_sha256,creada_en)
VALUES ('dco_DDDDDDDDDDDDDDDDDDDDDD','per_TTTTTTTTTTTTTTTTTTTTTT','emp_EEEEEEEEEEEEEEEEEEEEEE','rel_RRRRRRRRRRRRRRRRRRRRRR','U1',1,'acto','fuente',1,DATE '2026-09-23',DATE '2026-09-23','Visita sintética','["GR1","GR2"]','clave_devolucion_d6_0001',repeat('a',64),TIMESTAMPTZ '2026-09-23 07:00:00+00'),
       ('dco_BBBBBBBBBBBBBBBBBBBBBB','per_TTTTTTTTTTTTTTTTTTTTTT','emp_EEEEEEEEEEEEEEEEEEEEEE','rel_RRRRRRRRRRRRRRRRRRRRRR','U1',1,'acto','fuente',1,DATE '2026-09-24',DATE '2026-09-24','Borrador sintético','["GR1","GR2"]','clave_devolucion_d6_0002',repeat('b',64),TIMESTAMPTZ '2026-09-24 07:00:00+00');
INSERT INTO vec_dietas.numero_documento_comision VALUES
 ('dco_DDDDDDDDDDDDDDDDDDDDDD',900001,'VEC-D-2026-900001',TIMESTAMPTZ '2026-09-23 07:00:00+00'),
 ('dco_BBBBBBBBBBBBBBBBBBBBBB',900002,'VEC-D-2026-900002',TIMESTAMPTZ '2026-09-24 07:00:00+00');
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,registrada_en)
SELECT c.ref,c.v,c.estado,c.fecha,c.fecha,'08:00','12:00',c.motivo,'["GR1","GR2"]','{}','{}',
 (SELECT regla_ref FROM vec_dietas.regla_devengo_provisional LIMIT 1),c.en
FROM (VALUES
 ('dco_DDDDDDDDDDDDDDDDDDDDDD',2,'enviado_pendiente_revision',DATE '2026-09-23','Visita sintética',TIMESTAMPTZ '2026-09-23 08:00:00+00'),
 ('dco_DDDDDDDDDDDDDDDDDDDDDD',3,'devuelta',DATE '2026-09-23','Visita sintética',TIMESTAMPTZ '2026-09-23 09:00:00+00'),
 ('dco_BBBBBBBBBBBBBBBBBBBBBB',2,'borrador',DATE '2026-09-24','Borrador sintético',TIMESTAMPTZ '2026-09-24 08:00:00+00')) c(ref,v,estado,fecha,motivo,en);
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT r.recibo,r.ref,r.v,r.op,r.clave,'{}',repeat('c',64),'dec',repeat('d',64),'aud','act_sintetico','per_TTTTTTTTTTTTTTTTTTTTTT',
 (SELECT regla_ref FROM vec_dietas.regla_devengo_provisional LIMIT 1),repeat('e',64),r.en
FROM (VALUES
 ('rcd_00000000-0000-4000-8000-000000000002','dco_DDDDDDDDDDDDDDDDDDDDDD',2,'enviar','claveEnvioSintetica0001',TIMESTAMPTZ '2026-09-23 08:00:00+00'),
 ('rcd_00000000-0000-4000-8000-000000000003','dco_DDDDDDDDDDDDDDDDDDDDDD',3,'circuito_autorizacion','claveDevolucionSint0001',TIMESTAMPTZ '2026-09-23 09:00:00+00'),
 ('rcd_00000000-0000-4000-8000-000000000012','dco_BBBBBBBBBBBBBBBBBBBBBB',2,'editar','claveEdicionSintetica01',TIMESTAMPTZ '2026-09-24 08:00:00+00')) r(recibo,ref,v,op,clave,en);
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_02','dco_DDDDDDDDDDDDDDDDDDDDDD',2,'rcd_00000000-0000-4000-8000-000000000002','borrador','enviado_pendiente_revision','enviar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 08:00:00+00'),
 ('hdi_sintetico_03','dco_DDDDDDDDDDDDDDDDDDDDDD',3,'rcd_00000000-0000-4000-8000-000000000003','pendiente_autorizacion','devuelta','circuito_autorizacion','Falta el justificante del taxi','act_responsable',TIMESTAMPTZ '2026-09-23 09:00:00.123456+00'),
 ('hdi_sintetico_12','dco_BBBBBBBBBBBBBBBBBBBBBB',2,'rcd_00000000-0000-4000-8000-000000000012','borrador','borrador','editar',NULL,'act_titular',TIMESTAMPTZ '2026-09-24 08:00:00+00');
COMMIT;
-- Sólo en este contenedor: el login de prueba invoca las proyecciones.
GRANT EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean),
 vec_dietas.proyectar_revision_exacta_v2(text,bigint) TO vec_prueba_dietas;
