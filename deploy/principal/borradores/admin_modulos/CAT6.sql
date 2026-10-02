\set ON_ERROR_STOP on
-- BORRADOR CAT6 / reserva catalogos_configurables 000006. NO INSTALAR.
-- Gobierno central del catálogo administracion.modulos; CAT5/Cronos permanece ajeno.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000006',0));
DO $pendiente$
BEGIN
 -- Retirar sólo después de dos revisiones y ensayo de la postimagen AD151 real.
 RAISE EXCEPTION 'CAT6: borrador pendiente de configuración institucional, AD152 y ensayo' USING ERRCODE='55000';
END $pendiente$;
-- Provisión DBA separada: no crea LOGIN ni concesiones de ADMIN.
DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_catalogos_configurables_ejecutor_modulos') IS NOT NULL THEN
   RAISE EXCEPTION 'CAT6: provisión de rol incompatible' USING ERRCODE='55000'; END IF;
 CREATE ROLE vec_catalogos_configurables_ejecutor_modulos NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
END $rol$;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL timezone='UTC';
DO $pre$
BEGIN
 IF pg_catalog.to_regprocedure('vec_catalogos_configurables.rechazar_cambio_inmutable()') IS NULL
    OR pg_catalog.to_regclass('vec_catalogos_configurables.modulos_historia') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname='vec_catalogos_configurables'
        AND nspowner=current_user::regrole)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user
        AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)) THEN
   RAISE EXCEPTION 'CAT6: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario REVOKE ALL ON TYPES FROM PUBLIC;

