\set ON_ERROR_STOP on
SET ROLE vec_personal_propietario;
INSERT INTO vec_personal.org_nodo_historia
 (nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,
  catalogo_entrada_clave,denominacion,retirado,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES
 ('abcdefab-cdef-4abc-8def-abcdefabcdef',1,'org:tempo','unidad:a','centro','estructura-organizativa-dipgra',1,1,
  'unidad:a','Centro A',false,'2024-01-01','2025-01-01+00','fuente:prueba','acto:inicio',repeat('a',64)),
 ('abcdefab-cdef-4abc-8def-abcdefabcdef',2,'org:tempo','unidad:b','centro','estructura-organizativa-dipgra',1,2,
  'unidad:b','Centro B',false,'2025-01-01','2026-01-01+00','fuente:prueba','acto:traslado',repeat('b',64)),
 ('abcdefab-cdef-4abc-8def-abcdefabcdef',3,'org:tempo','unidad:a','centro','estructura-organizativa-dipgra',1,3,
  'unidad:a','Centro A rectificado',false,'2024-01-01','2026-06-01+00','fuente:prueba','acto:rectificacion',repeat('c',64)),
 ('abcdefab-cdef-4abc-8def-abcdefabcdef',4,'org:tempo','unidad:a','centro','estructura-organizativa-dipgra',1,4,
  'unidad:a','Centro A retirado',true,'2024-01-01','2026-09-01+00','fuente:prueba','acto:retirada',repeat('d',64));
RESET ROLE;
SET SESSION AUTHORIZATION vec_prueba_personal;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $casos$
DECLARE j integer; unidad text; fecha text; conocimiento text; esperado integer;
 m text; canon text; h text; d jsonb; c jsonb; r jsonb; filas jsonb;
BEGIN
 FOR j IN 1..5 LOOP
  CASE j
   WHEN 1 THEN unidad:='unidad:a'; fecha:='2024-06-01'; conocimiento:='2026-03-01T00:00:00.000000Z'; esperado:=1;
   WHEN 2 THEN unidad:='unidad:a'; fecha:='2025-06-01'; conocimiento:='2026-03-01T00:00:00.000000Z'; esperado:=0;
   WHEN 3 THEN unidad:='unidad:b'; fecha:='2025-06-01'; conocimiento:='2026-03-01T00:00:00.000000Z'; esperado:=2;
   WHEN 4 THEN unidad:='unidad:a'; fecha:='2024-06-01'; conocimiento:='2026-07-01T00:00:00.000000Z'; esperado:=3;
   WHEN 5 THEN unidad:='unidad:a'; fecha:='2024-06-01'; conocimiento:='2026-09-20T00:00:00.000000Z'; esperado:=0;
  END CASE;
  m:=jsonb_build_object('esquema','vec.personal.organizacion-historica.v1',
   'organismo_ref','org:tempo','unidad_clave',unidad,'vigente_en',fecha,
   'conocido_en',conocimiento,'version_rpt_ref','','version_plantilla_ref','',
   'limite',100,'cursor','','actor_ref','actor:uno','contexto_actor_ref','contexto:uno',
   'contexto_version',1,'persona_version',1,'perfil_ref','perfil:uno','perfil_version',1)::text;
  canon:='{"ambitos":{"organismo_ref":"org:tempo","unidad_clave":'||to_jsonb(unidad)::text||
   '},"atributos":{"conocido_en":'||to_jsonb(conocimiento)::text||
   ',"cursor":"sin_seleccion","limite":"100","material_sha256":"'||
    encode(sha256(convert_to(m,'UTF8')),'hex')||'","version_plantilla_ref":"sin_seleccion","version_rpt_ref":"sin_seleccion","vigente_en":'||
    to_jsonb(fecha)::text||'}}';
  h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
  d:=jsonb_build_object('decision_ref','decision:temporal:'||j,'principal_id','actor:uno',
   'perfil_activo_ref','perfil:uno','concedida',true,'valida_hasta','2099-01-01T00:00:00.000000Z',
   'accion','personal.organizacion_historica.consultar','modulo_id','personal',
   'tipo_recurso','organizacion_historica','finalidad','consultar_organizacion_historica',
   'recurso_ref','org:tempo','campos_permitidos',
   '["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb,
   'obligaciones','[]'::jsonb,'contexto_recurso_huella_sha256',h);
  c:=jsonb_build_object('audiencia_consumo','vec_personal.organizacion_historica.consultar.v1',
   'operacion','personal.organizacion_historica.consultar','efecto_ref','org:tempo','huella_efecto_sha256',h);
  r:=vec_personal.consultar_organizacion_historica_v1(m,convert_to(c::text,'UTF8'),
   convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
  filas:=r #> '{pagina,unidades}';
  IF esperado=0 THEN
   IF jsonb_array_length(filas)<>0 OR r #>> '{pagina,cobertura,unidades}'<>'sin_datos' THEN
    RAISE EXCEPTION 'corte temporal % resucitó hecho: %',j,r;
   END IF;
  ELSE
   IF jsonb_array_length(filas)<>1 OR filas #>> '{0,traza,version}'<>esperado::text THEN
    RAISE EXCEPTION 'corte temporal % perdió revisión %: %',j,esperado,r;
   END IF;
   IF j=1 AND filas #>> '{0,traza,conocido_hasta}' IS NOT NULL THEN
    RAISE EXCEPTION 'consulta as-of filtró rectificación futura: %',r;
   END IF;
   IF j=3 AND filas #>> '{0,traza,conocido_hasta}' IS NOT NULL THEN
    RAISE EXCEPTION 'traslado 2025 cerrado por rectificación de 2024: %',r;
   END IF;
  END IF;
 END LOOP;
END $casos$;
COMMIT;
RESET SESSION AUTHORIZATION;
