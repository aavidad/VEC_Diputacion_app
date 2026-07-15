package memory

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var instanteCargasMemoriaPrueba = time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC)

type relojCargasMemoriaPrueba struct {
	mu    sync.Mutex
	ahora time.Time
}

func (r *relojCargasMemoriaPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ahora
}

func (r *relojCargasMemoriaPrueba) fijar(instante time.Time) {
	r.mu.Lock()
	r.ahora = instante.UTC()
	r.mu.Unlock()
}

type escenarioPreparacionCargaMemoria struct {
	reserva      ports.ReservaCargaDocumental
	solicitud    ports.SolicitudReservarCargaDocumental
	preparada    domain.CargaDocumental
	confirmacion ports.ConfirmacionTransicionCargaDocumental
	manifiesto   domain.ManifiestoPreparacionCargaDirectaV1
}

func escenarioCargaMemoriaPrueba(t *testing.T) escenarioPreparacionCargaMemoria {
	return escenarioCargaMemoriaPruebaConMarca(
		t, "b", "decision:preparar:0123456789abcdef", "efecto:carga:preparar:0123456789abcdef",
	)
}

func escenarioCargaMemoriaPruebaConMarca(
	t *testing.T,
	marca, decisionRef, efectoRef string,
) escenarioPreparacionCargaMemoria {
	t.Helper()
	sufijo := strings.Repeat(marca, 16)
	reservada, err := domain.NuevaCargaDocumental(
		"carga:memoria:"+sufijo, "persona:0123456789abcdef", "solicitud:"+sufijo,
		"bolsa", "solicitud", "operacion:carga:"+sufijo, "correlacion:carga:"+sufijo,
		"aportar_documentacion_bolsa", "datos_personales", "application/pdf", 128,
		strings.Repeat("a", 64), "hmac-sha256:idempotencia_v1:"+strings.Repeat(marca, 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("c", 64), instanteCargasMemoriaPrueba,
		instanteCargasMemoriaPrueba.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := ports.SolicitudReservarCargaDocumental{
		IndiceIdempotenciaHMAC: reservada.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    reservada.HuellaSolicitudHMAC, Carga: reservada,
		DecisionPreparacion: ports.ConsumoDecisionPreparacionCargaDocumentalV1{
			DecisionRef: decisionRef, EfectoRef: efectoRef,
			HuellaPlanEfectoSHA256: strings.Repeat("1", 64),
			EsquemaHuellaDecision:  ports.EsquemaHuellaDecisionAutorizacionReforzadaV1,
			HuellaDecisionSHA256:   strings.Repeat("2", 64),
		},
		SolicitadaEn:    instanteCargasMemoriaPrueba,
		ReservaExpiraEn: instanteCargasMemoriaPrueba.Add(5 * time.Minute),
	}
	preparada, err := reservada.Preparar(
		"hmac-sha256:sesion_v1:"+strings.Repeat("d", 64),
		decisionRef, instanteCargasMemoriaPrueba.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaAnterior, _ := reservada.HuellaSHA256()
	huellaSiguiente, _ := preparada.HuellaSHA256()
	auditoria := domain.AuditEntry{
		ActorID: preparada.PrincipalID, ActorProfile: "perfil:externo:0123456789abcdef",
		AuthorizationRef: preparada.AutorizacionPreparacionRef, Purpose: preparada.Finalidad,
		Action: "vec.documentos.carga.preparar", ModuleID: preparada.ModuloID, SubjectRef: preparada.ID,
		ObjectVersion: preparada.Version, Result: "correcto", BeforeHash: huellaAnterior, AfterHash: huellaSiguiente,
		CorrelationRef: preparada.CorrelacionRef, Metadata: map[string]string{"estado": string(preparada.Estado)},
		OccurredAt: preparada.PreparadaEn,
	}
	evento := domain.Event{
		Type: "vec.documentos.carga.preparada", ModuleID: preparada.ModuloID, SubjectRef: preparada.ID,
		ActorID: auditoria.ActorID, OccurredAt: auditoria.OccurredAt,
		Payload: map[string]string{"carga_ref": preparada.ID, "estado": string(preparada.Estado), "version": "2"},
	}
	confirmacion, err := ports.NuevaConfirmacionTransicionCargaDocumental(reservada, preparada, auditoria, evento)
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := domain.NuevoManifiestoPreparacionCargaDirectaV1(
		preparada,
		domain.ContextoManifiestoPreparacionCargaDirectaV1{
			CargaRef:                preparada.IndiceIdempotenciaHMAC,
			SujetoSeudonimoHMAC:     "hmac-sha256:seudonimo_v1:" + strings.Repeat("e", 64),
			HuellaRecursoBaseSHA256: strings.Repeat("3", 64),
			HuellaRecursoSHA256:     strings.Repeat("f", 64), ConectorAlmacenID: "almacen_s3_corporativo",
			EsquemaContexto:        ports.EsquemaContextoOperacionAlmacenV1,
			AccionNegocio:          ports.AccionNegocioPrepararCargaDocumental,
			AccionTecnica:          ports.AccionAlmacenPrepararCargaDirecta,
			PasoRef:                string(ports.PasoAlmacenPrepararCargaDirecta),
			EfectoRef:              efectoRef,
			HuellaPlanEfectoSHA256: strings.Repeat("1", 64),
			EsquemaHuellaDecision:  ports.EsquemaHuellaDecisionAutorizacionReforzadaV1,
			DecisionRef:            preparada.AutorizacionPreparacionRef,
			HuellaDecisionSHA256:   strings.Repeat("2", 64),
			ContextoVerificadoEn:   instanteCargasMemoriaPrueba.Add(500 * time.Millisecond),
			DecisionValidaHasta:    preparada.ExpiraEn.Add(time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioPreparacionCargaMemoria{
		solicitud: solicitud, preparada: preparada, confirmacion: confirmacion, manifiesto: manifiesto,
	}
}

func TestRepositorioCargasMemoriaConfirmaReservaYManifiestoAtomicamente(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, err := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	if err != nil {
		t.Fatal(err)
	}
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	escenario.reserva = reserva
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	if err := repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
	}); err != nil {
		t.Fatal(err)
	}
	escenario.confirmacion.Auditoria.Metadata["estado"] = "alterado-tras-commit"
	escenario.confirmacion.Evento.Payload["estado"] = "alterado-tras-commit"
	repositorio.mu.RLock()
	estadoAuditado := repositorio.auditoria[0].Metadata["estado"]
	estadoEvento := repositorio.eventos[0].Payload["estado"]
	repositorio.mu.RUnlock()
	if estadoAuditado != string(escenario.preparada.Estado) || estadoEvento != string(escenario.preparada.Estado) {
		t.Fatal("el commit conservo alias mutables de la solicitud original")
	}
	persistida, err := repositorio.ObtenerPreparacion(context.Background(), escenario.preparada.ID)
	if err != nil || persistida.Validar() != nil {
		t.Fatalf("preparacion persistida: %#v, %v", persistida, err)
	}
	huellaEsperada, _ := escenario.manifiesto.HuellaSHA256()
	huellaPersistida, _ := persistida.Manifiesto.HuellaSHA256()
	if huellaEsperada != huellaPersistida {
		t.Fatal("el repositorio sustituyo el manifiesto")
	}
	repetida, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil || !repetida.Repetida || repetida.Carga.ID != escenario.preparada.ID {
		t.Fatalf("reserva idempotente confirmada: %#v, %v", repetida, err)
	}
}

func TestRepositorioCargasMemoriaNoDejaEstadoParcialSinManifiesto(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	err = repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion,
	})
	if !errors.Is(err, ports.ErrConfirmacionCargaDocumentalInvalida) {
		t.Fatalf("se acepto confirmar sin manifiesto: %v", err)
	}
	if _, err := repositorio.Obtener(context.Background(), escenario.preparada.ID); !errors.Is(err, ports.ErrCargaDocumentalNoEncontrada) {
		t.Fatal("se escribio el agregado sin manifiesto")
	}
	if err := repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
	}); err != nil {
		t.Fatalf("el fallo cerrado consumio parcialmente la reserva: %v", err)
	}
}

