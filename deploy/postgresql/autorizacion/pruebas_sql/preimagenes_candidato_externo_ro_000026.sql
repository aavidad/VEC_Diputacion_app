\set ON_ERROR_STOP on
-- Exclusivamente en clon sintético desechable H1→SQL62→AUT26, antes de AD132.
-- El invocador debe fijar vec.aut26_clon_desechable=on. No ejecutar en principal.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '30s';
DO $seguro$
BEGIN
    IF pg_catalog.current_setting('vec.aut26_clon_desechable', true) IS DISTINCT FROM 'on'
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname IN ('vec_externo_v3_fuente_autorizacion_desarrollo', 'aut26_ajeno_sintetico'))
       OR EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa)
       OR EXISTS (SELECT 1 FROM vec_autorizacion.version_rol
            WHERE rol_id = 'candidato_bolsa_historial_propio_desarrollo')
    THEN RAISE EXCEPTION 'fixture AUT26 requiere clon vacío de candidato y login nominal ausente'; END IF;
END $seguro$;
-- Estos LOGIN son fixtures; la migración AUT26 no crea ningún rol ni LOGIN.
CREATE ROLE vec_externo_v3_fuente_autorizacion_desarrollo LOGIN INHERIT NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_fuente_externa TO vec_externo_v3_fuente_autorizacion_desarrollo
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE aut26_ajeno_sintetico LOGIN INHERIT NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $acl$
DECLARE f oid; t text;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text)'::regprocedure::oid,
        'vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()'::regprocedure::oid
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid = f AND p.prosecdef
            AND p.provolatile = 's' AND p.proowner = 'vec_autorizacion_propietario'::regrole
            AND p.proconfig = ARRAY['search_path=pg_catalog']::text[])
           OR (SELECT count(*) FROM pg_catalog.aclexplode((SELECT proacl FROM pg_catalog.pg_proc WHERE oid = f))) <> 2
           OR pg_catalog.has_function_privilege('aut26_ajeno_sintetico', f, 'EXECUTE')
           OR pg_catalog.has_function_privilege('vec_autorizacion_publicador_candidato_externo', f, 'EXECUTE')
           OR NOT pg_catalog.has_function_privilege('vec_externo_v3_fuente_autorizacion_desarrollo', f, 'EXECUTE')
        THEN RAISE EXCEPTION 'ACL o atributos de función AUT26 divergentes'; END IF;
    END LOOP;
    FOREACH t IN ARRAY ARRAY['version_rol', 'control_vigencia_version_rol',
        'control_vigencia_version_rol_actual', 'asignacion_perfil_externa',
        'asignacion_perfil_actual_externa', 'motivo_v2_checkpoint_origen',
        'motivo_v2_evento_origen', 'motivo_v2_catalogo_publicado', 'motivo_v2_retirada', 'motivo_v2_entrada'] LOOP
        IF pg_catalog.has_table_privilege('vec_externo_v3_fuente_autorizacion_desarrollo',
            'vec_autorizacion.' || t, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
           OR pg_catalog.has_any_column_privilege('vec_externo_v3_fuente_autorizacion_desarrollo',
            'vec_autorizacion.' || t, 'SELECT,INSERT,UPDATE,REFERENCES')
        THEN RAISE EXCEPTION 'lector obtuvo acceso directo a tabla %', t; END IF;
    END LOOP;
END $acl$;
COMMIT;

SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
BEGIN READ ONLY;
DO $ausencia$
DECLARE d jsonb; m jsonb;
BEGIN
    d := vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
        'candidato_bolsa_historial_propio_desarrollo', 'rol:candidato_bolsa_historial_propio_desarrollo:v1',
        'principal:aut26:sintetico', 'perfil:aut26:ausente');
    IF d #>> '{rol,estado_observacion}' IS DISTINCT FROM 'ausente'
       OR d #>> '{control_rol,estado_observacion}' IS DISTINCT FROM 'ausente'
       OR d #>> '{asignacion,estado_observacion}' IS DISTINCT FROM 'ausente'
       OR (d #>> '{asignacion,versiones}')::bigint <> 0
    THEN RAISE EXCEPTION 'ausencia auténtica sin perfil provisionado no observada'; END IF;
    m := vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1();
    IF pg_catalog.jsonb_array_length(m -> 'catalogos') <> 3
       OR m #>> '{checkpoint,coherente}' IS DISTINCT FROM 'true'
    THEN RAISE EXCEPTION 'checkpoint inicial incoherente o catálogos incompletos'; END IF;
    BEGIN
        PERFORM vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
            'rrhh', 'rol:rrhh:v1', 'principal:aut26:sintetico', 'perfil:aut26:ausente');
        RAISE EXCEPTION 'rol ajeno admitido';
    EXCEPTION WHEN insufficient_privilege THEN NULL; END;
    BEGIN
        PERFORM vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
            NULL, NULL, NULL, NULL);
        RAISE EXCEPTION 'entradas nulas admitidas';
    EXCEPTION WHEN insufficient_privilege THEN NULL; END;
    BEGIN
        PERFORM 1 FROM vec_autorizacion.asignacion_perfil_externa;
        RAISE EXCEPTION 'SELECT directo admitido';
    EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ausencia$;
