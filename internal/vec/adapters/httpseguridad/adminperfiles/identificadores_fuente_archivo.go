package adminperfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

type fuenteIdentificadoresArchivoADMIN struct{ entradas []entradaIdentificadoresADMIN }
type entradaIdentificadoresADMIN struct {
	Persona     string `json:"persona_ref"`
	Cuenta      string `json:"cuenta_ref"`
	Ordinaria   string `json:"cuenta_ordinaria_ref"`
	Certificado string `json:"certificado_sha256"`
	CA          string `json:"ca_sha256"`
	Espacio     string `json:"espacio_identidad"`
	Dominio     string `json:"dominio_hmac_ref"`
	Clave       string `json:"clave_hmac_id"`
	Version     uint64 `json:"clave_hmac_version"`
	Fuente      string `json:"fuente_ref"`
	FuenteSHA   string `json:"fuente_sha256"`
	SujetoID    string `json:"sujeto_id"`
	CuentaID    string `json:"cuenta_id"`
	OrdinariaID string `json:"cuenta_ordinaria_id"`
}

// El archivo se obtiene del productor original y su SHA se aprueba por el
// canal privado. Este cargador no reconstruye identificadores ni calcula HMAC.
func NuevaFuenteIdentificadoresADMINDesdeArchivo(ruta, shaAprobado string) (FuenteIdentificadoresADMIN, error) {
	if !huella(shaAprobado) || ruta == "" {
		return nil, api.ErrConfiguracionIncompleta
	}
	b, err := leerArchivoPrivadoIdentificadoresADMIN(ruta)
	if err != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	defer clear(b)
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != shaAprobado {
		return nil, api.ErrConfiguracionIncompleta
	}
	var doc struct {
		Version  uint64                        `json:"version"`
		Entradas []entradaIdentificadoresADMIN `json:"entradas"`
	}
	if jsonCerradoIS16(b, &doc) != nil || doc.Version != 1 || len(doc.Entradas) < 1 || len(doc.Entradas) > 16 {
		return nil, api.ErrConfiguracionIncompleta
	}
	for i, e := range doc.Entradas {
		if !referencia(e.Persona, "per_") || !referencia(e.Cuenta, "cta_") || !referencia(e.Ordinaria, "cta_") || e.Cuenta == e.Ordinaria ||
			!huella(e.Certificado) || !huella(e.CA) || !huella(e.FuenteSHA) || !referencia(e.Dominio, "idh_") ||
			!strings.HasPrefix(e.Espacio, "https://") || e.Version == 0 || e.Version > 1<<63-1 || e.Clave == "" || e.Fuente == "" ||
			!identificadorOriginalADMIN(e.SujetoID) || !identificadorOriginalADMIN(e.CuentaID) || !identificadorOriginalADMIN(e.OrdinariaID) ||
			e.CuentaID == e.OrdinariaID || strings.ToLower(e.CuentaID) != e.CuentaID || strings.ToLower(e.OrdinariaID) != e.OrdinariaID {
			return nil, api.ErrConfiguracionIncompleta
		}
		for _, anterior := range doc.Entradas[:i] {
			if anterior.Certificado == e.Certificado || anterior.Cuenta == e.Cuenta {
				return nil, api.ErrConfiguracionIncompleta
			}
		}
	}
	return &fuenteIdentificadoresArchivoADMIN{doc.Entradas}, nil
}

