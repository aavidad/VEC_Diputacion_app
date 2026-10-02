\set ON_ERROR_STOP on
-- CT164. Circuito R5 dentro de las confirmaciones y versiones existentes.
-- Preimagen de funciones: clon frío sintético hito1 PG18.4 (29/09/2026).
-- Dependencia causal CT163: guardas estructurales y lectura nominal AD151.
-- El flujo inicial procede del resolutor Go y del material/HMAC atestado del alta.
-- Esta migración no publica perfiles ni configura fuentes desde una petición.
-- Sin fuente nominal de firma de Técnico/Delegación, no se confirma el análisis
-- R5 ni se acepta una historia inventada que permita saltar a crédito/Bolsa.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000164',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR to_regprocedure('vec_contratacion_temporal.circuito_vinculado_ct163(jsonb)') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.circuito_siguiente_ct163(jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.circuito_flujo_nuevo_ct164(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT164: dependencia CT163 o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Terna de la definición gobernada que publica el operador en configuración.
-- La decisión, el material del alta y los sellos existentes ligan este flujo.
CREATE FUNCTION vec_contratacion_temporal.circuito_flujo_nuevo_ct164(p_flujo jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT coalesce(p_flujo = '{"definicion_ref":"flujo:ct:rrhh:20261002","version":2,"huella_sha256":"f9b83c1291fdf96f339233b9e7a2036b67803388cea8568b4d2b02b6bcd4e9fc"}'::jsonb,false)
$f$;

CREATE FUNCTION vec_contratacion_temporal.circuito_agregado_valido_ct164(p_agregado jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 IF p_agregado IS NULL OR jsonb_typeof(p_agregado) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
 IF p_agregado #>> '{flujo,definicion_ref}' IS DISTINCT FROM 'flujo:ct:rrhh:20261002' THEN
  RETURN NOT (p_agregado ? 'circuito');
 END IF;
 IF vec_contratacion_temporal.circuito_flujo_nuevo_ct164(p_agregado->'flujo') IS NOT TRUE
  OR vec_contratacion_temporal.circuito_vinculado_ct163(p_agregado) IS NOT TRUE
  OR vec_contratacion_temporal.claves_json_exactas_v1(p_agregado->'circuito',
    ARRAY['definicion','estado_actual','hitos']) IS NOT TRUE THEN RETURN false; END IF;
 IF p_agregado->>'version'='1' THEN
  RETURN p_agregado->>'fase_actual'='solicitud' AND p_agregado->>'estado_actual'='en_curso'
   AND p_agregado->'circuito'=jsonb_build_object('definicion',p_agregado->'flujo',
    'estado_actual','solicitud','hitos','[]'::jsonb);
 END IF;
 RETURN true;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.circuito_acto_admitido_ct164(p_anterior jsonb,p_siguiente jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 IF vec_contratacion_temporal.circuito_agregado_valido_ct164(p_anterior) IS NOT TRUE
  OR vec_contratacion_temporal.circuito_agregado_valido_ct164(p_siguiente) IS NOT TRUE THEN RETURN false; END IF;
 IF NOT (p_anterior ? 'circuito') AND NOT (p_siguiente ? 'circuito') THEN RETURN true; END IF;
 -- El consumo V3 del acto anterior no acredita firmas de cargos, crédito ni
 -- oferta/adjudicación de Bolsa. No existe hoy un puerto nominal SQL que pueda
 -- verificar esas evidencias. Se deniega incluso con referencias/huellas o
 -- declaraciones positivas del cliente. Su habilitación exige otra migración
 -- revisada con la fuente y el vínculo exactos, sin una bandera configurable.
 RETURN false;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.circuito_proyectar_acto_ct164(
 p_anterior jsonb,p_siguiente jsonb,p_esperado jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 IF vec_contratacion_temporal.circuito_acto_admitido_ct164(p_anterior,p_siguiente) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT164: acto del circuito sin fuente nominal verificada' USING ERRCODE='42501'; END IF;
 IF p_siguiente ? 'circuito' THEN
  RETURN jsonb_set(p_esperado,'{circuito}',p_siguiente->'circuito',true);
 END IF;
 RETURN p_esperado;
END $f$;

-- Defensa en la única tabla de versiones: no se omite el circuito al usar la
-- nueva terna y ninguna operación secundaria puede añadir versiones/hitos falsos.
CREATE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct164()
RETURNS trigger LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE v_anterior jsonb; v_anterior_circuito boolean:=false;
BEGIN
 IF NEW.version>1 THEN
  SELECT agregado_json INTO v_anterior FROM vec_contratacion_temporal.expediente_version_integral
   WHERE expediente_ref=NEW.expediente_ref AND version=NEW.version-1;
  v_anterior_circuito:=coalesce(v_anterior ? 'circuito',false);
 END IF;
 IF v_anterior_circuito OR NEW.flujo_ref='flujo:ct:rrhh:20261002' OR NEW.agregado_json ? 'circuito' THEN
  IF vec_contratacion_temporal.circuito_agregado_valido_ct164(NEW.agregado_json) IS NOT TRUE
   OR NEW.agregado_json #>> '{flujo,definicion_ref}' IS DISTINCT FROM NEW.flujo_ref
   OR (NEW.agregado_json #>> '{flujo,version}')::numeric IS DISTINCT FROM NEW.flujo_version
   OR NEW.agregado_json #>> '{flujo,huella_sha256}' IS DISTINCT FROM NEW.flujo_huella_sha256
   OR NEW.agregado_json->>'version' IS DISTINCT FROM NEW.version::text
  THEN RAISE EXCEPTION 'CT164: terna o circuito inicial divergente' USING ERRCODE='42501'; END IF;
  IF NEW.version>1 THEN
   IF v_anterior IS NULL OR vec_contratacion_temporal.circuito_acto_admitido_ct164(v_anterior,NEW.agregado_json) IS NOT TRUE
   THEN RAISE EXCEPTION 'CT164: acto del circuito sin fuente nominal verificada' USING ERRCODE='42501'; END IF;
  END IF;
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.circuito_flujo_nuevo_ct164(jsonb),
 vec_contratacion_temporal.circuito_agregado_valido_ct164(jsonb),
 vec_contratacion_temporal.circuito_acto_admitido_ct164(jsonb,jsonb),
 vec_contratacion_temporal.circuito_proyectar_acto_ct164(jsonb,jsonb,jsonb),
 vec_contratacion_temporal.proteger_version_circuito_ct164()
 FROM PUBLIC;
CREATE TRIGGER expediente_version_integral_circuito_ct164
 BEFORE INSERT ON vec_contratacion_temporal.expediente_version_integral
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.proteger_version_circuito_ct164();

-- Parche nominal materializar_version_inicial_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.materializar_version_inicial_v1(text,numeric,bytea,text,numeric,text,text,text,timestamp with time zone)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    v_agregado_huella := pg_catalog.encode(
$marca$;
 reemplazo text:=$reemplazo$    -- CT164: la terna del flujo llega en el material atestado, no en HTTP.
    IF p_flujo_ref='flujo:ct:rrhh:20261002' THEN
        IF vec_contratacion_temporal.circuito_flujo_nuevo_ct164(v_efecto->'flujo') IS NOT TRUE
        THEN RAISE EXCEPTION 'CT164: definición R5 divergente' USING ERRCODE='42501'; END IF;
        v_agregado := jsonb_set(v_agregado,'{circuito}',jsonb_build_object(
            'definicion',v_efecto->'flujo','estado_actual','solicitud','hitos','[]'::jsonb),true);
    END IF;
    v_agregado_huella := pg_catalog.encode(
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente materializar_version_inicial_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '0b1872b16ad62786eaccea72d4981fd76839adee57a9093b2dd6bed8b1108bd0'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM 'dda0fe399101565ffb8f362060efa052bdd2d4d74447429fde11da1e3fea818a'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible materializar_version_inicial_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados materializar_version_inicial_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal expediente_analisis_valido_v2: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    IF p_exige_analisis IS NULL
$marca$;
 reemplazo text:=$reemplazo$    IF vec_contratacion_temporal.circuito_agregado_valido_ct164(e) IS NOT TRUE THEN RETURN false; END IF;
    IF e ? 'circuito' THEN
        SELECT array_agg(k ORDER BY k) INTO v_claves FROM unnest(array_append(v_claves,'circuito')) k;
    END IF;
    IF p_exige_analisis IS NULL
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente expediente_analisis_valido_v2' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'b3b06bff38b7289676c4bfbd7ee6b7c20c53e7cb08a2660d7ccdb7b194a69705'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '9eb4e8b00da3745d27f6d28898575ed15422ed8e67426ca3c61dbe36ee4f39a4'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible expediente_analisis_valido_v2' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados expediente_analisis_valido_v2' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal transicion_confirmacion_analisis_valida_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(jsonb,jsonb)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$BEGIN
    IF vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2($marca$;
 reemplazo text:=$reemplazo$BEGIN
    IF vec_contratacion_temporal.circuito_acto_admitido_ct164(anterior,siguiente) IS NOT TRUE THEN RETURN false; END IF;
    IF vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2($reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente transicion_confirmacion_analisis_valida_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'd8b0f30326d0d1badf826b4bff790a7920aa54b783798537baff610fcd5004d5'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '09886769afbd66f637233fa5e2a3c224997f5884122f7bf189ba34f1d6a36f2a'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible transicion_confirmacion_analisis_valida_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados transicion_confirmacion_analisis_valida_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal confirmar_operacion_analisis_v3: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    RETURN QUERY
$marca$;
 reemplazo text:=$reemplazo$    IF vec_contratacion_temporal.circuito_acto_admitido_ct164(
        o->'expediente_anterior',o->'expediente_siguiente') IS NOT TRUE
    THEN RAISE EXCEPTION 'CT164: análisis del circuito sin fuente nominal verificada' USING ERRCODE='42501'; END IF;
    RETURN QUERY
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente confirmar_operacion_analisis_v3' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '35ac905214da14f99a881f46b34d563a546e6afd2768ead28a2c999f0bac018f'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '0e964cb6f8c68a4fd64d62b75d202b4ca481c3394f0f21ae21e7528f7832e54e'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible confirmar_operacion_analisis_v3' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados confirmar_operacion_analisis_v3' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal o404e_transicion_exacta_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.o404e_transicion_exacta_v1(jsonb,jsonb,jsonb,jsonb,timestamp with time zone)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$BEGIN
    IF pg_catalog.jsonb_typeof(p_anterior)$marca$;
 reemplazo text:=$reemplazo$BEGIN
    IF vec_contratacion_temporal.circuito_acto_admitido_ct164(p_anterior,p_siguiente) IS NOT TRUE THEN RETURN false; END IF;
    IF pg_catalog.jsonb_typeof(p_anterior)$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente o404e_transicion_exacta_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '841486956d55f53192d1e53352337ee97db3aac553d7d1d013d88a7611897413'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM 'b4df981a3c9e4c36f13ffb383f74c97c7b08b9726a81209dfe9d985a90ce9c26'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible o404e_transicion_exacta_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados o404e_transicion_exacta_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal confirmar_asignacion_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.confirmar_asignacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    IF p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
$marca$;
 reemplazo text:=$reemplazo$    v_expediente_esperado := vec_contratacion_temporal.circuito_proyectar_acto_ct164(
        v_actual.agregado_json,p_operacion->'expediente_siguiente',v_expediente_esperado);
    IF p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente confirmar_asignacion_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'f20c58b07740b8e2b5907d1d9e017d649b641812de0ce6ded41c64583ab02276'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '65466bed9b1461df2550b57afa1a0f8d008593e9a3ff92417030fd05ef2aef56'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible confirmar_asignacion_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados confirmar_asignacion_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal confirmar_informe_juridico_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.confirmar_informe_juridico_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    IF pg_catalog.jsonb_array_length(
           v_actual.agregado_json -> 'actuaciones') <> 4
$marca$;
 reemplazo text:=$reemplazo$    v_expediente_esperado := vec_contratacion_temporal.circuito_proyectar_acto_ct164(
        v_actual.agregado_json,p_operacion->'expediente_siguiente',v_expediente_esperado);
    IF pg_catalog.jsonb_array_length(
           v_actual.agregado_json -> 'actuaciones') <> 4
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente confirmar_informe_juridico_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'c71c053abf0f2f98fb01b7534d55f5e5c56e4ede92e81822911c36b7fe1da39e'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM 'bde7c18fd552f86124c9b4dc2dd7fb7864a3a6339cf545c876cb0498cf929d1e'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible confirmar_informe_juridico_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados confirmar_informe_juridico_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Parche nominal confirmar_fiscalizacion_v1: preimagen exacta y metadatos conservados.
DO $patch$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 marca text:=$marca$    IF pg_catalog.jsonb_array_length(
           v_actual.agregado_json -> 'actuaciones') <> 5
$marca$;
 reemplazo text:=$reemplazo$    v_expediente_esperado := vec_contratacion_temporal.circuito_proyectar_acto_ct164(
        v_actual.agregado_json,p_operacion->'expediente_siguiente',v_expediente_esperado);
    IF pg_catalog.jsonb_array_length(
           v_actual.agregado_json -> 'actuaciones') <> 5
$reemplazo$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'CT164: función ausente confirmar_fiscalizacion_v1' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
  INTO deps_compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '27e369c38267d18d3ed258eca1a986c5835717d9c9630740d2291033fddeb147'
  OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM 'bb21608248394f30ef120618c26c5a753d93d0ba12fd98f69a6953ea9154aecf'
  OR length(original)-length(replace(original,marca,''))<>length(marca)
  OR strpos(original,'ct164')<>0
 THEN RAISE EXCEPTION 'CT164: preimagen incompatible confirmar_fiscalizacion_v1' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'CT164: metadatos alterados confirmar_fiscalizacion_v1' USING ERRCODE='55000'; END IF;
END $patch$;

-- Los helpers son internos: tampoco se conserva acceso por ACL predeterminada.
DO $acl$
DECLARE f regprocedure; firma text; a record;
BEGIN
 FOREACH firma IN ARRAY ARRAY[
 'vec_contratacion_temporal.circuito_flujo_nuevo_ct164(jsonb)',
 'vec_contratacion_temporal.circuito_agregado_valido_ct164(jsonb)',
 'vec_contratacion_temporal.circuito_acto_admitido_ct164(jsonb,jsonb)',
 'vec_contratacion_temporal.circuito_proyectar_acto_ct164(jsonb,jsonb,jsonb)',
 'vec_contratacion_temporal.proteger_version_circuito_ct164()'] LOOP
  f:=firma::regprocedure;
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
  END LOOP;
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
   OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog']
   OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
       WHERE p.oid=f AND (x.grantee<>p.proowner OR x.privilege_type<>'EXECUTE' OR x.is_grantable))
  THEN RAISE EXCEPTION 'CT164: ACL helper abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
