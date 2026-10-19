\set ON_ERROR_STOP on
-- B144 (F1-60, primer corte): la bolsa guarda su ámbito organizativo y su
-- unidad gestora como dato gobernado de su constitución.
--
-- * ambito_bolsa: una fila inmutable por bolsa_ref con el par de ámbitos V3
--   (ambito_ref = organización o ámbito organizativo; unidad_ref = unidad
--   gestora), su origen y la referencia que lo acredita.
-- * confirmar_carga_convoca_v2 (B95) escribe ese par, el mismo que autorizó la
--   decisión V3 consumida, en la misma transacción que la constitución. Si la
--   bolsa ya tiene ámbito, una confirmación o recuperación desde otro ámbito se
--   deniega con 42501 antes de consumir. Una constitución nueva también se
--   deniega si la bolsa ya existía sin ámbito o si sustituiría (B14) una bolsa
--   vigente de su categoría con otro ámbito o sin él. Firma, propietario,
--   configuración y ACL se conservan; el cuerpo se sustituye solo si coincide
--   la preimagen.
-- * declarar_ambito_bolsa_v1: vía solo del propietario (sin concesiones) para
--   las bolsas constituidas antes de B144. Se usa con una semilla declarada; la
--   migración no rellena ninguna fila ni deduce el ámbito.
--
-- Los actos de Bolsa todavía no cotejan esta tabla (segundo corte de F1-60).
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000144',0));

DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        definicion text; fuente text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR f IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.recibo_carga_convoca') IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.sustitucion_bolsa') IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.ambito_bolsa') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(text,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'B144: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.pg_get_functiondef(p.oid),p.prosrc INTO definicion,fuente
 FROM pg_catalog.pg_proc p WHERE p.oid=f
  AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
  AND p.prosecdef AND p.provolatile='v'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s',
    'statement_timeout=15s','idle_in_transaction_session_timeout=20s'];
 IF definicion IS NULL OR fuente IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex')
    IS DISTINCT FROM '4d9b879dc67bfed72269ecdb6cced1288b94f5afc3e5aa02791530786a747313'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex')
    IS DISTINCT FROM 'db2bb4620a79e127bfc1c1751642c6f540b9cf7bf7a785764aec038c71be64f6'
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)
    IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 THEN RAISE EXCEPTION 'B144: definición o ACL B95 instalada divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.ambito_bolsa (
 bolsa_ref text PRIMARY KEY,
 ambito_ref text NOT NULL,
 unidad_ref text NOT NULL,
 origen text NOT NULL,
 fuente_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(bolsa_ref,512)),
 CHECK (pg_catalog.octet_length(ambito_ref) BETWEEN 1 AND 256 AND ambito_ref ~ '^[A-Za-z0-9:._/-]+$'),
 CHECK (pg_catalog.octet_length(unidad_ref) BETWEEN 1 AND 256 AND unidad_ref ~ '^[A-Za-z0-9:._/-]+$'),
 CHECK (origen IN ('constitucion','semilla_declarada')),
 CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(fuente_ref,512)),
 CHECK (origen<>'constitucion' OR fuente_ref ~ '^acta:importacion-convoca:[0-9a-f]{64}$')
);
COMMENT ON TABLE vec_bolsa_llamamientos.ambito_bolsa IS
 'B144: ámbito organizativo (ambito_ref) y unidad gestora (unidad_ref) de cada bolsa; inmutable.';
REVOKE ALL ON vec_bolsa_llamamientos.ambito_bolsa FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.ambito_bolsa FROM PUBLIC;
ALTER TABLE vec_bolsa_llamamientos.ambito_bolsa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.ambito_bolsa FORCE ROW LEVEL SECURITY;
CREATE POLICY ambito_solo_propietario ON vec_bolsa_llamamientos.ambito_bolsa
 FOR ALL TO vec_bolsa_llamamientos_propietario USING (true) WITH CHECK (true);
