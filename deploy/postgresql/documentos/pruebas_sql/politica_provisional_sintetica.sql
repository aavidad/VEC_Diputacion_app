\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor efímero de probar_integracion_pg18.sh.
-- Requiere custodia_externa_sintetica.sql (ensayo_externa.preparar y las
-- fachadas AD3 sintéticas). Ejercita el estado de política provisional en un
-- expediente propio (exp ...0003) para no alterar las cuentas del resto del
-- ensayo. No es una prueba COSE ni se instala en otra base.
BEGIN;
CREATE SCHEMA ensayo_provisional AUTHORIZATION postgres;
GRANT USAGE ON SCHEMA ensayo_provisional TO vec_documentos_ensayo;

CREATE FUNCTION ensayo_provisional.preimagen_alta(p_id text,p_clave text,p_estado text) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT convert_to('{"accion":"documentos.generado.alta","id":"'||p_id||'","clave_idempotencia":"'||p_clave||'","modulo_id":"dietas","expediente_ref":"exp:00000000-0000-4000-8000-000000000003","tipo_ref":"tipo:00000000-0000-4000-8000-000000000001","version":1,"mime":"application/pdf","tamano":3,"huella_sha256":"'||repeat('a',64)||'","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"'||repeat('c',64)||'","proteccion":"conservacion","conservacion_hasta":"2030-01-01T00:00:00Z","estado_politica":"'||p_estado||'"}','UTF8')
$f$;

CREATE FUNCTION ensayo_provisional.preimagen_externa(p_id text,p_clave text,p_estado text) RETURNS bytea
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT convert_to('{"accion":"documentos.externo.registrar","id":"'||p_id||'","clave_idempotencia":"'||p_clave||'","modulo_id":"dietas","expediente_ref":"exp:00000000-0000-4000-8000-000000000003","tipo_ref":"tipo:00000000-0000-4000-8000-000000000002","version":1,"mime":"","tamano":0,"huella_sha256":"'||repeat('e',64)||'","custodio_id":"dietas.justificantes","custodia_ref":"justificante:provisional:0001","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"'||repeat('c',64)||'","proteccion":"conservacion","conservacion_hasta":"2030-01-01T00:00:00Z","estado_politica":"'||p_estado||'"}','UTF8')
$f$;

-- Recibo del conector: retenido_hasta JSON null significa sin retención.
CREATE FUNCTION ensayo_provisional.objeto(p_retenido jsonb,p_inmovilizado boolean) RETURNS jsonb
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('objeto_ref','obj:00000000-0000-4000-8000-0000000000c1','objeto_version','ov1',
  'conector_ref','ficheros_ensayo','recibo_objeto_ref','recibo:00000000-0000-4000-8000-0000000000c1',
  'recibo_objeto_huella_sha256',repeat('d',64),'retenido_hasta',p_retenido,'inmovilizado',p_inmovilizado,
  'mime','application/pdf','tamano',3,'huella_sha256',repeat('a',64))
$f$;

