#!/usr/bin/env python3
"""Install the two preserved synthetic H1 Users identities in an owned clone.

The helper calls existing identity/context/authorization authorities from a
private pinned source copy. It never patches application source or SQL schema.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime, timezone
from urllib.parse import urlencode, urlsplit

H1_PROVISION = "f9c081fc89909239d236466d6bb52cbe6272cdd7"
H1_IDENTITY = "d0487b4f18f18ddf7e0493e8548b27c3da295d41"


class UsersError(RuntimeError):
    pass


def fail(reason):
    raise UsersError(reason)


def private(path):
    if path.is_symlink() or not path.is_file():
        fail("Missing private installation input.")
    info = path.stat()
    if info.st_uid != os.getuid() or info.st_nlink != 1 or info.st_mode & 0o077:
        fail("Unsafe private installation input.")
    return path.read_bytes()


def sha(data):
    return hashlib.sha256(data).hexdigest()


def atomic(path, data):
    temporary = path.with_suffix(path.suffix + ".new")
    descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, "wb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)


def account_only(before, canonical):
    data = json.loads(before)
    if len(data.get("cuentas", [])) != 1 or not re.fullmatch(r"cta_[A-Za-z0-9_:-]+", canonical):
        fail("Invalid canonical identity account.")
    old = data["cuentas"][0]["cuenta_ref"]
    pattern = re.compile(rb'("cuenta_ref"\s*:\s*")' + re.escape(old.encode()) + rb'(")')
    if len(pattern.findall(before)) != 1:
        fail("Ambiguous private account reference.")
    result = pattern.sub(lambda match: match.group(1) + canonical.encode() + match.group(2), before, count=1)
    data["cuentas"][0]["cuenta_ref"] = canonical
    if json.loads(result) != data:
        fail("The private replacement changed more than the account.")
    return result


def selected_go_functions(source, names):
    functions = {match.group(1): match.group(0) for match in re.finditer(
        r"^func ([A-Za-z_][A-Za-z0-9_]*)\([^\n]*(?:\n.*?)*?^}\n?", source, re.MULTILINE)}
    # Go declarations close at column zero in these pinned, gofmt formatted inputs.
    if set(names) - set(functions):
        fail("The pinned H1 authority source does not match the expected contract.")
    return "\n\n".join(functions[name] for name in names)


def historical(repo, commit, path):
    result = subprocess.run(["git", "-C", str(repo), "show", commit + ":" + path], capture_output=True)
    if result.returncode:
        fail("The pinned H1 authority source is unavailable.")
    return result.stdout.decode()


def make_harness(repo):
    first = historical(repo, H1_PROVISION, "internal/app/bootstrap/usuarios_preferencias_hito1_provision_test.go")
    second = historical(repo, H1_IDENTITY, "internal/app/bootstrap/usuarios_preferencias_hito1_identidad_test.go")
    # Reconstruct historical DTOs without creating or validating the H1
    # authentication/session, whose one-day window must never gate replay.
    constructor = selected_go_functions(first, ["construirCuentaProvisionPreferenciasHito1"])
    legacy_constructor = constructor.replace("func construirCuentaProvisionPreferenciasHito1(", "func codexMLegacyConstructorExpired(", 1).replace("relojRutasDietas{}", "codexMExpiredClock{}")
    marker = "\tautenticacion := core.AutenticacionRevalidadaV1{"
    if constructor.count(marker) != 1:
        fail("The historical DTO constructor changed.")
    dto_constructor = constructor.split(marker, 1)[0] + "\treturn cuentaProvisionPreferenciasHito1{configuracion:c, cuenta:cuentaConfig, personaRef:personaRef, contexto:resultado, operacionRef:referenciaProvisionPreferenciasHito1(\"oca_\",base+\"\\x00publicar-contexto\")}, nil\n}\n"
    first = first.replace(constructor, dto_constructor, 1)
    identity_source = historical(repo, "7f1ecea2fd9f8912d255a80e74da84c69e46b978", "internal/vec/adapters/httpseguridad/postgres/capacidades.go")
    identity_query = re.search(r"const consultaAcreditarCapacidad = `(.+?)`", identity_source, re.DOTALL)
    if not identity_query:
        fail("Identity capability preflight contract is missing.")
    diagnostic = "\nconst codexMDiagnosticIdentitySQL = `" + identity_query.group(1) + "`\n"
    first_names = ["referenciaProvisionPreferenciasHito1", "huellaProvisionPreferenciasHito1",
                   "construirCuentaProvisionPreferenciasHito1", "instantaneaProvisionPreferenciasHito1",
                   "leerCertificadoProvisionPreferenciasHito1", "catalogoPoliticasProvisionPreferenciasHito1"]
    second_names = ["seudonimosCuentaPreferenciasHito1", "operacionProvisionPreferenciasHito1",
                    "crearOComprobarLoginProvisionadorHito1",
                    "cuentaIdentidadActivaHito1", "provisionarYRegistrarAliasIdentidadHito1",
                    "nuevoResultadoCuentaConciliadaHito1"]
    struct = re.search(r"type cuentaProvisionPreferenciasHito1 struct {.*?^}", first, re.MULTILINE | re.DOTALL)
    if not struct:
        fail("Missing pinned H1 account contract.")
    return GO_IMPORTS + 'const loginProvisionadorHito1 = "vec_hito1_pref_provisionador"\n' + struct.group(0) + "\n" + selected_go_functions(first, first_names) + "\n" + selected_go_functions(second, second_names) + diagnostic + legacy_constructor + GO_DRIVER


GO_IMPORTS = r'''package bootstrap
import (
 "bytes"
 "context"
 "crypto/tls"
 "crypto/x509"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http/httptest"
 "os"
 "path/filepath"
 "testing"
 "time"
 "crypto/sha256"
 "encoding/hex"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgxpool"
 "vec-diputacion-granada/config"
 usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
 postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
 contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
 vecapp "vec-diputacion-granada/internal/vec/application"
 core "vec-diputacion-granada/internal/vec/domain"
)
'''

GO_DRIVER = r'''

type codexMExpiredClock struct{}
func (codexMExpiredClock) Ahora()time.Time{value,_:=time.Parse(time.RFC3339,"2026-09-30T05:00:00Z");return value}
func TestCodexMHistoricalUsersDTOAfterSessionExpiry(t *testing.T){
 historic,_:=time.Parse(time.RFC3339,"2026-09-29T04:00:00Z")
 c:=configuracionUsuariosPreferenciasDesarrollo{Superficie:core.SuperficieAutenticacionInternaCorporativaV1,
 Cuentas:[]cuentaUsuariosPreferenciasDesarrollo{{cuentaRutasDietasDesarrollo:cuentaRutasDietasDesarrollo{
 CuentaRef:"cta_0123456789abcdefghijklmnop",PerfilRef:"prf_0123456789abcdefghijklmnop",CertificadoSHA256:"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}}}
 if _,err:=codexMLegacyConstructorExpired(c,historic);err==nil{t.Fatal("old H1 session unexpectedly valid after its 24h deadline")}
 account,err:=construirCuentaProvisionPreferenciasHito1(c,historic)
 if err!=nil||account.contexto.Validar()!=nil||account.personaRef==""||account.contexto.Contexto.PerfilActivoRef!=c.Cuentas[0].PerfilRef{t.Fatal("historical DTO depends on expired H1 authentication")}
 if !account.contexto.Contexto.Instantanea.VigenteHasta.Equal(historic.Add(14*24*time.Hour)){t.Fatal("historical person validity was renewed")}
}

func codexMConfiguration(path string) (configuracionUsuariosPreferenciasDesarrollo,error) {
 var value configuracionUsuariosPreferenciasDesarrollo
 data,err:=leerFicheroMaterialSeguro(path,128<<10)
 if err!=nil||validarClavesJSONUnicas(data)!=nil {return value,fmt.Errorf("private configuration")}
 decoder:=json.NewDecoder(bytes.NewReader(data));decoder.DisallowUnknownFields()
 var extra any
 if decoder.Decode(&value)!=nil||!errors.Is(decoder.Decode(&extra),io.EOF)||len(value.Cuentas)!=1 {return value,fmt.Errorf("configuration contract")}
 return value,nil
}

func codexMLookup(ctx context.Context,admin *pgxpool.Pool,derivador *derivadorIdentidadOperacionDesarrollo,original,current configuracionUsuariosPreferenciasDesarrollo)(string,error){
 found:=""
 for _,ref:=range []string{original.Cuentas[0].CuentaRef,current.Cuentas[0].CuentaRef}{
  h,err:=seudonimosCuentaPreferenciasHito1(ctx,derivador,original,"desarrollo:"+ref)
  if err!=nil{return "",fmt.Errorf("identity HMAC")}
  rows,err:=admin.Query(ctx,`SELECT c.cuenta_ref FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a
 JOIN vec_identidad_sesiones_v1.cuenta c USING(cuenta_ref)
 WHERE a.esquema_hmac=$1 AND a.dominio_hmac_ref=$2 AND a.clave_hmac_id=$3 AND a.clave_hmac_version=$4
 AND a.cuenta_id_hmac=$5 AND a.sujeto_id_hmac=$6`,h.Esquema,h.DominioRef,h.ClaveID,int64(h.ClaveVersion),h.CuentaIDHMAC[:],h.SujetoIDHMAC[:])
  if err!=nil{return "",fmt.Errorf("identity lookup")}
  for rows.Next(){var candidate string;if rows.Scan(&candidate)!=nil||(found!=""&&found!=candidate){rows.Close();return "",fmt.Errorf("ambiguous identity")};found=candidate}
  if rows.Err()!=nil{rows.Close();return "",fmt.Errorf("identity lookup")};rows.Close()
  var n int
  if admin.QueryRow(ctx,`SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta_ref=$1`,ref).Scan(&n)!=nil{return "",fmt.Errorf("account preimage")}
  if n>0&&found!=ref{return "",fmt.Errorf("account has no exact canonical alias")}
 }
 if found!=""&&cuentaIdentidadActivaHito1(ctx,admin,found)!=nil{return "",fmt.Errorf("identity is not active")}
 return found,nil
}

func codexMIdentityFlags(ctx context.Context,pool *pgxpool.Pool,group string,functions []string)string{
 var login,current string;var flags [7]bool
 err:=pool.QueryRow(ctx,codexMDiagnosticIdentitySQL,group,functions).Scan(&login,&current,&flags[0],&flags[1],&flags[2],&flags[3],&flags[4],&flags[5],&flags[6])
 if err!=nil{return fmt.Sprintf("query (%T)",err)}
 return fmt.Sprintf("login=%t group=%t direct=%t closure=%t functions=%t login_no_authority=%t exact_group_acl=%t",flags[0],flags[1],flags[2],flags[3],flags[4],flags[5],flags[6])
}

func codexMRestoreIdentityConnect(ctx context.Context,admin *pgxpool.Pool)(int,error){
 tx,err:=admin.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{return 0,err};defer tx.Rollback(ctx)
 if _,err=tx.Exec(ctx,`SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:codexm:usuarios:identity-connect',0))`);err!=nil{return 0,err}
 var pending []string
 for _,group:=range []string{"vec_identidad_sesiones_v1_registrador","vec_identidad_sesiones_v1_revalidador"}{
  var safe bool;var aclCount int
  if tx.QueryRow(ctx,`SELECT current_database()='postgres' AND NOT(r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolinherit OR r.rolreplication OR r.rolbypassrls)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=r.oid),
 (SELECT count(*) FROM pg_catalog.pg_database b CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) a WHERE b.datname=current_database() AND a.grantee=r.oid)
 FROM pg_catalog.pg_roles r WHERE r.rolname=$1`,group).Scan(&safe,&aclCount)!=nil||!safe{return 0,fmt.Errorf("identity group preimage")}
  if aclCount==0{
   expectedFunctions:=[]string{"vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)","vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)"}
   if group=="vec_identidad_sesiones_v1_revalidador"{expectedFunctions=[]string{"vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)","vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)","vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)"}}
   var exactDependencies bool
   if tx.QueryRow(ctx,`SELECT count(*)=1+cardinality($2::text[]) AND bool_and(d.deptype='a' AND d.objsubid=0 AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=current_database()) AND
 ((d.classid='pg_catalog.pg_namespace'::regclass AND d.objid='vec_identidad_sesiones_v1'::regnamespace) OR
 (d.classid='pg_catalog.pg_proc'::regclass AND d.objid IN (SELECT pg_catalog.to_regprocedure(f) FROM unnest($2::text[]) f))))
 FROM pg_catalog.pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=$1::regrole`,group,expectedFunctions).Scan(&exactDependencies)!=nil||!exactDependencies{return 0,fmt.Errorf("identity group has additional authority")}
   pending=append(pending,group);continue
  }
  var exact bool
  if tx.QueryRow(ctx,`SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_catalog.pg_database b CROSS JOIN LATERAL pg_catalog.aclexplode(b.datacl) a WHERE b.datname=current_database() AND a.grantee=$1::regrole`,group).Scan(&exact)!=nil||!exact{return 0,fmt.Errorf("identity group database ACL differs")}
 }
 for _,group:=range pending{if _,err=tx.Exec(ctx,`GRANT CONNECT ON DATABASE postgres TO `+pgx.Identifier{group}.Sanitize());err!=nil{return 0,err}}
 if tx.Commit(ctx)!=nil{return 0,fmt.Errorf("identity CONNECT commit")};return len(pending),nil
}

func codexMSession(ctx context.Context,c configuracionUsuariosPreferenciasDesarrollo,derivador *derivadorIdentidadOperacionDesarrollo,certificate *x509.Certificate,chain []*x509.Certificate)(core.VinculoAutenticacionActorV2,error){
 var empty core.VinculoAutenticacionActorV2
 reg,_,err:=abrirPoolRutasDietas(ctx,c.DSNRegistroIdentidad,"vec_identidad_sesiones_v1_registrador");if err!=nil{return empty,fmt.Errorf("open-registro (%T)",err)};defer reg.Close()
 rev,_,err:=abrirPoolRutasDietas(ctx,c.DSNRevalidacionIdentidad,"vec_identidad_sesiones_v1_revalidador");if err!=nil{return empty,fmt.Errorf("open-revalidacion (%T)",err)};defer rev.Close()
 contexts,_,err:=abrirPoolRutasDietas(ctx,c.DSNContexto,"vec_contexto_actor_v1_runtime");if err!=nil{return empty,fmt.Errorf("open-contexto (%T)",err)};defer contexts.Close()
 registro,err:=postgresidentidad.NuevoRegistroSesionesPostgreSQL(ctx,reg,rev,&seudonimizadorSesionDesarrollo{derivador:derivador},espacioIdentidadSesionDesarrollo,dominioIdentidadSesionDesarrollo);if err!=nil{return empty,fmt.Errorf("construct-registro: reg{%s} rev{%s}",
 codexMIdentityFlags(ctx,reg,"vec_identidad_sesiones_v1_registrador",[]string{"vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)","vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)"}),
 codexMIdentityFlags(ctx,rev,"vec_identidad_sesiones_v1_revalidador",[]string{"vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)","vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)","vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)"}))}
 revalidador,err:=postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx,rev);if err!=nil{return empty,fmt.Errorf("construct-revalidador (%T)",err)}
 resolutor,err:=contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx,contexts);if err!=nil{return empty,fmt.Errorf("construct-resolutor (%T)",err)}
 servicio,err:=vecapp.NuevoServicioContextoActorProductivoV2(resolutor,contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(),relojRutasDietas{});if err!=nil{return empty,fmt.Errorf("construct-servicio (%T)",err)}
 autoridad,err:=vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio);if err!=nil{return empty,fmt.Errorf("construct-autoridad (%T)",err)}
 instance,err:=nonceRutasDietas();if err!=nil{return empty,fmt.Errorf("nonce (%T)",err)}
 base:=&autoridadRutasDietasDesarrollo{registro:registro,revalidador:revalidador,contextos:autoridad,reloj:relojRutasDietas{},instancia:instance}
 users:=&autoridadPreferenciasUsuariosDesarrollo{base:base,reloj:relojRutasDietas{},superficie:c.Superficie}
 request:=httptest.NewRequest("GET","https://localhost/instalacion-privada-usuarios",nil)
 request.TLS=&tls.ConnectionState{VerifiedChains:[][]*x509.Certificate{chain},PeerCertificates:[]*x509.Certificate{certificate}}
 vinculo,_,err:=users.resolverSesion(request,c.Cuentas[0],time.Now().UTC().Truncate(time.Microsecond));if err!=nil{return empty,fmt.Errorf("resolve-session (%T)",err)};return vinculo,nil
}

func codexMModuleHistory(ctx context.Context,admin *pgxpool.Pool)(string,error){
 rows,err:=admin.Query(ctx,`SELECT schemaname,tablename FROM pg_catalog.pg_tables
 WHERE schemaname='vec_contratacion_temporal' OR schemaname='vec_bolsa' OR schemaname LIKE 'vec_bolsa_%'
 ORDER BY schemaname,tablename`);if err!=nil{return "",err}
 var tables [][2]string
 for rows.Next(){var item [2]string;if rows.Scan(&item[0],&item[1])!=nil{rows.Close();return "",fmt.Errorf("module inventory")};tables=append(tables,item)}
 if rows.Err()!=nil{rows.Close();return "",fmt.Errorf("module inventory")};rows.Close()
 if len(tables)==0{return "",fmt.Errorf("module inventory empty")}
 hash:=sha256.New()
 for _,item:=range tables{var fingerprint string
  query:=`SELECT md5(COALESCE(string_agg(row_to_json(t)::text,E'\n' ORDER BY row_to_json(t)::text),'')) FROM `+pgx.Identifier{item[0],item[1]}.Sanitize()+` t`
  if admin.QueryRow(ctx,query).Scan(&fingerprint)!=nil{return "",fmt.Errorf("module history")}
  fmt.Fprintf(hash,"%s.%s:%s\n",item[0],item[1],fingerprint)
 }
 return hex.EncodeToString(hash.Sum(nil)),nil
}

func TestCodexMInstallUsers(t *testing.T){
 if os.Getenv("VEC_CODEXM_USUARIOS_INSTALAR")!="CLON_PRIVADO_REVISADO"{t.Fatal("installation guard")}
 root:=os.Getenv("VEC_CODEXM_STATE");material:=filepath.Join(root,"material")
 planBytes,err:=os.ReadFile(filepath.Join(root,"usuarios-plan.json"));if err!=nil{t.Fatal("installation plan")}
 var plan struct{InstallAt string `json:"install_at"`;SystemID string `json:"system_id"`;PGPort uint16 `json:"pg_port"`;Logins []struct{Name string `json:"name"`;Group string `json:"group"`} `json:"logins"`}
 if json.Unmarshal(planBytes,&plan)!=nil{t.Fatal("installation plan contract")}
 _,err=time.Parse(time.RFC3339Nano,plan.InstallAt);if err!=nil{t.Fatal("installation time")}
 historic,err:=time.Parse(time.RFC3339,"2026-09-29T04:00:00Z");if err!=nil{t.Fatal("historic time")}
 ctx,cancel:=context.WithTimeout(context.Background(),120*time.Second);defer cancel()
 adminCfg,err:=pgxpool.ParseConfig(os.Getenv("VEC_CODEXM_ADMIN_DSN"));if err!=nil||adminCfg.ConnConfig.Host!="127.0.0.1"||adminCfg.ConnConfig.Port!=plan.PGPort||len(adminCfg.ConnConfig.Fallbacks)!=0||validarTLSPostgreSQLBorradores(&adminCfg.ConnConfig.Config,false)!=nil{t.Fatal("admin TLS destination")}
 admin,err:=pgxpool.NewWithConfig(ctx,adminCfg);if err!=nil{t.Fatal("admin pool")};defer admin.Close()
 var systemID string;if admin.QueryRow(ctx,`SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&systemID)!=nil||systemID!=plan.SystemID{t.Fatal("clone identity")}
 gobierno,_,err:=abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx,os.Getenv("VEC_CT_GOBIERNO_DATABASE_URL"),"vec-codexm-usuarios-instalacion",rolGobiernoPostgreSQLContratacionTemporalDesarrollo);if err!=nil{t.Fatal("government pool")};defer gobierno.Close()
 materialHMAC,err:=cargarMaterialIdempotenciaDesarrollo(material,filepath.Join(material,"idempotencia/configuracion.json"));if err!=nil{t.Fatal("HMAC material")}
 derivador,err:=nuevoDerivadorIdentidadOperacionDesarrollo(&materialHMAC);if err!=nil{t.Fatal("HMAC derivation")};defer derivador.borrar()
 revision,catalogue,err:=catalogoPoliticasProvisionPreferenciasHito1(ctx,gobierno);if err!=nil{t.Fatal("policy catalogue")}
 ca,err:=leerCertificadoProvisionPreferenciasHito1(filepath.Join(material,"ca/ca.crt"));if err!=nil{t.Fatal("CA")};roots:=x509.NewCertPool();roots.AddCert(ca)
 moduleBefore,err:=codexMModuleHistory(ctx,admin);if err!=nil{t.Fatal("module history preimage")}
 // The technical identities are H1 inputs, installed only when absent. All
 // existing identities must have the exact one-group, non-administrative ACL.
 if len(plan.Logins)!=16{t.Fatal("technical Users role plan")}
 txRoles,err:=admin.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{t.Fatal("technical role transaction")};defer txRoles.Rollback(ctx)
 if _,err=txRoles.Exec(ctx,`SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:codexm:usuarios:logins',0))`);err!=nil{t.Fatal("technical role lock")}
 var missing []int
 for i,entry:=range plan.Logins{
  var exists bool;if txRoles.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=$1)`,entry.Name).Scan(&exists)!=nil{t.Fatal("technical role preimage")}
  if !exists{missing=append(missing,i);continue}
  var valid bool
  if txRoles.QueryRow(ctx,`SELECT r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE m.member=r.oid AND g.rolname=$2 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 FROM pg_catalog.pg_roles r WHERE r.rolname=$1`,entry.Name,entry.Group).Scan(&valid)!=nil||!valid{t.Fatal("existing technical role privileges diverged")}
 }
 var originals,currents []configuracionUsuariosPreferenciasDesarrollo
 var prior []cuentaProvisionPreferenciasHito1
 var instances []core.InstantaneaAutorizacion
 var certificates []*x509.Certificate
 var chains [][]*x509.Certificate
 var canonical []string
 for index,surface:=range []string{"interna","externa"}{
  original,err:=codexMConfiguration(filepath.Join(root,"usuarios-before",surface+"-original.json"));if err!=nil{t.Fatal("original H1 configuration")}
  current,err:=codexMConfiguration(filepath.Join(material,"identidad","usuarios-preferencias-"+surface+".json"));if err!=nil{t.Fatal("current H1 configuration")}
  if original.Superficie!=current.Superficie||original.Cuentas[0].Sujeto!=current.Cuentas[0].Sujeto||original.Cuentas[0].PerfilRef!=current.Cuentas[0].PerfilRef||original.Cuentas[0].CertificadoSHA256!=current.Cuentas[0].CertificadoSHA256{t.Fatal("identity preimage changed")}
  certificateFile:="cliente";identityFile:="identidad";role:="tecnico_rrhh";if index==1{certificateFile="intervencion";identityFile="intervencion";role="intervencion"}
  cert,err:=leerCertificadoProvisionPreferenciasHito1(filepath.Join(material,"mtls",certificateFile+".crt"));if err!=nil{t.Fatal("certificate")}
  verified,err:=cert.Verify(x509.VerifyOptions{Roots:roots,KeyUsages:[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}});if err!=nil||len(verified)!=1{t.Fatal("certificate trust")}
  identity,err:=cargarIdentidadDesarrollo(filepath.Join(material,"identidad",identityFile+".json"),cert,role);if err!=nil{t.Fatal("identity material")}
  otherCertFile:="intervencion";otherIdentityFile:="intervencion";otherRole:="intervencion";if index==1{otherCertFile="cliente";otherIdentityFile="identidad";otherRole="tecnico_rrhh"}
  otherCert,err:=leerCertificadoProvisionPreferenciasHito1(filepath.Join(material,"mtls",otherCertFile+".crt"));if err!=nil{t.Fatal("other certificate")}
  otherIdentity,err:=cargarIdentidadDesarrollo(filepath.Join(material,"identidad",otherIdentityFile+".json"),otherCert,otherRole);if err!=nil{t.Fatal("other identity")}
  var expected *resolvedorIdentidadDesarrollo
  if index==0{expected,err=nuevoResolvedorIdentidadDesarrollo(identity,otherIdentity)}else{expected,err=nuevoResolvedorIdentidadDesarrollo(otherIdentity,identity)}
  if err!=nil{t.Fatal("identity resolver")}
  if _,err=cuentasPreferenciasAcreditadas(expected,current);err!=nil{t.Fatal("subject/certificate mismatch")}
  old,err:=construirCuentaProvisionPreferenciasHito1(original,historic);if err!=nil{t.Fatal("original person context")}
  definition,err:=construirCuentaProvisionPreferenciasHito1(original,historic);if err!=nil{t.Fatal("original role context")}
  instant,err:=instantaneaProvisionPreferenciasHito1(definition,historic,revision,catalogue);if err!=nil||!instant.AsignacionPerfil.VigenteEn(time.Now().UTC()){t.Fatal("historic role validity")}
  account,err:=codexMLookup(ctx,admin,derivador,original,current);if err!=nil{t.Fatal("identity aliases diverged")}
  var personVersions,personCurrent,profileVersions,profileCurrent int
  if admin.QueryRow(ctx,`SELECT
 (SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones WHERE persona_ref=$1),
 (SELECT count(*) FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref=$1),
 (SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=$2),
 (SELECT count(*) FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=$2)`,old.personaRef,current.Cuentas[0].PerfilRef).Scan(&personVersions,&personCurrent,&profileVersions,&profileCurrent)!=nil{t.Fatal("context preimage")}
  if (personVersions!=0||personCurrent!=0||profileVersions!=0||profileCurrent!=0)&&(personVersions!=1||personCurrent!=1||profileVersions!=1||profileCurrent!=1){t.Fatal("partial historic person/profile")}
  if account==""&&personVersions>0{t.Fatal("person/profile exist without demonstrable canonical identity")}
  accountCount:=0;if account!=""{accountCount=1}
  t.Logf("preimage surface=%s account=%d person_versions=%d person_current=%d profile_versions=%d profile_current=%d",surface,accountCount,personVersions,personCurrent,profileVersions,profileCurrent)
  roleHash,_:=instant.VersionRol.HuellaSHA256();controlHash,_:=instant.ControlVigenciaVersionRol.HuellaSHA256()
  roleDocument,_:=json.Marshal(instant.VersionRol);controlDocument,_:=json.Marshal(instant.ControlVigenciaVersionRol)
  var roleN,controlN,controlCurrentN int
  if admin.QueryRow(ctx,`SELECT
 (SELECT count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref=$1),
 (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol WHERE version_rol_ref=$1),
 (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref=$1)`,instant.VersionRol.Referencia()).Scan(&roleN,&controlN,&controlCurrentN)!=nil{t.Fatal("role preimage")}
  if (roleN!=0||controlN!=0||controlCurrentN!=0)&&(roleN!=1||controlN!=1||controlCurrentN!=1){t.Fatal("partial historic role")}
  if roleN==1{var exact bool
   if admin.QueryRow(ctx,`SELECT r.huella_sha256=$2 AND r.documento=$3::jsonb AND c.huella_sha256=$4 AND c.documento=$5::jsonb AND c.estado='habilitada' AND c.revision=1 AND a.revision=1
 FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
 WHERE r.version_rol_ref=$1`,instant.VersionRol.Referencia(),roleHash,roleDocument,controlHash,controlDocument).Scan(&exact)!=nil||!exact{t.Fatal("historic role is divergent or withdrawn")}
  }
  var assignmentCount int;if admin.QueryRow(ctx,`SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=$1`,current.Cuentas[0].PerfilRef).Scan(&assignmentCount)!=nil||assignmentCount>1{t.Fatal("assignment preimage")}
  if assignmentCount==1{
   tx,err:=gobierno.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable});if err!=nil{t.Fatal("assignment inspection")}
   if _,err=tx.Exec(ctx,`SET LOCAL ROLE vec_autorizacion_propietario`);err!=nil{tx.Rollback(ctx);t.Fatal("assignment authority")}
   actual,exists,err:=leerAsignacionActualPostgreSQLDesarrollo(ctx,tx,current.Cuentas[0].PerfilRef)
   hash,_:=instant.AsignacionPerfil.HuellaSHA256()
   if err!=nil||!exists||actual.huella!=hash||actual.principalID!=old.personaRef||!asignacionActualOperativaPostgreSQLDesarrollo(actual,"habilitada",time.Now().UTC()){tx.Rollback(ctx);t.Fatal("existing assignment is not exact active H1")}
   tx.Rollback(ctx)
  }
  originals=append(originals,original);currents=append(currents,current);prior=append(prior,old);instances=append(instances,instant);canonical=append(canonical,account);certificates=append(certificates,cert);chains=append(chains,verified[0])
 }
 // Both surfaces were checked before the first authority call. All IDs and
 // installation time are stable, so a partial failure is recovered by replay.
 for _,index:=range missing{entry:=plan.Logins[index]
  if _,err=txRoles.Exec(ctx,`CREATE ROLE `+pgx.Identifier{entry.Name}.Sanitize()+` LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`);err!=nil{t.Fatal("create absent technical role")}
  if _,err=txRoles.Exec(ctx,`GRANT `+pgx.Identifier{entry.Group}.Sanitize()+` TO `+pgx.Identifier{entry.Name}.Sanitize()+` WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`);err!=nil{t.Fatal("grant one technical capability")}
 }
 if txRoles.Commit(ctx)!=nil{t.Fatal("technical role commit")}
 connectRestored,err:=codexMRestoreIdentityConnect(ctx,admin);if err!=nil{t.Fatal("identity CONNECT preimage")};t.Logf("identity CONNECT restored=%d",connectRestored)
 if crearOComprobarLoginProvisionadorHito1(ctx,admin)!=nil{t.Fatal("identity provisioner authority")}
 provisionCfg:=adminCfg.Copy();provisionCfg.ConnConfig.User=loginProvisionadorHito1;provisionCfg.ConnConfig.Password=""
 provisioner,err:=pgxpool.NewWithConfig(ctx,provisionCfg);if err!=nil{t.Fatal("identity provisioner pool")};defer provisioner.Close()
 for i,current:=range currents{
  if canonical[i]==""{
   account,err:=provisionarYRegistrarAliasIdentidadHito1(ctx,provisioner,admin,derivador,originals[i]);if err!=nil{t.Fatal("account authority")};canonical[i]=account
  }
  current.Cuentas[0].CuentaRef=canonical[i]
  contextResult:=prior[i].contexto;operation:=prior[i].operacionRef
  if canonical[i]!=prior[i].cuenta.CuentaRef{contextResult,operation,err=nuevoResultadoCuentaConciliadaHito1(prior[i],canonical[i]);if err!=nil{t.Fatal("canonical context")}}
  if err=publicarResultadoContextoPostgreSQLDesarrollo(ctx,gobierno,contextResult,operation);err!=nil{t.Fatal("context publication exact replay")}
  vinculo,err:=codexMSession(ctx,current,derivador,certificates[i],chains[i]);if err!=nil{t.Fatalf("real 120s session: %s",err.Error())}
  authority:=autoridadPostgreSQLDesarrollo{pool:gobierno,vinculo:vinculo,prefijoBloqueo:"vec:hito1:usuarios:preferencias:",actoControlRol:"acto:hito1:usuarios:preferencias:control:"+string(current.Superficie),actoAsignacion:"acto:hito1:usuarios:preferencias:asignacion:"+string(current.Superficie),actoSesion:"acto:codexm:usuarios:instalacion:"+string(current.Superficie),exigirOrigenOperativo:true}
  if err=authority.PublicarInstantanea(ctx,instances[i]);err!=nil{t.Fatal("authorization publication exact replay")}
  if cuentaIdentidadActivaHito1(ctx,admin,canonical[i])!=nil{t.Fatal("canonical account validity")}
 }
 moduleAfter,err:=codexMModuleHistory(ctx,admin);if err!=nil||moduleAfter!=moduleBefore{t.Fatal("module history changed")}
 result:=map[string]any{"module_history_before":moduleBefore,"module_history_after":moduleAfter,"version":1,"install_at":plan.InstallAt,"accounts":canonical,"persons":[]string{prior[0].personaRef,prior[1].personaRef},"profiles":[]string{currents[0].Cuentas[0].PerfilRef,currents[1].Cuentas[0].PerfilRef},"references_preserved":true,"technical_logins_checked":len(plan.Logins),"technical_logins_created":len(missing),"identity_connect_restored":connectRestored}
 encoded,err:=json.Marshal(result);if err!=nil{t.Fatal("result")}
 if err=os.WriteFile(filepath.Join(root,"usuarios-authority-result.json"),encoded,0600);err!=nil{t.Fatal("private result")}
 _=config.Config{}
 t.Log("two synthetic canonical Users identities installed by existing authorities")
}
'''


def login_plan(source, configs):
    text = source.decode()
    expression = re.compile(r"CREATE ROLE ([a-z][a-z0-9_]*) LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;\s*GRANT ([a-z][a-z0-9_]*) TO \1 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;")
    pairs = expression.findall(text)
    remainder = expression.sub("", text)
    if len(pairs) != 16 or re.sub(r"\s+", "", remainder) != "BEGIN;COMMIT;" or len(dict(pairs)) != 16:
        fail("The private H1 technical roles contract changed.")
    configured = {}
    base = {"dsn_registro_identidad": "vec_identidad_sesiones_v1_registrador",
            "dsn_revalidacion_identidad": "vec_identidad_sesiones_v1_revalidador",
            "dsn_contexto": "vec_contexto_actor_v1_runtime", "dsn_fuente_autorizacion": "vec_autorizacion_fuente",
            "dsn_registro_autorizacion": "vec_autorizacion_registro", "dsn_motivos": "vec_autorizacion_motivos_evaluador"}
    for surface, config in configs.items():
        role_surface = "interno" if surface == "interna" else "externo"
        expected = dict(base, dsn_usuarios="vec_usuarios_ejecutor_" + role_surface,
                        dsn_usuarios_frontera="vec_usuarios_registrador_frontera_" + role_surface)
        for field, group in expected.items():
            name = urlsplit(config[field]).username
            if not name or name in configured:
                fail("Technical Users accounts are not separated.")
            configured[name] = group
    if configured != dict(pairs):
        fail("Configured Users DSN do not match the preserved technical roles contract.")
    return [{"name": name, "group": group} for name, group in pairs]


def provision(repo, container, state, material, pg_port, engine="docker"):
    repo, state, material = Path(repo), Path(state), Path(material)
    module_path = Path(__file__).with_name("clon_material.py")
    spec = importlib.util.spec_from_file_location("material_contract", module_path)
    material_module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(material_module)
    inspect = json.loads(material_module.run([engine, "inspect", container]))[0]
    material_module.validate_container(inspect, pg_port)
    marker = json.loads(private(state / "DB_READY.json"))
    if marker.get("propietario") != "Codex-M" or marker.get("contenedor") != container or marker.get("puerto_pg") != pg_port:
        fail("Clone readiness marker does not match the owned destination.")
    if (state / "runtime-process.json").exists():
        fail("Stop the owned application before provisioning Users.")
    source = state / ("source-" + marker["commit"])
    if not source.is_dir():
        fail("The pinned application source is not available.")
    env = json.loads(private(state / "runtime-config.json"))
    runtime_path = Path(__file__).with_name("clon_runtime.py")
    spec = importlib.util.spec_from_file_location("runtime_contract", runtime_path)
    runtime = importlib.util.module_from_spec(spec);spec.loader.exec_module(runtime)
    runtime.validate_state(repo.resolve(), state)
    for name, value in env.items():
        if name.endswith("_DATABASE_URL"):
            runtime.validate_dsn(value, pg_port, state)
    original_root = Path.home() / ".local/state/vec-clon/material-hito1/identidad"
    before_dir = state / "usuarios-before"
    before_dir.mkdir(mode=0o700, exist_ok=True)
    preimages = []
    for surface in ("interna", "externa"):
        original = private(original_root / (".usuarios-preferencias-" + surface + ".hito1-antes.json"))
        current_path = material / "identidad" / ("usuarios-preferencias-" + surface + ".json")
        current = private(current_path)
        path = before_dir / (surface + "-original.json")
        if path.exists() and private(path) != original:
            fail("Original H1 person preimage changed.")
        if not path.exists():
            atomic(path, original)
        preimages.append((current_path, current))
    configurations = {surface: json.loads(private(material / "identidad" / ("usuarios-preferencias-" + surface + ".json")))
                      for surface in ("interna", "externa")}
    roles_input = private(Path.home() / ".local/state/vec-clon/roles-usuarios-hito1.sql")
    roles = login_plan(roles_input, configurations)
    system_id = material_module.run([engine, "exec", "--user", "postgres", container, "psql", "-XAt", "-d", "postgres", "-c", "SELECT system_identifier::text FROM pg_catalog.pg_control_system()"])
    plan_path = state / "usuarios-plan.json"
    if not plan_path.exists():
        plan = {"install_at": datetime.now(timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z"),
                "system_id": system_id.decode().strip(), "pg_port": pg_port,
                "preimages": [sha(current) for _, current in preimages]}
        atomic(plan_path, json.dumps(plan).encode())
    plan = json.loads(private(plan_path))
    if plan["system_id"] != system_id.decode().strip() or plan["pg_port"] != pg_port:
        fail("Users installation plan belongs to a different clone.")
    if "logins" in plan and plan["logins"] != roles:
        fail("The technical role preimage changed after planning the installation.")
    plan["logins"] = roles
    plan["technical_roles_source_sha256"] = sha(roles_input)
    atomic(plan_path, json.dumps(plan).encode())
    # The administrative session uses only the owned clone's loopback trust
    # rule and verified TLS; it never inherits a password or remote endpoint.
    admin = "postgres://postgres@127.0.0.1:" + str(pg_port) + "/postgres?" + urlencode({"sslmode": "verify-full", "sslrootcert": str(material / "pg/ca.crt")})
    installation = state / "usuarios-source"
    if installation.exists():
        shutil.rmtree(installation)
    shutil.copytree(source, installation)
    harness = installation / "internal/app/bootstrap/codexm_usuarios_instalacion_test.go"
    harness.write_text(make_harness(repo))
    go = runtime.local_go(source, None)
    with tempfile.TemporaryDirectory(prefix="vec-codexm-pref-") as temporary:
        execution = {"PATH": str(go.parent) + ":/usr/bin:/bin", "HOME": str(state), "TMPDIR": temporary,
                     "GOCACHE": "/dev/shm/go-build", "GOPATH": str(Path.home() / "go"),
                     "GOMODCACHE": str(Path.home() / "go/pkg/mod"), "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
                     "VEC_CODEXM_USUARIOS_INSTALAR": "CLON_PRIVADO_REVISADO", "VEC_CODEXM_STATE": str(state),
                     "VEC_CODEXM_ADMIN_DSN": admin, "VEC_CT_GOBIERNO_DATABASE_URL": env["VEC_CT_GOBIERNO_DATABASE_URL"]}
        descriptor = os.open(state / "usuarios-install.log", os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, "wb") as log:
            result = subprocess.run([str(go), "test", "-p", "32", "./internal/app/bootstrap", "-run", "^TestCodexM(InstallUsers|HistoricalUsersDTOAfterSessionExpiry)$", "-count=1", "-v"],
                                    cwd=installation, env=execution, stdout=log, stderr=subprocess.STDOUT, timeout=240)
        if result.returncode:
            fail("Users authority installation failed; inspect the private usuarios-install.log.")
    result = json.loads(private(state / "usuarios-authority-result.json"))
    if len(result.get("accounts", [])) != 2 or result["accounts"][0] == result["accounts"][1]:
        fail("Users authority returned ambiguous accounts.")
    updates = []
    for (path, before), canonical in zip(preimages, result["accounts"]):
        if private(path) != before:
            fail("Private account configuration changed concurrently.")
        updates.append((path, account_only(before, canonical)))
    for path, updated in updates:
        atomic(path, updated)
    atomic(state / "usuarios-result.json", json.dumps(result).encode())
    return {"env": {}, "profiles": {"usuarios": result}, "blockers": []}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--container", required=True)
    parser.add_argument("--state", required=True, type=Path)
    parser.add_argument("--pg-port", required=True, type=int)
    args = parser.parse_args()
    result = provision(args.repo, args.container, args.state, args.state / "material", args.pg_port)
    print(json.dumps({"status": "installed_material_seal_pending", "blockers": result["blockers"]}))


if __name__ == "__main__":
    try:
        main()
    except (UsersError, OSError, ValueError, KeyError, subprocess.TimeoutExpired):
        print("Private Users installation stopped; no automatic identity or permission repair.", file=sys.stderr)
        sys.exit(1)
