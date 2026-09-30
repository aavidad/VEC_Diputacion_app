-- Base desechable con roles_up, 000001 UP y 000002 UP instalados.
-- Los hechos sinteticos de esta prueba se revierten.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $prueba$
DECLARE
    doc1 text := '{"id":"rpt-read-demo","modulo_id":"personal","version":1,"estado":"publicado","entradas":[{"clave":"cat-alfa","etiqueta":"Alfa"},{"clave":"cat-beta","etiqueta":"Beta"}]}';
    doc2 text := '{"id":"rpt-read-demo","modulo_id":"personal","version":2,"estado":"publicado","entradas":[{"clave":"cat-beta","etiqueta":"Beta nueva"},{"clave":"cat-gamma","etiqueta":"Gamma"}]}';
    h1 text;
    h2 text;
    pre jsonb;
    hp text;
    -- Referencia de entrada versionada, sintetica y con la forma del dominio.
    motivo_ref text := 'motivos_rpt:1:motivo_11111111111111111111111111111111';
    vacia_h text := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{}','UTF8')),'hex');
    pagina jsonb;
    segunda jsonb;
    historica jsonb;
    uso jsonb;
BEGIN
    IF pg_catalog.has_table_privilege('vec_autorizacion_atestada_v3_propietario',
        'vec_catalogos_configurables.categoria_control','SELECT')
       OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',
        'vec_catalogos_configurables.listar_habilitadas(text,text,integer)','EXECUTE') THEN
        RAISE EXCEPTION 'lectura directa expuesta';
    END IF;
    h1 := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc1,'UTF8')),'hex');
    h2 := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc2,'UTF8')),'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-read-demo',1,h1,doc1,'{}'::jsonb,vacia_h,
        'aprobacion:a1','aprobacion:b1','actor:uno','decision:pub1','recibo:pub1',motivo_ref);
    pre := pg_catalog.jsonb_build_object('cat-beta',pg_catalog.jsonb_build_object(
        'version',1,'huella_sha256',h1,'revision',1,'estado','habilitada'));
    hp := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex');
    PERFORM vec_catalogos_configurables.publicar('rpt-read-demo',2,h2,doc2,pre,hp,
        'aprobacion:a2','aprobacion:b2','actor:uno','decision:pub2','recibo:pub2',motivo_ref);

    pagina := vec_catalogos_configurables.listar_habilitadas('rpt-read-demo',NULL,2);
    IF pagina->>'encontrado'<>'true'
       OR pagina#>>'{datos,anclaje_publicacion,documento_canonico}'<>doc1
       OR pagina#>>'{datos,anclaje_publicacion,huella_sha256}'<>h1
       OR pg_catalog.jsonb_array_length(pagina#>'{datos,items}')<>2
       OR pagina#>>'{datos,items,0,categoria_id}'<>'cat-alfa'
       OR pagina#>>'{datos,items,0,version}'<>'1'
       OR pagina#>>'{datos,items,1,categoria_id}'<>'cat-beta'
       OR pagina#>>'{datos,items,1,version}'<>'2'
       OR pagina#>>'{datos,hay_mas}'<>'true'
       OR pagina#>>'{datos,siguiente_cursor}'<>'cat-beta'
       OR pg_catalog.jsonb_array_length(pagina#>'{datos,publicaciones}')<>1
       OR pagina#>>'{datos,publicaciones,0,documento_canonico}'<>doc2 THEN
        RAISE EXCEPTION 'pagina multiversion incorrecta';
    END IF;
    segunda := vec_catalogos_configurables.listar_habilitadas('rpt-read-demo','cat-beta',2);
    IF pg_catalog.jsonb_array_length(segunda#>'{datos,items}')<>1
       OR segunda#>>'{datos,anclaje_publicacion,documento_canonico}'<>doc1
       OR segunda#>>'{datos,items,0,categoria_id}'<>'cat-gamma'
       OR segunda#>>'{datos,hay_mas}'<>'false'
       OR segunda#>'{datos,siguiente_cursor}'<>'null'::jsonb
       OR pg_catalog.jsonb_array_length(segunda#>'{datos,publicaciones}')<>1 THEN
        RAISE EXCEPTION 'cursor o fin de pagina incorrecto';
    END IF;
    IF vec_catalogos_configurables.listar_habilitadas('rpt-ausente',NULL,10)->>'encontrado'<>'false'
       OR vec_catalogos_configurables.listar_habilitadas('rpt-read-demo','cat-zeta',10)#>>'{datos,hay_mas}'<>'false'
       OR vec_catalogos_configurables.listar_habilitadas('rpt-read-demo','cat-zeta',10)#>>'{datos,anclaje_publicacion,documento_canonico}'<>doc1
       OR pg_catalog.jsonb_array_length(vec_catalogos_configurables.listar_habilitadas('rpt-read-demo','cat-zeta',10)#>'{datos,publicaciones}')<>0 THEN
        RAISE EXCEPTION 'catalogo ausente o pagina vacia incorrectos';
    END IF;

    historica := vec_catalogos_configurables.leer_publicacion_categoria('rpt-read-demo',1,h1,'cat-beta');
    IF historica->>'encontrado'<>'true'
       OR historica#>>'{datos,publicacion,documento_canonico}'<>doc1
       OR historica#>>'{datos,control_actual,version}'<>'2'
       OR historica#>>'{datos,entrada,clave}'<>'cat-beta'
       OR vec_catalogos_configurables.leer_publicacion_categoria('rpt-read-demo',1,h2,'cat-beta')->>'encontrado'<>'false' THEN
        RAISE EXCEPTION 'historia exacta o control actual incorrectos';
    END IF;
    PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-alfa',1,'deshabilitar',NULL,NULL,
        'actor:uno','decision:des-alfa','recibo:des-alfa',motivo_ref);
    historica := vec_catalogos_configurables.leer_publicacion_categoria('rpt-read-demo',1,h1,'cat-alfa');
    IF historica#>>'{datos,control_actual,estado}'<>'deshabilitada'
       OR pg_catalog.jsonb_array_length(vec_catalogos_configurables.listar_habilitadas('rpt-read-demo',NULL,10)#>'{datos,items}')<>2 THEN
        RAISE EXCEPTION 'deshabilitacion altero historia o lista';
    END IF;

    PERFORM vec_catalogos_configurables.reservar('contratacion-temporal','expediente:uno',
        'cat-beta','rpt-read-demo',2,h2,'actor:uno','decision:reserva','recibo:reserva',motivo_ref);
    PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-beta',2,'deshabilitar',NULL,NULL,
        'actor:uno','decision:des-beta','recibo:des-beta',motivo_ref);
    uso := vec_catalogos_configurables.consultar_uso('contratacion-temporal','expediente:uno','recibo:reserva');
    IF uso#>>'{datos,estado}'<>'reservado'
       OR uso#>>'{datos,huella_sha256}'<>h2
       OR uso#>'{datos,terminal_recibo_ref}'<>'null'::jsonb
       OR vec_catalogos_configurables.consultar_uso('bolsa','expediente:uno','recibo:reserva')->>'encontrado'<>'false'
       OR vec_catalogos_configurables.consultar_uso('contratacion-temporal','expediente:uno','recibo:ajeno')->>'encontrado'<>'false' THEN
        RAISE EXCEPTION 'consulta de reserva no exacta';
    END IF;
    PERFORM vec_catalogos_configurables.terminar_uso('contratacion-temporal','expediente:uno',
        'recibo:reserva','confirmado','actor:uno','decision:terminal','recibo:terminal',motivo_ref);
    uso := vec_catalogos_configurables.consultar_uso('contratacion-temporal','expediente:uno','recibo:reserva');
    IF uso#>>'{datos,estado}'<>'confirmado' OR uso#>>'{datos,terminal_recibo_ref}'<>'recibo:terminal' THEN
        RAISE EXCEPTION 'conclusion de reserva previa perdida';
    END IF;
    BEGIN
        PERFORM vec_catalogos_configurables.reservar('contratacion-temporal','expediente:dos',
            'cat-beta','rpt-read-demo',2,h2,'actor:uno','decision:nueva','recibo:nueva',motivo_ref);
        RAISE EXCEPTION 'deshabilitada admitio uso nuevo';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    PERFORM vec_catalogos_configurables.cambiar_proyeccion('cat-gamma',1,'deshabilitar',NULL,NULL,
        'actor:uno','decision:des-gamma','recibo:des-gamma',motivo_ref);
    pagina := vec_catalogos_configurables.listar_habilitadas('rpt-read-demo',NULL,10);
    IF pagina->>'encontrado'<>'true'
       OR pg_catalog.jsonb_array_length(pagina#>'{datos,items}')<>0
       OR pg_catalog.jsonb_array_length(pagina#>'{datos,publicaciones}')<>0
       OR pagina#>>'{datos,anclaje_publicacion,documento_canonico}'<>doc1 THEN
        RAISE EXCEPTION 'catalogo sin habilitadas perdio anclaje verificable';
    END IF;
END $prueba$;
ROLLBACK;
