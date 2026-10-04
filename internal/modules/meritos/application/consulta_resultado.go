package application

import (
	"encoding/hex"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// ValidarResultadoConsultaPropia es el cotejo compartido con el adaptador antes
// de COMMIT. La persistencia del recibo solo la acredita el retorno del puerto
// tras confirmar la transacción; una referencia aislada no acredita consumo V3.
func ValidarResultadoConsultaPropia(o ports.OrdenConsultaPropia, r ports.ResultadoConsultaPropia) error {
	if ValidarOrdenConsultaPropia(o) != nil {
		return ports.ErrConsultaNoDisponible
	}
	if r.Codigo == "denegada" {
		if r.HechoActual == nil && r.ReciboConsulta == nil {
			return nil
		}
		return ports.ErrConsultaNoDisponible
	}
	if r.Codigo != "obtenida" && r.Codigo != "no_encontrada" || r.ReciboConsulta == nil {
		return ports.ErrConsultaNoDisponible
	}
	c := r.ReciboConsulta
	d, err := o.Autorizacion.Solicitud.Datos()
	if err != nil {
		return ports.ErrConsultaNoDisponible
	}
	correlacion, err := d.Correlacion.ValorCanonico()
	resumen := o.Autorizacion.Material.ResumenCapacidad()
	_, offset := c.ConsultadaEn.Zone()
	if err != nil || !domain.ReferenciaValida(c.Referencia) || c.HechoRef != o.HechoRef ||
		c.DecisionRef != resumen.DecisionRef() || !domain.ReferenciaValida(c.AuditoriaRef) ||
		c.CorrelacionRef != correlacion || !sha256ConsultaValida(c.ConsumoHuellaSHA256) ||
		c.ConsultadaEn.IsZero() || offset != 0 || c.ConsultadaEn.Before(resumen.EmitidaEn()) || !c.ConsultadaEn.Before(resumen.ExpiraEn()) {
		return ports.ErrConsultaNoDisponible
	}
	if r.Codigo == "no_encontrada" {
		if r.HechoActual != nil || c.VersionConsultada != 0 {
			return ports.ErrConsultaNoDisponible
		}
		return nil
	}
	h := r.HechoActual
	if h == nil || validarFichaHechoPropio(*h) != nil || h.Referencia != o.HechoRef || c.VersionConsultada != h.Version {
		return ports.ErrConsultaNoDisponible
	}
	return nil
}

func sha256ConsultaValida(h string) bool {
	b, err := hex.DecodeString(h)
	return err == nil && len(b) == 32 && h == strings.ToLower(h)
}

// Valida la proyección cerrada sin reconstruir un Hecho con identidades
// inventadas. La titularidad y el origen del registro los comprueba el puerto
// autorizado en PostgreSQL, al seleccionar la versión actual de esa persona.
func validarFichaHechoPropio(h ports.FichaHechoPropio) error {
	if !domain.ReferenciaValida(h.Referencia) || h.Version < 1 || h.Version > 1<<31-1 ||
		!domain.ReferenciaValida(h.ConceptoRef) || !textoConsultaValido(h.Denominacion, 512) ||
		!domain.ReferenciaValida(h.Procedencia.FuenteRef) || !domain.ReferenciaValida(h.Procedencia.Version) ||
		!domain.ReferenciaValida(h.Procedencia.HechoOrigenRef) || !instanteConsultaValido(h.Procedencia.CapturadaEn) ||
		!fechaConsultaValida(h.Vigencia.Desde) || h.Vigencia.Hasta != "" && (!fechaConsultaValida(h.Vigencia.Hasta) || h.Vigencia.Hasta < h.Vigencia.Desde) ||
		len(h.Evidencias) > 32 {
		return ports.ErrConsultaNoDisponible
	}
	switch h.Tipo {
	case "titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro":
	default:
		return ports.ErrConsultaNoDisponible
	}
	if h.Horas != nil && (*h.Horas < 0 || *h.Horas > 1<<31-1 || h.Tipo != "curso_asistencia" && h.Tipo != "curso_superacion") {
		return ports.ErrConsultaNoDisponible
	}
	switch h.Estado {
	case domain.Declarado, domain.Pendiente:
	case domain.Acreditado, domain.Rechazado:
		if h.Revision == nil || h.Estado == domain.Acreditado && len(h.Evidencias) == 0 {
			return ports.ErrConsultaNoDisponible
		}
	default:
		return ports.ErrConsultaNoDisponible
	}
	if h.Revision != nil && (!domain.ReferenciaValida(h.Revision.Referencia) || !domain.ReferenciaValida(h.Revision.MotivoRef) || !instanteConsultaValido(h.Revision.Fecha)) {
		return ports.ErrConsultaNoDisponible
	}
	seen := make(map[vec.ReferenciaDocumento]bool, len(h.Evidencias))
	for _, e := range h.Evidencias {
		if e.Validar() != nil || !domain.ReferenciaValida(e.ID) || e.Version > 1<<31-1 || seen[e] {
			return ports.ErrConsultaNoDisponible
		}
		seen[e] = true
	}
	return nil
}

func textoConsultaValido(s string, limite int) bool {
	return s != "" && len(s) <= limite && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}

func fechaConsultaValida(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10
}

func instanteConsultaValido(s string) bool {
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil && len(s) <= 40
}
