\set ON_ERROR_STOP on
-- Cat4: propuestas inmutables y doble aprobación. Solo la fachada AD3-134
-- puede acreditar actores y decisiones; este esquema no concede permisos V3.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000004',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regclass('vec_catalogos_configurables.propuesta_gobierno') IS NOT NULL THEN
  RAISE EXCEPTION 'Cat4: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;
CREATE TABLE vec_catalogos_configurables.propuesta_gobierno(
 propuesta_ref text PRIMARY KEY CHECK(propuesta_ref ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
 contenido jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(contenido)='object' AND pg_catalog.octet_length(contenido::text)<=33554432),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$' AND huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(contenido::text,'UTF8')),'hex')),
 editor_ref text NOT NULL CHECK(pg_catalog.octet_length(editor_ref) BETWEEN 3 AND 160),
 decision_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 160),
 recibo_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 UNIQUE(propuesta_ref,huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.control_gobierno(
 propuesta_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.propuesta_gobierno,
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 4),
 estado text NOT NULL CHECK(estado IN('propuesta','una_aprobacion','aprobada','confirmada')),
 CHECK((revision=1 AND estado='propuesta') OR(revision=2 AND estado='una_aprobacion') OR(revision=3 AND estado='aprobada') OR(revision=4 AND estado='confirmada'))
);
CREATE TABLE vec_catalogos_configurables.aprobacion_gobierno(
 propuesta_ref text NOT NULL,
 huella_sha256 text NOT NULL,
 ordinal integer NOT NULL CHECK(ordinal BETWEEN 1 AND 2),
 actor_ref text NOT NULL CHECK(pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 160),
 decision_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 160),
 auditoria_ref text NOT NULL CHECK(pg_catalog.octet_length(auditoria_ref) BETWEEN 3 AND 160),
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(propuesta_ref,ordinal), UNIQUE(propuesta_ref,actor_ref),
 FOREIGN KEY(propuesta_ref,huella_sha256) REFERENCES vec_catalogos_configurables.propuesta_gobierno(propuesta_ref,huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.confirmacion_gobierno(
 propuesta_ref text PRIMARY KEY,
 huella_sha256 text NOT NULL,
 actor_ref text NOT NULL CHECK(pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 160),
 decision_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 3 AND 160),
 auditoria_ref text NOT NULL CHECK(pg_catalog.octet_length(auditoria_ref) BETWEEN 3 AND 160),
 recibo_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 resultado jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(resultado)='object'),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(propuesta_ref,huella_sha256) REFERENCES vec_catalogos_configurables.propuesta_gobierno(propuesta_ref,huella_sha256)
);
-- El evento durable es mínimo; no contiene documento, atestaciones ni personas.
CREATE TABLE vec_catalogos_configurables.outbox_gobierno(
 recibo_ref text PRIMARY KEY CHECK(pg_catalog.octet_length(recibo_ref) BETWEEN 3 AND 160),
 propuesta_ref text NOT NULL REFERENCES vec_catalogos_configurables.propuesta_gobierno,
 accion text NOT NULL CHECK(accion IN('proponer','aprobar','confirmar')),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 4),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
