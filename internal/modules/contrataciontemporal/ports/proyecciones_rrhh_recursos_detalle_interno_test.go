package ports

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestNuevosRecursosConsultaDetalleRRHHDerivanContratoCerrado(t *testing.T) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaDetalleRRHHPrueba(t)
	recursos, err := NuevosRecursosConsultaDetalleRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear recursos de detalle: %v", err)
	}
	huella, err := huellaSolicitudDetalleRRHH(solicitud)
	if err != nil {
		t.Fatalf("calcular huella esperada: %v", err)
	}
	esperado := dominiovec.RecursoAutorizable{
		Referencia: solicitud.expedienteRef,
		ModuloID:   ModuloContratacion,
		Tipo:       TipoRecursoExpediente,
		Ambitos: map[string]string{
			ambitoOrganizacionRecursoRRHH: contexto.organizacionRef,
			ambitoClaseRecursoRRHH:        string(AmbitoOrganizacionRRHH),
			ambitoReferenciaRecursoRRHH:   contexto.organizacionRef,
		},
		Atributos: map[string]string{
			atributoDominioConsultaRRHH: DominioHuellaConsultaDetalleRRHH,
			atributoHuellaConsultaRRHH:  huella,
		},
	}
	if !reflect.DeepEqual(recursos.recurso, esperado) {
		t.Fatalf("recurso derivado inesperado: %#v", recursos)
	}
	if len(recursos.recurso.Ambitos) != 3 ||
		len(recursos.recurso.Atributos) != 2 ||
		recursos.validarParaDetalle(contexto, solicitud, ahora) != nil {
		t.Fatal("el recurso exacto no supera su revalidador privado")
	}
}

func TestNuevosRecursosConsultaDetalleRRHHNoAdmitenConfiguracionLibre(
	t *testing.T,
) {
	t.Parallel()

	tipo := reflect.TypeOf(NuevosRecursosConsultaDetalleRRHH)
	entradas := []reflect.Type{
		reflect.TypeOf(ContextoConsultaRRHH{}),
		reflect.TypeOf(SolicitudDetalleRRHH{}),
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

func TestNuevosRecursosConsultaDetalleRRHHFallanCerrado(t *testing.T) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaDetalleRRHHPrueba(t)
	casos := []struct {
		nombre    string
		contexto  ContextoConsultaRRHH
		solicitud SolicitudDetalleRRHH
		instante  time.Time
	}{
		{
			nombre: "contexto_cero", solicitud: solicitud, instante: ahora,
		},
		{
			nombre: "solicitud_cero", contexto: contexto, instante: ahora,
		},
		{
			nombre: "instante_cero", contexto: contexto, solicitud: solicitud,
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
			recursos, err := NuevosRecursosConsultaDetalleRRHH(
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

func TestRecursosConsultaDetalleRRHHRechazanTodaMutacionEstructural(
	t *testing.T,
) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaDetalleRRHHPrueba(t)
	base, err := NuevosRecursosConsultaDetalleRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear recursos base: %v", err)
	}
	solicitudCuadro, err := NuevaSolicitudCuadroRRHH("", "", "", 25, "")
	if err != nil {
		t.Fatalf("crear solicitud de cuadro cruzada: %v", err)
	}
	huellaCuadro, err := huellaSolicitudCuadroRRHH(solicitudCuadro)
	if err != nil {
		t.Fatalf("calcular huella de cuadro cruzada: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*RecursosConsultaRRHH)
	}{
		{"referencia_cruzada", func(r *RecursosConsultaRRHH) {
			r.recurso.Referencia = "expediente:rrhh:otro"
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
		{"tipo_cuadro", func(r *RecursosConsultaRRHH) {
			r.recurso.Tipo = TipoRecursoCuadroRRHH
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
				DominioHuellaConsultaCuadroRRHH
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
		{"recurso_cuadro_completo", func(r *RecursosConsultaRRHH) {
			r.recurso.Referencia = contexto.organizacionRef
			r.recurso.Tipo = TipoRecursoCuadroRRHH
			r.recurso.Atributos[atributoDominioConsultaRRHH] =
				DominioHuellaConsultaCuadroRRHH
			r.recurso.Atributos[atributoHuellaConsultaRRHH] = huellaCuadro
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			mutada := clonarRecursosConsultaDetalleRRHHPrueba(base)
			caso.mutar(&mutada)
			if err := mutada.validarParaDetalle(
				contexto,
				solicitud,
				ahora,
			); !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
				t.Fatalf("mutación aceptada: %v", err)
			}
		})
	}
}

func TestRecursosConsultaDetalleRRHHLiganVersionSinCambiarReferencia(
	t *testing.T,
) {
	t.Parallel()

	ahora, contexto, solicitud := datosRecursosConsultaDetalleRRHHPrueba(t)
	primero, err := NuevosRecursosConsultaDetalleRRHH(
		contexto,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear primer recurso: %v", err)
	}
	otra, err := NuevaSolicitudDetalleRRHH(
		solicitud.expedienteRef,
		solicitud.versionObservada+1,
	)
	if err != nil {
		t.Fatalf("crear solicitud con otra versión: %v", err)
	}
	segundo, err := NuevosRecursosConsultaDetalleRRHH(contexto, otra, ahora)
	if err != nil {
		t.Fatalf("crear segundo recurso: %v", err)
	}
	if primero.recurso.Referencia != solicitud.expedienteRef ||
		segundo.recurso.Referencia != solicitud.expedienteRef {
		t.Fatal("la versión observada alteró la referencia del expediente")
	}
	huellaPrimera := primero.recurso.Atributos[atributoHuellaConsultaRRHH]
	huellaSegunda := segundo.recurso.Atributos[atributoHuellaConsultaRRHH]
	if huellaPrimera == huellaSegunda {
		t.Fatal("dos versiones observadas produjeron la misma huella")
	}
	if err := primero.validarParaDetalle(
		contexto,
		otra,
		ahora,
	); !errors.Is(err, ErrCapacidadConsultaRRHHInvalida) {
		t.Fatalf("el primer recurso aceptó otra versión observada: %v", err)
	}
	if segundo.validarParaDetalle(contexto, otra, ahora) != nil {
		t.Fatal("el segundo recurso no quedó ligado a la nueva versión")
	}
}

func datosRecursosConsultaDetalleRRHHPrueba(
	t *testing.T,
) (time.Time, ContextoConsultaRRHH, SolicitudDetalleRRHH) {
	t.Helper()

	ahora := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	autoridad := autoridadContextoConsultaRRHHPrueba(t, ahora, "d2")
	contexto, err := NuevoContextoConsultaRRHH(
		autoridad,
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto: %v", err)
	}
	solicitud, err := NuevaSolicitudDetalleRRHH(
		"expediente:contratacion-temporal:2026:000047",
		7,
	)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	return ahora, contexto, solicitud
}

func clonarRecursosConsultaDetalleRRHHPrueba(
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
