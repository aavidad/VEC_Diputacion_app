package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestCatalogosAltaContratacionTemporalDesarrolloCompartenOrigenConAltaEnMTLSReal(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)
	ruta := baseURL + rutaCatalogosAltaContratacionTemporalDesarrollo

	t.Run("sin certificado no alcanza el catalogo", func(t *testing.T) {
		cliente := nuevoClienteSinCertificadoContratacionTemporalDesarrollo(t, rutas)
		peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
			t, http.MethodGet, ruta,
		)
		respuesta, err := cliente.Do(peticion)
		if err == nil {
			contenido, _ := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			t.Fatalf("el listener acepto catalogo sin certificado: %d %s", respuesta.StatusCode, contenido)
		}
	})

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	t.Cleanup(cliente.CloseIdleConnections)

	for _, caso := range []struct{ nombre, cabecera, valor string }{
		{"bearer", "Authorization", "Bearer autoridad-forjada"},
		{"cookie", "Cookie", "sesion=forjada"},
		{"principal cliente", "X-Vec-Principal", "administrador"},
		{"identidad cliente", "Identity", "administrador"},
		{"rol cliente", "X-Role", "administrador"},
	} {
		t.Run(caso.nombre+" no aporta autoridad", func(t *testing.T) {
			peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
				t, http.MethodGet, ruta,
			)
			peticion.Header.Set(caso.cabecera, caso.valor)
			respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
				t, cliente, peticion,
			)
			if respuesta.StatusCode == http.StatusOK ||
				bytes.Contains(contenido, []byte(esquemaCatalogosAltaContratacionTemporal)) ||
				len(respuesta.Header.Values("Set-Cookie")) != 0 {
				t.Fatalf("cabecera cliente alcanzo el catalogo: %d %s", respuesta.StatusCode, contenido)
			}
		})
	}

	peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
		t, http.MethodGet, ruta,
	)
	respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticion,
	)
	if respuesta.StatusCode != http.StatusOK {
		t.Fatalf("catalogo=%d %s", respuesta.StatusCode, contenido)
	}
	if !strings.Contains(respuesta.Header.Get("Cache-Control"), "no-store") ||
		respuesta.Header.Get("Content-Type") != "application/json; charset=utf-8" ||
		len(respuesta.Header.Values("Set-Cookie")) != 0 {
		t.Fatalf("cabeceras inseguras: %v", respuesta.Header)
	}
	var sobreCrudo map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &sobreCrudo); err != nil ||
		len(sobreCrudo) != 1 || sobreCrudo["data"] == nil {
		t.Fatalf("sobre abierto o invalido: %v %s", err, contenido)
	}
	var datosCrudos map[string]json.RawMessage
	if err := json.Unmarshal(sobreCrudo["data"], &datosCrudos); err != nil ||
		len(datosCrudos) != 5 {
		t.Fatalf("datos abiertos o invalidos: %v %s", err, sobreCrudo["data"])
	}
	for _, campo := range []string{"esquema", "centros", "categorias", "motivos", "documentos"} {
		if datosCrudos[campo] == nil {
			t.Fatalf("falta campo cerrado %q: %s", campo, sobreCrudo["data"])
		}
	}

	var catalogos respuestaCatalogosAltaContratacionTemporalDesarrollo
	if err := json.Unmarshal(contenido, &catalogos); err != nil {
		t.Fatalf("decodificar catalogos: %v: %s", err, contenido)
	}
	if catalogos.Data.Esquema != esquemaCatalogosAltaContratacionTemporal ||
		len(catalogos.Data.Centros) != 1 ||
		len(catalogos.Data.Centros[0].Contactos) != 1 ||
		len(catalogos.Data.Categorias) != 1 ||
		len(catalogos.Data.Categorias[0].GruposSubgrupos) != 1 ||
		len(catalogos.Data.Motivos) != 1 ||
		catalogos.Data.Documentos == nil || len(catalogos.Data.Documentos) != 0 {
		t.Fatalf("catalogo no minimo: %+v", catalogos.Data)
	}
	centro := catalogos.Data.Centros[0]
	contacto := centro.Contactos[0]
	categoria := catalogos.Data.Categorias[0]
	grupo := categoria.GruposSubgrupos[0]
	motivo := catalogos.Data.Motivos[0]
	if centro.Referencia != centroAltaContratacionTemporalDesarrollo ||
		contacto.Referencia != contactoAltaContratacionTemporalDesarrollo ||
		categoria.Referencia != categoriaAltaContratacionTemporalDesarrollo ||
		grupo.Clave != grupoSubgrupoAltaContratacionTemporalDesarrollo ||
		motivo.Clave != string(motivoAltaContratacionTemporalDesarrollo) {
		t.Fatalf("referencias divergentes del alta: %+v", catalogos.Data)
	}
	for nombre, valores := range map[string][2]string{
		"centro":    {centro.Etiqueta, "Centro solicitante"},
		"contacto":  {contacto.Etiqueta, "Contacto del centro"},
		"categoria": {categoria.Etiqueta, "Categoría C2"},
		"grupo":     {grupo.Etiqueta, "Grupo C2"},
		"motivo":    {motivo.Etiqueta, "Sustitución temporal"},
	} {
		etiqueta, esperada := valores[0], valores[1]
		minusculas := strings.ToLower(etiqueta)
		if etiqueta != esperada || strings.Contains(minusculas, "desarrollo") ||
			strings.Contains(minusculas, "no autoritativ") ||
			strings.Contains(minusculas, "demo") {
			t.Fatalf("%s muestra una etiqueta no neutra: %q", nombre, etiqueta)
		}
	}

	cuerpoAlta := cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
		t, centro.Referencia, contacto.Referencia, categoria.Referencia,
		grupo.Clave, motivo.Clave,
	)
	peticionAlta := nuevaPeticionJSONContratacionTemporalDesarrolloPrueba(
		t, baseURL+httpinterno.RutaAltaSolicitudes, cuerpoAlta,
	)
	respuestaAlta, contenidoAlta := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionAlta,
	)
	if respuestaAlta.StatusCode != http.StatusCreated ||
		!bytes.Contains(contenidoAlta, []byte(`"expediente_ref"`)) {
		t.Fatalf("las referencias catalogadas no alcanzan el alta: %d %s", respuestaAlta.StatusCode, contenidoAlta)
	}

	peticionHEAD := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
		t, http.MethodHead, ruta,
	)
	respuestaHEAD, contenidoHEAD := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
		t, cliente, peticionHEAD,
	)
	if respuestaHEAD.StatusCode != http.StatusOK || len(contenidoHEAD) != 0 ||
		!strings.Contains(respuestaHEAD.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("HEAD invalido: %d %q %v", respuestaHEAD.StatusCode, contenidoHEAD, respuestaHEAD.Header)
	}

	for _, metodo := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions,
	} {
		t.Run("rechaza "+metodo, func(t *testing.T) {
			peticion := nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
				t, metodo, ruta,
			)
			respuesta, contenido := ejecutarPeticionContratacionTemporalDesarrolloPrueba(
				t, cliente, peticion,
			)
			if respuesta.StatusCode != http.StatusMethodNotAllowed ||
				respuesta.Header.Get("Allow") != "GET, HEAD" ||
				bytes.Contains(contenido, []byte(esquemaCatalogosAltaContratacionTemporal)) {
				t.Fatalf("%s no fue rechazado: %d %s", metodo, respuesta.StatusCode, contenido)
			}
		})
	}
}

