\set ON_ERROR_STOP 1

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    recibo record;
    sesion record;
    vinculo jsonb;
    decision jsonb;
    alterada jsonb;
    decision_canonica bytea;
    motivo bytea;
    emitida timestamptz(6) := clock_timestamp();
    hasta timestamptz(6) := emitida + interval '2 minutes';
    z_emitida text;
    z_hasta text;
    clave text;
    obtenido record;
    primera_huella text;
    primera_fecha timestamptz;
BEGIN
    SELECT * INTO STRICT recibo
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_registro_v3_000000000000000000000000';
    SELECT base.autenticacion_verificada_en, base.sesion_emitida_en,
           control.sesion_revalidada_en, control.sesion_valida_hasta
      INTO STRICT sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_autorizacion.control_sesion_v1 AS control
        USING (sesion_ref)
     WHERE base.sesion_ref =
           'ses_registro_v3_0000000000000000000000';
    z_emitida := to_char(
        emitida AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    z_hasta := to_char(
        hasta AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    motivo := convert_to(
      '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_id":"motivos_v3","catalogo_version":1,"catalogo_huella_sha256":"' ||
      repeat('9',64) ||
      '","entrada_clave":"motivo_33333333333333333333333333333333"}}',
      'UTF8'
    );
    vinculo := jsonb_build_object(
      'esquema','vec.autenticacion-actor.vinculo.v2.contexto-registrado',
      'bloque_version',2,
      'autenticacion_ref','aut_registro_v3_0000000000000000000000',
      'autenticacion_huella_sha256',repeat('5',64),
      'asercion_ref','ase_registro_v3_0000000000000000000000',
      'sesion_ref','ses_registro_v3_0000000000000000000000',
      'control_sesion_ref','cse_registro_v3_0000000000000000000000',
      'control_sesion_revision',1,
      'control_sesion_huella_sha256',repeat('7',64),
      'cuenta_ref','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
      'cuenta_ordinaria_ref','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
      'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
      'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
      'cuenta_privilegiada',false,'superficie','interna_corporativa',
      'metodo_observado','certificado','garantia_observada','alto',
      'politica_garantia_ref','pga_registro_v3_0000000000000000000000',
      'politica_garantia_huella_sha256',repeat('6',64),
      'autenticacion_verificada_en',to_char(
          sesion.autenticacion_verificada_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'sesion_emitida_en',to_char(sesion.sesion_emitida_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'sesion_valida_hasta',to_char(sesion.sesion_valida_hasta AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'sesion_revalidada_en',to_char(sesion.sesion_revalidada_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'registro_contexto_ref',recibo.registro_contexto_ref,
      'contexto_actor_esquema','vec.contexto-actor.vinculado.v2',
      'contexto_actor_ref','vca_sintetico_dddddddddddddddddddddddd',
      'contexto_actor_version',2,'contexto_actor_cuenta_version',2,
      'contexto_actor_huella_sha256',recibo.huella_sha256,
      'manifiesto_procedencia_huella_sha256',
          recibo.manifiesto_procedencia_huella_sha256,
      'autoridad_efectiva','autoridad_maestra_acreditada'
    );
    decision := jsonb_build_object(
      'esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2',
      'bloque_version',3,'decision_ref','decision:registro-v3:positiva',
      'concedida',true,'codigo','concedida',
      'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
      'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
      'accion','consultar','recurso_ref','expediente_000000000000000000000000',
      'modulo_id','bolsa','tipo_recurso','expediente',
      'contexto_recurso_huella_sha256',repeat('a',64),'finalidad','gestion',
      'correlacion_ref','correlacion_11111111111111111111111111111111',
      'esquema_huella_solicitud',
          'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2',
      'solicitud_huella_sha256',repeat('b',64),
      'esquema_huella_motivo',
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
      'motivo_huella_sha256',encode(sha256(motivo),'hex'),
      'vinculo_autenticacion_actor',vinculo,
      'asignacion_ref','asignacion:registro_v3:v1',
      'asignacion_huella_sha256',repeat('4',64),
      'version_rol_ref','rol:registro_v3:v1',
      'version_rol_huella_sha256',repeat('2',64),
      'control_vigencia_version_rol_ref','rol:registro_v3:v1',
      'control_vigencia_version_rol_revision',1,
      'control_vigencia_version_rol_huella_sha256',repeat('3',64),
      'revision_catalogo_politicas',1,
      'catalogo_politicas_huella_sha256',
          '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
      'politicas_evaluadas','[]'::jsonb,'politicas_aplicables','[]'::jsonb,
      'garantia_minima','alto','campos_permitidos',jsonb_build_array('estado'),
      'obligaciones',jsonb_build_array('auditar'),
      'emitida_en',z_emitida,'valida_hasta',z_hasta
    );
    decision_canonica :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(decision);

    -- Cada campo V3 y cada campo del vinculo V2 es obligatorio y cerrado.
    FOR clave IN SELECT jsonb_object_keys(decision) LOOP
        alterada := decision - clave;
        IF vec_autorizacion.decision_contexto_actor_v3_valida(alterada) THEN
            RAISE EXCEPTION 'campo V3 ausente aceptado: %', clave;
        END IF;
    END LOOP;
    FOR clave IN SELECT jsonb_object_keys(vinculo) LOOP
        alterada := jsonb_set(decision,'{vinculo_autenticacion_actor}',
                              vinculo - clave,false);
        IF vec_autorizacion.decision_contexto_actor_v3_valida(alterada) THEN
            RAISE EXCEPTION 'campo V2 ausente aceptado: %', clave;
        END IF;
    END LOOP;
    IF vec_autorizacion.decision_contexto_actor_v3_valida(
           jsonb_set(decision,'{esquema}',
                     '"vec.autorizacion.decision.reforzada.v2.solicitud-ligada"')
       ) OR vec_autorizacion.vinculo_contexto_actor_v2_valido(
           jsonb_set(vinculo,'{bloque_version}','1')
       ) THEN
        RAISE EXCEPTION 'downgrade V1/V2 aceptado';
    END IF;
    alterada := jsonb_set(decision, '{accion}', '"consultar-á"');
    IF vec_autorizacion.decision_contexto_actor_v3_valida(alterada)
       OR vec_autorizacion.decision_contexto_actor_v3_valida(
              jsonb_set(decision, '{catalogo_politicas_huella_sha256}',
                        to_jsonb(repeat('f',64)))
          )
       OR vec_autorizacion.decision_contexto_actor_v3_valida(
              jsonb_set(decision, '{emitida_en}',
                        to_jsonb(regexp_replace(
                          z_emitida,'[.]([0-9]{6})Z$','Z'
                        )))
          )
       OR vec_autorizacion.decision_contexto_actor_v3_valida(
              jsonb_set(
                jsonb_set(decision,
                  '{vinculo_autenticacion_actor,garantia_observada}', '"bajo"'),
                '{garantia_minima}', '"alto"'
              )
          ) THEN
        RAISE EXCEPTION 'canon/semantica Go V3 divergente aceptada';
    END IF;
    alterada := jsonb_set(
        jsonb_set(decision,'{control_vigencia_version_rol_revision}',
                  '18446744073709551615'),
        '{revision_catalogo_politicas}','18446744073709551615'
    );
    alterada := jsonb_set(alterada,
        '{vinculo_autenticacion_actor,control_sesion_revision}',
        '18446744073709551615');
    alterada := jsonb_set(alterada,
        '{vinculo_autenticacion_actor,contexto_actor_version}',
        '18446744073709551615');
    alterada := jsonb_set(alterada,
        '{vinculo_autenticacion_actor,contexto_actor_cuenta_version}',
        '18446744073709551615');
    IF vec_autorizacion.decision_contexto_actor_v3_valida(alterada) IS NOT TRUE
       OR vec_autorizacion.decision_contexto_actor_v3_valida(
              jsonb_set(alterada,
                '{revision_catalogo_politicas}','18446744073709551616')
          ) THEN
        RAISE EXCEPTION 'frontera uint64 V3 incorrecta';
    END IF;

    -- jsonb::text reordena y espacia: era aceptado antes pese a no ser la
    -- salida de encoding/json congelada en Go.
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          convert_to(decision::text,'UTF8'),motivo,2,2
        )
    ) THEN
        RAISE EXCEPTION 'bytes JSON no canonicos aceptados';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          convert_to('{}','UTF8'),motivo,2,2
        )
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          decision_canonica,convert_to('{}','UTF8'),2,2
        )
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          decode('ff','hex'),motivo,2,2
        )
    ) THEN
        RAISE EXCEPTION 'entrada malformada produjo capacidad V3';
    END IF;

    SELECT * INTO STRICT obtenido
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
        decision_canonica,motivo,2,2
      );
    IF NOT obtenido.concedida OR obtenido.codigo <> 'concedida'
       OR obtenido.decision_huella_sha256 <>
          encode(sha256(decision_canonica),'hex')
       OR obtenido.registrada_en < emitida OR obtenido.registrada_en >= hasta
       OR (SELECT count(*)
             FROM vec_autorizacion.decision_concedida_contexto_actor_v3) <> 1
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3
       ) THEN
        RAISE EXCEPTION 'concesion positiva no quedo separada/durable';
    END IF;
    primera_huella := obtenido.decision_huella_sha256;
    primera_fecha := obtenido.registrada_en;

    SELECT * INTO STRICT obtenido
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
        decision_canonica,motivo,2,2
      );
    IF obtenido.decision_huella_sha256 <> primera_huella
       OR obtenido.registrada_en <> primera_fecha
       OR (SELECT count(*)
             FROM vec_autorizacion.decision_concedida_contexto_actor_v3) <> 1 THEN
        RAISE EXCEPTION 'replay exacto no fue idempotente';
    END IF;
    BEGIN
      PERFORM * FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
        decision_canonica,motivo,3,2
      );
      RAISE EXCEPTION 'replay con ResultadoContexto distinto aceptado';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    decision := jsonb_set(decision,'{decision_ref}',
                          '"decision:registro-v3:denegada"');
    decision := jsonb_set(decision,'{concedida}','false');
    decision := jsonb_set(decision,'{codigo}','"accion_no_concedida"');
    decision_canonica :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(decision);
    SELECT * INTO STRICT obtenido
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
        decision_canonica,motivo,2,2
      );
    IF obtenido.concedida OR obtenido.codigo <> 'accion_no_concedida'
       OR (SELECT count(*)
             FROM vec_autorizacion.decision_denegada_contexto_actor_v3) <> 1 THEN
        RAISE EXCEPTION 'denegacion probatoria no quedo separada';
    END IF;

    -- Fallo de segunda evidencia no deja fila parcial.
    decision := jsonb_set(decision,'{decision_ref}',
                          '"decision:registro-v3:adulterada"');
    decision := jsonb_set(decision,
        '{vinculo_autenticacion_actor,contexto_actor_cuenta_version}','3');
    decision_canonica :=
        vec_autorizacion.decision_contexto_actor_v3_canonica(decision);
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          decision_canonica,motivo,2,2
        )
    ) OR EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3
         WHERE decision_ref='decision:registro-v3:adulterada'
    ) THEN
        RAISE EXCEPTION 'adulteracion ContextoActor dejo registro parcial';
    END IF;
