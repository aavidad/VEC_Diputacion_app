package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El preparador del portal de la copia (portal_copia/preparar_portal.py)
// escribe este mismo perfil en consultas-rrhh.json y en la cuenta de
// inscripción. Si la derivación cambia aquí, el arranque con inscripción
// vuelve a detenerse en inscripcion_rrhh_cuentas_lectores.
func TestManifiestoLectorTecnicoDeLaCopiaCargaYCompone(t *testing.T) {
	perfil := referenciaAltaContratacionTemporalDesarrollo("prf_", "per_vector_tecnico\x00"+strings.Repeat("a", 64)+
		"\x00"+discriminadorPerfilFijoCTDesarrollo(clavePerfilFijoAltaCTDesarrollo).perfil)
	if perfil != "prf_40c9ab032a57e021e5d044148feacb33" {
		t.Fatalf("la derivación del perfil fijo de Alta cambió: %s", perfil)
	}
}

type motivosConsultaRRHHPrueba struct{}

func (motivosConsultaRRHHPrueba) ResolverMotivoCuadroRRHH(context.Context, time.Time) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return dominiovec.ReferenciaEntradaCatalogo{}, nil
}

func (motivosConsultaRRHHPrueba) ResolverMotivoDetalleRRHH(context.Context, time.Time) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return dominiovec.ReferenciaEntradaCatalogo{}, nil
}

type cuadroRRHHPrueba struct{}

func (cuadroRRHHPrueba) Consultar(context.Context, ctports.SolicitudCuadroRRHH) (ctports.PaginaCuadroRRHH, error) {
	return ctports.PaginaCuadroRRHH{}, nil
}

type detalleRRHHPrueba struct{ nombre string }

func (detalleRRHHPrueba) Consultar(context.Context, ctports.SolicitudDetalleRRHH) (ctports.DetalleExpedienteRRHH, error) {
	return ctports.DetalleExpedienteRRHH{}, nil
}

// La técnica figura en el manifiesto pero consulta con el circuito del alta:
// no se abre otro soporte ni se publica nada sobre su contexto fijo.
func TestLectorTecnicoReutilizaElCircuitoDelAlta(t *testing.T) {
	alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
	cerrado := 0
	base := dependenciasConsultasRRHHDesarrollo{
		cuadroHTTP: cuadroRRHHPrueba{}, detalleHTTP: detalleRRHHPrueba{"detalle"},
		originalPropuestaHTTP: detalleRRHHPrueba{"original"}, cerrar: func() { cerrado++ },
	}
	deps, err := nuevasDependenciasLectoresRRHHDesarrollo(context.Background(), config.Config{}, alta, nil,
		relojContratacionTemporalDesarrollo{}, []identidadConsultaRRHHDesarrollo{lector}, &postgresct.PoolConsultasRRHHPostgreSQL{},
		motivosConsultaRRHHPrueba{}, dominiovec.ReferenciaEntradaCatalogo{}, dominiovec.ReferenciaEntradaCatalogo{}, base,
		catalogoFronterasComunDesarrollo{}, false)
	if err != nil {
		t.Fatalf("lector técnico: %v", err)
	}
	mux, ok := deps.cuadroHTTP.(*multiplexoresLectoresRRHHDesarrollo)
	id := lector.identidad.principal.ID
	if !ok || mux.soportes[id] != alta.soporte || mux.cuadros[id] != base.cuadroHTTP ||
		mux.detalles[id] != base.detalleHTTP || mux.originalesPropuesta[id] != base.originalPropuestaHTTP {
		t.Fatal("la técnica no usa el cuadro, el detalle y el soporte del alta")
	}
	if alta.soporte.lectorConsultasRRHH {
		t.Fatal("se alteró el soporte del alta")
	}
	deps.cerrar()
	if cerrado != 1 {
		t.Fatalf("cierre base ejecutado %d veces", cerrado)
	}
}

