-- Semilla pública F0-29 para la copia desechable HZ37 sin datos.
-- Fuente: AD172 (tabla y RLS), AD233 (audiencias y operaciones),
-- AD278 (orígenes), contexto_actor_v1/roles_up.sql (migrador de CTX45)
-- y documentos/migraciones/000009_custodia_documento_firmado.up.sql
-- (tipo documental reservado que D19 espera en la copia).
-- CTX15 y CTX2 aportan las filas iniciales de sus dos controles técnicos;
-- autorizacion/000003 aporta la del punto de control de motivos V2.
-- Sólo prepara procedencia y catálogo técnicos: no publica permisos de negocio, actos,
-- personas, firmas, recibos ni material criptográfico.
-- Ejecutar una vez como superusuario local tras restaurar esquema_hz37.sql.gz
-- y antes de lista_sql_main_tras_hz37.txt; nunca instalar en producción.
BEGIN;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '2min';
SET LOCAL search_path = pg_catalog;

DO $pre$
DECLARE
  relacion record;
  total integer := 0;
  contiene_filas boolean;
BEGIN
  IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
     OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper)
     OR to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
     OR to_regclass('vec_documentos.modulo_referencia_expediente_v1') IS NULL
     OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
     OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
     OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
     OR to_regrole('vec_contexto_actor_v1_migrador') IS NOT NULL
     OR to_regrole('vec_ct_o207_runtime') IS NOT NULL
     OR to_regrole('vec_bolsa_llamamientos_desarrollo') IS NOT NULL
  THEN RAISE EXCEPTION 'F0-29: esquema o roles de partida incompatibles' USING ERRCODE = '55000';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_roles
    WHERE rolname IN ('vec_contratacion_temporal_ejecutor',
                      'vec_bolsa_llamamientos_ejecutor',
                      'vec_autorizacion_atestada_v3_propietario')
      AND rolcanlogin
  ) OR NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'vec_autorizacion_atestada_v3'
      AND c.relname = 'configuracion_origen_consumos_v1'
      AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner = 'vec_autorizacion_atestada_v3_propietario'::regrole
  ) OR NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'vec_documentos'
      AND c.relname = 'modulo_referencia_expediente_v1'
      AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner = 'vec_documentos_propietario'::regrole
  ) THEN RAISE EXCEPTION 'F0-29: propietario o RLS incompatibles' USING ERRCODE = '55000';
  END IF;

  -- Protege frente a una ejecución accidental sobre una base con historia.
  FOR relacion IN
    SELECT n.nspname, c.relname
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE (n.nspname = 'public' OR left(n.nspname, 4) = 'vec_')
      AND c.relkind IN ('r', 'm')
      AND NOT EXISTS (
        SELECT 1 FROM pg_depend d
        WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e')
  LOOP
    EXECUTE format('SELECT EXISTS (SELECT 1 FROM %I.%I)',
                   relacion.nspname, relacion.relname) INTO contiene_filas;
    IF contiene_filas THEN
      RAISE EXCEPTION 'F0-29: la base contiene filas; semilla sólo para HZ37 vacío'
        USING ERRCODE = '55000';
    END IF;
    total := total + 1;
  END LOOP;
  IF total <> 662 THEN
    RAISE EXCEPTION 'F0-29: relaciones esperadas 662, encontradas %', total
      USING ERRCODE = '55000';
  END IF;
END $pre$;

-- Los roles globales y ACL de base no pertenecen al volcado de esquema.
-- Contratos: roles_candidato_externo_up.sql y CTX15, sin pertenencias nuevas.
DO $grupos_ctx15$
DECLARE nombre text;
BEGIN
  IF (SELECT count(*) FROM pg_roles
      WHERE rolname IN ('vec_contexto_actor_v1_candidato_externo',
                        'vec_contexto_actor_v1_usuarios_externo')) <> 2
     OR EXISTS (
       SELECT 1 FROM pg_roles WHERE rolname IN (
         'vec_contexto_actor_v1_candidato_externo', 'vec_contexto_actor_v1_usuarios_externo')
       AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
            OR rolreplication OR rolbypassrls OR rolconfig IS NOT NULL))
  THEN RAISE EXCEPTION 'F0-29: grupos externos de contexto incompatibles'
    USING ERRCODE = '55000';
  END IF;
  FOREACH nombre IN ARRAY ARRAY[
    'vec_contexto_actor_v1_candidato_externo', 'vec_contexto_actor_v1_usuarios_externo']
  LOOP
    EXECUTE format('ALTER ROLE %I NOINHERIT', nombre);
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), nombre);
  END LOOP;
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END $grupos_ctx15$;

