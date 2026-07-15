package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfiguracionCriptografiaCotejoInvalida = errors.New("seguridad: configuracion criptografica de cotejo invalida")
	ErrMaterialCriptograficoCotejoInvalido     = errors.New("seguridad: material criptografico de cotejo invalido")
	ErrCriptografiaCotejoCerrada               = errors.New("seguridad: criptografia de cotejo cerrada")
	ErrEntropiaCotejoNoDisponible              = errors.New("seguridad: entropia de cotejo no disponible")
)

const (
	longitudMinimaClaveHMACCotejo = 32
	maximoClavesHistoricasCotejo  = 15
	longitudValorCodigoCotejo     = 26
	bitsPorCaracterCodigoCotejo   = 5
	bytesIdentificadorCotejo      = 20
	alfabetoCodigoCotejo          = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

// ConfiguracionClaveHMACCotejo transporta una clave desde el ensamblado. Sus
// representaciones textuales y JSON siempre ocultan el material. El
// constructor del adaptador vuelve a copiarlo antes de conservarlo.
type ConfiguracionClaveHMACCotejo struct {
	Identificador string `json:"identificador"`
	Material      []byte `json:"-"`
}

func (ConfiguracionClaveHMACCotejo) String() string {
	return "seguridad.ConfiguracionClaveHMACCotejo{[MATERIAL-OCULTO]}"
}

func (ConfiguracionClaveHMACCotejo) GoString() string {
	return "seguridad.ConfiguracionClaveHMACCotejo{[MATERIAL-OCULTO]}"
}

func (c ConfiguracionClaveHMACCotejo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ConfiguracionClaveHMACCotejo) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"clave_hmac_cotejo","material":"oculto"}`), nil
}

func (ConfiguracionClaveHMACCotejo) MarshalText() ([]byte, error) {
	return []byte("[MATERIAL-HMAC-COTEJO-OCULTO]"), nil
}

func (c ConfiguracionClaveHMACCotejo) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// ConfiguracionCriptografiaCotejo separa expresamente la finalidad de indice
// de la de idempotencia. La primera clave historica se consulta antes que la
// segunda, pero solo ClaveIndiceActual se usa al emitir codigos nuevos.
type ConfiguracionCriptografiaCotejo struct {
	VersionGenerador       string                         `json:"version_generador"`
	ClaveIndiceActual      ConfiguracionClaveHMACCotejo   `json:"clave_indice_actual"`
	ClavesIndiceHistoricas []ConfiguracionClaveHMACCotejo `json:"claves_indice_historicas,omitempty"`
	ClaveSolicitud         ConfiguracionClaveHMACCotejo   `json:"clave_solicitud"`
}

func (ConfiguracionCriptografiaCotejo) String() string {
	return "seguridad.ConfiguracionCriptografiaCotejo{[MATERIAL-OCULTO]}"
}

func (ConfiguracionCriptografiaCotejo) GoString() string {
	return "seguridad.ConfiguracionCriptografiaCotejo{[MATERIAL-OCULTO]}"
}

func (c ConfiguracionCriptografiaCotejo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ConfiguracionCriptografiaCotejo) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"configuracion_criptografia_cotejo","material":"oculto"}`), nil
}

func (ConfiguracionCriptografiaCotejo) MarshalText() ([]byte, error) {
	return []byte("[CONFIGURACION-CRIPTOGRAFIA-COTEJO-OCULTA]"), nil
}

func (c ConfiguracionCriptografiaCotejo) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type claveHMACCotejo struct {
	identificador string
	material      []byte
}

// AdaptadorCriptograficoCotejo implementa generacion, indexacion e
// idempotencia sin custodiar el CSV. Para custodia se requiere otro adaptador
// respaldado por KMS, HSM o un gestor de secretos.
type AdaptadorCriptograficoCotejo struct {
	mu                     sync.RWMutex
	versionGenerador       string
	claveIndiceActual      claveHMACCotejo
	clavesIndiceHistoricas []claveHMACCotejo
	claveSolicitud         claveHMACCotejo
	cerrado                bool
}

