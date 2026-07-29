package ports_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestEmisorMaterialConsultaRRHHExponeSoloMetodosNominales(
	t *testing.T,
) {
	t.Parallel()
	tipo := reflect.TypeOf((*ports.EmisorMaterialConsultaRRHH)(nil))
	nominales := 0
	for _, nombre := range []string{
		"EmitirMaterialCuadroRRHH",
		"EmitirMaterialDetalleRRHH",
	} {
		metodo, existe := tipo.MethodByName(nombre)
		if !existe {
			t.Fatalf("falta método nominal %s", nombre)
		}
		if metodo.Type.NumIn() != 4 || metodo.Type.NumOut() != 2 ||
			metodo.Type.In(1) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
			metodo.Type.In(2) != reflect.TypeOf(ports.ContextoConsultaRRHH{}) {
			t.Fatalf("firma nominal inesperada en %s: %v", nombre, metodo.Type)
		}
		nominales++
	}
	for i := 0; i < tipo.NumMethod(); i++ {
		metodo := tipo.Method(i)
		if strings.HasPrefix(metodo.Name, "EmitirMaterial") &&
			metodo.Name != "EmitirMaterialCuadroRRHH" &&
			metodo.Name != "EmitirMaterialDetalleRRHH" {
			t.Fatalf("método emisor genérico o adicional: %s", metodo.Name)
		}
	}
	if nominales != 2 {
		t.Fatalf("número de métodos nominales inesperado: %d", nominales)
	}
	constructor := reflect.TypeOf(ports.NuevoEmisorMaterialConsultaRRHH)
	for i := 0; i < constructor.NumIn(); i++ {
		entrada := constructor.In(i)
		if entrada.Kind() == reflect.String || entrada.Kind() == reflect.Map ||
			entrada == reflect.TypeOf(time.Time{}) {
			t.Fatalf("selector libre en constructor: %v", entrada)
		}
	}
	tipoValor := tipo.Elem()
	for i := 0; i < tipoValor.NumField(); i++ {
		if tipoValor.Field(i).PkgPath == "" {
			t.Fatalf("campo público inesperado: %s", tipoValor.Field(i).Name)
		}
	}
}

func TestEmisorMaterialConsultaRRHHEmiteCuadroConCadenaExacta(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)
	final := inicial.Add(2 * time.Millisecond)
	guardian, motivos, correlaciones, reloj, cuadro, detalle :=
		nuevoGuardianConsultaRRHHPrueba(t, []time.Time{inicial, final})
	contexto := contextoPuertosRRHH(t, inicial)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH(
		"Auxiliar administrativo", "", "", 25, "",
	)
	if err != nil {
		t.Fatal(err)
	}

	material, err := guardian.EmitirMaterialCuadroRRHH(
		context.Background(), contexto, solicitud,
	)
	if err != nil {
		t.Fatalf("emitir material de cuadro: %v", err)
	}
	if _, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, final,
	); err != nil {
		t.Fatalf("el material no acredita el cuadro exacto: %v", err)
	}
	datos, err := cuadro.ultimaSolicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.Accion != ports.AccionConsultarCuadroRRHH ||
		datos.Finalidad != ports.FinalidadConsultarCuadroRRHH ||
		datos.ReferenciaMotivo != motivos.motivoCuadro ||
		datos.VinculoAutenticacionActor.ValidarPara(cuadro.ultimoResultado) != nil {
		t.Fatal("la cadena perdió semántica, motivo o resultado nominal")
	}
	if motivos.llamadasCuadro != 1 || motivos.llamadasDetalle != 0 ||
		correlaciones.llamadas != 1 || reloj.llamadas != 2 ||
		cuadro.llamadas != 1 || detalle.llamadas != 0 {
		t.Fatalf(
			"cardinalidad incorrecta: motivos=%d/%d correlaciones=%d reloj=%d emisores=%d/%d",
			motivos.llamadasCuadro, motivos.llamadasDetalle,
			correlaciones.llamadas, reloj.llamadas,
			cuadro.llamadas, detalle.llamadas,
		)
	}
}

