-- Proyeccion fisicamente separada y minimizada de Bolsa. No contiene persona,
-- candidatura, DNI, contacto, expediente interno, actor ni trazas de RRHH.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000001', 0)
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
);

DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_bolsa_publica_migrador'
       OR NOT pg_catalog.pg_has_role(
           current_user, 'vec_bolsa_publica_propietario', 'SET'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'migracion de bolsa publica rechazada: identidad incorrecta';
    END IF;
    IF NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_datos'
              AND pg_catalog.pg_get_userbyid(nspowner) = 'vec_bolsa_publica_propietario'
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_lectura'
              AND pg_catalog.pg_get_userbyid(nspowner) = 'vec_bolsa_publica_propietario'
       )
       OR NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_bolsa_publica_publicacion'
              AND pg_catalog.pg_get_userbyid(nspowner) =
                  'vec_bolsa_publica_publicacion_propietario'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class AS objeto
             JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.relnamespace
            WHERE esquema.nspname IN (
                'vec_bolsa_publica_datos', 'vec_bolsa_publica_lectura',
                'vec_bolsa_publica_publicacion'
            )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS objeto
             JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.pronamespace
            WHERE esquema.nspname IN (
                'vec_bolsa_publica_datos', 'vec_bolsa_publica_lectura',
                'vec_bolsa_publica_publicacion'
            )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type AS objeto
             JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.typnamespace
            WHERE esquema.nspname IN (
                'vec_bolsa_publica_datos', 'vec_bolsa_publica_lectura',
                'vec_bolsa_publica_publicacion'
            )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion de bolsa publica: esquemas ausentes, ajenos o no vacios';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_bolsa_publica_propietario;

CREATE TABLE vec_bolsa_publica_datos.fuente (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    revision text NOT NULL CHECK (
        revision ~ '^[a-z0-9][a-z0-9._-]{0,79}$'
    ),
    actualizada_en timestamptz(6) NOT NULL,
    manifiesto_sha256 text NOT NULL CHECK (manifiesto_sha256 ~ '^[a-f0-9]{64}$')
);

-- Registro monotono de anclas ya consumidas. No forma parte de la superficie
-- publica y el propietario de la funcion solo puede insertar: ni una
-- invalidacion ni una publicacion posterior permiten volver de B a A.
CREATE TABLE vec_bolsa_publica_datos.manifiesto_consumido (
    manifiesto_sha256 text PRIMARY KEY CHECK (
        manifiesto_sha256 ~ '^[a-f0-9]{64}$'
        AND manifiesto_sha256 <> pg_catalog.repeat('0', 64)
    ),
    registrado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.statement_timestamp()
);

