-- F4b activación. Precedido por las variables del ejecutor y por comun.sql
-- (que abre la transacción). Termina con :f4b_finalizar (ROLLBACK o COMMIT).
CREATE TEMP TABLE f4b_plan (dato jsonb NOT NULL) ON COMMIT DROP;
INSERT INTO f4b_plan VALUES (convert_from(decode(:'f4b_plan_b64','base64'),'UTF8')::jsonb);
GRANT SELECT ON f4b_plan TO vec_autorizacion_propietario;
-- Verificadores SCRAM (nunca la contraseña) de las tres cuentas nuevas,
-- derivados por el ejecutor desde el estado privado 0600 y recibidos por stdin.
CREATE TEMP TABLE f4b_verificador (nombre text PRIMARY KEY, verificador text NOT NULL) ON COMMIT DROP;
INSERT INTO f4b_verificador
SELECT key, value FROM jsonb_each_text(convert_from(decode(:'f4b_verificadores_b64','base64'),'UTF8')::jsonb);

-- Lista positiva mínima: lo que la composición actual llama con cada grupo
-- tras Dietas 000001-000011, AD3-50/81 y Personal 000012/000013, más las
-- fachadas que acredita al arrancar con las tres cuentas nuevas. Frente a F4:
-- - el ejecutor crea comisiones con crear_o_recuperar_comision_catalogada_v2
--   (Dietas 000006 le revoca crear_o_recuperar_comision_calculada_v1);
-- - leer_concesion_historica_contexto_actor_v3 (autorización 000011) solo la
--   usa Contratación temporal, no Dietas: no se exige;
-- - resolver_motivo_cobertura_historico_v1 solo lo usa Contratación temporal;
-- - el USAGE de vec_autorizacion_atestada_v3 del ejecutor lo concede AD3-81
--   (la fachada de rutas de AD3-50 es inalcanzable sin él).
CREATE TEMP TABLE f4b_acl_requerida (grupo text, clase text, objeto text, privilegio text) ON COMMIT DROP;
INSERT INTO f4b_acl_requerida
SELECT DISTINCT c.grupo,'base',current_database(),'CONNECT' FROM f4b_cuenta c;
INSERT INTO f4b_acl_requerida VALUES
 ('vec_identidad_sesiones_v1_registrador','esquema','vec_identidad_sesiones_v1','USAGE'),
 ('vec_identidad_sesiones_v1_revalidador','esquema','vec_identidad_sesiones_v1','USAGE'),
 ('vec_contexto_actor_v1_runtime','esquema','vec_contexto_actor_v1','USAGE'),
 ('vec_autorizacion_fuente','esquema','vec_autorizacion','USAGE'),
 ('vec_autorizacion_registro','esquema','vec_autorizacion','USAGE'),
 ('vec_autorizacion_motivos_evaluador','esquema','vec_autorizacion','USAGE'),
 ('vec_dietas_ejecutor','esquema','vec_dietas','USAGE'),
 ('vec_dietas_ejecutor','esquema','vec_personal','USAGE'),
 ('vec_dietas_ejecutor','esquema','vec_autorizacion_atestada_v3','USAGE'),
 ('vec_dietas_registrador_frontera','esquema','vec_dietas','USAGE'),
 ('vec_personal_d7_ejecutor','esquema','vec_personal','USAGE'),
 ('vec_personal_registrador_frontera','esquema','vec_personal','USAGE'),
 ('vec_dietas_ejecutor','relacion','vec_dietas.version_tarifa_provisional','SELECT'),
 ('vec_dietas_ejecutor','relacion','vec_dietas.importe_dieta_provisional','SELECT'),
 ('vec_dietas_ejecutor','relacion','vec_dietas.importe_km_provisional','SELECT'),
 ('vec_identidad_sesiones_v1_registrador','funcion','vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)','EXECUTE'),
 ('vec_identidad_sesiones_v1_registrador','funcion','vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)','EXECUTE'),
 ('vec_identidad_sesiones_v1_revalidador','funcion','vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)','EXECUTE'),
 ('vec_identidad_sesiones_v1_revalidador','funcion','vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)','EXECUTE'),
 ('vec_contexto_actor_v1_runtime','funcion','vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()','EXECUTE'),
 ('vec_contexto_actor_v1_runtime','funcion','vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)','EXECUTE'),
 ('vec_contexto_actor_v1_runtime','funcion','vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)','EXECUTE'),
 ('vec_autorizacion_fuente','funcion','vec_autorizacion.obtener_instantanea(text,text)','EXECUTE'),
 ('vec_autorizacion_registro','funcion','vec_autorizacion.registrar_decision_si_vigente(jsonb)','EXECUTE'),
 ('vec_autorizacion_registro','funcion','vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)','EXECUTE'),
 ('vec_autorizacion_motivos_evaluador','funcion','vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.crear_o_recuperar_comision_catalogada_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.consultar_comisiones_propias_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.mutar_comision_propia_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.recuperar_mutacion_por_clave_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_ejecutor','funcion','vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_dietas_registrador_frontera','funcion','vec_dietas.registrar_auditoria_frontera_comision_v2(text,text,text,text,text,text)','EXECUTE'),
 ('vec_personal_d7_ejecutor','funcion','vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_personal_d7_ejecutor','funcion','vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_personal_d7_ejecutor','funcion','vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_personal_d7_ejecutor','funcion','vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
 ('vec_personal_registrador_frontera','funcion','vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)','EXECUTE');

-- Personal 000012 crea el grupo D7 sin CONNECT; el paquete D7 lo concedía al
-- preparar. F4b lo concede aquí (idempotente) antes de las comprobaciones.
DO $conectar$
BEGIN
 IF to_regrole('vec_personal_d7_ejecutor') IS NOT NULL
    AND NOT has_database_privilege('vec_personal_d7_ejecutor',current_database(),'CONNECT') THEN
   EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_personal_d7_ejecutor',current_database());
 END IF;
END $conectar$;

DO $pre$
DECLARE v jsonb; n integer;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_plan;
 IF v->>'actor'<>'administracion:f4b:acceso-dietas'
    OR v->>'acto'<>'acto:f4b:acceso-dietas:20260925'
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
   RAISE EXCEPTION 'plan F4b invalido' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM f4b_verificador)<>3
    OR EXISTS (SELECT 1 FROM f4b_cuenta c WHERE c.nueva
               AND NOT EXISTS (SELECT 1 FROM f4b_verificador x WHERE x.nombre=c.nombre))
    OR EXISTS (SELECT 1 FROM f4b_verificador x
               WHERE x.verificador !~ '^SCRAM-SHA-256\$4096:[A-Za-z0-9+/]{22}==\$[A-Za-z0-9+/]{43}=:[A-Za-z0-9+/]{43}=$') THEN
   RAISE EXCEPTION 'verificadores SCRAM F4b ausentes o invalidos' USING ERRCODE='55000';
 END IF;
 SELECT count(*) INTO n FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$';
 IF n<>8 THEN RAISE EXCEPTION 'F4b requiere exactamente las ocho cuentas R1D; revisar cuenta ajena con ese prefijo' USING ERRCODE='55000'; END IF;
 IF EXISTS (
   SELECT 1 FROM f4b_cuenta c
   LEFT JOIN pg_authid r ON r.rolname=c.nombre
   LEFT JOIN pg_roles g ON g.rolname=c.grupo
   WHERE g.oid IS NULL
      OR (r.oid IS NULL AND NOT c.nueva)
      OR (r.oid IS NOT NULL AND (
         r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR NOT r.rolinherit
         OR r.rolreplication OR r.rolbypassrls OR r.rolconnlimit<>-1 OR r.rolvaliduntil IS NOT NULL
         OR (NOT c.nueva AND r.rolpassword NOT LIKE 'SCRAM-SHA-256$%')
         OR (NOT c.nueva AND r.rolpassword IS NULL)
         OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid)<>1
         OR (SELECT count(*) FROM pg_auth_members m WHERE m.roleid=r.oid)<>0
         OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=r.oid AND m.roleid=g.oid
                          AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
         OR has_database_privilege(r.oid,current_database(),'CREATE')
         OR has_database_privilege(r.oid,current_database(),'TEMP')
         OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                      AND d.refobjid=r.oid AND d.deptype IN ('a','o'))))
      OR g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
      OR g.rolreplication OR g.rolbypassrls
      OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
      OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                   AND d.refobjid=g.oid AND d.deptype='o')
 ) THEN RAISE EXCEPTION 'cuentas o grupos F4b con atributos, contrasena, propiedad o membresias inesperadas' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_roles l JOIN pg_roles g
   ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera',
                    'vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
   WHERE l.rolcanlogin AND NOT l.rolsuper
     AND l.rolname NOT IN (SELECT nombre FROM f4b_cuenta)
     AND pg_has_role(l.oid,g.oid,'MEMBER')) THEN
   RAISE EXCEPTION 'LOGIN ajeno conserva ruta a grupo Dietas o Personal D7' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   SELECT 1 FROM f4b_acl_requerida e
   WHERE NOT EXISTS (
     SELECT 1 FROM f4b_acl_grupo a
     WHERE a.grupo=e.grupo AND a.clase=e.clase AND a.privilegio=e.privilegio AND NOT a.grantable
       AND a.objeto=CASE e.clase
         WHEN 'relacion' THEN (SELECT to_regclass(e.objeto)::text)
         WHEN 'funcion' THEN (SELECT to_regprocedure(e.objeto)::text)
         ELSE e.objeto END)
 ) THEN
   RAISE EXCEPTION 'objeto o ACL requerida para Dietas F4b ausente' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   SELECT 1 FROM f4b_acl_grupo a
   WHERE a.grantable OR NOT (
        (a.clase='base' AND a.objeto=current_database() AND a.privilegio='CONNECT')
     OR (EXISTS (SELECT 1 FROM f4b_esquema_permitido p
                 WHERE p.grupo=a.grupo AND p.esquema=a.esquema AND NOT p.solo_requerida)
         AND ((a.clase='esquema' AND a.privilegio='USAGE')
           OR (a.clase IN ('relacion','columna') AND a.privilegio='SELECT')
           OR (a.clase='funcion' AND a.privilegio='EXECUTE')
           OR (a.clase='tipo' AND a.privilegio='USAGE')))
     -- Esquema restringido: solo lo que figura en la lista positiva.
     OR (EXISTS (SELECT 1 FROM f4b_esquema_permitido p
                 WHERE p.grupo=a.grupo AND p.esquema=a.esquema AND p.solo_requerida)
         AND EXISTS (SELECT 1 FROM f4b_acl_requerida e
                     WHERE e.grupo=a.grupo AND e.clase=a.clase AND e.privilegio=a.privilegio
                       AND e.clase IN ('esquema','funcion')
                       AND a.objeto=CASE e.clase
                         WHEN 'funcion' THEN (SELECT to_regprocedure(e.objeto)::text)
                         ELSE e.objeto END)))
 ) OR EXISTS (
   SELECT 1 FROM pg_shdepend d JOIN pg_roles g ON g.oid=d.refobjid
   WHERE g.rolname IN (SELECT grupo FROM f4b_cuenta)
     AND d.refclassid='pg_authid'::regclass AND d.deptype='a'
     AND (d.dbid NOT IN (0,(SELECT oid FROM pg_database WHERE datname=current_database()))
       OR d.classid NOT IN ('pg_database'::regclass,'pg_namespace'::regclass,
                           'pg_class'::regclass,'pg_proc'::regclass,'pg_type'::regclass))
 ) THEN
   RAISE EXCEPTION 'ACL de grupo fuera de los esquemas permitidos F4b' USING ERRCODE='55000';
 END IF;
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
   RAISE EXCEPTION 'ACL PUBLIC fuera de preimagen F4b permitida' USING ERRCODE='55000';
 END IF;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename IN (SELECT nombre FROM f4b_cuenta)) THEN
   RAISE EXCEPTION 'F4b requiere cero sesiones de las once cuentas' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE ROW EXCLUSIVE MODE;
