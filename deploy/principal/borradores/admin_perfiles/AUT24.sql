\set ON_ERROR_STOP on
-- BORRADOR AUT24. No instalar. Dependencias descritas en CONTRATO.md.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $preimagen$
BEGIN
 IF to_regclass('vec_autorizacion.control_continuidad_admin') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz)') IS NULL
 THEN
  RAISE EXCEPTION 'AUT24: dependencia esperado=presente observado=ausente' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_minimo_personas=2 AND minimo_personas=1) THEN RAISE EXCEPTION 'AUT24: continuidad esperado=bootstrap2_minimo1 observado=divergente' USING ERRCODE='55000'; END IF;
END $preimagen$;
-- No se crean LOGIN ni personas. El rol técnico no permite SET ROLE owner.
DO $rol$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_ejecutor') THEN
  CREATE ROLE vec_admin_perfiles_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_ejecutor'
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_perfiles_ejecutor'::regrole) THEN
  RAISE EXCEPTION 'AUT24: rol_runtime esperado=aislado observado=privilegiado' USING ERRCODE='55000';
 END IF;
END $rol$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.json_cadena_canonica_go_admin_v1(p_valor text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT replace(replace(replace(replace(replace(to_jsonb(p_valor)::text,'<',E'\\u003c'),'>',E'\\u003e'),'&',E'\\u0026'),U&'\2028',E'\\u2028'),U&'\2029',E'\\u2029')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.json_cadena_canonica_go_admin_v1(text) FROM PUBLIC;

-- Fragmento AUT24: ensamblar después de json_cadena_canonica_go_admin_v1 y
-- antes de publicar rol v3. No es una migración ni un instalador autónomo.
-- Fuente: internal/vec/domain/autorizacion.go y huellaAutorizacion (json.Marshal).
-- Devuelve los bytes JSON de los tipos Go, nunca la representación jsonb::text.
-- Cuenta, vínculo, referencia de acto y huella de rol pertenecen al enlace SQL;
-- no son campos de AsignacionPerfil y este serializador los rechaza.
-- No concede permisos ni sustituye la validación nominal del efecto.

CREATE FUNCTION vec_autorizacion.fecha_canonica_go_admin_v1(p_fecha text, p_permitir_cero boolean DEFAULT false)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE partes text[]; fraccion text; ano integer; mes integer; dia integer;
BEGIN
 IF p_fecha IS NULL AND p_permitir_cero IS TRUE THEN
  RETURN '0001-01-01T00:00:00Z';
 END IF;
 partes:=regexp_match(p_fecha,'^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})([.][0-9]{1,9})?Z$');
 IF partes IS NULL THEN RAISE EXCEPTION 'AUT24: fecha canónica inválida' USING ERRCODE='22023'; END IF;
 ano:=partes[1]::integer; mes:=partes[2]::integer; dia:=partes[3]::integer;
 IF ano NOT BETWEEN 1 AND 9999 OR mes NOT BETWEEN 1 AND 12 OR dia NOT BETWEEN 1 AND 31
 OR partes[4]::integer>23 OR partes[5]::integer>59 OR partes[6]::integer>59 THEN
  RAISE EXCEPTION 'AUT24: fecha canónica inválida' USING ERRCODE='22023';
 END IF;
 -- make_date valida el calendario sin DateStyle ni TimeZone y no redondea.
 PERFORM make_date(ano,mes,dia);
 fraccion:=rtrim(coalesce(substring(partes[7] FROM 2),''),'0');
 -- El dominio rechaza restos inferiores al microsegundo antes de formar SHA.
 IF length(fraccion)>6 THEN RAISE EXCEPTION 'AUT24: precisión temporal inválida' USING ERRCODE='22023'; END IF;
 IF fraccion<>'' THEN fraccion:='.'||fraccion; END IF;
 IF p_permitir_cero IS NOT TRUE AND substring(p_fecha FROM 1 FOR 19)||fraccion||'Z'='0001-01-01T00:00:00Z' THEN
  RAISE EXCEPTION 'AUT24: instante obligatorio ausente' USING ERRCODE='22023';
 END IF;
 RETURN substring(p_fecha FROM 1 FOR 19)||fraccion||'Z';
EXCEPTION WHEN datetime_field_overflow THEN
 RAISE EXCEPTION 'AUT24: fecha canónica inválida' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.fecha_canonica_go_admin_v1(text,boolean) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.array_cadenas_canonico_go_admin_v1(p_array jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE elemento jsonb; valor text; salida text:=''; vistos text[]:=ARRAY[]::text[];
BEGIN
 IF jsonb_typeof(p_array) IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'AUT24: lista canónica inválida' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_array)>512 THEN RAISE EXCEPTION 'AUT24: lista canónica excesiva' USING ERRCODE='22023'; END IF;
 FOR elemento IN SELECT value FROM jsonb_array_elements(p_array) WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
  IF jsonb_typeof(elemento) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AUT24: elemento canónico inválido' USING ERRCODE='22023'; END IF;
  valor:=elemento#>>'{}';
  IF octet_length(valor) NOT BETWEEN 1 AND 512 OR valor COLLATE "C" ~ '[^!-~]' OR strpos(valor,'*')>0 OR valor=ANY(vistos) THEN
   RAISE EXCEPTION 'AUT24: elemento canónico inválido' USING ERRCODE='22023';
  END IF;
  vistos:=array_append(vistos,valor);
  IF salida<>'' THEN salida:=salida||','; END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(valor);
 END LOOP;
 RETURN '['||salida||']';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.array_cadenas_canonico_go_admin_v1(jsonb) FROM PUBLIC;

-- Esquemas cerrados y orden de declaración de los structs, incluidos anidados.
-- Los únicos omitempty son strings/listas vacíos; time.Time siempre se escribe.
CREATE FUNCTION vec_autorizacion.objeto_canonico_go_admin_v1(p_documento jsonb,p_tipo text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE campos text[]; tipos text[]; opcionales text[]:=ARRAY[]::text[];
 campo text; tipo text; valor jsonb; texto text; serial text; salida text:='';
 elemento jsonb; lista text; numero numeric; i integer; maximo integer;
BEGIN
 CASE p_tipo
 WHEN 'asignacion' THEN
  campos:=ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref','estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en','revocada_por','revocada_en','revocacion_ref'];
  tipos:=ARRAY['string','int','string','string','string','string','ambito[]','fecha','fecha','string','fecha','string','fecha','string'];
  opcionales:=ARRAY['revocada_por','revocada_en','revocacion_ref'];
 WHEN 'rol' THEN
  campos:=ARRAY['rol_id','version','nombre','estado','concesiones','publicada_por','publicada_en','retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo'];
  tipos:=ARRAY['string','int','string','string','concesion[]','string','fecha','string','fecha','string','string'];
  opcionales:=ARRAY['retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo'];
 WHEN 'control' THEN
  campos:=ARRAY['version_rol_ref','revision','estado','actualizado_por','actualizado_en','acto_ref','motivo_codigo'];
  tipos:=ARRAY['string','uint64','string','string','fecha','string','string'];
  opcionales:=ARRAY['acto_ref','motivo_codigo'];
 WHEN 'ambito' THEN
  campos:=ARRAY['clave','valores']; tipos:=ARRAY['string','string[]'];
 WHEN 'concesion' THEN
  campos:=ARRAY['accion','modulo_id','tipo_recurso','finalidades','garantia_minima','campos_permitidos','obligaciones'];
  tipos:=ARRAY['string','string','string','string[]','string','string[]','string[]'];
  opcionales:=ARRAY['campos_permitidos','obligaciones'];
 ELSE RAISE EXCEPTION 'AUT24: tipo canónico desconocido' USING ERRCODE='22023';
 END CASE;
 IF jsonb_typeof(p_documento) IS DISTINCT FROM 'object' THEN RAISE EXCEPTION 'AUT24: objeto canónico inválido' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_object_keys(p_documento) AS k(clave) WHERE NOT(clave=ANY(campos))) THEN
  RAISE EXCEPTION 'AUT24: campo canónico desconocido' USING ERRCODE='22023';
 END IF;
 FOR i IN 1..array_length(campos,1) LOOP
  campo:=campos[i]; tipo:=tipos[i]; valor:=p_documento->campo; serial:=NULL;
  IF valor IS NULL OR valor='null'::jsonb THEN
   IF NOT(campo=ANY(opcionales)) THEN RAISE EXCEPTION 'AUT24: campo canónico obligatorio ausente' USING ERRCODE='22023'; END IF;
   -- time.Time.UnmarshalJSON(null) deja el valor cero; omitempty no lo omite.
   IF tipo='fecha' THEN serial:='"0001-01-01T00:00:00Z"'; ELSE CONTINUE; END IF;
  ELSIF tipo='string' THEN
   IF jsonb_typeof(valor) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AUT24: cadena canónica inválida' USING ERRCODE='22023'; END IF;
   texto:=valor#>>'{}';
   IF campo=ANY(opcionales) AND texto='' THEN CONTINUE; END IF;
   maximo:=CASE WHEN campo IN ('rol_id','modulo_id','tipo_recurso','clave','motivo_codigo','motivo_retirada_codigo') THEN 128 WHEN campo='accion' THEN 256 ELSE 512 END;
   IF octet_length(texto) NOT BETWEEN 1 AND maximo THEN RAISE EXCEPTION 'AUT24: cadena canónica inválida' USING ERRCODE='22023'; END IF;
   IF campo<>'nombre' AND (texto COLLATE "C" ~ '[^!-~]' OR strpos(texto,'*')>0) THEN RAISE EXCEPTION 'AUT24: cadena positiva inválida' USING ERRCODE='22023'; END IF;
   IF campo='nombre' AND (texto<>btrim(texto) OR texto ~ '[[:cntrl:]]') THEN RAISE EXCEPTION 'AUT24: nombre canónico inválido' USING ERRCODE='22023'; END IF;
   IF campo='garantia_minima' AND texto NOT IN ('bajo','sustancial','alto') THEN RAISE EXCEPTION 'AUT24: garantía inválida' USING ERRCODE='22023'; END IF;
   IF p_tipo='ambito' AND campo='clave' AND texto='global' THEN RAISE EXCEPTION 'AUT24: ámbito inválido' USING ERRCODE='22023'; END IF;
   IF campo='estado' AND ((p_tipo='asignacion' AND texto NOT IN ('activa','revocada')) OR (p_tipo='rol' AND texto NOT IN ('publicada','retirada')) OR (p_tipo='control' AND texto NOT IN ('habilitada','retirada'))) THEN RAISE EXCEPTION 'AUT24: estado canónico inválido' USING ERRCODE='22023'; END IF;
   serial:=vec_autorizacion.json_cadena_canonica_go_admin_v1(texto);
  ELSIF tipo IN ('int','uint64') THEN
   IF jsonb_typeof(valor) IS DISTINCT FROM 'number' OR (valor#>>'{}') !~ '^[1-9][0-9]*$' THEN RAISE EXCEPTION 'AUT24: entero canónico inválido' USING ERRCODE='22023'; END IF;
   numero:=(valor#>>'{}')::numeric;
   IF (tipo='int' AND numero>9223372036854775807) OR (tipo='uint64' AND numero>18446744073709551615) THEN RAISE EXCEPTION 'AUT24: entero canónico fuera de rango' USING ERRCODE='22023'; END IF;
   serial:=valor#>>'{}';
  ELSIF tipo='fecha' THEN
   IF jsonb_typeof(valor) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AUT24: fecha canónica inválida' USING ERRCODE='22023'; END IF;
   serial:=vec_autorizacion.json_cadena_canonica_go_admin_v1(vec_autorizacion.fecha_canonica_go_admin_v1(valor#>>'{}',campo=ANY(opcionales)));
  ELSE
   IF jsonb_typeof(valor) IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'AUT24: lista canónica inválida' USING ERRCODE='22023'; END IF;
   IF jsonb_array_length(valor)=0 AND campo=ANY(opcionales) THEN CONTINUE; END IF;
   IF jsonb_array_length(valor) NOT BETWEEN 1 AND 512 THEN RAISE EXCEPTION 'AUT24: cardinalidad canónica inválida' USING ERRCODE='22023'; END IF;
   IF tipo='string[]' THEN serial:=vec_autorizacion.array_cadenas_canonico_go_admin_v1(valor);
   ELSE
    lista:='';
    FOR elemento IN SELECT value FROM jsonb_array_elements(valor) WITH ORDINALITY AS e(value,n) ORDER BY n LOOP
     IF lista<>'' THEN lista:=lista||','; END IF;
     lista:=lista||vec_autorizacion.objeto_canonico_go_admin_v1(elemento,CASE tipo WHEN 'ambito[]' THEN 'ambito' ELSE 'concesion' END);
    END LOOP;
    serial:='['||lista||']';
   END IF;
  END IF;
  IF salida<>'' THEN salida:=salida||','; END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(campo)||':'||serial;
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.objeto_canonico_go_admin_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.canon_asignacion_perfil_admin_v1(p_documento jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT vec_autorizacion.objeto_canonico_go_admin_v1(p_documento,'asignacion')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.canon_version_rol_admin_v1(p_documento jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT vec_autorizacion.objeto_canonico_go_admin_v1(p_documento,'rol')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_version_rol_admin_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.canon_control_rol_admin_v1(p_documento jsonb)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT vec_autorizacion.objeto_canonico_go_admin_v1(p_documento,'control')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_control_rol_admin_v1(jsonb) FROM PUBLIC;

-- Vectores sintéticos: Go estándar, structs extraídos literalmente de la fuente.
-- Generados con json.Decoder.DisallowUnknownFields + json.Marshal + SHA256.
-- Ejecutados sin red en bwrap, GOMAXPROCS=2, go run -p 8; no paquetes del repo.
-- Cada línea GOLDEN aporta entrada, texto exacto y SHA256 UTF-8 para el ensayo SQL.
-- GOLDEN {"tipo":"asignacion","input":{"asignacion_id":"asignacion_sintetica","version":1,"perfil_activo_ref":"perfil:sintetico","principal_id":"persona:sintetica","version_rol_ref":"rol:sintetico:v1","estado":"activa","ambitos":[{"clave":"unidad","valores":["unidad:b","unidad:a"]}],"vigente_desde":"2026-10-02T12:00:00.120000000Z","vigente_hasta":"2027-10-02T12:00:00Z","emitida_por":"persona:emisora","emitida_en":"2026-10-02T12:00:00Z","revocada_por":null,"revocacion_ref":""},"canon":"{\"asignacion_id\":\"asignacion_sintetica\",\"version\":1,\"perfil_activo_ref\":\"perfil:sintetico\",\"principal_id\":\"persona:sintetica\",\"version_rol_ref\":\"rol:sintetico:v1\",\"estado\":\"activa\",\"ambitos\":[{\"clave\":\"unidad\",\"valores\":[\"unidad:b\",\"unidad:a\"]}],\"vigente_desde\":\"2026-10-02T12:00:00.12Z\",\"vigente_hasta\":\"2027-10-02T12:00:00Z\",\"emitida_por\":\"persona:emisora\",\"emitida_en\":\"2026-10-02T12:00:00Z\",\"revocada_en\":\"0001-01-01T00:00:00Z\"}","sha256":"0d7ec33d2506150f40debc8448caa12b5eb136531455051ed4757267d27dd7a6"}
-- GOLDEN {"tipo":"asignacion","input":{"asignacion_id":"asignacion_sintetica","version":2,"perfil_activo_ref":"perfil:sintetico","principal_id":"persona:sintetica","version_rol_ref":"rol:sintetico:v1","estado":"revocada","ambitos":[{"clave":"unidad","valores":["unidad:a"]},{"clave":"centro","valores":["centro:b","centro:a"]}],"vigente_desde":"2026-10-02T12:00:00.000001Z","vigente_hasta":"2027-10-02T12:00:00Z","emitida_por":"persona:emisora","emitida_en":"2026-10-02T12:00:00Z","revocada_por":"persona:revocadora","revocada_en":"2026-10-02T12:01:00.123400000Z","revocacion_ref":"acto:revocacion"},"canon":"{\"asignacion_id\":\"asignacion_sintetica\",\"version\":2,\"perfil_activo_ref\":\"perfil:sintetico\",\"principal_id\":\"persona:sintetica\",\"version_rol_ref\":\"rol:sintetico:v1\",\"estado\":\"revocada\",\"ambitos\":[{\"clave\":\"unidad\",\"valores\":[\"unidad:a\"]},{\"clave\":\"centro\",\"valores\":[\"centro:b\",\"centro:a\"]}],\"vigente_desde\":\"2026-10-02T12:00:00.000001Z\",\"vigente_hasta\":\"2027-10-02T12:00:00Z\",\"emitida_por\":\"persona:emisora\",\"emitida_en\":\"2026-10-02T12:00:00Z\",\"revocada_por\":\"persona:revocadora\",\"revocada_en\":\"2026-10-02T12:01:00.1234Z\",\"revocacion_ref\":\"acto:revocacion\"}","sha256":"d219fce4659582e4feb3cddd95e63937ef9b2027d25e01587b030d6e7a706d44"}
-- GOLDEN {"tipo":"rol","input":{"rol_id":"sintetico","version":3,"nombre":"Rol \u003c\u003e\u0026 sintético\u2028A\u2029B","estado":"publicada","concesiones":[{"accion":"administracion.perfiles.consultar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["finalidad:b","finalidad:a"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":null},{"accion":"administracion.perfiles.otorgar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["finalidad:a"],"garantia_minima":"alto","campos_permitidos":["campo:b","campo:a"],"obligaciones":["obligacion:b","obligacion:a"]}],"publicada_por":"migracion:autorizacion:000024","publicada_en":"2026-10-02T12:00:00.000000000Z"},"canon":"{\"rol_id\":\"sintetico\",\"version\":3,\"nombre\":\"Rol \\u003c\\u003e\\u0026 sintético\\u2028A\\u2029B\",\"estado\":\"publicada\",\"concesiones\":[{\"accion\":\"administracion.perfiles.consultar\",\"modulo_id\":\"administracion\",\"tipo_recurso\":\"perfil\",\"finalidades\":[\"finalidad:b\",\"finalidad:a\"],\"garantia_minima\":\"alto\"},{\"accion\":\"administracion.perfiles.otorgar\",\"modulo_id\":\"administracion\",\"tipo_recurso\":\"perfil\",\"finalidades\":[\"finalidad:a\"],\"garantia_minima\":\"alto\",\"campos_permitidos\":[\"campo:b\",\"campo:a\"],\"obligaciones\":[\"obligacion:b\",\"obligacion:a\"]}],\"publicada_por\":\"migracion:autorizacion:000024\",\"publicada_en\":\"2026-10-02T12:00:00Z\",\"retirada_en\":\"0001-01-01T00:00:00Z\"}","sha256":"a67f1e01784d9433fef3a37eaee41cbb7f9c3bd50d42eb44a959bb29d229d182"}
-- GOLDEN {"tipo":"rol","input":{"rol_id":"sintetico","version":3,"nombre":"Rol sintético retirado","estado":"retirada","concesiones":[{"accion":"administracion.perfiles.consultar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["finalidad:a"],"garantia_minima":"sustancial"}],"publicada_por":"persona:publicadora","publicada_en":"2026-10-02T12:00:00Z","retirada_por":"persona:retiradora","retirada_en":"2026-10-02T12:01:00.123456Z","retirada_ref":"acto:retirada","motivo_retirada_codigo":"motivo:retirada"},"canon":"{\"rol_id\":\"sintetico\",\"version\":3,\"nombre\":\"Rol sintético retirado\",\"estado\":\"retirada\",\"concesiones\":[{\"accion\":\"administracion.perfiles.consultar\",\"modulo_id\":\"administracion\",\"tipo_recurso\":\"perfil\",\"finalidades\":[\"finalidad:a\"],\"garantia_minima\":\"sustancial\"}],\"publicada_por\":\"persona:publicadora\",\"publicada_en\":\"2026-10-02T12:00:00Z\",\"retirada_por\":\"persona:retiradora\",\"retirada_en\":\"2026-10-02T12:01:00.123456Z\",\"retirada_ref\":\"acto:retirada\",\"motivo_retirada_codigo\":\"motivo:retirada\"}","sha256":"c6a5848459f8076a8ed5e3cc7499b12d961ce349e82b3fcced1323186297a00d"}
-- GOLDEN {"tipo":"control","input":{"version_rol_ref":"rol:sintetico:v3","revision":1,"estado":"habilitada","actualizado_por":"migracion:autorizacion:000024","actualizado_en":"2026-10-02T12:00:00.100000Z","acto_ref":null,"motivo_codigo":""},"canon":"{\"version_rol_ref\":\"rol:sintetico:v3\",\"revision\":1,\"estado\":\"habilitada\",\"actualizado_por\":\"migracion:autorizacion:000024\",\"actualizado_en\":\"2026-10-02T12:00:00.1Z\"}","sha256":"d710b88224abf0ee59bd522c316d995ef60d2f7e6df3838371dd21ca4c8f36eb"}
-- GOLDEN {"tipo":"control","input":{"version_rol_ref":"rol:sintetico:v3","revision":18446744073709551615,"estado":"retirada","actualizado_por":"persona:retiradora","actualizado_en":"2026-10-02T12:01:00.123456Z","acto_ref":"acto:retirada","motivo_codigo":"motivo:retirada"},"canon":"{\"version_rol_ref\":\"rol:sintetico:v3\",\"revision\":18446744073709551615,\"estado\":\"retirada\",\"actualizado_por\":\"persona:retiradora\",\"actualizado_en\":\"2026-10-02T12:01:00.123456Z\",\"acto_ref\":\"acto:retirada\",\"motivo_codigo\":\"motivo:retirada\"}","sha256":"7376121bdc44edacda82d3dd112fc526cee45246f7754b672780fadac6aead85"}
-- Negativos del ensayo pendiente (SQLSTATE 22023): extra cuenta_ref/vinculo_ref/
-- referencia_acto/rol_huella_sha256; extras anidados; campo obligatorio ausente;
-- versión decimal/fuera de int64; revisión fuera de uint64; lista sin strings;
-- fecha no UTC, inválida, cero obligatorio o con precisión submicrosegundo.
-- La entrada jsonb ya no conserva claves duplicadas: la frontera Go debe usar
-- su decoder estricto antes de llegar aquí; estos auxiliares no reconstruyen
-- un documento SQL-only ni autorizan datos que el dominio rechace.

DO $rol_lector_v3$
DECLARE anterior record;d jsonb;control jsonb;ahora timestamptz:=date_trunc('second',clock_timestamp());texto_fecha text;
BEGIN
 SELECT * INTO STRICT anterior FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v2';
 IF anterior.rol_id IS DISTINCT FROM 'administracion_perfiles' OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto WHERE version_rol_ref=anterior.version_rol_ref AND clase='administrador' AND huella_sha256=anterior.huella_sha256) THEN RAISE EXCEPTION 'AUT24: rol lector previo divergente' USING ERRCODE='55000'; END IF;
 texto_fecha:=to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 d:=jsonb_set(anterior.documento,'{version}','3'::jsonb);
 SELECT jsonb_set(d,'{concesiones}',jsonb_agg(CASE WHEN x->>'accion'='administracion.perfiles.consultar' THEN jsonb_set(x,'{campos_permitidos}','["actos_disponibles","capacidades","perfil_ref","persona","personas","preimagen","roles","siguiente_cursor","version","vinculo_ref"]'::jsonb) WHEN x->>'accion'='administracion.perfiles.historial.consultar' THEN jsonb_set(x,'{campos_permitidos}','["propuesta","propuestas"]'::jsonb) WHEN x->>'accion'='administracion.perfiles.recibo.consultar' THEN jsonb_set(x,'{campos_permitidos}','["recibo"]'::jsonb) ELSE x END ORDER BY n)) INTO d FROM jsonb_array_elements(anterior.documento->'concesiones') WITH ORDINALITY AS e(x,n);
 d:=jsonb_set(d,'{concesiones}',(d->'concesiones')||jsonb_build_array(jsonb_build_object('accion','copias_restauracion_revisar','modulo_id','administracion','tipo_recurso','propuesta_restauracion_copia','finalidades',jsonb_build_array('revisar_restauracion_copia'),'garantia_minima','alto','campos_permitidos',jsonb_build_array('propuesta','recibo','revision'))));
 d:=jsonb_set(jsonb_set(d,'{publicada_por}','"migracion:autorizacion:000024"'::jsonb),'{publicada_en}',to_jsonb(texto_fecha));
 IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: rol lector fijo invalido' USING ERRCODE='23514'; END IF;
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES('rol:administracion_perfiles:v3','administracion_perfiles',3,encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex'),ahora,d);
 control:=jsonb_build_object('version_rol_ref','rol:administracion_perfiles:v3','revision',1,'estado','habilitada','actualizado_por','migracion:autorizacion:000024','actualizado_en',texto_fecha);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento,creada_en) VALUES('rol:administracion_perfiles:v3',1,'habilitada',encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex'),ahora,control,clock_timestamp());
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES('rol:administracion_perfiles:v3',1,ahora,'migracion:autorizacion:000024','migracion:autorizacion:000024');
 INSERT INTO vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase,huella_sha256) SELECT version_rol_ref,'administrador',huella_sha256 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v3';
END $rol_lector_v3$;
DO $rol_sistemas_v1$
DECLARE d jsonb;c jsonb;ahora timestamptz:=date_trunc('second',clock_timestamp());fecha text;
BEGIN
 fecha:=to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"');
 d:=jsonb_build_object('rol_id','operador_plataforma','version',1,'nombre','administracion.sistemas.rol','estado','publicada','concesiones',jsonb_build_array(
 jsonb_build_object('accion','administracion.copias.orden.emitir','modulo_id','administracion','tipo_recurso','orden_copia','finalidades',jsonb_build_array('emitir_orden_copia'),'garantia_minima','alto','campos_permitidos',jsonb_build_array('orden','recibo')),
 jsonb_build_object('accion','copias_restauracion_proponer','modulo_id','administracion','tipo_recurso','propuesta_restauracion_copia','finalidades',jsonb_build_array('proponer_restauracion_copia'),'garantia_minima','alto','campos_permitidos',jsonb_build_array('propuesta','recibo'))),
 'publicada_por','migracion:autorizacion:000024','publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
 IF vec_autorizacion.concesiones_positivas_validas(d) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: plantilla Sistemas invalida' USING ERRCODE='23514'; END IF;
 INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES('rol:operador_plataforma:v1','operador_plataforma',1,encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(d),'UTF8')),'hex'),ahora,d);
 c:=jsonb_build_object('version_rol_ref','rol:operador_plataforma:v1','revision',1,'estado','habilitada','actualizado_por','migracion:autorizacion:000024','actualizado_en',fecha);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES('rol:operador_plataforma:v1',1,'habilitada',encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(c),'UTF8')),'hex'),ahora,c);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES('rol:operador_plataforma:v1',1,ahora,'migracion:autorizacion:000024','migracion:autorizacion:000024');
 INSERT INTO vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase,huella_sha256) SELECT version_rol_ref,'administrador',huella_sha256 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:operador_plataforma:v1';
END $rol_sistemas_v1$;
CREATE TABLE vec_autorizacion.rol_administrable_exacto_v1(
 version_rol_ref text PRIMARY KEY REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 clase text NOT NULL CHECK(clase IN ('ordinario','administrador','intervencion')),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz NOT NULL,
 vigente_hasta timestamptz NOT NULL,
 unidad_requerida boolean NOT NULL,
 audiencia_administrativa text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(audiencia_administrativa,256) IS TRUE),
 ambitos_fijos jsonb NOT NULL CHECK(jsonb_typeof(ambitos_fijos)='array'),
 duracion_propuesta interval NOT NULL CHECK(duracion_propuesta>interval '0 seconds'),
 categoria_admin text NOT NULL CHECK(categoria_admin IN ('aplicacion','sistemas','')),
 CHECK((clase='administrador' AND categoria_admin IN ('aplicacion','sistemas')) OR (clase<>'administrador' AND categoria_admin='')),
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
-- Este catálogo registra referencias publicadas; nunca publica concesiones.
-- Dietas entra por su rol fijo AUT27 y una huella verificada, no por prefijos.
-- No se insertan valores de ejemplo en la autoridad de permisos.
CREATE TABLE vec_autorizacion.vinculacion_motivo_admin_v1(
 accion text NOT NULL CHECK(accion IN('administracion.perfiles.consultar','administracion.perfiles.proponer','administracion.perfiles.aprobar','administracion.perfiles.rechazar','administracion.perfiles.otorgar','administracion.perfiles.revocar','administracion.perfiles.historial.consultar','administracion.perfiles.recibo.consultar')),
 tipo_recurso text NOT NULL CHECK(tipo_recurso IN('perfil','propuesta_perfil','historial_perfil','recibo_perfil')),
 finalidad text NOT NULL CHECK(finalidad='gestion_perfiles'),
 catalogo_id text NOT NULL,
 catalogo_version integer NOT NULL,
 catalogo_huella_sha256 text NOT NULL CHECK(catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 entrada_clave text NOT NULL,
 PRIMARY KEY(accion,tipo_recurso,finalidad),
 FOREIGN KEY(catalogo_id,catalogo_version,entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id,catalogo_version,entrada_clave)
);
CREATE TABLE vec_autorizacion.vinculacion_asignacion_admin_v1(
 asignacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 cuenta_ref text NOT NULL,
 vinculo_ref text NOT NULL,
 rol_huella_sha256 text NOT NULL CHECK(rol_huella_sha256 ~ '^[0-9a-f]{64}$')
);
CREATE TABLE vec_autorizacion.registro_acto_admin_v1(
 operacion_ref text PRIMARY KEY,
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 65536),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(material),'hex')),
 actor_persona_ref text NOT NULL,
 actor_perfil_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 resultado jsonb NOT NULL CHECK(jsonb_typeof(resultado)='object'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_autorizacion.outbox_acto_admin_v1(
 operacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.registro_acto_admin_v1(operacion_ref),
 evento text NOT NULL CHECK(evento IN ('perfil_otorgado','perfil_revocado','propuesta_registrada','propuesta_aprobada','propuesta_rechazada')),
 auditoria_ref text NOT NULL,
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en))
);
DO $historia$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['rol_administrable_exacto_v1','vinculacion_motivo_admin_v1','vinculacion_asignacion_admin_v1','registro_acto_admin_v1','outbox_acto_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor',t);
 END LOOP;
END $historia$;
CREATE FUNCTION vec_autorizacion.resolver_rol_administrable_v1(p_version_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record; instante timestamptz:=clock_timestamp();
BEGIN
 SELECT a.* INTO r FROM vec_autorizacion.rol_administrable_exacto_v1 a
 JOIN vec_autorizacion.version_rol v USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE a.version_rol_ref=p_version_ref AND a.huella_sha256=v.huella_sha256
 AND v.documento->>'estado'='publicada' AND c.estado='habilitada'
 AND instante>=a.vigente_desde AND instante<a.vigente_hasta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: rol no administrable' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('version_ref',r.version_rol_ref,'clase',r.clase,'huella_sha256',r.huella_sha256,
  'vigente_desde',r.vigente_desde,'vigente_hasta',r.vigente_hasta,'unidad_requerida',r.unidad_requerida);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_rol_administrable_v1(text) FROM PUBLIC;

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c record;pe record;pf record;v record; dc jsonb;dp jsonb;df jsonb:=null;dv jsonb:=null;ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: preimagen requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO STRICT c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones x USING(cuenta_ref,version) WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 SELECT x.* INTO STRICT pe FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version) WHERE a.persona_ref=p_persona FOR UPDATE OF a;
 SELECT x.* INTO pf FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version) WHERE a.perfil_ref=p_perfil FOR UPDATE OF a;
 IF FOUND THEN df:=jsonb_build_object('perfil_ref',pf.perfil_ref,'version',pf.version,'persona_ref',pf.persona_ref,'estado',pf.estado,'procedencia_ref',pf.procedencia_ref,'procedencia_version',pf.procedencia_version,'procedencia_huella_sha256',pf.procedencia_huella_sha256,'procedencia_autoridad',pf.procedencia_autoridad,'vigente_desde',pf.vigente_desde,'vigente_hasta',pf.vigente_hasta); END IF;
 SELECT x.* INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version) WHERE a.vinculo_ref=p_vinculo FOR UPDATE OF a;
 IF FOUND THEN dv:=jsonb_build_object('vinculo_ref',v.vinculo_ref,'version',v.version,'cuenta_ref',v.cuenta_ref,'persona_ref',v.persona_ref,'perfil_ref',v.perfil_ref,'estado',v.estado,'procedencia_ref',v.procedencia_ref,'procedencia_version',v.procedencia_version,'procedencia_huella_sha256',v.procedencia_huella_sha256,'procedencia_autoridad',v.procedencia_autoridad,'vigente_desde',v.vigente_desde,'vigente_hasta',v.vigente_hasta); END IF;
 IF c.estado<>'activo' OR pe.estado<>'activo' OR c.procedencia_autoridad<>'autoridad_maestra_acreditada' OR pe.procedencia_autoridad<>'autoridad_maestra_acreditada' OR ahora<GREATEST(c.vigente_desde,pe.vigente_desde) OR ahora>=LEAST(c.vigente_hasta,pe.vigente_hasta) THEN RAISE EXCEPTION 'AUT24: objetivo no acreditado' USING ERRCODE='42501'; END IF;
 dc:=jsonb_build_object('cuenta_ref',c.cuenta_ref,'version',c.version,'estado',c.estado,'procedencia_ref',c.procedencia_ref,'procedencia_version',c.procedencia_version,'procedencia_huella_sha256',c.procedencia_huella_sha256,'procedencia_autoridad',c.procedencia_autoridad,'vigente_desde',c.vigente_desde,'vigente_hasta',c.vigente_hasta);
 dp:=jsonb_build_object('persona_ref',pe.persona_ref,'version',pe.version,'estado',pe.estado,'procedencia_ref',pe.procedencia_ref,'procedencia_version',pe.procedencia_version,'procedencia_huella_sha256',pe.procedencia_huella_sha256,'procedencia_autoridad',pe.procedencia_autoridad,'vigente_desde',pe.vigente_desde,'vigente_hasta',pe.vigente_hasta);
 RETURN jsonb_build_object('cuenta',dc,'persona',dp,'perfil',df,'vinculo',dv);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(text,text,text,text),vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz),vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text),vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric) TO vec_autorizacion_propietario;