-- CTX15 inicializa este contador; el volcado de solo esquema omite la fila.
-- Se repone después de verificar todas las tablas vacías, sin cambiar guardas.
DO $control_ctx15_pre$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'vec_contexto_actor_v1'
      AND c.relname = 'control_generacion_contexto_externo_v1'
      AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner = 'vec_contexto_actor_v1_propietario'::regrole
  ) THEN RAISE EXCEPTION 'F0-29: control técnico CTX15 incompatible'
    USING ERRCODE = '55000';
  END IF;
END $control_ctx15_pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.control_generacion_contexto_externo_v1
  (control_id, generacion) VALUES (true, 0);
DO $control_ctx15_post$
BEGIN
  IF (SELECT count(*) FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1) <> 1
     OR NOT EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.control_generacion_contexto_externo_v1
       WHERE control_id AND generacion = 0)
  THEN RAISE EXCEPTION 'F0-29: fila técnica CTX15 divergente'
    USING ERRCODE = '55000';
  END IF;
END $control_ctx15_post$;
-- CTX2 inicializa el cerrojo MVCC de los punteros actuales; el volcado de solo
-- esquema también omite su fila. Sin ella no se puede mover ningún puntero de
-- cuenta, persona, perfil o vínculo (prueba CA48 de la tarea F1-105).
INSERT INTO vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  (control_id, generacion, actualizada_en) VALUES (true, 0, clock_timestamp());
DO $control_ctx2_post$
BEGIN
  IF (SELECT count(*) FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2) <> 1
     OR NOT EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
       WHERE control_id AND generacion = 0)
  THEN RAISE EXCEPTION 'F0-29: fila técnica CTX2 divergente'
    USING ERRCODE = '55000';
  END IF;
END $control_ctx2_post$;
-- Cierre de tipos de CTX1: pg_dump no conserva la ACL de los tipos fila.
DO $cerrar_tipos_contexto$
DECLARE t record;
BEGIN
  FOR t IN
    SELECT ty.typname FROM pg_type ty JOIN pg_namespace ns ON ns.oid = ty.typnamespace
    WHERE ns.nspname = 'vec_contexto_actor_v1' AND ty.typtype IN ('c', 'd', 'e')
  LOOP
    EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC, vec_contexto_actor_v1_runtime', t.typname);
  END LOOP;
END $cerrar_tipos_contexto$;
RESET ROLE;

-- D14 creó esta fila antes de activar RLS; HZ37 conservó sólo su esquema.
-- La copia vacía repone el mismo catálogo técnico, sin conceder permisos.
INSERT INTO vec_documentos.modulo_referencia_expediente_v1
  (codigo, modulo_id, version, patron_id)
VALUES ('ct', 'contratacion_temporal', 1, '^[0-9a-f]{64}$');
DO $catalogo_d14$
BEGIN
  IF (SELECT count(*) FROM vec_documentos.modulo_referencia_expediente_v1) <> 1
     OR NOT EXISTS (
       SELECT 1 FROM vec_documentos.modulo_referencia_expediente_v1
       WHERE codigo = 'ct' AND modulo_id = 'contratacion_temporal'
         AND version = 1 AND patron_id = '^[0-9a-f]{64}$')
  THEN RAISE EXCEPTION 'F0-29: catálogo técnico D14 divergente'
    USING ERRCODE = '55000';
  END IF;
END $catalogo_d14$;

-- autorizacion/000003 crea el punto de control de los motivos V2
-- con su fila (true, 0); el volcado sin datos la omite y su disparador impide
-- insertarla después. Sin ella ningún catálogo de motivos se publica (CRN18).
-- Se repone como superusuario sin disparadores de réplica, tras comprobar las
-- 662 tablas vacías, y se vuelve enseguida al modo normal.
DO $checkpoint_motivos_v2_pre$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'vec_autorizacion'
      AND c.relname = 'motivo_v2_checkpoint_origen'
      AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner = 'vec_autorizacion_propietario'::regrole
  ) OR EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_checkpoint_origen)
  THEN RAISE EXCEPTION 'F0-29: punto de control de motivos V2 incompatible'
    USING ERRCODE = '55000';
  END IF;
