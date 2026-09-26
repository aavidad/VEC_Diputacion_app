package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Las cuatro publicaciones fijas (v1 y v2) conservan sus bytes exactos: la v2
// se construye ahora desde las vías de siempre y el replay SQL exige el mismo
// contenido que ya está publicado en las bases.
func TestPublicacionesFijasGobiernoCoberturaConservanSusBytes(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	publicaciones, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		t.Fatal(err)
	}
	esperadas := []string{
		"7d4abac4ca1358b9903bcc73a7d66ce763054274bd5c64263c77842eed92c701",
		"777c5fc8d8f69614897ea68ddf86062e1c78a7788c635d2d71f1e904f67739c7",
		"dacc2922a4ba78337cfe2dcdac7c4015a16ced079132f0713b94175d7a41efde",
		"b1a890bdb355fd50e9e5d0e93cb934884b6a4d5bf7037f01371f442608633995",
	}
	if len(publicaciones) != len(esperadas) {
		t.Fatalf("publicaciones=%d", len(publicaciones))
	}
	for indice, publicacion := range publicaciones {
		contenido, err := json.Marshal(publicacion)
		if err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(contenido)
		if hex.EncodeToString(suma[:]) != esperadas[indice] {
			t.Fatalf("publicación fija %d alterada", indice+1)
		}
	}
}

func reglaViaCoberturaPrueba(clave, valor, procedencia, prioridad string) reglas.Regla {
	return reglas.Regla{
		Clave: prefijoViaCoberturaCT + clave, Etiqueta: "Vía " + clave, Unidad: reglas.UnidadLista,
		Valor: valor, Atributos: map[string]string{
			atributoProcedenciaViaCT: procedencia, atributoPrioridadViaCT: prioridad,
		},
	}
}

func TestViasCoberturaDelPaqueteDeEjemploSonLasDeSiempre(t *testing.T) {
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, rutaReglasCTEjemploPrueba))
	if err != nil {
		t.Fatal(err)
	}
	if len(opciones.viasCobertura) != 3 ||
		huellaViasCoberturaCT(opciones.viasCobertura) != huellaViasCoberturaCT(viasCoberturaPredeterminadasCT()) {
		t.Fatalf("vías del ejemplo distintas de las de siempre: %+v", opciones.viasCobertura)
	}
	if opciones.viasCobertura[1].Etiqueta != "Oferta SAE" {
		t.Fatalf("etiqueta perdida: %+v", opciones.viasCobertura[1])
	}
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	deseado, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, opciones.viasCoberturaVigentes())
	if err != nil {
		t.Fatal(err)
	}
	fijas, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		t.Fatal(err)
	}
	// Mismo contenido que la v2: se reutiliza, no se publica otra versión.
	for indice, actuacion := range deseado.actuaciones {
		if actuacion.HuellaSHA256 != fijas[indice+2].Actuacion.HuellaSHA256 {
			t.Fatalf("el catálogo de ejemplo no reutiliza la v2 (acción %d)", indice)
		}
	}
	if sin, err := nuevasOpcionesAnalisisCT(t.Context(), nil); err != nil || sin.viasCobertura != nil {
		t.Fatalf("sin catálogo no hay vías propias: %+v %v", sin, err)
	}
}

func TestViasCoberturaNuevasPublicanVersionPropiaDeterminista(t *testing.T) {
	vias, err := viasCoberturaDesdeReglasCT([]reglas.Regla{
		reglaViaCoberturaPrueba("oferta_sae", "oferta_sae_disponible", "sae", "20"),
		reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "10"),
		reglaViaCoberturaPrueba("bolsa_otra_categoria", "existe_bolsa_afin,hay_candidaturas_disponibles", "bolsa", "30"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if vias[0].Clave != "bolsa_vigente" || vias[2].Clave != "bolsa_otra_categoria" ||
		len(vias[2].Comprobaciones) != 2 {
		t.Fatalf("orden por prioridad no respetado: %+v", vias)
	}
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	primero, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, vias)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, vias)
	if err != nil {
		t.Fatal(err)
	}
	prefijo := "catalogo:ct:desarrollo:cobertura:reglas-" + huellaViasCoberturaCT(vias)[:16]
	if primero.catalogo.Referencia != prefijo || primero.catalogo.Version != versionGobiernoCatalogoCT ||
		len(primero.catalogo.Vias) != 3 || len(primero.actuaciones) != 2 {
		t.Fatalf("publicación del catálogo inesperada: %+v", primero.catalogo)
	}
	for indice := range primero.actuaciones {
		if primero.actuaciones[indice].HuellaSHA256 != segundo.actuaciones[indice].HuellaSHA256 {
			t.Fatal("el mismo catálogo dio otra huella")
		}
	}
	// La etiqueta no forma parte del gobierno: cambiarla no publica nada.
	renombradas := append([]viaCoberturaCT(nil), vias...)
	renombradas[0].Etiqueta = "Otra etiqueta"
	if huellaViasCoberturaCT(renombradas) != huellaViasCoberturaCT(vias) {
		t.Fatal("la etiqueta cambió la huella")
	}
	motivos := motivosAlternativaViasCoberturaCT(vias)
	if len(motivos) != 2 || motivos[0].ViaClave != "oferta_sae" || motivos[1].ViaClave != "bolsa_otra_categoria" {
		t.Fatalf("motivos de alternativa inesperados: %+v", motivos)
	}
	plantillas := plantillasViasCoberturaCT(vias)
	if len(plantillas) != 4 || plantillas[3].comprobacion != "hay_candidaturas_disponibles" || plantillas[3].orden != 2 {
		t.Fatalf("plantillas inesperadas: %+v", plantillas)
	}
}

