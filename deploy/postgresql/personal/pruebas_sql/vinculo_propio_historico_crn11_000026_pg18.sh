#!/usr/bin/env bash
# Ensayo CRN11 nominal sobre el clon privado cedido por Dirección.
# Genera un overlay Go efímero fuera de Git, reutilizando los emisores V3.
# No reaplica prefijos ni ejecuta DOWN; no consulta principal ni servicios externos.
set -euo pipefail
umask 077
base_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
container=${VEC_CRN11_CLONE_CONTAINER:?clon privado cedido requerido}
scratch=${VEC_CRN11_SCRATCH:?scratch propio externo requerido}
helper="$scratch/generate_overlay.py"
modcache=${VEC_CRN11_MODCACHE:?cache local de módulos requerida sin descargas}
fallo() { printf 'CRN11 FALLO: %s\n' "$1" >&2; exit 1; }
[[ $container == codexb-crn11-20261003-pg ]] || fallo 'nombre de clon CRN11 propio requerido'
[[ $scratch == /dev/shm/codexb-crn11-20261003-* && -d $scratch && ! -L $scratch && -O $scratch && $(stat -c %a "$scratch") == 700 ]] || fallo 'scratch propio 0700 requerido'
[[ -d $modcache && ! -L $modcache ]] || fallo 'cache local de módulos ausente'
[[ $(docker inspect --format '{{.HostConfig.NetworkMode}} {{.HostConfig.Memory}} {{.HostConfig.NanoCpus}} {{.HostConfig.PidsLimit}}' "$container") == 'none 2147483648 2000000000 128' ]] || fallo 'clon fuera de límites aislados'
psql_run() { local user=$1; shift; docker exec -i "$container" /usr/bin/env -i PATH=/usr/local/bin:/usr/bin:/bin psql -h /tmp -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U "$user" -d postgres "$@"; }
valor() { psql_run postgres -c "$1"; }
archivo() { psql_run postgres < "$1" > "$scratch/sql.log" 2>&1 || fallo 'SQL falló; diagnóstico privado en scratch/sql.log'; }
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'PostgreSQL18.4 requerido'
[[ ! -e $scratch/capacidad_v3_vector_sql_test.go && ! -e $scratch/overlay.json ]] || fallo 'scratch ya usado: conservarlo y elegir uno nuevo'
# Captura todos los consumidores presentes, incluidos cuerpos PL/pgSQL que
# pg_depend no rastrea. El núcleo se normaliza únicamente por la extensión149.
python3 - "$repo_dir" "$scratch/preservacion.sql" <<'PYPRESERVACION'
import pathlib,re,sys
raiz=pathlib.Path(sys.argv[1]);salida=pathlib.Path(sys.argv[2])
fuente=(raiz/'deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql').read_text()
def fragmento(nombre):
    encontrados=re.findall(r'\b'+nombre+r' text:=\$'+nombre+r'\$(.*?)\$'+nombre+r'\$;',fuente,re.S)
    assert len(encontrados)==1, 'AD149 debe contener cada segmento una vez'
    return encontrados[0]
normalizar='p.prosrc'
for i,(viejo,nuevo) in enumerate((('extension','marca'),('excl_nuevo','excl'),('runtime_nuevo','runtime'))):
    retirar=fragmento(viejo)+(fragmento('marca') if viejo=='extension' else '')
    restaurar=fragmento(nuevo)
    tag='$crn11segmento'+str(i)+'$'
    assert tag not in retirar+restaurar
    normalizar='replace('+normalizar+','+tag+retirar+tag+','+tag+restaurar+tag+')'
consulta=r"""
SET search_path=pg_catalog,pg_temp;
WITH consumidores AS (
 SELECT p.* FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname IN ('vec_autorizacion_atestada_v3','vec_personal','vec_contexto_actor_v1')
 AND p.proname NOT IN ('consumir_vinculo_propio_crn11_v3_atestada','consultar_vinculo_propio_historico_crn11_v1')
), fuentes AS (
 SELECT p.*, CASE WHEN p.proname='consumir_decision_mutacion_v3_interna'
 THEN __NORMALIZAR__ ELSE p.prosrc END AS normalizada FROM consumidores p
)
SELECT jsonb_build_object('consumidores',(
 SELECT jsonb_agg(jsonb_build_object('firma',p.oid::regprocedure::text,
 'definicion',replace(pg_get_functiondef(p.oid),p.prosrc,p.normalizada),
 'fuente',p.normalizada,'metadatos',to_jsonb(p)-'prosrc'-'normalizada',
 'dependencias',coalesce((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),'[]'::jsonb),
 'dependencias_compartidas',coalesce((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
 FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=p.oid),'[]'::jsonb)) ORDER BY p.oid) FROM fuentes p),
 'audiencias',(SELECT jsonb_build_object('definicion',replace(pg_get_constraintdef(c.oid,true),
 ', ''vec_personal.vinculo_propio.crn11.v1''::text',''),
 'metadatos',to_jsonb(c)-'oid'-'conbin',
 'not_null',(SELECT a.attnotnull FROM pg_attribute a WHERE a.attrelid=c.conrelid AND a.attname='audiencia_consumo'))
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check'));
"""
salida.write_text(consulta.replace('__NORMALIZAR__',normalizar))
PYPRESERVACION
capturar_preservacion() {
 psql_run postgres < "$scratch/preservacion.sql" > "$1" 2> "$scratch/preservacion.log" || fallo 'captura de consumidores falló'
 python3 - "$1" <<'PYVALIDAR'
import json,pathlib,sys
s=json.loads(pathlib.Path(sys.argv[1]).read_text())
assert len(s['consumidores'])>1
assert all(c['definicion'] and c['fuente'] for c in s['consumidores'])
assert s['audiencias']['metadatos']['convalidated'] and s['audiencias']['not_null']
PYVALIDAR
}
capturar_preservacion "$scratch/preservacion_antes.json"

