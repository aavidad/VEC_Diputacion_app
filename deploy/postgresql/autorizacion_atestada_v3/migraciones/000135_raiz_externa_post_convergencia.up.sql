\set ON_ERROR_STOP on
-- BORRADOR NO INSTALABLE. AD3-135 sustituirá hacia delante AD3-123.
-- Entrada causal: AD3-133 nueva, sobre #222; AD124, Documentos11 y Usuarios15
-- son posteriores. AD134/136 no forman parte de la preimagen.
-- La versión final cotejará la postimagen GLOBAL de AD133 en ambos linajes,
-- más CHECK de audiencias, membresías, TEMP, OID, ACL, propietario y dependencias.
-- Sólo después añadirá la raíz externa propia y sustituirá los cuatro lectores
-- delimitados, sin alterar cuerpos AD111/114 ni registrar AD123 como ejecutada.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '90s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo', 0));
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000135', 0));
DO $preimagen$
DECLARE
    nombre pg_catalog.text;
    -- Estas matrices se rellenarán únicamente con las mediciones completas
    -- de los dos linajes después de AD133 nueva. Ninguna huella heredada.
    ampliada pg_catalog.jsonb := NULL;
    historica pg_catalog.jsonb := NULL;
    postimagenes pg_catalog.jsonb := NULL;
    frontera_a pg_catalog.jsonb := NULL;
    frontera_b pg_catalog.jsonb := NULL;
    frontera_actual pg_catalog.jsonb;
    actual pg_catalog.jsonb;
    catalogo_antes pg_catalog.jsonb;
    e pg_catalog.jsonb;
    lector pg_catalog.record;
    antes pg_catalog.pg_proc%ROWTYPE;
    despues pg_catalog.pg_proc%ROWTYPE;
    fuente pg_catalog.text;
    definicion pg_catalog.text;
    a pg_catalog.text;
    b pg_catalog.text;
    dependencias_antes pg_catalog.jsonb;
    dependencias_despues pg_catalog.jsonb;
    linaje_elegido pg_catalog.text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('server_version_num')::pg_catalog.int4
          NOT BETWEEN 180000 AND 189999
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'AD3-135: migrador o PG18 no autorizados'
          USING ERRCODE = '55000';
    END IF;
    -- Detectar incluso una instalación parcial antes de cualquier escritura.
    FOREACH nombre IN ARRAY ARRAY[
      'puntero_configuracion_externa',
      'puntero_clave_emision_externa',
      'checkpoint_gobierno_externo'] LOOP
        IF pg_catalog.to_regclass('vec_autorizacion_atestada_v3.' || nombre)
           IS NOT NULL
           OR pg_catalog.to_regtype('vec_autorizacion_atestada_v3.' || nombre)
              IS NOT NULL THEN
            RAISE EXCEPTION 'AD3-135: raíz externa previa o parcial'
              USING ERRCODE = '55000';
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc AS p
               WHERE p.pronamespace =
                 pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
                 AND p.proname = ANY (ARRAY[
                   'avanzar_checkpoint_externo',
                   'leer_estado_publicacion_externa_v1',
                   'material_publico_externo_v1',
                   'preparar_publicacion_externa_v1',
                   'publicar_confianza_externa_v1'])) THEN
        RAISE EXCEPTION 'AD3-135: funciones externas previas o parciales'
          USING ERRCODE = '55000';
    END IF;
    -- Antes de completar esta barrera deben cotejarse ambos inventarios
    -- íntegros (funciones, CHECK, ACL de tipos, membresías y TEMP/PUBLIC),
    -- medir las ocho postimágenes de lectores y comprobar negativas.
    IF ampliada IS NULL OR historica IS NULL OR postimagenes IS NULL
       OR frontera_a IS NULL OR frontera_b IS NULL THEN
        RAISE EXCEPTION 'AD3-135: borrador sin matrices PG18 post-AD133'
          USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.jsonb_array_length(postimagenes) IS DISTINCT FROM 8
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_to_recordset(postimagenes)
           AS x(linaje pg_catalog.text,firma pg_catalog.text)
           WHERE x.linaje='a') IS DISTINCT FROM 4
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_to_recordset(postimagenes)
           AS x(linaje pg_catalog.text,firma pg_catalog.text)
           WHERE x.linaje='b') IS DISTINCT FROM 4
       OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_to_recordset(postimagenes)
           AS x(linaje pg_catalog.text,firma pg_catalog.text)
           WHERE x.firma NOT IN (
             'vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)',
             'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
             'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
             'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)')) THEN
        RAISE EXCEPTION 'AD3-135: matriz de cuatro lectores por linaje incompleta'
          USING ERRCODE='55000';
    END IF;
