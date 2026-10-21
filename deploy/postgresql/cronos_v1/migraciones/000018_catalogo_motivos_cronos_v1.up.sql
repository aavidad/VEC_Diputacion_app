\set ON_ERROR_STOP on
-- CRN18. Catálogo sintético de motivos de Cronos para la candidata.
-- Publicación por la función nominal V2. No expresa aprobación ni regla laboral.
-- Requiere la proyección de motivos V2 instalada; no modifica su definición.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:migracion:000018:motivos',0));

DO $publicar$
<<publicar>>
DECLARE
  catalogo_id constant text := 'motivos_cronos_v1';
  catalogo_version constant integer := 1;
  catalogo_huella constant text := '82136314c0c8b9946b49debcb4497b62cbe9be7e0501c92dc4278ef528f5a822';
  evento_ref constant text := 'evento_6951212ecd41820b8cfa930ec471915e';
  evento_huella constant text := 'ba7b420e92c80ffb1ac4db111846b3fbf52a1126661de8633e5794a9e78cfe3e';
  publicado_en constant timestamptz := '2026-10-14T00:00:00.000000Z';
  entradas constant jsonb := '[
    {"clave":"motivo_a28bf9dd817c4ef9ccba903beb358c3f","vigente_desde":"2026-10-01T00:00:00.000000Z","vigente_hasta":null},
    {"clave":"motivo_1de68e77a196b06508201b9753248d16","vigente_desde":"2026-10-01T00:00:00.000000Z","vigente_hasta":null},
    {"clave":"motivo_0dc9dd0b008c4ebf09a65d5a64563693","vigente_desde":"2026-10-01T00:00:00.000000Z","vigente_hasta":null}
  ]';
  firma constant text := 'vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamp with time zone,jsonb)';
  secuencia bigint;
  publicada boolean;
  entradas_actuales jsonb;
  entradas_esperadas jsonb;
  evento_actual record;
BEGIN
  IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
     OR current_user <> 'vec_autorizacion_propietario'
     OR to_regprocedure(firma) IS NULL
     OR to_regclass('vec_autorizacion.motivo_v2_checkpoint_origen') IS NULL
     OR to_regclass('vec_autorizacion.motivo_v2_evento_origen') IS NULL
     OR to_regclass('vec_autorizacion.motivo_v2_catalogo_publicado') IS NULL
     OR to_regclass('vec_autorizacion.motivo_v2_entrada') IS NULL THEN
    RAISE EXCEPTION 'CRN18: preimagen de motivos V2 incompatible' USING ERRCODE='55000';
  END IF;
  IF (SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
        FROM pg_proc p WHERE p.oid=to_regprocedure(firma))
       IS DISTINCT FROM '9c75498d3ef5cbe073430868a1cfded66f16956678040e210b5e5d368b2ff02f'
     OR (SELECT p.proowner FROM pg_proc p WHERE p.oid=to_regprocedure(firma))
        IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
     OR (SELECT p.prosecdef FROM pg_proc p WHERE p.oid=to_regprocedure(firma)) IS DISTINCT FROM true
     OR (SELECT p.proconfig FROM pg_proc p WHERE p.oid=to_regprocedure(firma))
        IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
     OR NOT has_function_privilege('vec_autorizacion_propietario',to_regprocedure(firma),'EXECUTE')
     OR NOT vec_autorizacion.motivo_v2_entradas_validas(entradas) THEN
    RAISE EXCEPTION 'CRN18: publicador nominal incompatible' USING ERRCODE='55000';
  END IF;
  IF publicado_en > clock_timestamp() THEN
    RAISE EXCEPTION 'CRN18: fecha de publicación futura' USING ERRCODE='55000';
  END IF;

  -- La función bloquea el checkpoint y comprueba la secuencia. Esta lectura
  -- conserva la misma transacción y evita reservar un número fijo global.
  SELECT ultima_secuencia + 1 INTO STRICT secuencia
    FROM vec_autorizacion.motivo_v2_checkpoint_origen
   WHERE control_id = true FOR UPDATE;
  SELECT e.secuencia_origen, e.huella_evento_sha256, e.catalogo_id,
         e.catalogo_version, e.tipo_evento
    INTO evento_actual
    FROM vec_autorizacion.motivo_v2_evento_origen e
   WHERE e.evento_origen_ref = evento_ref;
  IF FOUND THEN
    IF evento_actual.huella_evento_sha256 IS DISTINCT FROM evento_huella
       OR evento_actual.catalogo_id IS DISTINCT FROM catalogo_id
       OR evento_actual.catalogo_version IS DISTINCT FROM catalogo_version
       OR evento_actual.tipo_evento IS DISTINCT FROM 'publicacion' THEN
      RAISE EXCEPTION 'CRN18: referencia de evento ya usada con otro contenido' USING ERRCODE='55000';
    END IF;
    secuencia := evento_actual.secuencia_origen;
  END IF;

  publicada := vec_autorizacion.publicar_motivos_autorizacion_v2(
    evento_ref, secuencia, evento_huella, catalogo_id, catalogo_version,
    catalogo_huella, publicado_en, entradas);
  IF publicada IS DISTINCT FROM true THEN
    RAISE EXCEPTION 'CRN18: publicación rechazada por contrato V2' USING ERRCODE='55000';
  END IF;

  SELECT jsonb_agg(value ORDER BY value->>'clave') INTO entradas_esperadas
    FROM jsonb_array_elements(entradas);
  SELECT jsonb_agg(jsonb_build_object(
      'clave',entrada_clave,
      'vigente_desde',to_char(vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'vigente_hasta',CASE WHEN vigente_hasta IS NULL THEN 'null'::jsonb
        ELSE to_jsonb(to_char(vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
      ORDER BY entrada_clave)
    INTO entradas_actuales
    FROM vec_autorizacion.motivo_v2_entrada e
   WHERE e.catalogo_id=publicar.catalogo_id AND e.catalogo_version=publicar.catalogo_version;
  IF entradas_actuales IS DISTINCT FROM entradas_esperadas
     OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado c
          WHERE c.catalogo_id=publicar.catalogo_id AND c.catalogo_version=publicar.catalogo_version
            AND c.catalogo_huella_publicada_sha256=publicar.catalogo_huella
            AND c.publicado_en=publicar.publicado_en
            AND c.evento_origen_ref=publicar.evento_ref
            AND c.secuencia_origen=publicar.secuencia)
     OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
          WHERE r.catalogo_id=publicar.catalogo_id AND r.catalogo_version=publicar.catalogo_version) THEN
    RAISE EXCEPTION 'CRN18: posimagen del catálogo incompatible' USING ERRCODE='55000';
  END IF;
END $publicar$;
COMMIT;
