package httpcopias

import (
	"context"
	"log/slog"
	"time"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// RevisionPreparada checks the informed-review model, not permission or truth
// of the observations. FuenteRevision owns provenance and fresh observations.
func RevisionPreparada(v p.Propuesta, now time.Time) bool {
	m := v.MetadatosRevision
	if m == nil || v.Version == 0 || !v.CopiaPreviaRequerida || !v.DobleControl || v.Estado == "" ||
		m.PropuestaRef != v.PropuestaRef || m.PropuestaHuellaSHA256 != v.HuellaSHA256 ||
		m.ConjuntoRef != v.ConjuntoRef || m.ConjuntoHuellaSHA256 != v.ConjuntoHuellaSHA256 ||
		m.DestinoRef != v.DestinoRef || m.PreimagenSHA256 != v.PreimagenSHA256 {
		return false
	}
	for _, ref := range []string{v.PropuestaRef, v.ConjuntoRef, v.DestinoRef, v.PoliticaRef, v.MotivoRef, v.VentanaRef} {
		if !referenciaSegura(ref) {
			return false
		}
	}
	for _, hash := range []string{v.HuellaSHA256, v.ConjuntoHuellaSHA256, v.PreimagenSHA256, v.PoliticaHuellaSHA256} {
		if !sha256Valida(hash) {
			return false
		}
	}
	for _, stamp := range []time.Time{m.FechaCopia, m.PerdidaDesde, m.ObservadaEn, v.CaducaEn, v.VentanaInicio, v.VentanaFin} {
		if !instanteUTC(stamp) {
			return false
		}
	}
	if m.FechaCopia.After(m.ObservadaEn) || m.PerdidaDesde.After(m.ObservadaEn) || m.ObservadaEn.After(now) || !now.Before(v.CaducaEn) || !v.VentanaInicio.Before(v.VentanaFin) || v.CaducaEn.After(v.VentanaFin) {
		return false
	}
	if v.PerdidaDesde != nil && !v.PerdidaDesde.Equal(m.PerdidaDesde) {
		return false
	}
	if !versionCompleta(m.Actual) || !versionCompleta(m.Resultante) || m.Compatibilidad.Estado != "compatible" || len(m.Compatibilidad.Razones) < 1 || len(m.Compatibilidad.Razones) > 32 {
		return false
	}
	for _, key := range m.Compatibilidad.Razones {
		switch key {
		case "api.admin.copias.compatibilidad.compatible", "api.admin.copias.compatibilidad.comprobacion_conjunto_compatible":
		default:
			return false
		}
	}
	return true
}
func versionCompleta(v p.VersionObservada) bool {
	return referenciaSegura(v.ReleaseRef) && referenciaSegura(v.EsquemaRef) && versionSegura(v.AppVersion) && versionSegura(v.PostgreSQLVersion) && sha256Valida(v.DescriptorHuellaSHA256)
}
func versionSegura(v string) bool {
	if len(v) < 1 || len(v) > 96 {
		return false
	}
	for _, c := range v {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' || c == '+' {
			continue
		}
		return false
	}
	return true
}
func referenciaSegura(v string) bool {
	if len(v) < 1 || len(v) > 192 || v[0] == '.' || v[0] == ':' || v[0] == '-' || v[0] == '_' {
		return false
	}
	for _, c := range v {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' || c == ':' {
			continue
		}
		return false
	}
	return true
}
func sha256Valida(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c >= 'a' && c <= 'f' || c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}
func instanteUTC(v time.Time) bool { _, offset := v.Zone(); return !v.IsZero() && offset == 0 }

// PrepararRevision obtains the current canonical record before handing control
// to the backend. Missing metadata remains unavailable; it never approves.
func (s *Servicio) PrepararRevision(ctx context.Context, ses p.Sesion, ref string, request p.SolicitudControl) (p.Propuesta, error) {
	if s == nil || Ausente(s.FuenteRevision) {
		return p.Propuesta{}, p.ErrNoDisponible
	}
	if err := s.Autorizar(ctx, ses, p.Consultar, ref); err != nil {
		return p.Propuesta{}, err
	}
	v, err := s.FuenteRevision.PropuestaParaRevision(ctx, ses, ref)
	if err != nil {
		return p.Propuesta{}, err
	}
	if ctx.Err() != nil {
		return p.Propuesta{}, p.ErrNoDisponible
	}
	if v.PropuestaRef != ref || v.DestinoRef != request.DestinoRef || v.HuellaSHA256 != request.PropuestaHuellaSHA256 || v.Version != request.VersionEsperada {
		return p.Propuesta{}, p.ErrConflicto
	}
	if !RevisionPreparada(v, time.Now().UTC()) {
		return p.Propuesta{}, p.ErrNoDisponible
	}
	return v, nil
}
func (s *Servicio) revisarDisponible(ctx context.Context, ses p.Sesion) bool {
	if s == nil || Ausente(s.FuenteRevision) || Ausente(s.Control) || Ausente(s.Lecturas) {
		return false
	}
	propuestas, err := s.Lecturas.Propuestas(ctx, ses)
	if err != nil {
		slog.Warn("copias_revision_no_disponible", "causa", "consulta_propuestas_fallida")
		return false
	}
	if len(propuestas) > 100 {
		return false
	}
	for _, v := range propuestas {
		if s.Autorizar(ctx, ses, p.Revisar, v.PropuestaRef) != nil {
			continue
		}
		canonical, err := s.PrepararRevision(ctx, ses, v.PropuestaRef, p.SolicitudControl{DestinoRef: v.DestinoRef, PropuestaHuellaSHA256: v.HuellaSHA256, VersionEsperada: v.Version})
		if err == nil && RevisionPreparada(canonical, time.Now().UTC()) {
			return true
		}
	}
	return false
}
