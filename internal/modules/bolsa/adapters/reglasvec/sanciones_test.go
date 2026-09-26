package reglasvec

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestEfectosYReversionDelPaqueteDeEjemplo(t *testing.T) {
	catalogo := catalogoPrueba(t, &calculadoraPrueba{})
	consecuencias, err := catalogo.Consecuencias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range consecuencias {
		esperadoFinal := c.Clave == reglas.BolsaPrefijoSanciones+"pasar_al_final"
		esperadoFin := c.Clave == reglas.BolsaPrefijoSanciones+"suspension"
		if c.OrdenFinal != esperadoFinal || c.FinAutomatico != esperadoFin {
			t.Fatalf("efectos de %s: %+v", c.Clave, c)
		}
	}
	reversion, err := catalogo.ReversionRecurso(t.Context())
	if err != nil || !reversion.Revierte("estimado") || reversion.Revierte("desestimado") || reversion.Motivo == "" ||
		reversion.ReglaRef != "vec.bolsa.reglas:1:b24.recurso_revierte" || len(reversion.Huella) != 64 {
		t.Fatalf("reversión=%+v err=%v", reversion, err)
	}
}

// catalogoModificado reescribe una entrada del paquete de ejemplo en un
// fichero temporal para comprobar que un atributo mal formado no se
// interpreta.
func catalogoModificado(t *testing.T, cambiar func(map[string]any)) *CatalogoSanciones {
	t.Helper()
	datos, err := os.ReadFile(rutaReglasBolsa)
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]any
	if err := json.Unmarshal(datos, &documento); err != nil {
		t.Fatal(err)
	}
	cambiar(documento)
	salida, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "bolsa_reglas.ejemplo.demo.json")
	if err := os.WriteFile(ruta, salida, 0o600); err != nil {
		t.Fatal(err)
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoBolsa, ModuloID: reglas.ModuloBolsa,
		Reloj: relojFijo(time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)), Calculadora: &calculadoraPrueba{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return NuevoCatalogoSanciones(resolutor)
}

func entradaCatalogo(documento map[string]any, clave string) map[string]any {
	entradas, _ := documento["catalogo"].(map[string]any)["entradas"].([]any)
	for _, e := range entradas {
		if entrada := e.(map[string]any); entrada["clave"] == clave {
			return entrada
		}
	}
	return nil
}

func TestEfectosMalFormadosNoSeInterpretan(t *testing.T) {
	casos := map[string]func(map[string]any){
		"orden desconocido": func(d map[string]any) {
			entradaCatalogo(d, reglas.BolsaPrefijoSanciones+"pasar_al_final")["atributos"].(map[string]any)["orden"] = "principio"
		},
		"baja al final": func(d map[string]any) {
			entradaCatalogo(d, reglas.BolsaPrefijoSanciones+"baja_sin_contacto")["atributos"].(map[string]any)["orden"] = "final"
		},
		"fin sin plazo": func(d map[string]any) {
			entradaCatalogo(d, reglas.BolsaPrefijoSanciones+"pasar_al_final")["atributos"].(map[string]any)["fin"] = "automatico"
		},
	}
	for nombre, cambiar := range casos {
		if _, err := catalogoModificado(t, cambiar).Consecuencias(t.Context()); !errors.Is(err, ports.ErrSancionesNoConfiguradas) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
	sinEfectos := catalogoModificado(t, func(d map[string]any) {
		delete(entradaCatalogo(d, reglas.BolsaPrefijoSanciones+"suspension")["atributos"].(map[string]any), "fin")
		delete(entradaCatalogo(d, reglas.BolsaPrefijoSanciones+"pasar_al_final")["atributos"].(map[string]any), "orden")
	})
	consecuencias, err := sinEfectos.Consecuencias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range consecuencias {
		if c.OrdenFinal || c.FinAutomatico {
			t.Fatalf("sin atributos no hay efecto: %+v", c)
		}
	}
	sinReversion := catalogoModificado(t, func(d map[string]any) {
		catalogo := d["catalogo"].(map[string]any)
		entradas := catalogo["entradas"].([]any)
		filtradas := make([]any, 0, len(entradas))
		for _, e := range entradas {
			if e.(map[string]any)["clave"] != reglas.BolsaEstadosRecursoRevocatorios {
				filtradas = append(filtradas, e)
			}
		}
		catalogo["entradas"] = filtradas
	})
	if reversion, err := sinReversion.ReversionRecurso(t.Context()); err != nil || reversion.Revierte("estimado") {
		t.Fatalf("sin entrada no revierte: %+v %v", reversion, err)
	}
	malReversion := catalogoModificado(t, func(d map[string]any) {
		entradaCatalogo(d, reglas.BolsaEstadosRecursoRevocatorios)["atributos"].(map[string]any)["valor"] = "Estimado"
	})
	if _, err := malReversion.ReversionRecurso(t.Context()); !errors.Is(err, ports.ErrSancionesNoConfiguradas) {
		t.Fatalf("estado mal formado: %v", err)
	}
}

// La política de no incorporación (Bolsa 000042) recoge todas las
// consecuencias b24 y la regla del recurso, con la referencia de una misma
// versión del catálogo.
func TestPoliticaNoIncorporacionDelPaqueteDeEjemplo(t *testing.T) {
	catalogo := catalogoPrueba(t, nil)
	politica, err := catalogo.PoliticaNoIncorporacion(t.Context())
	if err != nil || len(politica.Consecuencias) != 6 || politica.RecursoReglaRef == "" || len(politica.RecursoReglaHuellaSHA256) != 64 ||
		!strings.HasSuffix(politica.CatalogoRef, ":no_incorporacion") || politica.CatalogoSHA256 != politica.RecursoReglaHuellaSHA256 {
		t.Fatalf("política=%+v err=%v", politica, err)
	}
	baja, ok := politica.Consecuencias["b24.sancion.baja_llamamiento_directo"]
	if !ok || baja.Efecto != dominiobolsa.OperacionExcluir || baja.ConPlazo || baja.OrdenFinal || baja.FinAutomatico || len(baja.ReglaHuellaSHA256) != 64 {
		t.Fatalf("baja=%+v", baja)
	}
	var nulo *CatalogoSanciones
	if _, err := nulo.PoliticaNoIncorporacion(t.Context()); err == nil {
		t.Fatal("sin catálogo no hay política")
	}
}
