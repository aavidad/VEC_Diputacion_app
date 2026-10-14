\set ON_ERROR_STOP on
-- Datos ficticios del recorrido: corrige una etiqueta y conserva las revisiones anteriores.
-- No crea cargos, permisos, firmas, actos de administración ni recibos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:organizacion:1',0));
DO $datos$
DECLARE
 anterior vec_contratacion_temporal.organizacion_catalogo_revision%ROWTYPE;
 doc jsonb; etiqueta text; nuevo text; ahora timestamptz; marca text;
 clave constant text := 'local-4f671181-dfff-4da4-ac9b-67d59d963e9d';
BEGIN
 SELECT * INTO STRICT anterior
 FROM vec_contratacion_temporal.organizacion_catalogo_revision
 WHERE version=1 ORDER BY revision DESC LIMIT 1;
 doc:=anterior.catalogo::jsonb;
 IF doc->>'id' IS DISTINCT FROM 'estructura-organizativa-dipgra'
    OR doc->>'estado' IS DISTINCT FROM 'borrador'
    OR (SELECT count(*) FROM jsonb_array_elements(doc->'entradas') e WHERE e->>'clave'=clave)<>1
 THEN RAISE EXCEPTION 'cargo del recorrido: catálogo incompatible' USING ERRCODE='55000'; END IF;
 SELECT e->>'etiqueta' INTO STRICT etiqueta
 FROM jsonb_array_elements(doc->'entradas') e WHERE e->>'clave'=clave;
 IF etiqueta='Subdirección de Informática' THEN RETURN; END IF;
 IF etiqueta IS DISTINCT FROM 'Subdirección de Informática — prueba sintética'
 THEN RAISE EXCEPTION 'cargo del recorrido: etiqueta distinta de la prevista' USING ERRCODE='55000'; END IF;
 -- Conserva el canon Go instalado: JSONB reordena las claves y rompería la
 -- huella que devuelve la pantalla para una edición posterior del catálogo.
 nuevo:=anterior.catalogo;
 marca:='"version":1,"revision":'||anterior.revision||',';
 IF (length(nuevo)-length(replace(nuevo,marca,'')))/length(marca)<>1
    OR (length(nuevo)-length(replace(nuevo,'"etiqueta":'||to_json(etiqueta)::text,''))) /
       length('"etiqueta":'||to_json(etiqueta)::text)<>1
 THEN RAISE EXCEPTION 'cargo del recorrido: canon de partida incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(nuevo,marca,'"version":1,"revision":'||(anterior.revision+1)||',');
 nuevo:=replace(nuevo,'"etiqueta":'||to_json(etiqueta)::text,
   '"etiqueta":'||to_json('Subdirección de Informática'::text)::text);
 ahora:=date_trunc('seconds',clock_timestamp());
 nuevo:=replace(nuevo,'"ultima_modificacion_por":'||to_json(doc->>'ultima_modificacion_por')::text,
   '"ultima_modificacion_por":"datos:recorrido:hz39"');
 nuevo:=replace(nuevo,'"ultima_modificacion_en":'||to_json(doc->>'ultima_modificacion_en')::text,
   '"ultima_modificacion_en":'||to_json(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'))::text);
 nuevo:=replace(nuevo,'"motivo_modificacion":'||to_json(doc->>'motivo_modificacion')::text,
   '"motivo_modificacion":"Corrección de la denominación del puesto"');
 IF (nuevo::jsonb)->>'revision' IS DISTINCT FROM (anterior.revision+1)::text
    OR (nuevo::jsonb)->>'ultima_modificacion_por' IS DISTINCT FROM 'datos:recorrido:hz39'
 THEN RAISE EXCEPTION 'cargo del recorrido: corrección incompleta' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contratacion_temporal.organizacion_catalogo_revision
   (version,revision,catalogo,huella_sha256,registrada_en)
 VALUES(1,anterior.revision+1,nuevo,encode(sha256(convert_to(nuevo,'UTF8')),'hex'),ahora);
END $datos$;
COMMIT;
