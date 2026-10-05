package application

import (
	"bytes"
	"context"

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
	MaterialMultiple   *ports.MaterialFirmaVerificadaV2
	Material           ports.MaterialFirmaVec
	MotivoVerificacion docports.MotivoVerificacionFirma
	Custodiado         ports.DocumentoCustodiado
}

// ServicioFirmaVec prepara la vía R5 ligada al original. No utiliza
// CT118/AD3-85. El montaje espera el alcance aprobado de la política de
// coincidencia de persona, AD159 y la autoridad nominal de competencia.
type ServicioFirmaVec struct {
	base        *ServicioFirmaDocumento
	registro    ports.RegistroFirmasVec
	consulta    ports.AutorizadorConsultaFirmasR5
	autorizador ports.AutorizadorFirmaVec
	competencia ports.FuenteCompetenciaFirmante
	politica    ports.FuentePoliticaMismaPersonaEnPasos
	multiple    *dependenciasFirmaMultipleR5
}

func (s *ServicioFirmaVec) ComponerPoliticaMismaPersonaEnPasos(f ports.FuentePoliticaMismaPersonaEnPasos) error {
	if s == nil || nula(f) || s.politica != nil {
		return ErrCircuitoFirmaNoDisponible
	}
	s.politica = f
	return nil
}

