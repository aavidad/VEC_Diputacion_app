BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000001:canon:v1', 0));
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog, vec_personal;

-- Escapado compatible con encoding/json para los textos que admite Material.
CREATE FUNCTION vec_personal.texto_json_go_v1(valor text) RETURNS text
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $$
DECLARE resultado text;
BEGIN
  -- PostgreSQL text no admite U+0000; tampoco se invoca chr(0), inválido.
  resultado := to_json(valor)::text;
  resultado := replace(replace(replace(resultado, '<', E'\\u003c'), '>', E'\\u003e'), '&', E'\\u0026');
  resultado := replace(replace(resultado, chr(8232), E'\\u2028'), chr(8233), E'\\u2029');
  RETURN resultado;
END $$;

CREATE FUNCTION vec_personal.campo_canonico_alta_v1(nombre text, valor text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $$
 SELECT octet_length(convert_to($1,'UTF8'))::text || ':' || $1 || octet_length(convert_to($2,'UTF8'))::text || ':' || $2
$$;

CREATE FUNCTION vec_personal.referencia_alta_ejercicio_valida_v1(valor text) RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $$
 SELECT $1 ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
$$;

CREATE FUNCTION vec_personal.fecha_civil_go_valida_v1(valor text) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $$
DECLARE a integer; m integer; d integer; limite integer;
BEGIN
 IF valor !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN RETURN false; END IF;
 a:=substring(valor FROM 1 FOR 4)::integer;
 m:=substring(valor FROM 6 FOR 2)::integer;
 d:=substring(valor FROM 9 FOR 2)::integer;
 IF m < 1 OR m > 12 OR d < 1 THEN RETURN false; END IF;
 limite:=CASE m WHEN 2 THEN CASE WHEN (a % 4 = 0 AND (a % 100 <> 0 OR a % 400 = 0)) THEN 29 ELSE 28 END WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31 END;
 RETURN d <= limite;
END $$;

CREATE FUNCTION vec_personal.solicitud_alta_ejercicio_canonica_v1(solicitud jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog AS $$
DECLARE claves text[]; f jsonb; clave text; entero jsonb; resultado text := '';
BEGIN
 IF jsonb_typeof(solicitud) IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
 END IF;
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_object_keys(solicitud) key;
 IF claves IS DISTINCT FROM ARRAY['capacidad_ref','contrato_version','correlacion_ref','esquema','expediente_ref','fuente_rpt','idempotencia_ref','plaza_ref','puesto_ref','solicitud_ref','version_expediente']
    OR (solicitud->>'esquema') IS DISTINCT FROM 'vec.contratacion-temporal.personal-rpt.alta.v1'
    OR jsonb_typeof(solicitud->'esquema') IS DISTINCT FROM 'string'
    OR jsonb_typeof(solicitud->'contrato_version') IS DISTINCT FROM 'number'
    OR (solicitud->>'contrato_version') IS DISTINCT FROM '1' THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
 END IF;
 f:=solicitud->'fuente_rpt';
 IF jsonb_typeof(f) IS DISTINCT FROM 'object' THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
 END IF;
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_object_keys(f) key;
 IF claves IS DISTINCT FROM ARRAY['huella_sha256','referencia','version'] THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
 END IF;
 FOREACH entero IN ARRAY ARRAY[solicitud->'version_expediente',f->'version'] LOOP
   IF jsonb_typeof(entero) IS DISTINCT FROM 'number'
      OR (entero #>> '{}') !~ '^[1-9][0-9]{0,15}$' THEN
     RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
   END IF;
   IF (entero #>> '{}')::numeric>9007199254740991 THEN
     RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
   END IF;
 END LOOP;
 FOREACH clave IN ARRAY ARRAY['solicitud_ref','expediente_ref','capacidad_ref','correlacion_ref','idempotencia_ref','plaza_ref','puesto_ref'] LOOP
   IF jsonb_typeof(solicitud->clave) IS DISTINCT FROM 'string'
      OR vec_personal.referencia_alta_ejercicio_valida_v1(solicitud->>clave) IS NOT TRUE THEN
     RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
   END IF;
 END LOOP;
 IF jsonb_typeof(f->'referencia') IS DISTINCT FROM 'string'
    OR jsonb_typeof(f->'huella_sha256') IS DISTINCT FROM 'string'
    OR vec_personal.referencia_alta_ejercicio_valida_v1(f->>'referencia') IS NOT TRUE
    OR (f->>'huella_sha256') !~ '^[a-f0-9]{64}$'
    OR (f->>'huella_sha256')=repeat('0',64) THEN
   RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='solicitud inválida';
 END IF;
 resultado:=
   vec_personal.campo_canonico_alta_v1('esquema',solicitud->>'esquema')||
   vec_personal.campo_canonico_alta_v1('contrato_version','1')||
   vec_personal.campo_canonico_alta_v1('solicitud_ref',solicitud->>'solicitud_ref')||
   vec_personal.campo_canonico_alta_v1('expediente_ref',solicitud->>'expediente_ref')||
   vec_personal.campo_canonico_alta_v1('version_expediente',solicitud->>'version_expediente')||
   vec_personal.campo_canonico_alta_v1('capacidad_ref',solicitud->>'capacidad_ref')||
   vec_personal.campo_canonico_alta_v1('correlacion_ref',solicitud->>'correlacion_ref')||
   vec_personal.campo_canonico_alta_v1('idempotencia_ref',solicitud->>'idempotencia_ref')||
   vec_personal.campo_canonico_alta_v1('rpt_ref',f->>'referencia')||
   vec_personal.campo_canonico_alta_v1('rpt_version',f->>'version')||
   vec_personal.campo_canonico_alta_v1('rpt_huella_sha256',f->>'huella_sha256')||
   vec_personal.campo_canonico_alta_v1('puesto_ref',solicitud->>'puesto_ref')||
   vec_personal.campo_canonico_alta_v1('plaza_ref',solicitud->>'plaza_ref');
 RETURN convert_to(resultado,'UTF8');
END $$;

CREATE FUNCTION vec_personal.material_alta_ejercicio_canonico_v1(entrada jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog, vec_personal AS $$
DECLARE m jsonb; p jsonb; s jsonb; f jsonb; x jsonb; claves text[]; salida text; js text; jr text;
BEGIN
 IF jsonb_typeof(entrada) IS DISTINCT FROM 'object' OR (SELECT array_agg(key ORDER BY key) FROM jsonb_object_keys(entrada) key) IS DISTINCT FROM ARRAY['Esquema','Material'] OR (entrada->>'Esquema') IS DISTINCT FROM 'vec.personal.alta-ejercicio.material.v1' THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material inválido'; END IF;
 m:=entrada->'Material'; IF jsonb_typeof(m)<>'object' OR (SELECT array_agg(key ORDER BY key) FROM jsonb_object_keys(m) key) IS DISTINCT FROM ARRAY['ActorRef','OrganizacionRef','PerfilRef','Preparacion'] THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material inválido'; END IF;
 p:=m->'Preparacion'; IF jsonb_typeof(p)<>'object' OR (SELECT array_agg(key ORDER BY key) FROM jsonb_object_keys(p) key) IS DISTINCT FROM ARRAY['Fuente','Solicitud','Vinculo'] THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material inválido'; END IF;
 s:=p->'Solicitud'; PERFORM vec_personal.solicitud_alta_ejercicio_canonica_v1(s);
 f:=p->'Fuente'; x:=p->'Vinculo';
 IF jsonb_typeof(f)<>'object' OR jsonb_typeof(x)<>'object' OR (SELECT array_agg(key ORDER BY key) FROM jsonb_object_keys(f) key) IS DISTINCT FROM ARRAY['HuellaSHA256','Referencia','Version'] OR (SELECT array_agg(key ORDER BY key) FROM jsonb_object_keys(x) key) IS DISTINCT FROM ARRAY['AntecedenteEjercicioRef','CentroRef','Desde','FuenteRPT','Hasta','PersonaSinteticaRef','PlazaRef','PuestoRef'] THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material inválido'; END IF;
 IF jsonb_typeof(f->'Referencia')<>'string' OR jsonb_typeof(f->'Version')<>'number' OR jsonb_typeof(f->'HuellaSHA256')<>'string' OR jsonb_typeof(x->'FuenteRPT')<>'object' OR EXISTS (SELECT 1 FROM unnest(ARRAY['OrganizacionRef','ActorRef','PerfilRef']) k WHERE jsonb_typeof(m->k)<>'string' OR NOT vec_personal.referencia_alta_ejercicio_valida_v1(m->>k)) OR EXISTS (SELECT 1 FROM unnest(ARRAY['PersonaSinteticaRef','CentroRef','PuestoRef','PlazaRef','Desde','Hasta','AntecedenteEjercicioRef']) k WHERE jsonb_typeof(x->k)<>'string') OR NOT vec_personal.referencia_alta_ejercicio_valida_v1(f->>'Referencia') OR f->>'Version' !~ '^[1-9][0-9]{0,15}$' OR (f->>'Version')::numeric > 9007199254740991 OR f->>'HuellaSHA256' !~ '^[a-f0-9]{64}$' OR f->>'HuellaSHA256'=repeat('0',64) OR NOT vec_personal.referencia_alta_ejercicio_valida_v1(x->>'PersonaSinteticaRef') OR x->>'PersonaSinteticaRef' !~ '^persona:ejercicio:' OR x->>'PersonaSinteticaRef'=m->>'ActorRef' OR NOT vec_personal.referencia_alta_ejercicio_valida_v1(x->>'CentroRef') OR x->>'PuestoRef'<>s->>'puesto_ref' OR x->>'PlazaRef'<>s->>'plaza_ref' OR x->'FuenteRPT'<>s->'fuente_rpt' OR NOT vec_personal.fecha_civil_go_valida_v1(x->>'Desde') OR NOT vec_personal.fecha_civil_go_valida_v1(x->>'Hasta') OR x->>'Hasta'<x->>'Desde' OR (x->>'AntecedenteEjercicioRef'<>'' AND NOT vec_personal.referencia_alta_ejercicio_valida_v1(x->>'AntecedenteEjercicioRef')) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material inválido'; END IF;
 jr := '{"referencia":'||vec_personal.texto_json_go_v1((s->'fuente_rpt')->>'referencia')||',"version":'||((s->'fuente_rpt')->>'version')||',"huella_sha256":'||vec_personal.texto_json_go_v1((s->'fuente_rpt')->>'huella_sha256')||'}';
 js := '{"esquema":'||vec_personal.texto_json_go_v1(s->>'esquema')||',"contrato_version":'||(s->>'contrato_version')||',"solicitud_ref":'||vec_personal.texto_json_go_v1(s->>'solicitud_ref')||',"expediente_ref":'||vec_personal.texto_json_go_v1(s->>'expediente_ref')||',"version_expediente":'||(s->>'version_expediente')||',"capacidad_ref":'||vec_personal.texto_json_go_v1(s->>'capacidad_ref')||',"correlacion_ref":'||vec_personal.texto_json_go_v1(s->>'correlacion_ref')||',"idempotencia_ref":'||vec_personal.texto_json_go_v1(s->>'idempotencia_ref')||',"fuente_rpt":'||jr||',"puesto_ref":'||vec_personal.texto_json_go_v1(s->>'puesto_ref')||',"plaza_ref":'||vec_personal.texto_json_go_v1(s->>'plaza_ref')||'}';
 salida := '{"Esquema":'||vec_personal.texto_json_go_v1(entrada->>'Esquema')||',"Material":{"Preparacion":{"Solicitud":'||js||',"Fuente":{"Referencia":'||vec_personal.texto_json_go_v1(f->>'Referencia')||',"Version":'||(f->>'Version')||',"HuellaSHA256":'||vec_personal.texto_json_go_v1(f->>'HuellaSHA256')||'},"Vinculo":{"PersonaSinteticaRef":'||vec_personal.texto_json_go_v1(x->>'PersonaSinteticaRef')||',"CentroRef":'||vec_personal.texto_json_go_v1(x->>'CentroRef')||',"PuestoRef":'||vec_personal.texto_json_go_v1(x->>'PuestoRef')||',"PlazaRef":'||vec_personal.texto_json_go_v1(x->>'PlazaRef')||',"FuenteRPT":'||jr||',"Desde":'||vec_personal.texto_json_go_v1(x->>'Desde')||',"Hasta":'||vec_personal.texto_json_go_v1(x->>'Hasta')||',"AntecedenteEjercicioRef":'||vec_personal.texto_json_go_v1(x->>'AntecedenteEjercicioRef')||'}},"OrganizacionRef":'||vec_personal.texto_json_go_v1(m->>'OrganizacionRef')||',"ActorRef":'||vec_personal.texto_json_go_v1(m->>'ActorRef')||',"PerfilRef":'||vec_personal.texto_json_go_v1(m->>'PerfilRef')||'}}';
 RETURN convert_to(salida,'UTF8');
END $$;

CREATE FUNCTION vec_personal.contexto_alta_ejercicio_canonico_v1(entrada jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path = pg_catalog, vec_personal AS $$
DECLARE m jsonb; s jsonb; f jsonb; x jsonb; h text; hs text;
BEGIN
 PERFORM vec_personal.material_alta_ejercicio_canonico_v1(entrada); m:=entrada->'Material'; s:=m->'Preparacion'->'Solicitud'; f:=m->'Preparacion'->'Fuente'; x:=m->'Preparacion'->'Vinculo';
 h:=encode(pg_catalog.sha256(vec_personal.material_alta_ejercicio_canonico_v1(entrada)),'hex'); hs:=encode(pg_catalog.sha256(vec_personal.solicitud_alta_ejercicio_canonica_v1(s)),'hex');
 RETURN convert_to('{"ambitos":{"centro_ref":'||vec_personal.texto_json_go_v1(x->>'CentroRef')||',"organizacion_ref":'||vec_personal.texto_json_go_v1(m->>'OrganizacionRef')||'},"atributos":{"actor_ref":'||vec_personal.texto_json_go_v1(m->>'ActorRef')||',"capacidad_ref":'||vec_personal.texto_json_go_v1(s->>'capacidad_ref')||',"desde":'||vec_personal.texto_json_go_v1(x->>'Desde')||',"expediente_ref":'||vec_personal.texto_json_go_v1(s->>'expediente_ref')||',"fuente_ref":'||vec_personal.texto_json_go_v1(f->>'Referencia')||',"fuente_sha256":'||vec_personal.texto_json_go_v1(f->>'HuellaSHA256')||',"fuente_version":'||vec_personal.texto_json_go_v1(f->>'Version')||',"hasta":'||vec_personal.texto_json_go_v1(x->>'Hasta')||',"material_sha256":'||vec_personal.texto_json_go_v1(h)||',"perfil_ref":'||vec_personal.texto_json_go_v1(m->>'PerfilRef')||',"persona_sintetica_ref":'||vec_personal.texto_json_go_v1(x->>'PersonaSinteticaRef')||',"plaza_ref":'||vec_personal.texto_json_go_v1(x->>'PlazaRef')||',"puesto_ref":'||vec_personal.texto_json_go_v1(x->>'PuestoRef')||',"rpt_ref":'||vec_personal.texto_json_go_v1((s->'fuente_rpt')->>'referencia')||',"rpt_sha256":'||vec_personal.texto_json_go_v1((s->'fuente_rpt')->>'huella_sha256')||',"rpt_version":'||vec_personal.texto_json_go_v1((s->'fuente_rpt')->>'version')||',"solicitud_sha256":'||vec_personal.texto_json_go_v1(hs)||',"tipo_validacion":"ejercicio_sintetico","version_expediente":'||vec_personal.texto_json_go_v1(s->>'version_expediente')||'}}','UTF8');
END $$;
REVOKE ALL ON FUNCTION vec_personal.texto_json_go_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.campo_canonico_alta_v1(text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.material_alta_ejercicio_canonico_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.solicitud_alta_ejercicio_canonica_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.contexto_alta_ejercicio_canonico_v1(jsonb) FROM PUBLIC;
COMMIT;
