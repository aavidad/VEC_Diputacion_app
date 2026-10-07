\set ON_ERROR_STOP on
-- Ejecutar solo después de instalar AD159 y CT170 en un clon PG18 desechable.
-- Comprueba la frontera nueva, los 29 campos concedidos y AD125/CT152.
-- Incluye rechazo uniforme de dos referencias sin capacidad. Sin datos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
DO $prueba$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 legado regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 proyeccion text:='["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]';
 nucleo text; fachada text; a record; referencia text; anterior text; mensaje text;
BEGIN
 IF f IS NULL OR legado IS NULL THEN
  RAISE EXCEPTION 'AD159 prueba: fachada nueva o legado ausentes'; END IF;
 SELECT p.prosrc INTO nucleo FROM pg_proc p WHERE p.oid=
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT p.prosrc INTO fachada FROM pg_proc p WHERE p.oid=f;
 IF strpos(nucleo,'''consulta_firmas_r5_ct''')=0
    OR strpos(nucleo,'contratacion_temporal.documento.firmas_r5.consultar')=0
    OR strpos(nucleo,'vec_contratacion_temporal.firmas_r5.consultar.v1')=0
    OR strpos(nucleo,proyeccion)=0
    OR strpos(fachada,proyeccion)=0
    OR strpos(fachada,'consumo_nuevo IS NOT TRUE')=0
    OR strpos(fachada,'d->>''recurso_ref'' IS DISTINCT FROM c->>''efecto_ref''')=0
 THEN RAISE EXCEPTION 'AD159 prueba: contrato nuevo incompleto'; END IF;
 IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD159 prueba: propietario o definidora divergente'; END IF;
 FOR a IN SELECT x.grantee,x.privilege_type,x.is_grantable,p.proowner FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid=f LOOP
  IF a.privilege_type<>'EXECUTE' OR a.is_grantable
     OR (a.grantee<>a.proowner AND a.grantee<>'vec_contratacion_temporal_propietario'::regrole::oid)
  THEN RAISE EXCEPTION 'AD159 prueba: ACL abierta'; END IF;
 END LOOP;
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
    OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD159 prueba: ACL de CT divergente'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
    AND strpos(pg_get_constraintdef(c.oid,true),'''vec_contratacion_temporal.firmas_r5.consultar.v1''')>0)
 THEN RAISE EXCEPTION 'AD159 prueba: audiencia no inscrita'; END IF;
 -- El consumidor deniega de modo indistinguible dos referencias opacas
 -- carentes de decisión. CT170 cubre la ausencia autorizada en su prueba.
 FOR referencia IN SELECT unnest(ARRAY['expediente:ad159:a','expediente:ad159:b']) LOOP
  mensaje:=NULL;
  BEGIN
   PERFORM 1 FROM vec_autorizacion_atestada_v3.consumir_consulta_firmas_r5_ct_v3_atestada(
    convert_to(jsonb_build_object('operacion','contratacion_temporal.documento.firmas_r5.consultar',
     'audiencia_consumo','vec_contratacion_temporal.firmas_r5.consultar.v1',
     'efecto_ref',referencia)::text,'UTF8'),
    convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),
    1,1,convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),decode(repeat('00',44),'hex'));
  EXCEPTION WHEN SQLSTATE '42501' THEN
   GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  END;
  IF mensaje IS DISTINCT FROM 'AD3-159: consulta de firmas denegada' THEN
   RAISE EXCEPTION 'AD159 prueba: guarda nominal no ejecutada'; END IF;
  IF anterior IS NOT NULL AND mensaje IS DISTINCT FROM anterior THEN
   RAISE EXCEPTION 'AD159 prueba: rechazo revela referencia'; END IF;
  anterior:=mensaje;
 END LOOP;
END $prueba$;
ROLLBACK;
