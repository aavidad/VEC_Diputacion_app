package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

//go:embed textos/es.json
var catalogoBytes []byte

type textos map[string]string

func (t textos) decir(w io.Writer, clave string) { fmt.Fprintln(w, t[clave]) }

type opciones struct {
	Registro, DER, Vinculo, Estado, Desde, Hasta, Evidencia, EvidenciaSHA string
	Preimagen, PreimagenSHA                                               string
	Revision, PreimagenRevision                                           uint64
	SoloKit                                                               bool
}

type destino struct {
	CuentaRef      string `json:"cuenta_ref"`
	CuentaVersion  uint64 `json:"cuenta_version"`
	PersonaRef     string `json:"persona_ref"`
	PersonaVersion uint64 `json:"persona_version"`
	VinculoRef     string `json:"vinculo_cuenta_persona_ref"`
	VinculoVersion uint64 `json:"vinculo_cuenta_persona_version"`
}

// El orden de los campos reproduce la reconstrucción byte a byte de CA24.
type descriptor struct {
	Esquema                     string `json:"esquema"`
	VinculoRef                  string `json:"vinculo_ref"`
	Revision                    uint64 `json:"revision"`
	CertificadoSHA              string `json:"certificado_der_sha256"`
	CuentaRef                   string `json:"cuenta_ref"`
	CuentaVersion               uint64 `json:"cuenta_version"`
	PersonaRef                  string `json:"persona_ref"`
	PersonaVersion              uint64 `json:"persona_version"`
	VinculoCuentaPersonaRef     string `json:"vinculo_cuenta_persona_ref"`
	VinculoCuentaPersonaVersion uint64 `json:"vinculo_cuenta_persona_version"`
	Estado                      string `json:"estado"`
	VigenteDesde                string `json:"vigente_desde"`
	VigenteHasta                string `json:"vigente_hasta"`
	EvidenciaRef                string `json:"evidencia_ref"`
	EvidenciaSHA                string `json:"evidencia_sha256"`
	PreimagenRef                string `json:"preimagen_ref"`
	PreimagenRevision           uint64 `json:"preimagen_revision"`
	PreimagenSHA                string `json:"preimagen_sha256"`
}

