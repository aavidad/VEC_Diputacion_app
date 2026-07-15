package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojMutableCotejoGobiernoPrueba struct {
	ahora time.Time
}

func (r *relojMutableCotejoGobiernoPrueba) Ahora() time.Time { return r.ahora }

type autorizadorCotejoGobiernoPrueba struct {
	reloj       *relojMutableCotejoGobiernoPrueba
	error       error
	solicitudes []domain.SolicitudAutorizacion
}

func (a *autorizadorCotejoGobiernoPrueba) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.solicitudes = append(a.solicitudes, solicitud)
	if a.error != nil {
		return domain.DecisionAutorizacion{}, a.error
	}
	ahora := a.reloj.Ahora().UTC()
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            fmt.Sprintf("decision-cotejo-gobierno-%03d", len(a.solicitudes)),
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            solicitud.Principal.ID,
		PerfilActivoRef:        solicitud.PerfilActivoRef,
		Accion:                 solicitud.Accion,
		RecursoRef:             solicitud.Recurso.Referencia,
		Finalidad:              solicitud.Finalidad,
		CorrelacionRef:         solicitud.CorrelacionRef,
		AsignacionRef:          "asignacion:cotejo-gobierno:v1",
		AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef:          "rol:cotejo-gobierno:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima:         domain.AuthAssuranceHigh,
		EmitidaEn:              ahora.Add(-time.Second),
		ValidaHasta:            ahora.Add(time.Minute),
	}), nil
}

type confirmacionPoliticaCotejoGobiernoPrueba struct {
	operacion      string
	huellaAnterior string
	politica       domain.PoliticaCotejo
	auditoria      domain.AuditEntry
	evento         domain.Event
}