func NuevoAdaptadorCriptograficoCotejo(
	configuracion ConfiguracionCriptografiaCotejo,
) (*AdaptadorCriptograficoCotejo, error) {
	if !configuracionCriptografiaCotejoValida(configuracion) {
		return nil, ErrConfiguracionCriptografiaCotejoInvalida
	}
	adaptador := &AdaptadorCriptograficoCotejo{
		versionGenerador:       configuracion.VersionGenerador,
		claveIndiceActual:      copiarClaveHMACCotejo(configuracion.ClaveIndiceActual),
		clavesIndiceHistoricas: make([]claveHMACCotejo, len(configuracion.ClavesIndiceHistoricas)),
		claveSolicitud:         copiarClaveHMACCotejo(configuracion.ClaveSolicitud),
	}
	for indice, clave := range configuracion.ClavesIndiceHistoricas {
		adaptador.clavesIndiceHistoricas[indice] = copiarClaveHMACCotejo(clave)
	}
	return adaptador, nil
}

// GenerarValorCodigoCotejo obtiene 26 simbolos uniformes de un alfabeto de 32
// elementos. Cada simbolo conserva cinco bits independientes: 130 bits reales
// de entropia, por encima del minimo contractual de 128.
func (a *AdaptadorCriptograficoCotejo) GenerarValorCodigoCotejo(
	ctx context.Context,
) (ports.ValorCodigoCotejoGenerado, error) {
	if err := validarContextoCriptografiaCotejo(ctx); err != nil {
		return ports.ValorCodigoCotejoGenerado{}, err
	}
	version, err := a.versionGeneradorDisponible()
	if err != nil {
		return ports.ValorCodigoCotejoGenerado{}, err
	}
	aleatorio := make([]byte, longitudValorCodigoCotejo)
	valor := make([]byte, longitudValorCodigoCotejo)
	defer borrarBytesCotejo(aleatorio)
	defer borrarBytesCotejo(valor)
	if _, err := rand.Read(aleatorio); err != nil {
		return ports.ValorCodigoCotejoGenerado{}, fmt.Errorf("%w: generar CSV", ErrEntropiaCotejoNoDisponible)
	}
	for indice := range aleatorio {
		valor[indice] = alfabetoCodigoCotejo[int(aleatorio[indice])&31]
	}
	secreto, err := ports.NuevoSecretoCodigoCotejo(string(valor))
	if err != nil {
		return ports.ValorCodigoCotejoGenerado{}, ErrMaterialCriptograficoCotejoInvalido
	}
	return ports.ValorCodigoCotejoGenerado{
		Secreto:          secreto,
		EntropiaBits:     longitudValorCodigoCotejo * bitsPorCaracterCodigoCotejo,
		VersionGenerador: version,
	}, nil
}

// NuevoIDCodigoCotejo crea una referencia opaca de 160 bits sin datos de
// persona, expediente, documento ni secuencias predecibles.
func (a *AdaptadorCriptograficoCotejo) NuevoIDCodigoCotejo() (string, error) {
	if _, err := a.versionGeneradorDisponible(); err != nil {
		return "", err
	}
	aleatorio := make([]byte, bytesIdentificadorCotejo)
	defer borrarBytesCotejo(aleatorio)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", fmt.Errorf("%w: generar identificador", ErrEntropiaCotejoNoDisponible)
	}
	return "codigo-cotejo-" + hex.EncodeToString(aleatorio), nil
}

func (a *AdaptadorCriptograficoCotejo) SellarIndiceCodigoCotejo(
	ctx context.Context,
	secreto ports.SecretoCodigoCotejo,
) (string, error) {
	if err := validarContextoCriptografiaCotejo(ctx); err != nil {
		return "", err
	}
	if secreto.Validar() != nil {
		return "", ErrMaterialCriptograficoCotejoInvalido
	}
	if a == nil {
		return "", ErrConfiguracionCriptografiaCotejoInvalida
	}
	material := []byte(secreto.Revelar())
	defer borrarBytesCotejo(material)
	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := a.validarDisponibleBloqueado(); err != nil {
		return "", err
	}
	return sellarHMACCotejo(a.claveIndiceActual, material), nil
}

