\set ON_ERROR_STOP on
-- CT-190. La captura nace del INSERT CT110 dentro de la transacción de publicación.
-- La historia anterior queda marcada expresamente como legado, sin regla atribuida.
-- No instalar antes de H6 ni activar la edición de ajustes antes del consumidor Go.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000190', 0));
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh IN SHARE MODE;
LOCK TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh IN SHARE MODE;
DO $pre$
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.regla_base_publicada_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.regla_base_activacion_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.fase_regla_instantanea_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.regla_ajuste_version_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
 THEN RAISE EXCEPTION 'CT-190: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Se publica con el artefacto canónico producido por el mismo dominio Go que
-- calcula HuellaSHA256. Cada versión permanece aunque otra se active después.
CREATE TABLE vec_contratacion_temporal.regla_base_publicada_v1 (
 catalogo_id text NOT NULL CHECK (catalogo_id='vec.contratacion_temporal.reglas'),
 version bigint NOT NULL CHECK (version BETWEEN 1 AND 9999999),
 huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
 canonico text NOT NULL CHECK (octet_length(canonico) BETWEEN 2 AND 262144),
 fuente_sha256 text NOT NULL CHECK (fuente_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK (char_length(aprobacion_ref) BETWEEN 1 AND 200
    AND aprobacion_ref=btrim(aprobacion_ref) AND aprobacion_ref !~ '[[:cntrl:]]'),
 publicada_por text NOT NULL DEFAULT session_user,
 publicada_en timestamptz(6) NOT NULL DEFAULT date_trunc('microseconds',clock_timestamp()),
 PRIMARY KEY (catalogo_id,version,huella_sha256),
 UNIQUE (catalogo_id,version),
 CHECK (huella_sha256=encode(sha256(convert_to(canonico,'UTF8')),'hex')),
 CHECK (canonico::jsonb->>'id'=catalogo_id),
 CHECK ((canonico::jsonb->>'version')::bigint=version),
 CHECK (canonico::jsonb->>'estado'='publicado'),
 CHECK (canonico::jsonb->>'aprobacion_ref'=aprobacion_ref)
);

-- Activar o desactivar no reescribe la base. La secuencia esperada es CAS;
-- el disparador comprueba la cabeza y la PK resuelve las carreras concurrentes.
CREATE TABLE vec_contratacion_temporal.regla_base_activacion_v1 (
 secuencia bigint PRIMARY KEY CHECK (secuencia BETWEEN 1 AND 9999999),
 secuencia_esperada bigint NOT NULL CHECK (secuencia_esperada=secuencia-1),
 activa boolean NOT NULL,
 catalogo_id text CHECK (catalogo_id='vec.contratacion_temporal.reglas'),
 version bigint,
 huella_sha256 text,
 aprobacion_ref text NOT NULL CHECK (char_length(aprobacion_ref) BETWEEN 1 AND 200
    AND aprobacion_ref=btrim(aprobacion_ref) AND aprobacion_ref !~ '[[:cntrl:]]'),
 activada_por text NOT NULL DEFAULT session_user,
 activada_en timestamptz(6) NOT NULL DEFAULT date_trunc('microseconds',clock_timestamp()),
 CHECK ((activa AND catalogo_id IS NOT NULL AND version IS NOT NULL AND huella_sha256 IS NOT NULL)
     OR (NOT activa AND catalogo_id IS NULL AND version IS NULL AND huella_sha256 IS NULL)),
 FOREIGN KEY (catalogo_id,version,huella_sha256)
  REFERENCES vec_contratacion_temporal.regla_base_publicada_v1(catalogo_id,version,huella_sha256)
);
CREATE FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' AS $f$
DECLARE v_cabeza bigint;
BEGIN
 IF TG_OP <> 'INSERT' OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
    OR TG_TABLE_NAME <> 'regla_base_activacion_v1'
    OR current_user <> 'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT-190: activación denegada' USING ERRCODE='42501'; END IF;
 SELECT coalesce(max(secuencia),0) INTO v_cabeza
 FROM vec_contratacion_temporal.regla_base_activacion_v1;
 IF NEW.secuencia=1 AND NOT NEW.activa THEN
  RAISE EXCEPTION 'CT-190: primera base debe estar activa' USING ERRCODE='55000';
 END IF;
 IF NEW.secuencia_esperada <> v_cabeza THEN
  RAISE EXCEPTION 'CT-190: cabeza de activación distinta' USING ERRCODE='40001';
 END IF;
 IF NEW.activa AND NOT EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.regla_base_publicada_v1 b
  WHERE b.catalogo_id=NEW.catalogo_id AND b.version=NEW.version
    AND b.huella_sha256=NEW.huella_sha256 AND b.aprobacion_ref=NEW.aprobacion_ref
 ) THEN
  RAISE EXCEPTION 'CT-190: activación sin base aprobada' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER regla_base_activacion_cas BEFORE INSERT
