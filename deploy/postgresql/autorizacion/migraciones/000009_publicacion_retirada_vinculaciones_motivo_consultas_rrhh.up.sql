-- Publicacion/retirada nominal; 000008 conserva publicaciones y 000009 sus eventos.
BEGIN;
SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL default_table_access_method='heap'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0)); SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0));
SET LOCAL ROLE vec_autorizacion_propietario;
LOCK TABLE vec_autorizacion.motivo_v2_evento_origen,
  vec_autorizacion.motivo_v2_catalogo_publicado, vec_autorizacion.motivo_v2_entrada,
  vec_autorizacion.motivo_v2_retirada, vec_autorizacion.motivo_v2_checkpoint_origen,
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1,
  vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
  IN ACCESS SHARE MODE;
DO $prevalidacion$ DECLARE v_presentes integer; v_bloqueo oid :=
      'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure;
    v_avance oid := 'vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()'::regprocedure;
    v_fundamentos regclass[] := ARRAY[
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
      'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,
      'vec_autorizacion.motivo_v2_evento_origen'::regclass, 'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,
      'vec_autorizacion.motivo_v2_entrada'::regclass, 'vec_autorizacion.motivo_v2_retirada'::regclass,
      'vec_autorizacion.motivo_v2_checkpoint_origen'::regclass];
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer / 10000 <> 18
       OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_autorizacion'
              AND nspowner = 'vec_autorizacion_propietario'::regrole
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_motivos_proyector'
              AND NOT rolcanlogin AND NOT rolsuper
              AND NOT rolcreaterole AND NOT rolcreatedb
              AND NOT rolinherit AND NOT rolreplication AND NOT rolbypassrls
       )
       OR pg_catalog.obj_description(
           'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
           'pg_class') IS DISTINCT FROM
          'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008'
       OR pg_catalog.obj_description(
           'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'
           ::regclass, 'pg_class') IS DISTINCT FROM
          'vec_autorizacion:vinculacion-motivo-consulta-rrhh:checkpoint-v1:000008'
    THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='fundamento 000008/V2 no exacto';
    END IF;
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class AS c
         WHERE c.oid=ANY(v_fundamentos)
           AND c.relkind='r' AND c.relpersistence='p'
           AND c.relam=(SELECT oid FROM pg_catalog.pg_am
                         WHERE amname='heap' AND amtype='t')
           AND NOT c.relispartition
           AND c.relowner='vec_autorizacion_propietario'::regrole
           AND c.relrowsecurity AND c.relforcerowsecurity) <> 7
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_inherits
            WHERE inhrelid=ANY(v_fundamentos) OR inhparent=ANY(v_fundamentos)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_rewrite
            WHERE ev_class=ANY(v_fundamentos)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
            CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) AS a
           WHERE c.oid=ANY(v_fundamentos) AND a.grantee<>c.relowner
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute AS a
            CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS permiso
           WHERE a.attrelid=ANY(v_fundamentos) AND a.attnum>0
             AND permiso.grantee<>'vec_autorizacion_propietario'::regrole
       )
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger AS t
            WHERE t.tgrelid=ANY(v_fundamentos[1:2]) AND NOT t.tgisinternal
              AND t.tgenabled='O' AND
                ROW(t.tgparentid,t.tgconstraint,t.tgconstrrelid,
                    t.tgconstrindid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
                    t.tgattr::text,pg_catalog.encode(t.tgargs,'hex'),
                    t.tgqual IS NULL,t.tgoldtable IS NULL,t.tgnewtable IS NULL)
                =ROW(0::oid,0::oid,0::oid,0::oid,false,false,0,'','',
                     true,true,true)
              AND ROW(t.tgname,t.tgtype,t.tgfoid) IN (
                ROW('vinculacion_motivo_rrhh_inmutable',27::smallint,v_bloqueo),
                ROW('vinculacion_motivo_rrhh_no_truncar',34::smallint,v_bloqueo),
                ROW('vinculacion_motivo_rrhh_checkpoint_avance',19::smallint,v_avance),
                ROW('vinculacion_motivo_rrhh_checkpoint_inmutable',15::smallint,v_bloqueo),
                ROW('vinculacion_motivo_rrhh_checkpoint_no_truncar',34::smallint,v_bloqueo)
              )) <> 5
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
            WHERE tgrelid=ANY(v_fundamentos[1:2])
              AND NOT tgisinternal) <> 5
       OR EXISTS (
           WITH esperado(tabla,nombre,tipo,definicion,marca) AS (VALUES
             ('vec_autorizacion.motivo_v2_evento_origen'::regclass,'motivo_v2_evento_coordenadas_unicas','u','UNIQUE (secuencia_origen, evento_origen_ref)',NULL::text),
             ('vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,'motivo_v2_catalogo_evento_fk','f','FOREIGN KEY (secuencia_origen, evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)',NULL::text),
             ('vec_autorizacion.motivo_v2_entrada'::regclass,'motivo_v2_entrada_catalogo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version)',NULL::text),
             ('vec_autorizacion.motivo_v2_retirada'::regclass,'motivo_v2_retirada_catalogo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version)',NULL::text),
             ('vec_autorizacion.motivo_v2_retirada'::regclass,'motivo_v2_retirada_evento_fk','f','FOREIGN KEY (secuencia_origen, evento_origen_ref) REFERENCES vec_autorizacion.motivo_v2_evento_origen(secuencia_origen, evento_origen_ref)',NULL::text),
             ('vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,'motivo_v2_catalogo_referencia_completa_unica','u','UNIQUE (catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)','vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008'),
             ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_publicacion_completa_unica','u','UNIQUE (clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)',NULL::text),
             ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_catalogo_completo_fk','f','FOREIGN KEY (catalogo_id, catalogo_version, catalogo_huella_sha256) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)',NULL::text),
             ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,'vinculacion_motivo_rrhh_entrada_fk','f','FOREIGN KEY (catalogo_id, catalogo_version, entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id, catalogo_version, entrada_clave)',NULL::text),
             ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,'vinculacion_motivo_rrhh_checkpoint_historia_fk','f','FOREIGN KEY (clase_consulta, ultima_publicacion_version, ultima_publicacion_ref, ultima_publicacion_huella_sha256) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)',NULL::text)
           ), actual AS (
             SELECT c.conrelid,c.conname::text,c.contype::text,
               pg_catalog.pg_get_constraintdef(c.oid,true),pg_catalog.obj_description(c.oid,'pg_constraint')
             FROM pg_catalog.pg_constraint c JOIN esperado e ON (e.tabla,e.nombre)=(c.conrelid,c.conname)
             WHERE c.convalidated AND NOT c.condeferrable AND NOT c.condeferred AND c.connoinherit
           ), claves AS (
             SELECT c.oid,c.conrelid,c.confrelid,c.conindid FROM pg_catalog.pg_constraint c
             JOIN esperado e ON (e.tabla,e.nombre)=(c.conrelid,c.conname) WHERE e.tipo='f'
           ), ri_esperado AS (
             SELECT c.oid,x.tabla,x.otra,c.conindid,x.tipo,'O',true,x.funcion,true FROM claves c
             CROSS JOIN LATERAL (VALUES (c.conrelid,c.confrelid,5::smallint,'RI_FKey_check_ins'),
               (c.conrelid,c.confrelid,17::smallint,'RI_FKey_check_upd'),
               (c.confrelid,c.conrelid,9::smallint,'RI_FKey_noaction_del'),
               (c.confrelid,c.conrelid,17::smallint,'RI_FKey_noaction_upd')) x(tabla,otra,tipo,funcion)
           ), ri_actual AS (
             SELECT t.tgconstraint,t.tgrelid,t.tgconstrrelid,t.tgconstrindid,t.tgtype,
               t.tgenabled::text,t.tgisinternal,p.proname::text,
               ROW(t.tgparentid,t.tgdeferrable,t.tginitdeferred,t.tgnargs,t.tgattr::text,
                   pg_catalog.encode(t.tgargs,'hex'),t.tgqual IS NULL,t.tgoldtable IS NULL,t.tgnewtable IS NULL)
                 =ROW(0::oid,false,false,0,'','',true,true,true)
             FROM pg_catalog.pg_trigger t JOIN pg_catalog.pg_proc p ON p.oid=t.tgfoid
             WHERE t.tgconstraint=ANY(ARRAY(SELECT oid FROM claves))
           ), diferencia AS ((SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
             UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)),
           diferencia_ri AS ((SELECT * FROM ri_esperado EXCEPT ALL SELECT * FROM ri_actual)
             UNION ALL (SELECT * FROM ri_actual EXCEPT ALL SELECT * FROM ri_esperado))
           SELECT 1 FROM diferencia UNION ALL SELECT 1 FROM diferencia_ri)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid=ANY(v_fundamentos)
              AND polname='acceso_propietario_exacto' AND polcmd='*'
              AND polpermissive AND polroles=ARRAY['vec_autorizacion_propietario'::regrole::oid]
              AND pg_catalog.pg_get_expr(polqual,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
              AND pg_catalog.pg_get_expr(polwithcheck,polrelid)='(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
          ) <> 7
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
            WHERE polrelid=ANY(v_fundamentos)) <> 7 THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='fundamentos degradados';
    END IF;
    SELECT pg_catalog.count(*) INTO v_presentes FROM pg_catalog.pg_proc AS p
     WHERE p.pronamespace='vec_autorizacion'::regnamespace AND p.proname=ANY(ARRAY[
           'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1', 'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
           'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1', 'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
           'publicar_vinculacion_motivo_cuadro_rrhh_v1', 'publicar_vinculacion_motivo_detalle_rrhh_v1',
           'retirar_vinculacion_motivo_cuadro_rrhh_v1', 'retirar_vinculacion_motivo_detalle_rrhh_v1'
       ]::name[]);
    IF pg_catalog.to_regclass('vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1')
       IS NOT NULL OR v_presentes<>0 THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='000009 no adopta objetos';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1)
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            WHERE ultima_publicacion_version=0 AND ultima_publicacion_ref IS NULL
              AND ultima_publicacion_huella_sha256 IS NULL)<>2 THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='evidencia previa sin prueba 000009';
    END IF;