func TestRepositorioCargasMemoriaTransicionPersisteLaInstantaneaValidada(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	if err := repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
	}); err != nil {
		t.Fatal(err)
	}
	registradoEn := instanteCargasMemoriaPrueba.Add(3 * time.Second)
	contenido := domain.ContenidoCargaDocumental{
		ConectorID: "almacen_s3_corporativo", Referencia: "objeto:cuarentena:0123456789abcdef", Version: "v1",
		Zona: domain.ZonaContenidoCargaCuarentena, MIME: escenario.preparada.MIMEDeclarado,
		Tamano: escenario.preparada.TamanoDeclarado, HuellaSHA256: escenario.preparada.HuellaDeclaradaSHA256,
		EvidenciaRef: "evidencia:confirmacion:0123456789abcdef", RegistradoEn: registradoEn,
	}
	cuarentena, err := escenario.preparada.RegistrarCuarentena(
		contenido, "decision:confirmar:0123456789abcdef", registradoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaAnterior, _ := escenario.preparada.HuellaSHA256()
	huellaSiguiente, _ := cuarentena.HuellaSHA256()
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
			"carga_ref": cuarentena.ID, "estado": string(cuarentena.Estado), "version": "3",
		},
	}
	confirmacion, err := ports.NuevaConfirmacionTransicionCargaDocumental(
		escenario.preparada, cuarentena, auditoria, evento,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repositorio.ConfirmarTransicion(context.Background(), confirmacion); err != nil {
		t.Fatal(err)
	}
	confirmacion.Carga.ContenidoCuarentena.MIME = "application/zip"
	confirmacion.Auditoria.ActorRoles[0] = "rol_mutado"
	confirmacion.Auditoria.Metadata["estado"] = "alterado"
	confirmacion.Evento.Payload["estado"] = "alterado"
	repositorio.mu.RLock()
	persistida := repositorio.cargasPorID[cuarentena.ID]
	auditada := repositorio.auditoria[len(repositorio.auditoria)-1]
	emitido := repositorio.eventos[len(repositorio.eventos)-1]
	repositorio.mu.RUnlock()
	if persistida.ContenidoCuarentena == nil || persistida.ContenidoCuarentena.MIME != cuarentena.MIMEDeclarado ||
		auditada.ActorRoles[0] != "usuario_externo" || auditada.Metadata["estado"] != string(cuarentena.Estado) ||
		emitido.Payload["estado"] != string(cuarentena.Estado) {
		t.Fatal("la transicion persistida compartio alias con el argumento original")
	}
}

