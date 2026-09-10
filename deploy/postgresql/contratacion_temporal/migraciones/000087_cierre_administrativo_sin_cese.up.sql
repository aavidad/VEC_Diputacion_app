\set ON_ERROR_STOP on
-- CT87: una continuación administrativa sobre la raíz existente. No publica
-- definiciones ni libros, no confirma incorporación y no altera períodos/cese.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.cierre_administrativo_sin_cese.v1',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
DECLARE n text; f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8' THEN RAISE EXCEPTION 'CT87: propietario incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2','incorporacion_registro_v2','anotacion_administrativa_incorporacion_v1','actuacion_expediente_integral'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_contratacion_temporal.'||n) AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity) THEN RAISE EXCEPTION 'CT87: fuente incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF to_regclass('vec_contratacion_temporal.cierre_administrativo_registro_v1') IS NOT NULL THEN RAISE EXCEPTION 'CT87: objeto preexistente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_administrativo_sin_cese_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE') THEN RAISE EXCEPTION 'CT87: AD3-31 requerido' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_texto(s text) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE b bytea;
BEGIN
 IF s IS NULL OR octet_length(s)>1048576 THEN RAISE EXCEPTION 'CT87: texto inválido' USING ERRCODE='22023'; END IF;
 b:=convert_to(s,'UTF8');RETURN int4send(octet_length(b))||b;
END $$;
CREATE FUNCTION vec_contratacion_temporal.cierre87_cabecera(d text) RETURNS bytea
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
 SELECT vec_contratacion_temporal.cierre87_texto(d)||int2send(1::smallint)||vec_contratacion_temporal.cierre87_texto('sha-256')
