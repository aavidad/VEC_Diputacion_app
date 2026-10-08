\set ON_ERROR_STOP on
-- Fixture sintética B96: ejecutar sólo en PG18 desechable con BC9/CC11 y
-- fachadas V3 de prueba que devuelvan una decisión nueva. No acredita V3 real.
\if :{?B96_DISPOSABLE_CLONE}
\else
\echo 'B96: prueba restringida a clon desechable (-v B96_DISPOSABLE_CLONE=1)'
\quit
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='30s';
DO $datos$
DECLARE
 cat_doc text:='{"id":"bolsa.categorias.inscripcion","version":1}';
 pol_doc text:='{"id":"bolsa.politica.inscripcion","version":1}';
 cat_sha text; pol_sha text; material bytea; canon jsonb;
BEGIN
 cat_sha:=encode(sha256(convert_to(cat_doc,'UTF8')),'hex');
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 INSERT INTO vec_catalogos_configurables.publicacion(
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES('bolsa.categorias.inscripcion',1,cat_sha,cat_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:b96:cat:a','aprobacion:b96:cat:b',
  'actor:b96','decision:b96:cat','recibo:b96:cat'),
 ('bolsa.politica.inscripcion',1,pol_sha,pol_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:b96:pol:a','aprobacion:b96:pol:b',
  'actor:b96','decision:b96:pol','recibo:b96:pol');
 INSERT INTO vec_catalogos_configurables.entrada_publicada(
  catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
 ('bolsa.categorias.inscripcion',1,cat_sha,'cat.alpha','Alfa ES',
  '{"etiquetas":{"es":"Alfa ES","en":"Alpha EN"}}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.con.pendientes','Política',
  '{"valor":true,"canal":"externa_personal"}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'requisito.pendiente','Pendiente ES',
  '{"etiquetas":{"es":"Pendiente ES","en":"Pending EN"}}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'requisito.cumple','Cumple ES',
  '{"etiquetas":{"es":"Cumple ES","en":"Meets EN"}}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'solicitud.existente','Ya solicitada ES',
  '{"etiquetas":{"es":"Ya solicitada ES","en":"Already applied EN"}}'),
 ('bolsa.politica.inscripcion',1,pol_sha,'presentacion.no.disponible','No disponible ES',
  '{"etiquetas":{"es":"No disponible ES","en":"Unavailable EN"}}');
 canon:=jsonb_build_object(
  'id','proceso:bolsa:inscripcion-b96','secuencia',1,
  'estado_gobierno','publicada','publicada_en',clock_timestamp()-interval '1 day',
  'aprobacion_publicacion',jsonb_build_object(
    'convocatoria_ref','proceso:bolsa:inscripcion-b96#1','ref','aprobacion:b96'),
  'comprobacion_dependencias',jsonb_build_object(
    'convocatoria_ref','proceso:bolsa:inscripcion-b96#1','ref','dependencias:b96'),
  'contenido',jsonb_build_object(
    'identificador_publico','inscripcion-b96','titulo','Convocatoria sintética',
    'tipo','bolsa',
    'resumen','Prueba sintética de inscripción',
    'categorias',jsonb_build_array('cat.alpha'),
    'catalogo_categorias',jsonb_build_object(
      'catalogo_id','bolsa.categorias.inscripcion','catalogo_version',1,
      'catalogo_huella_sha256',cat_sha),
    'plazos',jsonb_build_array(jsonb_build_object(
      'referencia','plazo.b96','tipo','inscripcion',
      'abre_en',clock_timestamp()-interval '1 hour',
      'cierra_en',clock_timestamp()+interval '1 hour')),
    'requisitos',jsonb_build_array(jsonb_build_object(
      'referencia','identidad_certificada','orden',1,
      'descripcion','Identidad acreditada por certificado','obligatorio',true))),
  'configuracion',jsonb_build_object(
    'catalogos',jsonb_build_object('id','bolsa.politica.inscripcion',
      'version',1,'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo.b96','version',1,
      'huella_contenido_sha256',repeat('a',64)),
    'documentos',jsonb_build_array(jsonb_build_object(
      'rol','bases','publicacion_ref','bases.b96'))));
 material:=convert_to(canon::text,'UTF8');
 INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
  convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
 VALUES('proceso:bolsa:inscripcion-b96',1,'proceso:bolsa:inscripcion-b96#1',
  'publicada',material,encode(sha256(material),'hex'),clock_timestamp());
END $datos$;

SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $casos$
DECLARE
 ref_conv text:='cv1_'||encode(sha256(convert_to(
   'proceso:bolsa:inscripcion-b96','UTF8')),'hex')||'_v1';
 persona text; perfil text:='prf_'||repeat('p',22);
 cuenta text:='cta_'||repeat('c',22);
 material text; contenido_sha text; recurso text; recurso_sha text;
 solicitud_ref text; cap bytea; dec bytea; contexto bytea; captura jsonb;
 recibo jsonb; recibo_primero text; ref_primera text; caso integer;
BEGIN
 IF NOT has_function_privilege(current_user,
  'vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'EXECUTE') THEN RAISE EXCEPTION 'B96 fixture: ACL externa ausente'; END IF;
 FOR caso IN 1..4 LOOP
  persona:=CASE WHEN caso=4 THEN 'per_'||repeat('b',22) ELSE 'per_'||repeat('a',22) END;
  material:='{"esquema":"vec.bolsa.inscripcion.presentar.v1",'
    ||'"convocatoria_ref":'||to_json(ref_conv)::text||','
    ||'"categoria_ref":"cat.alpha","catalogo_version":1,'
    ||'"clave_idempotencia":"ClavePruebaB96001","declaraciones":'
    ||CASE WHEN caso=3 THEN '[{"requisito_codigo":"identidad_certificada"}]'
      ELSE '[]' END||'}';
  contenido_sha:=encode(sha256(convert_to(material,'UTF8')),'hex');
  solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
   persona||chr(31)||ref_conv||chr(31)||'cat.alpha','UTF8')),'hex');
  recurso:='{"ambitos":{"categoria_ref":"cat.alpha","convocatoria_ref":'
   ||to_json(ref_conv)::text||'},"atributos":{"material_sha256":"'||contenido_sha||'"}}';
  recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
  cap:=convert_to(jsonb_build_object(
   'operacion','bolsa.inscripcion.presentar',
   'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.presentar.v1',
   'efecto_ref',solicitud_ref,'huella_efecto_sha256',recurso_sha)::text,'UTF8');
  dec:=convert_to(jsonb_build_object(
   'decision_ref','decision:b96:'||caso,'principal_id',persona,
   'perfil_activo_ref',perfil,'correlacion_ref','correlacion_'||repeat('a',32),
   'concedida',true,'accion','bolsa.inscripcion.presentar','modulo_id','bolsa',
   'tipo_recurso','inscripcion_convocatoria','finalidad','presentar_inscripcion',
   'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'))::text,'UTF8');
  contexto:=convert_to(jsonb_build_object('principal_ref',persona,
   'perfil_activo_ref',perfil,'cuenta_ref',cuenta,
   'metodo','certificado','garantia','alto','contexto_actor_ref','vca_'||repeat('v',22),
   'contexto_version',1)::text,'UTF8');
  captura:=jsonb_build_object('persona_ref',persona,'perfil_ref',perfil,
   'cuenta_ref',cuenta,'canal','externa_personal','idioma','es');
  IF caso=3 THEN
   BEGIN
    PERFORM vec_bolsa_llamamientos.solicitar_inscripcion_v1(
     material,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
     convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
     convert_to(repeat('a',44),'UTF8'));
    RAISE EXCEPTION 'B96 fixture: material conflictivo aceptado';
   EXCEPTION WHEN SQLSTATE 'B9603' THEN NULL;
   END;
  ELSE
   recibo:=vec_bolsa_llamamientos.solicitar_inscripcion_v1(
    material,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
    convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
    convert_to(repeat('a',44),'UTF8'));
   IF caso=1 THEN
    recibo_primero:=recibo->>'recibo_ref';ref_primera:=recibo->>'solicitud_ref';
    IF recibo->>'repetida'<>'false' OR recibo->>'estado'<>'pendiente'
     OR coalesce(recibo->>'declaracion_ref','')=''
    THEN RAISE EXCEPTION 'B96 fixture: primer recibo incompleto %',recibo; END IF;
   ELSIF caso=2 THEN
    IF recibo->>'repetida'<>'true' OR recibo->>'recibo_ref'<>recibo_primero
     OR recibo->>'solicitud_ref'<>ref_primera
    THEN RAISE EXCEPTION 'B96 fixture: replay divergente %',recibo; END IF;
   ELSE
    IF recibo->>'recibo_ref'=recibo_primero OR recibo->>'solicitud_ref'=ref_primera
    THEN RAISE EXCEPTION 'B96 fixture: dos personas comparten recibo %',recibo; END IF;
   END IF;
  END IF;
 END LOOP;
