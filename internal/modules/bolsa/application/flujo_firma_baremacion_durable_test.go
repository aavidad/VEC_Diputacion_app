package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	adaptadormemoria "vec-diputacion-granada/internal/modules/bolsa/adapters/memory"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	claveIdempotenciaFlujoFirmaPrueba = "idempotencia-flujo-firma-baremacion-001"
	claveRefEstadoFlujoFirmaPrueba    = "clave-estado-flujo-firma-v1"
)

var claveEstadoFlujoFirmaPrueba = bytes.Repeat([]byte{0x5a}, 32)

func TestFachadaFirmaBaremacionDurableReanudaTrasTimeoutYReinicioSinDuplicarEfectos(t *testing.T) {
	entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
	entorno.ejecutor.fallarTrasEfecto[puertosbolsa.PasoCustodiarFirmaBaremacion] = true

	lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
	if err != nil {
		t.Fatalf("Preparar() error = %v; preparacion_declarada=%v", err, entorno.repositorio.preparacionDeclaradaAntesDelEfecto())
	}
	if !entorno.repositorio.preparacionDeclaradaAntesDelEfecto() {
		t.Fatal("la preparacion se ejecuto sin persistir antes su punto de control")
	}

	orden := OrdenReanudarFlujoFirmaBaremacion{
		FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
	}
	if _, err := entorno.fachada.Finalizar(context.Background(), orden); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("primer Finalizar() error = %v; se esperaba timeout ambiguo", err)
	}
	if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoCustodiarFirmaBaremacion); obtuvo != 1 {
		t.Fatalf("efectos semanticos de custodia tras timeout = %d; se esperaba 1", obtuvo)
	}
	sesionReautenticada := sesionBaremacionAutenticacionAlternativaPrueba(t)
	entorno.sesiones.mu.Lock()
	entorno.sesiones.sesiones = []SesionAutenticadaBaremacion{sesionReautenticada}
	entorno.sesiones.mu.Unlock()

	// Simula otra replica o un reinicio: nueva fachada y nuevo AEAD, con la
	// misma clave versionada, una nueva sesion del mismo actor y el mismo
	// repositorio durable compartido.
	protectorReiniciado, err := adaptadormemoria.NuevoProtectorEstadoFlujoFirmaBaremacion(
		claveRefEstadoFlujoFirmaPrueba,
		claveEstadoFlujoFirmaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	fachadaReiniciada := nuevaFachadaFlujoFirmaPrueba(
		t,
		entorno.repositorio,
		protectorReiniciado,
		entorno.ejecutor,
		entorno.sesiones,
		entorno.reloj,
		entorno.sellador,
	)
	resultado, err := fachadaReiniciada.Finalizar(context.Background(), orden)
	if err != nil {
		t.Fatalf("Finalizar() tras reinicio error = %v", err)
	}

	for _, paso := range puertosbolsa.PasosFlujoFirmaBaremacion() {
		if obtuvo := entorno.ejecutor.efectosSemanticos(paso); obtuvo != 1 {
			t.Errorf("efectos semanticos de %s = %d; se esperaba 1", paso, obtuvo)
		}
	}
	if obtuvo := entorno.ejecutor.invocaciones(puertosbolsa.PasoCustodiarFirmaBaremacion); obtuvo != 2 {
		t.Fatalf("invocaciones de custodia = %d; se esperaba recuperacion idempotente", obtuvo)
	}

	invocacionesAntes := entorno.ejecutor.totalInvocaciones()
	recuperado, err := entorno.fachada.Finalizar(context.Background(), orden)
	if err != nil {
		t.Fatalf("Finalizar() de flujo completado error = %v", err)
	}
	if !reflect.DeepEqual(recuperado, resultado) {
		t.Fatalf("resultado recuperado distinto:\nobtenido: %#v\nesperado: %#v", recuperado, resultado)
	}
	if obtuvo := entorno.ejecutor.totalInvocaciones(); obtuvo != invocacionesAntes {
		t.Fatalf("un flujo completado volvio a ejecutar efectos: antes=%d despues=%d", invocacionesAntes, obtuvo)
	}
}

