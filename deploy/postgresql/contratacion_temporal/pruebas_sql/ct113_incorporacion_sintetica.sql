-- Inserta, como superusuario y sin disparadores (session_replication_role),
-- una incorporación CT75 sintética para la propuesta número :n (orden por
-- expediente). No hace BEGIN ni COMMIT: lo decide quien la incluye, para
-- poder mantener la transacción abierta en la prueba de concurrencia.
SET LOCAL session_replication_role = replica;
SELECT set_config('prueba.ct113.n', :'n', true);
DO $fixture$
DECLARE p record; n integer := current_setting('prueba.ct113.n')::integer; ref text; hx text;
BEGIN
 SELECT * INTO STRICT p FROM vec_contratacion_temporal.propuesta_formalizacion ORDER BY expediente_ref OFFSET n - 1 LIMIT 1;
 hx := encode(sha256(convert_to('ct113:' || p.expediente_ref, 'UTF8')), 'hex');
 ref := 'ref:' || hx;
 INSERT INTO vec_contratacion_temporal.incorporacion_outbox_v2(outbox_ref, recibo_ref, evento_json, estado_sha256, creada_en)
 VALUES ('ref:outbox:' || hx, ref, jsonb_build_object('recibo_ref', ref), repeat('a', 64), clock_timestamp());
 INSERT INTO vec_contratacion_temporal.incorporacion_registro_v2(
   recibo_ref, seguimiento_ref, organizacion_ref, idempotencia_ref, solicitud_ref, expediente_ref,
   version_expediente, version_anterior, version_resultante, estado_anterior_sha256, estado_resultante_sha256,
   material_json, material_canonico, material_sha256, intencion_canonica, intencion_sha256, exportacion_ct,
   persona_version, perfil_version, recibo_json, auditoria_ref, outbox_ref, registrada_en, evidencia_orden_json)
 VALUES (ref, 'seguimiento:ct113:' || n, p.organizacion_ref, 'idem:ct113:' || n, 'solicitud:ct113:' || n, p.expediente_ref,
   p.version_resultante, 0, 1, repeat('b', 64), repeat('c', 64),
   jsonb_build_object('Confirmacion', jsonb_build_object('PeriodoIncorporacion',
      jsonb_build_object('desde', '2027-01-04T00:00:00Z', 'hasta', '2027-03-31T00:00:00Z'))),
   '\x01'::bytea, encode(sha256('\x01'::bytea), 'hex'), '\x02'::bytea, encode(sha256('\x02'::bytea), 'hex'),
   ARRAY['\x01','\x02','\x03','\x04','\x05','\x06','\x07','\x08']::bytea[], 1, 1,
   jsonb_build_object('MaterialOriginalSHA256', encode(sha256('\x01'::bytea), 'hex'),
     'IntencionSHA256', encode(sha256('\x02'::bytea), 'hex'), 'SeguimientoRef', 'seguimiento:ct113:' || n,
     'AuditoriaCTRef', 'ref:auditoria:' || hx, 'OutboxCTRef', 'ref:outbox:' || hx,
     'Transicion', jsonb_build_object('recibo_ref', ref), 'EjercicioSintetico', true,
     'FirmaOficial', false, 'EficaciaAdministrativa', false),
   'ref:auditoria:' || hx, 'ref:outbox:' || hx, clock_timestamp(), '{}'::jsonb);
END $fixture$;
SET LOCAL session_replication_role = origin;
