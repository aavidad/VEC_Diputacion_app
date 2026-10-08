\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 f_presentar oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 f_revisar oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 f_lectura oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)');
 f_contrato oid:=to_regprocedure('vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(text,text,text,text)');
 f_login oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_lector_inscripciones_valido_v1(text,text)');
 r record; n bigint; h text:=repeat('a',64); rol text;
BEGIN
 IF f_presentar IS NULL OR f_revisar IS NULL OR f_lectura IS NULL OR f_contrato IS NULL OR f_login IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_ratificacion_catalogo_admin_v1(jsonb)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND a.attname='ratificacion_catalogo_detalle' AND a.atttypid='jsonb'::regtype
   AND NOT a.attisdropped)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'ratificacion_catalogo_admin')>0
   AND strpos(pg_get_constraintdef(c.oid,false),'intento_ratificacion_catalogo_admin')>0
   AND strpos(pg_get_constraintdef(c.oid,false),'lectura_inscripcion')>0)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid IN(f_presentar,f_revisar,f_lectura,f_contrato,f_login) AND a.grantee=0)
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_presentar,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_revisar,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f_lectura,'EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND t.tgname='encolar_sellado_ad207' AND t.tgenabled='O')
 THEN RAISE EXCEPTION 'AD228: ACL, RLS o cola v5 divergente';END IF;
 FOREACH rol IN ARRAY ARRAY['vec_bolsa_llamamientos_lector_inscripciones',
  'vec_bolsa_llamamientos_lector_inscripciones_empleado',
  'vec_bolsa_llamamientos_lector_inscripciones_rrhh'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=rol
     AND NOT(rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
  OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=rol::regrole)
  OR has_function_privilege(rol,f_lectura,'EXECUTE')
  THEN RAISE EXCEPTION 'AD228: rol lector divergente %',rol;END IF;
 END LOOP;

 FOR r IN SELECT * FROM (VALUES
  ('bolsa.inscripcion.convocatorias.listar','consulta_convocatoria_abierta','externa_personal','inscripciones_abiertas_'||h),
  ('bolsa.inscripcion.convocatoria.consultar','consulta_convocatoria_abierta','externa_personal','cv1_prueba_v1'),
  ('bolsa.inscripcion.propias.listar','consulta_inscripcion_propia','externa_personal','inscripciones_propias_'||h),
  ('bolsa.inscripcion.propia.consultar','consulta_inscripcion_propia','externa_personal','solicitud_inscripcion_'||h),
  ('bolsa.inscripcion.rrhh.listar','consulta_inscripcion_rrhh','interna_corporativa','inscripciones_rrhh_'||h),
  ('bolsa.inscripcion.rrhh.consultar','consulta_inscripcion_rrhh','interna_corporativa','solicitud_inscripcion_'||h),
  ('bolsa.inscripcion.rrhh.motivos','consulta_motivos_inscripcion_rrhh','interna_corporativa','motivos_inscripcion_'||h))
  x(accion,finalidad,canal,recurso) LOOP
  IF vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(r.accion,r.finalidad,r.canal,r.recurso) IS NOT TRUE
  OR vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(r.accion,r.finalidad,'administracion_privilegiada',r.recurso) IS NOT FALSE
  OR vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(r.accion,'otra_finalidad',r.canal,r.recurso) IS NOT FALSE
  OR vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(r.accion,r.finalidad,r.canal,'otro_'||h) IS NOT FALSE
  THEN RAISE EXCEPTION 'AD228: contrato de lectura divergente %',r.accion;END IF;
  IF r.accion IN ('bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
                  'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar') THEN
   IF vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
        r.accion,r.finalidad,'interna_corporativa',r.recurso) IS NOT TRUE
   THEN RAISE EXCEPTION 'AD228: autoservicio interno denegado %',r.accion;END IF;
  ELSE
   IF vec_autorizacion_atestada_v3.contrato_lectura_inscripcion_v1(
        r.accion,r.finalidad,'externa_personal',r.recurso) IS NOT FALSE
   THEN RAISE EXCEPTION 'AD228: RRHH exterior admitido %',r.accion;END IF;
  END IF;
 END LOOP;

 SELECT count(*) INTO n FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(NULL,NULL,NULL);
  RAISE EXCEPTION 'AD228: emisor aceptó LOGIN sin permiso';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD228: presentación aceptó LOGIN ajeno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_revision_inscripcion_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD228: revisión aceptó LOGIN ajeno';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;END;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)<>n
 THEN RAISE EXCEPTION 'AD228: denegación produjo auditoría';END IF;
END $prueba$;
ROLLBACK;
