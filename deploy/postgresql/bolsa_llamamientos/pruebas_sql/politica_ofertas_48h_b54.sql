-- Se ejecuta en PostgreSQL 18 efímero con B47 y B51 instaladas y el doble
-- delimitado AD3 de probar_consulta_politica_ofertas_pg18.sh.
DO $acl$
BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('public',
      'vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_calculador_politica',
      'vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'B54 ACL ampliada'; END IF;
END $acl$;
DO $vector$
DECLARE h text;
BEGIN
 h:=encode(sha256(convert_to(array_to_string(ARRAY[
  'bolsa:of:1','2026-03-28T12:17:13.123456Z','2026-03-30T12:17:13.123456Z',
  'politica-ofertas:bolsa:of:1:2',repeat('a',64),'horas_naturales','48',
  'continuo_utc','18087','2026-03-30','2','calendario:utc-continuo:v1',
  '2026-03-28T12:17:13.123456Z','2026-03-30T12:17:13.123456Z'
 ],chr(31)),'UTF8')),'hex');
 IF h<>'fa8852b8fc417012ce19880e4525e2d57247ef78ef33f8d0da4568b58e735255'
 THEN RAISE EXCEPTION 'B54 vector Go/SQL divergente'; END IF;
END $vector$;
DO $prueba$
DECLARE
 p jsonb:='{"plazo":{"unidad":"horas_naturales","cantidad":48,"computo":"continuo_utc","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}';
 cap bytea:=convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 dec bytea:=convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.publicar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gobierno_politica_ofertas_bolsa","recurso_ref":"bolsa:of:1"}','UTF8');
 datos jsonb:='{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}';
 publicada timestamptz:=clock_timestamp(); vence timestamptz; forjado timestamptz;
 ultima text; plazo jsonb; falso jsonb; capof bytea:=convert_to('{"efecto_ref":"bolsa:of:1"}','UTF8');
 decof bytea; h text; contexto text; v jsonb; r record; q record; i integer;
 ap timestamptz; fin timestamptz; ph jsonb;
BEGIN
 FOR i IN 1..2 LOOP
  BEGIN
   PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
    'bolsa:of:1',1,jsonb_set(jsonb_set(jsonb_set(p,'{plazo,unidad}',to_jsonb(
      CASE WHEN i=1 THEN 'dias_habiles' ELSE 'horas_naturales' END)),
      '{plazo,cantidad}',to_jsonb(CASE WHEN i=1 THEN 2 ELSE 48 END)),
      '{plazo,computo}','null'::jsonb),
    'per_actoractoractoractoractor','clave-null-computo-'||i::text,
    'recibo:politica-ofertas:'||repeat((i+2)::text,64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'B54 aceptó cómputo nulo';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
  'bolsa:of:1',1,p,'per_actoractoractoractoractor','clave-horas-0001',
  'recibo:politica-ofertas:'||repeat('d',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r.reutilizada OR r.politica->>'version'<>'2' THEN RAISE EXCEPTION 'B54 política v2 no publicada'; END IF;
 SELECT * INTO q FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
  'bolsa:of:1',1,p,'per_actoractoractoractoractor','clave-horas-0001',
  'recibo:politica-ofertas:'||repeat('d',64),cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF NOT q.reutilizada OR q.politica IS DISTINCT FROM r.politica THEN RAISE EXCEPTION 'B54 replay política cambió'; END IF;
 IF (SELECT o.plazo->>'politica_version' FROM vec_bolsa_llamamientos.oferta_publicada o
     WHERE o.oferta_ref='oferta:'||repeat('1',64))<>'1' THEN RAISE EXCEPTION 'B54 alteró recibo B47'; END IF;
 v:=r.politica; vence:=publicada+interval '48 hours';
 ultima:=((vence-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text;
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:bolsa:of:1:2','huella_catalogo',v->>'huella_sha256',
  'unidad','horas_naturales','cantidad',48,'computo','continuo_utc','municipio_sede','18087',
  'ultimo_dia',ultima,'ejemplo',true,'politica_version',2,
  'calendarios',jsonb_build_array('calendario:utc-continuo:v1'),
  'apertura_en',to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vence_en',to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 h:=encode(sha256(convert_to(array_to_string(ARRAY['bolsa:of:1',
  to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  plazo->>'regla_ref',plazo->>'huella_catalogo','horas_naturales','48','continuo_utc','18087',
  ultima,'2','calendario:utc-continuo:v1',plazo->>'apertura_en',plazo->>'vence_en'],chr(31)),'UTF8')),'hex');
 contexto:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 decof:=convert_to((jsonb_build_object('principal_id','per_actoractoractoractoractor','accion','llamamiento.emitir.v1',
  'modulo_id','bolsa','tipo_recurso','bolsa_constituida','finalidad','gestion_llamamientos_bolsa',
  'recurso_ref','bolsa:of:1','contexto_recurso_huella_sha256',contexto))::text,'UTF8');
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_oferta_v2('oferta:'||repeat('6',64),
  'recibo:oferta:'||repeat('6',64),'bolsa:of:1','per_actoractoractoractoractor','clave-horas-oferta-0001',
  datos,plazo,publicada,vence,capof,decof,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF r.reutilizada OR r.oferta->'plazo'->>'apertura_en'<>plazo->>'apertura_en' OR
    r.oferta->'plazo'->>'vence_en'<>plazo->>'vence_en' THEN RAISE EXCEPTION 'B54 oferta no conserva instante'; END IF;
 SELECT * INTO q FROM vec_bolsa_llamamientos.publicar_oferta_v2('oferta:'||repeat('6',64),
  'recibo:oferta:'||repeat('6',64),'bolsa:of:1','per_actoractoractoractoractor','clave-horas-oferta-0001',
  datos,plazo,publicada,vence,capof,decof,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
 IF NOT q.reutilizada OR q.oferta->>'recibo_ref' IS DISTINCT FROM r.oferta->>'recibo_ref' OR
    (SELECT count(*) FROM vec_bolsa_llamamientos.oferta_publicada WHERE clave_idempotencia='clave-horas-oferta-0001')<>1
 THEN RAISE EXCEPTION 'B54 replay oferta duplicado'; END IF;
 -- Incluso con una huella V3 recalculada para un plazo internamente coherente,
 -- el trigger exige la cantidad publicada: +90 días no caben en 48 horas.
 forjado:=vence+interval '90 days';
 falso:=plazo||jsonb_build_object('vence_en',to_char(forjado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'ultimo_dia',((forjado-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text);
 h:=encode(sha256(convert_to(array_to_string(ARRAY['bolsa:of:1',
  to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(forjado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  falso->>'regla_ref',falso->>'huella_catalogo','horas_naturales','48','continuo_utc','18087',
  falso->>'ultimo_dia','2','calendario:utc-continuo:v1',falso->>'apertura_en',falso->>'vence_en'],chr(31)),'UTF8')),'hex');
 contexto:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 decof:=convert_to((jsonb_build_object('principal_id','per_actoractoractoractoractor','accion','llamamiento.emitir.v1',
  'modulo_id','bolsa','tipo_recurso','bolsa_constituida','finalidad','gestion_llamamientos_bolsa',
  'recurso_ref','bolsa:of:1','contexto_recurso_huella_sha256',contexto))::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v2('oferta:'||repeat('7',64),
   'recibo:oferta:'||repeat('7',64),'bolsa:of:1','per_actoractoractoractoractor','clave-horas-forjada-0002',
   datos,falso,publicada,forjado,capof,decof,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa');
  RAISE EXCEPTION 'B54 aceptó +90 días forjados';
 EXCEPTION WHEN SQLSTATE 'VBP02' THEN NULL; END;
 -- Ambos cambios de horario tienen 48 horas UTC aunque la hora local varíe.
 FOR i IN 1..2 LOOP
  ap:=CASE WHEN i=1 THEN '2026-03-28T12:17:13.123456Z'::timestamptz
                       ELSE '2026-10-24T12:17:13.123456Z'::timestamptz END;
  fin:=ap+interval '48 hours';
  ph:=plazo||jsonb_build_object('apertura_en',to_char(ap AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vence_en',to_char(fin AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'ultimo_dia',((fin-interval '1 microsecond') AT TIME ZONE 'Europe/Madrid')::date::text);
  INSERT INTO vec_bolsa_llamamientos.oferta_publicada(
   oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
   publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
  VALUES('oferta:'||repeat((i+7)::text,64),'recibo:oferta:'||repeat((i+7)::text,64),
   'bolsa:of:1','per_actoractoractoractoractor','clave-dst-'||i::text||'-0001',datos,ph,
   ap,fin,repeat('a',64),'decision:dst:'||i::text);
  IF fin-ap<>interval '48 hours' OR
     (i=1 AND extract(hour FROM ap AT TIME ZONE 'Europe/Madrid')<>13) OR
     (i=1 AND extract(hour FROM fin AT TIME ZONE 'Europe/Madrid')<>14) OR
     (i=2 AND extract(hour FROM ap AT TIME ZONE 'Europe/Madrid')<>14) OR
     (i=2 AND extract(hour FROM fin AT TIME ZONE 'Europe/Madrid')<>13)
  THEN RAISE EXCEPTION 'B54 DST cálculo incorrecto'; END IF;
 END LOOP;
END $prueba$;
