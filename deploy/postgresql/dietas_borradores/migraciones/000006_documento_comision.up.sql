\set ON_ERROR_STOP on
-- D2-D5: versiones del documento del empleado. Las tarifas y reglas siguen
-- marcadas provisionales; este corte no acredita liquidación ni entrega.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000006:documento:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regclass('vec_dietas.calculo_comision') IS NULL
    OR to_regclass('vec_dietas.regla_devengo_provisional') IS NOT NULL
    OR to_regclass('vec_dietas.numero_documento_comision') IS NOT NULL
    OR to_regclass('vec_dietas.comision_revision') IS NOT NULL
    OR to_regprocedure('vec_dietas.rechazar_mutacion_borrador_v1()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000006: faltan preimagen, AD3-59 o Personal-12' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_dietas.borrador_comision IN SHARE ROW EXCLUSIVE MODE;

-- Número interno visible, global y estable. No se le atribuye numeración
-- oficial de RRHH. La fecha de apertura procede de la alta original v1.
CREATE SEQUENCE vec_dietas.numero_documento_comision_seq AS bigint START WITH 1 NO CYCLE;
CREATE TABLE vec_dietas.numero_documento_comision (
 comision_ref text PRIMARY KEY REFERENCES vec_dietas.borrador_comision(referencia),
 secuencia bigint NOT NULL UNIQUE CHECK(secuencia>0),
 numero_documento text NOT NULL UNIQUE CHECK(numero_documento~'^VEC-D-[0-9]{4}-[0-9]{6,18}$'),
 fecha_apertura timestamptz(6) NOT NULL
);
ALTER SEQUENCE vec_dietas.numero_documento_comision_seq OWNER TO vec_dietas_propietario;
REVOKE ALL ON SEQUENCE vec_dietas.numero_documento_comision_seq FROM PUBLIC,vec_dietas_ejecutor;

-- El backfill requiere DBA porque la tabla base tiene FORCE RLS por persona.
-- No modifica ninguna fila histórica; materializa una referencia nueva.
RESET ROLE;
WITH ordenada AS (
 SELECT b.referencia,b.creada_en AS fecha_apertura,
        (b.creada_en AT TIME ZONE 'Europe/Madrid')::date AS fecha_local,
        row_number() OVER (ORDER BY b.creada_en,b.referencia) AS n
 FROM vec_dietas.borrador_comision b
)
INSERT INTO vec_dietas.numero_documento_comision
 (comision_ref,secuencia,numero_documento,fecha_apertura)
SELECT referencia,n,'VEC-D-'||to_char(fecha_local,'YYYY')||'-'||lpad(n::text,6,'0'),fecha_apertura
FROM ordenada;
SELECT setval('vec_dietas.numero_documento_comision_seq'::regclass,
  coalesce((SELECT max(secuencia) FROM vec_dietas.numero_documento_comision),1),
  EXISTS(SELECT 1 FROM vec_dietas.numero_documento_comision));
DO $numeracion$
BEGIN
 IF (SELECT count(*) FROM vec_dietas.numero_documento_comision)
    <> (SELECT count(*) FROM vec_dietas.borrador_comision) THEN
  RAISE EXCEPTION 'Dietas 000006: numeración histórica incompleta' USING ERRCODE='55000'; END IF;
END $numeracion$;
SET LOCAL ROLE vec_dietas_propietario;

CREATE FUNCTION vec_dietas.numerar_comision_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $$
DECLARE n bigint; fecha date;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER') THEN
  RAISE EXCEPTION 'numeración Dietas inválida' USING ERRCODE='42501'; END IF;
 n:=nextval('vec_dietas.numero_documento_comision_seq'::regclass);
 fecha:=(NEW.creada_en AT TIME ZONE 'Europe/Madrid')::date;
 INSERT INTO vec_dietas.numero_documento_comision
  (comision_ref,secuencia,numero_documento,fecha_apertura)
 VALUES(NEW.referencia,n,'VEC-D-'||to_char(fecha,'YYYY')||'-'||lpad(n::text,6,'0'),NEW.creada_en);
 RETURN NEW;
END $$;
ALTER FUNCTION vec_dietas.numerar_comision_v1() OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.numerar_comision_v1() FROM PUBLIC;
CREATE TRIGGER numerar_comision AFTER INSERT ON vec_dietas.borrador_comision
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.numerar_comision_v1();

CREATE TABLE vec_dietas.regla_devengo_provisional (
 regla_ref text PRIMARY KEY CHECK(regla_ref~'^provisional:regla:[a-z0-9:-]{8,120}$'),
 version_tarifa_ref text NOT NULL REFERENCES vec_dietas.version_tarifa_provisional(version_ref),
 pais_iso2 text NOT NULL CHECK(pais_iso2='ES'),
 variante text NOT NULL CHECK(variante='nacional_ordinaria'),
 configuracion jsonb NOT NULL CHECK(jsonb_typeof(configuracion)='object'),
 huella_sha256 text NOT NULL CHECK(huella_sha256~'^[0-9a-f]{64}$'),
 publicada_en timestamptz(6) NOT NULL,
 UNIQUE(version_tarifa_ref,pais_iso2,variante)
);
INSERT INTO vec_dietas.regla_devengo_provisional
 (regla_ref,version_tarifa_ref,pais_iso2,variante,configuracion,huella_sha256,publicada_en)
SELECT 'provisional:regla:nacional-ordinaria:20260923',v.version_ref,'ES','nacional_ordinaria',
 x.configuracion,encode(sha256(convert_to(x.configuracion::text,'UTF8')),'hex'),clock_timestamp()
FROM vec_dietas.version_tarifa_provisional v
CROSS JOIN (SELECT '{"regla":"nacional_ordinaria_provisional_v1","zona":"Europe/Madrid","duracion_minima_mismo_dia_horas":5,"hora_salida_100_antes_de":14,"hora_salida_50_antes_de":22,"hora_regreso_50_despues_de":14,"hora_regreso_mismo_dia_despues_de":16,"dias_maximos":31,"porcentaje_mismo_dia":50,"porcentaje_salida_temprana":100,"porcentaje_salida_media":50,"porcentaje_regreso":50,"porcentaje_intermedio":100,"porcentaje_alojamiento_tope":100,"alojamiento":"tope_pendiente_justificante","liquidable":false}'::jsonb AS configuracion) x
WHERE v.version_ref='provisional:rd462:20260923';
DO $regla$ BEGIN
 IF (SELECT count(*) FROM vec_dietas.regla_devengo_provisional)<>1 THEN
  RAISE EXCEPTION 'Dietas 000006: tarifa provisional ausente' USING ERRCODE='55000';
 END IF;
END $regla$;

CREATE TABLE vec_dietas.comision_revision (
 comision_ref text NOT NULL REFERENCES vec_dietas.borrador_comision(referencia),
 version bigint NOT NULL CHECK(version>=2),
 estado text NOT NULL CHECK(estado IN ('borrador','eliminado','enviado_pendiente_revision','pendiente_autorizacion','pendiente_liquidacion','pendiente_fiscalizacion','fiscalizada','devuelta')),
 fecha_inicio date NOT NULL, fecha_fin date NOT NULL CHECK(fecha_fin>=fecha_inicio),
 hora_inicio text NOT NULL CHECK(hora_inicio~'^([01][0-9]|2[0-3]):[0-5][0-9]$'),
 hora_fin text NOT NULL CHECK(hora_fin~'^([01][0-9]|2[0-3]):[0-5][0-9]$'),
 motivo text NOT NULL CHECK(length(motivo) BETWEEN 3 AND 600),
 codigos_ruta jsonb NOT NULL CHECK(jsonb_typeof(codigos_ruta)='array'),
 vehiculo_propio boolean, rutas jsonb,
 calculo jsonb NOT NULL CHECK(jsonb_typeof(calculo)='object'),
 documento jsonb CHECK(documento IS NULL OR jsonb_typeof(documento)='object'),
 regla_ref text NOT NULL REFERENCES vec_dietas.regla_devengo_provisional(regla_ref),
 asignacion_ref text, asignacion_version bigint, grupo_dieta smallint,
 centro_ref text, administrativo_persona_ref text, responsable_persona_ref text,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(comision_ref,version),
 CHECK((vehiculo_propio IS NULL AND rutas IS NULL)
    OR (vehiculo_propio IS NOT NULL AND jsonb_typeof(rutas)='array')),
 CHECK((asignacion_ref IS NULL AND asignacion_version IS NULL AND grupo_dieta IS NULL
        AND centro_ref IS NULL AND administrativo_persona_ref IS NULL AND responsable_persona_ref IS NULL)
       OR (asignacion_ref IS NOT NULL AND asignacion_version>0 AND grupo_dieta BETWEEN 1 AND 3
        AND centro_ref IS NOT NULL AND administrativo_persona_ref IS NOT NULL AND responsable_persona_ref IS NOT NULL))
);
CREATE INDEX comision_revision_estado_fecha_idx ON vec_dietas.comision_revision(estado,fecha_inicio,comision_ref,version DESC);

CREATE TABLE vec_dietas.recibo_regla_creacion_comision (
 comision_ref text PRIMARY KEY REFERENCES vec_dietas.borrador_comision(referencia),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_dietas.recibo_borrador_comision(referencia),
 regla_ref text NOT NULL REFERENCES vec_dietas.regla_devengo_provisional(regla_ref),
 regla_huella_sha256 text NOT NULL CHECK(regla_huella_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL
);

CREATE TABLE vec_dietas.recibo_operacion_comision (
 referencia text PRIMARY KEY CHECK(referencia~'^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
 comision_ref text NOT NULL, version bigint NOT NULL,
 operacion text NOT NULL CHECK(operacion~'^[a-z][a-z_]{2,79}$'),
 clave_idempotencia text NOT NULL CHECK(clave_idempotencia~'^[A-Za-z0-9_-]{16,128}$'),
 comando jsonb NOT NULL CHECK(jsonb_typeof(comando)='object'),
 huella_semantica_sha256 text NOT NULL CHECK(huella_semantica_sha256~'^[0-9a-f]{64}$'),
 decision_ref text NOT NULL, consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL, actor_ref text NOT NULL, persona_ref text NOT NULL,
 regla_ref text NOT NULL REFERENCES vec_dietas.regla_devengo_provisional(regla_ref),
 regla_huella_sha256 text NOT NULL CHECK(regla_huella_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(comision_ref,version), UNIQUE(persona_ref,operacion,clave_idempotencia),
 FOREIGN KEY(comision_ref,version) REFERENCES vec_dietas.comision_revision(comision_ref,version)
);
CREATE TABLE vec_dietas.historia_operacion_comision (
 evento_ref text PRIMARY KEY, comision_ref text NOT NULL, version bigint NOT NULL,
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_dietas.recibo_operacion_comision(referencia),
 estado_anterior text NOT NULL, estado_nuevo text NOT NULL, tipo text NOT NULL,
 motivo text, actor_ref text NOT NULL, registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(comision_ref,version) REFERENCES vec_dietas.comision_revision(comision_ref,version)
);
CREATE TABLE vec_dietas.outbox_comision (
 evento_ref text PRIMARY KEY, comision_ref text NOT NULL, version bigint NOT NULL,
 tipo text NOT NULL, persona_ref text NOT NULL, destinatario_persona_ref text,
 destinatario_unidad_ref text, destinatario_etapa text,
 estado text NOT NULL DEFAULT 'pendiente' CHECK(estado='pendiente'),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(comision_ref,version) REFERENCES vec_dietas.comision_revision(comision_ref,version),
 CHECK((destinatario_persona_ref IS NOT NULL AND destinatario_unidad_ref IS NULL AND destinatario_etapa IS NULL)
    OR (destinatario_persona_ref IS NULL AND destinatario_unidad_ref IS NOT NULL AND destinatario_etapa IS NOT NULL))
);

DO $seguridad$ DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['numero_documento_comision','regla_devengo_provisional','comision_revision','recibo_regla_creacion_comision','recibo_operacion_comision','historia_operacion_comision','outbox_comision'] LOOP
  EXECUTE format('ALTER TABLE vec_dietas.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_dietas.%I FORCE ROW LEVEL SECURITY',tabla);
  IF tabla='regla_devengo_provisional' THEN
   EXECUTE format('CREATE POLICY propietario_catalogo ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (true) WITH CHECK (true)',tabla);
  ELSIF tabla='recibo_operacion_comision' THEN
   EXECUTE format('CREATE POLICY persona_contextual ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (persona_ref=current_setting(''vec.dietas.persona_ref'',true)) WITH CHECK (persona_ref=current_setting(''vec.dietas.persona_ref'',true))',tabla);
  ELSE
   EXECUTE format('CREATE POLICY persona_contextual ON vec_dietas.%I FOR ALL TO vec_dietas_propietario USING (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=comision_ref AND b.persona_ref=current_setting(''vec.dietas.persona_ref'',true))) WITH CHECK (EXISTS (SELECT 1 FROM vec_dietas.borrador_comision b WHERE b.referencia=comision_ref AND b.persona_ref=current_setting(''vec.dietas.persona_ref'',true)))',tabla);
  END IF;
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_dietas.%I FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_dietas.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1()',tabla);
  EXECUTE format('REVOKE ALL ON vec_dietas.%I FROM PUBLIC,vec_dietas_ejecutor',tabla);
 END LOOP;
END $seguridad$;

-- Lectura interna nominal del catálogo. Una versión vacía resuelve una única
-- vigente; la ambigüedad falla cerrada. No se expone SQL ejecutable.
CREATE FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(
 p_version_tarifa_ref text,p_fecha date,p_pais_iso2 text,p_variante text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE r vec_dietas.regla_devengo_provisional%ROWTYPE; n int; fecha_elegida date;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT ((pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_dietas_migrador','MEMBER'))
      OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)) THEN
  RAISE EXCEPTION 'ejecutor catálogo Dietas inválido' USING ERRCODE='42501'; END IF;
 IF p_fecha IS NULL OR p_pais_iso2<>'ES' OR p_variante<>'nacional_ordinaria'
    OR p_version_tarifa_ref IS NULL
    OR (p_version_tarifa_ref<>'' AND p_version_tarifa_ref !~ '^provisional:[a-z0-9:-]{8,120}$') THEN
  RAISE EXCEPTION 'consulta regla Dietas inválida' USING ERRCODE='22023'; END IF;
 IF p_version_tarifa_ref='' THEN
  SELECT max(v.vigente_desde) INTO fecha_elegida
  FROM vec_dietas.regla_devengo_provisional x
  JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
  WHERE x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
    AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 END IF;
 SELECT count(*) INTO n FROM vec_dietas.regla_devengo_provisional x
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
 WHERE ((p_version_tarifa_ref='' AND v.vigente_desde=fecha_elegida)
    OR (p_version_tarifa_ref<>'' AND x.version_tarifa_ref=p_version_tarifa_ref))
   AND x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
   AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 IF n<>1 THEN RAISE EXCEPTION 'regla Dietas no disponible' USING ERRCODE='PD010'; END IF;
 SELECT x.* INTO STRICT r FROM vec_dietas.regla_devengo_provisional x
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=x.version_tarifa_ref
 WHERE ((p_version_tarifa_ref='' AND v.vigente_desde=fecha_elegida)
    OR (p_version_tarifa_ref<>'' AND x.version_tarifa_ref=p_version_tarifa_ref))
   AND x.pais_iso2=p_pais_iso2 AND x.variante=p_variante
   AND v.vigente_desde<=p_fecha AND (v.vigente_hasta IS NULL OR p_fecha<v.vigente_hasta);
 IF r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(r.configuracion::text,'UTF8')),'hex')
    OR ARRAY(SELECT jsonb_object_keys(r.configuracion) ORDER BY 1) IS DISTINCT FROM
       ARRAY['alojamiento','dias_maximos','duracion_minima_mismo_dia_horas',
        'hora_regreso_50_despues_de','hora_regreso_mismo_dia_despues_de',
        'hora_salida_100_antes_de','hora_salida_50_antes_de','liquidable',
        'porcentaje_alojamiento_tope','porcentaje_intermedio','porcentaje_mismo_dia',
        'porcentaje_regreso','porcentaje_salida_media','porcentaje_salida_temprana',
        'regla','zona']
    OR r.configuracion->>'regla'<>'nacional_ordinaria_provisional_v1'
    OR r.configuracion->>'zona'<>'Europe/Madrid'
    OR r.configuracion->>'alojamiento'<>'tope_pendiente_justificante'
    OR r.configuracion->>'liquidable'<>'false'
    OR (r.configuracion->>'dias_maximos')::int NOT BETWEEN 1 AND 31
    OR (r.configuracion->>'duracion_minima_mismo_dia_horas')::int NOT BETWEEN 1 AND 24
    OR EXISTS (SELECT 1 FROM jsonb_each_text(r.configuracion) h(clave,valor)
       WHERE h.clave LIKE 'hora_%' AND
         (h.valor !~ '^(0|[1-9][0-9]?)$' OR h.valor::int NOT BETWEEN 0 AND 23))
    OR EXISTS (SELECT 1 FROM jsonb_each_text(r.configuracion) p(clave,valor)
       WHERE p.clave LIKE 'porcentaje_%' AND
         (p.valor !~ '^(0|[1-9][0-9]{0,2})$' OR p.valor::int NOT BETWEEN 0 AND 100)) THEN
  RAISE EXCEPTION 'regla Dietas incoherente' USING ERRCODE='PD010'; END IF;
 RETURN jsonb_build_object('regla_ref',r.regla_ref,
  'version_tarifa_ref',r.version_tarifa_ref,'pais_iso2',r.pais_iso2,
  'variante',r.variante,'configuracion',r.configuracion,'huella_sha256',r.huella_sha256);
END $$;
ALTER FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text) TO vec_dietas_ejecutor;

-- Reproduce la huella de RecursoAutorizable de Go para el material v2.
-- AD3 verifica sello, concesión y vigencia; esta guarda impide sustituir el
-- material entre la decisión y su consumo. Ninguna GUC del cliente concede ACL.
CREATE FUNCTION vec_dietas.cotejar_recurso_documento_v2(p_material text,p_capacidad bytea,p_decision bytea,p_contexto bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; v jsonb; operacion text; accion text; finalidad text; audiencia text;
        amb text; atr text; h text; hasta text; hc text; campos jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN RETURN false; END IF;
 IF p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL THEN RETURN false; END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
       d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RETURN false; END;
 i:=m->'identidad'; operacion:=m->>'operacion';
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(i)<>'object'
    OR m->>'esquema'<>'vec.dietas.borrador-operacion.v2'
    OR vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) IS NOT TRUE THEN RETURN false; END IF;
 IF operacion IN ('editar','borrar','enviar') THEN
  IF m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
     OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref' THEN RETURN false; END IF;
  accion:='dietas.borrador.propio.'||operacion;
  finalidad:=operacion||'_borrador_propio';
  audiencia:='vec_dietas.borrador_propio.'||operacion||'.v1';
  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 ELSIF operacion IN ('detalle','lista') THEN
  IF (operacion='detalle' AND (m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$' OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref'))
     OR (operacion='lista' AND m->>'recurso_ref'<>'dietas:borradores:propios') THEN RETURN false; END IF;
  accion:='dietas.documento.propio.consultar'; finalidad:='consultar_documento_propio_dietas';
  audiencia:='vec_dietas.documento_propio.consultar.v1';
  campos:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
 ELSE RETURN false; END IF;
 IF d->>'esquema'<>'vec.autorizacion.decision.v3.solicitud-ligada.actor-v2'
    OR d->>'concedida'<>'true' OR d->>'bloque_version'<>'3'
    OR d->>'accion' IS DISTINCT FROM accion OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'modulo_id'<>'dietas' OR d->>'tipo_recurso'<>'comision_borrador'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'esquema'<>'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
    OR c->>'operacion' IS DISTINCT FROM accion
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN RETURN false; END IF;
 v:=d->'vinculo_autenticacion_actor'; hc:=encode(sha256(p_contexto),'hex');
 IF v->>'esquema'<>'vec.autorizacion.vinculo-autenticacion-actor.v2'
    OR v->>'autoridad_efectiva'<>'autoridad_maestra_acreditada'
    OR v->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR v->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR v->>'cuenta_ref' IS DISTINCT FROM i->>'cuenta_ref'
    OR v->>'contexto_actor_ref' IS DISTINCT FROM i->>'contexto_actor_ref'
    OR v->>'contexto_actor_version' IS DISTINCT FROM i->>'contexto_version'
    OR v->>'contexto_actor_cuenta_version' IS DISTINCT FROM i->>'cuenta_version'
    OR v->>'contexto_actor_huella_sha256' IS DISTINCT FROM hc
    OR c->>'contexto_ref' IS DISTINCT FROM v->>'registro_contexto_ref'
    OR c->>'huella_contexto_sha256' IS DISTINCT FROM hc
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref' THEN RETURN false; END IF;
 hasta:=coalesce(i->>'vigente_hasta',''); IF hasta='' THEN hasta:='sin_fin'; END IF;
 amb:='{"empleado_ref":'||(i->'empleado_ref')::text||',"persona_ref":'||(i->'persona_ref')::text||'}';
 atr:='{"contexto_actor_ref":'||(i->'contexto_actor_ref')::text||',"contexto_version":'||to_jsonb(i->>'contexto_version')::text||',"cuenta_ref":'||(i->'cuenta_ref')::text||',"cuenta_version":'||to_jsonb(i->>'cuenta_version')::text||',"fecha_referencia":'||(i->'fecha_referencia')::text||',"fuente_ref":'||(i->'fuente_ref')::text||',"fuente_version":'||to_jsonb(i->>'fuente_version')::text||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'","operacion":'||to_jsonb(operacion)::text||',"perfil_version":'||to_jsonb(i->>'perfil_version')::text||',"persona_version":'||to_jsonb(i->>'persona_version')::text||',"procedencia_acto_ref":'||(i->'procedencia_acto_ref')::text||',"recurso_ref":'||(m->'recurso_ref')::text||',"relacion_version":'||to_jsonb(i->>'relacion_version')::text||',"vigente_desde":'||(i->'vigente_desde')::text||',"vigente_hasta":'||to_jsonb(hasta)::text||'}';
 h:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 RETURN d->>'contexto_recurso_huella_sha256'=h AND c->>'huella_efecto_sha256'=h;
END $$;
ALTER FUNCTION vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.cotejar_recurso_documento_v2(text,bytea,bytea,bytea) FROM PUBLIC;

-- El comando aparece penúltimo en el struct JSON de Go. Su fragmento exacto
-- conserva escapes y orden de campos de encoding/json para cotejar la huella
-- semántica publicada, incluida la preconsulta sin cálculo.
CREATE FUNCTION vec_dietas.huella_semantica_mutacion_v2(p_material text) RETURNS text
LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE m jsonb; i jsonb; c jsonb; marcador text:=',"comando":'; sufijo text;
        inicio int; longitud int; bruto text; canon text;
BEGIN
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando';
 EXCEPTION WHEN others THEN RETURN NULL; END;
 IF jsonb_typeof(i) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR m->>'referencia' IS NULL OR i->>'relacion_version' !~ '^[1-9][0-9]*$' THEN RETURN NULL; END IF;
 sufijo:=',"referencia":'||vec_dietas.cadena_json_go_v1(m->>'referencia')||'}';
 inicio:=strpos(p_material,marcador);
 IF inicio=0 OR right(p_material,length(sufijo))<>sufijo
    OR strpos(substr(p_material,inicio+length(marcador)),marcador)>0 THEN RETURN NULL; END IF;
 longitud:=length(p_material)-length(sufijo)-inicio-length(marcador)+1;
 IF longitud<2 THEN RETURN NULL; END IF;
 bruto:=substr(p_material,inicio+length(marcador),longitud);
 BEGIN IF bruto::jsonb IS DISTINCT FROM c THEN RETURN NULL; END IF;
 EXCEPTION WHEN others THEN RETURN NULL; END;
 canon:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')||
  ',"empleado_ref":'||vec_dietas.cadena_json_go_v1(i->>'empleado_ref')||
  ',"relacion_ref":'||vec_dietas.cadena_json_go_v1(i->>'relacion_ref')||
  ',"unidad_ref":'||vec_dietas.cadena_json_go_v1(i->>'unidad_ref')||
  ',"relacion_version":'||(i->>'relacion_version')||
  ',"referencia":'||vec_dietas.cadena_json_go_v1(m->>'referencia')||
  ',"comando":'||bruto||'}';
 RETURN encode(sha256(convert_to(canon,'UTF8')),'hex');
EXCEPTION WHEN others THEN RETURN NULL;
END $$;
ALTER FUNCTION vec_dietas.huella_semantica_mutacion_v2(text) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.huella_semantica_mutacion_v2(text) FROM PUBLIC;

CREATE FUNCTION vec_dietas.validar_rutas_d4_v2(p_comando jsonb,p_tarifa numeric,
 p_version_ref text,p_rotulo text) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE calc jsonb:=p_comando->'calculo'; ruta jsonb; cr jsonb; tramo jsonb;
        rutas jsonb; calculadas jsonb; codigos jsonb; lineas jsonb:='[]'::jsonb;
        km_base numeric; km_final numeric; total_km numeric:=0; ajuste numeric;
        importe bigint; total_importe bigint:=0; n int; t int; motivo text;
BEGIN
 IF jsonb_typeof(p_comando->'vehiculo_propio') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(p_comando->'rutas') IS DISTINCT FROM 'array'
    OR jsonb_typeof(calc->'tramos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'tramos_ruta')<>0 THEN RETURN NULL; END IF;
 rutas:=p_comando->'rutas'; calculadas:=coalesce(calc->'rutas','[]'::jsonb);
 IF jsonb_typeof(calculadas)<>'array' THEN RETURN NULL; END IF;
 IF p_comando->>'vehiculo_propio'='false' THEN
  IF jsonb_array_length(rutas)<>0 OR jsonb_array_length(calculadas)<>0
     OR (calc ? 'vehiculo_propio' AND calc->>'vehiculo_propio' IS DISTINCT FROM 'false')
     OR calc->>'kilometros'<>'0.0000'
     OR (calc->>'importe_kilometraje_centimos')::bigint<>0 THEN RETURN NULL; END IF;
  RETURN jsonb_build_object('lineas','[]'::jsonb,'importe_centimos',0);
 END IF;
 IF calc->>'vehiculo_propio' IS DISTINCT FROM 'true'
    OR jsonb_array_length(rutas) NOT BETWEEN 1 AND 8
    OR jsonb_array_length(calculadas)<>jsonb_array_length(rutas) THEN RETURN NULL; END IF;
 FOR n IN 0..jsonb_array_length(rutas)-1 LOOP
  ruta:=rutas->n; cr:=calculadas->n; codigos:=ruta->'codigos_ruta';
  IF jsonb_typeof(codigos) IS DISTINCT FROM 'array' OR jsonb_array_length(codigos) NOT BETWEEN 2 AND 12
     OR ruta-(ARRAY['codigos_ruta','ajuste_kilometros','motivo_ajuste'])<>'{}'::jsonb
     OR cr->'codigos_ruta' IS DISTINCT FROM codigos
     OR jsonb_typeof(cr->'tramos_ruta') IS DISTINCT FROM 'array'
     OR jsonb_array_length(cr->'tramos_ruta')<>jsonb_array_length(codigos)-1
     OR length(coalesce(cr->>'version_grafo','')) NOT BETWEEN 1 AND 160
     OR cr->>'version_grafo' IS DISTINCT FROM calc->>'version_grafo'
     OR ruta->>'ajuste_kilometros' !~ '^-?(0|[1-9][0-9]{0,3})\.[0-9]{4}$'
     OR cr->>'ajuste_kilometros' IS DISTINCT FROM ruta->>'ajuste_kilometros'
     OR cr->>'motivo_ajuste' IS DISTINCT FROM ruta->>'motivo_ajuste'
     OR cr->>'kilometros_base' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
     OR cr->>'kilometros_finales' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
  THEN RETURN NULL; END IF;
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(codigos) x(valor)
      WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'string'
         OR x.valor #>> '{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$')
  THEN RETURN NULL; END IF;
  ajuste:=(ruta->>'ajuste_kilometros')::numeric;
  motivo:=ruta->>'motivo_ajuste';
  IF abs(ajuste)>1000 OR (ajuste=0 AND ruta->>'ajuste_kilometros'<>'0.0000')
     OR (ajuste=0 AND motivo<>'')
     OR (ajuste<>0 AND (length(coalesce(motivo,'')) NOT BETWEEN 3 AND 500
       OR motivo<>btrim(motivo) OR motivo~'[[:cntrl:]]')) THEN RETURN NULL; END IF;
  km_base:=0;
  FOR t IN 0..jsonb_array_length(cr->'tramos_ruta')-1 LOOP
   tramo:=cr->'tramos_ruta'->t;
   IF tramo->>'origen_codigo' IS DISTINCT FROM codigos->>t
      OR tramo->>'destino_codigo' IS DISTINCT FROM codigos->>(t+1)
      OR tramo->>'origen_codigo' IS NOT DISTINCT FROM tramo->>'destino_codigo'
      OR tramo->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
      OR (tramo->>'kilometros')::numeric<=0 THEN RETURN NULL; END IF;
   km_base:=km_base+(tramo->>'kilometros')::numeric;
  END LOOP;
  km_final:=km_base+ajuste;
  IF km_base IS DISTINCT FROM (cr->>'kilometros_base')::numeric
     OR km_final IS DISTINCT FROM (cr->>'kilometros_finales')::numeric
     OR km_final<=0 OR km_final>10000 THEN RETURN NULL; END IF;
  importe:=round(km_final*p_tarifa*100)::bigint;
  IF importe IS DISTINCT FROM (cr->>'importe_centimos')::bigint THEN RETURN NULL; END IF;
  total_km:=total_km+km_final; total_importe:=total_importe+importe;
  lineas:=lineas||jsonb_build_array(jsonb_build_object(
   'tipo','kilometraje','ruta_indice',n+1,
   'origen_codigo',codigos->>0,
   'destino_codigo',codigos->>(jsonb_array_length(codigos)-1),
   'kilometros',cr->>'kilometros_finales',
   'kilometros_base',cr->>'kilometros_base',
   'ajuste_kilometros',cr->>'ajuste_kilometros',
   'importe_centimos',importe,'version_grafo',cr->>'version_grafo',
   'version_tarifa_ref',p_version_ref,'rotulo',p_rotulo)||
   CASE WHEN ajuste=0 THEN '{}'::jsonb
        ELSE jsonb_build_object('motivo_ajuste',motivo) END);
 END LOOP;
 IF total_km>10000 OR total_km IS DISTINCT FROM (calc->>'kilometros')::numeric
    OR total_importe IS DISTINCT FROM (calc->>'importe_kilometraje_centimos')::bigint THEN RETURN NULL; END IF;
 RETURN jsonb_build_object('lineas',lineas,'importe_centimos',total_importe);
EXCEPTION WHEN others THEN RETURN NULL;
END $$;
ALTER FUNCTION vec_dietas.validar_rutas_d4_v2(jsonb,numeric,text,text) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.validar_rutas_d4_v2(jsonb,numeric,text,text) FROM PUBLIC;

-- El cálculo conserva tres alternativas provisionales. El documento acepta
-- tramos solo del grupo revalidado en Personal, con total singular. Los otros
-- gastos son declaraciones; una referencia/huella opcional no custodia fichero.
CREATE FUNCTION vec_dietas.validar_documento_v2(p_comando jsonb) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE calc jsonb; doc jsonb; linea jsonb; tramo jsonb; opcion jsonb; esperado jsonb:='[]'::jsonb;
        rutas_d4 jsonb; seleccion jsonb; regla_catalogo jsonb; regla_config jsonb;
        v_version_tarifa text; rotulo text; tarifa numeric; pct int; porcentajes int[];
        man numeric; alo numeric;
        importe bigint; suma_man bigint; suma_alo bigint; seleccionado_man bigint:=0;
        seleccionado_alo bigint:=0; km_centimos bigint:=0; otros_centimos bigint:=0;
        n int; grupo int; grupo_actual int; indice int; anterior_indice int:=-1; n_tramos int;
        otros int:=0; base_lineas int; fecha_inicio date; fecha_fin date;
BEGIN
 IF jsonb_typeof(p_comando)<>'object' OR jsonb_typeof(p_comando->'calculo')<>'object'
    OR jsonb_typeof(p_comando->'documento')<>'object' THEN RETURN false; END IF;
 calc:=p_comando->'calculo'; doc:=p_comando->'documento';
 IF jsonb_typeof(doc->'lineas') IS DISTINCT FROM 'array'
    OR doc->'vehiculo_propio' IS DISTINCT FROM p_comando->'vehiculo_propio'
    OR jsonb_array_length(doc->'lineas')>256
    OR jsonb_typeof(p_comando->'asignacion') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'tramos_aceptados') IS DISTINCT FROM 'array'
    OR p_comando->>'version_tarifa_aceptada' IS DISTINCT FROM calc->>'version_tarifa'
    OR doc->>'version_tarifa_aceptada' IS DISTINCT FROM calc->>'version_tarifa'
    OR doc->'tramos_aceptados' IS DISTINCT FROM p_comando->'tramos_aceptados'
    OR doc->>'grupo_dieta' IS DISTINCT FROM p_comando->'asignacion'->>'grupo_dieta'
    OR jsonb_typeof(calc->'opciones_dieta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'opciones_dieta')<>3
    OR jsonb_typeof(p_comando->'codigos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_comando->'codigos_ruta') NOT BETWEEN 2 AND 12
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(p_comando->'codigos_ruta') x(valor)
        WHERE jsonb_typeof(x.valor) IS DISTINCT FROM 'string'
           OR x.valor #>> '{}' !~ '^[A-Za-z0-9:_-]{1,64}$')
    OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(p_comando->'codigos_ruta') x(valor)
        GROUP BY x.valor HAVING count(*)>1)
    OR (p_comando->>'vehiculo_propio'='true' AND
        (calc->>'procedencia' IS DISTINCT FROM 'osrm_interno'
         OR calc->>'motor' IS DISTINCT FROM 'OSRM'
         OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160))
    OR (p_comando->>'vehiculo_propio'='false' AND
        (calc->>'procedencia' IS DISTINCT FROM 'sin_vehiculo_propio'
         OR calc->>'motor' IS DISTINCT FROM 'no_aplica'
         OR calc->>'version_grafo' IS DISTINCT FROM 'no_aplica'))
    OR calc->>'rotulo'<>'PROVISIONAL · pendiente de confirmación por RRHH'
    OR calc->>'version_tarifa' !~ '^provisional:[a-z0-9:-]{8,120}$'
    OR calc->>'hora_inicio' IS DISTINCT FROM p_comando->>'hora_inicio'
    OR calc->>'hora_fin' IS DISTINCT FROM p_comando->>'hora_fin'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$' THEN RETURN false; END IF;
 fecha_inicio:=(p_comando->>'fecha_inicio')::date;
 fecha_fin:=(p_comando->>'fecha_fin')::date;
 v_version_tarifa:=calc->>'version_tarifa'; rotulo:=calc->>'rotulo';
 regla_catalogo:=vec_dietas.consultar_regla_devengo_dietas_v1(
  v_version_tarifa,fecha_inicio,'ES','nacional_ordinaria');
 regla_config:=regla_catalogo->'configuracion';
 IF calc->>'regla_ref' IS DISTINCT FROM regla_catalogo->>'regla_ref'
    OR calc->>'regla_huella_sha256' IS DISTINCT FROM regla_catalogo->>'huella_sha256' THEN RETURN false; END IF;
 porcentajes:=ARRAY[(regla_config->>'porcentaje_mismo_dia')::int,
  (regla_config->>'porcentaje_salida_temprana')::int,
  (regla_config->>'porcentaje_salida_media')::int,
  (regla_config->>'porcentaje_regreso')::int,
  (regla_config->>'porcentaje_intermedio')::int];
 SELECT k.eur_por_km INTO STRICT tarifa FROM vec_dietas.importe_km_provisional k
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=k.version_ref
 WHERE k.version_ref=v_version_tarifa AND k.vehiculo='automovil'
   AND v.vigente_desde<=fecha_inicio AND (v.vigente_hasta IS NULL OR fecha_fin<v.vigente_hasta);
 IF tarifa IS DISTINCT FROM (calc->>'eur_por_km')::numeric THEN RETURN false; END IF;
 FOR grupo_actual IN 1..3 LOOP
  opcion:=calc->'opciones_dieta'->(grupo_actual-1);
  SELECT d.manutencion_eur,d.alojamiento_eur INTO STRICT man,alo
  FROM vec_dietas.importe_dieta_provisional d
  WHERE d.version_ref=v_version_tarifa AND d.pais_iso2='ES' AND d.grupo=grupo_actual;
  IF (opcion->>'grupo')::int IS DISTINCT FROM grupo_actual
     OR opcion->'calculo'->>'version_tarifa_ref' IS DISTINCT FROM v_version_tarifa
     OR opcion->'calculo'->>'rotulo' IS DISTINCT FROM rotulo
     OR jsonb_typeof(opcion->'calculo'->'tramos')<>'array'
     OR jsonb_array_length(opcion->'calculo'->'tramos')>62 THEN RETURN false; END IF;
  suma_man:=0; suma_alo:=0;
  FOR tramo IN SELECT value FROM jsonb_array_elements(opcion->'calculo'->'tramos') LOOP
   IF tramo->>'version_tarifa_ref' IS DISTINCT FROM v_version_tarifa
      OR tramo->>'rotulo' IS DISTINCT FROM rotulo
      OR tramo->>'fecha' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
      OR (tramo->>'fecha')::date NOT BETWEEN fecha_inicio AND fecha_fin THEN RETURN false; END IF;
   importe:=(tramo->>'importe_centimos')::bigint;
   pct:=(tramo->>'porcentaje')::int;
   IF tramo->>'tipo'='manutencion' AND pct=ANY(porcentajes)
      AND importe=ceil(man*100*pct/100)::bigint
      THEN suma_man:=suma_man+importe;
   ELSIF tramo->>'tipo'='alojamiento_tope_pendiente_justificante'
      AND pct=(regla_config->>'porcentaje_alojamiento_tope')::int
      AND importe=ceil(alo*100*pct/100)::bigint
      THEN suma_alo:=suma_alo+importe;
   ELSE RETURN false; END IF;
  END LOOP;
  IF suma_man IS DISTINCT FROM (opcion->'calculo'->>'manutencion_centimos')::bigint
     OR suma_alo IS DISTINCT FROM (opcion->'calculo'->>'alojamiento_tope_centimos')::bigint
     OR suma_man+suma_alo IS DISTINCT FROM (opcion->'calculo'->>'total_maximo_orientativo_centimos')::bigint
  THEN RETURN false; END IF;
 END LOOP;
 grupo:=(p_comando->'asignacion'->>'grupo_dieta')::int;
 IF grupo NOT BETWEEN 1 AND 3 THEN RETURN false; END IF;
 seleccion:=p_comando->'tramos_aceptados';
 opcion:=calc->'opciones_dieta'->(grupo-1); n_tramos:=jsonb_array_length(opcion->'calculo'->'tramos');
 IF (n_tramos=0 AND jsonb_array_length(seleccion)<>0)
    OR (n_tramos>0 AND jsonb_array_length(seleccion) NOT IN (1,n_tramos)) THEN RETURN false; END IF;
 FOR linea IN SELECT value FROM jsonb_array_elements(seleccion) LOOP
  IF jsonb_typeof(linea) IS DISTINCT FROM 'number'
     OR linea::text !~ '^(0|[1-9][0-9]*)$' THEN RETURN false; END IF;
  indice:=(linea::text)::int;
  IF indice<=anterior_indice OR indice>=n_tramos THEN RETURN false; END IF;
  anterior_indice:=indice;
  tramo:=opcion->'calculo'->'tramos'->indice;
  IF tramo->>'tipo'='manutencion' THEN seleccionado_man:=seleccionado_man+(tramo->>'importe_centimos')::bigint;
  ELSIF tramo->>'tipo'='alojamiento_tope_pendiente_justificante' THEN seleccionado_alo:=seleccionado_alo+(tramo->>'importe_centimos')::bigint;
  ELSE RETURN false; END IF;
  esperado:=esperado||jsonb_build_array(jsonb_build_object(
   'tipo','dieta','grupo',grupo,'indice_tramo',indice,
   'fecha',tramo->>'fecha','concepto',tramo->>'tipo',
   'importe_centimos',(tramo->>'importe_centimos')::bigint,
   'version_tarifa_ref',v_version_tarifa,'rotulo',rotulo));
 END LOOP;
 IF n_tramos>0 AND jsonb_array_length(seleccion)=n_tramos
    AND (SELECT jsonb_agg(to_jsonb(s.n) ORDER BY s.n) FROM generate_series(0,n_tramos-1) s(n))
        IS DISTINCT FROM seleccion THEN RETURN false; END IF;
 rutas_d4:=vec_dietas.validar_rutas_d4_v2(p_comando,tarifa,v_version_tarifa,rotulo);
 IF rutas_d4 IS NULL THEN RETURN false; END IF;
 esperado:=esperado||(rutas_d4->'lineas');
 km_centimos:=(rutas_d4->>'importe_centimos')::bigint;
 base_lineas:=jsonb_array_length(esperado);
 IF coalesce(jsonb_typeof(p_comando->'otros'),'array')<>'array'
    OR (p_comando ? 'otros' AND jsonb_array_length(p_comando->'otros')>32) THEN RETURN false; END IF;
 FOR linea IN SELECT x.value FROM jsonb_array_elements(doc->'lineas') WITH ORDINALITY x(value,ord)
   WHERE x.ord>base_lineas LOOP
  otros:=otros+1;
  IF otros>32 OR linea->>'tipo' NOT IN ('otro_medio','otro_gasto')
     OR linea->>'justificante_ref' IS NULL
     OR linea->>'justificante_sha256' IS NULL
     OR jsonb_typeof(linea->'importe_centimos') IS DISTINCT FROM 'number'
     OR length(coalesce(linea->>'concepto','')) NOT BETWEEN 3 AND 500
     OR linea->>'concepto'<>btrim(linea->>'concepto')
     OR linea->>'concepto'~'[[:cntrl:]]'
     OR NOT (
       (linea->>'justificante_ref'='' AND linea->>'justificante_sha256'='')
       OR (linea->>'justificante_ref' ~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'
           AND linea->>'justificante_sha256' ~ '^[0-9a-f]{64}$'))
     OR (linea->>'importe_centimos')::bigint NOT BETWEEN 1 AND 100000000
     OR linea IS DISTINCT FROM jsonb_build_object('tipo',linea->>'tipo',
       'concepto',linea->>'concepto','importe_centimos',(linea->>'importe_centimos')::bigint,
       'justificante_ref',linea->>'justificante_ref',
       'justificante_sha256',linea->>'justificante_sha256') THEN RETURN false; END IF;
  IF linea IS DISTINCT FROM p_comando->'otros'->(otros-1) THEN RETURN false; END IF;
  otros_centimos:=otros_centimos+(linea->>'importe_centimos')::bigint;
  esperado:=esperado||jsonb_build_array(linea);
 END LOOP;
 IF otros IS DISTINCT FROM coalesce(jsonb_array_length(p_comando->'otros'),0) THEN RETURN false; END IF;
 RETURN doc->'lineas' IS NOT DISTINCT FROM esperado
   AND (doc->>'manutencion_centimos')::bigint IS NOT DISTINCT FROM seleccionado_man
   AND (doc->>'alojamiento_tope_centimos')::bigint IS NOT DISTINCT FROM seleccionado_alo
   AND (doc->>'kilometraje_centimos')::bigint IS NOT DISTINCT FROM km_centimos
   AND (doc->>'otros_centimos')::bigint IS NOT DISTINCT FROM otros_centimos
   AND (doc->>'total_orientativo_centimos')::bigint IS NOT DISTINCT FROM
     seleccionado_man+seleccionado_alo+km_centimos+otros_centimos;
EXCEPTION WHEN others THEN RETURN false;
END $$;
ALTER FUNCTION vec_dietas.validar_documento_v2(jsonb) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.validar_documento_v2(jsonb) FROM PUBLIC;

-- Valida la instantánea de alta contra importes y regla publicados. Conserva
-- el contrato v1 de material/AD3, con porcentajes leídos del catálogo.
CREATE FUNCTION vec_dietas.validar_calculo_catalogado_v2(p_comando jsonb) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE calc jsonb; op jsonb; tr jsonb; regla jsonb; cfg jsonb;
        tarifa numeric; man numeric; alo numeric; km numeric:=0; importe bigint;
        suma_man bigint; suma_alo bigint; porcentajes int[]; pct int;
        v_tarifa text; inicio date; fin date; g int; n int;
BEGIN
 IF jsonb_typeof(p_comando) IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'calculo') IS DISTINCT FROM 'object'
    OR jsonb_typeof(p_comando->'codigos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_comando->'codigos_ruta') NOT BETWEEN 2 AND 12 THEN RETURN false; END IF;
 calc:=p_comando->'calculo';
 IF calc->>'procedencia' IS DISTINCT FROM 'osrm_interno'
    OR calc->>'motor' IS DISTINCT FROM 'OSRM'
    OR length(coalesce(calc->>'version_grafo','')) NOT BETWEEN 1 AND 160
    OR calc->>'rotulo' IS DISTINCT FROM 'PROVISIONAL · pendiente de confirmación por RRHH'
    OR calc->>'version_tarifa' !~ '^provisional:[a-z0-9:-]{8,120}$'
    OR calc->>'hora_inicio' IS DISTINCT FROM p_comando->>'hora_inicio'
    OR calc->>'hora_fin' IS DISTINCT FROM p_comando->>'hora_fin'
    OR calc->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
    OR calc->>'eur_por_km' !~ '^0\.[0-9]{4}$'
    OR jsonb_typeof(calc->'tramos_ruta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'tramos_ruta')<>jsonb_array_length(p_comando->'codigos_ruta')-1
    OR jsonb_typeof(calc->'opciones_dieta') IS DISTINCT FROM 'array'
    OR jsonb_array_length(calc->'opciones_dieta')<>3 THEN RETURN false; END IF;
 inicio:=(p_comando->>'fecha_inicio')::date; fin:=(p_comando->>'fecha_fin')::date;
 IF fin<inicio OR fin-inicio>30 THEN RETURN false; END IF;
 v_tarifa:=calc->>'version_tarifa';
 regla:=vec_dietas.consultar_regla_devengo_dietas_v1(v_tarifa,inicio,'ES','nacional_ordinaria');
 cfg:=regla->'configuracion';
 IF calc->>'regla_ref' IS DISTINCT FROM regla->>'regla_ref'
    OR calc->>'regla_huella_sha256' IS DISTINCT FROM regla->>'huella_sha256' THEN RETURN false; END IF;
 porcentajes:=ARRAY[(cfg->>'porcentaje_mismo_dia')::int,
  (cfg->>'porcentaje_salida_temprana')::int,(cfg->>'porcentaje_salida_media')::int,
  (cfg->>'porcentaje_regreso')::int,(cfg->>'porcentaje_intermedio')::int];
 SELECT k.eur_por_km INTO STRICT tarifa FROM vec_dietas.importe_km_provisional k
 JOIN vec_dietas.version_tarifa_provisional v ON v.version_ref=k.version_ref
 WHERE k.version_ref=v_tarifa AND k.vehiculo='automovil'
   AND v.vigente_desde<=inicio AND (v.vigente_hasta IS NULL OR fin<v.vigente_hasta);
 IF tarifa IS DISTINCT FROM (calc->>'eur_por_km')::numeric THEN RETURN false; END IF;
 FOR n IN 0..jsonb_array_length(calc->'tramos_ruta')-1 LOOP
  tr:=calc->'tramos_ruta'->n;
  IF tr->>'origen_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>n
     OR tr->>'destino_codigo' IS DISTINCT FROM p_comando->'codigos_ruta'->>(n+1)
     OR tr->>'kilometros' !~ '^(0|[1-9][0-9]{0,4})\.[0-9]{4}$'
     OR (tr->>'kilometros')::numeric<=0 THEN RETURN false; END IF;
  km:=km+(tr->>'kilometros')::numeric;
 END LOOP;
 IF km>10000 OR km IS DISTINCT FROM (calc->>'kilometros')::numeric
    OR (calc->>'importe_kilometraje_centimos')::bigint
       IS DISTINCT FROM round(km*tarifa*100)::bigint THEN RETURN false; END IF;
 FOR g IN 1..3 LOOP
  op:=calc->'opciones_dieta'->(g-1);
  SELECT d.manutencion_eur,d.alojamiento_eur INTO STRICT man,alo
  FROM vec_dietas.importe_dieta_provisional d
  WHERE d.version_ref=v_tarifa AND d.pais_iso2='ES' AND d.grupo=g;
  IF (op->>'grupo')::int IS DISTINCT FROM g
     OR op->'calculo'->>'version_tarifa_ref' IS DISTINCT FROM v_tarifa
     OR op->'calculo'->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
     OR jsonb_typeof(op->'calculo'->'tramos') IS DISTINCT FROM 'array'
     OR jsonb_array_length(op->'calculo'->'tramos')>62 THEN RETURN false; END IF;
  suma_man:=0; suma_alo:=0;
  FOR tr IN SELECT value FROM jsonb_array_elements(op->'calculo'->'tramos') LOOP
   IF tr->>'version_tarifa_ref' IS DISTINCT FROM v_tarifa
      OR tr->>'rotulo' IS DISTINCT FROM calc->>'rotulo'
      OR (tr->>'fecha')::date NOT BETWEEN inicio AND fin THEN RETURN false; END IF;
   pct:=(tr->>'porcentaje')::int; importe:=(tr->>'importe_centimos')::bigint;
   IF tr->>'tipo'='manutencion' AND pct=ANY(porcentajes)
      AND importe=ceil(man*100*pct/100)::bigint THEN suma_man:=suma_man+importe;
   ELSIF tr->>'tipo'='alojamiento_tope_pendiente_justificante'
      AND pct=(cfg->>'porcentaje_alojamiento_tope')::int
      AND importe=ceil(alo*100*pct/100)::bigint THEN suma_alo:=suma_alo+importe;
   ELSE RETURN false; END IF;
  END LOOP;
  IF suma_man IS DISTINCT FROM (op->'calculo'->>'manutencion_centimos')::bigint
     OR suma_alo IS DISTINCT FROM (op->'calculo'->>'alojamiento_tope_centimos')::bigint
     OR suma_man+suma_alo IS DISTINCT FROM (op->'calculo'->>'total_maximo_orientativo_centimos')::bigint
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN others THEN RETURN false;
END $$;
ALTER FUNCTION vec_dietas.validar_calculo_catalogado_v2(jsonb) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.validar_calculo_catalogado_v2(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; c jsonb; calc jsonb; salida jsonb; anterior jsonb;
        ref text; recibo_ref text; repeticion boolean;
        regla vec_dietas.regla_devengo_provisional%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN
  RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; c:=m->'comando'; calc:=c->'calculo';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END;
 IF m->>'esquema'<>'vec.dietas.borrador-operacion.v1'
    OR m->>'operacion'<>'crear' OR m->>'recurso_ref'<>'dietas:borradores:propios'
    OR jsonb_typeof(calc) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'alta Dietas inválida' USING ERRCODE='22023'; END IF;
 -- El núcleo v1 consume AD3-49, liga material completo, revalida Personal y
 -- crea el recibo base. Cualquier rechazo posterior revierte TODA la tx.
 salida:=vec_dietas.crear_o_recuperar_borrador_propio_v1(
  p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
  p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ref:=salida->'comision'->>'referencia';
 recibo_ref:=salida->'recibo'->>'referencia';
 repeticion:=(salida->'recibo'->>'repeticion')::boolean;
 IF ref !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR recibo_ref !~ '^rcd_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR repeticion IS NULL THEN RAISE EXCEPTION 'alta Dietas no reconciliable' USING ERRCODE='PD003'; END IF;
 IF NOT repeticion THEN
  IF vec_dietas.validar_calculo_catalogado_v2(c) IS NOT TRUE THEN
   RAISE EXCEPTION 'cálculo catalogado Dietas inválido' USING ERRCODE='PD001'; END IF;
  SELECT * INTO STRICT regla FROM vec_dietas.regla_devengo_provisional
   WHERE regla_ref=calc->>'regla_ref' AND version_tarifa_ref=calc->>'version_tarifa'
     AND huella_sha256=calc->>'regla_huella_sha256';
  INSERT INTO vec_dietas.calculo_comision(comision_ref,calculo,version_tarifa,registrado_en)
   VALUES(ref,calc,calc->>'version_tarifa',clock_timestamp());
  INSERT INTO vec_dietas.recibo_regla_creacion_comision
   (comision_ref,recibo_ref,regla_ref,regla_huella_sha256,registrada_en)
   VALUES(ref,recibo_ref,regla.regla_ref,regla.huella_sha256,clock_timestamp());
 END IF;
 SELECT x.calculo INTO STRICT anterior FROM vec_dietas.calculo_comision x WHERE x.comision_ref=ref;
 IF anterior ? 'regla_ref' AND NOT EXISTS (
   SELECT 1 FROM vec_dietas.recibo_regla_creacion_comision x
   WHERE x.comision_ref=ref AND x.recibo_ref=recibo_ref
     AND x.regla_ref=anterior->>'regla_ref'
     AND x.regla_huella_sha256=anterior->>'regla_huella_sha256') THEN
  RAISE EXCEPTION 'recibo de regla Dietas incoherente' USING ERRCODE='PD003'; END IF;
 RETURN jsonb_set(salida,'{comision,calculo}',anterior,true);
END $$;
ALTER FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_dietas_ejecutor;

CREATE FUNCTION vec_dietas.proyectar_comision_v2(p_ref text,p_repeticion boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $$
DECLARE b vec_dietas.borrador_comision%ROWTYPE; x vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE; r1 vec_dietas.recibo_borrador_comision%ROWTYPE;
        regla_creacion vec_dietas.recibo_regla_creacion_comision%ROWTYPE;
        numero vec_dietas.numero_documento_comision%ROWTYPE;
        calc jsonb; comision jsonb; recibo jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER') THEN
  RAISE EXCEPTION 'proyección Dietas inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO b FROM vec_dietas.borrador_comision
 WHERE referencia=p_ref AND persona_ref=current_setting('vec.dietas.persona_ref',true);
 IF NOT FOUND THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=p_ref;
 SELECT * INTO x FROM vec_dietas.comision_revision WHERE comision_ref=p_ref ORDER BY version DESC LIMIT 1;
 IF FOUND THEN
  SELECT * INTO STRICT r FROM vec_dietas.recibo_operacion_comision
   WHERE comision_ref=p_ref AND version=x.version;
  comision:=jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
   'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado',x.estado,'version',x.version,
   'fecha_inicio',x.fecha_inicio::text,'fecha_fin',x.fecha_fin::text,'motivo',x.motivo,
   'codigos_ruta',x.codigos_ruta,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'centro_ref',x.centro_ref,'calculo',x.calculo,
   'documento',x.documento);
  IF x.vehiculo_propio IS NOT NULL THEN
   comision:=comision||jsonb_build_object('vehiculo_propio',x.vehiculo_propio,'rutas',x.rutas);
  END IF;
  recibo:=jsonb_build_object('referencia',r.referencia,'version',r.version,
   'regla_ref',r.regla_ref,'regla_huella_sha256',r.regla_huella_sha256,
   'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',p_repeticion);
 ELSE
  SELECT calculo INTO STRICT calc FROM vec_dietas.calculo_comision WHERE comision_ref=p_ref;
  SELECT * INTO STRICT r1 FROM vec_dietas.recibo_borrador_comision WHERE comision_ref=p_ref;
  comision:=jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
   'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado','borrador','version',1,
   'fecha_inicio',b.fecha_inicio::text,'fecha_fin',b.fecha_fin::text,'motivo',b.motivo,
   'codigos_ruta',b.codigos_ruta,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'centro_ref',NULL,'calculo',calc,
   'documento',NULL);
  recibo:=jsonb_build_object('referencia',r1.referencia,'version',1,
   'registrado_en',to_char(r1.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',p_repeticion);
  SELECT * INTO regla_creacion FROM vec_dietas.recibo_regla_creacion_comision
   WHERE comision_ref=p_ref AND recibo_ref=r1.referencia;
  IF FOUND THEN recibo:=recibo||jsonb_build_object('regla_ref',regla_creacion.regla_ref,
   'regla_huella_sha256',regla_creacion.regla_huella_sha256); END IF;
 END IF;
 RETURN jsonb_build_object('resultado','concedido','comision',comision,'recibo',recibo);
END $$;
ALTER FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) FROM PUBLIC;

CREATE FUNCTION vec_dietas.autorizar_documento_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; d jsonb; cap jsonb; ctx jsonb; v record;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER') THEN
  RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN m:=p_material::jsonb; i:=m->'identidad';
       d:=convert_from(p_decision,'UTF8')::jsonb;
       cap:=convert_from(p_capacidad,'UTF8')::jsonb;
       ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 IF vec_dietas.cotejar_recurso_documento_v2(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR (m->>'operacion' IN ('editar','borrar','enviar')
        AND m->>'huella_semantica' IS DISTINCT FROM vec_dietas.huella_semantica_mutacion_v2(p_material))
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric THEN
  RAISE EXCEPTION 'preimagen Dietas no acreditada' USING ERRCODE='PD003'; END IF;
 IF m->>'operacion' IN ('detalle','lista') THEN
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_consulta_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
    OR vec_dietas.cotejar_contexto_dietas_borrador_v1(i,p_contexto) IS NOT TRUE THEN
  RAISE EXCEPTION 'sello Dietas no ligado' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',i->>'persona_ref',true);
 PERFORM vec_personal.revalidar_relacion_dietas_v1(
  i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',
  i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),
  (i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',
  (i->>'fuente_version')::bigint,(i->>'fecha_referencia')::date);
 RETURN jsonb_build_object('decision_ref',v.decision_ref,
  'consumo_huella_sha256',v.consumo_huella_sha256,'auditoria_ad3_ref',v.auditoria_ref,
  'actor_ref',d->>'principal_id','correlacion_ref',d->>'correlacion_ref');
END $$;
ALTER FUNCTION vec_dietas.autorizar_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.autorizar_documento_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.consultar_comisiones_propias_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; q jsonb; a jsonb; b vec_dietas.borrador_comision%ROWTYPE;
        salida jsonb; items jsonb:='[]'::jsonb; siguiente text:=''; vistos int:=0;
        ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; q:=m->'consulta';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'consulta Dietas inválida' USING ERRCODE='22023'; END;
 IF m->>'operacion' NOT IN ('detalle','lista')
    OR (m->>'operacion'='detalle' AND m->>'referencia' !~ '^dco_[A-Za-z0-9_-]{22,128}$')
    OR (m->>'operacion'='lista' AND
       (jsonb_typeof(q)<>'object' OR (q->>'limite')::int NOT BETWEEN 1 AND 50
        OR (coalesce(q->>'cursor','')<>'' AND q->>'cursor' !~ '^dco_[A-Za-z0-9_-]{22,128}$')))
 THEN RAISE EXCEPTION 'consulta Dietas inválida' USING ERRCODE='22023'; END IF;
 a:=vec_dietas.autorizar_documento_v2(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF m->>'operacion'='detalle' THEN
  salida:=vec_dietas.proyectar_comision_v2(m->>'referencia',true);
  INSERT INTO vec_dietas.auditoria_borrador_comision VALUES(
   'adi_'||md5(a->>'auditoria_ad3_ref'||m->>'referencia'||ahora::text),
   CASE WHEN salida->>'resultado'='concedido' THEN m->>'referencia' ELSE NULL END,
   m->>'referencia','consultar',a->>'actor_ref',i->>'persona_ref',
   CASE WHEN salida->>'resultado'='concedido' THEN 'concedido' ELSE 'no_encontrado' END,
   a->>'correlacion_ref',ahora);
  RETURN salida;
 END IF;
 FOR b IN SELECT b.* FROM vec_dietas.borrador_comision b
  LEFT JOIN LATERAL (SELECT r.estado FROM vec_dietas.comision_revision r
   WHERE r.comision_ref=b.referencia ORDER BY r.version DESC LIMIT 1) actual ON true
  WHERE b.persona_ref=i->>'persona_ref' AND b.empleado_ref=i->>'empleado_ref'
    AND b.relacion_ref=i->>'relacion_ref' AND b.unidad_ref=i->>'unidad_ref'
    AND coalesce(actual.estado,'borrador')<>'eliminado'
    AND (coalesce(q->>'cursor','')='' OR b.referencia>q->>'cursor')
  ORDER BY b.referencia LIMIT (q->>'limite')::int+1 LOOP
  vistos:=vistos+1; IF vistos>(q->>'limite')::int THEN EXIT; END IF;
  siguiente:=b.referencia;
  items:=items||jsonb_build_array(vec_dietas.proyectar_comision_v2(b.referencia,true));
 END LOOP;
 IF vistos<=(q->>'limite')::int THEN siguiente:=''; END IF;
 INSERT INTO vec_dietas.auditoria_borrador_comision VALUES(
  'adi_'||md5(a->>'auditoria_ad3_ref'||'dietas:borradores:propios'||ahora::text),
  NULL,'dietas:borradores:propios','consultar',a->>'actor_ref',i->>'persona_ref',
  'concedido',a->>'correlacion_ref',ahora);
 RETURN jsonb_build_object('items',items,'siguiente_cursor',siguiente);
END $$;
ALTER FUNCTION vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.proyectar_revision_exacta_v2(p_ref text,p_version bigint)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' AS $$
DECLARE b vec_dietas.borrador_comision%ROWTYPE; x vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE;
        numero vec_dietas.numero_documento_comision%ROWTYPE;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER') THEN
  RAISE EXCEPTION 'proyección Dietas inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT b FROM vec_dietas.borrador_comision
  WHERE referencia=p_ref AND persona_ref=current_setting('vec.dietas.persona_ref',true);
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=p_ref;
 SELECT * INTO STRICT x FROM vec_dietas.comision_revision
  WHERE comision_ref=p_ref AND version=p_version;
 SELECT * INTO STRICT r FROM vec_dietas.recibo_operacion_comision
  WHERE comision_ref=p_ref AND version=p_version;
 RETURN jsonb_build_object('resultado','concedido',
  'comision',jsonb_build_object('referencia',p_ref,'numero_documento',numero.numero_documento,
    'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'estado',x.estado,'version',x.version,
    'fecha_inicio',x.fecha_inicio::text,'fecha_fin',x.fecha_fin::text,'motivo',x.motivo,
    'codigos_ruta',x.codigos_ruta,'relacion_ref',b.relacion_ref,
    'unidad_ref',b.unidad_ref,'centro_ref',x.centro_ref,
    'calculo',x.calculo,'documento',x.documento)||
      CASE WHEN x.vehiculo_propio IS NULL THEN '{}'::jsonb
           ELSE jsonb_build_object('vehiculo_propio',x.vehiculo_propio,'rutas',x.rutas) END,
  'recibo',jsonb_build_object('referencia',r.referencia,'version',r.version,
    'regla_ref',r.regla_ref,'regla_huella_sha256',r.regla_huella_sha256,
    'registrado_en',to_char(r.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'repeticion',true));
END $$;
ALTER FUNCTION vec_dietas.proyectar_revision_exacta_v2(text,bigint) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.proyectar_revision_exacta_v2(text,bigint) FROM PUBLIC;

CREATE FUNCTION vec_dietas.intencion_documento_v2(p_comando jsonb) RETURNS jsonb
LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT CASE WHEN $1 ? 'asignacion' THEN
  jsonb_set($1 - ARRAY['calculo','documento'],'{asignacion}',
   ($1->'asignacion') - ARRAY['recibo_ref','decision_ref','efecto_ref',
    'consumo_huella_sha256','auditoria_ref','registrada_en','estado_local'])
 ELSE $1 - ARRAY['calculo','documento'] END
$$;
ALTER FUNCTION vec_dietas.intencion_documento_v2(jsonb) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.intencion_documento_v2(jsonb) FROM PUBLIC;

-- Antes de llamar de nuevo a OSRM, un PUT puede consultar su propia clave.
-- La intención se coteja sin cálculo/documento y con una autorización nueva.
CREATE FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; c jsonb; a jsonb; b vec_dietas.borrador_comision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE; salida jsonb;
BEGIN
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'preconsulta Dietas inválida' USING ERRCODE='22023'; END;
 IF m->>'operacion'<>'editar' OR c ? 'calculo' OR c ? 'documento' OR c ? 'asignacion'
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR (c->>'version_esperada')::bigint<1
    OR c->>'relacion_ref' IS DISTINCT FROM i->>'relacion_ref'
    OR m->>'recurso_ref' IS DISTINCT FROM m->>'referencia'
 THEN RAISE EXCEPTION 'preconsulta Dietas inválida' USING ERRCODE='22023'; END IF;
 a:=vec_dietas.autorizar_documento_v2(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO b FROM vec_dietas.borrador_comision
  WHERE referencia=m->>'referencia' AND persona_ref=i->>'persona_ref'
    AND empleado_ref=i->>'empleado_ref' AND relacion_ref=i->>'relacion_ref'
    AND unidad_ref=i->>'unidad_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'comisión Dietas ausente' USING ERRCODE='PD004'; END IF;
 SELECT * INTO r FROM vec_dietas.recibo_operacion_comision
  WHERE persona_ref=i->>'persona_ref' AND operacion='editar'
    AND clave_idempotencia=c->>'clave_idempotencia';
 IF NOT FOUND THEN RETURN jsonb_build_object('encontrado',false); END IF;
 IF r.comision_ref IS DISTINCT FROM b.referencia
    OR vec_dietas.intencion_documento_v2(r.comando)-'asignacion'
       IS DISTINCT FROM vec_dietas.intencion_documento_v2(c) THEN
  RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
 IF vec_personal.revalidar_asignacion_dietas_v1(
    b.relacion_ref,b.persona_ref,b.unidad_ref,
    r.comando->'asignacion'->>'asignacion_ref',
    (r.comando->'asignacion'->>'version')::bigint,
    (r.comando->'asignacion'->>'grupo_dieta')::smallint,
    r.comando->'asignacion'->>'centro_ref',
    r.comando->'asignacion'->>'administrativo_persona_ref',
    r.comando->'asignacion'->>'responsable_persona_ref',
    (i->>'fecha_referencia')::date) IS NOT TRUE THEN
  RAISE EXCEPTION 'asignación Dietas no vigente' USING ERRCODE='PD003'; END IF;
 salida:=vec_dietas.proyectar_revision_exacta_v2(r.comision_ref,r.version);
 RETURN salida||jsonb_build_object('encontrado',true);
END $$;
ALTER FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.mutar_comision_propia_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $$
DECLARE m jsonb; i jsonb; c jsonb; asignacion jsonb; a jsonb;
        b vec_dietas.borrador_comision%ROWTYPE; anterior vec_dietas.comision_revision%ROWTYPE;
        r vec_dietas.recibo_operacion_comision%ROWTYPE;
        estado_anterior text; estado_nuevo text; operacion text; version_actual bigint;
        nuevo bigint; ref_recibo text; fecha_inicio date; fecha_fin date;
        hora_inicio text; hora_fin text; motivo text; codigos jsonb; calculo jsonb; documento jsonb;
        vehiculo boolean; rutas jsonb;
        regla text; regla_huella text;
        ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
        huella text; salida jsonb;
BEGIN
 BEGIN m:=p_material::jsonb; i:=m->'identidad'; c:=m->'comando';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 operacion:=m->>'operacion';
 IF operacion NOT IN ('editar','borrar','enviar')
    OR jsonb_typeof(c)<>'object'
    OR m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR m->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR (c->>'version_esperada')::bigint<1
    OR c->>'relacion_ref' IS DISTINCT FROM i->>'relacion_ref'
    OR m->>'huella_semantica' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END IF;
 IF operacion IN ('borrar','enviar') AND
    ((operacion='borrar' AND c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref'])<>'{}'::jsonb)
      OR (operacion='enviar' AND c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref','asignacion'])<>'{}'::jsonb))
 THEN RAISE EXCEPTION 'comando Dietas inválido' USING ERRCODE='22023'; END IF;
 a:=vec_dietas.autorizar_documento_v2(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO b FROM vec_dietas.borrador_comision
 WHERE referencia=m->>'referencia' AND persona_ref=i->>'persona_ref'
   AND empleado_ref=i->>'empleado_ref' AND relacion_ref=i->>'relacion_ref'
   AND unidad_ref=i->>'unidad_ref' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'comisión Dietas ausente' USING ERRCODE='PD004'; END IF;
 SELECT * INTO r FROM vec_dietas.recibo_operacion_comision rco
 WHERE rco.persona_ref=i->>'persona_ref' AND rco.operacion=m->>'operacion'
   AND rco.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF r.comision_ref IS DISTINCT FROM b.referencia
     OR vec_dietas.intencion_documento_v2(r.comando)
        IS DISTINCT FROM vec_dietas.intencion_documento_v2(c) THEN
   RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  IF operacion IN ('editar','enviar') AND vec_personal.revalidar_asignacion_dietas_v1(
      b.relacion_ref,b.persona_ref,b.unidad_ref,
      r.comando->'asignacion'->>'asignacion_ref',
      (r.comando->'asignacion'->>'version')::bigint,
      (r.comando->'asignacion'->>'grupo_dieta')::smallint,
      r.comando->'asignacion'->>'centro_ref',
      r.comando->'asignacion'->>'administrativo_persona_ref',
      r.comando->'asignacion'->>'responsable_persona_ref',
      (i->>'fecha_referencia')::date) IS NOT TRUE THEN
   RAISE EXCEPTION 'asignación Dietas no vigente' USING ERRCODE='PD003'; END IF;
  RETURN vec_dietas.proyectar_revision_exacta_v2(r.comision_ref,r.version);
 END IF;
 SELECT * INTO anterior FROM vec_dietas.comision_revision
 WHERE comision_ref=b.referencia ORDER BY version DESC LIMIT 1;
 IF FOUND THEN
  version_actual:=anterior.version; estado_anterior:=anterior.estado;
  fecha_inicio:=anterior.fecha_inicio; fecha_fin:=anterior.fecha_fin;
  hora_inicio:=anterior.hora_inicio; hora_fin:=anterior.hora_fin;
  motivo:=anterior.motivo; codigos:=anterior.codigos_ruta;
  calculo:=anterior.calculo; documento:=anterior.documento;
  vehiculo:=anterior.vehiculo_propio; rutas:=anterior.rutas;
  regla:=anterior.regla_ref;
 ELSE
  version_actual:=1; estado_anterior:='borrador';
  fecha_inicio:=b.fecha_inicio; fecha_fin:=b.fecha_fin;
  SELECT x.calculo INTO STRICT calculo FROM vec_dietas.calculo_comision x
   WHERE x.comision_ref=b.referencia;
  hora_inicio:=calculo->>'hora_inicio'; hora_fin:=calculo->>'hora_fin';
  motivo:=b.motivo; codigos:=b.codigos_ruta; documento:=NULL;
  vehiculo:=NULL; rutas:=NULL;
  regla:='provisional:regla:nacional-ordinaria:20260923';
 END IF;
 IF (c->>'version_esperada')::bigint IS DISTINCT FROM version_actual
    OR (operacion='borrar' AND estado_anterior<>'borrador')
    OR (operacion IN ('editar','enviar') AND estado_anterior NOT IN ('borrador','devuelta')) THEN
  RAISE EXCEPTION 'versión o estado Dietas incompatible' USING ERRCODE='PD005'; END IF;
 IF operacion IN ('editar','enviar') THEN
  asignacion:=c->'asignacion';
  IF jsonb_typeof(asignacion) IS DISTINCT FROM 'object'
     OR asignacion->>'relacion_ref' IS DISTINCT FROM b.relacion_ref
     OR asignacion->>'persona_ref' IS DISTINCT FROM b.persona_ref
     OR asignacion->>'unidad_ref' IS DISTINCT FROM b.unidad_ref
     OR asignacion->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$'
     OR (asignacion->>'version')::bigint<1
     OR (asignacion->>'grupo_dieta')::smallint NOT BETWEEN 1 AND 3
     OR asignacion->>'administrativo_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR asignacion->>'responsable_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR asignacion->>'centro_ref' IS NULL
     OR vec_personal.revalidar_asignacion_dietas_v1(
       b.relacion_ref,b.persona_ref,b.unidad_ref,asignacion->>'asignacion_ref',
       (asignacion->>'version')::bigint,(asignacion->>'grupo_dieta')::smallint,
       asignacion->>'centro_ref',asignacion->>'administrativo_persona_ref',
       asignacion->>'responsable_persona_ref',(i->>'fecha_referencia')::date) IS NOT TRUE THEN
   RAISE EXCEPTION 'asignación Dietas no vigente' USING ERRCODE='PD003'; END IF;
 END IF;
 IF operacion='editar' THEN
  IF c-(ARRAY['clave_idempotencia','version_esperada','relacion_ref','fecha_inicio','fecha_fin','hora_inicio','hora_fin','motivo','codigos_ruta','vehiculo_propio','rutas','tramos_aceptados','version_tarifa_aceptada','asignacion','calculo','documento','otros'])<>'{}'::jsonb
     OR c->>'fecha_inicio' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR c->>'fecha_fin' !~ '^20[0-9]{2}-[0-9]{2}-[0-9]{2}$'
     OR c->>'hora_inicio' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
     OR c->>'hora_fin' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
     OR length(coalesce(c->>'motivo','')) NOT BETWEEN 3 AND 600
     OR jsonb_typeof(c->'codigos_ruta')<>'array'
     OR jsonb_typeof(c->'vehiculo_propio') IS DISTINCT FROM 'boolean'
     OR jsonb_typeof(c->'rutas') IS DISTINCT FROM 'array'
     OR vec_dietas.validar_documento_v2(c) IS NOT TRUE THEN
   RAISE EXCEPTION 'edición Dietas inválida' USING ERRCODE='PD001'; END IF;
  fecha_inicio:=(c->>'fecha_inicio')::date; fecha_fin:=(c->>'fecha_fin')::date;
  hora_inicio:=c->>'hora_inicio'; hora_fin:=c->>'hora_fin'; motivo:=c->>'motivo';
  codigos:=c->'codigos_ruta'; calculo:=c->'calculo'; documento:=c->'documento';
  vehiculo:=(c->>'vehiculo_propio')::boolean; rutas:=c->'rutas';
  regla:=calculo->>'regla_ref';
  IF fecha_fin<fecha_inicio OR fecha_fin-fecha_inicio>30
     OR NOT EXISTS (SELECT 1 FROM vec_dietas.regla_devengo_provisional p
         WHERE p.regla_ref=regla AND p.version_tarifa_ref=calculo->>'version_tarifa'
           AND p.huella_sha256=calculo->>'regla_huella_sha256'
           AND p.huella_sha256=encode(sha256(convert_to(p.configuracion::text,'UTF8')),'hex')) THEN
   RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD001'; END IF;
  estado_nuevo:='borrador';
 ELSIF operacion='borrar' THEN
  estado_nuevo:='eliminado';
 ELSE
  IF documento IS NULL OR vehiculo IS NULL OR rutas IS NULL THEN
   RAISE EXCEPTION 'documento Dietas incompleto' USING ERRCODE='PD005'; END IF;
  IF anterior.asignacion_ref IS DISTINCT FROM asignacion->>'asignacion_ref'
     OR anterior.asignacion_version IS DISTINCT FROM (asignacion->>'version')::bigint
     OR anterior.grupo_dieta IS DISTINCT FROM (asignacion->>'grupo_dieta')::smallint THEN
   RAISE EXCEPTION 'documento Dietas requiere actualización de asignación' USING ERRCODE='PD005'; END IF;
  estado_nuevo:='enviado_pendiente_revision';
 END IF;
 IF operacion IN ('editar','enviar') THEN
  -- El sello de asignación usa fecha de canal actual; la relación Personal
  -- también debe cubrir los dos extremos civiles de la comisión declarada.
  PERFORM vec_personal.revalidar_relacion_dietas_v1(
   i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',
   i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),
   (i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',
   (i->>'fuente_version')::bigint,fecha_inicio);
  PERFORM vec_personal.revalidar_relacion_dietas_v1(
   i->>'relacion_ref',i->>'persona_ref',i->>'empleado_ref',i->>'unidad_ref',
   i->>'vigente_desde',coalesce(i->>'vigente_hasta',''),
   (i->>'relacion_version')::bigint,i->>'procedencia_acto_ref',i->>'fuente_ref',
   (i->>'fuente_version')::bigint,fecha_fin);
 END IF;
 SELECT p.huella_sha256 INTO STRICT regla_huella
 FROM vec_dietas.regla_devengo_provisional p WHERE p.regla_ref=regla;
 IF calculo ? 'regla_huella_sha256'
    AND calculo->>'regla_huella_sha256' IS DISTINCT FROM regla_huella THEN
  RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD001'; END IF;
 nuevo:=version_actual+1;
 INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,
   hora_inicio,hora_fin,motivo,codigos_ruta,vehiculo_propio,rutas,calculo,documento,regla_ref,
   asignacion_ref,asignacion_version,grupo_dieta,centro_ref,
   administrativo_persona_ref,responsable_persona_ref,registrada_en)
 VALUES(b.referencia,nuevo,estado_nuevo,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,
   codigos,vehiculo,rutas,calculo,documento,regla,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'asignacion_ref' ELSE anterior.asignacion_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN (asignacion->>'version')::bigint ELSE anterior.asignacion_version END,
   CASE WHEN operacion IN ('editar','enviar') THEN (asignacion->>'grupo_dieta')::smallint ELSE anterior.grupo_dieta END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'centro_ref' ELSE anterior.centro_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'administrativo_persona_ref' ELSE anterior.administrativo_persona_ref END,
   CASE WHEN operacion IN ('editar','enviar') THEN asignacion->>'responsable_persona_ref' ELSE anterior.responsable_persona_ref END,
   ahora);
 huella:=encode(sha256(convert_to(jsonb_build_object('persona_ref',b.persona_ref,
   'empleado_ref',b.empleado_ref,'relacion_ref',b.relacion_ref,
   'unidad_ref',b.unidad_ref,'relacion_version',b.relacion_version,
   'referencia',b.referencia,'operacion',operacion,
   'comando',vec_dietas.intencion_documento_v2(c))::text,'UTF8')),'hex');
 ref_recibo:='rcd_'||substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,8)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,4)||'-'||
   substr(md5(b.referencia||nuevo::text||ahora::text||random()::text),1,12);
 INSERT INTO vec_dietas.recibo_operacion_comision VALUES(ref_recibo,b.referencia,nuevo,
  operacion,c->>'clave_idempotencia',c,huella,a->>'decision_ref',
  a->>'consumo_huella_sha256',a->>'auditoria_ad3_ref',a->>'actor_ref',b.persona_ref,
  regla,regla_huella,ahora);
 INSERT INTO vec_dietas.historia_operacion_comision VALUES(
  'hdi_'||md5(ref_recibo),b.referencia,nuevo,ref_recibo,estado_anterior,estado_nuevo,
  operacion,NULL,a->>'actor_ref',ahora);
 IF operacion='enviar' THEN
  INSERT INTO vec_dietas.outbox_comision(evento_ref,comision_ref,version,tipo,persona_ref,
   destinatario_persona_ref,registrada_en)
  VALUES('odi_'||md5(ref_recibo),b.referencia,nuevo,'pendiente_revision',b.persona_ref,
   asignacion->>'administrativo_persona_ref',ahora);
 END IF;
 salida:=vec_dietas.proyectar_revision_exacta_v2(b.referencia,nuevo);
 RETURN jsonb_set(salida,'{recibo,repeticion}','false'::jsonb);
END $$;
ALTER FUNCTION vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- La auditoría de rechazos de frontera se mantiene en su rol técnico propio.
-- Referencias de URL se normalizan antes de llegar aquí: no se guardan IDs.
CREATE FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(
 p_correlacion_ref text,p_motivo text,p_superficie text,p_ruta text,p_accion text,p_actor_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET lock_timeout='1s' SET statement_timeout='2s' AS $$
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
      WHERE m.member=session_user::regrole
        AND m.roleid='vec_dietas_registrador_frontera'::regrole
        AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles rol
      WHERE rol.oid<>session_user::regrole
        AND rol.oid<>'vec_dietas_registrador_frontera'::regrole
        AND pg_catalog.pg_has_role(session_user,rol.oid,'MEMBER')) THEN
  RAISE EXCEPTION 'registrador frontera Dietas inválido' USING ERRCODE='42501'; END IF;
 IF p_correlacion_ref IS NULL
    OR (p_correlacion_ref<>'corr_no_disponible' AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_ruta IS NULL OR p_accion IS NULL
    OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado','dependencia_no_disponible')
    OR p_superficie IS DISTINCT FROM 'api.dietas.comisiones'
    OR p_ruta NOT IN ('/api/vec/dietas/comisiones',
      '/api/vec/dietas/comisiones/detalle',
      '/api/vec/dietas/comisiones/enviar',
      '/api/vec/dietas/comisiones/circuito')
    OR p_accion NOT IN ('listar','crear','consultar_detalle','editar','borrar','enviar',
      'consultar_bandeja','decidir','preleer','metodo_no_admitido')
    OR (p_accion<>'metodo_no_admitido' AND NOT (
      (p_ruta='/api/vec/dietas/comisiones' AND p_accion IN ('listar','crear'))
      OR (p_ruta='/api/vec/dietas/comisiones/detalle' AND p_accion IN ('consultar_detalle','editar','borrar'))
      OR (p_ruta='/api/vec/dietas/comisiones/enviar' AND p_accion='enviar')
      OR (p_ruta='/api/vec/dietas/comisiones/circuito' AND p_accion IN ('consultar_bandeja','decidir','preleer'))))
    OR (p_motivo='autenticacion_requerida' AND p_actor_ref IS NOT NULL)
    OR (p_actor_ref IS NOT NULL AND (length(p_actor_ref)>512
      OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$')) THEN
  RAISE EXCEPTION 'auditoría frontera Dietas inválida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_dietas.auditoria_frontera_comision
  (correlacion_ref,motivo,superficie,ruta,accion,actor_ref,registrada_en)
 VALUES(p_correlacion_ref,p_motivo,p_superficie,p_ruta,p_accion,p_actor_ref,
  date_trunc('microseconds',clock_timestamp()));
 RETURN true;
END $$;
ALTER FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text) FROM PUBLIC,vec_dietas_ejecutor,vec_dietas_migrador;
GRANT EXECUTE ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)
 TO vec_dietas_registrador_frontera;

GRANT EXECUTE ON FUNCTION vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
COMMENT ON TABLE vec_dietas.regla_devengo_provisional IS 'Regla orientativa versionada para ensayo nacional; pendiente de confirmación RRHH y no liquidable.';
COMMENT ON TABLE vec_dietas.numero_documento_comision IS 'Número interno VEC-D visible, global y estable; no equivale a registro oficial de RRHH.';
COMMENT ON TABLE vec_dietas.comision_revision IS 'Versiones inmutables del documento; v1 permanece en borrador_comision/calculo_comision.';
COMMENT ON TABLE vec_dietas.outbox_comision IS 'Intención durable de distribución interna; no acredita envío ni entrega.';
COMMIT;