func TestRepositorioCargasMemoriaSoloUnaCarreraConsumeLaReserva(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	solicitud := ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
	}
	var correctas atomic.Int32
	var wg sync.WaitGroup
	errores := make(chan error, 2)
	for indice := 0; indice < 2; indice++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repositorio.ConfirmarPreparacion(context.Background(), solicitud)
			if err == nil {
				correctas.Add(1)
				return
			}
			errores <- err
		}()
	}
	wg.Wait()
	close(errores)
	if correctas.Load() != 1 || len(errores) != 1 {
		t.Fatalf("resultados de carrera: correctas=%d errores=%d", correctas.Load(), len(errores))
	}
	for err := range errores {
		if !errors.Is(err, ports.ErrConfirmacionCargaDocumentalInvalida) {
			t.Fatalf("error de carrera inesperado: %v", err)
		}
	}
	preparada, err := repositorio.ObtenerPreparacion(context.Background(), escenario.preparada.ID)
	if err != nil || preparada.Validar() != nil {
		t.Fatalf("resultado unico no quedo coherente: %v", err)
	}
}

func TestRepositorioCargasMemoriaDecisionPreparacionEsUnicaYElConflictoNoDejaEstadoParcial(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	const decisionCompartida = "decision:preparar:compartida:0123456789abcdef"
	primera := escenarioCargaMemoriaPruebaConMarca(
		t, "b", decisionCompartida, "efecto:carga:primero:0123456789abcdef",
	)
	segunda := escenarioCargaMemoriaPruebaConMarca(
		t, "c", decisionCompartida, "efecto:carga:segundo:0123456789abcdef",
	)
	repetida := escenarioCargaMemoriaPruebaConMarca(
		t, "d", decisionCompartida, "efecto:carga:primero:0123456789abcdef",
	)
	reservaPrimera, err := repositorio.Reservar(context.Background(), primera.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.Reservar(context.Background(), segunda.solicitud); !errors.Is(
		err, ports.ErrDecisionPreparacionCargaNoDisponible,
	) {
		t.Fatalf("se reclamo DecisionRef para otro efecto antes del almacen: %v", err)
	}
	if _, err := repositorio.Reservar(context.Background(), repetida.solicitud); !errors.Is(
		err, ports.ErrDecisionPreparacionCargaNoDisponible,
	) {
		t.Fatalf("se repitio la reclamacion exacta de DecisionRef: %v", err)
	}
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	if err := repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reservaPrimera.Token, Confirmacion: primera.confirmacion, Manifiesto: primera.manifiesto,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.Obtener(context.Background(), segunda.preparada.ID); !errors.Is(err, ports.ErrCargaDocumentalNoEncontrada) {
		t.Fatal("el conflicto de decision escribio el segundo agregado")
	}
	repositorio.mu.RLock()
	reclamacion := repositorio.decisionesPreparacion[decisionCompartida]
	consumos := len(repositorio.decisionesPreparacion)
	manifiestos := len(repositorio.manifiestosPorID)
	repositorio.mu.RUnlock()
	if reclamacion.estado != estadoDecisionPreparacionCargaConsumida ||
		reclamacion.consumo != primera.solicitud.DecisionPreparacion || consumos != 1 || manifiestos != 1 {
		t.Fatalf("el conflicto altero la reclamacion ganadora: estado=%d decisiones=%d manifiestos=%d",
			reclamacion.estado, consumos, manifiestos)
	}
}

func TestRepositorioCargasMemoriaCarreraDecisionCompartidaSoloPreparaUnEfecto(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	const decisionCompartida = "decision:preparar:carrera:0123456789abcdef"
	escenarios := []escenarioPreparacionCargaMemoria{
		escenarioCargaMemoriaPruebaConMarca(t, "b", decisionCompartida, "efecto:carga:carrera:uno:0123456789abcdef"),
		escenarioCargaMemoriaPruebaConMarca(t, "c", decisionCompartida, "efecto:carga:carrera:dos:0123456789abcdef"),
	}
	type resultadoReclamacion struct {
		indice  int
		reserva ports.ReservaCargaDocumental
		err     error
	}
	resultados := make(chan resultadoReclamacion, len(escenarios))
	var wg sync.WaitGroup
	for indice, escenario := range escenarios {
		indice, escenario := indice, escenario
		wg.Add(1)
		go func() {
			defer wg.Done()
			reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
			resultados <- resultadoReclamacion{indice: indice, reserva: reserva, err: err}
		}()
	}
	wg.Wait()
	close(resultados)
	ganadora := resultadoReclamacion{indice: -1}
	errores := 0
	for resultado := range resultados {
		if resultado.err == nil {
			ganadora = resultado
			continue
		}
		errores++
		if !errors.Is(resultado.err, ports.ErrDecisionPreparacionCargaNoDisponible) {
			t.Fatalf("error de reclamacion inesperado: %v", resultado.err)
		}
	}
	if ganadora.indice < 0 || errores != 1 {
		t.Fatalf("carrera de DecisionRef: ganadora=%d errores=%d", ganadora.indice, errores)
	}
	reloj.fijar(instanteCargasMemoriaPrueba.Add(2 * time.Second))
	escenarioGanador := escenarios[ganadora.indice]
	if err := repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: ganadora.reserva.Token, Confirmacion: escenarioGanador.confirmacion, Manifiesto: escenarioGanador.manifiesto,
	}); err != nil {
		t.Fatalf("confirmar reclamacion ganadora: %v", err)
	}
	repositorio.mu.RLock()
	consumos := len(repositorio.decisionesPreparacion)
	reclamacion := repositorio.decisionesPreparacion[decisionCompartida]
	cargas := len(repositorio.cargasPorID)
	manifiestos := len(repositorio.manifiestosPorID)
	repositorio.mu.RUnlock()
	if consumos != 1 || reclamacion.estado != estadoDecisionPreparacionCargaConsumida ||
		cargas != 1 || manifiestos != 1 {
		t.Fatalf("commit de carrera incoherente: decisiones=%d cargas=%d manifiestos=%d",
			consumos, cargas, manifiestos)
	}
}