type repositorioPoliticasCotejoGobiernoPrueba struct {
	politica               domain.PoliticaCotejo
	errorConsulta          error
	errorConfirmacion      error
	intentosConfirmacion   int
	confirmacionesExitosas []confirmacionPoliticaCotejoGobiernoPrueba
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ObtenerPoliticaCotejo(
	_ context.Context,
	id string,
	version int,
) (domain.PoliticaCotejo, error) {
	if r.errorConsulta != nil {
		return domain.PoliticaCotejo{}, r.errorConsulta
	}
	if r.politica.ID != strings.TrimSpace(id) || r.politica.Version != version {
		return domain.PoliticaCotejo{}, ports.ErrPoliticaCotejoNoEncontrada
	}
	return r.politica, nil
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ListarVersionesPoliticaCotejo(
	_ context.Context,
	id string,
) ([]domain.PoliticaCotejo, error) {
	if r.errorConsulta != nil {
		return nil, r.errorConsulta
	}
	if r.politica.ID != strings.TrimSpace(id) {
		return nil, nil
	}
	return []domain.PoliticaCotejo{r.politica}, nil
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ConfirmarAltaBorradorPoliticaCotejo(
	_ context.Context,
	politica domain.PoliticaCotejo,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	return r.confirmar("alta", "", politica, auditoria, evento)
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ConfirmarActualizacionBorradorPoliticaCotejo(
	_ context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	return r.confirmar("actualizacion", huellaAnterior, politica, auditoria, evento)
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ConfirmarPublicacionPoliticaCotejo(
	_ context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	return r.confirmar("publicacion", huellaAnterior, politica, auditoria, evento)
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) ConfirmarRetiradaPoliticaCotejo(
	_ context.Context,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	return r.confirmar("retirada", huellaAnterior, politica, auditoria, evento)
}

func (r *repositorioPoliticasCotejoGobiernoPrueba) confirmar(
	operacion string,
	huellaAnterior string,
	politica domain.PoliticaCotejo,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	r.intentosConfirmacion++
	if r.errorConfirmacion != nil {
		return r.errorConfirmacion
	}
	r.politica = politica
	r.confirmacionesExitosas = append(r.confirmacionesExitosas, confirmacionPoliticaCotejoGobiernoPrueba{
		operacion:      operacion,
		huellaAnterior: huellaAnterior,
		politica:       politica,
		auditoria:      auditoria,
		evento:         evento,
	})
	return nil
}

// Estos puertos son obligatorios para construir el servicio, pero los casos de
// uso de gobierno no deben invocarlos. Las interfaces embebidas mantienen el
// doble deliberadamente estrecho y cualquier uso accidental provocaria fallo.
type puertosNoUsadosCotejoGobiernoPrueba struct {
	ports.RepositorioCodigosCotejo
	ports.RepositorioDocumentosLogicos
	ports.GeneradorValorCodigoCotejo
	ports.GeneradorIDCodigoCotejo
	ports.SelladorIndiceCodigoCotejo
	ports.SelladorSolicitudCotejo
	ports.ProtectorCodigoCotejo
	ports.FuenteEvidenciaEmisionDocumento
}

type dependenciasCotejoGobiernoPrueba struct {
	politicas         ports.CatalogoPoliticasCotejo
	gobiernoPoliticas ports.RepositorioGobiernoPoliticasCotejo
	codigos           ports.RepositorioCodigosCotejo
	documentos        ports.RepositorioDocumentosLogicos
	autorizador       ports.Autorizador
	generadorValor    ports.GeneradorValorCodigoCotejo
	generadorID       ports.GeneradorIDCodigoCotejo
	selladorIndice    ports.SelladorIndiceCodigoCotejo
	selladorSolicitud ports.SelladorSolicitudCotejo
	protector         ports.ProtectorCodigoCotejo
	evidenciasEmision ports.FuenteEvidenciaEmisionDocumento
	reloj             ports.Reloj
}

func nuevasDependenciasCotejoGobiernoPrueba() (
	dependenciasCotejoGobiernoPrueba,
	*repositorioPoliticasCotejoGobiernoPrueba,
	*autorizadorCotejoGobiernoPrueba,
	*relojMutableCotejoGobiernoPrueba,
) {
	reloj := &relojMutableCotejoGobiernoPrueba{
		ahora: time.Date(2026, time.July, 14, 9, 0, 0, 0, time.UTC),
	}
	repositorio := &repositorioPoliticasCotejoGobiernoPrueba{}
	autorizador := &autorizadorCotejoGobiernoPrueba{reloj: reloj}
	noUsados := &puertosNoUsadosCotejoGobiernoPrueba{}
	return dependenciasCotejoGobiernoPrueba{
		politicas:         repositorio,
		gobiernoPoliticas: repositorio,
		codigos:           noUsados,
		documentos:        noUsados,
		autorizador:       autorizador,
		generadorValor:    noUsados,
		generadorID:       noUsados,
		selladorIndice:    noUsados,
		selladorSolicitud: noUsados,
		protector:         noUsados,
		evidenciasEmision: noUsados,
		reloj:             reloj,
	}, repositorio, autorizador, reloj
}

func (d dependenciasCotejoGobiernoPrueba) construir() (*ServicioCotejo, error) {
	return NuevoServicioCotejo(
		d.politicas,
		d.gobiernoPoliticas,
		d.codigos,
		d.documentos,
		d.autorizador,
		d.generadorValor,
		d.generadorID,
		d.selladorIndice,
		d.selladorSolicitud,
		d.protector,
		d.evidenciasEmision,
		d.reloj,
	)
}

func nuevoEntornoCotejoGobiernoPrueba(t *testing.T) (
	*ServicioCotejo,
	*repositorioPoliticasCotejoGobiernoPrueba,
	*autorizadorCotejoGobiernoPrueba,
	*relojMutableCotejoGobiernoPrueba,
) {
	t.Helper()
	dependencias, repositorio, autorizador, reloj := nuevasDependenciasCotejoGobiernoPrueba()
	servicio, err := dependencias.construir()
	if err != nil {
		t.Fatalf("NuevoServicioCotejo() error = %v", err)
	}
	return servicio, repositorio, autorizador, reloj
}

func principalCotejoGobiernoPrueba() domain.Principal {
	return domain.Principal{
		ID:            personaAutorizacionPrueba("responsable-rrhh-1"),
		Roles:         []string{"responsable_rrhh", "tecnico_rrhh"},
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
}

func configuracionCotejoGobiernoPrueba() ConfiguracionPoliticaCotejo {
	return ConfiguracionPoliticaCotejo{
		Nombre:                   "Cotejo de documentos de RRHH",
		Descripcion:              "Politica de verificacion de documentos emitidos",
		Modulos:                  []string{"personal", "bolsa"},
		TiposDocumentales:        []string{"resolucion", "contrato"},
		Clasificaciones:          []string{"publica", "datos_personales_alta"},
		ClaseAcceso:              domain.ClaseAccesoCotejoPublico,
		CamposPublicos:           []domain.CampoPublicoCotejo{domain.CampoPublicoCotejoOrgano, domain.CampoPublicoCotejoFechaEmision},
		PermiteDescargaDocumento: true,
		RequiereFirma:            true,
		RequiereSelloTiempo:      true,
		RequiereRegistro:         true,
		GarantiaMinima:           domain.AuthAssuranceLow,
		DiasPlazoActivacion:      7,
		DiasDisponibilidad:       365,
		FuenteRef:                "politica-seguridad-cotejo-2026",
	}
}

func ordenCrearCotejoGobiernoPrueba() OrdenCrearBorradorPoliticaCotejo {
	return OrdenCrearBorradorPoliticaCotejo{
		Principal:      principalCotejoGobiernoPrueba(),
		PerfilActivo:   perfilAutorizacionPrueba("responsable-rrhh:v1"),
		Finalidad:      "gobernar_cotejo_documental",
		ID:             "politica_documentos_rrhh",
		Version:        1,
		Configuracion:  configuracionCotejoGobiernoPrueba(),
		Motivo:         "Alta inicial de la politica",
		CorrelacionRef: "correlacion-cotejo-gobierno-001",
	}
}

func TestNuevoServicioCotejoGobiernoRechazaDependenciasNulas(t *testing.T) {
	casos := []struct {
		nombre string
		anular func(*dependenciasCotejoGobiernoPrueba)
	}{
		{"catalogo politicas", func(d *dependenciasCotejoGobiernoPrueba) { d.politicas = nil }},
		{"gobierno politicas", func(d *dependenciasCotejoGobiernoPrueba) { d.gobiernoPoliticas = nil }},
		{"repositorio codigos", func(d *dependenciasCotejoGobiernoPrueba) { d.codigos = nil }},
		{"repositorio documentos", func(d *dependenciasCotejoGobiernoPrueba) { d.documentos = nil }},
		{"autorizador", func(d *dependenciasCotejoGobiernoPrueba) { d.autorizador = nil }},
		{"generador valor", func(d *dependenciasCotejoGobiernoPrueba) { d.generadorValor = nil }},
		{"generador id", func(d *dependenciasCotejoGobiernoPrueba) { d.generadorID = nil }},
		{"sellador indice", func(d *dependenciasCotejoGobiernoPrueba) { d.selladorIndice = nil }},
		{"sellador solicitud", func(d *dependenciasCotejoGobiernoPrueba) { d.selladorSolicitud = nil }},
		{"protector", func(d *dependenciasCotejoGobiernoPrueba) { d.protector = nil }},
		{"evidencias emision", func(d *dependenciasCotejoGobiernoPrueba) { d.evidenciasEmision = nil }},
		{"reloj", func(d *dependenciasCotejoGobiernoPrueba) { d.reloj = nil }},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			dependencias, _, _, _ := nuevasDependenciasCotejoGobiernoPrueba()
			caso.anular(&dependencias)
			servicio, err := dependencias.construir()
			if servicio != nil || !errors.Is(err, ErrDependenciaCotejoRequerida) {
				t.Fatalf("NuevoServicioCotejo() = (%v, %v)", servicio, err)
			}
		})
	}
}

func TestCrearBorradorPoliticaCotejoConfirmaCanonicoConAuditoriaYEvento(t *testing.T) {
	servicio, repositorio, autorizador, reloj := nuevoEntornoCotejoGobiernoPrueba(t)
	orden := ordenCrearCotejoGobiernoPrueba()

	creada, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), orden)
	if err != nil {
		t.Fatalf("CrearBorradorPoliticaCotejo() error = %v", err)
	}
	if !reflect.DeepEqual(creada.Modulos, []string{"bolsa", "personal"}) ||
		!reflect.DeepEqual(creada.TiposDocumentales, []string{"contrato", "resolucion"}) ||
		!reflect.DeepEqual(creada.Clasificaciones, []string{"datos_personales_alta", "publica"}) ||
		!reflect.DeepEqual(creada.CamposPublicos, []domain.CampoPublicoCotejo{
			domain.CampoPublicoCotejoFechaEmision,
			domain.CampoPublicoCotejoOrgano,
		}) {
		t.Fatalf("politica no canonica: %+v", creada)
	}
	if creada.Estado != domain.EstadoPoliticaCotejoBorrador || creada.Revision != 1 ||
		creada.CreadaPor != orden.Principal.ID || !creada.CreadaEn.Equal(reloj.Ahora()) {
		t.Fatalf("alta incompleta: %+v", creada)
	}
	if len(autorizador.solicitudes) != 1 || autorizador.solicitudes[0].Accion != AccionCrearBorradorPoliticaCotejo ||
		autorizador.solicitudes[0].Recurso.Referencia != creada.Referencia() {
		t.Fatalf("autorizacion incorrecta: %+v", autorizador.solicitudes)
	}
	if len(repositorio.confirmacionesExitosas) != 1 {
		t.Fatalf("confirmaciones = %d", len(repositorio.confirmacionesExitosas))
	}
	confirmacion := repositorio.confirmacionesExitosas[0]
	huella, err := creada.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() error = %v", err)
	}
	if confirmacion.operacion != "alta" || !reflect.DeepEqual(confirmacion.politica, creada) ||
		confirmacion.auditoria.Action != eventoBorradorPoliticaCotejoCreado ||
		confirmacion.auditoria.AuthorizationRef != "decision-cotejo-gobierno-001" ||
		confirmacion.auditoria.BeforeHash != "" || confirmacion.auditoria.AfterHash != huella ||
		confirmacion.auditoria.SubjectRef != creada.Referencia() ||
		confirmacion.auditoria.Result != "correcto" || !confirmacion.auditoria.OccurredAt.Equal(reloj.Ahora()) {
		t.Fatalf("auditoria de alta incorrecta: %+v", confirmacion.auditoria)
	}
	if confirmacion.evento.Type != eventoBorradorPoliticaCotejoCreado ||
		confirmacion.evento.SubjectRef != creada.Referencia() || confirmacion.evento.ActorID != orden.Principal.ID ||
		confirmacion.evento.Payload["huella_sha256"] != huella || confirmacion.evento.Payload["estado"] != "borrador" ||
		!confirmacion.evento.OccurredAt.Equal(reloj.Ahora()) {
		t.Fatalf("evento de alta incorrecto: %+v", confirmacion.evento)
	}
}

func TestActualizarBorradorPoliticaCotejoRegistraAntesYDespues(t *testing.T) {
	servicio, repositorio, _, reloj := nuevoEntornoCotejoGobiernoPrueba(t)
	ordenAlta := ordenCrearCotejoGobiernoPrueba()
	anterior, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), ordenAlta)
	if err != nil {
		t.Fatalf("alta previa error = %v", err)
	}
	huellaAnterior, _ := anterior.HuellaSHA256()

	reloj.ahora = reloj.ahora.Add(time.Hour)
	configuracion := configuracionCotejoGobiernoPrueba()
	configuracion.Nombre = "Politica de cotejo ampliada"
	configuracion.Modulos = []string{"dietas", "bolsa"}
	configuracion.TiposDocumentales = []string{"certificado", "contrato"}
	orden := OrdenActualizarBorradorPoliticaCotejo{
		Principal:      ordenAlta.Principal,
		PerfilActivo:   ordenAlta.PerfilActivo,
		Finalidad:      ordenAlta.Finalidad,
		ID:             anterior.ID,
		Version:        anterior.Version,
		Configuracion:  configuracion,
		Motivo:         "Ampliacion de documentos admitidos",
		CorrelacionRef: "correlacion-cotejo-gobierno-002",
	}
	actualizada, err := servicio.ActualizarBorradorPoliticaCotejo(context.Background(), orden)
	if err != nil {
		t.Fatalf("ActualizarBorradorPoliticaCotejo() error = %v", err)
	}
	huellaNueva, _ := actualizada.HuellaSHA256()
	if huellaNueva == huellaAnterior || actualizada.Revision != 2 ||
		!reflect.DeepEqual(actualizada.Modulos, []string{"bolsa", "dietas"}) ||
		actualizada.ActualizadaPor != orden.Principal.ID || !actualizada.ActualizadaEn.Equal(reloj.Ahora()) {
		t.Fatalf("actualizacion incorrecta: %+v", actualizada)
	}
	if len(repositorio.confirmacionesExitosas) != 2 {
		t.Fatalf("confirmaciones = %d", len(repositorio.confirmacionesExitosas))
	}
	confirmacion := repositorio.confirmacionesExitosas[1]
	if confirmacion.operacion != "actualizacion" || confirmacion.huellaAnterior != huellaAnterior ||
		confirmacion.auditoria.Action != eventoBorradorPoliticaCotejoActualizado ||
		confirmacion.auditoria.BeforeHash != huellaAnterior || confirmacion.auditoria.AfterHash != huellaNueva ||
		confirmacion.evento.Type != eventoBorradorPoliticaCotejoActualizado ||
		confirmacion.evento.Payload["revision"] != "2" || confirmacion.evento.Payload["huella_sha256"] != huellaNueva {
		t.Fatalf("evidencia before/after incorrecta: %+v", confirmacion)
	}
}

