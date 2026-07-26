package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

var instanteSeguimientoBase = time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)

func TestSeguimientoRecorreIncorporacionProrrogaIncidenciaYCese(t *testing.T) {
	definicion := definicionSeguimientoValida(t, false)
	seguimiento := seguimientoNuevoValido(t, definicion)

	incorporacion := datosSeguimiento("acto_incorporacion_01", "confirmar_incorporacion", 1)
	incorporacion.Periodo = punteroIntervalo(seguimiento.Estado().PeriodoPrevisto)
	incorporacion.EfectivoEn = incorporacion.Periodo.Desde
	incorporacion.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_incorporacion", Referencia: referenciaSeguimientoPrueba("documento_incorporacion_01")},
	}
	incorporacion.MotivoClave = "necesidad_servicio"
	var err error
	seguimiento, err = seguimiento.Aplicar(definicion, 0, incorporacion)
	if err != nil {
		t.Fatalf("incorporación: %v", err)
	}

	prorroga := datosSeguimiento("acto_prorroga_01", "registrar_prorroga", 2)
	prorroga.MotivoClave = "necesidad_servicio"
	prorroga.Periodo = &IntervaloSeguimiento{
		Desde: seguimiento.Estado().PeriodoPrevisto.Hasta,
		Hasta: seguimiento.Estado().PeriodoPrevisto.Hasta.AddDate(0, 1, 0),
	}
	prorroga.EfectivoEn = prorroga.Periodo.Desde
	prorroga.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_prorroga", Referencia: referenciaSeguimientoPrueba("documento_prorroga_01")},
	}
	prorroga.Calendario = calendarioSeguimiento(prorroga.RegistradaEn)
	seguimiento, err = seguimiento.Aplicar(definicion, 1, prorroga)
	if err != nil {
		t.Fatalf("prórroga: %v", err)
	}

	incidencia := datosSeguimiento("acto_incidencia_01", "registrar_incidencia", 3)
	incidencia.MotivoClave = "incidencia_catalogada"
	incidencia.Documentos = []DocumentoSeguimiento{
		{TipoClave: "parte_incidencia", Referencia: referenciaSeguimientoPrueba("documento_incidencia_01")},
	}
	seguimiento, err = seguimiento.Aplicar(definicion, 2, incidencia)
	if err != nil {
		t.Fatalf("incidencia: %v", err)
	}

	cese := datosSeguimiento("acto_cese_01", "registrar_cese", 4)
	cese.MotivoClave = "fin_previsto"
	cese.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_cese", Referencia: referenciaSeguimientoPrueba("documento_cese_01")},
	}
	cese.EfectivoEn = seguimiento.PeriodosResultantes()[0].Intervalo.Desde.Add(
		24 * time.Hour,
	)
	seguimiento, err = seguimiento.Aplicar(definicion, 3, cese)
	if err != nil {
		t.Fatalf("cese: %v", err)
	}

	if seguimiento.Version() != 4 || seguimiento.EstadoActual() != "cesada" {
		t.Fatalf(
			"proyección final inesperada: versión=%d estado=%s",
			seguimiento.Version(),
			seguimiento.EstadoActual(),
		)
	}
	if periodos := seguimiento.PeriodosResultantes(); len(periodos) != 2 ||
		!periodos[0].Intervalo.Hasta.Equal(periodos[1].Intervalo.Desde) {
		t.Fatalf("la prórroga no conservó dos tramos contiguos: %#v", periodos)
	}
	if ceseEfectivo := seguimiento.CeseEfectivo(); ceseEfectivo == nil ||
		ceseEfectivo.ActuacionRef != referenciaSeguimientoPrueba("acto_cese_01") {
		t.Fatalf("cese efectivo no proyectado: %#v", ceseEfectivo)
	}
	if err := seguimiento.Validar(definicion); err != nil {
		t.Fatalf("agregado final inválido: %v", err)
	}
}

