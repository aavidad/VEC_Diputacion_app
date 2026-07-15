package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type escenarioUsoAutorizacionDocumento struct {
	store        *Store
	plantilla    domain.PlantillaDocumento
	documento    domain.DocumentoGenerado
	traza        domain.AuditEntry
	evento       domain.Event
	decision     domain.DecisionAutorizacion
	evidencia    ports.EvidenciaUsoDecisionAutorizacion
	verificadaEn time.Time
}

func TestConfirmarGeneracionConsumeAutorizacionEIdempotenciaNoDuplica(t *testing.T) {
	escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "idempotente")
	ctx := context.Background()
	if err := escenario.store.ConfirmarGeneracion(
		ctx, escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
	); err != nil {
		t.Fatalf("ConfirmarGeneracion() error = %v", err)
	}
	if err := escenario.store.ConfirmarGeneracion(
		ctx, escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
	); err != nil {
		t.Fatalf("reintento idempotente: %v", err)
	}

	guardado, err := escenario.store.ObtenerDocumento(ctx, escenario.documento.ID)
	if err != nil || guardado.ID != escenario.documento.ID || guardado.HuellaSHA256 != escenario.documento.HuellaSHA256 {
		t.Fatalf("ObtenerDocumento() = %+v, %v", guardado, err)
	}
	auditoria, err := escenario.store.ListAudit(ctx, escenario.documento.ID)
	if err != nil || len(auditoria) != 1 || auditoria[0].AuthorizationRef != escenario.decision.DecisionRef {
		t.Fatalf("auditoria = %+v, %v", auditoria, err)
	}
	eventos, err := escenario.store.ListEvents(ctx, []string{"vec.documento.generado"})
	if err != nil || len(eventos) != 1 || eventos[0].Payload["auditoria_ref"] != auditoria[0].ID {
		t.Fatalf("eventos = %+v, %v", eventos, err)
	}
	escenario.store.mu.RLock()
	usos := len(escenario.store.usosAutorizacionDoc)
	escenario.store.mu.RUnlock()
	if usos != 1 {
		t.Fatalf("usos de autorizacion = %d, esperado 1", usos)
	}
}

func TestConfirmarGeneracionDeniegaCadaDesvinculacionDeAutorizacion(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*escenarioUsoAutorizacionDocumento)
	}{
		{"accion", func(e *escenarioUsoAutorizacionDocumento) { e.decision.Accion = "bolsa.documentos.anular" }},
		{"principal", func(e *escenarioUsoAutorizacionDocumento) { e.decision.PrincipalID = "tecnico-rrhh-otro" }},
		{"perfil", func(e *escenarioUsoAutorizacionDocumento) { e.decision.PerfilActivoRef = "jefatura_rrhh" }},
		{"recurso", func(e *escenarioUsoAutorizacionDocumento) { e.decision.RecursoRef = "EXP-BOLSA-OTRO" }},
		{"modulo", func(e *escenarioUsoAutorizacionDocumento) { e.decision.ModuloID = "personal" }},
		{"tipo recurso", func(e *escenarioUsoAutorizacionDocumento) { e.decision.TipoRecurso = "documento" }},
		{"contexto recurso", func(e *escenarioUsoAutorizacionDocumento) {
			e.decision.ContextoRecursoHuellaSHA256 = strings.Repeat("9", 64)
		}},
		{"finalidad", func(e *escenarioUsoAutorizacionDocumento) { e.decision.Finalidad = "publicacion_oficial" }},
		{"correlacion", func(e *escenarioUsoAutorizacionDocumento) { e.decision.CorrelacionRef = "corr-distinta" }},
		{"permiso de plantilla", func(e *escenarioUsoAutorizacionDocumento) {
			plantilla := e.store.plantillas[clavePlantilla(e.plantilla.ID, e.plantilla.Version)]
			plantilla.PermisoGenerar = "bolsa.documentos.emitir"
			e.store.plantillas[clavePlantilla(plantilla.ID, plantilla.Version)] = plantilla
		}},
		{"comodin en permiso de plantilla", func(e *escenarioUsoAutorizacionDocumento) {
			plantilla := e.store.plantillas[clavePlantilla(e.plantilla.ID, e.plantilla.Version)]
			plantilla.PermisoGenerar = "bolsa.documentos.*"
			e.store.plantillas[clavePlantilla(plantilla.ID, plantilla.Version)] = plantilla
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioUsoAutorizacionDocumento(t, strings.ReplaceAll(caso.nombre, " ", "-"))
			caso.mutar(&escenario)
			if caso.nombre == "principal" || caso.nombre == "perfil" {
				if _, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(
					escenario.decision, escenario.verificadaEn,
				); !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) ||
					!errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
					t.Fatalf("mezcla de identidad produjo evidencia: %v", err)
				}
				comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)
				return
			}
			if caso.nombre != "permiso de plantilla" && caso.nombre != "comodin en permiso de plantilla" {
				escenario.evidencia = nuevaEvidenciaUsoAutorizacionDocumento(t, escenario.decision, escenario.verificadaEn)
			}
			err := escenario.store.ConfirmarGeneracion(
				context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
				t.Fatalf("desvinculacion aceptada o error no cerrado: %v", err)
			}
			comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)
		})
	}
}

