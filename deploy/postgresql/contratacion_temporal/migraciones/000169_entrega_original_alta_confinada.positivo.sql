\set ON_ERROR_STOP on
-- Sólo clon sintético. DOBLE NOMINAL V3 DECLARADO: se sustituye temporalmente
-- el consumidor de entrega, no CT169 ni CT150. No acredita COSE/PDP vigente.
-- Todas las filas son fixtures nuevas; triggers y constraints siguen activos.
-- Los sellos HMAC de fixture sólo comprueban correlación y generación; su
-- validez criptográfica queda para la prueba Go con las claves gobernadas.
SELECT md5(coalesce((SELECT prosrc FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.registrar_y_consumir_entrega_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'')||coalesce((SELECT string_agg(j::text,'' ORDER BY j::text) FROM (SELECT to_jsonb(x) j FROM vec_contratacion_temporal.confirmacion_agregado_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.expediente_alta_version x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_acceso x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_reserva x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.alias_ambito_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.alias_huella_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_outbox x) q),'')) AS huella \gset ct169_fuente_
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE ct169_fuente AS
 SELECT jsonb_build_object('identidad',to_jsonb(i),'version_reserva',to_jsonb(rv),
   'alta',to_jsonb(b),'version_alta',to_jsonb(v),'actuacion',to_jsonb(t),
   'auditoria',to_jsonb(a),'outbox',to_jsonb(o),'confirmacion',to_jsonb(c),
   'peticion',to_jsonb(p),'acceso',to_jsonb(ac),'acceso_confirmacion',to_jsonb(ac2),'reserva_entrega',to_jsonb(s),
   'confirmacion_entrega',to_jsonb(ec)) datos
 FROM vec_contratacion_temporal.entrega_peticion_centro_reserva s
 JOIN vec_contratacion_temporal.entrega_peticion_centro_confirmacion ec USING(peticion_ref)
 JOIN vec_contratacion_temporal.alias_ambito_alta x ON x.alias_hmac=s.ambito_alta_hmac
 JOIN vec_contratacion_temporal.identidad_reserva_alta i ON i.ambito_hmac=x.ambito_raiz_hmac
 JOIN vec_contratacion_temporal.confirmacion_agregado_alta c ON c.ambito_hmac=i.ambito_hmac
 JOIN vec_contratacion_temporal.reserva_alta_version rv ON rv.ambito_hmac=i.ambito_hmac AND rv.revision=c.reserva_revision
 JOIN vec_contratacion_temporal.expediente_alta b ON b.expediente_ref=c.expediente_ref
 JOIN vec_contratacion_temporal.expediente_alta_version v ON v.expediente_ref=c.expediente_ref AND v.version=1
 JOIN vec_contratacion_temporal.actuacion_alta t ON t.expediente_ref=c.expediente_ref
 JOIN vec_contratacion_temporal.auditoria_alta a ON a.auditoria_ref=c.auditoria_ref
 JOIN vec_contratacion_temporal.outbox_alta o ON o.evento_ref=c.evento_ref
 JOIN vec_contratacion_temporal.peticion_centro_revision p ON p.peticion_ref=s.peticion_ref AND p.version=2
 JOIN vec_contratacion_temporal.entrega_peticion_centro_acceso ac ON ac.acceso_ref=s.acceso_preparacion_ref
 JOIN vec_contratacion_temporal.entrega_peticion_centro_acceso ac2 ON ac2.acceso_ref=ec.acceso_confirmacion_ref LIMIT 1;
CREATE TEMP TABLE ct169_mapa(viejo text PRIMARY KEY,nuevo text NOT NULL);
CREATE TEMP TABLE ct169_casos(caso text PRIMARY KEY,material text,decision bytea,
 esperado jsonb,estado text,error_esperado text,canon_sha256 text,fecha text);
CREATE FUNCTION pg_temp.ct169_remap(j jsonb) RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE t text:=j::text; r record;
BEGIN
 FOR r IN SELECT * FROM ct169_mapa ORDER BY length(viejo) DESC LOOP
  t:=replace(t,r.viejo,r.nuevo);
 END LOOP;
 RETURN t::jsonb;
END $f$;
CREATE FUNCTION pg_temp.ct169_insertar(tabla text,j jsonb) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE format('INSERT INTO vec_contratacion_temporal.%I SELECT (jsonb_populate_record(NULL::vec_contratacion_temporal.%I,$1)).*',tabla,tabla) USING j;
END $f$;
DO $fixtures$
DECLARE f jsonb; j jsonb; a jsonb; b bytea; p jsonb; politica jsonb; recibo jsonb;
 n integer:=0; caso text; con_fin boolean; confirmada boolean; historica boolean;
 claves text[]:=ARRAY['ambito_hmac','reserva_ref','expediente_ref','numero_visible','recibo_ref',
  'auditoria_ref','evento_ref','confirmacion_ref','decision_ref','efecto_ref'];
 k text; v text; z text; r record; t text; x record; actual text; m text; contexto text;
 huella text; payload bytea; fila jsonb; esperado jsonb;
BEGIN
 SELECT datos INTO STRICT f FROM ct169_fuente;
 FOREACH caso IN ARRAY ARRAY['legado_preparada','legado_confirmada_historica',
    'fin_preparada','fin_confirmada_historica','legado_autoenlace_historico','alias_incompleto','alias_generacion','sin_alta_confirmada'] LOOP
  n:=n+1; con_fin:=caso LIKE 'fin_%';
  confirmada:=caso IN ('legado_confirmada_historica','fin_confirmada_historica'); historica:=caso LIKE '%historica%' OR caso LIKE '%historico%';
  DELETE FROM ct169_mapa;
  FOREACH t IN ARRAY ARRAY['identidad','confirmacion','peticion','acceso','acceso_confirmacion','reserva_entrega','confirmacion_entrega'] LOOP
   j:=f->t;
   FOREACH k IN ARRAY claves||ARRAY['peticion_ref','acceso_ref','acceso_preparacion_ref','acceso_confirmacion_ref','entrega_ref','consumo_huella_sha256'] LOOP
    v:=j->>k;
    IF v IS NOT NULL THEN
     z:=CASE WHEN k='numero_visible' THEN '2099/CT169_'||n
       WHEN k='confirmacion_ref' THEN 'cnf_ct_'||md5(caso||v)
       WHEN k LIKE '%hmac' THEN regexp_replace(v,'[a-f0-9]{64}$',encode(sha256(convert_to(caso||v,'UTF8')),'hex'))
       WHEN k LIKE '%sha256' THEN encode(sha256(convert_to(caso||v,'UTF8')),'hex')
       ELSE 'fixture:ct169:'||md5(caso||v) END;
     INSERT INTO ct169_mapa VALUES(v,z) ON CONFLICT DO NOTHING;
    END IF;
   END LOOP;
  END LOOP;
  FOR x IN SELECT alias_hmac FROM vec_contratacion_temporal.alias_ambito_alta WHERE ambito_raiz_hmac=f#>>'{identidad,ambito_hmac}'
    UNION SELECT alias_hmac FROM vec_contratacion_temporal.alias_huella_alta WHERE ambito_raiz_hmac=f#>>'{identidad,ambito_hmac}' LOOP
   INSERT INTO ct169_mapa VALUES(x.alias_hmac,regexp_replace(x.alias_hmac,'[a-f0-9]{64}$',encode(sha256(convert_to(caso||x.alias_hmac,'UTF8')),'hex'))) ON CONFLICT DO NOTHING;
  END LOOP;
  FOR k,v IN SELECT key,value#>>'{}' FROM jsonb_each(f->'confirmacion') WHERE key LIKE '%sha256' LOOP
   INSERT INTO ct169_mapa VALUES(v,encode(sha256(convert_to(caso||v,'UTF8')),'hex')) ON CONFLICT DO NOTHING;
  END LOOP;
  a:=pg_temp.ct169_remap(convert_from(decode(substr(f#>>'{version_alta,alta_canonica}',3),'hex'),'UTF8')::jsonb);
  -- to_jsonb(bytea) usa el formato hex de PostgreSQL.
  IF con_fin THEN
   politica:=jsonb_build_object('regla_ref','regla:ct169:fin','catalogo_version',1,
     'catalogo_huella_sha256',repeat('a',64),'fecha_fin','obligatoria');
   a:=jsonb_set(a,'{solicitud,periodo,politica_fin}',politica,true);
  ELSE politica:=NULL; END IF;
  b:=vec_contratacion_temporal.reconstruir_efecto_alta_v2(a);
  huella:=encode(sha256(b),'hex');
  IF caso<>'sin_alta_confirmada' THEN
  j:=pg_temp.ct169_remap(f->'identidad');
  PERFORM pg_temp.ct169_insertar('identidad_reserva_alta',j);
  FOR r IN SELECT to_jsonb(y) datos FROM vec_contratacion_temporal.alias_ambito_alta y WHERE ambito_raiz_hmac=f#>>'{identidad,ambito_hmac}' LOOP
   PERFORM pg_temp.ct169_insertar('alias_ambito_alta',pg_temp.ct169_remap(r.datos));
  END LOOP;
  IF caso<>'alias_incompleto' THEN
   FOR r IN SELECT to_jsonb(y) datos FROM vec_contratacion_temporal.alias_huella_alta y WHERE ambito_raiz_hmac=f#>>'{identidad,ambito_hmac}' LOOP
    fila:=pg_temp.ct169_remap(r.datos);
    IF caso='alias_generacion' THEN
     fila:=fila||jsonb_build_object('generacion',(fila->>'generacion')::integer+100,
       'alias_hmac',replace(fila->>'alias_hmac','/v'||(fila->>'generacion')||':','/v'||((fila->>'generacion')::integer+100)::text||':'));
    END IF;
    PERFORM pg_temp.ct169_insertar('alias_huella_alta',fila);
   END LOOP;
  END IF;
  PERFORM pg_temp.ct169_insertar('reserva_alta_version',pg_temp.ct169_remap(f->'version_reserva'));
  PERFORM pg_temp.ct169_insertar('expediente_alta',pg_temp.ct169_remap(f->'alta'));
  j:=pg_temp.ct169_remap(f->'version_alta')||jsonb_build_object('alta_canonica',b,'huella_alta_sha256',huella);
  PERFORM pg_temp.ct169_insertar('expediente_alta_version',j);
  PERFORM pg_temp.ct169_insertar('actuacion_alta',pg_temp.ct169_remap(f->'actuacion'));
  j:=pg_temp.ct169_remap(f->'auditoria')||jsonb_build_object('secuencia',(SELECT max(secuencia)+1 FROM vec_contratacion_temporal.auditoria_alta));
  PERFORM pg_temp.ct169_insertar('auditoria_alta',j);
  payload:=convert_to(pg_temp.ct169_remap(convert_from(decode(substr(f#>>'{outbox,payload_canonico}',3),'hex'),'UTF8')::jsonb)::text,'UTF8');
  j:=pg_temp.ct169_remap(f->'outbox')||jsonb_build_object('secuencia',(SELECT max(secuencia)+1 FROM vec_contratacion_temporal.outbox_alta),
     'payload_canonico',payload,'payload_huella_sha256',encode(sha256(payload),'hex'));
  PERFORM pg_temp.ct169_insertar('outbox_alta',j);
  j:=pg_temp.ct169_remap(f->'confirmacion')||jsonb_build_object('huella_alta_sha256',huella,
    'auditoria_secuencia',(SELECT max(secuencia) FROM vec_contratacion_temporal.auditoria_alta),
    'outbox_secuencia',(SELECT max(secuencia) FROM vec_contratacion_temporal.outbox_alta),'payload_huella_sha256',encode(sha256(payload),'hex'));
  PERFORM pg_temp.ct169_insertar('confirmacion_agregado_alta',j);
  END IF;
  j:=pg_temp.ct169_remap(f->'peticion');
  IF con_fin THEN j:=jsonb_set(j,'{peticion,solicitud,periodo,politica_fin}',politica,true); END IF;
  j:=j||jsonb_build_object('clave_idempotencia',gen_random_uuid(),'material_sha256',encode(sha256(convert_to(j->>'material','UTF8')),'hex'));
  PERFORM pg_temp.ct169_insertar('peticion_centro_revision',j);
  p:=j->'peticion';
  PERFORM pg_temp.ct169_insertar('entrega_peticion_centro_acceso',pg_temp.ct169_remap(f->'acceso'));
  j:=pg_temp.ct169_remap(f->'reserva_entrega')||jsonb_build_object('clave_alta',gen_random_uuid());
  PERFORM pg_temp.ct169_insertar('entrega_peticion_centro_reserva',j);
  IF confirmada OR caso IN ('alias_incompleto','alias_generacion') THEN
   PERFORM pg_temp.ct169_insertar('entrega_peticion_centro_acceso',pg_temp.ct169_remap(f->'acceso_confirmacion'));
   PERFORM pg_temp.ct169_insertar('entrega_peticion_centro_confirmacion',pg_temp.ct169_remap(f->'confirmacion_entrega'));
  END IF;
  actual:=CASE WHEN historica THEN 'perfil:ct169:actual' ELSE j->>'perfil_ref' END;
  m:=jsonb_build_object('modo','preparar','actor_ref',j->>'actor_ref','perfil_ref',actual,
   'peticion_ref',j->>'peticion_ref','version_esperada',2,'clave_alta_candidata',gen_random_uuid(),
   'ambito_alta_hmac',j->>'ambito_alta_hmac','centro_ref',p#>>'{solicitud,centro_ref}',
   'categoria_ref',p#>>'{solicitud,categoria_ref}')::text;
  contexto:='{"ambitos":{"categoria_ref":'||vec_contratacion_temporal.texto_json_go_v1(p#>>'{solicitud,categoria_ref}')||
   ',"centro_ref":'||vec_contratacion_temporal.texto_json_go_v1(p#>>'{solicitud,centro_ref}')||
   ',"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(m,'UTF8')),'hex')||'"}}';
  esperado:=jsonb_build_object('perfil_ref',j->>'perfil_ref','flujo',a->'flujo','politica_fin',politica,
    'ambito_hmac',j->>'ambito_alta_hmac','recibo_alta',pg_temp.ct169_remap(f#>'{confirmacion_entrega,recibo_alta}'),
    'huella_peticion_hmac',(SELECT h.alias_hmac FROM vec_contratacion_temporal.alias_huella_alta h
      JOIN vec_contratacion_temporal.alias_ambito_alta ax ON ax.ambito_raiz_hmac=h.ambito_raiz_hmac AND ax.generacion=h.generacion
      WHERE ax.alias_hmac=j->>'ambito_alta_hmac'));
  INSERT INTO ct169_casos VALUES(caso,m,convert_to(jsonb_build_object('accion','contratacion_temporal.peticion_centro.rrhh.entregar',
    'modulo_id','contratacion_temporal','tipo_recurso','entrega_peticion_centro','finalidad','tramitar_peticion_centro_rrhh',
    'recurso_ref',j->>'peticion_ref','principal_id',j->>'actor_ref','perfil_activo_ref',actual,
    'contexto_recurso_huella_sha256',encode(sha256(convert_to(contexto,'UTF8')),'hex'))::text,'UTF8'),esperado,
    CASE WHEN confirmada OR caso IN ('legado_autoenlace_historico','alias_incompleto','alias_generacion') THEN 'confirmada' ELSE 'preparada' END,
    CASE WHEN caso IN ('alias_incompleto','alias_generacion') THEN '55000' END,huella,a->>'creado_en');
 END LOOP;
END
$fixtures$;
SET CONSTRAINTS ALL IMMEDIATE;
SET CONSTRAINTS ALL DEFERRED;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_entrega_peticion_centro_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $doble_nominal_declarado$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:ct169:'||gen_random_uuid(),d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
   encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),'auditoria:ct169:'||gen_random_uuid(),clock_timestamp(),true;
END $doble_nominal_declarado$;
CREATE ROLE vec_ct169_positivo_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct169_positivo_login;
GRANT SELECT ON ct169_casos TO vec_ct169_positivo_login;
SET SESSION AUTHORIZATION vec_ct169_positivo_login;
DO $positivo$
DECLARE c record; r record; i integer:=0;
BEGIN
 FOR c IN SELECT * FROM ct169_casos ORDER BY caso LOOP
  BEGIN
   SELECT * INTO STRICT r FROM vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
     c.material,NULL,c.decision,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
   IF c.error_esperado IS NOT NULL THEN RAISE EXCEPTION 'CT169 aceptó alias incompleto'; END IF;
   IF c.caso='sin_alta_confirmada' THEN
    IF r.original_alta IS NOT NULL OR r.entrega->>'estado_entrega' IS DISTINCT FROM 'preparada' THEN
     RAISE EXCEPTION 'CT169 fabricó original sin alta confirmada';
    END IF;
   ELSIF r.entrega->>'estado_entrega' IS DISTINCT FROM c.estado
      OR r.original_alta->'flujo' IS DISTINCT FROM c.esperado->'flujo'
      OR r.original_alta->'politica_fin' IS DISTINCT FROM c.esperado->'politica_fin'
      OR r.original_alta->>'perfil_ref' IS DISTINCT FROM c.esperado->>'perfil_ref'
      OR r.original_alta->>'ambito_hmac' IS DISTINCT FROM c.esperado->>'ambito_hmac'
      OR r.original_alta->'recibo_alta' IS DISTINCT FROM c.esperado->'recibo_alta'
      OR r.original_alta->>'huella_peticion_hmac' IS DISTINCT FROM c.esperado->>'huella_peticion_hmac'
      OR substring(r.original_alta->>'huella_peticion_hmac' FROM '/v([1-9][0-9]{0,8}):')
         IS DISTINCT FROM substring(r.original_alta->>'ambito_hmac' FROM '/v([1-9][0-9]{0,8}):') THEN
      RAISE EXCEPTION 'CT169 devolvió material diferente en %',c.caso;
   END IF;
   i:=i+1;
   IF c.caso='sin_alta_confirmada' THEN
    RAISE NOTICE 'CT169 positivo funcional sin_alta_confirmada: preparada y original NULL';
   ELSE
    RAISE NOTICE 'CT169 positivo funcional %: original mínimo idéntico al canon efecto-alta.v2 SHA256 %, fecha %, flujo/perfil/política/par HMAC/recibo idénticos',c.caso,c.canon_sha256,c.fecha;
   END IF;
  EXCEPTION WHEN SQLSTATE '55000' THEN
   IF c.error_esperado IS DISTINCT FROM '55000' THEN RAISE; END IF;
   i:=i+1; RAISE NOTICE 'CT169 negativo funcional %: cerrado 55000',c.caso;
  END;
 END LOOP;
 IF i<>8 THEN RAISE EXCEPTION 'CT169 casos incompletos'; END IF;
END
$positivo$;
RESET SESSION AUTHORIZATION;
SET CONSTRAINTS ALL IMMEDIATE;
ROLLBACK;
SELECT 1/(CASE WHEN md5(coalesce((SELECT prosrc FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.registrar_y_consumir_entrega_peticion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'')||coalesce((SELECT string_agg(j::text,'' ORDER BY j::text) FROM (SELECT to_jsonb(x) j FROM vec_contratacion_temporal.confirmacion_agregado_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.expediente_alta_version x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_acceso x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_reserva x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.alias_ambito_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.alias_huella_alta x UNION ALL SELECT to_jsonb(x) j FROM vec_contratacion_temporal.entrega_peticion_centro_outbox x) q),''))=:'ct169_fuente_huella' THEN 1 ELSE 0 END) AS restaurada \gset ct169_
\echo CT169-ROLLBACK-FUENTE-Y-CONSUMIDOR-IDENTICOS
\echo CT169-POSITIVO-FUNCIONAL-DOBLE-NOMINAL-V3-OK
