package bootstrap

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// El pool se entrega solo como dependencia de composición. Este constructor
// no consulta PostgreSQL: la comprobación de asignación publicada sigue en
// la autoridad de lectura al atender cada petición.
func escenarioLectorTecnicoFijoPrueba(t *testing.T) (*dependenciasAltaContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, identidadConsultaRRHHDesarrollo) {
	t.Helper()
	soporte, _ := escenarioPerfilesFijosPrueba(t)
	var altaFija *perfilFijoCTDesarrollo
	for _, p := range soporte.perfilesFijosRegistrados() {
		if p != nil && p.clave == clavePerfilFijoAltaCTDesarrollo {
			altaFija = p
			break
		}
	}
	if altaFija == nil || altaFija.contexto.Resultado.Validar() != nil || altaFija.perfilRef() == "" {
		t.Fatal("perfil fijo de Alta no compuesto")
	}
	principal := principalDeSoportePrueba(soporte)
	lector := identidadConsultaRRHHDesarrollo{
		identidad:       identidadCertificadoDesarrollo{principal: principal},
		perfilRef:       altaFija.perfilRef(),
		organizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		clase:           ctports.AmbitoOrganizacionRRHH,
		ambitoRef:       organizacionAltaContratacionTemporalDesarrollo,
	}
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte,
		postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{gobierno: &pgxpool.Pool{}}}
	return alta, altaFija, lector
}

// La técnica consulta con el circuito del alta. El constructor de soportes
// de lectores no le abre otro, ni siquiera a la titular exacta del alta.
func TestSoporteLectorDeniegaALaTecnica(t *testing.T) {
	alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
	if !lectorTecnicoEsTitularAltaDesarrollo(alta, lector) {
		t.Fatal("el escenario no es la titular exacta del alta")
	}
	// Con el perfil fijo de Alta y con el que le daría el circuito de los
	// lectores ordinarios: en ningún caso se le abre un soporte propio.
	sintetico, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(lector.identidad.principal, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	datos, err := sintetico.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{lector.perfilRef, datos.PerfilActivoRef} {
		lector.perfilRef = perfil
		if soporte, autorizador, err := nuevoSoporteLectorRRHHDesarrollo(alta, lector, relojContratacionTemporalDesarrollo{}); err == nil || soporte != nil || autorizador != nil {
			t.Fatalf("abrió un soporte de lectora para la técnica con %s: %v", perfil, err)
		}
	}
}