func (a *AdaptadorCriptograficoCotejo) SellarIndicesConsultaCodigoCotejo(
	ctx context.Context,
	secreto ports.SecretoCodigoCotejo,
) ([]string, error) {
	if err := validarContextoCriptografiaCotejo(ctx); err != nil {
		return nil, err
	}
	if secreto.Validar() != nil {
		return nil, ErrMaterialCriptograficoCotejoInvalido
	}
	if a == nil {
		return nil, ErrConfiguracionCriptografiaCotejoInvalida
	}
	material := []byte(secreto.Revelar())
	defer borrarBytesCotejo(material)
	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := a.validarDisponibleBloqueado(); err != nil {
		return nil, err
	}
	indices := make([]string, 0, 1+len(a.clavesIndiceHistoricas))
	indices = append(indices, sellarHMACCotejo(a.claveIndiceActual, material))
	for _, clave := range a.clavesIndiceHistoricas {
		indices = append(indices, sellarHMACCotejo(clave, material))
	}
	return indices, nil
}

func (a *AdaptadorCriptograficoCotejo) SellarSolicitudCotejo(
	ctx context.Context,
	datos []byte,
) (string, error) {
	if err := validarContextoCriptografiaCotejo(ctx); err != nil {
		return "", err
	}
	if len(datos) == 0 {
		return "", ErrMaterialCriptograficoCotejoInvalido
	}
	if a == nil {
		return "", ErrConfiguracionCriptografiaCotejoInvalida
	}
	copia := append([]byte(nil), datos...)
	defer borrarBytesCotejo(copia)
	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := a.validarDisponibleBloqueado(); err != nil {
		return "", err
	}
	return sellarHMACCotejo(a.claveSolicitud, copia), nil
}

// Cerrar realiza un borrado logico de las copias de claves conservadas. Go no
// permite garantizar la eliminacion de copias internas del recolector o de la
// primitiva HMAC, por lo que esto no sustituye la custodia en KMS/HSM.
func (a *AdaptadorCriptograficoCotejo) Cerrar() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrado {
		return
	}
	borrarBytesCotejo(a.claveIndiceActual.material)
	a.claveIndiceActual = claveHMACCotejo{}
	for indice := range a.clavesIndiceHistoricas {
		borrarBytesCotejo(a.clavesIndiceHistoricas[indice].material)
		a.clavesIndiceHistoricas[indice] = claveHMACCotejo{}
	}
	a.clavesIndiceHistoricas = nil
	borrarBytesCotejo(a.claveSolicitud.material)
	a.claveSolicitud = claveHMACCotejo{}
	a.versionGenerador = ""
	a.cerrado = true
}

func (*AdaptadorCriptograficoCotejo) String() string {
	return "seguridad.AdaptadorCriptograficoCotejo{[MATERIAL-OCULTO]}"
}

func (*AdaptadorCriptograficoCotejo) GoString() string {
	return "seguridad.AdaptadorCriptograficoCotejo{[MATERIAL-OCULTO]}"
}

func (a *AdaptadorCriptograficoCotejo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}