func TestFachadaFirmaBaremacionDurableReintentaCadaEfectoFinalConLaMismaIdentidad(t *testing.T) {
	pasos := []puertosbolsa.PasoFlujoFirmaBaremacion{
		puertosbolsa.PasoCompletarFirmaBaremacion,
		puertosbolsa.PasoRetenerFirmaBaremacion,
		puertosbolsa.PasoReservarFirmaBaremacion,
		puertosbolsa.PasoConfirmarFirmaBaremacion,
	}
	for _, paso := range pasos {
		paso := paso
		t.Run(string(paso), func(t *testing.T) {
			entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
			lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
			if err != nil {
				t.Fatal(err)
			}
			entorno.ejecutor.fallarTrasEfecto[paso] = true
			orden := OrdenReanudarFlujoFirmaBaremacion{
				FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
			}
			if _, err := entorno.fachada.Finalizar(context.Background(), orden); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Finalizar() con timeout tras %s error = %v", paso, err)
			}

			protectorReplica, err := adaptadormemoria.NuevoProtectorEstadoFlujoFirmaBaremacion(
				claveRefEstadoFlujoFirmaPrueba,
				claveEstadoFlujoFirmaPrueba,
			)
			if err != nil {
				t.Fatal(err)
			}
			fachadaReplica := nuevaFachadaFlujoFirmaPrueba(
				t,
				entorno.repositorio,
				protectorReplica,
				entorno.ejecutor,
				entorno.sesiones,
				entorno.reloj,
				entorno.sellador,
			)
			if _, err := fachadaReplica.Finalizar(context.Background(), orden); err != nil {
				t.Fatalf("Finalizar() recuperando %s error = %v", paso, err)
			}
			for _, pasoEjecutado := range puertosbolsa.PasosFlujoFirmaBaremacion() {
				if obtuvo := entorno.ejecutor.efectosSemanticos(pasoEjecutado); obtuvo != 1 {
					t.Errorf("efectos semanticos de %s = %d; se esperaba 1", pasoEjecutado, obtuvo)
				}
			}
			if obtuvo := entorno.ejecutor.invocaciones(paso); obtuvo != 2 {
				t.Fatalf("invocaciones de %s = %d; se esperaba una recuperacion", paso, obtuvo)
			}
		})
	}
}

func TestFachadaFirmaBaremacionDurableReanudaPreparacionSinExponerProyeccionPrematura(t *testing.T) {
	entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
	entorno.ejecutor.fallarTrasEfecto[puertosbolsa.PasoPrepararFirmaBaremacion] = true
	orden := ordenPrepararFlujoFirmaPrueba(t)

	if proyeccion, err := entorno.fachada.Preparar(context.Background(), orden); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("primer Preparar() = (%#v, %v); se esperaba timeout sin proyeccion", proyeccion, err)
	}
	if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoPrepararFirmaBaremacion); obtuvo != 1 {
		t.Fatalf("efectos semanticos de preparacion tras timeout = %d", obtuvo)
	}

	proyeccion, err := entorno.fachada.Preparar(context.Background(), orden)
	if err != nil {
		t.Fatalf("Preparar() reintentado error = %v", err)
	}
	if proyeccion.Validar() != nil {
		t.Fatalf("proyeccion recuperada invalida: %#v", proyeccion)
	}
	if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoPrepararFirmaBaremacion); obtuvo != 1 {
		t.Fatalf("el reintento duplico la preparacion: %d efectos", obtuvo)
	}
	if obtuvo := entorno.ejecutor.invocaciones(puertosbolsa.PasoPrepararFirmaBaremacion); obtuvo != 2 {
		t.Fatalf("invocaciones de preparacion = %d; se esperaba recuperar el mismo efecto", obtuvo)
	}
	consulta, err := entorno.fachada.Consultar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
		FlujoRef: proyeccion.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
	})
	if err != nil || consulta.Estado != puertosbolsa.EstadoExpedienteFirmaPendienteInteraccion || consulta.Lanzamiento == nil {
		t.Fatalf("estado persistido tras devolver la proyeccion = (%#v, %v)", consulta, err)
	}
}

