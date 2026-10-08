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

DO $proteccion$
DECLARE tabla text; r record;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['regla_base_publicada_v1','regla_base_activacion_v1','fase_regla_instantanea_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
  EXECUTE format('CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
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

-- El grupo conserva el contexto exacto de cada tramo. No se reescribe la
-- función de CT184/CT187: comparte su corte y sus predicados nominales.
CREATE FUNCTION vec_contratacion_temporal.contar_plazos_con_instantanea_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p_cursor text)
RETURNS TABLE(fase_clave text,fase_desde timestamptz,urgente boolean,numero numeric,
 estado text,base_catalogo_id text,base_version bigint,base_huella_sha256 text,
 ajustes_catalogo_id text,ajustes_encontrados boolean,ajustes_version bigint,
 ajustes_huella_sha256 text,ajustes_canonico text,ajustes_vigente_desde timestamptz,
 capturada_en timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $grupos$
DECLARE v_corte_global numeric;
BEGIN
 IF CURRENT_USER<>'vec_contratacion_temporal_propietario'
    OR p_alcance IS NULL OR p_consulta IS NULL OR p_cursor IS NULL THEN
  RAISE EXCEPTION 'grupos de plazo no disponibles' USING ERRCODE='42501';
 END IF;
 PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
 PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v1(p_consulta);
 IF p_cursor='' THEN
  SELECT ultimo_corte INTO STRICT v_corte_global
  FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control;
 ELSE
  SELECT familia.corte_global INTO STRICT v_corte_global
  FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
  JOIN vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia USING (familia_ref)
  WHERE cursor.token_huella_sha256=pg_catalog.encode(
   pg_catalog.sha256(pg_catalog.convert_to(p_cursor,'UTF8')),'hex');
 END IF;
 IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN 0 AND 9007199254740991::numeric
    OR v_corte_global<>pg_catalog.trunc(v_corte_global) THEN
  RAISE EXCEPTION 'corte de grupos no disponible' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 WITH ultimas AS MATERIALIZED (
  SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
   publicada.expediente_ref,publicada.version,publicada.organizacion_ref,
   vec_contratacion_temporal.numero_visible_vigente_v1(
    publicada.expediente_ref,publicada.numero_visible) AS numero_visible,
   publicada.fase_clave,publicada.estado_clave,publicada.centro_ref,publicada.unidad_ref
  FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
  WHERE publicada.corte_global<=v_corte_global
  ORDER BY publicada.expediente_ref COLLATE "C",publicada.corte_global DESC
 ), filtradas AS MATERIALIZED (
  SELECT ultima.expediente_ref,ultima.version,ultima.estado_clave,ultima.fase_clave
  FROM ultimas ultima
  WHERE ultima.organizacion_ref COLLATE "C"=p_alcance.organizacion_ref COLLATE "C"
   AND (p_alcance.clase_ambito='organizacion'
    OR p_alcance.clase_ambito='centro' AND ultima.centro_ref COLLATE "C"=p_alcance.ambito_ref COLLATE "C"
    OR p_alcance.clase_ambito='unidad_gestion' AND ultima.unidad_ref IS NOT NULL
       AND ultima.unidad_ref COLLATE "C"=p_alcance.ambito_ref COLLATE "C")
   AND (p_consulta.texto='' OR pg_catalog.left(ultima.numero_visible,pg_catalog.char_length(p_consulta.texto)) COLLATE "C"=p_consulta.texto COLLATE "C")
   AND (p_consulta.estado_clave='' OR ultima.estado_clave COLLATE "C"=p_consulta.estado_clave COLLATE "C")
   AND (p_consulta.fase_clave='' OR ultima.fase_clave COLLATE "C"=p_consulta.fase_clave COLLATE "C")
 )
 SELECT filtrada.fase_clave,entrada.fase_desde,
  EXISTS (SELECT 1 FROM vec_contratacion_temporal.urgencia_expediente_analisis u
          WHERE u.expediente_ref=filtrada.expediente_ref AND u.version<=filtrada.version),
  pg_catalog.count(*)::numeric,s.estado,s.base_catalogo_id,s.base_version,
  s.base_huella_sha256,s.ajustes_catalogo_id,s.ajustes_encontrados,
  s.ajustes_version,s.ajustes_huella_sha256,s.ajustes_canonico,
  s.ajustes_vigente_desde,s.capturada_en
 FROM filtradas filtrada
 LEFT JOIN vec_contratacion_temporal.fase_entrada_publicacion_rrhh entrada
  ON entrada.expediente_ref=filtrada.expediente_ref AND entrada.version=filtrada.version
 LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
  ON s.expediente_ref=filtrada.expediente_ref AND s.version=filtrada.version
 WHERE filtrada.estado_clave NOT IN ('completado','cancelado')
 GROUP BY filtrada.fase_clave,entrada.fase_desde,3,s.estado,s.base_catalogo_id,
  s.base_version,s.base_huella_sha256,s.ajustes_catalogo_id,s.ajustes_encontrados,
  s.ajustes_version,s.ajustes_huella_sha256,s.ajustes_canonico,
  s.ajustes_vigente_desde,s.capturada_en;
END $grupos$;
ALTER FUNCTION vec_contratacion_temporal.contar_plazos_con_instantanea_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)
 OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.contar_plazos_con_instantanea_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text) FROM PUBLIC;

