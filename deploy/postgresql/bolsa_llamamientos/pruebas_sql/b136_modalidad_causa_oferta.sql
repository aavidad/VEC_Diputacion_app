\set ON_ERROR_STOP on
-- Ejecutar después de CT235 y B136 en copia desechable. No fabrica decisiones ni recibos V3.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE datos jsonb;origen jsonb;v5 text;v6 text;canon text;v_otro text;
BEGIN
 datos:=jsonb_build_object('categoria','Administracion','centro','Granada',
  'fecha_inicio','2026-10-20','descripcion','Cobertura temporal');
 origen:=jsonb_build_object('organizacion_ref','org:granada','unidad_ref','unidad:rrhh',
  'expediente_ref','expediente:ct235','version_expediente',3,
  'clave_ct','11111111-1111-4111-8111-111111111111',
  'necesidad_ref','necesidad:administracion','acta_ref','acta:ct235',
  'categoria_ref','categoria:administracion','bolsa_ref','bolsa:administracion',
  'version_bolsa',1,'huella_bolsa_sha256',repeat('a',64),
  'instantanea_ref','instantanea:1','version_instantanea',1,
  'huella_instantanea_sha256',repeat('b',64),'politica_ref','politica:1',
  'version_politica',1,'huella_politica_sha256',repeat('c',64),
  'lectura',jsonb_build_object('corte','2026-10-18T08:00:00.000000Z',
    'entradas',jsonb_build_array(jsonb_build_object('orden_vigente',1))));
 v5:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,datos,origen,NULL,NULL,NULL,NULL);
 IF v5 IS DISTINCT FROM 'b8913711f67cf664cdd459381088eba1021b201ceefa6211b7ac8103d522684c' THEN
  RAISE EXCEPTION 'B136: vector v5 histórico cambió'; END IF;
 datos:=datos||jsonb_build_object('modalidad_clave','sustitucion','causa_clave','baja_temporal');
 v6:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,datos,origen,NULL,NULL,NULL,NULL);
 SELECT string_agg(octet_length(convert_to(x.valor,'UTF8'))::text||':'||x.valor,'' ORDER BY x.ord)
 INTO canon FROM unnest(ARRAY['vec.bolsa.oferta.canon.v6',v5,'sustitucion','baja_temporal'])
 WITH ORDINALITY x(valor,ord);
 IF v6 IS DISTINCT FROM '2335ca607f0f2650235b9af0680942f9ace8a7f52732b14b7802942c2ab8a840'
    OR v6 IS DISTINCT FROM encode(sha256(convert_to(canon,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B136: framing v6 distinto del contrato'; END IF;
 v_otro:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,
  jsonb_set(datos,'{modalidad_clave}','"interinidad"'::jsonb),origen,NULL,NULL,NULL,NULL);
 IF v_otro=v6 THEN RAISE EXCEPTION 'B136: modalidad no liga la huella'; END IF;
 v_otro:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,
  jsonb_set(datos,'{causa_clave}','"vacante"'::jsonb),origen,NULL,NULL,NULL,NULL);
 IF v_otro=v6 THEN RAISE EXCEPTION 'B136: causa no liga la huella'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,datos-'causa_clave',origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: par parcial admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,
   jsonb_set(datos,'{causa_clave}','null'::jsonb),origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: causa JSON null admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,datos,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: origen ausente admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_proc p,
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
   WHERE p.oid='vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)'::regprocedure
     AND acl.grantee=0 AND acl.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'B136: helper público'; END IF;
END $test$;
ROLLBACK;
