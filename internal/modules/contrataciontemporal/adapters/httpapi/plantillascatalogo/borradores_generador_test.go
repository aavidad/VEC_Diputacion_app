package plantillascatalogo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	pgplantillas "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func detalleGeneradorPrueba() ports.DetalleExpedienteRRHH {
	inicio := time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)
	periodo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	d := ports.DetalleExpedienteRRHH{
		Resumen: ports.ResumenExpedienteRRHH{
			ExpedienteRef: "expediente:ct:sintetico:informe", OrganizacionRef: "organizacion:desarrollo:dipgra",
			NumeroVisible: "2026/CT-0001", Version: 7, FlujoRef: "flujo:ct:sintetico", FlujoVersion: 1,
			FlujoHuella: strings.Repeat("a", 64), FaseClave: "nombramiento", EstadoClave: domain.EstadoEnCurso,
			CentroRef: "centro:sintetico:001", CategoriaRef: "categoria:sintetica:c2", ModalidadClave: "sustitucion",
			UnidadRef: "unidad:sintetica:rrhh", CreadoEn: inicio, ActualizadoEn: inicio.Add(6 * time.Minute),
		},
		Solicitud: ports.SolicitudOperativaRRHH{GrupoSubgrupo: "C2", MotivoClave: "sustitucion", PeriodoInicio: periodo, PeriodoFin: periodo.AddDate(0, 3, 0)},
		Analisis: &ports.AnalisisOperativoRRHH{ModalidadClave: "sustitucion", CategoriaRef: "categoria:sintetica:c2", CausaClave: "necesidad_temporal", PeriodoInicio: periodo,
			PeriodoFin: periodo.AddDate(0, 3, 0), PorcentajeJornada: 10000, ResultadoRC: domain.RCValidada},
		Cobertura: &ports.CoberturaOperativaRRHH{ViaClave: "bolsa_vigente", DecisionGobernada: true, Comprobaciones: []ports.ComprobacionOperativaRRHH{{
			Clave: "existe_bolsa_vigente", Resultado: domain.ComprobacionAfirmativa,
		}}},
		Asignacion: &ports.AsignacionOperativaRRHH{UnidadRef: "unidad:sintetica:rrhh", AsignadaEn: inicio},
	}
	for i := uint64(1); i <= 7; i++ {
		h := ports.HitoExpedienteRRHH{Secuencia: i, VersionExpediente: i, AccionClave: "actuacion_sintetica", RealizadaEn: inicio.Add(time.Duration(i-1) * time.Minute),
			FaseOrigen: "fiscalizacion", FaseDestino: "fiscalizacion", EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}
		if i == 1 {
			h.FaseOrigen = ""
			h.EstadoOrigen = domain.EstadoPendiente
		}
		if i == 7 {
			h.FaseDestino = "nombramiento"
			h.AccionClave = "registrar_propuesta_formalizacion"
		}
		d.Hitos = append(d.Hitos, h)
	}
	return d
}

