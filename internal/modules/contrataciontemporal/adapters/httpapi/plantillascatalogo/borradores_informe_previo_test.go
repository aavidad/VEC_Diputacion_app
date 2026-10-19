package plantillascatalogo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

// plantillasPublicadasPrueba carga un catálogo versionado del repositorio con
// el mismo lector que el arranque.
func plantillasPublicadasPrueba(t *testing.T, nombre string, version int) *informejuridico.PlantillasBorrador {
	t.Helper()
	ruta := filepath.Join("..", "..", "..", "..", "..", "..", "data", "catalogos", "contratacion-temporal", nombre)
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable(ruta)
	if err != nil {
		t.Fatal(err)
	}
	c, err := consulta.ObtenerCatalogo(context.Background(), informejuridico.CatalogoPlantillasBorradorID, version)
	if err != nil {
		t.Fatal(err)
	}
	p, err := informejuridico.NuevasPlantillasBorrador(c, time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// detalleInformeJuridicoPrueba es el expediente del paso 7 del caso 4.0: v5,
// informe_juridico/en_curso y el informe jurídico como última actuación.
func detalleInformeJuridicoPrueba() ports.DetalleExpedienteRRHH {
	d := detalleGeneradorPrueba()
	d.Hitos = d.Hitos[:5]
	informe := &d.Hitos[4]
	informe.AccionClave = domain.AccionEmitirInformeJuridico
	informe.FaseDestino = domain.FaseInformeJuridico
	d.Resumen.Version = 5
	d.Resumen.FaseClave = domain.FaseInformeJuridico
	d.Resumen.ActualizadoEn = informe.RealizadaEn
	return d
}

func listarInformePrevio(t *testing.T, h *ManejadorBorradores, version string) (int, *tipoDisponible) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradoresDisponibles, "application/json",
		`{"expediente_ref":"expediente:ct:sintetico:informe","version_observada":`+version+`}`))
	var lista listaDisponibles
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &lista) != nil {
		t.Fatalf("lista: %d %q", w.Code, w.Body.String())
	}
	for i := range lista.Tipos {
		if lista.Tipos[i].Clave == string(ports.BorradorInformeDefinitivo) {
			return len(lista.Tipos), &lista.Tipos[i]
		}
	}
	return len(lista.Tipos), nil
}

// Con las plantillas v4, el informe definitivo que se firma en el paso 7b se
// prepara y descarga en v5, tras el informe jurídico, con su recibo de catálogo.
func TestInformeDefinitivoDisponibleTrasInformeJuridicoConPlantillasV4(t *testing.T) {
	p := plantillasPublicadasPrueba(t, "plantillas_documentos.v4.json", 4)
	c := &consultaDetalleBorradorPrueba{detalle: detalleInformeJuridicoPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p}
	h, err := NuevoManejadorBorradores(c, fuente, GeneradorBorradoresCatalogo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}},
		func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if _, informe := listarInformePrevio(t, h, "5"); informe == nil || !informe.Disponible {
		t.Fatalf("informe definitivo no disponible en v5: %+v", informe)
	}
	base := `{"expediente_ref":"expediente:ct:sintetico:informe","version_observada":5,"tipo":"informe_definitivo","formato":"`
	for _, formato := range []struct{ nombre, accept, firma string }{{"pdf", "application/pdf", "%PDF-"}, {"docx", MIMEDOCX, "PK\x03\x04"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, formato.accept, base+formato.nombre+`"}`))
		if w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte(formato.firma)) ||
			w.Header().Get("X-VEC-Catalogo-Ref") != p.Referencia() ||
			w.Header().Get("X-VEC-Plantilla-Procedencia-Ref") != reciboCatalogoPrueba ||
			w.Header().Get("X-VEC-Documento-SHA256") == "" {
			t.Fatalf("%s: %d %q", formato.nombre, w.Code, w.Body.String())
		}
	}
	if fuente.material.VersionObservada != 5 || fuente.material.Tipo != "informe_definitivo" {
		t.Fatal("la descarga no quedó ligada a la versión consultada")
	}
}

// Fiscalizado el expediente, el informe previo deja de ser el vigente: la
// lista lo muestra sin preparar y la descarga responde 409.
func TestInformeDefinitivoV4NoSePreparaSinInformeJuridicoVigente(t *testing.T) {
	p := plantillasPublicadasPrueba(t, "plantillas_documentos.v4.json", 4)
	d := detalleInformeJuridicoPrueba()
	ultimo := d.Hitos[4]
	d.Hitos = append(d.Hitos, ports.HitoExpedienteRRHH{
		Secuencia: 6, VersionExpediente: 6, AccionClave: "actuacion_sintetica", RealizadaEn: ultimo.RealizadaEn.Add(time.Minute),
		FaseOrigen: ultimo.FaseDestino, FaseDestino: domain.FaseFiscalizacion, EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso,
	})
	d.Resumen.Version, d.Resumen.FaseClave, d.Resumen.ActualizadoEn = 6, domain.FaseFiscalizacion, ultimo.RealizadaEn.Add(time.Minute)
	h, _ := NuevoManejadorBorradores(&consultaDetalleBorradorPrueba{detalle: d}, &proveedorBorradorPrueba{plantillas: p},
		GeneradorBorradoresCatalogo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}}, func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) })
	if _, informe := listarInformePrevio(t, h, "6"); informe == nil || informe.Disponible {
		t.Fatalf("informe sin emisión vigente ofrecido: %+v", informe)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf",
		`{"expediente_ref":"expediente:ct:sintetico:informe","version_observada":6,"tipo":"informe_definitivo","formato":"pdf"}`))
	if w.Code != 409 || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("descarga sin informe vigente: %d", w.Code)
	}
}

// Con las plantillas v3 el informe definitivo sigue siendo el de la propuesta
// de formalización (v7): en v5 aparece, pero sin preparar.
func TestInformeDefinitivoV3EnV5SigueSinPreparar(t *testing.T) {
	p := plantillasPublicadasPrueba(t, "plantillas_documentos.v3.json", 3)
	h, _ := NuevoManejadorBorradores(&consultaDetalleBorradorPrueba{detalle: detalleInformeJuridicoPrueba()}, &proveedorBorradorPrueba{plantillas: p},
		GeneradorBorradoresCatalogo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}}, func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) })
	if _, informe := listarInformePrevio(t, h, "5"); informe == nil || informe.Disponible {
		t.Fatalf("v3 en v5: %+v", informe)
	}
}
