package ginpixfichero

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type ficheroGINPIXPrueba struct {
	Esquema   string `json:"esquema"`
	Version   uint64 `json:"version"`
	Metadatos struct {
		EsquemaModelo        string `json:"esquema_modelo"`
		EsquemaMapeo         string `json:"esquema_mapeo"`
		EsquemaCarga         string `json:"esquema_carga"`
		VersionExpediente    uint64 `json:"version_expediente"`
		ExpedienteRef        string `json:"expediente_ref"`
		IncorporacionRef     string `json:"incorporacion_ref"`
		ProcedenciaModeloRef string `json:"procedencia_modelo_ref"`
		CorrelacionRef       string `json:"correlacion_ref"`
		IdempotenciaRef      string `json:"idempotencia_ref"`
		HuellaModeloSHA256   string `json:"huella_modelo_sha256"`
		MapeoRef             string `json:"mapeo_ref"`
		MapeoVersion         uint64 `json:"mapeo_version"`
		ProcedenciaMapeoRef  string `json:"procedencia_mapeo_ref"`
		HuellaMapeoSHA256    string `json:"huella_mapeo_sha256"`
		HuellaCargaSHA256    string `json:"huella_carga_sha256"`
	} `json:"metadatos"`
	Campos []campoFicheroGINPIX `json:"campos"`
}

func TestCodificarEsDeterministaYConservaMetadatosYEstados(t *testing.T) {
	cargaA := cargaGINPIXFicheroPrueba(t, false)
	cargaB := cargaGINPIXFicheroPrueba(t, true)

	contenidoA, err := Codificar(cargaA)
	if err != nil {
		t.Fatalf("codificar carga sintética: %v", err)
	}
	contenidoB, err := Codificar(cargaB)
	if err != nil {
		t.Fatalf("codificar carga sintética reordenada: %v", err)
	}
	if !bytes.Equal(contenidoA, contenidoB) {
		t.Fatal("el fichero depende del orden de entrada")
	}
	suma := sha256.Sum256(contenidoA)
	if got, want := hex.EncodeToString(suma[:]), "8e394cb904195d32b8b211f913641b175030ed0829b5b7b8fc96db86a8fb0d98"; got != want {
		t.Fatalf("vector de fichero divergente: got %s want %s", got, want)
	}

	var fichero ficheroGINPIXPrueba
	if err := json.Unmarshal(contenidoA, &fichero); err != nil {
		t.Fatalf("fichero no es JSON estructurado: %v", err)
	}
	datos := cargaA.Datos()
	if fichero.Esquema != EsquemaFicheroGINPIXV1 ||
		fichero.Version != VersionFormatoFicheroGINPIXV1 ||
		fichero.Metadatos.EsquemaModelo != domain.EsquemaModeloCanonicoGINPIXV1 ||
		fichero.Metadatos.EsquemaMapeo != domain.EsquemaMapeoGINPIXV1 ||
		fichero.Metadatos.EsquemaCarga != domain.EsquemaCargaMapeadaGINPIXV1 ||
		fichero.Metadatos.VersionExpediente != datos.VersionExpediente ||
		fichero.Metadatos.ExpedienteRef != datos.ExpedienteRef ||
		fichero.Metadatos.IncorporacionRef != datos.IncorporacionRef ||
		fichero.Metadatos.ProcedenciaModeloRef != datos.ProcedenciaModeloRef ||
		fichero.Metadatos.CorrelacionRef != datos.CorrelacionRef ||
		fichero.Metadatos.IdempotenciaRef != datos.IdempotenciaRef ||
		fichero.Metadatos.HuellaModeloSHA256 != datos.ModeloHuellaSHA256 ||
		fichero.Metadatos.MapeoRef != datos.MapeoRef ||
		fichero.Metadatos.MapeoVersion != datos.MapeoVersion ||
		fichero.Metadatos.ProcedenciaMapeoRef != datos.ProcedenciaMapeoRef ||
		fichero.Metadatos.HuellaMapeoSHA256 != datos.MapeoHuellaSHA256 ||
		fichero.Metadatos.HuellaCargaSHA256 != datos.HuellaSHA256 {
		t.Fatalf("metadatos incompletos o divergentes: %+v", fichero.Metadatos)
	}

	esperados := []campoFicheroGINPIX{
		{Clave: "destino_a_nulo", Estado: "nulo", Valor: ""},
		{Clave: "destino_b_valor", Estado: "valor", Valor: "SINT-01"},
		{Clave: "destino_m_vacio", Estado: "valor", Valor: ""},
		{Clave: "destino_z_ausente", Estado: "ausente", Valor: ""},
	}
	if len(fichero.Campos) != len(esperados) {
		t.Fatalf("cardinalidad de campos divergente: %d", len(fichero.Campos))
	}
	for indice := range esperados {
		if fichero.Campos[indice] != esperados[indice] {
			t.Fatalf(
				"campo %d divergente: got %+v want %+v",
				indice,
				fichero.Campos[indice],
				esperados[indice],
			)
		}
	}

	var superficie map[string]json.RawMessage
	if err := json.Unmarshal(contenidoA, &superficie); err != nil {
		t.Fatalf("inspeccionar superficie: %v", err)
	}
	if len(superficie) != 4 || superficie["esquema"] == nil ||
		superficie["version"] == nil || superficie["metadatos"] == nil ||
		superficie["campos"] == nil {
		t.Fatalf("superficie inesperada: %v", superficie)
	}
}

