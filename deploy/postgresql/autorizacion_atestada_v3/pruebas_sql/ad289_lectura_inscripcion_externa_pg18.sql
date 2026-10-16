\set ON_ERROR_STOP on
-- AD289/CA47: la lectura de inscripción del portal externo asienta su auditoría
-- con el contexto de registros_contexto_externo_v2; contextos no verificados, de
-- otra persona o presentes en los dos almacenes se deniegan; RRHH sigue igual.
-- Ejecutar SOLO en copia aislada PG18 desechable, con -v AD289_DISPOSABLE_CLONE=1.
-- Todo ocurre en una transacción que termina en ROLLBACK: los LOGIN, EXECUTE,
-- provisión y recibos sintéticos no quedan en la base.
-- Con -v AD289_REPRODUCIR=1 solo comprueba el defecto previo (P0002) y sale.
\if :{?AD289_DISPOSABLE_CLONE}
\else
\echo 'AD289: falta AD289_DISPOSABLE_CLONE; prueba omitida sin efectos'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='120s';

-- Emisores del contrato AD228 (copia desechable; se revierte).
DO $emisores$
DECLARE r record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('vec_bolsa_inscripciones_lector','vec_bolsa_llamamientos_lector_inscripciones'),
   ('vec_bolsa_inscripciones_rrhh_lector','vec_bolsa_llamamientos_lector_inscripciones_rrhh')) x(login,grupo)
 LOOP
  IF to_regrole(r.login) IS NULL THEN
   EXECUTE format('CREATE ROLE %I LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS',r.login);
  ELSE
   EXECUTE format('ALTER ROLE %I LOGIN',r.login);
  END IF;
  IF NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=r.login::regrole AND m.roleid=r.grupo::regrole) THEN
   EXECUTE format('GRANT %I TO %I WITH INHERIT TRUE, SET FALSE, ADMIN FALSE',r.grupo,r.login);
  END IF;
 END LOOP;
END $emisores$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_inscripciones_lector,vec_bolsa_inscripciones_rrhh_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(bytea,bytea,jsonb)
 TO vec_bolsa_inscripciones_lector,vec_bolsa_inscripciones_rrhh_lector;

CREATE TEMP TABLE ad289_material(caso text PRIMARY KEY,contexto bytea NOT NULL,vinculo bytea NOT NULL,orden jsonb NOT NULL);
GRANT SELECT ON ad289_material TO vec_bolsa_inscripciones_lector,vec_bolsa_inscripciones_rrhh_lector;
CREATE TEMP TABLE ad289_base(clase text PRIMARY KEY,contexto bytea NOT NULL,vinculo jsonb NOT NULL);

CREATE FUNCTION pg_temp.ad289_orden(p_caso text,p_contexto bytea,p_vinculo jsonb,p_canal text)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object(
  'intento_ref','lectura_'||substr(encode(sha256(convert_to('ad289:'||p_caso,'UTF8')),'hex'),1,32),
  'registro_contexto_ref',p_vinculo->>'registro_contexto_ref',
  'contexto_sha256',encode(sha256(p_contexto),'hex'),
  'procedencia_sha256',p_vinculo->>'manifiesto_procedencia_huella_sha256',
  'autenticacion_ref',p_vinculo->>'autenticacion_ref','sesion_ref',p_vinculo->>'sesion_ref',
  'autenticacion_sha256',p_vinculo->>'autenticacion_huella_sha256',
  'accion',CASE p_canal WHEN 'externa_personal' THEN 'bolsa.inscripcion.propias.listar' ELSE 'bolsa.inscripcion.rrhh.listar' END,
  'modulo_id','bolsa',
  'recurso_ref',CASE p_canal WHEN 'externa_personal' THEN 'inscripciones_propias_' ELSE 'inscripciones_rrhh_' END||repeat('a',64),
  'finalidad_ref',CASE p_canal WHEN 'externa_personal' THEN 'consulta_inscripcion_propia' ELSE 'consulta_inscripcion_rrhh' END,
  'resultado','obtenida','motivo_ref','inscripcion_lectura_correcta','proceso','vec-server',
  'canal',p_canal,'correlacion_ref','correlacion_ad289_'||p_caso,
  'lectura_revision_permisos','1','lectura_instantanea_sha256',repeat('b',64))
