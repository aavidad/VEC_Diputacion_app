package internactproveedores

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialCTNoDisponible = errors.New("composicion interna: material CT no disponible")

const maximoInventarioCT = 64 << 10

type PoolMaterial struct {
	DSN   string `json:"dsn"`
	Login string `json:"login"`
}

func (PoolMaterial) String() string               { return "[material PostgreSQL privado]" }
func (m PoolMaterial) GoString() string           { return m.String() }
func (m PoolMaterial) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(m.String())) }
func (m PoolMaterial) LogValue() slog.Value       { return slog.StringValue(m.String()) }
func (PoolMaterial) MarshalJSON() ([]byte, error) {
	return []byte(`"[material PostgreSQL privado]"`), nil
}

type capacidadMaterial struct {
	ClaveID          string    `json:"clave_id"`
	Version          uint64    `json:"version"`
	Archivo          string    `json:"archivo"`
	SHA256           string    `json:"sha256"`
	EmisorID         string    `json:"emisor_id"`
	Desde            time.Time `json:"desde"`
	Hasta            time.Time `json:"hasta"`
	RevisionGobierno uint64    `json:"revision_gobierno"`
	HuellaGobierno   string    `json:"huella_gobierno"`
}

type materialV3 struct {
	ClaveID                string    `json:"clave_id"`
	ClaveVersion           uint64    `json:"clave_version"`
	ClaveArchivo           string    `json:"clave_archivo"`
	ClaveSHA256            string    `json:"clave_sha256"`
	Audiencia              string    `json:"audiencia"`
	RaizDesde              time.Time `json:"raiz_desde"`
	RaizHasta              time.Time `json:"raiz_hasta"`
	ConfiguracionRef       string    `json:"configuracion_ref"`
	ConfiguracionOrden     uint64    `json:"configuracion_orden"`
	ConfiguracionPublicada time.Time `json:"configuracion_publicada"`
	ConfiguracionExpira    time.Time `json:"configuracion_expira"`
	ConfiguracionSHA256    string    `json:"configuracion_sha256"`
	Capacidades            struct {
		Alta    capacidadMaterial `json:"alta"`
		Lectura capacidadMaterial `json:"lectura"`
		CT      capacidadMaterial `json:"ct"`
		Cuadro  capacidadMaterial `json:"cuadro"`
		Detalle capacidadMaterial `json:"detalle"`
	} `json:"capacidades"`
}

type ternaPersonalMaterial struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// El inventario es privado y se abre una sola vez. No conserva material de
// clave en formato JSON; sólo nombres locales y huellas de archivos 0600.
type Material struct {
	Version         int                                `json:"version"`
	Pools           map[string]PoolMaterial            `json:"pools"`
	PlanesArchivo   string                             `json:"planes_archivo"`
	TernaPlanes     ct.ReferenciaVersionadaPersonalRPT `json:"terna_planes"`
	PersonalArchivo string                             `json:"personal_archivo"`
	TernaPersonal   ternaPersonalMaterial              `json:"terna_personal"`
	CatalogoMotivos string                             `json:"catalogo_motivos"`
	MotivoAlta      core.ReferenciaEntradaCatalogo     `json:"motivo_alta"`
	MotivoLectura   core.ReferenciaEntradaCatalogo     `json:"motivo_lectura"`
	V3              materialV3                         `json:"v3"`
	raiz            *os.Root
}

func (Material) String() string               { return "[material CT privado]" }
func (m Material) GoString() string           { return m.String() }
func (m Material) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(m.String())) }
func (m Material) LogValue() slog.Value       { return slog.StringValue(m.String()) }
func (Material) MarshalJSON() ([]byte, error) { return []byte(`"[material CT privado]"`), nil }

func (m *Material) Cerrar() error {
	if m == nil || m.raiz == nil {
		return nil
	}
	err := m.raiz.Close()
	m.raiz = nil
	return err
}

