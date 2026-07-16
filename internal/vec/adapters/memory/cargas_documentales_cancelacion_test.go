package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type contextoCargaCanceladoTrasPrimeraConsulta struct {
	context.Context
	consultas atomic.Uint32
	terminado chan struct{}
	cancelar  sync.Once
}

func nuevoContextoCargaCanceladoTrasPrimeraConsulta() *contextoCargaCanceladoTrasPrimeraConsulta {
	return &contextoCargaCanceladoTrasPrimeraConsulta{
		Context: context.Background(), terminado: make(chan struct{}),
	}
}

func (c *contextoCargaCanceladoTrasPrimeraConsulta) Done() <-chan struct{} { return c.terminado }

func (c *contextoCargaCanceladoTrasPrimeraConsulta) Err() error {
	if c.consultas.Add(1) == 1 {
		return nil
	}
	c.cancelar.Do(func() { close(c.terminado) })
	return context.Canceled
}

func TestRepositorioCargasMemoriaRevalidaCancelacionDentroDelMutex(t *testing.T) {
	t.Run("reservar", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		ctx := nuevoContextoCargaCanceladoTrasPrimeraConsulta()

		if _, err := repositorio.Reservar(ctx, escenario.solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("reserva cancelada durante el bloqueo: %v", err)
		}
		if len(repositorio.reservasPorIndice) != 0 || len(repositorio.indicePorHuellaToken) != 0 ||
			len(repositorio.decisionesPreparacion) != 0 || len(repositorio.cargasPorID) != 0 ||
			len(repositorio.manifiestosPorID) != 0 || len(repositorio.auditoria) != 0 ||
			len(repositorio.eventos) != 0 {
			t.Fatal("la reserva cancelada publico estado parcial")
		}
	})

	t.Run("confirmar_preparacion", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		huellaToken, err := reserva.Token.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
		ctx := nuevoContextoCargaCanceladoTrasPrimeraConsulta()
		solicitud := ports.SolicitudConfirmarPreparacionCargaDocumental{
			Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
		}

		if err := repositorio.ConfirmarPreparacion(ctx, solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("confirmacion cancelada durante el bloqueo: %v", err)
		}
		persistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
		reclamacion := repositorio.decisionesPreparacion[escenario.solicitud.DecisionPreparacion.DecisionRef]
		_, indexada := repositorio.indicePorHuellaToken[huellaToken]
		if persistida.estado != estadoReservaCargaMemoriaActiva ||
			persistida.huellaTokenSHA256 != huellaToken || !indexada ||
			reclamacion.estado != estadoDecisionPreparacionCargaActiva ||
			reclamacion.huellaTokenSHA256 != huellaToken || len(repositorio.cargasPorID) != 0 ||
			len(repositorio.manifiestosPorID) != 0 || len(repositorio.auditoria) != 0 ||
			len(repositorio.eventos) != 0 {
			t.Fatal("la confirmacion cancelada consumio autoridad o dejo efectos")
		}
	})

	t.Run("confirmar_transicion", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
		if err := repositorio.ConfirmarPreparacion(
			context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
				Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
			},
		); err != nil {
			t.Fatal(err)
		}
		confirmacion := confirmacionRecepcionCargaCancelacionPrueba(t, escenario)
		ctx := nuevoContextoCargaCanceladoTrasPrimeraConsulta()

		if err := repositorio.ConfirmarTransicion(ctx, confirmacion); !errors.Is(err, context.Canceled) {
			t.Fatalf("transicion cancelada durante el bloqueo: %v", err)
		}
		persistida := repositorio.cargasPorID[escenario.preparada.ID]
		reservaPersistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
		if persistida.Version != escenario.preparada.Version || persistida.Estado != escenario.preparada.Estado ||
			persistida.ContenidoCuarentena != nil || reservaPersistida.carga.Version != escenario.preparada.Version ||
			len(repositorio.auditoria) != 1 || len(repositorio.eventos) != 1 {
			t.Fatal("la transicion cancelada altero el agregado o su evidencia")
		}
	})

	t.Run("abandonar", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		huellaToken, err := reserva.Token.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		ctx := nuevoContextoCargaCanceladoTrasPrimeraConsulta()

		if err := repositorio.AbandonarReserva(ctx, reserva.Token); !errors.Is(err, context.Canceled) {
			t.Fatalf("abandono cancelado durante el bloqueo: %v", err)
		}
		persistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
		reclamacion := repositorio.decisionesPreparacion[escenario.solicitud.DecisionPreparacion.DecisionRef]
		_, indexada := repositorio.indicePorHuellaToken[huellaToken]
		if persistida.estado != estadoReservaCargaMemoriaActiva ||
			persistida.huellaTokenSHA256 != huellaToken || !indexada ||
			reclamacion.estado != estadoDecisionPreparacionCargaActiva ||
			reclamacion.huellaTokenSHA256 != huellaToken {
			t.Fatal("el abandono cancelado consumio la capacidad")
		}
	})
}

