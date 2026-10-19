\set ON_ERROR_STOP on
-- AD299 (F1-85): el intento nominal denegado del aspirante en el portal externo
-- coteja el contexto (CA47) y la sesión (IS22) en los almacenes del portal
-- externo. Antes iba siempre a CA26/CA27 e IS20 y fallaba con P0002 (503).
-- Un intento externo con sesión vigente al resolver el contexto deja asiento;
-- inexistente, ajeno, alterado, caducado, en los dos almacenes o de cuenta
-- externa en el almacén interno se deniega sin asiento y con su código exacto,
-- nunca P0002. El canal interno (RRHH) sigue igual, también su repetición.
-- Ejecutar SOLO en copia aislada PG18 desechable, con -v AD299_DISPOSABLE_CLONE=1.
-- Todo ocurre en una transacción que termina en ROLLBACK.
-- Con -v AD299_REPRODUCIR=1 (antes de AD299) solo muestra el fallo previo.
\if :{?AD299_DISPOSABLE_CLONE}
\else
\echo 'AD299: falta AD299_DISPOSABLE_CLONE; prueba omitida sin efectos'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='180s';
-- Primero el latido del sellado, para no chocar con el sellador en marcha.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp() WHERE control;

-- Registradores técnicos AD169 de la prueba, uno por canal (se revierten).
CREATE ROLE prueba_ad299_reg_ext LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE prueba_ad299_reg_int LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_registrador_intentos TO prueba_ad299_reg_ext WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_autorizacion_atestada_v3_registrador_intentos TO prueba_ad299_reg_int WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_runtime_intentos(login_nombre,proceso,canal) VALUES
 ('prueba_ad299_reg_ext','vec-prueba-ad299-ext','externa_personal'),
 ('prueba_ad299_reg_int','vec-prueba-ad299-int','interna_corporativa');

CREATE TEMP TABLE ad299_material(caso text PRIMARY KEY,contexto bytea NOT NULL,vinculo bytea NOT NULL,orden jsonb NOT NULL);
GRANT SELECT ON ad299_material TO prueba_ad299_reg_ext,prueba_ad299_reg_int,vec_autorizacion_atestada_v3_propietario;
CREATE TEMP TABLE ad299_base(clase text PRIMARY KEY,contexto bytea NOT NULL,vinculo jsonb NOT NULL);
-- Partes de autenticación de las sesiones sintéticas (claves del vínculo).
CREATE TEMP TABLE ad299_sesion(nombre text PRIMARY KEY,cuenta_ref text NOT NULL,parte jsonb NOT NULL);

-- Orden AD169 de un intento denegado (16 claves).
CREATE FUNCTION pg_temp.ad299_orden(p_caso text,p_contexto bytea,p_vinculo jsonb,p_canal text)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object(
  'intento_ref','intento_'||substr(encode(sha256(convert_to('ad299:'||p_caso,'UTF8')),'hex'),1,32),
  'registro_contexto_ref',p_vinculo->>'registro_contexto_ref',
  'contexto_sha256',encode(sha256(p_contexto),'hex'),
  'procedencia_sha256',p_vinculo->>'manifiesto_procedencia_huella_sha256',
  'autenticacion_ref',p_vinculo->>'autenticacion_ref','sesion_ref',p_vinculo->>'sesion_ref',
  'autenticacion_sha256',p_vinculo->>'autenticacion_huella_sha256',
  'accion',CASE p_canal WHEN 'externa_personal' THEN 'bolsa.inscripcion.solicitud.registrar' ELSE 'bolsa.inscripcion.rrhh.listar' END,
  'modulo_id','bolsa',
  'recurso_ref','bolsa:convocatoria:'||substr(encode(sha256(convert_to('ad299:recurso','UTF8')),'hex'),1,32),
  'finalidad_ref',CASE p_canal WHEN 'externa_personal' THEN 'inscripcion_convocatoria' ELSE 'consulta_inscripcion_rrhh' END,
  'resultado','denegado','motivo_ref','permiso_denegado',
  'proceso',CASE p_canal WHEN 'externa_personal' THEN 'vec-prueba-ad299-ext' ELSE 'vec-prueba-ad299-int' END,
  'canal',p_canal,'correlacion_ref','correlacion_ad299_'||p_caso)
$f$;

