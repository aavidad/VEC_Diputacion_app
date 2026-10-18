package application

import (
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestCanonOfertaModalidadYCausaF111(t *testing.T) {
	datos := datosOfertaPrueba()
	base := huellaMaterialRecuperarOfertaV1("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, datos, nil, dominiobolsa.NotificacionOferta{})
	if base != "4cf4b94a62b90c0867ebba6d0c87616b5012f6f0bbc8a49f2820948712da54f3" {
		t.Fatalf("canon legado cambiado: %s", base)
	}
	publicada := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	plazo := puertosbolsa.PlazoOferta{Inicio: dominiobolsa.InicioPlazoOfertaPublicacion, ReglaRef: "regla:1", Calendarios: []string{"cal:1"}}
	publicaBase := huellaMaterialPublicarOferta("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, datos, nil, publicada, publicada.Add(48*time.Hour), plazo)
	if publicaBase != "e581107cf7831badd418840dfeec4590e1f59459434141d8e7eb5b425f1c9bbd" {
		t.Fatalf("canon de publicación legado cambiado: %s", publicaBase)
	}
	datos.ModalidadClave, datos.CausaClave = "interinidad", "sustitucion"
	recupera := huellaMaterialRecuperarOfertaV1("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, datos, nil, dominiobolsa.NotificacionOferta{})
	publica := huellaMaterialPublicarOferta("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, datos, nil, publicada, publicada.Add(48*time.Hour), plazo)
	if recupera == base || publica == publicaBase {
		t.Fatal("las claves no modifican ambas huellas")
	}
	for _, cambiar := range []func(*dominiobolsa.DatosOferta){
		func(d *dominiobolsa.DatosOferta) { d.ModalidadClave = "laboral" },
		func(d *dominiobolsa.DatosOferta) { d.CausaClave = "vacante" },
	} {
		otra := datos
		cambiar(&otra)
		if huellaMaterialRecuperarOfertaV1("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, otra, nil, dominiobolsa.NotificacionOferta{}) == recupera ||
			huellaMaterialPublicarOferta("bolsa:of", "oferta-clave-0001", "per_0123456789abcdefghijkl", 3, otra, nil, publicada, publicada.Add(48*time.Hour), plazo) == publica {
			t.Fatal("cambio de una clave no afecta a ambas huellas")
		}
	}
}

func TestPublicarOfertaF111ExigeOrigenParaClaves(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Datos.ModalidadClave, q.Datos.CausaClave = "interinidad", "sustitucion"
	repo := &repositorioOfertasPrueba{}
	servicio, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) || repo.publicado != nil {
		t.Fatalf("datos sin origen admitidos: %v", err)
	}
}