ON vec_contratacion_temporal.regla_base_activacion_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1();

-- La primera activación fija una base de transición para las 170 entradas
-- anteriores. No se atribuye retrospectivamente como regla de su origen.
CREATE TABLE vec_contratacion_temporal.regla_legado_transicion_v1 (
 unico boolean PRIMARY KEY DEFAULT true CHECK (unico),
 base_catalogo_id text NOT NULL,
 base_version bigint NOT NULL,
 base_huella_sha256 text NOT NULL,
 ajustes_catalogo_id text NOT NULL CHECK (ajustes_catalogo_id='vec.contratacion_temporal.reglas.ajustes'),
 ajustes_encontrados boolean NOT NULL,
 ajustes_version bigint NOT NULL,
 ajustes_huella_sha256 text NOT NULL,
 ajustes_canonico text NOT NULL,
 ajustes_vigente_desde timestamptz(6),
 capturada_en timestamptz(6) NOT NULL,
 FOREIGN KEY (base_catalogo_id,base_version,base_huella_sha256)
  REFERENCES vec_contratacion_temporal.regla_base_publicada_v1(catalogo_id,version,huella_sha256),
 CHECK (ajustes_huella_sha256=encode(sha256(convert_to(ajustes_canonico,'UTF8')),'hex')),
 CHECK ((ajustes_encontrados AND ajustes_version>0 AND ajustes_vigente_desde IS NOT NULL)
  OR (NOT ajustes_encontrados AND ajustes_version=0 AND ajustes_vigente_desde IS NULL
   AND ajustes_canonico='{}'))
);
COMMENT ON TABLE vec_contratacion_temporal.regla_legado_transicion_v1 IS
'Base congelada en la primera activación CT190 para conservar cálculos existentes; no acredita qué regla originó un tramo histórico.';
CREATE FUNCTION vec_contratacion_temporal.congelar_regla_legado_transicion_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET "TimeZone"='UTC' AS $function$
DECLARE v_ahora timestamptz(6);
BEGIN
 IF TG_OP<>'INSERT' OR TG_TABLE_SCHEMA<>'vec_contratacion_temporal'
  OR TG_TABLE_NAME<>'regla_base_activacion_v1' OR current_user<>'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'transición CT190 denegada' USING ERRCODE='42501'; END IF;
 IF NEW.secuencia<>1 OR NOT NEW.activa OR EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.regla_legado_transicion_v1) THEN
  RETURN NULL;
 END IF;
 -- CT148 y CT191 usan este mismo advisory lock. El bloqueo de tabla cubre
 -- también escrituras ajenas al protocolo y fija la ausencia de ajustes.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:ajustes_reglas',0));
 LOCK TABLE vec_contratacion_temporal.regla_ajuste_version_v1 IN SHARE MODE;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1) THEN
  RAISE EXCEPTION 'CT-190: legado con ajustes previos sin baseline aprobado' USING ERRCODE='55000';
 END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_contratacion_temporal.regla_legado_transicion_v1
  (base_catalogo_id,base_version,base_huella_sha256,ajustes_catalogo_id,
   ajustes_encontrados,ajustes_version,ajustes_huella_sha256,ajustes_canonico,
   ajustes_vigente_desde,capturada_en)
 VALUES (NEW.catalogo_id,NEW.version,NEW.huella_sha256,
  'vec.contratacion_temporal.reglas.ajustes',false,0,
  encode(sha256(convert_to('{}','UTF8')),'hex'),'{}',NULL,v_ahora);
 RETURN NULL;
END $function$;
CREATE TRIGGER regla_legado_transicion_primera_activacion AFTER INSERT
ON vec_contratacion_temporal.regla_base_activacion_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.congelar_regla_legado_transicion_v1();

