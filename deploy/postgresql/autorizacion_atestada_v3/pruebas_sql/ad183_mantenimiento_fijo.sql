\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';SET LOCAL statement_timeout='20s';
CREATE TEMP TABLE ad183_pre ON COMMIT DROP AS SELECT auditoria_ref,to_jsonb(a) fila FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a;
GRANT SELECT ON pg_temp.ad183_pre TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $t$
DECLARE e jsonb;v jsonb;m bytea;h text;k text;orden text[];dominio text;x record;y record;inicio numeric;total int:=0;f oid;confirmado jsonb;
BEGIN
 FOREACH k IN ARRAY ARRAY['registrar_mantenimiento_perfil_fijo_admin_v1','registrar_intento_mantenimiento_perfil_fijo_admin_v1'] LOOP
 f:=to_regprocedure('vec_autorizacion_atestada_v3.'||k||'(jsonb)');
 IF f IS NULL OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE') OR has_function_privilege('vec_personal_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
 AND(a.grantee NOT IN('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole) OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'AD183 prueba: ACL ampliada';END IF;END LOOP;
 SELECT secuencia INTO STRICT inicio FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id;
 FOR v IN SELECT value FROM jsonb_array_elements($v${
  "vectores": [
    {
      "evento": {
        "tipo_registro": "mantenimiento_perfil_fijo_admin",
        "evento_ref": "evento_0000000000000000000000000000477d",
        "operador_login": "operador_ámbito_sintético",
        "plan_sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "preimagen_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "catalogo_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "rol_origen_ref": "rol_fijo:aplicacion:v4",
        "rol_origen_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "rol_destino_ref": "rol_fijo:aplicacion:v5",
        "rol_destino_sha256": "5555555555555555555555555555555555555555555555555555555555555555",
        "asignacion_1_origen_ref": "asignacion:opaca:primera:v1",
        "asignacion_1_origen_sha256": "6666666666666666666666666666666666666666666666666666666666666666",
        "asignacion_1_destino_ref": "asignacion:opaca:primera:v2",
        "asignacion_1_destino_sha256": "7777777777777777777777777777777777777777777777777777777777777777",
        "asignacion_2_origen_ref": "asignacion:opaca:segunda:v1",
        "asignacion_2_origen_sha256": "8888888888888888888888888888888888888888888888888888888888888888",
        "asignacion_2_destino_ref": "asignacion:opaca:segunda:v2",
        "asignacion_2_destino_sha256": "9999999999999999999999999999999999999999999999999999999999999999",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "mantenimiento_perfil_fijo_admin",
        "correlacion_ref": "correlacion_0000000000000000000000000000477d"
      },
      "secuencia": 1,
      "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "auditoria_ref": "aud_v3_mf_0000000000000000000000000000477d",
      "registrada_en": "2026-10-04T02:05:00.123456Z",
      "evento_material_sha256": "965a3fe7ca9026d9b56b929c499d002c3e6d1bc584e9f94bee58f33aeb4c6193",
      "huella_sha256": "0f77d05f4f2973bfbb5486903c8a24ad431cdf7d569b5de7b6d76d549c4272f3"
    },
    {
      "evento": {
        "tipo_registro": "intento_mantenimiento_perfil_fijo_admin",
        "evento_ref": "evento_0000000000000000000000000000477e",
        "operador_login": "operador_mantenimiento_sintetico",
        "solicitud_sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "accion": "mantener_version_perfil_fijo_admin_v1",
        "recurso_ref": "solicitud_mantenimiento:00000000000000000000000000000002",
        "resultado": "permitido",
        "motivo_ref": "mantenimiento_registrado",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "mantenimiento_perfil_fijo_admin",
        "correlacion_ref": "correlacion_0000000000000000000000000000477e"
      },
      "secuencia": 2,
      "anterior_sha256": "0f77d05f4f2973bfbb5486903c8a24ad431cdf7d569b5de7b6d76d549c4272f3",
      "auditoria_ref": "aud_v3_mfi_0000000000000000000000000000477e",
      "registrada_en": "2026-10-04T02:05:00.123456Z",
      "evento_material_sha256": "2ed67e8a42a26504b55496dbd1c52e1546938ccf18cd9a712632af718b42ae8e",
      "huella_sha256": "a3ae5aeab7a9fe4c854febc9a48487e3f13dba8a48fcde7f48ad960cbb75f50b"
    },
    {
      "evento": {
        "tipo_registro": "intento_mantenimiento_perfil_fijo_admin",
        "evento_ref": "evento_0000000000000000000000000000477f",
        "operador_login": "operador_mantenimiento_sintetico",
        "solicitud_sha256": "3333333333333333333333333333333333333333333333333333333333333333",
        "accion": "mantener_version_perfil_fijo_admin_v1",
        "recurso_ref": "solicitud_mantenimiento:00000000000000000000000000000003",
        "resultado": "permitido",
        "motivo_ref": "mantenimiento_replay",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "mantenimiento_perfil_fijo_admin",
        "correlacion_ref": "correlacion_0000000000000000000000000000477f"
      },
      "secuencia": 3,
      "anterior_sha256": "a3ae5aeab7a9fe4c854febc9a48487e3f13dba8a48fcde7f48ad960cbb75f50b",
      "auditoria_ref": "aud_v3_mfi_0000000000000000000000000000477f",
      "registrada_en": "2026-10-04T02:05:00.123456Z",
      "evento_material_sha256": "5e1efbabd30b0713ebdd632f13caaf2ac77f77e45e0854fcb08d6f1c79c21e20",
      "huella_sha256": "260f5cb03ea64e02d9edd2bd792d2500691409ce8f0177c9a71bd6d558183bc4"
    },
    {
      "evento": {
        "tipo_registro": "intento_mantenimiento_perfil_fijo_admin",
        "evento_ref": "evento_00000000000000000000000000004780",
        "operador_login": "operador_mantenimiento_sintetico",
        "solicitud_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
        "accion": "mantener_version_perfil_fijo_admin_v1",
        "recurso_ref": "solicitud_mantenimiento:00000000000000000000000000000004",
        "resultado": "denegado",
        "motivo_ref": "mantenimiento_denegado",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "mantenimiento_perfil_fijo_admin",
        "correlacion_ref": "correlacion_00000000000000000000000000004780"
      },
      "secuencia": 4,
      "anterior_sha256": "260f5cb03ea64e02d9edd2bd792d2500691409ce8f0177c9a71bd6d558183bc4",
      "auditoria_ref": "aud_v3_mfi_00000000000000000000000000004780",
      "registrada_en": "2026-10-04T02:05:00.123456Z",
      "evento_material_sha256": "ac7541aa0bf7cd8254cfa654654a56a0fd7aa376ca10700566c360bdb62300d1",
      "huella_sha256": "45651b0b5a3e712e6b168bb30d14f0d59aae0c83cb399724632d99c030a66aab"
    },
    {
      "evento": {
        "tipo_registro": "intento_mantenimiento_perfil_fijo_admin",
        "evento_ref": "evento_00000000000000000000000000004781",
        "operador_login": "operador_mantenimiento_sintetico",
        "solicitud_sha256": "5555555555555555555555555555555555555555555555555555555555555555",
        "accion": "mantener_version_perfil_fijo_admin_v1",
        "recurso_ref": "solicitud_mantenimiento:00000000000000000000000000000005",
        "resultado": "error",
        "motivo_ref": "mantenimiento_error",
        "proceso": "postgresql",
        "canal": "operacion_tecnica_privada",
        "finalidad_ref": "mantenimiento_perfil_fijo_admin",
        "correlacion_ref": "correlacion_00000000000000000000000000004781"
      },
      "secuencia": 5,
      "anterior_sha256": "45651b0b5a3e712e6b168bb30d14f0d59aae0c83cb399724632d99c030a66aab",
      "auditoria_ref": "aud_v3_mfi_00000000000000000000000000004781",
      "registrada_en": "2026-10-04T02:05:00.123456Z",
      "evento_material_sha256": "50c1e2583d1bf39484a6a2d884e269aa7caa8e575bcd54323f2940e09e5fa4f4",
      "huella_sha256": "49977a8729280d37c796c77ea622b2eb62e315c18088e13980df860a6b4a1fce"
    }
  ]
}$v$::jsonb->'vectores') LOOP
 e:=v->'evento';
 IF e->>'tipo_registro'='mantenimiento_perfil_fijo_admin' THEN
 orden:=ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','preimagen_sha256','catalogo_sha256','rol_origen_ref','rol_origen_sha256','rol_destino_ref','rol_destino_sha256','asignacion_1_origen_ref','asignacion_1_origen_sha256','asignacion_1_destino_ref','asignacion_1_destino_sha256','asignacion_2_origen_ref','asignacion_2_origen_sha256','asignacion_2_destino_ref','asignacion_2_destino_sha256','proceso','canal','finalidad_ref','correlacion_ref'];dominio:='mantenimiento-perfil-fijo-admin.v1';
 ELSE orden:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256','accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];dominio:='intento-mantenimiento-perfil-fijo-admin.v1';END IF;
 m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.'||dominio);FOREACH k IN ARRAY orden LOOP m:=m||vec_autorizacion_atestada_v3.encuadrar_mac(e->>k);END LOOP;
 h:=encode(sha256(m),'hex');IF h IS DISTINCT FROM v->>'evento_material_sha256' THEN RAISE EXCEPTION 'AD183 prueba: vector material divergente';END IF;
 m:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.'||dominio)||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'secuencia')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'anterior_sha256')||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'auditoria_ref')||vec_autorizacion_atestada_v3.encuadrar_mac(h)||vec_autorizacion_atestada_v3.encuadrar_mac(v->>'registrada_en');
 IF encode(sha256(m),'hex') IS DISTINCT FROM v->>'huella_sha256' THEN RAISE EXCEPTION 'AD183 prueba: vector eslabón divergente';END IF;
 e:=e||jsonb_build_object('operador_login',session_user::text);
 IF e->>'tipo_registro'='mantenimiento_perfil_fijo_admin' THEN confirmado:=e;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(e);SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(e);
 ELSE SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(e);SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(e);END IF;
 total:=total+1;IF to_jsonb(x) IS DISTINCT FROM to_jsonb(y) OR x.secuencia<>inicio+total THEN RAISE EXCEPTION 'AD183 prueba: replay/recibo divergente';END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.auditoria_ref=x.auditoria_ref AND a.operador_login=session_user::name
 AND a.actor_ref IS NULL AND a.perfil_activo_ref IS NULL AND a.decision_ref IS NULL AND a.fuente_ref IS NULL AND a.aprobacion_ref IS NULL
 AND a.accion='mantener_version_perfil_fijo_admin_v1' AND a.finalidad_ref='mantenimiento_perfil_fijo_admin') THEN RAISE EXCEPTION 'AD183 prueba: falsa identidad o acción';END IF;
 END LOOP;
 BEGIN PERFORM * FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(confirmado||jsonb_build_object('rol_destino_sha256',repeat('a',64)));RAISE EXCEPTION 'AD183 prueba: cambio de rol aceptado';EXCEPTION WHEN unique_violation THEN NULL;END;
 BEGIN PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(e||'{"perfil_activo_ref":"inventado"}'::jsonb);RAISE EXCEPTION 'AD183 prueba: perfil ficticio aceptado';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM * FROM vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(confirmado||'{"catalogo_sha256":null}'::jsonb);RAISE EXCEPTION 'AD183 prueba: catálogo nulo';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(e||'{"operador_login":"otro"}'::jsonb);RAISE EXCEPTION 'AD183 prueba: LOGIN ficticio';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN UPDATE vec_autorizacion_atestada_v3.auditoria_consumo_v3 SET motivo_ref='otro' WHERE auditoria_ref=x.auditoria_ref;RAISE EXCEPTION 'AD183 prueba: auditoría mutable';EXCEPTION WHEN SQLSTATE '55000' THEN NULL;END;
 IF total<>5 OR(SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)<>inicio+total OR EXISTS(SELECT 1 FROM pg_temp.ad183_pre p LEFT JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref) WHERE to_jsonb(a) IS DISTINCT FROM p.fila) THEN RAISE EXCEPTION 'AD183 prueba: historia alterada';END IF;
END $t$;
ROLLBACK;
