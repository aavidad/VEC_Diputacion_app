\set ON_ERROR_STOP on
-- AUT-16. AUT-15 debe estar instalado antes. No traslada asignaciones
-- compartidas ni decisiones antiguas: cualquier fila externa previa bloquea.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000016',0));

DO $preimagen$
DECLARE n integer; r oid; f pg_catalog.pg_proc%ROWTYPE; esperado text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_publicador_candidato_externo') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_externa') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual_externa') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',
      'vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)','EXECUTE')
    OR pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_denegada_contexto_actor_v3_externa') IS NULL
 THEN RAISE EXCEPTION 'AUT-16: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH r IN ARRAY ARRAY[
  'vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text)'::regprocedure::oid,
  'vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure::oid,
  'vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure::oid]
 LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=r
    AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
    AND p.proconfig @> ARRAY['search_path=pg_catalog']::text[])
  THEN RAISE EXCEPTION 'AUT-16: función externa no exacta' USING ERRCODE='55000'; END IF;
  SELECT * INTO STRICT f FROM pg_catalog.pg_proc WHERE oid=r;
  esperado:=CASE f.proname
   WHEN 'obtener_instantanea_candidato_externo_v1' THEN 'd79a4a688c231a516ad7c71f081020252aa7050a27d6852dd38e86c42d93b9d0'
   WHEN 'registrar_decision_contexto_actor_v3_externa_interna' THEN 'ef7f3642c25ba3709360812c005cfafa653a640e75c821039d34de066393ab81'
   WHEN 'revalidar_decision_contexto_actor_v3_externa_interna' THEN 'e949eccf0f799ccb040d616e6017b9caeb459cffa3d7bc5c72b85b2857d7ec96'
   ELSE NULL END;
  IF esperado IS NULL OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(r),'UTF8')),'hex') IS DISTINCT FROM esperado
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(COALESCE(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole,
         CASE WHEN f.proname='obtener_instantanea_candidato_externo_v1' THEN 'vec_autorizacion_fuente_externa'::regrole ELSE 'vec_autorizacion_propietario'::regrole END)
         OR a.privilege_type<>'EXECUTE' OR (a.grantee<>f.proowner AND a.is_grantable))
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',r,'EXECUTE')
    OR (f.proname='obtener_instantanea_candidato_externo_v1' AND NOT pg_catalog.has_function_privilege('vec_autorizacion_fuente_externa',r,'EXECUTE'))
  THEN RAISE EXCEPTION 'AUT-16: definición o ACL externa divergente %',r USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT count(*) INTO n FROM pg_catalog.pg_constraint c
  WHERE c.contype='f' AND c.confrelid='vec_autorizacion.asignacion_perfil'::regclass
    AND c.conrelid IN ('vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass,
      'vec_autorizacion.decision_denegada_contexto_actor_v3_externa'::regclass);
 IF n<>2 OR EXISTS (SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa)
    OR EXISTS (SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa)
 THEN RAISE EXCEPTION 'AUT-16: decisiones externas existentes o FK incompatibles' USING ERRCODE='55000'; END IF;
END $preimagen$;

CREATE ROLE vec_autorizacion_publicador_candidato_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conectar$
BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_publicador_candidato_externo',pg_catalog.current_database());
END $conectar$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.asignacion_perfil_externa
 (LIKE vec_autorizacion.asignacion_perfil INCLUDING ALL);
CREATE TABLE vec_autorizacion.asignacion_perfil_actual_externa
 (LIKE vec_autorizacion.asignacion_perfil_actual INCLUDING ALL);
ALTER TABLE vec_autorizacion.asignacion_perfil_externa
 ADD CONSTRAINT asignacion_externa_rol_fk FOREIGN KEY(version_rol_ref)
 REFERENCES vec_autorizacion.version_rol(version_rol_ref);
