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
	vecports "vec-diputacion-granada/internal/vec/ports"
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

// autoridadCorporativaPrueba imita ContextoActor 000009: responde con el estado
// actual del vínculo corporativo en cada llamada, sin caché.
type autoridadCorporativaPrueba struct {
	vigente     bool
	errorFuente error
	solicitudes []vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1
}

func (a *autoridadCorporativaPrueba) RevalidarVinculoCorporativoRRHHV1(_ context.Context, s vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1) error {
	a.solicitudes = append(a.solicitudes, s)
	switch {
	case a.errorFuente != nil:
		return a.errorFuente
	case !a.vigente:
		return vecports.ErrVinculoCorporativoRRHHNoVigente
	}
	return nil
}

func configuracionFuentePrueba() ConfiguracionFuenteF1 {
	ahora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	return ConfiguracionFuenteF1{
		Identidad:          new(httpseguridad.ServicioIdentidad),
		Revalidador:        revalidadorInalcanzable{},
		Resolutor:          resolutorInalcanzable{},
		VinculoCorporativo: &autoridadCorporativaPrueba{},
		Reloj:              relojPrueba{ahora},
		PorCuenta:          map[string]ContextoNominal{"cta_0123456789abcdef0123456789abcdef": {PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef", OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64)}},
		MotivoAlta:         core.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "alta"},
		MotivoLectura:      core.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "lectura"},
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

func TestFuenteF1ExigeRevalidadorCorporativo(t *testing.T) {
	c := configuracionFuentePrueba()
	c.VinculoCorporativo = nil
	if _, err := NuevaFuenteF1(c); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("sin revalidador corporativo: %v", err)
	}
}

// Cada petición vuelve a preguntar a la autoridad: revocar el vínculo
// corporativo deniega la siguiente sin detalle y restituirlo con versión nueva
// vuelve a autorizar. La indisponibilidad también deniega.
func TestFuenteF1RevalidaVinculoCorporativoEnCadaPeticion(t *testing.T) {
	autoridad := &autoridadCorporativaPrueba{vigente: true}
	c := configuracionFuentePrueba()
	c.VinculoCorporativo = autoridad
	f, err := NuevaFuenteF1(c)
	if err != nil {
		t.Fatal(err)
	}
	datos := core.DatosVinculoAutenticacionActorV2{
		CuentaRef: "cta_0123456789abcdef0123456789abcdef", PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef",
		PrincipalID: "per_0123456789abcdef0123456789abcdef", ContextoActorRef: "vca_0123456789abcdef0123456789abcdef",
		ContextoActorVersion: 7,
	}
	ctx := context.Background()
	if err := f.exigirVinculoCorporativoVigente(ctx, datos); err != nil {
		t.Fatalf("vigente: %v", err)
	}
	want := vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1{CuentaRef: datos.CuentaRef, PerfilRef: datos.PerfilActivoRef,
		PersonaRef: datos.PrincipalID, VinculoContextoRef: datos.ContextoActorRef, VinculoContextoVersion: 7}
	if autoridad.solicitudes[0] != want {
		t.Fatalf("solicitud = %+v", autoridad.solicitudes[0])
	}
	autoridad.vigente = false // revocación publicada por la fuente corporativa
	err = f.exigirVinculoCorporativoVigente(ctx, datos)
	if !errors.Is(err, ErrGobiernoInternoNoDisponible) || errors.Is(err, vecports.ErrVinculoCorporativoRRHHNoVigente) {
		t.Fatalf("tras revocar, la siguiente petición debía denegar sin motivo: %v", err)
	}
	autoridad.vigente = true // restitución con versión nueva
	if err := f.exigirVinculoCorporativoVigente(ctx, datos); err != nil {
		t.Fatalf("restituido: %v", err)
	}
	autoridad.errorFuente = vecports.ErrVinculoCorporativoRRHHNoDisponible
	if err := f.exigirVinculoCorporativoVigente(ctx, datos); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("indisponible: %v", err)
	}
	if len(autoridad.solicitudes) != 4 {
		t.Fatalf("la fuente conservó una respuesta: %d consultas para 4 peticiones", len(autoridad.solicitudes))
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := f.exigirVinculoCorporativoVigente(cancelado, datos); !errors.Is(err, ErrGobiernoInternoNoDisponible) || len(autoridad.solicitudes) != 4 {
		t.Fatalf("contexto cancelado: %v", err)
	}
}