-- Configuración offline revisada. El registro son IDs del ensamblaje aprobado,
-- no el catálogo HTTP /modules ni una lista que pueda enviar el formulario.
CREATE TABLE vec_catalogos_configurables.modulos_configuracion (
 version integer PRIMARY KEY CHECK(version BETWEEN 1 AND 1000000),
 canonico_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(canonico_exacto) BETWEEN 2 AND 65536),
 huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 registro_version_ref text NOT NULL CHECK(pg_catalog.octet_length(registro_version_ref) BETWEEN 3 AND 512),
 registro_sha256 text NOT NULL CHECK(registro_sha256 ~ '^[0-9a-f]{64}$'),
 perfil_admin_ref text NOT NULL CHECK(pg_catalog.octet_length(perfil_admin_ref) BETWEEN 3 AND 512),
 aprobacion_ref text NOT NULL CHECK(pg_catalog.octet_length(aprobacion_ref) BETWEEN 3 AND 512),
 registrados jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(registrados)='array'),
 gobernados jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(gobernados)='array'),
 provisionada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK(pg_catalog.encode(pg_catalog.sha256(canonico_exacto),'hex')=huella_sha256),
 UNIQUE(version,huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.modulos_configuracion_cabeza (
 catalogo_id text PRIMARY KEY CHECK(catalogo_id='administracion.modulos'),
 version integer NOT NULL, huella_sha256 text NOT NULL,
 FOREIGN KEY(version,huella_sha256) REFERENCES vec_catalogos_configurables.modulos_configuracion(version,huella_sha256)
);
-- Provisión offline: autoridad real de la sesión SQL y aprobación externa.
-- No representa una decisión V3 ni la firma legal de esa aprobación.
CREATE TABLE vec_catalogos_configurables.modulos_configuracion_provision (
 version integer PRIMARY KEY REFERENCES vec_catalogos_configurables.modulos_configuracion(version),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 version_anterior integer NOT NULL CHECK(version_anterior>=0), huella_anterior text,
 actor_sql text NOT NULL CHECK(pg_catalog.octet_length(actor_sql) BETWEEN 1 AND 256),
 autoridad text NOT NULL CHECK(autoridad='sql_offline_propietario'),
 aprobacion_ref text NOT NULL CHECK(pg_catalog.octet_length(aprobacion_ref) BETWEEN 3 AND 512),
 registro_version_ref text NOT NULL, registro_sha256 text NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:[0-9a-f-]{36}$'),
 provisionada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(version,huella_sha256) REFERENCES vec_catalogos_configurables.modulos_configuracion(version,huella_sha256),
 CHECK((version_anterior=0 AND huella_anterior IS NULL) OR (version_anterior>0 AND huella_anterior ~ '^[0-9a-f]{64}$'))
);
CREATE TABLE vec_catalogos_configurables.modulos_configuracion_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.modulos_configuracion_provision(recibo_ref),
 evento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
-- Cada revisión y cada transición terminal conserva los bytes centrales de Go.
-- Publicar y retirar mantienen Revision según CatalogoConfigurable.
CREATE TABLE vec_catalogos_configurables.modulos_historia (
 catalogo_id text NOT NULL CHECK(catalogo_id='administracion.modulos'),
 version integer NOT NULL CHECK(version BETWEEN 1 AND 1000000),
 revision integer NOT NULL CHECK(revision BETWEEN 1 AND 1000000),
 estado text NOT NULL CHECK(estado IN ('borrador','publicado','retirado')),
 canonico_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(canonico_exacto) BETWEEN 2 AND 1048576),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 configuracion_version integer NOT NULL REFERENCES vec_catalogos_configurables.modulos_configuracion(version),
 actor_ref text NOT NULL CHECK(pg_catalog.octet_length(actor_ref) BETWEEN 3 AND 512),
 perfil_ref text NOT NULL CHECK(pg_catalog.octet_length(perfil_ref) BETWEEN 3 AND 512),
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(catalogo_id,version,revision,estado),
 UNIQUE(catalogo_id,version,revision,estado,huella_sha256),
 CHECK(pg_catalog.encode(pg_catalog.sha256(canonico_exacto),'hex')=huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.modulos_cabeza (
 catalogo_id text PRIMARY KEY CHECK(catalogo_id='administracion.modulos'),
 publicada_version integer, publicada_revision integer, publicada_estado text, publicada_huella text,
 borrador_version integer, borrador_revision integer, borrador_huella text,
 borrador_estado text GENERATED ALWAYS AS (CASE WHEN borrador_version IS NOT NULL THEN 'borrador' END) STORED,
 CHECK((publicada_version IS NULL AND publicada_revision IS NULL AND publicada_estado IS NULL AND publicada_huella IS NULL)
    OR (publicada_version IS NOT NULL AND publicada_revision IS NOT NULL AND publicada_estado IN ('publicado','retirado') AND publicada_huella IS NOT NULL)),
 CHECK((borrador_version IS NULL AND borrador_revision IS NULL AND borrador_huella IS NULL)
    OR (borrador_version IS NOT NULL AND borrador_revision IS NOT NULL AND borrador_huella IS NOT NULL)),
 FOREIGN KEY(catalogo_id,publicada_version,publicada_revision,publicada_estado,publicada_huella)
    REFERENCES vec_catalogos_configurables.modulos_historia(catalogo_id,version,revision,estado,huella_sha256),
 FOREIGN KEY(catalogo_id,borrador_version,borrador_revision,borrador_estado,borrador_huella)
    REFERENCES vec_catalogos_configurables.modulos_historia(catalogo_id,version,revision,estado,huella_sha256)
);
CREATE TABLE vec_catalogos_configurables.modulos_recibo (
 actor_ref text NOT NULL, catalogo_id text NOT NULL CHECK(catalogo_id='administracion.modulos'),
 clave_operacion text NOT NULL CHECK(pg_catalog.octet_length(clave_operacion) BETWEEN 3 AND 160),
 operacion text NOT NULL CHECK(operacion IN ('crear','actualizar','publicar','retirar')),
 material_exacto bytea NOT NULL CHECK(pg_catalog.octet_length(material_exacto) BETWEEN 2 AND 2097152),
 material_semantico bytea NOT NULL CHECK(pg_catalog.octet_length(material_semantico) BETWEEN 2 AND 65536),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^recibo:[0-9a-f-]{36}$'),
 version integer NOT NULL, revision integer NOT NULL, estado text NOT NULL, huella_sha256 text NOT NULL,
 decision_ref text NOT NULL UNIQUE, auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 confirmada_en timestamptz(6) NOT NULL,
 recibo_canonico bytea NOT NULL, recibo_sha256 text NOT NULL,
 PRIMARY KEY(actor_ref,catalogo_id,clave_operacion),
 FOREIGN KEY(catalogo_id,version,revision,estado,huella_sha256)
    REFERENCES vec_catalogos_configurables.modulos_historia(catalogo_id,version,revision,estado,huella_sha256),
 CHECK(pg_catalog.encode(pg_catalog.sha256(material_semantico),'hex')=material_sha256),
 CHECK(pg_catalog.encode(pg_catalog.sha256(recibo_canonico),'hex')=recibo_sha256)
);
-- Evidencia local complementaria; auditoria_ref enlaza autoridad V3 segregada.
CREATE TABLE vec_catalogos_configurables.modulos_outbox (
 recibo_ref text PRIMARY KEY REFERENCES vec_catalogos_configurables.modulos_recibo(recibo_ref),
 traza jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(traza)='object'),
 evento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(evento)='object'),
 creada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['modulos_configuracion','modulos_configuracion_cabeza','modulos_configuracion_provision','modulos_configuracion_outbox','modulos_historia','modulos_cabeza','modulos_recibo','modulos_outbox'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_catalogos_configurables.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY %I ON vec_catalogos_configurables.%I TO vec_catalogos_configurables_propietario USING(true) WITH CHECK(true)',t||'_propietario',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM PUBLIC',t);
  IF t IN ('modulos_configuracion','modulos_configuracion_provision','modulos_configuracion_outbox','modulos_historia','modulos_recibo','modulos_outbox') THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',t||'_inmutable',t);
  ELSE
   EXECUTE pg_catalog.format('CREATE TRIGGER %I BEFORE DELETE OR TRUNCATE ON vec_catalogos_configurables.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable()',t||'_no_borrar',t);
  END IF;
 END LOOP;
END $tablas$;

-- Única provisión offline, no concedida al ejecutor. Huella de la configuración
-- anterior y número esperado impiden sustituir aprobaciones concurrentes.
CREATE FUNCTION vec_catalogos_configurables.provisionar_configuracion_modulos_v1(
 p_version_esperada integer,p_huella_esperada text,p_canonico bytea,p_huella text
) RETURNS integer LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE c jsonb; anterior record; nueva integer; id jsonb; recibo text;
BEGIN
 IF p_canonico IS NULL OR pg_catalog.octet_length(p_canonico) NOT BETWEEN 2 AND 65536
    OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
    OR pg_catalog.encode(pg_catalog.sha256(p_canonico),'hex') IS DISTINCT FROM p_huella
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 999999 THEN
  RAISE EXCEPTION 'CAT6: configuración offline inválida' USING ERRCODE='22023'; END IF;
 c:=pg_catalog.convert_from(p_canonico,'UTF8')::jsonb; nueva:=p_version_esperada+1;
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR c IS DISTINCT FROM pg_catalog.jsonb_build_object('catalogo_id',c->'catalogo_id','version',c->'version',
      'registro_version_ref',c->'registro_version_ref','registro_sha256',c->'registro_sha256','perfil_admin_ref',c->'perfil_admin_ref',
      'aprobacion_ref',c->'aprobacion_ref','registrados',c->'registrados','gobernados',c->'gobernados')
    OR c->>'catalogo_id' IS DISTINCT FROM 'administracion.modulos' OR c->>'version' IS DISTINCT FROM nueva::text
    OR pg_catalog.octet_length(coalesce(c->>'registro_version_ref','')) NOT BETWEEN 3 AND 512
    OR coalesce(c->>'registro_sha256','') !~ '^[0-9a-f]{64}$'
    OR pg_catalog.octet_length(coalesce(c->>'perfil_admin_ref','')) NOT BETWEEN 3 AND 512
    OR pg_catalog.octet_length(coalesce(c->>'aprobacion_ref','')) NOT BETWEEN 3 AND 512
    OR c->>'perfil_admin_ref' IN ('ADMIN','admin','gestor_perfiles','perfil:gestor_perfiles')
    OR pg_catalog.jsonb_typeof(c->'registrados') IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_typeof(c->'gobernados') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'CAT6: aprobación nominal o ensamblaje ausentes' USING ERRCODE='22023'; END IF;
 IF pg_catalog.jsonb_array_length(c->'registrados') NOT BETWEEN 1 AND 128
    OR pg_catalog.jsonb_array_length(c->'gobernados') NOT BETWEEN 1 AND 128 THEN
  RAISE EXCEPTION 'CAT6: ensamblaje fuera de límites' USING ERRCODE='22023'; END IF;
 FOR id IN SELECT value FROM pg_catalog.jsonb_array_elements(c->'registrados') LOOP
  IF pg_catalog.jsonb_typeof(id) IS DISTINCT FROM 'string' OR (id#>>'{}') !~ '^[a-z][a-z0-9_.:-]{2,127}$'
     OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'registrados') r WHERE r.value=id)<>1 THEN
   RAISE EXCEPTION 'CAT6: registro inválido o duplicado' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR id IN SELECT value FROM pg_catalog.jsonb_array_elements(c->'gobernados') LOOP
  IF pg_catalog.jsonb_typeof(id) IS DISTINCT FROM 'string' OR NOT (c->'registrados' @> pg_catalog.jsonb_build_array(id))
     OR (id#>>'{}') IN ('administracion','ADMIN','admin','coreinfra','vec.module.administracion','vec.module.usuarios','vec.coreinfra')
     OR (id#>>'{}') ~ '(^|[.:])coreinfra([.:]|$)'
     OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'gobernados') r WHERE r.value=id)<>1 THEN
   RAISE EXCEPTION 'CAT6: módulo no gobernable' USING ERRCODE='22023'; END IF;
 END LOOP;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:modulos',0));
 SELECT * INTO anterior FROM vec_catalogos_configurables.modulos_configuracion_cabeza WHERE catalogo_id='administracion.modulos' FOR UPDATE;
 IF coalesce(anterior.version,0) IS DISTINCT FROM p_version_esperada
    OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada THEN
  RAISE EXCEPTION 'CAT6: CAS de configuración fallido' USING ERRCODE='40001'; END IF;
 IF EXISTS (SELECT 1 FROM vec_catalogos_configurables.modulos_cabeza WHERE borrador_version IS NOT NULL) THEN
  RAISE EXCEPTION 'CAT6: configuración con borrador pendiente' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_catalogos_configurables.modulos_configuracion(version,canonico_exacto,huella_sha256,registro_version_ref,registro_sha256,perfil_admin_ref,aprobacion_ref,registrados,gobernados)
 VALUES(nueva,p_canonico,p_huella,c->>'registro_version_ref',c->>'registro_sha256',c->>'perfil_admin_ref',c->>'aprobacion_ref',c->'registrados',c->'gobernados');
 INSERT INTO vec_catalogos_configurables.modulos_configuracion_cabeza VALUES('administracion.modulos',nueva,p_huella)
 ON CONFLICT(catalogo_id) DO UPDATE SET version=EXCLUDED.version,huella_sha256=EXCLUDED.huella_sha256;
 recibo:='recibo:'||pg_catalog.gen_random_uuid()::text;
 INSERT INTO vec_catalogos_configurables.modulos_configuracion_provision
   (version,huella_sha256,version_anterior,huella_anterior,actor_sql,autoridad,aprobacion_ref,registro_version_ref,registro_sha256,recibo_ref)
 VALUES(nueva,p_huella,p_version_esperada,p_huella_esperada,session_user,'sql_offline_propietario',c->>'aprobacion_ref',c->>'registro_version_ref',c->>'registro_sha256',recibo);
 INSERT INTO vec_catalogos_configurables.modulos_configuracion_outbox(recibo_ref,evento)
 VALUES(recibo,pg_catalog.jsonb_build_object('tipo','vec.catalogos.modulos.configuracion.provisionada','version',nueva,
   'huella_sha256',p_huella,'version_anterior',p_version_esperada,'huella_anterior',p_huella_esperada,
   'actor_sql',session_user,'autoridad','sql_offline_propietario','aprobacion_ref',c->>'aprobacion_ref',
   'registro_version_ref',c->>'registro_version_ref','registro_sha256',c->>'registro_sha256','recibo_ref',recibo));
 RETURN nueva;
