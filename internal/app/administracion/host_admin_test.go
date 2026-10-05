package administracion

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpapi/adminselector"
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

// La composición entrega al selector el origen con el puerto público: un
// Origin sin puerto u otro puerto se rechaza antes de consultar la autoridad.
func TestMontajeSelectorUsaOrigenConPuertoPublico(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	host, ok := analizarHostAdmin("admin.invalid:8444")
	if !ok {
		t.Fatal("host sintético inválido")
	}
	for origen, esperado := range map[string]int{
		"https://admin.invalid:8444": http.StatusOK,
		"https://admin.invalid":      http.StatusForbidden,
		"https://admin.invalid:443":  http.StatusForbidden,
		"https://admin.invalid:8443": http.StatusForbidden,
	} {
		f := &fuenteActivosPrueba{resultado: LecturaPropiosAuditadaADMIN{Propios: propiosActivosPrueba(), AuditoriaComunRef: "auditoria:prueba:lectura"}}
		deps := DependenciasPerfiles{Activos: activosMapaPrueba(), ContextoConexion: func(ctx context.Context, _ net.Conn) context.Context { return ctx },
			ObservadorSelector: observadorActivosPrueba{o: observacionActivosPrueba(ahora)}, FuenteSeleccion: f,
			AudienciaSelector: "audiencia:admin", Auditor: &auditorActivosPrueba{}, Reloj: relojActivosPrueba{ahora}}
		h, err := montarActivosPerfiles(http.NotFoundHandler(), deps, host)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodGet, "https://admin.invalid:8444"+adminselector.RutaPropios, nil)
		r.Host = "admin.invalid:8444"
		r.Header.Set("Origin", origen)
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		r.Header.Set("Sec-Fetch-Mode", "cors")
		r.Header.Set("Sec-Fetch-Dest", "empty")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != esperado || (f.llamadas != 0) != (esperado == http.StatusOK) {
			t.Fatalf("Origin %q: %d (esperado %d) consultas=%d", origen, w.Code, esperado, f.llamadas)
		}
	}
}