func TestViasCoberturaRechazanCatalogoRoto(t *testing.T) {
	casos := map[string][]reglas.Regla{
		"prioridad repetida": {
			reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "1"),
			reglaViaCoberturaPrueba("oferta_sae", "oferta_sae_disponible", "sae", "1"),
		},
		"prioridad no numérica":   {reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "uno")},
		"prioridad no canónica":   {reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "01")},
		"sin procedencia":         {reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "", "1")},
		"comprobación inválida":   {reglaViaCoberturaPrueba("bolsa_vigente", "Existe Bolsa", "bolsa", "1")},
		"comprobación repetida":   {reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente,existe_bolsa_vigente", "bolsa", "1")},
		"clave de vía inválida":   {reglaViaCoberturaPrueba("Bolsa", "existe_bolsa_vigente", "bolsa", "1")},
		"procedencia incoherente": {reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "1"), reglaViaCoberturaPrueba("oferta_sae", "existe_bolsa_vigente", "sae", "2")},
	}
	for nombre, entradas := range casos {
		if _, err := viasCoberturaDesdeReglasCT(entradas); !errors.Is(err, errViasCoberturaNoValidas) {
			t.Fatalf("%s: se aceptó un catálogo roto (%v)", nombre, err)
		}
	}
	sinUnidad := reglaViaCoberturaPrueba("bolsa_vigente", "existe_bolsa_vigente", "bolsa", "1")
	sinUnidad.Unidad = reglas.UnidadNinguna
	if _, err := viasCoberturaDesdeReglasCT([]reglas.Regla{sinUnidad}); !errors.Is(err, errViasCoberturaNoValidas) {
		t.Fatal("una vía sin lista de comprobaciones debía rechazarse")
	}
}

// gobiernoSQLSimulado reproduce lo que decide gobi_o404b_publicar: secuencia
// contigua compartida, replay idéntico «repetida», replay divergente rechazado
// y puntero por acción que solo avanza.
type gobiernoSQLSimulado struct {
	eventos     map[uint64]string
	checkpoint  uint64
	puntero     map[domain.ClaveCatalogo]string
	conflictos  int
	publicacion int
}

func nuevoGobiernoSQLSimulado(t *testing.T, fijas []publicacionGobiernoCoberturaDesarrollo) *gobiernoSQLSimulado {
	t.Helper()
	simulado := &gobiernoSQLSimulado{eventos: map[uint64]string{}, puntero: map[domain.ClaveCatalogo]string{}}
	for _, publicacion := range fijas {
		if resultado, err := simulado.publicar(t.Context(), publicacion); err != nil || resultado != intentoGobiernoPublicado {
			t.Fatalf("publicación fija %d: %s %v", publicacion.Secuencia, resultado, err)
		}
	}
	return simulado
}

func (g *gobiernoSQLSimulado) vigente(_ context.Context, accion domain.ClaveCatalogo) (string, error) {
	return g.puntero[accion], nil
}

func (g *gobiernoSQLSimulado) publicar(_ context.Context, publicacion publicacionGobiernoCoberturaDesarrollo) (string, error) {
	contenido, err := json.Marshal(publicacion)
	if err != nil {
		return "", err
	}
	if g.conflictos > 0 {
		g.conflictos--
		return intentoGobiernoReintentar, nil
	}
	switch {
	case publicacion.Secuencia <= g.checkpoint:
		return intentoGobiernoOcupado, nil
	case publicacion.Secuencia != g.checkpoint+1:
		return "", errors.New("secuencia no contigua")
	}
	g.eventos[publicacion.Secuencia] = string(contenido)
	g.checkpoint = publicacion.Secuencia
	g.puntero[publicacion.Actuacion.Accion] = publicacion.Actuacion.HuellaSHA256
	g.publicacion++
	return intentoGobiernoPublicado, nil
}