-- El consumidor RRHH ya atestado puede reunir estos materiales dentro de su
-- propia función SECURITY DEFINER. La función no decide ni consume V3.
CREATE FUNCTION vec_contratacion_temporal.leer_instantaneas_pagina_rrhh_v1(
 p_contenido_canonico bytea)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
AS $f$
DECLARE v_filas integer; v_capturas integer; v_instantaneas jsonb;
 v_bases jsonb; v_ajustes jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR p_contenido_canonico IS NULL OR octet_length(p_contenido_canonico)>1048576
 THEN RAISE EXCEPTION 'CT-190: instantáneas no disponibles' USING ERRCODE='42501'; END IF;
 WITH pagina AS MATERIALIZED (
  SELECT e.orden,e.expediente_ref,e.version
  FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(p_contenido_canonico) e
 )
 SELECT count(*)::integer,count(s.version)::integer,
  coalesce(jsonb_agg(CASE WHEN s.estado='capturada' THEN jsonb_build_object(
   'estado',s.estado,'expediente_ref',p.expediente_ref,'version_expediente',p.version,
   'fase',s.fase_clave,'fase_desde',s.fase_desde,
   'catalogo_base_id',s.base_catalogo_id,'base_version',s.base_version,
   'base_huella_sha256',s.base_huella_sha256,
   'catalogo_ajustes_id',s.ajustes_catalogo_id,
   'ajustes_encontrados',s.ajustes_encontrados,'ajustes_version',s.ajustes_version,
   'ajustes_huella_sha256',s.ajustes_huella_sha256,
   'ajustes_vigente_desde',s.ajustes_vigente_desde,'capturada_en',s.capturada_en)
   ELSE jsonb_build_object('estado','legado_sin_instantanea',
   'expediente_ref',p.expediente_ref,'version_expediente',p.version,
   'fase',s.fase_clave,'fase_desde',s.fase_desde) END ORDER BY p.orden),'[]'::jsonb)
 INTO v_filas,v_capturas,v_instantaneas
 FROM pagina p LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
   ON s.expediente_ref=p.expediente_ref AND s.version=p.version;
 IF v_filas>100 OR v_filas<>v_capturas THEN
  RAISE EXCEPTION 'CT-190: página sin instantáneas completas' USING ERRCODE='42501';
 END IF;
 WITH usados AS MATERIALIZED (
  SELECT DISTINCT s.base_catalogo_id,s.base_version,s.base_huella_sha256,
   s.ajustes_catalogo_id,s.ajustes_version,s.ajustes_huella_sha256,s.ajustes_canonico
  FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(p_contenido_canonico) e
  JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
   ON s.expediente_ref=e.expediente_ref AND s.version=e.version
  WHERE s.estado='capturada'
 )
 SELECT
  (SELECT coalesce(jsonb_agg(jsonb_build_object(
    'catalogo_base_id',b.catalogo_id,'base_version',b.version,
    'base_huella_sha256',b.huella_sha256,'base_canonico',b.canonico)
    ORDER BY b.catalogo_id,b.version,b.huella_sha256),'[]'::jsonb)
   FROM vec_contratacion_temporal.regla_base_publicada_v1 b
   WHERE EXISTS (SELECT 1 FROM usados u WHERE u.base_catalogo_id=b.catalogo_id
     AND u.base_version=b.version AND u.base_huella_sha256=b.huella_sha256)),
  (SELECT coalesce(jsonb_agg(jsonb_build_object(
    'catalogo_ajustes_id',u.ajustes_catalogo_id,'ajustes_version',u.ajustes_version,
    'ajustes_huella_sha256',u.ajustes_huella_sha256,'ajustes_canonico',u.ajustes_canonico)
    ORDER BY u.ajustes_catalogo_id,u.ajustes_version,u.ajustes_huella_sha256),'[]'::jsonb)
   FROM (SELECT DISTINCT ajustes_catalogo_id,ajustes_version,ajustes_huella_sha256,
      ajustes_canonico FROM usados) u)
 INTO v_bases,v_ajustes;
 IF octet_length(v_instantaneas::text)+octet_length(v_bases::text)+
    octet_length(v_ajustes::text)>4194304 THEN
  RAISE EXCEPTION 'CT-190: página de contextos demasiado grande' USING ERRCODE='54000';
 END IF;
 RETURN jsonb_build_object('instantaneas_regla',v_instantaneas,
  'bases_regla',v_bases,'ajustes_regla',v_ajustes);
