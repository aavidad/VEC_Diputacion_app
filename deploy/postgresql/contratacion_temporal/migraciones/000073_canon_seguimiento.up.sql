\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000073:seguimiento:v1',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $$ BEGIN
 IF current_setting('server_encoding')<>'UTF8' OR EXISTS (
  SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND
   (p.proname LIKE 'seguimiento73\_%' ESCAPE '\' OR p.proname='estado_seguimiento_canonico_v1')) THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='codec seguimiento: precondicion incompatible';
 END IF;
END $$;

-- Gramática privada. NULL SQL jamás produce una aceptación por lógica ternaria.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_exigir(ok boolean) RETURNS void
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$ BEGIN
 IF ok IS DISTINCT FROM true THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='seguimiento canonico invalido';
 END IF;
END $$;

CREATE FUNCTION vec_contratacion_temporal.seguimiento73_forma(j jsonb, requeridas text[], opcionales text[] DEFAULT '{}') RETURNS void
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$ BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='object');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(j ?& requeridas AND NOT EXISTS (
  SELECT 1 FROM jsonb_object_keys(j) k WHERE NOT k=ANY(requeridas||opcionales)));
END $$;

CREATE FUNCTION vec_contratacion_temporal.seguimiento73_micro(j jsonb, permite_cero boolean DEFAULT false) RETURNS bigint
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE s text; t timestamptz; r bigint;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='string');
 s:=j#>>'{}';
 -- JSON UTC explícito; fracciones adicionales sólo si son ceros (precisión Go µs).
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(s ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]{1,6}0*)?Z$' AND left(s,4)<>'0000');
 t:=s::timestamptz;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS')=left(s,19));
 r:=(extract(epoch FROM t)*1000000)::bigint;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(permite_cero OR r<>-62135596800000000);
 RETURN r;
EXCEPTION WHEN datetime_field_overflow OR invalid_datetime_format OR numeric_value_out_of_range THEN
 RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='seguimiento canonico invalido';
END $$;

CREATE FUNCTION vec_contratacion_temporal.seguimiento73_escalar(j jsonb, tipo text) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE s text; n numeric; b bytea;
BEGIN
 IF tipo='bool' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='boolean');
  RETURN CASE WHEN j='true'::jsonb THEN decode('01','hex') ELSE decode('00','hex') END;
 ELSIF tipo='micro' THEN RETURN int8send(vec_contratacion_temporal.seguimiento73_micro(j));
 ELSIF tipo IN ('u64','u1') THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='number' AND j::text ~ '^(0|[1-9][0-9]*)$');
  n:=j::text::numeric;
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(n<=18446744073709551615 AND (tipo='u64' OR n>0));
  IF n>9223372036854775807 THEN n:=n-18446744073709551616; END IF;
  RETURN int8send(n::bigint);
 END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='string');
 s:=j#>>'{}';
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(CASE tipo
  WHEN 'ref' THEN s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64)
  WHEN 'hash' THEN s ~ '^[0-9a-f]{64}$' AND s<>repeat('0',64)
  WHEN 'clave' THEN s ~ '^[a-z][a-z0-9._-]{1,79}$'
  WHEN 'clase' THEN s IN ('ordinaria','rectificacion','reapertura')
  WHEN 'efecto' THEN s IN ('ninguno','abrir','ampliar','cerrar','rectificar_tramo','rectificar_cese','reabrir')
  ELSE false END);
 b:=convert_to(s,'UTF8'); RETURN int4send(octet_length(b))||b;
END $$;

-- Arreglos de DTO Go: sólo los slices que realmente admiten nil aceptan null.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_array(j jsonb, maximo int, minimo int DEFAULT 0, nulo boolean DEFAULT false) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$ BEGIN
 IF nulo AND j='null'::jsonb THEN j:='[]'::jsonb; END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_typeof(j)='array');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(jsonb_array_length(j) BETWEEN minimo AND maximo);
 RETURN j;
END $$;

-- Gramática binaria cerrada; los checks de máquina de estados están separados.
-- Prefijo ? = presencia opcional, * = campo validado fuera de este escritor.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_nodo(j jsonb, tipo text) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE spec text; par text; k text; t text; requeridas text[]:='{}'; opcionales text[]:='{}';
 b bytea:=''::bytea; x bytea; v jsonb; a jsonb; dominio text; opc boolean;