func (g *gobiernoSQLSimulado) retirar() {
	g.checkpoint++
	g.eventos[g.checkpoint] = "retirada"
}

func (g *gobiernoSQLSimulado) dependencias() dependenciasSincronizacionGobiernoCT {
	return dependenciasSincronizacionGobiernoCT{vigente: g.vigente, publicar: g.publicar}
}

func TestSincronizacionGobiernoCoberturaSoloPublicaSiCambiaElContenido(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	fijas, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		t.Fatal(err)
	}
	gobierno := nuevoGobiernoSQLSimulado(t, fijas)
	sincronizar := func(vias []viaCoberturaCT) int {
		t.Helper()
		deseado, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, vias)
		if err != nil {
			t.Fatal(err)
		}
		publicadas, err := sincronizarGobiernoCoberturaCT(t.Context(), gobierno.dependencias(), deseado, uint64(len(fijas))+1)
		if err != nil {
			t.Fatal(err)
		}
		return publicadas
	}
	// Sin catálogo (o con el de ejemplo): dos arranques, ninguna publicación.
	for arranque := 1; arranque <= 2; arranque++ {
		if publicadas := sincronizar(viasCoberturaPredeterminadasCT()); publicadas != 0 || gobierno.checkpoint != 4 {
			t.Fatalf("arranque %d sin cambios publicó %d (secuencia %d)", arranque, publicadas, gobierno.checkpoint)
		}
	}
	otras := append(viasCoberturaPredeterminadasCT()[:2:2], viaCoberturaCT{
		Clave: "bolsa_otra_categoria", Procedencia: "bolsa", Comprobaciones: []domain.ClaveCatalogo{"existe_bolsa_afin"},
	})
	// Cambio de catálogo: una versión nueva, en las secuencias 5 y 6.
	if publicadas := sincronizar(otras); publicadas != 2 || gobierno.checkpoint != 6 {
		t.Fatalf("cambio de catálogo publicó %d (secuencia %d)", publicadas, gobierno.checkpoint)
	}
	for arranque := 1; arranque <= 2; arranque++ {
		if publicadas := sincronizar(otras); publicadas != 0 || gobierno.checkpoint != 6 {
			t.Fatalf("rearranque %d con el mismo catálogo publicó %d", arranque, publicadas)
		}
	}
	// Una retirada ocupa la secuencia 7; volver a las de siempre publica en 8 y 9.
	gobierno.retirar()
	if publicadas := sincronizar(viasCoberturaPredeterminadasCT()); publicadas != 2 || gobierno.checkpoint != 9 {
		t.Fatalf("volver a las de siempre publicó %d (secuencia %d)", publicadas, gobierno.checkpoint)
	}
	// Repetir el catálogo anterior salta sus eventos antiguos y publica en 10 y 11,
	// tras reintentar un conflicto de serialización.
	gobierno.conflictos = 2
	if publicadas := sincronizar(otras); publicadas != 2 || gobierno.checkpoint != 11 {
		t.Fatalf("repetir el catálogo anterior publicó %d (secuencia %d)", publicadas, gobierno.checkpoint)
	}
	if !strings.Contains(gobierno.eventos[10], "reglas-") {
		t.Fatal("la secuencia 10 no es la versión del catálogo")
	}
}

func TestSincronizacionGobiernoCoberturaFallaCerrada(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	deseado, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, viasCoberturaPredeterminadasCT())
	if err != nil {
		t.Fatal(err)
	}
	fallo := errors.New("sin conexión")
	casos := []dependenciasSincronizacionGobiernoCT{
		{},
		{vigente: func(context.Context, domain.ClaveCatalogo) (string, error) { return "", fallo }, publicar: func(context.Context, publicacionGobiernoCoberturaDesarrollo) (string, error) {
			return intentoGobiernoPublicado, nil
		}},
		{vigente: func(context.Context, domain.ClaveCatalogo) (string, error) { return "", nil }, publicar: func(context.Context, publicacionGobiernoCoberturaDesarrollo) (string, error) { return "", fallo }},
		{vigente: func(context.Context, domain.ClaveCatalogo) (string, error) { return "", nil }, publicar: func(context.Context, publicacionGobiernoCoberturaDesarrollo) (string, error) {
			return intentoGobiernoReintentar, nil
		}},
	}
	for indice, dependencias := range casos {
		if _, err := sincronizarGobiernoCoberturaCT(t.Context(), dependencias, deseado, 5); !errors.Is(err, errGobiernoCoberturaCatalogoNoPublicado) {
			t.Fatalf("caso %d: no falló cerrado (%v)", indice, err)
		}
	}
}
