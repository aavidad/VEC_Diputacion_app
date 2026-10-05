\set ON_ERROR_STOP on
-- Prueba estructural AUT51: no publica Rol7 ni crea asignaciones; todo en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
DO $p$
DECLARE f text;src text;c jsonb;
BEGIN
 -- Las cuatro concesiones nuevas son exactamente las que exige AD177 y nada más.
 c:=vec_autorizacion.concesiones_plan_firma_admin_v1();
 IF jsonb_array_length(c)<>4
 OR (SELECT count(*) FROM jsonb_array_elements(c) x WHERE x->>'accion' IN('vec.catalogos.crear','vec.catalogos.actualizar','vec.catalogos.publicar','vec.catalogos.retirar')
     AND x->>'modulo_id'='contratacion_temporal' AND x->>'tipo_recurso'='catalogo_configurable'
     AND x->'finalidades'='["gestionar_contratacion_temporal"]' AND x->>'garantia_minima'='alto'
     AND x->'campos_permitidos'='[]' AND x->'obligaciones'='[]' AND (SELECT count(*) FROM jsonb_object_keys(x))=7)<>4
 THEN RAISE EXCEPTION 'AUT51: concesiones distintas de las de AD177: %',c; END IF;
 -- Funciones privadas: sólo el propietario; el operador sólo ejecuta la fachada.
 FOREACH f IN ARRAY ARRAY['vec_autorizacion.concesiones_plan_firma_admin_v1()','vec_autorizacion.exigir_operador_mantenimiento_plan_firma_admin_v1()',
  'vec_autorizacion.documento_asignacion_destino_mantenimiento_plan_firma_v1(jsonb,jsonb,text)','vec_autorizacion.preimagen_mantenimiento_plan_firma_admin_v1(jsonb)',
  'vec_autorizacion.aplicar_mantenimiento_plan_firma_admin_v1(text,text)'] LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f::regprocedure AND x.grantee<>p.proowner)
  OR (SELECT proowner FROM pg_proc WHERE oid=f::regprocedure)<>'vec_autorizacion_propietario'::regrole
  THEN RAISE EXCEPTION 'AUT51: ACL abierta o propietario distinto en %',f; END IF;
 END LOOP;
 IF (SELECT array_agg(x.grantee::regrole::text ORDER BY 1) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x
     WHERE p.oid='vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1(text,text)'::regprocedure AND x.grantee<>p.proowner)
    IS DISTINCT FROM ARRAY['vec_admin_mantenimiento_plan_firma_ejecutor']
 THEN RAISE EXCEPTION 'AUT51: la fachada no es exclusiva del grupo operador'; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_mantenimiento_plan_firma_ejecutor' AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 THEN RAISE EXCEPTION 'AUT51: grupo operador con atributos'; END IF;
 -- La puerta del lote ya no fija versión y exige la concesión exacta del lote.
 SELECT prosrc INTO src FROM pg_proc WHERE oid='vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)'::regprocedure;
 IF src ~ 'administracion_perfiles:v[0-9]' OR src ~ 'version=[0-9]' OR strpos(src,'concesiones_lote_ordinario_admin_v1()')=0
 OR strpos(src,'fuente_version=r.version')=0 OR strpos(src,'c=concesion_lote')=0
 THEN RAISE EXCEPTION 'AUT51: la puerta del lote sigue fijada o no exige la concesión exacta'; END IF;
 IF (SELECT array_agg(x.grantee::regrole::text ORDER BY 1) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) x
     WHERE p.oid='vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)'::regprocedure AND x.grantee<>p.proowner)
    IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario']
 THEN RAISE EXCEPTION 'AUT51: la ACL de la puerta del lote cambió'; END IF;
 -- Otra acción, otra versión o un rol ajeno: la puerta deniega antes de leer.
 IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:administracion_perfiles:v99','a','p','f','administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]','{}') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:otro:v7','a','p','f','administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]','{}') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:administracion_perfiles:v6','a','p','f','vec.catalogos.publicar','contratacion_temporal','catalogo_configurable','gestionar_contratacion_temporal','[]','{}') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT51: la puerta del lote acepta fuera de su concesión'; END IF;
 -- Sin operador configurado, la fachada no publica nada.
 BEGIN
  PERFORM vec_autorizacion.aplicar_mantenimiento_plan_firma_admin_v1('{}',repeat('0',64));
  RAISE EXCEPTION 'AUT51: aplicar sin operador';
 EXCEPTION WHEN insufficient_privilege OR invalid_transaction_state THEN NULL;
 END;
END $p$;
ROLLBACK;
SELECT 'AUT51-ESTRUCTURA-OK';
