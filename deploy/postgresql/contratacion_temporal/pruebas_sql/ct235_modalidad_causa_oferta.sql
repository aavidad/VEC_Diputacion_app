\set ON_ERROR_STOP on
-- Ejecutar después de CT235 en copia desechable. El efecto firmado se verifica en el recorrido.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE i jsonb;c jsonb;clave text;v boolean;
BEGIN
 i:=jsonb_build_object('organizacion_ref','org:granada','expediente_ref','expediente:ct235',
  'version_expediente',3,'clave_idempotencia','11111111-1111-4111-8111-111111111111',
  'actor_ref','per_1234567890123456789012','perfil_ref','perfil:rrhh',
  'categoria_ref','categoria:administracion','centro_ref','centro:granada');
 clave:='ct-oferta:'||encode(sha256(convert_to(
  (i->>'organizacion_ref')||chr(31)||(i->>'expediente_ref')||chr(31)||
  (i->>'version_expediente')||chr(31)||(i->>'clave_idempotencia'),'UTF8')),'hex');
 c:=jsonb_build_object('organizacion_ref',i->>'organizacion_ref',
  'expediente_ref',i->>'expediente_ref','version_expediente',3,
  'actor_ref',i->>'actor_ref','bolsa_ref','bolsa:administracion',
  'necesidad_ref','necesidad:administracion','numero_plazas',2,
  'numero_elegibles',2,'clave_bolsa',clave,'inicio_plazo','publicacion',
  'datos',jsonb_build_object('categoria','Administracion','centro','Granada',
   'fecha_inicio','2026-10-20','descripcion','Cobertura temporal'),
  'origen_ct',jsonb_build_object('organizacion_ref',i->>'organizacion_ref',
   'expediente_ref',i->>'expediente_ref','version_expediente',3,
   'clave_ct',i->>'clave_idempotencia','bolsa_ref','bolsa:administracion',
   'necesidad_ref','necesidad:administracion','categoria_ref',i->>'categoria_ref',
   'lectura',jsonb_build_object('corte','2026-10-18T08:00:00.000000Z','entradas','[]'::jsonb)));
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c,i) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT235: comando histórico rechazado'; END IF;
 c:=jsonb_set(c,'{datos,modalidad_clave}','"sustitucion"'::jsonb);
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c,i) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT235: par incompleto aceptado'; END IF;
 c:=jsonb_set(c,'{datos,causa_clave}','"baja_temporal"'::jsonb);
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c,i) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT235: par válido rechazado'; END IF;
 c:=jsonb_set(c,'{datos,modalidad_clave}','"Sustitucion"'::jsonb);
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c,i) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT235: modalidad no canónica aceptada'; END IF;
 c:=jsonb_set(c,'{datos,modalidad_clave}','"sustitucion"'::jsonb);
 c:=jsonb_set(c,'{datos,causa_clave}','"_baja"'::jsonb);
 IF vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(c,i) IS NOT FALSE THEN
  RAISE EXCEPTION 'CT235: causa no canónica aceptada'; END IF;
END $test$;
ROLLBACK;
