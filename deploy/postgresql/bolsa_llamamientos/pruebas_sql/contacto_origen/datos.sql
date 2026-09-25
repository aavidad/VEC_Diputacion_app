-- Una bolsa constituida con dos participaciones, como la deja la constitución.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,vigente_desde,estado,registrada_en)
VALUES('bolsa:1',1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),'categoria:prueba','2026-09-18T00:00:00Z','vigente','2026-09-18T00:00:00Z');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
SELECT 'instantanea:1',1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),'bolsa:1',1,huella_bolsa_sha256,2,registrada_en,registrada_en,registrada_en FROM vec_bolsa_llamamientos.bolsa_constituida;
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:1','bolsa:1',1,b.huella_bolsa_sha256,'instantanea:1',1,i.huella_instantanea_sha256,'categoria:prueba','actor:prueba',b.registrada_en,b.registrada_en FROM vec_bolsa_llamamientos.bolsa_constituida b JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i USING(bolsa_ref);
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero) VALUES('instantanea:1',1,1,'participacion:1',2),('instantanea:1',1,2,'participacion:2',3);
COMMIT;