func NuevoServicioFirmaVec(
	base *ServicioFirmaDocumento, registro ports.RegistroFirmasVec,
	consulta ports.AutorizadorConsultaFirmasR5,
	autorizador ports.AutorizadorFirmaVec, competencia ports.FuenteCompetenciaFirmante,
) (*ServicioFirmaVec, error) {
	if base == nil || nula(registro) || nula(consulta) || nula(autorizador) || nula(competencia) ||
		base.original == nil || base.verificador == nil || base.custodio == nil ||
		len(base.tiposCustodia) == 0 {
		return nil, ports.ErrRegistroFirmaVecNoDisponible
	}
	return &ServicioFirmaVec{base: base, registro: registro, consulta: consulta, autorizador: autorizador, competencia: competencia}, nil
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
	if s.multiple != nil {
		r, err := registrarFirmaMultipleR5(ctx, s.base, s.competencia, s.politica, s.multiple, solicitudMultipleDesdeVec(sol))
		return ResultadoFirmaVec{Recibo: r.recibo, MaterialMultiple: r.material, MotivoVerificacion: r.motivo, Custodiado: r.custodiado}, err
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
	if !existe || sol.PasoOrden > len(doc.Pasos) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	preparacion, err := prepararFirmaR5(ctx, s.base, s.competencia,
		sol.OrganizacionRef, sol.ExpedienteRef, sol.Documento, sol.OriginalRef, sol.OriginalVersion,
		sol.PDFFirmado, circuito, doc.Pasos[sol.PasoOrden-1])
	if err != nil {
		return cero, err
	}
	// Una actualización del catálogo entre verificación y lectura cancela la
	// operación; no se aplica una competencia de otro paso.
	preparadoCatalogoRef, preparadoCatalogoHuella := circuito.CatalogoRef, circuito.HuellaCatalogo
	lectura := ports.MaterialConsultaFirmasR5{OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef,
		VersionExpediente: sol.VersionExpediente, Documento: sol.Documento,
		FirmantePrincipalCandidatoRef: preparacion.competencia.FirmantePrincipalRef,
		ClaveIdempotencia:             sol.ClaveIdempotencia, PasoOrden: sol.PasoOrden,
		CatalogoHuella: circuito.HuellaCatalogo}
	capacidadLectura, err := s.consulta.AutorizarConsultaFirmasR5(ctx, lectura)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadConsultaFirmasR5(capacidadLectura, lectura); err != nil {
		return cero, err
	}
	lecturaFirmas, err := s.registro.ConsultarFirmasAutorizadas(ctx, lectura, capacidadLectura)
	if err != nil {
		return cero, err
	}
	if err := validarProyeccionFirmasR5(lectura, lecturaFirmas); err != nil {
		return cero, err
	}
	actualCircuito, err := s.base.circuitoValido(ctx)
	if err != nil || actualCircuito.CatalogoRef != preparadoCatalogoRef ||
		actualCircuito.HuellaCatalogo != preparadoCatalogoHuella {
		return cero, ErrCircuitoFirmaNoDisponible
	}
	firmas := lecturaFirmas.Firmas
	tipo, custodiar := s.base.tiposCustodia[sol.Documento]
	if !custodiar || tipo == "" {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	var secuencia int
	var originalEsperado string
	historiaRevision, historiaHuella := lecturaFirmas.HistoriaRevision, lecturaFirmas.HistoriaHuella
	previa, repetida := firmaConClave(firmas, sol.Documento, sol.ClaveIdempotencia)
	if repetida {
		if previa.Via != ports.ViaFirmaCertificadoVEC || previa.PasoOrden != sol.PasoOrden ||
			previa.Resultado != domain.ResultadoFirmaFirmado || previa.CatalogoHuella != circuito.HuellaCatalogo ||
			previa.ExpedienteVersion != sol.VersionExpediente || len(firmas) != 1 {
			return cero, ports.ErrClaveFirmaDocumentoUsada
		}
		secuencia, originalEsperado = previa.Secuencia, previa.OriginalHuella
		historiaRevision, historiaHuella = previa.HistoriaRevision, previa.HistoriaHuella
	} else {
		rondas, err := s.base.inicioRonda(ctx, sol.OrganizacionRef, sol.ExpedienteRef)
		if err != nil {
			return cero, err
		}
		if err := validarOriginalNuevoTrasReparo(firmas, sol.Documento, sol.PasoOrden, rondas[sol.Documento],
			sol.OriginalRef, sol.OriginalVersion, preparacion.original.HuellaSHA256); err != nil {
			return cero, err
		}
		estado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(
			doc, circuito.HuellaCatalogo, eventosDocumento(firmas, sol.Documento), rondas[sol.Documento])
		if err != nil {
			return cero, err
		}
		if estado.Completo || estado.PasoPendiente != sol.PasoOrden {
			return cero, ErrPasoFirmaNoPendiente
		}
		secuencia, originalEsperado = estado.UltimaSecuencia+1, estado.OriginalEsperadoHuella
		if !antecedentesR5Acreditados(estado, firmas, sol.Documento, sol.PasoOrden, sol.OriginalRef, sol.OriginalVersion, originalEsperado) {
			return cero, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		permite, err := ResolverPoliticaMismaPersonaEnPasos(ctx, s.politica, circuito.CatalogoRef, circuito.HuellaCatalogo)
		if err != nil {
			return cero, err
		}
		if err := EvaluarCoincidenciaPersonaR5(permite, lecturaFirmas.CoincideFirmanteEnOtroPaso, !lecturaFirmas.HistoriaSeparacionAcreditada); err != nil {
			return cero, err
		}
	}
	paso := doc.Pasos[sol.PasoOrden-1]
	original, dictamen, evidencia := preparacion.original, preparacion.dictamen, preparacion.competencia
	if originalEsperado != "" && original.HuellaSHA256 != originalEsperado {
		return cero, ports.ErrCadenaFirmaDocumentoRota
	}
	r := dictamen.Resultado
	m := ports.MaterialFirmaVec(ports.MaterialFirmaExterna{
		Via:             ports.ViaFirmaCertificadoVEC,
		OrganizacionRef: sol.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, VersionExpediente: sol.VersionExpediente,
		Documento: sol.Documento, CatalogoRef: circuito.CatalogoRef, CatalogoHuella: circuito.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, Secuencia: secuencia,
		HistoriaRevision: historiaRevision, HistoriaHuella: historiaHuella,
		OriginalRef: sol.OriginalRef, OriginalVersion: sol.OriginalVersion, OriginalHuella: original.HuellaSHA256,
		FirmadoHuella: r.HuellaFirmadoSHA256, CertificadoHuella: r.CertificadoHuellaSHA256, FirmanteRef: r.FirmanteRef,
		FirmantePrincipalRef: evidencia.FirmantePrincipalRef, PerfilFirmanteRef: evidencia.PerfilFirmanteRef,
		CargoFirmante: evidencia.CargoFirmante, UnidadFirmanteRef: evidencia.UnidadFirmanteRef,
		PerfilActivoFirmanteRef: evidencia.PerfilActivoFirmanteRef,
		PuestoFirmanteRef:       evidencia.PuestoFirmanteRef, AmbitoFirmanteRef: evidencia.AmbitoFirmanteRef,
		AsignacionFirmanteRef: evidencia.AsignacionFirmanteRef, AsignacionFirmanteVersion: evidencia.AsignacionFirmanteVersion,
		AsignacionFirmanteHuella: evidencia.AsignacionFirmanteHuella,
		VersionRolFirmanteRef:    evidencia.VersionRolFirmanteRef, VersionRolFirmanteHuella: evidencia.VersionRolFirmanteHuella,
		ControlVigenciaFirmanteRef:      evidencia.ControlVigenciaFirmanteRef,
		ControlVigenciaFirmanteRevision: evidencia.ControlVigenciaFirmanteRevision,
		ControlVigenciaFirmanteHuella:   evidencia.ControlVigenciaFirmanteHuella,
		AsignacionVigenteDesde:          evidencia.AsignacionVigenteDesde, AsignacionVigenteHasta: evidencia.AsignacionVigenteHasta,
		ActoCompetenciaRef: evidencia.ActoCompetenciaRef, DelegacionRef: evidencia.DelegacionRef,
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

// ComponerFirmaMultiple fija las autoridades V2 antes de atender solicitudes.
func (s *ServicioFirmaVec) ComponerFirmaMultiple(
	v docports.VerificadorFirmasDocumento, r ports.RegistroFirmasVerificadasV2,
	a ports.AutorizadorFirmaVerificadaV2, c ports.AutorizadorConsultaFirmasR5V2,
	f ports.FuentePDFFirmaAnterior,
) error {
	if s == nil || s.multiple != nil {
		return ErrCircuitoFirmaNoDisponible
	}
	d, err := nuevasDependenciasFirmaMultipleR5(v, r, a, c, f)
	if err != nil {
		return err
	}
	s.multiple = d
	return nil
}
