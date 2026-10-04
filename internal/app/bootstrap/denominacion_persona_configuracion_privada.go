package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var ErrConfiguracionDenominacionPrivada = errors.New("vec.denominacion_persona.configuracion_privada_no_disponible")

type ClaveDenominacionPersonaDesarrollo struct {
	Referencia   string    `json:"referencia"`
	Version      uint64    `json:"version"`
	Revocada     bool      `json:"revocada"`
	RetenerHasta time.Time `json:"retener_hasta"`
}
type NormaDenominacionPersonaDesarrollo struct {
	Referencia   string `json:"referencia"`
	Case         string `json:"case"`
	FormaUnicode string `json:"forma_unicode"`
	Separadores  string `json:"separadores"`
	MaxBytes     int    `json:"max_bytes"`
	MaxTokens    int    `json:"max_tokens"`
}
type MetadatosDenominacionPersonaDesarrollo struct {
	Version          uint64                               `json:"version"`
	Entorno          string                               `json:"entorno"`
	Norma            NormaDenominacionPersonaDesarrollo   `json:"norma"`
	Cifrado          ClaveDenominacionPersonaDesarrollo   `json:"cifrado"`
	Busqueda         ClaveDenominacionPersonaDesarrollo   `json:"busqueda"`
	CifradoRetenidas []ClaveDenominacionPersonaDesarrollo `json:"cifrado_retenidas"`
}

func (m MetadatosDenominacionPersonaDesarrollo) norma() ports.NormaDenominacionPersona {
	return ports.NormaDenominacionPersona{Ref: m.Norma.Referencia, Case: m.Norma.Case, FormaUnicode: m.Norma.FormaUnicode, Separadores: m.Norma.Separadores, MaxBytes: m.Norma.MaxBytes, MaxTokens: m.Norma.MaxTokens}
}
func (m ClaveDenominacionPersonaDesarrollo) referenciaVersionada() string {
	return m.Referencia + ":v" + strconv.FormatUint(m.Version, 10)
}
func (m ClaveDenominacionPersonaDesarrollo) validar(prefijo string, retenida bool) bool {
	ref := m.referenciaVersionada()
	if !strings.HasPrefix(m.Referencia, prefijo) || len(m.Referencia) <= len(prefijo) || len(ref) > 128 || m.Version == 0 || m.Version > 1<<53-1 || retenida && m.RetenerHasta.IsZero() {
		return false
	}
	for _, r := range ref {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return false
		}
	}
	return m.RetenerHasta.IsZero() || m.RetenerHasta.Year() >= 1 && m.RetenerHasta.Year() <= 9999
}
func (m MetadatosDenominacionPersonaDesarrollo) validar() error {
	if m.Version != 1 || m.Entorno != "desarrollo" || m.CifradoRetenidas == nil || len(m.CifradoRetenidas) > 8 || !m.Cifrado.validar("clave:denominacion:cifrado:", false) || !m.Busqueda.validar("clave:denominacion:busqueda:", false) {
		return ErrConfiguracionDenominacionPrivada
	}
	refs := map[string]bool{m.Cifrado.referenciaVersionada(): true, m.Busqueda.referenciaVersionada(): true}
	for _, c := range m.CifradoRetenidas {
		if !c.validar("clave:denominacion:cifrado:", true) || refs[c.referenciaVersionada()] {
			return ErrConfiguracionDenominacionPrivada
		}
		refs[c.referenciaVersionada()] = true
	}
	return nil
}

type fuenteConfiguracionPrivadaDenominacion struct{ ruta string }

