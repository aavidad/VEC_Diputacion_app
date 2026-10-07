\set ON_ERROR_STOP on
-- Personal 28: cargo y ejercicio nominales privados. Un perfil RBAC no crea
-- titularidad. La procedencia del acto es obligatoria, sin presumir su validez.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000028:cargos',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
 OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
 OR to_regclass('vec_personal.ocupacion_empleado_historia') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR to_regclass('vec_personal.cargo_competencial_historia') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 28: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Las revisiones son de solo adición. El puntero actual cambia por CAS dentro
-- de la fachada publicada, y está bloqueado hasta COMMIT al leer o escribir.
CREATE TABLE vec_personal.cargo_competencial_historia (
 cargo_ref text NOT NULL CHECK(cargo_ref ~ '^car_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL CHECK(organizacion_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 puesto_ref uuid,puesto_revision integer,
 organo_ref uuid NOT NULL,organo_revision integer NOT NULL CHECK(organo_revision>0),
 denominacion_catalogo_ref text NOT NULL CHECK(denominacion_catalogo_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','retirado')),
 vigente_desde timestamptz(6) NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta timestamptz(6) NOT NULL CHECK(isfinite(vigente_hasta)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 acto_version bigint NOT NULL CHECK(acto_version>0),
 acto_huella_sha256 text NOT NULL CHECK(acto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,recibo_ref text NOT NULL,
 PRIMARY KEY(cargo_ref,version),UNIQUE(recibo_ref),
 FOREIGN KEY(puesto_ref,puesto_revision) REFERENCES vec_personal.puesto_rpt_historia(puesto_ref,revision),
 FOREIGN KEY(organo_ref,organo_revision) REFERENCES vec_personal.org_nodo_historia(nodo_ref,revision),
 CHECK(vigente_hasta>vigente_desde),
 CHECK((puesto_ref IS NULL)=(puesto_revision IS NULL))
);
CREATE TABLE vec_personal.cargo_competencial_actual (
 cargo_ref text PRIMARY KEY,version bigint NOT NULL,huella_sha256 text NOT NULL,
 FOREIGN KEY(cargo_ref,version) REFERENCES vec_personal.cargo_competencial_historia(cargo_ref,version)
);

CREATE TABLE vec_personal.enlace_cargo_competencial_historia (
 enlace_ref text NOT NULL CHECK(enlace_ref ~ '^enc_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 cargo_ref text NOT NULL, cargo_version bigint NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clase text NOT NULL CHECK(clase IN ('titular','delegacion_competencia','delegacion_firma','suplencia')),
 titular_enlace_ref text,titular_enlace_version bigint,titular_enlace_sha256 text,
 delegante_persona_ref text,
 accion_ref text NOT NULL CHECK(accion_ref ~ '^[a-z][a-z0-9_.:-]{2,255}$'),
 recurso_ref text NOT NULL CHECK(recurso_ref ~ '^[a-z][a-z0-9_.:/#-]{2,511}$'),
 finalidad_ref text NOT NULL CHECK(finalidad_ref ~ '^[a-z][a-z0-9_.:-]{2,511}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','retirado')),
 vigente_desde timestamptz(6) NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta timestamptz(6) NOT NULL CHECK(isfinite(vigente_hasta)),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 acto_version bigint NOT NULL CHECK(acto_version>0),
 acto_huella_sha256 text NOT NULL CHECK(acto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 fuente_version bigint NOT NULL CHECK(fuente_version>0),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 -- Solo un empleado exige enlace laboral existente; el electo no lo inventa.
 requiere_enlace_laboral boolean NOT NULL,
 empleado_ref text,ocupacion_ref text,ocupacion_revision integer,
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,recibo_ref text NOT NULL,
 PRIMARY KEY(enlace_ref,version),UNIQUE(recibo_ref),
 FOREIGN KEY(cargo_ref,cargo_version) REFERENCES vec_personal.cargo_competencial_historia(cargo_ref,version),
 FOREIGN KEY(titular_enlace_ref,titular_enlace_version)
   REFERENCES vec_personal.enlace_cargo_competencial_historia(enlace_ref,version),
 FOREIGN KEY(ocupacion_ref,ocupacion_revision) REFERENCES vec_personal.ocupacion_empleado_historia(ocupacion_ref,revision),
 CHECK(vigente_hasta>vigente_desde),
 CHECK((clase='titular' AND titular_enlace_ref IS NULL AND titular_enlace_version IS NULL
        AND titular_enlace_sha256 IS NULL AND delegante_persona_ref IS NULL)
       OR (clase<>'titular' AND titular_enlace_ref IS NOT NULL
           AND titular_enlace_version IS NOT NULL AND titular_enlace_version>0
           AND titular_enlace_sha256 IS NOT NULL
           AND titular_enlace_sha256 ~ '^[0-9a-f]{64}$' AND delegante_persona_ref IS NOT NULL
           AND persona_ref<>delegante_persona_ref)),
 CHECK((empleado_ref IS NULL AND ocupacion_ref IS NULL AND ocupacion_revision IS NULL)
       OR (empleado_ref IS NOT NULL AND ocupacion_ref IS NOT NULL AND ocupacion_revision IS NOT NULL)),
 CHECK(NOT requiere_enlace_laboral OR ocupacion_ref IS NOT NULL)
);
CREATE INDEX enlace_cargo_busqueda_idx ON vec_personal.enlace_cargo_competencial_historia(cargo_ref,clase,persona_ref);
CREATE TABLE vec_personal.enlace_cargo_competencial_actual (
 enlace_ref text PRIMARY KEY,version bigint NOT NULL,huella_sha256 text NOT NULL,
 FOREIGN KEY(enlace_ref,version) REFERENCES vec_personal.enlace_cargo_competencial_historia(enlace_ref,version)
);
CREATE TABLE vec_personal.recibo_publicacion_cargo_competencial (
 clave_idempotencia text PRIMARY KEY CHECK(clave_idempotencia ~ '^[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE,operacion text NOT NULL CHECK(operacion IN ('cargo','enlace')),
 objeto_ref text NOT NULL,version bigint NOT NULL CHECK(version>0),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL CHECK(isfinite(registrada_en))
);

DO $acl$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['cargo_competencial_historia','cargo_competencial_actual',
   'enlace_cargo_competencial_historia','enlace_cargo_competencial_actual','recibo_publicacion_cargo_competencial'] LOOP
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_personal.%I FOR ALL TO vec_personal_propietario USING(current_user=''vec_personal_propietario'') WITH CHECK(current_user=''vec_personal_propietario'')',n);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',n);
  EXECUTE format('REVOKE ALL ON TYPE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor',n);
  IF n NOT IN ('cargo_competencial_actual','enlace_cargo_competencial_actual') THEN
   EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_personal.%I FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',n);
  END IF;
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1()',n);
 END LOOP;
END $acl$;

-- AUT32 invoca esta lectura en su transacción CT. Se bloquean los punteros y
-- versiones publicados, y el mismo cerrojo que usa la escritura de Personal.
-- La fecha histórica viene del canon firmado, nunca de una fecha libre añadida
-- por la petición SQL. También en replay se exige vigencia actual, además de
-- la histórica; una rectificación de la fuente cambia el puntero.
CREATE FUNCTION vec_personal.leer_revalidar_cargo_ocupante_ct_v1(p_contexto_nominal bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE j jsonb;p jsonb;d jsonb;c record;t record;e record;o record;eo record;org record;puesto record;
        cargo_ref text;persona text;organizacion text;unidad text;accion text;recurso text;finalidad text;
        fecha timestamptz(6);ahora timestamptz(6);ejerciente text;tipo text;
        enlace_b2 jsonb;enlace_ejerciente_b2 jsonb;delegacion jsonb;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR current_setting('TimeZone')<>'UTC'
  OR current_setting('role')<>'none'
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
  OR p_contexto_nominal IS NULL OR octet_length(p_contexto_nominal) NOT BETWEEN 1 AND 32768
 THEN RAISE EXCEPTION 'cargo_nominal_denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_contexto_nominal,'UTF8')::jsonb;
  IF j->>'esquema' IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
     OR jsonb_typeof(j->'personal') IS DISTINCT FROM 'object'
     OR jsonb_typeof(j->'recurso') IS DISTINCT FROM 'object'
     OR jsonb_typeof(j->'identidad') IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION 'cargo_nominal_contexto_invalido' USING ERRCODE='22023'; END IF;
  p:=j->'personal';d:=p->'delegacion';
  cargo_ref:=p#>>'{cargo,referencia}';persona:=j#>>'{identidad,persona_ref}';
  organizacion:=j#>>'{recurso,organizacion_ref}';unidad:=j#>>'{recurso,unidad_ref}';
  accion:=j->>'accion';recurso:=j#>>'{recurso,recurso_autorizable_ref}';finalidad:=j->>'finalidad';
  fecha:=(j->>'fecha_historica')::timestamptz;
  IF cargo_ref !~ '^car_[A-Za-z0-9_-]{22,128}$'
   OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
   OR organizacion !~ '^[a-z][a-z0-9_:-]{2,127}$'
   OR unidad !~ '^[a-z][a-z0-9_:-]{2,127}$'
   OR accion IS NULL OR recurso IS NULL OR finalidad IS NULL
   OR NOT isfinite(fecha) OR fecha>clock_timestamp()
   OR p#>>'{cargo,huella_sha256}' !~ '^[0-9a-f]{64}$'
   OR p#>>'{enlace_ocupante,huella_sha256}' !~ '^[0-9a-f]{64}$' THEN
   RAISE EXCEPTION 'cargo_nominal_contexto_invalido' USING ERRCODE='22023'; END IF;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN
  RAISE EXCEPTION 'cargo_nominal_contexto_invalido' USING ERRCODE='22023';
 END;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cargos_competenciales:v1',0));
 -- P10 carece de punteros actuales: bloquear sus historias impide publicar
 -- revisiones estructurales nuevas entre la lectura y el COMMIT de CT.
 LOCK TABLE vec_personal.org_nodo_historia,vec_personal.puesto_rpt_historia IN SHARE MODE;
 ahora:=clock_timestamp();
 SELECT h.* INTO STRICT c FROM vec_personal.cargo_competencial_actual a
 JOIN vec_personal.cargo_competencial_historia h USING(cargo_ref,version)
 WHERE a.cargo_ref=cargo_ref FOR SHARE OF a,h;
 IF c.version::text IS DISTINCT FROM p#>>'{cargo,version}'
  OR c.huella_sha256 IS DISTINCT FROM p#>>'{cargo,huella_sha256}'
  OR c.organizacion_ref IS DISTINCT FROM organizacion OR c.unidad_ref IS DISTINCT FROM unidad
  OR c.estado<>'vigente' OR fecha<c.vigente_desde OR fecha>=c.vigente_hasta
  OR ahora<c.vigente_desde OR ahora>=c.vigente_hasta
  OR c.vigente_desde IS DISTINCT FROM (p->>'cargo_vigente_desde')::timestamptz
  OR c.vigente_hasta IS DISTINCT FROM (p->>'cargo_vigente_hasta')::timestamptz THEN
  RAISE EXCEPTION 'cargo_nominal_fuente_cambiada' USING ERRCODE='42501'; END IF;
 SELECT n.* INTO STRICT org FROM vec_personal.org_nodo_historia n
 WHERE n.nodo_ref=c.organo_ref AND n.revision=c.organo_revision FOR SHARE;
 IF org.organismo_ref IS DISTINCT FROM organizacion OR org.unidad_ref IS DISTINCT FROM unidad
  OR org.retirado OR org.revision IS DISTINCT FROM
   (SELECT max(n.revision) FROM vec_personal.org_nodo_historia n WHERE n.nodo_ref=org.nodo_ref)
  OR org.vigente_desde>fecha::date OR org.vigente_desde>ahora::date
  OR (org.vigente_hasta IS NOT NULL AND
   (org.vigente_hasta<=fecha::date OR org.vigente_hasta<=ahora::date)) THEN
  RAISE EXCEPTION 'cargo_nominal_organo_cambiado' USING ERRCODE='42501'; END IF;
 IF c.puesto_ref IS NOT NULL THEN
  SELECT x.* INTO STRICT puesto FROM vec_personal.puesto_rpt_historia x
  WHERE x.puesto_ref=c.puesto_ref AND x.revision=c.puesto_revision FOR SHARE;
  IF puesto.organismo_ref IS DISTINCT FROM organizacion OR puesto.unidad_ref IS DISTINCT FROM unidad
   OR puesto.retirado OR puesto.estado_estructural<>'vigente'
   OR puesto.revision IS DISTINCT FROM
    (SELECT max(x.revision) FROM vec_personal.puesto_rpt_historia x WHERE x.puesto_ref=puesto.puesto_ref)
   OR puesto.vigente_desde>fecha::date OR puesto.vigente_desde>ahora::date
   OR (puesto.vigente_hasta IS NOT NULL AND
    (puesto.vigente_hasta<=fecha::date OR puesto.vigente_hasta<=ahora::date)) THEN
   RAISE EXCEPTION 'cargo_nominal_puesto_cambiado' USING ERRCODE='42501'; END IF;
 END IF;
 SELECT h.* INTO STRICT t FROM vec_personal.enlace_cargo_competencial_actual a
 JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
 WHERE a.enlace_ref=p#>>'{enlace_ocupante,referencia}' FOR SHARE OF a,h;
 IF t.version::text IS DISTINCT FROM p#>>'{enlace_ocupante,version}'
  OR t.huella_sha256 IS DISTINCT FROM p#>>'{enlace_ocupante,huella_sha256}'
  OR t.cargo_ref IS DISTINCT FROM cargo_ref OR t.cargo_version IS DISTINCT FROM c.version
  OR t.persona_ref IS DISTINCT FROM p->>'ocupante_persona_ref'
  OR t.cargo_ref IS DISTINCT FROM p->>'cargo_ref_enlace'
  OR t.clase<>'titular' OR t.estado<>'vigente'
  OR t.accion_ref IS DISTINCT FROM accion OR t.recurso_ref IS DISTINCT FROM recurso
  OR t.finalidad_ref IS DISTINCT FROM finalidad
  OR fecha<t.vigente_desde OR fecha>=t.vigente_hasta
  OR ahora<t.vigente_desde OR ahora>=t.vigente_hasta
  OR t.vigente_desde IS DISTINCT FROM (p->>'enlace_vigente_desde')::timestamptz
  OR t.vigente_hasta IS DISTINCT FROM (p->>'enlace_vigente_hasta')::timestamptz THEN
  RAISE EXCEPTION 'cargo_nominal_ocupante_cambiado' USING ERRCODE='42501'; END IF;
 ejerciente:=t.persona_ref;tipo:='titular';delegacion:=NULL;enlace_ejerciente_b2:=NULL;
 IF d IS NOT NULL AND d<>'null'::jsonb THEN
  SELECT h.* INTO STRICT e FROM vec_personal.enlace_cargo_competencial_actual a
  JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
  WHERE h.acto_ref=d#>>'{acto,referencia}' AND h.acto_version::text=d#>>'{acto,version}'
   AND h.acto_huella_sha256=d#>>'{acto,huella_sha256}'
   AND h.titular_enlace_ref=t.enlace_ref AND h.titular_enlace_version=t.version
   AND h.titular_enlace_sha256=t.huella_sha256 AND h.persona_ref=persona
   AND h.cargo_ref=cargo_ref AND h.accion_ref=accion
   AND h.recurso_ref=recurso AND h.finalidad_ref=finalidad FOR SHARE OF a,h;
  IF e.clase NOT IN ('delegacion_competencia','delegacion_firma','suplencia')
   OR e.estado<>'vigente' OR e.cargo_ref IS DISTINCT FROM cargo_ref
   OR e.persona_ref IS DISTINCT FROM persona OR e.delegante_persona_ref IS DISTINCT FROM t.persona_ref
   OR e.persona_ref IS DISTINCT FROM d->>'delegado_persona_ref'
   OR e.delegante_persona_ref IS DISTINCT FROM d->>'delegante_persona_ref'
   OR e.cargo_ref IS DISTINCT FROM d->>'cargo_ref'
   OR e.accion_ref IS DISTINCT FROM accion OR e.recurso_ref IS DISTINCT FROM recurso
   OR e.finalidad_ref IS DISTINCT FROM finalidad
   OR fecha<e.vigente_desde OR fecha>=e.vigente_hasta
   OR ahora<e.vigente_desde OR ahora>=e.vigente_hasta
   OR e.vigente_desde IS DISTINCT FROM (d->>'vigente_desde')::timestamptz
   OR e.vigente_hasta IS DISTINCT FROM (d->>'vigente_hasta')::timestamptz THEN
   RAISE EXCEPTION 'cargo_nominal_ejercicio_cambiado' USING ERRCODE='42501'; END IF;
  ejerciente:=e.persona_ref;tipo:=e.clase;
  delegacion:=jsonb_build_object('acto',jsonb_build_object('referencia',e.acto_ref,'version',e.acto_version,'huella_sha256',e.acto_huella_sha256),
    'delegante_persona_ref',e.delegante_persona_ref,'delegado_persona_ref',e.persona_ref,
    'cargo_ref',e.cargo_ref,'vigente_desde',e.vigente_desde,'vigente_hasta',e.vigente_hasta);
  IF e.requiere_enlace_laboral AND e.ocupacion_ref IS NULL THEN
   RAISE EXCEPTION 'cargo_nominal_delegado_sin_enlace_laboral' USING ERRCODE='42501'; END IF;
  IF e.ocupacion_ref IS NOT NULL THEN
   LOCK TABLE vec_personal.relacion_servicio_historia,
    vec_personal.ocupacion_empleado_historia IN SHARE MODE;
   SELECT x.*,r.estado AS relacion_estado,r.vigente_desde AS relacion_desde,
     r.vigente_hasta AS relacion_hasta INTO STRICT eo
   FROM vec_personal.ocupacion_empleado_historia x
   JOIN vec_personal.relacion_servicio_historia r
    ON r.relacion_ref=x.relacion_ref AND r.revision=x.relacion_revision
   WHERE x.ocupacion_ref=e.ocupacion_ref AND x.revision=e.ocupacion_revision
    AND x.empleado_ref=e.empleado_ref AND r.persona_ref=e.persona_ref FOR SHARE OF x,r;
   IF eo.estado<>'vigente' OR eo.relacion_estado<>'vigente'
    OR eo.organismo_ref IS DISTINCT FROM organizacion OR eo.unidad_ref IS DISTINCT FROM unidad
    OR eo.revision IS DISTINCT FROM
      (SELECT max(x.revision) FROM vec_personal.ocupacion_empleado_historia x WHERE x.ocupacion_ref=eo.ocupacion_ref)
    OR eo.relacion_revision IS DISTINCT FROM
      (SELECT max(x.revision) FROM vec_personal.relacion_servicio_historia x WHERE x.relacion_ref=eo.relacion_ref)
    OR eo.vigente_desde>fecha::date OR eo.vigente_desde>ahora::date
    OR eo.relacion_desde>fecha::date OR eo.relacion_desde>ahora::date
    OR (eo.vigente_hasta IS NOT NULL AND (eo.vigente_hasta<=fecha::date OR eo.vigente_hasta<=ahora::date))
    OR (eo.relacion_hasta IS NOT NULL AND (eo.relacion_hasta<=fecha::date OR eo.relacion_hasta<=ahora::date)) THEN
    RAISE EXCEPTION 'cargo_nominal_delegado_laboral_cambiado' USING ERRCODE='42501'; END IF;
   enlace_ejerciente_b2:=jsonb_build_object('empleado_ref',e.empleado_ref,
    'ocupacion_ref',e.ocupacion_ref,'ocupacion_revision',e.ocupacion_revision,
    'relacion_ref',eo.relacion_ref,'relacion_revision',eo.relacion_revision);
  END IF;
 ELSE
  IF persona IS DISTINCT FROM t.persona_ref THEN
   RAISE EXCEPTION 'cargo_nominal_persona_divergente' USING ERRCODE='42501'; END IF;
 END IF;
 enlace_b2:=NULL;
 IF t.requiere_enlace_laboral AND t.ocupacion_ref IS NULL THEN
  RAISE EXCEPTION 'cargo_nominal_titular_sin_enlace_laboral' USING ERRCODE='42501'; END IF;
 IF t.ocupacion_ref IS NOT NULL THEN
  -- B2 tampoco expone un puntero actual de relación/ocupación.
  LOCK TABLE vec_personal.relacion_servicio_historia,
   vec_personal.ocupacion_empleado_historia IN SHARE MODE;
  SELECT o.*,r.persona_ref,r.estado AS relacion_estado,
    r.vigente_desde AS relacion_desde,r.vigente_hasta AS relacion_hasta
  INTO STRICT o FROM vec_personal.ocupacion_empleado_historia o
  JOIN vec_personal.relacion_servicio_historia r
    ON r.relacion_ref=o.relacion_ref AND r.revision=o.relacion_revision
  WHERE o.ocupacion_ref=t.ocupacion_ref AND o.revision=t.ocupacion_revision
    AND o.empleado_ref=t.empleado_ref AND r.persona_ref=t.persona_ref
  FOR SHARE OF o,r;
  IF o.estado<>'vigente' OR o.relacion_estado<>'vigente'
    OR o.revision IS DISTINCT FROM (SELECT max(x.revision) FROM vec_personal.ocupacion_empleado_historia x
      WHERE x.ocupacion_ref=o.ocupacion_ref)
    OR o.relacion_revision IS DISTINCT FROM (SELECT max(x.revision) FROM vec_personal.relacion_servicio_historia x
      WHERE x.relacion_ref=o.relacion_ref)
    OR o.relacion_desde>fecha::date OR o.relacion_desde>ahora::date
    OR (o.relacion_hasta IS NOT NULL AND
      (o.relacion_hasta<=fecha::date OR o.relacion_hasta<=ahora::date))
    OR (c.puesto_ref IS NOT NULL AND
      (o.puesto_ref IS DISTINCT FROM c.puesto_ref OR o.puesto_revision IS DISTINCT FROM c.puesto_revision))
    OR o.organismo_ref IS DISTINCT FROM organizacion
    OR o.unidad_ref IS DISTINCT FROM unidad OR o.vigente_desde>fecha::date
    OR o.vigente_desde>ahora::date
    OR (o.vigente_hasta IS NOT NULL AND (o.vigente_hasta<=fecha::date OR o.vigente_hasta<=ahora::date)) THEN
   RAISE EXCEPTION 'cargo_nominal_enlace_laboral_cambiado' USING ERRCODE='42501'; END IF;
  enlace_b2:=jsonb_build_object('empleado_ref',t.empleado_ref,'ocupacion_ref',t.ocupacion_ref,
    'ocupacion_revision',t.ocupacion_revision,'relacion_ref',o.relacion_ref,'relacion_revision',o.relacion_revision);
 END IF;
 IF d IS NULL OR d='null'::jsonb THEN enlace_ejerciente_b2:=enlace_b2; END IF;
 RETURN jsonb_build_object('esquema','vec.personal.cargo-ocupante.ct.v1',
  'cargo',jsonb_build_object('referencia',c.cargo_ref,'version',c.version,'huella_sha256',c.huella_sha256),
  'enlace_ocupante',jsonb_build_object('referencia',t.enlace_ref,'version',t.version,'huella_sha256',t.huella_sha256),
  'ocupante_persona_ref',t.persona_ref,'cargo_ref_enlace',t.cargo_ref,
  'cargo_vigente_desde',c.vigente_desde,'cargo_vigente_hasta',c.vigente_hasta,
  'enlace_vigente_desde',t.vigente_desde,'enlace_vigente_hasta',t.vigente_hasta,
  'delegacion',delegacion,'organizacion_ref',c.organizacion_ref,'unidad_ref',c.unidad_ref,
  'puesto_ref',c.puesto_ref,'persona_ejerciente_ref',ejerciente,'tipo_ejercicio',tipo,
  'enlace_personal',enlace_b2,'enlace_ejerciente_personal',enlace_ejerciente_b2,
  'procedencia_ref',t.fuente_ref,'recibo_ref',t.recibo_ref);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'cargo_nominal_fuente_ausente_o_ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea) FROM PUBLIC,vec_personal_ejecutor;
GRANT USAGE ON SCHEMA vec_personal TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_personal.publicar_cargo_competencial_v1(
 p_material bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb;dato jsonb;c jsonb;d jsonb;v record;actual record;cargo record;titular record;org record;puesto record;
  recibo record;operacion text;objeto text;clave text;sha text;dato_sha text;
  organizacion text;unidad text;recurso_canonico text;recurso_sha text;
  esperada bigint;version_nueva bigint;fecha timestamptz(6);recibo_ref text;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
  OR current_setting('role')<>'none'
  OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
  OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 32768
  OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
  OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'cargo_publicacion_denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=convert_from(p_material,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  dato:=m->'datos';operacion:=m->>'operacion';clave:=m->>'clave_idempotencia';
  objeto:=m->>'objeto_ref';organizacion:=m->>'organizacion_ref';unidad:=m->>'unidad_ref';
  esperada:=(m->>'version_esperada')::bigint;
  version_nueva:=(dato->>'version')::bigint;
  IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(dato) IS DISTINCT FROM 'object'
   OR NOT (m ?& ARRAY['esquema','operacion','clave_idempotencia','objeto_ref',
    'organizacion_ref','unidad_ref','version_esperada','huella_esperada','datos'])
   OR (SELECT count(*) FROM jsonb_object_keys(m))<>9
   OR m->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-competencial.publicacion.v1'
   OR operacion NOT IN ('cargo','enlace') OR clave !~ '^[0-9a-f]{32}$'
   OR organizacion !~ '^[a-z][a-z0-9_:-]{2,127}$'
   OR unidad !~ '^[a-z][a-z0-9_:-]{2,127}$'
   OR jsonb_typeof(m->'version_esperada') IS DISTINCT FROM 'number'
   OR (m->>'version_esperada') !~ '^(0|[1-9][0-9]*)$'
   OR jsonb_typeof(dato->'version') IS DISTINCT FROM 'number'
   OR (dato->>'version') !~ '^[1-9][0-9]*$'
   OR esperada IS NULL OR esperada<0 OR version_nueva IS DISTINCT FROM esperada+1
   OR (esperada>0 AND (m->>'huella_esperada' IS NULL
       OR m->>'huella_esperada' !~ '^[0-9a-f]{64}$'))
   OR (esperada=0 AND m->>'huella_esperada' IS NOT NULL)
   OR dato->>'estado' NOT IN ('vigente','retirado')
   OR (operacion='cargo' AND (objeto !~ '^car_[A-Za-z0-9_-]{22,128}$' OR dato->>'cargo_ref' IS DISTINCT FROM objeto))
   OR (operacion='enlace' AND (objeto !~ '^enc_[A-Za-z0-9_-]{22,128}$' OR dato->>'enlace_ref' IS DISTINCT FROM objeto))
   OR d->>'concedida' IS DISTINCT FROM 'true'
   OR d->>'accion' IS DISTINCT FROM 'personal.cargo_competencial.publicar'
   OR d->>'modulo_id' IS DISTINCT FROM 'personal'
   OR d->>'tipo_recurso' IS DISTINCT FROM 'cargo_competencial'
   OR d->>'finalidad' IS DISTINCT FROM 'administrar_cargos_competenciales'
   OR d->>'recurso_ref' IS DISTINCT FROM objeto
   OR d->'campos_permitidos' IS DISTINCT FROM '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb
   OR c->>'operacion' IS DISTINCT FROM 'personal.cargo_competencial.publicar'
   OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.cargo_competencial.publicar.v1'
   OR c->>'efecto_ref' IS DISTINCT FROM objeto
   OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref' THEN
   RAISE EXCEPTION 'cargo_publicacion_invalida' USING ERRCODE='22023'; END IF;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
  RAISE EXCEPTION 'cargo_publicacion_invalida' USING ERRCODE='22023';
 END;
 sha:=encode(sha256(p_material),'hex');
 dato_sha:=encode(sha256(convert_to(dato::text,'UTF8')),'hex');
 recurso_canonico:='{"ambitos":{"organizacion_ref":'||to_jsonb(organizacion)::text||
  ',"unidad_ref":'||to_jsonb(unidad)::text||'},"atributos":{"material_sha256":"'||sha||
  '","operacion":'||to_jsonb(operacion)::text||'}}';
 recurso_sha:=encode(sha256(convert_to(recurso_canonico,'UTF8')),'hex');
 IF c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha THEN
  RAISE EXCEPTION 'cargo_publicacion_recurso_divergente' USING ERRCODE='42501'; END IF;
 -- La autorización y la auditoría común preceden a todo replay o escritura.
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_publicacion_cargo_competencial_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.efecto_ref IS DISTINCT FROM objeto OR v.huella_efecto_sha256 IS DISTINCT FROM recurso_sha
  OR v.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
  OR v.auditoria_ref IS NULL OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
  OR v.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'cargo_publicacion_consumo_divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cargos_competenciales:v1',0));
 LOCK TABLE vec_personal.org_nodo_historia,vec_personal.puesto_rpt_historia IN SHARE MODE;
 SELECT * INTO recibo FROM vec_personal.recibo_publicacion_cargo_competencial
 WHERE clave_idempotencia=clave FOR SHARE;
 IF FOUND THEN
  IF recibo.material_sha256 IS DISTINCT FROM sha OR recibo.operacion IS DISTINCT FROM operacion
   OR recibo.objeto_ref IS DISTINCT FROM objeto THEN
   RAISE EXCEPTION 'cargo_publicacion_clave_reutilizada' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('recibo_ref',recibo.recibo_ref,'objeto_ref',objeto,'version',recibo.version,
   'registrada_en',recibo.registrada_en,'auditoria_ref',recibo.auditoria_ref,
   'acceso_actual_auditoria_ref',v.auditoria_ref,'estado_replay','recuperada');
 END IF;
 fecha:=v.consumida_en;
 recibo_ref:='percar_'||replace(gen_random_uuid()::text,'-','');
 IF operacion='cargo' THEN
  IF NOT (dato ?& ARRAY['cargo_ref','version','organizacion_ref','unidad_ref',
   'puesto_ref','puesto_revision','organo_ref','organo_revision','denominacion_catalogo_ref',
   'estado','vigente_desde','vigente_hasta','acto_ref','acto_version','acto_huella_sha256',
   'fuente_ref','fuente_version','fuente_huella_sha256'])
   OR (SELECT count(*) FROM jsonb_object_keys(dato))<>18 THEN
   RAISE EXCEPTION 'cargo_publicacion_campos_invalidos' USING ERRCODE='22023'; END IF;
  SELECT * INTO actual FROM vec_personal.cargo_competencial_actual
   WHERE cargo_ref=objeto FOR UPDATE;
  IF esperada IS NULL OR (esperada=0 AND FOUND) OR (esperada>0 AND (NOT FOUND
    OR actual.version IS DISTINCT FROM esperada
    OR actual.huella_sha256 IS DISTINCT FROM m->>'huella_esperada')) THEN
   RAISE EXCEPTION 'cargo_publicacion_cas_divergente' USING ERRCODE='40001'; END IF;
  IF dato->>'organizacion_ref' IS DISTINCT FROM organizacion
   OR dato->>'unidad_ref' IS DISTINCT FROM unidad
   OR dato->>'organo_ref' IS NULL OR (dato->>'organo_revision')::integer<1
   OR (dato->>'puesto_ref' IS NULL) IS DISTINCT FROM (dato->>'puesto_revision' IS NULL)
   OR dato->>'denominacion_catalogo_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR dato->>'acto_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR (dato->>'acto_version')::bigint<1 OR dato->>'acto_huella_sha256' !~ '^[0-9a-f]{64}$'
   OR dato->>'fuente_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR (dato->>'fuente_version')::bigint<1 OR dato->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
   OR (dato->>'vigente_hasta')::timestamptz <= (dato->>'vigente_desde')::timestamptz THEN
   RAISE EXCEPTION 'cargo_publicacion_datos_invalidos' USING ERRCODE='22023'; END IF;
  SELECT n.* INTO STRICT org FROM vec_personal.org_nodo_historia n
   WHERE n.nodo_ref=(dato->>'organo_ref')::uuid AND n.revision=(dato->>'organo_revision')::integer FOR SHARE;
  IF org.organismo_ref IS DISTINCT FROM organizacion OR org.unidad_ref IS DISTINCT FROM unidad
   OR org.retirado OR org.revision IS DISTINCT FROM
    (SELECT max(n.revision) FROM vec_personal.org_nodo_historia n WHERE n.nodo_ref=org.nodo_ref)
   OR org.vigente_desde>fecha::date OR (org.vigente_hasta IS NOT NULL AND org.vigente_hasta<=fecha::date) THEN
   RAISE EXCEPTION 'cargo_publicacion_organo_no_acreditado' USING ERRCODE='42501'; END IF;
  IF dato->>'puesto_ref' IS NOT NULL THEN
   SELECT x.* INTO STRICT puesto FROM vec_personal.puesto_rpt_historia x
    WHERE x.puesto_ref=(dato->>'puesto_ref')::uuid AND x.revision=(dato->>'puesto_revision')::integer FOR SHARE;
   IF puesto.organismo_ref IS DISTINCT FROM organizacion OR puesto.unidad_ref IS DISTINCT FROM unidad
    OR puesto.retirado OR puesto.estado_estructural<>'vigente'
    OR puesto.revision IS DISTINCT FROM
     (SELECT max(x.revision) FROM vec_personal.puesto_rpt_historia x WHERE x.puesto_ref=puesto.puesto_ref)
    OR puesto.vigente_desde>fecha::date OR (puesto.vigente_hasta IS NOT NULL AND puesto.vigente_hasta<=fecha::date) THEN
    RAISE EXCEPTION 'cargo_publicacion_puesto_no_acreditado' USING ERRCODE='42501'; END IF;
  END IF;
  INSERT INTO vec_personal.cargo_competencial_historia(cargo_ref,version,huella_sha256,
   organizacion_ref,unidad_ref,puesto_ref,puesto_revision,organo_ref,organo_revision,denominacion_catalogo_ref,estado,
   vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,
   fuente_ref,fuente_version,fuente_huella_sha256,publicada_en,decision_ref,auditoria_ref,recibo_ref)
  VALUES(objeto,version_nueva,dato_sha,dato->>'organizacion_ref',dato->>'unidad_ref',
   (dato->>'puesto_ref')::uuid,(dato->>'puesto_revision')::integer,
   (dato->>'organo_ref')::uuid,(dato->>'organo_revision')::integer,
   dato->>'denominacion_catalogo_ref',dato->>'estado',
   (dato->>'vigente_desde')::timestamptz,(dato->>'vigente_hasta')::timestamptz,
   dato->>'acto_ref',(dato->>'acto_version')::bigint,dato->>'acto_huella_sha256',
   dato->>'fuente_ref',(dato->>'fuente_version')::bigint,dato->>'fuente_huella_sha256',
   fecha,v.decision_ref,v.auditoria_ref,recibo_ref);
  INSERT INTO vec_personal.cargo_competencial_actual(cargo_ref,version,huella_sha256)
   VALUES(objeto,version_nueva,dato_sha)
   ON CONFLICT(cargo_ref) DO UPDATE SET version=EXCLUDED.version,huella_sha256=EXCLUDED.huella_sha256;
 ELSE
  IF NOT (dato ?& ARRAY['enlace_ref','version','cargo_ref','cargo_version','persona_ref',
   'clase','titular_enlace_ref','titular_enlace_version','titular_enlace_sha256',
   'delegante_persona_ref','accion_ref','recurso_ref','finalidad_ref','estado',
   'vigente_desde','vigente_hasta','acto_ref','acto_version','acto_huella_sha256',
   'fuente_ref','fuente_version','fuente_huella_sha256','requiere_enlace_laboral',
   'empleado_ref','ocupacion_ref','ocupacion_revision'])
   OR (SELECT count(*) FROM jsonb_object_keys(dato))<>26
   OR jsonb_typeof(dato->'requiere_enlace_laboral') IS DISTINCT FROM 'boolean' THEN
   RAISE EXCEPTION 'cargo_publicacion_campos_invalidos' USING ERRCODE='22023'; END IF;
  SELECT * INTO actual FROM vec_personal.enlace_cargo_competencial_actual
   WHERE enlace_ref=objeto FOR UPDATE;
  IF esperada IS NULL OR (esperada=0 AND FOUND) OR (esperada>0 AND (NOT FOUND
    OR actual.version IS DISTINCT FROM esperada
    OR actual.huella_sha256 IS DISTINCT FROM m->>'huella_esperada')) THEN
   RAISE EXCEPTION 'cargo_publicacion_cas_divergente' USING ERRCODE='40001'; END IF;
  SELECT h.* INTO STRICT cargo FROM vec_personal.cargo_competencial_actual a
    JOIN vec_personal.cargo_competencial_historia h USING(cargo_ref,version)
    WHERE a.cargo_ref=dato->>'cargo_ref' FOR SHARE OF a,h;
  IF cargo.version::text IS DISTINCT FROM dato->>'cargo_version'
   OR cargo.estado<>'vigente' OR cargo.organizacion_ref IS DISTINCT FROM organizacion
   OR cargo.unidad_ref IS DISTINCT FROM unidad
   OR dato->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
   OR dato->>'clase' NOT IN ('titular','delegacion_competencia','delegacion_firma','suplencia')
   OR dato->>'accion_ref' IS NULL OR dato->>'recurso_ref' IS NULL OR dato->>'finalidad_ref' IS NULL
   OR dato->>'acto_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR (dato->>'acto_version')::bigint<1 OR dato->>'acto_huella_sha256' !~ '^[0-9a-f]{64}$'
   OR dato->>'fuente_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
   OR (dato->>'fuente_version')::bigint<1 OR dato->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
   OR (dato->>'vigente_hasta')::timestamptz <= (dato->>'vigente_desde')::timestamptz
   OR (dato->>'vigente_desde')::timestamptz<cargo.vigente_desde
   OR (dato->>'vigente_hasta')::timestamptz>cargo.vigente_hasta THEN
   RAISE EXCEPTION 'cargo_publicacion_enlace_invalido' USING ERRCODE='22023'; END IF;
  IF dato->>'clase'='titular' THEN
   IF dato->>'titular_enlace_ref' IS NOT NULL OR dato->>'delegante_persona_ref' IS NOT NULL
    OR dato->>'titular_enlace_version' IS NOT NULL OR dato->>'titular_enlace_sha256' IS NOT NULL THEN
    RAISE EXCEPTION 'cargo_publicacion_titular_invalido' USING ERRCODE='22023'; END IF;
   IF EXISTS(SELECT 1 FROM vec_personal.enlace_cargo_competencial_actual a
     JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
     WHERE h.cargo_ref=cargo.cargo_ref AND h.clase='titular' AND h.estado='vigente'
       AND h.enlace_ref<>objeto AND h.vigente_desde<(dato->>'vigente_hasta')::timestamptz
       AND h.vigente_hasta>(dato->>'vigente_desde')::timestamptz) THEN
    RAISE EXCEPTION 'cargo_publicacion_titular_solapado' USING ERRCODE='23505'; END IF;
  ELSE
   SELECT h.* INTO STRICT titular FROM vec_personal.enlace_cargo_competencial_actual a
    JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
    WHERE a.enlace_ref=dato->>'titular_enlace_ref' FOR SHARE OF a,h;
   IF titular.clase<>'titular' OR titular.estado<>'vigente' OR titular.cargo_ref<>cargo.cargo_ref
    OR titular.persona_ref IS DISTINCT FROM dato->>'delegante_persona_ref'
    OR titular.persona_ref IS NOT DISTINCT FROM dato->>'persona_ref'
    OR titular.version::text IS DISTINCT FROM dato->>'titular_enlace_version'
    OR titular.huella_sha256 IS DISTINCT FROM dato->>'titular_enlace_sha256'
    OR (dato->>'vigente_desde')::timestamptz<titular.vigente_desde
    OR (dato->>'vigente_hasta')::timestamptz>titular.vigente_hasta THEN
    RAISE EXCEPTION 'cargo_publicacion_delegacion_invalida' USING ERRCODE='42501'; END IF;
  END IF;
  IF dato->>'ocupacion_ref' IS NOT NULL THEN
   LOCK TABLE vec_personal.relacion_servicio_historia,
    vec_personal.ocupacion_empleado_historia IN SHARE MODE;
   IF NOT EXISTS(SELECT 1 FROM vec_personal.ocupacion_empleado_historia o
      JOIN vec_personal.relacion_servicio_historia r
       ON r.relacion_ref=o.relacion_ref AND r.revision=o.relacion_revision
      WHERE o.ocupacion_ref=dato->>'ocupacion_ref'
       AND o.revision=(dato->>'ocupacion_revision')::integer
       AND o.empleado_ref=dato->>'empleado_ref' AND r.persona_ref=dato->>'persona_ref'
       AND o.organismo_ref=cargo.organizacion_ref AND o.unidad_ref=cargo.unidad_ref
       AND (dato->>'clase'<>'titular' OR cargo.puesto_ref IS NULL OR
        (o.puesto_ref=cargo.puesto_ref AND o.puesto_revision=cargo.puesto_revision))
       AND o.estado='vigente' AND r.estado='vigente'
       AND o.revision=(SELECT max(x.revision) FROM vec_personal.ocupacion_empleado_historia x
         WHERE x.ocupacion_ref=o.ocupacion_ref)
       AND r.revision=(SELECT max(x.revision) FROM vec_personal.relacion_servicio_historia x
         WHERE x.relacion_ref=r.relacion_ref)
       AND o.vigente_desde<=fecha::date AND (o.vigente_hasta IS NULL OR o.vigente_hasta>fecha::date)
       AND r.vigente_desde<=fecha::date AND (r.vigente_hasta IS NULL OR r.vigente_hasta>fecha::date)) THEN
    RAISE EXCEPTION 'cargo_publicacion_ocupacion_no_acreditada' USING ERRCODE='42501'; END IF;
  ELSIF dato->>'empleado_ref' IS NOT NULL OR dato->>'ocupacion_revision' IS NOT NULL THEN
   RAISE EXCEPTION 'cargo_publicacion_ocupacion_parcial' USING ERRCODE='22023';
  END IF;
  INSERT INTO vec_personal.enlace_cargo_competencial_historia(enlace_ref,version,huella_sha256,
   cargo_ref,cargo_version,persona_ref,clase,titular_enlace_ref,titular_enlace_version,
   titular_enlace_sha256,delegante_persona_ref,
   accion_ref,recurso_ref,finalidad_ref,estado,vigente_desde,vigente_hasta,
   acto_ref,acto_version,acto_huella_sha256,fuente_ref,fuente_version,fuente_huella_sha256,
   requiere_enlace_laboral,empleado_ref,ocupacion_ref,ocupacion_revision,
   publicada_en,decision_ref,auditoria_ref,recibo_ref)
  VALUES(objeto,version_nueva,dato_sha,dato->>'cargo_ref',cargo.version,dato->>'persona_ref',
   dato->>'clase',dato->>'titular_enlace_ref',(dato->>'titular_enlace_version')::bigint,
   dato->>'titular_enlace_sha256',dato->>'delegante_persona_ref',
   dato->>'accion_ref',dato->>'recurso_ref',dato->>'finalidad_ref',dato->>'estado',
   (dato->>'vigente_desde')::timestamptz,(dato->>'vigente_hasta')::timestamptz,
   dato->>'acto_ref',(dato->>'acto_version')::bigint,dato->>'acto_huella_sha256',
   dato->>'fuente_ref',(dato->>'fuente_version')::bigint,dato->>'fuente_huella_sha256',
   (dato->>'requiere_enlace_laboral')::boolean,
   dato->>'empleado_ref',dato->>'ocupacion_ref',(dato->>'ocupacion_revision')::integer,
   fecha,v.decision_ref,v.auditoria_ref,recibo_ref);
  INSERT INTO vec_personal.enlace_cargo_competencial_actual(enlace_ref,version,huella_sha256)
   VALUES(objeto,version_nueva,dato_sha)
   ON CONFLICT(enlace_ref) DO UPDATE SET version=EXCLUDED.version,huella_sha256=EXCLUDED.huella_sha256;
 END IF;
 INSERT INTO vec_personal.recibo_publicacion_cargo_competencial(
  clave_idempotencia,recibo_ref,operacion,objeto_ref,version,material_sha256,
  decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(clave,recibo_ref,operacion,objeto,version_nueva,sha,v.decision_ref,v.auditoria_ref,
  v.consumo_huella_sha256,fecha);
 RETURN jsonb_build_object('recibo_ref',recibo_ref,'objeto_ref',objeto,'version',version_nueva,
  'huella_sha256',dato_sha,'registrada_en',fecha,'auditoria_ref',v.auditoria_ref,'estado_replay','nueva');
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'cargo_publicacion_fuente_ausente_o_ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.publicar_cargo_competencial_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.publicar_cargo_competencial_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