-- Material histórico real de la copia: una sesión externa heredada (almacén
-- interno, proceso integrado) y una interna de RRHH que superan los cotejos.
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
     INSERT INTO pg_temp.ad299_base VALUES(superficie,r.representacion_canonica,v);
     n:=1; EXIT;
    END IF;
   EXCEPTION WHEN others THEN CONTINUE;
   END;
  END LOOP;
  IF n<>1 THEN RAISE EXCEPTION 'AD299: sin material histórico apto para %',superficie; END IF;
 END LOOP;
 INSERT INTO pg_temp.ad299_material
 SELECT 'legado_externo',b.contexto,convert_to(b.vinculo::text,'UTF8'),
        pg_temp.ad299_orden('legado_externo',b.contexto,b.vinculo,'externa_personal')
 FROM pg_temp.ad299_base b WHERE b.clase='externa_personal'
 UNION ALL
 SELECT 'interno',b.contexto,convert_to(b.vinculo::text,'UTF8'),
        pg_temp.ad299_orden('interno',b.contexto,b.vinculo,'interna_corporativa')
 FROM pg_temp.ad299_base b WHERE b.clase='interna_corporativa'
 UNION ALL
 -- Material interno presentado por el registrador externo.
 SELECT 'interno_por_externo',b.contexto,convert_to(b.vinculo::text,'UTF8'),
        pg_temp.ad299_orden('interno_por_externo',b.contexto,b.vinculo,'externa_personal')
 FROM pg_temp.ad299_base b WHERE b.clase='interna_corporativa';
END $base$;

-- Sesión sintética en el almacén externo IS07, con la misma forma que
-- registrar_sesion_v1 (huella de control incluida). Devuelve su parte del vínculo.
CREATE FUNCTION pg_temp.ad299_nueva_sesion(p_clave text,p_cuenta text,p_inicio timestamptz,p_duracion interval,
 p_esquema text,p_dominio text,p_clave_id text,p_clave_version bigint,p_metodo text,p_garantia text,
 p_politica text,p_politica_sha text)
RETURNS void LANGUAGE plpgsql AS $f$
DECLARE sem text:=encode(sha256(convert_to('ad299:sesion:'||p_clave,'UTF8')),'hex');
 ses text:='ses_'||substr(sem,1,36); aut text:='aut_'||substr(sem,5,36);
 ase text:='ase_'||substr(sem,9,36); cse text:='cse_'||substr(sem,13,36);
 opr text:='opr_ad299_'||substr(sem,17,24); huella_aut text:=encode(sha256(convert_to('ad299:aut:'||p_clave,'UTF8')),'hex');
 huella_cse text; fin timestamptz:=p_inicio+p_duracion;
 iso text:='YYYY-MM-DD"T"HH24:MI:SS.US"Z"';
BEGIN
 huella_cse:=vec_identidad_sesiones_v1.huella_control_sesion_v1(cse,1,ses,'activa',p_inicio,fin,opr);
 INSERT INTO vec_identidad_externa_v1.sesion VALUES(ses,aut,ase,opr,p_cuenta,1,p_esquema,p_dominio,p_clave_id,p_clave_version,
  sha256(convert_to('ad299:ase:'||p_clave,'UTF8')),sha256(convert_to('ad299:ses:'||p_clave,'UTF8')),
  sha256(convert_to('ad299:suj:'||p_clave,'UTF8')),sha256(convert_to('ad299:cta:'||p_clave,'UTF8')),
  p_metodo,p_garantia,huella_aut,p_inicio,p_inicio,p_politica,p_politica_sha,p_inicio);
 INSERT INTO vec_identidad_externa_v1.control_sesion VALUES(cse,ses,1,'activa',huella_cse,p_inicio,fin);
 INSERT INTO pg_temp.ad299_sesion VALUES(p_clave,p_cuenta,jsonb_build_object(
  'autenticacion_ref',aut,'autenticacion_huella_sha256',huella_aut,'asercion_ref',ase,'sesion_ref',ses,
  'control_sesion_ref',cse,'control_sesion_revision',1,'control_sesion_huella_sha256',huella_cse,
  'cuenta_ref',p_cuenta,'cuenta_ordinaria_ref',p_cuenta,'cuenta_privilegiada',false,'superficie','externa_personal',
  'metodo_observado',p_metodo,'garantia_observada',p_garantia,'politica_garantia_ref',p_politica,
  'politica_garantia_huella_sha256',p_politica_sha,
  'autenticacion_verificada_en',to_char(p_inicio AT TIME ZONE 'UTC',iso),
  'sesion_emitida_en',to_char(p_inicio AT TIME ZONE 'UTC',iso),
  'sesion_revalidada_en',to_char(p_inicio AT TIME ZONE 'UTC',iso),
  'sesion_valida_hasta',to_char(fin AT TIME ZONE 'UTC',iso)));