SELECT pg_catalog.jsonb_agg(entrada ORDER BY entrada->>'firma') INTO actual FROM (
WITH sig(firma) AS (SELECT x->>'firma' FROM pg_catalog.jsonb_array_elements(ampliada) x)
select jsonb_build_object(
 'firma',s.firma,'sha_prosrc',encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),
 'owner',pg_get_userbyid(p.proowner),'language',l.lanname,
 'argtypes',(select jsonb_agg(format_type(t.x,null) order by t.i) from unnest(p.proargtypes::oid[]) with ordinality t(x,i)),
 'allargtypes',(select jsonb_agg(format_type(t.x,null) order by t.i) from unnest(p.proallargtypes) with ordinality t(x,i)),
 'return',format_type(p.prorettype,null),'variadic',case when p.provariadic=0 then null else format_type(p.provariadic,null) end,
 'pg_proc',to_jsonb(p)-array['oid','pronamespace','proowner','prolang','proargtypes','proallargtypes','prorettype','provariadic','prosrc','proacl'],
 'acl_effective',coalesce((select jsonb_agg(jsonb_build_object('grantee',coalesce(gr.rolname,'PUBLIC'),'grantor',go.rolname,'privilege',a.privilege_type,'grantable',a.is_grantable) order by coalesce(gr.rolname,'PUBLIC'),go.rolname,a.privilege_type,a.is_grantable) from aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a left join pg_roles gr on gr.oid=a.grantee left join pg_roles go on go.oid=a.grantor),'[]'::jsonb),
 'pg_depend',coalesce((select jsonb_agg(jsonb_build_object('class',d.classid::regclass::text,'objsubid',d.objsubid,'refclass',d.refclassid::regclass::text,'refobjsubid',d.refobjsubid,'deptype',d.deptype,'target',to_jsonb(i)) order by d.classid::regclass::text,d.objsubid,d.refclassid::regclass::text,d.refobjsubid,d.deptype,i.identity) from pg_depend d cross join lateral pg_identify_object(d.refclassid,d.refobjid,d.refobjsubid) i where d.classid='pg_proc'::regclass and d.objid=p.oid),'[]'::jsonb),
 'pg_shdepend',coalesce((select jsonb_agg(jsonb_build_object('class',d.classid::regclass::text,'objsubid',d.objsubid,'refclass',d.refclassid::regclass::text,'refobjsubid',0,'deptype',d.deptype,'target',to_jsonb(i)) order by d.classid::regclass::text,d.objsubid,d.refclassid::regclass::text,d.deptype,i.identity) from pg_shdepend d cross join lateral pg_identify_object(d.refclassid,d.refobjid,0) i where d.classid='pg_proc'::regclass and d.objid=p.oid and d.dbid=(select oid from pg_database where datname=current_database())),'[]'::jsonb)
) from sig s join pg_proc p on p.oid=to_regprocedure(s.firma) join pg_language l on l.oid=p.prolang
) AS conjunto(entrada);
 IF pg_catalog.jsonb_array_length(actual) IS DISTINCT FROM pg_catalog.jsonb_array_length(ampliada)
 OR pg_catalog.jsonb_array_length(ampliada) IS DISTINCT FROM pg_catalog.jsonb_array_length(historica)
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
     WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3'))
    IS DISTINCT FROM pg_catalog.jsonb_array_length(ampliada)
 OR (actual IS DISTINCT FROM ampliada AND actual IS DISTINCT FROM historica)
 THEN RAISE EXCEPTION 'AD3-135: preimagen global incompatible' USING ERRCODE='55000'; END IF;
    SELECT pg_catalog.jsonb_build_object(
      'audiencias',(SELECT pg_catalog.jsonb_build_object(
        'definicion',pg_catalog.pg_get_constraintdef(c.oid,true),
        'metadatos',pg_catalog.to_jsonb(c)-ARRAY[
          'oid','connamespace','conrelid','conbin'])
        FROM pg_catalog.pg_constraint AS c
        WHERE c.conrelid=pg_catalog.to_regclass(
          'vec_autorizacion_atestada_v3.clave_capacidad_version')
          AND c.conname='clave_capacidad_version_audiencia_consumo_check'),
      'tipos',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'nombre',t.typname,'propietario',pg_catalog.pg_get_userbyid(t.typowner),
        'acl',pg_catalog.coalesce((SELECT pg_catalog.jsonb_agg(
           pg_catalog.jsonb_build_object(
             'grantee',pg_catalog.coalesce(r.rolname,'PUBLIC'),
             'privilege',a.privilege_type,'grantable',a.is_grantable)
           ORDER BY pg_catalog.coalesce(r.rolname,'PUBLIC'),
                    a.privilege_type,a.is_grantable)
           FROM pg_catalog.aclexplode(pg_catalog.coalesce(
             t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
           LEFT JOIN pg_catalog.pg_roles AS r ON r.oid=a.grantee),
           '[]'::pg_catalog.jsonb)) ORDER BY t.typname)
         FROM pg_catalog.pg_type AS t
         WHERE t.typnamespace=pg_catalog.to_regnamespace(
           'vec_autorizacion_atestada_v3') AND t.typtype='c'),
      'membresias',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'rol',g.rolname,'miembro',m.rolname,'admin',a.admin_option,
        'inherit',a.inherit_option,'set',a.set_option)
        ORDER BY g.rolname,m.rolname)
        FROM pg_catalog.pg_auth_members AS a
        JOIN pg_catalog.pg_roles AS g ON g.oid=a.roleid
        JOIN pg_catalog.pg_roles AS m ON m.oid=a.member
        WHERE pg_catalog.left(g.rolname,4)='vec_'
           OR pg_catalog.left(m.rolname,4)='vec_'),
      'public_temp',(SELECT pg_catalog.coalesce(pg_catalog.bool_or(
        a.grantee=0 AND a.privilege_type='TEMPORARY'),false)
        FROM pg_catalog.pg_database AS d
        LEFT JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
          d.datacl,pg_catalog.acldefault('d',d.datdba))) AS a ON true
        WHERE d.datname=pg_catalog.current_database())) INTO frontera_actual;
    IF frontera_actual->'audiencias' IS NULL
       OR frontera_actual->'audiencias'='null'::pg_catalog.jsonb
       OR frontera_actual->>'public_temp' IS DISTINCT FROM 'false'
       OR frontera_actual IS DISTINCT FROM
          CASE WHEN actual=ampliada THEN frontera_a ELSE frontera_b END THEN
       RAISE EXCEPTION 'AD3-135: CHECK, tipos, membresías o TEMP incompatibles'
         USING ERRCODE='55000';
    END IF;
 linaje_elegido:=CASE WHEN actual=ampliada THEN 'a' ELSE 'b' END;
 PERFORM pg_catalog.set_config('vec.ad135.linaje',linaje_elegido,true);
