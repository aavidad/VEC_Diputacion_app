package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	clavePeticionSeleccionPrueba  = "vec.contratacion-temporal.integracion-bolsa-peticion/v1"
	claveRespuestaSeleccionPrueba = "vec.contratacion-temporal.integracion-bolsa-respuesta/v1"
	claveIdempotenciaSeleccion    = "018f47a2-6b31-4c80-8a95-4d2e707c5a11"
)

func TestSeleccionLlamamientoOrquestaPrimeraPosicionEvaluable(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	recibo, err := e.servicio.SeleccionarYLlamar(
		context.Background(),
		SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
	)
	if err != nil {
		t.Fatalf("seleccionar y llamar: %v", err)
	}
	if !recibo.PropuestaGenerada || recibo.OrdenSeleccionado != 2 ||
		recibo.Necesidad != e.preparador.necesidad ||
		recibo.Bolsa != e.preparador.bolsa || recibo.Politica != e.preparador.politica ||
		recibo.LlamamientoRef != "llamamiento:seleccion:001" {
		t.Fatalf("recibo minimizado inesperado: %#v", recibo)
	}
	datos, err := e.llamamientos.ultimo.DatosEn(e.instante)
	if err != nil || datos.MaximaPosicionEvaluable != e.ordenes.total ||
		datos.TotalPosicionesOrden != e.ordenes.total {
		t.Fatalf("Bolsa no recibió el orden completo como límite evaluable: %+v err=%v", datos, err)
	}
	if e.disponibilidad.llamadas != 1 || e.ordenes.llamadas != 1 ||
		e.llamamientos.llamadas != 1 || e.llamamientos.creaciones != 1 {
		t.Fatalf("fronteras invocadas en orden incorrecto: disponibilidad=%d orden=%d llamamiento=%d creaciones=%d",
			e.disponibilidad.llamadas, e.ordenes.llamadas,
			e.llamamientos.llamadas, e.llamamientos.creaciones)
	}
	tipo := reflect.TypeOf(SolicitudSeleccionLlamamiento{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "ClaveIdempotencia" {
		t.Fatalf("la entrada permite elegir bolsa/candidato/posición: %v", tipo)
	}
}

func TestSeleccionLlamamientoFallaCerradoSinBolsaODisponibilidad(t *testing.T) {
	casos := []struct {
		nombre          string
		bolsaEncontrada bool
		disponible      bool
		cantidad        uint32
		cantidadExacta  bool
	}{
		{"sin bolsa", false, false, 0, true},
		{"sin disponibilidad", true, false, 0, true},
		{"cantidad inexacta", true, true, 1, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.disponibilidad.bolsaEncontrada = caso.bolsaEncontrada
			e.disponibilidad.disponible = caso.disponible
			e.disponibilidad.cantidad = caso.cantidad
			e.disponibilidad.cantidadExacta = caso.cantidadExacta
			_, err := e.servicio.SeleccionarYLlamar(
				context.Background(),
				SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
			)
			if !errors.Is(err, ErrSeleccionLlamamientoNoDisponible) ||
				e.ordenes.llamadas != 0 || e.llamamientos.llamadas != 0 {
				t.Fatalf("la ausencia avanzó al efecto: orden=%d llamamiento=%d err=%v",
					e.ordenes.llamadas, e.llamamientos.llamadas, err)
			}
		})
	}
}

func TestSeleccionLlamamientoRechazaOrdenAusenteOIncompleto(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		generada bool
		completa bool
		total    uint32
	}{
		{"ausente", false, false, 0},
		{"incompleto", true, false, 3},
		{"sin posiciones", true, true, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.ordenes.generada, e.ordenes.completa, e.ordenes.total =
				caso.generada, caso.completa, caso.total
			_, err := e.servicio.SeleccionarYLlamar(
				context.Background(),
				SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
			)
			if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
				e.llamamientos.llamadas != 0 {
				t.Fatalf("orden no completa produjo llamamiento: llamadas=%d err=%v",
					e.llamamientos.llamadas, err)
			}
		})
	}
}