CREATE FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_reconciliacion_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_cuenta_ver numeric;v_persona_ver numeric;v_perfil_ver numeric;v_vinculo_ver numeric;c record;pe record;pf record;v record; dc jsonb;dp jsonb;df jsonb:=null;dv jsonb:=null;ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: reconciliacion requiere READ COMMITTED' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT a.version INTO v_cuenta_ver FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a WHERE a.cuenta_ref=p_cuenta FOR UPDATE;
 SELECT x.* INTO STRICT c FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones x WHERE x.cuenta_ref=p_cuenta AND x.version=v_cuenta_ver;
 SELECT a.version INTO v_perfil_ver FROM vec_contexto_actor_v1.perfil_actual a WHERE a.perfil_ref=p_perfil FOR UPDATE;
 SELECT x.* INTO pf FROM vec_contexto_actor_v1.perfil_versiones x WHERE x.perfil_ref=p_perfil AND x.version=v_perfil_ver;
 IF FOUND THEN df:=jsonb_build_object('perfil_ref',pf.perfil_ref,'version',pf.version,'persona_ref',pf.persona_ref,'estado',pf.estado,'procedencia_ref',pf.procedencia_ref,'procedencia_version',pf.procedencia_version,'procedencia_huella_sha256',pf.procedencia_huella_sha256,'procedencia_autoridad',pf.procedencia_autoridad,'vigente_desde',pf.vigente_desde,'vigente_hasta',pf.vigente_hasta); END IF;
 SELECT a.version INTO v_vinculo_ver FROM vec_contexto_actor_v1.vinculo_contexto_actual a WHERE a.vinculo_ref=p_vinculo FOR UPDATE;
 SELECT x.* INTO v FROM vec_contexto_actor_v1.vinculo_contexto_versiones x WHERE x.vinculo_ref=p_vinculo AND x.version=v_vinculo_ver;
 IF FOUND THEN dv:=jsonb_build_object('vinculo_ref',v.vinculo_ref,'version',v.version,'cuenta_ref',v.cuenta_ref,'persona_ref',v.persona_ref,'perfil_ref',v.perfil_ref,'estado',v.estado,'procedencia_ref',v.procedencia_ref,'procedencia_version',v.procedencia_version,'procedencia_huella_sha256',v.procedencia_huella_sha256,'procedencia_autoridad',v.procedencia_autoridad,'vigente_desde',v.vigente_desde,'vigente_hasta',v.vigente_hasta); END IF;
 SELECT a.version INTO v_persona_ver FROM vec_contexto_actor_v1.persona_actual a WHERE a.persona_ref=p_persona FOR UPDATE;
 SELECT x.* INTO STRICT pe FROM vec_contexto_actor_v1.persona_versiones x WHERE x.persona_ref=p_persona AND x.version=v_persona_ver;
 IF c.estado<>'activo' OR pe.estado<>'activo' OR c.procedencia_autoridad<>'autoridad_maestra_acreditada' OR pe.procedencia_autoridad<>'autoridad_maestra_acreditada' OR ahora<GREATEST(c.vigente_desde,pe.vigente_desde) OR ahora>=LEAST(c.vigente_hasta,pe.vigente_hasta) THEN RAISE EXCEPTION 'AUT24: objetivo no acreditado' USING ERRCODE='42501'; END IF;
 dc:=jsonb_build_object('cuenta_ref',c.cuenta_ref,'version',c.version,'estado',c.estado,'procedencia_ref',c.procedencia_ref,'procedencia_version',c.procedencia_version,'procedencia_huella_sha256',c.procedencia_huella_sha256,'procedencia_autoridad',c.procedencia_autoridad,'vigente_desde',c.vigente_desde,'vigente_hasta',c.vigente_hasta);
 dp:=jsonb_build_object('persona_ref',pe.persona_ref,'version',pe.version,'estado',pe.estado,'procedencia_ref',pe.procedencia_ref,'procedencia_version',pe.procedencia_version,'procedencia_huella_sha256',pe.procedencia_huella_sha256,'procedencia_autoridad',pe.procedencia_autoridad,'vigente_desde',pe.vigente_desde,'vigente_hasta',pe.vigente_hasta);
 RETURN jsonb_build_object('cuenta',dc,'persona',dp,'perfil',df,'vinculo',dv);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_reconciliacion_v1(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_reconciliacion_v1(text,text,text,text) TO vec_autorizacion_propietario;

RESET ROLE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
-- Población efectiva para continuidad. No sustituye la revalidación de sesión
-- de quien propone/aprueba: AD150 la exige por la fachada nominal IS12.
CREATE FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(p_persona text,p_cuenta text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;pol record;c record;
BEGIN
 SELECT * INTO STRICT pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 WHERE singleton FOR SHARE;
 SELECT v.vinculo_ref,v.version,v.estado,v.vigente_desde,v.vigente_hasta INTO STRICT b
 FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
 WHERE v.persona_ref=p_persona AND v.cuenta_privilegiada_ref=p_cuenta AND v.politica_ref=pol.politica_ref AND v.ca_sha256=pol.ca_sha256 FOR SHARE OF a;
 SELECT pc.cuenta_privilegiada,pc.cuenta_ordinaria_ref,s.estado,s.revision,so.estado AS estado_ordinaria,so.revision AS revision_ordinaria INTO STRICT c
 FROM vec_identidad_sesiones_v1.cuenta pc
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
 JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ao ON ao.cuenta_ref=pc.cuenta_ordinaria_ref
 JOIN vec_identidad_sesiones_v1.estado_cuenta so ON so.cuenta_ref=ao.cuenta_ref AND so.revision=ao.revision
 WHERE pc.cuenta_ref=p_cuenta FOR SHARE OF a,ao;
 IF NOT pol.activa OR clock_timestamp()>=pol.vigente_hasta OR b.estado<>'activo' OR clock_timestamp()<b.vigente_desde OR clock_timestamp()>=b.vigente_hasta OR NOT c.cuenta_privilegiada OR c.estado<>'activa' OR c.estado_ordinaria<>'activa' THEN RETURN null; END IF;
 RETURN jsonb_build_object('persona_ref',p_persona,'cuenta_ref',p_cuenta,'cuenta_revision',c.revision,'cuenta_ordinaria_ref',c.cuenta_ordinaria_ref,'cuenta_ordinaria_revision',c.revision_ordinaria,'certificado_vinculo_ref',b.vinculo_ref,'certificado_vinculo_version',b.version,'politica_ref',pol.politica_ref,'politica_huella',pol.huella_aprobacion_sha256,'vigente_desde',b.vigente_desde,'vigente_hasta',LEAST(b.vigente_hasta,pol.vigente_hasta));
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN null;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(text,text) TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.administradores_efectivos_internos_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE a record;ca jsonb; identidad jsonb;resultado jsonb:='[]'::jsonb;
BEGIN
 FOR a IN SELECT x.*,va.cuenta_ref,va.vinculo_ref,va.rol_huella_sha256 FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref) JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref)
 JOIN vec_autorizacion.rol_sensible_exacto s USING(version_rol_ref) JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=rc.version_rol_ref AND c.revision=rc.revision
 WHERE s.clase='administrador' AND r.rol_id='administracion_perfiles' AND x.documento->>'estado'='activa' AND s.huella_sha256=r.huella_sha256 AND c.estado='habilitada'
 AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz ORDER BY x.principal_id,x.perfil_activo_ref FOR UPDATE OF ac LOOP
  ca:=vec_contexto_actor_v1.preimagen_admin_interna_v1(a.cuenta_ref,a.principal_id,a.perfil_activo_ref,a.vinculo_ref);
  identidad:=vec_identidad_sesiones_v1.estado_admin_interno_v1(a.principal_id,a.cuenta_ref);
  IF identidad IS NOT NULL AND ca#>>'{perfil,estado}'='activo' AND ca#>>'{vinculo,estado}'='activo'
  AND ca#>>'{perfil,persona_ref}'=a.principal_id AND ca#>>'{vinculo,persona_ref}'=a.principal_id
  AND clock_timestamp()>=(ca#>>'{perfil,vigente_desde}')::timestamptz AND clock_timestamp()<(ca#>>'{perfil,vigente_hasta}')::timestamptz
  AND clock_timestamp()>=(ca#>>'{vinculo,vigente_desde}')::timestamptz AND clock_timestamp()<(ca#>>'{vinculo,vigente_hasta}')::timestamptz THEN
   resultado:=resultado||jsonb_build_array(jsonb_build_object('persona_ref',a.principal_id,'perfil_ref',a.perfil_activo_ref,'asignacion_ref',a.asignacion_ref,'asignacion_version',a.version,'asignacion_huella',a.huella_sha256,'contexto',ca,'identidad',identidad));
  END IF;
 END LOOP;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_efectivos_internos_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE o jsonb:=p_material->'objetivo';c jsonb;r record;cv record;a record;continuidad record;actual_asignacion jsonb:=null;efectivos jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT continuidad FROM vec_autorizacion.control_continuidad_admin WHERE control_id FOR UPDATE;
 c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(o->>'cuenta_ref',o->>'persona_ref',o->>'perfil_ref',o->>'vinculo_ref');
 SELECT v.* INTO STRICT r FROM vec_autorizacion.version_rol v WHERE version_rol_ref=p_material->>'rol_version_ref';
 SELECT v.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol_actual a JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision) WHERE a.version_rol_ref=r.version_rol_ref FOR UPDATE OF a;
 SELECT v.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil v USING(asignacion_ref) WHERE ac.perfil_activo_ref=o->>'perfil_ref' FOR UPDATE OF ac;
 IF FOUND THEN actual_asignacion:=jsonb_build_object('asignacion_ref',a.asignacion_ref,'version',a.version,'huella_sha256',a.huella_sha256,'estado',a.documento->>'estado','ambitos',a.documento->'ambitos','version_rol_ref',a.version_rol_ref,'principal_id',a.principal_id); END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 RETURN jsonb_build_object('esquema','vec.admin.perfiles.preimagen.v1','operacion',p_material->>'operacion','rol_version_ref',r.version_rol_ref,'rol_huella_sha256',r.huella_sha256,'control_revision',cv.revision,'control_huella_sha256',cv.huella_sha256,'contexto',c,'asignacion',actual_asignacion,'unidad_ref',p_material->>'unidad_ref','objetivo',o-'huella_sha256','continuidad',jsonb_build_object('revision',continuidad.revision,'bootstrap_estado',continuidad.bootstrap_estado,'minimo_personas',continuidad.minimo_personas,'bootstrap_minimo_personas',continuidad.bootstrap_minimo_personas),'administradores_efectivos',efectivos);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_cambio_admin_interna_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.motivo_admin_publicado_interno_v1(p_accion text,p_tipo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m record;
BEGIN
 SELECT v.* INTO STRICT m FROM vec_autorizacion.vinculacion_motivo_admin_v1 v
 JOIN vec_autorizacion.motivo_v2_catalogo_publicado c USING(catalogo_id,catalogo_version)
 JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version,entrada_clave)
 WHERE v.accion=p_accion AND v.tipo_recurso=p_tipo AND v.finalidad='gestion_perfiles'
 AND v.catalogo_huella_sha256=c.catalogo_huella_publicada_sha256
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r WHERE r.catalogo_id=v.catalogo_id AND r.catalogo_version=v.catalogo_version)
 AND clock_timestamp()>=e.vigente_desde AND(e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta);
 RETURN jsonb_build_object('catalogo_id',m.catalogo_id,'catalogo_version',m.catalogo_version,'catalogo_huella_sha256',m.catalogo_huella_sha256,'entrada_clave',m.entrada_clave);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.motivo_admin_publicado_interno_v1(text,text) FROM PUBLIC;