func TestFachadaFirmaBaremacionDurableExcluyeFinalizacionesConcurrentes(t *testing.T) {
	entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
	lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	entorno.ejecutor.configurarBloqueo(puertosbolsa.PasoCompletarFirmaBaremacion)
	orden := OrdenReanudarFlujoFirmaBaremacion{
		FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
	}

	resultados := make(chan puertosbolsa.ResultadoFinalFlujoFirmaBaremacion, 1)
	errores := make(chan error, 1)
	go func() {
		resultado, errorFinal := entorno.fachada.Finalizar(context.Background(), orden)
		resultados <- resultado
		errores <- errorFinal
	}()

	entorno.ejecutor.esperarPasoBloqueado(t)
	protectorReplica, err := adaptadormemoria.NuevoProtectorEstadoFlujoFirmaBaremacion(
		claveRefEstadoFlujoFirmaPrueba,
		claveEstadoFlujoFirmaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	fachadaReplica := nuevaFachadaFlujoFirmaPrueba(
		t,
		entorno.repositorio,
		protectorReplica,
		entorno.ejecutor,
		entorno.sesiones,
		entorno.reloj,
		entorno.sellador,
	)
	if _, err := fachadaReplica.Finalizar(context.Background(), orden); !errors.Is(err, puertosbolsa.ErrFlujoFirmaBaremacionOcupado) {
		t.Fatalf("Finalizar() concurrente error = %v; se esperaba flujo ocupado", err)
	}
	entorno.ejecutor.desbloquearPaso()

	if err := <-errores; err != nil {
		t.Fatalf("Finalizar() propietario error = %v", err)
	}
	resultado := <-resultados
	if resultado.Validar() != nil {
		t.Fatalf("resultado propietario invalido: %#v", resultado)
	}
	for _, paso := range puertosbolsa.PasosFlujoFirmaBaremacion() {
		if obtuvo := entorno.ejecutor.efectosSemanticos(paso); obtuvo != 1 {
			t.Errorf("efectos semanticos concurrentes de %s = %d; se esperaba 1", paso, obtuvo)
		}
	}
}

func TestFachadaFirmaBaremacionDurableDeniegaCruceDeClaveYEstadoAlterado(t *testing.T) {
	t.Run("clave cruzada y reutilizacion con otros datos", func(t *testing.T) {
		entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
		ordenInicial := ordenPrepararFlujoFirmaPrueba(t)
		lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenInicial)
		if err != nil {
			t.Fatal(err)
		}

		_, err = entorno.fachada.Finalizar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
			FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: "idempotencia-flujo-firma-cruzada-999",
		})
		if !errors.Is(err, puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado) {
			t.Fatalf("Finalizar() con clave cruzada error = %v", err)
		}
		if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoCompletarFirmaBaremacion); obtuvo != 0 {
			t.Fatalf("la clave cruzada alcanzo el ejecutor: %d efectos", obtuvo)
		}

		ordenCambiada := ordenInicial
		ordenCambiada.DecisionRef = "decision-baremacion-distinta-999"
		if _, err := entorno.fachada.Preparar(context.Background(), ordenCambiada); !errors.Is(err, puertosbolsa.ErrClaveFlujoFirmaBaremacionReutilizada) {
			t.Fatalf("Preparar() reutilizando clave con otra decision error = %v", err)
		}
	})

	t.Run("sobre AEAD alterado", func(t *testing.T) {
		entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
		lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
		if err != nil {
			t.Fatal(err)
		}
		entorno.repositorio.alterarSobreEnSiguienteArrendamiento()
		_, err = entorno.fachada.Finalizar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
			FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
		})
		if !errors.Is(err, puertosbolsa.ErrEstadoFlujoFirmaAlterado) {
			t.Fatalf("Finalizar() con estado alterado error = %v", err)
		}
		if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoCompletarFirmaBaremacion); obtuvo != 0 {
			t.Fatalf("el estado alterado alcanzo el ejecutor: %d efectos", obtuvo)
		}
	})

	t.Run("clave de cifrado cruzada entre replicas", func(t *testing.T) {
		entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
		lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
		if err != nil {
			t.Fatal(err)
		}
		protectorCruzado, err := adaptadormemoria.NuevoProtectorEstadoFlujoFirmaBaremacion(
			claveRefEstadoFlujoFirmaPrueba,
			bytes.Repeat([]byte{0x33}, 32),
		)
		if err != nil {
			t.Fatal(err)
		}
		fachadaCruzada := nuevaFachadaFlujoFirmaPrueba(
			t,
			entorno.repositorio,
			protectorCruzado,
			entorno.ejecutor,
			entorno.sesiones,
			entorno.reloj,
			entorno.sellador,
		)
		_, err = fachadaCruzada.Finalizar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
			FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
		})
		if !errors.Is(err, puertosbolsa.ErrEstadoFlujoFirmaAlterado) {
			t.Fatalf("Finalizar() con clave AEAD cruzada error = %v", err)
		}
		if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoCompletarFirmaBaremacion); obtuvo != 0 {
			t.Fatalf("la clave AEAD cruzada alcanzo el ejecutor: %d efectos", obtuvo)
		}
	})
}