func TestSeleccionLlamamientoRechazaEvidenciaCruzada(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.disponibilidad.sello = selloSeleccionPrueba('2')
	_, err := e.servicio.SeleccionarYLlamar(
		context.Background(),
		SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
	)
	if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
		e.ordenes.llamadas != 0 || e.llamamientos.llamadas != 0 {
		t.Fatalf("evidencia de orden cruzada con disponibilidad avanzó: err=%v", err)
	}
}

func TestSeleccionLlamamientoRespetaCancelacionEntrePasos(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	ctx, cancelar := context.WithCancel(context.Background())
	e.disponibilidad.cancelar = cancelar
	_, err := e.servicio.SeleccionarYLlamar(
		ctx,
		SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
	)
	if !errors.Is(err, context.Canceled) || e.ordenes.llamadas != 0 ||
		e.llamamientos.llamadas != 0 {
		t.Fatalf("cancelación avanzó: orden=%d llamamiento=%d err=%v",
			e.ordenes.llamadas, e.llamamientos.llamadas, err)
	}
}

func TestSeleccionLlamamientoRechazaReciboCruzado(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.llamamientos.cruzarRecibo = true
	_, err := e.servicio.SeleccionarYLlamar(
		context.Background(),
		SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
	)
	if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) {
		t.Fatalf("recibo de otro orden aceptado: %v", err)
	}
}

func TestSeleccionLlamamientoReplayExactoUsaIdempotenciaDelPuerto(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	primero, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer intento: %v", err)
	}
	segundo, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if primero != segundo || e.llamamientos.llamadas != 2 ||
		e.llamamientos.creaciones != 1 {
		t.Fatalf("replay no conservó recibo/efecto: igual=%v llamadas=%d creaciones=%d",
			primero == segundo, e.llamamientos.llamadas, e.llamamientos.creaciones)
	}
}

func TestSeleccionLlamamientoColisionFallaSinSegundoEfecto(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	if _, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud); err != nil {
		t.Fatalf("primer intento: %v", err)
	}
	e.preparador.alternarPolitica = true
	_, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if !errors.Is(err, ErrSeleccionLlamamientoNoDisponible) ||
		e.llamamientos.creaciones != 1 || strings.Contains(err.Error(), "detalle privado") {
		t.Fatalf("colisión no quedó cerrada y opaca: creaciones=%d err=%v",
			e.llamamientos.creaciones, err)
	}
}

func TestSeleccionLlamamientoOcultaErrorPrivadoYRechazaLimite(t *testing.T) {
	t.Run("error privado", func(t *testing.T) {
		e := nuevoEscenarioSeleccionLlamamiento(t)
		e.disponibilidad.err = errors.New("detalle privado: dsn y persona")
		_, err := e.servicio.SeleccionarYLlamar(
			context.Background(),
			SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
		)
		if !errors.Is(err, ErrSeleccionLlamamientoNoDisponible) ||
			strings.Contains(err.Error(), "dsn") || e.llamamientos.llamadas != 0 {
			t.Fatalf("error privado expuesto o falso llamamiento: %v", err)
		}
	})
	t.Run("limite", func(t *testing.T) {
		e := nuevoEscenarioSeleccionLlamamiento(t)
		e.preparador.maximoResultados = ports.MaximoElementosIntegracionBolsa + 1
		_, err := e.servicio.SeleccionarYLlamar(
			context.Background(),
			SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion},
		)
		if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
			e.disponibilidad.llamadas != 0 || e.llamamientos.llamadas != 0 {
			t.Fatalf("límite inválido alcanzó conectores: %v", err)
		}
	})
}

