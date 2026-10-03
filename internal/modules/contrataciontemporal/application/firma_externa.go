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
	MaterialMultiple   *ports.MaterialFirmaVerificadaV2
	Material           ports.MaterialFirmaExterna
	MotivoVerificacion docports.MotivoVerificacionFirma
	Custodiado         ports.DocumentoCustodiado
}

// ServicioFirmaExterna utiliza el mismo catálogo/ronda/custodia/verificador de
// la firma VEC. Su registro consulta una historia unificada CT118+CT170. Es
// preparatorio: el montaje espera el alcance aprobado de la política de
// coincidencia de persona, AD159 y la autoridad nominal de competencia.
type ServicioFirmaExterna struct {
	base        *ServicioFirmaDocumento
	registro    ports.RegistroFirmasExternas
	consulta    ports.AutorizadorConsultaFirmasR5
	autorizador ports.AutorizadorRegistroFirmaExterna
	competencia ports.FuenteCompetenciaFirmante
	politica    ports.FuentePoliticaMismaPersonaEnPasos
	multiple    *dependenciasFirmaMultipleR5
}

// La fuente solo puede ampliar el default NO si devuelve un valor ligado a la
// referencia y huella publicadas. Se fija una vez antes del montaje.
func (s *ServicioFirmaExterna) ComponerPoliticaMismaPersonaEnPasos(f ports.FuentePoliticaMismaPersonaEnPasos) error {
	if s == nil || nula(f) || s.politica != nil {
		return ErrCircuitoFirmaNoDisponible
	}
	s.politica = f
	return nil
}

