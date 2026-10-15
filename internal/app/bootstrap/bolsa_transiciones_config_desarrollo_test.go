package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

func TestSelectorGobiernoTransicionesBolsaEsOpcionalYEstricto(t *testing.T) {
	t.Setenv(envGobiernoTransicionesBolsa, "")
	if _, activa := leerSeleccionGobiernoTransicionesBolsa(); activa {
		t.Fatal("una selección vacía activó la edición")
	}
	sha := strings.Repeat("a", 64)
	t.Setenv(envGobiernoTransicionesBolsa, `{"catalogo_version":1,"catalogo_sha256":"`+sha+`",`+
		`"perfil_ref":"prf_rrhh_sintetico_1234567890123456",`+
		`"motivo":{"catalogo_id":"motivos_transiciones_bolsa","catalogo_version":1,`+
		`"catalogo_huella_sha256":"`+sha+`","entrada_clave":"motivo_publicar"}}`)
	if s, activa := leerSeleccionGobiernoTransicionesBolsa(); !activa || s.CatalogoVersion != 1 || s.PerfilRef == "" {
		t.Fatalf("selección válida rechazada: %+v %v", s, activa)
	}
	t.Setenv(envGobiernoTransicionesBolsa, `{"catalogo_version":1,"catalogo_sha256":"`+sha+`",`+
		`"perfil_ref":"prf_rrhh_sintetico_1234567890123456",`+
		`"motivo":{"catalogo_id":"motivos_transiciones_bolsa","catalogo_version":1,`+
		`"catalogo_huella_sha256":"`+sha+`","entrada_clave":"motivo_publicar"},"accion":"conceder"}`)
	if _, activa := leerSeleccionGobiernoTransicionesBolsa(); activa {
		t.Fatal("la selección privada aceptó un campo de concesión")
	}
}

func TestGobiernoTransicionesSinSelectorConservaFronteraYResponde503(t *testing.T) {
	t.Setenv(envGobiernoTransicionesBolsa, "")
	fronteras, acciones := descriptoresGobiernoTransicionesBolsaDesarrollo(
		"prf_bolsa_bback", politicaDescriptoresBolsaPrueba(t))
	if len(fronteras) != 2 || len(acciones) != 2 {
		t.Fatalf("fronteras=%d acciones=%d", len(fronteras), len(acciones))
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		if _, ok := catalogo.resolver(metodo, bolsahttp.RutaGobiernoTransicionesSituacion); !ok {
			t.Fatalf("falta frontera %s", metodo)
		}
	}
	ruta := componerRutaGobiernoTransicionesBolsaDesarrollo(context.Background(), nil, nil, nil, nil, nil, nil, "")
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, bolsahttp.RutaGobiernoTransicionesSituacion, nil))
	if ruta.Ruta != bolsahttp.RutaGobiernoTransicionesSituacion || w.Code != http.StatusServiceUnavailable {
		t.Fatalf("ruta opcional=%s estado=%d", ruta.Ruta, w.Code)
	}
}
