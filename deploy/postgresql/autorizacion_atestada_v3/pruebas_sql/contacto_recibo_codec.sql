\set ON_ERROR_STOP on
-- Fixture sintético del codec36; no prueba concesión, firmas, ACL ni persistencia.
-- Fuente preparada para ensayo coordinado, no ejecutada por estos tests Python.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
DO $foco$
DECLARE
    auditoria bytea:=convert_to($json${"id":"","seq":0,"actor_id":"hmac-sha256:prueba:1111111111111111111111111111111111111111111111111111111111111111","actor_profile":"perfil_prueba_contacto","actor_roles":["rol_prueba_contacto_v1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"gestion_contacto_propio","action":"vec.contacto_usuario.consultar","module_id":"vec.module.usuarios","subject_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","object_version":1,"result":"accepted","correlation_ref":"corr_prueba_contacto","occurred_at":"2026-09-12T18:00:00Z","signature":""}$json$,'UTF8');
    negocio bytea:=convert_to($json${"Esquema":"vec.contacto_usuario.recibo.v1","SujetoRef":"per_aaaaaaaaaaaaaaaaaaaaaa","FinalidadRef":"gestion_contacto_propio","Audiencia":"vec.contacto_usuario.recibo.v1","Version":1}$json$,'UTF8');
    recurso bytea:=convert_to($json${"ambitos":{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa"},"atributos":{"auditoria_sha256":"d859cdc75aba68a8c0aa98bbc5e9763ab9869fdcd5d9066a9c41dd473687f57a","contacto_finalidad_ref":"gestion_contacto_propio","contacto_sujeto_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","contacto_version":"1","contacto_version_esperada":"1","material_sha256":"25d783235554e7b8258d40cf49e2d5bd33b6000e64dbc679901949996a6f3ca0"}}$json$,'UTF8');
    decision bytea:=convert_to($json${"decision_ref":"decision_prueba_contacto","concedida":true,"codigo":"concedida","accion":"vec.contacto_usuario.consultar","modulo_id":"vec.module.usuarios","tipo_recurso":"contacto_usuario","recurso_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","finalidad":"gestion_contacto_propio","contexto_recurso_huella_sha256":"71d49b24495ac974fcfdf2a30b3cd41fe7cc20d7b2b549ca6e54825a0ff2a3c6","campos_permitidos":[],"obligaciones":[],"perfil_activo_ref":"perfil_prueba_contacto","version_rol_ref":"rol_prueba_contacto_v1","correlacion_ref":"corr_prueba_contacto","vinculo_autenticacion_actor":{"metodo_observado":"certificado","garantia_observada":"alto"},"principal_id":"principal_prueba_contacto"}$json$,'UTF8');
    contexto bytea:=convert_to($json${"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","principal_ref":"principal_prueba_contacto","perfil_activo_ref":"perfil_prueba_contacto","metodo":"certificado","garantia":"alto"}$json$,'UTF8');
    v bytea[]; casos bytea[][]; n integer; fallo boolean;
BEGIN
    PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
        'vec.contacto_usuario.consultar',negocio,recurso,auditoria,decision,contexto);
    FOR n IN 1..5 LOOP
        v:=ARRAY[negocio,recurso,auditoria,decision,contexto]; v[n]:=NULL; fallo:=false;
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
                'vec.contacto_usuario.consultar',v[1],v[2],v[3],v[4],v[5]);
        EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
        END;
        IF NOT fallo THEN RAISE EXCEPTION 'foco36: NULL aceptado'; END IF;
    END LOOP;
    casos:=ARRAY[
        ARRAY[negocio,recurso,auditoria,decision,convert_to(replace(convert_from(contexto,'UTF8'),'per_aaaaaaaaaaaaaaaaaaaaaa','per_bbbbbbbbbbbbbbbbbbbbbb'),'UTF8')],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"obligaciones":[]','"obligaciones":["firma"]'),'UTF8'),contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"campos_permitidos":[]','"campos_permitidos":["correo"]'),'UTF8'),contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'"concedida":true','"concedida":null'),'UTF8'),contexto],
        ARRAY[negocio,recurso,convert_to(replace(convert_from(auditoria,'UTF8'),'hmac-sha256:prueba:','hmac-sha256:ajena:'),'UTF8'),decision,contexto],
        ARRAY[negocio,convert_to(replace(convert_from(recurso,'UTF8'),'"contacto_version":"1"','"contacto_version":"2"'),'UTF8'),auditoria,decision,contexto],
        ARRAY[negocio,recurso,convert_to(replace(convert_from(auditoria,'UTF8'),'"signature":""','"signature":"","authorization_ref":"anticipada"'),'UTF8'),decision,contexto],
        ARRAY[convert_to(replace(convert_from(negocio,'UTF8'),'recibo.v1','consulta.v1'),'UTF8'),recurso,auditoria,decision,contexto],
        ARRAY[convert_to(replace(convert_from(negocio,'UTF8'),'"Version":1','"Version":1,"Version":1'),'UTF8'),recurso,auditoria,decision,contexto],
        ARRAY[negocio,recurso,auditoria,convert_to(replace(convert_from(decision,'UTF8'),'gestion_contacto_propio','envio_llamamiento'),'UTF8'),contexto]
    ];
    FOREACH v SLICE 1 IN ARRAY casos LOOP
        fallo:=false;
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
                'vec.contacto_usuario.consultar',v[1],v[2],v[3],v[4],v[5]);
        EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
        END;
        IF NOT fallo THEN RAISE EXCEPTION 'foco36: variante aceptada'; END IF;
    END LOOP;
    fallo:=false;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.contacto_recibo_validar_material_v1(
            NULL,negocio,recurso,auditoria,decision,contexto);
    EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN fallo:=true;
    END;
    IF NOT fallo THEN RAISE EXCEPTION 'foco36: acción NULL aceptada'; END IF;
END $foco$;
ROLLBACK;