DO $proteccion$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['propuesta_gobierno','control_gobierno','aprobacion_gobierno','confirmacion_gobierno','outbox_gobierno'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('CREATE POLICY propietario ON vec_catalogos_configurables.%I TO vec_catalogos_configurables_propietario USING(true) WITH CHECK(true)',tabla);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM PUBLIC',tabla);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM PUBLIC',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_borrar BEFORE DELETE ON vec_catalogos_configurables.%I FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',tabla);
  IF tabla<>'control_gobierno' THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER no_actualizar BEFORE UPDATE ON vec_catalogos_configurables.%I FOR EACH ROW EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',tabla);
  END IF;
 END LOOP;
END $proteccion$;
CREATE FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(
 p_ref text,p_contenido jsonb,p_huella text,p_actor text,p_decision text,p_recibo text,p_motivo text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE previo vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; doc jsonb; item jsonb; modulo_real text;
BEGIN
 IF p_ref IS NULL OR p_ref !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR p_contenido IS NULL OR pg_catalog.jsonb_typeof(p_contenido) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p_contenido::text)>33554432
 OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
 OR p_huella IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_contenido::text,'UTF8')),'hex')
 OR p_actor IS NULL OR pg_catalog.octet_length(p_actor) NOT BETWEEN 3 AND 160
 OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 3 AND 160
 OR p_recibo IS NULL OR pg_catalog.octet_length(p_recibo) NOT BETWEEN 3 AND 160
 OR p_motivo IS NULL OR p_motivo !~ '^[a-z][a-z0-9._-]{0,127}:[1-9][0-9]{0,9}:[a-z][a-z0-9._-]{0,127}$' THEN
  RAISE EXCEPTION 'Cat4: propuesta invalida' USING ERRCODE='22023';
 END IF;
 IF NOT (p_contenido ?& ARRAY['accion','catalogo_id','modulo_id','version','documento_canonico','documento_huella_sha256','preimagenes_control','preimagenes_huella_sha256','categoria_id','revision_esperada','motivo_ref','fuente_ref'])
 OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_contenido))<>12
 OR p_contenido->>'accion' IS NULL OR p_contenido->>'accion' NOT IN('publicar','deshabilitar')
 OR pg_catalog.jsonb_typeof(p_contenido->'catalogo_id') IS DISTINCT FROM 'string' OR p_contenido->>'catalogo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR pg_catalog.jsonb_typeof(p_contenido->'modulo_id') IS DISTINCT FROM 'string' OR p_contenido->>'modulo_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
 OR pg_catalog.jsonb_typeof(p_contenido->'version') IS DISTINCT FROM 'number' OR p_contenido->>'version' !~ '^[1-9][0-9]{0,9}$'
 OR (p_contenido->>'version')::numeric>2147483647
 OR pg_catalog.jsonb_typeof(p_contenido->'fuente_ref') IS DISTINCT FROM 'string' OR pg_catalog.octet_length(p_contenido->>'fuente_ref') NOT BETWEEN 3 AND 160
 OR p_contenido->>'motivo_ref' IS DISTINCT FROM p_motivo
 OR pg_catalog.jsonb_typeof(p_contenido->'preimagenes_control') IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length((p_contenido->'preimagenes_control')::text)>1048576
 OR p_contenido->>'preimagenes_huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((p_contenido->'preimagenes_control')::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'Cat4: contenido incompatible' USING ERRCODE='22023';
 END IF;
 IF p_contenido->>'accion'='publicar' THEN
  IF p_contenido->'categoria_id' IS DISTINCT FROM 'null'::jsonb OR p_contenido->'revision_esperada' IS DISTINCT FROM 'null'::jsonb
  OR pg_catalog.jsonb_typeof(p_contenido->'documento_canonico') IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_contenido->>'documento_canonico') NOT BETWEEN 2 AND 16777216
  OR p_contenido->>'documento_huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_contenido->>'documento_canonico','UTF8')),'hex') THEN
   RAISE EXCEPTION 'Cat4: documento incompatible' USING ERRCODE='22023';
  END IF;
  doc:=(p_contenido->>'documento_canonico')::jsonb;
  IF doc->>'id' IS DISTINCT FROM p_contenido->>'catalogo_id' OR doc->>'modulo_id' IS DISTINCT FROM p_contenido->>'modulo_id'
  OR doc->>'version' IS DISTINCT FROM p_contenido->>'version' OR doc->>'estado' IS DISTINCT FROM 'publicado'
  OR doc->>'fuente_ref' IS DISTINCT FROM p_contenido->>'fuente_ref'
  OR doc->>'creado_por' IS DISTINCT FROM p_actor OR doc->>'publicado_por' IS NULL
  OR doc->>'publicado_por'=p_actor OR pg_catalog.octet_length(doc->>'publicado_por') NOT BETWEEN 3 AND 160
  OR (coalesce(doc->>'ultima_modificacion_por','')<>'' AND doc->>'ultima_modificacion_por' IS DISTINCT FROM p_actor)
  OR pg_catalog.jsonb_typeof(doc->'entradas') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(doc->'entradas') NOT BETWEEN 1 AND 10000 THEN
   RAISE EXCEPTION 'Cat4: separacion o documento incompatible' USING ERRCODE='42501';
  END IF;
 ELSE
  IF p_contenido->'documento_canonico' IS DISTINCT FROM 'null'::jsonb OR p_contenido->'documento_huella_sha256' IS DISTINCT FROM 'null'::jsonb
  OR pg_catalog.jsonb_typeof(p_contenido->'categoria_id') IS DISTINCT FROM 'string' OR p_contenido->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
  OR pg_catalog.jsonb_typeof(p_contenido->'revision_esperada') IS DISTINCT FROM 'number'
  OR p_contenido->>'revision_esperada' !~ '^[1-9][0-9]{0,17}$'
  OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_contenido->'preimagenes_control'))<>1 THEN
   RAISE EXCEPTION 'Cat4: deshabilitacion incompatible' USING ERRCODE='22023';
  END IF;
  item:=p_contenido->'preimagenes_control'->(p_contenido->>'categoria_id');
  IF item IS NULL OR pg_catalog.jsonb_typeof(item) IS DISTINCT FROM 'object'
  OR item IS DISTINCT FROM pg_catalog.jsonb_build_object('version',(p_contenido->>'version')::integer,'huella_sha256',item->>'huella_sha256','revision',(p_contenido->>'revision_esperada')::bigint,'estado','habilitada')
  OR item->>'huella_sha256' IS NULL OR item->>'huella_sha256' !~ '^[0-9a-f]{64}$' THEN
   RAISE EXCEPTION 'Cat4: preimagen incompatible' USING ERRCODE='22023';
  END IF;
  SELECT (documento_canonico::jsonb)->>'modulo_id' INTO modulo_real
   FROM vec_catalogos_configurables.publicacion
   WHERE catalogo_id=p_contenido->>'catalogo_id' AND version=(p_contenido->>'version')::integer
    AND huella_sha256=item->>'huella_sha256';
  IF modulo_real IS DISTINCT FROM p_contenido->>'modulo_id' THEN
   RAISE EXCEPTION 'Cat4: modulo ajeno a publicacion' USING ERRCODE='42501';
  END IF;
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT * INTO previo FROM vec_catalogos_configurables.propuesta_gobierno WHERE propuesta_ref=p_ref;
 IF FOUND THEN
  IF previo.huella_sha256 IS DISTINCT FROM p_huella OR previo.contenido IS DISTINCT FROM p_contenido OR previo.editor_ref IS DISTINCT FROM p_actor OR previo.recibo_ref IS DISTINCT FROM p_recibo THEN
   RAISE EXCEPTION 'Cat4: propuesta en conflicto' USING ERRCODE='23505';
  END IF;
 ELSE
  INSERT INTO vec_catalogos_configurables.propuesta_gobierno(propuesta_ref,contenido,huella_sha256,editor_ref,decision_ref,recibo_ref) VALUES(p_ref,p_contenido,p_huella,p_actor,p_decision,p_recibo);
  INSERT INTO vec_catalogos_configurables.control_gobierno VALUES(p_ref,1,'propuesta');
  INSERT INTO vec_catalogos_configurables.outbox_gobierno(recibo_ref,propuesta_ref,accion,huella_sha256,revision) VALUES(p_recibo,p_ref,'proponer',p_huella,1);
 END IF;
 RETURN pg_catalog.jsonb_build_object('propuesta_ref',p_ref,'huella_sha256',p_huella,'revision',1,'estado','propuesta','recibo_ref',p_recibo);