END $preimagen$;

-- Cada DDL se envía como sentencia propia: los CREATE posteriores dependen
-- de tablas y funciones creadas antes. La barrera anterior detiene este
-- borrador antes de la primera sentencia con efecto.

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE TABLE vec_autorizacion_atestada_v3.puntero_configuracion_externa (
 orden pg_catalog.numeric(20,0) PRIMARY KEY CHECK(orden BETWEEN 1 AND 9007199254740991),
 configuracion_revision pg_catalog.text NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.configuracion_confianza_version,
 establecida_en pg_catalog.timestamptz(6) NOT NULL,
 acto_ref pg_catalog.text NOT NULL UNIQUE,
 registrada_en pg_catalog.timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 huella_aprobacion_sha256 pg_catalog.text NOT NULL CHECK(vec_autorizacion_atestada_v3.huella_sha256_valida(huella_aprobacion_sha256)),
 preimagen_sha256 pg_catalog.text NOT NULL CHECK(vec_autorizacion_atestada_v3.huella_sha256_valida(preimagen_sha256)),
 material_publico pg_catalog.jsonb NOT NULL
);
CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision_externa (
 orden pg_catalog.numeric(20,0) PRIMARY KEY CHECK(orden BETWEEN 1 AND 9007199254740991),
 clave_id pg_catalog.text NOT NULL,
 version pg_catalog.numeric(20,0) NOT NULL UNIQUE,
 establecida_en pg_catalog.timestamptz(6) NOT NULL,
 acto_ref pg_catalog.text NOT NULL UNIQUE,
 registrada_en pg_catalog.timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(clave_id,version) REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version
);
CREATE TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno_externo (
 control_id pg_catalog.bool PRIMARY KEY DEFAULT true CHECK(control_id),
 revision pg_catalog.numeric(20,0) NOT NULL CHECK(revision BETWEEN 0 AND 9007199254740991),
 configuracion_secuencia_minima pg_catalog.numeric(20,0) NOT NULL CHECK(configuracion_secuencia_minima BETWEEN 0 AND 9007199254740991),
 raiz_version_minima pg_catalog.numeric(20,0) NOT NULL CHECK(raiz_version_minima BETWEEN 0 AND 9007199254740991),
 actualizada_en pg_catalog.timestamptz(6) NOT NULL
);
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno_externo VALUES(true,0,0,0,pg_catalog.clock_timestamp());
DO $proteccion$
DECLARE t pg_catalog.text;
BEGIN
 FOREACH t IN ARRAY ARRAY['puntero_configuracion_externa','puntero_clave_emision_externa','checkpoint_gobierno_externo'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I FOR ALL TO vec_autorizacion_atestada_v3_propietario USING(current_user=''vec_autorizacion_atestada_v3_propietario'') WITH CHECK(current_user=''vec_autorizacion_atestada_v3_propietario'')',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON vec_autorizacion_atestada_v3.%I FROM PUBLIC',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.%I FROM PUBLIC',t);
 END LOOP;
END $proteccion$;
CREATE TRIGGER no_mutar BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.puntero_clave_emision_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_mutar BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.puntero_configuracion_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();

CREATE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo()
RETURNS pg_catalog.trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r pg_catalog.numeric; c pg_catalog.numeric;
BEGIN
 SELECT cr.raiz_version,cfg.secuencia INTO STRICT r,c
 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
 JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cfg ON cfg.revision=cr.configuracion_revision
 WHERE cr.configuracion_revision=NEW.configuracion_revision;
 UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno_externo
 SET revision=revision+1,configuracion_secuencia_minima=c,raiz_version_minima=r,actualizada_en=pg_catalog.clock_timestamp()
 WHERE control_id AND configuracion_secuencia_minima<c AND raiz_version_minima<=r;
 IF NOT FOUND THEN RAISE EXCEPTION 'AD3-135: retroceso externo' USING ERRCODE='42501'; END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo() FROM PUBLIC;
CREATE TRIGGER checkpoint_despues AFTER INSERT ON vec_autorizacion_atestada_v3.puntero_configuracion_externa FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint_externo();

-- Preparación y publicación usan la autoridad existente, después de SET LOCAL ROLE.
-- ACL solo propietario; ningún emisor, consumidor o preflight puede ejecutar estas funciones.
CREATE FUNCTION vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1()
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE estado pg_catalog.jsonb; t pg_catalog.text; filas pg_catalog.jsonb; actual pg_catalog.jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD3-135: gobierno rechazado' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Los máximos globales conservan las restricciones UNIQUE históricas. La
 -- preimagen minimizada también liga asociaciones, revocaciones y checkpoints.
 estado:='{}'::pg_catalog.jsonb;
 FOREACH t IN ARRAY ARRAY['clave_capacidad_version','puntero_clave_emision','puntero_clave_emision_externa','configuracion_confianza_version','raiz_confianza_version','configuracion_raiz','puntero_configuracion_actual','puntero_configuracion_externa','revocacion_clave_capacidad','revocacion_configuracion','revocacion_raiz','checkpoint_gobierno','checkpoint_gobierno_externo'] LOOP
  EXECUTE pg_catalog.format('SELECT coalesce(pg_catalog.jsonb_agg(f ORDER BY f::pg_catalog.text),''[]''::pg_catalog.jsonb) FROM (SELECT pg_catalog.to_jsonb(x)-''secreto_hmac'' AS f FROM vec_autorizacion_atestada_v3.%I x) q',t) INTO filas;
  estado:=estado||pg_catalog.jsonb_build_object(t,filas);
 END LOOP;
 SELECT material_publico INTO actual FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa ORDER BY orden DESC LIMIT 1;
 RETURN pg_catalog.jsonb_build_object(
 'preimagen_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(estado::pg_catalog.text,'UTF8')),'hex'),
 'configuracion_secuencia_siguiente',(SELECT coalesce(pg_catalog.max(secuencia),0)+1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),
 'clave_version_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'revision_gobierno_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'clave_orden_siguiente',(SELECT greatest(coalesce(pg_catalog.max(version),0),coalesce(pg_catalog.max(revision_gobierno),0),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),(SELECT coalesce(pg_catalog.max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa))+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
 'publicacion_externa_actual',actual);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.material_publico_externo_v1(m pg_catalog.jsonb)
RETURNS pg_catalog.jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT pg_catalog.jsonb_build_object('configuracion',m->'configuracion','raiz',m->'raiz','claves',
 (SELECT pg_catalog.jsonb_agg(k-'secreto_hmac_hex' ORDER BY k->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') k))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.material_publico_externo_v1(pg_catalog.jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m pg_catalog.jsonb)
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; audiencias pg_catalog.text[]:=ARRAY[
 'vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1',
 'vec_usuarios.correos.consultar.externa_personal.v1','vec_usuarios.correos.anadir.externa_personal.v1','vec_usuarios.correos.reenviar.externa_personal.v1','vec_usuarios.correos.verificar.externa_personal.v1','vec_usuarios.correos.activar.externa_personal.v1','vec_usuarios.correos.retirar.externa_personal.v1',
 'vec_usuarios.imagen.consultar.externa_personal.v1','vec_usuarios.imagen.actualizar.externa_personal.v1',
 'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1','vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1','vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1','vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'];
BEGIN
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();
 IF pg_catalog.octet_length(m::pg_catalog.text)>262144 OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(m))<>3
 OR NOT(m ?& ARRAY['configuracion','raiz','claves']) THEN RAISE EXCEPTION 'AD3-135: material rechazado' USING ERRCODE='42501'; END IF;
 c:=m->'configuracion';r:=m->'raiz';
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
 OR NOT(c ?& ARRAY['revision','secuencia','huella_configuracion_sha256','publicada_en','expira_en'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(c) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='secuencia' THEN 'number' ELSE 'string' END)
 OR pg_catalog.jsonb_typeof(r) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(r))<>8
 OR NOT(r ?& ARRAY['clave_id','version','clave_publica_spki_hex','huella_spki_sha256','valida_desde','valida_hasta','suite','audiencia_despliegue'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(r) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='version' THEN 'number' ELSE 'string' END)
 OR NOT pg_catalog.starts_with(c->>'revision','confianza:atestacion:externo:')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(c->>'huella_configuracion_sha256')
 OR (c->>'secuencia')!~'^[1-9][0-9]{0,15}$'
 OR (r->>'version')!~'^[1-9][0-9]{0,15}$'
 OR r->>'clave_id' IS DISTINCT FROM 'clave:atestacion:externo:'||(r->>'huella_spki_sha256')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(r->>'huella_spki_sha256')
 OR (r->>'clave_publica_spki_hex')!~'^302a300506032b6570032100[0-9a-f]{64}$'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(r->>'clave_publica_spki_hex','hex')),'hex') IS DISTINCT FROM r->>'huella_spki_sha256'
 OR r->>'suite' IS DISTINCT FROM 'VEC-AD-3-COSE-EDDSA-1'
 OR r->>'audiencia_despliegue' IS DISTINCT FROM 'vec:desarrollo:contratacion-temporal:atestacion:v3'
 OR NOT pg_catalog.isfinite((c->>'publicada_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((c->>'expira_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_desde')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_hasta')::pg_catalog.timestamptz)
 OR (c->>'publicada_en')::pg_catalog.timestamptz>pg_catalog.clock_timestamp()
 OR (c->>'expira_en')::pg_catalog.timestamptz<=pg_catalog.clock_timestamp()
 OR (r->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
 OR (r->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.puntero_configuracion_actual p ON p.configuracion_revision=cr.configuracion_revision WHERE (cr.raiz_clave_id,cr.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version x WHERE x.huella_spki_sha256=r->>'huella_spki_sha256' AND (x.clave_id,x.version) IS DISTINCT FROM (r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE (x.raiz_clave_id,x.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=c->>'revision')
 OR pg_catalog.jsonb_typeof(m->'claves') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(m->'claves')<>17
 THEN RAISE EXCEPTION 'AD3-135: material rechazado' USING ERRCODE='42501'; END IF;
 IF (SELECT pg_catalog.count(DISTINCT u.item->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>17
 THEN RAISE EXCEPTION 'AD3-135: audiencias rechazadas' USING ERRCODE='42501'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_array_elements(m->'claves') x LOOP
  IF pg_catalog.jsonb_typeof(k) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(k))<>11
  OR NOT(k ?& ARRAY['clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac_hex','huella_secreto_sha256','emisor_id','audiencia_consumo','valida_desde','valida_hasta','orden'])
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(k) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key=ANY(ARRAY['version','revision_gobierno','orden']) THEN 'number' ELSE 'string' END)
  OR k->>'audiencia_consumo'<>ALL(audiencias)
  OR NOT vec_autorizacion_atestada_v3.texto_tecnico_valido(k->>'clave_id',512)
  OR NOT pg_catalog.starts_with(k->>'emisor_id','emisor:externo:')
  OR (k->>'version')!~'^[1-9][0-9]{0,15}$' OR (k->>'revision_gobierno')!~'^[1-9][0-9]{0,15}$' OR (k->>'orden')!~'^[1-9][0-9]{0,15}$'
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_gobierno_sha256')
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_secreto_sha256')
  OR (k->>'secreto_hmac_hex')!~'^[0-9a-f]+$' OR pg_catalog.length(k->>'secreto_hmac_hex') NOT BETWEEN 64 AND 8192 OR pg_catalog.length(k->>'secreto_hmac_hex')%2<>0
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(k->>'secreto_hmac_hex','hex')),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR (k->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
  OR (k->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
  OR NOT pg_catalog.isfinite((k->>'valida_desde')::pg_catalog.timestamptz) OR NOT pg_catalog.isfinite((k->>'valida_hasta')::pg_catalog.timestamptz)
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad x WHERE (x.clave_id,x.version)=(k->>'clave_id',(k->>'version')::pg_catalog.numeric))
  THEN RAISE EXCEPTION 'AD3-135: clave externa rechazada' USING ERRCODE='42501'; END IF;
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('preimagen_sha256',e->>'preimagen_sha256',
 'huella_aprobacion_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion_atestada_v3.material_publico_externo_v1(m)::pg_catalog.text,'UTF8')),'hex'));
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'AD3-135: propuesta externa rechazada' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(pg_catalog.jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(m pg_catalog.jsonb,aprobacion pg_catalog.text,preimagen pg_catalog.text)
RETURNS pg_catalog.jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; anterior pg_catalog.jsonb; acto pg_catalog.text; previa_aprobacion pg_catalog.text; previa_preimagen pg_catalog.text;
BEGIN
 e:=vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m);
 IF aprobacion IS NULL OR preimagen IS NULL OR aprobacion IS DISTINCT FROM e->>'huella_aprobacion_sha256'
 THEN RAISE EXCEPTION 'AD3-135: aprobación o CAS rechazado' USING ERRCODE='42501'; END IF;
 c:=m->'configuracion';r:=m->'raiz';acto:='acto:externo:confianza:r'||(c->>'secuencia');
 SELECT material_publico,huella_aprobacion_sha256,preimagen_sha256 INTO anterior,previa_aprobacion,previa_preimagen FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa WHERE configuracion_revision=c->>'revision';
 IF FOUND THEN
  IF aprobacion IS DISTINCT FROM previa_aprobacion OR (preimagen IS DISTINCT FROM previa_preimagen AND preimagen IS DISTINCT FROM e->>'preimagen_sha256')
  OR anterior IS DISTINCT FROM vec_autorizacion_atestada_v3.material_publico_externo_v1(m)
  OR (SELECT configuracion_revision FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa ORDER BY orden DESC LIMIT 1) IS DISTINCT FROM c->>'revision'
  THEN RAISE EXCEPTION 'AD3-135: replay externo rechazado' USING ERRCODE='42501'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(anterior->'claves') x WHERE
   (SELECT (p.clave_id,p.version,p.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k ON (k.clave_id,k.version)=(p.clave_id,p.version) WHERE k.audiencia_consumo=x->>'audiencia_consumo' ORDER BY p.orden DESC LIMIT 1)
   IS DISTINCT FROM (x->>'clave_id',(x->>'version')::pg_catalog.numeric,(x->>'orden')::pg_catalog.numeric))
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo cp WHERE cp.control_id AND cp.configuracion_secuencia_minima=(c->>'secuencia')::pg_catalog.numeric AND cp.raiz_version_minima<=(r->>'version')::pg_catalog.numeric)
  THEN RAISE EXCEPTION 'AD3-135: replay de puntero rechazado' USING ERRCODE='42501'; END IF;
  RETURN anterior;
 END IF;
 IF preimagen IS DISTINCT FROM e->>'preimagen_sha256' THEN RAISE EXCEPTION 'AD3-135: CAS rechazado' USING ERRCODE='42501'; END IF;
 IF (c->>'secuencia')::pg_catalog.numeric<(SELECT configuracion_secuencia_minima+1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo WHERE control_id)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version WHERE revision=c->>'revision')
 OR (c->>'secuencia')::pg_catalog.numeric<>(vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1()->>'configuracion_secuencia_siguiente')::pg_catalog.numeric
 THEN RAISE EXCEPTION 'AD3-135: secuencia externa rechazada' USING ERRCODE='42501'; END IF;
 -- ON CONFLICT sólo permite el mismo material. Nunca sustituye una fila histórica.
 INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
 VALUES(r->>'clave_id',(r->>'version')::pg_catalog.numeric,pg_catalog.decode(r->>'clave_publica_spki_hex','hex'),r->>'huella_spki_sha256',(r->>'valida_desde')::pg_catalog.timestamptz,(r->>'valida_hasta')::pg_catalog.timestamptz,r->>'suite',r->>'audiencia_despliegue','acto:externo:raiz:'||(r->>'huella_spki_sha256')) ON CONFLICT(clave_id,version) DO NOTHING;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version x WHERE x.clave_id=r->>'clave_id' AND x.version=(r->>'version')::pg_catalog.numeric AND x.clave_publica_spki=pg_catalog.decode(r->>'clave_publica_spki_hex','hex') AND x.huella_spki_sha256=r->>'huella_spki_sha256' AND x.valida_desde=(r->>'valida_desde')::pg_catalog.timestamptz AND x.valida_hasta=(r->>'valida_hasta')::pg_catalog.timestamptz AND x.suite=r->>'suite' AND x.audiencia_despliegue=r->>'audiencia_despliegue') THEN RAISE EXCEPTION 'AD3-135: raíz histórica distinta' USING ERRCODE='42501'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_array_elements(m->'claves') x LOOP
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::pg_catalog.numeric,(k->>'revision_gobierno')::pg_catalog.numeric,k->>'huella_gobierno_sha256',pg_catalog.decode(k->>'secreto_hmac_hex','hex'),k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia_consumo',(k->>'valida_desde')::pg_catalog.timestamptz,(k->>'valida_hasta')::pg_catalog.timestamptz,'acto:externo:clave:r'||(k->>'revision_gobierno')) ON CONFLICT(clave_id,version) DO NOTHING;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' AND x.version=(k->>'version')::pg_catalog.numeric AND x.revision_gobierno=(k->>'revision_gobierno')::pg_catalog.numeric AND x.huella_gobierno_sha256=k->>'huella_gobierno_sha256' AND x.huella_secreto_sha256=k->>'huella_secreto_sha256' AND x.secreto_hmac=pg_catalog.decode(k->>'secreto_hmac_hex','hex') AND x.emisor_id=k->>'emisor_id' AND x.audiencia_consumo=k->>'audiencia_consumo' AND x.valida_desde=(k->>'valida_desde')::pg_catalog.timestamptz AND x.valida_hasta=(k->>'valida_hasta')::pg_catalog.timestamptz) THEN RAISE EXCEPTION 'AD3-135: clave histórica distinta' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision_externa(orden,clave_id,version,establecida_en,acto_ref)
  VALUES((k->>'orden')::pg_catalog.numeric,k->>'clave_id',(k->>'version')::pg_catalog.numeric,(c->>'publicada_en')::pg_catalog.timestamptz,'acto:externo:puntero-clave:r'||(k->>'orden')) ON CONFLICT(orden) DO NOTHING;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p WHERE p.orden=(k->>'orden')::pg_catalog.numeric AND p.clave_id=k->>'clave_id' AND p.version=(k->>'version')::pg_catalog.numeric)
  OR (SELECT (p.clave_id,p.version,p.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p JOIN vec_autorizacion_atestada_v3.clave_capacidad_version x ON (x.clave_id,x.version)=(p.clave_id,p.version) WHERE x.audiencia_consumo=k->>'audiencia_consumo' ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM (k->>'clave_id',(k->>'version')::pg_catalog.numeric,(k->>'orden')::pg_catalog.numeric) THEN RAISE EXCEPTION 'AD3-135: puntero de clave distinto' USING ERRCODE='42501'; END IF;
 END LOOP;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(c->>'revision',(c->>'secuencia')::pg_catalog.numeric,c->>'huella_configuracion_sha256',(c->>'publicada_en')::pg_catalog.timestamptz,(c->>'expira_en')::pg_catalog.timestamptz,acto);
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES(c->>'revision',r->>'clave_id',(r->>'version')::pg_catalog.numeric);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_externa(orden,configuracion_revision,establecida_en,acto_ref,huella_aprobacion_sha256,preimagen_sha256,material_publico)
 VALUES((c->>'secuencia')::pg_catalog.numeric,c->>'revision',(c->>'publicada_en')::pg_catalog.timestamptz,acto||':puntero',aprobacion,preimagen,vec_autorizacion_atestada_v3.material_publico_externo_v1(m));
 RETURN vec_autorizacion_atestada_v3.material_publico_externo_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.publicar_confianza_externa_v1(pg_catalog.jsonb,pg_catalog.text,pg_catalog.text) FROM PUBLIC;

RESET ROLE;

DO $lectores$
DECLARE
 postimagenes pg_catalog.jsonb := NULL; -- medir tras AD133 nueva
 catalogo_antes pg_catalog.jsonb;
 e pg_catalog.jsonb;
 lector pg_catalog.record;
 antes pg_catalog.pg_proc%ROWTYPE;
 despues pg_catalog.pg_proc%ROWTYPE;
 fuente pg_catalog.text;
 definicion pg_catalog.text;
 a pg_catalog.text;
 b pg_catalog.text;
 dependencias_antes pg_catalog.jsonb;
 dependencias_despues pg_catalog.jsonb;
 linaje_elegido pg_catalog.text := pg_catalog.current_setting('vec.ad135.linaje',true);
BEGIN
 IF linaje_elegido NOT IN ('a','b') OR postimagenes IS NULL THEN
  RAISE EXCEPTION 'AD3-135: postimagen de lectores sin medir' USING ERRCODE='55000';
 END IF;
 -- Catálogo local completo: los OID no se comparan entre clones diferentes.
 SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(p) ORDER BY p.oid) INTO catalogo_antes
 FROM pg_catalog.pg_proc p WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3');
 SELECT pg_catalog.jsonb_build_object(
 'locales',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(catalogo_antes) x)),
 'compartidas',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
 FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(catalogo_antes) x))) INTO dependencias_antes;
 -- Efectos de gobierno de AD123: mismo esquema y contratos, tipos temporales cerrados.

 FOR lector IN SELECT * FROM pg_catalog.jsonb_to_recordset(postimagenes)
 AS x(linaje pg_catalog.text,firma pg_catalog.text,antes pg_catalog.text,despues pg_catalog.text,punteros pg_catalog.int4,checkpoints pg_catalog.int4,claves pg_catalog.int4)
 WHERE x.linaje=linaje_elegido LOOP
  SELECT p.* INTO STRICT antes FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(lector.firma);
  SELECT x INTO STRICT e FROM pg_catalog.jsonb_array_elements(catalogo_antes) x WHERE (x->>'oid')::pg_catalog.oid=antes.oid;
  IF pg_catalog.to_jsonb(antes) IS DISTINCT FROM e
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(antes.prosrc,'UTF8')),'hex') IS DISTINCT FROM lector.antes
  THEN RAISE EXCEPTION 'AD3-135: lector modificado tras preimagen %',lector.firma USING ERRCODE='55000'; END IF;
  fuente:=antes.prosrc;
  a:='vec_autorizacion_atestada_v3.puntero_configuracion_actual';b:='vec_autorizacion_atestada_v3.puntero_configuracion_externa';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*lector.punteros THEN RAISE EXCEPTION 'AD3-135: punteros incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  a:='vec_autorizacion_atestada_v3.checkpoint_gobierno';b:='vec_autorizacion_atestada_v3.checkpoint_gobierno_externo';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*lector.checkpoints THEN RAISE EXCEPTION 'AD3-135: checkpoints incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  a:='vec_autorizacion_atestada_v3.puntero_clave_emision';b:='vec_autorizacion_atestada_v3.puntero_clave_emision_externa';
  IF (pg_catalog.length(fuente)-pg_catalog.length(pg_catalog.replace(fuente,a,'')))<>pg_catalog.length(a)*lector.claves THEN RAISE EXCEPTION 'AD3-135: claves incompatibles' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(fuente,a,b);
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM lector.despues
  THEN RAISE EXCEPTION 'AD3-135: postimagen incompatible' USING ERRCODE='55000'; END IF;
  -- La sustitución inversa reproduce íntegro el cuerpo, incluidos AD111/114.
  fuente:=pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(fuente,
   'vec_autorizacion_atestada_v3.puntero_configuracion_externa','vec_autorizacion_atestada_v3.puntero_configuracion_actual'),
   'vec_autorizacion_atestada_v3.checkpoint_gobierno_externo','vec_autorizacion_atestada_v3.checkpoint_gobierno'),
   'vec_autorizacion_atestada_v3.puntero_clave_emision_externa','vec_autorizacion_atestada_v3.puntero_clave_emision');
  IF fuente IS DISTINCT FROM antes.prosrc THEN RAISE EXCEPTION 'AD3-135: cuerpo no reversible' USING ERRCODE='55000'; END IF;
  fuente:=pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(fuente,
   'vec_autorizacion_atestada_v3.puntero_configuracion_actual','vec_autorizacion_atestada_v3.puntero_configuracion_externa'),
   'vec_autorizacion_atestada_v3.checkpoint_gobierno','vec_autorizacion_atestada_v3.checkpoint_gobierno_externo'),
   'vec_autorizacion_atestada_v3.puntero_clave_emision','vec_autorizacion_atestada_v3.puntero_clave_emision_externa');
  definicion:=pg_catalog.pg_get_functiondef(antes.oid);
  IF (pg_catalog.length(definicion)-pg_catalog.length(pg_catalog.replace(definicion,antes.prosrc,'')))<>pg_catalog.length(antes.prosrc)
  THEN RAISE EXCEPTION 'AD3-135: definición incompatible' USING ERRCODE='55000'; END IF;
  EXECUTE pg_catalog.replace(definicion,antes.prosrc,fuente);
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=antes.oid;
  IF despues.prosrc IS DISTINCT FROM fuente OR pg_catalog.to_jsonb(despues)-'prosrc' IS DISTINCT FROM e-'prosrc'
  THEN RAISE EXCEPTION 'AD3-135: metadatos alterados' USING ERRCODE='55000'; END IF;
 END LOOP;
 -- Ninguna función anterior desaparece ni cambia fuera de esos cuatro cuerpos.
 FOR e IN SELECT x FROM pg_catalog.jsonb_array_elements(catalogo_antes) x LOOP
  SELECT p.* INTO STRICT despues FROM pg_catalog.pg_proc p WHERE p.oid=(e->>'oid')::pg_catalog.oid;
  IF EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(postimagenes) x WHERE x->>'linaje'=linaje_elegido AND pg_catalog.to_regprocedure(x->>'firma')=despues.oid) THEN
   IF pg_catalog.to_jsonb(despues)-'prosrc' IS DISTINCT FROM e-'prosrc'
   THEN RAISE EXCEPTION 'AD3-135: metadatos históricos alterados' USING ERRCODE='55000'; END IF;
  ELSIF pg_catalog.to_jsonb(despues) IS DISTINCT FROM e THEN
   RAISE EXCEPTION 'AD3-135: núcleo histórico alterado' USING ERRCODE='55000';
  END IF;
 END LOOP;
 SELECT pg_catalog.jsonb_build_object(
 'locales',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(catalogo_antes) x)),
 'compartidas',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
 FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())
 AND d.objid IN(SELECT (x->>'oid')::pg_catalog.oid FROM pg_catalog.jsonb_array_elements(catalogo_antes) x))) INTO dependencias_despues;
 IF dependencias_despues IS DISTINCT FROM dependencias_antes
 THEN RAISE EXCEPTION 'AD3-135: dependencias históricas alteradas' USING ERRCODE='55000'; END IF;
END $lectores$;
COMMIT;