func TestConfirmarGeneracionDeniegaEvidenciaNulaOCaducadaSinEfectoParcial(t *testing.T) {
	t.Run("nula", func(t *testing.T) {
		escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "nula")
		err := escenario.store.ConfirmarGeneracion(
			context.Background(), escenario.documento, escenario.traza, escenario.evento,
			ports.EvidenciaUsoDecisionAutorizacion{},
		)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
			!errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
			t.Fatalf("evidencia nula: %v", err)
		}
		comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)
	})

	t.Run("caducada antes del efecto", func(t *testing.T) {
		escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "caducada")
		instanteEfecto := escenario.decision.ValidaHasta
		fijarInstanteEfectoDocumento(&escenario, instanteEfecto)
		err := escenario.store.ConfirmarGeneracion(
			context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
		)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
			!errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
			t.Fatalf("evidencia caducada: %v", err)
		}
		comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)
	})
}

func TestConfirmarGeneracionNoReconstruyeManifiestoCompuestoDesdeMetadata(t *testing.T) {
	escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "manifiesto-no-tipado")
	recursoCompuesto := domain.RecursoAutorizable{
		Referencia: escenario.documento.ExpedienteRef,
		ModuloID:   escenario.documento.ModuloID,
		Tipo:       "expediente",
		Atributos: map[string]string{
			"plantilla_id":                              escenario.plantilla.ID,
			"tipo_documental":                           escenario.plantilla.TipoDocumental,
			ports.AtributoAlmacenOperacionRef:           "operacion:generacion:no-tipado",
			ports.AtributoAlmacenCargaRef:               "carga:generacion:no-tipado",
			ports.AtributoAlmacenClasificacion:          "datos_personales_alta",
			ports.AtributoAlmacenSujetoSeudonimoHMAC:    "hmac-sha256:sujeto_documental_v1:" + strings.Repeat("a", 64),
			ports.AtributoAlmacenHuellaSolicitudHMAC:    "hmac-sha256:solicitud_documental_v1:" + strings.Repeat("b", 64),
			ports.AtributoAlmacenEfectoRef:              "efecto:generacion:no-tipado",
			ports.AtributoAlmacenHuellaManifiestoSHA256: strings.Repeat("c", 64),
		},
	}
	huellaRecurso, err := recursoCompuesto.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	escenario.decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	escenario.evidencia = nuevaEvidenciaUsoAutorizacionDocumento(
		t, escenario.decision, escenario.verificadaEn,
	)
	err = escenario.store.ConfirmarGeneracion(
		context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
	)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("decision compuesta sin atestacion tipada no se cerro: %v", err)
	}
	comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)

	// Metadata es texto aportado por el llamador y no sustituye Contexto,
	// Manifiesto ni la reserva durable opacos. Incluso valores con forma valida
	// deben ignorarse como fuente de autoridad.
	escenario.traza.Metadata["almacen_manifiesto_sha256"] = strings.Repeat("c", 64)
	escenario.traza.Metadata["almacen_paso_sha256"] = strings.Repeat("d", 64)
	escenario.traza.Metadata["almacen_efecto_ref"] = "efecto:generacion:no-tipado"
	err = escenario.store.ConfirmarGeneracion(
		context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
	)
	if err == nil {
		t.Fatal("Metadata no tipada reconstruyo autoridad para un manifiesto compuesto")
	}
	comprobarSinEfectoDocumental(t, escenario.store, escenario.documento.ID)
}

