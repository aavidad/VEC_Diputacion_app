package application

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const candidatoModalidadesPrueba = "can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func opcionesModalidadesPrueba() []ports.ModalidadPreferencia {
	var opciones []ports.ModalidadPreferencia
	for _, clave := range []string{"vacante", "programa", "sustitucion"} {
		opciones = append(opciones, ports.ModalidadPreferencia{Clave: clave,
			Fuente: dominiovec.ReferenciaEntradaCatalogo{
				CatalogoID: "vec.contratacion_temporal.reglas", CatalogoVersion: 3,
				CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "c12.modalidad." + clave,
			}})
	}
	return opciones
}

func TestHuellaSolicitudPreferenciasModalidadesGoldenYOrden(t *testing.T) {
	opciones := opcionesModalidadesPrueba()
	seleccion := []string{"vacante", "sustitucion"}
	got, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 7, seleccion, opciones)
	const golden = "2b894c9c699723d958f3c0ca679e632ef478651966d99e2d64656bf29e26c76a"
	if err != nil || got != golden {
		t.Fatalf("framing de bytes distinto del vector independiente: %s, %v", got, err)
	}
	if !slices.Equal(seleccion, []string{"vacante", "sustitucion"}) ||
		!slices.EqualFunc(opciones, opcionesModalidadesPrueba(), func(a, b ports.ModalidadPreferencia) bool { return a == b }) {
		t.Fatal("el cálculo alteró las entradas")
	}
	slices.Reverse(seleccion)
	slices.Reverse(opciones)
	reordenada, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 7, seleccion, opciones)
	if err != nil || reordenada != golden {
		t.Fatalf("el orden de entrada cambió la huella: %s, %v", reordenada, err)
	}
	if !slices.Equal(seleccion, []string{"sustitucion", "vacante"}) || opciones[0].Clave != "sustitucion" {
		t.Fatal("el cálculo ordenó los argumentos originales")
	}
}

func TestHuellaSolicitudPreferenciasModalidadesDistingueAusenciaYFuente(t *testing.T) {
	opciones := opcionesModalidadesPrueba()
	todas, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, nil, opciones)
	if err != nil {
		t.Fatal(err)
	}
	ninguna, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, []string{}, opciones)
	if err != nil || todas == ninguna {
		t.Fatalf("nil y selección vacía deben tener huellas distintas: %s/%s %v", todas, ninguna, err)
	}
	version, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 1, nil, opciones)
	if err != nil || version == todas {
		t.Fatal("la versión no quedó ligada")
	}
	conOtraFuente := opcionesModalidadesPrueba()
	conOtraFuente[0].Fuente.CatalogoVersion = 4
	fuente, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, nil, conOtraFuente)
	if err != nil || fuente == todas {
		t.Fatal("la versión del catálogo no quedó ligada")
	}
	conOtraFuente = opcionesModalidadesPrueba()
	conOtraFuente[0].Fuente.CatalogoHuellaSHA256 = strings.Repeat("c", 64)
	fuente, err = HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, nil, conOtraFuente)
	if err != nil || fuente == todas {
		t.Fatal("la huella del catálogo no quedó ligada")
	}
	conOtraFuente = opcionesModalidadesPrueba()
	conOtraFuente = append(conOtraFuente, ports.ModalidadPreferencia{Clave: "relevo",
		Fuente: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "vec.contratacion_temporal.reglas", CatalogoVersion: 3,
			CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "c12.modalidad.relevo",
		}})
	fuente, err = HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, nil, conOtraFuente)
	if err != nil || fuente == todas {
		t.Fatal("el conjunto disponible no quedó ligado")
	}
}

func TestHuellaSolicitudPreferenciasModalidadesEncuadraClaveRetirada(t *testing.T) {
	opciones := opcionesModalidadesPrueba()
	actual, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 3, []string{"vacante"}, opciones)
	if err != nil {
		t.Fatal(err)
	}
	retirada, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 3, []string{"vacante", "relevo"}, opciones)
	if err != nil || retirada == actual {
		t.Fatalf("clave retirada no quedó ligada a la huella: %s %v", retirada, err)
	}
	opciones[0].Fuente.CatalogoVersion++
	conFuenteNueva, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 3, []string{"vacante", "relevo"}, opciones)
	if err != nil || conFuenteNueva == retirada {
		t.Fatalf("instantánea nueva no quedó ligada a la huella: %s %v", conFuenteNueva, err)
	}
}

func TestHuellaSolicitudPreferenciasModalidadesRechazos(t *testing.T) {
	opciones := opcionesModalidadesPrueba()
	paraSeleccion := []struct {
		nombre    string
		candidato string
		version   uint64
		seleccion []string
	}{
		{"candidato", "can_corto", 0, nil},
		{"versión", candidatoModalidadesPrueba, 2147483647, nil},
		{"repetida", candidatoModalidadesPrueba, 0, []string{"vacante", "vacante"}},
		{"clave inválida", candidatoModalidadesPrueba, 0, []string{"Vacante"}},
	}
	for _, tc := range paraSeleccion {
		t.Run(tc.nombre, func(t *testing.T) {
			_, err := HuellaSolicitudPreferenciasModalidades(tc.candidato, tc.version, tc.seleccion, opciones)
			if !errors.Is(err, ports.ErrOrdenPreferenciasInvalido) {
				t.Fatalf("error de selección: %v", err)
			}
		})
	}
	paraCatalogo := []struct {
		nombre   string
		opciones []ports.ModalidadPreferencia
	}{
		{"ausente", nil},
		{"duplicada", append(opcionesModalidadesPrueba(), opciones[0])},
		{"fuente sin huella", func() []ports.ModalidadPreferencia {
			a := opcionesModalidadesPrueba()
			a[0].Fuente.CatalogoHuellaSHA256 = "mal"
			return a
		}()},
		{"entrada ajena", func() []ports.ModalidadPreferencia {
			a := opcionesModalidadesPrueba()
			a[0].Fuente.EntradaClave = "c12.modalidad.otro"
			return a
		}()},
	}
	for _, tc := range paraCatalogo {
		t.Run(tc.nombre, func(t *testing.T) {
			_, err := HuellaSolicitudPreferenciasModalidades(candidatoModalidadesPrueba, 0, nil, tc.opciones)
			if !errors.Is(err, ports.ErrPreferenciasNoDisponibles) {
				t.Fatalf("error de catálogo: %v", err)
			}
		})
	}
}
