package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestFuentesAsignacionDesarrolloAceptanSoloDestinoSinteticoInicial(t *testing.T) {
	instante := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	solicitudDestino := ports.SolicitudResolverDestinoAsignacion{
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:desarrollo:001",
		VersionExpediente: 3,
		ActorRef:          "persona:tecnica:desarrollo:001",
		UnidadRef:         unidadCoberturaContratacionTemporalDesarrollo,
		ResponsableRef:    responsableAsignacionContratacionTemporalDesarrollo,
		Instante:          instante,
	}
	destino, err := (resolutorDestinoAsignacionContratacionTemporalDesarrollo{}).
		ResolverDestinoAsignacion(context.Background(), solicitudDestino)
	if err != nil {
		t.Fatal(err)
	}
	solicitudPolitica := ports.SolicitudResolverPoliticaAsignacion{
		Operacion:         ports.OperacionRegistrarAsignacion,
		OrganizacionRef:   solicitudDestino.OrganizacionRef,
		ExpedienteRef:     solicitudDestino.ExpedienteRef,
		VersionExpediente: solicitudDestino.VersionExpediente,
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FasePrevia:   domain.ClaveFase("asignacion_unidad"),
		EstadoPrevio: domain.EstadoEnCurso,
		ActorRef:     solicitudDestino.ActorRef,
		PerfilRef:    "perfil:tecnico:desarrollo:001",
		Destino:      destino,
		Instante:     instante,
	}
	politica, err := (resolutorPoliticaAsignacionContratacionTemporalDesarrollo{}).
		ResolverPoliticaAsignacion(context.Background(), solicitudPolitica)
	if err != nil || politica.ValidarPara(solicitudPolitica, instante) != nil {
		t.Fatalf("política inicial no disponible: %#v / %v", politica, err)
	}

	solicitudDestino.ResponsableRef = solicitudDestino.ActorRef
	if _, err := (resolutorDestinoAsignacionContratacionTemporalDesarrollo{}).
		ResolverDestinoAsignacion(
			context.Background(),
			solicitudDestino,
		); err == nil {
		t.Fatal("segregación de funciones no aplicada")
	}
	solicitudPolitica.Operacion = ports.OperacionRegistrarReasignacion
	if _, err := (resolutorPoliticaAsignacionContratacionTemporalDesarrollo{}).
		ResolverPoliticaAsignacion(
			context.Background(),
			solicitudPolitica,
		); err == nil {
		t.Fatal("reasignación fuera del corte fue aceptada")
	}
}

func TestAutoridadAsignacionDesarrolloConcedeSoloSolicitudExacta(t *testing.T) {
	instante := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	instantanea, err :=
		nuevaInstantaneaAutorizacionAsignacionContratacionTemporalDesarrollo(
			"persona:tecnica:desarrollo:001",
			"perfil:tecnico:desarrollo:001",
			instante,
		)
	if err != nil || instantanea.Validar() != nil ||
		len(instantanea.VersionRol.Concesiones) != 1 ||
		instantanea.VersionRol.Concesiones[0].Accion != ports.AccionRegistrarAsignacion {
		t.Fatalf("concesión de asignación inválida: %#v / %v", instantanea, err)
	}
	expedienteRef := "expediente:ct:desarrollo:001"
	sello := func(dominio string, digito string) string {
		return "hmac-sha256:" + dominio + "/v1:" + strings.Repeat(digito, 64)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		ReferenciaMotivo: referenciaMotivoAutorizacionAsignacionDesarrollo(),
		Accion:           ports.AccionRegistrarAsignacion,
		Finalidad:        finalidadAsignacionContratacionTemporalDesarrollo,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: expedienteRef,
			ModuloID:   ports.ModuloContratacion,
			Tipo:       ports.TipoRecursoAsignacion,
			Ambitos: map[string]string{
				"organizacion_ref":   organizacionAltaContratacionTemporalDesarrollo,
				"expediente_ref":     expedienteRef,
				"fase_previa":        "asignacion_unidad",
				"estado_previo":      string(domain.EstadoEnCurso),
				"unidad_destino_ref": unidadCoberturaContratacionTemporalDesarrollo,
			},
			Atributos: map[string]string{
				ports.AtributoOperacionAsignacion:       string(ports.OperacionRegistrarAsignacion),
				ports.AtributoVersionAsignacion:         "3",
				ports.AtributoPoliticaAsignacionRef:     definicionPoliticaAsignacionDesarrollo,
				ports.AtributoPoliticaAsignacionVersion: "1",
				ports.AtributoPoliticaAsignacionHuella:  huellaAltaContratacionTemporalDesarrollo("politica-asignacion"),
				ports.AtributoEvidenciaDestinoRef:       evidenciaDestinoAsignacionDesarrollo,
				ports.AtributoEvidenciaDestinoHuella:    huellaAltaContratacionTemporalDesarrollo("evidencia-destino-asignacion"),
				ports.AtributoUnidadDestino:             unidadCoberturaContratacionTemporalDesarrollo,
				ports.AtributoResponsableDestino:        responsableAsignacionContratacionTemporalDesarrollo,
				ports.AtributoAmbitoIdempotenciaActivo:  sello(ports.DominioAmbitoIdempotenciaAsignacion, "a"),
				ports.AtributoHuellaPeticionAsignacion:  sello(ports.DominioHuellaPeticionAsignacion, "b"),
				ports.AtributoSegregacionAsignacion:     "true",
			},
		},
	}
	if !solicitudAutorizacionAsignacionContratacionTemporalDesarrolloValida(
		httpinterno.RutaAsignaciones,
		datos,
	) {
		t.Fatal("solicitud exacta denegada")
	}
	datos.Recurso.Atributos["campo_ajeno"] = "no"
	if solicitudAutorizacionAsignacionContratacionTemporalDesarrolloValida(
		httpinterno.RutaAsignaciones,
		datos,
	) {
		t.Fatal("solicitud con atributo ajeno aceptada")
	}
}
