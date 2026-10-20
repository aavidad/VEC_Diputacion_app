package fichero

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPaqueteCatalogoAdmiteHuellasYReferenciasOpacasValidas(t *testing.T) {
	paquete := paqueteCatalogoPrivacidadPrueba(t)
	// Ambas cadenas tienen cifras que el patrón de teléfono confunde con un
	// contacto cuando se inspecciona el JSON completo.
	huella := strings.Repeat("a", 20) + "612345678" + strings.Repeat("b", 35)
	referencia := "car_" + strings.Repeat("a", 8) + "612345678" + strings.Repeat("b", 15)
	if len(huella) != 64 || len(referencia) != 36 || !patronTelefonoPaquete.MatchString(huella) ||
		!patronTelefonoPaquete.MatchString(referencia) {
		t.Fatal("la prueba no reproduce la coincidencia falsa")
	}
	paquete.Fuente.OrigenSHA256 = huella
	paquete.Catalogo.Entradas[0].Atributos["circuito_sha256"] = huella
	paquete.Catalogo.Entradas[0].Atributos["cargo_ref"] = referencia
	contenido, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevaConsultaCatalogosDesdeJSONConMarca(contenido, false); err != nil {
		t.Fatalf("huella y referencia opaca rechazadas: %v", err)
	}
}

func TestPaqueteCatalogoAdmiteActoresDePersonaOpacos(t *testing.T) {
	referencia := "per_" + strings.Repeat("a", 8) + "612345678" + strings.Repeat("b", 15)
	if !patronTelefonoPaquete.MatchString(referencia) {
		t.Fatal("la referencia no reproduce el falso positivo de teléfono")
	}
	for _, campo := range []string{"creado_por", "publicado_por"} {
		t.Run(campo, func(t *testing.T) {
			paquete := paqueteCatalogoPrivacidadPrueba(t)
			if campo == "creado_por" {
				paquete.Catalogo.CreadoPor = referencia
			} else {
				paquete.Catalogo.PublicadoPor = referencia
			}
			contenido, err := json.Marshal(paquete)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := nuevaConsultaCatalogosDesdeJSONConMarca(contenido, false); err != nil {
				t.Fatalf("actor de Persona canónico rechazado: %v", err)
			}
		})
	}
}

func TestPrivacidadCatalogoNoEximeOtrosActoresNiValores(t *testing.T) {
	referencia := "per_" + strings.Repeat("a", 8) + "612345678" + strings.Repeat("b", 15)
	valores := []string{
		"Contacto 612345678", "persona@example.org", "Titular 12345678Z",
		"per_612345678", referencia + "b",
		strings.ToUpper(referencia), strings.Replace(referencia, "per_", "car_", 1),
		" " + referencia, referencia + " contacto",
		"-----BEGIN PRIVATE KEY-----",
	}
	for _, campo := range []string{"creado_por", "publicado_por"} {
		for i, valor := range valores {
			if !contieneDatoPrivadoCatalogo(map[string]any{"catalogo": map[string]any{campo: valor}}, "") {
				t.Errorf("se eximió el valor no canónico %d de %s", i, campo)
			}
		}
	}
	for _, caso := range []struct{ ruta, campo string }{
		{"", "creado_por"}, {"fuente", "publicado_por"},
		{"catalogo.entradas[].atributos", "creado_por"},
		{"catalogo", "otro_por"}, {"catalogo", "descripcion"},
	} {
		if !contieneDatoPrivadoCatalogo(map[string]any{caso.campo: referencia}, caso.ruta) {
			t.Errorf("referencia eximida en %s.%s", caso.ruta, caso.campo)
		}
	}
}

func TestPaqueteCatalogoPrivacidadRechazaTextoYReferenciasNoOpacas(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*paqueteCatalogo)
	}{
		{"correo en texto", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Descripcion = "Contacto persona@example.org"
		}},
		{"telefono en texto", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Descripcion = "Contacto 612345678"
		}},
		{"dni en texto", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Descripcion = "Titular 12345678Z"
		}},
		{"telefono en ref malformada", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["cargo_ref"] = "car_612345678"
		}},
		{"telefono en ref generica", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["otro_ref"] = "612345678"
		}},
		{"telefono en huella malformada", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["circuito_sha256"] = "612345678"
		}},
		{"huella con cifras en texto libre", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Descripcion = strings.Repeat("a", 20) + "612345678" + strings.Repeat("b", 35)
		}},
		{"secreto en ref malformada", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["cargo_ref"] = "-----BEGIN PRIVATE KEY-----"
		}},
		{"telefono en orden numerico", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Orden = 612345678
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			paquete := paqueteCatalogoPrivacidadPrueba(t)
			caso.alterar(&paquete)
			contenido, err := json.Marshal(paquete)
			if err != nil {
				t.Fatal(err)
			}
			_, err = nuevaConsultaCatalogosDesdeJSONConMarca(contenido, false)
			if err == nil || !strings.Contains(err.Error(), "posible dato personal o secreto") {
				t.Fatalf("detección de privacidad no aplicada: %v", err)
			}
		})
	}
}

func TestPaqueteCatalogoRechazaEstructuraDesconocidaAntesDeRecorrerla(t *testing.T) {
	paquete := paqueteCatalogoPrivacidadPrueba(t)
	contenido, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	anidado := strings.Repeat("[", 64) + `"612345678"` + strings.Repeat("]", 64)
	conDesconocido := strings.Replace(string(contenido), `"version_esquema":1`,
		`"desconocido":`+anidado+`,"version_esquema":1`, 1)
	if conDesconocido == string(contenido) {
		t.Fatal("no se preparó el campo desconocido")
	}
	if _, err := nuevaConsultaCatalogosDesdeJSONConMarca([]byte(conDesconocido), false); err == nil ||
		!strings.Contains(err.Error(), "esquema no valido") {
		t.Fatalf("estructura desconocida recorrida antes del esquema: %v", err)
	}
}

func TestPaqueteCatalogoPrivacidadDetectaEscapesYDuplicados(t *testing.T) {
	paquete := paqueteCatalogoPrivacidadPrueba(t)
	paquete.Catalogo.Entradas[0].Descripcion = "Contacto 612345678"
	contenido, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	escapado := strings.Replace(string(contenido), "Contacto 612345678", `Contacto \u003612345678`, 1)
	if escapado == string(contenido) {
		t.Fatal("no se preparó el valor escapado")
	}
	if _, err := nuevaConsultaCatalogosDesdeJSONConMarca([]byte(escapado), false); err == nil ||
		!strings.Contains(err.Error(), "posible dato personal o secreto") {
		t.Fatalf("teléfono escapado admitido: %v", err)
	}

	duplicado := strings.Replace(string(contenido), `"version_esquema":1`, `"version_esquema":1,"version_esquema":1`, 1)
	if duplicado == string(contenido) {
		t.Fatal("no se preparó la clave duplicada")
	}
	if _, err := nuevaConsultaCatalogosDesdeJSONConMarca([]byte(duplicado), false); err == nil ||
		!strings.Contains(err.Error(), "contenido JSON rechazado") {
		t.Fatalf("clave duplicada admitida: %v", err)
	}
}

func paqueteCatalogoPrivacidadPrueba(t *testing.T) paqueteCatalogo {
	t.Helper()
	contenido, err := os.ReadFile(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	var paquete paqueteCatalogo
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	return paquete
}