END $prevalidacion$;
CREATE TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 (
    clase_consulta text NOT NULL,
    operacion text NOT NULL,
    publicacion_version bigint NOT NULL,
    evento_ref text NOT NULL,
    evento_huella_sha256 text NOT NULL,
    publicacion_ref text NOT NULL,
    publicacion_huella_sha256 text NOT NULL,
    ocurrida_en timestamptz(6) NOT NULL,
    actor_tecnico_ref text NOT NULL,
    prueba_vec_secuencia_origen bigint NOT NULL,
    prueba_vec_evento_origen_ref text NOT NULL,
    prueba_vec_evento_huella_sha256 text NOT NULL,
    prueba_vec_validada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    registrada_txid bigint NOT NULL DEFAULT pg_catalog.txid_current(),
    CONSTRAINT vinculacion_motivo_rrhh_evento_pk
        PRIMARY KEY (clase_consulta, operacion, publicacion_version),
    CONSTRAINT vinculacion_motivo_rrhh_evento_ref_unica UNIQUE (evento_ref),
    CONSTRAINT vinculacion_motivo_rrhh_evento_huella_unica
        UNIQUE (evento_huella_sha256),
    CONSTRAINT vinculacion_motivo_rrhh_evento_clase_cerrada
        CHECK (clase_consulta IN ('cuadro', 'detalle')),
    CONSTRAINT vinculacion_motivo_rrhh_evento_operacion_cerrada
        CHECK (operacion IN ('publicacion', 'retirada')),
    CONSTRAINT vinculacion_motivo_rrhh_evento_version_positiva
        CHECK (publicacion_version > 0),
    CONSTRAINT vinculacion_motivo_rrhh_evento_refs_validas CHECK (
        evento_ref ~ '^evento_vinculacion_motivo_rrhh_[0-9a-f]{32}$'
        AND publicacion_ref ~ '^publicacion_motivo_rrhh_[0-9a-f]{32}$'
        AND prueba_vec_evento_origen_ref ~ '^evento_[0-9a-f]{32}$'
    ),
    CONSTRAINT vinculacion_motivo_rrhh_evento_huellas_validas CHECK (
        evento_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND evento_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND publicacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND publicacion_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND prueba_vec_evento_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_vec_evento_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CONSTRAINT vinculacion_motivo_rrhh_evento_actor_tecnico
        CHECK (pg_catalog.char_length(actor_tecnico_ref) BETWEEN 1 AND 63),
    CONSTRAINT vinculacion_motivo_rrhh_evento_prueba_secuencia
        CHECK (prueba_vec_secuencia_origen > 0),
    CONSTRAINT vinculacion_motivo_rrhh_evento_instantes CHECK (
        pg_catalog.isfinite(ocurrida_en)
        AND pg_catalog.isfinite(prueba_vec_validada_en)
        AND pg_catalog.isfinite(registrada_en)
        AND ocurrida_en <= registrada_en
        AND prueba_vec_validada_en <= registrada_en
        AND extract(year FROM (ocurrida_en AT TIME ZONE 'UTC')) BETWEEN 1 AND 9999
        AND extract(year FROM (prueba_vec_validada_en AT TIME ZONE 'UTC'))
            BETWEEN 1 AND 9999
        AND extract(year FROM (registrada_en AT TIME ZONE 'UTC')) BETWEEN 1 AND 9999
    ),
    CONSTRAINT vinculacion_motivo_rrhh_evento_txid
        CHECK (registrada_txid > 0),
    CONSTRAINT vinculacion_motivo_rrhh_evento_publicacion_fk
      FOREIGN KEY (clase_consulta, publicacion_version, publicacion_ref,
                   publicacion_huella_sha256)
      REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
        (clase_consulta, publicacion_version, publicacion_ref,
         publicacion_huella_sha256),
    CONSTRAINT vinculacion_motivo_rrhh_evento_prueba_vec_fk FOREIGN KEY (
        prueba_vec_secuencia_origen, prueba_vec_evento_origen_ref
    ) REFERENCES vec_autorizacion.motivo_v2_evento_origen
        (secuencia_origen, evento_origen_ref)
);
COMMENT ON TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:evento-v1:000009';
CREATE FUNCTION vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $funcion$ BEGIN
    RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='eventos RRHH inmutables';
END $funcion$;
CREATE FUNCTION vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $funcion$ BEGIN
    IF NEW.actor_tecnico_ref IS DISTINCT FROM session_user::text
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
             JOIN vec_autorizacion.motivo_v2_catalogo_publicado AS c
               ON c.catalogo_id = h.catalogo_id
              AND c.catalogo_version = h.catalogo_version
              AND c.catalogo_huella_publicada_sha256 =
                  h.catalogo_huella_sha256
             JOIN vec_autorizacion.motivo_v2_entrada AS entrada
               ON entrada.catalogo_id = h.catalogo_id
              AND entrada.catalogo_version = h.catalogo_version
              AND entrada.entrada_clave = h.entrada_clave
             JOIN vec_autorizacion.motivo_v2_evento_origen AS prueba
               ON prueba.secuencia_origen = c.secuencia_origen
              AND prueba.evento_origen_ref = c.evento_origen_ref
            WHERE h.clase_consulta = NEW.clase_consulta
              AND h.publicacion_version = NEW.publicacion_version
              AND h.publicacion_ref = NEW.publicacion_ref
              AND h.publicacion_huella_sha256 =
                  NEW.publicacion_huella_sha256
              AND prueba.tipo_evento = 'publicacion'
              AND prueba.huella_evento_sha256 =
                  NEW.prueba_vec_evento_huella_sha256
              AND prueba.secuencia_origen =
                  NEW.prueba_vec_secuencia_origen
              AND prueba.evento_origen_ref =
                  NEW.prueba_vec_evento_origen_ref
       )
       OR (
           NEW.operacion = 'publicacion'
           AND (
               NEW.prueba_vec_validada_en < NEW.ocurrida_en
               OR NOT EXISTS (
                   SELECT 1
                     FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
                    WHERE h.clase_consulta = NEW.clase_consulta
                      AND h.publicacion_version = NEW.publicacion_version
                      AND h.publicacion_ref = NEW.publicacion_ref
                      AND h.publicacion_huella_sha256 =
                          NEW.publicacion_huella_sha256
                      AND h.registrada_txid = pg_catalog.txid_current()
               )
           )
       )
       OR (
           NEW.operacion = 'retirada'
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS p
                WHERE p.clase_consulta = NEW.clase_consulta
                  AND p.operacion = 'publicacion'
                  AND p.publicacion_version = NEW.publicacion_version
                  AND p.publicacion_ref = NEW.publicacion_ref
                  AND p.publicacion_huella_sha256 =
                      NEW.publicacion_huella_sha256
                  AND p.ocurrida_en <= NEW.ocurrida_en
                  AND p.prueba_vec_secuencia_origen =
                      NEW.prueba_vec_secuencia_origen
                  AND p.prueba_vec_evento_origen_ref =
                      NEW.prueba_vec_evento_origen_ref
                  AND p.prueba_vec_evento_huella_sha256 =
                      NEW.prueba_vec_evento_huella_sha256
                  AND p.prueba_vec_validada_en =
                      NEW.prueba_vec_validada_en
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE='23514', MESSAGE='actor o prueba RRHH no exactos';
    END IF;
    RETURN NEW;
END $funcion$;
CREATE TRIGGER vinculacion_motivo_rrhh_evento_validar
    BEFORE INSERT ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion
      .validar_insercion_vinculacion_motivo_rrhh_evento_v1();
CREATE TRIGGER vinculacion_motivo_rrhh_evento_inmutable
    BEFORE UPDATE OR DELETE ON vec_autorizacion
      .vinculacion_motivo_consulta_rrhh_evento_v1
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion
      .bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1();
CREATE TRIGGER vinculacion_motivo_rrhh_evento_no_truncar
    BEFORE TRUNCATE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion
      .bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1();
CREATE FUNCTION vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
    p_clase text, p_evento_ref text, p_evento_huella_sha256 text,
    p_publicacion_version bigint, p_publicacion_ref text,
    p_publicacion_huella_sha256 text, p_catalogo_id text,
    p_catalogo_version integer, p_catalogo_huella_sha256 text,
    p_entrada_clave text, p_ocurrida_en timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $funcion$ DECLARE
    v_actor text := session_user::text;
    v_checkpoint record;
    v_prueba record;
    v_validada_en timestamptz(6);
    v_colisiones integer;
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_clase NOT IN ('cuadro', 'detalle')
       OR (p_evento_ref ~
           '^evento_vinculacion_motivo_rrhh_[0-9a-f]{32}$') IS NOT TRUE
       OR (p_evento_huella_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_evento_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_publicacion_version IS NULL OR p_publicacion_version < 1
       OR (p_publicacion_ref ~
           '^publicacion_motivo_rrhh_[0-9a-f]{32}$') IS NOT TRUE
       OR (p_publicacion_huella_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_publicacion_huella_sha256 = pg_catalog.repeat('0', 64)
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL OR p_catalogo_version < 1
       OR (p_catalogo_huella_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_catalogo_huella_sha256 = pg_catalog.repeat('0', 64)
       OR (p_entrada_clave ~ '^motivo_[0-9a-f]{32}$') IS NOT TRUE
       OR p_ocurrida_en IS NULL
       OR pg_catalog.isfinite(p_ocurrida_en) IS NOT TRUE
       OR pg_catalog.date_trunc('microseconds',p_ocurrida_en)<>p_ocurrida_en
       OR p_ocurrida_en > pg_catalog.clock_timestamp()
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles AS r
             JOIN pg_catalog.pg_auth_members AS m ON m.member = r.oid
            WHERE r.rolname = v_actor
              AND r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreaterole
              AND NOT r.rolcreatedb AND NOT r.rolreplication
              AND NOT r.rolbypassrls
              AND m.roleid='vec_autorizacion_motivos_proyector'::regrole
              AND NOT m.admin_option AND m.inherit_option
              AND NOT m.set_option
       ) THEN
        RETURN false;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0));
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id
     FOR SHARE;
    IF NOT FOUND THEN RETURN false; END IF;
    PERFORM 1
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
     WHERE clase_consulta IN ('cuadro','detalle')
     ORDER BY clase_consulta
     FOR UPDATE;
    SELECT * INTO STRICT v_checkpoint
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
     WHERE clase_consulta = p_clase;
    SELECT pg_catalog.count(*) INTO v_colisiones
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
     WHERE (e.clase_consulta = p_clase
            AND e.operacion = 'publicacion'
            AND e.publicacion_version = p_publicacion_version)
        OR e.evento_ref = p_evento_ref
        OR e.evento_huella_sha256 = p_evento_huella_sha256;
    IF v_colisiones > 0 THEN
        IF v_colisiones = 1 AND EXISTS (
            SELECT 1
              FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
              JOIN vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
                ON h.clase_consulta = e.clase_consulta
               AND h.publicacion_version = e.publicacion_version
               AND h.publicacion_ref = e.publicacion_ref
               AND h.publicacion_huella_sha256 =
                   e.publicacion_huella_sha256
             WHERE e.clase_consulta = p_clase
               AND e.operacion = 'publicacion'
               AND e.publicacion_version = p_publicacion_version
               AND e.evento_ref = p_evento_ref
               AND e.evento_huella_sha256 = p_evento_huella_sha256
               AND e.publicacion_ref = p_publicacion_ref
               AND e.publicacion_huella_sha256 =
                   p_publicacion_huella_sha256
               AND e.ocurrida_en = p_ocurrida_en
               AND h.catalogo_id = p_catalogo_id
               AND h.catalogo_version = p_catalogo_version
               AND h.catalogo_huella_sha256 =
                   p_catalogo_huella_sha256
               AND h.entrada_clave = p_entrada_clave
        ) THEN
            RETURN true;
        END IF;
        RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='colision de identidad RRHH';
    END IF;
    IF p_publicacion_version IS DISTINCT FROM
          v_checkpoint.ultima_publicacion_version + 1
       OR (
           v_checkpoint.ultima_publicacion_version > 0
           AND (
               NOT EXISTS (
                   SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
                    WHERE e.clase_consulta = p_clase
                      AND e.operacion = 'publicacion'
                      AND e.publicacion_version =
                          v_checkpoint.ultima_publicacion_version
                      AND e.publicacion_ref =
                          v_checkpoint.ultima_publicacion_ref
                      AND e.publicacion_huella_sha256 =
                          v_checkpoint.ultima_publicacion_huella_sha256
                      AND e.ocurrida_en <= p_ocurrida_en
               )
               OR NOT EXISTS (
                   SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
                    WHERE e.clase_consulta = p_clase
                      AND e.operacion = 'retirada'
                      AND e.publicacion_version =
                          v_checkpoint.ultima_publicacion_version
                      AND e.publicacion_ref =
                          v_checkpoint.ultima_publicacion_ref
                      AND e.publicacion_huella_sha256 =
                          v_checkpoint.ultima_publicacion_huella_sha256
                      AND e.ocurrida_en <= p_ocurrida_en
               )
           )
       ) THEN
        RETURN false;
    END IF;
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
         WHERE h.clase_consulta<>p_clase
           AND (h.catalogo_id,h.catalogo_version,h.catalogo_huella_sha256,
                h.entrada_clave)=(p_catalogo_id,p_catalogo_version,
                                  p_catalogo_huella_sha256,p_entrada_clave)
           AND NOT EXISTS (
               SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS r
                WHERE r.clase_consulta=h.clase_consulta
                  AND r.operacion='retirada'
                  AND r.publicacion_version=h.publicacion_version
                  AND r.ocurrida_en<=p_ocurrida_en)
    ) THEN RETURN false; END IF;
    v_validada_en := pg_catalog.clock_timestamp();
    SELECT prueba.secuencia_origen, prueba.evento_origen_ref,
           prueba.huella_evento_sha256
      INTO v_prueba
      FROM vec_autorizacion.motivo_v2_catalogo_publicado AS c
      JOIN vec_autorizacion.motivo_v2_entrada AS entrada
        ON entrada.catalogo_id = c.catalogo_id
       AND entrada.catalogo_version = c.catalogo_version
      JOIN vec_autorizacion.motivo_v2_evento_origen AS prueba
        ON prueba.secuencia_origen = c.secuencia_origen
       AND prueba.evento_origen_ref = c.evento_origen_ref
     WHERE c.catalogo_id = p_catalogo_id
       AND c.catalogo_version = p_catalogo_version
       AND c.catalogo_huella_publicada_sha256 =
           p_catalogo_huella_sha256
       AND entrada.entrada_clave = p_entrada_clave
       AND prueba.tipo_evento = 'publicacion'
       AND prueba.catalogo_id = c.catalogo_id
       AND prueba.catalogo_version = c.catalogo_version
       AND prueba.huella_evento_sha256 <> pg_catalog.repeat('0', 64)
       AND c.publicado_en <= p_ocurrida_en
       AND entrada.vigente_desde <= p_ocurrida_en
       AND (entrada.vigente_hasta IS NULL
            OR p_ocurrida_en < entrada.vigente_hasta)
       AND c.publicado_en <= v_validada_en
       AND entrada.vigente_desde <= v_validada_en
       AND (entrada.vigente_hasta IS NULL
            OR v_validada_en < entrada.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1 FROM vec_autorizacion.motivo_v2_retirada AS r
            WHERE r.catalogo_id = c.catalogo_id
              AND r.catalogo_version = c.catalogo_version
       );
    IF NOT FOUND THEN RETURN false; END IF;
    INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
        clase_consulta, publicacion_version, publicacion_ref,
        publicacion_huella_sha256, catalogo_id, catalogo_version,
        catalogo_huella_sha256, entrada_clave, publicada_en
    ) VALUES (
        p_clase, p_publicacion_version, p_publicacion_ref,
        p_publicacion_huella_sha256, p_catalogo_id, p_catalogo_version,
        p_catalogo_huella_sha256, p_entrada_clave, p_ocurrida_en
    );
    INSERT INTO
      vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 (
        clase_consulta, operacion, publicacion_version,
        evento_ref, evento_huella_sha256,
        publicacion_ref, publicacion_huella_sha256, ocurrida_en,
        actor_tecnico_ref, prueba_vec_secuencia_origen,
        prueba_vec_evento_origen_ref,
        prueba_vec_evento_huella_sha256, prueba_vec_validada_en
    ) VALUES (
        p_clase, 'publicacion', p_publicacion_version,
        p_evento_ref, p_evento_huella_sha256,
        p_publicacion_ref, p_publicacion_huella_sha256, p_ocurrida_en,
        v_actor, v_prueba.secuencia_origen,
        v_prueba.evento_origen_ref, v_prueba.huella_evento_sha256,
        v_validada_en
    );
    UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
       SET ultima_publicacion_version = p_publicacion_version,
           ultima_publicacion_ref = p_publicacion_ref,
           ultima_publicacion_huella_sha256 =
               p_publicacion_huella_sha256,
           actualizado_en = pg_catalog.clock_timestamp()
     WHERE clase_consulta = p_clase;
    RETURN true;
