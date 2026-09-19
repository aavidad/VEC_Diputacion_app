package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func actorPrueba(t *testing.T) core.ContextoActor {
	t.Helper()
	ahora := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	ref := func(p string) string { return p + strings.Repeat("A", 22) }
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: ref("cta_"), Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	actor, e := core.NuevoContextoActor(cuenta, core.InstantaneaContextoActor{
		VinculoRef: ref("vca_"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1,
		PersonaRef: ref("per_"), PersonaVersion: 1, PerfilActivoRef: ref("prf_"), PerfilVersion: 1,
		Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []core.VinculoReferenciaContextoActor{{VinculoRef: ref("vin_"), Version: 1, Tipo: core.TipoReferenciaContextoActorEmpleado, Referencia: ref("emp_"), Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}, ahora)
	if e != nil {
		t.Fatal(e)
	}
	return actor
}
func solicitudPrueba(t *testing.T) ports.SolicitudCrearBorradorPropio {
	a := actorPrueba(t)
	b, err := domain.PrepararBorrador(domain.BorradorComision{
		PersonaRef: a.PersonaRef, Objeto: "Visita sintética", FechaInicio: "2026-09-19", FechaFin: "2026-09-19", HoraInicio: "10:00", HoraFin: "11:00", VehiculoPropio: true,
		Gastos: domain.GastosComision{ManutencionEUR: "12.00", AlojamientoEUR: "0.00", OtrosEUR: "2.00"},
		Ruta: domain.RutaVersionada{Fuente: "osrm_interno", Version: "grafo:prueba:1", Referencia: "ruta:prueba", CatalogoVersion: "catalogo:1", AlternativaRef: "ruta:prueba", Recomendada: true, Kilometros: "10",
			Paradas: []domain.ParadaRuta{{Codigo: "origen", Nombre: "Origen", Latitud: 37, Longitud: -3}, {Codigo: "destino", Nombre: "Destino", Latitud: 38, Longitud: -3}},
			Tramos:  []domain.TramoRuta{{OrigenCodigo: "origen", DestinoCodigo: "destino", Kilometros: "10", DuracionMinutos: 20, AjusteKilometros: "0"}}, Trazado: [][2]float64{{37, -3}, {38, -3}}},
	}, domain.PoliticaKilometraje{Referencia: "politica:prueba", Version: "1", TarifaEURPorKM: "0.2600"}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudCrearBorradorPropio{ContextoActor: a, ClaveOperacion: "Operacion_AAAAAAA", Borrador: b}
}
func TestMaterialBorradorLigaEmpleadoYMaterialCanonico(t *testing.T) {
	x := solicitudPrueba(t)
	b, r, e := materialCreacion(x)
	if e != nil {
		t.Fatal(e)
	}
	if r.ModuloID != "dietas" || r.Tipo != "borrador_comision" || r.Ambitos["persona_ref"] != x.ContextoActor.PersonaRef || r.Ambitos["empleado_ref"] != x.ContextoActor.Instantanea.Vinculos[0].Referencia {
		t.Fatal("recurso incompleto")
	}
	x.Borrador = x.Borrador.Clonar()
	b2, r2, e := materialCreacion(x)
	if e != nil || string(b) != string(b2) || r.Atributos["material_sha256"] != r2.Atributos["material_sha256"] {
		t.Fatal("clon y original divergentes")
	}
	x.Borrador.Objeto = "Otra visita"
	_, r3, e := materialCreacion(x)
	if e != nil || r3.Atributos["material_sha256"] == r.Atributos["material_sha256"] {
		t.Fatal("material nuevo no ligado")
	}
	x.ContextoActor.Instantanea.Vinculos = nil
	if _, _, e = materialCreacion(x); !errors.Is(e, ports.ErrAccesoBorradorDenegado) {
		t.Fatal("sin empleado")
	}
}

type poolObservado struct{ llamadas int }

func (p *poolObservado) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	return nil, errors.New("no debe consultar")
}

type autorizadorVacio struct{ llamadas int }

func (p *autorizadorVacio) AutorizarBorradorPropio(context.Context, core.ContextoActor, string, core.RecursoAutorizable) (AutorizacionBorrador, error) {
	p.llamadas++
	return AutorizacionBorrador{}, nil
}
func TestMaterialNoAtestadoNoAbreTransaccion(t *testing.T) {
	p := &poolObservado{}
	a := &autorizadorVacio{}
	r := &RepositorioBorradorComision{pool: p, proveedor: a}
	if _, e := r.CrearBorradorPropio(context.Background(), solicitudPrueba(t)); !errors.Is(e, ports.ErrBorradorNoDisponible) {
		t.Fatal("material vacío aceptado")
	}
	if a.llamadas != 1 || p.llamadas != 0 {
		t.Fatal("se alcanzó SQL sin material atestado")
	}
}
func TestMaterialAjenoNoSolicitaConcesion(t *testing.T) {
	p := &poolObservado{}
	a := &autorizadorVacio{}
	r := &RepositorioBorradorComision{pool: p, proveedor: a}
	x := solicitudPrueba(t)
	x.Borrador.PersonaRef = "per:otra"
	if _, e := r.CrearBorradorPropio(context.Background(), x); !errors.Is(e, domain.ErrBorradorComisionInvalido) {
		t.Fatal("persona cruzada")
	}
	if a.llamadas != 0 || p.llamadas != 0 {
		t.Fatal("se autorizó material ajeno")
	}
}
func TestReciboYJSONCerrados(t *testing.T) {
	r := ports.ReciboBorradorComision{ComisionRef: "dietas:borrador:Operacion_AAAAAAA", ReciboRef: "recibo:prueba", CorrelacionRef: "corr:prueba", Version: 1, RegistradoEn: time.Now().UTC().Truncate(time.Microsecond)}
	if validarRecibo(r, r.ComisionRef) != nil {
		t.Fatal("recibo válido")
	}
	if validarRecibo(r, "dietas:borrador:otro") == nil {
		t.Fatal("recibo cruzado")
	}
	var salida ports.ReciboBorradorComision
	for _, b := range []string{`{"version":1,"extra":true}`, `{} {}`} {
		if decodificar([]byte(b), &salida) == nil {
			t.Fatal("salida abierta")
		}
	}
}

func TestListadoBorradorLigaPaginacionYSujeto(t *testing.T) {
	a := actorPrueba(t)
	q := ports.ConsultaBorradoresPropios{Limite: 20}
	b, recurso, err := materialListado(a, q)
	if err != nil || recurso.Referencia != "dietas:borradores:propios" || !strings.Contains(string(b), `"limite":20,"despues":""`) {
		t.Fatal("listado sin ligadura exacta")
	}
	q.Despues = "dietas:borrador:Operacion_AAAAAAA"
	_, otra, err := materialListado(a, q)
	if err != nil || otra.Atributos["material_sha256"] == recurso.Atributos["material_sha256"] {
		t.Fatal("cursor no ligado")
	}
	for _, invalida := range []ports.ConsultaBorradoresPropios{{Limite: 0}, {Limite: 21}, {Limite: 1, Despues: "valor no canónico"}} {
		if _, _, err := materialListado(a, invalida); err == nil {
			t.Fatal("paginación abierta")
		}
	}
}

func TestListadoBorradorRechazaAjenoDesordenadoYCursorFalso(t *testing.T) {
	x := solicitudPrueba(t)
	uno := ports.BorradorConRecibo{Borrador: x.Borrador.Resumen(), Recibo: ports.ReciboBorradorComision{ComisionRef: "dietas:borrador:Operacion_AAAAAAA", ReciboRef: "recibo:uno", CorrelacionRef: "corr:uno", Version: 1, RegistradoEn: x.Borrador.Inicio}}
	dos := uno
	dos.Recibo.ComisionRef = "dietas:borrador:Operacion_BBBBBBB"
	q := ports.ConsultaBorradoresPropios{Limite: 2}
	valida := ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{uno, dos}, Siguiente: dos.Recibo.ComisionRef}
	if validarPagina(valida, x.ContextoActor.PersonaRef, q) != nil {
		t.Fatal("página válida rechazada")
	}
	if validarPagina(ports.PaginaBorradoresPropios{Borradores: []ports.BorradorConRecibo{}}, x.ContextoActor.PersonaRef, q) != nil {
		t.Fatal("vacío acreditado rechazado")
	}
	ajeno := uno
	ajeno.Borrador.PersonaRef = "per_" + strings.Repeat("B", 22)
	for _, invalida := range []ports.PaginaBorradoresPropios{
		{},
		{Borradores: []ports.BorradorConRecibo{ajeno}},
		{Borradores: []ports.BorradorConRecibo{dos, uno}},
		{Borradores: []ports.BorradorConRecibo{uno, uno}},
		{Borradores: []ports.BorradorConRecibo{uno, dos}, Siguiente: uno.Recibo.ComisionRef},
		{Borradores: []ports.BorradorConRecibo{uno}, Siguiente: uno.Recibo.ComisionRef},
		{Borradores: []ports.BorradorConRecibo{uno, dos, dos}},
	} {
		if validarPagina(invalida, x.ContextoActor.PersonaRef, q) == nil {
			t.Fatal("página inválida aceptada")
		}
	}
	q.Despues = uno.Recibo.ComisionRef
	if validarPagina(valida, x.ContextoActor.PersonaRef, q) == nil {
		t.Fatal("cursor no avanza")
	}
}

func TestRepositorioBorradorSinProveedorNoAutoriza(t *testing.T) {
	p := &poolObservado{}
	r := &RepositorioBorradorComision{pool: p}
	if _, err := r.CrearBorradorPropio(context.Background(), solicitudPrueba(t)); !errors.Is(err, ports.ErrBorradorNoDisponible) || p.llamadas != 0 {
		t.Fatal("operación sin autoridad nominal")
	}
}
