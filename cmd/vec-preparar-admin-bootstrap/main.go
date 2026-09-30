// Command vec-preparar-admin-bootstrap prepara un plan privado; nunca concede perfiles.
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
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const limiteDocumento = 64 << 10

type evidencia struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type certificado struct {
	PersonaRef     string    `json:"persona_ref"`
	CuentaRef      string    `json:"cuenta_ref"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CAHuellaSHA256 string    `json:"ca_huella_sha256"`
	Acreditacion   evidencia `json:"acreditacion"`
}

type persona struct {
	CuentaRef             string      `json:"cuenta_ref"`
	CuentaVersion         uint64      `json:"cuenta_version"`
	PersonaRef            string      `json:"persona_ref"`
	PersonaVersion        uint64      `json:"persona_version"`
	PerfilRef             string      `json:"perfil_ref"`
	VinculoRef            string      `json:"vinculo_ref"`
	PreimagenHuellaSHA256 string      `json:"preimagen_huella_sha256"`
	Procedencia           evidencia   `json:"procedencia"`
	VigenteHasta          time.Time   `json:"vigente_hasta"`
	Certificado           certificado `json:"certificado_admin"`
}

type rol struct {
	VersionRef          string `json:"version_ref"`
	HuellaSHA256        string `json:"huella_sha256"`
	ControlRevision     uint64 `json:"control_revision"`
	ControlHuellaSHA256 string `json:"control_huella_sha256"`
}

// material es una declaración privada de fuentes y preimágenes. AUT24 deberá
// cotejarla con las autoridades originales dentro de su transacción.
type material struct {
	Version                            uint64     `json:"version"`
	PreparadoEn                        time.Time  `json:"preparado_en"`
	CaducaEn                           time.Time  `json:"caduca_en"`
	ControlContinuidadRevisionEsperada uint64     `json:"control_continuidad_revision_esperada"`
	BootstrapEstadoEsperado            string     `json:"bootstrap_estado_esperado"`
	Rol                                rol        `json:"rol"`
	FuenteIdentidad                    evidencia  `json:"fuente_identidad"`
	FuenteCA                           evidencia  `json:"fuente_ca_admin"`
	Personas                           [2]persona `json:"personas"`
}

type documento struct {
	Plan             material `json:"plan"`
	HuellaPlanSHA256 string   `json:"huella_plan_sha256"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

func ejecutar(args []string, salida, errores io.Writer) int {
	f := flag.NewFlagSet("vec-preparar-admin-bootstrap", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var fuente, destino string
	var cotejar bool
	f.StringVar(&fuente, "fuente", "", "")
	f.StringVar(&destino, "plan", "", "")
	f.BoolVar(&cotejar, "cotejar", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 || fuente == "" || destino == "" || fuente == destino {
		return fallar(errores, "uso_invalido")
	}
	b, err := leerPrivado(fuente)
	if err != nil {
		return fallar(errores, "fuente_insegura")
	}
	defer clear(b)
	var m material
	if decodificarEstricto(b, &m) != nil || validar(m) != nil {
		return fallar(errores, "fuente_invalida")
	}
	contenido, err := json.Marshal(m)
	if err != nil {
		return fallar(errores, "plan_invalido")
	}
	h := sha256.Sum256(contenido)
	huella := hex.EncodeToString(h[:])
	p := domain.PreimagenBootstrapAdministracionPerfiles{
		Primera: preimagen(m.Personas[0]), Segunda: preimagen(m.Personas[1]), HuellaPlanSHA256: huella,
	}
	if p.Validar() != nil {
		return fallar(errores, "plan_invalido")
	}
	doc, err := json.Marshal(documento{Plan: m, HuellaPlanSHA256: huella})
	if err != nil || len(doc)+1 > limiteDocumento {
		return fallar(errores, "plan_invalido")
	}
	doc = append(doc, '\n')
	defer clear(doc)
	if cotejar {
		previo, e := leerPrivado(destino)
		if e != nil {
			return fallar(errores, "plan_ausente_o_inseguro")
		}
		defer clear(previo)
		if !bytes.Equal(previo, doc) {
			return fallar(errores, "plan_divergente")
		}
	} else if err = crearOComparar(destino, doc); err != nil {
		return fallar(errores, "plan_divergente_o_destino_inseguro")
	}
	if _, err = fmt.Fprintln(salida, huella); err != nil {
		return 1
	}
	return 0
}

func fallar(w io.Writer, codigo string) int { _, _ = fmt.Fprintln(w, codigo); return 1 }

func preimagen(p persona) domain.PreimagenAdministracionPerfiles {
	return domain.PreimagenAdministracionPerfiles{
		CuentaRef: p.CuentaRef, CuentaVersion: p.CuentaVersion, PersonaRef: p.PersonaRef,
		PersonaVersion: p.PersonaVersion, PerfilRef: p.PerfilRef, VinculoRef: p.VinculoRef,
		HuellaSHA256: p.PreimagenHuellaSHA256, ProcedenciaRef: p.Procedencia.Referencia,
		ProcedenciaVersion: p.Procedencia.Version, ProcedenciaHuellaSHA256: p.Procedencia.HuellaSHA256,
		VigenteHasta: p.VigenteHasta,
	}
}

func validar(m material) error {
	if m.Version != 1 || m.Rol.VersionRef != "rol:administracion_perfiles:v2" ||
		m.ControlContinuidadRevisionEsperada != 1 || m.BootstrapEstadoEsperado != "pendiente" ||
		!instanteCanonico(m.PreparadoEn) || !instanteCanonico(m.CaducaEn) || !m.CaducaEn.After(m.PreparadoEn) ||
		!domain.HuellaAdministracionPerfilesValida(m.Rol.HuellaSHA256) ||
		m.Rol.ControlRevision == 0 || !domain.HuellaAdministracionPerfilesValida(m.Rol.ControlHuellaSHA256) ||
		!evidenciaValida(m.FuenteIdentidad) || !evidenciaValida(m.FuenteCA) ||
		m.Personas[0].PerfilRef == m.Personas[1].PerfilRef ||
		m.Personas[0].VinculoRef == m.Personas[1].VinculoRef ||
		m.Personas[0].Certificado.HuellaSHA256 == m.Personas[1].Certificado.HuellaSHA256 {
		return errors.New("material invalido")
	}
	for _, p := range m.Personas {
		if preimagen(p).ValidarBootstrap() != nil || !evidenciaValida(p.Procedencia) ||
			!evidenciaValida(p.Certificado.Acreditacion) ||
			p.Certificado.PersonaRef != p.PersonaRef || p.Certificado.CuentaRef != p.CuentaRef ||
			!domain.HuellaAdministracionPerfilesValida(p.Certificado.HuellaSHA256) ||
			p.Certificado.CAHuellaSHA256 != m.FuenteCA.HuellaSHA256 ||
			!instanteCanonico(p.VigenteHasta) || p.VigenteHasta.Before(m.CaducaEn) {
			return errors.New("persona invalida")
		}
	}
	if m.Personas[0].PersonaRef >= m.Personas[1].PersonaRef {
		return errors.New("orden no canonico")
	}
	return nil
}

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
	case reflect.Array:
		var elementos []json.RawMessage
		if err := json.Unmarshal(b, &elementos); err != nil || len(elementos) != tipo.Len() {
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

func abrirRaizPrivada(ruta string) (*os.Root, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, errors.New("ruta")
	}
	padre := filepath.Dir(ruta)
	resuelta, err := filepath.EvalSymlinks(padre)
	if err != nil || resuelta != padre {
		return nil, errors.New("enlace")
	}
	dir, err := os.Lstat(padre)
	if err != nil || !dir.IsDir() || dir.Mode().Perm() != 0700 || !propio(dir) {
		return nil, errors.New("directorio")
	}
	for actual := padre; actual != "/"; actual = filepath.Dir(actual) {
		if _, err := os.Lstat(filepath.Join(actual, ".git")); err == nil {
			return nil, errors.New("repositorio")
		}
	}
	raiz, err := os.OpenRoot(padre)
	if err != nil {
		return nil, err
	}
	abierto, err := raiz.Stat(".")
	if err != nil || !os.SameFile(dir, abierto) || abierto.Mode().Perm() != 0700 || !propio(abierto) {
		_ = raiz.Close()
		return nil, errors.New("directorio cambiado")
	}
	return raiz, nil
}

func propio(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}

func leerPrivado(ruta string) ([]byte, error) {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	return leerEnRaiz(raiz, filepath.Base(ruta))
}

func leerEnRaiz(raiz *os.Root, nombre string) ([]byte, error) {
	i, err := raiz.Lstat(nombre)
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !propio(i) || i.Size() == 0 || i.Size() > limiteDocumento {
		return nil, errors.New("fichero")
	}
	f, err := raiz.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(i, actual) {
		return nil, errors.New("cambio")
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, errors.New("lectura")
	}
	return b, nil
}

func crearOComparar(ruta string, b []byte) error {
	raiz, err := abrirRaizPrivada(ruta)
	if err != nil {
		return err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if errors.Is(err, syscall.EEXIST) {
		previo, e := leerEnRaiz(raiz, filepath.Base(ruta))
		if e != nil {
			return e
		}
		defer clear(previo)
		if !bytes.Equal(previo, b) {
			return errors.New("divergencia")
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	if err := f.Sync(); err != nil {
		_ = raiz.Remove(filepath.Base(ruta))
		return err
	}
	return nil
}
