\set ON_ERROR_STOP on
-- Solo en el clon efímero, tras AD143. La transacción siempre se descarta.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
DO $contratos$
DECLARE nombre text;f oid;permitido oid:='vec_administracion_copias_propietario'::regrole;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consumir_orden_copias_v3_atestada',
  'consumir_propuesta_restauracion_copias_v3_atestada','consumir_revision_restauracion_copias_v3_atestada'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
    AND p.prosecdef AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version',
     'p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz',
     'decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256',
     'auditoria_ref','consumida_en','consumo_nuevo','persona_ref'])
   THEN RAISE EXCEPTION 'AD143: firma de fachada incompatible'; END IF;
  IF (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR NOT has_function_privilege(permitido,f,'EXECUTE')
  OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,permitido) OR a.grantor<>p.proowner
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR has_function_privilege('vec_administracion_copias_ejecutor',f,'EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD143: fachada accesible fuera del propietario nominal'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY[
  'consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'consumir_copias_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre);
  IF f IS NULL OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable))
  OR has_function_privilege(permitido,f,'EXECUTE')
  OR has_function_privilege('vec_administracion_copias_ejecutor',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD143: núcleo o despachador privado accesible'; END IF;
 END LOOP;
 IF NOT has_schema_privilege(permitido,'vec_autorizacion_atestada_v3','USAGE')
 OR has_schema_privilege('vec_administracion_copias_ejecutor','vec_autorizacion_atestada_v3','USAGE')
 OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='vec_autorizacion_atestada_v3' AND c.relkind IN ('r','p','v','m','f')
  AND (has_table_privilege(permitido,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
    OR has_any_column_privilege(permitido,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 THEN RAISE EXCEPTION 'AD143: lectura o escritura directa del núcleo habilitada'; END IF;
 IF EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname IN (
  'vec_administracion_copias_propietario','vec_administracion_copias_ejecutor','vec_administracion_copias_migrador')
  AND (r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_administracion_copias_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'AD143: autoridad técnica abierta'; END IF;
END $contratos$;
DO $rechazos$
DECLARE nombre text;antes bigint;despues bigint;rechazos integer:=0;
BEGIN
 SELECT count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 FOREACH nombre IN ARRAY ARRAY['consumir_orden_copias_v3_atestada',
  'consumir_propuesta_restauracion_copias_v3_atestada','consumir_revision_restauracion_copias_v3_atestada'] LOOP
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING convert_to('{}','UTF8'),convert_to('{}','UTF8'),''::bytea,convert_to('{}','UTF8'),
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD143: contrato pendiente autorizó objetos vacíos';
  EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazos:=rechazos+1; END;
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING decode('ff','hex'),convert_to('{}','UTF8'),''::bytea,convert_to('{}','UTF8'),
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD143: UTF8 inválido autorizado';
  EXCEPTION WHEN invalid_parameter_value THEN rechazos:=rechazos+1; END;
 END LOOP;
 SELECT count(*) INTO despues FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF rechazos<>6 OR antes<>despues THEN RAISE EXCEPTION 'AD143: rechazo parcial o con efectos'; END IF;
END $rechazos$;
-- Las candidatas de E no son una publicación. Sistemas no puede revisar;
-- en las otras acciones el contrato pendiente mantiene la denegación.
-- El material sin atestación no demuestra un positivo criptográfico.
DO $perfiles_pendientes$
DECLARE nombre text;accion text;audiencia text;tipo text;finalidad text;campos jsonb;
 ctx jsonb;v jsonb;c jsonb;d jsonb;ctx_bytes bytea;antes bigint;despues bigint;rechazos integer:=0;
BEGIN
 SELECT count(*) INTO antes FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 ctx:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','persona_ref','persona:ad143:sintetica',
  'cuenta_ref','cuenta:ad143:sintetica','perfil_activo_ref','perfil:ad143:sintetico');
 ctx_bytes:=convert_to(ctx::text,'UTF8');
 v:=jsonb_build_object('superficie','administracion_privilegiada','cuenta_privilegiada',true,
  'garantia_observada','alto','cuenta_ref','cuenta:ad143:sintetica',
  'contexto_actor_huella_sha256',encode(sha256(ctx_bytes),'hex'));
 FOREACH nombre IN ARRAY ARRAY['consumir_orden_copias_v3_atestada',
  'consumir_propuesta_restauracion_copias_v3_atestada','consumir_revision_restauracion_copias_v3_atestada'] LOOP
  IF nombre='consumir_orden_copias_v3_atestada' THEN
   accion:='administracion.copias.orden.emitir';audiencia:='vec_administracion_copias.comprometer_orden.v1';
   tipo:='orden_copia';finalidad:='emitir_orden_copia';campos:='["orden","recibo"]'::jsonb;
  ELSIF nombre='consumir_propuesta_restauracion_copias_v3_atestada' THEN
   accion:='copias_restauracion_proponer';audiencia:='vec_administracion_copias.proponer_restauracion.v1';
   tipo:='propuesta_restauracion_copia';finalidad:='proponer_restauracion_copia';campos:='["propuesta","recibo"]'::jsonb;
  ELSE
   accion:='copias_restauracion_revisar';audiencia:='vec_administracion_copias.revisar_restauracion.v1';
   tipo:='propuesta_restauracion_copia';finalidad:='revisar_restauracion_copia';campos:='["propuesta","recibo","revision"]'::jsonb;
  END IF;
  c:=jsonb_build_object('audiencia_consumo',audiencia,'operacion',accion,
   'efecto_ref','efecto:ad143:sintetico','huella_efecto_sha256',repeat('a',64));
  d:=jsonb_build_object('accion',accion,'modulo_id','administracion','tipo_recurso',tipo,
   'finalidad',finalidad,'recurso_ref','efecto:ad143:sintetico',
   'contexto_recurso_huella_sha256',repeat('a',64),'campos_permitidos',campos,'obligaciones','[]'::jsonb,
   'version_rol_ref','rol:operador_plataforma:v1','perfil_activo_ref','perfil:ad143:sintetico',
   'vinculo_autenticacion_actor',v);
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,ctx_bytes,
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD143: candidato Sistemas aceptado sin publicación';
  EXCEPTION
   WHEN insufficient_privilege THEN
    IF nombre<>'consumir_revision_restauracion_copias_v3_atestada' THEN RAISE; END IF;
    rechazos:=rechazos+1;
   WHEN object_not_in_prerequisite_state THEN
    IF nombre='consumir_revision_restauracion_copias_v3_atestada' THEN RAISE; END IF;
    rechazos:=rechazos+1;
  END;
  d:=jsonb_set(d,'{version_rol_ref}','"rol:administracion_perfiles:v3"'::jsonb);
  BEGIN
   EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',nombre)
    USING convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,ctx_bytes,
     1::numeric,1::numeric,''::bytea,''::bytea,''::bytea,''::bytea;
   RAISE EXCEPTION 'AD143: candidato Aplicación aceptado sin publicación';
  EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazos:=rechazos+1; END;
 END LOOP;
 SELECT count(*) INTO despues FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF rechazos<>6 OR antes<>despues THEN RAISE EXCEPTION 'AD143: candidatas pendientes con efectos'; END IF;
END $perfiles_pendientes$;
ROLLBACK;
