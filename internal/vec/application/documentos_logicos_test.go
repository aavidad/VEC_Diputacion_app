package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func ordenDocumentoLogicoPrueba() OrdenGenerarDocumentoLogico {
	return OrdenGenerarDocumentoLogico{
		Principal: domain.Principal{
			ID:            personaAutorizacionPrueba("tecnico-rrhh-1"),
			Roles:         []string{"rol_declarado_no_autoritativo"},
			Permissions:   []string{"permiso_declarado_no_autoritativo"},
			AuthMethod:    domain.AuthMethodCertificate,
			AuthAssurance: domain.AuthAssuranceHigh,
		},
		PerfilActivo:      perfilAutorizacionPrueba("tecnico_rrhh"),
		Finalidad:         "gestion_contratacion_temporal",
		Clasificacion:     "datos_personales_alta",
		ClaveIdempotencia: "generacion-contrato-2026-00000042",
		PlantillaID:       "contrato_bolsa",
		PlantillaVersion:  7,
		Relaciones: []domain.RelacionDocumento{
			{Tipo: domain.TipoRelacionPersona, Referencia: "persona-001", Rol: "interesada"},
			{Tipo: domain.TipoRelacionLlamamiento, Referencia: "llamamiento-042", Rol: "origen"},
			{Tipo: domain.TipoRelacionExpediente, Referencia: "EXP-BOLSA-2026-0042", Rol: "principal"},
		},
		Representaciones: []domain.SolicitudRepresentacionDocumento{
			{Tipo: domain.TipoRepresentacionVisualizacion, Formato: domain.FormatoDocumentoPDF},
			{Tipo: domain.TipoRepresentacionTrabajo, Formato: domain.FormatoDocumentoDOCX},
		},
		Datos: map[string]string{
			"numero":        "CT-2026-42",
			"persona":       "Maria Nunez",
			"dni":           "00000000T",
			"observaciones": "Incorporacion prevista",
		},
		Motivo:         "Generacion de contrato tras llamamiento aceptado",
		CorrelacionRef: "corr-documento-logico-0001",
	}
}

func TestServicioDocumentalGeneraUnDocumentoLogicoMultiformatoEIdempotente(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	orden := ordenDocumentoLogicoPrueba()
	resultado, err := servicio.GenerarDocumentoLogico(context.Background(), orden)
	if err != nil {
		t.Fatalf("GenerarDocumentoLogico() error = %v", err)
	}
	if resultado.Repetida || resultado.Documento.ID != "documento-001" ||
		resultado.Documento.Estado != domain.EstadoDocumentoLogicoBorrador || len(resultado.Representaciones) != 2 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.Representaciones[0].Tipo != domain.TipoRepresentacionTrabajo ||
		resultado.Representaciones[0].Formato != domain.FormatoDocumentoDOCX ||
		resultado.Representaciones[1].Tipo != domain.TipoRepresentacionVisualizacion ||
		resultado.Representaciones[1].Formato != domain.FormatoDocumentoPDF {
		t.Fatalf("representaciones no canonicas: %+v", resultado.Representaciones)
	}
	if resultado.Representaciones[0].HuellaFuenteHMAC != resultado.Documento.HuellaFuenteHMAC ||
		resultado.Representaciones[1].HuellaFuenteHMAC != resultado.Documento.HuellaFuenteHMAC ||
		resultado.Representaciones[0].HuellaContenidoSHA256 == resultado.Representaciones[1].HuellaContenidoSHA256 {
		t.Fatalf("huellas de fuente o contenido incorrectas: %+v", resultado.Representaciones)
	}

	guardado, err := store.ObtenerDocumentoLogico(context.Background(), resultado.Documento.Referencia())
	if err != nil || guardado.HuellaFuenteHMAC != resultado.Documento.HuellaFuenteHMAC {
		t.Fatalf("ObtenerDocumentoLogico() = %+v, %v", guardado, err)
	}
	representaciones, err := store.ListarRepresentacionesDocumento(context.Background(), resultado.Documento.Referencia())
	if err != nil || len(representaciones) != 2 {
		t.Fatalf("ListarRepresentacionesDocumento() = %+v, %v", representaciones, err)
	}
	referencia := resultado.Documento.ID + ":1"
	auditoria, _ := store.ListAudit(context.Background(), referencia)
	eventos, _ := store.ListEvents(context.Background(), []string{AccionDocumentoLogicoGenerado})
	if len(auditoria) != 1 || auditoria[0].AuthorizationRef != "decision-interna-003" ||
		auditoria[0].ExpedienteRef != "EXP-BOLSA-2026-0042" || len(eventos) != 1 ||
		eventos[0].Payload["auditoria_ref"] != auditoria[0].ID {
		t.Fatalf("trazabilidad logica incompleta: auditoria=%+v eventos=%+v", auditoria, eventos)
	}

	repetida, err := servicio.GenerarDocumentoLogico(context.Background(), orden)
	if err != nil || !repetida.Repetida || repetida.Documento.ID != resultado.Documento.ID {
		t.Fatalf("repeticion idempotente = %+v, %v", repetida, err)
	}
	auditoria, _ = store.ListAudit(context.Background(), referencia)
	eventos, _ = store.ListEvents(context.Background(), []string{AccionDocumentoLogicoGenerado})
	if len(auditoria) != 1 || len(eventos) != 1 {
		t.Fatalf("el reintento duplico evidencia: auditoria=%d eventos=%d", len(auditoria), len(eventos))
	}
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	efecto, err := registro.primerEfecto()
	if err != nil || len(efecto.pasos) != 2 ||
		efecto.pasos[0].Estado != ports.EstadoPasoEfectoDocumentalConfirmado ||
		efecto.pasos[1].Estado != ports.EstadoPasoEfectoDocumentalConfirmado {
		t.Fatalf("plan N=2 no confirmado de forma durable: efecto=%+v error=%v", efecto, err)
	}
}