CREATE TRIGGER negar_mutacion BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.ambito_bolsa
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER negar_truncado BEFORE TRUNCATE ON vec_bolsa_llamamientos.ambito_bolsa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Declaración de una bolsa constituida antes de B144. Solo el propietario la
-- ejecuta (despliegue o copia local con una semilla declarada). Idempotente:
-- el mismo par devuelve la fila existente; otro par da 23505 sin escribir.
CREATE FUNCTION vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(
 p_bolsa_ref text,p_ambito_ref text,p_unidad_ref text,p_declaracion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE
 SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE previo vec_bolsa_llamamientos.ambito_bolsa%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'B144: declaración no autorizada' USING ERRCODE='42501'; END IF;
 IF NOT coalesce(vec_bolsa_llamamientos.constitucion_texto_valido(p_bolsa_ref,512),false)
 OR p_ambito_ref IS NULL OR pg_catalog.octet_length(p_ambito_ref) NOT BETWEEN 1 AND 256
 OR p_ambito_ref !~ '^[A-Za-z0-9:._/-]+$'
 OR p_unidad_ref IS NULL OR pg_catalog.octet_length(p_unidad_ref) NOT BETWEEN 1 AND 256
 OR p_unidad_ref !~ '^[A-Za-z0-9:._/-]+$'
 OR p_declaracion_ref IS NULL OR p_declaracion_ref !~ '^semilla:[a-z0-9][a-z0-9:._-]{2,120}$'
 THEN RAISE EXCEPTION 'B144: declaración de ámbito inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:ambito_bolsa:'||p_bolsa_ref,0));
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B144: bolsa no constituida' USING ERRCODE='22023'; END IF;
 SELECT * INTO previo FROM vec_bolsa_llamamientos.ambito_bolsa WHERE bolsa_ref=p_bolsa_ref;
 IF NOT FOUND THEN
  INSERT INTO vec_bolsa_llamamientos.ambito_bolsa(bolsa_ref,ambito_ref,unidad_ref,origen,fuente_ref)
  VALUES(p_bolsa_ref,p_ambito_ref,p_unidad_ref,'semilla_declarada',p_declaracion_ref)
  RETURNING * INTO previo;
  RETURN pg_catalog.jsonb_build_object('bolsa_ref',previo.bolsa_ref,'ambito_ref',previo.ambito_ref,
   'unidad_ref',previo.unidad_ref,'origen',previo.origen,'reutilizada',false);
 END IF;
 IF previo.ambito_ref<>p_ambito_ref OR previo.unidad_ref<>p_unidad_ref THEN
  RAISE EXCEPTION 'B144: la bolsa ya tiene otro ámbito' USING ERRCODE='23505'; END IF;
 RETURN pg_catalog.jsonb_build_object('bolsa_ref',previo.bolsa_ref,'ambito_ref',previo.ambito_ref,
  'unidad_ref',previo.unidad_ref,'origen',previo.origen,'reutilizada',true);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(text,text,text,text) FROM PUBLIC;

