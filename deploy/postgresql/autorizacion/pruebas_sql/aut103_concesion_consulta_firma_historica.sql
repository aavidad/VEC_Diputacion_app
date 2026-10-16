\set ON_ERROR_STOP on
BEGIN READ ONLY;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
DECLARE h oid:=to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(text)');
 r oid:=to_regprocedure('vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)');
 anterior jsonb;nuevo jsonb;esperados jsonb;cargo jsonb;plan jsonb;rol jsonb;
BEGIN
 IF h IS NULL OR r IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=h AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='77bb5555ee4d450a44b0ea6fa294b8e015216ca232d541160c66ee8457817e2d')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=r AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='d3057878ac315982a222628a33d9606732d78f663178d171e4d1434dd0de2ad7')
 THEN RAISE EXCEPTION 'AUT103: constructor o ACL divergentes' USING ERRCODE='55000'; END IF;
 FOR anterior,nuevo IN SELECT vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(op),
  vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(op)
  FROM unnest(ARRAY['consultar_detalle','consultar_cuadro','descargar_borrador',
   'consultar_r5','firmar_vec','consultar_circuito']) op LOOP
  IF anterior IS DISTINCT FROM nuevo THEN RAISE EXCEPTION 'AUT103: concesión anterior modificada' USING ERRCODE='55000'; END IF;
 END LOOP;
 anterior:=vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2('consultar_detalle');
 nuevo:=vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3('consultar_detalle_selectores');
 IF nuevo-'campos_permitidos' IS DISTINCT FROM anterior-'campos_permitidos'
  OR nuevo->'campos_permitidos' IS DISTINCT FROM '["selectores_recuperacion_firmas"]'::jsonb
 THEN RAISE EXCEPTION 'AUT103: campo adicional divergente' USING ERRCODE='55000'; END IF;
 nuevo:=vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3('recuperar_r5');
 SELECT jsonb_agg(valor ORDER BY valor COLLATE "C") INTO esperados
 FROM jsonb_array_elements_text(vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2('consultar_r5')->'campos_permitidos'
  || '["CanonNominal","CanonNominalRef","CanonNominalSHA256","MaterialRootSHA256"]'::jsonb) AS e(valor);
 IF nuevo->>'accion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.recuperar'
  OR nuevo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR nuevo->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
  OR nuevo->'finalidades' IS DISTINCT FROM '["gestionar_contratacion_temporal"]'::jsonb
  OR nuevo->>'garantia_minima' IS DISTINCT FROM 'sustancial'
  OR nuevo->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR jsonb_array_length(nuevo->'campos_permitidos') IS DISTINCT FROM 48
  OR nuevo->'campos_permitidos' IS DISTINCT FROM esperados
 THEN RAISE EXCEPTION 'AUT103: recuperación o 48 campos divergentes' USING ERRCODE='55000'; END IF;
 cargo:=jsonb_build_object('rol_id','ct_cargo_jefatura_firma_dev','version',2,
  'nombre','Jefatura de RRHH','version_anterior_sha256',repeat('a',64),
  'operaciones_v2','["consultar_r5","firmar_vec","recuperar_r5"]'::jsonb,
  'competencias','[{"accion":"contratacion_temporal.documento.firma_vec.registrar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}]'::jsonb,
  'version_rol_sha256',repeat('b',64),'regla_asignacion','lote_ordinario',
  'organizacion_ref','org:rrhh','vigente_desde','2026-10-16T00:00:00Z',
  'vigente_hasta','2027-10-16T00:00:00Z','duracion_propuesta_segundos',3600);
 plan:=jsonb_build_object('esquema','vec.admin.cargos-firma.plan.v2',
  'operacion_ref','rpa_cf_00000000000000000000000000000103','preparado_en','2026-10-16T00:00:00Z');
 rol:=vec_autorizacion.documento_rol_cargo_firma_v1(cargo,plan);
 IF jsonb_array_length(rol->'concesiones')<>4 OR rol#>>'{concesiones,2,accion}' IS DISTINCT FROM nuevo->>'accion'
 THEN RAISE EXCEPTION 'AUT103: rol firma no publica recuperación exacta' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_autorizacion.documento_rol_cargo_firma_v1(
   jsonb_set(jsonb_set(cargo,'{rol_id}','"ct_cargo_jefatura_consulta_dev"'),'{competencias}','[]'::jsonb),plan);
  RAISE EXCEPTION 'AUT103: cargo consulta admitió recuperación' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion.documento_rol_cargo_firma_v1(cargo,jsonb_set(plan,'{esquema}','"vec.admin.cargos-firma.plan.v1"'));
  RAISE EXCEPTION 'AUT103: plan alto admitió recuperación de desarrollo' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $prueba$;
ROLLBACK;
