\set ON_ERROR_STOP on
-- Cat4: propuesta A, aprobación B distinta y confirmación por B. Solo la fachada AD3-134
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
 OR pg_catalog.to_regclass('vec_catalogos_configurables.propuesta_gobierno') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class WHERE oid='vec_catalogos_configurables.publicacion'::regclass AND relowner='vec_catalogos_configurables_propietario'::regrole AND relrowsecurity AND relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_catalogos_configurables.publicacion'::regclass AND attname='aprobacion_b_ref' AND attnotnull AND NOT attisdropped)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_catalogos_configurables.publicacion'::regclass AND attname IN('circuito','propuesta_ref') AND NOT attisdropped) THEN
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
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 3),
 estado text NOT NULL CHECK(estado IN('propuesta','aprobada','confirmada')),
 CHECK((revision=1 AND estado='propuesta') OR(revision=2 AND estado='aprobada') OR(revision=3 AND estado='confirmada'))
);
CREATE TABLE vec_catalogos_configurables.aprobacion_gobierno(
 propuesta_ref text NOT NULL,
 huella_sha256 text NOT NULL,
 ordinal integer NOT NULL CHECK(ordinal=1),
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
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 3),
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
LOCK TABLE vec_catalogos_configurables.publicacion IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_catalogos_configurables.publicacion
 ALTER COLUMN aprobacion_b_ref DROP NOT NULL,
 ADD COLUMN circuito text NOT NULL DEFAULT 'doble_aprobacion',
 ADD COLUMN propuesta_ref text REFERENCES vec_catalogos_configurables.propuesta_gobierno;
ALTER TABLE vec_catalogos_configurables.publicacion ADD CONSTRAINT publicacion_circuito_check CHECK(
 (circuito='doble_aprobacion' AND propuesta_ref IS NULL AND aprobacion_b_ref IS NOT NULL)
 OR(circuito='propuesta_aprobacion_rrhh' AND propuesta_ref IS NOT NULL AND aprobacion_b_ref IS NULL));