EXCEPTION WHEN unique_violation THEN
    RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='colision de identidad RRHH';
WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR no_data_found OR too_many_rows THEN RETURN false;
END $funcion$;
CREATE FUNCTION vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
    p_clase text, p_evento_ref text, p_evento_huella_sha256 text,
    p_publicacion_version bigint, p_publicacion_ref text,
    p_publicacion_huella_sha256 text, p_ocurrida_en timestamptz
)
RETURNS boolean LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $funcion$ DECLARE
    v_actor text := session_user::text;
    v_checkpoint record;
    v_publicacion record;
    v_colisiones integer;
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_clase NOT IN ('cuadro', 'detalle')
       OR (p_evento_ref ~
           '^evento_vinculacion_motivo_rrhh_[0-9a-f]{32}$') IS NOT TRUE
       OR (p_evento_huella_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_evento_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_publicacion_version IS NULL OR p_publicacion_version < 1
       OR (p_publicacion_ref ~
           '^publicacion_motivo_rrhh_[0-9a-f]{32}$') IS NOT TRUE
       OR (p_publicacion_huella_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR p_publicacion_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_ocurrida_en IS NULL
       OR pg_catalog.isfinite(p_ocurrida_en) IS NOT TRUE
       OR pg_catalog.date_trunc('microseconds',p_ocurrida_en)<>p_ocurrida_en
       OR p_ocurrida_en > pg_catalog.clock_timestamp()
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles AS r
             JOIN pg_catalog.pg_auth_members AS m ON m.member = r.oid
            WHERE r.rolname = v_actor
              AND r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreaterole
              AND NOT r.rolcreatedb AND NOT r.rolreplication
              AND NOT r.rolbypassrls
              AND m.roleid='vec_autorizacion_motivos_proyector'::regrole
              AND NOT m.admin_option AND m.inherit_option
              AND NOT m.set_option
       ) THEN
        RETURN false;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0));
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0));
    SELECT * INTO STRICT v_checkpoint
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
     WHERE clase_consulta = p_clase
     FOR UPDATE;
    SELECT pg_catalog.count(*) INTO v_colisiones
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
     WHERE (e.clase_consulta = p_clase
            AND e.operacion = 'retirada'
            AND e.publicacion_version = p_publicacion_version)
        OR e.evento_ref = p_evento_ref
        OR e.evento_huella_sha256 = p_evento_huella_sha256;
    IF v_colisiones > 0 THEN
        IF v_colisiones = 1 AND EXISTS (
            SELECT 1
              FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
             WHERE e.clase_consulta = p_clase
               AND e.operacion = 'retirada'
               AND e.publicacion_version = p_publicacion_version
               AND e.evento_ref = p_evento_ref
               AND e.evento_huella_sha256 = p_evento_huella_sha256
               AND e.publicacion_ref = p_publicacion_ref
               AND e.publicacion_huella_sha256 =
                   p_publicacion_huella_sha256
               AND e.ocurrida_en = p_ocurrida_en
        ) THEN
            RETURN true;
        END IF;
        RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='colision de identidad RRHH';
    END IF;
    IF v_checkpoint.ultima_publicacion_version IS DISTINCT FROM
          p_publicacion_version
       OR v_checkpoint.ultima_publicacion_ref IS DISTINCT FROM
          p_publicacion_ref
       OR v_checkpoint.ultima_publicacion_huella_sha256 IS DISTINCT FROM
          p_publicacion_huella_sha256
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
            WHERE e.clase_consulta = p_clase
              AND e.operacion = 'retirada'
              AND e.publicacion_version = p_publicacion_version
       ) THEN
        RETURN false;
    END IF;
    SELECT * INTO v_publicacion
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 AS e
     WHERE e.clase_consulta = p_clase
       AND e.operacion = 'publicacion'
       AND e.publicacion_version = p_publicacion_version
       AND e.publicacion_ref = p_publicacion_ref
       AND e.publicacion_huella_sha256 = p_publicacion_huella_sha256;
    IF NOT FOUND OR p_ocurrida_en < v_publicacion.ocurrida_en THEN
        RETURN false;
    END IF;
    INSERT INTO
      vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 (
        clase_consulta, operacion, publicacion_version,
        evento_ref, evento_huella_sha256,
        publicacion_ref, publicacion_huella_sha256, ocurrida_en,
        actor_tecnico_ref, prueba_vec_secuencia_origen,
        prueba_vec_evento_origen_ref,
        prueba_vec_evento_huella_sha256, prueba_vec_validada_en
    ) VALUES (
        p_clase, 'retirada', p_publicacion_version,
        p_evento_ref, p_evento_huella_sha256,
        p_publicacion_ref, p_publicacion_huella_sha256, p_ocurrida_en,
        v_actor, v_publicacion.prueba_vec_secuencia_origen,
        v_publicacion.prueba_vec_evento_origen_ref,
        v_publicacion.prueba_vec_evento_huella_sha256,
        v_publicacion.prueba_vec_validada_en
    );
    RETURN true;
