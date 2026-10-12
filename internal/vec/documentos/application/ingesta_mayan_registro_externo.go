package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const dominioReferenciaMeritosMayan = "vec.documentos.meritos.mayan.referencia.v1:"

// ServicioRegistroMayan emite la referencia D3 solo desde un plan confirmado
// que una fuente autorizada devuelve al actor V2. No recibe IDs Mayan,
// referencia de custodia, huella, tipo o política del cliente.
type ServicioRegistroMayan struct {
	Planes      ports.FuentePlanConfirmadoRegistroExterno
	Politicas   ports.PoliticaRegistroPlanMayan
	Registrador ports.RegistradorExternoMayan
	Autoridad   ports.AutorizadorRegistroExterno
}

func (s *ServicioRegistroMayan) Registrar(ctx context.Context, planRef string,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	vinculo vecdomain.VinculoAutenticacionActorV2) (domain.Documento, error) {
	if s == nil || ctx == nil || ctx.Err() != nil ||
		dependenciaNula(s.Planes) || dependenciaNula(s.Politicas) ||
		dependenciaNula(s.Registrador) || dependenciaNula(s.Autoridad) ||
		!domain.ReferenciaOpacaValida(planRef) ||
		resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	p, err := s.Planes.ConsultarPlanConfirmadoParaRegistro(ctx, planRef, resultado, vinculo)
	if err != nil {
		return domain.Documento{}, err
	}
	d := ports.DocumentoMayan{DocumentoID: p.DocumentoID, ArchivoID: p.ArchivoID,
		VersionID: p.VersionID, HuellaSHA256: p.HuellaSHA256, MIME: p.MIME, Tamano: p.Tamano}
	if p.PlanRef != planRef || p.ValidarPersistido() != nil || p.Estado != ports.EstadoIngestaConfirmada ||
		p.PersonaRef != resultado.Contexto.PersonaRef ||
		d.ValidarContra(p) != nil {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	solicitud, err := s.Politicas.SolicitudParaPlan(ctx, p)
	if err != nil {
		return domain.Documento{}, err
	}
	if solicitud.Validar() != nil || solicitud.ExpedienteRef() != p.ExpedienteRef ||
		solicitud.TipoDocumentalRef() != p.TipoRef || solicitud.PoliticaRef() != p.PoliticaRef ||
		int64(solicitud.VersionPolitica()) != p.VersionPolitica ||
		hex.EncodeToString(solicitud.HuellaPoliticaSHA256()) != p.HuellaPoliticaSHA256 {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	id := sha256.Sum256([]byte(dominioReferenciaMeritosMayan + planRef))
	alta := ports.AltaExterna{ID: "ref:" + hex.EncodeToString(id[:]),
		ClaveIdempotencia: planRef, ModuloID: "meritos", ExpedienteRef: p.ExpedienteRef,
		TipoRef: p.TipoRef, Version: 1, MIME: p.MIME, Tamano: p.Tamano,
		Custodia: domain.ReferenciaCustodiaExterna{CustodioID: "mayan",
			Referencia:   "mayan:" + d.DocumentoID + ":" + d.ArchivoID + ":" + d.VersionID,
			HuellaSHA256: p.HuellaSHA256},
		SolicitudPolitica: solicitud}
	r, err := s.Registrador.RegistrarExternoAutorizado(ctx, alta, s.Autoridad)
	if err != nil {
		return domain.Documento{}, err
	}
	if r.Validar() != nil || r.ID != alta.ID || r.ModuloID != alta.ModuloID ||
		r.ExpedienteRef != alta.ExpedienteRef || r.TipoRef != alta.TipoRef ||
		r.Version != alta.Version || r.MIME != alta.MIME || r.Tamano != alta.Tamano ||
		r.Custodia != domain.CustodiaExterna || r.CustodiaExternaRef != alta.Custodia ||
		r.PoliticaRef != p.PoliticaRef || int64(r.VersionPolitica) != p.VersionPolitica ||
		r.HuellaPoliticaSHA256 != p.HuellaPoliticaSHA256 ||
		!r.ConservacionHasta.Equal(p.ConservacionHasta) ||
		r.EstadoPolitica != p.EstadoPolitica || r.Proteccion != p.Proteccion {
		return domain.Documento{}, ports.ErrCapacidadNoDisponible
	}
	return r, nil
}