func TestServicioDocumentalNoAceptaLaRespuestaDeOtroPasoDelMismoManifiesto(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	almacen := &almacenContenidoRepitePrimerResultadoPrueba{base: servicio.almacen}
	servicio.almacen = almacen
	orden := ordenDocumentoLogicoPrueba()

	_, err := servicio.GenerarDocumentoLogico(context.Background(), orden)
	if !errors.Is(err, ports.ErrPasoGeneracionDocumentalIndeterminado) || almacen.llamadas != 2 {
		t.Fatalf("respuesta cruzada entre pasos: llamadas=%d error=%v", almacen.llamadas, err)
	}
	registro := servicio.registroEfectos.(*registroEfectosDocumentalesPrueba)
	efecto, errEstado := registro.primerEfecto()
	if errEstado != nil || len(efecto.pasos) != 2 ||
		efecto.pasos[0].Estado != ports.EstadoPasoEfectoDocumentalConfirmado ||
		efecto.pasos[1].Estado != ports.EstadoPasoEfectoDocumentalIndeterminado {
		t.Fatalf("transiciones de pasos cruzados: efecto=%+v error=%v", efecto, errEstado)
	}
	if _, err := store.ObtenerDocumentoLogico(context.Background(), domain.ReferenciaDocumento{ID: "documento-001", Version: 1}); !errors.Is(err, ports.ErrDocumentoLogicoNoEncontrado) {
		t.Fatalf("se confirmo el agregado con un paso indeterminado: %v", err)
	}
}

func TestServicioDocumentalImpideCambiarUnaPeticionIdempotente(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	orden := ordenDocumentoLogicoPrueba()
	resultado, err := servicio.GenerarDocumentoLogico(context.Background(), orden)
	if err != nil {
		t.Fatalf("primera generacion: %v", err)
	}
	orden.Datos["numero"] = "CT-2026-99"
	if _, err := servicio.GenerarDocumentoLogico(context.Background(), orden); !errors.Is(err, ports.ErrClaveIdempotenciaReutilizada) {
		t.Fatalf("misma clave con otros datos: error = %v", err)
	}
	auditoria, _ := store.ListAudit(context.Background(), resultado.Documento.ID+":1")
	if len(auditoria) != 1 {
		t.Fatalf("la peticion conflictiva altero la auditoria: %+v", auditoria)
	}
}

func TestServicioDocumentalLogicoExigeExpedientePrincipalYAutorizacionInterna(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	orden := ordenDocumentoLogicoPrueba()
	orden.Relaciones = orden.Relaciones[:2]
	if _, err := servicio.GenerarDocumentoLogico(context.Background(), orden); !errors.Is(err, domain.ErrRequisitoRelacionDocumentoIncumplido) {
		t.Fatalf("sin expediente principal: error = %v", err)
	}
	orden = ordenDocumentoLogicoPrueba()
	orden.Principal.ID = "sin-autorizacion"
	orden.Principal.Roles = []string{"administrador", "superusuario"}
	orden.Principal.Permissions = []string{"administracion.total", "bolsa.documentos.generar"}
	if _, err := servicio.GenerarDocumentoLogico(context.Background(), orden); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("roles declarados sin autorizacion interna: error = %v", err)
	}
	documentos, err := store.ListarRepresentacionesDocumento(context.Background(), domain.ReferenciaDocumento{ID: "documento-001", Version: 1})
	if !errors.Is(err, ports.ErrDocumentoLogicoNoEncontrado) || len(documentos) != 0 {
		t.Fatalf("una denegacion dejo representaciones: %+v, %v", documentos, err)
	}
}

type renderizadorQueFallaUnaVez struct {
	base   ports.RenderizadorDocumento
	fallar bool
}

func (r *renderizadorQueFallaUnaVez) Formato() domain.FormatoDocumento { return r.base.Formato() }

func (r *renderizadorQueFallaUnaVez) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error) {
	if r.fallar {
		r.fallar = false
		return nil, errors.New("fallo de renderizado simulado")
	}
	return r.base.Renderizar(ctx, contenido)
}

func (r *renderizadorQueFallaUnaVez) ValidarSalida(ctx context.Context, contenido []byte) error {
	return r.base.ValidarSalida(ctx, contenido)
}

func TestServicioDocumentalLiberaReservaTrasFalloYPermiteReintentar(t *testing.T) {
	servicio, store := nuevoServicioDocumentalPrueba(t)
	base := servicio.renderizadores[domain.FormatoDocumentoPDF]
	servicio.renderizadores[domain.FormatoDocumentoPDF] = &renderizadorQueFallaUnaVez{base: base, fallar: true}
	orden := ordenDocumentoLogicoPrueba()
	if _, err := servicio.GenerarDocumentoLogico(context.Background(), orden); err == nil {
		t.Fatal("se esperaba el fallo de renderizado simulado")
	}
	resultado, err := servicio.GenerarDocumentoLogico(context.Background(), orden)
	if err != nil {
		t.Fatalf("reintento tras liberar reserva: %v", err)
	}
	if resultado.Documento.ID != "documento-001" || resultado.Repetida {
		t.Fatalf("resultado del reintento inesperado: %+v", resultado)
	}
	auditoria, _ := store.ListAudit(context.Background(), resultado.Documento.ID+":1")
	if len(auditoria) != 1 {
		t.Fatalf("el reintento no se confirmo una sola vez: %+v", auditoria)
	}
}