func TestCatalogosAltaContratacionTemporalNoSeRegistranFueraDeDesarrollo(
	t *testing.T,
) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatalf("componer dependencias de prueba: %v", err)
	}
	resolvedor, err := composicion.ResolvedorIdentidad()
	if err != nil {
		t.Fatalf("resolver identidad de prueba: %v", err)
	}
	casos := []struct {
		nombre    string
		modificar func(*config.Config)
	}{
		{"produccion", func(actual *config.Config) {
			actual.ExecutionProfile = config.ExecutionProfileProduction
		}},
		{"sin segunda llave", func(actual *config.Config) {
			actual.DevelopmentGuard = ""
		}},
		{"red no loopback", func(actual *config.Config) {
			actual.Address = "0.0.0.0:0"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := cfg
			caso.modificar(&actual)
			rutas, autoridad, cerrar, err := nuevasRutasContratacionTemporalDesarrollo(
				actual, resolvedor, composicion.derivadorIdempotencia, io.Discard,
			)
			if !errors.Is(err, ErrActivacionDesarrolloInvalida) ||
				rutas != nil || autoridad != nil || cerrar != nil {
				t.Fatalf("configuracion invalida registro rutas CT: rutas=%v autoridad=%v error=%v", rutas, autoridad, err)
			}
		})
	}
}

func nuevaPeticionCatalogosAltaContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	metodo string,
	ruta string,
) *http.Request {
	t.Helper()
	peticion, err := http.NewRequest(metodo, ruta, nil)
	if err != nil {
		t.Fatal(err)
	}
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func cuerpoAltaDesdeCatalogosContratacionTemporalDesarrolloPrueba(
	t *testing.T,
	centro string,
	contacto string,
	categoria string,
	grupo string,
	motivo string,
) string {
	t.Helper()
	ahora := time.Now().UTC()
	inicio := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, 1)
	fin := inicio.AddDate(0, 3, 0)
	cuerpo := map[string]any{
		"clave_idempotencia": "781972a8-59d6-4168-a877-d9c61c0ae8e4",
		"solicitud": map[string]any{
			"centro_ref": centro, "contacto_ref": contacto,
			"categoria_ref": categoria, "grupo_subgrupo": grupo,
			"motivo_clave": motivo,
			"detalle":      "Cobertura temporal en entorno local no autoritativo.",
			"periodo": map[string]string{
				"inicio": inicio.Format("2006-01-02T15:04:05Z"),
				"fin":    fin.Format("2006-01-02T15:04:05Z"),
			},
			"rc":                  map[string]bool{"existe": false},
			"documentos_adjuntos": []string{},
			"observaciones":       "",
		},
	}
	contenido, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	return string(contenido)
}
