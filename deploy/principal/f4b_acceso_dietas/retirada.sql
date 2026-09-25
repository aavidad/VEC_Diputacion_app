-- F4b retirada. Precedido por las variables del ejecutor y por comun.sql.
-- Crea v4 revocada y vuelve las once cuentas a NOLOGIN; conserva roles,
-- verificadores, membresías e historia V3. Termina con :f4b_retirada_finalizar.
CREATE TEMP TABLE f4b_retirada_plan (dato jsonb NOT NULL) ON COMMIT DROP;
INSERT INTO f4b_retirada_plan VALUES
 (convert_from(decode(:'f4b_retirada_plan_b64','base64'),'UTF8')::jsonb);
GRANT SELECT ON f4b_retirada_plan TO vec_autorizacion_propietario;

DO $pre$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_retirada_plan;
 IF v->>'actor'<>'administracion:f4b:retirada-acceso-dietas'
    OR v->>'acto'<>'acto:f4b:retirada-acceso-dietas:20260925'
    OR v->>'anterior_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v3'
    OR v->>'nueva_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v4'
    OR v->>'nueva_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'anterior_huella' !~ '^[0-9a-f]{64}$'
    OR v->'documento'->>'estado'<>'revocada'
    OR v->'documento'->>'revocada_por' IS DISTINCT FROM v->>'actor'
    OR v->'documento'->>'revocacion_ref' IS DISTINCT FROM v->>'acto'
    OR v->'documento'->>'version_rol_ref'<>'rol:dietas_r1d_provisional:v1' THEN
   RAISE EXCEPTION 'plan Go F4b retirada invalido' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$')<>8 THEN
   RAISE EXCEPTION 'F4b retirada requiere exactamente las ocho cuentas R1D' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_roles l JOIN pg_roles g
   ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera',
                    'vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
   WHERE l.rolcanlogin AND NOT l.rolsuper
     AND l.rolname NOT IN (SELECT nombre FROM f4b_cuenta)
     AND pg_has_role(l.oid,g.oid,'MEMBER')) THEN
   RAISE EXCEPTION 'LOGIN ajeno conserva ruta a grupo Dietas o Personal D7' USING ERRCODE='55000';
 END IF;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename IN (SELECT nombre FROM f4b_cuenta)) THEN
   RAISE EXCEPTION 'F4b retirada requiere cero sesiones de las once cuentas' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   SELECT 1 FROM f4b_cuenta c
   LEFT JOIN pg_roles r ON r.rolname=c.nombre
   WHERE r.oid IS NULL OR NOT r.rolcanlogin OR r.rolsuper OR r.rolcreatedb
      OR r.rolcreaterole OR NOT r.rolinherit OR r.rolreplication OR r.rolbypassrls
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid)<>1
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.roleid=r.oid)<>0
      OR NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
        WHERE m.member=r.oid AND g.rolname=c.grupo
          AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 ) THEN
   RAISE EXCEPTION 'atributos o membresias F4b retirada incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $revocar$
DECLARE v jsonb; anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
        puntero vec_autorizacion.asignacion_perfil_actual%ROWTYPE;
        nueva jsonb; filas integer;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_retirada_plan;
 SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
  WHERE perfil_activo_ref=v->'documento'->>'perfil_activo_ref' FOR UPDATE;
 SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=puntero.asignacion_ref;
 nueva:=v->'documento';
 IF puntero.asignacion_ref IS DISTINCT FROM v->>'anterior_ref'
    OR puntero.actualizada_por<>'administracion:f4b:acceso-dietas'
    OR puntero.acto_ref<>'acto:f4b:acceso-dietas:20260925'
    OR anterior.huella_sha256 IS DISTINCT FROM v->>'anterior_huella'
    OR anterior.documento->>'estado'<>'activa'
    OR anterior.version<>3
    OR anterior.version_rol_ref<>'rol:dietas_r1d_provisional:v1'
    OR nueva->>'asignacion_id' IS DISTINCT FROM anterior.asignacion_id
    OR nueva->>'principal_id' IS DISTINCT FROM anterior.principal_id
    OR nueva->>'perfil_activo_ref' IS DISTINCT FROM anterior.perfil_activo_ref
    OR nueva->>'version_rol_ref' IS DISTINCT FROM anterior.version_rol_ref
    OR nueva->'ambitos' IS DISTINCT FROM anterior.documento->'ambitos'
    OR nueva->>'vigente_desde' IS DISTINCT FROM anterior.documento->>'vigente_desde'
    OR nueva->>'vigente_hasta' IS DISTINCT FROM anterior.documento->>'vigente_hasta'
    OR nueva->>'emitida_por' IS DISTINCT FROM anterior.documento->>'emitida_por'
    OR nueva->>'emitida_en' IS DISTINCT FROM anterior.documento->>'emitida_en'
    OR (nueva->>'version')::bigint<>anterior.version+1
    OR v->>'nueva_ref'<>'asignacion:'||anterior.asignacion_id||':v'||(anterior.version+1)::text
    OR (nueva-'version'-'estado'-'revocada_por'-'revocada_en'-'revocacion_ref')
       IS DISTINCT FROM (anterior.documento-'version'-'estado'-'revocada_por'-'revocada_en'-'revocacion_ref') THEN
   RAISE EXCEPTION 'preimagen F4b retirada cambio o documento alterado' USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
   version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES (v->>'nueva_ref',anterior.asignacion_id,anterior.version+1,
   anterior.perfil_activo_ref,anterior.principal_id,anterior.version_rol_ref,
   v->>'nueva_huella',anterior.emitida_en,nueva);
 UPDATE vec_autorizacion.asignacion_perfil_actual
 SET asignacion_ref=v->>'nueva_ref',actualizada_en=clock_timestamp(),
     actualizada_por=v->>'actor',acto_ref=v->>'acto'
 WHERE perfil_activo_ref=puntero.perfil_activo_ref
   AND asignacion_ref=puntero.asignacion_ref;
 GET DIAGNOSTICS filas=ROW_COUNT;
 IF filas<>1 THEN RAISE EXCEPTION 'puntero F4b retirada no avanzo' USING ERRCODE='55000'; END IF;
END $revocar$;
RESET ROLE;

DO $nologin$
DECLARE c record;
BEGIN
 FOR c IN SELECT nombre FROM f4b_cuenta ORDER BY nombre LOOP
   EXECUTE format('ALTER ROLE %I NOLOGIN',c.nombre);
 END LOOP;
END $nologin$;

DO $post$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM f4b_retirada_plan;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (SELECT nombre FROM f4b_cuenta) AND rolcanlogin)
    OR EXISTS (
      SELECT 1 FROM pg_roles l JOIN pg_roles g
        ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera',
                         'vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
      WHERE l.rolcanlogin AND NOT l.rolsuper AND pg_has_role(l.oid,g.oid,'MEMBER'))
    OR EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename IN (SELECT nombre FROM f4b_cuenta))
    OR NOT EXISTS (
      SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
      JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
      WHERE p.asignacion_ref=v->>'nueva_ref' AND a.huella_sha256=v->>'nueva_huella'
        AND a.documento=v->'documento' AND a.documento->>'estado'='revocada') THEN
   RAISE EXCEPTION 'postcondicion F4b retirada fallida' USING ERRCODE='55000';
 END IF;
END $post$;

:f4b_retirada_finalizar;
