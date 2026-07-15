package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type repositorioDocumentosEstrictoPrueba struct {
	mu         sync.Mutex
	documentos map[string]domain.DocumentoGenerado
	auditoria  map[string][]domain.AuditEntry
	eventos    []domain.Event
	decisiones map[string]string
}

func nuevoRepositorioDocumentosEstrictoPrueba() *repositorioDocumentosEstrictoPrueba {
	return &repositorioDocumentosEstrictoPrueba{
		documentos: make(map[string]domain.DocumentoGenerado),
		auditoria:  make(map[string][]domain.AuditEntry),
		decisiones: make(map[string]string),
	}
}

func (r *repositorioDocumentosEstrictoPrueba) ConfirmarGeneracion(
	ctx context.Context,
	documento domain.DocumentoGenerado,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	datos, err := evidencia.Datos()
	decision := datos.Decision
	if err != nil || documento.Validar() != nil || evidencia.ValidarEn(documento.GeneradoEn) != nil ||
		decision.DecisionRef != traza.AuthorizationRef || decision.PrincipalID != documento.GeneradoPor ||
		decision.PrincipalID != traza.ActorID || decision.PerfilActivoRef != traza.ActorProfile ||
		decision.RecursoRef != documento.ExpedienteRef || decision.ModuloID != documento.ModuloID ||
		decision.TipoRecurso != "expediente" || decision.Finalidad != traza.Purpose ||
		decision.CorrelacionRef != documento.CorrelacionRef || len(decision.CamposPermitidos) != 0 ||
		len(decision.Obligaciones) != 0 || traza.Action != accionDocumentoGenerado ||
		traza.SubjectRef != documento.ID || traza.DocumentRef != documento.ID ||
		traza.AfterHash != documento.HuellaSHA256 || !traza.OccurredAt.Equal(documento.GeneradoEn) ||
		evento.Type != eventoDocumentoGenerado || evento.SubjectRef != documento.ID ||
		!evento.OccurredAt.Equal(documento.GeneradoEn) ||
		strings.TrimSpace(traza.Metadata["almacen_conector"]) == "" ||
		strings.TrimSpace(traza.Metadata["almacen_evidencia_ref"]) == "" {
		return domain.ErrDocumentoInvalido
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if documentoAnterior, existe := r.decisiones[decision.DecisionRef]; existe {
		if documentoAnterior == documento.ID {
			return nil
		}
		return ports.ErrDecisionAutorizacionConsumida
	}
	if _, existe := r.documentos[documento.ID]; existe {
		return ports.ErrDocumentoYaExiste
	}
	r.documentos[documento.ID] = documento
	r.decisiones[decision.DecisionRef] = documento.ID
	traza.ID = fmt.Sprintf("auditoria-documental-%06d", len(r.auditoria[documento.ID])+1)
	traza.IntegrityAlgorithm = "hmac-sha256-prueba"
	traza.Signature = "firma-auditoria-prueba"
	r.auditoria[documento.ID] = append(r.auditoria[documento.ID], traza)
	evento.ID = fmt.Sprintf("evento-documental-%06d", len(r.eventos)+1)
	evento.Payload = clonarMapaTextoDocumentalPrueba(evento.Payload)
	evento.Payload["auditoria_ref"] = traza.ID
	r.eventos = append(r.eventos, evento)
	return nil
}

func (r *repositorioDocumentosEstrictoPrueba) ObtenerDocumento(
	_ context.Context,
	id string,
) (domain.DocumentoGenerado, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	documento, existe := r.documentos[id]
	if !existe {
		return domain.DocumentoGenerado{}, ports.ErrDocumentoNoEncontrado
	}
	return documento, nil
}

func (r *repositorioDocumentosEstrictoPrueba) ListarDocumentosExpediente(
	_ context.Context,
	expedienteRef string,
) ([]domain.DocumentoGenerado, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	resultado := make([]domain.DocumentoGenerado, 0)
	for _, documento := range r.documentos {
		if documento.ExpedienteRef == expedienteRef {
			resultado = append(resultado, documento)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].ID < resultado[j].ID })
	return resultado, nil
}

func (r *repositorioDocumentosEstrictoPrueba) ListAudit(subjectRef string) []domain.AuditEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.AuditEntry(nil), r.auditoria[subjectRef]...)
}