func TestEmisorMaterialConsultaRRHHEmiteDetalleConCadenaExacta(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 8, 30, 0, 0, time.UTC)
	final := inicial.Add(2 * time.Millisecond)
	guardian, motivos, correlaciones, reloj, cuadro, detalle :=
		nuevoGuardianConsultaRRHHPrueba(t, []time.Time{inicial, final})
	contexto := contextoPuertosRRHH(t, inicial)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:contratacion-temporal:2026:000047", 7,
	)
	if err != nil {
		t.Fatal(err)
	}

	material, err := guardian.EmitirMaterialDetalleRRHH(
		context.Background(), contexto, solicitud,
	)
	if err != nil {
		t.Fatalf("emitir material de detalle: %v", err)
	}
	if _, err := ports.NuevaCapacidadConsultaDetalleRRHH(
		contexto, material, solicitud, final,
	); err != nil {
		t.Fatalf("el material no acredita el detalle exacto: %v", err)
	}
	datos, err := detalle.ultimaSolicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.Accion != ports.AccionConsultarDetalleRRHH ||
		datos.Finalidad != ports.FinalidadConsultarDetalleRRHH ||
		datos.ReferenciaMotivo != motivos.motivoDetalle ||
		datos.VinculoAutenticacionActor.ValidarPara(detalle.ultimoResultado) != nil {
		t.Fatal("la cadena perdió semántica, motivo o resultado nominal")
	}
	if motivos.llamadasCuadro != 0 || motivos.llamadasDetalle != 1 ||
		correlaciones.llamadas != 1 || reloj.llamadas != 2 ||
		cuadro.llamadas != 0 || detalle.llamadas != 1 {
		t.Fatalf(
			"cardinalidad incorrecta: motivos=%d/%d correlaciones=%d reloj=%d emisores=%d/%d",
			motivos.llamadasCuadro, motivos.llamadasDetalle,
			correlaciones.llamadas, reloj.llamadas,
			cuadro.llamadas, detalle.llamadas,
		)
	}
}

func TestNuevoEmisorMaterialConsultaRRHHFallaCerradoEnDependencias(
	t *testing.T,
) {
	t.Parallel()
	instante := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	motivos := &resolutorMotivoGuardianConsultaRRHHPrueba{}
	correlaciones := &generadorCorrelacionGuardianConsultaRRHHPrueba{}
	reloj := &relojGuardianConsultaRRHHPrueba{instantes: []time.Time{instante}}
	cuadro := &emisorGuardianConsultaRRHHPrueba{}
	detalle := &emisorGuardianConsultaRRHHPrueba{}
	var motivosNulos *resolutorMotivoGuardianConsultaRRHHPrueba
	var correlacionesNulas *generadorCorrelacionGuardianConsultaRRHHPrueba
	var relojNulo *relojGuardianConsultaRRHHPrueba
	var emisorNulo *emisorGuardianConsultaRRHHPrueba

	casos := []struct {
		nombre        string
		motivos       *resolutorMotivoGuardianConsultaRRHHPrueba
		correlaciones *generadorCorrelacionGuardianConsultaRRHHPrueba
		reloj         *relojGuardianConsultaRRHHPrueba
		cuadro        *emisorGuardianConsultaRRHHPrueba
		detalle       *emisorGuardianConsultaRRHHPrueba
	}{
		{"motivos_nulos", motivosNulos, correlaciones, reloj, cuadro, detalle},
		{"correlaciones_nulas", motivos, correlacionesNulas, reloj, cuadro, detalle},
		{"reloj_nulo", motivos, correlaciones, relojNulo, cuadro, detalle},
		{"cuadro_nulo", motivos, correlaciones, reloj, emisorNulo, detalle},
		{"detalle_nulo", motivos, correlaciones, reloj, cuadro, emisorNulo},
		{"misma_instancia", motivos, correlaciones, reloj, cuadro, cuadro},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			guardian, err := ports.NuevoEmisorMaterialConsultaRRHH(
				caso.motivos, caso.correlaciones, caso.reloj,
				caso.cuadro, caso.detalle,
			)
			if guardian != nil ||
				!errors.Is(err, ports.ErrCapacidadConsultaRRHHInvalida) {
				t.Fatalf("dependencia inválida aceptada: %v, %v", guardian, err)
			}
		})
	}
}

