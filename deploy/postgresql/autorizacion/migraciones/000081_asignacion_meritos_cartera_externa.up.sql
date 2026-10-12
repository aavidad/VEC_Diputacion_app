\set ON_ERROR_STOP on
-- AUT81: asignación externa M4 con rol gobernado AUT80 y provisión CA15.
-- Preimagen capturada del archivo HZ21 instalado post-AUT18/21.
-- No crea cuentas, roles, asignaciones ni historia al instalar. Sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000081',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;acl jsonb;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)');
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'grantee',r.rolname,'grantor',g.rolname,'privilege',a.privilege_type,'grantable',a.is_grantable)
  ORDER BY r.rolname,g.rolname,a.privilege_type,a.is_grantable),'[]'::jsonb) INTO acl
 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 LEFT JOIN pg_roles r ON r.oid=a.grantee LEFT JOIN pg_roles g ON g.oid=a.grantor;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR p.oid IS NULL OR p.proowner IS DISTINCT FROM to_regrole('vec_autorizacion_propietario')
 OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_language WHERE lanname='plpgsql')
 OR p.provolatile IS DISTINCT FROM 'v'::"char" OR p.prosecdef IS NOT TRUE
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 OR octet_length(p.prosrc)<>5996
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  'c830fc51eeb8e1a841ba33a176c6ded203bc9e8c2627c7d01c6a24fba48e17e1'
 OR acl IS DISTINCT FROM '[{"grantee":"vec_autorizacion_propietario","grantor":"vec_autorizacion_propietario","privilege":"EXECUTE","grantable":false},{"grantee":"vec_autorizacion_publicador_usuarios_externo","grantor":"vec_autorizacion_propietario","privilege":"EXECUTE","grantable":false}]'::jsonb
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=
  to_regprocedure('vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb)')) IS DISTINCT FROM
  '6a32d06608d62cb3caaaba1c15e3600b2ab998602154939821c29279a89d2fc8'
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=
  to_regprocedure('vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()')) IS DISTINCT FROM
  '0180a5752c84875defdc49235e5d4a9cca4d94fc702a9c301214b9cbdcd386a0'
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)') IS NULL
 THEN RAISE EXCEPTION 'AUT81: preimagen instalada divergente' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $parche$
