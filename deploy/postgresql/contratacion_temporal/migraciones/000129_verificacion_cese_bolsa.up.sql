\set ON_ERROR_STOP on
-- CT129: origen comprobable del cese CT115 para el consumidor Bolsa.
-- No crea un cese ni acredita por sí solo una baja en Personal. Devuelve el
-- hecho mínimo de CT115 únicamente si coincide con el evento CT113 publicado,
-- el outbox CT115 y la relación opaca confirmada por Personal en CT75.
-- Orden de instalación: Bolsa 000045 (rol, bandeja y receptor) antes de CT129.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000129',0));

DO $pre$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NOT NULL THEN
        RAISE EXCEPTION 'CT129: rol incompatible o migración ya instalada' USING ERRCODE='55000';
    END IF;
    IF to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.incorporacion_registro_v2') IS NULL
       OR to_regclass('vec_contratacion_temporal.outbox_expediente_integral') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.posicion_contrato_bolsa_v1(xid8)') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin)
       OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal','USAGE')
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_relevo_cese'
                       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
       OR NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
                       WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='restriccion_cese_bolsa' AND c.relkind='r')
       OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                       WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_restriccion_cese_bolsa_v1'
                         AND p.pronargs=3 AND p.proargtypes[0]='text'::regtype
                         AND p.proargtypes[1]='text'::regtype AND p.proargtypes[2]='bigint'::regtype) THEN
        RAISE EXCEPTION 'CT129: CT75/CT113/CT115 y Bolsa 000045 requeridos' USING ERRCODE='55000';
    END IF;
END
$pre$;

-- La fila de CT115 solo es visible mientras la comprueba el LOGIN nominal del
-- relevo de ceses. La opción es fijable por cualquier sesión: por ello la
-- política exige la membresía operativa y excluye a ambos migradores y
-- propietarios, incluso cuando el migrador CT asume SET ROLE propietario.
-- Bolsa no obtiene SELECT directo: la función vacía la opción antes de salir.
CREATE POLICY verificacion_cese_bolsa_ct129
    ON vec_contratacion_temporal.cese_nombramiento_v1 FOR SELECT
    TO vec_contratacion_temporal_propietario
    USING (evento_ref=current_setting('vec.ct129.origen_ref',true)
        AND pg_has_role(session_user,to_regrole('vec_bolsa_llamamientos_relevo_cese'),'MEMBER')
        AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
        AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper));