ALTER TABLE vec_autorizacion.asignacion_perfil_actual_externa
 ADD CONSTRAINT asignacion_actual_externa_fk FOREIGN KEY(perfil_activo_ref,asignacion_ref)
 REFERENCES vec_autorizacion.asignacion_perfil_externa(perfil_activo_ref,asignacion_ref);
CREATE TABLE vec_autorizacion.publicacion_candidato_externo_evento (
 tipo text NOT NULL CHECK(tipo IN ('rol_control','asignacion')),
 acto_ref text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(acto_ref,512) IS TRUE),
 objeto_ref text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(objeto_ref,512) IS TRUE),
 version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615::numeric),
 huella_previa text CHECK(huella_previa ~ '^[0-9a-f]{64}$'),
 huella_nueva text NOT NULL CHECK(huella_nueva ~ '^[0-9a-f]{64}$'),
 publicada_por text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(publicada_por,512) IS TRUE),
 publicada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(tipo,acto_ref)
);

DO $rls$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['asignacion_perfil_externa','asignacion_perfil_actual_externa','publicacion_candidato_externo_evento'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user=''vec_autorizacion_propietario'') WITH CHECK (current_user=''vec_autorizacion_propietario'')',n);
 END LOOP;
END $rls$;
CREATE TRIGGER asignacion_externa_inmutable BEFORE UPDATE OR DELETE
 ON vec_autorizacion.asignacion_perfil_externa FOR EACH ROW
 EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER asignacion_externa_no_truncar BEFORE TRUNCATE
 ON vec_autorizacion.asignacion_perfil_externa FOR EACH STATEMENT
 EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER asignacion_actual_externa_no_eliminar BEFORE DELETE OR TRUNCATE
 ON vec_autorizacion.asignacion_perfil_actual_externa FOR EACH STATEMENT
 EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();
CREATE TRIGGER publicacion_externa_inmutable BEFORE UPDATE OR DELETE
 ON vec_autorizacion.publicacion_candidato_externo_evento FOR EACH ROW
 EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER publicacion_externa_no_truncar BEFORE TRUNCATE
 ON vec_autorizacion.publicacion_candidato_externo_evento FOR EACH STATEMENT
 EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE FUNCTION vec_autorizacion.validar_avance_asignacion_actual_externa()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE anterior vec_autorizacion.asignacion_perfil_externa%ROWTYPE;
        nueva vec_autorizacion.asignacion_perfil_externa%ROWTYPE;
BEGIN
 SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref=OLD.asignacion_ref;
 SELECT * INTO STRICT nueva FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref=NEW.asignacion_ref;
 IF NEW.perfil_activo_ref IS DISTINCT FROM OLD.perfil_activo_ref
    OR nueva.asignacion_id IS DISTINCT FROM anterior.asignacion_id
    OR nueva.principal_id IS DISTINCT FROM anterior.principal_id
    OR nueva.version<=anterior.version THEN
  RAISE EXCEPTION 'avance de asignación externa inválido' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER asignacion_actual_externa_avance BEFORE UPDATE
 ON vec_autorizacion.asignacion_perfil_actual_externa FOR EACH ROW
 EXECUTE FUNCTION vec_autorizacion.validar_avance_asignacion_actual_externa();

-- AUT-15 dejó las FK de decisión hacia la tabla compartida. Se cambia cada
-- referencia exacta; el guard anterior exige que ninguna decisión se pierda.
DO $fk$
DECLARE d record;
BEGIN
 FOR d IN SELECT c.conrelid,c.conname FROM pg_catalog.pg_constraint c
   WHERE c.contype='f' AND c.confrelid='vec_autorizacion.asignacion_perfil'::regclass
     AND c.conrelid IN ('vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass,
       'vec_autorizacion.decision_denegada_contexto_actor_v3_externa'::regclass)
 LOOP
  EXECUTE pg_catalog.format('ALTER TABLE %s DROP CONSTRAINT %I',d.conrelid::regclass,d.conname);
  EXECUTE pg_catalog.format('ALTER TABLE %s ADD CONSTRAINT %I FOREIGN KEY(asignacion_ref) REFERENCES vec_autorizacion.asignacion_perfil_externa(asignacion_ref)',d.conrelid::regclass,d.conname);
 END LOOP;
