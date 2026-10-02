\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';

DO $prueba$
DECLARE
 persona constant text := 'per_prueba_admin_is9_persona_01';
 ordinaria constant text := 'cta_prueba_admin_is9_ordinaria01';
 privilegiada constant text := 'cta_prueba_admin_is9_privilegiada01';
 vinculo constant text := 'vca_prueba_admin_is9_certificado01';
 politica constant text := 'pga_prueba_admin_is9_politica01';
 certificado constant text := pg_catalog.repeat('a',64);
 ca constant text := pg_catalog.repeat('b',64);
 ahora timestamptz;
 fallo boolean;
BEGIN
 IF pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_registrador',
    'vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_revalidador',
    'vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean)','EXECUTE')
    OR pg_catalog.has_table_privilege('vec_identidad_sesiones_v1_registrador',
    'vec_identidad_sesiones_v1.politica_certificado_admin_v1','SELECT') THEN
  RAISE EXCEPTION 'IS9: privilegio runtime inesperado'; END IF;
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   pg_catalog.clock_timestamp(),false,false) THEN
  RAISE EXCEPTION 'IS9: aceptó política ausente'; END IF;
 INSERT INTO vec_identidad_sesiones_v1.cuenta
  (cuenta_ref,cuenta_privilegiada,cuenta_ordinaria_ref,provisionada_en,acto_ref)
 VALUES(ordinaria,false,NULL,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_ordinaria01'),
       (privilegiada,true,ordinaria,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_privilegiada01');
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta
  (cuenta_ref,revision,estado,registrada_en,acto_ref)
 VALUES(ordinaria,1,'activa',pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_ord01'),
       (privilegiada,1,'activa',pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_pri01');
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual
  (cuenta_ref,revision,actualizada_en,acto_ref)
 VALUES(ordinaria,1,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_ord01'),
       (privilegiada,1,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_pri01');
 INSERT INTO vec_identidad_sesiones_v1.politica_certificado_admin_v1
  (politica_ref,entorno,host_admin,ca_sha256,huella_aprobacion_sha256,
   maxima_edad_revocacion,vigente_hasta)
 VALUES(politica,'cidonia','admin.test.invalid',ca,pg_catalog.repeat('c',64),
        interval '3 minutes',pg_catalog.clock_timestamp()+interval '1 day');
 PERFORM vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1(
  vinculo,persona,privilegiada,certificado,ca,politica,
  pg_catalog.clock_timestamp()+interval '1 hour',
  'acto_admin:00000000000000000000000000000001');
 ahora:=pg_catalog.clock_timestamp();
 IF NOT vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   ahora,false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,false,
   ahora,false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,
   pg_catalog.repeat('d',64),true,ahora,false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin-otro.test.invalid',persona,privilegiada,certificado,ca,true,
   ahora,false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   ahora-interval '4 minutes',false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'produccion','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   ahora,false,false) THEN
  RAISE EXCEPTION 'IS9: frontera, CA, host o frescura incorrecta'; END IF;
 fallo:=false;
 BEGIN
  INSERT INTO vec_identidad_sesiones_v1.estado_cuenta
   (cuenta_ref,revision,estado,registrada_en,acto_ref)
  VALUES(privilegiada,2,'inactiva',pg_catalog.clock_timestamp(),
         'opr_prueba_admin_is9_inactivar01');
 EXCEPTION WHEN sqlstate '55000' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS9: inactivación omitió continuidad'; END IF;
 fallo:=false;
 BEGIN
  UPDATE vec_identidad_sesiones_v1.politica_certificado_admin_v1
   SET activa=false WHERE singleton;
 EXCEPTION WHEN sqlstate '55000' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS9: retirada omitió continuidad'; END IF;
 PERFORM vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1(
  vinculo,1,'acto_admin:00000000000000000000000000000002');
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   pg_catalog.clock_timestamp(),false,false) THEN
  RAISE EXCEPTION 'IS9: vínculo revocado admitido'; END IF;
 fallo:=false;
 BEGIN
  PERFORM vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1(
   vinculo,1,'acto_admin:00000000000000000000000000000003');
 EXCEPTION WHEN sqlstate '40001' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS9: CAS repetido admitido'; END IF;
 UPDATE vec_identidad_sesiones_v1.politica_certificado_admin_v1 SET activa=false WHERE singleton;
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid',persona,privilegiada,certificado,ca,true,
   pg_catalog.clock_timestamp(),false,false) THEN
  RAISE EXCEPTION 'IS9: política retirada admitida'; END IF;
END $prueba$;
ROLLBACK;

-- Política de producción real en la prueba: cada control se comprueba por
-- separado, sin confundir el rechazo por entorno con Kerberos o red.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $produccion$
DECLARE
 persona constant text := 'per_prueba_admin_is9_persona_02';
 ordinaria constant text := 'cta_prueba_admin_is9_ordinaria02';
 privilegiada constant text := 'cta_prueba_admin_is9_privilegiada02';
 certificado constant text := pg_catalog.repeat('e',64);
 ca constant text := pg_catalog.repeat('f',64);
 ahora timestamptz;
BEGIN
 INSERT INTO vec_identidad_sesiones_v1.cuenta
  (cuenta_ref,cuenta_privilegiada,cuenta_ordinaria_ref,provisionada_en,acto_ref)
 VALUES(ordinaria,false,NULL,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_ordinaria02'),
       (privilegiada,true,ordinaria,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_privilegiada02');
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta
  (cuenta_ref,revision,estado,registrada_en,acto_ref)
 VALUES(ordinaria,1,'activa',pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_ord02'),
       (privilegiada,1,'activa',pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_pri02');
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual
  (cuenta_ref,revision,actualizada_en,acto_ref)
 VALUES(ordinaria,1,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_ord02'),
       (privilegiada,1,pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_pri02');
 INSERT INTO vec_identidad_sesiones_v1.politica_certificado_admin_v1
  (politica_ref,entorno,host_admin,ca_sha256,huella_aprobacion_sha256,
   maxima_edad_revocacion,vigente_hasta)
 VALUES('pga_prueba_admin_is9_politica02','produccion','admin.test.invalid',
        ca,pg_catalog.repeat('c',64),interval '3 minutes',
        pg_catalog.clock_timestamp()+interval '1 day');
 PERFORM vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1(
  'vca_prueba_admin_is9_certificado02',persona,privilegiada,certificado,ca,
  'pga_prueba_admin_is9_politica02',pg_catalog.clock_timestamp()+interval '1 hour',
  'acto_admin:00000000000000000000000000000004');
 ahora:=pg_catalog.clock_timestamp();
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'produccion','admin.test.invalid',persona,privilegiada,certificado,ca,
   true,ahora,false,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'produccion','admin.test.invalid',persona,privilegiada,certificado,ca,
   true,ahora,true,false)
    OR vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'produccion','admin.test.invalid',persona,privilegiada,certificado,ca,
   true,ahora,false,true)
    OR NOT vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'produccion','admin.test.invalid',persona,privilegiada,certificado,ca,
   true,ahora,true,true) THEN
  RAISE EXCEPTION 'IS9: Kerberos o CIDR de producción omitidos'; END IF;
END $produccion$;
ROLLBACK;

-- Un snapshot REPEATABLE READ anterior al alta de un vínculo no puede
-- revalidar ni causar una reducción de administradores.
BEGIN ISOLATION LEVEL REPEATABLE READ;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $snapshot$
DECLARE fallo boolean;
BEGIN
 INSERT INTO vec_identidad_sesiones_v1.politica_certificado_admin_v1
  (politica_ref,entorno,host_admin,ca_sha256,huella_aprobacion_sha256,
   maxima_edad_revocacion,vigente_hasta)
 VALUES('pga_prueba_admin_is9_politica03','cidonia','admin.test.invalid',
        pg_catalog.repeat('b',64),pg_catalog.repeat('c',64),
        interval '3 minutes',pg_catalog.clock_timestamp()+interval '1 day');
 fallo:=false;
 BEGIN
  UPDATE vec_identidad_sesiones_v1.politica_certificado_admin_v1
   SET activa=false WHERE singleton;
 EXCEPTION WHEN sqlstate '25000' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS9: política cambió bajo snapshot RR'; END IF;
 fallo:=false;
 BEGIN
  INSERT INTO vec_identidad_sesiones_v1.estado_cuenta
   (cuenta_ref,revision,estado,registrada_en,acto_ref)
  VALUES('cta_prueba_admin_is9_ordinaria03',2,'inactiva',
         pg_catalog.clock_timestamp(),'opr_prueba_admin_is9_estado_rr03');
 EXCEPTION WHEN sqlstate '25000' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'IS9: cuenta cambió bajo snapshot RR'; END IF;
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
   'cidonia','admin.test.invalid','per_prueba_admin_is9_persona_03',
   'cta_prueba_admin_is9_ordinaria03',pg_catalog.repeat('a',64),
   pg_catalog.repeat('b',64),true,pg_catalog.clock_timestamp(),false,false) THEN
  RAISE EXCEPTION 'IS9: revalidación aceptó snapshot RR'; END IF;
END $snapshot$;
ROLLBACK;