func TestConfirmarGeneracionImpideReutilizarDecisionConOtroEfecto(t *testing.T) {
	escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "reutilizada")
	ctx := context.Background()
	if err := escenario.store.ConfirmarGeneracion(
		ctx, escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
	); err != nil {
		t.Fatal(err)
	}

	documentoAlterado := escenario.documento
	documentoAlterado.ReferenciaContenido = "objeto:contenido-distinto"
	err := escenario.store.ConfirmarGeneracion(
		ctx, documentoAlterado, escenario.traza, escenario.evento, escenario.evidencia,
	)
	if !errors.Is(err, ports.ErrDecisionAutorizacionConsumida) || !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("misma identidad con otro contenido: %v", err)
	}

	documentoOtro, trazaOtra, eventoOtro := otroEfectoDocumento(escenario, "documento-segundo")
	err = escenario.store.ConfirmarGeneracion(ctx, documentoOtro, trazaOtra, eventoOtro, escenario.evidencia)
	if !errors.Is(err, ports.ErrDecisionAutorizacionConsumida) || !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("otra identidad con la misma decision: %v", err)
	}
	if _, err := escenario.store.ObtenerDocumento(ctx, documentoOtro.ID); !errors.Is(err, ports.ErrDocumentoNoEncontrado) {
		t.Fatalf("el segundo efecto quedo persistido: %v", err)
	}
}

func TestConfirmarGeneracionConcurrenteSoloEscribeUnaVez(t *testing.T) {
	escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "concurrente-idempotente")
	const intentos = 48
	inicio := make(chan struct{})
	errores := make(chan error, intentos)
	var grupo sync.WaitGroup
	grupo.Add(intentos)
	for indice := 0; indice < intentos; indice++ {
		go func() {
			defer grupo.Done()
			<-inicio
			errores <- escenario.store.ConfirmarGeneracion(
				context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
			)
		}()
	}
	close(inicio)
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("reintento concurrente: %v", err)
		}
	}
	auditoria, _ := escenario.store.ListAudit(context.Background(), escenario.documento.ID)
	eventos, _ := escenario.store.ListEvents(context.Background(), []string{"vec.documento.generado"})
	if len(auditoria) != 1 || len(eventos) != 1 {
		t.Fatalf("efecto duplicado: auditoria=%d eventos=%d", len(auditoria), len(eventos))
	}
}

func TestConfirmarGeneracionConcurrenteMismaDecisionSoloAutorizaUnEfecto(t *testing.T) {
	escenario := nuevoEscenarioUsoAutorizacionDocumento(t, "concurrente-competida")
	documentoOtro, trazaOtra, eventoOtro := otroEfectoDocumento(escenario, "documento-competidor")
	inicio := make(chan struct{})
	errores := make(chan error, 2)
	var grupo sync.WaitGroup
	grupo.Add(2)
	go func() {
		defer grupo.Done()
		<-inicio
		errores <- escenario.store.ConfirmarGeneracion(
			context.Background(), escenario.documento, escenario.traza, escenario.evento, escenario.evidencia,
		)
	}()
	go func() {
		defer grupo.Done()
		<-inicio
		errores <- escenario.store.ConfirmarGeneracion(
			context.Background(), documentoOtro, trazaOtra, eventoOtro, escenario.evidencia,
		)
	}()
	close(inicio)
	grupo.Wait()
	close(errores)

	correctos, denegados := 0, 0
	for err := range errores {
		switch {
		case err == nil:
			correctos++
		case errors.Is(err, ports.ErrDecisionAutorizacionConsumida) && errors.Is(err, domain.ErrAutorizacionDenegada):
			denegados++
		default:
			t.Fatalf("resultado concurrente inesperado: %v", err)
		}
	}
	if correctos != 1 || denegados != 1 {
		t.Fatalf("correctos=%d denegados=%d", correctos, denegados)
	}
	escenario.store.mu.RLock()
	documentos := len(escenario.store.documentos)
	usos := len(escenario.store.usosAutorizacionDoc)
	auditorias := len(escenario.store.audit)
	eventos := len(escenario.store.events)
	escenario.store.mu.RUnlock()
	if documentos != 1 || usos != 1 || auditorias != 1 || eventos != 1 {
		t.Fatalf("estado no atomico: documentos=%d usos=%d auditorias=%d eventos=%d",
			documentos, usos, auditorias, eventos)
	}
}