func TestFachadaFirmaBaremacionDurableDerivaIdentidadDeSesionVigente(t *testing.T) {
	entorno := nuevoEntornoFlujoFirmaDurablePrueba(t)
	lanzamiento, err := entorno.fachada.Preparar(context.Background(), ordenPrepararFlujoFirmaPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	sesionAjena := sesionAutenticadaBaremacionIdentidadPrueba(
		t,
		"per_otra_persona_abcdefghijkl",
		"prf_otro_perfil_abcdefghijkl",
	)
	entorno.sesiones.mu.Lock()
	entorno.sesiones.sesiones = []SesionAutenticadaBaremacion{sesionAjena}
	entorno.sesiones.mu.Unlock()
	_, err = entorno.fachada.Finalizar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
		FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
	})
	if !errors.Is(err, puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado) {
		t.Fatalf("Finalizar() desde otra identidad error = %v", err)
	}

	entorno.sesiones.mu.Lock()
	entorno.sesiones.sesiones = nil
	entorno.sesiones.mu.Unlock()

	_, err = entorno.fachada.Finalizar(context.Background(), OrdenReanudarFlujoFirmaBaremacion{
		FlujoRef: lanzamiento.FlujoRef, ClaveIdempotencia: claveIdempotenciaFlujoFirmaPrueba,
	})
	if err == nil {
		t.Fatal("Finalizar() sin sesion autoritativa no denego la operacion")
	}
	if obtuvo := entorno.ejecutor.efectosSemanticos(puertosbolsa.PasoCompletarFirmaBaremacion); obtuvo != 0 {
		t.Fatalf("una orden sin sesion alcanzo el ejecutor: %d efectos", obtuvo)
	}
}

type entornoFlujoFirmaDurablePrueba struct {
	fachada     *FachadaFirmaBaremacionDurable
	repositorio *repositorioObservadoFlujoFirmaPrueba
	ejecutor    *ejecutorFlujoFirmaPrueba
	sesiones    *sesionesFlujoFirmaPrueba
	reloj       relojBaremacionPrueba
	sellador    selladorVerificadorFlujoFirmaPrueba
}