func TestSeleccionLlamamientoRechazaDependenciasYEntradaNoConfiable(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	casosConstructor := []struct {
		nombre         string
		preparador     ports.PreparadorSeleccionLlamamiento
		disponibilidad ports.ConsultaDisponibilidadBolsa
		ordenes        ports.PreparadorOrdenBolsa
		llamamientos   ports.GestorLlamamientosBolsa
		verificador    *ports.VerificadorEvidenciaIntegracionBolsa
		reloj          ports.Reloj
	}{
		{"preparador nulo", nil, e.disponibilidad, e.ordenes, e.llamamientos, e.servicio.verificador, e.servicio.reloj},
		{"preparador nulo tipado", (*preparadorSeleccionLlamamientoPrueba)(nil), e.disponibilidad, e.ordenes, e.llamamientos, e.servicio.verificador, e.servicio.reloj},
		{"disponibilidad nula", e.preparador, nil, e.ordenes, e.llamamientos, e.servicio.verificador, e.servicio.reloj},
		{"orden nula", e.preparador, e.disponibilidad, nil, e.llamamientos, e.servicio.verificador, e.servicio.reloj},
		{"llamamiento nulo", e.preparador, e.disponibilidad, e.ordenes, nil, e.servicio.verificador, e.servicio.reloj},
		{"verificador nulo", e.preparador, e.disponibilidad, e.ordenes, e.llamamientos, nil, e.servicio.reloj},
		{"reloj nulo", e.preparador, e.disponibilidad, e.ordenes, e.llamamientos, e.servicio.verificador, nil},
	}
	for _, caso := range casosConstructor {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := NuevoServicioSeleccionLlamamiento(
				caso.preparador, caso.disponibilidad, caso.ordenes,
				caso.llamamientos, caso.verificador, caso.reloj,
			)
			if !errors.Is(err, ErrServicioSeleccionLlamamientoInvalido) {
				t.Fatalf("dependencia inválida aceptada: %v", err)
			}
		})
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	var servicioNulo *ServicioSeleccionLlamamiento
	casosEntrada := []struct {
		nombre    string
		servicio  *ServicioSeleccionLlamamiento
		ctx       context.Context
		intencion string
		esperado  error
	}{
		{"servicio nulo", servicioNulo, context.Background(), claveIdempotenciaSeleccion, ErrServicioSeleccionLlamamientoInvalido},
		{"contexto nulo", e.servicio, nil, claveIdempotenciaSeleccion, ErrServicioSeleccionLlamamientoInvalido},
		{"uuid no canonica", e.servicio, context.Background(), "candidato:42", ErrSolicitudSeleccionLlamamientoInvalida},
		{"cancelada antes de empezar", e.servicio, cancelado, claveIdempotenciaSeleccion, context.Canceled},
	}
	for _, caso := range casosEntrada {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := caso.servicio.SeleccionarYLlamar(
				caso.ctx, SolicitudSeleccionLlamamiento{ClaveIdempotencia: caso.intencion},
			)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("entrada no cerrada: %v", err)
			}
		})
	}
}

func TestSeleccionLlamamientoCancelaAntesDeCadaFronteraSiguiente(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*escenarioSeleccionLlamamientoPrueba, context.CancelFunc)
		llamadas [3]int
	}{
		{"tras preparar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "consulta", c
		}, [3]int{}},
		{"tras consultar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.disponibilidad.cancelar = c }, [3]int{1, 0, 0}},
		{"tras preparar orden", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "orden", c
		}, [3]int{1, 0, 0}},
		{"tras recibir orden", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.ordenes.cancelar = c }, [3]int{1, 1, 0}},
		{"tras preparar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "llamamiento", c
		}, [3]int{1, 1, 0}},
		{"tras solicitar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.llamamientos.cancelar = c }, [3]int{1, 1, 1}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			ctx, cancelar := context.WithCancel(context.Background())
			caso.preparar(&e, cancelar)
			_, err := e.servicio.SeleccionarYLlamar(ctx, SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion})
			obtenidas := [3]int{e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			if !errors.Is(err, context.Canceled) || obtenidas != caso.llamadas {
				t.Fatalf("cancelación avanzó: llamadas=%v err=%v", obtenidas, err)
			}
		})
	}

	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.disponibilidad.err = context.DeadlineExceeded
	if _, err := e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("la frontera perdió la causa de cancelación: %v", err)
	}
}