func TestSeguimientoAdmiteTransicionNuevaSinCambiarElNucleo(t *testing.T) {
	definicion := definicionSeguimientoValida(t, true)
	seguimiento := seguimientoIncorporado(t, definicion)
	nueva := datosSeguimiento(
		"acto_espera_administrativa_01",
		"iniciar_espera_administrativa",
		2,
	)
	nueva.Documentos = []DocumentoSeguimiento{
		{TipoClave: "acuerdo_espera", Referencia: referenciaSeguimientoPrueba("documento_espera_01")},
	}

	resultado, err := seguimiento.Aplicar(definicion, 1, nueva)
	if err != nil {
		t.Fatalf("transición gobernada nueva: %v", err)
	}
	if resultado.Version() != 2 || resultado.EstadoActual() != "espera_administrativa" ||
		resultado.Actuaciones()[1].TransicionClave != "iniciar_espera_administrativa" {
		t.Fatalf("transición nueva no materializada: %#v", resultado.Estado())
	}
}

func TestSeguimientoSoloExponeReferenciasOpacasYClaves(t *testing.T) {
	tipos := []reflect.Type{
		reflect.TypeOf(AltaSeguimiento{}),
		reflect.TypeOf(DatosTransicionSeguimiento{}),
		reflect.TypeOf(ActuacionSeguimiento{}),
		reflect.TypeOf(EstadoPersistidoSeguimiento{}),
	}
	prohibidos := []string{
		"nombre", "dni", "correo", "email", "telefono", "domicilio",
		"ubicacion", "fichaje", "medic", "salud",
	}
	for _, tipo := range tipos {
		for indice := 0; indice < tipo.NumField(); indice++ {
			nombre := strings.ToLower(tipo.Field(indice).Name)
			for _, prohibido := range prohibidos {
				if strings.Contains(nombre, prohibido) {
					t.Fatalf("%s incorpora el campo personal %s", tipo.Name(), nombre)
				}
			}
		}
	}
	for _, personal := range []string{"12345678Z", "Ana.Garcia", "958123456"} {
		if referenciaOpacaSeguimientoValida(personal) {
			t.Fatalf("se aceptó como opaca la referencia personal %q", personal)
		}
	}
	definicion := definicionSeguimientoValida(t, false)
	valido := seguimientoNuevoValido(t, definicion).Estado()
	if _, err := NuevoSeguimiento(definicion, AltaSeguimiento{
		Referencia: valido.Referencia, OrganizacionRef: "12345678Z",
		ExpedienteRef: valido.ExpedienteRef, RelacionRef: valido.RelacionRef,
		PeriodoPrevisto: valido.PeriodoPrevisto, CreadoEn: valido.CreadoEn,
	}); !errors.Is(err, ErrSeguimientoInvalido) {
		t.Fatalf("el agregado aceptó una referencia no opaca: %v", err)
	}
}

