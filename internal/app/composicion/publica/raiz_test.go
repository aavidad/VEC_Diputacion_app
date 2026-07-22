package publica

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestNuevoServidorExigeAutenticacionDeshabilitada(t *testing.T) {
	for _, modo := range []string{
		config.AuthModeFake,
		config.AuthModeTrustedHeaders,
		config.AuthModeDevelopment,
	} {
		t.Run(modo, func(t *testing.T) {
			servidor, err := NuevoServidor(Configuracion{
				PerfilEjecucion:         config.ExecutionProfileProduction,
				AutenticacionSolicitada: modo,
			})
			if servidor != nil || !errors.Is(err, ErrAutenticacionNoAdmitida) {
				t.Fatalf("modo %q = (%v, %v); debe rechazarse", modo, servidor, err)
			}
		})
	}
}

func TestNuevoServidorPermaneceCerradoSinConexionAutoritativa(t *testing.T) {
	servidor, err := NuevoServidor(Configuracion{
		PerfilEjecucion:         config.ExecutionProfileProduction,
		AutenticacionSolicitada: config.AuthModeDisabled,
	})
	if servidor != nil || !errors.Is(err, ErrConfiguracionAutoritativaInvalida) ||
		!errors.Is(err, config.ErrConfiguracionPostgreSQLPublicaIncompleta) {
		t.Fatalf("raiz productiva = (%v, %v)", servidor, err)
	}
}

func TestNuevoServidorNoSustituyeCatalogoAusentePorElDemo(t *testing.T) {
	postgresql, err := config.NuevaConfiguracionPostgreSQLPublica(
		"postgres://lector@localhost/vec?sslmode=verify-full&sslrootcert=/ca.pem",
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor, err := NuevoServidor(Configuracion{
		PerfilEjecucion:         config.ExecutionProfileProduction,
		AutenticacionSolicitada: config.AuthModeDisabled,
		PostgreSQL:              postgresql,
	})
	if servidor != nil || !errors.Is(err, ErrConfiguracionAutoritativaInvalida) {
		t.Fatalf("raiz sin catalogo explicito = (%v, %v)", servidor, err)
	}
}

func TestNuevoServidorRechazaTestigoDeManifiestoReservado(t *testing.T) {
	postgresql, err := config.NuevaConfiguracionPostgreSQLPublica(
		"postgres://lector@localhost/vec?sslmode=verify-full&sslrootcert=/ca.pem",
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor, err := NuevoServidor(Configuracion{
		PerfilEjecucion:            config.ExecutionProfileProduction,
		AutenticacionSolicitada:    config.AuthModeDisabled,
		CatalogoCategorias:         "categorias-profesionales",
		VersionCategorias:          1,
		HuellaCategorias:           strings.Repeat("a", 64),
		HuellaProyeccionCategorias: strings.Repeat("b", 64),
		HuellaManifiesto:           strings.Repeat("0", 64),
		PostgreSQL:                 postgresql,
	})
	if servidor != nil || !errors.Is(err, ErrConfiguracionAutoritativaInvalida) {
		t.Fatalf("raiz con testigo reservado = (%v, %v)", servidor, err)
	}
}

func TestNuevoServidorRechazaSelectoresNoProductivos(t *testing.T) {
	pruebas := []struct {
		nombre string
		cfg    Configuracion
		error  error
	}{
		{
			nombre: "desarrollo",
			cfg: Configuracion{
				PerfilEjecucion:         config.ExecutionProfileDevelopment,
				AutenticacionSolicitada: config.AuthModeDisabled,
			},
			error: ErrActivacionDesarrolloInvalida,
		},
		{
			nombre: "presentacion",
			cfg: Configuracion{
				PerfilEjecucion:         config.ExecutionProfileRRHHPresentation,
				AutenticacionSolicitada: config.AuthModeDisabled,
			},
			error: ErrSelectoresPresentacionProhibidos,
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			servidor, err := NuevoServidor(prueba.cfg)
			if servidor != nil || !errors.Is(err, prueba.error) {
				t.Fatalf("resultado = (%v, %v); se esperaba %v", servidor, err, prueba.error)
			}
		})
	}
}