CREATE TABLE vec_contratacion_temporal.fase_regla_instantanea_v1 (
 expediente_ref text NOT NULL,
 version numeric(20,0) NOT NULL,
 fase_clave text NOT NULL,
 fase_desde timestamptz(6) NOT NULL,
 estado text NOT NULL CHECK (estado IN ('capturada','legado_sin_instantanea')),
 base_catalogo_id text,
 base_version bigint,
 base_huella_sha256 text,
 ajustes_catalogo_id text,
 ajustes_encontrados boolean,
 ajustes_version bigint,
 ajustes_huella_sha256 text,
 ajustes_canonico text,
 ajustes_vigente_desde timestamptz(6),
 capturada_en timestamptz(6),
 PRIMARY KEY (expediente_ref,version),
 FOREIGN KEY (expediente_ref,version)
  REFERENCES vec_contratacion_temporal.fase_entrada_publicacion_rrhh(expediente_ref,version),
 FOREIGN KEY (base_catalogo_id,base_version,base_huella_sha256)
  REFERENCES vec_contratacion_temporal.regla_base_publicada_v1(catalogo_id,version,huella_sha256),
 CHECK ((estado='legado_sin_instantanea' AND base_catalogo_id IS NULL AND base_version IS NULL
         AND base_huella_sha256 IS NULL AND ajustes_catalogo_id IS NULL AND ajustes_encontrados IS NULL
         AND ajustes_version IS NULL AND ajustes_huella_sha256 IS NULL AND ajustes_canonico IS NULL
         AND ajustes_vigente_desde IS NULL AND capturada_en IS NULL)
     OR (estado='capturada' AND base_catalogo_id IS NOT NULL AND base_version IS NOT NULL
         AND base_huella_sha256 IS NOT NULL AND ajustes_catalogo_id='vec.contratacion_temporal.reglas.ajustes'
         AND ajustes_encontrados IS NOT NULL AND ajustes_version IS NOT NULL
         AND ajustes_huella_sha256 IS NOT NULL AND ajustes_canonico IS NOT NULL AND capturada_en IS NOT NULL)),
 CHECK (estado<>'capturada' OR
        (octet_length(ajustes_canonico)<=16384 AND
         ajustes_huella_sha256=encode(sha256(convert_to(ajustes_canonico,'UTF8')),'hex') AND
         ((ajustes_encontrados AND ajustes_version>0 AND ajustes_vigente_desde IS NOT NULL)
          OR (NOT ajustes_encontrados AND ajustes_version=0 AND ajustes_vigente_desde IS NULL
              AND ajustes_canonico='{}'))))
);
COMMENT ON TABLE vec_contratacion_temporal.fase_regla_instantanea_v1 IS
'Instantánea del catálogo base activo y cabeza CT148 visible al abrir el tramo; versiones posteriores de la misma fase heredan la captura. Legados explícitos sin atribución retroactiva.';

INSERT INTO vec_contratacion_temporal.fase_regla_instantanea_v1
 (expediente_ref,version,fase_clave,fase_desde,estado)
SELECT expediente_ref,version,fase_clave,fase_desde,'legado_sin_instantanea'
FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh;

CREATE FUNCTION vec_contratacion_temporal.registrar_instantanea_fase_regla_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC' AS $f$
DECLARE anterior record; captura record; v_capturada_en timestamptz(6);
BEGIN
 IF TG_OP <> 'INSERT' OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
    OR TG_TABLE_NAME <> 'fase_entrada_publicacion_rrhh'
    OR current_user <> 'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT-190: registro denegado' USING ERRCODE='42501'; END IF;
 IF NEW.version>1 THEN
  SELECT * INTO STRICT anterior FROM vec_contratacion_temporal.fase_regla_instantanea_v1
   WHERE expediente_ref=NEW.expediente_ref AND version=NEW.version-1;
  IF anterior.fase_clave=NEW.fase_clave AND anterior.fase_desde=NEW.fase_desde THEN
   INSERT INTO vec_contratacion_temporal.fase_regla_instantanea_v1
    (expediente_ref,version,fase_clave,fase_desde,estado,base_catalogo_id,base_version,
     base_huella_sha256,ajustes_catalogo_id,ajustes_encontrados,ajustes_version,
     ajustes_huella_sha256,ajustes_canonico,ajustes_vigente_desde,capturada_en)
   VALUES (NEW.expediente_ref,NEW.version,NEW.fase_clave,NEW.fase_desde,anterior.estado,
     anterior.base_catalogo_id,anterior.base_version,anterior.base_huella_sha256,
     anterior.ajustes_catalogo_id,anterior.ajustes_encontrados,anterior.ajustes_version,
     anterior.ajustes_huella_sha256,anterior.ajustes_canonico,anterior.ajustes_vigente_desde,
     anterior.capturada_en);
   RETURN NULL;
  END IF;
 END IF;
 v_capturada_en:=date_trunc('microseconds',clock_timestamp());
 -- Una única sentencia MVCC fija base activa y ajuste efectivo ya visible.
 -- La ausencia de CT148 se conserva como versión cero y huella de {}.
 SELECT b.catalogo_id,b.version,b.huella_sha256,
        j.version AS ajustes_version,j.huella_sha256 AS ajustes_huella,
        j.ajustes_canonico,j.vigente_desde
 INTO captura
 FROM (SELECT * FROM vec_contratacion_temporal.regla_base_activacion_v1
       ORDER BY secuencia DESC LIMIT 1) a
 JOIN vec_contratacion_temporal.regla_base_publicada_v1 b
   ON b.catalogo_id=a.catalogo_id AND b.version=a.version AND b.huella_sha256=a.huella_sha256
 LEFT JOIN LATERAL (
   SELECT version,huella_sha256,ajustes_canonico,vigente_desde
   FROM vec_contratacion_temporal.regla_ajuste_version_v1
   WHERE catalogo_id='vec.contratacion_temporal.reglas.ajustes'
     AND vigente_desde<=v_capturada_en
   ORDER BY vigente_desde DESC,version DESC LIMIT 1
 ) j ON true
 WHERE a.activa;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT-190: base de reglas no publicada' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_contratacion_temporal.fase_regla_instantanea_v1
  (expediente_ref,version,fase_clave,fase_desde,estado,base_catalogo_id,base_version,
   base_huella_sha256,ajustes_catalogo_id,ajustes_encontrados,ajustes_version,
   ajustes_huella_sha256,ajustes_canonico,ajustes_vigente_desde,capturada_en)
 VALUES (NEW.expediente_ref,NEW.version,NEW.fase_clave,NEW.fase_desde,'capturada',
   captura.catalogo_id,captura.version,captura.huella_sha256,
   'vec.contratacion_temporal.reglas.ajustes',captura.ajustes_version IS NOT NULL,
   coalesce(captura.ajustes_version,0),
   coalesce(captura.ajustes_huella,encode(sha256(convert_to('{}','UTF8')),'hex')),
   coalesce(captura.ajustes_canonico,'{}'),captura.vigente_desde,
   v_capturada_en);
 RETURN NULL;
