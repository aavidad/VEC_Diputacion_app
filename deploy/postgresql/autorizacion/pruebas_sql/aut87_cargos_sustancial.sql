\set ON_ERROR_STOP on
BEGIN;
DO $check$
DECLARE c jsonb;d jsonb;o text;
BEGIN
 FOREACH o IN ARRAY ARRAY['consultar_r5','firmar_vec'] LOOP
  IF vec_autorizacion.concesion_v2_cargo_firma_v1(o)->>'garantia_minima' IS DISTINCT FROM 'alto'
  OR vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(o)->>'garantia_minima' IS DISTINCT FROM 'sustancial' THEN
  RAISE EXCEPTION 'AUT87: PARO clave=garantia actual=divergente esperado=v1_alto_v2_sustancial'; END IF;
 END LOOP;
 IF vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2('administrar_perfiles') IS NOT NULL THEN
 RAISE EXCEPTION 'AUT87: PARO clave=op_desconocida actual=concedida esperado=NULL'; END IF;
 c:=jsonb_build_object('rol_id','ct_cargo_tecnico_documentos_dev','version',1,'nombre','Documentos de RRHH','version_anterior_sha256','',
 'operaciones_v2',jsonb_build_array('confirmar_original','custodiar_firmado','descargar_original','escribir_original','reservar_original'),
 'competencias','[]'::jsonb,'version_rol_sha256',repeat('a',64),'regla_asignacion','lote_ordinario',
 'organizacion_ref','organizacion:desarrollo:dipgra','vigente_desde','2026-10-12T00:00:00Z','vigente_hasta','2026-10-20T00:00:00Z','duracion_propuesta_segundos',86400);
 d:=vec_autorizacion.documento_rol_cargo_firma_v1(c,jsonb_build_object('esquema','vec.admin.cargos-firma.plan.v2','operacion_ref','rpa_cf_00000000000000000000000000000087','preparado_en','2026-10-13T00:00:00Z'));
 IF jsonb_array_length(d->'concesiones')<>5 OR EXISTS(SELECT 1 FROM jsonb_array_elements(d->'concesiones') x WHERE x->>'garantia_minima'<>'sustancial') THEN
 RAISE EXCEPTION 'AUT87: PARO clave=rol_documentos actual=divergente esperado=5_concesiones_sustanciales'; END IF;
 BEGIN
  PERFORM vec_autorizacion.documento_rol_cargo_firma_v1(jsonb_set(c,'{rol_id}','"ct_cargo_secretaria_documentos_dev"'),jsonb_build_object('esquema','vec.admin.cargos-firma.plan.v2'));
  RAISE EXCEPTION 'AUT87: PARO clave=preparar_original_secretaria actual=permitido esperado=22023';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid='vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(text)'::regprocedure AND a.grantee=0) THEN
 RAISE EXCEPTION 'AUT87: PARO clave=PUBLIC actual=permiso esperado=sin_permiso'; END IF;
END $check$;
ROLLBACK;