END $fk$;

-- Estos dos clones AUT-15 conservan su semántica, cambiando únicamente el
-- almacén de asignación. Fallan si una definición intermedia cambió.
DO $clones$
DECLARE f regprocedure; ddl text; nuevo text; marca text; n integer;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure,
  'vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)'::regprocedure]
 LOOP
  SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT ddl;
  marca:='vec_autorizacion.asignacion_perfil_actual';
  n:=(length(ddl)-length(replace(ddl,marca,'')))/length(marca);
  IF n<>1 OR strpos(ddl,'vec_autorizacion.asignacion_perfil_externa')<>0 THEN
   RAISE EXCEPTION 'AUT-16: clon de asignación externa incompatible %',f USING ERRCODE='55000'; END IF;
  nuevo:=replace(ddl,marca,marca||'_externa');
  marca:='vec_autorizacion.asignacion_perfil';
  n:=(length(nuevo)-length(replace(nuevo,marca||' ','')))/length(marca||' ');
  IF n<>1 THEN RAISE EXCEPTION 'AUT-16: clon de historia externa incompatible %',f USING ERRCODE='55000'; END IF;
  nuevo:=replace(nuevo,marca||' ',marca||'_externa ');
  EXECUTE nuevo;
 END LOOP;
END $clones$;

CREATE OR REPLACE FUNCTION vec_autorizacion.obtener_instantanea_candidato_externo_v1(p_principal_id text,p_perfil_activo_ref text)
RETURNS TABLE(documento_asignacion jsonb,documento_rol jsonb,documento_control_rol jsonb,revision_catalogo text,huella_catalogo text,documentos_politicas jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_fuente_autorizacion_desarrollo','vec_autorizacion_fuente_externa') IS NOT TRUE
    OR vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(p_principal_id,p_perfil_activo_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'fuente externa rechazada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.documento,r.documento,c.documento,cat.revision::text,cat.huella_sha256,
   COALESCE((SELECT pg_catalog.jsonb_agg(p.documento ORDER BY p.politica_ref)
     FROM vec_autorizacion.politica_restrictiva_actual pa
     JOIN vec_autorizacion.politica_restrictiva p ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref),'[]'::jsonb)
 FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.perfil_activo_ref=aa.perfil_activo_ref AND a.asignacion_ref=aa.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 CROSS JOIN vec_autorizacion.control_catalogo_politicas cat
 WHERE aa.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id AND cat.control_id=true
   AND r.rol_id IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo');
END $f$;

CREATE FUNCTION vec_autorizacion.publicador_candidato_externo_interno_valido_v1()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none'
   AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles l WHERE l.rolname=session_user
     AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
     AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls AND l.rolconfig IS NULL)
   AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles g WHERE g.rolname='vec_autorizacion_publicador_candidato_externo'
     AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolsuper AND NOT g.rolcreatedb
     AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls AND g.rolconfig IS NULL)
   AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1
   AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
     WHERE m.member=session_user::regrole AND m.roleid='vec_autorizacion_publicador_candidato_externo'::regrole
       AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>session_user::regrole
     AND r.oid<>'vec_autorizacion_publicador_candidato_externo'::regrole
     AND pg_catalog.pg_has_role(session_user::regrole,r.oid,'MEMBER'))
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>'vec_autorizacion_publicador_candidato_externo'::regrole
     AND pg_catalog.pg_has_role('vec_autorizacion_publicador_candidato_externo'::regrole,r.oid,'MEMBER'))
$f$;