END $checkpoint_motivos_v2_pre$;
SET LOCAL session_replication_role = replica;
INSERT INTO vec_autorizacion.motivo_v2_checkpoint_origen
  (control_id, ultima_secuencia, actualizado_en) VALUES (true, 0, clock_timestamp());
SET LOCAL session_replication_role = origin;
DO $checkpoint_motivos_v2_post$
BEGIN
  IF (SELECT count(*) FROM vec_autorizacion.motivo_v2_checkpoint_origen) <> 1
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion.motivo_v2_checkpoint_origen
       WHERE control_id AND ultima_secuencia = 0
         AND ultimo_evento_ref IS NULL AND ultima_huella_evento_sha256 IS NULL)
  THEN RAISE EXCEPTION 'F0-29: fila técnica de motivos V2 divergente'
    USING ERRCODE = '55000';
  END IF;
END $checkpoint_motivos_v2_post$;

-- El extractor de roles del DDL no ve este grupo: CTX45 le concede EXECUTE.
CREATE ROLE vec_contexto_actor_v1_migrador NOLOGIN NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ct_o207_runtime LOGIN INHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
CREATE ROLE vec_bolsa_llamamientos_desarrollo LOGIN INHERIT NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
DO $caducidad$
DECLARE
  limite text := (clock_timestamp() + INTERVAL '90 days')::text;
BEGIN
  EXECUTE format('ALTER ROLE vec_ct_o207_runtime VALID UNTIL %L', limite);
  EXECUTE format('ALTER ROLE vec_bolsa_llamamientos_desarrollo VALID UNTIL %L', limite);
END $caducidad$;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct_o207_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_bolsa_llamamientos_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
  (login_nombre, audiencia_consumo, operacion, proceso, canal_permitido)
VALUES
  ('vec_ct_o207_runtime',
   'vec_contratacion_temporal.vinculo_emision_bolsa.v1',
   'contratacion_temporal.bolsa.vincular',
   'vec-server', 'interna_corporativa'),
  ('vec_bolsa_llamamientos_desarrollo',
   'vec_bolsa_llamamientos.llamamiento.emitir.v1',
   'llamamiento.emitir.v1',
   'bolsa_llamamientos_ci', 'interna_corporativa'),
  ('vec_bolsa_llamamientos_desarrollo',
   'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1',
   'bolsa.situacion_participacion.cambiar',
   'vec-server', 'interna_corporativa');
RESET ROLE;

DO $post$
DECLARE
  rol record;
