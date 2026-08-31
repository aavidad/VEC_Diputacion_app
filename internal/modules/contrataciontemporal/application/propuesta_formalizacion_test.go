package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type transaccionPropuestaFormalizacionPrueba struct {
	resultado ports.ResultadoPropuestaFormalizacion
	err       error
	antes     func(*ports.SolicitudPropuestaFormalizacion)
	recibida  ports.SolicitudPropuestaFormalizacion
	llamadas  int
}

func (t *transaccionPropuestaFormalizacionPrueba) ConfirmarPropuesta(
	_ context.Context,
	solicitud ports.SolicitudPropuestaFormalizacion,
) (ports.ResultadoPropuestaFormalizacion, error) {
	t.llamadas++
	t.recibida = solicitud.Clonar()
	if t.antes != nil {
		t.antes(&solicitud)
	}
	return t.resultado, t.err
}

type contextoPropuestaFinalizadoPrueba struct {
	context.Context
	err  error
	done chan struct{}
}

func nuevoContextoPropuestaFinalizadoPrueba() *contextoPropuestaFinalizadoPrueba {
	return &contextoPropuestaFinalizadoPrueba{
		Context: context.Background(),
		done:    make(chan struct{}),
	}
}

func (c *contextoPropuestaFinalizadoPrueba) Done() <-chan struct{} { return c.done }
func (c *contextoPropuestaFinalizadoPrueba) Err() error            { return c.err }
func (c *contextoPropuestaFinalizadoPrueba) finalizar(err error) {
	c.err = err
	close(c.done)
}

func TestNuevoServicioPropuestaFormalizacionFallaCerrado(t *testing.T) {
	if _, err := NuevoServicioPropuestaFormalizacion(nil); !errors.Is(
		err,
		ErrServicioPropuestaFormalizacionInvalido,
	) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
	var transaccionNula *transaccionPropuestaFormalizacionPrueba
	if _, err := NuevoServicioPropuestaFormalizacion(transaccionNula); !errors.Is(
		err,
		ErrServicioPropuestaFormalizacionInvalido,
	) {
		t.Fatalf("dependencia nula tipada aceptada: %v", err)
	}
}

func TestServicioPropuestaFormalizacionSoloDependeDeCommitLocal(t *testing.T) {
	tipo := reflect.TypeOf(ServicioPropuestaFormalizacion{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "transaccion" ||
		tipo.Field(0).Type != reflect.TypeOf((*ports.TransaccionPropuestaFormalizacion)(nil)).Elem() {
		t.Fatalf("el servicio incorpora una frontera fuera del commit local: %v", tipo)
	}
}

func TestServicioPropuestaFormalizacionNormalizaYConfirma(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
	a, b := solicitud.Anexos[0], solicitud.Anexos[1]
	solicitud.Anexos = []ports.AnexoPropuestaFormalizacion{b, a, a}
	normalizada, err := solicitud.Normalizar()
	if err != nil {
		t.Fatalf("preparar prueba: %v", err)
	}
	transaccion := &transaccionPropuestaFormalizacionPrueba{
		resultado: resultadoPropuestaFormalizacionAplicacionPrueba(normalizada),
	}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)

	resultado, err := servicio.PrepararYConfirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("confirmar propuesta: %v", err)
	}
	if resultado.ValidarPara(normalizada) != nil || transaccion.llamadas != 1 ||
		transaccion.recibida.Validar() != nil || len(transaccion.recibida.Anexos) != 2 ||
		transaccion.recibida.Anexos[0] != a || transaccion.recibida.Anexos[1] != b {
		t.Fatalf("confirmacion local incoherente: resultado=%+v recibida=%+v", resultado, transaccion.recibida)
	}
}

func TestServicioPropuestaFormalizacionDevuelveReplayConfirmado(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
	resultado := resultadoPropuestaFormalizacionAplicacionPrueba(solicitud)
	resultado.Estado = ports.ResultadoPropuestaFormalizacionReplay
	transaccion := &transaccionPropuestaFormalizacionPrueba{resultado: resultado}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)

	obtenido, err := servicio.PrepararYConfirmar(context.Background(), solicitud)
	if err != nil || !obtenido.EsReplayConfirmado() || transaccion.llamadas != 1 {
		t.Fatalf("replay rechazado: resultado=%+v err=%v", obtenido, err)
	}
}

