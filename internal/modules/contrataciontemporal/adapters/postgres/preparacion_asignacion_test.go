package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type generadorReferenciasAsignacionPrueba struct {
	referencias ports.ReferenciasEfectoAsignacion
	err         error
	llamadas    int
}

func (g *generadorReferenciasAsignacionPrueba) GenerarReferenciasAsignacion(
	_ context.Context,
) (ports.ReferenciasEfectoAsignacion, error) {
	g.llamadas++
	return g.referencias, g.err
}

func TestPreparadorAsignacionPostgreSQLReservaSinExponerClave(t *testing.T) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	solicitud := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	referencias := referenciasAsignacionPostgreSQLPrueba()
	fila := filaAsignacionPostgreSQLPrueba(
		t,
		"reservada",
		solicitud,
		expediente,
		referencias,
	)
	tx := &transaccionPreparacionPrueba{fila: fila}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	generador := &generadorReferenciasAsignacionPrueba{
		referencias: referencias,
	}
	preparador, err := nuevoPreparadorAsignacionPostgreSQL(
		iniciador,
		generador,
	)
	if err != nil {
		t.Fatal(err)
	}

	preparacion, err := preparador.PrepararAsignacion(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	if preparacion.ValidarPara(solicitud) != nil ||
		preparacion.Estado != ports.PreparacionAsignacionReservada ||
		tx.confirmaciones != 1 || !tx.configurada ||
		iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadWrite ||
		generador.llamadas != 1 {
		t.Fatalf(
			"preparación o transacción inesperada: %#v tx=%#v",
			preparacion,
			tx,
		)
	}
	if !strings.Contains(tx.consulta, funcionPrepararAsignacion) ||
		strings.Contains(
			string(tx.operacion),
			solicitud.ClaveIdempotencia,
		) ||
		!strings.Contains(string(tx.operacion), `"reserva_ref"`) ||
		strings.Contains(string(tx.operacion), `"ReservaRef"`) {
		t.Fatalf("consulta insegura o clave filtrada: %s", tx.operacion)
	}
	var operacion operacionPrepararAsignacionV1
	if err := json.Unmarshal(tx.operacion, &operacion); err != nil ||
		operacion.Esquema != esquemaPrepararAsignacion ||
		operacion.ReferenciasCandidatas.puertos() != referencias {
		t.Fatalf("operación inesperada: %#v, %v", operacion, err)
	}
}

func TestPreparadorAsignacionPostgreSQLRechazaRespuestaAdulterada(
	t *testing.T,
) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	solicitud := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	referencias := referenciasAsignacionPostgreSQLPrueba()
	fila := filaAsignacionPostgreSQLPrueba(
		t,
		"reservada",
		solicitud,
		expediente,
		referencias,
	).(filaPreparacionPrueba)
	fila.valores[4] = "notificacion:ajena:0123456789abcdef"
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador := preparadorAsignacionPostgreSQLPrueba(
		t,
		&iniciadorPreparacionPrueba{tx: tx},
		referencias,
	)

	_, err := preparador.PrepararAsignacion(
		context.Background(),
		solicitud,
	)
	if !errors.Is(err, ports.ErrPersistenciaAsignacionNoDisponible) ||
		tx.confirmaciones != 0 {
		t.Fatalf("respuesta adulterada aceptada: %v", err)
	}
}

func TestPreparadorAsignacionPostgreSQLDevuelveReplayConfirmado(
	t *testing.T,
) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	solicitud := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	referencias := referenciasAsignacionPostgreSQLPrueba()
	tx := &transaccionPreparacionPrueba{
		fila: filaAsignacionPostgreSQLPrueba(
			t,
			"confirmada",
			solicitud,
			expediente,
			referencias,
		),
	}
	preparador := preparadorAsignacionPostgreSQLPrueba(
		t,
		&iniciadorPreparacionPrueba{tx: tx},
		referencias,
	)

	preparacion, err := preparador.PrepararAsignacion(
		context.Background(),
		solicitud,
	)
	if err != nil || preparacion.Estado !=
		ports.PreparacionAsignacionConfirmada ||
		preparacion.ReciboConfirmado == nil ||
		preparacion.ReciboConfirmado.ValidarParaPreparacion(
			preparacion,
		) != nil {
		t.Fatalf("replay confirmado inválido: %#v, %v", preparacion, err)
	}
}

