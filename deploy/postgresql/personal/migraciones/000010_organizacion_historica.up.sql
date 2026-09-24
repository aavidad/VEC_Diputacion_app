\set ON_ERROR_STOP on
-- B3. Historia estructural de Organización/Personal. No importa datos ni
-- sustituye el catálogo preparatorio vec_contratacion_temporal.organizacion_*.
-- La clave de entrada, versión y revisión de ese catálogo son referencias opacas;
-- ninguna tabla de otro módulo se consulta o modifica desde aquí.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000010:organizacion-historica:v1',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_personal_propietario'
    OR to_regclass('vec_personal.org_nodo_historia') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_organizacion_historica_v1(text,text,date,timestamptz,integer)') IS NOT NULL
    OR to_regprocedure('vec_personal.rechazar_mutacion_organizacion_v1()') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL THEN
   RAISE EXCEPTION 'Personal 000010: dependencias o preimagen incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $fn$
BEGIN
 RAISE EXCEPTION 'historia de organización inmutable' USING ERRCODE='55000';
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.rechazar_mutacion_organizacion_v1() FROM PUBLIC,vec_personal_ejecutor;

-- Cada revisión es solo de adición. El eje de conocimiento se obtiene eligiendo
-- la mayor revisión conocida a la fecha; una rectificación posterior no borra
-- la respuesta que el sistema podía dar antes de conocerla. Hasta exclusivo.
CREATE TABLE vec_personal.org_nodo_historia (
 nodo_ref uuid NOT NULL,
 revision integer NOT NULL CHECK (revision>0),
 organismo_ref text NOT NULL CHECK (organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK (unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 clase text NOT NULL CHECK (clase IN ('delegacion','centro','puesto_responsabilidad')),
 catalogo_ref text NOT NULL CHECK (catalogo_ref='estructura-organizativa-dipgra'),
 catalogo_version integer NOT NULL CHECK (catalogo_version>0),
 catalogo_revision integer NOT NULL CHECK (catalogo_revision>0),
 catalogo_entrada_clave text NOT NULL CHECK (catalogo_entrada_clave~'^[a-z][a-z0-9_:-]{2,159}$'),
 denominacion text NOT NULL CHECK (length(denominacion) BETWEEN 1 AND 300 AND denominacion !~ '[[:cntrl:]]'),
 centro_padre_ref uuid,
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL,
 vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK (fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK (acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK (huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(nodo_ref,revision),
 CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
 CHECK (clase<>'delegacion' OR centro_padre_ref IS NULL),
 CHECK (centro_padre_ref IS NULL OR centro_padre_ref<>nodo_ref)
);
CREATE UNIQUE INDEX org_nodo_catalogo_revision_uq ON vec_personal.org_nodo_historia
 (organismo_ref,catalogo_ref,catalogo_version,catalogo_revision,catalogo_entrada_clave);
CREATE UNIQUE INDEX org_nodo_historia_conocido_uq ON vec_personal.org_nodo_historia(nodo_ref,conocido_desde);
CREATE INDEX org_nodo_ambito_fecha_idx ON vec_personal.org_nodo_historia
 (organismo_ref,unidad_ref,conocido_desde DESC);

CREATE TABLE vec_personal.version_rpt_historia (
 version_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 codigo_version_fuente text NOT NULL CHECK(length(codigo_version_fuente) BETWEEN 1 AND 160),
 estado text NOT NULL CHECK(estado IN ('preparacion','reconciliada','aprobada','publicada','sustituida','retirada')),
 aprobada_en date, publicada_en date,
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 version_previa_ref uuid,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 documento_ref text NOT NULL CHECK(length(documento_ref) BETWEEN 1 AND 256),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(version_ref,revision),
 UNIQUE(version_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
 CHECK(version_previa_ref IS NULL OR version_previa_ref<>version_ref)
);
CREATE UNIQUE INDEX version_rpt_historia_conocido_uq ON vec_personal.version_rpt_historia(version_ref,conocido_desde);
CREATE INDEX version_rpt_organismo_fecha_idx ON vec_personal.version_rpt_historia (organismo_ref,conocido_desde DESC);

CREATE TABLE vec_personal.version_plantilla_historia (
 version_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 ejercicio integer NOT NULL CHECK(ejercicio BETWEEN 1900 AND 9999),
 codigo_version_fuente text NOT NULL CHECK(length(codigo_version_fuente) BETWEEN 1 AND 160),
 estado text NOT NULL CHECK(estado IN ('preparacion','reconciliada','aprobada','publicada','sustituida','retirada')),
 aprobada_en date, publicada_en date,
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 version_previa_ref uuid,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 documento_ref text NOT NULL CHECK(length(documento_ref) BETWEEN 1 AND 256),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(version_ref,revision),
 UNIQUE(version_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
 CHECK(version_previa_ref IS NULL OR version_previa_ref<>version_ref)
);
CREATE UNIQUE INDEX version_plantilla_historia_conocido_uq ON vec_personal.version_plantilla_historia(version_ref,conocido_desde);
CREATE INDEX version_plantilla_organismo_fecha_idx ON vec_personal.version_plantilla_historia (organismo_ref,conocido_desde DESC);

CREATE TABLE vec_personal.puesto_tipo_historia (
 tipo_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 rpt_version_ref uuid NOT NULL, rpt_revision integer NOT NULL,
 codigo_fila_fuente text NOT NULL CHECK(length(codigo_fila_fuente) BETWEEN 1 AND 128 AND codigo_fila_fuente=btrim(codigo_fila_fuente) AND codigo_fila_fuente !~ '[[:cntrl:]]'),
 codigo_datos_reserva_fuente text,
 denominacion text NOT NULL CHECK(length(denominacion) BETWEEN 1 AND 300 AND denominacion=btrim(denominacion) AND denominacion !~ '[[:cntrl:]]'),
 clasificacion_ref text NOT NULL CHECK(clasificacion_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 regimen_ref text CHECK(regimen_ref IS NULL OR length(regimen_ref) BETWEEN 1 AND 160),
 forma_provision_ref text CHECK(forma_provision_ref IS NULL OR length(forma_provision_ref) BETWEEN 1 AND 160),
 nivel_destino integer CHECK(nivel_destino BETWEEN 1 AND 30),
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(tipo_ref,revision),
 UNIQUE(tipo_ref,revision,organismo_ref),
 FOREIGN KEY(rpt_version_ref,rpt_revision,organismo_ref) REFERENCES vec_personal.version_rpt_historia(version_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE UNIQUE INDEX puesto_tipo_historia_conocido_uq ON vec_personal.puesto_tipo_historia(tipo_ref,conocido_desde);
CREATE INDEX puesto_tipo_ambito_fecha_idx ON vec_personal.puesto_tipo_historia (organismo_ref,unidad_ref,conocido_desde DESC);

-- Dotación es una cantidad declarada en la fila RPT; no crea puestos individuales.
CREATE TABLE vec_personal.dotacion_rpt_historia (
 dotacion_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 tipo_ref uuid NOT NULL, tipo_revision integer NOT NULL,
 cantidad integer NOT NULL CHECK(cantidad BETWEEN 1 AND 100000),
 reconciliacion text NOT NULL CHECK(reconciliacion IN ('pendiente','parcial','reconciliada')),
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(dotacion_ref,revision),
 FOREIGN KEY(tipo_ref,tipo_revision,organismo_ref) REFERENCES vec_personal.puesto_tipo_historia(tipo_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE UNIQUE INDEX dotacion_rpt_historia_conocido_uq ON vec_personal.dotacion_rpt_historia(dotacion_ref,conocido_desde);
CREATE INDEX dotacion_ambito_fecha_idx ON vec_personal.dotacion_rpt_historia (organismo_ref,unidad_ref,conocido_desde DESC);

CREATE TABLE vec_personal.plaza_plantilla_historia (
 plaza_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 plantilla_version_ref uuid NOT NULL, plantilla_revision integer NOT NULL,
 codigo_plaza_fuente text NOT NULL CHECK(length(codigo_plaza_fuente) BETWEEN 1 AND 128 AND codigo_plaza_fuente=btrim(codigo_plaza_fuente) AND codigo_plaza_fuente !~ '[[:cntrl:]]'),
 clasificacion_ref text NOT NULL CHECK(length(clasificacion_ref) BETWEEN 1 AND 160),
 estado_estructural text NOT NULL CHECK(estado_estructural IN ('vigente','amortizada')),
 dotacion_presupuestaria text NOT NULL CHECK(dotacion_presupuestaria IN ('acreditada','no_acreditada','desconocida')),
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(plaza_ref,revision),
 UNIQUE(plaza_ref,revision,organismo_ref),
 FOREIGN KEY(plantilla_version_ref,plantilla_revision,organismo_ref) REFERENCES vec_personal.version_plantilla_historia(version_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE UNIQUE INDEX plaza_plantilla_historia_conocido_uq ON vec_personal.plaza_plantilla_historia(plaza_ref,conocido_desde);
CREATE INDEX plaza_ambito_fecha_idx ON vec_personal.plaza_plantilla_historia (organismo_ref,unidad_ref,conocido_desde DESC);

CREATE TABLE vec_personal.puesto_rpt_historia (
 puesto_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 tipo_ref uuid NOT NULL, tipo_revision integer NOT NULL,
 codigo_puesto_fuente text NOT NULL CHECK(length(codigo_puesto_fuente) BETWEEN 1 AND 128 AND codigo_puesto_fuente=btrim(codigo_puesto_fuente) AND codigo_puesto_fuente !~ '[[:cntrl:]]'),
 estado_estructural text NOT NULL CHECK(estado_estructural IN ('vigente','suprimido')),
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(puesto_ref,revision),
 UNIQUE(puesto_ref,revision,organismo_ref),
 FOREIGN KEY(tipo_ref,tipo_revision,organismo_ref) REFERENCES vec_personal.puesto_tipo_historia(tipo_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE UNIQUE INDEX puesto_rpt_historia_conocido_uq ON vec_personal.puesto_rpt_historia(puesto_ref,conocido_desde);
CREATE INDEX puesto_ambito_fecha_idx ON vec_personal.puesto_rpt_historia (organismo_ref,unidad_ref,conocido_desde DESC);

CREATE TABLE vec_personal.vinculo_plaza_puesto_historia (
 vinculo_ref uuid NOT NULL, revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 plaza_ref uuid NOT NULL, plaza_revision integer NOT NULL,
 puesto_ref uuid NOT NULL, puesto_revision integer NOT NULL,
 estado text NOT NULL CHECK(estado IN ('confirmado','pendiente_reconciliacion','terminado')),
 retirado boolean NOT NULL DEFAULT false,
 vigente_desde date NOT NULL, vigente_hasta date,
 conocido_desde timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 acto_ref text NOT NULL CHECK(acto_ref~'^[a-z][a-z0-9_:-]{2,159}$'),
 huella_fuente_sha256 text NOT NULL CHECK(huella_fuente_sha256~'^[0-9a-f]{64}$'),
 PRIMARY KEY(vinculo_ref,revision),
 FOREIGN KEY(plaza_ref,plaza_revision,organismo_ref) REFERENCES vec_personal.plaza_plantilla_historia(plaza_ref,revision,organismo_ref),
 FOREIGN KEY(puesto_ref,puesto_revision,organismo_ref) REFERENCES vec_personal.puesto_rpt_historia(puesto_ref,revision,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE UNIQUE INDEX vinculo_plaza_puesto_historia_conocido_uq ON vec_personal.vinculo_plaza_puesto_historia(vinculo_ref,conocido_desde);
CREATE INDEX vinculo_ambito_fecha_idx ON vec_personal.vinculo_plaza_puesto_historia (organismo_ref,unidad_ref,conocido_desde DESC);

CREATE TABLE vec_personal.recibo_consulta_organizacion (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref~'^orgconsulta:[0-9a-f-]{36}$'),
 organismo_ref text NOT NULL CHECK(organismo_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 -- Mismo dominio que la función y las tablas de historia; '' = sin unidad.
 unidad_ref text NOT NULL CHECK(unidad_ref='' OR unidad_ref~'^[a-z][a-z0-9_:-]{2,127}$'),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 decision_ref text NOT NULL CHECK(length(decision_ref) BETWEEN 1 AND 256),
 auditoria_ref text NOT NULL CHECK(length(auditoria_ref) BETWEEN 1 AND 256),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 100),
 consultada_en timestamptz(6) NOT NULL
);
-- Una versión oficial no puede retroceder a preparación por una revisión
-- posterior del mismo ID. Las nuevas versiones efectivas usan otro version_ref
-- o una revisión con estado no regresivo y fecha de efectos explícita.
CREATE FUNCTION vec_personal.validar_revision_instrumento_organizacion_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $fn$
DECLARE anterior record; tabla text; encontrados integer;
BEGIN
 IF TG_TABLE_SCHEMA<>'vec_personal' OR TG_TABLE_NAME NOT IN ('version_rpt_historia','version_plantilla_historia') THEN
  RAISE EXCEPTION 'instrumento de organización desconocido' USING ERRCODE='55000';
 END IF;
 tabla:=format('vec_personal.%I',TG_TABLE_NAME);
 EXECUTE format('SELECT revision,organismo_ref,estado,conocido_desde FROM %s WHERE version_ref=$1 ORDER BY revision DESC LIMIT 1',tabla)
  INTO anterior USING NEW.version_ref;
 GET DIAGNOSTICS encontrados = ROW_COUNT;
 IF encontrados=0 THEN
  IF NEW.revision<>1 THEN RAISE EXCEPTION 'primera revisión de instrumento inválida' USING ERRCODE='55000'; END IF;
 ELSE
  IF NEW.revision<>anterior.revision+1 OR NEW.organismo_ref IS DISTINCT FROM anterior.organismo_ref
     OR NEW.conocido_desde<=anterior.conocido_desde
     OR NOT (CASE anterior.estado
       WHEN 'preparacion' THEN NEW.estado IN ('preparacion','reconciliada','aprobada','publicada','retirada')
       WHEN 'reconciliada' THEN NEW.estado IN ('reconciliada','aprobada','publicada','retirada')
       WHEN 'aprobada' THEN NEW.estado IN ('aprobada','publicada','retirada')
       WHEN 'publicada' THEN NEW.estado IN ('publicada','sustituida','retirada')
       WHEN 'sustituida' THEN NEW.estado IN ('sustituida','retirada')
       ELSE false END) THEN
   RAISE EXCEPTION 'regresión o discontinuidad de instrumento' USING ERRCODE='55000';
  END IF;
 END IF;
 IF NEW.retirado AND NEW.estado<>'retirada' THEN
  RAISE EXCEPTION 'retirada de instrumento sin estado retirado' USING ERRCODE='55000';
 END IF;
 RETURN NEW;
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.validar_revision_instrumento_organizacion_v1() FROM PUBLIC,vec_personal_ejecutor;
CREATE TRIGGER revision_no_regresiva BEFORE INSERT ON vec_personal.version_rpt_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_revision_instrumento_organizacion_v1();
CREATE TRIGGER revision_no_regresiva BEFORE INSERT ON vec_personal.version_plantilla_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_revision_instrumento_organizacion_v1();

DO $cerrar$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['org_nodo_historia','version_rpt_historia','version_plantilla_historia',
  'puesto_tipo_historia','dotacion_rpt_historia','plaza_plantilla_historia','puesto_rpt_historia',
  'vinculo_plaza_puesto_historia','recibo_consulta_organizacion'] LOOP
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_interno ON vec_personal.%I FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true)',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',t);
 END LOOP;
END $cerrar$;

-- Proyección de la misma traza bitemporal para todas las clases.
CREATE FUNCTION vec_personal.traza_organizacion_v1(
 p_id text,p_revision integer,p_fuente text,p_acto text,p_huella text,
 p_desde date,p_hasta date,p_conocido timestamptz,p_conocido_hasta timestamptz
) RETURNS jsonb LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $fn$
 SELECT jsonb_build_object('id',p_id,'version',p_revision,'fuente_ref',p_fuente,
  'acto_ref',p_acto,'huella_sha256',p_huella,'efectos_desde',p_desde,
  'efectos_hasta',p_hasta,'conocido_desde',to_char(p_conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'conocido_hasta',CASE WHEN p_conocido_hasta IS NULL THEN NULL ELSE to_char(p_conocido_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
$fn$;
REVOKE ALL ON FUNCTION vec_personal.traza_organizacion_v1(text,integer,text,text,text,date,date,timestamptz,timestamptz) FROM PUBLIC,vec_personal_ejecutor;

-- AD3-51 es la única autoridad de consumo; se invoca antes de consultar.
-- Una página contiene <=100 hechos totales y el cursor liga todos los filtros.
CREATE FUNCTION vec_personal.consultar_organizacion_historica_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $fn$
DECLARE
 m jsonb; d jsonb; c jsonb; v_consumo record;
 v_organismo text; v_unidad text; v_fecha date; v_conocido timestamptz(6);
 v_rpt text; v_plantilla text; v_rpt_uuid uuid; v_plantilla_uuid uuid;
 v_limite integer; v_cursor text; v_offset integer:=0; v_base_sha text;
 v_material_sha text; v_contexto_sha text; v_contexto_canon text;
 v_n integer; v_mas boolean; v_arrays jsonb; v_cobertura jsonb;
 v_recibo text; v_ahora timestamptz(6); v_cursor_siguiente text; v_valida_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL
    OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'consulta histórica de organización denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  v_organismo:=m->>'organismo_ref'; v_unidad:=m->>'unidad_clave';
  v_fecha:=(m->>'vigente_en')::date;
  v_conocido:=(m->>'conocido_en')::timestamptz;
  v_limite:=(m->>'limite')::integer; v_cursor:=m->>'cursor';
  v_rpt:=m->>'version_rpt_ref'; v_plantilla:=m->>'version_plantilla_ref';
  v_valida_hasta:=(d->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material histórico inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
      'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cursor','esquema',
      'limite','organismo_ref','perfil_ref','perfil_version','persona_version',
      'unidad_clave','version_plantilla_ref','version_rpt_ref','vigente_en']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.organizacion-historica.v1'
    OR v_organismo IS NULL OR v_unidad IS NULL OR v_fecha IS NULL OR v_conocido IS NULL
    OR v_limite IS NULL OR v_cursor IS NULL OR v_rpt IS NULL OR v_plantilla IS NULL
    OR v_organismo !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR (v_unidad<>'' AND v_unidad !~ '^[a-z][a-z0-9_:-]{2,127}$')
    OR v_fecha::text IS DISTINCT FROM m->>'vigente_en'
    OR to_char(v_conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
    OR v_conocido>transaction_timestamp()
    OR v_limite NOT BETWEEN 1 AND 100
    OR (v_rpt<>'' AND v_rpt !~ '^rpt:[0-9a-f-]{36}$')
    OR (v_plantilla<>'' AND v_plantilla !~ '^plantilla:[0-9a-f-]{36}$')
    OR v_cursor !~ '^(|p_[0-9]{1,8}_[0-9a-f]{64})$'
    OR m->>'actor_ref' IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'accion' IS DISTINCT FROM 'personal.organizacion_historica.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'organizacion_historica'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_organizacion_historica'
    OR d->>'recurso_ref' IS DISTINCT FROM v_organismo
    OR d->'campos_permitidos' IS DISTINCT FROM '["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.organizacion_historica.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR v_valida_hasta IS NULL OR clock_timestamp()>=v_valida_hasta THEN
  RAISE EXCEPTION 'consulta histórica incompatible' USING ERRCODE='42501';
 END IF;
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_contexto_canon:='{"ambitos":{"organismo_ref":'||to_jsonb(v_organismo)::text||
  ',"unidad_clave":'||to_jsonb(CASE WHEN v_unidad='' THEN 'sin_seleccion' ELSE v_unidad END)::text||'},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"cursor":'||to_jsonb(CASE WHEN v_cursor='' THEN 'sin_seleccion' ELSE v_cursor END)::text||
  ',"limite":'||to_jsonb(v_limite::text)::text||',"material_sha256":"'||v_material_sha||'"'||
  ',"version_plantilla_ref":'||to_jsonb(CASE WHEN v_plantilla='' THEN 'sin_seleccion' ELSE v_plantilla END)::text||
  ',"version_rpt_ref":'||to_jsonb(CASE WHEN v_rpt='' THEN 'sin_seleccion' ELSE v_rpt END)::text||
  ',"vigente_en":'||to_jsonb(v_fecha::text)::text||'}}';
 v_contexto_sha:=encode(sha256(convert_to(v_contexto_canon,'UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_sha
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_contexto_sha
    OR c->>'efecto_ref' IS DISTINCT FROM v_organismo THEN
  RAISE EXCEPTION 'selector y autorización divergentes' USING ERRCODE='42501';
 END IF;
 v_base_sha:=encode(sha256(convert_to((m-'cursor')::text,'UTF8')),'hex');
 IF v_cursor<>'' THEN
  v_offset:=split_part(v_cursor,'_',2)::integer;
  IF v_offset<1 OR v_offset>1000000 OR split_part(v_cursor,'_',3)<>v_base_sha THEN
   RAISE EXCEPTION 'cursor histórico divergente' USING ERRCODE='22023';
  END IF;
 END IF;
 IF v_rpt<>'' THEN v_rpt_uuid:=substr(v_rpt,5)::uuid; END IF;
 IF v_plantilla<>'' THEN v_plantilla_uuid:=substr(v_plantilla,11)::uuid; END IF;
 -- Las versiones seleccionadas han de estar vigentes y conocidas. Si hay más
 -- de una versión candidata sin selector explícito, no se inventa resolución.
 IF v_rpt_uuid IS NULL THEN
  SELECT count(*),min('rpt:'||version_ref::text) INTO v_n,v_rpt
   FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_rpt_historia
      WHERE organismo_ref=v_organismo AND conocido_desde<=v_conocido
        AND vigente_desde<=v_fecha AND (vigente_hasta IS NULL OR v_fecha<vigente_hasta)
      ORDER BY version_ref,conocido_desde DESC,revision DESC) r
   WHERE NOT retirado AND estado IN ('aprobada','publicada','sustituida');
 ELSE
  SELECT count(*),min('rpt:'||version_ref::text) INTO v_n,v_rpt
   FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_rpt_historia
      WHERE organismo_ref=v_organismo AND version_ref=v_rpt_uuid AND conocido_desde<=v_conocido
        AND vigente_desde<=v_fecha AND (vigente_hasta IS NULL OR v_fecha<vigente_hasta)
      ORDER BY version_ref,conocido_desde DESC,revision DESC) r
   WHERE NOT retirado AND estado IN ('aprobada','publicada','sustituida');
 END IF;
 IF v_n>1 THEN RAISE EXCEPTION 'versiones RPT superpuestas' USING ERRCODE='55000'; END IF;
 v_rpt:=coalesce(v_rpt,'');
 v_rpt_uuid:=CASE WHEN v_rpt='' THEN NULL ELSE substr(v_rpt,5)::uuid END;
 IF v_plantilla_uuid IS NULL THEN
  SELECT count(*),min('plantilla:'||version_ref::text) INTO v_n,v_plantilla
   FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_plantilla_historia
      WHERE organismo_ref=v_organismo AND conocido_desde<=v_conocido
        AND vigente_desde<=v_fecha AND (vigente_hasta IS NULL OR v_fecha<vigente_hasta)
      ORDER BY version_ref,conocido_desde DESC,revision DESC) p
   WHERE NOT retirado AND estado IN ('aprobada','publicada','sustituida');
 ELSE
  SELECT count(*),min('plantilla:'||version_ref::text) INTO v_n,v_plantilla
   FROM (SELECT DISTINCT ON (version_ref) * FROM vec_personal.version_plantilla_historia
      WHERE organismo_ref=v_organismo AND version_ref=v_plantilla_uuid AND conocido_desde<=v_conocido
        AND vigente_desde<=v_fecha AND (vigente_hasta IS NULL OR v_fecha<vigente_hasta)
      ORDER BY version_ref,conocido_desde DESC,revision DESC) p
   WHERE NOT retirado AND estado IN ('aprobada','publicada','sustituida');
 END IF;
 IF v_n>1 THEN RAISE EXCEPTION 'versiones de plantilla superpuestas' USING ERRCODE='55000'; END IF;
 v_plantilla:=coalesce(v_plantilla,'');
 v_plantilla_uuid:=CASE WHEN v_plantilla='' THEN NULL ELSE substr(v_plantilla,11)::uuid END;
 -- Consumo y auditoría AD3 preceden exactamente a la lectura, en la misma TX.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_organismo
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_sha THEN
  RAISE EXCEPTION 'consumo histórico divergente' USING ERRCODE='42501';
 END IF;
 WITH hechos AS (
SELECT 'unidades'::text AS clase, 'orgunidad:'||h.nodo_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('orgunidad:'||h.nodo_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'catalogo_id',h.catalogo_ref,'catalogo_version',h.catalogo_version,'catalogo_revision',h.catalogo_revision,'clave_catalogo',h.catalogo_entrada_clave,'padre_id',CASE WHEN h.centro_padre_ref IS NULL THEN '' ELSE 'orgunidad:'||h.centro_padre_ref::text END,'tipo',h.clase,'etiqueta',h.denominacion) AS objeto
 FROM (SELECT DISTINCT ON (nodo_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.org_nodo_historia k
     WHERE k.nodo_ref=x.nodo_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.org_nodo_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY nodo_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)

 UNION ALL
SELECT 'puestos_tipo'::text AS clase, 'ptipo:'||h.tipo_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('ptipo:'||h.tipo_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'version_rpt_ref',v_rpt,'codigo_fuente',h.codigo_fila_fuente,'unidad_id',h.unidad_ref,'denominacion',h.denominacion,'clasificacion_ref',h.clasificacion_ref) AS objeto
 FROM (SELECT DISTINCT ON (tipo_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.puesto_tipo_historia k
     WHERE k.tipo_ref=x.tipo_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.puesto_tipo_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY tipo_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)
 AND v_rpt<>'' AND h.rpt_version_ref=v_rpt_uuid
 UNION ALL
SELECT 'dotaciones'::text AS clase, 'dotacion:'||h.dotacion_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('dotacion:'||h.dotacion_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'version_rpt_ref',v_rpt,'puesto_tipo_id','ptipo:'||h.tipo_ref::text,'cantidad',h.cantidad) AS objeto
 FROM (SELECT DISTINCT ON (dotacion_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.dotacion_rpt_historia k
     WHERE k.dotacion_ref=x.dotacion_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.dotacion_rpt_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY dotacion_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)
 AND v_rpt<>'' AND EXISTS (SELECT 1 FROM vec_personal.puesto_tipo_historia t WHERE t.tipo_ref=h.tipo_ref AND t.revision=h.tipo_revision AND t.rpt_version_ref=v_rpt_uuid)
 UNION ALL
SELECT 'plazas'::text AS clase, 'plaza:'||h.plaza_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('plaza:'||h.plaza_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'version_plantilla_ref',v_plantilla,'codigo_fuente',h.codigo_plaza_fuente,'clasificacion_ref',h.clasificacion_ref,'unidad_id',h.unidad_ref,'estado_estructural',h.estado_estructural) AS objeto
 FROM (SELECT DISTINCT ON (plaza_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.plaza_plantilla_historia k
     WHERE k.plaza_ref=x.plaza_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.plaza_plantilla_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY plaza_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)
 AND v_plantilla<>'' AND h.plantilla_version_ref=v_plantilla_uuid
 UNION ALL
SELECT 'puestos_individuales'::text AS clase, 'puesto:'||h.puesto_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('puesto:'||h.puesto_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'version_rpt_ref',v_rpt,'codigo_fuente',h.codigo_puesto_fuente,'puesto_tipo_id','ptipo:'||h.tipo_ref::text,'unidad_id',h.unidad_ref,'estado_estructural',h.estado_estructural) AS objeto
 FROM (SELECT DISTINCT ON (puesto_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.puesto_rpt_historia k
     WHERE k.puesto_ref=x.puesto_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.puesto_rpt_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY puesto_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)
 AND v_rpt<>'' AND EXISTS (SELECT 1 FROM vec_personal.puesto_tipo_historia t WHERE t.tipo_ref=h.tipo_ref AND t.revision=h.tipo_revision AND t.rpt_version_ref=v_rpt_uuid)
 UNION ALL
SELECT 'vinculos'::text AS clase, 'vinculo:'||h.vinculo_ref::text AS id,
 jsonb_build_object('traza',vec_personal.traza_organizacion_v1('vinculo:'||h.vinculo_ref::text,h.revision,
   h.fuente_ref,h.acto_ref,h.huella_fuente_sha256,h.vigente_desde,h.vigente_hasta,
   h.conocido_desde,h.conocido_hasta),'plaza_id','plaza:'||h.plaza_ref::text,'puesto_id','puesto:'||h.puesto_ref::text) AS objeto
 FROM (SELECT DISTINCT ON (vinculo_ref) x.*,
    (SELECT min(k.conocido_desde) FROM vec_personal.vinculo_plaza_puesto_historia k
     WHERE k.vinculo_ref=x.vinculo_ref AND k.conocido_desde>x.conocido_desde
       AND k.conocido_desde<=v_conocido
       AND k.vigente_desde<=v_fecha AND (k.vigente_hasta IS NULL OR v_fecha<k.vigente_hasta)) AS conocido_hasta
   FROM vec_personal.vinculo_plaza_puesto_historia x
   WHERE x.organismo_ref=v_organismo
     AND x.conocido_desde<=v_conocido
     AND x.vigente_desde<=v_fecha AND (x.vigente_hasta IS NULL OR v_fecha<x.vigente_hasta)
   ORDER BY vinculo_ref,conocido_desde DESC,revision DESC) h
 WHERE NOT h.retirado AND (v_unidad='' OR h.unidad_ref=v_unidad)
 AND v_rpt<>'' AND v_plantilla<>'' AND EXISTS (SELECT 1 FROM vec_personal.plaza_plantilla_historia p WHERE p.plaza_ref=h.plaza_ref AND p.revision=h.plaza_revision AND p.plantilla_version_ref=v_plantilla_uuid) AND EXISTS (SELECT 1 FROM vec_personal.puesto_rpt_historia t JOIN vec_personal.puesto_tipo_historia pt ON pt.tipo_ref=t.tipo_ref AND pt.revision=t.tipo_revision WHERE t.puesto_ref=h.puesto_ref AND t.revision=h.puesto_revision AND pt.rpt_version_ref=v_rpt_uuid)
 ), pagina AS (
  SELECT clase,id,objeto FROM hechos ORDER BY clase,id
  OFFSET v_offset LIMIT v_limite+1
 )
 SELECT jsonb_build_object(
  'unidades',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='unidades' AND posicion<=v_limite),'[]'::jsonb),
  'puestos_tipo',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='puestos_tipo' AND posicion<=v_limite),'[]'::jsonb),
  'dotaciones',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='dotaciones' AND posicion<=v_limite),'[]'::jsonb),
  'plazas',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='plazas' AND posicion<=v_limite),'[]'::jsonb),
  'puestos_individuales',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='puestos_individuales' AND posicion<=v_limite),'[]'::jsonb),
  'vinculos',coalesce(jsonb_agg(objeto ORDER BY id) FILTER (WHERE clase='vinculos' AND posicion<=v_limite),'[]'::jsonb)
 ),count(*) FILTER(WHERE posicion<=v_limite),count(*)>v_limite
 INTO v_arrays,v_n,v_mas
 FROM (SELECT *,row_number() OVER (ORDER BY clase,id) AS posicion FROM pagina) z;
 v_ahora:=clock_timestamp();
 IF v_ahora >= v_valida_hasta THEN
  RAISE EXCEPTION 'consulta histórica caducada' USING ERRCODE='42501';
 END IF;
 v_recibo:='orgconsulta:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_consulta_organizacion
  (recibo_ref,organismo_ref,unidad_ref,material_sha256,decision_ref,auditoria_ref,
   consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(v_recibo,v_organismo,v_unidad,v_material_sha,v_consumo.decision_ref,
  v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_n,v_ahora);
 IF v_mas THEN v_cursor_siguiente:='p_'||(v_offset+v_limite)::text||'_'||v_base_sha;
 ELSE v_cursor_siguiente:=''; END IF;
 v_cobertura:=jsonb_build_object(
  'unidades',CASE WHEN jsonb_array_length(v_arrays->'unidades')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END,
  'puestos_tipo',CASE WHEN jsonb_array_length(v_arrays->'puestos_tipo')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END,
  'dotaciones',CASE WHEN jsonb_array_length(v_arrays->'dotaciones')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END,
  'plazas',CASE WHEN jsonb_array_length(v_arrays->'plazas')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END,
  'puestos_individuales',CASE WHEN jsonb_array_length(v_arrays->'puestos_individuales')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END,
  'vinculos',CASE WHEN jsonb_array_length(v_arrays->'vinculos')=0 AND v_offset=0 AND NOT v_mas THEN 'sin_datos' ELSE 'parcial' END
 );
 RETURN jsonb_build_object(
  'pagina',jsonb_build_object(
   'selector',jsonb_build_object('organismo_ref',v_organismo,'unidad_clave',v_unidad,
    'vigente_en',v_fecha::text,'conocido_en',m->>'conocido_en',
    'version_rpt_ref',m->>'version_rpt_ref','version_plantilla_ref',m->>'version_plantilla_ref',
    'limite',v_limite,'cursor',v_cursor),
   'version_rpt_ref',v_rpt,'version_plantilla_ref',v_plantilla,
   'cobertura',v_cobertura,
   'unidades',v_arrays->'unidades',
   'puestos_tipo',v_arrays->'puestos_tipo',
   'dotaciones',v_arrays->'dotaciones',
   'plazas',v_arrays->'plazas',
   'puestos_individuales',v_arrays->'puestos_individuales',
   'vinculos',v_arrays->'vinculos',
   'cursor_siguiente',v_cursor_siguiente),
  'evidencia',jsonb_build_object('recibo_ref',v_recibo,
   'decision_ref',v_consumo.decision_ref,'efecto_ref',v_consumo.efecto_ref,
   'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
   'auditoria_ref',v_consumo.auditoria_ref,'consultada_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMENT ON FUNCTION vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 IS 'Consulta B3 con autorización V3 consumida y auditoría; historia estructural sin ocupaciones ni vacantes.';
COMMIT;
