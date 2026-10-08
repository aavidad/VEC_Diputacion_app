package application

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Reutiliza la lectura R5 y la ronda del servicio existente. No registra
// firmas, reserva documentos ni convierte disponibilidad en autorización.
type ServicioPreflightFirmaR5 struct {
	firmas             *ServicioFirmaExterna
	contextos          ports.ResolutorContextoAutorizacionAltaV3
	disponibilidad     ports.VerificadorDisponibilidadFirmaR5
	preparacionExterna ports.ComprobadorPreparacionExternaR5
}

func NuevoServicioPreflightFirmaR5(firmas *ServicioFirmaExterna, contextos ports.ResolutorContextoAutorizacionAltaV3, disponibilidad ports.VerificadorDisponibilidadFirmaR5) (*ServicioPreflightFirmaR5, error) {
	if firmas == nil || firmas.base == nil || (firmas.multiple == nil && (dependenciaNula(firmas.consulta) || dependenciaNula(firmas.registro))) || dependenciaNula(contextos) {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	if dependenciaNula(disponibilidad) {
		disponibilidad = nil
	}
	return &ServicioPreflightFirmaR5{firmas: firmas, contextos: contextos, disponibilidad: disponibilidad}, nil
}

// NuevoServicioPreflightFirmaR5V2Externa ofrece exclusivamente preparar la
// vía externa. Su comprobante no afirma salud remota, firma válida ni
// competencia de una persona que todavía no ha aportado su PDF.
func NuevoServicioPreflightFirmaR5V2Externa(firmas *ServicioFirmaExterna,
	contextos ports.ResolutorContextoAutorizacionAltaV3, fuente ports.ComprobadorPreparacionExternaR5,
) (*ServicioPreflightFirmaR5, error) {
	if dependenciaNula(fuente) {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	s, err := NuevoServicioPreflightFirmaR5(firmas, contextos, nil)
	if err != nil || s.firmas.multiple == nil {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	s.preparacionExterna = fuente
	return s, nil
}

func ValidarSolicitudPreflightFirmaR5(q ports.SolicitudPreflightFirmaR5) error {
	if q.Canal.Validar() != nil || q.Canal.VersionObservada > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(q.Documento) || !domain.ReferenciaOpacaValida(q.OriginalRef) ||
		q.OriginalVersion == 0 || q.OriginalVersion > 9007199254740991 {
		return ports.ErrPreflightFirmaR5Invalido
	}
	return nil
}

// ValidarResultadoPreflightFirmaR5 protege también la frontera HTTP de un
// consultor que entregue una proyección incompleta, ajena o repetida.
func ValidarResultadoPreflightFirmaR5(r ports.ResultadoPreflightFirmaR5, q ports.SolicitudPreflightFirmaR5) error {
	if ValidarSolicitudPreflightFirmaR5(q) != nil || r.VersionExpediente != q.Canal.VersionObservada ||
		r.Documento != q.Documento || r.OriginalRef != q.OriginalRef || r.OriginalVersion != q.OriginalVersion ||
		!domain.ReferenciaOpacaValida(r.CatalogoRef) || !domain.HuellaSHA256FirmaValida(r.CatalogoHuella) ||
		r.PasoPendiente < 0 || r.PasoPendiente > domain.MaximoPasosCircuitoFirma || r.ViasDisponibles == nil ||
		len(r.ViasDisponibles) > 2 || (r.PasoPendiente == 0 && len(r.ViasDisponibles) != 0) {
		return ports.ErrPreflightFirmaR5NoConfiable
	}
	vistos := make(map[string]bool)
	for _, via := range r.ViasDisponibles {
		if (via != ports.ViaFirmaCertificadoVEC && via != ports.ViaFirmaExternaPortafirmas) || vistos[via] {
			return ports.ErrPreflightFirmaR5NoConfiable
		}
		vistos[via] = true
	}
	return nil
}

func (s *ServicioPreflightFirmaR5) Consultar(ctx context.Context, q ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5, error) {
	r, err := s.consultar(ctx, q, false)
	return r.ResultadoPreflightFirmaR5, err
}

// ConsultarV2 exige la lectura acumulada y devuelve el PDF exacto de entrada.
// Un estado antiguo de la interfaz nunca selecciona aquí el paso de firma.
func (s *ServicioPreflightFirmaR5) ConsultarV2(ctx context.Context, q ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5V2, error) {
	if s == nil || s.firmas == nil || s.firmas.multiple == nil {
		return ports.ResultadoPreflightFirmaR5V2{}, ports.ErrPreflightFirmaR5NoDisponible
	}
	return s.consultar(ctx, q, true)
}

func (s *ServicioPreflightFirmaR5) consultar(ctx context.Context, q ports.SolicitudPreflightFirmaR5, conEntrada bool) (ports.ResultadoPreflightFirmaR5V2, error) {
	var cero ports.ResultadoPreflightFirmaR5V2
	if s == nil || ctx == nil || s.firmas == nil || s.firmas.base == nil || dependenciaNula(s.contextos) {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	if err := ValidarSolicitudPreflightFirmaR5(q); err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	canal := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: q.Canal.AutenticacionRef, SesionRef: q.Canal.SesionRef, PerfilRef: q.Canal.PerfilRef,
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(ctx, canal)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil || contexto.ValidarPara(canal, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	actor := contexto.Resultado.Contexto.Principal.ID
	if !strings.HasPrefix(actor, "per_") || !domain.ReferenciaOpacaValida(actor) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	// Una copia por consulta conserva el servicio compartido y obliga a que
	// cada decisión R5 pertenezca al contexto que acaba de resolver VEC.
	firmas := *s.firmas
	if firmas.multiple == nil {
		firmas.consulta = consultaPreflightR5ContextoActual{
			origen: s.firmas.consulta, contextoRef: contexto.Resultado.RegistroContextoRef,
			contextoHuella: contexto.Resultado.HuellaSHA256,
		}
	} else {
		multiple := *firmas.multiple
		multiple.consulta = consultaPreflightR5V2ContextoActual{
			origen: multiple.consulta, contextoRef: contexto.Resultado.RegistroContextoRef,
			contextoHuella: contexto.Resultado.HuellaSHA256,
		}
		firmas.multiple = &multiple
	}
	nominal := *s
	nominal.firmas = &firmas
	r, err := nominal.consultarNominalConEntrada(ctx, q, actor, contexto.Resultado.HuellaSHA256, conEntrada)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, err
	}
	if contexto.ValidarPara(canal, time.Now().UTC().Truncate(time.Microsecond)) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return r, nil
}

func (s *ServicioPreflightFirmaR5) consultarNominal(ctx context.Context, q ports.SolicitudPreflightFirmaR5, actor, contextoHuella string) (ports.ResultadoPreflightFirmaR5, error) {
	r, err := s.consultarNominalConEntrada(ctx, q, actor, contextoHuella, false)
	return r.ResultadoPreflightFirmaR5, err
}

func (s *ServicioPreflightFirmaR5) consultarNominalConEntrada(ctx context.Context, q ports.SolicitudPreflightFirmaR5, actor, contextoHuella string, conEntrada bool) (ports.ResultadoPreflightFirmaR5V2, error) {
	var cero ports.ResultadoPreflightFirmaR5V2
	if conEntrada && s.firmas.multiple == nil {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	// La clave es efímera, aleatoria y de servidor: esta consulta no recupera
	// una firma anterior elegida por el navegador ni crea una operación.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	catalogo, doc, lectura, err := s.firmas.leerHistoriaPreflightFirmaR5(ctx, q.Canal.OrganizacionRef,
		q.Canal.ExpedienteRef, q.Documento, q.Canal.VersionObservada, actor, hex.EncodeToString(nonce[:]), 1)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, err
	}
	rondas, err := s.firmas.base.inicioRonda(ctx, q.Canal.OrganizacionRef, q.Canal.ExpedienteRef)
	if err != nil {
		return cero, err
	}
	estado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(doc, catalogo.HuellaCatalogo,
		eventosDocumento(lectura.Firmas, q.Documento), rondas[q.Documento])
	if err != nil {
		return cero, ports.ErrPreflightFirmaR5NoConfiable
	}
	// La concesión del paso inicial no autoriza consultar la preparación de
	// otro paso. Releer con el pendiente exacto exige una segunda decisión.
	if estado.PasoPendiente > 1 {
		actualCatalogo, actualDoc, actualLectura, err := s.firmas.leerHistoriaPreflightFirmaR5(ctx, q.Canal.OrganizacionRef,
			q.Canal.ExpedienteRef, q.Documento, q.Canal.VersionObservada, actor, hex.EncodeToString(nonce[:]), estado.PasoPendiente)
		if err != nil {
			return cero, err
		}
		if actualCatalogo.CatalogoRef != catalogo.CatalogoRef || actualCatalogo.HuellaCatalogo != catalogo.HuellaCatalogo ||
			actualLectura.HistoriaRevision != lectura.HistoriaRevision || actualLectura.HistoriaHuella != lectura.HistoriaHuella {
			return cero, ports.ErrPreflightFirmaR5NoConfiable
		}
		actualEstado, err := domain.CalcularEstadoCircuitoFirmaEnRonda(actualDoc, catalogo.HuellaCatalogo,
			eventosDocumento(actualLectura.Firmas, q.Documento), rondas[q.Documento])
		if err != nil || actualEstado.PasoPendiente != estado.PasoPendiente {
			return cero, ports.ErrPreflightFirmaR5NoConfiable
		}
		doc, lectura, estado = actualDoc, actualLectura, actualEstado
	}
	r := ports.ResultadoPreflightFirmaR5V2{ResultadoPreflightFirmaR5: ports.ResultadoPreflightFirmaR5{
		VersionExpediente: q.Canal.VersionObservada, Documento: q.Documento,
		CatalogoRef: catalogo.CatalogoRef, CatalogoHuella: catalogo.HuellaCatalogo,
		PasoPendiente: estado.PasoPendiente, OriginalRef: q.OriginalRef, OriginalVersion: q.OriginalVersion,
		ViasDisponibles: []string{},
	}}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if estado.Completo || (!conEntrada && s.disponibilidad == nil) {
		return r, nil
	}
	original, err := obtenerOriginalFirmaAutorizado(ctx, s.firmas.base.original, ports.SolicitudOriginalFirma{
		OrganizacionRef: q.Canal.OrganizacionRef, ExpedienteRef: q.Canal.ExpedienteRef,
		Documento: q.Documento, OriginalRef: q.OriginalRef, OriginalVersion: q.OriginalVersion,
	})
	if err != nil {
		return cero, err
	}
	if estado.OriginalEsperadoHuella != "" && estado.OriginalEsperadoHuella != original.HuellaSHA256 {
		return cero, ports.ErrCadenaFirmaDocumentoRota
	}
	if err := validarOriginalNuevoTrasReparo(lectura.Firmas, q.Documento, estado.PasoPendiente,
		rondas[q.Documento], q.OriginalRef, q.OriginalVersion, original.HuellaSHA256); err != nil {
		return cero, err
	}
	if !antecedentesR5Acreditados(estado, lectura.Firmas, q.Documento, estado.PasoPendiente,
		q.OriginalRef, q.OriginalVersion, original.HuellaSHA256) {
		if conEntrada {
			return cero, ports.ErrAntecedenteFirmaR5NoAcreditado
		}
		return r, nil
	}
	if conEntrada && estado.PasoPendiente == 1 {
		r.EntradaDocumentoRef, r.EntradaDocumentoVersion, r.EntradaDocumentoHuella = original.Solicitud.OriginalRef, original.Solicitud.OriginalVersion, original.HuellaSHA256
	}
	if s.firmas.multiple != nil && estado.PasoPendiente > 1 {
		// La disponibilidad del segundo paso exige custodia V2 recuperable.
		// Una firma legada o un PDF distinto no preparan una revisión nueva.
		sol := solicitudFirmaMultipleR5{SolicitudFirmaExterna: SolicitudFirmaExterna{
			OrganizacionRef: q.Canal.OrganizacionRef, ExpedienteRef: q.Canal.ExpedienteRef,
			Documento: q.Documento, PasoOrden: estado.PasoPendiente,
		}}
		anteriores, err := recuperarAntecedentesMultiple(ctx, s.firmas.multiple.pdf, sol, estado, lectura.Firmas)
		if err != nil {
			return cero, err
		}
		for _, a := range anteriores {
			if a.Firma.FirmanteRef == "" || a.Firma.CertificadoHuella == "" {
				if conEntrada {
					return cero, ports.ErrAntecedenteFirmaR5NoAcreditado
				}
				return r, nil
			}
			if a.Firma.OriginalRef != q.OriginalRef || a.Firma.OriginalVersion != q.OriginalVersion ||
				a.Firma.OriginalHuella != original.HuellaSHA256 || !bytes.HasPrefix(a.PDFFirmado, original.Contenido) {
				return cero, ports.ErrCadenaFirmaDocumentoRota
			}
		}
		if conEntrada {
			if len(anteriores) != estado.PasoPendiente-1 {
				return cero, ports.ErrAntecedenteFirmaR5NoAcreditado
			}
			entrada := anteriores[len(anteriores)-1].Firma
			r.EntradaDocumentoRef, r.EntradaDocumentoVersion, r.EntradaDocumentoHuella = entrada.DocumentoCustodiaRef, entrada.DocumentoCustodiaVersion, entrada.FirmadoHuella
		}
	}
	if conEntrada && ValidarResultadoPreflightFirmaR5V2(r, q) != nil {
		return cero, ports.ErrPreflightFirmaR5NoConfiable
	}
	if conEntrada && s.preparacionExterna != nil {
		paso := doc.Pasos[estado.PasoPendiente-1]
		solicitud := ports.SolicitudDisponibilidadFirmaR5{
			Preflight: q, ActorRef: actor, PerfilRef: q.Canal.PerfilRef, ContextoHuella: contextoHuella,
			CatalogoRef: catalogo.CatalogoRef, CatalogoHuella: catalogo.HuellaCatalogo,
			PasoRef: paso.Referencia, PasoOrden: paso.Orden, HistoriaRevision: lectura.HistoriaRevision,
			HistoriaHuella: lectura.HistoriaHuella, OriginalHuella: original.HuellaSHA256,
		}
		comprobada, err := s.preparacionExterna.ComprobarPreparacionExternaR5(ctx, solicitud)
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errCabeza := s.validarCabezaPreflightConEntrada(ctx, q, actor, hex.EncodeToString(nonce[:]),
			paso.Orden, catalogo, lectura); errCabeza != nil {
			return cero, errCabeza
		}
		if errors.Is(err, ports.ErrPreparacionExternaR5NoAcreditada) {
			return r, nil
		}
		if err != nil {
			if errors.Is(err, ports.ErrFirmaDocumentoDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) {
				return cero, ports.ErrFirmaDocumentoDenegada
			}
			return cero, ports.ErrPreflightFirmaR5NoDisponible
		}
		if !preparacionExternaR5Valida(comprobada, solicitud, time.Now().UTC().Truncate(time.Microsecond)) {
			return cero, ports.ErrPreflightFirmaR5NoConfiable
		}
		r.ViasDisponibles = append(r.ViasDisponibles, ports.ViaFirmaExternaPortafirmas)
		if ValidarResultadoPreflightFirmaR5V2(r, q) != nil {
			return cero, ports.ErrPreflightFirmaR5NoConfiable
		}
		return r, nil
	}
	if s.disponibilidad == nil {
		if err := s.validarCabezaPreflightConEntrada(ctx, q, actor, hex.EncodeToString(nonce[:]), estado.PasoPendiente, catalogo, lectura); err != nil {
			return cero, err
		}
		return r, nil
	}
	permite, err := ResolverPoliticaMismaPersonaEnPasos(ctx, s.firmas.politica, catalogo.CatalogoRef, catalogo.HuellaCatalogo)
	if err != nil {
		return cero, err
	}
	// La coincidencia del actor actual restringe la vía VEC. El firmante de
	// un PDF externo solo se conocerá en su verificación; el registrador
	// actual no se convierte en ese firmante por consultar el preflight.
	coincidePermitido := EvaluarCoincidenciaPersonaR5(permite, lectura.CoincideFirmanteEnOtroPaso, !lectura.HistoriaSeparacionAcreditada) == nil
	paso := doc.Pasos[estado.PasoPendiente-1]
	solicitud := ports.SolicitudDisponibilidadFirmaR5{
		Preflight: q, ActorRef: actor, PerfilRef: q.Canal.PerfilRef, ContextoHuella: contextoHuella,
		CatalogoRef: catalogo.CatalogoRef, CatalogoHuella: catalogo.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, HistoriaRevision: lectura.HistoriaRevision,
		HistoriaHuella: lectura.HistoriaHuella, OriginalHuella: original.HuellaSHA256,
	}
	vias, err := s.disponibilidad.VerificarDisponibilidadFirmaR5(ctx, solicitud)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ports.ErrFirmaDocumentoDenegada) || errors.Is(err, ports.ErrAutorizacionDenegada) {
			return cero, ports.ErrFirmaDocumentoDenegada
		}
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	actual, _, cabeza, err := s.firmas.leerHistoriaPreflightFirmaR5(ctx, q.Canal.OrganizacionRef,
		q.Canal.ExpedienteRef, q.Documento, q.Canal.VersionObservada, actor, hex.EncodeToString(nonce[:]), paso.Orden)
	if err != nil || actual.CatalogoRef != catalogo.CatalogoRef || actual.HuellaCatalogo != catalogo.HuellaCatalogo ||
		cabeza.HistoriaRevision != lectura.HistoriaRevision || cabeza.HistoriaHuella != lectura.HistoriaHuella {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	if len(vias) > 2 {
		return cero, ports.ErrPreflightFirmaR5NoConfiable
	}
	for _, via := range vias {
		if !disponibilidadFirmaR5Valida(via, solicitud, time.Now().UTC().Truncate(time.Microsecond)) {
			return cero, ports.ErrPreflightFirmaR5NoConfiable
		}
		if via.Via == ports.ViaFirmaCertificadoVEC && !coincidePermitido {
			continue
		}
		r.ViasDisponibles = append(r.ViasDisponibles, via.Via)
	}
	if err := ValidarResultadoPreflightFirmaR5(r.ResultadoPreflightFirmaR5, q); err != nil {
		return cero, err
	}
	return r, nil
}

func disponibilidadFirmaR5Valida(d ports.DisponibilidadFirmaR5Verificada, q ports.SolicitudDisponibilidadFirmaR5, ahora time.Time) bool {
	if d.Solicitud != q || d.Original.Referencia != q.Preflight.OriginalRef ||
		d.Original.HuellaSHA256 != q.OriginalHuella || (d.Via != ports.ViaFirmaCertificadoVEC && d.Via != ports.ViaFirmaExternaPortafirmas) ||
		!domain.InstanteUTCCanonico(d.ComprobadaEn) || !domain.InstanteUTCCanonico(d.ValidaHasta) ||
		d.ComprobadaEn.After(ahora) || !ahora.Before(d.ValidaHasta) {
		return false
	}
	for _, c := range []ports.ComprobanteComponenteFirmaR5{d.Original, d.Verificador, d.Competencia, d.Perfil, d.Custodia, d.Registro} {
		if !domain.ReferenciaOpacaValida(c.Referencia) || !domain.HuellaSHA256FirmaValida(c.HuellaSHA256) {
			return false
		}
	}
	return true
}

// La validez del resumen externo no excede la de ninguna fuente consultada ni
// una cota técnica breve. El constructor del proveedor acredita la procedencia
// de cada huella; aquí se rechazan resultados incompletos, cruzados o futuros.
func preparacionExternaR5Valida(p ports.PreparacionExternaR5Comprobada,
	q ports.SolicitudDisponibilidadFirmaR5, ahora time.Time,
) bool {
	if p.Solicitud != q || p.Original.Referencia != q.Preflight.OriginalRef ||
		p.Original.Version != q.Preflight.OriginalVersion || p.Original.HuellaSHA256 != q.OriginalHuella ||
		!domain.InstanteUTCCanonico(p.ComprobadaEn) || !domain.InstanteUTCCanonico(p.ValidaHasta) ||
		p.ComprobadaEn.After(ahora) || !ahora.Before(p.ValidaHasta) ||
		p.ValidaHasta.After(p.ComprobadaEn.Add(time.Minute)) {
		return false
	}
	evidencias := []ports.EvidenciaPreparacionExternaR5{
		p.Original, p.PlanCompetenciaVigente, p.PerfilRegistradorVigente,
		p.CustodiaPreparada, p.RegistroConPlanPreparado, p.ConfiguracionVerificadorValidada,
	}
	referencias := make(map[string]bool, len(evidencias))
	for _, e := range evidencias {
		if !domain.ReferenciaOpacaValida(e.Referencia) || referencias[e.Referencia] ||
			e.Version == 0 || !domain.HuellaSHA256FirmaValida(e.HuellaSHA256) ||
			!domain.InstanteUTCCanonico(e.VigenteHasta) || p.ValidaHasta.After(e.VigenteHasta) {
			return false
		}
		referencias[e.Referencia] = true
	}
	return true
}

// El resumen solo cruza ligaduras. El lector existente conserva la verificación
// criptográfica y el consumo SQL auditado de la capacidad original.
type consultaPreflightR5ContextoActual struct {
	origen         ports.AutorizadorConsultaFirmasR5
	contextoRef    string
	contextoHuella string
}

func (a consultaPreflightR5ContextoActual) AutorizarConsultaFirmasR5(ctx context.Context, q ports.MaterialConsultaFirmasR5) (ports.CapacidadConsultaFirmasR5, error) {
	var cero ports.CapacidadConsultaFirmasR5
	capacidad, err := a.origen.AutorizarConsultaFirmasR5(ctx, q)
	if err != nil {
		return cero, err
	}
	material := capacidad.ExportarMaterialParaConsumidor()
	resumen := material.ResumenCapacidad()
	if material.ValidarEstructura() != nil || resumen.ContextoRef() != a.contextoRef ||
		resumen.ContextoHuellaSHA256() != a.contextoHuella {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return capacidad, nil
}
