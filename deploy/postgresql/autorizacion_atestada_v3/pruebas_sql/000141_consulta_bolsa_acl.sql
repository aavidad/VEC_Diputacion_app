\set ON_ERROR_STOP on
-- Ejecutar dentro del ensayo aislado, tras AD141, sin COMMIT.
BEGIN ISOLATION LEVEL SERIALIZABLE;
-- Las entradas vacías son solo negativas; no simulan atestaciones válidas.
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contratos$
DECLARE x record; f oid; padre oid; acl_count integer;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('consumir_consulta_version_convocatoria_v3_atestada','vec_bolsa_convocatorias_propietario')) v(nombre,rol) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  padre:=x.rol::regrole;
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
      AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz','decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  THEN RAISE EXCEPTION 'AD141: contrato de fachada inesperado'; END IF;
  SELECT count(*) INTO acl_count FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f;
  IF acl_count<>2 OR NOT has_schema_privilege(padre,'vec_autorizacion_atestada_v3','USAGE')
     OR NOT has_function_privilege(padre,f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid=f AND (a.grantee NOT IN (padre,p.proowner) OR a.grantor<>p.proowner
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD141: ACL de fachada abierta'; END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable))
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD141: núcleo accesible por consumidor o runtime'; END IF;
 IF has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',
  'vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'AD141: runtime accede directamente a fachada'; END IF;
END $contratos$;
DO $negativos$
DECLARE capacidad jsonb; decision jsonb; nombre text; errores integer:=0;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consumir_consulta_version_convocatoria_v3_atestada'] LOOP
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD141: material vacío autorizado';
  EXCEPTION WHEN insufficient_privilege THEN errores:=errores+1; END;
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING decode('ff','hex'),convert_to('{}','UTF8'),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD141: UTF8 inválido autorizado';
  EXCEPTION WHEN invalid_parameter_value THEN errores:=errores+1; END;
 END LOOP;
 IF errores<>2 THEN RAISE EXCEPTION 'AD141: negativos incompletos'; END IF;
END $negativos$;
ROLLBACK;
