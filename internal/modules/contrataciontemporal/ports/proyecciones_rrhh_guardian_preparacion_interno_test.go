package ports

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type generadorCorrelacionPreparacionRRHHPrueba struct{ valor string }

func (g generadorCorrelacionPreparacionRRHHPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioPreparacionAutorizacionRRHHPrueba struct {
	ahora              time.Time
	contextoCuadro     ContextoConsultaRRHH
	contextoDetalle    ContextoConsultaRRHH
	solicitudCuadro    SolicitudCuadroRRHH
	solicitudDetalle   SolicitudDetalleRRHH
	motivoCuadro       dominiovec.ReferenciaEntradaCatalogo
	motivoDetalle      dominiovec.ReferenciaEntradaCatalogo
	correlacionCuadro  dominiovec.ReferenciaCorrelacionAutorizacionV2
	correlacionDetalle dominiovec.ReferenciaCorrelacionAutorizacionV2
}

func TestPreparacionesAutorizacionConsultaRRHHConstruyenNominalesExactos(
	t *testing.T,
) {
	t.Parallel()
	e := nuevoEscenarioPreparacionAutorizacionRRHHPrueba(t)

	t.Run("cuadro", func(t *testing.T) {
		preparacion, err := prepararAutorizacionCuadroRRHH(
			e.contextoCuadro,
			e.solicitudCuadro,
			e.motivoCuadro,
			e.correlacionCuadro,
			e.ahora,
		)
		if err != nil || preparacion.validarEn(e.ahora) != nil {
			t.Fatalf("preparar cuadro: %v", err)
		}
		exigirPreparacionAutorizacionRRHHPrueba(
			t,
			preparacion,
			clasePreparacionAutorizacionCuadroRRHH,
			AccionConsultarCuadroRRHH,
			FinalidadConsultarCuadroRRHH,
		)
	})

	t.Run("detalle", func(t *testing.T) {
		preparacion, err := prepararAutorizacionDetalleRRHH(
			e.contextoDetalle,
			e.solicitudDetalle,
			e.motivoDetalle,
			e.correlacionDetalle,
			e.ahora,
		)
		if err != nil || preparacion.validarEn(e.ahora) != nil {
			t.Fatalf("preparar detalle: %v", err)
		}
		exigirPreparacionAutorizacionRRHHPrueba(
			t,
			preparacion,
			clasePreparacionAutorizacionDetalleRRHH,
			AccionConsultarDetalleRRHH,
			FinalidadConsultarDetalleRRHH,
		)
	})
}

func TestPreparacionesAutorizacionConsultaRRHHTienenFirmaCerrada(
	t *testing.T,
) {
	t.Parallel()
	tipoContexto := reflect.TypeOf(ContextoConsultaRRHH{})
	tipoMotivo := reflect.TypeOf(dominiovec.ReferenciaEntradaCatalogo{})
	tipoCorrelacion := reflect.TypeOf(
		dominiovec.ReferenciaCorrelacionAutorizacionV2{},
	)
	tipoInstante := reflect.TypeOf(time.Time{})
	casos := []struct {
		nombre    string
		fabrica   any
		solicitud reflect.Type
	}{
		{"cuadro", prepararAutorizacionCuadroRRHH, reflect.TypeOf(SolicitudCuadroRRHH{})},
		{"detalle", prepararAutorizacionDetalleRRHH, reflect.TypeOf(SolicitudDetalleRRHH{})},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tipo := reflect.TypeOf(caso.fabrica)
			esperadas := []reflect.Type{
				tipoContexto, caso.solicitud, tipoMotivo, tipoCorrelacion,
				tipoInstante,
			}
			if tipo.NumIn() != len(esperadas) || tipo.NumOut() != 2 {
				t.Fatalf("firma abierta: %s", tipo)
			}
			for indice, esperada := range esperadas {
				if tipo.In(indice) != esperada {
					t.Fatalf(
						"entrada %d inesperada: %s", indice, tipo.In(indice),
					)
				}
			}
			if tipo.Out(0) != reflect.TypeOf(
				preparacionAutorizacionConsultaRRHH{},
			) || tipo.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
				t.Fatalf("salidas inesperadas: %s", tipo)
			}
		})
	}

	tipoNominal := reflect.TypeOf(preparacionAutorizacionConsultaRRHH{})
	for indice := 0; indice < tipoNominal.NumField(); indice++ {
		campo := tipoNominal.Field(indice)
		if campo.PkgPath == "" {
			t.Fatalf("campo público inesperado: %s", campo.Name)
		}
	}
}

