package ports_test

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCanonContenidoDetalleLigaCadaCampoFuncional(t *testing.T) {
	t.Parallel()
	base := datosDetalleMinimizadoPrueba(3)
	entradaBase := construirEntradaDetalleMinimizadaPrueba(t, base)
	canonBase, err := entradaBase.ExportarContenidoCanonicoParaSQL(
		instantePuertosRRHH().Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*datosDetalleMinimizado){
		"grupo": func(d *datosDetalleMinimizado) {
			d.solicitud.GrupoSubgrupo = "A1"
		},
		"motivo_solicitud": func(d *datosDetalleMinimizado) {
			d.solicitud.MotivoClave = "vacante"
		},
		"inicio_solicitud": func(d *datosDetalleMinimizado) {
			d.solicitud.PeriodoInicio = d.solicitud.PeriodoInicio.AddDate(0, 0, 1)
		},
		"fin_solicitud": func(d *datosDetalleMinimizado) {
			d.solicitud.PeriodoFin = d.solicitud.PeriodoFin.AddDate(0, 0, 1)
		},
		"modalidad": func(d *datosDetalleMinimizado) {
			d.resumen.ModalidadClave = "sustitucion"
			d.analisis.ModalidadClave = "sustitucion"
		},
		"categoria_analisis": func(d *datosDetalleMinimizado) {
			d.resumen.CategoriaRef = "categoria:rrhh:alternativa"
			d.analisis.CategoriaRef = d.resumen.CategoriaRef
		},
		"causa": func(d *datosDetalleMinimizado) {
			d.analisis.CausaClave = "vacante"
		},
		"inicio_analisis": func(d *datosDetalleMinimizado) {
			d.analisis.PeriodoInicio = d.analisis.PeriodoInicio.AddDate(0, 0, 1)
		},
		"fin_analisis": func(d *datosDetalleMinimizado) {
			d.analisis.PeriodoFin = d.analisis.PeriodoFin.AddDate(0, 0, 1)
		},
		"jornada": func(d *datosDetalleMinimizado) {
			d.analisis.PorcentajeJornada = 5_000
		},
		"resultado_rc": func(d *datosDetalleMinimizado) {
			d.analisis.ResultadoRC = domain.RCValidada
		},
		"coste": func(d *datosDetalleMinimizado) {
			d.analisis.CostePrevisto.Centimos++
		},
		"fuente_coste": func(d *datosDetalleMinimizado) {
			d.analisis.FuenteCosteRef = "fuente:coste:alternativa"
		},
		"via": func(d *datosDetalleMinimizado) {
			d.cobertura.ViaClave = "ope"
		},
		"procedimiento": func(d *datosDetalleMinimizado) {
			d.cobertura.ProcedimientoRef = "procedimiento:rrhh:alternativo"
		},
		"bolsa": func(d *datosDetalleMinimizado) {
			d.cobertura.BolsaRef = "bolsa:rrhh:alternativa"
		},
		"clave_comprobacion": func(d *datosDetalleMinimizado) {
			d.cobertura.Comprobaciones[0].Clave = "incompatibilidades"
		},
		"resultado_comprobacion": func(d *datosDetalleMinimizado) {
			d.cobertura.Comprobaciones[0].Resultado =
				domain.ComprobacionNoConsta
		},
		"unidad": func(d *datosDetalleMinimizado) {
			d.resumen.UnidadRef = "unidad:rrhh:alternativa"
			d.asignacion.UnidadRef = d.resumen.UnidadRef
		},
		"motivo_asignacion": func(d *datosDetalleMinimizado) {
			d.asignacion.MotivoClave = "necesidad_servicio"
		},
		"accion_hito": func(d *datosDetalleMinimizado) {
			d.hitos[0].AccionClave = "actuacion.minimizada.alternativa"
		},
	}
	for nombre, mutar := range mutaciones {
		nombre, mutar := nombre, mutar
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			datos := datosDetalleMinimizadoPrueba(3)
			mutar(&datos)
			entrada, err := nuevaEntradaDetalleMinimizadaPrueba(t, datos)
			if err != nil {
				t.Fatalf("mutación válida rechazada al construir: %v", err)
			}
			canon, err := entrada.ExportarContenidoCanonicoParaSQL(
				instantePuertosRRHH().Add(time.Minute),
			)
			if err != nil {
				t.Fatalf("mutación válida rechazada al exportar: %v", err)
			}
			if canon.HuellaSHA256() == canonBase.HuellaSHA256() {
				t.Fatal("la huella no ligó el campo mutado")
			}
		})
	}
}