func (r *repositorioDocumentosEstrictoPrueba) ListEvents(tipo string) []domain.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	resultado := make([]domain.Event, 0)
	for _, evento := range r.eventos {
		if evento.Type == tipo {
			resultado = append(resultado, evento)
		}
	}
	return resultado
}

func clonarMapaTextoDocumentalPrueba(origen map[string]string) map[string]string {
	resultado := make(map[string]string, len(origen)+1)
	for clave, valor := range origen {
		resultado[clave] = valor
	}
	return resultado
}

var _ ports.RepositorioDocumentos = (*repositorioDocumentosEstrictoPrueba)(nil)

type objetoContenidoDocumentalPrueba struct {
	referenciaLogica string
	objeto           ports.ReferenciaObjetoAlmacen
	zona             ports.ZonaAlmacen
	mime             string
	huella           string
	contenido        []byte
}

type almacenContenidoDocumentalEstrictoPrueba struct {
	mu            sync.Mutex
	reloj         ports.Reloj
	siguiente     int
	evidencia     int
	objetos       map[string]objetoContenidoDocumentalPrueba
	idempotencias map[string]string
	errorGuardar  error
}

func nuevoAlmacenContenidoDocumentalEstrictoPrueba(reloj ports.Reloj) *almacenContenidoDocumentalEstrictoPrueba {
	return &almacenContenidoDocumentalEstrictoPrueba{
		reloj: reloj, objetos: make(map[string]objetoContenidoDocumentalPrueba),
		idempotencias: make(map[string]string),
	}
}