func TestEmisorMaterialConsultaRRHHNoFabricaMaterialTrasConcesionDurable(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 9, 30, 0, 0, time.UTC)
	final := inicial.Add(2 * time.Millisecond)
	guardian, _, _, reloj, cuadro, _ := nuevoGuardianConsultaRRHHPrueba(
		t, []time.Time{inicial, final},
	)
	cuadro.err = errorPrivadoGuardianConsultaRRHHPrueba()
	contexto := contextoPuertosRRHH(t, inicial)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 25, "")
	if err != nil {
		t.Fatal(err)
	}

	material, err := guardian.EmitirMaterialCuadroRRHH(
		context.Background(), contexto, solicitud,
	)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		strings.Contains(err.Error(), "SECRETO") ||
		!reflect.ValueOf(material).IsZero() ||
		cuadro.llamadas != 1 || reloj.llamadas != 2 {
		t.Fatalf("se fabricó material o filtró causa: %+v, %v", material, err)
	}
}

func TestEmisorMaterialConsultaRRHHRechazaTiempoYMaterialNoConfiables(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre     string
		instantes  []time.Time
		configurar func(*emisorGuardianConsultaRRHHPrueba)
	}{
		{
			nombre: "reloj_inicial_no_canonico",
			instantes: []time.Time{
				inicial.Add(time.Nanosecond), inicial.Add(time.Millisecond),
			},
		},
		{
			nombre:    "retroceso",
			instantes: []time.Time{inicial, inicial.Add(-time.Millisecond)},
			configurar: func(e *emisorGuardianConsultaRRHHPrueba) {
				e.instanteMaterial = inicial.Add(time.Millisecond)
			},
		},
		{
			nombre:    "reloj_final_no_canonico",
			instantes: []time.Time{inicial, inicial.Add(time.Millisecond + time.Nanosecond)},
			configurar: func(e *emisorGuardianConsultaRRHHPrueba) {
				e.instanteMaterial = inicial.Add(time.Millisecond)
			},
		},
		{
			nombre:     "exportador_nulo_tipado",
			instantes:  []time.Time{inicial, inicial.Add(time.Millisecond)},
			configurar: func(e *emisorGuardianConsultaRRHHPrueba) { e.exportadorNulo = true },
		},
		{
			nombre:    "audiencia_cruzada",
			instantes: []time.Time{inicial, inicial.Add(time.Millisecond)},
			configurar: func(e *emisorGuardianConsultaRRHHPrueba) {
				e.audiencia = ports.AudienciaConsumoConsultaDetalleRRHHV3
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			guardian, _, _, _, cuadro, _ :=
				nuevoGuardianConsultaRRHHPrueba(t, caso.instantes)
			if caso.configurar != nil {
				caso.configurar(cuadro)
			}
			contexto := contextoPuertosRRHH(t, inicial)
			solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
			if err != nil {
				t.Fatal(err)
			}
			material, err := guardian.EmitirMaterialCuadroRRHH(
				context.Background(), contexto, solicitud,
			)
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
				!reflect.ValueOf(material).IsZero() {
				t.Fatalf("evidencia no confiable aceptada: %+v, %v", material, err)
			}
		})
	}
}

func TestEmisorMaterialConsultaRRHHDetieneCadaFronteraFallida(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 10, 30, 0, 0, time.UTC)
	final := inicial.Add(time.Millisecond)
	casos := []struct {
		nombre     string
		configurar func(
			*resolutorMotivoGuardianConsultaRRHHPrueba,
			*generadorCorrelacionGuardianConsultaRRHHPrueba,
			*emisorGuardianConsultaRRHHPrueba,
		)
		emisoresEsperados int
	}{
		{
			nombre: "motivo_falla",
			configurar: func(
				r *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				r.errCuadro = errorPrivadoGuardianConsultaRRHHPrueba()
			},
		},
		{
			nombre: "motivo_invalido",
			configurar: func(
				r *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				r.motivoCuadro = motivoGuardianConsultaRRHHPrueba("x")
			},
		},
		{
			nombre: "correlacion_falla",
			configurar: func(
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				g *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				g.err = errorPrivadoGuardianConsultaRRHHPrueba()
			},
		},
		{
			nombre: "solicitud_invalida",
			configurar: func(
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			guardian, motivos, correlaciones, _, cuadro, _ :=
				nuevoGuardianConsultaRRHHPrueba(
					t, []time.Time{inicial, final},
				)
			caso.configurar(motivos, correlaciones, cuadro)
			contexto := contextoPuertosRRHH(t, inicial)
			solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
			if err != nil {
				t.Fatal(err)
			}
			if caso.nombre == "solicitud_invalida" {
				solicitud = ports.SolicitudCuadroRRHH{}
			}
			material, err := guardian.EmitirMaterialCuadroRRHH(
				context.Background(), contexto, solicitud,
			)
			if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
				strings.Contains(err.Error(), "SECRETO") ||
				!reflect.ValueOf(material).IsZero() ||
				cuadro.llamadas != caso.emisoresEsperados {
				t.Fatalf("frontera fallida no cerró: %+v, %v", material, err)
			}
		})
	}
}