END $f$;
CREATE TRIGGER fase_entrada_instantanea_regla AFTER INSERT
ON vec_contratacion_temporal.fase_entrada_publicacion_rrhh FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.registrar_instantanea_fase_regla_v1();

CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.regla_base_publicada_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE
ON vec_contratacion_temporal.regla_base_publicada_v1 FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.regla_base_activacion_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE
ON vec_contratacion_temporal.regla_base_activacion_v1 FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.fase_regla_instantanea_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE
ON vec_contratacion_temporal.fase_regla_instantanea_v1 FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.regla_legado_transicion_v1 FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE
ON vec_contratacion_temporal.regla_legado_transicion_v1 FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

DO $proteccion$
DECLARE tabla text; r record;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['regla_base_publicada_v1','regla_base_activacion_v1','fase_regla_instantanea_v1','regla_legado_transicion_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_total ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',tabla);
  FOR r IN SELECT DISTINCT x.grantee FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
   WHERE c.oid=to_regclass('vec_contratacion_temporal.'||tabla) AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',tabla,pg_get_userbyid(r.grantee));
  END LOOP;
  EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM PUBLIC',tabla);
 END LOOP;
END $proteccion$;

ALTER FUNCTION vec_contratacion_temporal.congelar_regla_legado_transicion_v1()
 OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.congelar_regla_legado_transicion_v1() FROM PUBLIC;
ALTER FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1()
 OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.registrar_instantanea_fase_regla_v1()
 OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1(),
 vec_contratacion_temporal.registrar_instantanea_fase_regla_v1() FROM PUBLIC;

