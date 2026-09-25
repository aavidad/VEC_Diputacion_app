\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('p6:retirada-dietas-r1d:20260923',0));
SELECT pg_advisory_xact_lock(hashtextextended('f4:reactivacion-dietas-r1d:20260924',0));

CREATE TEMP TABLE f4_plan (dato jsonb NOT NULL) ON COMMIT DROP;
INSERT INTO f4_plan VALUES (convert_from(decode(:'f4_plan_b64','base64'),'UTF8')::jsonb);
GRANT SELECT ON f4_plan TO vec_autorizacion_propietario;

-- Estado exacto necesario para esta vertical: cada objeto y cada ACL de la
-- lista debe existir. El fixture PG18 crea stubs solo en su base desechable.
CREATE TEMP TABLE f4_acl_permitida
 (grupo text, clase oid, objeto oid, columna integer, privilegio text, grantable boolean)
 ON COMMIT DROP;
INSERT INTO f4_acl_permitida VALUES
 ('vec_identidad_sesiones_v1_registrador','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_identidad_sesiones_v1_revalidador','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_contexto_actor_v1_runtime','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_autorizacion_fuente','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_autorizacion_registro','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_autorizacion_motivos_evaluador','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_dietas_ejecutor','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_identidad_sesiones_v1_registrador','pg_namespace'::regclass,to_regnamespace('vec_identidad_sesiones_v1'),0,'USAGE',false),
 ('vec_identidad_sesiones_v1_revalidador','pg_namespace'::regclass,to_regnamespace('vec_identidad_sesiones_v1'),0,'USAGE',false),
 ('vec_contexto_actor_v1_runtime','pg_namespace'::regclass,to_regnamespace('vec_contexto_actor_v1'),0,'USAGE',false),
 ('vec_autorizacion_fuente','pg_namespace'::regclass,to_regnamespace('vec_autorizacion'),0,'USAGE',false),
 ('vec_autorizacion_registro','pg_namespace'::regclass,to_regnamespace('vec_autorizacion'),0,'USAGE',false),
 ('vec_autorizacion_motivos_evaluador','pg_namespace'::regclass,to_regnamespace('vec_autorizacion'),0,'USAGE',false),
 ('vec_dietas_ejecutor','pg_namespace'::regclass,to_regnamespace('vec_dietas'),0,'USAGE',false),
 ('vec_dietas_ejecutor','pg_namespace'::regclass,to_regnamespace('vec_personal'),0,'USAGE',false),
 ('vec_dietas_ejecutor','pg_namespace'::regclass,to_regnamespace('vec_autorizacion_atestada_v3'),0,'USAGE',false),
 ('vec_dietas_ejecutor','pg_class'::regclass,to_regclass('vec_dietas.version_tarifa_provisional'),0,'SELECT',false),
 ('vec_dietas_ejecutor','pg_class'::regclass,to_regclass('vec_dietas.importe_dieta_provisional'),0,'SELECT',false),
 ('vec_dietas_ejecutor','pg_class'::regclass,to_regclass('vec_dietas.importe_km_provisional'),0,'SELECT',false),
 ('vec_identidad_sesiones_v1_registrador','pg_proc'::regclass,to_regprocedure('vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)'),0,'EXECUTE',false),
 ('vec_identidad_sesiones_v1_registrador','pg_proc'::regclass,to_regprocedure('vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)'),0,'EXECUTE',false),
 ('vec_identidad_sesiones_v1_revalidador','pg_proc'::regclass,to_regprocedure('vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)'),0,'EXECUTE',false),
 ('vec_identidad_sesiones_v1_revalidador','pg_proc'::regclass,to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'),0,'EXECUTE',false),
 ('vec_contexto_actor_v1_runtime','pg_proc'::regclass,to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'),0,'EXECUTE',false),
 ('vec_contexto_actor_v1_runtime','pg_proc'::regclass,to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),0,'EXECUTE',false),
 ('vec_contexto_actor_v1_runtime','pg_proc'::regclass,to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),0,'EXECUTE',false),
 ('vec_autorizacion_fuente','pg_proc'::regclass,to_regprocedure('vec_autorizacion.obtener_instantanea(text,text)'),0,'EXECUTE',false),
 ('vec_autorizacion_registro','pg_proc'::regclass,to_regprocedure('vec_autorizacion.registrar_decision_si_vigente(jsonb)'),0,'EXECUTE',false),
 ('vec_autorizacion_registro','pg_proc'::regclass,to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'),0,'EXECUTE',false),
 ('vec_autorizacion_registro','pg_proc'::regclass,to_regprocedure('vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)'),0,'EXECUTE',false),
 ('vec_autorizacion_motivos_evaluador','pg_proc'::regclass,to_regprocedure('vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)'),0,'EXECUTE',false),
 ('vec_autorizacion_motivos_evaluador','pg_proc'::regclass,to_regprocedure('vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamptz)'),0,'EXECUTE',false),
 ('vec_dietas_ejecutor','pg_proc'::regclass,to_regprocedure('vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_dietas_ejecutor','pg_proc'::regclass,to_regprocedure('vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_dietas_ejecutor','pg_proc'::regclass,to_regprocedure('vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_dietas_ejecutor','pg_proc'::regclass,to_regprocedure('vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_dietas_ejecutor','pg_proc'::regclass,to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false);

DO $pre$
DECLARE v jsonb; n integer;
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR current_setting('server_version_num')::integer >= 190000
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR session_user<>current_user OR inet_server_addr() IS NOT NULL
    OR current_database()<>'postgres' THEN
   RAISE EXCEPTION 'F4 requiere PostgreSQL 18, DBA y socket local de postgres' USING ERRCODE='42501';
 END IF;
 SELECT dato INTO STRICT v FROM f4_plan;
 IF v->>'actor'<>'administracion:f4:reactivacion-dietas-r1d'
    OR v->>'acto'<>'acto:f4:reactivacion-dietas-r1d:20260924'
    OR v->>'original_ref' !~ '^asignacion:dietas_r1d_[a-z0-9]+:v1$'
    OR v->>'anterior_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v2'
    OR v->>'nueva_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v3'
    OR v->>'original_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'anterior_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'nueva_huella' !~ '^[0-9a-f]{64}$'
    OR v->'documento'->>'estado'<>'activa'
    OR v->'documento'->>'version_rol_ref'<>'rol:dietas_r1d_provisional:v1'
    OR v->'documento'->>'emitida_por' IS DISTINCT FROM v->>'actor'
    OR v->'documento'->>'emitida_en' IS DISTINCT FROM v->'documento'->>'vigente_desde'
    OR (v->'documento' ? 'revocada_por') OR (v->'documento' ? 'revocacion_ref') THEN
   RAISE EXCEPTION 'plan F4 invalido' USING ERRCODE='55000';
 END IF;
 SELECT count(*) INTO n FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$';
 IF n<>8 THEN RAISE EXCEPTION 'inventario F4 requiere exactamente ocho cuentas; revisar novena auditoria' USING ERRCODE='55000'; END IF;
 IF EXISTS (
   WITH esperados(nombre,grupo) AS (VALUES
    ('vec_dietas_r1d_registro_identidad_desarrollo','vec_identidad_sesiones_v1_registrador'),
    ('vec_dietas_r1d_revalidacion_identidad_desarrollo','vec_identidad_sesiones_v1_revalidador'),
    ('vec_dietas_r1d_contexto_desarrollo','vec_contexto_actor_v1_runtime'),
    ('vec_dietas_r1d_fuente_autorizacion_desarrollo','vec_autorizacion_fuente'),
    ('vec_dietas_r1d_registro_autorizacion_desarrollo','vec_autorizacion_registro'),
    ('vec_dietas_r1d_motivos_desarrollo','vec_autorizacion_motivos_evaluador'),
    ('vec_dietas_r1d_dietas_desarrollo','vec_dietas_ejecutor'),
    ('vec_dietas_r1d_personal_desarrollo','vec_dietas_ejecutor')
   )
   SELECT 1 FROM esperados e
   LEFT JOIN pg_roles r ON r.rolname=e.nombre
   LEFT JOIN pg_roles g ON g.rolname=e.grupo
   WHERE r.oid IS NULL OR g.oid IS NULL OR r.rolcanlogin OR r.rolsuper OR r.rolcreatedb
      OR r.rolcreaterole OR NOT r.rolinherit OR r.rolreplication OR r.rolbypassrls
      OR r.rolconnlimit<>-1 OR r.rolvaliduntil IS NOT NULL
      OR NOT has_database_privilege(r.oid,current_database(),'CONNECT')
      OR has_database_privilege(r.oid,current_database(),'CREATE')
      OR has_database_privilege(r.oid,current_database(),'TEMP')
      OR g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
      OR g.rolreplication OR g.rolbypassrls
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid)<>1
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.roleid=r.oid)<>0
      -- Los grupos técnicos terminales no pueden heredar pg_read_all_data,
      -- un propietario u otro rol superior, ni permitir SET ROLE hacia él.
      OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
      OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                   AND d.refobjid=r.oid AND d.deptype='a')
      OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                   AND d.refobjid IN (r.oid,g.oid) AND d.deptype='o')
      OR NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
         WHERE m.member=r.oid AND g.rolname=e.grupo AND NOT m.admin_option
           AND m.inherit_option AND NOT m.set_option)
 ) THEN RAISE EXCEPTION 'cuentas F4 con atributos, ACL o membresias inesperadas' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_roles l JOIN pg_roles g
   ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera')
   WHERE l.rolcanlogin AND NOT l.rolsuper
     AND l.rolname NOT IN (
       'vec_dietas_r1d_registro_identidad_desarrollo',
       'vec_dietas_r1d_revalidacion_identidad_desarrollo',
       'vec_dietas_r1d_contexto_desarrollo',
       'vec_dietas_r1d_fuente_autorizacion_desarrollo',
       'vec_dietas_r1d_registro_autorizacion_desarrollo',
       'vec_dietas_r1d_motivos_desarrollo',
       'vec_dietas_r1d_dietas_desarrollo',
       'vec_dietas_r1d_personal_desarrollo')
     AND pg_has_role(l.oid,g.oid,'MEMBER')) THEN
   RAISE EXCEPTION 'LOGIN ajeno conserva ruta a grupo Dietas' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   SELECT 1 FROM f4_acl_permitida e
   WHERE e.objeto IS NULL OR NOT EXISTS (
     SELECT 1 FROM (
       SELECT g.rolname grupo,'pg_database'::regclass::oid clase,d.oid objeto,0 columna,
              a.privilege_type::text privilegio,a.is_grantable grantable
         FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
         JOIN pg_roles g ON g.oid=a.grantee
       UNION ALL
       SELECT g.rolname,'pg_namespace'::regclass::oid,n.oid,0,a.privilege_type::text,a.is_grantable
         FROM pg_namespace n,LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
         JOIN pg_roles g ON g.oid=a.grantee
       UNION ALL
       SELECT g.rolname,'pg_class'::regclass::oid,c.oid,0,a.privilege_type::text,a.is_grantable
         FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) a
         JOIN pg_roles g ON g.oid=a.grantee
       UNION ALL
       SELECT g.rolname,'pg_proc'::regclass::oid,p.oid,0,a.privilege_type::text,a.is_grantable
         FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         JOIN pg_roles g ON g.oid=a.grantee
     ) actual
     WHERE (actual.grupo,actual.clase,actual.objeto,actual.columna,
            actual.privilegio,actual.grantable)
        = (e.grupo,e.clase,e.objeto,e.columna,e.privilegio,e.grantable)
   )
 ) THEN
   RAISE EXCEPTION 'objeto o ACL requerida para vertical F4 ausente' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   WITH grupos AS (
     SELECT oid,rolname FROM pg_roles WHERE rolname IN (
       'vec_identidad_sesiones_v1_registrador','vec_identidad_sesiones_v1_revalidador',
       'vec_contexto_actor_v1_runtime','vec_autorizacion_fuente','vec_autorizacion_registro',
       'vec_autorizacion_motivos_evaluador','vec_dietas_ejecutor')
   ), observadas AS (
     SELECT g.rolname grupo,'pg_database'::regclass::oid clase,d.oid objeto,0 columna,
            a.privilege_type::text privilegio,a.is_grantable grantable
       FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
       JOIN grupos g ON g.oid=a.grantee
     UNION ALL
     SELECT g.rolname,'pg_namespace'::regclass::oid,n.oid,0,a.privilege_type::text,a.is_grantable
       FROM pg_namespace n,LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
       JOIN grupos g ON g.oid=a.grantee
     UNION ALL
     SELECT g.rolname,'pg_class'::regclass::oid,c.oid,0,a.privilege_type::text,a.is_grantable
       FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) a
       JOIN grupos g ON g.oid=a.grantee
     UNION ALL
     SELECT g.rolname,'pg_class'::regclass::oid,c.oid,x.attnum,a.privilege_type::text,a.is_grantable
       FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid,
            LATERAL aclexplode(x.attacl) a
       JOIN grupos g ON g.oid=a.grantee WHERE x.attnum>0
     UNION ALL
     SELECT g.rolname,'pg_proc'::regclass::oid,p.oid,0,a.privilege_type::text,a.is_grantable
       FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       JOIN grupos g ON g.oid=a.grantee
     UNION ALL
     SELECT g.rolname,'pg_type'::regclass::oid,t.oid,0,a.privilege_type::text,a.is_grantable
       FROM pg_type t,LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
       JOIN grupos g ON g.oid=a.grantee
   )
   SELECT 1 FROM (
     SELECT * FROM observadas
     EXCEPT
     SELECT grupo,clase,objeto,columna,privilegio,grantable
       FROM f4_acl_permitida WHERE objeto IS NOT NULL
   ) extras
 ) OR EXISTS (
   SELECT 1 FROM pg_shdepend d JOIN pg_roles g ON g.oid=d.refobjid
   WHERE g.rolname IN (
       'vec_identidad_sesiones_v1_registrador','vec_identidad_sesiones_v1_revalidador',
       'vec_contexto_actor_v1_runtime','vec_autorizacion_fuente','vec_autorizacion_registro',
       'vec_autorizacion_motivos_evaluador','vec_dietas_ejecutor')
     AND d.refclassid='pg_authid'::regclass AND d.deptype='a'
     AND (d.dbid NOT IN (0,(SELECT oid FROM pg_database WHERE datname=current_database()))
       OR d.classid NOT IN ('pg_database'::regclass,'pg_namespace'::regclass,
                           'pg_class'::regclass,'pg_proc'::regclass,'pg_type'::regclass))
 ) THEN
   RAISE EXCEPTION 'ACL directa de grupo fuera de preimagen F4 permitida' USING ERRCODE='55000';
 END IF;
 -- La preimagen permitida no concede derechos a PUBLIC en la base ni en
 -- esquemas/objetos VEC. Un GRANT PUBLIC reabriría acceso de cualquiera de
 -- estos LOGIN aunque su membresía nominal permanezca intacta.
 IF EXISTS (SELECT 1 FROM pg_database d,
      LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) acl
      WHERE d.datname=current_database() AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_namespace n,
      LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) acl
      WHERE left(n.nspname,4)='vec_' AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
      LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) acl
      WHERE left(n.nspname,4)='vec_' AND c.relkind IN ('r','p','v','m','S','f') AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
      LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
      WHERE left(n.nspname,4)='vec_' AND acl.grantee=0) THEN
   RAISE EXCEPTION 'ACL PUBLIC fuera de preimagen F4 permitida' USING ERRCODE='55000';
 END IF;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$') THEN
   RAISE EXCEPTION 'F4 requiere cero sesiones Dietas' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Bloquea la creación/avance concurrente de otro puntero Dietas antes de
