\set ON_ERROR_STOP on
-- Solo clon desechable H1+H3+H4+CA19+AUT22+AUT23. Sin COMMIT.
-- Esta focal prueba la separación de personas y los umbrales declarados.
-- AUT24 debe probar el efecto real: bootstrap con dos personas; baja 2→1
-- autorizada por B proponente y A aprobador; 1→0 denegada; con una persona
-- las nuevas operaciones sensibles esperan recuperación excepcional.
BEGIN;
DO $prueba$
DECLARE r record; a record; v record; documento jsonb; ref text;
        bloqueado boolean; mensaje text; revision_actual bigint;
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion.clase_perfil_sensible)<>2
    OR (SELECT count(*) FROM vec_autorizacion.operacion_perfil_sensible)<>4
    OR (SELECT count(*) FROM vec_autorizacion.rol_sensible_exacto)<>1
    OR (SELECT count(*) FROM vec_autorizacion.propuesta_perfil_sensible)<>0
    OR (SELECT count(*) FROM vec_autorizacion.acto_perfil_sensible)<>0
    OR (SELECT count(*) FROM vec_autorizacion.recibo_perfil_sensible)<>0
    OR (SELECT count(*) FROM vec_autorizacion.control_continuidad_admin
        WHERE control_id AND bootstrap_minimo_personas=2 AND minimo_personas=1
          AND bootstrap_estado='pendiente'
          AND bootstrap_acto_ref IS NULL)<>1
 THEN RAISE EXCEPTION 'catálogo o historia inicial incompatibles'; END IF;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol
 WHERE version_rol_ref='rol:administracion_perfiles:v2';
 IF pg_catalog.jsonb_array_length(r.documento->'concesiones')<>8
    OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
               WHERE c->>'modulo_id'<>'administracion' OR c->>'garantia_minima'<>'alto')
    OR EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil
               WHERE version_rol_ref=r.version_rol_ref)
    OR pg_catalog.has_function_privilege('vec_autorizacion_fuente',
       'vec_autorizacion.avanzar_continuidad_admin_interna_v1(bigint)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_propietario',
       'vec_autorizacion.avanzar_continuidad_admin_interna_v1(bigint)','EXECUTE')
 THEN RAISE EXCEPTION 'rol v2 o ACL abierto'; END IF;

 -- El publicador antiguo no puede asignar ni siquiera un perfil sintético nuevo.
 SELECT aa.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual aa LIMIT 1;
 SELECT * INTO STRICT v FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=a.asignacion_ref;
 documento:=v.documento;
 documento:=pg_catalog.jsonb_set(documento,'{asignacion_id}','"admin_aut23_sintetico"'::jsonb);
 documento:=pg_catalog.jsonb_set(documento,'{version}','1'::jsonb);
 documento:=pg_catalog.jsonb_set(documento,'{perfil_activo_ref}','"prf_aut23_sintetico"'::jsonb);
 documento:=pg_catalog.jsonb_set(documento,'{principal_id}','"actor_aut23_sintetico"'::jsonb);
 documento:=pg_catalog.jsonb_set(documento,'{version_rol_ref}',pg_catalog.to_jsonb(r.version_rol_ref));
 INSERT INTO vec_autorizacion.asignacion_perfil
 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES('asignacion:admin_aut23_sintetico:v1','admin_aut23_sintetico',1,
        'prf_aut23_sintetico','actor_aut23_sintetico',r.version_rol_ref,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento::text,'UTF8')),'hex'),v.emitida_en,documento);
 bloqueado:=false;
 BEGIN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual
  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES('prf_aut23_sintetico','asignacion:admin_aut23_sintetico:v1',pg_catalog.clock_timestamp(),
         'actor_aut23_sintetico','acto_aut23_sintetico');
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  bloqueado:=mensaje='asignación sensible cerrada hasta AUT24';
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'publicador genérico asignó ADMIN'; END IF;

 -- El catálogo Intervención no se puebla con un rol inventado: las
 -- asignaciones institucionales existentes siguen actualizables por su dueño.
 SELECT aa.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual aa
 JOIN vec_autorizacion.asignacion_perfil x ON x.asignacion_ref=aa.asignacion_ref
 JOIN vec_autorizacion.version_rol rr ON rr.version_rol_ref=x.version_rol_ref
 WHERE rr.rol_id LIKE '%intervencion%' LIMIT 1;
 SELECT * INTO STRICT v FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=a.asignacion_ref;
 SELECT 'asignacion:'||v.asignacion_id||':v'||(max(version)+1) INTO ref
 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id=v.asignacion_id;
 documento:=pg_catalog.jsonb_set(v.documento,'{version}',pg_catalog.to_jsonb(v.version+1));
 INSERT INTO vec_autorizacion.asignacion_perfil
 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,v.asignacion_id,v.version+1,v.perfil_activo_ref,v.principal_id,v.version_rol_ref,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento::text,'UTF8')),'hex'),v.emitida_en,documento);
 UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=ref
 WHERE perfil_activo_ref=a.perfil_activo_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'Intervención quedó bloqueada'; END IF;

 -- B propone su propia baja; B no puede autoaprobarse, A sí puede
 -- aprobarla. No se aplica efecto: la población 2→1/1→0 queda para AUT24.
 SELECT revision INTO STRICT revision_actual FROM vec_autorizacion.control_continuidad_admin WHERE control_id=true;
 INSERT INTO vec_autorizacion.propuesta_perfil_sensible
 (propuesta_ref,clase,operacion,version_rol_ref,objetivo_persona_ref,objetivo_cuenta_ref,
  objetivo_perfil_ref,proponente_persona_ref,proponente_cuenta_ref,proponente_perfil_ref,
  motivo_codigo,revision_continuidad_esperada,preimagen_huella_sha256,
  documento_canonico,huella_sha256,creada_en,caduca_en)
 VALUES('propuesta_admin:'||pg_catalog.repeat('a',32),'administrador','revocar',r.version_rol_ref,
  'per_aut23_proponente','cta_aut23_proponente','prf_aut23_proponente',
  'per_aut23_proponente','cta_aut23_proponente','prf_aut23_proponente','motivo_aut23',
  revision_actual,pg_catalog.repeat('a',64),pg_catalog.convert_to('{}','UTF8'),
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex'),
  pg_catalog.clock_timestamp(),pg_catalog.clock_timestamp()+interval '1 hour');
 bloqueado:=false;
 BEGIN
  INSERT INTO vec_autorizacion.cierre_propuesta_perfil_sensible
  (propuesta_ref,cierre_ref,resultado,aprobador_persona_ref,aprobador_cuenta_ref,
   aprobador_perfil_ref,motivo_codigo,revision_continuidad_observada,
   documento_canonico,huella_sha256,cerrada_en)
  VALUES('propuesta_admin:'||pg_catalog.repeat('a',32),'cierre_admin:'||pg_catalog.repeat('b',32),
   'aprobada','per_aut23_proponente','cta_aut23_otro','prf_aut23_otro','motivo_aut23',
   revision_actual,pg_catalog.convert_to('{}','UTF8'),
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex'),
   pg_catalog.clock_timestamp());
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  bloqueado:=mensaje='aprobación sensible no independiente o caducada';
 END;
 IF NOT bloqueado THEN RAISE EXCEPTION 'autoaprobación aceptada'; END IF;
 INSERT INTO vec_autorizacion.cierre_propuesta_perfil_sensible
 (propuesta_ref,cierre_ref,resultado,aprobador_persona_ref,aprobador_cuenta_ref,
  aprobador_perfil_ref,motivo_codigo,revision_continuidad_observada,
  documento_canonico,huella_sha256,cerrada_en)
 VALUES('propuesta_admin:'||pg_catalog.repeat('a',32),'cierre_admin:'||pg_catalog.repeat('c',32),
  'aprobada','per_aut23_aprobador','cta_aut23_aprobador','prf_aut23_aprobador','motivo_aut23',
  revision_actual,pg_catalog.convert_to('{}','UTF8'),
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex'),
  pg_catalog.clock_timestamp());
 IF (SELECT count(*) FROM vec_autorizacion.cierre_propuesta_perfil_sensible)<>1
    OR (SELECT count(*) FROM vec_autorizacion.acto_perfil_sensible)<>0
 THEN RAISE EXCEPTION 'cierre duplicado o efecto anticipado'; END IF;
END $prueba$;
ROLLBACK;
