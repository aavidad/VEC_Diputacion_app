\set ON_ERROR_STOP on
-- CT90: prueba ordinaria del validador puro. No concede ni consume V3,
-- no escribe historia, no publica definiciones y no prueba el cierre instrumentado.
-- Ejecutar únicamente tras CT90 en uno de los dos clones autorizados.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='20s';
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $ct90_prueba$
DECLARE
 v jsonb:=$vector${"Original":{"canon":{"algoritmo":"sha-256","dominio":"vec.dipgra.contratacion-temporal.seguimiento.definicion","version_esquema":1},"estado_inicial":"pendiente_incorporacion","estados":[{"clave":"cancelada","final":true},{"clave":"pendiente_incorporacion","final":false},{"clave":"vigente","final":false}],"huella_sha256":"784362e9ba26f2f4747e93a26ac5d24eb0bd4eee4839eb4adf8822593fa0e3b1","motivos":["ejercicio_incorporacion"],"prohibe_ciclos_silenciosos":true,"publicado_en":"2026-09-07T09:00:00Z","referencia":"ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","transiciones":[{"clase":"ordinaria","clave":"confirmar_incorporacion","destino":"vigente","documentos":[{"obligatorio":false,"tipo_clave":"anexo_ejercicio"},{"obligatorio":true,"tipo_clave":"resolucion_ejercicio"}],"efecto_periodo":"abrir","exige_actor_distinto":false,"motivo_obligatorio":true,"motivos_permitidos":["ejercicio_incorporacion"],"origen":"pendiente_incorporacion","requiere_periodo":true}],"version":1,"vigencia":{"desde":"2026-09-08T00:00:00Z","hasta":"2027-09-08T00:00:00Z"}},"Sucesora":{"canon":{"algoritmo":"sha-256","dominio":"vec.dipgra.contratacion-temporal.seguimiento.definicion","version_esquema":1},"estado_inicial":"pendiente_incorporacion","estados":[{"clave":"cancelada","final":true},{"clave":"cerrado_administrativamente","final":true},{"clave":"pendiente_incorporacion","final":false},{"clave":"vigente","final":false}],"huella_sha256":"63a07c4d20bed9d390b2120da8af53a5cf4e5e4db72a835fc8d4db17a82c40a6","motivos":["cierre_administrativo_ejercicio","ejercicio_incorporacion"],"prohibe_ciclos_silenciosos":true,"publicado_en":"2026-09-10T15:00:00Z","referencia":"ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","transiciones":[{"clase":"ordinaria","clave":"cerrar_administrativamente_sin_cese","destino":"cerrado_administrativamente","documentos":null,"efecto_periodo":"ninguno","exige_actor_distinto":false,"motivo_obligatorio":true,"motivos_permitidos":["cierre_administrativo_ejercicio"],"origen":"vigente","requiere_periodo":false},{"clase":"ordinaria","clave":"confirmar_incorporacion","destino":"vigente","documentos":[{"obligatorio":false,"tipo_clave":"anexo_ejercicio"},{"obligatorio":true,"tipo_clave":"resolucion_ejercicio"}],"efecto_periodo":"abrir","exige_actor_distinto":false,"motivo_obligatorio":true,"motivos_permitidos":["ejercicio_incorporacion"],"origen":"pendiente_incorporacion","requiere_periodo":true}],"version":2,"vigencia":{"desde":"2026-09-10T15:00:00Z","hasta":"2027-09-08T00:00:00Z"}}}$vector$::jsonb;
 o jsonb; s jsonb; m jsonb; t jsonb; rechazada boolean; caso integer;
BEGIN
 IF current_database() !~ '^vec_ct(86|87)_prueba_[0-9a-f]{12}$'
  OR current_user<>'vec_contratacion_temporal_propietario'
  OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
      WHERE oid='vec_contratacion_temporal.cierre87_validar_sucesora(jsonb,jsonb)'::regprocedure) IS DISTINCT FROM 'd465e183252d25f42850a298873dde245a90541a9816cd7b4e24d67bb4730a8f'
 THEN RAISE EXCEPTION 'CT90: clon o función corregida no acreditados'; END IF;
 o:=v->'Original';s:=v->'Sucesora';
 t:=vec_contratacion_temporal.cierre87_validar_sucesora(o,s);
 IF t IS DISTINCT FROM (vec_contratacion_temporal.seguimiento73_definicion(s)#>'{transiciones,0}') THEN RAISE EXCEPTION 'CT90: transición nominal divergente'; END IF;
 -- Cada variante se vuelve a sellar y se valida primero como definición válida:
 -- el rechazo posterior debe proceder del contrato de sucesión, no de su hash.
 FOR caso IN 1..8 LOOP
  m:=CASE caso
   WHEN 1 THEN jsonb_set(s,'{referencia}',to_jsonb('ref:'||repeat('b',64)))
   WHEN 2 THEN jsonb_set(s,'{version}','3'::jsonb)
   WHEN 3 THEN jsonb_set(s,'{publicado_en}',o->'publicado_en')
   WHEN 4 THEN jsonb_set(s,'{estados,0,final}','false'::jsonb)
   WHEN 5 THEN jsonb_set(s,'{transiciones,1,documentos,0,obligatorio}','true'::jsonb)
   WHEN 6 THEN jsonb_set(s,'{transiciones,0,efecto_periodo}','"cerrar"'::jsonb)
   WHEN 7 THEN jsonb_set(s,'{estados,1,final}','false'::jsonb)
   WHEN 8 THEN jsonb_set(s,'{transiciones,0,motivo_obligatorio}','false'::jsonb)
  END;
  m:=jsonb_set(m,'{huella_sha256}',to_jsonb(encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(m,'publicacion')),'hex')));
  PERFORM vec_contratacion_temporal.seguimiento73_definicion(m);
  rechazada:=false;
  BEGIN PERFORM vec_contratacion_temporal.cierre87_validar_sucesora(o,m);
  EXCEPTION WHEN SQLSTATE '22023' THEN rechazada:=true; END;
  IF NOT rechazada THEN RAISE EXCEPTION 'CT90: variante % admitida',caso; END IF;
 END LOOP;
 -- Un motivo original sin uso es válido y también debe quedar conservado.
 o:=jsonb_set(o,'{motivos}','["ejercicio_incorporacion","motivo_original_sin_uso"]'::jsonb);
 o:=jsonb_set(o,'{huella_sha256}',to_jsonb(encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(o,'publicacion')),'hex')));
 PERFORM vec_contratacion_temporal.seguimiento73_definicion(o);
 rechazada:=false;
 BEGIN PERFORM vec_contratacion_temporal.cierre87_validar_sucesora(o,s);
 EXCEPTION WHEN SQLSTATE '22023' THEN rechazada:=true; END;
 IF NOT rechazada THEN RAISE EXCEPTION 'CT90: pérdida de motivo original admitida'; END IF;
 RAISE NOTICE 'CT90: sucesora válida y nueve rechazos de contrato comprobados; sin escrituras';
END $ct90_prueba$;
ROLLBACK;
