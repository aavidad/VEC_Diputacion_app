\set ON_ERROR_STOP on
-- CA25. Vínculo nominal gobernado de la huella DER exacta; sin perfil ni cargo.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000025',0));
DO $pre$
DECLARE nombre text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') IS NULL
  OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
  OR pg_catalog.to_regclass('vec_contexto_actor_v1.certificado_firmante_versiones') IS NOT NULL
  OR pg_catalog.to_regclass('vec_contexto_actor_v1.certificado_firmante_actual') IS NOT NULL
  OR pg_catalog.to_regclass('vec_contexto_actor_v1.certificado_firmante_nominal_versiones') IS NOT NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA25: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['proyeccion_cuenta_actual','proyeccion_cuenta_versiones','persona_actual','persona_versiones','vinculo_contexto_actual','vinculo_contexto_versiones','control_generacion_punteros_actuales_v2'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_contexto_actor_v1.'||nombre)
   AND c.relowner='vec_contexto_actor_v1_propietario'::regrole AND c.relkind='r' AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
    WHERE a.grantee<>c.relowner))
  THEN RAISE EXCEPTION 'CA25: fuente ausente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['organizacion_actual','organizacion_versiones','vinculo_corporativo_actual','vinculo_corporativo_versiones'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_contexto_actor_v1.'||nombre)
   AND c.relowner='vec_contexto_actor_v1_propietario'::regrole AND c.relkind='r'
   AND c.relrowsecurity AND c.relforcerowsecurity)
  THEN RAISE EXCEPTION 'CA25: organización corporativa ausente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 IF pg_catalog.to_regprocedure('vec_contexto_actor_v1.rechazar_mutacion_historia()') IS NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.rechazar_truncado()') IS NULL
 THEN RAISE EXCEPTION 'CA25: guardas de historia ausentes' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