END $f$;
ALTER FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1()
 OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.registrar_instantanea_fase_regla_v1()
 OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.leer_instantaneas_pagina_rrhh_v1(bytea)
 OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1(),
 vec_contratacion_temporal.registrar_instantanea_fase_regla_v1(),
 vec_contratacion_temporal.leer_instantaneas_pagina_rrhh_v1(bytea) FROM PUBLIC;
DO $post$
DECLARE paginas regprocedure:='vec_contratacion_temporal.leer_instantaneas_pagina_rrhh_v1(bytea)'::regprocedure;
 grupos regprocedure:=
 'vec_contratacion_temporal.contar_plazos_con_instantanea_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)'::regprocedure;
BEGIN
 IF EXISTS (SELECT 1 FROM pg_class c
  WHERE c.oid IN ('vec_contratacion_temporal.regla_base_publicada_v1'::regclass,
     'vec_contratacion_temporal.regla_base_activacion_v1'::regclass,
     'vec_contratacion_temporal.fase_regla_instantanea_v1'::regclass)
   AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole
        OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
  OR EXISTS (SELECT 1 FROM pg_class c,
     LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
    WHERE c.oid IN ('vec_contratacion_temporal.regla_base_publicada_v1'::regclass,
       'vec_contratacion_temporal.regla_base_activacion_v1'::regclass,
       'vec_contratacion_temporal.fase_regla_instantanea_v1'::regclass)
      AND x.grantee<>c.relowner)
  OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (paginas,grupos)
    AND (NOT p.prosecdef OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
      OR NOT EXISTS (SELECT 1 FROM unnest(p.proconfig) c
       WHERE replace(c,' ','')='search_path=pg_catalog,pg_temp')))
  OR EXISTS (SELECT 1 FROM pg_proc p,
     LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
    WHERE p.oid IN (paginas,grupos) AND x.grantee<>p.proowner)
  OR (SELECT count(*) FROM vec_contratacion_temporal.fase_regla_instantanea_v1)
     <> (SELECT count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh)
  OR (SELECT count(*) FROM pg_trigger WHERE tgname IN (
       'regla_base_activacion_cas','fase_entrada_instantanea_regla'))<>2
 THEN RAISE EXCEPTION 'CT-190: postcondición incumplida' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