func TestSeleccionLlamamientoCierraFallosDeTodasLasFronteras(t *testing.T) {
	casos := []struct {
		nombre     string
		configurar func(*escenarioSeleccionLlamamientoPrueba)
		llamadas   [3]int
	}{
		{"preparar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "consulta" }, [3]int{}},
		{"preparar orden", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "orden" }, [3]int{1, 0, 0}},
		{"orden", func(e *escenarioSeleccionLlamamientoPrueba) { e.ordenes.err = errors.New("detalle privado") }, [3]int{1, 1, 0}},
		{"preparar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "llamamiento" }, [3]int{1, 1, 0}},
		{"llamamiento", func(e *escenarioSeleccionLlamamientoPrueba) { e.llamamientos.err = errors.New("detalle privado") }, [3]int{1, 1, 1}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			caso.configurar(&e)
			_, err := e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion})
			obtenidas := [3]int{e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			if !errors.Is(err, ErrSeleccionLlamamientoNoDisponible) || obtenidas != caso.llamadas || strings.Contains(err.Error(), "detalle") {
				t.Fatalf("fallo no quedó cerrado: llamadas=%v err=%v", obtenidas, err)
			}
		})
	}
}

func TestSeleccionLlamamientoRechazaContextosDesligadosYRelojRetrogrado(t *testing.T) {
	casos := []struct {
		nombre, operacion string
		mutar             func(*ports.DatosContextoPeticionIntegracionBolsa)
		ordenes           int
	}{
		{"organizacion de otra orden", "operacion:orden:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) { d.OrganizacionRef = "organizacion:ajena" }, 0},
		{"operacion de orden repetida", "operacion:orden:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) { d.OperacionRef = "operacion:disponibilidad:001" }, 0},
		{"correlacion de otro llamamiento", "operacion:llamamiento:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) { d.CorrelacionRef = "correlacion:ajena" }, 1},
		{"recurso de otro llamamiento", "operacion:llamamiento:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) {
			d.Recurso = referenciaSeleccionPrueba("orden:bolsa:ajena", '1')
		}, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.preparador.mutarContexto = func(operacion string, datos *ports.DatosContextoPeticionIntegracionBolsa) {
				if operacion == caso.operacion {
					caso.mutar(datos)
				}
			}
			_, err := e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion})
			if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
				e.ordenes.llamadas != caso.ordenes || e.llamamientos.llamadas != 0 {
				t.Fatalf("contextos desligados avanzaron: orden=%d llamada=%d err=%v", e.ordenes.llamadas, e.llamamientos.llamadas, err)
			}
		})
	}

	base := time.Date(2026, 8, 22, 10, 3, 0, 0, time.UTC)
	for _, secuencia := range [][]time.Time{
		{base, base.Add(-time.Minute)},
		{base, base.Add(time.Minute), base},
		{base, base, base.Add(time.Minute), base},
		{base, base, base, base.Add(time.Minute), base},
		{base, base, base, base, base.Add(time.Minute), base},
	} {
		e := nuevoEscenarioSeleccionLlamamiento(t)
		e.servicio.reloj = &relojSecuenciaSeleccionPrueba{instantes: secuencia}
		_, err := e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion})
		if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) {
			t.Fatalf("reloj retrógrado aceptado (%d lecturas): %v", len(secuencia), err)
		}
	}
}

type escenarioSeleccionLlamamientoPrueba struct {
	servicio       *ServicioSeleccionLlamamiento
	preparador     *preparadorSeleccionLlamamientoPrueba
	disponibilidad *disponibilidadSeleccionPrueba
	ordenes        *ordenSeleccionPrueba
	llamamientos   *llamamientoSeleccionPrueba
	instante       time.Time
}