// Reutiliza el protocolo de cmd/vec-mantener-admin-fijo/archivos.go:
// raíz privada propia, sin repositorio ni enlaces en la ruta y archivo propio.
func leerArchivoPrivadoIdentificadoresADMIN(ruta string) ([]byte, error) {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return nil, api.ErrConfiguracionIncompleta
	}
	padre := filepath.Dir(ruta)
	real, err := filepath.EvalSymlinks(padre)
	if err != nil || real != padre {
		return nil, api.ErrConfiguracionIncompleta
	}
	i, err := os.Lstat(padre)
	if err != nil || !i.IsDir() || i.Mode().Perm() != 0700 || !archivoIdentificadoresADMINPropio(i) {
		return nil, api.ErrConfiguracionIncompleta
	}
	for actual := padre; actual != "/"; actual = filepath.Dir(actual) {
		marca := filepath.Join(actual, ".git")
		if info, e := os.Lstat(marca); e == nil {
			if !info.IsDir() {
				return nil, api.ErrConfiguracionIncompleta
			}
			if _, e := os.Lstat(filepath.Join(marca, "HEAD")); e == nil {
				return nil, api.ErrConfiguracionIncompleta
			}
		}
	}
	root, err := os.OpenRoot(padre)
	if err != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	defer root.Close()
	abierta, err := root.Stat(".")
	if err != nil || !abierta.IsDir() || abierta.Mode().Perm() != 0700 || !archivoIdentificadoresADMINPropio(abierta) {
		return nil, api.ErrConfiguracionIncompleta
	}
	f, err := root.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	defer f.Close()
	i, err = f.Stat()
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !archivoIdentificadoresADMINPropio(i) || i.Size() < 1 || i.Size() > 32768 {
		return nil, api.ErrConfiguracionIncompleta
	}
	b, err := io.ReadAll(io.LimitReader(f, 32769))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, api.ErrConfiguracionIncompleta
	}
	return b, nil
}

func archivoIdentificadoresADMINPropio(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}

func identificadorOriginalADMIN(id string) bool {
	if id == "" || len(id) > 512 {
		return false
	}
	for _, c := range id {
		if c <= 32 || c >= 127 {
			return false
		}
	}
	return true
}

func (f *fuenteIdentificadoresArchivoADMIN) ResolverIdentificadoresADMIN(ctx context.Context, r ReferenciaFuenteIdentificadoresADMIN) (IdentificadoresFuenteADMIN, error) {
	if f == nil || ctx == nil || ctx.Err() != nil {
		return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
	}
	for _, e := range f.entradas {
		if e.Persona == r.PersonaRef && e.Cuenta == r.CuentaRef && e.Ordinaria == r.CuentaOrdinariaRef && e.Certificado == r.CertificadoSHA256 && e.CA == r.CASHA256 &&
			e.Espacio == r.EspacioIdentidad && e.Dominio == r.DominioHMACRef && e.Clave == r.ClaveHMACID && e.Version == r.ClaveHMACVersion && e.Fuente == r.FuenteRef && e.FuenteSHA == r.FuenteSHA256 {
			return IdentificadoresFuenteADMIN{SujetoID: e.SujetoID, CuentaID: e.CuentaID, CuentaOrdinariaID: e.OrdinariaID, EspacioIdentidad: e.Espacio, DominioHMACRef: e.Dominio, ClaveHMACID: e.Clave, ClaveHMACVersion: e.Version, FuenteRef: e.Fuente, FuenteSHA256: e.FuenteSHA}, nil
		}
	}
	return IdentificadoresFuenteADMIN{}, api.ErrConfiguracionIncompleta
}

func (IdentificadoresFuenteADMIN) String() string               { return "[IDENTIFICADORES-ADMIN-PRIVADOS]" }
func (v IdentificadoresFuenteADMIN) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, v.String()) }
func (v IdentificadoresFuenteADMIN) LogValue() slog.Value       { return slog.StringValue(v.String()) }
func (*fuenteIdentificadoresArchivoADMIN) String() string {
	return "[FUENTE-IDENTIFICADORES-ADMIN-PRIVADA]"
}
func (v *fuenteIdentificadoresArchivoADMIN) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, v.String())
}
func (v *fuenteIdentificadoresArchivoADMIN) LogValue() slog.Value {
	return slog.StringValue(v.String())
}
func (CuentaADMIN) MarshalJSON() ([]byte, error) { return []byte(`{"cuenta_admin":"oculta"}`), nil }
func (CuentaADMIN) String() string               { return "[CUENTA-ADMIN-PRIVADA]" }
func (v CuentaADMIN) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, v.String()) }
func (v CuentaADMIN) LogValue() slog.Value       { return slog.StringValue(v.String()) }
