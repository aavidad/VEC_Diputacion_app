\set ON_ERROR_STOP on
-- Solo clon sintético desechable: conserva la fixture válida para negativas
-- de aislamiento y la carrera. Desechar el clon al terminar; no usar historia real.
-- Ejecutar primero sin variables. Después, abrir T1 con -f este archivo
-- -v ca21_fase=t1 e stdin abierto; esperar CA21_T1_SNAPSHOT_LISTO, ejecutar
-- otra sesión con -v ca21_fase=t2 y esperar COMMIT. Enviar una línea a stdin
-- de T1: debe observar 40001. Sincronización explícita, sin espera temporal.
\if :{?ca21_fase}
\else
\set ca21_fase focal
\endif
SELECT :'ca21_fase'='t1' AS ca21_t1, :'ca21_fase'='t2' AS ca21_t2 \gset
\if :ca21_t1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $snapshot$
BEGIN
 IF (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_rpt_actual
     WHERE perfil_ref='prf_sintetico_ca21_rpt_00000000000001'
     AND catalogo_id='personal.categorias' AND modulo_id='contratacion_temporal')<>1
 THEN RAISE EXCEPTION 'T1 requiere una sola cuenta RPT'; END IF;
END $snapshot$;
SELECT generacion AS ca21_generacion_t1
 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id=true \gset
\prompt 'CA21_T1_SNAPSHOT_LISTO ' ca21_continuar
SET LOCAL ROLE vec_autorizacion_propietario;
DO $snapshot_obsoleto$
BEGIN
 BEGIN
  PERFORM vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64));
  RAISE EXCEPTION 'T1 no rechazó la inserción posterior al snapshot';
 EXCEPTION WHEN serialization_failure THEN
  RAISE NOTICE 'CA21_T1_40001_OK';
 END;
END $snapshot_obsoleto$;
ROLLBACK;
\elif :ca21_t2
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $segunda_cuenta$
DECLARE d jsonb; g numeric;
BEGIN
 SELECT generacion INTO STRICT g FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id=true;
 SELECT documento||jsonb_build_object(
  'vinculo_rpt_ref','vcr_sintetico_ca21_rpt_00000000000002',
  'cuenta_ref','cta_sintetica_ca21_000000000000000002',
  'vinculo_contexto_ref','vca_sintetico_ca21_rpt_00000000000002') INTO STRICT d
 FROM vec_contexto_actor_v1.vinculo_rpt_versiones
 WHERE vinculo_rpt_ref='vcr_sintetico_ca21_rpt_00000000000001' AND version=1;
 PERFORM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,0,NULL,
  encode(sha256(convert_to(jsonb_build_array(d,1)::text,'UTF8')),'hex'),
  'operador_sintetico_ca21','evidencia_sintetica_ca21');
 IF (SELECT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id=true)<=g
 THEN RAISE EXCEPTION 'T2 no avanzó la generación CA2'; END IF;
END $segunda_cuenta$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $ambiguedad$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT FALSE
 THEN RAISE EXCEPTION 'T2 concedió con dos cuentas'; END IF;
END $ambiguedad$;
COMMIT;
\echo CA21_T2_INSERT_COMMIT_OK
\else
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_sintetica_ca21_000000000000000001',1,'prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_sintetica_ca21_000000000000000001',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_sintetica_ca21_000000000000000001',1,'prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_sintetica_ca21_000000000000000001',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_sintetico_ca21_ct_000000000000001',1,'per_sintetica_ca21_000000000000000001','prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour'),
 ('prf_sintetico_ca21_rpt_00000000000001',1,'per_sintetica_ca21_000000000000000001','prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_sintetico_ca21_ct_000000000000001',1),('prf_sintetico_ca21_rpt_00000000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_sintetico_ca21_ct_000000000000001',1,'cta_sintetica_ca21_000000000000000001','prf_sintetico_ca21_ct_000000000000001','per_sintetica_ca21_000000000000000001','prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour'),
 ('vca_sintetico_ca21_rpt_00000000000001',1,'cta_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','per_sintetica_ca21_000000000000000001','prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_sintetico_ca21_ct_000000000000001',1),('vca_sintetico_ca21_rpt_00000000000001',1);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_sinteticaca210000000001',1,'prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES('org_sinteticaca210000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_sintetico_ca21_ct_000000000000001',1,'cta_sintetica_ca21_000000000000000001',1,'per_sintetica_ca21_000000000000000001',1,'prf_sintetico_ca21_ct_000000000000001',1,'vca_sintetico_ca21_ct_000000000000001',1,'org_sinteticaca210000000001',1,'prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh','prc_sintetica_ca21_000000000000000001',1,repeat('a',64),'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_sintetica_ca21_000000000000000001','interna_corporativa','consulta_rrhh','vcr_sintetico_ca21_ct_000000000000001',1);