CREATE TABLE vec_bolsa_publica_datos.catalogo_publico (
    referencia text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    activo boolean NOT NULL DEFAULT false,
    PRIMARY KEY (referencia, version),
    CHECK (referencia ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$')
);
CREATE UNIQUE INDEX catalogo_publico_un_activo
    ON vec_bolsa_publica_datos.catalogo_publico(referencia)
    WHERE activo;

CREATE TABLE vec_bolsa_publica_datos.entrada_catalogo_publico (
    referencia text NOT NULL,
    version integer NOT NULL,
    clave text NOT NULL,
    etiqueta text NOT NULL CHECK (char_length(etiqueta) BETWEEN 1 AND 120),
    descripcion text NOT NULL DEFAULT '' CHECK (char_length(descripcion) <= 600),
    semantica text NOT NULL CHECK (
        semantica ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'
    ),
    orden integer NOT NULL CHECK (orden > 0),
    publicable boolean NOT NULL DEFAULT true,
    PRIMARY KEY (referencia, version, clave),
    UNIQUE (referencia, version, orden),
    FOREIGN KEY (referencia, version)
        REFERENCES vec_bolsa_publica_datos.catalogo_publico(referencia, version)
        ON DELETE CASCADE,
    CHECK (clave ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$')
);

CREATE TABLE vec_bolsa_publica_datos.catalogo_categorias (
    catalogo_id text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    huella_gobernada_sha256 text NOT NULL CHECK (
        huella_gobernada_sha256 ~ '^[a-f0-9]{64}$'
    ),
    huella_proyeccion_publica_sha256 text NOT NULL CHECK (
        huella_proyeccion_publica_sha256 ~ '^[a-f0-9]{64}$'
    ),
    actual boolean NOT NULL DEFAULT false,
    PRIMARY KEY (catalogo_id, version),
    UNIQUE (
        catalogo_id, version, huella_gobernada_sha256,
        huella_proyeccion_publica_sha256
    ),
    CHECK (catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$')
);
CREATE UNIQUE INDEX catalogo_categorias_un_actual
    ON vec_bolsa_publica_datos.catalogo_categorias((actual))
    WHERE actual;

CREATE TABLE vec_bolsa_publica_datos.categoria_publica (
    catalogo_id text NOT NULL,
    version integer NOT NULL,
    clave text NOT NULL,
    etiqueta text NOT NULL CHECK (char_length(etiqueta) BETWEEN 1 AND 120),
    descripcion text NOT NULL DEFAULT '' CHECK (char_length(descripcion) <= 600),
    semantica text NOT NULL CHECK (semantica ~ '^[a-z][a-z0-9_]{0,79}$'),
    orden integer NOT NULL CHECK (orden > 0),
    area text NOT NULL CHECK (area ~ '^[a-z][a-z0-9_]{0,79}$'),
    area_etiqueta text NOT NULL CHECK (char_length(area_etiqueta) BETWEEN 1 AND 120),
    suscribible boolean NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    publicable boolean NOT NULL DEFAULT true,
    PRIMARY KEY (catalogo_id, version, clave),
    UNIQUE (catalogo_id, version, orden),
    FOREIGN KEY (catalogo_id, version)
        REFERENCES vec_bolsa_publica_datos.catalogo_categorias(catalogo_id, version)
        ON DELETE CASCADE,
    CHECK (clave ~ '^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$'),
    CHECK (vigente_hasta IS NULL OR vigente_desde < vigente_hasta)
);

CREATE TABLE vec_bolsa_publica_datos.convocatoria_publica (
    identificador_publico text PRIMARY KEY CHECK (
        identificador_publico ~ '^[a-z0-9][a-z0-9-]{2,79}$'
    ),
    version_publica text NOT NULL CHECK (
        version_publica ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$'
    ),
    estado text NOT NULL CHECK (estado ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    tipo text NOT NULL CHECK (tipo ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    catalogo_categorias_id text NOT NULL,
    catalogo_categorias_version integer NOT NULL,
    catalogo_categorias_huella_sha256 text NOT NULL CHECK (
        catalogo_categorias_huella_sha256 ~ '^[a-f0-9]{64}$'
    ),
	catalogo_categorias_huella_proyeccion_sha256 text NOT NULL CHECK (
		catalogo_categorias_huella_proyeccion_sha256 ~ '^[a-f0-9]{64}$'
	),
	huella_publica_sha256 text NOT NULL CHECK (
		huella_publica_sha256 ~ '^[a-f0-9]{64}$'
	),
    huella_resumen_publico_sha256 text NOT NULL CHECK (
        huella_resumen_publico_sha256 ~ '^[a-f0-9]{64}$'
    ),
    titulo text NOT NULL CHECK (char_length(titulo) BETWEEN 1 AND 180),
    resumen text NOT NULL CHECK (char_length(resumen) BETWEEN 1 AND 500),
    descripcion text NOT NULL CHECK (char_length(descripcion) BETWEEN 1 AND 12000),
    publicada_en timestamptz(6) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
	busqueda tsvector GENERATED ALWAYS AS (
		setweight(to_tsvector('spanish'::regconfig, titulo), 'A') ||
		setweight(to_tsvector('spanish'::regconfig, resumen), 'B') ||
		setweight(to_tsvector('spanish'::regconfig, descripcion), 'C')
	) STORED,
    publicable boolean NOT NULL DEFAULT false,
    revocada_en timestamptz(6),
    UNIQUE (identificador_publico, catalogo_categorias_id, catalogo_categorias_version),
    FOREIGN KEY (
        catalogo_categorias_id, catalogo_categorias_version,
        catalogo_categorias_huella_sha256,
        catalogo_categorias_huella_proyeccion_sha256
    ) REFERENCES vec_bolsa_publica_datos.catalogo_categorias(
        catalogo_id, version, huella_gobernada_sha256,
        huella_proyeccion_publica_sha256
    ),
    CHECK (publicada_en <= actualizada_en),
    CHECK ((publicable AND revocada_en IS NULL) OR (NOT publicable)),
    CHECK (revocada_en IS NULL OR actualizada_en <= revocada_en)
);

CREATE INDEX convocatoria_publica_busqueda_idx
    ON vec_bolsa_publica_datos.convocatoria_publica USING gin (busqueda)
    WHERE publicable AND revocada_en IS NULL;

CREATE TABLE vec_bolsa_publica_datos.convocatoria_categoria (
    identificador_publico text NOT NULL,
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    categoria_clave text NOT NULL,
    PRIMARY KEY (identificador_publico, categoria_clave),
    FOREIGN KEY (identificador_publico, catalogo_id, catalogo_version)
        REFERENCES vec_bolsa_publica_datos.convocatoria_publica(
            identificador_publico, catalogo_categorias_id, catalogo_categorias_version
        )
        ON DELETE CASCADE,
    FOREIGN KEY (catalogo_id, catalogo_version, categoria_clave)
        REFERENCES vec_bolsa_publica_datos.categoria_publica(
            catalogo_id, version, clave
        ),
    CHECK (categoria_clave ~ '^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$')
);

CREATE TABLE vec_bolsa_publica_datos.plazo_convocatoria (
    identificador_publico text NOT NULL,
    referencia text NOT NULL,
    tipo text NOT NULL CHECK (tipo ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    titulo text NOT NULL CHECK (char_length(titulo) BETWEEN 1 AND 180),
    descripcion text NOT NULL CHECK (char_length(descripcion) BETWEEN 1 AND 1000),
    abre_en timestamptz(6) NOT NULL,
    cierra_en timestamptz(6) NOT NULL,
    PRIMARY KEY (identificador_publico, referencia),
    FOREIGN KEY (identificador_publico)
        REFERENCES vec_bolsa_publica_datos.convocatoria_publica(identificador_publico)
        ON DELETE CASCADE,
    CHECK (referencia ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$'),
    CHECK (abre_en < cierra_en)
);

CREATE TABLE vec_bolsa_publica_datos.requisito_convocatoria (
    identificador_publico text NOT NULL,
    referencia text NOT NULL,
    orden integer NOT NULL CHECK (orden > 0),
    titulo text NOT NULL CHECK (char_length(titulo) BETWEEN 1 AND 180),
    descripcion text NOT NULL CHECK (char_length(descripcion) BETWEEN 1 AND 3000),
    obligatorio boolean NOT NULL,
    PRIMARY KEY (identificador_publico, referencia),
    UNIQUE (identificador_publico, orden),
    FOREIGN KEY (identificador_publico)
        REFERENCES vec_bolsa_publica_datos.convocatoria_publica(identificador_publico)
        ON DELETE CASCADE,
    CHECK (referencia ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$')
);

CREATE TABLE vec_bolsa_publica_datos.documento_convocatoria (
    identificador_publico text NOT NULL,
    referencia text NOT NULL,
    tipo text NOT NULL CHECK (tipo ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    orden integer NOT NULL CHECK (orden > 0),
    titulo text NOT NULL CHECK (char_length(titulo) BETWEEN 1 AND 180),
    descripcion text NOT NULL CHECK (char_length(descripcion) BETWEEN 1 AND 1000),
    formato text NOT NULL CHECK (formato ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    url text NOT NULL CHECK (char_length(url) BETWEEN 1 AND 2048),
    publicado_en timestamptz(6) NOT NULL,
    PRIMARY KEY (identificador_publico, referencia),
    UNIQUE (identificador_publico, orden),
    FOREIGN KEY (identificador_publico)
        REFERENCES vec_bolsa_publica_datos.convocatoria_publica(identificador_publico)
        ON DELETE CASCADE,
    CHECK (referencia ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$'),
    CHECK (url ~ '^/(?:[A-Za-z0-9._~!$&''()*+,;=:@%-]+/?)+$')
);

CREATE TABLE vec_bolsa_publica_datos.ayuda_convocatoria (
    identificador_publico text NOT NULL,
    referencia text NOT NULL,
    categoria text NOT NULL CHECK (categoria ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$'),
    orden integer NOT NULL CHECK (orden > 0),
    pregunta text NOT NULL CHECK (char_length(pregunta) BETWEEN 1 AND 300),
    respuesta text NOT NULL CHECK (char_length(respuesta) BETWEEN 1 AND 5000),
    PRIMARY KEY (identificador_publico, referencia),
    UNIQUE (identificador_publico, orden),
    FOREIGN KEY (identificador_publico)
        REFERENCES vec_bolsa_publica_datos.convocatoria_publica(identificador_publico)
        ON DELETE CASCADE,
    CHECK (referencia ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$')
);

-- Cualquier DML fuera de la publicación completa invalida de inmediato el
-- testigo barato que comprueba cada lectura. El superusuario/DBA queda fuera
-- del modelo de amenaza; ningún rol operativo puede desactivar estos triggers.
CREATE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $invalidar$
BEGIN
    UPDATE vec_bolsa_publica_datos.fuente
       SET manifiesto_sha256 = pg_catalog.repeat('0', 64)
     WHERE manifiesto_sha256 <> pg_catalog.repeat('0', 64);
    RETURN NULL;
END
$invalidar$;
REVOKE ALL ON FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1()
    FROM PUBLIC;

CREATE TRIGGER invalidar_manifiesto_catalogo_publico
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.catalogo_publico
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_entrada_catalogo
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.entrada_catalogo_publico
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_catalogo_categorias
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.catalogo_categorias
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_categoria
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.categoria_publica
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_convocatoria
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.convocatoria_publica
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_convocatoria_categoria
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.convocatoria_categoria
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_plazo
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.plazo_convocatoria
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_requisito
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.requisito_convocatoria
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_documento
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.documento_convocatoria
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_ayuda
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.ayuda_convocatoria
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();

GRANT USAGE ON SCHEMA vec_bolsa_publica_datos
    TO vec_bolsa_publica_publicacion_propietario;
GRANT INSERT ON ALL TABLES IN SCHEMA vec_bolsa_publica_datos
    TO vec_bolsa_publica_publicacion_propietario;
GRANT DELETE ON
    vec_bolsa_publica_datos.fuente,
    vec_bolsa_publica_datos.catalogo_publico,
    vec_bolsa_publica_datos.catalogo_categorias,
    vec_bolsa_publica_datos.convocatoria_publica
TO vec_bolsa_publica_publicacion_propietario;

SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;

-- Predicado privado para no aceptar campos silenciosamente ignorados. En esta
-- frontera una clave desconocida puede ser un dato personal clasificado de
-- forma errónea, por lo que se aplica allowlist exacta a cada objeto anidado.
CREATE FUNCTION vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
    valor jsonb,
    claves text[]
) RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog, pg_temp
RETURN pg_catalog.jsonb_typeof(valor) = 'object'
   AND valor ?& claves
   AND valor - claves = '{}'::jsonb;
REVOKE ALL ON FUNCTION
    vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(jsonb, text[])
    FROM PUBLIC;

-- Única frontera de escritura en operación. El rol publicador no recibe DML;
-- entrega una proyección pública completa, ya clasificada y aprobada. La
-- función serializa publicadores y migraciones, y cambia datos + testigo en
-- una sola transacción. Los lectores obtienen atomicidad mediante MVCC.
CREATE FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
    proyeccion jsonb,
    ancla_manifiesto_sha256 text
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET log_parameter_max_length_on_error = 0
SET work_mem = '8MB'
SET jit = 'off'
SET timezone = 'UTC'
SET datestyle = 'ISO, YMD'
AS $publicar$
DECLARE
    catalogo jsonb;
    entrada jsonb;
    categorias jsonb;
    snapshot jsonb;
    categoria jsonb;
    convocatoria jsonb;
    filas_insertadas bigint;
    total_entradas_catalogos bigint := 0;
    total_categorias bigint := 0;
BEGIN
    IF session_user <> 'vec_bolsa_publica_publicador_login'
       OR current_user <> 'vec_bolsa_publica_publicacion_propietario'
       OR NOT pg_catalog.pg_has_role(
        session_user, 'vec_bolsa_publica_publicador', 'MEMBER'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'publicacion rechazada: identidad sin rol publicador';
    END IF;
    -- Los SET de una funcion se aplican cuando la sentencia ya ha empezado y
    -- se restauran al retornar. Por eso esta frontera exige que el LOGIN haya
    -- iniciado ya la sentencia/transaccion con la politica gobernada.
    IF pg_catalog.current_setting('application_name') <> 'vec-bolsa-publicador'
       OR pg_catalog.replace(
           pg_catalog.current_setting('search_path'), ' ', ''
       ) <> 'pg_catalog,pg_temp'
       OR pg_catalog.current_setting('statement_timeout')::interval
          <> interval '60 seconds'
       OR pg_catalog.current_setting('lock_timeout')::interval
          <> interval '5 seconds'
       OR pg_catalog.current_setting('idle_in_transaction_session_timeout')::interval
          <> interval '5 seconds'
       OR pg_catalog.current_setting('transaction_timeout')::interval
          <> interval '2 minutes'
       OR pg_catalog.current_setting('log_parameter_max_length_on_error') <> '0'
       OR NOT (
           SELECT COALESCE(identidad.rolconfig, ARRAY[]::text[]) @> ARRAY[
               'application_name=vec-bolsa-publicador',
               'search_path="pg_catalog,pg_temp"',
               'statement_timeout=60s',
               'lock_timeout=5s',
               'idle_in_transaction_session_timeout=5s',
               'transaction_timeout=2min',
               'log_parameter_max_length_on_error=0'
           ]
             FROM pg_catalog.pg_roles AS identidad
            WHERE identidad.rolname = session_user
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'publicacion rechazada: LOGIN publicador sin limites gobernados';
    END IF;
    IF ancla_manifiesto_sha256 !~ '^[a-f0-9]{64}$'
       OR ancla_manifiesto_sha256 = pg_catalog.repeat('0', 64)
       OR pg_catalog.jsonb_typeof(proyeccion) <> 'object'
       OR proyeccion - ARRAY['fuente','catalogos','categorias','convocatorias'] <> '{}'::jsonb
       OR NOT (proyeccion ?& ARRAY['fuente','catalogos','categorias','convocatorias'])
       OR pg_catalog.jsonb_typeof(proyeccion->'fuente') <> 'object'
       OR pg_catalog.jsonb_typeof(proyeccion->'catalogos') <> 'array'
       OR pg_catalog.jsonb_typeof(proyeccion->'categorias') <> 'object'
       OR pg_catalog.jsonb_typeof(proyeccion->'convocatorias') <> 'array'
       OR pg_catalog.octet_length(proyeccion::text) > 268435456
       OR pg_catalog.jsonb_array_length(proyeccion->'catalogos') NOT BETWEEN 1 AND 1024
       OR pg_catalog.jsonb_array_length(proyeccion->'convocatorias') > 12000 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: contrato de proyeccion invalido';
    END IF;

    IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
           proyeccion->'fuente', ARRAY['revision','actualizada_en']
       ) IS DISTINCT FROM true
       OR pg_catalog.jsonb_typeof(proyeccion#>'{fuente,revision}') <> 'string'
       OR pg_catalog.jsonb_typeof(proyeccion#>'{fuente,actualizada_en}') <> 'string' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: fuente invalida';
    END IF;

    -- Primera pasada: el contrato recursivo completo se valida antes de tomar
    -- el candado o ejecutar DML. Un error nunca alcanza las tablas ni sus logs.
    FOR catalogo IN
        SELECT value FROM pg_catalog.jsonb_array_elements(proyeccion->'catalogos')
    LOOP
        IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
               catalogo, ARRAY['referencia','version','entradas']
           ) IS DISTINCT FROM true
           OR pg_catalog.jsonb_typeof(catalogo->'referencia') <> 'string'
           OR pg_catalog.jsonb_typeof(catalogo->'version') <> 'number'
           OR (catalogo->>'version') !~ '^[1-9][0-9]{0,9}$'
           OR (catalogo->>'version')::numeric > 2147483647
           OR pg_catalog.jsonb_typeof(catalogo->'entradas') <> 'array'
           OR pg_catalog.jsonb_array_length(catalogo->'entradas') NOT BETWEEN 1 AND 256 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: catalogo invalido';
        END IF;
        total_entradas_catalogos := total_entradas_catalogos
            + pg_catalog.jsonb_array_length(catalogo->'entradas');
        FOR entrada IN
            SELECT value FROM pg_catalog.jsonb_array_elements(catalogo->'entradas')
        LOOP
            IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                   entrada,
                   ARRAY['clave','etiqueta','descripcion','semantica','orden']
               ) IS DISTINCT FROM true
               OR pg_catalog.jsonb_typeof(entrada->'clave') <> 'string'
               OR pg_catalog.jsonb_typeof(entrada->'etiqueta') <> 'string'
               OR pg_catalog.jsonb_typeof(entrada->'descripcion') <> 'string'
               OR pg_catalog.jsonb_typeof(entrada->'semantica') <> 'string'
               OR pg_catalog.jsonb_typeof(entrada->'orden') <> 'number'
               OR (entrada->>'orden') !~ '^[1-9][0-9]{0,9}$'
               OR (entrada->>'orden')::numeric > 2147483647 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'publicacion rechazada: entrada de catalogo invalida';
            END IF;
        END LOOP;
    END LOOP;
    IF total_entradas_catalogos > 1024 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: demasiadas entradas de catalogo';
    END IF;

    categorias := proyeccion->'categorias';
    IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
           categorias, ARRAY['actual','snapshots']
       ) IS DISTINCT FROM true
       OR vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
              categorias->'actual',
              ARRAY[
                  'catalogo_id','catalogo_version','catalogo_huella_sha256',
                  'catalogo_huella_proyeccion_sha256'
              ]
          ) IS DISTINCT FROM true
       OR pg_catalog.jsonb_typeof(categorias#>'{actual,catalogo_id}') <> 'string'
       OR pg_catalog.jsonb_typeof(categorias#>'{actual,catalogo_version}') <> 'number'
       OR (categorias#>>'{actual,catalogo_version}') !~ '^[1-9][0-9]{0,9}$'
       OR (categorias#>>'{actual,catalogo_version}')::numeric > 2147483647
       OR pg_catalog.jsonb_typeof(categorias#>'{actual,catalogo_huella_sha256}') <> 'string'
       OR (categorias#>>'{actual,catalogo_huella_sha256}') !~ '^[a-f0-9]{64}$'
       OR pg_catalog.jsonb_typeof(categorias#>'{actual,catalogo_huella_proyeccion_sha256}') <> 'string'
       OR (categorias#>>'{actual,catalogo_huella_proyeccion_sha256}') !~ '^[a-f0-9]{64}$'
       OR pg_catalog.jsonb_typeof(categorias->'snapshots') <> 'array'
       OR pg_catalog.jsonb_array_length(categorias->'snapshots') NOT BETWEEN 1 AND 64 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: catalogo profesional invalido';
    END IF;

    FOR snapshot IN
        SELECT value FROM pg_catalog.jsonb_array_elements(categorias->'snapshots')
    LOOP
        IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
               snapshot,
               ARRAY[
                   'catalogo_id','version','huella_gobernada_sha256',
                   'huella_proyeccion_publica_sha256','categorias'
               ]
           ) IS DISTINCT FROM true
           OR pg_catalog.jsonb_typeof(snapshot->'catalogo_id') <> 'string'
           OR snapshot->>'catalogo_id' <> categorias#>>'{actual,catalogo_id}'
           OR pg_catalog.jsonb_typeof(snapshot->'version') <> 'number'
           OR (snapshot->>'version') !~ '^[1-9][0-9]{0,9}$'
           OR (snapshot->>'version')::numeric > 2147483647
           OR pg_catalog.jsonb_typeof(snapshot->'huella_gobernada_sha256') <> 'string'
           OR (snapshot->>'huella_gobernada_sha256') !~ '^[a-f0-9]{64}$'
           OR pg_catalog.jsonb_typeof(snapshot->'huella_proyeccion_publica_sha256') <> 'string'
           OR (snapshot->>'huella_proyeccion_publica_sha256') !~ '^[a-f0-9]{64}$'
           OR pg_catalog.jsonb_typeof(snapshot->'categorias') <> 'array'
           OR pg_catalog.jsonb_array_length(snapshot->'categorias') NOT BETWEEN 1 AND 1024 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: snapshot profesional invalido';
        END IF;

        total_categorias := total_categorias
            + pg_catalog.jsonb_array_length(snapshot->'categorias');
        FOR categoria IN
            SELECT value FROM pg_catalog.jsonb_array_elements(snapshot->'categorias')
        LOOP
            IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                   categoria,
                   ARRAY[
                       'clave','etiqueta','descripcion','semantica','orden','area',
                       'area_etiqueta','suscribible','vigente_desde','vigente_hasta'
                   ]
               ) IS DISTINCT FROM true
               OR pg_catalog.jsonb_typeof(categoria->'clave') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'etiqueta') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'descripcion') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'semantica') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'orden') <> 'number'
               OR (categoria->>'orden') !~ '^[1-9][0-9]{0,9}$'
               OR (categoria->>'orden')::numeric > 2147483647
               OR pg_catalog.jsonb_typeof(categoria->'area') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'area_etiqueta') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'suscribible') <> 'boolean'
               OR pg_catalog.jsonb_typeof(categoria->'vigente_desde') <> 'string'
               OR pg_catalog.jsonb_typeof(categoria->'vigente_hasta') NOT IN ('string','null') THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'publicacion rechazada: categoria profesional invalida';
            END IF;
        END LOOP;
    END LOOP;
    IF total_categorias > 4096
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(categorias->'snapshots') AS elemento
         GROUP BY elemento.value->>'catalogo_id', elemento.value->>'version'
           HAVING count(*) > 1
       )
       OR 1 <> (
           SELECT count(*)
             FROM pg_catalog.jsonb_array_elements(categorias->'snapshots') AS elemento
            WHERE elemento.value->>'catalogo_id' = categorias#>>'{actual,catalogo_id}'
              AND elemento.value->>'version' = categorias#>>'{actual,catalogo_version}'
              AND elemento.value->>'huella_gobernada_sha256' =
                  categorias#>>'{actual,catalogo_huella_sha256}'
              AND elemento.value->>'huella_proyeccion_publica_sha256' =
                  categorias#>>'{actual,catalogo_huella_proyeccion_sha256}'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: snapshots profesionales invalidos';
    END IF;

    FOR convocatoria IN
        SELECT value FROM pg_catalog.jsonb_array_elements(proyeccion->'convocatorias')
    LOOP
        IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
               convocatoria,
               ARRAY[
                   'identificador_publico','version_publica','estado','tipo',
                   'huella_publica_sha256','huella_resumen_publico_sha256',
                   'titulo','resumen','descripcion','publicada_en','actualizada_en',
                   'catalogo_categorias','categorias','plazos','requisitos','documentos','ayuda'
               ]
           ) IS DISTINCT FROM true
           OR pg_catalog.jsonb_typeof(convocatoria->'identificador_publico') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'version_publica') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'estado') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'tipo') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'huella_publica_sha256') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'huella_resumen_publico_sha256') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'titulo') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'resumen') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'descripcion') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'publicada_en') <> 'string'
           OR pg_catalog.jsonb_typeof(convocatoria->'actualizada_en') <> 'string'
           OR vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                  convocatoria->'catalogo_categorias',
                  ARRAY[
                      'catalogo_id','catalogo_version','catalogo_huella_sha256',
                      'catalogo_huella_proyeccion_sha256'
                  ]
              ) IS DISTINCT FROM true
           OR pg_catalog.jsonb_typeof(
                  convocatoria#>'{catalogo_categorias,catalogo_id}'
              ) <> 'string'
           OR pg_catalog.jsonb_typeof(
                  convocatoria#>'{catalogo_categorias,catalogo_version}'
              ) <> 'number'
           OR (convocatoria#>>'{catalogo_categorias,catalogo_version}') !~ '^[1-9][0-9]{0,9}$'
           OR (convocatoria#>>'{catalogo_categorias,catalogo_version}')::numeric > 2147483647
           OR pg_catalog.jsonb_typeof(
                  convocatoria#>'{catalogo_categorias,catalogo_huella_sha256}'
              ) <> 'string'
           OR (convocatoria#>>'{catalogo_categorias,catalogo_huella_sha256}') !~ '^[a-f0-9]{64}$'
           OR pg_catalog.jsonb_typeof(
                  convocatoria#>'{catalogo_categorias,catalogo_huella_proyeccion_sha256}'
              ) <> 'string'
           OR (convocatoria#>>'{catalogo_categorias,catalogo_huella_proyeccion_sha256}') !~ '^[a-f0-9]{64}$'
           OR pg_catalog.jsonb_typeof(convocatoria->'categorias') <> 'array'
           OR pg_catalog.jsonb_typeof(convocatoria->'plazos') <> 'array'
           OR pg_catalog.jsonb_typeof(convocatoria->'requisitos') <> 'array'
           OR pg_catalog.jsonb_typeof(convocatoria->'documentos') <> 'array'
           OR pg_catalog.jsonb_typeof(convocatoria->'ayuda') <> 'array'
           OR pg_catalog.jsonb_array_length(convocatoria->'categorias') NOT BETWEEN 1 AND 128
           OR pg_catalog.jsonb_array_length(convocatoria->'plazos') > 64
           OR pg_catalog.jsonb_array_length(convocatoria->'requisitos') > 256
           OR pg_catalog.jsonb_array_length(convocatoria->'documentos') > 256
           OR pg_catalog.jsonb_array_length(convocatoria->'ayuda') > 128 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: convocatoria invalida';
        END IF;
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.jsonb_array_elements(convocatoria->'categorias') AS elemento
             WHERE pg_catalog.jsonb_typeof(elemento.value) <> 'string'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: categorias de convocatoria invalidas';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_catalog.jsonb_array_elements(convocatoria->'plazos') AS elemento
             WHERE vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                       elemento.value,
                       ARRAY['referencia','tipo','titulo','descripcion','abre_en','cierra_en']
                   ) IS DISTINCT FROM true
                OR pg_catalog.jsonb_typeof(elemento.value->'referencia') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'tipo') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'titulo') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'descripcion') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'abre_en') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'cierra_en') <> 'string'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: plazo invalido';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_catalog.jsonb_array_elements(convocatoria->'requisitos') AS elemento
             WHERE vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                       elemento.value,
                       ARRAY['referencia','orden','titulo','descripcion','obligatorio']
                   ) IS DISTINCT FROM true
                OR pg_catalog.jsonb_typeof(elemento.value->'referencia') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'orden') <> 'number'
                OR (elemento.value->>'orden') !~ '^[1-9][0-9]{0,9}$'
                OR (elemento.value->>'orden')::numeric > 2147483647
                OR pg_catalog.jsonb_typeof(elemento.value->'titulo') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'descripcion') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'obligatorio') <> 'boolean'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: requisito invalido';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_catalog.jsonb_array_elements(convocatoria->'documentos') AS elemento
             WHERE vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                       elemento.value,
                       ARRAY[
                           'referencia','tipo','orden','titulo','descripcion',
                           'formato','url','publicado_en'
                       ]
                   ) IS DISTINCT FROM true
                OR pg_catalog.jsonb_typeof(elemento.value->'referencia') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'tipo') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'orden') <> 'number'
                OR (elemento.value->>'orden') !~ '^[1-9][0-9]{0,9}$'
                OR (elemento.value->>'orden')::numeric > 2147483647
                OR pg_catalog.jsonb_typeof(elemento.value->'titulo') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'descripcion') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'formato') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'url') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'publicado_en') <> 'string'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: documento invalido';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_catalog.jsonb_array_elements(convocatoria->'ayuda') AS elemento
             WHERE vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                       elemento.value,
                       ARRAY['referencia','categoria','orden','pregunta','respuesta']
                   ) IS DISTINCT FROM true
                OR pg_catalog.jsonb_typeof(elemento.value->'referencia') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'categoria') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'orden') <> 'number'
                OR (elemento.value->>'orden') !~ '^[1-9][0-9]{0,9}$'
                OR (elemento.value->>'orden')::numeric > 2147483647
                OR pg_catalog.jsonb_typeof(elemento.value->'pregunta') <> 'string'
                OR pg_catalog.jsonb_typeof(elemento.value->'respuesta') <> 'string'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: ayuda invalida';
        END IF;
    END LOOP;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
    );

    INSERT INTO vec_bolsa_publica_datos.manifiesto_consumido(manifiesto_sha256)
    VALUES (ancla_manifiesto_sha256)
    ON CONFLICT DO NOTHING;
    GET DIAGNOSTICS filas_insertadas = ROW_COUNT;
    IF filas_insertadas <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: el manifiesto ya fue consumido';
    END IF;

    DELETE FROM vec_bolsa_publica_datos.convocatoria_publica;
    DELETE FROM vec_bolsa_publica_datos.catalogo_categorias;
    DELETE FROM vec_bolsa_publica_datos.catalogo_publico;
    DELETE FROM vec_bolsa_publica_datos.fuente;

    FOR catalogo IN
        SELECT value FROM pg_catalog.jsonb_array_elements(proyeccion->'catalogos')
    LOOP
        INSERT INTO vec_bolsa_publica_datos.catalogo_publico(
            referencia, version, activo
        ) VALUES (
            catalogo->>'referencia', (catalogo->>'version')::integer, true
        );
        FOR entrada IN
            SELECT value FROM pg_catalog.jsonb_array_elements(catalogo->'entradas')
        LOOP
            INSERT INTO vec_bolsa_publica_datos.entrada_catalogo_publico(
                referencia, version, clave, etiqueta, descripcion,
                semantica, orden, publicable
            ) VALUES (
                catalogo->>'referencia', (catalogo->>'version')::integer,
                entrada->>'clave', entrada->>'etiqueta',
                entrada->>'descripcion', entrada->>'semantica',
                (entrada->>'orden')::integer, true
            );
        END LOOP;
    END LOOP;

    FOR snapshot IN
        SELECT value FROM pg_catalog.jsonb_array_elements(categorias->'snapshots')
    LOOP
        INSERT INTO vec_bolsa_publica_datos.catalogo_categorias(
            catalogo_id, version, huella_gobernada_sha256,
            huella_proyeccion_publica_sha256, actual
        ) VALUES (
            snapshot->>'catalogo_id', (snapshot->>'version')::integer,
            snapshot->>'huella_gobernada_sha256',
            snapshot->>'huella_proyeccion_publica_sha256',
            snapshot->>'catalogo_id' = categorias#>>'{actual,catalogo_id}'
                AND snapshot->>'version' = categorias#>>'{actual,catalogo_version}'
                AND snapshot->>'huella_gobernada_sha256' =
                    categorias#>>'{actual,catalogo_huella_sha256}'
                AND snapshot->>'huella_proyeccion_publica_sha256' =
                    categorias#>>'{actual,catalogo_huella_proyeccion_sha256}'
        );
        FOR categoria IN
            SELECT value FROM pg_catalog.jsonb_array_elements(snapshot->'categorias')
        LOOP
            INSERT INTO vec_bolsa_publica_datos.categoria_publica(
                catalogo_id, version, clave, etiqueta, descripcion, semantica,
                orden, area, area_etiqueta, suscribible, vigente_desde,
                vigente_hasta, publicable
            ) VALUES (
                snapshot->>'catalogo_id', (snapshot->>'version')::integer,
                categoria->>'clave', categoria->>'etiqueta',
                categoria->>'descripcion', categoria->>'semantica',
                (categoria->>'orden')::integer, categoria->>'area',
                categoria->>'area_etiqueta', (categoria->>'suscribible')::boolean,
                (categoria->>'vigente_desde')::timestamptz,
                NULLIF(categoria->>'vigente_hasta', '')::timestamptz, true
            );
        END LOOP;
    END LOOP;

    FOR convocatoria IN
        SELECT value FROM pg_catalog.jsonb_array_elements(proyeccion->'convocatorias')
    LOOP
        INSERT INTO vec_bolsa_publica_datos.convocatoria_publica(
            identificador_publico, version_publica, estado, tipo,
            catalogo_categorias_id, catalogo_categorias_version,
            catalogo_categorias_huella_sha256,
            catalogo_categorias_huella_proyeccion_sha256,
            huella_publica_sha256,
            huella_resumen_publico_sha256, titulo, resumen, descripcion,
            publicada_en, actualizada_en, publicable
        ) VALUES (
            convocatoria->>'identificador_publico', convocatoria->>'version_publica',
            convocatoria->>'estado', convocatoria->>'tipo',
            convocatoria#>>'{catalogo_categorias,catalogo_id}',
            (convocatoria#>>'{catalogo_categorias,catalogo_version}')::integer,
            convocatoria#>>'{catalogo_categorias,catalogo_huella_sha256}',
            convocatoria#>>'{catalogo_categorias,catalogo_huella_proyeccion_sha256}',
            convocatoria->>'huella_publica_sha256',
            convocatoria->>'huella_resumen_publico_sha256', convocatoria->>'titulo',
            convocatoria->>'resumen', convocatoria->>'descripcion',
            (convocatoria->>'publicada_en')::timestamptz,
            (convocatoria->>'actualizada_en')::timestamptz, true
        );
        INSERT INTO vec_bolsa_publica_datos.convocatoria_categoria(
            identificador_publico, catalogo_id, catalogo_version, categoria_clave
        )
        SELECT convocatoria->>'identificador_publico',
               convocatoria#>>'{catalogo_categorias,catalogo_id}',
               (convocatoria#>>'{catalogo_categorias,catalogo_version}')::integer,
               value
          FROM pg_catalog.jsonb_array_elements_text(convocatoria->'categorias');

        INSERT INTO vec_bolsa_publica_datos.plazo_convocatoria(
            identificador_publico, referencia, tipo, titulo, descripcion,
            abre_en, cierra_en
        )
        SELECT convocatoria->>'identificador_publico', dato.referencia, dato.tipo,
               dato.titulo, dato.descripcion, dato.abre_en, dato.cierra_en
          FROM pg_catalog.jsonb_to_recordset(convocatoria->'plazos') AS dato(
              referencia text, tipo text, titulo text, descripcion text,
              abre_en timestamptz, cierra_en timestamptz
          );
        INSERT INTO vec_bolsa_publica_datos.requisito_convocatoria(
            identificador_publico, referencia, orden, titulo, descripcion, obligatorio
        )
        SELECT convocatoria->>'identificador_publico', dato.referencia, dato.orden,
               dato.titulo, dato.descripcion, dato.obligatorio
          FROM pg_catalog.jsonb_to_recordset(convocatoria->'requisitos') AS dato(
              referencia text, orden integer, titulo text, descripcion text,
              obligatorio boolean
          );
        INSERT INTO vec_bolsa_publica_datos.documento_convocatoria(
            identificador_publico, referencia, tipo, orden, titulo,
            descripcion, formato, url, publicado_en
        )
        SELECT convocatoria->>'identificador_publico', dato.referencia, dato.tipo,
               dato.orden, dato.titulo, dato.descripcion, dato.formato,
               dato.url, dato.publicado_en
          FROM pg_catalog.jsonb_to_recordset(convocatoria->'documentos') AS dato(
              referencia text, tipo text, orden integer, titulo text,
              descripcion text, formato text, url text, publicado_en timestamptz
          );
        INSERT INTO vec_bolsa_publica_datos.ayuda_convocatoria(
            identificador_publico, referencia, categoria, orden, pregunta, respuesta
        )
        SELECT convocatoria->>'identificador_publico', dato.referencia,
               dato.categoria, dato.orden, dato.pregunta, dato.respuesta
          FROM pg_catalog.jsonb_to_recordset(convocatoria->'ayuda') AS dato(
              referencia text, categoria text, orden integer,
              pregunta text, respuesta text
          );
    END LOOP;

    INSERT INTO vec_bolsa_publica_datos.fuente(
        control_id, revision, actualizada_en, manifiesto_sha256
    ) VALUES (
        true, proyeccion#>>'{fuente,revision}',
        (proyeccion#>>'{fuente,actualizada_en}')::timestamptz,
        ancla_manifiesto_sha256
    );
EXCEPTION
    WHEN data_exception OR integrity_constraint_violation THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: contenido invalido';
END;
$publicar$;

REVOKE ALL ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb, text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_publica_publicacion
    TO vec_bolsa_publica_publicador;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb, text)
    TO vec_bolsa_publica_publicador_login;

SET LOCAL ROLE vec_bolsa_publica_propietario;

CREATE VIEW vec_bolsa_publica_lectura.fuente_publica_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT control_id, revision, actualizada_en, manifiesto_sha256
  FROM vec_bolsa_publica_datos.fuente;

CREATE VIEW vec_bolsa_publica_lectura.entradas_catalogos_publicos_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT catalogo.referencia, catalogo.version, entrada.clave,
       entrada.etiqueta, entrada.descripcion, entrada.semantica,
       entrada.orden, entrada.publicable
  FROM vec_bolsa_publica_datos.catalogo_publico AS catalogo
  JOIN vec_bolsa_publica_datos.entrada_catalogo_publico AS entrada
    ON entrada.referencia = catalogo.referencia
   AND entrada.version = catalogo.version
 WHERE catalogo.activo AND entrada.publicable;

CREATE VIEW vec_bolsa_publica_lectura.catalogos_categorias_publicos_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT catalogo.catalogo_id, catalogo.version,
       catalogo.huella_gobernada_sha256,
       catalogo.huella_proyeccion_publica_sha256,
	   catalogo.actual,
       fuente.revision, fuente.actualizada_en
  FROM vec_bolsa_publica_datos.catalogo_categorias AS catalogo
 CROSS JOIN vec_bolsa_publica_datos.fuente AS fuente;

CREATE VIEW vec_bolsa_publica_lectura.categorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT categoria.catalogo_id, categoria.version, categoria.clave,
       categoria.etiqueta, categoria.descripcion, categoria.semantica,
       categoria.orden, categoria.area, categoria.area_etiqueta,
       categoria.suscribible, categoria.vigente_desde, categoria.vigente_hasta
  FROM vec_bolsa_publica_datos.categoria_publica AS categoria
  JOIN vec_bolsa_publica_datos.catalogo_categorias AS catalogo
    ON catalogo.catalogo_id = categoria.catalogo_id
   AND catalogo.version = categoria.version
 WHERE categoria.publicable;

CREATE VIEW vec_bolsa_publica_lectura.convocatorias_publicadas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT identificador_publico, version_publica,
       estado, tipo, catalogo_categorias_id, catalogo_categorias_version,
       catalogo_categorias_huella_sha256,
       catalogo_categorias_huella_proyeccion_sha256,
       huella_publica_sha256,
       huella_resumen_publico_sha256,
       titulo, resumen, descripcion, publicada_en, actualizada_en, busqueda
  FROM vec_bolsa_publica_datos.convocatoria_publica
 WHERE publicable AND revocada_en IS NULL;

CREATE VIEW vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT categoria.identificador_publico, categoria.catalogo_id,
       categoria.catalogo_version, categoria.categoria_clave
  FROM vec_bolsa_publica_datos.convocatoria_categoria AS categoria
  JOIN vec_bolsa_publica_datos.convocatoria_publica AS convocatoria
    ON convocatoria.identificador_publico = categoria.identificador_publico
 WHERE convocatoria.publicable AND convocatoria.revocada_en IS NULL;

CREATE VIEW vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT plazo.identificador_publico, plazo.referencia, plazo.tipo,
       plazo.titulo, plazo.descripcion, plazo.abre_en, plazo.cierra_en
  FROM vec_bolsa_publica_datos.plazo_convocatoria AS plazo
  JOIN vec_bolsa_publica_datos.convocatoria_publica AS convocatoria
    ON convocatoria.identificador_publico = plazo.identificador_publico
 WHERE convocatoria.publicable AND convocatoria.revocada_en IS NULL;

CREATE VIEW vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT requisito.identificador_publico, requisito.referencia,
       requisito.orden, requisito.titulo, requisito.descripcion,
       requisito.obligatorio
  FROM vec_bolsa_publica_datos.requisito_convocatoria AS requisito
  JOIN vec_bolsa_publica_datos.convocatoria_publica AS convocatoria
    ON convocatoria.identificador_publico = requisito.identificador_publico
 WHERE convocatoria.publicable AND convocatoria.revocada_en IS NULL;

CREATE VIEW vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT documento.identificador_publico, documento.referencia,
       documento.tipo, documento.orden, documento.titulo,
       documento.descripcion, documento.formato, documento.url,
       documento.publicado_en
  FROM vec_bolsa_publica_datos.documento_convocatoria AS documento
  JOIN vec_bolsa_publica_datos.convocatoria_publica AS convocatoria
    ON convocatoria.identificador_publico = documento.identificador_publico
 WHERE convocatoria.publicable AND convocatoria.revocada_en IS NULL;

CREATE VIEW vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v2
WITH (security_barrier = true, security_invoker = false) AS
SELECT ayuda.identificador_publico, ayuda.referencia, ayuda.categoria,
       ayuda.orden, ayuda.pregunta, ayuda.respuesta
  FROM vec_bolsa_publica_datos.ayuda_convocatoria AS ayuda
  JOIN vec_bolsa_publica_datos.convocatoria_publica AS convocatoria
    ON convocatoria.identificador_publico = ayuda.identificador_publico
 WHERE convocatoria.publicable AND convocatoria.revocada_en IS NULL;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_publica_datos FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_publica_lectura FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_publica_datos
    FROM vec_bolsa_publica_publicador;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_publica_lectura
    FROM vec_bolsa_publica_publicador;
REVOKE ALL ON SCHEMA vec_bolsa_publica_datos, vec_bolsa_publica_lectura
    FROM vec_bolsa_publica_publicador;
GRANT USAGE ON SCHEMA vec_bolsa_publica_lectura TO vec_bolsa_publica_consulta;
GRANT SELECT ON
    vec_bolsa_publica_lectura.fuente_publica_v2,
    vec_bolsa_publica_lectura.entradas_catalogos_publicos_v2,
    vec_bolsa_publica_lectura.catalogos_categorias_publicos_v2,
    vec_bolsa_publica_lectura.categorias_publicas_v2,
    vec_bolsa_publica_lectura.convocatorias_publicadas_v2,
    vec_bolsa_publica_lectura.categorias_convocatorias_publicas_v2,
    vec_bolsa_publica_lectura.plazos_convocatorias_publicas_v2,
    vec_bolsa_publica_lectura.requisitos_convocatorias_publicas_v2,
    vec_bolsa_publica_lectura.documentos_convocatorias_publicas_v2,
    vec_bolsa_publica_lectura.ayuda_convocatorias_publicas_v2
TO vec_bolsa_publica_consulta;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_publica_propietario
    IN SCHEMA vec_bolsa_publica_datos REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_publica_propietario
    IN SCHEMA vec_bolsa_publica_lectura REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_publica_propietario
    IN SCHEMA vec_bolsa_publica_datos REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_publica_publicacion_propietario
    IN SCHEMA vec_bolsa_publica_publicacion REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

COMMIT;
