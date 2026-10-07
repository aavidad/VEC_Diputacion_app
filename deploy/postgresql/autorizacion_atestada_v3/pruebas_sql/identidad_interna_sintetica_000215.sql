\set ON_ERROR_STOP on
-- Vector focal para el clon desechable PG18 con AD207 + AD215 + AUT57.
-- Ejecuta Dirección como superusuario, con sellador AD207 activo y latido vigente.
-- El envoltorio y el LOGIN sintéticos sólo existen dentro de este ROLLBACK.
-- Verifica la familia AD; no sustituye el recorrido AUT57/CA37/IS17 ni el
-- verificador offline, la prueba de concurrencia o la recuperación tras reinicio.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regrole('ad215_prueba_operador') IS NOT NULL
 OR pg_catalog.to_regrole('vec_identidad_interna_sintetica_ejecutor') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3._probar_ad215_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'AD215 prueba: entorno incompatible'; END IF;
END $pre$;
CREATE ROLE ad215_prueba_operador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_interna_sintetica_ejecutor TO ad215_prueba_operador WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO ad215_prueba_operador;

CREATE FUNCTION vec_autorizacion_atestada_v3._probar_ad215_v1()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $prueba$
DECLARE
 e jsonb;i jsonb;x record;y record;a record;v record;k text;material bytea;huella text;
 corte jsonb;historia jsonb;familia text;dominio text;orden text[];n integer:=0;
 orden_efecto constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','operacion_ref','plan_ref','plan_sha256',
 'preimagen_sha256','configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref',
 'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];
 orden_intento constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256',
 'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