func NuevoServicioFirmaExterna(
	base *ServicioFirmaDocumento,
	registro ports.RegistroFirmasExternas,
	consulta ports.AutorizadorConsultaFirmasR5,
	autorizador ports.AutorizadorRegistroFirmaExterna,
	competencia ports.FuenteCompetenciaFirmante,
) (*ServicioFirmaExterna, error) {
	if base == nil || nula(registro) || nula(consulta) || nula(autorizador) || nula(competencia) ||
		base.original == nil || base.verificador == nil || base.custodio == nil ||
		len(base.tiposCustodia) == 0 {
		return nil, ports.ErrRegistroFirmaExternaNoDisponible
	}
	return &ServicioFirmaExterna{base: base, registro: registro, consulta: consulta, autorizador: autorizador, competencia: competencia}, nil
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

// El instante de esta lectura acredita que la fuente evaluó una asignación
// dentro de su ventana; no forma parte del material idempotente. AUT30
// revalidará vigencia y revocación con su propio reloj al confirmar el efecto.
func competenciaTemporalValida(e ports.EvidenciaCompetenciaFirmante) bool {
	desde, okDesde := ports.FechaFirmaExternaCanonica(e.AsignacionVigenteDesde)
	hasta, okHasta := ports.FechaFirmaExternaCanonica(e.AsignacionVigenteHasta)
	comprobada, okComprobada := ports.FechaFirmaExternaCanonica(e.CompetenciaComprobadaEn)
	return e.Vigente && okDesde && okHasta && okComprobada &&
		!comprobada.Before(desde) && comprobada.Before(hasta)
}

// Una fila CT118 legada no contiene persona acreditada ni vía R5. El estado
// visual del circuito puede incluirla, pero no habilita un paso R5 posterior.
func antecedentesR5Acreditados(estado domain.EstadoCircuitoDocumento, firmas []ports.FirmaRegistrada, documento string, pasoOrden int, originalRef string, originalVersion uint64, originalHuella string) bool {
	for orden := 1; orden < pasoOrden; orden++ {
		var recibo string
		for _, p := range estado.Pasos {
			if p.Orden == orden && p.Estado == domain.EstadoPasoFirmado {
				recibo = p.ReciboRef
				break
			}
		}
		if recibo == "" {
			return false
		}
		coincidencias := 0
		for _, f := range firmas {
			if f.Documento != documento || f.ReciboRef != recibo || f.PasoOrden != orden {
				continue
			}
			if f.Resultado != domain.ResultadoFirmaFirmado ||
				(f.Via != ports.ViaFirmaCertificadoVEC && f.Via != ports.ViaFirmaExternaPortafirmas) ||
				!f.FirmantePrincipalAcreditado ||
				f.OriginalRef != originalRef || f.OriginalVersion != originalVersion ||
				f.OriginalHuella != originalHuella {
				return false
			}
			coincidencias++
		}
		if coincidencias != 1 {
			return false
		}
	}
	return true
}

func (s *ServicioFirmaExterna) leerHistoria(
	ctx context.Context, org, exp, documento string, version uint64, candidato, clave string, pasoOrden int,
) (domain.CircuitoFirma, domain.CircuitoFirmaDocumento, ports.LecturaFirmasR5, error) {
	circuito, err := s.base.circuitoValido(ctx)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, err
	}
	doc, ok := circuito.Documento(documento)
	if !ok {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	lectura := ports.MaterialConsultaFirmasR5{OrganizacionRef: org, ExpedienteRef: exp, VersionExpediente: version,
		Documento: documento, FirmantePrincipalCandidatoRef: candidato, ClaveIdempotencia: clave,
		PasoOrden: pasoOrden, CatalogoHuella: circuito.HuellaCatalogo}
	capacidad, err := s.consulta.AutorizarConsultaFirmasR5(ctx, lectura)
	if err != nil {
		if ctx.Err() != nil {
			return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, ctx.Err()
		}
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadConsultaFirmasR5(capacidad, lectura); err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, err
	}
	firmas, err := s.registro.ConsultarFirmasAutorizadas(ctx, lectura, capacidad)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, err
	}
	if err := validarProyeccionFirmasR5(lectura, firmas); err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, err
	}
	return circuito, doc, firmas, nil
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
	if s.multiple != nil {
		r, err := registrarFirmaMultipleR5(ctx, s.base, s.competencia, s.politica, s.multiple, solicitudMultipleDesdeExterna(sol))
		return ResultadoFirmaExterna{Recibo: r.recibo, MaterialMultiple: r.material, MotivoVerificacion: r.motivo, Custodiado: r.custodiado}, err
	}
	sol.PDFFirmado = bytes.Clone(sol.PDFFirmado)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	previoCircuito, err := s.base.circuitoValido(ctx)
	if err != nil {
		return cero, err
	}
	previoDoc, existe := previoCircuito.Documento(sol.Documento)
	if !existe || sol.PasoOrden > len(previoDoc.Pasos) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	preparacion, err := prepararFirmaR5(ctx, s.base, s.competencia,
		sol.OrganizacionRef, sol.ExpedienteRef, sol.Documento, sol.OriginalRef, sol.OriginalVersion,
		sol.PDFFirmado, previoCircuito, previoDoc.Pasos[sol.PasoOrden-1])
	if err != nil {
		return cero, err
	}
	circuito, doc, lectura, err := s.leerHistoria(ctx, sol.OrganizacionRef, sol.ExpedienteRef,
		sol.Documento, sol.VersionExpediente, preparacion.competencia.FirmantePrincipalRef,
		sol.ClaveIdempotencia, sol.PasoOrden)
	if err != nil {
		return cero, err
	}
	if circuito.CatalogoRef != previoCircuito.CatalogoRef || circuito.HuellaCatalogo != previoCircuito.HuellaCatalogo ||
		sol.PasoOrden > len(doc.Pasos) || doc.Pasos[sol.PasoOrden-1].Referencia != previoDoc.Pasos[sol.PasoOrden-1].Referencia {
		return cero, ErrCircuitoFirmaNoDisponible
	}
	tipo, custodiar := s.base.tiposCustodia[sol.Documento]
	if !custodiar || tipo == "" {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	firmas := lectura.Firmas
	var secuencia int
	var originalEsperado string
	historiaRevision, historiaHuella := lectura.HistoriaRevision, lectura.HistoriaHuella
	previa, repetida := firmaConClave(firmas, sol.Documento, sol.ClaveIdempotencia)
	if repetida {
		if previa.Via != ports.ViaFirmaExternaPortafirmas || previa.PasoOrden != sol.PasoOrden ||
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
		if err := EvaluarCoincidenciaPersonaR5(permite, lectura.CoincideFirmanteEnOtroPaso, !lectura.HistoriaSeparacionAcreditada); err != nil {
			return cero, err
		}
	}
	paso := doc.Pasos[sol.PasoOrden-1]
	original, dictamen, evidencia := preparacion.original, preparacion.dictamen, preparacion.competencia
	if originalEsperado != "" && original.HuellaSHA256 != originalEsperado {
		return cero, ports.ErrCadenaFirmaDocumentoRota
	}
	r := dictamen.Resultado
	m := ports.MaterialFirmaExterna{
		Via:             ports.ViaFirmaExternaPortafirmas,
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

// ComponerFirmaMultiple fija las autoridades V2 antes de atender solicitudes.
func (s *ServicioFirmaExterna) ComponerFirmaMultiple(
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
