\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000070:canon:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $precondicion$
BEGIN
 IF current_setting('server_encoding') <> 'UTF8' THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='codec CT requiere UTF8';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname=ANY(ARRAY['texto_json_incorporacion_go_v2','fecha_civil_incorporacion_go_v2','instante_incorporacion_go_v2','clave_instante_incorporacion_v2','bytes_incorporacion_v2','nodo_incorporacion_canonico_v2','campo_solicitud_incorporacion_v2','solicitud_personal_incorporacion_canonica_v2','material_incorporacion_ejercicio_canonico_v2','material_incorporacion_ejercicio_sha256_v2','contexto_incorporacion_ejercicio_canonico_v2','contexto_incorporacion_ejercicio_sha256_v2','recurso_incorporacion_ejercicio_v2'])) THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='codec CT ya existente o sobrecarga incompatible';
 END IF;
END $precondicion$;

-- Funciones privadas puras. Sin acceso a tablas/Personal/AD3 ni concesión.
-- Entrada: el DTO devuelto por MaterialCanonico(), no Datos() ni JSON browser.
CREATE FUNCTION vec_contratacion_temporal.texto_json_incorporacion_go_v2(valor text) RETURNS text
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE r text;
BEGIN
 r:=to_json(valor)::text;
 RETURN replace(replace(replace(replace(replace(r,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029');
END $$;

-- Calendario proléptico Go: 0000 es bisiesto. Los instantes CT/V3 restringen
-- después a 0001..9999/microsegundo. Personal opaco NO se convierte a PG date.
CREATE FUNCTION vec_contratacion_temporal.fecha_civil_incorporacion_go_v2(s text) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE a int; m int; d int; limite int;
BEGIN
 IF s !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN RETURN false; END IF;
 a:=substring(s,1,4)::int; m:=substring(s,6,2)::int; d:=substring(s,9,2)::int;
 IF m<1 OR m>12 OR d<1 THEN RETURN false; END IF;
 limite:=CASE m WHEN 2 THEN CASE WHEN a%4=0 AND (a%100<>0 OR a%400=0) THEN 29 ELSE 28 END
  WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31 END;
 RETURN d<=limite;
END $$;

CREATE FUNCTION vec_contratacion_temporal.instante_incorporacion_go_v2(s text, fijo boolean) RETURNS boolean
LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $$
BEGIN
 IF s !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$' THEN RETURN false; END IF;
 IF left(s,4)='0000' OR vec_contratacion_temporal.fecha_civil_incorporacion_go_v2(left(s,10)) IS NOT TRUE
 OR substring(s,12,2)::int>23 OR substring(s,15,2)::int>59 OR substring(s,18,2)::int>59 THEN RETURN false; END IF;
 IF fijo THEN
  RETURN s ~ '\.[0-9]{6}Z$' AND s<>'0001-01-01T00:00:00.000000Z';
 END IF;
 RETURN (s !~ '\.' OR s !~ '0Z$') AND s<>'0001-01-01T00:00:00Z';
END $$;

-- Orden temporal independiente de locale y de la precisión de timestamptz.
CREATE FUNCTION vec_contratacion_temporal.clave_instante_incorporacion_v2(s text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $$
 SELECT left(s,19)||'.'||rpad(CASE WHEN substring(s,20,1)='.' THEN substring(s,21,length(s)-21) ELSE '' END,6,'0')
$$;

CREATE FUNCTION vec_contratacion_temporal.bytes_incorporacion_v2(v jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE s text; b bytea;
BEGIN
 IF jsonb_typeof(v) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'bytes'; END IF;
 s:=v#>>'{}';
 IF length(s)=0 OR length(s)>87384 OR s !~ '^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$' THEN RAISE EXCEPTION 'bytes'; END IF;
 b:=decode(s,'base64');
 IF octet_length(b)=0 OR octet_length(b)>65536 OR replace(encode(b,'base64'),chr(10),'')<>s THEN RAISE EXCEPTION 'bytes'; END IF;
 RETURN b;
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='bytes canónicos CT inválidos';
END $$;

-- Gramática cerrada, orden de campos idéntico al struct Go, sin jsonb::text
-- como preimagen. Los únicos casts escalares son enteros ya validados.
CREATE FUNCTION vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v jsonb, tipo text) RETURNS text
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE claves text[]; tipos text[]; claves_actuales text[]; esperadas text[];
 salida text; s text; i int; n numeric; item jsonb; previo text[]; orden text[]; vistos text[]:=ARRAY[]::text[];
BEGIN
 IF v IS NULL OR v='null'::jsonb OR tipo IS NULL THEN RAISE EXCEPTION 'nodo'; END IF;
 CASE tipo
 WHEN 'material' THEN claves:=ARRAY['Esquema','Confirmacion','VersionActualExpediente','Preparacion','SolicitudContexto','Vinculo','ContextoCanonico','ProcedenciaCanonica','MotivoV3','CorrelacionV3','Personal','EjercicioSintetico','FirmaOficial','EficaciaAdministrativa'];
   tipos:=ARRAY['=vec.contratacion-temporal.incorporacion.ejercicio.material.v2','confirmacion','seguro1','preparacion','solicitud_contexto','vinculo','bytes','bytes','motivo','correlacion','personal','true','false','false'];
 WHEN 'confirmacion' THEN claves:=ARRAY['SolicitudPersonal','ResultadoPersonal','VersionSeguimientoEsperada','PeriodoIncorporacion','MotivoClave','Documentos'];
   tipos:=ARRAY['solicitud','resultado','seguimiento_version','periodo','clave','documentos'];
 WHEN 'solicitud' THEN claves:=ARRAY['esquema','contrato_version','solicitud_ref','expediente_ref','version_expediente','capacidad_ref','correlacion_ref','idempotencia_ref','fuente_rpt','puesto_ref','plaza_ref'];
   tipos:=ARRAY['=vec.contratacion-temporal.personal-rpt.alta.v1','uno','ref','ref','seguro1','ref','ref','ref','fuente','ref','ref'];
 WHEN 'fuente' THEN claves:=ARRAY['referencia','version','huella_sha256'];
   tipos:=ARRAY['ref','seguro1','sha_no_cero'];
 WHEN 'resultado' THEN claves:=ARRAY['esquema','contrato_version','resultado_ref','recibo_ref','solicitud_ref','correlacion_ref','idempotencia_ref','huella_solicitud_sha256','estado','relacion_ref','ocupacion_ref','motivo_rechazo'];
   tipos:=ARRAY['=vec.contratacion-temporal.personal-rpt.alta.v1','uno','ref','ref','ref','ref','ref','sha','=confirmada','ref','ref','motivo_cero'];
 WHEN 'motivo_cero' THEN claves:=ARRAY['referencia','version','huella_sha256'];
   tipos:=ARRAY['=','cero','='];
 WHEN 'periodo' THEN claves:=ARRAY['desde','hasta'];
   tipos:=ARRAY['instante','instante'];
 WHEN 'documento' THEN claves:=ARRAY['tipo_clave','referencia'];
   tipos:=ARRAY['clave','ref'];
 WHEN 'preparacion' THEN claves:=ARRAY['OrganizacionRef','UnidadRef','ActorRef','CorrelacionRef'];
   tipos:=ARRAY['ref','ref','ref','ref'];
 WHEN 'solicitud_contexto' THEN claves:=ARRAY['AutenticacionRef','SesionRef','PerfilRef'];
   tipos:=ARRAY['aut_','ses_','prf_'];
 WHEN 'motivo' THEN claves:=ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'];
   tipos:=ARRAY['clave_catalogo','int31','sha_no_cero','motivo_opaco'];
 WHEN 'personal' THEN claves:=ARRAY['Solicitud','Resultado','MaterialCanonico','MaterialSHA256','RegistradoEn','DecisionOriginalRef','AuditoriaRef','OutboxRef','EjercicioSintetico','FirmaOficial','EficaciaAdministrativa'];
   tipos:=ARRAY['solicitud','resultado','bytes','sha','instante','ref','ref','ref','true','false','false'];
 WHEN 'vinculo' THEN claves:=ARRAY['esquema','bloque_version','autenticacion_ref','autenticacion_huella_sha256','asercion_ref','sesion_ref','control_sesion_ref','control_sesion_revision','control_sesion_huella_sha256','cuenta_ref','cuenta_ordinaria_ref','principal_id','perfil_activo_ref','cuenta_privilegiada','superficie','metodo_observado','garantia_observada','politica_garantia_ref','politica_garantia_huella_sha256','autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en','registro_contexto_ref','contexto_actor_esquema','contexto_actor_ref','contexto_actor_version','contexto_actor_cuenta_version','contexto_actor_huella_sha256','manifiesto_procedencia_huella_sha256','autoridad_efectiva'];
   tipos:=ARRAY['=vec.autenticacion-actor.vinculo.v2.contexto-registrado','dos','aut_','sha','ase_','ses_','cse_','u64','sha','cta_','cta_','per_','prf_','bool','superficie','metodo','=alto','pga_','sha','instante','instante','instante','instante','rca_','=vec.contexto-actor.vinculado.v2','vca_','u64','u64','sha','sha','=autoridad_maestra_acreditada'];
 WHEN 'contexto' THEN claves:=ARRAY['esquema','principal_ref','metodo','garantia','perfil_activo_ref','persona_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','persona_version','perfil_version','estado','vigente_desde','vigente_hasta','resuelto_en','vinculos'];
   tipos:=ARRAY['=vec.contexto-actor.vinculado.v2','per_','metodo','=alto','prf_','per_','vca_','u64','cta_','u64','u64','u64','=activo','instante_fijo','instante_fijo','instante_fijo','vinculos_contexto'];
 WHEN 'vinculo_contexto' THEN claves:=ARRAY['vinculo_ref','version','tipo','referencia','estado','vigente_desde','vigente_hasta'];
   tipos:=ARRAY['vin_','u64','tipo_vinculo','ref_vinculo','estado_vinculo','instante_fijo','instante_fijo'];
 WHEN 'procedencia' THEN claves:=ARRAY['esquema','autoridad_efectiva','cuenta','persona','perfil','contexto','vinculos'];
   tipos:=ARRAY['=vec.contexto-actor.procedencia-manifiesto.v1','=autoridad_maestra_acreditada','procedencia_cuenta','procedencia_persona','procedencia_perfil','procedencia_contexto','vinculos_procedencia'];
 WHEN 'procedencia_cuenta' THEN claves:=ARRAY['cuenta_ref','version','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'];
   tipos:=ARRAY['cta_','u64','prc_','u64','sha','=autoridad_maestra_acreditada'];
 WHEN 'procedencia_persona' THEN claves:=ARRAY['persona_ref','version','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'];
   tipos:=ARRAY['per_','u64','prc_','u64','sha','=autoridad_maestra_acreditada'];
 WHEN 'procedencia_perfil' THEN claves:=ARRAY['perfil_ref','version','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'];
   tipos:=ARRAY['prf_','u64','prc_','u64','sha','=autoridad_maestra_acreditada'];
 WHEN 'procedencia_contexto' THEN claves:=ARRAY['vinculo_ref','version','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'];
   tipos:=ARRAY['vca_','u64','prc_','u64','sha','=autoridad_maestra_acreditada'];
 WHEN 'procedencia_vinculo' THEN claves:=ARRAY['vinculo_ref','version','tipo','referencia','procedencia_ref','procedencia_version','procedencia_huella_sha256','procedencia_autoridad'];
   tipos:=ARRAY['vin_','u64','tipo_vinculo','ref_vinculo','prc_','u64','sha','=autoridad_maestra_acreditada'];
 ELSE claves:=NULL;
 END CASE;
 IF claves IS NOT NULL THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'object' THEN RAISE EXCEPTION 'objeto'; END IF;
  SELECT array_agg(k COLLATE "C" ORDER BY k COLLATE "C") INTO claves_actuales FROM jsonb_object_keys(v) k;
  SELECT array_agg(k COLLATE "C" ORDER BY k COLLATE "C") INTO esperadas FROM unnest(claves) k;
  IF claves_actuales IS DISTINCT FROM esperadas THEN RAISE EXCEPTION 'claves'; END IF;
  salida:='{';
  FOR i IN 1..cardinality(claves) LOOP
   IF i>1 THEN salida:=salida||','; END IF;
   salida:=salida||vec_contratacion_temporal.texto_json_incorporacion_go_v2(claves[i])||':'||
    vec_contratacion_temporal.nodo_incorporacion_canonico_v2(v->claves[i],tipos[i]);
  END LOOP;
  RETURN salida||'}';
 END IF;
 IF tipo IN ('documentos','vinculos_contexto','vinculos_procedencia') THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'array' OR jsonb_array_length(v)>(CASE WHEN tipo='documentos' THEN 32 ELSE 128 END)
   OR (tipo='documentos' AND jsonb_array_length(v)=0) THEN RAISE EXCEPTION 'array'; END IF;
  salida:='['; i:=0;
  FOR item IN SELECT value FROM jsonb_array_elements(v) LOOP
   s:=vec_contratacion_temporal.nodo_incorporacion_canonico_v2(item,
     CASE tipo WHEN 'documentos' THEN 'documento' WHEN 'vinculos_contexto' THEN 'vinculo_contexto' ELSE 'procedencia_vinculo' END);
   IF tipo='documentos' THEN
    orden:=ARRAY[item->>'tipo_clave',item->>'referencia'];
    IF item->>'referencia'=ANY(vistos) THEN RAISE EXCEPTION 'duplicado'; END IF;
    vistos:=array_append(vistos,item->>'referencia');
   ELSE
    orden:=ARRAY[item->>'tipo',item->>'referencia',lpad(item->>'version',20,'0'),item->>'vinculo_ref'];
    IF item->>'tipo'=ANY(vistos) OR item->>'vinculo_ref'=ANY(vistos)
     OR (item->>'tipo'='empleado' AND item->>'referencia' !~ '^emp_')
     OR (item->>'tipo'='candidato' AND item->>'referencia' !~ '^can_') THEN RAISE EXCEPTION 'vínculo'; END IF;
    vistos:=vistos||ARRAY[item->>'tipo',item->>'vinculo_ref'];
   END IF;
   IF previo IS NOT NULL AND (orden COLLATE "C") <= (previo COLLATE "C") THEN RAISE EXCEPTION 'orden'; END IF;
   previo:=orden;
   IF i>0 THEN salida:=salida||','; END IF;
   salida:=salida||s; i:=i+1;
  END LOOP;
  RETURN salida||']';
 END IF;
 s:=v#>>'{}';
 IF tipo IN ('bool','true','false') THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'boolean' OR (tipo<>'bool' AND s<>tipo) THEN RAISE EXCEPTION 'boolean'; END IF;
  RETURN s;
 END IF;
 IF tipo IN ('seguro1','seguimiento_version','u64','int31','uno','dos','cero') THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR s !~ '^(0|[1-9][0-9]{0,19})$' THEN RAISE EXCEPTION 'entero'; END IF;
  n:=s::numeric;
  IF n>(CASE tipo WHEN 'u64' THEN 18446744073709551615 WHEN 'int31' THEN 2147483647
   WHEN 'seguimiento_version' THEN 9007199254740990 ELSE 9007199254740991 END)
   OR (tipo IN ('seguro1','u64','int31') AND n=0) OR (tipo='uno' AND n<>1) OR (tipo='dos' AND n<>2) OR (tipo='cero' AND n<>0)
   THEN RAISE EXCEPTION 'entero'; END IF;
  RETURN s;
 END IF;
 IF jsonb_typeof(v) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'texto'; END IF;
 CASE
 WHEN left(tipo,1)='=' THEN IF s<>substring(tipo,2) THEN RAISE EXCEPTION 'constante'; END IF;
 WHEN tipo='ref' THEN IF s !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'referencia'; END IF;
 WHEN tipo='clave' THEN IF s !~ '^[a-z][a-z0-9._-]{1,79}$' THEN RAISE EXCEPTION 'clave'; END IF;
 WHEN tipo='clave_catalogo' THEN IF s !~ '^[a-z][a-z0-9._-]{0,127}$' THEN RAISE EXCEPTION 'catálogo'; END IF;
 WHEN tipo IN ('sha','sha_no_cero') THEN
  IF s !~ '^[0-9a-f]{64}$' OR (tipo='sha_no_cero' AND s=repeat('0',64)) THEN RAISE EXCEPTION 'huella'; END IF;
 WHEN tipo='correlacion' THEN IF s !~ '^correlacion_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'correlación'; END IF;
 WHEN tipo='motivo_opaco' THEN IF s !~ '^motivo_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'motivo'; END IF;
 WHEN tipo IN ('aut_','ase_','ses_','cse_','cta_','per_','prf_','pga_','vca_','vin_','prc_') THEN
  IF s !~ ('^'||tipo||'[A-Za-z0-9_-]{22,128}$') THEN RAISE EXCEPTION 'token'; END IF;
 WHEN tipo='rca_' THEN IF s !~ '^rca_[A-Za-z0-9_-]{24,128}$' THEN RAISE EXCEPTION 'registro'; END IF;
 WHEN tipo='ref_vinculo' THEN IF s !~ '^(can_|emp_)[A-Za-z0-9_-]{22,128}$' THEN RAISE EXCEPTION 'vínculo'; END IF;
 WHEN tipo='tipo_vinculo' THEN IF s NOT IN ('candidato','empleado') THEN RAISE EXCEPTION 'tipo'; END IF;
 WHEN tipo='estado_vinculo' THEN IF s<>'activo' THEN RAISE EXCEPTION 'estado'; END IF;
 WHEN tipo='metodo' THEN IF s NOT IN ('certificado','dnie','sso','clave','kerberos_ad') THEN RAISE EXCEPTION 'método'; END IF;
 WHEN tipo='superficie' THEN IF s NOT IN ('interna_corporativa','administracion_privilegiada') THEN RAISE EXCEPTION 'superficie'; END IF;
 WHEN tipo IN ('instante','instante_fijo') THEN
  IF vec_contratacion_temporal.instante_incorporacion_go_v2(s,tipo='instante_fijo') IS NOT TRUE THEN RAISE EXCEPTION 'instante'; END IF;
 WHEN tipo='bytes' THEN PERFORM vec_contratacion_temporal.bytes_incorporacion_v2(v);
 ELSE RAISE EXCEPTION 'tipo desconocido';
 END CASE;
 RETURN vec_contratacion_temporal.texto_json_incorporacion_go_v2(s);
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material canónico CT inválido';
END $$;

-- Canon de solicitud reutiliza la disciplina de campos con longitud EN BYTES
-- del contrato CT, sin invocar esquema ni codec de Personal.
CREATE FUNCTION vec_contratacion_temporal.campo_solicitud_incorporacion_v2(k text,v text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $$
 SELECT octet_length(convert_to(k,'UTF8'))::text||':'||k||octet_length(convert_to(v,'UTF8'))::text||':'||v
$$;

CREATE FUNCTION vec_contratacion_temporal.solicitud_personal_incorporacion_canonica_v2(s jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE k text; r text:=''; f jsonb; nombres text[]:=ARRAY['esquema','contrato_version','solicitud_ref','expediente_ref','version_expediente','capacidad_ref','correlacion_ref','idempotencia_ref'];
BEGIN
 PERFORM vec_contratacion_temporal.nodo_incorporacion_canonico_v2(s,'solicitud');
 FOREACH k IN ARRAY nombres LOOP r:=r||vec_contratacion_temporal.campo_solicitud_incorporacion_v2(k,s->>k); END LOOP;
 f:=s->'fuente_rpt';
 r:=r||vec_contratacion_temporal.campo_solicitud_incorporacion_v2('rpt_ref',f->>'referencia')||
 vec_contratacion_temporal.campo_solicitud_incorporacion_v2('rpt_version',f->>'version')||
 vec_contratacion_temporal.campo_solicitud_incorporacion_v2('rpt_huella_sha256',f->>'huella_sha256')||
 vec_contratacion_temporal.campo_solicitud_incorporacion_v2('puesto_ref',s->>'puesto_ref')||
 vec_contratacion_temporal.campo_solicitud_incorporacion_v2('plaza_ref',s->>'plaza_ref');
 RETURN convert_to(r,'UTF8');
END $$;

CREATE FUNCTION vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(entrada jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE canon text; c jsonb; s jsonb; r jsonb; p jsonb; v jsonb; sc jsonb;
 contexto jsonb; procedencia jsonb; bc bytea; bp bytea; k text; par text[];
 i int; cv jsonb; pv jsonb; resuelto text; desde text; hasta text;
BEGIN
 canon:=vec_contratacion_temporal.nodo_incorporacion_canonico_v2(entrada,'material');
 c:=entrada->'Confirmacion'; s:=c->'SolicitudPersonal'; r:=c->'ResultadoPersonal';
 p:=entrada->'Personal'; v:=entrada->'Vinculo'; sc:=entrada->'SolicitudContexto';
 IF p->'Solicitud' IS DISTINCT FROM s OR p->'Resultado' IS DISTINCT FROM r THEN RAISE EXCEPTION 'original'; END IF;
 FOREACH k IN ARRAY ARRAY['solicitud_ref','correlacion_ref','idempotencia_ref'] LOOP
  IF r->>k IS DISTINCT FROM s->>k THEN RAISE EXCEPTION 'resultado'; END IF;
 END LOOP;
 IF r->>'huella_solicitud_sha256' IS DISTINCT FROM encode(pg_catalog.sha256(
    vec_contratacion_temporal.solicitud_personal_incorporacion_canonica_v2(s)),'hex')
 OR p->>'MaterialSHA256' IS DISTINCT FROM encode(pg_catalog.sha256(
    vec_contratacion_temporal.bytes_incorporacion_v2(p->'MaterialCanonico')),'hex') THEN RAISE EXCEPTION 'hash original'; END IF;
 desde:=vec_contratacion_temporal.clave_instante_incorporacion_v2(c->'PeriodoIncorporacion'->>'desde');
 hasta:=vec_contratacion_temporal.clave_instante_incorporacion_v2(c->'PeriodoIncorporacion'->>'hasta');
 IF (hasta COLLATE "C") <= (desde COLLATE "C") THEN RAISE EXCEPTION 'periodo'; END IF;
 IF sc->>'AutenticacionRef' IS DISTINCT FROM v->>'autenticacion_ref'
 OR sc->>'SesionRef' IS DISTINCT FROM v->>'sesion_ref'
 OR sc->>'PerfilRef' IS DISTINCT FROM v->>'perfil_activo_ref' THEN RAISE EXCEPTION 'contexto'; END IF;
 IF (v->>'cuenta_privilegiada'='true' AND (v->>'superficie'<>'administracion_privilegiada' OR v->>'cuenta_ref'=v->>'cuenta_ordinaria_ref'))
 OR (v->>'cuenta_privilegiada'='false' AND (v->>'superficie'='administracion_privilegiada' OR v->>'cuenta_ref'<>v->>'cuenta_ordinaria_ref'))
 THEN RAISE EXCEPTION 'cuenta'; END IF;
 IF (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'autenticacion_verificada_en') COLLATE "C") >
    (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_emitida_en') COLLATE "C")
 OR (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_emitida_en') COLLATE "C") >
    (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_revalidada_en') COLLATE "C")
 OR (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_revalidada_en') COLLATE "C") >=
    (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_valida_hasta') COLLATE "C")
 THEN RAISE EXCEPTION 'ventana'; END IF;

 bc:=vec_contratacion_temporal.bytes_incorporacion_v2(entrada->'ContextoCanonico');
 bp:=vec_contratacion_temporal.bytes_incorporacion_v2(entrada->'ProcedenciaCanonica');
 IF encode(pg_catalog.sha256(bc),'hex') IS DISTINCT FROM v->>'contexto_actor_huella_sha256'
 OR encode(pg_catalog.sha256(bp),'hex') IS DISTINCT FROM v->>'manifiesto_procedencia_huella_sha256'
 THEN RAISE EXCEPTION 'canon original'; END IF;
 contexto:=convert_from(bc,'UTF8')::jsonb; procedencia:=convert_from(bp,'UTF8')::jsonb;
 IF bc IS DISTINCT FROM convert_to(vec_contratacion_temporal.nodo_incorporacion_canonico_v2(contexto,'contexto'),'UTF8')
 OR bp IS DISTINCT FROM convert_to(vec_contratacion_temporal.nodo_incorporacion_canonico_v2(procedencia,'procedencia'),'UTF8')
 THEN RAISE EXCEPTION 'canon alternativo'; END IF;
 -- Comparación explícita de todas las ligaduras de Vinculo.ValidarPara.
 FOREACH par SLICE 1 IN ARRAY ARRAY[
 ['cuenta_ref','cuenta_ref'],['principal_id','principal_ref'],['principal_id','persona_ref'],
 ['perfil_activo_ref','perfil_activo_ref'],['metodo_observado','metodo'],['garantia_observada','garantia'],
 ['contexto_actor_ref','contexto_actor_ref'],['contexto_actor_version','contexto_version'],
 ['contexto_actor_cuenta_version','cuenta_version']
 ] LOOP
  IF v->par[1] IS DISTINCT FROM contexto->par[2] THEN RAISE EXCEPTION 'vínculo cruzado'; END IF;
 END LOOP;
 IF procedencia->'autoridad_efectiva' IS DISTINCT FROM v->'autoridad_efectiva' THEN RAISE EXCEPTION 'procedencia'; END IF;
 FOREACH par SLICE 1 IN ARRAY ARRAY[
 ['cuenta','cuenta_ref','cuenta_ref','cuenta_version'],['persona','persona_ref','persona_ref','persona_version'],
 ['perfil','perfil_ref','perfil_activo_ref','perfil_version'],['contexto','vinculo_ref','contexto_actor_ref','contexto_version']
 ] LOOP
  IF procedencia->par[1]->par[2] IS DISTINCT FROM contexto->par[3]
  OR procedencia->par[1]->'version' IS DISTINCT FROM contexto->par[4] THEN RAISE EXCEPTION 'procedencia cruzada'; END IF;
 END LOOP;
 resuelto:=vec_contratacion_temporal.clave_instante_incorporacion_v2(contexto->>'resuelto_en');
 desde:=vec_contratacion_temporal.clave_instante_incorporacion_v2(contexto->>'vigente_desde');
 hasta:=vec_contratacion_temporal.clave_instante_incorporacion_v2(contexto->>'vigente_hasta');
 IF (resuelto COLLATE "C") < (desde COLLATE "C") OR (resuelto COLLATE "C") >= (hasta COLLATE "C")
 OR (resuelto COLLATE "C") < (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_revalidada_en') COLLATE "C")
 OR (resuelto COLLATE "C") >= (vec_contratacion_temporal.clave_instante_incorporacion_v2(v->>'sesion_valida_hasta') COLLATE "C")
 THEN RAISE EXCEPTION 'vigencia contexto'; END IF;
 IF jsonb_array_length(contexto->'vinculos')<>jsonb_array_length(procedencia->'vinculos') THEN RAISE EXCEPTION 'vínculos'; END IF;
 FOR i IN 0..jsonb_array_length(contexto->'vinculos')-1 LOOP
  cv:=contexto->'vinculos'->i; pv:=procedencia->'vinculos'->i;
  FOREACH k IN ARRAY ARRAY['vinculo_ref','version','tipo','referencia'] LOOP
   IF cv->k IS DISTINCT FROM pv->k THEN RAISE EXCEPTION 'vínculo procedencia'; END IF;
  END LOOP;
  IF (resuelto COLLATE "C") < (vec_contratacion_temporal.clave_instante_incorporacion_v2(cv->>'vigente_desde') COLLATE "C")
  OR (resuelto COLLATE "C") >= (vec_contratacion_temporal.clave_instante_incorporacion_v2(cv->>'vigente_hasta') COLLATE "C")
  THEN RAISE EXCEPTION 'vigencia vínculo'; END IF;
 END LOOP;
 -- No reloj, firma, revocación ni commit: deben comprobarse en consumidor AD3 y
 -- acreditador propietario. No se examina semántica dentro del material Personal.
 RETURN convert_to(canon,'UTF8');
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='material de incorporación CT inválido';
END $$;

CREATE FUNCTION vec_contratacion_temporal.material_incorporacion_ejercicio_sha256_v2(entrada jsonb) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
 SELECT encode(pg_catalog.sha256(vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(entrada)),'hex')
$$;

-- Solo estos dos mapas componen HuellaContextoAutorizacionSHA256. Identidad de
-- recurso (ref/módulo/tipo) se devuelve aparte y debe cotejarse explícitamente.
CREATE FUNCTION vec_contratacion_temporal.contexto_incorporacion_ejercicio_canonico_v2(entrada jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE h text; p jsonb; v jsonb; salida text;
BEGIN
 h:=vec_contratacion_temporal.material_incorporacion_ejercicio_sha256_v2(entrada);
 p:=entrada->'Preparacion'; v:=entrada->'Vinculo';
 salida:='{"ambitos":{"organizacion_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(p->>'OrganizacionRef')||
 ',"unidad_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(p->>'UnidadRef')||
 '},"atributos":{"actor_seguimiento_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(p->>'ActorRef')||
 ',"correlacion_seguimiento_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(p->>'CorrelacionRef')||
 ',"correlacion_v3_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(entrada->>'CorrelacionV3')||
 ',"material_sha256":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(h)||
 ',"motivo_v3_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(entrada->'MotivoV3'->>'entrada_clave')||
 ',"perfil_v3_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(v->>'perfil_activo_ref')||
 ',"principal_v3_ref":'||vec_contratacion_temporal.texto_json_incorporacion_go_v2(v->>'principal_id')||
 ',"tipo_validacion":"ejercicio_sintetico"}}';
 RETURN convert_to(salida,'UTF8');
END $$;

CREATE FUNCTION vec_contratacion_temporal.contexto_incorporacion_ejercicio_sha256_v2(entrada jsonb) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
 SELECT encode(pg_catalog.sha256(vec_contratacion_temporal.contexto_incorporacion_ejercicio_canonico_v2(entrada)),'hex')
$$;

CREATE FUNCTION vec_contratacion_temporal.recurso_incorporacion_ejercicio_v2(entrada jsonb) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE c jsonb;
BEGIN
 c:=convert_from(vec_contratacion_temporal.contexto_incorporacion_ejercicio_canonico_v2(entrada),'UTF8')::jsonb;
 RETURN jsonb_build_object('referencia',entrada->'Confirmacion'->'SolicitudPersonal'->>'expediente_ref',
 'modulo_id','contratacion_temporal','tipo','confirmacion_incorporacion_ejercicio_v2',
 'ambitos',c->'ambitos','atributos',c->'atributos');
END $$;

-- Elimina también grants de ALTER DEFAULT PRIVILEGES: solo propietario CT.
DO $acl$
DECLARE f record; g record;
BEGIN
 FOR f IN SELECT p.oid,p.proowner FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_contratacion_temporal' AND p.proname=ANY(ARRAY['texto_json_incorporacion_go_v2','fecha_civil_incorporacion_go_v2','instante_incorporacion_go_v2','clave_instante_incorporacion_v2','bytes_incorporacion_v2','nodo_incorporacion_canonico_v2','campo_solicitud_incorporacion_v2','solicitud_personal_incorporacion_canonica_v2','material_incorporacion_ejercicio_canonico_v2','material_incorporacion_ejercicio_sha256_v2','contexto_incorporacion_ejercicio_canonico_v2','contexto_incorporacion_ejercicio_sha256_v2','recurso_incorporacion_ejercicio_v2']) LOOP
  IF f.proowner<>'vec_contratacion_temporal_propietario'::regrole THEN RAISE EXCEPTION 'owner codec incompatible' USING ERRCODE='55000'; END IF;
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
  FOR g IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid=f.oid AND a.grantee<>0 AND a.grantee<>f.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.oid::regprocedure,pg_get_userbyid(g.grantee));
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;