func TestPreparadorAsignacionPostgreSQLReintentaSerializable(
	t *testing.T,
) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	solicitud := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	referencias := referenciasAsignacionPostgreSQLPrueba()
	primera := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{
			err: &pgconn.PgError{Code: "40001"},
		},
	}
	segunda := &transaccionPreparacionPrueba{
		fila: filaAsignacionPostgreSQLPrueba(
			t,
			"reservada",
			solicitud,
			expediente,
			referencias,
		),
	}
	iniciador := &iniciadorPreparacionPrueba{
		transacciones: []pgx.Tx{primera, segunda},
	}
	preparador := preparadorAsignacionPostgreSQLPrueba(
		t,
		iniciador,
		referencias,
	)

	if _, err := preparador.PrepararAsignacion(
		context.Background(),
		solicitud,
	); err != nil {
		t.Fatal(err)
	}
	if iniciador.inicios != 2 || segunda.confirmaciones != 1 {
		t.Fatalf("reintento inesperado: %#v", iniciador)
	}
}

func preparadorAsignacionPostgreSQLPrueba(
	t *testing.T,
	iniciador iniciadorTransacciones,
	referencias ports.ReferenciasEfectoAsignacion,
) *PreparadorAsignacionPostgreSQL {
	t.Helper()
	preparador, err := nuevoPreparadorAsignacionPostgreSQL(
		iniciador,
		&generadorReferenciasAsignacionPrueba{
			referencias: referencias,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return preparador
}

func solicitudAsignacionPostgreSQLPrueba(
	t *testing.T,
	expediente domain.Expediente,
) ports.SolicitudPrepararAsignacion {
	t.Helper()
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloHMACPrueba(
			ports.DominioAmbitoIdempotenciaAsignacion+"/v2",
			"a",
		),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		selloHMACPrueba(
			ports.DominioHuellaPeticionAsignacion+"/v2",
			"b",
		),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudPrepararAsignacion{
		ClaveIdempotencia:   "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		AmbitosHMAC:         ambitos,
		HuellasPeticionHMAC: huellas,
		Operacion:           ports.OperacionRegistrarAsignacion,
		OrganizacionRef:     expediente.OrganizacionRef,
		ExpedienteRef:       expediente.Referencia,
		VersionExpediente:   expediente.Version,
		ActorRef:            "persona:tecnica-sintetica-001",
		PerfilRef:           "perfil:tecnico-sintetico-001",
		UnidadRef:           "unidad:seleccion-sintetica-001",
		ResponsableRef:      "persona:responsable-sintetica-001",
	}
}

func referenciasAsignacionPostgreSQLPrueba() ports.ReferenciasEfectoAsignacion {
	return ports.ReferenciasEfectoAsignacion{
		ReservaRef:      "reserva:asignacion-sintetica-001",
		ReciboRef:       "recibo:asignacion-sintetica-001",
		NotificacionRef: "notificacion:asignacion-sintetica-001",
		BandejaRef:      "bandeja:asignacion-sintetica-001",
		AuditoriaRef:    "auditoria:asignacion-sintetica-001",
		EventoRef:       "evento:asignacion-sintetica-001",
	}
}

func filaAsignacionPostgreSQLPrueba(
	t *testing.T,
	resultado string,
	solicitud ports.SolicitudPrepararAsignacion,
	expediente domain.Expediente,
	referencias ports.ReferenciasEfectoAsignacion,
) pgx.Row {
	t.Helper()
	contenido, err := json.Marshal(expediente)
	if err != nil {
		t.Fatal(err)
	}
	datosAmbitos, err := solicitud.AmbitosHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosHuellas, err := solicitud.HuellasPeticionHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	version := pgtype.Int8{}
	decision := pgtype.Text{}
	confirmada := pgtype.Timestamptz{}
	estado := string(ports.PreparacionAsignacionReservada)
	if resultado == "confirmada" {
		estado = string(ports.PreparacionAsignacionConfirmada)
		version = pgtype.Int8{
			Int64: int64(expediente.Version + 1),
			Valid: true,
		}
		decision = pgtype.Text{
			String: "decision:asignacion-sintetica-001",
			Valid:  true,
		}
		confirmada = pgtype.Timestamptz{
			Time:  expediente.ActualizadoEn.Add(time.Minute),
			Valid: true,
		}
	}
	return filaPreparacionPrueba{valores: []any{
		resultado,
		string(contenido),
		referencias.ReservaRef,
		referencias.ReciboRef,
		referencias.NotificacionRef,
		referencias.BandejaRef,
		referencias.AuditoriaRef,
		referencias.EventoRef,
		datosAmbitos.Activo.Valor,
		datosHuellas.Activo.Valor,
		string(solicitud.Operacion),
		solicitud.OrganizacionRef,
		solicitud.ActorRef,
		solicitud.PerfilRef,
		solicitud.UnidadRef,
		solicitud.ResponsableRef,
		estado,
		version,
		decision,
		confirmada,
	}}
}

func expedienteAsignacionPostgreSQLPrueba(t *testing.T) domain.Expediente {
	t.Helper()
	instante := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente:temporal-sintetico-001",
		OrganizacionRef: "organizacion:sintetica-001",
		NumeroVisible:   "2026/SINT-0001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:temporal-sintetico-001",
			Version:       1,
			HuellaSHA256:  strings.Repeat("7", 64),
		},
		FaseInicial: "recepcion_sintetica",
		Solicitud: domain.SolicitudCentro{
			CentroRef:     "centro:sintetico-001",
			ContactoRef:   "contacto:sintetico-001",
			CategoriaRef:  "categoria:sintetica-001",
			GrupoSubgrupo: "C2",
			MotivoClave:   "motivo.sintetico",
			Detalle:       "Necesidad completamente sintética.",
			Periodo:       periodo,
			RC:            domain.DeclaracionRC{Existe: false},
			DocumentosAdjuntos: []string{
				"documento:sintetico-001",
			},
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
	entradaRC := domain.VinculoEntradaRC{
		Referencia:   "entrada:rc-sintetica-001",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	analisis := domain.AnalisisRRHH{
		ModalidadClave:    "modalidad.sintetica",
		CategoriaRef:      "categoria:sintetica-001",
		GrupoSubgrupo:     "C2",
		CausaClave:        "causa.sintetica",
		Periodo:           periodo,
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRCEsperada: entradaRC,
		ValidacionRC: domain.ValidacionRC{
			Resultado:           domain.RCNoRequerida,
			EntradaRef:          entradaRC.Referencia,
			HuellaEntradaSHA256: entradaRC.HuellaSHA256,
			FuenteRef:           "fuente:rc-sintetica-001",
			ReciboRef:           "recibo:rc-sintetica-001",
			ValidadaEn:          instante.Add(time.Minute),
			Motivo:              "No requiere retención de crédito.",
		},
	}
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		analisis,
		domain.DatosActuacion{
			AccionClave:   "analisis.registrado",
			ActorRef:      "persona:analista-sintetica-001",
			UnidadRef:     "unidad:rrhh-sintetica-001",
			ReciboRef:     "recibo:analisis-sintetico-001",
			RealizadaEn:   instante.Add(time.Minute),
			FaseDestino:   expediente.FaseActual,
			EstadoDestino: expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	expediente, err = expediente.RegistrarViaCobertura(
		expediente.Version,
		domain.DecisionViaCobertura{
			ViaClave:         "via.bolsa",
			ProcedimientoRef: "procedimiento:sintetico-001",
			BolsaRef:         "bolsa:sintetica-001",
			Comprobaciones: []domain.ComprobacionCobertura{{
				Clave:      "bolsa.disponible",
				Resultado:  domain.ComprobacionAfirmativa,
				FuenteRef:  "fuente:bolsa-sintetica-001",
				ReciboRef:  "recibo:bolsa-sintetica-001",
				EvaluadaEn: instante.Add(2 * time.Minute),
			}},
			Motivacion: "Cobertura sintética disponible.",
		},
		domain.DatosActuacion{
			AccionClave:   "cobertura.registrada",
			ActorRef:      "persona:analista-sintetica-001",
			UnidadRef:     "unidad:rrhh-sintetica-001",
			ReciboRef:     "recibo:cobertura-sintetica-001",
			RealizadaEn:   instante.Add(2 * time.Minute),
			FaseDestino:   expediente.FaseActual,
			EstadoDestino: expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}