DECLARE f oid:=to_regprocedure(
 'vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)');
 original text;nuevo text;meta jsonb;
 marca text:=$marca$
 IF NOT FOUND OR r.rol_id NOT IN ('candidato_usuarios_propios_desarrollo')
    OR vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS NOT TRUE
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
$marca$;
 sustituto text:=$sustituto$
 IF NOT FOUND OR (CASE r.rol_id
    WHEN 'candidato_usuarios_propios_desarrollo' THEN
     vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS NOT TRUE
    WHEN 'persona_meritos_cartera' THEN
     r.documento->>'rol_id' IS DISTINCT FROM 'persona_meritos_cartera'
     OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
     OR (r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb
      AND r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]},{"accion":"meritos.hecho.declarar","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["declaracion_hecho_propio"],"garantia_minima":"alto","campos_permitidos":["declarante_ref","hecho","recibo","version"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb)
    ELSE true END)
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
$sustituto$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL OR
  (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT81: guard de rol sin marca única' USING ERRCODE='55000';END IF;
 nuevo:=replace(original,marca,sustituto);
END $parche$;

CREATE OR REPLACE FUNCTION vec_autorizacion.publicar_asignacion_usuarios_externo_v1(p_documento_canonico bytea, p_huella_sha256 text, p_version_esperada bigint, p_huella_esperada text, p_actualizada_por text, p_acto_ref text, p_cuenta_ref text)
 RETURNS TABLE(asignacion_ref text, version bigint, huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE d jsonb; r record; anterior record; ahora timestamptz:=pg_catalog.clock_timestamp();
        id text; perfil text; principal text; rol text; nueva_version bigint; ref text;
BEGIN
 IF vec_autorizacion.publicador_usuarios_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_documento_canonico IS NULL OR pg_catalog.octet_length(p_documento_canonico) NOT BETWEEN 1 AND 65536
    OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_documento_canonico),'hex') IS DISTINCT FROM p_huella_sha256
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR (p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
    OR (p_version_esperada>0 AND p_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_cuenta_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN d:=pg_catalog.convert_from(p_documento_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'asignacion_id'; perfil:=d->>'perfil_activo_ref'; principal:=d->>'principal_id'; rol:=d->>'version_rol_ref';
 nueva_version:=p_version_esperada+1; ref:='asignacion:'||id||':v'||nueva_version::text;
 IF vec_autorizacion.texto_positivo_valido(id,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(perfil,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(principal,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(rol,512) IS NOT TRUE
    OR (d->>'version')::bigint IS DISTINCT FROM nueva_version
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(11+CASE WHEN d ? 'revocada_en' THEN 1 ELSE 0 END)
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d ?| ARRAY['revocada_por','revocacion_ref']
    OR (d ? 'revocada_en' AND d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR pg_catalog.jsonb_array_length(d->'ambitos')<>1
    OR pg_catalog.jsonb_typeof(d->'ambitos'->0) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d->'ambitos'->0))<>2
    OR NOT (d->'ambitos'->0) ?& ARRAY['clave','valores']
    OR d->'ambitos'->0->>'clave' IS DISTINCT FROM 'persona_ref'
    OR pg_catalog.jsonb_array_length(d->'ambitos'->0->'valores')<>1
    OR vec_autorizacion.texto_positivo_valido(d->'ambitos'->0->'valores'->>0,512) IS NOT TRUE
    OR d->'ambitos'->0->'valores'->>0 IS DISTINCT FROM principal
    OR d->>'emitida_por' IS DISTINCT FROM p_actualizada_por
    OR d->>'vigente_desde' IS NULL OR d->>'vigente_hasta' IS NULL
    OR (d->>'vigente_desde')::timestamptz>ahora
    OR (d->>'vigente_hasta')::timestamptz<=ahora
    OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(p_cuenta_ref,principal,perfil) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación externa inválida o sin provisión' USING ERRCODE='42501'; END IF;
 SELECT vr.rol_id,vr.huella_sha256,vr.documento INTO r FROM vec_autorizacion.version_rol vr
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=vr.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE vr.version_rol_ref=rol AND c.estado='habilitada' FOR SHARE OF ca;
 IF NOT FOUND OR (CASE r.rol_id
    WHEN 'candidato_usuarios_propios_desarrollo' THEN
     vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS NOT TRUE
    WHEN 'persona_meritos_cartera' THEN
     r.documento->>'rol_id' IS DISTINCT FROM 'persona_meritos_cartera'
     OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
     OR (r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb
      AND r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]},{"accion":"meritos.hecho.declarar","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["declaracion_hecho_propio"],"garantia_minima":"alto","campos_permitidos":["declarante_ref","hecho","recibo","version"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb)
    ELSE true END)
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:asignacion_usuarios_externo:'||perfil,0));
 SELECT a.asignacion_ref,a.asignacion_id,a.version,a.principal_id,a.huella_sha256
 INTO anterior FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.asignacion_ref=aa.asignacion_ref
 WHERE aa.perfil_activo_ref=perfil FOR UPDATE OF aa;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
      OR anterior.asignacion_id IS DISTINCT FROM id OR anterior.principal_id IS DISTINCT FROM principal))
 THEN RAISE EXCEPTION 'asignación externa: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,nueva_version,perfil,principal,rol,p_huella_sha256,(d->>'emitida_en')::timestamptz,d);
 IF p_version_esperada=0 THEN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES(perfil,ref,ahora,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.asignacion_perfil_actual_externa AS puntero
   SET asignacion_ref=ref,actualizada_en=ahora,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.perfil_activo_ref=perfil AND puntero.asignacion_ref=anterior.asignacion_ref;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación externa: puntero cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('asignacion',p_acto_ref,ref,nueva_version,p_huella_esperada,p_huella_sha256,p_actualizada_por);
 RETURN QUERY SELECT ref,nueva_version,p_huella_sha256;
END $function$;

DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM 'ed39739ff168213b1cfd896491fcbf1ea5f8dc02d93d81d56c7843bc0b2d5eb9'
 THEN RAISE EXCEPTION 'AUT81: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;

RESET ROLE;
COMMIT;
