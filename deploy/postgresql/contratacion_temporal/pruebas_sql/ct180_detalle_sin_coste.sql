\set ON_ERROR_STOP on
-- CT180: el detalle RRHH se abre cuando el análisis no tiene coste previsto.
-- Antes de CT180 falla con «detalle RRHH no disponible» (fallo SQL-1 del
-- recorrido del 05/10/2026: expediente 404 tras analizar una vacante sin fecha
-- de fin).
--
-- SÓLO PARA CLONES DE ENSAYO. Toma la última versión publicada de un
-- expediente sintético con análisis y coste, cambia su agregado dentro de la
-- transacción (los disparadores de inmutabilidad se omiten con
-- session_replication_role=replica, que exige superusuario) y termina en
-- ROLLBACK. Exige `SET vec.ensayo_clon = 'si'` antes de ejecutarla.
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';
DO $guarda$
BEGIN
 IF pg_catalog.current_setting('vec.ensayo_clon',true) IS DISTINCT FROM 'si'
    OR NOT (SELECT r.rolsuper FROM pg_catalog.pg_roles r WHERE r.rolname=current_user) THEN
  RAISE EXCEPTION 'CT180: prueba sólo para clones de ensayo';
 END IF;
END
$guarda$;
SET LOCAL session_replication_role=replica;
DO $prueba$
DECLARE
 v_exp text; v_version numeric; v_org text; v_corte numeric; v_original jsonb;
 v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
 v_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1;
 v_mat vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
 v_candidato record;
 v_variante jsonb;
 v_estado text;
 v_huella text;
 v_abierto jsonb:='{"inicio":"2027-01-01T00:00:00Z","causa_fin":"reincorporacion_titular"}';
