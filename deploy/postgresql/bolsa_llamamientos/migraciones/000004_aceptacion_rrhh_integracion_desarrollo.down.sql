\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000004',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo,
    vec_bolsa_llamamientos.llamamiento_integracion_desarrollo,
    vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
    vec_bolsa_llamamientos.outbox_integracion_desarrollo IN ACCESS EXCLUSIVE MODE;

-- Las historias y los eventos están ligados por FK a su operación inmutable.
-- No se borra ni transforma un terminal para permitir la reversión.
DO $retirar$
DECLARE
    v_def text; v_acl aclitem[]; v_cambio record; v_bloque text;
    v_inicio integer; v_fin integer;
    v_desde text := $marca$ IF r->>'tipo'='aceptacion_rrhh' THEN$marca$;
    v_hasta text := $marca$ ELSIF r->>'tipo'='orden' THEN$marca$;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo
        WHERE tipo='aceptacion_rrhh' OR apertura_operacion_ref IS NOT NULL) THEN
        RAISE EXCEPTION 'reversión denegada: historia terminal de aceptación RRHH' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
    v_inicio:=strpos(v_def,v_desde);
    v_fin:=strpos(v_def,v_hasta);
    IF v_inicio=0 OR v_fin<=v_inicio OR
       length(v_def)-length(replace(v_def,v_desde,''))<>length(v_desde) OR
       length(v_def)-length(replace(v_def,v_hasta,''))<>length(v_hasta) THEN
        RAISE EXCEPTION 'rama terminal Bolsa incompatible para reversión' USING ERRCODE='55000';
    END IF;
    v_bloque:=substring(v_def FROM v_inicio FOR v_fin-v_inicio+length(v_hasta));
    IF encode(sha256(convert_to(v_bloque,'UTF8')),'hex')<>
       '8812bb9833cc8ad96405cd28160095456ca94eb15300800651ce6565d93515d9' THEN
        RAISE EXCEPTION 'rama terminal Bolsa modificada: conservarla' USING ERRCODE='55000';
    END IF;
    v_def:=replace(v_def,v_bloque,$original$ IF r->>'tipo'='orden' THEN$original$);
    FOR v_cambio IN SELECT * FROM (VALUES
      ('variables',
       $antes$ v_orden record; v_anterior record; v_consumo record; v_existente record;$antes$,
       $despues$ v_orden record; v_anterior record; v_consumo record; v_existente record;
 v_resolucion jsonb; v_apertura record; v_apertura_json jsonb; v_llamamiento record;$despues$),
      ('tipo',
       $antes$r->>'tipo' IN ('orden','propuesta') AND$antes$,
       $despues$r->>'tipo' IN ('orden','propuesta','aceptacion_rrhh') AND$despues$),
      ('material_resolucion',
       $antes$ i:=r->'instantanea'; f:=r->'fuente'; p:=r->'propuesta'; l:=r->'llamamiento';$antes$,
       $despues$ i:=r->'instantanea'; f:=r->'fuente'; p:=r->'propuesta'; l:=r->'llamamiento';
 v_resolucion:=r->'resolucion';
 IF r->>'tipo' IS DISTINCT FROM 'aceptacion_rrhh' AND r?'resolucion' THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='resolución fuera de aceptación RRHH';
 END IF;$despues$),
      ('accion',
       $antes$ v_accion:=CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.preparar' ELSE 'bolsa.llamamiento.abrir' END;$antes$,
       $despues$ v_accion:=CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.preparar'
  WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrar' ELSE 'bolsa.llamamiento.abrir' END;$despues$),
      ('insercion_columnas',
       $antes$ INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo VALUES($antes$,
       $despues$ INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo (
  operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,registro_canonico,
  registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en,apertura_operacion_ref
 ) VALUES($despues$),
      ('insercion_apertura',
       $antes$  nullif(r->>'orden_operacion_ref',''),p_registro,v_hash,v_contexto_hash,v_consumo.decision_ref,v_recibo,v_ahora);$antes$,
       $despues$  nullif(r->>'orden_operacion_ref',''),p_registro,v_hash,v_contexto_hash,v_consumo.decision_ref,v_recibo,v_ahora,
  v_resolucion->>'apertura_operacion_ref');$despues$),
      ('evento',
       $antes$  'tipo',CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.confirmada' ELSE 'bolsa.llamamiento.abierto' END,$antes$,
       $despues$  'tipo',CASE r->>'tipo' WHEN 'orden' THEN 'bolsa.orden.confirmada'
   WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrada' ELSE 'bolsa.llamamiento.abierto' END,$despues$)
    ) AS cambios(nombre,anterior,nuevo)
    LOOP
        IF length(v_def)-length(replace(v_def,v_cambio.nuevo,''))<>length(v_cambio.nuevo) THEN
            RAISE EXCEPTION 'función Bolsa incompatible para reversión: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.nuevo,v_cambio.anterior);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'reversión alteró permisos de integración Bolsa' USING ERRCODE='55000';
    END IF;
END
$retirar$;
ALTER TABLE vec_bolsa_llamamientos.integracion_desarrollo
    DROP CONSTRAINT integracion_desarrollo_apertura_check,
    DROP CONSTRAINT integracion_desarrollo_terminal_apertura_unico,
    DROP CONSTRAINT integracion_desarrollo_apertura_fk,
    DROP CONSTRAINT integracion_desarrollo_tipo_check,
    DROP CONSTRAINT integracion_desarrollo_check1,
    DROP COLUMN apertura_operacion_ref,
    ADD CONSTRAINT integracion_desarrollo_tipo_check CHECK (tipo IN ('orden','propuesta')),
    ADD CONSTRAINT integracion_desarrollo_check1 CHECK (
        (tipo='orden' AND orden_operacion_ref IS NULL) OR
        (tipo='propuesta' AND orden_operacion_ref IS NOT NULL));
COMMIT;