END
$prueba$;
COMMIT;

DO $acl$
DECLARE
    exterior regprocedure :=
      'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'::regprocedure;
    acreditacion regprocedure :=
      'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure;
    funcion record;
BEGIN
    IF NOT has_function_privilege('vec_autorizacion_registro',exterior,'EXECUTE')
       OR has_function_privilege('public',exterior,'EXECUTE')
       OR has_function_privilege('vec_autorizacion_fuente',exterior,'EXECUTE')
       OR has_table_privilege('vec_autorizacion_registro',
             'vec_autorizacion.decision_concedida_contexto_actor_v3','SELECT')
       OR has_table_privilege('vec_autorizacion_registro',
             'vec_autorizacion.decision_denegada_contexto_actor_v3','SELECT')
       OR has_function_privilege('vec_autorizacion_registro',
             'vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamptz,timestamptz,timestamptz)',
             'EXECUTE') THEN
        RAISE EXCEPTION 'ACL V3 efectiva no es minima';
    END IF;
    SELECT p.prosecdef, p.provolatile, p.proconfig,
           p.proowner = 'vec_autorizacion_propietario'::regrole AS propietario
      INTO STRICT funcion
      FROM pg_catalog.pg_proc AS p WHERE p.oid = exterior;
    IF NOT funcion.prosecdef OR funcion.provolatile <> 'v'
       OR NOT funcion.propietario
       OR funcion.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
       OR pg_catalog.pg_has_role(
            'vec_autorizacion_registro','vec_autorizacion_propietario','SET'
          )
       OR NOT pg_catalog.has_schema_privilege(
            'vec_autorizacion_propietario','vec_contexto_actor_v1','USAGE'
          )
       OR pg_catalog.has_schema_privilege(
            'vec_autorizacion_propietario','vec_contexto_actor_v1','CREATE'
          )
       OR NOT pg_catalog.has_function_privilege(
            'vec_autorizacion_propietario',acreditacion,'EXECUTE'
          ) THEN
        RAISE EXCEPTION 'definidor/search_path/capacidad cruzada V3 incorrectos';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_contexto_actor_v1' AND p.oid<>acreditacion
           AND pg_catalog.has_function_privilege(
                 'vec_autorizacion_propietario',p.oid,'EXECUTE'
               )
    ) OR EXISTS (
        SELECT 1
         FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
         WHERE n.nspname='vec_contexto_actor_v1'
           AND CASE WHEN c.relkind IN ('r','p','v','m') THEN
                 pg_catalog.has_any_column_privilege(
                   'vec_autorizacion_propietario',c.oid,
                   'SELECT,INSERT,UPDATE,REFERENCES'
                 ) OR pg_catalog.has_table_privilege(
                   'vec_autorizacion_propietario',c.oid,
                   'DELETE,TRUNCATE,TRIGGER,MAINTAIN'
                 )
               ELSE false END
    ) OR EXISTS (
        SELECT 1
         FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
         WHERE n.nspname='vec_contexto_actor_v1'
           AND CASE WHEN c.relkind='S' THEN
                 pg_catalog.has_sequence_privilege(
                   'vec_autorizacion_propietario',c.oid,
                   'USAGE,SELECT,UPDATE'
                 )
               ELSE false END
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid=t.typnamespace
         WHERE n.nspname IN ('vec_autorizacion','vec_contexto_actor_v1')
           AND t.typtype IN ('c','d','e','m','r')
           AND (pg_catalog.has_type_privilege('public',t.oid,'USAGE')
                OR pg_catalog.has_type_privilege(
                     'vec_autorizacion_registro',t.oid,'USAGE'
                   ))
    ) THEN
        RAISE EXCEPTION 'ACL efectiva heredada/PUBLIC/tipos amplifica V3';
    END IF;
END
$acl$;

DO $append_only$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
      'decision_concedida_contexto_actor_v3',
      'decision_denegada_contexto_actor_v3'
    ] LOOP
      BEGIN
        EXECUTE format('TRUNCATE vec_autorizacion.%I',tabla);
        RAISE EXCEPTION 'TRUNCATE aceptado: %',tabla;
      EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
      END;
    END LOOP;
END
$append_only$;