EXCEPTION WHEN unique_violation THEN
    RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='colision de identidad RRHH';
WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR no_data_found OR too_many_rows THEN RETURN false;
END $funcion$;
CREATE FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
    text, text, bigint, text, text, text, integer, text, text, timestamptz)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
SELECT vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
    'cuadro', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10) $funcion$;
CREATE FUNCTION vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
    text, text, bigint, text, text, text, integer, text, text, timestamptz)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
SELECT vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
    'detalle', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10) $funcion$;
CREATE FUNCTION vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
    text, text, bigint, text, text, timestamptz)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
SELECT vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
    'cuadro', $1, $2, $3, $4, $5, $6) $funcion$;
CREATE FUNCTION vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
    text, text, bigint, text, text, timestamptz)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
SELECT vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
    'detalle', $1, $2, $3, $4, $5, $6) $funcion$;
COMMENT ON FUNCTION vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
    text,text,text,bigint,text,text,text,integer,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:kernel-publicacion-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
    text,text,text,bigint,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:kernel-retirada-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1() IS
 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:bloqueo-evento-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1() IS
 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:validacion-evento-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
    text,text,bigint,text,text,text,integer,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:publicar-cuadro-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
    text,text,bigint,text,text,text,integer,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:publicar-detalle-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
    text,text,bigint,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:retirar-cuadro-v1:000009';
