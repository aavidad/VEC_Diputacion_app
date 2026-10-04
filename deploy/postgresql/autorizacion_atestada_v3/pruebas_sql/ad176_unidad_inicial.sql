\set ON_ERROR_STOP on
-- Clon desechable AD176: resultado, replay, ACL e historia. Todo hace ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';
CREATE TEMP TABLE ad176_preimagen ON COMMIT DROP AS
 SELECT auditoria_ref,pg_catalog.to_jsonb(a) AS fila FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a;
GRANT SELECT ON TABLE pg_temp.ad176_preimagen TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE e jsonb;x record;y record;v_confirmado jsonb;v_secuencia numeric;v_total integer:=0;f oid;v jsonb;m bytea;h text;k text;v_dominio text;
 orden text[];
BEGIN
 FOREACH k IN ARRAY ARRAY['registrar_unidad_inicial_personal_v1','registrar_intento_unidad_inicial_personal_v1'] LOOP
  f:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.'||k||'(jsonb)');
  IF f IS NULL OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
   LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_personal_propietario'::regrole)
    OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR NOT pg_catalog.has_function_privilege('vec_personal_propietario',f,'EXECUTE')
  OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD176 prueba: ACL privada divergente'; END IF;
 END LOOP;
 SELECT secuencia INTO STRICT v_secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 FOR v IN SELECT value FROM pg_catalog.jsonb_array_elements($vectores${
  "vectores": [
    {
      "evento": {
        "tipo_registro": "unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c1",
        "operador_login": "operador_unidad_sintetica",
        "plan_ref": "pui_aaaaaaaaaaaaaaaaaaaaaa1",
        "plan_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "preimagen_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "configuracion_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "aprobacion_ref": "aprobacion:unidad:1",
        "alcance_fuente": "sintetico_declarado",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "unidad:00000000-0000-0000-0000-000000000001",
        "resultado": "permitido",
        "motivo_ref": "unidad_registrada",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c1",
        "fuente_ref": "fuente_unidad:ad176:1",
        "fuente_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "recibo_ref": "recibo_unidad:00000000000000000000000000000001",
        "recibo_sha256": "5555555555555555555555555555555555555555555555555555555555555555"
      },
      "secuencia": 1,
      "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "auditoria_ref": "aud_v3_u_000000000000000000000000000044c1",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "eafbc73f273bad2c3e6e782b64efa90b83ce29bc580af6f07eedc8f9a4cbb3c7",
      "huella_sha256": "26f77559f22083d015c4481f6eed42a23302aa7461de1987c0986cae543f3001"
    },
    {
      "evento": {
        "tipo_registro": "unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c2",
        "operador_login": "operador_ámbito_sintético",
        "plan_ref": "pui_aaaaaaaaaaaaaaaaaaaaaa2",
        "plan_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "preimagen_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "configuracion_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "aprobacion_ref": "aprobacion:unidad:2",
        "alcance_fuente": "sintetico_declarado",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "unidad:00000000-0000-0000-0000-000000000002",
        "resultado": "permitido",
        "motivo_ref": "unidad_registrada",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c2",
        "fuente_ref": "fuente_unidad:ad176:2",
        "fuente_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "recibo_ref": "recibo_unidad:00000000000000000000000000000002",
        "recibo_sha256": "5555555555555555555555555555555555555555555555555555555555555555"
      },
      "secuencia": 2,
      "anterior_sha256": "26f77559f22083d015c4481f6eed42a23302aa7461de1987c0986cae543f3001",
      "auditoria_ref": "aud_v3_u_000000000000000000000000000044c2",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "dbc42bb94bab19d7826bc19ea104d1248b8438bd1d95dd550a66f85e4c6afbe4",
      "huella_sha256": "0621758f6f3ca7565e7cd4421575010e12694774e5d8d168c6b5f30d026a42fe"
    },
    {
      "evento": {
        "tipo_registro": "intento_unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c3",
        "operador_login": "operador_unidad_sintetica",
        "solicitud_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "solicitud_unidad:00000000000000000000000000000003",
        "resultado": "permitido",
        "motivo_ref": "unidad_registrada",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c3"
      },
      "secuencia": 3,
      "anterior_sha256": "0621758f6f3ca7565e7cd4421575010e12694774e5d8d168c6b5f30d026a42fe",
      "auditoria_ref": "aud_v3_ui_000000000000000000000000000044c3",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "e96e29babb5d36ef7703287167eecf3ad08321b99c8f2d5e4ac0e241722e04eb",
      "huella_sha256": "9a88ce7c29ede08e5e06ca9c4ba945e847d194a3ac417da83f3244861fa0613e"
    },
    {
      "evento": {
        "tipo_registro": "intento_unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c4",
        "operador_login": "operador_unidad_sintetica",
        "solicitud_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "solicitud_unidad:00000000000000000000000000000004",
        "resultado": "permitido",
        "motivo_ref": "unidad_replay",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c4"
      },
      "secuencia": 4,
      "anterior_sha256": "9a88ce7c29ede08e5e06ca9c4ba945e847d194a3ac417da83f3244861fa0613e",
      "auditoria_ref": "aud_v3_ui_000000000000000000000000000044c4",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "21a5395b6d3249b689142a1167803f96f3b67308561210a2a340961df8dbf586",
      "huella_sha256": "78fc151e35a4e0e1d66ac22e6012fa3e2a3236e44d95f8f06b93eeb44e8a3ff4"
    },
    {
      "evento": {
        "tipo_registro": "intento_unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c5",
        "operador_login": "operador_unidad_sintetica",
        "solicitud_sha256": "5555555555555555555555555555555555555555555555555555555555555555",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "solicitud_unidad:00000000000000000000000000000005",
        "resultado": "denegado",
        "motivo_ref": "unidad_denegada",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c5"
      },
      "secuencia": 5,
      "anterior_sha256": "78fc151e35a4e0e1d66ac22e6012fa3e2a3236e44d95f8f06b93eeb44e8a3ff4",
      "auditoria_ref": "aud_v3_ui_000000000000000000000000000044c5",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "4ecf8546636e2ad19b6663943b494b6ae15d7163cb43ed427edf601749861a9c",
      "huella_sha256": "cf177dfcfd884623ca55dca31df94f6356d7533f3fe9dc93a9655f5b98bce067"
    },
    {
      "evento": {
        "tipo_registro": "intento_unidad_inicial_personal",
        "evento_ref": "evento_000000000000000000000000000044c6",
        "operador_login": "operador_unidad_sintetica",
        "solicitud_sha256": "6666666666666666666666666666666666666666666666666666666666666666",
        "accion": "inicializar_unidad_sintetica_admin_v1",
        "recurso_ref": "solicitud_unidad:00000000000000000000000000000006",
        "resultado": "error",
        "motivo_ref": "unidad_error",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "inicializar_unidad_sintetica_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000044c6"
      },
      "secuencia": 6,
      "anterior_sha256": "cf177dfcfd884623ca55dca31df94f6356d7533f3fe9dc93a9655f5b98bce067",
      "auditoria_ref": "aud_v3_ui_000000000000000000000000000044c6",
      "registrada_en": "2026-10-04T00:00:01.123456Z",
      "evento_material_sha256": "b9d645cf98310ab4a3f7a346000d036dda6d5b8e4292f9b6c2eb859466e4e637",
      "huella_sha256": "8f757414a6bb61974bcb62fdd2863ac88cf34bb3842ba5985983df1e15bad5e4"
    }
  ]
}$vectores$::jsonb->'vectores') LOOP
  e:=v->'evento';
  IF e->>'tipo_registro'='unidad_inicial_personal' THEN
   orden:=ARRAY['tipo_registro','evento_ref','operador_login','plan_ref','plan_sha256','preimagen_sha256',
    'configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref','resultado','motivo_ref',
    'proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256','recibo_ref','recibo_sha256'];
   v_dominio:='unidad-inicial-personal.v1';
  ELSE
   orden:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256','accion','recurso_ref',
    'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
   v_dominio:='intento-unidad-inicial-personal.v1';
  END IF;
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.'||v_dominio);
  FOREACH k IN ARRAY orden LOOP m:=m||vec_autorizacion_atestada_v3.encuadrar_mac(e->>k); END LOOP;
  h:=pg_catalog.encode(pg_catalog.sha256(m),'hex');
  IF h IS DISTINCT FROM v->>'evento_material_sha256' THEN RAISE EXCEPTION 'AD176 prueba: vector material divergente'; END IF;
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.'||v_dominio)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'secuencia')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'anterior_sha256')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'auditoria_ref')||vec_autorizacion_atestada_v3.encuadrar_mac(h)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'registrada_en');
  IF pg_catalog.encode(pg_catalog.sha256(m),'hex') IS DISTINCT FROM v->>'huella_sha256'
  THEN RAISE EXCEPTION 'AD176 prueba: vector eslabón divergente'; END IF;
  e:=e||pg_catalog.jsonb_build_object('operador_login',session_user::text);
  IF e->>'tipo_registro'='unidad_inicial_personal' THEN
   v_confirmado:=e;
   SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(e);
   SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(e);
   IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref
     AND a.unidad_recibo_ref=e->>'recibo_ref' AND a.unidad_recibo_sha256=e->>'recibo_sha256' AND a.unidad_plan_ref=e->>'plan_ref'
     AND a.fuente_ref=e->>'fuente_ref' AND a.fuente_sha256=e->>'fuente_sha256' AND a.unidad_solicitud_sha256 IS NULL)
   THEN RAISE EXCEPTION 'AD176 prueba: vínculo de fuente/recibo/plan divergente'; END IF;
  ELSE
   SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e);
   SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e);
   IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref
     AND a.unidad_solicitud_sha256=e->>'solicitud_sha256' AND a.unidad_plan_ref IS NULL AND a.unidad_preimagen_sha256 IS NULL
     AND a.unidad_configuracion_sha256 IS NULL AND a.unidad_alcance IS NULL AND a.unidad_recibo_ref IS NULL AND a.unidad_recibo_sha256 IS NULL
     AND a.fuente_ref IS NULL AND a.fuente_sha256 IS NULL AND a.plan_sha256 IS NULL AND a.aprobacion_ref IS NULL)
   THEN RAISE EXCEPTION 'AD176 prueba: intento mezcló confirmación ficticia'; END IF;
  END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND attname='version_consumo' AND NOT attisdropped) THEN
   BEGIN
    EXECUTE 'INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 SELECT (pg_catalog.jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)).*'
    USING (SELECT pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('version_consumo',3,'auditoria_ref','aud_v3_u_000000000000000000000000000a0176',
     'evento_ref','evento_000000000000000000000000000a0176','secuencia',v_secuencia+v_total+2) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref);
    RAISE EXCEPTION 'AD176 prueba: familia técnica aceptó version_consumo nominal';
   EXCEPTION WHEN check_violation THEN NULL; END;
  END IF;
  v_total:=v_total+1;
  IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y) OR x.secuencia<>v_secuencia+v_total
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref
   AND a.operador_login=session_user::name AND a.modulo_id='personal' AND a.resultado=e->>'resultado' AND a.motivo_ref=e->>'motivo_ref'
   AND a.proceso='postgresql' AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL
   AND a.fuentes_plan_ref IS NULL AND a.fuentes_preimagen_sha256 IS NULL AND a.fuentes_configuracion_sha256 IS NULL
   AND a.fuentes_alcance IS NULL AND a.fuentes_solicitud_sha256 IS NULL)
  THEN RAISE EXCEPTION 'AD176 prueba: recibo/replay/familia divergente'; END IF;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(v_confirmado||pg_catalog.jsonb_build_object('recibo_sha256',pg_catalog.repeat('9',64)));
  RAISE EXCEPTION 'AD176 prueba: aceptó confirmado cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||pg_catalog.jsonb_build_object('solicitud_sha256',pg_catalog.repeat('9',64)));
  RAISE EXCEPTION 'AD176 prueba: aceptó intento cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||pg_catalog.jsonb_build_object('evento_ref',v_confirmado->>'evento_ref'));
  RAISE EXCEPTION 'AD176 prueba: reutilizó evento entre familias';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 FOR k IN SELECT value FROM pg_catalog.jsonb_array_elements_text('["actor_ref","perfil_activo_ref","fuente_ref","recibo_ref","aprobacion_ref"]'::jsonb) LOOP
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||pg_catalog.jsonb_build_object(k,'inventado'));
   RAISE EXCEPTION 'AD176 prueba: aceptó campo cruzado';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||'{"operador_login":"otro_operador"}'::jsonb);
  RAISE EXCEPTION 'AD176 prueba: aceptó LOGIN ajeno';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(v_confirmado||'{"accion":"provisionar_fuentes_iniciales_admin_v1"}'::jsonb);
  RAISE EXCEPTION 'AD176 prueba: atribuyó acción de fuentes CA/IS';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(v_confirmado||'{"recibo_ref":null}'::jsonb);
  RAISE EXCEPTION 'AD176 prueba: admitió recibo nulo';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||'{"proceso":"cli"}'::jsonb);
  RAISE EXCEPTION 'AD176 prueba: atribuyó observador falso';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(e||'{"resultado":"permitido"}'::jsonb);
  RAISE EXCEPTION 'AD176 prueba: resultado y motivo cruzados';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET unidad_solicitud_sha256=pg_catalog.repeat('9',64) WHERE auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD176 prueba: permitió mutar auditoría';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF v_total<>6 OR (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>v_secuencia+v_total
 OR EXISTS(SELECT 1 FROM pg_temp.ad176_preimagen p LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref)
   WHERE pg_catalog.to_jsonb(a) IS DISTINCT FROM p.fila)
 THEN RAISE EXCEPTION 'AD176 prueba: cadena o historia divergente tras rechazos'; END IF;
END $prueba$;
ROLLBACK;
