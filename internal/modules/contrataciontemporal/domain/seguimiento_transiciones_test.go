package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSeguimientoRechazaTransicionEstadoMotivoDocumentoYCalendarioIncorrectos(
	t *testing.T,
) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoIncorporado(t, definicion)
	casos := []struct {
		nombre   string
		preparar func() DatosTransicionSeguimiento
	}{
		{
			"transición ausente",
			func() DatosTransicionSeguimiento {
				return datosSeguimiento("acto_ausente_01", "transicion_no_publicada", 2)
			},
		},
		{
			"estado de origen incorrecto",
			func() DatosTransicionSeguimiento {
				datos := datosSeguimiento("acto_incorporacion_02", "confirmar_incorporacion", 2)
				datos.MotivoClave = "necesidad_servicio"
				datos.Periodo = punteroIntervalo(base.Estado().PeriodoPrevisto)
				datos.Documentos = []DocumentoSeguimiento{
					{TipoClave: "resolucion_incorporacion", Referencia: referenciaSeguimientoPrueba("documento_incorporacion_02")},
				}
				return datos
			},
		},
		{
			"motivo no permitido",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.MotivoClave = "incidencia_catalogada"
				return datos
			},
		},
		{
			"documento obligatorio ausente",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Documentos = nil
				return datos
			},
		},
		{
			"tipo documental ajeno",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Documentos[0].TipoClave = "documento_no_publicado"
				return datos
			},
		},
		{
			"calendario ausente",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Calendario = nil
				return datos
			},
		},
		{
			"ámbito de calendario incorrecto",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Calendario.AmbitoTerritorialClave = "ambito_no_publicado"
				return datos
			},
		},
		{
			"resultado de calendario incorrecto",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Calendario.ResultadoClave = "resultado_no_publicado"
				return datos
			},
		},
		{
			"cálculo de calendario futuro",
			func() DatosTransicionSeguimiento {
				datos := prorrogaSeguimientoValida(base, 2)
				datos.Calendario.CalculadoEn = datos.RegistradaEn.Add(time.Microsecond)
				return datos
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := base.Aplicar(definicion, 1, caso.preparar()); !errors.Is(
				err,
				ErrTransicionInvalida,
			) {
				t.Fatalf("se aceptó el caso adversarial: %v", err)
			}
			if base.Version() != 1 || len(base.Actuaciones()) != 1 {
				t.Fatal("el rechazo mutó el agregado original")
			}
		})
	}
}

func TestSeguimientoRechazaSolapamientoUTCYOrdenTemporal(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoIncorporado(t, definicion)
	casos := []struct {
		nombre    string
		modificar func(*DatosTransicionSeguimiento)
	}{
		{
			"solapamiento",
			func(d *DatosTransicionSeguimiento) {
				d.Periodo.Desde = d.Periodo.Desde.Add(-time.Microsecond)
			},
		},
		{
			"instante no UTC",
			func(d *DatosTransicionSeguimiento) {
				d.RegistradaEn = d.RegistradaEn.In(time.FixedZone("CET", 3600))
			},
		},
		{
			"precisión superior a microsegundo",
			func(d *DatosTransicionSeguimiento) {
				d.EfectivoEn = d.EfectivoEn.Add(time.Nanosecond)
			},
		},
		{
			"registro anterior",
			func(d *DatosTransicionSeguimiento) {
				d.RegistradaEn = base.Estado().ActualizadoEn.Add(-time.Microsecond)
				d.EfectivoEn = d.RegistradaEn
				d.Calendario.CalculadoEn = d.RegistradaEn
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := prorrogaSeguimientoValida(base, 2)
			caso.modificar(&datos)
			if _, err := base.Aplicar(definicion, 1, datos); err == nil {
				t.Fatal("se aceptó un solapamiento o instante no canónico")
			}
		})
	}
}