func TestServicioPropuestaFormalizacionNoInvocaPuertoConEntradaInvalida(t *testing.T) {
	transaccion := &transaccionPropuestaFormalizacionPrueba{}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
	solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
	solicitud.ClaveIdempotencia = "invalida"

	resultado, err := servicio.PrepararYConfirmar(context.Background(), solicitud)
	if !resultado.EsCero() || !errors.Is(err, ErrSolicitudPropuestaFormalizacionInvalida) ||
		transaccion.llamadas != 0 {
		t.Fatalf("entrada invalida alcanzo el puerto: resultado=%+v err=%v llamadas=%d", resultado, err, transaccion.llamadas)
	}
}

func TestServicioPropuestaFormalizacionClasificaFallosSinFiltrarDetalle(t *testing.T) {
	casos := []struct {
		nombre   string
		puerto   error
		esperado error
	}{
		{"denegada", ports.ErrOperacionPropuestaFormalizacionDenegada, ErrPropuestaFormalizacionDenegada},
		{"occ", ports.ErrVersionPropuestaFormalizacionEnConflicto, ErrVersionPropuestaFormalizacionEnConflicto},
		{"idempotencia", ports.ErrClavePropuestaFormalizacionUsada, ErrClavePropuestaFormalizacionEnColision},
		{"no_aceptada", ports.ErrResolucionLlamamientoNoAceptada, ErrResolucionFormalizacionNoAceptada},
		{"indisponible", errors.New("detalle privado del adaptador"), ErrPropuestaFormalizacionNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			transaccion := &transaccionPropuestaFormalizacionPrueba{err: caso.puerto}
			servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
			resultado, err := servicio.PrepararYConfirmar(
				context.Background(),
				solicitudPropuestaFormalizacionAplicacionPrueba(),
			)
			if !resultado.EsCero() || !errors.Is(err, caso.esperado) || errors.Is(err, caso.puerto) && caso.nombre == "indisponible" {
				t.Fatalf("clasificacion inesperada: resultado=%+v err=%v", resultado, err)
			}
		})
	}
}

func TestServicioPropuestaFormalizacionRechazaResultadoMasError(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
	transaccion := &transaccionPropuestaFormalizacionPrueba{
		resultado: resultadoPropuestaFormalizacionAplicacionPrueba(solicitud),
		err:       errors.New("resultado contradictorio"),
	}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)

	resultado, err := servicio.PrepararYConfirmar(context.Background(), solicitud)
	if !resultado.EsCero() || !errors.Is(err, ErrResultadoPropuestaFormalizacionNoConfiable) {
		t.Fatalf("resultado+error aceptado: resultado=%+v err=%v", resultado, err)
	}
}

func TestServicioPropuestaFormalizacionPriorizaCancelacionAntesYDespuesDelPuerto(t *testing.T) {
	t.Run("antes", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		transaccion := &transaccionPropuestaFormalizacionPrueba{}
		servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
		solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
		solicitud.ClaveIdempotencia = "tambien-invalida"
		resultado, err := servicio.PrepararYConfirmar(
			ctx,
			solicitud,
		)
		if !resultado.EsCero() || !errors.Is(err, context.Canceled) || transaccion.llamadas != 0 {
			t.Fatalf("cancelacion previa no priorizada: resultado=%+v err=%v", resultado, err)
		}
	})

	for _, fallo := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run("despues_"+fallo.Error(), func(t *testing.T) {
			ctx := nuevoContextoPropuestaFinalizadoPrueba()
			solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
			transaccion := &transaccionPropuestaFormalizacionPrueba{
				resultado: resultadoPropuestaFormalizacionAplicacionPrueba(solicitud),
				antes: func(*ports.SolicitudPropuestaFormalizacion) {
					ctx.finalizar(fallo)
				},
			}
			servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
			resultado, err := servicio.PrepararYConfirmar(ctx, solicitud)
			if !resultado.EsCero() || !errors.Is(err, fallo) {
				t.Fatalf("resultado valido sobrevivio a cancelacion: resultado=%+v err=%v", resultado, err)
			}
		})
	}
}

func TestServicioPropuestaFormalizacionPriorizaCancelacionConErrorDelPuerto(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	transaccion := &transaccionPropuestaFormalizacionPrueba{
		err: errors.New("fallo privado"),
		antes: func(*ports.SolicitudPropuestaFormalizacion) {
			cancelar()
		},
	}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
	resultado, err := servicio.PrepararYConfirmar(
		ctx,
		solicitudPropuestaFormalizacionAplicacionPrueba(),
	)
	if !resultado.EsCero() || !errors.Is(err, context.Canceled) {
		t.Fatalf("error del puerto oculto cancelacion: resultado=%+v err=%v", resultado, err)
	}
}

