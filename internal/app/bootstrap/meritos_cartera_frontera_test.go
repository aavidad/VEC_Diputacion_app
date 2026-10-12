package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestFronteraCarteraAdmiteFiltroSoloTrasIdentidadYNoRelajaUsuarios(t *testing.T) {
	registrador := &registradorDenegacionPreferenciasPrueba{}
	cartera := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{},
		manejador:   http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("sin certificado llegó al handler") }),
		registrador: registrador, ruta: meritoshttp.RutaCartera,
		superficieAuditoria: vecports.SuperficieAuditoriaFronteraRutaExactaMeritosCartera}
	w := httptest.NewRecorder()
	cartera.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("sin certificado llegó al despacho") })).
		ServeHTTP(w, httptest.NewRequest(http.MethodGet, meritoshttp.RutaCartera+"?filtro=retirados&lang=en", nil))
	if w.Code != http.StatusUnauthorized || len(registrador.ordenes) != 1 ||
		registrador.ordenes[0].Ruta != meritoshttp.RutaCartera ||
		registrador.ordenes[0].Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaMeritosCartera {
		t.Fatalf("selector antes de sesión: estado=%d ordenes=%+v", w.Code, registrador.ordenes)
	}
	usuarios := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{},
		manejador:   http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("query Usuarios llegó al handler") }),
		registrador: registrador, ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal}
	w = httptest.NewRecorder()
	usuarios.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("query Usuarios llegó al despacho") })).
		ServeHTTP(w, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferenciasAreaPersonal+"?filtro=retirados", nil))
	if w.Code != http.StatusUnprocessableEntity || len(registrador.ordenes) != 1 {
		t.Fatalf("ruta Usuarios relajada: estado=%d ordenes=%d", w.Code, len(registrador.ordenes))
	}
}