func TestPublicarYRetirarPoliticaCotejoConfirmanTransiciones(t *testing.T) {
	servicio, repositorio, autorizador, reloj := nuevoEntornoCotejoGobiernoPrueba(t)
	ordenAlta := ordenCrearCotejoGobiernoPrueba()
	borrador, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), ordenAlta)
	if err != nil {
		t.Fatalf("alta previa error = %v", err)
	}
	huellaBorrador, _ := borrador.HuellaSHA256()

	reloj.ahora = reloj.ahora.Add(time.Hour)
	ordenPublicar := OrdenPublicarPoliticaCotejo{
		Principal:      ordenAlta.Principal,
		PerfilActivo:   ordenAlta.PerfilActivo,
		Finalidad:      ordenAlta.Finalidad,
		ID:             borrador.ID,
		Version:        borrador.Version,
		AprobacionRef:  "aprobacion-publicacion-cotejo-001",
		Motivo:         "Aprobacion y entrada en vigor",
		CorrelacionRef: "correlacion-cotejo-gobierno-003",
	}
	publicada, err := servicio.PublicarPoliticaCotejo(context.Background(), ordenPublicar)
	if err != nil {
		t.Fatalf("PublicarPoliticaCotejo() error = %v", err)
	}
	huellaPublicada, _ := publicada.HuellaSHA256()
	publicacion := repositorio.confirmacionesExitosas[1]
	if publicada.Estado != domain.EstadoPoliticaCotejoPublicada || publicada.Revision != 2 ||
		publicacion.operacion != "publicacion" || publicacion.huellaAnterior != huellaBorrador ||
		publicacion.auditoria.BeforeHash != huellaBorrador || publicacion.auditoria.AfterHash != huellaPublicada ||
		publicacion.auditoria.RuleRef != ordenPublicar.AprobacionRef ||
		publicacion.evento.Type != eventoPoliticaCotejoPublicada || publicacion.evento.ActorID != ordenAlta.Principal.ID {
		t.Fatalf("publicacion incorrecta: politica=%+v confirmacion=%+v", publicada, publicacion)
	}

	reloj.ahora = reloj.ahora.Add(time.Hour)
	ordenRetirar := OrdenRetirarPoliticaCotejo{
		Principal:      ordenAlta.Principal,
		PerfilActivo:   ordenAlta.PerfilActivo,
		Finalidad:      ordenAlta.Finalidad,
		ID:             publicada.ID,
		Version:        publicada.Version,
		AprobacionRef:  "aprobacion-retirada-cotejo-001",
		Motivo:         "Sustitucion por una nueva version",
		CorrelacionRef: "correlacion-cotejo-gobierno-004",
	}
	retirada, err := servicio.RetirarPoliticaCotejo(context.Background(), ordenRetirar)
	if err != nil {
		t.Fatalf("RetirarPoliticaCotejo() error = %v", err)
	}
	huellaRetirada, _ := retirada.HuellaSHA256()
	retiradaConfirmada := repositorio.confirmacionesExitosas[2]
	if retirada.Estado != domain.EstadoPoliticaCotejoRetirada || retirada.Revision != 3 ||
		retiradaConfirmada.operacion != "retirada" || retiradaConfirmada.huellaAnterior != huellaPublicada ||
		retiradaConfirmada.auditoria.BeforeHash != huellaPublicada || retiradaConfirmada.auditoria.AfterHash != huellaRetirada ||
		retiradaConfirmada.auditoria.RuleRef != ordenRetirar.AprobacionRef ||
		retiradaConfirmada.evento.Type != eventoPoliticaCotejoRetirada || retiradaConfirmada.evento.ActorID != ordenAlta.Principal.ID {
		t.Fatalf("retirada incorrecta: politica=%+v confirmacion=%+v", retirada, retiradaConfirmada)
	}
	if len(autorizador.solicitudes) != 3 ||
		autorizador.solicitudes[1].Accion != AccionPublicarPoliticaCotejo ||
		autorizador.solicitudes[2].Accion != AccionRetirarPoliticaCotejo {
		t.Fatalf("acciones autorizadas incorrectas: %+v", autorizador.solicitudes)
	}
}

