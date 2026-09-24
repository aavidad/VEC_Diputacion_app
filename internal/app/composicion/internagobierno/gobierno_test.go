package internagobierno

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

type revalidadorInalcanzable struct{}

func (revalidadorInalcanzable) RevalidarAutenticacionActorV1(context.Context, core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	panic("sin cápsula no debe revalidar")
}

type resolutorInalcanzable struct{}

func (resolutorInalcanzable) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	panic("sin cápsula no debe consultar F1")
}

func configuracionFuentePrueba() ConfiguracionFuenteF1 {
	ahora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	return ConfiguracionFuenteF1{
		Identidad:     new(httpseguridad.ServicioIdentidad),
		Revalidador:   revalidadorInalcanzable{},
		Resolutor:     resolutorInalcanzable{},
		Reloj:         relojPrueba{ahora},
		PorCuenta:     map[string]ContextoNominal{"cta_0123456789abcdef0123456789abcdef": {PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef", OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64)}},
		MotivoAlta:    core.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "alta"},
		MotivoLectura: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "lectura"},
		Politica: inc.PoliticaConsultaDesarrollo{Tipo: httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal,
			Referencia: "pga_0123456789abcdef0123456789abcdef", HuellaSHA256: strings.Repeat("a", 64), RetiradaEn: ahora.Add(time.Hour)},
	}
}

func TestFuenteF1DeniegaSinCapsulaYNoLlamaF1(t *testing.T) {
	f, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.PeticionVerificada(context.Background()); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("sin cápsula: %v", err)
	}
	if _, err := f.ResolverContexto(context.Background()); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("sin cápsula: %v", err)
	}
}

func TestFuenteF1DeniegaPoliticaRetiradaYSelectorInvalido(t *testing.T) {
	c := configuracionFuentePrueba()
	c.Politica.RetiradaEn = c.Reloj.Ahora()
	if _, err := NuevaFuenteF1(c); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("política retirada: %v", err)
	}
	c = configuracionFuentePrueba()
	c.PorCuenta["cta_0123456789abcdef0123456789abcdef"] = ContextoNominal{PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef", OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "unidad:con espacio"}
	if _, err := NuevaFuenteF1(c); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("unidad no canónica: %v", err)
	}
}

func TestPostgreSQLSinPoolsFallaAntesDeArrancar(t *testing.T) {
	ctx := context.Background()
	if _, err := NuevoIdentidadPostgreSQL(ctx, nil, nil, nil, "", ""); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("identidad: %v", err)
	}
	if _, err := NuevoContextoPostgreSQL(ctx, nil, nil, relojPrueba{time.Now()}); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("contexto: %v", err)
	}
	if _, err := NuevoRegistradorAuditoriaPostgreSQL(ctx, nil); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("auditoría: %v", err)
	}
}

func TestAutoridadRutaSeguimientoDeniegaSinCapsulaYOtraRuta(t *testing.T) {
	f, err := NuevaFuenteF1(configuracionFuentePrueba())
	if err != nil {
		t.Fatal(err)
	}
	a, err := NuevaAutoridadRutaSeguimiento(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{httpct.RutaConsultaSeguimientoV2, "/api/vec/contratacion-temporal/otra"} {
		if err := a.AutorizarRutaExacta(context.Background(), ruta); !errors.Is(err, httpapi.ErrAccesoRutaExactaDenegado) {
			t.Fatalf("ruta %q sin contexto vivo: %v", ruta, err)
		}
	}
	if _, err := NuevaAutoridadRutaSeguimiento(nil); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("fuente ausente: %v", err)
	}
}