BEGIN
 IF tipo IN ('bool','micro','u64','u1','ref','hash','clave','clase','efecto') THEN
  RETURN vec_contratacion_temporal.seguimiento73_escalar(j,tipo);
 END IF;
 IF left(tipo,1)='[' THEN
  t:=split_part(substr(tipo,2),',',1);
  a:=vec_contratacion_temporal.seguimiento73_array(j,split_part(tipo,',',2)::int,0,split_part(tipo,',',3)='nil');
  b:=int4send(jsonb_array_length(a));
  FOR v IN SELECT value FROM jsonb_array_elements(a) LOOP
   x:=vec_contratacion_temporal.seguimiento73_nodo(v,t);
   IF t='actuacion' THEN x:=int4send(octet_length(x))||x||vec_contratacion_temporal.seguimiento73_escalar(v->'huella_actuacion_sha256','hash'); END IF;
   b:=b||x;
  END LOOP;
  RETURN b;
 END IF;
 CASE tipo
 WHEN 'defref' THEN spec:='referencia:ref|version:u1|huella_sha256:hash';
 WHEN 'intervalo' THEN spec:='desde:micro|hasta:micro';
 WHEN 'vigencia' THEN spec:='desde:micro|hasta:vigencia_fin';
 WHEN 'documento' THEN spec:='tipo_clave:clave|referencia:ref';
 WHEN 'requisito' THEN spec:='tipo_clave:clave|obligatorio:bool';
 WHEN 'estado_def' THEN spec:='clave:clave|final:bool';
 WHEN 'cal_req' THEN spec:='ambitos_permitidos:[clave,64,no|resultados_permitidos:[clave,64,no';
 WHEN 'calendario' THEN spec:='referencia:ref|version:u1|huella_sha256:hash|ambito_territorial_clave:clave|resultado_clave:clave|calculado_en:micro';
 WHEN 'periodo_resultante' THEN spec:='intervalo:intervalo|actuacion_ref:ref';
 WHEN 'cese' THEN spec:='efectivo_en:micro|actuacion_ref:ref';
 WHEN 'transicion' THEN spec:='clave:clave|origen:clave|destino:clave|clase:clase|motivos_permitidos:[clave,256,nil|motivo_obligatorio:bool|documentos:[requisito,32,nil|?calendario:cal_req|requiere_periodo:bool|efecto_periodo:efecto|exige_actor_distinto:bool';
 WHEN 'publicacion' THEN
  dominio:='definicion';
  spec:='referencia:ref|version:u1|publicado_en:micro|vigencia:vigencia|estado_inicial:clave|prohibe_ciclos_silenciosos:bool|estados:[estado_def,128,no|motivos:[clave,256,nil|transiciones:[transicion,512,no|*canon:skip|*huella_sha256:skip';
 WHEN 'raiz' THEN
  dominio:='raiz'; spec:='referencia:ref|organizacion_ref:ref|expediente_ref:ref|relacion_ref:ref|definicion:defref|estado_actual:clave|periodo_previsto:intervalo|creado_en:micro';
 WHEN 'peticion','actuacion' THEN
  dominio:=tipo;
  spec:='actuacion_ref:ref|transicion_clave:clave|?motivo_clave:clave|actor_ref:ref|unidad_ref:ref|efectivo_en:micro|registrada_en:micro|documentos:[documento,32,nil|?periodo:intervalo|?calendario:calendario|recibo_ref:ref|correlacion_ref:ref|?rectifica_actuacion_ref:ref';
  IF tipo='actuacion' THEN spec:='secuencia:u1|version_seguimiento:u1|definicion:defref|clase:clase|estado_origen:clave|estado_destino:clave|'||spec||'|huella_peticion_sha256:hash|huella_anterior_sha256:hash|*huella_actuacion_sha256:skip'; END IF;
 WHEN 'estado' THEN
  dominio:='estado'; spec:='referencia:ref|organizacion_ref:ref|expediente_ref:ref|relacion_ref:ref|definicion:defref|version:u64|estado_actual:clave|periodo_previsto:intervalo|creado_en:micro|actualizado_en:micro|huella_raiz_sha256:hash|periodos_resultantes:[periodo_resultante,10000,nil|?cese_efectivo:cese|actuaciones:[actuacion,10000,no';
 ELSE RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='tipo canon seguimiento invalido';
 END CASE;
 FOREACH par IN ARRAY string_to_array(spec,'|') LOOP
  k:=split_part(par,':',1);
  IF left(k,1)='?' THEN opcionales:=array_append(opcionales,substr(k,2));
  ELSE requeridas:=array_append(requeridas,ltrim(k,'*')); END IF;
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_forma(j,requeridas,opcionales);
 IF dominio IS NOT NULL THEN
  x:=convert_to('vec.dipgra.contratacion-temporal.seguimiento.'||dominio,'UTF8');
  b:=int4send(octet_length(x))||x||int2send(1::smallint)||int4send(7)||convert_to('sha-256','UTF8');
 END IF;
 FOREACH par IN ARRAY string_to_array(spec,'|') LOOP
  k:=split_part(par,':',1); t:=split_part(par,':',2); opc:=left(k,1)='?';
  IF left(k,1)='*' THEN CONTINUE; END IF;
  IF opc THEN
   k:=substr(k,2);
   IF NOT j ? k OR (t IN ('clave','ref') AND j->k='""'::jsonb) THEN b:=b||decode('00','hex'); CONTINUE; END IF;
   b:=b||decode('01','hex');
  END IF;
  IF t='vigencia_fin' THEN
   IF vec_contratacion_temporal.seguimiento73_micro(j->k,true)=-62135596800000000 THEN b:=b||decode('00','hex');
   ELSE b:=b||decode('01','hex')||int8send(vec_contratacion_temporal.seguimiento73_micro(j->k)); END IF;
  ELSE b:=b||vec_contratacion_temporal.seguimiento73_nodo(j->k,t); END IF;
 END LOOP;
 IF tipo='intervalo' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(j->'hasta')>vec_contratacion_temporal.seguimiento73_micro(j->'desde'));
 ELSIF tipo='vigencia' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(j->'hasta',true)=-62135596800000000 OR vec_contratacion_temporal.seguimiento73_micro(j->'hasta',true)>vec_contratacion_temporal.seguimiento73_micro(j->'desde'));
 END IF;
 RETURN b;
