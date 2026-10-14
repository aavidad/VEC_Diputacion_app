\set ON_ERROR_STOP on
-- AD268. Sucesora prospectiva de AD75, que no llegó a instalarse: la copia
-- vigente tiene las tres fachadas de AD59/AD80 con search_path endurecido a
-- pg_catalog, pg_temp. AD75 espera otra proconfig y se detiene antes de escribir.
-- Solo se ajustan las listas cerradas de campos y su comparación como en AD75:
-- la titular recibe las listas que coteja Dietas 000006, opcionalmente con
-- devolución de Dietas 000010/000011; el revisor conserva su lista base y
-- admite la devolución. Toda lista parcial o adicional sigue denegada.
-- Requiere AD59 y AD80. No crea acción, permiso, tabla ni efecto externo.
-- Guarda de definición SHA256, metadatos y dependencias; tres definiciones
-- finales explícitas, con postimagen SHA256 y los mismos invariantes. Una sola instalación.
-- Sin DOWN sobre historia. Dirección ensaya e instala con Dietas D9-D11.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000268',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $guard_pre$
DECLARE x record; f oid; p pg_proc%ROWTYPE;
 acl_cerrada text:='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_dietas_propietario=X/vec_autorizacion_atestada_v3_propietario}';
 hechas integer:=0;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD268: entorno incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('registrar_y_consumir_dietas_documento_v3_atestada','0ae0a5f64c1fd8a9ba6cd57ff86f2857','be3605e10950bfd386f44716873f2640fda6c2305e268c4438879f3857754602','25761c5d3043a77f683aa5eab3f35d17','7cae47bdaabccb07db5391971e5317e760650755d62dcdd9e18285dcb60496eb'),
  ('registrar_y_consumir_dietas_documento_consulta_v3_atestada','d0052e3fcaddbdc33e9b7f7d017a125e','7080ead4d594e041013d8e27d5ad60b5f8bbef0f6683db0acedf6198986355bf','500cb2bbcf38de7143ff81ac323e7659','1af5b1035daf3a59ed819232ac02e806efb31e5c3fe675efc1ca499424b8c2c1'),
  ('registrar_y_consumir_dietas_revisor_documento_v3_atestada','b0b8039b47e499237078fdd3477b295b','f0edfa8d3e70ed77993da388b234ab9a4f61a99f79e0b5b4339ab6e07ef31020','4ab4c204d9e14b6bf9749550cab4914a','b9d8f2bdb400bb178e0f4c00c0405ca5bae5096b6d197ad923ce8f592dec1a15')
 ) v(nombre,pre_md5,pre_sha,post_md5,post_sha) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD268: fachada ausente %',x.nombre USING ERRCODE='55000'; END IF;
  SELECT * INTO p FROM pg_proc WHERE oid=f;
  IF NOT FOUND
     OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM x.pre_sha
     OR md5(p.prosrc) IS DISTINCT FROM x.pre_md5
     OR p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR p.prosecdef IS NOT TRUE OR p.prokind IS DISTINCT FROM 'f'
     OR p.provolatile IS DISTINCT FROM 'v' OR p.proparallel IS DISTINCT FROM 'u'
     OR p.proisstrict IS NOT FALSE OR p.proleakproof IS NOT FALSE
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
     OR p.proacl::text IS DISTINCT FROM acl_cerrada
     OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR (SELECT count(*) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)<>2
     OR NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
        AND d.refclassid='pg_language'::regclass AND d.refobjid=p.prolang AND d.deptype='n')
     OR NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
        AND d.refclassid='pg_namespace'::regclass AND d.refobjid=p.pronamespace AND d.deptype='n')
  THEN RAISE EXCEPTION 'AD268: preimagen de % incompatible',x.nombre USING ERRCODE='55000'; END IF;
  hechas:=hechas+1;
 END LOOP;
 IF hechas<>3 THEN RAISE EXCEPTION 'AD268: preimagen incompleta' USING ERRCODE='55000'; END IF;