func nuevoEntornoFlujoFirmaDurablePrueba(t *testing.T) *entornoFlujoFirmaDurablePrueba {
	t.Helper()
	reloj := relojBaremacionPrueba{instante: instanteBaremacionPrueba}
	contextoActor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(instanteBaremacionPrueba)
	sesion, err := NuevaSesionAutenticadaBaremacion(contextoActor, vinculo)
	if err != nil {
		t.Fatal(err)
	}
	sesiones := &sesionesFlujoFirmaPrueba{sesiones: []SesionAutenticadaBaremacion{sesion}}
	sellador := selladorVerificadorFlujoFirmaPrueba{}
	repositorioBase, err := adaptadormemoria.NuevoRepositorioFlujosFirmaBaremacion(reloj, sellador)
	if err != nil {
		t.Fatal(err)
	}
	repositorio := &repositorioObservadoFlujoFirmaPrueba{RepositorioFlujosFirmaBaremacion: repositorioBase}
	protector, err := adaptadormemoria.NuevoProtectorEstadoFlujoFirmaBaremacion(
		claveRefEstadoFlujoFirmaPrueba,
		claveEstadoFlujoFirmaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	ejecutor := nuevoEjecutorFlujoFirmaPrueba(instanteBaremacionPrueba)
	ejecutor.antesDeEjecutar = func(solicitud puertosbolsa.SolicitudEjecutarPasoFirmaBaremacion) error {
		if solicitud.Paso == puertosbolsa.PasoPrepararFirmaBaremacion &&
			!repositorio.preparacionDeclaradaAntesDelEfecto() {
			return errors.New("prueba: preparacion no persistida antes del efecto")
		}
		return nil
	}
	fachada := nuevaFachadaFlujoFirmaPrueba(t, repositorio, protector, ejecutor, sesiones, reloj, sellador)
	return &entornoFlujoFirmaDurablePrueba{
		fachada: fachada, repositorio: repositorio,
		ejecutor: ejecutor, sesiones: sesiones, reloj: reloj, sellador: sellador,
	}
}

func nuevaFachadaFlujoFirmaPrueba(
	t *testing.T,
	repositorio puertosbolsa.RepositorioFlujosFirmaBaremacion,
	protector puertosbolsa.ProtectorEstadoFlujoFirmaBaremacion,
	ejecutor puertosbolsa.EjecutorPasosFirmaBaremacion,
	sesiones FuenteSesionAutenticadaBaremacion,
	reloj puertosbolsa.Reloj,
	sellador puertosbolsa.SelladorSolicitudBaremacion,
) *FachadaFirmaBaremacionDurable {
	t.Helper()
	fachada, err := NuevaFachadaFirmaBaremacionDurable(
		repositorio,
		protector,
		ejecutor,
		adaptadormemoria.GeneradorReferenciasFlujoFirmaBaremacion{},
		sellador,
		selladorVerificadorFlujoFirmaPrueba{},
		sesiones,
		reloj,
		OpcionesFachadaFirmaBaremacionDurable{DuracionArrendamiento: 10 * time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	return fachada
}

func ordenPrepararFlujoFirmaPrueba(t *testing.T) OrdenPrepararFlujoFirmaBaremacion {
	t.Helper()
	estado, err := puertosbolsa.NuevaCargaProtegida([]byte("estado-servidor-sin-capacidades-v1"))
	if err != nil {
		t.Fatal(err)
	}
	return OrdenPrepararFlujoFirmaBaremacion{
		ClaveIdempotencia:    claveIdempotenciaFlujoFirmaPrueba,
		ProcesoRef:           "proceso-selectivo-001",
		SolicitudRef:         "solicitud-participacion-001",
		BaremacionMeritoRef:  "baremacion-merito-001",
		DecisionRef:          "decision-baremacion-001",
		EstadoTrabajoInicial: estado,
	}
}

type sesionesFlujoFirmaPrueba struct {
	mu       sync.Mutex
	sesiones []SesionAutenticadaBaremacion
}

func (s *sesionesFlujoFirmaPrueba) BuscarSesionesAutenticadasBaremacion(
	context.Context,
) ([]SesionAutenticadaBaremacion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SesionAutenticadaBaremacion(nil), s.sesiones...), nil
}

type selladorVerificadorFlujoFirmaPrueba struct{}

func (selladorVerificadorFlujoFirmaPrueba) SellarSolicitudBaremacion(
	_ context.Context,
	carga puertosbolsa.CargaProtegida,
) (string, error) {
	if carga.Validar() != nil {
		return "", puertosbolsa.ErrCargaProtegidaInvalida
	}
	huella := sha256.Sum256(carga.Revelar())
	return "hmac-sha256:flujo_firma_prueba_v1:" + hex.EncodeToString(huella[:]), nil
}

func (s selladorVerificadorFlujoFirmaPrueba) VerificarEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion,
) error {
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	esperado, err := s.SellarSolicitudBaremacion(ctx, solicitud.RepresentacionCanonica)
	if err != nil || esperado != solicitud.SelloHMAC {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

type repositorioObservadoFlujoFirmaPrueba struct {
	puertosbolsa.RepositorioFlujosFirmaBaremacion

	mu                          sync.Mutex
	preparacionDeclarada        bool
	alterarSiguienteAdquisicion bool
}

func (r *repositorioObservadoFlujoFirmaPrueba) GuardarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	guardado, err := r.RepositorioFlujosFirmaBaremacion.GuardarFlujoFirmaBaremacion(ctx, solicitud)
	if err != nil {
		return guardado, err
	}
	if len(guardado.PuntosControl) == 1 {
		punto := guardado.PuntosControl[0]
		if punto.Paso == puertosbolsa.PasoPrepararFirmaBaremacion &&
			punto.Estado == puertosbolsa.EstadoPuntoControlFirmaDeclarado {
			r.mu.Lock()
			r.preparacionDeclarada = true
			r.mu.Unlock()
		}
	}
	return guardado, err
}

func (r *repositorioObservadoFlujoFirmaPrueba) AdquirirArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error) {
	resultado, err := r.RepositorioFlujosFirmaBaremacion.AdquirirArrendamientoFlujoFirmaBaremacion(ctx, solicitud)
	if err != nil {
		return resultado, err
	}
	r.mu.Lock()
	alterar := r.alterarSiguienteAdquisicion
	r.alterarSiguienteAdquisicion = false
	r.mu.Unlock()
	if !alterar {
		return resultado, nil
	}
	datos, err := resultado.Expediente.EstadoProtegido.DatosPersistencia()
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, err
	}
	datos.Nonce[0] ^= 0xff
	resultado.Expediente.EstadoProtegido, err = puertosbolsa.ImportarEstadoProtegidoFlujoFirmaBaremacion(datos)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, err
	}
	return resultado, nil
}

