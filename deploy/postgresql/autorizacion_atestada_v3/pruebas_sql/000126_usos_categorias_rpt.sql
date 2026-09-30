-- Sondas negativas de contrato sobre clon con 000003 y AD3-126. No fabrican V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $prueba$
DECLARE f regprocedure;
BEGIN
    FOREACH f IN ARRAY ARRAY[
      'vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ] LOOP
        IF NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
           OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
           OR NOT pg_catalog.has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
               WHERE p.oid=f AND a.grantee=0) THEN
            RAISE EXCEPTION 'ACL de fachada incorrecta';
        END IF;
    END LOOP;
    IF pg_catalog.has_function_privilege('vec_personal_ejecutor',
      'vec_autorizacion_atestada_v3.autorizar_uso_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR pg_catalog.has_function_privilege('vec_personal_ejecutor',
      'vec_autorizacion_atestada_v3.ejecutar_uso_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR pg_catalog.has_function_privilege('vec_personal_ejecutor',
      'vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)','EXECUTE') THEN
        RAISE EXCEPTION 'helper o core expuesto';
    END IF;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
            '{}'::jsonb,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'material incompleto admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":"personal","uso_ref":"uso:á","categoria_id":"cat-uno","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reserva_recibo_ref":"recibo:uno"}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'uso_ref fuera del recurso V3 nativo admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":"personal","uso_ref":"uso*uno","categoria_id":"cat-uno","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reserva_recibo_ref":"recibo:uno"}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'asterisco de recurso admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":true,"uso_ref":"uso:uno","categoria_id":"cat-uno","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reserva_recibo_ref":"recibo:uno"}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'consumidor boolean admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":"personal","uso_ref":"uso:uno","categoria_id":"cat-uno","version":"1","huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reserva_recibo_ref":"recibo:uno"}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'version string admitida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":"personal","uso_ref":"uso:uno","categoria_id":"cat-uno","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reserva_recibo_ref":"recibo:uno","terminal_recibo_ref":"recibo:term","evidencia_ref":"evidencia:uno","evidencia_sha256":1}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'evidencia numerica admitida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
END $prueba$;
ROLLBACK;
