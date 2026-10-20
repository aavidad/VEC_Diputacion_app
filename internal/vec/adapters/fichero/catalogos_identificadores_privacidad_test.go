package fichero

import (
	"encoding/json"
	"strings"
	"testing"
)

// F1-108: el UUID de mapeo_fuente_ref llevaba un tramo de cifras que el
// patrón de teléfono tomaba por un contacto y Contratación temporal no
// arrancaba en copias nuevas.
const uuidConCifrasDeTelefono = "71a77621-ca82-49e8-b11f-612345678abc"

func TestPaqueteCatalogoAdmiteIdentificadoresOpacosCompletosEnRef(t *testing.T) {
	huella := strings.Repeat("a", 20) + "612345678" + strings.Repeat("b", 35)
	casos := []struct{ clave, valor string }{
		{"mapeo_fuente_ref", "correspondencia-org:" + uuidConCifrasDeTelefono},
		{"mapeo_fuente_ref", uuidConCifrasDeTelefono},
		{"organizacion_ref", "fuente:plan:" + uuidConCifrasDeTelefono},
		{"circuito_ref", "catalogo:circuito:" + huella},
	}
	for _, caso := range casos {
		t.Run(caso.clave+"="+caso.valor, func(t *testing.T) {
			if !patronTelefonoPaquete.MatchString(caso.valor) {
				t.Fatal("la prueba no reproduce la coincidencia falsa")
			}
			paquete := paqueteCatalogoPrivacidadPrueba(t)
			paquete.Catalogo.Entradas[0].Atributos[caso.clave] = caso.valor
			if _, err := nuevaConsultaCatalogosDesdeJSONConMarca(paqueteJSONPrueba(t, paquete), false); err != nil {
				t.Fatalf("identificador opaco rechazado: %v", err)
			}
		})
	}
}

func TestPaqueteCatalogoSigueDetectandoTelefonosJuntoAIdentificadores(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*paqueteCatalogo)
	}{
		{"movil sin prefijo", func(p *paqueteCatalogo) { p.Catalogo.Entradas[0].Descripcion = "Llamar al 612345678" }},
		{"movil con +34", func(p *paqueteCatalogo) { p.Catalogo.Entradas[0].Descripcion = "Llamar al +34612345678" }},
		{"movil con +34 y espacios", func(p *paqueteCatalogo) { p.Catalogo.Entradas[0].Descripcion = "Llamar al +34 612 345 678" }},
		{"fijo con espacios", func(p *paqueteCatalogo) { p.Catalogo.Entradas[0].Descripcion = "Oficina 958 24 77 00" }},
		{"movil con guiones", func(p *paqueteCatalogo) { p.Catalogo.Entradas[0].Descripcion = "Tel. 712-345-678" }},
		{"uuid en texto libre", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Descripcion = "Fuente " + uuidConCifrasDeTelefono
		}},
		{"uuid en atributo que no es ref", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["documento"] = uuidConCifrasDeTelefono
		}},
		{"telefono en el prefijo de la ref", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = "tel-612345678:" + uuidConCifrasDeTelefono
		}},
		{"telefono tras el uuid", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = uuidConCifrasDeTelefono + ":612345678"
		}},
		{"uuid en mayusculas no canonico", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = strings.ToUpper(uuidConCifrasDeTelefono)
		}},
		{"uuid incompleto", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = "org:" + uuidConCifrasDeTelefono[1:]
		}},
		{"correo en el prefijo de la ref", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = "persona@example.org:" + uuidConCifrasDeTelefono
		}},
		{"secreto en ref", func(p *paqueteCatalogo) {
			p.Catalogo.Entradas[0].Atributos["mapeo_fuente_ref"] = "-----BEGIN PRIVATE KEY-----"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			paquete := paqueteCatalogoPrivacidadPrueba(t)
			caso.alterar(&paquete)
			_, err := nuevaConsultaCatalogosDesdeJSONConMarca(paqueteJSONPrueba(t, paquete), false)
			if err == nil || !strings.Contains(err.Error(), "posible dato personal o secreto") {
				t.Fatalf("detección de privacidad no aplicada: %v", err)
			}
		})
	}
}

func paqueteJSONPrueba(t *testing.T, paquete paqueteCatalogo) []byte {
	t.Helper()
	contenido, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}
