package separacionportales

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Todas las conexiones y claves de estas pruebas son sintéticas.

func escribir(t *testing.T, raiz, relativa, contenido string) {
	t.Helper()
	ruta := filepath.Join(raiz, filepath.FromSlash(relativa))
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
}

// materialSintetico prepara un directorio mínimo de un portal con claves
// propias (el contenido se distingue por el portal).
func materialSintetico(t *testing.T, p Portal) string {
	t.Helper()
	raiz := filepath.Join(t.TempDir(), "material-"+string(p))
	if err := os.Mkdir(raiz, 0o700); err != nil {
		t.Fatal(err)
	}
	escribir(t, raiz, FicheroMarcaPortal, `{"version":1,"portal":"`+string(p)+`"}`)
	escribir(t, raiz, "manifiesto.json", `{"version":1}`)
	escribir(t, raiz, "ca/ca.crt", certificadoCASintetico(t))
	escribir(t, raiz, "tls/servidor.crt", "certificado servidor")
	escribir(t, raiz, "tls/servidor.key", "clave tls "+string(p))
	escribir(t, raiz, "idempotencia/g1-localizador.bin", "idem "+string(p))
	escribir(t, raiz, "desarrollo.env", "VEC_EXECUTION_PROFILE=desarrollo\nVEC_TLS_KEY_FILE="+raiz+"/tls/servidor.key\n")
	if p == PortalInterno {
		escribir(t, raiz, "kms/clave-maestra.bin", "kms "+string(p))
		escribir(t, raiz, "tsa/clave-hmac.bin", "tsa "+string(p))
		escribir(t, raiz, "mtls/cliente.crt", "certificado rrhh")
		escribir(t, raiz, "identidad/identidad.json", `{"version":1}`)
		escribir(t, raiz, "identidad/cronos-empleado.json", `{"dsn":"postgresql://vec_cronos_sintetico:x@127.0.0.1:5432/vec"}`)
	} else {
		escribir(t, raiz, "mtls/candidato.crt", "certificado candidato")
		escribir(t, raiz, "identidad/bolsa-candidato.json", `{"version":1}`)
		escribir(t, raiz, "identidad/usuarios-preferencias-externa.json", `{"dsn_usuarios":"postgresql://vec_pref_sintetico_e:x@127.0.0.1:5432/vec"}`)
	}
	return raiz
}

func certificadoCASintetico(t *testing.T) string {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificado := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA sintetica de pruebas"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, certificado, certificado, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestComprobarSeparacionRechazaCAMismaConOtroFormatoPEM(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	ca, err := os.ReadFile(filepath.Join(interno, "ca", "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	escribir(t, externo, "ca/ca.crt", "\n"+strings.ReplaceAll(string(ca), "\n", "\r\n")+"\n")
	_, err = ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo})
	if got := motivo(t, err); got != "ca/ca.crt" {
		t.Fatalf("se esperaba rechazar CA repetida: %s", got)
	}
}

func TestComprobarSeparacionRechazaCANoValida(t *testing.T) {
	for _, invalida := range []string{"", "certificado no valido", certificadoCASintetico(t) + certificadoCASintetico(t)} {
		interno := materialSintetico(t, PortalInterno)
		externo := materialSintetico(t, PortalExterno)
		escribir(t, externo, "ca/ca.crt", invalida)
		_, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo})
		if got := motivo(t, err); got != "ca/ca.crt" {
			t.Fatalf("CA no valida aceptada: %s", got)
		}
	}
}

func motivo(t *testing.T, err error) string {
	t.Helper()
	var e *ErrorSeparacion
	if !errors.As(err, &e) || !errors.Is(err, ErrSeparacionPortales) {
		t.Fatalf("se esperaba un rechazo de separacion y llego %v", err)
	}
	return e.Elemento
}

func TestParsearSoloAdmiteValoresCerrados(t *testing.T) {
	for _, valido := range []string{"", "interno", "externo"} {
		if _, err := Parsear(valido); err != nil {
			t.Fatalf("%q deberia admitirse: %v", valido, err)
		}
	}
	for _, invalido := range []string{"Interno", " externo", "ambos", "combinado", "publico"} {
		if _, err := Parsear(invalido); !errors.Is(err, ErrPortalDesconocido) {
			t.Fatalf("%q deberia rechazarse", invalido)
		}
	}
}