-- Fuentes de la segunda cuenta ya comprometidas antes del snapshot T1.
-- T2 insertará solo el vínculo RPT, sin cambiar las filas leídas del primero.
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
 SELECT 'cta_sintetica_ca21_000000000000000002',version,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta
 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones WHERE cuenta_ref='cta_sintetica_ca21_000000000000000001' AND version=1;
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_sintetica_ca21_000000000000000002',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 SELECT 'vca_sintetico_ca21_rpt_00000000000002',version,'cta_sintetica_ca21_000000000000000002',perfil_ref,persona_ref,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta
 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref='vca_sintetico_ca21_rpt_00000000000001' AND version=1;
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_sintetico_ca21_rpt_00000000000002',1);

SET LOCAL ROLE vec_autorizacion_propietario;
DO $ausencia$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_ct_000000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) THEN
  RAISE EXCEPTION 'consulta_rrhh concedió RPT';
 END IF;
END $ausencia$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $publicar_y_cas$
DECLARE d jsonb; h text; r record; cantidad integer;
BEGIN
 SELECT (to_jsonb(v)-ARRAY['vinculo_corporativo_ref','version'])||jsonb_build_object(
  'vinculo_rpt_ref','vcr_sintetico_ca21_rpt_00000000000001',
  'perfil_ref','prf_sintetico_ca21_rpt_00000000000001','vinculo_contexto_ref','vca_sintetico_ca21_rpt_00000000000001',
  'uso','gobierno_categorias_rpt','catalogo_id','personal.categorias','modulo_id','contratacion_temporal',
  'rol_ref','rol:rrhh_gobierno_categorias_rpt:v1','rol_version',1,'rol_huella_sha256',repeat('b',64)) INTO STRICT d
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v WHERE v.vinculo_corporativo_ref='vcr_sintetico_ca21_ct_000000000000001';
 h:=encode(sha256(convert_to(jsonb_build_array(d,1)::text,'UTF8')),'hex');
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,0,NULL,h,'operador_sintetico_ca21','evidencia_sintetica_ca21');
 IF r.version<>1 OR r.huella_sha256<>h THEN RAISE EXCEPTION 'publicación divergente'; END IF;
 BEGIN
  PERFORM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,0,NULL,h,'operador_sintetico_ca21','evidencia_sintetica_ca21');
  RAISE EXCEPTION 'CAS permitió repetir versión inicial';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,1,repeat('f',64),h,'operador_sintetico_ca21','evidencia_sintetica_ca21');
  RAISE EXCEPTION 'CAS aceptó huella ajena';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 BEGIN
  PERFORM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,1,r.huella_sha256,repeat('f',64),'operador_sintetico_ca21','evidencia_sintetica_ca21');
  RAISE EXCEPTION 'contenido no aprobado publicado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT count(*) INTO cantidad FROM vec_contexto_actor_v1.vinculo_rpt_eventos WHERE vinculo_rpt_ref='vcr_sintetico_ca21_rpt_00000000000001';
 IF cantidad<>1 THEN RAISE EXCEPTION 'CAS duplicó historia/eventos'; END IF;
 BEGIN
  UPDATE vec_contexto_actor_v1.vinculo_rpt_versiones SET estado='revocado' WHERE vinculo_rpt_ref='vcr_sintetico_ca21_rpt_00000000000001';
  RAISE EXCEPTION 'historia mutable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $publicar_y_cas$;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $ambitos_y_roles$
DECLARE c text:='personal.categorias'; m text:='contratacion_temporal'; rol text:='rol:rrhh_gobierno_categorias_rpt:v1';
 p text:='per_sintetica_ca21_000000000000000001'; f text:='prf_sintetico_ca21_rpt_00000000000001';
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,c,m,rol,1,repeat('b',64)) IS NOT TRUE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,c,'bolsa',rol,1,repeat('b',64)) IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,'catalogo.ajeno',m,rol,1,repeat('b',64)) IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,c,m,rol,2,repeat('b',64)) IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,c,m,rol,1,repeat('c',64)) IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,f,c,m,NULL,1,repeat('b',64)) IS NOT FALSE
 OR vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(p,'prf_sintetico_ca21_ct_000000000000001',c,m,rol,1,repeat('b',64)) IS NOT FALSE
 THEN RAISE EXCEPTION 'vínculo/ámbito/rol no exacto'; END IF;
