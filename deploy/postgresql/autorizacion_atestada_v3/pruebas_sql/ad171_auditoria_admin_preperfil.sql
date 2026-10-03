\set ON_ERROR_STOP on
-- Ejecutar solo en el clon desechable con AD171 instalada. Todo hace ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
CREATE TEMP TABLE ad171_preimagen ON COMMIT DROP AS
 SELECT auditoria_ref,huella_sha256,anterior_sha256,registrada_en
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
GRANT SELECT ON TABLE pg_temp.ad171_preimagen TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE e jsonb;x record;y record;v_secuencia numeric;f oid;
BEGIN
 f:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)');
 IF f IS NULL OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
  LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN (
   'vec_autorizacion_atestada_v3_propietario'::regrole,
   'vec_identidad_sesiones_v1_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_propietario',f,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD171 prueba: ACL privada divergente'; END IF;
 SELECT secuencia INTO STRICT v_secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 e:=pg_catalog.jsonb_build_object('tipo_registro','preperfil_autenticado',
  'evento_ref','evento_00000000000000000000000000000171',
  'actor_ref','per_AAAAAAAAAAAAAAAAAAAAAA','accion','listar_perfiles_propios_admin',
  'recurso_ref','per_AAAAAAAAAAAAAAAAAAAAAA','resultado','permitido','motivo_ref','admin.perfiles.permitido',
  'proceso','admin_sintetico','canal','administracion_privilegiada','finalidad_ref','seleccion_perfil',
  'correlacion_ref','correlacion_00000000000000000000000000000171',
  'fuente_ref','observacion_admin_preperfil:00000000000000000000000000000171','fuente_sha256',pg_catalog.repeat('1',64));
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(e);
 SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(e);
 IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y) OR x.secuencia<>v_secuencia+1
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
   WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='preperfil_autenticado'
   AND a.decision_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.registro_contexto_ref IS NULL
   AND a.sesion_ref IS NULL AND a.autenticacion_ref IS NULL AND a.operador_login IS NULL)
 THEN RAISE EXCEPTION 'AD171 prueba: acuse/replay/familia divergente'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(
   e||pg_catalog.jsonb_build_object('resultado','denegado'));
  RAISE EXCEPTION 'AD171 prueba: aceptó material cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(e||'{"perfil_activo_ref":"inventado"}'::jsonb);
  RAISE EXCEPTION 'AD171 prueba: aceptó perfil inventado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(e-'fuente_ref');
  RAISE EXCEPTION 'AD171 prueba: aceptó fuente ausente';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(e||'{"actor_ref":null}'::jsonb);
  RAISE EXCEPTION 'AD171 prueba: aceptó actor nulo';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET resultado='error' WHERE auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD171 prueba: permitió mutar auditoría';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>v_secuencia+1
 OR EXISTS(SELECT 1 FROM pg_temp.ad171_preimagen p
    LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref)
    WHERE (a.huella_sha256,a.anterior_sha256,a.registrada_en)
      IS DISTINCT FROM (p.huella_sha256,p.anterior_sha256,p.registrada_en))
 THEN RAISE EXCEPTION 'AD171 prueba: cambió cadena o historia tras rechazo'; END IF;
END $prueba$;
ROLLBACK;
