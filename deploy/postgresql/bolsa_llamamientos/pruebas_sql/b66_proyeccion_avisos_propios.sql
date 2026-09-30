\set ON_ERROR_STOP on
-- Sólo PG18 desechable con B62/B65/B66. Datos opacos sintéticos, ROLLBACK.
-- Los dos consumidores V3 se sustituyen dentro de esta transacción: se prueba
-- la proyección y las guardas Own, no COSE, auditoría institucional ni HTTP.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regclass('vec_bolsa_llamamientos.aviso_externo_outbox') IS NULL
 THEN RAISE EXCEPTION 'B66: ensayo requiere clon DBA sintético'; END IF;
END $pre$;
CREATE TEMP TABLE b66_propia AS
 SELECT v.candidato_ref,p.participacion_ref,p.bolsa_ref,p.categoria_ref
 FROM vec_bolsa_llamamientos.vinculo_candidato v
 CROSS JOIN LATERAL vec_bolsa_llamamientos.listar_participaciones_candidato_v1(v.candidato_ref) p
 WHERE v.participacion_ref=p.participacion_ref AND v.candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$'
 ORDER BY p.bolsa_ref,p.participacion_ref LIMIT 1;
DO $fixture$ BEGIN
 IF (SELECT count(*) FROM pg_temp.b66_propia)<>1
 THEN RAISE EXCEPTION 'B66: falta participación sintética legible'; END IF;
END $fixture$;
CREATE TEMP TABLE b66_consumos(tipo text NOT NULL);
CREATE TEMP TABLE b66_casos AS
 SELECT n, 'llamamiento:'||encode(sha256(convert_to('b66:llamamiento:'||n,'UTF8')),'hex') AS llamamiento,
 'recibo:llamamiento:'||encode(sha256(convert_to('b66:llamamiento:'||n,'UTF8')),'hex') AS recibo,
 '2040-01-01T00:00:00Z'::timestamptz+n*interval '1 minute' AS emitido,
 CASE n WHEN 2 THEN 'reservado_incierto' WHEN 3 THEN 'aceptado'
  WHEN 4 THEN 'no_aceptado' WHEN 5 THEN 'sin_destino' WHEN 8 THEN 'aceptado' WHEN 26 THEN 'aceptado' END AS estado
 FROM generate_series(1,26) n;
DO $temp$ DECLARE esquema text; BEGIN
 SELECT nspname INTO STRICT esquema FROM pg_namespace WHERE oid=pg_my_temp_schema();
 EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_bolsa_llamamientos_propietario,vec_autorizacion_atestada_v3_propietario',esquema);
END $temp$;
GRANT SELECT ON pg_temp.b66_propia,pg_temp.b66_casos TO vec_bolsa_llamamientos_propietario;
GRANT INSERT,SELECT ON pg_temp.b66_consumos TO vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 INSERT INTO pg_temp.b66_consumos VALUES('mi_bolsa');
 RETURN QUERY SELECT 'decision:b66:doble',(convert_from(p_capacidad,'UTF8')::jsonb)->>'efecto_ref',repeat('a',64),repeat('a',64),'auditoria:b66:doble',clock_timestamp(),true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 INSERT INTO pg_temp.b66_consumos VALUES('historial');
 RETURN QUERY SELECT 'decision:b66:doble',(convert_from(p_capacidad,'UTF8')::jsonb)->>'efecto_ref',repeat('a',64),repeat('a',64),'auditoria:b66:doble',clock_timestamp(),true;
END $f$;
-- Las altas de ensayo usan tablas reales con constraints y RLS; no se desactivan
-- triggers, FK ni inmutabilidad. No se ejecuta ningún transporte.
INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
 llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,
 huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
SELECT c.llamamiento,c.recibo,p.bolsa_ref,'per_AAAAAAAAAAAAAAAAAAAAAA','b66:clave:'||c.n,
 jsonb_build_array(p.participacion_ref),jsonb_build_object('plantilla_version','bolsa-llamamiento-v1'),
 repeat('a',64),decode(repeat('a',64),'hex'),'emision_reservada',c.emitido,'decision:b66:'||c.n