-- Metadatos fuera del contenido canónico y de su recibo. Solo los consume la
-- fachada atestada tras la decisión V3; las bases se envían una sola vez.
CREATE FUNCTION vec_contratacion_temporal.leer_capturas_pagina_rrhh_v1(p_contenido bytea)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security='on' SET "TimeZone"='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $function$
DECLARE v_resultado jsonb; v_filas integer; v_capturas integer;
BEGIN
 IF CURRENT_USER<>'vec_contratacion_temporal_propietario' OR p_contenido IS NULL THEN
  RAISE EXCEPTION 'capturas RRHH no disponibles' USING ERRCODE='42501';
 END IF;
 WITH pagina AS MATERIALIZED (
  SELECT * FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(p_contenido)
 ), datos AS MATERIALIZED (
  SELECT p.orden,CASE WHEN s.estado='legado_sin_instantanea' AND t.unico THEN 'legado_base_transicion' ELSE s.estado END AS estado,
   s.fase_clave,s.fase_desde,coalesce(s.base_catalogo_id,t.base_catalogo_id) AS base_catalogo_id,
   coalesce(s.base_version,t.base_version) AS base_version,
   coalesce(s.base_huella_sha256,t.base_huella_sha256) AS base_huella_sha256,
   b.canonico AS base_canonico,coalesce(s.ajustes_catalogo_id,t.ajustes_catalogo_id) AS ajustes_catalogo_id,
   coalesce(s.ajustes_encontrados,t.ajustes_encontrados) AS ajustes_encontrados,
   coalesce(s.ajustes_version,t.ajustes_version) AS ajustes_version,
   coalesce(s.ajustes_huella_sha256,t.ajustes_huella_sha256) AS ajustes_huella_sha256,
   coalesce(s.ajustes_canonico,t.ajustes_canonico) AS ajustes_canonico,
   coalesce(s.ajustes_vigente_desde,t.ajustes_vigente_desde) AS ajustes_vigente_desde,
   coalesce(s.capturada_en,t.capturada_en) AS capturada_en
  FROM pagina p LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
   ON s.expediente_ref=p.expediente_ref AND s.version=p.version
  LEFT JOIN vec_contratacion_temporal.regla_legado_transicion_v1 t
   ON s.estado='legado_sin_instantanea'
  LEFT JOIN vec_contratacion_temporal.regla_base_publicada_v1 b
   ON b.catalogo_id=coalesce(s.base_catalogo_id,t.base_catalogo_id)
    AND b.version=coalesce(s.base_version,t.base_version)
    AND b.huella_sha256=coalesce(s.base_huella_sha256,t.base_huella_sha256)
 )
 SELECT jsonb_build_object(
  'filas',coalesce(jsonb_agg(jsonb_build_object(
   'estado',d.estado,'fase',d.fase_clave,'fase_desde',d.fase_desde,
   'base_id',d.base_catalogo_id,'base_version',d.base_version,
   'base_huella',d.base_huella_sha256,'ajustes_id',d.ajustes_catalogo_id,
   'ajustes_encontrados',d.ajustes_encontrados,'ajustes_version',d.ajustes_version,
   'ajustes_huella',d.ajustes_huella_sha256,
   'ajustes_vigente_desde',d.ajustes_vigente_desde,'capturada_en',d.capturada_en)
   ORDER BY d.orden),'[]'::jsonb),
  'bases',coalesce(jsonb_object_agg(DISTINCT d.base_huella_sha256,d.base_canonico)
   FILTER (WHERE d.estado IN ('capturada','legado_base_transicion')),'{}'::jsonb),
  'ajustes',coalesce(jsonb_object_agg(DISTINCT d.ajustes_huella_sha256,d.ajustes_canonico)
   FILTER (WHERE d.estado IN ('capturada','legado_base_transicion')),'{}'::jsonb)),
  count(*)::integer,count(*) FILTER (WHERE d.estado IS NOT NULL)::integer
 INTO v_resultado,v_filas,v_capturas FROM datos d;
 IF v_filas>100 OR v_filas<>v_capturas OR octet_length(v_resultado::text)>4194304 THEN
  RAISE EXCEPTION 'capturas RRHH incompletas' USING ERRCODE='42501';
 END IF;
 RETURN v_resultado;
END $function$;
ALTER FUNCTION vec_contratacion_temporal.leer_capturas_pagina_rrhh_v1(bytea) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_capturas_pagina_rrhh_v1(bytea) FROM PUBLIC;

