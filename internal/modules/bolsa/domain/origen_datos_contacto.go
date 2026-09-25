package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Duda 45: los aspirantes importados de CONVOCA no tienen contacto propio en
// VEC. RRHH registra el correo y los teléfonos que trae de CONVOCA como una
// versión de B4 marcada «contacto de origen CONVOCA». La marca tiene la
// vigencia que fija la regla del catálogo y vale solo para su versión: una
// versión posterior (registrada por RRHH o confirmada por la persona) la
// sustituye. Vencida, el contacto se sigue usando, pero no está confirmado.

var ErrOrigenDatosContactoInvalido = errors.New("bolsa: origen de datos de contacto invalido")

// OrigenDatosContactoConvoca es el único origen marcado. Un contacto sin marca
// es propio: registrado por RRHH o confirmado por la persona.
const OrigenDatosContactoConvoca = "convoca"

// Estados visibles de la marca.
const (
	EstadoOrigenContactoVigente = "vigente"
	EstadoOrigenContactoVencido = "vencido"
)

var huellaReglaOrigenContacto = regexp.MustCompile(`^[0-9a-f]{64}$`)

// MarcaOrigenDatosContacto conserva la regla aplicada (referencia
// catálogo:versión:entrada y huella del catálogo) junto a la vigencia que dio.
type MarcaOrigenDatosContacto struct {
	Origen string
	// VigenteHasta es el primer instante en que la marca ya ha vencido.
	VigenteHasta time.Time
	// UltimoDia es el último día de vigencia en hora peninsular (AAAA-MM-DD).
	UltimoDia   string
	ReglaRef    string
	ReglaHuella string
}

func (m MarcaOrigenDatosContacto) Validar() error {
	if m.Origen != OrigenDatosContactoConvoca || m.VigenteHasta.IsZero() || m.ReglaRef == "" ||
		strings.TrimSpace(m.ReglaRef) != m.ReglaRef || len(m.ReglaRef) > 512 || !huellaReglaOrigenContacto.MatchString(m.ReglaHuella) {
		return ErrOrigenDatosContactoInvalido
	}
	if _, err := time.Parse(time.DateOnly, m.UltimoDia); err != nil {
		return ErrOrigenDatosContactoInvalido
	}
	return nil
}

// Vencida indica si en ese instante el contacto de origen ya no está vigente.
func (m MarcaOrigenDatosContacto) Vencida(instante time.Time) bool {
	return !instante.Before(m.VigenteHasta)
}

// Estado devuelve vigente o vencido en ese instante.
func (m MarcaOrigenDatosContacto) Estado(instante time.Time) string {
	if m.Vencida(instante) {
		return EstadoOrigenContactoVencido
	}
	return EstadoOrigenContactoVigente
}

// Igual compara dos marcas (replay idempotente).
func (m MarcaOrigenDatosContacto) Igual(otra MarcaOrigenDatosContacto) bool {
	return m.Origen == otra.Origen && m.VigenteHasta.Equal(otra.VigenteHasta) && m.UltimoDia == otra.UltimoDia &&
		m.ReglaRef == otra.ReglaRef && m.ReglaHuella == otra.ReglaHuella
}

// OrigenDatosContactoAdmitido valida el origen pedido en un alta: vacío (propio)
// o CONVOCA.
func OrigenDatosContactoAdmitido(origen string) bool {
	return origen == "" || origen == OrigenDatosContactoConvoca
}
