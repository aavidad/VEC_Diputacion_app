\set ON_ERROR_STOP on
-- En copia desechable después de CT233/AD290/B133. No simula autorización.
BEGIN;
SET LOCAL timezone='UTC';
-- El operador crea la utilidad temporal; el propietario no necesita TEMP.
CREATE FUNCTION pg_temp.hecho(v jsonb) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('valor',v,'documento_ref','ref:'||repeat('b',64),'huella',repeat('a',64))
$f$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $test$ DECLARE h jsonb;c jsonb;r jsonb;caso jsonb;
BEGIN
 c:='{"fuente_normativa":"Reglamento de bolsas, BOP 9/2026, art. 9, apartado cuarto (numerado 3)","mejora":{
 "habilitada":true,"modalidades_destino":["vacante"],"causas_destino":["vacante"],"maximo_vacante_meses":36,
 "supuestos_reduccion_vacante":["programa"],"reduccion_por_modalidades_previas":["acumulacion_tareas","sustitucion"],
 "condiciones_actuales":[{"modalidad":"acumulacion_tareas"},{"modalidad":"sustitucion","causa":"baja",
 "requiere_baja_comprobada":true,"meses_transcurridos":{"meses":9,"incluye_limite":true}},
 {"modalidad":"programa","meses_duracion_inicial":{"meses":9,"incluye_limite":false}}]}}';
 SELECT jsonb_object_agg(key,pg_temp.hecho(value)) INTO h FROM jsonb_each(
 '{"trabaja_en_diputacion":true,"tipo_relacion":"sustitucion","modalidad":"sustitucion","causa_relacion":"baja",
 "inicio":"2026-04-01","fin_previsto":"2027-01-01","es_vacante":true,"prelacion":1,
 "antecedentes_completos":true,"fecha_inicial_baja":"2026-01-16"}'::jsonb);
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h,c,'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'pendiente' OR r->>'motivo_clave'<>'guarda_contratacion_pendiente'
 THEN RAISE EXCEPTION 'baja: nueve meses incluidos: %',r; END IF;
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h,c,'2026-10-17T00:00:00Z');
 IF r->>'resultado'<>'no_aplicable' THEN RAISE EXCEPTION 'se midió desde sustituto, no baja: %',r; END IF;
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h-'fecha_inicial_baja',c,'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'pendiente' THEN RAISE EXCEPTION 'se presumió la baja: %',r; END IF;
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h-'causa_relacion',c,'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'pendiente' THEN RAISE EXCEPTION 'se presumió la causa contractual: %',r; END IF;
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(
  jsonb_set(h,'{causa_relacion,valor}','"otra"'),c,'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'no_aplicable' THEN RAISE EXCEPTION 'se ignoró la causa contractual: %',r; END IF;
 h:=h-'fecha_inicial_baja'||jsonb_build_object('tipo_relacion',pg_temp.hecho('"programa"'),
  'modalidad',pg_temp.hecho('"programa"'),'fin_inicial_programa',pg_temp.hecho('"2026-12-31"'));
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h,c,'2026-10-16T00:00:00Z');
 IF r->>'motivo_clave'<>'descuento_pendiente_criterio_rrhh' THEN RAISE EXCEPTION 'programa <9: %',r; END IF;
 caso:=jsonb_set(h,'{fin_inicial_programa,valor}','"2027-01-01"');
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(caso,c,'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'no_aplicable' THEN RAISE EXCEPTION 'programa de nueve meses excluidos: %',r; END IF;
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(h-'antecedentes_completos',c,'2026-10-16T00:00:00Z');
 IF r->>'motivo_clave'<>'antecedentes_incompletos' THEN RAISE EXCEPTION 'antecedentes inventados: %',r; END IF;
 -- Cambiar el umbral publicado cambia el resultado sin modificar el cálculo.
 r:=vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(caso,
  jsonb_set(c,'{mejora,condiciones_actuales,2,meses_duracion_inicial,meses}','10'),'2026-10-16T00:00:00Z');
 IF r->>'resultado'<>'pendiente' THEN RAISE EXCEPTION 'umbral fijo en código: %',r; END IF;
 FOR caso IN SELECT value FROM jsonb_array_elements(jsonb_build_array(
  h||'{"elegible":true}',jsonb_set(h,'{es_vacante}','true'),
  jsonb_set(h,'{inicio}',(h->'inicio')-'huella'),
  jsonb_set(h,'{inicio}',(h->'inicio')||'{"verificada_por":"persona:otra"}'))) LOOP
  BEGIN
   PERFORM vec_bolsa_llamamientos.calcular_comprobacion_ofrecimiento_v1(caso,c,'2026-10-16T00:00:00Z');
   RAISE EXCEPTION 'se aceptó resultado o evidencia inventados';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
END $test$;
RESET ROLE;
DO $seguridad$ DECLARE f regprocedure;x record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
 'vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
 'vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(jsonb,boolean)'::regprocedure,
 'vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF (SELECT prosecdef AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp'] FROM pg_proc WHERE oid=f) IS NOT TRUE
  OR EXISTS(SELECT 1 FROM pg_proc p,aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'función abierta: %',f; END IF;
 END LOOP;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1','SELECT')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1','INSERT')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'se abrió acceso directo'; END IF;
 IF (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
  WHERE oid='vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1'::regclass) IS NOT TRUE
 THEN RAISE EXCEPTION 'RLS no forzada'; END IF;
 -- Incluso el propietario rechaza mutar o truncar la historia.
 BEGIN
  TRUNCATE vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1;
  RAISE EXCEPTION 'se permitió truncar';
 EXCEPTION WHEN insufficient_privilege OR object_not_in_prerequisite_state THEN NULL; END;
 -- La consulta de motivos no confiere esta acción.
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
   convert_to('{"operacion":"bolsa.origen_ct.consultar"}','UTF8'),'{}','{}','{}',1,1,'{}','{}','{}','{}');
  RAISE EXCEPTION 'consulta convertida en escritura';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $seguridad$;
