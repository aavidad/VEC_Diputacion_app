\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE p pg_proc%ROWTYPE; c record; anterior text;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1(jsonb)'::regprocedure;
 SELECT pg_get_constraintdef(oid,false) AS definicion INTO STRICT c FROM pg_constraint WHERE
  conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND
  conname='auditoria_tipo_disjunto_v4' AND contype='c' AND convalidated;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole OR
  NOT p.prosecdef OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) WHERE grantee=0) OR
  NOT has_function_privilege('vec_autorizacion_propietario',p.oid,'EXECUTE') OR
  has_function_privilege('vec_autorizacion_fuente',p.oid,'EXECUTE') OR
  position('reservar_asiento_auditoria_v5()' IN p.prosrc)=0 OR
  position('control_cadena_auditoria' IN p.prosrc)<>0 OR
  encode(sha256(convert_to(c.definicion,'UTF8')),'hex') IS DISTINCT FROM
   '182b3fed22360afa10c6a496ca11e36c863ce6c731141d9314735b78504b9326'
 THEN RAISE EXCEPTION 'AD287: auditoría preperfil abierta o sin CHECK'; END IF;
END $test$;
DO $vector$
DECLARE material text; huella text; eslabon text;
BEGIN
 material:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.preperfil-bolsa-rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac('evento_'||repeat('a',32))||
  vec_autorizacion_atestada_v3.encuadrar_mac('per_'||repeat('b',32))||
  vec_autorizacion_atestada_v3.encuadrar_mac('cuenta:'||repeat('c',64))||
  vec_autorizacion_atestada_v3.encuadrar_mac('permitido')||
  vec_autorizacion_atestada_v3.encuadrar_mac('perfil_unico')||
  vec_autorizacion_atestada_v3.encuadrar_mac('correlacion_'||repeat('d',32))||
  vec_autorizacion_atestada_v3.encuadrar_mac(repeat('e',64))),'hex');
 huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.preperfil-bolsa-rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac('1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(repeat('f',64))||
  vec_autorizacion_atestada_v3.encuadrar_mac('aud_v3_b_'||repeat('a',32))||
  vec_autorizacion_atestada_v3.encuadrar_mac(material)||
  vec_autorizacion_atestada_v3.encuadrar_mac('2026-10-16T00:00:00.000001Z')),'hex');
 eslabon:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.v5')||
  vec_autorizacion_atestada_v3.encuadrar_mac('interna')||
  vec_autorizacion_atestada_v3.encuadrar_mac('1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(repeat('0',64))||
  vec_autorizacion_atestada_v3.encuadrar_mac('1')||
  vec_autorizacion_atestada_v3.encuadrar_mac('aud_v3_b_'||repeat('a',32))||
  vec_autorizacion_atestada_v3.encuadrar_mac('preperfil_bolsa_rrhh')||
  vec_autorizacion_atestada_v3.encuadrar_mac(huella)||
  vec_autorizacion_atestada_v3.encuadrar_mac('2026-10-16T00:00:00.000001Z')||
  vec_autorizacion_atestada_v3.encuadrar_mac('2026-10-16T00:00:01.000001Z')),'hex');
 IF material IS DISTINCT FROM 'd4d6c3fe4c924659db44ea64670f11c6f940ee9117ed91b70a2c3543f2019ca2' OR
  huella IS DISTINCT FROM '681b6395c23e8a05a31254b564ceec83d8ceca408f14486f8a7511e695ecbcae' OR
  eslabon IS DISTINCT FROM 'eac69309698a4af4c35a5269fd2c86250b79d36f807e71cead44cc5a75f72386'
 THEN RAISE EXCEPTION 'AD287: vector SQL y exportación Go divergentes'; END IF;
END $vector$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $test$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_preperfil_bolsa_rrhh_v1('{}'::jsonb);
  RAISE EXCEPTION 'AD287: aceptó evento sin fuente acreditada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $test$;
ROLLBACK;
