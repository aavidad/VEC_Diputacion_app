\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000005',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo,
    vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
    vec_bolsa_llamamientos.outbox_integracion_desarrollo IN ACCESS EXCLUSIVE MODE;
-- Mismo registro, misma apertura y único terminal: aceptación O renuncia.
-- No cambia la apertura histórica v1 ni la fuente firmada, orden o propuesta.
-- Conserva los índices únicos, las FK, el CHECK de SHA y el consumo nuevo.
DO $cambio$
DECLARE v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='consumir_decision_mutacion_v3_interna'
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
          AND strpos(pg_get_functiondef(p.oid),'bolsa.llamamiento.renuncia_rrhh.registrar')>0) THEN
        RAISE EXCEPTION 'falta permiso nominal de renuncia RRHH' USING ERRCODE='55000';
    END IF;
    -- Se cambia solo el miembro del tipo permitido. Los demás predicados
    -- de cada CHECK se mantienen; cualquier marca distinta o repetida falla.
    FOR v_cambio IN SELECT * FROM (VALUES
      ('integracion_desarrollo_tipo_check',$antes$ARRAY['orden'::text, 'propuesta'::text, 'aceptacion_rrhh'::text]$antes$,$despues$ARRAY['orden'::text, 'propuesta'::text, 'aceptacion_rrhh'::text, 'renuncia_rrhh'::text]$despues$),
      ('integracion_desarrollo_check1',$antes$ARRAY['propuesta'::text, 'aceptacion_rrhh'::text]$antes$,$despues$ARRAY['propuesta'::text, 'aceptacion_rrhh'::text, 'renuncia_rrhh'::text]$despues$),
      ('integracion_desarrollo_apertura_check',$antes$tipo = 'aceptacion_rrhh'::text$antes$,$despues$tipo = ANY (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text])$despues$),
      ('integracion_desarrollo_apertura_check',$antes$tipo <> 'aceptacion_rrhh'::text$antes$,$despues$tipo <> ALL (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text])$despues$)
    ) AS cambios(nombre,anterior,nuevo) LOOP
        SELECT pg_get_constraintdef(oid,true) INTO STRICT v_def FROM pg_constraint
         WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
           AND conname=v_cambio.nombre AND contype='c' AND convalidated;
        IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior) THEN
            RAISE EXCEPTION 'CHECK Bolsa incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.integracion_desarrollo DROP CONSTRAINT %I, ADD CONSTRAINT %I %s',
            v_cambio.nombre,v_cambio.nombre,replace(v_def,v_cambio.anterior,v_cambio.nuevo));
    END LOOP;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
    FOR v_cambio IN SELECT * FROM (VALUES
      ('tipo',$antes$r->>'tipo' IN ('orden','propuesta','aceptacion_rrhh') AND$antes$,$despues$r->>'tipo' IN ('orden','propuesta','aceptacion_rrhh','renuncia_rrhh') AND$despues$),
      ('material_resolucion',$antes$ IF r->>'tipo' IS DISTINCT FROM 'aceptacion_rrhh' AND r?'resolucion' THEN$antes$,$despues$ IF (r->>'tipo') NOT IN ('aceptacion_rrhh','renuncia_rrhh') AND r?'resolucion' THEN$despues$),
      ('accion',$antes$  WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrar' ELSE 'bolsa.llamamiento.abrir' END;$antes$,$despues$  WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrar'
  WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar' ELSE 'bolsa.llamamiento.abrir' END;$despues$),
      ('terminal',$antes$ IF r->>'tipo'='aceptacion_rrhh' THEN$antes$,$despues$ IF r->>'tipo' IN ('aceptacion_rrhh','renuncia_rrhh') THEN$despues$),
      ('estado',$antes$   r->>'estado_llamamiento' IS DISTINCT FROM 'aceptacion' OR$antes$,$despues$   r->>'estado_llamamiento' IS DISTINCT FROM
       (CASE r->>'tipo' WHEN 'aceptacion_rrhh' THEN 'aceptacion' ELSE 'renuncia' END) OR$despues$),
      ('evento',$antes$   WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrada' ELSE 'bolsa.llamamiento.abierto' END,$antes$,$despues$   WHEN 'aceptacion_rrhh' THEN 'bolsa.llamamiento.aceptacion_rrhh.registrada'
   WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrada' ELSE 'bolsa.llamamiento.abierto' END,$despues$)
    ) AS cambios(nombre,anterior,nuevo) LOOP
        IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior) THEN
            RAISE EXCEPTION 'función Bolsa incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.anterior,v_cambio.nuevo);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'renuncia alteró permisos Bolsa' USING ERRCODE='55000';
    END IF;
END
$cambio$;
COMMIT;