func TestRepositorioCargasMemoriaPropagaCancelacionPreexistente(t *testing.T) {
	contextoCancelado := func() context.Context {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		return ctx
	}

	t.Run("reservar", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		if _, err := repositorio.Reservar(contextoCancelado(), escenario.solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion inicial convertida en error de dominio: %v", err)
		}
		if len(repositorio.reservasPorIndice) != 0 || len(repositorio.indicePorHuellaToken) != 0 ||
			len(repositorio.decisionesPreparacion) != 0 {
			t.Fatal("la reserva inicialmente cancelada dejo estado")
		}
	})

	t.Run("confirmar_preparacion", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		solicitud := ports.SolicitudConfirmarPreparacionCargaDocumental{
			Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
		}
		if err := repositorio.ConfirmarPreparacion(contextoCancelado(), solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion inicial convertida en error de dominio: %v", err)
		}
		persistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
		if persistida.estado != estadoReservaCargaMemoriaActiva || len(repositorio.cargasPorID) != 0 {
			t.Fatal("la confirmacion inicialmente cancelada consumio la reserva")
		}
	})

	t.Run("confirmar_transicion", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		if err := repositorio.ConfirmarTransicion(
			contextoCancelado(), escenario.confirmacion,
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion inicial convertida en error de dominio: %v", err)
		}
		if len(repositorio.cargasPorID) != 0 || len(repositorio.auditoria) != 0 || len(repositorio.eventos) != 0 {
			t.Fatal("la transicion inicialmente cancelada dejo efectos")
		}
	})

	t.Run("abandonar", func(t *testing.T) {
		reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
		repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
		escenario := escenarioCargaMemoriaPrueba(t)
		reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		if err := repositorio.AbandonarReserva(contextoCancelado(), reserva.Token); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion inicial convertida en error de dominio: %v", err)
		}
		persistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
		if persistida.estado != estadoReservaCargaMemoriaActiva {
			t.Fatal("el abandono inicialmente cancelado consumio la reserva")
		}
	})
}

func confirmacionRecepcionCargaCancelacionPrueba(
	t *testing.T,
	escenario escenarioPreparacionCargaMemoria,
) ports.ConfirmacionTransicionCargaDocumental {
	t.Helper()
	registradoEn := instanteCargasMemoriaPrueba.Add(3 * time.Second)
	contenido := domain.ContenidoCargaDocumental{
		ConectorID: "almacen_s3_corporativo", Referencia: "objeto:cuarentena:cancelacion:0123456789abcdef",
		Version: "v1", Zona: domain.ZonaContenidoCargaCuarentena, MIME: escenario.preparada.MIMEDeclarado,
		Tamano: escenario.preparada.TamanoDeclarado, HuellaSHA256: escenario.preparada.HuellaDeclaradaSHA256,
		EvidenciaRef: "evidencia:confirmacion:cancelacion:0123456789abcdef", RegistradoEn: registradoEn,
	}
	cuarentena, err := escenario.preparada.RegistrarCuarentena(
		contenido, "decision:confirmar:cancelacion:0123456789abcdef", registradoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaAnterior, err := escenario.preparada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaSiguiente, err := cuarentena.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	auditoria := domain.AuditEntry{
		ActorID: escenario.preparada.PrincipalID, ActorProfile: "perfil:externo:0123456789abcdef",
		ActorRoles: []string{"usuario_externo"}, AuthorizationRef: cuarentena.AutorizacionRecepcionRef,
		Purpose: cuarentena.Finalidad, Action: "vec.documentos.carga.confirmar", ModuleID: cuarentena.ModuloID,
		SubjectRef: cuarentena.ID, ObjectVersion: cuarentena.Version, Result: "correcto",
		BeforeHash: huellaAnterior, AfterHash: huellaSiguiente, CorrelationRef: cuarentena.CorrelacionRef,
		Metadata: map[string]string{"estado": string(cuarentena.Estado)}, OccurredAt: registradoEn,
	}
	evento := domain.Event{
		Type: "vec.documentos.carga.recibida", ModuleID: cuarentena.ModuloID, SubjectRef: cuarentena.ID,
		ActorID: auditoria.ActorID, OccurredAt: registradoEn, Payload: map[string]string{
			"carga_ref": cuarentena.ID, "estado": string(cuarentena.Estado),
			"version": "3",
		},
	}
	confirmacion, err := ports.NuevaConfirmacionTransicionCargaDocumental(
		escenario.preparada, cuarentena, auditoria, evento,
	)
	if err != nil {
		t.Fatal(err)
	}
	return confirmacion
}