func TestRepositorioCargasMemoriaAbandonoRetiraDecisionYReintentoExigeOtra(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if err := repositorio.AbandonarReserva(context.Background(), reserva.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.Reservar(context.Background(), escenario.solicitud); !errors.Is(
		err, ports.ErrDecisionPreparacionCargaNoDisponible,
	) {
		t.Fatalf("el abandono libero la misma DecisionRef: %v", err)
	}
	reintento := escenarioCargaMemoriaPruebaConMarca(
		t, "b", "decision:preparar:nueva:0123456789abcdef", escenario.solicitud.DecisionPreparacion.EfectoRef,
	)
	if _, err := repositorio.Reservar(context.Background(), reintento.solicitud); err != nil {
		t.Fatalf("una decision nueva no pudo iniciar el reintento: %v", err)
	}
	repositorio.mu.RLock()
	retirada := repositorio.decisionesPreparacion[escenario.solicitud.DecisionPreparacion.DecisionRef]
	nueva := repositorio.decisionesPreparacion[reintento.solicitud.DecisionPreparacion.DecisionRef]
	repositorio.mu.RUnlock()
	if retirada.estado != estadoDecisionPreparacionCargaAbandonada ||
		nueva.estado != estadoDecisionPreparacionCargaActiva {
		t.Fatalf("estados tras abandono/reintento: retirada=%d nueva=%d", retirada.estado, nueva.estado)
	}
}

func TestRepositorioCargasMemoriaLeaseVencidoExpiraTokenYDecisionSinFallback(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	reserva, err := repositorio.Reservar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	reloj.fijar(escenario.solicitud.ReservaExpiraEn)
	err = repositorio.ConfirmarPreparacion(context.Background(), ports.SolicitudConfirmarPreparacionCargaDocumental{
		Token: reserva.Token, Confirmacion: escenario.confirmacion, Manifiesto: escenario.manifiesto,
	})
	if !errors.Is(err, ports.ErrConfirmacionCargaDocumentalInvalida) {
		t.Fatalf("se confirmo una lease vencida: %v", err)
	}
	valorToken, _ := reserva.Token.RevelarParaPersistencia()
	repositorio.mu.RLock()
	_, tokenActivo := repositorio.indicePorToken[valorToken]
	reclamacion := repositorio.decisionesPreparacion[escenario.solicitud.DecisionPreparacion.DecisionRef]
	reservaPersistida := repositorio.reservasPorIndice[escenario.solicitud.IndiceIdempotenciaHMAC]
	repositorio.mu.RUnlock()
	if tokenActivo || reclamacion.estado != estadoDecisionPreparacionCargaExpirada ||
		reservaPersistida.estado != estadoReservaCargaMemoriaExpirada {
		t.Fatalf("lease vencida ambigua: token=%v decision=%d reserva=%d",
			tokenActivo, reclamacion.estado, reservaPersistida.estado)
	}
}

func TestRepositorioCargasMemoriaRechazaCruceDeSolicitudIdempotente(t *testing.T) {
	reloj := &relojCargasMemoriaPrueba{ahora: instanteCargasMemoriaPrueba}
	repositorio, _ := NuevoRepositorioCargasDocumentalesMemoria(reloj)
	escenario := escenarioCargaMemoriaPrueba(t)
	if _, err := repositorio.Reservar(context.Background(), escenario.solicitud); err != nil {
		t.Fatal(err)
	}
	cruzada := escenario.solicitud
	cruzada.HuellaSolicitudHMAC = "hmac-sha256:solicitud_v2:" + strings.Repeat("9", 64)
	cruzada.Carga.HuellaSolicitudHMAC = cruzada.HuellaSolicitudHMAC
	if _, err := repositorio.Reservar(context.Background(), cruzada); !errors.Is(err, ports.ErrReservaCargaDocumentalOcupada) {
		t.Fatalf("se acepto otra solicitud para el mismo indice: %v", err)
	}
}