$f$;

-- Material histórico real de la copia: una sesión externa y otra interna cuyos
-- contextos y autenticaciones superan los cotejos instalados.
DO $base$
DECLARE r record; v jsonb; c jsonb; superficie text; n integer;
BEGIN
 FOREACH superficie IN ARRAY ARRAY['externa_personal','interna_corporativa'] LOOP
  n:=0;
  FOR r IN SELECT c.registro_contexto_ref,c.huella_sha256,c.manifiesto_procedencia_huella_sha256,
                  c.representacion_canonica,d.decision_canonica
           FROM vec_autorizacion.decision_concedida_contexto_actor_v3 d
           JOIN vec_contexto_actor_v1.registros_contexto c
             ON c.registro_contexto_ref=convert_from(d.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,registro_contexto_ref}'
           WHERE convert_from(d.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,superficie}'=superficie
           ORDER BY c.resuelto_en DESC LIMIT 100
  LOOP
   BEGIN
    v:=convert_from(r.decision_canonica,'UTF8')::jsonb->'vinculo_autenticacion_actor';
    c:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
    IF (c->>'resuelto_en')::timestamptz>=(v->>'sesion_revalidada_en')::timestamptz
       AND (c->>'resuelto_en')::timestamptz<(v->>'sesion_valida_hasta')::timestamptz
       AND vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(r.registro_contexto_ref,
             r.huella_sha256,r.manifiesto_procedencia_huella_sha256,r.representacion_canonica)
       AND vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(convert_to(v::text,'UTF8')) THEN
     INSERT INTO pg_temp.ad289_base VALUES(superficie,r.representacion_canonica,v);
     n:=1; EXIT;
    END IF;
   EXCEPTION WHEN others THEN CONTINUE;
   END;
  END LOOP;
  IF n<>1 THEN RAISE EXCEPTION 'AD289: sin material histórico apto para %',superficie; END IF;
 END LOOP;
 INSERT INTO pg_temp.ad289_material
 SELECT 'legado_externo',b.contexto,convert_to(b.vinculo::text,'UTF8'),
        pg_temp.ad289_orden('legado_externo',b.contexto,b.vinculo,'externa_personal')
 FROM pg_temp.ad289_base b WHERE b.clase='externa_personal'
 UNION ALL
 SELECT 'interno',b.contexto,convert_to(b.vinculo::text,'UTF8'),
        pg_temp.ad289_orden('interno',b.contexto,b.vinculo,'interna_corporativa')
 FROM pg_temp.ad289_base b WHERE b.clase='interna_corporativa';
END $base$;
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp() WHERE control;

\if :{?AD289_REPRODUCIR}
\else
-- 1. Recibo externo anterior a CA15 (almacén interno, cuenta sin población
--    segregada): se sigue asentando igual que hasta ahora.
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
DO $legado$
DECLARE x record;
BEGIN
 SELECT * INTO STRICT x FROM pg_temp.ad289_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='legado_externo';
 IF x.auditoria_ref !~ '^aud_v3_li_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD289: acuse legado inválido'; END IF;
END $legado$;
RESET SESSION AUTHORIZATION;
-- 2. RRHH (canal interno) sigue igual.
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_rrhh_lector;
DO $interno$
DECLARE x record;
BEGIN
 SELECT * INTO STRICT x FROM pg_temp.ad289_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='interno';
 IF x.auditoria_ref !~ '^aud_v3_li_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD289: acuse interno inválido'; END IF;
END $interno$;
RESET SESSION AUTHORIZATION;
\endif