func nuevoEscenarioUsoAutorizacionDocumento(t *testing.T, sufijo string) escenarioUsoAutorizacionDocumento {
	t.Helper()
	fecha := time.Date(2026, time.July, 15, 1, 2, 3, 456000000, time.UTC)
	plantilla := domain.PlantillaDocumento{
		ID:             "resolucion_bolsa",
		Version:        3,
		ModuloID:       "bolsa",
		TipoDocumental: "resolucion",
		Nombre:         "Resolucion de bolsa",
		Titulo:         "Resolucion {{numero}}",
		Parrafos:       []string{"Contenido de la resolucion."},
		Campos: []domain.CampoPlantillaDocumento{
			{Clave: "numero", Etiqueta: "Numero", Obligatorio: true},
		},
		Formatos:          []domain.FormatoDocumento{domain.FormatoDocumentoPDF},
		PermisoGenerar:    "bolsa.documentos.generar",
		GarantiaMinima:    domain.AuthAssuranceSubstantial,
		Estado:            domain.EstadoPlantillaPublicada,
		CreadaPor:         "tecnico-catalogo",
		CreadaEn:          fecha.Add(-2 * time.Hour),
		PublicadaPor:      "jefatura-catalogo",
		PublicadaEn:       fecha.Add(-time.Hour),
		AprobacionRef:     "aprobacion:plantilla:3",
		MotivoPublicacion: "Revision de seleccion externa",
	}
	if err := plantilla.Validar(); err != nil {
		t.Fatalf("plantilla de prueba: %v", err)
	}
	store := NewStore()
	store.plantillas[clavePlantilla(plantilla.ID, plantilla.Version)] = clonarPlantilla(plantilla)

	contenido := []byte("%PDF-1.7\ncontenido-" + sufijo + "\n%%EOF")
	suma := sha256.Sum256(contenido)
	huellaContenido := hex.EncodeToString(suma[:])
	documentoID := "documento-" + sufijo
	documento := domain.DocumentoGenerado{
		ID:                  documentoID,
		Version:             1,
		PlantillaID:         plantilla.ID,
		PlantillaVersion:    plantilla.Version,
		ModuloID:            plantilla.ModuloID,
		TipoDocumental:      plantilla.TipoDocumental,
		ExpedienteRef:       "EXP-BOLSA-2026-0042",
		Formato:             domain.FormatoDocumentoPDF,
		MIME:                domain.FormatoDocumentoPDF.MIME(),
		NombreFichero:       documentoID + ".pdf",
		Tamano:              int64(len(contenido)),
		HuellaSHA256:        huellaContenido,
		HuellaDatosHMAC:     "hmac-sha256:datos-documentales:" + strings.Repeat("d", 64),
		ReferenciaContenido: "objeto:documental:" + sufijo,
		Estado:              domain.EstadoDocumentoGenerado,
		EstadoAntivirus:     domain.EstadoAntivirusNoAplica,
		GeneradoPor:         "per_0123456789abcdefghijkl",
		GeneradoEn:          fecha,
		CorrelacionRef:      "corr-documento-" + sufijo,
		Motivo:              "Emision de resolucion revisada",
		ENI: domain.MetadatosENI{
			Identificador:     documentoID,
			Organo:            "ORGANO-DIPUTACION-GRANADA",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    plantilla.TipoDocumental,
			FechaCaptura:      fecha,
		},
	}
	if err := documento.Validar(); err != nil {
		t.Fatalf("documento de prueba: %v", err)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: documento.ExpedienteRef,
		ModuloID:   plantilla.ModuloID,
		Tipo:       "expediente",
		Atributos: map[string]string{
			"plantilla_id":    plantilla.ID,
			"tipo_documental": plantilla.TipoDocumental,
		},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	const perfilActivo = "prf_0123456789abcdefghijkl"
	_, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		fecha,
		documento.GeneradoPor,
		perfilActivo,
		domain.AuthMethodCertificate,
		domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef:                           "decision:documento:" + sufijo,
		Concedida:                             true,
		Codigo:                                "concedida",
		PrincipalID:                           documento.GeneradoPor,
		PerfilActivoRef:                       perfilActivo,
		Accion:                                plantilla.PermisoGenerar,
		RecursoRef:                            documento.ExpedienteRef,
		ModuloID:                              plantilla.ModuloID,
		TipoRecurso:                           "expediente",
		ContextoRecursoHuellaSHA256:           huellaContexto,
		Finalidad:                             "gestion_bolsa",
		CorrelacionRef:                        documento.CorrelacionRef,
		VinculoAutenticacionActor:             vinculo,
		AsignacionRef:                         "asignacion:tecnico-rrhh-1:v4",
		AsignacionHuellaSHA256:                strings.Repeat("a", 64),
		VersionRolRef:                         "rol:tecnico_rrhh:v7",
		VersionRolHuellaSHA256:                strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef:          "rol:tecnico_rrhh:v7",
		ControlVigenciaVersionRolRevision:     2,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             9,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		GarantiaMinima:                        domain.AuthAssuranceSubstantial,
		EmitidaEn:                             fecha.Add(-30 * time.Second),
		ValidaHasta:                           fecha.Add(30 * time.Second),
	}
	evidencia := nuevaEvidenciaUsoAutorizacionDocumento(t, decision, fecha)
	traza := domain.AuditEntry{
		ActorID:          documento.GeneradoPor,
		ActorProfile:     decision.PerfilActivoRef,
		ActorRoles:       []string{"informativo_tecnico_rrhh"},
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: decision.DecisionRef,
		Purpose:          decision.Finalidad,
		Action:           "vec.documento.generado",
		ModuleID:         documento.ModuloID,
		SubjectRef:       documento.ID,
		ObjectVersion:    documento.Version,
		ExpedienteRef:    documento.ExpedienteRef,
		DocumentRef:      documento.ID,
		RuleRef:          clavePlantilla(plantilla.ID, plantilla.Version),
		Reason:           documento.Motivo,
		Result:           "correcto",
		AfterHash:        documento.HuellaSHA256,
		CorrelationRef:   documento.CorrelacionRef,
		OccurredAt:       fecha,
		Metadata: map[string]string{
			"almacen_conector":      "s3-compatible-prueba",
			"almacen_evidencia_ref": "evidencia:almacen:" + sufijo,
			"formato":               string(documento.Formato),
			"huella_datos_hmac":     documento.HuellaDatosHMAC,
			"mime":                  documento.MIME,
			"plantilla_id":          documento.PlantillaID,
			"plantilla_version":     fmt.Sprint(documento.PlantillaVersion),
			"tamano":                fmt.Sprint(documento.Tamano),
		},
	}
	evento := domain.Event{
		Type:       "vec.documento.generado",
		ModuleID:   documento.ModuloID,
		SubjectRef: documento.ID,
		ActorID:    documento.GeneradoPor,
		OccurredAt: fecha,
		Payload: map[string]string{
			"documento_ref":  documento.ID,
			"expediente_ref": documento.ExpedienteRef,
			"formato":        string(documento.Formato),
			"huella_sha256":  documento.HuellaSHA256,
		},
	}
	return escenarioUsoAutorizacionDocumento{
		store: store, plantilla: plantilla, documento: documento, traza: traza, evento: evento,
		decision: decision, evidencia: evidencia, verificadaEn: fecha,
	}
}

