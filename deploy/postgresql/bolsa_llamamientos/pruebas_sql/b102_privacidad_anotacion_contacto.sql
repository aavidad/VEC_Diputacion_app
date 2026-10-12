\set ON_ERROR_STOP on
-- Se ejecuta después de B102 en un PostgreSQL 18 desechable.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
CREATE TEMP TABLE b102_contacto (LIKE vec_bolsa_llamamientos.contacto_participacion INCLUDING CONSTRAINTS) ON COMMIT DROP;
DO $prueba$
DECLARE caso record; rechazado boolean; n integer:=0;
BEGIN
 FOR caso IN
  SELECT * FROM (VALUES
   ('Acepta por teléfono el llamamiento',false),
   ('TELÉFONO sin respuesta',false),
   ('Se llamó 2 veces y respondió por teléfono',false),
   ('Por teléfono a las 12:30',false),
   ('Por teléfono el 11/10/2026',false),
   ('Teléfono 600123456',true),
   ('teléfono: 6',true),
   ('teléfono / (6)',true),
   ('TELÉFONO +34 600 123 456',true),
   ('Llamar al 600123456',true),
   ('Llamar al 600-123-456',true),
   ('Llamar al +34 600 123 456',true),
   ('Teléfono fijo 812345678',true),
   ('Llamar al 812345678',true),
   ('Teléfono fijo +44 20 7946 0123',true),
   ('Llamar al +44 20 7946 0123',true),
   ('Documento DNI 12345678Z',true),
   ('Pasaporte de otra persona',true),
   ('Correo tercero@example.test',true),
   ('Enviar por email',true)
  ) AS v(anotacion,debe_rechazar)
 LOOP
  n:=n+1;
  rechazado:=false;
  BEGIN
   INSERT INTO pg_temp.b102_contacto(contacto_ref,bolsa_ref,participacion_ref,canal,
     instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref,registrada_en,instante_servidor)
   VALUES('contacto:b102:'||n,'bolsa:b102','participacion:b102','telefono',
     '2026-10-11T10:00:00Z','per_0123456789abcdefghijkl','contactado',
     caso.anotacion,'b102:'||n,'recibo:b102:'||n,'2026-10-11T10:00:00Z',false);
  EXCEPTION WHEN check_violation THEN rechazado:=true;
  END;
  IF rechazado IS DISTINCT FROM caso.debe_rechazar THEN
   RAISE EXCEPTION 'B102: resultado de privacidad inesperado en caso %',n;
  END IF;
 END LOOP;
END $prueba$;
SELECT 'B102_PRIVACIDAD_OK';
ROLLBACK;
