package administracion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnalizarHostAdminFormaCerrada(t *testing.T) {
	for valor, esperado := range map[string]hostAdmin{
		"admin.cidonia.cloud":      {nombre: "admin.cidonia.cloud", autoridad: "admin.cidonia.cloud"},
		"admin.cidonia.cloud:443":  {nombre: "admin.cidonia.cloud", autoridad: "admin.cidonia.cloud"},
		"admin.cidonia.cloud:8444": {nombre: "admin.cidonia.cloud", autoridad: "admin.cidonia.cloud:8444"},
		"admin.cidonia.cloud:1":    {nombre: "admin.cidonia.cloud", autoridad: "admin.cidonia.cloud:1"},
	} {
		h, ok := analizarHostAdmin(valor)
		if !ok || h != esperado {
			t.Fatalf("%q: %+v %v", valor, h, ok)
		}
	}
	if h, _ := analizarHostAdmin("admin.cidonia.cloud:8444"); h.origen() != "https://admin.cidonia.cloud:8444" {
		t.Fatalf("origen %q", h.origen())
	}
	if h, _ := analizarHostAdmin("admin.cidonia.cloud:443"); h.origen() != "https://admin.cidonia.cloud" {
		t.Fatalf("origen 443 %q", h.origen())
	}
	for _, valor := range []string{"", "admin", "ADMIN.cidonia.cloud", "admin.cidonia.cloud:", "admin.cidonia.cloud:0",
		"admin.cidonia.cloud:65536", "admin.cidonia.cloud:+8444", "admin.cidonia.cloud:08444", "admin.cidonia.cloud:8444:1",
		"admin.cidonia.cloud:http", ":8444", "[::1]:8444", "admin.cidonia.cloud.", "admin..cloud", "-admin.cloud",
		"admin.cidonia.cloud/x", "admin_x.cloud", " admin.cidonia.cloud", "admin.cidonia.cloud :8444"} {
		if h, ok := analizarHostAdmin(valor); ok {
			t.Fatalf("%q admitido: %+v", valor, h)
		}
	}
}

// Con puerto público la puerta de activos exige la autoridad exacta en Host y
// el nombre sin puerto en la observación que se contrasta con host_admin.
func TestGateActivosConPuertoPublico(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	host, ok := analizarHostAdmin("admin.invalid:8444")
	if !ok {
		t.Fatal("host sintético inválido")
	}
	for _, caso := range []struct {
		cabecera, observado string
		esperado            int
	}{
		{"admin.invalid:8444", "admin.invalid", http.StatusOK},
		{"admin.invalid", "admin.invalid", http.StatusUnauthorized},
		{"admin.invalid:443", "admin.invalid", http.StatusUnauthorized},
		{"admin.invalid:8443", "admin.invalid", http.StatusUnauthorized},
		{"admin.invalid:8444", "admin.invalid:8444", http.StatusUnauthorized},
		{"admin.invalid:8444", "otro.invalid", http.StatusUnauthorized},
	} {
		obs := observacionActivosPrueba(ahora)
		obs.Host = caso.observado
		f := &fuenteActivosPrueba{resultado: LecturaPropiosAuditadaADMIN{Propios: propiosActivosPrueba(), AuditoriaComunRef: "auditoria:prueba:lectura"}}
		h := handlerPerfilesADMIN{observador: observadorActivosPrueba{o: obs}, fuenteSeleccion: f, auditor: &auditorActivosPrueba{}, host: host, audienciaSelector: "audiencia:admin", reloj: relojActivosPrueba{ahora}}
		r := httptest.NewRequest(http.MethodGet, "https://admin.invalid/admin/usuarios/", nil)
		r.Host = caso.cabecera
		if actual := h.estadoActivosSelector(context.Background(), r); actual != caso.esperado {
			t.Fatalf("Host %q observado %q: %d, esperado %d", caso.cabecera, caso.observado, actual, caso.esperado)
		}
		if caso.esperado != http.StatusOK && f.llamadas != 0 {
			t.Fatalf("Host %q consultó la autoridad", caso.cabecera)
		}
	}
}