COMMIT;
BEGIN READ WRITE;
DO $ro$
BEGIN
    BEGIN
        PERFORM vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1();
        RAISE EXCEPTION 'transacción de escritura admitida';
    EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $ro$;
COMMIT;
RESET SESSION AUTHORIZATION;

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $fixture$
DECLARE d jsonb; c jsonb; perfil text; v bigint; n bigint; ref text; motivo_version integer;
        rol text := 'rol:candidato_bolsa_historial_propio_desarrollo:v1';
BEGIN
    d := '{"rol_id":"candidato_bolsa_historial_propio_desarrollo","version":1,
      "nombre":"Consulta propia de bolsa en desarrollo","estado":"publicada",
      "publicada_por":"seguridad:desarrollo:no-autoritativa","publicada_en":"2020-01-01T00:00:00.000000Z",
      "concesiones":[
        {"accion":"bolsa.historial_propio.consultar","modulo_id":"bolsa","tipo_recurso":"participaciones_candidato",
         "finalidades":["consulta_historial_propio"],"campos_permitidos":["contratos_propios","llamamientos_propios","renuncias_propias"],"garantia_minima":"alto"},
        {"accion":"bolsa.participaciones_propias.consultar","modulo_id":"bolsa","tipo_recurso":"participaciones_candidato",
         "finalidades":["consulta_participaciones_propias"],"campos_permitidos":["participaciones_candidato_minimizadas"],"garantia_minima":"alto"}]}'::jsonb;
    IF vec_autorizacion.rol_candidato_externo_acotado_v1(d) IS NOT TRUE THEN RAISE EXCEPTION 'rol fixture no acotado'; END IF;
    INSERT INTO vec_autorizacion.version_rol(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
        VALUES (rol, d ->> 'rol_id', 1, repeat('a',64), '2020-01-01Z', d);
    c := pg_catalog.jsonb_build_object('version_rol_ref',rol,'revision',1,'estado','habilitada',
        'actualizado_en','2020-01-01T00:00:00.000000Z');
    INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
        VALUES(rol,1,'habilitada',repeat('b',64),'2020-01-01Z',c);
    INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
        VALUES(rol,1,'2020-01-01Z','fixture:aut26','acto:aut26:control');
    FOREACH perfil IN ARRAY ARRAY['activa','revocada','expirada','colision','sin_puntero','puntero_antiguo'] LOOP
        FOR v IN 1..(CASE WHEN perfil = 'puntero_antiguo' THEN 2 ELSE 1 END) LOOP
            ref := 'asignacion:aut26_' || perfil || ':v' || v::text;
            d := pg_catalog.jsonb_build_object('asignacion_id','aut26_'||perfil,'version',v,
                'perfil_activo_ref','perfil:aut26:'||perfil,'principal_id',
                CASE WHEN perfil='colision' THEN 'principal:aut26:otro' ELSE 'principal:aut26:sintetico' END,
                'version_rol_ref',rol,'estado',CASE WHEN perfil='revocada' THEN 'revocada' ELSE 'activa' END,
                'ambitos','[{"clave":"candidato_ref","valores":["can_aut26_sintetico"]}]'::jsonb,
                'emitida_en','2020-01-01T00:00:00.000000Z','vigente_desde','2020-01-01T00:00:00.000000Z',
                'vigente_hasta',CASE WHEN perfil='expirada' THEN '2021-01-01T00:00:00.000000Z' ELSE '2099-01-01T00:00:00.000000Z' END);
            INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,
                principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
                VALUES(ref,'aut26_'||perfil,v,'perfil:aut26:'||perfil,d->>'principal_id',rol,repeat('c',64),'2020-01-01Z',d);
            IF perfil <> 'sin_puntero' AND v = 1 THEN
                INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
                    VALUES('perfil:aut26:'||perfil,ref,'2020-01-01Z','fixture:aut26','acto:aut26:'||perfil);
            END IF;
        END LOOP;
    END LOOP;
    -- Publicación y retirada reales avanzan juntos el checkpoint y la historia.
    SELECT ultima_secuencia INTO n FROM vec_autorizacion.motivo_v2_checkpoint_origen;
    SELECT coalesce(max(catalogo_version),0)+1 INTO motivo_version
      FROM vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id='motivos_mi_bolsa_desarrollo';
    IF vec_autorizacion.publicar_motivos_autorizacion_v2('evento_'||pg_catalog.lpad((n+1)::text,32,'0'),n+1,
        repeat('d',64),'motivos_mi_bolsa_desarrollo',motivo_version,repeat('e',64),pg_catalog.statement_timestamp(),
        '[{"clave":"motivo_00000000000000000000000000000999","vigente_desde":"2020-01-01T00:00:00.000000Z","vigente_hasta":null}]'::jsonb) IS NOT TRUE
    THEN RAISE EXCEPTION 'publicación fixture rechazada'; END IF;
    IF vec_autorizacion.retirar_motivos_autorizacion_v2('evento_'||pg_catalog.lpad((n+2)::text,32,'0'),n+2,
        repeat('f',64),'motivos_mi_bolsa_desarrollo',motivo_version,repeat('e',64),repeat('1',64),pg_catalog.statement_timestamp()) IS NOT TRUE
    THEN RAISE EXCEPTION 'retirada fixture rechazada'; END IF;
END $fixture$;
COMMIT;

SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
DO $estados$
DECLARE caso record; d jsonb; m jsonb; pub jsonb;
BEGIN
    FOR caso IN SELECT * FROM (VALUES ('activa','activa'), ('revocada','revocada'),
        ('expirada','no_vigente'), ('colision','inconsistente'), ('sin_puntero','inconsistente'),
        ('puntero_antiguo','inconsistente')) AS casos(perfil, estado) LOOP
        d := vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
            'candidato_bolsa_historial_propio_desarrollo','rol:candidato_bolsa_historial_propio_desarrollo:v1',
            'principal:aut26:sintetico','perfil:aut26:'||caso.perfil);
        IF d #>> '{rol,estado_observacion}' IS DISTINCT FROM 'publicada'
           OR d #>> '{control_rol,estado_observacion}' IS DISTINCT FROM 'habilitada'
           OR d #>> '{asignacion,estado_observacion}' IS DISTINCT FROM caso.estado
        THEN RAISE EXCEPTION 'estado incorrecto para %: %',caso.perfil,d; END IF;
        IF caso.perfil='colision' AND d #>> '{asignacion,asignacion_ref}' IS NOT NULL THEN
            RAISE EXCEPTION 'se reveló referencia de otro principal'; END IF;
    END LOOP;
    m := vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1();
    SELECT o INTO pub FROM pg_catalog.jsonb_array_elements(m->'catalogos') cat,
        LATERAL pg_catalog.jsonb_array_elements(cat->'observaciones') o
        WHERE cat->>'catalogo_id'='motivos_mi_bolsa_desarrollo' AND o->>'huella_publicada_sha256'=repeat('e',64);
    IF m #>> '{checkpoint,coherente}' IS DISTINCT FROM 'true'
       OR pub->>'retirada' IS DISTINCT FROM 'true' OR pub->>'entradas' IS DISTINCT FROM '1'
       OR pub->>'coherente' IS DISTINCT FROM 'true'
       OR pub->>'huella_retirada_sha256' IS DISTINCT FROM repeat('1',64)
       OR (pub->>'secuencia_retirada')::bigint <> (pub->>'secuencia_publicacion')::bigint+1
    THEN RAISE EXCEPTION 'motivos no conserva retirada/cardinalidad/secuencia: %',m; END IF;
END $estados$;
COMMIT;
RESET SESSION AUTHORIZATION;

-- El superusuario tampoco es el login nominal de la fachada.
BEGIN READ ONLY;
DO $nominal$
BEGIN
    BEGIN
        PERFORM vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1();
        RAISE EXCEPTION 'login superusuario no nominal admitido';
    EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $nominal$;
COMMIT;
SELECT 'AUT26_PRELIMINAR_OK' AS resultado;
