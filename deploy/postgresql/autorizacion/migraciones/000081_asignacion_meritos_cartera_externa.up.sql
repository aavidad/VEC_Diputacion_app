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
     OR r.documento->'concesiones' IS DISTINCT FROM
      '[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]}]'::pg_catalog.jsonb
    ELSE true END)
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
$sustituto$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO original,meta FROM pg_proc p WHERE p.oid=f;
 IF original IS NULL OR
  (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT81: guard de rol sin marca única' USING ERRCODE='55000';END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR replace(pg_get_functiondef(f),sustituto,marca) IS DISTINCT FROM original
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
  IS DISTINCT FROM 'd628be56a728fe4ff39f61764eb7c24e543a39a75ea28678ff3cbfe7e9ec2f80'
 THEN RAISE EXCEPTION 'AUT81: modificación fuera del guard de rol' USING ERRCODE='55000';END IF;
END $parche$;
RESET ROLE;
COMMIT;