func definicionSeguimientoValida(
	t *testing.T,
	incluirNueva bool,
) DefinicionSeguimiento {
	t.Helper()
	transiciones := []TransicionDefinidaSeguimiento{
		{
			Clave: "confirmar_incorporacion", Origen: "pendiente_incorporacion",
			Destino: "vigente", Clase: TransicionOrdinaria,
			MotivosPermitidos: []ClaveCatalogo{"necesidad_servicio"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_incorporacion", Obligatorio: true},
			},
			RequierePeriodo: true, EfectoPeriodo: EfectoPeriodoAbrir,
		},
		{
			Clave: "registrar_prorroga", Origen: "vigente", Destino: "vigente",
			Clase:             TransicionOrdinaria,
			MotivosPermitidos: []ClaveCatalogo{"necesidad_servicio"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_prorroga", Obligatorio: true},
			},
			Calendario: &RequisitoCalendarioSeguimiento{
				AmbitosPermitidos:    []ClaveCatalogo{"provincia_granada"},
				ResultadosPermitidos: []ClaveCatalogo{"fecha_habil"},
			},
			RequierePeriodo: true, EfectoPeriodo: EfectoPeriodoAmpliar,
		},
		{
			Clave: "registrar_incidencia", Origen: "vigente", Destino: "vigente",
			Clase:             TransicionOrdinaria,
			MotivosPermitidos: []ClaveCatalogo{"incidencia_catalogada"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "parte_incidencia", Obligatorio: true},
			},
			EfectoPeriodo: EfectoPeriodoNinguno,
		},
		{
			Clave: "registrar_cese", Origen: "vigente", Destino: "cesada",
			Clase:             TransicionOrdinaria,
			MotivosPermitidos: []ClaveCatalogo{"fin_previsto"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_cese", Obligatorio: true},
			},
			EfectoPeriodo: EfectoPeriodoCerrar,
		},
		{
			Clave: "rectificar_periodo", Origen: "vigente", Destino: "vigente",
			Clase:             TransicionRectificacion,
			MotivosPermitidos: []ClaveCatalogo{"rectificacion_material"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_rectificacion", Obligatorio: true},
			},
			RequierePeriodo: true, EfectoPeriodo: EfectoPeriodoRectificarTramo,
			ExigeActorDistinto: true,
		},
		{
			Clave: "rectificar_cese", Origen: "cesada", Destino: "cesada",
			Clase:             TransicionRectificacion,
			MotivosPermitidos: []ClaveCatalogo{"rectificacion_material"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_rectificacion", Obligatorio: true},
			},
			EfectoPeriodo: EfectoPeriodoRectificarCese, ExigeActorDistinto: true,
		},
		{
			Clave: "reabrir_seguimiento", Origen: "cesada", Destino: "vigente",
			Clase:             TransicionReapertura,
			MotivosPermitidos: []ClaveCatalogo{"rectificacion_material"},
			MotivoObligatorio: true,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "resolucion_reapertura", Obligatorio: true},
			},
			EfectoPeriodo: EfectoPeriodoReabrir,
		},
	}
	if incluirNueva {
		transiciones = append(transiciones, TransicionDefinidaSeguimiento{
			Clave: "iniciar_espera_administrativa", Origen: "vigente",
			Destino: "espera_administrativa", Clase: TransicionOrdinaria,
			Documentos: []RequisitoDocumentoSeguimiento{
				{TipoClave: "acuerdo_espera", Obligatorio: true},
			},
			EfectoPeriodo: EfectoPeriodoNinguno,
		})
	}
	definicion, err := PublicarDefinicionSeguimiento(BorradorDefinicionSeguimiento{
		Referencia: referenciaSeguimientoPrueba("definicion_seguimiento_prueba_01"), Version: 7,
		PublicadoEn: instanteSeguimientoBase.Add(-48 * time.Hour),
		Vigencia: VigenciaSeguimiento{
			Desde: instanteSeguimientoBase.Add(-24 * time.Hour),
			Hasta: instanteSeguimientoBase.AddDate(1, 0, 0),
		},
		EstadoInicial:            "pendiente_incorporacion",
		ProhibeCiclosSilenciosos: true,
		Estados: []EstadoDefinidoSeguimiento{
			{Clave: "pendiente_incorporacion"},
			{Clave: "vigente"},
			{Clave: "espera_administrativa"},
			{Clave: "cesada", Final: true},
		},
		Motivos: []ClaveCatalogo{
			"necesidad_servicio", "incidencia_catalogada", "fin_previsto",
			"rectificacion_material",
		},
		Transiciones: transiciones,
	})
	if err != nil {
		t.Fatalf("publicar definición válida: %v", err)
	}
	return definicion
}