func TestPortalCombinadoNoCambiaNada(t *testing.T) {
	entorno := Entorno{Variables: []string{
		"VEC_CT_DATABASE_URL=postgresql://vec_ct:x@127.0.0.1/vec",
		"VEC_EXTERNO_BOLSA_DATABASE_URL=postgresql://vec_ext:x@127.0.0.1/vec",
		"PGPASSWORD=sintetica",
	}}
	if err := ComprobarEntorno(PortalCombinado, entorno); err != nil {
		t.Fatalf("el portal combinado no debe comprobar el entorno: %v", err)
	}
	material := filepath.Join(t.TempDir(), "sin-marca")
	if err := os.Mkdir(material, 0o700); err != nil {
		t.Fatal(err)
	}
	escribir(t, material, "ca/ca.key", "clave ca")
	if err := ComprobarMaterial(PortalCombinado, material); err != nil {
		t.Fatalf("el portal combinado no debe comprobar el material historico: %v", err)
	}
}

func TestProcesoExternoNoPuedeRecibirConexionesInternas(t *testing.T) {
	for _, variable := range []string{
		"VEC_CT_DATABASE_URL=postgresql://vec_ct_o207_runtime:x@127.0.0.1/vec",
		"VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL=postgresql://vec_bolsa:x@127.0.0.1/vec",
		"VEC_CALENDARIOS_DATABASE_URL=postgresql://vec_cal:x@127.0.0.1/vec",
		"VEC_PRINCIPAL_ADMIN_DATABASE_URL=postgresql://postgres:x@127.0.0.1/vec",
		"VEC_NOMBRE_RARO=postgresql://vec_oculto:x@127.0.0.1/vec",
		"VEC_OTRA=host=127.0.0.1 user=vec_oculto password=x",
		"VEC_OTRA_MAS=host=127.0.0.1 user=vec_oculto sslkey=/clave",
	} {
		err := ComprobarEntorno(PortalExterno, Entorno{Variables: []string{"HOME=/tmp", variable}})
		nombre, _, _ := strings.Cut(variable, "=")
		if motivo(t, err) != nombre {
			t.Fatalf("se esperaba rechazar %s y llego %v", nombre, err)
		}
		if strings.Contains(err.Error(), "x@") || strings.Contains(err.Error(), "password") {
			t.Fatalf("el error no debe contener el valor: %v", err)
		}
	}
}

func TestProcesoExternoAdmiteSusConexionesPropias(t *testing.T) {
	err := ComprobarEntorno(PortalExterno, Entorno{Variables: []string{
		"VEC_EXTERNO_USUARIOS_DATABASE_URL=postgresql://vec_usuarios_ext:x@127.0.0.1/vec",
		"VEC_TLS_KEY_FILE=/material-externo/tls/servidor.key",
		"VEC_EXECUTION_PROFILE=desarrollo",
		"VEC_BOLSA_PORTAL_CANDIDATO_ENABLED=true",
	}})
	if err != nil {
		t.Fatalf("el portal externo debe admitir sus conexiones: %v", err)
	}
}

func TestProcesoExternoNoPuedeRecibirSecretosInternos(t *testing.T) {
	for _, variable := range []string{
		"VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR=/custodia",
		"VEC_FIRMA_VERIFICACION_TOKEN_FILE=/token",
		"VEC_CT_INCORPORACION_V2_FILE=/incorporacion.json",
		"VEC_NUEVO_SERVICIO_TOKEN=abc",
	} {
		nombre, _, _ := strings.Cut(variable, "=")
		if got := motivo(t, ComprobarEntorno(PortalExterno, Entorno{Variables: []string{variable}})); got != nombre {
			t.Fatalf("se esperaba rechazar %s, llego %s", nombre, got)
		}
	}
}

func TestProcesoInternoNoPuedeRecibirConexionesNiCapacidadesExternas(t *testing.T) {
	for _, variable := range []string{
		"VEC_EXTERNO_USUARIOS_DATABASE_URL=postgresql://vec_usuarios_ext:x@127.0.0.1/vec",
		"VEC_EXTERNO_CLAVE_FILE=/externo/clave",
		"VEC_BOLSA_PORTAL_CANDIDATO_ENABLED=true",
	} {
		nombre, _, _ := strings.Cut(variable, "=")
		if got := motivo(t, ComprobarEntorno(PortalInterno, Entorno{Variables: []string{variable}})); got != nombre {
			t.Fatalf("se esperaba rechazar %s, llego %s", nombre, got)
		}
	}
	err := ComprobarEntorno(PortalInterno, Entorno{Variables: []string{
		"VEC_CT_DATABASE_URL=postgresql://vec_ct:x@127.0.0.1/vec",
		"VEC_BOLSA_PORTAL_CANDIDATO_ENABLED=false",
		"VEC_FIRMA_VERIFICACION_TOKEN_FILE=/token",
	}})
	if err != nil {
		t.Fatalf("el interno conserva sus conexiones: %v", err)
	}
}

