package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	adaptadoralmacen "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojDocumentalFijo struct{ ahora time.Time }

func (r relojDocumentalFijo) Ahora() time.Time { return r.ahora }

type relojDocumentalActual struct{}

func (relojDocumentalActual) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type relojDocumentalMutable struct{ ahora time.Time }

func (r *relojDocumentalMutable) Ahora() time.Time { return r.ahora }

type renderizadorDocumentalQueAvanzaReloj struct {
	base  ports.RenderizadorDocumento
	reloj *relojDocumentalMutable
	salto time.Duration
}

type renderizadorDocumentalObservado struct {
	base         ports.RenderizadorDocumento
	renderizados *int
	validados    *int
}

func (r renderizadorDocumentalObservado) Formato() domain.FormatoDocumento {
	return r.base.Formato()
}

func (r renderizadorDocumentalObservado) Renderizar(
	ctx context.Context,
	contenido domain.ContenidoDocumento,
) ([]byte, error) {
	*r.renderizados++
	return r.base.Renderizar(ctx, contenido)
}

func (r renderizadorDocumentalObservado) ValidarSalida(ctx context.Context, contenido []byte) error {
	*r.validados++
	return r.base.ValidarSalida(ctx, contenido)
}

func (r renderizadorDocumentalQueAvanzaReloj) Formato() domain.FormatoDocumento {
	return r.base.Formato()
}

func (r renderizadorDocumentalQueAvanzaReloj) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error) {
	salida, err := r.base.Renderizar(ctx, contenido)
	r.reloj.ahora = r.reloj.ahora.Add(r.salto)
	return salida, err
}

func (r renderizadorDocumentalQueAvanzaReloj) ValidarSalida(ctx context.Context, contenido []byte) error {
	return r.base.ValidarSalida(ctx, contenido)
}

type idsDocumentalesSecuenciales struct{ siguiente int }

func (g *idsDocumentalesSecuenciales) NuevoIDDocumento() (string, error) {
	g.siguiente++
	return fmt.Sprintf("documento-%03d", g.siguiente), nil
}

type autorizadorDocumentalPrueba struct {
	ahora     time.Time
	siguiente int
}

type funcionAutorizadoraDocumental func(context.Context, domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error)

