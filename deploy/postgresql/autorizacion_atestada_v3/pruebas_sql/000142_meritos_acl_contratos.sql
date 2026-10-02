\set ON_ERROR_STOP on
-- Ejecutar en una transacción del ensayo aislado, tras AD142 y sin COMMIT.
-- AD141 pertenece a la entrega S1; este ensayo comprueba únicamente Méritos.
-- Las entradas vacías son solo negativas; no simulan atestaciones válidas.
RESET ROLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $contratos$
DECLARE x record; f oid; padre oid; acl_count integer;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('consumir_operacion_meritos_v3_atestada','vec_meritos_propietario')) v(nombre,rol) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  padre:=x.rol::regrole;
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
      AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz','decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  THEN RAISE EXCEPTION 'AD142: contrato de fachada inesperado'; END IF;
  SELECT count(*) INTO acl_count FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f;
  IF acl_count<>2 OR NOT has_schema_privilege(padre,'vec_autorizacion_atestada_v3','USAGE')
     OR NOT has_function_privilege(padre,f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid=f AND (a.grantee NOT IN (padre,p.proowner) OR a.grantor<>p.proowner
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD142: ACL de fachada abierta'; END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable))
 OR has_function_privilege('vec_meritos_ejecutor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD142: núcleo accesible por consumidor o runtime'; END IF;
 IF has_function_privilege('vec_meritos_ejecutor',
  'vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'AD142: runtime accede directamente a fachada'; END IF;
 IF EXISTS (SELECT 1 FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check'
    AND (strpos(pg_get_constraintdef(c.oid,true),'vec_meritos.hecho.verificar')>0
      OR NOT c.convalidated))
 THEN RAISE EXCEPTION 'AD142: verificación o CHECK sin validar'; END IF;
END $contratos$;
DO $negativos$
DECLARE capacidad jsonb; decision jsonb; nombre text; errores integer:=0;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consumir_operacion_meritos_v3_atestada'] LOOP
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD142: material vacío autorizado';
  EXCEPTION WHEN insufficient_privilege THEN errores:=errores+1; END;
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING decode('ff','hex'),convert_to('{}','UTF8'),''::bytea,''::bytea,1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD142: UTF8 inválido autorizado';
  EXCEPTION WHEN invalid_parameter_value THEN errores:=errores+1; END;
 END LOOP;
 capacidad:=jsonb_build_object('operacion','meritos.hecho.verificar','audiencia_consumo','vec_meritos.hecho.rechazar.v1','efecto_ref','hecho:sintetico:prueba','huella_efecto_sha256',repeat('a',64));
 decision:=jsonb_build_object('accion','meritos.hecho.verificar','modulo_id','meritos','tipo_recurso','hecho','finalidad','revision_hecho_merito','recurso_ref','hecho:sintetico:prueba','contexto_recurso_huella_sha256',repeat('a',64),'campos_permitidos',jsonb_build_array('declarante_ref','hecho','recibo','version'),'obligaciones',jsonb_build_array('auditar'),'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'));
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(
   convert_to(capacidad::text,'UTF8'),convert_to(decision::text,'UTF8'),''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
  RAISE EXCEPTION 'AD142: verificar autorizada';
 EXCEPTION WHEN insufficient_privilege THEN errores:=errores+1; END;
 IF errores<>3 THEN RAISE EXCEPTION 'AD142: negativos incompletos'; END IF;
END $negativos$;
DO $exterior_cerrado$
DECLARE accion text; c jsonb; d jsonb; mensaje text; rechazados integer:=0;
 antes bigint; despues bigint; fuente text;
BEGIN
 SELECT count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 SELECT prosrc INTO STRICT fuente FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 -- El perfil exterior queda únicamente excluido del bloque general CT:
 -- no está seleccionado en runtime ni admitido por ninguna rama de consumo.
 IF length(fuente)-length(replace(fuente,'meritos_hecho_propio_externo',''))
     <> length('meritos_hecho_propio_externo')
 OR strpos(fuente,'AND p_perfil_mutacion IS DISTINCT FROM ''meritos_hecho_propio_externo''')=0
 THEN RAISE EXCEPTION 'AD142: perfil exterior seleccionable'; END IF;
 FOREACH accion IN ARRAY ARRAY['declarar','rectificar','rechazar'] LOOP
  c:=jsonb_build_object('operacion','meritos.hecho.'||accion,
   'audiencia_consumo','vec_meritos.hecho.'||accion||'.v1',
   'efecto_ref','hecho:sintetico:exterior','huella_efecto_sha256',repeat('a',64));
  d:=jsonb_build_object('accion','meritos.hecho.'||accion,'modulo_id','meritos',
   'tipo_recurso','hecho','finalidad',CASE accion WHEN 'declarar' THEN 'declaracion_hecho_propio'
     WHEN 'rectificar' THEN 'rectificacion_hecho_propio' ELSE 'revision_hecho_merito' END,
   'recurso_ref','hecho:sintetico:exterior','contexto_recurso_huella_sha256',repeat('a',64),
   'campos_permitidos',jsonb_build_array('declarante_ref','hecho','recibo','version'),
   'obligaciones',jsonb_build_array('auditar'),
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'));
  BEGIN
   -- Material de forma nominal correcta; la rama exterior debe denegar antes
   -- de consultar criptografía o gobierno. No es una atestación positiva.
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(
    convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,''::bytea,1,1,
    ''::bytea,''::bytea,''::bytea,''::bytea);
   RAISE EXCEPTION 'AD142: operación exterior autorizada';
  EXCEPTION WHEN insufficient_privilege THEN
   GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
   IF mensaje IS DISTINCT FROM 'AD3-142: exterior no habilitado' THEN
    RAISE EXCEPTION 'AD142: exterior no rechazado por la guarda estructural';
   END IF;
   rechazados:=rechazados+1;
  END;
 END LOOP;
 SELECT count(*) INTO despues FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF rechazados<>3 OR despues IS DISTINCT FROM antes THEN
  RAISE EXCEPTION 'AD142: exterior consumió o negativos incompletos';
 END IF;
END $exterior_cerrado$;

SELECT 'AD142-MERITOS-ACL-NEGATIVOS-OK';