func seguimientoNuevoValido(
	t *testing.T,
	definicion DefinicionSeguimiento,
) Seguimiento {
	t.Helper()
	seguimiento, err := NuevoSeguimiento(definicion, AltaSeguimiento{
		Referencia:      referenciaSeguimientoPrueba("seguimiento_laboral_01"),
		OrganizacionRef: referenciaSeguimientoPrueba("organizacion_publica_01"),
		ExpedienteRef:   referenciaSeguimientoPrueba("expediente_temporal_01"),
		RelacionRef:     referenciaSeguimientoPrueba("relacion_laboral_opaca_01"),
		PeriodoPrevisto: IntervaloSeguimiento{
			Desde: instanteSeguimientoBase.AddDate(0, 0, 8),
			Hasta: instanteSeguimientoBase.AddDate(0, 1, 8),
		},
		CreadoEn: instanteSeguimientoBase,
	})
	if err != nil {
		t.Fatalf("crear seguimiento: %v", err)
	}
	return seguimiento
}

func seguimientoIncorporado(
	t *testing.T,
	definicion DefinicionSeguimiento,
) Seguimiento {
	t.Helper()
	seguimiento := seguimientoNuevoValido(t, definicion)
	datos := datosSeguimiento("acto_incorporacion_01", "confirmar_incorporacion", 1)
	datos.MotivoClave = "necesidad_servicio"
	datos.Periodo = punteroIntervalo(seguimiento.Estado().PeriodoPrevisto)
	datos.EfectivoEn = datos.Periodo.Desde
	datos.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_incorporacion", Referencia: referenciaSeguimientoPrueba("documento_incorporacion_01")},
	}
	resultado, err := seguimiento.Aplicar(definicion, 0, datos)
	if err != nil {
		t.Fatalf("incorporar seguimiento: %v", err)
	}
	return resultado
}

func seguimientoCesado(
	t *testing.T,
	definicion DefinicionSeguimiento,
) Seguimiento {
	t.Helper()
	seguimiento := seguimientoIncorporado(t, definicion)
	cese := datosSeguimiento("acto_cese_01", "registrar_cese", 2)
	cese.MotivoClave = "fin_previsto"
	cese.Documentos = []DocumentoSeguimiento{
		{TipoClave: "resolucion_cese", Referencia: referenciaSeguimientoPrueba("documento_cese_01")},
	}
	cese.EfectivoEn = seguimiento.PeriodosResultantes()[0].Intervalo.Desde
	resultado, err := seguimiento.Aplicar(definicion, 1, cese)
	if err != nil {
		t.Fatalf("cesar seguimiento: %v", err)
	}
	return resultado
}

func datosSeguimiento(
	actuacion string,
	transicion ClaveCatalogo,
	orden int,
) DatosTransicionSeguimiento {
	registrada := instanteSeguimientoBase.Add(time.Duration(orden) * time.Hour)
	return DatosTransicionSeguimiento{
		ActuacionRef: referenciaSeguimientoPrueba(actuacion), TransicionClave: transicion,
		ActorRef:   referenciaSeguimientoPrueba("actor_publico_opaco_01"),
		UnidadRef:  referenciaSeguimientoPrueba("unidad_gestora_opaca_01"),
		EfectivoEn: registrada, RegistradaEn: registrada,
		Documentos:     []DocumentoSeguimiento{},
		ReciboRef:      referenciaSeguimientoPrueba("recibo_" + actuacion),
		CorrelacionRef: referenciaSeguimientoPrueba("correlacion_" + actuacion),
	}
}

func calendarioSeguimiento(
	calculadoEn time.Time,
) *EvidenciaCalendarioSeguimiento {
	return &EvidenciaCalendarioSeguimiento{
		Referencia: referenciaSeguimientoPrueba("calendario_publicado_01"), Version: 3,
		HuellaSHA256:           strings.Repeat("a", 64),
		AmbitoTerritorialClave: "provincia_granada",
		ResultadoClave:         "fecha_habil", CalculadoEn: calculadoEn,
	}
}

func punteroIntervalo(
	intervalo IntervaloSeguimiento,
) *IntervaloSeguimiento {
	return &intervalo
}

func referenciaSeguimientoPrueba(etiqueta string) string {
	return fmt.Sprintf("ref:%x", sha256.Sum256([]byte(etiqueta)))
}