$$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_continuacion_canonica(c jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE b bytea; k text;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_forma(c,ARRAY['seguimiento_ref','organizacion_ref','expediente_ref','relacion_ref','definicion_original','definicion_sucesora','huella_raiz_sha256','version_anterior','huella_estado_anterior_sha256','huella_actuacion_anterior_sha256','primera_secuencia','actuacion_ref','recibo_ref','correlacion_ref','registrada_en','huella_peticion_sha256']);
 b:=vec_contratacion_temporal.cierre87_cabecera('vec.dipgra.contratacion-temporal.seguimiento.continuacion');
 FOREACH k IN ARRAY ARRAY['seguimiento_ref','organizacion_ref','expediente_ref','relacion_ref'] LOOP b:=b||vec_contratacion_temporal.cierre87_texto(c->>k); END LOOP;
 b:=b||vec_contratacion_temporal.seguimiento73_nodo(c->'definicion_original','defref')||vec_contratacion_temporal.seguimiento73_nodo(c->'definicion_sucesora','defref')
  ||vec_contratacion_temporal.seguimiento73_escalar(c->'huella_raiz_sha256','hash')||vec_contratacion_temporal.seguimiento73_escalar(c->'version_anterior','u1')
  ||vec_contratacion_temporal.seguimiento73_escalar(c->'huella_estado_anterior_sha256','hash')||vec_contratacion_temporal.seguimiento73_escalar(c->'huella_actuacion_anterior_sha256','hash')
  ||vec_contratacion_temporal.seguimiento73_escalar(c->'primera_secuencia','u1');
 FOREACH k IN ARRAY ARRAY['actuacion_ref','recibo_ref','correlacion_ref'] LOOP b:=b||vec_contratacion_temporal.seguimiento73_escalar(c->k,'ref'); END LOOP;
 RETURN b||vec_contratacion_temporal.seguimiento73_escalar(c->'registrada_en','micro')||vec_contratacion_temporal.seguimiento73_escalar(c->'huella_peticion_sha256','hash');
END $$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_validar_sucesora(o jsonb,p jsonb) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE a jsonb; b jsonb; t jsonb;
BEGIN
 a:=vec_contratacion_temporal.seguimiento73_definicion(o);b:=vec_contratacion_temporal.seguimiento73_definicion(p);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(b->'referencia'=a->'referencia' AND (b->>'version')::numeric=(a->>'version')::numeric+1
  AND vec_contratacion_temporal.seguimiento73_micro(b->'publicado_en')>vec_contratacion_temporal.seguimiento73_micro(a->'publicado_en')
  AND vec_contratacion_temporal.seguimiento73_micro(b#>'{vigencia,desde}')>=vec_contratacion_temporal.seguimiento73_micro(b->'publicado_en')
  AND b->'estado_inicial'=a->'estado_inicial' AND b->'prohibe_ciclos_silenciosos'=a->'prohibe_ciclos_silenciosos'
  AND jsonb_array_length(b->'estados')=jsonb_array_length(a->'estados')+1 AND b->'estados' @> a->'estados'
  AND NOT a->'estados' @> '[{"clave":"cerrado_administrativamente"}]'::jsonb
  AND b->'estados' @> '[{"clave":"vigente","final":false},{"clave":"cerrado_administrativamente","final":true}]'::jsonb
  AND jsonb_array_length(b->'transiciones')=jsonb_array_length(a->'transiciones')+1
  AND b->'transiciones' @> a->'transiciones' AND b->'motivos' @> a->'motivos');
 SELECT value INTO STRICT t FROM jsonb_array_elements(b->'transiciones') WHERE value->>'clave'='cerrar_administrativamente_sin_cese';
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(t->>'origen'='vigente' AND t->>'destino'='cerrado_administrativamente' AND t->>'clase'='ordinaria'
  AND t->>'efecto_periodo'='ninguno' AND t->'motivo_obligatorio'='true'::jsonb AND t->'requiere_periodo'='false'::jsonb
  AND NOT t ? 'calendario' AND t->'exige_actor_distinto'='false'::jsonb
  AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(a->'transiciones') z WHERE z->>'clave'='cerrar_administrativamente_sin_cese'));
 RETURN t;
END $$;

-- Codec explícito nuevo. El escritor CT73 se reutiliza sin cambiar su función:
-- su gramática valida los nodos y este contrato valida la adopción y el replay.
CREATE FUNCTION vec_contratacion_temporal.estado_seguimiento_continuado_canonico_v1(o jsonb,p jsonb,e jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE c jsonb; ant jsonb; acts jsonb; ultima jsonb; previa jsonb; datos jsonb; tr jsonb; esperada jsonb; esperado jsonb; def jsonb;
 canon_ant bytea; canon_estado bytea; canon_c bytea; n int; k text; momento bigint; inicio bigint; fin bigint; hp text; ha text; c_esperada jsonb; req jsonb; doc jsonb; docs jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(pg_column_size(e)<=8388608 AND e ? 'continuacion');
 c:=e->'continuacion';PERFORM vec_contratacion_temporal.cierre87_continuacion_canonica(c);
 acts:=vec_contratacion_temporal.seguimiento73_array(e->'actuaciones',10000,2,false);n:=jsonb_array_length(acts);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir((e->>'version')::numeric=n AND (c->>'version_anterior')::numeric=n-1 AND (c->>'primera_secuencia')::numeric=n);
 ultima:=acts->(n-1);previa:=acts->(n-2);
 ant:=(e-'continuacion')||jsonb_build_object('version',n-1,'estado_actual',previa->'estado_destino','actualizado_en',previa->'registrada_en','actuaciones',acts-(n-1));
 canon_ant:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(o,ant);
 tr:=vec_contratacion_temporal.cierre87_validar_sucesora(o,p);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(ant->>'estado_actual'='vigente' AND jsonb_array_length(ant->'periodos_resultantes')>0 AND NOT ant ? 'cese_efectivo');
 datos:=ultima-ARRAY['secuencia','version_seguimiento','definicion','clase','estado_origen','estado_destino','huella_peticion_sha256','huella_anterior_sha256','huella_actuacion_sha256'];
 docs:=vec_contratacion_temporal.seguimiento73_array(datos->'documentos',32,0,true);
 SELECT coalesce(jsonb_agg(value ORDER BY (value->>'tipo_clave') COLLATE "C",(value->>'referencia') COLLATE "C"),'[]'::jsonb) INTO docs FROM jsonb_array_elements(docs);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_nodo(datos,'peticion')=vec_contratacion_temporal.seguimiento73_nodo(jsonb_set(datos,'{documentos}',docs),'peticion'));
 hp:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(datos,'peticion')),'hex');
 momento:=vec_contratacion_temporal.seguimiento73_micro(datos->'registrada_en');inicio:=vec_contratacion_temporal.seguimiento73_micro(p#>'{vigencia,desde}');fin:=vec_contratacion_temporal.seguimiento73_micro(p#>'{vigencia,hasta}',true);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(datos->>'transicion_clave'='cerrar_administrativamente_sin_cese'
  AND vec_contratacion_temporal.seguimiento73_micro(datos->'efectivo_en')=momento AND momento>=vec_contratacion_temporal.seguimiento73_micro(ant->'actualizado_en')
  AND momento>=vec_contratacion_temporal.seguimiento73_micro(p->'publicado_en') AND momento>=inicio AND (fin=-62135596800000000 OR momento<fin)
  AND NOT datos ?| ARRAY['periodo','calendario','rectifica_actuacion_ref'] AND tr->'motivos_permitidos' @> jsonb_build_array(datos->'motivo_clave')
  AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ant->'actuaciones') z WHERE z->'actuacion_ref'=datos->'actuacion_ref' OR z->'recibo_ref'=datos->'recibo_ref'));
 FOR req IN SELECT value FROM jsonb_array_elements(coalesce(tr->'documentos','[]'::jsonb)) LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(req->'obligatorio'='false'::jsonb OR EXISTS(SELECT 1 FROM jsonb_array_elements(datos->'documentos') z WHERE z->'tipo_clave'=req->'tipo_clave'));
 END LOOP;
 FOR doc IN SELECT value FROM jsonb_array_elements(coalesce(nullif(datos->'documentos','null'::jsonb),'[]'::jsonb)) LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(EXISTS(SELECT 1 FROM jsonb_array_elements(tr->'documentos') z WHERE z->'tipo_clave'=doc->'tipo_clave'));
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT z->>'referencia') FROM jsonb_array_elements(coalesce(nullif(datos->'documentos','null'::jsonb),'[]'::jsonb)) z));
 def:=jsonb_build_object('referencia',p->'referencia','version',p->'version','huella_sha256',p->'huella_sha256');
 esperada:=datos||jsonb_build_object('secuencia',n,'version_seguimiento',n,'definicion',def,'clase','ordinaria','estado_origen','vigente','estado_destino','cerrado_administrativamente','huella_peticion_sha256',hp,'huella_anterior_sha256',previa->'huella_actuacion_sha256','huella_actuacion_sha256',repeat('1',64));
 ha:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(esperada,'actuacion')),'hex');esperada:=jsonb_set(esperada,'{huella_actuacion_sha256}',to_jsonb(ha));
 esperado:=ant||jsonb_build_object('version',n,'estado_actual','cerrado_administrativamente','actualizado_en',datos->'registrada_en','actuaciones',(ant->'actuaciones')||jsonb_build_array(esperada));
 c_esperada:=jsonb_build_object('seguimiento_ref',ant->'referencia','organizacion_ref',ant->'organizacion_ref','expediente_ref',ant->'expediente_ref','relacion_ref',ant->'relacion_ref',
  'definicion_original',ant->'definicion','definicion_sucesora',def,'huella_raiz_sha256',ant->'huella_raiz_sha256','version_anterior',n-1,'huella_estado_anterior_sha256',encode(sha256(canon_ant),'hex'),
  'huella_actuacion_anterior_sha256',previa->'huella_actuacion_sha256','primera_secuencia',n,'actuacion_ref',datos->'actuacion_ref','recibo_ref',datos->'recibo_ref','correlacion_ref',datos->'correlacion_ref','registrada_en',datos->'registrada_en','huella_peticion_sha256',hp);
 canon_c:=vec_contratacion_temporal.cierre87_continuacion_canonica(c);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(canon_c=vec_contratacion_temporal.cierre87_continuacion_canonica(c_esperada));
 canon_estado:=vec_contratacion_temporal.seguimiento73_nodo(e-'continuacion','estado');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(canon_estado=vec_contratacion_temporal.seguimiento73_nodo(esperado,'estado'));
 -- Sustituir únicamente el encabezado de ESTE nuevo material; ninguna raíz ni
 -- fila/canon anterior se modifica ni se admite este estado en el lector V1.
 RETURN vec_contratacion_temporal.cierre87_texto('vec.dipgra.contratacion-temporal.seguimiento.estado-continuado')
  ||substring(canon_estado FROM 5+octet_length('vec.dipgra.contratacion-temporal.seguimiento.estado'))||int4send(octet_length(canon_c))||canon_c;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CT87: continuidad inválida' USING ERRCODE='22023';