END $f$;

CREATE FUNCTION vec_catalogos_configurables.configuracion_modulos_v1()
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
 SELECT pg_catalog.jsonb_build_object('version',c.version,'huella_sha256',c.huella_sha256,
  'registro_version_ref',c.registro_version_ref,'registro_sha256',c.registro_sha256,'perfil_admin_ref',c.perfil_admin_ref,'registrados',c.registrados,'gobernados',c.gobernados)
 FROM vec_catalogos_configurables.modulos_configuracion c JOIN vec_catalogos_configurables.modulos_configuracion_cabeza h
 ON h.version=c.version AND h.huella_sha256=c.huella_sha256 WHERE h.catalogo_id='administracion.modulos'
$f$;

CREATE FUNCTION vec_catalogos_configurables.confirmar_gobierno_modulos_v1(
 p_material_exacto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(catalogo_canonico bytea,recibo_canonico bytea,recibo_sha256 text,recuperada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; cap jsonb; cfg jsonb; canon bytea; op text; actor text; perfil text;
 v integer; r integer; h text; mh text; ch text; accion text; entrada jsonb; momento timestamptz;
 cab vec_catalogos_configurables.modulos_cabeza%ROWTYPE; anterior jsonb; pub jsonb; bor jsonb;
 previo vec_catalogos_configurables.modulos_recibo%ROWTYPE; usado record; recibo text;
 sem bytea; sj jsonb; neutral bytea; nj jsonb; semh text; rb bytea; rh text; confirmado timestamptz; original bytea;
BEGIN
 IF p_material_exacto IS NULL OR pg_catalog.octet_length(p_material_exacto) NOT BETWEEN 2 AND 2097152
    OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 65536
    OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 65536 THEN
  RAISE EXCEPTION 'CAT6: material fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN
  m:=pg_catalog.convert_from(p_material_exacto,'UTF8')::jsonb;
  sem:=pg_catalog.decode(m->>'material_semantico_base64','base64'); sj:=pg_catalog.convert_from(sem,'UTF8')::jsonb;
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb; cap:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
  canon:=pg_catalog.decode(m->>'catalogo_canonico_base64','base64'); c:=pg_catalog.convert_from(canon,'UTF8')::jsonb;
  v:=(c->>'version')::integer; r:=(c->>'revision')::integer;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CAT6: material inválido' USING ERRCODE='22023'; END;
 op:=m->>'operacion'; h:=m->>'huella_comun'; semh:=m->>'huella_material_sha256'; actor:=d->>'principal_id'; perfil:=d->>'perfil_activo_ref';
 mh:=pg_catalog.encode(pg_catalog.sha256(p_material_exacto),'hex');
 IF sem IS NULL OR pg_catalog.octet_length(sem) NOT BETWEEN 2 AND 65536
    OR semh IS NULL OR semh !~ '^[0-9a-f]{64}$' OR pg_catalog.encode(pg_catalog.sha256(sem),'hex') IS DISTINCT FROM semh
    OR pg_catalog.jsonb_typeof(sj) IS DISTINCT FROM 'object'
    OR sj IS DISTINCT FROM pg_catalog.jsonb_build_object('Esquema',sj->'Esquema','Accion',sj->'Accion','CatalogoID',sj->'CatalogoID',
       'Version',sj->'Version','ActorRef',sj->'ActorRef','Cabeza',sj->'Cabeza','Contenido',sj->'Contenido',
       'HuellaBorradorSHA256',sj->'HuellaBorradorSHA256','Finalidad',sj->'Finalidad','Motivo',sj->'Motivo','AprobacionRef',sj->'AprobacionRef')
    OR sj->>'Esquema' IS DISTINCT FROM 'vec.catalogos.operacion.v1'
    OR sj->>'Accion' IS DISTINCT FROM 'vec.catalogos.'||op OR sj->>'ActorRef' IS DISTINCT FROM actor
    OR sj->>'CatalogoID' IS DISTINCT FROM 'administracion.modulos' OR sj->>'Version' IS DISTINCT FROM v::text
    OR sj#>>'{Cabeza,CatalogoID}' IS DISTINCT FROM 'administracion.modulos'
    OR sj->'Cabeza' IS DISTINCT FROM pg_catalog.jsonb_build_object('CatalogoID','administracion.modulos','Version',m#>'{cabeza_publicada_esperada,version}',
       'HuellaSHA256',m#>'{cabeza_publicada_esperada,huella_sha256}','Estado',m#>'{cabeza_publicada_esperada,estado}')
    OR sj->>'Finalidad' IS DISTINCT FROM m#>>'{traza,purpose}'
    OR sj->>'Motivo' IS DISTINCT FROM (CASE op WHEN 'crear' THEN c->>'motivo_creacion' WHEN 'actualizar' THEN c->>'motivo_modificacion' WHEN 'publicar' THEN c->>'motivo_publicacion' ELSE c->>'motivo_retirada' END)
    OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR op IS NULL OR op NOT IN ('crear','actualizar','publicar','retirar')
    OR pg_catalog.octet_length(coalesce(m->>'clave_operacion','')) NOT BETWEEN 3 AND 160
    OR v IS NULL OR v NOT BETWEEN 1 AND 1000000 OR r IS NULL OR r NOT BETWEEN 1 AND 1000000
    OR h IS NULL OR h !~ '^[0-9a-f]{64}$' OR pg_catalog.encode(pg_catalog.sha256(canon),'hex') IS DISTINCT FROM h
    OR canon IS NULL OR pg_catalog.octet_length(canon) NOT BETWEEN 2 AND 1048576
    OR c - ARRAY['id','version','revision','version_anterior_ref','modulo_id','nombre','descripcion','fuente_ref','motivo_creacion','entradas','estado',
       'creado_por','creado_en','ultima_modificacion_por','ultima_modificacion_en','motivo_modificacion','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion',
       'retirado_por','retirado_en','retirada_aprobacion_ref','motivo_retirada'] IS DISTINCT FROM '{}'::jsonb
    OR c->>'id' IS DISTINCT FROM 'administracion.modulos' OR c->>'modulo_id' IS DISTINCT FROM 'vec.module.administracion'
    OR c->>'version_anterior_ref' IS DISTINCT FROM (CASE WHEN v=1 THEN NULL ELSE 'administracion.modulos:'||(v-1)::text END)
    OR c->>'estado' IS DISTINCT FROM (CASE WHEN op IN ('crear','actualizar') THEN 'borrador' WHEN op='publicar' THEN 'publicado' ELSE 'retirado' END)
    OR pg_catalog.octet_length(coalesce(actor,'')) NOT BETWEEN 3 AND 512
    OR pg_catalog.octet_length(coalesce(perfil,'')) NOT BETWEEN 3 AND 512
    OR d->>'accion' IS DISTINCT FROM 'vec.catalogos.'||op OR d->>'modulo_id' IS DISTINCT FROM 'vec.module.administracion'
    OR d->>'recurso_ref' IS DISTINCT FROM 'administracion.modulos:'||v::text
    OR cap->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref' THEN
  RAISE EXCEPTION 'CAT6: contrato de gobierno inválido' USING ERRCODE='22023'; END IF;
 -- El recurso firmado liga los bytes del comando, incluidas ambas preimágenes.
 IF pg_catalog.octet_length(coalesce(c->>'nombre','')) NOT BETWEEN 1 AND 2048
    OR pg_catalog.octet_length(coalesce(c->>'descripcion',''))>32768
    OR pg_catalog.octet_length(coalesce(c->>'fuente_ref','')) NOT BETWEEN 1 AND 512
    OR pg_catalog.octet_length(coalesce(c->>'motivo_creacion','')) NOT BETWEEN 1 AND 32768 THEN
  RAISE EXCEPTION 'CAT6: metadatos centrales inválidos' USING ERRCODE='22023'; END IF;
 BEGIN
  momento:=(c->>'creado_en')::timestamptz;
  IF momento IS NULL OR NOT pg_catalog.isfinite(momento) OR momento='0001-01-01T00:00:00Z'::timestamptz THEN RAISE EXCEPTION 'fecha inválida'; END IF;
  IF op='publicar' AND ((c->>'publicado_en')::timestamptz IS NULL OR (c->>'publicado_en')::timestamptz<momento) THEN RAISE EXCEPTION 'fecha inválida'; END IF;
  IF op='retirar' AND ((c->>'retirado_en')::timestamptz IS NULL OR (c->>'retirado_en')::timestamptz<(c->>'publicado_en')::timestamptz) THEN RAISE EXCEPTION 'fecha inválida'; END IF;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CAT6: tiempos centrales inválidos' USING ERRCODE='22023'; END;
 ch:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  '{"ambitos":{},"atributos":{"estado":"'||(c->>'estado')||'","material_sha256":"'||mh||'","revision":"'||r::text||'"}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ch OR cap->>'huella_efecto_sha256' IS DISTINCT FROM ch THEN
  RAISE EXCEPTION 'CAT6: comando no ligado a V3' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:modulos',0));
 cfg:=vec_catalogos_configurables.configuracion_modulos_v1();
 IF cfg IS NULL OR cfg->>'version' IS DISTINCT FROM m->>'configuracion_version'
    OR cfg->>'huella_sha256' IS DISTINCT FROM m->>'configuracion_huella_sha256'
    OR cfg->>'registro_version_ref' IS DISTINCT FROM m->>'registro_version_ref'
    OR cfg->>'registro_sha256' IS DISTINCT FROM m->>'registro_sha256'
    OR cfg->>'perfil_admin_ref' IS DISTINCT FROM perfil THEN
  RAISE EXCEPTION 'CAT6: configuración o perfil ADMIN no vigentes' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_typeof(c->'entradas') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'CAT6: entradas inválidas' USING ERRCODE='22023'; END IF;
 IF pg_catalog.jsonb_array_length(c->'entradas')<>pg_catalog.jsonb_array_length(cfg->'gobernados') THEN
  RAISE EXCEPTION 'CAT6: catálogo incompleto' USING ERRCODE='22023'; END IF;
 FOR entrada IN SELECT value FROM pg_catalog.jsonb_array_elements(c->'entradas') LOOP
  IF pg_catalog.jsonb_typeof(entrada) IS DISTINCT FROM 'object'
     OR entrada - ARRAY['clave','etiqueta','descripcion','orden','vigente_desde','vigente_hasta','atributos'] IS DISTINCT FROM '{}'::jsonb
     OR NOT (cfg->'gobernados' @> pg_catalog.jsonb_build_array(entrada->>'clave'))
     OR entrada->'atributos' IS DISTINCT FROM pg_catalog.jsonb_build_object('habilitado',entrada#>'{atributos,habilitado}')
     OR entrada#>'{atributos,habilitado}' NOT IN ('"true"'::jsonb,'"false"'::jsonb)
     OR entrada#>'{atributos,habilitado}' IS NULL
     OR pg_catalog.octet_length(coalesce(entrada->>'etiqueta','')) NOT BETWEEN 1 AND 2048
     OR pg_catalog.octet_length(coalesce(entrada->>'descripcion',''))>32768
     OR coalesce(entrada->>'orden','') !~ '^[0-9]{1,9}$'
     OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'entradas') x WHERE x.value->>'clave'=entrada->>'clave')<>1 THEN
   RAISE EXCEPTION 'CAT6: entrada desconocida, duplicada o atributo inválido' USING ERRCODE='22023'; END IF;
  BEGIN
   momento:=(entrada->>'vigente_desde')::timestamptz;
   IF momento IS NULL OR NOT pg_catalog.isfinite(momento) OR momento='0001-01-01T00:00:00Z'::timestamptz
      OR (entrada->>'vigente_hasta' IS NOT NULL AND entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND (NOT pg_catalog.isfinite((entrada->>'vigente_hasta')::timestamptz)
          OR (entrada->>'vigente_hasta')::timestamptz<=momento)) THEN
    RAISE EXCEPTION 'vigencia inválida'; END IF;
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'CAT6: vigencia inválida' USING ERRCODE='22023'; END;
 END LOOP;
 accion:=CASE op WHEN 'crear' THEN 'vec.catalogos.borrador.creado' WHEN 'actualizar' THEN 'vec.catalogos.borrador.actualizado'
   WHEN 'publicar' THEN 'vec.catalogos.publicado' ELSE 'vec.catalogos.retirado' END;
 IF pg_catalog.jsonb_typeof(m->'traza') IS DISTINCT FROM 'object' OR pg_catalog.jsonb_typeof(m->'evento') IS DISTINCT FROM 'object'
    OR pg_catalog.octet_length((m->'traza')::text)>65536 OR pg_catalog.octet_length((m->'evento')::text)>65536
    OR m#>>'{traza,actor_id}' IS DISTINCT FROM actor OR m#>>'{traza,actor_profile}' IS DISTINCT FROM perfil
    OR m#>>'{evento,actor_id}' IS DISTINCT FROM actor OR m#>>'{traza,action}' IS DISTINCT FROM accion
    OR m#>>'{evento,type}' IS DISTINCT FROM accion OR m#>>'{traza,module_id}' IS DISTINCT FROM 'vec.module.administracion'
    OR m#>>'{evento,module_id}' IS DISTINCT FROM 'vec.module.administracion' OR m#>>'{traza,result}' IS DISTINCT FROM 'correcto'
    OR m#>>'{traza,subject_ref}' IS DISTINCT FROM d->>'recurso_ref' OR m#>>'{evento,subject_ref}' IS DISTINCT FROM d->>'recurso_ref'
    OR m#>>'{traza,after_hash}' IS DISTINCT FROM h OR m#>>'{evento,payload,huella_sha256}' IS DISTINCT FROM h
    OR (CASE op WHEN 'crear' THEN c->>'creado_por' WHEN 'actualizar' THEN c->>'ultima_modificacion_por'
         WHEN 'publicar' THEN c->>'publicado_por' ELSE c->>'retirado_por' END) IS DISTINCT FROM actor THEN
  RAISE EXCEPTION 'CAT6: actor, traza o evento incoherentes' USING ERRCODE='22023'; END IF;
 -- Bytes semánticos centrales verificados; contenido estructural exacto.
 IF op IN ('crear','actualizar') THEN
  IF sj->>'HuellaBorradorSHA256' IS DISTINCT FROM '' OR sj->>'AprobacionRef' IS DISTINCT FROM ''
     OR sj->'Contenido' IS DISTINCT FROM pg_catalog.jsonb_build_object(
       'CatalogoID',c->'id','Version',c->'version','Revision',c->'revision','VersionAnteriorRef',coalesce(c->'version_anterior_ref','""'::jsonb),
       'ModuloID',c->'modulo_id','Nombre',c->'nombre','Descripcion',coalesce(c->'descripcion','""'::jsonb),
       'FuenteRef',c->'fuente_ref','Entradas',c->'entradas') THEN
   RAISE EXCEPTION 'CAT6: contenido semántico incompatible' USING ERRCODE='22023'; END IF;
 ELSE
  IF sj->'Contenido' IS DISTINCT FROM 'null'::jsonb OR sj->>'HuellaBorradorSHA256' IS DISTINCT FROM
       (CASE WHEN op='publicar' THEN m#>>'{borrador_esperado,huella_sha256}' ELSE m#>>'{cabeza_publicada_esperada,huella_sha256}' END)
     OR sj->>'AprobacionRef' IS DISTINCT FROM (CASE WHEN op='publicar' THEN c->>'aprobacion_ref' ELSE c->>'retirada_aprobacion_ref' END) THEN
   RAISE EXCEPTION 'CAT6: aprobación semántica incompatible' USING ERRCODE='22023'; END IF;
 END IF;
 -- Recuperar por actor + catálogo + clave ANTES de CAS. Un cambio del comando
 -- bajo la misma clave nunca se convierte en otra operación ni otro recibo.
 SELECT * INTO previo FROM vec_catalogos_configurables.modulos_recibo
 WHERE actor_ref=actor AND catalogo_id='administracion.modulos' AND clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
  IF previo.operacion IS DISTINCT FROM op OR previo.material_semantico IS DISTINCT FROM sem
     OR previo.material_sha256 IS DISTINCT FROM semh THEN
   RAISE EXCEPTION 'CAT6: clave semántica en conflicto' USING ERRCODE='23505'; END IF;
  SELECT * INTO STRICT usado FROM vec_autorizacion_atestada_v3.consumir_gobierno_modulos_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
  SELECT x.canonico_exacto INTO STRICT original FROM vec_catalogos_configurables.modulos_historia x
   WHERE x.catalogo_id=previo.catalogo_id AND x.version=previo.version AND x.revision=previo.revision AND x.estado=previo.estado;
  RETURN QUERY SELECT original,previo.recibo_canonico,previo.recibo_sha256,true; RETURN;
 END IF;
 SELECT * INTO cab FROM vec_catalogos_configurables.modulos_cabeza WHERE catalogo_id='administracion.modulos' FOR UPDATE;
 IF cab.publicada_version IS NULL OR cab.publicada_estado IS DISTINCT FROM 'publicado' THEN
  RAISE EXCEPTION 'CAT6: cabeza inicial o retirada no habilitada para cambio' USING ERRCODE='55000'; END IF;
 pub:=CASE WHEN cab.publicada_version IS NULL THEN 'null'::jsonb ELSE pg_catalog.jsonb_build_object(
  'version',cab.publicada_version,'estado',cab.publicada_estado,'huella_sha256',cab.publicada_huella) END;
 bor:=CASE WHEN cab.borrador_version IS NULL THEN 'null'::jsonb ELSE pg_catalog.jsonb_build_object(
  'version',cab.borrador_version,'revision',cab.borrador_revision,'huella_sha256',cab.borrador_huella) END;
 IF m->'cabeza_publicada_esperada' IS DISTINCT FROM pub OR m->'borrador_esperado' IS DISTINCT FROM bor THEN
  RAISE EXCEPTION 'CAT6: CAS de cabeza o borrador fallido' USING ERRCODE='40001'; END IF;
 IF op='crear' THEN
  IF cab.publicada_version IS NULL OR cab.publicada_estado IS DISTINCT FROM 'publicado'
     OR cab.borrador_version IS NOT NULL OR v<>cab.publicada_version+1 OR r<>1
     OR c ? 'ultima_modificacion_por' OR c ? 'publicado_por' OR c ? 'retirado_por' THEN
   RAISE EXCEPTION 'CAT6: secuencia de creación inválida' USING ERRCODE='40001'; END IF;
 ELSE
  SELECT pg_catalog.convert_from(x.canonico_exacto,'UTF8')::jsonb INTO anterior
  FROM vec_catalogos_configurables.modulos_historia x WHERE x.catalogo_id='administracion.modulos'
    AND x.version=CASE WHEN op='retirar' THEN cab.publicada_version ELSE cab.borrador_version END
    AND x.revision=CASE WHEN op='retirar' THEN cab.publicada_revision ELSE cab.borrador_revision END
    AND x.estado=CASE WHEN op='retirar' THEN 'publicado' ELSE 'borrador' END;
  IF anterior IS NULL OR v<>(anterior->>'version')::integer OR c->>'creado_por' IS DISTINCT FROM anterior->>'creado_por'
     OR c->>'creado_en' IS DISTINCT FROM anterior->>'creado_en' THEN
   RAISE EXCEPTION 'CAT6: antecedente ausente o cambiado' USING ERRCODE='40001'; END IF;
  IF op='actualizar' THEN
   IF r<>(anterior->>'revision')::integer+1 OR c ? 'publicado_por' OR c ? 'retirado_por' THEN
    RAISE EXCEPTION 'CAT6: revisión de borrador inválida' USING ERRCODE='40001'; END IF;
  ELSIF op='publicar' THEN
   IF r<>(anterior->>'revision')::integer OR actor=anterior->>'creado_por'
      OR actor=coalesce(anterior->>'ultima_modificacion_por',anterior->>'creado_por')
      OR pg_catalog.octet_length(coalesce(c->>'aprobacion_ref','')) NOT BETWEEN 3 AND 512
      OR c - ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion'] IS DISTINCT FROM anterior - ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion']
      OR c ? 'retirado_por' THEN
    RAISE EXCEPTION 'CAT6: publicación exige otro actor y el borrador exacto' USING ERRCODE='42501'; END IF;
  ELSE
   IF cab.borrador_version IS NOT NULL OR r<>(anterior->>'revision')::integer OR actor=anterior->>'publicado_por'
      OR pg_catalog.octet_length(coalesce(c->>'retirada_aprobacion_ref','')) NOT BETWEEN 3 AND 512
      OR c - ARRAY['estado','retirado_por','retirado_en','retirada_aprobacion_ref','motivo_retirada'] IS DISTINCT FROM anterior - ARRAY['estado','retirado_por','retirado_en','retirada_aprobacion_ref','motivo_retirada'] THEN
    RAISE EXCEPTION 'CAT6: retirada incompatible' USING ERRCODE='40001'; END IF;
  END IF;
 END IF;
 -- Consumo, auditoría segregada, estado, historia, recibo y outbox: una TX.
 SELECT * INTO STRICT usado FROM vec_autorizacion_atestada_v3.consumir_gobierno_modulos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF usado.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'CAT6: consumo nuevo requerido' USING ERRCODE='42501'; END IF;
 recibo:='recibo:'||pg_catalog.gen_random_uuid()::text;
 confirmado:=(CASE op WHEN 'crear' THEN c->>'creado_en' WHEN 'actualizar' THEN c->>'ultima_modificacion_en' WHEN 'publicar' THEN c->>'publicado_en' ELSE c->>'retirado_en' END)::timestamptz;
 IF confirmado IS NULL OR NOT pg_catalog.isfinite(confirmado) OR confirmado<>pg_catalog.date_trunc('microseconds',confirmado) THEN
  RAISE EXCEPTION 'CAT6: instante original inválido' USING ERRCODE='22023'; END IF;
 rb:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('referencia',recibo,'clave_idempotencia',m->>'clave_operacion',
   'huella_material_sha256',semh,'accion','vec.catalogos.'||op,'catalogo_id','administracion.modulos','version',v,
   'huella_sha256',h,'estado',c->>'estado','actor_ref',actor,'auditoria_ref',usado.auditoria_ref,'outbox_ref',recibo,
   'confirmado_en',pg_catalog.to_char(confirmado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,'UTF8');
 rh:=pg_catalog.encode(pg_catalog.sha256(rb),'hex');
 INSERT INTO vec_catalogos_configurables.modulos_historia
 VALUES('administracion.modulos',v,r,c->>'estado',canon,h,(cfg->>'version')::integer,actor,perfil,recibo,usado.decision_ref,usado.consumida_en);
 INSERT INTO vec_catalogos_configurables.modulos_cabeza(catalogo_id) VALUES('administracion.modulos') ON CONFLICT DO NOTHING;
 IF op IN ('crear','actualizar') THEN
  UPDATE vec_catalogos_configurables.modulos_cabeza SET borrador_version=v,borrador_revision=r,borrador_huella=h WHERE catalogo_id='administracion.modulos';
 ELSE
  UPDATE vec_catalogos_configurables.modulos_cabeza SET publicada_version=v,publicada_revision=r,publicada_estado=c->>'estado',publicada_huella=h,
   borrador_version=NULL,borrador_revision=NULL,borrador_huella=NULL WHERE catalogo_id='administracion.modulos';
 END IF;
 INSERT INTO vec_catalogos_configurables.modulos_recibo
 VALUES(actor,'administracion.modulos',m->>'clave_operacion',op,p_material_exacto,sem,semh,recibo,v,r,c->>'estado',h,
  usado.decision_ref,usado.auditoria_ref,usado.consumo_huella_sha256,confirmado,rb,rh);
 INSERT INTO vec_catalogos_configurables.modulos_outbox(recibo_ref,traza,evento) VALUES(recibo,m->'traza',m->'evento');
 RETURN QUERY SELECT canon,rb,rh,false;
END $f$;

CREATE FUNCTION vec_catalogos_configurables.recuperar_gobierno_modulos_v1(
 p_material_exacto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(catalogo_canonico bytea,recibo_canonico bytea,recibo_sha256 text,recuperada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE m jsonb; d jsonb; cap jsonb; sem bytea; sj jsonb; cfg jsonb; actor text; op text; ch text; mh text;
 previo vec_catalogos_configurables.modulos_recibo%ROWTYPE; usado record; original bytea;
BEGIN
 IF p_material_exacto IS NULL OR pg_catalog.octet_length(p_material_exacto) NOT BETWEEN 2 AND 131072
    OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 65536
    OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 65536 THEN
  RAISE EXCEPTION 'CAT6: recuperación fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN
  m:=pg_catalog.convert_from(p_material_exacto,'UTF8')::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  cap:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
  sem:=pg_catalog.decode(m->>'material_semantico_base64','base64'); sj:=pg_catalog.convert_from(sem,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CAT6: recuperación inválida' USING ERRCODE='22023'; END;
 actor:=d->>'principal_id'; op:=m->>'operacion';
 IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR pg_catalog.jsonb_typeof(sj) IS DISTINCT FROM 'object'
    OR sj IS DISTINCT FROM pg_catalog.jsonb_build_object('Esquema',sj->'Esquema','Accion',sj->'Accion','CatalogoID',sj->'CatalogoID',
       'Version',sj->'Version','ActorRef',sj->'ActorRef','Cabeza',sj->'Cabeza','Contenido',sj->'Contenido',
       'HuellaBorradorSHA256',sj->'HuellaBorradorSHA256','Finalidad',sj->'Finalidad','Motivo',sj->'Motivo','AprobacionRef',sj->'AprobacionRef')
    OR sem IS NULL OR pg_catalog.octet_length(sem) NOT BETWEEN 2 AND 65536
    OR pg_catalog.encode(pg_catalog.sha256(sem),'hex') IS DISTINCT FROM m->>'huella_material_sha256'
    OR sj->>'Esquema' IS DISTINCT FROM 'vec.catalogos.operacion.v1'
    OR sj->>'ActorRef' IS DISTINCT FROM actor OR sj->>'Accion' IS DISTINCT FROM 'vec.catalogos.'||op
    OR sj->>'CatalogoID' IS DISTINCT FROM 'administracion.modulos' OR sj->>'Version' IS DISTINCT FROM m->>'version'
    OR m->>'catalogo_id' IS DISTINCT FROM 'administracion.modulos'
    OR pg_catalog.octet_length(coalesce(m->>'clave_operacion','')) NOT BETWEEN 3 AND 160
    OR op IS NULL OR op NOT IN ('crear','actualizar','publicar','retirar')
    OR coalesce(m->>'version','') !~ '^[1-9][0-9]{0,6}$' OR coalesce(m->>'revision','') !~ '^[1-9][0-9]{0,6}$'
    OR (m->>'version')::integer>1000000 OR (m->>'revision')::integer>1000000
    OR m->>'estado' IS DISTINCT FROM (CASE WHEN op IN ('crear','actualizar') THEN 'borrador' WHEN op='publicar' THEN 'publicado' ELSE 'retirado' END)
    OR d->>'accion' IS DISTINCT FROM sj->>'Accion' OR d->>'modulo_id' IS DISTINCT FROM 'vec.module.administracion'
    OR d->>'recurso_ref' IS DISTINCT FROM 'administracion.modulos:'||(m->>'version')
    OR cap->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref' THEN
  RAISE EXCEPTION 'CAT6: contrato de recuperación inválido' USING ERRCODE='42501'; END IF;
 mh:=pg_catalog.encode(pg_catalog.sha256(p_material_exacto),'hex');
 ch:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  '{"ambitos":{},"atributos":{"estado":"'||(m->>'estado')||'","material_sha256":"'||mh||'","revision":"'||(m->>'revision')||'"}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM ch OR cap->>'huella_efecto_sha256' IS DISTINCT FROM ch THEN
  RAISE EXCEPTION 'CAT6: recuperación no ligada a V3' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:modulos',0));
 cfg:=vec_catalogos_configurables.configuracion_modulos_v1();
 IF cfg IS NULL OR cfg->>'perfil_admin_ref' IS DISTINCT FROM d->>'perfil_activo_ref'
    OR cfg->>'version' IS DISTINCT FROM m->>'configuracion_version'
    OR cfg->>'huella_sha256' IS DISTINCT FROM m->>'configuracion_huella_sha256'
    OR cfg->>'registro_version_ref' IS DISTINCT FROM m->>'registro_version_ref'
    OR cfg->>'registro_sha256' IS DISTINCT FROM m->>'registro_sha256' THEN
  RAISE EXCEPTION 'CAT6: recuperación sin configuración vigente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT usado FROM vec_autorizacion_atestada_v3.consumir_gobierno_modulos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previo FROM vec_catalogos_configurables.modulos_recibo x
 WHERE x.actor_ref=actor AND x.catalogo_id='administracion.modulos' AND x.clave_operacion=m->>'clave_operacion';
 IF NOT FOUND THEN RETURN; END IF;
 IF previo.material_semantico IS DISTINCT FROM sem OR previo.material_sha256 IS DISTINCT FROM m->>'huella_material_sha256'
    OR previo.operacion IS DISTINCT FROM op OR previo.version::text IS DISTINCT FROM m->>'version'
    OR previo.revision::text IS DISTINCT FROM m->>'revision' OR previo.estado IS DISTINCT FROM m->>'estado' THEN
  RAISE EXCEPTION 'CAT6: clave de recuperación en conflicto' USING ERRCODE='23505'; END IF;
 SELECT x.canonico_exacto INTO STRICT original FROM vec_catalogos_configurables.modulos_historia x
 WHERE x.catalogo_id=previo.catalogo_id AND x.version=previo.version AND x.revision=previo.revision AND x.estado=previo.estado;
 RETURN QUERY SELECT original,previo.recibo_canonico,previo.recibo_sha256,true;
END $f$;

-- Consultas internas del puerto: nunca rutas HTTP ni autorización funcional.
CREATE FUNCTION vec_catalogos_configurables.obtener_catalogo_modulos_v1(p_id text,p_version integer,p_max_bytes integer)
RETURNS TABLE(catalogo_canonico bytea,huella_sha256 text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE x record;
BEGIN
 IF p_id IS DISTINCT FROM 'administracion.modulos' OR p_version IS NULL OR p_version NOT BETWEEN 1 AND 1000000
    OR p_max_bytes IS NULL OR p_max_bytes NOT BETWEEN 1 AND 4194304 THEN
  RAISE EXCEPTION 'CAT6: consulta exacta inválida' USING ERRCODE='22023'; END IF;
 SELECT h.canonico_exacto,h.huella_sha256 INTO x FROM vec_catalogos_configurables.modulos_historia h
 WHERE h.catalogo_id=p_id AND h.version=p_version
 ORDER BY CASE h.estado WHEN 'retirado' THEN 3 WHEN 'publicado' THEN 2 ELSE 1 END DESC,h.revision DESC LIMIT 1;
 IF NOT FOUND THEN RETURN; END IF;
 IF pg_catalog.octet_length(x.canonico_exacto)>p_max_bytes THEN
  RAISE EXCEPTION 'CAT6: consulta excede presupuesto' USING ERRCODE='54000'; END IF;
 RETURN QUERY SELECT x.canonico_exacto,x.huella_sha256;
END $f$;
CREATE FUNCTION vec_catalogos_configurables.obtener_cabeza_modulos_v1(p_id text,p_max_bytes integer)
RETURNS TABLE(catalogo_canonico bytea,huella_sha256 text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE x record;
BEGIN
 IF p_id IS DISTINCT FROM 'administracion.modulos' OR p_max_bytes IS NULL OR p_max_bytes NOT BETWEEN 1 AND 4194304 THEN
  RAISE EXCEPTION 'CAT6: consulta de cabeza inválida' USING ERRCODE='22023'; END IF;
 SELECT h.canonico_exacto,h.huella_sha256 INTO x FROM vec_catalogos_configurables.modulos_cabeza c
 JOIN vec_catalogos_configurables.modulos_historia h ON h.catalogo_id=c.catalogo_id AND h.version=c.publicada_version
  AND h.revision=c.publicada_revision AND h.estado=c.publicada_estado WHERE c.catalogo_id=p_id;
 IF NOT FOUND THEN RETURN; END IF;
 IF pg_catalog.octet_length(x.canonico_exacto)>p_max_bytes THEN
  RAISE EXCEPTION 'CAT6: cabeza excede presupuesto' USING ERRCODE='54000'; END IF;
 RETURN QUERY SELECT x.canonico_exacto,x.huella_sha256;
END $f$;
CREATE FUNCTION vec_catalogos_configurables.listar_catalogos_modulos_v1(p_id text,p_max_versiones integer,p_max_bytes integer)
RETURNS TABLE(catalogo_canonico bytea,huella_sha256 text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE total bigint; bytes bigint;
BEGIN
 IF p_id IS DISTINCT FROM 'administracion.modulos' OR p_max_versiones IS NULL OR p_max_versiones NOT BETWEEN 1 AND 64
    OR p_max_bytes IS NULL OR p_max_bytes NOT BETWEEN 1 AND 4194304 THEN
  RAISE EXCEPTION 'CAT6: lista fuera de límites' USING ERRCODE='22023'; END IF;
 SELECT count(*),coalesce(sum(pg_catalog.octet_length(x.canonico_exacto)),0) INTO total,bytes FROM (
  SELECT DISTINCT ON (h.version) h.version,h.canonico_exacto FROM vec_catalogos_configurables.modulos_historia h
  WHERE h.catalogo_id=p_id ORDER BY h.version,CASE h.estado WHEN 'retirado' THEN 3 WHEN 'publicado' THEN 2 ELSE 1 END DESC,h.revision DESC
  LIMIT p_max_versiones+1) x;
 IF total>p_max_versiones OR bytes>p_max_bytes THEN
  RAISE EXCEPTION 'CAT6: lista truncada; usar cabeza operativa' USING ERRCODE='54000'; END IF;
 RETURN QUERY SELECT x.canonico_exacto,x.huella_sha256 FROM (
  SELECT DISTINCT ON (h.version) h.version,h.canonico_exacto,h.huella_sha256 FROM vec_catalogos_configurables.modulos_historia h
  WHERE h.catalogo_id=p_id ORDER BY h.version,CASE h.estado WHEN 'retirado' THEN 3 WHEN 'publicado' THEN 2 ELSE 1 END DESC,h.revision DESC
  LIMIT p_max_versiones+1) x ORDER BY x.version;
END $f$;

-- Retirar también concesiones de ALTER DEFAULT PRIVILEGES ajenas a PUBLIC.
DO $cerrar_acl$
DECLARE x record; t text; f record;
BEGIN
 FOREACH t IN ARRAY ARRAY['modulos_configuracion','modulos_configuracion_cabeza','modulos_configuracion_provision','modulos_configuracion_outbox','modulos_historia','modulos_cabeza','modulos_recibo','modulos_outbox'] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_class c
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
   WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t) AND a.grantee<>c.relowner LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_catalogos_configurables.%I FROM %s',t,
     CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_type typ
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(typ.typacl,pg_catalog.acldefault('T',typ.typowner))) a
   WHERE typ.oid=pg_catalog.to_regtype('vec_catalogos_configurables.'||t) AND a.grantee<>typ.typowner LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_catalogos_configurables.%I FROM %s',t,
     CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
 END LOOP;
 FOR f IN SELECT p.oid,p.oid::regprocedure AS firma FROM pg_catalog.pg_proc p
  WHERE p.pronamespace='vec_catalogos_configurables'::regnamespace
   AND p.proname IN ('provisionar_configuracion_modulos_v1','configuracion_modulos_v1','confirmar_gobierno_modulos_v1',
       'recuperar_gobierno_modulos_v1','obtener_catalogo_modulos_v1','obtener_cabeza_modulos_v1','listar_catalogos_modulos_v1') LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f.oid AND a.grantee<>p.proowner LOOP
   EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f.firma,
     CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
  END LOOP;
 END LOOP;
END $cerrar_acl$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.provisionar_configuracion_modulos_v1(integer,text,bytea,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.configuracion_modulos_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_modulos_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.recuperar_gobierno_modulos_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.obtener_catalogo_modulos_v1(text,integer,integer),
 vec_catalogos_configurables.obtener_cabeza_modulos_v1(text,integer),vec_catalogos_configurables.listar_catalogos_modulos_v1(text,integer,integer) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_catalogos_configurables_ejecutor_modulos;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.confirmar_gobierno_modulos_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_catalogos_configurables.recuperar_gobierno_modulos_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_catalogos_configurables.obtener_catalogo_modulos_v1(text,integer,integer),vec_catalogos_configurables.obtener_cabeza_modulos_v1(text,integer),
 vec_catalogos_configurables.listar_catalogos_modulos_v1(text,integer,integer) TO vec_catalogos_configurables_ejecutor_modulos;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.configuracion_modulos_v1() TO vec_autorizacion_atestada_v3_propietario;
DO $acl_final$
DECLARE f record; permitido oid;
BEGIN
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_catalogos_configurables_ejecutor_modulos'::regrole)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname IN ('vec_catalogos_configurables_propietario','vec_catalogos_configurables_ejecutor_modulos')
       AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR rolinherit OR rolreplication)) THEN
  RAISE EXCEPTION 'CAT6: roles o herencias incompatibles' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT p.* FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_catalogos_configurables'::regnamespace
  AND p.proname IN ('provisionar_configuracion_modulos_v1','configuracion_modulos_v1','confirmar_gobierno_modulos_v1',
       'recuperar_gobierno_modulos_v1','obtener_catalogo_modulos_v1','obtener_cabeza_modulos_v1','listar_catalogos_modulos_v1') LOOP
  permitido:=CASE WHEN f.proname='provisionar_configuracion_modulos_v1' THEN f.proowner
    WHEN f.proname='configuracion_modulos_v1' THEN 'vec_autorizacion_atestada_v3_propietario'::regrole::oid
    ELSE 'vec_catalogos_configurables_ejecutor_modulos'::regrole::oid END;
  IF f.proowner<>'vec_catalogos_configurables_propietario'::regrole OR NOT f.prosecdef
     OR f.provolatile<>'v' OR f.proparallel<>'u'
     OR NOT ('search_path=pg_catalog, pg_temp'=ANY(f.proconfig))
     OR NOT pg_catalog.has_function_privilege(permitido,f.oid,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(f.proacl,pg_catalog.acldefault('f',f.proowner))) a
       WHERE a.grantee NOT IN (f.proowner,permitido) OR a.grantor<>f.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
   RAISE EXCEPTION 'CAT6: ACL o entorno incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl_final$;
COMMIT;
