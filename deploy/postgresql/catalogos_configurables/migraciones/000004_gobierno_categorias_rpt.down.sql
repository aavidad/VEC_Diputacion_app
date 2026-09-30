\set ON_ERROR_STOP on
-- Solo reversión de ensayo sin hechos. Nunca ejecutar sobre historia instalada.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000004',0));
LOCK TABLE vec_catalogos_configurables.propuesta_gobierno,
 vec_catalogos_configurables.control_gobierno,
 vec_catalogos_configurables.aprobacion_gobierno,
 vec_catalogos_configurables.confirmacion_gobierno,
 vec_catalogos_configurables.outbox_gobierno IN ACCESS EXCLUSIVE MODE;
DO $guardia$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.propuesta_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.control_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.aprobacion_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.confirmacion_gobierno)
 OR EXISTS(SELECT 1 FROM vec_catalogos_configurables.outbox_gobierno) THEN
  RAISE EXCEPTION 'Cat4: historia conservada; reversion prohibida' USING ERRCODE='55000';
 END IF;
END $guardia$;
LOCK TABLE vec_catalogos_configurables.publicacion IN ACCESS EXCLUSIVE MODE;
DO $publicaciones$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.publicacion WHERE circuito<>'doble_aprobacion' OR propuesta_ref IS NOT NULL OR aprobacion_b_ref IS NULL) THEN
  RAISE EXCEPTION 'Cat4: publicacion gobernada conservada' USING ERRCODE='55000';
 END IF;