func TestProcesoSeparadoRechazaCredencialesImplicitasDePostgreSQL(t *testing.T) {
	for _, p := range []Portal{PortalInterno, PortalExterno} {
		for _, variable := range []string{"PGPASSWORD", "PGUSER"} {
			if got := motivo(t, ComprobarEntorno(p, Entorno{Variables: []string{variable + "=sintetica"}})); got != variable {
				t.Fatalf("%s: se esperaba rechazar %s, llego %s", p, variable, got)
			}
		}
		personal := t.TempDir()
		escribir(t, personal, ".pgpass", "127.0.0.1:5432:*:vec:x")
		if got := motivo(t, ComprobarEntorno(p, Entorno{DirectorioPersonal: personal})); got != "~/.pgpass" {
			t.Fatalf("%s: se esperaba rechazar ~/.pgpass, llego %s", p, got)
		}
		personal = t.TempDir()
		escribir(t, personal, ".postgresql/postgresql.key", "clave cliente")
		if got := motivo(t, ComprobarEntorno(p, Entorno{DirectorioPersonal: personal})); got != "~/.postgresql/postgresql.key" {
			t.Fatalf("%s: se esperaba rechazar la clave de cliente, llego %s", p, got)
		}
	}
}

func TestMaterialPropioDeCadaPortalSeAdmite(t *testing.T) {
	for _, p := range []Portal{PortalInterno, PortalExterno} {
		if err := ComprobarMaterial(p, materialSintetico(t, p)); err != nil {
			t.Fatalf("%s: material propio rechazado: %v", p, err)
		}
	}
}

func TestProcesoSeparadoSoloAdmiteMaterialNominalTLSKMSYTSA(t *testing.T) {
	for _, p := range []Portal{PortalInterno, PortalExterno} {
		material := materialSintetico(t, p)
		if p == PortalInterno {
			for _, relativa := range []string{
				"kms/atestacion-ed25519.key", "kms/atestacion-ed25519.pub",
				"kms/revalidacion-ed25519.key", "kms/revalidacion-ed25519.pub",
			} {
				escribir(t, material, relativa, "material sintetico")
			}
		}
		if err := ComprobarMaterial(p, material); err != nil {
			t.Fatalf("%s: material nominal rechazado: %v", p, err)
		}
		for _, relativa := range []string{
			"tls/cliente.key", "tls/intervencion.key", "tls/cliente.pem", "tls/otro.crt",
			"kms/cliente.key", "kms/cliente.pem", "tsa/cliente.pem",
			"idempotencia/cliente.pem", "externo/cliente.pem",
		} {
			material := materialSintetico(t, p)
			escribir(t, material, relativa, "material sintetico")
			if got := motivo(t, ComprobarMaterial(p, material)); got != relativa {
				t.Fatalf("%s: se esperaba rechazar %s, llego %s", p, relativa, got)
			}
		}
	}
}

func TestProcesoExternoNoPuedeAbrirMaterialInterno(t *testing.T) {
	for _, relativa := range []string{
		"kms/clave-maestra.bin", "kms/atestacion-ed25519.key", "kms/atestacion-ed25519.pub",
		"kms/revalidacion-ed25519.key", "kms/revalidacion-ed25519.pub", "tsa/clave-hmac.bin",
		"identidad/identidad.json",
		"identidad/intervencion.json",
		"identidad/cronos-empleado.json",
		"identidad/usuarios-preferencias-interna.json",
		"identidad/documentos.json",
		"mtls/cliente.crt",
		"comunicaciones/aviso-sintetico.json",
		"reglas-ejemplo/bolsa.json",
		"desconocido.bin",
	} {
		material := materialSintetico(t, PortalExterno)
		escribir(t, material, relativa, "sintetico")
		if got := motivo(t, ComprobarMaterial(PortalExterno, material)); got != relativa {
			t.Fatalf("se esperaba rechazar %s, llego %s", relativa, got)
		}
	}
}

