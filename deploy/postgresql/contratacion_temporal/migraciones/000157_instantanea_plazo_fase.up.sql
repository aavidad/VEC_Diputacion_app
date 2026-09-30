\set ON_ERROR_STOP on
-- CT-157. La captura nace del INSERT CT110 dentro de la transacción de publicación.
-- La historia anterior queda marcada expresamente como legado, sin regla atribuida.
-- No instalar antes de H6 ni activar la edición de ajustes antes del consumidor Go.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000157', 0));
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh IN SHARE MODE;
LOCK TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh IN SHARE MODE;
DO $pre$
BEGIN
 IF current_user <> 'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.regla_base_publicada_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.regla_base_activacion_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.fase_regla_instantanea_v1') IS NOT NULL
    OR to_regclass('vec_contratacion_temporal.regla_ajuste_version_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
 THEN RAISE EXCEPTION 'CT-157: preimagen incompatible' USING ERRCODE='55000'; END IF;
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
SET search_path=pg_catalog SET row_security='on' AS $f$
DECLARE v_cabeza bigint;
BEGIN
 IF TG_OP <> 'INSERT' OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
    OR TG_TABLE_NAME <> 'regla_base_activacion_v1'
    OR current_user <> 'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT-157: activación denegada' USING ERRCODE='42501'; END IF;
 SELECT coalesce(max(secuencia),0) INTO v_cabeza
 FROM vec_contratacion_temporal.regla_base_activacion_v1;
 IF NEW.secuencia_esperada <> v_cabeza THEN
  RAISE EXCEPTION 'CT-157: cabeza de activación distinta' USING ERRCODE='40001';
 END IF;
 IF NEW.activa AND NOT EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.regla_base_publicada_v1 b
  WHERE b.catalogo_id=NEW.catalogo_id AND b.version=NEW.version
    AND b.huella_sha256=NEW.huella_sha256 AND b.aprobacion_ref=NEW.aprobacion_ref
 ) THEN
  RAISE EXCEPTION 'CT-157: activación sin base aprobada' USING ERRCODE='42501';
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
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $f$
DECLARE anterior record; captura record;
BEGIN
 IF TG_OP <> 'INSERT' OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
    OR TG_TABLE_NAME <> 'fase_entrada_publicacion_rrhh'
    OR current_user <> 'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT-157: registro denegado' USING ERRCODE='42501'; END IF;
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
 -- Una única sentencia MVCC fija base activa y cabeza de ajustes ya visibles.
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
   ORDER BY version DESC LIMIT 1
 ) j ON true
 WHERE a.activa;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT-157: base de reglas no publicada' USING ERRCODE='55000'; END IF;
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
   date_trunc('microseconds',clock_timestamp()));
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

-- El cuadro v5 conserva el recibo/atestado v4 intacto y añade datos de fila
-- alineados con su contenido canónico. Ningún identificador de persona viaja.
CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,
 p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload_vec_ad_3 bytea,p_sobre_cose_sign_1 bytea,p_evidencia_verificacion bytea,
 p_raiz_publica_spki bytea)
