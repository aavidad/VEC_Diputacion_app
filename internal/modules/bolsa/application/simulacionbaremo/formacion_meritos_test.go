package simulacionbaremo

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/calculomeritos"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestServicioMeritosV2CategoriaPlazaYReglaPublicada(t *testing.T) {
	base := solicitudPrueba(t, "meritos_reglas_a", "meritos_entrada")
	var c calculomeritos.Conjunto
	var e calculomeritos.Entrada
	if err := json.Unmarshal(base.ConjuntoCanonico, &c); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(base.EntradaCanonica, &e); err != nil {
		t.Fatal(err)
	}
	c.Esquema, e.Esquema = calculomeritos.EsquemaReglasV2, calculomeritos.EsquemaEntradaV2
	c.CategoriaRef = "cat_00000000000000000000000000000021"
	c.PlazaRef = "pla_00000000000000000000000000000022"
	desde, err := baremacion.NuevaFechaCivil(2020, 9, 30)
	if err != nil {
		t.Fatal(err)
	}
	reciente, err := baremacion.PuntosDesdeMicropuntos(40_000)
	if err != nil {
		t.Fatal(err)
	}
	c.Reglas[0].ConceptosRelacionados = []string{"con_00000000000000000000000000000031"}
	c.Reglas[0].Temporal = &calculomeritos.PoliticaFormacion{DesdeBonificacion: desde, EfectoAnterior: "bonificar", PuntosRecientes: reciente}
	e.Meritos = e.Meritos[:1]
	e.Meritos[0].TipoHecho = "curso_asistencia"
	e.Meritos[0].ConceptoRef = c.Reglas[0].ConceptosRelacionados[0]
	e.Meritos[0].OrigenRef = "ori_00000000000000000000000000000051"
	cb, err := c.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	eb, err := e.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	r, err := (ServicioMeritos{}).SimularMeritos(Solicitud{cb, huellaPrueba(cb), eb, huellaPrueba(eb)})
	if err != nil {
		t.Fatal(err)
	}
	var envoltorio struct {
		Esquema   string
		Alcance   string
		Resultado struct {
			Esquema string
			Bases   struct {
				Referencia   string
				Version      uint64
				HuellaSHA256 string `json:"huella_sha256"`
			}
			Estado       string
			CategoriaRef string `json:"categoria_ref"`
			PlazaRef     string `json:"plaza_ref"`
			Meritos      []struct {
				Validez         string
				ReglaClave      string `json:"regla_clave"`
				ReglaDefinicion struct {
					Referencia   string
					Version      uint64
					HuellaSHA256 string `json:"huella_sha256"`
				} `json:"regla_definicion"`
				Codigo              string
				Corte               string
				PuntosProvisionales string `json:"puntos_provisionales"`
			}
		}
	}
	if err := json.Unmarshal(r.RepresentacionCanonica(), &envoltorio); err != nil {
		t.Fatal(err)
	}
	sem := envoltorio.Resultado
	if envoltorio.Esquema != "vec.bolsa.simulacion_meritos.v2" || envoltorio.Alcance != "simulacion" || sem.Esquema != calculomeritos.EsquemaResultadoV2 || sem.Bases.Referencia != c.Bases.Referencia || sem.Bases.Version != c.Bases.Version || sem.Bases.HuellaSHA256 != c.Bases.HuellaSHA256 || sem.Estado != "completado" || sem.CategoriaRef != c.CategoriaRef || sem.PlazaRef != c.PlazaRef || len(sem.Meritos) != 1 || sem.Meritos[0].Validez != "aplicable_segun_regla" || sem.Meritos[0].ReglaClave != "curso" || sem.Meritos[0].ReglaDefinicion.Referencia != c.Reglas[0].Definicion.Referencia || sem.Meritos[0].Codigo != "admitido" || sem.Meritos[0].Corte != c.FechaCorteInclusiva.String() || sem.Meritos[0].PuntosProvisionales == "" {
		t.Fatalf("simulación V2 no llegó al consumidor: %+v", sem)
	}
	dir := t.TempDir()
	reglas, entrada := filepath.Join(dir, "reglas.json"), filepath.Join(dir, "entrada.json")
	if err := os.WriteFile(reglas, cb, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrada, eb, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/vec-baremador", "--modo", "meritos", "--reglas", reglas,
		"--reglas-sha256", huellaPrueba(cb), "--entrada", entrada, "--entrada-sha256", huellaPrueba(eb))
	cmd.Dir = filepath.Join("..", "..", "..", "..", "..")
	salida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI V2 falló: %v: %s", err, salida)
	}
	if !bytes.Equal(salida, r.RepresentacionCanonica()) {
		t.Fatal("CLI real no consume la simulación V2")
	}
}
