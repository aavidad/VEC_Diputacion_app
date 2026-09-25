package reglasvec

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaReglasBolsa = "../../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

// calculadoraPrueba suma meses de fecha a fecha, sin calendario: basta para
// comprobar que el adaptador pide el plazo correcto de cada regla.
type calculadoraPrueba struct {
	pedidas []reglas.SolicitudVencimiento
	err     error
}

func (c *calculadoraPrueba) CalcularVencimiento(_ context.Context, s reglas.SolicitudVencimiento) (reglas.Vencimiento, error) {
	c.pedidas = append(c.pedidas, s)
	if c.err != nil {
		return reglas.Vencimiento{}, c.err
	}
	fin := s.Inicio.AddDate(0, s.Cantidad, 0)
	return reglas.Vencimiento{UltimoDia: fin.Format(time.DateOnly), VenceAntesDe: fin.Add(24 * time.Hour)}, nil
}

func catalogoPrueba(t *testing.T, calculadora reglas.CalculadoraPlazos) *CatalogoSanciones {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasBolsa)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoBolsa, ModuloID: reglas.ModuloBolsa,
		Reloj: relojFijo(time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)), Calculadora: calculadora,
	})
	if err != nil {
		t.Fatal(err)
	}
	return NuevoCatalogoSanciones(resolutor)
}

func TestConsecuenciasDelPaqueteDeEjemplo(t *testing.T) {
	catalogo := catalogoPrueba(t, nil)
	consecuencias, err := catalogo.Consecuencias(t.Context())
	if err != nil || len(consecuencias) != 6 {
		t.Fatalf("consecuencias=%d err=%v", len(consecuencias), err)
	}
	efectos := map[string]int{}
	for _, c := range consecuencias {
		efectos[c.Efecto]++
		if c.ReglaRef == "" || len(c.Huella) != 64 {
			t.Fatalf("consecuencia sin referencia: %+v", c)
		}
	}
	if efectos[dominiobolsa.OperacionExcluir] != 4 || efectos[dominiobolsa.OperacionPausar] != 1 || efectos[dominiobolsa.EfectoSancionNinguno] != 1 {
		t.Fatalf("efectos inesperados: %v", efectos)
	}
	estados, err := catalogo.EstadosRecurso(t.Context())
	if err != nil || len(estados) != 5 || estados[0] != "interpuesto" {
		t.Fatalf("estados=%v err=%v", estados, err)
	}
}

func TestResolverSuspensionYRecurso(t *testing.T) {
	calculadora := &calculadoraPrueba{}
	catalogo := catalogoPrueba(t, calculadora)
	notificada := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	res, err := catalogo.ResolverSancion(t.Context(), reglas.BolsaPrefijoSanciones+"suspension", notificada)
	if err != nil || res.SuspensionHasta != "2027-03-20" || res.Recurso.UltimoDia != "2026-10-20" ||
		res.Recurso.ReglaRef != "vec.bolsa.reglas:1:b24.consecuencias" || !res.Consecuencia.Ejemplo {
		t.Fatalf("resolución=%+v err=%v", res, err)
	}
	if len(calculadora.pedidas) != 2 || calculadora.pedidas[0].Computo != reglas.ComputoCivil || calculadora.pedidas[1].Computo != reglas.ComputoAdministrativo {
		t.Fatalf("plazos pedidos: %+v", calculadora.pedidas)
	}
	baja, err := catalogo.ResolverSancion(t.Context(), reglas.BolsaPrefijoSanciones+"baja_sin_contacto", notificada)
	if err != nil || baja.SuspensionHasta != "" || baja.Consecuencia.Efecto != dominiobolsa.OperacionExcluir || baja.Consecuencia.Articulo != "art. 11.1.a" {
		t.Fatalf("baja=%+v err=%v", baja, err)
	}
}

func TestResolverRechazaClaveAjenaYFallaCerradoSinCalendario(t *testing.T) {
	catalogo := catalogoPrueba(t, &calculadoraPrueba{err: errors.New("sin calendario")})
	notificada := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	if _, err := catalogo.ResolverSancion(t.Context(), reglas.BolsaPlazoRespuesta, notificada); !errors.Is(err, dominiobolsa.ErrSancionParticipacionInvalida) {
		t.Fatalf("clave ajena: %v", err)
	}
	if _, err := catalogo.ResolverSancion(t.Context(), reglas.BolsaPrefijoSanciones+"inexistente", notificada); !errors.Is(err, dominiobolsa.ErrSancionParticipacionInvalida) {
		t.Fatalf("clave inexistente: %v", err)
	}
	if _, err := catalogo.ResolverSancion(t.Context(), reglas.BolsaPrefijoSanciones+"pasar_al_final", notificada); !errors.Is(err, ports.ErrSancionesNoConfiguradas) {
		t.Fatalf("sin calendario no se supone un vencimiento: %v", err)
	}
	if NuevoCatalogoSanciones(nil) != nil {
		t.Fatal("sin resolutor debe quedar sin catálogo")
	}
}
