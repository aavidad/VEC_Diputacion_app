package administracion

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

type fuenteIdentificadoresComposicionPrueba struct{}

func (*fuenteIdentificadoresComposicionPrueba) ResolverIdentificadoresADMIN(
	context.Context, adminperfiles.ReferenciaFuenteIdentificadoresADMIN,
) (adminperfiles.IdentificadoresFuenteADMIN, error) {
	return adminperfiles.IdentificadoresFuenteADMIN{}, errors.New("fuente no llamada")
}

type seudonimizadorComposicionPrueba struct{}

func (*seudonimizadorComposicionPrueba) SeudonimizarAlta(
	context.Context, identidad.IdentificadoresAlta,
) (identidad.SeudonimosAlta, error) {
	return identidad.SeudonimosAlta{}, errors.New("proveedor no llamado")
}

type fuenteAutorizacionComposicionPrueba struct{}

func (*fuenteAutorizacionComposicionPrueba) ObtenerInstantaneaAutorizacion(
	context.Context, string, string,
) (domain.InstantaneaAutorizacion, error) {
	return domain.InstantaneaAutorizacion{}, errors.New("fuente no llamada")
}

type auditorComposicionPrueba struct{}

func (*auditorComposicionPrueba) RegistrarDenegacionADMIN(
	context.Context, api.DenegacionADMIN,
) error {
	return errors.New("auditor no llamado")
}

type relojComposicionPrueba struct{}

func (*relojComposicionPrueba) Ahora() time.Time { return time.Now().UTC() }

func dependenciasRuntimeComposicionPrueba() DependenciasComposicionPerfiles {
	return DependenciasComposicionPerfiles{
		Confianza:                  ConfianzaPerfilesV3{Fuente: new(fuenteAutorizacionComposicionPrueba)},
		PoolCuentas:                new(pgxpool.Pool),
		PoolRegistroSesion:         new(pgxpool.Pool),
		PoolRevalidacionSesion:     new(pgxpool.Pool),
		PoolContextoADMIN:          new(pgxpool.Pool),
		Seudonimizador:             new(seudonimizadorComposicionPrueba),
		FuenteIdentificadoresADMIN: new(fuenteIdentificadoresComposicionPrueba),
		ConfiguracionContextoADMIN: adminperfiles.ConfiguracionContextoADMIN{Proceso: "vec_admin_prueba"},
		Auditor:                    new(auditorComposicionPrueba),
		Reloj:                      new(relojComposicionPrueba),
		Activos:                    fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("prueba")}},
	}
}

func TestComposicionADMINRechazaFuenteYPoolContextoNoSegregados(t *testing.T) {
	var fuenteNula *fuenteIdentificadoresComposicionPrueba
	var auditorNulo *auditorComposicionPrueba
	casos := []struct {
		nombre  string
		cambiar func(*DependenciasComposicionPerfiles)
	}{
		{"fuente ausente", func(d *DependenciasComposicionPerfiles) { d.FuenteIdentificadoresADMIN = nil }},
		{"fuente tipada nula", func(d *DependenciasComposicionPerfiles) { d.FuenteIdentificadoresADMIN = fuenteNula }},
		{"auditor tipado nulo", func(d *DependenciasComposicionPerfiles) { d.Auditor = auditorNulo }},
		{"seudonimizador ausente", func(d *DependenciasComposicionPerfiles) { d.Seudonimizador = nil }},
		{"contexto ausente", func(d *DependenciasComposicionPerfiles) { d.PoolContextoADMIN = nil }},
		{"cuentas prestadas a contexto", func(d *DependenciasComposicionPerfiles) { d.PoolContextoADMIN = d.PoolCuentas }},
		{"registro prestado a contexto", func(d *DependenciasComposicionPerfiles) { d.PoolContextoADMIN = d.PoolRegistroSesion }},
		{"revalidacion prestada a contexto", func(d *DependenciasComposicionPerfiles) { d.PoolContextoADMIN = d.PoolRevalidacionSesion }},
		{"proceso contexto ausente", func(d *DependenciasComposicionPerfiles) { d.ConfiguracionContextoADMIN.Proceso = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			deps := dependenciasRuntimeComposicionPrueba()
			caso.cambiar(&deps)
			servidor, err := ComponerServidorPerfiles(context.Background(), Configuracion{}, deps)
			if servidor != nil || !errors.Is(err, ErrConfiguracion) {
				t.Fatal("un runtime ADMIN sin autoridad propia salió del guard inicial")
			}
		})
	}
}
