\set ON_ERROR_STOP on
-- Fixture real preparado mediante productores propietarios: plan aprobado
-- vigente y una tercera APP4v1 con CA activa y cuenta IS revocada. No siembra
-- asignaciones/perfiles ni modifica historia para obtener autoridad.
-- GUC vec.ensayo.objetivo_no_efectivo: el objetivo de once campos cerrado.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $objetivos$
DECLARE p jsonb:=current_setting('vec.ensayo.plan_mantenimiento')::jsonb;t jsonb:=current_setting('vec.ensayo.objetivo_no_efectivo')::jsonb;e jsonb;mensaje text;
BEGIN
 e:=vec_autorizacion.administradores_aplicacion_efectivos_internos_v3();
 IF jsonb_array_length(e)<>2 OR EXISTS(SELECT 1 FROM jsonb_array_elements(e) x WHERE x->>'asignacion_ref'=t->>'asignacion_origen_ref')
 OR vec_contexto_actor_v1.bloquear_contexto_admin_v1(t->>'cuenta_ref',t->>'persona_ref',t->>'perfil_ref',t->>'vinculo_ref',(t->>'cuenta_version')::numeric,(t->>'persona_version')::numeric,(t->>'perfil_version')::numeric,(t->>'vinculo_version')::numeric) IS NOT TRUE
 OR vec_identidad_sesiones_v1.estado_admin_interno_v1(t->>'persona_ref',t->>'cuenta_ref') IS NOT NULL THEN
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=fixture_objetivos actual=no_preparado esperado=dos_efectivas_y_tercera_CA_activa_IS_revocada';
 END IF;
 PERFORM vec_autorizacion.preimagen_mantenimiento_perfil_fijo_admin_v1(p);
 p:=jsonb_set(p,'{asignaciones,0}',t);
 BEGIN
  PERFORM vec_autorizacion.preimagen_mantenimiento_perfil_fijo_admin_v1(p);
  RAISE EXCEPTION 'AUT42 prueba: PARO clave=APP_objetivos actual=admitido esperado=conjunto_exacto';
 EXCEPTION WHEN serialization_failure THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  IF strpos(mensaje,'clave=APP_objetivos ')=0 THEN
   RAISE EXCEPTION 'AUT42 prueba: PARO clave=APP_objetivos actual=otra_guarda esperado=rechazo_conjunto_exacto';
  END IF;
 END;
END $objetivos$;
ROLLBACK;
