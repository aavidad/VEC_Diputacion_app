\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000004',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo IN ACCESS EXCLUSIVE MODE;

-- El terminal pertenece al mismo registro inmutable de operaciones de Bolsa.
-- La fila de llamamiento de 000003 sigue siendo su apertura histórica v1;
-- no se crea otra tabla ni una aceptación CT que compita con este terminal.
DO $estructura$
DECLARE v_def text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc p
        WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
          AND strpos(pg_get_functiondef(p.oid),'bolsa.llamamiento.aceptacion_rrhh.registrar')>0) THEN
        RAISE EXCEPTION 'falta autorización nominal de aceptación RRHH' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid,true) INTO STRICT v_def FROM pg_constraint
     WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
       AND conname='integracion_desarrollo_tipo_check' AND contype='c' AND convalidated;
    IF v_def IS DISTINCT FROM $tipo$CHECK (tipo = ANY (ARRAY['orden'::text, 'propuesta'::text]))$tipo$ THEN
        RAISE EXCEPTION 'tipos de integración Bolsa incompatibles' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid,true) INTO STRICT v_def FROM pg_constraint
     WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass
       AND conname='integracion_desarrollo_check1' AND contype='c' AND convalidated;
    IF v_def IS DISTINCT FROM $vinculo$CHECK (tipo = 'orden'::text AND orden_operacion_ref IS NULL OR tipo = 'propuesta'::text AND orden_operacion_ref IS NOT NULL)$vinculo$ THEN
        RAISE EXCEPTION 'vínculo de orden Bolsa incompatible' USING ERRCODE='55000';
    END IF;
END
$estructura$;
ALTER TABLE vec_bolsa_llamamientos.integracion_desarrollo
    ADD COLUMN apertura_operacion_ref text,
    DROP CONSTRAINT integracion_desarrollo_tipo_check,
    DROP CONSTRAINT integracion_desarrollo_check1,
    ADD CONSTRAINT integracion_desarrollo_tipo_check CHECK (tipo IN ('orden','propuesta','aceptacion_rrhh')),
    ADD CONSTRAINT integracion_desarrollo_check1 CHECK (
        (tipo='orden' AND orden_operacion_ref IS NULL) OR
        (tipo IN ('propuesta','aceptacion_rrhh') AND orden_operacion_ref IS NOT NULL)),
    ADD CONSTRAINT integracion_desarrollo_apertura_fk FOREIGN KEY (apertura_operacion_ref)
        REFERENCES vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref),
    ADD CONSTRAINT integracion_desarrollo_terminal_apertura_unico UNIQUE (apertura_operacion_ref),
    ADD CONSTRAINT integracion_desarrollo_apertura_check CHECK (
        (tipo='aceptacion_rrhh' AND apertura_operacion_ref IS NOT NULL AND apertura_operacion_ref<>operacion_ref) OR
        (tipo<>'aceptacion_rrhh' AND apertura_operacion_ref IS NULL));
COMMENT ON COLUMN vec_bolsa_llamamientos.integracion_desarrollo.apertura_operacion_ref IS
    'Único terminal derivado de una apertura confirmada; apertura, fuente, propuesta e historia originales no se modifican.';

DO $ampliar$
DECLARE v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
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
      ('antecedente_terminal',
       $antes$ IF r->>'tipo'='orden' THEN$antes$,
       $despues$ IF r->>'tipo'='aceptacion_rrhh' THEN
  -- Esta rama no verifica correo ni calcula un plazo legal. La autorización
  -- específica debe proceder del proveedor que acredita dato y plazo.
  IF jsonb_typeof(v_resolucion) IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='falta resolución de aceptación RRHH';
  END IF;
  IF (SELECT count(*) FROM jsonb_object_keys(v_resolucion))<>8 OR
   (SELECT count(*) FROM json_each(convert_from(p_registro,'UTF8')::json->'resolucion'))<>8 OR
   NOT (v_resolucion ?& ARRAY['apertura_operacion_ref','justificante_ref','evaluacion_plazo_ref',
       'politica_ref','politica_version','politica_sha256','version_esperada','resuelta_en']) OR
   EXISTS (SELECT 1 FROM unnest(ARRAY['apertura_operacion_ref','justificante_ref','evaluacion_plazo_ref','politica_ref']) AS k(clave)
       WHERE jsonb_typeof(v_resolucion->k.clave) IS DISTINCT FROM 'string'
          OR (v_resolucion->>k.clave)!~'^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$') OR
   jsonb_typeof(v_resolucion->'politica_version') IS DISTINCT FROM 'number' OR
   (v_resolucion->>'politica_version')!~'^[1-9][0-9]{0,15}$' OR
   jsonb_typeof(v_resolucion->'politica_sha256') IS DISTINCT FROM 'string' OR
   (v_resolucion->>'politica_sha256')!~'^[0-9a-f]{64}$' OR
   v_resolucion->>'politica_sha256'=repeat('0',64) OR
   v_resolucion->'version_esperada' IS DISTINCT FROM '1'::jsonb OR
   jsonb_typeof(v_resolucion->'resuelta_en') IS DISTINCT FROM 'string' OR
   (v_resolucion->>'resuelta_en')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{1,6})?Z$' THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='resolución de aceptación RRHH inválida';
  END IF;
  IF (v_resolucion->>'politica_version')::numeric>9007199254740991 OR
   (v_resolucion->>'resuelta_en')::timestamptz='0001-01-01T00:00:00Z'::timestamptz THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='versión o fecha de resolución inválida';
  END IF;
  SELECT * INTO STRICT v_apertura FROM vec_bolsa_llamamientos.integracion_desarrollo o
   WHERE o.operacion_ref=v_resolucion->>'apertura_operacion_ref' AND o.tipo='propuesta' FOR SHARE;
  v_apertura_json:=convert_from(v_apertura.registro_canonico,'UTF8')::jsonb;
  SELECT * INTO STRICT v_llamamiento FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo a
   WHERE a.operacion_ref=v_apertura.operacion_ref FOR SHARE;
  IF r->>'operacion_ref'=v_apertura.operacion_ref OR
   (r-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) IS DISTINCT FROM
       (v_apertura_json-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) OR
   jsonb_typeof(l) IS DISTINCT FROM 'object' OR l->'Version' IS DISTINCT FROM '2'::jsonb OR
   (l-'Version') IS DISTINCT FROM ((v_apertura_json->'llamamiento')-'Version') OR
   r->>'estado_llamamiento' IS DISTINCT FROM 'aceptacion' OR
   v_apertura_json->>'estado_llamamiento' IS DISTINCT FROM 'abierto' OR
   v_llamamiento.llamamiento_ref IS DISTINCT FROM l->>'LlamamientoRef' OR
   v_llamamiento.version IS DISTINCT FROM 1::bigint OR v_llamamiento.estado IS DISTINCT FROM 'abierto' OR
   v_llamamiento.datos_canonicos IS DISTINCT FROM v_apertura_json->'llamamiento' OR
   (v_resolucion->>'resuelta_en')::timestamptz<v_apertura.confirmada_en OR
   (v_resolucion->>'resuelta_en')::timestamptz>clock_timestamp() THEN
   RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='aceptación desligada de su apertura original';
  END IF;
 ELSIF r->>'tipo'='orden' THEN$despues$),
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
        IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior)
           OR strpos(v_def,v_cambio.nuevo)<>0 THEN
            RAISE EXCEPTION 'función Bolsa incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.anterior,v_cambio.nuevo);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'aceptación alteró permisos de integración Bolsa' USING ERRCODE='55000';
    END IF;
END
$ampliar$;
COMMIT;