DO $activar$
DECLARE v jsonb; anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
 original vec_autorizacion.asignacion_perfil%ROWTYPE;
 puntero vec_autorizacion.asignacion_perfil_actual%ROWTYPE;
 nueva jsonb; filas integer; punteros integer;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_plan;
 SELECT count(*) INTO punteros FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE r.rol_id='dietas_r1d_provisional';
 IF punteros<>1 THEN
   RAISE EXCEPTION 'F4b requiere un unico puntero Dietas global antes de LOGIN' USING ERRCODE='55000';
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
   RAISE EXCEPTION 'preimagen, vigencia o control V3 F4b incompatibles' USING ERRCODE='55000';
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
 IF filas<>1 THEN RAISE EXCEPTION 'puntero F4b no avanzo' USING ERRCODE='55000'; END IF;
END $activar$;
RESET ROLE;

-- Cuentas nuevas: se crean NOLOGIN con su verificador y su única membresía;
-- si ya existían (p. ej. preparadas por D7) solo reciben el verificador.
-- Un fallo de las sentencias dinámicas con verificador se sustituye por un
-- error sin CONTEXT de la sentencia interna (que llevaría el verificador al
-- registro del servidor y al cliente); se conserva el SQLSTATE original.
DO $cuentas$
DECLARE c record;
BEGIN
 FOR c IN SELECT k.nombre,k.grupo,x.verificador FROM f4b_cuenta k
          JOIN f4b_verificador x ON x.nombre=k.nombre WHERE k.nueva ORDER BY k.nombre LOOP
   BEGIN
     IF to_regrole(c.nombre) IS NULL THEN
       EXECUTE format('CREATE ROLE %I NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD %L',
                      c.nombre,c.verificador);
     ELSE
       EXECUTE format('ALTER ROLE %I PASSWORD %L',c.nombre,c.verificador);
     END IF;
   EXCEPTION WHEN OTHERS THEN
     RAISE EXCEPTION 'F4b no pudo fijar el verificador de %', c.nombre USING ERRCODE=SQLSTATE;
   END;
   IF NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=to_regrole(c.nombre)) THEN
     EXECUTE format('GRANT %I TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE',c.grupo,c.nombre);
   END IF;
 END LOOP;
