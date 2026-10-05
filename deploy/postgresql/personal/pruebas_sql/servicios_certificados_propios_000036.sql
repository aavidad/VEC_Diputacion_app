\set ON_ERROR_STOP on
-- Comprobación de AD195 + Personal36 en un clon desechable, como superusuario.
-- Todo termina en ROLLBACK: el LOGIN de prueba y cualquier intento desaparecen.
-- No crea permisos, claves, firmas ni dobles: sólo estructura, ACL y rechazos
-- antes de consumir. El positivo exige un vector firmado por el emisor real.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $estructura$
DECLARE
 f oid:=to_regprocedure('vec_personal.consultar_servicios_certificados_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 c oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_servicios_certificados_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 n oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 def text;
BEGIN
 IF f IS NULL OR c IS NULL OR n IS NULL THEN RAISE EXCEPTION 'P36: objetos ausentes'; END IF;
 IF NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE') OR has_function_privilege('vec_personal_registrador_frontera',f,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR has_function_privilege('vec_personal_ejecutor',c,'EXECUTE') OR NOT has_function_privilege('vec_personal_propietario',c,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=c)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=n)<>1
 OR has_table_privilege('vec_personal_ejecutor','vec_personal.servicio_reconocido_historia','SELECT')
 OR has_table_privilege('vec_personal_ejecutor','vec_personal.relacion_servicio_historia','SELECT')
 THEN RAISE EXCEPTION 'P36: ACL ampliada'; END IF;
 def:=pg_get_functiondef(n);
 -- El perfil aparece en las tres marcas (LOGIN, exclusión y contrato) y en
 -- ningún otro sitio; los consumidores previos se conservan.
 IF (length(def)-length(replace(def,'''servicios_certificados_propios''','')))/length('''servicios_certificados_propios''')<>3
 OR strpos(def,'usuarios_admin_listar')=0 OR strpos(def,'persona_denominacion_leer')=0 OR strpos(def,'registro_empleado_b2')=0
 OR strpos(def,'consumo_confirmado_v4')=0
 THEN RAISE EXCEPTION 'P36: núcleo sin la extensión mínima esperada'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint k WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND k.conname='clave_capacidad_version_audiencia_consumo_check' AND k.convalidated
   AND strpos(pg_get_constraintdef(k.oid,false),'vec_personal.servicios_certificados.v1')>0
   AND strpos(pg_get_constraintdef(k.oid,false),'vec.admin.usuarios.consultar.v1')>0)
 THEN RAISE EXCEPTION 'P36: audiencia ausente o previas perdidas'; END IF;
END $estructura$;

CREATE TEMP TABLE p36_antes ON COMMIT DROP AS
 SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3) AS consumos,
        (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3) AS auditorias;
CREATE ROLE vec_prueba_p36_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_prueba_p36_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;

-- Un superusuario no es el LOGIN nominal: rechazo antes de leer.
DO $superusuario$
BEGIN
 PERFORM vec_personal.consultar_servicios_certificados_propios_v1('{}','\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'P36: superusuario admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $superusuario$;

SET SESSION AUTHORIZATION vec_prueba_p36_login;
DO $negativos$
DECLARE
 contexto text:='{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaa","estado":"activo","vinculos":[]}';
 material text:='{"esquema":"vec.personal.servicios-certificados.consulta.v1","operacion":"servicios_certificados_propios","empleado_ref":"emp_bbbbbbbbbbbbbbbbbbbbbbbb","organismo_ref":"dipgra","vigente_en":"2026-09-25","conocido_en":"2026-09-25T09:59:59.123456Z","actor_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaa","contexto_actor_ref":"vca_aaaaaaaaaaaaaaaaaaaaaaaa","contexto_version":3,"cuenta_ref":"cta_aaaaaaaaaaaaaaaaaaaaaaaa","cuenta_version":2,"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","perfil_version":5,"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaa","persona_version":4}';
 caso text; estado text;
BEGIN
 FOREACH caso IN ARRAY ARRAY['json','claves','esquema','canon','actor','sin_firma'] LOOP
  BEGIN
   PERFORM vec_personal.consultar_servicios_certificados_propios_v1(
    CASE caso WHEN 'json' THEN '{' WHEN 'claves' THEN '{"esquema":"x"}'
     WHEN 'esquema' THEN replace(material,'servicios-certificados.consulta.v1','ficha-propia.consulta.v1')
     WHEN 'canon' THEN replace(material,',"operacion"',' ,"operacion"') ELSE material END,
    convert_to('{"operacion":"personal.servicios_certificados.consultar"}','UTF8'),
    convert_to('{"concedida":true}','UTF8'),'\x6d',
    convert_to(CASE caso WHEN 'actor' THEN replace(contexto,'per_aaaa','per_cccc') ELSE contexto END,'UTF8'),
    4,5,'\x70','\x73','\x65',decode(repeat('00',44),'hex'));
   RAISE EXCEPTION 'P36: caso % admitido',caso;
  EXCEPTION WHEN insufficient_privilege OR invalid_parameter_value THEN
   GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
   IF estado NOT IN ('42501','22023') THEN RAISE; END IF;
  END;
 END LOOP;
END $negativos$;
RESET SESSION AUTHORIZATION;

-- Ningún rechazo ha consumido ni auditado una concesión.
DO $sin_efectos$
BEGIN
 IF (SELECT consumos FROM p36_antes)<>(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
 OR (SELECT auditorias FROM p36_antes)<>(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
 THEN RAISE EXCEPTION 'P36: un rechazo dejó consumo o auditoría'; END IF;
END $sin_efectos$;
SELECT 'P36-COMPROBACION-OK' AS resultado;
ROLLBACK;
