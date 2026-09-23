#!/usr/bin/env bash
# Preparación privada de Mi bolsa. Ejecutar como usuario del servicio en desarrollo.
set -Eeuo pipefail
umask 077

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
for orden in podman openssl python3 sha256sum flock stat mktemp realpath git date; do
  command -v "$orden" >/dev/null || fallar "falta $orden"
done

material=${VEC_DEVELOPMENT_MATERIAL_DIR:-${XDG_STATE_HOME:-${HOME:?HOME no definido}/.local/state}/vec-diputacion/desarrollo}
[[ $material == /* && -d $material && ! -L $material ]] || fallar 'directorio privado inválido'
material=$(realpath -- "$material")
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
case "$material" in
  "$repo"|"$repo"/*) fallar 'el material privado no puede estar dentro del repositorio' ;;
esac
[[ $(stat -c %u -- "$material") == "$(id -u)" ]] || fallar 'el material no pertenece al usuario del servicio'
[[ $(stat -c %a -- "$material") == 700 ]] || fallar 'el material debe tener modo 0700'

postgres=${VEC_PRINCIPAL_POSTGRES_CONTAINER:-vec-postgresql-20260906}
aplicacion=${VEC_PRINCIPAL_APP_CONTAINER:-vec-aplicacion-incorporacion-20260910}
base=${VEC_CANDIDATE_DATABASE:-postgres}
usuario=${VEC_CANDIDATE_DATABASE_USER:-postgres}
url=${VEC_CANDIDATE_BASE_URL:-https://localhost:8443}
bolsa=${VEC_CANDIDATE_BOLSA_REF:-}
[[ $postgres =~ ^[a-zA-Z0-9_.-]+$ && $aplicacion =~ ^[a-zA-Z0-9_.-]+$ ]] || fallar 'nombre de contenedor inválido'
[[ $base =~ ^[a-zA-Z0-9_-]+$ && $usuario =~ ^[a-zA-Z0-9_-]+$ ]] || fallar 'base o usuario inválidos'
[[ $url == https://* ]] || fallar 'la URL debe ser HTTPS'

for ruta in ca ca/ca.crt ca/ca.key mtls identidad; do
  [[ ! -L $material/$ruta ]] || fallar "enlace simbólico rechazado: $ruta"
done
for ruta in ca/ca.crt ca/ca.key; do
  [[ -f $material/$ruta && $(stat -c %u -- "$material/$ruta") == "$(id -u)" ]] || fallar "CA inaccesible: $ruta"
  [[ $(stat -c %a -- "$material/$ruta") == 600 ]] || fallar "permisos inseguros de la CA: $ruta"
done
install -d -m 0700 -- "$material/mtls" "$material/identidad"
exec 9>"$material/.preparar-candidato.lock"
flock -x 9

psql_contenedor() {
  podman exec -i "$postgres" psql -X -q -v ON_ERROR_STOP=1 -U "$usuario" -d "$base" "$@"
}

cert=$material/mtls/candidato.crt
clave=$material/mtls/candidato.key
p12=$material/mtls/candidato.p12
password=$material/mtls/candidato.p12.password
identidad=$material/identidad/candidato.json
manifiesto=$material/identidad/bolsa-candidato.json
for ruta in "$cert" "$clave" "$p12" "$password" "$identidad" "$manifiesto"; do
  [[ ! -L $ruta ]] || fallar 'hay un enlace simbólico en el material candidato'
done

if [[ -e $cert || -e $clave || -e $p12 || -e $password ]]; then
  [[ -f $cert && -f $clave && -f $p12 && -f $password ]] || fallar 'material candidato parcial; revisar manualmente sin sobrescribir'
else
  temporal=$(mktemp -d -- "$material/mtls/.candidato.XXXXXXXX")
  trap 'rm -rf -- "${temporal:-}"' EXIT
  openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$temporal/candidato.key" 2>/dev/null
  openssl req -new -sha256 -key "$temporal/candidato.key" \
    -subj '/CN=candidato-bolsa-desarrollo/O=VEC Desarrollo/OU=NO AUTORITATIVO' \
    -out "$temporal/candidato.csr" 2>/dev/null
  cat >"$temporal/candidato.ext" <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
subjectAltName=URI:urn:vec:desarrollo:candidato-bolsa
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EXT
  serial=0x$(openssl rand -hex 16)
  openssl x509 -req -sha256 -days 397 -in "$temporal/candidato.csr" \
    -CA "$material/ca/ca.crt" -CAkey "$material/ca/ca.key" -set_serial "$serial" \
    -extfile "$temporal/candidato.ext" -out "$temporal/candidato.crt" 2>/dev/null
  openssl rand -hex 32 >"$temporal/candidato.p12.password"
  openssl pkcs12 -export -out "$temporal/candidato.p12" \
    -inkey "$temporal/candidato.key" -in "$temporal/candidato.crt" \
    -certfile "$material/ca/ca.crt" -name 'VEC desarrollo - candidato Bolsa' \
    -passout "file:$temporal/candidato.p12.password" 2>/dev/null
  for nombre in candidato.crt candidato.key candidato.p12 candidato.p12.password; do
    install -m 0600 -- "$temporal/$nombre" "$material/mtls/$nombre"
  done
  rm -rf -- "$temporal"
  temporal=
  trap - EXIT
fi

for ruta in "$cert" "$clave" "$p12" "$password"; do
  [[ $(stat -c %a -- "$ruta") == 600 && $(stat -c %u -- "$ruta") == "$(id -u)" ]] || fallar 'permisos del material candidato inválidos'
done
openssl verify -purpose sslclient -CAfile "$material/ca/ca.crt" "$cert" >/dev/null || fallar 'certificado ajeno a la CA o sin uso cliente'
openssl x509 -in "$cert" -checkend 0 -noout >/dev/null || fallar 'certificado candidato caducado'
openssl x509 -in "$cert" -noout -subject -nameopt RFC2253 | grep -Eq '(^|,)CN=candidato-bolsa-desarrollo($|,)' || fallar 'CN candidato incorrecto'
openssl x509 -in "$cert" -noout -ext subjectAltName | grep -Fq 'URI:urn:vec:desarrollo:candidato-bolsa' || fallar 'SAN candidato incorrecto'
[[ $(openssl x509 -in "$cert" -pubkey -noout | openssl pkey -pubin -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) == \
   $(openssl pkey -in "$clave" -pubout -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) ]] || fallar 'certificado y clave no coinciden'
openssl pkcs12 -in "$p12" -passin "file:$password" -noout 2>/dev/null || fallar 'PKCS#12 inválido'
huella=$(openssl x509 -in "$cert" -outform DER | sha256sum | cut -d' ' -f1)
[[ $(openssl pkcs12 -in "$p12" -passin "file:$password" -clcerts -nokeys 2>/dev/null |
  openssl x509 -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) == "$huella" ]] || fallar 'PKCS#12 no corresponde al certificado candidato'
[[ $(openssl pkcs12 -in "$p12" -passin "file:$password" -nocerts -nodes 2>/dev/null |
  openssl pkey -pubout -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) == \
  $(openssl pkey -in "$clave" -pubout -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) ]] || fallar 'PKCS#12 no corresponde a la clave candidata'
[[ $(openssl pkcs12 -in "$p12" -passin "file:$password" -cacerts -nokeys 2>/dev/null |
  openssl x509 -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1) == \
  $(openssl x509 -in "$material/ca/ca.crt" -outform DER | sha256sum | cut -d' ' -f1) ]] || fallar 'PKCS#12 no contiene la CA de desarrollo'
hasta=$(date -u -d "$(openssl x509 -in "$cert" -enddate -noout | cut -d= -f2)" +%Y-%m-%dT%H:%M:%SZ)
for otro in "$material/mtls/cliente.crt" "$material/mtls/intervencion.crt"; do
  if [[ -f $otro ]]; then
    [[ $(openssl x509 -in "$otro" -outform DER | sha256sum | cut -d' ' -f1) != "$huella" ]] || fallar 'el candidato comparte certificado con un rol interno'
  fi
done

# La intención privada conserva la elección entre COMMIT y publicación JSON.
# Se retira solo después de confirmar la proyección y ambos manifiestos.
intencion=$material/identidad/.bolsa-candidato-intencion.json
[[ ! -L $intencion ]] || fallar 'intención privada enlazada'
if [[ -e $intencion ]]; then
  [[ -f $intencion && $(stat -c %a -- "$intencion") == 600 ]] || fallar 'intención privada insegura'
  fila=$(python3 - "$intencion" "$huella" <<'PY'
import json, re, sys
with open(sys.argv[1], encoding='utf-8') as archivo:
    datos = json.load(archivo)
if datos.get('certificate_sha256') != sys.argv[2] or datos.get('version') != 1:
    raise SystemExit('intención privada no corresponde al certificado')
for clave in ('candidato_ref', 'participacion_ref', 'bolsa_ref'):
    valor = datos.get(clave)
    if not isinstance(valor, str) or not valor or '|' in valor or '\n' in valor or '\r' in valor:
        raise SystemExit('intención privada inválida')
print('|'.join(datos[clave] for clave in ('candidato_ref', 'participacion_ref', 'bolsa_ref')))
PY
  )
else
  # Sin intención: primera participación por bolsa, orden y referencia.
  if [[ -f $manifiesto ]]; then
    bolsa=$(python3 - "$manifiesto" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as archivo:
    print(json.load(archivo)['candidato_ref'])
PY
    )
    seleccion="AND vc.candidato_ref = :'seleccion'"
    vigencia=''
  else
    seleccion="AND (:'seleccion' = '' OR c.bolsa_ref = :'seleccion')"
    vigencia="AND b.estado = 'vigente' AND b.vigente_desde <= clock_timestamp()
      AND (b.vigente_hasta IS NULL OR clock_timestamp() < b.vigente_hasta)
      AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida posterior
                      WHERE posterior.bolsa_ref = b.bolsa_ref AND posterior.version > b.version)"
  fi
  fila=$(psql_contenedor -At -F '|' -v seleccion="$bolsa" <<SQL
  SET ROLE vec_bolsa_llamamientos_propietario;
  SELECT vc.candidato_ref, vc.participacion_ref, c.bolsa_ref
  FROM vec_bolsa_llamamientos.vinculo_candidato vc
  JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref = vc.acta_ref
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
    ON e.instantanea_ref = vc.instantanea_ref AND e.version_instantanea = vc.version_instantanea
   AND e.participacion_ref = vc.participacion_ref
  JOIN vec_bolsa_llamamientos.bolsa_constituida b
    ON b.bolsa_ref = c.bolsa_ref AND b.version = c.version_bolsa
   AND b.huella_bolsa_sha256 = c.huella_bolsa_sha256
  WHERE true $vigencia $seleccion
  ORDER BY c.bolsa_ref, e.orden, vc.participacion_ref LIMIT 1;
SQL
  )
fi
[[ $fila == *'|'*'|'* && $fila != *$'\n'* ]] || fallar 'no hay una participación importada inequívoca'
candidato=${fila%%|*}
resto=${fila#*|}
participacion=${resto%%|*}
bolsa_ref=${resto#*|}
[[ $candidato =~ ^can_[A-Za-z0-9_-]{22,128}$ && -n $participacion && -n $bolsa_ref ]] || fallar 'referencia importada inválida'
[[ -z ${VEC_CANDIDATE_BOLSA_REF:-} || ! -e $intencion || $bolsa_ref == "$VEC_CANDIDATE_BOLSA_REF" ]] || fallar 'intención previa de otra bolsa'

if [[ ! -e $intencion ]]; then
  temporal_intencion=$(mktemp -- "$material/identidad/.candidato-intencion.XXXXXXXX")
  trap 'rm -f -- "${temporal_intencion:-}"' EXIT
  python3 - "$temporal_intencion" "$huella" "$candidato" "$participacion" "$bolsa_ref" <<'PY'
import json, sys
with open(sys.argv[1], 'w', encoding='utf-8') as archivo:
    json.dump(dict(version=1, certificate_sha256=sys.argv[2], candidato_ref=sys.argv[3],
                   participacion_ref=sys.argv[4], bolsa_ref=sys.argv[5]), archivo, separators=(',', ':'))
    archivo.write('\n')
PY
  mv -nT -- "$temporal_intencion" "$intencion"
  temporal_intencion=
  trap - EXIT
fi

temporal=$(mktemp -d -- "$material/.proyeccion-candidato.XXXXXXXX")
trap 'rm -rf -- "${temporal:-}"' EXIT
python3 - "$temporal" "$material" "$huella" "$candidato" "$participacion" "$bolsa_ref" "$hasta" <<'PY'
import hashlib, json, pathlib, re, sys

destino, material, huella, candidato, participacion, bolsa_ref, hasta = sys.argv[1:]
destino, material = pathlib.Path(destino), pathlib.Path(material)
if not re.fullmatch(r'[0-9a-f]{64}', huella) or not re.fullmatch(r'can_[A-Za-z0-9_-]{22,128}', candidato):
    raise SystemExit('referencia inválida')
if not participacion or len(participacion.encode()) > 512 or any(ord(c) < 32 for c in participacion):
    raise SystemExit('participación inválida')
if not bolsa_ref or len(bolsa_ref.encode()) > 512 or any(ord(c) < 32 for c in bolsa_ref):
    raise SystemExit('bolsa inválida')

def hash_ref(prefijo, material):
    return prefijo + hashlib.sha256(b'vec.ct.alta.desarrollo.v1\0' + material).hexdigest()[:32]

subject = hash_ref('per_', b'candidato-bolsa\0' + huella.encode())
base = subject.encode() + b'\0' + huella.encode()
cuenta = hash_ref('cta_', base + b'\0cuenta')
persona = hash_ref('per_', base + b'\0persona')
perfil = hash_ref('prf_', base + b'\0perfil')
contexto = hash_ref('vca_', base + b'\0contexto')
vinculo = hash_ref('vin_', base + b'\0candidato')
procedencia = hash_ref('prc_', base + b'\0procedencia')
huella_procedencia = hashlib.sha256(b'vec.mi-bolsa.proyeccion.v1\0' + base + b'\0' + candidato.encode()).hexdigest()
identidad = dict(version=1, autoridad='no_autoritativo', certificate_sha256=huella,
                 subject=subject, display_name='Candidato sintético', roles=['candidato_bolsa'])
manifiesto = dict(version=1, autoridad='no_autoritativo', certificado='mtls/candidato.crt',
                  identidad='identidad/candidato.json', sujeto=subject, cuenta_ref=cuenta,
                  persona_ref=persona, perfil_ref=perfil, candidato_ref=candidato)
for nombre, valor in [('candidato.json', identidad), ('bolsa-candidato.json', manifiesto)]:
    (destino / nombre).write_text(json.dumps(valor, ensure_ascii=False, separators=(',', ':')) + '\n')

def literal(s):
    return "'" + s.replace("'", "''") + "'"

p, h, authority = map(literal, [procedencia, huella_procedencia, 'autoridad_maestra_acreditada'])
c, pe, pr, vc, vr, can = map(literal, [cuenta, persona, perfil, contexto, vinculo, candidato])
part, bolsa_sql = map(literal, [participacion, bolsa_ref])
fin = literal(hasta)
sql = f'''BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended({pe}, 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec.mi-bolsa.candidato:' || {can}, 0));
-- Si la proyección ya existe, el DO final comprueba su coincidencia íntegra.
-- La recuperación no depende de que la bolsa siga vigente días después.
SELECT CASE WHEN EXISTS (
  SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
  JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
  WHERE a.vinculo_ref={vr} AND a.version=1 AND v.persona_ref={pe}
    AND v.tipo='candidato' AND v.referencia={can}
) THEN 'true' ELSE 'false' END AS recuperacion \\gset
\\if :recuperacion
\\else
  SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
  -- Bloquea nuevas versiones de la bolsa hasta terminar el COMMIT.
  LOCK TABLE vec_bolsa_llamamientos.bolsa_constituida IN SHARE MODE;
  DO $seleccion$
  BEGIN
    IF NOT EXISTS (
      SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato vc
      JOIN vec_bolsa_llamamientos.constitucion c ON c.acta_ref=vc.acta_ref
      JOIN vec_bolsa_llamamientos.constitucion_entrada e
        ON e.instantanea_ref=vc.instantanea_ref AND e.version_instantanea=vc.version_instantanea
       AND e.participacion_ref=vc.participacion_ref
      JOIN vec_bolsa_llamamientos.bolsa_constituida b
        ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
       AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
      WHERE vc.candidato_ref={can} AND vc.participacion_ref={part}
        AND c.bolsa_ref={bolsa_sql} AND b.estado='vigente'
        AND b.vigente_desde <= clock_timestamp()
        AND (b.vigente_hasta IS NULL OR clock_timestamp() < b.vigente_hasta)
        AND NOT EXISTS (
          SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida posterior
          WHERE posterior.bolsa_ref=b.bolsa_ref AND posterior.version>b.version
        )
    ) THEN RAISE EXCEPTION 'participación importada ya no vigente'; END IF;
  END
  $seleccion$;
  SET LOCAL ROLE vec_contexto_actor_v1_propietario;
\\endif
DO $proyectar$
DECLARE
  existentes integer;
  completos boolean;
BEGIN
  SELECT count(*) INTO existentes FROM vec_contexto_actor_v1.vinculo_referencia_actual a
  JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref, version)
  WHERE v.persona_ref = {pe} AND v.tipo = 'candidato' AND v.estado = 'activo'
    AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta;
  IF existentes > 1 THEN RAISE EXCEPTION 'varios vínculos candidato vigentes'; END IF;
  IF EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
    JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING (vinculo_ref, version)
    WHERE v.persona_ref <> {pe} AND v.tipo = 'candidato' AND v.referencia = {can}
      AND v.estado = 'activo' AND clock_timestamp() >= v.vigente_desde
      AND clock_timestamp() < v.vigente_hasta
  ) THEN RAISE EXCEPTION 'referencia candidato asignada a otra persona'; END IF;
  SELECT EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.procedencias x WHERE x.procedencia_ref={p}
      AND x.procedencia_version=1 AND x.procedencia_huella_sha256={h}
      AND x.procedencia_autoridad={authority}
  ) AND EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
    JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
    WHERE a.cuenta_ref={c} AND a.version=1 AND v.estado='activo'
      AND v.procedencia_ref={p} AND v.procedencia_version=1 AND v.procedencia_huella_sha256={h}
      AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta
  ) AND EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.persona_actual a
    JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
    WHERE a.persona_ref={pe} AND a.version=1 AND v.estado='activo'
      AND v.procedencia_ref={p} AND v.procedencia_version=1 AND v.procedencia_huella_sha256={h}
      AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta
  ) AND EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a
    JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
    WHERE a.perfil_ref={pr} AND a.version=1 AND v.persona_ref={pe} AND v.estado='activo'
      AND v.procedencia_ref={p} AND v.procedencia_version=1 AND v.procedencia_huella_sha256={h}
      AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta
  ) AND EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
    JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
    WHERE a.vinculo_ref={vc} AND a.version=1 AND v.cuenta_ref={c} AND v.perfil_ref={pr}
      AND v.persona_ref={pe} AND v.estado='activo' AND v.procedencia_ref={p}
      AND v.procedencia_version=1 AND v.procedencia_huella_sha256={h}
      AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta
  ) AND EXISTS (
    SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
    JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
    WHERE a.vinculo_ref={vr} AND a.version=1 AND v.persona_ref={pe}
      AND v.tipo='candidato' AND v.referencia={can} AND v.estado='activo'
      AND v.procedencia_ref={p} AND v.procedencia_version=1 AND v.procedencia_huella_sha256={h}
      AND clock_timestamp() >= v.vigente_desde AND clock_timestamp() < v.vigente_hasta
  ) INTO completos;
  IF completos AND existentes = 1 THEN RETURN; END IF;
  IF existentes <> 0 OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.procedencias WHERE procedencia_ref={p}) OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref={c}) OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref={pe}) OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref={pr}) OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref={vc}) OR
     EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual WHERE vinculo_ref={vr})
  THEN RAISE EXCEPTION 'proyección candidato preexistente distinta o parcial'; END IF;
  INSERT INTO vec_contexto_actor_v1.procedencias VALUES ({p},1,{h},{authority});
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
    ({c},1,{p},1,{h},{authority},'activo',clock_timestamp()-interval '1 hour',{fin});
  INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ({c},1);
  INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
    ({pe},1,{p},1,{h},{authority},'activo',clock_timestamp()-interval '1 hour',{fin});
  INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ({pe},1);
  INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
    ({pr},1,{pe},{p},1,{h},{authority},'activo',clock_timestamp()-interval '1 hour',{fin});
  INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES ({pr},1);
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
    ({vc},1,{c},{pr},{pe},{p},1,{h},{authority},'activo',clock_timestamp()-interval '1 hour',{fin});
  INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ({vc},1);
  INSERT INTO vec_contexto_actor_v1.vinculo_referencia_versiones VALUES
    ({vr},1,{pe},'candidato',{can},{p},1,{h},{authority},'activo',clock_timestamp()-interval '1 hour',{fin});
  INSERT INTO vec_contexto_actor_v1.vinculo_referencia_actual VALUES ({vr},1);
END
$proyectar$;
-- La misma proyección se ensaya con ROLLBACK antes de admitir el COMMIT.
FINALIZAR;
'''
(destino / 'proyeccion.sql').write_text(sql)
PY

for nombre in candidato.json bolsa-candidato.json; do
  ruta=$material/identidad/$nombre
  if [[ -e $ruta ]]; then
    [[ -f $ruta && $(stat -c %a -- "$ruta") == 600 ]] || fallar 'manifiesto candidato inseguro'
    cmp -s -- "$temporal/$nombre" "$ruta" || fallar 'manifiesto candidato preexistente distinto'
  fi
done

sed 's/^FINALIZAR;$/ROLLBACK;/' "$temporal/proyeccion.sql" | psql_contenedor >/dev/null || fallar 'ensayo SQL fallido; no se hizo COMMIT'
sed 's/^FINALIZAR;$/COMMIT;/' "$temporal/proyeccion.sql" | psql_contenedor >/dev/null || fallar 'COMMIT SQL fallido'
for nombre in candidato.json bolsa-candidato.json; do
  if [[ ! -e $material/identidad/$nombre ]]; then
    install -m 0600 -- "$temporal/$nombre" "$material/identidad/$nombre"
  fi
done
rm -f -- "$intencion"

printf 'Candidato de desarrollo preparado; proyección ensayada y confirmada.\n'
printf 'Reiniciar solo la aplicación: podman restart %q\n' "$aplicacion"
printf 'Consulta: curl --fail-with-body --cert %q --key %q --cacert %q %q\n' \
  "$cert" "$clave" "$material/ca/ca.crt" "${url%/}/api/vec/bolsa/mi-bolsa"
printf 'PKCS#12 para navegador: %s (contraseña en %s).\n' "$p12" "$password"