func nuevoEscenarioSeleccionLlamamiento(t *testing.T) escenarioSeleccionLlamamientoPrueba {
	t.Helper()
	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	emisor, err := ports.NuevoEmisorContextoPeticionIntegracionBolsa(
		"autoridad:contratacion-temporal", clavePeticionSeleccionPrueba,
		selladorContextoSeleccionPrueba{},
	)
	if err != nil {
		t.Fatalf("crear emisor: %v", err)
	}
	preparador := &preparadorSeleccionLlamamientoPrueba{
		base: base, emisor: emisor,
		necesidad:        referenciaSeleccionPrueba("necesidad:cobertura:001", 'a'),
		bolsa:            referenciaSeleccionPrueba("bolsa:vigente:001", 'b'),
		politica:         referenciaSeleccionPrueba("politica:llamamiento:001", 'c'),
		maximoResultados: 3,
	}
	disponibilidad := &disponibilidadSeleccionPrueba{
		base: base, bolsa: preparador.bolsa, bolsaEncontrada: true,
		disponible: true, cantidad: 3, cantidadExacta: true,
		sello: selloSeleccionPrueba('1'),
	}
	ordenes := &ordenSeleccionPrueba{
		base: base, generada: true, completa: true, total: 3,
	}
	llamamientos := &llamamientoSeleccionPrueba{base: base}
	verificador, err := ports.NuevoVerificadorEvidenciaIntegracionBolsa(
		"autoridad:bolsa", claveRespuestaSeleccionPrueba, nil,
		verificadorEvidenciaSeleccionPrueba{},
	)
	if err != nil {
		t.Fatalf("crear verificador: %v", err)
	}
	instante := base.Add(3 * time.Minute)
	servicio, err := NuevoServicioSeleccionLlamamiento(
		preparador, disponibilidad, ordenes, llamamientos, verificador,
		relojSeleccionLlamamientoPrueba{instante: instante},
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return escenarioSeleccionLlamamientoPrueba{
		servicio: servicio, preparador: preparador, disponibilidad: disponibilidad,
		ordenes: ordenes, llamamientos: llamamientos, instante: instante,
	}
}

type relojSeleccionLlamamientoPrueba struct{ instante time.Time }

func (r relojSeleccionLlamamientoPrueba) Ahora() time.Time { return r.instante }

type relojSecuenciaSeleccionPrueba struct {
	instantes []time.Time
	indice    int
}

func (r *relojSecuenciaSeleccionPrueba) Ahora() time.Time {
	if r.indice >= len(r.instantes) {
		return r.instantes[len(r.instantes)-1]
	}
	instante := r.instantes[r.indice]
	r.indice++
	return instante
}

type selladorContextoSeleccionPrueba struct{}

func (selladorContextoSeleccionPrueba) SellarDatos(context.Context, []byte) (string, error) {
	return "hmac-sha256:" + clavePeticionSeleccionPrueba + ":" + strings.Repeat("a", 64), nil
}

type verificadorEvidenciaSeleccionPrueba struct{}

func (verificadorEvidenciaSeleccionPrueba) VerificarDatos(
	ctx context.Context,
	clave string,
	material []byte,
	sello string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	esperado := byte(0)
	switch {
	case bytes.Contains(material, []byte("disponibilidad_volatil")):
		esperado = '1'
	case bytes.Contains(material, []byte("recibo_orden")):
		esperado = '2'
	case bytes.Contains(material, []byte("recibo_llamamiento")):
		esperado = '3'
	}
	if clave != claveRespuestaSeleccionPrueba || esperado == 0 ||
		sello != selloSeleccionPrueba(esperado) {
		return errors.New("detalle privado: evidencia cruzada")
	}
	return nil
}

type preparadorSeleccionLlamamientoPrueba struct {
	base              time.Time
	emisor            *ports.EmisorContextoPeticionIntegracionBolsa
	necesidad         ports.ReferenciaVersionadaIntegracionBolsa
	bolsa             ports.ReferenciaVersionadaIntegracionBolsa
	politica          ports.ReferenciaVersionadaIntegracionBolsa
	maximoResultados  uint32
	ordenesPreparadas int
	alternarPolitica  bool
	cancelarEn        string
	cancelar          context.CancelFunc
	mutarContexto     func(string, *ports.DatosContextoPeticionIntegracionBolsa)
	fallarEn          string
}

func (p *preparadorSeleccionLlamamientoPrueba) PrepararConsultaDisponibilidad(
	ctx context.Context,
	_ string,
) (ports.SolicitudDisponibilidadBolsa, error) {
	contexto, err := p.contexto(
		ctx, "operacion:disponibilidad:001", p.necesidad,
		referenciaSeleccionPrueba("accion:disponibilidad:001", 'd'),
	)
	if p.cancelarEn == "consulta" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "consulta" {
		err = errors.New("detalle privado")
	}
	return ports.SolicitudDisponibilidadBolsa{
		Contexto: contexto, Necesidad: p.necesidad, CategoriaRef: "categoria:auxiliar",
		MaximoResultados: p.maximoResultados,
	}, err
}

func (p *preparadorSeleccionLlamamientoPrueba) PrepararOrdenCompleto(
	ctx context.Context,
	_ string,
	resultado ports.ResultadoDisponibilidadBolsa,
) (ports.ComandoPrepararOrdenBolsa, error) {
	p.ordenesPreparadas++
	politica := p.politica
	if p.alternarPolitica && p.ordenesPreparadas > 1 {
		politica = referenciaSeleccionPrueba("politica:llamamiento:002", '9')
	}
	contexto, err := p.contexto(
		ctx, "operacion:orden:001", resultado.Bolsa,
		referenciaSeleccionPrueba("accion:orden:001", 'e'),
	)
	if p.cancelarEn == "orden" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "orden" {
		err = errors.New("detalle privado")
	}
	return ports.ComandoPrepararOrdenBolsa{
		Contexto: contexto, Necesidad: resultado.Necesidad, Bolsa: resultado.Bolsa,
		Politica: politica, MaximoPosiciones: p.maximoResultados,
	}, err
}

func (p *preparadorSeleccionLlamamientoPrueba) PrepararContextoLlamamiento(
	ctx context.Context,
	_ string,
	recibo ports.ReciboOrdenBolsa,
) (ports.ContextoPeticionIntegracionBolsa, error) {
	contexto, err := p.contexto(ctx, "operacion:llamamiento:001", recibo.Orden, recibo.AccionLlamamiento)
	if p.cancelarEn == "llamamiento" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "llamamiento" {
		err = errors.New("detalle privado")
	}
	return contexto, err
}

func (p *preparadorSeleccionLlamamientoPrueba) contexto(
	ctx context.Context,
	operacion string,
	recurso ports.ReferenciaVersionadaIntegracionBolsa,
	accion ports.ReferenciaVersionadaIntegracionBolsa,
) (ports.ContextoPeticionIntegracionBolsa, error) {
	datos := ports.DatosContextoPeticionIntegracionBolsa{
		OperacionRef: operacion, OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal:001", VersionExpediente: 7,
		CorrelacionRef:       "correlacion:seleccion:001",
		ContratoVersion:      ports.VersionContratoIntegracionBolsa,
		AutoridadSolicitante: "autoridad:contratacion-temporal",
		Autorizacion:         referenciaSeleccionPrueba("autorizacion:seleccion:001", 'f'),
		Accion:               accion, Recurso: recurso,
		Finalidad:    referenciaSeleccionPrueba("finalidad:seleccion:001", '6'),
		SolicitadaEn: p.base, ValidaHasta: p.base.Add(10 * time.Minute),
	}
	if p.mutarContexto != nil {
		p.mutarContexto(operacion, &datos)
	}
	return p.emisor.Emitir(ctx, datos, p.base)
}

type disponibilidadSeleccionPrueba struct {
	base            time.Time
	bolsa           ports.ReferenciaVersionadaIntegracionBolsa
	bolsaEncontrada bool
	disponible      bool
	cantidad        uint32
	cantidadExacta  bool
	sello           string
	cancelar        context.CancelFunc
	err             error
	llamadas        int
}

func (d *disponibilidadSeleccionPrueba) ConsultarDisponibilidad(
	ctx context.Context,
	solicitud ports.SolicitudDisponibilidadBolsa,
) (ports.ResultadoDisponibilidadBolsa, error) {
	d.llamadas++
	if d.err != nil {
		return ports.ResultadoDisponibilidadBolsa{}, d.err
	}
	datos, _ := solicitud.Contexto.DatosEn(d.base.Add(3 * time.Minute))
	resultado := ports.ResultadoDisponibilidadBolsa{
		OperacionRef: datos.OperacionRef, OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		CorrelacionRef: datos.CorrelacionRef, Necesidad: solicitud.Necesidad,
		CategoriaRef:    solicitud.CategoriaRef,
		Resultado:       referenciaSeleccionPrueba("resultado:disponibilidad:001", '7'),
		BolsaEncontrada: d.bolsaEncontrada, Disponible: d.disponible,
		CantidadDisponible: d.cantidad, CantidadExacta: d.cantidadExacta,
		Procedencia: procedenciaSeleccionPrueba(d.base, "respuesta:disponibilidad:001", "evidencia:disponibilidad:001", d.sello),
	}
	if d.bolsaEncontrada {
		resultado.Bolsa = d.bolsa
	}
	if d.cancelar != nil {
		d.cancelar()
	}
	return resultado, nil
}

type ordenSeleccionPrueba struct {
	base     time.Time
	generada bool
	completa bool
	total    uint32
	cancelar context.CancelFunc
	err      error
	llamadas int
}

func (o *ordenSeleccionPrueba) PrepararOrden(
	_ context.Context,
	comando ports.ComandoPrepararOrdenBolsa,
) (ports.ReciboOrdenBolsa, error) {
	o.llamadas++
	if o.err != nil {
		return ports.ReciboOrdenBolsa{}, o.err
	}
	datos, _ := comando.Contexto.DatosEn(o.base.Add(3 * time.Minute))
	recibo := ports.ReciboOrdenBolsa{
		OperacionRef: datos.OperacionRef, OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		CorrelacionRef: datos.CorrelacionRef, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Politica: comando.Politica,
		Resultado:     referenciaSeleccionPrueba("resultado:orden:001", '7'),
		OrdenGenerada: o.generada, OrdenCompleta: o.completa,
		ReciboRef: "recibo:orden:001", AuditoriaRef: "auditoria:orden:001",
		EventoRef: "evento:orden:001", ConfirmadaEn: o.base.Add(time.Minute),
		Procedencia: procedenciaSeleccionPrueba(o.base, "respuesta:orden:001", "evidencia:orden:001", selloSeleccionPrueba('2')),
	}
	if o.generada {
		recibo.Orden = referenciaSeleccionPrueba("orden:bolsa:001", '8')
		recibo.AccionLlamamiento = referenciaSeleccionPrueba("accion:llamamiento:001", '5')
		recibo.TotalPosiciones = o.total
	}
	if o.cancelar != nil {
		o.cancelar()
	}
	return recibo, nil
}

type identidadComandoLlamamientoPrueba struct {
	operacion string
	necesidad ports.ReferenciaVersionadaIntegracionBolsa
	bolsa     ports.ReferenciaVersionadaIntegracionBolsa
	orden     ports.ReferenciaVersionadaIntegracionBolsa
	politica  ports.ReferenciaVersionadaIntegracionBolsa
	maximo    uint32
}

type llamamientoSeleccionPrueba struct {
	base         time.Time
	llamadas     int
	creaciones   int
	cruzarRecibo bool
	cancelar     context.CancelFunc
	err          error
	identidad    *identidadComandoLlamamientoPrueba
	recibo       ports.ReciboSolicitudLlamamientoBolsa
	ultimo       ports.ComandoSolicitarLlamamientoBolsa
}

func (l *llamamientoSeleccionPrueba) SolicitarLlamamiento(
	_ context.Context,
	comando ports.ComandoSolicitarLlamamientoBolsa,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	l.llamadas++
	l.ultimo = comando
	if l.err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, l.err
	}
	datos, err := comando.DatosEn(l.base.Add(3 * time.Minute))
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	contexto, _ := datos.Contexto.DatosEn(l.base.Add(3 * time.Minute))
	identidad := identidadComandoLlamamientoPrueba{
		operacion: contexto.OperacionRef, necesidad: datos.Necesidad,
		bolsa: datos.Bolsa, orden: datos.Orden, politica: datos.Politica,
		maximo: datos.MaximaPosicionEvaluable,
	}
	if l.identidad != nil {
		if *l.identidad != identidad {
			return ports.ReciboSolicitudLlamamientoBolsa{}, errors.New("detalle privado: colision semantica")
		}
		return l.recibo, nil
	}
	seudonimo, _ := ports.NuevoSeudonimoSeleccionBolsa(
		"hmac-sha256:vec.contratacion-temporal.seleccion/v1:" + strings.Repeat("9", 64),
	)
	recibo := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: contexto.OperacionRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, Necesidad: datos.Necesidad,
		Bolsa: datos.Bolsa, Orden: datos.Orden, Politica: datos.Politica,
		Resultado:         referenciaSeleccionPrueba("resultado:llamamiento:001", '7'),
		PropuestaGenerada: true,
		Propuesta:         referenciaSeleccionPrueba("propuesta:llamamiento:001", '4'),
		AccionEvento:      referenciaSeleccionPrueba("accion:evento:001", '3'),
		LlamamientoRef:    "llamamiento:seleccion:001", SeleccionRef: seudonimo,
		RetencionSeleccion: referenciaSeleccionPrueba("retencion:seleccion:001", '2'),
		OrdenSeleccionado:  2, ReciboRef: "recibo:llamamiento:001",
		AuditoriaRef: "auditoria:llamamiento:001", EventoRef: "evento:llamamiento:001",
		ConfirmadaEn: l.base.Add(time.Minute),
		Procedencia:  procedenciaSeleccionPrueba(l.base, "respuesta:llamamiento:001", "evidencia:llamamiento:001", selloSeleccionPrueba('3')),
	}
	if l.cruzarRecibo {
		recibo.Orden = referenciaSeleccionPrueba("orden:bolsa:ajena", '1')
	}
	if l.cancelar != nil {
		l.cancelar()
	}
	l.identidad, l.recibo = &identidad, recibo
	l.creaciones++
	return recibo, nil
}

