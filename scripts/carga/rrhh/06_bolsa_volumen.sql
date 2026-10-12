\set ON_ERROR_STOP on
-- Banco desechable propio. Una bolsa de proyección para resumen/lista global.
-- El detalle nominal de esta bolsa no se puede abrir: no existe lote CONVOCA
-- protegido en el recuperador. Las 12 bolsas originales sí conservan su lote.
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
WITH personas AS (
  SELECT i,'participacion:rendimiento:' || lpad(i::text,5,'0') AS participacion_ref,
    (ARRAY['Ana','Lucía','María','Carmen','Elena','Paula','Isabel','Laura','Marta','Rocío',
           'Miguel','Antonio','José','Manuel','Juan','Daniel','Pablo','Javier','Sergio','David'])[1+(i%20)]
    || ' ' ||
    (ARRAY['García','López','Martín','Sánchez','Pérez','Gómez','Ruiz','Hernández','Jiménez','Díaz',
           'Moreno','Muñoz','Álvarez','Romero','Alonso','Gutiérrez','Navarro','Torres','Domínguez','Vázquez'])[1+((i/20)%20)]
    || ' ' ||
    (ARRAY['Castro','Ortega','Serrano','Medina','Molina','Delgado','Herrera','Ramos','Gil','Suárez'])[1+((i/400)%10)]
      AS nombre_sintetico
  FROM generate_series(1,2000) g(i)
), lote AS (
  SELECT convert_to(jsonb_build_object(
    'tipo','proyeccion_rendimiento',
    'bolsa_ref','bolsa:rendimiento:20261012',
    'categoria_ref','categoria:rpt:analista-programador',
    'participaciones',jsonb_agg(jsonb_build_object(
      'ref',p.participacion_ref,'nombre_sintetico',p.nombre_sintetico) ORDER BY p.i)
  )::text,'UTF8') AS bolsa_canonica FROM personas p
)
SELECT 'bolsa:rendimiento:20261012',1,encode(sha256(bolsa_canonica),'hex'),
       bolsa_canonica,'categoria:rpt:analista-programador',now()-interval '30 days',NULL,'vigente',
       now()-interval '30 days'
FROM lote;
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
 (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
  bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
  referida_en,generada_en,registrada_en)
WITH instantanea AS (
  SELECT convert_to(jsonb_build_object(
    'tipo','proyeccion_rendimiento',
    'instantanea_ref','instantanea:rendimiento:20261012',
    'orden',jsonb_agg(jsonb_build_object(
      'ref','participacion:rendimiento:' || lpad(i::text,5,'0'),
      'orden',i) ORDER BY i)
  )::text,'UTF8') AS instantanea_canonica
  FROM generate_series(1,2000) g(i)
)
SELECT 'instantanea:rendimiento:20261012',1,
       encode(sha256(i.instantanea_canonica),'hex'),i.instantanea_canonica,
       b.bolsa_ref,b.version,b.huella_bolsa_sha256,
       2000,now()-interval '30 days',now()-interval '30 days',
       now()-interval '30 days'
FROM vec_bolsa_llamamientos.bolsa_constituida b CROSS JOIN instantanea i
WHERE b.bolsa_ref='bolsa:rendimiento:20261012';
INSERT INTO vec_bolsa_llamamientos.constitucion
 (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
  instantanea_ref,version_instantanea,huella_instantanea_sha256,
  categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:rendimiento:20261012',b.bolsa_ref,b.version,
       b.huella_bolsa_sha256,i.instantanea_ref,i.version,
       i.huella_instantanea_sha256,b.categoria_ref,'actor:rendimiento:20261012',
       now()-interval '30 days',now()-interval '30 days'
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
       'instantanea:rendimiento:20261012',1,now()-interval '30 days'
FROM generate_series(1,2000) g(i);
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa
 (politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,
  provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
SELECT 'politica:rendimiento:20261012','bolsa:rendimiento:20261012',1,
       criterio,tipo_lista,reposicion,provisional,rotulo,
       'actor:rendimiento:20261012',now()-interval '30 days',NULL,
       now()-interval '30 days'
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
       date_trunc('microseconds',now()-(i%60+1)*interval '1 day'),
       NULL,
       CASE WHEN i % 100 BETWEEN 97 AND 98
            THEN date_trunc('microseconds',now()+(i%14+1)*interval '1 day')
            ELSE NULL END,
       'Situación sintética del banco de rendimiento',
       'actor:rendimiento:20261012',
       date_trunc('microseconds',now()),
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
