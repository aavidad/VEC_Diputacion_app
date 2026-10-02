package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestPoliticaTransicionesCompiladaEsLaVersion1De000032(t *testing.T) {
	var cero PoliticaTransicionesSituacion
	for _, politica := range []PoliticaTransicionesSituacion{cero, PoliticaTransicionesSituacionCompilada()} {
		pares := politica.Pares()
		if len(pares) != 18 || !slices.Contains(pares, "renuncia>disponible") || !slices.Contains(pares, "disponible>disponible_desde") || slices.Contains(pares, "renuncia>no_disponible") ||
			!politica.Admite(SituacionRenuncia, SituacionDisponible) || politica.Admite(SituacionExcluido, SituacionDisponible) {
			t.Fatalf("compilada: %v", pares)
		}
	}
}

func TestPoliticaTransicionesDelReglamentoYSuFormaCanonica(t *testing.T) {
	tabla := map[string][]string{}
	for _, origen := range SituacionesParticipacion() {
		tabla[origen] = DestinosSituacionParticipacion(origen)
	}
	tabla[SituacionRenuncia] = []string{SituacionExcluido, SituacionNoDisponible}
	politica, err := NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(politica.Destinos(SituacionRenuncia), []string{SituacionNoDisponible, SituacionExcluido}) ||
		politica.Admite(SituacionRenuncia, SituacionDisponible) {
		t.Fatalf("renuncia: %v", politica.Destinos(SituacionRenuncia))
	}
	leida, err := PoliticaTransicionesDesdePares(politica.Pares())
	if err != nil || !slices.Equal(leida.Pares(), politica.Pares()) {
		t.Fatalf("ida y vuelta: %v %v", leida.Pares(), err)
	}
	copia := politica.Destinos(SituacionRenuncia)
	copia[0] = SituacionDisponible
	if politica.Admite(SituacionRenuncia, SituacionDisponible) {
		t.Fatal("Destinos no devuelve una copia")
	}
	restringida := politica.Restringir(SituacionRenuncia, []string{SituacionExcluido})
	if restringida.Admite(SituacionRenuncia, SituacionNoDisponible) || !politica.Admite(SituacionRenuncia, SituacionNoDisponible) {
		t.Fatal("Restringir altera la original o no restringe")
	}
	cambio := CambioSituacionParticipacion{ParticipacionRef: "participacion:b2", Origen: SituacionRenuncia, Destino: SituacionNoDisponible, Motivo: "Renuncia justificada"}
	if cambio.Validar() == nil {
		t.Fatal("la compilada no admite renuncia→no_disponible")
	}
}

func TestPoliticaTransicionesRechazaLasQueRompenLasInvariantes(t *testing.T) {
	base := func() map[string][]string {
		tabla := map[string][]string{}
		for _, origen := range SituacionesParticipacion() {
			tabla[origen] = DestinosSituacionParticipacion(origen)
		}
		return tabla
	}
	casos := map[string]func(map[string][]string){
		"excluido a pausa":    func(t map[string][]string) { t[SituacionExcluido] = []string{SituacionNoDisponible} },
		"a sí misma":          func(t map[string][]string) { t[SituacionRenuncia] = append(t[SituacionRenuncia], SituacionRenuncia) },
		"sin baja definitiva": func(t map[string][]string) { t[SituacionRenuncia] = []string{SituacionNoDisponible} },
		"destino desconocido": func(t map[string][]string) { t[SituacionRenuncia] = append(t[SituacionRenuncia], "readmitido") },
		"origen desconocido":  func(t map[string][]string) { t["readmitido"] = []string{SituacionExcluido} },
		"repetido":            func(t map[string][]string) { t[SituacionRenuncia] = append(t[SituacionRenuncia], SituacionExcluido) },
	}
	for nombre, alterar := range casos {
		tabla := base()
		alterar(tabla)
		if _, err := NuevaPoliticaTransicionesSituacion(tabla); !errors.Is(err, ErrPoliticaTransicionesInvalida) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	for _, pares := range [][]string{nil, {"renuncia"}, {"a>b>c"}, {"renuncia>excluido"}} {
		if _, err := PoliticaTransicionesDesdePares(pares); !errors.Is(err, ErrPoliticaTransicionesInvalida) {
			t.Errorf("%v: %v", pares, err)
		}
	}
}

func TestPoliticaTransicionesReincorporacionRRHHSoloPublicadaADisponible(t *testing.T) {
	tabla := map[string][]string{}
	for _, origen := range SituacionesParticipacion() {
		tabla[origen] = DestinosSituacionParticipacion(origen)
	}
	tabla[SituacionExcluido] = []string{SituacionDisponible}
	politica, err := NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil || !politica.Admite(SituacionExcluido, SituacionDisponible) {
		t.Fatalf("política de reincorporación: %v", err)
	}
	leida, err := PoliticaTransicionesDesdePares(politica.Pares())
	if err != nil || !slices.Equal(leida.Pares(), politica.Pares()) {
		t.Fatalf("política publicada: %v", err)
	}
	if PoliticaTransicionesSituacionCompilada().Admite(SituacionExcluido, SituacionDisponible) {
		t.Fatal("sin publicación no se habilita la reincorporación")
	}
	for _, destino := range SituacionesParticipacion() {
		if destino != SituacionDisponible && politica.Admite(SituacionExcluido, destino) {
			t.Fatalf("salida de excluido distinta de disponible: %s", destino)
		}
	}
}

func TestPoliticaRevisionConservaHistorialYExigeSalidaDeBaja(t *testing.T) {
	base := map[string][]string{}
	for _, origen := range SituacionesParticipacion() {
		base[origen] = DestinosSituacionParticipacion(origen)
	}
	antigua, err := NuevaPoliticaTransicionesSituacion(base)
	if err != nil || len(antigua.Pares()) != 18 {
		t.Fatalf("política histórica alterada: %v, %v", antigua.Pares(), err)
	}
	base[SituacionRenuncia] = []string{SituacionEnRevision, SituacionExcluido}
	if _, err := NuevaPoliticaTransicionesSituacion(base); !errors.Is(err, ErrPoliticaTransicionesInvalida) {
		t.Fatalf("revisión sin salida a exclusión: %v", err)
	}
	base[SituacionEnRevision] = []string{SituacionDisponible, SituacionExcluido}
	nueva, err := NuevaPoliticaTransicionesSituacion(base)
	if err != nil || !nueva.Admite(SituacionRenuncia, SituacionEnRevision) || !nueva.Admite(SituacionEnRevision, SituacionDisponible) {
		t.Fatalf("política revisión: %v, %v", nueva.Pares(), err)
	}
	leida, err := PoliticaTransicionesDesdePares(nueva.Pares())
	if err != nil || !slices.Equal(leida.Pares(), nueva.Pares()) {
		t.Fatalf("ida y vuelta de revisión: %v", err)
	}
}
