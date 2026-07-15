package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func TestGobiernoPlantillaRechazaEvidenciaNoVinculada(t *testing.T) {
	fecha := time.Date(2026, time.July, 14, 15, 0, 0, 0, time.UTC)
	plantilla := domain.PlantillaDocumento{
		ID:             "resolucion_bolsa",
		Version:        1,
		ModuloID:       "bolsa",
		TipoDocumental: "resolucion",
		Nombre:         "Resolucion de bolsa",
		Titulo:         "Resolucion {{numero}}",
		Parrafos:       []string{"Contenido administrativo."},
		Campos: []domain.CampoPlantillaDocumento{
			{Clave: "numero", Etiqueta: "Numero", Obligatorio: true},
		},
		Formatos:       []domain.FormatoDocumento{domain.FormatoDocumentoPDF},
		PermisoGenerar: "bolsa.documentos.generar",
		GarantiaMinima: domain.AuthAssuranceHigh,
		Estado:         domain.EstadoPlantillaBorrador,
		CreadaPor:      "tecnico-rrhh-1",
		CreadaEn:       fecha,
	}
	huella, err := plantilla.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() error = %v", err)
	}
	referencia := "resolucion_bolsa:1"
	traza := domain.AuditEntry{
		ActorID:          plantilla.CreadaPor,
		ActorProfile:     "tecnico_rrhh",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-1",
		Purpose:          "gobierno_documental",
		Action:           "vec.documentos.plantilla.borrador.creado",
		ModuleID:         plantilla.ModuloID,
		SubjectRef:       referencia,
		ObjectVersion:    plantilla.Version,
		Reason:           "alta_catalogo",
		Result:           "correcto",
		AfterHash:        huella,
		CorrelationRef:   "corr-1",
		OccurredAt:       fecha,
	}
	evento := domain.Event{
		Type:       traza.Action,
		ModuleID:   plantilla.ModuloID,
		SubjectRef: referencia,
		ActorID:    "actor-distinto",
		OccurredAt: fecha,
		Payload: map[string]string{
			"plantilla_id":      plantilla.ID,
			"plantilla_version": "1",
			"estado":            string(plantilla.Estado),
			"huella_sha256":     huella,
		},
	}
	store := NewStore()
	if err := store.ConfirmarAltaBorradorPlantilla(context.Background(), plantilla, traza, evento); !errors.Is(err, domain.ErrPlantillaDocumentoInvalida) {
		t.Fatalf("ConfirmarAltaBorradorPlantilla() error = %v", err)
	}
	if _, err := store.ObtenerPlantilla(context.Background(), plantilla.ID, plantilla.Version); !errors.Is(err, ports.ErrPlantillaDocumentoNoEncontrada) {
		t.Fatalf("una evidencia invalida persistio la plantilla: %v", err)
	}
}

func TestConsultasDocumentalesNoInterpretanFiltrosVaciosComoTodos(t *testing.T) {
	store := NewStore()
	for _, moduloID := range []string{"", " ", " bolsa", "*"} {
		plantillas, err := store.ListarPlantillas(context.Background(), moduloID)
		if !errors.Is(err, domain.ErrPermissionDenied) || plantillas != nil {
			t.Fatalf("ListarPlantillas(%q) = (%#v, %v)", moduloID, plantillas, err)
		}
	}
	for _, expedienteRef := range []string{"", " ", " EXP-2026-1", "EXP-*"} {
		documentos, err := store.ListarDocumentosExpediente(context.Background(), expedienteRef)
		if !errors.Is(err, domain.ErrPermissionDenied) || documentos != nil {
			t.Fatalf("ListarDocumentosExpediente(%q) = (%#v, %v)", expedienteRef, documentos, err)
		}
	}
}

