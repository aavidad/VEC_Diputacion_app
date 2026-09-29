#!/usr/bin/env bash
# Usuarios 000008 + ContextoActor 000010 + Bolsa 000059 sobre un clon de la
# principal ya migrado con la lista de B59. Todo ocurre en UNA transacción
# que termina en ROLLBACK: no deja roles, datos ni funciones cambiadas.
#
# La fachada AD3-109 se sustituye, sólo dentro de la transacción, por un doble
# que devuelve un consumo sintético: esta prueba no acredita COSE ni el núcleo
# V3 (eso lo cubre el recorrido real). Sí prueba el material, la huella del
# recurso calculada como en Go, la elección del correo, la separación de
# superficies, la RLS, las ACL y el registro de la fuente en Bolsa.
#
# Requiere un clon RECIÉN migrado: siembra correos para la única candidata
# con persona vigente y se niega a seguir si ya los tiene (por ejemplo, tras
# el recorrido de Mailpit sobre el mismo clon).
#
# Uso: correo_avisos_clon.sh <contenedor_del_clon>
set -Eeuo pipefail
contenedor=${1:?contenedor del clon}
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }

candidato=$(psql_pg -At -c "SELECT v.referencia FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref,version) WHERE v.tipo='candidato' AND v.estado='activo' AND statement_timestamp()<v.vigente_hasta AND vec_contexto_actor_v1.persona_candidato_avisos_v1(v.referencia) IS NOT NULL LIMIT 1")
[ -n "$candidato" ] || { echo "el clon no tiene ninguna persona candidata vigente"; exit 2; }

# Material y huella construidos como en Go (canonico.RecursoCorreoAvisos).
eval "$(python3 - "$candidato" <<'PY'
import hashlib, json, sys, shlex
cand = sys.argv[1]
def material(llamamiento, superficie="interna_corporativa"):
    m = {"esquema": "vec.usuarios.correo-avisos-llamamiento.v1", "superficie": superficie,
         "finalidad_ref": "gestion_llamamientos_bolsa", "bolsa_ref": "bolsa:prueba:b59",
         "unidad_ref": "unidad:rrhh", "ambito_ref": "ambito:bolsa", "llamamiento_ref": llamamiento,
         "candidato_ref": cand}
    return json.dumps(m, separators=(",", ":"), ensure_ascii=False)
def huella(mat):
    m = json.loads(mat)
    canon = '{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{"material_sha256":"%s"}}' % (
        json.dumps(m["ambito_ref"]), json.dumps(m["unidad_ref"]), hashlib.sha256(mat.encode()).hexdigest())
    return hashlib.sha256(canon.encode()).hexdigest()
def piezas(mat, n, h=None, audiencia="vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1", superficie="interna_corporativa"):
    h = h or huella(mat)
    c = {"audiencia_consumo": audiencia, "operacion": "llamamiento.emitir.v1", "efecto_ref": "bolsa:prueba:b59", "huella_efecto_sha256": h}
    d = {"decision_ref": "dec_b59_%d" % n, "accion": "llamamiento.emitir.v1", "modulo_id": "bolsa", "tipo_recurso": "bolsa_constituida",
         "finalidad": "gestion_llamamientos_bolsa", "recurso_ref": "bolsa:prueba:b59", "concedida": True,
         "contexto_recurso_huella_sha256": h, "vinculo_autenticacion_actor": {"superficie": superficie}}
    return json.dumps(c), json.dumps(d)
ll = "llamamiento:" + "b5" * 32
m1 = material(ll)
c1, d1 = piezas(m1, 1)
c2, d2 = piezas(m1, 2, h="0" * 64)
mext = material(ll, "externa_personal")
c3, d3 = piezas(mext, 3, superficie="externa_personal")
c4, d4 = piezas(m1, 4, audiencia="vec_usuarios.correos.consultar.interna_corporativa.v1")
for k, v in dict(M1=m1, C1=c1, D1=d1, C2=c2, D2=d2, MEXT=mext, C3=c3, D3=d3, C4=c4, D4=d4).items():
    print("%s=%s" % (k, shlex.quote(v)))
PY
)"

psql_pg -v cand="$candidato" -v m1="$M1" -v c1="$C1" -v d1="$D1" -v c2="$C2" -v d2="$D2" \
 -v mext="$MEXT" -v c3="$C3" -v d3="$D3" -v c4="$C4" -v d4="$D4" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SELECT set_config('prueba.cand',:'cand',true), set_config('prueba.m1',:'m1',true), set_config('prueba.c1',:'c1',true),
 set_config('prueba.d1',:'d1',true), set_config('prueba.c2',:'c2',true), set_config('prueba.d2',:'d2',true),
 set_config('prueba.mext',:'mext',true), set_config('prueba.c3',:'c3',true), set_config('prueba.d3',:'d3',true),
 set_config('prueba.c4',:'c4',true), set_config('prueba.d4',:'d4',true) \gset