END $$;
-- Normaliza sólo colecciones que PublicarDefinicion ordena. Nunca ordena historia.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_ordenar(a jsonb, campo text DEFAULT '') RETURNS jsonb
LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog
BEGIN ATOMIC
 SELECT coalesce(jsonb_agg(value ORDER BY (CASE WHEN campo='' THEN value#>>'{}' ELSE value->>campo END) COLLATE "C"),'[]'::jsonb)
 FROM jsonb_array_elements(a);
END;

CREATE FUNCTION vec_contratacion_temporal.seguimiento73_definicion(p jsonb) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE q jsonb:=p; t jsonb; e jsonb; c jsonb; ts jsonb:='[]'; estados jsonb:='{}';
 motivos jsonb; lista jsonb; clave text; origen text; destino text; clase text; efecto text;
 vistos text[]:='{}'; ofinal boolean; dfinal boolean; compatibles boolean; ciclos boolean;
BEGIN
 -- La forma y los tipos se verifican ANTES de casts o normalización.
 PERFORM vec_contratacion_temporal.seguimiento73_nodo(p,'publicacion');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(p->'canon'='{"dominio":"vec.dipgra.contratacion-temporal.seguimiento.definicion","version_esquema":1,"algoritmo":"sha-256"}'::jsonb AND p#>>'{canon,version_esquema}'='1');
 PERFORM vec_contratacion_temporal.seguimiento73_escalar(p->'huella_sha256','hash');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(p->'publicado_en')<=vec_contratacion_temporal.seguimiento73_micro(p#>'{vigencia,desde}'));
 q:=jsonb_set(q,'{estados}',vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(p->'estados',128,2),'clave'));
 motivos:=vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(p->'motivos',256,0,true));
 PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value) FROM jsonb_array_elements(motivos)));
 q:=jsonb_set(q,'{motivos}',motivos);
 FOR e IN SELECT value FROM jsonb_array_elements(q->'estados') LOOP
  clave:=e->>'clave';
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(NOT estados ? clave);
  estados:=estados||jsonb_build_object(clave,e->'final');
 END LOOP;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(estados->(p->>'estado_inicial')='false'::jsonb AND EXISTS (SELECT 1 FROM jsonb_each(estados) WHERE value='true'::jsonb));
 FOR t IN SELECT value FROM jsonb_array_elements(vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(p->'transiciones',512,1),'clave')) LOOP
  clave:=t->>'clave'; origen:=t->>'origen'; destino:=t->>'destino'; clase:=t->>'clase'; efecto:=t->>'efecto_periodo';
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(NOT clave=ANY(vistos) AND estados ? origen AND estados ? destino);
  vistos:=array_append(vistos,clave); ofinal:=(estados->>origen)::boolean; dfinal:=(estados->>destino)::boolean;
  compatibles:=CASE clase
   WHEN 'ordinaria' THEN efecto IN ('ninguno','abrir','ampliar') OR (efecto='cerrar' AND dfinal)
   WHEN 'rectificacion' THEN efecto IN ('ninguno','rectificar_tramo') OR (efecto='rectificar_cese' AND ofinal AND dfinal)
   WHEN 'reapertura' THEN ofinal AND NOT dfinal AND efecto='reabrir' ELSE false END;
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(compatibles AND NOT (ofinal AND clase='ordinaria') AND NOT (ofinal AND clase='rectificacion' AND NOT dfinal)
   AND (clase='rectificacion' OR t->'exige_actor_distinto'='false'::jsonb)
   AND (efecto IN ('abrir','ampliar','rectificar_tramo'))=(t->>'requiere_periodo')::boolean);
  lista:=vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(t->'motivos_permitidos',256,0,true));
  PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value) FROM jsonb_array_elements(lista)) AND motivos @> lista
   AND (t->'motivo_obligatorio'='false'::jsonb OR jsonb_array_length(lista)>0));
  t:=jsonb_set(t,'{motivos_permitidos}',lista);
  lista:=vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(t->'documentos',32,0,true),'tipo_clave');
  PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value->>'tipo_clave') FROM jsonb_array_elements(lista)));
  t:=jsonb_set(t,'{documentos}',lista);
  IF t ? 'calendario' THEN
   c:=t->'calendario';
   FOREACH clave IN ARRAY ARRAY['ambitos_permitidos','resultados_permitidos'] LOOP
    lista:=vec_contratacion_temporal.seguimiento73_ordenar(vec_contratacion_temporal.seguimiento73_array(c->clave,64,1));
    PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value) FROM jsonb_array_elements(lista)));
    c:=jsonb_set(c,ARRAY[clave],lista);
   END LOOP;
   t:=jsonb_set(t,'{calendario}',c);
  END IF;
  ts:=ts||jsonb_build_array(t);
 END LOOP;
 q:=jsonb_set(q,'{transiciones}',ts);
 IF (q->>'prohibe_ciclos_silenciosos')::boolean THEN
  -- Cierre transitivo finito por UNION (no enumeración exponencial de caminos).
  WITH RECURSIVE aristas AS (
   SELECT value->>'origen' o,value->>'destino' d FROM jsonb_array_elements(ts)
   WHERE value->>'clase'='ordinaria' AND value->'motivo_obligatorio'='false'::jsonb
    AND value->'requiere_periodo'='false'::jsonb AND value->>'efecto_periodo'='ninguno'
    AND NOT value ? 'calendario' AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(value->'documentos') z WHERE z->'obligatorio'='true'::jsonb)
  ), caminos(o,d) AS (SELECT o,d FROM aristas UNION SELECT c.o,a.d FROM caminos c JOIN aristas a ON a.o=c.d)
  SELECT EXISTS(SELECT 1 FROM caminos WHERE o=d) INTO ciclos;
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(NOT ciclos);
 END IF;
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(q,'publicacion')),'hex')=p->>'huella_sha256');
 RETURN q;