END $guard_pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE c jsonb; d jsonb; x record; verbo text;
 campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: material Dietas inválido' USING ERRCODE='22023'; END;
 verbo:=split_part(c->>'operacion','.',4);
 IF verbo IS NULL OR verbo NOT IN ('editar','borrar','enviar')
    OR c->>'operacion' IS DISTINCT FROM 'dietas.borrador.propio.'||verbo
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.borrador_propio.'||verbo||'.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'comision_borrador'
    OR d->>'finalidad' IS DISTINCT FROM verbo||'_borrador_propio'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false)
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: operación documento Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documento_dietas_mutacion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: documento Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_consulta_v3_atestada(p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE c jsonb; d jsonb; x record;
 campos jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.centro_ref","items.comision.codigos_ruta","items.comision.devolucion","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.unidad_ref","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.regla_huella_sha256","items.recibo.regla_ref","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: consulta Dietas inválida' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'dietas.documento.propio.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.documento_propio.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'comision_borrador'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_documento_propio_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false)
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: consulta documento Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documento_dietas_consulta',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: consulta Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE c jsonb; d jsonb; x record;
 campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
 campos_devolucion jsonb:='["comision.calculo","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-80: lectura del revisor Dietas inválida' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'dietas.circuito.documento.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.circuito.documento.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'documento_dietas'
    OR d->>'finalidad' IS DISTINCT FROM 'revisar_documento_circuito_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR NOT coalesce(d->'campos_permitidos'=campos OR d->'campos_permitidos'=campos_devolucion,false)
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-80: lectura del revisor Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'revisor_documento_dietas',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-80: lectura del revisor Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $function$;

DO $guard_post$
DECLARE x record; f oid; p pg_proc%ROWTYPE;
 acl_cerrada text:='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_dietas_propietario=X/vec_autorizacion_atestada_v3_propietario}';
 hechas integer:=0;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD268: entorno incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('registrar_y_consumir_dietas_documento_v3_atestada','0ae0a5f64c1fd8a9ba6cd57ff86f2857','be3605e10950bfd386f44716873f2640fda6c2305e268c4438879f3857754602','25761c5d3043a77f683aa5eab3f35d17','7cae47bdaabccb07db5391971e5317e760650755d62dcdd9e18285dcb60496eb'),
  ('registrar_y_consumir_dietas_documento_consulta_v3_atestada','d0052e3fcaddbdc33e9b7f7d017a125e','7080ead4d594e041013d8e27d5ad60b5f8bbef0f6683db0acedf6198986355bf','500cb2bbcf38de7143ff81ac323e7659','1af5b1035daf3a59ed819232ac02e806efb31e5c3fe675efc1ca499424b8c2c1'),
  ('registrar_y_consumir_dietas_revisor_documento_v3_atestada','b0b8039b47e499237078fdd3477b295b','f0edfa8d3e70ed77993da388b234ab9a4f61a99f79e0b5b4339ab6e07ef31020','4ab4c204d9e14b6bf9749550cab4914a','b9d8f2bdb400bb178e0f4c00c0405ca5bae5096b6d197ad923ce8f592dec1a15')
 ) v(nombre,pre_md5,pre_sha,post_md5,post_sha) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD268: fachada ausente %',x.nombre USING ERRCODE='55000'; END IF;
  SELECT * INTO p FROM pg_proc WHERE oid=f;
  IF NOT FOUND
     OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM x.post_sha
     OR md5(p.prosrc) IS DISTINCT FROM x.post_md5
     OR p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR p.prosecdef IS NOT TRUE OR p.prokind IS DISTINCT FROM 'f'
     OR p.provolatile IS DISTINCT FROM 'v' OR p.proparallel IS DISTINCT FROM 'u'
     OR p.proisstrict IS NOT FALSE OR p.proleakproof IS NOT FALSE
     OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
     OR p.proacl::text IS DISTINCT FROM acl_cerrada
     OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR (SELECT count(*) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)<>2
     OR NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
        AND d.refclassid='pg_language'::regclass AND d.refobjid=p.prolang AND d.deptype='n')
     OR NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
        AND d.refclassid='pg_namespace'::regclass AND d.refobjid=p.pronamespace AND d.deptype='n')
  THEN RAISE EXCEPTION 'AD268: postimagen de % incompatible',x.nombre USING ERRCODE='55000'; END IF;
  hechas:=hechas+1;
 END LOOP;
 IF hechas<>3 THEN RAISE EXCEPTION 'AD268: postimagen incompleta' USING ERRCODE='55000'; END IF;
END $guard_post$;

COMMIT;
