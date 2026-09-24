\set ON_ERROR_STOP on
-- Registro B2. Cada revisión es un hecho nuevo; no se corrige una fila antigua.
-- El alta sintética histórica y la fachada de Dietas siguen teniendo autoridad
-- y contratos propios. Este registro no los convierte en empleo canónico.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000017:registro-b2',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_personal_propietario'
    OR to_regclass('vec_personal.relacion_servicio_historia') IS NOT NULL
    OR to_regclass('vec_personal.ocupacion_empleado_historia') IS NOT NULL
    OR to_regclass('vec_personal.servicio_reconocido_historia') IS NOT NULL
    OR to_regclass('vec_personal.situacion_empleado_historia') IS NOT NULL
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL
    OR to_regclass('vec_personal.plaza_plantilla_historia') IS NULL
    OR to_regclass('vec_personal.puesto_rpt_historia') IS NULL
 THEN RAISE EXCEPTION 'Personal 000017: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
BEGIN
 RAISE EXCEPTION 'Registro de Personal inmutable' USING ERRCODE='55000';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1() FROM PUBLIC,vec_personal_ejecutor;

-- La referencia de relación es única por acto jurídico. Un empleado mantiene
-- varias relaciones simultáneas; el usuario/acto selecciona cada rel_.
CREATE TABLE vec_personal.relacion_servicio_historia (
 relacion_ref text NOT NULL CHECK(relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 revision integer NOT NULL CHECK(revision>0),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 organismo_ref text NOT NULL CHECK(organismo_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 regimen_ref text NOT NULL CHECK(regimen_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 modalidad_ref text NOT NULL CHECK(modalidad_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','suspendida','finalizada')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date CHECK(vigente_hasta IS NULL OR isfinite(vigente_hasta)),
 conocido_desde timestamptz(6) NOT NULL CHECK(isfinite(conocido_desde)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 firma_oficial boolean NOT NULL DEFAULT false CHECK(NOT firma_oficial),
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK(NOT eficacia_administrativa),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 PRIMARY KEY(relacion_ref,revision),
 UNIQUE(relacion_ref,revision,empleado_ref,organismo_ref),
 UNIQUE(relacion_ref,conocido_desde),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE INDEX relacion_servicio_empleado_idx ON vec_personal.relacion_servicio_historia(empleado_ref,conocido_desde DESC);

-- La ocupación identifica individualmente plaza y, si procede, puesto. La
-- revisión estructural se conserva; no se acepta una dotación agrupada.
CREATE TABLE vec_personal.ocupacion_empleado_historia (
 ocupacion_ref text NOT NULL CHECK(ocupacion_ref ~ '^ocu_[A-Za-z0-9_-]{22,128}$'),
 revision integer NOT NULL CHECK(revision>0),
 relacion_ref text NOT NULL,
 relacion_revision integer NOT NULL,
 empleado_ref text NOT NULL,
 organismo_ref text NOT NULL,
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 plaza_ref uuid NOT NULL,
 plaza_revision integer NOT NULL,
 puesto_ref uuid,
 puesto_revision integer,
 clase text NOT NULL CHECK(clase IN ('titular','provisional','temporal','reserva')),
 modalidad_ref text NOT NULL CHECK(modalidad_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','finalizada')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date CHECK(vigente_hasta IS NULL OR isfinite(vigente_hasta)),
 conocido_desde timestamptz(6) NOT NULL CHECK(isfinite(conocido_desde)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 firma_oficial boolean NOT NULL DEFAULT false CHECK(NOT firma_oficial),
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK(NOT eficacia_administrativa),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 PRIMARY KEY(ocupacion_ref,revision),
 UNIQUE(ocupacion_ref,conocido_desde),
 FOREIGN KEY(relacion_ref,relacion_revision,empleado_ref,organismo_ref)
   REFERENCES vec_personal.relacion_servicio_historia(relacion_ref,revision,empleado_ref,organismo_ref),
 FOREIGN KEY(plaza_ref,plaza_revision,organismo_ref)
   REFERENCES vec_personal.plaza_plantilla_historia(plaza_ref,revision,organismo_ref),
 FOREIGN KEY(puesto_ref,puesto_revision,organismo_ref)
   REFERENCES vec_personal.puesto_rpt_historia(puesto_ref,revision,organismo_ref),
 CHECK((puesto_ref IS NULL)=(puesto_revision IS NULL)),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE INDEX ocupacion_empleado_idx ON vec_personal.ocupacion_empleado_historia(empleado_ref,conocido_desde DESC);
CREATE INDEX ocupacion_plaza_idx ON vec_personal.ocupacion_empleado_historia(plaza_ref,conocido_desde DESC);

-- Servicios reconocidos son resoluciones, no horas de Cronos ni suma derivada
-- de contratos. Días y tramo se inscriben tal como se acreditan en el acto.
CREATE TABLE vec_personal.servicio_reconocido_historia (
 servicio_ref text NOT NULL CHECK(servicio_ref ~ '^srv_[A-Za-z0-9_-]{22,128}$'),
 revision integer NOT NULL CHECK(revision>0),
 relacion_ref text NOT NULL,
 relacion_revision integer NOT NULL,
 empleado_ref text NOT NULL,
 organismo_ref text NOT NULL,
 clase_ref text NOT NULL CHECK(clase_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 dias_reconocidos integer NOT NULL CHECK(dias_reconocidos>=0),
 periodo_desde date NOT NULL CHECK(isfinite(periodo_desde)),
 periodo_hasta date NOT NULL CHECK(isfinite(periodo_hasta) AND periodo_hasta>periodo_desde),
 estado text NOT NULL CHECK(estado IN ('declarado','comprobado','reconocido')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date CHECK(vigente_hasta IS NULL OR isfinite(vigente_hasta)),
 conocido_desde timestamptz(6) NOT NULL CHECK(isfinite(conocido_desde)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 firma_oficial boolean NOT NULL DEFAULT false CHECK(NOT firma_oficial),
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK(NOT eficacia_administrativa),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 PRIMARY KEY(servicio_ref,revision),
 UNIQUE(servicio_ref,conocido_desde),
 FOREIGN KEY(relacion_ref,relacion_revision,empleado_ref,organismo_ref)
   REFERENCES vec_personal.relacion_servicio_historia(relacion_ref,revision,empleado_ref,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE INDEX servicio_reconocido_empleado_idx ON vec_personal.servicio_reconocido_historia(empleado_ref,conocido_desde DESC);

-- El código de situación procede de catálogo gobernado por régimen. Sus
-- efectos (reserva, servicios, nómina) no se infieren aquí de un texto libre.
CREATE TABLE vec_personal.situacion_empleado_historia (
 situacion_ref text NOT NULL CHECK(situacion_ref ~ '^sit_[A-Za-z0-9_-]{22,128}$'),
 revision integer NOT NULL CHECK(revision>0),
 relacion_ref text NOT NULL,
 relacion_revision integer NOT NULL,
 empleado_ref text NOT NULL,
 organismo_ref text NOT NULL,
 situacion_codigo text NOT NULL CHECK(situacion_codigo ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','finalizada','rectificada')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date CHECK(vigente_hasta IS NULL OR isfinite(vigente_hasta)),
 conocido_desde timestamptz(6) NOT NULL CHECK(isfinite(conocido_desde)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 firma_oficial boolean NOT NULL DEFAULT false CHECK(NOT firma_oficial),
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK(NOT eficacia_administrativa),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 PRIMARY KEY(situacion_ref,revision),
 UNIQUE(situacion_ref,conocido_desde),
 FOREIGN KEY(relacion_ref,relacion_revision,empleado_ref,organismo_ref)
   REFERENCES vec_personal.relacion_servicio_historia(relacion_ref,revision,empleado_ref,organismo_ref),
 CHECK(vigente_hasta IS NULL OR vigente_hasta>vigente_desde)
);
CREATE INDEX situacion_empleado_idx ON vec_personal.situacion_empleado_historia(empleado_ref,conocido_desde DESC);

-- La ausencia de ocupación no prueba una vacante si no se ha recibido toda la
-- plantilla de ocupaciones. Solo una fuente reconciliada puede acreditar un
-- intervalo completo; este corte no crea ni importa tales afirmaciones.
CREATE TABLE vec_personal.cobertura_ocupaciones_historia (
 cobertura_ref uuid NOT NULL,
 revision integer NOT NULL CHECK(revision>0),
 organismo_ref text NOT NULL CHECK(organismo_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 plantilla_version_ref uuid NOT NULL,
 plantilla_revision integer NOT NULL,
 estado text NOT NULL CHECK(estado IN ('parcial','completa','revocada')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date NOT NULL CHECK(isfinite(vigente_hasta) AND vigente_hasta>vigente_desde),
 conocido_desde timestamptz(6) NOT NULL CHECK(isfinite(conocido_desde)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(cobertura_ref,revision),
 UNIQUE(cobertura_ref,conocido_desde),
 FOREIGN KEY(plantilla_version_ref,plantilla_revision,organismo_ref)
  REFERENCES vec_personal.version_plantilla_historia(version_ref,revision,organismo_ref)
);
CREATE INDEX cobertura_ocupaciones_ambito_idx ON vec_personal.cobertura_ocupaciones_historia(organismo_ref,conocido_desde DESC);

-- Continuidad por agregado y eje de conocimiento, incluso para escrituras
-- del propietario fuera de las fachadas. Los locks serializan dos revisiones
-- concurrentes antes de leer la última. El empleado y la relación de una
-- revisión nunca pueden cambiar silenciosamente.
CREATE FUNCTION vec_personal.validar_revision_registro_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE clave text; ultima record;
BEGIN
 clave:=CASE TG_TABLE_NAME
   WHEN 'relacion_servicio_historia' THEN to_jsonb(NEW)->>'relacion_ref'
   WHEN 'ocupacion_empleado_historia' THEN to_jsonb(NEW)->>'ocupacion_ref'
   WHEN 'servicio_reconocido_historia' THEN to_jsonb(NEW)->>'servicio_ref'
   WHEN 'situacion_empleado_historia' THEN to_jsonb(NEW)->>'situacion_ref'
   ELSE NULL END;
 IF clave IS NULL THEN RAISE EXCEPTION 'historia desconocida' USING ERRCODE='55000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:registro-b2:'||TG_TABLE_NAME||':'||clave,0));
 EXECUTE format('SELECT revision,empleado_ref,relacion_ref,organismo_ref,%s AS persona_ref,conocido_desde FROM vec_personal.%I WHERE %I=$1 ORDER BY revision DESC LIMIT 1',
   CASE WHEN TG_TABLE_NAME='relacion_servicio_historia' THEN 'persona_ref' ELSE 'NULL::text' END,
   TG_TABLE_NAME,CASE TG_TABLE_NAME WHEN 'relacion_servicio_historia' THEN 'relacion_ref'
     WHEN 'ocupacion_empleado_historia' THEN 'ocupacion_ref'
     WHEN 'servicio_reconocido_historia' THEN 'servicio_ref' ELSE 'situacion_ref' END)
   INTO ultima USING clave;
 NEW.conocido_desde:=clock_timestamp();
 IF NOT FOUND THEN
  IF NEW.revision<>1 THEN RAISE EXCEPTION 'primera revisión inválida' USING ERRCODE='23505'; END IF;
 ELSE
  IF NEW.revision<>ultima.revision+1 OR NEW.empleado_ref<>ultima.empleado_ref
     OR NEW.organismo_ref<>ultima.organismo_ref
     OR (TG_TABLE_NAME<>'relacion_servicio_historia' AND NEW.relacion_ref<>ultima.relacion_ref) THEN
   RAISE EXCEPTION 'continuidad de revisión inválida' USING ERRCODE='23505';
  END IF;
  IF TG_TABLE_NAME='relacion_servicio_historia' THEN
   IF NEW.persona_ref<>ultima.persona_ref THEN
    RAISE EXCEPTION 'persona de relación inmutable' USING ERRCODE='23505';
   END IF;
  END IF;
  NEW.conocido_desde:=greatest(NEW.conocido_desde,ultima.conocido_desde + interval '1 microsecond');
 END IF;
 IF TG_TABLE_NAME='ocupacion_empleado_historia' THEN
  IF NOT EXISTS (
    SELECT 1 FROM vec_personal.plaza_plantilla_historia p
    JOIN vec_personal.version_plantilla_historia v
      ON v.version_ref=p.plantilla_version_ref AND v.revision=p.plantilla_revision
    WHERE p.plaza_ref=NEW.plaza_ref AND p.revision=NEW.plaza_revision
      AND p.organismo_ref=NEW.organismo_ref AND p.estado_estructural='vigente'
      AND p.dotacion_presupuestaria='acreditada' AND NOT p.retirado
      AND v.estado='publicada' AND NOT v.retirado
      AND p.vigente_desde<=NEW.vigente_desde
      AND (p.vigente_hasta IS NULL OR NEW.vigente_desde<p.vigente_hasta)
      AND v.vigente_desde<=NEW.vigente_desde
      AND (v.vigente_hasta IS NULL OR NEW.vigente_desde<v.vigente_hasta)
  ) THEN RAISE EXCEPTION 'plaza individual no acreditada' USING ERRCODE='42501'; END IF;
  IF NEW.puesto_ref IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM vec_personal.puesto_rpt_historia p
    JOIN vec_personal.version_rpt_historia v
      ON v.version_ref=(SELECT t.rpt_version_ref FROM vec_personal.puesto_tipo_historia t
                         WHERE t.tipo_ref=p.tipo_ref AND t.revision=p.tipo_revision)
    WHERE p.puesto_ref=NEW.puesto_ref AND p.revision=NEW.puesto_revision
      AND p.organismo_ref=NEW.organismo_ref AND p.estado_estructural='vigente'
      AND NOT p.retirado AND v.estado='publicada' AND NOT v.retirado
      AND p.vigente_desde<=NEW.vigente_desde
      AND (p.vigente_hasta IS NULL OR NEW.vigente_desde<p.vigente_hasta)
  ) THEN RAISE EXCEPTION 'puesto individual no acreditado' USING ERRCODE='42501'; END IF;
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_revision_registro_empleado_v1() FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.validar_revision_cobertura_ocupaciones_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE previa record;
BEGIN
 -- B3 000011 aún no publica ni reconcilia un universo de ocupaciones. Una
 -- migración posterior deberá aportar autoridad V3 y prueba de completitud
 -- antes de permitir estado completa; ninguna carga manual lo suple.
 IF NEW.estado='completa' THEN
  RAISE EXCEPTION 'cobertura de ocupaciones no acreditada' USING ERRCODE='P7401';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cobertura-ocupaciones:'||NEW.cobertura_ref::text,0));
 SELECT revision,organismo_ref,plantilla_version_ref,conocido_desde,estado INTO previa
   FROM vec_personal.cobertura_ocupaciones_historia
   WHERE cobertura_ref=NEW.cobertura_ref ORDER BY revision DESC LIMIT 1;
 NEW.conocido_desde:=clock_timestamp();
 IF NOT FOUND THEN
  IF NEW.revision<>1 THEN RAISE EXCEPTION 'cobertura inicial inválida' USING ERRCODE='23505'; END IF;
 ELSE
  IF NEW.revision<>previa.revision+1 OR NEW.organismo_ref<>previa.organismo_ref
     OR NEW.plantilla_version_ref<>previa.plantilla_version_ref
     OR previa.estado='revocada' THEN
   RAISE EXCEPTION 'continuidad de cobertura inválida' USING ERRCODE='23505';
  END IF;
  NEW.conocido_desde:=greatest(NEW.conocido_desde,previa.conocido_desde + interval '1 microsecond');
 END IF;
 IF NOT EXISTS (
  SELECT 1 FROM vec_personal.version_plantilla_historia p
   WHERE p.version_ref=NEW.plantilla_version_ref AND p.revision=NEW.plantilla_revision
     AND p.organismo_ref=NEW.organismo_ref AND p.estado='publicada' AND NOT p.retirado
     AND p.vigente_desde<=NEW.vigente_desde
     AND (p.vigente_hasta IS NULL OR NEW.vigente_hasta<=p.vigente_hasta)
 ) THEN RAISE EXCEPTION 'plantilla de cobertura no publicada' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_revision_cobertura_ocupaciones_v1() FROM PUBLIC,vec_personal_ejecutor;

-- Protección de todas las historias, también contra TRUNCATE del ejecutor.
DO $proteccion$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['relacion_servicio_historia','ocupacion_empleado_historia',
   'servicio_reconocido_historia','situacion_empleado_historia','cobertura_ocupaciones_historia'] LOOP
  IF nombre<>'cobertura_ocupaciones_historia' THEN
   EXECUTE format('CREATE TRIGGER revision_continua BEFORE INSERT ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_revision_registro_empleado_v1()',nombre);
  ELSE
   EXECUTE 'CREATE TRIGGER revision_continua BEFORE INSERT ON vec_personal.cobertura_ocupaciones_historia FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_revision_cobertura_ocupaciones_v1()';
  END IF;
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',nombre);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',nombre);
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE format('CREATE POLICY propietario_interno ON vec_personal.%I FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true)',nombre);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',nombre);
 END LOOP;
END $proteccion$;
COMMIT;
