-- Bolsa sintética para el ensayo de las migraciones revisadas: una bolsa
-- vigente con cuatro participaciones disponibles, sembrada como superusuario
-- sin la cadena de acta (session_replication_role = replica). Solo para la
-- base desechable del ensayo.
BEGIN;
SET LOCAL timezone = 'UTC';
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:rev',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'categoria:rev',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
VALUES('instantanea:rev',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,'bolsa:rev',1,encode(sha256('{}'::bytea),'hex'),4,now()-interval '10 days',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
VALUES('acta:rev','bolsa:rev',1,encode(sha256('{}'::bytea),'hex'),'instantanea:rev',1,encode(sha256('{}'::bytea),'hex'),'categoria:rev','sistema:prueba',now()-interval '10 days',now()-interval '10 days');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
SELECT 'instantanea:rev',1,n,'participacion:rev:' || n,n FROM generate_series(1,4) n;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT 'participacion:rev:' || n,'disponible',now()-interval '10 days','Constitución de bolsa','sistema:constitucion',now()-interval '10 days','constitucion:participacion:rev:' || n,'recibo:situacion:constitucion:participacion:rev:' || n
  FROM generate_series(1,4) n;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES('politica:rev','bolsa:rev',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Prueba revisión','sistema:prueba',now()-interval '10 days',NULL,now()-interval '10 days');
COMMIT;
