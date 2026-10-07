package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type dependenciasFirmaMultipleR5 struct {
	verificador docports.VerificadorFirmasDocumento
	registro    ports.RegistroFirmasVerificadasV2
	autorizador ports.AutorizadorFirmaVerificadaV2
	consulta    ports.AutorizadorConsultaFirmasR5V2
	pdf         ports.FuentePDFFirmaAnterior
	operador    ports.FuentePerfilActivoOperadorFirmaV2
}

func nuevasDependenciasFirmaMultipleR5(v docports.VerificadorFirmasDocumento, r ports.RegistroFirmasVerificadasV2, a ports.AutorizadorFirmaVerificadaV2, c ports.AutorizadorConsultaFirmasR5V2, f ports.FuentePDFFirmaAnterior) (*dependenciasFirmaMultipleR5, error) {
	if nula(v) || nula(r) || nula(a) || nula(c) || nula(f) {
		return nil, ErrCircuitoFirmaNoDisponible
	}
	operador, ok := a.(ports.FuentePerfilActivoOperadorFirmaV2)
	if !ok || nula(operador) {
		return nil, ErrCircuitoFirmaNoDisponible
	}
	return &dependenciasFirmaMultipleR5{v, r, a, c, f, operador}, nil
}

type solicitudFirmaMultipleR5 struct {
	SolicitudFirmaExterna
	via string
}
type resultadoFirmaMultipleR5 struct {
	recibo     ports.ReciboFirmaDocumento
	material   *ports.MaterialFirmaVerificadaV2
	motivo     docports.MotivoVerificacionFirma
	custodiado ports.DocumentoCustodiado
}

func solicitudMultipleDesdeVec(s SolicitudFirmaVec) solicitudFirmaMultipleR5 {
	return solicitudFirmaMultipleR5{SolicitudFirmaExterna: SolicitudFirmaExterna{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente, Documento: s.Documento, PasoOrden: s.PasoOrden, OriginalRef: s.OriginalRef, OriginalVersion: s.OriginalVersion, PDFFirmado: s.PDFFirmado, ClaveIdempotencia: s.ClaveIdempotencia}, via: ports.ViaFirmaCertificadoVEC}
}
func solicitudMultipleDesdeExterna(s SolicitudFirmaExterna) solicitudFirmaMultipleR5 {
	return solicitudFirmaMultipleR5{s, ports.ViaFirmaExternaPortafirmas}
}

// Cada perfil publicado se resuelve por la autoridad nominal. No se infiere un
// cargo de una etiqueta del catálogo ni se suman asignaciones alternativas.
func acreditarCompetenciaPasoMultiple(ctx context.Context, fuente ports.FuenteCompetenciaFirmante, q ports.SolicitudCompetenciaFirmante, paso domain.PasoCircuitoFirma) (ports.EvidenciaCompetenciaFirmante, error) {
	var salida ports.EvidenciaCompetenciaFirmante
	perfiles := append([]string{paso.PerfilRef}, paso.PerfilesAlternativos...)
	vistos := map[string]bool{}
	coincidencias := 0
	for _, perfil := range perfiles {
		if vistos[perfil] || !domain.ReferenciaOpacaValida(perfil) {
			return salida, ports.ErrCompetenciaFirmanteNoAcreditada
		}
		vistos[perfil] = true
		q.PerfilFirmanteRef = perfil
		q.CargoFirmante = ""
		e, err := fuente.AcreditarCompetenciaFirmante(ctx, q)
		if err != nil {
			if ctx.Err() != nil {
				return salida, ctx.Err()
			}
			if errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
				continue
			}
			return salida, ports.ErrCompetenciaFirmanteNoDisponible
		}
		if e.Solicitud != q || !competenciaTemporalValida(e) || e.PerfilFirmanteRef != perfil || !ports.ReferenciaPortafirmasDeclaradaValida(e.CargoFirmante) || !domain.ReferenciaOpacaValida(e.RolIDFirmante) || !domain.ReferenciaOpacaValida(e.CuentaFirmanteRef) || !domain.ReferenciaOpacaValida(e.VinculoCredencialFirmanteRef) || e.VinculoCredencialFirmanteRevision == 0 || !domain.HuellaSHA256FirmaValida(e.VinculoCredencialFirmanteHuella) {
			return salida, ports.ErrCompetenciaFirmanteNoAcreditada
		}
		salida = e
		coincidencias++
	}
	if coincidencias != 1 {
		return ports.EvidenciaCompetenciaFirmante{}, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	return salida, nil
}

