\set ON_ERROR_STOP on
-- Predicados de B126. Solo usa valores ficticios y no escribe datos.
DO $prueba$
DECLARE c record;
BEGIN
 FOR c IN SELECT * FROM (VALUES
  ('perfil 2',repeat('A',2),false),
  ('perfil 3',repeat('A',3),true),
  ('perfil 255',repeat('A',255),true),
  ('perfil 256',repeat('A',256),true),
  ('perfil 257',repeat('A',257),false),
  ('perfil carácter ajeno','abc@',false)
 ) AS v(nombre,valor,esperado) LOOP
  IF (octet_length(c.valor) BETWEEN 3 AND 256
      AND c.valor ~ '^[A-Za-z0-9:_-]+$') IS DISTINCT FROM c.esperado
  THEN RAISE EXCEPTION 'B126: límite de % divergente',c.nombre; END IF;
 END LOOP;
 FOR c IN SELECT * FROM (VALUES
  ('clave 7',repeat('B',7),false),
  ('clave 8',repeat('B',8),true),
  ('clave 255',repeat('B',255),true),
  ('clave 256',repeat('B',256),true),
  ('clave 257',repeat('B',257),false),
  ('clave carácter ajeno','abcdefgh@',false)
 ) AS v(nombre,valor,esperado) LOOP
  IF (octet_length(c.valor) BETWEEN 8 AND 256
      AND c.valor ~ '^[A-Za-z0-9:_-]+$') IS DISTINCT FROM c.esperado
  THEN RAISE EXCEPTION 'B126: límite de % divergente',c.nombre; END IF;
 END LOOP;
END $prueba$;
