BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref,tipo,necesidad_ref,version_necesidad,registro_canonico,registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en)
VALUES('operacion:2','orden','necesidad:2',1,convert_to('{"propuesta":{"participacion_seleccionada_ref":"participacion:1"}}','UTF8'),encode(sha256(convert_to('{"propuesta":{"participacion_seleccionada_ref":"participacion:1"}}','UTF8')),'hex'),repeat('a',64),'decision:2','recibo:op:2','2026-09-20T00:00:00Z');
INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo(llamamiento_ref,operacion_ref,propuesta_ref,bolsa_ref,necesidad_ref,version,estado,abierto_en,datos_canonicos)
VALUES('llamamiento:2','operacion:2','propuesta:2','bolsa:1','necesidad:2',1,'abierto','2026-09-20T00:00:00Z','{}');
COMMIT;
