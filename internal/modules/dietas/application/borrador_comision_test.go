package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type unidadBorradorPrueba struct {
	crear  int
	ultima ports.SolicitudCrearBorradorPropio
	pagina ports.PaginaBorradoresPropios
}

func (u *unidadBorradorPrueba) ListarBorradoresPropios(context.Context, vecdomain.ContextoActor, ports.ConsultaBorradoresPropios) (ports.PaginaBorradoresPropios, error) {
	return u.pagina, nil
}

type politicaBorradorPrueba struct {
	falla  bool
	ultima ports.SolicitudPoliticaKilometraje
}

func (p *politicaBorradorPrueba) ResolverPoliticaKilometraje(_ context.Context, s ports.SolicitudPoliticaKilometraje) (domain.PoliticaKilometraje, error) {
	p.ultima = s
	if p.falla {
		return domain.PoliticaKilometraje{}, errors.New("sin política")
	}
	return domain.PoliticaKilometraje{Referencia: "pol_sintetica", Version: "v1", TarifaEURPorKM: "0.5000"}, nil
}

func (u *unidadBorradorPrueba) CrearBorradorPropio(_ context.Context, x ports.SolicitudCrearBorradorPropio) (ports.ReciboBorradorComision, error) {
	u.crear++
	u.ultima = x
	return ports.ReciboBorradorComision{ComisionRef: "com_001", ReciboRef: "rec_001", CorrelacionRef: "cor_001", Version: 1, RegistradoEn: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)}, nil
}
func (u *unidadBorradorPrueba) RecuperarBorradorPropio(context.Context, vecdomain.ContextoActor, string) (domain.BorradorComision, ports.ReciboBorradorComision, error) {
	return domain.BorradorComision{}, ports.ReciboBorradorComision{}, errors.New("no usado")
}
func actorBorradorPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	ahora := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	a, e := vecdomain.NuevoContextoActor(cuenta, vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}, ahora)
	if e != nil {
		t.Fatal(e)
	}
	a.Instantanea.Vinculos = []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_0123456789abcdefghijkl", Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_0123456789abcdefghijkl", Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}
	return a
}
func borradorAplicacionPrueba(persona string) domain.BorradorComision {
	return domain.BorradorComision{PersonaRef: persona, Objeto: "Visita", FechaInicio: "2026-01-02", FechaFin: "2026-01-02", HoraInicio: "09:00", HoraFin: "11:00", VehiculoPropio: true, Gastos: domain.GastosComision{ManutencionEUR: "10.00", AlojamientoEUR: "20.00", OtrosEUR: "5.00"}, Ruta: domain.RutaVersionada{Fuente: "osrm_interno", Version: "grafo-v1", Referencia: "RUTA-01", CatalogoVersion: "catalogo-v1", AlternativaRef: "ALT-01", Recomendada: true, Kilometros: "10", Paradas: []domain.ParadaRuta{{Codigo: "A1", Nombre: "Origen", Latitud: 37, Longitud: -3}, {Codigo: "B1", Nombre: "Destino", Latitud: 38, Longitud: -3}}, Tramos: []domain.TramoRuta{{OrigenCodigo: "A1", DestinoCodigo: "B1", Kilometros: "10", DuracionMinutos: 15, AjusteKilometros: "0"}}, Trazado: [][2]float64{{37, -3}, {38, -3}}}}
}
func TestCrearPropioEntregaUnicaSolicitudAUnidadTrabajo(t *testing.T) {
	u := &unidadBorradorPrueba{}
	s, e := NuevoServicioBorradorComision(u, &politicaBorradorPrueba{}, time.UTC)
	if e != nil {
		t.Fatal(e)
	}
	a := actorBorradorPrueba(t)
	r, e := s.CrearPropio(context.Background(), ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "abcdefghijklmnop", Borrador: borradorAplicacionPrueba(a.PersonaRef)})
	if e != nil {
		t.Fatal(e)
	}
	if u.crear != 1 || u.ultima.ClaveOperacion != "abcdefghijklmnop" || r.ReciboRef != "rec_001" {
		t.Fatal("no cruzó una única frontera transaccional")
	}
}
func TestCrearPropioRechazaBorradorDeOtraPersonaAntesDeUnidad(t *testing.T) {
	u := &unidadBorradorPrueba{}
	s, _ := NuevoServicioBorradorComision(u, &politicaBorradorPrueba{}, time.UTC)
	a := actorBorradorPrueba(t)
	_, e := s.CrearPropio(context.Background(), ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "abcdefghijklmnop", Borrador: borradorAplicacionPrueba("per_otra")})
	if !errors.Is(e, domain.ErrBorradorComisionInvalido) || u.crear != 0 {
		t.Fatal("titularidad no cerrada")
	}
}