func (f funcionAutorizadoraDocumental) Exigir(ctx context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	decision, err := f(ctx, solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	return completarDecisionAutorizacionPrueba(solicitud, decision), nil
}

type almacenContenidoDocumentalObservado struct {
	base        ports.AlmacenContenidoDocumento
	solicitudes []ports.SolicitudGuardarContenido
}

func (a *almacenContenidoDocumentalObservado) GuardarContenido(
	ctx context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error) {
	a.solicitudes = append(a.solicitudes, solicitud)
	return a.base.GuardarContenido(ctx, solicitud)
}

func (a *almacenContenidoDocumentalObservado) LeerContenido(
	ctx context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error) {
	return a.base.LeerContenido(ctx, solicitud)
}

func (a *autorizadorDocumentalPrueba) Exigir(_ context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	if err := solicitud.Validar(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if solicitud.Principal.ID == "sin-autorizacion" {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	a.siguiente++
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            fmt.Sprintf("decision-interna-%03d", a.siguiente),
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            strings.TrimSpace(solicitud.Principal.ID),
		PerfilActivoRef:        strings.TrimSpace(solicitud.PerfilActivoRef),
		Accion:                 strings.TrimSpace(solicitud.Accion),
		RecursoRef:             strings.TrimSpace(solicitud.Recurso.Referencia),
		Finalidad:              strings.TrimSpace(solicitud.Finalidad),
		CorrelacionRef:         strings.TrimSpace(solicitud.CorrelacionRef),
		AsignacionRef:          "asignacion:prueba:v1",
		AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef:          "rol:prueba:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima:         domain.AuthAssuranceLow,
		EmitidaEn:              a.ahora,
		ValidaHasta:            a.ahora.Add(time.Minute),
	}), nil
}

func plantillaDocumentalPrueba() domain.PlantillaDocumento {
	fecha := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	return domain.PlantillaDocumento{
		ID:             "contrato_bolsa",
		Version:        7,
		ModuloID:       "bolsa",
		TipoDocumental: "contrato",
		Nombre:         "Contrato de bolsa",
		Titulo:         "Contrato {{numero}}",
		Parrafos:       []string{"Persona: {{persona}}", "Documento: {{dni}}", "Observaciones: {{observaciones}}"},
		Campos: []domain.CampoPlantillaDocumento{
			{Clave: "numero", Etiqueta: "Numero", Obligatorio: true},
			{Clave: "persona", Etiqueta: "Persona", Obligatorio: true, Sensible: true},
			{Clave: "dni", Etiqueta: "DNI", Obligatorio: true, Sensible: true},
			{Clave: "observaciones", Etiqueta: "Observaciones"},
		},
		Formatos:          []domain.FormatoDocumento{domain.FormatoDocumentoDOCX, domain.FormatoDocumentoPDF},
		PermisoGenerar:    "bolsa.documentos.generar",
		GarantiaMinima:    domain.AuthAssuranceSubstantial,
		Estado:            domain.EstadoPlantillaPublicada,
		CreadaPor:         personaAutorizacionPrueba("rrhh-creador"),
		CreadaEn:          fecha.Add(-time.Hour),
		PublicadaPor:      personaAutorizacionPrueba("rrhh-publicador"),
		PublicadaEn:       fecha,
		AprobacionRef:     "aprobacion-plantilla-7",
		MotivoPublicacion: "Validada por Seleccion Externa",
	}
}

func nuevoServicioDocumentalPrueba(t *testing.T) (*ServicioDocumental, *memory.Store) {
	t.Helper()
	store := memory.NewStore()
	sellador, err := seguridad.NuevoSelladorHMAC("prueba-1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("NuevoSelladorHMAC() error = %v", err)
	}
	selladorSolicitud, err := seguridad.NuevoSelladorHMAC("idempotencia-prueba-1", []byte("abcdef0123456789abcdef0123456789"))
	if err != nil {
		t.Fatalf("NuevoSelladorHMAC(idempotencia) error = %v", err)
	}
	seudonimizador, err := seguridad.NuevoSelladorHMAC("seudonimo-almacen-prueba-1", []byte("fedcba9876543210fedcba9876543210"))
	if err != nil {
		t.Fatalf("NuevoSelladorHMAC(seudonimizacion) error = %v", err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	almacenObjetos, err := memory.NuevoAlmacenObjetosMemoria(
		"memoria-documental-pruebas", 32*1024*1024, relojDocumentalActual{},
	)
	if err != nil {
		t.Fatalf("NuevoAlmacenObjetosMemoria() error = %v", err)
	}
	almacenDocumental, err := adaptadoralmacen.NuevoContenidoDocumental(
		context.Background(), almacenObjetos, ports.RequisitosAlmacenObjetos{
			EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
			IntegridadSHA256: true, PreservaObjetoOriginal: true,
		},
	)
	if err != nil {
		t.Fatalf("NuevoContenidoDocumental() error = %v", err)
	}
	repositorioDocumental := nuevoRepositorioDocumentosEstrictoPrueba()
	servicio, err := NuevoServicioDocumental(
		store,
		store,
		&autorizadorDocumentalPrueba{ahora: ahora},
		almacenDocumental,
		nuevoRegistroEfectosDocumentalesPrueba(),
		repositorioDocumental,
		store,
		sellador,
		selladorSolicitud,
		seudonimizador,
		&idsDocumentalesSecuenciales{},
		relojDocumentalFijo{ahora: ahora},
		OpcionesServicioDocumental{OrganoENI: "ORGANO-PRUEBA"},
		docx.Renderizador{},
		pdf.Renderizador{},
	)
	if err != nil {
		t.Fatalf("NuevoServicioDocumental() error = %v", err)
	}
	base := plantillaDocumentalPrueba()
	creador := domain.Principal{
		ID:            base.CreadaPor,
		Roles:         []string{"rol_declarado_no_autoritativo"},
		Permissions:   []string{"permiso_declarado_no_autoritativo"},
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
	borrador, err := servicio.CrearBorradorPlantilla(context.Background(), OrdenCrearBorradorPlantilla{
		Principal:      creador,
		PerfilActivo:   perfilAutorizacionPrueba("responsable_rrhh"),
		Finalidad:      "gobierno_catalogo_documental",
		ID:             base.ID,
		Version:        base.Version,
		ModuloID:       base.ModuloID,
		TipoDocumental: base.TipoDocumental,
		Nombre:         base.Nombre,
		Titulo:         base.Titulo,
		Parrafos:       base.Parrafos,
		Campos:         base.Campos,
		Formatos:       base.Formatos,
		PermisoGenerar: base.PermisoGenerar,
		GarantiaMinima: base.GarantiaMinima,
		Motivo:         "Preparacion de plantilla de prueba",
		CorrelacionRef: "corr-semilla-plantilla-alta",
	})
	if err != nil {
		t.Fatalf("CrearBorradorPlantilla(semilla) error = %v", err)
	}
	publicador := creador
	publicador.ID = base.PublicadaPor
	_, err = servicio.PublicarPlantilla(context.Background(), OrdenPublicarPlantilla{
		Principal:      publicador,
		PerfilActivo:   perfilAutorizacionPrueba("responsable_rrhh"),
		Finalidad:      "gobierno_catalogo_documental",
		PlantillaID:    borrador.ID,
		Version:        borrador.Version,
		AprobacionRef:  base.AprobacionRef,
		Motivo:         base.MotivoPublicacion,
		CorrelacionRef: "corr-semilla-plantilla-publicar",
	})
	if err != nil {
		t.Fatalf("PublicarPlantilla(semilla) error = %v", err)
	}
	return servicio, store
}

func ordenDocumentalPrueba(formato domain.FormatoDocumento) OrdenGenerarDocumento {
	return OrdenGenerarDocumento{
		Principal: domain.Principal{
			ID:            personaAutorizacionPrueba("tecnico-rrhh-1"),
			Roles:         []string{"tecnico_rrhh"},
			Permissions:   []string{"bolsa.documentos.generar"},
			AuthMethod:    domain.AuthMethodCertificate,
			AuthAssurance: domain.AuthAssuranceHigh,
		},
		PerfilActivo:     perfilAutorizacionPrueba("tecnico_rrhh"),
		Finalidad:        "gestion_contratacion_temporal",
		Clasificacion:    "datos_personales_alta",
		PlantillaID:      "contrato_bolsa",
		PlantillaVersion: 7,
		Formato:          formato,
		ExpedienteRef:    "EXP-BOLSA-2026-0042",
		Datos: map[string]string{
			"numero":        "CT-2026-42",
			"persona":       "Maria Nunez",
			"dni":           "00000000T",
			"observaciones": "Incorporacion prevista",
		},
		Motivo:         "Generacion de contrato tras llamamiento aceptado",
		CorrelacionRef: "corr-documento-0001",
	}
}

func contextoLecturaDocumentoPrueba(
	t *testing.T,
	documento domain.DocumentoGenerado,
	sufijo string,
) ports.ContextoOperacionAlmacen {
	t.Helper()
	contexto, err := pruebasvec.NuevoContextoAlmacen(
		time.Now().UTC(), "lectura-documento-"+sufijo, ports.AccionAlmacenLeer,
		ports.ReferenciaObjetoAlmacen{Referencia: documento.ReferenciaContenido, Version: "1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return contexto
}

func TestNuevoServicioDocumentalExigeRegistroDuraderoDeEfectos(t *testing.T) {
	base, _ := nuevoServicioDocumentalPrueba(t)
	var nuloTipado *registroEfectosDocumentalesPrueba
	for nombre, registro := range map[string]ports.RegistroEfectosGeneracionDocumental{
		"nulo": nil, "nulo_tipado": nuloTipado,
	} {
		t.Run(nombre, func(t *testing.T) {
			servicio, err := NuevoServicioDocumental(
				base.catalogo, base.gobierno, base.autorizador, base.almacen, registro,
				base.repositorio, base.repositorioLogico, base.selladorDatos, base.selladorSolicitud,
				base.seudonimizador, base.generadorID, base.reloj,
				OpcionesServicioDocumental{OrganoENI: base.organoENI, LimiteBytes: base.limiteBytes},
				base.renderizadores[domain.FormatoDocumentoDOCX],
				base.renderizadores[domain.FormatoDocumentoPDF],
			)
			if servicio != nil || !errors.Is(err, ErrDependenciaDocumentalRequerida) {
				t.Fatalf("registro durable ausente aceptado: servicio=%v error=%v", servicio, err)
			}
		})
	}
}

func TestServicioDocumentalGeneraDOCXYConfirmaTrazabilidad(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	documento, err := servicio.Generar(context.Background(), ordenDocumentalPrueba(domain.FormatoDocumentoDOCX))
	if err != nil {
		t.Fatalf("Generar() error = %v", err)
	}
	if documento.ID != "documento-001" || documento.Estado != domain.EstadoDocumentoBorrador ||
		documento.PlantillaVersion != 7 || documento.Formato != domain.FormatoDocumentoDOCX ||
		!strings.HasPrefix(documento.HuellaDatosHMAC, "hmac-sha256:prueba-1:") {
		t.Fatalf("documento inesperado: %+v", documento)
	}
	lectura, err := servicio.almacen.LeerContenido(context.Background(), ports.SolicitudLeerContenido{
		Contexto:   contextoLecturaDocumentoPrueba(t, documento, "001"),
		Referencia: documento.ReferenciaContenido,
		Zona:       ports.ZonaAlmacenAdmitida,
		Limite:     4 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("LeerContenido() error = %v", err)
	}
	contenido := lectura.Contenido
	if !bytes.HasPrefix(contenido, []byte("PK")) {
		t.Fatalf("el contenido no es un contenedor DOCX: %q", contenido[:min(8, len(contenido))])
	}

	repositorio := servicio.repositorio.(*repositorioDocumentosEstrictoPrueba)
	auditoria := repositorio.ListAudit(documento.ID)
	if len(auditoria) != 1 {
		t.Fatalf("ListAudit() = %+v", auditoria)
	}
	traza := auditoria[0]
	if traza.ActorProfile != perfilAutorizacionPrueba("tecnico_rrhh") || traza.AuthorizationRef != "decision-interna-003" ||
		traza.ExpedienteRef != documento.ExpedienteRef || traza.DocumentRef != documento.ID ||
		traza.AfterHash != documento.HuellaSHA256 || traza.Signature == "" || traza.IntegrityAlgorithm == "" {
		t.Fatalf("traza incompleta: %+v", traza)
	}
	serializada, _ := json.Marshal(traza)
	for _, datoPersonal := range []string{"Maria Nunez", "00000000T", "Incorporacion prevista"} {
		if bytes.Contains(serializada, []byte(datoPersonal)) {
			t.Fatalf("la auditoria contiene el dato personal %q: %s", datoPersonal, serializada)
		}
	}
	eventos := repositorio.ListEvents("vec.documento.generado")
	if len(eventos) != 1 || eventos[0].Payload["auditoria_ref"] != traza.ID {
		t.Fatalf("evento/outbox = %+v", eventos)
	}
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	efecto, err := registro.primerEfecto()
	if err != nil || len(efecto.pasos) != 1 ||
		efecto.pasos[0].Estado != ports.EstadoPasoEfectoDocumentalConfirmado {
		t.Fatalf("plan N=1 no confirmado de forma durable: efecto=%+v error=%v", efecto, err)
	}
}

func TestServicioDocumentalGeneraPDFReproducible(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	documento, err := servicio.Generar(context.Background(), ordenDocumentalPrueba(domain.FormatoDocumentoPDF))
	if err != nil {
		t.Fatalf("Generar(PDF) error = %v", err)
	}
	if documento.Estado != domain.EstadoDocumentoGenerado || documento.MIME != "application/pdf" ||
		!strings.HasSuffix(documento.NombreFichero, ".pdf") {
		t.Fatalf("documento PDF inesperado: %+v", documento)
	}
	lectura, err := servicio.almacen.LeerContenido(context.Background(), ports.SolicitudLeerContenido{
		Contexto:   contextoLecturaDocumentoPrueba(t, documento, "pdf-001"),
		Referencia: documento.ReferenciaContenido,
		Zona:       ports.ZonaAlmacenAdmitida,
		Limite:     4 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("LeerContenido(PDF) error = %v", err)
	}
	contenido := lectura.Contenido
	if !bytes.HasPrefix(contenido, []byte("%PDF-")) {
		t.Fatalf("contenido PDF invalido: %q", contenido[:min(8, len(contenido))])
	}
}

func TestServicioDocumentalSeudonimizaInteresadoYLigaEscrituraCompleta(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	observado := &almacenContenidoDocumentalObservado{base: servicio.almacen}
	servicio.almacen = observado
	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	orden.RepresentadoRef = "persona-interesada-00000042"
	documento, err := servicio.Generar(context.Background(), orden)
	if err != nil {
		t.Fatal(err)
	}
	if len(observado.solicitudes) != 1 {
		t.Fatalf("escrituras=%d, esperada una", len(observado.solicitudes))
	}
	contexto := observado.solicitudes[0].Contexto
	serializado := fmt.Sprintf("%#v", contexto)
	proyeccion, err := contexto.Proyeccion()
	if err != nil || contexto.ValidarParaEn(ports.AccionAlmacenEscribir, servicio.reloj.Ahora().UTC()) != nil ||
		proyeccion.CargaRef != documento.ID || proyeccion.RecursoRef != orden.ExpedienteRef ||
		proyeccion.ModuloID != documento.ModuloID ||
		!strings.HasPrefix(proyeccion.SujetoSeudonimoHMAC, "hmac-sha256:seudonimo-almacen-prueba-1:") ||
		!strings.HasPrefix(proyeccion.HuellaSolicitudHMAC, "hmac-sha256:idempotencia-prueba-1:") ||
		strings.Contains(serializado, orden.Principal.ID) || strings.Contains(serializado, orden.RepresentadoRef) {
		t.Fatalf("contexto de escritura incompleto o con identidad real: %#v", contexto)
	}
}

func TestServicioDocumentalDeniegaAntesDeGenerar(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	orden.Principal.ID = "sin-autorizacion"
	orden.Principal.Roles = []string{"administrador", "superusuario"}
	orden.Principal.Permissions = []string{"administracion.total", "bolsa.documentos.generar"}
	_, err := servicio.Generar(context.Background(), orden)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("sin autorizacion interna: error = %v", err)
	}
	orden = ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	orden.Principal.AuthAssurance = domain.AuthAssuranceLow
	_, err = servicio.Generar(context.Background(), orden)
	if !errors.Is(err, domain.ErrGarantiaInsuficiente) {
		t.Fatalf("garantia insuficiente: error = %v", err)
	}
	documentos, err := servicio.repositorio.ListarDocumentosExpediente(context.Background(), orden.ExpedienteRef)
	if err != nil || len(documentos) != 0 {
		t.Fatalf("una denegacion dejo documentos: %+v, %v", documentos, err)
	}
}

func TestServicioDocumentalNoUsaRolesNiPermisosDeclaradosComoAutoridad(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	orden.Principal.Roles = nil
	orden.Principal.Permissions = nil
	if _, err := servicio.Generar(context.Background(), orden); err != nil {
		t.Fatalf("la decision interna concedida no debe depender de roles o permisos declarados: %v", err)
	}
}

func TestServicioDocumentalRechazaDecisionInternaNoVinculadaOVencida(t *testing.T) {
	casos := []struct {
		nombre    string
		modificar func(*domain.DecisionAutorizacion)
	}{
		{
			nombre: "accion distinta",
			modificar: func(decision *domain.DecisionAutorizacion) {
				decision.Accion = "otra.accion"
			},
		},
		{
			nombre: "recurso distinto",
			modificar: func(decision *domain.DecisionAutorizacion) {
				decision.RecursoRef = "otro-expediente"
			},
		},
		{
			nombre: "decision vencida",
			modificar: func(decision *domain.DecisionAutorizacion) {
				decision.EmitidaEn = decision.EmitidaEn.Add(-2 * time.Minute)
				decision.ValidaHasta = decision.EmitidaEn.Add(time.Minute)
			},
		},
		{
			nombre: "campo no consumido",
			modificar: func(decision *domain.DecisionAutorizacion) {
				decision.CamposPermitidos = []string{"contenido"}
			},
		},
		{
			nombre: "obligacion no implementada",
			modificar: func(decision *domain.DecisionAutorizacion) {
				decision.Obligaciones = []string{"doble_control"}
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, _ := nuevoServicioDocumentalPrueba(t)
			base := &autorizadorDocumentalPrueba{ahora: servicio.reloj.Ahora().UTC()}
			servicio.autorizador = funcionAutorizadoraDocumental(func(ctx context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
				decision, err := base.Exigir(ctx, solicitud)
				caso.modificar(&decision)
				return decision, err
			})
			orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
			_, err := servicio.Generar(context.Background(), orden)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) {
				t.Fatalf("decision no fiable: error = %v", err)
			}
			documentos, err := servicio.repositorio.ListarDocumentosExpediente(context.Background(), orden.ExpedienteRef)
			if err != nil || len(documentos) != 0 {
				t.Fatalf("la decision no fiable dejo documentos: %+v, %v", documentos, err)
			}
		})
	}
}

func TestServicioDocumentalNoAlmacenaSiLaDecisionCaducaDuranteElRenderizado(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	inicio := servicio.reloj.Ahora().UTC()
	reloj := &relojDocumentalMutable{ahora: inicio}
	servicio.reloj = reloj
	base := pdf.Renderizador{}
	servicio.renderizadores[domain.FormatoDocumentoPDF] = renderizadorDocumentalQueAvanzaReloj{
		base: base, reloj: reloj, salto: 2 * time.Minute,
	}
	observado := &almacenContenidoDocumentalObservado{base: servicio.almacen}
	servicio.almacen = observado

	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	documento, err := servicio.Generar(context.Background(), orden)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, domain.ErrDecisionAutorizacionInvalida) || documento.ID != "" {
		t.Fatalf("decision caducada durante render: documento=%+v error=%v", documento, err)
	}
	if len(observado.solicitudes) != 0 {
		t.Fatalf("se almaceno contenido tras caducar la decision: %#v", observado.solicitudes)
	}
	documentos, err := servicio.repositorio.ListarDocumentosExpediente(context.Background(), orden.ExpedienteRef)
	if err != nil || len(documentos) != 0 {
		t.Fatalf("se confirmo documento tras caducar la decision: %+v, %v", documentos, err)
	}
}

func TestServicioDocumentalRenderizaYValidaTodoAntesDeConsultarPDP(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	renderizados, validados := 0, 0
	for formato, base := range servicio.renderizadores {
		servicio.renderizadores[formato] = renderizadorDocumentalObservado{
			base: base, renderizados: &renderizados, validados: &validados,
		}
	}
	base := &autorizadorDocumentalPrueba{ahora: servicio.reloj.Ahora().UTC()}
	servicio.autorizador = funcionAutorizadoraDocumental(func(
		ctx context.Context,
		solicitud domain.SolicitudAutorizacion,
	) (domain.DecisionAutorizacion, error) {
		if renderizados != 2 || validados != 2 {
			return domain.DecisionAutorizacion{}, errors.New("el PDP fue consultado antes de disponer de todos los bytes")
		}
		return base.Exigir(ctx, solicitud)
	})
	if _, err := servicio.GenerarDocumentoLogico(context.Background(), ordenDocumentoLogicoPrueba()); err != nil {
		t.Fatalf("orden seguro de renderizado/PDP: %v", err)
	}
}

func TestServicioDocumentalRevalidaCaducidadTrasReservarYAntesDelAlmacen(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	inicio := servicio.reloj.Ahora().UTC()
	reloj := &relojDocumentalMutable{ahora: inicio}
	servicio.reloj = reloj
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	registro.despuesReservar = func() { reloj.ahora = reloj.ahora.Add(2 * time.Minute) }
	observado := &almacenContenidoDocumentalObservado{base: servicio.almacen}
	servicio.almacen = observado

	_, err := servicio.Generar(context.Background(), ordenDocumentalPrueba(domain.FormatoDocumentoPDF))
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || len(observado.solicitudes) != 0 {
		t.Fatalf("caducidad entre reserva y efecto: escrituras=%d error=%v", len(observado.solicitudes), err)
	}
	if registro.reservas != 1 || registro.confirmaciones != 0 || registro.indeterminados != 0 {
		t.Fatalf("transiciones tras caducidad: reservas=%d confirmaciones=%d indeterminados=%d",
			registro.reservas, registro.confirmaciones, registro.indeterminados)
	}
}

func TestServicioDocumentalMarcaIndeterminadoUnFalloRemotoAmbiguo(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	falloRemoto := errors.New("respuesta remota perdida")
	ambiguo := &almacenContenidoAmbiguoPrueba{base: servicio.almacen, causa: falloRemoto}
	servicio.almacen = ambiguo

	_, err := servicio.Generar(context.Background(), ordenDocumentalPrueba(domain.FormatoDocumentoPDF))
	if !errors.Is(err, ports.ErrPasoGeneracionDocumentalIndeterminado) || !errors.Is(err, falloRemoto) || ambiguo.llamadas != 1 {
		t.Fatalf("resultado remoto ambiguo: llamadas=%d error=%v", ambiguo.llamadas, err)
	}
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	efecto, errEstado := registro.primerEfecto()
	if errEstado != nil || len(efecto.pasos) != 1 ||
		efecto.pasos[0].Estado != ports.EstadoPasoEfectoDocumentalIndeterminado ||
		efecto.pasos[0].IncidenteRef == "" {
		t.Fatalf("estado durable ambiguo: efecto=%+v error=%v", efecto, errEstado)
	}
	documentos, _ := servicio.repositorio.ListarDocumentosExpediente(
		context.Background(), ordenDocumentalPrueba(domain.FormatoDocumentoPDF).ExpedienteRef,
	)
	if len(documentos) != 0 {
		t.Fatalf("se confirmo agregado con efecto indeterminado: %+v", documentos)
	}
}

func TestServicioDocumentalUnaDecisionSoloPuedeReservarUnEfecto(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	base := &autorizadorDocumentalPrueba{ahora: servicio.reloj.Ahora().UTC()}
	servicio.autorizador = funcionAutorizadoraDocumental(func(
		ctx context.Context,
		solicitud domain.SolicitudAutorizacion,
	) (domain.DecisionAutorizacion, error) {
		decision, err := base.Exigir(ctx, solicitud)
		decision.DecisionRef = "decision-documental-no-reutilizable"
		return decision, err
	})
	observado := &almacenContenidoDocumentalObservado{base: servicio.almacen}
	servicio.almacen = observado
	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	if _, err := servicio.Generar(context.Background(), orden); err != nil {
		t.Fatalf("primer efecto: %v", err)
	}
	if _, err := servicio.Generar(context.Background(), orden); !errors.Is(err, ports.ErrDecisionAutorizacionConsumida) {
		t.Fatalf("replay de DecisionRef: %v", err)
	}
	if len(observado.solicitudes) != 1 {
		t.Fatalf("el replay alcanzo el almacen: %d escrituras", len(observado.solicitudes))
	}
}

func TestEjecutorDocumentalRecuperaPasosConfirmadosSinRepetirElAlmacen(t *testing.T) {
	servicio, _ := nuevoServicioDocumentalPrueba(t)
	instante := servicio.reloj.Ahora().UTC()
	baseAutorizador := &autorizadorDocumentalPrueba{ahora: instante}
	servicio.autorizador = funcionAutorizadoraDocumental(func(
		ctx context.Context,
		solicitud domain.SolicitudAutorizacion,
	) (domain.DecisionAutorizacion, error) {
		decision, err := baseAutorizador.Exigir(ctx, solicitud)
		decision.DecisionRef = "decision-documental-replay-estable"
		return decision, err
	})
	plantilla, err := servicio.catalogo.ObtenerPlantilla(context.Background(), "contrato_bolsa", 7)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenDocumentalPrueba(domain.FormatoDocumentoPDF)
	fusionado, err := plantilla.Fusionar(orden.Datos)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := servicio.renderizadores[domain.FormatoDocumentoPDF].Renderizar(context.Background(), fusionado)
	if err != nil {
		t.Fatal(err)
	}
	if err := servicio.renderizadores[domain.FormatoDocumentoPDF].ValidarSalida(context.Background(), contenido); err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(contenido)
	huellaContenido := hex.EncodeToString(suma[:])
	huellaSolicitud, err := servicio.selladorSolicitud.SellarSolicitudDocumento(
		context.Background(), solicitudDocumentoGeneradoCanonica(plantilla, orden),
	)
	if err != nil {
		t.Fatal(err)
	}
	const documentoID = "documento-replay-estable"
	solicitud := solicitudPlanGeneracionDocumental{
		principal: orden.Principal, perfilActivo: orden.PerfilActivo,
		finalidad: orden.Finalidad, motivo: orden.Motivo, correlacionRef: orden.CorrelacionRef,
		clasificacion: orden.Clasificacion, plantilla: plantilla,
		recursoBase: domain.RecursoAutorizable{
			Referencia: orden.ExpedienteRef, ModuloID: plantilla.ModuloID, Tipo: "expediente",
			Atributos: map[string]string{
				"plantilla_id": plantilla.ID, "tipo_documental": plantilla.TipoDocumental,
				"representaciones": "1",
			},
		},
		sujetoRef: orden.Principal.ID, ambitoSujetoRef: "documento-replay:" + huellaSolicitud,
		operacionRef: "operacion-documental-replay-estable", cargaRef: documentoID,
		efectoRef: "efecto-documental-replay:" + huellaSolicitud, huellaSolicitud: huellaSolicitud,
		contenidos: []contenidoGeneracionDocumental{{
			declaracion: ports.DeclaracionRepresentacionGeneracionDocumental{
				ReferenciaLogica: documentoID, ClaveIdempotencia: "contenido:" + documentoID,
				Formato: domain.FormatoDocumentoPDF, Zona: ports.ZonaAlmacenAdmitida,
				MIME: domain.FormatoDocumentoPDF.MIME(), Tamano: int64(len(contenido)),
				HuellaSHA256: huellaContenido,
			},
			contenido: contenido,
		}},
	}
	observado := &almacenContenidoDocumentalObservado{base: servicio.almacen}
	servicio.almacen = observado
	primero, err := servicio.ejecutarPlanGeneracionDocumental(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer efecto: %v", err)
	}
	segundo, err := servicio.ejecutarPlanGeneracionDocumental(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("replay durable: %v", err)
	}
	if len(observado.solicitudes) != 1 || primero.pasos[documentoID] != segundo.pasos[documentoID] ||
		segundo.pasos[documentoID].conectorID == "" {
		t.Fatalf("replay no identico: escrituras=%d primero=%+v segundo=%+v",
			len(observado.solicitudes), primero.pasos[documentoID], segundo.pasos[documentoID])
	}
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	if registro.reservas != 2 || registro.confirmaciones != 1 || registro.indeterminados != 0 {
		t.Fatalf("transiciones del replay: reservas=%d confirmaciones=%d indeterminados=%d",
			registro.reservas, registro.confirmaciones, registro.indeterminados)
	}
}

func TestGobiernoPlantillasExigeDobleControlYDejaTrazabilidad(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	creador := domain.Principal{
		ID:            personaAutorizacionPrueba("rrhh-creador-2"),
		Roles:         []string{"rol_declarado_no_autoritativo"},
		Permissions:   []string{"permiso_declarado_no_autoritativo"},
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
	ordenAlta := OrdenCrearBorradorPlantilla{
		Principal:      creador,
		PerfilActivo:   perfilAutorizacionPrueba("responsable_rrhh"),
		Finalidad:      "gobierno_catalogo_documental",
		ID:             "certificado_servicios",
		Version:        1,
		ModuloID:       "personal",
		TipoDocumental: "certificado",
		Nombre:         "Certificado de servicios prestados",
		Titulo:         "Certificado {{numero}}",
		Parrafos:       []string{"Se certifican los servicios de {{persona}}."},
		Campos: []domain.CampoPlantillaDocumento{
			{Clave: "numero", Etiqueta: "Numero", Obligatorio: true},
			{Clave: "persona", Etiqueta: "Persona", Obligatorio: true, Sensible: true},
		},
		Formatos:       []domain.FormatoDocumento{domain.FormatoDocumentoDOCX, domain.FormatoDocumentoPDF},
		PermisoGenerar: "personal.certificados.generar",
		GarantiaMinima: domain.AuthAssuranceSubstantial,
		Motivo:         "Alta de un nuevo tipo solicitado por Recursos Humanos",
		CorrelacionRef: "corr-plantilla-alta-1",
	}
	borrador, err := servicio.CrearBorradorPlantilla(context.Background(), ordenAlta)
	if err != nil {
		t.Fatalf("CrearBorradorPlantilla() error = %v", err)
	}
	if borrador.Estado != domain.EstadoPlantillaBorrador || borrador.CreadaPor != creador.ID {
		t.Fatalf("borrador inesperado: %+v", borrador)
	}

	ordenPublicacion := OrdenPublicarPlantilla{
		Principal:      creador,
		PerfilActivo:   perfilAutorizacionPrueba("responsable_rrhh"),
		Finalidad:      "gobierno_catalogo_documental",
		PlantillaID:    borrador.ID,
		Version:        borrador.Version,
		AprobacionRef:  "flujo-aprobacion-plantilla-1",
		Motivo:         "Plantilla revisada y aprobada",
		CorrelacionRef: "corr-plantilla-publicar-1",
	}
	if _, err := servicio.PublicarPlantilla(context.Background(), ordenPublicacion); !errors.Is(err, domain.ErrPlantillaDocumentoInvalida) {
		t.Fatalf("autopublicacion: error = %v", err)
	}

	publicador := creador
	publicador.ID = personaAutorizacionPrueba("rrhh-publicador-2")
	ordenPublicacion.Principal = publicador
	publicada, err := servicio.PublicarPlantilla(context.Background(), ordenPublicacion)
	if err != nil {
		t.Fatalf("PublicarPlantilla() error = %v", err)
	}
	if publicada.Estado != domain.EstadoPlantillaPublicada || publicada.PublicadaPor != publicador.ID ||
		publicada.AprobacionRef != ordenPublicacion.AprobacionRef {
		t.Fatalf("publicacion inesperada: %+v", publicada)
	}
	if _, err := servicio.PublicarPlantilla(context.Background(), ordenPublicacion); !errors.Is(err, domain.ErrPlantillaDocumentoInvalida) {
		t.Fatalf("segunda publicacion: error = %v", err)
	}

	referencia := "certificado_servicios:1"
	auditoria, err := store.ListAudit(context.Background(), referencia)
	if err != nil || len(auditoria) != 2 {
		t.Fatalf("auditoria de plantilla = %+v, %v", auditoria, err)
	}
	if auditoria[0].BeforeHash != "" || auditoria[0].AfterHash == "" ||
		auditoria[1].BeforeHash != auditoria[0].AfterHash || auditoria[1].AfterHash == "" ||
		auditoria[1].RuleRef != ordenPublicacion.AprobacionRef || auditoria[1].Signature == "" {
		t.Fatalf("trazabilidad de gobierno incompleta: %+v", auditoria)
	}
	eventos, err := store.ListEvents(context.Background(), []string{
		"vec.documentos.plantilla.borrador.creado",
		"vec.documentos.plantilla.publicada",
	})
	eventosPlantilla := make([]domain.Event, 0, 2)
	for _, evento := range eventos {
		if evento.SubjectRef == referencia {
			eventosPlantilla = append(eventosPlantilla, evento)
		}
	}
	if err != nil || len(eventosPlantilla) != 2 || eventosPlantilla[0].Payload["auditoria_ref"] == "" ||
		eventosPlantilla[1].Payload["auditoria_ref"] == "" {
		t.Fatalf("outbox de gobierno = %+v, %v", eventos, err)
	}
}