END $f$;
CREATE FUNCTION vec_catalogos_configurables.aprobar_propuesta_gobierno(
 p_ref text,p_huella text,p_revision bigint,p_actor text,p_decision text,p_auditoria text,p_consumo_huella text,p_recibo text,p_motivo text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE p vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; c vec_catalogos_configurables.control_gobierno%ROWTYPE; a vec_catalogos_configurables.aprobacion_gobierno%ROWTYPE;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT * INTO p FROM vec_catalogos_configurables.propuesta_gobierno WHERE propuesta_ref=p_ref;
 IF NOT FOUND OR p.huella_sha256 IS DISTINCT FROM p_huella OR p.contenido->>'motivo_ref' IS DISTINCT FROM p_motivo OR p_actor IS NULL THEN
  RAISE EXCEPTION 'Cat4: aprobacion incompatible' USING ERRCODE='42501';
 END IF;
 SELECT * INTO c FROM vec_catalogos_configurables.control_gobierno WHERE propuesta_ref=p_ref FOR UPDATE;
 SELECT * INTO a FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND actor_ref=p_actor;
 IF FOUND THEN
  IF a.huella_sha256 IS DISTINCT FROM p_huella OR a.recibo_ref IS DISTINCT FROM p_recibo OR p_revision IS DISTINCT FROM a.ordinal::bigint THEN
   RAISE EXCEPTION 'Cat4: segunda aprobacion de la misma persona' USING ERRCODE='23505';
  END IF;
 ELSE
  IF c.revision IS DISTINCT FROM p_revision OR c.revision NOT IN(1,2) THEN
   RAISE EXCEPTION 'Cat4: CAS de aprobacion fallido' USING ERRCODE='40001';
  END IF;
  INSERT INTO vec_catalogos_configurables.aprobacion_gobierno(propuesta_ref,huella_sha256,ordinal,actor_ref,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref)
   VALUES(p_ref,p_huella,c.revision::integer,p_actor,p_decision,p_auditoria,p_consumo_huella,p_recibo);
  UPDATE vec_catalogos_configurables.control_gobierno SET revision=revision+1,estado=CASE WHEN revision=1 THEN 'una_aprobacion' ELSE 'aprobada' END WHERE propuesta_ref=p_ref AND revision=p_revision;
  IF NOT FOUND THEN RAISE EXCEPTION 'Cat4: CAS de aprobacion fallido' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_catalogos_configurables.outbox_gobierno(recibo_ref,propuesta_ref,accion,huella_sha256,revision) VALUES(p_recibo,p_ref,'aprobar',p_huella,p_revision+1);
 END IF;
 RETURN pg_catalog.jsonb_build_object('propuesta_ref',p_ref,'huella_sha256',p_huella,'revision',p_revision+1,'estado',CASE WHEN p_revision=1 THEN 'una_aprobacion' ELSE 'aprobada' END,'recibo_ref',p_recibo);
END $f$;
CREATE FUNCTION vec_catalogos_configurables.consultar_aprobaciones_gobierno(p_ref text,p_huella text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE resultado jsonb;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT pg_catalog.jsonb_build_object('propuesta',pg_catalog.jsonb_build_object('contenido',p.contenido,'editor_ref',p.editor_ref,'revision',c.revision),
  'aprobaciones',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(a) ORDER BY a.ordinal) FROM vec_catalogos_configurables.aprobacion_gobierno a WHERE a.propuesta_ref=p_ref),'[]'::jsonb))
 INTO resultado FROM vec_catalogos_configurables.propuesta_gobierno p JOIN vec_catalogos_configurables.control_gobierno c USING(propuesta_ref)
 WHERE p.propuesta_ref=p_ref AND p.huella_sha256=p_huella;
 IF resultado IS NULL THEN RAISE EXCEPTION 'Cat4: propuesta ausente' USING ERRCODE='42501'; END IF;
 RETURN resultado;