BEGIN
 -- Expediente cuya versión publicada vigente tiene análisis con coste y se
 -- abre hoy sin errores.
 FOR v_candidato IN
  SELECT DISTINCT ON (p.expediente_ref)
         p.expediente_ref, p.version, p.organizacion_ref, p.corte_global, h.agregado_json
    FROM vec_contratacion_temporal.publicacion_version_rrhh p
    JOIN vec_contratacion_temporal.expediente_version_integral h
      ON h.expediente_ref=p.expediente_ref AND h.version=p.version
   ORDER BY p.expediente_ref, p.corte_global DESC
 LOOP
  CONTINUE WHEN pg_catalog.jsonb_typeof(v_candidato.agregado_json#>'{analisis,coste_previsto}')
      IS DISTINCT FROM 'object';
  v_alcance:=ROW(v_candidato.organizacion_ref,'organizacion',v_candidato.organizacion_ref);
  v_consulta:=ROW(v_candidato.expediente_ref,0);
  BEGIN
   SET LOCAL ROLE vec_contratacion_temporal_propietario;
   v_mat:=vec_contratacion_temporal.materializar_detalle_rrhh_v1(
     v_alcance,v_consulta,v_candidato.corte_global);
   RESET ROLE;
  EXCEPTION WHEN OTHERS THEN
   RESET ROLE;
   CONTINUE;
  END;
  v_exp:=v_candidato.expediente_ref; v_version:=v_candidato.version;
  v_org:=v_candidato.organizacion_ref; v_corte:=v_candidato.corte_global;
  v_original:=v_candidato.agregado_json;
  EXIT;
 END LOOP;
 IF v_exp IS NULL THEN
  RAISE EXCEPTION 'CT180: el clon no tiene un expediente con análisis y coste que se abra';
 END IF;
 IF (v_mat.detalle).analisis.coste_presente IS NOT TRUE THEN
  RAISE EXCEPTION 'CT180: el coste presente dejó de leerse';
 END IF;

 FOR v_variante IN SELECT * FROM pg_catalog.jsonb_array_elements(pg_catalog.jsonb_build_array(
   -- Positivo: análisis sin coste previsto, con fecha de fin.
   pg_catalog.jsonb_build_object('caso','sin_coste','abre',true,
     'agregado',v_original #- '{analisis,coste_previsto}' #- '{analisis,fuente_coste_ref}'),
   -- Positivo: vacante sin fecha de fin y sin coste (el caso del recorrido).
   pg_catalog.jsonb_build_object('caso','sin_coste_sin_fin','abre',true,
     'agregado',pg_catalog.jsonb_set(pg_catalog.jsonb_set(
       v_original #- '{analisis,coste_previsto}' #- '{analisis,fuente_coste_ref}',
       '{analisis,periodo}',v_abierto),'{solicitud,periodo}',v_abierto)),
   -- Negativo: coste presente sin fuente sigue rechazado.
   pg_catalog.jsonb_build_object('caso','coste_sin_fuente','abre',false,
     'agregado',v_original #- '{analisis,fuente_coste_ref}'),
   -- Negativo: coste presente a cero sigue rechazado.
   pg_catalog.jsonb_build_object('caso','coste_cero','abre',false,
     'agregado',pg_catalog.jsonb_set(v_original,'{analisis,coste_previsto,centimos}','0'::jsonb)),
   -- Negativo: fuente de coste sin coste sigue rechazada.
   pg_catalog.jsonb_build_object('caso','fuente_sin_coste','abre',false,
     'agregado',v_original #- '{analisis,coste_previsto}')
 )) LOOP
  -- La huella del agregado se recalcula en la versión y en su publicación
  -- para que la lectura llegue a la validación del contenido.
  v_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    (v_variante->'agregado')::text,'UTF8')),'hex');
  UPDATE vec_contratacion_temporal.expediente_version_integral
     SET agregado_json=v_variante->'agregado',agregado_json_huella_sha256=v_huella
   WHERE expediente_ref=v_exp AND version=v_version;
  UPDATE vec_contratacion_temporal.publicacion_version_rrhh
     SET agregado_huella_sha256=v_huella
   WHERE expediente_ref=v_exp AND version=v_version;
  v_estado:=NULL;
  BEGIN
   SET LOCAL ROLE vec_contratacion_temporal_propietario;
   v_mat:=vec_contratacion_temporal.materializar_detalle_rrhh_v1(
     ROW(v_org,'organizacion',v_org)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
     ROW(v_exp,0)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,v_corte);
   RESET ROLE;
   v_estado:='abre';
  EXCEPTION WHEN SQLSTATE '42501' THEN
   RESET ROLE;
   v_estado:='rechaza';
  END;
  IF (v_variante->>'abre')::boolean AND v_estado<>'abre' THEN
   RAISE EXCEPTION 'CT180: detalle rechazado en el caso %',v_variante->>'caso';
  END IF;
  IF NOT (v_variante->>'abre')::boolean AND v_estado<>'rechaza' THEN
   RAISE EXCEPTION 'CT180: detalle aceptado en el caso negativo %',v_variante->>'caso';
  END IF;
  IF v_estado='abre' AND (
      (v_mat.detalle).analisis.coste_presente IS DISTINCT FROM false
      OR (v_mat.detalle).analisis.coste_centimos IS DISTINCT FROM 0
      OR (v_mat.detalle).analisis.coste_moneda IS DISTINCT FROM ''
      OR (v_mat.detalle).analisis.fuente_coste_ref IS DISTINCT FROM '') THEN
   RAISE EXCEPTION 'CT180: «sin coste» mal materializado en el caso %',v_variante->>'caso';
  END IF;
  IF v_variante->>'caso'='sin_coste_sin_fin' AND (
      (v_mat.detalle).analisis.periodo_fin IS NOT NULL
      OR (v_mat.detalle).analisis.periodo_causa_fin IS DISTINCT FROM 'reincorporacion_titular') THEN
   RAISE EXCEPTION 'CT180: periodo abierto mal materializado';
  END IF;
 END LOOP;
 RAISE NOTICE 'CT180-PRUEBA-OK';
END
$prueba$;
ROLLBACK;