END $cuentas$;

DO $login$
DECLARE c record;
BEGIN
 FOR c IN SELECT nombre FROM f4b_cuenta ORDER BY nombre LOOP
   EXECUTE format('ALTER ROLE %I LOGIN',c.nombre);
 END LOOP;
END $login$;

DO $post$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_plan;
 PERFORM pg_stat_clear_snapshot();
 IF (SELECT count(*) FROM pg_authid a JOIN f4b_cuenta c ON c.nombre=a.rolname
      WHERE a.rolcanlogin AND a.rolinherit AND NOT a.rolsuper AND NOT a.rolcreatedb
        AND NOT a.rolcreaterole AND NOT a.rolreplication AND NOT a.rolbypassrls
        AND a.rolpassword LIKE 'SCRAM-SHA-256$%'
        AND has_database_privilege(a.oid,current_database(),'CONNECT')
        AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=a.oid)=1
        AND EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
                     WHERE m.member=a.oid AND g.rolname=c.grupo
                       AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option))<>11
    OR EXISTS (SELECT 1 FROM f4b_verificador x JOIN pg_authid a ON a.rolname=x.nombre
               WHERE a.rolpassword IS DISTINCT FROM x.verificador)
    OR (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin)<>8
    OR EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename IN (SELECT nombre FROM f4b_cuenta))
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
       JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
       WHERE p.asignacion_ref=v->>'nueva_ref' AND a.huella_sha256=v->>'nueva_huella'
         AND a.documento=v->'documento' AND a.documento->>'estado'='activa') THEN
   RAISE EXCEPTION 'postcondicion F4b fallida' USING ERRCODE='55000';
 END IF;
END $post$;

:f4b_finalizar;
