\set ON_ERROR_STOP on
-- Bolsa 000039: terminal «sin respuesta» (expiracion_rrhh) en la integración
-- con Contratación temporal. Cuando RRHH confirma que venció el plazo sin
-- respuesta, el llamamiento se cierra con estado expiracion_gobernada por el
-- mismo mecanismo que la renuncia (000005): mismo registro inmutable, única
-- terminal por apertura, auditoría y outbox en la misma transacción. Usa el
-- permiso nominal ya existente de no aceptación registrada por RRHH
-- (bolsa.llamamiento.renuncia_rrhh.registrar): no crea acción ni consumidor.
-- La continuación de 000006 admite también este terminal como antecedente.
-- Las referencias de la resolución admiten además «_»: las reglas del
-- catálogo se identifican como catalogo:version:entrada (b08.sin_respuesta_baja).
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000039',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo,
    vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
    vec_bolsa_llamamientos.outbox_integracion_desarrollo IN ACCESS EXCLUSIVE MODE;
DO $cambio$
DECLARE v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
    IF strpos((SELECT pg_get_constraintdef(oid,true) FROM pg_constraint
        WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
          AND conname='integracion_desarrollo_tipo_check'),'expiracion_rrhh')>0 THEN
        RAISE EXCEPTION 'Bolsa 000039 ya instalada: no se reaplica' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='consumir_decision_mutacion_v3_interna'
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
          AND strpos(pg_get_functiondef(p.oid),'bolsa.llamamiento.renuncia_rrhh.registrar')>0
          AND strpos(pg_get_functiondef(p.oid),'bolsa.llamamiento.siguiente.abrir')>0) THEN
        RAISE EXCEPTION 'falta permiso nominal de no aceptación RRHH o de continuación' USING ERRCODE='55000';
    END IF;
    -- Solo se añade el miembro del tipo; el resto de cada CHECK se mantiene.
    FOR v_cambio IN SELECT * FROM (VALUES
      ('integracion_desarrollo_tipo_check',$antes$'aceptacion_rrhh'::text, 'renuncia_rrhh'::text]$antes$,$despues$'aceptacion_rrhh'::text, 'renuncia_rrhh'::text, 'expiracion_rrhh'::text]$despues$),
      ('integracion_desarrollo_check1',$antes$'aceptacion_rrhh'::text, 'renuncia_rrhh'::text]$antes$,$despues$'aceptacion_rrhh'::text, 'renuncia_rrhh'::text, 'expiracion_rrhh'::text]$despues$),
      ('integracion_desarrollo_apertura_check',$antes$tipo = ANY (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text])$antes$,$despues$tipo = ANY (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text, 'expiracion_rrhh'::text])$despues$),
      ('integracion_desarrollo_apertura_check',$antes$tipo <> ALL (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text])$antes$,$despues$tipo <> ALL (ARRAY['aceptacion_rrhh'::text, 'renuncia_rrhh'::text, 'expiracion_rrhh'::text])$despues$)
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
      ('material_resolucion',$antes$ IF (r->>'tipo') NOT IN ('aceptacion_rrhh','renuncia_rrhh') AND r?'resolucion' THEN$antes$,
       $despues$ IF (r->>'tipo') NOT IN ('aceptacion_rrhh','renuncia_rrhh','expiracion_rrhh') AND r?'resolucion' THEN$despues$),
      ('tipo',$antes$  r->>'tipo' IN ('orden','propuesta','aceptacion_rrhh','renuncia_rrhh') AND$antes$,
       $despues$  r->>'tipo' IN ('orden','propuesta','aceptacion_rrhh','renuncia_rrhh','expiracion_rrhh') AND$despues$),
      ('accion',$antes$  WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar'
$antes$,$despues$  WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar'
  WHEN 'expiracion_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar'
$despues$),
      ('terminal',$antes$ IF r->>'tipo' IN ('aceptacion_rrhh','renuncia_rrhh') THEN$antes$,
       $despues$ IF r->>'tipo' IN ('aceptacion_rrhh','renuncia_rrhh','expiracion_rrhh') THEN$despues$),
      ('estado',$antes$       (CASE r->>'tipo' WHEN 'aceptacion_rrhh' THEN 'aceptacion' ELSE 'renuncia' END) OR$antes$,
       $despues$       (CASE r->>'tipo' WHEN 'aceptacion_rrhh' THEN 'aceptacion'
        WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' ELSE 'renuncia' END) OR$despues$),
      ('continuacion_terminal',$antes$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND tipo='renuncia_rrhh' FOR SHARE;$antes$,
       $despues$    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND tipo IN ('renuncia_rrhh','expiracion_rrhh') FOR SHARE;$despues$),
      ('continuacion_estado',$antes$      v_terminal_json->>'estado_llamamiento' IS DISTINCT FROM 'renuncia' OR$antes$,
       $despues$      v_terminal_json->>'estado_llamamiento' IS DISTINCT FROM
       (CASE v_terminal_anterior.tipo WHEN 'expiracion_rrhh' THEN 'expiracion_gobernada' ELSE 'renuncia' END) OR$despues$),
      ('referencias_resolucion',$antes$          OR (v_resolucion->>k.clave)!~'^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$') OR$antes$,
       $despues$          OR (v_resolucion->>k.clave)!~'^[A-Za-z0-9][A-Za-z0-9:._/_-]{0,191}$') OR$despues$),
      ('evento',$antes$   WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrada' ELSE 'bolsa.llamamiento.abierto' END,$antes$,
       $despues$   WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrada'
   WHEN 'expiracion_rrhh' THEN 'bolsa.llamamiento.expiracion_rrhh.registrada' ELSE 'bolsa.llamamiento.abierto' END,$despues$)
    ) AS cambios(nombre,anterior,nuevo) LOOP
        IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior) THEN
            RAISE EXCEPTION 'función Bolsa incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.anterior,v_cambio.nuevo);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'expiración alteró permisos Bolsa' USING ERRCODE='55000';
    END IF;
END
$cambio$;
COMMIT;