func TestCodificarUsaCopiasYNoRetieneEstado(t *testing.T) {
	carga := cargaGINPIXFicheroPrueba(t, false)
	contenido, err := Codificar(carga)
	if err != nil {
		t.Fatalf("codificar carga sintética: %v", err)
	}
	referencia := append([]byte(nil), contenido...)

	snapshot := carga.Datos()
	snapshot.Campos[0].Campo.Valor = "MUTADO"
	contenido[0] = 'X'
	posterior, err := Codificar(carga)
	if err != nil {
		t.Fatalf("recodificar carga sintética: %v", err)
	}
	if !bytes.Equal(referencia, posterior) {
		t.Fatal("el codificador retuvo memoria mutable de entrada o salida")
	}
}

func TestCodificarDeniegaCargaCeroYLimitaAntesDeReservar(t *testing.T) {
	if contenido, err := Codificar(domain.CargaMapeadaGINPIX{}); contenido != nil ||
		!errors.Is(err, ErrCargaGINPIXInvalida) {
		t.Fatalf("carga cero aceptada: %q / %v", contenido, err)
	}

	datos := cargaGINPIXFicheroPrueba(t, false).Datos()
	datos.Campos = make([]domain.CampoMapeadoGINPIX, maximoCamposFicheroGINPIX+1)
	asignaciones := testing.AllocsPerRun(100, func() {
		if _, err := presupuestoCodificacion(datos); !errors.Is(
			err,
			ErrLimiteFicheroGINPIXExcedido,
		) {
			panic("cardinalidad fuera de límite aceptada")
		}
	})
	if asignaciones != 0 {
		t.Fatalf("el límite de cardinalidad reservó memoria: %.2f", asignaciones)
	}
}

func TestCodificarCargaMaximaPermaneceAcotada(t *testing.T) {
	datos := make([]domain.DatoCanonicoGINPIX, maximoCamposFicheroGINPIX)
	reglas := make([]domain.ReglaMapeoGINPIX, maximoCamposFicheroGINPIX)
	for indice := range datos {
		clave := domain.ClaveCatalogo(fmt.Sprintf("campo_%03d", indice))
		valor, err := domain.CampoValorGINPIX(strings.Repeat("<", 480))
		if err != nil {
			t.Fatalf("crear valor límite %d: %v", indice, err)
		}
		datos[indice] = domain.DatoCanonicoGINPIX{Clave: clave, Campo: valor}
		reglas[indice] = domain.ReglaMapeoGINPIX{
			CampoCanonico: clave,
			CampoDestino:  domain.ClaveCatalogo(fmt.Sprintf("destino_%03d", indice)),
			Obligatorio:   true,
		}
	}
	carga := nuevaCargaGINPIXFicheroPrueba(t, datos, reglas)
	presupuesto, err := presupuestoCodificacion(carga.Datos())
	if err != nil {
		t.Fatalf("presupuestar carga válida máxima: %v", err)
	}
	contenido, err := Codificar(carga)
	if err != nil {
		t.Fatalf("codificar carga válida máxima: %v", err)
	}
	if len(contenido) > presupuesto || len(contenido) > MaximoBytesFicheroGINPIX {
		t.Fatalf(
			"salida fuera de cota: bytes=%d presupuesto=%d máximo=%d",
			len(contenido),
			presupuesto,
			MaximoBytesFicheroGINPIX,
		)
	}
	if !json.Valid(contenido) ||
		bytes.Count(contenido, []byte(`\u003c`)) != maximoCamposFicheroGINPIX*480 {
		t.Fatal("la carga límite no conserva una codificación JSON válida")
	}
}

