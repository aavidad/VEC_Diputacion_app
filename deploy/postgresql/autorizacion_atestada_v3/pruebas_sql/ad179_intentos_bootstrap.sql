\set ON_ERROR_STOP on
-- Clon desechable AD179 nueva candidata. Sin DOWN, todo hace ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';
CREATE TEMP TABLE ad179_preimagen ON COMMIT DROP AS
 SELECT auditoria_ref,pg_catalog.to_jsonb(a) AS fila
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a;
GRANT SELECT ON TABLE pg_temp.ad179_preimagen TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE e jsonb;x record;y record;v_secuencia numeric;v_total integer:=0;f oid;v jsonb;m bytea;h text;k text;
 orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256','accion',
  'recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
BEGIN
 f:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(jsonb)');
 IF f IS NULL OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
  LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,
   'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_personal_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD179 intento prueba: ACL privada divergente'; END IF;
 SELECT secuencia INTO STRICT v_secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 FOR v IN SELECT value FROM pg_catalog.jsonb_array_elements($vectores${
  "vectores": [
    {
      "evento": {
        "tipo_registro": "intento_bootstrap_central_admin",
        "evento_ref": "evento_000000000000000000000000000045ed",
        "operador_login": "operador_arranque_sintetico",
        "solicitud_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "accion": "registrar_bootstrap_central_admin_v3",
        "recurso_ref": "solicitud_bootstrap:00000000000000000000000000000001",
        "resultado": "permitido",
        "motivo_ref": "bootstrap_registrado",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "bootstrap_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000045ed"
      },
      "secuencia": 1,
      "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "auditoria_ref": "aud_v3_bi_000000000000000000000000000045ed",
      "registrada_en": "2026-10-04T01:31:00.123456Z",
      "evento_material_sha256": "609c75445583f9adc4adde92fa2c97ba4012c9992ffaa1382c5cd4d0c56335ad",
      "huella_sha256": "1bbf8b7f8a90ca404a132a44c77c7fcef72eb3797daa968b57d7620ee1326a08"
    },
    {
      "evento": {
        "tipo_registro": "intento_bootstrap_central_admin",
        "evento_ref": "evento_000000000000000000000000000045ee",
        "operador_login": "operador_ámbito_sintético",
        "solicitud_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "accion": "registrar_bootstrap_central_admin_v3",
        "recurso_ref": "solicitud_bootstrap:00000000000000000000000000000002",
        "resultado": "permitido",
        "motivo_ref": "bootstrap_replay",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "bootstrap_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000045ee"
      },
      "secuencia": 2,
      "anterior_sha256": "1bbf8b7f8a90ca404a132a44c77c7fcef72eb3797daa968b57d7620ee1326a08",
      "auditoria_ref": "aud_v3_bi_000000000000000000000000000045ee",
      "registrada_en": "2026-10-04T01:31:00.123456Z",
      "evento_material_sha256": "fdc8ccb236da9f7ec61291587d9e87bd6222f4990f4d7c47aa8f1cee3238c0f2",
      "huella_sha256": "09b2cbd350bbb867180d8fed2fc6229d515b65a775cc5ecc980d6f3bbf1c4f43"
    },
    {
      "evento": {
        "tipo_registro": "intento_bootstrap_central_admin",
        "evento_ref": "evento_000000000000000000000000000045ef",
        "operador_login": "operador_arranque_sintetico",
        "solicitud_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "accion": "registrar_bootstrap_central_admin_v3",
        "recurso_ref": "solicitud_bootstrap:00000000000000000000000000000003",
        "resultado": "denegado",
        "motivo_ref": "bootstrap_denegado",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "bootstrap_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000045ef"
      },
      "secuencia": 3,
      "anterior_sha256": "09b2cbd350bbb867180d8fed2fc6229d515b65a775cc5ecc980d6f3bbf1c4f43",
      "auditoria_ref": "aud_v3_bi_000000000000000000000000000045ef",
      "registrada_en": "2026-10-04T01:31:00.123456Z",
      "evento_material_sha256": "f0e36c0b1c430ffbbbf10c81ebec0af65201ea780c6300c3e7812b8ebc69cafb",
      "huella_sha256": "a4f0de1d62620593c219d8cd3f2a22ef70ec9aa50c07377f5b70396fd263e7f1"
    },
    {
      "evento": {
        "tipo_registro": "intento_bootstrap_central_admin",
        "evento_ref": "evento_000000000000000000000000000045f0",
        "operador_login": "operador_arranque_sintetico",
        "solicitud_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "accion": "registrar_bootstrap_central_admin_v3",
        "recurso_ref": "solicitud_bootstrap:00000000000000000000000000000004",
        "resultado": "error",
        "motivo_ref": "bootstrap_error",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "bootstrap_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000045f0"
      },
      "secuencia": 4,
      "anterior_sha256": "a4f0de1d62620593c219d8cd3f2a22ef70ec9aa50c07377f5b70396fd263e7f1",
      "auditoria_ref": "aud_v3_bi_000000000000000000000000000045f0",
      "registrada_en": "2026-10-04T01:31:00.123456Z",
      "evento_material_sha256": "1fa7f8f3289895368be32c01e56d1733a8e8231cf84052825f1a98ae33c15065",
      "huella_sha256": "2ed908594b00fdcd7e0889c0ee11acfb5f4f3aa69d0c852f62aea6b08b7ce098"
    }
  ]
}$vectores$::jsonb->'vectores') LOOP
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-bootstrap-central-admin.v1');
  FOREACH k IN ARRAY orden LOOP m:=m||vec_autorizacion_atestada_v3.encuadrar_mac(v->'evento'->>k); END LOOP;
  h:=pg_catalog.encode(pg_catalog.sha256(m),'hex');
  IF h IS DISTINCT FROM v->>'evento_material_sha256' THEN RAISE EXCEPTION 'AD179 intento prueba: vector material divergente'; END IF;
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-bootstrap-central-admin.v1')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'secuencia')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'anterior_sha256')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'auditoria_ref')||vec_autorizacion_atestada_v3.encuadrar_mac(h)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'registrada_en');
  IF pg_catalog.encode(pg_catalog.sha256(m),'hex') IS DISTINCT FROM v->>'huella_sha256'
  THEN RAISE EXCEPTION 'AD179 intento prueba: vector eslabón divergente'; END IF;
  e:=(v->'evento')||pg_catalog.jsonb_build_object('operador_login',session_user::text);
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e);
  SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e);
  v_total:=v_total+1;
  IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y) OR x.secuencia<>v_secuencia+v_total
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
   WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='intento_bootstrap_central_admin'
    AND a.operador_login=session_user::name AND a.bootstrap_solicitud_sha256=e->>'solicitud_sha256'
    AND a.resultado=e->>'resultado' AND a.motivo_ref=e->>'motivo_ref' AND a.proceso='postgresql'
    AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL
    AND a.fuente_ref IS NULL AND a.fuente_sha256 IS NULL AND a.plan_sha256 IS NULL AND a.aprobacion_ref IS NULL
    AND a.fuentes_plan_ref IS NULL AND a.fuentes_preimagen_sha256 IS NULL AND a.fuentes_configuracion_sha256 IS NULL
    AND a.unidad_plan_ref IS NULL AND a.unidad_solicitud_sha256 IS NULL AND a.unidad_recibo_ref IS NULL)
  THEN RAISE EXCEPTION 'AD179 intento prueba: recibo/replay/familia divergente'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='bootstrap_operador') THEN
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||pg_catalog.jsonb_build_object('evento_ref',
    (SELECT evento_ref FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='bootstrap_operador' ORDER BY secuencia LIMIT 1)));
   RAISE EXCEPTION 'AD179 prueba: reutilizó un evento confirmado AD171';
  EXCEPTION WHEN unique_violation THEN NULL; END;
 END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||pg_catalog.jsonb_build_object('solicitud_sha256',pg_catalog.repeat('9',64)));
  RAISE EXCEPTION 'AD179 intento prueba: aceptó material cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 FOR k IN SELECT value FROM pg_catalog.jsonb_array_elements_text('["actor_ref","perfil_activo_ref","fuente_ref","preimagen_sha256","aprobacion_ref","recibo_ref"]'::jsonb) LOOP
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||pg_catalog.jsonb_build_object(k,'inventado'));
   RAISE EXCEPTION 'AD179 intento prueba: aceptó campo cruzado';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||'{"solicitud_sha256":null}'::jsonb);
  RAISE EXCEPTION 'AD179 intento prueba: aceptó solicitud nula';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||'{"operador_login":"otro_operador"}'::jsonb);
  RAISE EXCEPTION 'AD179 intento prueba: aceptó LOGIN ajeno';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||'{"resultado":"permitido"}'::jsonb);
  RAISE EXCEPTION 'AD179 intento prueba: aceptó resultado y motivo cruzados';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||'{"motivo_ref":"libre"}'::jsonb);
  RAISE EXCEPTION 'AD179 intento prueba: aceptó motivo libre';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(e||'{"proceso":"cli"}'::jsonb);
  RAISE EXCEPTION 'AD179 intento prueba: atribuyó observador no acreditado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND attname='version_consumo' AND NOT attisdropped) THEN
  BEGIN
   EXECUTE 'INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 SELECT (pg_catalog.jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)).*'
   USING (SELECT pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('version_consumo',3,'auditoria_ref','aud_v3_bi_000000000000000000000000000a0179',
    'evento_ref','evento_000000000000000000000000000a0179','secuencia',v_secuencia+v_total+1) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref);
   RAISE EXCEPTION 'AD179 prueba: intento técnico aceptó version_consumo nominal';
  EXCEPTION WHEN check_violation THEN NULL; END;
 END IF;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET bootstrap_solicitud_sha256=pg_catalog.repeat('9',64) WHERE auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD179 intento prueba: permitió mutar auditoría';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF v_total<>4 OR (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>v_secuencia+v_total
 OR EXISTS(SELECT 1 FROM pg_temp.ad179_preimagen p
   LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref)
   WHERE pg_catalog.to_jsonb(a) IS DISTINCT FROM p.fila)
 THEN RAISE EXCEPTION 'AD179 intento prueba: cadena o historia divergente tras rechazos'; END IF;
END $prueba$;
ROLLBACK;