-- Evolución explícita desde Cat1 de main@105c537e2: no se reaplica Cat1.
-- Huellas candidatas medidas sobre los cuerpos literales; instalación NO-GO
-- hasta cotejar prosrc y pg_get_functiondef en PostgreSQL 18 desechable.
DO $contrato_publicar$
DECLARE
    f oid := 'vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)'::regprocedure;
    original text; nuevo text; meta jsonb; deps jsonb;
    vieja_0 text := $marca_0$    preimagenes_restantes jsonb;$marca_0$;
    nueva_0 text := $cambio_0$    preimagenes_restantes jsonb;
    gobierno_ref text;
    estado_publicado text;$cambio_0$;
    vieja_1 text := $marca_1$       OR p_aprobacion_b IS NULL OR pg_catalog.octet_length(p_aprobacion_b) NOT BETWEEN 3 AND 160$marca_1$;
    nueva_1 text := $cambio_1$       OR (p_aprobacion_b IS NOT NULL AND pg_catalog.octet_length(p_aprobacion_b) NOT BETWEEN 3 AND 160)$cambio_1$;
    vieja_2 text := $marca_2$    preimagenes_restantes := p_preimagenes;$marca_2$;
    nueva_2 text := $cambio_2$    -- Cat4: ausencia de segunda aprobación sólo con propuesta y aprobación reales.
    IF p_aprobacion_b IS NULL THEN
        SELECT g.propuesta_ref INTO gobierno_ref
          FROM vec_catalogos_configurables.propuesta_gobierno g
          JOIN vec_catalogos_configurables.control_gobierno cg USING(propuesta_ref)
          JOIN vec_catalogos_configurables.aprobacion_gobierno ag USING(propuesta_ref)
         WHERE g.contenido->>'catalogo_id'=p_catalogo_id
           AND g.contenido->>'version'=p_version::text
           AND g.contenido->>'documento_canonico'=p_documento
           AND g.contenido->>'documento_huella_sha256'=p_huella
           AND g.contenido->'preimagenes_control'=p_preimagenes
           AND g.contenido->>'preimagenes_huella_sha256'=p_preimagenes_huella
           AND g.contenido->>'motivo_ref'=p_motivo_ref
           AND g.editor_ref<>p_actor AND cg.revision=2 AND cg.estado='aprobada'
           AND ag.ordinal=1 AND ag.actor_ref=p_actor AND ag.recibo_ref=p_aprobacion_a
           AND ag.huella_sha256=g.huella_sha256
           AND (p_documento::jsonb)->>'publicado_por'=p_actor
           AND (p_documento::jsonb)->>'aprobacion_ref'=ag.recibo_ref;
        IF gobierno_ref IS NULL THEN
            RAISE EXCEPTION 'Cat4: publicacion sin propuesta y aprobacion reales' USING ERRCODE='42501';
        END IF;
    END IF;
    preimagenes_restantes := p_preimagenes;$cambio_2$;
    vieja_3 text := $marca_3$    SELECT * INTO anterior FROM vec_catalogos_configurables.publicacion
     WHERE catalogo_id = p_catalogo_id AND version = p_version;$marca_3$;
    nueva_3 text := $cambio_3$    IF gobierno_ref IS NOT NULL THEN
        -- Toda categoría anterior permanece en la nueva versión completa.
        PERFORM 1 FROM vec_catalogos_configurables.categoria_control
         WHERE catalogo_id=p_catalogo_id ORDER BY categoria_id FOR UPDATE;
        IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.categoria_control cc
          WHERE cc.catalogo_id=p_catalogo_id AND NOT EXISTS(
            SELECT 1 FROM pg_catalog.jsonb_array_elements(contenido->'entradas') e WHERE e->>'clave'=cc.categoria_id)) THEN
            RAISE EXCEPTION 'Cat4: version incompleta' USING ERRCODE='22023';
        END IF;
    END IF;
    SELECT * INTO anterior FROM vec_catalogos_configurables.publicacion
     WHERE catalogo_id = p_catalogo_id AND version = p_version;$cambio_3$;
    vieja_4 text := $marca_4$         aprobacion_b_ref, actor_ref, decision_ref, recibo_ref)$marca_4$;
    nueva_4 text := $cambio_4$         aprobacion_b_ref, actor_ref, decision_ref, recibo_ref, circuito, propuesta_ref)$cambio_4$;
    vieja_5 text := $marca_5$            p_aprobacion_b, p_actor, p_decision, p_recibo);$marca_5$;
    nueva_5 text := $cambio_5$            p_aprobacion_b, p_actor, p_decision, p_recibo,
            CASE WHEN gobierno_ref IS NULL THEN 'doble_aprobacion' ELSE 'propuesta_aprobacion_rrhh' END,gobierno_ref);$cambio_5$;
    vieja_6 text := $marca_6$        INSERT INTO vec_catalogos_configurables.entrada_publicada$marca_6$;
    nueva_6 text := $cambio_6$        estado_publicado:=CASE WHEN gobierno_ref IS NULL THEN coalesce(control_anterior.estado,'habilitada') ELSE item#>>'{atributos,estado}' END;
        IF gobierno_ref IS NOT NULL AND (estado_publicado IS NULL OR estado_publicado NOT IN('habilitada','deshabilitada')
           OR (control_anterior.estado='deshabilitada' AND estado_publicado<>'deshabilitada')
           OR (estado_publicado IS DISTINCT FROM coalesce(control_anterior.estado,'habilitada')
               AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.propuesta_gobierno g
                 WHERE g.propuesta_ref=gobierno_ref AND g.contenido->>'accion'='deshabilitar'
                   AND g.contenido->>'categoria_id'=clave AND control_anterior.estado='habilitada'
                   AND estado_publicado='deshabilitada'))) THEN
            RAISE EXCEPTION 'Cat4: transicion de entrada incompatible' USING ERRCODE='42501';
        END IF;
        INSERT INTO vec_catalogos_configurables.entrada_publicada$cambio_6$;
    vieja_7 text := $marca_7$            VALUES (clave, p_catalogo_id, p_version, p_huella, 1, 'habilitada');$marca_7$;
    nueva_7 text := $cambio_7$            VALUES (clave, p_catalogo_id, p_version, p_huella, 1, estado_publicado);$cambio_7$;
    vieja_8 text := $marca_8$               SET version = p_version, huella_sha256 = p_huella, revision = revision + 1,$marca_8$;
    nueva_8 text := $cambio_8$               SET version = p_version, huella_sha256 = p_huella, revision = revision + 1, estado=estado_publicado,$cambio_8$;
    vieja_9 text := $marca_9$        SELECT clave, 'publicar', revision, p_recibo || ':' || clave, p_decision, p_actor,$marca_9$;
    nueva_9 text := $cambio_9$        SELECT clave, CASE WHEN gobierno_ref IS NOT NULL AND control_anterior.estado='habilitada' AND estado_publicado='deshabilitada' THEN 'deshabilitar' ELSE 'publicar' END, revision, p_recibo || ':' || clave, p_decision, p_actor,$cambio_9$;
    vieja_10 text := $marca_10$    IF (p_version > 1 AND NOT EXISTS (
            SELECT 1 FROM vec_catalogos_configurables.publicacion
             WHERE catalogo_id = p_catalogo_id AND version = p_version - 1)) THEN
        RAISE EXCEPTION 'version de publicacion en conflicto' USING ERRCODE = '23505';
    END IF;$marca_10$;
    nueva_10 text := $cambio_10$    IF p_version > 1 THEN
        SELECT * INTO anterior FROM vec_catalogos_configurables.publicacion
         WHERE catalogo_id = p_catalogo_id AND version = p_version - 1;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'version de publicacion en conflicto' USING ERRCODE = '23505';
        END IF;
        -- Una versión nueva conserva el módulo propietario; la ausencia
        -- histórica del campo tampoco autoriza asignarlo después.
        IF (anterior.documento_canonico::jsonb->>'modulo_id')
            IS DISTINCT FROM (contenido->>'modulo_id') THEN
            RAISE EXCEPTION 'modulo de catalogo incompatible' USING ERRCODE = '42501';
        END IF;
    END IF;$cambio_10$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_catalog.pg_proc p WHERE p.oid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=5s','statement_timeout=30s'] AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')='f04abe6afb52343e58e51ce43d03d6ad2df8eb073edfb1810c40c411e01cb564') THEN
        RAISE EXCEPTION 'Cat4: contrato publicar preimagen incompatible' USING ERRCODE='55000';
    END IF;
    nuevo:=original;
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_0,''))<>pg_catalog.length(vieja_0) THEN RAISE EXCEPTION 'Cat4: marca 0 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_0,nueva_0);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_1,''))<>pg_catalog.length(vieja_1) THEN RAISE EXCEPTION 'Cat4: marca 1 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_1,nueva_1);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_2,''))<>pg_catalog.length(vieja_2) THEN RAISE EXCEPTION 'Cat4: marca 2 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_2,nueva_2);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_3,''))<>pg_catalog.length(vieja_3) THEN RAISE EXCEPTION 'Cat4: marca 3 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_3,nueva_3);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_4,''))<>pg_catalog.length(vieja_4) THEN RAISE EXCEPTION 'Cat4: marca 4 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_4,nueva_4);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_5,''))<>pg_catalog.length(vieja_5) THEN RAISE EXCEPTION 'Cat4: marca 5 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_5,nueva_5);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_6,''))<>pg_catalog.length(vieja_6) THEN RAISE EXCEPTION 'Cat4: marca 6 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_6,nueva_6);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_7,''))<>pg_catalog.length(vieja_7) THEN RAISE EXCEPTION 'Cat4: marca 7 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_7,nueva_7);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_8,''))<>pg_catalog.length(vieja_8) THEN RAISE EXCEPTION 'Cat4: marca 8 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_8,nueva_8);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_9,''))<>pg_catalog.length(vieja_9) THEN RAISE EXCEPTION 'Cat4: marca 9 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_9,nueva_9);
    IF pg_catalog.length(nuevo)-pg_catalog.length(pg_catalog.replace(nuevo,vieja_10,''))<>pg_catalog.length(vieja_10) THEN RAISE EXCEPTION 'Cat4: marca 10 incompatible' USING ERRCODE='55000'; END IF;
    nuevo:=pg_catalog.replace(nuevo,vieja_10,nueva_10);
    EXECUTE nuevo;
    IF (SELECT pg_catalog.pg_get_functiondef(f)) IS DISTINCT FROM nuevo
    OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=f)<>'f6aaf535445d27b5c1c6f8e64622c1246d687c72414c0d14b2ac7c41a6b054c1'
    OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'Cat4: contrato publicar postimagen incompatible' USING ERRCODE='55000';
    END IF;
