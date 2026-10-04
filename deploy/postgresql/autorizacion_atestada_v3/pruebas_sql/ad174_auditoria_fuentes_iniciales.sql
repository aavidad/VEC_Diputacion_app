\set ON_ERROR_STOP on
-- Solo clon desechable con AD174; comprueba resultado y conserva ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';
CREATE TEMP TABLE ad174_preimagen ON COMMIT DROP AS
 SELECT auditoria_ref,huella_sha256,anterior_sha256,registrada_en
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
GRANT SELECT ON TABLE pg_temp.ad174_preimagen TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE e jsonb;x record;y record;v_secuencia numeric;f oid;v jsonb;m bytea;h text;k text;
 orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_ref','plan_sha256',
 'preimagen_sha256','configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref',
 'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];
BEGIN
 f:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)');
 IF f IS NULL OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
  LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,
   'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD174 prueba: ACL privada divergente'; END IF;
 FOR v IN SELECT value FROM pg_catalog.jsonb_array_elements($vectores${
  "vectores": [
    {
      "evento": {
        "tipo_registro": "provision_fuentes_iniciales_admin",
        "evento_ref": "evento_00000000000000000000000000000001",
        "operador_login": "operador_fuentes_sintetico",
        "plan_ref": "plan_fuentes:ad174:1",
        "plan_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "preimagen_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "configuracion_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "aprobacion_ref": "aprobacion:ad174:1",
        "alcance_fuente": "sintetico_declarado",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "plan_fuentes:ad174:1",
        "resultado": "permitido",
        "motivo_ref": "admin.fuentes.permitido",
        "proceso": "fuentes_sinteticas",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_00000000000000000000000000000001",
        "fuente_ref": "recibos_fuentes:ad174:1",
        "fuente_sha256": "4444444444444444444444444444444444444444444444444444444444444444"
      },
      "secuencia": 1,
      "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "auditoria_ref": "aud_v3_f_00000000000000000000000000000001",
      "registrada_en": "2026-10-03T21:00:00.123456Z",
      "evento_material_sha256": "534bb859b593e9311143555caf163ba942bf8689889cef7d1ee81a51cefd0ef5",
      "huella_sha256": "a6ebcbda7015497b9f1ae385d1db06df31867eed239391fd32433e9f90efd9fb"
    },
    {
      "evento": {
        "tipo_registro": "provision_fuentes_iniciales_admin",
        "evento_ref": "evento_00000000000000000000000000000002",
        "operador_login": "operador_ámbito_sintético",
        "plan_ref": "plan_fuentes:ad174:2",
        "plan_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "preimagen_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "configuracion_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "aprobacion_ref": "aprobacion:ad174:2",
        "alcance_fuente": "sintetico_declarado",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "plan_fuentes:ad174:2",
        "resultado": "permitido",
        "motivo_ref": "admin.fuentes.permitido",
        "proceso": "fuentes_sinteticas",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_00000000000000000000000000000002",
        "fuente_ref": "recibos_fuentes:ad174:2",
        "fuente_sha256": "4444444444444444444444444444444444444444444444444444444444444444"
      },
      "secuencia": 2,
      "anterior_sha256": "a6ebcbda7015497b9f1ae385d1db06df31867eed239391fd32433e9f90efd9fb",
      "auditoria_ref": "aud_v3_f_00000000000000000000000000000002",
      "registrada_en": "2026-10-03T21:00:00.123456Z",
      "evento_material_sha256": "231c70e826293a7e53cad01a2090b17ae4a12009d25784744a99d6ab77568873",
      "huella_sha256": "3beb0ffa5838844482d92c6d356bd354af60c7d7b6001c438418fcf809e73981"
    }
  ]
}$vectores$::jsonb->'vectores') LOOP
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.fuentes-iniciales.v1');
  FOREACH k IN ARRAY orden LOOP m:=m||vec_autorizacion_atestada_v3.encuadrar_mac(v->'evento'->>k); END LOOP;
  h:=pg_catalog.encode(pg_catalog.sha256(m),'hex');
  IF h IS DISTINCT FROM v->>'evento_material_sha256' THEN RAISE EXCEPTION 'AD174 prueba: vector material divergente'; END IF;
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.fuentes-iniciales.v1')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'secuencia')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'anterior_sha256')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'auditoria_ref')||vec_autorizacion_atestada_v3.encuadrar_mac(h)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'registrada_en');
  IF pg_catalog.encode(pg_catalog.sha256(m),'hex') IS DISTINCT FROM v->>'huella_sha256'
  THEN RAISE EXCEPTION 'AD174 prueba: vector eslabón divergente'; END IF;
  e:=v->'evento';
 END LOOP;
 e:=e||pg_catalog.jsonb_build_object('operador_login',session_user::text,
   'evento_ref','evento_00000000000000000000000000000174');
 SELECT secuencia INTO STRICT v_secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e);
 SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e);
 IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y) OR x.secuencia<>v_secuencia+1
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
  WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='provision_fuentes_iniciales_admin'
   AND a.operador_login=session_user::name AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL
   AND a.decision_ref IS NULL AND a.registro_contexto_ref IS NULL AND a.sesion_ref IS NULL
   AND a.fuentes_plan_ref=e->>'plan_ref' AND a.fuentes_alcance='sintetico_declarado')
 THEN RAISE EXCEPTION 'AD174 prueba: recibo/replay/familia divergente'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e||'{"plan_ref":"plan:distinto"}'::jsonb);
  RAISE EXCEPTION 'AD174 prueba: aceptó material cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 FOR k IN SELECT value FROM pg_catalog.jsonb_array_elements_text('["actor_ref","perfil_activo_ref","decision_ref","sesion_ref"]'::jsonb) LOOP
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e||pg_catalog.jsonb_build_object(k,'inventado'));
   RAISE EXCEPTION 'AD174 prueba: aceptó campo cruzado';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e-'fuente_ref');
  RAISE EXCEPTION 'AD174 prueba: aceptó fuente ausente';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e||'{"operador_login":"otro_operador"}'::jsonb);
  RAISE EXCEPTION 'AD174 prueba: aceptó LOGIN ajeno';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e||'{"alcance_fuente":"institucional"}'::jsonb);
  RAISE EXCEPTION 'AD174 prueba: aceptó fuente institucional';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(e||'{"resultado":"denegado"}'::jsonb);
  RAISE EXCEPTION 'AD174 prueba: inventó evento denegado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND attname='version_consumo' AND NOT attisdropped) THEN
  BEGIN
   EXECUTE 'INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 SELECT (pg_catalog.jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)).*'
   USING (SELECT pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('version_consumo',3,'auditoria_ref','aud_v3_f_000000000000000000000000000a0174',
    'evento_ref','evento_000000000000000000000000000a0174','secuencia',v_secuencia+2) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref);
   RAISE EXCEPTION 'AD174 prueba: fuente técnica aceptó version_consumo nominal';
  EXCEPTION WHEN check_violation THEN NULL; END;
 END IF;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET fuentes_plan_ref='plan:distinto' WHERE auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD174 prueba: permitió mutar auditoría';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>v_secuencia+1
 OR EXISTS(SELECT 1 FROM pg_temp.ad174_preimagen p
   LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref)
   WHERE (a.huella_sha256,a.anterior_sha256,a.registrada_en) IS DISTINCT FROM (p.huella_sha256,p.anterior_sha256,p.registrada_en))
 THEN RAISE EXCEPTION 'AD174 prueba: cadena o historia divergente tras rechazos'; END IF;
END $prueba$;
ROLLBACK;