func nuevaEvidenciaUsoAutorizacionDocumento(
	t *testing.T,
	decision domain.DecisionAutorizacion,
	instante time.Time,
) ports.EvidenciaUsoDecisionAutorizacion {
	t.Helper()
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		t.Fatalf("NuevaEvidenciaUsoDecisionAutorizacion() error = %v", err)
	}
	return evidencia
}

func fijarInstanteEfectoDocumento(e *escenarioUsoAutorizacionDocumento, instante time.Time) {
	e.documento.GeneradoEn = instante
	e.documento.ENI.FechaCaptura = instante
	e.traza.OccurredAt = instante
	e.evento.OccurredAt = instante
}

func otroEfectoDocumento(
	escenario escenarioUsoAutorizacionDocumento,
	documentoID string,
) (domain.DocumentoGenerado, domain.AuditEntry, domain.Event) {
	documento := clonarDocumento(escenario.documento)
	documento.ID = documentoID
	documento.NombreFichero = documentoID + ".pdf"
	documento.ReferenciaContenido = "objeto:documental:" + documentoID
	documento.ENI.Identificador = documentoID
	traza := cloneAuditEntry(escenario.traza)
	traza.SubjectRef = documentoID
	traza.DocumentRef = documentoID
	evento := cloneEvent(escenario.evento)
	evento.SubjectRef = documentoID
	evento.Payload["documento_ref"] = documentoID
	return documento, traza, evento
}

func comprobarSinEfectoDocumental(t *testing.T, store *Store, documentoID string) {
	t.Helper()
	if _, err := store.ObtenerDocumento(context.Background(), documentoID); !errors.Is(err, ports.ErrDocumentoNoEncontrado) {
		t.Fatalf("documento persistido pese a denegacion: %v", err)
	}
	store.mu.RLock()
	documentos := len(store.documentos)
	usos := len(store.usosAutorizacionDoc)
	auditorias := len(store.audit)
	eventos := len(store.events)
	store.mu.RUnlock()
	if documentos != 0 || usos != 0 || auditorias != 0 || eventos != 0 {
		t.Fatalf("efecto parcial: documentos=%d usos=%d auditorias=%d eventos=%d",
			documentos, usos, auditorias, eventos)
	}
}