func TestEmisorMaterialConsultaRRHHPriorizaCancelacion(
	t *testing.T,
) {
	t.Parallel()
	inicial := time.Date(2026, 7, 30, 11, 0, 0, 0, time.UTC)
	final := inicial.Add(time.Millisecond)
	casos := []struct {
		nombre     string
		configurar func(
			context.CancelFunc,
			*resolutorMotivoGuardianConsultaRRHHPrueba,
			*generadorCorrelacionGuardianConsultaRRHHPrueba,
			*emisorGuardianConsultaRRHHPrueba,
		)
	}{
		{
			nombre: "antes_de_empezar",
			configurar: func(
				cancelar context.CancelFunc,
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				cancelar()
			},
		},
		{
			nombre: "tras_motivo",
			configurar: func(
				cancelar context.CancelFunc,
				r *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				r.cancelarCuadro = cancelar
			},
		},
		{
			nombre: "tras_correlacion",
			configurar: func(
				cancelar context.CancelFunc,
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				g *generadorCorrelacionGuardianConsultaRRHHPrueba,
				_ *emisorGuardianConsultaRRHHPrueba,
			) {
				g.cancelar = cancelar
			},
		},
		{
			nombre: "tras_emisor",
			configurar: func(
				cancelar context.CancelFunc,
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				e *emisorGuardianConsultaRRHHPrueba,
			) {
				e.cancelar = cancelar
			},
		},
		{
			nombre: "tras_exportacion",
			configurar: func(
				cancelar context.CancelFunc,
				_ *resolutorMotivoGuardianConsultaRRHHPrueba,
				_ *generadorCorrelacionGuardianConsultaRRHHPrueba,
				e *emisorGuardianConsultaRRHHPrueba,
			) {
				e.exportadorCancela = cancelar
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			guardian, motivos, correlaciones, _, cuadro, _ :=
				nuevoGuardianConsultaRRHHPrueba(
					t, []time.Time{inicial, final},
				)
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			caso.configurar(cancelar, motivos, correlaciones, cuadro)
			contexto := contextoPuertosRRHH(t, inicial)
			solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 10, "")
			if err != nil {
				t.Fatal(err)
			}
			material, err := guardian.EmitirMaterialCuadroRRHH(
				ctx, contexto, solicitud,
			)
			if !errors.Is(err, context.Canceled) ||
				!errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
				!reflect.ValueOf(material).IsZero() {
				t.Fatalf("cancelación no prevaleció: %+v, %v", material, err)
			}
		})
	}
}

func TestEmisorMaterialConsultaRRHHRechazaContextoNuloYReceptorNulo(
	t *testing.T,
) {
	t.Parallel()
	var guardian *ports.EmisorMaterialConsultaRRHH
	material, err := guardian.EmitirMaterialCuadroRRHH(
		context.Background(),
		ports.ContextoConsultaRRHH{},
		ports.SolicitudCuadroRRHH{},
	)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		!reflect.ValueOf(material).IsZero() {
		t.Fatalf("receptor nulo aceptado: %+v, %v", material, err)
	}

	inicial := time.Date(2026, 7, 30, 11, 30, 0, 0, time.UTC)
	guardian, _, _, _, _, _ = nuevoGuardianConsultaRRHHPrueba(
		t, []time.Time{inicial, inicial.Add(time.Millisecond)},
	)
	material, err = guardian.EmitirMaterialCuadroRRHH(
		nil, ports.ContextoConsultaRRHH{}, ports.SolicitudCuadroRRHH{},
	)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		!reflect.ValueOf(material).IsZero() {
		t.Fatalf("contexto nulo aceptado: %+v, %v", material, err)
	}
	ctx, cancelar := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancelar()
	material, err = guardian.EmitirMaterialCuadroRRHH(
		ctx, ports.ContextoConsultaRRHH{}, ports.SolicitudCuadroRRHH{},
	)
	if !errors.Is(err, context.DeadlineExceeded) ||
		!errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		!reflect.ValueOf(material).IsZero() {
		t.Fatalf("vencimiento no preservado: %+v, %v", material, err)
	}
}
