package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCargarDefinicionCircuitoRRHHExigePublicacionIntacta(t *testing.T) {
	definicion, err := domain.NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:prueba", 2, "solicitud",
		[]domain.TransicionCircuitoRRHH{{
			Clave: "peticion_firmada", Tipo: domain.HitoPeticionFirmada,
			Origen: "solicitud", Destino: "peticion_firmada",
			RequiereDocumento: true, RequiereFirma: true,
			PerfilClave:      "tecnico_solicitante",
			FirmasRequeridas: []domain.ClaveCatalogo{"tecnico_solicitante", "delegacion"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(definicion)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "circuito.json")
	escribir := func(datos []byte) {
		t.Helper()
		if err := os.WriteFile(ruta, datos, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(contenido)
	leida, err := cargarDefinicionCircuitoRRHH(ruta)
	if err != nil || leida.Flujo != definicion.Flujo || len(leida.Transiciones) != 1 {
		t.Fatalf("publicacion valida: definicion=%+v error=%v", leida.Flujo, err)
	}

	alterada := strings.Replace(string(contenido), "tecnico_solicitante", "otro_perfil", 1)
	escribir([]byte(alterada))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto contenido distinto de la huella publicada")
	}
	escribir(append(contenido, []byte(` {"fuente":"cliente"}`)...))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto un segundo documento JSON")
	}
	escribir([]byte(strings.Replace(string(contenido), `"transiciones":`, `"fuente":"cliente","transiciones":`, 1)))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto un campo ajeno a la definicion")
	}
}

func TestCatalogoRRHHV2PublicaVistosBuenosAlternativos(t *testing.T) {
	d, err := cargarDefinicionCircuitoRRHH("../../../config/contratacion_temporal_circuito_rrhh_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if d.Flujo.HuellaSHA256 != "1721c3a66576b21163b590602589f1627095bd6b6bfa37c79862af62775146e2" {
		t.Fatalf("huella v2 inesperada: %s", d.Flujo.HuellaSHA256)
	}
	for _, base := range []string{
		"contratacion_temporal.circuito.resolucion_firmada",
		"contratacion_temporal.circuito.resolucion_sin_cambio",
	} {
		jefatura, direccion := false, false
		for _, tr := range d.Transiciones {
			switch tr.Clave {
			case domain.ClaveCatalogo(base):
				jefatura = len(tr.FirmasRequeridas) == 2 && tr.FirmasRequeridas[0] == "jefatura_servicio_rrhh" && tr.FirmasRequeridas[1] == "diputacion_delegada_rrhh"
			case domain.ClaveCatalogo(base + "_visto_bueno_direccion"):
				direccion = len(tr.FirmasRequeridas) == 2 && tr.FirmasRequeridas[0] == "direccion_rrhh" && tr.FirmasRequeridas[1] == "diputacion_delegada_rrhh"
			}
		}
		if !jefatura || !direccion {
			t.Fatalf("faltan alternativas ordenadas en %s", base)
		}
	}
}

// F1-98: con el circuito v2 configurado (orden `firma` de la copia), los
// expedientes nuevos nacían con circuito y el análisis respondía 503 porque
// ninguna fuente acredita sus evidencias. El alta no lo adjunta; la
// presentación conjunta pinta el flujo del alta y conserva la del circuito.
func TestAltaNoAdjuntaCircuitoRRHHSinFuentesDeEvidencias(t *testing.T) {
	d, err := cargarDefinicionCircuitoRRHH("../../../config/contratacion_temporal_circuito_rrhh_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	flujo := flujoAltaContratacionTemporalDesarrollo(referenciaOrganizativaCTMontajePrueba.unidadRef)
	if flujo.Validar() != nil || flujo.DefinicionCircuito != nil || flujo.Flujo == d.Flujo ||
		flujo.FaseInicial != domain.ClaveFase("solicitud") {
		t.Fatalf("el alta adjunta el circuito sin fuentes de evidencias: %#v", flujo)
	}
	legado, err := LeerLectorFlujoVisualRRHH(strings.NewReader(config.PresentacionFlujoRRHHDesarrollo()))
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := LeerLectorFlujoVisualRRHH(strings.NewReader(config.PresentacionCircuitoRRHH()))
	if err != nil || circuito.origen != d.Flujo {
		t.Fatalf("presentación del circuito: %v", err)
	}
	conjunto, err := NuevoLectorFlujosVisualesRRHH(legado, circuito)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := conjunto.Resolver(t.Context(), flujo.Flujo, flujo.FaseInicial); err != nil || len(p.Fases) != 8 {
		t.Fatalf("el expediente nuevo no se pinta con las ocho fases: %d, %v", len(p.Fases), err)
	}
	if _, err := conjunto.Resolver(t.Context(), d.Flujo, d.EstadoInicial); err != nil {
		t.Fatalf("se perdió la presentación del circuito: %v", err)
	}
}