func (r *repositorioObservadoFlujoFirmaPrueba) preparacionDeclaradaAntesDelEfecto() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.preparacionDeclarada
}

func (r *repositorioObservadoFlujoFirmaPrueba) alterarSobreEnSiguienteArrendamiento() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alterarSiguienteAdquisicion = true
}

type ejecutorFlujoFirmaPrueba struct {
	mu sync.Mutex

	ahora               time.Time
	resultados          map[string]puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion
	invocacionesPorPaso map[puertosbolsa.PasoFlujoFirmaBaremacion]int
	efectosPorPaso      map[puertosbolsa.PasoFlujoFirmaBaremacion]int
	fallarTrasEfecto    map[puertosbolsa.PasoFlujoFirmaBaremacion]bool
	antesDeEjecutar     func(puertosbolsa.SolicitudEjecutarPasoFirmaBaremacion) error
	bloquearEn          puertosbolsa.PasoFlujoFirmaBaremacion
	pasoIniciado        chan struct{}
	continuar           chan struct{}
	bloqueoConsumido    bool
}

func nuevoEjecutorFlujoFirmaPrueba(ahora time.Time) *ejecutorFlujoFirmaPrueba {
	return &ejecutorFlujoFirmaPrueba{
		ahora:               ahora,
		resultados:          make(map[string]puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion),
		invocacionesPorPaso: make(map[puertosbolsa.PasoFlujoFirmaBaremacion]int),
		efectosPorPaso:      make(map[puertosbolsa.PasoFlujoFirmaBaremacion]int),
		fallarTrasEfecto:    make(map[puertosbolsa.PasoFlujoFirmaBaremacion]bool),
	}
}