func TestGobiernoPoliticaCotejoExigeGarantiaAlta(t *testing.T) {
	servicio, repositorio, autorizador, _ := nuevoEntornoCotejoGobiernoPrueba(t)
	orden := ordenCrearCotejoGobiernoPrueba()
	orden.Principal.AuthAssurance = domain.AuthAssuranceSubstantial

	resultado, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), orden)
	if !errors.Is(err, domain.ErrGarantiaInsuficiente) || !reflect.DeepEqual(resultado, domain.PoliticaCotejo{}) {
		t.Fatalf("CrearBorradorPoliticaCotejo() = (%+v, %v)", resultado, err)
	}
	if len(autorizador.solicitudes) != 0 || repositorio.intentosConfirmacion != 0 {
		t.Fatalf("hubo efectos sin garantia alta: autorizaciones=%d confirmaciones=%d", len(autorizador.solicitudes), repositorio.intentosConfirmacion)
	}
}

func TestGobiernoPoliticaCotejoNoPersisteConAutorizacionDenegada(t *testing.T) {
	servicio, repositorio, autorizador, _ := nuevoEntornoCotejoGobiernoPrueba(t)
	autorizador.error = domain.ErrAutorizacionDenegada

	resultado, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), ordenCrearCotejoGobiernoPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !reflect.DeepEqual(resultado, domain.PoliticaCotejo{}) {
		t.Fatalf("CrearBorradorPoliticaCotejo() = (%+v, %v)", resultado, err)
	}
	if len(autorizador.solicitudes) != 1 || repositorio.intentosConfirmacion != 0 {
		t.Fatalf("efectos tras denegacion: autorizaciones=%d confirmaciones=%d", len(autorizador.solicitudes), repositorio.intentosConfirmacion)
	}
}

func TestGobiernoPoliticaCotejoNoDevuelveExitoSiFallaRepositorio(t *testing.T) {
	servicio, repositorio, _, _ := nuevoEntornoCotejoGobiernoPrueba(t)
	falloRepositorio := errors.New("fallo atomico de gobierno de cotejo")
	repositorio.errorConfirmacion = falloRepositorio

	resultado, err := servicio.CrearBorradorPoliticaCotejo(context.Background(), ordenCrearCotejoGobiernoPrueba())
	if !errors.Is(err, falloRepositorio) || !reflect.DeepEqual(resultado, domain.PoliticaCotejo{}) {
		t.Fatalf("CrearBorradorPoliticaCotejo() = (%+v, %v)", resultado, err)
	}
	if repositorio.intentosConfirmacion != 1 || len(repositorio.confirmacionesExitosas) != 0 {
		t.Fatalf("estado de repositorio incorrecto: intentos=%d exitos=%d", repositorio.intentosConfirmacion, len(repositorio.confirmacionesExitosas))
	}
}
