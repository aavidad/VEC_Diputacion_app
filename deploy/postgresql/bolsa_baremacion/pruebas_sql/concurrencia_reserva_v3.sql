-- Una sesion de la carrera. Las decisiones y operaciones se preparan antes
-- para que las dos conexiones concurrentes ejecuten solo DML funcional.
\if :{?SUFIJO}
\else
    \echo 'falta la variable SUFIJO'
    \quit 3
\endif

\set VERBOSITY verbose
SELECT :'SUFIJO' IN ('a', 'b') AS sufijo_valido
\gset
\if :sufijo_valido
\else
    \echo 'SUFIJO solo admite a o b'
    \quit 3
\endif

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- Fija el snapshot SERIALIZABLE antes de la compuerta de prueba. Cada sesion
-- usa una clave distinta: al liberar ambas no se serializan artificialmente
-- entre si y recorren completa la revalidacion productiva.
SELECT current_setting('transaction_isolation') = 'serializable'
       AS aislamiento_valido
\gset
\if :aislamiento_valido
\else
    ROLLBACK;
    \quit 5
\endif
SELECT pg_current_snapshot() IS NOT NULL AS snapshot_fijado
\gset
\if :snapshot_fijado
\else
    ROLLBACK;
    \quit 6
\endif
SELECT pg_advisory_xact_lock(hashtextextended(
           'vec_prueba_bolsa_baremacion_v3:barrera_carrera:' ||
               :'SUFIJO', 0
       ));

WITH entrada AS (
    SELECT *
      FROM vec_prueba_bolsa_baremacion_v3.entrada_concurrencia
     WHERE sufijo = :'SUFIJO'
), respuesta AS MATERIALIZED (
    SELECT entrada.sufijo, resultado.*
      FROM entrada
      CROSS JOIN LATERAL
        vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
            entrada.operacion, entrada.prueba,
            entrada.decision_canonica, entrada.recurso_canonico
        ) AS resultado
), guardado AS (
    INSERT INTO vec_prueba_bolsa_baremacion_v3.resultado_concurrencia (
        sufijo, resultado, reserva_ref
    )
    SELECT sufijo, resultado, reserva_ref
      FROM respuesta
     WHERE resultado IN ('reservada', 'en_curso')
       AND archivo_probatorio_documento IS NULL
    RETURNING resultado
)
SELECT count(*) = 1 AS resultado_concurrencia_valido
  FROM guardado
\gset

\if :resultado_concurrencia_valido
    COMMIT;
\else
    ROLLBACK;
    \quit 4
\endif