END $contrato_publicar$;
CREATE FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(
 p_ref text,p_contenido jsonb,p_huella text,p_actor text,p_decision text,p_recibo text,p_motivo text
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE previo vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; doc jsonb; item jsonb;
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
 IF pg_catalog.jsonb_typeof(p_contenido->'documento_canonico') IS DISTINCT FROM 'string'
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
 OR pg_catalog.octet_length(doc->>'aprobacion_ref') NOT BETWEEN 3 AND 160 OR doc->>'aprobacion_ref' IS NULL
 OR (coalesce(doc->>'ultima_modificacion_por','')<>'' AND doc->>'ultima_modificacion_por' IS DISTINCT FROM p_actor)
 OR pg_catalog.jsonb_typeof(doc->'entradas') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(doc->'entradas') NOT BETWEEN 1 AND 10000 THEN
  RAISE EXCEPTION 'Cat4: separacion o documento incompatible' USING ERRCODE='42501';
 END IF;
 FOR item IN SELECT value FROM pg_catalog.jsonb_array_elements(doc->'entradas') LOOP
  IF pg_catalog.jsonb_typeof(item#>'{atributos,estado}') IS DISTINCT FROM 'string'
  OR item#>>'{atributos,estado}' NOT IN('habilitada','deshabilitada') THEN
   RAISE EXCEPTION 'Cat4: estado de entrada incompatible' USING ERRCODE='22023';
  END IF;
 END LOOP;
 IF p_contenido->>'accion'='publicar' THEN
  IF p_contenido->'categoria_id' IS DISTINCT FROM 'null'::jsonb OR p_contenido->'revision_esperada' IS DISTINCT FROM 'null'::jsonb THEN
   RAISE EXCEPTION 'Cat4: publicacion incompatible' USING ERRCODE='22023';
  END IF;
 ELSE
  IF pg_catalog.jsonb_typeof(p_contenido->'categoria_id') IS DISTINCT FROM 'string' OR p_contenido->>'categoria_id' !~ '^[a-z][a-z0-9_.:-]{2,127}$'
  OR pg_catalog.jsonb_typeof(p_contenido->'revision_esperada') IS DISTINCT FROM 'number'
  OR p_contenido->>'revision_esperada' !~ '^[1-9][0-9]{0,17}$'
  OR (p_contenido->>'version')::integer<2 THEN
   RAISE EXCEPTION 'Cat4: deshabilitacion incompatible' USING ERRCODE='22023';
  END IF;
  item:=p_contenido->'preimagenes_control'->(p_contenido->>'categoria_id');
  IF item IS NULL OR item IS DISTINCT FROM pg_catalog.jsonb_build_object('version',(p_contenido->>'version')::integer-1,'huella_sha256',item->>'huella_sha256','revision',(p_contenido->>'revision_esperada')::bigint,'estado','habilitada')
  OR item->>'huella_sha256' IS NULL OR item->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(doc->'entradas') e WHERE e->>'clave'=p_contenido->>'categoria_id' AND e#>>'{atributos,estado}'='deshabilitada')<>1 THEN
   RAISE EXCEPTION 'Cat4: preimagen incompatible' USING ERRCODE='22023';
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
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE p vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; c vec_catalogos_configurables.control_gobierno%ROWTYPE; a vec_catalogos_configurables.aprobacion_gobierno%ROWTYPE;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT * INTO p FROM vec_catalogos_configurables.propuesta_gobierno WHERE propuesta_ref=p_ref;
 IF NOT FOUND OR p.huella_sha256 IS DISTINCT FROM p_huella OR p.contenido->>'motivo_ref' IS DISTINCT FROM p_motivo OR p_actor IS NULL OR p_actor=p.editor_ref
 OR (p.contenido->>'documento_canonico')::jsonb->>'publicado_por' IS DISTINCT FROM p_actor
 OR (p.contenido->>'documento_canonico')::jsonb->>'aprobacion_ref' IS DISTINCT FROM p_recibo THEN
  RAISE EXCEPTION 'Cat4: aprobacion incompatible' USING ERRCODE='42501';
 END IF;
 SELECT * INTO c FROM vec_catalogos_configurables.control_gobierno WHERE propuesta_ref=p_ref FOR UPDATE;
 SELECT * INTO a FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND actor_ref=p_actor;
 IF FOUND THEN
  IF a.huella_sha256 IS DISTINCT FROM p_huella OR a.recibo_ref IS DISTINCT FROM p_recibo OR p_revision IS DISTINCT FROM a.ordinal::bigint THEN
   RAISE EXCEPTION 'Cat4: segunda aprobacion de la misma persona' USING ERRCODE='23505';
  END IF;
 ELSE
  IF c.revision IS DISTINCT FROM p_revision OR c.revision<>1 THEN
   RAISE EXCEPTION 'Cat4: CAS de aprobacion fallido' USING ERRCODE='40001';
  END IF;
  INSERT INTO vec_catalogos_configurables.aprobacion_gobierno(propuesta_ref,huella_sha256,ordinal,actor_ref,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref)
   VALUES(p_ref,p_huella,c.revision::integer,p_actor,p_decision,p_auditoria,p_consumo_huella,p_recibo);
  UPDATE vec_catalogos_configurables.control_gobierno SET revision=revision+1,estado='aprobada' WHERE propuesta_ref=p_ref AND revision=p_revision;
  IF NOT FOUND THEN RAISE EXCEPTION 'Cat4: CAS de aprobacion fallido' USING ERRCODE='40001'; END IF;
  INSERT INTO vec_catalogos_configurables.outbox_gobierno(recibo_ref,propuesta_ref,accion,huella_sha256,revision) VALUES(p_recibo,p_ref,'aprobar',p_huella,p_revision+1);
 END IF;
 RETURN pg_catalog.jsonb_build_object('propuesta_ref',p_ref,'huella_sha256',p_huella,'revision',p_revision+1,'estado','aprobada','recibo_ref',p_recibo);
END $f$;
CREATE FUNCTION vec_catalogos_configurables.consultar_aprobaciones_gobierno(p_ref text,p_huella text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
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
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE p vec_catalogos_configurables.propuesta_gobierno%ROWTYPE; c vec_catalogos_configurables.control_gobierno%ROWTYPE;
 previo vec_catalogos_configurables.confirmacion_gobierno%ROWTYPE; a text; d jsonb; revision_final bigint; resultado jsonb; modulo_real text;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:gobierno:'||p_ref,0));
 SELECT * INTO p FROM vec_catalogos_configurables.propuesta_gobierno WHERE propuesta_ref=p_ref;
 IF NOT FOUND OR p.huella_sha256 IS DISTINCT FROM p_huella OR p.contenido->>'motivo_ref' IS DISTINCT FROM p_motivo OR p_actor IS NULL OR p_actor=p.editor_ref THEN
  RAISE EXCEPTION 'Cat4: confirmacion incompatible' USING ERRCODE='42501';
 END IF;
 SELECT * INTO previo FROM vec_catalogos_configurables.confirmacion_gobierno WHERE propuesta_ref=p_ref;
 IF FOUND THEN
  IF previo.huella_sha256 IS DISTINCT FROM p_huella OR previo.recibo_ref IS DISTINCT FROM p_recibo OR previo.actor_ref IS DISTINCT FROM p_actor OR p_revision IS DISTINCT FROM 2::bigint THEN
   RAISE EXCEPTION 'Cat4: confirmacion en conflicto' USING ERRCODE='23505';
  END IF;
  RETURN previo.resultado;
 END IF;
 SELECT * INTO c FROM vec_catalogos_configurables.control_gobierno WHERE propuesta_ref=p_ref FOR UPDATE;
 IF c.revision IS DISTINCT FROM p_revision OR c.revision<>2 OR (SELECT count(*) FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND huella_sha256=p_huella)<>1 THEN
  RAISE EXCEPTION 'Cat4: CAS de confirmacion fallido' USING ERRCODE='40001';
 END IF;
 SELECT recibo_ref INTO STRICT a FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND ordinal=1;
 IF NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.aprobacion_gobierno WHERE propuesta_ref=p_ref AND ordinal=1 AND actor_ref=p_actor) THEN
  RAISE EXCEPTION 'Cat4: confirmador ajeno a aprobacion' USING ERRCODE='42501';
 END IF;
 d:=p.contenido;
 -- Publicación completa también al deshabilitar. Cat1 conserva su historia.
 IF ((d->>'documento_canonico')::jsonb)->>'publicado_por' IS DISTINCT FROM p_actor THEN
  RAISE EXCEPTION 'Cat4: publicador ajeno al documento' USING ERRCODE='42501';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:'||(d->>'catalogo_id'),0));
 IF (d->>'version')::integer IS DISTINCT FROM (SELECT coalesce(max(version),0)+1 FROM vec_catalogos_configurables.publicacion WHERE catalogo_id=d->>'catalogo_id') THEN
  RAISE EXCEPTION 'Cat4: version de publicacion obsoleta' USING ERRCODE='40001';
 END IF;
 IF (d->>'version')::integer>1 THEN
  SELECT (documento_canonico::jsonb)->>'modulo_id' INTO modulo_real
   FROM vec_catalogos_configurables.publicacion
   WHERE catalogo_id=d->>'catalogo_id' AND version=(d->>'version')::integer-1;
  IF NOT FOUND OR modulo_real IS DISTINCT FROM d->>'modulo_id' THEN
   RAISE EXCEPTION 'Cat4: modulo ajeno a publicacion anterior' USING ERRCODE='42501';
  END IF;
 END IF;
 PERFORM vec_catalogos_configurables.publicar(d->>'catalogo_id',(d->>'version')::integer,d->>'documento_huella_sha256',d->>'documento_canonico',d->'preimagenes_control',d->>'preimagenes_huella_sha256',a,NULL,p_actor,p_decision,p_recibo,p_motivo);
 IF d->>'accion'='deshabilitar' THEN
  SELECT revision INTO STRICT revision_final FROM vec_catalogos_configurables.categoria_control WHERE categoria_id=d->>'categoria_id';
 ELSE
  revision_final:=(d->>'version')::bigint;
 END IF;
 resultado:=pg_catalog.jsonb_build_object('propuesta_ref',p_ref,'huella_sha256',p_huella,'revision',3,'estado','confirmada','recibo_ref',p_recibo,'accion',d->>'accion','version',d->'version','revision_categoria',revision_final);
 INSERT INTO vec_catalogos_configurables.confirmacion_gobierno(propuesta_ref,huella_sha256,actor_ref,decision_ref,auditoria_ref,recibo_ref,resultado) VALUES(p_ref,p_huella,p_actor,p_decision,p_auditoria,p_recibo,resultado);
 UPDATE vec_catalogos_configurables.control_gobierno SET revision=3,estado='confirmada' WHERE propuesta_ref=p_ref AND revision=p_revision;
 IF NOT FOUND THEN RAISE EXCEPTION 'Cat4: CAS de confirmacion fallido' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_catalogos_configurables.outbox_gobierno(recibo_ref,propuesta_ref,accion,huella_sha256,revision) VALUES(p_recibo,p_ref,'confirmar',p_huella,3);
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
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
  OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantor<>p.proowner OR a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
   RAISE EXCEPTION 'Cat4: ACL incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
