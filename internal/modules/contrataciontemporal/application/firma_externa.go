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

// SolicitudFirmaExterna describe el registro por RRHH de un PDF firmado fuera
// de VEC. No recibe firmante, competencia, original ni identidad registradora:
// estas proceden respectivamente del dictamen, fuente de competencia, fuente
// del original y decisión V3.
type SolicitudFirmaExterna struct {
	OrganizacionRef, ExpedienteRef                            string
	VersionExpediente                                         uint64
	Documento                                                 string
	PasoOrden                                                 int
	OriginalRef                                               string
	OriginalVersion                                           uint64
	PDFFirmado                                                []byte
	ReferenciaPortafirmasDeclarada, FechaPortafirmasDeclarada string
	ClaveIdempotencia                                         string
}

type ResultadoFirmaExterna struct {
	Recibo             ports.ReciboFirmaDocumento
	Material           ports.MaterialFirmaExterna
	MotivoVerificacion docports.MotivoVerificacionFirma
	Custodiado         ports.DocumentoCustodiado
}

// ServicioFirmaExterna utiliza el mismo catálogo/ronda/custodia/verificador de
// la firma VEC. Su registro consulta una historia unificada CT118+CT170.
type ServicioFirmaExterna struct {
	base        *ServicioFirmaDocumento
	registro    ports.RegistroFirmasExternas
	autorizador ports.AutorizadorRegistroFirmaExterna
	competencia ports.FuenteCompetenciaFirmante
}

func NuevoServicioFirmaExterna(
	base *ServicioFirmaDocumento,
	registro ports.RegistroFirmasExternas,
	autorizador ports.AutorizadorRegistroFirmaExterna,
	competencia ports.FuenteCompetenciaFirmante,
) (*ServicioFirmaExterna, error) {
	if base == nil || nula(registro) || nula(autorizador) || nula(competencia) ||
		base.original == nil || base.verificador == nil || base.custodio == nil ||
		len(base.tiposCustodia) == 0 {
		return nil, ports.ErrRegistroFirmaExternaNoDisponible
	}
	return &ServicioFirmaExterna{base: base, registro: registro, autorizador: autorizador, competencia: competencia}, nil
}

func obtenerOriginalFirmaAutorizado(
	ctx context.Context, fuente ports.FuenteOriginalFirmaAutorizado, q ports.SolicitudOriginalFirma,
) (ports.OriginalFirmaAutorizado, error) {
	var cero ports.OriginalFirmaAutorizado
	if ctx == nil || nula(fuente) || !domain.ReferenciaOpacaValida(q.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(q.ExpedienteRef) || !domain.ClaveDocumentoFirmaValida(q.Documento) ||
		!domain.ReferenciaOpacaValida(q.OriginalRef) || q.OriginalVersion == 0 || q.OriginalVersion > 9007199254740991 {
		return cero, ports.ErrOriginalFirmaNoAutorizado
	}
	o, err := fuente.ObtenerOriginalFirma(ctx, q)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ports.ErrOriginalFirmaNoAutorizado) {
			return cero, ports.ErrOriginalFirmaNoAutorizado
		}
		return cero, ports.ErrFuenteOriginalFirmaNoDisponible
	}
	if o.Solicitud != q || len(o.Contenido) == 0 || len(o.Contenido) > ports.MaximoDocumentoFirmaBytes ||
		!domain.HuellaSHA256FirmaValida(o.HuellaSHA256) || huella(o.Contenido) != o.HuellaSHA256 {
		return cero, ports.ErrOriginalFirmaNoAutorizado
	}
	o.Contenido = bytes.Clone(o.Contenido)
	return o, nil
}

func (s *ServicioFirmaExterna) estado(
	ctx context.Context, org, exp, documento string,
) (domain.CircuitoFirma, domain.CircuitoFirmaDocumento, domain.EstadoCircuitoDocumento, []ports.FirmaRegistrada, error) {
	circuito, err := s.base.circuitoValido(ctx)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, domain.EstadoCircuitoDocumento{}, nil, err
	}
	doc, ok := circuito.Documento(documento)
	if !ok {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, domain.EstadoCircuitoDocumento{}, nil, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	firmas, err := s.registro.ConsultarFirmas(ctx, org, exp)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, domain.EstadoCircuitoDocumento{}, nil, err
	}
	rondas, err := s.base.inicioRonda(ctx, org, exp)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, domain.EstadoCircuitoDocumento{}, nil, err
	}
	estado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(doc, circuito.HuellaCatalogo, eventosDocumento(firmas, documento), rondas[documento])
	return circuito, doc, estado, firmas, err
}

