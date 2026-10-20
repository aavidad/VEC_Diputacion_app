#!/usr/bin/env python3
"""Etapa ADMIN H10 para una copia local de desarrollo sintético autorizada.

Invocar por etapas con --config <archivo privado 0600> --etapa <nombre>.
El archivo de configuración, propio 0600 en directorio 0700, declara:
  version=1, entorno="desarrollo", alcance_fuente="sintetico_declarado",
  destino="principal_desarrollo", backup={ref,sha256},
  kit={source_snapshot,source_copy,source_manifest,source_manifest_sha256,
       modcache,gocache_cache,go,openssl,helper_hmac,helper_hmac_sha256,
       cli:{fuentes_preparar,fuentes_aplicar,unidad,bootstrap_preparar,bootstrap_aplicar},
       cli_sha256:{mismas_claves},catalogos:{mismas_claves},catalogo_sha256:{mismas_claves}},
  privado=<directorio 0700>, psql_comando=[argumentos confiables sin DSN],
  proveedor={directorio_material,ruta_configuracion_hmac,espacio_identidad,
             dominio_ref,espacio_clave,dominio_hmac,incluir_cuenta_ordinaria},
  fuentes={nombres:[dos nombres sintéticos],host_admin,ca_dias,
           maxima_edad_revocacion_segundos,ttl_plan_segundos,
           ttl_personas_segundos,ttl_organizacion_segundos,politica_aprobacion},
  unidad={denominacion,clase,catalogo_ref,catalogo_version,catalogo_revision,
          catalogo_entrada_clave,ttl_plan_segundos,vigencia_dias},
  bootstrap={ttl_plan_segundos,ttl_perfiles_segundos,audiencia_selector,
             audiencia_administrativa,app_role_ref,sys_role_ref,duracion_propuesta_segundos},
  dba={fuentes:{login,proceso,motivo_ref,aprobacion_ref,conexion,aprobacion_externa},
       unidad:{login,aprobacion_ref,conexion,aprobacion_externa},
       bootstrap:{login,proceso,aprobacion_ref,conexion,aprobacion_externa}}.

El operador aporta las aprobaciones externas después de conocer cada huella de
plan. Cada una es JSON privado con `huella_plan_sha256`, `aprobacion_ref` y
`backup_ref` exactos. El programa calcula el SHA256 de ese archivo y exige su
coincidencia; no crea la aprobación ni una firma. La conexión es un JSON
privado propio de la CLI: no se lee ni imprime el DSN.

Etapas: comprobar, preparar_fuentes, configurar_fuentes, aplicar_fuentes,
preparar_unidad, configurar_unidad, aplicar_unidad, preparar_bootstrap,
configurar_bootstrap, aplicar_bootstrap, recuperar_fuentes,
recuperar_unidad, recuperar_bootstrap, verificar. Las tres aplicaciones son
transacciones separadas. Una etapa confirmada se recupera por su recibo; no
se genera otra identidad ni plan al reanudar.
"""
import argparse, datetime as dt, hashlib, json, os, re, secrets, shutil, stat, subprocess, sys, uuid
from pathlib import Path

HMAC_GO = '// Copiar a cmd/vec-fixture-originales-k/main.go en el checkout exacto antes de compilar.\n// Este helper sólo llama al proveedor DEV de Identidad. No conecta con PostgreSQL.\npackage main\n\nimport (\n\t"context"\n\t"crypto/rand"\n\t"encoding/hex"\n\t"encoding/json"\n\t"errors"\n\t"flag"\n\t"fmt"\n\t"io"\n\t"os"\n\t"path/filepath"\n\t"time"\n\n\t"vec-diputacion-granada/internal/app/bootstrap"\n\tidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"\n)\n\ntype config struct {\n\tDirectorioMaterial     string `json:"directorio_material"`\n\tRutaConfiguracionHMAC  string `json:"ruta_configuracion_hmac"`\n\tEspacioIdentidad       string `json:"espacio_identidad"`\n\tDominioRef             string `json:"dominio_ref"`\n\tEspacioClave           string `json:"espacio_clave"`\n\tDominioHMAC            string `json:"dominio_hmac"`\n\tIncluirCuentaOrdinaria bool   `json:"incluir_cuenta_ordinaria"`\n}\ntype original struct {\n\tNombre                     string `json:"nombre"`\n\tPersonaRef                 string `json:"persona_ref"`\n\tCuentaOrdinariaOriginal    string `json:"cuenta_ordinaria_original"`\n\tCuentaPrivilegiadaOriginal string `json:"cuenta_privilegiada_original"`\n\tSujetoOriginal             string `json:"sujeto_original"`\n}\ntype originales struct {\n\tEntorno         string     `json:"entorno"`\n\tAlcance         string     `json:"alcance"`\n\tOrganizacionRef string     `json:"organizacion_ref"`\n\tPersonas        []original `json:"personas"`\n}\ntype personaMaterial struct {\n\tPersonaRef         string `json:"persona_ref"`\n\tCuentaOrdinaria    string `json:"cuenta_ordinaria_id_hmac_hex"`\n\tCuentaPrivilegiada string `json:"cuenta_privilegiada_id_hmac_hex"`\n\tSujeto             string `json:"sujeto_id_hmac_hex"`\n}\ntype material struct {\n\tVersion       int               `json:"version"`\n\tFuenteRef     string            `json:"fuente_ref"`\n\tFuenteVersion int               `json:"fuente_version"`\n\tEsquema       string            `json:"esquema_hmac"`\n\tDominioRef    string            `json:"dominio_hmac_ref"`\n\tClaveID       string            `json:"clave_hmac_id"`\n\tClaveVersion  uint64            `json:"clave_hmac_version"`\n\tPersonas      []personaMaterial `json:"personas"`\n}\n\nfunc read(path string, max int64, out any) error {\n\tif !filepath.IsAbs(path) {\n\t\treturn errors.New("ruta no absoluta")\n\t}\n\tst, err := os.Lstat(path)\n\tif err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() < 1 || st.Size() > max {\n\t\treturn errors.New("archivo privado inválido")\n\t}\n\tf, err := os.Open(path)\n\tif err != nil {\n\t\treturn err\n\t}\n\tdefer f.Close()\n\tb, err := io.ReadAll(io.LimitReader(f, max+1))\n\tif err != nil || int64(len(b)) != st.Size() {\n\t\treturn errors.New("lectura privada inválida")\n\t}\n\td := json.NewDecoder(bytesReader(b))\n\td.DisallowUnknownFields()\n\tif err = d.Decode(out); err != nil {\n\t\treturn errors.New("JSON inválido")\n\t}\n\tvar extra any\n\tif d.Decode(&extra) != io.EOF {\n\t\treturn errors.New("JSON adicional")\n\t}\n\treturn nil\n}\n\n// bytesReader evita conservar otra copia del documento con identificadores.\ntype byteReader struct {\n\tb []byte\n\ti int\n}\n\nfunc (r *byteReader) Read(p []byte) (int, error) {\n\tif r.i >= len(r.b) {\n\t\treturn 0, io.EOF\n\t}\n\tn := copy(p, r.b[r.i:])\n\tr.i += n\n\treturn n, nil\n}\nfunc bytesReader(b []byte) io.Reader { return &byteReader{b: b} }\nfunc randomRef() (string, error) {\n\tvar b [16]byte\n\tif _, err := rand.Read(b[:]); err != nil {\n\t\treturn "", err\n\t}\n\treturn "prc_" + hex.EncodeToString(b[:]), nil\n}\nfunc main() {\n\tos.Exit(run())\n}\nfunc run() int {\n\tvar cfgPath, origPath, outPath string\n\tflag.StringVar(&cfgPath, "cfg", "", "")\n\tflag.StringVar(&origPath, "originales", "", "")\n\tflag.StringVar(&outPath, "salida", "", "")\n\tflag.Parse()\n\tif cfgPath == "" || origPath == "" || outPath == "" || flag.NArg() != 0 {\n\t\tfmt.Fprintln(os.Stderr, "argumentos inválidos")\n\t\treturn 1\n\t}\n\tvar c config\n\tvar o originales\n\tif read(cfgPath, 65536, &c) != nil || read(origPath, 65536, &o) != nil || len(o.Personas) != 2 || o.Entorno != "desarrollo" || o.Alcance != "sintetico_declarado" || c.IncluirCuentaOrdinaria {\n\t\tfmt.Fprintln(os.Stderr, "entrada privada inválida")\n\t\treturn 1\n\t}\n\tif !filepath.IsAbs(outPath) || filepath.Clean(outPath) != outPath {\n\t\tfmt.Fprintln(os.Stderr, "salida inválida")\n\t\treturn 1\n\t}\n\tprovider, closeProvider, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(bootstrap.ConfiguracionSeudonimosSesionPrivada{\n\t\tDirectorioMaterial: c.DirectorioMaterial, RutaConfiguracionHMAC: c.RutaConfiguracionHMAC,\n\t\tEspacioIdentidad: c.EspacioIdentidad, DominioRef: c.DominioRef, EspacioClave: c.EspacioClave,\n\t\tDominioHMAC: c.DominioHMAC, IncluirCuentaOrdinaria: false,\n\t})\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, "proveedor no disponible")\n\t\treturn 1\n\t}\n\tdefer closeProvider()\n\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)\n\tdefer cancel()\n\tresult := material{Version: 1, FuenteVersion: 1, Personas: make([]personaMaterial, 0, 2)}\n\tseen := map[string]bool{}\n\tfor _, p := range o.Personas {\n\t\tif p.PersonaRef == "" || p.CuentaOrdinariaOriginal == "" || p.CuentaPrivilegiadaOriginal == "" || p.SujetoOriginal == "" || seen[p.PersonaRef] {\n\t\t\tfmt.Fprintln(os.Stderr, "original inválido")\n\t\t\treturn 1\n\t\t}\n\t\tseen[p.PersonaRef] = true\n\t\tids := identidad.IdentificadoresAlta{EspacioIdentidad: c.EspacioIdentidad, AsercionID: p.SujetoOriginal, SesionID: p.SujetoOriginal, SujetoID: p.SujetoOriginal, CuentaID: p.CuentaOrdinariaOriginal}\n\t\tordinary, e := provider.SeudonimizarAlta(ctx, ids)\n\t\tif e != nil {\n\t\t\tfmt.Fprintln(os.Stderr, "seudonimización rechazada")\n\t\t\treturn 1\n\t\t}\n\t\tids.CuentaID = p.CuentaPrivilegiadaOriginal\n\t\tprivileged, e := provider.SeudonimizarAlta(ctx, ids)\n\t\tif e != nil {\n\t\t\tfmt.Fprintln(os.Stderr, "seudonimización rechazada")\n\t\t\treturn 1\n\t\t}\n\t\tif ordinary.Esquema != identidad.EsquemaHMACSHA256V1 || ordinary.Esquema != privileged.Esquema || ordinary.DominioRef != privileged.DominioRef || ordinary.ClaveID != privileged.ClaveID || ordinary.ClaveVersion != privileged.ClaveVersion || ordinary.SujetoIDHMAC != privileged.SujetoIDHMAC || ordinary.CuentaIDHMAC == privileged.CuentaIDHMAC || ordinary.SujetoIDHMAC == [32]byte{} || ordinary.CuentaIDHMAC == [32]byte{} || privileged.CuentaIDHMAC == [32]byte{} {\n\t\t\tfmt.Fprintln(os.Stderr, "coordenadas divergentes")\n\t\t\treturn 1\n\t\t}\n\t\tif result.Esquema == "" {\n\t\t\tresult.Esquema = ordinary.Esquema\n\t\t\tresult.DominioRef = ordinary.DominioRef\n\t\t\tresult.ClaveID = ordinary.ClaveID\n\t\t\tresult.ClaveVersion = ordinary.ClaveVersion\n\t\t} else if result.Esquema != ordinary.Esquema || result.DominioRef != ordinary.DominioRef || result.ClaveID != ordinary.ClaveID || result.ClaveVersion != ordinary.ClaveVersion {\n\t\t\tfmt.Fprintln(os.Stderr, "coordenadas distintas")\n\t\t\treturn 1\n\t\t}\n\t\tresult.Personas = append(result.Personas, personaMaterial{PersonaRef: p.PersonaRef, CuentaOrdinaria: hex.EncodeToString(ordinary.CuentaIDHMAC[:]), CuentaPrivilegiada: hex.EncodeToString(privileged.CuentaIDHMAC[:]), Sujeto: hex.EncodeToString(ordinary.SujetoIDHMAC[:])})\n\t}\n\tresult.FuenteRef, err = randomRef()\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, "referencia no disponible")\n\t\treturn 1\n\t}\n\tdata, err := json.Marshal(result)\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, "material no serializable")\n\t\treturn 1\n\t}\n\tf, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)\n\tif err != nil {\n\t\tfmt.Fprintln(os.Stderr, "salida no disponible")\n\t\treturn 1\n\t}\n\tdefer f.Close()\n\tif _, err = f.Write(data); err != nil || f.Sync() != nil {\n\t\tfmt.Fprintln(os.Stderr, "salida fallida")\n\t\treturn 1\n\t}\n\tfmt.Fprintln(os.Stdout, "Material HMAC oficial preparado en archivo privado")\n\treturn 0\n}\n'

