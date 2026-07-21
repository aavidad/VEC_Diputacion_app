SET search_path = pg_catalog;
DO $pruebas$
BEGIN
    IF vec_contexto_actor_v1.referencia_operacion_valida(
         'oca_' || repeat('a',24),'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(
         'rca_' || repeat('a',24),'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(
         'oca_' || repeat('a',23),'oca_') IS NOT FALSE
       OR vec_contexto_actor_v1.referencia_valida(
         'cta_' || repeat('a',22),'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
         'cta_' || repeat('a',21),'cta_') IS NOT FALSE THEN
        RAISE EXCEPTION 'limites de referencias oca_/rca_/componentes incoherentes';
    END IF;
    BEGIN
        INSERT INTO vec_contexto_actor_v1.procedencias VALUES
         ('prc_fixture_sintetico_no_corporativo_01',1,repeat('9',64),'no_autoritativa');
        RAISE EXCEPTION 'dos revisiones compartieron referencia y version';
    EXCEPTION WHEN SQLSTATE '23505' THEN NULL;
    END;
    INSERT INTO vec_contexto_actor_v1.procedencias VALUES
     ('prc_borde_uint64_maximo_0000000001',18446744073709551615,
      repeat('8',64),'no_autoritativa');
    BEGIN
        INSERT INTO vec_contexto_actor_v1.procedencias VALUES
         ('prc_borde_uint64_cero_00000000001',0,repeat('7',64),'no_autoritativa');
        RAISE EXCEPTION 'version cero fue aceptada';
    EXCEPTION WHEN SQLSTATE '23514' THEN NULL;
    END;
    BEGIN
        INSERT INTO vec_contexto_actor_v1.procedencias VALUES
         ('prc_borde_uint64_exceso_000000001',18446744073709551616,
          repeat('6',64),'no_autoritativa');
        RAISE EXCEPTION 'version superior a uint64 fue aceptada';
    EXCEPTION WHEN SQLSTATE '23514' THEN NULL;
    END;
    BEGIN
        UPDATE vec_contexto_actor_v1.proyeccion_cuenta_versiones SET estado='revocado'
         WHERE cuenta_ref='cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa' AND version=1;
        RAISE EXCEPTION 'UPDATE de historia fue aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_contexto_actor_v1.persona_versiones
         WHERE persona_ref='per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb' AND version=1;
        RAISE EXCEPTION 'DELETE de historia fue aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
    BEGIN
        TRUNCATE vec_contexto_actor_v1.registros_contexto;
        RAISE EXCEPTION 'TRUNCATE de historia fue aceptado';
    EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
    END;
END
$pruebas$;
