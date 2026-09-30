BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
DO $prueba$
BEGIN
    IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
        'aut_0123456789abcdefghijkl',repeat('a',64),
        'ase_0123456789abcdefghijkl','ses_0123456789abcdefghijkl',
        'cta_0123456789abcdefghijkl','cta_0123456789abcdefghijkl',
        false,'externa_personal','certificado','alto',
        'pga_0123456789abcdefghijkl',repeat('b',64),
        clock_timestamp(),clock_timestamp(),
        'cse_0123456789abcdefghijkl','1','activa',repeat('c',64),
        clock_timestamp(),clock_timestamp()+interval '1 minute') IS NOT FALSE THEN
        RAISE EXCEPTION 'LOGIN no aprobado accedio a identidad externa';
    END IF;
    IF has_table_privilege('vec_autorizacion_propietario',
           'vec_identidad_externa_v1.cuenta','SELECT')
       OR has_table_privilege('vec_autorizacion_propietario',
           'vec_identidad_externa_v1.llamante_autorizacion','SELECT')
       OR has_function_privilege('vec_autorizacion_propietario',
           'vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)',
           'EXECUTE') THEN
        RAISE EXCEPTION 'AUT obtuvo lectura directa de Identidad externa';
    END IF;
    IF NOT has_function_privilege('vec_autorizacion_propietario',
           'vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)',
           'EXECUTE') THEN
        RAISE EXCEPTION 'AUT carece de la fachada nominal';
    END IF;
END $prueba$;
ROLLBACK;