class Paro(Exception):
    def __init__(self,clave,actual='no_observado',esperado='coincidente'):
        self.clave=clave;self.actual=actual;self.esperado=esperado;super().__init__(clave)

def parar(clave,actual='no_observado',esperado='coincidente'):
    if not all(re.fullmatch(r'[a-z0-9._-]{1,80}',v) for v in (clave,actual,esperado)):raise Paro('diagnostico_inseguro')
    raise Paro(clave,actual,esperado)
def sha(b): return hashlib.sha256(b).hexdigest()
def canon(x):
    texto=json.dumps(x,ensure_ascii=False,allow_nan=False,separators=(',',':'))
    for visible,escape in (('&',r'\u0026'),('<',r'\u003c'),('>',r'\u003e'),('\u2028',r'\u2028'),('\u2029',r'\u2029')):texto=texto.replace(visible,escape)
    return texto.encode()
def ref(pref): return pref+secrets.token_hex(16)
def fmt(t): return t.strftime('%Y-%m-%dT%H:%M:%SZ')
def parse(t): return dt.datetime.strptime(t,'%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=dt.timezone.utc)
def instante_fuente(t):
    if not isinstance(t,str):parar('vigencia_fuente_invalida')
    try:instante=dt.datetime.fromisoformat(t)
    except ValueError:parar('vigencia_fuente_invalida')
    if instante.tzinfo is None:parar('vigencia_fuente_invalida')
    return instante.astimezone(dt.timezone.utc)
def fecha_unidad_utc(t):
    if not isinstance(t,str) or not re.fullmatch(r'\d{4}-\d{2}-\d{2}',t):parar('vigencia_unidad_invalida')
    try:fecha=dt.date.fromisoformat(t)
    except ValueError:parar('vigencia_unidad_invalida')
    return dt.datetime.combine(fecha,dt.time.min,tzinfo=dt.timezone.utc)
def q(s): return "'"+str(s).replace("'","''")+"'"
def now(): return dt.datetime.now(dt.timezone.utc).replace(microsecond=0)

def privado(p,max_bytes=65536):
    p=Path(p)
    if not p.is_absolute() or p.is_symlink():parar('archivo_ruta_insegura')
    s=p.lstat()
    if not stat.S_ISREG(s.st_mode) or stat.S_IMODE(s.st_mode)!=0o600 or s.st_uid!=os.getuid() or not 0<s.st_size<=max_bytes:parar('archivo_privado_inseguro')
    return p.read_bytes()
def crear(p,b):
    p=Path(p)
    if not p.is_absolute() or p.exists() or p.is_symlink():parar('salida_existente_o_insegura')
    fd=os.open(p,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
    with os.fdopen(fd,'wb') as f:f.write(b);f.flush();os.fsync(f.fileno())
def raiz_privada(p,create=False):
    p=Path(p)
    if not p.is_absolute():parar('raiz_no_absoluta')
    if create:p.mkdir(parents=True,mode=0o700,exist_ok=True)
    s=p.lstat()
    if not stat.S_ISDIR(s.st_mode) or p.is_symlink() or stat.S_IMODE(s.st_mode)!=0o700 or s.st_uid!=os.getuid():parar('raiz_privada_insegura')
    return p

def exigir_hash(v,clave):
    if not isinstance(v,str) or not re.fullmatch(r'[0-9a-f]{64}',v) or v=='0'*64:parar(clave)
    return v

def exigir_vigencia_autoridad(valor,en_dias=False):
    unidad=1 if en_dias else 86400
    if type(valor) is not int or not 60*unidad<=valor<=90*unidad:
        parar('vigencia_autoridad_fuera_de_60_90_dias')

def objeto_sin_duplicados(pares):
    doc={}
    for clave,valor in pares:
        if clave in doc:parar('json_clave_duplicada')
        doc[clave]=valor
    return doc

def json_privado(p):
    try:return json.loads(privado(p),object_pairs_hook=objeto_sin_duplicados)
    except (ValueError,UnicodeDecodeError):parar('json_privado_invalido')

def verifica_codigo(c):
    kit=c['kit'];snapshot=Path(kit['source_snapshot']);copy=Path(kit['source_copy'])
    if not snapshot.is_absolute() or not copy.is_absolute() or snapshot==copy or (snapshot/'.git').exists() or (copy/'.git').exists():parar('kit_fuente_insegura')
    for path in (snapshot,copy):
        st=path.lstat()
        if not stat.S_ISDIR(st.st_mode) or stat.S_IMODE(st.st_mode)!=0o700 or st.st_uid!=os.getuid() or path.is_symlink():parar('kit_copia_privada_insegura')
    if not copy.is_relative_to(Path(c['privado'])):parar('kit_copia_privada_insegura')
    cache=Path(kit['modcache']);buildcache=Path(kit['gocache_cache'])
    if buildcache != Path('/dev/shm/vec/go-build') or buildcache.resolve() != buildcache:
        parar('gocache_fuera_de_memoria')
    estado_cache=buildcache.lstat()
    if not stat.S_ISDIR(estado_cache.st_mode) or buildcache.is_symlink() or estado_cache.st_uid!=os.getuid() or stat.S_IMODE(estado_cache.st_mode)&0o002:
        parar('gocache_privado_inseguro')
    if not cache.is_absolute() or cache!=Path.home()/'go/pkg/mod' or cache.is_symlink():parar('modcache_inseguro')
    manifest=Path(kit['source_manifest']);exigir_hash(kit['source_manifest_sha256'],'manifiesto_sha_invalido')
    raw=privado(manifest,2_000_000)
    if sha(raw)!=kit['source_manifest_sha256']:parar('manifiesto_divergente')
    doc=json.loads(raw)
    if doc.get('version')!=1 or not isinstance(doc.get('files'),dict) or len(doc['files'])<1000:parar('manifiesto_invalido')
    expected=set(doc['files'])
    for root in (snapshot,copy):
        actual={str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()}
        if actual!=expected:parar('kit_copia_archivos_divergentes')
        for rel,h in doc['files'].items():
            if rel.startswith('/') or '..' in Path(rel).parts:parar('manifiesto_ruta_insegura')
            exigir_hash(h,'manifiesto_huella_invalida')
            if sha((root/rel).read_bytes())!=h:parar('kit_copia_fuente_divergente')
    for kind in ('fuentes_preparar','fuentes_aplicar','unidad','bootstrap_preparar','bootstrap_aplicar'):
        for group in ('cli','catalogos'):
            path=Path(kit[group][kind]);st=path.lstat()
            if not path.is_absolute() or path.is_symlink() or not stat.S_ISREG(st.st_mode):parar('kit_artefacto_ausente')
            hash_key='cli_sha256' if group=='cli' else 'catalogo_sha256'
            exigir_hash(kit[hash_key][kind],'kit_huella_ausente')
            if sha(path.read_bytes())!=kit[hash_key][kind]:parar('kit_artefacto_divergente')
    for binary in ('go','openssl'):
        if not Path(kit[binary]).is_absolute():parar('kit_herramienta_no_absoluta')
    helper=Path(kit['helper_hmac']);exigir_hash(kit['helper_hmac_sha256'],'helper_huella_ausente')
    if not helper.is_absolute() or helper.is_symlink() or sha(helper.read_bytes())!=kit['helper_hmac_sha256']:parar('helper_kit_divergente')

def cargar_config(path):
    c=json_privado(path)
    for k in ('version','entorno','alcance_fuente','destino','backup','kit','privado','psql_comando','proveedor','fuentes','unidad','bootstrap','dba'):
        if k not in c:parar('config_campo_ausente')
    if c['version']!=1 or c['entorno']!='desarrollo' or c['alcance_fuente']!='sintetico_declarado' or c['destino']!='principal_desarrollo':parar('fuente_entorno_incompatible')
    if not c['backup'].get('ref') or not c['backup'].get('ref').strip():parar('backup_ref_ausente')
    exigir_hash(c['backup'].get('sha256'),'backup_sha_ausente')
    c['privado']=str(raiz_privada(c['privado'],True))
    cmd=c['psql_comando']
    if not isinstance(cmd,list) or not cmd or not all(isinstance(x,str) and x for x in cmd) or not Path(cmd[0]).is_absolute() or Path(cmd[0]).name not in ('psql','docker') or not any(Path(x).name=='psql' for x in cmd) or '-X' not in cmd or not any('ON_ERROR_STOP=1' in x for x in cmd) or any(x in ('-c','--command') or x.startswith('--command=') or any(bad in x.lower() for bad in ('postgres://','password=',';')) for x in cmd):parar('psql_vector_inseguro')
    if Path(cmd[0]).name=='docker' and (len(cmd)<4 or cmd[1]!='exec' or '-i' not in cmd):parar('psql_vector_inseguro')
    for phase in ('fuentes','unidad','bootstrap'):
        d=c['dba'][phase]
        for k in ('login','aprobacion_ref','conexion','aprobacion_externa'):
            if k not in d:parar('dba_campo_ausente')
        if not re.fullmatch(r'[a-z][a-z0-9_]{5,63}',d['login']) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._:-]{0,127}',d['aprobacion_ref']):parar('dba_identificador_invalido')
    if len({c['dba'][x]['login'] for x in ('fuentes','unidad','bootstrap')})!=3:parar('dba_login_repetido')
    for k in ('proceso','motivo_ref'):
        if k not in c['dba']['fuentes']:parar('dba_fuentes_incompleto')
    if 'proceso' not in c['dba']['bootstrap']:parar('dba_bootstrap_incompleto')
    names=c['fuentes']['nombres']
    if not isinstance(names,list) or len(names)!=2 or not all(isinstance(x,str) and 8<=len(x)<=100 and x==x.strip() and re.fullmatch(r'[A-Za-zÁÉÍÓÚÜÑáéíóúüñ ]+',x) for x in names) or names[0]==names[1]:parar('nombres_sinteticos_invalidos')
    for campo in ('ttl_personas_segundos','ttl_organizacion_segundos'):
        exigir_vigencia_autoridad(c['fuentes'][campo])
    exigir_vigencia_autoridad(c['unidad']['vigencia_dias'],en_dias=True)
    exigir_vigencia_autoridad(c['bootstrap']['ttl_perfiles_segundos'])
    for k in ('ttl_plan_segundos','ttl_personas_segundos','ttl_organizacion_segundos','ca_dias'):
        if not isinstance(c['fuentes'][k],int) or c['fuentes'][k]<=0:parar('fuentes_ventana_invalida')
    if not c['fuentes']['ttl_plan_segundos']<c['fuentes']['ttl_personas_segundos']<=c['fuentes']['ttl_organizacion_segundos']:parar('fuentes_ventana_invalida')
    if not isinstance(c['fuentes']['maxima_edad_revocacion_segundos'],int) or not 1<=c['fuentes']['maxima_edad_revocacion_segundos']<=2147483647:parar('revocacion_fuente_invalida')
    if not isinstance(c['unidad']['ttl_plan_segundos'],int) or not isinstance(c['unidad']['vigencia_dias'],int) or c['unidad']['ttl_plan_segundos']<=0 or c['unidad']['vigencia_dias']<=0 or c['unidad']['clase'] not in ('centro','delegacion','puesto_responsabilidad') or not (c['unidad'].get('unidad_ref') is None or (isinstance(c['unidad']['unidad_ref'],str) and re.fullmatch(r'unidad:[a-z0-9]+(?::[a-z0-9_-]+){1,4}',c['unidad']['unidad_ref']))):parar('unidad_config_invalida')
    if not all(isinstance(c['bootstrap'][k],int) and c['bootstrap'][k]>0 for k in ('ttl_plan_segundos','ttl_perfiles_segundos','duracion_propuesta_segundos')) or c['bootstrap']['ttl_plan_segundos']>=c['bootstrap']['ttl_perfiles_segundos'] or c['bootstrap']['duracion_propuesta_segundos']>c['bootstrap']['ttl_plan_segundos']:parar('bootstrap_config_invalida')
    if c['proveedor'].get('incluir_cuenta_ordinaria') is not False:parar('proveedor_finalidad_incompatible')
    if not all(isinstance(c['bootstrap'].get(k),str) and c['bootstrap'][k].startswith('rol:') for k in ('app_role_ref','sys_role_ref')):parar('bootstrap_roles_no_declarados')
    return c

def sql(c,statement,timeout=30):
    env={'PATH':'/usr/bin:/bin','HOME':c['privado'],'LANG':'C'}
    for k,v in c.get('psql_env',{}).items():
        if k not in ('PGPASSFILE','PGSERVICEFILE','PGSYSCONFDIR','PGSSLROOTCERT') or not Path(v).is_absolute():parar('psql_env_inseguro')
        env[k]=v
    r=subprocess.run(c['psql_comando'],input=statement,text=True,capture_output=True,timeout=timeout,env=env)
    if r.returncode:parar('sql_rechazado')
    return r.stdout

def fila_json(s,clave):
    a=[v for v in s.splitlines() if v.startswith('{') or v.startswith('[')]
    if len(a)!=1:parar(clave)
    try:return json.loads(a[0])
    except ValueError:parar(clave)
def fila_hash(s,clave):
    a=re.findall(r'(?m)^([0-9a-f]{64})$',s)
    if len(a)!=1:parar(clave)
    return a[0]
def getter_hash(c,expr):
    s="BEGIN ISOLATION LEVEL SERIALIZABLE;SET LOCAL timezone='UTC';SELECT encode(sha256(convert_to(("+expr+")::text,'UTF8')),'hex');ROLLBACK;"
    return fila_hash(sql(c,s),'getter_preimagen_invalido')

def comprobar(c,vacio=False):
    verifica_codigo(c)
    stmt="""SELECT jsonb_build_object('pg18',current_setting('server_version_num')::int BETWEEN 180000 AND 189999,
      'aut39',to_regprocedure('vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(text,text)') IS NOT NULL,
      'aut40',to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)') IS NOT NULL,
      'personal33',to_regprocedure('vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text)') IS NOT NULL,
      'fuentes_aut',(SELECT count(*) FROM vec_autorizacion.fuentes_iniciales_admin_v1),
      'fuentes_is',(SELECT count(*) FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1),
      'fuentes_ca',(SELECT count(*) FROM vec_contexto_actor_v1.fuentes_iniciales_admin_v1),
      'politicas',(SELECT count(*) FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1),
      'control',(SELECT jsonb_build_object('estado',bootstrap_estado,'revision',revision,'acto_nulo',bootstrap_acto_ref IS NULL) FROM vec_autorizacion.control_continuidad_admin WHERE control_id),
      'efectivos',jsonb_array_length(vec_autorizacion.administradores_aplicacion_efectivos_internos_v3()));"""
    d=fila_json(sql(c,stmt),'preestado_no_disponible')
    if not all(d[k] for k in ('pg18','aut39','aut40','personal33')):parar('dependencia_sql_ausente')
    if vacio and (any(d[k]!=0 for k in ('fuentes_aut','fuentes_is','fuentes_ca','politicas','efectivos')) or d['control']!={'estado':'pendiente','revision':1,'acto_nulo':True}):parar('preestado_no_vacio')
    return d

def carpeta(c,phase):return raiz_privada(Path(c['privado'])/phase,True)
def cli(c,name,args,acuse=None):
    if acuse is not None and Path(acuse).exists():parar('acuse_destino_ocupado')
    cmd=[c['kit']['cli'][name],*map(str,args)]
    r=subprocess.run(cmd,capture_output=True,text=True,timeout=60,env={'PATH':'/usr/bin:/bin','HOME':c['privado'],'LANG':'C'})
    # El diagnóstico puede describir el estado, nunca se imprime su cuerpo.
    if r.returncode!=0:parar('cli_'+name+'_codigo_'+str(r.returncode))
    if acuse is not None:
        d=json_privado(acuse)
        if d.get('estado')!='permitido' or not d.get('recibo'):parar('cli_respuesta_no_permitida')
        return d
    return None

def aprobacion(c,phase,plan_hash):
    d=c['dba'][phase];raw=privado(d['aprobacion_externa'])
    try:a=json.loads(raw,object_pairs_hook=objeto_sin_duplicados)
    except ValueError:parar('aprobacion_json_invalida')
    if set(a)!={'huella_plan_sha256','aprobacion_ref','backup_ref'} or a['huella_plan_sha256']!=plan_hash or a['aprobacion_ref']!=d['aprobacion_ref'] or a['backup_ref']!=c['backup']['ref']:parar('aprobacion_externa_divergente')
    return sha(raw)

def valor_sql(campo,valor):
    if campo in ('vigente_desde','vigente_hasta'):return q(valor)+'::timestamptz'
    if campo in ('fuente_reparto','fuente_identidad','fuente_ca_admin'):return q(valor)+'::jsonb'
    return q(valor)

def consulta_config(c,doc):
    phase=doc['phase'];name=doc['fields']['login_nombre'];group=doc['group'];table=doc['table']
    condiciones=[]
    for col,val in doc['fields'].items():
        campo='cfg.'+col+'::text' if col=='login_nombre' else 'cfg.'+col
        condiciones.append(campo+' IS NOT DISTINCT FROM '+valor_sql(col,val))
    campos=' AND '.join(condiciones)
    query="SELECT jsonb_build_object('rol',(SELECT count(*) FROM pg_roles WHERE rolname="+q(name)+"),'rol_ok',EXISTS(SELECT 1 FROM pg_roles l WHERE l.rolname="+q(name)+" AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls AND l.rolconfig IS NULL AND (SELECT count(*)=1 AND bool_and(roleid="+q(group)+"::regrole AND inherit_option AND NOT set_option AND NOT admin_option) FROM pg_auth_members WHERE member=l.oid)),'cfg',(SELECT count(*) FROM "+table+" WHERE login_nombre="+q(name)+"),'cfg_ok',EXISTS(SELECT 1 FROM "+table+" cfg WHERE "+campos+"));"
    return fila_json(sql(c,query),'config_estado_no_disponible')

def compromiso_config(c,phase,F,table,campos):
    group={'fuentes':'vec_admin_fuentes_iniciales_ejecutor','unidad':'vec_personal_unidad_inicial_ejecutor','bootstrap':'vec_admin_bootstrap_central_v3_ejecutor'}[phase]
    path=F/'config-esperada.json'
    if path.exists():
        previo=json_privado(path)
        if previo.get('phase')!=phase or previo.get('table')!=table or previo.get('group')!=group or set(previo.get('fields',{}))!=set(campos)|{'vigente_desde'}:parar('config_compromiso_divergente')
        campos['vigente_desde']=previo['fields']['vigente_desde']
        if previo['fields']!=campos:parar('config_compromiso_divergente')
        return previo
    campos['vigente_desde']=fmt(now()-dt.timedelta(minutes=1))
    doc={'phase':phase,'table':table,'group':group,'fields':campos}
    crear(path,canon(doc))
    return doc

def configurar_login(c,phase,F,table,campos):
    doc=compromiso_config(c,phase,F,table,campos)
    actual=consulta_config(c,doc)
    if actual['rol']==0 and actual['cfg']==0:
        name=doc['fields']['login_nombre'];group=doc['group']
        columnas=','.join(doc['fields']);valores=','.join(valor_sql(k,v) for k,v in doc['fields'].items())
        insertar='INSERT INTO '+table+'('+columnas+') VALUES('+valores+')'
        stmt="BEGIN;DO $guard$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname="+q(name)+") THEN RAISE EXCEPTION 'H10 LOGIN existente' USING ERRCODE='55000';END IF;END $guard$;CREATE ROLE "+name+" LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;GRANT "+group+' TO '+name+" WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;"+insertar+';COMMIT;'
        sql(c,stmt)
        actual=consulta_config(c,doc)
    if actual!={'rol':1,'rol_ok':True,'cfg':1,'cfg_ok':True}:parar('config_confirmada_divergente',actual='no_coincidente',esperado='login_y_fila_exactos')
    crear_o_cotejar(F/'config-confirmada.json',canon({'compromiso_sha256':sha(canon(doc))}))

def exigir_config_confirmada(c,phase,F):
    doc=json_privado(F/'config-esperada.json');mark=json_privado(F/'config-confirmada.json')
    if doc.get('phase')!=phase or mark!={'compromiso_sha256':sha(canon(doc))}:parar('config_marcador_divergente')
    actual=consulta_config(c,doc)
    if actual!={'rol':1,'rol_ok':True,'cfg':1,'cfg_ok':True}:parar('config_live_divergente',actual='no_coincidente',esperado='login_y_fila_exactos')
def estado(phase,**data):
    print(json.dumps({'etapa':phase,**data},separators=(',',':')))

def openssl(c,F,*args):
    r=subprocess.run([c['kit']['openssl'],*map(str,args)],cwd=F,env={'PATH':'/usr/bin:/bin','HOME':str(F),'LANG':'C'},stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=30)
    if r.returncode:parar('certificado_dev_rechazado')

def hmac_proveedor(c,F,originales):
    copy=Path(c['kit']['source_copy']);cmdroot=copy/'cmd'
    if not cmdroot.is_dir() or (copy/'.git').exists():parar('copia_go_no_privada')
    # La implementación ADMIN/seudonimización entre 345 y 298 es idéntica.
    # Reutilizamos el helper compilado y cotejado del kit H, sin compilar.
    kit_helper=Path(c['kit']['helper_hmac'])
    if sha(kit_helper.read_bytes())!=c['kit']['helper_hmac_sha256']:parar('helper_kit_divergente')
    binary=F/'helper-hmac'
    shutil.copyfile(kit_helper,binary)
    binary.chmod(0o700)
    cfg=F/'proveedor.json';crear(cfg,canon(c['proveedor']))
    material=F/'material-hmac.raw'
    run=subprocess.run([str(binary),'-cfg',str(cfg),'-originales',str(originales),'-salida',str(material)],cwd=F,env={'PATH':'/usr/bin:/bin','HOME':str(F)},stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=30)
    if run.returncode or not material.is_file():parar('hmac_proveedor_rechazado')
    material_raw=privado(material,32768);m=json.loads(material_raw)
    if not re.fullmatch(r'prc_[0-9a-f]{32}',m.get('fuente_ref','')) or m.get('esquema_hmac')!='vec.identidad.hmac-sha256.v1' or len(m.get('personas',[]))!=2:parar('hmac_material_invalido')
    binary.unlink()
    return m,material_raw

def preparar_fuentes(c):
    comprobar(c,vacio=True)
    f=c['fuentes'];policy_raw=privado(f['politica_aprobacion'])
    F=carpeta(c,'fuentes');momento=now();limite=momento+dt.timedelta(seconds=f['ttl_plan_segundos'])
    if f['ca_dias']*86400<=f['ttl_personas_segundos']:parar('ca_vigencia_insuficiente')
    org=ref('org_');people=[]
    for name in f['nombres']:
        people.append({'nombre':name,'persona_ref':ref('per_'),'cuenta_ordinaria_original':'devord_'+secrets.token_hex(16),'cuenta_privilegiada_original':'devadm_'+secrets.token_hex(16),'sujeto_original':'devsuj_'+secrets.token_hex(16)})
    people.sort(key=lambda x:x['persona_ref'])
    originals={'entorno':'desarrollo','alcance':'sintetico_declarado','organizacion_ref':org,'personas':people}
    crear(F/'originales.json',canon(originals)+b'\n')
    provenance={'esquema':'vec.h10.procedencia.sintetica.v1','entorno':'desarrollo','backup_ref':c['backup']['ref'],'organizacion_ref':org,'personas':[{'nombre':p['nombre'],'persona_ref':p['persona_ref']} for p in people],'preparado_en':fmt(momento),'caduca_en':fmt(limite)}
    provenance_raw=canon(provenance)+b'\n';crear(F/'procedencia.json',provenance_raw)
    titular=[]
    for i,p in enumerate(people,1):
        b=canon({'esquema':'vec.h10.titularidad.sintetica.v1','persona_ref':p['persona_ref'],'nombre_sintetico':p['nombre'],'cuenta_ordinaria_original':p['cuenta_ordinaria_original'],'cuenta_privilegiada_original':p['cuenta_privilegiada_original'],'sujeto_original':p['sujeto_original']})+b'\n'
        crear(F/f'titularidad-{i}.json',b);titular.append({'referencia':ref('prc_'),'version':1,'huella_sha256':sha(b)})
    crear(F/'ca.cnf',b'[req]\ndistinguished_name=dn\nx509_extensions=v3_ca\nprompt=no\n[dn]\nCN=VEC H10 DEV CA\n[v3_ca]\nbasicConstraints=critical,CA:TRUE,pathlen:0\nkeyUsage=critical,keyCertSign,cRLSign\nsubjectKeyIdentifier=hash\n')
    crear(F/'cliente.ext',b'[client]\nbasicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n')
    openssl(c,F,'genpkey','-algorithm','EC','-pkeyopt','ec_paramgen_curve:P-256','-out','ca.key')
    openssl(c,F,'req','-new','-x509','-sha256','-days',str(f['ca_dias']),'-key','ca.key','-config','ca.cnf','-extensions','v3_ca','-out','ca.pem')
    openssl(c,F,'x509','-in','ca.pem','-outform','DER','-out','ca.der')
    ca_sha=sha(privado(F/'ca.der'))
    certs=[]
    for i,p in enumerate(people,1):
        openssl(c,F,'genpkey','-algorithm','EC','-pkeyopt','ec_paramgen_curve:P-256','-out',f'cliente-{i}.key')
        openssl(c,F,'req','-new','-sha256','-key',f'cliente-{i}.key','-subj','/CN='+p['nombre']+'/O=VEC DEV Synthetic','-out',f'cliente-{i}.csr')
        openssl(c,F,'x509','-req','-sha256','-days',str(f['ca_dias']),'-in',f'cliente-{i}.csr','-CA','ca.pem','-CAkey','ca.key','-set_serial','0x'+secrets.token_hex(16),'-extfile','cliente.ext','-extensions','client','-out',f'cliente-{i}.pem')
        openssl(c,F,'x509','-in',f'cliente-{i}.pem','-outform','DER','-out',f'cliente-{i}.der')
        certs.append({'persona_ref':p['persona_ref'],'nombre_sintetico':p['nombre'],'cert_der_sha256':sha(privado(F/f'cliente-{i}.der')),'ca_der_sha256':ca_sha})
    crear(F/'certificados.json',canon({'entorno':'desarrollo','certificados':certs})+b'\n')
    m,material=hmac_proveedor(c,F,F/'originales.json')
    if [x['persona_ref'] for x in m['personas']]!=[x['persona_ref'] for x in people]:parar('hmac_personas_divergentes')
    policy={'politica_ref':ref('pga_'),'host_admin':f['host_admin'],'ca_sha256':ca_sha,'huella_aprobacion_sha256':sha(policy_raw),'maxima_edad_revocacion_segundos':f['maxima_edad_revocacion_segundos'],'vigente_hasta':fmt(momento+dt.timedelta(seconds=f['ttl_personas_segundos']))}
    source={'version':1,'operacion_ref':ref('pfi_'),'preparado_en':fmt(momento),'caduca_en':fmt(limite),'entorno':'desarrollo','alcance_fuente':'sintetico_declarado','procedencia':{'referencia':ref('prc_'),'version':1,'huella_sha256':sha(provenance_raw)},'organizacion':{'organizacion_ref':org,'version_esperada':0,'vigente_hasta':fmt(momento+dt.timedelta(seconds=f['ttl_organizacion_segundos']))},'personas':[{'persona_ref':p['persona_ref'],'version_esperada':0,'vigente_hasta':fmt(momento+dt.timedelta(seconds=f['ttl_personas_segundos'])),'operacion_cuenta_ordinaria_ref':ref('opr_'),'operacion_cuenta_privilegiada_ref':ref('opr_'),'fuente_titularidad':titular[i]} for i,p in enumerate(people)],'fuente_hmac':{'referencia':m['fuente_ref'],'version':1,'huella_sha256':sha(material)},'politica_admin':policy}
    crear(F/'fuente.json',canon(source)+b'\n')
    cli(c,'fuentes_preparar',['--fuente',F/'fuente.json','--plan',F/'plan.json','--textos',c['kit']['catalogos']['fuentes_preparar']])
    cli(c,'fuentes_preparar',['--fuente',F/'fuente.json','--plan',F/'plan.json','--textos',c['kit']['catalogos']['fuentes_preparar'],'--cotejar'])
    plan=json_privado(F/'plan.json');estado('preparar_fuentes',plan_sha256=plan['huella_plan_sha256'],siguiente='configurar_fuentes')

def crear_o_cotejar(p,b):
    p=Path(p)
    if p.exists():
        if privado(p)!=b:parar('artefacto_divergente')
    else:crear(p,b)

def preparar_cli_config(c,phase,F,plan_hash):
    conn=json_privado(c['dba'][phase]['conexion'])
    if set(conn)!={'dsn','permitir_socket_desarrollo'} or not isinstance(conn['dsn'],str) or not conn['dsn']:parar('conexion_privada_invalida')
    crear_o_cotejar(F/'aprobacion.json',canon({'huella_plan_sha256':plan_hash}))
    crear_o_cotejar(F/'conexion.json',canon(conn))

def plan_y_hash(F):
    d=json_privado(F/'plan.json')
    if set(d)!={'plan','huella_plan_sha256'}:parar('plan_formato_invalido')
    b=canon(d['plan'])
    if sha(b)!=d['huella_plan_sha256']:parar('plan_huella_divergente')
    if parse(d['plan']['caduca_en'])<=now():parar('plan_caducado')
    return d['plan'],b,d['huella_plan_sha256']

def plan_sin_caducidad(F,phase):
    documento=json_privado(F/('plan-v3.json' if phase=='bootstrap' else 'plan.json'))
    if set(documento)!={'plan','huella_plan_sha256'} or sha(canon(documento['plan']))!=documento['huella_plan_sha256']:parar('plan_huella_divergente')
    return documento['plan'],documento['huella_plan_sha256']

def recibo_ligado(F,phase,d):
    if d.get('estado')!='permitido' or not isinstance(d.get('recibo'),dict) or not isinstance(d.get('auditoria_intento'),dict):return False
    p,h=plan_sin_caducidad(F,phase);r=d['recibo']
    if not r.get('auditoria_ref'):return False
    if phase=='fuentes':
        ca=r.get('ca') or {};isr=r.get('is') or {};personas=(ca.get('datos') or {}).get('personas') or []
        return r.get('plan_sha256')==h and ca.get('plan_sha256')==h and isr.get('plan_sha256')==h and (ca.get('datos') or {}).get('organizacion_ref')==p['organizacion']['organizacion_ref'] and sorted(x.get('persona_ref') for x in personas)==sorted(x['persona_ref'] for x in p['personas']) and len(personas)==2
    if phase=='unidad':
        return r.get('plan_sha256')==h and (r.get('unidad') or {}).get('unidad_ref')==p['unidad']['unidad_ref'] and r.get('fuente_sha256')==p['fuente']['huella_sha256']
    if r.get('huella_plan_sha256')!=h:return False
    perfiles=r.get('perfiles') or []
    expected={(x['persona_ref'],x['cuenta_ref'],x['perfil_ref'],x['vinculo_ref'],p['rol']['version_ref']) for x in p['personas']}
    expected.update((x['persona_ref'],x['cuenta_ref'],y['perfil_ref'],y['vinculo_ref'],y['rol']['version_ref']) for x in p['personas'] for y in x['sistemas'])
    actual={(x.get('persona_ref'),x.get('cuenta_ref'),x.get('perfil_ref'),x.get('vinculo_ref'),x.get('rol_version_ref')) for x in perfiles}
    return len(perfiles)==3 and actual==expected

def confirmacion_vigente(F,phase):
    preferido=F/'acuse-primero.json';paths=[]
    if preferido.exists():paths.append(preferido)
    paths.extend(x for x in sorted(F.glob('acuse-intento-*.json')) if x!=preferido)
    validos=[]
    for path in paths:
        try:d=json_privado(path)
        except (Paro,OSError,ValueError):continue
        try:ligado=recibo_ligado(F,phase,d)
        except (KeyError,TypeError,AttributeError):ligado=False
        if ligado:validos.append((path,d))
    if not validos:parar('acuse_confirmado_ausente',actual='ninguno',esperado='permitido_ligado')
    primero=validos[0]
    for path,d in validos[1:]:
        if d['recibo']!=primero[1]['recibo'] or d['auditoria_intento']==primero[1]['auditoria_intento']:parar('acuse_confirmado_divergente',actual='distinto',esperado='mismo_recibo_auditoria_nueva')
    return primero

def nuevo_acuse(F):
    path=F/('acuse-intento-'+secrets.token_hex(16)+'.json')
    if path.exists():parar('acuse_destino_ocupado')
    return path

def configurar_fuentes(c):
    F=carpeta(c,'fuentes');plan,raw,h=plan_y_hash(F);material=privado(F/'material-hmac.raw',32768)
    if plan['fuente_hmac']['huella_sha256']!=sha(material):parar('material_hmac_divergente')
    approval_sha=aprobacion(c,'fuentes',h)
    pre=getter_hash(c,'vec_autorizacion.preimagen_orquestada_fuentes_admin_v1('+q(raw.decode())+'::jsonb,'+q(material.decode())+')')
    crear_o_cotejar(F/'preimagen.sha256',(pre+'\n').encode())
    preparar_cli_config(c,'fuentes',F,h)
    d=c['dba']['fuentes']
    campos={'login_nombre':d['login'],'proceso':d['proceso'],'motivo_ref':d['motivo_ref'],'entorno':'desarrollo','alcance_fuente':'sintetico_declarado','plan_sha256':h,'preimagen_sha256':pre,'aprobacion_ref':d['aprobacion_ref'],'aprobacion_sha256':approval_sha,'material_hmac_canonico':material.decode(),'vigente_hasta':plan['caduca_en']}
    configurar_login(c,'fuentes',F,'vec_autorizacion.config_fuentes_iniciales_admin_v1',campos)
    estado('configurar_fuentes',plan_sha256=h,siguiente='aplicar_fuentes')

def aplicar_fase(c,phase,replay=False):
    F=carpeta(c,phase);plan,h=plan_sin_caducidad(F,phase)
    if parse(plan['caduca_en'])<=now():parar('plan_caducado',actual='vencido',esperado='vigente')
    exigir_config_confirmada(c,phase,F)
    anterior=None
    if replay:
        try:anterior=confirmacion_vigente(F,phase)[1]
        except Paro as e:
            if e.clave!='acuse_confirmado_ausente':raise
    ack=nuevo_acuse(F) if replay else F/'acuse-primero.json'
    if phase=='fuentes':
        name='fuentes_aplicar';args=['--plan',F/'plan.json','--conexion',F/'conexion.json','--aprobacion',F/'aprobacion.json','--acuse',ack,'--textos',c['kit']['catalogos'][name],'--timeout','30s']
    elif phase=='unidad':
        name='unidad';args=['--material',F/'material.json','--fuente',F/'fuente.json','--plan',F/'plan.json','--textos',c['kit']['catalogos'][name],'--cotejar','--aplicar','--conexion',F/'conexion.json','--aprobacion',F/'aprobacion.json','--acuse',ack,'--timeout','30s']
    else:
        name='bootstrap_aplicar';args=['--plan',F/'plan-v3.json','--conexion',F/'conexion.json','--aprobacion',F/'aprobacion.json','--acuse',ack,'--textos',c['kit']['catalogos'][name],'--timeout','30s']
    d=cli(c,name,args,ack)
    if not recibo_ligado(F,phase,d):parar('recibo_no_ligado',actual='divergente',esperado='plan_y_efecto')
    if replay:
        if d.get('replay') is not True:parar('recuperacion_no_replay',actual='nuevo_efecto',esperado='replay_confirmado')
        if anterior is not None and (d['recibo']!=anterior['recibo'] or d['auditoria_intento']==anterior['auditoria_intento']):parar('replay_divergente',actual='distinto',esperado='recibo_igual_intento_nuevo')
    # El selector vuelve a cotejar todos los acuses válidos; jamás reemplaza el primero.
    confirmacion_vigente(F,phase)
    siguiente={'fuentes':'preparar_unidad','unidad':'preparar_bootstrap','bootstrap':'verificar'}[phase]
    estado(('recuperar_' if replay else 'aplicar_')+phase,permitido=True,replay=bool(replay),adopcion_sin_primero=bool(replay and anterior is None),siguiente=siguiente)

def aplicar_fuentes(c,replay=False):
    aplicar_fase(c,'fuentes',replay)

def preparar_unidad(c):
    src=carpeta(c,'fuentes');F=carpeta(c,'unidad');acuse=confirmacion_vigente(src,'fuentes')[1]
    if acuse.get('estado')!='permitido':parar('fuentes_no_confirmadas')
    u=c['unidad'];t=now();token=secrets.token_hex(16);until=t+dt.timedelta(seconds=u['ttl_plan_segundos'])
    if u['vigencia_dias']*86400<=u['ttl_plan_segundos']:parar('unidad_vigencia_insuficiente')
    org=acuse['recibo']['ca']['datos']['organizacion_ref']
    unit={'nodo_ref':str(uuid.uuid4()),'organizacion_ref':org,'unidad_ref':u.get('unidad_ref') or 'unidad:sintetica:h10:'+token,'clase':u['clase'],'denominacion':u['denominacion'],'catalogo_ref':u['catalogo_ref'],'catalogo_version':u['catalogo_version'],'catalogo_revision':u['catalogo_revision'],'catalogo_entrada_clave':u['catalogo_entrada_clave'],'revision_esperada':0,'vigente_desde':t.date().isoformat(),'vigente_hasta':(t+dt.timedelta(days=u['vigencia_dias'])).date().isoformat()}
    source={'version':1,'referencia':'fuente:unidad_admin_h10:'+token,'entorno':'desarrollo','alcance_fuente':'sintetico_declarado','unidad':unit,'acto_tecnico_ref':'acto_tecnico:unidad_admin_h10:'+token}
    source_raw=canon(source);crear(F/'fuente.json',source_raw)
    material={'version':1,'operacion_ref':ref('pui_'),'preparado_en':fmt(t),'caduca_en':fmt(until),'entorno':'desarrollo','alcance_fuente':'sintetico_declarado','unidad':unit,'acto_tecnico_ref':source['acto_tecnico_ref'],'fuente':{'referencia':source['referencia'],'version':1,'huella_sha256':sha(source_raw)}}
    crear(F/'material.json',canon(material));cli(c,'unidad',['--material',F/'material.json','--fuente',F/'fuente.json','--plan',F/'plan.json','--textos',c['kit']['catalogos']['unidad']])
    cli(c,'unidad',['--material',F/'material.json','--fuente',F/'fuente.json','--plan',F/'plan.json','--textos',c['kit']['catalogos']['unidad'],'--cotejar'])
    _,_,h=plan_y_hash(F);estado('preparar_unidad',plan_sha256=h,siguiente='configurar_unidad')

def configurar_unidad(c):
    F=carpeta(c,'unidad');plan,raw,h=plan_y_hash(F);source=privado(F/'fuente.json')
    if plan['fuente']['huella_sha256']!=sha(source):parar('unidad_fuente_divergente')
    approval_sha=aprobacion(c,'unidad',h)
    pre=getter_hash(c,'vec_personal.preimagen_unidad_inicial_admin_v1('+q(raw.decode())+'::jsonb)')
    crear_o_cotejar(F/'preimagen.sha256',(pre+'\n').encode())
    preparar_cli_config(c,'unidad',F,h)
    d=c['dba']['unidad']
    campos={'login_nombre':d['login'],'entorno':'desarrollo','alcance_fuente':'sintetico_declarado','plan_sha256':h,'fuente_sha256':sha(source),'preimagen_sha256':pre,'aprobacion_ref':d['aprobacion_ref'],'aprobacion_sha256':approval_sha,'vigente_hasta':plan['caduca_en']}
    configurar_login(c,'unidad',F,'vec_personal.config_unidad_inicial_admin_v1',campos)
    estado('configurar_unidad',plan_sha256=h,siguiente='aplicar_unidad')

def aplicar_unidad(c,replay=False):
    aplicar_fase(c,'unidad',replay)

def metadata_bootstrap(c):
    query="""SELECT jsonb_build_object(
 'roles',(SELECT jsonb_agg(jsonb_build_object('version_ref',r.version_rol_ref,'rol_id',r.rol_id,'huella',r.huella_sha256,'estado',r.documento->>'estado','control',(SELECT to_jsonb(c) FROM vec_autorizacion.control_vigencia_version_rol c JOIN vec_autorizacion.control_vigencia_version_rol_actual ac USING(version_rol_ref,revision) WHERE c.version_rol_ref=r.version_rol_ref),'categoria',(SELECT to_jsonb(m) FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 m WHERE m.version_rol_ref=r.version_rol_ref),'dimensiones',(SELECT jsonb_agg(d ORDER BY d) FROM(SELECT DISTINCT value d FROM vec_autorizacion.catalogo_accion_nominal_v1 x,LATERAL jsonb_array_elements_text(x.dimensiones_ambito) WHERE x.version_rol_ref=r.version_rol_ref) ds))) FROM vec_autorizacion.version_rol r WHERE r.rol_id IN('administracion_perfiles','operador_plataforma')),
 'control',(SELECT to_jsonb(c) FROM vec_autorizacion.control_continuidad_admin c WHERE control_id),
 'politica',(SELECT to_jsonb(p) FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 p),
 'fuente',(SELECT recibo FROM vec_autorizacion.fuentes_iniciales_admin_v1),
 'unidad',(SELECT recibo FROM vec_personal.unidad_inicial_admin_v1));"""
    return fila_json(sql(c,query),'metadata_bootstrap_invalida')

def vencimiento_bootstrap(t,b,fuente_plan,unidad_recibo,politica):
    # CA33 publica persona, proyecciones de cuenta y titularidad con la fecha
    # exacta del plan de fuentes; el recibo confirmado está ligado a su huella.
    limites=[t+dt.timedelta(seconds=b['ttl_perfiles_segundos']),
             instante_fuente(fuente_plan['organizacion']['vigente_hasta']),
             instante_fuente(fuente_plan['politica_admin']['vigente_hasta']),
             instante_fuente(politica['vigente_hasta']),
             fecha_unidad_utc(unidad_recibo['unidad']['vigente_hasta'])]
    limites.extend(instante_fuente(p['vigente_hasta']) for p in fuente_plan['personas'])
    hasta=min(limites)
    if hasta<=t+dt.timedelta(seconds=b['ttl_plan_segundos']) or hasta<t+dt.timedelta(days=60):
        parar('bootstrap_autoridad_agotada')
    return hasta

def preparar_bootstrap(c):
    F=carpeta(c,'bootstrap');fuentes=carpeta(c,'fuentes');unidad=carpeta(c,'unidad')
    source=json_privado(fuentes/'fuente.json');fr=confirmacion_vigente(fuentes,'fuentes')[1];ur=confirmacion_vigente(unidad,'unidad')[1]
    if fr.get('estado')!='permitido' or ur.get('estado')!='permitido':parar('antecedentes_no_confirmados')
    fuente_plan,fuente_huella=plan_sin_caducidad(fuentes,'fuentes')
    m=metadata_bootstrap(c)
    if m['control']['bootstrap_estado']!='pendiente' or m['control']['revision']!=1 or m['control']['bootstrap_acto_ref'] is not None or m['fuente']['recibo_ref']!=fr['recibo']['recibo_ref'] or m['unidad']['recibo_ref']!=ur['recibo']['recibo_ref'] or fuente_huella!=fr['recibo']['plan_sha256'] or fuente_huella!=m['fuente']['plan_sha256'] or m['unidad']['unidad']!=ur['recibo']['unidad']:parar('bootstrap_preimagen_divergente')
    roles=[x for x in m['roles'] if x['categoria'] and x['estado']=='publicada' and x['control'] and x['control']['estado']=='habilitada']
    app=[x for x in roles if x['categoria']['categoria_administrativa']=='aplicacion' and x['version_ref']==c['bootstrap']['app_role_ref']];sysrole=[x for x in roles if x['categoria']['categoria_administrativa']=='sistemas' and x['version_ref']==c['bootstrap']['sys_role_ref']]
    if len(app)!=1 or len(sysrole)!=1:parar('roles_administrativos_ambiguos')
    app,sysrole=app[0],sysrole[0]
    role=lambda x:{'version_ref':x['version_ref'],'huella_sha256':x['huella'],'control_revision':x['control']['revision'],'control_huella_sha256':x['control']['huella_sha256']}
    people=fr['recibo']['ca']['datos']['personas'];org=fr['recibo']['ca']['datos']['organizacion_ref'];unit=m['unidad']['unidad']['unidad_ref']
    if len(people)!=2 or [x['persona_ref'] for x in people]!=sorted(x['persona_ref'] for x in people) or [x['persona_ref'] for x in people]!=[x['persona_ref'] for x in fuente_plan['personas']] or fuente_plan['organizacion']['organizacion_ref']!=org or m['politica']['politica_ref']!=source['politica_admin']['politica_ref'] or m['politica']['politica_ref']!=fuente_plan['politica_admin']['politica_ref']:parar('fuentes_bootstrap_divergentes')
    t=now();b=c['bootstrap'];expiry=t+dt.timedelta(seconds=b['ttl_plan_segundos'])
    until=vencimiento_bootstrap(t,b,fuente_plan,m['unidad'],m['politica'])
    org_ev=source['procedencia'];unit_ev={'referencia':m['unidad']['fuente_ref'],'version':1,'huella_sha256':m['unidad']['fuente_sha256']}
    amb_org={'dimension':'organizacion_ref','valores':[org],'fuente':org_ev};amb_unit={'dimension':'unidad_ref','valores':[unit],'fuente':unit_ev}
    ca_sha=sha(privado(fuentes/'ca.der'));certs=json_privado(fuentes/'certificados.json')['certificados']
    if ca_sha!=source['politica_admin']['ca_sha256']:parar('ca_fuente_divergente')
    p={'version':3,'preparado_en':fmt(t),'caduca_en':fmt(expiry),'control_continuidad_revision_esperada':1,'bootstrap_estado_esperado':'pendiente','rol':role(app),'fuente_identidad':source['procedencia'],'fuente_ca_admin':{'referencia':ref('prc_'),'version':1,'huella_sha256':ca_sha},'personas':[]}
    for i,person in enumerate(people,1):
        cert=next((x for x in certs if x['persona_ref']==person['persona_ref']),None)
        if cert is None or cert['cert_der_sha256']!=sha(privado(fuentes/f'cliente-{i}.der')) or cert['ca_der_sha256']!=ca_sha:parar('certificado_persona_divergente')
        proof={'persona_ref':person['persona_ref'],'cuenta_ref':person['cuenta_privilegiada_ref'],'certificado_sha256_der':cert['cert_der_sha256'],'ca_sha256_der':ca_sha,'preparado_en':fmt(t),'alcance_fuente':'sintetico_declarado'}
        proof_raw=canon(proof);crear(F/f'acreditacion-certificado-{i}.json',proof_raw)
        pe={'cuenta_ref':person['cuenta_privilegiada_ref'],'cuenta_version':person['proyeccion_cuenta_version'],'persona_ref':person['persona_ref'],'persona_version':person['persona_version'],'perfil_ref':ref('prf_'),'vinculo_ref':ref('vca_'),'preimagen_huella_sha256':sha(b'pendiente getter propietario'),'procedencia':source['procedencia'],'vigente_hasta':fmt(until),'certificado_admin':{'persona_ref':person['persona_ref'],'cuenta_ref':person['cuenta_privilegiada_ref'],'huella_sha256':cert['cert_der_sha256'],'ca_huella_sha256':ca_sha,'acreditacion':{'referencia':ref('prc_'),'version':1,'huella_sha256':sha(proof_raw)}},'sistemas':[],'ambitos':[amb_org,amb_unit]}
        if i==1:pe['sistemas']=[{'rol':role(sysrole),'perfil_ref':ref('prf_'),'vinculo_ref':ref('vca_'),'vigente_hasta':fmt(until),'ambitos':[amb_org]}]
        p['personas'].append(pe)
    gov=[]
    for r in (app,sysrole):
        category=r['categoria'];isapp=category['categoria_administrativa']=='aplicacion';limits=[{'clave':'organizacion_ref','valores':[org]}]
        if isapp:limits.append({'clave':'unidad_ref','valores':[unit]})
        gov.append({'version_ref':r['version_ref'],'huella_sha256':r['huella'],'clase':'administrador','categoria_admin':category['categoria_administrativa'],'unidad_requerida':isapp,'ambitos_fijos':limits,'vigente_desde':fmt(t),'vigente_hasta':fmt(until),'duracion_propuesta_segundos':b['duracion_propuesta_segundos'],'fuente_categoria':{'referencia':category['fuente_ref'],'version':category['fuente_version'],'huella_sha256':category['fuente_huella_sha256']},'dimensiones_ambito':r['dimensiones'],'rol_id':r['rol_id']})
    p['gobierno']={'audiencia_selector_admin':b['audiencia_selector'],'audiencia_administrativa':b['audiencia_administrativa'],'politica_certificado_ref':m['politica']['politica_ref'],'politica_certificado_huella_sha256':m['politica']['huella_aprobacion_sha256'],'roles':sorted(gov,key=lambda x:x['version_ref']),'motivos':[]}
    reparto={'esquema':'vec.ensayo.reparto.admin.v1','version':1,'alcance_fuente':'sintetico_declarado','personas':[{'persona_ref':x['persona_ref'],'cuenta_ref':x['cuenta_ref'],'perfil_aplicacion':x['perfil_ref'],'perfil_sistemas':[y['perfil_ref'] for y in x['sistemas']]} for x in p['personas']],'organizacion_ref':org,'unidad_ref':unit,'unidad_recibo_ref':m['unidad']['recibo_ref']}
    map_raw=canon(reparto);crear(F/'reparto-propuesto.json',map_raw);p['fuente_reparto_aprobado']={'referencia':ref('prc_'),'version':1,'huella_sha256':sha(map_raw)}
    for pe in p['personas']:
        pe['preimagen_huella_sha256']=getter_hash(c,'vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3('+q(canon(pe).decode())+'::jsonb)')
    raw=canon(p);crear(F/'material-v3.json',raw)
    pre=getter_hash(c,'vec_autorizacion.preimagen_bootstrap_central_admin_v3('+q(raw.decode())+'::jsonb)')
    crear(F/'preimagen.sha256',(pre+'\n').encode())
    cli(c,'bootstrap_preparar',['-fuente',F/'material-v3.json','-plan',F/'plan-v3.json','-version-plan','3','-textos',c['kit']['catalogos']['bootstrap_preparar']])
    plan=json_privado(F/'plan-v3.json');estado('preparar_bootstrap',plan_sha256=plan['huella_plan_sha256'],siguiente='configurar_bootstrap')

def configurar_bootstrap(c):
    F=carpeta(c,'bootstrap');d=json_privado(F/'plan-v3.json');p=d['plan'];h=d['huella_plan_sha256'];raw=canon(p)
    if sha(raw)!=h or parse(p['caduca_en'])<=now():parar('bootstrap_plan_invalido')
    pre=privado(F/'preimagen.sha256').decode().strip();exigir_hash(pre,'bootstrap_preimagen_invalida')
    approval_sha=aprobacion(c,'bootstrap',h);preparar_cli_config(c,'bootstrap',F,h)
    cli(c,'bootstrap_preparar',['-fuente',F/'material-v3.json','-plan',F/'plan-v3.json','-version-plan','3','-cotejar','-aprobacion',F/'aprobacion.json','-textos',c['kit']['catalogos']['bootstrap_preparar']])
    dba=c['dba']['bootstrap']
    campos={'login_nombre':dba['login'],'proceso':dba['proceso'],'plan_sha256':h,'preimagen_sha256':pre,'aprobacion_ref':dba['aprobacion_ref'],'aprobacion_sha256':approval_sha,'fuente_reparto':canon(p['fuente_reparto_aprobado']).decode(),'fuente_identidad':canon(p['fuente_identidad']).decode(),'fuente_ca_admin':canon(p['fuente_ca_admin']).decode(),'vigente_hasta':p['caduca_en']}
    configurar_login(c,'bootstrap',F,'vec_autorizacion.config_bootstrap_central_admin_v3',campos)
    estado('configurar_bootstrap',plan_sha256=h,siguiente='aplicar_bootstrap')

def aplicar_bootstrap(c,replay=False):
    aplicar_fase(c,'bootstrap',replay)

def verificar(c):
    verifica_codigo(c)
    f=confirmacion_vigente(carpeta(c,'fuentes'),'fuentes')[1];u=confirmacion_vigente(carpeta(c,'unidad'),'unidad')[1];b=confirmacion_vigente(carpeta(c,'bootstrap'),'bootstrap')[1]
    if any(x.get('estado')!='permitido' or not x.get('auditoria_intento') or not x.get('recibo',{}).get('auditoria_ref') for x in (f,u,b)):parar('acuse_positivo_o_auditoria_ausente')
    plan=json_privado(carpeta(c,'bootstrap')/'plan-v3.json')['plan']
    query="""SELECT jsonb_build_object(
      'fuentes',(SELECT count(*) FROM vec_autorizacion.fuentes_iniciales_admin_v1),
      'fuentes_ref',(SELECT recibo_ref FROM vec_autorizacion.fuentes_iniciales_admin_v1),
      'is',(SELECT count(*) FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1),
      'ca',(SELECT count(*) FROM vec_contexto_actor_v1.fuentes_iniciales_admin_v1),
      'unidad',(SELECT count(*) FROM vec_personal.unidad_inicial_admin_v1),
      'unidad_ref',(SELECT recibo_ref FROM vec_personal.unidad_inicial_admin_v1),
      'bootstrap',(SELECT count(*) FROM vec_autorizacion.bootstrap_central_admin_v3),
      'bootstrap_ref',(SELECT recibo_ref FROM vec_autorizacion.bootstrap_central_admin_v3),
      'control',(SELECT bootstrap_estado FROM vec_autorizacion.control_continuidad_admin WHERE control_id),
      'app_refs',vec_autorizacion.administradores_aplicacion_efectivos_internos_v3(),
      'sys_refs',(SELECT coalesce(jsonb_agg(jsonb_build_object('persona_ref',x.principal_id,'perfil_ref',x.perfil_activo_ref)),'[]'::jsonb)
        FROM vec_autorizacion.asignacion_perfil_actual a
        JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
        JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m USING(version_rol_ref)
        WHERE m.categoria_administrativa='sistemas' AND x.documento->>'estado'='activa'));"""
    d=fila_json(sql(c,"BEGIN ISOLATION LEVEL SERIALIZABLE;SET LOCAL timezone='UTC';"+query+'ROLLBACK;'),'postestado_no_disponible')
    if any(d[k]!=1 for k in ('fuentes','is','ca','unidad','bootstrap')) or d['control']!='consumido':parar('postestado_divergente')
    if (d['fuentes_ref'],d['unidad_ref'],d['bootstrap_ref'])!=(f['recibo']['recibo_ref'],u['recibo']['recibo_ref'],b['recibo']['recibo_ref']):parar('recibos_no_conservados')
    expected_app={(x['persona_ref'],x['perfil_ref']) for x in plan['personas']}
    actual_app={(x['persona_ref'],x['perfil_ref']) for x in d['app_refs']}
    expected_sys={(x['persona_ref'],y['perfil_ref']) for x in plan['personas'] for y in x['sistemas']}
    actual_sys={(x['persona_ref'],x['perfil_ref']) for x in d['sys_refs']}
    if len(d['app_refs'])!=2 or len(d['sys_refs'])!=1 or expected_app!=actual_app or expected_sys!=actual_sys:parar('perfiles_efectivos_divergentes')
    estado('verificar',fuentes=1,unidad=1,app=2,sys=1,recibos_conservados=True,pendiente_cotejo_cadena_y_frio=True)

def main():
    parser=argparse.ArgumentParser(description='Arranque H10 por etapas, con recibos recuperables')
    parser.add_argument('--config',required=True)
    parser.add_argument('--etapa',required=True,choices=('comprobar','preparar_fuentes','configurar_fuentes','aplicar_fuentes','preparar_unidad','configurar_unidad','aplicar_unidad','preparar_bootstrap','configurar_bootstrap','aplicar_bootstrap','recuperar_fuentes','recuperar_unidad','recuperar_bootstrap','verificar'))
    args=parser.parse_args();os.umask(0o077)
    try:
        c=cargar_config(args.config)
        if args.etapa=='comprobar':
            d=comprobar(c,vacio=True);estado('comprobar',pg18=d['pg18'],fuentes_vacias=True,control_pendiente=True,backup_ref_declarada=bool(c['backup']['ref']),siguiente='preparar_fuentes')
        else:
            verifica_codigo(c)
            acciones={'preparar_fuentes':preparar_fuentes,'configurar_fuentes':configurar_fuentes,'aplicar_fuentes':aplicar_fuentes,'preparar_unidad':preparar_unidad,'configurar_unidad':configurar_unidad,'aplicar_unidad':aplicar_unidad,'preparar_bootstrap':preparar_bootstrap,'configurar_bootstrap':configurar_bootstrap,'aplicar_bootstrap':aplicar_bootstrap,'recuperar_fuentes':lambda c:aplicar_fuentes(c,True),'recuperar_unidad':lambda c:aplicar_unidad(c,True),'recuperar_bootstrap':lambda c:aplicar_bootstrap(c,True),'verificar':verificar}
            acciones[args.etapa](c)
    except Paro as e:
        print(json.dumps({'estado':'detenido','clave':e.clave,'actual':e.actual,'esperado':e.esperado},separators=(',',':')),file=sys.stderr)
        return 1
    except (OSError,ValueError,KeyError,IndexError,TypeError,subprocess.TimeoutExpired):
        print(json.dumps({'estado':'detenido','clave':'entrada_o_dependencia_invalida','actual':'no_observado','esperado':'contrato_valido'},separators=(',',':')),file=sys.stderr)
        return 2
    return 0
if __name__=='__main__':sys.exit(main())
