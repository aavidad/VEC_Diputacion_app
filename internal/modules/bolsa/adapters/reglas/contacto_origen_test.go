package reglas

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

const rutaCatalogoContactoOrigenPrueba = "../../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"

type relojContactoOrigenPrueba struct{}

func (relojContactoOrigenPrueba) Ahora() time.Time {
	return time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
}

// calculadoraContactoOrigenPrueba registra lo que pide la regla y responde
// una fecha fija: el cómputo real se prueba en la composición.
type calculadoraContactoOrigenPrueba struct {
	pedido vecreglas.SolicitudVencimiento
}

func (c *calculadoraContactoOrigenPrueba) CalcularVencimiento(_ context.Context, s vecreglas.SolicitudVencimiento) (vecreglas.Vencimiento, error) {
	c.pedido = s
	return vecreglas.Vencimiento{UltimoDia: "2027-09-28", VenceAntesDe: time.Date(2027, 9, 28, 22, 0, 0, 0, time.UTC)}, nil
}

func resolutorContactoOrigenPrueba(t *testing.T, ruta string, calculadora vecreglas.CalculadoraPlazos) *vecreglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa,
		Reloj: relojContactoOrigenPrueba{}, Calculadora: calculadora, MunicipioSede: vecreglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestContactoOrigenLeeLaReglaB29DelCatalogoDeEjemplo(t *testing.T) {
	calculadora := &calculadoraContactoOrigenPrueba{}
	politica := NuevoContactoOrigen(resolutorContactoOrigenPrueba(t, rutaCatalogoContactoOrigenPrueba, calculadora))
	registrada := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	marca, err := politica.MarcaOrigenConvoca(t.Context(), registrada)
	if err != nil || marca.Validar() != nil {
		t.Fatalf("marca=%+v err=%v", marca, err)
	}
	p := calculadora.pedido
	if p.Unidad != vecreglas.UnidadMeses || p.Cantidad != 12 || p.Computo != vecreglas.ComputoCivil || !p.Inicio.Equal(registrada) {
		t.Fatalf("la vigencia debe salir de la regla: %+v", p)
	}
	if marca.Origen != dominiobolsa.OrigenDatosContactoConvoca || marca.UltimoDia != "2027-09-28" ||
		marca.ReglaRef != "vec.bolsa.reglas:1:b29.contacto_origen_convoca" || len(marca.ReglaHuella) != 64 {
		t.Fatalf("marca: %+v", marca)
	}
}

func TestContactoOrigenSinCatalogoOReglaIncompletaNoDaVigencia(t *testing.T) {
	if NuevoContactoOrigen(nil).Configurada() {
		t.Fatal("sin catálogo no está configurada")
	}
	if _, err := NuevoContactoOrigen(nil).MarcaOrigenConvoca(t.Context(), time.Now()); !errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado) {
		t.Fatalf("sin catálogo: %v", err)
	}
	// Una regla b29 sin el origen al que se aplica no sirve.
	contenido, err := os.ReadFile(rutaCatalogoContactoOrigenPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	for _, e := range paquete["catalogo"].(map[string]any)["entradas"].([]any) {
		entrada := e.(map[string]any)
		if entrada["clave"] == vecreglas.BolsaContactoOrigenConvoca {
			delete(entrada["atributos"].(map[string]any), "origen_contacto")
		}
	}
	ruta := filepath.Join(t.TempDir(), "bolsa_reglas.demo.json")
	contenido, _ = json.Marshal(paquete)
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	politica := NuevoContactoOrigen(resolutorContactoOrigenPrueba(t, ruta, &calculadoraContactoOrigenPrueba{}))
	if _, err := politica.MarcaOrigenConvoca(t.Context(), time.Now()); !errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado) {
		t.Fatalf("regla incompleta: %v", err)
	}
	// Sin cálculo de plazos no hay vigencia supuesta.
	sinCalculo := NuevoContactoOrigen(resolutorContactoOrigenPrueba(t, rutaCatalogoContactoOrigenPrueba, nil))
	if _, err := sinCalculo.MarcaOrigenConvoca(t.Context(), time.Now()); !errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado) {
		t.Fatalf("sin cálculo: %v", err)
	}
}
