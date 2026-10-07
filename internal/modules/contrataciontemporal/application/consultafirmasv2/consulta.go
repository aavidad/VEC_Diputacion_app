// Package consultafirmasv2 recupera metadatos persistidos, sin registrar firmas.
package consultafirmasv2

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Contexto procede de una fuente nominal registrada y revalidada, nunca del
// certificado, el cargo o la petición. En consulta y recuperación HTTP la
// persona candidata es la del contexto registrado de quien consulta.
type Contexto struct {
	OrganizacionRef               string
	FirmantePrincipalCandidatoRef string
}

type FuenteContexto interface {
	ResolverContextoConsultaFirmasR5V2(context.Context) (Contexto, error)
}

// Lector recibe capacidad nueva. La composición debe envolver el repositorio
// con auditoriafirma.Registro: éxito auditado en TX y fallos después del rollback.
type Lector interface {
	ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error)
}

type Solicitud struct {
	ExpedienteRef     string
	VersionExpediente uint64
	Documento         string
	PasoOrden         int
	ClaveIdempotencia string
	CatalogoHuella    string
	Via               string
}

func (s Solicitud) Validar() error {
	if !domain.ReferenciaOpacaValida(s.ExpedienteRef) || s.VersionExpediente < 1 || s.VersionExpediente > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(s.Documento) || s.PasoOrden < 1 || s.PasoOrden > 2 ||
		!ports.ClaveIdempotenciaFirmaValida(s.ClaveIdempotencia) || !domain.HuellaSHA256FirmaValida(s.CatalogoHuella) ||
		(s.Via != ports.ViaFirmaCertificadoVEC && s.Via != ports.ViaFirmaExternaPortafirmas) {
		return ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

type Documento struct {
	Ref     string
	Version uint64
	SHA256  string
}

type RevisionPDF struct {
	OrdenFirma             int
	FirmaAnteriorRef       string
	ReciboAnteriorRef      string
	Entrada                Documento
	EntradaLongitud        uint64
	ByteRange              [4]uint64
	RevisionSHA256         string
	ContenidoFirmadoSHA256 string
	RevisionLongitud       uint64
	EvidenciaFirmasSHA256  string
}

type Firma struct {
	FirmaRef, ReciboRef, PasoRef string
	Documento                    string
	RegistradaEn                 time.Time
	Secuencia, PasoOrden         int
	VersionExpediente            uint64
	Via                          string
	Resultado                    domain.ResultadoFirmaDocumento
	CatalogoRef, CatalogoHuella  string
	Original                     Documento
	Custodiado                   *Documento
	RevisionPDF                  *RevisionPDF
}

type Resultado struct {
	ExpedienteRef, Documento, ExpedienteDocumentalRef string
	VersionExpediente, HistoriaRevision               uint64
	HistoriaHuella                                    string
	Firmas                                            []Firma
}

type Servicio struct {
	fuente      FuenteContexto
	autorizador ports.AutorizadorConsultaFirmasR5V2
	lector      Lector
}

func Nuevo(fuente FuenteContexto, autorizador ports.AutorizadorConsultaFirmasR5V2, lector Lector) (*Servicio, error) {
	if nulo(fuente) || nulo(autorizador) || nulo(lector) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &Servicio{fuente, autorizador, lector}, nil
}

func (s *Servicio) Consultar(ctx context.Context, q Solicitud) (Resultado, error) {
	var cero Resultado
	if ctx == nil || s == nil || nulo(s.fuente) || nulo(s.autorizador) || nulo(s.lector) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := q.Validar(); err != nil {
		return cero, err
	}
	c, err := s.fuente.ResolverContextoConsultaFirmasR5V2(ctx)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, err
	}
	if !domain.ReferenciaOpacaValida(c.OrganizacionRef) || !domain.ReferenciaOpacaValida(c.FirmantePrincipalCandidatoRef) ||
		!strings.HasPrefix(c.FirmantePrincipalCandidatoRef, "per_") {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	m := ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: c.OrganizacionRef, ExpedienteRef: q.ExpedienteRef, VersionExpediente: q.VersionExpediente,
		Documento: q.Documento, FirmantePrincipalCandidatoRef: c.FirmantePrincipalCandidatoRef,
		PasoOrden: q.PasoOrden, ClaveIdempotencia: q.ClaveIdempotencia, CatalogoHuella: q.CatalogoHuella}, Via: q.Via}
	if _, err := m.Canonico(); err != nil {
		return cero, err
	}
	capacidad, err := s.autorizador.AutorizarConsultaFirmasR5V2(ctx, m)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, err
	}
	if err := firmaautorizacionv2.ValidarCapacidadConsultaFirmasR5V2(capacidad, m); err != nil {
		return cero, err
	}
	lectura, err := s.lector.ConsultarFirmasAutorizadasV2(ctx, m, capacidad)
	// La cancelación no oculta un fallo de auditoría comunicado por el lector.
	if err != nil {
		return cero, &falloLector{err}
	}
	if ctx.Err() != nil {
		return cero, &falloLector{ctx.Err()}
	}
	return proyectar(m, lectura)
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

// El repositorio envuelto ya intentó auditar el fallo tras cerrar su TX. El
// transporte conserva la denegación/indisponibilidad sin duplicar ese intento.
type falloLector struct{ causa error }

func (e *falloLector) Error() string   { return ports.ErrRegistroFirmaDocumentoNoDisponible.Error() }
func (e *falloLector) Unwrap() error   { return e.causa }
func CubiertoPorLector(err error) bool { var e *falloLector; return errors.As(err, &e) }