RETURNS TABLE (
 contenido_canonico bytea,cursor_siguiente text,esquema text,acceso_ref text,
 secuencia numeric,anterior_sha256 text,huella_sha256 text,
 vinculo_identidad_huella_sha256 text,alcance_huella_sha256 text,
 registrada_en timestamptz,auditoria_vec_ref text,auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text,contenido_huella_sha256 text,
 resultado_huella_sha256 text,cursor_huella_sha256 text,generada_en timestamptz,
 expediente_ref text,version_expediente numeric,total smallint,recibo_sello_sha256 text,
 total_filtrado numeric,en_tramitacion numeric,con_incidencia numeric,
 en_llamamiento numeric,fase_desde_expedientes text[],fase_desde_instantes timestamptz[],
 urgente_expedientes boolean[],instantaneas_regla jsonb[],
 bases_regla jsonb,ajustes_regla jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $f$
DECLARE v_v4 record; v_instantaneas jsonb[]; v_bases jsonb; v_ajustes jsonb;
 v_filas integer; v_con_fila integer;
BEGIN
 SELECT * INTO STRICT v_v4 FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
  p_alcance,p_consulta,p_capacidad_canonica,p_decision_canonica,p_motivo_canonico,
  p_contexto_actor_canonico,p_persona_version,p_perfil_version,p_payload_vec_ad_3,
  p_sobre_cose_sign_1,p_evidencia_verificacion,p_raiz_publica_spki);
 SELECT coalesce(array_agg(
  CASE WHEN s.estado='capturada' THEN jsonb_build_object(
   'estado',s.estado,'expediente_ref',e.expediente_ref,
   'version_expediente',e.version,'fase',s.fase_clave,'fase_desde',s.fase_desde,
   'catalogo_base_id',s.base_catalogo_id,'base_version',s.base_version,
   'base_huella_sha256',s.base_huella_sha256,
   'catalogo_ajustes_id',s.ajustes_catalogo_id,'ajustes_encontrados',s.ajustes_encontrados,
   'ajustes_version',s.ajustes_version,'ajustes_huella_sha256',s.ajustes_huella_sha256,
   'ajustes_vigente_desde',s.ajustes_vigente_desde,
   'capturada_en',s.capturada_en)
  ELSE jsonb_build_object('estado','legado_sin_instantanea',
   'expediente_ref',e.expediente_ref,'version_expediente',e.version,
   'fase',s.fase_clave,'fase_desde',s.fase_desde) END ORDER BY e.orden),ARRAY[]::jsonb[]),
  count(*)::integer,count(s.version)::integer
 INTO v_instantaneas,v_filas,v_con_fila
 FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(v_v4.contenido_canonico) e
 LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
  ON s.expediente_ref=e.expediente_ref AND s.version=e.version
 ;
 IF v_filas<>v_v4.total OR v_con_fila<>v_filas OR
    cardinality(v_instantaneas)<>cardinality(v_v4.fase_desde_expedientes) THEN
  RAISE EXCEPTION 'CT-157: instantánea no disponible' USING ERRCODE='42501';
 END IF;
 -- Las definiciones completas se envían una sola vez por contexto distinto.
 -- La página de 100 con una base de ~30 KiB no multiplica la base por 100.
 WITH pagina AS MATERIALIZED (
  SELECT e.expediente_ref,e.version
  FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(v_v4.contenido_canonico) e
 ), usados AS MATERIALIZED (
  SELECT DISTINCT s.base_catalogo_id,s.base_version,s.base_huella_sha256,
    s.ajustes_catalogo_id,s.ajustes_version,s.ajustes_huella_sha256,
    s.ajustes_canonico
  FROM pagina p JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
    ON s.expediente_ref=p.expediente_ref AND s.version=p.version
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
 IF octet_length(v_bases::text)+octet_length(v_ajustes::text)>4194304 THEN
  RAISE EXCEPTION 'CT-157: página de contextos demasiado grande' USING ERRCODE='54000';
 END IF;
 RETURN QUERY SELECT v_v4.contenido_canonico,v_v4.cursor_siguiente,v_v4.esquema,
  v_v4.acceso_ref,v_v4.secuencia,v_v4.anterior_sha256,v_v4.huella_sha256,
  v_v4.vinculo_identidad_huella_sha256,v_v4.alcance_huella_sha256,v_v4.registrada_en,
  v_v4.auditoria_vec_ref,v_v4.auditoria_vec_huella_sha256,v_v4.consumo_vec_huella_sha256,
  v_v4.contenido_huella_sha256,v_v4.resultado_huella_sha256,v_v4.cursor_huella_sha256,
  v_v4.generada_en,v_v4.expediente_ref,v_v4.version_expediente,v_v4.total,
  v_v4.recibo_sello_sha256,v_v4.total_filtrado,v_v4.en_tramitacion,
  v_v4.con_incidencia,v_v4.en_llamamiento,v_v4.fase_desde_expedientes,
  v_v4.fase_desde_instantes,v_v4.urgente_expedientes,v_instantaneas,
  v_bases,v_ajustes;
EXCEPTION
 WHEN SQLSTATE '40001' OR SQLSTATE '40P01' OR SQLSTATE '55P03'
   OR SQLSTATE '57014' OR SQLSTATE '54000' THEN RAISE;
 WHEN OTHERS THEN RAISE EXCEPTION 'consulta RRHH rechazada' USING ERRCODE='42501';
END $f$;
ALTER FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1() OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.registrar_instantanea_fase_regla_v1() OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_activacion_regla_base_v1(),
 vec_contratacion_temporal.registrar_instantanea_fase_regla_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh;
DO $post$
DECLARE v5 regprocedure:=
 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
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
  OR NOT (SELECT prosecdef AND proowner='vec_contratacion_temporal_propietario'::regrole
       FROM pg_proc WHERE oid=v5)
  OR NOT has_function_privilege('vec_contratacion_temporal_consultor_rrhh',v5,'EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',v5,'EXECUTE')
  OR (SELECT count(*) FROM vec_contratacion_temporal.fase_regla_instantanea_v1)
     <> (SELECT count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh)
  OR (SELECT count(*) FROM pg_trigger WHERE tgname IN (
       'regla_base_activacion_cas','fase_entrada_instantanea_regla'))<>2
 THEN RAISE EXCEPTION 'CT-157: postcondición incumplida' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