END $ambitos_y_roles$;

-- Conservar una preimagen nominal válida para READ COMMITTED y T1/T2.
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SAVEPOINT antes_cambio_fuente;
INSERT INTO vec_contexto_actor_v1.persona_versiones
 SELECT persona_ref,2,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta
 FROM vec_contexto_actor_v1.persona_versiones WHERE persona_ref='per_sintetica_ca21_000000000000000001' AND version=1;
UPDATE vec_contexto_actor_v1.persona_actual SET version=2 WHERE persona_ref='per_sintetica_ca21_000000000000000001';
SET LOCAL ROLE vec_autorizacion_propietario;
DO $fuente_nueva$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT FALSE THEN
  RAISE EXCEPTION 'versión fuente nueva permitió vínculo antiguo';
 END IF;
END $fuente_nueva$;
ROLLBACK TO antes_cambio_fuente;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $caducidad_y_revocacion$
DECLARE d jsonb; caducado jsonb; h text; r record; cantidad integer;
BEGIN
 SELECT documento,huella_sha256 INTO STRICT d,h FROM vec_contexto_actor_v1.vinculo_rpt_versiones
 WHERE vinculo_rpt_ref='vcr_sintetico_ca21_rpt_00000000000001' AND version=1;
 caducado:=d||jsonb_build_object('vigente_hasta',clock_timestamp()-interval '1 second');
 BEGIN
  PERFORM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(caducado,1,h,
   encode(sha256(convert_to(jsonb_build_array(caducado,2)::text,'UTF8')),'hex'),
   'operador_sintetico_ca21','evidencia_sintetica_ca21');
  RAISE EXCEPTION 'caducado concedió';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 d:=d||'{"estado":"revocado"}'::jsonb;
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(d,1,h,
   encode(sha256(convert_to(jsonb_build_array(d,2)::text,'UTF8')),'hex'),'operador_sintetico_ca21','evidencia_sintetica_ca21');
 IF r.version<>2 THEN RAISE EXCEPTION 'revocación sin versión'; END IF;
 SELECT count(*) INTO cantidad FROM vec_contexto_actor_v1.vinculo_rpt_versiones WHERE vinculo_rpt_ref='vcr_sintetico_ca21_rpt_00000000000001';
 IF cantidad<>2 THEN RAISE EXCEPTION 'historia RPT alterada'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones WHERE vinculo_corporativo_ref='vcr_sintetico_ca21_ct_000000000000001' AND version=1 AND uso='consulta_rrhh' AND estado='activo')
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual WHERE cuenta_ref='cta_sintetica_ca21_000000000000000001' AND uso='consulta_rrhh' AND version=1)
 THEN RAISE EXCEPTION 'RPT alteró CT'; END IF;
END $caducidad_y_revocacion$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $retirada$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT FALSE THEN
  RAISE EXCEPTION 'revocado concedió';
 END IF;
END $retirada$;
RESET ROLE;
DO $acl$
DECLARE f oid:='vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(text,text,text,text,text,numeric,text)'::regprocedure;
BEGIN
 IF pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f AND(a.grantee=0 OR a.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'ACL RPT abrió otra autoridad'; END IF;
END $acl$;
ROLLBACK;
BEGIN ISOLATION LEVEL READ COMMITTED;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $aislamiento$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT FALSE THEN
  RAISE EXCEPTION 'aislamiento ordinario concedió';
 END IF;
END $aislamiento$;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SAVEPOINT antes_ausencia_generacion;
DELETE FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id=true;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $generacion_ausente$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT FALSE
 THEN RAISE EXCEPTION 'ausencia de generación concedió'; END IF;
END $generacion_ausente$;
ROLLBACK TO antes_ausencia_generacion;
DO $preimagen_valida$
BEGIN
 IF vec_contexto_actor_v1.acreditar_rpt_rrhh_v1('per_sintetica_ca21_000000000000000001','prf_sintetico_ca21_rpt_00000000000001','personal.categorias','contratacion_temporal','rol:rrhh_gobierno_categorias_rpt:v1',1,repeat('b',64)) IS NOT TRUE
 THEN RAISE EXCEPTION 'fixture válida no conservada'; END IF;
END $preimagen_valida$;
ROLLBACK;
\echo CA21 pruebas focales OK; fixture válida conservada en clon desechable
\endif