func (e *ejecutorFlujoFirmaPrueba) EjecutarPasoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudEjecutarPasoFirmaBaremacion,
) (puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion, error) {
	if solicitud.Validar() != nil {
		return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	if e.antesDeEjecutar != nil {
		if err := e.antesDeEjecutar(solicitud); err != nil {
			return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, err
		}
	}
	clave := string(solicitud.Paso) + "\x00" + solicitud.EfectoRef + "\x00" + solicitud.ClaveIdempotenciaHMAC
	e.mu.Lock()
	e.invocacionesPorPaso[solicitud.Paso]++
	if existente, encontrado := e.resultados[clave]; encontrado {
		e.mu.Unlock()
		return clonarResultadoPasoFirmaPrueba(existente), nil
	}
	bloquear := solicitud.Paso == e.bloquearEn && !e.bloqueoConsumido
	if bloquear {
		e.bloqueoConsumido = true
		iniciado := e.pasoIniciado
		continuar := e.continuar
		e.mu.Unlock()
		close(iniciado)
		select {
		case <-continuar:
		case <-ctx.Done():
			return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, ctx.Err()
		}
		e.mu.Lock()
	}

	estadoPlano := append(solicitud.EstadoTrabajo.Revelar(), []byte("|"+string(solicitud.Paso))...)
	estadoTrabajo, err := puertosbolsa.NuevaCargaProtegida(estadoPlano)
	if err != nil {
		e.mu.Unlock()
		return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, err
	}
	resultadoRef := "resultado:" + string(solicitud.Paso) + ":1"
	resultado := puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{
		Paso: solicitud.Paso, EfectoRef: solicitud.EfectoRef, ResultadoRef: resultadoRef,
		HuellaResultadoSHA256: huellaFlujoFirmaPrueba(resultadoRef),
		EstadoTrabajo:         estadoTrabajo,
		EjecutadoEn:           e.ahora,
	}
	if solicitud.Paso == puertosbolsa.PasoPrepararFirmaBaremacion {
		resultado.ProyeccionLanzamiento = &puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{
			FlujoRef: solicitud.FlujoRef, SesionFirmaRef: "sesion-firma:" + solicitud.FlujoRef,
			LanzamientoRef:        "lanzamiento-firma:" + solicitud.FlujoRef,
			CanalLanzamientoClave: "autofirma_local", PreparadaEn: e.ahora, ExpiraEn: e.ahora.Add(5 * time.Minute),
		}
	}
	if solicitud.Paso == puertosbolsa.PasoConfirmarFirmaBaremacion {
		resultado.ResultadoFinal = &puertosbolsa.ResultadoFinalFlujoFirmaBaremacion{
			FlujoRef: solicitud.FlujoRef, DecisionRef: solicitud.DecisionRef,
			DocumentoFirmadoRef:          "documento-firmado:" + solicitud.FlujoRef,
			HuellaDocumentoFirmadoSHA256: huellaFlujoFirmaPrueba("documento:" + solicitud.FlujoRef),
			VersionBaremacion:            2,
			EvidenciaConfirmacionRef:     "evidencia-confirmacion:" + solicitud.FlujoRef,
			HuellaResultadoSHA256:        huellaFlujoFirmaPrueba("confirmacion:" + solicitud.FlujoRef),
			CompletadoEn:                 e.ahora,
		}
	}
	if resultado.ValidarPara(solicitud) != nil {
		e.mu.Unlock()
		return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, errors.New("prueba: resultado de paso invalido")
	}
	e.resultados[clave] = clonarResultadoPasoFirmaPrueba(resultado)
	e.efectosPorPaso[solicitud.Paso]++
	fallar := e.fallarTrasEfecto[solicitud.Paso]
	if fallar {
		e.fallarTrasEfecto[solicitud.Paso] = false
	}
	e.mu.Unlock()
	if fallar {
		return puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion{}, context.DeadlineExceeded
	}
	return resultado, nil
}

func clonarResultadoPasoFirmaPrueba(
	original puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion,
) puertosbolsa.ResultadoEjecutarPasoFirmaBaremacion {
	clon := original
	estado, _ := puertosbolsa.NuevaCargaProtegida(original.EstadoTrabajo.Revelar())
	clon.EstadoTrabajo = estado
	if original.ProyeccionLanzamiento != nil {
		proyeccion := *original.ProyeccionLanzamiento
		clon.ProyeccionLanzamiento = &proyeccion
	}
	if original.ResultadoFinal != nil {
		resultado := *original.ResultadoFinal
		clon.ResultadoFinal = &resultado
	}
	return clon
}

func (e *ejecutorFlujoFirmaPrueba) configurarBloqueo(paso puertosbolsa.PasoFlujoFirmaBaremacion) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.bloquearEn = paso
	e.pasoIniciado = make(chan struct{})
	e.continuar = make(chan struct{})
	e.bloqueoConsumido = false
}

func (e *ejecutorFlujoFirmaPrueba) esperarPasoBloqueado(t *testing.T) {
	t.Helper()
	e.mu.Lock()
	iniciado := e.pasoIniciado
	e.mu.Unlock()
	select {
	case <-iniciado:
	case <-time.After(3 * time.Second):
		t.Fatal("el ejecutor no alcanzo el paso bloqueado")
	}
}

func (e *ejecutorFlujoFirmaPrueba) desbloquearPaso() {
	e.mu.Lock()
	continuar := e.continuar
	e.mu.Unlock()
	close(continuar)
}

func (e *ejecutorFlujoFirmaPrueba) efectosSemanticos(paso puertosbolsa.PasoFlujoFirmaBaremacion) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.efectosPorPaso[paso]
}

func (e *ejecutorFlujoFirmaPrueba) invocaciones(paso puertosbolsa.PasoFlujoFirmaBaremacion) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.invocacionesPorPaso[paso]
}

func (e *ejecutorFlujoFirmaPrueba) totalInvocaciones() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	total := 0
	for _, cantidad := range e.invocacionesPorPaso {
		total += cantidad
	}
	return total
}

func huellaFlujoFirmaPrueba(valor string) string {
	huella := sha256.Sum256([]byte(valor))
	return hex.EncodeToString(huella[:])
}
