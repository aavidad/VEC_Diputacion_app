\set ON_ERROR_STOP on
-- Focal de catálogo e inmutabilidad; el recorrido V3 se ensaya con la aplicación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $ct252_prueba$
DECLARE
 v_funcion text; v_old text; v_new text; v_proc pg_proc%ROWTYPE;
 v_no_nulo boolean; v_default text; v_forma text; v_validada boolean;
 v_ref text; v_version smallint; v_antes jsonb; v_despues jsonb;
 v_clave uuid; v_material text:='{"prueba":"ct252","tipo":"fixture"}';
BEGIN
 SELECT a.attnotnull,pg_get_expr(d.adbin,d.adrelid)
 INTO STRICT v_no_nulo,v_default
 FROM pg_attribute a JOIN pg_attrdef d
   ON d.adrelid=a.attrelid AND d.adnum=a.attnum
 WHERE a.attrelid='vec_contratacion_temporal.peticion_centro_revision'::regclass
   AND a.attname='organizacion_ref' AND NOT a.attisdropped;
 SELECT pg_get_constraintdef(c.oid),c.convalidated
 INTO STRICT v_forma,v_validada
 FROM pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.peticion_centro_revision'::regclass
   AND c.conname='peticion_centro_revision_organizacion_ref_forma';
 IF v_no_nulo IS NOT TRUE OR v_validada IS NOT TRUE
    OR position('organizacion:desarrollo:dipgra' in v_default)=0
    OR position('organizacion:desarrollo:dipgra' in v_forma)=0
    OR position('^org_[a-z0-9]{16,80}$' in v_forma)=0 THEN
   RAISE EXCEPTION 'CT252: columna, legado o forma organizativa divergentes';
 END IF;
 FOREACH v_funcion IN ARRAY ARRAY['consultar_peticiones_centro_v1',
   'registrar_peticion_centro_v1','gestionar_entrega_peticion_centro_v1',
   'preparar_entrega_peticion_centro_con_original_v1'] LOOP
   v_old:='vec_contratacion_temporal.'||v_funcion||
     '(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
   v_new:='vec_contratacion_temporal.'||v_funcion||
     '(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
   IF to_regprocedure(v_old) IS NOT NULL THEN
     RAISE EXCEPTION 'CT252: firma anterior abierta: %',v_funcion;
   END IF;
   SELECT * INTO STRICT v_proc FROM pg_proc
   WHERE oid=to_regprocedure(v_new);
   IF v_proc.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
      OR v_proc.prosecdef IS NOT TRUE
      OR v_proc.pronargs<>12
      OR NOT (v_proc.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
      OR v_proc.proacl IS DISTINCT FROM ARRAY[
       'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
       'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[] THEN
     RAISE EXCEPTION 'CT252: contrato o permisos divergentes: %',v_funcion;
   END IF;
 END LOOP;
 IF to_regprocedure('vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text)') IS NOT NULL THEN
   RAISE EXCEPTION 'CT252: proyección anterior abierta';
 END IF;
 SELECT * INTO STRICT v_proc FROM pg_proc WHERE oid=
   'vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1(text,text)'::regprocedure;
 IF v_proc.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR v_proc.prosecdef IS NOT TRUE OR v_proc.provolatile<>'s' OR v_proc.pronargs<>2
    OR v_proc.proargnames[3] IS DISTINCT FROM 'centro_ref'
    OR v_proc.proargnames[4] IS DISTINCT FROM 'categoria_ref'
    OR v_proc.proargmodes[3] IS DISTINCT FROM 't'
    OR v_proc.proargmodes[4] IS DISTINCT FROM 't'
    OR NOT (v_proc.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
    OR v_proc.proacl IS DISTINCT FROM ARRAY[
      'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
      'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[] THEN
   RAISE EXCEPTION 'CT252: contrato o permisos de proyección divergentes';
 END IF;
 -- Si hay legado, se comprueba esa fila sin tocarla. En una copia vacía se
 -- crea una sola fila de prueba, revertida al acabar; el recorrido V3 va por HTTP.
 SELECT r.peticion_ref,r.version,to_jsonb(r)
 INTO v_ref,v_version,v_antes
 FROM vec_contratacion_temporal.peticion_centro_revision r
 WHERE r.organizacion_ref='organizacion:desarrollo:dipgra'
 ORDER BY r.registrada_en,r.peticion_ref,r.version LIMIT 1;
 IF NOT FOUND THEN
   v_clave:=gen_random_uuid();
   v_ref:='peticion:centro:'||v_clave::text;
   INSERT INTO vec_contratacion_temporal.peticion_centro_revision(
     peticion_ref,version,clave_idempotencia,operacion,material,material_sha256,
     actor_ref,perfil_ref,centro_ref,puesto_ref,configuracion_ref,
     configuracion_version,comando,peticion,estado,recibo_ref,recibo_json,
     auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
   VALUES(v_ref,1,v_clave,'presentar',v_material,
     encode(sha256(convert_to(v_material,'UTF8')),'hex'),
     'actor:ct252:fixture','perfil:ct252:fixture','centro:ct252:fixture',
     'puesto:ct252:fixture','config:ct252:fixture',1,'{}'::jsonb,
     jsonb_build_object('referencia',v_ref),'pendiente_ratificacion',
     'recibo:ct252:'||v_clave::text,
     jsonb_build_object('recibo_ref','recibo:ct252:'||v_clave::text),
     'auditoria:ct252:'||v_clave::text,'decision:ct252:'||v_clave::text,
     encode(sha256(convert_to('ct252:'||v_clave::text,'UTF8')),'hex'),
     clock_timestamp()) RETURNING version INTO v_version;
   SELECT to_jsonb(r) INTO STRICT v_antes
   FROM vec_contratacion_temporal.peticion_centro_revision r
   WHERE r.peticion_ref=v_ref AND r.version=v_version;
   IF v_antes->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra' THEN
     RAISE EXCEPTION 'CT252: DEFAULT legado no aplicado al fixture';
   END IF;
 END IF;
 BEGIN
   UPDATE vec_contratacion_temporal.peticion_centro_revision
   SET organizacion_ref='org_aaaaaaaaaaaaaaaa'
   WHERE peticion_ref=v_ref AND version=v_version;
   RAISE EXCEPTION 'CT252: UPDATE de organización permitido';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
   UPDATE vec_contratacion_temporal.peticion_centro_revision
   SET material=material||' '
   WHERE peticion_ref=v_ref AND version=v_version;
   RAISE EXCEPTION 'CT252: UPDATE de material permitido';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 SELECT to_jsonb(r) INTO STRICT v_despues
 FROM vec_contratacion_temporal.peticion_centro_revision r
 WHERE r.peticion_ref=v_ref AND r.version=v_version;
 IF v_despues IS DISTINCT FROM v_antes THEN
   RAISE EXCEPTION 'CT252: historia modificada tras rechazar UPDATE';
 END IF;
END
$ct252_prueba$;
ROLLBACK;
