-- Base desechable con 000001 corregida, 000002 y 000003. ROLLBACK final.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
DO $configuracion$
DECLARE f regprocedure;
BEGIN
    FOREACH f IN ARRAY ARRAY[
      'vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)'::regprocedure,
      'vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)'::regprocedure
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f AND prosecdef
             AND proowner='vec_catalogos_configurables_propietario'::regrole
             AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
           OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE')
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
               WHERE p.oid=f AND (a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable
                 OR a.grantee NOT IN (p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
            RAISE EXCEPTION 'funcion del catalogo con configuracion o ACL incorrecta';
        END IF;
    END LOOP;
END $configuracion$;
DO $prueba$
DECLARE
    doc text := '{"id":"rpt-usos-demo","modulo_id":"personal","version":1,"estado":"publicado","entradas":[{"clave":"cat-uno","etiqueta":"Uno"}]}';
    h text; vacia_h text; r jsonb; motivo text := 'motivos_autorizacion:1:motivo_reserva';
BEGIN
    IF pg_catalog.has_function_privilege('vec_personal_ejecutor',
        'vec_catalogos_configurables.obtener_uso_publicacion(text,text,text)','EXECUTE')
       OR pg_catalog.has_function_privilege('vec_personal_ejecutor',
        'vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text)','EXECUTE')
       OR pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario',
        'vec_catalogos_configurables.evidencia_terminal','SELECT') THEN
        RAISE EXCEPTION 'core 000003 expuesto fuera de autoridad';
    END IF;
    h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc,'UTF8')),'hex');
    vacia_h := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-usos-demo',1,h,doc,'{}'::jsonb,vacia_h,
        'aprobacion:a1','aprobacion:b1','actor:uno','decision:pub1','recibo:pub1',motivo);
    PERFORM vec_catalogos_configurables.reservar('personal','plan:uno','cat-uno','rpt-usos-demo',1,h,
        'actor:uno','decision:res1','recibo:res1',motivo);
    r := vec_catalogos_configurables.obtener_uso_publicacion('personal','plan:uno','recibo:res1');
    IF r#>>'{datos,uso,estado}' IS DISTINCT FROM 'reservado'
       OR r#>>'{datos,publicacion,documento_canonico}' IS DISTINCT FROM doc
       OR r#>>'{datos,publicacion,huella_sha256}' IS DISTINCT FROM h
       OR vec_catalogos_configurables.obtener_uso_publicacion('bolsa','plan:uno','recibo:res1')->>'encontrado' IS DISTINCT FROM 'false' THEN
        RAISE EXCEPTION 'lectura volatile o publicacion historica incorrecta';
    END IF;
    PERFORM vec_catalogos_configurables.terminar_uso_con_evidencia(
        'personal','plan:uno','recibo:res1','confirmado','actor:uno','decision:term1',
        'recibo:term1',motivo,'evidencia:efecto1',pg_catalog.repeat('a',64));
    r := vec_catalogos_configurables.obtener_uso_publicacion('personal','plan:uno','recibo:res1');
    IF r#>>'{datos,uso,estado}' IS DISTINCT FROM 'confirmado'
       OR r#>>'{datos,uso,revision}' IS DISTINCT FROM '2'
       OR r#>>'{datos,uso,terminal_recibo_ref}' IS DISTINCT FROM 'recibo:term1' THEN
        RAISE EXCEPTION 'terminal o readback incorrecto';
    END IF;
    IF vec_catalogos_configurables.reservar('personal','plan:uno','cat-uno','rpt-usos-demo',1,h,
        'actor:uno','decision:res2','recibo:res1',motivo) IS DISTINCT FROM 'recibo:res1'
       OR vec_catalogos_configurables.terminar_uso_con_evidencia(
        'personal','plan:uno','recibo:res1','confirmado','actor:uno','decision:term2',
        'recibo:term1',motivo,'evidencia:efecto1',pg_catalog.repeat('a',64)) IS DISTINCT FROM 'recibo:term1' THEN
        RAISE EXCEPTION 'replay exacto no conservo recibo';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.terminar_uso_con_evidencia(
          'personal','plan:uno','recibo:res1','confirmado','actor:uno','decision:term3',
          'recibo:term1',motivo,'evidencia:otra',pg_catalog.repeat('b',64));
        RAISE EXCEPTION 'replay alterado admitido';
    EXCEPTION WHEN SQLSTATE '23505' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.reservar('personal','plan:dos','cat-uno','rpt-usos-demo',1,h,
        'actor:uno','decision:res3','recibo:res3',motivo);
    PERFORM vec_catalogos_configurables.terminar_uso('personal','plan:dos','recibo:res3',
        'cancelado','actor:uno','decision:legacy','recibo:legacy',motivo);
    BEGIN
        PERFORM vec_catalogos_configurables.terminar_uso_con_evidencia(
          'personal','plan:dos','recibo:res3','cancelado','actor:uno','decision:late',
          'recibo:legacy',motivo,'evidencia:retro',pg_catalog.repeat('c',64));
        RAISE EXCEPTION 'legacy terminal rellenado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.reservar('personal','plan:tres','cat-uno','rpt-usos-demo',1,h,
        'actor:uno','decision:res4','recibo:res4',motivo);
    PERFORM vec_catalogos_configurables.terminar_uso_con_evidencia(
        'personal','plan:tres','recibo:res4','cancelado','actor:uno','decision:term4',
        'recibo:term4',motivo,'evidencia:sin-efecto',pg_catalog.repeat('d',64));
    r := vec_catalogos_configurables.obtener_uso_publicacion('personal','plan:tres','recibo:res4');
    IF r#>>'{datos,uso,estado}' IS DISTINCT FROM 'cancelado' THEN
        RAISE EXCEPTION 'cancelacion no terminal';
    END IF;
END $prueba$;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
DO $inmutable$
BEGIN
    IF (SELECT count(*) FROM vec_catalogos_configurables.evidencia_terminal)<>2 THEN
        RAISE EXCEPTION 'numero de vinculos terminales incorrecto';
    END IF;
    BEGIN
        UPDATE vec_catalogos_configurables.evidencia_terminal SET evidencia_ref='evidencia:otra'
         WHERE consumidor='personal' AND uso_ref='plan:uno';
        RAISE EXCEPTION 'vinculo terminal reescrito';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_catalogos_configurables.evidencia_terminal;
        RAISE EXCEPTION 'vinculos terminales truncados';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END $inmutable$;
ROLLBACK;
