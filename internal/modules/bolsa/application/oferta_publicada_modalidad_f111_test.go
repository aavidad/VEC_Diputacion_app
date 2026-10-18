package application

import (
	"errors"
	"reflect"
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

func solicitudCTHistoricaOfertaF111(t *testing.T, ahora time.Time) puertosbolsa.SolicitudPublicarOferta {
	t.Helper()
	q := solicitudPublicarOfertaPrueba(t, ahora)
	q.Notificacion = dominiobolsa.NotificacionOferta{}
	ordenes := []uint64{1, 2, 3}
	lectura := puertosbolsa.OrigenLlamamientoCT{
		ActaRef: "acta:prueba", CategoriaRef: "categoria:prueba", BolsaRef: q.BolsaRef,
		VersionBolsa: 1, HuellaBolsaSHA256: "huella:bolsa", InstantaneaRef: "instantanea:prueba",
		VersionInstantanea: 1, HuellaInstantaneaSHA256: "huella:instantanea",
		PoliticaRef: "politica:prueba", VersionPolitica: 1, HuellaPoliticaSHA256: "huella:politica", Corte: ahora,
	}
	for _, orden := range ordenes {
		ordenCopia := orden
		lectura.Entradas = append(lectura.Entradas, puertosbolsa.EntradaOrigenLlamamientoCT{OrdenVigente: &ordenCopia, Elegible: true})
	}
	q.OrigenCT = &puertosbolsa.OrigenOfertaCT{
		OrganizacionRef: "organizacion:prueba", UnidadRef: "unidad:prueba", ExpedienteRef: "expediente:prueba",
		VersionExpediente: 1, ClaveCT: "clave:prueba", NecesidadRef: "necesidad:prueba",
		ActaRef: lectura.ActaRef, CategoriaRef: lectura.CategoriaRef, BolsaRef: lectura.BolsaRef,
		VersionBolsa: lectura.VersionBolsa, HuellaBolsaSHA256: lectura.HuellaBolsaSHA256,
		InstantaneaRef: lectura.InstantaneaRef, VersionInstantanea: lectura.VersionInstantanea,
		HuellaInstantaneaSHA256: lectura.HuellaInstantaneaSHA256,
		PoliticaRef:             lectura.PoliticaRef, VersionPolitica: lectura.VersionPolitica,
		HuellaPoliticaSHA256: lectura.HuellaPoliticaSHA256, Lectura: lectura,
	}
	if err := q.OrigenCT.ValidarPara(q.BolsaRef, q.NumeroPlazas); err != nil {
		t.Fatalf("origen del fixture inválido: %v", err)
	}
	return q
}

func TestPublicarOfertaF111CTNuevaExigeParYRecuperaLegado(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	q := solicitudCTHistoricaOfertaF111(t, ahora)
	repo := &recuperadorOfertaPrueba{}
	calculador := &calculadorInicioOfertaPrueba{inicio: dominiobolsa.InicioPlazoOfertaPublicacion}
	servicio, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, calculador, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.PublicarOferta(t.Context(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) ||
		repo.publicado != nil || repo.consultas != 1 || calculador.consultas != 0 {
		t.Fatalf("CT nueva sin par admitida: %v", err)
	}
	original := puertosbolsa.OfertaPublicada{
		OfertaRef: "oferta:historica", ReciboRef: "recibo:oferta:historica", BolsaRef: q.BolsaRef,
		Datos: q.Datos, NumeroPlazas: q.NumeroPlazas,
		Plazo:       puertosbolsa.PlazoOferta{Inicio: dominiobolsa.InicioPlazoOfertaPublicacion},
		PublicadaEn: ahora.Add(-time.Hour), VenceAntesDe: ahora.Add(47 * time.Hour),
	}
	repo.recuperada, repo.encontrada = original, true
	repetida, err := servicio.PublicarOferta(t.Context(), q)
	if err != nil || !reflect.DeepEqual(repetida, original) || repo.publicado != nil || repo.consultas != 2 || calculador.consultas != 0 {
		t.Fatalf("replay CT histórico no conservó recibo: %+v, %v", repetida, err)
	}
}
