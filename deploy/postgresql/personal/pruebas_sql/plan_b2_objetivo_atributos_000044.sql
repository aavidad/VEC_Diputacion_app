\set ON_ERROR_STOP on
-- Personal44 en un clon con el núcleo V3 real instalado (no lo sustituye).
-- Lo ejecuta un LOGIN miembro de vec_personal_ejecutor (por ejemplo, el de
-- planes B2), nunca el propietario ni el migrador:
--   psql -X -U <login_planes_b2> -d <base> -f plan_b2_objetivo_atributos_000044.sql
-- Todo termina en ROLLBACK. Las decisiones no van firmadas: lo que se prueba
-- es la comprobación propia de Personal antes del núcleo.
--  1. Una decisión con el canon anterior (plan en ámbitos) o con otro plan en
--     los atributos se rechaza en Personal («permiso nominal divergente»).
--  2. Con el canon nuevo, Personal la deja pasar y el núcleo la rechaza por
--     la firma, ya no por «límites VEC-AD-3 ausentes» (statement_timeout 15 s).
--  3. La lectura de clases conserva el canon con el objetivo en ámbitos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL lock_timeout='2s';
SET LOCAL timezone='UTC';
DO $p44$
DECLARE
 org text:='org_sintetico_p44';
 plan text:='perplan_'||repeat('a',32);
 otro text:='perplan_'||repeat('b',32);
 actor text:='{"actor_ref":"per_sintetica_p44_000000000000001","contexto_actor_ref":"ctx:sintetico:p44","contexto_version":1,"cuenta_ref":"cta_sintetica_p44_000000000000001","cuenta_version":1,"perfil_ref":"prf_sintetico_p44_000000000000001","perfil_version":1,"persona_ref":"per_sintetica_p44_000000000000001","persona_version":1}';
 x jsonb:='{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_sintetica_p44_000000000000001","contexto_actor_ref":"ctx:sintetico:p44","contexto_version":1,"cuenta_ref":"cta_sintetica_p44_000000000000001","cuenta_version":1,"perfil_activo_ref":"prf_sintetico_p44_000000000000001","persona_ref":"per_sintetica_p44_000000000000001","persona_version":1,"perfil_version":1}';
 material text; mh text; rh text; op text; sel text; campos jsonb; err text; caso record;
BEGIN
 IF (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid='vec_personal.plan_incorporacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid='vec_personal.registrar_acto_plan_incorporacion_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
      IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s'] THEN
  RAISE EXCEPTION 'Personal44: statement_timeout de las fachadas del plan no es 15 s'; END IF;
 FOR caso IN SELECT * FROM (VALUES
   ('consultar','ambitos',false),('consultar','otro',false),('consultar','atributos',true),
   ('clases_ocupacion','ambitos',true),('clases_ocupacion','atributos',false)) v(op,forma,pasa) LOOP
  op:=caso.op;
  sel:=CASE WHEN op='clases_ocupacion' THEN org ELSE plan END;
  material:='{"esquema":"vec.personal.plan-incorporacion-ct.v1","operacion":"'||op||'","plan_ref":"'||sel||
   '","organismo_ref":"'||org||'","datos":null,"negocio_sha256":"","actor":'||actor||'}';
  mh:=encode(sha256(convert_to(material,'UTF8')),'hex');
  rh:=encode(sha256(convert_to(CASE caso.forma
   WHEN 'ambitos' THEN '{"ambitos":{"objetivo_ref":"'||sel||'","organismo_ref":"'||org||'"},"atributos":{"material_sha256":"'||mh||'","operacion":"'||op||'"}}'
   WHEN 'otro' THEN '{"ambitos":{"organismo_ref":"'||org||'"},"atributos":{"material_sha256":"'||mh||'","objetivo_ref":"'||otro||'","operacion":"'||op||'"}}'
   ELSE '{"ambitos":{"organismo_ref":"'||org||'"},"atributos":{"material_sha256":"'||mh||'","objetivo_ref":"'||sel||'","operacion":"'||op||'"}}' END,'UTF8')),'hex');
  campos:=CASE WHEN op='clases_ocupacion' THEN '["catalogo","evidencia"]'::jsonb
   ELSE '["ejecucion_huella_sha256","ejecucion_recibo_ref","estado","evidencia","plan","recibo_alta_relacion","recibo_ocupacion"]'::jsonb END;
  BEGIN
   PERFORM vec_personal.plan_incorporacion_ct_v1(material,
    convert_to(jsonb_build_object('operacion','personal.plan_incorporacion_ct.'||op,'audiencia_consumo','vec_personal.plan_incorporacion_ct.v1',
      'efecto_ref',sel,'huella_efecto_sha256',rh,'emitida_en','2026-01-01T00:00:00Z','expira_en','2100-01-01T00:00:00Z',
      'decision_valida_hasta','2100-01-01T00:00:00Z')::text,'UTF8'),
    convert_to(jsonb_build_object('principal_id','per_sintetica_p44_000000000000001','perfil_activo_ref','prf_sintetico_p44_000000000000001',
      'concedida',true,'modulo_id','personal','obligaciones','[]'::jsonb,'tipo_recurso','plan_incorporacion_ct',
      'finalidad','gestionar_incorporacion_ct','campos_permitidos',campos,'accion','personal.plan_incorporacion_ct.'||op,
      'recurso_ref',sel,'contexto_recurso_huella_sha256',rh,'decision_ref','decision:sintetica:p44','valida_hasta','2100-01-01T00:00:00Z')::text,'UTF8'),
    convert_to('{}','UTF8'),convert_to(x::text,'UTF8'),1,1,'x'::bytea,'y'::bytea,'z'::bytea,'w'::bytea);
   RAISE EXCEPTION 'Personal44: una decisión sin firmar llegó a consumirse (% %)',op,caso.forma;
  EXCEPTION WHEN others THEN
   GET STACKED DIAGNOSTICS err=MESSAGE_TEXT;
   IF err LIKE 'Personal44:%' THEN RAISE; END IF;
   IF caso.pasa AND (err='Personal23: permiso nominal divergente' OR err LIKE '%límites VEC-AD-3 ausentes%') THEN
    RAISE EXCEPTION 'Personal44: % con %: % (esperado: rechazo del núcleo por la firma)',op,caso.forma,err; END IF;
   IF NOT caso.pasa AND err IS DISTINCT FROM 'Personal23: permiso nominal divergente' THEN
    RAISE EXCEPTION 'Personal44: % con %: % (esperado: permiso nominal divergente)',op,caso.forma,err; END IF;
   RAISE NOTICE 'Personal44: % con canon %: %',op,caso.forma,err;
  END;
 END LOOP;
END $p44$;
ROLLBACK;
