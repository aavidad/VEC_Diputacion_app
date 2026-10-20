package domain

import "testing"

// La v1 publicada empieza por participaciones_propias.consultar: AUT104 compara
// las siete concesiones base como conjunto exacto y el dominio debe coincidir.
func TestVersionarRolBolsaExternaBasePublicadaEnOtroOrden(t *testing.T) {
	c, base, i, ahora := escenarioVersionarRolBolsaExterna(t)
	base.Rol.Concesiones[0], base.Rol.Concesiones[1] = base.Rol.Concesiones[1], base.Rol.Concesiones[0]
	if base.Rol.Concesiones[0].Accion != "bolsa.participaciones_propias.consultar" {
		t.Fatal("el escenario no reproduce el orden publicado")
	}
	refrescarBaseExterna(t, &base, &i)
	if estadoRolBolsaExterna(base.Rol) != "base" {
		t.Fatal("la v1 publicada no se reconoce como base")
	}
	if _, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora); err != nil {
		t.Fatal("el plan desde la v1 publicada se rechaza:", err)
	}
	// El conjunto sigue siendo exacto: una acción repetida deja de ser base
	// aunque conserve las siete posiciones.
	repetida := base.Rol
	repetida.Concesiones = append([]ConcesionRol(nil), base.Rol.Concesiones...)
	repetida.Concesiones[1] = repetida.Concesiones[0]
	if estadoRolBolsaExterna(repetida) != "" {
		t.Fatal("una base con acción repetida se acepta")
	}
}
