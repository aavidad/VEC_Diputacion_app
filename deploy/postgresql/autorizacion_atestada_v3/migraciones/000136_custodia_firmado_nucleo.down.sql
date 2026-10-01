\set ON_ERROR_STOP on
-- AD3-136 DOWN queda cerrado por defecto. AD3-133/135 todavía no publican una
-- cadena inversa medida. Retirar sólo el caso del núcleo sin conocer esa
-- postimagen podría dejar consumidores o historia sin autoridad válida.
-- Una futura reversión exige clon vacío sin custodias, Documentos9 retirada,
-- preimagen global post-B/post-AD136 y dos revisiones sobre el orden completo.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
DO $denegar$
BEGIN
 RAISE EXCEPTION 'AD3-136 DOWN: falta cadena inversa medida de AD3-133/135'
   USING ERRCODE='55000';
END $denegar$;
COMMIT;