COMMENT ON FUNCTION vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
    text,text,bigint,text,text,timestamptz)
IS 'vec_autorizacion:vinculacion-motivo-consulta-rrhh:retirar-detalle-v1:000009';
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto ON
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    FOR ALL TO vec_autorizacion_propietario
    USING (current_user = 'vec_autorizacion_propietario')
    WITH CHECK (current_user = 'vec_autorizacion_propietario');
REVOKE ALL ON TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
REVOKE ALL ON FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1(),
    vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1(),
    vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(
      text,text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(
      text,text,text,bigint,text,text,timestamptz)
    FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
REVOKE ALL ON FUNCTION
    vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,timestamptz)
    FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,text,integer,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(
      text,text,bigint,text,text,timestamptz),
    vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
      text,text,bigint,text,text,timestamptz)
    TO vec_autorizacion_motivos_proyector;
REVOKE ALL ON TYPE vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro,
     vec_autorizacion_motivos_proyector, vec_autorizacion_motivos_evaluador;
DO $acl_exacta$ DECLARE v_tabla regclass :=
  'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass;
    v_funciones name[] := ARRAY[
      'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1', 'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
      'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1', 'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
      'publicar_vinculacion_motivo_cuadro_rrhh_v1', 'publicar_vinculacion_motivo_detalle_rrhh_v1',
      'retirar_vinculacion_motivo_cuadro_rrhh_v1', 'retirar_vinculacion_motivo_detalle_rrhh_v1'];