func TestSeguimientoRechazaCeseTemporalmenteIncoherenteYOperacionTrasFinal(
	t *testing.T,
) {
	definicion := definicionSeguimientoValida(t, false)
	incorporado := seguimientoIncorporado(t, definicion)

	antesIncorporacion := ceseSeguimientoValido(incorporado, 2)
	antesIncorporacion.EfectivoEn = incorporado.PeriodosResultantes()[0].
		Intervalo.Desde.Add(-time.Microsecond)
	if _, err := incorporado.Aplicar(definicion, 1, antesIncorporacion); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptó un cese previo a la incorporación: %v", err)
	}

	antesDeSuActo := ceseSeguimientoValido(incorporado, 2)
	antesDeSuActo.RegistradaEn = incorporado.PeriodosResultantes()[0].
		Intervalo.Desde.Add(time.Hour)
	antesDeSuActo.EfectivoEn = incorporado.PeriodosResultantes()[0].Intervalo.Desde
	if _, err := incorporado.Aplicar(definicion, 1, antesDeSuActo); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptó un cese anterior a su acto: %v", err)
	}

	cesado := seguimientoCesado(t, definicion)
	incidencia := datosSeguimiento("acto_incidencia_posterior_01", "registrar_incidencia", 3)
	incidencia.MotivoClave = "incidencia_catalogada"
	incidencia.Documentos = []DocumentoSeguimiento{
		{TipoClave: "parte_incidencia", Referencia: referenciaSeguimientoPrueba("documento_incidencia_posterior_01")},
	}
	if _, err := cesado.Aplicar(definicion, 2, incidencia); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptó una operación ordinaria tras estado final: %v", err)
	}
}

func TestRectificacionEsCompensatoriaYAplicaSegregacionPublicada(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	cesado := seguimientoCesado(t, definicion)
	antes := cesado.Estado()

	rectificacion := datosSeguimiento("acto_rectificacion_01", "rectificar_cese", 3)
	rectificacion.MotivoClave = "rectificacion_material"
	rectificacion.RectificaActuacionRef = referenciaSeguimientoPrueba("acto_cese_01")
	rectificacion.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_rectificacion", Referencia: referenciaSeguimientoPrueba("documento_rectificacion_01")},
	}
	if _, err := cesado.Aplicar(definicion, 2, rectificacion); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptó al mismo actor pese a la segregación: %v", err)
	}

	rectificacion.ActorRef = referenciaSeguimientoPrueba("actor_publico_opaco_02")
	rectificacion.EfectivoEn = cesado.PeriodosResultantes()[0].Intervalo.Desde.Add(
		24 * time.Hour,
	)
	rectificado, err := cesado.Aplicar(definicion, 2, rectificacion)
	if err != nil {
		t.Fatalf("rectificar con actor segregado: %v", err)
	}
	despues := rectificado.Estado()
	if despues.Version != antes.Version+1 ||
		len(despues.Actuaciones) != len(antes.Actuaciones)+1 ||
		despues.Actuaciones[1].HuellaActuacionSHA256 !=
			antes.Actuaciones[1].HuellaActuacionSHA256 ||
		despues.Actuaciones[2].RectificaActuacionRef !=
			referenciaSeguimientoPrueba("acto_cese_01") ||
		despues.CeseEfectivo.ActuacionRef !=
			referenciaSeguimientoPrueba("acto_rectificacion_01") {
		t.Fatal("la rectificación reescribió historia o no quedó enlazada")
	}
}