CREATE ROLE vec_b59_prueba_interna LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_ejecutor_interno TO vec_b59_prueba_interna WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_b59_prueba_externa LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_ejecutor_externo TO vec_b59_prueba_externa WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;

-- Doble de la fachada AD3-109, sólo dentro de esta transacción.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT convert_from(p_decision,'UTF8')::jsonb->>'decision_ref', convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
  convert_from(p_capacidad,'UTF8')::jsonb->>'huella_efecto_sha256', encode(sha256(p_decision),'hex'), 'aud_b59_prueba', clock_timestamp(), true
$f$;

-- Persona de la candidata: un correo activo verificado desde el área externa
-- y otro verificado desde la interna (no activo).
DO $datos$
DECLARE p text:=vec_contexto_actor_v1.persona_candidato_avisos_v1(current_setting('prueba.cand'));
 ahora timestamptz:=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF p IS NULL THEN RAISE EXCEPTION 'sin persona'; END IF;
 IF EXISTS(SELECT 1 FROM vec_usuarios.correos_conjunto WHERE persona_ref=p) THEN RAISE EXCEPTION 'la persona ya tiene correos en el clon'; END IF;
 INSERT INTO vec_usuarios.correos_conjunto VALUES(p,3,'clave:igualdad:b59',ahora);
 INSERT INTO vec_usuarios.correos_direccion(persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en,verificado_en)
 VALUES(p,'correo:'||repeat('e',32),1,'clave:sobre:b59','clave:igualdad:b59',decode(repeat('01',12),'hex'),decode(repeat('02',24),'hex'),decode(repeat('03',32),'hex'),'verificado',true,ahora,ahora),
       (p,'correo:'||repeat('f',32),1,'clave:sobre:b59','clave:igualdad:b59',decode(repeat('01',12),'hex'),decode(repeat('04',24),'hex'),decode(repeat('05',32),'hex'),'verificado',false,ahora,ahora);
 INSERT INTO vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref,huella_codigo,clave_ref,vence_en,estado,intentos,creado_en)
 VALUES(p,'correo:'||repeat('e',32),'desafio:'||repeat('e',32),decode(repeat('06',32),'hex'),'clave:codigo:b59',ahora+interval '1 hour','usado',0,ahora),
       (p,'correo:'||repeat('f',32),'desafio:'||repeat('f',32),decode(repeat('07',32),'hex'),'clave:codigo:b59',ahora+interval '1 hour','usado',0,ahora);
 INSERT INTO vec_usuarios.correos_envio(envio_ref,persona_ref,correo_ref,superficie,tipo,desafio_ref,recibo_ref,reserva_sha256,estado,creado_en,resuelto_en)
 VALUES('correo_envio:'||repeat('e',32),p,'correo:'||repeat('e',32),'externa_personal','verificacion','desafio:'||repeat('e',32),'correo_recibo:'||repeat('e',32),repeat('e',64),'aceptado',ahora,ahora),
       ('correo_envio:'||repeat('f',32),p,'correo:'||repeat('f',32),'interna_corporativa','verificacion','desafio:'||repeat('f',32),'correo_recibo:'||repeat('f',32),repeat('f',64),'aceptado',ahora,ahora);
END $datos$;

SET SESSION AUTHORIZATION vec_b59_prueba_interna;
-- 1) Correo activo verificado desde el área externa: se entrega su sobre.
DO $p1$ DECLARE r jsonb; BEGIN
 r:=vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.m1'),convert_to(current_setting('prueba.c1'),'UTF8'),convert_to(current_setting('prueba.d1'),'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r->>'encontrado'<>'true' OR r->>'correo_ref'<>'correo:'||repeat('e',32) OR r->>'auditoria_ref'<>'aud_b59_prueba'
    OR r->'sobre'->>'cifrado_hex'<>repeat('02',24) OR (SELECT count(*) FROM jsonb_object_keys(r))<>5
 THEN RAISE EXCEPTION 'caso 1: %', r; END IF;
END $p1$;
-- 2) Huella del recurso que no casa con el material: 42501.
DO $p2$ BEGIN
 PERFORM vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.m1'),convert_to(current_setting('prueba.c2'),'UTF8'),convert_to(current_setting('prueba.d2'),'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'caso 2 admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $p2$;
-- 3) Material de superficie externa: 42501.
DO $p3$ BEGIN
 PERFORM vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.mext'),convert_to(current_setting('prueba.c3'),'UTF8'),convert_to(current_setting('prueba.d3'),'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'caso 3 admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $p3$;
