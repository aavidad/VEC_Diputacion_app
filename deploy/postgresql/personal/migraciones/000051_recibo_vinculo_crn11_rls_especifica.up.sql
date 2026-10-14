\set ON_ERROR_STOP on
-- P51: limita el recibo de lectura histórica CRN11 de Personal26 a las
-- operaciones SELECT/INSERT y a la pareja persona-empleado ya acreditada.
-- Personal26 queda intacta en Git; esta sucesora conserva su OID y ACL.
-- Requiere AD269 y Personal26 recién instalada. Si hubiera recibos previos,
-- para sin modificarlos: ese estado necesita una migración de historia aparte.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000051',0));
DO $pre$
DECLARE politica record; f oid:='vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_personal.recibo_vinculo_propio_crn11') IS NULL
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL
    OR to_regprocedure('vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND a.attname IN ('persona_ref','proyeccion_ref','proyeccion_version') AND NOT a.attisdropped)
    OR EXISTS (SELECT 1 FROM vec_personal.recibo_vinculo_propio_crn11)
 THEN RAISE EXCEPTION 'P51: preimagen incompatible o historia previa' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR (SELECT array_agg(a.attname::text ORDER BY a.attnum) FROM pg_attribute a
         WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass AND a.attnum>0 AND NOT a.attisdropped)
       IS DISTINCT FROM ARRAY['recibo_ref','empleado_ref','material_sha256','decision_ref','auditoria_ref','consumo_huella_sha256','consultada_en']
    OR EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
         AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attnotnull)
    OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
       WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass AND a.grantee<>c.relowner)
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.proyeccion_empleado_persona_historia'::regclass
       AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_personal.proyeccion_empleado_persona_historia'::regclass
       AND c.conname='proyeccion_empleado_persona_historia_pkey' AND c.contype='p'
       AND pg_get_constraintdef(c.oid,true)='PRIMARY KEY (proyeccion_ref, version)')
 THEN RAISE EXCEPTION 'P51: tabla o fuente incompatible' USING ERRCODE='55000'; END IF;
 SELECT pol.polname AS nombre,pol.polcmd AS comando,pol.polroles AS roles,
   pg_get_expr(pol.polqual,pol.polrelid) AS condicion,pg_get_expr(pol.polwithcheck,pol.polrelid) AS condicion_insercion
   INTO STRICT politica FROM pg_policy pol WHERE pol.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass;
 IF politica.nombre<>'propietario_interno' OR politica.comando<>'*'
    OR politica.roles<>ARRAY['vec_personal_propietario'::regrole::oid]
    OR politica.condicion<>'true' OR politica.condicion_insercion<>'true' THEN
   RAISE EXCEPTION 'P51: política Personal26 incompatible' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
       AND t.tgname IN ('historia_inmutable','no_truncar') AND t.tgenabled='O'
       AND t.tgfoid='vec_personal.rechazar_mutacion_proyeccion_empleado_v1()'::regprocedure)<>2 THEN
   RAISE EXCEPTION 'P51: historia sin protección' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_personal_propietario'::regrole
       AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
       AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='ab9dc0b7838bfd0fa580239cee7f6ac1e8506ccfcb98d4af5201cf0720dab8d7')
    OR NOT has_function_privilege('vec_personal_ejecutor','vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee NOT IN (p.proowner,'vec_personal_ejecutor'::regrole))
 THEN RAISE EXCEPTION 'P51: función Personal26 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_personal.recibo_vinculo_propio_crn11
 ADD COLUMN persona_ref text NOT NULL CHECK (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 ADD COLUMN proyeccion_ref text NOT NULL CHECK (proyeccion_ref ~ '^pep_[A-Za-z0-9_-]{22,128}$'),
 ADD COLUMN proyeccion_version bigint NOT NULL CHECK (proyeccion_version>0),
 ADD CONSTRAINT recibo_vinculo_crn11_fuente_fk FOREIGN KEY (proyeccion_ref,proyeccion_version)
   REFERENCES vec_personal.proyeccion_empleado_persona_historia(proyeccion_ref,version);

CREATE FUNCTION vec_personal.recibo_crn11_fuente_valida_v1(p_proyeccion text,p_version bigint,p_persona text,p_empleado text,p_consultada timestamptz)
RETURNS boolean LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref=p_proyeccion AND h.version=p_version
    AND h.persona_ref=p_persona AND h.empleado_ref=p_empleado AND h.estado='activa'
    AND h.vigente_desde<=p_consultada AND p_consultada<h.vigente_hasta)
$f$;
REVOKE ALL ON FUNCTION vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

DO $funcion$
DECLARE f oid:='vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=$x$ INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);$x$;
 sustitucion text:=$x$ INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,persona_ref,proyeccion_ref,proyeccion_version,
  material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,persona,fuente.proyeccion_ref,fuente.version,
  material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora
  FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to((SELECT prosrc FROM pg_proc WHERE oid=f),'UTF8')),'hex')<>'ab9dc0b7838bfd0fa580239cee7f6ac1e8506ccfcb98d4af5201cf0720dab8d7'
    OR (length(original)-length(replace(original,marca,'')))<>length(marca)
 THEN RAISE EXCEPTION 'P51: punto de inserción incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustitucion);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,sustitucion,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'P51: función alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $funcion$;

DROP POLICY propietario_interno ON vec_personal.recibo_vinculo_propio_crn11;
CREATE POLICY lectura_fuente ON vec_personal.recibo_vinculo_propio_crn11 FOR SELECT TO vec_personal_propietario
 USING (vec_personal.recibo_crn11_fuente_valida_v1(proyeccion_ref,proyeccion_version,persona_ref,empleado_ref,consultada_en));
CREATE POLICY adicion_fuente ON vec_personal.recibo_vinculo_propio_crn11 FOR INSERT TO vec_personal_propietario
 WITH CHECK (vec_personal.recibo_crn11_fuente_valida_v1(proyeccion_ref,proyeccion_version,persona_ref,empleado_ref,consultada_en));
REVOKE ALL ON TYPE vec_personal.recibo_vinculo_propio_crn11 FROM PUBLIC,vec_personal_ejecutor,vec_personal_migrador;

DO $final$
BEGIN
 IF (SELECT count(*) FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND p.polname IN ('lectura_fuente','adicion_fuente')
      AND ((p.polname='lectura_fuente' AND p.polcmd='r') OR (p.polname='adicion_fuente' AND p.polcmd='a'))
      AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid])<>2
    OR EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND p.polname NOT IN ('lectura_fuente','adicion_fuente'))
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.recibo_vinculo_propio_crn11'::regclass
      AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR EXISTS (SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
      WHERE t.oid='vec_personal.recibo_vinculo_propio_crn11'::regtype AND a.grantee<>t.typowner)
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid='vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)'::regprocedure
      AND p.proowner='vec_personal_propietario'::regrole AND NOT p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid='vec_personal.recibo_crn11_fuente_valida_v1(text,bigint,text,text,timestamptz)'::regprocedure
        AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'P51: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $final$;
COMMIT;
