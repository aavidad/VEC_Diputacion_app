\set ON_ERROR_STOP on
-- BORRADOR AD150. AD137 queda SIN USAR. Orden: después de AD149 real publicado.
-- Dirección captura estas dos huellas en el clon causal; no sustituir por dobles.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 esperada text:='202b1580f00e1618e0fb311dcf0911992e56d9eb5f17bc642d58c73768f360f1';
 original text; nuevo text; metadatos jsonb; acl aclitem[];
 marca text:=$x$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$x$;
 exclusion text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permiso_solicitar'$x$;
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 permiso text:=E'           )\n           OR (p_perfil_mutacion=''administracion_perfiles_acto''\n             AND pg_catalog.pg_has_role(session_user,''vec_admin_perfiles_ejecutor'',''MEMBER'')\n             AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_admin_perfiles_ejecutor''::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)\n             AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1\n             AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=''vec_admin_perfiles_ejecutor''::regrole)\n             AND NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.oid=session_user::regrole AND (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolbypassrls))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (p_perfil_mutacion='administracion_perfiles_acto'
             AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
             AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
             AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
             AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
             AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
             AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
             AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
             AND d->'campos_permitidos' IN ('[]'::jsonb,'["actos_disponibles","preimagen"]'::jsonb,'["capacidades"]'::jsonb,'["personas","siguiente_cursor"]'::jsonb,'["persona"]'::jsonb,'["roles"]'::jsonb,'["propuestas"]'::jsonb,'["propuesta"]'::jsonb,'["recibo"]'::jsonb)
             AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
             AND (
               (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.ordinario.v1'
                 AND d->>'accion' IN ('administracion.perfiles.otorgar','administracion.perfiles.revocar') AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='[]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.propuesta.v1'
                 AND d->>'accion'='administracion.perfiles.proponer' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='[]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.consulta.v1' AND d->>'accion'='administracion.perfiles.consultar' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='["actos_disponibles","preimagen"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.cierre.v1'
                 AND d->>'accion' IN ('administracion.perfiles.aprobar','administracion.perfiles.rechazar') AND d->>'tipo_recurso'='propuesta_perfil' AND d->'campos_permitidos'='[]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.capacidades.v1' AND d->>'accion'='administracion.perfiles.consultar' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='["capacidades"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1' AND d->>'accion'='administracion.perfiles.consultar' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='["personas","siguiente_cursor"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.consultar_persona.v1' AND d->>'accion'='administracion.perfiles.consultar' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='["persona"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.listar_roles.v1' AND d->>'accion'='administracion.perfiles.consultar' AND d->>'tipo_recurso'='perfil' AND d->'campos_permitidos'='["roles"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.listar_propuestas.v1' AND d->>'accion'='administracion.perfiles.historial.consultar' AND d->>'tipo_recurso'='historial_perfil' AND d->'campos_permitidos'='["propuestas"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.consultar_propuesta.v1' AND d->>'accion'='administracion.perfiles.historial.consultar' AND d->>'tipo_recurso'='historial_perfil' AND d->'campos_permitidos'='["propuesta"]'::jsonb)
               OR (c->>'audiencia_consumo'='vec_autorizacion.administracion_perfiles.lectura.consultar_recibo.v1' AND d->>'accion'='administracion.perfiles.recibo.consultar' AND d->>'tipo_recurso'='recibo_perfil' AND d->'campos_permitidos'='["recibo"]'::jsonb)
             ))
$x$;
BEGIN
 IF f IS NULL OR esperada !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'AD150: preimagen esperado=postAD149_real observado=pendiente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',proacl INTO STRICT original,metadatos,acl FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND(a.grantee<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR (length(original)-length(replace(original,marca,'')))<>length(marca)
 OR (length(original)-length(replace(original,exclusion,'')))<>length(exclusion)
 OR (length(original)-length(replace(original,guarda,'')))<>length(guarda)
 OR strpos(original,'administracion_perfiles_acto')<>0 THEN
  RAISE EXCEPTION 'AD150: contrato_nucleo esperado=exacto observado=divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 nuevo:=replace(nuevo,exclusion,exclusion||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''administracion_perfiles_acto''');
 nuevo:=replace(nuevo,guarda,permiso);
 EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE oid=f) IS DISTINCT FROM metadatos OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl THEN
  RAISE EXCEPTION 'AD150: postimagen esperado=parche_acotado observado=divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE original text; esperada text:='d5c8048786b283485016af29fba41ff68b93076ba4f37f2badfa6bb7d5532fd9';
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT original FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated;
 IF esperada !~ '^[0-9a-f]{64}$' OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada
 OR strpos(original,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(original,3)<>']))'
 OR strpos(original,'vec_autorizacion.administracion_perfiles.')<>0 THEN
  RAISE EXCEPTION 'AD150: audiencia esperado=postAD149_real observado=divergente' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||left(original,length(original)-3)||', ''vec_autorizacion.administracion_perfiles.ordinario.v1''::text, ''vec_autorizacion.administracion_perfiles.propuesta.v1''::text, ''vec_autorizacion.administracion_perfiles.cierre.v1''::text, ''vec_autorizacion.administracion_perfiles.consulta.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.capacidades.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.consultar_persona.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.listar_roles.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.listar_propuestas.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.consultar_propuesta.v1''::text, ''vec_autorizacion.administracion_perfiles.lectura.consultar_recibo.v1''::text]))';
END $audiencias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE d jsonb; x record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'AD150: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('administracion_perfiles_acto',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(d#>>'{vinculo_autenticacion_actor,autenticacion_ref}',d#>>'{vinculo_autenticacion_actor,sesion_ref}',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',d#>>'{vinculo_autenticacion_actor,cuenta_ordinaria_ref}',d->>'principal_id',d->>'perfil_activo_ref',d#>>'{vinculo_autenticacion_actor,politica_garantia_ref}',d#>>'{vinculo_autenticacion_actor,politica_garantia_huella_sha256}') IS NOT TRUE THEN RAISE EXCEPTION 'AD150: sesion de perfiles no vigente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,x.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;
COMMIT;