func (*AdaptadorCriptograficoCotejo) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"adaptador_criptografico_cotejo","material":"oculto"}`), nil
}

func (*AdaptadorCriptograficoCotejo) MarshalText() ([]byte, error) {
	return []byte("[ADAPTADOR-CRIPTOGRAFICO-COTEJO-OCULTO]"), nil
}

func (a *AdaptadorCriptograficoCotejo) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func (a *AdaptadorCriptograficoCotejo) versionGeneradorDisponible() (string, error) {
	if a == nil {
		return "", ErrConfiguracionCriptografiaCotejoInvalida
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := a.validarDisponibleBloqueado(); err != nil {
		return "", err
	}
	return a.versionGenerador, nil
}

func (a *AdaptadorCriptograficoCotejo) validarDisponibleBloqueado() error {
	if a.cerrado {
		return ErrCriptografiaCotejoCerrada
	}
	if !versionGeneradorCotejoValida(a.versionGenerador) ||
		!claveHMACCotejoInternaValida(a.claveIndiceActual) ||
		!claveHMACCotejoInternaValida(a.claveSolicitud) {
		return ErrConfiguracionCriptografiaCotejoInvalida
	}
	return nil
}

func configuracionCriptografiaCotejoValida(configuracion ConfiguracionCriptografiaCotejo) bool {
	if !versionGeneradorCotejoValida(configuracion.VersionGenerador) ||
		!configuracionClaveHMACCotejoValida(configuracion.ClaveIndiceActual) ||
		!configuracionClaveHMACCotejoValida(configuracion.ClaveSolicitud) ||
		len(configuracion.ClavesIndiceHistoricas) > maximoClavesHistoricasCotejo {
		return false
	}
	clavesIndice := make([]ConfiguracionClaveHMACCotejo, 0, 1+len(configuracion.ClavesIndiceHistoricas))
	clavesIndice = append(clavesIndice, configuracion.ClaveIndiceActual)
	clavesIndice = append(clavesIndice, configuracion.ClavesIndiceHistoricas...)
	identificadores := make(map[string]struct{}, len(clavesIndice)+1)
	for indice, clave := range clavesIndice {
		if !configuracionClaveHMACCotejoValida(clave) {
			return false
		}
		if _, repetido := identificadores[clave.Identificador]; repetido {
			return false
		}
		identificadores[clave.Identificador] = struct{}{}
		for anterior := 0; anterior < indice; anterior++ {
			if hmac.Equal(clave.Material, clavesIndice[anterior].Material) {
				return false
			}
		}
		if hmac.Equal(clave.Material, configuracion.ClaveSolicitud.Material) {
			return false
		}
	}
	if _, repetido := identificadores[configuracion.ClaveSolicitud.Identificador]; repetido {
		return false
	}
	return true
}

func configuracionClaveHMACCotejoValida(clave ConfiguracionClaveHMACCotejo) bool {
	return identificadorClaveHMACCotejoValido(clave.Identificador) && len(clave.Material) >= longitudMinimaClaveHMACCotejo
}

func claveHMACCotejoInternaValida(clave claveHMACCotejo) bool {
	return identificadorClaveHMACCotejoValido(clave.identificador) && len(clave.material) >= longitudMinimaClaveHMACCotejo
}

func identificadorClaveHMACCotejoValido(identificador string) bool {
	if identificador == "" || identificador != strings.TrimSpace(identificador) || len(identificador) > 128 ||
		identificador[0] < 'a' || identificador[0] > 'z' {
		return false
	}
	for _, caracter := range identificador[1:] {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == '.' || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func versionGeneradorCotejoValida(version string) bool {
	if version == "" || version != strings.TrimSpace(version) || len(version) > 128 {
		return false
	}
	for _, caracter := range version {
		if caracter < 32 || caracter == 127 {
			return false
		}
	}
	return true
}

func copiarClaveHMACCotejo(clave ConfiguracionClaveHMACCotejo) claveHMACCotejo {
	return claveHMACCotejo{
		identificador: clave.Identificador,
		material:      append([]byte(nil), clave.Material...),
	}
}

func sellarHMACCotejo(clave claveHMACCotejo, datos []byte) string {
	mac := hmac.New(sha256.New, clave.material)
	_, _ = mac.Write(datos)
	suma := mac.Sum(nil)
	resultado := "hmac-sha256:" + clave.identificador + ":" + hex.EncodeToString(suma)
	borrarBytesCotejo(suma)
	return resultado
}

func validarContextoCriptografiaCotejo(ctx context.Context) error {
	if ctx == nil {
		return ErrMaterialCriptograficoCotejoInvalido
	}
	return ctx.Err()
}

func borrarBytesCotejo(datos []byte) {
	for indice := range datos {
		datos[indice] = 0
	}
}

var _ ports.GeneradorValorCodigoCotejo = (*AdaptadorCriptograficoCotejo)(nil)
var _ ports.GeneradorIDCodigoCotejo = (*AdaptadorCriptograficoCotejo)(nil)
var _ ports.SelladorIndiceCodigoCotejo = (*AdaptadorCriptograficoCotejo)(nil)
var _ ports.SelladorSolicitudCotejo = (*AdaptadorCriptograficoCotejo)(nil)
