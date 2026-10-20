-- Dato de copia L, derivado del preparador cedido por O. No se instala en producto.
-- El volcado HZ37 fijado conserva 97 claves técnicas del publicador CT
-- (base y consumidores); ninguna está revocada ni referida por emisión externa.
BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
    WHERE "clave_id"='clave:capacidad:ct:desarrollo:v2' AND version=2
      AND emisor_id='emisor:ct:desarrollo:v2'
      AND acto_ref LIKE 'acto:ct:desarrollo:clave-capacidad:%')
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version c
     WHERE c.emisor_id='emisor:ct:desarrollo:v2' AND c.version>1
       AND c.acto_ref LIKE 'acto:ct:desarrollo:clave-capacidad:%') <> 97
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version c
     WHERE c.emisor_id='emisor:ct:desarrollo:v2' AND c.version>1
       AND c.acto_ref LIKE 'acto:ct:desarrollo:clave-capacidad:%'
       AND ((c.clave_id NOT LIKE 'clave:capacidad:%:desarrollo:v2'
               AND (c.clave_id,c.version) NOT IN (
                ('clave:capacidad:ct-anotacion-administrativa:continuidad:20260912:v2',9),
                ('clave:capacidad:ct-cierre-administrativo-sin-cese:continuidad:20260912:v2',10)))
         OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
                    WHERE (r.clave_id,r.version)=(c.clave_id,c.version))
         OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa e
                    WHERE (e.clave_id,e.version)=(c.clave_id,c.version))))
 OR (SELECT count(*) FROM pg_catalog.pg_constraint
     WHERE contype='f' AND confrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass) <> 3
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_constraint f
     WHERE f.contype='f' AND f.confrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND (f.conrelid NOT IN ('vec_autorizacion_atestada_v3.puntero_clave_emision'::regclass,
           'vec_autorizacion_atestada_v3.puntero_clave_emision_externa'::regclass,
           'vec_autorizacion_atestada_v3.revocacion_clave_capacidad'::regclass)
         OR pg_catalog.pg_get_constraintdef(f.oid) <> 'FOREIGN KEY (clave_id, version) REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id, version)'))
 THEN RAISE EXCEPTION 'Preimagen de datos de copia L divergente'; END IF;
END $pre$;
ALTER TABLE vec_autorizacion_atestada_v3.puntero_clave_emision DISABLE TRIGGER ALL;
ALTER TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual DISABLE TRIGGER ALL;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DISABLE TRIGGER ALL;
TRUNCATE vec_autorizacion_atestada_v3.puntero_clave_emision,vec_autorizacion_atestada_v3.puntero_configuracion_actual;
DO $borrado$
DECLARE retiradas integer;
BEGIN
 DELETE FROM vec_autorizacion_atestada_v3.clave_capacidad_version c
  WHERE c.emisor_id='emisor:ct:desarrollo:v2' AND c.version>1
    AND c.acto_ref LIKE 'acto:ct:desarrollo:clave-capacidad:%';
 GET DIAGNOSTICS retiradas = ROW_COUNT;
 IF retiradas <> 97 OR EXISTS (
      SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
      WHERE "clave_id"='clave:capacidad:ct:desarrollo:v2' AND version=2)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version c
      WHERE c.emisor_id='emisor:ct:desarrollo:v2' AND c.version>1
        AND c.acto_ref LIKE 'acto:ct:desarrollo:clave-capacidad:%')
 THEN RAISE EXCEPTION 'Retirada de datos de copia L incompleta'; END IF;
END $borrado$;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ENABLE TRIGGER ALL;
ALTER TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual ENABLE TRIGGER ALL;
ALTER TABLE vec_autorizacion_atestada_v3.puntero_clave_emision ENABLE TRIGGER ALL;
COMMIT;