func NuevaFuenteConfiguracionPrivadaDenominacionPersona(ruta string) (FuenteMetadatosDenominacionPersonaDesarrollo, error) {
	f := &fuenteConfiguracionPrivadaDenominacion{ruta: ruta}
	if _, err := f.CargarMetadatosDenominacionPersona(context.Background()); err != nil {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	return f, nil
}
func (f *fuenteConfiguracionPrivadaDenominacion) CargarMetadatosDenominacionPersona(ctx context.Context) (MetadatosDenominacionPersonaDesarrollo, error) {
	var m MetadatosDenominacionPersonaDesarrollo
	if f == nil || ctx == nil || ctx.Err() != nil {
		return m, ErrConfiguracionDenominacionPrivada
	}
	b, err := LeerArchivoPrivadoDenominacionPersona(f.ruta, 65536)
	if err != nil {
		return m, ErrConfiguracionDenominacionPrivada
	}
	defer clear(b)
	if camposMetadatosDenominacion(b) != nil {
		return m, ErrConfiguracionDenominacionPrivada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || m.validar() != nil || ctx.Err() != nil {
		return MetadatosDenominacionPersonaDesarrollo{}, ErrConfiguracionDenominacionPrivada
	}
	return m, nil
}
func camposMetadatosDenominacion(b []byte) error {
	var o map[string]json.RawMessage
	if json.Unmarshal(b, &o) != nil || !clavesConfiguracionDenominacion(o, "version", "entorno", "norma", "cifrado", "busqueda", "cifrado_retenidas") {
		return ErrConfiguracionDenominacionPrivada
	}
	var norma map[string]json.RawMessage
	if json.Unmarshal(o["norma"], &norma) != nil || !clavesConfiguracionDenominacion(norma, "referencia", "case", "forma_unicode", "separadores", "max_bytes", "max_tokens") {
		return ErrConfiguracionDenominacionPrivada
	}
	var retenidas []json.RawMessage
	if json.Unmarshal(o["cifrado_retenidas"], &retenidas) != nil || retenidas == nil {
		return ErrConfiguracionDenominacionPrivada
	}
	for _, b := range append([]json.RawMessage{o["cifrado"], o["busqueda"]}, retenidas...) {
		var clave map[string]json.RawMessage
		if json.Unmarshal(b, &clave) != nil || !clavesConfiguracionDenominacion(clave, "referencia", "version", "revocada", "retener_hasta") {
			return ErrConfiguracionDenominacionPrivada
		}
		if string(clave["revocada"]) != "true" && string(clave["revocada"]) != "false" || bytes.Equal(bytes.TrimSpace(clave["retener_hasta"]), []byte("null")) {
			return ErrConfiguracionDenominacionPrivada
		}
	}
	// El scanner rechaza claves duplicadas, incluso dentro de objetos anidados.
	d := json.NewDecoder(bytes.NewReader(b))
	if objetoUnicoDenominacion(d) != nil {
		return ErrConfiguracionDenominacionPrivada
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrConfiguracionDenominacionPrivada
	}
	return nil
}
func clavesConfiguracionDenominacion(o map[string]json.RawMessage, keys ...string) bool {
	if len(o) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := o[k]; !ok {
			return false
		}
	}
	return true
}
func objetoUnicoDenominacion(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		if t == nil {
			return ErrConfiguracionDenominacionPrivada
		}
		return nil
	}
	if delim == '{' {
		keys := map[string]bool{}
		for d.More() {
			t, err := d.Token()
			if err != nil {
				return err
			}
			k, ok := t.(string)
			if !ok || keys[k] {
				return ErrConfiguracionDenominacionPrivada
			}
			keys[k] = true
			if err := objetoUnicoDenominacion(d); err != nil {
				return err
			}
		}
	} else if delim == '[' {
		for d.More() {
			if err := objetoUnicoDenominacion(d); err != nil {
				return err
			}
		}
	} else {
		return ErrConfiguracionDenominacionPrivada
	}
	_, err = d.Token()
	return err
}

// AbrirRaizPrivadaDenominacionPersona conserva un descriptor de directorio
// propio 0700 fuera de Git. Los nombres se abren después mediante ese root.
func AbrirRaizPrivadaDenominacionPersona(ruta string) (*os.Root, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	padre := filepath.Dir(ruta)
	real, err := filepath.EvalSymlinks(padre)
	if err != nil || real != padre {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	info, err := os.Lstat(padre)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !archivoDenominacionPropio(info) {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	for p := padre; p != "/"; p = filepath.Dir(p) {
		if x, e := os.Lstat(filepath.Join(p, ".git")); e == nil {
			if !x.IsDir() {
				return nil, ErrConfiguracionDenominacionPrivada
			}
			if _, e := os.Lstat(filepath.Join(p, ".git", "HEAD")); e == nil {
				return nil, ErrConfiguracionDenominacionPrivada
			}
		}
	}
	root, err := os.OpenRoot(padre)
	if err != nil {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	abierta, err := root.Stat(".")
	if err != nil || !abierta.IsDir() || abierta.Mode().Perm() != 0700 || !archivoDenominacionPropio(abierta) {
		_ = root.Close()
		return nil, ErrConfiguracionDenominacionPrivada
	}
	return root, nil
}
func archivoDenominacionPropio(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}
func LeerArchivoPrivadoDenominacionPersona(ruta string, max int64) ([]byte, error) {
	root, err := AbrirRaizPrivadaDenominacionPersona(ruta)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return LeerEnRaizPrivadaDenominacionPersona(root, filepath.Base(ruta), max)
}
func LeerEnRaizPrivadaDenominacionPersona(root *os.Root, nombre string, max int64) ([]byte, error) {
	if root == nil || max < 1 {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	f, err := root.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !archivoDenominacionPropio(i) || i.Size() < 1 || i.Size() > max {
		return nil, ErrConfiguracionDenominacionPrivada
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, ErrConfiguracionDenominacionPrivada
	}
	return b, nil
}
