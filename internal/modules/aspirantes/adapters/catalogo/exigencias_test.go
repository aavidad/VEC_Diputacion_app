package catalogo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/datospersonales"
)

type resolutorPrueba struct {
	datos map[datospersonales.TipoConvocatoria][]datospersonales.DatoRequerido
	err   error
	otra  bool
}

func (r resolutorPrueba) Para(_ context.Context, t datospersonales.TipoConvocatoria, m datospersonales.Momento) ([]datospersonales.DatoRequerido, datospersonales.Catalogo, error) {
	if m != datospersonales.MomentoInscripcion {
		return nil, datospersonales.Catalogo{}, errors.New("momento")
	}
	huella := strings.Repeat("a", 64)
	if r.otra && t == datospersonales.TipoSelectivoLibre {
		huella = strings.Repeat("b", 64)
	}
	return r.datos[t], datospersonales.Catalogo{CatalogoID: datospersonales.CatalogoDatosPersonales, Version: 1, HuellaCatalogo: huella, PaqueteEjemplo: true}, r.err
}

func dato(nombre string, o datospersonales.Obligatoriedad, condicion string) datospersonales.DatoRequerido {
	return datospersonales.DatoRequerido{Dato: nombre, Obligatoriedad: o, Condicion: condicion, Custodia: datospersonales.CustodiaAspirantes,
		Categoria: datospersonales.CategoriaOrdinaria, Origen: datospersonales.OrigenEjemplo}
}

// Valores del paquete de ejemplo del catálogo (PR #140).
var ejemplo = map[datospersonales.TipoConvocatoria][]datospersonales.DatoRequerido{
	datospersonales.TipoBolsa: {
		dato("nombre_apellidos", datospersonales.Obligatorio, ""), dato("correo", datospersonales.Obligatorio, ""),
		dato("telefono", datospersonales.Obligatorio, ""), dato("telefono_secundario", datospersonales.Voluntario, ""),
		dato("domicilio_notificacion", datospersonales.Condicional, "si_elige_notificacion_papel"),
		{Dato: "discapacidad_grado", Obligatoriedad: datospersonales.Condicional, Custodia: datospersonales.CustodiaAspirantes, Categoria: datospersonales.CategoriaEspecial},
	},
	datospersonales.TipoSelectivoLibre: {
		dato("telefono", datospersonales.Voluntario, ""),
		dato("domicilio_notificacion", datospersonales.Condicional, "si_elige_notificacion_papel"),
	},
}

func TestBolsaPideTelefonoObligatorioYDomicilioCondicional(t *testing.T) {
	a, err := Nuevo(resolutorPrueba{datos: ejemplo}, []datospersonales.TipoConvocatoria{datospersonales.TipoBolsa})
	if err != nil {
		t.Fatal(err)
	}
	e, err := a.ExigenciasContactoFichaPropia(context.Background())
	if err != nil || !e.Ejemplo || !strings.HasPrefix(e.CatalogoRef, "vec.aspirantes.datos_personales:1:") {
		t.Fatalf("%+v %v", e, err)
	}
	esperado := []domain.ExigenciaCampo{
		{Campo: domain.CampoTelefono, Obligatorio: true}, {Campo: domain.CampoMovil},
		{Campo: domain.CampoDomicilio, Condicion: "si_elige_notificacion_papel"}, {Campo: domain.CampoCodigoPostal, Condicion: "si_elige_notificacion_papel"},
	}
	if len(e.Campos) != len(esperado) {
		t.Fatalf("%+v", e.Campos)
	}
	for i := range esperado {
		if e.Campos[i] != esperado[i] {
			t.Fatalf("%d: %+v", i, e.Campos[i])
		}
	}
}

func TestVariosTiposYErrores(t *testing.T) {
	a, _ := Nuevo(resolutorPrueba{datos: ejemplo}, []datospersonales.TipoConvocatoria{datospersonales.TipoSelectivoLibre, datospersonales.TipoBolsa})
	e, err := a.ExigenciasContactoFichaPropia(context.Background())
	if err != nil || e.Campos[0] != (domain.ExigenciaCampo{Campo: domain.CampoTelefono, Obligatorio: true}) {
		t.Fatalf("el obligatorio de bolsa manda: %+v %v", e.Campos, err)
	}
	a, _ = Nuevo(resolutorPrueba{datos: ejemplo, err: datospersonales.ErrCatalogoNoDisponible}, []datospersonales.TipoConvocatoria{datospersonales.TipoBolsa})
	if _, err := a.ExigenciasContactoFichaPropia(context.Background()); err == nil {
		t.Fatal("catálogo caído")
	}
	a, _ = Nuevo(resolutorPrueba{datos: ejemplo, otra: true}, []datospersonales.TipoConvocatoria{datospersonales.TipoBolsa, datospersonales.TipoSelectivoLibre})
	if _, err := a.ExigenciasContactoFichaPropia(context.Background()); err == nil {
		t.Fatal("dos versiones del catálogo mezcladas")
	}
	for _, tipos := range [][]datospersonales.TipoConvocatoria{nil, {datospersonales.TipoPromocionInterna}, {datospersonales.TipoBolsa, datospersonales.TipoBolsa}, {"otro"}} {
		if _, err := Nuevo(resolutorPrueba{}, tipos); err == nil {
			t.Fatalf("tipos %v aceptados", tipos)
		}
	}
	a, _ = Nuevo(resolutorPrueba{datos: map[datospersonales.TipoConvocatoria][]datospersonales.DatoRequerido{}}, []datospersonales.TipoConvocatoria{datospersonales.TipoBolsa})
	if e, err := a.ExigenciasContactoFichaPropia(context.Background()); err != nil || len(e.Campos) != 0 {
		t.Fatalf("sin datos no se pide nada: %+v %v", e, err)
	}
}

type relojFijo struct{}

func (relojFijo) Ahora() time.Time { return time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC) }

// Con el paquete de ejemplo real (pendiente de RRHH y del DPD).
func TestPaqueteDeEjemploReal(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../../../data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := datospersonales.NuevoResolutor(consulta, consulta, relojFijo{})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Nuevo(r, []datospersonales.TipoConvocatoria{datospersonales.TipoBolsa})
	e, err := a.ExigenciasContactoFichaPropia(context.Background())
	if err != nil || !e.Ejemplo || len(e.Campos) != 4 || e.Campos[0] != (domain.ExigenciaCampo{Campo: domain.CampoTelefono, Obligatorio: true}) {
		t.Fatalf("%+v %v", e, err)
	}
}