BEGIN
    IF (SELECT (pg_catalog.count(*)=8 AND pg_catalog.bool_and(
                 a.grantee=c.relowner AND NOT a.is_grantable
                 AND a.privilege_type=ANY(ARRAY[
                   'INSERT','SELECT','UPDATE','DELETE','TRUNCATE',
                   'REFERENCES','TRIGGER','MAINTAIN']))) IS TRUE
          FROM pg_catalog.pg_class AS c
          CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
         WHERE c.oid=v_tabla) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_attribute AS a
            CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl)
           WHERE a.attrelid=v_tabla AND a.attnum>0)
       OR (SELECT (pg_catalog.count(*)=1 AND pg_catalog.bool_and(
                    a.grantee=t.typowner AND a.privilege_type='USAGE'
                    AND NOT a.is_grantable)) IS TRUE
             FROM pg_catalog.pg_type AS t
            CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
            WHERE t.typrelid=v_tabla) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_type AS base
             JOIN pg_catalog.pg_type AS vector ON vector.oid=base.typarray
            WHERE base.typrelid=v_tabla
              AND (vector.typelem<>base.oid OR vector.typacl IS NOT NULL))
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc
            WHERE pronamespace='vec_autorizacion'::regnamespace
              AND proname=ANY(v_funciones)) <> 8
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
            CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
           WHERE p.pronamespace='vec_autorizacion'::regnamespace
             AND p.proname=ANY(v_funciones)
             AND a.privilege_type='EXECUTE' AND NOT a.is_grantable
             AND (a.grantee=p.proowner OR
                  (p.prosecdef AND a.grantee=
                   'vec_autorizacion_motivos_proyector'::regrole))) <> 12
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
            CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
           WHERE p.pronamespace='vec_autorizacion'::regnamespace
             AND p.proname=ANY(v_funciones)
             AND NOT (a.privilege_type='EXECUTE' AND NOT a.is_grantable
               AND (a.grantee=p.proowner OR (p.prosecdef AND a.grantee=
                 'vec_autorizacion_motivos_proyector'::regrole)))) THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='ACL 000009 no exactas';
    END IF;
END $acl_exacta$; COMMIT;
