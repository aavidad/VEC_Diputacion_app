\set ON_ERROR_STOP on
-- AD208: el rechazo de AD172 «origen de consumo no acreditado» deja de usar
-- SQLSTATE 42501 (el de una denegación de permisos) y pasa a VA172, propio.
-- Falta de configuración técnica no es una denegación: los adaptadores lo
-- tratan como «no disponible» y no como «prohibido». El DETAIL dice qué terna
-- falta (audiencia, operación y canal; nunca el LOGIN) para que el DBA sepa
-- qué fila añadir. Solo cambia esa sentencia del núcleo: firma, propietario,
-- ACL, configuración y dependencias quedan iguales. Sin DOWN; la reejecución
-- se rechaza.
--
-- Va después de toda migración que mida el núcleo con su huella completa.
-- Pendientes en la principal al escribirla: AD195, AD196, AD178, AD177 y AD190
-- (en main) y AD197, AD203 y AD200 (PR abiertas). Las que lleguen después
-- miden la postimagen de esta. AD207 mide por marcas y conmuta con esta.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000208',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $nucleo$
DECLARE
 f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;actual text;meta jsonb;acl aclitem[];propietario oid;config text[];definidora boolean;
 deps jsonb;compartidas jsonb;nueva text;
 antiguo text:=$antiguo$        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='origen de consumo no acreditado';$antiguo$;
 nuevo text:=$nuevo$        -- AD208: falta de configuración técnica, no denegación. Sin LOGIN.
        RAISE EXCEPTION USING ERRCODE='VA172', MESSAGE='origen de consumo no acreditado',
            DETAIL=pg_catalog.format('audiencia=%s operacion=%s canal=%s',
                c ->> 'audiencia_consumo', c ->> 'operacion', v_canal_origen),
            HINT='falta su fila en configuracion_origen_consumos_v1 o el LOGIN de la sesión no es el de la fila';$nuevo$;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR f IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
 THEN RAISE EXCEPTION 'AD208: PARO clave=preimagen actual=incompatible esperado=AD172_instalada' USING ERRCODE='55000'; END IF;

 SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF pg_catalog.strpos(original,'ERRCODE=''VA172''')<>0
 THEN RAISE EXCEPTION 'AD208: PARO clave=reejecucion actual=VA172_presente esperado=42501' USING ERRCODE='55000'; END IF;
 -- Mismas garantías que comprueba AD172 sobre el núcleo.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
 OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_catalog.aclexplode(coalesce(acl,pg_catalog.acldefault('f',propietario))))<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(coalesce(acl,pg_catalog.acldefault('f',propietario))) a
   WHERE a.grantee=propietario AND a.grantor=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD208: PARO clave=metadatos_nucleo actual=distintos esperado=AD172' USING ERRCODE='55000'; END IF;
 -- La sentencia aparece una sola vez, justo tras resolver el origen.
 IF pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,antiguo,''))<>pg_catalog.length(antiguo)
 OR pg_catalog.strpos(original,'IF v_proceso_origen IS NULL THEN'||E'\n'||antiguo)=0
 THEN RAISE EXCEPTION 'AD208: PARO clave=marca actual=no_unica esperado=una_coincidencia_tras_resolver' USING ERRCODE='55000'; END IF;

 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database());

 nueva:=pg_catalog.replace(original,antiguo,nuevo);
 EXECUTE nueva;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR pg_catalog.replace(actual,nuevo,antiguo) IS DISTINCT FROM original
 OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_catalog.pg_shdepend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f
      AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD208: PARO clave=delta_o_metadatos actual=divergente esperado=solo_una_sentencia' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
