\set ON_ERROR_STOP on
-- Prueba focal para ejecutar únicamente en una base desechable tras AD3-000044
-- y Bolsa-000011. No crea identidades, capacidades ni material criptográfico.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;

DO $f$
DECLARE
    tabla regclass := 'vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass;
BEGIN
    IF to_regprocedure('vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity)
       OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=tabla AND polname='borrador_llamamiento_solo_propietario'
                      AND polroles=ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid])
       OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
                  WHERE c.oid=tabla AND a.grantee=0 AND a.privilege_type IN ('SELECT','INSERT','UPDATE','DELETE'))
       OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',tabla,'SELECT,INSERT,UPDATE,DELETE')
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                  WHERE p.oid='vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
                    AND a.grantee=0 AND a.privilege_type='EXECUTE')
       OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento()'::regprocedure),'''vec_bolsa_llamamientos_ejecutor''')=0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento()'::regprocedure),'''vec_bolsa_llamamientos\_%'' ESCAPE ''\''')=0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento()'::regprocedure),'''vec_contratacion_temporal\_%'' ESCAPE ''\''')=0
       OR strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'''creacion_borrador_llamamiento_interno_bolsa''')=0
       OR strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'''consulta_borrador_llamamiento_interno_bolsa''')=0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'''campos_permitidos''')=0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'''{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{}}''')=0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),$busca$d->'ambitos'$busca$)<>0
       OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),$busca$d->'campos'$busca$)<>0
       OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=tabla AND contype='u'
                      AND pg_get_constraintdef(oid,true)='UNIQUE (propietario_ref, unidad_ref, clave_idempotencia)') THEN
        RAISE EXCEPTION 'B-BACK-01 estructura, RLS o ACL incompatible' USING ERRCODE='55000';
    END IF;
END $f$;

-- Regresión estructural de AD3-44: invierte exactamente las tres extensiones
-- B-BACK de la postimagen y exige ambos fingerprints de la preimagen canónica.
DO $nucleo_ad3_44$
DECLARE
    f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    actual text;
    reconstruida text;
    marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    exclusion_ct_original text := $ct$ p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$ct$;
    exclusion_ct_bback text := $ct$ p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'$ct$;
    runtime_bolsa_original text := $g$               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')$g$;
    runtime_bolsa_bback text := $g$               (p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento' OR p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa' OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')$g$;
    extension text := $p$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_borradores_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno')
$p$;
BEGIN
    SELECT pg_get_functiondef(f) INTO STRICT actual;
    IF length(actual)-length(replace(actual,extension||marca,''))<>length(extension||marca)
       OR length(actual)-length(replace(actual,exclusion_ct_bback,''))<>length(exclusion_ct_bback)
       OR length(actual)-length(replace(actual,runtime_bolsa_bback,''))<>length(runtime_bolsa_bback) THEN
        RAISE EXCEPTION 'AD3-44 no conserva la transformación B-BACK exacta' USING ERRCODE='55000';
    END IF;
    reconstruida:=replace(actual,extension||marca,marca);
    reconstruida:=replace(reconstruida,runtime_bolsa_bback,runtime_bolsa_original);
    reconstruida:=replace(reconstruida,exclusion_ct_bback,exclusion_ct_original);
    IF md5(reconstruida)<>'a5421f99d431ca1b24b746e2cab95623'
       OR encode(sha256(convert_to(reconstruida,'UTF8')),'hex')<>'1091df6714ab61f8dc4d7ca34ee50f3246e8080530ffcff348893a71886cb45d' THEN
        RAISE EXCEPTION 'AD3-44 no reconstruye la preimagen canónica' USING ERRCODE='55000';
    END IF;
END $nucleo_ad3_44$;

DO $bitacora$
DECLARE t regclass := 'vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass;
        f regprocedure := 'vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_frontera' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND rolinherit AND NOT rolreplication AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles miembro ON miembro.oid=m.member WHERE miembro.rolname='vec_bolsa_llamamientos_registrador_frontera')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_registrador_frontera'::regrole AND pg_has_role('vec_bolsa_llamamientos_registrador_frontera',r.oid,'MEMBER'))
    OR has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','CREATE')
    OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_type WHERE typowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t AND relrowsecurity AND relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='bitacora_intento_borrador_llamamiento_solo_propietario')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR NOT has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','USAGE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',f,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl WHERE p.oid=f AND (acl.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_registrador_frontera'::regrole) OR (acl.grantee='vec_bolsa_llamamientos_registrador_frontera'::regrole AND acl.is_grantable)) AND acl.privilege_type='EXECUTE')
    OR strpos(pg_get_functiondef(f),'PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()')=0
    OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()'::regprocedure),'''vec_bolsa_llamamientos_registrador_frontera''')=0
    OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()'::regprocedure),'''vec_bolsa_llamamientos\_%'' ESCAPE ''\''')=0
    OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()'::regprocedure),'''vec_contratacion_temporal\_%'' ESCAPE ''\''')=0 THEN
  RAISE EXCEPTION 'B-BACK-01 bitácora o ACL incompatible' USING ERRCODE='55000';
 END IF;
END $bitacora$;

DO $historia$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.borrador_llamamiento_historia'::regclass
                   AND contype='u' AND pg_get_constraintdef(oid,true)='UNIQUE (borrador_ref, secuencia)')
       OR EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.borrador_llamamiento_historia'::regclass
                  AND contype='u' AND pg_get_constraintdef(oid,true)='UNIQUE (secuencia)') THEN
        RAISE EXCEPTION 'B-BACK-01 historia no admite dos borradores' USING ERRCODE='55000';
    END IF;
END $historia$;

DO $acl_total$
DECLARE t regclass; f regprocedure; clase record; funcion record;
BEGIN
 FOREACH t IN ARRAY ARRAY['vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_historia'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_auditoria'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_outbox'::regclass,'vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass] LOOP
  IF EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) acl WHERE c.oid=t AND acl.grantee<>c.relowner AND acl.privilege_type IN ('SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER','MAINTAIN')) THEN RAISE EXCEPTION 'ACL inesperada en %',t USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR clase IN SELECT c.oid::regclass AS objeto FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind IN ('r','p','v','m','f') LOOP
  IF has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',clase.objeto,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',clase.objeto,'SELECT,INSERT,UPDATE,REFERENCES') THEN RAISE EXCEPTION 'registrador con ACL de relación o columna en %',clase.objeto USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR clase IN SELECT c.oid::regclass AS objeto FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='S' LOOP
  IF has_sequence_privilege('vec_bolsa_llamamientos_registrador_frontera',clase.objeto,'USAGE,SELECT,UPDATE') THEN RAISE EXCEPTION 'registrador con ACL de secuencia en %',clase.objeto USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY['vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl WHERE p.oid=f AND (acl.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::regrole) OR (acl.grantee='vec_bolsa_llamamientos_ejecutor'::regrole AND acl.is_grantable)) AND acl.privilege_type='EXECUTE') THEN RAISE EXCEPTION 'ACL de función inesperada en %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 f:='vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
 IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl WHERE p.oid=f AND (acl.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_registrador_frontera'::regrole) OR (acl.grantee='vec_bolsa_llamamientos_registrador_frontera'::regrole AND acl.is_grantable)) AND acl.privilege_type='EXECUTE') THEN RAISE EXCEPTION 'ACL de registrador inesperada en %',f USING ERRCODE='55000'; END IF;
 FOR funcion IN SELECT p.oid::regprocedure AS objeto FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' LOOP
  IF funcion.objeto<>f AND has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',funcion.objeto,'EXECUTE') THEN RAISE EXCEPTION 'registrador con función de negocio en %',funcion.objeto USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY['vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl WHERE p.oid=f AND (acl.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR (acl.grantee='vec_bolsa_llamamientos_propietario'::regrole AND acl.is_grantable)) AND acl.privilege_type='EXECUTE') THEN RAISE EXCEPTION 'ACL AD3 inesperada en %',f USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl_total$;

-- Caso hostil ejecutable: un GRANT por columna no concede tabla completa, pero
-- tras el USAGE del esquema permitiría lectura. La sonda debe detectarlo; el
-- ROLLBACK final conserva íntegra la preimagen de esta prueba desechable.
DO $columna_hostil$
DECLARE t regclass := 'vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass;
BEGIN
 GRANT SELECT (borrador_ref) ON TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno TO vec_bolsa_llamamientos_registrador_frontera;
 IF has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',t,'SELECT')
    OR NOT has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',t,'SELECT,INSERT,UPDATE,REFERENCES') THEN
  RAISE EXCEPTION 'sonda de ACL por columna incompatible' USING ERRCODE='55000';
 END IF;
END $columna_hostil$;

-- PostgreSQL 17/18: MAINTAIN también es autoridad sobre la relación y no
-- puede quedar fuera del catálogo efectivo del registrador.
DO $maintain_hostil$
DECLARE t regclass := 'vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass;
BEGIN
 GRANT MAINTAIN ON TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno TO vec_bolsa_llamamientos_registrador_frontera;
 IF NOT has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',t,'MAINTAIN') THEN
  RAISE EXCEPTION 'sonda de ACL MAINTAIN incompatible' USING ERRCODE='55000';
 END IF;
END $maintain_hostil$;

DO $contenido$
BEGIN
    IF vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"Necesidad de ingeniero <>& ágil"}'::jsonb) IS NOT TRUE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"dni sintético"}'::jsonb) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"NIE sintético"}'::jsonb) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"NIE: sintético"}'::jsonb) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"correo email@ejemplo"}'::jsonb) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido('{"resumen":"teléfono sintético"}'::jsonb) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb_build_object('resumen','corte'||chr(8232)||'línea')) IS NOT FALSE
       OR vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb_build_object('resumen','corte'||chr(8233)||'línea')) IS NOT FALSE THEN
        RAISE EXCEPTION 'B-BACK-01 filtro de minimización incompatible' USING ERRCODE='55000';
    END IF;
END $contenido$;
ROLLBACK;

-- Regresión funcional pendiente de arnés AD3 autorizado: emitir dos decisiones
-- vivas separadas (crear/consultar) para la misma persona/unidad/ámbito y probar:
-- 1. mismo comando -> mismo recibo y cero historia/outbox adicional;
-- 2. misma clave con huella distinta -> VBL01 y cero efectos;
-- 3. otro propietario -> 42501 al consultar, sin fila ni auditoría expuesta;
-- 4. revocación entre prelectura y consumo -> rollback íntegro.
-- 5. resumen `dni`, `NIE`, `nif`, `pasaporte`, `email`, `teléfono` y `@`
--    se rechazan con 22023 antes de consumir una decisión.
-- 6. clave de siete caracteres se rechaza; ocho se admite si el resto del
--    comando y la autorización sintética son válidos.
-- 7. Estado hostil: antes de instalar, conceder USAGE/CREATE por PUBLIC,
--    EXECUTE de cualquier función histórica por PUBLIC, un privilegio de tabla
--    o secuencia, propiedad de objeto/esquema, o una membresía transitiva al
--    grupo; 000011 debe fallar con 55000 y el ROLLBACK deja la preimagen igual.
-- 8. Con el grupo nominal aislado, SET ROLE de un LOGIN miembro solo ejecuta
--    registrar_intento_borrador_llamamiento_v1; la misma sesión no ejecuta
--    guardar/consultar ni lee o muta relación o secuencia alguna.