-- El cambio de columnas obliga a retirar explícitamente v5 antes de v4.
DROP FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text);

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_cursor text)
 RETURNS TABLE(clase text, estado_clave text, fase_clave text, fase_desde timestamp with time zone, urgente boolean, numero numeric, captura jsonb)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_corte_global numeric;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_cursor IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resumen de cuadro RRHH no disponible';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_consulta);
    IF p_cursor = '' THEN
        SELECT ultimo_corte INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.control_publicacion_rrhh
         WHERE control;
    ELSE
        SELECT familia.corte_global INTO STRICT v_corte_global
          FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
          JOIN vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
            USING (familia_ref)
         WHERE cursor.token_huella_sha256 = pg_catalog.encode(
             pg_catalog.sha256(pg_catalog.convert_to(p_cursor, 'UTF8')), 'hex'
         );
    END IF;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       0 AND 9007199254740991::numeric OR
       v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'corte de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.version,
               publicada.organizacion_ref,
               publicada.numero_visible AS numero_visible, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref, publicada.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= v_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    ), filtradas AS MATERIALIZED (
        SELECT ultima.expediente_ref, ultima.version,
               ultima.estado_clave, ultima.fase_clave
          FROM ultimas ultima
          LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
            ON numeracion.expediente_ref = ultima.expediente_ref
         WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
           AND (p_alcance.clase_ambito = 'organizacion'
                OR p_alcance.clase_ambito = 'centro' AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
                OR p_alcance.clase_ambito = 'unidad_gestion' AND ultima.unidad_ref IS NOT NULL AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
           AND (p_consulta.texto = '' OR pg_catalog.left(COALESCE(numeracion.numero_visible, ultima.numero_visible), pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
           AND (p_consulta.estado_clave = '' OR ultima.estado_clave COLLATE "C" = p_consulta.estado_clave COLLATE "C")
           AND (p_consulta.fase_clave = '' OR ultima.fase_clave COLLATE "C" = p_consulta.fase_clave COLLATE "C")
    )
    SELECT 'estado_fase'::text, filtrada.estado_clave, filtrada.fase_clave,
           NULL::timestamptz, NULL::boolean, pg_catalog.count(*)::numeric, NULL::jsonb
      FROM filtradas filtrada
     GROUP BY filtrada.estado_clave, filtrada.fase_clave
    UNION ALL
    -- Un expediente en trámite sin fecha de entrada en fase aparece con
    -- fase_desde nula: la fachada lo rechaza en lugar de omitirlo.
    SELECT 'plazo'::text, NULL::text, filtrada.fase_clave, entrada.fase_desde,
           COALESCE(urgencia.primera_version <= filtrada.version, false),
           pg_catalog.count(*)::numeric,
           jsonb_build_object('estado',CASE WHEN s.estado='legado_sin_instantanea' AND t.unico THEN 'legado_base_transicion' ELSE s.estado END,'fase',s.fase_clave,
             'fase_desde',s.fase_desde,'base_id',coalesce(s.base_catalogo_id,t.base_catalogo_id),
             'base_version',coalesce(s.base_version,t.base_version),'base_huella',coalesce(s.base_huella_sha256,t.base_huella_sha256),
             'base_canonico',b.canonico,'ajustes_id',coalesce(s.ajustes_catalogo_id,t.ajustes_catalogo_id),
             'ajustes_encontrados',coalesce(s.ajustes_encontrados,t.ajustes_encontrados),
             'ajustes_version',coalesce(s.ajustes_version,t.ajustes_version),'ajustes_huella',coalesce(s.ajustes_huella_sha256,t.ajustes_huella_sha256),
             'ajustes_canonico',coalesce(s.ajustes_canonico,t.ajustes_canonico),
             'ajustes_vigente_desde',coalesce(s.ajustes_vigente_desde,t.ajustes_vigente_desde),'capturada_en',coalesce(s.capturada_en,t.capturada_en))
      FROM filtradas filtrada
      LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
        ON entrada.expediente_ref = filtrada.expediente_ref
       AND entrada.version = filtrada.version
      LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
        ON s.expediente_ref=filtrada.expediente_ref AND s.version=filtrada.version
      LEFT JOIN vec_contratacion_temporal.regla_legado_transicion_v1 t
        ON s.estado='legado_sin_instantanea'
      LEFT JOIN vec_contratacion_temporal.regla_base_publicada_v1 b
        ON b.catalogo_id=coalesce(s.base_catalogo_id,t.base_catalogo_id)
         AND b.version=coalesce(s.base_version,t.base_version)
         AND b.huella_sha256=coalesce(s.base_huella_sha256,t.base_huella_sha256)
      LEFT JOIN (
          SELECT expediente_ref, pg_catalog.min(version) AS primera_version
            FROM vec_contratacion_temporal.urgencia_expediente_analisis
           GROUP BY expediente_ref
      ) urgencia ON urgencia.expediente_ref = filtrada.expediente_ref
     WHERE filtrada.estado_clave NOT IN ('completado', 'cancelado')
     GROUP BY filtrada.fase_clave, entrada.fase_desde, 5, 7;
END
$function$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(contenido_canonico bytea, cursor_siguiente text, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, total_filtrado numeric, en_tramitacion numeric, con_incidencia numeric, en_llamamiento numeric, fase_desde_expedientes text[], fase_desde_instantes timestamp with time zone[], urgente_expedientes boolean[], capturas_plazo jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_v3 record;
    v_urgentes boolean[];
    v_filas integer;
    v_capturas jsonb;
BEGIN
    SELECT * INTO STRICT v_v3
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    SELECT COALESCE(pg_catalog.array_agg(EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.urgencia_expediente_analisis u
                WHERE u.expediente_ref = leido.expediente_ref
                  AND u.version <= leido.version
           ) ORDER BY leido.orden), '{}'),
           pg_catalog.count(*)::integer
      INTO v_urgentes, v_filas
      FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(
               v_v3.contenido_canonico
           ) leido;
    IF v_filas <> v_v3.total
       OR pg_catalog.cardinality(v_urgentes)
          <> pg_catalog.cardinality(v_v3.fase_desde_expedientes) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'urgencia de cuadro RRHH no disponible';
    END IF;
    v_capturas := vec_contratacion_temporal.leer_capturas_pagina_rrhh_v1(v_v3.contenido_canonico);
    RETURN QUERY SELECT v_v3.contenido_canonico, v_v3.cursor_siguiente,
        v_v3.esquema, v_v3.acceso_ref, v_v3.secuencia,
        v_v3.anterior_sha256, v_v3.huella_sha256,
        v_v3.vinculo_identidad_huella_sha256,
        v_v3.alcance_huella_sha256, v_v3.registrada_en,
        v_v3.auditoria_vec_ref, v_v3.auditoria_vec_huella_sha256,
        v_v3.consumo_vec_huella_sha256, v_v3.contenido_huella_sha256,
        v_v3.resultado_huella_sha256, v_v3.cursor_huella_sha256,
        v_v3.generada_en, v_v3.expediente_ref, v_v3.version_expediente,
        v_v3.total, v_v3.recibo_sello_sha256, v_v3.total_filtrado,
        v_v3.en_tramitacion, v_v3.con_incidencia, v_v3.en_llamamiento,
        v_v3.fase_desde_expedientes, v_v3.fase_desde_instantes, v_urgentes, v_capturas;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$function$;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign_1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(contenido_canonico bytea, cursor_siguiente text, esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text, huella_sha256 text, vinculo_identidad_huella_sha256 text, alcance_huella_sha256 text, registrada_en timestamp with time zone, auditoria_vec_ref text, auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text, contenido_huella_sha256 text, resultado_huella_sha256 text, cursor_huella_sha256 text, generada_en timestamp with time zone, expediente_ref text, version_expediente numeric, total smallint, recibo_sello_sha256 text, total_filtrado numeric, en_tramitacion numeric, con_incidencia numeric, en_llamamiento numeric, fase_desde_expedientes text[], fase_desde_instantes timestamp with time zone[], urgente_expedientes boolean[], capturas_plazo jsonb, recuento_estados text[], recuento_fases text[], recuento_numeros numeric[], plazo_fases text[], plazo_desde timestamp with time zone[], plazo_urgentes boolean[], plazo_numeros numeric[], capturas_grupos jsonb)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_v4 record;
    v_estados text[];
    v_fases text[];
    v_numeros numeric[];
    v_total_recuento numeric;
    v_en_tramite numeric;
    v_plazo_fases text[];
    v_plazo_desde timestamptz[];
    v_plazo_urgentes boolean[];
    v_plazo_numeros numeric[];
    v_total_plazos numeric;
    v_sin_fase integer;
    v_sin_captura integer;
    v_contextos jsonb;
BEGIN
    -- Autorización, consumo, auditoría de acceso y página: los de la v4.
    SELECT * INTO STRICT v_v4
      FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
          p_alcance, p_consulta, p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico, p_persona_version,
          p_perfil_version, p_payload_vec_ad_3, p_sobre_cose_sign_1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
    WITH resumen AS MATERIALIZED (
        SELECT *
          FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
              p_alcance, p_consulta, p_consulta.cursor
          )
    ), recuento AS (
        SELECT * FROM resumen WHERE clase = 'estado_fase'
    ), plazo AS (
        SELECT * FROM resumen WHERE clase = 'plazo'
    )
    SELECT (SELECT COALESCE(pg_catalog.array_agg(r.estado_clave ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.estado_clave COLLATE "C", r.fase_clave COLLATE "C"), '{}') FROM recuento r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM recuento r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM recuento r
             WHERE r.estado_clave NOT IN ('completado', 'cancelado')),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_clave ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente, r.captura->>'base_huella', r.captura->>'ajustes_huella', r.captura->>'capturada_en'), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.fase_desde ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente, r.captura->>'base_huella', r.captura->>'ajustes_huella', r.captura->>'capturada_en'), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.urgente ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente, r.captura->>'base_huella', r.captura->>'ajustes_huella', r.captura->>'capturada_en'), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.array_agg(r.numero ORDER BY r.fase_clave COLLATE "C", r.fase_desde, r.urgente, r.captura->>'base_huella', r.captura->>'ajustes_huella', r.captura->>'capturada_en'), '{}') FROM plazo r),
           (SELECT COALESCE(pg_catalog.sum(r.numero), 0) FROM plazo r),
           (SELECT pg_catalog.count(*)::integer FROM plazo r WHERE r.fase_desde IS NULL),
           (SELECT jsonb_build_object(
      'grupos',coalesce(jsonb_agg(
        (r.captura - 'base_canonico' - 'ajustes_canonico') ||
        jsonb_build_object('numero',r.numero,'urgente',r.urgente)
        ORDER BY r.fase_clave COLLATE "C",r.fase_desde,r.urgente,
          r.captura->>'base_huella',r.captura->>'ajustes_huella',r.captura->>'capturada_en'),'[]'::jsonb),
      'bases',coalesce(jsonb_object_agg(DISTINCT r.captura->>'base_huella',r.captura->>'base_canonico')
        FILTER (WHERE r.captura->>'estado' IN ('capturada','legado_base_transicion')),'{}'::jsonb),
      'ajustes',coalesce(jsonb_object_agg(DISTINCT r.captura->>'ajustes_huella',r.captura->>'ajustes_canonico')
        FILTER (WHERE r.captura->>'estado' IN ('capturada','legado_base_transicion')),'{}'::jsonb)) FROM plazo r),
           (SELECT count(*)::integer FROM plazo r WHERE r.captura->>'estado' IS NULL)
      INTO v_estados, v_fases, v_numeros, v_total_recuento, v_en_tramite,
           v_plazo_fases, v_plazo_desde, v_plazo_urgentes, v_plazo_numeros,
           v_total_plazos, v_sin_fase, v_contextos, v_sin_captura;
    -- Coherencia con los totales de la v4 (mismo corte y filtros): el
    -- recuento suma el total filtrado y los grupos de plazo suman los
    -- expedientes en trámite, todos con su entrada en fase. Si no cuadra, no
    -- se publica nada.
    IF v_total_recuento <> v_v4.total_filtrado
       OR v_total_plazos <> v_en_tramite
       OR v_sin_fase <> 0 OR v_sin_captura <> 0
       OR octet_length(v_contextos::text)>4194304 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'resumen de cuadro RRHH no disponible';
    END IF;
    RETURN QUERY SELECT v_v4.contenido_canonico, v_v4.cursor_siguiente,
        v_v4.esquema, v_v4.acceso_ref, v_v4.secuencia,
        v_v4.anterior_sha256, v_v4.huella_sha256,
        v_v4.vinculo_identidad_huella_sha256,
        v_v4.alcance_huella_sha256, v_v4.registrada_en,
        v_v4.auditoria_vec_ref, v_v4.auditoria_vec_huella_sha256,
        v_v4.consumo_vec_huella_sha256, v_v4.contenido_huella_sha256,
        v_v4.resultado_huella_sha256, v_v4.cursor_huella_sha256,
        v_v4.generada_en, v_v4.expediente_ref, v_v4.version_expediente,
        v_v4.total, v_v4.recibo_sello_sha256, v_v4.total_filtrado,
        v_v4.en_tramitacion, v_v4.con_incidencia, v_v4.en_llamamiento,
        v_v4.fase_desde_expedientes, v_v4.fase_desde_instantes,
        v_v4.urgente_expedientes, v_v4.capturas_plazo,
        v_estados, v_fases, v_numeros,
        v_plazo_fases, v_plazo_desde, v_plazo_urgentes, v_plazo_numeros,
        v_contextos;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta RRHH rechazada';
