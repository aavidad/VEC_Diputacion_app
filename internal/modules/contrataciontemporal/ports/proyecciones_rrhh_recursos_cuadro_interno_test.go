package ports

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestNuevosRecursosConsultaCuadroRRHHDerivanContratoCerrado(t *testing.T) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaCuadroRRHHPrueba(t)
	recursos, err := NuevosRecursosConsultaCuadroRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear recursos de cuadro: %v", err)
	}
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	if err != nil {
		t.Fatalf("calcular huella esperada: %v", err)
	}
	esperado := dominiovec.RecursoAutorizable{
		Referencia: contexto.organizacionRef,
		ModuloID:   ModuloContratacion,
		Tipo:       TipoRecursoCuadroRRHH,
		Ambitos: map[string]string{
			ambitoOrganizacionRecursoRRHH: contexto.organizacionRef,
			ambitoClaseRecursoRRHH:        string(AmbitoOrganizacionRRHH),
			ambitoReferenciaRecursoRRHH:   contexto.organizacionRef,
		},
		Atributos: map[string]string{
			atributoDominioConsultaRRHH: DominioHuellaConsultaCuadroRRHH,
			atributoHuellaConsultaRRHH:  huella,
		},
	}
	if !reflect.DeepEqual(recursos.recurso, esperado) {
		t.Fatalf("recurso derivado inesperado: %#v", recursos)
	}
	if len(recursos.recurso.Ambitos) != 3 ||
		len(recursos.recurso.Atributos) != 2 ||
		recursos.validarParaCuadro(contexto, solicitud, ahora) != nil {
		t.Fatal("el recurso exacto no supera su revalidador privado")
	}
}

func TestNuevosRecursosConsultaCuadroRRHHNoAdmitenConfiguracionLibre(
	t *testing.T,
) {
	t.Parallel()

	tipo := reflect.TypeOf(NuevosRecursosConsultaCuadroRRHH)
	entradas := []reflect.Type{
		reflect.TypeOf(ContextoConsultaRRHH{}),
		reflect.TypeOf(SolicitudCuadroRRHH{}),
		reflect.TypeOf(time.Time{}),
	}
	if tipo.NumIn() != len(entradas) || tipo.NumOut() != 2 {
		t.Fatalf("firma inesperada: %s", tipo)
	}
	for indice, esperado := range entradas {
		if tipo.In(indice) != esperado {
			t.Fatalf(
				"parámetro %d inesperado: obtenido=%s esperado=%s",
				indice,
				tipo.In(indice),
				esperado,
			)
		}
	}
	if tipo.Out(0) != reflect.TypeOf(RecursosConsultaRRHH{}) ||
		tipo.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatalf("salidas inesperadas: %s, %s", tipo.Out(0), tipo.Out(1))
	}
}

func TestNuevosRecursosConsultaCuadroRRHHFallanCerrado(t *testing.T) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaCuadroRRHHPrueba(t)
	casos := []struct {
		nombre    string
		contexto  ContextoConsultaRRHH
		solicitud SolicitudCuadroRRHH
		instante  time.Time
	}{
		{
			nombre: "contexto_cero", solicitud: solicitud, instante: ahora,
		},
		{
			nombre: "solicitud_cero", contexto: contexto, instante: ahora,
		},
		{
			nombre: "contexto_caducado", contexto: contexto,
			solicitud: solicitud, instante: contexto.ValidoHasta(),
		},
		{
			nombre: "contexto_aun_no_vigente", contexto: contexto,
			solicitud: solicitud,
			instante:  contexto.ResueltoEn().Add(-time.Microsecond),
		},
		{
			nombre: "instante_no_canonico", contexto: contexto,
			solicitud: solicitud, instante: ahora.Add(time.Nanosecond),
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			recursos, err := NuevosRecursosConsultaCuadroRRHH(
				caso.contexto,
				caso.solicitud,
				caso.instante,
			)
			if !reflect.DeepEqual(recursos, RecursosConsultaRRHH{}) ||
				!errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
				t.Fatalf(
					"entrada inválida produjo recursos: %#v, %v",
					recursos,
					err,
				)
			}
		})
	}
}