func TestRectificacionDeTramoActualizaProyeccionSinReescribirHistoria(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	incorporado := seguimientoIncorporado(t, definicion)
	anterior := incorporado.Actuaciones()[0]
	rectificacion := datosSeguimiento(
		"acto_rectificacion_periodo_01",
		"rectificar_periodo",
		2,
	)
	rectificacion.MotivoClave = "rectificacion_material"
	rectificacion.ActorRef = referenciaSeguimientoPrueba("actor_publico_opaco_02")
	rectificacion.RectificaActuacionRef = anterior.ActuacionRef
	rectificacion.Periodo = &IntervaloSeguimiento{
		Desde: incorporado.PeriodosResultantes()[0].Intervalo.Desde.Add(24 * time.Hour),
		Hasta: incorporado.PeriodosResultantes()[0].Intervalo.Hasta,
	}
	rectificacion.EfectivoEn = rectificacion.Periodo.Desde
	rectificacion.Documentos = []DocumentoSeguimiento{{
		TipoClave:  "resolucion_rectificacion",
		Referencia: referenciaSeguimientoPrueba("documento_rectificacion_periodo_01"),
	}}
	rectificado, err := incorporado.Aplicar(definicion, 1, rectificacion)
	if err != nil {
		t.Fatalf("rectificar tramo: %v", err)
	}
	if rectificado.Actuaciones()[0].HuellaActuacionSHA256 !=
		anterior.HuellaActuacionSHA256 ||
		rectificado.PeriodosResultantes()[0].ActuacionRef !=
			rectificacion.ActuacionRef ||
		!rectificado.PeriodosResultantes()[0].Intervalo.Desde.Equal(
			rectificacion.Periodo.Desde,
		) {
		t.Fatal("la rectificación no compensó la proyección append-only")
	}
}

func TestReaperturaCompensaCeseYPermiteNuevoCese(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	cesado := seguimientoCesado(t, definicion)
	reapertura := datosSeguimiento("acto_reapertura_01", "reabrir_seguimiento", 3)
	reapertura.MotivoClave = "rectificacion_material"
	reapertura.Documentos = []DocumentoSeguimiento{{
		TipoClave:  "resolucion_reapertura",
		Referencia: referenciaSeguimientoPrueba("documento_reapertura_01"),
	}}
	reabierto, err := cesado.Aplicar(definicion, 2, reapertura)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	if reabierto.EstadoActual() != "vigente" || reabierto.CeseEfectivo() != nil {
		t.Fatal("la reapertura no compensó la proyección del cese")
	}
	nuevoCese := ceseSeguimientoValido(reabierto, 4)
	nuevoCese.ActuacionRef = referenciaSeguimientoPrueba("acto_cese_posterior_01")
	nuevoCese.ReciboRef = referenciaSeguimientoPrueba("recibo_cese_posterior_01")
	nuevoCese.CorrelacionRef = referenciaSeguimientoPrueba("correlacion_cese_posterior_01")
	nuevoCese.EfectivoEn = reabierto.PeriodosResultantes()[0].Intervalo.Desde.Add(
		48 * time.Hour,
	)
	recerrado, err := reabierto.Aplicar(definicion, 3, nuevoCese)
	if err != nil {
		t.Fatalf("cese posterior a reapertura: %v", err)
	}
	if recerrado.CeseEfectivo() == nil ||
		recerrado.CeseEfectivo().ActuacionRef != nuevoCese.ActuacionRef {
		t.Fatal("el nuevo cese no sustituyó la proyección compensada")
	}
}

func TestPeriodosGobernadosAdmitenIncorporacionDistintaYAmpliacionConHueco(
	t *testing.T,
) {
	definicion := definicionSeguimientoValida(t, false)
	base := seguimientoNuevoValido(t, definicion)
	incorporacion := datosSeguimiento("acto_incorporacion_diferida_01", "confirmar_incorporacion", 1)
	incorporacion.MotivoClave = "necesidad_servicio"
	incorporacion.Periodo = &IntervaloSeguimiento{
		Desde: base.Estado().PeriodoPrevisto.Desde.Add(24 * time.Hour),
		Hasta: base.Estado().PeriodoPrevisto.Hasta.Add(24 * time.Hour),
	}
	incorporacion.EfectivoEn = incorporacion.Periodo.Desde
	incorporacion.Documentos = []DocumentoSeguimiento{{
		TipoClave:  "resolucion_incorporacion",
		Referencia: referenciaSeguimientoPrueba("documento_incorporacion_diferida_01"),
	}}
	incorporado, err := base.Aplicar(definicion, 0, incorporacion)
	if err != nil {
		t.Fatalf("incorporación distinta del periodo previsto: %v", err)
	}
	prorroga := prorrogaSeguimientoValida(incorporado, 2)
	prorroga.Periodo.Desde = prorroga.Periodo.Desde.Add(24 * time.Hour)
	prorroga.Periodo.Hasta = prorroga.Periodo.Hasta.Add(24 * time.Hour)
	prorroga.EfectivoEn = prorroga.Periodo.Desde
	prorrogado, err := incorporado.Aplicar(definicion, 1, prorroga)
	if err != nil {
		t.Fatalf("ampliación no solapada con hueco: %v", err)
	}
	if !prorrogado.PeriodosResultantes()[1].Intervalo.Desde.After(
		prorrogado.PeriodosResultantes()[0].Intervalo.Hasta,
	) {
		t.Fatal("el caso no conservó el hueco gobernado")
	}
}