func TestPreparacionesAutorizacionConsultaRRHHFallanCerradasConCeros(
	t *testing.T,
) {
	t.Parallel()
	e := nuevoEscenarioPreparacionAutorizacionRRHHPrueba(t)
	casosCuadro := []struct {
		nombre      string
		contexto    ContextoConsultaRRHH
		solicitud   SolicitudCuadroRRHH
		motivo      dominiovec.ReferenciaEntradaCatalogo
		correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2
		instante    time.Time
	}{
		{"contexto_cero", ContextoConsultaRRHH{}, e.solicitudCuadro, e.motivoCuadro, e.correlacionCuadro, e.ahora},
		{"solicitud_cero", e.contextoCuadro, SolicitudCuadroRRHH{}, e.motivoCuadro, e.correlacionCuadro, e.ahora},
		{"motivo_cero", e.contextoCuadro, e.solicitudCuadro, dominiovec.ReferenciaEntradaCatalogo{}, e.correlacionCuadro, e.ahora},
		{"correlacion_cero", e.contextoCuadro, e.solicitudCuadro, e.motivoCuadro, dominiovec.ReferenciaCorrelacionAutorizacionV2{}, e.ahora},
		{"instante_cero", e.contextoCuadro, e.solicitudCuadro, e.motivoCuadro, e.correlacionCuadro, time.Time{}},
		{"contexto_vencido", e.contextoCuadro, e.solicitudCuadro, e.motivoCuadro, e.correlacionCuadro, e.contextoCuadro.ValidoHasta()},
	}
	for _, caso := range casosCuadro {
		t.Run("cuadro_"+caso.nombre, func(t *testing.T) {
			obtenida, err := prepararAutorizacionCuadroRRHH(
				caso.contexto, caso.solicitud, caso.motivo,
				caso.correlacion, caso.instante,
			)
			exigirPreparacionRechazadaRRHHPrueba(t, obtenida, err)
		})
	}

	casosDetalle := []struct {
		nombre      string
		contexto    ContextoConsultaRRHH
		solicitud   SolicitudDetalleRRHH
		motivo      dominiovec.ReferenciaEntradaCatalogo
		correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2
		instante    time.Time
	}{
		{"contexto_cero", ContextoConsultaRRHH{}, e.solicitudDetalle, e.motivoDetalle, e.correlacionDetalle, e.ahora},
		{"solicitud_cero", e.contextoDetalle, SolicitudDetalleRRHH{}, e.motivoDetalle, e.correlacionDetalle, e.ahora},
		{"motivo_cero", e.contextoDetalle, e.solicitudDetalle, dominiovec.ReferenciaEntradaCatalogo{}, e.correlacionDetalle, e.ahora},
		{"correlacion_cero", e.contextoDetalle, e.solicitudDetalle, e.motivoDetalle, dominiovec.ReferenciaCorrelacionAutorizacionV2{}, e.ahora},
		{"instante_cero", e.contextoDetalle, e.solicitudDetalle, e.motivoDetalle, e.correlacionDetalle, time.Time{}},
		{"contexto_vencido", e.contextoDetalle, e.solicitudDetalle, e.motivoDetalle, e.correlacionDetalle, e.contextoDetalle.ValidoHasta()},
	}
	for _, caso := range casosDetalle {
		t.Run("detalle_"+caso.nombre, func(t *testing.T) {
			obtenida, err := prepararAutorizacionDetalleRRHH(
				caso.contexto, caso.solicitud, caso.motivo,
				caso.correlacion, caso.instante,
			)
			exigirPreparacionRechazadaRRHHPrueba(t, obtenida, err)
		})
	}
}

