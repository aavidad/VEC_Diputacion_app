package consultafirmasv2

import (
	"context"
	"errors"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type ResultadoRecuperacion struct {
	Resultado
	Recuperaciones []ports.RecuperacionFirmaV2
}

type ServicioRecuperacion struct {
	fuente      FuenteContexto
	autorizador ports.AutorizadorRecuperacionFirmasV2
	lector      ports.LectorRecuperacionFirmasV2
}

func NuevaRecuperacion(f FuenteContexto, a ports.AutorizadorRecuperacionFirmasV2, l ports.LectorRecuperacionFirmasV2) (*ServicioRecuperacion, error) {
	if nulo(f) || nulo(a) || nulo(l) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &ServicioRecuperacion{f, a, l}, nil
}

// Recuperar exige una decisión nueva para esta operación y conserva los bytes
// históricos que entrega la autoridad. La fuente de contexto es nominal.
func (s *ServicioRecuperacion) Recuperar(ctx context.Context, q Solicitud) (ResultadoRecuperacion, error) {
	var cero ResultadoRecuperacion
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
	if err := ctx.Err(); err != nil {
		return cero, err
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
	capacidad, err := s.autorizador.AutorizarRecuperacionFirmasV2(ctx, m)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil {
		return cero, err
	}
	if err := firmaautorizacionv2.ValidarCapacidadRecuperacionFirmasV2(capacidad, m); err != nil {
		return cero, err
	}
	lectura, err := s.lector.RecuperarFirmasAutorizadasV2(ctx, m, capacidad)
	if err != nil {
		return cero, &falloLector{err}
	}
	if err := ctx.Err(); err != nil {
		return cero, &falloLector{err}
	}
	resultado, err := proyectarRecuperacion(m, lectura)
	if cancelada := ctx.Err(); cancelada != nil {
		return cero, &falloLector{cancelada}
	}
	return resultado, err
}

func proyectarRecuperacion(m ports.MaterialConsultaFirmasR5V2, l ports.LecturaRecuperacionFirmasV2) (ResultadoRecuperacion, error) {
	var cero ResultadoRecuperacion
	r, err := proyectar(m, l.LecturaFirmasR5V2)
	if err != nil {
		return cero, err
	}
	if len(l.Recuperaciones) != len(l.RevisionesPDF) || len(l.Recuperaciones) > MaximoFilas {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	porRef := make(map[string]Firma, len(r.Firmas))
	for _, f := range r.Firmas {
		porRef[f.FirmaRef] = f
	}
	registros := make(map[string]ports.FirmaRegistrada, len(l.Firmas))
	for _, f := range l.Firmas {
		registros[f.FirmaRef] = f
	}
	vistos := make(map[string]bool, len(l.Recuperaciones))
	for _, v := range l.Recuperaciones {
		f, ok := porRef[v.FirmaRef]
		if !ok || vistos[v.FirmaRef] || f.RevisionPDF == nil {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		if err := validarRecuperacion(m, f, registros[v.FirmaRef], v); err != nil {
			return cero, err
		}
		vistos[v.FirmaRef] = true
	}
	for _, f := range r.Firmas {
		if f.RevisionPDF != nil && !vistos[f.FirmaRef] {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	return ResultadoRecuperacion{r, append([]ports.RecuperacionFirmaV2(nil), l.Recuperaciones...)}, nil
}

func validarRecuperacion(m ports.MaterialConsultaFirmasR5V2, f Firma, registro ports.FirmaRegistrada, v ports.RecuperacionFirmaV2) error {
	if !domain.HuellaSHA256FirmaValida(v.MaterialRootSHA256) ||
		!strings.HasPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:") ||
		!domain.HuellaSHA256FirmaValida(strings.TrimPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:")) ||
		len(v.CanonNominal) < 512 {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	// La autoridad común comprueba esquema, tipos, enlaces, vigencias, huella y
	// bytes canónicos exactos. Se devuelve el texto original, nunca el modelo.
	canon, err := vecdomain.RecuperarCanonCompetenciaFirmanteHistoricaV1([]byte(v.CanonNominal), v.CanonNominalSHA256)
	if err != nil {
		return errors.Join(ports.ErrResultadoFirmaDocumentoInvalido, err)
	}
	if f.Custodiado == nil || f.RevisionPDF == nil {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	r := canon.Recurso
	if !canon.FechaHistorica.Equal(registro.RegistradaEn) {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	if canon.Identidad.CertificadoDERSHA256 != registro.CertificadoHuella ||
		canon.PasoRef != registro.PasoRef ||
		!ordenCanonValido(canon.PasoOrden, registro.PasoOrden) ||
		r.OrganizacionRef != m.OrganizacionRef || r.ExpedienteRef != m.ExpedienteRef ||
		r.ModuloID != ports.ModuloContratacion ||
		r.DocumentoRef != f.Original.Ref ||
		r.Original.Referencia != f.Original.Ref ||
		r.Original.Version != f.Original.Version ||
		r.Original.HuellaSHA256 != f.Original.SHA256 ||
		r.Firmado.Referencia != f.Custodiado.Ref ||
		r.Firmado.Version != f.Custodiado.Version ||
		r.Firmado.HuellaSHA256 != f.Custodiado.SHA256 ||
		!ordenCanonValido(r.NumeroFirmas, f.RevisionPDF.OrdenFirma) ||
		canon.Circuito.Referencia != f.CatalogoRef ||
		canon.Circuito.HuellaSHA256 != f.CatalogoHuella {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	if f.RevisionPDF.OrdenFirma == 1 {
		if r.EntradaRevision == nil {
			return nil
		}
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	if r.EntradaRevision != nil &&
		r.EntradaRevision.Referencia == f.RevisionPDF.Entrada.Ref &&
		r.EntradaRevision.Version == f.RevisionPDF.Entrada.Version &&
		r.EntradaRevision.HuellaSHA256 == f.RevisionPDF.Entrada.SHA256 {
		return nil
	}
	return ports.ErrResultadoFirmaDocumentoInvalido
}

func ordenCanonValido(canon uint64, orden int) bool {
	return orden == 1 && canon == 1 || orden == 2 && canon == 2
}