func referenciaSeleccionPrueba(
	referencia string,
	caracter byte,
) ports.ReferenciaVersionadaIntegracionBolsa {
	return ports.ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 1, HuellaSHA256: strings.Repeat(string(caracter), 64),
	}
}

func procedenciaSeleccionPrueba(
	base time.Time,
	respuestaRef string,
	evidenciaRef string,
	sello string,
) ports.ProcedenciaIntegracionBolsa {
	return ports.ProcedenciaIntegracionBolsa{
		AutoridadRef: "autoridad:bolsa", RespuestaRef: respuestaRef,
		ContratoVersion: ports.VersionContratoIntegracionBolsa,
		Fuente:          referenciaSeleccionPrueba("fuente:bolsa:001", 'd'),
		Evidencia: ports.EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: evidenciaRef, ClaveVerificacionRef: claveRespuestaSeleccionPrueba,
			SelloHMAC: sello, EmitidaEn: base.Add(2 * time.Minute),
			ValidaHasta: base.Add(8 * time.Minute), RetenerHasta: base.Add(24 * time.Hour),
		},
	}
}

func selloSeleccionPrueba(caracter byte) string {
	return fmt.Sprintf("hmac-sha256:%s:%s", claveRespuestaSeleccionPrueba,
		strings.Repeat(string(caracter), 64))
}