END
$function$;

ALTER FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text) FROM PUBLIC;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_consultor_rrhh;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_consultor_rrhh;

DO $post$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_class c
  WHERE c.oid IN ('vec_contratacion_temporal.regla_base_publicada_v1'::regclass,
     'vec_contratacion_temporal.regla_base_activacion_v1'::regclass,
     'vec_contratacion_temporal.fase_regla_instantanea_v1'::regclass,
     'vec_contratacion_temporal.regla_legado_transicion_v1'::regclass)
   AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole
        OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
  OR EXISTS (SELECT 1 FROM pg_class c,
     LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
    WHERE c.oid IN ('vec_contratacion_temporal.regla_base_publicada_v1'::regclass,
       'vec_contratacion_temporal.regla_base_activacion_v1'::regclass,
       'vec_contratacion_temporal.fase_regla_instantanea_v1'::regclass,
     'vec_contratacion_temporal.regla_legado_transicion_v1'::regclass)
      AND x.grantee<>c.relowner)
  OR (SELECT count(*) FROM vec_contratacion_temporal.fase_regla_instantanea_v1)
     <> (SELECT count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh)
  OR (SELECT count(*) FROM pg_trigger WHERE tgname IN (
       'regla_base_activacion_cas','fase_entrada_instantanea_regla',
       'regla_legado_transicion_primera_activacion'))<>3
 THEN RAISE EXCEPTION 'CT-190: postcondición incumplida' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