func cargaGINPIXFicheroPrueba(
	t *testing.T,
	invertida bool,
) domain.CargaMapeadaGINPIX {
	t.Helper()
	datos := []domain.DatoCanonicoGINPIX{
		{Clave: "campo_ausente", Campo: domain.CampoAusenteGINPIX()},
		{Clave: "campo_nulo", Campo: domain.CampoNuloGINPIX()},
		{Clave: "campo_vacio", Campo: campoValorGINPIXFicheroPrueba(t, "")},
		{Clave: "campo_valor", Campo: campoValorGINPIXFicheroPrueba(t, "SINT-01")},
	}
	reglas := []domain.ReglaMapeoGINPIX{
		{CampoCanonico: "campo_ausente", CampoDestino: "destino_z_ausente"},
		{CampoCanonico: "campo_nulo", CampoDestino: "destino_a_nulo", PermiteNulo: true},
		{CampoCanonico: "campo_vacio", CampoDestino: "destino_m_vacio", PermiteVacio: true},
		{CampoCanonico: "campo_valor", CampoDestino: "destino_b_valor", Obligatorio: true},
	}
	if invertida {
		for izquierda, derecha := 0, len(datos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
			datos[izquierda], datos[derecha] = datos[derecha], datos[izquierda]
			reglas[izquierda], reglas[derecha] = reglas[derecha], reglas[izquierda]
		}
	}
	return nuevaCargaGINPIXFicheroPrueba(t, datos, reglas)
}

func nuevaCargaGINPIXFicheroPrueba(
	t *testing.T,
	datos []domain.DatoCanonicoGINPIX,
	reglas []domain.ReglaMapeoGINPIX,
) domain.CargaMapeadaGINPIX {
	t.Helper()
	modelo, err := domain.NuevoModeloCanonicoGINPIX(
		domain.BorradorModeloCanonicoGINPIX{
			Esquema:           domain.EsquemaModeloCanonicoGINPIXV1,
			VersionExpediente: 11,
			ExpedienteRef:     "expediente_sintetico_ginpix_fichero_01",
			IncorporacionRef:  "incorporacion_sintetica_ginpix_fichero_01",
			ProcedenciaRef:    "procedencia_sintetica_modelo_fichero_01",
			CorrelacionRef:    "correlacion_sintetica_ginpix_fichero_01",
			IdempotenciaRef:   "idempotencia_sintetica_ginpix_fichero_01",
			Datos:             datos,
		},
	)
	if err != nil {
		t.Fatalf("crear modelo sintético: %v", err)
	}
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(
		domain.BorradorMapeoVersionadoGINPIX{
			Esquema:        domain.EsquemaMapeoGINPIXV1,
			Referencia:     "mapeo_sintetico_ginpix_fichero_01",
			Version:        5,
			ProcedenciaRef: "procedencia_sintetica_mapeo_fichero_01",
			Reglas:         reglas,
		},
	)
	if err != nil {
		t.Fatalf("crear mapeo sintético: %v", err)
	}
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatalf("crear carga sintética: %v", err)
	}
	return carga
}

func campoValorGINPIXFicheroPrueba(
	t *testing.T,
	valor string,
) domain.CampoGINPIX {
	t.Helper()
	campo, err := domain.CampoValorGINPIX(valor)
	if err != nil {
		t.Fatalf("crear campo sintético: %v", err)
	}
	return campo
}
