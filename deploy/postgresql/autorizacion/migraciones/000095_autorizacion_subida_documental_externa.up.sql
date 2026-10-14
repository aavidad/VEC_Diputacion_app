\set ON_ERROR_STOP on
-- AUT95: extensión prospectiva de M4, misma cuenta, perfil y fuente externa.
-- Las tres acciones publicadas conservan su significado y sus recibos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000095',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  to_regrole('vec_documentos_subida_datos') IS NULL OR
  to_regprocedure('vec_autorizacion.acciones_meritos_cartera_v2()') IS NOT NULL OR
  to_regprocedure('vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)') IS NULL OR
  to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)') IS NULL OR
  to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)') IS NULL OR
  to_regprocedure('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)') IS NULL
 THEN RAISE EXCEPTION 'AUT95: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion.acciones_meritos_cartera_v2() RETURNS text[]
LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT ARRAY[
  'meritos.cartera.comprobante.descargar_propio',
  'meritos.cartera.comprobante.subir_propio',
  'meritos.cartera.consultar_propia',
  'meritos.cartera.retirar_propio',
  'meritos.hecho.declarar']::text[]
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acciones_meritos_cartera_v2() FROM PUBLIC;

-- La fuente M4 existente admite los tres estados exactos 2/3/5.
DO $fuente$
DECLARE f oid:='vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)'::regprocedure;
 s text; a text;
