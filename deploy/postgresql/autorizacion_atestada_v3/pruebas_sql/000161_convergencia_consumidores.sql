\set ON_ERROR_STOP on
-- El ejecutor del ensayo abre SERIALIZABLE e incluye esta prueba antes de ROLLBACK.
-- Las entradas negativas no son atestaciones positivas ni registran negocio.
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contratos$
DECLARE x record; f oid; padre oid; nucleo oid;
BEGIN
 nucleo:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR x IN SELECT * FROM (VALUES
  ('consumir_consulta_version_convocatoria_v3_atestada','vec_bolsa_convocatorias_propietario','vec_bolsa_convocatorias_ejecutor_consulta'),
  ('consumir_operacion_meritos_v3_atestada','vec_meritos_propietario','vec_meritos_ejecutor'),
  ('registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada','vec_bolsa_reglas_baremo_propietario','vec_bolsa_reglas_baremo_ejecutor_gobierno'),
  ('consumir_vinculo_propio_crn11_v3_atestada','vec_personal_propietario','vec_personal_ejecutor')
 ) v(nombre,propietario,ejecutor) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  padre:=x.propietario::regrole;
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz','decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  OR NOT has_schema_privilege(padre,'vec_autorizacion_atestada_v3','USAGE')
  OR NOT has_function_privilege(padre,f,'EXECUTE')
  OR has_function_privilege(x.ejecutor,f,'EXECUTE')
  OR has_function_privilege(x.ejecutor,nucleo,'EXECUTE')
  OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN (padre,p.proowner) OR a.grantor<>p.proowner
    OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD161: contrato o separación de autoridad incompatible'; END IF;
 END LOOP;
END $contratos$;
DO $material_sin_autoridad$
DECLARE x text; rechazados integer:=0; antes bigint;
BEGIN
 SELECT count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 FOREACH x IN ARRAY ARRAY[
  'consumir_consulta_version_convocatoria_v3_atestada',
  'consumir_operacion_meritos_v3_atestada',
  'registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada',
  'consumir_vinculo_propio_crn11_v3_atestada'
 ] LOOP
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',x)
    USING convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,''::bytea,
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD161: material sin autoridad aceptado';
  EXCEPTION WHEN insufficient_privilege THEN rechazados:=rechazados+1; END;
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',x)
    USING decode('ff','hex'),convert_to('{}','UTF8'),''::bytea,''::bytea,
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD161: material inválido aceptado';
  EXCEPTION WHEN invalid_parameter_value THEN rechazados:=rechazados+1; END;
 END LOOP;
 IF rechazados<>8 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3) IS DISTINCT FROM antes
 THEN RAISE EXCEPTION 'AD161: rechazos incompletos o efectos sin autorización'; END IF;
END $material_sin_autoridad$;
SELECT 'AD161-CONTRATOS-Y-8-RECHAZOS-OK';
