package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La autoridad se configura con sesiones y perfiles de servidor. Ninguna
// exportacion V3 forma parte de la solicitud del canal.
type ServicioVinculoCategoriaRPT struct {
	autoridad ports.AutoridadVinculoCategoriaRPT
	fuente    ports.FuenteVinculoCategoriaRPT
	rpt       ports.FuentePublicacionCategoriaRPT
	reloj     ports.Reloj
}

func NuevoServicioVinculoCategoriaRPT(a ports.AutoridadVinculoCategoriaRPT, f ports.FuenteVinculoCategoriaRPT, r ports.FuentePublicacionCategoriaRPT, reloj ports.Reloj) (*ServicioVinculoCategoriaRPT, error) {
	if dependenciaNula(a) || dependenciaNula(f) || dependenciaNula(r) || dependenciaNula(reloj) {
		return nil, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return &ServicioVinculoCategoriaRPT{a, f, r, reloj}, nil
}

func (s *ServicioVinculoCategoriaRPT) Consultar(ctx context.Context, c ports.ConsultaVinculoCategoriaRPT) (ports.LecturaVinculoCategoriaRPT, error) {
	if s == nil || ctx == nil || c.Validar() != nil {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTInvalido
	}
	cap, e := s.autoridad.ConsultaCT(ctx, c)
	if e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, e
	}
	if !capacidadVinculoRPT(cap, ports.AccionConsultarVinculoCategoriaRPT, ports.AudienciaConsultarVinculoCategoriaRPT, s.reloj.Ahora()) {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	l, e := s.fuente.Consultar(ctx, c, cap)
	if e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, e
	}
	if l.ValidarPara(c) != nil {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return l, nil
}

func (s *ServicioVinculoCategoriaRPT) Registrar(ctx context.Context, m ports.RegistroVinculoCategoriaRPT) (ports.ReciboVinculoCategoriaRPT, error) {
	if s == nil || ctx == nil || m.Validar() != nil {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTInvalido
	}
	// La primera consulta solo prepara la intencion. La transaccion CT vuelve a
	// leer la publicacion mediante la autoridad RPT y coteja el analisis bloqueado.
	publicacion := m.Publicacion()
	precap, e := s.autoridad.LecturaRPT(ctx, publicacion)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if !capacidadVinculoRPT(precap, ports.AccionConsultarPublicacionCategoriaRPT, ports.AudienciaConsultarPublicacionCategoriaRPT, s.reloj.Ahora()) {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	preimagen, e := s.rpt.ConsultarPublicacionCategoriaRPT(ctx, publicacion, precap)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if preimagen != publicacion || !preimagen.CorrespondeA(m.CategoriaRef) {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTConflicto
	}
	capCT, e := s.autoridad.RegistroCT(ctx, m)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	capRPT, e := s.autoridad.LecturaRPT(ctx, publicacion)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	ahora := s.reloj.Ahora()
	if !capacidadVinculoRPT(capCT, ports.AccionRegistrarVinculoCategoriaRPT, ports.AudienciaRegistrarVinculoCategoriaRPT, ahora) ||
		!capacidadVinculoRPT(capRPT, ports.AccionConsultarPublicacionCategoriaRPT, ports.AudienciaConsultarPublicacionCategoriaRPT, ahora) ||
		capRPT.ResumenCapacidad().DecisionRef() == precap.ResumenCapacidad().DecisionRef() ||
		capCT.ResumenCapacidad().DecisionRef() == capRPT.ResumenCapacidad().DecisionRef() {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	r, e := s.fuente.Registrar(ctx, m, capCT, capRPT)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if r.ValidarPara(m) != nil {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return r, nil
}

func capacidadVinculoRPT(e vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, audiencia string, ahora time.Time) bool {
	if e.ValidarEstructura() != nil || !domain.InstanteUTCCanonico(ahora) {
		return false
	}
	r := e.ResumenCapacidad()
	return r.Operacion() == accion && r.AudienciaConsumo() == audiencia &&
		!ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}
