package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	dominioAmbitoAnalisisPrueba    = "vec.contratacion-temporal.analisis.ambito-idempotencia"
	dominioSemanticaAnalisisPrueba = "vec.contratacion-temporal.analisis.huella-semantica"
)

func TestPreparadorOperacionAnalisisPostgreSQLReservaDurable(
	t *testing.T,
) {
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	solicitud := solicitudAnalisisPostgreSQLPrueba(t, expediente)
	tx := &transaccionPreparacionPrueba{
		fila: filaAnalisisPostgreSQLPrueba(
			t,
			"reservada",
			solicitud,
			expediente,
			nil,
		),
	}
	preparador, err := nuevoPreparadorOperacionAnalisisPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	preparacion, err := preparador.PrepararOperacionAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	datos, err := preparacion.DatosPara(solicitud)
	if err != nil ||
		datos.Estado != ports.PreparacionOperacionAnalisisReservada ||
		datos.ExpedienteAnterior == nil ||
		datos.AmbitoConsultaHMAC == "" ||
		datos.HuellaConsultaHMAC == "" ||
		tx.confirmaciones != 1 || !tx.configurada {
		t.Fatalf("reserva inesperada: %#v, %v", datos, err)
	}
	if !strings.Contains(tx.consulta, funcionPrepararAnalisis) ||
		strings.Contains(
			string(tx.operacion),
			"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		) ||
		!strings.Contains(string(tx.operacion), `"sellos_hmac"`) {
		t.Fatalf("operación insegura: %s", tx.operacion)
	}
	var operacion operacionPrepararAnalisisV1
	if err := json.Unmarshal(tx.operacion, &operacion); err != nil ||
		operacion.Esquema != esquemaPrepararAnalisis ||
		operacion.ExpedienteRef != expediente.Referencia {
		t.Fatalf("operación inesperada: %#v, %v", operacion, err)
	}
}

func TestPreparadorOperacionAnalisisPostgreSQLRechazaAgregadoAdulterado(
	t *testing.T,
) {
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	solicitud := solicitudAnalisisPostgreSQLPrueba(t, expediente)
	fila := filaAnalisisPostgreSQLPrueba(
		t,
		"reservada",
		solicitud,
		expediente,
		nil,
	).(filaPreparacionPrueba)
	fila.valores[1] = `{"referencia":"expediente:adulterado"}`
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorOperacionAnalisisPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = preparador.PrepararOperacionAnalisis(
		context.Background(),
		solicitud,
	)
	if !errors.Is(
		err,
		ports.ErrPersistenciaOperacionAnalisisNoDisponible,
	) || tx.confirmaciones != 0 {
		t.Fatalf("agregado adulterado aceptado: %v", err)
	}
}

func TestPreparadorOperacionAnalisisPostgreSQLConsultaReplay(
	t *testing.T,
) {
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	solicitud := solicitudAnalisisPostgreSQLPrueba(t, expediente)
	datosPreparacion, err := preparacionAnalisisPostgreSQLPrueba(
		t,
		solicitud,
		expediente,
	).DatosPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	recibo := reciboAnalisisPostgreSQLPrueba(datosPreparacion)
	contenido, err := json.Marshal(recibo)
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{valores: []any{string(contenido)}},
	}
	preparador, err := nuevoPreparadorOperacionAnalisisPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	encontrado, existe, err := preparador.ConsultarOperacionAnalisisConfirmada(
		context.Background(),
		solicitud.IdentidadConsulta,
	)
	if err != nil || !existe || encontrado != recibo ||
		tx.confirmaciones != 1 ||
		!strings.Contains(tx.consulta, funcionConsultarAnalisis) {
		t.Fatalf("replay inesperado: %#v, %t, %v", encontrado, existe, err)
	}
}

func TestPreparadorOperacionAnalisisPostgreSQLConsultaAusente(
	t *testing.T,
) {
	expediente := expedienteInicialAnalisisPostgreSQLPrueba(t)
	solicitud := solicitudAnalisisPostgreSQLPrueba(t, expediente)
	tx := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{err: pgx.ErrNoRows},
	}
	preparador, err := nuevoPreparadorOperacionAnalisisPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, existe, err := preparador.ConsultarOperacionAnalisisConfirmada(
		context.Background(),
		solicitud.IdentidadConsulta,
	)
	if err != nil || existe || tx.confirmaciones != 1 {
		t.Fatalf("ausencia inesperada: %t, %v", existe, err)
	}
}