FROM pg_temp.b66_casos c CROSS JOIN pg_temp.b66_propia p;
INSERT INTO vec_bolsa_llamamientos.aviso_externo_outbox(
 productor_ref,evento_ref,llamamiento_ref,participacion_ref,evento,canon,huella_sha256,recibo_outbox_ref,registrada_en)
SELECT 'productor:bolsa:b66',e.evento->>'evento_ref',c.llamamiento,p.participacion_ref,e.evento,
 canon.valor,encode(sha256(convert_to(canon.valor,'UTF8')),'hex'),
 'recibo_outbox:'||encode(sha256(convert_to('b66:outbox:'||c.n,'UTF8')),'hex'),
 c.emitido+CASE WHEN c.n=8 THEN interval '25 seconds' ELSE interval '0 seconds' END
FROM pg_temp.b66_casos c CROSS JOIN pg_temp.b66_propia p CROSS JOIN LATERAL (
 SELECT jsonb_build_object('evento_ref','evento_aviso:'||encode(sha256(convert_to(c.llamamiento||chr(31)||p.participacion_ref,'UTF8')),'hex'),
 'productor_ref','productor:bolsa:b66','tipo_versionado','vec.bolsa.aviso-llamamiento.v1',
 'ocurrido_en',to_char(c.emitido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 'correlacion_ref','corr_b66_sintetica','destinatario_externo_ref',p.candidato_ref,
 'comunicacion_ref',c.llamamiento,'plantilla_ref','bolsa-llamamiento-v1',
 'plantilla_version','bolsa-llamamiento-v1','recurso_publico_ref','') AS evento
) e CROSS JOIN LATERAL (SELECT vec_bolsa_llamamientos.canon_aviso_externo_v1(e.evento) AS valor) canon
WHERE c.n<>7;
INSERT INTO vec_bolsa_llamamientos.aviso_externo_aceptacion(
 productor_ref,evento_ref,huella_sha256,recibo_externo_ref,aceptada_en)
SELECT o.productor_ref,o.evento_ref,o.huella_sha256,
 'aviso_recibo:'||substr(encode(sha256(convert_to('b66:ack:'||c.n,'UTF8')),'hex'),1,32),
 greatest(c.emitido+interval '1 second',o.registrada_en)
FROM pg_temp.b66_casos c JOIN vec_bolsa_llamamientos.aviso_externo_outbox o ON o.llamamiento_ref=c.llamamiento
WHERE c.n IN(2,3,4,5,6,8,26);
-- Incierto primero y terminal después: versión más reciente visible al corte.
INSERT INTO vec_bolsa_llamamientos.aviso_externo_resultado(
 productor_ref,evento_ref,version,huella_sha256,recibo_externo_ref,estado,registrada_en)
SELECT o.productor_ref,o.evento_ref,1,o.huella_sha256,a.recibo_externo_ref,'reservado_incierto',c.emitido+interval '5 seconds'
FROM pg_temp.b66_casos c JOIN vec_bolsa_llamamientos.aviso_externo_outbox o ON o.llamamiento_ref=c.llamamiento
JOIN vec_bolsa_llamamientos.aviso_externo_aceptacion a USING(productor_ref,evento_ref)
WHERE c.n IN(2,3);
INSERT INTO vec_bolsa_llamamientos.aviso_externo_resultado(
 productor_ref,evento_ref,version,huella_sha256,recibo_externo_ref,estado,registrada_en)
SELECT o.productor_ref,o.evento_ref,CASE WHEN c.n=3 THEN 2 ELSE 1 END,o.huella_sha256,a.recibo_externo_ref,c.estado,
 c.emitido+CASE WHEN c.n=8 THEN interval '30 seconds' ELSE interval '20 seconds' END
FROM pg_temp.b66_casos c JOIN vec_bolsa_llamamientos.aviso_externo_outbox o ON o.llamamiento_ref=c.llamamiento
JOIN vec_bolsa_llamamientos.aviso_externo_aceptacion a USING(productor_ref,evento_ref)
WHERE c.n IN(3,4,5,8,26);
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $contactos_b62$ DECLARE fila record; BEGIN
 FOR fila IN SELECT o.productor_ref,o.evento_ref FROM vec_bolsa_llamamientos.aviso_externo_outbox o
  JOIN pg_temp.b66_casos c ON c.llamamiento=o.llamamiento_ref WHERE c.n IN(3,4,5,8,26)
 LOOP PERFORM vec_bolsa_llamamientos.proyectar_contacto_aviso_externo_v1(fila.productor_ref,fila.evento_ref); END LOOP;
END $contactos_b62$;
RESET ROLE;
INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
 contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref)