func TestRecursosConsultaCuadroRRHHRechazanTodaMutacionEstructural(
	t *testing.T,
) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaCuadroRRHHPrueba(t)
	base, err := NuevosRecursosConsultaCuadroRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear recursos base: %v", err)
	}
	solicitudDetalle, err := NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:cruzado",
		1,
	)
	if err != nil {
		t.Fatalf("crear solicitud de detalle cruzada: %v", err)
	}
	huellaDetalle, err := huellaSolicitudDetalleRRHH(solicitudDetalle)
	if err != nil {
		t.Fatalf("calcular huella de detalle cruzada: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*RecursosConsultaRRHH)
	}{
		{"referencia_cruzada", func(r *RecursosConsultaRRHH) {
			r.recurso.Referencia = "organizacion:otra-entidad"
		}},
		{"referencia_invalida", func(r *RecursosConsultaRRHH) {
			r.recurso.Referencia = "*"
		}},
		{"modulo_ajeno", func(r *RecursosConsultaRRHH) {
			r.recurso.ModuloID = "otro_modulo"
		}},
		{"modulo_invalido", func(r *RecursosConsultaRRHH) {
			r.recurso.ModuloID = "*"
		}},
		{"tipo_ajeno", func(r *RecursosConsultaRRHH) {
			r.recurso.Tipo = TipoRecursoExpediente
		}},
		{"tipo_invalido", func(r *RecursosConsultaRRHH) {
			r.recurso.Tipo = "*"
		}},
		{"ambitos_nulos", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos = nil
		}},
		{"ambitos_con_clave_extra", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos["extra"] = "valor"
		}},
		{"ambito_sin_organizacion", func(r *RecursosConsultaRRHH) {
			delete(r.recurso.Ambitos, ambitoOrganizacionRecursoRRHH)
		}},
		{"ambito_sin_clase", func(r *RecursosConsultaRRHH) {
			delete(r.recurso.Ambitos, ambitoClaseRecursoRRHH)
		}},
		{"ambito_sin_referencia", func(r *RecursosConsultaRRHH) {
			delete(r.recurso.Ambitos, ambitoReferenciaRecursoRRHH)
		}},
		{"organizacion_cruzada", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos[ambitoOrganizacionRecursoRRHH] =
				"organizacion:otra-entidad"
		}},
		{"clase_centro", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos[ambitoClaseRecursoRRHH] =
				string(AmbitoCentroRRHH)
		}},
		{"clase_unidad", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos[ambitoClaseRecursoRRHH] =
				string(AmbitoUnidadGestionRRHH)
		}},
		{"clase_desconocida", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos[ambitoClaseRecursoRRHH] = "desconocido"
		}},
		{"referencia_ambito_cruzada", func(r *RecursosConsultaRRHH) {
			r.recurso.Ambitos[ambitoReferenciaRecursoRRHH] =
				"organizacion:otra-entidad"
		}},
		{"atributos_nulos", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos = nil
		}},
		{"atributos_con_clave_extra", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos["extra"] = "valor"
		}},
		{"atributos_sin_dominio", func(r *RecursosConsultaRRHH) {
			delete(r.recurso.Atributos, atributoDominioConsultaRRHH)
		}},
		{"atributos_sin_huella", func(r *RecursosConsultaRRHH) {
			delete(r.recurso.Atributos, atributoHuellaConsultaRRHH)
		}},
		{"dominio_cruzado", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos[atributoDominioConsultaRRHH] =
				DominioHuellaConsultaDetalleRRHH
		}},
		{"huella_cruzada", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos[atributoHuellaConsultaRRHH] =
				strings.Repeat("e", 64)
		}},
		{"huella_nula", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos[atributoHuellaConsultaRRHH] =
				strings.Repeat("0", 64)
		}},
		{"huella_no_canonica", func(r *RecursosConsultaRRHH) {
			r.recurso.Atributos[atributoHuellaConsultaRRHH] =
				strings.Repeat("E", 64)
		}},
		{"recurso_detalle_completo", func(r *RecursosConsultaRRHH) {
			r.recurso.Referencia = "expediente:rrhh:cruzado"
			r.recurso.Tipo = TipoRecursoExpediente
			r.recurso.Atributos[atributoDominioConsultaRRHH] =
				DominioHuellaConsultaDetalleRRHH
			r.recurso.Atributos[atributoHuellaConsultaRRHH] = huellaDetalle
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			mutada := clonarRecursosConsultaCuadroRRHHPrueba(base)
			caso.mutar(&mutada)
			if err := mutada.validarParaCuadro(
				contexto,
				solicitud,
				ahora,
			); !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
				t.Fatalf("mutación aceptada: %v", err)
			}
		})
	}
}

func TestRecursosConsultaCuadroRRHHQuedanLigadosALaSolicitud(t *testing.T) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaCuadroRRHHPrueba(t)
	recursos, err := NuevosRecursosConsultaCuadroRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear recursos: %v", err)
	}
	otra, err := NuevaSolicitudCuadroRRHH(
		solicitud.Texto(),
		solicitud.EstadoClave(),
		solicitud.FaseClave(),
		solicitud.Limite()+1,
		solicitud.Cursor(),
	)
	if err != nil {
		t.Fatalf("crear solicitud cruzada: %v", err)
	}
	if err := recursos.validarParaCuadro(
		contexto,
		otra,
		ahora,
	); !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("recursos aceptados para otra solicitud: %v", err)
	}
}

func datosRecursosConsultaCuadroRRHHPrueba(
	t *testing.T,
) (time.Time, ContextoConsultaRRHH, SolicitudCuadroRRHH) {
	t.Helper()

	ahora := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	autoridad := autoridadContextoConsultaRRHHPrueba(t, ahora, "d1")
	contexto, err := NuevoContextoConsultaRRHH(
		autoridad,
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto: %v", err)
	}
	solicitud, err := NuevaSolicitudCuadroRRHH(
		"Auxiliar administrativo",
		"",
		"",
		25,
		"",
	)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	return ahora, contexto, solicitud
}

func clonarRecursosConsultaCuadroRRHHPrueba(
	origen RecursosConsultaRRHH,
) RecursosConsultaRRHH {
	copia := origen
	copia.recurso.Ambitos = make(
		map[string]string,
		len(origen.recurso.Ambitos),
	)
	for clave, valor := range origen.recurso.Ambitos {
		copia.recurso.Ambitos[clave] = valor
	}
	copia.recurso.Atributos = make(
		map[string]string,
		len(origen.recurso.Atributos),
	)
	for clave, valor := range origen.recurso.Atributos {
		copia.recurso.Atributos[clave] = valor
	}
	return copia
}