func TestDefinicionSeguimientoRechazaDuplicadosYCicloSilencioso(t *testing.T) {
	base := borradorDesdeDefinicion(definicionSeguimientoValida(t, false))
	casos := []struct {
		nombre    string
		adulterar func(*BorradorDefinicionSeguimiento)
	}{
		{
			"estado duplicado",
			func(b *BorradorDefinicionSeguimiento) {
				b.Estados = append(b.Estados, b.Estados[0])
			},
		},
		{
			"transición duplicada",
			func(b *BorradorDefinicionSeguimiento) {
				b.Transiciones = append(b.Transiciones, b.Transiciones[0])
			},
		},
		{
			"estado inicial final",
			func(b *BorradorDefinicionSeguimiento) {
				for indice := range b.Estados {
					if b.Estados[indice].Clave == b.EstadoInicial {
						b.Estados[indice].Final = true
					}
				}
			},
		},
		{
			"salida ordinaria desde final",
			func(b *BorradorDefinicionSeguimiento) {
				b.Transiciones = append(b.Transiciones, TransicionDefinidaSeguimiento{
					Clave: "salida_final_ordinaria", Origen: "cesada",
					Destino: "vigente", Clase: TransicionOrdinaria,
					EfectoPeriodo: EfectoPeriodoNinguno,
				})
			},
		},
		{
			"ciclo silencioso prohibido",
			func(b *BorradorDefinicionSeguimiento) {
				b.Transiciones = append(
					b.Transiciones,
					TransicionDefinidaSeguimiento{
						Clave: "entrar_espera", Origen: "vigente",
						Destino: "espera_administrativa", Clase: TransicionOrdinaria,
						EfectoPeriodo: EfectoPeriodoNinguno,
					},
					TransicionDefinidaSeguimiento{
						Clave: "salir_espera", Origen: "espera_administrativa",
						Destino: "vigente", Clase: TransicionOrdinaria,
						EfectoPeriodo: EfectoPeriodoNinguno,
					},
				)
			},
		},
		{
			"ciclo silencioso con documentos opcionales",
			func(b *BorradorDefinicionSeguimiento) {
				opcional := []RequisitoDocumentoSeguimiento{{
					TipoClave: "anotacion_opcional",
				}}
				b.Transiciones = append(
					b.Transiciones,
					TransicionDefinidaSeguimiento{
						Clave: "entrar_espera_opcional", Origen: "vigente",
						Destino: "espera_administrativa", Clase: TransicionOrdinaria,
						Documentos: opcional, EfectoPeriodo: EfectoPeriodoNinguno,
					},
					TransicionDefinidaSeguimiento{
						Clave: "salir_espera_opcional", Origen: "espera_administrativa",
						Destino: "vigente", Clase: TransicionOrdinaria,
						Documentos: opcional, EfectoPeriodo: EfectoPeriodoNinguno,
					},
				)
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := base
			borrador.Estados = append([]EstadoDefinidoSeguimiento(nil), base.Estados...)
			borrador.Transiciones = clonarTransicionesSeguimiento(base.Transiciones)
			caso.adulterar(&borrador)
			if _, err := PublicarDefinicionSeguimiento(borrador); !errors.Is(
				err,
				ErrDefinicionSeguimientoInvalida,
			) {
				t.Fatalf("se aceptó definición inválida: %v", err)
			}
		})
	}
}