func TestProcesoInternoNoPuedeAbrirMaterialExterno(t *testing.T) {
	for _, relativa := range []string{
		"identidad/bolsa-candidato.json",
		"identidad/candidato.json",
		"identidad/usuarios-preferencias-externa.json",
		"mtls/candidato.crt",
		"externo/cualquier-cosa.json",
	} {
		material := materialSintetico(t, PortalInterno)
		escribir(t, material, relativa, "sintetico")
		if got := motivo(t, ComprobarMaterial(PortalInterno, material)); got != relativa {
			t.Fatalf("se esperaba rechazar %s, llego %s", relativa, got)
		}
	}
}

func TestProcesoSeparadoRechazaClavesDeEmision(t *testing.T) {
	for _, p := range []Portal{PortalInterno, PortalExterno} {
		for _, relativa := range []string{"ca/ca.key", "ca/serie", "ca/intermedia.key", "ca/ca.key.bak", "mtls/cliente.key", "mtls/candidato.p12", "mtls/cliente.p12.password", "externo/y.key", "externo/x.p12", "identidad/otra.key"} {
			material := materialSintetico(t, p)
			escribir(t, material, relativa, "sintetico")
			if got := motivo(t, ComprobarMaterial(p, material)); got != relativa {
				t.Fatalf("%s: se esperaba rechazar %s, llego %s", p, relativa, got)
			}
		}
	}
}

func TestMarcaDePortalObligatoriaYCoherente(t *testing.T) {
	material := materialSintetico(t, PortalInterno)
	if got := motivo(t, ComprobarMaterial(PortalExterno, material)); got != FicheroMarcaPortal {
		t.Fatalf("el externo no debe aceptar material interno: %s", got)
	}
	if got := motivo(t, ComprobarMaterial(PortalCombinado, material)); got != FicheroMarcaPortal {
		t.Fatalf("el combinado no debe aceptar material separado: %s", got)
	}
	for _, marca := range []string{"", `{"version":1,"portal":""}`, `{"version":2,"portal":"interno"}`, `{"version":1,"portal":"interno","extra":1}`, `{"version":1,"portal":"interno"}{}`} {
		material := materialSintetico(t, PortalInterno)
		ruta := filepath.Join(material, FicheroMarcaPortal)
		if marca == "" {
			if err := os.Remove(ruta); err != nil {
				t.Fatal(err)
			}
		} else {
			escribir(t, material, FicheroMarcaPortal, marca)
		}
		if got := motivo(t, ComprobarMaterial(PortalInterno, material)); got != FicheroMarcaPortal {
			t.Fatalf("marca %q aceptada: %s", marca, got)
		}
	}
	material = materialSintetico(t, PortalInterno)
	if err := os.Chmod(filepath.Join(material, FicheroMarcaPortal), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := motivo(t, ComprobarMaterial(PortalInterno, material)); got != FicheroMarcaPortal {
		t.Fatalf("marca legible por otros aceptada: %s", got)
	}
}

func TestMaterialSinEnlacesNiRutasRelativas(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	if err := os.Mkdir(filepath.Join(externo, "kms"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(interno, "kms", "clave-maestra.bin"), filepath.Join(externo, "kms", "otra.bin")); err != nil {
		t.Fatal(err)
	}
	if got := motivo(t, ComprobarMaterial(PortalExterno, externo)); got != "kms/otra.bin" {
		t.Fatalf("enlace aceptado: %s", got)
	}
	if err := ComprobarMaterial(PortalExterno, "material-relativo"); !errors.Is(err, ErrSeparacionPortales) {
		t.Fatalf("ruta relativa aceptada: %v", err)
	}
}

func TestEntornoDeclaradoEnElMaterialTambienSeComprueba(t *testing.T) {
	material := materialSintetico(t, PortalExterno)
	escribir(t, material, "desarrollo.env", "export VEC_CT_DATABASE_URL=\"postgresql://vec_ct:x@127.0.0.1/vec\"\n")
	if got := motivo(t, ComprobarMaterial(PortalExterno, material)); got != "VEC_CT_DATABASE_URL" {
		t.Fatalf("conexion interna declarada aceptada: %s", got)
	}
}

func TestComprobarSeparacionAceptaProcesosDistintos(t *testing.T) {
	informe, err := ComprobarSeparacion(
		Proceso{Material: materialSintetico(t, PortalInterno), Entorno: []byte("export VEC_CT_DATABASE_URL=\"postgresql://vec_ct_runtime:x@127.0.0.1:$vec_local_pg_puerto/vec\"\n")},
		Proceso{Material: materialSintetico(t, PortalExterno), Entorno: []byte("VEC_EXTERNO_BOLSA_DATABASE_URL='postgresql://vec_bolsa_ext:x@127.0.0.1/vec'\n")},
	)
	if err != nil {
		t.Fatalf("procesos separados rechazados: %v", err)
	}
	if informe.SecretosInterno != 4 || informe.SecretosExterno != 2 || informe.UsuariosInterno != 2 || informe.UsuariosExterno != 2 {
		t.Fatalf("informe inesperado: %+v", informe)
	}
}

func TestComprobarSeparacionDetectaClaveCopiada(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	escribir(t, externo, "idempotencia/g1-localizador.bin", "idem interno")
	_, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo})
	if got := motivo(t, err); got != "idempotencia/g1-localizador.bin = idempotencia/g1-localizador.bin" {
		t.Fatalf("clave de idempotencia compartida aceptada: %s", got)
	}
	externo = materialSintetico(t, PortalExterno)
	escribir(t, externo, "externo/otra.bin", "idem externo")
	if _, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo}); err == nil {
		t.Fatal("un mismo secreto usado para dos fines debe rechazarse")
	}
}

