\set ON_ERROR_STOP on
-- Sólo PostgreSQL 18 desechable con IS6 e IS10 y políticas sin provisionar.
-- ROLLBACK conserva sin cambios la preimagen del fixture y sus roles.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
CREATE ROLE is10_registrador_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE is10_revalidador_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_registrador TO is10_registrador_prueba WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
GRANT vec_identidad_sesiones_v1_revalidador TO is10_revalidador_prueba WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
CREATE TEMP TABLE is10_fixture(
    verificada timestamptz,retirar_en timestamptz,vigente_desde timestamptz,
    sesion jsonb, sesion_substantial jsonb);
GRANT SELECT,UPDATE ON TABLE pg_temp.is10_fixture TO is10_registrador_prueba,is10_revalidador_prueba;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DO $inicial$
DECLARE tabla regclass:='vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1';r name; f record;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1)
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1) THEN
  RAISE EXCEPTION 'IS10 fixture requiere políticas sin provisionar';
 END IF;
 IF vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(
    'pga_is10_sintetica_rpt_00000001',repeat('a',64),clock_timestamp()) THEN
  RAISE EXCEPTION 'IS10 admitió política ausente';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy WHERE polrelid=tabla AND 0=ANY(polroles))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a
           WHERE c.oid=tabla AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_type t CROSS JOIN LATERAL aclexplode(t.typacl) a
           WHERE t.typrelid=tabla AND a.grantee=0) THEN
  RAISE EXCEPTION 'IS10 RLS o PUBLIC divergente';
 END IF;
 FOREACH r IN ARRAY ARRAY['is10_registrador_prueba','is10_revalidador_prueba',
    'vec_contexto_actor_v1_propietario','vec_autorizacion_propietario']::name[] LOOP
  IF has_table_privilege(r,tabla,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN
   RAISE EXCEPTION 'IS10 runtime recibió tabla de política';
  END IF;
 END LOOP;
 FOR f IN SELECT p.oid FROM pg_proc p WHERE p.pronamespace='vec_identidad_sesiones_v1'::regnamespace
   AND p.proname IN ('proteger_politica_high_rrhh_rpt_sintetica_v1','admite_politica_high_rrhh_rpt_sintetica_v1',
       'validar_politica_high_rrhh_rpt_sintetica_v1','coincide_politica_high_rrhh_rpt_sintetica_v1') LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f.oid AND a.grantee=0) THEN
   RAISE EXCEPTION 'IS10 PUBLIC recibió función';
  END IF;
 END LOOP;
END $inicial$;
SELECT cuenta_ref FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
 'opr_is10_provision_sintetica_000001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),false,NULL);
RESET ROLE;
SET SESSION AUTHORIZATION is10_registrador_prueba;
DO $ausente$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
 'opr_is10_sesion_ausente_00000001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
 decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),NULL,false,
 'interna_corporativa','certificado','alto',repeat('c',64),clock_timestamp(),clock_timestamp(),
 clock_timestamp()+interval '4 minutes','pga_is10_sintetica_rpt_00000001',repeat('a',64))) THEN
  RAISE EXCEPTION 'IS10 High sin política creó sesión';
 END IF;
 BEGIN
  PERFORM 1 FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1;
  RAISE EXCEPTION 'IS10 registrador leyó tabla';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ausente$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
INSERT INTO vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
 (politica_ref,huella_sha256,alcance,uso,metodo,garantia,vigente_desde,retirar_en,aprobacion_ref)
VALUES('pga_is10_sintetica_rpt_00000001',repeat('a',64),'solo-sintetico;sin-kerberos;no-corporativa',
 'gobierno_categorias_rpt','certificado','alto',clock_timestamp()+interval '1 second',
 clock_timestamp()+interval '1 day','opr_is10_politica_aprobada_000001');
INSERT INTO vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
 (politica_ref,huella_sha256,retirar_en)
