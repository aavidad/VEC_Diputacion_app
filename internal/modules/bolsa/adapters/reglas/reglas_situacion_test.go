package reglas

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

const rutaReglasBolsaPrueba = "../../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"

type relojPrueba struct{}

func (relojPrueba) Ahora() time.Time { return time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC) }

// calculadoraPrueba devuelve un vencimiento fijo y recuerda la solicitud.
type calculadoraPrueba struct {
	recibida vecreglas.SolicitudVencimiento
}

func (c *calculadoraPrueba) CalcularVencimiento(_ context.Context, s vecreglas.SolicitudVencimiento) (vecreglas.Vencimiento, error) {
	c.recibida = s
	return vecreglas.Vencimiento{UltimoDia: "2026-06-15", VenceAntesDe: time.Date(2026, 6, 15, 22, 0, 0, 0, time.UTC)}, nil
}

func reglasPrueba(t *testing.T, calculadora vecreglas.CalculadoraPlazos) *ReglasSituacion {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasBolsaPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa,
		Reloj: relojPrueba{}, Calculadora: calculadora, MunicipioSede: vecreglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return NuevasReglasSituacion(resolutor)
}

func TestSinCatalogoNadaEstaConfigurado(t *testing.T) {
	reglas := NuevasReglasSituacion(nil)
	if reglas.Configurada() {
		t.Fatal("sin resolutor no hay catálogo")
	}
	if destinos, configurada, err := reglas.DestinosSituacion(t.Context(), "renuncia"); destinos != nil || configurada || err != nil {
		t.Fatalf("sin catálogo rige la tabla compilada: %v %v %v", destinos, configurada, err)
	}
	if _, err := reglas.CausasBaja(t.Context()); !errors.Is(err, puertosbolsa.ErrReglasSituacionNoConfiguradas) {
		t.Fatalf("causas sin catálogo: %v", err)
	}
	if _, err := reglas.ProponerReposicion(t.Context(), time.Now(), ""); !errors.Is(err, puertosbolsa.ErrReglasSituacionNoConfiguradas) {
		t.Fatalf("propuesta sin catálogo: %v", err)
	}
}

func TestTransicionesDelCatalogoSoloParaLosOrigenesDeclarados(t *testing.T) {
	reglas := reglasPrueba(t, nil)
	destinos, configurada, err := reglas.DestinosSituacion(t.Context(), "renuncia")
	if err != nil || !configurada || !slices.Equal(destinos, []string{"excluido"}) {
		t.Fatalf("renuncia: %v %v %v", destinos, configurada, err)
	}
	if destinos, configurada, err := reglas.DestinosSituacion(t.Context(), "trabajando"); destinos != nil || configurada || err != nil {
		t.Fatalf("trabajando no está en el catálogo: %v %v %v", destinos, configurada, err)
	}
}

func TestCausasBajaConSuArticulo(t *testing.T) {
	causas, err := reglasPrueba(t, nil).CausasBaja(t.Context())
	if err != nil || len(causas) != 6 {
		t.Fatalf("causas=%+v err=%v", causas, err)
	}
	if causas[0].Codigo != "no_acepta" || causas[0].Procedencia.Articulo != "art. 11.1.a" ||
		causas[0].Procedencia.Referencia != "vec.bolsa.reglas:1:b27.causa_baja.no_acepta" || causas[0].Procedencia.Ejemplo {
		t.Fatalf("primera causa: %+v", causas[0])
	}
	if causas[5].Codigo != "renuncia_nombramiento" || causas[5].Procedencia.Articulo != "art. 11.2" {
		t.Fatalf("última causa: %+v", causas[5])
	}
}

func TestPropuestaEligeLaReglaPorModalidad(t *testing.T) {
	calculadora := &calculadoraPrueba{}
	reglas := reglasPrueba(t, calculadora)
	modalidades, err := reglas.ModalidadesReposicion(t.Context())
	if err != nil || len(modalidades) != 1 || modalidades[0] != (puertosbolsa.ModalidadReposicion{Codigo: "acumulacion_tareas", Meses: 9}) {
		t.Fatalf("modalidades=%+v err=%v", modalidades, err)
	}
	fin := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		modalidad, clave string
		meses            int
	}{
		{"", vecreglas.BolsaReposicionGeneral, 5},
		{ModalidadGeneral, vecreglas.BolsaReposicionGeneral, 5},
		{"sustitucion", vecreglas.BolsaReposicionGeneral, 5},
		{"acumulacion_tareas", vecreglas.BolsaReposicionAcumulacionTareas, 9},
	}
	for _, caso := range casos {
		propuesta, err := reglas.ProponerReposicion(t.Context(), fin, caso.modalidad)
		if err != nil || propuesta.Meses != caso.meses || propuesta.Procedencia.Clave != caso.clave ||
			propuesta.Procedencia.Articulo != "art. 9.1" || calculadora.recibida.Computo != vecreglas.ComputoCivil ||
			calculadora.recibida.Cantidad != caso.meses || !calculadora.recibida.Inicio.Equal(fin) ||
			!propuesta.FechaDisponible.Equal(time.Date(2026, 6, 15, 22, 0, 0, 0, time.UTC)) || propuesta.UltimoDiaNoDisponible != "2026-06-15" {
			t.Fatalf("%q: propuesta=%+v err=%v", caso.modalidad, propuesta, err)
		}
	}
	if _, err := reglas.ProponerReposicion(t.Context(), time.Time{}, ""); !errors.Is(err, puertosbolsa.ErrReposicionNoCalculable) {
		t.Fatalf("sin fecha de fin: %v", err)
	}
}

func TestPropuestaSinCalculadoraNoInventaUnaFecha(t *testing.T) {
	_, err := reglasPrueba(t, nil).ProponerReposicion(t.Context(), time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), "")
	if !errors.Is(err, puertosbolsa.ErrReglasSituacionNoDisponibles) {
		t.Fatalf("sin cálculo de plazos: %v", err)
	}
}