func TestRepositorioDocumentalRespetaContextoCanceladoSinEfectos(t *testing.T) {
	store := NewStore()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	if err := store.ConfirmarAltaBorradorPlantilla(
		ctx, domain.PlantillaDocumento{}, domain.AuditEntry{}, domain.Event{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("ConfirmarAltaBorradorPlantilla() error = %v", err)
	}
	if err := store.ConfirmarPublicacionPlantilla(
		ctx, "huella", domain.PlantillaDocumento{}, domain.AuditEntry{}, domain.Event{},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("ConfirmarPublicacionPlantilla() error = %v", err)
	}
	if _, err := store.ObtenerPlantilla(ctx, "plantilla", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("ObtenerPlantilla() error = %v", err)
	}
	if _, err := store.ListarPlantillas(ctx, "bolsa"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListarPlantillas() error = %v", err)
	}
	if _, err := store.LeerContenido(ctx, ports.SolicitudLeerContenido{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("LeerContenido() error = %v", err)
	}
	if _, err := store.ObtenerDocumento(ctx, "documento"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ObtenerDocumento() error = %v", err)
	}
	if _, err := store.ListarDocumentosExpediente(ctx, "expediente"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListarDocumentosExpediente() error = %v", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.plantillas) != 0 || len(store.documentos) != 0 || len(store.audit) != 0 || len(store.events) != 0 {
		t.Fatalf("el contexto cancelado produjo efectos: plantillas=%d documentos=%d auditoria=%d eventos=%d",
			len(store.plantillas), len(store.documentos), len(store.audit), len(store.events))
	}
}

func TestGuardarContenidoRevalidaContextoJustoAntesDelEfecto(t *testing.T) {
	store := NewStore()
	contenido := []byte("%PDF-1.7\ncontenido cancelado\n%%EOF")
	solicitud := solicitudContenidoDocumentalMemoriaPrueba(
		t, "documento-cancelado", "representacion:cancelada:pdf", "idempotencia:cancelada:pdf",
		domain.FormatoDocumentoPDF, ports.ZonaAlmacenAdmitida, contenido,
	)
	ctx := nuevoContextoCanceladoEnSegundaComprobacion()
	if _, err := store.GuardarContenido(ctx, solicitud); !errors.Is(err, context.Canceled) {
		t.Fatalf("GuardarContenido() error = %v", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.contenidos) != 0 || len(store.idempotenciasContenido) != 0 {
		t.Fatalf("la cancelacion produjo contenido parcial: contenidos=%d idempotencias=%d",
			len(store.contenidos), len(store.idempotenciasContenido))
	}
}

func TestGuardarContenidoExigePasoDocumentalExactoYConservaEvidenciaV1(t *testing.T) {
	store := NewStore()
	contenido := []byte("%PDF-1.7\ncontenido exacto\n%%EOF")
	solicitud := solicitudContenidoDocumentalMemoriaPrueba(
		t, "contrato-exacto", "representacion:contrato:pdf", "idempotencia:contrato:pdf",
		domain.FormatoDocumentoPDF, ports.ZonaAlmacenAdmitida, contenido,
	)
	guardado, err := store.GuardarContenido(context.Background(), solicitud)
	if err != nil || guardado.ValidarContra(solicitud) != nil ||
		guardado.ReferenciaLogica != solicitud.DocumentoID || guardado.MIME != solicitud.MIME ||
		guardado.Referencia != guardado.EvidenciaOperacion.Objeto.Referencia ||
		guardado.Version != guardado.EvidenciaOperacion.Objeto.Version ||
		guardado.EvidenciaOperacion.HuellaManifiestoSHA256 == "" ||
		guardado.EvidenciaOperacion.HuellaPasoSHA256 == "" {
		t.Fatalf("resultado documental no ligado: %+v, %v", guardado, err)
	}
	repetido, err := store.GuardarContenido(context.Background(), solicitud)
	if err != nil || repetido.ValidarContra(solicitud) != nil ||
		repetido.Referencia != guardado.Referencia || !repetido.EvidenciaOperacion.ReintentoIdempotente {
		t.Fatalf("reintento exacto no idempotente: %+v, %v", repetido, err)
	}
	conflictiva := solicitudContenidoDocumentalMemoriaPrueba(
		t, "contrato-conflictivo", "representacion:contrato:otra-capacidad",
		solicitud.ClaveIdempotencia, domain.FormatoDocumentoPDF,
		ports.ZonaAlmacenAdmitida, contenido,
	)
	if _, err := store.GuardarContenido(context.Background(), conflictiva); !errors.Is(err, ports.ErrIdempotenciaAlmacenReutilizada) {
		t.Fatalf("clave idempotente reutilizada por otro plan: %v", err)
	}
	solicitud.Contenido[0] ^= 1
	store.mu.RLock()
	objetoPersistido := store.contenidos[guardado.Referencia]
	numeroContenidos := len(store.contenidos)
	numeroIdempotencias := len(store.idempotenciasContenido)
	store.mu.RUnlock()
	if !bytes.Equal(objetoPersistido.Datos, contenido) || numeroContenidos != 1 || numeroIdempotencias != 1 {
		t.Fatalf("estado no defensivo o efecto parcial: contenidos=%d idempotencias=%d", numeroContenidos, numeroIdempotencias)
	}

	alterada := solicitud
	alterada.Contenido = append([]byte(nil), contenido...)
	alterada.DocumentoID = "representacion:contrato:otra"
	if _, err := store.GuardarContenido(context.Background(), alterada); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("referencia logica ajena al paso aceptada: %v", err)
	}
	contextoGenerico, err := pruebasvec.NuevoContextoAlmacen(
		time.Now().UTC(), "escritura-generica-documental", ports.AccionAlmacenEscribir,
		ports.ReferenciaObjetoAlmacen{},
	)
	if err != nil {
		t.Fatal(err)
	}
	generica := solicitud
	generica.Contexto = contextoGenerico
	generica.Contenido = append([]byte(nil), contenido...)
	if _, err := store.GuardarContenido(context.Background(), generica); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("contexto generico habilito la generacion: %v", err)
	}
}

func TestConfirmarGeneracionRechazaTrazaDeOtroModulo(t *testing.T) {
	fecha := time.Date(2026, time.July, 14, 16, 0, 0, 0, time.UTC)
	contenido := []byte("%PDF-1.7\n%%EOF")
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	store := NewStore()
	solicitudContenido := solicitudContenidoDocumentalMemoriaPrueba(
		t, "documento-1", "documento-1", "contenido-documento-1",
		domain.FormatoDocumentoPDF, ports.ZonaAlmacenAdmitida, contenido,
	)
	guardado, err := store.GuardarContenido(context.Background(), solicitudContenido)
	if err != nil {
		t.Fatalf("GuardarContenido() error = %v", err)
	}
	documento := domain.DocumentoGenerado{
		ID:                  "documento-1",
		Version:             1,
		PlantillaID:         "resolucion_bolsa",
		PlantillaVersion:    1,
		ModuloID:            "bolsa",
		TipoDocumental:      "resolucion",
		ExpedienteRef:       "EXP-2026-1",
		Formato:             domain.FormatoDocumentoPDF,
		MIME:                domain.FormatoDocumentoPDF.MIME(),
		NombreFichero:       "resolucion-documento-1.pdf",
		Tamano:              int64(len(contenido)),
		HuellaSHA256:        huella,
		HuellaDatosHMAC:     "hmac-sha256:clave:valor",
		ReferenciaContenido: guardado.Referencia,
		Estado:              domain.EstadoDocumentoGenerado,
		EstadoAntivirus:     domain.EstadoAntivirusNoAplica,
		GeneradoPor:         "tecnico-rrhh-1",
		GeneradoEn:          fecha,
		CorrelacionRef:      "corr-documento-1",
		Motivo:              "emitir_resolucion",
		ENI: domain.MetadatosENI{
			Identificador:     "documento-1",
			Organo:            "ORGANO-1",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    "resolucion",
			FechaCaptura:      fecha,
		},
	}
	traza := domain.AuditEntry{
		ActorID:          documento.GeneradoPor,
		ActorProfile:     "tecnico_rrhh",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-2",
		Purpose:          "gestion_bolsa",
		Action:           "vec.documento.generado",
		ModuleID:         "personal",
		SubjectRef:       documento.ID,
		ObjectVersion:    documento.Version,
		ExpedienteRef:    documento.ExpedienteRef,
		DocumentRef:      documento.ID,
		RuleRef:          "resolucion_bolsa:1",
		Reason:           documento.Motivo,
		Result:           "correcto",
		AfterHash:        documento.HuellaSHA256,
		CorrelationRef:   documento.CorrelacionRef,
		OccurredAt:       fecha,
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
	if err := store.ConfirmarGeneracion(context.Background(), documento, traza, evento, ports.EvidenciaUsoDecisionAutorizacion{}); !errors.Is(err, domain.ErrDocumentoInvalido) {
		t.Fatalf("ConfirmarGeneracion() error = %v", err)
	}
	if _, err := store.ObtenerDocumento(context.Background(), documento.ID); !errors.Is(err, ports.ErrDocumentoNoEncontrado) {
		t.Fatalf("una traza invalida persistio el documento: %v", err)
	}
}

func solicitudContenidoDocumentalMemoriaPrueba(
	t *testing.T,
	sufijo, referenciaLogica, claveIdempotencia string,
	formato domain.FormatoDocumento,
	zona ports.ZonaAlmacen,
	contenido []byte,
) ports.SolicitudGuardarContenido {
	t.Helper()
	instante := time.Now().UTC().Truncate(time.Microsecond)
	const principal = "per_0123456789abcdefghijkl"
	const perfil = "prf_0123456789abcdefghijkl"
	_, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instante, principal, perfil, domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	sumaContenido := sha256.Sum256(contenido)
	huellaContenido := hex.EncodeToString(sumaContenido[:])
	plantilla := domain.PlantillaDocumento{
		ID: "plantilla_memoria_" + sufijo, Version: 1, ModuloID: "bolsa", TipoDocumental: "documento_prueba",
		Nombre: "Plantilla documental de prueba", Titulo: "Documento {{numero}}",
		Parrafos: []string{"Contenido de prueba."},
		Campos:   []domain.CampoPlantillaDocumento{{Clave: "numero", Etiqueta: "Numero", Obligatorio: true}},
		Formatos: []domain.FormatoDocumento{formato}, PermisoGenerar: "bolsa.documentos.generar",
		GarantiaMinima: domain.AuthAssuranceSubstantial, Estado: domain.EstadoPlantillaPublicada,
		CreadaPor: "tecnico:rrhh:prueba", CreadaEn: instante.Add(-2 * time.Hour),
		PublicadaPor: "jefatura:rrhh:prueba", PublicadaEn: instante.Add(-time.Hour),
		AprobacionRef: "aprobacion:plantilla:" + sufijo, MotivoPublicacion: "prueba_contrato",
	}
	declaracion := ports.DeclaracionRepresentacionGeneracionDocumental{
		ReferenciaLogica: referenciaLogica, ClaveIdempotencia: claveIdempotencia,
		Formato: formato, Zona: zona, MIME: formato.MIME(), Tamano: int64(len(contenido)),
		HuellaSHA256: huellaContenido,
	}
	manifiesto, err := ports.NuevoManifiestoGeneracionDocumental(
		plantilla, []ports.DeclaracionRepresentacionGeneracionDocumental{declaracion},
	)
	if err != nil {
		t.Fatalf("crear manifiesto documental: %v", err)
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion-documental-" + sufijo, CargaRef: "carga-documental-" + sufijo,
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:seudonimo_documental_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_documental_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto-documental-" + sufijo,
	}
	baseRecurso := domain.RecursoAutorizable{
		Referencia: "documento-generado-" + sufijo, ModuloID: plantilla.ModuloID, Tipo: "documento_bolsa",
		Ambitos:   map[string]string{"sujeto_ref": principal},
		Atributos: map[string]string{"expediente_ref": "expediente-prueba-" + sufijo},
	}
	recurso, err := ports.VincularRecursoGeneracionDocumental(baseRecurso, manifiesto, vinculos)
	if err != nil {
		t.Fatalf("vincular manifiesto al recurso: %v", err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision-almacen-" + sufijo, Concedida: true, Codigo: "concedida",
		PrincipalID: principal, PerfilActivoRef: perfil,
		Accion: plantilla.PermisoGenerar, RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "prueba_repositorio_documental", CorrelacionRef: "correlacion-documental-" + sufijo,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion-documental-" + sufijo, AsignacionHuellaSHA256: strings.Repeat("c", 64),
		VersionRolRef: "rol-documental-v1", VersionRolHuellaSHA256: strings.Repeat("d", 64),
		ControlVigenciaVersionRolRef: "rol-documental-v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("e", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima: domain.AuthAssuranceSubstantial,
		EmitidaEn:      instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute),
	}
	contexto, err := ports.NuevoContextoGeneracionDocumentalAlmacen(
		decision, recurso, manifiesto, vinculos, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudGuardarContenido{
		Contexto: contexto, ClaveIdempotencia: claveIdempotencia,
		DocumentoID: referenciaLogica, Zona: zona, MIME: formato.MIME(),
		HuellaSHA256: huellaContenido, Tamano: int64(len(contenido)),
		Contenido: append([]byte(nil), contenido...),
	}
}