END $$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_libro_canonico(p jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE b bytea; t jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_forma(p,ARRAY['Referencia','Version','Ambito','Tareas']);
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(p->>'Referencia' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  AND (p->>'Version')::numeric BETWEEN 1 AND 9007199254740991 AND p->>'Ambito'='cierre_administrativo_ejercicio_incorporacion_y_primera_anotacion'
  AND p->'Tareas'='[{"Clave":"incorporacion_original_acreditada","Evidencia":"incorporacion_original"},{"Clave":"primera_anotacion_administrativa_acreditada","Evidencia":"primera_anotacion_administrativa"}]'::jsonb);
 b:=vec_contratacion_temporal.cierre87_cabecera('vec.dipgra.contratacion-temporal.cierre-administrativo.libro')||vec_contratacion_temporal.cierre87_texto(p->>'Referencia')
  ||vec_contratacion_temporal.seguimiento73_escalar(p->'Version','u1')||vec_contratacion_temporal.cierre87_texto(p->>'Ambito')||int4send(2);
 FOR t IN SELECT value FROM jsonb_array_elements(p->'Tareas') LOOP b:=b||vec_contratacion_temporal.cierre87_texto(t->>'Clave')||vec_contratacion_temporal.cierre87_texto(t->>'Evidencia'); END LOOP;
 RETURN b;
END $$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_inventario_canonico(s jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE b bytea; libro bytea; e jsonb; v jsonb; k text; n int:=0; raiz text; refs text[]:='{}'; huellas text[]:='{}'; tarea jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_forma(s,ARRAY['Libro','OrganizacionRef','ExpedienteRef','SeguimientoRef','Estados']);
 libro:=vec_contratacion_temporal.cierre87_libro_canonico(s->'Libro');
 b:=vec_contratacion_temporal.cierre87_cabecera('vec.dipgra.contratacion-temporal.cierre-administrativo.inventario')||int4send(octet_length(libro))||libro;
 FOREACH k IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef','SeguimientoRef'] LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(s->k)='string' AND s->>k ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$');
  b:=b||vec_contratacion_temporal.cierre87_texto(s->>k);
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(s->'Estados')='array' AND jsonb_array_length(s->'Estados')=2);b:=b||int4send(2);
 FOR e IN SELECT value FROM jsonb_array_elements(s->'Estados') LOOP
  PERFORM vec_contratacion_temporal.seguimiento73_forma(e,ARRAY['Clave','Pendiente','Evidencia']);v:=e->'Evidencia';tarea:=s#>ARRAY['Libro','Tareas',n::text];
  PERFORM vec_contratacion_temporal.seguimiento73_forma(v,ARRAY['Tipo','Referencia','OrganizacionRef','ExpedienteRef','SeguimientoRef','VersionSeguimientoOriginal','HuellaRaizSeguimientoSHA256','VersionEvidencia','HuellaEvidenciaSHA256']);
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(e->'Clave'=tarea->'Clave' AND v->'Tipo'=tarea->'Evidencia' AND v->'VersionSeguimientoOriginal'='1'::jsonb
   AND (v->>'VersionEvidencia')::numeric BETWEEN 1 AND 9007199254740991 AND v->>'Referencia' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
   AND NOT (v->>'Referencia')=ANY(refs) AND NOT (v->>'HuellaEvidenciaSHA256')=ANY(huellas));
  refs:=array_append(refs,v->>'Referencia');huellas:=array_append(huellas,v->>'HuellaEvidenciaSHA256');
  IF n=0 THEN raiz:=v->>'HuellaRaizSeguimientoSHA256';ELSE PERFORM vec_contratacion_temporal.seguimiento73_exigir(raiz=v->>'HuellaRaizSeguimientoSHA256');END IF;
  b:=b||vec_contratacion_temporal.cierre87_texto(e->>'Clave')||vec_contratacion_temporal.seguimiento73_escalar(e->'Pendiente','bool')||vec_contratacion_temporal.cierre87_texto(v->>'Tipo')||vec_contratacion_temporal.cierre87_texto(v->>'Referencia');
  FOREACH k IN ARRAY ARRAY['OrganizacionRef','ExpedienteRef','SeguimientoRef'] LOOP PERFORM vec_contratacion_temporal.seguimiento73_exigir(v->k=s->k);b:=b||vec_contratacion_temporal.cierre87_texto(v->>k);END LOOP;
  b:=b||vec_contratacion_temporal.seguimiento73_escalar(v->'VersionSeguimientoOriginal','u1')||vec_contratacion_temporal.seguimiento73_escalar(v->'HuellaRaizSeguimientoSHA256','hash')
   ||vec_contratacion_temporal.seguimiento73_escalar(v->'VersionEvidencia','u1')||vec_contratacion_temporal.seguimiento73_escalar(v->'HuellaEvidenciaSHA256','hash');n:=n+1;
 END LOOP;
 RETURN b;
END $$;

-- Publicación y configuración explícitas por el propietario. Esta migración no
-- añade ninguna fila ni permite a la aplicación publicar/sustituir políticas.
CREATE TABLE vec_contratacion_temporal.cierre_administrativo_libro_v1(
 libro_ref text NOT NULL,version numeric(20,0) NOT NULL,publicacion_json jsonb NOT NULL,canon bytea NOT NULL,huella_sha256 text NOT NULL,publicada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(libro_ref,version),UNIQUE(libro_ref,version,huella_sha256),
 CHECK(canon=vec_contratacion_temporal.cierre87_libro_canonico(publicacion_json) AND huella_sha256=encode(sha256(canon),'hex')),
 CHECK((publicacion_json->>'Referencia'=libro_ref AND (publicacion_json->>'Version')::numeric=version) IS TRUE),CHECK(isfinite(publicada_en))
);
CREATE TABLE vec_contratacion_temporal.cierre_administrativo_configuracion_v1(
 organizacion_ref text NOT NULL,definicion_ref text NOT NULL,definicion_version numeric(20,0) NOT NULL,definicion_sha256 text NOT NULL,
 sucesora_version numeric(20,0) NOT NULL,sucesora_sha256 text NOT NULL,libro_ref text NOT NULL,libro_version numeric(20,0) NOT NULL,libro_sha256 text NOT NULL,publicada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(organizacion_ref,definicion_ref,definicion_version),
 FOREIGN KEY(definicion_ref,definicion_version,definicion_sha256) REFERENCES vec_contratacion_temporal.seguimiento_definicion_v2(definicion_ref,definicion_version,definicion_sha256),
 FOREIGN KEY(definicion_ref,sucesora_version,sucesora_sha256) REFERENCES vec_contratacion_temporal.seguimiento_definicion_v2(definicion_ref,definicion_version,definicion_sha256),
 FOREIGN KEY(libro_ref,libro_version,libro_sha256) REFERENCES vec_contratacion_temporal.cierre_administrativo_libro_v1(libro_ref,version,huella_sha256),
 CHECK(sucesora_version=definicion_version+1 AND isfinite(publicada_en))
);

