package bootstrap

import (
	"reflect"
	"strings"
	"testing"

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

func TestLectorTecnicoConsumeContextoFijoAltaRegistrado(t *testing.T) {
	alta, fijo, lector := escenarioLectorTecnicoFijoPrueba(t)
	soporte, autorizador, err := nuevoSoporteLectorRRHHDesarrollo(alta, lector, relojContratacionTemporalDesarrollo{})
	if err != nil || soporte == nil || autorizador == nil {
		t.Fatalf("lector técnico fijo: %v", err)
	}
	if !reflect.DeepEqual(soporte.contexto, fijo.contexto) ||
		soporte.contexto.Resultado.Contexto.PerfilActivoRef != fijo.perfilRef() ||
		soporte.contexto.Resultado.Contexto.PerfilActivoRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
		!soporte.tecnicoConsultaRRHH || !soporte.lectorConsultasRRHH ||
		fijo.plantilla.VersionRol.RolID == "" || len(fijo.plantilla.VersionRol.Concesiones) == 0 {
		t.Fatal("se sustituyó el contexto Alta o se perdió el perfil y su alcance")
	}
}

func TestLectorTecnicoDeniegaPerfilFijoNoCoincidente(t *testing.T) {
	for nombre, alterar := range map[string]func(*dependenciasAltaContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, *identidadConsultaRRHHDesarrollo){
		"alta ausente": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *identidadConsultaRRHHDesarrollo) {
			filtrados := a.soporte.perfilesFijos[:0]
			for _, p := range a.soporte.perfilesFijos {
				if p.clave != clavePerfilFijoAltaCTDesarrollo {
					filtrados = append(filtrados, p)
				}
			}
			a.soporte.perfilesFijos = filtrados
		},
		"perfil genérico": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.perfilRef = a.soporte.contexto.Resultado.Contexto.PerfilActivoRef
		},
		"perfil ajeno": func(_ *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.perfilRef = "prf_" + strings.Repeat("f", 32)
		},
		"alta duplicada": func(a *dependenciasAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo, _ *identidadConsultaRRHHDesarrollo) {
			a.soporte.perfilesFijos = append(a.soporte.perfilesFijos, p)
		},
		"certificado ajeno": func(_ *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
		},
		"persona ajena": func(_ *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.ID = "tecnico_ajeno"
		},
		"rol vacío": func(_ *dependenciasAltaContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.Roles = nil
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alta, fijo, lector := escenarioLectorTecnicoFijoPrueba(t)
			alterar(alta, fijo, &lector)
			if soporte, autorizador, err := nuevoSoporteLectorRRHHDesarrollo(alta, lector, relojContratacionTemporalDesarrollo{}); err == nil || soporte != nil || autorizador != nil {
				t.Fatalf("aceptó lector técnico sin origen fijo exacto: %v", err)
			}
		})
	}
}