BEGIN
  FOR rol IN
    SELECT r.rolname, r.rolcanlogin, r.rolinherit, r.rolsuper,
           r.rolcreatedb, r.rolcreaterole, r.rolreplication,
           r.rolbypassrls, r.rolvaliduntil, r.rolconfig,
           a.rolpassword AS password_hash,
           m.roleid, m.admin_option, m.inherit_option, m.set_option
    FROM pg_roles r
    JOIN pg_authid a ON a.oid = r.oid
    JOIN pg_auth_members m ON m.member = r.oid
    WHERE r.rolname IN ('vec_ct_o207_runtime',
                        'vec_bolsa_llamamientos_desarrollo')
  LOOP
    IF NOT rol.rolcanlogin OR NOT rol.rolinherit
       OR rol.rolsuper OR rol.rolcreatedb OR rol.rolcreaterole
       OR rol.rolreplication OR rol.rolbypassrls
       OR rol.password_hash IS NOT NULL OR rol.rolconfig IS NOT NULL
       OR rol.rolvaliduntil IS NULL
       OR rol.rolvaliduntil < clock_timestamp() + INTERVAL '60 days'
       OR rol.rolvaliduntil > clock_timestamp() + INTERVAL '90 days'
       OR rol.admin_option OR NOT rol.inherit_option OR rol.set_option
       OR rol.roleid <> (CASE rol.rolname
           WHEN 'vec_ct_o207_runtime' THEN 'vec_contratacion_temporal_ejecutor'::regrole
           ELSE 'vec_bolsa_llamamientos_ejecutor'::regrole END)
    THEN RAISE EXCEPTION 'F0-29: rol o membresía fuera del contrato: %', rol.rolname
      USING ERRCODE = '55000';
    END IF;
  END LOOP;
  IF (SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid = m.member
      WHERE r.rolname IN ('vec_ct_o207_runtime',
                          'vec_bolsa_llamamientos_desarrollo')) <> 2
     OR (SELECT count(*) FROM pg_roles
         WHERE rolname IN ('vec_ct_o207_runtime',
                           'vec_bolsa_llamamientos_desarrollo')) <> 2
  THEN RAISE EXCEPTION 'F0-29: se esperaba una membresía por LOGIN' USING ERRCODE = '55000';
  END IF;
  IF NOT EXISTS (
       SELECT 1 FROM pg_roles r JOIN pg_authid a ON a.oid = r.oid
       WHERE r.rolname = 'vec_contexto_actor_v1_migrador'
         AND NOT (r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolcreatedb
                  OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
         AND a.rolpassword IS NULL AND r.rolconfig IS NULL
     ) OR EXISTS (
       SELECT 1 FROM pg_auth_members m
       WHERE m.member = 'vec_contexto_actor_v1_migrador'::regrole
          OR m.roleid = 'vec_contexto_actor_v1_migrador'::regrole
     )
  THEN RAISE EXCEPTION 'F0-29: migrador CTX45 fuera del contrato' USING ERRCODE = '55000';
  END IF;
END $post$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $post_filas$
BEGIN
  IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1) <> 3
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
       WHERE login_nombre = 'vec_ct_o207_runtime'
         AND audiencia_consumo = 'vec_contratacion_temporal.vinculo_emision_bolsa.v1'
         AND operacion = 'contratacion_temporal.bolsa.vincular'
         AND proceso = 'vec-server'
         AND canal_permitido = 'interna_corporativa')
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
       WHERE login_nombre = 'vec_bolsa_llamamientos_desarrollo'
         AND audiencia_consumo = 'vec_bolsa_llamamientos.llamamiento.emitir.v1'
         AND operacion = 'llamamiento.emitir.v1'
         AND proceso = 'bolsa_llamamientos_ci'
         AND canal_permitido = 'interna_corporativa')
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
       WHERE login_nombre = 'vec_bolsa_llamamientos_desarrollo'
         AND audiencia_consumo = 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
         AND operacion = 'bolsa.situacion_participacion.cambiar'
         AND proceso = 'vec-server'
         AND canal_permitido = 'interna_corporativa')
  THEN RAISE EXCEPTION 'F0-29: orígenes previos de AD278 incompatibles'
    USING ERRCODE = '55000';
  END IF;
END $post_filas$;
RESET ROLE;
INSERT INTO vec_documentos.tipo_reservado_firmado VALUES
 ('ref:f0074a505ef2a693dfd5bf2195228c47a7ce2a1435d4acf6a6b9f50e89e5663c',
  'contratacion_temporal.resolucion_firmada.v1','contratacion_temporal');

-- AD198/202/204/205/234/248: el volcado sin datos omite los conjuntos
-- cerrados 1–6. Son nombres técnicos de capacidades, sin claves ni permisos
-- de negocio; AD304 exige conservarlos antes de añadir el conjunto 7.
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1
 (version,audiencias,segmentos) VALUES
 (1,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote']::text[]),
 (2,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma']::text[]),
 (3,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial']::text[]),
 (4,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal']::text[]),
 (5,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1','vec_autorizacion.versionar_rol_bolsa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa.cierre.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal','perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre']::text[]),
 (6,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1','vec_autorizacion.versionar_rol_bolsa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa.cierre.v1','vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1']::text[],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal','perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre','perfiles:version-bolsa-externa:propuesta','perfiles:version-bolsa-externa:cierre']::text[]);
DO $conjuntos_admin_post$ BEGIN
 IF (SELECT array_agg(version ORDER BY version)
     FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)
      IS DISTINCT FROM ARRAY[1,2,3,4,5,6]
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version)
 THEN RAISE EXCEPTION 'F0-29: conjuntos técnicos incompatibles o claves publicadas'
   USING ERRCODE='55000'; END IF;
END $conjuntos_admin_post$;
RESET ROLE;
COMMIT;