func TestCanonContenidoDetalleLigaPresenciaYCardinalidadDeBloques(t *testing.T) {
	t.Parallel()
	huellas := make(map[string]struct{}, 4)
	for bloques := 0; bloques <= 3; bloques++ {
		entrada, _ := entradaDetalleMinimizadaPrueba(t, bloques)
		canon, err := entrada.ExportarContenidoCanonicoParaSQL(
			instantePuertosRRHH().Add(time.Minute),
		)
		if err != nil {
			t.Fatalf("%d bloques: %v", bloques, err)
		}
		if _, repetida := huellas[canon.HuellaSHA256()]; repetida {
			t.Fatalf("presencia de %d bloques no quedó ligada", bloques)
		}
		huellas[canon.HuellaSHA256()] = struct{}{}
	}

	datos := datosDetalleMinimizadoPrueba(2)
	datos.cobertura.Comprobaciones = append(
		datos.cobertura.Comprobaciones,
		ports.ComprobacionOperativaRRHH{
			Clave:     "incompatibilidades",
			Resultado: domain.ComprobacionNegativa,
		},
	)
	entrada := construirEntradaDetalleMinimizadaPrueba(t, datos)
	canon, err := entrada.ExportarContenidoCanonicoParaSQL(
		instantePuertosRRHH().Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := entradaDetalleMinimizadaPrueba(t, 2)
	canonBase, err := base.ExportarContenidoCanonicoParaSQL(
		instantePuertosRRHH().Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if canon.HuellaSHA256() == canonBase.HuellaSHA256() {
		t.Fatal("la cardinalidad de comprobaciones no quedó ligada")
	}
}

func TestCanonContenidoDetalleRechazaOrdenYVinculosImposibles(t *testing.T) {
	t.Parallel()
	casos := map[string]func(*datosDetalleMinimizado){
		"hitos_desordenados": func(d *datosDetalleMinimizado) {
			d.hitos[0], d.hitos[1] = d.hitos[1], d.hitos[0]
		},
		"hito_no_monotono": func(d *datosDetalleMinimizado) {
			d.hitos[2].RealizadaEn = d.hitos[1].RealizadaEn.Add(-time.Minute)
		},
		"resumen_no_coincide": func(d *datosDetalleMinimizado) {
			d.resumen.UnidadRef = "unidad:rrhh:divergente"
		},
		"coste_sin_fuente": func(d *datosDetalleMinimizado) {
			d.analisis.FuenteCosteRef = ""
		},
		"comprobacion_duplicada": func(d *datosDetalleMinimizado) {
			d.cobertura.Comprobaciones = append(
				d.cobertura.Comprobaciones,
				d.cobertura.Comprobaciones[0],
			)
		},
	}
	for nombre, mutar := range casos {
		nombre, mutar := nombre, mutar
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			datos := datosDetalleMinimizadoPrueba(3)
			mutar(&datos)
			entrada, err := nuevaEntradaDetalleMinimizadaPrueba(t, datos)
			if err == nil {
				_, err = entrada.ExportarContenidoCanonicoParaSQL(
					instantePuertosRRHH().Add(time.Minute),
				)
			}
			if !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("estado imposible aceptado: %v", err)
			}
		})
	}
}