func registrarFirmaMultipleR5(ctx context.Context, base *ServicioFirmaDocumento, fuente ports.FuenteCompetenciaFirmante, politica ports.FuentePoliticaMismaPersonaEnPasos, d *dependenciasFirmaMultipleR5, s solicitudFirmaMultipleR5) (resultadoFirmaMultipleR5, error) {
	var cero resultadoFirmaMultipleR5
	s.PDFFirmado = bytes.Clone(s.PDFFirmado)
	circuito, err := base.circuitoValido(ctx)
	if err != nil {
		return cero, err
	}
	doc, ok := circuito.Documento(s.Documento)
	if !ok || s.PasoOrden > len(doc.Pasos) || circuito.CatalogoVersion == 0 {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	paso := doc.Pasos[s.PasoOrden-1]
	original, err := obtenerOriginalFirmaAutorizado(ctx, base.original, ports.SolicitudOriginalFirma{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, OriginalRef: s.OriginalRef, OriginalVersion: s.OriginalVersion})
	if err != nil {
		return cero, err
	}
	q := docports.SolicitudVerificacionFirma{DocumentoID: s.OriginalRef, Version: s.OriginalVersion, FormatoEsperado: "PAdES", HuellaOriginalSHA256: original.HuellaSHA256, ContenidoOriginal: original.Contenido, ContenidoFirmado: s.PDFFirmado}
	dictamen, err := d.verificador.VerificarFirmas(ctx, q)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, DictamenRechazado{Estado: docports.EstadoVerificacionIndeterminada, Motivo: docports.MotivoRespuestaNoInterpretable}
	}
	// El estado global gobierna la decisión. Un detalle individual favorable
	// no permite continuar si el verificador no acredita la cadena completa.
	if dictamen.Estado != docports.EstadoVerificacionValida || dictamen.Motivo != docports.MotivoFirmaVerificada {
		estado, motivo := dictamen.Estado, dictamen.Motivo
		if motivo.EstadoAsociado() == "" || motivo.EstadoAsociado() != estado {
			estado, motivo = docports.EstadoVerificacionIndeterminada, docports.MotivoRespuestaNoInterpretable
		}
		return cero, DictamenRechazado{Estado: estado, Motivo: motivo}
	}
	// Esta selección solo identifica el candidato para la lectura autorizada.
	// La validación íntegra ocurre después de recuperar la historia y custodia.
	if len(dictamen.Firmas) != s.PasoOrden {
		return cero, ErrFirmaNoVerificada
	}
	candidata := dictamen.Firmas[s.PasoOrden-1]
	if candidata.Orden != s.PasoOrden || !domain.HuellaSHA256FirmaValida(candidata.CertificadoHuellaSHA256) || candidata.FirmanteRef != "ref:"+candidata.CertificadoHuellaSHA256 {
		return cero, ErrFirmaNoVerificada
	}
	evidencia, err := acreditarCompetenciaPasoMultiple(ctx, fuente, ports.SolicitudCompetenciaFirmante{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, CatalogoVersion: circuito.CatalogoVersion, CatalogoRef: circuito.CatalogoRef, CatalogoHuella: circuito.HuellaCatalogo, PasoRef: paso.Referencia, PasoOrden: paso.Orden, FirmanteRef: candidata.FirmanteRef, CertificadoHuella: candidata.CertificadoHuellaSHA256}, paso)
	if err != nil {
		return cero, err
	}
	consulta := ports.MaterialConsultaFirmasR5V2{Via: s.via, MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente, Documento: s.Documento, FirmantePrincipalCandidatoRef: evidencia.FirmantePrincipalRef, ClaveIdempotencia: s.ClaveIdempotencia, PasoOrden: s.PasoOrden, CatalogoHuella: circuito.HuellaCatalogo}}
	// En la vía VEC consulta quien firma, con la asignación de la competencia
	// acreditada: la unidad del paso va en UnidadRef (CT186 la liga al plan).
	if s.via == ports.ViaFirmaCertificadoVEC {
		consulta.UnidadRef = evidencia.UnidadFirmanteRef
	}
	if _, err := consulta.Canonico(); err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	capLectura, err := d.consulta.AutorizarConsultaFirmasR5V2(ctx, consulta)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadConsultaFirmasR5V2(capLectura, consulta); err != nil {
		return cero, err
	}
	lectura, err := d.registro.ConsultarFirmasAutorizadasV2(ctx, consulta, capLectura)
	if err != nil {
		return cero, err
	}
	if err := validarHistoriaMultipleR5(consulta.MaterialConsultaFirmasR5, lectura); err != nil {
		return cero, err
	}
	actual, err := base.circuitoValido(ctx)
	if err != nil || actual.CatalogoRef != circuito.CatalogoRef || actual.HuellaCatalogo != circuito.HuellaCatalogo || actual.CatalogoVersion != circuito.CatalogoVersion {
		return cero, ErrCircuitoFirmaNoDisponible
	}
	firmas := enriquecerFirmasRevisionMultiple(lectura)
	var inicioRonda uint64
	previa, repetida := firmaConClave(firmas, s.Documento, s.ClaveIdempotencia)
	revision, historia := lectura.HistoriaRevision, lectura.HistoriaHuella
	secuencia := 0
	firmasPrevias := firmas
	var estado domain.EstadoCircuitoDocumento
	if repetida {
		if previa.Via != s.via || previa.PasoOrden != s.PasoOrden || previa.Resultado != domain.ResultadoFirmaFirmado || previa.CatalogoHuella != circuito.HuellaCatalogo || previa.ExpedienteVersion != s.VersionExpediente {
			return cero, ports.ErrClaveFirmaDocumentoUsada
		}
		secuencia, revision, historia = previa.Secuencia, previa.HistoriaRevision, previa.HistoriaHuella
		firmasPrevias, estado, err = antecedentesReplayMultiple(previa, firmas, lectura.RevisionesPDF)
		if err != nil {
			return cero, err
		}
	} else {
		rondas, err := base.inicioRonda(ctx, s.OrganizacionRef, s.ExpedienteRef)
		if err != nil {
			return cero, err
		}
		inicioRonda = rondas[s.Documento]
		if err := validarOriginalNuevoTrasReparo(firmas, s.Documento, s.PasoOrden, inicioRonda, s.OriginalRef, s.OriginalVersion, original.HuellaSHA256); err != nil {
			return cero, err
		}
		permite, err := ResolverPoliticaMismaPersonaEnPasos(ctx, politica, circuito.CatalogoRef, circuito.HuellaCatalogo)
		if err != nil {
			return cero, err
		}
		if err := EvaluarCoincidenciaPersonaR5(permite, lectura.CoincideFirmanteEnOtroPaso, !lectura.HistoriaSeparacionAcreditada); err != nil {
			return cero, err
		}
	}
	if !repetida {
		estado, err = domain.CalcularEstadoCircuitoFirmaEnRonda(doc, circuito.HuellaCatalogo, eventosDocumento(firmasPrevias, s.Documento), inicioRonda)
		if err != nil {
			return cero, err
		}
	}
	if estado.Completo || estado.PasoPendiente != s.PasoOrden {
		return cero, ErrPasoFirmaNoPendiente
	}
	if !repetida {
		secuencia = estado.UltimaSecuencia + 1
	}
	if estado.OriginalEsperadoHuella != "" && estado.OriginalEsperadoHuella != original.HuellaSHA256 {
		return cero, ports.ErrCadenaFirmaDocumentoRota
	}
	anteriores, err := recuperarAntecedentesMultiple(ctx, d.pdf, s, estado, firmasPrevias)
	if err != nil {
		return cero, err
	}
	if err := validarRevisionesAnterioresMultiple(dictamen, anteriores, lectura.RevisionesPDF); err != nil {
		return cero, err
	}
	firma, err := ValidarFirmaMultipleParaPaso(q, dictamen, s.Documento, s.PasoOrden, anteriores)
	if err != nil {
		return cero, err
	}
	perfilOperador, err := d.operador.ObtenerPerfilActivoOperadorFirmaV2(ctx)
	if err != nil || !domain.ReferenciaOpacaValida(perfilOperador) {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	m, err := materialFirmaMultipleR5(s, circuito, paso, secuencia, revision, historia, original, evidencia, dictamen, firma, anteriores, perfilOperador)
	if err != nil {
		return cero, err
	}
	capacidad, err := d.autorizador.AutorizarFirmaVerificadaV2(ctx, m)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadFirmaVerificadaV2(capacidad, m); err != nil {
		return cero, err
	}
	tipo, ok := base.tiposCustodia[s.Documento]
	if !ok || tipo == "" {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	custodiado, err := custodiarFirmaVerificada(base, ctx, s.PDFFirmado, m.MaterialFirmaExterna, tipo)
	if err != nil {
		return cero, err
	}
	recibo, err := d.registro.RegistrarFirmaVerificadaV2(ctx, m, capacidad)
	if err != nil {
		return cero, err
	}
	h, _ := m.HuellaSHA256()
	if recibo.SolicitudHuella != h || recibo.Resultado != domain.ResultadoFirmaFirmado || !domain.ReferenciaOpacaValida(recibo.FirmaRef) || !domain.ReferenciaOpacaValida(recibo.ReciboRef) || !domain.ReferenciaOpacaValida(recibo.ActorRef) || !domain.ReferenciaOpacaValida(recibo.PerfilRef) || recibo.RegistradaEn.IsZero() || recibo.DocumentoCustodiaRef != m.DocumentoCustodiaRef || recibo.DocumentoCustodiaVersion != m.DocumentoCustodiaVersion || recibo.Secuencia != m.Secuencia || recibo.ExpedienteVersion != m.VersionExpediente || (s.via == ports.ViaFirmaCertificadoVEC && recibo.ActorRef != m.FirmantePrincipalRef) || (s.via == ports.ViaFirmaExternaPortafirmas && recibo.ActorRef == m.FirmantePrincipalRef) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	if repetida && (!recibo.YaRegistrada || recibo.FirmaRef != previa.FirmaRef ||
		recibo.ReciboRef != previa.ReciboRef || !recibo.RegistradaEn.Equal(previa.RegistradaEn)) {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return resultadoFirmaMultipleR5{recibo, &m, dictamen.Motivo, custodiado}, nil
}

func validarHistoriaMultipleR5(q ports.MaterialConsultaFirmasR5, l ports.LecturaFirmasR5V2) error {
	// V2 concede la evidencia técnica por una acción separada; la proyección
	// pública R5 V1 continúa sin certificados ni identidades.
	copia := l.LecturaFirmasR5
	copia.Firmas = append([]ports.FirmaRegistrada(nil), l.Firmas...)
	for i := range copia.Firmas {
		copia.Firmas[i].CertificadoHuella = ""
		copia.Firmas[i].FirmanteRef = ""
	}
	if err := validarProyeccionFirmasR5(q, copia); err != nil {
		return err
	}
	vistosFirmas, vistosRecibos, vistasClaves := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, f := range l.Firmas {
		if vistosFirmas[f.FirmaRef] || vistosRecibos[f.ReciboRef] || vistasClaves[f.ClaveIdempotencia] {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistosFirmas[f.FirmaRef], vistosRecibos[f.ReciboRef], vistasClaves[f.ClaveIdempotencia] = true, true, true
	}
	vistos := map[string]bool{}
	for _, r := range l.RevisionesPDF {
		if vistos[r.FirmaRef] || !domain.HuellaSHA256FirmaValida(r.CertificadoHuella) || r.FirmanteRef != "ref:"+r.CertificadoHuella || r.RevisionHuellaSHA256 != r.FirmadoHuella || r.OrdenFirmaPDF != r.PasoOrden {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistos[r.FirmaRef] = true
		encontrado := false
		for _, f := range l.Firmas {
			if f.FirmaRef != r.FirmaRef {
				continue
			}
			proyectada, tecnicamenteAcreditada := f, r.FirmaRegistrada
			if (f.FirmanteRef != "" && f.FirmanteRef != r.FirmanteRef) || (f.CertificadoHuella != "" && f.CertificadoHuella != r.CertificadoHuella) {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
			proyectada.FirmanteRef, proyectada.CertificadoHuella = "", ""
			tecnicamenteAcreditada.FirmanteRef, tecnicamenteAcreditada.CertificadoHuella = "", ""
			if proyectada != tecnicamenteAcreditada {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
			encontrado = true
		}
		if !encontrado {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	return nil
}

func recuperarAntecedentesMultiple(ctx context.Context, fuente ports.FuentePDFFirmaAnterior, s solicitudFirmaMultipleR5, estado domain.EstadoCircuitoDocumento, firmas []ports.FirmaRegistrada) ([]AntecedenteFirmaMultiple, error) {
	anteriores := make([]AntecedenteFirmaMultiple, 0, s.PasoOrden-1)
	for orden := 1; orden < s.PasoOrden; orden++ {
		recibo := ""
		for _, p := range estado.Pasos {
			if p.Orden == orden && p.Estado == domain.EstadoPasoFirmado {
				recibo = p.ReciboRef
			}
		}
		var elegida ports.FirmaRegistrada
		coincidencias := 0
		for _, f := range firmas {
			if f.Documento == s.Documento && f.PasoOrden == orden && f.ReciboRef == recibo {
				elegida = f
				coincidencias++
			}
		}
		if recibo == "" || coincidencias != 1 {
			return nil, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		q := ports.SolicitudPDFFirmaAnterior{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, FirmaRef: elegida.FirmaRef, ReciboRef: elegida.ReciboRef, DocumentoRef: elegida.DocumentoCustodiaRef, DocumentoVersion: elegida.DocumentoCustodiaVersion, DocumentoHuella: elegida.FirmadoHuella}
		if q.Validar() != nil {
			return nil, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		pdf, err := fuente.ObtenerPDFFirmaAnterior(ctx, q)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		if pdf.Solicitud != q || len(pdf.Contenido) == 0 || len(pdf.Contenido) > ports.MaximoDocumentoFirmaBytes || huella(pdf.Contenido) != q.DocumentoHuella {
			return nil, ports.ErrCadenaFirmaDocumentoRota
		}
		anteriores = append(anteriores, AntecedenteFirmaMultiple{Firma: elegida, PDFFirmado: bytes.Clone(pdf.Contenido)})
	}
	return anteriores, nil
}

func materialFirmaMultipleR5(s solicitudFirmaMultipleR5, c domain.CircuitoFirma, p domain.PasoCircuitoFirma, secuencia int, revision uint64, historia string, o ports.OriginalFirmaAutorizado, e ports.EvidenciaCompetenciaFirmante, v docports.VerificacionFirmasDocumento, f docports.FirmaPDFVerificada, anteriores []AntecedenteFirmaMultiple, perfilOperador string) (ports.MaterialFirmaVerificadaV2, error) {
	q := docports.SolicitudVerificacionFirma{DocumentoID: s.OriginalRef, Version: s.OriginalVersion, FormatoEsperado: "PAdES", HuellaOriginalSHA256: o.HuellaSHA256, ContenidoOriginal: o.Contenido, ContenidoFirmado: s.PDFFirmado}
	canon, err := CanonicoEvidenciaFirmasMultiple(q, v, s.Documento, s.PasoOrden, anteriores)
	if err != nil {
		return ports.MaterialFirmaVerificadaV2{}, err
	}
	m := ports.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ports.MaterialFirmaExterna{
		Via: s.via, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente,
		Documento: s.Documento, CatalogoRef: c.CatalogoRef, CatalogoHuella: c.HuellaCatalogo, PasoRef: p.Referencia, PasoOrden: p.Orden, Secuencia: secuencia, HistoriaRevision: revision, HistoriaHuella: historia,
		OriginalRef: s.OriginalRef, OriginalVersion: s.OriginalVersion, OriginalHuella: o.HuellaSHA256, FirmadoHuella: v.HuellaFirmadoSHA256, CertificadoHuella: f.CertificadoHuellaSHA256, FirmanteRef: f.FirmanteRef,
		FirmantePrincipalRef: e.FirmantePrincipalRef, PerfilFirmanteRef: e.PerfilFirmanteRef, CargoFirmante: e.CargoFirmante, UnidadFirmanteRef: e.UnidadFirmanteRef, PerfilActivoFirmanteRef: e.PerfilActivoFirmanteRef,
		PuestoFirmanteRef: e.PuestoFirmanteRef, AmbitoFirmanteRef: e.AmbitoFirmanteRef, AsignacionFirmanteRef: e.AsignacionFirmanteRef, AsignacionFirmanteVersion: e.AsignacionFirmanteVersion, AsignacionFirmanteHuella: e.AsignacionFirmanteHuella,
		VersionRolFirmanteRef: e.VersionRolFirmanteRef, VersionRolFirmanteHuella: e.VersionRolFirmanteHuella, ControlVigenciaFirmanteRef: e.ControlVigenciaFirmanteRef, ControlVigenciaFirmanteRevision: e.ControlVigenciaFirmanteRevision, ControlVigenciaFirmanteHuella: e.ControlVigenciaFirmanteHuella,
		AsignacionVigenteDesde: e.AsignacionVigenteDesde, AsignacionVigenteHasta: e.AsignacionVigenteHasta, ActoCompetenciaRef: e.ActoCompetenciaRef, DelegacionRef: e.DelegacionRef,
		PoliticaVerificacion: PoliticaVerificacionFirmaMultipleV2, RevocacionEstado: f.RevocacionEstado, SelloTiempoEstado: f.SelloTiempoEstado,
		ReferenciaPortafirmasDeclarada: s.ReferenciaPortafirmasDeclarada, FechaPortafirmasDeclarada: s.FechaPortafirmasDeclarada, ClaveIdempotencia: s.ClaveIdempotencia,
		DocumentoCustodiaRef: ports.DocumentoCustodiaRef(s.OrganizacionRef, s.ExpedienteRef, s.ClaveIdempotencia), DocumentoCustodiaVersion: ports.VersionDocumentoCustodiado,
	}, PerfilActivoOperadorRef: perfilOperador, CatalogoVersion: c.CatalogoVersion, RolIDFirmante: e.RolIDFirmante, CuentaFirmanteRef: e.CuentaFirmanteRef, VinculoCredencialFirmanteRef: e.VinculoCredencialFirmanteRef, VinculoCredencialFirmanteRevision: e.VinculoCredencialFirmanteRevision, VinculoCredencialFirmanteHuella: e.VinculoCredencialFirmanteHuella, EntradaDocumentoRef: s.OriginalRef, EntradaDocumentoVersion: s.OriginalVersion, EntradaDocumentoLongitud: uint64(len(o.Contenido)), EntradaDocumentoHuella: o.HuellaSHA256,
		OrdenFirmaPDF: f.Orden, ByteRange: f.ByteRange, RevisionHuellaSHA256: f.RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256: f.ContenidoFirmadoHuellaSHA256, RevisionLongitud: f.RevisionLongitud, EvidenciaFirmasCanonica: json.RawMessage(canon), EvidenciaFirmasHuellaSHA256: huella(canon), ComprobadaEn: v.ComprobadoEn}
	if len(anteriores) > 0 {
		a := anteriores[len(anteriores)-1]
		m.FirmaAnteriorRef = a.Firma.FirmaRef
		m.ReciboAnteriorRef = a.Firma.ReciboRef
		m.EntradaDocumentoRef = a.Firma.DocumentoCustodiaRef
		m.EntradaDocumentoVersion = a.Firma.DocumentoCustodiaVersion
		m.EntradaDocumentoLongitud = uint64(len(a.PDFFirmado))
		m.EntradaDocumentoHuella = a.Firma.FirmadoHuella
	}
	if err := m.Validar(); err != nil {
		return ports.MaterialFirmaVerificadaV2{}, err
	}
	return m, nil
}

func enriquecerFirmasRevisionMultiple(l ports.LecturaFirmasR5V2) []ports.FirmaRegistrada {
	firmas := append([]ports.FirmaRegistrada(nil), l.Firmas...)
	for i := range firmas {
		for _, r := range l.RevisionesPDF {
			if firmas[i].FirmaRef == r.FirmaRef {
				firmas[i].FirmanteRef = r.FirmanteRef
				firmas[i].CertificadoHuella = r.CertificadoHuella
			}
		}
	}
	return firmas
}

func validarRevisionesAnterioresMultiple(v docports.VerificacionFirmasDocumento, anteriores []AntecedenteFirmaMultiple, revisiones []ports.FirmaRegistradaRevisionPDFV2) error {
	for i, a := range anteriores {
		if i >= len(v.Firmas) {
			return ports.ErrCadenaFirmaDocumentoRota
		}
		f := v.Firmas[i]
		matches := 0
		for _, r := range revisiones {
			if r.FirmaRef != a.Firma.FirmaRef {
				continue
			}
			matches++
			if r.OrdenFirmaPDF != f.Orden || r.ByteRange != f.ByteRange || r.RevisionLongitud != f.RevisionLongitud || r.RevisionHuellaSHA256 != f.RevisionHuellaSHA256 || r.ContenidoFirmadoHuellaSHA256 != f.ContenidoFirmadoHuellaSHA256 || r.CertificadoHuella != f.CertificadoHuellaSHA256 || r.FirmanteRef != f.FirmanteRef {
				return ports.ErrCadenaFirmaDocumentoRota
			}
		}
		if matches != 1 {
			return ports.ErrAntecedenteFirmaR5NoAcreditado
		}
	}
	return nil
}

// Un replay sigue la cadena exacta guardada por el acto. Una ronda posterior
// no puede reinterpretar sus antecedentes ni sustituirlos por firmas nuevas.
func antecedentesReplayMultiple(actual ports.FirmaRegistrada, firmas []ports.FirmaRegistrada, revisiones []ports.FirmaRegistradaRevisionPDFV2) ([]ports.FirmaRegistrada, domain.EstadoCircuitoDocumento, error) {
	estado := domain.EstadoCircuitoDocumento{Documento: actual.Documento, PasoPendiente: actual.PasoOrden, UltimaSecuencia: actual.Secuencia - 1}
	cadena := make([]ports.FirmaRegistrada, actual.PasoOrden-1)
	siguiente := actual
	for orden := actual.PasoOrden; orden >= 1; orden-- {
		var revision ports.FirmaRegistradaRevisionPDFV2
		coincidencias := 0
		for _, r := range revisiones {
			if r.FirmaRef == siguiente.FirmaRef {
				revision = r
				coincidencias++
			}
		}
		if coincidencias != 1 || revision.ReciboRef != siguiente.ReciboRef || revision.OrdenFirmaPDF != orden {
			return nil, estado, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		if orden == 1 {
			if revision.FirmaAnteriorRef != "" || revision.ReciboAnteriorRef != "" {
				return nil, estado, ports.ErrCadenaFirmaDocumentoRota
			}
			break
		}
		coincidencias = 0
		var anterior ports.FirmaRegistrada
		for _, f := range firmas {
			if f.FirmaRef == revision.FirmaAnteriorRef && f.ReciboRef == revision.ReciboAnteriorRef {
				anterior = f
				coincidencias++
			}
		}
		if coincidencias != 1 || anterior.Documento != actual.Documento || anterior.PasoOrden != orden-1 || anterior.Secuencia >= siguiente.Secuencia || anterior.Resultado != domain.ResultadoFirmaFirmado || anterior.CatalogoHuella != actual.CatalogoHuella || anterior.OriginalRef != actual.OriginalRef || anterior.OriginalVersion != actual.OriginalVersion || anterior.OriginalHuella != actual.OriginalHuella {
			return nil, estado, ports.ErrCadenaFirmaDocumentoRota
		}
		cadena[orden-2] = anterior
		siguiente = anterior
	}
	for _, f := range cadena {
		estado.Pasos = append(estado.Pasos, domain.EstadoPasoCalculado{Orden: f.PasoOrden, Estado: domain.EstadoPasoFirmado, ReciboRef: f.ReciboRef, RegistradaEn: f.RegistradaEn})
	}
	if len(cadena) > 0 {
		estado.OriginalEsperadoHuella = cadena[len(cadena)-1].OriginalHuella
	}
	return cadena, estado, nil
}