-- Un LOGIN del ejecutor tampoco puede registrar sin capacidad V3 ni leer tablas.
DO $login$ BEGIN
 EXECUTE format('CREATE ROLE vec_b133_ejecutor_fixture LOGIN INHERIT NOSUPERUSER NOCREATEDB '
  'NOCREATEROLE NOREPLICATION NOBYPASSRLS VALID UNTIL %L',clock_timestamp()+interval '90 days');
END $login$;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b133_ejecutor_fixture
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET LOCAL SESSION AUTHORIZATION vec_b133_ejecutor_fixture;
DO $sin_autoridad$ DECLARE c jsonb;p jsonb;
BEGIN
 SELECT jsonb_object_agg(k,to_jsonb('ref:fixture'::text)) INTO c FROM unnest(ARRAY[
  'ambito_ref','bolsa_ref','categoria_ref','causa_ofrecida','expediente_ref',
  'modalidad_ofrecida','necesidad_ref','organizacion_ref','participacion_ref','unidad_ref']) k;
 c:=c||jsonb_build_object('huella_politica_ofertas_sha256',repeat('a',64),'orden_acta',1,
  'version_bolsa',1,'version_expediente',6,'version_instantanea',1,'version_politica_ofertas',1);
 p:=jsonb_build_object('contexto',c,'hechos','{}'::jsonb,'version_esperada',0,'clave_idempotencia','b133:sin:autoridad');
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_comprobacion_ofrecimiento_v1(
   p,'{}','{}','{}','{}',1,1,'{}','{}','{}','{}');
  RAISE EXCEPTION 'se registró sin autoridad V3';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1;
  RAISE EXCEPTION 'el LOGIN leyó los hechos directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $sin_autoridad$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