var (
	referenciaSufijo = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	huellaPatron     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	instantePatron   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$`)
)

func referenciaValida(valor, prefijo string) bool {
	return strings.HasPrefix(valor, prefijo) && len(valor) >= len(prefijo)+22 && len(valor) <= len(prefijo)+128 && referenciaSufijo.MatchString(valor[len(prefijo):])
}
func registroValido(valor string) bool {
	return strings.HasPrefix(valor, "rca_") && len(valor) >= len("rca_")+24 && len(valor) <= len("rca_")+128 && referenciaSufijo.MatchString(valor[len("rca_"):])
}
func huellaValida(valor string) bool {
	return huellaPatron.MatchString(valor) && valor != strings.Repeat("0", 64)
}
func instante(valor string) (time.Time, error) {
	if !instantePatron.MatchString(valor) {
		return time.Time{}, errors.New("instante")
	}
	t, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil || t.Year() < 1 || t.Year() > 9999 || t.Location() != time.UTC {
		return time.Time{}, errors.New("instante")
	}
	return t, nil
}

func certificadoSHA(ruta string) (string, error) {
	if !strings.EqualFold(ruta[max(0, len(ruta)-4):], ".der") {
		return "", errors.New("der")
	}
	f, err := os.Open(ruta)
	if err != nil {
		return "", errors.New("der")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 128*1024+1))
	if err != nil || len(b) == 0 || len(b) > 128*1024 {
		return "", errors.New("der")
	}
	c, err := x509.ParseCertificate(b)
	if err != nil || !bytes.Equal(c.Raw, b) {
		return "", errors.New("der")
	}
	h := sha256.Sum256(c.Raw)
	return hex.EncodeToString(h[:]), nil
}
func validarOpciones(o opciones, sha string) error {
	desde, e1 := instante(o.Desde)
	hasta, e2 := instante(o.Hasta)
	if !o.SoloKit || !registroValido(o.Registro) || !referenciaValida(o.Vinculo, "vcc_") || !referenciaValida(o.Evidencia, "evi_") ||
		o.Revision == 0 ||
		(o.Estado != "activo" && o.Estado != "revocado") || !huellaValida(sha) || !huellaValida(o.EvidenciaSHA) ||
		e1 != nil || e2 != nil || !hasta.After(desde) {
		return errors.New("entrada")
	}
	if o.Preimagen == "" {
		if o.PreimagenRevision != 0 || o.PreimagenSHA != "" || o.Revision != 1 || o.Estado != "activo" {
			return errors.New("entrada")
		}
	} else if !referenciaValida(o.Preimagen, "vcc_") || o.PreimagenRevision == 0 || !huellaValida(o.PreimagenSHA) ||
		o.Preimagen != o.Vinculo || o.Revision != o.PreimagenRevision+1 || o.PreimagenRevision == ^uint64(0) {
		return errors.New("entrada")
	}
	return nil
}
func construir(o opciones, d destino, sha string) (descriptor, error) {
	if validarOpciones(o, sha) != nil || !referenciaValida(d.CuentaRef, "cta_") || !referenciaValida(d.PersonaRef, "per_") ||
		!referenciaValida(d.VinculoRef, "vca_") || d.CuentaVersion == 0 || d.PersonaVersion == 0 || d.VinculoVersion == 0 {
		return descriptor{}, errors.New("destino")
	}
	return descriptor{"vec.certificado-cuenta.binding.v1", o.Vinculo, o.Revision, sha, d.CuentaRef, d.CuentaVersion, d.PersonaRef, d.PersonaVersion, d.VinculoRef, d.VinculoVersion, o.Estado, o.Desde, o.Hasta, o.Evidencia, o.EvidenciaSHA, o.Preimagen, o.PreimagenRevision, o.PreimagenSHA}, nil
}

func parsear(args []string, t textos) (opciones, error) {
	var o opciones
	fs := flag.NewFlagSet("vec-certificado-firmante-kit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.SoloKit, "solo-kit-sintetico", false, t["solo_kit"])
	fs.StringVar(&o.Registro, "registro-destino-ref", "", t["registro"])
	fs.StringVar(&o.DER, "certificado-der", "", t["der"])
	fs.StringVar(&o.Vinculo, "vinculo-ref", "", t["vinculo"])
	fs.Uint64Var(&o.Revision, "revision", 0, t["revision"])
	fs.StringVar(&o.Estado, "estado", "", t["estado"])
	fs.StringVar(&o.Desde, "vigente-desde", "", t["desde"])
	fs.StringVar(&o.Hasta, "vigente-hasta", "", t["hasta"])
	fs.StringVar(&o.Evidencia, "evidencia-ref", "", t["evidencia"])
	fs.StringVar(&o.EvidenciaSHA, "evidencia-sha256", "", t["evidencia_sha"])
	fs.StringVar(&o.Preimagen, "preimagen-ref", "", t["preimagen"])
	fs.Uint64Var(&o.PreimagenRevision, "preimagen-revision", 0, t["preimagen_revision"])
	fs.StringVar(&o.PreimagenSHA, "preimagen-sha256", "", t["preimagen_sha"])
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || !o.SoloKit {
		return o, errors.New("argumentos")
	}
	return o, nil
}

func ejecutar(args []string, salida, diagnostico io.Writer, consultar func(string) (destino, error)) int {
	var t textos
	if json.Unmarshal(catalogoBytes, &t) != nil || len(t) == 0 {
		fmt.Fprintln(diagnostico, "certificado_firmante.textos_no_disponibles")
		return 2
	}
	o, err := parsear(args, t)
	if err != nil {
		t.decir(diagnostico, "argumentos")
		return 2
	}
	sha, err := certificadoSHA(o.DER)
	if err != nil {
		t.decir(diagnostico, "certificado")
		return 2
	}
	// La validación local precede a la conexión; los metadatos se obtienen de CA24.
	if err = validarOpciones(o, sha); err != nil {
		t.decir(diagnostico, "argumentos")
		return 2
	}
	d, err := consultar(o.Registro)
	if err != nil {
		t.decir(diagnostico, "consulta")
		return 1
	}
	resultado, err := construir(o, d, sha)
	if err != nil {
		t.decir(diagnostico, "destino")
		return 1
	}
	b, err := json.Marshal(resultado)
	if err != nil {
		t.decir(diagnostico, "salida")
		return 1
	}
	if _, err = salida.Write(append(b, '\n')); err != nil {
		t.decir(diagnostico, "salida")
		return 1
	}
	return 0
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, consultarDestino)) }
