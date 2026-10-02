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
