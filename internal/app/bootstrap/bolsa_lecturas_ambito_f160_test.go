package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// preparadorLecturasAmbitoF160 monta el preparador con la sesión y la
// instantánea publicadas de la prueba (unidad:prueba, ambito:prueba) y cuatro
// bolsas: la propia, una de otra unidad, una de otro ámbito y una sin ámbito.
func preparadorLecturasAmbitoF160(t *testing.T) (*preparadorBorradorLlamamientoDesarrollo, *fuenteLecturaActualInscripcionPrueba, context.Context) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: accionLecturaExclusionesOfrecimientoRRHH, ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
		Finalidades:      []string{finalidadLecturaExclusionesOfrecimientoRRHH},
		CamposPermitidos: slices.Clone(camposLecturaExclusionesOfrecimientoRRHH),
		GarantiaMinima:   vecdomain.AuthAssuranceHigh, Obligaciones: []string{"auditar"},
	})
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(actor.Resultado.Contexto.PerfilActivoRef)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	descriptor, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaPoliticaCese)
	if !ok {
		t.Fatal("falta la frontera GET de prueba")
	}
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodGet, ruta: bolsahttp.RutaPoliticaCese, superficie: superficieInternaSeguridadComunDesarrollo,
		catalogo: catalogo, descriptor: descriptor})
	seguridad, err := nuevaSeguridadComunDesarrollo(proveedorContextoComunConsultasRRHHDesarrolloPrueba{
		resolver: func(context.Context) (contextoSeguridadComunDesarrollo, error) { return actor, nil }}, &relojSesionConsultaPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	bolsas := &bolsasVigentesFalsas{vigentes: map[string]bool{}}
	bolsas.declarar("bolsa:prueba", bolsaports.AmbitoBolsa{UnidadRef: "unidad:prueba", AmbitoRef: "ambito:prueba"})
	bolsas.declarar("bolsa:otra-unidad", bolsaports.AmbitoBolsa{UnidadRef: "unidad:ajena", AmbitoRef: "ambito:prueba"})
	bolsas.declarar("bolsa:otro-ambito", bolsaports.AmbitoBolsa{UnidadRef: "unidad:prueba", AmbitoRef: "ambito:ajeno"})
	p := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, fuenteRespuestas: fuente, vigentes: bolsas,
		generar: seguridadvec.GeneradorReferenciasCriptograficas{}, soporte: &soporteSesionBorradorBolsaDesarrollo{
			unidadRef: "unidad:prueba", ambitoRef: "ambito:prueba", soporteCanal: &soporteAltaContratacionTemporalDesarrollo{
				contexto: ctports.ContextoAutorizacionAltaV3{Vinculo: actor.Vinculo, Resultado: actor.Resultado}}}}
	return p, fuente, ctx
}

// F1-60 (revisión de seguridad): las lecturas de respuestas y de motivos de
// exclusión de un llamamiento usan la unidad y el ámbito de la bolsa, no los
// de la composición. Una bolsa de otra unidad o de otro ámbito se deniega
// aunque el actor tenga el permiso en la suya; una sin ámbito, sin consultar
// siquiera el permiso.
func TestLecturasDeLlamamientoUsanElAmbitoDeLaBolsa(t *testing.T) {
	const llamamiento = "llamamiento:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	p, fuente, ctx := preparadorLecturasAmbitoF160(t)
	respuestas := func(bolsa string) error {
		_, err := p.PrepararConsultaRespuestasPortalLlamamiento(ctx, bolsa, llamamiento)
		return err
	}
	exclusiones := func(bolsa string) error {
		_, err := p.PrepararConsultaExclusionesOfrecimientoRRHH(ctx, bolsa, llamamiento, 0)
		return err
	}
	if err := respuestas("bolsa:prueba"); err != nil {
		t.Fatalf("respuestas de la bolsa propia: %v", err)
	}
	if err := exclusiones("bolsa:prueba"); err != nil {
		t.Fatalf("exclusiones de la bolsa propia: %v", err)
	}
	for _, bolsa := range []string{"bolsa:otra-unidad", "bolsa:otro-ambito", "bolsa:sin-ambito"} {
		if err := respuestas(bolsa); !errors.Is(err, bolsaports.ErrRespuestasPortalRRHHNoDisponibles) {
			t.Fatalf("respuestas de %s no denegadas: %v", bolsa, err)
		}
		if err := exclusiones(bolsa); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
			t.Fatalf("exclusiones de %s no denegadas: %v", bolsa, err)
		}
	}
	antes := fuente.llamadas
	_ = respuestas("bolsa:sin-ambito")
	_ = exclusiones("bolsa:sin-ambito")
	if fuente.llamadas != antes {
		t.Fatal("una bolsa sin ámbito no debía llegar a consultar el permiso")
	}
}

// La lectura del orden de la bolsa que hace Contratación temporal toma la
// unidad de la bolsa: sin ámbito se deniega y, si la unidad del expediente no
// es la de la bolsa, no sigue.
func TestOrigenCTUsaElAmbitoDeLaBolsa(t *testing.T) {
	p, _, ctx := preparadorLecturasAmbitoF160(t)
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: p.soporte.soporteCanal}
	leer := nuevoConstructorOrdenOrigenCT(alta, &puenteBolsaLlamamientoDesarrollo{}, p, "proceso", "canal")
	d := preparacionLlamamientoDesarrollo{unidad: "unidad:prueba"}
	if _, err := leer(ctx, d, "bolsa:sin-ambito"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa sin ámbito: %v", err)
	}
	if _, err := leer(ctx, d, "bolsa:otra-unidad"); !errors.Is(err, bolsaports.ErrOrigenLlamamientoCTNoDisponible) ||
		errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("bolsa de otra unidad: %v", err)
	}
}
