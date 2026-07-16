-- Verifica persistencia y recuperacion de respuesta perdida tras reiniciar
-- PostgreSQL. Debe ejecutarse inmediatamente despues del fixture confirmado:
-- la prueba reforzada de autorizacion tiene una frescura maxima de 30 s.
SET search_path = pg_catalog;
SET timezone = 'UTC';

SELECT operacion_prevalidacion::text AS operacion_prevalidacion,
       prueba_prevalidacion::text AS prueba_prevalidacion,
       encode(canonica_prevalidacion, 'hex') AS canonica_prevalidacion,
       encode(recurso_canonico, 'hex') AS recurso_canonico,
       operacion_confirmacion::text AS operacion_confirmacion,
       prueba_confirmacion::text AS prueba_confirmacion,
       encode(canonica_confirmacion, 'hex') AS canonica_confirmacion,
       encode(agregado_canonico, 'hex') AS agregado_canonico,
       manifiesto::text AS manifiesto,
       encode(contenido_manifiesto, 'hex') AS contenido_manifiesto,
       encode(representacion_manifiesto, 'hex') AS representacion_manifiesto,
       encode(preimagen_manifiesto, 'hex') AS preimagen_manifiesto,
       huella_prevalidacion_entrada,
       archivo_esperado::text AS archivo_esperado,
       huella_prevalidacion_final
  FROM vec_prueba_bolsa_baremacion_v3.recuperacion
\gset rec_

SELECT EXISTS (
           SELECT 1
             FROM vec_bolsa_baremacion.version_baremacion
            WHERE baremacion_merito_ref = 'baremacion:001' AND numero = 2
       ) AND (SELECT count(*)
                FROM vec_bolsa_baremacion.manifiesto_probatorio_v3) = 1
         AND vec_bolsa_baremacion.construir_archivo_probatorio_v3(
                 'baremacion:001', 2
             ) = :'rec_archivo_esperado'::jsonb
       AS persistencia_reinicio_valida
\gset
\if :persistencia_reinicio_valida
\else
    \quit 5
\endif

SET ROLE vec_bolsa_baremacion_ejecutor;
SELECT p.resultado = 'confirmada'
       AND p.archivo_probatorio_documento = :'rec_archivo_esperado'::jsonb
       AND p.huella_prevalidacion_sha256 =
           :'rec_huella_prevalidacion_final'
       AS recuperacion_prevalidacion_valida
  FROM vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
      :'rec_operacion_prevalidacion'::jsonb,
      :'rec_prueba_prevalidacion'::jsonb,
      decode(:'rec_canonica_prevalidacion', 'hex'),
      decode(:'rec_recurso_canonico', 'hex')
  ) AS p
\gset
\if :recuperacion_prevalidacion_valida
\else
    \quit 6
\endif

SELECT c.resultado = 'confirmada'
       AND c.archivo_probatorio_documento = :'rec_archivo_esperado'::jsonb
       AND c.huella_prevalidacion_sha256 =
           :'rec_huella_prevalidacion_final'
       AS recuperacion_confirmacion_valida
  FROM vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
      :'rec_operacion_confirmacion'::jsonb,
      :'rec_prueba_confirmacion'::jsonb,
      decode(:'rec_canonica_confirmacion', 'hex'),
      decode(:'rec_recurso_canonico', 'hex'),
      decode(:'rec_agregado_canonico', 'hex'),
      :'rec_manifiesto'::jsonb,
      decode(:'rec_contenido_manifiesto', 'hex'),
      decode(:'rec_representacion_manifiesto', 'hex'),
      decode(:'rec_preimagen_manifiesto', 'hex'),
      :'rec_huella_prevalidacion_entrada'
  ) AS c
\gset
\if :recuperacion_confirmacion_valida
\else
    \quit 7
\endif
RESET ROLE;
