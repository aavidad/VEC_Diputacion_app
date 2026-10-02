\set ON_ERROR_STOP on
-- Comprobaciones de estructura y denegación de BR4. No sustituyen el ensayo
-- positivo con una concesión nominal nueva emitida por la autoridad V3 real.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL statement_timeout = '30s';
SET LOCAL lock_timeout = '2s';

DO $estructura$
DECLARE
    funcion oid := to_regprocedure(
        'vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    objeto oid;
    nombre text;
    rol text;
BEGIN
    IF funcion IS NULL OR NOT EXISTS (
        SELECT 1 FROM pg_proc
         WHERE oid = funcion AND prosecdef AND provolatile = 'v'
           AND proowner = 'vec_bolsa_reglas_baremo_propietario'::regrole
           AND proconfig @> ARRAY['search_path=pg_catalog', 'row_security=on']
    ) THEN
        RAISE EXCEPTION 'BR4: fachada ausente o autoridad incompatible';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_roles
         WHERE rolname = 'vec_bolsa_reglas_baremo_propietario'
           AND (rolcanlogin OR rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb)
    ) THEN
        RAISE EXCEPTION 'BR4: propietario con privilegios incompatibles';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'acceso_borrador_v3', 'outbox_borrador_v3', 'recibo_borrador_v3'
    ] LOOP
        objeto := to_regclass('vec_bolsa_reglas_baremo.' || nombre);
        IF objeto IS NULL OR NOT EXISTS (
            SELECT 1 FROM pg_class
             WHERE oid = objeto AND relrowsecurity AND relforcerowsecurity
               AND relowner = 'vec_bolsa_reglas_baremo_propietario'::regrole
        ) OR (SELECT count(*) FROM pg_policy WHERE polrelid = objeto) <> 1
          OR NOT EXISTS (
              SELECT 1 FROM pg_policy
               WHERE polrelid = objeto
                 AND polroles = ARRAY['vec_bolsa_reglas_baremo_propietario'::regrole::oid]
          ) OR (SELECT count(*) FROM pg_trigger
                 WHERE tgrelid = objeto AND NOT tgisinternal
                   AND tgname IN ('inmutable', 'no_truncar')) <> 2 THEN
            RAISE EXCEPTION 'BR4: RLS, política o inmutabilidad incompatibles';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_type t
            CROSS JOIN LATERAL aclexplode(coalesce(t.typacl, acldefault('T', t.typowner))) a
             WHERE t.typrelid = objeto AND a.grantee = 0
        ) THEN
            RAISE EXCEPTION 'BR4: tipo de fila accesible para PUBLIC';
        END IF;
        IF EXISTS (SELECT 1 FROM pg_class c,
            LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
            WHERE c.oid=objeto AND a.grantee=0) THEN
            RAISE EXCEPTION 'BR4: tabla accesible para PUBLIC';
        END IF;
        FOREACH rol IN ARRAY ARRAY[
            'vec_bolsa_reglas_baremo_ejecutor_gobierno',
            'vec_bolsa_reglas_baremo_ejecutor_consulta',
            'vec_bolsa_reglas_baremo_publicador_outbox'
        ] LOOP
            IF has_table_privilege(rol, objeto, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
               OR (rol<>'vec_bolsa_reglas_baremo_ejecutor_gobierno'
                   AND has_function_privilege(rol, funcion, 'EXECUTE')) THEN
                RAISE EXCEPTION 'BR4: acceso no nominal concedido a runtime';
            END IF;
        END LOOP;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM pg_proc p
        CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
         WHERE p.pronamespace = 'vec_bolsa_reglas_baremo'::regnamespace
           AND a.privilege_type = 'EXECUTE' AND a.grantee = 0
    ) THEN
        RAISE EXCEPTION 'BR4: función de Gobierno accesible para PUBLIC';
    END IF;
    IF NOT has_function_privilege('vec_bolsa_reglas_baremo_ejecutor_gobierno',funcion,'EXECUTE')
       OR NOT has_schema_privilege('vec_bolsa_reglas_baremo_ejecutor_gobierno','vec_bolsa_reglas_baremo','USAGE')
       OR has_schema_privilege('vec_bolsa_reglas_baremo_ejecutor_consulta','vec_bolsa_reglas_baremo','USAGE')
       OR has_schema_privilege('vec_bolsa_reglas_baremo_publicador_outbox','vec_bolsa_reglas_baremo','USAGE')
       OR (SELECT count(*) FROM pg_proc p,
           LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=funcion)<>2
       OR EXISTS(SELECT 1 FROM pg_proc p,
           LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
           WHERE p.oid=funcion AND (a.grantee NOT IN(p.proowner,'vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole)
             OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
       OR EXISTS(SELECT 1 FROM pg_proc p,
           LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
           WHERE p.pronamespace='vec_bolsa_reglas_baremo'::regnamespace
             AND p.oid<>funcion AND a.grantee<>p.proowner)
    THEN RAISE EXCEPTION 'BR4: runtime fuera del contrato nominal'; END IF;
END
$estructura$;

-- El consumo central admite 1..15000 ms. La configuración propia de la
-- fachada no puede elevar los 15s que fija el adaptador al abrir la TX.
DO $limite_consumo$
DECLARE f oid:='vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        config text[];
BEGIN
 SELECT proconfig INTO STRICT config FROM pg_proc WHERE oid=f;
 IF config IS NULL OR NOT config @> ARRAY['statement_timeout=15s','lock_timeout=2s']
 OR (SELECT count(*) FROM unnest(config) c(valor) WHERE c.valor LIKE 'statement_timeout=%')<>1
 THEN RAISE EXCEPTION 'BR4: límite de fachada incompatible con el consumo V3'; END IF;
END $limite_consumo$;

SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
DO $entrada_invalida$
BEGIN
    BEGIN
        PERFORM vec_bolsa_reglas_baremo.validar_material_borrador_v3(NULL, NULL);
        RAISE EXCEPTION 'BR4: entrada nula aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_bolsa_reglas_baremo.validar_material_borrador_v3(
            convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8')
        );
        RAISE EXCEPTION 'BR4: proyecciones vacías aceptadas';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
END
$entrada_invalida$;
DO $acciones_lectura$
DECLARE m jsonb; motivo bytea:=convert_to('{}','UTF8'); operacion text;
 accion text; rechazados integer:=0;
BEGIN
 FOREACH operacion IN ARRAY ARRAY['consultar_exacta','recuperar_recibo'] LOOP
  accion:=CASE WHEN operacion='consultar_exacta' THEN 'bolsa.reglas_baremo.version.consultar'
   ELSE 'bolsa.reglas_baremo.recibo.consultar' END;
  m:=jsonb_build_object('esquema','vec.bolsa.gobierno-borrador.material.v3',
   'operacion',operacion,'accion',accion,'modulo_id','bolsa',
   'tipo_recurso','version_reglas_baremo_gobernada','finalidad','consulta_gobierno_reglas_baremo',
   'persona_ref','per_'||repeat('a',22),'perfil_ref','perfil:sintetico:rrhh',
   'convocatoria_ref','convocatoria:sintetica','expediente_ref','expediente:sintetico',
   'estado',jsonb_build_object('referencia','reglas:sinteticas','version',1000000000,
    'huella_contenido_sha256',repeat('a',64),'revision',1,'huella_estado_sha256',repeat('b',64)),
   'estado_esperado',NULL,'version_canonica',NULL,
   'clave_operacion',CASE WHEN operacion='recuperar_recibo' THEN repeat('c',32) ELSE '' END,
   'huella_solicitud_sha256',CASE WHEN operacion='recuperar_recibo' THEN repeat('d',64) ELSE '' END,
   'motivo_canonico',encode(motivo,'base64'),'solicitada_en',clock_timestamp());
  -- Sólo el validador material, sin simular autorización: prueba la frontera
  -- común con Go y el límite inclusivo 1e9 de la versión de contenido.
  IF vec_bolsa_reglas_baremo.validar_material_borrador_v3(convert_to(m::text,'UTF8'),motivo)
     IS DISTINCT FROM m THEN RAISE EXCEPTION 'BR4: lectura nominal rechazada'; END IF;
  m:=jsonb_set(m,'{accion}',to_jsonb(CASE WHEN operacion='consultar_exacta'
    THEN 'bolsa.reglas_baremo.recibo.consultar' ELSE 'bolsa.reglas_baremo.version.consultar' END));
  BEGIN
   PERFORM vec_bolsa_reglas_baremo.validar_material_borrador_v3(convert_to(m::text,'UTF8'),motivo);
   RAISE EXCEPTION 'BR4: acción cruzada aceptada';
  EXCEPTION WHEN invalid_parameter_value THEN rechazados:=rechazados+1; END;
 END LOOP;
 IF rechazados<>2 THEN RAISE EXCEPTION 'BR4: acciones cruzadas incompletas'; END IF;
END $acciones_lectura$;
ROLLBACK;