END $publicaciones$;
-- Evolución explícita de Cat1 instalada: nunca se edita ni reaplica su archivo.
DO $contrato_publicar$
DECLARE
    f oid := 'vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)'::regprocedure;
    original text; nuevo text; meta jsonb; deps jsonb;
    vieja_0 text := $marca_0$    preimagenes_restantes jsonb;
    gobierno_ref text;
    estado_publicado text;$marca_0$;
    nueva_0 text := $cambio_0$    preimagenes_restantes jsonb;$cambio_0$;
    vieja_1 text := $marca_1$       OR (p_aprobacion_b IS NOT NULL AND pg_catalog.octet_length(p_aprobacion_b) NOT BETWEEN 3 AND 160)$marca_1$;
    nueva_1 text := $cambio_1$       OR p_aprobacion_b IS NULL OR pg_catalog.octet_length(p_aprobacion_b) NOT BETWEEN 3 AND 160$cambio_1$;
    vieja_2 text := $marca_2$    -- Cat4: ausencia de segunda aprobación sólo con propuesta y aprobación reales.
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
           AND contenido->>'publicado_por'=p_actor
           AND contenido->>'aprobacion_ref'=ag.recibo_ref;
        IF gobierno_ref IS NULL THEN
            RAISE EXCEPTION 'Cat4: publicacion sin propuesta y aprobacion reales' USING ERRCODE='42501';
        END IF;
    END IF;
    preimagenes_restantes := p_preimagenes;$marca_2$;
    nueva_2 text := $cambio_2$    preimagenes_restantes := p_preimagenes;$cambio_2$;
    vieja_3 text := $marca_3$    IF gobierno_ref IS NOT NULL THEN
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
     WHERE catalogo_id = p_catalogo_id AND version = p_version;$marca_3$;
    nueva_3 text := $cambio_3$    SELECT * INTO anterior FROM vec_catalogos_configurables.publicacion
     WHERE catalogo_id = p_catalogo_id AND version = p_version;$cambio_3$;
    vieja_4 text := $marca_4$         aprobacion_b_ref, actor_ref, decision_ref, recibo_ref, circuito, propuesta_ref)$marca_4$;
    nueva_4 text := $cambio_4$         aprobacion_b_ref, actor_ref, decision_ref, recibo_ref)$cambio_4$;
    vieja_5 text := $marca_5$            p_aprobacion_b, p_actor, p_decision, p_recibo,
            CASE WHEN gobierno_ref IS NULL THEN 'doble_aprobacion' ELSE 'propuesta_aprobacion_rrhh' END,gobierno_ref);$marca_5$;
    nueva_5 text := $cambio_5$            p_aprobacion_b, p_actor, p_decision, p_recibo);$cambio_5$;
    vieja_6 text := $marca_6$        estado_publicado:=CASE WHEN gobierno_ref IS NULL THEN coalesce(control_anterior.estado,'habilitada') ELSE item#>>'{atributos,estado}' END;
        IF gobierno_ref IS NOT NULL AND (estado_publicado IS NULL OR estado_publicado NOT IN('habilitada','deshabilitada')
           OR (control_anterior.estado='deshabilitada' AND estado_publicado<>'deshabilitada')
           OR (estado_publicado IS DISTINCT FROM coalesce(control_anterior.estado,'habilitada')
               AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.propuesta_gobierno g
                 WHERE g.propuesta_ref=gobierno_ref AND g.contenido->>'accion'='deshabilitar'
                   AND g.contenido->>'categoria_id'=clave AND control_anterior.estado='habilitada'
                   AND estado_publicado='deshabilitada'))) THEN
            RAISE EXCEPTION 'Cat4: transicion de entrada incompatible' USING ERRCODE='42501';
        END IF;
        INSERT INTO vec_catalogos_configurables.entrada_publicada$marca_6$;
    nueva_6 text := $cambio_6$        INSERT INTO vec_catalogos_configurables.entrada_publicada$cambio_6$;
    vieja_7 text := $marca_7$            VALUES (clave, p_catalogo_id, p_version, p_huella, 1, estado_publicado);$marca_7$;
    nueva_7 text := $cambio_7$            VALUES (clave, p_catalogo_id, p_version, p_huella, 1, 'habilitada');$cambio_7$;
    vieja_8 text := $marca_8$               SET version = p_version, huella_sha256 = p_huella, revision = revision + 1, estado=estado_publicado,$marca_8$;
    nueva_8 text := $cambio_8$               SET version = p_version, huella_sha256 = p_huella, revision = revision + 1,$cambio_8$;
    vieja_9 text := $marca_9$        SELECT clave, CASE WHEN gobierno_ref IS NOT NULL AND control_anterior.estado='habilitada' AND estado_publicado='deshabilitada' THEN 'deshabilitar' ELSE 'publicar' END, revision, p_recibo || ':' || clave, p_decision, p_actor,$marca_9$;
    nueva_9 text := $cambio_9$        SELECT clave, 'publicar', revision, p_recibo || ':' || clave, p_decision, p_actor,$cambio_9$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_catalog.pg_proc p WHERE p.oid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND proowner='vec_catalogos_configurables_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=5s','statement_timeout=30s'] AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')='3678f3d95376bc2a10e91d0099da92ec2cb0c37d6c0cf35a2c81708370b9756a') THEN
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
    EXECUTE nuevo;
    IF (SELECT pg_catalog.pg_get_functiondef(f)) IS DISTINCT FROM nuevo
    OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=f)<>'d44a94df48182b45d97095c47cb9ee7505d2a267399d541150fc22176606f8dd'
    OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'Cat4: contrato publicar postimagen incompatible' USING ERRCODE='55000';
    END IF;
END $contrato_publicar$;
ALTER TABLE vec_catalogos_configurables.publicacion
 DROP CONSTRAINT publicacion_circuito_check,
 DROP COLUMN circuito,
 DROP COLUMN propuesta_ref,
 ALTER COLUMN aprobacion_b_ref SET NOT NULL;
DROP FUNCTION vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.consultar_aprobaciones_gobierno(text,text);
DROP FUNCTION vec_catalogos_configurables.aprobar_propuesta_gobierno(text,text,bigint,text,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.registrar_propuesta_gobierno(text,jsonb,text,text,text,text,text);
DROP TABLE vec_catalogos_configurables.outbox_gobierno;
DROP TABLE vec_catalogos_configurables.confirmacion_gobierno;
DROP TABLE vec_catalogos_configurables.aprobacion_gobierno;
DROP TABLE vec_catalogos_configurables.control_gobierno;
DROP TABLE vec_catalogos_configurables.propuesta_gobierno;
COMMIT;
