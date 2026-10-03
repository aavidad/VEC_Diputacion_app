\set ON_ERROR_STOP on
-- CA27 corrige el cotejo histórico de empleado en la auditoría común.
-- Personal16/CA7 conservan sus contratos y tablas. No concede permisos nuevos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000027:cotejo-empleado-auditoria',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:cotejo_auditoria_intentos:v1',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $cambio$
DECLARE
 f regprocedure := 'vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)'::regprocedure;
 w regprocedure := 'vec_contexto_actor_v1.proyeccion_empleado_personal_v2(text,timestamptz)'::regprocedure;
 pe regprocedure := 'vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)'::regprocedure;
 propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
 ad oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
 personal oid := 'vec_personal_propietario'::regrole;
 p pg_proc%ROWTYPE; antes jsonb; deps jsonb; shdeps jsonb; def text;
 a1 text := $antes1$ registro record; cuenta record; perfil record; persona record; enlace record;$antes1$;
 n1 text := $despues1$ registro record; cuenta record; perfil record; persona record; enlace record;
 empleado record; item_empleado jsonb; numero_empleados integer := 0;$despues1$;
 a2 text := $antes2$           OR vec_contexto_actor_v1.referencia_valida(item->>'vinculo_ref','vin_') IS NOT TRUE THEN$antes2$;
 n2 text := $despues2$           OR (vec_contexto_actor_v1.referencia_valida(item->>'vinculo_ref','vin_') IS NOT TRUE
               AND (item->>'tipo' IS DISTINCT FROM 'empleado'
                    OR vec_contexto_actor_v1.referencia_valida(item->>'vinculo_ref','pep_') IS NOT TRUE)) THEN$despues2$;
 a3 text := $antes3$            RAISE EXCEPTION 'vinculo historico invalido' USING ERRCODE='22023';
        END IF;
    END LOOP;$antes3$;
 n3 text := $despues3$            RAISE EXCEPTION 'vinculo historico invalido' USING ERRCODE='22023';
        END IF;
        IF vec_contexto_actor_v1.referencia_valida(item->>'vinculo_ref','pep_') IS TRUE THEN
            numero_empleados := numero_empleados + 1;
            IF numero_empleados > 1 THEN
                RAISE EXCEPTION 'proyeccion historica empleado duplicada' USING ERRCODE='22023';
            END IF;
            item_empleado := item;
        END IF;
    END LOOP;$despues3$;
 a4 text := $antes4$    IF numero_enlaces <> jsonb_array_length(doc->'vinculos')$antes4$;
 n4 text := $despues4$    IF numero_enlaces + numero_empleados <> jsonb_array_length(doc->'vinculos')$despues4$;
 a5 text := $antes5$    -- Mismos formatos, orden y escapes del serializador propietario base V2.$antes5$;
 n5 text := $despues5$    -- CA27: Personal conserva la autoridad histórica de los enlaces pep_.
    -- El instante ORIGINAL limita también registrada_en antes de elegir versión.
    -- No se consulta el puntero actual ni se renueva una concesión.
    IF numero_empleados = 1 THEN
        IF EXISTS (SELECT 1 FROM jsonb_array_elements(doc->'vinculos') j(e)
                   WHERE j.e->>'tipo'='empleado'
                     AND vec_contexto_actor_v1.referencia_valida(j.e->>'vinculo_ref','vin_') IS TRUE) THEN
            RAISE EXCEPTION 'enlaces historicos empleado ambiguos' USING ERRCODE='22023';
        END IF;
        SELECT * INTO STRICT empleado
          FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(
              perfil.persona_ref, registro.resuelto_en);
        IF empleado.resultado IS DISTINCT FROM 'empleado'
           OR item_empleado->>'vinculo_ref' IS DISTINCT FROM empleado.proyeccion_ref
           OR (item_empleado->>'version')::numeric IS DISTINCT FROM empleado.version
           OR item_empleado->>'referencia' IS DISTINCT FROM empleado.empleado_ref
           OR item_empleado->>'estado' IS DISTINCT FROM 'activo'
           OR item_empleado->>'vigente_desde' IS DISTINCT FROM
              to_char(empleado.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
           OR item_empleado->>'vigente_hasta' IS DISTINCT FROM
              to_char(empleado.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') THEN
            RAISE EXCEPTION 'proyeccion historica empleado divergente' USING ERRCODE='22023';
        END IF;
        -- CA7 añade Personal al final tras excluir los empleados del núcleo.
        enlaces_texto := concat_ws(',', enlaces_texto, format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"estado":"activo","vigente_desde":%s,"vigente_hasta":%s}',
          to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          to_json(empleado.empleado_ref)::text,
          to_json(to_char(empleado.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
          to_json(to_char(empleado.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text));
        enlaces_procedencia_texto := concat_ws(',', enlaces_procedencia_texto, format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":"autoridad_maestra_acreditada"}',
          to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          to_json(empleado.empleado_ref)::text, to_json(empleado.procedencia_ref)::text,
          empleado.procedencia_version::text, to_json(empleado.procedencia_huella_sha256)::text));
    END IF;
    -- Mismos formatos, orden y escapes del serializador propietario base V2.$despues5$;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF current_user<>'vec_contexto_actor_v1_propietario'
    OR p.proowner<>propietario OR NOT p.prosecdef
    OR p.provolatile<>'v' OR p.proparallel<>'u' OR p.prorettype<>'boolean'::regtype
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','statement_timeout=5s','lock_timeout=2s']::text[]
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')<>'852e5101f3d751f43772fe16dd6e4af6a4dcb2b27be34a1bd7e8b9619b26e337'
    OR p.proacl IS DISTINCT FROM ARRAY[
       'vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
       'vec_autorizacion_atestada_v3_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[] THEN
  RAISE EXCEPTION 'PARO clave=CA27.preimagen, fuente_actual=%, fuente_esperada=852e5101f3d751f43772fe16dd6e4af6a4dcb2b27be34a1bd7e8b9619b26e337',
    encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc x WHERE x.oid=w AND x.proowner=propietario AND NOT x.prosecdef
       AND x.proconfig=ARRAY['search_path=pg_catalog']::text[]
       AND encode(sha256(convert_to(x.prosrc,'UTF8')),'hex')='2289982c05a581463ca02de2b49b18dc74d9411d8c5dc329a871efee6c4b2ca8'
       AND x.proacl=ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[])
    OR NOT EXISTS(SELECT 1 FROM pg_proc x WHERE x.oid=pe AND x.proowner=personal AND x.prosecdef
       AND x.proconfig=ARRAY['search_path=pg_catalog','row_security=on']::text[]
       AND encode(sha256(convert_to(x.prosrc,'UTF8')),'hex')='ee33f00aee1cfdacfad434125265f8c407c9679cd6e11da9a8dd31355db2d52a'
       AND x.proacl=ARRAY['vec_personal_propietario=X/vec_personal_propietario',
         'vec_contexto_actor_v1_propietario=X/vec_personal_propietario']::aclitem[]) THEN
  RAISE EXCEPTION 'PARO clave=CA27.dependencias_historicas, esperado=Personal16_CA7_intactos' USING ERRCODE='55000';
 END IF;
 antes := to_jsonb(p)-'prosrc';
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO shdeps FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 def := pg_get_functiondef(f);
 IF length(def)-length(replace(def,a1,''))<>length(a1) THEN
  RAISE EXCEPTION 'PARO clave=CA27.segmento_1, esperado=1' USING ERRCODE='55000';
 END IF;
 def := replace(def,a1,n1);
 IF length(def)-length(replace(def,a2,''))<>length(a2) THEN
  RAISE EXCEPTION 'PARO clave=CA27.segmento_2, esperado=1' USING ERRCODE='55000';
 END IF;
 def := replace(def,a2,n2);
 IF length(def)-length(replace(def,a3,''))<>length(a3) THEN
  RAISE EXCEPTION 'PARO clave=CA27.segmento_3, esperado=1' USING ERRCODE='55000';
 END IF;
 def := replace(def,a3,n3);
 IF length(def)-length(replace(def,a4,''))<>length(a4) THEN
  RAISE EXCEPTION 'PARO clave=CA27.segmento_4, esperado=1' USING ERRCODE='55000';
 END IF;
 def := replace(def,a4,n4);
 IF length(def)-length(replace(def,a5,''))<>length(a5) THEN
  RAISE EXCEPTION 'PARO clave=CA27.segmento_5, esperado=1' USING ERRCODE='55000';
 END IF;
 def := replace(def,a5,n5);
 EXECUTE def;
 IF (SELECT to_jsonb(x)-'prosrc' FROM pg_proc x WHERE x.oid=f) IS DISTINCT FROM antes
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM shdeps
    OR (SELECT encode(sha256(convert_to(x.prosrc,'UTF8')),'hex') FROM pg_proc x WHERE x.oid=f)<>'310cc2852d16cc2c47b281603390be31e05436c9c255bb6f7151c9878ad9e04b' THEN
  RAISE EXCEPTION 'PARO clave=CA27.postimagen, esperado=metadata_ACL_dependencias_intactas_delta_exacto' USING ERRCODE='55000';
 END IF;
END
$cambio$;
COMMIT;