-- 4) Capacidad de otra audiencia («Mis correos» propio): 42501.
DO $p4$ BEGIN
 PERFORM vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.m1'),convert_to(current_setting('prueba.c4'),'UTF8'),convert_to(current_setting('prueba.d4'),'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'caso 4 admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $p4$;
-- 5) El ejecutor no lee tablas ni la fachada de identidad.
DO $p5$ BEGIN
 BEGIN PERFORM 1 FROM vec_usuarios.correos_direccion; RAISE EXCEPTION 'caso 5a admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_contexto_actor_v1.persona_candidato_avisos_v1(current_setting('prueba.cand')); RAISE EXCEPTION 'caso 5b admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada('\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00'); RAISE EXCEPTION 'caso 5c admitido'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $p5$;
RESET SESSION AUTHORIZATION;
-- El contexto de lectura se retira antes de devolver.
DO $p1b$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_usuarios.correos_contexto WHERE modo='avisos') THEN RAISE EXCEPTION 'caso 1: contexto residual'; END IF;
END $p1b$;

-- 6) Si el activo pasa a ser el verificado desde la interna, no se entrega nada.
UPDATE vec_usuarios.correos_direccion SET activo=false WHERE correo_ref='correo:'||repeat('e',32);
UPDATE vec_usuarios.correos_direccion SET activo=true WHERE correo_ref='correo:'||repeat('f',32);
SET SESSION AUTHORIZATION vec_b59_prueba_interna;
DO $p6$ DECLARE r jsonb; BEGIN
 r:=vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.m1'),convert_to(current_setting('prueba.c1'),'UTF8'),convert_to(replace(current_setting('prueba.d1'),'dec_b59_1','dec_b59_6'),'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF r IS DISTINCT FROM '{"encontrado":false,"auditoria_ref":"aud_b59_prueba"}'::jsonb THEN RAISE EXCEPTION 'caso 6: %', r; END IF;
END $p6$;
RESET SESSION AUTHORIZATION;

-- 7) El ejecutor externo no puede ejecutar la lectura.
SET SESSION AUTHORIZATION vec_b59_prueba_externa;
DO $p7$ BEGIN
 PERFORM vec_usuarios.correo_activo_avisos_llamamiento_v1(current_setting('prueba.m1'),'\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 RAISE EXCEPTION 'caso 7 admitido';
EXCEPTION WHEN insufficient_privilege THEN NULL; END $p7$;
RESET SESSION AUTHORIZATION;

-- 8) Candidato desconocido: sin persona, sin correo y sin error.
DO $p8$ BEGIN
 IF vec_contexto_actor_v1.persona_candidato_avisos_v1('can_'||repeat('Q',30)) IS NOT NULL
    OR vec_contexto_actor_v1.persona_candidato_avisos_v1('per_'||repeat('Q',30)) IS NOT NULL
 THEN RAISE EXCEPTION 'caso 8'; END IF;
END $p8$;

-- 9) Bolsa 000059: registrar contactos con fuente en una emisión sintética.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
VALUES('llamamiento:'||repeat('b5',32),'recibo:llamamiento:'||repeat('b5',32),'bolsa:prueba:b59','per_'||repeat('A',24),'clave-b59-prueba',
 '["participacion:b59:1","participacion:b59:2"]','{"plantilla_version":"bolsa-llamamiento-v1"}',repeat('a',64),sha256('\x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20'::bytea),'emision_reservada',date_trunc('microseconds',clock_timestamp()),'dec_b59_emision');
