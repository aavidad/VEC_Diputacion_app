package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Dobles que no se llaman: la composición solo publica y lee la política.
type contextoSituacionNoUsadoPrueba struct {
	puertosbolsa.ResolutorContextoSituacionParticipacion
}
type autorizadorSituacionNoUsadoPrueba struct {
	puertosbolsa.AutorizadorSituacionParticipacionV3
}

// repositorioPoliticaTransicionesComposicionPrueba imita una base con la
// migración 000032.
type repositorioPoliticaTransicionesComposicionPrueba struct {
	puertosbolsa.RepositorioSituacionParticipacion
	vigente    puertosbolsa.PoliticaTransicionesVigente
	err        error
	publicadas []puertosbolsa.PublicacionPoliticaTransicionesSituacion
}

func (r *repositorioPoliticaTransicionesComposicionPrueba) PoliticaTransicionesSituacion(context.Context) (puertosbolsa.PoliticaTransicionesVigente, error) {
	return r.vigente, r.err
}

func (r *repositorioPoliticaTransicionesComposicionPrueba) PublicarPoliticaTransicionesSituacion(_ context.Context, p puertosbolsa.PublicacionPoliticaTransicionesSituacion) (puertosbolsa.PoliticaTransicionesVigente, error) {
	if r.err != nil {
		return puertosbolsa.PoliticaTransicionesVigente{}, r.err
	}
	r.publicadas = append(r.publicadas, p)
	r.vigente = puertosbolsa.PoliticaTransicionesVigente{Version: 2, CatalogoRef: p.CatalogoRef, Politica: p.Politica}
	return r.vigente, nil
}

func servicioSituacionComposicionPrueba(t *testing.T, repo puertosbolsa.RepositorioSituacionParticipacion) *aplicacionbolsa.ServicioSituacionParticipacion {
	t.Helper()
	servicio, err := aplicacionbolsa.NuevoServicioSituacionParticipacion(contextoSituacionNoUsadoPrueba{}, autorizadorSituacionNoUsadoPrueba{}, repo, relojPresentacionReglasEjemplo.Ahora)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func TestComposicionPublicaLaPoliticaDeTransicionesDelCatalogo(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioPoliticaTransicionesComposicionPrueba{}
	ruta, err := componerReglasSituacionBolsaDesarrollo(compuestas.bolsa, &manejadorParticipacionBolsaDesarrollo{servicioSituacion: servicioSituacionComposicionPrueba(t, repo)})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.publicadas) != 1 || !slices.Contains(repo.publicadas[0].Politica.Pares(), "renuncia>no_disponible") ||
		slices.Contains(repo.publicadas[0].Politica.Pares(), "renuncia>disponible") || len(repo.publicadas[0].CatalogoSHA256) != 64 {
		t.Fatalf("publicación: %+v", repo.publicadas)
	}
	codigo, cuerpo := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo)
	if codigo != 200 || !slices.Equal(cuerpo.Data.Transiciones["renuncia"], []string{"no_disponible", "excluido"}) {
		t.Fatalf("la pantalla ofrece lo publicado: %d %v", codigo, cuerpo.Data.Transiciones)
	}
}

func TestComposicionNoArrancaSiLaBaseRechazaLaPolitica(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioPoliticaTransicionesComposicionPrueba{err: puertosbolsa.ErrSituacionParticipacionNoDisponible}
	if _, err := componerReglasSituacionBolsaDesarrollo(compuestas.bolsa, &manejadorParticipacionBolsaDesarrollo{servicioSituacion: servicioSituacionComposicionPrueba(t, repo)}); !errors.Is(err, errPoliticaTransicionesEjemploNoValida) {
		t.Fatalf("una base que rechaza la política impide arrancar: %v", err)
	}
}

func TestComposicionSinCatalogoNoPublicaYMuestraLaPoliticaDeLaBase(t *testing.T) {
	repo := &repositorioPoliticaTransicionesComposicionPrueba{}
	servicio := servicioSituacionComposicionPrueba(t, repo)
	// La base ya tiene publicada la política del Reglamento de un arranque anterior.
	if _, err := repo.PublicarPoliticaTransicionesSituacion(context.Background(), puertosbolsa.PublicacionPoliticaTransicionesSituacion{Politica: politicaReglamentoComposicionPrueba(t)}); err != nil {
		t.Fatal(err)
	}
	ruta, err := componerReglasSituacionBolsaDesarrollo(nil, &manejadorParticipacionBolsaDesarrollo{servicioSituacion: servicio})
	if err != nil || len(repo.publicadas) != 1 {
		t.Fatalf("sin catálogo no se publica: %v %d", err, len(repo.publicadas))
	}
	codigo, cuerpo := consultarReglasSituacionPrueba(t, ruta.Manejador, rutaReglasSituacionBolsaDesarrollo)
	if codigo != 200 || cuerpo.Data.Configuradas || !slices.Equal(cuerpo.Data.Transiciones["renuncia"], []string{"no_disponible", "excluido"}) {
		t.Fatalf("sin catálogo rige la política de la base: %d %+v", codigo, cuerpo.Data)
	}
}

func politicaReglamentoComposicionPrueba(t *testing.T) dominiobolsa.PoliticaTransicionesSituacion {
	t.Helper()
	tabla := map[string][]string{}
	for _, origen := range dominiobolsa.SituacionesParticipacion() {
		tabla[origen] = dominiobolsa.DestinosSituacionParticipacion(origen)
	}
	tabla[dominiobolsa.SituacionRenuncia] = []string{dominiobolsa.SituacionNoDisponible, dominiobolsa.SituacionExcluido}
	politica, err := dominiobolsa.NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}
