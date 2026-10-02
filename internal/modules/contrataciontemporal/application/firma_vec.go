package application

import (
	"bytes"
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// SolicitudFirmaVec recibe el PDF firmado en el equipo, pero nunca acepta
// original ni identidad del firmante enviados por el cliente.
type SolicitudFirmaVec struct {
	OrganizacionRef, ExpedienteRef string
	VersionExpediente              uint64
	Documento                      string
	PasoOrden                      int
	OriginalRef                    string
	OriginalVersion                uint64
	PDFFirmado                     []byte
	ClaveIdempotencia              string
}

type ResultadoFirmaVec struct {
	Recibo             ports.ReciboFirmaDocumento
	Material           ports.MaterialFirmaVec
	MotivoVerificacion docports.MotivoVerificacionFirma
	Custodiado         ports.DocumentoCustodiado
}

// ServicioFirmaVec es la vía R5 original-bound. No utiliza CT118/AD3-85.
type ServicioFirmaVec struct {
	base        *ServicioFirmaDocumento
	registro    ports.RegistroFirmasVec
	autorizador ports.AutorizadorFirmaVec
	competencia ports.FuenteCompetenciaFirmante
}

func NuevoServicioFirmaVec(
	base *ServicioFirmaDocumento, registro ports.RegistroFirmasVec,
	autorizador ports.AutorizadorFirmaVec, competencia ports.FuenteCompetenciaFirmante,
) (*ServicioFirmaVec, error) {
	if base == nil || nula(registro) || nula(autorizador) || nula(competencia) ||
		base.original == nil || base.verificador == nil || base.custodio == nil ||
		len(base.tiposCustodia) == 0 {
		return nil, ports.ErrRegistroFirmaVecNoDisponible
	}
	return &ServicioFirmaVec{base: base, registro: registro, autorizador: autorizador, competencia: competencia}, nil
}

func (s *ServicioFirmaVec) Firmar(ctx context.Context, sol SolicitudFirmaVec) (ResultadoFirmaVec, error) {
	var cero ResultadoFirmaVec
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(sol.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(sol.ExpedienteRef) || sol.VersionExpediente == 0 ||
		!domain.ClaveDocumentoFirmaValida(sol.Documento) || sol.PasoOrden < 1 ||
		!domain.ReferenciaOpacaValida(sol.OriginalRef) || sol.OriginalVersion == 0 ||
		len(sol.PDFFirmado) == 0 || len(sol.PDFFirmado) > ports.MaximoDocumentoFirmaBytes ||
		!ports.ClaveIdempotenciaFirmaValida(sol.ClaveIdempotencia) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	sol.PDFFirmado = bytes.Clone(sol.PDFFirmado)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	circuito, err := s.base.circuitoValido(ctx)
	if err != nil {
		return cero, err
	}
	doc, existe := circuito.Documento(sol.Documento)
	if !existe {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	firmas, err := s.registro.ConsultarFirmas(ctx, sol.OrganizacionRef, sol.ExpedienteRef)
	if err != nil {
		return cero, err
	}
	rondas, err := s.base.inicioRonda(ctx, sol.OrganizacionRef, sol.ExpedienteRef)
	if err != nil {
		return cero, err
	}
	estado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(
		doc, circuito.HuellaCatalogo, eventosDocumento(firmas, sol.Documento), rondas[sol.Documento],
	)
	if err != nil {
		return cero, err
	}
	tipo, custodiar := s.base.tiposCustodia[sol.Documento]
	if !custodiar || tipo == "" {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	secuencia, originalEsperado := estado.UltimaSecuencia+1, estado.OriginalEsperadoHuella
	previa, repetida := firmaConClave(firmas, sol.Documento, sol.ClaveIdempotencia)
	switch {
	case repetida:
		if previa.Via != ports.ViaFirmaCertificadoVEC || previa.PasoOrden != sol.PasoOrden ||
			previa.Resultado != domain.ResultadoFirmaFirmado || previa.CatalogoHuella != circuito.HuellaCatalogo ||
			previa.ExpedienteVersion != sol.VersionExpediente || sol.PasoOrden > len(doc.Pasos) {
			return cero, ports.ErrClaveFirmaDocumentoUsada
		}
		secuencia, originalEsperado = previa.Secuencia, previa.OriginalHuella
	case estado.Completo || estado.PasoPendiente != sol.PasoOrden:
		return cero, ErrPasoFirmaNoPendiente
	}
	paso := doc.Pasos[sol.PasoOrden-1]
	original, err := obtenerOriginalFirmaAutorizado(ctx, s.base.original, ports.SolicitudOriginalFirma{
		OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef,
		Documento: sol.Documento, OriginalRef: sol.OriginalRef, OriginalVersion: sol.OriginalVersion,
	})
	if err != nil {
		return cero, err
	}
	if originalEsperado != "" && original.HuellaSHA256 != originalEsperado {
		return cero, ports.ErrCadenaFirmaDocumentoRota
	}
	peticion := docports.SolicitudVerificacionFirma{
		DocumentoID: sol.OriginalRef, Version: sol.OriginalVersion, FormatoEsperado: "PAdES",
		HuellaOriginalSHA256: original.HuellaSHA256, ContenidoOriginal: original.Contenido,
		ContenidoFirmado: sol.PDFFirmado,
	}
	dictamen, err := s.base.verificador.VerificarMotivado(ctx, peticion)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, DictamenRechazado{Estado: docports.EstadoVerificacionIndeterminada, Motivo: docports.MotivoRespuestaNoInterpretable}
	}
	if dictamen.ValidarContra(peticion) != nil || dictamen.Motivo != docports.MotivoFirmaVerificada {
		motivo := dictamen.Motivo
		if motivo.EstadoAsociado() == "" || motivo == docports.MotivoFirmaVerificada {
			motivo = docports.MotivoRespuestaNoInterpretable
		}
		return cero, DictamenRechazado{Estado: dictamen.Resultado.Estado, Motivo: motivo}
	}
	r := dictamen.Resultado
	competenciaQ := ports.SolicitudCompetenciaFirmante{
		OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, Documento: sol.Documento,
		CatalogoRef: circuito.CatalogoRef, CatalogoHuella: circuito.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, CargoFirmante: paso.Cargo,
		PerfilFirmanteRef: paso.PerfilRef, FirmanteRef: r.FirmanteRef, CertificadoHuella: r.CertificadoHuellaSHA256,
	}
	evidencia, err := s.competencia.AcreditarCompetenciaFirmante(ctx, competenciaQ)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
			return cero, ports.ErrCompetenciaFirmanteNoAcreditada
		}
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if evidencia.Solicitud != competenciaQ || !evidencia.Vigente ||
		evidencia.CargoFirmante != paso.Cargo || evidencia.PerfilFirmanteRef != paso.PerfilRef {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	m := ports.MaterialFirmaVec(ports.MaterialFirmaExterna{
		Via:             ports.ViaFirmaCertificadoVEC,
		OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, VersionExpediente: sol.VersionExpediente,
		Documento: sol.Documento, CatalogoRef: circuito.CatalogoRef, CatalogoHuella: circuito.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, Secuencia: secuencia,
		OriginalRef: sol.OriginalRef, OriginalVersion: sol.OriginalVersion, OriginalHuella: original.HuellaSHA256,
		FirmadoHuella: r.HuellaFirmadoSHA256, CertificadoHuella: r.CertificadoHuellaSHA256, FirmanteRef: r.FirmanteRef,
		FirmantePrincipalRef: evidencia.FirmantePrincipalRef, PerfilFirmanteRef: evidencia.PerfilFirmanteRef,
		CargoFirmante: evidencia.CargoFirmante, UnidadFirmanteRef: evidencia.UnidadFirmanteRef,
		PuestoFirmanteRef: evidencia.PuestoFirmanteRef, AmbitoFirmanteRef: evidencia.AmbitoFirmanteRef,
		AsignacionFirmanteRef: evidencia.AsignacionFirmanteRef, AsignacionFirmanteVersion: evidencia.AsignacionFirmanteVersion,
		AsignacionFirmanteHuella: evidencia.AsignacionFirmanteHuella,
		AsignacionVigenteDesde:   evidencia.AsignacionVigenteDesde, AsignacionVigenteHasta: evidencia.AsignacionVigenteHasta,
		CompetenciaComprobadaEn: evidencia.CompetenciaComprobadaEn,
		ActoCompetenciaRef:      evidencia.ActoCompetenciaRef, DelegacionRef: evidencia.DelegacionRef,
		PoliticaVerificacion: ports.PoliticaVerificacionFirma,
		RevocacionEstado:     r.RevocacionEstado, SelloTiempoEstado: r.SelloTiempoEstado,
		ClaveIdempotencia:        sol.ClaveIdempotencia,
		DocumentoCustodiaRef:     ports.DocumentoCustodiaRef(sol.OrganizacionRef, sol.ExpedienteRef, sol.ClaveIdempotencia),
		DocumentoCustodiaVersion: ports.VersionDocumentoCustodiado,
	})
	if m.Validar() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	capacidad, err := s.autorizador.AutorizarFirmaVec(ctx, m)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadFirmaVec(capacidad, m); err != nil {
		return cero, err
	}
	custodiado, err := custodiarFirmaVerificada(s.base, ctx, sol.PDFFirmado, ports.MaterialFirmaExterna(m), tipo)
	if err != nil {
		return cero, err
	}
	recibo, err := s.registro.RegistrarFirmaVec(ctx, m, capacidad)
	if err != nil {
		return cero, err
	}
	h, _ := m.HuellaSHA256()
	if recibo.SolicitudHuella != h || recibo.Resultado != domain.ResultadoFirmaFirmado ||
		!domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
		!domain.ReferenciaOpacaValida(recibo.ActorRef) || !domain.ReferenciaOpacaValida(recibo.PerfilRef) ||
		recibo.RegistradaEn.IsZero() || recibo.DocumentoCustodiaRef != m.DocumentoCustodiaRef ||
		recibo.DocumentoCustodiaVersion != m.DocumentoCustodiaVersion ||
		recibo.ActorRef != m.FirmantePrincipalRef ||
		recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return ResultadoFirmaVec{Recibo: recibo, Material: m, MotivoVerificacion: dictamen.Motivo, Custodiado: custodiado}, nil
}

func RecursoFirmaVec(m ports.MaterialFirmaVec) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return vecdomain.RecursoAutorizable{
		Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaVec,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": h},
	}, nil
}

func ValidarCapacidadFirmaVec(c ports.CapacidadFirmaVec, m ports.MaterialFirmaVec) error {
	material := c.ExportarMaterialParaConsumidor()
	recurso, err := RecursoFirmaVec(m)
	if err != nil || material.ValidarEstructura() != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	contexto, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != ports.AccionRegistrarFirmaVec ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != contexto ||
		resumen.AudienciaConsumo() != ports.AudienciaFirmaVecV3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