func TestComprobarSeparacionDetectaUsuarioCompartido(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	_, err := ComprobarSeparacion(
		Proceso{Material: interno, Entorno: []byte("export VEC_CT_DATABASE_URL=\"postgresql://vec_compartido:x@127.0.0.1/vec\"\n")},
		Proceso{Material: externo, Entorno: []byte("VEC_EXTERNO_X_DATABASE_URL=host=127.0.0.1 user=vec_compartido password=y\n")},
	)
	if got := motivo(t, err); got != "vec_compartido" {
		t.Fatalf("usuario compartido aceptado: %s", got)
	}
	escribir(t, externo, "identidad/usuarios-preferencias-externa.json", `{"dsn":"postgresql://vec_cronos_sintetico:otra@127.0.0.1/vec"}`)
	_, err = ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo})
	if got := motivo(t, err); got != "vec_cronos_sintetico" {
		t.Fatalf("usuario compartido en el material aceptado: %s", got)
	}
}

func TestComprobarSeparacionRechazaDirectorioCompartido(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	if _, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: interno}); !errors.Is(err, ErrSeparacionPortales) {
		t.Fatalf("el mismo directorio debe rechazarse: %v", err)
	}
	anidado := filepath.Join(interno, "externo")
	if _, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: anidado}); !errors.Is(err, ErrSeparacionPortales) {
		t.Fatalf("un directorio anidado debe rechazarse: %v", err)
	}
}

func TestUsuarioConexion(t *testing.T) {
	casos := map[string]string{
		"postgresql://vec_a:secreto@127.0.0.1:5432/vec":             "vec_a",
		"postgres://vec_b@127.0.0.1:$vec_local_pg_puerto/vec":       "vec_b",
		"postgresql://vec_c:x@h/vec?sslmode=verify-full&user=vec_d": "vec_d",
		"host=127.0.0.1 user=vec_e password=x dbname=vec":           "vec_e",
		"host=127.0.0.1 user='vec_f' password=x":                    "vec_f",
		"postgresql://127.0.0.1/vec":                                usuarioSistemaAnonimo,
		"postgresql://vec%5Fg:x@127.0.0.1/vec":                      "vec_g",
	}
	for dsn, esperado := range casos {
		if got := usuarioConexion(dsn); got != esperado {
			t.Fatalf("%s: esperado %s, llego %s", dsn, esperado, got)
		}
	}
}

func TestComprobarSeparacionDetectaClaveCopiadaEnExterno(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	escribir(t, externo, "externo/clave.bin", "kms interno")
	if _, err := ComprobarSeparacion(Proceso{Material: interno}, Proceso{Material: externo}); err == nil {
		t.Fatal("una clave interna copiada bajo externo/ debe rechazarse")
	}
}

func TestComprobarSeparacionExigeUsuarioExplicito(t *testing.T) {
	interno := materialSintetico(t, PortalInterno)
	externo := materialSintetico(t, PortalExterno)
	_, err := ComprobarSeparacion(
		Proceso{Material: interno},
		Proceso{Material: externo, Entorno: []byte("VEC_EXTERNO_X_DATABASE_URL=\"host=127.0.0.1 password=x dbname=vec\"\n")},
	)
	if !errors.Is(err, ErrSeparacionPortales) {
		t.Fatalf("una conexion sin usuario explicito debe rechazarse: %v", err)
	}
}
