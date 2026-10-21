\set ON_ERROR_STOP on
-- CT262: la propuesta v7 deja de servir a los documentos tras una no
-- incorporación efectiva o una sustitución. Se ejecuta como superusuario en
-- una copia desechable: replica desactiva solo en esta transacción las claves
-- ajenas y los disparadores de historia para insertar una sustitución
-- sintética. Termina siempre en ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
SET LOCAL session_replication_role=replica;
INSERT INTO vec_contratacion_temporal.propuesta_sustitucion_v1(
 propuesta_ref,propuesta_anterior_ref,no_incorporacion_recibo_ref,
 organizacion_ref,expediente_ref,ronda,registrada_en)
VALUES ('propuesta:ct262:sucesora','propuesta:ct262:sustituida','recibo:ct262:no-incorporacion',
 'organizacion:ct262:prueba','expediente:ct262:prueba',2,now());
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $prueba$
DECLARE
 base jsonb:=jsonb_build_object('actuaciones',jsonb_build_array(
  jsonb_build_object('accion_clave','registrar_propuesta_formalizacion'),
  jsonb_build_object('accion_clave','registrar_resolucion_formalizacion')));
 no_incorporada jsonb;
BEGIN
 no_incorporada:=jsonb_set(base,'{actuaciones}',(base->'actuaciones')||jsonb_build_array(
  jsonb_build_object('accion_clave','contratacion_temporal.incorporacion.no_incorporacion')));
 IF vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:vigente',base) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT262: propuesta vigente rechazada' USING ERRCODE='55000';
 END IF;
 -- Una propuesta de no incorporación pendiente no retira la aceptación.
 IF vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:vigente',
   jsonb_set(base,'{actuaciones}',(base->'actuaciones')||jsonb_build_array(
    jsonb_build_object('accion_clave','contratacion_temporal.incorporacion.no_incorporacion_propuesta')))) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT262: propuesta de no incorporación tratada como efectiva' USING ERRCODE='55000';
 END IF;
 IF vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:vigente',no_incorporada) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT262: no incorporación efectiva admitida' USING ERRCODE='55000';
 END IF;
 IF vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:sustituida',base) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT262: propuesta sustituida admitida' USING ERRCODE='55000';
 END IF;
 IF vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:vigente','{}'::jsonb) IS NOT FALSE
  OR vec_contratacion_temporal.propuesta_documental_vigente_ct262(NULL,base) IS NOT FALSE
  OR vec_contratacion_temporal.propuesta_documental_vigente_ct262('propuesta:ct262:vigente',NULL) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT262: agregado o propuesta ausentes admitidos' USING ERRCODE='55000';
 END IF;
END $prueba$;
ROLLBACK;
