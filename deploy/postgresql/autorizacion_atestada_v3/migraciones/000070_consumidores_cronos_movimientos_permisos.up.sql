\set ON_ERROR_STOP on
-- AD3-000070: consumidores nominales de Cronos para la persona empleada,
-- segundo corte: consultar sus movimientos (calendario, absentismos y
-- solicitudes de corrección), solicitar la corrección de un olvido de
-- marcaje, consultar sus permisos del año y solicitar un permiso. Una
-- audiencia por acción. Requiere AD3-53 instalada: extiende la guarda de
-- sesión del ejecutor de Cronos y su bloque de contrato, sin tocar los de
-- AD3-53 ni los de AD3-59. Se numera 000070 para dejar libres 000060–000069.
-- Orden: AD3-53, esta migración y después cronos_v1 000008. Instalación en
-- serie con cualquier otra que reescriba el núcleo: toma el mismo consultivo
-- común vec_autorizacion_atestada_v3:nucleo antes de leer su preimagen.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000070',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 -- Lista de perfiles de AD3-53: aparece una vez en la guarda de sesión y otra
 -- en el bloque de contrato. Ambas se amplían con los cuatro perfiles nuevos.
 lista text:=$x$p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio')$x$;
 lista_nueva text:=$x$p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar')$x$;
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_saldo_propio'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_movimientos_propio''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_correccion_solicitar''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_permisos_propio''\n               AND p_perfil_mutacion IS DISTINCT FROM ''cronos_permiso_solicitar''';
 -- Cierre del último contrato de AD3-53 (saldo propio); se añaden detrás.
 cierre text:=E'    AND d->''campos_permitidos'' IS NOT DISTINCT FROM ''["detalle","periodo","resumen"]''::jsonb)\n ))';
 cierre_nuevo text:=$x$    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["detalle","periodo","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_movimientos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.movimientos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.movimientos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'movimientos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_movimientos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["absentismos","calendario","correcciones","periodo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion_propia.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correccion_marcaje'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_correccion_marcaje'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permisos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permisos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permisos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'permisos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_permisos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","pendientes","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permiso_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permiso_propio.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permiso.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_permiso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_permiso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
 ))$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_saldo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_movimientos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_correccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_permisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-70: precondición incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR (length(original)-length(replace(original,lista,'')))<>2*length(lista)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,cierre,''))<>length(cierre)
    OR strpos(original,'cronos_movimientos_propio')<>0 OR strpos(original,'cronos_permiso')<>0 OR strpos(original,'cronos_correccion')<>0
 THEN RAISE EXCEPTION 'AD3-70: núcleo sin AD3-53 o incompatible' USING ERRCODE='55000'; END IF;
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
 THEN RAISE EXCEPTION 'AD3-70: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_cronos_v1.saldo_propio.consultar.v1')=0
    OR strpos(d,'vec_cronos_v1.movimientos_propio')<>0 OR strpos(d,'vec_cronos_v1.correccion_propia')<>0
    OR strpos(d,'vec_cronos_v1.permisos_propio')<>0 OR strpos(d,'vec_cronos_v1.permiso_propio')<>0
 THEN RAISE EXCEPTION 'AD3-70: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_cronos_v1.movimientos_propio.consultar.v1''::text, ''vec_cronos_v1.correccion_propia.solicitar.v1''::text, ''vec_cronos_v1.permisos_propio.consultar.v1''::text, ''vec_cronos_v1.permiso_propio.solicitar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

-- Las cuatro fachadas repiten el sello que exige el núcleo y rechazan el
-- replay de una capacidad: cada lectura o escritura consume una decisión
-- nueva en la misma transacción que la función de cronos_v1 que la invoca.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(p_perfil text,p_audiencia text,p_accion text,p_tipo text,p_finalidad text,p_campos jsonb,
  p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-70: material Cronos inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM p_audiencia OR c->>'operacion' IS DISTINCT FROM p_accion
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'cronos' OR d->>'tipo_recurso' IS DISTINCT FROM p_tipo
    OR d->>'finalidad' IS DISTINCT FROM p_finalidad OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM p_campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-70: operación Cronos rechazada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-70: operación Cronos requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_movimientos_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_movimientos_propio','vec_cronos_v1.movimientos_propio.consultar.v1',
   'cronos.movimientos.propio.consultar','movimientos_propio','consultar_movimientos_propio','["absentismos","calendario","correcciones","periodo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_correccion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_solicitar','vec_cronos_v1.correccion_propia.solicitar.v1',
   'cronos.correccion.solicitar','correccion_marcaje','solicitar_correccion_marcaje','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_permisos_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permisos_propio','vec_cronos_v1.permisos_propio.consultar.v1',
   'cronos.permisos.propio.consultar','permisos_propio','consultar_permisos_propio','["catalogo","pendientes","resumen"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_permiso_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permiso_solicitar','vec_cronos_v1.permiso_propio.solicitar.v1',
   'cronos.permiso.solicitar','solicitud_permiso','solicitar_permiso_propio','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_cronos_v1_propietario;
DO $acl$
DECLARE f regprocedure; x record;
BEGIN
 f:='vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(text,text,text,text,text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 EXECUTE format('ALTER FUNCTION %s OWNER TO vec_autorizacion_atestada_v3_propietario',f::text);
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
 IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner) THEN
   RAISE EXCEPTION 'AD3-70: ACL interna abierta' USING ERRCODE='55000';
 END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.consumir_cronos_movimientos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_correccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.consumir_cronos_permisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  EXECUTE format('ALTER FUNCTION %s OWNER TO vec_autorizacion_atestada_v3_propietario',f::text);
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_propietario',f::text);
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner AND a.grantee<>'vec_cronos_v1_propietario'::regrole LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_cronos_v1_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR (a.grantee='vec_cronos_v1_propietario'::regrole AND a.is_grantable))) THEN
   RAISE EXCEPTION 'AD3-70: ACL de fachada abierta' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;
COMMIT;