CREATE FUNCTION ensayo_provisional.alta(p_caso text,p_id text,p_clave text,p_estado text,p_objeto jsonb,p_decision text) RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE m ensayo_externa.material; r jsonb;
BEGIN
 PERFORM ensayo_externa.preparar(p_caso,'documentos.generado.alta',p_id,'exp:00000000-0000-4000-8000-000000000003',
  'alta_documento_generado','documento_generado','["documento","recibo"]',
  ensayo_provisional.preimagen_alta(p_id,p_clave,p_estado),p_decision);
 SELECT * INTO STRICT m FROM ensayo_externa.material WHERE caso=p_caso;
 SELECT vec_documentos.confirmar_alta_v2(m.preimagen,p_objeto,m.auth,m.capacidad,m.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 RETURN r;
END $f$;

CREATE FUNCTION ensayo_provisional.externa(p_caso text,p_id text,p_clave text,p_estado text,p_decision text) RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM ensayo_externa.preparar(p_caso,'documentos.externo.registrar',p_id,'exp:00000000-0000-4000-8000-000000000003',
  'registrar_documento_externo','documento_externo','["documento","recibo"]',
  ensayo_provisional.preimagen_externa(p_id,p_clave,p_estado),p_decision);
 RETURN ensayo_externa.invocar(p_caso);
END $f$;

-- Ejecuta una operación y devuelve su SQLSTATE ('ok' si no falla); la
-- subtransacción revierte cualquier consumo sintético del caso fallido.
CREATE FUNCTION ensayo_provisional.estado(p_sql text) RETURNS text
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 EXECUTE p_sql;
 RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;
END $f$;

CREATE FUNCTION ensayo_provisional.probar() RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r jsonb; r2 jsonb; m ensayo_externa.material; e text; c record;
 id text:='doc:00000000-0000-4000-8000-0000000000c1'; clave text:='idem:00000000-0000-4000-8000-0000000000c1';
 sin_retencion jsonb:=ensayo_provisional.objeto('null'::jsonb,false);
BEGIN
 -- 1. Alta provisional sin retención: se confirma y el recibo declara el estado.
 r:=ensayo_provisional.alta('prov_alta',id,clave,'provisional',sin_retencion,'decision:00000000-0000-4000-8000-0000000000c1');
 IF r->>'estado_politica' IS DISTINCT FROM 'provisional' OR r->>'estado_firma'<>'pendiente_proveedor'
    OR r->>'numero_vec' !~ '^VEC-[0-9]{4}-[0-9]+$' OR r ? 'retenido_hasta'
 THEN RAISE EXCEPTION 'alta provisional incoherente: %',r; END IF;
 -- Replay con el mismo material: el mismo recibo.
 SELECT * INTO STRICT m FROM ensayo_externa.material WHERE caso='prov_alta';
 SELECT vec_documentos.confirmar_alta_v2(m.preimagen,sin_retencion,m.auth,m.capacidad,m.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r2;
 IF r2 IS DISTINCT FROM r THEN RAISE EXCEPTION 'replay provisional cambió el recibo'; END IF;

 -- 2. Estados fuera de catálogo y combinaciones incoherentes: denegados (42501)
 -- antes de consumir la autorización.
 FOR c IN SELECT * FROM (VALUES
   ('fuera_catalogo','doc:00000000-0000-4000-8000-0000000000c2','definitiva','null'::jsonb,false),
   ('retirada','doc:00000000-0000-4000-8000-0000000000c3','retirada','null'::jsonb,false),
   ('vacio','doc:00000000-0000-4000-8000-0000000000c4','','null'::jsonb,false),
   ('provisional_retenido','doc:00000000-0000-4000-8000-0000000000c5','provisional','"2031-01-01T00:00:00Z"'::jsonb,false),
   ('provisional_inmovilizado','doc:00000000-0000-4000-8000-0000000000c6','provisional','null'::jsonb,true),
   ('aprobada_sin_retencion','doc:00000000-0000-4000-8000-0000000000c7','aprobada','null'::jsonb,false)
  ) AS v(nombre,doc,estado_politica,retenido,inmovilizado) LOOP
  e:=ensayo_provisional.estado(format('SELECT ensayo_provisional.alta(%L,%L,%L,%L,%L::jsonb,%L)',
   c.nombre,c.doc,replace(c.doc,'doc:','idem:'),c.estado_politica,
   ensayo_provisional.objeto(c.retenido,c.inmovilizado),replace(c.doc,'doc:','decision:')));
  IF e<>'42501' THEN RAISE EXCEPTION 'FALLO %: sqlstate % (se esperaba 42501)',c.nombre,e; END IF;
 END LOOP;
 -- Un campo de estado ausente cambia la lista cerrada de campos: denegado.
 e:=ensayo_provisional.estado($s$SELECT vec_documentos.confirmar_alta_v2(
   convert_to(replace(convert_from(ensayo_provisional.preimagen_alta('doc:00000000-0000-4000-8000-0000000000c8','idem:00000000-0000-4000-8000-0000000000c8','provisional'),'UTF8'),',"estado_politica":"provisional"',''),'UTF8'),
   ensayo_provisional.objeto('null'::jsonb,false),'{}'::jsonb,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)$s$);
 IF e<>'42501' THEN RAISE EXCEPTION 'FALLO: alta sin estado de política: sqlstate %',e; END IF;

 -- 3. Replay de la misma clave con el estado cambiado (aprobada con retención)
 -- y decisión fresca: conflicto, sin sustituir el recibo provisional.
 e:=ensayo_provisional.estado(format('SELECT ensayo_provisional.alta(%L,%L,%L,%L,%L::jsonb,%L)',
  'prov_cambio',id,clave,'aprobada',ensayo_provisional.objeto('"2031-01-01T00:00:00Z"'::jsonb,false),
  'decision:00000000-0000-4000-8000-0000000000c9'));
 IF e<>'23505' THEN RAISE EXCEPTION 'FALLO: replay con estado cambiado: sqlstate % (se esperaba 23505)',e; END IF;

 -- 4. Custodia externa provisional: se registra y declara el estado; estado
 -- fuera de catálogo denegado; replay con estado cambiado en conflicto.
 r:=ensayo_provisional.externa('prov_externa','doc:00000000-0000-4000-8000-0000000000d1','idem:00000000-0000-4000-8000-0000000000d1',
  'provisional','decision:00000000-0000-4000-8000-0000000000d1');
 IF r->>'estado_politica' IS DISTINCT FROM 'provisional' OR r->>'custodia'<>'externa'
 THEN RAISE EXCEPTION 'registro externo provisional incoherente: %',r; END IF;
 e:=ensayo_provisional.estado($s$SELECT ensayo_provisional.externa('ext_fuera','doc:00000000-0000-4000-8000-0000000000d2',
  'idem:00000000-0000-4000-8000-0000000000d2','definitiva','decision:00000000-0000-4000-8000-0000000000d2')$s$);
 IF e<>'42501' THEN RAISE EXCEPTION 'FALLO: externa con estado fuera de catálogo: sqlstate %',e; END IF;
 e:=ensayo_provisional.estado($s$SELECT ensayo_provisional.externa('ext_cambio','doc:00000000-0000-4000-8000-0000000000d1',
  'idem:00000000-0000-4000-8000-0000000000d1','aprobada','decision:00000000-0000-4000-8000-0000000000d3')$s$);
 IF e<>'23505' THEN RAISE EXCEPTION 'FALLO: externa con estado cambiado: sqlstate %',e; END IF;
END $f$;

GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA ensayo_provisional TO vec_documentos_ensayo;
COMMIT;