func TestServicioPropuestaFormalizacionRevalidaSalidaYClonaFronteras(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionAplicacionPrueba()
	original := solicitud.Anexos[0]
	resultadoPuerto := resultadoPropuestaFormalizacionAplicacionPrueba(solicitud)
	transaccion := &transaccionPropuestaFormalizacionPrueba{
		resultado: resultadoPuerto,
		antes: func(recibida *ports.SolicitudPropuestaFormalizacion) {
			recibida.Anexos[0].DocumentoRef = "anexo:mutado-por-puerto"
		},
	}
	servicio := nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)

	resultado, err := servicio.PrepararYConfirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("confirmacion rechazada: %v", err)
	}
	if solicitud.Anexos[0] != original {
		t.Fatal("el puerto pudo mutar la entrada del llamador")
	}
	transaccion.resultado.Solicitud.Anexos[0].DocumentoRef = "anexo:mutado-tras-retorno"
	if resultado.Solicitud.Anexos[0] != original {
		t.Fatal("la salida conserva alias con el resultado del puerto")
	}

	alterado := resultadoPropuestaFormalizacionAplicacionPrueba(solicitud)
	alterado.Solicitud.PlanFirma.Version++
	transaccion = &transaccionPropuestaFormalizacionPrueba{resultado: alterado}
	servicio = nuevoServicioPropuestaFormalizacionAplicacionPrueba(t, transaccion)
	resultado, err = servicio.PrepararYConfirmar(context.Background(), solicitud)
	if !resultado.EsCero() || !errors.Is(err, ErrResultadoPropuestaFormalizacionNoConfiable) {
		t.Fatalf("salida no ligada aceptada: resultado=%+v err=%v", resultado, err)
	}
}

func nuevoServicioPropuestaFormalizacionAplicacionPrueba(
	t *testing.T,
	transaccion ports.TransaccionPropuestaFormalizacion,
) *ServicioPropuestaFormalizacion {
	t.Helper()
	servicio, err := NuevoServicioPropuestaFormalizacion(transaccion)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func solicitudPropuestaFormalizacionAplicacionPrueba() ports.SolicitudPropuestaFormalizacion {
	return ports.SolicitudPropuestaFormalizacion{
		ClaveIdempotencia:                "828f47a6-5d2b-4c10-aa11-1234567890ab",
		OrganizacionRef:                  "organizacion:aplicacion-formalizacion",
		ExpedienteRef:                    "expediente:aplicacion-formalizacion",
		LlamamientoRef:                   "llamamiento:aplicacion-formalizacion",
		ResolucionLlamamientoAceptadaRef: "resolucion:aplicacion-aceptada",
		ReciboResolucionAceptadaRef:      "recibo:aplicacion-aceptada",
		VersionEsperada:                  21,
		TipoFormalizacion:                snapshotPropuestaFormalizacionAplicacionPrueba("tipo:formalizacion", "7"),
		Plantilla:                        snapshotPropuestaFormalizacionAplicacionPrueba("plantilla:formalizacion", "8"),
		Anexos: []ports.AnexoPropuestaFormalizacion{
			anexoPropuestaFormalizacionAplicacionPrueba("a", "9", 4096),
			anexoPropuestaFormalizacionAplicacionPrueba("b", "a", 8192),
		},
		PoliticaFirma: snapshotPropuestaFormalizacionAplicacionPrueba("politica:firma", "b"),
		PlanFirma:     snapshotPropuestaFormalizacionAplicacionPrueba("plan:firma", "c"),
	}
}

func snapshotPropuestaFormalizacionAplicacionPrueba(
	referencia string,
	digito string,
) ports.SnapshotGobernadoFormalizacion {
	return ports.SnapshotGobernadoFormalizacion{
		Referencia: referencia, Version: 9, HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func anexoPropuestaFormalizacionAplicacionPrueba(
	sufijo string,
	digito string,
	tamano uint64,
) ports.AnexoPropuestaFormalizacion {
	return ports.AnexoPropuestaFormalizacion{
		DocumentoRef: "anexo:aplicacion-" + sufijo, Version: 5,
		HuellaSHA256: strings.Repeat(digito, 64), TamanoBytes: tamano,
	}
}

func resultadoPropuestaFormalizacionAplicacionPrueba(
	solicitud ports.SolicitudPropuestaFormalizacion,
) ports.ResultadoPropuestaFormalizacion {
	return ports.ResultadoPropuestaFormalizacion{
		Solicitud: solicitud.Clonar(), PropuestaRef: "propuesta:aplicacion-local",
		ReciboLocalRef: "recibo:aplicacion-local", AuditoriaRef: "auditoria:aplicacion-local",
		VersionResultante: solicitud.VersionEsperada + 1,
		ConfirmadaEn:      time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC),
		Estado:            ports.ResultadoPropuestaFormalizacionConfirmado,
	}
}