END $casos$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $revision$
DECLARE
 ref_conv text:='cv1_'||encode(sha256(convert_to(
   'proceso:bolsa:inscripcion-b96','UTF8')),'hex')||'_v1';
 solicitud_ref text; actor text:='per_'||repeat('r',22);
 perfil text:='prf_'||repeat('r',22); cuenta text:='cta_'||repeat('r',22);
 material text; contenido_sha text; recurso text; recurso_sha text;
 cap bytea; dec bytea; contexto bytea; captura jsonb; recibo jsonb;
 recibo_primero text; caso integer;
 actor_caso text;perfil_caso text;cuenta_caso text;
 decision_invalida jsonb; material_invalido text;
BEGIN
 solicitud_ref:='solicitud_inscripcion_'||encode(sha256(convert_to(
  'per_'||repeat('a',22)||chr(31)||ref_conv||chr(31)||'cat.alpha','UTF8')),'hex');
 FOR caso IN 1..5 LOOP
  actor_caso:=CASE WHEN caso=3 THEN 'per_'||repeat('a',22)
                   WHEN caso=4 THEN 'per_'||repeat('s',22) ELSE actor END;
  perfil_caso:=CASE WHEN caso=5 THEN 'prf_'||repeat('s',22) ELSE perfil END;
  cuenta_caso:=CASE WHEN caso=5 THEN 'cta_'||repeat('s',22) ELSE cuenta END;
  material:='{"esquema":"vec.bolsa.inscripcion.decidir.v1",'
   ||'"solicitud_ref":'||to_json(solicitud_ref)::text||','
   ||'"decision":"admitir","motivo_codigo":"",'
   ||'"version_esperada":1,"clave_idempotencia":"ClaveRevisionB96001"}';
  contenido_sha:=encode(sha256(convert_to(material,'UTF8')),'hex');
  recurso:='{"ambitos":{"solicitud_ref":'||to_json(solicitud_ref)::text
   ||'},"atributos":{"material_sha256":"'||contenido_sha||'"}}';
  recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
  cap:=convert_to(jsonb_build_object('operacion','bolsa.inscripcion.rrhh.decidir',
   'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.revisar.v1',
   'efecto_ref',solicitud_ref,'huella_efecto_sha256',recurso_sha)::text,'UTF8');
  dec:=convert_to(jsonb_build_object('decision_ref','decision:b96:revision:'||caso,
   'principal_id',actor_caso,
   'perfil_activo_ref',perfil_caso,'concedida',true,
   'accion','bolsa.inscripcion.rrhh.decidir','modulo_id','bolsa',
   'tipo_recurso','solicitud_inscripcion','finalidad','revisar_inscripcion',
   'recurso_ref',solicitud_ref,'contexto_recurso_huella_sha256',recurso_sha,
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
  contexto:=convert_to(jsonb_build_object('principal_ref',actor_caso,
   'perfil_activo_ref',perfil_caso,'cuenta_ref',cuenta_caso,'metodo','certificado',
   'garantia','alto')::text,'UTF8');
  captura:=jsonb_build_object('persona_ref',actor_caso,
   'perfil_ref',perfil_caso,'cuenta_ref',cuenta_caso,
   'canal','interna_corporativa','idioma','es');
  IF caso=3 THEN
   BEGIN
    PERFORM vec_bolsa_llamamientos.revisar_inscripcion_v1(
     material,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
     convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
     convert_to(repeat('a',44),'UTF8'));
    RAISE EXCEPTION 'B96 fixture: solicitante revisó su solicitud';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
  ELSIF caso IN(4,5) THEN
   BEGIN
    PERFORM vec_bolsa_llamamientos.revisar_inscripcion_v1(
     material,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
     convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
     convert_to(repeat('a',44),'UTF8'));
    RAISE EXCEPTION 'B96 fixture: otro revisor recuperó recibo ajeno caso %',caso;
   EXCEPTION WHEN SQLSTATE 'B9604' THEN NULL;
   END;
  ELSE
   recibo:=vec_bolsa_llamamientos.revisar_inscripcion_v1(
    material,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
    convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
    convert_to(repeat('a',44),'UTF8'));
   IF caso=1 THEN
    recibo_primero:=recibo->>'recibo_ref';
    IF recibo->>'estado'<>'admitida_a_convocatoria' OR recibo->>'version'<>'2'
     OR recibo->>'repetida'<>'false'
    THEN RAISE EXCEPTION 'B96 fixture: admisión incompleta %',recibo; END IF;
   ELSE
    IF recibo->>'repetida'<>'true' OR recibo->>'recibo_ref'<>recibo_primero
     OR recibo->>'version'<>'2'
    THEN RAISE EXCEPTION 'B96 fixture: replay decisión divergente %',recibo; END IF;
   END IF;
  END IF;
 END LOOP;
 FOR decision_invalida IN SELECT valor FROM jsonb_array_elements('[null,123,{}]'::jsonb) AS x(valor) LOOP
  material_invalido:=jsonb_set(material::jsonb,'{decision}',decision_invalida)::text;
  BEGIN
   PERFORM vec_bolsa_llamamientos.revisar_inscripcion_v1(
    material_invalido,captura,cap,dec,convert_to('{}','UTF8'),contexto,1,1,
    convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
    convert_to(repeat('a',44),'UTF8'));
   RAISE EXCEPTION 'B96 fixture: decisión JSON inválida aceptada %',decision_invalida;
  EXCEPTION WHEN SQLSTATE 'B9605' THEN NULL;
  END;
 END LOOP;
END $revision$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_lector;
DO $lecturas$
DECLARE
 ref_conv text:='cv1_'||encode(sha256(convert_to(
   'proceso:bolsa:inscripcion-b96','UTF8')),'hex')||'_v1';
 ref_solicitud text; persona text; perfil text:='prf_'||repeat('p',22);
 cuenta text:='cta_'||repeat('c',22); accion text; finalidad text;
 recurso text; selector jsonb; filtro jsonb; canon_recurso text;
 prefijo text; contexto bytea; vinculo bytea; captura jsonb;
 resultado jsonb; captura_invalida jsonb; caso integer; canal text:='externa_personal';
BEGIN
 ref_solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
  'per_'||repeat('a',22)||chr(31)||ref_conv||chr(31)||'cat.alpha','UTF8')),'hex');
 FOR caso IN 1..5 LOOP
  persona:=CASE WHEN caso=4 THEN 'per_'||repeat('b',22) ELSE 'per_'||repeat('a',22) END;
  prefijo:=NULL;
  IF caso IN(1,4) THEN
   accion:='bolsa.inscripcion.propia.consultar';
   finalidad:='consulta_inscripcion_propia';
   selector:=jsonb_build_object('solicitud_ref',ref_solicitud);
   filtro:=jsonb_build_object('estado','','convocatoria_ref','','limite',0,'cursor','');
   recurso:=ref_solicitud;
  ELSIF caso=2 THEN
   accion:='bolsa.inscripcion.propias.listar';
   finalidad:='consulta_inscripcion_propia';
   selector:=jsonb_build_object('estado','','convocatoria_ref','','limite',10,'cursor','');
   filtro:=selector;
   prefijo:='inscripciones_propias_';
  ELSIF caso=3 THEN
   accion:='bolsa.inscripcion.convocatorias.listar';
   finalidad:='consulta_convocatoria_abierta';
   selector:=jsonb_build_object('limite',10,'cursor','');
   filtro:=jsonb_build_object('estado','','convocatoria_ref','','limite',10,'cursor','');
   prefijo:='inscripciones_abiertas_';
  ELSE
   accion:='bolsa.inscripcion.convocatoria.consultar';
   finalidad:='consulta_convocatoria_abierta';
   selector:=jsonb_build_object('convocatoria_ref',ref_conv);
   filtro:=jsonb_build_object('estado','','convocatoria_ref','','limite',0,'cursor','');
   recurso:=ref_conv;
  END IF;
  IF prefijo IS NOT NULL THEN
   canon_recurso:='{"accion":'||to_json(accion)::text||
    ',"persona_ref":'||to_json(persona)::text||
    ',"idioma":"es","estado":"","convocatoria_ref":"",'
    ||'"limite":10,"cursor":"","ref":""}';
   recurso:=prefijo||encode(sha256(convert_to(canon_recurso,'UTF8')),'hex');
  END IF;
  contexto:=convert_to(jsonb_build_object('principal_ref',persona,
   'perfil_activo_ref',perfil,'cuenta_ref',cuenta,
   'metodo','certificado','garantia','alto')::text,'UTF8');
  vinculo:=convert_to(jsonb_build_object('principal_id',persona,
   'perfil_activo_ref',perfil,'cuenta_ref',cuenta,
   'sesion_ref','ses_'||repeat('s',22),'autenticacion_ref','aut_'||repeat('a',22),
   'superficie',canal,'registro_contexto_ref','registro.b96',
   'contexto_actor_huella_sha256',encode(sha256(contexto),'hex'),
   'manifiesto_procedencia_huella_sha256',repeat('b',64),
   'autenticacion_huella_sha256',repeat('c',64))::text,'UTF8');
  captura:=jsonb_build_object('intento_ref','lectura_'||lpad(caso::text,32,'0'),
   'persona_ref',persona,'perfil_ref',perfil,'cuenta_ref',cuenta,
   'sesion_ref','ses_'||repeat('s',22),'autenticacion_ref','aut_'||repeat('a',22),
   'canal',canal,'idioma','es','accion',accion,'recurso_ref',recurso,
   'finalidad',finalidad,'correlacion_ref','correlacion_'||repeat('a',32),
   'revision_permisos',1,'emitida_en',clock_timestamp()-interval '1 minute',
   'valida_hasta',clock_timestamp()+interval '1 hour','filtro',filtro);
  resultado:=vec_bolsa_llamamientos.consultar_inscripcion_v1(
   accion,selector,contexto,vinculo,captura);
  IF caso=1 THEN
   IF resultado->>'resultado'<>'obtenida'
    OR resultado#>>'{proyeccion,estado}'<>'admitida_a_convocatoria'
    OR resultado#>>'{proyeccion,categoria}'<>'Alfa ES'
    OR coalesce(resultado#>>'{proyeccion,declaracion_ref}','')=''
    OR resultado#>>'{proyeccion,requisitos,0,motivo_etiqueta}'<>'Cumple ES'
   THEN RAISE EXCEPTION 'B96 fixture: detalle propio incompleto %',resultado; END IF;
  ELSIF caso=2 THEN
   IF resultado->>'resultado'<>'obtenida'
    OR resultado#>>'{proyeccion,total}'<>'1'
    OR resultado#>>'{proyeccion,solicitudes,0,solicitud_ref}'<>ref_solicitud
    OR coalesce(resultado#>>'{proyeccion,solicitudes,0,declaracion_ref}','')=''
   THEN RAISE EXCEPTION 'B96 fixture: página propia incompleta %',resultado; END IF;
   BEGIN
    PERFORM vec_bolsa_llamamientos.consultar_inscripcion_v1(accion,
     jsonb_set(selector,'{estado}','"rechazada"'::jsonb),contexto,vinculo,captura);
    RAISE EXCEPTION 'B96 fixture: filtro cambiado con captura antigua';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
   BEGIN
    PERFORM vec_bolsa_llamamientos.consultar_inscripcion_v1(accion,
     selector,contexto,vinculo,jsonb_set(captura,'{idioma}','"en"'::jsonb));
    RAISE EXCEPTION 'B96 fixture: idioma cambiado con recurso antiguo';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
   FOR captura_invalida IN SELECT valor FROM jsonb_array_elements(jsonb_build_array(
     captura-'emitida_en',
     jsonb_set(captura,'{emitida_en}','null'::jsonb),
     jsonb_set(captura,'{emitida_en}',to_jsonb(clock_timestamp()+interval '1 hour')),
     jsonb_set(captura,'{valida_hasta}',to_jsonb(clock_timestamp()-interval '1 hour'))
   )) AS x(valor) LOOP
    BEGIN
     PERFORM vec_bolsa_llamamientos.consultar_inscripcion_v1(accion,
      selector,contexto,vinculo,captura_invalida);
     RAISE EXCEPTION 'B96 fixture: ventana inválida aceptada %',captura_invalida;
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
   END LOOP;
  ELSIF caso=3 THEN
   IF resultado->>'resultado'<>'obtenida'
    OR resultado#>>'{proyeccion,total}'<>'1'
    OR resultado#>>'{proyeccion,convocatorias,0,puede_iniciar}'<>'false'
    OR resultado#>>'{proyeccion,convocatorias,0,numero_categorias}'<>'1'
    OR (resultado#>'{proyeccion,convocatorias,0}') ? 'categorias'
    OR (resultado#>'{proyeccion,convocatorias,0}') ? 'categorias_resumen'
    OR (resultado#>'{proyeccion,convocatorias,0}') ? 'requisitos'
    OR resultado#>>'{proyeccion,convocatorias,0,impedimento_etiqueta}'<>'Ya solicitada ES'
   THEN RAISE EXCEPTION 'B96 fixture: abierta propia incompleta %',resultado; END IF;
  ELSIF caso=4 THEN
   IF resultado->>'resultado'<>'no_encontrada'
    OR resultado->'proyeccion' IS DISTINCT FROM 'null'::jsonb
    OR coalesce(resultado->>'auditoria_ref','')=''
   THEN RAISE EXCEPTION 'B96 fixture: ajena filtrada incorrectamente %',resultado; END IF;
  ELSE
   IF resultado->>'resultado'<>'obtenida'
    OR resultado#>>'{proyeccion,numero_categorias}'<>'1'
    OR jsonb_array_length(resultado#>'{proyeccion,categorias}')<>1
    OR jsonb_array_length(resultado#>'{proyeccion,requisitos}')<>1
   THEN RAISE EXCEPTION 'B96 fixture: detalle abierto incompleto %',resultado; END IF;
  END IF;
 END LOOP;
END $lecturas$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_empleado_lector;
DO $empleado$
DECLARE ref_conv text:='cv1_'||encode(sha256(convert_to(
 'proceso:bolsa:inscripcion-b96','UTF8')),'hex')||'_v1';
 ref_solicitud text; persona text:='per_'||repeat('a',22);
 perfil text:='prf_'||repeat('p',22);cuenta text:='cta_'||repeat('c',22);
 contexto bytea;vinculo bytea;captura jsonb;filtro jsonb;selector jsonb;respuesta jsonb;
BEGIN
 ref_solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
  persona||chr(31)||ref_conv||chr(31)||'cat.alpha','UTF8')),'hex');
 selector:=jsonb_build_object('solicitud_ref',ref_solicitud);
 filtro:=jsonb_build_object('estado','','convocatoria_ref','','limite',0,'cursor','');
 contexto:=convert_to(jsonb_build_object('principal_ref',persona,'perfil_activo_ref',perfil,
  'cuenta_ref',cuenta,'metodo','certificado','garantia','alto')::text,'UTF8');
 vinculo:=convert_to(jsonb_build_object('principal_id',persona,'perfil_activo_ref',perfil,
  'cuenta_ref',cuenta,'sesion_ref','ses_'||repeat('s',22),
  'autenticacion_ref','aut_'||repeat('a',22),'superficie','interna_corporativa',
  'registro_contexto_ref','registro.b96',
  'contexto_actor_huella_sha256',encode(sha256(contexto),'hex'),
  'manifiesto_procedencia_huella_sha256',repeat('b',64),
  'autenticacion_huella_sha256',repeat('c',64))::text,'UTF8');
 captura:=jsonb_build_object('intento_ref','lectura_'||repeat('e',32),
  'persona_ref',persona,'perfil_ref',perfil,'cuenta_ref',cuenta,
  'sesion_ref','ses_'||repeat('s',22),'autenticacion_ref','aut_'||repeat('a',22),
  'canal','interna_corporativa','idioma','es',
  'accion','bolsa.inscripcion.propia.consultar','recurso_ref',ref_solicitud,
  'finalidad','consulta_inscripcion_propia','correlacion_ref','correlacion_'||repeat('a',32),
  'revision_permisos',1,'emitida_en',clock_timestamp()-interval '1 minute',
  'valida_hasta',clock_timestamp()+interval '1 hour','filtro',filtro);
 respuesta:=vec_bolsa_llamamientos.consultar_inscripcion_v1(
  'bolsa.inscripcion.propia.consultar',selector,contexto,vinculo,captura);
 IF respuesta->>'resultado'<>'obtenida' OR respuesta#>>'{proyeccion,solicitud_ref}'<>ref_solicitud
 THEN RAISE EXCEPTION 'B96 fixture: lectura empleada no obtenida %',respuesta; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_inscripcion_v1(
   'bolsa.inscripcion.rrhh.consultar',selector,contexto,vinculo,
   captura||jsonb_build_object('accion','bolsa.inscripcion.rrhh.consultar',
    'finalidad','consulta_inscripcion_rrhh'));
  RAISE EXCEPTION 'B96 fixture: LOGIN empleado leyó RRHH';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $empleado$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_bolsa_inscripciones_rrhh_lector;
DO $rrhh$
DECLARE ref_conv text:='cv1_'||encode(sha256(convert_to(
 'proceso:bolsa:inscripcion-b96','UTF8')),'hex')||'_v1';
 ref_solicitud text; persona text:='per_'||repeat('r',22);
 perfil text:='prf_'||repeat('r',22);cuenta text:='cta_'||repeat('r',22);
 contexto bytea;vinculo bytea;captura jsonb;filtro jsonb;selector jsonb;respuesta jsonb;
BEGIN
 ref_solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
  'per_'||repeat('a',22)||chr(31)||ref_conv||chr(31)||'cat.alpha','UTF8')),'hex');
 selector:=jsonb_build_object('solicitud_ref',ref_solicitud);
 filtro:=jsonb_build_object('estado','','convocatoria_ref','','limite',0,'cursor','');
 contexto:=convert_to(jsonb_build_object('principal_ref',persona,'perfil_activo_ref',perfil,
  'cuenta_ref',cuenta,'metodo','certificado','garantia','alto')::text,'UTF8');
 vinculo:=convert_to(jsonb_build_object('principal_id',persona,'perfil_activo_ref',perfil,
  'cuenta_ref',cuenta,'sesion_ref','ses_'||repeat('s',22),
  'autenticacion_ref','aut_'||repeat('a',22),'superficie','interna_corporativa',
  'registro_contexto_ref','registro.b96',
  'contexto_actor_huella_sha256',encode(sha256(contexto),'hex'),
  'manifiesto_procedencia_huella_sha256',repeat('b',64),
  'autenticacion_huella_sha256',repeat('c',64))::text,'UTF8');
 captura:=jsonb_build_object('intento_ref','lectura_'||repeat('f',32),
  'persona_ref',persona,'perfil_ref',perfil,'cuenta_ref',cuenta,
  'sesion_ref','ses_'||repeat('s',22),'autenticacion_ref','aut_'||repeat('a',22),
  'canal','interna_corporativa','idioma','es',
  'accion','bolsa.inscripcion.rrhh.consultar','recurso_ref',ref_solicitud,
  'finalidad','consulta_inscripcion_rrhh','correlacion_ref','correlacion_'||repeat('a',32),
  'revision_permisos',1,'emitida_en',clock_timestamp()-interval '1 minute',
  'valida_hasta',clock_timestamp()+interval '1 hour','filtro',filtro);
 respuesta:=vec_bolsa_llamamientos.consultar_inscripcion_v1(
  'bolsa.inscripcion.rrhh.consultar',selector,contexto,vinculo,captura);
 IF respuesta->>'resultado'<>'obtenida' OR respuesta#>>'{proyeccion,solicitud_ref}'<>ref_solicitud
 THEN RAISE EXCEPTION 'B96 fixture: lectura RRHH no obtenida %',respuesta; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_inscripcion_v1(
   'bolsa.inscripcion.propia.consultar',selector,contexto,vinculo,
   captura||jsonb_build_object('accion','bolsa.inscripcion.propia.consultar',
    'finalidad','consulta_inscripcion_propia'));
  RAISE EXCEPTION 'B96 fixture: LOGIN RRHH leyó perfil propio';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $rrhh$;
ROLLBACK;
