#!/usr/bin/env python3
"""Advance preserved H1 Users grants for mail and image in the owned clone."""
import ast
from datetime import datetime, timezone
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
from urllib.parse import urlencode


class H4Error(RuntimeError):
    pass


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def material_contract(source):
    """Read only the closed data contract; never run the private SQL kit."""
    constants = {}
    for node in ast.parse(source).body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name):
            name = node.targets[0].id
            if name in {"PREFERENCIAS", "CORREOS", "IMAGEN"}:
                if name in constants:
                    raise H4Error("Repeated private H4 contract.")
                constants[name] = ast.literal_eval(node.value)
    if set(constants) != {"PREFERENCIAS", "CORREOS", "IMAGEN"}:
        raise H4Error("Incomplete private H4 contract.")
    entries = []
    for name, expected in [("PREFERENCIAS", 2), ("CORREOS", 6), ("IMAGEN", 2)]:
        purpose, resource, actions = constants[name]
        if len(actions) != expected:
            raise H4Error("Unexpected private H4 action count.")
        for action, fields in actions:
            entries.append({"accion": action, "finalidad": purpose, "tipo": resource, "campos": fields})
    if len({item["accion"] for item in entries}) != 10:
        raise H4Error("Repeated private H4 action.")
    return entries


def make_harness(repo, users):
    # H1 reconstructs the historical person DTO, then resolves a fresh 120s
    # session through the existing identity and ContextoActor authorities.
    return users.make_harness(repo) + GO_H4


def identity_preimage(result):
    return {key: result[key] for key in ("accounts", "persons", "profiles", "install_at", "references_preserved")}


def preflight(repo, container, state, material, pg_port, engine="docker"):
    repo, state, material = Path(repo), Path(state), Path(material)
    if engine != "docker" or container != "vec-codexm-recorridos-20260930" or pg_port != 55531:
        raise H4Error("H4 destination is outside the owned clone.")
    users = load(Path(__file__).with_name("clon_usuarios.py"), "h4_h1")
    runtime = load(Path(__file__).with_name("clon_runtime.py"), "h4_runtime")
    material_helper = load(Path(__file__).with_name("clon_material.py"), "h4_material")
    runtime.validate_state(repo.resolve(), state)
    if material.resolve() != state / "material" or material.is_symlink():
        raise H4Error("H4 material destination changed.")
    inspect = json.loads(material_helper.run([engine, "inspect", container]))[0]
    material_helper.validate_container(inspect, pg_port)
    marker = json.loads(users.private(state / "DB_READY.json"))
    if marker.get("propietario") != "Codex-M" or marker.get("contenedor") != container or marker.get("puerto_pg") != pg_port:
        raise H4Error("H4 clone readiness marker changed.")
    if (state / "runtime-process.json").exists():
        raise H4Error("Stop the owned application before H4 provisioning.")
    result = json.loads(users.private(state / "usuarios-result.json"))
    if result.get("references_preserved") is not True:
        raise H4Error("Complete preserved H1 installation is required.")
    for key in ("accounts", "persons", "profiles"):
        values = result.get(key, [])
        if len(values) != 2 or len(set(values)) != 2:
            raise H4Error("Ambiguous preserved H1 result.")
    env = json.loads(users.private(state / "runtime-config.json"))
    runtime.validate_dsn(env["VEC_CT_GOBIERNO_DATABASE_URL"], pg_port, state)
    if not (state / ("source-" + marker["commit"])).is_dir():
        raise H4Error("Pinned H4 application source is unavailable.")
    return {"env": {}, "profiles": {"usuarios_h4": {"h1_ready": True}}, "blockers": []}


