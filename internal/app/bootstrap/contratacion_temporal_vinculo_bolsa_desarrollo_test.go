package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestVinculoBolsaEsperaProvisionSinRomperAlta(t *testing.T) {
	s, fuente := escenarioPerfilesFijosPrueba(t)
	p := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	if p == nil {
		t.Fatal("perfil de alta ausente")
	}
	anterior := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
	fuente.asignaciones = map[string]instantaneaPublicadaDesarrollo{
		p.perfilRef(): {instantanea: anterior, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo},
	}
	if actual, ok := s.instantaneaPerfilFijoParaContexto(context.Background(), httpinterno.RutaAltaSolicitudes, p); !ok ||
		actual.VersionRol.Version != 1 {
		t.Fatal("sin CT201 el alta antigua perdió su rol v1")
	}
	nueva, err := plantillaAltaConVinculoBolsaCT(anterior, nil)
	if err != nil || anterior.VersionRol.Version != 1 || len(anterior.VersionRol.Concesiones) != 1 ||
		nueva.VersionRol.Version != 2 || len(nueva.VersionRol.Concesiones) != 2 ||
		nueva.VersionRol.Concesiones[1].Accion != ports.AccionVincularEmisionBolsa {
		t.Fatalf("ampliación alteró rol publicado: %v", err)
	}
	p.plantillaVinculoBolsa = &nueva
	p.rutas[httpinterno.RutaVinculosEmisionBolsa] = struct{}{}
	if actual, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, httpinterno.RutaAltaSolicitudes); !ok ||
		actual.VersionRol.Version != 1 {
		t.Fatal("el alta dejó de consumir su rol v1 mientras vínculo espera CAS")
	}
	if _, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, httpinterno.RutaVinculosEmisionBolsa); ok {
		t.Fatal("el vínculo consumió el rol v1 de alta")
	}
	provisionada := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(nueva)
	provisionada.AsignacionPerfil.Version = anterior.AsignacionPerfil.Version + 1
	if provisionada.Validar() != nil {
		t.Fatal("plantilla v2 no válida")
	}
	fuente.asignaciones[p.perfilRef()] = instantaneaPublicadaDesarrollo{
		instantanea: provisionada, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	for _, ruta := range []string{httpinterno.RutaAltaSolicitudes, httpinterno.RutaVinculosEmisionBolsa} {
		actual, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, ruta)
		if !ok || actual.VersionRol.Version != 2 {
			t.Fatalf("rol v2 provisionado no sirve %s", ruta)
		}
	}
	provisionada.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	fuente.asignaciones[p.perfilRef()] = instantaneaPublicadaDesarrollo{
		instantanea: provisionada, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	if _, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, httpinterno.RutaVinculosEmisionBolsa); ok {
		t.Fatal("vínculo admitió asignación revocada")
	}
}

// Fallo de cidonia (10/10): el expediente nacido de una petición del centro
// guarda «centro-520» y la asignación de alta solo cubría «centro:rpt:520»;
// el PDP denegaba y la frontera lo contaba como 503.
func TestVinculoBolsaCubreCentroDePeticionYConservaAsignacionPrevia(t *testing.T) {
	s, fuente := escenarioPerfilesFijosPrueba(t)
	p := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	centros := s.origen.centrosOrganizacionPeticion()
	if p == nil || len(centros) == 0 {
		t.Fatal("escenario sin perfil de alta o sin centros de petición")
	}
	anterior := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
	previa, errPrevia := plantillaAltaConVinculoBolsaCT(anterior, nil)
	nueva, err := plantillaAltaConVinculoBolsaCT(anterior, centros)
	if errPrevia != nil || err != nil {
		t.Fatal("no se pudo ampliar la plantilla de alta")
	}
	recurso := func(centro string) core.RecursoAutorizable {
		return core.RecursoAutorizable{Referencia: "expediente:ct:prueba", ModuloID: ports.ModuloContratacion,
			Tipo: ports.TipoRecursoVinculoEmisionBolsa,
			Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"centro_ref": centro, "categoria_ref": categoriaAltaContratacionTemporalDesarrollo},
			Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}
	}
	if previa.AsignacionPerfil.Cubre(recurso(centros[0])) {
		t.Fatal("la prueba no reproduce el fallo: la asignación previa ya cubría el centro de petición")
	}
	if !nueva.AsignacionPerfil.Cubre(recurso(centros[0])) || !nueva.AsignacionPerfil.Cubre(recurso(centroAltaContratacionTemporalDesarrollo)) {
		t.Fatal("la asignación ampliada no cubre a la vez el centro de petición y el del alta")
	}
	if nueva.AsignacionPerfil.Cubre(recurso("centro:ajeno:999")) {
		t.Fatal("la ampliación abrió un centro fuera del catálogo")
	}
	if len(anterior.AsignacionPerfil.Ambitos[1].Valores) != len(p.plantilla.AsignacionPerfil.Ambitos[1].Valores) {
		t.Fatal("la ampliación alteró la plantilla de alta original")
	}
	p.plantillaVinculoBolsa, p.plantillaVinculoBolsaPrevia = &nueva, &previa
	p.rutas[httpinterno.RutaVinculosEmisionBolsa] = struct{}{}
	publicar := func(i core.InstantaneaAutorizacion) {
		i = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(i)
		i.AsignacionPerfil.Version = anterior.AsignacionPerfil.Version + 1
		fuente.asignaciones[p.perfilRef()] = instantaneaPublicadaDesarrollo{
			instantanea: i, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	}
	fuente.asignaciones = map[string]instantaneaPublicadaDesarrollo{}
	// Publicada la v2 aprobada ayer (sin centros de petición): alta y vínculo
	// siguen funcionando con sus permisos de antes mientras llega el CAS.
	publicar(previa)
	for _, ruta := range []string{httpinterno.RutaAltaSolicitudes, httpinterno.RutaVinculosEmisionBolsa} {
		actual, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, ruta)
		if !ok || actual.AsignacionPerfil.Cubre(recurso(centros[0])) {
			t.Fatalf("la asignación previa dejó de servir o cubre de más en %s", ruta)
		}
	}
	// Tras el CAS, el vínculo cubre el centro guardado por la petición.
	publicar(nueva)
	actual, ok := s.consumirPerfilAltaConVinculoBolsa(context.Background(), p, httpinterno.RutaVinculosEmisionBolsa)
	if !ok || !actual.AsignacionPerfil.Cubre(recurso(centros[0])) {
		t.Fatal("la asignación ampliada no se consume para el vínculo")
	}
}

func TestVinculoBolsaDenegacionV3NoSaleComoIndisponible(t *testing.T) {
	ctx := context.Background()
	if err := errorAutorizacionVinculoEmisionBolsa(ctx, core.ErrAutorizacionDenegada); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("una denegación del PDP debe responder 403")
	}
	for _, causa := range []error{vecports.ErrFuenteAutorizacionNoDisponible,
		vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible} {
		if err := errorAutorizacionVinculoEmisionBolsa(ctx, errors.Join(core.ErrAutorizacionDenegada, causa)); !errors.Is(err, ports.ErrVinculoEmisionBolsaNoDisponible) {
			t.Fatalf("una fuente caída debe responder 503: %v", causa)
		}
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := errorAutorizacionVinculoEmisionBolsa(cancelado, core.ErrAutorizacionDenegada); !errors.Is(err, context.Canceled) {
		t.Fatal("la cancelación debe conservarse")
	}
}
