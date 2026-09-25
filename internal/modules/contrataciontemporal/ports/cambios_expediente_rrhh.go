package ports

import (
	"context"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Petición RRHH p.4: histórico del expediente con el valor anterior y el nuevo
// de cada campo cambiado entre versiones consecutivas. Los textos libres llegan
// solo como huella "sha256:..." desde la base (migración 000117).

// MaximoCambiosExpedienteRRHH acota la respuesta igual que la consulta SQL.
const MaximoCambiosExpedienteRRHH = 500

var (
	patronRutaCambioRRHH   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*|\[[0-9]{1,4}\])*$`)
	patronOrigenCambioRRHH = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
)

// ValorCambioProtegidoRRHH es la marca con la que la base entrega el valor de
// un campo fuera de la lista cerrada de campos visibles (texto libre, datos
// personales): solo dice que cambió, sin huella. Dos valores protegidos
// distintos llegan con la misma marca a ambos lados.
const ValorCambioProtegidoRRHH = "*protegido"

type CambioExpedienteRRHH struct {
	VersionExpediente uint64
	RegistradaEn      time.Time
	OrigenVersion     string
	OperacionRef      string
	Ruta              string
	ValorAnterior     *string
	ValorNuevo        *string
}

type ResultadoConsultaCambiosRRHH struct {
	ExpedienteRef     string
	VersionExpediente uint64
	Cambios           []CambioExpedienteRRHH
	// Recortado indica que había más cambios que los entregados (límite de
	// filas o de tamaño de la respuesta).
	Recortado             bool
	ConsumoHuellaSHA256   string
	AuditoriaRef          string
	AuditoriaHuellaSHA256 string
	ConsumidaEn           time.Time
}

func (ResultadoConsultaCambiosRRHH) String() string {
	return "[resultado-consulta-cambios-rrhh-redactado]"
}

func (ResultadoConsultaCambiosRRHH) GoString() string {
	return "[resultado-consulta-cambios-rrhh-redactado]"
}

// SesionConsultaCambiosRRHH lee los cambios con el mismo permiso que el
// detalle completo; la implementación SQL consume AD-3 antes de leer.
type SesionConsultaCambiosRRHH interface {
	ConsultarCambiosYRegistrar(context.Context, OrdenConsultaDetalleRRHH) (ResultadoConsultaCambiosRRHH, error)
}

func valorCambioRRHHValido(v *string) bool {
	return v == nil || (len(*v) <= 200 && !strings.ContainsAny(*v, "\x00\n\r"))
}

func (r ResultadoConsultaCambiosRRHH) ValidarPara(orden OrdenConsultaDetalleRRHH) error {
	if orden.capacidad.validaPara(orden.contexto,
		DominioHuellaConsultaDetalleRRHH, orden.consultaHuella,
		AccionConsultarDetalleRRHH, FinalidadConsultarDetalleRRHH,
		orden.solicitud.ExpedienteRef(), orden.instante) != nil ||
		orden.capacidad.AutorizaCamposSeguimiento() ||
		r.ExpedienteRef != orden.solicitud.ExpedienteRef() ||
		r.VersionExpediente < 1 || r.VersionExpediente > 9_007_199_254_740_991 ||
		(orden.solicitud.VersionObservada() != 0 && orden.solicitud.VersionObservada() != r.VersionExpediente) ||
		len(r.Cambios) > MaximoCambiosExpedienteRRHH ||
		!patronHuellaRRHH.MatchString(r.ConsumoHuellaSHA256) ||
		r.ConsumoHuellaSHA256 == strings.Repeat("0", 64) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!patronHuellaRRHH.MatchString(r.AuditoriaHuellaSHA256) ||
		r.AuditoriaHuellaSHA256 == strings.Repeat("0", 64) ||
		!domain.InstanteUTCCanonico(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(orden.instante) ||
		!r.ConsumidaEn.Before(orden.capacidad.ValidaHasta()) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	anterior := uint64(0)
	for _, c := range r.Cambios {
		if c.VersionExpediente < 2 || c.VersionExpediente > r.VersionExpediente || c.VersionExpediente < anterior ||
			!domain.InstanteUTCCanonico(c.RegistradaEn) || !patronOrigenCambioRRHH.MatchString(c.OrigenVersion) ||
			!domain.ReferenciaOpacaValida(c.OperacionRef) || len(c.Ruta) > 400 || !patronRutaCambioRRHH.MatchString(c.Ruta) ||
			!valorCambioRRHHValido(c.ValorAnterior) || !valorCambioRRHHValido(c.ValorNuevo) ||
			(c.ValorAnterior == nil && c.ValorNuevo == nil) ||
			(c.ValorAnterior != nil && c.ValorNuevo != nil && *c.ValorAnterior == *c.ValorNuevo && *c.ValorNuevo != ValorCambioProtegidoRRHH) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		anterior = c.VersionExpediente
	}
	return nil
}
