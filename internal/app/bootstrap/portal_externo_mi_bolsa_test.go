package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPreparacionPerfilExteriorExigeRolPropioYPreimagenCAS(t *testing.T) {
	identidad := &identidadCandidatoBolsaDesarrollo{
		personaRef:   "per_candidato_sintetico_1234567890123456",
		perfilRef:    "prf_candidato_sintetico_1234567890123456",
		candidatoRef: "can_candidato_sintetico_1234567890123456",
	}
	semilla, err := semillaMiBolsaPortalExterno(identidad, time.Now().UTC(), "areaPersonal.miBolsa.rolPortal")
	if err != nil || semilla.Validar() != nil || semilla.VersionRol.RolID != rolPortalMiBolsaDesarrollo {
		t.Fatalf("semilla portal exterior: %v", err)
	}
	rol, err := prepararRolMiBolsaPortalExterno(semilla, 0, "")
	if err != nil || rol.RevisionEsperada != 0 || rol.ControlHuellaEsperada != nil {
		t.Fatalf("rol inicial sin preimagen: %v", err)
	}
	var publicado dominiovec.VersionRol
	if json.Unmarshal(rol.RolDocumento, &publicado) != nil || publicado.Referencia() != semilla.VersionRol.Referencia() {
		t.Fatal("el documento de rol no conserva la referencia fija")
	}
	if _, err := prepararRolMiBolsaPortalExterno(semilla, 1, ""); err == nil {
		t.Fatal("aceptó control revisado sin huella de preimagen")
	}
	documento, err := prepararAsignacionMiBolsaPortalExterno(semilla, semilla.VersionRol, 0, "")
	if err != nil || documento.VersionEsperada != 0 || documento.HuellaEsperada != nil {
		t.Fatalf("alta sin preimagen: %v", err)
	}
	h := sha256.Sum256(documento.Documento)
	if documento.HuellaSHA256 != hex.EncodeToString(h[:]) {
		t.Fatal("huella distinta del documento canónico")
	}
	var asignacion dominiovec.AsignacionPerfil
	if json.Unmarshal(documento.Documento, &asignacion) != nil || asignacion.Validar() != nil ||
		asignacion.Version != 1 || asignacion.VersionRolRef != semilla.VersionRol.Referencia() ||
		asignacion.Ambitos[0].Valores[0] != identidad.candidatoRef {
		t.Fatal("el documento no conserva el perfil y ámbito propio")
	}
	if _, err := prepararAsignacionMiBolsaPortalExterno(semilla, semilla.VersionRol, 1, ""); err == nil {
		t.Fatal("aceptó revisión sin huella de preimagen")
	}
	rolAjeno := semilla.VersionRol
	rolAjeno.RolID = "otro"
	if _, err := prepararAsignacionMiBolsaPortalExterno(semilla, rolAjeno, 0, ""); err == nil {
		t.Fatal("aceptó rol ajeno")
	}
	revision, err := prepararAsignacionMiBolsaPortalExterno(semilla, semilla.VersionRol, 1, strings.Repeat("a", 64))
	if err != nil || revision.HuellaEsperada == nil || revision.VersionEsperada != 1 {
		t.Fatalf("revisión con CAS: %v", err)
	}
}

type preparadorExteriorSinEfectos struct{}

func (preparadorExteriorSinEfectos) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	return mibolsa.Orden{}, errMiBolsaNoDisponible
}

func TestRutasMiBolsaExteriorSoloSiDependenciasCompletas(t *testing.T) {
	d := dependenciasMiBolsaPortalExterno{
		preparador:  preparadorExteriorSinEfectos{},
		autorizador: new(aplicacionvec.ServicioAutorizacionSolicitudLigadaV3),
		bolsa:       &pgxpool.Pool{},
		proveedores: map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{
			bolsapuertos.AudienciaMiBolsa:          {},
			bolsapuertos.AudienciaHistorialMiBolsa: {},
		},
		reloj: relojContratacionTemporalDesarrollo{},
	}
	if _, err := nuevasRutasMiBolsaPortalExterno(dependenciasMiBolsaPortalExterno{}); err == nil {
		t.Fatal("montó Bolsa sin autoridad")
	}
	rutas, err := nuevasRutasMiBolsaPortalExterno(d)
	if err != nil || len(rutas) != 2 || rutas[0].Ruta != bolsahttp.RutaMiBolsa || rutas[1].Ruta != bolsahttp.RutaMiBolsaHistorial {
		t.Fatalf("rutas de consulta: %v, %v", rutas, err)
	}
	d.reglas = reglasPortalCandidatoDesarrollo{}
	if _, err := nuevasRutasMiBolsaPortalExterno(d); err == nil {
		t.Fatal("montó acciones sin material por audiencia")
	}
	for _, par := range accionesPropiasPortalDesarrollo() {
		d.proveedores[par[1]] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	rutas, err = nuevasRutasMiBolsaPortalExterno(d)
	if err != nil || len(rutas) != 7 {
		t.Fatalf("rutas propias: %v, %v", rutas, err)
	}
	esperadas := []string{bolsahttp.RutaMiBolsa, bolsahttp.RutaMiBolsaHistorial,
		bolsahttp.RutaMiBolsaSolicitudes, bolsahttp.RutaMiBolsaSolicitudesDocumentales, bolsahttp.RutaMiBolsaRespuestas,
		bolsahttp.RutaMiBolsaDisposiciones, bolsahttp.RutaMiBolsaContacto}
	for i, ruta := range rutas {
		if ruta.Ruta != esperadas[i] || ruta.Manejador == nil {
			t.Fatalf("ruta %d no cerrada: %v", i, ruta.Ruta)
		}
	}
}
