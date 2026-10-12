\set ON_ERROR_STOP on
-- Ensayo sintético B67/B68. Parámetros privados proceden del recibo de esta ejecución.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT set_config('vec.fixture.apertura', :'apertura', true);
SELECT set_config('vec.fixture.llamamiento', :'llamamiento', true);
SELECT set_config('vec.fixture.candidato', :'candidato', true);
-- La fuente sintética de Bolsa no pasó por constitución B7. Alcance local de
-- esta transacción; no cambia sesiones, funciones, ACL ni reglas de producto.
SET LOCAL session_replication_role=replica;
DO $fixture$
<<fixture_vars>>
DECLARE
  apertura_ref text:=current_setting('vec.fixture.apertura');
  llamamiento_ref text:=current_setting('vec.fixture.llamamiento');
  candidato_ref text:=current_setting('vec.fixture.candidato');
  acta_ref text:='acta:fixture:b2:'||md5(apertura_ref);
  apertura jsonb;
  apertura_json json;
  instantanea jsonb;
  propuesta jsonb;
  canonico text;
  huella text;
  bolsa_ref text;
  participacion_ref text;
  instante timestamptz;
  total integer;
BEGIN
 SELECT convert_from(registro_canonico,'UTF8')::jsonb,
        convert_from(registro_canonico,'UTF8')::json
   INTO STRICT apertura,apertura_json
   FROM vec_bolsa_llamamientos.integracion_desarrollo
  WHERE operacion_ref=apertura_ref AND tipo='propuesta';
 IF apertura->'llamamiento'->>'LlamamientoRef' IS DISTINCT FROM llamamiento_ref THEN
   RAISE EXCEPTION 'Fixture B67/B68: apertura no pertenece al llamamiento esperado';
 END IF;
 instantanea:=apertura->'instantanea'; propuesta:=apertura->'propuesta';
 canonico:=(apertura_json->'instantanea')::text;
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 bolsa_ref:=instantanea->>'bolsa_ref';
 participacion_ref:=propuesta->>'participacion_seleccionada_ref';
 instante:=(instantanea->>'generada_en')::timestamptz;
 total:=jsonb_array_length(instantanea->'entradas');
 IF bolsa_ref IS NULL OR participacion_ref IS NULL OR instante IS NULL OR total<1
    OR propuesta->>'orden_seleccionado' IS NULL THEN
   RAISE EXCEPTION 'Fixture B67/B68: apertura canónica incompleta';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo
     WHERE operacion_ref=apertura_ref AND tipo='propuesta') THEN
   RAISE EXCEPTION 'Fixture B67/B68: falta apertura original';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa
   (instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,
    bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,
    referida_en,generada_en,registrada_en)
 VALUES (instantanea->>'instantanea_ref',(instantanea->>'version')::bigint,
   huella,convert_to(canonico,'UTF8'),bolsa_ref,(instantanea->>'version_bolsa')::bigint,
   instantanea->>'huella_bolsa_sha256',total,
   (instantanea->>'referida_en')::timestamptz,instante,clock_timestamp())
 ON CONFLICT DO NOTHING;
 INSERT INTO vec_bolsa_llamamientos.constitucion
   (acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
    version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,
    confirmada_en,registrada_en)
 VALUES (acta_ref,bolsa_ref,(instantanea->>'version_bolsa')::bigint,
   instantanea->>'huella_bolsa_sha256',instantanea->>'instantanea_ref',
   (instantanea->>'version')::bigint,huella,apertura->>'categoria_ref',
   'actor:fixture:b2',instante,clock_timestamp())
 ON CONFLICT DO NOTHING;
 INSERT INTO vec_bolsa_llamamientos.constitucion_entrada
   (instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
 VALUES (instantanea->>'instantanea_ref',(instantanea->>'version')::bigint,
   (propuesta->>'orden_seleccionado')::bigint,participacion_ref,
   (propuesta->>'orden_seleccionado')::integer)
 ON CONFLICT DO NOTHING;
 INSERT INTO vec_bolsa_llamamientos.vinculo_candidato
   (participacion_ref,candidato_ref,acta_ref,instantanea_ref,
    version_instantanea,registrada_en)
 VALUES (participacion_ref,candidato_ref,acta_ref,instantanea->>'instantanea_ref',
   (instantanea->>'version')::bigint,clock_timestamp())
 ON CONFLICT DO NOTHING;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato v
      JOIN vec_bolsa_llamamientos.constitucion ct
        ON ct.acta_ref=v.acta_ref AND ct.instantanea_ref=v.instantanea_ref
       AND ct.version_instantanea=v.version_instantanea
      JOIN vec_bolsa_llamamientos.constitucion_entrada e
        ON e.instantanea_ref=v.instantanea_ref
       AND e.version_instantanea=v.version_instantanea
       AND e.participacion_ref=v.participacion_ref
      JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io
        ON io.instantanea_ref=ct.instantanea_ref
       AND io.version=ct.version_instantanea
       AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
     WHERE v.participacion_ref=fixture_vars.participacion_ref
       AND v.candidato_ref=fixture_vars.candidato_ref
       AND ct.acta_ref=fixture_vars.acta_ref AND ct.bolsa_ref=fixture_vars.bolsa_ref
       AND ct.categoria_ref=apertura->>'categoria_ref'
       AND convert_from(io.instantanea_canonica,'UTF8')=canonico)<>1 THEN
   RAISE EXCEPTION 'Fixture B67/B68: vínculo exacto no confirmado';
 END IF;
END $fixture$;
COMMIT;
