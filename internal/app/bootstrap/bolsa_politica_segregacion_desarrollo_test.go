package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const rutaRolesSegregacionEjemploPrueba = "../../../data/demo/reglas/bolsa_roles_segregacion.demo.json"

type publicadorPoliticaSegregacionPrueba struct {
	publicadas []puertosbolsa.PublicacionPoliticaSegregacion
	err        error
}

func (p *publicadorPoliticaSegregacionPrueba) PublicarPoliticaSegregacion(_ context.Context, publicacion puertosbolsa.PublicacionPoliticaSegregacion) (puertosbolsa.PoliticaSegregacionVigente, error) {
	p.publicadas = append(p.publicadas, publicacion)
	return puertosbolsa.PoliticaSegregacionVigente{Version: 2, CatalogoRef: publicacion.CatalogoRef, Politica: publicacion.Politica}, p.err
}

func configuracionRolesSegregacionPrueba(ruta string) config.Config {
	cfg := configuracionDesarrolloReglasEjemplo("", "")
	cfg.ReglasEjemplo.BolsaRolesSegregacionSourcePath = ruta
	return cfg
}

func TestPoliticaSegregacionEjemploSePublicaDesdeElCatalogo(t *testing.T) {
	publicador := &publicadorPoliticaSegregacionPrueba{}
	if err := publicarPoliticaSegregacionDesarrollo(context.Background(), configuracionRolesSegregacionPrueba(rutaRolesSegregacionEjemploPrueba), publicador, relojPresentacionReglasEjemplo); err != nil {
		t.Fatal(err)
	}
	if len(publicador.publicadas) != 1 {
		t.Fatalf("publicaciones=%d", len(publicador.publicadas))
	}
	p := publicador.publicadas[0]
	if !slices.Equal(p.Politica.Operaciones(), []string{dominiobolsa.OperacionExcluir}) ||
		p.CatalogoRef != "vec.bolsa.roles_segregacion:1:s01.segunda_persona" || len(p.CatalogoSHA256) != 64 {
		t.Fatalf("publicación inesperada: %+v %v", p, p.Politica.Operaciones())
	}
}

func TestPoliticaSegregacionSinCatalogoNoPublica(t *testing.T) {
	publicador := &publicadorPoliticaSegregacionPrueba{}
	if err := publicarPoliticaSegregacionDesarrollo(context.Background(), configuracionRolesSegregacionPrueba(""), publicador, relojPresentacionReglasEjemplo); err != nil || len(publicador.publicadas) != 0 {
		t.Fatalf("sin catálogo nada cambia: %v %d", err, len(publicador.publicadas))
	}
}

func TestPoliticaSegregacionInvalidaOFallidaImpideArrancar(t *testing.T) {
	original, err := os.ReadFile(rutaRolesSegregacionEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	sinExclusion := filepath.Join(t.TempDir(), "roles.demo.json")
	if err := os.WriteFile(sinExclusion, []byte(strings.Replace(string(original), `"valor": "excluir"`, `"valor": "pausar"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	publicador := &publicadorPoliticaSegregacionPrueba{}
	if err := publicarPoliticaSegregacionDesarrollo(context.Background(), configuracionRolesSegregacionPrueba(sinExclusion), publicador, relojPresentacionReglasEjemplo); !errors.Is(err, errPoliticaSegregacionEjemploNoValida) || len(publicador.publicadas) != 0 {
		t.Fatalf("un catálogo sin exclusión debe impedir arrancar: %v", err)
	}
	caido := &publicadorPoliticaSegregacionPrueba{err: errors.New("base caida")}
	if err := publicarPoliticaSegregacionDesarrollo(context.Background(), configuracionRolesSegregacionPrueba(rutaRolesSegregacionEjemploPrueba), caido, relojPresentacionReglasEjemplo); !errors.Is(err, errPoliticaSegregacionEjemploNoValida) {
		t.Fatalf("una publicación fallida debe impedir arrancar: %v", err)
	}
	produccion := config.Config{ReglasEjemplo: config.ConfiguracionReglasEjemplo{BolsaRolesSegregacionSourcePath: rutaRolesSegregacionEjemploPrueba}}
	if err := publicarPoliticaSegregacionDesarrollo(context.Background(), produccion, publicador, relojPresentacionReglasEjemplo); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("fuera de desarrollo debe impedir arrancar: %v", err)
	}
}