-- Valida el DTO de negocio, nunca utiliza su actor como prueba de identidad.
CREATE FUNCTION vec_autorizacion.validar_material_acto_admin_v1(p_material text,p_propuesta boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; o jsonb; r jsonb; claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
  RAISE EXCEPTION 'AUT24: material invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb; o:=m->'objetivo';
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_acto_v1'
 OR NOT m ?& ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','correlacion_ref']
 OR jsonb_path_exists(m,'$.** ? (@ == null)')
 OR claves <@ ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','correlacion_ref','unidad_ref','referencia_acto'] IS NOT TRUE
 OR jsonb_typeof(o) IS DISTINCT FROM 'object'
 OR jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object'
 OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']
 OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE
 OR (m#>>'{motivo,catalogo_version}')::numeric<1
 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$'
 OR m->>'operacion' NOT IN ('otorgar','revocar')
 OR (CASE WHEN p_propuesta THEN m->>'operacion_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' ELSE m->>'operacion_ref' !~ '^acto_admin:[0-9a-f]{32}$' END)
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_persona_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_perfil_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'correlacion_ref',512) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT24: DTO de acto invalido' USING ERRCODE='22023'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 IF r->>'huella_sha256' IS DISTINCT FROM m->>'rol_huella_sha256'
 OR (p_propuesta AND r->>'clase' NOT IN ('administrador','intervencion'))
 OR (NOT p_propuesta AND r->>'clase' IS DISTINCT FROM 'ordinario')
 OR ((r->>'unidad_requerida')::boolean AND vec_autorizacion.texto_positivo_valido(m->>'unidad_ref',512) IS NOT TRUE)
 OR (NOT (r->>'unidad_requerida')::boolean AND m ? 'unidad_ref')
 OR (m ? 'referencia_acto' AND vec_autorizacion.texto_positivo_valido(m->>'referencia_acto',512) IS NOT TRUE)
 OR (m->>'actor_persona_ref' IS NOT DISTINCT FROM o->>'persona_ref'
  AND NOT(p_propuesta AND r->>'clase'='administrador' AND m->>'operacion'='revocar')) THEN
  RAISE EXCEPTION 'AUT24: rol, unidad o independencia invalidos' USING ERRCODE='42501'; END IF;
 IF NOT o ?& ARRAY['cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','perfil_version','vinculo_ref','vinculo_version','huella_sha256','revision_continuidad','procedencia_ref','procedencia_version','procedencia_huella_sha256','vigente_hasta']
 OR (SELECT count(*) FROM jsonb_object_keys(o))<>14
 OR o->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR o->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR o->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR o->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR o->>'huella_sha256' !~ '^[0-9a-f]{64}$'
 OR o->>'procedencia_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR (o->>'cuenta_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'persona_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'procedencia_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (m->>'operacion'='otorgar' AND ((o->>'perfil_version')::numeric<>0 OR (o->>'vinculo_version')::numeric<>0 OR NOT isfinite((o->>'vigente_hasta')::timestamptz)))
 OR (m->>'operacion'='revocar' AND ((o->>'perfil_version')::numeric<1 OR (o->>'vinculo_version')::numeric<1 OR o->>'vigente_hasta'<>'0001-01-01T00:00:00Z')) THEN
  RAISE EXCEPTION 'AUT24: preimagen estructural invalida' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_material_acto_admin_v1(text,boolean) FROM PUBLIC;

-- Esta huella es la del contexto RecursoAutorizable común de Go.
-- La huella de preimagen SQL-only usa otro algoritmo y no se modifica.
CREATE FUNCTION vec_autorizacion.huella_contexto_material_admin_v1(p_ambitos jsonb,p_material_sha256 text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE contenido text;
BEGIN
 IF jsonb_typeof(p_ambitos) IS DISTINCT FROM 'object' OR p_material_sha256 IS NULL OR p_material_sha256 !~ '^[0-9a-f]{64}$' OR EXISTS(SELECT 1 FROM jsonb_each(p_ambitos) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string') THEN RAISE EXCEPTION 'AUT24: contexto comun invalido' USING ERRCODE='22023'; END IF;
 SELECT coalesce(string_agg(vec_autorizacion.json_cadena_canonica_go_admin_v1(key)||':'||vec_autorizacion.json_cadena_canonica_go_admin_v1(value#>>'{}'),',' ORDER BY key COLLATE "C"),'') INTO contenido FROM jsonb_each(p_ambitos);
 RETURN encode(sha256(convert_to('{"ambitos":{'||contenido||'},"atributos":{"material_sha256":"'||p_material_sha256||'"}}','UTF8')),'hex');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.huella_contexto_material_admin_v1(jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.ambitos_registro_admin_interna_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE cfg record;ref text;
BEGIN
 SELECT convert_from(b.plan_canonico,'UTF8')::jsonb#>>'{rol,version_ref}' INTO STRICT ref
 FROM vec_autorizacion.control_continuidad_admin c JOIN vec_autorizacion.bootstrap_admin_v2 b ON b.acto_ref=c.bootstrap_acto_ref
 WHERE c.control_id AND c.bootstrap_estado='consumido';
 SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=ref;
 PERFORM vec_autorizacion.resolver_rol_administrable_v1(ref);
 IF cfg.clase IS DISTINCT FROM 'administrador' OR cfg.unidad_requerida OR cfg.ambitos_fijos IS DISTINCT FROM '[{"clave":"administracion","valores":["perfiles"]}]'::jsonb THEN RAISE EXCEPTION 'AUT24: registro administrativo sin ambito explicito publicado' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('administracion',cfg.ambitos_fijos#>>'{0,valores,0}');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ambitos_registro_admin_interna_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.resolver_recurso_admin_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;actor record;cfg record;propuesta record;registro record;material_objetivo jsonb;referencia_rol text;unidad text;tipo text:='perfil';ambitos jsonb:='{}'::jsonb;e jsonb;huella text;referencia text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: material de contexto invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;referencia:=m->>'operacion_ref';
 IF vec_autorizacion.texto_positivo_valido(referencia,512) IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m->>'actor_persona_ref',512) IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m->>'actor_perfil_ref',512) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: referencias de contexto invalidas' USING ERRCODE='22023'; END IF;
 SELECT x.* INTO STRICT actor FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref)
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref) JOIN vec_autorizacion.rol_sensible_exacto sx USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref) JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=rc.version_rol_ref AND cv.revision=rc.revision
 WHERE x.principal_id=m->>'actor_persona_ref' AND x.perfil_activo_ref=m->>'actor_perfil_ref' AND x.documento->>'estado'='activa' AND r.rol_id='administracion_perfiles' AND r.huella_sha256=sx.huella_sha256 AND sx.clase='administrador' AND r.documento->>'estado'='publicada' AND cv.estado='habilitada' AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_estado='consumido') THEN RAISE EXCEPTION 'AUT24: bootstrap pendiente' USING ERRCODE='42501'; END IF;
 CASE m->>'esquema'
 WHEN 'administracion_perfiles_acto_v1' THEN referencia_rol:=m->>'rol_version_ref';unidad:=m->>'unidad_ref';
 WHEN 'administracion_perfiles_preimagen_v1' THEN referencia_rol:=m->>'rol_version_ref';unidad:=m->>'unidad_ref';
 WHEN 'administracion_perfiles_cierre_v1' THEN
  SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=m->>'propuesta_ref';
  IF propuesta.huella_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256' THEN RAISE EXCEPTION 'AUT24: propuesta de contexto divergente' USING ERRCODE='40001'; END IF;
  referencia_rol:=propuesta.version_rol_ref;unidad:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb->>'unidad_ref';tipo:='propuesta_perfil';
 WHEN 'administracion_perfiles_lectura_v1' THEN
  referencia_rol:=actor.version_rol_ref;
  IF m->>'consulta' IN ('listar_propuestas','consultar_propuesta') THEN
   tipo:='historial_perfil';
   IF m->>'consulta'='consultar_propuesta' THEN
    SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=m->>'propuesta_ref';
    referencia_rol:=propuesta.version_rol_ref;unidad:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb->>'unidad_ref';
   END IF;
  ELSIF m->>'consulta'='consultar_recibo' THEN
   tipo:='recibo_perfil';
   SELECT * INTO STRICT registro FROM vec_autorizacion.registro_acto_admin_v1 WHERE resultado->>'recibo_ref'=m->>'recibo_ref' OR resultado#>>'{recibo,recibo_ref}'=m->>'recibo_ref';
   material_objetivo:=convert_from(registro.material,'UTF8')::jsonb;
   IF material_objetivo->>'esquema'='administracion_perfiles_cierre_v1' THEN SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=material_objetivo->>'propuesta_ref'; material_objetivo:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb; END IF;
   referencia_rol:=material_objetivo->>'rol_version_ref';unidad:=material_objetivo->>'unidad_ref';
  END IF;
 ELSE RAISE EXCEPTION 'AUT24: esquema de contexto no admitido' USING ERRCODE='22023';
 END CASE;
 SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=referencia_rol;
 PERFORM vec_autorizacion.resolver_rol_administrable_v1(referencia_rol);
 ambitos:=vec_autorizacion.ambitos_registro_admin_interna_v1();
 huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 RETURN jsonb_build_object('referencia',referencia,'modulo_id','administracion','tipo',tipo,'ambitos',ambitos,'atributos',jsonb_build_object('material_sha256',huella));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_recurso_admin_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_recurso_admin_v1(text) TO vec_admin_perfiles_ejecutor;

-- Primitiva compartida: consumo V3 antes de toda recuperación o cambio.
-- p_contexto sigue siendo material central; no se reconstruye desde JSON HTTP.
CREATE FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb:=p_material::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb; c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; x record; motivo_publicado jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'AUT24: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF c->>'efecto_ref' IS DISTINCT FROM m->>'operacion_ref'
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM vec_autorizacion.huella_contexto_material_admin_v1(vec_autorizacion.resolver_recurso_admin_v1(p_material)->'ambitos',encode(sha256(convert_to(p_material,'UTF8')),'hex'))
 OR d->>'principal_id' IS DISTINCT FROM m->>'actor_persona_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'actor_perfil_ref'
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true' THEN
  RAISE EXCEPTION 'AUT24: sello de negocio divergente' USING ERRCODE='42501'; END IF;
 motivo_publicado:=vec_autorizacion.motivo_admin_publicado_interno_v1(d->>'accion',d->>'tipo_recurso');
 IF m ? 'motivo' AND m->'motivo' IS DISTINCT FROM motivo_publicado THEN RAISE EXCEPTION 'AUT24: motivo administrativo no publicado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- AD150 exige IS12 nominal dentro de la misma transacción, también replay.
 RETURN jsonb_build_object('decision_ref',x.decision_ref,'auditoria_ref',x.auditoria_ref,'consumida_en',x.consumida_en,'consumo_nuevo',x.consumo_nuevo);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE TABLE vec_autorizacion.sello_efecto_admin_tx_v1(
 asignacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 transaccion bigint NOT NULL,
 operacion_ref text NOT NULL,
 auditoria_ref text NOT NULL
);
ALTER TABLE vec_autorizacion.sello_efecto_admin_tx_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.sello_efecto_admin_tx_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR ALL TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER sello_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER sello_no_truncar BEFORE TRUNCATE ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.sello_efecto_admin_tx_v1 FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor;
REVOKE ALL ON TYPE vec_autorizacion.sello_efecto_admin_tx_v1 FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor;
-- Reemplaza sólo la guarda prospectiva AUT23 por un sello nominal indivisible.
-- Una variable de sesión editable o SET ROLE no puede fabricar este sello.
CREATE OR REPLACE FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE rol text;referencia text;
BEGIN
 SELECT r.rol_id,r.version_rol_ref INTO STRICT rol,referencia FROM vec_autorizacion.asignacion_perfil a JOIN vec_autorizacion.version_rol r USING(version_rol_ref) WHERE a.asignacion_ref=NEW.asignacion_ref;
 IF (rol='administracion_perfiles' OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto WHERE version_rol_ref=referencia))
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.sello_efecto_admin_tx_v1 WHERE asignacion_ref=NEW.asignacion_ref AND transaccion=txid_current()) THEN
  RAISE EXCEPTION 'AUT24: asignacion sensible sin acto atestado' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.ejecutar_cambio_admin_interno_v1(p_material jsonb,p_operacion_ref text,p_propuesta_ref text,p_auditoria text,p_ejecutor_ref text,p_bootstrap boolean,p_decision_ref text,p_material_sha256 text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE o jsonb:=p_material->'objetivo';configuracion record;antes jsonb;despues jsonb;r jsonb;a record;doc jsonb;
 huella text;asig_id text;asig_ref text;ver bigint;ahora timestamptz;acto text;recibo text;estado text;version_ca numeric;ambitos jsonb;hdespues text;procedencia_acto jsonb;rol_id_destino text;
BEGIN
 IF p_bootstrap IS NULL OR (p_bootstrap AND (p_material->>'operacion' IS DISTINCT FROM 'otorgar')) THEN RAISE EXCEPTION 'AUT24: contexto privado de efecto invalido' USING ERRCODE='22023'; END IF;
 antes:=vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material);
 huella:=encode(sha256(convert_to(antes::text,'UTF8')),'hex');
 IF NOT p_bootstrap AND huella IS DISTINCT FROM o->>'huella_sha256' THEN RAISE EXCEPTION 'AUT24: preimagen_huella esperado=% observado=%',o->>'huella_sha256',huella USING ERRCODE='40001'; END IF;
 IF (antes#>>'{contexto,cuenta,version}')::numeric IS DISTINCT FROM (o->>'cuenta_version')::numeric
 OR (antes#>>'{contexto,persona,version}')::numeric IS DISTINCT FROM (o->>'persona_version')::numeric
 OR (NOT p_bootstrap AND antes#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'consumido')
 OR (p_bootstrap AND (antes#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'pendiente' OR rtrim(o->>'revision_continuidad') IS DISTINCT FROM '0')) THEN
  RAISE EXCEPTION 'AUT24: CAS de objetivo o bootstrap divergente' USING ERRCODE='40001'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(p_material->>'rol_version_ref');
 SELECT vr.rol_id INTO STRICT rol_id_destino FROM vec_autorizacion.version_rol vr WHERE vr.version_rol_ref=p_material->>'rol_version_ref';
 IF r->>'huella_sha256' IS DISTINCT FROM p_material->>'rol_huella_sha256' THEN RAISE EXCEPTION 'AUT24: rol no vigente' USING ERRCODE='42501'; END IF;
 IF NOT p_bootstrap AND r->>'clase'<>'ordinario' AND ((antes#>>'{continuidad,revision}')::bigint IS DISTINCT FROM (o->>'revision_continuidad')::bigint
 OR (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(antes->'administradores_efectivos') x)<2) THEN
  RAISE EXCEPTION 'AUT24: doble control o continuidad divergente' USING ERRCODE='40001'; END IF;
 ahora:=clock_timestamp();acto:='acto_admin:'||substr(encode(sha256(convert_to(p_operacion_ref,'UTF8')),'hex'),1,32);
 recibo:='recibo_admin:'||substr(encode(sha256(convert_to(p_operacion_ref||':recibo','UTF8')),'hex'),1,32);
 IF p_material->>'operacion'='otorgar' THEN
  IF r->>'clase'='administrador' AND (vec_identidad_sesiones_v1.estado_admin_interno_v1(o->>'persona_ref',o->>'cuenta_ref') IS NULL OR (rol_id_destino='administracion_perfiles' AND EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'administradores_efectivos') x WHERE x->>'persona_ref'=o->>'persona_ref'))) THEN RAISE EXCEPTION 'AUT24: alta de administrador sin identidad privilegiada unica' USING ERRCODE='42501'; END IF;
  IF antes#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb OR antes#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb OR antes->'asignacion' IS DISTINCT FROM 'null'::jsonb THEN RAISE EXCEPTION 'AUT24: alta requiere referencias nuevas' USING ERRCODE='40001'; END IF;
  PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(o->>'cuenta_ref',o->>'persona_ref',(o->>'cuenta_version')::numeric,(o->>'persona_version')::numeric,o->>'perfil_ref',o->>'vinculo_ref',o->>'procedencia_ref',(o->>'procedencia_version')::numeric,o->>'procedencia_huella_sha256',(o->>'vigente_hasta')::timestamptz);
  version_ca:=1;estado:='activo';ver:=1;asig_id:='admin_'||substr(encode(sha256(convert_to(p_operacion_ref,'UTF8')),'hex'),1,32);
  IF (r->>'unidad_requerida')::boolean THEN ambitos:=jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(p_material->>'unidad_ref')));
  ELSE SELECT * INTO STRICT configuracion FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=p_material->>'rol_version_ref'; ambitos:=configuracion.ambitos_fijos; IF vec_autorizacion.ambitos_positivos_validos(jsonb_build_object('ambitos',ambitos)) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: ambito fijo no publicado' USING ERRCODE='42501'; END IF; END IF;
  doc:=jsonb_build_object('asignacion_id',asig_id,'version',ver,'perfil_activo_ref',o->>'perfil_ref','principal_id',o->>'persona_ref','version_rol_ref',p_material->>'rol_version_ref','estado','activa','ambitos',ambitos,'emitida_por',p_ejecutor_ref,'emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',to_char((o->>'vigente_hasta')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 ELSE
  IF antes#>>'{contexto,perfil,persona_ref}' IS DISTINCT FROM o->>'persona_ref' OR antes#>>'{contexto,vinculo,persona_ref}' IS DISTINCT FROM o->>'persona_ref' OR antes#>>'{contexto,vinculo,cuenta_ref}' IS DISTINCT FROM o->>'cuenta_ref' OR antes#>>'{contexto,vinculo,perfil_ref}' IS DISTINCT FROM o->>'perfil_ref'
  OR (antes#>>'{contexto,perfil,version}')::numeric IS DISTINCT FROM (o->>'perfil_version')::numeric OR (antes#>>'{contexto,vinculo,version}')::numeric IS DISTINCT FROM (o->>'vinculo_version')::numeric
  OR antes#>>'{asignacion,version_rol_ref}' IS DISTINCT FROM p_material->>'rol_version_ref' OR antes#>>'{asignacion,estado}' IS DISTINCT FROM 'activa' THEN RAISE EXCEPTION 'AUT24: baja sin vinculo y asignacion exactos' USING ERRCODE='40001'; END IF;
  IF (r->>'unidad_requerida')::boolean AND antes#>'{asignacion,ambitos}' IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(p_material->>'unidad_ref'))) THEN RAISE EXCEPTION 'AUT24: unidad de baja divergente' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT a FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=antes#>>'{asignacion,asignacion_ref}';
  IF o->>'procedencia_ref' IS DISTINCT FROM antes#>>'{contexto,perfil,procedencia_ref}' OR (o->>'procedencia_version')::numeric IS DISTINCT FROM (antes#>>'{contexto,perfil,procedencia_version}')::numeric OR o->>'procedencia_huella_sha256' IS DISTINCT FROM antes#>>'{contexto,perfil,procedencia_huella_sha256}' THEN RAISE EXCEPTION 'AUT24: procedencia actual de perfil divergente' USING ERRCODE='40001'; END IF;
  procedencia_acto:=vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1(acto,p_operacion_ref,p_material_sha256,p_decision_ref,p_auditoria,o->>'perfil_ref',o->>'vinculo_ref',(o->>'perfil_version')::numeric,(o->>'vinculo_version')::numeric,'revocado');
  PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(o->>'cuenta_ref',o->>'persona_ref',o->>'perfil_ref',o->>'vinculo_ref',(o->>'cuenta_version')::numeric,(o->>'persona_version')::numeric,(o->>'perfil_version')::numeric,(o->>'vinculo_version')::numeric,procedencia_acto->>'procedencia_ref',(procedencia_acto->>'procedencia_version')::numeric,procedencia_acto->>'procedencia_huella_sha256');
  version_ca:=(o->>'perfil_version')::numeric+1;estado:='revocado';ver:=a.version+1;asig_id:=a.asignacion_id;
  doc:=jsonb_set(jsonb_set(a.documento,'{version}',to_jsonb(ver)),'{estado}','"revocada"'::jsonb)||jsonb_build_object('revocada_por',p_ejecutor_ref,'revocada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'revocacion_ref',acto);
 END IF;
 asig_ref:='asignacion:'||asig_id||':v'||ver;
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(asig_ref,asig_id,ver,o->>'perfil_ref',o->>'persona_ref',p_material->>'rol_version_ref',encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex'),(doc->>'emitida_en')::timestamptz,doc);
 INSERT INTO vec_autorizacion.vinculacion_asignacion_admin_v1 VALUES(asig_ref,o->>'cuenta_ref',o->>'vinculo_ref',p_material->>'rol_huella_sha256');
 INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(asig_ref,txid_current(),p_operacion_ref,p_auditoria);
 IF p_material->>'operacion'='otorgar' THEN INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES(o->>'perfil_ref',asig_ref,ahora,p_ejecutor_ref,acto);
 ELSE UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=asig_ref,actualizada_en=ahora,actualizada_por=p_ejecutor_ref,acto_ref=acto WHERE perfil_activo_ref=o->>'perfil_ref' AND asignacion_ref=antes#>>'{asignacion,asignacion_ref}'; IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: CAS de asignacion perdido' USING ERRCODE='40001'; END IF; END IF;
 IF NOT p_bootstrap AND r->>'clase'<>'ordinario' THEN
  PERFORM vec_autorizacion.avanzar_continuidad_admin_interna_v1((o->>'revision_continuidad')::bigint);
  IF (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)<1 THEN RAISE EXCEPTION 'AUT24: ultimo administrador no revocable' USING ERRCODE='42501'; END IF;
 END IF;
 despues:=vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material);
 hdespues:=encode(sha256(convert_to(despues::text,'UTF8')),'hex');
 RETURN jsonb_build_object('operacion_ref',p_operacion_ref,'acto_ref',acto,'recibo_ref',recibo,'propuesta_ref',coalesce(p_propuesta_ref,''),'auditoria_ref',p_auditoria,'objetivo_persona_ref',o->>'persona_ref','perfil_ref',o->>'perfil_ref','vinculo_ref',o->>'vinculo_ref','estado_posterior',estado,'version_posterior',version_ca,'huella_antes_sha256',huella,'huella_despues_sha256',hdespues,'confirmado_en',ahora,'unidad_ref',coalesce(p_material->>'unidad_ref',''),'referencia_acto',coalesce(p_material->>'referencia_acto',''));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ejecutar_cambio_admin_interno_v1(jsonb,text,text,text,text,boolean,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.registrar_resultado_admin_interno_v1(p_material text,p_consumo jsonb,p_resultado jsonb,p_evento text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb:=p_material::jsonb;
BEGIN
 INSERT INTO vec_autorizacion.registro_acto_admin_v1 VALUES(m->>'operacion_ref',convert_to(p_material,'UTF8'),encode(sha256(convert_to(p_material,'UTF8')),'hex'),m->>'actor_persona_ref',m->>'actor_perfil_ref',p_consumo->>'decision_ref',p_consumo->>'auditoria_ref',p_resultado,clock_timestamp());
 INSERT INTO vec_autorizacion.outbox_acto_admin_v1 VALUES(m->>'operacion_ref',p_evento,p_consumo->>'auditoria_ref',clock_timestamp());
 RETURN p_resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_resultado_admin_interno_v1(text,jsonb,jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record;m jsonb:=p_material::jsonb;
BEGIN
 SELECT * INTO r FROM vec_autorizacion.registro_acto_admin_v1 WHERE operacion_ref=m->>'operacion_ref';
 IF NOT FOUND THEN RETURN null; END IF;
 IF (convert_from(r.material,'UTF8')::jsonb-'correlacion_ref') IS DISTINCT FROM (m-'correlacion_ref') THEN RAISE EXCEPTION 'AUT24: idempotencia divergente' USING ERRCODE='23505'; END IF;
 RETURN r.resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recuperar_resultado_admin_interno_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;consumo jsonb;resultado jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;efectivos jsonb;
BEGIN
 m:=vec_autorizacion.validar_material_acto_admin_v1(p_material,false);
 IF d->>'accion' IS DISTINCT FROM 'administracion.perfiles.'||(m->>'operacion') THEN RAISE EXCEPTION 'AUT24: accion ordinaria divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: administrador no efectivo' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 resultado:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(m,m->>'operacion_ref',null,consumo->>'auditoria_ref',m->>'actor_persona_ref',false,consumo->>'decision_ref',encode(sha256(convert_to(p_material,'UTF8')),'hex'));
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,CASE WHEN m->>'operacion'='otorgar' THEN 'perfil_otorgado' ELSE 'perfil_revocado' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.proponer_acto_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;consumo jsonb;resultado jsonb;preimagen jsonb;rol jsonb;huella text;caduca timestamptz;ahora timestamptz;configuracion record;efectivos jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 m:=vec_autorizacion.validar_material_acto_admin_v1(p_material,true);
 IF d->>'accion' IS DISTINCT FROM 'administracion.perfiles.proponer' THEN RAISE EXCEPTION 'AUT24: accion propuesta divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: doble control no disponible' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 preimagen:=vec_autorizacion.preimagen_cambio_admin_interna_v1(m);
 huella:=encode(sha256(convert_to(preimagen::text,'UTF8')),'hex');
 IF huella IS DISTINCT FROM m#>>'{objetivo,huella_sha256}' OR (preimagen#>>'{continuidad,revision}')::bigint IS DISTINCT FROM (m#>>'{objetivo,revision_continuidad}')::bigint OR preimagen#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'consumido' THEN RAISE EXCEPTION 'AUT24: preimagen de propuesta divergente' USING ERRCODE='40001'; END IF;
 rol:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 SELECT * INTO STRICT configuracion FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=m->>'rol_version_ref';
 ahora:=clock_timestamp();caduca:=LEAST(ahora+configuracion.duracion_propuesta,configuracion.vigente_hasta);
 INSERT INTO vec_autorizacion.propuesta_perfil_sensible(propuesta_ref,clase,operacion,version_rol_ref,objetivo_persona_ref,objetivo_cuenta_ref,objetivo_perfil_ref,proponente_persona_ref,proponente_cuenta_ref,proponente_perfil_ref,motivo_codigo,revision_continuidad_esperada,preimagen_huella_sha256,documento_canonico,huella_sha256,creada_en,caduca_en)
 VALUES(m->>'operacion_ref',rol->>'clase',m->>'operacion',m->>'rol_version_ref',m#>>'{objetivo,persona_ref}',m#>>'{objetivo,cuenta_ref}',m#>>'{objetivo,perfil_ref}',m->>'actor_persona_ref',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',m->>'actor_perfil_ref',m#>>'{motivo,entrada_clave}',(m#>>'{objetivo,revision_continuidad}')::bigint,huella,convert_to(p_material,'UTF8'),encode(sha256(convert_to(p_material,'UTF8')),'hex'),ahora,caduca);
 resultado:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',m->>'operacion_ref','huella_sha256',encode(sha256(convert_to(p_material,'UTF8')),'hex'),'proponente_persona_ref',m->>'actor_persona_ref','objetivo_persona_ref',m#>>'{objetivo,persona_ref}','caduca_en',caduca);
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,'propuesta_registrada');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb:=p_material::jsonb;consumo jsonb;resultado jsonb;propuesta record;material_propuesta jsonb;rol jsonb;revision bigint;efectivos jsonb;huella text;ahora timestamptz;recibo jsonb:=null;acto jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: cierre invalido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[] OR jsonb_path_exists(m,'$.** ? (@ == null)') OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_cierre_v1' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$' OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'decision' NOT IN ('aprobada','rechazada') THEN RAISE EXCEPTION 'AUT24: DTO cierre invalido' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object' OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'] OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE OR (m#>>'{motivo,catalogo_version}')::numeric<1 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'AUT24: motivo de cierre invalido' USING ERRCODE='22023'; END IF;
 IF d->>'accion' IS DISTINCT FROM (CASE WHEN m->>'decision'='aprobada' THEN 'administracion.perfiles.aprobar' ELSE 'administracion.perfiles.rechazar' END) THEN RAISE EXCEPTION 'AUT24: accion cierre divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: aprobador no efectivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=m->>'propuesta_ref' FOR SHARE;
 IF propuesta.huella_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256' OR m->>'actor_persona_ref' IN (propuesta.proponente_persona_ref,propuesta.objetivo_persona_ref) THEN RAISE EXCEPTION 'AUT24: cierre no independiente' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 SELECT cc.revision INTO STRICT revision FROM vec_autorizacion.control_continuidad_admin cc WHERE cc.control_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible WHERE propuesta_ref=propuesta.propuesta_ref) THEN RAISE EXCEPTION 'AUT24: propuesta ya cerrada' USING ERRCODE='23505'; END IF;
 material_propuesta:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb;
 IF m->>'decision'='aprobada' THEN
  IF clock_timestamp()>=propuesta.caduca_en OR revision IS DISTINCT FROM propuesta.revision_continuidad_esperada OR (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=propuesta.proponente_persona_ref AND x->>'perfil_ref'=propuesta.proponente_perfil_ref) THEN RAISE EXCEPTION 'AUT24: propuesta obsoleta o proponente revocado' USING ERRCODE='40001'; END IF;
 END IF;
 ahora:=clock_timestamp();huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 INSERT INTO vec_autorizacion.cierre_propuesta_perfil_sensible(propuesta_ref,cierre_ref,resultado,aprobador_persona_ref,aprobador_cuenta_ref,aprobador_perfil_ref,motivo_codigo,revision_continuidad_observada,documento_canonico,huella_sha256,cerrada_en)
 VALUES(propuesta.propuesta_ref,m->>'operacion_ref',m->>'decision',m->>'actor_persona_ref',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',m->>'actor_perfil_ref',m#>>'{motivo,entrada_clave}',revision,convert_to(p_material,'UTF8'),huella,ahora);
 IF m->>'decision'='aprobada' THEN
  recibo:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(material_propuesta,m->>'operacion_ref',propuesta.propuesta_ref,consumo->>'auditoria_ref',m->>'actor_persona_ref',false,consumo->>'decision_ref',encode(sha256(convert_to(p_material,'UTF8')),'hex'));
  ahora:=(recibo->>'confirmado_en')::timestamptz;
  acto:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'preimagen_huella_sha256',recibo->>'huella_antes_sha256','postimagen_huella_sha256',recibo->>'huella_despues_sha256','auditoria_ref',consumo->>'auditoria_ref');
  INSERT INTO vec_autorizacion.acto_perfil_sensible VALUES(recibo->>'acto_ref',propuesta.propuesta_ref,m->>'operacion_ref',revision,revision+1,recibo->>'huella_antes_sha256',recibo->>'huella_despues_sha256',convert_to(acto::text,'UTF8'),encode(sha256(convert_to(acto::text,'UTF8')),'hex'),ahora);
  INSERT INTO vec_autorizacion.recibo_perfil_sensible VALUES(recibo->>'recibo_ref',recibo->>'acto_ref',convert_to(recibo::text,'UTF8'),encode(sha256(convert_to(recibo::text,'UTF8')),'hex'),ahora);
 END IF;
 resultado:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'decision',m->>'decision','huella_cierre_sha256',huella,'confirmado_en',ahora,'recibo',recibo);
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,CASE WHEN m->>'decision'='aprobada' THEN 'propuesta_aprobada' ELSE 'propuesta_rechazada' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_rol_administrable_v1(text) TO vec_admin_perfiles_ejecutor;
CREATE FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE a record;c jsonb;i jsonb;cfg record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: contexto requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_estado='consumido') THEN RETURN; END IF;
 FOR a IN SELECT x.*,va.cuenta_ref,va.vinculo_ref,va.rol_huella_sha256 FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref) JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref)
 JOIN vec_autorizacion.rol_sensible_exacto sx USING(version_rol_ref) JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=rc.version_rol_ref AND cv.revision=rc.revision
 WHERE va.cuenta_ref=p_cuenta AND x.documento->>'estado'='activa'
 AND r.rol_id IN ('administracion_perfiles','operador_plataforma') AND sx.clase='administrador' AND sx.huella_sha256=r.huella_sha256 AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
 AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz
 ORDER BY x.principal_id,x.perfil_activo_ref FOR UPDATE OF ac LOOP
  SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=a.version_rol_ref AND huella_sha256=a.rol_huella_sha256;
  IF cfg.clase<>'administrador' OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN CONTINUE; END IF;
  c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p_cuenta,a.principal_id,a.perfil_activo_ref,a.vinculo_ref);
  i:=vec_identidad_sesiones_v1.estado_admin_interno_v1(a.principal_id,p_cuenta);
  IF i IS NULL OR c#>>'{perfil,estado}' IS DISTINCT FROM 'activo' OR c#>>'{vinculo,estado}' IS DISTINCT FROM 'activo' OR c#>>'{perfil,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,cuenta_ref}' IS DISTINCT FROM p_cuenta OR c#>>'{vinculo,perfil_ref}' IS DISTINCT FROM a.perfil_activo_ref THEN CONTINUE; END IF;
  RETURN QUERY SELECT a.principal_id,a.perfil_activo_ref,a.vinculo_ref,(c#>>'{cuenta,version}')::numeric,(c#>>'{persona,version}')::numeric,(c#>>'{perfil,version}')::numeric,(c#>>'{vinculo,version}')::numeric,cfg.audiencia_administrativa,LEAST((c#>>'{cuenta,vigente_hasta}')::timestamptz,(c#>>'{persona,vigente_hasta}')::timestamptz,(c#>>'{perfil,vigente_hasta}')::timestamptz,(c#>>'{vinculo,vigente_hasta}')::timestamptz,(i->>'vigente_hasta')::timestamptz,(a.documento->>'vigente_hasta')::timestamptz,cfg.vigente_hasta);
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE a record;puntero_asig text;puntero_control numeric;c jsonb;i jsonb;cfg record;
BEGIN
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: reconciliacion requiere READ COMMITTED' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_estado='consumido') THEN RETURN; END IF;
 FOR puntero_asig IN SELECT ac.asignacion_ref FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref) WHERE va.cuenta_ref=p_cuenta ORDER BY ac.perfil_activo_ref FOR UPDATE OF ac LOOP
 SELECT x.*,va.cuenta_ref,va.vinculo_ref,va.rol_huella_sha256 INTO STRICT a FROM vec_autorizacion.asignacion_perfil x JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref) WHERE x.asignacion_ref=puntero_asig;
 SELECT rc.revision INTO STRICT puntero_control FROM vec_autorizacion.control_vigencia_version_rol_actual rc WHERE rc.version_rol_ref=a.version_rol_ref FOR UPDATE;
 SELECT x.*,va.cuenta_ref,va.vinculo_ref,va.rol_huella_sha256 INTO a FROM vec_autorizacion.asignacion_perfil x JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref)
 JOIN vec_autorizacion.rol_sensible_exacto sx USING(version_rol_ref) JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=rc.version_rol_ref AND cv.revision=rc.revision
 WHERE va.cuenta_ref=p_cuenta AND x.documento->>'estado'='activa'
 AND r.rol_id IN ('administracion_perfiles','operador_plataforma') AND sx.clase='administrador' AND sx.huella_sha256=r.huella_sha256 AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
 AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz
 AND x.asignacion_ref=puntero_asig AND cv.revision=puntero_control;
 IF NOT FOUND THEN CONTINUE; END IF;
  SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=a.version_rol_ref AND huella_sha256=a.rol_huella_sha256;
 IF cfg.clase<>'administrador' OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN CONTINUE; END IF;
  c:=vec_contexto_actor_v1.preimagen_admin_interna_reconciliacion_v1(p_cuenta,a.principal_id,a.perfil_activo_ref,a.vinculo_ref);
  i:=vec_identidad_sesiones_v1.estado_admin_interno_v1(a.principal_id,p_cuenta);
  IF i IS NULL OR c#>>'{perfil,estado}' IS DISTINCT FROM 'activo' OR c#>>'{vinculo,estado}' IS DISTINCT FROM 'activo' OR c#>>'{perfil,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,cuenta_ref}' IS DISTINCT FROM p_cuenta OR c#>>'{vinculo,perfil_ref}' IS DISTINCT FROM a.perfil_activo_ref THEN CONTINUE; END IF;
  RETURN QUERY SELECT a.principal_id,a.perfil_activo_ref,a.vinculo_ref,(c#>>'{cuenta,version}')::numeric,(c#>>'{persona,version}')::numeric,(c#>>'{perfil,version}')::numeric,(c#>>'{vinculo,version}')::numeric,cfg.audiencia_administrativa,LEAST((c#>>'{cuenta,vigente_hasta}')::timestamptz,(c#>>'{persona,vigente_hasta}')::timestamptz,(c#>>'{perfil,vigente_hasta}')::timestamptz,(c#>>'{vinculo,vigente_hasta}')::timestamptz,(i->>'vigente_hasta')::timestamptz,(a.documento->>'vigente_hasta')::timestamptz,cfg.vigente_hasta);
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(text) TO vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta text,p_perfil text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF vec_autorizacion.texto_positivo_valido(p_perfil,512) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: perfil seleccionado ausente' USING ERRCODE='22023'; END IF;
 RETURN QUERY SELECT x.* FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta) x WHERE x.perfil_ref=p_perfil;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text,text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta text,p_perfil text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF vec_autorizacion.texto_positivo_valido(p_perfil,512) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: perfil seleccionado ausente' USING ERRCODE='22023'; END IF;
 RETURN QUERY SELECT x.* FROM vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta) x WHERE x.perfil_ref=p_perfil;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(text,text) TO vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,rol_version_ref text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RETURN QUERY SELECT x.persona_ref,x.perfil_ref,x.vinculo_ref,x.cuenta_version,x.persona_version,x.perfil_version,x.vinculo_version,x.audiencia,x.vigente_hasta,a.version_rol_ref,r.documento->>'nombre',g.categoria_admin
 FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta) x
 JOIN vec_autorizacion.asignacion_perfil_actual ac ON ac.perfil_activo_ref=x.perfil_ref
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=ac.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref JOIN vec_autorizacion.rol_administrable_exacto_v1 g ON g.version_rol_ref=r.version_rol_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_v1(text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_reconciliacion_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,rol_version_ref text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RETURN QUERY SELECT x.persona_ref,x.perfil_ref,x.vinculo_ref,x.cuenta_version,x.persona_version,x.perfil_version,x.vinculo_version,x.audiencia,x.vigente_hasta,a.version_rol_ref,r.documento->>'nombre',g.categoria_admin
 FROM vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta) x
 JOIN vec_autorizacion.asignacion_perfil_actual ac ON ac.perfil_activo_ref=x.perfil_ref
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=ac.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref JOIN vec_autorizacion.rol_administrable_exacto_v1 g ON g.version_rol_ref=r.version_rol_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_reconciliacion_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_asignaciones_admin_perfiles_propias_reconciliacion_v1(text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.consultar_metadata_perfil_admin_v1(p_cuenta text,p_perfil text)
RETURNS TABLE(rol_id text,rol_version_ref text,rol_huella_sha256 text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE validado record;
BEGIN
 IF current_setting('transaction_isolation')='serializable' THEN SELECT * INTO STRICT validado FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta,p_perfil);
 ELSIF current_setting('transaction_isolation')='read committed' THEN SELECT * INTO STRICT validado FROM vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta,p_perfil);
 ELSE RAISE EXCEPTION 'AUT24: aislamiento de metadata no admitido' USING ERRCODE='25000'; END IF;
 RETURN QUERY SELECT r.rol_id,r.version_rol_ref,r.huella_sha256,r.documento->>'nombre',g.categoria_admin
 FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref) JOIN vec_autorizacion.rol_administrable_exacto_v1 g USING(version_rol_ref)
 WHERE a.perfil_activo_ref=p_perfil AND a.principal_id=validado.persona_ref AND g.huella_sha256=r.huella_sha256;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_metadata_perfil_admin_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_metadata_perfil_admin_v1(text,text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.revalidar_audiencia_selector_admin_v1(p_audiencia text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE fuente text;
BEGIN
 SELECT convert_from(b.plan_canonico,'UTF8')::jsonb#>>'{gobierno,audiencia_selector_admin}' INTO fuente
 FROM vec_autorizacion.control_continuidad_admin c JOIN vec_autorizacion.bootstrap_admin_v2 b ON b.acto_ref=c.bootstrap_acto_ref WHERE c.control_id AND c.bootstrap_estado='consumido';
 RETURN fuente IS NOT NULL AND fuente=p_audiencia;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_audiencia_selector_admin_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_audiencia_selector_admin_v1(text) TO vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_autorizacion.preparar_preimagen_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;consumo jsonb;c record;d jsonb;r jsonb;o jsonb;acto jsonb;pi jsonb;efectivos jsonb;objetivo jsonb;operaciones jsonb;v_persona numeric;v_cuenta numeric;v_perfil numeric;v_vinculo numeric;revision bigint;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: consulta de preimagen invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_preimagen_v1' OR NOT m ?& ARRAY['operacion_ref','actor_persona_ref','actor_perfil_ref','operacion','rol_version_ref','cuenta_ref','persona_ref','perfil_ref','vinculo_ref','procedencia_ref','procedencia_version','procedencia_huella_sha256','vigente_hasta','motivo','correlacion_ref'] OR jsonb_path_exists(m,'$.** ? (@ == null)') OR m->>'operacion_ref' !~ '^consulta_admin:[0-9a-f]{32}$' OR (m->>'operacion' IN ('otorgar','revocar')) IS NOT TRUE OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$' OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR m->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$' OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar' THEN RAISE EXCEPTION 'AUT24: DTO de preimagen invalido' USING ERRCODE='22023'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: preimagen requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no efectivo' USING ERRCODE='42501'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 IF ((r->>'unidad_requerida')::boolean AND vec_autorizacion.texto_positivo_valido(m->>'unidad_ref',512) IS NOT TRUE) OR (NOT(r->>'unidad_requerida')::boolean AND m ? 'unidad_ref') THEN RAISE EXCEPTION 'AUT24: unidad de preimagen divergente' USING ERRCODE='42501'; END IF;
 IF m->>'actor_persona_ref' IS NOT DISTINCT FROM m->>'persona_ref' AND NOT(r->>'clase'='administrador' AND m->>'operacion'='revocar') THEN RAISE EXCEPTION 'AUT24: preimagen sin acto independiente disponible' USING ERRCODE='42501'; END IF;
 o:=jsonb_build_object('cuenta_ref',m->>'cuenta_ref','cuenta_version',0,'persona_ref',m->>'persona_ref','persona_version',0,'perfil_ref',m->>'perfil_ref','perfil_version',0,'vinculo_ref',m->>'vinculo_ref','vinculo_version',0,'huella_sha256','','revision_continuidad',0,'procedencia_ref',m->>'procedencia_ref','procedencia_version',m->'procedencia_version','procedencia_huella_sha256',m->>'procedencia_huella_sha256','vigente_hasta',m->>'vigente_hasta');
 acto:=jsonb_build_object('operacion',m->>'operacion','rol_version_ref',m->>'rol_version_ref','objetivo',o,'unidad_ref',m->>'unidad_ref');
 pi:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 v_persona:=(pi#>>'{contexto,persona,version}')::numeric;v_cuenta:=(pi#>>'{contexto,cuenta,version}')::numeric;
 IF m->>'operacion'='otorgar' THEN
  IF pi#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb OR pi#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb OR pi->'asignacion' IS DISTINCT FROM 'null'::jsonb OR (m->>'vigente_hasta')::timestamptz<=clock_timestamp() OR (m->>'vigente_hasta')::timestamptz>(r->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'AUT24: alta sin referencias nuevas o vigencia' USING ERRCODE='40001'; END IF;
  v_perfil:=0;v_vinculo:=0;
 ELSE
  IF pi#>>'{contexto,perfil,persona_ref}' IS DISTINCT FROM m->>'persona_ref' OR pi#>>'{contexto,vinculo,persona_ref}' IS DISTINCT FROM m->>'persona_ref' OR pi#>>'{contexto,vinculo,cuenta_ref}' IS DISTINCT FROM m->>'cuenta_ref' OR pi#>>'{contexto,vinculo,perfil_ref}' IS DISTINCT FROM m->>'perfil_ref' OR pi#>>'{contexto,perfil,estado}' IS DISTINCT FROM 'activo' OR pi#>>'{contexto,vinculo,estado}' IS DISTINCT FROM 'activo' OR pi#>>'{asignacion,estado}' IS DISTINCT FROM 'activa' OR pi#>>'{asignacion,version_rol_ref}' IS DISTINCT FROM m->>'rol_version_ref' OR m->>'vigente_hasta' IS DISTINCT FROM '0001-01-01T00:00:00Z' THEN RAISE EXCEPTION 'AUT24: baja sin perfil exacto vivo' USING ERRCODE='40001'; END IF;
  IF (r->>'unidad_requerida')::boolean AND pi#>'{asignacion,ambitos}' IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(m->>'unidad_ref'))) THEN RAISE EXCEPTION 'AUT24: unidad de preimagen ajena' USING ERRCODE='42501'; END IF;
  v_perfil:=(pi#>>'{contexto,perfil,version}')::numeric;v_vinculo:=(pi#>>'{contexto,vinculo,version}')::numeric;
 END IF;
 revision:=CASE WHEN r->>'clase'='ordinario' THEN 0 ELSE (pi#>>'{continuidad,revision}')::bigint END;
 o:=o||jsonb_build_object('cuenta_version',v_cuenta,'persona_version',v_persona,'perfil_version',v_perfil,'vinculo_version',v_vinculo,'revision_continuidad',revision);
 acto:=jsonb_set(acto,'{objetivo}',o);
 pi:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 objetivo:=o||jsonb_build_object('huella_sha256',encode(sha256(convert_to(pi::text,'UTF8')),'hex'));
 operaciones:=jsonb_build_array(jsonb_build_object('clase',r->>'clase','operacion',m->>'operacion','rol_version_ref',m->>'rol_version_ref'));
 RETURN jsonb_build_object('preimagen',objetivo,'actos_disponibles',operaciones);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preparar_preimagen_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.preparar_preimagen_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

RESET ROLE;
DO $rol_bootstrap$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_bootstrap_ejecutor') THEN CREATE ROLE vec_admin_perfiles_bootstrap_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_bootstrap_ejecutor' AND(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)) OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_perfiles_bootstrap_ejecutor'::regrole) THEN RAISE EXCEPTION 'AUT24: rol_bootstrap esperado=aislado observado=privilegiado' USING ERRCODE='55000'; END IF;
END $rol_bootstrap$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(p_persona text,p_cuenta text,p_gobierno jsonb,p_ca text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE pol record;c record;
BEGIN
 SELECT * INTO STRICT pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 WHERE singleton FOR SHARE;
 IF pol.politica_ref IS DISTINCT FROM p_gobierno->>'politica_certificado_ref' OR pol.huella_aprobacion_sha256 IS DISTINCT FROM p_gobierno->>'politica_certificado_huella_sha256' OR pol.ca_sha256 IS DISTINCT FROM p_ca OR NOT pol.activa OR clock_timestamp()>=pol.vigente_hasta THEN RAISE EXCEPTION 'AUT24: politica_certificado esperado=aprobada_vigente observado=divergente' USING ERRCODE='40001'; END IF;
 SELECT pc.cuenta_privilegiada,pc.cuenta_ordinaria_ref,s.estado,s.revision,so.estado AS estado_ordinaria,so.revision AS revision_ordinaria INTO STRICT c
 FROM vec_identidad_sesiones_v1.cuenta pc JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref) JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ao ON ao.cuenta_ref=pc.cuenta_ordinaria_ref JOIN vec_identidad_sesiones_v1.estado_cuenta so ON so.cuenta_ref=ao.cuenta_ref AND so.revision=ao.revision WHERE pc.cuenta_ref=p_cuenta FOR SHARE OF a,ao;
 IF NOT c.cuenta_privilegiada OR c.estado<>'activa' OR c.estado_ordinaria<>'activa' THEN RAISE EXCEPTION 'AUT24: cuenta_bootstrap esperado=privilegiada_activa observado=indisponible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('persona_ref',p_persona,'cuenta_ref',p_cuenta,'cuenta_revision',c.revision,'cuenta_ordinaria_ref',c.cuenta_ordinaria_ref,'cuenta_ordinaria_revision',c.revision_ordinaria,'politica_ref',pol.politica_ref,'politica_huella_sha256',pol.huella_aprobacion_sha256,'ca_sha256',pol.ca_sha256,'vigente_hasta',pol.vigente_hasta);
END $f$;
CREATE FUNCTION vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(p_persona jsonb,p_gobierno jsonb,p_acto text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE identidad jsonb;b record;cert jsonb:=p_persona->'certificado_admin';
BEGIN
 identidad:=vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(p_persona->>'persona_ref',p_persona->>'cuenta_ref',p_gobierno,cert->>'ca_huella_sha256');
 IF cert->>'persona_ref' IS DISTINCT FROM p_persona->>'persona_ref' OR cert->>'cuenta_ref' IS DISTINCT FROM p_persona->>'cuenta_ref' OR (p_persona->>'vigente_hasta')::timestamptz>(identidad->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'AUT24: certificado_bootstrap discordante' USING ERRCODE='40001'; END IF;
 SELECT x.* INTO b FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version) WHERE x.certificado_sha256=cert->>'huella_sha256' FOR UPDATE OF a;
 IF FOUND THEN
  IF b.persona_ref IS DISTINCT FROM p_persona->>'persona_ref' OR b.cuenta_privilegiada_ref IS DISTINCT FROM p_persona->>'cuenta_ref' OR b.ca_sha256 IS DISTINCT FROM cert->>'ca_huella_sha256' OR b.politica_ref IS DISTINCT FROM p_gobierno->>'politica_certificado_ref' OR b.estado IS DISTINCT FROM 'activo' OR clock_timestamp()<b.vigente_desde OR clock_timestamp()>=b.vigente_hasta OR b.vigente_hasta<(p_persona->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'AUT24: certificado_bootstrap no reutilizable' USING ERRCODE='40001'; END IF;
  RETURN true;
 END IF;
 RETURN vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1(p_persona->>'vinculo_ref',p_persona->>'persona_ref',p_persona->>'cuenta_ref',cert->>'huella_sha256',cert->>'ca_huella_sha256',p_gobierno->>'politica_certificado_ref',(p_persona->>'vigente_hasta')::timestamptz,p_acto);
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(text,text,jsonb,text),vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(jsonb,jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(text,text,jsonb,text),vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(jsonb,jsonb,text) TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.bootstrap_admin_v2(
 acto_ref text PRIMARY KEY CHECK(acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 plan_canonico bytea NOT NULL CHECK(octet_length(plan_canonico) BETWEEN 1 AND 65536),
 huella_plan_sha256 text NOT NULL UNIQUE CHECK(huella_plan_sha256=encode(sha256(plan_canonico),'hex')),
 operador_sql text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 resultado jsonb NOT NULL,
 confirmado_en timestamptz NOT NULL CHECK(isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.auditoria_bootstrap_admin_v2(
 auditoria_ref text PRIMARY KEY,
 acto_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.bootstrap_admin_v2(acto_ref),
 operador_sql text NOT NULL,
 huella_plan_sha256 text NOT NULL,
 resultado text NOT NULL CHECK(resultado='aplicado'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_autorizacion.outbox_bootstrap_admin_v2(
 acto_ref text PRIMARY KEY REFERENCES vec_autorizacion.bootstrap_admin_v2(acto_ref),
 auditoria_ref text NOT NULL REFERENCES vec_autorizacion.auditoria_bootstrap_admin_v2(auditoria_ref),
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en))
);
DO $historia_bootstrap$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['bootstrap_admin_v2','auditoria_bootstrap_admin_v2','outbox_bootstrap_admin_v2'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_admin_perfiles_bootstrap_ejecutor,vec_autorizacion_fuente',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_admin_perfiles_bootstrap_ejecutor,vec_autorizacion_fuente',t);
 END LOOP;
END $historia_bootstrap$;

CREATE FUNCTION vec_autorizacion.material_persona_bootstrap_interno_v2(p_plan jsonb,p_persona jsonb,p_operador text,p_operacion text)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT jsonb_build_object('operacion_ref',p_operacion,'operacion','otorgar','rol_version_ref',p_plan#>>'{rol,version_ref}','rol_huella_sha256',p_plan#>>'{rol,huella_sha256}','actor_persona_ref',p_operador,'objetivo',jsonb_build_object('cuenta_ref',p_persona->>'cuenta_ref','cuenta_version',p_persona->'cuenta_version','persona_ref',p_persona->>'persona_ref','persona_version',p_persona->'persona_version','perfil_ref',p_persona->>'perfil_ref','perfil_version',0,'vinculo_ref',p_persona->>'vinculo_ref','vinculo_version',0,'huella_sha256',p_persona->>'preimagen_huella_sha256','revision_continuidad',0,'procedencia_ref',p_persona#>>'{procedencia,referencia}','procedencia_version',p_persona#>'{procedencia,version}','procedencia_huella_sha256',p_persona#>>'{procedencia,huella_sha256}','vigente_hasta',p_persona->>'vigente_hasta'))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.material_persona_bootstrap_interno_v2(jsonb,jsonb,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.provisionar_dos_administradores_iniciales_v2(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE plan jsonb;gov jsonb;rol record;control record;continuidad record;previo record;cfg jsonb;persona jsonb;m jsonb;pi jsonb;identidad jsonb;actor text:=session_user;huella text;acto text;audit text;recibo text;resultado jsonb;efecto jsonb;partes jsonb:='[]'::jsonb;ahora timestamptz;clase text;duracion interval;sis jsonb;plan_sis jsonb;persona_sis jsonb;rol_sis record;control_sis record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: bootstrap requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 IF NOT pg_has_role(session_user,'vec_admin_perfiles_bootstrap_ejecutor','MEMBER') OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid='vec_admin_perfiles_bootstrap_ejecutor'::regrole AND inherit_option AND NOT set_option AND NOT admin_option) OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=session_user::regrole AND (NOT rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)) THEN RAISE EXCEPTION 'AUT24: operador bootstrap no autorizado' USING ERRCODE='42501'; END IF;
 IF p_plan_canonico IS NULL OR octet_length(p_plan_canonico) NOT BETWEEN 1 AND 65536 OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'AUT24: aprobacion de bootstrap ausente' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(p_plan_canonico,'UTF8')),'hex');
 IF huella IS DISTINCT FROM p_huella_aprobada THEN RAISE EXCEPTION 'AUT24: plan_huella esperado=% observado=%',p_huella_aprobada,huella USING ERRCODE='40001'; END IF;
 plan:=p_plan_canonico::jsonb;gov:=plan->'gobierno';
 IF plan->>'version' IS DISTINCT FROM '2' OR NOT plan ?& ARRAY['preparado_en','caduca_en','control_continuidad_revision_esperada','bootstrap_estado_esperado','rol','fuente_identidad','fuente_ca_admin','personas','gobierno'] OR jsonb_path_exists(plan,'$.** ? (@ == null)') OR jsonb_typeof(plan->'personas') IS DISTINCT FROM 'array' OR jsonb_array_length(plan->'personas')<>2 OR plan#>>'{personas,0,persona_ref}'>=plan#>>'{personas,1,persona_ref}' OR plan#>>'{personas,0,cuenta_ref}'=plan#>>'{personas,1,cuenta_ref}' OR plan#>>'{personas,0,perfil_ref}'=plan#>>'{personas,1,perfil_ref}' OR plan#>>'{personas,0,vinculo_ref}'=plan#>>'{personas,1,vinculo_ref}' OR plan#>>'{personas,0,certificado_admin,huella_sha256}'=plan#>>'{personas,1,certificado_admin,huella_sha256}' OR plan->>'control_continuidad_revision_esperada' IS DISTINCT FROM '1' OR plan->>'bootstrap_estado_esperado' IS DISTINCT FROM 'pendiente' OR jsonb_typeof(gov->'roles') IS DISTINCT FROM 'array' OR jsonb_array_length(gov->'roles') NOT BETWEEN 1 AND 64 THEN RAISE EXCEPTION 'AUT24: plan de bootstrap invalido' USING ERRCODE='22023'; END IF;
 IF vec_autorizacion.texto_positivo_valido(gov->>'audiencia_selector_admin',256) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: audiencia selector ausente en gobierno aprobado' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(plan#>'{personas,0,sistemas}') IS DISTINCT FROM 'array' OR jsonb_typeof(plan#>'{personas,1,sistemas}') IS DISTINCT FROM 'array' OR jsonb_array_length(plan#>'{personas,0,sistemas}')+jsonb_array_length(plan#>'{personas,1,sistemas}')<>1 THEN RAISE EXCEPTION 'AUT24: kit requiere dos personas Aplicacion y una asignacion Sistemas' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 acto:='acto_admin:'||substr(huella,1,32);audit:='auditoria_bootstrap:'||huella;recibo:='recibo_admin:'||substr(encode(sha256(convert_to(huella||':recibo','UTF8')),'hex'),1,32);
 SELECT * INTO STRICT continuidad FROM vec_autorizacion.control_continuidad_admin WHERE control_id FOR UPDATE;
 SELECT * INTO previo FROM vec_autorizacion.bootstrap_admin_v2 WHERE acto_ref=acto;
 IF FOUND THEN
  IF previo.plan_canonico IS DISTINCT FROM convert_to(p_plan_canonico,'UTF8') OR previo.huella_plan_sha256 IS DISTINCT FROM huella OR continuidad.bootstrap_estado IS DISTINCT FROM 'consumido' OR continuidad.bootstrap_acto_ref IS DISTINCT FROM acto THEN RAISE EXCEPTION 'AUT24: bootstrap previo divergente' USING ERRCODE='40001'; END IF;
  FOR persona IN SELECT value FROM jsonb_array_elements(plan->'personas') LOOP
   PERFORM vec_contexto_actor_v1.preimagen_admin_interna_v1(persona->>'cuenta_ref',persona->>'persona_ref',persona->>'perfil_ref',persona->>'vinculo_ref');
   PERFORM vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(persona->>'persona_ref',persona->>'cuenta_ref',gov,plan#>>'{fuente_ca_admin,huella_sha256}');
  END LOOP;
  RETURN previo.resultado;
 END IF;
 IF continuidad.bootstrap_estado IS DISTINCT FROM 'pendiente' OR continuidad.revision<>1 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil a JOIN vec_autorizacion.version_rol r USING(version_rol_ref) WHERE r.rol_id='administracion_perfiles') THEN RAISE EXCEPTION 'AUT24: bootstrap_estado esperado=pendiente_vacio observado=consumido_o_historia' USING ERRCODE='40001'; END IF;
 ahora:=clock_timestamp();
 IF NOT isfinite((plan->>'preparado_en')::timestamptz) OR NOT isfinite((plan->>'caduca_en')::timestamptz) OR (plan->>'preparado_en')::timestamptz>ahora OR (plan->>'caduca_en')::timestamptz<=ahora OR (plan->>'caduca_en')::timestamptz<=(plan->>'preparado_en')::timestamptz THEN RAISE EXCEPTION 'AUT24: plan de bootstrap caducado' USING ERRCODE='40001'; END IF;
 SELECT * INTO STRICT rol FROM vec_autorizacion.version_rol WHERE version_rol_ref=plan#>>'{rol,version_ref}';
 SELECT v.* INTO STRICT control FROM vec_autorizacion.control_vigencia_version_rol_actual a JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision) WHERE a.version_rol_ref=rol.version_rol_ref FOR UPDATE OF a;
 IF rol.rol_id IS DISTINCT FROM 'administracion_perfiles' OR rol.documento->>'estado' IS DISTINCT FROM 'publicada' OR rol.huella_sha256 IS DISTINCT FROM plan#>>'{rol,huella_sha256}' OR control.estado IS DISTINCT FROM 'habilitada' OR control.revision IS DISTINCT FROM (plan#>>'{rol,control_revision}')::numeric OR control.huella_sha256 IS DISTINCT FROM plan#>>'{rol,control_huella_sha256}' OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto sx WHERE sx.version_rol_ref=rol.version_rol_ref AND sx.clase='administrador' AND sx.huella_sha256=rol.huella_sha256) THEN RAISE EXCEPTION 'AUT24: rol bootstrap publicado divergente' USING ERRCODE='40001'; END IF;
 -- La tercera asignación del kit usa la misma persona/cuenta acreditada y
 -- referencias nuevas propias. Su rol y control se cotejan sin inferir permisos.
 FOR persona IN SELECT value FROM jsonb_array_elements(plan->'personas') LOOP
  FOR sis IN SELECT value FROM jsonb_array_elements(persona->'sistemas') LOOP
   SELECT * INTO STRICT rol_sis FROM vec_autorizacion.version_rol WHERE version_rol_ref=sis#>>'{rol,version_ref}';
   SELECT cv.* INTO STRICT control_sis FROM vec_autorizacion.control_vigencia_version_rol_actual rc JOIN vec_autorizacion.control_vigencia_version_rol cv USING(version_rol_ref,revision) WHERE rc.version_rol_ref=rol_sis.version_rol_ref FOR UPDATE OF rc;
   IF rol_sis.rol_id IS DISTINCT FROM 'operador_plataforma' OR rol_sis.huella_sha256 IS DISTINCT FROM sis#>>'{rol,huella_sha256}' OR control_sis.estado IS DISTINCT FROM 'habilitada' OR control_sis.revision IS DISTINCT FROM (sis#>>'{rol,control_revision}')::numeric OR control_sis.huella_sha256 IS DISTINCT FROM sis#>>'{rol,control_huella_sha256}' THEN RAISE EXCEPTION 'AUT24: kit Sistemas no corresponde a plantilla publicada' USING ERRCODE='40001'; END IF;
   pi:=vec_contexto_actor_v1.preimagen_admin_interna_v1(persona->>'cuenta_ref',persona->>'persona_ref',sis->>'perfil_ref',sis->>'vinculo_ref');
   IF pi->'perfil' IS DISTINCT FROM 'null'::jsonb OR pi->'vinculo' IS DISTINCT FROM 'null'::jsonb OR sis->>'perfil_ref' IN(plan#>>'{personas,0,perfil_ref}',plan#>>'{personas,1,perfil_ref}') OR sis->>'vinculo_ref' IN(plan#>>'{personas,0,vinculo_ref}',plan#>>'{personas,1,vinculo_ref}') OR NOT isfinite((sis->>'vigente_hasta')::timestamptz) OR (sis->>'vigente_hasta')::timestamptz<(plan->>'caduca_en')::timestamptz OR (sis->>'vigente_hasta')::timestamptz>(persona->>'vigente_hasta')::timestamptz OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual ac WHERE ac.perfil_activo_ref=sis->>'perfil_ref') THEN RAISE EXCEPTION 'AUT24: preimagen Sistemas requiere referencias nuevas y vigencia' USING ERRCODE='40001'; END IF;
  END LOOP;
 END LOOP;
 -- Ambas preimágenes se cotejan antes de la primera escritura. Los bloqueos
 -- se conservan hasta COMMIT; no se recalcula la segunda tras la primera alta.
 FOR persona IN SELECT value FROM jsonb_array_elements(plan->'personas') LOOP
  m:=vec_autorizacion.material_persona_bootstrap_interno_v2(plan,persona,actor,acto);
  pi:=vec_autorizacion.preimagen_cambio_admin_interna_v1(m);
  identidad:=vec_identidad_sesiones_v1.preimagen_bootstrap_admin_interna_v2(persona->>'persona_ref',persona->>'cuenta_ref',gov,plan#>>'{fuente_ca_admin,huella_sha256}');
  IF encode(sha256(convert_to(pi::text,'UTF8')),'hex') IS DISTINCT FROM persona->>'preimagen_huella_sha256' OR (pi#>>'{contexto,cuenta,version}')::numeric IS DISTINCT FROM (persona->>'cuenta_version')::numeric OR (pi#>>'{contexto,persona,version}')::numeric IS DISTINCT FROM (persona->>'persona_version')::numeric OR pi#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb OR pi#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb OR pi->'asignacion' IS DISTINCT FROM 'null'::jsonb OR (persona->>'vigente_hasta')::timestamptz<(plan->>'caduca_en')::timestamptz OR (persona->>'vigente_hasta')::timestamptz>(identidad->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'AUT24: preimagen bootstrap divergente' USING ERRCODE='40001'; END IF;
 END LOOP;
 FOR cfg IN SELECT value FROM jsonb_array_elements(gov->'roles') LOOP
  IF NOT cfg ?& ARRAY['version_ref','huella_sha256','clase','unidad_requerida','ambitos_fijos','vigente_desde','vigente_hasta','duracion_propuesta_segundos','categoria_admin'] OR jsonb_typeof(cfg->'unidad_requerida') IS DISTINCT FROM 'boolean' OR cfg->>'clase' NOT IN ('ordinario','administrador','intervencion') OR jsonb_typeof(cfg->'ambitos_fijos') IS DISTINCT FROM 'array' OR ((cfg->>'unidad_requerida')::boolean AND jsonb_array_length(cfg->'ambitos_fijos')<>0) OR (NOT (cfg->>'unidad_requerida')::boolean AND vec_autorizacion.ambitos_positivos_validos(jsonb_build_object('ambitos',cfg->'ambitos_fijos')) IS NOT TRUE) OR (cfg->>'vigente_desde')::timestamptz>(plan->>'preparado_en')::timestamptz OR (cfg->>'vigente_hasta')::timestamptz<(plan->>'caduca_en')::timestamptz OR (cfg->>'duracion_propuesta_segundos')::numeric<1 OR (cfg->>'duracion_propuesta_segundos')::numeric>extract(epoch FROM ((cfg->>'vigente_hasta')::timestamptz-(plan->>'preparado_en')::timestamptz)) THEN RAISE EXCEPTION 'AUT24: gobierno de rol invalido' USING ERRCODE='22023'; END IF;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol v JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref) JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision WHERE v.version_rol_ref=cfg->>'version_ref' AND v.huella_sha256=cfg->>'huella_sha256' AND v.documento->>'estado'='publicada' AND c.estado='habilitada') THEN RAISE EXCEPTION 'AUT24: catalogo no puede publicar permisos' USING ERRCODE='42501'; END IF;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol vr WHERE vr.version_rol_ref=cfg->>'version_ref' AND ((vr.rol_id='administracion_perfiles' AND cfg->>'categoria_admin'='aplicacion') OR (vr.rol_id='operador_plataforma' AND cfg->>'categoria_admin'='sistemas') OR (cfg->>'clase'<>'administrador' AND cfg->>'categoria_admin'=''))) THEN RAISE EXCEPTION 'AUT24: categoria no corresponde a familia publicada' USING ERRCODE='42501'; END IF;
  SELECT s.clase INTO clase FROM vec_autorizacion.rol_sensible_exacto s WHERE s.version_rol_ref=cfg->>'version_ref' AND s.huella_sha256=cfg->>'huella_sha256';
  IF coalesce(clase,'ordinario') IS DISTINCT FROM cfg->>'clase' THEN RAISE EXCEPTION 'AUT24: clase de rol no declarable' USING ERRCODE='42501'; END IF;
  duracion:=make_interval(secs=>(cfg->>'duracion_propuesta_segundos')::double precision);
  INSERT INTO vec_autorizacion.rol_administrable_exacto_v1 VALUES(cfg->>'version_ref',cfg->>'clase',cfg->>'huella_sha256',(cfg->>'vigente_desde')::timestamptz,(cfg->>'vigente_hasta')::timestamptz,(cfg->>'unidad_requerida')::boolean,gov->>'audiencia_administrativa',cfg->'ambitos_fijos',duracion,cfg->>'categoria_admin');
 END LOOP;
 IF gov ? 'motivos' THEN
  IF jsonb_typeof(gov->'motivos') IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'AUT24: gobierno de motivos invalido' USING ERRCODE='22023'; END IF;
  FOR cfg IN SELECT value FROM jsonb_array_elements(gov->'motivos') LOOP
   IF NOT cfg ?& ARRAY['accion','tipo_recurso','finalidad','referencia'] OR jsonb_path_exists(cfg,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'AUT24: referencia de motivo invalida' USING ERRCODE='22023'; END IF;
   INSERT INTO vec_autorizacion.vinculacion_motivo_admin_v1 VALUES(cfg->>'accion',cfg->>'tipo_recurso',cfg->>'finalidad',cfg#>>'{referencia,catalogo_id}',(cfg#>>'{referencia,catalogo_version}')::integer,cfg#>>'{referencia,catalogo_huella_sha256}',cfg#>>'{referencia,entrada_clave}');
   IF vec_autorizacion.motivo_admin_publicado_interno_v1(cfg->>'accion',cfg->>'tipo_recurso') IS DISTINCT FROM cfg->'referencia' THEN RAISE EXCEPTION 'AUT24: motivo configurado no corresponde a catalogo publicado' USING ERRCODE='42501'; END IF;
  END LOOP;
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 ra WHERE ra.version_rol_ref=rol.version_rol_ref AND ra.clase='administrador' AND NOT ra.unidad_requerida AND ra.huella_sha256=rol.huella_sha256 AND ra.ambitos_fijos='[{"clave":"administracion","valores":["perfiles"]}]'::jsonb) THEN RAISE EXCEPTION 'AUT24: gobierno no incluye el rol bootstrap' USING ERRCODE='42501'; END IF;
 FOR persona IN SELECT value FROM jsonb_array_elements(plan->'personas') LOOP
  PERFORM vec_identidad_sesiones_v1.crear_certificado_bootstrap_admin_interno_v2(persona,gov,'acto_admin:'||substr(encode(sha256(convert_to(huella||':'||(persona->>'persona_ref')||':certificado','UTF8')),'hex'),1,32));
  m:=vec_autorizacion.material_persona_bootstrap_interno_v2(plan,persona,actor,acto);
  efecto:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(m,'acto_admin:'||substr(encode(sha256(convert_to(huella||':'||(persona->>'persona_ref')||':perfil','UTF8')),'hex'),1,32),null,audit,actor,true,null,huella);
  partes:=partes||jsonb_build_array(efecto);
 END LOOP;
 FOR persona IN SELECT value FROM jsonb_array_elements(plan->'personas') LOOP
  FOR sis IN SELECT value FROM jsonb_array_elements(persona->'sistemas') LOOP
   IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 g WHERE g.version_rol_ref=sis#>>'{rol,version_ref}' AND g.categoria_admin='sistemas' AND g.clase='administrador' AND NOT g.unidad_requerida AND g.huella_sha256=sis#>>'{rol,huella_sha256}') THEN RAISE EXCEPTION 'AUT24: gobierno Sistemas no publicado' USING ERRCODE='42501'; END IF;
   plan_sis:=jsonb_set(plan,'{rol}',sis->'rol');
   persona_sis:=persona||jsonb_build_object('perfil_ref',sis->>'perfil_ref','vinculo_ref',sis->>'vinculo_ref','vigente_hasta',sis->>'vigente_hasta');
   m:=vec_autorizacion.material_persona_bootstrap_interno_v2(plan_sis,persona_sis,actor,acto);
   efecto:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(m,'acto_admin:'||substr(encode(sha256(convert_to(huella||':'||(persona->>'persona_ref')||':sistemas','UTF8')),'hex'),1,32),null,audit,actor,true,null,huella);
   partes:=partes||jsonb_build_array(efecto);
  END LOOP;
 END LOOP;
 IF (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)<>continuidad.bootstrap_minimo_personas THEN RAISE EXCEPTION 'AUT24: bootstrap_personas esperado=2 observado=divergente' USING ERRCODE='42501'; END IF;
 UPDATE vec_autorizacion.control_continuidad_admin SET revision=revision+1,bootstrap_estado='consumido',bootstrap_acto_ref=acto,actualizado_en=clock_timestamp() WHERE control_id AND revision=1 AND bootstrap_estado='pendiente';
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: CAS de bootstrap perdido' USING ERRCODE='40001'; END IF;
 ahora:=clock_timestamp();resultado:=jsonb_build_object('acto_ref',acto,'recibo_ref',recibo,'huella_plan_sha256',huella,'primera_persona_ref',plan#>>'{personas,0,persona_ref}','segunda_persona_ref',plan#>>'{personas,1,persona_ref}','confirmado_en',ahora);
 INSERT INTO vec_autorizacion.bootstrap_admin_v2 VALUES(acto,convert_to(p_plan_canonico,'UTF8'),huella,actor,audit,resultado,ahora);
 INSERT INTO vec_autorizacion.auditoria_bootstrap_admin_v2 VALUES(audit,acto,actor,huella,'aplicado',ahora);
 INSERT INTO vec_autorizacion.outbox_bootstrap_admin_v2 VALUES(acto,audit,ahora);
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_bootstrap_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text) TO vec_admin_perfiles_bootstrap_ejecutor;

-- Fragmento privado para ensamblar en AUT24, antes de las siete fachadas.
-- No es una migración autónoma: no contiene BEGIN/COMMIT ni crea concesiones.
-- Dependencias: CA20, AUT24 y sus tablas/funciones privadas.
-- La metadata vinculacion_asignacion_admin_v1 permanece propiedad de AUT24;
-- este fragmento no la recrea ni amplía el documento canónico de asignación.
-- La fachada lectora debe consumir material V3 nuevo y revalidar el actor IS12
-- antes de llamar a proyectar_lectura_admin_interna_v1 en la misma transacción.
-- Las etiquetas visibles se localizan en Go/UI; CA no publica nombres.

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(
 p_busqueda text,p_cursor text,p_limite integer
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE datos jsonb; siguiente text; cantidad integer; ahora timestamptz:=clock_timestamp();
BEGIN
 IF p_limite IS DISTINCT FROM 50 OR p_busqueda IS NULL
 OR octet_length(p_busqueda)>80 OR p_busqueda !~ '^per_[A-Za-z0-9_-]*$'
 OR (coalesce(p_cursor,'')<>'' AND vec_contexto_actor_v1.referencia_valida(p_cursor,'per_') IS NOT TRUE) THEN
  RAISE EXCEPTION 'AUT24: busqueda nominal invalida' USING ERRCODE='22023';
 END IF;
 -- Comparación literal de prefijo: '_' y '%' nunca son comodines de búsqueda.
 WITH pagina AS (
  SELECT x.persona_ref FROM vec_contexto_actor_v1.persona_actual a
  JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version)
  WHERE left(x.persona_ref,length(p_busqueda))=p_busqueda
   AND (coalesce(p_cursor,'')='' OR x.persona_ref COLLATE "C">p_cursor COLLATE "C")
   AND x.estado='activo' AND x.procedencia_autoridad='autoridad_maestra_acreditada'
   AND ahora>=x.vigente_desde AND ahora<x.vigente_hasta
  ORDER BY x.persona_ref COLLATE "C" LIMIT 51
 ), numerada AS (
  SELECT persona_ref,row_number() OVER(ORDER BY persona_ref COLLATE "C") n FROM pagina
 ) SELECT coalesce(jsonb_agg(jsonb_build_object('persona_ref',persona_ref,
    'nombre','','unidad_nombre','','unidad_clave_i18n','') ORDER BY persona_ref COLLATE "C")
    FILTER(WHERE n<=50),'[]'::jsonb),count(*),max(persona_ref) FILTER(WHERE n=50)
  INTO datos,cantidad,siguiente FROM numerada;
 RETURN jsonb_build_object('personas',datos,'siguiente_cursor',CASE WHEN cantidad>50 THEN siguiente ELSE '' END);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(text,text,integer)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(text,text,integer)
 TO vec_autorizacion_propietario;

-- Devuelve solo referencias/versiones/procedencias técnicas a AUT owner.
-- No permite elegir una cuenta ajena: los contextos proceden del vínculo CA.
CREATE FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(p_persona text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pe record; contextos jsonb; perfiles jsonb; ahora timestamptz:=clock_timestamp();
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT24: persona nominal invalida' USING ERRCODE='22023';
 END IF;
 SELECT x.* INTO pe FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version)
 WHERE a.persona_ref=p_persona AND x.estado='activo'
 AND x.procedencia_autoridad='autoridad_maestra_acreditada'
 AND ahora>=x.vigente_desde AND ahora<x.vigente_hasta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: persona no disponible' USING ERRCODE='P0002'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.perfil_ref),'[]'::jsonb) INTO perfiles
 FROM (SELECT x.perfil_ref,x.version,x.persona_ref,x.estado,x.vigente_desde,x.vigente_hasta,
  x.procedencia_ref,x.procedencia_version,x.procedencia_huella_sha256,x.procedencia_autoridad
  FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
  WHERE x.persona_ref=p_persona AND x.procedencia_autoridad='autoridad_maestra_acreditada'
  ORDER BY x.perfil_ref LIMIT 50) t;
 SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.cuenta_ref,t.vinculo_ref),'[]'::jsonb) INTO contextos
 FROM (SELECT c.cuenta_ref,c.version AS cuenta_version,c.procedencia_ref,
  c.procedencia_version,c.procedencia_huella_sha256,c.vigente_hasta AS cuenta_vigente_hasta,
  v.vinculo_ref,v.version AS vinculo_version,v.perfil_ref,p.version AS perfil_version
  FROM vec_contexto_actor_v1.vinculo_contexto_actual a
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c ON c.cuenta_ref=ca.cuenta_ref AND c.version=ca.version
  JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
  JOIN vec_contexto_actor_v1.perfil_versiones p ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
  WHERE v.persona_ref=p_persona AND p.persona_ref=p_persona
  AND v.estado='activo' AND c.estado='activo' AND p.estado='activo'
  AND v.procedencia_autoridad='autoridad_maestra_acreditada'
  AND c.procedencia_autoridad='autoridad_maestra_acreditada'
  AND p.procedencia_autoridad='autoridad_maestra_acreditada'
  AND ahora>=GREATEST(v.vigente_desde,c.vigente_desde,p.vigente_desde)
  AND ahora<LEAST(v.vigente_hasta,c.vigente_hasta,p.vigente_hasta)
  ORDER BY c.cuenta_ref,v.vinculo_ref LIMIT 51) t;
 -- Cardinalidad excesiva no produce una selección arbitraria de cuenta.
 IF jsonb_array_length(contextos)>50 THEN contextos:='[]'::jsonb; END IF;
 RETURN jsonb_build_object('persona',to_jsonb(pe),'perfiles',perfiles,'contextos',contextos);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(text)
 TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Prepara una alta con fuente actual. No inventa motivo, procedencia ni unidad.
-- Es privada y no se devuelve como acto disponible hasta ligar un motivo ADMIN.
-- La baja permanece cerrada: CA20 exige una procedencia NUEVA de revocación.
CREATE FUNCTION vec_autorizacion.objetivo_alta_admin_lectura_interna_v1(m jsonb,p_rol text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE fuente jsonb; cuenta jsonb; cfg record; o jsonb; acto jsonb; pi jsonb;
 unidad text; cantidad integer; fecha timestamptz; revision bigint; efectivos jsonb;
BEGIN
 IF m->>'actor_persona_ref' IS NOT DISTINCT FROM m->>'persona_ref' THEN RETURN NULL; END IF;
 SELECT a.* INTO cfg FROM vec_autorizacion.rol_administrable_exacto_v1 a
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
 WHERE a.version_rol_ref=p_rol AND a.huella_sha256=r.huella_sha256
 AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
 AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta;
 IF NOT FOUND THEN RETURN NULL; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x
  WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RETURN NULL; END IF;
 IF cfg.clase<>'ordinario' AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 THEN RETURN NULL; END IF;
 fuente:=vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(m->>'persona_ref');
 SELECT count(DISTINCT x->>'cuenta_ref') INTO cantidad FROM jsonb_array_elements(fuente->'contextos') x;
 IF cantidad<>1 THEN RETURN NULL; END IF;
 SELECT x INTO cuenta FROM jsonb_array_elements(fuente->'contextos') x ORDER BY x->>'vinculo_ref' LIMIT 1;
 IF cfg.clase='administrador' AND (vec_identidad_sesiones_v1.estado_admin_interno_v1(m->>'persona_ref',cuenta->>'cuenta_ref') IS NULL
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'persona_ref')) THEN RETURN NULL; END IF;
 -- No usa el ámbito aportado por el cliente ni una configuración ambiental.
 IF cfg.unidad_requerida THEN
  SELECT count(DISTINCT u.unidad),min(u.unidad) INTO cantidad,unidad
  FROM vec_autorizacion.asignacion_perfil_actual ac
  JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  CROSS JOIN LATERAL jsonb_array_elements(a.documento->'ambitos') ab
  CROSS JOIN LATERAL jsonb_array_elements_text(ab->'valores') u(unidad)
  WHERE a.principal_id=m->>'persona_ref' AND a.documento->>'estado'='activa'
  AND ab->>'clave'='unidad' AND clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
  AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz;
  IF cantidad<>1 OR vec_autorizacion.texto_positivo_valido(unidad,512) IS NOT TRUE THEN RETURN NULL; END IF;
 END IF;
 SELECT c.revision INTO revision FROM vec_autorizacion.control_continuidad_admin c WHERE control_id AND bootstrap_estado='consumido';
 IF NOT FOUND THEN RETURN NULL; END IF;
 fecha:=LEAST(cfg.vigente_hasta,(cuenta->>'cuenta_vigente_hasta')::timestamptz,(fuente#>>'{persona,vigente_hasta}')::timestamptz);
 IF fecha<=clock_timestamp() THEN RETURN NULL; END IF;
 o:=jsonb_build_object('cuenta_ref',cuenta->>'cuenta_ref','cuenta_version',cuenta->'cuenta_version',
  'persona_ref',m->>'persona_ref','persona_version',fuente#>'{persona,version}',
  'perfil_ref','prf_'||replace(gen_random_uuid()::text,'-',''),'perfil_version',0,
  'vinculo_ref','vca_'||replace(gen_random_uuid()::text,'-',''),'vinculo_version',0,
  'huella_sha256','','revision_continuidad',revision,'procedencia_ref',cuenta->>'procedencia_ref',
  'procedencia_version',cuenta->'procedencia_version','procedencia_huella_sha256',cuenta->>'procedencia_huella_sha256',
  'vigente_hasta',fecha);
 acto:=jsonb_build_object('operacion','otorgar','rol_version_ref',p_rol,'objetivo',o);
 IF unidad IS NOT NULL THEN acto:=acto||jsonb_build_object('unidad_ref',unidad); END IF;
 pi:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 o:=jsonb_set(o,'{huella_sha256}',to_jsonb(encode(sha256(convert_to(pi::text,'UTF8')),'hex')));
 -- La unidad viaja en el DTO de lectura; al escribir está en el material raíz.
 IF unidad IS NOT NULL THEN o:=o||jsonb_build_object('unidad_ref',unidad); END IF;
 RETURN o;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.objetivo_alta_admin_lectura_interna_v1(jsonb,text)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;

CREATE FUNCTION vec_autorizacion.objetivo_baja_admin_lectura_interna_v1(m jsonb,p_asignacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE a record;cfg record;ca jsonb;o jsonb;acto jsonb;preimagen jsonb;unidad text;
BEGIN
 SELECT x.*,va.cuenta_ref,va.vinculo_ref INTO a FROM vec_autorizacion.asignacion_perfil x
 JOIN vec_autorizacion.asignacion_perfil_actual ac USING(asignacion_ref)
 JOIN vec_autorizacion.vinculacion_asignacion_admin_v1 va USING(asignacion_ref)
 WHERE x.asignacion_ref=p_asignacion AND x.principal_id=m->>'persona_ref' AND x.documento->>'estado'='activa';
 IF NOT FOUND THEN RETURN null; END IF;
 SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=a.version_rol_ref;
 IF m->>'actor_persona_ref'=a.principal_id AND cfg.clase<>'administrador' THEN RETURN null; END IF;
 ca:=vec_contexto_actor_v1.preimagen_admin_interna_v1(a.cuenta_ref,a.principal_id,a.perfil_activo_ref,a.vinculo_ref);
 IF ca#>>'{perfil,estado}' IS DISTINCT FROM 'activo' OR ca#>>'{vinculo,estado}' IS DISTINCT FROM 'activo' THEN RETURN null; END IF;
 IF cfg.unidad_requerida THEN
  SELECT e#>>'{valores,0}' INTO STRICT unidad FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='unidad' AND jsonb_array_length(e->'valores')=1;
 END IF;
 o:=jsonb_build_object('cuenta_ref',a.cuenta_ref,'cuenta_version',(ca#>>'{cuenta,version}')::numeric,'persona_ref',a.principal_id,'persona_version',(ca#>>'{persona,version}')::numeric,'perfil_ref',a.perfil_activo_ref,'perfil_version',(ca#>>'{perfil,version}')::numeric,'vinculo_ref',a.vinculo_ref,'vinculo_version',(ca#>>'{vinculo,version}')::numeric,'huella_sha256','','revision_continuidad',0,'procedencia_ref',ca#>>'{perfil,procedencia_ref}','procedencia_version',(ca#>>'{perfil,procedencia_version}')::numeric,'procedencia_huella_sha256',ca#>>'{perfil,procedencia_huella_sha256}','vigente_hasta','0001-01-01T00:00:00Z');
 acto:=jsonb_build_object('operacion','revocar','rol_version_ref',a.version_rol_ref,'objetivo',o,'unidad_ref',unidad);
 preimagen:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 IF cfg.clase<>'ordinario' THEN o:=jsonb_set(o,'{revision_continuidad}',preimagen#>'{continuidad,revision}'); END IF;
 acto:=jsonb_set(acto,'{objetivo}',o);preimagen:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 o:=jsonb_set(o,'{huella_sha256}',to_jsonb(encode(sha256(convert_to(preimagen::text,'UTF8')),'hex')));
 IF unidad IS NOT NULL THEN o:=o||jsonb_build_object('unidad_ref',unidad); END IF;
 RETURN o;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.objetivo_baja_admin_lectura_interna_v1(jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.motivos_admin_lectura_interna_v1(p_accion text,p_tipo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE motivo jsonb;
BEGIN
 motivo:=vec_autorizacion.motivo_admin_publicado_interno_v1(p_accion,p_tipo);
 RETURN jsonb_build_array(motivo||jsonb_build_object('etiqueta',''));
EXCEPTION WHEN no_data_found THEN RETURN '[]'::jsonb;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.motivos_admin_lectura_interna_v1(text,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(p_propuesta text,m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE p record; efectivos jsonb; continuidad record; vigente boolean; independiente boolean; motivos jsonb:='[]'::jsonb;
BEGIN
 SELECT * INTO p FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=p_propuesta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: propuesta no disponible' USING ERRCODE='P0002'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 SELECT * INTO STRICT continuidad FROM vec_autorizacion.control_continuidad_admin WHERE control_id;
 independiente:=m->>'actor_persona_ref' IS DISTINCT FROM p.proponente_persona_ref
  AND m->>'actor_persona_ref' IS DISTINCT FROM p.objetivo_persona_ref;
 vigente:=continuidad.bootstrap_estado='consumido' AND continuidad.revision=p.revision_continuidad_esperada
  AND clock_timestamp()<p.caduca_en
  AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible WHERE propuesta_ref=p.propuesta_ref)
  AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)>=2
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x
   WHERE x->>'persona_ref'=p.proponente_persona_ref AND x->>'perfil_ref'=p.proponente_perfil_ref)
  AND EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a
   JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
   WHERE a.version_rol_ref=p.version_rol_ref AND a.huella_sha256=r.huella_sha256
    AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
    AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta);
 motivos:=vec_autorizacion.motivos_admin_lectura_interna_v1('administracion.perfiles.aprobar','propuesta_perfil');
 RETURN jsonb_build_object('propuesta_ref',p.propuesta_ref,'proponente_persona_ref',p.proponente_persona_ref,
  'objetivo_persona_ref',p.objetivo_persona_ref,'objetivo_nombre','','rol_version_ref',p.version_rol_ref,
  'operacion',p.operacion,'huella_sha256',p.huella_sha256,'caduca_en',p.caduca_en,
  'puede_cerrar',independiente AND vigente AND jsonb_array_length(motivos)>0,'motivos_cierre',motivos);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(text,jsonb)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;

CREATE FUNCTION vec_autorizacion.proyectar_lectura_admin_interna_v1(m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE fuente jsonb;datos jsonb;historia jsonb;acciones jsonb;recibo jsonb;actos jsonb:='[]'::jsonb;r record;objetivo jsonb;motivos jsonb;
BEGIN
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1'
 OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') THEN
  RAISE EXCEPTION 'AUT24: proyeccion nominal invalida' USING ERRCODE='22023';
 END IF;
 -- Defensa interna adicional; solo las siete fachadas tienen EXECUTE runtime.
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x
 WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN
  RAISE EXCEPTION 'AUT24: lector no efectivo' USING ERRCODE='42501';
 END IF;
 CASE m->>'consulta'
 WHEN 'capacidades' THEN
  SELECT vr.documento INTO STRICT fuente FROM vec_autorizacion.asignacion_perfil_actual ac
  JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
  WHERE a.perfil_activo_ref=m->>'actor_perfil_ref' AND a.principal_id=m->>'actor_persona_ref'
   AND a.documento->>'estado'='activa' AND vr.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
   AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz;
  WITH concesiones AS (SELECT x->>'accion' accion FROM jsonb_array_elements(fuente->'concesiones') x
   WHERE x->>'modulo_id'='administracion' AND x->'finalidades' ? 'gestion_perfiles'),
  disponibles AS (
   SELECT 'consultar' codigo WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.consultar')
   UNION ALL SELECT 'aplicar_ordinario' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.otorgar')
    AND EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.revocar')
   UNION ALL SELECT 'proponer' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.proponer')
    AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)>=2
   UNION ALL SELECT 'cerrar_propuesta' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.aprobar')
    AND EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.rechazar')
    AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)>=2
  ) SELECT coalesce(jsonb_agg(codigo ORDER BY codigo),'[]'::jsonb) INTO acciones FROM disponibles;
  RETURN jsonb_build_object('version','v1','actor_persona_ref',m->>'actor_persona_ref','acciones',acciones);
 WHEN 'buscar_personas' THEN
  RETURN vec_contexto_actor_v1.buscar_personas_admin_interna_v1(m->>'busqueda',coalesce(m->>'cursor',''),50);
 WHEN 'consultar_persona' THEN
  fuente:=vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(m->>'persona_ref');
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.perfil_ref),'[]'::jsonb) INTO datos
  FROM (SELECT a.perfil_activo_ref AS perfil_ref,a.version_rol_ref AS rol_version_ref,
   p->>'estado' AS estado,(p->>'version')::numeric AS version,p->>'vigente_hasta' AS vigente_hasta
   FROM vec_autorizacion.asignacion_perfil_actual ac
   JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
   JOIN LATERAL jsonb_array_elements(fuente->'perfiles') p ON p->>'perfil_ref'=a.perfil_activo_ref
   WHERE a.principal_id=m->>'persona_ref' ORDER BY a.perfil_activo_ref LIMIT 50) t;
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.confirmado_en DESC,t.acto_ref),'[]'::jsonb) INTO historia
  FROM (SELECT rr->>'acto_ref' AS acto_ref,coalesce(proyeccion.material->>'operacion',propuesta.operacion) AS operacion,
   rr->>'estado_posterior' AS estado,(rr->>'confirmado_en')::timestamptz AS confirmado_en
   FROM vec_autorizacion.registro_acto_admin_v1 a
   CROSS JOIN LATERAL (SELECT convert_from(a.material,'UTF8')::jsonb AS material,
    CASE WHEN jsonb_typeof(a.resultado->'recibo')='object' THEN a.resultado->'recibo' ELSE a.resultado END AS rr) proyeccion
   LEFT JOIN vec_autorizacion.propuesta_perfil_sensible propuesta
    ON propuesta.propuesta_ref=proyeccion.material->>'propuesta_ref'
   WHERE rr->>'objetivo_persona_ref'=m->>'persona_ref' AND rr ? 'acto_ref'
   AND coalesce(proyeccion.material->>'operacion',propuesta.operacion) IN ('otorgar','revocar')
   ORDER BY (rr->>'confirmado_en')::timestamptz DESC,rr->>'acto_ref' LIMIT 50) t;
  FOR r IN SELECT a.* FROM vec_autorizacion.rol_administrable_exacto_v1 a
   JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=rc.version_rol_ref AND cv.revision=rc.revision
   WHERE a.huella_sha256=vr.huella_sha256 AND vr.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta ORDER BY a.version_rol_ref LIMIT 50 LOOP
   motivos:=vec_autorizacion.motivos_admin_lectura_interna_v1(CASE WHEN r.clase='ordinario' THEN 'administracion.perfiles.otorgar' ELSE 'administracion.perfiles.proponer' END,'perfil');
   IF jsonb_array_length(motivos)=0 THEN CONTINUE; END IF;
   objetivo:=vec_autorizacion.objetivo_alta_admin_lectura_interna_v1(m,r.version_rol_ref);
   IF objetivo IS NOT NULL THEN actos:=actos||jsonb_build_array(jsonb_build_object('operacion','otorgar','rol_version_ref',r.version_rol_ref,'objetivo',objetivo,'motivos',motivos)); END IF;
  END LOOP;
  FOR r IN SELECT a.asignacion_ref,a.version_rol_ref,c.clase FROM vec_autorizacion.asignacion_perfil_actual ac
   JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref) JOIN vec_autorizacion.rol_administrable_exacto_v1 c USING(version_rol_ref)
   WHERE a.principal_id=m->>'persona_ref' AND a.documento->>'estado'='activa' ORDER BY a.perfil_activo_ref LIMIT 50 LOOP
   motivos:=vec_autorizacion.motivos_admin_lectura_interna_v1(CASE WHEN r.clase='ordinario' THEN 'administracion.perfiles.revocar' ELSE 'administracion.perfiles.proponer' END,'perfil');
   IF jsonb_array_length(motivos)=0 THEN CONTINUE; END IF;
   objetivo:=vec_autorizacion.objetivo_baja_admin_lectura_interna_v1(m,r.asignacion_ref);
   IF objetivo IS NOT NULL AND jsonb_array_length(actos)<50 THEN actos:=actos||jsonb_build_array(jsonb_build_object('operacion','revocar','rol_version_ref',r.version_rol_ref,'objetivo',objetivo,'motivos',motivos)); END IF;
  END LOOP;
  RETURN jsonb_build_object('persona_ref',m->>'persona_ref','nombre','','unidad_nombre','',
   'perfiles',datos,'actos_disponibles',actos,'historia',historia);
 WHEN 'listar_roles' THEN
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.version_ref),'[]'::jsonb) INTO datos
  FROM (SELECT a.version_rol_ref AS version_ref,a.clase,vr.documento->>'nombre' AS clave_i18n,
   ''::text AS etiqueta,a.huella_sha256 FROM vec_autorizacion.rol_administrable_exacto_v1 a
   JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
   WHERE a.huella_sha256=vr.huella_sha256 AND vr.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta
   ORDER BY a.version_rol_ref LIMIT 50) t;
  RETURN jsonb_build_object('roles',datos);
 WHEN 'listar_propuestas' THEN
  SELECT coalesce(jsonb_agg(vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(t.propuesta_ref,m)
   ORDER BY t.creada_en DESC,t.propuesta_ref),'[]'::jsonb) INTO datos
  FROM (SELECT p.propuesta_ref,p.creada_en FROM vec_autorizacion.propuesta_perfil_sensible p
   WHERE NOT EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible c WHERE c.propuesta_ref=p.propuesta_ref)
   ORDER BY p.creada_en DESC,p.propuesta_ref LIMIT 50) t;
  RETURN jsonb_build_object('propuestas',datos);
 WHEN 'consultar_propuesta' THEN
  RETURN vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(m->>'propuesta_ref',m);
 WHEN 'consultar_recibo' THEN
  SELECT t.rr INTO STRICT recibo FROM vec_autorizacion.registro_acto_admin_v1 a
   CROSS JOIN LATERAL (SELECT CASE WHEN jsonb_typeof(a.resultado->'recibo')='object'
    THEN a.resultado->'recibo' ELSE a.resultado END AS rr) t
   WHERE t.rr->>'recibo_ref'=m->>'recibo_ref' AND t.rr ? 'acto_ref';
  -- STRICT rechaza una referencia ausente (P0002) o ambigua.
  RETURN recibo;
 ELSE RAISE EXCEPTION 'AUT24: consulta desconocida' USING ERRCODE='22023';
 END CASE;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_lectura_admin_interna_v1(jsonb)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
-- Ningún GRANT al runtime: el ensamblador publica únicamente siete wrappers.


CREATE FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'capacidades' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["capacidades"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.capacidades.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.buscar_personas_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'buscar_personas' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["personas","siguiente_cursor"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.buscar_personas_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.buscar_personas_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.consultar_persona_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'consultar_persona' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["persona"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.consultar_persona.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_persona_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_persona_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.listar_roles_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'listar_roles' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["roles"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.listar_roles.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_roles_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_roles_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.listar_propuestas_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'listar_propuestas' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.historial.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'historial_perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["propuestas"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.listar_propuestas.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_propuestas_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_propuestas_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.consultar_propuesta_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'consultar_propuesta' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.historial.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'historial_perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["propuesta"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.consultar_propuesta.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.consultar_recibo_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;d jsonb;c jsonb;consumo jsonb;efectivos jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: lectura invalida' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1' OR m->>'consulta' IS DISTINCT FROM 'consultar_recibo' OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') OR d->>'accion' IS DISTINCT FROM 'administracion.perfiles.recibo.consultar' OR d->>'tipo_recurso' IS DISTINCT FROM 'recibo_perfil' OR d->'campos_permitidos' IS DISTINCT FROM '["recibo"]'::jsonb OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.consultar_recibo.v1' THEN RAISE EXCEPTION 'AUT24: sello de lectura divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo->>'consumo_nuevo' IS DISTINCT FROM 'true' THEN RAISE EXCEPTION 'AUT24: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: lector no vigente' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion.proyectar_lectura_admin_interna_v1(m);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_recibo_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_recibo_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

RESET ROLE;
DO $rol_auditoria_frontera$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_auditoria_ejecutor') THEN CREATE ROLE vec_admin_perfiles_auditoria_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS; END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_auditoria_ejecutor' AND(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)) OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_perfiles_auditoria_ejecutor'::regrole) THEN RAISE EXCEPTION 'AUT24: rol_auditoria esperado=aislado observado=privilegiado' USING ERRCODE='55000'; END IF;
END $rol_auditoria_frontera$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.denegacion_frontera_admin_v1(
 operacion_ref text PRIMARY KEY,
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 4096),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(material),'hex')),
 operador_sql text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
ALTER TABLE vec_autorizacion.denegacion_frontera_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.denegacion_frontera_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.denegacion_frontera_admin_v1 FOR ALL TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER denegacion_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.denegacion_frontera_admin_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER denegacion_no_truncable BEFORE TRUNCATE ON vec_autorizacion.denegacion_frontera_admin_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.denegacion_frontera_admin_v1 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_admin_perfiles_bootstrap_ejecutor,vec_admin_perfiles_auditoria_ejecutor,vec_autorizacion_fuente;
REVOKE ALL ON TYPE vec_autorizacion.denegacion_frontera_admin_v1 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_admin_perfiles_bootstrap_ejecutor,vec_admin_perfiles_auditoria_ejecutor,vec_autorizacion_fuente;
CREATE FUNCTION vec_autorizacion.registrar_denegacion_frontera_admin_v1(p_datos text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;r record;claves text[];audit text;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: auditoria requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 IF NOT pg_has_role(session_user,'vec_admin_perfiles_auditoria_ejecutor','MEMBER') OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid='vec_admin_perfiles_auditoria_ejecutor'::regrole AND inherit_option AND NOT set_option AND NOT admin_option) OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=session_user::regrole AND(NOT rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)) THEN RAISE EXCEPTION 'AUT24: auditor no autorizado' USING ERRCODE='42501'; END IF;
 IF p_datos IS NULL OR octet_length(p_datos) NOT BETWEEN 1 AND 4096 THEN RAISE EXCEPTION 'AUT24: evento de frontera invalido' USING ERRCODE='22023'; END IF;
 m:=p_datos::jsonb;SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF NOT m ?& ARRAY['codigo','accion','recurso_ref','operacion_ref'] OR claves <@ ARRAY['codigo','accion','recurso_ref','operacion_ref','actor_persona_ref','perfil_activo_ref','correlacion_ref'] IS NOT TRUE OR jsonb_path_exists(m,'$.** ? (@ == null)') OR (m->>'codigo' ~ '^[a-z][a-z0-9_.:-]{1,127}$') IS NOT TRUE OR (m->>'accion' ~ '^[a-z][a-z0-9_.:-]{1,255}$') IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m->>'recurso_ref',512) IS NOT TRUE OR (m->>'operacion_ref' ~ '^frontera_admin:[0-9a-f]{32}$') IS NOT TRUE OR (m ? 'actor_persona_ref' AND (m->>'actor_persona_ref' ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE) OR (m ? 'perfil_activo_ref' AND (m->>'perfil_activo_ref' ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE) OR (m ? 'correlacion_ref' AND vec_autorizacion.texto_positivo_valido(m->>'correlacion_ref',256) IS NOT TRUE) THEN RAISE EXCEPTION 'AUT24: auditoria rechaza datos libres' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:frontera:'||(m->>'operacion_ref'),0));
 SELECT * INTO r FROM vec_autorizacion.denegacion_frontera_admin_v1 WHERE operacion_ref=m->>'operacion_ref';
 IF FOUND THEN IF r.material IS DISTINCT FROM convert_to(p_datos,'UTF8') THEN RAISE EXCEPTION 'AUT24: auditoria idempotente divergente' USING ERRCODE='23505'; END IF; RETURN jsonb_build_object('operacion_ref',r.operacion_ref,'auditoria_ref',r.auditoria_ref,'registrada_en',r.registrada_en); END IF;
 ahora:=clock_timestamp();audit:='auditoria_frontera:'||encode(sha256(convert_to(p_datos,'UTF8')),'hex');
 INSERT INTO vec_autorizacion.denegacion_frontera_admin_v1 VALUES(m->>'operacion_ref',convert_to(p_datos,'UTF8'),encode(sha256(convert_to(p_datos,'UTF8')),'hex'),session_user,audit,ahora);
 RETURN jsonb_build_object('operacion_ref',m->>'operacion_ref','auditoria_ref',audit,'registrada_en',ahora);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_denegacion_frontera_admin_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_auditoria_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_denegacion_frontera_admin_v1(text) TO vec_admin_perfiles_auditoria_ejecutor;

DO $acl_bootstrap_y_lectura$
BEGIN
 IF has_function_privilege('vec_admin_perfiles_ejecutor','vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text)','EXECUTE')
 OR has_function_privilege('vec_admin_perfiles_bootstrap_ejecutor','vec_autorizacion.preparar_preimagen_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_table_privilege('vec_admin_perfiles_ejecutor','vec_autorizacion.bootstrap_admin_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 OR has_table_privilege('vec_admin_perfiles_bootstrap_ejecutor','vec_autorizacion.bootstrap_admin_v2','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member IN('vec_admin_perfiles_ejecutor'::regrole,'vec_admin_perfiles_bootstrap_ejecutor'::regrole))
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid IN('vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text)'::regprocedure,'vec_autorizacion.preparar_preimagen_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
 AND(a.grantee=0 OR a.is_grantable)) THEN RAISE EXCEPTION 'AUT24: ACL_bootstrap_lectura esperado=segregado observado=abierto' USING ERRCODE='55000'; END IF;
END $acl_bootstrap_y_lectura$;
COMMIT;