SELECT 'contacto:'||h.huella,p.bolsa_ref,p.participacion_ref,c.llamamiento,'correo',c.emitido,
 'per_AAAAAAAAAAAAAAAAAAAAAA','enviado','aviso:legado:b66','b66:clave:7:correo:1','recibo:contacto:'||h.huella
FROM pg_temp.b66_casos c CROSS JOIN pg_temp.b66_propia p CROSS JOIN LATERAL (
 SELECT encode(sha256(convert_to(p.bolsa_ref||chr(31)||'b66:clave:7'||chr(31)||p.participacion_ref,'UTF8')),'hex') AS huella
) h WHERE c.n=7;
CREATE FUNCTION pg_temp.b66_consultar(p_tipo text,p_corte timestamptz,p_pagina integer DEFAULT 1,p_ajeno boolean DEFAULT false)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE f record; c jsonb; d jsonb; x jsonb; candidato text;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b66_propia;
 candidato:=CASE WHEN p_ajeno THEN 'can_ZZZZZZZZZZZZZZZZZZZZZZ' ELSE f.candidato_ref END;
 c:=jsonb_build_object('efecto_ref','mi-bolsa:'||candidato,'huella_efecto_sha256',repeat('a',64),
  'operacion','bolsa.historial_propio.consultar','audiencia_consumo','vec.bolsa.mi-bolsa.historial.v1');
 d:=jsonb_build_object('recurso_ref','mi-bolsa:'||candidato,'contexto_recurso_huella_sha256',repeat('a',64),
  'accion','bolsa.historial_propio.consultar','modulo_id','bolsa','tipo_recurso','participaciones_candidato',
  'finalidad','consulta_historial_propio','campos_permitidos',jsonb_build_array('contratos_propios','llamamientos_propios','renuncias_propias'),'obligaciones','[]'::jsonb);
 x:=jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',f.candidato_ref)));
 IF p_tipo='historial' THEN
  RETURN vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(candidato,p_corte,p_pagina,
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,convert_to(x::text,'UTF8'),1,1,''::bytea,''::bytea,''::bytea,''::bytea);
 END IF;
 RETURN vec_bolsa_llamamientos.consultar_mi_bolsa_v1(candidato,p_corte,
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),''::bytea,convert_to(x::text,'UTF8'),1,1,''::bytea,''::bytea,''::bytea,''::bytea);
END $f$;
ALTER FUNCTION pg_temp.b66_consultar(text,timestamptz,integer,boolean) OWNER TO vec_bolsa_llamamientos_propietario;
DO $cortes$
DECLARE c record; f record; corte timestamptz; j jsonb; fila jsonb; valor text; n integer; tipos text[]:=ARRAY['mi_bolsa','historial']; tipo text; delta integer;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b66_propia;
 FOR c IN SELECT a.* FROM pg_temp.b66_casos a WHERE a.n<=8 ORDER BY a.n LOOP
  FOREACH delta IN ARRAY ARRAY[10,31] LOOP
   corte:=c.emitido+delta*interval '1 second';
   valor:=CASE WHEN c.n=7 THEN 'enviado' WHEN c.n=8 AND delta=10 THEN 'enviado'
    WHEN delta=10 OR c.n IN(1,2,6) THEN 'aviso_pendiente'
    WHEN c.n IN(3,8) THEN 'enviado' ELSE 'no_enviado' END;
   FOREACH tipo IN ARRAY tipos LOOP
    j:=pg_temp.b66_consultar(tipo,corte);
    IF tipo='mi_bolsa' THEN
     SELECT e INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') e WHERE e->>'bolsa'=f.bolsa_ref;
     IF fila#>>'{ultimo_llamamiento,resultado}' IS DISTINCT FROM valor
     THEN RAISE EXCEPTION 'B66: Mi Bolsa caso % delta %',c.n,delta; END IF;
    ELSE
     SELECT count(*),min(e->>'resultado') INTO n,valor FROM jsonb_array_elements(j->'items') e
      WHERE e->>'clase'='llamamiento' AND e->>'ocurrido_en'=to_char(c.emitido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
     IF (c.n=8 AND delta=10 AND n<>0) OR (NOT(c.n=8 AND delta=10) AND (n<>1 OR valor IS DISTINCT FROM
       CASE WHEN c.n=7 THEN 'enviado' WHEN delta=10 OR c.n IN(1,2,6) THEN 'aviso_pendiente'
        WHEN c.n IN(3,8) THEN 'enviado' ELSE 'no_enviado' END))
     THEN RAISE EXCEPTION 'B66: historial duplicado/corte incorrecto caso % delta %',c.n,delta; END IF;
    END IF;
   END LOOP;
  END LOOP;
 END LOOP;
 -- Ni la emisión futura ni un outbox aún no registrado son visibles.
 j:=pg_temp.b66_consultar('historial','2040-01-01T00:00:59Z');
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(j->'items') e WHERE (e->>'ocurrido_en')::timestamptz>='2040-01-01T00:01:00Z')
 THEN RAISE EXCEPTION 'B66: emisión futura visible'; END IF;
 j:=pg_temp.b66_consultar('mi_bolsa','2040-01-01T00:08:26Z');
 SELECT e INTO STRICT fila FROM jsonb_array_elements(j->'participaciones') e WHERE e->>'bolsa'=f.bolsa_ref;
 IF fila#>>'{ultimo_llamamiento,resultado}' IS DISTINCT FROM 'aviso_pendiente'
 THEN RAISE EXCEPTION 'B66: outbox sin terminal no aparece pendiente'; END IF;