CREATE TABLE vec_contexto_actor_v1.certificado_firmante_nominal_versiones (
 vinculo_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_ref,'vcc_') IS TRUE),
 version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615),
 certificado_der_sha256 text NOT NULL CHECK(certificado_der_sha256 ~ '^[0-9a-f]{64}$' AND certificado_der_sha256<>pg_catalog.repeat('0',64)),
 cuenta_ref text NOT NULL,cuenta_version numeric(20,0) NOT NULL,cuenta_huella_sha256 text NOT NULL CHECK(cuenta_huella_sha256 ~ '^[0-9a-f]{64}$'),
 persona_ref text NOT NULL,persona_version numeric(20,0) NOT NULL,persona_huella_sha256 text NOT NULL CHECK(persona_huella_sha256 ~ '^[0-9a-f]{64}$'),
 vinculo_cuenta_persona_ref text NOT NULL,vinculo_cuenta_persona_version numeric(20,0) NOT NULL,
 vinculo_cuenta_persona_huella_sha256 text NOT NULL CHECK(vinculo_cuenta_persona_huella_sha256 ~ '^[0-9a-f]{64}$'),
 estado text NOT NULL CHECK(estado IN ('vigente','retirado')),
 vigente_desde timestamptz(6) NOT NULL,vigente_hasta timestamptz(6) NOT NULL,
 evidencia_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(evidencia_ref,'evi_') IS TRUE),
 evidencia_sha256 text NOT NULL CHECK(evidencia_sha256 ~ '^[0-9a-f]{64}$' AND evidencia_sha256<>pg_catalog.repeat('0',64)),
 decision_ref text NOT NULL,auditoria_ref text NOT NULL,recibo_ref text NOT NULL,
 clave text NOT NULL CHECK(clave ~ '^[0-9a-f]{32}$'),
 descriptor_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(descriptor_canonico) BETWEEN 2 AND 16384),
 documento_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(documento_canonico) BETWEEN 2 AND 32768),
 huella_sha256 text NOT NULL CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(vinculo_ref,version), UNIQUE(certificado_der_sha256,vinculo_ref,version,huella_sha256),
 UNIQUE(clave),UNIQUE(decision_ref),UNIQUE(recibo_ref),
 FOREIGN KEY(cuenta_ref,cuenta_version) REFERENCES vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref,version),
 FOREIGN KEY(persona_ref,persona_version) REFERENCES vec_contexto_actor_v1.persona_versiones(persona_ref,version),
 FOREIGN KEY(vinculo_cuenta_persona_ref,vinculo_cuenta_persona_version) REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref,version),
 CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde) IS TRUE AND vec_contexto_actor_v1.instante_valido(vigente_hasta) IS TRUE AND vigente_hasta>vigente_desde)
);
CREATE TABLE vec_contexto_actor_v1.certificado_firmante_nominal_actual (
 certificado_der_sha256 text PRIMARY KEY,vinculo_ref text NOT NULL,version numeric(20,0) NOT NULL,huella_sha256 text NOT NULL,
 FOREIGN KEY(certificado_der_sha256,vinculo_ref,version,huella_sha256)
 REFERENCES vec_contexto_actor_v1.certificado_firmante_nominal_versiones(certificado_der_sha256,vinculo_ref,version,huella_sha256)
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.certificado_firmante_nominal_versiones
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.certificado_firmante_nominal_versiones
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_no_borrable BEFORE DELETE ON vec_contexto_actor_v1.certificado_firmante_nominal_actual
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER puntero_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.certificado_firmante_nominal_actual
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
DO $acl$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['certificado_firmante_nominal_versiones','certificado_firmante_nominal_actual'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK(current_user=''vec_contexto_actor_v1_propietario'')',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
 END LOOP;
END $acl$;

-- Bloquea punteros de las tres fuentes hasta COMMIT y comprueba sus filas
-- versionadas antes de producir la huella. No atribuye cargo ni perfil.
CREATE FUNCTION vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(c text,p text,v text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET TimeZone='UTC' AS $f$
DECLARE ca record;pe record;vi record;ahora timestamptz;generacion numeric;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR vec_contexto_actor_v1.referencia_valida(c,'cta_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(p,'per_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(v,'vca_') IS NOT TRUE THEN RETURN NULL; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT x.* INTO ca FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones x USING(cuenta_ref,version) WHERE a.cuenta_ref=c FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO pe FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version) WHERE a.persona_ref=p FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO vi FROM vec_contexto_actor_v1.vinculo_contexto_actual a
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version) WHERE a.vinculo_ref=v FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.generacion INTO STRICT generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 x WHERE control_id=true FOR SHARE OF x;
 ahora:=pg_catalog.clock_timestamp();
 IF ca.estado<>'activo' OR pe.estado<>'activo' OR vi.estado<>'activo'
  OR ca.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR pe.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR vi.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR vi.cuenta_ref<>c OR vi.persona_ref<>p
  OR ahora<GREATEST(ca.vigente_desde,pe.vigente_desde,vi.vigente_desde)
  OR ahora>=LEAST(ca.vigente_hasta,pe.vigente_hasta,vi.vigente_hasta) THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object(
  'cuenta',pg_catalog.jsonb_build_object('referencia',c,'version',ca.version,'huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(ca)::text,'UTF8')),'hex')),
  'persona',pg_catalog.jsonb_build_object('referencia',p,'version',pe.version,'huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(pe)::text,'UTF8')),'hex')),
  'vinculo_cuenta_persona',pg_catalog.jsonb_build_object('referencia',v,'version',vi.version,'huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(vi)::text,'UTF8')),'hex'),'cuenta_ref',vi.cuenta_ref,'persona_ref',vi.persona_ref,'perfil_ref',vi.perfil_ref));
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(text,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
 c text,p text,v text,v_version numeric,org_esperada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET TimeZone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb;pr record;cr record;o record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR vec_contexto_actor_v1.organizacion_ref_valida(org_esperada) IS NOT TRUE
  OR v_version IS NULL OR v_version<1 OR pg_catalog.scale(v_version)<>0 THEN RETURN NULL; END IF;
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(c,p,v);
 IF s IS NULL OR (s#>>'{vinculo_cuenta_persona,version}')::numeric<>v_version THEN RETURN NULL; END IF;
 SELECT x.* INTO pr FROM vec_contexto_actor_v1.perfil_actual a
 JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
 WHERE a.perfil_ref=s#>>'{vinculo_cuenta_persona,perfil_ref}' FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO cr FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones x
  ON x.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND x.version=a.version
 WHERE a.cuenta_ref=c AND a.superficie='interna_corporativa' AND a.uso='consulta_rrhh' FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO o FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones x USING(organizacion_ref,version)
 WHERE a.organizacion_ref=org_esperada FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF pr.persona_ref<>p OR pr.estado<>'activo' OR pr.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR cr.cuenta_ref<>c OR cr.persona_ref<>p OR cr.perfil_ref<>pr.perfil_ref
  OR cr.vinculo_contexto_ref<>v OR cr.vinculo_contexto_version<>v_version
  OR cr.cuenta_version<>(s#>>'{cuenta,version}')::numeric
  OR cr.persona_version<>(s#>>'{persona,version}')::numeric OR cr.perfil_version<>pr.version
  OR cr.organizacion_ref<>org_esperada OR cr.organizacion_version<>o.version
  OR cr.organizacion_procedencia_ref<>o.procedencia_ref
  OR cr.organizacion_procedencia_version<>o.procedencia_version
  OR cr.organizacion_procedencia_huella_sha256<>o.procedencia_huella_sha256
  OR cr.organizacion_procedencia_autoridad<>o.procedencia_autoridad
  OR cr.estado<>'activo' OR o.estado<>'activo'
  OR cr.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR ahora<GREATEST(pr.vigente_desde,cr.vigente_desde,o.vigente_desde)
  OR ahora>=LEAST(pr.vigente_hasta,cr.vigente_hasta,o.vigente_hasta)
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_ref',o.organizacion_ref,'organizacion_version',o.version,
  'organizacion_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(o)::text,'UTF8')),'hex'),
  'organizacion_procedencia_ref',o.procedencia_ref,'organizacion_procedencia_version',o.procedencia_version,
  'organizacion_procedencia_sha256',o.procedencia_huella_sha256,
  'vinculo_corporativo_ref',cr.vinculo_corporativo_ref,'vinculo_corporativo_version',cr.version,
  'vinculo_corporativo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(cr)::text,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)
 TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(b bytea,decision text,auditoria text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb;s jsonb;org jsonb;a record;h record;fecha timestamptz(6);doc bytea;sha text;recibo text;result jsonb;canon text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR b IS NULL OR pg_catalog.octet_length(b) NOT BETWEEN 2 AND 16384
  OR decision IS NULL OR pg_catalog.octet_length(decision) NOT BETWEEN 1 AND 512
  OR auditoria IS NULL OR pg_catalog.octet_length(auditoria) NOT BETWEEN 1 AND 512
 THEN RAISE EXCEPTION 'CA25: publicación denegada' USING ERRCODE='42501'; END IF;
 d:=pg_catalog.convert_from(b,'UTF8')::jsonb;
 IF pg_catalog.jsonb_typeof(d)<>'object' OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>17
  OR NOT d ?& ARRAY['esquema','clave','vinculo_ref','version','certificado_der_sha256','cuenta_ref','persona_ref','vinculo_cuenta_persona_ref','organizacion_ref','estado','vigente_desde','vigente_hasta','evidencia_ref','evidencia_sha256','preimagen_ref','preimagen_version','preimagen_sha256']
  OR d->>'esquema'<>'vec.contexto-actor.certificado-firmante.publicacion.v2'
  OR d->>'clave' !~ '^[0-9a-f]{32}$' OR d->>'certificado_der_sha256' !~ '^[0-9a-f]{64}$'
  OR d->>'certificado_der_sha256'=pg_catalog.repeat('0',64)
  OR vec_contexto_actor_v1.referencia_valida(d->>'vinculo_ref','vcc_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'cuenta_ref','cta_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'persona_ref','per_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'vinculo_cuenta_persona_ref','vca_') IS NOT TRUE
  OR vec_contexto_actor_v1.organizacion_ref_valida(d->>'organizacion_ref') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(d->>'evidencia_ref','evi_') IS NOT TRUE
  OR d->>'evidencia_sha256' !~ '^[0-9a-f]{64}$' OR d->>'evidencia_sha256'=pg_catalog.repeat('0',64)
  OR d->>'estado' NOT IN ('vigente','retirado')
  OR d->>'version' !~ '^[1-9][0-9]{0,19}$' OR (d->>'version')::numeric>18446744073709551615
  OR d->>'preimagen_version' !~ '^(0|[1-9][0-9]{0,19})$' OR (d->>'preimagen_version')::numeric>18446744073709551615
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(d) e
   WHERE e.key NOT IN ('version','preimagen_version') AND pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'string')
  OR pg_catalog.jsonb_typeof(d->'version') IS DISTINCT FROM 'number'
  OR pg_catalog.jsonb_typeof(d->'preimagen_version') IS DISTINCT FROM 'number'
  OR d->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
  OR d->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
  OR NOT pg_catalog.isfinite((d->>'vigente_desde')::timestamptz)
  OR NOT pg_catalog.isfinite((d->>'vigente_hasta')::timestamptz)
  OR (d->>'vigente_hasta')::timestamptz<=(d->>'vigente_desde')::timestamptz
 THEN RAISE EXCEPTION 'CA25: descriptor inválido' USING ERRCODE='22023'; END IF;
 canon:=pg_catalog.format('{"esquema":%s,"clave":%s,"vinculo_ref":%s,"version":%s,"certificado_der_sha256":%s,"cuenta_ref":%s,"persona_ref":%s,"vinculo_cuenta_persona_ref":%s,"organizacion_ref":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"evidencia_ref":%s,"evidencia_sha256":%s,"preimagen_ref":%s,"preimagen_version":%s,"preimagen_sha256":%s}',
  pg_catalog.to_json(d->>'esquema'),pg_catalog.to_json(d->>'clave'),pg_catalog.to_json(d->>'vinculo_ref'),d->>'version',
  pg_catalog.to_json(d->>'certificado_der_sha256'),pg_catalog.to_json(d->>'cuenta_ref'),pg_catalog.to_json(d->>'persona_ref'),
  pg_catalog.to_json(d->>'vinculo_cuenta_persona_ref'),pg_catalog.to_json(d->>'organizacion_ref'),pg_catalog.to_json(d->>'estado'),pg_catalog.to_json(d->>'vigente_desde'),
  pg_catalog.to_json(d->>'vigente_hasta'),pg_catalog.to_json(d->>'evidencia_ref'),pg_catalog.to_json(d->>'evidencia_sha256'),
  pg_catalog.to_json(d->>'preimagen_ref'),d->>'preimagen_version',pg_catalog.to_json(d->>'preimagen_sha256'));
 IF pg_catalog.convert_to(canon,'UTF8') IS DISTINCT FROM b
 THEN RAISE EXCEPTION 'CA25: descriptor no canónico' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:certificado_firmante_v2:'||(d->>'certificado_der_sha256'),0));
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref');
 IF s IS NULL THEN RAISE EXCEPTION 'CA25: fuente no vigente' USING ERRCODE='42501'; END IF;
 org:=vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
  d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref',
  (s#>>'{vinculo_cuenta_persona,version}')::numeric,d->>'organizacion_ref');
 IF org IS NULL THEN RAISE EXCEPTION 'CA25: organización no acreditada' USING ERRCODE='42501'; END IF;
 SELECT x.* INTO a FROM vec_contexto_actor_v1.certificado_firmante_nominal_actual x
 WHERE x.certificado_der_sha256=d->>'certificado_der_sha256' FOR UPDATE;
 SELECT x.* INTO h FROM vec_contexto_actor_v1.certificado_firmante_nominal_versiones x WHERE x.clave=d->>'clave';
 IF FOUND THEN
  -- Cada replay llega con nueva autorización/auditoría V3; el recibo y la
  -- decisión originales permanecen inmutables en esta fila histórica.
  IF h.descriptor_canonico IS DISTINCT FROM b
   OR (pg_catalog.convert_from(h.documento_canonico,'UTF8')::jsonb)->'organizacion_destino' IS DISTINCT FROM org
   OR a.vinculo_ref IS DISTINCT FROM h.vinculo_ref OR a.version IS DISTINCT FROM h.version OR a.huella_sha256 IS DISTINCT FROM h.huella_sha256
   OR (s#>>'{cuenta,version}')::numeric IS DISTINCT FROM h.cuenta_version
   OR s#>>'{cuenta,huella_sha256}' IS DISTINCT FROM h.cuenta_huella_sha256
   OR (s#>>'{persona,version}')::numeric IS DISTINCT FROM h.persona_version
   OR s#>>'{persona,huella_sha256}' IS DISTINCT FROM h.persona_huella_sha256
   OR (s#>>'{vinculo_cuenta_persona,version}')::numeric IS DISTINCT FROM h.vinculo_cuenta_persona_version
   OR s#>>'{vinculo_cuenta_persona,huella_sha256}' IS DISTINCT FROM h.vinculo_cuenta_persona_huella_sha256
  THEN RAISE EXCEPTION 'CA25: replay divergente' USING ERRCODE='23514'; END IF;
  RETURN pg_catalog.convert_from(h.documento_canonico,'UTF8')::jsonb;
 END IF;
 IF a.certificado_der_sha256 IS NULL THEN
  IF d->>'preimagen_ref'<>'' OR (d->>'preimagen_version')::numeric<>0 OR d->>'preimagen_sha256'<>'none'
   OR (d->>'version')::numeric<>1 OR d->>'estado'<>'vigente' THEN
   RAISE EXCEPTION 'CA25: preimagen inicial inválida' USING ERRCODE='40001'; END IF;
 ELSE
  IF a.vinculo_ref IS DISTINCT FROM d->>'preimagen_ref'
   OR a.version IS DISTINCT FROM (d->>'preimagen_version')::numeric
   OR a.huella_sha256 IS DISTINCT FROM d->>'preimagen_sha256'
   OR a.vinculo_ref IS DISTINCT FROM d->>'vinculo_ref'
   OR a.version+1 IS DISTINCT FROM (d->>'version')::numeric
   OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.certificado_firmante_nominal_versiones x
    WHERE x.vinculo_ref=a.vinculo_ref AND x.version=a.version AND
     (x.cuenta_ref<>d->>'cuenta_ref' OR x.persona_ref<>d->>'persona_ref' OR x.estado='retirado'))
  THEN RAISE EXCEPTION 'CA25: CAS o titular divergente' USING ERRCODE='40001'; END IF;
 END IF;
 fecha:=pg_catalog.clock_timestamp();
 IF d->>'estado'='vigente' AND (fecha<(d->>'vigente_desde')::timestamptz OR fecha>=(d->>'vigente_hasta')::timestamptz)
 THEN RAISE EXCEPTION 'CA25: vigencia inválida' USING ERRCODE='42501'; END IF;
 recibo:='recibo_certificado_nominal:'||(d->>'clave');
 result:=pg_catalog.jsonb_build_object('esquema','vec.contexto-actor.certificado-firmante.recibo.v2',
  'clave',d->>'clave','vinculo_ref',d->>'vinculo_ref','version',(d->>'version')::numeric,
  'certificado_der_sha256',d->>'certificado_der_sha256','estado',d->>'estado',
  'decision_ref',decision,'auditoria_ref',auditoria,'recibo_ref',recibo,
  'descriptor_sha256',pg_catalog.encode(pg_catalog.sha256(b),'hex'),'organizacion_destino',org,
  'registrada_en',pg_catalog.to_char(fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 doc:=pg_catalog.convert_to(result::text,'UTF8');sha:=pg_catalog.encode(pg_catalog.sha256(doc),'hex');
 INSERT INTO vec_contexto_actor_v1.certificado_firmante_nominal_versiones
 (vinculo_ref,version,certificado_der_sha256,cuenta_ref,cuenta_version,cuenta_huella_sha256,
 persona_ref,persona_version,persona_huella_sha256,vinculo_cuenta_persona_ref,vinculo_cuenta_persona_version,vinculo_cuenta_persona_huella_sha256,
 estado,vigente_desde,vigente_hasta,evidencia_ref,evidencia_sha256,decision_ref,auditoria_ref,recibo_ref,clave,descriptor_canonico,documento_canonico,huella_sha256,registrada_en)
 VALUES(d->>'vinculo_ref',(d->>'version')::numeric,d->>'certificado_der_sha256',d->>'cuenta_ref',(s#>>'{cuenta,version}')::numeric,s#>>'{cuenta,huella_sha256}',
 d->>'persona_ref',(s#>>'{persona,version}')::numeric,s#>>'{persona,huella_sha256}',d->>'vinculo_cuenta_persona_ref',(s#>>'{vinculo_cuenta_persona,version}')::numeric,s#>>'{vinculo_cuenta_persona,huella_sha256}',
 d->>'estado',(d->>'vigente_desde')::timestamptz,(d->>'vigente_hasta')::timestamptz,d->>'evidencia_ref',d->>'evidencia_sha256',decision,auditoria,recibo,d->>'clave',b,doc,sha,fecha);
 INSERT INTO vec_contexto_actor_v1.certificado_firmante_nominal_actual AS x VALUES(d->>'certificado_der_sha256',d->>'vinculo_ref',(d->>'version')::numeric,sha)
 ON CONFLICT(certificado_der_sha256) DO UPDATE SET vinculo_ref=excluded.vinculo_ref,version=excluded.version,huella_sha256=excluded.huella_sha256
 WHERE x.vinculo_ref=a.vinculo_ref AND x.version=a.version AND x.huella_sha256=a.huella_sha256;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA25: CAS perdido' USING ERRCODE='40001'; END IF;
 RETURN result;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(bytea,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(bytea,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(der text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v record;s jsonb;org jsonb;recibo jsonb;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR der IS NULL OR der !~ '^[0-9a-f]{64}$' OR der=pg_catalog.repeat('0',64)
 THEN RETURN NULL; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:certificado_firmante_v2:'||der,0));
 SELECT x.* INTO v FROM vec_contexto_actor_v1.certificado_firmante_nominal_actual a
 JOIN vec_contexto_actor_v1.certificado_firmante_nominal_versiones x USING(certificado_der_sha256,vinculo_ref,version,huella_sha256)
 WHERE a.certificado_der_sha256=der FOR SHARE OF a,x;
 IF NOT FOUND OR v.estado<>'vigente' OR v.certificado_der_sha256<>der
  OR pg_catalog.encode(pg_catalog.sha256(v.documento_canonico),'hex')<>v.huella_sha256
  OR (pg_catalog.convert_from(v.documento_canonico,'UTF8')::jsonb)->>'descriptor_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(v.descriptor_canonico),'hex')
 THEN RETURN NULL; END IF;
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(v.cuenta_ref,v.persona_ref,v.vinculo_cuenta_persona_ref);
 recibo:=pg_catalog.convert_from(v.documento_canonico,'UTF8')::jsonb;
 org:=vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
  v.cuenta_ref,v.persona_ref,v.vinculo_cuenta_persona_ref,v.vinculo_cuenta_persona_version,
  (pg_catalog.convert_from(v.descriptor_canonico,'UTF8')::jsonb)->>'organizacion_ref');
 ahora:=pg_catalog.clock_timestamp();
 IF s IS NULL OR org IS NULL OR recibo->'organizacion_destino' IS DISTINCT FROM org
  OR ahora<v.vigente_desde OR ahora>=v.vigente_hasta
  OR (s#>>'{cuenta,version}')::numeric<>v.cuenta_version OR s#>>'{cuenta,huella_sha256}'<>v.cuenta_huella_sha256
  OR (s#>>'{persona,version}')::numeric<>v.persona_version OR s#>>'{persona,huella_sha256}'<>v.persona_huella_sha256
  OR (s#>>'{vinculo_cuenta_persona,version}')::numeric<>v.vinculo_cuenta_persona_version
  OR s#>>'{vinculo_cuenta_persona,huella_sha256}'<>v.vinculo_cuenta_persona_huella_sha256
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('esquema','vec.contexto-actor.certificado-firmante-ct.v2',
  'certificado_der_sha256',der,'persona_ref',v.persona_ref,'cuenta',s->'cuenta','persona',s->'persona',
  'vinculo_cuenta_persona',(s->'vinculo_cuenta_persona')-'perfil_ref','organizacion_destino',org,'estado','vigente',
  'vinculo_certificado',pg_catalog.jsonb_build_object('referencia',v.vinculo_ref,'version',v.version,'huella_sha256',v.huella_sha256,
   'cuenta_ref',v.cuenta_ref,'persona_ref',v.persona_ref,'certificado_der_sha256',v.certificado_der_sha256,
   'estado',v.estado,'vigente_desde',v.vigente_desde,'vigente_hasta',v.vigente_hasta,
   'evidencia_ref',v.evidencia_ref,'evidencia_sha256',v.evidencia_sha256,
   'recibo_ref',v.recibo_ref,'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref));
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text) TO vec_autorizacion_propietario;

-- El histórico conserva los bytes originales. Sólo AUT puede recuperarlo
-- dentro de una operación autorizada; esta función no concede autoridad.
CREATE FUNCTION vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(ref text,ver numeric,sha text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE x record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR vec_contexto_actor_v1.referencia_valida(ref,'vcc_') IS NOT TRUE
  OR ver IS NULL OR ver<1 OR pg_catalog.scale(ver)<>0
  OR sha IS NULL OR sha !~ '^[0-9a-f]{64}$' THEN RETURN NULL; END IF;
 SELECT * INTO x FROM vec_contexto_actor_v1.certificado_firmante_nominal_versiones h
 WHERE h.vinculo_ref=ref AND h.version=ver AND h.huella_sha256=sha FOR SHARE;
 IF NOT FOUND OR pg_catalog.encode(pg_catalog.sha256(x.documento_canonico),'hex')<>x.huella_sha256
  OR (pg_catalog.convert_from(x.documento_canonico,'UTF8')::jsonb)->>'descriptor_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(x.descriptor_canonico),'hex')
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('esquema','vec.contexto-actor.certificado-firmante-historia.v2',
  'vinculo_ref',x.vinculo_ref,'version',x.version,'huella_sha256',x.huella_sha256,
  'organizacion_destino',(pg_catalog.convert_from(x.documento_canonico,'UTF8')::jsonb)->'organizacion_destino',
  'descriptor_canonico_base64',pg_catalog.encode(x.descriptor_canonico,'base64'),
  'documento_canonico_base64',pg_catalog.encode(x.documento_canonico,'base64'),
  'cuenta_ref',x.cuenta_ref,'cuenta_version',x.cuenta_version,'cuenta_huella_sha256',x.cuenta_huella_sha256,
  'persona_ref',x.persona_ref,'persona_version',x.persona_version,'persona_huella_sha256',x.persona_huella_sha256,
  'vinculo_cuenta_persona_ref',x.vinculo_cuenta_persona_ref,'vinculo_cuenta_persona_version',x.vinculo_cuenta_persona_version,
  'vinculo_cuenta_persona_huella_sha256',x.vinculo_cuenta_persona_huella_sha256);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(text,numeric,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.recuperar_historia_certificado_firmante_ct_v2(text,numeric,text) TO vec_autorizacion_propietario;
COMMIT;
