\set ON_ERROR_STOP on
-- Foco de codec AD3-35: sólo referencias/sobres sintéticos, sin autoridad V3.
-- No prueba ACL, firmas ni persistencia; requiere fuente instalada en clúster aislado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
DO $foco$
DECLARE
    auditoria bytea:=convert_to($json${"id":"","seq":0,"actor_id":"hmac-sha256:prueba:1111111111111111111111111111111111111111111111111111111111111111","actor_profile":"perfil_prueba_contacto","actor_roles":["rol_prueba_contacto_v1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"gestion_contacto_propio","action":"vec.contacto_usuario.alta","module_id":"vec.module.usuarios","subject_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","object_version":1,"result":"accepted","correlation_ref":"corr_prueba_contacto","occurred_at":"2026-09-12T18:00:00Z","signature":""}$json$,'UTF8');
    negocio bytea:=convert_to($json${"Esquema":"vec.contacto_usuario.registro-cuerpo.v1","SujetoRef":"per_aaaaaaaaaaaaaaaaaaaaaa","Audiencia":"vec.contacto_usuario.registro.v1","FinalidadRef":"gestion_contacto_propio","VersionEsperada":0,"VersionNueva":1,"Sobre":{"Version":1,"ClaveRef":"clave_prueba_contacto_v1","Nonce":"AQEBAQEBAQEBAQEB","Cifrado":"AQEBAQEBAQEBAQEBAQEBAQ=="},"Auditoria":{"id":"","seq":0,"actor_id":"hmac-sha256:prueba:1111111111111111111111111111111111111111111111111111111111111111","actor_profile":"perfil_prueba_contacto","actor_roles":["rol_prueba_contacto_v1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"gestion_contacto_propio","action":"vec.contacto_usuario.alta","module_id":"vec.module.usuarios","subject_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","object_version":1,"result":"accepted","correlation_ref":"corr_prueba_contacto","occurred_at":"2026-09-12T18:00:00Z","signature":""}}$json$,'UTF8');
    recurso bytea:=convert_to($json${"ambitos":{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa"},"atributos":{"contacto_finalidad_ref":"gestion_contacto_propio","contacto_sujeto_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","contacto_version":"1","contacto_version_esperada":"0","material_sha256":"8e752041bd3df0e8e2efb02ceef713c949dacb18ae8a87ba9dea14b2366f64b5"}}$json$,'UTF8');
    decision bytea:=convert_to($json${"decision_ref":"decision_prueba_contacto","concedida":true,"codigo":"concedida","accion":"vec.contacto_usuario.alta","modulo_id":"vec.module.usuarios","tipo_recurso":"contacto_usuario","recurso_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","finalidad":"gestion_contacto_propio","contexto_recurso_huella_sha256":"0c48ed1efe7ed9aff0572d97f04a0abf5c5de7e6a6322537fccbcaed2ba629c4","campos_permitidos":[],"obligaciones":[],"perfil_activo_ref":"perfil_prueba_contacto","version_rol_ref":"rol_prueba_contacto_v1","correlacion_ref":"corr_prueba_contacto","vinculo_autenticacion_actor":{"metodo_observado":"certificado","garantia_observada":"alto"},"principal_id":"principal_prueba_contacto"}$json$,'UTF8');
    contexto bytea:=convert_to($json${"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","principal_ref":"principal_prueba_contacto","perfil_activo_ref":"perfil_prueba_contacto","metodo":"certificado","garantia":"alto"}$json$,'UTF8');
    casos bytea[][]; v bytea[]; resultado jsonb; fallo boolean; n integer;
BEGIN
    resultado:=vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.alta',negocio,recurso,auditoria,decision,contexto);
    IF resultado IS DISTINCT FROM convert_from(negocio,'UTF8')::jsonb THEN
        RAISE EXCEPTION 'foco35: positivo de codec divergente';
    END IF;
    -- Todos los NULL deniegan: no se confía en la lógica ternaria de OR.
    FOR n IN 1..5 LOOP
        v:=ARRAY[negocio,recurso,auditoria,decision,contexto]; v[n]:=NULL; fallo:=false;
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1('vec.contacto_usuario.alta',v[1],v[2],v[3],v[4],v[5]);
        EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
        END;
        IF NOT fallo THEN RAISE EXCEPTION 'foco35: NULL aceptado'; END IF;
    END LOOP;
    casos:=ARRAY[
        ARRAY[negocio,recurso,convert_to(replace(convert_from(auditoria,'UTF8'),'hmac-sha256:prueba:','hmac-sha256:ajena:'),'UTF8'),decision,contexto],
        ARRAY[negocio,convert_to(replace(convert_from(recurso,'UTF8'),'"contacto_version":"1"','"contacto_version":"2"'),'UTF8'),auditoria,decision,contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"obligaciones":[]','"obligaciones":["firma"]'),'UTF8'),contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"campos_permitidos":[]','"campos_permitidos":["correo"]'),'UTF8'),contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"concedida":true','"concedida":null'),'UTF8'),contexto],
        ARRAY[negocio,recurso,auditoria,decision,convert_to(replace(convert_from(contexto,'UTF8'),'per_aaaaaaaaaaaaaaaaaaaaaa','per_bbbbbbbbbbbbbbbbbbbbbb'),'UTF8')],
        ARRAY[convert_to(replace(convert_from(negocio,'UTF8'),'AQEBAQEBAQEBAQEB','AgEBAQEBAQEBAQEB'),'UTF8'),recurso,auditoria,decision,contexto],
        ARRAY[negocio,recurso,convert_to(replace(convert_from(auditoria,'UTF8'),'"signature":""','"signature":"","authorization_ref":"anticipada"'),'UTF8'),decision,contexto],
        ARRAY[negocio,convert_to(replace(convert_from(recurso,'UTF8'),'"material_sha256":','"material_sha256":null,"material_sha256":'),'UTF8'),auditoria,decision,contexto]
    ];
    FOREACH v SLICE 1 IN ARRAY casos LOOP
        fallo:=false;
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1('vec.contacto_usuario.alta',v[1],v[2],v[3],v[4],v[5]);
        EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
        END;
        IF NOT fallo THEN RAISE EXCEPTION 'foco35: material divergente aceptado'; END IF;
    END LOOP;
    -- Lectura con auditoría preparada comprometida antes del consumo.
    auditoria:=convert_to($json${"id":"","seq":0,"actor_id":"hmac-sha256:prueba:1111111111111111111111111111111111111111111111111111111111111111","actor_profile":"perfil_prueba_contacto","actor_roles":["rol_prueba_contacto_v1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"envio_llamamiento","action":"vec.contacto_usuario.consultar","module_id":"vec.module.usuarios","subject_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","object_version":1,"result":"accepted","correlation_ref":"corr_prueba_contacto","occurred_at":"2026-09-12T18:00:00Z","signature":""}$json$,'UTF8');
    negocio:=convert_to($json${"Esquema":"vec.contacto_usuario.consulta.v1","SujetoRef":"per_aaaaaaaaaaaaaaaaaaaaaa","FinalidadRef":"envio_llamamiento","Audiencia":"vec.contacto_usuario.consulta.v1","Version":1}$json$,'UTF8');
    recurso:=convert_to($json${"ambitos":{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa"},"atributos":{"auditoria_sha256":"8bdefaff9f484ef67d15122b2a74a47f7a7fd00471eda8af685146a9ed5cef02","contacto_finalidad_ref":"envio_llamamiento","contacto_sujeto_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","contacto_version":"1","contacto_version_esperada":"1","material_sha256":"c93735dbd56c49bc63d0093d0e5a0e0b16b014f1a103c83597f6729e94e89538"}}$json$,'UTF8');
    decision:=convert_to($json${"decision_ref":"decision_prueba_contacto","concedida":true,"codigo":"concedida","accion":"vec.contacto_usuario.consultar","modulo_id":"vec.module.usuarios","tipo_recurso":"contacto_usuario","recurso_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","finalidad":"envio_llamamiento","contexto_recurso_huella_sha256":"94918aea62619dcc8ac7dcaf72c530ebc110e0d280276acdfe85ac6d20f17bf0","campos_permitidos":[],"obligaciones":[],"perfil_activo_ref":"perfil_prueba_contacto","version_rol_ref":"rol_prueba_contacto_v1","correlacion_ref":"corr_prueba_contacto","vinculo_autenticacion_actor":{"metodo_observado":"certificado","garantia_observada":"alto"},"principal_id":"principal_prueba_contacto"}$json$,'UTF8');
    contexto:=convert_to($json${"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","principal_ref":"principal_prueba_contacto","perfil_activo_ref":"perfil_prueba_contacto","metodo":"certificado","garantia":"alto"}$json$,'UTF8');
    PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
        'vec.contacto_usuario.consultar',negocio,recurso,auditoria,decision,contexto);
    fallo:=false;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.contacto_validar_material_v1(
            'vec.contacto_usuario.consultar',negocio,recurso,
            convert_to(replace(convert_from(auditoria,'UTF8'),'hmac-sha256:prueba:','hmac-sha256:ajena:'),'UTF8'),decision,contexto);
    EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
    END;
    IF NOT fallo THEN RAISE EXCEPTION 'foco35: auditoría de lectura ajena aceptada'; END IF;
END $foco$;
ROLLBACK;