-- Definición final explícita de B95 con las dos comprobaciones de B144.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v2(p_acta jsonb, p_filas_cifradas jsonb, p_original_cifrado jsonb, p_acta_ref text, p_actor_ref text, p_categoria_ref text, p_bolsa_ref text, p_version_bolsa bigint, p_bolsa_canonica bytea, p_vigente_desde timestamp with time zone, p_instantanea_ref text, p_version_instantanea bigint, p_instantanea_canonica bytea, p_referida_en timestamp with time zone, p_generada_en timestamp with time zone, p_entradas jsonb, p_confirmada_en timestamp with time zone, p_vinculos jsonb, p_contexto_recurso bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
DECLARE decision jsonb; bolsa jsonb; recurso jsonb; capacidad jsonb; consumo record; estado jsonb; acta_durable jsonb;
        carga record; recibo jsonb; vinculos_recibo jsonb; previo vec_bolsa_llamamientos.recibo_carga_convoca%ROWTYPE;
        huella_entradas text; huella_vinculos text; nueva boolean;
        ambito text; unidad text; canon text; huella_contexto text;
        registrada_db timestamptz; registrada_texto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.jsonb_typeof(p_acta) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_filas_cifradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_original_cifrado) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_vinculos) IS DISTINCT FROM 'array'
 OR p_acta_ref IS NULL OR p_acta_ref !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256
 OR p_actor_ref<>pg_catalog.btrim(p_actor_ref)
 OR p_acta->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR p_acta->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR p_acta->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR p_acta->>'actor_ref' IS DISTINCT FROM 'actor:rrhh:'||pg_catalog.substr(
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||p_actor_ref,'UTF8')),'hex'),1,32)
 OR p_acta->>'huella_fichero_sha256' IS NULL
 OR p_acta->>'huella_fichero_sha256' !~ '^[0-9a-f]{64}$'
 OR p_acta_ref IS DISTINCT FROM 'acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to((p_acta->>'huella_fichero_sha256')||pg_catalog.chr(31)||p_categoria_ref,'UTF8')),'hex')
 OR p_contexto_recurso IS NULL OR pg_catalog.octet_length(p_contexto_recurso) NOT BETWEEN 1 AND 1024
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN
  decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  bolsa:=pg_catalog.convert_from(p_bolsa_canonica,'UTF8')::jsonb;
  recurso:=pg_catalog.convert_from(p_contexto_recurso,'UTF8')::jsonb;
  capacidad:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B79: material inválido' USING ERRCODE='42501'; END;
 IF pg_catalog.jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR recurso->'atributos' IS DISTINCT FROM '{}'::jsonb
 OR pg_catalog.jsonb_typeof(capacidad) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B95: contexto de confirmación inválido' USING ERRCODE='42501'; END IF;
 ambito:=recurso#>>'{ambitos,ambito_ref}';
 unidad:=recurso#>>'{ambitos,unidad_ref}';
 IF ambito IS NULL OR pg_catalog.octet_length(ambito) NOT BETWEEN 1 AND 256
 OR ambito !~ '^[A-Za-z0-9:._/-]+$'
 OR unidad IS NULL OR pg_catalog.octet_length(unidad) NOT BETWEEN 1 AND 256
 OR unidad !~ '^[A-Za-z0-9:._/-]+$'
 THEN RAISE EXCEPTION 'B95: ámbito de confirmación inválido' USING ERRCODE='42501'; END IF;
 -- La preimagen V3 usa json.Marshal Go sobre dos mapas. La igualdad exacta
 -- de bytes impide que un contexto de vista previa (fase/filtro/página) autorice
 -- la confirmación y rechaza claves duplicadas o codificaciones alternativas.
 canon:='{"ambitos":{"ambito_ref":'||pg_catalog.to_json(ambito)::text||
  ',"unidad_ref":'||pg_catalog.to_json(unidad)::text||'},"atributos":{}}';
 IF p_contexto_recurso IS DISTINCT FROM pg_catalog.convert_to(canon,'UTF8')
 THEN RAISE EXCEPTION 'B95: contexto de confirmación no canónico' USING ERRCODE='42501'; END IF;
 huella_contexto:=pg_catalog.encode(pg_catalog.sha256(p_contexto_recurso),'hex');
 IF decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella_contexto
 OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM huella_contexto
 OR capacidad->>'efecto_ref' IS DISTINCT FROM p_acta_ref
 THEN RAISE EXCEPTION 'B95: contexto y decisión divergentes' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_typeof(decision) IS DISTINCT FROM 'object'
 OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR decision->>'recurso_ref' IS DISTINCT FROM p_acta_ref
 OR pg_catalog.jsonb_typeof(bolsa) IS DISTINCT FROM 'object'
 OR bolsa->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR bolsa->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR bolsa->>'huella_listado_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_array_length(p_entradas)<1
 OR pg_catalog.jsonb_array_length(p_vinculos)>pg_catalog.jsonb_array_length(p_entradas)
 OR pg_catalog.jsonb_array_length(p_entradas)<>pg_catalog.jsonb_array_length(p_filas_cifradas)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_entradas) e
     WHERE pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'object'
        OR e.value->>'participacion_ref' IS NULL OR e.value->>'fila_numero' IS NULL)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_vinculos) v
     WHERE pg_catalog.jsonb_typeof(v.value) IS DISTINCT FROM 'object'
        OR v.value->>'participacion_ref' IS NULL OR v.value->>'candidato_ref' IS NULL)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'fila_numero')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT v.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_vinculos) v)<>pg_catalog.jsonb_array_length(p_vinculos)
 OR EXISTS (
   SELECT v.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_vinculos) v
   EXCEPT SELECT e.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_entradas) e)
 OR EXISTS (
   (SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e
    EXCEPT SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f)
   UNION ALL
   (SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f
    EXCEPT SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e))
 THEN RAISE EXCEPTION 'B79: filas y vínculos incompatibles' USING ERRCODE='22023'; END IF;
 huella_entradas:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_entradas::text,'UTF8')),'hex');
 huella_vinculos:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_vinculos::text,'UTF8')),'hex');
 -- B144: el ámbito de la bolsa se coteja antes de consumir la decisión y de
 -- escribir, con el mismo cerrojo que declarar_ambito_bolsa_v1.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:ambito_bolsa:'||p_bolsa_ref,0));
 -- Una bolsa con ámbito solo se confirma o recupera desde ese mismo ámbito.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref
   AND (a.ambito_ref IS DISTINCT FROM ambito OR a.unidad_ref IS DISTINCT FROM unidad))
 -- Una constitución nueva no da ámbito a una bolsa que ya existía sin él, ni
 -- sustituye (B14, por categoría) una bolsa vigente de otro ámbito o sin él.
 OR (NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.recibo_carga_convoca r WHERE r.acta_ref=p_acta_ref)
  AND (
   (EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=p_bolsa_ref)
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref))
   OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
     WHERE c.categoria_ref=p_categoria_ref AND c.bolsa_ref<>p_bolsa_ref
       AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
         WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
           AND s.huella_sustituida_sha256=c.huella_bolsa_sha256)
       AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a
         WHERE a.bolsa_ref=c.bolsa_ref AND a.ambito_ref=ambito AND a.unidad_ref=unidad))))
 THEN RAISE EXCEPTION 'B144: bolsa de otro ámbito' USING ERRCODE='42501'; END IF;
 -- La fachada AD218 exige consumo_nuevo=true; aquí no existe acceso por mera
 -- posesión de una referencia histórica. El reintento recibe otra decisión.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_acta_ref
 OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL OR consumo.consumida_en IS NULL
 THEN RAISE EXCEPTION 'B79: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Los locks internos de guardar_lote y B7 siguen el mismo orden en todos los
 -- intentos. En el replay se pasa el acta histórica para no cambiar custodia.
 estado:=vec_bolsa_importacion_convoca.consultar_estado_v1(
  p_acta->>'huella_fichero_sha256',p_categoria_ref);
 acta_durable:=COALESCE(estado->'acta',p_acta);
 IF acta_durable->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR acta_durable->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR acta_durable->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR acta_durable->>'actor_ref' IS DISTINCT FROM p_acta->>'actor_ref'
 OR acta_durable->>'nombre_fichero' IS DISTINCT FROM p_acta->>'nombre_fichero'
 OR acta_durable->>'fichero_custodiado_ref' IS DISTINCT FROM p_acta->>'fichero_custodiado_ref'
 OR acta_durable->>'huella_fichero_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: acta histórica incompatible' USING ERRCODE='23505'; END IF;
 SELECT * INTO STRICT carga FROM vec_bolsa_importacion_convoca.guardar_lote_v1(acta_durable,p_filas_cifradas);
 -- BIC3 fija registrada_en con el reloj PostgreSQL al crear el lote; Go
 -- aporta una fecha técnica previa. Todas las demás claves del acta deben
 -- permanecer idénticas, también actor, categoría, custodia y demás fechas.
 IF pg_catalog.jsonb_typeof(carga.acta_canonica) IS DISTINCT FROM 'object'
 OR carga.acta_canonica - 'registrada_en' IS DISTINCT FROM acta_durable - 'registrada_en'
 OR pg_catalog.jsonb_typeof(carga.acta_canonica->'registrada_en') IS DISTINCT FROM 'string'
 THEN
  RAISE EXCEPTION 'B79: acta durable divergente' USING ERRCODE='55000'; END IF;
 registrada_texto:=carga.acta_canonica->>'registrada_en';
 IF registrada_texto !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
 THEN RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000'; END IF;
 BEGIN
  registrada_db:=registrada_texto::timestamptz;
 EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow OR invalid_text_representation THEN
  RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000';
 END;
 IF NOT pg_catalog.isfinite(registrada_db)
 OR pg_catalog.to_char(registrada_db AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM registrada_texto
 THEN RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000'; END IF;
 PERFORM vec_bolsa_importacion_convoca.guardar_original_v1(carga.acta_canonica,p_original_cifrado);
 SELECT * INTO previo FROM vec_bolsa_llamamientos.recibo_carga_convoca WHERE acta_ref=p_acta_ref FOR SHARE;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor_ref OR previo.categoria_ref<>p_categoria_ref OR previo.bolsa_ref<>p_bolsa_ref
  OR previo.huella_fichero_sha256<>p_acta->>'huella_fichero_sha256'
  OR previo.recibo_constitucion->>'version_bolsa' IS DISTINCT FROM p_version_bolsa::text
  OR previo.recibo_constitucion->>'instantanea_ref' IS DISTINCT FROM p_instantanea_ref
  OR previo.recibo_constitucion->>'version_instantanea' IS DISTINCT FROM p_version_instantanea::text
  OR previo.huella_entradas_sha256<>huella_entradas OR previo.huella_vinculos_sha256<>huella_vinculos
  THEN RAISE EXCEPTION 'B79: reintento incompatible' USING ERRCODE='23505'; END IF;
  RETURN previo.recibo_constitucion||pg_catalog.jsonb_build_object(
   'reutilizada',true,
   'decision_ref',previo.decision_ref,'auditoria_ref',previo.auditoria_ref,
   'consumida_en',pg_catalog.to_char(previo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'auditoria_acceso_ref',consumo.auditoria_ref,
   'decision_acceso_ref',consumo.decision_ref,
   'acta_reutilizada',true,'vinculos',previo.recibo_vinculos);
 END IF;
 recibo:=vec_bolsa_llamamientos.constituir_bolsa_v1(
  p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_version_bolsa,p_bolsa_canonica,p_vigente_desde,
  p_instantanea_ref,p_version_instantanea,p_instantanea_canonica,p_referida_en,p_generada_en,p_entradas,p_confirmada_en);
 IF pg_catalog.jsonb_typeof(recibo) IS DISTINCT FROM 'object'
 OR recibo->>'acta_ref' IS DISTINCT FROM p_acta_ref OR recibo->>'reutilizada' IS DISTINCT FROM 'false'
 THEN RAISE EXCEPTION 'B79: constitución previa sin recibo B1' USING ERRCODE='23505'; END IF;
 -- B144: el ámbito autorizado de la constitución queda como dato gobernado de
 -- la bolsa en la misma transacción. Si ya existía, debe coincidir.
 INSERT INTO vec_bolsa_llamamientos.ambito_bolsa(bolsa_ref,ambito_ref,unidad_ref,origen,fuente_ref)
 VALUES(p_bolsa_ref,ambito,unidad,'constitucion',p_acta_ref) ON CONFLICT (bolsa_ref) DO NOTHING;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref
   AND a.ambito_ref=ambito AND a.unidad_ref=unidad)
 THEN RAISE EXCEPTION 'B144: bolsa de otro ámbito' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_array_length(p_vinculos)>0 THEN
  vinculos_recibo:=vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(p_acta_ref,p_vinculos,p_confirmada_en);
 ELSE
  vinculos_recibo:=pg_catalog.jsonb_build_object('nuevos',0,'existentes',0);
 END IF;
 IF pg_catalog.jsonb_typeof(vinculos_recibo) IS DISTINCT FROM 'object'
 OR (vinculos_recibo->>'nuevos')::integer IS DISTINCT FROM pg_catalog.jsonb_array_length(p_vinculos)
 OR (vinculos_recibo->>'existentes')::integer IS DISTINCT FROM 0
 THEN RAISE EXCEPTION 'B79: vínculos divergentes' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.recibo_carga_convoca(
  acta_ref,actor_ref,categoria_ref,bolsa_ref,huella_fichero_sha256,
  huella_entradas_sha256,huella_vinculos_sha256,recibo_constitucion,recibo_vinculos,
  decision_ref,auditoria_ref,consumida_en)
 VALUES(p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_acta->>'huella_fichero_sha256',
  huella_entradas,huella_vinculos,recibo,vinculos_recibo,
  consumo.decision_ref,consumo.auditoria_ref,consumo.consumida_en);
 RETURN recibo||pg_catalog.jsonb_build_object(
  'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumida_en',pg_catalog.to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'auditoria_acceso_ref',consumo.auditoria_ref,'decision_acceso_ref',consumo.decision_ref,
  'acta_reutilizada',carga.reutilizada,'vinculos',vinculos_recibo);
END $function$;

DO $post$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        d oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(text,text,text,text)');
        definicion text;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(p.oid) INTO definicion
 FROM pg_catalog.pg_proc p WHERE p.oid=f
  AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
  AND p.prosecdef AND p.provolatile='v'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s',
    'statement_timeout=15s','idle_in_transaction_session_timeout=20s'];
 IF definicion IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex')
    IS DISTINCT FROM '03cf6fa9b2e2b46f6e50fb05be177ae2b7d5748fbae8b08bdd7e068733fb9fc0'
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)
    IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR d IS NULL
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=d AND a.grantee<>p.proowner)
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
     WHERE c.oid='vec_bolsa_llamamientos.ambito_bolsa'::pg_catalog.regclass AND a.grantee<>c.relowner)
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
     WHERE c.oid='vec_bolsa_llamamientos.ambito_bolsa'::pg_catalog.regclass
       AND c.relowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
       AND c.relrowsecurity AND c.relforcerowsecurity)
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
     WHERE t.oid='vec_bolsa_llamamientos.ambito_bolsa'::pg_catalog.regtype AND a.grantee<>t.typowner)
 OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa)
 THEN RAISE EXCEPTION 'B144: posimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