def provision(repo, container, state, material, pg_port, engine="docker"):
    preflight(repo, container, state, material, pg_port, engine)
    repo, state, material = Path(repo), Path(state), Path(material)
    users = load(Path(__file__).with_name("clon_usuarios.py"), "h4_users")
    runtime = load(Path(__file__).with_name("clon_runtime.py"), "h4_runtime_provision")
    marker = json.loads(users.private(state / "DB_READY.json"))
    h1_plan = json.loads(users.private(state / "usuarios-plan.json"))
    h1_result = identity_preimage(json.loads(users.private(state / "usuarios-result.json")))
    env = json.loads(users.private(state / "runtime-config.json"))
    # This is a private input, never copied into Git or executed as a program.
    kit = Path.home() / "Trabajo/hito4/paquete/h4_material_508bc.py"
    contract = material_contract(kit.read_text())
    plan_path = state / "usuarios-h4-plan.json"
    if not plan_path.exists():
        plan = {"version": 1, "emit_at": datetime.now(timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z"),
                "system_id": h1_plan["system_id"], "pg_port": pg_port, "h1": h1_result, "contract": contract}
        users.atomic(plan_path, json.dumps(plan).encode())
    plan = json.loads(users.private(plan_path))
    if plan.get("version") != 1 or plan.get("system_id") != h1_plan["system_id"] or plan.get("pg_port") != pg_port or plan.get("h1") != h1_result or plan.get("contract") != contract:
        raise H4Error("The preserved H4 installation plan changed.")
    source = state / ("source-" + marker["commit"])
    installation = state / "usuarios-h4-source"
    if installation.is_symlink():
        raise H4Error("Unsafe H4 private source directory.")
    if installation.exists():
        shutil.rmtree(installation)
    shutil.copytree(source, installation)
    harness = installation / "internal/app/bootstrap/codexm_usuarios_h4_instalacion_test.go"
    harness.write_text(make_harness(repo, users))
    go = runtime.local_go(source, None)
    admin = "postgres://postgres@127.0.0.1:" + str(pg_port) + "/postgres?" + urlencode({"sslmode": "verify-full", "sslrootcert": str(material / "pg/ca.crt")})
    with tempfile.TemporaryDirectory(prefix="h4-", dir=state / "tmp") as temporary:
        execution = {"PATH": str(go.parent) + ":/usr/bin:/bin", "HOME": str(state), "TMPDIR": temporary,
                     "GOCACHE": "/dev/shm/go-build", "GOPATH": str(Path.home() / "go"), "GOMODCACHE": str(Path.home() / "go/pkg/mod"),
                     "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
                     "VEC_CODEXM_H4_INSTALL": "CLON_PRIVADO_REVISADO", "VEC_CODEXM_STATE": str(state),
                     "VEC_CODEXM_ADMIN_DSN": admin, "VEC_CT_GOBIERNO_DATABASE_URL": env["VEC_CT_GOBIERNO_DATABASE_URL"]}
        descriptor = os.open(state / "usuarios-h4-install.log", os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "wb") as log:
            result = subprocess.run([str(go), "test", "-p", "32", "./internal/app/bootstrap", "-run", "^TestCodexMH4(Contract|Install)$", "-count=1", "-v"],
                                    cwd=installation, env=execution, stdout=log, stderr=subprocess.STDOUT, timeout=240)
        if result.returncode:
            raise H4Error("H4 authority installation failed; inspect the private H4 log.")
    result = json.loads(users.private(state / "usuarios-h4-authority-result.json"))
    users.atomic(state / "usuarios-h4-result.json", json.dumps(result).encode())
    return {"env": {}, "profiles": {"usuarios_h4": result}, "blockers": []}


GO_H4 = r'''

const codexMH4Actor = "desarrollo:usuarios-correos-imagen-508bc:hito4"

func codexMH4Snapshot(before core.InstantaneaAutorizacion, emitted time.Time)(core.InstantaneaAutorizacion,error){
 after:=clonarInstantaneaAutorizacionPostgreSQLDesarrollo(before)
 after.VersionRol.Version=2;after.VersionRol.PublicadaPor=codexMH4Actor;after.VersionRol.PublicadaEn=emitted
 for _,action:=range accionesCorreosUsuarios {after.VersionRol.Concesiones=append(after.VersionRol.Concesiones,core.ConcesionRol{Accion:action.accion,ModuloID:"usuarios",TipoRecurso:usuariosports.TipoRecursoCorreos,Finalidades:[]string{usuariosports.FinalidadCorreosPropios},GarantiaMinima:core.AuthAssuranceHigh,CamposPermitidos:usuariosports.CamposPermitidosCorreos(action.accion)})}
 for _,action:=range accionesImagenUsuarios {after.VersionRol.Concesiones=append(after.VersionRol.Concesiones,core.ConcesionRol{Accion:action.accion,ModuloID:"usuarios",TipoRecurso:usuariosports.TipoRecursoImagen,Finalidades:[]string{usuariosports.FinalidadImagenPropia},GarantiaMinima:core.AuthAssuranceHigh,CamposPermitidos:usuariosports.CamposPermitidosImagen(action.accion)})}
 after.ControlVigenciaVersionRol.VersionRolRef=after.VersionRol.Referencia();after.ControlVigenciaVersionRol.ActualizadoPor=codexMH4Actor;after.ControlVigenciaVersionRol.ActualizadoEn=emitted
 after.AsignacionPerfil.Version=2;after.AsignacionPerfil.VersionRolRef=after.VersionRol.Referencia();after.AsignacionPerfil.VigenteDesde=emitted;after.AsignacionPerfil.EmitidaPor=codexMH4Actor;after.AsignacionPerfil.EmitidaEn=emitted
 return after,after.Validar()
}

func TestCodexMH4Contract(t *testing.T){
 historic,_:=time.Parse(time.RFC3339,"2026-09-29T04:00:00Z")
 catalogue,_:=core.HuellaCatalogoPoliticasAutorizacion(nil)
 for _,surface:=range []core.SuperficieAutenticacionActorV1{core.SuperficieAutenticacionInternaCorporativaV1,core.SuperficieAutenticacionExternaPersonalV1}{
  c:=configuracionUsuariosPreferenciasDesarrollo{Superficie:surface,Cuentas:[]cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo:cuentaRutasDietasDesarrollo{CuentaRef:"cta_0123456789abcdefghijklmnop",PerfilRef:"prf_0123456789abcdefghijklmnop",CertificadoSHA256:"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}}}
  account,err:=construirCuentaProvisionPreferenciasHito1(c,historic);if err!=nil{t.Fatal("historic DTO")}
  before,err:=instantaneaProvisionPreferenciasHito1(account,historic,1,catalogue);if err!=nil{t.Fatal("historic snapshot")}
  after,err:=codexMH4Snapshot(before,historic.Add(25*time.Hour));if err!=nil||len(after.VersionRol.Concesiones)!=10||len(before.VersionRol.Concesiones)!=2||!after.AsignacionPerfil.VigenteHasta.Equal(before.AsignacionPerfil.VigenteHasta)||after.AsignacionPerfil.PrincipalID!=before.AsignacionPerfil.PrincipalID||after.AsignacionPerfil.PerfilActivoRef!=before.AsignacionPerfil.PerfilActivoRef||len(after.AsignacionPerfil.Ambitos)!=1||after.AsignacionPerfil.Ambitos[0].Clave!="persona_ref"||len(after.AsignacionPerfil.Ambitos[0].Valores)!=1||after.AsignacionPerfil.Ambitos[0].Valores[0]!=account.personaRef {t.Fatal("H4 broadens identity, lifetime or scope")}
 }
}

func codexMH4Exact(ctx context.Context,pool *pgxpool.Pool,snapshot core.InstantaneaAutorizacion,authority autoridadPostgreSQLDesarrollo)(bool,error){
 tx,err:=pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return false,err};defer tx.Rollback(ctx)
 if _,err=tx.Exec(ctx,`SET LOCAL ROLE vec_autorizacion_propietario`);err!=nil{return false,err}
 actual,exists,err:=leerAsignacionActualPostgreSQLDesarrollo(ctx,tx,snapshot.AsignacionPerfil.PerfilActivoRef);if err!=nil||!exists{return false,fmt.Errorf("H4 current assignment")}
 hash,_:=snapshot.AsignacionPerfil.HuellaSHA256();document,_:=json.Marshal(snapshot.AsignacionPerfil)
 if actual.referencia!=snapshot.AsignacionPerfil.Referencia()||actual.huella!=hash||actual.actoRef!=authority.actoAsignacion{return false,nil}
 var exact bool
 if tx.QueryRow(ctx,`SELECT $1::jsonb=$2::jsonb`,actual.documento,document).Scan(&exact)!=nil||!exact{return false,fmt.Errorf("H4 assignment document")}
 if authority.comprobarOrigenOperativo(ctx,tx,actual)!=nil{return false,fmt.Errorf("H4 assignment withdrawn or expired")}
 roleHash,_:=snapshot.VersionRol.HuellaSHA256();controlHash,_:=snapshot.ControlVigenciaVersionRol.HuellaSHA256();roleDocument,_:=json.Marshal(snapshot.VersionRol);controlDocument,_:=json.Marshal(snapshot.ControlVigenciaVersionRol)
 if tx.QueryRow(ctx,`SELECT r.huella_sha256=$2 AND r.documento=$3::jsonb AND c.huella_sha256=$4 AND c.documento=$5::jsonb AND a.revision=$6 AND a.acto_ref=$7 AND a.actualizada_por=$8 AND a.actualizada_en=$9
 FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref) JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision WHERE r.version_rol_ref=$1 FOR SHARE OF r,a,c`,snapshot.VersionRol.Referencia(),roleHash,roleDocument,controlHash,controlDocument,snapshot.ControlVigenciaVersionRol.Revision,authority.actoControlRol,snapshot.ControlVigenciaVersionRol.ActualizadoPor,snapshot.ControlVigenciaVersionRol.ActualizadoEn).Scan(&exact)!=nil||!exact{return false,fmt.Errorf("H4 exact role/control")}
 return true,nil
}

func TestCodexMH4Install(t *testing.T){
 if os.Getenv("VEC_CODEXM_H4_INSTALL")!="CLON_PRIVADO_REVISADO"{t.Fatal("H4 installation guard")}
 root:=os.Getenv("VEC_CODEXM_STATE");material:=filepath.Join(root,"material")
 var plan struct{EmitAt string `json:"emit_at"`;SystemID string `json:"system_id"`;PGPort uint16 `json:"pg_port"`;H1 struct{Accounts,Persons,Profiles []string};Contract []struct{Action string `json:"accion"`;Purpose string `json:"finalidad"`;Resource string `json:"tipo"`;Fields []string `json:"campos"`}}
 data,err:=os.ReadFile(filepath.Join(root,"usuarios-h4-plan.json"));if err!=nil||json.Unmarshal(data,&plan)!=nil||len(plan.H1.Accounts)!=2||len(plan.H1.Persons)!=2||len(plan.H1.Profiles)!=2||len(plan.Contract)!=10{t.Fatal("H4 plan")}
 emitted,err:=time.Parse(time.RFC3339Nano,plan.EmitAt);if err!=nil||emitted.After(time.Now().UTC()){t.Fatal("H4 stable emission")}
 historic,_:=time.Parse(time.RFC3339,"2026-09-29T04:00:00Z")
 ctx,cancel:=context.WithTimeout(context.Background(),120*time.Second);defer cancel()
 adminCfg,err:=pgxpool.ParseConfig(os.Getenv("VEC_CODEXM_ADMIN_DSN"));if err!=nil||adminCfg.ConnConfig.Host!="127.0.0.1"||adminCfg.ConnConfig.Port!=plan.PGPort||plan.PGPort!=55531||len(adminCfg.ConnConfig.Fallbacks)!=0||validarTLSPostgreSQLBorradores(&adminCfg.ConnConfig.Config,false)!=nil{t.Fatal("H4 admin TLS destination")}
 admin,err:=pgxpool.NewWithConfig(ctx,adminCfg);if err!=nil{t.Fatal("H4 admin pool")};defer admin.Close()
 var systemID string;if admin.QueryRow(ctx,`SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&systemID)!=nil||systemID!=plan.SystemID{t.Fatal("H4 clone identity")}
 var objects bool;if admin.QueryRow(ctx,`SELECT to_regproc('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada') IS NOT NULL AND to_regproc('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada') IS NOT NULL AND to_regclass('vec_usuarios.correos_direccion') IS NOT NULL AND to_regclass('vec_usuarios.imagen_actual') IS NOT NULL AND to_regclass('vec_documentos.imagen_personal') IS NOT NULL`).Scan(&objects)!=nil||!objects{t.Fatal("H4 SQL prerequisites")}
 gobierno,_,err:=abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx,os.Getenv("VEC_CT_GOBIERNO_DATABASE_URL"),"vec-codexm-usuarios-h4",rolGobiernoPostgreSQLContratacionTemporalDesarrollo);if err!=nil{t.Fatal("H4 government pool")};defer gobierno.Close()
 revision,catalogue,err:=catalogoPoliticasProvisionPreferenciasHito1(ctx,gobierno);if err!=nil{t.Fatal("H4 policy catalogue")}
 hmacMaterial,err:=cargarMaterialIdempotenciaDesarrollo(material,filepath.Join(material,"idempotencia/configuracion.json"));if err!=nil{t.Fatal("H4 HMAC material")}
 derivador,err:=nuevoDerivadorIdentidadOperacionDesarrollo(&hmacMaterial);if err!=nil{t.Fatal("H4 HMAC derivation")};defer derivador.borrar()
 ca,err:=leerCertificadoProvisionPreferenciasHito1(filepath.Join(material,"ca/ca.crt"));if err!=nil{t.Fatal("H4 CA")};roots:=x509.NewCertPool();roots.AddCert(ca)
 historyBefore,err:=codexMModuleHistory(ctx,admin);if err!=nil{t.Fatal("H4 module history before")}
 var before,after []core.InstantaneaAutorizacion;var configs []configuracionUsuariosPreferenciasDesarrollo;var certificates []*x509.Certificate;var chains [][]*x509.Certificate;var authorities []autoridadPostgreSQLDesarrollo
 for index,surface:=range []string{"interna","externa"}{
  original,err:=codexMConfiguration(filepath.Join(root,"usuarios-before",surface+"-original.json"));if err!=nil{t.Fatal("H4 original H1 configuration")}
  current,err:=codexMConfiguration(filepath.Join(material,"identidad","usuarios-preferencias-"+surface+".json"));if err!=nil{t.Fatal("H4 current configuration")}
  if original.Superficie!=current.Superficie||original.Cuentas[0].Sujeto!=current.Cuentas[0].Sujeto||original.Cuentas[0].PerfilRef!=current.Cuentas[0].PerfilRef||original.Cuentas[0].CertificadoSHA256!=current.Cuentas[0].CertificadoSHA256||current.Cuentas[0].CuentaRef!=plan.H1.Accounts[index]||current.Cuentas[0].PerfilRef!=plan.H1.Profiles[index]{t.Fatal("H4 identity preimage changed")}
  account,err:=construirCuentaProvisionPreferenciasHito1(original,historic);if err!=nil||account.personaRef!=plan.H1.Persons[index]{t.Fatal("H4 historical person changed")}
  prior,err:=instantaneaProvisionPreferenciasHito1(account,historic,revision,catalogue);if err!=nil||!prior.AsignacionPerfil.VigenteEn(time.Now().UTC()){t.Fatal("H4 expired H1 assignment")}
  next,err:=codexMH4Snapshot(prior,emitted);if err!=nil{t.Fatal("H4 snapshot")}
  for i,concession:=range next.VersionRol.Concesiones{expected:=plan.Contract[i];fields,_:=json.Marshal(concession.CamposPermitidos);expectedFields,_:=json.Marshal(expected.Fields);if concession.Accion!=expected.Action||concession.ModuloID!="usuarios"||concession.TipoRecurso!=expected.Resource||len(concession.Finalidades)!=1||concession.Finalidades[0]!=expected.Purpose||string(fields)!=string(expectedFields){t.Fatal("H4 private/source grant contract diverged")}}
  authority:=autoridadPostgreSQLDesarrollo{pool:gobierno,prefijoBloqueo:"vec:hito1:usuarios:preferencias:",actoControlRol:"acto:hito1:usuarios:preferencias:control:"+string(current.Superficie),actoAsignacion:"acto:hito1:usuarios:preferencias:asignacion:"+string(current.Superficie),actoSesion:"acto:h4:usuarios:correos-imagen:sesion:"+string(current.Superficie),exigirOrigenOperativo:true}
  exactPrior,err:=codexMH4Exact(ctx,gobierno,prior,authority);if err!=nil{t.Fatal("H4 historical preimage revoked or divergent")}
  exactNext,err:=codexMH4Exact(ctx,gobierno,next,authority);if err!=nil{t.Fatal("H4 postimage revoked or divergent")}
  if !exactPrior&&!exactNext{t.Fatal("H4 CAS preimage differs")}
  certificateFile:="cliente";if index==1{certificateFile="intervencion"}
  certificate,err:=leerCertificadoProvisionPreferenciasHito1(filepath.Join(material,"mtls",certificateFile+".crt"));if err!=nil{t.Fatal("H4 certificate")}
  verified,err:=certificate.Verify(x509.VerifyOptions{Roots:roots,KeyUsages:[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}});if err!=nil||len(verified)!=1{t.Fatal("H4 certificate trust")}
  configs=append(configs,current);certificates=append(certificates,certificate);chains=append(chains,verified[0]);before=append(before,prior);after=append(after,next);authorities=append(authorities,authority)
 }
 // Both surfaces have exact H1 or H4 postimages before the first effect.
 // The existing authority repeats CAS and anti-revocation inside its own
 // serializable transaction; no manual SQL write is introduced here.
 for i,current:=range configs{
  vinculo,err:=codexMSession(ctx,current,derivador,certificates[i],chains[i]);if err!=nil{t.Fatal("H4 real short identity/context session")};authorities[i].vinculo=vinculo
  sessionData,err:=vinculo.Datos();if err!=nil{t.Fatal("H4 fresh session data")}
  var actualSessionAct string
  if admin.QueryRow(ctx,`SELECT acto_ref FROM vec_autorizacion.control_sesion_actual_v1 WHERE sesion_ref=$1 AND control_sesion_ref=$2 AND revision=$3 AND actualizada_en=$4`,sessionData.SesionRef,sessionData.ControlSesionRef,sessionData.ControlSesionRevision,sessionData.SesionRevalidadaEn).Scan(&actualSessionAct)!=nil||actualSessionAct==""{t.Fatal("H4 authoritative fresh session act")}
  authorities[i].actoSesion=actualSessionAct
  if err=authorities[i].publicarInstantaneaDesdePreimagen(ctx,after[i],before[i]);err!=nil{t.Fatal("H4 exact authority transition rejected")}
  exact,err:=codexMH4Exact(ctx,gobierno,after[i],authorities[i]);if err!=nil||!exact{t.Fatal("H4 exact postimage missing")}
 }
 historyAfter,err:=codexMModuleHistory(ctx,admin);if err!=nil||historyAfter!=historyBefore{t.Fatal("H4 changed CT/Bolsa module history")}
 result:=map[string]any{"version":1,"roles_version":2,"assignments_version":2,"grants_per_role":10,"surfaces":2,"historical_persons_preserved":true,"historical_end_preserved":true,"h1_acts_preserved":true,"session_act_source":"identity_authority","emit_at":plan.EmitAt,"module_history_before":historyBefore,"module_history_after":historyAfter}
 encoded,err:=json.Marshal(result);if err!=nil||os.WriteFile(filepath.Join(root,"usuarios-h4-authority-result.json"),encoded,0600)!=nil{t.Fatal("H4 private result")}
 t.Log("two preserved H1 Users scopes advanced to mail/image v2 by exact authority CAS")
}
'''
