\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE f_prop oid := to_regprocedure('vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(text,bigint,text,text)');
 f_rrhh oid := to_regprocedure('vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text,text)');
 f_propio oid := to_regprocedure('vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(text,text,text,text,timestamptz)');
 f_consulta oid := to_regprocedure('vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(text,text,text,text,text)');
 f_respuesta oid := to_regprocedure('vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 f_interna oid := to_regprocedure('vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,text)');
 f_frontera oid := to_regprocedure('vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)');
BEGIN
 IF f_prop IS NULL OR f_rrhh IS NULL OR f_propio IS NULL OR f_consulta IS NULL
    OR f_respuesta IS NULL OR f_interna IS NULL OR f_frontera IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_justificante_documental_b119') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119') IS NULL
 THEN RAISE EXCEPTION 'B119: faltan piezas de integración' USING ERRCODE='55000'; END IF;
 IF vec_bolsa_llamamientos.recurso_justificante_renuncia_v1('p-1','llamamiento:'||repeat('a',64))
    IS DISTINCT FROM 'justificante-renuncia:'||encode(sha256(convert_to('p-1'||chr(31)||'llamamiento:'||repeat('a',64),'UTF8')),'hex')
    OR vec_bolsa_llamamientos.recurso_justificante_renuncia_v1('p'||chr(31)||'1','llamamiento:'||repeat('a',64)) IS NOT NULL
 THEN RAISE EXCEPTION 'B119: recurso no une participación y llamamiento sin ambigüedad' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (f_prop,f_rrhh,f_propio,f_consulta)
   AND (p.proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR NOT p.prosecdef OR p.provolatile<>'v'))
    OR NOT has_function_privilege('vec_documentos_propietario',f_prop,'EXECUTE')
    OR NOT has_function_privilege('vec_documentos_propietario',f_rrhh,'EXECUTE')
    OR NOT has_function_privilege('vec_documentos_propietario',f_propio,'EXECUTE')
    OR NOT has_function_privilege('vec_documentos_propietario',f_consulta,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_prop,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_rrhh,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_propio,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_consulta,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
      WHERE p.oid IN (f_prop,f_rrhh,f_propio,f_consulta) AND a.grantee=0)
 THEN RAISE EXCEPTION 'B119: autorización SQL abierta' USING ERRCODE='55000'; END IF;
 IF strpos(pg_get_functiondef(f_respuesta),'validar_justificante_renuncia_bolsa_v1')=0
    OR strpos(pg_get_functiondef(f_respuesta),'respuesta_justificante_documental_b119')=0
    OR strpos(pg_get_functiondef(f_interna),'respuesta_justificante_documental_b119')=0
    OR strpos(pg_get_functiondef(f_frontera),'/api/vec/bolsa/mi-bolsa/justificantes')=0
 THEN RAISE EXCEPTION 'B119: respuesta o replay sin custodia' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.respuesta_justificante_documental_b119'::regclass
    AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.preparacion_justificante_renuncia_b119'::regclass
     AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_constraint co
     WHERE co.conrelid='vec_bolsa_llamamientos.respuesta_justificante_documental_b119'::regclass
       AND co.contype='f' AND co.condeferrable AND co.condeferred)
 THEN RAISE EXCEPTION 'B119: evidencia durable sin RLS o FK diferida' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
