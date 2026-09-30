-- Ensayo estructural y negativo de AD3-117 en base desechable.
-- La prueba positiva V3 exige capacidad y COSE sinteticos emitidos por el
-- proveedor confiable; este SQL no finge una decision autorizada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $prueba$
DECLARE
    f regprocedure;
    helper regprocedure := 'vec_autorizacion_atestada_v3.autorizar_lectura_categorias_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    rol text;
    x record;
    mensaje text;
    material jsonb := '{"catalogo_id":"rpt-demo","modulo_id":"personal","cursor_categoria_id":null,"limite":1}'::jsonb;
BEGIN
    IF pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',helper,'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_personal_ejecutor',helper,'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',helper,'EXECUTE')
       OR pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario',
           'vec_catalogos_configurables.publicacion','SELECT') THEN
        RAISE EXCEPTION 'AD3-117: helper o tablas accesibles directamente';
    END IF;
    FOREACH f IN ARRAY ARRAY[
        'vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p
                       WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
                         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN
            RAISE EXCEPTION 'AD3-117: propietario o entorno de fachada incorrecto';
        END IF;
        FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
          FROM pg_catalog.pg_proc AS p
          CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
         WHERE p.oid=f LOOP
            IF x.privilege_type<>'EXECUTE' OR x.is_grantable
               OR x.grantee=0
               OR x.grantee NOT IN (x.proowner,
                    'vec_contratacion_temporal_ejecutor'::regrole,
                    'vec_bolsa_llamamientos_ejecutor'::regrole,
                    'vec_personal_ejecutor'::regrole) THEN
                RAISE EXCEPTION 'AD3-117: ACL de fachada abierta';
            END IF;
        END LOOP;
        FOREACH rol IN ARRAY ARRAY['vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor'] LOOP
            IF NOT pg_catalog.has_function_privilege(rol,f,'EXECUTE')
               OR NOT pg_catalog.has_schema_privilege(rol,'vec_autorizacion_atestada_v3','USAGE') THEN
                RAISE EXCEPTION 'AD3-117: rol tecnico sin fachada';
            END IF;
        END LOOP;
    END LOOP;
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint AS c
                   WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
                     AND c.conname='clave_capacidad_version_audiencia_consumo_check'
                     AND c.convalidated
                     AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid),
                         'vec_catalogos_configurables.lectura_categorias.v1')>0) THEN
        RAISE EXCEPTION 'AD3-117: audiencia no cerrada';
    END IF;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
            '{}'::jsonb,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: lista sin descriptor admitida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
            pg_catalog.jsonb_set(material,'{limite}','"1"'::jsonb),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: limite string admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
            pg_catalog.jsonb_set(material,'{catalogo_id}','true'::jsonb),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: catalogo boolean admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
            material,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: lista sin V3 admitida';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(
            '{"catalogo_id":"rpt-demo","modulo_id":"personal","consumidor":"bolsa","uso_ref":"uso:uno","reserva_recibo_ref":"recibo:uno"}'::jsonb,
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: consumidor ajeno admitido';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
            pg_catalog.jsonb_build_object('catalogo_id','rpt-demo','modulo_id','personal',
                'version',2147483647,'huella_sha256',pg_catalog.repeat('0',64),'categoria_id','cat-demo'),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: version maxima sin V3 admitida';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
            pg_catalog.jsonb_build_object('catalogo_id','rpt-demo','modulo_id','personal',
                'version',2147483648,'huella_sha256',pg_catalog.repeat('0',64),'categoria_id','cat-demo'),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: version fuera de integer admitida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
            pg_catalog.jsonb_build_object('catalogo_id','rpt-demo','modulo_id','personal',
                'version','1','huella_sha256',pg_catalog.repeat('0',64),'categoria_id','cat-demo'),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: version string admitida';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(
            pg_catalog.jsonb_build_object('catalogo_id','rpt-demo','modulo_id','personal',
                'consumidor','bolsa','uso_ref',1234,'reserva_recibo_ref','recibo:uno'),
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: uso_ref numerico admitido';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    -- La principal revoca TEMP: esta prueba no puede crear tablas sombra.
    -- La suplantacion dinamica pertenece a un fixture PostgreSQL aislado con
    -- TEMP. Aqui se comprueba la ACL y la denegacion real del nucleo sin V3.
    IF pg_catalog.has_database_privilege(current_user,current_database(),'TEMP') THEN
        RAISE EXCEPTION 'AD3-117: TEMP disponible al propietario en la principal';
    END IF;
    PERFORM pg_catalog.set_config('search_path','pg_temp,pg_catalog',true);
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
            'lectura_categorias',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-117: consumo sin decision admitido';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
        IF mensaje<>'consumo VEC-AD-3 rechazado' THEN
            RAISE EXCEPTION 'AD3-117: rechazo posterior no prueba el cierre del nucleo';
        END IF;
    END;
END $prueba$;
ROLLBACK;
