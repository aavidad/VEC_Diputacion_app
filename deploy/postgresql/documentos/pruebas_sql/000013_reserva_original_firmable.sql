\set ON_ERROR_STOP on
-- Comprobación estructural, de solo lectura, sobre la postimagen causal.
-- El recorrido V3 y la recuperación exigen las decisiones reales de AD3-158.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog, pg_temp;
DO $comprobar$
DECLARE t text; f regprocedure; a record; esperado oid:='vec_documentos_ejecutor'::regrole::oid;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_documentos.tipo_original_firmable'::regclass
   AND relrowsecurity AND relforcerowsecurity)
    OR (SELECT count(*) FROM vec_documentos.tipo_original_firmable)<>6
    OR EXISTS(SELECT 1 FROM vec_documentos.tipo_original_firmable
      WHERE modulo_id<>'contratacion_temporal'
        OR tipo_ref<>('ref:'||encode(sha256(convert_to('vec.documentos.conservacion.v1','UTF8')||'\x00'::bytea||convert_to('tipo','UTF8')||'\x00'::bytea||convert_to(tipo,'UTF8')),'hex')))
 THEN RAISE EXCEPTION 'Documentos-13: tipos originales no coinciden con catálogo v2'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class c
    CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
    WHERE c.oid='vec_documentos.tipo_original_firmable'::regclass AND x.grantee<>c.relowner)
 THEN RAISE EXCEPTION 'Documentos-13: ACL de tipos abierta'; END IF;
 FOREACH t IN ARRAY ARRAY['reserva_original_firmable','intento_original_firmable','confirmacion_original_firmable'] LOOP
  IF to_regclass('vec_documentos.'||t) IS NULL
     OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_documentos.'||t)
         AND relrowsecurity AND relforcerowsecurity)
     OR (SELECT count(*) FROM pg_policy WHERE polrelid=to_regclass('vec_documentos.'||t)
         AND polname IN ('alcance_lectura','alcance_alta'))<>2
     OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=to_regclass('vec_documentos.'||t)
         AND tgname IN ('inmutable','no_truncar') AND NOT tgisinternal)<>2
  THEN RAISE EXCEPTION 'Documentos-13: tabla % sin frontera o inmutabilidad',t; END IF;
  FOR a IN SELECT x.grantee,x.privilege_type,c.relowner FROM pg_class c
    CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
    WHERE c.oid=to_regclass('vec_documentos.'||t) LOOP
   IF a.grantee<>a.relowner THEN RAISE EXCEPTION 'Documentos-13: ACL tabla % abierta',t; END IF;
  END LOOP;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.reserva_original_firmable'::regclass
       AND contype='u' AND pg_get_constraintdef(oid)='UNIQUE (modulo_id, expediente_ref, tipo_ref, version)')
    OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.confirmacion_original_firmable'::regclass
       AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (reserva_ref, intento_num)%')
    OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.confirmacion_original_firmable'::regclass
       AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (documento_id)%'
       AND condeferrable AND condeferred)
    OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.confirmacion_original_firmable'::regclass
       AND contype='p')
 THEN RAISE EXCEPTION 'Documentos-13: unicidad o vínculo de intento incompleto'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.auditoria_operacion'::regclass
   AND conname='auditoria_operacion_accion_check'
   AND strpos(pg_get_constraintdef(oid),'documentos.original_firmable.reservar')>0
   AND strpos(pg_get_constraintdef(oid),'documentos.original_firmable.confirmar')>0)
 THEN RAISE EXCEPTION 'Documentos-13: auditoría de fases ausente'; END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_documentos.confirmar_original_firmable_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_documentos_propietario'::regrole
    AND prosecdef AND 'search_path=pg_catalog, pg_temp'=ANY(proconfig)
    AND 'row_security=on'=ANY(proconfig))
  THEN RAISE EXCEPTION 'Documentos-13: función % sin propietario/frontera',f; END IF;
  FOR a IN SELECT x.grantee,x.privilege_type,x.is_grantable,p.proowner FROM pg_proc p
    CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
    WHERE p.oid=f LOOP
   IF (a.grantee<>a.proowner AND a.grantee<>esperado)
      OR a.privilege_type<>'EXECUTE' OR (a.grantee=esperado AND a.is_grantable)
   THEN RAISE EXCEPTION 'Documentos-13: ACL función % abierta',f; END IF;
  END LOOP;
 END LOOP;
 IF to_regprocedure('vec_documentos.exigir_confirmacion_original_firmable_v1()') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_documentos.documento'::regclass
       AND tgname='exigir_confirmacion_original_firmable' AND tgdeferrable AND tginitdeferred)
    OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_documentos.referencia_externa'::regclass
       AND tgname='rechazar_tipo_original_firmable' AND NOT tgisinternal)
    OR to_regclass('vec_documentos.identificador_documental') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid='vec_documentos.reservar_identificador_v1()'::regprocedure
       AND strpos(prosrc,'confirmacion_original_firmable')>0
       AND strpos(prosrc,'identificador_documental')>0
       AND proowner='vec_documentos_propietario'::regrole AND prosecdef)
    OR EXISTS(SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
       WHERE p.oid='vec_documentos.reservar_identificador_v1()'::regprocedure
         AND x.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'Documentos-13: bypass genérico después de reserva'; END IF;
END $comprobar$;
ROLLBACK;