func CargarMaterial(directorio string) (Material, error) {
	var m Material
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio || strings.TrimSpace(directorio) != directorio {
		return m, ErrMaterialCTNoDisponible
	}
	resuelta, err := filepath.EvalSymlinks(directorio)
	if err != nil || resuelta != directorio || dentroRepositorioGit(directorio) {
		return m, ErrMaterialCTNoDisponible
	}
	info, err := os.Lstat(directorio)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return m, ErrMaterialCTNoDisponible
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return m, ErrMaterialCTNoDisponible
	}
	cerrada := false
	defer func() {
		if !cerrada {
			_ = raiz.Close()
		}
	}()
	actual, err := raiz.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		return m, ErrMaterialCTNoDisponible
	}
	b, err := leerArchivoPrivado(raiz, "ct_v3.json", maximoInventarioCT)
	if err != nil {
		return m, ErrMaterialCTNoDisponible
	}
	defer clear(b)
	if clavesJSONDuplicadas(b) {
		return m, ErrMaterialCTNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || m.Version != 1 || len(m.Pools) != 6 ||
		m.TernaPlanes.Validar() != nil || m.TernaPersonal.Referencia == "" || m.TernaPersonal.Version == 0 ||
		len(m.TernaPersonal.HuellaSHA256) != 64 || m.CatalogoMotivos == "" ||
		!core.ReferenciaMotivoAutorizacionV2Valida(m.MotivoAlta) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(m.MotivoLectura) ||
		m.MotivoAlta.CatalogoID != m.CatalogoMotivos || m.MotivoLectura.CatalogoID != m.CatalogoMotivos {
		return Material{}, ErrMaterialCTNoDisponible
	}
	loginVistos := map[string]bool{}
	for _, nombre := range []string{"fuente_autorizacion", "registro_autorizacion", "motivos_autorizacion", "motivos_rrhh", "consulta_rrhh", "gobierno_v3"} {
		p, ok := m.Pools[nombre]
		if !ok || p.DSN == "" || p.Login == "" || strings.TrimSpace(p.DSN) != p.DSN || strings.TrimSpace(p.Login) != p.Login || loginVistos[p.Login] {
			return Material{}, ErrMaterialCTNoDisponible
		}
		loginVistos[p.Login] = true
	}
	for _, nombre := range []string{m.PlanesArchivo, m.PersonalArchivo, m.V3.ClaveArchivo,
		m.V3.Capacidades.Alta.Archivo, m.V3.Capacidades.Lectura.Archivo, m.V3.Capacidades.CT.Archivo,
		m.V3.Capacidades.Cuadro.Archivo, m.V3.Capacidades.Detalle.Archivo} {
		if !filepath.IsLocal(nombre) || nombre == "." {
			return Material{}, ErrMaterialCTNoDisponible
		}
	}
	m.raiz = raiz
	cerrada = true
	return m, nil
}

func dentroRepositorioGit(ruta string) bool {
	for actual := ruta; actual != ""; actual = filepath.Dir(actual) {
		if _, err := os.Lstat(filepath.Join(actual, ".git")); err == nil {
			return true
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			break
		}
	}
	return false
}

func leerArchivoPrivado(raiz *os.Root, nombre string, limite int64) ([]byte, error) {
	if raiz == nil || !filepath.IsLocal(nombre) || nombre == "." {
		return nil, ErrMaterialCTNoDisponible
	}
	info, err := raiz.Lstat(nombre)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() < 1 || info.Size() > limite {
		return nil, ErrMaterialCTNoDisponible
	}
	f, err := raiz.Open(nombre)
	if err != nil {
		return nil, ErrMaterialCTNoDisponible
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, ErrMaterialCTNoDisponible
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) != info.Size() {
		clear(b)
		return nil, ErrMaterialCTNoDisponible
	}
	return b, nil
}

func leerConHuella(raiz *os.Root, nombre, huella string, limite int64) ([]byte, error) {
	b, err := leerArchivoPrivado(raiz, nombre, limite)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != huella {
		clear(b)
		return nil, ErrMaterialCTNoDisponible
	}
	return b, nil
}

func clavesJSONDuplicadas(b []byte) bool {
	d := json.NewDecoder(bytes.NewReader(b))
	var leer func() error
	leer = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			vistas := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				s, ok := k.(string)
				if !ok || vistas[s] {
					return ErrMaterialCTNoDisponible
				}
				vistas[s] = true
				if e = leer(); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e := leer(); e != nil {
					return e
				}
			}
		default:
			return ErrMaterialCTNoDisponible
		}
		_, err = d.Token()
		return err
	}
	if leer() != nil {
		return true
	}
	_, err := d.Token()
	return err != io.EOF
}