VALUES('pga_is10_substantial_distinta_00001',repeat('b',64),clock_timestamp()+interval '1 day');
DO $antes$
BEGIN
 IF vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(
    'pga_is10_sintetica_rpt_00000001',repeat('a',64),clock_timestamp()) THEN
  RAISE EXCEPTION 'IS10 admitió política antes de su vigencia';
 END IF;
END $antes$;
RESET ROLE;
SELECT pg_sleep(1.05);
INSERT INTO pg_temp.is10_fixture
SELECT clock_timestamp(),retirar_en,vigente_desde,NULL,NULL
FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DO $cotejos$
DECLARE t record;
BEGIN
 SELECT * INTO t FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1;
 IF NOT vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(t.politica_ref,t.huella_sha256,clock_timestamp())
 OR vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(t.politica_ref,repeat('b',64),clock_timestamp())
 OR vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1('pga_is10_desconocida_0000000001',t.huella_sha256,clock_timestamp())
 OR vec_identidad_sesiones_v1.admite_politica_high_rrhh_rpt_sintetica_v1(t.politica_ref,t.huella_sha256,t.retirar_en)
 OR vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(
    'administracion_privilegiada','certificado','alto',true,t.politica_ref,t.huella_sha256,clock_timestamp())
 OR vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(
    'externa_personal','certificado','alto',false,t.politica_ref,t.huella_sha256,clock_timestamp())
 OR vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(
    'interna_corporativa','certificado','sustancial',false,t.politica_ref,t.huella_sha256,clock_timestamp())
 OR vec_identidad_sesiones_v1.validar_politica_high_rrhh_rpt_sintetica_v1(
    'interna_corporativa','kerberos_ad','alto',false,t.politica_ref,t.huella_sha256,clock_timestamp()) THEN
  RAISE EXCEPTION 'IS10 confundió vigencia, fuente, ADMIN o garantía';
 END IF;
 BEGIN
  UPDATE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 SET retirar_en=retirar_en+interval '1 day';
  RAISE EXCEPTION 'IS10 amplió vigencia';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  DELETE FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1;
  RAISE EXCEPTION 'IS10 borró historia';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  TRUNCATE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1;
  RAISE EXCEPTION 'IS10 truncó historia';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $cotejos$;
RESET ROLE;
SET SESSION AUTHORIZATION is10_registrador_prueba;
DO $registrar$
DECLARE t record;s record;
BEGIN
 SELECT * INTO t FROM pg_temp.is10_fixture;
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
 'opr_is10_sesion_confirmada_000001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
 decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),NULL,false,
 'interna_corporativa','certificado','alto',repeat('c',64),t.verificada,t.verificada,
 t.verificada+interval '4 minutes','pga_is10_sintetica_rpt_00000001',repeat('a',64));
 UPDATE pg_temp.is10_fixture SET sesion=to_jsonb(s);
 IF (SELECT count(*) FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
 'opr_is10_sesion_confirmada_000001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
 decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),NULL,false,
 'interna_corporativa','certificado','alto',repeat('c',64),t.verificada,t.verificada,
 t.verificada+interval '4 minutes','pga_is10_sintetica_rpt_00000001',repeat('a',64)))<>1 THEN
  RAISE EXCEPTION 'IS10 recuperación de registro divergente';
 END IF;
 -- IS6 conserva Substantial; sólo cambian aserción y sesión HMAC.
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
 'opr_is10_sesion_substantial_00001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('55',32),'hex'),decode(repeat('66',32),'hex'),
 decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),NULL,false,
 'interna_corporativa','certificado','sustancial',repeat('d',64),t.verificada,t.verificada,
 t.verificada+interval '4 minutes','pga_is10_substantial_distinta_00001',repeat('b',64));
 UPDATE pg_temp.is10_fixture SET sesion_substantial=to_jsonb(s);