func solicitudAnalisisPostgreSQLPrueba(
	t *testing.T,
	expediente domain.Expediente,
) ports.SolicitudPrepararOperacionAnalisis {
	t.Helper()
	datos := ports.DatosFuncionalesOperacionAnalisis{
		ModalidadClave:    "modalidad.interinidad",
		CategoriaRef:      expediente.Solicitud.CategoriaRef,
		GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
		CausaClave:        "causa.sustitucion",
		Periodo:           expediente.Solicitud.Periodo,
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRC: domain.VinculoEntradaRC{
			Referencia:   "entrada:rc-sintetica-001",
			HuellaSHA256: strings.Repeat("4", 64),
		},
	}
	sellos := sellosAnalisisPostgreSQLPrueba(t, "a", "b")
	sellosConsulta := sellosAnalisisPostgreSQLPrueba(t, "c", "d")
	identidad, err := ports.NuevaSolicitudConsultarOperacionAnalisisConfirmada(
		ports.DatosPreimagenesConsultaOperacionAnalisis{
			ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
			Operacion:         ports.OperacionRegistrarAnalisis,
			OrganizacionRef:   expediente.OrganizacionRef,
			ExpedienteRef:     expediente.Referencia,
			VersionExpediente: expediente.Version,
			ActorRef:          "persona:tecnica-rrhh-sintetica-001",
			PerfilRef:         "perfil:tecnica-rrhh-sintetica-001",
			ArtefactoRef:      "artefacto:analisis-sintetico-001",
			DatosFuncionales:  datos,
		},
		sellosConsulta,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudPrepararOperacionAnalisis{
		Operacion:             ports.OperacionRegistrarAnalisis,
		OrganizacionRef:       expediente.OrganizacionRef,
		ExpedienteRef:         expediente.Referencia,
		VersionExpediente:     expediente.Version,
		ActorRef:              "persona:tecnica-rrhh-sintetica-001",
		PerfilRef:             "perfil:tecnica-rrhh-sintetica-001",
		ArtefactoRef:          "artefacto:analisis-sintetico-001",
		ArtefactoHuellaSHA256: strings.Repeat("5", 64),
		Sellos:                sellos,
		IdentidadConsulta:     identidad,
	}
}

func sellosAnalisisPostgreSQLPrueba(
	t *testing.T,
	ambito string,
	semantica string,
) ports.SellosOperacionAnalisis {
	t.Helper()
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloHMACPrueba(
			dominioAmbitoAnalisisPrueba+"/v2",
			ambito,
		),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		selloHMACPrueba(
			dominioSemanticaAnalisisPrueba+"/v2",
			semantica,
		),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SellosOperacionAnalisis{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasSemanticasHMAC:   huellas,
	}
}

func preparacionAnalisisPostgreSQLPrueba(
	t *testing.T,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
	expediente domain.Expediente,
) ports.PreparacionOperacionAnalisis {
	t.Helper()
	ambito, huella, err := solicitud.Sellos.ParActivo()
	if err != nil {
		t.Fatal(err)
	}
	preparacion, err := ports.NuevaPreparacionOperacionAnalisis(
		solicitud,
		ports.DatosPreparacionOperacionAnalisis{
			ReservaRef:             "reserva:analisis-sintetica-001",
			ReciboRef:              "recibo:analisis-sintetica-001",
			Operacion:              solicitud.Operacion,
			OrganizacionRef:        solicitud.OrganizacionRef,
			ExpedienteRef:          solicitud.ExpedienteRef,
			VersionExpediente:      solicitud.VersionExpediente,
			ActorRef:               solicitud.ActorRef,
			PerfilRef:              solicitud.PerfilRef,
			ArtefactoRef:           solicitud.ArtefactoRef,
			ArtefactoHuellaSHA256:  solicitud.ArtefactoHuellaSHA256,
			AmbitoIdempotenciaHMAC: ambito,
			HuellaSemanticaHMAC:    huella,
			Estado:                 ports.PreparacionOperacionAnalisisReservada,
			ExpedienteAnterior:     &expediente,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return preparacion
}

func reciboAnalisisPostgreSQLPrueba(
	preparacion ports.DatosPreparacionOperacionAnalisis,
) ports.ReciboOperacionAnalisis {
	return ports.ReciboOperacionAnalisis{
		Operacion:              preparacion.Operacion,
		OrganizacionRef:        preparacion.OrganizacionRef,
		ExpedienteRef:          preparacion.ExpedienteRef,
		VersionAnterior:        preparacion.VersionExpediente,
		VersionResultante:      preparacion.VersionExpediente + 1,
		SecuenciaActuacion:     preparacion.VersionExpediente + 1,
		ArtefactoRef:           preparacion.ArtefactoRef,
		ArtefactoHuellaSHA256:  preparacion.ArtefactoHuellaSHA256,
		ReciboRef:              preparacion.ReciboRef,
		AuditoriaRef:           "auditoria:analisis-sintetica-001",
		EventoRef:              "evento:analisis-sintetica-001",
		ConsumoFuentesRef:      "consumo:analisis-sintetica-001",
		HuellaConsumoFuentes:   strings.Repeat("6", 64),
		ConcesionV3DecisionRef: "decision:analisis-sintetica-001",
		HuellaSemanticaHMAC:    preparacion.HuellaSemanticaHMAC,
		AmbitoConsultaHMAC:     preparacion.AmbitoConsultaHMAC,
		HuellaConsultaHMAC:     preparacion.HuellaConsultaHMAC,
		ConfirmadaEn: time.Date(
			2026, 7, 24, 12, 0, 0, 0, time.UTC,
		),
	}
}

func filaAnalisisPostgreSQLPrueba(
	t *testing.T,
	resultado string,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
	expediente domain.Expediente,
	recibo *ports.ReciboOperacionAnalisis,
) pgx.Row {
	t.Helper()
	contenidoExpediente := ""
	contenidoRecibo := ""
	estado := string(ports.PreparacionOperacionAnalisisReservada)
	if recibo == nil {
		bytesExpediente, err := json.Marshal(expediente)
		if err != nil {
			t.Fatal(err)
		}
		contenidoExpediente = string(bytesExpediente)
	} else {
		bytesRecibo, err := json.Marshal(recibo)
		if err != nil {
			t.Fatal(err)
		}
		contenidoRecibo = string(bytesRecibo)
		estado = string(ports.PreparacionOperacionAnalisisConfirmada)
	}
	ambito, huella, err := solicitud.Sellos.ParActivo()
	if err != nil {
		t.Fatal(err)
	}
	return filaPreparacionPrueba{valores: []any{
		resultado, contenidoExpediente, contenidoRecibo,
		"reserva:analisis-sintetica-001",
		"recibo:analisis-sintetica-001",
		string(solicitud.Operacion), solicitud.OrganizacionRef,
		solicitud.ExpedienteRef, int64(solicitud.VersionExpediente),
		solicitud.ActorRef, solicitud.PerfilRef, solicitud.ArtefactoRef,
		solicitud.ArtefactoHuellaSHA256, ambito, huella, estado,
	}}
}

func expedienteInicialAnalisisPostgreSQLPrueba(
	t *testing.T,
) domain.Expediente {
	t.Helper()
	instante := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente:analisis-sintetico-001",
		OrganizacionRef: "organizacion:sintetica-001",
		NumeroVisible:   "2026/ANA-0001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:temporal-sintetico-001",
			Version:       1,
			HuellaSHA256:  strings.Repeat("7", 64),
		},
		FaseInicial: "recepcion_sintetica",
		Solicitud: domain.SolicitudCentro{
			CentroRef:          "centro:sintetico-001",
			ContactoRef:        "contacto:sintetico-001",
			CategoriaRef:       "categoria:sintetica-001",
			GrupoSubgrupo:      "C2",
			MotivoClave:        "motivo.sintetico",
			Detalle:            "Necesidad completamente sintética.",
			Periodo:            periodo,
			RC:                 domain.DeclaracionRC{Existe: false},
			DocumentosAdjuntos: []string{"documento:sintetico-001"},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave:   "solicitud.registrada",
			ActorRef:      "persona:solicitante-sintetica-001",
			UnidadRef:     "unidad:origen-sintetica-001",
			ReciboRef:     "recibo:alta-sintetica-001",
			RealizadaEn:   instante,
			FaseDestino:   "recepcion_sintetica",
			EstadoDestino: domain.EstadoEnCurso,
			DocumentosRef: []string{"documento:sintetico-001"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}