func TestPreparacionAutorizacionConsultaRRHHRechazaCrucesNominales(
	t *testing.T,
) {
	t.Parallel()
	e := nuevoEscenarioPreparacionAutorizacionRRHHPrueba(t)
	cuadro, err := prepararAutorizacionCuadroRRHH(
		e.contextoCuadro, e.solicitudCuadro, e.motivoCuadro,
		e.correlacionCuadro, e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := prepararAutorizacionDetalleRRHH(
		e.contextoDetalle, e.solicitudDetalle, e.motivoDetalle,
		e.correlacionDetalle, e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	otraSolicitudCuadro, err := NuevaSolicitudCuadroRRHH(
		e.solicitudCuadro.Texto(),
		e.solicitudCuadro.EstadoClave(),
		e.solicitudCuadro.FaseClave(),
		e.solicitudCuadro.Limite()+1,
		e.solicitudCuadro.Cursor(),
	)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*preparacionAutorizacionConsultaRRHH){
		"clase_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.clase = clasePreparacionAutorizacionDetalleRRHH
		},
		"solicitud_cuadro": func(p *preparacionAutorizacionConsultaRRHH) {
			p.solicitudCuadro = otraSolicitudCuadro
		},
		"solicitud_detalle_inyectada": func(p *preparacionAutorizacionConsultaRRHH) {
			p.solicitudDetalle = detalle.solicitudDetalle
		},
		"contexto_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.contexto = detalle.contexto
		},
		"resultado_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.resultado = detalle.resultado
		},
		"recursos_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.recursos = detalle.recursos
		},
		"motivo_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.motivo = detalle.motivo
		},
		"correlacion_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.correlacion = detalle.correlacion
		},
		"solicitud_vec_detalle": func(p *preparacionAutorizacionConsultaRRHH) {
			p.solicitudVEC = detalle.solicitudVEC
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			mutada := cuadro
			mutar(&mutada)
			if err := mutada.validarEn(e.ahora); !errors.Is(
				err, ErrCapacidadConsultaRRHHInvalida,
			) {
				t.Fatalf("cruce aceptado: %v", err)
			}
		})
	}

	detalleConCuadro := detalle
	detalleConCuadro.recursos = cuadro.recursos
	if err := detalleConCuadro.validarEn(e.ahora); !errors.Is(
		err, ErrCapacidadConsultaRRHHInvalida,
	) {
		t.Fatalf("detalle aceptó recursos de cuadro: %v", err)
	}
	if err := (preparacionAutorizacionConsultaRRHH{}).validarEn(
		e.ahora,
	); !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("nominal cero aceptado: %v", err)
	}
}

func TestPreparacionAutorizacionConsultaRRHHConservaCopiasDefensivas(
	t *testing.T,
) {
	t.Parallel()
	e := nuevoEscenarioPreparacionAutorizacionRRHHPrueba(t)
	preparacion, err := prepararAutorizacionCuadroRRHH(
		e.contextoCuadro, e.solicitudCuadro, e.motivoCuadro,
		e.correlacionCuadro, e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}

	mutarResultadoContextoConsultaRRHHPrueba(
		&e.contextoCuadro.autoridad.Resultado,
	)
	datos, err := preparacion.solicitudVEC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Recurso.Ambitos["organizacion_ref"] = "organizacion:adulterada"
	datos.Recurso.Atributos["consulta_huella_sha256"] = strings.Repeat("f", 64)
	if err := preparacion.validarEn(e.ahora); err != nil {
		t.Fatalf("una mutación externa alcanzó el nominal: %v", err)
	}

	mutada := preparacion
	mutada.recursos = clonarRecursosConsultaCuadroRRHHPrueba(
		preparacion.recursos,
	)
	mutada.recursos.recurso.Ambitos["organizacion_ref"] =
		"organizacion:adulterada"
	if err := mutada.validarEn(e.ahora); !errors.Is(
		err, ErrCapacidadConsultaRRHHInvalida,
	) {
		t.Fatalf("mutación interna no detectada: %v", err)
	}
}

