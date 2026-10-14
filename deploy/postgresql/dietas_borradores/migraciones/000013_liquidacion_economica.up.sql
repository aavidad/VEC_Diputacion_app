\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000013',0));

DO $pre$ BEGIN
 IF current_setting('server_version_num')::int < 180000
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR to_regclass('vec_dietas.cola_circuito_comision') IS NULL
    OR to_regclass('vec_dietas.numero_documento_comision') IS NULL
    OR to_regclass('vec_dietas.recibo_operacion_comision') IS NULL
    OR to_regprocedure('vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.cadena_json_go_v1(text)') IS NULL
    OR to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb)') IS NULL
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb)','EXECUTE')
    OR to_regclass('vec_dietas.catalogo_liquidacion_publicado') IS NOT NULL
    OR to_regclass('vec_dietas.version_tarifa_registrada') IS NOT NULL
    OR to_regclass('vec_dietas.regla_devengo_registrada') IS NOT NULL
    OR to_regclass('vec_dietas.liquidacion_economica') IS NOT NULL
    OR to_regprocedure('vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb)') IS NOT NULL
    OR to_regprocedure('vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text)') IS NOT NULL
    OR to_regprocedure('vec_dietas.calcular_tramos_publicados_v1(date,text,date,text,bigint,text,text,text,jsonb)') IS NOT NULL
    OR to_regprocedure('vec_dietas.cotejar_economia_publicada_v1(jsonb,jsonb,jsonb,date,text,date,text)') IS NOT NULL
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       <> 'd28248b05c4730160640a1e57f5da9402ccc19c2253e743b6d23d08e86b485c6'
    OR (SELECT p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
        OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
        OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
        FROM pg_proc p WHERE p.oid='vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
 THEN RAISE EXCEPTION 'Dietas 000013: PARO preimagen o AD271 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La publicación es una fila de configuración con fuente y huella propias.
-- Su carga administrativa exige evidencia fuera de este consumidor de negocio.
CREATE TABLE vec_dietas.catalogo_liquidacion_publicado (
 referencia text NOT NULL,
 version text NOT NULL,
 huella_sha256 text NOT NULL CHECK(huella_sha256~'^[0-9a-f]{64}$'),
 catalogo_canonico bytea NOT NULL,
 catalogo jsonb NOT NULL CHECK(jsonb_typeof(catalogo)='object'),
 publicacion_ref text NOT NULL,
 fecha_publicacion date NOT NULL,
 normas jsonb NOT NULL CHECK(jsonb_typeof(normas)='array' AND jsonb_array_length(normas) BETWEEN 2 AND 8),
 vigente_desde date NOT NULL,
 vigente_hasta date NOT NULL CHECK(vigente_hasta>=vigente_desde),
 version_tarifa_ref text NOT NULL,
 pais_iso2 text NOT NULL CHECK(pais_iso2='ES'),
 grupo smallint NOT NULL CHECK(grupo BETWEEN 1 AND 3),
 vehiculo text NOT NULL CHECK(vehiculo IN ('propio','sin_vehiculo_propio')),
 regla_ref text NOT NULL,
 regla_huella_sha256 text NOT NULL CHECK(regla_huella_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(referencia,version),
 UNIQUE(referencia,version,huella_sha256),
 UNIQUE(referencia,version,huella_sha256,version_tarifa_ref),
 UNIQUE(referencia,version,huella_sha256,version_tarifa_ref,regla_ref,regla_huella_sha256),
 UNIQUE(version_tarifa_ref,pais_iso2,grupo,vehiculo),
 CHECK(referencia~'^[A-Za-z][A-Za-z0-9:_-]{2,127}$' AND version~'^[A-Za-z][A-Za-z0-9:_-]{2,127}$'),
 CHECK(publicacion_ref~'^[A-Za-z][A-Za-z0-9:_-]{2,127}$'),
 CHECK(catalogo=convert_from(catalogo_canonico,'UTF8')::jsonb),
 CHECK(catalogo->>'referencia'=referencia AND catalogo->>'version'=version
   AND catalogo->>'huella_sha256'=huella_sha256
   AND catalogo->>'publicacion_ref'=publicacion_ref
   AND (catalogo->>'fecha_publicacion')::date=fecha_publicacion
   AND catalogo->'normas'=normas
   AND (catalogo->>'vigente_desde')::date=vigente_desde
   AND (catalogo->>'vigente_hasta')::date=vigente_hasta
   AND catalogo->>'version_tarifa_ref'=version_tarifa_ref
   AND catalogo->>'pais_iso2'=pais_iso2
   AND (catalogo->>'grupo')::smallint=grupo
   AND catalogo->>'vehiculo'=vehiculo
   AND catalogo->>'rotulo_clave'='dietas_tarifa_publicada'
   AND catalogo->'regla_devengo'->>'referencia'=regla_ref
   AND catalogo->'regla_devengo'->>'huella_sha256'=regla_huella_sha256)
);
CREATE INDEX catalogo_liquidacion_vigencia_idx ON vec_dietas.catalogo_liquidacion_publicado(vigente_desde,vigente_hasta,referencia,version);

-- Las referencias antiguas y publicadas comparten una FK positiva sin alterar
-- el origen histórico. Ninguna fila de publicación procede del módulo provisional.
CREATE TABLE vec_dietas.version_tarifa_registrada (
 version_ref text PRIMARY KEY,
 origen text NOT NULL CHECK(origen IN ('provisional','publicada')),
 provisional_ref text UNIQUE REFERENCES vec_dietas.version_tarifa_provisional(version_ref),
 catalogo_ref text,
 catalogo_version text,
 catalogo_sha256 text,
 CHECK ((origen='provisional' AND provisional_ref=version_ref AND catalogo_ref IS NULL
          AND catalogo_version IS NULL AND catalogo_sha256 IS NULL)
     OR (origen='publicada' AND provisional_ref IS NULL AND catalogo_ref IS NOT NULL
          AND catalogo_version IS NOT NULL AND catalogo_sha256 IS NOT NULL)),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256,version_ref)
  REFERENCES vec_dietas.catalogo_liquidacion_publicado(referencia,version,huella_sha256,version_tarifa_ref)
);
CREATE TABLE vec_dietas.regla_devengo_registrada (
 regla_ref text PRIMARY KEY,
 version_ref text NOT NULL REFERENCES vec_dietas.version_tarifa_registrada(version_ref),
 origen text NOT NULL CHECK(origen IN ('provisional','publicada')),
 huella_sha256 text NOT NULL CHECK(huella_sha256~'^[0-9a-f]{64}$'),
 provisional_ref text UNIQUE REFERENCES vec_dietas.regla_devengo_provisional(regla_ref),
 catalogo_ref text,
 catalogo_version text,
 catalogo_sha256 text,
 CHECK ((origen='provisional' AND provisional_ref=regla_ref AND catalogo_ref IS NULL
          AND catalogo_version IS NULL AND catalogo_sha256 IS NULL)
     OR (origen='publicada' AND provisional_ref IS NULL AND catalogo_ref IS NOT NULL
          AND catalogo_version IS NOT NULL AND catalogo_sha256 IS NOT NULL)),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256,version_ref,regla_ref,huella_sha256)
  REFERENCES vec_dietas.catalogo_liquidacion_publicado(referencia,version,huella_sha256,version_tarifa_ref,regla_ref,regla_huella_sha256)
);
INSERT INTO vec_dietas.version_tarifa_registrada(version_ref,origen,provisional_ref)
 SELECT version_ref,'provisional',version_ref FROM vec_dietas.version_tarifa_provisional;
INSERT INTO vec_dietas.regla_devengo_registrada(regla_ref,version_ref,origen,huella_sha256,provisional_ref)
 SELECT regla_ref,version_tarifa_ref,'provisional',huella_sha256,regla_ref
 FROM vec_dietas.regla_devengo_provisional;

CREATE TABLE vec_dietas.liquidacion_economica (
 referencia text PRIMARY KEY CHECK(referencia~'^liq_[0-9a-f-]{36}$'),
 comision_ref text NOT NULL,
 version_entrada bigint NOT NULL CHECK(version_entrada>0),
 version_resultante bigint NOT NULL CHECK(version_resultante=version_entrada+1),
 titular_persona_ref text NOT NULL,
 decisor_persona_ref text NOT NULL CHECK(decisor_persona_ref<>titular_persona_ref),
 unidad_ref text NOT NULL,
 numero_documento text NOT NULL CHECK(numero_documento~'^VEC-D-[0-9]{4}-[0-9]{6,18}$'),
 clave_idempotencia text NOT NULL CHECK(clave_idempotencia~'^[A-Za-z0-9_-]{16,128}$'),
 huella_semantica_sha256 text NOT NULL CHECK(huella_semantica_sha256~'^[0-9a-f]{64}$'),
 comando jsonb NOT NULL CHECK(jsonb_typeof(comando)='object'),
 economia_canonica bytea NOT NULL,
 economia jsonb NOT NULL CHECK(jsonb_typeof(economia)='object'),
 instantanea_canonica bytea NOT NULL,
 instantanea jsonb NOT NULL CHECK(jsonb_typeof(instantanea)='object'),
 snapshot_sha256 text NOT NULL CHECK(snapshot_sha256~'^[0-9a-f]{64}$'),
 catalogo_ref text NOT NULL,
 catalogo_version text NOT NULL,
 catalogo_sha256 text NOT NULL CHECK(catalogo_sha256~'^[0-9a-f]{64}$'),
 lineas jsonb NOT NULL CHECK(jsonb_typeof(lineas)='array' AND jsonb_array_length(lineas) BETWEEN 1 AND 102),
 totales jsonb NOT NULL CHECK(jsonb_typeof(totales)='object'),
 publicacion_ref text NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^rcd_[0-9a-f-]{36}$'),
 auditoria_ref text NOT NULL,
 evento_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(comision_ref,version_entrada),
 UNIQUE(comision_ref,version_resultante),
 UNIQUE(decisor_persona_ref,clave_idempotencia),
 FOREIGN KEY(comision_ref,version_entrada) REFERENCES vec_dietas.comision_revision(comision_ref,version),
 FOREIGN KEY(comision_ref,version_resultante) REFERENCES vec_dietas.comision_revision(comision_ref,version),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256)
   REFERENCES vec_dietas.catalogo_liquidacion_publicado(referencia,version,huella_sha256),
 CHECK(economia=convert_from(economia_canonica,'UTF8')::jsonb),
 CHECK(instantanea=convert_from(instantanea_canonica,'UTF8')::jsonb),
 CHECK(instantanea->>'snapshot_sha256'=snapshot_sha256)
);
CREATE INDEX liquidacion_economica_titular_idx ON vec_dietas.liquidacion_economica(titular_persona_ref,comision_ref,version_resultante DESC);

