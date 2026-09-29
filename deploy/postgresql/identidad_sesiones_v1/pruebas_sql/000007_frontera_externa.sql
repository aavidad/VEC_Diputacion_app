BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE
    cuenta constant text := 'cta_bolsa_candidata_sintetica_20260930';
    dominio constant text := 'idh_0123456789abcdefghijkl';
    esquema constant text := 'vec.identidad.hmac-sha256.v1';
    clave constant text := 'clave-desarrollo';
    cuenta_h bytea := decode(repeat('11',32),'hex');
    sujeto_h bytea := decode(repeat('22',32),'hex');
    aprobacion constant text := 'apr_0123456789abcdefghijkl';
    operacion constant text := 'opr_0123456789abcdefghijkl';
    alta record;
    material text;
    ausencia text;
    emitida timestamptz(6) := clock_timestamp() - interval '1 second';
    expira timestamptz(6) := clock_timestamp() + interval '2 minutes';
BEGIN
    material := vec_identidad_externa_v1.huella_material_alta_v1(
        cuenta,esquema,dominio,clave,1,cuenta_h,sujeto_h);
    ausencia := vec_identidad_externa_v1.huella_ausencia_v1(cuenta);
    INSERT INTO vec_identidad_externa_v1.aprobacion_alta(
        aprobacion_ref,huella_material_sha256,huella_preimagen_sha256,revision_esperada)
    VALUES (aprobacion,material,ausencia,0);
    IF vec_identidad_externa_v1.confirmar_alta_v1(
        operacion,aprobacion,cuenta,esquema,dominio,clave,1,cuenta_h,sujeto_h,
        material,ausencia,0) IS DISTINCT FROM cuenta THEN
        RAISE EXCEPTION 'alta externa aprobada denegada';
    END IF;
    IF vec_identidad_externa_v1.confirmar_alta_v1(
        operacion,aprobacion,cuenta,esquema,dominio,clave,1,cuenta_h,sujeto_h,
        material,ausencia,0) IS DISTINCT FROM cuenta THEN
        RAISE EXCEPTION 'replay de alta externa divergente';
    END IF;
    IF vec_identidad_externa_v1.confirmar_alta_v1(
        'opr_1234567890abcdefghijkl',aprobacion,cuenta,esquema,dominio,
        clave,1,cuenta_h,sujeto_h,material,ausencia,0) IS NOT NULL THEN
        RAISE EXCEPTION 'otra operacion apropio aprobacion';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.cuenta AS comun
               WHERE comun.cuenta_ref=cuenta) THEN
        RAISE EXCEPTION 'cuenta externa escrita en tabla comun';
    END IF;
    SELECT * INTO alta FROM vec_identidad_externa_v1.registrar_sesion_v1(
        'opr_abcdefghij0123456789kl',esquema,dominio,clave,1,
        decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
        sujeto_h,cuenta_h,NULL,false,'externa_personal','certificado','alto',
        repeat('a',64),emitida,emitida,expira,
        'pga_0123456789abcdefghijkl',repeat('b',64));
    IF alta.sesion_ref IS NULL THEN RAISE EXCEPTION 'sesion externa no registrada'; END IF;
    IF vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
        alta.autenticacion_ref,repeat('a',64),alta.asercion_ref,alta.sesion_ref,
        cuenta,cuenta,false,'externa_personal','certificado','alto',
        'pga_0123456789abcdefghijkl',repeat('b',64),emitida,emitida,
        alta.control_sesion_ref,alta.control_sesion_revision_texto,
        alta.control_sesion_estado,alta.control_sesion_huella_sha256,
        alta.sesion_revalidada_en,alta.sesion_valida_hasta) IS NOT TRUE THEN
        RAISE EXCEPTION 'sesion externa activa denegada';
    END IF;
    IF vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
        alta.autenticacion_ref,repeat('a',64),alta.asercion_ref,alta.sesion_ref,
        cuenta,cuenta,false,'interna_corporativa','certificado','alto',
        'pga_0123456789abcdefghijkl',repeat('b',64),emitida,emitida,
        alta.control_sesion_ref,alta.control_sesion_revision_texto,
        alta.control_sesion_estado,alta.control_sesion_huella_sha256,
        alta.sesion_revalidada_en,alta.sesion_valida_hasta) IS NOT FALSE THEN
        RAISE EXCEPTION 'sesion externa obtuvo superficie corporativa';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(
            alta.autenticacion_ref,alta.sesion_ref)) <> 1 THEN
        RAISE EXCEPTION 'revalidacion de actor externo ausente';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
        'opr_abcdefghij0123456789kl',esquema,dominio,clave,1,
        decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
        sujeto_h,cuenta_h,NULL,false,'externa_personal','certificado','alto',
        repeat('a',64),emitida,emitida,expira,
        'pga_0123456789abcdefghijkl',repeat('b',64))) <> 1 THEN
        RAISE EXCEPTION 'recuperacion de registro externo ausente';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
        'opr_abcdefghij0123456789kl',esquema,dominio,clave,1,
        decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
        sujeto_h,cuenta_h,NULL,false,'interna_corporativa','certificado','alto',
        repeat('a',64),emitida,emitida,expira,
        'pga_0123456789abcdefghijkl',repeat('b',64))) <> 0 THEN
        RAISE EXCEPTION 'recuperacion cruzo a superficie interna';
    END IF;
    IF vec_identidad_externa_v1.cambiar_estado_cuenta_v1(
        cuenta,1,'inactiva','opr_abcdefghij0123456789zz') IS DISTINCT FROM 2 THEN
        RAISE EXCEPTION 'revocacion de cuenta externa ausente';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(
            alta.autenticacion_ref,alta.sesion_ref)) <> 0 THEN
        RAISE EXCEPTION 'sesion resucito tras revocacion';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
        'opr_abcdefghij0123456789kl',esquema,dominio,clave,1,
        decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
        sujeto_h,cuenta_h,NULL,false,'externa_personal','certificado','alto',
        repeat('a',64),emitida,emitida,expira,
        'pga_0123456789abcdefghijkl',repeat('b',64))) <> 0 THEN
        RAISE EXCEPTION 'reconciliacion ignoro revocacion';
    END IF;
    IF vec_identidad_externa_v1.cambiar_estado_cuenta_v1(
        cuenta,2,'activa','opr_abcdefghij0123456789yy') IS DISTINCT FROM 3 THEN
        RAISE EXCEPTION 'reactivacion de cuenta externa ausente';
    END IF;
    IF (SELECT count(*) FROM vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(
            alta.autenticacion_ref,alta.sesion_ref)) <> 0 THEN
        RAISE EXCEPTION 'sesion antigua resucito tras reactivar';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion
               WHERE sesion_ref=alta.sesion_ref) THEN
        RAISE EXCEPTION 'consumo externo escrito en tabla comun';
    END IF;
END $prueba$;

DO $acl$
BEGIN
    IF has_schema_privilege('vec_identidad_externa_v1_registrador',
                            'vec_identidad_sesiones_v1','USAGE')
       OR has_table_privilege('vec_identidad_externa_v1_registrador',
                             'vec_identidad_externa_v1.cuenta','SELECT')
       OR has_function_privilege('vec_identidad_externa_v1_registrador',
            'vec_identidad_externa_v1.confirmar_alta_v1(text,text,text,text,text,text,bigint,bytea,bytea,text,text,bigint)',
            'EXECUTE')
       OR has_function_privilege('vec_identidad_externa_v1_revalidador',
            'vec_identidad_externa_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)',
            'EXECUTE') THEN
        RAISE EXCEPTION 'ACL externa demasiado amplia';
    END IF;
END $acl$;
ROLLBACK;