func TestPreparacionAutorizacionConsultaRRHHBloqueaCodecsYRedacta(
	t *testing.T,
) {
	t.Parallel()
	e := nuevoEscenarioPreparacionAutorizacionRRHHPrueba(t)
	preparacion, err := prepararAutorizacionDetalleRRHH(
		e.contextoDetalle, e.solicitudDetalle, e.motivoDetalle,
		e.correlacionDetalle, e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{
		"valor": preparacion, "puntero": &preparacion,
	} {
		t.Run(nombre, func(t *testing.T) {
			comprobarBloqueoPreparacionRRHHPrueba(t, valor)
			comprobarRedaccionPreparacionRRHHPrueba(t, valor, e)
		})
	}
	comprobarBloqueoLecturaPreparacionRRHHPrueba(t, &preparacion)
}

func nuevoEscenarioPreparacionAutorizacionRRHHPrueba(
	t *testing.T,
) escenarioPreparacionAutorizacionRRHHPrueba {
	t.Helper()
	ahora := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	contextoCuadro, err := NuevoContextoConsultaRRHH(
		autoridadContextoConsultaRRHHPrueba(t, ahora, "pa"),
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	contextoDetalle, err := NuevoContextoConsultaRRHH(
		autoridadContextoConsultaRRHHPrueba(t, ahora, "pb"),
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudCuadro, err := NuevaSolicitudCuadroRRHH(
		"Auxiliar administrativo", "", "", 25, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudDetalle, err := NuevaSolicitudDetalleRRHH(
		"expediente:contratacion-temporal:2026:000047", 7,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioPreparacionAutorizacionRRHHPrueba{
		ahora: ahora, contextoCuadro: contextoCuadro,
		contextoDetalle:    contextoDetalle,
		solicitudCuadro:    solicitudCuadro,
		solicitudDetalle:   solicitudDetalle,
		motivoCuadro:       motivoPreparacionRRHHPrueba("1"),
		motivoDetalle:      motivoPreparacionRRHHPrueba("2"),
		correlacionCuadro:  correlacionPreparacionRRHHPrueba(t, "1"),
		correlacionDetalle: correlacionPreparacionRRHHPrueba(t, "2"),
	}
}

func motivoPreparacionRRHHPrueba(
	marca string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat(marca, 64),
		EntradaClave:         "motivo_" + strings.Repeat(marca, 32),
	}
}

func correlacionPreparacionRRHHPrueba(
	t *testing.T,
	marca string,
) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionPreparacionRRHHPrueba{
			valor: "correlacion_" + strings.Repeat(marca, 32),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return correlacion
}

func exigirPreparacionAutorizacionRRHHPrueba(
	t *testing.T,
	preparacion preparacionAutorizacionConsultaRRHH,
	clase clasePreparacionAutorizacionConsultaRRHH,
	accion, finalidad string,
) {
	t.Helper()
	datos, err := preparacion.solicitudVEC.Datos()
	correlacion, errCorrelacion := preparacion.correlacion.ValorCanonico()
	correlacionVEC, errCorrelacionVEC := datos.Correlacion.ValorCanonico()
	if err != nil || errCorrelacion != nil || errCorrelacionVEC != nil ||
		preparacion.clase != clase ||
		datos.Accion != accion || datos.Finalidad != finalidad ||
		datos.ReferenciaMotivo != preparacion.motivo ||
		correlacion != correlacionVEC ||
		!reflect.DeepEqual(datos.Recurso, preparacion.recursos.recurso) ||
		!datos.VinculoAutenticacionActor.CoincideExactamenteCon(
			preparacion.contexto.autoridad.Vinculo,
		) ||
		datos.VinculoAutenticacionActor.ValidarPara(
			preparacion.resultado,
		) != nil {
		t.Fatalf("nominal incompleto o cruzado: %v", err)
	}
}

func exigirPreparacionRechazadaRRHHPrueba(
	t *testing.T,
	preparacion preparacionAutorizacionConsultaRRHH,
	err error,
) {
	t.Helper()
	if !reflect.DeepEqual(
		preparacion, preparacionAutorizacionConsultaRRHH{},
	) || !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("entrada inválida produjo nominal: %#v, %v", preparacion, err)
	}
}

func comprobarBloqueoPreparacionRRHHPrueba(t *testing.T, valor any) {
	t.Helper()
	codecs, ok := valor.(interface {
		MarshalText() ([]byte, error)
		MarshalBinary() ([]byte, error)
		GobEncode() ([]byte, error)
		MarshalCBOR() ([]byte, error)
		MarshalYAML() (any, error)
	})
	if !ok {
		t.Fatalf("%T no bloquea todos los codecs", valor)
	}
	var gobBytes bytes.Buffer
	errores := []error{
		func() error { _, err := json.Marshal(valor); return err }(),
		func() error { _, err := xml.Marshal(valor); return err }(),
		func() error { _, err := codecs.MarshalText(); return err }(),
		func() error { _, err := codecs.MarshalBinary(); return err }(),
		func() error { _, err := codecs.GobEncode(); return err }(),
		func() error { _, err := codecs.MarshalCBOR(); return err }(),
		func() error { _, err := codecs.MarshalYAML(); return err }(),
		gob.NewEncoder(&gobBytes).Encode(valor),
	}
	for indice, err := range errores {
		if !errors.Is(err, ErrMaterialConsultaRRHHSensible) {
			t.Fatalf("codec %d no bloqueado: %v", indice, err)
		}
	}
}

func comprobarBloqueoLecturaPreparacionRRHHPrueba(
	t *testing.T,
	preparacion *preparacionAutorizacionConsultaRRHH,
) {
	t.Helper()
	codecs := any(preparacion).(interface {
		UnmarshalText([]byte) error
		UnmarshalBinary([]byte) error
		GobDecode([]byte) error
		UnmarshalCBOR([]byte) error
		UnmarshalYAML(func(any) error) error
	})
	errores := []error{
		json.Unmarshal([]byte(`{}`), preparacion),
		xml.Unmarshal([]byte(`<preparacion/>`), preparacion),
		codecs.UnmarshalText(nil),
		codecs.UnmarshalBinary(nil),
		codecs.GobDecode(nil),
		codecs.UnmarshalCBOR(nil),
		codecs.UnmarshalYAML(func(any) error { return nil }),
	}
	for indice, err := range errores {
		if !errors.Is(err, ErrMaterialConsultaRRHHSensible) {
			t.Fatalf("decodificador %d no bloqueado: %v", indice, err)
		}
	}
}

func comprobarRedaccionPreparacionRRHHPrueba(
	t *testing.T,
	valor any,
	e escenarioPreparacionAutorizacionRRHHPrueba,
) {
	t.Helper()
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info(
		"preparacion_autorizacion_rrhh", "valor", valor,
	)
	correlacion, _ := e.correlacionDetalle.ValorCanonico()
	for _, texto := range []string{
		fmt.Sprintf("%v %+v %#v", valor, valor, valor),
		registro.String(),
	} {
		if !strings.Contains(texto, "MATERIAL-CONSULTA-RRHH-OPACO") {
			t.Fatalf("representación no opaca: %q", texto)
		}
		for _, sensible := range []string{
			e.solicitudDetalle.ExpedienteRef(),
			e.contextoDetalle.ActorRef(),
			e.motivoDetalle.EntradaClave,
			correlacion,
		} {
			if strings.Contains(texto, sensible) {
				t.Fatalf("representación filtra %q: %q", sensible, texto)
			}
		}
	}
}