func TestIntervaloSeguimientoAdmiteExtremosTransportables(t *testing.T) {
	minimo := time.Date(1, 1, 1, 0, 0, 0, 1000, time.UTC)
	maximo := time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)
	if err := (IntervaloSeguimiento{Desde: minimo, Hasta: maximo}).Validar(); err != nil {
		t.Fatalf("se rechazaron extremos transportables: %v", err)
	}
	fuera := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := (IntervaloSeguimiento{Desde: minimo, Hasta: fuera}).Validar(); !errors.Is(
		err,
		ErrSeguimientoInvalido,
	) {
		t.Fatalf("se aceptó un extremo fuera de rango: %v", err)
	}
}

func TestDefinicionSeguimientoVigenciaSemiabierta(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	publicacion := definicion.Publicacion()
	seguimiento := seguimientoIncorporado(t, definicion)
	incidencia := datosSeguimiento("acto_limite_vigencia_01", "registrar_incidencia", 2)
	incidencia.MotivoClave = "incidencia_catalogada"
	incidencia.Documentos = []DocumentoSeguimiento{
		{TipoClave: "parte_incidencia", Referencia: referenciaSeguimientoPrueba("documento_limite_vigencia_01")},
	}
	incidencia.RegistradaEn = publicacion.Vigencia.Hasta
	incidencia.EfectivoEn = publicacion.Vigencia.Hasta
	if _, err := seguimiento.Aplicar(definicion, 1, incidencia); !errors.Is(
		err,
		ErrTransicionInvalida,
	) {
		t.Fatalf("se aceptó el extremo final de vigencia: %v", err)
	}
}

func prorrogaSeguimientoValida(
	base Seguimiento,
	orden int,
) DatosTransicionSeguimiento {
	ultimo := base.PeriodosResultantes()[len(base.PeriodosResultantes())-1]
	datos := datosSeguimiento("acto_prorroga_adversarial_01", "registrar_prorroga", orden)
	datos.MotivoClave = "necesidad_servicio"
	datos.Periodo = &IntervaloSeguimiento{
		Desde: ultimo.Intervalo.Hasta,
		Hasta: ultimo.Intervalo.Hasta.AddDate(0, 1, 0),
	}
	datos.EfectivoEn = datos.Periodo.Desde
	datos.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_prorroga", Referencia: referenciaSeguimientoPrueba("documento_prorroga_adversarial_01")},
	}
	datos.Calendario = calendarioSeguimiento(datos.RegistradaEn)
	return datos
}

func ceseSeguimientoValido(
	base Seguimiento,
	orden int,
) DatosTransicionSeguimiento {
	datos := datosSeguimiento("acto_cese_adversarial_01", "registrar_cese", orden)
	datos.MotivoClave = "fin_previsto"
	datos.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_cese", Referencia: referenciaSeguimientoPrueba("documento_cese_adversarial_01")},
	}
	datos.EfectivoEn = base.PeriodosResultantes()[0].Intervalo.Desde
	return datos
}

func borradorDesdeDefinicion(
	definicion DefinicionSeguimiento,
) BorradorDefinicionSeguimiento {
	p := definicion.Publicacion()
	return BorradorDefinicionSeguimiento{
		Referencia: p.Referencia, Version: p.Version, PublicadoEn: p.PublicadoEn,
		Vigencia: p.Vigencia, EstadoInicial: p.EstadoInicial,
		ProhibeCiclosSilenciosos: p.ProhibeCiclosSilenciosos,
		Estados:                  p.Estados, Motivos: p.Motivos, Transiciones: p.Transiciones,
	}
}

func TestErroresSeguimientoNoFiltranContenidoPrivado(t *testing.T) {
	errores := []error{
		ErrDefinicionSeguimientoInvalida,
		ErrSeguimientoInvalido,
		ErrActuacionSeguimientoEnConflicto,
		ErrVersionEnConflicto,
		ErrTransicionInvalida,
	}
	for _, err := range errores {
		texto := strings.ToLower(err.Error())
		if strings.Contains(texto, "actor_publico") ||
			strings.Contains(texto, "documento_") {
			t.Fatalf("el error filtra una referencia privada: %q", texto)
		}
	}
}
