package plantillascatalogo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

var instanteBorradorPrueba = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

type consultaDetalleBorradorPrueba struct {
	detalle  ports.DetalleExpedienteRRHH
	llamadas int
}

func (c *consultaDetalleBorradorPrueba) Consultar(_ context.Context, _ ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	c.llamadas++
	return c.detalle, nil
}

type proveedorBorradorPrueba struct {
	plantillas  *informejuridico.PlantillasBorrador
	llamadas    int
	material    plantillasapp.SolicitudDocumental
	err         error
	procedencia string
}

const reciboCatalogoPrueba = "recibo:12345678-1234-1234-1234-123456789abc"

func (p *proveedorBorradorPrueba) ObtenerPlantillasDocumento(_ context.Context, s plantillasapp.SolicitudDocumental, _ time.Time) (*informejuridico.PlantillasBorrador, string, error) {
	p.llamadas++
	p.material = s
	if p.err != nil {
		return nil, "", p.err
	}
	if p.procedencia != "" {
		return p.plantillas, p.procedencia, nil
	}
	return p.plantillas, reciboCatalogoPrueba, nil
}

func detalleBorradorPrueba() ports.DetalleExpedienteRRHH {
	creado := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	ultimo := creado.Add(3 * time.Hour)
	return ports.DetalleExpedienteRRHH{
		Resumen:   ports.ResumenExpedienteRRHH{ExpedienteRef: "expediente:ct:0001", OrganizacionRef: "organizacion:rrhh:secreta", NumeroVisible: "2026/CT-0001", Version: 3, FlujoRef: "flujo:ct:ordinario", FlujoVersion: 2, FlujoHuella: strings.Repeat("a", 64), FaseClave: "analisis", EstadoClave: domain.EstadoEnCurso, CentroRef: "centro:dipgra:001", CategoriaRef: "categoria:auxiliar:001", CreadoEn: creado, ActualizadoEn: ultimo},
		Solicitud: ports.SolicitudOperativaRRHH{GrupoSubgrupo: "C2", MotivoClave: "sustitucion", PeriodoInicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), PeriodoFin: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		Hitos: []ports.HitoExpedienteRRHH{
			{Secuencia: 1, VersionExpediente: 1, AccionClave: "alta", RealizadaEn: creado, FaseDestino: "solicitud", EstadoOrigen: domain.EstadoPendiente, EstadoDestino: domain.EstadoEnCurso},
			{Secuencia: 2, VersionExpediente: 2, AccionClave: "analizar", RealizadaEn: creado.Add(time.Hour), FaseOrigen: "solicitud", FaseDestino: "analisis", EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
			{Secuencia: 3, VersionExpediente: 3, AccionClave: "actualizar", RealizadaEn: ultimo, FaseOrigen: "analisis", FaseDestino: "analisis", EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
		},
	}
}

func plantillasBorradorPrueba(t *testing.T) *informejuridico.PlantillasBorrador {
	t.Helper()
	ruta := filepath.Join("..", "..", "..", "..", "..", "..", "data", "demo", "plantillas", "ct_plantillas_documentos.ejemplo.demo.json")
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	c, err := consulta.ObtenerCatalogo(context.Background(), informejuridico.CatalogoPlantillasBorradorID, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, err := informejuridico.NuevasPlantillasBorrador(c, instanteBorradorPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func peticionBorradorPrueba(ruta, accept, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", accept)
	return r
}

func TestBorradorGenericoFijaCatalogoYConsultaDetalleAntesDeDescargar(t *testing.T) {
	p := plantillasBorradorPrueba(t)
	c := &consultaDetalleBorradorPrueba{detalle: detalleBorradorPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p}
	llamadas := 0
	g := GeneradorConInstantaneaFunc(func(_ context.Context, recibido *informejuridico.PlantillasBorrador, formato string, tipo ports.TipoBorradorRRHH, _ ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error) {
		llamadas++
		if recibido != p || c.llamadas != 1 || fuente.llamadas != 1 || formato != "pdf" || tipo != "informe_definitivo" {
			t.Fatal("render sin lectura autorizada o instantánea distinta")
		}
		plantilla, _ := p.Plantilla(tipo)
		return DocumentoCatalogado{Contenido: []byte("%PDF-1.4\n%%EOF\n"), Formato: formato, Tipo: tipo, PlantillaRef: plantilla.Referencia, CatalogoRef: p.Referencia(), CatalogoHuellaSHA256: p.Huella()}, nil
	})
	h, err := NuevoManejadorBorradores(c, fuente, g, func() time.Time { return instanteBorradorPrueba })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", `{"expediente_ref":"expediente:ct:0001","version_observada":3,"tipo":"informe_definitivo","formato":"pdf"}`))
	if w.Code != 200 || llamadas != 1 || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || w.Header().Get("X-VEC-Catalogo-Huella-SHA256") != p.Huella() || w.Header().Get("X-VEC-Documento-SHA256") == "" || w.Header().Get("X-VEC-Plantilla-Procedencia-Ref") != reciboCatalogoPrueba || w.Header().Get("Content-Disposition") != `attachment; filename="informe_definitivo-borrador.pdf"` {
		t.Fatalf("descarga: %d, %q", w.Code, w.Body.String())
	}
}

func TestListaBorradoresUsaConsultaExpedienteSinGobiernoCatalogo(t *testing.T) {
	p := plantillasBorradorPrueba(t)
	c := &consultaDetalleBorradorPrueba{detalle: detalleBorradorPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p}
	g := GeneradorConInstantaneaFunc(func(context.Context, *informejuridico.PlantillasBorrador, string, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error) {
		t.Fatal("la lista no debe generar")
		return DocumentoCatalogado{}, nil
	})
	h, _ := NuevoManejadorBorradores(c, fuente, g, func() time.Time { return instanteBorradorPrueba })
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradoresDisponibles, "application/json", `{"expediente_ref":"expediente:ct:0001","version_observada":3}`))
	var z listaDisponibles
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &z) != nil || z.Esquema != EsquemaBorradoresDisponibles || z.CatalogoHuellaSHA256 != p.Huella() || z.ProcedenciaRef != reciboCatalogoPrueba || len(z.Tipos) == 0 || c.llamadas != 1 || fuente.llamadas != 1 {
		t.Fatalf("lista: %d %q", w.Code, w.Body.String())
	}
}

func TestBorradorGenericoNoUsaTipoFueraCatalogo(t *testing.T) {
	p := plantillasBorradorPrueba(t)
	c := &consultaDetalleBorradorPrueba{detalle: detalleBorradorPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p}
	g := GeneradorConInstantaneaFunc(func(context.Context, *informejuridico.PlantillasBorrador, string, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error) {
		t.Fatal("render de tipo no publicado")
		return DocumentoCatalogado{}, nil
	})
	h, _ := NuevoManejadorBorradores(c, fuente, g, func() time.Time { return instanteBorradorPrueba })
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", `{"expediente_ref":"expediente:ct:0001","version_observada":3,"tipo":"documento_inexistente","formato":"pdf"}`))
	if w.Code != 409 || c.llamadas != 1 || fuente.llamadas != 1 {
		t.Fatalf("tipo no publicado: %d", w.Code)
	}
}

// La lista marca qué documentos se pueden preparar ya: antes del nombramiento
// ninguno; con la propuesta de formalización registrada, los de esa fase.
func TestListaBorradoresIndicaDisponibilidadSegunFase(t *testing.T) {
	p := plantillasBorradorPrueba(t)
	listar := func(detalle ports.DetalleExpedienteRRHH) map[string]bool {
		t.Helper()
		g := GeneradorConInstantaneaFunc(func(context.Context, *informejuridico.PlantillasBorrador, string, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (DocumentoCatalogado, error) {
			t.Fatal("la lista no debe generar")
			return DocumentoCatalogado{}, nil
		})
		h, _ := NuevoManejadorBorradores(&consultaDetalleBorradorPrueba{detalle: detalle}, &proveedorBorradorPrueba{plantillas: p}, g, func() time.Time { return instanteBorradorPrueba })
		w := httptest.NewRecorder()
		cuerpo := `{"expediente_ref":"` + detalle.Resumen.ExpedienteRef + `","version_observada":` + strconv.FormatUint(detalle.Resumen.Version, 10) + `}`
		h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradoresDisponibles, "application/json", cuerpo))
		var z listaDisponibles
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &z) != nil || len(z.Tipos) == 0 {
			t.Fatalf("lista: %d %q", w.Code, w.Body.String())
		}
		disponibles := map[string]bool{}
		for _, tipo := range z.Tipos {
			disponibles[tipo.Clave] = tipo.Disponible
		}
		return disponibles
	}
	for clave, disponible := range listar(detalleBorradorPrueba()) {
		if disponible {
			t.Fatalf("%s disponible en fase de análisis", clave)
		}
	}
	if enNombramiento := listar(detalleGeneradorPrueba()); !enNombramiento["informe_definitivo"] || !enNombramiento["resolucion"] {
		t.Fatalf("documentos de nombramiento no disponibles: %v", enNombramiento)
	}
}
