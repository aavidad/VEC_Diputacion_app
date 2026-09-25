\set ON_ERROR_STOP on
-- AD3-000057: consumidores nominales de Cronos para resolver permisos y para
-- los avisos de resolución de la persona empleada: consultar la bandeja de
-- solicitudes pendientes de un paso, resolver (conceder o denegar) una
-- solicitud, consultar los avisos propios y archivar un aviso propio. Una
-- audiencia por acción. Número reservado para Cronos en el canal de
-- directores; se instala DESPUÉS de AD3-70, cuya postimagen del núcleo
-- amplía (sin tocar los contratos de AD3-53, AD3-70 ni los de Dietas).
-- Orden: AD3-53, AD3-70, esta migración y después cronos_v1 000009.
-- Instalación en serie con cualquier otra que reescriba el núcleo: toma el
-- consultivo común vec_autorizacion_atestada_v3:nucleo antes de leerlo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000057',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 -- Lista de perfiles tras AD3-70: aparece una vez en la guarda de sesión y
 -- otra en el bloque de contrato. Ambas se amplían con los cuatro nuevos.
 lista text:=$x$p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar')$x$;
 lista_nueva text:=$x$p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar','cronos_permisos_bandeja','cronos_permiso_resolver','cronos_avisos_propio','cronos_aviso_archivar')$x$;
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permiso_solicitar'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_permisos_bandeja''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_permiso_resolver''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_avisos_propio''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_aviso_archivar''';
 -- Cierre del último contrato de AD3-70 (solicitud de permiso propio).
 cierre text:=$x$    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_permiso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
 ))$x$;
 cierre_nuevo text:=$x$    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_permiso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permisos_bandeja'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permisos_bandeja.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permisos.bandeja.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_permisos'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_permisos'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["pendientes"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permiso_resolver'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permiso.resolver.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permiso.resolver'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_permiso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_permiso'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_avisos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.avisos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.avisos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'avisos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_avisos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["avisos"]'::jsonb)
   OR (p_perfil_mutacion='cronos_aviso_archivar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.aviso_propio.archivar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.aviso.propio.archivar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'archivo_aviso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'archivar_aviso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
 ))$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(text,text,text,text,text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-57: precondición incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR (length(original)-length(replace(original,lista,'')))<>2*length(lista)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,cierre,''))<>length(cierre)
    OR strpos(original,'cronos_permisos_bandeja')<>0 OR strpos(original,'cronos_permiso_resolver')<>0
    OR strpos(original,'cronos_avisos_propio')<>0 OR strpos(original,'cronos_aviso_archivar')<>0
 THEN RAISE EXCEPTION 'AD3-57: núcleo sin AD3-70 o incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,lista,lista_nueva); nuevo:=replace(nuevo,excl,excl_nuevo); nuevo:=replace(nuevo,cierre,cierre_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,cierre_nuevo,cierre),excl_nuevo,excl),lista_nueva,lista) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-57: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_cronos_v1.permiso_propio.solicitar.v1')=0
    OR strpos(d,'vec_cronos_v1.permisos_bandeja')<>0 OR strpos(d,'vec_cronos_v1.permiso.resolver')<>0
    OR strpos(d,'vec_cronos_v1.avisos_propio')<>0 OR strpos(d,'vec_cronos_v1.aviso_propio')<>0
 THEN RAISE EXCEPTION 'AD3-57: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_cronos_v1.permisos_bandeja.consultar.v1''::text, ''vec_cronos_v1.permiso.resolver.v1''::text, ''vec_cronos_v1.avisos_propio.consultar.v1''::text, ''vec_cronos_v1.aviso_propio.archivar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

-- Las cuatro fachadas reutilizan el consumidor nominal interno de AD3-70:
-- cada lectura o escritura consume una decisión nueva en la misma
-- transacción que la función de cronos_v1 que la invoca.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permisos_bandeja','vec_cronos_v1.permisos_bandeja.consultar.v1',
   'cronos.permisos.bandeja.consultar','bandeja_permisos','consultar_bandeja_permisos','["pendientes"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permiso_resolver','vec_cronos_v1.permiso.resolver.v1',
   'cronos.permiso.resolver','resolucion_permiso','resolver_permiso','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_avisos_propio','vec_cronos_v1.avisos_propio.consultar.v1',
   'cronos.avisos.propio.consultar','avisos_propio','consultar_avisos_propio','["avisos"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_aviso_archivar','vec_cronos_v1.aviso_propio.archivar.v1',
   'cronos.aviso.propio.archivar','archivo_aviso','archivar_aviso_propio','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

DO $acl$
DECLARE f regprocedure; x record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  EXECUTE format('ALTER FUNCTION %s OWNER TO vec_autorizacion_atestada_v3_propietario',f::text);
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_propietario',f::text);
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner AND a.grantee<>'vec_cronos_v1_propietario'::regrole LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_cronos_v1_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR (a.grantee='vec_cronos_v1_propietario'::regrole AND a.is_grantable))) THEN
   RAISE EXCEPTION 'AD3-57: ACL de fachada abierta' USING ERRCODE='55000';
  END IF;
 END LOOP;
 f:='vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(text,text,text,text,text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner) THEN
   RAISE EXCEPTION 'AD3-57: ACL interna abierta' USING ERRCODE='55000';
 END IF;
END $acl$;
COMMIT;
