\set ON_ERROR_STOP on
-- Clon desechable AD174 nueva candidata. Sin DOWN, todo hace ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';
CREATE TEMP TABLE ad174_intento_preimagen ON COMMIT DROP AS
 SELECT auditoria_ref,huella_sha256,anterior_sha256,registrada_en
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
GRANT SELECT ON TABLE pg_temp.ad174_intento_preimagen TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $prueba$
DECLARE e jsonb;x record;y record;v_secuencia numeric;v_total integer:=0;f oid;v jsonb;m bytea;h text;k text;
 orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256','accion',
  'recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
BEGIN
 f:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb)');
 IF f IS NULL OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
  LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,
   'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_identidad_sesiones_v1_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD174 intento prueba: ACL privada divergente'; END IF;
 SELECT secuencia INTO STRICT v_secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 FOR v IN SELECT value FROM pg_catalog.jsonb_array_elements($vectores${
  "vectores": [
    {
      "evento": {
        "tipo_registro": "intento_fuentes_iniciales_admin",
        "evento_ref": "evento_000000000000000000000000000043f9",
        "operador_login": "operador_fuentes_sintetico",
        "solicitud_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "solicitud_fuentes:00000000000000000000000000000001",
        "resultado": "permitido",
        "motivo_ref": "fuentes_registradas",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000043f9"
      },
      "secuencia": 1,
      "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "auditoria_ref": "aud_v3_fi_000000000000000000000000000043f9",
      "registrada_en": "2026-10-03T21:00:01.123456Z",
      "evento_material_sha256": "ea99a77e81e7cd2bfb45763a765c35634bfaa908eae5432ada1bf0da99d34710",
      "huella_sha256": "7ed646d78f0273c1748395914f05f50fff85cb5b7e04dc7819dfa01db89d439a"
    },
    {
      "evento": {
        "tipo_registro": "intento_fuentes_iniciales_admin",
        "evento_ref": "evento_000000000000000000000000000043fa",
        "operador_login": "operador_ámbito_sintético",
        "solicitud_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "solicitud_fuentes:00000000000000000000000000000002",
        "resultado": "permitido",
        "motivo_ref": "fuentes_replay",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000043fa"
      },
      "secuencia": 2,
      "anterior_sha256": "7ed646d78f0273c1748395914f05f50fff85cb5b7e04dc7819dfa01db89d439a",
      "auditoria_ref": "aud_v3_fi_000000000000000000000000000043fa",
      "registrada_en": "2026-10-03T21:00:01.123456Z",
      "evento_material_sha256": "e0d478de107b10ab6195cef641b19bc34e9fe1584cb2a8f55248aa73d1dbcb13",
      "huella_sha256": "ab476c1cd960d1648a6b2a4e7abcb80755d2a9e7a5246e8d34612133a281fa74"
    },
    {
      "evento": {
        "tipo_registro": "intento_fuentes_iniciales_admin",
        "evento_ref": "evento_000000000000000000000000000043fb",
        "operador_login": "operador_fuentes_sintetico",
        "solicitud_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "solicitud_fuentes:00000000000000000000000000000003",
        "resultado": "denegado",
        "motivo_ref": "fuentes_denegadas",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000043fb"
      },
      "secuencia": 3,
      "anterior_sha256": "ab476c1cd960d1648a6b2a4e7abcb80755d2a9e7a5246e8d34612133a281fa74",
      "auditoria_ref": "aud_v3_fi_000000000000000000000000000043fb",
      "registrada_en": "2026-10-03T21:00:01.123456Z",
      "evento_material_sha256": "4901344da7871d63d1e41964bc499da49666b92986bc300ca04cf898a304f89f",
      "huella_sha256": "d25c4e435803db09a84034800a9c36ec481f12e2757a547bdffd3bdaaee3c321"
    },
    {
      "evento": {
        "tipo_registro": "intento_fuentes_iniciales_admin",
        "evento_ref": "evento_000000000000000000000000000043fc",
        "operador_login": "operador_fuentes_sintetico",
        "solicitud_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "accion": "provisionar_fuentes_iniciales_admin_v1",
        "recurso_ref": "solicitud_fuentes:00000000000000000000000000000004",
        "resultado": "error",
        "motivo_ref": "fuentes_error",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "provision_fuentes_iniciales_admin",
        "correlacion_ref": "correlacion_000000000000000000000000000043fc"
      },
      "secuencia": 4,
      "anterior_sha256": "d25c4e435803db09a84034800a9c36ec481f12e2757a547bdffd3bdaaee3c321",
      "auditoria_ref": "aud_v3_fi_000000000000000000000000000043fc",
      "registrada_en": "2026-10-03T21:00:01.123456Z",
      "evento_material_sha256": "fc998352e04aea5a89a8551e930844530ea0e0cd4f85ed075d30cb8d47b1c400",
      "huella_sha256": "43e70ab30c758b72481812d3792a9b0ff92e07a45c676bbeac47b3ae35fb56d9"
    }
  ]
}$vectores$::jsonb->'vectores') LOOP
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-fuentes-iniciales.v1');
  FOREACH k IN ARRAY orden LOOP m:=m||vec_autorizacion_atestada_v3.encuadrar_mac(v->'evento'->>k); END LOOP;
  h:=pg_catalog.encode(pg_catalog.sha256(m),'hex');
  IF h IS DISTINCT FROM v->>'evento_material_sha256' THEN RAISE EXCEPTION 'AD174 intento prueba: vector material divergente'; END IF;
  m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-fuentes-iniciales.v1')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'secuencia')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'anterior_sha256')||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'auditoria_ref')||vec_autorizacion_atestada_v3.encuadrar_mac(h)||
   vec_autorizacion_atestada_v3.encuadrar_mac(v->>'registrada_en');
  IF pg_catalog.encode(pg_catalog.sha256(m),'hex') IS DISTINCT FROM v->>'huella_sha256'
  THEN RAISE EXCEPTION 'AD174 intento prueba: vector eslabón divergente'; END IF;
  e:=(v->'evento')||pg_catalog.jsonb_build_object('operador_login',session_user::text);
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e);
  SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e);
  v_total:=v_total+1;
  IF pg_catalog.to_jsonb(x) IS DISTINCT FROM pg_catalog.to_jsonb(y) OR x.secuencia<>v_secuencia+v_total
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
   WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='intento_fuentes_iniciales_admin'
    AND a.operador_login=session_user::name AND a.fuentes_solicitud_sha256=e->>'solicitud_sha256'
    AND a.resultado=e->>'resultado' AND a.motivo_ref=e->>'motivo_ref' AND a.proceso='postgresql'
    AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL
    AND a.fuente_ref IS NULL AND a.fuente_sha256 IS NULL AND a.plan_sha256 IS NULL AND a.aprobacion_ref IS NULL
    AND a.fuentes_plan_ref IS NULL AND a.fuentes_preimagen_sha256 IS NULL AND a.fuentes_configuracion_sha256 IS NULL)
  THEN RAISE EXCEPTION 'AD174 intento prueba: recibo/replay/familia divergente'; END IF;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||pg_catalog.jsonb_build_object('solicitud_sha256',pg_catalog.repeat('9',64)));
  RAISE EXCEPTION 'AD174 intento prueba: aceptó material cambiado';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 FOR k IN SELECT value FROM pg_catalog.jsonb_array_elements_text('["actor_ref","perfil_activo_ref","fuente_ref","preimagen_sha256","aprobacion_ref"]'::jsonb) LOOP
  BEGIN
   PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||pg_catalog.jsonb_build_object(k,'inventado'));
   RAISE EXCEPTION 'AD174 intento prueba: aceptó campo cruzado';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||'{"solicitud_sha256":null}'::jsonb);
  RAISE EXCEPTION 'AD174 intento prueba: aceptó solicitud nula';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||'{"operador_login":"otro_operador"}'::jsonb);
  RAISE EXCEPTION 'AD174 intento prueba: aceptó LOGIN ajeno';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||'{"resultado":"permitido"}'::jsonb);
  RAISE EXCEPTION 'AD174 intento prueba: aceptó resultado y motivo cruzados';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||'{"motivo_ref":"libre"}'::jsonb);
  RAISE EXCEPTION 'AD174 intento prueba: aceptó motivo libre';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(e||'{"proceso":"cli"}'::jsonb);
  RAISE EXCEPTION 'AD174 intento prueba: atribuyó observador no acreditado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND attname='version_consumo' AND NOT attisdropped) THEN
  BEGIN
   EXECUTE 'INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 SELECT (pg_catalog.jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)).*'
   USING (SELECT pg_catalog.to_jsonb(a)||pg_catalog.jsonb_build_object('version_consumo',3,'auditoria_ref','aud_v3_fi_000000000000000000000000000b0174',
    'evento_ref','evento_000000000000000000000000000b0174','secuencia',v_secuencia+v_total+1) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref);
   RAISE EXCEPTION 'AD174 prueba: intento técnico aceptó version_consumo nominal';
  EXCEPTION WHEN check_violation THEN NULL; END;
 END IF;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET fuentes_solicitud_sha256=pg_catalog.repeat('9',64) WHERE auditoria_ref=x.auditoria_ref;
  RAISE EXCEPTION 'AD174 intento prueba: permitió mutar auditoría';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 IF v_total<>4 OR (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>v_secuencia+v_total
 OR EXISTS(SELECT 1 FROM pg_temp.ad174_intento_preimagen p
   LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref)
   WHERE (a.huella_sha256,a.anterior_sha256,a.registrada_en) IS DISTINCT FROM (p.huella_sha256,p.anterior_sha256,p.registrada_en))
 THEN RAISE EXCEPTION 'AD174 intento prueba: cadena o historia divergente tras rechazos'; END IF;
END $prueba$;
ROLLBACK;
