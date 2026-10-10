package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrServicioVinculoEmisionBolsaNoDisponible = errors.New("ct: servicio de vínculo de emisión Bolsa no disponible")

type ServicioVinculoEmisionBolsa struct {
	autorizador ports.AutorizadorVinculoEmisionBolsa
	repositorio ports.RepositorioVinculoEmisionBolsa
}

func NuevoServicioVinculoEmisionBolsa(a ports.AutorizadorVinculoEmisionBolsa, r ports.RepositorioVinculoEmisionBolsa) (*ServicioVinculoEmisionBolsa, error) {
	if dependenciaNula(a) || dependenciaNula(r) {
		return nil, ErrServicioVinculoEmisionBolsaNoDisponible
	}
	return &ServicioVinculoEmisionBolsa{autorizador: a, repositorio: r}, nil
}

func (s *ServicioVinculoEmisionBolsa) Vincular(ctx context.Context, solicitud ports.SolicitudVinculoEmisionBolsa) (ports.ReciboVinculoEmisionBolsa, error) {
	var vacio ports.ReciboVinculoEmisionBolsa
	if ctx == nil || s == nil || ValidarSolicitudVinculoEmisionBolsa(solicitud) != nil {
		return vacio, ports.ErrVinculoEmisionBolsaInvalido
	}
	material, materialSHA256, err := s.repositorio.CodificarMaterialVinculoEmisionBolsa(solicitud)
	if err != nil {
		return vacio, err
	}
	defer clear(material)
	autorizacion, err := s.autorizador.AutorizarVinculoEmisionBolsa(ctx, solicitud, materialSHA256)
	if err != nil {
		return vacio, err
	}
	recibo, err := s.repositorio.RegistrarVinculoEmisionBolsa(ctx, material, autorizacion)
	if err != nil {
		return vacio, err
	}
	if ValidarReciboVinculoEmisionBolsa(recibo, solicitud) != nil {
		return vacio, ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return recibo, nil
}

func ValidarSolicitudVinculoEmisionBolsa(s ports.SolicitudVinculoEmisionBolsa) error {
	datos := domain.DatosVinculoEmisionBolsa{OrganizacionRef: s.OrganizacionRef,
		ExpedienteRef: s.ExpedienteRef, VersionEsperada: s.VersionEsperada,
		BolsaRef: s.BolsaRef, LlamamientoRef: s.LlamamientoRef,
		ReciboEmisionRef: s.ReciboEmisionRef, ClaveIdempotencia: s.ClaveIdempotencia}
	if datos.Validar() != nil {
		return ports.ErrVinculoEmisionBolsaInvalido
	}
	return nil
}

func ValidarReciboVinculoEmisionBolsa(r ports.ReciboVinculoEmisionBolsa, s ports.SolicitudVinculoEmisionBolsa) error {
	if ValidarSolicitudVinculoEmisionBolsa(s) != nil || r.ExpedienteRef != s.ExpedienteRef ||
		r.BolsaRef != s.BolsaRef || r.LlamamientoRef != s.LlamamientoRef ||
		r.ReciboEmisionRef != s.ReciboEmisionRef ||
		!domain.ReferenciaOpacaValida(r.ReciboVinculoRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.VinculadoEn) {
		return ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return nil
}

// La autoridad obtiene centro y categoría del expediente ya autorizado.
func NuevoRecursoVinculoEmisionBolsa(s ports.SolicitudVinculoEmisionBolsa, materialSHA256, centroRef, categoriaRef string) (core.RecursoAutorizable, error) {
	if ValidarSolicitudVinculoEmisionBolsa(s) != nil || len(materialSHA256) != 64 ||
		!domain.ReferenciaOpacaValida(centroRef) || !domain.ReferenciaOpacaValida(categoriaRef) {
		return core.RecursoAutorizable{}, ports.ErrVinculoEmisionBolsaInvalido
	}
	r := core.RecursoAutorizable{Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal",
		Tipo:      ports.TipoRecursoVinculoEmisionBolsa,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef, "centro_ref": centroRef, "categoria_ref": categoriaRef},
		Atributos: map[string]string{"material_sha256": materialSHA256}}
	return r, nil
}