END $$;

-- Efectos puros de período. No persiste ni aplica incorporación CT V2.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_periodos(s jsonb, a jsonb, efecto text) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE ps jsonb:=s->'periodos_resultantes'; c jsonb:=s->'cese_efectivo'; p jsonb:=a->'periodo';
 n int:=jsonb_array_length(ps); i int; ef bigint:=vec_contratacion_temporal.seguimiento73_micro(a->'efectivo_en');
 rg bigint:=vec_contratacion_temporal.seguimiento73_micro(a->'registrada_en');
BEGIN
 CASE efecto
 WHEN 'ninguno' THEN NULL;
 WHEN 'abrir','ampliar' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(c IS NULL AND p IS NOT NULL AND ef=vec_contratacion_temporal.seguimiento73_micro(p->'desde'));
  IF efecto='abrir' THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(n=0);
  ELSE
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(n>0);
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(p->'desde')>=vec_contratacion_temporal.seguimiento73_micro(ps#>ARRAY[(n-1)::text,'intervalo','hasta']));
  END IF;
  ps:=ps||jsonb_build_array(jsonb_build_object('intervalo',p,'actuacion_ref',a->'actuacion_ref'));
 WHEN 'cerrar','rectificar_cese' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(n>0 AND ef>=rg);
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(ef>=vec_contratacion_temporal.seguimiento73_micro(ps#>'{0,intervalo,desde}'));
  IF efecto='cerrar' THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(c IS NULL);
  ELSE PERFORM vec_contratacion_temporal.seguimiento73_exigir(c->'actuacion_ref'=a->'rectifica_actuacion_ref'); END IF;
  c:=jsonb_build_object('efectivo_en',a->'efectivo_en','actuacion_ref',a->'actuacion_ref');
 WHEN 'rectificar_tramo' THEN
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(p IS NOT NULL AND ef=vec_contratacion_temporal.seguimiento73_micro(p->'desde'));
  SELECT ord::int-1 INTO i FROM jsonb_array_elements(ps) WITH ORDINALITY x(v,ord) WHERE v->'actuacion_ref'=a->'rectifica_actuacion_ref';
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(i IS NOT NULL);
  IF i>0 THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(ef>=vec_contratacion_temporal.seguimiento73_micro(ps#>ARRAY[(i-1)::text,'intervalo','hasta'])); END IF;
  IF i+1<n THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(p->'hasta')<=vec_contratacion_temporal.seguimiento73_micro(ps#>ARRAY[(i+1)::text,'intervalo','desde'])); END IF;
  IF i=0 AND c IS NOT NULL THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_micro(c->'efectivo_en')>=ef); END IF;
  ps:=jsonb_set(ps,ARRAY[i::text],jsonb_build_object('intervalo',p,'actuacion_ref',a->'actuacion_ref'));
 WHEN 'reabrir' THEN PERFORM vec_contratacion_temporal.seguimiento73_exigir(c IS NOT NULL); c:=NULL;
 ELSE RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='efecto seguimiento invalido';
 END CASE;
 RETURN jsonb_build_object('periodos_resultantes',ps)||CASE WHEN c IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('cese_efectivo',c) END;
END $$;

CREATE FUNCTION vec_contratacion_temporal.seguimiento73_rehidratar(p jsonb, e jsonb) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE def jsonb; raiz jsonb; s jsonb; a jsonb; t jsonb; docs jsonb; d jsonb; datos jsonb;
 indice jsonb:='{}'; transiciones jsonb:='{}'; periodos jsonb;
 anterior text; peticion text; refdef jsonb; numero int:=0; desde bigint; hasta bigint; rg bigint; actualizado bigint;
 canon bytea; entrada bytea; normal bytea;
BEGIN
 def:=vec_contratacion_temporal.seguimiento73_definicion(p);
 entrada:=vec_contratacion_temporal.seguimiento73_nodo(e,'estado');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(octet_length(entrada)<=8388608 AND (e->>'version')::numeric=jsonb_array_length(e->'actuaciones'));
 refdef:=jsonb_build_object('referencia',def->'referencia','version',def->'version','huella_sha256',def->'huella_sha256');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(e->'definicion'=refdef);
 desde:=vec_contratacion_temporal.seguimiento73_micro(def#>'{vigencia,desde}');
 hasta:=vec_contratacion_temporal.seguimiento73_micro(def#>'{vigencia,hasta}',true);
 actualizado:=vec_contratacion_temporal.seguimiento73_micro(e->'creado_en');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(actualizado>=desde AND (hasta=-62135596800000000 OR actualizado<hasta));
 raiz:=jsonb_build_object('referencia',e->'referencia','organizacion_ref',e->'organizacion_ref','expediente_ref',e->'expediente_ref',
  'relacion_ref',e->'relacion_ref','definicion',refdef,'estado_actual',def->'estado_inicial','periodo_previsto',e->'periodo_previsto','creado_en',e->'creado_en');
 anterior:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(raiz,'raiz')),'hex');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(anterior=e->>'huella_raiz_sha256');
 s:=raiz||jsonb_build_object('version',0,'actualizado_en',e->'creado_en','huella_raiz_sha256',anterior,'periodos_resultantes','[]'::jsonb,'actuaciones','[]'::jsonb);
 -- Índice único construido una vez. La posición física impide rectificar el
 -- futuro; la historia sólo se adjunta al final, sin clonarla en cada paso.
 PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value->>'actuacion_ref') FROM jsonb_array_elements(e->'actuaciones')));
 SELECT coalesce(jsonb_object_agg(v->>'actuacion_ref',jsonb_build_object('posicion',ord,'actor_ref',v->'actor_ref')),'{}'::jsonb)
 INTO indice FROM jsonb_array_elements(e->'actuaciones') WITH ORDINALITY x(v,ord);
 FOR t IN SELECT value FROM jsonb_array_elements(def->'transiciones') LOOP transiciones:=transiciones||jsonb_build_object(t->>'clave',t); END LOOP;
 FOR a IN SELECT value FROM jsonb_array_elements(e->'actuaciones') LOOP
  numero:=numero+1; t:=transiciones->(a->>'transicion_clave');
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(t IS NOT NULL
   AND (a->>'secuencia')::numeric=numero AND (a->>'version_seguimiento')::numeric=numero AND a->'definicion'=refdef
   AND a->>'huella_anterior_sha256'=anterior AND a->'clase'=t->'clase' AND a->'estado_origen'=t->'origen'
   AND a->'estado_destino'=t->'destino' AND s->'estado_actual'=t->'origen');
  rg:=vec_contratacion_temporal.seguimiento73_micro(a->'registrada_en');
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(rg>=actualizado AND rg>=desde AND (hasta=-62135596800000000 OR rg<hasta));
  docs:=vec_contratacion_temporal.seguimiento73_array(a->'documentos',32,0,true);
  PERFORM vec_contratacion_temporal.seguimiento73_exigir((SELECT count(*)=count(DISTINCT value->>'referencia') FROM jsonb_array_elements(docs)));
  SELECT coalesce(jsonb_agg(value ORDER BY (value->>'tipo_clave') COLLATE "C",(value->>'referencia') COLLATE "C"),'[]'::jsonb) INTO docs FROM jsonb_array_elements(docs);
  datos:=a-ARRAY['secuencia','version_seguimiento','definicion','clase','estado_origen','estado_destino','huella_peticion_sha256','huella_anterior_sha256','huella_actuacion_sha256'];
  datos:=jsonb_set(datos,'{documentos}',docs);
  peticion:=encode(sha256(vec_contratacion_temporal.seguimiento73_nodo(datos,'peticion')),'hex');
  canon:=vec_contratacion_temporal.seguimiento73_nodo(a,'actuacion');
  normal:=vec_contratacion_temporal.seguimiento73_nodo(jsonb_set(a,'{documentos}',docs),'actuacion');
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(peticion=a->>'huella_peticion_sha256' AND canon=normal AND encode(sha256(canon),'hex')=a->>'huella_actuacion_sha256');
  PERFORM vec_contratacion_temporal.seguimiento73_exigir(((t->'motivo_obligatorio'='false'::jsonb) OR coalesce(a->>'motivo_clave','')<>'')
   AND (coalesce(a->>'motivo_clave','')='' OR t->'motivos_permitidos' ? (a->>'motivo_clave'))
   AND (t->>'requiere_periodo')::boolean=(a ? 'periodo') AND (t ? 'calendario')=(a ? 'calendario'));
  FOR d IN SELECT value FROM jsonb_array_elements(docs) LOOP
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(EXISTS(SELECT 1 FROM jsonb_array_elements(t->'documentos') r WHERE r->'tipo_clave'=d->'tipo_clave'));
  END LOOP;
  FOR d IN SELECT value FROM jsonb_array_elements(t->'documentos') WHERE value->'obligatorio'='true'::jsonb LOOP
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(EXISTS(SELECT 1 FROM jsonb_array_elements(docs) r WHERE r->'tipo_clave'=d->'tipo_clave'));
  END LOOP;
  IF t ? 'calendario' THEN
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(t#>'{calendario,ambitos_permitidos}' ? (a#>>'{calendario,ambito_territorial_clave}')
    AND t#>'{calendario,resultados_permitidos}' ? (a#>>'{calendario,resultado_clave}') AND vec_contratacion_temporal.seguimiento73_micro(a#>'{calendario,calculado_en}')<=rg);
  END IF;
  IF t->>'clase'='rectificacion' THEN
   d:=indice->coalesce(a->>'rectifica_actuacion_ref','');
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(d IS NOT NULL AND (d->>'posicion')::int<numero AND (t->'exige_actor_distinto'='false'::jsonb OR d->'actor_ref'<>a->'actor_ref'));
  ELSE PERFORM vec_contratacion_temporal.seguimiento73_exigir(coalesce(a->>'rectifica_actuacion_ref','')=''); END IF;
  periodos:=vec_contratacion_temporal.seguimiento73_periodos(s,a,t->>'efecto_periodo');
  actualizado:=rg; anterior:=a->>'huella_actuacion_sha256';
  s:=(s-'cese_efectivo')||periodos||jsonb_build_object('version',numero,'estado_actual',t->'destino','actualizado_en',a->'registrada_en');
 END LOOP;
 s:=jsonb_set(s,'{actuaciones}',e->'actuaciones');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(vec_contratacion_temporal.seguimiento73_nodo(s,'estado')=entrada);
 RETURN s;
END $$;

-- JSONB no conserva duplicados/trailing: deben rechazarse antes del cast.
CREATE FUNCTION vec_contratacion_temporal.seguimiento73_elementos(j jsonb, profundidad int DEFAULT 0) RETURNS int
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE n int:=1; v jsonb;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(j IS NOT NULL AND profundidad<=32);
 IF jsonb_typeof(j)='object' THEN
  FOR v IN SELECT value FROM jsonb_each(j) LOOP
   n:=n+vec_contratacion_temporal.seguimiento73_elementos(v,profundidad+1);
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(n<=400000);
  END LOOP;
 ELSIF jsonb_typeof(j)='array' THEN
  FOR v IN SELECT value FROM jsonb_array_elements(j) LOOP
   n:=n+vec_contratacion_temporal.seguimiento73_elementos(v,profundidad+1);
   PERFORM vec_contratacion_temporal.seguimiento73_exigir(n<=400000);
  END LOOP;
 END IF;
 RETURN n;
END $$;

CREATE FUNCTION vec_contratacion_temporal.estado_seguimiento_canonico_v1(publicacion jsonb, estado jsonb) RETURNS bytea
LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$
DECLARE s jsonb; b bytea;
BEGIN
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(octet_length(publicacion::text)<=131072 AND octet_length(estado::text)<=8388608);
 PERFORM vec_contratacion_temporal.seguimiento73_elementos(publicacion);
 PERFORM vec_contratacion_temporal.seguimiento73_elementos(estado);
 s:=vec_contratacion_temporal.seguimiento73_rehidratar(publicacion,estado);
 b:=vec_contratacion_temporal.seguimiento73_nodo(s,'estado');
 PERFORM vec_contratacion_temporal.seguimiento73_exigir(octet_length(b)<=8388608);
 RETURN b;
END $$;

-- Owner-only, invoker y puros; ningún grant de runtime ni tablas de negocio.
DO $acl$
DECLARE f record; g record;
BEGIN
 FOR f IN SELECT p.oid,p.proowner FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE 'seguimiento73\_%' ESCAPE '\' OR p.proname='estado_seguimiento_canonico_v1') LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f.oid::regprocedure);
  FOR g IN SELECT DISTINCT grantee FROM aclexplode((SELECT proacl FROM pg_proc WHERE oid=f.oid)) WHERE grantee<>0 AND grantee<>f.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f.oid::regprocedure,pg_get_userbyid(g.grantee));
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