[[ $(valor "SELECT to_regprocedure('vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] || fallo 'Personal26 ya instalada: no se reaplica este ensayo'
# Preimagen alterada: el cambio propio vive sólo en una TX que aborta.
# El fallo no debe crear la fachada ni alterar consumidores/audiencias previos.
python3 - "$repo_dir" "$scratch/preimagen_alterada.sql" <<'PYNEGATIVA'
import pathlib,sys
raiz=pathlib.Path(sys.argv[1])
fuente=(raiz/'deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql').read_text()
fuente='\n'.join(l for l in fuente.splitlines() if l not in ('BEGIN;','COMMIT;'))
pre=r"""BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $preimagen$ DECLARE d text; n text;
BEGIN
 SELECT pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')) INTO STRICT d;
 n:=replace(d,E'AS $function$\n',E'AS $function$\n-- ensayo CRN11: preimagen alterada\n');
 IF n IS NOT DISTINCT FROM d THEN RAISE EXCEPTION 'marca de ensayo ausente'; END IF;
 EXECUTE n;
END $preimagen$;
"""
pathlib.Path(sys.argv[2]).write_text(pre+fuente+'\nROLLBACK;\n')
PYNEGATIVA
if psql_run postgres < "$scratch/preimagen_alterada.sql" > "$scratch/preimagen.out" 2> "$scratch/preimagen.log"; then
 fallo 'AD149 admitió preimagen alterada'
fi
[[ $(cat "$scratch/preimagen.log") == *'55000'* ]] || fallo 'negativa de preimagen inesperada'
capturar_preservacion "$scratch/preservacion_negativa.json"
cmp -s "$scratch/preservacion_antes.json" "$scratch/preservacion_negativa.json" || fallo 'negativa de preimagen dejó efectos'
[[ $(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] || fallo 'negativa creó fachada149'
printf 'CRN11 preimagen alterada: SQLSTATE 55000 y conservación íntegra.\n'

# Las únicas migraciones instalables son las nuevas y ausentes.
sha256sum "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql" "$repo_dir/deploy/postgresql/personal/migraciones/000026_vinculo_propio_historico_crn11.up.sql" > "$scratch/migraciones_journal.txt"
sha256sum "$scratch/preservacion_antes.json" >> "$scratch/migraciones_journal.txt"
if [[ $(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]; then
 archivo "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql"
 printf 'AD149 instalada una vez\n' >> "$scratch/migraciones_journal.txt"
fi
archivo "$repo_dir/deploy/postgresql/personal/migraciones/000026_vinculo_propio_historico_crn11.up.sql"
printf 'Personal26 instalada una vez\n' >> "$scratch/migraciones_journal.txt"
capturar_preservacion "$scratch/preservacion_post149.json"
cmp -s "$scratch/preservacion_antes.json" "$scratch/preservacion_post149.json" || fallo 'AD149 alteró consumidores, ACL, dependencias o audiencias anteriores'
# El CHECK rechaza desconocidas; NOT NULL conserva su rechazo de NULL.
valor "DO \$audiencias\$ DECLARE e text; desconocida boolean; propia boolean;
 BEGIN
 SELECT pg_get_expr(c.conbin,c.conrelid) INTO STRICT e FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated;
 EXECUTE 'SELECT ('||e||') FROM (SELECT \$1::text AS audiencia_consumo) x' INTO desconocida USING 'audiencia:crn11:desconocida';
 EXECUTE 'SELECT ('||e||') FROM (SELECT \$1::text AS audiencia_consumo) x' INTO propia USING 'vec_personal.vinculo_propio.crn11.v1';
 IF desconocida IS DISTINCT FROM false OR propia IS DISTINCT FROM true
 OR NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND attname='audiencia_consumo' AND attnotnull)
 THEN RAISE EXCEPTION 'CHECK o NOT NULL de audiencia incompatible'; END IF;
 END \$audiencias\$;" > /dev/null
archivo "$base_dir/vinculo_propio_historico_crn11_000026.sql"
# Usuarios mínimos propios del ensayo. No hay SET ROLE del runtime al propietario.
valor "CREATE ROLE vec_crn11_ensayo_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 GRANT vec_personal_ejecutor TO vec_crn11_ensayo_runtime WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
 CREATE ROLE vec_crn11_ensayo_ca LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 GRANT vec_contexto_actor_v1_runtime TO vec_crn11_ensayo_ca WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
 GRANT CONNECT ON DATABASE postgres TO vec_crn11_ensayo_runtime;
 GRANT USAGE ON SCHEMA public TO vec_crn11_ensayo_runtime;
 GRANT SELECT ON public.crn11_ensayo_vector TO vec_crn11_ensayo_runtime;
 GRANT EXECUTE ON FUNCTION public.crn11_ensayo_consultar(text,text) TO vec_crn11_ensayo_runtime;" > /dev/null
# Secretaría sintética: bytes nuevos sólo en archivo 0600 y base aislada.
head -c 32 /dev/urandom > "$scratch/hmac.bin"
python3 - "$scratch/hmac.bin" "$scratch/gobierno.sql" <<'PY'
import pathlib,sys
s=pathlib.Path(sys.argv[1]).read_bytes().hex()
p=pathlib.Path(sys.argv[2])
p.write_text("BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref) SELECT 'clave:crn11:ensayo',max(version)+1,max(revision_gobierno)+1,repeat('f',64),decode('"+s+"','hex'),encode(sha256(decode('"+s+"','hex')),'hex'),'broker-crn11-sintetico','vec_personal.vinculo_propio.crn11.v1',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours','acto:crn11:clave' FROM vec_autorizacion_atestada_v3.clave_capacidad_version; INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) SELECT (SELECT max(orden)+1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision),clave_id,version,clock_timestamp(),'acto:crn11:puntero-clave' FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:crn11:ensayo'; COMMIT;\n")
PY
archivo "$scratch/gobierno.sql"
motivo_secuencia=$(valor "SELECT ultima_secuencia+1 FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id")
valor "BEGIN; SET LOCAL ROLE vec_autorizacion_motivos_proyector;
 SELECT vec_autorizacion.publicar_motivos_autorizacion_v2('evento_1111111111111111111111111111c011',
 $motivo_secuencia,repeat('e',64),'motivos_crn11_ensayo',1,repeat('e',64),clock_timestamp()-interval '1 minute',
 jsonb_build_array(jsonb_build_object('clave','motivo_11111111111111111111111111111111','vigente_desde',to_char(clock_timestamp()-interval '2 minutes','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),'vigente_hasta',NULL))); COMMIT;" > "$scratch/motivo.log" 2>&1 || fallo 'publicación motivo sintético falló'

# Generación de overlay con fuente original de sólo lectura.
cat > "$helper" <<'CRN11_GENERADOR_PY'
#!/usr/bin/env python3
# Generates an ephemeral Go test overlay without changing the target worktree.
import json
import os
import pathlib
import sys

if len(sys.argv) != 3:
    raise SystemExit('usage: generate_overlay.py WORKTREE SCRATCH')
root = pathlib.Path(sys.argv[1]).resolve(strict=True)
scratch = pathlib.Path(sys.argv[2]).resolve(strict=True)
source = root / 'internal/vec/adapters/seguridad/confianzaatestacion/capacidad_v3_vector_sql_test.go'
original = source.read_text()
original = original.replace('"crypto/rand"', '"crypto/rand"\n\t"crypto/sha256"\n\t"fmt"\n\t"io"')
original = original.replace('"vec-diputacion-granada/internal/vec/domain"', '"vec-diputacion-granada/internal/vec/application"\n\t"vec-diputacion-granada/internal/vec/domain"')
extra = r'''

// CRN11 is an ephemeral synthetic PDP fixture. Neither this test nor its
// import bridge establishes production identity, PDP governance or database I/O.
// The parent SQL runner must register the exact preparation before importing
// its receipt; SQL consumers revalidate that durable state before any effect.
const crn11Accion = "personal.vinculo_propio.crn11.consultar"
const crn11Audiencia = "vec_personal.vinculo_propio.crn11.v1"
const crn11Finalidad = "acreditar_vinculo_historico_propio_crn11"
const crn11Operacion = "vinculo_propio_historico_crn11"
const crn11Tipo = "vinculo_historico_propio_crn11"
const crn11FormatoTiempo = "2006-01-02T15:04:05.000000Z"
var crn11Campos = []string{"empleado_ref", "fuente_ref", "persona_ref", "version", "vinculo_ref"}

type crn11Confirmacion struct {
    DecisionRef string `json:"decision_ref"`
    DecisionHuellaSHA256 string `json:"decision_huella_sha256"`
    EmitidaEn string `json:"emitida_en"`
    ValidaHasta string `json:"valida_hasta"`
    RegistradaEn string `json:"registrada_en"`
}
type crn11Entrada struct {
    entradaVectorSQLO205
    Fase string `json:"fase"`
    EmpleadoRef string `json:"empleado_ref"`
    MaterialCanonicoB64 string `json:"material_canonico_b64"`
    ClaveHMACArchivo string `json:"clave_hmac_archivo"`
    Confirmacion *crn11Confirmacion `json:"confirmacion,omitempty"`
}
type crn11Salida struct {
    bundleVectorO205
    Fase string `json:"fase"`
    MaterialCanonicoB64 string `json:"material_canonico_b64"`
    MaterialHuellaSHA256 string `json:"material_huella_sha256"`
    DecisionHuellaSHA256 string `json:"decision_huella_sha256"`
    Confirmacion *crn11Confirmacion `json:"confirmacion,omitempty"`
    Campos []string `json:"campos"`
    Obligaciones []string `json:"obligaciones"`
    Politicas []domain.PoliticaRestrictiva `json:"politicas"`
    RevisionCatalogo uint64 `json:"revision_catalogo"`
    HuellaCatalogoSHA256 string `json:"huella_catalogo_sha256"`
    PDP string `json:"pdp"`
}

func TestGenerarVectorCRN11ParaSQL(t *testing.T) {
    input, output := os.Getenv("VEC_CRN11_VECTOR_ENTRADA"), os.Getenv("VEC_CRN11_VECTOR_SALIDA")
    if input == "" || output == "" { t.Skip("ephemeral CRN11 runner only") }
    contenido, err := os.ReadFile(input)
    if err != nil || len(contenido) > 2*1024*1024 { t.Fatal("CRN11 input unavailable or oversized") }
    var e crn11Entrada
    d := json.NewDecoder(bytes.NewReader(contenido)); d.DisallowUnknownFields()
    if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF { t.Fatal("CRN11 input invalid") }
    if (e.Fase != "preparar" && e.Fase != "emitir") || e.Caso == "" || strings.ContainsAny(e.Caso, "/\\") { t.Fatal("CRN11 phase/case invalid") }
    if e.ClaveHMACB64 != "" { t.Fatal("HMAC must be supplied in a private file") }
    salida := crn11Generar(t, e)
    canon, err := json.Marshal(salida)
    if err != nil { t.Fatal("CRN11 output encoding failed") }
    crn11EscribirPrivado(t, output, canon)
}

func crn11EscribirPrivado(t *testing.T, ruta string, contenido []byte) {
    t.Helper()
    f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
    if err != nil { t.Fatal("CRN11 private output unavailable") }
    if _, err := f.Write(contenido); err != nil { f.Close(); t.Fatal("CRN11 private output write failed") }
    if f.Close() != nil { t.Fatal("CRN11 output close failed") }
}

func crn11Generar(t *testing.T, e crn11Entrada) crn11Salida {
    t.Helper()
    ahora := crn11Instante(t, e.Ahora)
    resuelto := crn11Instante(t, e.ResueltoEn)
    contexto := exigirBase64O205(t, e.ContextoB64)
    manifiesto := exigirBase64O205(t, e.ManifiestoB64)
    material := exigirBase64O205(t, e.MaterialCanonicoB64)
    if len(material) == 0 || len(material) > 128*1024 || !json.Valid(material) { t.Fatal("CRN11 material invalid") }
    // Canonical PostgreSQL JSONB text is supplied by the parent. It is hashed
    // byte-for-byte; Go must not reserialize it or alter ordering/whitespace.
    var objeto map[string]json.RawMessage
    if json.Unmarshal(material, &objeto) != nil || len(objeto) != 13 { t.Fatal("CRN11 material requires exactly 13 keys") }
    claves := []string{"esquema", "empleado_ref", "actor_ref", "contexto_actor_ref", "contexto_version", "cuenta_ref", "cuenta_version", "perfil_ref", "perfil_version", "persona_ref", "persona_version", "vinculo_ref", "vinculo_version"}
    for _, clave := range claves { if _, ok := objeto[clave]; !ok { t.Fatal("CRN11 material key missing") } }
    crn11ExigirCampo(t, objeto, "esquema", "vec.personal.vinculo-propio-crn11.consulta.v1")
    crn11ExigirCampo(t, objeto, "empleado_ref", e.EmpleadoRef)
    suma := sha256.Sum256(material)
    huellaMaterial := fmt.Sprintf("%x", suma)
    var plantilla decisionPlantillaO205
    var motivo motivoCanonicoO205
    if json.Unmarshal(exigirBase64O205(t, e.DecisionPlantillaB64), &plantilla) != nil ||
        json.Unmarshal(exigirBase64O205(t, e.MotivoB64), &motivo) != nil { t.Fatal("CRN11 decision/motive invalid") }
    if plantilla.Accion != crn11Accion || plantilla.ModuloID != "personal" || plantilla.TipoRecurso != crn11Tipo || plantilla.Finalidad != crn11Finalidad || plantilla.RecursoRef != e.EmpleadoRef || e.AudienciaConsumo != crn11Audiencia { t.Fatal("CRN11 nominal authority mismatch") }
    if e.EmpleadoRef == "" { t.Fatal("CRN11 employee reference missing") }
    motivoCanon, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(motivo.Referencia)
    if err != nil || !bytes.Equal(motivoCanon, exigirBase64O205(t, e.MotivoB64)) { t.Fatal("CRN11 motive is not canonical") }
    actor, err := domain.RehidratarContextoActorVinculadoV2(contexto)
    if err != nil { t.Fatal("CRN11 actor invalid") }
    crn11ExigirCampo(t, objeto, "actor_ref", actor.Principal.ID)
    crn11ExigirCampo(t, objeto, "contexto_actor_ref", actor.Instantanea.VinculoRef)
    crn11ExigirCampo(t, objeto, "cuenta_ref", actor.Instantanea.CuentaRef)
    crn11ExigirCampo(t, objeto, "cuenta_version", actor.Instantanea.CuentaVersion)
    crn11ExigirCampo(t, objeto, "perfil_ref", actor.PerfilActivoRef)
    crn11ExigirCampo(t, objeto, "perfil_version", actor.Instantanea.PerfilVersion)
    crn11ExigirCampo(t, objeto, "persona_ref", actor.Instantanea.PersonaRef)
    crn11ExigirCampo(t, objeto, "persona_version", actor.Instantanea.PersonaVersion)
    if actor.Instantanea.PersonaVersion != e.PersonaVersion || actor.Instantanea.PerfilVersion != e.PerfilVersion { t.Fatal("CRN11 actor versions mismatch") }
    // The granted scope must be supported by the active actor snapshot.
    empleadoLigado := false
    for _, enlace := range actor.Instantanea.Vinculos {
        if enlace.Tipo == domain.TipoReferenciaContextoActorEmpleado && enlace.Referencia == e.EmpleadoRef && enlace.VigenteEn(ahora) { empleadoLigado = true }
    }
    if !empleadoLigado { t.Fatal("CRN11 employee is not bound to actor") }
    resultado := domain.ResultadoContextoActorRegistradoV2{
        RegistroContextoRef: plantilla.VinculoAutenticacionActor.RegistroContextoRef,
        Contexto: actor, RepresentacionCanonica: contexto,
        HuellaSHA256: plantilla.VinculoAutenticacionActor.ContextoActorHuellaSHA256,
        ManifiestoProcedenciaCanonico: manifiesto,
        ManifiestoProcedenciaHuellaSHA256: e.ManifiestoHuellaSHA256,
        AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorV1(e.AutoridadEfectiva),
        ResueltoEnAutoritativo: resuelto,
    }
    if resultado.Validar() != nil { t.Fatal("CRN11 registered context invalid") }
    vinculo, err := domain.CrearVinculoAutenticacionActorV2(context.Background(),
        revalidadorConfianzaAtestacionV3Prueba{resultado: plantilla.VinculoAutenticacionActor.Autenticacion()},
        domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: plantilla.VinculoAutenticacionActor.AutenticacionRef, SesionRef: plantilla.VinculoAutenticacionActor.SesionRef},
        resolutorConfianzaAtestacionV3Prueba{resultado: resultado},
        domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: actor.Instantanea.CuentaRef, Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}, PerfilActivoRef: actor.PerfilActivoRef},
        &relojConfianzaAtestacionV3Prueba{ahora: ahora})
    if err != nil { t.Fatal("CRN11 nominal authentication binding failed") }
    correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionConfianzaAtestacionV3Prueba{valor: plantilla.CorrelacionRef})
    if err != nil { t.Fatal("CRN11 correlation failed") }
    solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
        VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo.Referencia,
        Accion: crn11Accion, Finalidad: crn11Finalidad, Correlacion: correlacion,
        Recurso: domain.RecursoAutorizable{Referencia: e.EmpleadoRef, ModuloID: "personal", Tipo: crn11Tipo,
            Ambitos: map[string]string{"empleado_ref": e.EmpleadoRef},
            Atributos: map[string]string{"material_sha256": huellaMaterial, "operacion": crn11Operacion}},
    })
    if err != nil { t.Fatal("CRN11 nominal request failed") }
    id := strings.ReplaceAll(e.Caso, "-", "_")
    rol := domain.VersionRol{
        RolID: "crn11_go_"+id, Version: 1, Nombre: "CRN11 synthetic nominal fixture", Estado: domain.EstadoVersionRolPublicada,
        Concesiones: []domain.ConcesionRol{{Accion: crn11Accion, ModuloID: "personal", TipoRecurso: crn11Tipo, Finalidades: []string{crn11Finalidad}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: append([]string(nil), crn11Campos...), Obligaciones: []string{}}},
        PublicadaPor: "autoridad-crn11-go-sintetica", PublicadaEn: ahora.Add(-10*time.Minute),
    }
    control := domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora.Add(-5*time.Minute)}
    asignacion := domain.AsignacionPerfil{AsignacionID: e.AsignacionID, Version: e.AsignacionVersion, PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.Principal.ID, VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
        Ambitos: []domain.AmbitoPerfil{{Clave: "empleado_ref", Valores: []string{e.EmpleadoRef}}}, EmitidaPor: rol.PublicadaPor, EmitidaEn: ahora.Add(-10*time.Minute), VigenteDesde: ahora.Add(-5*time.Minute), VigenteHasta: ahora.Add(30*time.Minute)}
    huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(e.Politicas)
    if err != nil || huellaCatalogo != e.HuellaCatalogoSHA256 || e.RevisionCatalogo == 0 { t.Fatal("CRN11 policy catalog mismatch") }
    instantanea := domain.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol, ControlVigenciaVersionRol: control, Politicas: e.Politicas, RevisionCatalogoPoliticas: e.RevisionCatalogo, CatalogoPoliticasHuellaSHA256: e.HuellaCatalogoSHA256}
    evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, plantilla.DecisionRef, ahora, ahora.Add(90*time.Second))
    if err != nil { t.Fatal("CRN11 nominal evaluation failed") }
    decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
    if err != nil { t.Fatal("CRN11 nominal decision failed") }
    concedida, codigo, err := decision.Resultado()
    if err != nil || !concedida || codigo != "concedida" { t.Fatal("CRN11 policy fixture denied") }
    decisionCanon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
    if err != nil { t.Fatal("CRN11 decision serialization failed") }
    huellaDecision, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
    if err != nil { t.Fatal("CRN11 decision hash failed") }
    salida := crn11Salida{Fase: e.Fase, MaterialCanonicoB64: e.MaterialCanonicoB64, MaterialHuellaSHA256: huellaMaterial, DecisionHuellaSHA256: huellaDecision, Campos: append([]string(nil), crn11Campos...), Obligaciones: []string{}, Politicas: e.Politicas, RevisionCatalogo: e.RevisionCatalogo, HuellaCatalogoSHA256: e.HuellaCatalogoSHA256, PDP: "fixture_nominal_sintetica_sin_proveedor_operativo"}
    salida.DecisionB64 = base64.StdEncoding.EncodeToString(decisionCanon)
    salida.MotivoB64 = base64.StdEncoding.EncodeToString(motivoCanon)
    salida.ContextoB64 = e.ContextoB64
    salida.PersonaVersion = e.PersonaVersion; salida.PerfilVersion = e.PerfilVersion
    salida.VersionRolDocumento = exigirJSONO205(t, rol); salida.ControlRolDocumento = exigirJSONO205(t, control); salida.AsignacionDocumento = exigirJSONO205(t, asignacion)
    if e.Fase == "preparar" {
        if e.Confirmacion != nil { t.Fatal("CRN11 preparation must not import a receipt") }
        return salida
    }
    if e.Confirmacion == nil { t.Fatal("CRN11 emission requires SQL receipt") }
    registradaEn := crn11Instante(t, e.Confirmacion.RegistradaEn)
    if registradaEn.Before(ahora) { t.Fatal("CRN11 registration precedes decision") }
    instanteCripto := time.Now().UTC().Truncate(time.Microsecond)
    if instanteCripto.Before(registradaEn) { instanteCripto = registradaEn }
    puente := &crn11PDP{t: t, decision: decision, confirmacion: *e.Confirmacion}
    publica, privada, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatal("CRN11 ephemeral root generation failed") }
    defer borrarBytesConfianzaAtestacion(privada)
    raiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(e.RaizClaveID, e.RaizVersion, publica, e.AudienciaDespliegue, EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-10*time.Minute), ahora.Add(time.Hour), time.Time{})
    if err != nil { t.Fatal("CRN11 ephemeral root invalid") }
    config, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(e.RevisionConfianza, e.SecuenciaConfianza, ahora.Add(-5*time.Minute), ahora.Add(30*time.Minute), raiz)
    if err != nil { t.Fatal("CRN11 trust fixture invalid") }
    reloj := &relojConfianzaAtestacionV3Prueba{ahora: instanteCripto}
    confianza, err := NuevoServicioConfianzaAtestacionAutorizacionV3(config, reloj)
    if err != nil { t.Fatal("CRN11 trust service failed") }
    cabecera := domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: e.RaizClaveID, Audiencia: e.AudienciaDespliegue}
    firmante := crn11Firmante{t: t, privada: privada, cabecera: cabecera, ahora: instanteCripto}
    atestador, err := application.NuevoServicioAtestacionesAutorizacionV3(cabecera, firmante)
    if err != nil { t.Fatal("CRN11 attestation service failed") }
    claveBytes := crn11LeerHMAC(t, e.ClaveHMACArchivo)
    defer borrarBytesConfianzaAtestacion(claveBytes)
    clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3(e.ClaveID, e.ClaveVersion, claveBytes, e.EmisorID, crn11Audiencia, EstadoClaveHMACCapacidadAtestacionV3Emision, crn11Instante(t,e.ClaveValidaDesde), crn11Instante(t,e.ClaveValidaHasta), time.Time{}, e.RevisionGobierno, e.HuellaGobiernoSHA256)
    if err != nil { t.Fatal("CRN11 ephemeral HMAC invalid") }
    capacidades, err := NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
    if err != nil { t.Fatal("CRN11 capability emitter failed") }
    emisor, err := NuevoEmisorMaterialAutorizacionAtestadaV3(puente, atestador, confianza, capacidades)
    if err != nil { t.Fatal("CRN11 nominal material emitter failed") }
    _, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), solicitud, resultado)
    if err != nil { t.Fatal("CRN11 public nominal emission failed") }
    exportacion, err := exportador.ExportarMaterialParaConsumidor()
    if err != nil { t.Fatal("CRN11 public nominal material export failed") }
    cd, err := confirmacion.Datos()
    if err != nil { t.Fatal("CRN11 confirmation invalid") }
    salida.Confirmacion = &crn11Confirmacion{DecisionRef: cd.DecisionRef, DecisionHuellaSHA256: cd.DecisionHuellaSHA256, EmitidaEn: cd.EmitidaEn.Format(crn11FormatoTiempo), ValidaHasta: cd.ValidaHasta.Format(crn11FormatoTiempo), RegistradaEn: cd.RegistradaEn.Format(crn11FormatoTiempo)}
    salida.CapacidadB64 = base64.StdEncoding.EncodeToString(exportacion.CapacidadCanonica())
    salida.DecisionB64 = base64.StdEncoding.EncodeToString(exportacion.DecisionCanonica())
    salida.MotivoB64 = base64.StdEncoding.EncodeToString(exportacion.MotivoCanonico())
    salida.ContextoB64 = base64.StdEncoding.EncodeToString(exportacion.ContextoActorCanonico())
    salida.PayloadB64 = base64.StdEncoding.EncodeToString(exportacion.PayloadVECAD3())
    salida.COSEB64 = base64.StdEncoding.EncodeToString(exportacion.SobreCOSESign1())
    salida.EvidenciaB64 = base64.StdEncoding.EncodeToString(exportacion.EvidenciaVerificacion())
    salida.SPKIB64 = base64.StdEncoding.EncodeToString(exportacion.RaizPublicaSPKI())
    return salida
}

func crn11ExigirCampo(t *testing.T, objeto map[string]json.RawMessage, clave string, esperado any) {
    t.Helper()
    canon, err := json.Marshal(esperado)
    if err != nil || !bytes.Equal(objeto[clave], canon) { t.Fatal("CRN11 material context mismatch") }
}
func crn11ParsearInstante(valor string) (time.Time, error) {
    instante, err := time.Parse(time.RFC3339Nano, valor)
    if err != nil || instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 || instante.Nanosecond()%1000 != 0 { return time.Time{}, errors.New("CRN11 timestamp invalid") }
    _, desplazamiento := instante.Zone()
    if desplazamiento != 0 { return time.Time{}, errors.New("CRN11 timestamp must be UTC") }
    return instante.UTC(), nil
}
func crn11Instante(t *testing.T, valor string) time.Time {
    t.Helper()
    instante, err := crn11ParsearInstante(valor)
    if err != nil { t.Fatal("CRN11 timestamp invalid") }
    return instante
}

func crn11LeerHMAC(t *testing.T, ruta string) []byte {
    t.Helper()
    info, err := os.Lstat(ruta)
    if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() != 32 { t.Fatal("CRN11 HMAC file must be regular, 0600 and 32 raw bytes") }
    f, err := os.Open(ruta)
    if err != nil { t.Fatal("CRN11 HMAC file unavailable") }
    defer f.Close()
    actual, err := f.Stat()
    if err != nil || !os.SameFile(info, actual) { t.Fatal("CRN11 HMAC file changed") }
    clave, err := io.ReadAll(io.LimitReader(f, 33))
    if err != nil || len(clave) != 32 { t.Fatal("CRN11 HMAC file invalid") }
    return clave
}

type crn11PDP struct { t *testing.T; decision domain.DecisionAutorizacionLigadaV3; confirmacion crn11Confirmacion }
func (p *crn11PDP) ExigirSolicitudLigadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
    d, err := s.Datos()
    if err != nil { return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err }
    orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, p.decision, d.ReferenciaMotivo, r)
    if err != nil { return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err }
    confirmada, err := ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, p, orden)
    return p.decision, confirmada, err
}
func (p *crn11PDP) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(_ context.Context, orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
    d, err := orden.Datos()
    if err != nil { return time.Time{}, err }
    canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d.Decision)
    if err != nil { return time.Time{}, err }
    var esperada crn11Confirmacion
    if json.Unmarshal(canon, &esperada) != nil { return time.Time{}, errors.New("CRN11 decision invalid") }
    esperada.DecisionHuellaSHA256, err = domain.HuellaSHA256DecisionAutorizacionV3(d.Decision)
    if err != nil { return time.Time{}, err }
    // Only the registered timestamp comes from SQL; identity, decision hash
    // and validity must match the prepared nominal exactly.
    if esperada.DecisionRef != p.confirmacion.DecisionRef || esperada.DecisionHuellaSHA256 != p.confirmacion.DecisionHuellaSHA256 { return time.Time{}, errors.New("CRN11 durable receipt mismatch") }
    for _, par := range [][2]string{{esperada.EmitidaEn, p.confirmacion.EmitidaEn}, {esperada.ValidaHasta, p.confirmacion.ValidaHasta}} {
        a, ea := crn11ParsearInstante(par[0]); b, eb := crn11ParsearInstante(par[1])
        if ea != nil || eb != nil || !a.Equal(b) { return time.Time{}, errors.New("CRN11 durable receipt time mismatch") }
    }
    return crn11ParsearInstante(p.confirmacion.RegistradaEn)
}

type crn11Firmante struct { t *testing.T; privada ed25519.PrivateKey; cabecera domain.CabeceraAtestacionAutorizacionV3; ahora time.Time }
func (f crn11Firmante) FirmarAtestacionAutorizacionV3(ctx context.Context, solicitud ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
    if ctx.Err() != nil { return ports.ResultadoFirmaAtestacionAutorizacionV3{}, ctx.Err() }
    payload, err := solicitud.Mensaje()
    if err != nil { return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err }
    aad, err := AADExternoAtestacionAutorizacionV3(f.cabecera.Audiencia)
    if err != nil { return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err }
    sobre := firmarSobreConfianzaAtestacionV3Prueba(f.t, f.privada, []byte(f.cabecera.ClaveID), payload, aad)
    return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(solicitud, sobre, "evidencia:crn11:fixture:crypto-real", f.ahora)
}
'''
output = scratch / 'capacidad_v3_vector_sql_test.go'
with output.open('x') as f:
    os.chmod(output, 0o600)
    f.write(original + extra)
with (scratch / 'overlay.json').open('x') as f:
    os.chmod(scratch / 'overlay.json', 0o600)
    json.dump({'Replace': {str(source): str(output)}}, f)
print('overlay prepared')
CRN11_GENERADOR_PY
python3 "$helper" "$repo_dir" "$scratch" > "$scratch/generador.log" 2>&1 || fallo 'generador falló'
python3 - "$scratch/overlay.json" "$repo_dir" "$scratch" <<'PY'
import json,pathlib,sys
p=pathlib.Path(sys.argv[1]); o=json.loads(p.read_text()); o['Replace']={k.replace(sys.argv[2],'/src',1):v.replace(sys.argv[3],'/scratch',1) for k,v in o['Replace'].items()};p.write_text(json.dumps(o))
PY
toolchain=/modcache/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64
sandbox() {
 systemd-run --user --quiet --wait --pipe --collect -p MemoryMax=2G -p TasksMax=256 -p CPUQuota=200% \
 /usr/bin/env -i PATH=/usr/bin:/bin /usr/bin/timeout 1800 /usr/bin/bwrap --unshare-all --die-with-parent \
 --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 --proc /proc --dev /dev --tmpfs /tmp \
 --ro-bind "$repo_dir" /src --ro-bind "$modcache" /modcache --bind "$scratch" /scratch --chdir /src \
 /usr/bin/env -i PATH="$toolchain/bin:/usr/bin:/bin" HOME=/scratch GOROOT="$toolchain" GOTOOLCHAIN=local GOPATH=/scratch/gopath GOMODCACHE=/modcache GOCACHE=/scratch/gocache GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOMAXPROCS=2 \
 VEC_CRN11_VECTOR_ENTRADA=/scratch/entrada.json VEC_CRN11_VECTOR_SALIDA=/scratch/salida.json "$@"
}
sandbox "$toolchain/bin/go" test -c -p 8 -overlay /scratch/overlay.json -o /scratch/crn11.test ./internal/vec/adapters/seguridad/confianzaatestacion > "$scratch/compilar.log" 2>&1 || fallo 'compilación focal aislada falló'

normalizar() { # entrada exportadaSQL; fase; confirmación opcional
 python3 - "$scratch" "$1" "${2:-}" <<'PY'
import base64,datetime,json,pathlib,sys
s=pathlib.Path(sys.argv[1]);p=s/'entrada.json';j=json.loads(p.read_text())
def tiempo(x):
    return datetime.datetime.fromisoformat(x.replace('Z','+00:00')).astimezone(datetime.timezone.utc).isoformat(timespec='microseconds').replace('+00:00','Z')
for k in ('ahora','resuelto_en','clave_valida_desde','clave_valida_hasta'):j[k]=tiempo(j[k])
t=json.loads(base64.b64decode(j['decision_plantilla_b64']));v=t['vinculo_autenticacion_actor']
for k in ('autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en'):v[k]=tiempo(v[k])
j['decision_plantilla_b64']=base64.b64encode(json.dumps(t,separators=(',',':')).encode()).decode()
j['fase']=sys.argv[2];j['clave_hmac_archivo']='/scratch/hmac.bin'
if sys.argv[3]:
    c=json.loads((s/'confirmacion.json').read_text());c['registrada_en']=tiempo(c['registrada_en']);j['confirmacion']=c
p.write_text(json.dumps(j,separators=(',',':')))
PY
}
importar() { # función de ensayo; caso
 python3 - "$scratch" "$1" "$2" <<'PY'
import pathlib,sys
s=pathlib.Path(sys.argv[1]);b=(s/'salida.json').read_text()
assert '$crn11bundle$' not in b
(s/'importar.sql').write_text("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SELECT public."+sys.argv[2]+"('"+sys.argv[3]+"',$crn11bundle$"+b+"$crn11bundle$::jsonb); COMMIT;\n")
PY
 psql_run postgres < "$scratch/importar.sql" > "$scratch/confirmacion.json" 2> "$scratch/importar.log" || fallo 'importación/registro durable falló'
}
preparar() {
 local caso=$1 s
 s=$(printf '%s' "$caso" | sha256sum | cut -c1-32)
 valor "SELECT public.crn11_ensayo_actor('$caso')" > /dev/null
 psql_run vec_crn11_ensayo_ca -c "BEGIN ISOLATION LEVEL SERIALIZABLE;
 SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_crn11_$s','rca_crn11_$s','cta_crn11_$s','prf_crn11_$s','certificado','alto',clock_timestamp(),ARRAY['empleado']); COMMIT;" > "$scratch/contexto.log" 2>&1 || fallo 'registroCA7 nominal falló'
 valor "SELECT public.crn11_ensayo_entrada('$caso')" > "$scratch/entrada.json"
 normalizar preparar
 python3 - "$scratch/salida.json" <<'PYSALIDA'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if p.is_file() and not p.is_symlink():p.unlink()
PYSALIDA
 sandbox /scratch/crn11.test -test.run '^TestGenerarVectorCRN11ParaSQL$' -test.count=1 > "$scratch/preparar.log" 2>&1 || fallo 'preparaciónGo nominal falló'
 importar crn11_ensayo_preparado "$caso"
 normalizar emitir confirmada
 python3 - "$scratch/salida.json" <<'PYSALIDA'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if p.is_file() and not p.is_symlink():p.unlink()
PYSALIDA
 sandbox /scratch/crn11.test -test.run '^TestGenerarVectorCRN11ParaSQL$' -test.count=1 > "$scratch/emitir.log" 2>&1 || fallo 'emisiónCOSE/HMAC real falló'
 importar crn11_ensayo_firmado "$caso"
}
contadores() { valor 'SELECT (SELECT count(*) FROM vec_personal.recibo_vinculo_propio_crn11)||'"'|'"'||(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)||'"'|'"'||(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)'; }
consultar() {
 psql_run vec_crn11_ensayo_runtime -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT public.crn11_ensayo_consultar('$1','${2:-valida}'); COMMIT;"
}
rechazar() {
 local antes despues
 antes=$(contadores)
 if consultar "$1" "${2:-valida}" > "$scratch/denegada.out" 2> "$scratch/denegada.log"; then fallo 'un caso negativo devolvió datos'; fi
 [[ ! -s $scratch/denegada.out ]] || fallo 'una denegación filtró datos'
 [[ $(cat "$scratch/denegada.log") == *'42501'* || $(cat "$scratch/denegada.log") == *'22023'* ]] || fallo 'negativa inesperada: consultar diagnóstico privado'
 despues=$(contadores); [[ $antes == "$despues" ]] || fallo 'una denegación dejó recibo/consumo/auditoría'
 printf 'CRN11 rechazo comprobado: %s/%s\n' "$1" "${2:-valida}"
}
preparar positivo
antes=$(contadores)
consultar positivo > "$scratch/positivo.json" 2> "$scratch/positivo.log" || fallo 'positivo real falló'
python3 - "$scratch/positivo.json" <<'PY'
import json,pathlib,sys
j=json.loads(pathlib.Path(sys.argv[1]).read_text())
assert set(j)=={'vinculo','evidencia'}
assert set(j['vinculo'])=={'persona_ref','empleado_ref','vinculo_ref','fuente_ref','version'}
assert set(j['evidencia'])=={'recibo_ref','decision_ref','efecto_ref','consumo_huella_sha256','auditoria_ref','consultada_en'}
assert j['vinculo']['version']==1
assert j['evidencia']['recibo_ref'].startswith('vinculocrn11:')
PY
despues=$(contadores)
python3 - "$antes" "$despues" <<'PYCOUNTS'
import sys
a=list(map(int,sys.argv[1].split('|')));b=list(map(int,sys.argv[2].split('|')))
assert b==[x+1 for x in a], 'positivo sin recibo, consumo y auditoría nuevos'
PYCOUNTS
[[ $(valor 'SELECT count(*) FROM vec_personal.recibo_vinculo_propio_crn11') == 1 ]] || fallo 'positivo sin recibo único'
printf 'CRN11 positivo real: vínculo mínimo, consumo y auditoría nuevos, recibo durable.\n'
rechazar positivo
for variante in actor ambito campos cose; do preparar "$variante"; rechazar "$variante" "$variante"; done
# Cambios Personal16 después de emitir, antes de consumir.
for estado in activa revocada; do
 caso="avance_$estado"; preparar "$caso"
 s=$(printf '%s' "$caso" | sha256sum | cut -c1-32)
 valor "BEGIN; SET LOCAL ROLE vec_personal_propietario;
 SELECT version FROM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_crn11_$s',2,'per_crn11_$s','emp_crn11_$s','$estado',
 (SELECT vigente_desde FROM vec_personal.proyeccion_empleado_persona_historia WHERE proyeccion_ref='pep_crn11_$s' AND version=1),
 (SELECT vigente_hasta FROM vec_personal.proyeccion_empleado_persona_historia WHERE proyeccion_ref='pep_crn11_$s' AND version=1),
 CASE WHEN '$estado'='revocada' THEN 'error_material' ELSE NULL END,'prc_crn11_$s',1,repeat('a',64)); COMMIT;" > /dev/null
 rechazar "$caso"
done

preparar concurrente
s=$(printf '%s' concurrente | sha256sum | cut -c1-32)
antes=$(contadores)
# El publicador mantiene el consultivo y la generación Personal16 hasta COMMIT.
psql_run postgres -c "SET application_name='crn11_publicador_lento'; BEGIN; SET LOCAL ROLE vec_personal_propietario;
 SELECT version FROM vec_personal.publicar_proyeccion_empleado_persona_v1('pep_crn11_$s',2,'per_crn11_$s','emp_crn11_$s','revocada',
 (SELECT vigente_desde FROM vec_personal.proyeccion_empleado_persona_historia WHERE proyeccion_ref='pep_crn11_$s' AND version=1),
 (SELECT vigente_hasta FROM vec_personal.proyeccion_empleado_persona_historia WHERE proyeccion_ref='pep_crn11_$s' AND version=1),
 'error_material','prc_crn11_$s',1,repeat('a',64)); SELECT pg_sleep(2); COMMIT;" > "$scratch/publicador.out" 2> "$scratch/publicador.log" &
publicador_pid=$!
lista=false
for _ in {1..80}; do
 if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='crn11_publicador_lento' AND wait_event='PgSleep')") == t ]]; then lista=true; break; fi
 sleep 0.02
done
[[ $lista == true ]] || fallo 'publicador no alcanzó barrera concurrente'
if consultar concurrente > "$scratch/concurrente.out" 2> "$scratch/concurrente.log"; then fallo 'consumo concurrente entregó vínculo revocado'; fi
wait "$publicador_pid" || fallo 'publicación concurrente falló'
[[ $(cat "$scratch/concurrente.log") == *'40001'* ]] || fallo 'barrera concurrente no produjo40001'
[[ ! -s $scratch/concurrente.out && $(contadores) == "$antes" ]] || fallo '40001 dejó datos/efectos'
printf 'CRN11 barrera concurrente: SQLSTATE40001, sin datos ni efectos.\n'
# ACL nominal: runtime no accede a proyección, secreto, recibos ni núcleo.
[[ $(valor "SELECT NOT has_table_privilege('vec_crn11_ensayo_runtime','vec_personal.proyeccion_empleado_persona_historia','SELECT')
 AND NOT has_table_privilege('vec_crn11_ensayo_runtime','vec_personal.recibo_vinculo_propio_crn11','SELECT')
 AND NOT has_table_privilege('vec_crn11_ensayo_runtime','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')
 AND NOT has_function_privilege('vec_crn11_ensayo_runtime','vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND (SELECT count(*)=1 FROM pg_auth_members WHERE member='vec_crn11_ensayo_runtime'::regrole)") == t ]] || fallo 'ACL nominal abierta'
capturar_preservacion "$scratch/preservacion_final.json"
cmp -s "$scratch/preservacion_antes.json" "$scratch/preservacion_final.json" || fallo 'el ensayo alteró consumidores, ACL, dependencias o audiencias anteriores'
[[ $(valor 'SELECT count(*) FROM vec_personal.recibo_vinculo_propio_crn11') == 1 ]] || fallo 'negativos añadieron recibos'
# Se retira únicamente el transporte auxiliar propio. La historia sintética
# queda conservada en el clon para las revisiones; no hay DOWN ni reinicio.
valor "REVOKE EXECUTE ON FUNCTION public.crn11_ensayo_consultar(text,text) FROM vec_crn11_ensayo_runtime;
 REVOKE SELECT ON public.crn11_ensayo_vector FROM vec_crn11_ensayo_runtime;
 REVOKE USAGE ON SCHEMA public FROM vec_crn11_ensayo_runtime;
 DROP FUNCTION public.crn11_ensayo_consultar(text,text),public.crn11_ensayo_firmado(text,jsonb),public.crn11_ensayo_preparado(text,jsonb),public.crn11_ensayo_entrada(text),public.crn11_ensayo_actor(text);
 DROP TABLE public.crn11_ensayo_vector;
 REVOKE CONNECT ON DATABASE postgres FROM vec_crn11_ensayo_runtime;
 REVOKE vec_personal_ejecutor FROM vec_crn11_ensayo_runtime;
 REVOKE vec_contexto_actor_v1_runtime FROM vec_crn11_ensayo_ca;
 DROP ROLE vec_crn11_ensayo_runtime,vec_crn11_ensayo_ca;" > /dev/null
sha256sum "$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000149_consumidor_vinculo_propio_crn11.up.sql" "$repo_dir/deploy/postgresql/personal/migraciones/000026_vinculo_propio_historico_crn11.up.sql" "$base_dir/vinculo_propio_historico_crn11_000026.sql" "${BASH_SOURCE[0]}"
printf 'ENSAYO-OK CRN11 PG18.4: COSE/HMAC reales, CA7/Personal16, V3 durable, positivo y negativas, barrera40001 y conservación de todos los consumidores previos y audiencias. Política e identidades sintéticas; sin HTTP, proveedor operativo ni despliegue.\n'