-- comprobar la unicidad global y antes de habilitar los LOGIN.
LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE ROW EXCLUSIVE MODE;
DO $activar$
DECLARE v jsonb; anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
 original vec_autorizacion.asignacion_perfil%ROWTYPE;
 puntero vec_autorizacion.asignacion_perfil_actual%ROWTYPE;
 nueva jsonb; filas integer; punteros integer;
BEGIN
 SELECT dato INTO STRICT v FROM f4_plan;
 SELECT count(*) INTO punteros FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE r.rol_id='dietas_r1d_provisional';
 IF punteros<>1 THEN
   RAISE EXCEPTION 'F4 requiere un unico puntero Dietas global antes de LOGIN' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
  WHERE perfil_activo_ref=v->'documento'->>'perfil_activo_ref' FOR UPDATE;
 SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=puntero.asignacion_ref;
 SELECT * INTO STRICT original FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=v->>'original_ref';
 nueva:=v->'documento';
 IF puntero.asignacion_ref IS DISTINCT FROM v->>'anterior_ref'
    OR puntero.actualizada_por<>'administracion:p6:retirada-dietas-r1d'
    OR puntero.acto_ref<>'acto:p6:revocacion-dietas-r1d:20260923'
    OR original.version<>1 OR original.documento->>'estado'<>'activa'
    OR original.huella_sha256 IS DISTINCT FROM v->>'original_huella'
    OR anterior.version<>2 OR anterior.documento->>'estado'<>'revocada'
    OR anterior.documento->>'revocada_por'<>'administracion:p6:retirada-dietas-r1d'
    OR anterior.documento->>'revocacion_ref'<>'acto:p6:revocacion-dietas-r1d:20260923'
    OR anterior.huella_sha256 IS DISTINCT FROM v->>'anterior_huella'
    OR anterior.version_rol_ref<>'rol:dietas_r1d_provisional:v1'
    OR anterior.asignacion_id IS DISTINCT FROM original.asignacion_id
    OR anterior.principal_id IS DISTINCT FROM original.principal_id
    OR anterior.perfil_activo_ref IS DISTINCT FROM original.perfil_activo_ref
    OR anterior.documento->'ambitos' IS DISTINCT FROM original.documento->'ambitos'
    OR anterior.documento->>'vigente_desde' IS DISTINCT FROM original.documento->>'vigente_desde'
    OR anterior.documento->>'vigente_hasta' IS DISTINCT FROM original.documento->>'vigente_hasta'
    OR anterior.documento->>'emitida_por' IS DISTINCT FROM original.documento->>'emitida_por'
    OR anterior.documento->>'emitida_en' IS DISTINCT FROM original.documento->>'emitida_en'
    OR nueva->>'asignacion_id' IS DISTINCT FROM anterior.asignacion_id
    OR nueva->>'principal_id' IS DISTINCT FROM anterior.principal_id
    OR nueva->>'perfil_activo_ref' IS DISTINCT FROM anterior.perfil_activo_ref
    OR nueva->>'version_rol_ref' IS DISTINCT FROM anterior.version_rol_ref
    OR nueva->'ambitos' IS DISTINCT FROM anterior.documento->'ambitos'
    OR nueva->>'vigente_hasta' IS DISTINCT FROM anterior.documento->>'vigente_hasta'
    OR nueva->>'vigente_desde' IS DISTINCT FROM nueva->>'emitida_en'
    OR (nueva->>'vigente_desde')::timestamptz <= (anterior.documento->>'revocada_en')::timestamptz
    OR (nueva->>'vigente_desde')::timestamptz >= (nueva->>'vigente_hasta')::timestamptz
    OR clock_timestamp() >= (nueva->>'vigente_hasta')::timestamptz
    OR nueva->>'version'<>'3'
    OR (nueva-'version'-'estado'-'vigente_desde'-'emitida_por'-'emitida_en'-'revocada_por'-'revocada_en'-'revocacion_ref')
       IS DISTINCT FROM (anterior.documento-'version'-'estado'-'vigente_desde'-'emitida_por'-'emitida_en'-'revocada_por'-'revocada_en'-'revocacion_ref')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
      WHERE r.version_rol_ref='rol:dietas_r1d_provisional:v1' AND r.documento->>'estado'='publicada'
        AND c.documento->>'estado'='habilitada') THEN
   RAISE EXCEPTION 'preimagen, vigencia o control V3 F4 incompatibles' USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
   version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES (v->>'nueva_ref',anterior.asignacion_id,3,anterior.perfil_activo_ref,
  anterior.principal_id,anterior.version_rol_ref,v->>'nueva_huella',
  (nueva->>'emitida_en')::timestamptz,nueva);
 UPDATE vec_autorizacion.asignacion_perfil_actual
 SET asignacion_ref=v->>'nueva_ref',actualizada_en=clock_timestamp(),
     actualizada_por=v->>'actor',acto_ref=v->>'acto'
 WHERE perfil_activo_ref=puntero.perfil_activo_ref AND asignacion_ref=puntero.asignacion_ref;
 GET DIAGNOSTICS filas=ROW_COUNT;
 IF filas<>1 THEN RAISE EXCEPTION 'puntero F4 no avanzo' USING ERRCODE='55000'; END IF;
END $activar$;
RESET ROLE;

ALTER ROLE vec_dietas_r1d_registro_identidad_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_revalidacion_identidad_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_contexto_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_fuente_autorizacion_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_registro_autorizacion_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_motivos_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_dietas_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_personal_desarrollo LOGIN;

DO $post$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM f4_plan;
 PERFORM pg_stat_clear_snapshot();
 IF (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin)<>8
    OR EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
       JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
       WHERE p.asignacion_ref=v->>'nueva_ref' AND a.huella_sha256=v->>'nueva_huella'
         AND a.documento=v->'documento' AND a.documento->>'estado'='activa') THEN
   RAISE EXCEPTION 'postcondicion F4 fallida' USING ERRCODE='55000';
 END IF;
END $post$;

:f4_finalizar;
