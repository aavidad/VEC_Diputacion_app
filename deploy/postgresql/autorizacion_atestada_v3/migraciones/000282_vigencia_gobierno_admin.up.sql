\set ON_ERROR_STOP on
-- AD282: plazo de claves y confianza del gobierno ADMIN nominal en desarrollo.
-- Parte de las dos funciones de efecto instaladas; preserva recibos y replay.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pre$
DECLARE f regprocedure; actual text; esperado text;
BEGIN
 IF current_setting('server_version_num')::integer < 180000
 OR to_regclass('vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'AD282: PARO clave=dependencias actual=ausentes esperado=PG18_AD188_AD198' USING ERRCODE='55000'; END IF;
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'9104549c5d240f06ad9b343559e15e9d46a7a7de736e682a57d47ecb57aa1d1b'),  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'aadaaa7f9a1029564e4c02587394a175ab4a5dd45c67526d50ea034f8b6ef973')
 ) AS v(f,h) LOOP
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
  IF actual IS DISTINCT FROM esperado THEN
   RAISE EXCEPTION 'AD282: PARO clave=preimagen actual=% esperado=%',coalesce(actual,'ausente'),esperado USING ERRCODE='55000';
  END IF;
  IF (SELECT p.proowner::regrole::text='vec_autorizacion_atestada_v3_propietario'
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog']
      AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM true THEN
   RAISE EXCEPTION 'AD282: PARO clave=seguridad_funcion actual=distinta esperado=propietario_definer_acl' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $pre$;
LOCK TABLE vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1,
 vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 IN SHARE MODE;
DO $rebuild$
DECLARE f regprocedure; cuerpo text; marca text; nuevo text;
BEGIN
 FOR f IN SELECT v.f FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure),  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure)
 ) AS v(f) LOOP
  cuerpo:=pg_get_functiondef(f);
  marca:=$fecha_antigua$OR (g->>'publicada_en')::timestamptz<>date_trunc('day',ahora) OR (g->>'expira_en')::timestamptz<>(g->>'publicada_en')::timestamptz+interval '1 day'$fecha_antigua$;
  nuevo:=$fecha_nueva$OR c.entorno IS DISTINCT FROM 'desarrollo'
 OR (g->>'publicada_en')::timestamptz>ahora
 OR date_trunc('day',(g->>'publicada_en')::timestamptz)<>date_trunc('day',ahora)
 OR (g->>'expira_en')::timestamptz<(g->>'publicada_en')::timestamptz+interval '1 hour'
 OR (g->>'expira_en')::timestamptz>(g->>'publicada_en')::timestamptz+interval '400 days'
 OR (g->>'expira_en')::timestamptz>root.valida_hasta$fecha_nueva$;
  IF length(cuerpo)-length(replace(cuerpo,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'AD282: PARO clave=marca_fecha actual=no_unica esperado=una' USING ERRCODE='55000';
  END IF;
  cuerpo:=replace(cuerpo,marca,nuevo);
  marca:=$clave_antigua$OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora OR (k->>'valida_hasta')::timestamptz>root.valida_hasta$clave_antigua$;
  nuevo:=$clave_nueva$OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora
  OR (k->>'valida_hasta')::timestamptz>root.valida_hasta
  OR (k->>'valida_hasta')::timestamptz>(g->>'expira_en')::timestamptz
  OR (k->>'valida_hasta')::timestamptz<(p->>'preparado_en')::timestamptz+interval '1 hour'
  OR (k->>'valida_hasta')::timestamptz>(p->>'preparado_en')::timestamptz+interval '400 days'$clave_nueva$;
  IF length(cuerpo)-length(replace(cuerpo,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'AD282: PARO clave=marca_clave actual=no_unica esperado=una' USING ERRCODE='55000';
  END IF;
  cuerpo:=replace(cuerpo,marca,nuevo);
  marca:=$ruta_antigua$SET search_path TO 'pg_catalog'$ruta_antigua$;
  nuevo:=$ruta_nueva$SET search_path TO 'pg_catalog', 'pg_temp'$ruta_nueva$;
  IF length(cuerpo)-length(replace(cuerpo,marca,''))<>length(marca) THEN
   RAISE EXCEPTION 'AD282: PARO clave=search_path actual=no_unico esperado=uno' USING ERRCODE='55000';
  END IF;
  EXECUTE replace(cuerpo,marca,nuevo);
 END LOOP;
END $rebuild$;
DO $post$
DECLARE f regprocedure; actual text; esperado text;
BEGIN
 FOR f, esperado IN SELECT v.f, v.h FROM (VALUES
  ('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure,'7a2c53d075e5c45a7455f2e8175498b13e8c82e120e08fcd9cdc79194d3c73a3'),  ('vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)'::regprocedure,'8330b0ad391ce73d62fc3f2781b2ada7094d1e61d4b340afca6ba68dd769639f')
 ) AS v(f,h) LOOP
  SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
  IF actual IS DISTINCT FROM esperado THEN
   RAISE EXCEPTION 'AD282: PARO clave=postimagen actual=% esperado=%',coalesce(actual,'ausente'),esperado USING ERRCODE='55000';
  END IF;
  IF (SELECT p.proowner::regrole::text='vec_autorizacion_atestada_v3_propietario'
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']
      AND p.proacl=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM true
  THEN
   RAISE EXCEPTION 'AD282: PARO clave=seguridad_final actual=distinta esperado=propietario_definer_acl' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $post$;
COMMIT;
