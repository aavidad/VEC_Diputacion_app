\set ON_ERROR_STOP on
-- CA48 (tarea F1-105): el cotejo de lectura de Bolsa bloquea los punteros en el
-- orden común del resolutor (CA36) y de la acreditación de uso (CA2).
-- Comprueba el contrato del cotejo, el orden de las tres funciones y un cotejo
-- positivo con un contexto sintético. Termina en ROLLBACK salvo que la prueba
-- de concurrencia (fase1/concurrencia_lectura_bolsa.py) pida conservarlo con
-- -v ca48_persistir_para_concurrencia=1.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $contrato$
DECLARE
 f regprocedure:='vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)'::regprocedure;
 c text; a text; r text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
     AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef
     AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[])
  OR has_function_privilege('public',f,'EXECUTE')
  OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
     IS DISTINCT FROM '16b225cf7f61482fb14748c8bdbea409413e41a9528be59f5039451cd4c83893'
 THEN RAISE EXCEPTION 'CA48 prueba: dueño, configuración, permisos o definición del cotejo divergentes'; END IF;
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to('{}','UTF8')) IS NOT FALSE
 THEN RAISE EXCEPTION 'CA48 prueba: contexto vacío admitido'; END IF;
 -- Orden común: cuenta, perfil, vínculo de contexto y persona.
 SELECT prosrc INTO STRICT c FROM pg_proc WHERE oid=f;
 SELECT prosrc INTO STRICT a FROM pg_proc WHERE oid=
  'vec_contexto_actor_v1.acreditar_uso_registro_contexto_interno_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure;
 SELECT prosrc INTO STRICT r FROM pg_proc WHERE oid=
  'vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])'::regprocedure;
 IF NOT (0<strpos(c,'FOR SHARE OF ca;') AND strpos(c,'FOR SHARE OF ca;')<strpos(c,'FOR SHARE OF pa;')
    AND strpos(c,'FOR SHARE OF pa;')<strpos(c,'FOR SHARE OF va;')
    AND strpos(c,'FOR SHARE OF va;')<strpos(c,'FOR SHARE OF xa;')
    AND strpos(c,'FOR SHARE OF xa;')<strpos(c,'FOR SHARE OF ca,pa,va,xa,vc,oa;'))
  OR NOT (0<strpos(a,'proyeccion_cuenta_actual AS a')
    AND strpos(a,'proyeccion_cuenta_actual AS a')<strpos(a,'perfil_actual AS a')
    AND strpos(a,'perfil_actual AS a')<strpos(a,'vinculo_contexto_actual AS a')
    AND strpos(a,'vinculo_contexto_actual AS a')<strpos(a,'persona_actual AS a'))
  OR NOT (0<strpos(r,'FOR UPDATE OF ca;') AND strpos(r,'FOR UPDATE OF ca;')<strpos(r,'FOR UPDATE OF pa;')
    AND strpos(r,'FOR UPDATE OF pa;')<strpos(r,'FOR UPDATE OF va;')
    AND strpos(r,'FOR UPDATE OF va;')<strpos(r,'FOR UPDATE OF pe;'))
 THEN RAISE EXCEPTION 'CA48 prueba: el cotejo, la acreditación y el resolutor no bloquean en el mismo orden'; END IF;
END $contrato$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $positivo$
DECLARE
 cuenta text:='cta_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 persona text:='per_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 perfil text:='prf_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 vinculo text:='vca_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 organizacion text:='org_ca48aaaaaaaaaaaaaaaaaaaaaa';
 corporativo text:='vcr_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 procedencia text:='prc_ca48_aaaaaaaaaaaaaaaaaaaaaa';
 desde timestamptz(6):=date_trunc('second',clock_timestamp())-interval '1 hour';
 hasta timestamptz(6):=date_trunc('second',clock_timestamp())+interval '3 hours';
 contexto bytea;
BEGIN
 INSERT INTO vec_contexto_actor_v1.procedencias
 VALUES(procedencia,1,repeat('a',64),'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
 VALUES(cuenta,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES(cuenta,1);
 INSERT INTO vec_contexto_actor_v1.persona_versiones
 VALUES(persona,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.persona_actual VALUES(persona,1);
 INSERT INTO vec_contexto_actor_v1.perfil_versiones
 VALUES(perfil,1,persona,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES(perfil,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 VALUES(vinculo,1,cuenta,perfil,persona,procedencia,1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES(vinculo,1);
 INSERT INTO vec_contexto_actor_v1.organizacion_versiones
 VALUES(organizacion,1,procedencia,1,repeat('a',64),'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES(organizacion,1);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 (corporativo,1,cuenta,1,persona,1,perfil,1,vinculo,1,organizacion,1,
  procedencia,1,repeat('a',64),'autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh',procedencia,1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',desde,hasta);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual
 VALUES(cuenta,'interna_corporativa','consulta_rrhh',corporativo,1,perfil);
 contexto:=convert_to(jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2',
  'estado','activo','cuenta_ref',cuenta,'persona_ref',persona,
  'perfil_activo_ref',perfil,'contexto_actor_ref',vinculo,
  'cuenta_version',1,'persona_version',1,'perfil_version',1,'contexto_version',1,
  'vigente_hasta',to_char(hasta-interval '1 hour','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,'UTF8');
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(contexto) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA48 prueba: contexto corporativo vivo denegado'; END IF;
 IF vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to(jsonb_set(
   convert_from(contexto,'UTF8')::jsonb,'{cuenta_version}','2')::text,'UTF8')) IS NOT FALSE
 THEN RAISE EXCEPTION 'CA48 prueba: versión de cuenta ajena admitida'; END IF;
END $positivo$;
RESET ROLE;
\echo CA48-PRUEBA-OK
\if :{?ca48_persistir_para_concurrencia}
 \if :ca48_persistir_para_concurrencia
  COMMIT;
 \else
  ROLLBACK;
 \endif
\else
 ROLLBACK;
\endif