RESET ROLE;
DO $p9$ DECLARE r jsonb; token bytea:='\x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20';
 rec1 text:='recibo:contacto:'||encode(sha256(convert_to('bolsa:prueba:b59'||chr(31)||'clave-b59-prueba'||chr(31)||'participacion:b59:1','UTF8')),'hex');
 rec2 text:='recibo:contacto:'||encode(sha256(convert_to('bolsa:prueba:b59'||chr(31)||'clave-b59-prueba'||chr(31)||'participacion:b59:2','UTF8')),'hex');
 bueno jsonb; BEGIN
 bueno:=jsonb_build_array(
  jsonb_build_object('participacion_ref','participacion:b59:1','resultado','enviado','recibo_ref',rec1,'fuente_correo',jsonb_build_object('fuente','mis_correos','motivo','correo_activo','correo_ref','correo:'||repeat('e',32))),
  jsonb_build_object('participacion_ref','participacion:b59:2','resultado','enviado','recibo_ref',rec2,'fuente_correo',jsonb_build_object('fuente','alta_bolsa','motivo','sin_correo_activo')));
 -- Formas inválidas: fuente con dirección, alta con referencia, clave de más.
 BEGIN PERFORM vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2('bolsa:prueba:b59','clave-b59-prueba','per_'||repeat('A',24),token,
   jsonb_set(bueno,'{0,fuente_correo,correo_ref}','"x@example.org"')); RAISE EXCEPTION 'caso 9a admitido'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2('bolsa:prueba:b59','clave-b59-prueba','per_'||repeat('A',24),token,
   jsonb_set(bueno,'{1,fuente_correo,correo_ref}','"correo:0123456789abcdef0123456789abcdef"')); RAISE EXCEPTION 'caso 9b admitido'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2('bolsa:prueba:b59','clave-b59-prueba','per_'||repeat('A',24),token,
   jsonb_set(bueno,'{0,direccion}','"x@example.org"')); RAISE EXCEPTION 'caso 9c admitido'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 r:=vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2('bolsa:prueba:b59','clave-b59-prueba','per_'||repeat('A',24),token,bueno);
 IF r->>'estado'<>'emitido_pendiente_respuesta' OR r#>>ARRAY['fuentes_correo',rec1,'fuente']<>'mis_correos'
    OR r#>>ARRAY['fuentes_correo',rec2,'motivo']<>'sin_correo_activo' OR r#>ARRAY['fuentes_correo',rec2] ? 'correo_ref'
 THEN RAISE EXCEPTION 'caso 9: %', r; END IF;
 -- Repetición: no duplica ni cambia la fuente.
 r:=vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2('bolsa:prueba:b59','clave-b59-prueba','per_'||repeat('A',24),token,
   jsonb_set(bueno,'{1,fuente_correo}','{"fuente":"alta_bolsa","motivo":"mis_correos_no_disponible"}'));
 IF r#>>ARRAY['fuentes_correo',rec2,'motivo']<>'sin_correo_activo' OR (SELECT count(*) FROM vec_bolsa_llamamientos.contacto_fuente_correo WHERE recibo_ref IN (rec1,rec2))<>2
 THEN RAISE EXCEPTION 'caso 9 repetición: %', r; END IF;
 IF vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1('bolsa:prueba:b59','clave-b59-prueba') IS DISTINCT FROM r->'fuentes_correo'
 THEN RAISE EXCEPTION 'caso 9 recuperación'; END IF;
 BEGIN UPDATE vec_bolsa_llamamientos.contacto_fuente_correo SET motivo='sin_persona_vinculada' WHERE recibo_ref=rec2; RAISE EXCEPTION 'caso 9 mutación admitida';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'caso 9%' THEN RAISE; END IF; END;
END $p9$;
-- 9b) El candidato sólo se resuelve para una participación de un llamamiento
-- reservado de esa bolsa.
DO $p9b$ DECLARE p text; c text; b text; BEGIN
 SELECT v.participacion_ref, v.candidato_ref, x.bolsa_ref INTO p, c, b FROM vec_bolsa_llamamientos.vinculo_candidato v
  JOIN vec_bolsa_llamamientos.constitucion x ON x.acta_ref=v.acta_ref LIMIT 1;
 IF vec_bolsa_llamamientos.candidato_participacion_avisos_v1(b,'llamamiento:'||repeat('b5',32),p) IS NOT NULL
 THEN RAISE EXCEPTION 'caso 9b: participación ajena al llamamiento resuelta'; END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES('llamamiento:'||repeat('c6',32),'recibo:llamamiento:'||repeat('c6',32),b,'per_'||repeat('A',24),'clave-b59-9b',jsonb_build_array(p),'{}',repeat('a',64),sha256('\x00'::bytea),'emision_reservada',clock_timestamp(),'dec_b59_9b');
 RESET ROLE;
 IF vec_bolsa_llamamientos.candidato_participacion_avisos_v1(b,'llamamiento:'||repeat('c6',32),p) IS DISTINCT FROM c
    OR vec_bolsa_llamamientos.candidato_participacion_avisos_v1('bolsa:otra','llamamiento:'||repeat('c6',32),p) IS NOT NULL
 THEN RAISE EXCEPTION 'caso 9b: resolución del candidato'; END IF;
END $p9b$;
-- 10) ACL de Bolsa: el ejecutor tiene las tres fachadas y ninguna tabla nueva.
DO $p10$ BEGIN
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.contacto_fuente_correo','SELECT')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.registrar_contactos_llamamiento_v2(text,text,text,bytea,jsonb)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.candidato_participacion_avisos_v1(text,text,text)','EXECUTE')
    OR has_function_privilege('public','vec_bolsa_llamamientos.fuentes_correo_llamamiento_v1(text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'caso 10'; END IF;
END $p10$;
ROLLBACK;
SQL
echo "PRUEBA-B59-OK"