CREATE TABLE vec_dietas.historia_liquidacion_economica (
 evento_ref text PRIMARY KEY REFERENCES vec_dietas.liquidacion_economica(evento_ref),
 comision_ref text NOT NULL,
 version_entrada bigint NOT NULL,
 version_resultante bigint NOT NULL,
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_dietas.liquidacion_economica(recibo_ref),
 titular_persona_ref text NOT NULL,
 actor_ref text NOT NULL,
 estado_anterior text NOT NULL CHECK(estado_anterior='pendiente_liquidacion'),
 estado_nuevo text NOT NULL CHECK(estado_nuevo='pendiente_fiscalizacion'),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_dietas.outbox_liquidacion_economica (
 evento_ref text PRIMARY KEY,
 liquidacion_ref text NOT NULL REFERENCES vec_dietas.liquidacion_economica(referencia),
 comision_ref text NOT NULL,
 version_resultante bigint NOT NULL,
 titular_persona_ref text NOT NULL,
 destinatario_unidad_ref text NOT NULL,
 destinatario_etapa text NOT NULL CHECK(destinatario_etapa='fiscalizacion'),
 estado text NOT NULL CHECK(estado='pendiente'),
 registrada_en timestamptz(6) NOT NULL
);

ALTER TABLE vec_dietas.catalogo_liquidacion_publicado ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.catalogo_liquidacion_publicado FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.catalogo_liquidacion_publicado
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.catalogo_liquidacion_publicado
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.catalogo_liquidacion_publicado FROM PUBLIC,vec_dietas_ejecutor;
ALTER TABLE vec_dietas.version_tarifa_registrada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.version_tarifa_registrada FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.version_tarifa_registrada
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.version_tarifa_registrada
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.version_tarifa_registrada FROM PUBLIC,vec_dietas_ejecutor;
ALTER TABLE vec_dietas.regla_devengo_registrada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.regla_devengo_registrada FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.regla_devengo_registrada
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.regla_devengo_registrada
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.regla_devengo_registrada FROM PUBLIC,vec_dietas_ejecutor;
ALTER TABLE vec_dietas.liquidacion_economica ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.liquidacion_economica FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.liquidacion_economica
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.liquidacion_economica
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.liquidacion_economica FROM PUBLIC,vec_dietas_ejecutor;
ALTER TABLE vec_dietas.historia_liquidacion_economica ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.historia_liquidacion_economica FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.historia_liquidacion_economica
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.historia_liquidacion_economica
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.historia_liquidacion_economica FROM PUBLIC,vec_dietas_ejecutor;
ALTER TABLE vec_dietas.outbox_liquidacion_economica ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.outbox_liquidacion_economica FORCE ROW LEVEL SECURITY;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_dietas.outbox_liquidacion_economica
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.outbox_liquidacion_economica
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON vec_dietas.outbox_liquidacion_economica FROM PUBLIC,vec_dietas_ejecutor;
CREATE POLICY catalogo_publicado_lectura ON vec_dietas.catalogo_liquidacion_publicado
 FOR SELECT TO vec_dietas_propietario USING (true);
CREATE POLICY catalogo_publicado_alta ON vec_dietas.catalogo_liquidacion_publicado
 FOR INSERT TO vec_dietas_propietario WITH CHECK (true);
CREATE POLICY version_tarifa_registrada_lectura ON vec_dietas.version_tarifa_registrada
 FOR SELECT TO vec_dietas_propietario USING (true);
CREATE POLICY version_tarifa_registrada_alta ON vec_dietas.version_tarifa_registrada
 FOR INSERT TO vec_dietas_propietario WITH CHECK (true);
CREATE POLICY regla_devengo_registrada_lectura ON vec_dietas.regla_devengo_registrada
 FOR SELECT TO vec_dietas_propietario USING (true);
CREATE POLICY regla_devengo_registrada_alta ON vec_dietas.regla_devengo_registrada
 FOR INSERT TO vec_dietas_propietario WITH CHECK (true);
CREATE POLICY liquidacion_lectura ON vec_dietas.liquidacion_economica
 FOR SELECT TO vec_dietas_propietario USING (
  titular_persona_ref=current_setting('vec.dietas.persona_ref',true)
  OR decisor_persona_ref=current_setting('vec.dietas.decisor_persona_ref',true));
CREATE POLICY liquidacion_alta ON vec_dietas.liquidacion_economica
 FOR INSERT TO vec_dietas_propietario WITH CHECK (titular_persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE POLICY liquidacion_consulta_exacta ON vec_dietas.liquidacion_economica
 FOR SELECT TO vec_dietas_propietario USING (
  comision_ref=current_setting('vec.dietas.consulta_comision_ref',true)
  AND (current_setting('vec.dietas.consulta_version',true)='0'
       OR version_resultante::text=current_setting('vec.dietas.consulta_version',true)));
CREATE POLICY historia_liquidacion_lectura ON vec_dietas.historia_liquidacion_economica
 FOR SELECT TO vec_dietas_propietario USING (titular_persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE POLICY historia_liquidacion_alta ON vec_dietas.historia_liquidacion_economica
 FOR INSERT TO vec_dietas_propietario WITH CHECK (titular_persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE POLICY outbox_liquidacion_lectura ON vec_dietas.outbox_liquidacion_economica
 FOR SELECT TO vec_dietas_propietario USING (titular_persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE POLICY outbox_liquidacion_alta ON vec_dietas.outbox_liquidacion_economica
 FOR INSERT TO vec_dietas_propietario WITH CHECK (titular_persona_ref=current_setting('vec.dietas.persona_ref',true));

-- Alta sin versión elige solo si hay una publicación inequívoca. Edición
-- conserva la versión del expediente. Los tres grupos deben compartir la
-- misma regla y huella bajo esa versión, también cuando se elige un vehículo.
CREATE FUNCTION vec_dietas.consultar_tarifa_publicada_v1(
 p_version_tarifa_ref text,p_fecha date,p_grupo smallint,p_vehiculo text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' AS $tarifa$
DECLARE elegida vec_dietas.catalogo_liquidacion_publicado%ROWTYPE;
 referencia_elegida text; version_elegida text;
 numero integer; grupos integer; reglas integer; huellas integer; publicaciones integer; normas_distintas integer;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor de tarifa Dietas incompatible' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'on'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'transacción de tarifa Dietas incompatible' USING ERRCODE='25000'; END IF;
 IF p_fecha IS NULL OR p_grupo NOT BETWEEN 1 AND 3
    OR p_vehiculo NOT IN ('propio','sin_vehiculo_propio')
    OR (p_version_tarifa_ref<>'' AND p_version_tarifa_ref !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$')
 THEN RAISE EXCEPTION 'selección de tarifa Dietas incompatible' USING ERRCODE='22023'; END IF;
 SELECT count(*), min(x.referencia), min(x.version) INTO numero,referencia_elegida,version_elegida
 FROM vec_dietas.catalogo_liquidacion_publicado x
 WHERE (p_version_tarifa_ref='' OR x.version_tarifa_ref=p_version_tarifa_ref)
   AND x.pais_iso2='ES' AND x.grupo=p_grupo AND x.vehiculo=p_vehiculo
   AND x.fecha_publicacion<=current_date AND p_fecha BETWEEN x.vigente_desde AND x.vigente_hasta;
 IF numero<>1 THEN RAISE EXCEPTION 'tarifa publicada no disponible' USING ERRCODE='PD004'; END IF;
 SELECT * INTO STRICT elegida FROM vec_dietas.catalogo_liquidacion_publicado x
 WHERE x.referencia=referencia_elegida AND x.version=version_elegida;
 SELECT count(*),count(DISTINCT x.grupo),count(DISTINCT x.regla_ref),
  count(DISTINCT x.regla_huella_sha256),count(DISTINCT x.publicacion_ref),count(DISTINCT x.normas)
 INTO numero,grupos,reglas,huellas,publicaciones,normas_distintas
 FROM vec_dietas.catalogo_liquidacion_publicado x
 WHERE x.version_tarifa_ref=elegida.version_tarifa_ref AND x.pais_iso2='ES'
   AND x.vehiculo=p_vehiculo AND x.fecha_publicacion<=current_date
   AND p_fecha BETWEEN x.vigente_desde AND x.vigente_hasta;
 IF numero<>3 OR grupos<>3 OR reglas<>1 OR huellas<>1 OR publicaciones<>1 OR normas_distintas<>1
    OR octet_length(elegida.catalogo_canonico)>262144
 THEN RAISE EXCEPTION 'tarifa publicada incompleta' USING ERRCODE='PD004'; END IF;
 RETURN elegida.catalogo_canonico;
END $tarifa$;
REVOKE ALL ON FUNCTION vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text) TO vec_dietas_ejecutor;

-- Calcula los tramos civiles desde horas, fechas y configuración publicada.
-- El alojamiento permanece cerrado hasta que exista su regla competente.
CREATE FUNCTION vec_dietas.calcular_tramos_publicados_v1(
 p_inicio date,p_hora_inicio text,p_fin date,p_hora_fin text,p_base_centimos bigint,
 p_version text,p_regla text,p_huella text,p_cfg jsonb)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
AS $tramos$
DECLARE n integer; dia date; porcentaje integer; importe bigint; suma bigint:=0;
 tramos jsonb:='[]'::jsonb; salida timestamp with time zone; regreso timestamp with time zone;
BEGIN
 IF p_inicio IS NULL OR p_fin IS NULL OR p_hora_inicio IS NULL OR p_hora_fin IS NULL
    OR p_fin<p_inicio OR p_fin-p_inicio>30
    OR p_hora_inicio !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR p_hora_fin !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR p_base_centimos NOT BETWEEN 1 AND 1000000
    OR p_version !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
    OR p_regla !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
    OR p_huella !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_cfg) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_cfg) k) IS DISTINCT FROM
       ARRAY['alojamiento','dias_maximos','duracion_minima_mismo_dia_horas',
        'esquema','hora_regreso_50_despues_de','hora_regreso_mismo_dia_despues_de',
        'hora_salida_100_antes_de','hora_salida_50_antes_de','porcentaje_intermedio',
        'porcentaje_mismo_dia','porcentaje_regreso','porcentaje_salida_media',
        'porcentaje_salida_temprana','zona']::text[]
    OR p_cfg->>'esquema' IS DISTINCT FROM 'nacional_ordinaria_v1'
    OR p_cfg->>'zona' IS DISTINCT FROM 'Europe/Madrid'
    OR p_cfg->>'alojamiento' IS DISTINCT FROM 'cerrado_pendiente_rrhh'
    OR (p_cfg->>'dias_maximos')::integer NOT BETWEEN 1 AND 31
    OR (p_cfg->>'duracion_minima_mismo_dia_horas')::integer NOT BETWEEN 1 AND 24
    OR EXISTS (SELECT 1 FROM jsonb_each_text(p_cfg) x(clave,valor)
      WHERE (x.clave LIKE 'hora_%' AND (x.valor !~ '^(0|[1-9][0-9]?)$' OR x.valor::integer NOT BETWEEN 0 AND 23))
         OR (x.clave LIKE 'porcentaje_%' AND (x.valor !~ '^(0|[1-9][0-9]{0,2})$' OR x.valor::integer NOT BETWEEN 0 AND 100)))
    OR (p_cfg->>'hora_salida_100_antes_de')::integer >= (p_cfg->>'hora_salida_50_antes_de')::integer
    OR (p_cfg->>'hora_regreso_50_despues_de')::integer >= (p_cfg->>'hora_regreso_mismo_dia_despues_de')::integer
 THEN RETURN NULL; END IF;
 salida:=(p_inicio+p_hora_inicio::time) AT TIME ZONE 'Europe/Madrid';
 regreso:=(p_fin+p_hora_fin::time) AT TIME ZONE 'Europe/Madrid';
 IF (salida AT TIME ZONE 'Europe/Madrid') IS DISTINCT FROM
      p_inicio+p_hora_inicio::time
    OR (regreso AT TIME ZONE 'Europe/Madrid') IS DISTINCT FROM
      p_fin+p_hora_fin::time
    OR ((salida+interval '1 hour') AT TIME ZONE 'Europe/Madrid')=
      p_inicio+p_hora_inicio::time
    OR ((salida-interval '1 hour') AT TIME ZONE 'Europe/Madrid')=
      p_inicio+p_hora_inicio::time
    OR ((regreso+interval '1 hour') AT TIME ZONE 'Europe/Madrid')=
      p_fin+p_hora_fin::time
    OR ((regreso-interval '1 hour') AT TIME ZONE 'Europe/Madrid')=
      p_fin+p_hora_fin::time
    OR regreso<=salida OR p_fin-p_inicio+1>(p_cfg->>'dias_maximos')::integer
 THEN RETURN NULL; END IF;
 FOR n IN 0..p_fin-p_inicio LOOP
  dia:=p_inicio+n; porcentaje:=0;
  IF p_inicio=p_fin THEN
   IF regreso-salida >= make_interval(hours=>(p_cfg->>'duracion_minima_mismo_dia_horas')::integer)
      AND extract(hour FROM p_hora_inicio::time)<(p_cfg->>'hora_salida_100_antes_de')::integer
      AND (extract(hour FROM p_hora_fin::time)>(p_cfg->>'hora_regreso_mismo_dia_despues_de')::integer
           OR (extract(hour FROM p_hora_fin::time)=(p_cfg->>'hora_regreso_mismo_dia_despues_de')::integer
               AND extract(minute FROM p_hora_fin::time)>0))
   THEN porcentaje:=(p_cfg->>'porcentaje_mismo_dia')::integer; END IF;
  ELSIF dia=p_inicio THEN
   IF extract(hour FROM p_hora_inicio::time)<(p_cfg->>'hora_salida_100_antes_de')::integer
   THEN porcentaje:=(p_cfg->>'porcentaje_salida_temprana')::integer;
   ELSIF (extract(hour FROM p_hora_inicio::time)>(p_cfg->>'hora_salida_100_antes_de')::integer
          OR (extract(hour FROM p_hora_inicio::time)=(p_cfg->>'hora_salida_100_antes_de')::integer
              AND extract(minute FROM p_hora_inicio::time)>0))
      AND extract(hour FROM p_hora_inicio::time)<(p_cfg->>'hora_salida_50_antes_de')::integer
   THEN porcentaje:=(p_cfg->>'porcentaje_salida_media')::integer; END IF;
  ELSIF dia=p_fin THEN
   IF extract(hour FROM p_hora_fin::time)>(p_cfg->>'hora_regreso_50_despues_de')::integer
      OR (extract(hour FROM p_hora_fin::time)=(p_cfg->>'hora_regreso_50_despues_de')::integer
          AND extract(minute FROM p_hora_fin::time)>0)
   THEN porcentaje:=(p_cfg->>'porcentaje_regreso')::integer; END IF;
  ELSE porcentaje:=(p_cfg->>'porcentaje_intermedio')::integer;
  END IF;
  IF porcentaje>0 THEN
   importe:=(p_base_centimos*porcentaje+99)/100;
   suma:=suma+importe;
   tramos:=tramos||jsonb_build_array(jsonb_build_object(
    'fecha',dia::text,'tipo','manutencion','porcentaje',porcentaje,
    'importe_centimos',importe,'version_tarifa_ref',p_version,
    'rotulo','dietas_tarifa_publicada'));
  END IF;
 END LOOP;
 RETURN jsonb_build_object('tramos',tramos,'manutencion_centimos',suma,
  'alojamiento_tope_centimos',0,'total_maximo_orientativo_centimos',suma,
  'version_tarifa_ref',p_version,'rotulo','dietas_tarifa_publicada',
  'regla_ref',p_regla,'regla_huella_sha256',p_huella);
EXCEPTION WHEN others THEN RETURN NULL;
END $tramos$;
REVOKE ALL ON FUNCTION vec_dietas.calcular_tramos_publicados_v1(date,text,date,text,bigint,text,text,text,jsonb) FROM PUBLIC;

-- La preimagen firmada no sustituye el cotejo económico. Cada línea se
-- recalcula contra el documento durable y la publicación exacta antes de
-- registrar el acto y avanzar la cola, dentro de la misma transacción V3.
CREATE FUNCTION vec_dietas.cotejar_economia_publicada_v1(
 p_economia jsonb,p_documento jsonb,p_catalogo jsonb,
 p_inicio date,p_hora_inicio text,p_fin date,p_hora_fin text)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
AS $cotejo$
DECLARE lineas jsonb; doc_linea jsonb; linea jsonb; regla jsonb; tramos jsonb;
 original bigint; calculado bigint; reconocido bigint; rechazado bigint;
 total_original bigint:=0; total_calculado bigint:=0; total_reconocido bigint:=0;
 total_rechazado bigint:=0; subtotal_dieta bigint:=0; subtotal_km bigint:=0;
 subtotal_otros bigint:=0; subtotal_diferencia bigint:=0;
 base bigint; km_escalados bigint:=0; km_actual bigint; km_base bigint;
 ajuste bigint; numero integer; i integer; tramos_leidos integer:=0;
 rutas_leidas integer:=0; concepto text; motivo text;
BEGIN
 IF jsonb_typeof(p_economia) IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_documento) IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_catalogo) IS DISTINCT FROM 'object'
    OR p_economia->'decision_pendiente' IS DISTINCT FROM 'false'::jsonb
    OR p_economia->>'fecha_inicio' IS DISTINCT FROM p_inicio::text
    OR p_economia->>'hora_inicio' IS DISTINCT FROM p_hora_inicio
    OR p_economia->>'fecha_fin' IS DISTINCT FROM p_fin::text
    OR p_economia->>'hora_fin' IS DISTINCT FROM p_hora_fin
    OR p_economia->'documento' IS DISTINCT FROM p_documento
    OR p_economia->'catalogo' IS DISTINCT FROM p_catalogo
    OR p_documento->>'version_tarifa_aceptada' IS DISTINCT FROM p_catalogo->>'version_tarifa_ref'
    OR p_documento->>'grupo_dieta' IS DISTINCT FROM p_catalogo->>'grupo'
    OR p_documento->>'vehiculo_propio' IS DISTINCT FROM
       CASE WHEN p_catalogo->>'vehiculo'='propio' THEN 'true' ELSE 'false' END
    OR p_documento->>'alojamiento_tope_centimos' IS DISTINCT FROM '0'
    OR jsonb_typeof(p_documento->'lineas') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p_documento->'tramos_aceptados') IS DISTINCT FROM 'array'
    OR jsonb_typeof(p_economia->'lineas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_documento->'lineas') NOT BETWEEN 1 AND 102
    OR jsonb_array_length(p_economia->'lineas')<>jsonb_array_length(p_documento->'lineas')
    OR p_inicio<(p_catalogo->>'vigente_desde')::date
    OR p_fin>(p_catalogo->>'vigente_hasta')::date
 THEN RETURN false; END IF;
 SELECT count(*),max((r->>'tope_centimos')::bigint)
 INTO numero,base FROM jsonb_array_elements(p_catalogo->'reglas') r
 WHERE r->>'tipo'='dieta' AND r->>'concepto'='manutencion'
   AND r->>'grupo'=p_documento->>'grupo_dieta';
 IF numero<>1 OR base NOT BETWEEN 1 AND 1000000 THEN RETURN false; END IF;
 tramos:=vec_dietas.calcular_tramos_publicados_v1(p_inicio,p_hora_inicio,p_fin,p_hora_fin,
   base,p_catalogo->>'version_tarifa_ref',
   p_catalogo->'regla_devengo'->>'referencia',
   p_catalogo->'regla_devengo'->>'huella_sha256',
   p_catalogo->'regla_devengo'->'configuracion');
 IF tramos IS NULL THEN RETURN false; END IF;
 lineas:=p_economia->'lineas';
 FOR i IN 0..jsonb_array_length(lineas)-1 LOOP
  linea:=lineas->i; doc_linea:=p_documento->'lineas'->i;
  original:=(doc_linea->>'importe_centimos')::bigint;
  IF original<0 OR original>100000000 THEN RETURN false; END IF;
  concepto:=doc_linea->>'concepto';
  IF doc_linea->>'tipo'='kilometraje' THEN concepto:='vehiculo_propio';
  ELSIF doc_linea->>'tipo' IN ('otro_medio','otro_gasto') THEN
   concepto:=doc_linea->>'tipo_gasto'; END IF;
  SELECT count(*) INTO numero
  FROM jsonb_array_elements(p_catalogo->'reglas') r
  WHERE r->>'tipo'=doc_linea->>'tipo' AND r->>'concepto'=concepto;
  IF numero<>1 THEN RETURN false; END IF;
  SELECT r INTO regla FROM jsonb_array_elements(p_catalogo->'reglas') r
  WHERE r->>'tipo'=doc_linea->>'tipo' AND r->>'concepto'=concepto;
  IF jsonb_typeof(linea) IS DISTINCT FROM 'object'
     OR jsonb_typeof(doc_linea->'importe_centimos') IS DISTINCT FROM 'number'
     OR jsonb_typeof(linea->'indice') IS DISTINCT FROM 'number'
     OR jsonb_typeof(linea->'original_centimos') IS DISTINCT FROM 'number'
     OR jsonb_typeof(linea->'reconocido_propuesto_centimos') IS DISTINCT FROM 'number'
     OR jsonb_typeof(linea->'rechazado_centimos') IS DISTINCT FROM 'number'
     OR regla->>'referencia' IS DISTINCT FROM linea->>'regla_ref'
     OR linea->>'indice' IS DISTINCT FROM i::text
     OR linea->>'original_centimos' IS DISTINCT FROM original::text
  THEN RETURN false; END IF;
  CASE doc_linea->>'tipo'
  WHEN 'dieta' THEN
   IF doc_linea->>'concepto' IS DISTINCT FROM 'manutencion'
      OR doc_linea->>'grupo' IS DISTINCT FROM p_catalogo->>'grupo'
      OR doc_linea->>'version_tarifa_ref' IS DISTINCT FROM p_catalogo->>'version_tarifa_ref'
      OR doc_linea->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
      OR doc_linea->>'indice_tramo' IS DISTINCT FROM
         p_documento->'tramos_aceptados'->>tramos_leidos
      OR (doc_linea->>'indice_tramo')::integer NOT BETWEEN 0 AND 61
      OR doc_linea->>'fecha' IS DISTINCT FROM
         tramos->'tramos'->((doc_linea->>'indice_tramo')::integer)->>'fecha'
   THEN RETURN false; END IF;
   calculado:=(tramos->'tramos'->((doc_linea->>'indice_tramo')::integer)->>'importe_centimos')::bigint;
   tramos_leidos:=tramos_leidos+1; subtotal_dieta:=subtotal_dieta+original;
  WHEN 'kilometraje' THEN
   IF p_catalogo->>'vehiculo' IS DISTINCT FROM 'propio'
      OR doc_linea->>'ruta_indice' IS DISTINCT FROM (rutas_leidas+1)::text
      OR rutas_leidas>=8
      OR doc_linea->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
      OR doc_linea->>'kilometros_base' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
      OR doc_linea->>'ajuste_kilometros' !~ '^-?(0|[1-9][0-9]{0,3})\.[0-9]{4}$'
      OR doc_linea->>'origen_codigo' IS NULL OR doc_linea->>'destino_codigo' IS NULL
      OR doc_linea->>'origen_codigo'=doc_linea->>'destino_codigo'
      OR doc_linea->>'version_tarifa_ref' IS DISTINCT FROM p_catalogo->>'version_tarifa_ref'
      OR doc_linea->>'rotulo' IS DISTINCT FROM 'dietas_tarifa_publicada'
   THEN RETURN false; END IF;
   km_actual:=((doc_linea->>'kilometros')::numeric*10000)::bigint;
   km_base:=((doc_linea->>'kilometros_base')::numeric*10000)::bigint;
   ajuste:=((doc_linea->>'ajuste_kilometros')::numeric*10000)::bigint;
   IF km_actual<1 OR km_base<1 OR km_actual<>km_base+ajuste
      OR km_escalados+km_actual>100000000
      OR (ajuste=0 AND doc_linea->>'motivo_ajuste' IS NOT NULL)
      OR (ajuste<>0 AND (length(coalesce(doc_linea->>'motivo_ajuste','')) NOT BETWEEN 3 AND 500
        OR doc_linea->>'motivo_ajuste' IS DISTINCT FROM btrim(doc_linea->>'motivo_ajuste')))
      OR (regla->>'centimos_por_km')::bigint NOT BETWEEN 1 AND 99
   THEN RETURN false; END IF;
   km_escalados:=km_escalados+km_actual;
   calculado:=(km_actual*(regla->>'centimos_por_km')::bigint+5000)/10000;
   rutas_leidas:=rutas_leidas+1; subtotal_km:=subtotal_km+original;
  WHEN 'otro_medio','otro_gasto' THEN
   IF doc_linea->>'catalogo_version' IS DISTINCT FROM
        p_catalogo->>'catalogo_otros_gastos_version_ref'
      OR (doc_linea->>'justificante_ref' ~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$') IS NOT TRUE
      OR (doc_linea->>'justificante_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
      OR (doc_linea->>'fecha' ~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$') IS NOT TRUE
      OR (doc_linea->>'fecha')::date NOT BETWEEN p_inicio AND p_fin
      OR (regla->>'tope_centimos')::bigint<1
   THEN RETURN false; END IF;
   calculado:=least(original,(regla->>'tope_centimos')::bigint);
   subtotal_otros:=subtotal_otros+original;
  ELSE RETURN false;
  END CASE;
  reconocido:=(linea->>'reconocido_propuesto_centimos')::bigint;
  rechazado:=(linea->>'rechazado_centimos')::bigint;
  motivo:=linea->>'motivo_codigo';
  IF (calculado=0 AND linea ? 'calculado_publicado_centimos')
     OR (calculado>0 AND jsonb_typeof(linea->'calculado_publicado_centimos') IS DISTINCT FROM 'number')
     OR coalesce(linea->>'calculado_publicado_centimos','0') IS DISTINCT FROM calculado::text
     OR (linea ? 'diferencia_centimos' AND
         jsonb_typeof(linea->'diferencia_centimos') IS DISTINCT FROM 'number')
     OR coalesce((linea->>'diferencia_centimos')::bigint,0) IS DISTINCT FROM calculado-original
     OR (calculado=original AND linea ? 'diferencia_centimos')
     OR (calculado<>original AND NOT linea ? 'diferencia_centimos')
     OR reconocido NOT BETWEEN 0 AND calculado
     OR rechazado IS DISTINCT FROM calculado-reconocido
     OR (reconocido<>original AND (motivo ~ '^[a-z][a-z0-9_]{2,79}$') IS NOT TRUE)
     OR (reconocido=original AND motivo IS DISTINCT FROM '')
     OR linea-(ARRAY['indice','regla_ref','original_centimos','calculado_publicado_centimos',
       'diferencia_centimos','reconocido_propuesto_centimos','rechazado_centimos',
       'motivo_codigo'])<>'{}'::jsonb
  THEN RETURN false; END IF;
  total_original:=total_original+original;
  total_calculado:=total_calculado+calculado;
  total_reconocido:=total_reconocido+reconocido;
  total_rechazado:=total_rechazado+rechazado;
 END LOOP;
 subtotal_diferencia:=total_calculado-total_original;
 IF tramos_leidos<>jsonb_array_length(p_documento->'tramos_aceptados')
    OR subtotal_dieta IS DISTINCT FROM (p_documento->>'manutencion_centimos')::bigint
    OR subtotal_km IS DISTINCT FROM (p_documento->>'kilometraje_centimos')::bigint
    OR subtotal_otros IS DISTINCT FROM (p_documento->>'otros_centimos')::bigint
    OR total_original IS DISTINCT FROM (p_documento->>'total_orientativo_centimos')::bigint
    OR (p_catalogo->>'vehiculo'='propio' AND rutas_leidas=0)
    OR (p_catalogo->>'vehiculo'<>'propio' AND rutas_leidas<>0)
    OR p_economia->'totales' IS DISTINCT FROM
      (jsonb_build_object('original_centimos',total_original,
        'reconocido_propuesto_centimos',total_reconocido,
        'rechazado_centimos',total_rechazado)
       ||CASE WHEN total_calculado=0 THEN '{}'::jsonb ELSE
         jsonb_build_object('calculado_publicado_centimos',total_calculado) END
       ||CASE WHEN subtotal_diferencia=0 THEN '{}'::jsonb ELSE
         jsonb_build_object('diferencia_centimos',subtotal_diferencia) END)
 THEN RETURN false; END IF;
 RETURN true;
EXCEPTION WHEN others THEN RETURN false;
END $cotejo$;
REVOKE ALL ON FUNCTION vec_dietas.cotejar_economia_publicada_v1(jsonb,jsonb,jsonb,date,text,date,text) FROM PUBLIC;

-- El acto nominal consume la concesión antes de abrir filas de Dietas.
CREATE FUNCTION vec_dietas.registrar_liquidacion_economica_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; e jsonb; cap jsonb; d jsonb; ctx jsonb; v record;
 q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; anterior vec_dietas.comision_revision%ROWTYPE;
 numero vec_dietas.numero_documento_comision%ROWTYPE;
 cat vec_dietas.catalogo_liquidacion_publicado%ROWTYPE;
 existente vec_dietas.liquidacion_economica%ROWTYPE;
 econ_text text; econ_pre text; cat_pre text; doc_text text; sem_text text;
 fecha_text text; ref text; recibo text; evento text; outbox text; instante timestamptz(6);
 constancia text; instantanea_pre text; instantanea_text text; snapshot text;
 version_nueva bigint; max_version bigint; huella text; amb text; atr text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_setting('statement_timeout')='0'
    OR current_setting('idle_in_transaction_session_timeout')='0'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
  econ_text:=(p_material::json->'economia')::text;
  econ_pre:=convert_from(decode(m->>'preimagen_economia','base64'),'UTF8');
  cat_pre:=convert_from(decode(m->>'preimagen_catalogo','base64'),'UTF8');
  doc_text:=convert_from(decode(m->>'documento_canonico','base64'),'UTF8');
  sem_text:=convert_from(decode(m->>'semantica_canonica','base64'),'UTF8');
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; c:=m->'comando'; e:=m->'economia';
 IF jsonb_typeof(m)<>'object' OR m->>'esquema' IS DISTINCT FROM 'vec.dietas.liquidacion-economica.v1'
    OR m->>'operacion' IS DISTINCT FROM 'dietas.liquidacion_economica.registrar'
    OR m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR m->>'unidad_ref' !~ '^[A-Za-z0-9:_-]{3,128}$'
    OR i->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR i->>'actor_ref' IS NULL OR i->>'perfil_ref' IS NULL
    OR i->>'contexto_actor_ref' IS DISTINCT FROM ctx->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM ctx->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM ctx->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM ctx->>'cuenta_version'
    OR i->>'persona_ref' IS DISTINCT FROM ctx->>'persona_ref'
    OR i->>'persona_ref' IS DISTINCT FROM ctx->>'principal_ref'
    OR i->>'perfil_ref' IS DISTINCT FROM ctx->>'perfil_activo_ref'
    OR i->>'persona_version' IS DISTINCT FROM ctx->>'persona_version'
    OR i->>'perfil_version' IS DISTINCT FROM ctx->>'perfil_version'
    OR ctx->>'estado' IS DISTINCT FROM 'activo'
    OR cap->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex')
    OR cap->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR cap->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'liquidacion_economica'
    OR d->>'accion' IS DISTINCT FROM 'dietas.liquidacion_economica.registrar'
    OR d->>'finalidad' IS DISTINCT FROM 'registrar_liquidacion_economica_dietas'
    OR d->'campos_permitidos' IS DISTINCT FROM '["constancia","economia","esquema","snapshot_sha256"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR cap->>'operacion' IS DISTINCT FROM 'dietas.liquidacion_economica.registrar'
    OR cap->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.liquidacion_economica.registrar.v1'
    OR cap->>'efecto_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
    OR c->>'etapa' IS DISTINCT FROM 'liquidacion'
    OR c->>'decision' IS DISTINCT FROM 'aprobar'
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR c->>'version_esperada' !~ '^[1-9][0-9]{0,17}$'
    OR c->>'motivo' IS NULL OR c->>'motivo' IS DISTINCT FROM btrim(c->>'motivo')
    OR length(c->>'motivo')>600 OR c->>'motivo'~'[[:cntrl:]]'
    OR e->>'esquema' IS DISTINCT FROM 'vec_dietas_preparacion_liquidacion_publicada_v1'
    OR e->>'comision_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR e->>'comision_version' IS DISTINCT FROM c->>'version_esperada'
    OR e->>'snapshot_sha256' !~ '^[0-9a-f]{64}$'
    OR e->>'documento_sha256' !~ '^[0-9a-f]{64}$'
    OR e->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
    OR e->'catalogo'->>'huella_sha256' IS DISTINCT FROM e->>'catalogo_sha256'
    OR e->'catalogo'->>'referencia' IS DISTINCT FROM e->>'catalogo_ref'
    OR e->'catalogo'->>'version' IS DISTINCT FROM e->>'catalogo_version'
    OR e->>'fecha_inicio' IS NULL OR e->>'fecha_fin' IS NULL
    OR e->'documento' IS DISTINCT FROM doc_text::jsonb
    OR e->'lineas' IS NULL OR jsonb_typeof(e->'lineas')<>'array'
    OR jsonb_array_length(e->'lineas') NOT BETWEEN 1 AND 102
    OR e->'totales' IS NULL OR jsonb_typeof(e->'totales')<>'object'
    OR econ_pre::jsonb IS DISTINCT FROM (e||jsonb_build_object('snapshot_sha256',''))
    OR encode(sha256(convert_to(econ_pre,'UTF8')),'hex') IS DISTINCT FROM e->>'snapshot_sha256'
    OR cat_pre::jsonb IS DISTINCT FROM (e->'catalogo'||jsonb_build_object('huella_sha256',''))
    OR encode(sha256(convert_to(cat_pre,'UTF8')),'hex') IS DISTINCT FROM e->>'catalogo_sha256'
    OR encode(sha256(convert_to(doc_text,'UTF8')),'hex') IS DISTINCT FROM e->>'documento_sha256'
    OR sem_text::jsonb IS DISTINCT FROM jsonb_build_object('esquema',m->>'esquema',
       'persona_ref',i->>'persona_ref','comando',c,'economia',e)
    OR encode(sha256(convert_to(sem_text,'UTF8')),'hex') IS DISTINCT FROM m->>'huella_semantica'
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'material de liquidación incompatible' USING ERRCODE='PD003'; END IF;
 amb:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')
   ||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(m->>'unidad_ref')||'}';
 atr:='"contexto_actor_ref":'||vec_dietas.cadena_json_go_v1(i->>'contexto_actor_ref')
  ||',"contexto_version":'||vec_dietas.cadena_json_go_v1(i->>'contexto_version')
  ||',"cuenta_ref":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_ref')
  ||',"cuenta_version":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_version')
  ||',"etapa":"liquidacion"'
  ||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"'
  ||',"operacion":'||vec_dietas.cadena_json_go_v1(m->>'operacion')
  ||',"perfil_version":'||vec_dietas.cadena_json_go_v1(i->>'perfil_version')
  ||',"persona_version":'||vec_dietas.cadena_json_go_v1(i->>'persona_version')
  ||',"recurso_ref":'||vec_dietas.cadena_json_go_v1(m->>'recurso_ref');
 huella:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":{'||atr||'}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
    OR cap->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'efecto de liquidación incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'consumo de liquidación incompatible' USING ERRCODE='PD003'; END IF;

 -- La clave es propia del decisor, incluso si el recurso del segundo intento
 -- ya no está en cola. La concesión actual se ha consumido antes de consultarla.
 PERFORM set_config('vec.dietas.decisor_persona_ref',i->>'persona_ref',true);
 SELECT * INTO existente FROM vec_dietas.liquidacion_economica l
  WHERE l.decisor_persona_ref=i->>'persona_ref' AND l.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF existente.comision_ref IS DISTINCT FROM m->>'recurso_ref'
     OR existente.unidad_ref IS DISTINCT FROM m->>'unidad_ref'
     OR existente.version_entrada IS DISTINCT FROM (c->>'version_esperada')::bigint
     OR existente.comando IS DISTINCT FROM c
     OR existente.economia IS DISTINCT FROM e
     OR existente.huella_semantica_sha256 IS DISTINCT FROM m->>'huella_semantica'
  THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  RETURN existente.instantanea;
 END IF;
 IF v.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'consumo repetido sin recibo Dietas' USING ERRCODE='PD003'; END IF;

 PERFORM set_config('vec.dietas.circuito_etapa','liquidacion',true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 SELECT * INTO q FROM vec_dietas.cola_circuito_comision cola
  WHERE cola.comision_ref=m->>'recurso_ref' AND cola.version=(c->>'version_esperada')::bigint
    AND cola.etapa='liquidacion' AND cola.unidad_ref=m->>'unidad_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'documento Dietas no disponible' USING ERRCODE='PD004'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
 SELECT * INTO b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
 IF NOT FOUND OR b.persona_ref IS DISTINCT FROM q.persona_ref OR b.unidad_ref IS DISTINCT FROM q.unidad_ref
    OR i->>'persona_ref'=q.persona_ref THEN
  RAISE EXCEPTION 'competencia Dietas incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT anterior FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref AND version=q.version FOR UPDATE;
 SELECT max(version) INTO max_version FROM vec_dietas.comision_revision WHERE comision_ref=q.comision_ref;
 IF max_version IS DISTINCT FROM q.version OR anterior.estado IS DISTINCT FROM 'pendiente_liquidacion'
    OR anterior.asignacion_ref IS DISTINCT FROM q.asignacion_ref
    OR anterior.asignacion_version IS DISTINCT FROM q.asignacion_version
    OR anterior.documento IS NULL OR anterior.documento IS DISTINCT FROM e->'documento'
    OR anterior.fecha_inicio::text IS DISTINCT FROM e->>'fecha_inicio'
    OR anterior.hora_inicio IS DISTINCT FROM e->>'hora_inicio'
    OR anterior.fecha_fin::text IS DISTINCT FROM e->>'fecha_fin'
    OR anterior.hora_fin IS DISTINCT FROM e->>'hora_fin'
    OR anterior.grupo_dieta IS DISTINCT FROM (e->'documento'->>'grupo_dieta')::smallint
    OR anterior.vehiculo_propio IS DISTINCT FROM (e->'documento'->>'vehiculo_propio')::boolean
    OR anterior.calculo->>'version_tarifa' IS DISTINCT FROM e->'catalogo'->>'version_tarifa_ref'
 THEN RAISE EXCEPTION 'versión o documento Dietas incompatible' USING ERRCODE='PD005'; END IF;
 IF EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision h
   WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref')
 THEN RAISE EXCEPTION 'separación de funciones Dietas' USING ERRCODE='42501'; END IF;
 IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,b.unidad_ref,
    q.asignacion_ref,q.asignacion_version,anterior.grupo_dieta,anterior.centro_ref,
    anterior.administrativo_persona_ref,anterior.responsable_persona_ref,current_date) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación Personal no vigente' USING ERRCODE='P7201'; END IF;
 SELECT * INTO cat FROM vec_dietas.catalogo_liquidacion_publicado x
  WHERE x.referencia=e->>'catalogo_ref' AND x.version=e->>'catalogo_version'
    AND x.huella_sha256=e->>'catalogo_sha256';
 IF NOT FOUND OR cat.catalogo IS DISTINCT FROM e->'catalogo'
    OR cat.catalogo_canonico IS DISTINCT FROM convert_to((p_material::json->'economia'->'catalogo')::text,'UTF8')
    OR cat.publicacion_ref IS DISTINCT FROM e->'catalogo'->>'publicacion_ref'
    OR cat.normas IS DISTINCT FROM e->'catalogo'->'normas'
    OR cat.fecha_publicacion>current_date
    OR anterior.fecha_inicio<cat.vigente_desde OR anterior.fecha_fin>cat.vigente_hasta
    OR encode(sha256(convert_to(cat_pre,'UTF8')),'hex') IS DISTINCT FROM cat.huella_sha256
 THEN RAISE EXCEPTION 'catálogo publicado Dietas no disponible' USING ERRCODE='PD004'; END IF;
 IF vec_dietas.cotejar_economia_publicada_v1(e,anterior.documento,cat.catalogo,
     anterior.fecha_inicio,anterior.hora_inicio,anterior.fecha_fin,anterior.hora_fin) IS NOT TRUE
 THEN RAISE EXCEPTION 'cálculo de liquidación incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=q.comision_ref;
 version_nueva:=q.version+1;
 instante:=date_trunc('microseconds',clock_timestamp());
 fecha_text:=to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 ref:='liq_'||gen_random_uuid()::text;
 recibo:='rcd_'||gen_random_uuid()::text;
 evento:='hdi_'||md5(q.comision_ref||version_nueva::text||recibo);
 outbox:='odi_'||md5(q.comision_ref||version_nueva::text||recibo);
 constancia:='{"referencia":'||vec_dietas.cadena_json_go_v1(ref)
  ||',"clave_idempotencia":'||vec_dietas.cadena_json_go_v1(c->>'clave_idempotencia')
  ||',"numero_documento":'||vec_dietas.cadena_json_go_v1(numero.numero_documento)
  ||',"version_entrada":'||q.version::text
  ||',"version_resultante":'||version_nueva::text
  ||',"titular_persona_ref":'||vec_dietas.cadena_json_go_v1(q.persona_ref)
  ||',"decisor_persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')
  ||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(q.unidad_ref)
  ||',"fecha":'||vec_dietas.cadena_json_go_v1(fecha_text)
  ||',"recibo_ref":'||vec_dietas.cadena_json_go_v1(recibo)
  ||',"auditoria_ref":'||vec_dietas.cadena_json_go_v1(v.auditoria_ref)
  ||',"evento_ref":'||vec_dietas.cadena_json_go_v1(evento)||'}';
 instantanea_pre:='{"esquema":"vec_dietas_liquidacion_economica_v1","constancia":'
   ||constancia||',"economia":'||econ_text||',"snapshot_sha256":""}';
 snapshot:=encode(sha256(convert_to(instantanea_pre,'UTF8')),'hex');
 instantanea_text:=left(instantanea_pre,length(instantanea_pre)-2)||snapshot||'"}';
 INSERT INTO vec_dietas.comision_revision
 (comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,
  codigos_ruta,vehiculo_propio,rutas,calculo,documento,regla_ref,asignacion_ref,
  asignacion_version,grupo_dieta,centro_ref,administrativo_persona_ref,
  responsable_persona_ref,registrada_en)
 VALUES(q.comision_ref,version_nueva,'pendiente_fiscalizacion',anterior.fecha_inicio,anterior.fecha_fin,
  anterior.hora_inicio,anterior.hora_fin,anterior.motivo,anterior.codigos_ruta,
  anterior.vehiculo_propio,anterior.rutas,anterior.calculo,anterior.documento,
  anterior.regla_ref,anterior.asignacion_ref,anterior.asignacion_version,
  anterior.grupo_dieta,anterior.centro_ref,anterior.administrativo_persona_ref,
  anterior.responsable_persona_ref,instante);
 INSERT INTO vec_dietas.liquidacion_economica
 (referencia,comision_ref,version_entrada,version_resultante,titular_persona_ref,
  decisor_persona_ref,unidad_ref,numero_documento,clave_idempotencia,
  huella_semantica_sha256,comando,economia_canonica,economia,instantanea_canonica,
  instantanea,snapshot_sha256,catalogo_ref,catalogo_version,catalogo_sha256,lineas,
  totales,publicacion_ref,recibo_ref,auditoria_ref,evento_ref,decision_ref,
  consumo_huella_sha256,registrada_en)
 VALUES(ref,q.comision_ref,q.version,version_nueva,q.persona_ref,i->>'persona_ref',q.unidad_ref,
  numero.numero_documento,c->>'clave_idempotencia',m->>'huella_semantica',c,
  convert_to(econ_text,'UTF8'),e,convert_to(instantanea_text,'UTF8'),instantanea_text::jsonb,
  snapshot,e->>'catalogo_ref',e->>'catalogo_version',e->>'catalogo_sha256',e->'lineas',
  e->'totales',cat.publicacion_ref,recibo,v.auditoria_ref,evento,v.decision_ref,
  v.consumo_huella_sha256,instante);
 INSERT INTO vec_dietas.historia_liquidacion_economica
 (evento_ref,comision_ref,version_entrada,version_resultante,recibo_ref,
  titular_persona_ref,actor_ref,estado_anterior,estado_nuevo,registrada_en)
 VALUES(evento,q.comision_ref,q.version,version_nueva,recibo,q.persona_ref,i->>'actor_ref',
  'pendiente_liquidacion','pendiente_fiscalizacion',instante);
 INSERT INTO vec_dietas.outbox_liquidacion_economica
 (evento_ref,liquidacion_ref,comision_ref,version_resultante,titular_persona_ref,
  destinatario_unidad_ref,destinatario_etapa,estado,registrada_en)
 VALUES(outbox,ref,q.comision_ref,version_nueva,q.persona_ref,q.unidad_ref,
  'fiscalizacion','pendiente',instante);
 INSERT INTO vec_dietas.cola_circuito_comision
 (comision_ref,version,etapa,persona_ref,unidad_ref,asignacion_ref,
  asignacion_version,destinatario_persona_ref,fecha_inicio,fecha_fin,registrada_en)
 VALUES(q.comision_ref,version_nueva,'fiscalizacion',q.persona_ref,q.unidad_ref,
  q.asignacion_ref,q.asignacion_version,NULL,q.fecha_inicio,q.fecha_fin,instante);
 RETURN instantanea_text::jsonb;
END $funcion$;
ALTER FUNCTION vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_ejecutor;

-- La respuesta nominal sale solo después del asiento común de lectura.
-- La unidad y el titular se completan desde la liquidación inmutable, nunca
-- desde los parámetros del cliente ni desde una cabecera HTTP.
CREATE FUNCTION vec_dietas.consultar_liquidacion_economica_v1(
 p_comision_ref text,p_version_resultante bigint,p_contexto bytea,p_vinculo bytea,p_orden jsonb
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' AS $lectura$
#variable_conflict use_variable
DECLARE l vec_dietas.liquidacion_economica%ROWTYPE; orden jsonb; aud jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_setting('statement_timeout')='0'
    OR current_setting('idle_in_transaction_session_timeout')='0'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 IF p_comision_ref IS NULL OR p_comision_ref !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR p_version_resultante IS NULL
    OR (p_version_resultante<>0 AND p_version_resultante<2)
    OR p_contexto IS NULL OR octet_length(p_contexto)>65536
    OR p_vinculo IS NULL OR octet_length(p_vinculo)>16384
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>33
    OR p_orden ? 'unidad_ref' OR p_orden ? 'titular_persona_ref'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM p_comision_ref
    OR p_orden->>'accion' IS DISTINCT FROM 'dietas.liquidacion_economica.consultar'
 THEN RAISE EXCEPTION 'consulta Dietas inválida' USING ERRCODE='22023'; END IF;
 PERFORM set_config('vec.dietas.consulta_comision_ref',p_comision_ref,true);
 PERFORM set_config('vec.dietas.consulta_version',p_version_resultante::text,true);
 SELECT * INTO l FROM vec_dietas.liquidacion_economica x
  WHERE x.comision_ref=p_comision_ref
    AND (p_version_resultante=0 OR x.version_resultante=p_version_resultante)
  ORDER BY x.version_resultante DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'liquidación Dietas no disponible' USING ERRCODE='PD004'; END IF;
 orden:=p_orden||jsonb_build_object('unidad_ref',l.unidad_ref,
  'titular_persona_ref',l.titular_persona_ref);
 SELECT vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(
  p_contexto,p_vinculo,orden) INTO STRICT aud;
 IF jsonb_typeof(aud) IS DISTINCT FROM 'object'
    OR aud->>'auditoria_ref' !~ '^aud_v3_ld_[0-9a-f]{32}$'
    OR aud->>'auditoria_ref' IS DISTINCT FROM ('aud_v3_ld_'||substr(orden->>'intento_ref',9,32))
    OR aud->>'huella_sha256' !~ '^[0-9a-f]{64}$'
    OR aud->>'correlacion_ref' IS DISTINCT FROM orden->>'correlacion_ref'
    OR aud->>'secuencia' !~ '^[1-9][0-9]*$'
    OR aud->>'registrada_en' IS NULL
 THEN RAISE EXCEPTION 'asiento de lectura Dietas incompatible' USING ERRCODE='PD003'; END IF;
 RETURN jsonb_build_object('esquema','vec.dietas.consulta-liquidacion-economica.v1',
  'liquidacion',l.instantanea,'auditoria',aud);
END $lectura$;
ALTER FUNCTION vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb) TO vec_dietas_ejecutor;

-- La ruta antigua conserva su recibo histórico; nuevas aprobaciones en esta
-- etapa requieren la operación económica nominal y su catálogo publicado.
-- Definición explícita tomada de pg_get_functiondef en la copia PG18.4.
CREATE OR REPLACE FUNCTION vec_dietas.decidir_comision_v2(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; cap jsonb; d jsonb; ctx jsonb;
 v record; q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; anterior vec_dietas.comision_revision%ROWTYPE;
 recibo vec_dietas.recibo_operacion_comision%ROWTYPE;
 regla_catalogo jsonb; regla_fin jsonb; regla_huella text;
 etapa text; estado_previo text; estado_nuevo text; etapa_nueva text; destino text;
 huella text; operacion text; ahora timestamptz(6); version_nueva bigint;
 recibo_ref text; evento_ref text; outbox_ref text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; c:=m->'comando'; etapa:=c->>'etapa';
 estado_previo:=CASE etapa WHEN 'revision' THEN 'enviado_pendiente_revision'
  WHEN 'autorizacion' THEN 'pendiente_autorizacion'
  WHEN 'liquidacion' THEN 'pendiente_liquidacion'
  WHEN 'fiscalizacion' THEN 'pendiente_fiscalizacion' END;
 etapa_nueva:=CASE etapa WHEN 'revision' THEN 'autorizacion'
  WHEN 'autorizacion' THEN 'liquidacion'
  WHEN 'liquidacion' THEN 'fiscalizacion' END;
 estado_nuevo:=CASE WHEN c->>'decision'='devolver' THEN 'devuelta'
  ELSE CASE etapa WHEN 'revision' THEN 'pendiente_autorizacion'
   WHEN 'autorizacion' THEN 'pendiente_liquidacion'
   WHEN 'liquidacion' THEN 'pendiente_fiscalizacion'
   WHEN 'fiscalizacion' THEN 'fiscalizada' END END;
 IF estado_previo IS NULL OR estado_nuevo IS NULL OR m->>'operacion' IS DISTINCT FROM 'decidir'
    OR c->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
    OR c->>'decision' NOT IN ('aprobar','devolver')
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR c->>'version_esperada' !~ '^[1-9][0-9]{0,17}$'
    OR c->>'motivo' IS NULL OR length(c->>'motivo')>600
    OR c->>'motivo' IS DISTINCT FROM btrim(c->>'motivo')
    OR c->>'motivo' ~ '[[:cntrl:]]'
    OR (c->>'decision'='devolver' AND length(c->>'motivo')<3)
 THEN RAISE EXCEPTION 'decisión Dietas inválida' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(concat_ws(chr(31),i->>'persona_ref',m->>'recurso_ref',
  c->>'unidad_ref',etapa,c->>'decision',c->>'motivo',c->>'clave_idempotencia',
  c->>'version_esperada'),'UTF8')),'hex');
 IF m->>'huella_semantica' IS DISTINCT FROM huella
    OR vec_dietas.cotejar_efecto_circuito_v2(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'sello Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 Dietas incompatible' USING ERRCODE='PD003'; END IF;
 -- Solo tras el consumo nominal se abre la cola de la etapa y unidad exactas.
 PERFORM set_config('vec.dietas.circuito_etapa',etapa,true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 SELECT * INTO q FROM vec_dietas.cola_circuito_comision cola
 WHERE cola.comision_ref=m->>'recurso_ref' AND cola.etapa=etapa
    AND cola.unidad_ref=m->>'unidad_ref'
    AND cola.version=(c->>'version_esperada')::bigint;
 IF NOT FOUND THEN RAISE EXCEPTION 'documento Dietas no disponible' USING ERRCODE='PD004'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
 SELECT * INTO STRICT b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
 SELECT * INTO STRICT anterior FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref AND version=q.version FOR UPDATE;
 IF b.unidad_ref IS DISTINCT FROM q.unidad_ref
    OR anterior.estado IS DISTINCT FROM estado_previo
    OR anterior.asignacion_ref IS DISTINCT FROM q.asignacion_ref
    OR anterior.asignacion_version IS DISTINCT FROM q.asignacion_version
    OR anterior.administrativo_persona_ref IS NULL OR anterior.responsable_persona_ref IS NULL
 THEN RAISE EXCEPTION 'asignación Dietas incompatible' USING ERRCODE='PD003'; END IF;
 operacion:='circuito_'||etapa;
 -- La misma clave del mismo actor devuelve el recibo original aunque la
 -- asignación haya cambiado después: el efecto ya se registró y no se repite.
 SELECT * INTO recibo FROM vec_dietas.recibo_operacion_comision r
  WHERE r.persona_ref=q.persona_ref AND r.operacion=operacion
    AND r.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF recibo.comision_ref IS DISTINCT FROM q.comision_ref
     OR recibo.huella_semantica_sha256 IS DISTINCT FROM huella
     OR recibo.actor_ref IS DISTINCT FROM i->>'actor_ref'
  THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  SELECT estado INTO STRICT estado_nuevo FROM vec_dietas.comision_revision
   WHERE comision_ref=q.comision_ref AND version=recibo.version;
  RETURN jsonb_build_object('comision',jsonb_build_object('referencia',q.comision_ref,
   'estado',estado_nuevo,'version',recibo.version),'recibo',jsonb_build_object(
   'referencia',recibo.referencia,'version',recibo.version,
   'registrado_en',to_char(recibo.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',true));
 END IF;
 IF etapa='liquidacion' AND c->>'decision'='aprobar' THEN
  RAISE EXCEPTION 'liquidación económica nominal requerida' USING ERRCODE='PD005'; END IF;
 IF i->>'persona_ref'=q.persona_ref
    OR (etapa='revision' AND i->>'persona_ref' IS DISTINCT FROM anterior.administrativo_persona_ref)
    OR (etapa='autorizacion' AND i->>'persona_ref' IS DISTINCT FROM anterior.responsable_persona_ref)
    OR (q.destinatario_persona_ref IS NOT NULL
      AND q.destinatario_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision h
      WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref'
        AND h.tipo<>operacion)
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_liquidacion_economica h
      WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref')
 THEN RAISE EXCEPTION 'separación de funciones Dietas' USING ERRCODE='42501'; END IF;
 SELECT max(version) INTO version_nueva FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref;
 IF version_nueva IS DISTINCT FROM q.version THEN
  RAISE EXCEPTION 'versión Dietas desactualizada' USING ERRCODE='PD005';
 END IF;
 regla_catalogo:=vec_dietas.consultar_regla_devengo_dietas_v1(
  anterior.calculo->>'version_tarifa',anterior.fecha_inicio,'ES','nacional_ordinaria');
 regla_fin:=vec_dietas.consultar_regla_devengo_dietas_v1(
  anterior.calculo->>'version_tarifa',anterior.fecha_fin,'ES','nacional_ordinaria');
 regla_huella:=regla_catalogo->>'huella_sha256';
 IF anterior.regla_ref IS NULL
    OR anterior.regla_ref IS DISTINCT FROM anterior.calculo->>'regla_ref'
    OR anterior.regla_ref IS DISTINCT FROM regla_catalogo->>'regla_ref'
    OR anterior.regla_ref IS DISTINCT FROM regla_fin->>'regla_ref'
    OR regla_huella !~ '^[0-9a-f]{64}$'
    OR regla_huella IS DISTINCT FROM anterior.calculo->>'regla_huella_sha256'
    OR regla_huella IS DISTINCT FROM regla_fin->>'huella_sha256'
    OR regla_catalogo->>'version_tarifa_ref' IS DISTINCT FROM anterior.calculo->>'version_tarifa'
    OR regla_fin->>'version_tarifa_ref' IS DISTINCT FROM anterior.calculo->>'version_tarifa'
 THEN RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD003'; END IF;
 IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,
   b.unidad_ref,q.asignacion_ref,q.asignacion_version,anterior.grupo_dieta,
   anterior.centro_ref,anterior.administrativo_persona_ref,
   anterior.responsable_persona_ref,current_date) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación Personal no vigente' USING ERRCODE='P7201'; END IF;
 version_nueva:=q.version+1;
 ahora:=date_trunc('microseconds',clock_timestamp());
 recibo_ref:='rcd_'||gen_random_uuid()::text;
 evento_ref:='hdi_'||md5(q.comision_ref||version_nueva::text||recibo_ref);
 outbox_ref:='odi_'||md5(q.comision_ref||version_nueva::text||recibo_ref);
 INSERT INTO vec_dietas.comision_revision
 (comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,
  codigos_ruta,vehiculo_propio,rutas,calculo,documento,regla_ref,asignacion_ref,asignacion_version,
  grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
 VALUES (q.comision_ref,version_nueva,estado_nuevo,anterior.fecha_inicio,anterior.fecha_fin,
  anterior.hora_inicio,anterior.hora_fin,anterior.motivo,anterior.codigos_ruta,
  anterior.vehiculo_propio,anterior.rutas,anterior.calculo,anterior.documento,
  anterior.regla_ref,anterior.asignacion_ref,
  anterior.asignacion_version,anterior.grupo_dieta,anterior.centro_ref,
  anterior.administrativo_persona_ref,anterior.responsable_persona_ref,ahora);
 INSERT INTO vec_dietas.recibo_operacion_comision
 (referencia,comision_ref,version,operacion,clave_idempotencia,comando,
  huella_semantica_sha256,decision_ref,consumo_huella_sha256,auditoria_ad3_ref,
  actor_ref,persona_ref,regla_ref,regla_huella_sha256,registrada_en)
 VALUES (recibo_ref,q.comision_ref,version_nueva,operacion,c->>'clave_idempotencia',c,
  huella,v.decision_ref,v.consumo_huella_sha256,v.auditoria_ref,
  i->>'actor_ref',q.persona_ref,anterior.regla_ref,regla_huella,ahora);
 INSERT INTO vec_dietas.historia_operacion_comision
 (evento_ref,comision_ref,version,recibo_ref,estado_anterior,estado_nuevo,tipo,motivo,actor_ref,registrada_en)
 VALUES(evento_ref,q.comision_ref,version_nueva,recibo_ref,estado_previo,estado_nuevo,
  operacion,NULLIF(c->>'motivo',''),i->>'actor_ref',ahora);
 IF estado_nuevo IN ('devuelta','fiscalizada') THEN
  destino:=q.persona_ref;
 ELSIF estado_nuevo='pendiente_autorizacion' THEN
  destino:=anterior.responsable_persona_ref;
 ELSE destino:=NULL; END IF;
 INSERT INTO vec_dietas.outbox_comision
 (evento_ref,comision_ref,version,tipo,persona_ref,destinatario_persona_ref,
  destinatario_unidad_ref,destinatario_etapa,estado,registrada_en)
 VALUES(outbox_ref,q.comision_ref,version_nueva,'circuito_'||estado_nuevo,q.persona_ref,
  destino,CASE WHEN destino IS NULL THEN q.unidad_ref ELSE NULL END,
  CASE WHEN destino IS NULL THEN etapa_nueva ELSE NULL END,'pendiente',ahora);
 IF etapa_nueva IS NOT NULL AND estado_nuevo<>'devuelta' THEN
  INSERT INTO vec_dietas.cola_circuito_comision
  (comision_ref,version,etapa,persona_ref,unidad_ref,asignacion_ref,
   asignacion_version,destinatario_persona_ref,fecha_inicio,fecha_fin,registrada_en)
  VALUES(q.comision_ref,version_nueva,etapa_nueva,q.persona_ref,q.unidad_ref,
   q.asignacion_ref,q.asignacion_version,destino,q.fecha_inicio,q.fecha_fin,ahora);
 END IF;
 RETURN jsonb_build_object('comision',jsonb_build_object('referencia',q.comision_ref,
  'estado',estado_nuevo,'version',version_nueva),'recibo',jsonb_build_object(
  'referencia',recibo_ref,'version',version_nueva,
  'registrado_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'repeticion',false));
END $function$
;

DO $post$ BEGIN
 IF NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
       'vec_dietas.calcular_tramos_publicados_v1(date,text,date,text,bigint,text,text,text,jsonb)'::regprocedure
       AND p.proowner='vec_dietas_propietario'::regrole AND p.prosecdef
       AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario}'
       AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC']::text[])
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=
       'vec_dietas.cotejar_economia_publicada_v1(jsonb,jsonb,jsonb,date,text,date,text)'::regprocedure
       AND p.proowner='vec_dietas_propietario'::regrole AND p.prosecdef
       AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario}'
       AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC']::text[])
    OR EXISTS (
      SELECT 1 FROM pg_proc p WHERE p.oid IN (
       'vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
       'vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb)'::regprocedure,
       'vec_dietas.consultar_tarifa_publicada_v1(text,date,smallint,text)'::regprocedure)
       AND (p.proowner<>'vec_dietas_propietario'::regrole OR NOT p.prosecdef
        OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
        OR p.proconfig NOT @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']))
    OR (SELECT p.prosecdef IS NOT TRUE OR p.proowner<>'vec_dietas_propietario'::regrole
        OR p.proconfig NOT @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
        FROM pg_proc p WHERE p.oid='vec_dietas.registrar_liquidacion_economica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
    OR (SELECT p.prosecdef IS NOT TRUE OR p.proowner<>'vec_dietas_propietario'::regrole
        OR p.proconfig NOT @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
        FROM pg_proc p WHERE p.oid='vec_dietas.consultar_liquidacion_economica_v1(text,bigint,bytea,bytea,jsonb)'::regprocedure)
    OR (SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
        WHERE n.nspname='vec_dietas' AND c.relname IN
         ('catalogo_liquidacion_publicado','version_tarifa_registrada','regla_devengo_registrada',
          'liquidacion_economica','historia_liquidacion_economica','outbox_liquidacion_economica')
          AND c.relrowsecurity AND c.relforcerowsecurity AND c.relowner='vec_dietas_propietario'::regrole)<>6
    OR encode(sha256(convert_to(pg_get_functiondef('vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
       <> 'dc35e709187438bbee663ef6662e2d8dd36f70fd758a85e4db6f90526360b96c'
    OR (SELECT p.proowner<>'vec_dietas_propietario'::regrole OR p.prosecdef IS NOT TRUE
        OR p.proacl::text IS DISTINCT FROM '{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'
        OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
        FROM pg_proc p WHERE p.oid='vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
 THEN RAISE EXCEPTION 'Dietas 000013: PARO postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