func plantillasTipoNuevoPrueba(t *testing.T) *informejuridico.PlantillasBorrador {
	t.Helper()
	ruta := filepath.Join("..", "..", "..", "..", "..", "..", "data", "demo", "plantillas", "ct_plantillas_documentos.ejemplo.demo.json")
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	primera, err := consulta.ObtenerCatalogo(context.Background(), informejuridico.CatalogoPlantillasBorradorID, 1)
	if err != nil {
		t.Fatal(err)
	}
	fecha := instanteBorradorPrueba
	segunda, err := primera.NuevaVersion(2, "configurador:rrhh:001", "fuente:rrhh:plantillas", "Añadir tipo", fecha)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err = segunda.ActualizarBorrador(segunda.Revision, "configurador:rrhh:001", segunda.Nombre, segunda.Descripcion,
		segunda.FuenteRef, "Añadir nuevo tipo", append(segunda.Entradas, vecdomain.EntradaCatalogoConfigurable{
			Clave: "certificacion_servicio", Etiqueta: "Certificación de servicio", Orden: 11, VigenteDesde: fecha,
			Atributos: map[string]string{
				"titulo":     "Certificación de servicio — borrador",
				"parrafo.01": "BORRADOR SIN FIRMA NI VALIDEZ ADMINISTRATIVA.",
				"parrafo.02": "Expediente {{numero_expediente}}. Plantilla {{plantilla_ref}}.",
			},
		}), fecha.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	segunda, err = segunda.Publicar("revisor:rrhh:002", "aprobacion:rrhh:plantillas", "Ejercicio sintético", fecha.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	p, err := informejuridico.NuevasPlantillasBorrador(segunda, fecha.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBorradoresCatalogoTipoNuevoListaYPDFDOCXReales(t *testing.T) {
	p := plantillasTipoNuevoPrueba(t)
	c := &consultaDetalleBorradorPrueba{detalle: detalleGeneradorPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p}
	g := GeneradorBorradoresCatalogo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}}
	h, err := NuevoManejadorBorradores(c, fuente, g, func() time.Time { return instanteBorradorPrueba.Add(3 * time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	base := `{"expediente_ref":"expediente:ct:sintetico:informe","version_observada":7`
	lista := httptest.NewRecorder()
	h.ServeHTTP(lista, peticionBorradorPrueba(RutaBorradoresDisponibles, "application/json", base+`}`))
	var disponibles listaDisponibles
	if lista.Code != 200 || json.Unmarshal(lista.Body.Bytes(), &disponibles) != nil || disponibles.CatalogoRef != p.Referencia() || disponibles.CatalogoHuellaSHA256 != p.Huella() || disponibles.ProcedenciaRef != reciboCatalogoPrueba {
		t.Fatalf("lista: %d %q", lista.Code, lista.Body.String())
	}
	encontrado := false
	for _, tipo := range disponibles.Tipos {
		if tipo.Clave == "certificacion_servicio" && len(tipo.Formatos) == 2 && tipo.Formatos[0] == "pdf" && tipo.Formatos[1] == "docx" {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatal("tipo nuevo ausente de la lista publicada")
	}
	for _, formato := range []struct{ nombre, accept, firma string }{{"pdf", "application/pdf", "%PDF-"}, {"docx", MIMEDOCX, "PK\x03\x04"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, formato.accept, base+`,"tipo":"certificacion_servicio","formato":"`+formato.nombre+`"}`))
		suma := sha256.Sum256(w.Body.Bytes())
		if w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte(formato.firma)) ||
			w.Header().Get("X-VEC-Catalogo-Ref") != p.Referencia() || w.Header().Get("X-VEC-Catalogo-Huella-SHA256") != p.Huella() ||
			w.Header().Get("X-VEC-Plantilla-Procedencia-Ref") != reciboCatalogoPrueba ||
			w.Header().Get("X-VEC-Documento-SHA256") != hex.EncodeToString(suma[:]) {
			t.Fatalf("%s: %d, %q", formato.nombre, w.Code, w.Body.String())
		}
	}
	if c.llamadas != 3 || fuente.llamadas != 3 || fuente.material.OrganizacionRef != "organizacion:desarrollo:dipgra" ||
		fuente.material.ClaseAmbito != "organizacion" || fuente.material.AmbitoRef != fuente.material.OrganizacionRef ||
		fuente.material.VersionObservada != 7 || fuente.material.Tipo != "certificacion_servicio" {
		t.Fatal("la descarga no quedó ligada al expediente y versión consultados")
	}
}

func TestBorradoresCatalogoRevocacionYVersionDivergenteNoGeneran(t *testing.T) {
	p := plantillasTipoNuevoPrueba(t)
	c := &consultaDetalleBorradorPrueba{detalle: detalleGeneradorPrueba()}
	fuente := &proveedorBorradorPrueba{plantillas: p, err: pgplantillas.ErrDenegado}
	h, _ := NuevoManejadorBorradores(c, fuente, GeneradorBorradoresCatalogo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}},
		func() time.Time { return instanteBorradorPrueba.Add(3 * time.Minute) })
	cuerpo := `{"expediente_ref":"expediente:ct:sintetico:informe","version_observada":7,"tipo":"certificacion_servicio","formato":"pdf"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", cuerpo))
	if w.Code != 403 || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || fuente.llamadas != 1 {
		t.Fatalf("revocación: %d, %q", w.Code, w.Body.String())
	}
	fuente.err = errors.New("catálogo revocado")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", cuerpo))
	if w.Code != 503 || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("catálogo ausente: %d", w.Code)
	}
	fuente.err = nil
	fuente.procedencia = "recibo:invalido\r\nX-Inyectado: si"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", cuerpo))
	if w.Code != 503 || w.Header().Get("X-Inyectado") != "" || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("recibo inválido: %d", w.Code)
	}
	fuente.procedencia = ""
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionBorradorPrueba(RutaBorradores, "application/pdf", strings.Replace(cuerpo, `"version_observada":7`, `"version_observada":6`, 1)))
	if w.Code != 503 || fuente.llamadas != 3 || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("versión divergente: %d", w.Code)
	}
}