CREATE TABLE vec_contratacion_temporal.cierre_administrativo_preparacion_v1(
 recibo_ref text PRIMARY KEY,transaccion xid8 NOT NULL,sesion_sql name NOT NULL,solicitud jsonb NOT NULL,coordenadas jsonb NOT NULL,
 preparacion jsonb NOT NULL,valida_hasta timestamptz(6) NOT NULL,decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE,auditoria_ref text NOT NULL UNIQUE,
 creada_en timestamptz(6) NOT NULL,CHECK(isfinite(creada_en) AND valida_hasta>creada_en)
);
CREATE TABLE vec_contratacion_temporal.cierre_administrativo_registro_v1(
 recibo_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.cierre_administrativo_preparacion_v1,
 organizacion_ref text NOT NULL,clave_idempotencia text NOT NULL,seguimiento_ref text NOT NULL,expediente_ref text NOT NULL,
 version_anterior numeric(20,0) NOT NULL,version_resultante numeric(20,0) NOT NULL,estado_anterior_sha256 text NOT NULL,estado_resultante_sha256 text NOT NULL,
 libro_ref text NOT NULL,libro_version numeric(20,0) NOT NULL,libro_sha256 text NOT NULL,
 recibo_incorporacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.incorporacion_registro_v2,
 recibo_anotacion_ref text NOT NULL REFERENCES vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1,
 solicitud jsonb NOT NULL,inventario jsonb NOT NULL,inventario_canon bytea NOT NULL,inventario_sha256 text NOT NULL,recibo jsonb NOT NULL,registrada_en timestamptz(6) NOT NULL,
 UNIQUE(organizacion_ref,clave_idempotencia),UNIQUE(seguimiento_ref,version_resultante),
 FOREIGN KEY(seguimiento_ref,version_anterior,estado_anterior_sha256) REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento,estado_sha256),
 FOREIGN KEY(seguimiento_ref,version_resultante,estado_resultante_sha256) REFERENCES vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,version_seguimiento,estado_sha256),
 FOREIGN KEY(libro_ref,libro_version,libro_sha256) REFERENCES vec_contratacion_temporal.cierre_administrativo_libro_v1(libro_ref,version,huella_sha256),
 CHECK(version_anterior=1 AND version_resultante=2),CHECK(isfinite(registrada_en)),
 CHECK(inventario_canon=vec_contratacion_temporal.cierre87_inventario_canonico(inventario) AND inventario_sha256=encode(sha256(inventario_canon),'hex'))
);
-- Una preparación nunca puede confirmarse por sí sola: la FK diferida exige
-- que callback/confirmación completen el cierre en la misma transacción.
ALTER TABLE vec_contratacion_temporal.cierre_administrativo_preparacion_v1 ADD CONSTRAINT cierre87_preparacion_confirmada_fk
 FOREIGN KEY(recibo_ref) REFERENCES vec_contratacion_temporal.cierre_administrativo_registro_v1 DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE vec_contratacion_temporal.cierre_administrativo_auditoria_v1(
 auditoria_ref text PRIMARY KEY,recibo_ref text NOT NULL REFERENCES vec_contratacion_temporal.cierre_administrativo_registro_v1 DEFERRABLE INITIALLY DEFERRED,
 decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE,recuperado boolean NOT NULL,registrada_en timestamptz(6) NOT NULL,CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_contratacion_temporal.cierre_administrativo_outbox_v1(
 evento_ref text PRIMARY KEY,recibo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.cierre_administrativo_registro_v1,
 estado_sha256 text NOT NULL,payload_canonico bytea NOT NULL,payload_sha256 text NOT NULL,registrada_en timestamptz(6) NOT NULL,
 CHECK(payload_sha256=encode(sha256(payload_canonico),'hex') AND isfinite(registrada_en))
);

CREATE FUNCTION vec_contratacion_temporal.cierre87_inventario_fuentes(libro jsonb,org text,exp text,seg text) RETURNS jsonb
LANGUAGE plpgsql STABLE SET search_path=pg_catalog SET row_security=on AS $$
DECLARE i vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 an vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1%ROWTYPE;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 a vec_contratacion_temporal.actuacion_expediente_integral%ROWTYPE;
 original jsonb; ev1 jsonb; ev2 jsonb; salida jsonb;
BEGIN
 SELECT * INTO STRICT i FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE seguimiento_ref=seg AND organizacion_ref=org AND expediente_ref=exp AND version_resultante=1;
 SELECT * INTO STRICT an FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 WHERE expediente_ref=exp AND organizacion_ref=org AND recibo_incorporacion_ref=i.recibo_ref;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=seg;
 SELECT * INTO STRICT a FROM vec_contratacion_temporal.actuacion_expediente_integral WHERE expediente_ref=exp AND version_expediente=an.version_resultante AND recibo_ref=an.recibo_ref;
 -- Reutiliza el cotejo CT86 del recibo original CT75 y su estado canónico.
 original:=vec_contratacion_temporal.anotacion86_origen(an.material_json);
 IF original->'seguimiento_original' IS DISTINCT FROM an.seguimiento_original
  OR original->>'recibo_incorporacion_ref' IS DISTINCT FROM i.recibo_ref
  OR original->>'estado_seguimiento_sha256' IS DISTINCT FROM an.estado_seguimiento_sha256
  OR an.seguimiento_original IS DISTINCT FROM jsonb_build_object('seguimiento_ref',seg,'version_seguimiento',1,'huella_raiz_seguimiento_sha256',r.raiz_sha256)
  OR a.prueba_huella_sha256 IS DISTINCT FROM encode(sha256(a.prueba_canonica),'hex')
  OR a.actuacion_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(a.actuacion_json::text,'UTF8')),'hex')
  OR a.actuacion_json->>'recibo_ref' IS DISTINCT FROM an.recibo_ref
  OR a.actuacion_json->>'accion_clave' IS DISTINCT FROM 'contratacion_temporal.anotacion_administrativa.registrar' THEN
  RAISE EXCEPTION 'CT87: evidencia divergente' USING ERRCODE='P0872';END IF;
 ev1:=jsonb_build_object('Tipo','incorporacion_original','Referencia',i.recibo_ref,'OrganizacionRef',org,'ExpedienteRef',exp,'SeguimientoRef',seg,'VersionSeguimientoOriginal',1,
  'HuellaRaizSeguimientoSHA256',r.raiz_sha256,'VersionEvidencia',i.version_resultante,'HuellaEvidenciaSHA256',i.estado_resultante_sha256);
 ev2:=jsonb_build_object('Tipo','primera_anotacion_administrativa','Referencia',an.recibo_ref,'OrganizacionRef',org,'ExpedienteRef',exp,'SeguimientoRef',seg,'VersionSeguimientoOriginal',1,
  'HuellaRaizSeguimientoSHA256',r.raiz_sha256,'VersionEvidencia',an.version_resultante,'HuellaEvidenciaSHA256',a.prueba_huella_sha256);
 salida:=jsonb_build_object('Libro',libro,'OrganizacionRef',org,'ExpedienteRef',exp,'SeguimientoRef',seg,'Estados',jsonb_build_array(
  jsonb_build_object('Clave','incorporacion_original_acreditada','Pendiente',false,'Evidencia',ev1),
  jsonb_build_object('Clave','primera_anotacion_administrativa_acreditada','Pendiente',false,'Evidencia',ev2)));
 PERFORM vec_contratacion_temporal.cierre87_inventario_canonico(salida);RETURN salida;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CT87: inventario incompleto' USING ERRCODE='P0872';