BEGIN
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM
  '3843cf4b256a89f045c6e98b4b33c9a9a769b6b09b6bc6bfa232f6c0dc10f297'
 THEN RAISE EXCEPTION 'AUT95: fuente M4 instalada divergente' USING ERRCODE='55000'; END IF;
 a:=$m$ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[])$m$;
 IF (length(s)-length(replace(s,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'AUT95: marca fuente M4 divergente' USING ERRCODE='55000'; END IF;
 s:=replace(s,a,$n$ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[]
  AND (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
   FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   vec_autorizacion.acciones_meritos_cartera_v2())$n$);
 EXECUTE s;
END $fuente$;

-- M4 conserva sus casos 2/3 y consulta de ocho campos. El rol nuevo permite
-- nueve campos solo si orden y concesión coinciden de forma exacta.
DO $cartera$
DECLARE f oid:='vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)'::regprocedure;
 s text; a text;
BEGIN
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM
  '4eeffb4bb5a1ca7977d8392400d85b1ef25dcb4dd0db2e56226e2cc4a4dc26ea'
 THEN RAISE EXCEPTION 'AUT95: cotejo M4 instalado divergente' USING ERRCODE='55000'; END IF;
 a:=$m$WHEN 'meritos.cartera.consultar_propia' THEN
   '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb$m$;
 IF (length(s)-length(replace(s,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'AUT95: campos M4 divergentes' USING ERRCODE='55000'; END IF;
 s:=replace(s,a,$n$WHEN 'meritos.cartera.consultar_propia' THEN
   CASE WHEN p_orden->'campos_permitidos' IN (
    '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb,
    '["denominacion","disponibilidad","evidencias","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb)
   THEN p_orden->'campos_permitidos' ELSE NULL::jsonb END$n$);
 a:=$m$ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[])$m$;
 IF (length(s)-length(replace(s,a,'')))/length(a)<>1
 THEN RAISE EXCEPTION 'AUT95: rol cotejo M4 divergente' USING ERRCODE='55000'; END IF;
 s:=replace(s,a,$n$ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[]
  AND (SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
   FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
   vec_autorizacion.acciones_meritos_cartera_v2())$n$);
 EXECUTE s;
END $cartera$;

-- El registrador V3 M4 conserva las dos operaciones anteriores y acepta
-- subir/descargar solo con el rol de cinco concesiones gobernado.
DO $decision_pre$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)'::regprocedure),'UTF8')),'hex')
  IS DISTINCT FROM 'c2e91ee495253426458f8b9c6923d0ac10d2e1231566dfdf6cc59b4718382499'
 THEN RAISE EXCEPTION 'AUT95: decisión M4 instalada divergente' USING ERRCODE='55000'; END IF;
END $decision_pre$;
CREATE OR REPLACE FUNCTION vec_autorizacion.decision_meritos_externa_exacta_v1(d jsonb,m jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE v jsonb; accion text; campos jsonb; obligaciones jsonb; lista text[]; rol_documento jsonb;
BEGIN
 v:=d->'vinculo_autenticacion_actor'; accion:=d->>'accion';
 campos:=CASE accion
  WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
  WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
  WHEN 'meritos.cartera.comprobante.descargar_propio' THEN '["contenido"]'::jsonb END;
 obligaciones:=CASE WHEN accion IN ('meritos.cartera.comprobante.subir_propio',
  'meritos.cartera.comprobante.descargar_propio') THEN '[]'::jsonb ELSE '["auditar"]'::jsonb END;
 IF current_user<>'vec_autorizacion_propietario' OR
  vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE OR
  campos IS NULL OR d->>'modulo_id' IS DISTINCT FROM 'meritos' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
  d->>'finalidad' IS DISTINCT FROM (CASE
   WHEN accion='meritos.hecho.declarar' THEN 'declaracion_hecho_propio'
   WHEN accion='meritos.cartera.retirar_propio' THEN 'gestion_cartera_propia'
   ELSE 'custodiar_evidencia_meritos' END) OR
  (accion IN ('meritos.cartera.comprobante.subir_propio',
   'meritos.cartera.comprobante.descargar_propio') AND
   d->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$') OR
  v->>'superficie' IS DISTINCT FROM 'externa_personal' OR
  v->>'cuenta_privilegiada' IS DISTINCT FROM 'false' OR
  d->>'principal_id' IS DISTINCT FROM v->>'principal_id' OR
  d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref' OR
  vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
   v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE OR
  m#>>'{referencia,catalogo_id}' IS DISTINCT FROM
   (CASE WHEN accion='meritos.hecho.declarar' THEN 'motivos_meritos_declaracion'
    ELSE 'motivos_autorizacion_meritos_cartera' END)
 THEN RETURN false; END IF;
 SELECT (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
  FROM jsonb_array_elements(r.documento->'concesiones') x),r.documento
 INTO lista,rol_documento FROM vec_autorizacion.version_rol r
 WHERE r.version_rol_ref=d->>'version_rol_ref' AND r.rol_id='persona_meritos_cartera';
 IF lista IS NULL OR NOT (
  (accion='meritos.hecho.declarar' AND lista IN (
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[],
   vec_autorizacion.acciones_meritos_cartera_v2())) OR
  (accion='meritos.cartera.retirar_propio' AND lista IN (
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[],
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[],
   vec_autorizacion.acciones_meritos_cartera_v2())) OR
  (accion IN ('meritos.cartera.comprobante.subir_propio',
   'meritos.cartera.comprobante.descargar_propio') AND
   lista IS NOT DISTINCT FROM vec_autorizacion.acciones_meritos_cartera_v2() AND
   vec_autorizacion.rol_meritos_cartera_cinco_v2(rol_documento) IS TRUE))
 THEN RETURN false; END IF;
 IF d->>'concedida'='true' THEN
  RETURN d->'campos_permitidos' IS NOT DISTINCT FROM campos AND
   d->'obligaciones' IS NOT DISTINCT FROM obligaciones;
 END IF;
 RETURN d->>'concedida'='false' AND d->'campos_permitidos' <@ campos AND
  d->'obligaciones' <@ obligaciones;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $f$;

-- El publicador de asignaciones acepta la nueva versión M4 si cada una de
-- sus cinco concesiones tiene exactamente la semántica publicada.
CREATE FUNCTION vec_autorizacion.rol_meritos_cartera_cinco_v2(p_rol jsonb) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb; accion text; n integer:=0;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR
  p_rol->>'rol_id' IS DISTINCT FROM 'persona_meritos_cartera' OR
  jsonb_typeof(p_rol->'concesiones') IS DISTINCT FROM 'array' OR
  jsonb_array_length(p_rol->'concesiones')<>5
 THEN RETURN false; END IF;
 FOR c IN SELECT value FROM jsonb_array_elements(p_rol->'concesiones') LOOP
  accion:=c->>'accion'; n:=n+1;
  IF jsonb_typeof(c)<>'object' OR
   (SELECT count(*) FROM jsonb_object_keys(c))<>7 OR
   c->>'modulo_id' IS DISTINCT FROM 'meritos' OR
   c->>'garantia_minima' IS DISTINCT FROM 'alto' OR
   (accion=ANY(vec_autorizacion.acciones_meritos_cartera_v2())) IS NOT TRUE OR
   c->>'tipo_recurso' IS DISTINCT FROM
    (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera_personal' ELSE 'hecho' END) OR
   c->'finalidades' IS DISTINCT FROM
    (CASE WHEN accion='meritos.hecho.declarar' THEN '["declaracion_hecho_propio"]'::jsonb
     WHEN accion IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
      THEN '["custodiar_evidencia_meritos"]'::jsonb ELSE '["gestion_cartera_propia"]'::jsonb END) OR
   c->'campos_permitidos' IS DISTINCT FROM
    (CASE accion
     WHEN 'meritos.cartera.consultar_propia' THEN
      '["denominacion","disponibilidad","evidencias","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb
     WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
     WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
     WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
     ELSE '["contenido"]'::jsonb END) OR
   c->'obligaciones' IS DISTINCT FROM
    (CASE WHEN accion IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
      THEN '[]'::jsonb ELSE '["auditar"]'::jsonb END)
  THEN RETURN false; END IF;
 END LOOP;
 RETURN n=5 AND ((SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
  FROM jsonb_array_elements(p_rol->'concesiones') x)
   IS NOT DISTINCT FROM vec_autorizacion.acciones_meritos_cartera_v2());
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.rol_meritos_cartera_cinco_v2(jsonb) FROM PUBLIC;

DO $asignacion$
DECLARE f oid:='vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)'::regprocedure;
 s text; inicio integer; fin integer; fragmento text;
BEGIN
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM
  'ed39739ff168213b1cfd896491fcbf1ea5f8dc02d93d81d56c7843bc0b2d5eb9'
 THEN RAISE EXCEPTION 'AUT95: publicador M4 instalado divergente' USING ERRCODE='55000'; END IF;
 inicio:=strpos(s,$m$AND r.documento->'concesiones' IS DISTINCT FROM '[{"accion":"meritos.cartera.consultar_propia"$m$);
 IF inicio=0 THEN RAISE EXCEPTION 'AUT95: marca asignación M4 ausente' USING ERRCODE='55000'; END IF;
 fin:=inicio+strpos(substr(s,inicio),'::pg_catalog.jsonb)')+length('::pg_catalog.jsonb)')-2;
 IF fin<=inicio THEN RAISE EXCEPTION 'AUT95: cierre asignación M4 ausente' USING ERRCODE='55000'; END IF;
 fragmento:=substr(s,inicio,fin-inicio+1);
 IF (length(s)-length(replace(s,fragmento,'')))/length(fragmento)<>1
 THEN RAISE EXCEPTION 'AUT95: guarda de asignación esperada' USING ERRCODE='55000'; END IF;
 s:=replace(s,fragmento,left(fragmento,length(fragmento)-1)||
  E' AND vec_autorizacion.rol_meritos_cartera_cinco_v2(r.documento) IS NOT TRUE)');
 EXECUTE s;
END $asignacion$;

-- El núcleo AD273 ejecuta las decisiones del acto M4 con un LOGIN de datos
-- exclusivo. Los dos helpers M4 conservan sus cuentas anteriores y permiten
-- esa cuenta solo con un único grupo heredado y sin capacidad de SET ROLE.
DO $registro_datos$
DECLARE f oid; s text; marca text; guardia text;
BEGIN
 guardia:=$x$  OR (session_user='vec_externo_documentos_datos'
   AND pg_has_role(session_user,'vec_documentos_subida_datos','MEMBER')
   AND EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
    AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
   AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)$x$;
 f:='vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex')<>'4d87ca659d4fcc43e6f224f702d38e6ed1ebdedc4780ba19e5a90d681ffc2c9d'
 THEN RAISE EXCEPTION 'AUT95: registrador M4 divergente' USING ERRCODE='55000'; END IF;
 marca:=$x$   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)))$x$;
 IF (length(s)-length(replace(s,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT95: marca registrador M4' USING ERRCODE='55000'; END IF;
 s:=replace(s,marca,left(marca,length(marca)-1)||guardia||')');
 EXECUTE s;
 f:='vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)'::regprocedure;
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex')<>'d43b6897de107cfea37c23298235f63b26e51774b3eae03b974f0e1ad8e3834e'
 THEN RAISE EXCEPTION 'AUT95: revalidador M4 divergente' USING ERRCODE='55000'; END IF;
 marca:=$x$   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)))$x$;
 IF (length(s)-length(replace(s,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT95: marca revalidador M4' USING ERRCODE='55000'; END IF;
 s:=replace(s,marca,left(marca,length(marca)-1)||guardia||')');
 EXECUTE s;
END $registro_datos$;

-- La comprobación de la sesión activa M4 se usa tanto en el asiento como
-- en la revalidación posterior del consumo; el LOGIN de datos queda nominal.
DO $sesion_datos$
DECLARE f oid:='vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 s text; marca text;
BEGIN
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex')<>'78834a038bae17e5c1c632d4840ee58bb026bebda7cb96202af68f048e203b2b'
 THEN RAISE EXCEPTION 'AUT95: sesión M4 divergente' USING ERRCODE='55000'; END IF;
 marca:=$x$    OR (g.rolname='vec_meritos_ejecutor'
     AND l.rolname='vec_externo_meritos_operacion'))$x$;
 IF (length(s)-length(replace(s,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT95: marca sesión M4' USING ERRCODE='55000'; END IF;
 s:=replace(s,marca,$x$    OR (g.rolname='vec_meritos_ejecutor'
     AND l.rolname='vec_externo_meritos_operacion')
    OR (g.rolname='vec_documentos_subida_datos'
     AND l.rolname='vec_externo_documentos_datos'))$x$);
 EXECUTE s;
END $sesion_datos$;
COMMIT;
