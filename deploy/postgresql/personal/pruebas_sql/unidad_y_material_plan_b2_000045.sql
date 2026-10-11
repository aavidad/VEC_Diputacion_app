\set ON_ERROR_STOP on
-- Ensayo propio P45 en PG18 desechable, después de Personal41–44 y P45.
-- Ejecutar como DBA del clon. Todo termina en ROLLBACK. El núcleo V3 permanece
-- instalado e intacto: las decisiones de frontera y su sobre están deliberadamente
-- incompletos y sin firma. No acredita operaciones nominales positivas ni recuperación E2E.
-- La guarda de unidad se prueba extraída del cuerpo instalado (no una copia);
-- la autorización precede al replay y a esa guarda en la función completa.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL lock_timeout='2s';
CREATE TEMP TABLE p45_cuentas AS SELECT
 (SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo) recibos,
 (SELECT count(*) FROM vec_personal.relacion_servicio_historia) relaciones,
 (SELECT count(*) FROM vec_personal.ocupacion_empleado_historia) ocupaciones,
 (SELECT count(*) FROM vec_personal.acceso_plan_incorporacion_ct) accesos;
DO $acl$
DECLARE f regprocedure; a record; h text;
BEGIN
 FOR a IN SELECT * FROM (VALUES
 ('vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  '74c1f4df1e228f3a461a08618ecdf5ab7acd9024fd7633d293f7352a2db649b3',false),
 ('vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'ed3701c97406915c0cef13161a861fe44d2dd133aa981b4354ffc4ac6881a485',true)) x(firma,huella,ejecutor) LOOP
  f:=a.firma::regprocedure;
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO h;
  IF h IS DISTINCT FROM a.huella OR NOT (SELECT prosecdef FROM pg_proc WHERE oid=f)
     OR (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_personal_propietario'::regrole
     OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE') IS DISTINCT FROM a.ejecutor
     OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) z
       WHERE p.oid=f AND (z.grantee NOT IN ('vec_personal_propietario'::regrole,
         CASE WHEN a.ejecutor THEN 'vec_personal_ejecutor'::regrole ELSE 'vec_personal_propietario'::regrole END)
         OR z.privilege_type<>'EXECUTE' OR z.is_grantable)) THEN
   RAISE EXCEPTION 'P45: postimagen o ACL divergente'; END IF;
 END LOOP;
END $acl$;
SET LOCAL ROLE vec_personal_propietario;
-- Organismo, unidad y nodo son tres referencias diferentes.
INSERT INTO vec_personal.org_nodo_historia
 (nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,
  catalogo_revision,catalogo_entrada_clave,denominacion,retirado,vigente_desde,
  vigente_hasta,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
SELECT md5('p45:'||n)::uuid,1,CASE WHEN n='ajena' THEN 'org:otra' ELSE 'org:p45' END,
 'uni:p45:'||CASE WHEN n='duplicada2' THEN 'duplicada' ELSE n END,'centro',
 'estructura-organizativa-dipgra',1,1,'entrada:p45:'||n,'Unidad sintética P45',n='retirada',
 CASE WHEN n='futura' THEN DATE '2100-01-01' ELSE DATE '2026-01-01' END,
 CASE WHEN n='vencida' THEN DATE '2026-10-11' ELSE NULL END,
 CASE WHEN n='no_conocida' THEN clock_timestamp()+interval '1 day' ELSE clock_timestamp()-interval '2 days' END,
 'fuente:p45','acto:p45',repeat('f',64)
FROM unnest(ARRAY['valida','ajena','retirada','futura','vencida','no_conocida',
 'duplicada','duplicada2','cambio_org','retiro_posterior']) n;
INSERT INTO vec_personal.org_nodo_historia
SELECT nodo_ref,2,CASE WHEN unidad_ref='uni:p45:cambio_org' THEN 'org:otra' ELSE organismo_ref END,
 unidad_ref,clase,catalogo_ref,1,2,catalogo_entrada_clave,denominacion,centro_padre_ref,
 unidad_ref='uni:p45:retiro_posterior',vigente_desde,vigente_hasta,
 clock_timestamp()-interval '1 day',fuente_ref,acto_ref,huella_fuente_sha256
FROM vec_personal.org_nodo_historia WHERE unidad_ref IN ('uni:p45:cambio_org','uni:p45:retiro_posterior');
DO $unidad$
DECLARE fuente text; guarda text; caso record; codigo text;
BEGIN
 SELECT prosrc INTO STRICT fuente FROM pg_proc WHERE oid=
 'vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 guarda:='  SELECT count(*) INTO v_nodos'||split_part(split_part(fuente,'  SELECT count(*) INTO v_nodos',2),'  v_catalogo:=',1);
 IF guarda='' OR strpos(fuente,'IF FOUND THEN')>strpos(fuente,'-- Personal45:')
    OR strpos(fuente,'consumir_registro_empleado_b2_v3_atestada')>strpos(fuente,'IF FOUND THEN') THEN
  RAISE EXCEPTION 'P45: extracción u orden consumo/replay/unidad divergente'; END IF;
 FOR caso IN SELECT * FROM (VALUES
 ('valida','ok'),('ajena','42501'),('inexistente','42501'),('retirada','42501'),
 ('futura','42501'),('vencida','42501'),('no_conocida','42501'),('duplicada','42501'),
 ('cambio_org','42501'),('retiro_posterior','42501')) x(nombre,esperado) LOOP
  codigo:='ok';
  BEGIN
   EXECUTE format($q$DO $gate$
    DECLARE v_nodos bigint; v_org text:='org:p45'; v_unidad text:=%L;
      v_desde date:='2026-10-11'; v_ahora timestamptz:=clock_timestamp();
    BEGIN %s END $gate$;$q$,'uni:p45:'||caso.nombre,guarda);
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  END;
  IF codigo IS DISTINCT FROM caso.esperado THEN
   RAISE EXCEPTION 'P45 unidad %: actual %, esperado %',caso.nombre,codigo,caso.esperado; END IF;
 END LOOP;
 RAISE NOTICE 'P45_UNIDAD_OK: 10 casos del predicado instalado; no sustituye el recorrido nominal';
END $unidad$;
-- Dos planes sintéticos con claves reservadas exactas. No se ha consumido V3
-- para crear este fixture DBA; ningún recibo fixture acredita una alta nominal.
DO $planes$
DECLARE canon text:=$datos${"idempotencia_ref":"45000000-0000-4000-8000-000000000001","origen_ct_ref":"ct:p45","origen_ct_recibo_ref":"ctrec:p45","origen_ct_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","expediente_ref":"exp:p45","expediente_version":7,"organismo_ref":"org:p45","unidad_ref":"uni:p45","persona_ref":"per_pppppppppppppppppppppppp","persona_version":1,"fuente_bolsa_ref":"bolsa:p45","fuente_bolsa_version":1,"fuente_bolsa_recibo_ref":"bolsarec:p45","fuente_bolsa_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","regimen":{"ref":"reg:p45","version":1},"modalidad":{"ref":"mod:p45","version":1},"desde":"2026-10-11","hasta":"","plaza_ref":"plaza:45000000-0000-4000-8000-000000000003","puesto_ref":"puesto:45000000-0000-4000-8000-000000000004","clase_ocupacion":"temporal","version_plantilla_ref":"plantilla:p45","version_rpt_ref":"rpt:p45","revision_plaza":1,"revision_puesto":1,"fuente_organizacion_ref":"organizacion:p45","fuente_organizacion_huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","catalogo_rpt_id":"rpt:p45","catalogo_rpt_modulo":"personal","catalogo_rpt_categoria":"categoria:p45","catalogo_rpt_version":1,"catalogo_rpt_huella_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","vinculo_ct_recibo_ref":"vinculo:p45","procedencia":{"acto_ref":"acto:p45","fuente_ref":"fuente:p45","fuente_version":1,"fuente_huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","idempotencia_ref":"45000000-0000-4000-8000-000000000005"}}$datos$;
 d jsonb; c record; n int; modo text; emp text; pr text; rr text; ka uuid; ko uuid;
 uso text; res text; conf text; sha text; ph text;
BEGIN
 SELECT * INTO STRICT c FROM vec_personal.clases_ocupacion_plan_ct_catalogo
 WHERE ref='personal:incorporacion_ct:clases_ocupacion' AND version=1;
 FOR n IN 1..2 LOOP
  modo:=CASE WHEN n=1 THEN 'alta_empleado' ELSE 'nueva_relacion' END;
  emp:=CASE WHEN n=1 THEN '' ELSE 'emp_'||repeat('x',24) END;
  d:=canon::jsonb;
  IF n=2 THEN
   d:=jsonb_set(d,'{idempotencia_ref}',to_jsonb('45000000-0000-4000-8000-000000000002'::text));
   d:=jsonb_set(d,'{origen_ct_ref}','"ct:p45:segunda"'::jsonb);
  END IF;
  canon:=d::text; sha:=encode(sha256(convert_to(canon,'UTF8')),'hex');
  pr:='perplan_'||md5('plan:p45:'||n); rr:='perplanrec_'||md5('recibo:p45:'||n);
  ka:=md5('alta:p45:'||n)::uuid; ko:=md5('ocupacion:p45:'||n)::uuid;
  uso:='uso:p45:'||n;res:='reserva:p45:'||n;conf:='confirmacion:p45:'||n;
  ph:=encode(sha256(convert_to(sha||'|'||pr||'|'||rr||'|'||modo||'|'||emp||'|'||ka::text||'|'||ko::text||'|'||uso||'|'||res||'|'||conf||'|'||c.ref||'|'||c.version||'|'||c.huella_sha256,'UTF8')),'hex');
  INSERT INTO vec_personal.plan_incorporacion_ct VALUES((d->>'idempotencia_ref')::uuid,
   pr,rr,'org:p45',canon,d,sha,modo,emp,ka,ko,uso,res,conf,c.ref,c.version,c.huella_sha256,ph,
   'decision:p45:fixture','auditoria:p45:fixture',md5('consumo:p45:'||n)||md5('consumo:p45:'||n),clock_timestamp());
 END LOOP;
END $planes$;
RESET ROLE;
CREATE ROLE vec_p45_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_p45_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_p45_prueba;
DO $actos$
DECLARE caso record; m jsonb; clave text; err text; codigo text;
BEGIN
 FOR caso IN SELECT * FROM (VALUES
 ('alta','org:otra','uni:p45','per_'||repeat('p',24),'','',1,'organismo'),
 ('alta','org:p45','uni:otra','per_'||repeat('p',24),'','',1,'unidad'),
 ('alta','org:p45','uni:p45','per_'||repeat('q',24),'','',1,'persona'),
 ('hecho','org:p45','uni:p45','','emp_'||repeat('q',24),'relacion',2,'empleado'),
 ('hecho','org:p45','uni:p45','','emp_'||repeat('x',24),'ocupacion',1,'sin_relacion'),
 ('alta','org:p45','uni:p45','per_'||repeat('p',24),'','',1,'admitida'))
 x(op,org,uni,persona,emp,tipo,n,nombre) LOOP
  clave:=md5(CASE WHEN caso.nombre='sin_relacion' THEN 'ocupacion:p45:' ELSE 'alta:p45:' END||caso.n)::uuid::text;
  m:=jsonb_build_object('procedencia',jsonb_build_object('idempotencia_ref',clave),
    'organismo_ref',caso.org,'unidad_ref',caso.uni,'persona_ref',caso.persona,
    'empleado_ref',caso.emp,'tipo',caso.tipo);
  codigo:='ok';err:='';
  BEGIN
   PERFORM vec_personal.registrar_acto_plan_incorporacion_ct_v1(caso.op,m::text,
    '{}'::text::bytea,'{}'::text::bytea,'{}'::text::bytea,'{}'::text::bytea,
    1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,err=MESSAGE_TEXT;
  END;
  IF caso.nombre='admitida' THEN
   IF codigo='ok' OR err LIKE 'Personal45:%' OR err LIKE 'Personal23:%' THEN
    RAISE EXCEPTION 'P45: material compatible no alcanzó el original: % %',codigo,err; END IF;
  ELSIF codigo IS DISTINCT FROM '42501' OR err IS DISTINCT FROM
    (CASE WHEN caso.nombre='sin_relacion' THEN 'Personal45: ocupación fuera del plan' ELSE 'Personal45: material fuera del plan' END) THEN
   RAISE EXCEPTION 'P45: acto %: % %',caso.nombre,codigo,err;
  END IF;
 END LOOP;
 RAISE NOTICE 'P45_ACTOS_OK: 5 denegaciones de fachada real y paso al original sin firma';
END $actos$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_personal_propietario;
-- Recibo y relación DBA sintéticos para probar el enlace de ocupación. No
-- son efectos autorizados: se insertan solo en esta TX que acaba en ROLLBACK.
INSERT INTO vec_personal.relacion_servicio_historia
 (relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,
  modalidad_ref,estado,vigente_desde,conocido_desde,acto_ref,fuente_ref,fuente_version,
  fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
VALUES('rel_'||repeat('r',24),1,'per_'||repeat('p',24),'emp_'||repeat('r',24),'org:p45','uni:p45',
 'reg:p45','mod:p45','vigente','2026-10-11',clock_timestamp(),'acto:p45','fuente:p45',1,
 repeat('e',64),'decision:p45:fixture','auditoria:p45:fixture',
 '{"regimen":{"ref":"reg:p45","version":1},"modalidad":{"ref":"mod:p45","version":1}}');
INSERT INTO vec_personal.registro_empleado_b2_recibo
 (idempotencia_ref,operacion,material_sha256,actor_ref,efecto_ref,recibo_ref,empleado_ref,
  relacion_ref,proyeccion_ref,tipo,version,decision_ref,consumo_huella_sha256,auditoria_ref,
  registrado_en,catalogo_snapshot,acreditacion_persona_version,acreditacion_procedencia_ref,
  acreditacion_procedencia_version,acreditacion_procedencia_huella_sha256)
VALUES(md5('alta:p45:1')::uuid,'alta',repeat('0',64),'per_'||repeat('p',24),'per_'||repeat('p',24),
 'perrec_'||md5('p45:fixture'),'emp_'||repeat('r',24),'rel_'||repeat('r',24),'pep_'||repeat('r',24),
 'alta',1,'decision:p45:fixture',repeat('f',64),'auditoria:p45:fixture',clock_timestamp(),
 '{"regimen":{"ref":"reg:p45","version":1},"modalidad":{"ref":"mod:p45","version":1}}',
 1,'prc:p45:fixture',1,repeat('f',64));
RESET ROLE;
SET SESSION AUTHORIZATION vec_p45_prueba;
DO $ocupacion$
DECLARE caso record; m jsonb; codigo text; err text;
BEGIN
 FOR caso IN SELECT * FROM (VALUES
  ('otro_empleado','emp_'||repeat('s',24),'rel_'||repeat('r',24),1),
  ('otra_relacion','emp_'||repeat('r',24),'rel_'||repeat('s',24),1),
  ('otra_revision','emp_'||repeat('r',24),'rel_'||repeat('r',24),2),
  ('compatible','emp_'||repeat('r',24),'rel_'||repeat('r',24),1)) x(nombre,emp,rel,version) LOOP
  m:=jsonb_build_object('procedencia',jsonb_build_object('idempotencia_ref',md5('ocupacion:p45:1')::uuid::text),
    'organismo_ref','org:p45','unidad_ref','uni:p45','tipo','ocupacion',
    'empleado_ref',caso.emp,'relacion_ref',caso.rel,'relacion_version_esperada',caso.version);
  codigo:='ok';err:='';
  BEGIN
   PERFORM vec_personal.registrar_acto_plan_incorporacion_ct_v1('hecho',m::text,
    '{}'::text::bytea,'{}'::text::bytea,'{}'::text::bytea,'{}'::text::bytea,
    1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,err=MESSAGE_TEXT;
  END;
  IF caso.nombre='compatible' THEN
   IF codigo='ok' OR err LIKE 'Personal45:%' OR err LIKE 'Personal23:%' THEN
    RAISE EXCEPTION 'P45: ocupación compatible no alcanzó el original: % %',codigo,err; END IF;
  ELSIF codigo IS DISTINCT FROM '42501' OR err IS DISTINCT FROM 'Personal45: ocupación fuera del plan' THEN
   RAISE EXCEPTION 'P45: ocupación %: % %',caso.nombre,codigo,err; END IF;
 END LOOP;
 RAISE NOTICE 'P45_OCUPACION_OK: empleado, relación y revisión ligados al recibo previo';
END $ocupacion$;
DO $frontera_plan$
DECLARE actor text:='{"actor_ref":"per_pppppppppppppppppppppppp","contexto_actor_ref":"ctx:p45","contexto_version":1,"cuenta_ref":"cta_pppppppppppppppppppppppp","cuenta_version":1,"perfil_ref":"prf_pppppppppppppppppppppppp","perfil_version":1,"persona_ref":"per_pppppppppppppppppppppppp","persona_version":1}';
 x jsonb:='{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_pppppppppppppppppppppppp","contexto_actor_ref":"ctx:p45","contexto_version":1,"cuenta_ref":"cta_pppppppppppppppppppppppp","cuenta_version":1,"perfil_activo_ref":"prf_pppppppppppppppppppppppp","persona_ref":"per_pppppppppppppppppppppppp","persona_version":1,"perfil_version":1}';
 datos text:=$datos${"idempotencia_ref":"45000000-0000-4000-8000-000000000001","origen_ct_ref":"ct:p45","origen_ct_recibo_ref":"ctrec:p45","origen_ct_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","expediente_ref":"exp:p45","expediente_version":7,"organismo_ref":"org:p45","unidad_ref":"uni:p45","persona_ref":"per_pppppppppppppppppppppppp","persona_version":1,"fuente_bolsa_ref":"bolsa:p45","fuente_bolsa_version":1,"fuente_bolsa_recibo_ref":"bolsarec:p45","fuente_bolsa_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","regimen":{"ref":"reg:p45","version":1},"modalidad":{"ref":"mod:p45","version":1},"desde":"2026-10-11","hasta":"","plaza_ref":"plaza:45000000-0000-4000-8000-000000000003","puesto_ref":"puesto:45000000-0000-4000-8000-000000000004","clase_ocupacion":"temporal","version_plantilla_ref":"plantilla:p45","version_rpt_ref":"rpt:p45","revision_plaza":1,"revision_puesto":1,"fuente_organizacion_ref":"organizacion:p45","fuente_organizacion_huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","catalogo_rpt_id":"rpt:p45","catalogo_rpt_modulo":"personal","catalogo_rpt_categoria":"categoria:p45","catalogo_rpt_version":1,"catalogo_rpt_huella_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","vinculo_ct_recibo_ref":"vinculo:p45","procedencia":{"acto_ref":"acto:p45","fuente_ref":"fuente:p45","fuente_version":1,"fuente_huella_sha256":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","idempotencia_ref":"45000000-0000-4000-8000-000000000005"}}$datos$;
 material text; mh text; rh text; op text; sel text; raw text; negocio text;
 c jsonb; d jsonb; campos jsonb; codigo text; err text;
BEGIN
 FOREACH op IN ARRAY ARRAY['seleccionar','preparar','ejecutar','confirmar'] LOOP
  sel:=CASE op WHEN 'seleccionar' THEN 'plaza:45000000-0000-4000-8000-000000000003'
   WHEN 'preparar' THEN '45000000-0000-4000-8000-000000000001' ELSE 'perplan_'||md5('plan:p45:1') END;
  raw:=CASE WHEN op='preparar' THEN datos ELSE 'null' END;
  negocio:=CASE WHEN op='preparar' THEN encode(sha256(convert_to(datos,'UTF8')),'hex') ELSE '' END;
  material:='{"esquema":"vec.personal.plan-incorporacion-ct.v1","operacion":"'||op||
   '","plan_ref":"'||sel||'","organismo_ref":"org:p45","datos":'||raw||',"negocio_sha256":"'||negocio||'","actor":'||actor||'}';
  IF op='seleccionar' THEN material:=left(material,length(material)-1)||',"seleccion":{"plaza_ref":"'||sel||'","puesto_ref":"puesto:45000000-0000-4000-8000-000000000004","desde":"2026-10-11"}}'; END IF;
  mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
  rh:=encode(sha256(convert_to('{"ambitos":{"organismo_ref":"org:p45"},"atributos":{"material_sha256":"'||mh||'","objetivo_ref":"'||sel||'","operacion":"'||op||'"}}','UTF8')),'hex');
  campos:=CASE WHEN op='seleccionar' THEN '["evidencia","seleccion"]'::jsonb ELSE '["ejecucion_huella_sha256","ejecucion_recibo_ref","estado","evidencia","plan","recibo_alta_relacion","recibo_ocupacion"]'::jsonb END;
  c:=jsonb_build_object('operacion','personal.plan_incorporacion_ct.'||op,'audiencia_consumo','vec_personal.plan_incorporacion_ct.v1',
   'efecto_ref',sel,'huella_efecto_sha256',rh,'emitida_en','2026-01-01T00:00:00Z','expira_en','2100-01-01T00:00:00Z','decision_valida_hasta','2100-01-01T00:00:00Z');
  d:=jsonb_build_object('principal_id','per_pppppppppppppppppppppppp','perfil_activo_ref','prf_pppppppppppppppppppppppp',
   'concedida',true,'modulo_id','personal','obligaciones','[]'::jsonb,'tipo_recurso','plan_incorporacion_ct',
   'finalidad','gestionar_incorporacion_ct','campos_permitidos',campos,'accion','personal.plan_incorporacion_ct.'||op,
   'recurso_ref',sel,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:p45:'||op,'valida_hasta','2100-01-01T00:00:00Z');
  codigo:='ok';err:='';
  BEGIN
   PERFORM vec_personal.plan_incorporacion_ct_v1(material,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),
    '{}'::text::bytea,convert_to(x::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,err=MESSAGE_TEXT;
  END;
  IF codigo IS DISTINCT FROM '22023' OR err IS DISTINCT FROM 'entrada VEC-AD-3 inválida' THEN
   RAISE EXCEPTION 'P45: % no alcanzó el núcleo V3: % %',op,codigo,err; END IF;
  RAISE NOTICE 'P45_FRONTERA %: %; no es positivo nominal',op,err;
 END LOOP;
END $frontera_plan$;
RESET SESSION AUTHORIZATION;
DO $sin_efectos$
DECLARE a record;
BEGIN
 SELECT * INTO STRICT a FROM p45_cuentas;
 IF (SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo)<>a.recibos+1
    OR (SELECT count(*) FROM vec_personal.relacion_servicio_historia)<>a.relaciones+1
    OR (SELECT count(*) FROM vec_personal.ocupacion_empleado_historia)<>a.ocupaciones
    OR (SELECT count(*) FROM vec_personal.acceso_plan_incorporacion_ct)<>a.accesos THEN
  RAISE EXCEPTION 'P45: una denegación produjo efectos'; END IF;
END $sin_efectos$;
ROLLBACK;
\echo P45_FOCAL_OK