func TestCrearPropioExigeVinculoEmpleadoCanonico(t *testing.T) {
	u := &unidadBorradorPrueba{}
	s, _ := NuevoServicioBorradorComision(u, &politicaBorradorPrueba{}, time.UTC)
	a := actorBorradorPrueba(t)
	a.Instantanea.Vinculos = nil
	_, err := s.CrearPropio(context.Background(), ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "abcdefghijklmnop", Borrador: borradorAplicacionPrueba(a.PersonaRef)})
	if !errors.Is(err, ports.ErrAccesoBorradorDenegado) || u.crear != 0 {
		t.Fatal("se aceptó una persona sin vínculo empleado")
	}
}

func TestCrearPropioRecalculaConPoliticaDeServidor(t *testing.T) {
	u := &unidadBorradorPrueba{}
	p := &politicaBorradorPrueba{}
	s, _ := NuevoServicioBorradorComision(u, p, time.UTC)
	a := actorBorradorPrueba(t)
	b := borradorAplicacionPrueba(a.PersonaRef)
	b.Desglose.TotalEUR = "0.00"
	_, e := s.CrearPropio(context.Background(), ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "abcdefghijklmnop", Borrador: b, PoliticaReferencia: "pol_sintetica", PoliticaVersion: "v1"})
	if e != nil || u.ultima.Borrador.Desglose.TotalEUR != "40.00" || u.ultima.Borrador.Validar() != nil || p.ultima.ContextoActor.PersonaRef != a.PersonaRef {
		t.Fatal("cálculo o política no gobernados", e)
	}
}
func TestCrearPropioNoInventaPoliticaNiZona(t *testing.T) {
	a := actorBorradorPrueba(t)
	for _, c := range []struct {
		p ports.ProveedorPoliticaKilometrajeBorrador
		z *time.Location
	}{{nil, time.UTC}, {&politicaBorradorPrueba{}, nil}, {&politicaBorradorPrueba{falla: true}, time.UTC}} {
		u := &unidadBorradorPrueba{}
		s, _ := NuevoServicioBorradorComision(u, c.p, c.z)
		_, e := s.CrearPropio(context.Background(), ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "abcdefghijklmnop", Borrador: borradorAplicacionPrueba(a.PersonaRef)})
		if !errors.Is(e, ports.ErrPoliticaBorradorNoDisponible) || u.crear != 0 {
			t.Fatal("dependencia ausente permitió efecto", e)
		}
	}
}
func TestListaValidaTitularOrdenCursorYLimite(t *testing.T) {
	a := actorBorradorPrueba(t)
	b, e := domain.PrepararBorrador(borradorAplicacionPrueba(a.PersonaRef), domain.PoliticaKilometraje{Referencia: "pol_sintetica", Version: "v1", TarifaEURPorKM: "0.5000"}, time.UTC)
	if e != nil {
		t.Fatal(e)
	}
	item := ports.BorradorConRecibo{Borrador: b.Resumen(), Recibo: ports.ReciboBorradorComision{ComisionRef: "dietas:borrador:abcdefghijklmnop", ReciboRef: "rec_001", CorrelacionRef: "cor_001", Version: 1, RegistradoEn: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)}}
	for nombre, mutar := range map[string]func(*ports.PaginaBorradoresPropios){"ajeno": func(p *ports.PaginaBorradoresPropios) { p.Borradores[0].Borrador.PersonaRef = "per_otra" }, "cursor": func(p *ports.PaginaBorradoresPropios) { p.Siguiente = "dietas:borrador:otrareferenciakey" }, "duplicado": func(p *ports.PaginaBorradoresPropios) { p.Borradores = append(p.Borradores, p.Borradores[0]) }} {
		t.Run(nombre, func(t *testing.T) {
			p := ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{item}}
			mutar(&p)
			u := &unidadBorradorPrueba{pagina: p}
			s, _ := NuevoServicioBorradorComision(u, nil, nil)
			if _, e := s.ListarPropios(context.Background(), a, ports.ConsultaBorradoresPropios{Limite: 20}); !errors.Is(e, ErrComposicionBorradorInvalida) {
				t.Fatal("lista manipulada aceptada", e)
			}
		})
	}
	u := &unidadBorradorPrueba{pagina: ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{item}}}
	s, _ := NuevoServicioBorradorComision(u, nil, nil)
	if _, e := s.ListarPropios(context.Background(), a, ports.ConsultaBorradoresPropios{Limite: 20}); e != nil {
		t.Fatal("consulta dependió de política para crear", e)
	}
}
