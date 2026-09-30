-- Lecturas internas de categorias RPT. Requiere catalogos_configurables 000001.
-- Solo AD3-117 podra exponerlas tras autorizar y auditar cada consulta.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000002', 0));
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text)') IS NULL
       OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.listar_habilitadas(text,text,integer)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.consultar_uso(text,text,text)') IS NOT NULL THEN
        RAISE EXCEPTION 'catalogos 000002: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $pre$;

-- La lista refleja controles actuales. Una publicacion nueva solo cambia las
-- categorias incluidas, de modo que una pagina puede mezclar versiones exactas.
-- El presupuesto cubre los documentos canonicos deduplicados y las entradas.
CREATE FUNCTION vec_catalogos_configurables.listar_habilitadas(
    p_catalogo_id text, p_cursor_categoria_id text, p_limite integer
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '5s' AS $f$
DECLARE
    r record;
    items jsonb := '[]'::jsonb;
    publicaciones jsonb := '[]'::jsonb;
    vistos jsonb := '{}'::jsonb;
    item jsonb;
    publicacion jsonb;
    clave_publicacion text;
    cursor_siguiente text;
    total integer := 0;
    bytes_usados bigint := 0;
    hay_mas boolean := false;
    -- La fachada AD3 agrega un recibo: se reserva 4 KiB dentro de 48 MiB.
    presupuesto constant bigint := 50327552;
    resultado jsonb;
    documento_base text;
    version_base integer;
    huella_base text;
    anclaje jsonb;
BEGIN
    IF p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR (p_cursor_categoria_id IS NOT NULL AND p_cursor_categoria_id !~ '^[a-z][a-z0-9_.:-]{2,127}$')
       OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
        RAISE EXCEPTION 'lista de categorias invalida' USING ERRCODE = '22023';
    END IF;
    SELECT version,huella_sha256,documento_canonico
      INTO version_base,huella_base,documento_base FROM vec_catalogos_configurables.publicacion
     WHERE catalogo_id = p_catalogo_id ORDER BY version LIMIT 1;
    IF NOT FOUND THEN
        RETURN pg_catalog.jsonb_build_object('encontrado', false, 'datos', NULL);
    END IF;
    anclaje := pg_catalog.jsonb_build_object('catalogo_id',p_catalogo_id,
        'version',version_base,'huella_sha256',huella_base,'documento_canonico',documento_base);
    bytes_usados := pg_catalog.octet_length(anclaje::text)+256;
    IF bytes_usados>presupuesto THEN
        RAISE EXCEPTION 'anclaje publicado supera presupuesto de lectura' USING ERRCODE='54000';
    END IF;
    vistos := pg_catalog.jsonb_build_object(p_catalogo_id||':'||version_base::text||':'||huella_base,true);
    FOR r IN
        SELECT c.categoria_id,c.catalogo_id,c.version,c.huella_sha256,c.revision,c.estado,
               e.etiqueta,e.definicion,p.documento_canonico
          FROM vec_catalogos_configurables.categoria_control AS c
          JOIN vec_catalogos_configurables.entrada_publicada AS e
            ON e.catalogo_id=c.catalogo_id AND e.version=c.version AND e.categoria_id=c.categoria_id
           AND e.huella_sha256=c.huella_sha256
          JOIN vec_catalogos_configurables.publicacion AS p
            ON p.catalogo_id=e.catalogo_id AND p.version=e.version AND p.huella_sha256=e.huella_sha256
         WHERE c.catalogo_id=p_catalogo_id AND c.estado='habilitada'
           AND (p_cursor_categoria_id IS NULL OR c.categoria_id>p_cursor_categoria_id)
         ORDER BY c.categoria_id
         LIMIT p_limite+1
    LOOP
        IF total = p_limite THEN hay_mas := true; EXIT; END IF;
        item := pg_catalog.jsonb_build_object(
            'categoria_id',r.categoria_id,'catalogo_id',r.catalogo_id,
            'version',r.version,'huella_sha256',r.huella_sha256,
            'revision',r.revision,'estado',r.estado,
            'etiqueta',r.etiqueta,'definicion',r.definicion);
        clave_publicacion := r.catalogo_id || ':' || r.version::text || ':' || r.huella_sha256;
        publicacion := NULL;
        IF NOT (vistos ? clave_publicacion) THEN
            publicacion := pg_catalog.jsonb_build_object(
                'catalogo_id',r.catalogo_id,'version',r.version,
                'huella_sha256',r.huella_sha256,'documento_canonico',r.documento_canonico);
        END IF;
        IF bytes_usados + pg_catalog.octet_length(item::text)
           + COALESCE(pg_catalog.octet_length(publicacion::text),0) + 256 > presupuesto THEN
            IF total = 0 THEN
                RAISE EXCEPTION 'categoria publicada supera presupuesto de lectura' USING ERRCODE = '54000';
            END IF;
            hay_mas := true;
            EXIT;
        END IF;
        bytes_usados := bytes_usados + pg_catalog.octet_length(item::text)
            + COALESCE(pg_catalog.octet_length(publicacion::text),0) + 256;
        items := items || pg_catalog.jsonb_build_array(item);
        IF publicacion IS NOT NULL THEN
            publicaciones := publicaciones || pg_catalog.jsonb_build_array(publicacion);
            vistos := vistos || pg_catalog.jsonb_build_object(clave_publicacion,true);
        END IF;
        cursor_siguiente := r.categoria_id;
        total := total + 1;
    END LOOP;
    resultado := pg_catalog.jsonb_build_object('encontrado',true,'datos',pg_catalog.jsonb_build_object(
        'anclaje_publicacion',anclaje,'items',items,'publicaciones',publicaciones,'hay_mas',hay_mas,
        'siguiente_cursor',CASE WHEN hay_mas THEN cursor_siguiente ELSE NULL END));
    IF pg_catalog.octet_length(resultado::text)>presupuesto THEN
        RAISE EXCEPTION 'pagina de categorias supera presupuesto de lectura' USING ERRCODE = '54000';
    END IF;
    RETURN resultado;
END $f$;

-- Publicacion y entrada historicas son inmutables. El control actual se
-- devuelve aparte: puede apuntar a otra version o estar deshabilitado.
CREATE FUNCTION vec_catalogos_configurables.leer_publicacion_categoria(
    p_catalogo_id text, p_version integer, p_huella text, p_categoria_id text
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '5s' AS $f$
DECLARE
    r record;
BEGIN
    IF p_catalogo_id IS NULL OR p_catalogo_id !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_version IS NULL OR p_version < 1
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_categoria_id IS NULL OR p_categoria_id !~ '^[a-z][a-z0-9_.:-]{2,127}$' THEN
        RAISE EXCEPTION 'consulta historica de categoria invalida' USING ERRCODE = '22023';
    END IF;
    SELECT p.catalogo_id,p.version,p.huella_sha256,p.documento_canonico,p.publicada_en,
           e.definicion,c.catalogo_id AS control_catalogo_id,c.version AS control_version,
           c.huella_sha256 AS control_huella,c.revision AS control_revision,c.estado AS control_estado
      INTO r
      FROM vec_catalogos_configurables.publicacion AS p
      JOIN vec_catalogos_configurables.entrada_publicada AS e
        ON e.catalogo_id=p.catalogo_id AND e.version=p.version AND e.huella_sha256=p.huella_sha256
      LEFT JOIN vec_catalogos_configurables.categoria_control AS c ON c.categoria_id=e.categoria_id
     WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version AND p.huella_sha256=p_huella
       AND e.categoria_id=p_categoria_id;
    IF NOT FOUND THEN
        RETURN pg_catalog.jsonb_build_object('encontrado',false,'datos',NULL);
    END IF;
    RETURN pg_catalog.jsonb_build_object('encontrado',true,'datos',pg_catalog.jsonb_build_object(
        'publicacion',pg_catalog.jsonb_build_object(
            'catalogo_id',r.catalogo_id,'version',r.version,'huella_sha256',r.huella_sha256,
            'documento_canonico',r.documento_canonico,
            'publicada_en',pg_catalog.to_char(r.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
        'entrada',r.definicion,
        'control_actual',CASE WHEN r.control_catalogo_id IS NULL THEN NULL ELSE pg_catalog.jsonb_build_object(
            'catalogo_id',r.control_catalogo_id,'version',r.control_version,
            'huella_sha256',r.control_huella,'revision',r.control_revision,'estado',r.control_estado) END));
END $f$;

-- La referencia de reserva se comprueba junto al consumidor y a su uso. El
-- recibo por si solo no identifica al actor ni concede permiso.
CREATE FUNCTION vec_catalogos_configurables.consultar_uso(
    p_consumidor text, p_uso_ref text, p_reserva_recibo_ref text
) RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '5s' AS $f$
DECLARE
    u vec_catalogos_configurables.uso%ROWTYPE;
BEGIN
    IF p_consumidor IS NULL OR p_consumidor !~ '^[a-z][a-z0-9_.:-]{2,127}$'
       OR p_uso_ref IS NULL OR pg_catalog.octet_length(p_uso_ref) NOT BETWEEN 3 AND 160
       OR p_reserva_recibo_ref IS NULL OR pg_catalog.octet_length(p_reserva_recibo_ref) NOT BETWEEN 3 AND 160 THEN
        RAISE EXCEPTION 'consulta de uso invalida' USING ERRCODE = '22023';
    END IF;
    SELECT * INTO u FROM vec_catalogos_configurables.uso
     WHERE consumidor=p_consumidor AND uso_ref=p_uso_ref AND reserva_recibo_ref=p_reserva_recibo_ref;
    IF NOT FOUND THEN
        RETURN pg_catalog.jsonb_build_object('encontrado',false,'datos',NULL);
    END IF;
    RETURN pg_catalog.jsonb_build_object('encontrado',true,'datos',pg_catalog.jsonb_build_object(
        'consumidor',u.consumidor,'uso_ref',u.uso_ref,'categoria_id',u.categoria_id,
        'catalogo_id',u.catalogo_id,'version',u.version,'huella_sha256',u.huella_sha256,
        'estado',u.estado,'revision',u.revision,'reserva_recibo_ref',u.reserva_recibo_ref,
        'terminal_recibo_ref',u.terminal_recibo_ref,
        'reservado_en',pg_catalog.to_char(u.reservado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'terminal_en',CASE WHEN u.terminal_en IS NULL THEN NULL ELSE
            pg_catalog.to_char(u.terminal_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END));
END $f$;

REVOKE ALL ON FUNCTION vec_catalogos_configurables.listar_habilitadas(text,text,integer),
    vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text),
    vec_catalogos_configurables.consultar_uso(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.listar_habilitadas(text,text,integer),
    vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text),
    vec_catalogos_configurables.consultar_uso(text,text,text) TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
