\set ON_ERROR_STOP on
-- CA21: vínculo nominal RPT segregado de consulta_rrhh. AUT conserva roles,
-- asignaciones y motivos; esta proyección no concede una acción de negocio.
-- Publicación administrativa CAS; sin DOWN ni reescritura de historia CA4/CA9.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:rpt:000021',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer/10000<>18
 OR current_setting('server_encoding')<>'UTF8'
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_rpt_versiones') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1'
               AND nspowner='vec_contexto_actor_v1_propietario'::regrole) THEN
  RAISE EXCEPTION 'CA21: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.vinculo_rpt_versiones (
    vinculo_rpt_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    cuenta_ref text NOT NULL,
    cuenta_version numeric(20,0) NOT NULL,
    persona_ref text NOT NULL,
    persona_version numeric(20,0) NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20,0) NOT NULL,
    vinculo_contexto_ref text NOT NULL,
    vinculo_contexto_version numeric(20,0) NOT NULL,
    organizacion_ref text NOT NULL,
    organizacion_version numeric(20,0) NOT NULL,
    organizacion_procedencia_ref text NOT NULL,
    organizacion_procedencia_version numeric(20,0) NOT NULL,
    organizacion_procedencia_huella_sha256 text NOT NULL,
    organizacion_procedencia_autoridad text NOT NULL,
    superficie text NOT NULL,
    uso text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    catalogo_id text NOT NULL CHECK(catalogo_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
    modulo_id text NOT NULL CHECK(modulo_id ~ '^[a-z][a-z0-9_.:-]{2,127}$'),
    rol_ref text NOT NULL,
    rol_version numeric(20,0) NOT NULL CHECK(rol_version BETWEEN 1 AND 18446744073709551615::numeric),
    rol_huella_sha256 text NOT NULL CHECK(rol_huella_sha256 ~ '^[0-9a-f]{64}$'),
    documento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(documento)='object' AND pg_catalog.octet_length(documento::text)<=16384),
    huella_sha256 text NOT NULL CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(documento,version)::text,'UTF8')),'hex')),
    CHECK(rol_ref='rol:rrhh_gobierno_categorias_rpt:v'||rol_version::text),
    CONSTRAINT vinculo_rpt_versiones_pk
      PRIMARY KEY (vinculo_rpt_ref, version),
    UNIQUE(vinculo_rpt_ref,version,huella_sha256),
    CONSTRAINT vinculo_rpt_versiones_actual_uq UNIQUE
      (cuenta_ref, perfil_ref, catalogo_id, modulo_id, vinculo_rpt_ref, version),
    CONSTRAINT vinculo_rpt_versiones_cuenta_fk FOREIGN KEY
      (cuenta_ref, cuenta_version) REFERENCES
      vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version)
      MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_persona_fk FOREIGN KEY
      (persona_ref, persona_version) REFERENCES
      vec_contexto_actor_v1.persona_versiones(persona_ref,version) MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_perfil_persona_fk FOREIGN KEY
      (perfil_ref,perfil_version,persona_ref) REFERENCES
      vec_contexto_actor_v1.perfil_versiones(perfil_ref,version,persona_ref)
      MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_vinculo_contexto_fk FOREIGN KEY
      (vinculo_contexto_ref,vinculo_contexto_version,cuenta_ref,perfil_ref,persona_ref)
      REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones
      (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref) MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_organizacion_fk FOREIGN KEY
      (organizacion_ref,organizacion_version,organizacion_procedencia_ref,
       organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
       organizacion_procedencia_autoridad) REFERENCES
      vec_contexto_actor_v1.organizacion_versiones
      (organizacion_ref,version,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_procedencia_fk FOREIGN KEY
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) REFERENCES vec_contexto_actor_v1.procedencias
      (procedencia_ref,procedencia_version,procedencia_huella_sha256,
       procedencia_autoridad) MATCH FULL,
    CONSTRAINT vinculo_rpt_versiones_ref_ck CHECK
      (vec_contexto_actor_v1.referencia_valida(vinculo_rpt_ref,'vcr_')),
    CONSTRAINT vinculo_rpt_versiones_version_ck CHECK
      (version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_cuenta_version_ck CHECK
      (cuenta_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_persona_version_ck CHECK
      (persona_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_perfil_version_ck CHECK
      (perfil_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_vinculo_contexto_version_ck CHECK
      (vinculo_contexto_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_organizacion_version_ck CHECK
      (organizacion_version BETWEEN 1 AND 18446744073709551615::numeric),
    CONSTRAINT vinculo_rpt_versiones_superficie_ck CHECK
      (superficie='interna_corporativa'),
    CONSTRAINT vinculo_rpt_versiones_uso_ck CHECK (uso='gobierno_categorias_rpt'),
    CONSTRAINT vinculo_rpt_versiones_organizacion_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(
       organizacion_procedencia_ref,organizacion_procedencia_version,
       organizacion_procedencia_huella_sha256,organizacion_procedencia_autoridad)),
    CONSTRAINT vinculo_rpt_versiones_organizacion_autoridad_ck CHECK
      (organizacion_procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_rpt_versiones_procedencia_ck CHECK
      (vec_contexto_actor_v1.procedencia_valida(procedencia_ref,
       procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CONSTRAINT vinculo_rpt_versiones_procedencia_autoridad_ck CHECK
      (procedencia_autoridad='autoridad_maestra_acreditada'),
    CONSTRAINT vinculo_rpt_versiones_estado_ck CHECK
      (estado IN ('activo','revocado')),
    CONSTRAINT vinculo_rpt_versiones_vigente_desde_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CONSTRAINT vinculo_rpt_versiones_vigente_hasta_ck CHECK
      (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CONSTRAINT vinculo_rpt_versiones_ventana_ck CHECK
      (vigente_hasta > vigente_desde)
);

CREATE TABLE vec_contexto_actor_v1.vinculo_rpt_actual(
 cuenta_ref text NOT NULL,perfil_ref text NOT NULL,catalogo_id text NOT NULL,modulo_id text NOT NULL,
 vinculo_rpt_ref text NOT NULL,version numeric(20,0) NOT NULL,
 PRIMARY KEY(cuenta_ref,perfil_ref,catalogo_id,modulo_id),
 FOREIGN KEY(cuenta_ref,perfil_ref,catalogo_id,modulo_id,vinculo_rpt_ref,version)
 REFERENCES vec_contexto_actor_v1.vinculo_rpt_versiones(cuenta_ref,perfil_ref,catalogo_id,modulo_id,vinculo_rpt_ref,version)
);
CREATE TABLE vec_contexto_actor_v1.vinculo_rpt_eventos(
 vinculo_rpt_ref text NOT NULL,version numeric(20,0) NOT NULL,
 huella_sha256 text NOT NULL,actor_tecnico_ref text NOT NULL,evidencia_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(vinculo_rpt_ref,version),
 FOREIGN KEY(vinculo_rpt_ref,version,huella_sha256) REFERENCES vec_contexto_actor_v1.vinculo_rpt_versiones(vinculo_rpt_ref,version,huella_sha256),
 CHECK(pg_catalog.octet_length(actor_tecnico_ref) BETWEEN 3 AND 160),
 CHECK(pg_catalog.octet_length(evidencia_ref) BETWEEN 3 AND 160)
);
DO $proteccion$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['vinculo_rpt_versiones','vinculo_rpt_actual','vinculo_rpt_eventos'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I TO vec_contexto_actor_v1_propietario USING(current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK(current_user=''vec_contexto_actor_v1_propietario'')',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',t);
  IF t<>'vinculo_rpt_actual' THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()',t);
  END IF;
 END LOOP;
END $proteccion$;
-- Misma barrera que los punteros maestros: revalidación y mutación no se cruzan.
CREATE TRIGGER serializar_mutacion BEFORE INSERT OR UPDATE OR DELETE
 ON vec_contexto_actor_v1.vinculo_rpt_actual FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion AFTER INSERT OR UPDATE OR DELETE
 ON vec_contexto_actor_v1.vinculo_rpt_actual FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();

CREATE FUNCTION vec_contexto_actor_v1.revalidar_vinculo_rpt_rrhh_v1(
 p_cuenta_ref text,p_perfil_ref text,p_persona_ref text,p_vinculo_contexto_ref text,
 p_vinculo_contexto_version numeric,p_catalogo_id text,p_modulo_id text,
 p_rol_ref text,p_rol_version numeric,p_rol_huella_sha256 text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 -- FOR SHARE obliga a abortar si cualquier puntero cambió después de fijarse
 -- la instantánea SERIALIZABLE; el bloqueo asesor por sí solo no lo garantiza.
 SELECT cv.*,c.estado AS c_estado,c.vigente_desde AS c_desde,c.vigente_hasta AS c_hasta,
        p.estado AS p_estado,p.vigente_desde AS p_desde,p.vigente_hasta AS p_hasta,
        x.estado AS x_estado,x.vigente_desde AS x_desde,x.vigente_hasta AS x_hasta,
        v.estado AS v_estado,v.vigente_desde AS v_desde,v.vigente_hasta AS v_hasta,
        ov.estado AS o_estado,ov.vigente_desde AS o_desde,ov.vigente_hasta AS o_hasta
 INTO r
 FROM vec_contexto_actor_v1.vinculo_rpt_actual a
 JOIN vec_contexto_actor_v1.vinculo_rpt_versiones cv USING(vinculo_rpt_ref,version)
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON (ca.cuenta_ref,ca.version)=(cv.cuenta_ref,cv.cuenta_version)
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c ON (c.cuenta_ref,c.version)=(ca.cuenta_ref,ca.version)
 JOIN vec_contexto_actor_v1.perfil_actual pa ON (pa.perfil_ref,pa.version)=(cv.perfil_ref,cv.perfil_version)
 JOIN vec_contexto_actor_v1.perfil_versiones p ON (p.perfil_ref,p.version)=(pa.perfil_ref,pa.version)
 JOIN vec_contexto_actor_v1.persona_actual xa ON (xa.persona_ref,xa.version)=(cv.persona_ref,cv.persona_version)
 JOIN vec_contexto_actor_v1.persona_versiones x ON (x.persona_ref,x.version)=(xa.persona_ref,xa.version)
 JOIN vec_contexto_actor_v1.vinculo_contexto_actual va ON (va.vinculo_ref,va.version)=(cv.vinculo_contexto_ref,cv.vinculo_contexto_version)
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v ON (v.vinculo_ref,v.version)=(va.vinculo_ref,va.version)
 JOIN vec_contexto_actor_v1.organizacion_actual oa ON (oa.organizacion_ref,oa.version)=(cv.organizacion_ref,cv.organizacion_version)
 JOIN vec_contexto_actor_v1.organizacion_versiones ov ON (ov.organizacion_ref,ov.version)=(oa.organizacion_ref,oa.version)
 WHERE a.cuenta_ref=p_cuenta_ref AND a.perfil_ref=p_perfil_ref
 AND a.catalogo_id=p_catalogo_id AND a.modulo_id=p_modulo_id
 AND cv.persona_ref=p_persona_ref AND cv.vinculo_contexto_ref=p_vinculo_contexto_ref
 AND cv.vinculo_contexto_version=p_vinculo_contexto_version
 AND cv.rol_ref=p_rol_ref AND cv.rol_version=p_rol_version AND cv.rol_huella_sha256=p_rol_huella_sha256
 AND p.persona_ref=cv.persona_ref AND (v.cuenta_ref,v.perfil_ref,v.persona_ref)=(cv.cuenta_ref,cv.perfil_ref,cv.persona_ref)
 AND (cv.organizacion_procedencia_ref,cv.organizacion_procedencia_version,
      cv.organizacion_procedencia_huella_sha256,cv.organizacion_procedencia_autoridad)
     =(ov.procedencia_ref,ov.procedencia_version,ov.procedencia_huella_sha256,ov.procedencia_autoridad)
 AND c.procedencia_autoridad='autoridad_maestra_acreditada'
 AND p.procedencia_autoridad='autoridad_maestra_acreditada'
 AND x.procedencia_autoridad='autoridad_maestra_acreditada'
 AND v.procedencia_autoridad='autoridad_maestra_acreditada'
 FOR SHARE OF a,ca,pa,xa,va,oa;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.estado='activo' AND r.c_estado='activo' AND r.p_estado='activo'
 AND r.x_estado='activo' AND r.v_estado='activo' AND r.o_estado='activo'
 AND ahora>=r.vigente_desde AND ahora<r.vigente_hasta
 AND ahora>=r.c_desde AND ahora<r.c_hasta AND ahora>=r.p_desde AND ahora<r.p_hasta
 AND ahora>=r.x_desde AND ahora<r.x_hasta AND ahora>=r.v_desde AND ahora<r.v_hasta
 AND ahora>=r.o_desde AND ahora<r.o_hasta;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_rpt_rrhh_v1(text,text,text,text,numeric,text,text,text,numeric,text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(
 p_principal_id text,p_perfil_ref text,p_catalogo_id text,p_modulo_id text,
 p_rol_ref text,p_rol_version numeric,p_rol_huella_sha256 text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record; n integer:=0; valido boolean:=false;
BEGIN
 -- Un perfil activo no suma vínculos de cuentas distintas.
 FOR r IN SELECT v.* FROM vec_contexto_actor_v1.vinculo_rpt_actual a
 JOIN vec_contexto_actor_v1.vinculo_rpt_versiones v USING(vinculo_rpt_ref,version)
 WHERE v.persona_ref=p_principal_id AND v.perfil_ref=p_perfil_ref
 AND v.catalogo_id=p_catalogo_id AND v.modulo_id=p_modulo_id LOOP
  n:=n+1;
  valido:=vec_contexto_actor_v1.revalidar_vinculo_rpt_rrhh_v1(r.cuenta_ref,p_perfil_ref,
    p_principal_id,r.vinculo_contexto_ref,r.vinculo_contexto_version,p_catalogo_id,p_modulo_id,
    p_rol_ref,p_rol_version,p_rol_huella_sha256);
 END LOOP;
 RETURN n=1 AND valido;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(text,text,text,text,text,numeric,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_rpt_rrhh_v1(text,text,text,text,text,numeric,text)
 TO vec_autorizacion_propietario;

-- Aprovisionamiento administrativo de fuente, nunca permiso por petición.
-- La huella aprobada ata documento completo y versión resultante; AUT25
-- publicará después su asignación/rol por su propio circuito y autoridad.
CREATE FUNCTION vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(
 p_documento jsonb,p_version_esperada numeric,p_huella_esperada text,
 p_huella_aprobada text,p_actor_tecnico_ref text,p_evidencia_ref text
) RETURNS TABLE(version numeric,huella_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE r vec_contexto_actor_v1.vinculo_rpt_versiones%ROWTYPE;
 anterior vec_contexto_actor_v1.vinculo_rpt_versiones%ROWTYPE;
 nueva numeric; h text; k text; x jsonb;
BEGIN
 IF current_user<>'vec_contexto_actor_v1_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'CA21: publicación administrativa no autorizada' USING ERRCODE='42501';
 END IF;
 IF p_documento IS NULL OR pg_catalog.jsonb_typeof(p_documento)<>'object'
 OR pg_catalog.octet_length(p_documento::text)>16384
 OR NOT p_documento ?& ARRAY['vinculo_rpt_ref','cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','perfil_version','vinculo_contexto_ref','vinculo_contexto_version','organizacion_ref','organizacion_version','organizacion_procedencia_ref','organizacion_procedencia_version','organizacion_procedencia_huella_sha256','organizacion_procedencia_autoridad','superficie','uso','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad','estado','vigente_desde','vigente_hasta','catalogo_id','modulo_id','rol_ref','rol_version','rol_huella_sha256']
 OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_documento))<>29
 OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 18446744073709551614::numeric
 OR p_version_esperada<>pg_catalog.trunc(p_version_esperada)
 OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$'
 OR p_actor_tecnico_ref IS NULL OR pg_catalog.octet_length(p_actor_tecnico_ref) NOT BETWEEN 3 AND 160
 OR p_evidencia_ref IS NULL OR pg_catalog.octet_length(p_evidencia_ref) NOT BETWEEN 3 AND 160 THEN
  RAISE EXCEPTION 'CA21: documento administrativo inválido' USING ERRCODE='22023';
 END IF;
 FOREACH k IN ARRAY ARRAY['vinculo_rpt_ref','cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','perfil_version','vinculo_contexto_ref','vinculo_contexto_version','organizacion_ref','organizacion_version','organizacion_procedencia_ref','organizacion_procedencia_version','organizacion_procedencia_huella_sha256','organizacion_procedencia_autoridad','superficie','uso','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad','estado','vigente_desde','vigente_hasta','catalogo_id','modulo_id','rol_ref','rol_version','rol_huella_sha256'] LOOP
  x:=p_documento->k;
  IF pg_catalog.right(k,8)='_version' THEN
   IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'number'
      OR x::text !~ '^[1-9][0-9]{0,19}$'
      OR x::text::numeric>18446744073709551615::numeric THEN
    RAISE EXCEPTION 'CA21: versión inválida' USING ERRCODE='22023';
   END IF;
  ELSIF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'string' THEN
   RAISE EXCEPTION 'CA21: campo inválido' USING ERRCODE='22023';
  END IF;
 END LOOP;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 r:=pg_catalog.jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_rpt_versiones,p_documento);
 SELECT v.* INTO anterior FROM vec_contexto_actor_v1.vinculo_rpt_actual a
 JOIN vec_contexto_actor_v1.vinculo_rpt_versiones v USING(vinculo_rpt_ref,version)
 WHERE a.cuenta_ref=r.cuenta_ref AND a.perfil_ref=r.perfil_ref
 AND a.catalogo_id=r.catalogo_id AND a.modulo_id=r.modulo_id FOR UPDATE OF a;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL OR r.estado<>'activo'))
 OR (FOUND AND (anterior.version IS DISTINCT FROM p_version_esperada
     OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
     OR anterior.vinculo_rpt_ref IS DISTINCT FROM r.vinculo_rpt_ref
     OR anterior.persona_ref IS DISTINCT FROM r.persona_ref
     OR anterior.vinculo_contexto_ref IS DISTINCT FROM r.vinculo_contexto_ref))
 OR (r.estado='revocado' AND (p_documento-ARRAY['estado','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'])
     IS DISTINCT FROM (anterior.documento-ARRAY['estado','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'])) THEN
  RAISE EXCEPTION 'CA21: preimagen CAS divergente' USING ERRCODE='40001';
 END IF;
 nueva:=pg_catalog.trunc(p_version_esperada)+1;
 h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(p_documento,nueva)::text,'UTF8')),'hex');
 IF h IS DISTINCT FROM p_huella_aprobada THEN
  RAISE EXCEPTION 'CA21: contenido no aprobado' USING ERRCODE='42501';
 END IF;
 r.version:=nueva;r.documento:=p_documento;r.huella_sha256:=h;
 INSERT INTO vec_contexto_actor_v1.vinculo_rpt_versiones SELECT r.*;
 INSERT INTO vec_contexto_actor_v1.vinculo_rpt_actual VALUES(r.cuenta_ref,r.perfil_ref,r.catalogo_id,r.modulo_id,r.vinculo_rpt_ref,nueva)
 ON CONFLICT(cuenta_ref,perfil_ref,catalogo_id,modulo_id) DO UPDATE SET version=EXCLUDED.version;
 IF r.estado='activo' AND vec_contexto_actor_v1.revalidar_vinculo_rpt_rrhh_v1(
    r.cuenta_ref,r.perfil_ref,r.persona_ref,r.vinculo_contexto_ref,r.vinculo_contexto_version,
    r.catalogo_id,r.modulo_id,r.rol_ref,r.rol_version,r.rol_huella_sha256) IS NOT TRUE THEN
  RAISE EXCEPTION 'CA21: fuente nominal no vigente' USING ERRCODE='42501';
 END IF;
 INSERT INTO vec_contexto_actor_v1.vinculo_rpt_eventos(vinculo_rpt_ref,version,huella_sha256,actor_tecnico_ref,evidencia_ref)
 VALUES(r.vinculo_rpt_ref,nueva,h,p_actor_tecnico_ref,p_evidencia_ref);
 RETURN QUERY SELECT nueva,h;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_vinculo_rpt_rrhh_v1(jsonb,numeric,text,text,text,text) FROM PUBLIC;
COMMIT;
