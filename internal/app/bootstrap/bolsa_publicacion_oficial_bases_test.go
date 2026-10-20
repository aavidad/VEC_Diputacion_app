package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// La fuente gobernada lleva el canon de los tipos Go de dominio y la forma
// exacta que exige la composición (sin campos, auditada, garantía alta).
func TestPublicacionOficialFuenteGobernadaExacta(t *testing.T) {
	contenido, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "principal", "fuentes", "bolsa_publicacion_oficial_bases_rrhh_v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		Referencia string                                     `json:"referencia"`
		Version    int                                        `json:"version"`
		Huella     string                                     `json:"huella_sha256"`
		Canon      string                                     `json:"entradas_canon"`
		Entradas   []dominiovec.EntradaAccionAdministracionV1 `json:"entradas"`
	}
	if err := json.Unmarshal(contenido, &fuente); err != nil {
		t.Fatal(err)
	}
	type proyeccion struct {
		Referencia        string                  `json:"referencia"`
		Version           int                     `json:"version"`
		FuenteRef         string                  `json:"fuente_ref"`
		FuenteVersion     int                     `json:"fuente_version"`
		Concesion         dominiovec.ConcesionRol `json:"concesion"`
		DimensionesAmbito []string                `json:"dimensiones_ambito"`
		ClaseControl      string                  `json:"clase_control"`
		VigenteDesde      time.Time               `json:"vigente_desde"`
		VigenteHasta      time.Time               `json:"vigente_hasta,omitempty"`
	}
	if len(fuente.Entradas) != 1 {
		t.Fatal("la fuente debe tener una sola entrada")
	}
	e := fuente.Entradas[0]
	canon, err := json.Marshal([]proyeccion{{e.Referencia, e.Version, e.FuenteRef, e.FuenteVersion, e.Concesion,
		e.DimensionesAmbito, e.ClaseControl, e.VigenteDesde, e.VigenteHasta}})
	huella := sha256.Sum256(canon)
	if err != nil || string(canon) != fuente.Canon || hex.EncodeToString(huella[:]) != fuente.Huella ||
		e.Validar() != nil || e.FuenteRef != fuente.Referencia || e.FuenteHuellaSHA256 != fuente.Huella ||
		!concesionPublicacionOficialBasesExacta(e.Concesion) || e.VigenteHasta.Sub(e.VigenteDesde) < 60*24*time.Hour {
		t.Fatal("fuente de la publicación oficial incoherente")
	}
}

func TestPublicacionOficialConviveConLaConfirmacionDirecta(t *testing.T) {
	confirmacion := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarAceptacionDirecta,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	publicacion := concesionesFuenteBolsaPrueba(t, "fuentes/bolsa_publicacion_oficial_bases_rrhh_v1.json",
		puertosbolsa.AccionDeclararPublicacionOficialBases)[0]
	if !concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, publicacion}) {
		t.Fatal("la concesión exacta de publicación fue rechazada")
	}
	for nombre, alterar := range map[string]func(*dominiovec.ConcesionRol){
		"campos":     func(c *dominiovec.ConcesionRol) { c.CamposPermitidos = []string{"titulo"} },
		"finalidad":  func(c *dominiovec.ConcesionRol) { c.Finalidades = []string{"otra"} },
		"garantia":   func(c *dominiovec.ConcesionRol) { c.GarantiaMinima = dominiovec.AuthAssuranceLow },
		"recurso":    func(c *dominiovec.ConcesionRol) { c.TipoRecurso = "convocatoria" },
		"obligacion": func(c *dominiovec.ConcesionRol) { c.Obligaciones = nil },
	} {
		alterada := publicacion
		alterada.Finalidades = append([]string(nil), publicacion.Finalidades...)
		alterar(&alterada)
		if concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, alterada}) {
			t.Fatalf("%s: concesión alterada adoptada", nombre)
		}
	}
	if concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, publicacion, publicacion}) {
		t.Fatal("concesión duplicada adoptada")
	}
}

func TestPublicacionOficialMotivoPropio(t *testing.T) {
	if !motivoBorradorLlamamientoCorresponde(puertosbolsa.AccionDeclararPublicacionOficialBases, motivoPublicacionOficialBases()) ||
		motivoBorradorLlamamientoCorresponde(puertosbolsa.AccionDeclararPublicacionOficialBases, motivoComprobacionOfrecimiento()) ||
		motivoBorradorLlamamientoCorresponde(puertosbolsa.AccionComprobacionOfrecimiento, motivoPublicacionOficialBases()) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivoPublicacionOficialBases()) {
		t.Fatal("motivo de la publicación oficial no exclusivo")
	}
}

// Sin el emisor de su audiencia (núcleo sin consumidor) la ruta se monta pero
// nunca acepta: ni pide decisión ni toca la base de datos.
func TestPublicacionOficialSinEmisorDeniegaCerrado(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nadie@127.0.0.1:1/nada?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ruta := nuevaRutaPublicacionOficialBases(pool, &preparadorBorradorLlamamientoDesarrollo{}, &emisorBorradorLlamamientoDesarrollo{}, nil, nil, "vec-rrhh", time.Now)
	if ruta.Ruta != bolsahttp.RutaPublicacionesOficialesBases || ruta.Manejador == nil {
		t.Fatal("ruta no montada")
	}
	cuerpo := `{"convocatoria_ref":"cv1_` + strings.Repeat("a", 64) + `_v1","huella_version_sha256":"` + strings.Repeat("c", 64) +
		`","medio":"boe","boletin_numero":1,"anuncio_ref":"BOE-A-1","fecha_publicacion":"2026-10-15","direccion":null,"sustituye_ref":null}`
	r := httptest.NewRequest(http.MethodPost, bolsahttp.RutaPublicacionesOficialesBases, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Idempotency-Key", "f195-cerrada-0001")
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "publicacion_oficial_no_habilitada") {
		t.Fatalf("declaración sin núcleo: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, bolsahttp.RutaPublicacionesOficialesBases, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("lectura sin fuente ni auditoría: %d", w.Code)
	}
	if _, _, _, err := (&emisorBorradorLlamamientoDesarrollo{}).EmitirMaterialAutorizacionAtestadaV3(context.Background(),
		dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ResultadoContextoActorRegistradoV2{}); err == nil {
		t.Fatal("emisor sin audiencia emitió material")
	}
}
