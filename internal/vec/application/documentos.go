package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaDocumentalRequerida = errors.New("vec: dependencia documental requerida")
	ErrRenderizadorNoDisponible       = errors.New("vec: renderizador documental no disponible")
	ErrOrdenDocumentalInvalida        = errors.New("vec: orden documental invalida")
	ErrDocumentoDemasiadoGrande       = errors.New("vec: documento generado demasiado grande")
	ErrConfirmacionContenidoInvalida  = errors.New("vec: confirmacion de contenido invalida")
)

const (
	limiteDocumentoPredeterminado         = int64(32 * 1024 * 1024)
	limiteDocumentoMaximo                 = int64(256 * 1024 * 1024)
	AccionCrearBorradorPlantillaDocumento = "vec.documentos.plantillas.crear"
	AccionPublicarPlantillaDocumento      = "vec.documentos.plantillas.publicar"
	accionDocumentoGenerado               = "vec.documento.generado"
	eventoDocumentoGenerado               = "vec.documento.generado"
)

type OpcionesServicioDocumental struct {
	OrganoENI   string
	LimiteBytes int64
}

// ServicioDocumental orquesta la generacion sin conocer PDF, DOCX, S3,
// PostgreSQL ni el proveedor de sellado. Todos esos detalles viven detras de
// puertos intercambiables.
type ServicioDocumental struct {
	catalogo          ports.CatalogoPlantillasDocumento
	gobierno          ports.RepositorioGobiernoPlantillasDocumento
	autorizador       ports.Autorizador
	almacen           ports.AlmacenContenidoDocumento
	registroEfectos   ports.RegistroEfectosGeneracionDocumental
	repositorio       ports.RepositorioDocumentos
	repositorioLogico ports.RepositorioDocumentosLogicos
	selladorDatos     ports.SelladorDatosDocumento
	selladorSolicitud ports.SelladorSolicitudDocumento
	seudonimizador    ports.SeudonimizadorSujetoAlmacen
	generadorID       ports.GeneradorIDDocumento
	reloj             ports.Reloj
	renderizadores    map[domain.FormatoDocumento]ports.RenderizadorDocumento
	organoENI         string
	limiteBytes       int64
}

func NuevoServicioDocumental(
	catalogo ports.CatalogoPlantillasDocumento,
	gobierno ports.RepositorioGobiernoPlantillasDocumento,
	autorizador ports.Autorizador,
	almacen ports.AlmacenContenidoDocumento,
	registroEfectos ports.RegistroEfectosGeneracionDocumental,
	repositorio ports.RepositorioDocumentos,
	repositorioLogico ports.RepositorioDocumentosLogicos,
	selladorDatos ports.SelladorDatosDocumento,
	selladorSolicitud ports.SelladorSolicitudDocumento,
	seudonimizador ports.SeudonimizadorSujetoAlmacen,
	generadorID ports.GeneradorIDDocumento,
	reloj ports.Reloj,
	opciones OpcionesServicioDocumental,
	renderizadores ...ports.RenderizadorDocumento,
) (*ServicioDocumental, error) {
	dependencias := []any{
		catalogo, gobierno, autorizador, almacen, registroEfectos, repositorio, repositorioLogico,
		selladorDatos, selladorSolicitud, seudonimizador, generadorID, reloj,
	}
	for _, dependencia := range dependencias {
		if dependenciaDocumentalNula(dependencia) {
			return nil, ErrDependenciaDocumentalRequerida
		}
	}
	if strings.TrimSpace(opciones.OrganoENI) == "" {
		return nil, ErrDependenciaDocumentalRequerida
	}
	limite := opciones.LimiteBytes
	if limite == 0 {
		limite = limiteDocumentoPredeterminado
	}
	if limite < 1 || limite > limiteDocumentoMaximo {
		return nil, ErrDependenciaDocumentalRequerida
	}
	porFormato := make(map[domain.FormatoDocumento]ports.RenderizadorDocumento, len(renderizadores))
	for _, renderizador := range renderizadores {
		if dependenciaDocumentalNula(renderizador) || !renderizador.Formato().Valido() {
			return nil, ErrDependenciaDocumentalRequerida
		}
		if _, repetido := porFormato[renderizador.Formato()]; repetido {
			return nil, ErrDependenciaDocumentalRequerida
		}
		porFormato[renderizador.Formato()] = renderizador
	}
	if len(porFormato) == 0 {
		return nil, ErrDependenciaDocumentalRequerida
	}
	return &ServicioDocumental{
		catalogo:          catalogo,
		gobierno:          gobierno,
		autorizador:       autorizador,
		almacen:           almacen,
		registroEfectos:   registroEfectos,
		repositorio:       repositorio,
		repositorioLogico: repositorioLogico,
		selladorDatos:     selladorDatos,
		selladorSolicitud: selladorSolicitud,
		seudonimizador:    seudonimizador,
		generadorID:       generadorID,
		reloj:             reloj,
		renderizadores:    porFormato,
		organoENI:         strings.TrimSpace(opciones.OrganoENI),
		limiteBytes:       limite,
	}, nil
}

func dependenciaDocumentalNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

type OrdenCrearBorradorPlantilla struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	TipoDocumental string
	Nombre         string
	Titulo         string
	Parrafos       []string
	Campos         []domain.CampoPlantillaDocumento
	Formatos       []domain.FormatoDocumento
	PermisoGenerar string
	GarantiaMinima domain.AuthAssurance
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioDocumental) CrearBorradorPlantilla(ctx context.Context, orden OrdenCrearBorradorPlantilla) (domain.PlantillaDocumento, error) {
	if err := validarContextoGobiernoPlantilla(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.PlantillaDocumento{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.PlantillaDocumento{}, domain.ErrGarantiaInsuficiente
	}
	ahora := s.reloj.Ahora().UTC()
	plantilla := domain.PlantillaDocumento{
		ID:             strings.TrimSpace(orden.ID),
		Version:        orden.Version,
		ModuloID:       strings.TrimSpace(orden.ModuloID),
		TipoDocumental: strings.TrimSpace(orden.TipoDocumental),
		Nombre:         strings.TrimSpace(orden.Nombre),
		Titulo:         orden.Titulo,
		Parrafos:       append([]string(nil), orden.Parrafos...),
		Campos:         append([]domain.CampoPlantillaDocumento(nil), orden.Campos...),
		Formatos:       append([]domain.FormatoDocumento(nil), orden.Formatos...),
		PermisoGenerar: strings.TrimSpace(orden.PermisoGenerar),
		GarantiaMinima: orden.GarantiaMinima,
		Estado:         domain.EstadoPlantillaBorrador,
		CreadaPor:      orden.Principal.ID,
		CreadaEn:       ahora,
	}
	if err := plantilla.Validar(); err != nil {
		return domain.PlantillaDocumento{}, err
	}
	decision, err := s.exigirAutorizacionDocumental(ctx, orden.Principal, orden.PerfilActivo,
		AccionCrearBorradorPlantillaDocumento, domain.RecursoAutorizable{
			Referencia: referenciaPlantilla(plantilla),
			ModuloID:   plantilla.ModuloID,
			Tipo:       "plantilla_documento",
			Atributos: map[string]string{
				"estado": string(plantilla.Estado),
			},
		}, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	huella, err := plantilla.HuellaSHA256()
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	referencia := referenciaPlantilla(plantilla)
	traza := trazaGobiernoPlantilla(orden.Principal, orden.PerfilActivo, decision.DecisionRef, orden.Finalidad,
		"vec.documentos.plantilla.borrador.creado", plantilla, referencia, orden.Motivo, orden.CorrelacionRef, "", huella, ahora)
	evento := eventoGobiernoPlantilla("vec.documentos.plantilla.borrador.creado", plantilla, referencia, huella, orden.Principal.ID, ahora)
	if err := s.gobierno.ConfirmarAltaBorradorPlantilla(ctx, plantilla, traza, evento); err != nil {
		return domain.PlantillaDocumento{}, fmt.Errorf("confirmar borrador de plantilla: %w", err)
	}
	return plantilla, nil
}

type OrdenPublicarPlantilla struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	PlantillaID    string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioDocumental) PublicarPlantilla(ctx context.Context, orden OrdenPublicarPlantilla) (domain.PlantillaDocumento, error) {
	if err := validarContextoGobiernoPlantilla(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.PlantillaDocumento{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.PlantillaDocumento{}, domain.ErrGarantiaInsuficiente
	}
	if strings.TrimSpace(orden.AprobacionRef) == "" || strings.TrimSpace(orden.PlantillaID) == "" || orden.Version < 1 {
		return domain.PlantillaDocumento{}, ErrOrdenDocumentalInvalida
	}
	borrador, err := s.catalogo.ObtenerPlantilla(ctx, orden.PlantillaID, orden.Version)
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	decision, err := s.exigirAutorizacionDocumental(ctx, orden.Principal, orden.PerfilActivo,
		AccionPublicarPlantillaDocumento, domain.RecursoAutorizable{
			Referencia: referenciaPlantilla(borrador),
			ModuloID:   borrador.ModuloID,
			Tipo:       "plantilla_documento",
			Atributos: map[string]string{
				"estado": string(borrador.Estado),
			},
		}, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	huellaBorrador, err := borrador.HuellaSHA256()
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	publicada, err := borrador.Publicar(orden.Principal.ID, orden.AprobacionRef, orden.Motivo, ahora)
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	huellaPublicada, err := publicada.HuellaSHA256()
	if err != nil {
		return domain.PlantillaDocumento{}, err
	}
	referencia := referenciaPlantilla(publicada)
	traza := trazaGobiernoPlantilla(orden.Principal, orden.PerfilActivo, decision.DecisionRef, orden.Finalidad,
		"vec.documentos.plantilla.publicada", publicada, referencia, orden.Motivo, orden.CorrelacionRef,
		huellaBorrador, huellaPublicada, ahora)
	traza.RuleRef = strings.TrimSpace(orden.AprobacionRef)
	evento := eventoGobiernoPlantilla("vec.documentos.plantilla.publicada", publicada, referencia, huellaPublicada, orden.Principal.ID, ahora)
	if err := s.gobierno.ConfirmarPublicacionPlantilla(ctx, huellaBorrador, publicada, traza, evento); err != nil {
		return domain.PlantillaDocumento{}, fmt.Errorf("confirmar publicacion de plantilla: %w", err)
	}
	return publicada, nil
}

// OrdenGenerarDocumento contiene contexto administrativo, no solo datos de
// presentacion. Ninguna generacion queda huerfana de expediente o decision de
// autorizacion.
type OrdenGenerarDocumento struct {
	Principal        domain.Principal
	PerfilActivo     string
	RepresentadoRef  string
	Finalidad        string
	Clasificacion    string
	PlantillaID      string
	PlantillaVersion int
	Formato          domain.FormatoDocumento
	ExpedienteRef    string
	Datos            map[string]string
	Motivo           string
	CorrelacionRef   string
}

func (s *ServicioDocumental) Generar(ctx context.Context, orden OrdenGenerarDocumento) (domain.DocumentoGenerado, error) {
	if err := ctx.Err(); err != nil {
		return domain.DocumentoGenerado{}, err
	}
	if err := orden.Principal.Validate(); err != nil {
		return domain.DocumentoGenerado{}, err
	}
	if strings.TrimSpace(orden.PerfilActivo) == "" || strings.TrimSpace(orden.Finalidad) == "" ||
		strings.TrimSpace(orden.Clasificacion) == "" ||
		strings.TrimSpace(orden.PlantillaID) == "" ||
		orden.PlantillaVersion < 1 || !orden.Formato.Valido() ||
		strings.TrimSpace(orden.ExpedienteRef) == "" || strings.TrimSpace(orden.Motivo) == "" ||
		strings.TrimSpace(orden.CorrelacionRef) == "" {
		return domain.DocumentoGenerado{}, ErrOrdenDocumentalInvalida
	}

	plantilla, err := s.catalogo.ObtenerPlantilla(ctx, orden.PlantillaID, orden.PlantillaVersion)
	if err != nil {
		return domain.DocumentoGenerado{}, err
	}
	if err := plantilla.Validar(); err != nil {
		return domain.DocumentoGenerado{}, err
	}
	if plantilla.Estado != domain.EstadoPlantillaPublicada {
		return domain.DocumentoGenerado{}, domain.ErrPlantillaNoPublicada
	}
	if !orden.Principal.AuthAssurance.Cumple(plantilla.GarantiaMinima) {
		return domain.DocumentoGenerado{}, domain.ErrGarantiaInsuficiente
	}
	if !plantilla.AdmiteFormato(orden.Formato) {
		return domain.DocumentoGenerado{}, domain.ErrFormatoDocumentoInvalido
	}
	renderizador, existe := s.renderizadores[orden.Formato]
	if !existe {
		return domain.DocumentoGenerado{}, ErrRenderizadorNoDisponible
	}

	contenidoFusionado, err := plantilla.Fusionar(orden.Datos)
	if err != nil {
		return domain.DocumentoGenerado{}, err
	}
	contenido, err := renderizador.Renderizar(ctx, contenidoFusionado)
	if err != nil {
		return domain.DocumentoGenerado{}, fmt.Errorf("renderizar %s: %w", orden.Formato, err)
	}
	if len(contenido) == 0 {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	if int64(len(contenido)) > s.limiteBytes {
		return domain.DocumentoGenerado{}, ErrDocumentoDemasiadoGrande
	}
	if err := renderizador.ValidarSalida(ctx, contenido); err != nil {
		return domain.DocumentoGenerado{}, fmt.Errorf("validar salida %s: %w", orden.Formato, err)
	}
	documentoID, err := s.generadorID.NuevoIDDocumento()
	if err != nil {
		return domain.DocumentoGenerado{}, fmt.Errorf("generar identificador documental: %w", err)
	}
	if strings.TrimSpace(documentoID) == "" {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	huellaContenido := sha256.Sum256(contenido)
	huellaSHA256 := hex.EncodeToString(huellaContenido[:])
	datosCanonicos := datosFusionCanonicos(plantilla, orden)
	huellaDatos, err := s.selladorDatos.SellarDatos(ctx, datosCanonicos)
	if err != nil {
		return domain.DocumentoGenerado{}, fmt.Errorf("sellar datos documentales: %w", err)
	}
	if strings.TrimSpace(huellaDatos) == "" {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	huellaSolicitud, err := s.selladorSolicitud.SellarSolicitudDocumento(
		ctx, solicitudDocumentoGeneradoCanonica(plantilla, orden),
	)
	if err != nil || !huellaHMACDocumentalValida(huellaSolicitud) {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	plan, err := s.ejecutarPlanGeneracionDocumental(ctx, solicitudPlanGeneracionDocumental{
		principal: orden.Principal, perfilActivo: orden.PerfilActivo,
		finalidad: orden.Finalidad, motivo: orden.Motivo, correlacionRef: orden.CorrelacionRef,
		clasificacion: orden.Clasificacion, plantilla: plantilla,
		recursoBase: domain.RecursoAutorizable{
			Referencia: strings.TrimSpace(orden.ExpedienteRef), ModuloID: plantilla.ModuloID, Tipo: "expediente",
			Atributos: map[string]string{
				"plantilla_id": plantilla.ID, "tipo_documental": plantilla.TipoDocumental,
				"representaciones": "1",
			},
		},
		sujetoRef:       sujetoDocumentoRef(orden.Principal.ID, orden.RepresentadoRef),
		ambitoSujetoRef: "documento:" + documentoID + ":" + huellaSolicitud,
		operacionRef:    "almacenar-documento:" + documentoID,
		cargaRef:        documentoID,
		// El efecto se deriva de un HMAC estable de la solicitud, no del ID
		// aleatorio del intento. Tras un crash, un intento nuevo no puede reservar
		// silenciosamente otro efecto para la misma peticion.
		efectoRef:       "efecto-generacion-documento:" + huellaSolicitud,
		huellaSolicitud: huellaSolicitud,
		contenidos: []contenidoGeneracionDocumental{{
			declaracion: ports.DeclaracionRepresentacionGeneracionDocumental{
				ReferenciaLogica: documentoID, ClaveIdempotencia: "contenido-documento:" + documentoID,
				Formato: orden.Formato, Zona: ports.ZonaAlmacenAdmitida, MIME: orden.Formato.MIME(),
				Tamano: int64(len(contenido)), HuellaSHA256: huellaSHA256,
			},
			contenido: contenido,
		}},
	})
	if err != nil {
		return domain.DocumentoGenerado{}, err
	}
	pasoGuardado, existe := plan.pasos[documentoID]
	if !existe || pasoGuardado.objeto.Validar() != nil || pasoGuardado.evidenciaOperacion == "" {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	// La evidencia se crea despues del ultimo efecto externo y constituye la
	// ultima revalidacion antes de la confirmacion atomica. El mismo instante
	// canonico se incorpora al agregado, la auditoria y el evento para que el
	// repositorio pueda ligar sin inferencias la decision al efecto confirmado.
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() {
		return domain.DocumentoGenerado{}, ErrConfirmacionContenidoInvalida
	}
	evidenciaAutorizacion, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(plan.decision, ahora)
	if err != nil {
		return domain.DocumentoGenerado{}, err
	}
	estado := domain.EstadoDocumentoGenerado
	if orden.Formato == domain.FormatoDocumentoDOCX {
		estado = domain.EstadoDocumentoBorrador
	}
	documento := domain.DocumentoGenerado{
		ID:                  documentoID,
		Version:             1,
		PlantillaID:         plantilla.ID,
		PlantillaVersion:    plantilla.Version,
		ModuloID:            plantilla.ModuloID,
		TipoDocumental:      plantilla.TipoDocumental,
		ExpedienteRef:       strings.TrimSpace(orden.ExpedienteRef),
		Formato:             orden.Formato,
		MIME:                orden.Formato.MIME(),
		NombreFichero:       nombreFicheroDocumento(plantilla.TipoDocumental, documentoID, orden.Formato),
		Tamano:              int64(len(contenido)),
		HuellaSHA256:        huellaSHA256,
		HuellaDatosHMAC:     huellaDatos,
		ReferenciaContenido: pasoGuardado.objeto.Referencia,
		Estado:              estado,
		EstadoAntivirus:     domain.EstadoAntivirusNoAplica,
		GeneradoPor:         orden.Principal.ID,
		GeneradoEn:          ahora,
		CorrelacionRef:      strings.TrimSpace(orden.CorrelacionRef),
		Motivo:              strings.TrimSpace(orden.Motivo),
		ENI: domain.MetadatosENI{
			Identificador:     documentoID,
			Organo:            s.organoENI,
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    plantilla.TipoDocumental,
			FechaCaptura:      ahora,
		},
	}
	if err := documento.Validar(); err != nil {
		return domain.DocumentoGenerado{}, err
	}

	traza := domain.AuditEntry{
		ActorID:              orden.Principal.ID,
		ActorProfile:         strings.TrimSpace(orden.PerfilActivo),
		ActorRoles:           append([]string(nil), orden.Principal.Roles...),
		RepresentedSubjectID: strings.TrimSpace(orden.RepresentadoRef),
		AuthMethod:           orden.Principal.AuthMethod,
		AuthAssurance:        orden.Principal.AuthAssurance,
		AuthorizationRef:     plan.decision.DecisionRef,
		Purpose:              strings.TrimSpace(orden.Finalidad),
		Action:               accionDocumentoGenerado,
		ModuleID:             plantilla.ModuloID,
		SubjectRef:           documento.ID,
		ObjectVersion:        documento.Version,
		ExpedienteRef:        documento.ExpedienteRef,
		DocumentRef:          documento.ID,
		RuleRef:              plantilla.ID + ":" + strconv.Itoa(plantilla.Version),
		Reason:               documento.Motivo,
		Result:               "correcto",
		AfterHash:            documento.HuellaSHA256,
		CorrelationRef:       documento.CorrelacionRef,
		OccurredAt:           ahora,
		Metadata: map[string]string{
			"almacen_conector":      pasoGuardado.conectorID,
			"almacen_evidencia_ref": pasoGuardado.evidenciaOperacion,
			"formato":               string(documento.Formato),
			"huella_datos_hmac":     documento.HuellaDatosHMAC,
			"mime":                  documento.MIME,
			"plantilla_id":          documento.PlantillaID,
			"plantilla_version":     strconv.Itoa(documento.PlantillaVersion),
			"tamano":                strconv.FormatInt(documento.Tamano, 10),
		},
	}
	evento := domain.Event{
		Type:       eventoDocumentoGenerado,
		ModuleID:   plantilla.ModuloID,
		SubjectRef: documento.ID,
		ActorID:    orden.Principal.ID,
		OccurredAt: ahora,
		Payload: map[string]string{
			"documento_ref":  documento.ID,
			"expediente_ref": documento.ExpedienteRef,
			"formato":        string(documento.Formato),
			"huella_sha256":  documento.HuellaSHA256,
		},
	}
	if err := s.repositorio.ConfirmarGeneracion(ctx, documento, traza, evento, evidenciaAutorizacion); err != nil {
		return domain.DocumentoGenerado{}, fmt.Errorf("confirmar generacion documental: %w", err)
	}
	return documento, nil
}

func evidenciaAlmacenCorresponde(
	evidencia ports.EvidenciaOperacionAlmacen,
	contexto ports.ContextoOperacionAlmacen,
) bool {
	proyeccion, err := contexto.Proyeccion()
	return err == nil && evidencia.Validar() == nil &&
		evidencia.EsquemaContexto == proyeccion.Esquema &&
		evidencia.OperacionRef == proyeccion.OperacionRef &&
		evidencia.CorrelacionRef == proyeccion.CorrelacionRef &&
		evidencia.AutorizacionRef == proyeccion.AutorizacionRef &&
		evidencia.Finalidad == proyeccion.Finalidad &&
		evidencia.Clasificacion == proyeccion.Clasificacion &&
		evidencia.AccionNegocio == proyeccion.AccionNegocio &&
		evidencia.Accion == proyeccion.AccionTecnica &&
		evidencia.EfectoRef == proyeccion.EfectoRef &&
		evidencia.HuellaPlanEfectoSHA256 == proyeccion.HuellaPlanEfectoSHA256 &&
		evidencia.PasoRef == proyeccion.PasoRef &&
		evidencia.HuellaDecisionSHA256 == proyeccion.HuellaDecisionSHA256 &&
		evidencia.CargaRef == proyeccion.CargaRef &&
		evidencia.SujetoSeudonimoHMAC == proyeccion.SujetoSeudonimoHMAC &&
		evidencia.RecursoRef == proyeccion.RecursoRef &&
		evidencia.ModuloID == proyeccion.ModuloID &&
		evidencia.HuellaSolicitudHMAC == proyeccion.HuellaSolicitudHMAC &&
		evidencia.FundamentoRef == ""
}

func (s *ServicioDocumental) seudonimizarSujetoDocumento(
	ctx context.Context,
	sujetoRef, ambitoRef string,
) (string, error) {
	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(sujetoRef, ambitoRef)
	if err != nil {
		return "", err
	}
	seudonimo, err := s.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitud)
	if err != nil || !huellaHMACDocumentalValida(seudonimo) {
		return "", ports.ErrSeudonimizacionAlmacenNoDisponible
	}
	return seudonimo, nil
}

func sujetoDocumentoRef(principalID, representadoRef string) string {
	if representado := strings.TrimSpace(representadoRef); representado != "" {
		return representado
	}
	return strings.TrimSpace(principalID)
}

func huellaHMACDocumentalValida(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] == "" ||
		partes[1] != strings.TrimSpace(partes[1]) || strings.ContainsAny(partes[1], " \t\r\n") ||
		len(partes[1]) > 64 || len(partes[2]) != sha256.Size*2 || partes[2] != strings.ToLower(partes[2]) {
		return false
	}
	decodificada, err := hex.DecodeString(partes[2])
	return err == nil && len(decodificada) == sha256.Size
}

func solicitudDocumentoGeneradoCanonica(
	plantilla domain.PlantillaDocumento,
	orden OrdenGenerarDocumento,
) []byte {
	var salida strings.Builder
	escribirCampoCanonico(&salida, "esquema", "solicitud-documento-generado-v1")
	escribirCampoCanonico(&salida, "principal_id", strings.TrimSpace(orden.Principal.ID))
	escribirCampoCanonico(&salida, "perfil_activo", strings.TrimSpace(orden.PerfilActivo))
	escribirCampoCanonico(&salida, "representado_ref", strings.TrimSpace(orden.RepresentadoRef))
	escribirCampoCanonico(&salida, "finalidad", strings.TrimSpace(orden.Finalidad))
	escribirCampoCanonico(&salida, "clasificacion", strings.TrimSpace(orden.Clasificacion))
	escribirCampoCanonico(&salida, "plantilla_id", plantilla.ID)
	escribirCampoCanonico(&salida, "plantilla_version", strconv.Itoa(plantilla.Version))
	escribirCampoCanonico(&salida, "formato", string(orden.Formato))
	escribirCampoCanonico(&salida, "expediente_ref", strings.TrimSpace(orden.ExpedienteRef))
	escribirCampoCanonico(&salida, "datos_canonicos", string(datosFusionCanonicos(plantilla, orden)))
	escribirCampoCanonico(&salida, "motivo", strings.TrimSpace(orden.Motivo))
	escribirCampoCanonico(&salida, "correlacion_ref", strings.TrimSpace(orden.CorrelacionRef))
	return []byte(salida.String())
}

func datosFusionCanonicos(plantilla domain.PlantillaDocumento, orden OrdenGenerarDocumento) []byte {
	claves := make([]string, 0, len(orden.Datos))
	for clave := range orden.Datos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	var salida strings.Builder
	escribirCampoCanonico(&salida, "plantilla_id", plantilla.ID)
	escribirCampoCanonico(&salida, "plantilla_version", strconv.Itoa(plantilla.Version))
	escribirCampoCanonico(&salida, "formato", string(orden.Formato))
	escribirCampoCanonico(&salida, "expediente_ref", strings.TrimSpace(orden.ExpedienteRef))
	for _, clave := range claves {
		escribirCampoCanonico(&salida, clave, orden.Datos[clave])
	}
	return []byte(salida.String())
}

func escribirCampoCanonico(destino *strings.Builder, clave, valor string) {
	destino.WriteString(strconv.Itoa(len(clave)))
	destino.WriteByte(':')
	destino.WriteString(clave)
	destino.WriteByte('=')
	destino.WriteString(strconv.Itoa(len(valor)))
	destino.WriteByte(':')
	destino.WriteString(valor)
	destino.WriteByte('\n')
}

func nombreFicheroDocumento(tipo, documentoID string, formato domain.FormatoDocumento) string {
	limpiar := func(valor string) string {
		valor = strings.ToLower(strings.TrimSpace(valor))
		var salida strings.Builder
		for _, caracter := range valor {
			switch {
			case caracter >= 'a' && caracter <= 'z', caracter >= '0' && caracter <= '9':
				salida.WriteRune(caracter)
			case caracter == '-', caracter == '_':
				salida.WriteRune(caracter)
			default:
				salida.WriteByte('-')
			}
		}
		return strings.Trim(salida.String(), "-")
	}
	return limpiar(tipo) + "-" + limpiar(documentoID) + formato.Extension()
}

func validarContextoGobiernoPlantilla(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, finalidad, motivo, correlacionRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := principal.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(perfilActivo) == "" || strings.TrimSpace(finalidad) == "" ||
		strings.TrimSpace(motivo) == "" ||
		strings.TrimSpace(correlacionRef) == "" {
		return ErrOrdenDocumentalInvalida
	}
	return nil
}

func (s *ServicioDocumental) exigirAutorizacionDocumental(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, accion string,
	recurso domain.RecursoAutorizable,
	finalidad, correlacionRef, motivo string,
) (domain.DecisionAutorizacion, error) {
	return exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj, principal, perfilActivo,
		accion, recurso, finalidad, correlacionRef, motivo, usoCamposDecisionNoAplicables)
}

func referenciaPlantilla(plantilla domain.PlantillaDocumento) string {
	return strings.TrimSpace(plantilla.ID) + ":" + strconv.Itoa(plantilla.Version)
}

func trazaGobiernoPlantilla(
	principal domain.Principal,
	perfilActivo, autorizacionRef, finalidad, accion string,
	plantilla domain.PlantillaDocumento,
	referencia, motivo, correlacionRef, huellaAnterior, huellaPosterior string,
	fecha time.Time,
) domain.AuditEntry {
	formatos := make([]string, 0, len(plantilla.Formatos))
	for _, formato := range plantilla.Formatos {
		formatos = append(formatos, string(formato))
	}
	sort.Strings(formatos)
	return domain.AuditEntry{
		ActorID:          principal.ID,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          strings.TrimSpace(finalidad),
		Action:           strings.TrimSpace(accion),
		ModuleID:         plantilla.ModuloID,
		SubjectRef:       referencia,
		ObjectVersion:    plantilla.Version,
		Reason:           strings.TrimSpace(motivo),
		Result:           "correcto",
		BeforeHash:       strings.TrimSpace(huellaAnterior),
		AfterHash:        strings.TrimSpace(huellaPosterior),
		CorrelationRef:   strings.TrimSpace(correlacionRef),
		OccurredAt:       fecha.UTC(),
		Metadata: map[string]string{
			"estado":            string(plantilla.Estado),
			"formatos":          strings.Join(formatos, ","),
			"plantilla_id":      plantilla.ID,
			"plantilla_version": strconv.Itoa(plantilla.Version),
			"tipo_documental":   plantilla.TipoDocumental,
		},
	}
}

func eventoGobiernoPlantilla(
	tipo string,
	plantilla domain.PlantillaDocumento,
	referencia, huella, actorID string,
	fecha time.Time,
) domain.Event {
	return domain.Event{
		Type:       strings.TrimSpace(tipo),
		ModuleID:   plantilla.ModuloID,
		SubjectRef: referencia,
		ActorID:    strings.TrimSpace(actorID),
		OccurredAt: fecha.UTC(),
		Payload: map[string]string{
			"estado":            string(plantilla.Estado),
			"huella_sha256":     strings.TrimSpace(huella),
			"plantilla_id":      plantilla.ID,
			"plantilla_version": strconv.Itoa(plantilla.Version),
			"tipo_documental":   plantilla.TipoDocumental,
		},
	}
}