END $f$;
CREATE FUNCTION vec_catalogos_configurables.confirmar_propuesta_gobierno(
 p_ref text,p_huella text,p_revision bigint,p_actor text,p_decision text,p_auditoria text,p_recibo text,p_motivo text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE p vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; c vec_catalogos_configurables.control_gobierno%ROWTYPE;
 previo vec_catalogos_configurables.confirmacion_gobierno%ROWTYPE; a text; b text; d jsonb; control vec_catalogos_configurables.categoria_control%ROWTYPE; revision_final bigint; resultado jsonb; modulo_real text;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT * INTO p FROM vec_catalogos_configurables.propuesta_gobierno WHERE propuesta_ref=p_ref;
 IF NOT FOUND OR p.huella_sha256 IS DISTINCT FROM p_huella OR p.contenido->>'motivo_ref' IS DISTINCT FROM p_motivo OR p_actor IS NULL OR p_actor=p.editor_ref THEN
  RAISE EXCEPTION 'Cat4: confirmacion incompatible' USING ERRCODE='42501';
 END IF;
 SELECT * INTO previo FROM vec_catalogos_configurables.confirmacion_gobierno WHERE propuesta_ref=p_ref;
 IF FOUND THEN
  IF previo.huella_sha256 IS DISTINCT FROM p_huella OR previo.recibo_ref IS DISTINCT FROM p_recibo OR previo.actor_ref IS DISTINCT FROM p_actor OR p_revision IS DISTINCT FROM 3::bigint THEN
   RAISE EXCEPTION 'Cat4: confirmacion en conflicto' USING ERRCODE='23505';
  END IF;
  RETURN previo.resultado;
 END IF;
 SELECT * INTO c FROM vec_catalogos_configurables.control_gobierno WHERE propuesta_ref=p_ref FOR UPDATE;
 IF c.revision IS DISTINCT FROM p_revision OR c.revision<>3 OR (SELECT count(*) FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND huella_sha256=p_huella)<>2 THEN
  RAISE EXCEPTION 'Cat4: CAS de confirmacion fallido' USING ERRCODE='40001';
 END IF;
 SELECT recibo_ref INTO STRICT a FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND ordinal=1;
 SELECT recibo_ref INTO STRICT b FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND ordinal=2;
 d:=p.contenido;
 -- Serializa publicaciones con el mismo bloqueo de Cat1; deshabilitar y
 -- reservar usan la misma fila de control y conservan todos los usos previos.
 IF d->>'accion'='publicar' THEN
  IF ((d->>'documento_canonico')::jsonb)->>'publicado_por' IS DISTINCT FROM p_actor THEN
   RAISE EXCEPTION 'Cat4: publicador ajeno al documento' USING ERRCODE='42501';
  END IF;
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:'||(d->>'catalogo_id'),0));
  IF (d->>'version')::integer IS DISTINCT FROM (SELECT coalesce(max(version),0)+1 FROM vec_catalogos_configurables.publicacion WHERE catalogo_id=d->>'catalogo_id') THEN
   RAISE EXCEPTION 'Cat4: version de publicacion obsoleta' USING ERRCODE='40001';
  END IF;
  PERFORM vec_catalogos_configurables.publicar(d->>'catalogo_id',(d->>'version')::integer,d->>'documento_huella_sha256',d->>'documento_canonico',d->'preimagenes_control',d->>'preimagenes_huella_sha256',a,b,p_actor,p_decision,p_recibo,p_motivo);
  revision_final:=(d->>'version')::bigint;
 ELSE
  SELECT * INTO control FROM vec_catalogos_configurables.categoria_control WHERE categoria_id=d->>'categoria_id' FOR UPDATE;
  IF NOT FOUND OR control.catalogo_id IS DISTINCT FROM d->>'catalogo_id'
  OR d->'preimagenes_control'->(d->>'categoria_id') IS DISTINCT FROM pg_catalog.jsonb_build_object('version',control.version,'huella_sha256',control.huella_sha256,'revision',control.revision,'estado',control.estado) THEN
   RAISE EXCEPTION 'Cat4: preimagen de categoria obsoleta' USING ERRCODE='40001';
  END IF;
  SELECT (documento_canonico::jsonb)->>'modulo_id' INTO modulo_real
   FROM vec_catalogos_configurables.publicacion
   WHERE catalogo_id=control.catalogo_id AND version=control.version AND huella_sha256=control.huella_sha256;
  IF modulo_real IS DISTINCT FROM d->>'modulo_id' THEN
   RAISE EXCEPTION 'Cat4: modulo ajeno a publicacion' USING ERRCODE='42501';
  END IF;
  revision_final:=vec_catalogos_configurables.cambiar_proyeccion(d->>'categoria_id',(d->>'revision_esperada')::bigint,'deshabilitar',NULL,NULL,p_actor,p_decision,p_recibo,p_motivo);
 END IF;
 resultado:=pg_catalog.jsonb_build_object('propuesta_ref',p_ref,'huella_sha256',p_huella,'revision',4,'estado','confirmada','recibo_ref',p_recibo,'accion',d->>'accion','version',d->'version','revision_categoria',revision_final);
 INSERT INTO vec_catalogos_configurables.confirmacion_gobierno(propuesta_ref,huella_sha256,actor_ref,decision_ref,auditoria_ref,recibo_ref,resultado) VALUES(p_ref,p_huella,p_actor,p_decision,p_auditoria,p_recibo,resultado);
 UPDATE vec_catalogos_configurables.control_gobierno SET revision=4,estado='confirmada' WHERE propuesta_ref=p_ref AND revision=p_revision;
 IF NOT FOUND THEN RAISE EXCEPTION 'Cat4: CAS de confirmacion fallido' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_catalogos_configurables.outbox_gobierno(recibo_ref,propuesta_ref,accion,huella_sha256,revision) VALUES(p_recibo,p_ref,'confirmar',p_huella,4);
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text),
 vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text),
 vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text),
 vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text),
 vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text),
 vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text),
 vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;
DO $acl$
DECLARE f regprocedure;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text)'::regprocedure,
  'vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text)'::regprocedure,
  'vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text)'::regprocedure,
  'vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text)'::regprocedure
 ] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
   RAISE EXCEPTION 'Cat4: ACL incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