END $$;

CREATE FUNCTION vec_contratacion_temporal.cierre87_validar_solicitud(s jsonb,x jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE k text; ambitos jsonb; atributos jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_forma(s,ARRAY['operacion','organizacion_ref','expediente_ref','seguimiento_ref','version_esperada','clave_idempotencia','transicion_clave','motivo_clave']);
 PERFORM vec_contratacion_temporal.seguimiento73_forma(x,ARRAY['actor_ref','perfil_ref','unidad_ref','correlacion_ref','principal_v3_ref','correlacion_v3_ref']);
 IF s->>'operacion' IS DISTINCT FROM 'cerrar_administrativamente_sin_cese' OR s->>'transicion_clave' IS DISTINCT FROM 'cerrar_administrativamente_sin_cese'
  OR s->'version_esperada' IS DISTINCT FROM '1'::jsonb OR jsonb_typeof(s->'clave_idempotencia') IS DISTINCT FROM 'string'
  OR s->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN RAISE EXCEPTION 'CT87: solicitud inválida' USING ERRCODE='P0870';END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','seguimiento_ref'] LOOP
  IF jsonb_typeof(s->k) IS DISTINCT FROM 'string' OR s->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'CT87: ámbito inválido' USING ERRCODE='P0870';END IF;
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_escalar(s->'motivo_clave','clave');
 FOREACH k IN ARRAY ARRAY['actor_ref','perfil_ref','unidad_ref','correlacion_ref','principal_v3_ref','correlacion_v3_ref'] LOOP
  IF jsonb_typeof(x->k) IS DISTINCT FROM 'string' OR x->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'CT87: identidad inválida' USING ERRCODE='P0870';END IF;
 END LOOP;
 ambitos:=jsonb_build_object('organizacion_ref',s->>'organizacion_ref','expediente_ref',s->>'expediente_ref','seguimiento_ref',s->>'seguimiento_ref');
 atributos:=jsonb_build_object('operacion',s->>'operacion','version_esperada','1','transicion_clave',s->>'transicion_clave','motivo_clave',s->>'motivo_clave',
  'principal_v3_ref',x->>'principal_v3_ref','actor_seguimiento_ref',x->>'actor_ref','correlacion_v3_ref',x->>'correlacion_v3_ref','correlacion_seguimiento_ref',x->>'correlacion_ref');
 RETURN convert_to('{"ambitos":'||vec_contratacion_temporal.anotacion86_mapa(ambitos)||',"atributos":'||vec_contratacion_temporal.anotacion86_mapa(atributos)||'}','UTF8');
END $$;

CREATE FUNCTION vec_contratacion_temporal.preparar_cierre_administrativo_sin_cese_v1(
 s jsonb,x jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $$
DECLARE c jsonb; d jsonb; h text; ahora timestamptz; limite timestamptz; consumo record;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE; anterior vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE; posterior vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 cfg vec_contratacion_temporal.cierre_administrativo_configuracion_v1%ROWTYPE; libro vec_contratacion_temporal.cierre_administrativo_libro_v1%ROWTYPE;
 viejo vec_contratacion_temporal.cierre_administrativo_registro_v1%ROWTYPE; prep vec_contratacion_temporal.cierre_administrativo_preparacion_v1%ROWTYPE;
 o jsonb; p jsonb; tr jsonb; inv jsonb; invcanon bytea; salida jsonb; recibo text; actuacion text; maxversion numeric; recuperado boolean;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CT87: transacción denegada' USING ERRCODE='P0873';END IF;
 h:=encode(sha256(vec_contratacion_temporal.cierre87_validar_solicitud(s,x)),'hex');
 PERFORM vec_contratacion_temporal.incorporacion75_piezas(ARRAY[p_capacidad,p_decision,p_motivo,p_contexto,p_payload,p_sobre,p_evidencia,p_raiz],p_persona_version,p_perfil_version);
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 ahora:=date_trunc('microseconds',clock_timestamp());limite:=vec_contratacion_temporal.incorporacion75_ventana(c,d,ahora);
 IF d->'concedida' IS DISTINCT FROM 'true'::jsonb OR d->>'codigo' IS DISTINCT FROM 'concedida'
  OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.seguimiento.cerrar' OR d->>'finalidad' IS DISTINCT FROM 'cerrar_expediente_contratacion_temporal'
  OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'seguimiento_contratacion_temporal'
  OR d->>'recurso_ref' IS DISTINCT FROM s->>'seguimiento_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
  OR d->>'principal_id' IS DISTINCT FROM x->>'principal_v3_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_ref'
  OR d->>'correlacion_ref' IS DISTINCT FROM x->>'correlacion_v3_ref'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'
  OR c->>'operacion' IS DISTINCT FROM d->>'accion' OR c->>'efecto_ref' IS DISTINCT FROM s->>'seguimiento_ref'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM h THEN RAISE EXCEPTION 'CT87: autoridad no ligada' USING ERRCODE='P0873';END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_administrativo_sin_cese_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS DISTINCT FROM true OR consumo.efecto_ref IS DISTINCT FROM s->>'seguimiento_ref' OR consumo.huella_efecto_sha256 IS DISTINCT FROM h THEN RAISE EXCEPTION 'CT87: consumo no ligado' USING ERRCODE='P0873';END IF;
 -- El consumo actual precede a toda lectura de negocio, también en replay.
 PERFORM pg_advisory_xact_lock(hashtextextended('CT87:'||(s->>'organizacion_ref')||':'||(s->>'clave_idempotencia'),0));
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=s->>'seguimiento_ref' AND organizacion_ref=s->>'organizacion_ref' AND expediente_ref=s->>'expediente_ref' FOR UPDATE;
 SELECT * INTO viejo FROM vec_contratacion_temporal.cierre_administrativo_registro_v1 WHERE organizacion_ref=s->>'organizacion_ref' AND clave_idempotencia=s->>'clave_idempotencia' FOR SHARE;recuperado:=FOUND;
 IF recuperado AND viejo.solicitud IS DISTINCT FROM s THEN RAISE EXCEPTION 'CT87: clave reutilizada' USING ERRCODE='P0871';END IF;
 SELECT * INTO STRICT anterior FROM vec_contratacion_temporal.seguimiento_estado_v2 WHERE seguimiento_ref=r.seguimiento_ref AND version_seguimiento=1 FOR SHARE;
 SELECT publicacion_json INTO STRICT o FROM vec_contratacion_temporal.seguimiento_definicion_v2 WHERE definicion_ref=r.definicion_ref AND definicion_version=r.definicion_version AND definicion_sha256=r.definicion_sha256 FOR SHARE;
 IF anterior.estado_canonico IS DISTINCT FROM vec_contratacion_temporal.estado_seguimiento_canonico_v1(o,anterior.estado_json) OR anterior.estado_sha256 IS DISTINCT FROM encode(sha256(anterior.estado_canonico),'hex')
  OR r.raiz_sha256 IS DISTINCT FROM encode(sha256(r.raiz_canonica),'hex') THEN RAISE EXCEPTION 'CT87: historia divergente' USING ERRCODE='P0872';END IF;
 SELECT * INTO STRICT cfg FROM vec_contratacion_temporal.cierre_administrativo_configuracion_v1 WHERE organizacion_ref=r.organizacion_ref AND definicion_ref=r.definicion_ref AND definicion_version=r.definicion_version AND definicion_sha256=r.definicion_sha256 FOR SHARE;
 SELECT * INTO STRICT libro FROM vec_contratacion_temporal.cierre_administrativo_libro_v1 WHERE libro_ref=cfg.libro_ref AND version=cfg.libro_version AND huella_sha256=cfg.libro_sha256 FOR SHARE;
 SELECT publicacion_json INTO STRICT p FROM vec_contratacion_temporal.seguimiento_definicion_v2 WHERE definicion_ref=cfg.definicion_ref AND definicion_version=cfg.sucesora_version AND definicion_sha256=cfg.sucesora_sha256 FOR SHARE;
 tr:=vec_contratacion_temporal.cierre87_validar_sucesora(o,p);
 inv:=vec_contratacion_temporal.cierre87_inventario_fuentes(libro.publicacion_json,r.organizacion_ref,r.expediente_ref,r.seguimiento_ref);invcanon:=vec_contratacion_temporal.cierre87_inventario_canonico(inv);
 ahora:=date_trunc('microseconds',clock_timestamp());PERFORM vec_contratacion_temporal.incorporacion75_ventana(c,d,ahora);
 IF ahora<cfg.publicada_en OR ahora<libro.publicada_en OR libro.canon IS DISTINCT FROM vec_contratacion_temporal.cierre87_libro_canonico(libro.publicacion_json)
  OR x->>'unidad_ref' IS DISTINCT FROM (SELECT v.agregado_json#>>'{asignacion,unidad_ref}' FROM vec_contratacion_temporal.expediente_integral_actual a JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version) WHERE a.expediente_ref=r.expediente_ref) THEN RAISE EXCEPTION 'CT87: configuración no vigente' USING ERRCODE='P0873';END IF;
 IF recuperado THEN
  SELECT * INTO STRICT prep FROM vec_contratacion_temporal.cierre_administrativo_preparacion_v1 WHERE recibo_ref=viejo.recibo_ref;
  SELECT * INTO STRICT posterior FROM vec_contratacion_temporal.seguimiento_estado_v2 WHERE seguimiento_ref=r.seguimiento_ref AND version_seguimiento=2 AND estado_sha256=viejo.estado_resultante_sha256;
  IF viejo.inventario_canon IS DISTINCT FROM invcanon OR viejo.inventario IS DISTINCT FROM inv
   OR posterior.estado_canonico IS DISTINCT FROM vec_contratacion_temporal.estado_seguimiento_continuado_canonico_v1(o,p,posterior.estado_json)
   OR posterior.estado_sha256 IS DISTINCT FROM encode(sha256(posterior.estado_canonico),'hex') THEN RAISE EXCEPTION 'CT87: recibo divergente' USING ERRCODE='P0872';END IF;
  INSERT INTO vec_contratacion_temporal.cierre_administrativo_auditoria_v1 VALUES(consumo.auditoria_ref,viejo.recibo_ref,consumo.decision_ref,consumo.consumo_huella_sha256,true,consumo.consumida_en);
  RETURN prep.preparacion||jsonb_build_object('recuperado',true,'verificada_en',vec_contratacion_temporal.incorporacion75_instante(ahora),'recibo',viejo.recibo,
   'estado_posterior',posterior.estado_json,'posterior_canon_hex',encode(posterior.estado_canonico,'hex'),'posterior_sha256',posterior.estado_sha256);
 END IF;
 SELECT max(version_seguimiento) INTO maxversion FROM vec_contratacion_temporal.seguimiento_estado_v2 WHERE seguimiento_ref=r.seguimiento_ref;
 IF maxversion IS DISTINCT FROM 1::numeric THEN RAISE EXCEPTION 'CT87: versión en conflicto' USING ERRCODE='P0872';END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(tr->'documentos') z WHERE z->'obligatorio'='true'::jsonb) THEN RAISE EXCEPTION 'CT87: documentos pendientes' USING ERRCODE='P0872';END IF;
 recibo:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');actuacion:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
 salida:=jsonb_build_object('recuperado',false,'solicitud',s,'verificada_en',vec_contratacion_temporal.incorporacion75_instante(ahora),'publicacion_original',o,'publicacion_sucesora',p,
  'estado_anterior',anterior.estado_json,'anterior_canon_hex',encode(anterior.estado_canonico,'hex'),'anterior_sha256',anterior.estado_sha256,
  'inventario',inv,'inventario_canon_hex',encode(invcanon,'hex'),'inventario_sha256',encode(sha256(invcanon),'hex'),
  'actuacion_ref',actuacion,'recibo_ref',recibo,'registrada_en',vec_contratacion_temporal.incorporacion75_instante(ahora),'documentos','[]'::jsonb,'recibo',NULL,'estado_posterior',NULL,'posterior_canon_hex','','posterior_sha256','');
 INSERT INTO vec_contratacion_temporal.cierre_administrativo_preparacion_v1 VALUES(recibo,pg_current_xact_id(),session_user,s,x,salida,limite,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora);
 RETURN salida;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CT87: fuente no disponible' USING ERRCODE='P0872';
END $$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_cierre_administrativo_sin_cese_v1(recibo text,e jsonb,canon bytea,huella text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $$
DECLARE prep vec_contratacion_temporal.cierre_administrativo_preparacion_v1%ROWTYPE; r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 p jsonb; s jsonb; x jsonb; a jsonb; inv jsonb; invcanon bytea; validado bytea; maxversion numeric; ahora timestamptz; resultado jsonb; evento text; payload bytea;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CT87: transacción denegada' USING ERRCODE='P0873';END IF;
 SELECT * INTO STRICT prep FROM vec_contratacion_temporal.cierre_administrativo_preparacion_v1 WHERE recibo_ref=recibo FOR SHARE;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF prep.transaccion IS DISTINCT FROM pg_current_xact_id() OR prep.sesion_sql IS DISTINCT FROM session_user OR ahora>=prep.valida_hasta OR ahora<prep.creada_en
  OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.cierre_administrativo_registro_v1 cr WHERE cr.recibo_ref=$1) THEN RAISE EXCEPTION 'CT87: preparación ajena o vencida' USING ERRCODE='P0873';END IF;
 p:=prep.preparacion;s:=prep.solicitud;x:=prep.coordenadas;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=s->>'seguimiento_ref' FOR UPDATE;
 SELECT max(version_seguimiento) INTO maxversion FROM vec_contratacion_temporal.seguimiento_estado_v2 WHERE seguimiento_ref=r.seguimiento_ref;
 IF maxversion IS DISTINCT FROM 1::numeric THEN RAISE EXCEPTION 'CT87: versión en conflicto' USING ERRCODE='P0872';END IF;
 validado:=vec_contratacion_temporal.estado_seguimiento_continuado_canonico_v1(p->'publicacion_original',p->'publicacion_sucesora',e);
 IF canon IS DISTINCT FROM validado OR huella IS DISTINCT FROM encode(sha256(validado),'hex') OR octet_length(canon)>8388608
  OR e#>>'{continuacion,huella_estado_anterior_sha256}' IS DISTINCT FROM p->>'anterior_sha256'
  OR e->>'referencia' IS DISTINCT FROM r.seguimiento_ref OR e->>'organizacion_ref' IS DISTINCT FROM r.organizacion_ref OR e->>'expediente_ref' IS DISTINCT FROM r.expediente_ref
  OR e->>'relacion_ref' IS DISTINCT FROM r.relacion_ref OR e->>'huella_raiz_sha256' IS DISTINCT FROM r.raiz_sha256 THEN RAISE EXCEPTION 'CT87: estado no ligado' USING ERRCODE='P0872';END IF;
 a:=e->'actuaciones'->1;
 IF a->>'actuacion_ref' IS DISTINCT FROM p->>'actuacion_ref' OR a->>'recibo_ref' IS DISTINCT FROM recibo OR a->>'actor_ref' IS DISTINCT FROM x->>'actor_ref'
  OR a->>'unidad_ref' IS DISTINCT FROM x->>'unidad_ref' OR a->>'correlacion_ref' IS DISTINCT FROM x->>'correlacion_ref'
  OR a->>'transicion_clave' IS DISTINCT FROM s->>'transicion_clave' OR a->>'motivo_clave' IS DISTINCT FROM s->>'motivo_clave'
  OR vec_contratacion_temporal.seguimiento73_micro(a->'registrada_en') IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_micro(p->'registrada_en')
  OR jsonb_array_length(vec_contratacion_temporal.seguimiento73_array(a->'documentos',32,0,true))<>0 THEN RAISE EXCEPTION 'CT87: efecto no ligado' USING ERRCODE='P0872';END IF;
 inv:=vec_contratacion_temporal.cierre87_inventario_fuentes(p#>'{inventario,Libro}',r.organizacion_ref,r.expediente_ref,r.seguimiento_ref);
 invcanon:=vec_contratacion_temporal.cierre87_inventario_canonico(inv);
 IF inv IS DISTINCT FROM p->'inventario' OR encode(sha256(invcanon),'hex') IS DISTINCT FROM p->>'inventario_sha256' THEN RAISE EXCEPTION 'CT87: inventario cambió' USING ERRCODE='P0872';END IF;
 INSERT INTO vec_contratacion_temporal.seguimiento_estado_v2(seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,definicion_ref,definicion_version,definicion_sha256,raiz_sha256,
  version_seguimiento,version_anterior,estado_anterior_sha256,estado_json,estado_canonico,estado_sha256)
 VALUES(r.seguimiento_ref,r.organizacion_ref,r.expediente_ref,r.relacion_ref,r.definicion_ref,r.definicion_version,r.definicion_sha256,r.raiz_sha256,2,1,p->>'anterior_sha256',e,canon,huella);
 resultado:=jsonb_build_object('version_resultante',2,'actuacion_ref',a->>'actuacion_ref','recibo_ref',recibo,'actor_ref',a->>'actor_ref','correlacion_ref',a->>'correlacion_ref');
 INSERT INTO vec_contratacion_temporal.cierre_administrativo_registro_v1 VALUES(recibo,r.organizacion_ref,s->>'clave_idempotencia',r.seguimiento_ref,r.expediente_ref,1,2,p->>'anterior_sha256',huella,
  inv#>>'{Libro,Referencia}',(inv#>>'{Libro,Version}')::numeric,encode(sha256(vec_contratacion_temporal.cierre87_libro_canonico(inv->'Libro')),'hex'),
  inv#>>'{Estados,0,Evidencia,Referencia}',inv#>>'{Estados,1,Evidencia,Referencia}',s,inv,invcanon,encode(sha256(invcanon),'hex'),resultado,prep.creada_en);
 INSERT INTO vec_contratacion_temporal.cierre_administrativo_auditoria_v1 VALUES(prep.auditoria_ref,recibo,prep.decision_ref,prep.consumo_sha256,false,prep.creada_en);
 evento:='ref:'||encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
 payload:=vec_contratacion_temporal.cierre87_cabecera('vec.dipgra.contratacion-temporal.cierre-administrativo.evento')
  ||vec_contratacion_temporal.cierre87_texto(r.seguimiento_ref)||vec_contratacion_temporal.cierre87_texto(recibo)||int8send(2)||vec_contratacion_temporal.cierre87_texto(huella)
  ||vec_contratacion_temporal.cierre87_texto('cerrado_administrativamente_sin_cese')||int8send((extract(epoch FROM prep.creada_en)*1000000)::bigint);
 INSERT INTO vec_contratacion_temporal.cierre_administrativo_outbox_v1 VALUES(evento,recibo,huella,payload,encode(sha256(payload),'hex'),prep.creada_en);
 RETURN resultado;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CT87: preparación no disponible' USING ERRCODE='P0872';
 WHEN unique_violation THEN RAISE EXCEPTION 'CT87: cierre en conflicto' USING ERRCODE='P0871';
END $$;

-- Lector interno después de ConsultaDetalleRRHHV3 en la frontera para el mismo
-- expediente/organización. No consume autoridad de escritura, crea preparación
-- ni provisiona libro/sucesora. Sólo SELECT en una instantánea READ ONLY.
CREATE FUNCTION vec_contratacion_temporal.consultar_preparacion_cierre_administrativo_sin_cese_v1(org text,exp text,seg text) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $$
DECLARE r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE; e vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 cfg vec_contratacion_temporal.cierre_administrativo_configuracion_v1%ROWTYPE; libro vec_contratacion_temporal.cierre_administrativo_libro_v1%ROWTYPE;
 o jsonb; p jsonb; tr jsonb; inv jsonb; acciones jsonb:='[]'::jsonb; motivos jsonb; ahora timestamptz; canon bytea; configurada boolean;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR current_setting('transaction_read_only')<>'on' OR current_setting('transaction_isolation')<>'repeatable read'
  OR org IS NULL OR exp IS NULL OR seg IS NULL OR org !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR exp !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR seg !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
  RAISE EXCEPTION 'CT87: consulta interna denegada' USING ERRCODE='P0873';END IF;
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=seg AND organizacion_ref=org AND expediente_ref=exp;
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.seguimiento_estado_v2 WHERE seguimiento_ref=r.seguimiento_ref ORDER BY version_seguimiento DESC LIMIT 1;
 SELECT publicacion_json INTO STRICT o FROM vec_contratacion_temporal.seguimiento_definicion_v2 WHERE definicion_ref=r.definicion_ref AND definicion_version=r.definicion_version AND definicion_sha256=r.definicion_sha256;
 SELECT * INTO cfg FROM vec_contratacion_temporal.cierre_administrativo_configuracion_v1 WHERE organizacion_ref=r.organizacion_ref AND definicion_ref=r.definicion_ref AND definicion_version=r.definicion_version AND definicion_sha256=r.definicion_sha256;configurada:=FOUND;
 IF configurada THEN
  SELECT * INTO STRICT libro FROM vec_contratacion_temporal.cierre_administrativo_libro_v1 WHERE libro_ref=cfg.libro_ref AND version=cfg.libro_version AND huella_sha256=cfg.libro_sha256;
  SELECT publicacion_json INTO STRICT p FROM vec_contratacion_temporal.seguimiento_definicion_v2 WHERE definicion_ref=cfg.definicion_ref AND definicion_version=cfg.sucesora_version AND definicion_sha256=cfg.sucesora_sha256;
  tr:=vec_contratacion_temporal.cierre87_validar_sucesora(o,p);
  IF libro.canon IS DISTINCT FROM vec_contratacion_temporal.cierre87_libro_canonico(libro.publicacion_json) OR libro.huella_sha256 IS DISTINCT FROM encode(sha256(libro.canon),'hex') THEN
   RAISE EXCEPTION 'CT87: libro divergente' USING ERRCODE='P0872';END IF;
 END IF;
 IF e.estado_json ? 'continuacion' THEN
  IF NOT configurada THEN RAISE EXCEPTION 'CT87: continuación sin publicación' USING ERRCODE='P0872';END IF;
  canon:=vec_contratacion_temporal.estado_seguimiento_continuado_canonico_v1(o,p,e.estado_json);
 ELSE canon:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(o,e.estado_json);END IF;
 IF e.version_seguimiento<1 OR e.estado_canonico IS DISTINCT FROM canon OR e.estado_sha256 IS DISTINCT FROM encode(sha256(canon),'hex')
  OR r.raiz_sha256 IS DISTINCT FROM encode(sha256(r.raiz_canonica),'hex') THEN RAISE EXCEPTION 'CT87: estado divergente' USING ERRCODE='P0872';END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF configurada AND e.version_seguimiento=1 AND e.estado_json->>'estado_actual'='vigente' AND ahora>=cfg.publicada_en AND ahora>=libro.publicada_en
  AND ahora>=(p->>'publicado_en')::timestamptz AND ahora>=(p#>>'{vigencia,desde}')::timestamptz
  AND (NOT (p->'vigencia') ? 'hasta' OR (p#>>'{vigencia,hasta}')::timestamptz='0001-01-01T00:00:00Z'::timestamptz OR ahora<(p#>>'{vigencia,hasta}')::timestamptz)
  AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(tr->'documentos') z WHERE z->'obligatorio'='true'::jsonb) THEN
  BEGIN inv:=vec_contratacion_temporal.cierre87_inventario_fuentes(libro.publicacion_json,org,exp,seg);
  EXCEPTION WHEN SQLSTATE 'P0872' THEN inv:=NULL;END;
  IF inv IS NOT NULL THEN
   SELECT jsonb_agg(jsonb_build_object('motivo_clave',value) ORDER BY (value#>>'{}') COLLATE "C") INTO motivos FROM jsonb_array_elements(tr->'motivos_permitidos');
   acciones:=jsonb_build_array(jsonb_build_object('transicion_clave','cerrar_administrativamente_sin_cese','motivos',motivos));
  END IF;
 END IF;
 RETURN jsonb_build_object('expediente_ref',exp,'seguimiento_ref',seg,'version_actual',e.version_seguimiento,'estado_actual',e.estado_json->>'estado_actual','acciones',acciones,
  'preparada_en',vec_contratacion_temporal.incorporacion75_instante(ahora));
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CT87: preparación no disponible' USING ERRCODE='P0872';
END $$;

DO $seguridad$
DECLARE n text; f record; a record;
BEGIN
 FOREACH n IN ARRAY ARRAY['cierre_administrativo_libro_v1','cierre_administrativo_configuracion_v1','cierre_administrativo_preparacion_v1','cierre_administrativo_registro_v1','cierre_administrativo_auditoria_v1','cierre_administrativo_outbox_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true)',n);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',n);
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',n);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace,LATERAL aclexplode(c.relacl) x
   WHERE ns.nspname='vec_contratacion_temporal' AND c.relname=n AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',n,pg_get_userbyid(a.grantee));
  END LOOP;
  EXECUTE format('COMMENT ON TABLE vec_contratacion_temporal.%I IS %L',n,'CT87:administrativo-ejercicio;sin-cese;historia-inmutable');
 END LOOP;
 FOR f IN SELECT p.oid,p.proowner FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal'
  AND (p.proname LIKE 'cierre87_%' OR p.proname IN('estado_seguimiento_continuado_canonico_v1','preparar_cierre_administrativo_sin_cese_v1','confirmar_cierre_administrativo_sin_cese_v1','consultar_preparacion_cierre_administrativo_sin_cese_v1')) LOOP
  IF f.proowner<>current_user::regrole THEN RAISE EXCEPTION 'CT87: propietario de función incompatible' USING ERRCODE='55000';END IF;
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(p.proacl) x WHERE p.oid=f.oid AND x.grantee<>0 AND x.grantee<>f.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.oid::regprocedure,pg_get_userbyid(a.grantee));
  END LOOP;
 END LOOP;
END $seguridad$;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_cierre_administrativo_sin_cese_v1(jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.confirmar_cierre_administrativo_sin_cese_v1(text,jsonb,bytea,text) TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_preparacion_cierre_administrativo_sin_cese_v1(text,text,text) TO vec_contratacion_temporal_ejecutor;
COMMIT;