func (a *almacenContenidoDocumentalEstrictoPrueba) GuardarContenido(
	_ context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error) {
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.ValidarEn(ahora); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	if a.errorGuardar != nil {
		return ports.ContenidoDocumentoGuardado{}, a.errorGuardar
	}
	sumaPeticion := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%s",
		solicitud.DocumentoID, solicitud.Zona, solicitud.MIME, solicitud.Tamano, solicitud.HuellaSHA256)))
	huellaPeticion := hex.EncodeToString(sumaPeticion[:])
	a.mu.Lock()
	defer a.mu.Unlock()
	if anterior, existe := a.idempotencias[solicitud.ClaveIdempotencia]; existe {
		if anterior != huellaPeticion {
			return ports.ContenidoDocumentoGuardado{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		for _, objeto := range a.objetos {
			if objeto.referenciaLogica == solicitud.DocumentoID && objeto.huella == solicitud.HuellaSHA256 {
				return a.resultadoGuardado(solicitud, objeto, ahora, true), nil
			}
		}
		return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.siguiente++
	objeto := objetoContenidoDocumentalPrueba{
		referenciaLogica: solicitud.DocumentoID,
		objeto: ports.ReferenciaObjetoAlmacen{
			Referencia: fmt.Sprintf("objeto:documental:%06d", a.siguiente), Version: "1",
		},
		zona: solicitud.Zona, mime: solicitud.MIME, huella: solicitud.HuellaSHA256,
		contenido: append([]byte(nil), solicitud.Contenido...),
	}
	a.objetos[objeto.objeto.Referencia] = objeto
	a.idempotencias[solicitud.ClaveIdempotencia] = huellaPeticion
	return a.resultadoGuardado(solicitud, objeto, ahora, false), nil
}

func (a *almacenContenidoDocumentalEstrictoPrueba) LeerContenido(
	_ context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error) {
	ahora := time.Now().UTC()
	if solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora) != nil ||
		!solicitud.Zona.Valida() || solicitud.Limite < 1 {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil || proyeccion.ObjetoVinculado.Referencia != solicitud.Referencia {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	objeto, existe := a.objetos[solicitud.Referencia]
	if !existe || objeto.objeto != proyeccion.ObjetoVinculado || objeto.zona != solicitud.Zona {
		return ports.ContenidoDocumentoLeido{}, ports.ErrContenidoDocumentoNoEncontrado
	}
	if int64(len(objeto.contenido)) > solicitud.Limite {
		return ports.ContenidoDocumentoLeido{}, ports.ErrLimiteLecturaExcedido
	}
	evidencia := a.nuevaEvidencia(solicitud.Contexto, objeto.objeto, ahora, false)
	return ports.ContenidoDocumentoLeido{
		Contenido: append([]byte(nil), objeto.contenido...), ConectorID: evidencia.ConectorID,
		Zona: objeto.zona, HuellaSHA256: objeto.huella, Tamano: int64(len(objeto.contenido)),
		EvidenciaOperacion: evidencia,
	}, nil
}

func (a *almacenContenidoDocumentalEstrictoPrueba) resultadoGuardado(
	solicitud ports.SolicitudGuardarContenido,
	objeto objetoContenidoDocumentalPrueba,
	ahora time.Time,
	repetido bool,
) ports.ContenidoDocumentoGuardado {
	evidencia := a.nuevaEvidencia(solicitud.Contexto, objeto.objeto, ahora, repetido)
	return ports.ContenidoDocumentoGuardado{
		ReferenciaLogica: solicitud.DocumentoID, Referencia: objeto.objeto.Referencia,
		Version: objeto.objeto.Version, ConectorID: evidencia.ConectorID, Zona: objeto.zona,
		MIME: objeto.mime, HuellaSHA256: objeto.huella, Tamano: int64(len(objeto.contenido)),
		EvidenciaOperacion: evidencia,
	}
}

func (a *almacenContenidoDocumentalEstrictoPrueba) nuevaEvidencia(
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ReferenciaObjetoAlmacen,
	ahora time.Time,
	repetida bool,
) ports.EvidenciaOperacionAlmacen {
	a.evidencia++
	proyeccion, _ := contexto.Proyeccion()
	return ports.EvidenciaOperacionAlmacen{
		Referencia: fmt.Sprintf("evidencia:almacen-documental:%06d", a.evidencia),
		ConectorID: "almacen_documental_estricto_prueba", EsquemaContexto: proyeccion.Esquema,
		AccionNegocio: proyeccion.AccionNegocio, Accion: proyeccion.AccionTecnica,
		EfectoRef: proyeccion.EfectoRef, HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256,
		HuellaPasoSHA256:       proyeccion.HuellaPasoSHA256, PasoRef: proyeccion.PasoRef,
		HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256, Objeto: objeto,
		OperacionRef: proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: ahora,
		CargaRef: proyeccion.CargaRef, SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC,
		RecursoRef: proyeccion.RecursoRef, ModuloID: proyeccion.ModuloID,
		HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC, ReintentoIdempotente: repetida,
	}
}

var _ ports.AlmacenContenidoDocumento = (*almacenContenidoDocumentalEstrictoPrueba)(nil)

type almacenContenidoAmbiguoPrueba struct {
	base     ports.AlmacenContenidoDocumento
	llamadas int
	causa    error
}

func (a *almacenContenidoAmbiguoPrueba) GuardarContenido(
	ctx context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error) {
	a.llamadas++
	// El conector aplica el efecto pero la respuesta se pierde: el llamador no
	// puede distinguirlo de un fallo anterior al COMMIT remoto.
	_, _ = a.base.GuardarContenido(ctx, solicitud)
	return ports.ContenidoDocumentoGuardado{}, a.causa
}

func (a *almacenContenidoAmbiguoPrueba) LeerContenido(
	ctx context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error) {
	return a.base.LeerContenido(ctx, solicitud)
}

type almacenContenidoRepitePrimerResultadoPrueba struct {
	base     ports.AlmacenContenidoDocumento
	llamadas int
	primero  ports.ContenidoDocumentoGuardado
}

func (a *almacenContenidoRepitePrimerResultadoPrueba) GuardarContenido(
	ctx context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error) {
	a.llamadas++
	if a.llamadas == 1 {
		resultado, err := a.base.GuardarContenido(ctx, solicitud)
		a.primero = resultado
		return resultado, err
	}
	return a.primero, nil
}

func (a *almacenContenidoRepitePrimerResultadoPrueba) LeerContenido(
	ctx context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error) {
	return a.base.LeerContenido(ctx, solicitud)
}

var _ ports.AlmacenContenidoDocumento = (*almacenContenidoAmbiguoPrueba)(nil)
var _ ports.AlmacenContenidoDocumento = (*almacenContenidoRepitePrimerResultadoPrueba)(nil)

// registroEfectosDocumentalesPrueba es un fake estricto y solo de pruebas. No
// pretende sustituir el adaptador transaccional productivo: conserva las dos
// unicidades, valida cada capacidad opaca y hace transiciones condicionales.
type registroEfectosDocumentalesPrueba struct {
	mu sync.Mutex

	siguiente       int
	porDecision     map[string]*efectoDocumentalPrueba
	porEfecto       map[string]*efectoDocumentalPrueba
	porReserva      map[string]*efectoDocumentalPrueba
	reservas        int
	confirmaciones  int
	indeterminados  int
	errorReserva    error
	errorConfirmar  error
	errorMarcar     error
	despuesReservar func()
}

type efectoDocumentalPrueba struct {
	reservaRef       string
	efectoRef        string
	decisionRef      string
	huellaDecision   string
	huellaPlan       string
	huellaManifiesto string
	pasos            []ports.EstadoPasoDuraderoGeneracionDocumental
}

func nuevoRegistroEfectosDocumentalesPrueba() *registroEfectosDocumentalesPrueba {
	return &registroEfectosDocumentalesPrueba{
		porDecision: make(map[string]*efectoDocumentalPrueba),
		porEfecto:   make(map[string]*efectoDocumentalPrueba),
		porReserva:  make(map[string]*efectoDocumentalPrueba),
	}
}

func (r *registroEfectosDocumentalesPrueba) ReservarEfectoGeneracionDocumental(
	_ context.Context,
	solicitud ports.SolicitudReservarEfectoGeneracionDocumental,
) (ports.ResultadoReservaEfectoGeneracionDocumental, error) {
	contexto, errContexto := solicitud.Contexto.Proyeccion()
	manifiesto, errManifiesto := solicitud.Manifiesto.Proyeccion()
	if errContexto != nil || errManifiesto != nil || solicitud.ValidarEn(contexto.VerificadaEn) != nil {
		return ports.ResultadoReservaEfectoGeneracionDocumental{}, ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if r.errorReserva != nil {
		return ports.ResultadoReservaEfectoGeneracionDocumental{}, r.errorReserva
	}
	if anterior := r.porDecision[contexto.AutorizacionRef]; anterior != nil {
		resultado := anterior.resultado(true)
		if resultado.ValidarContra(solicitud) != nil {
			return ports.ResultadoReservaEfectoGeneracionDocumental{}, ports.ErrDecisionAutorizacionConsumida
		}
		return resultado, nil
	}
	if anterior := r.porEfecto[contexto.EfectoRef]; anterior != nil {
		return ports.ResultadoReservaEfectoGeneracionDocumental{}, ports.ErrDecisionAutorizacionConsumida
	}
	r.siguiente++
	efecto := &efectoDocumentalPrueba{
		reservaRef: "reserva:efecto-documental:" + fmt.Sprintf("%06d", r.siguiente),
		efectoRef:  contexto.EfectoRef, decisionRef: contexto.AutorizacionRef,
		huellaDecision: contexto.HuellaDecisionSHA256, huellaPlan: contexto.HuellaPlanEfectoSHA256,
		huellaManifiesto: contexto.HuellaManifiestoSHA256,
		pasos:            make([]ports.EstadoPasoDuraderoGeneracionDocumental, 0, len(manifiesto.Pasos)),
	}
	for _, paso := range manifiesto.Pasos {
		efecto.pasos = append(efecto.pasos, ports.EstadoPasoDuraderoGeneracionDocumental{
			PasoRef: paso.PasoRef, HuellaPasoSHA256: paso.HuellaPasoSHA256,
			Estado: ports.EstadoPasoEfectoDocumentalReservado,
		})
	}
	resultado := efecto.resultado(false)
	if resultado.ValidarContra(solicitud) != nil {
		return ports.ResultadoReservaEfectoGeneracionDocumental{}, ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	r.porDecision[efecto.decisionRef] = efecto
	r.porEfecto[efecto.efectoRef] = efecto
	r.porReserva[efecto.reservaRef] = efecto
	if r.despuesReservar != nil {
		r.despuesReservar()
	}
	return resultado, nil
}

func (r *registroEfectosDocumentalesPrueba) ConfirmarPasoGeneracionDocumental(
	_ context.Context,
	solicitud ports.SolicitudConfirmarPasoGeneracionDocumental,
) error {
	if solicitud.Validar() != nil {
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	contexto, err := solicitud.Contexto.Proyeccion()
	if err != nil {
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.confirmaciones++
	if r.errorConfirmar != nil {
		return r.errorConfirmar
	}
	efecto := r.porReserva[solicitud.ReservaRef]
	indice, err := efecto.indicePaso(contexto)
	if err != nil {
		return err
	}
	estado := &efecto.pasos[indice]
	objeto := ports.ReferenciaObjetoAlmacen{Referencia: solicitud.Guardado.Referencia, Version: solicitud.Guardado.Version}
	switch estado.Estado {
	case ports.EstadoPasoEfectoDocumentalReservado:
		estado.Estado = ports.EstadoPasoEfectoDocumentalConfirmado
		estado.Objeto = objeto
		estado.ConectorID = solicitud.Guardado.ConectorID
		estado.EvidenciaOperacionRef = solicitud.Guardado.EvidenciaOperacion.Referencia
		return nil
	case ports.EstadoPasoEfectoDocumentalConfirmado:
		if estado.Objeto == objeto && estado.ConectorID == solicitud.Guardado.ConectorID &&
			estado.EvidenciaOperacionRef == solicitud.Guardado.EvidenciaOperacion.Referencia {
			return nil
		}
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	case ports.EstadoPasoEfectoDocumentalIndeterminado:
		return ports.ErrPasoGeneracionDocumentalIndeterminado
	default:
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
}

func (r *registroEfectosDocumentalesPrueba) MarcarPasoGeneracionDocumentalIndeterminado(
	_ context.Context,
	solicitud ports.SolicitudMarcarPasoGeneracionDocumentalIndeterminado,
) error {
	if solicitud.Validar() != nil {
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	contexto, err := solicitud.Contexto.Proyeccion()
	if err != nil {
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.indeterminados++
	if r.errorMarcar != nil {
		return r.errorMarcar
	}
	efecto := r.porReserva[solicitud.ReservaRef]
	indice, err := efecto.indicePaso(contexto)
	if err != nil {
		return err
	}
	estado := &efecto.pasos[indice]
	switch estado.Estado {
	case ports.EstadoPasoEfectoDocumentalReservado:
		estado.Estado = ports.EstadoPasoEfectoDocumentalIndeterminado
		estado.IncidenteRef = solicitud.IncidenteRef
		return nil
	case ports.EstadoPasoEfectoDocumentalIndeterminado:
		if estado.IncidenteRef == solicitud.IncidenteRef {
			return nil
		}
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	case ports.EstadoPasoEfectoDocumentalConfirmado:
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	default:
		return ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
}

func (e *efectoDocumentalPrueba) indicePaso(contexto ports.ProyeccionContextoOperacionAlmacen) (int, error) {
	if e == nil || e.efectoRef != contexto.EfectoRef || e.decisionRef != contexto.AutorizacionRef ||
		e.huellaDecision != contexto.HuellaDecisionSHA256 || e.huellaPlan != contexto.HuellaPlanEfectoSHA256 ||
		e.huellaManifiesto != contexto.HuellaManifiestoSHA256 {
		return 0, ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	for indice := range e.pasos {
		if e.pasos[indice].PasoRef == contexto.PasoRef && e.pasos[indice].HuellaPasoSHA256 == contexto.HuellaPasoSHA256 {
			return indice, nil
		}
	}
	return 0, ports.ErrReservaEfectoGeneracionDocumentalInvalida
}

func (e *efectoDocumentalPrueba) resultado(repetida bool) ports.ResultadoReservaEfectoGeneracionDocumental {
	return ports.ResultadoReservaEfectoGeneracionDocumental{
		ReservaRef: e.reservaRef, EfectoRef: e.efectoRef,
		HuellaDecisionSHA256: e.huellaDecision, HuellaPlanEfectoSHA256: e.huellaPlan,
		HuellaManifiestoSHA256: e.huellaManifiesto, Repetida: repetida,
		Pasos: append([]ports.EstadoPasoDuraderoGeneracionDocumental(nil), e.pasos...),
	}
}

func (r *registroEfectosDocumentalesPrueba) primerEfecto() (*efectoDocumentalPrueba, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, efecto := range r.porReserva {
		copia := *efecto
		copia.pasos = append([]ports.EstadoPasoDuraderoGeneracionDocumental(nil), efecto.pasos...)
		return &copia, nil
	}
	return nil, errors.New("sin efectos documentales")
}