END $f$;


-- Aspirante sintético: provisión CA15 publicada, cuenta en el almacén externo
-- IS07 y sesiones registradas allí; contexto resuelto por el resolutor CA15.
CREATE ROLE prueba_ad299_candidato LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_candidato_externo TO prueba_ad299_candidato WITH INHERIT TRUE, SET FALSE;
DO $actor$
DECLARE
 b jsonb; s jsonb; k text; comp jsonb; h text; t timestamptz; x record; cuenta text;
 cta text:='cta_'||md5('ad299:cuenta'); cta2 text:='cta_'||md5('ad299:cuenta:otra');
 cta3 text:='cta_'||md5('ad299:cuenta:inactiva');
 prf text:='prf_'||md5('ad299:perfil'); per text:='per_'||md5('ad299:persona');
 dom text;
BEGIN
 SELECT vinculo INTO STRICT b FROM pg_temp.ad299_base WHERE clase='externa_personal';
 s:=jsonb_build_object('provision_ref','pce_ad299_sintetico_00000000000','poblacion','candidato','estado','activo');
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  comp:=jsonb_build_object('referencia',CASE k WHEN 'cuenta' THEN cta WHEN 'persona' THEN per
     WHEN 'perfil' THEN prf WHEN 'contexto' THEN 'vca_ad299_sintetico_00000000001' ELSE 'vin_ad299_sintetico_00000000001' END,
   'version',1,'procedencia_ref','prc_ad299_sintetico_000000000001','procedencia_version',1,
   'procedencia_huella_sha256',repeat('a',64),'procedencia_autoridad','autoridad_maestra_acreditada',
   'estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z');
  IF k='vinculo_candidato' THEN comp:=comp||jsonb_build_object('candidato_ref','can_ad299_sintetico_000000000001'); END IF;
  s:=s||jsonb_build_object(k,comp);
 END LOOP;
 h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(s,1);
 PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(s,0,null,h);
 -- Cuentas del almacén externo (propia y otra), con la forma de IS07.
 t:=clock_timestamp();
 dom:='idh_'||md5('ad299:dominio');
 FOREACH cuenta IN ARRAY ARRAY[cta,cta2,cta3] LOOP
  INSERT INTO vec_identidad_externa_v1.aprobacion_alta(aprobacion_ref,huella_material_sha256,huella_preimagen_sha256,revision_esperada)
  VALUES('apr_'||md5('ad299:apr:'||cuenta),encode(sha256(convert_to('m'||cuenta,'UTF8')),'hex'),
         encode(sha256(convert_to('p'||cuenta,'UTF8')),'hex'),0);
  INSERT INTO vec_identidad_externa_v1.cuenta VALUES(cuenta,dom,encode(sha256(convert_to('m'||cuenta,'UTF8')),'hex'),
   'apr_'||md5('ad299:apr:'||cuenta),'opr_'||md5('ad299:alta:'||cuenta),t-interval '1 hour');
  INSERT INTO vec_identidad_externa_v1.estado_cuenta VALUES(cuenta,1,CASE cuenta WHEN cta3 THEN 'inactiva' ELSE 'activa' END,
   t-interval '1 hour','opr_'||md5('ad299:estado:'||cuenta));
  INSERT INTO vec_identidad_externa_v1.estado_actual VALUES(cuenta,1,t-interval '1 hour','opr_'||md5('ad299:estado:'||cuenta));
 END LOOP;
 -- Política de la sesión externa heredada real; método y garantía del contexto CA15.
 SELECT x0.metodo_observado,x0.garantia_observada,x0.politica_garantia_ref,x0.politica_garantia_huella_sha256
   INTO STRICT x FROM vec_autorizacion.sesion_autenticacion_v1 x0 WHERE x0.sesion_ref=b->>'sesion_ref';
 -- propia, otra cuenta, caducada (terminó hace 8 min), dos para ambos almacenes
 -- y una de la cuenta inactiva en su revisión.
 PERFORM pg_temp.ad299_nueva_sesion('propia',cta,t,interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
 PERFORM pg_temp.ad299_nueva_sesion('ajena',cta2,t,interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
 PERFORM pg_temp.ad299_nueva_sesion('caducada',cta,t-interval '10 minutes',interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
 PERFORM pg_temp.ad299_nueva_sesion('ambos_ses',cta,t,interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
 PERFORM pg_temp.ad299_nueva_sesion('ambos_aut',cta,t,interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
 PERFORM pg_temp.ad299_nueva_sesion('inactiva',cta3,t,interval '2 minutes','vec.identidad.hmac-sha256.v1',dom,
   'vec.identidad.prueba.ad299',1,'certificado','alto',x.politica_garantia_ref,x.politica_garantia_huella_sha256);
END $actor$;
SELECT 'cta_'||md5('ad299:cuenta') AS ad299_cta,'prf_'||md5('ad299:perfil') AS ad299_prf \gset
SET LOCAL SESSION AUTHORIZATION prueba_ad299_candidato;
SELECT acreditada FROM vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1();
SELECT registro_contexto_ref FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(
 'oca_ad299_operacion_000000000001','rca_ad299_registro_0000000000001',:'ad299_cta',:'ad299_prf',clock_timestamp());
RESET SESSION AUTHORIZATION;

-- Vínculo = base heredada + autenticación de la sesión + contexto registrado.
DO $material$
DECLARE b jsonb; v jsonb; r record; ctx jsonb; caso record; vin jsonb;
BEGIN
 SELECT vinculo INTO STRICT b FROM pg_temp.ad299_base WHERE clase='externa_personal';
 SELECT x.* INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x
  WHERE x.registro_contexto_ref='rca_ad299_registro_0000000000001';
 ctx:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 v:=b||jsonb_build_object('registro_contexto_ref',r.registro_contexto_ref,
   'principal_id',ctx->'principal_ref','perfil_activo_ref',ctx->'perfil_activo_ref',
   'contexto_actor_ref',ctx->'contexto_actor_ref','contexto_actor_version',ctx->'contexto_version',
   'contexto_actor_cuenta_version',ctx->'cuenta_version','contexto_actor_huella_sha256',r.huella_sha256,
   'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256);
 FOR caso IN SELECT * FROM (VALUES
   ('externo','propia',NULL::jsonb),
   ('externo_canal_interno','propia',NULL),
   ('inexistente','propia',jsonb_build_object('sesion_ref','ses_'||repeat('9',36),'autenticacion_ref','aut_'||repeat('9',36),
     'asercion_ref','ase_'||repeat('9',36))),
   -- Huella de autenticación de otra sesión externa con mis referencias.
   ('huella_ajena','propia',jsonb_build_object('autenticacion_huella_sha256',
     encode(sha256(convert_to('ad299:aut:ajena','UTF8')),'hex'))),
   ('alterado','propia',jsonb_build_object('control_sesion_huella_sha256',repeat('c',64))),
   -- Sesión de otra cuenta del almacén externo con mi cuenta y mi contexto.
   ('ajena','ajena',jsonb_build_object('cuenta_ref','cta_'||md5('ad299:cuenta'),'cuenta_ordinaria_ref','cta_'||md5('ad299:cuenta'))),
   ('caducada','caducada',NULL),
   ('ambos_sesion','ambos_ses',NULL),
   ('ambos_aut','ambos_aut',NULL),
   ('inactiva','inactiva',NULL)) x(caso,sesion,cambio)
 LOOP
  SELECT v||s.parte||coalesce(caso.cambio,'{}'::jsonb) INTO STRICT vin FROM pg_temp.ad299_sesion s WHERE s.nombre=caso.sesion;
  INSERT INTO pg_temp.ad299_material VALUES(caso.caso,r.representacion_canonica,convert_to(vin::text,'UTF8'),
   pg_temp.ad299_orden(caso.caso,r.representacion_canonica,vin,
    CASE caso.caso WHEN 'externo_canal_interno' THEN 'interna_corporativa' ELSE 'externa_personal' END));
 END LOOP;
END $material$;

\if :{?AD299_REPRODUCIR}
SET LOCAL SESSION AUTHORIZATION prueba_ad299_reg_ext;
DO $reproducir$
DECLARE estado text; mensaje text;
BEGIN
 PERFORM * FROM pg_temp.ad299_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='externo';
 RAISE NOTICE 'AD299-REPRODUCIR: intento externo asentado (sin defecto)';
EXCEPTION WHEN others THEN
 GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
 RAISE NOTICE 'AD299-REPRODUCIR: intento externo rechazado sqlstate=% mensaje=%',estado,mensaje;
END $reproducir$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\quit
\endif

-- 0. Ni los registradores ni su rol pueden llamar a los cotejos de CA47 e IS22.
DO $acl$
DECLARE f oid; r text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)'::regprocedure::oid,
   'vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea)'::regprocedure::oid] LOOP
  FOREACH r IN ARRAY ARRAY['prueba_ad299_reg_ext','prueba_ad299_reg_int',
    'vec_autorizacion_atestada_v3_registrador_intentos','vec_bolsa_llamamientos_propietario'] LOOP
   IF has_function_privilege(r,f,'EXECUTE') THEN RAISE EXCEPTION 'AD299: % puede ejecutar %',r,f::regprocedure; END IF;
  END LOOP;
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f AND a.grantee=0) THEN
   RAISE EXCEPTION 'AD299: % abierta a PUBLIC',f::regprocedure; END IF;
 END LOOP;
END $acl$;

-- 1. Intentos externos aceptados: sesión del almacén externo y sesión heredada.
SET LOCAL SESSION AUTHORIZATION prueba_ad299_reg_ext;
DO $externo_ok$
DECLARE x record; y record; c text;
BEGIN
 FOREACH c IN ARRAY ARRAY['externo','legado_externo'] LOOP
  SELECT * INTO STRICT x FROM pg_temp.ad299_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
  WHERE m.caso=c;
  IF x.auditoria_ref !~ '^aud_v3_i_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD299: acuse % inválido',c; END IF;
  -- La repetición exacta devuelve el mismo acuse, sin otro asiento.
  SELECT * INTO STRICT y FROM pg_temp.ad299_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
  WHERE m.caso=c;
  IF (y.auditoria_ref,y.secuencia,y.huella_sha256) IS DISTINCT FROM (x.auditoria_ref,x.secuencia,x.huella_sha256) THEN
   RAISE EXCEPTION 'AD299: repetición % con acuse distinto',c; END IF;
 END LOOP;
END $externo_ok$;
RESET SESSION AUTHORIZATION;

-- 2. Canal interno (RRHH) igual que antes: asienta, repite y detecta conflicto.
SET LOCAL SESSION AUTHORIZATION prueba_ad299_reg_int;
DO $interno$
DECLARE x record; y record; estado text;
BEGIN
 SELECT * INTO STRICT x FROM pg_temp.ad299_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='interno';
 IF x.auditoria_ref !~ '^aud_v3_i_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'AD299: acuse interno inválido'; END IF;
 SELECT * INTO STRICT y FROM pg_temp.ad299_material m CROSS JOIN LATERAL
  vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
 WHERE m.caso='interno';
 IF (y.auditoria_ref,y.secuencia,y.huella_sha256) IS DISTINCT FROM (x.auditoria_ref,x.secuencia,x.huella_sha256) THEN
  RAISE EXCEPTION 'AD299: repetición interna con acuse distinto'; END IF;
 estado:=NULL;
 BEGIN
  PERFORM * FROM pg_temp.ad299_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,
    m.orden||jsonb_build_object('motivo_ref','otro_motivo'))
  WHERE m.caso='interno';
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
 END;
 IF estado IS DISTINCT FROM '23505' THEN
  RAISE EXCEPTION 'AD299: conflicto interno obtuvo %',coalesce(estado,'éxito'); END IF;
 -- RRHH no puede presentar el material externo.
 estado:=NULL;
 BEGIN
  PERFORM * FROM pg_temp.ad299_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
  WHERE m.caso='externo_canal_interno';
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
 END;
 IF estado IS DISTINCT FROM '22023' THEN
  RAISE EXCEPTION 'AD299: canal interno con material externo obtuvo %',coalesce(estado,'éxito'); END IF;
END $interno$;
RESET SESSION AUTHORIZATION;

-- Inyecciones fuera de la API (revertidas): cuenta del almacén externo con
--    sesión en el interno; las sesiones ambos_ses y ambos_aut también en el interno.
DO $mezcla$
DECLARE b jsonb; p jsonb; a jsonb; base_ses record; base_con record; t timestamptz:=clock_timestamp();
 cta text:='cta_'||md5('ad299:cuenta'); sem text:=encode(sha256(convert_to('ad299:interno','UTF8')),'hex');
 ses text; aut text; ase text; cse text; opr text:='opr_ad299_interno_000000000001'; huella_aut text; huella_cse text;
 iso text:='YYYY-MM-DD"T"HH24:MI:SS.US"Z"'; r record; ctx jsonb;
BEGIN
 SELECT vinculo INTO STRICT b FROM pg_temp.ad299_base WHERE clase='externa_personal';
 SELECT x.* INTO STRICT base_ses FROM vec_autorizacion.sesion_autenticacion_v1 x WHERE x.sesion_ref=b->>'sesion_ref';
 SELECT x.* INTO STRICT base_con FROM vec_identidad_sesiones_v1.consumo_asercion x WHERE x.sesion_ref=b->>'sesion_ref';
 ses:='ses_'||substr(sem,1,36); aut:='aut_'||substr(sem,5,36); ase:='ase_'||substr(sem,9,36); cse:='cse_'||substr(sem,13,36);
 huella_aut:=encode(sha256(convert_to('ad299:aut:interno','UTF8')),'hex');
 INSERT INTO vec_identidad_sesiones_v1.cuenta(cuenta_ref,cuenta_privilegiada,cuenta_ordinaria_ref,provisionada_en,acto_ref)
 VALUES(cta,false,NULL,t-interval '1 second','opr_ct_dev_cuenta_'||md5('ad299:acto'));
 INSERT INTO vec_identidad_sesiones_v1.estado_cuenta(cuenta_ref,revision,estado,registrada_en,acto_ref)
 VALUES(cta,1,'activa',t-interval '1 second','opr_ct_dev_estado_'||md5('ad299:acto'));
 -- Ventana que contiene la resolución del contexto propio.
 SELECT x.resuelto_en INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 x
  WHERE x.registro_contexto_ref='rca_ad299_registro_0000000000001';
 t:=r.resuelto_en-interval '1 second';
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
  sha256(convert_to('ad299:i:asercion','UTF8')),sha256(convert_to('ad299:i:sesion','UTF8')),
  sha256(convert_to('ad299:i:sujeto','UTF8')),sha256(convert_to('ad299:i:cuenta','UTF8')),NULL,
  aut,huella_aut,ase,ses,cse,1,cta,1,cta,1,t);
 SELECT convert_from(m.vinculo,'UTF8')::jsonb INTO STRICT p FROM pg_temp.ad299_material m WHERE m.caso='externo';
 p:=p||jsonb_build_object('autenticacion_ref',aut,'autenticacion_huella_sha256',huella_aut,'asercion_ref',ase,'sesion_ref',ses,
  'control_sesion_ref',cse,'control_sesion_revision',1,'control_sesion_huella_sha256',huella_cse,
  'metodo_observado',base_ses.metodo_observado,'garantia_observada',base_ses.garantia_observada,
  'politica_garantia_ref',base_ses.politica_garantia_ref,'politica_garantia_huella_sha256',base_ses.politica_garantia_huella_sha256,
  'autenticacion_verificada_en',to_char(t AT TIME ZONE 'UTC',iso),'sesion_emitida_en',to_char(t AT TIME ZONE 'UTC',iso),
  'sesion_revalidada_en',to_char(t AT TIME ZONE 'UTC',iso),
  'sesion_valida_hasta',to_char((t+interval '2 minutes') AT TIME ZONE 'UTC',iso));
 INSERT INTO pg_temp.ad299_material SELECT 'cuenta_externa_en_interno',m.contexto,convert_to(p::text,'UTF8'),
  pg_temp.ad299_orden('cuenta_externa_en_interno',m.contexto,p,'externa_personal')
  FROM pg_temp.ad299_material m WHERE m.caso='externo';
 -- Sesión ambos_ses también en el almacén interno (mismo sesion_ref) y ambos_aut
 -- con su autenticacion_ref en el interno (referencia cruzada).
 SELECT parte INTO STRICT a FROM pg_temp.ad299_sesion WHERE nombre='ambos_ses';
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1(sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
  cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,metodo_observado,garantia_observada,
  politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en,registrada_en)
 VALUES(a->>'sesion_ref','aut_'||repeat('7',36),repeat('7',64),'ase_'||repeat('7',36),cta,cta,false,'externa_personal',
  base_ses.metodo_observado,base_ses.garantia_observada,base_ses.politica_garantia_ref,base_ses.politica_garantia_huella_sha256,t,t,t);
 SELECT parte INTO STRICT a FROM pg_temp.ad299_sesion WHERE nombre='ambos_aut';
 INSERT INTO vec_autorizacion.sesion_autenticacion_v1(sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
  cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,metodo_observado,garantia_observada,
  politica_garantia_ref,politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en,registrada_en)
 VALUES('ses_'||repeat('8',36),a->>'autenticacion_ref',repeat('8',64),'ase_'||repeat('8',36),cta,cta,false,'externa_personal',
  base_ses.metodo_observado,base_ses.garantia_observada,base_ses.politica_garantia_ref,base_ses.politica_garantia_huella_sha256,t,t,t);
END $mezcla$;
-- 3. Denegaciones por el registrador externo: código exacto, nunca P0002.
SET LOCAL SESSION AUTHORIZATION prueba_ad299_reg_ext;
DO $denegaciones$
DECLARE r record; estado text;
BEGIN
 FOR r IN SELECT * FROM (VALUES
   ('inexistente','42501'),        -- IS22: sesión en ningún almacén
   ('huella_ajena','42501'),       -- IS22: huella de autenticación de otra sesión
   ('alterado','22023'),           -- IS22: control de sesión alterado
   ('ajena','22023'),              -- IS22: sesión de otra cuenta con mi cuenta
   ('ambos_sesion','42501'),       -- IS22: sesión en los dos almacenes
   ('ambos_aut','42501'),          -- IS22: autenticación en los dos almacenes
   ('cuenta_externa_en_interno','42501'), -- IS22: cuenta externa en el almacén interno
   ('caducada','22023'),           -- AD169: la ventana no contiene el contexto
   ('inactiva','22023'),           -- AD169: cuenta del vínculo distinta del contexto
   ('interno_por_externo','22023') -- AD169: superficie interna en canal externo
  ) x(caso,esperado) LOOP
  estado:=NULL;
  BEGIN
   PERFORM * FROM pg_temp.ad299_material m CROSS JOIN LATERAL
    vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
   WHERE m.caso=r.caso;
  EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  END;
  IF estado IS DISTINCT FROM r.esperado THEN
   RAISE EXCEPTION 'AD299: intento % obtuvo % y se esperaba %',r.caso,coalesce(estado,'éxito'),r.esperado;
  END IF;
 END LOOP;
 -- El registrador externo no escribe en el canal interno.
 estado:=NULL;
 BEGIN
  PERFORM * FROM pg_temp.ad299_material m CROSS JOIN LATERAL
   vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(m.contexto,m.vinculo,m.orden)
  WHERE m.caso='interno';
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
 END;
 IF estado IS DISTINCT FROM '42501' THEN
  RAISE EXCEPTION 'AD299: registrador externo en canal interno obtuvo %',coalesce(estado,'éxito'); END IF;
END $denegaciones$;
RESET SESSION AUTHORIZATION;

-- 4. Asientos: tres intentos aceptados (uno por caso) y ninguno de los denegados.
DO $asientos$
DECLARE aceptadas text[]; denegadas text[];
BEGIN
 SELECT array_agg(orden->>'intento_ref') INTO STRICT aceptadas FROM pg_temp.ad299_material
  WHERE caso IN('legado_externo','interno','externo');
 SELECT array_agg(orden->>'intento_ref') INTO STRICT denegadas FROM pg_temp.ad299_material
  WHERE caso NOT IN('legado_externo','interno','externo');
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE tipo_registro='intento_nominal' AND resultado='denegado' AND intento_ref=ANY(aceptadas))<>3
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     JOIN pg_temp.ad299_material m ON m.orden->>'intento_ref'=a.intento_ref AND m.caso='externo'
     WHERE a.canal='externa_personal' AND a.proceso='vec-prueba-ad299-ext'
       AND a.registro_contexto_ref='rca_ad299_registro_0000000000001'
       AND a.actor_ref='per_'||md5('ad299:persona')
       AND a.sesion_ref=(SELECT parte->>'sesion_ref' FROM pg_temp.ad299_sesion WHERE nombre='propia'))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE intento_ref=ANY(denegadas))
 THEN RAISE EXCEPTION 'AD299: asientos divergentes'; END IF;
END $asientos$;
ROLLBACK;
\echo AD299-PRUEBA-OK
