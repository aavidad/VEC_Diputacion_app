// Command vec-preparar-admin-bootstrap prepara un plan privado y permite aplicarlo
// exclusivamente por el canal de operador con aprobación y proveedor nominal.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const limiteDocumento = 64 << 10

type evidencia = domain.EvidenciaBootstrapAdministracion
type certificado = domain.CertificadoBootstrapAdministracion
type persona = domain.PersonaBootstrapAdministracion
type rol = domain.RolBootstrapAdministracion
type material = domain.PlanBootstrapAdministracionV2

type documento struct {
	Plan             material `json:"plan"`
	HuellaPlanSHA256 string   `json:"huella_plan_sha256"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

func ejecutar(args []string, salida, errores io.Writer) int {
	f := flag.NewFlagSet("vec-preparar-admin-bootstrap", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var fuente, destino, conexion, aprobacion, recibo, textos string
	var versionPlan uint64
	var cotejar, aplicar bool
	f.StringVar(&fuente, "fuente", "", "")
	f.StringVar(&destino, "plan", "", "")
	f.BoolVar(&cotejar, "cotejar", false, "")
	f.BoolVar(&aplicar, "aplicar", false, "")
	f.StringVar(&conexion, "conexion", "", "")
	f.StringVar(&aprobacion, "aprobacion", "", "")
	f.StringVar(&recibo, "recibo", "", "")
	f.StringVar(&textos, "textos", "", "")
	f.Uint64Var(&versionPlan, "version-plan", 0, "")
	if f.Parse(args) != nil || f.NArg() != 0 || fuente == "" || destino == "" || fuente == destino {
		return fallarEntradaCLI(errores, textos, "uso_invalido")
	}
	b, err := leerPrivado(fuente)
	if err != nil {
		return fallarEntradaCLI(errores, textos, "fuente_insegura")
	}
	defer clear(b)
	var formato struct {
		Version uint64 `json:"version"`
	}
	if json.Unmarshal(b, &formato) != nil || versionPlan != 0 && versionPlan != formato.Version {
		return fallarEntradaCLI(errores, textos, "fuente_invalida")
	}
	if formato.Version == 3 {
		return ejecutarBootstrapV3(b, opcionesBootstrapV3{fuente: fuente, destino: destino, textos: textos,
			cotejar: cotejar, aplicar: aplicar, conexion: conexion, aprobacion: aprobacion, recibo: recibo}, salida, errores)
	}
	if textos != "" || versionPlan != 0 && versionPlan != 2 ||
		aplicar && (!cotejar || conexion == "" || aprobacion == "" || recibo == "" || recibo == fuente || recibo == destino || recibo == conexion || recibo == aprobacion) ||
		!aplicar && (conexion != "" || aprobacion != "" || recibo != "") {
		return fallarEntradaCLI(errores, textos, "uso_invalido")
	}
	var m material
	if decodificarEstricto(b, &m) != nil || validar(m) != nil {
		return fallarEntradaCLI(errores, textos, "fuente_invalida")
	}
	contenido, err := json.Marshal(m)
	if err != nil {
		return fallarEntradaCLI(errores, textos, "plan_invalido")
	}
	h := sha256.Sum256(contenido)
	huella := hex.EncodeToString(h[:])
	p, err := m.Preimagen()
	if err != nil || p.Validar() != nil || p.HuellaPlanSHA256 != huella {
		return fallarEntradaCLI(errores, textos, "plan_invalido")
	}
	doc, err := json.Marshal(documento{Plan: m, HuellaPlanSHA256: huella})
	if err != nil || len(doc)+1 > limiteDocumento {
		return fallarEntradaCLI(errores, textos, "plan_invalido")
	}
	doc = append(doc, '\n')
	defer clear(doc)
	if cotejar {
		previo, e := leerPrivado(destino)
		if e != nil {
			return fallarEntradaCLI(errores, textos, "plan_ausente_o_inseguro")
		}
		defer clear(previo)
		if !bytes.Equal(previo, doc) {
			return fallarEntradaCLI(errores, textos, "plan_divergente")
		}
	} else if err = crearOComparar(destino, doc); err != nil {
		return fallarEntradaCLI(errores, textos, "plan_divergente_o_destino_inseguro")
	}
	if aplicar {
		r, err := aplicarPlan(m, conexion, aprobacion)
		if err != nil {
			return fallarEntradaCLI(errores, textos, "provision_no_confirmada")
		}
		b, err := json.Marshal(r)
		if err != nil {
			return fallarEntradaCLI(errores, textos, "recibo_invalido")
		}
		b = append(b, '\n')
		defer clear(b)
		if crearOComparar(recibo, b) != nil {
			return fallarEntradaCLI(errores, textos, "recibo_no_guardado")
		}
		if _, err = fmt.Fprintln(salida, r.ReciboRef); err != nil {
			return fallarEntradaCLI(errores, textos, "plan_salida_fallida")
		}
		return 0
	}
	if _, err = fmt.Fprintln(salida, huella); err != nil {
		return fallarEntradaCLI(errores, textos, "plan_salida_fallida")
	}
	return 0
}

func fallar(w io.Writer, codigo string) int { _, _ = fmt.Fprintln(w, codigo); return 1 }

func preimagen(p persona) domain.PreimagenAdministracionPerfiles { return p.Preimagen() }
func validar(m material) error                                   { return m.Validar() }

func evidenciaValida(e evidencia) bool {
	return len(e.Referencia) > 0 && len(e.Referencia) <= 256 &&
		!strings.ContainsAny(e.Referencia, "* \t\r\n") && e.Version > 0 &&
		domain.HuellaAdministracionPerfilesValida(e.HuellaSHA256)
}

func instanteCanonico(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond() == 0
}

func decodificarEstricto(b []byte, destino any) error {
	if err := clavesUnicas(b); err != nil {
		return err
	}
	if err := clavesExactas(b, reflect.TypeOf(destino).Elem()); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	var sobra any
	if !errors.Is(d.Decode(&sobra), io.EOF) {
		return errors.New("contenido adicional")
	}
	return nil
}

func clavesExactas(b []byte, tipo reflect.Type) error {
	if tipo == reflect.TypeOf(time.Time{}) {
		return nil
	}
	switch tipo.Kind() {
	case reflect.Array, reflect.Slice:
		var elementos []json.RawMessage
		if err := json.Unmarshal(b, &elementos); err != nil || elementos == nil || tipo.Kind() == reflect.Array && len(elementos) != tipo.Len() {
			return errors.New("longitud invalida")
		}
		for _, x := range elementos {
			if err := clavesExactas(x, tipo.Elem()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(b, &campos); err != nil || campos == nil {
			return errors.New("objeto invalido")
		}
		if len(campos) != tipo.NumField() {
			return errors.New("campos incompletos")
		}
		for i := range tipo.NumField() {
			campo := tipo.Field(i)
			nombre := campo.Tag.Get("json")
			valor, ok := campos[nombre]
			if !ok || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
				return errors.New("campo ausente")
			}
			if err := clavesExactas(valor, campo.Type); err != nil {
				return err
			}
		}
	}
	return nil
}

// El decodificador de Go acepta claves repetidas; este recorrido las deniega.
func clavesUnicas(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var recorrer func() error
	recorrer = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'):
			vistas := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				clave, ok := k.(string)
				if !ok || vistas[clave] {
					return errors.New("clave repetida")
				}
				vistas[clave] = true
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		default:
			return nil
		}
	}
	return recorrer()
}
