\set ON_ERROR_STOP on
-- Banco desechable propio. Una bolsa de proyección para resumen/lista global.
-- El detalle nominal de esta bolsa no se puede abrir: no existe lote CONVOCA
-- protegido en el recuperador. No modifica las 12 bolsas originales.
-- cargar.sh antepone dos cánones producidos por generar_bolsa_volumen.go con
-- NuevaBolsaConstituida, NuevaParticipacionBolsa y NuevaInstantaneaOrdenBolsa.
\if :{?bolsa_canon_b64}
\else
DO $missing$ BEGIN RAISE EXCEPTION 'falta canon de Bolsa generado'; END $missing$;
\endif
\if :{?instantanea_canon_b64}
\else
DO $missing$ BEGIN RAISE EXCEPTION 'falta canon de instantánea generado'; END $missing$;
\endif
BEGIN;
SET LOCAL session_replication_role = replica;
SET LOCAL statement_timeout = '120s';
DO $guard$
BEGIN
  IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida
             WHERE bolsa_ref = 'bolsa:rendimiento:20261012') THEN
    RAISE EXCEPTION 'clave=bolsa_proyeccion_previa esperado=false observado=true';
  END IF;
  IF (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion_entrada) <> 390 THEN
    RAISE EXCEPTION 'clave=bolsa_entradas_previa esperado=390 observado=%', (SELECT count(*) FROM vec_bolsa_llamamientos.constitucion_entrada);
  END IF;
END $guard$;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida
 (bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,
  vigente_desde,vigente_hasta,estado,registrada_en)
WITH canon AS (
  SELECT decode(:'bolsa_canon_b64','base64') AS contenido
)
SELECT 'bolsa:rendimiento:20261012',1,encode(sha256(contenido),'hex'),
       contenido,'categoria:rpt:analista-programador',
       '2026-09-20T00:00:00Z'::timestamptz,NULL,'vigente',
       '2026-09-20T00:00:00Z'::timestamptz
FROM canon;
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
 (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
  bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
  referida_en,generada_en,registrada_en)
WITH canon AS (
  SELECT decode(:'instantanea_canon_b64','base64') AS contenido
)
SELECT 'instantanea:rendimiento:20261012',1,
       encode(sha256(c.contenido),'hex'),c.contenido,
       b.bolsa_ref,b.version,b.huella_bolsa_sha256,
       2000,'2026-10-12T00:00:00Z'::timestamptz,
       '2026-10-12T00:00:00Z'::timestamptz,
       '2026-10-12T00:00:00Z'::timestamptz
FROM vec_bolsa_llamamientos.bolsa_constituida b CROSS JOIN canon c
WHERE b.bolsa_ref='bolsa:rendimiento:20261012';
INSERT INTO vec_bolsa_llamamientos.constitucion
 (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
  instantanea_ref,version_instantanea,huella_instantanea_sha256,
  categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:rendimiento:20261012',b.bolsa_ref,b.version,
       b.huella_bolsa_sha256,i.instantanea_ref,i.version,
       i.huella_instantanea_sha256,b.categoria_ref,'actor:rendimiento:20261012',
       '2026-10-12T00:00:00Z'::timestamptz,
       '2026-10-12T00:00:00Z'::timestamptz
FROM vec_bolsa_llamamientos.bolsa_constituida b
JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
  ON i.bolsa_ref=b.bolsa_ref AND i.version_bolsa=b.version
WHERE b.bolsa_ref='bolsa:rendimiento:20261012';
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
 (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
SELECT 'instantanea:rendimiento:20261012',1,i,
       'participacion:rendimiento:' || lpad(i::text,5,'0'),i
FROM generate_series(1,2000) g(i);
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato
 (participacion_ref,candidato_ref,acta_ref,instantanea_ref,
  version_instantanea,registrada_en)
SELECT 'participacion:rendimiento:' || lpad(i::text,5,'0'),
       'can_' || lpad(i::text,22,'0'),'acta:rendimiento:20261012',
       'instantanea:rendimiento:20261012',1,
       '2026-10-12T00:00:00Z'::timestamptz
FROM generate_series(1,2000) g(i);
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa
 (politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,
  provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
SELECT 'politica:rendimiento:20261012','bolsa:rendimiento:20261012',1,
       criterio,tipo_lista,reposicion,provisional,rotulo,
       'actor:rendimiento:20261012',
       '2026-09-20T00:00:00Z'::timestamptz,NULL,
       '2026-09-20T00:00:00Z'::timestamptz
FROM vec_bolsa_llamamientos.politica_orden_bolsa
WHERE bolsa_ref <> 'bolsa:rendimiento:20261012'
ORDER BY registrada_en DESC LIMIT 1;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,hasta,fecha_disponible,
  motivo,actor,registrada_en,clave_idempotencia,recibo_ref,
  politica_transiciones_version)
SELECT 'participacion:rendimiento:' || lpad(i::text,5,'0'),
       CASE WHEN i % 100 < 76 THEN 'disponible'
            WHEN i % 100 < 84 THEN 'no_disponible'
            WHEN i % 100 < 90 THEN 'trabajando'
            WHEN i % 100 < 94 THEN 'pendiente_incorporacion'
            WHEN i % 100 < 97 THEN 'renuncia'
            WHEN i % 100 < 99 THEN 'disponible_desde'
            ELSE 'excluido' END,
       '2026-09-20T00:00:00Z'::timestamptz
          + (i%20)*interval '1 day',
       NULL,
       CASE WHEN i % 100 BETWEEN 97 AND 98
            THEN '2026-10-13T00:00:00Z'::timestamptz
                + (i%14)*interval '1 day'
            ELSE NULL END,
       'Situación sintética del banco de rendimiento',
       'actor:rendimiento:20261012',
       '2026-10-12T00:00:00Z'::timestamptz,
       'clave:rendimiento:' || lpad(i::text, 5, '0'),
       'recibo:rendimiento:' || lpad(i::text, 5, '0'),
       NULL
FROM generate_series(1,2000) g(i);
COMMIT;
ANALYZE vec_bolsa_llamamientos.bolsa_constituida;
ANALYZE vec_bolsa_llamamientos.instantanea_orden_bolsa;
ANALYZE vec_bolsa_llamamientos.constitucion;
ANALYZE vec_bolsa_llamamientos.constitucion_entrada;
ANALYZE vec_bolsa_llamamientos.vinculo_candidato;
ANALYZE vec_bolsa_llamamientos.situacion_participacion;
