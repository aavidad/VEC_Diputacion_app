\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
SET LOCAL ROLE vec_administracion_copias_propietario;
DO $plan$
DECLARE p jsonb;m jsonb;x jsonb;
BEGIN
 m:=jsonb_build_object('solicitud_sha256',repeat('a',64),'orden','orden:sintetica','operacion','operacion:sintetica',
  'accion','restaurar_conjunto','conjunto','conjunto:sintetico','manifiesto_sha256',repeat('b',64),
  'preimagen_sha256',repeat('c',64),'destino','destino:sintetico','proponente_persona','persona:uno',
  'aprobador_persona','persona:dos','politica','politica:sintetica','politica_sha256',repeat('d',64),
  'epoca','epoca:sintetica','fence',1,'version_cas',1,'emitida_en','2026-10-01T00:00:00Z','caduca_en','2026-10-02T00:00:00Z');
 p:=jsonb_build_object('esquema','vec.administracion.plan-orden-copias.v1','datos',m);
 IF vec_administracion_copias.plan_v1(convert_to(p::text,'UTF8')) IS DISTINCT FROM m THEN RAISE EXCEPTION 'test_cs08_plan_perdida'; END IF;
 FOR x IN SELECT p||'{"otro":true}'::jsonb UNION ALL SELECT jsonb_set(p,'{datos,aprobador_persona}','"persona:uno"')
  UNION ALL SELECT jsonb_set(p,'{datos,fence}','null') UNION ALL SELECT jsonb_set(p,'{datos,fence}','1.2')
  UNION ALL SELECT jsonb_set(p,'{datos,destino}','null') UNION ALL SELECT jsonb_set(p,'{datos,manifiesto_sha256}','"alterada"') LOOP
  BEGIN
   PERFORM vec_administracion_copias.plan_v1(convert_to(x::text,'UTF8'));
   RAISE EXCEPTION 'test_cs08_plan_invalido_aceptado';
  EXCEPTION WHEN invalid_parameter_value THEN NULL;
  END;
 END LOOP;
END $plan$;
ROLLBACK;
SELECT 'CS08_PLAN_CERRADO_OK';
