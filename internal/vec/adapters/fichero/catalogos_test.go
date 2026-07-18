package fichero

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

const rutaCatalogoProfesionalDemo = "../../../../data/catalogos/categorias-profesionales/v1.demo.json"

const (
	huellaCatalogoProfesionalV1 = "b800a7e9c306fa8027709cfb4304cc8ccf8065f888673da71bd73a138c519233"
	huellaFuentePublica         = "de9af856fea93e91340e77aef6403d607e49b3822e5d8f7856bca4a5d6ad5172"
)

func TestPaqueteCategoriasProfesionalesEsEstrictoCompletoEInmutable(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), "categorias-profesionales", 1)
	if err != nil || len(catalogo.Entradas) != 68 {
		t.Fatalf("catalogo=%#v error=%v", catalogo, err)
	}
	areas := map[string]int{}
	for indice, entrada := range catalogo.Entradas {
		areas[entrada.Atributos["area"]]++
		if entrada.Atributos["area_etiqueta"] == "" || entrada.Orden != indice+1 {
			t.Fatalf("categoria %q sin area u orden canonico: %d", entrada.Clave, entrada.Orden)
		}
	}
	if areas["administracion_general"] != 5 || areas["administracion_especial"] != 60 ||
		areas["organismos_dependientes"] != 3 || len(areas) != 3 {
		t.Fatalf("distribucion de areas inesperada: %#v", areas)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil || huella != huellaCatalogoProfesionalV1 {
		t.Fatalf("huella del catalogo v1=%q error=%v", huella, err)
	}
	contenido, err := os.ReadFile(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	var paquete paqueteCatalogo
	if err := json.Unmarshal(contenido, &paquete); err != nil || paquete.Fuente.OrigenSHA256 != huellaFuentePublica {
		t.Fatalf("huella de la fuente historica=%q error=%v", paquete.Fuente.OrigenSHA256, err)
	}
	contenidoFuente, err := os.ReadFile("../../../../docs/demo/fuentes_publicas_bolsa.md")
	if err != nil {
		t.Fatal(err)
	}
	huellaFuenteReal := sha256.Sum256(contenidoFuente)
	if obtenida := fmt.Sprintf("%x", huellaFuenteReal); obtenida != paquete.Fuente.OrigenSHA256 {
		t.Fatalf("huella real del inventario=%q declarada=%q", obtenida, paquete.Fuente.OrigenSHA256)
	}
	// El 68 solo se fija en esta prueba de integridad del paquete DEMO actual; el
	// codigo de produccion admite cualquier version y numero valido de entradas.
	catalogo.Entradas[0].Etiqueta = "alterada"
	catalogo.Entradas[0].Atributos["source_path"] = "alterado"
	otra, err := consulta.ObtenerCatalogo(context.Background(), "categorias-profesionales", 1)
	if err != nil || otra.Entradas[0].Etiqueta == "alterada" || otra.Entradas[0].Atributos["source_path"] == "alterado" {
		t.Fatalf("la instantanea pudo mutarse: %#v, %v", otra.Entradas[0], err)
	}
	metadatos, err := consulta.ObtenerMetadatosFuenteCatalogos(context.Background())
	if err != nil || !metadatos.Demostracion || metadatos.Revision == "" {
		t.Fatalf("metadatos=%#v error=%v", metadatos, err)
	}
}

func TestPaqueteCatalogoRechazaCampoDesconocidoYClaveDuplicada(t *testing.T) {
	base, err := os.ReadFile(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]string{
		"campo desconocido": strings.Replace(string(base), `"version_esquema": 1,`, `"version_esquema": 1, "desconocido": true,`, 1),
		"clave duplicada":   strings.Replace(string(base), `"version_esquema": 1,`, `"version_esquema": 1, "version_esquema": 1,`, 1),
		"dni":               strings.Replace(string(base), "pendiente de validación por RRHH", "titular 12345678Z", 1),
		"nie":               strings.Replace(string(base), "pendiente de validación por RRHH", "titular X1234567L", 1),
		"nif juridico":      strings.Replace(string(base), "pendiente de validación por RRHH", "entidad B12345678", 1),
		"correo":            strings.Replace(string(base), "pendiente de validación por RRHH", "contacto persona@example.org", 1),
		"telefono":          strings.Replace(string(base), "pendiente de validación por RRHH", "contacto 612345678", 1),
		"iban":              strings.Replace(string(base), "pendiente de validación por RRHH", "cuenta ES9121000418450200051332", 1),
		"clave privada":     strings.Replace(string(base), "pendiente de validación por RRHH", "-----BEGIN PRIVATE KEY-----", 1),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := nuevaConsultaCatalogosDesdeJSON([]byte(contenido))
			if !errors.Is(err, domain.ErrCatalogoConfigurableInvalido) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestConsultaCatalogosRespetaContextoCancelado(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := consulta.ObtenerCatalogo(ctx, "categorias-profesionales", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