BEGIN
 IF session_user<>'ad215_prueba_operador' THEN RAISE EXCEPTION 'AD215 prueba: LOGIN equivocado'; END IF;
 -- El ejecutor y el LOGIN no pueden escribir directamente en AD ni leer tabla.
 FOREACH familia IN ARRAY ARRAY['registrar_provision_identidad_interna_sintetica_v1','registrar_intento_identidad_interna_sintetica_v1'] LOOP
  IF pg_catalog.has_function_privilege(session_user,'vec_autorizacion_atestada_v3.'||familia||'(jsonb)','EXECUTE')
  OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.'||familia||'(jsonb)','EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
    pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) ac
    WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.'||familia||'(jsonb)')
    AND (ac.grantee NOT IN (p.proowner,'vec_autorizacion_propietario'::regrole) OR ac.is_grantable OR ac.privilege_type<>'EXECUTE'))
  THEN RAISE EXCEPTION 'AD215 prueba: ACL ampliada'; END IF;
 END LOOP;
 IF pg_catalog.has_table_privilege(session_user,'vec_autorizacion_atestada_v3.auditoria_consumo_v3','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 THEN RAISE EXCEPTION 'AD215 prueba: tabla expuesta'; END IF;
 SELECT pg_catalog.to_jsonb(c) INTO STRICT corte FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(a) ORDER BY a.secuencia),'[]'::jsonb) INTO historia
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a;
 e:=pg_catalog.jsonb_build_object(
 'tipo_registro','provision_identidad_interna_sintetica','evento_ref','evento_21500000000000000000000000000001',
 'operador_login',session_user::text,'operacion_ref','piis_21500000000000000000000000000001',
 'plan_ref','plan:ad215:sintetico','plan_sha256',pg_catalog.repeat('1',64),
 'preimagen_sha256',pg_catalog.repeat('2',64),'configuracion_sha256',pg_catalog.repeat('3',64),
 'aprobacion_ref','aprobacion:ad215:sintetica','alcance_fuente','sintetico_declarado',
 'accion','provisionar_identidad_interna_sintetica_v1',
 'recurso_ref','identidad_interna_sintetica:piis_21500000000000000000000000000001',
 'resultado','permitido','motivo_ref','identidad_interna_registrada','proceso','postgresql',
 'canal','operacion_tecnica_privada','finalidad_ref','identidad_interna_sintetica',
 'correlacion_ref','correlacion_21500000000000000000000000000001',
 'fuente_ref','fuente:ad215:sintetica','fuente_sha256',pg_catalog.repeat('4',64));
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e);
 SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e);
 IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y)
 THEN RAISE EXCEPTION 'AD215 prueba: replay cambió recibo'; END IF;
 BEGIN
  INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3
  SELECT (pg_catalog.jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,
   pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('actor_ref','per_21500000000000000000000000000001'))).*
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD215 prueba: familia mezclada admitida';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e||pg_catalog.jsonb_build_object('fuente_sha256',pg_catalog.repeat('5',64)));
  RAISE EXCEPTION 'AD215 prueba: material alterado admitido';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e||'{"evento_ref":"evento_21500000000000000000000000000002"}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: operación duplicada admitida';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e||'{"nombre":"dato_sintetico_excluido"}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: campo personal admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e||'{"operador_login":"otro_operador"}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: operador suplantado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e-'plan_ref');
  RAISE EXCEPTION 'AD215 prueba: plan ausente admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(e||'{"plan_ref":null}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: plan nulo admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 i:=pg_catalog.jsonb_build_object('tipo_registro','intento_identidad_interna_sintetica',
 'evento_ref','evento_21500000000000000000000000000003','operador_login',session_user::text,
 'solicitud_sha256',pg_catalog.repeat('6',64),'accion','provisionar_identidad_interna_sintetica_v1',
 'recurso_ref','solicitud_identidad_interna:21500000000000000000000000000001',
 'resultado','permitido','motivo_ref','identidad_interna_registrada','proceso','postgresql',
 'canal','operacion_tecnica_privada','finalidad_ref','identidad_interna_sintetica',
 'correlacion_ref','correlacion_21500000000000000000000000000001');
 FOR v IN SELECT * FROM (VALUES
 ('provisionar_identidad_interna_sintetica_v1','permitido','identidad_interna_registrada'),
 ('provisionar_identidad_interna_sintetica_v1','permitido','identidad_interna_replay'),
 ('recuperar_identidad_interna_sintetica_v1','permitido','identidad_interna_recuperada'),
 ('provisionar_identidad_interna_sintetica_v1','denegado','identidad_interna_denegada'),
 ('recuperar_identidad_interna_sintetica_v1','denegado','identidad_interna_denegada'),
 ('provisionar_identidad_interna_sintetica_v1','error','identidad_interna_error'),
 ('recuperar_identidad_interna_sintetica_v1','error','identidad_interna_error')) z(accion,resultado,motivo)
 LOOP
  n:=n+1;
  i:=i||pg_catalog.jsonb_build_object('evento_ref','evento_215000000000000000000000000000'||pg_catalog.lpad((n+10)::text,2,'0'),
   'accion',v.accion,'resultado',v.resultado,'motivo_ref',v.motivo);
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(i);
  SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(i);
  IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y)
  THEN RAISE EXCEPTION 'AD215 prueba: replay de asiento cambió recibo'; END IF;
 END LOOP;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(i||'{"accion":"recuperar_identidad_interna_sintetica_v1","resultado":"permitido","motivo_ref":"identidad_interna_registrada"}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: motivo de alta admitido en lectura';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(i||'{"motivo_ref":"texto libre"}'::jsonb);
  RAISE EXCEPTION 'AD215 prueba: motivo libre admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE operador_login=session_user
 AND tipo_registro IN ('provision_identidad_interna_sintetica','intento_identidad_interna_sintetica'))<>8
 THEN RAISE EXCEPTION 'AD215 prueba: número de efectos/intentos incorrecto'; END IF;
 -- Comprueba el encuadre de material y huella de ambos tipos desde las columnas.
 FOR a IN SELECT * FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE operador_login=session_user
 AND tipo_registro IN ('provision_identidad_interna_sintetica','intento_identidad_interna_sintetica') LOOP
  IF a.tipo_registro='provision_identidad_interna_sintetica' THEN
   orden:=orden_efecto;dominio:='identidad-interna-sintetica.v1';
   e:=pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('operacion_ref',a.identidad_operacion_ref,
    'plan_ref',a.identidad_plan_ref,'preimagen_sha256',a.identidad_preimagen_sha256,
    'configuracion_sha256',a.identidad_configuracion_sha256,'alcance_fuente',a.identidad_alcance);
  ELSE
   orden:=orden_intento;dominio:='intento-identidad-interna-sintetica.v1';
   e:=pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('solicitud_sha256',a.identidad_solicitud_sha256);
  END IF;
  material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.'||dominio);
  FOREACH k IN ARRAY orden LOOP material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(e->>k); END LOOP;
  IF a.evento_material_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(material),'hex')
  THEN RAISE EXCEPTION 'AD215 prueba: material no ligado'; END IF;
  huella:=pg_catalog.encode(pg_catalog.sha256(
   vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.'||dominio)||
   vec_autorizacion_atestada_v3.encuadrar_mac(a.secuencia::text)||vec_autorizacion_atestada_v3.encuadrar_mac(a.anterior_sha256)||
   vec_autorizacion_atestada_v3.encuadrar_mac(a.auditoria_ref)||vec_autorizacion_atestada_v3.encuadrar_mac(a.evento_material_sha256)||
   vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
  IF a.huella_sha256 IS DISTINCT FROM huella OR a.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64)
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 p WHERE p.secuencia=a.secuencia)
  THEN RAISE EXCEPTION 'AD215 prueba: asiento fuera de cadena AD207'; END IF;
 END LOOP;
 IF (SELECT pg_catalog.to_jsonb(c) FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c) IS DISTINCT FROM corte
 OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(a) ORDER BY a.secuencia),'[]'::jsonb)
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE operador_login IS DISTINCT FROM session_user) IS DISTINCT FROM historia
 THEN RAISE EXCEPTION 'AD215 prueba: historia previa modificada'; END IF;
 RAISE NOTICE 'AD215: efecto único, siete intentos, ACL, ABI, replay, conflictos, material y cola AD207 comprobados';
END $prueba$;
ALTER FUNCTION vec_autorizacion_atestada_v3._probar_ad215_v1() OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3._probar_ad215_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3._probar_ad215_v1() TO ad215_prueba_operador;
SET LOCAL SESSION AUTHORIZATION ad215_prueba_operador;
SELECT vec_autorizacion_atestada_v3._probar_ad215_v1();
RESET SESSION AUTHORIZATION;
ROLLBACK;