END $cortes$;
DO $paginacion$
DECLARE uno jsonb; dos jsonb; antes jsonb; n integer; denegadas integer:=0;
BEGIN
 uno:=pg_temp.b66_consultar('historial','2040-01-01T00:26:31Z',1);
 dos:=pg_temp.b66_consultar('historial','2040-01-01T00:26:31Z',2);
 IF uno->>'hay_mas' IS DISTINCT FROM 'true' OR jsonb_array_length(uno->'items')<>20
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(uno->'items') a JOIN jsonb_array_elements(dos->'items') b ON a=b)
    OR uno IS DISTINCT FROM pg_temp.b66_consultar('historial','2040-01-01T00:26:31Z',1)
 THEN RAISE EXCEPTION 'B66: páginas inestables o duplicadas'; END IF;
 -- Cambia el resultado del último aviso pero no la fecha ni su posición.
 antes:=pg_temp.b66_consultar('historial','2040-01-01T00:26:10Z',1);
 IF (SELECT jsonb_agg(e-'resultado' ORDER BY ordinal) FROM jsonb_array_elements(antes->'items') WITH ORDINALITY AS a(e,ordinal))
 IS DISTINCT FROM (SELECT jsonb_agg(e-'resultado' ORDER BY ordinal) FROM jsonb_array_elements(uno->'items') WITH ORDINALITY AS a(e,ordinal))
 THEN RAISE EXCEPTION 'B66: resultado altera orden de página'; END IF;
 SELECT count(*) INTO n FROM pg_temp.b66_consumos;
 BEGIN PERFORM pg_temp.b66_consultar('mi_bolsa','2040-01-01T00:26:31Z',1,true);
 EXCEPTION WHEN insufficient_privilege THEN denegadas:=denegadas+1; END;
 BEGIN PERFORM pg_temp.b66_consultar('historial','2040-01-01T00:26:31Z',1,true);
 EXCEPTION WHEN insufficient_privilege THEN denegadas:=denegadas+1; END;
 IF denegadas<>2 OR (SELECT count(*) FROM pg_temp.b66_consumos)<>n
    OR NOT EXISTS(SELECT 1 FROM pg_temp.b66_consumos WHERE tipo='mi_bolsa')
    OR NOT EXISTS(SELECT 1 FROM pg_temp.b66_consumos WHERE tipo='historial')
 THEN RAISE EXCEPTION 'B66: guardas Own/consumo incorrectos'; END IF;
END $paginacion$;
SELECT 'B66-PROYECCION-OK: pendiente, incierto, terminales, ACK, corte, legado, páginas y ajeno' AS resultado;
ROLLBACK;
