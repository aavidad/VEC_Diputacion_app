package bootstrap

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestVinculoBolsaEsperaProvisionSinRomperAlta(t *testing.T) {
	s, fuente := escenarioPerfilesFijosPrueba(t)
	p := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	if p == nil {
		t.Fatal("perfil de alta ausente")
	}
	anterior := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla)
	nueva, err := plantillaAltaConVinculoBolsaCT(anterior)
	if err != nil || anterior.VersionRol.Version != 1 || len(anterior.VersionRol.Concesiones) != 1 ||
		nueva.VersionRol.Version != 2 || len(nueva.VersionRol.Concesiones) != 2 ||
		nueva.VersionRol.Concesiones[1].Accion != ports.AccionVincularEmisionBolsa {
		t.Fatalf("ampliación alteró rol publicado: %v", err)
	}
	p.plantillaVinculoBolsa = &nueva
	p.rutas[httpinterno.RutaVinculosEmisionBolsa] = struct{}{}
	fuente.asignaciones = map[string]instantaneaPublicadaDesarrollo{
		p.perfilRef(): {instantanea: anterior, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo},
	}
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