func (s *ServicioFirmaExterna) Registrar(ctx context.Context, sol SolicitudFirmaExterna) (ResultadoFirmaExterna, error) {
	var cero ResultadoFirmaExterna
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(sol.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(sol.ExpedienteRef) || sol.VersionExpediente == 0 ||
		!domain.ClaveDocumentoFirmaValida(sol.Documento) || sol.PasoOrden < 1 ||
		!domain.ReferenciaOpacaValida(sol.OriginalRef) || sol.OriginalVersion == 0 ||
		len(sol.PDFFirmado) == 0 || len(sol.PDFFirmado) > ports.MaximoDocumentoFirmaBytes ||
		!ports.ClaveIdempotenciaFirmaValida(sol.ClaveIdempotencia) ||
		!ports.ReferenciaPortafirmasDeclaradaValida(sol.ReferenciaPortafirmasDeclarada) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if _, ok := ports.FechaFirmaExternaCanonica(sol.FechaPortafirmasDeclarada); !ok {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	sol.PDFFirmado = bytes.Clone(sol.PDFFirmado)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	circuito, doc, estado, firmas, err := s.estado(ctx, sol.OrganizacionRef, sol.ExpedienteRef, sol.Documento)
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
		if previa.Via != ports.ViaFirmaExternaPortafirmas || previa.PasoOrden != sol.PasoOrden ||
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
	m := ports.MaterialFirmaExterna{
		Via:             ports.ViaFirmaExternaPortafirmas,
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
		ReferenciaPortafirmasDeclarada: sol.ReferenciaPortafirmasDeclarada,
		FechaPortafirmasDeclarada:      sol.FechaPortafirmasDeclarada, ClaveIdempotencia: sol.ClaveIdempotencia,
		DocumentoCustodiaRef:     ports.DocumentoCustodiaRef(sol.OrganizacionRef, sol.ExpedienteRef, sol.ClaveIdempotencia),
		DocumentoCustodiaVersion: ports.VersionDocumentoCustodiado,
	}
	if m.Validar() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	capacidad, err := s.autorizador.AutorizarRegistroFirmaExterna(ctx, m)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadFirmaExterna(capacidad, m); err != nil {
		return cero, err
	}
	custodiado, err := s.custodiar(ctx, sol, m, tipo)
	if err != nil {
		return cero, err
	}
	recibo, err := s.registro.RegistrarFirmaExterna(ctx, m, capacidad)
	if err != nil {
		return cero, err
	}
	h, _ := m.HuellaSHA256()
	if recibo.SolicitudHuella != h || recibo.Resultado != domain.ResultadoFirmaFirmado ||
		!domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
		!domain.ReferenciaOpacaValida(recibo.ActorRef) || !domain.ReferenciaOpacaValida(recibo.PerfilRef) ||
		recibo.RegistradaEn.IsZero() ||
		recibo.DocumentoCustodiaRef != m.DocumentoCustodiaRef ||
		recibo.DocumentoCustodiaVersion != m.DocumentoCustodiaVersion ||
		recibo.ActorRef == m.FirmantePrincipalRef ||
		recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return ResultadoFirmaExterna{Recibo: recibo, Material: m, MotivoVerificacion: dictamen.Motivo, Custodiado: custodiado}, nil
}

func (s *ServicioFirmaExterna) custodiar(ctx context.Context, sol SolicitudFirmaExterna, m ports.MaterialFirmaExterna, tipo string) (ports.DocumentoCustodiado, error) {
	return custodiarFirmaVerificada(s.base, ctx, sol.PDFFirmado, m, tipo)
}

func custodiarFirmaVerificada(base *ServicioFirmaDocumento, ctx context.Context, pdf []byte, m ports.MaterialFirmaExterna, tipo string) (ports.DocumentoCustodiado, error) {
	var cero ports.DocumentoCustodiado
	if huella(pdf) != m.FirmadoHuella {
		return cero, ports.ErrCustodiaFirmadoInvalida
	}
	orden := ports.OrdenCustodiaFirmado{
		DocumentoRef:      m.DocumentoCustodiaRef,
		ClaveIdempotencia: ports.ClaveCustodiaRef(m.OrganizacionRef, m.ExpedienteRef, m.ClaveIdempotencia),
		ExpedienteRef:     ports.ExpedienteDocumentalRef(m.OrganizacionRef, m.ExpedienteRef),
		TipoDocumental:    tipo, Version: m.DocumentoCustodiaVersion,
		Contenido: pdf, HuellaOriginalSHA256: m.OriginalHuella,
		FirmaOperacionRef: ports.OperacionFirmaRef(m.OrganizacionRef, m.ExpedienteRef, m.ClaveIdempotencia),
	}
	d, err := base.custodio.CustodiarFirmado(ctx, orden)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		for _, centinela := range []error{ports.ErrCustodiaFirmadoNoDisponible, ports.ErrCustodiaFirmadoInvalida, ports.ErrCustodiaFirmadoEnConflicto} {
			if errors.Is(err, centinela) {
				return cero, centinela
			}
		}
		return cero, ports.ErrCustodiaFirmadoDenegada
	}
	if d.Ref != orden.DocumentoRef || d.Version != orden.Version || d.HuellaSHA256 != m.FirmadoHuella {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return d, nil
}

func RecursoFirmaExterna(m ports.MaterialFirmaExterna) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return vecdomain.RecursoAutorizable{
		Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaExterna,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": h},
	}, nil
}

func ValidarCapacidadFirmaExterna(c ports.CapacidadFirmaExterna, m ports.MaterialFirmaExterna) error {
	material := c.ExportarMaterialParaConsumidor()
	recurso, err := RecursoFirmaExterna(m)
	if err != nil || material.ValidarEstructura() != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	contexto, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != ports.AccionRegistrarFirmaExterna ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != contexto ||
		resumen.AudienciaConsumo() != ports.AudienciaFirmaExternaV3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
