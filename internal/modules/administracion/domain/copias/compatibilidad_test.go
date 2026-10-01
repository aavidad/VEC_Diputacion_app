package copias

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type caso struct {
	Manifiesto Manifiesto       `json:"manifiesto"`
	Destino    Inventario       `json:"destino"`
	Politica   Politica         `json:"politica"`
	Modo       ModoRestauracion `json:"modo"`
}

func ejemplo(t *testing.T) caso {
	t.Helper()
	b, err := os.ReadFile("../../../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if err != nil {
		t.Fatal(err)
	}
	var e caso
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("ejemplo inválido")
	}
	return e
}

func contiene(r Resultado, codigo string) bool {
	for _, x := range r.Razones {
		if x.Codigo == codigo {
			return true
		}
	}
	return false
}

func TestConjuntoFinalArchivadoCompatibleAunqueDestinoTengaOtraRelease(t *testing.T) {
	e := ejemplo(t)
	if e.Manifiesto.Inventario.Release.ID == e.Destino.Release.ID {
		t.Fatal("el destino debe tener otra release")
	}
	r := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	if r.Estado != Compatible || len(r.Razones) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestBloqueosDeVersionesYEsquema(t *testing.T) {
	for _, tt := range []struct {
		nombre, codigo string
		editar         func(*caso)
	}{
		{"minor_pg", "postgresql_version_diferente", func(e *caso) { e.Destino.PostgreSQL.Version = "18.5" }},
		{"runtime", "runtime_diferente", func(e *caso) { e.Destino.PostgreSQL.RuntimeSHA256 = strings.Repeat("f", 64) }},
		{"herramienta", "runtime_diferente", func(e *caso) { e.Destino.PostgreSQL.Herramientas[0].SHA256 = strings.Repeat("f", 64) }},
		{"extension", "runtime_diferente", func(e *caso) { e.Destino.PostgreSQL.Extensiones[0].Version = "2.0" }},
		{"binario", "release_no_admitida", func(e *caso) { e.Manifiesto.Inventario.Release.Binarios[0].SHA256 = strings.Repeat("f", 64) }},
		{"migracion_misma_id_otra_huella", "esquema_migraciones_diferentes", func(e *caso) { e.Manifiesto.Inventario.Modulos[0].Migraciones[0].SHA256 = strings.Repeat("f", 64) }},
		{"modulo_omitido", "modulos_diferentes", func(e *caso) { e.Manifiesto.Inventario.Modulos = e.Manifiesto.Inventario.Modulos[:1] }},
		{"binario_nuevo_esquema_antiguo", "release_no_admitida", func(e *caso) {
			e.Manifiesto.Inventario.Release.EsquemaEsperado[0].EsquemaSHA256 = strings.Repeat("f", 64)
		}},
		{"release_revocada", "release_revocada", func(e *caso) { e.Politica.ReleasesRevocadas = []string{e.Manifiesto.Inventario.Release.ID} }},
		{"release_no_admitida", "release_no_admitida", func(e *caso) { e.Politica.Releases[0].ID = "release:otra" }},
		{"runtime_no_admitido", "runtime_no_admitido", func(e *caso) { e.Politica.Runtimes[0].RuntimeSHA256 = strings.Repeat("f", 64) }},
		{"base_ajena", "bases_diferentes", func(e *caso) { e.Destino.PostgreSQL.Bases = append(e.Destino.PostgreSQL.Bases, "base:ajena") }},
		{"solo_base", "modo_no_admitido", func(e *caso) { e.Modo = "solo_base"; e.Destino.Release = e.Manifiesto.Inventario.Release }},
		{"modo_desconocido", "modo_no_admitido", func(e *caso) { e.Modo = "forzar" }},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			e := ejemplo(t)
			tt.editar(&e)
			e.Manifiesto.InventarioSHA256 = HuellaInventario(e.Manifiesto.Inventario)
			r := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
			if r.Estado != Incompatible || !contiene(r, tt.codigo) {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestInventarioIncompletoODuplicadoNoEsComprobable(t *testing.T) {
	for _, tt := range []struct {
		nombre string
		editar func(*caso)
	}{
		{"inventario_incompleto", func(e *caso) { e.Destino.Completo = false }},
		{"cluster_compartido", func(e *caso) { e.Destino.PostgreSQL.AmbitoCompleto = false }},
		{"binario_ausente", func(e *caso) { e.Manifiesto.Inventario.Release.Binarios = nil }},
		{"huella_ausente", func(e *caso) { e.Manifiesto.Inventario.Modulos[0].Migraciones[0].SHA256 = "" }},
		{"esquema_ausente", func(e *caso) { e.Destino.Modulos[0].EsquemaSHA256 = "" }},
		{"modulos_vacios", func(e *caso) { e.Destino.Modulos = nil }},
		{"modulo_duplicado", func(e *caso) { e.Destino.Modulos = append(e.Destino.Modulos, e.Destino.Modulos[0]) }},
		{"migracion_duplicada", func(e *caso) {
			e.Destino.Modulos[0].Migraciones = append(e.Destino.Modulos[0].Migraciones, e.Destino.Modulos[0].Migraciones[0])
		}},
		{"artefacto_duplicado", func(e *caso) {
			e.Manifiesto.Componentes = append(e.Manifiesto.Componentes, e.Manifiesto.Componentes[0])
		}},
		{"base_duplicada", func(e *caso) {
			e.Destino.PostgreSQL.Bases = append(e.Destino.PostgreSQL.Bases, e.Destino.PostgreSQL.Bases[0])
		}},
		{"politica_release_duplicada", func(e *caso) { e.Politica.Releases = append(e.Politica.Releases, e.Politica.Releases[0]) }},
		{"politica_runtime_duplicado", func(e *caso) { e.Politica.Runtimes = append(e.Politica.Runtimes, e.Politica.Runtimes[0]) }},
		{"formato_desconocido", func(e *caso) { e.Manifiesto.FormatoVersion = 2 }},
		{"tipo_componente_faltante", func(e *caso) { e.Manifiesto.Componentes = e.Manifiesto.Componentes[:len(e.Manifiesto.Componentes)-1] }},
		{"total_incoherente", func(e *caso) { e.Manifiesto.TamanoBytes++ }},
		{"total_desbordado", func(e *caso) { e.Manifiesto.Componentes[0].TamanoBytes = math.MaxInt64 }},
		{"sin_parada", func(e *caso) { e.Manifiesto.Consistencia.ParadaLimpia = false }},
		{"fecha_invalida", func(e *caso) { e.Manifiesto.Fin = e.Manifiesto.Inicio.Add(-1) }},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			e := ejemplo(t)
			tt.editar(&e)
			if r := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo); r.Estado != NoComprobable {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func invertirInventario(i *Inventario) {
	slices.Reverse(i.Modulos)
	slices.Reverse(i.Release.EsquemaEsperado)
	slices.Reverse(i.Release.Binarios)
	slices.Reverse(i.Release.Componentes)
	slices.Reverse(i.PostgreSQL.Herramientas)
	slices.Reverse(i.PostgreSQL.Extensiones)
	slices.Reverse(i.PostgreSQL.Bases)
	slices.Reverse(i.PostgreSQL.Almacenes)
	for j := range i.Modulos {
		slices.Reverse(i.Modulos[j].Migraciones)
	}
	for j := range i.Release.EsquemaEsperado {
		slices.Reverse(i.Release.EsquemaEsperado[j].Migraciones)
	}
}

func TestListasSonConjuntosYComparacionNoMuta(t *testing.T) {
	e := ejemplo(t)
	antes, _ := json.Marshal(e)
	r1 := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	despues, _ := json.Marshal(e)
	if string(antes) != string(despues) {
		t.Fatal("la comparación mutó su entrada")
	}
	huella := HuellaInventario(e.Manifiesto.Inventario)
	invertirInventario(&e.Manifiesto.Inventario)
	invertirInventario(&e.Destino)
	slices.Reverse(e.Manifiesto.Componentes)
	slices.Reverse(e.Politica.Runtimes[0].Herramientas)
	slices.Reverse(e.Politica.Releases[0].Binarios)
	slices.Reverse(e.Politica.Releases[0].Componentes)
	slices.Reverse(e.Politica.Releases[0].EsquemaEsperado)
	if huella != HuellaInventario(e.Manifiesto.Inventario) {
		t.Fatal("el orden alteró la huella")
	}
	r2 := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("%+v / %+v", r1, r2)
	}
}

func TestComparacionPreCopiaSinManifiesto(t *testing.T) {
	e := ejemplo(t)
	i := e.Manifiesto.Inventario
	if r := CompararInventarios(i, i); r.Estado != Compatible {
		t.Fatalf("%+v", r)
	}
	observado := ejemplo(t).Manifiesto.Inventario
	observado.Modulos[0].Migraciones[0].SHA256 = strings.Repeat("f", 64)
	if r := CompararInventarios(i, observado); r.Estado != Incompatible || !contiene(r, "esquema_migraciones_diferentes") {
		t.Fatalf("%+v", r)
	}
}

func TestHuellaManifiestoAlteradaBloqueaConDosHuellasSinPublicarSecretos(t *testing.T) {
	e := ejemplo(t)
	e.Manifiesto.InventarioSHA256 = strings.Repeat("f", 64)
	r := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	if r.Estado != Incompatible || !contiene(r, "inventario_huella_diferente") {
		t.Fatalf("%+v", r)
	}
	b, _ := json.Marshal(r)
	if !strings.Contains(string(b), e.Manifiesto.InventarioSHA256) || !strings.Contains(string(b), HuellaInventario(e.Manifiesto.Inventario)) {
		t.Fatal("no muestra los dos valores exactos seguros")
	}
	if strings.Contains(string(b), e.Manifiesto.Inventario.Release.Componentes[0].SHA256) {
		t.Fatal("filtra huellas privadas")
	}
}

func TestMigracionAlteradaExponeClaveYDosHuellasSQLExactas(t *testing.T) {
	e := ejemplo(t)
	esperado := e.Manifiesto.Inventario.Release.EsquemaEsperado[0].Migraciones[0].SHA256
	obtenido := strings.Repeat("f", 64)
	e.Manifiesto.Inventario.Modulos[0].Migraciones[0].SHA256 = obtenido
	e.Manifiesto.InventarioSHA256 = HuellaInventario(e.Manifiesto.Inventario)
	r := CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	for _, x := range r.Razones {
		if x.Codigo == "migracion_huella_diferente" && x.Clave == "copia.esquema_instalado.modulos[0].migraciones[0].sha256" && x.Esperado == esperado && x.Obtenido == obtenido {
			return
		}
	}
	t.Fatalf("falta comparación puntual: %+v", r)
}
