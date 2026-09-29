\set ON_ERROR_STOP on
-- AD3-109: lectura del correo activo de «Mis correos» para el aviso de un
-- llamamiento de Bolsa. Instalar después de AD3-108 y antes de Usuarios
-- 000008. Una sola audiencia, interna: la lee el LOGIN ejecutor interno de
-- Usuarios mientras RRHH emite el llamamiento.
--
-- El permiso es el mismo que ya tiene quien emite el llamamiento (acción
-- llamamiento.emitir.v1 sobre la bolsa constituida, finalidad de gestión de
-- llamamientos). La capacidad lleva su propia audiencia y su propio perfil de
-- consumo: la capacidad de emisión de Bolsa no sirve aquí ni ésta sirve para
-- emitir. La bolsa, el llamamiento y la persona candidata quedan fijados en
-- la huella del material, que se audita con el consumo; que la candidata
-- pertenezca a ese llamamiento lo garantiza Bolsa antes de preguntar.
-- Quien puede emitir en una bolsa puede, por tanto, provocar esta lectura
-- para sus avisos. Sin DOWN tras historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000109',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-109: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Mismo parche acotado que AD3-107/108: el perfil nuevo entra en la
-- exclusión del bloque general, en las tres guardas de superficie de Usuarios
-- y en la extensión nominal, que sólo admite la superficie interna.
DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; excl_nuevo text; guarda_nueva text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''imagen_actualizar_usuarios''\n';
 guarda text:=E'p_perfil_mutacion IN (''preferencias_consulta_usuarios'',''preferencias_actualizacion_usuarios'',''correos_consultar_usuarios'',''correos_anadir_usuarios'',''correos_reenviar_usuarios'',''correos_verificar_usuarios'',''correos_activar_usuarios'',''correos_retirar_usuarios'',''imagen_consultar_usuarios'',''imagen_actualizar_usuarios'')';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correo_avisos_llamamiento_usuarios'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_llamamientos_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256')
$x$;
 meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(q)-'prosrc',q.proacl,q.proowner,q.proconfig,q.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc q WHERE q.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR (length(original)-length(replace(original,marca,'')))<>length(marca)
    OR (length(original)-length(replace(original,excl,'')))<>length(excl)
    OR (length(original)-length(replace(original,guarda,'')))<>3*length(guarda)
    OR strpos(original,'correo_avisos_llamamiento_usuarios')<>0
 THEN RAISE EXCEPTION 'AD3-109: núcleo incompatible' USING ERRCODE='55000'; END IF;
 excl_nuevo:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''correo_avisos_llamamiento_usuarios''\n';
 guarda_nueva:=left(guarda,length(guarda)-1)||',''correo_avisos_llamamiento_usuarios'')';
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,guarda,guarda_nueva);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),guarda_nueva,guarda),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE q.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-109: metadatos alterados' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; a text:='vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_usuarios.imagen.actualizar.externa_personal.v1')=0
    OR strpos(d,quote_literal(a))<>0
 THEN RAISE EXCEPTION 'AD3-109: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', '||quote_literal(a)||'::text';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

-- Fachada nominal. Sólo la invoca el propietario de Usuarios desde su
-- función de lectura, que además coteja material, bolsa y huella.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-109: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d #>> '{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'
    OR c->>'operacion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD3-109: lectura de correo denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'correo_avisos_llamamiento_usuarios',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-109: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_propietario;
DO $acl$ DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_function_privilege('vec_usuarios_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_usuarios_propietario'::regrole)))
 THEN RAISE EXCEPTION 'AD3-109: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
