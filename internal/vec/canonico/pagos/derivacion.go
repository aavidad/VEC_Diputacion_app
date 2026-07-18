package pagos

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ErrInicioOperacionCobroInvalido es el error contractual compartido por la
// representacion canonica y su reexportacion desde los puertos.
var ErrInicioOperacionCobroInvalido = errors.New("vec: inicio de operacion de cobro invalido")

// OrigenPasarelaCobroPublicado contiene los datos necesarios para validar y
// sellar un origen publicado de pasarela.
type OrigenPasarelaCobroPublicado struct {
	ID                        string
	Version                   int
	BaseHTTPS                 string
	RutasPermitidas           []string
	CamposHandoffPermitidos   []string
	HuellaConfiguracionSHA256 string
	PublicadaEn               time.Time
}

// ConfiguracionOrigenValidaSinHuella comprueba la forma previa al sellado.
func ConfiguracionOrigenValidaSinHuella(origen OrigenPasarelaCobroPublicado) bool {
	base, err := url.Parse(origen.BaseHTTPS)
	return err == nil && ClaveValida(origen.ID) && origen.Version >= 1 && base.Scheme == "https" &&
		base.Host != "" && base.Hostname() != "" && base.User == nil && base.Opaque == "" &&
		base.RawQuery == "" && !base.ForceQuery && base.Fragment == "" && base.RawPath == "" &&
		(base.Path == "" || base.Path == "/") && len(origen.BaseHTTPS) <= 2048 && !origen.PublicadaEn.IsZero() &&
		ListaCerradaValida(origen.RutasPermitidas, true) &&
		ListaCerradaValida(origen.CamposHandoffPermitidos, false)
}

// BytesConfiguracionOrigen fija una representacion estable. Las listas se
// tratan como conjuntos y se ordenan sobre copias defensivas.
func BytesConfiguracionOrigen(origen OrigenPasarelaCobroPublicado) ([]byte, error) {
	if !ConfiguracionOrigenValidaSinHuella(origen) {
		return nil, ErrInicioOperacionCobroInvalido
	}
	rutas := append([]string(nil), origen.RutasPermitidas...)
	campos := append([]string(nil), origen.CamposHandoffPermitidos...)
	sort.Strings(rutas)
	sort.Strings(campos)
	valor := struct {
		VersionEsquema int
		ID             string
		Version        int
		BaseHTTPS      string
		Rutas          []string
		Campos         []string
		PublicadaEn    string
	}{
		VersionEsquema: 1, ID: origen.ID, Version: origen.Version, BaseHTTPS: origen.BaseHTTPS,
		Rutas: rutas, Campos: campos, PublicadaEn: origen.PublicadaEn.UTC().Format(time.RFC3339Nano),
	}
	bytes, err := json.Marshal(valor)
	if err != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return append([]byte(nil), bytes...), nil
}

// HuellaConfiguracionOrigen deriva la huella SHA-256 de los bytes canonicos.
func HuellaConfiguracionOrigen(origen OrigenPasarelaCobroPublicado) (string, error) {
	bytes, err := BytesConfiguracionOrigen(origen)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(bytes)
	return fmt.Sprintf("%x", huella), nil
}

// ConfiguracionesOrigenIguales compara tambien la representacion canonica de
// las listas, por lo que su orden accidental no afecta al resultado.
func ConfiguracionesOrigenIguales(a, b OrigenPasarelaCobroPublicado) bool {
	if a.ID != b.ID || a.Version != b.Version || a.BaseHTTPS != b.BaseHTTPS ||
		a.HuellaConfiguracionSHA256 != b.HuellaConfiguracionSHA256 || !a.PublicadaEn.Equal(b.PublicadaEn) ||
		len(a.RutasPermitidas) != len(b.RutasPermitidas) || len(a.CamposHandoffPermitidos) != len(b.CamposHandoffPermitidos) {
		return false
	}
	bytesA, errA := BytesConfiguracionOrigen(a)
	bytesB, errB := BytesConfiguracionOrigen(b)
	return errA == nil && errB == nil && string(bytesA) == string(bytesB)
}

// IDAuditoria deriva el identificador de auditoria ligado a la mutacion.
func IDAuditoria(
	ordenRef string,
	version int,
	huellaPosterior string,
	hecho domain.TipoHechoCobro,
	accion domain.AccionCobro,
) string {
	contenido := fmt.Sprintf("vec.cobros.auditoria.v1\x00%s\x00%d\x00%s\x00%s\x00%s",
		ordenRef, version, huellaPosterior, hecho, accion)
	huella := sha256.Sum256([]byte(contenido))
	return fmt.Sprintf("aud_cob_%x", huella)
}

// IDEvento deriva el identificador de outbox ligado al ultimo hecho.
func IDEvento(
	ordenRef string,
	version int,
	secuencia int64,
	huellaHecho string,
	hecho domain.TipoHechoCobro,
	estado domain.EstadoCobro,
	accion domain.AccionCobro,
) string {
	contenido := fmt.Sprintf("vec.cobros.evento.v1\x00%s\x00%d\x00%d\x00%s\x00%s\x00%s\x00%s",
		ordenRef, version, secuencia, huellaHecho, hecho, estado, accion)
	huella := sha256.Sum256([]byte(contenido))
	return fmt.Sprintf("evt_cob_%x", huella)
}