func TestLectorTecnicoDeniegaSiNoEsElTitularDelAlta(t *testing.T) {
	for nombre, alterar := range map[string]func(*dependenciasAltaContratacionTemporalDesarrollo, *identidadConsultaRRHHDesarrollo){
		"persona ajena": func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.ID = "tecnico_ajeno"
		},
		"certificado ajeno": func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
		},
		"ámbito de centro": func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.clase, l.ambitoRef = ctports.AmbitoCentroRRHH, "centro:rpt:520"
		},
		// Solo cambia la clase: el ámbito sigue siendo la organización.
		"clase de centro con ámbito de organización": func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.clase = ctports.AmbitoCentroRRHH
		},
		"organización ajena": func(_ *dependenciasAltaContratacionTemporalDesarrollo, l *identidadConsultaRRHHDesarrollo) {
			l.organizacionRef, l.ambitoRef = "organizacion:otra", "organizacion:otra"
		},
		"alta que ya es lectora": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *identidadConsultaRRHHDesarrollo) {
			a.soporte.lectorConsultasRRHH = true
		},
		"sin alta": func(a *dependenciasAltaContratacionTemporalDesarrollo, _ *identidadConsultaRRHHDesarrollo) {
			a.soporte = nil
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
			alterar(alta, &lector)
			if lectorTecnicoEsTitularAltaDesarrollo(alta, lector) {
				t.Fatal("admitió una técnica distinta de la titular del alta")
			}
		})
	}
	alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
	if !lectorTecnicoEsTitularAltaDesarrollo(alta, lector) {
		t.Fatal("rechazó a la titular exacta del alta")
	}
	lector.identidad.principal.Roles = []string{"lector_rrhh"}
	if lectorTecnicoRRHHDesarrollo(lector) || lectorTecnicoEsTitularAltaDesarrollo(alta, lector) {
		t.Fatal("un lector ordinario no usa el circuito del alta")
	}
}

// Una técnica que no es la titular del alta detiene el arranque de las
// consultas: no se registra en el multiplexor ni se le abre otro soporte.
func TestArranqueLectoresDeniegaUnaTecnicaAjena(t *testing.T) {
	for nombre, alterar := range map[string]func(*identidadConsultaRRHHDesarrollo){
		"persona ajena": func(l *identidadConsultaRRHHDesarrollo) { l.identidad.principal.ID = "tecnico_ajeno" },
		"certificado ajeno": func(l *identidadConsultaRRHHDesarrollo) {
			l.identidad.principal.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alta, _, lector := escenarioLectorTecnicoFijoPrueba(t)
			alterar(&lector)
			cerrado := 0
			base := dependenciasConsultasRRHHDesarrollo{
				cuadroHTTP: cuadroRRHHPrueba{}, detalleHTTP: detalleRRHHPrueba{"detalle"},
				originalPropuestaHTTP: detalleRRHHPrueba{"original"}, cerrar: func() { cerrado++ },
			}
			deps, err := nuevasDependenciasLectoresRRHHDesarrollo(context.Background(), config.Config{}, alta, nil,
				relojContratacionTemporalDesarrollo{}, []identidadConsultaRRHHDesarrollo{lector}, &postgresct.PoolConsultasRRHHPostgreSQL{},
				motivosConsultaRRHHPrueba{}, dominiovec.ReferenciaEntradaCatalogo{}, dominiovec.ReferenciaEntradaCatalogo{}, base,
				catalogoFronterasComunDesarrollo{}, false)
			if !errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) || deps.cuadroHTTP != nil || deps.cerrar != nil {
				t.Fatalf("arrancó con una técnica ajena: %v", err)
			}
			if cerrado != 1 || alta.soporte.lectorConsultasRRHH {
				t.Fatalf("el arranque denegado no cerró la base (%d) o alteró el alta", cerrado)
			}
		})
	}
}

func TestFronterasDeConsultaNoAdmitenElPerfilFijoDeLaTecnica(t *testing.T) {
	_, fijo, tecnica := escenarioLectorTecnicoFijoPrueba(t)
	ordinario := tecnica
	ordinario.identidad.principal.Roles = []string{"lector_rrhh"}
	ordinario.perfilRef = "prf_" + strings.Repeat("b", 32)
	perfiles := perfilesLectoresConsultaRRHHDesarrollo([]identidadConsultaRRHHDesarrollo{tecnica, ordinario})
	if len(perfiles) != 1 || perfiles[0] != ordinario.perfilRef || perfiles[0] == fijo.perfilRef() {
		t.Fatalf("perfiles de consulta: %v", perfiles)
	}
}
