\set ON_ERROR_STOP on

SET search_path = pg_catalog;
SET timezone = 'UTC';

CREATE TEMP TABLE pg_temp.estado_restaurable_v3 (
    objeto text NOT NULL,
    contenido text NOT NULL
);

DO $estado_restaurable$
DECLARE
    relacion record;
BEGIN
    FOR relacion IN
        SELECT espacio.nspname AS esquema, clase.relname AS tabla
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE clase.relkind IN ('r', 'p')
           AND espacio.nspname IN (
               'vec_autorizacion',
               'vec_bolsa_baremacion',
               'vec_prueba_bolsa_baremacion_v3'
           )
         ORDER BY espacio.nspname, clase.relname
    LOOP
        INSERT INTO pg_temp.estado_restaurable_v3 (objeto, contenido)
        VALUES (
            relacion.esquema || '.' || relacion.tabla,
            '<tabla>'
        );

        EXECUTE pg_catalog.format(
            'INSERT INTO pg_temp.estado_restaurable_v3 (objeto, contenido)
             SELECT %L, pg_catalog.to_jsonb(fila)::text
               FROM %I.%I AS fila',
            relacion.esquema || '.' || relacion.tabla,
            relacion.esquema,
            relacion.tabla
        );
    END LOOP;
END
$estado_restaurable$;

SELECT objeto || E'\t' || contenido
  FROM pg_temp.estado_restaurable_v3
 ORDER BY objeto, contenido;
