-- La proyeccion es reconstruible, pero su retirada interrumpe el portal. Si
-- contiene filas exige una confirmacion operativa explicita.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000001', 0)
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v1', 0)
);

DO $prevalidacion$
BEGIN
	IF current_user <> 'vec_bolsa_publica_migrador'
	   OR NOT pg_catalog.pg_has_role(
	       current_user, 'vec_bolsa_publica_propietario', 'SET'
	   )
	   OR NOT pg_catalog.pg_has_role(
	       current_user, 'vec_bolsa_publica_publicacion_propietario', 'SET'
	   ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversion de bolsa publica rechazada: identidad incorrecta';
    END IF;
    IF NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_datos'
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_lectura'
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_publicacion'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion de bolsa publica: esquema ausente';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_bolsa_publica_propietario;

DO $confirmacion$
DECLARE
	contiene_datos boolean;
BEGIN
	SELECT EXISTS (
		SELECT 1 FROM vec_bolsa_publica_datos.fuente
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.catalogo_publico
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.entrada_catalogo_publico
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.catalogo_categorias
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.categoria_publica
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.convocatoria_publica
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.convocatoria_categoria
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.plazo_convocatoria
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.requisito_convocatoria
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.documento_convocatoria
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.ayuda_convocatoria
		UNION ALL SELECT 1 FROM vec_bolsa_publica_datos.manifiesto_consumido
	) INTO contiene_datos;
	IF contiene_datos AND current_setting(
		'vec.confirmar_retirada_proyeccion_bolsa_publica', true
	) IS DISTINCT FROM 'RETIRAR_PROYECCION_BOLSA_PUBLICA_RECONSTRUIBLE' THEN
		RAISE EXCEPTION USING ERRCODE = '55000',
			MESSAGE = 'reversion rechazada: la proyeccion contiene datos';
    END IF;
END
$confirmacion$;

DROP SCHEMA vec_bolsa_publica_lectura CASCADE;
SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
DROP SCHEMA vec_bolsa_publica_publicacion CASCADE;
SET LOCAL ROLE vec_bolsa_publica_propietario;
DROP SCHEMA vec_bolsa_publica_datos CASCADE;
COMMIT;
