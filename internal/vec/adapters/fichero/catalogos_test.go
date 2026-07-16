package fichero

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

const rutaCatalogoProfesionalDemo = "../../../../data/catalogos/categorias-profesionales/v1.demo.json"

const (
	huellaCatalogoProfesionalV1 = "2a9aa4a903b765c2f46ceb7f429f342a13b229e54ca45813472cb9d0aa1a4f3e"
	huellaFuenteOPES            = "4c94c36a2f024edda8b0c4d7c0cec965b97096f0ffbc64df3e13f64dad568b1b"
)

func TestPaqueteCategoriasProfesionalesEsEstrictoCompletoEInmutable(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos(rutaCatalogoProfesionalDemo)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), "categorias-profesionales", 1)
	if err != nil || len(catalogo.Entradas) != 58 {
		t.Fatalf("catalogo=%#v error=%v", catalogo, err)
	}
	areas := map[string]int{}
	for indice, entrada := range catalogo.Entradas {
		areas[entrada.Atributos["area"]]++
		if entrada.Atributos["area_etiqueta"] == "" || entrada.Orden != indice+1 {
			t.Fatalf("categoria %q sin area u orden canonico: %d", entrada.Clave, entrada.Orden)
		}
	}
	if areas["administracion_general"] != 5 || areas["administracion_especial"] != 53 || len(areas) != 2 {
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
	if err := json.Unmarshal(contenido, &paquete); err != nil || paquete.Fuente.OrigenSHA256 != huellaFuenteOPES {
		t.Fatalf("huella de la fuente historica=%q error=%v", paquete.Fuente.OrigenSHA256, err)
	}
	// El 58 solo se fija en esta prueba de integridad del paquete inicial; el
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