CREATE FUNCTION vec_autorizacion.rol_candidato_externo_acotado_v1(d jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE id text; x jsonb; acciones text[]; esperadas text[];
BEGIN
 id:=d->>'rol_id';
 IF pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR NOT d ?& ARRAY['rol_id','version','nombre','estado','concesiones','publicada_por','publicada_en']
    OR d->>'nombre' IS DISTINCT FROM (CASE WHEN id='candidato_bolsa_portal_historial_propio_desarrollo'
       THEN 'areaPersonal.miBolsa.rolPortal' ELSE 'Consulta propia de bolsa en desarrollo' END)
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 esperadas:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar'];
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' THEN
  esperadas:=esperadas||ARRAY['bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto'];
 END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RETURN false; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x) k WHERE k NOT IN
      ('accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','obligaciones','garantia_minima'))<>0
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'bolsa' OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR x->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN x->>'accion'='bolsa.participaciones_propias.manifestar_disposicion' THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END)
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[CASE
      WHEN x->>'accion'='bolsa.historial_propio.consultar' THEN 'consulta_historial_propio'
      WHEN x->>'accion'='bolsa.participaciones_propias.consultar' THEN 'consulta_participaciones_propias'
      ELSE 'gestion_participaciones_propias' END])
    OR (x->>'accion'='bolsa.participaciones_propias.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["participaciones_candidato_minimizadas"]'::jsonb)
    OR (x->>'accion'='bolsa.historial_propio.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb)
    OR (x->>'accion' NOT IN ('bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar')
      AND COALESCE(x->'campos_permitidos','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb)
    OR COALESCE(x->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $f$;

-- El catálogo de roles y sus controles son comunes. Esta fachada publica
-- únicamente las dos definiciones acotadas del candidato, sin asignar perfil.
-- El par rol/control se confirma en la misma transacción del operador que
-- llama después a publicar_asignacion_candidato_externo_v1.
CREATE FUNCTION vec_autorizacion.publicar_rol_candidato_externo_v1(
 p_rol_canonico bytea,p_rol_huella text,p_control_canonico bytea,p_control_huella text,
 p_revision_esperada numeric,p_control_huella_esperada text,p_actualizada_por text,p_acto_ref text)
RETURNS TABLE(version_rol_ref text,version bigint,revision numeric,huella_rol text,huella_control text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb; c jsonb; ref text; id text; v bigint; rev numeric;
        existente record; actual record; x jsonb; acciones text[]; esperadas text[];
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_rol_canonico IS NULL OR pg_catalog.octet_length(p_rol_canonico) NOT BETWEEN 1 AND 65536
    OR p_control_canonico IS NULL OR pg_catalog.octet_length(p_control_canonico) NOT BETWEEN 1 AND 16384
    OR p_rol_huella !~ '^[0-9a-f]{64}$' OR p_control_huella !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_rol_canonico),'hex') IS DISTINCT FROM p_rol_huella
    OR pg_catalog.encode(pg_catalog.sha256(p_control_canonico),'hex') IS DISTINCT FROM p_control_huella
    OR p_revision_esperada IS NULL OR p_revision_esperada<0 OR p_revision_esperada<>trunc(p_revision_esperada)
    OR (p_revision_esperada=0 AND p_control_huella_esperada IS NOT NULL)
    OR (p_revision_esperada>0 AND p_control_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación de rol externo rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_rol_canonico,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_control_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento de rol externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'rol_id'; v:=(d->>'version')::bigint;
 ref:='rol:'||id||':v'||v::text; rev:=(c->>'revision')::numeric;
 IF id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR vec_autorizacion.rol_candidato_externo_acotado_v1(d) IS NOT TRUE
    OR v IS NULL OR v<1 OR v=9223372036854775807
    OR d->>'estado' IS DISTINCT FROM 'publicada'
    OR d->>'nombre' IS DISTINCT FROM (CASE WHEN id='candidato_bolsa_portal_historial_propio_desarrollo'
       THEN 'areaPersonal.miBolsa.rolPortal' ELSE 'Consulta propia de bolsa en desarrollo' END)
    OR d->>'publicada_por' IS DISTINCT FROM 'seguridad:desarrollo:no-autoritativa'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR d ?| ARRAY['retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo']
    OR c->>'version_rol_ref' IS DISTINCT FROM ref
    OR c->>'estado' IS DISTINCT FROM 'habilitada'
    OR rev IS DISTINCT FROM p_revision_esperada+1
    OR c->>'actualizado_por' IS DISTINCT FROM p_actualizada_por
    OR c ?| ARRAY['acto_ref','motivo_codigo']
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
    OR pg_catalog.jsonb_typeof(d->'concesiones') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'rol externo inválido' USING ERRCODE='42501'; END IF;
 esperadas:=ARRAY['bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar'];
 IF id='candidato_bolsa_portal_historial_propio_desarrollo' THEN
  esperadas:=esperadas||ARRAY[
   'bolsa.participaciones_propias.solicitar_pausa',
   'bolsa.participaciones_propias.solicitar_reactivacion',
   'bolsa.participaciones_propias.responder_llamamiento',
   'bolsa.participaciones_propias.manifestar_disposicion',
   'bolsa.participaciones_propias.confirmar_contacto'];
 END IF;
 SELECT pg_catalog.array_agg(e->>'accion' ORDER BY e->>'accion') INTO acciones
 FROM pg_catalog.jsonb_array_elements(d->'concesiones') e;
 IF acciones IS DISTINCT FROM (SELECT pg_catalog.array_agg(a ORDER BY a) FROM pg_catalog.unnest(esperadas) a)
 THEN RAISE EXCEPTION 'concesiones de rol externo incompatibles' USING ERRCODE='42501'; END IF;
 FOR x IN SELECT value FROM pg_catalog.jsonb_array_elements(d->'concesiones') LOOP
  IF pg_catalog.jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(x) k WHERE k NOT IN
      ('accion','modulo_id','tipo_recurso','finalidades','campos_permitidos','obligaciones','garantia_minima'))<>0
    OR NOT x ?& ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima']
    OR x->>'modulo_id' IS DISTINCT FROM 'bolsa' OR x->>'garantia_minima' IS DISTINCT FROM 'alto'
    OR x->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN x->>'accion'='bolsa.participaciones_propias.manifestar_disposicion' THEN 'oferta_bolsa' ELSE 'participaciones_candidato' END)
    OR x->'finalidades' IS DISTINCT FROM pg_catalog.to_jsonb(ARRAY[CASE
      WHEN x->>'accion'='bolsa.historial_propio.consultar' THEN 'consulta_historial_propio'
      WHEN x->>'accion'='bolsa.participaciones_propias.consultar' THEN 'consulta_participaciones_propias'
      ELSE 'gestion_participaciones_propias' END])
    OR (x->>'accion'='bolsa.participaciones_propias.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["participaciones_candidato_minimizadas"]'::jsonb)
    OR (x->>'accion'='bolsa.historial_propio.consultar'
      AND x->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb)
    OR (x->>'accion' NOT IN ('bolsa.historial_propio.consultar','bolsa.participaciones_propias.consultar')
      AND COALESCE(x->'campos_permitidos','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb)
    OR COALESCE(x->'obligaciones','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
  THEN RAISE EXCEPTION 'concesión externa no acotada' USING ERRCODE='42501'; END IF;
 END LOOP;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:rol_candidato_externo:'||id,0));
 SELECT r.version_rol_ref,r.rol_id,r.version,r.huella_sha256,r.documento INTO existente
 FROM vec_autorizacion.version_rol r WHERE r.version_rol_ref=ref FOR SHARE;
 IF FOUND THEN
  IF existente.rol_id IS DISTINCT FROM id OR existente.version IS DISTINCT FROM v
     OR existente.huella_sha256 IS DISTINCT FROM p_rol_huella OR existente.documento IS DISTINCT FROM d
  THEN RAISE EXCEPTION 'rol externo publicado divergente' USING ERRCODE='40001'; END IF;
 ELSE
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES(ref,id,v,p_rol_huella,(d->>'publicada_en')::timestamptz,d);
 END IF;
 SELECT a.revision,h.huella_sha256 INTO actual
 FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol h USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_revision_esperada<>0 OR p_control_huella_esperada IS NOT NULL))
    OR (FOUND AND (actual.revision IS DISTINCT FROM p_revision_esperada
      OR actual.huella_sha256 IS DISTINCT FROM p_control_huella_esperada))
 THEN RAISE EXCEPTION 'control de rol externo: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(ref,rev,'habilitada',p_control_huella,(c->>'actualizado_en')::timestamptz,c);
 IF p_revision_esperada=0 THEN
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
  VALUES(ref,rev,(c->>'actualizado_en')::timestamptz,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.control_vigencia_version_rol_actual AS puntero SET revision=rev,
   actualizada_en=(c->>'actualizado_en')::timestamptz,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.version_rol_ref=ref AND puntero.revision=p_revision_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'control de rol externo cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('rol_control',p_acto_ref,ref,rev,p_control_huella_esperada,p_control_huella,p_actualizada_por);
 RETURN QUERY SELECT ref,v,rev,p_rol_huella,p_control_huella;
END $f$;

-- Fachada exclusiva de provisión interna. El bytea es el JSON canónico Go;
-- su huella se comprueba antes de convertirlo a jsonb. Sin alta automática.
CREATE FUNCTION vec_autorizacion.publicar_asignacion_candidato_externo_v1(
 p_documento_canonico bytea,p_huella_sha256 text,p_version_esperada bigint,
 p_huella_esperada text,p_actualizada_por text,p_acto_ref text)
RETURNS TABLE(asignacion_ref text,version bigint,huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb; r record; anterior record; ahora timestamptz:=pg_catalog.clock_timestamp();
        id text; perfil text; principal text; rol text; nueva_version bigint; ref text;
BEGIN
 IF vec_autorizacion.publicador_candidato_externo_interno_valido_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_documento_canonico IS NULL OR pg_catalog.octet_length(p_documento_canonico) NOT BETWEEN 1 AND 65536
    OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_documento_canonico),'hex') IS DISTINCT FROM p_huella_sha256
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR (p_version_esperada=0 AND p_huella_esperada IS NOT NULL)
    OR (p_version_esperada>0 AND p_huella_esperada !~ '^[0-9a-f]{64}$')
    OR vec_autorizacion.texto_positivo_valido(p_actualizada_por,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(p_acto_ref,512) IS NOT TRUE
 THEN RAISE EXCEPTION 'publicación externa rechazada' USING ERRCODE='42501'; END IF;
 BEGIN d:=pg_catalog.convert_from(p_documento_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'documento externo inválido' USING ERRCODE='22023'; END;
 id:=d->>'asignacion_id'; perfil:=d->>'perfil_activo_ref'; principal:=d->>'principal_id'; rol:=d->>'version_rol_ref';
 nueva_version:=p_version_esperada+1; ref:='asignacion:'||id||':v'||nueva_version::text;
 IF vec_autorizacion.texto_positivo_valido(id,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(perfil,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(principal,512) IS NOT TRUE
    OR vec_autorizacion.texto_positivo_valido(rol,512) IS NOT TRUE
    OR (d->>'version')::bigint IS DISTINCT FROM nueva_version
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>11
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d->>'revocada_por' IS NOT NULL OR d->>'revocada_en' IS NOT NULL OR d->>'revocacion_ref' IS NOT NULL
    OR pg_catalog.jsonb_array_length(d->'ambitos')<>1
    OR pg_catalog.jsonb_typeof(d->'ambitos'->0) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d->'ambitos'->0))<>2
    OR NOT (d->'ambitos'->0) ?& ARRAY['clave','valores']
    OR d->'ambitos'->0->>'clave' IS DISTINCT FROM 'candidato_ref'
    OR pg_catalog.jsonb_array_length(d->'ambitos'->0->'valores')<>1
    OR d->'ambitos'->0->'valores'->>0 !~ '^can_[A-Za-z0-9_-]+$'
    OR d->>'emitida_por' IS DISTINCT FROM p_actualizada_por
    OR d->>'vigente_desde' IS NULL OR d->>'vigente_hasta' IS NULL
    OR (d->>'vigente_desde')::timestamptz>ahora
    OR (d->>'vigente_hasta')::timestamptz<=ahora
    OR vec_contexto_actor_v1.acreditar_candidato_externo_v1(principal,perfil,d->'ambitos'->0->'valores'->>0) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación externa inválida o sin provisión' USING ERRCODE='42501'; END IF;
 SELECT vr.rol_id,vr.huella_sha256,vr.documento INTO r FROM vec_autorizacion.version_rol vr
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=vr.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE vr.version_rol_ref=rol AND c.estado='habilitada' FOR SHARE OF ca;
 IF NOT FOUND OR r.rol_id NOT IN ('candidato_bolsa_historial_propio_desarrollo','candidato_bolsa_portal_historial_propio_desarrollo')
    OR vec_autorizacion.rol_candidato_externo_acotado_v1(r.documento) IS NOT TRUE
 THEN RAISE EXCEPTION 'rol externo no aprobado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:asignacion_candidato_externo:'||perfil,0));
 SELECT a.asignacion_ref,a.asignacion_id,a.version,a.principal_id,a.huella_sha256
 INTO anterior FROM vec_autorizacion.asignacion_perfil_actual_externa aa
 JOIN vec_autorizacion.asignacion_perfil_externa a ON a.asignacion_ref=aa.asignacion_ref
 WHERE aa.perfil_activo_ref=perfil FOR UPDATE OF aa;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada
      OR anterior.asignacion_id IS DISTINCT FROM id OR anterior.principal_id IS DISTINCT FROM principal))
 THEN RAISE EXCEPTION 'asignación externa: CAS divergente' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil_externa(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(ref,id,nueva_version,perfil,principal,rol,p_huella_sha256,(d->>'emitida_en')::timestamptz,d);
 IF p_version_esperada=0 THEN
  INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
  VALUES(perfil,ref,ahora,p_actualizada_por,p_acto_ref);
 ELSE
  UPDATE vec_autorizacion.asignacion_perfil_actual_externa AS puntero
   SET asignacion_ref=ref,actualizada_en=ahora,actualizada_por=p_actualizada_por,acto_ref=p_acto_ref
  WHERE puntero.perfil_activo_ref=perfil AND puntero.asignacion_ref=anterior.asignacion_ref;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación externa: puntero cambiado' USING ERRCODE='40001'; END IF;
 END IF;
 INSERT INTO vec_autorizacion.publicacion_candidato_externo_evento(
  tipo,acto_ref,objeto_ref,version,huella_previa,huella_nueva,publicada_por)
 VALUES('asignacion',p_acto_ref,ref,nueva_version,p_huella_esperada,p_huella_sha256,p_actualizada_por);
 RETURN QUERY SELECT ref,nueva_version,p_huella_sha256;
END $f$;

REVOKE ALL ON TABLE vec_autorizacion.asignacion_perfil_externa,vec_autorizacion.asignacion_perfil_actual_externa,
 vec_autorizacion.publicacion_candidato_externo_evento FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.asignacion_perfil_externa,vec_autorizacion.asignacion_perfil_actual_externa,
 vec_autorizacion.publicacion_candidato_externo_evento FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_avance_asignacion_actual_externa(),
 vec_autorizacion.publicador_candidato_externo_interno_valido_v1(),
 vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb),
 vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text),
 vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_publicador_candidato_externo;
GRANT EXECUTE ON FUNCTION vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text),
 vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)
 TO vec_autorizacion_publicador_candidato_externo;
COMMIT;