-- Aspirante externo sintético: provisión CA15 publicada, cuenta y sesión
-- externa con la forma de las reales (filas insertadas aquí, sin aserción de
-- certificado) y contexto registrado por el resolutor oficial CA15.
CREATE ROLE prueba_ad289_candidato LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_candidato_externo TO prueba_ad289_candidato WITH INHERIT TRUE, SET FALSE;
CREATE TEMP TABLE ad289_actor(clase text PRIMARY KEY,cuenta_ref text NOT NULL,perfil_ref text NOT NULL,vinculo jsonb);
DO $actor$
DECLARE
 b jsonb; s jsonb; k text; comp jsonb; h text; t timestamptz; sem text;
 cta text:='cta_'||md5('ad289:cuenta'); prf text:='prf_'||md5('ad289:perfil'); per text:='per_'||md5('ad289:persona');
 ses text; aut text; ase text; cse text; opr text:='opr_ad289Sintetico0000000001'; huella_aut text; huella_cse text;
 base_ses record; base_con record; ajeno record;
BEGIN
 SELECT vinculo INTO STRICT b FROM pg_temp.ad289_base WHERE clase='externa_personal';
 sem:=encode(sha256(convert_to('ad289:sesion','UTF8')),'hex');
 ses:='ses_'||substr(sem,1,36); aut:='aut_'||substr(sem,5,36); ase:='ase_'||substr(sem,9,36); cse:='cse_'||substr(sem,13,36);
 huella_aut:=encode(sha256(convert_to('ad289:autenticacion','UTF8')),'hex');
 s:=jsonb_build_object('provision_ref','pce_ad289_sintetico_00000000000','poblacion','candidato','estado','activo');
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  comp:=jsonb_build_object('referencia',CASE k WHEN 'cuenta' THEN cta WHEN 'persona' THEN per
     WHEN 'perfil' THEN prf WHEN 'contexto' THEN 'vca_ad289_sintetico_00000000001' ELSE 'vin_ad289_sintetico_00000000001' END,
   'version',1,'procedencia_ref','prc_ad289_sintetico_000000000001','procedencia_version',1,
   'procedencia_huella_sha256',repeat('a',64),'procedencia_autoridad','autoridad_maestra_acreditada',
   'estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z');
  IF k='vinculo_candidato' THEN comp:=comp||jsonb_build_object('candidato_ref','can_ad289_sintetico_000000000001'); END IF;
  s:=s||jsonb_build_object(k,comp);
 END LOOP;
 h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(s,1);
 PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(s,0,null,h);
 -- Sesión externa: misma política, superficie, método y garantía que la real.
 SELECT x.* INTO STRICT base_ses FROM vec_autorizacion.sesion_autenticacion_v1 x
  WHERE x.sesion_ref=b->>'sesion_ref' AND x.autenticacion_ref=b->>'autenticacion_ref';
 SELECT x.* INTO STRICT base_con FROM vec_identidad_sesiones_v1.consumo_asercion x
  WHERE x.sesion_ref=b->>'sesion_ref';
 t:=clock_timestamp();
 INSERT INTO vec_identidad_sesiones_v1.cuenta(cuenta_ref,cuenta_privilegiada,cuenta_ordinaria_ref,provisionada_en,acto_ref)
 VALUES(cta,false,NULL,t-interval '1 second','opr_ct_dev_cuenta_'||md5('ad289:acto'));
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta(cuenta_ref,revision,estado,registrada_en,acto_ref)
 VALUES(cta,1,'activa',t-interval '1 second','opr_ct_dev_estado_'||md5('ad289:acto'));
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1(sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
  cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,metodo_observado,garantia_observada,
  politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en,registrada_en)
 VALUES(ses,aut,huella_aut,ase,cta,cta,false,'externa_personal',base_ses.metodo_observado,base_ses.garantia_observada,
  base_ses.politica_garantia_ref,base_ses.politica_garantia_huella_sha256,t,t,t);
 huella_cse:=vec_identidad_sesiones_v1.huella_control_sesion_v1(cse,1,ses,'activa',t,t+interval '2 minutes',opr);
 INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,
  sesion_revalidada_en,sesion_valida_hasta,registrada_en)
 VALUES(cse,1,ses,'activa',huella_cse,t,t+interval '2 minutes',t);
 INSERT INTO vec_identidad_sesiones_v1.consumo_asercion(operacion_ref,esquema_hmac,dominio_hmac_ref,clave_hmac_id,
  clave_hmac_version,asercion_id_hmac,sesion_id_hmac,sujeto_id_hmac,cuenta_id_hmac,cuenta_ordinaria_id_hmac,
  autenticacion_ref,autenticacion_huella_sha256,asercion_ref,sesion_ref,control_sesion_ref,control_sesion_revision,
  cuenta_ref,cuenta_revision,cuenta_ordinaria_ref,cuenta_ordinaria_revision,consumida_en)
 VALUES(opr,base_con.esquema_hmac,base_con.dominio_hmac_ref,base_con.clave_hmac_id,base_con.clave_hmac_version,
  sha256(convert_to('ad289:asercion','UTF8')),sha256(convert_to('ad289:sesion','UTF8')),
  sha256(convert_to('ad289:sujeto','UTF8')),sha256(convert_to('ad289:cuenta','UTF8')),NULL,
  aut,huella_aut,ase,ses,cse,1,cta,1,cta,1,t);
 INSERT INTO pg_temp.ad289_actor VALUES('propio',cta,prf,b||jsonb_build_object(
  'autenticacion_ref',aut,'autenticacion_huella_sha256',huella_aut,'asercion_ref',ase,'sesion_ref',ses,
  'control_sesion_ref',cse,'control_sesion_revision',1,'control_sesion_huella_sha256',huella_cse,
  'cuenta_ref',cta,'cuenta_ordinaria_ref',cta,'principal_id',per,'perfil_activo_ref',prf,'cuenta_privilegiada',false,
  'superficie','externa_personal','metodo_observado',base_ses.metodo_observado,
  'garantia_observada',base_ses.garantia_observada,'politica_garantia_ref',base_ses.politica_garantia_ref,
  'politica_garantia_huella_sha256',base_ses.politica_garantia_huella_sha256,
  'autenticacion_verificada_en',to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'sesion_emitida_en',to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'sesion_revalidada_en',to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'sesion_valida_hasta',to_char((t+interval '2 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 IF vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(convert_to(
     (SELECT vinculo::text FROM pg_temp.ad289_actor WHERE clase='propio'),'UTF8')) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD289: sesión sintética no coteja'; END IF;
 -- Otra persona: cualquier otra provisión externa candidata vigente ahora.
 SELECT x.cuenta_ref,x.perfil_ref INTO ajeno FROM vec_contexto_actor_v1.contexto_externo_actual a
  JOIN vec_contexto_actor_v1.contexto_externo_versiones x USING(provision_ref,version)
  WHERE x.cuenta_ref<>cta AND x.familia='candidato' AND x.snapshot->>'estado'='activo'
    AND NOT EXISTS(SELECT 1 FROM jsonb_each(x.snapshot) e WHERE jsonb_typeof(e.value)='object'
      AND (clock_timestamp()<(e.value->>'vigente_desde')::timestamptz
        OR clock_timestamp()>=(e.value->>'vigente_hasta')::timestamptz))
  ORDER BY x.provision_ref LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'AD289: sin otra provisión externa vigente'; END IF;
 INSERT INTO pg_temp.ad289_actor VALUES('ajeno',ajeno.cuenta_ref,ajeno.perfil_ref,NULL);
END $actor$;
-- El LOGIN de ContextoActor no puede tener otros privilegios: valores por psql.
SELECT cuenta_ref AS ad289_cta,perfil_ref AS ad289_prf FROM pg_temp.ad289_actor WHERE clase='propio' \gset
SELECT cuenta_ref AS ad289_cta_ajena,perfil_ref AS ad289_prf_ajeno FROM pg_temp.ad289_actor WHERE clase='ajeno' \gset
SET LOCAL SESSION AUTHORIZATION prueba_ad289_candidato;
SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1();
SELECT registro_contexto_ref FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(
 'oca_ad289_operacion_000000000001','rca_ad289_registro_0000000000001',:'ad289_cta',:'ad289_prf',clock_timestamp());
SELECT registro_contexto_ref FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(
 'oca_ad289_operacion_000000000003','rca_ad289_registro_0000000000003',:'ad289_cta_ajena',:'ad289_prf_ajeno',clock_timestamp());
RESET SESSION AUTHORIZATION;

DO $material$
DECLARE v jsonb; r record; ajeno record; p record; falso record; ctx jsonb; t timestamptz;
BEGIN
 SELECT vinculo INTO STRICT v FROM pg_temp.ad289_actor WHERE clase='propio';
 SELECT x.* INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x
  WHERE x.registro_contexto_ref='rca_ad289_registro_0000000000001';
 ctx:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 v:=v||jsonb_build_object('registro_contexto_ref',r.registro_contexto_ref,
   'contexto_actor_ref',ctx->'contexto_actor_ref','contexto_actor_version',ctx->'contexto_version',
   'contexto_actor_cuenta_version',ctx->'cuenta_version','contexto_actor_huella_sha256',r.huella_sha256,
   'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256);
 INSERT INTO pg_temp.ad289_material VALUES
  ('externo',r.representacion_canonica,convert_to(v::text,'UTF8'),
   pg_temp.ad289_orden('externo',r.representacion_canonica,v,'externa_personal')),
  -- Mismo material presentado por RRHH: superficie externa en canal interno.
  ('externo_canal_interno',r.representacion_canonica,convert_to(v::text,'UTF8'),
   pg_temp.ad289_orden('externo_canal_interno',r.representacion_canonica,v,'interna_corporativa')),
  -- Recibo inexistente en ambos almacenes.
  ('inexistente',r.representacion_canonica,
   convert_to((v||jsonb_build_object('registro_contexto_ref','rca_ad289_registro_0000000000009'))::text,'UTF8'),
   pg_temp.ad289_orden('inexistente',r.representacion_canonica,
    v||jsonb_build_object('registro_contexto_ref','rca_ad289_registro_0000000000009'),'externa_personal'));
 -- Recibo no verificado: bytes coherentes con su huella que no salen de la
 -- provisión publicada (versión de persona inventada), escrito fuera de la API.
 SELECT x.* INTO STRICT p FROM vec_contexto_actor_v1.contexto_externo_versiones x
  WHERE x.provision_ref=r.provision_ref AND x.version=r.provision_version;
 t:=clock_timestamp();
 SELECT * INTO STRICT falso FROM vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(
  jsonb_set(p.snapshot,'{persona,version}','7'::jsonb),t);
 INSERT INTO vec_contexto_actor_v1.registros_contexto_externo_v2(operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,
  metodo,garantia,solicitado_en,resuelto_en,representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,
  manifiesto_procedencia_huella_sha256,autoridad_efectiva,provision_ref,provision_version,provision_huella_sha256)
 VALUES('oca_ad289_operacion_000000000002','rca_ad289_registro_0000000000002',r.cuenta_ref,r.perfil_ref,
  'certificado','alto',t,t,falso.representacion,encode(sha256(falso.representacion),'hex'),
  falso.manifiesto,encode(sha256(falso.manifiesto),'hex'),'autoridad_maestra_acreditada',
  r.provision_ref,r.provision_version,r.provision_huella_sha256);
 ctx:=v||jsonb_build_object('registro_contexto_ref','rca_ad289_registro_0000000000002',
   'contexto_actor_huella_sha256',encode(sha256(falso.representacion),'hex'),
   'manifiesto_procedencia_huella_sha256',encode(sha256(falso.manifiesto),'hex'));
 INSERT INTO pg_temp.ad289_material VALUES('no_verificado',falso.representacion,convert_to(ctx::text,'UTF8'),
  pg_temp.ad289_orden('no_verificado',falso.representacion,ctx,'externa_personal'));
 -- Recibo propio verificado de OTRA persona presentado con esta sesión.
 SELECT x.* INTO STRICT ajeno FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x
  WHERE x.registro_contexto_ref='rca_ad289_registro_0000000000003';
 ctx:=convert_from(ajeno.representacion_canonica,'UTF8')::jsonb;
 v:=v||jsonb_build_object('registro_contexto_ref',ajeno.registro_contexto_ref,
   'contexto_actor_ref',ctx->'contexto_actor_ref','contexto_actor_version',ctx->'contexto_version',
   'contexto_actor_cuenta_version',ctx->'cuenta_version','contexto_actor_huella_sha256',ajeno.huella_sha256,
   'manifiesto_procedencia_huella_sha256',ajeno.manifiesto_procedencia_huella_sha256);
 INSERT INTO pg_temp.ad289_material VALUES('otra_persona',ajeno.representacion_canonica,convert_to(v::text,'UTF8'),
  pg_temp.ad289_orden('otra_persona',ajeno.representacion_canonica,v,'externa_personal'));
 -- La misma, suplantando también cuenta, perfil y principal en el vínculo.
 v:=v||jsonb_build_object('cuenta_ref',ajeno.cuenta_ref,'cuenta_ordinaria_ref',ajeno.cuenta_ref,
   'perfil_activo_ref',ajeno.perfil_ref,'principal_id',ctx->'principal_ref');
 INSERT INTO pg_temp.ad289_material VALUES('otra_persona_suplantada',ajeno.representacion_canonica,convert_to(v::text,'UTF8'),
  pg_temp.ad289_orden('otra_persona_suplantada',ajeno.representacion_canonica,v,'externa_personal'));
END $material$;

\if :{?AD289_REPRODUCIR}
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
DO $reproducir$
DECLARE estado text; mensaje text;
BEGIN
 PERFORM * FROM pg_temp.ad289_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='externo';
 RAISE NOTICE 'AD289-REPRODUCIR: lectura externa asentada (sin defecto)';
EXCEPTION WHEN others THEN
 GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
 RAISE NOTICE 'AD289-REPRODUCIR: lectura externa rechazada sqlstate=% mensaje=%',estado,mensaje;
END $reproducir$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\quit
\endif

-- 3. Lectura externa con contexto externo verificado: se asienta.
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
DO $externo_ok$
DECLARE x record;
BEGIN
 SELECT * INTO STRICT x FROM pg_temp.ad289_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='externo';
 IF x.auditoria_ref !~ '^aud_v3_li_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD289: acuse externo inválido'; END IF;
END $externo_ok$;
-- 4. Denegaciones: contexto no verificado, inexistente o de otra persona.
DO $denegaciones$
DECLARE r record; estado text;
BEGIN
 FOR r IN SELECT * FROM (VALUES('no_verificado','22023'),('inexistente','42501'),
   ('otra_persona','42501'),('otra_persona_suplantada','22023')) x(caso,esperado)
 LOOP
  estado:=NULL;
  BEGIN
   PERFORM * FROM pg_temp.ad289_material m CROSS JOIN LATERAL
    vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
   WHERE m.caso=r.caso;
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  END;
  IF estado IS DISTINCT FROM r.esperado THEN
   RAISE EXCEPTION 'AD289: caso % esperaba % y obtuvo %',r.caso,r.esperado,coalesce(estado,'éxito');
  END IF;
 END LOOP;
END $denegaciones$;
RESET SESSION AUTHORIZATION;
-- 5. RRHH no puede presentar el contexto ni la sesión externos.
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_rrhh_lector;
DO $canal_interno$
DECLARE estado text;
BEGIN
 BEGIN
  PERFORM * FROM pg_temp.ad289_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
  WHERE m.caso='externo_canal_interno';
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
 END;
 IF estado IS DISTINCT FROM '42501' THEN
  RAISE EXCEPTION 'AD289: canal interno con contexto externo obtuvo %',coalesce(estado,'éxito');
 END IF;
END $canal_interno$;
RESET SESSION AUTHORIZATION;

-- 6. Sin mezcla (inyecciones fuera de la API, revertidas al final): un recibo
--    presente en los dos almacenes no vale, ni un recibo interno de una cuenta
--    con población externa segregada.
INSERT INTO vec_contexto_actor_v1.registros_contexto(operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
 solicitado_en,resuelto_en,representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,
 manifiesto_procedencia_huella_sha256,autoridad_efectiva)
SELECT x.operacion_ref,x.registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,solicitado_en,resuelto_en,
 representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva
FROM vec_contexto_actor_v1.registros_contexto_externo_v2 e
CROSS JOIN LATERAL (VALUES(e.operacion_ref,e.registro_contexto_ref),
 ('oca_ad289_operacion_000000000005','rca_ad289_registro_0000000000005')) x(operacion_ref,registro_contexto_ref)
WHERE e.registro_contexto_ref='rca_ad289_registro_0000000000001';
INSERT INTO pg_temp.ad289_material
SELECT 'ambos_almacenes',contexto,vinculo,orden||jsonb_build_object('intento_ref',
 'lectura_'||substr(encode(sha256(convert_to('ad289:ambos_almacenes','UTF8')),'hex'),1,32))
FROM pg_temp.ad289_material WHERE caso='externo'
UNION ALL
SELECT 'interno_de_externo',contexto,
 convert_to((convert_from(vinculo,'UTF8')::jsonb||jsonb_build_object('registro_contexto_ref','rca_ad289_registro_0000000000005'))::text,'UTF8'),
 orden||jsonb_build_object('registro_contexto_ref','rca_ad289_registro_0000000000005','intento_ref',
 'lectura_'||substr(encode(sha256(convert_to('ad289:interno_de_externo','UTF8')),'hex'),1,32))
FROM pg_temp.ad289_material WHERE caso='externo';
SET LOCAL SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
DO $mezcla$
DECLARE r record; estado text;
BEGIN
 FOR r IN SELECT unnest(ARRAY['ambos_almacenes','interno_de_externo']) caso LOOP
  estado:=NULL;
  BEGIN
   PERFORM * FROM pg_temp.ad289_material m CROSS JOIN LATERAL
    vec_autorizacion_atestada_v3.registrar_lectura_inscripcion_v1(m.contexto,m.vinculo,m.orden)
   WHERE m.caso=r.caso;
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  END;
  IF estado IS DISTINCT FROM '42501' THEN
   RAISE EXCEPTION 'AD289: mezcla en caso % obtuvo %',r.caso,coalesce(estado,'éxito');
  END IF;
 END LOOP;
END $mezcla$;
RESET SESSION AUTHORIZATION;

-- 7. Asientos: tres lecturas aceptadas y ninguna de las denegadas.
DO $asientos$
DECLARE aceptadas text[]; denegadas text[];
BEGIN
 SELECT array_agg(orden->>'intento_ref') INTO STRICT aceptadas FROM pg_temp.ad289_material
  WHERE caso IN('legado_externo','interno','externo');
 SELECT array_agg(orden->>'intento_ref') INTO STRICT denegadas FROM pg_temp.ad289_material
  WHERE caso NOT IN('legado_externo','interno','externo');
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE tipo_registro='lectura_inscripcion' AND intento_ref=ANY(aceptadas))<>3
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     JOIN pg_temp.ad289_material m ON m.orden->>'intento_ref'=a.intento_ref AND m.caso='externo'
     WHERE a.canal='externa_personal' AND a.registro_contexto_ref='rca_ad289_registro_0000000000001'
       AND a.actor_ref='per_'||md5('ad289:persona'))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE intento_ref=ANY(denegadas))
 THEN RAISE EXCEPTION 'AD289: asientos divergentes'; END IF;
END $asientos$;
ROLLBACK;
\echo AD289-PRUEBA-OK