CREATE FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(
    p_origen_ref text, p_huella_sha256 text, p_posicion bigint
) RETURNS TABLE (
    modalidad_clave text,
    causa_contrato_clave text,
    causa_cese_clave text,
    fecha_efecto date,
    fuente_tipo text,
    fuente_ref text,
    fuente_sha256 text,
    relacion_ref text,
    version_resultante numeric,
    recibo_ref text,
    incorporacion_ref text,
    llamamiento_ref text,
    organizacion_ref text,
    expediente_ref text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
DECLARE v_marca_previa text:=current_setting('vec.ct129.origen_ref',true);
BEGIN
    -- Si el grupo desaparece después de instalar, falla cerrado; la política
    -- resuelve su OID con to_regrole para no romper otras lecturas CT115.
    IF to_regrole('vec_bolsa_llamamientos_relevo_cese') IS NULL THEN
        RETURN;
    END IF;
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR NOT pg_has_role(session_user,to_regrole('vec_bolsa_llamamientos_relevo_cese'),'MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
       OR p_origen_ref IS NULL OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512
       OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_posicion IS NULL OR p_posicion<0 THEN
        RETURN;
    END IF;
    PERFORM set_config('vec.ct129.origen_ref',p_origen_ref,true);
    RETURN QUERY
    WITH origen AS (
        SELECT c.*, i.material_json,
               i.organizacion_ref AS incorporacion_organizacion_ref,
               i.expediente_ref AS incorporacion_expediente_ref,
               o.payload_canonico, o.version_expediente AS outbox_version,
               o.expediente_ref AS outbox_expediente_ref
          FROM vec_contratacion_temporal.cese_nombramiento_v1 c
          JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref
          JOIN vec_contratacion_temporal.outbox_expediente_integral o ON o.evento_ref=c.evento_ref
              AND o.tipo_evento='ct.cese.v1'
         WHERE c.evento_ref=p_origen_ref AND c.estado='confirmada'
           AND c.llamamiento_ref IS NOT NULL
           AND c.transaccion_publicacion IS NOT NULL
           AND vec_contratacion_temporal.posicion_contrato_bolsa_v1(c.transaccion_publicacion)=p_posicion
           AND (c.transaccion_publicacion < pg_snapshot_xmin(pg_current_snapshot())
                OR c.transaccion_publicacion=pg_current_xact_id_if_assigned())
    ), cuerpo AS (
        SELECT x.*,
               'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('cese'||chr(31)||x.evento_ref,'UTF8')),'hex') AS publicacion_ref,
               jsonb_build_object(
                   'esquema','vec.contratacion-temporal.contrato-bolsa.v1',
                   'tipo','cese',
                   'origen_ref',x.evento_ref,
                   'organizacion_ref',x.organizacion_ref,
                   'expediente_ref',x.expediente_ref,
                   'llamamiento_ref',x.llamamiento_ref,
                   'inicio',vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (x.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
                   'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(x.fecha_efecto::timestamp AT TIME ZONE 'UTC'),
                   'modalidad_clave',x.expediente_siguiente_json #>> '{analisis,modalidad_clave}',
                   'categoria_ref',x.expediente_siguiente_json #>> '{analisis,categoria_ref}',
                   'causa_clave',x.causa_clave,
                   'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(x.registrada_en)
               ) AS publicacion_cuerpo
          FROM origen x
    )
    SELECT x.expediente_siguiente_json #>> '{analisis,modalidad_clave}',
           x.expediente_siguiente_json #>> '{analisis,causa_clave}',x.causa_clave,x.fecha_efecto,
           x.justificante_tipo,x.justificante_ref,x.justificante_sha256,
           x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}',
           x.version_esperada+1,x.recibo_ref,x.incorporacion_ref,x.llamamiento_ref,
           x.organizacion_ref,x.expediente_ref
      FROM cuerpo x
     WHERE x.incorporacion_organizacion_ref=x.organizacion_ref
       AND x.incorporacion_expediente_ref=x.expediente_ref
       AND x.outbox_version=x.version_esperada+1
       AND x.outbox_expediente_ref=x.expediente_ref
       AND x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}' ~ '^ref:[0-9a-f]{64}$'
       AND x.expediente_siguiente_json #>> '{analisis,modalidad_clave}' ~ '^[a-z][a-z0-9_.-]{1,79}$'
       AND x.expediente_siguiente_json #>> '{analisis,causa_clave}' ~ '^[a-z][a-z0-9_.-]{1,79}$'
       AND convert_from(x.payload_canonico,'UTF8')::jsonb = jsonb_build_object(
           'esquema','vec.contratacion-temporal.cese.v1',
           'organizacion_ref',x.organizacion_ref,
           'expediente_ref',x.expediente_ref,
           'version_resultante',x.version_esperada+1,
           'incorporacion_ref',x.incorporacion_ref,
           'llamamiento_ref',x.llamamiento_ref,
           'causa_clave',x.causa_clave,
           'fecha_efecto',to_char(x.fecha_efecto,'YYYY-MM-DD'),
           'recibo_ref',x.recibo_ref,
           'registrada_en',x.recibo_json->'registrada_en')
       AND encode(sha256(convert_to((x.publicacion_cuerpo||jsonb_build_object('evento_ref',x.publicacion_ref))::text,'UTF8')),'hex')
           =p_huella_sha256;
    PERFORM set_config('vec.ct129.origen_ref',coalesce(v_marca_previa,''),true);
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)
    TO vec_bolsa_llamamientos_propietario;
DO $acl$
BEGIN
    IF has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.cese_nombramiento_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR has_table_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.incorporacion_registro_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR has_function_privilege('public','vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)','EXECUTE')
       OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)','EXECUTE')
       OR (SELECT NOT prosecdef OR proowner<>'vec_contratacion_temporal_propietario'::regrole
             FROM pg_proc WHERE oid='vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)'::regprocedure) THEN
        RAISE EXCEPTION 'CT129: ACL de verificación incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMENT ON FUNCTION vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint) IS
    'CT129: verifica origen CT115 y publicación CT113 exactos; devuelve modalidad, causa del contrato y del cese, fecha, justificante y relación opaca CT75, solo a Bolsa.';
COMMIT;