END $registrar$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION is10_revalidador_prueba;
DO $revalidar$
DECLARE t record;s jsonb;
BEGIN
 SELECT * INTO t FROM pg_temp.is10_fixture;s:=t.sesion;
 IF NOT vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(
    'pga_is10_sintetica_rpt_00000001',repeat('a',64),t.vigente_desde,t.retirar_en)
 OR vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(
    'pga_is10_sintetica_rpt_00000001',repeat('a',64),t.vigente_desde,t.retirar_en+interval '1 microsecond')
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
    s->>'autenticacion_ref',s->>'sesion_ref'))<>1
 OR NOT vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
    s->>'autenticacion_ref',repeat('c',64),s->>'asercion_ref',s->>'sesion_ref',s->>'cuenta_ref',s->>'cuenta_ordinaria_ref',
    false,'interna_corporativa','certificado','alto','pga_is10_sintetica_rpt_00000001',repeat('a',64),t.verificada,t.verificada,
    s->>'control_sesion_ref',s->>'control_sesion_revision_texto',s->>'control_sesion_estado',s->>'control_sesion_huella_sha256',
    (s->>'sesion_revalidada_en')::timestamptz,(s->>'sesion_valida_hasta')::timestamptz) THEN
  RAISE EXCEPTION 'IS10 sesión/cápsula vigente no revalidada';
 END IF;
END $revalidar$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
UPDATE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
 SET activa=false,retiro_ref='opr_is10_politica_retirada_000001' WHERE singleton;
DO $retiro$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1
    WHERE NOT activa AND retirada_en IS NOT NULL AND retirada_por=session_user
      AND retiro_ref='opr_is10_politica_retirada_000001') THEN
  RAISE EXCEPTION 'IS10 retirada sin referencia/actor/fecha';
 END IF;
 BEGIN
  UPDATE vec_identidad_sesiones_v1.politica_high_rrhh_rpt_sintetica_v1 SET activa=true;
  RAISE EXCEPTION 'IS10 resucitó política';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
END $retiro$;
RESET ROLE;
SET SESSION AUTHORIZATION is10_revalidador_prueba;
DO $denegar$
DECLARE t record;s jsonb;original jsonb;
BEGIN
 SELECT * INTO t FROM pg_temp.is10_fixture;s:=t.sesion;original:=t.sesion_substantial;
 IF (SELECT count(*) FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
     s->>'autenticacion_ref',s->>'sesion_ref'))<>0
 OR vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
    s->>'autenticacion_ref',repeat('c',64),s->>'asercion_ref',s->>'sesion_ref',s->>'cuenta_ref',s->>'cuenta_ordinaria_ref',
    false,'interna_corporativa','certificado','alto','pga_is10_sintetica_rpt_00000001',repeat('a',64),t.verificada,t.verificada,
    s->>'control_sesion_ref',s->>'control_sesion_revision_texto',s->>'control_sesion_estado',s->>'control_sesion_huella_sha256',
    (s->>'sesion_revalidada_en')::timestamptz,(s->>'sesion_valida_hasta')::timestamptz)
 OR vec_identidad_sesiones_v1.coincide_politica_high_rrhh_rpt_sintetica_v1(
    'pga_is10_sintetica_rpt_00000001',repeat('a',64),t.vigente_desde,t.retirar_en)
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
     original->>'autenticacion_ref',original->>'sesion_ref'))<>1 THEN
  RAISE EXCEPTION 'IS10 retirada no cerró cápsula o alteró IS6';
 END IF;
END $denegar$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION is10_registrador_prueba;
DO $denegar_reconciliacion$
DECLARE t record;
BEGIN
 SELECT * INTO t FROM pg_temp.is10_fixture;
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
 'opr_is10_sesion_confirmada_000001','vec.identidad.hmac-sha256.v1','idh_is10_sintetica_rpt_00000001',
 'is10-clave-sintetica',1,decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
 decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),NULL,false,
 'interna_corporativa','certificado','alto',repeat('c',64),t.verificada,t.verificada,
 t.verificada+interval '4 minutes','pga_is10_sintetica_rpt_00000001',repeat('a',64))) THEN
  RAISE EXCEPTION 'IS10 recuperó sesión tras retirar política';
 END IF;
END $denegar_reconciliacion$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo IS10-POLITICA-OK
