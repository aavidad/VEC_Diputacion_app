package preparacionbases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrMaterialInvalido = errors.New("bolsa.preparacion_bases.material_invalido")

const MaximoBytesMaterial = 256 * 1024

// Material conserva contenido incompleto. No es ConfiguracionFijadaConvocatoria
// ni proporciona una conversion al circuito de gobierno, firma o publicacion.
type Material struct {
	Contenido   bolsa.ContenidoPublicableConvocatoria `json:"contenido"`
	Referencias []ReferenciaPropuesta                 `json:"referencias"`
}

// ReferenciaPropuesta identifica una dependencia exacta, sin afirmar vigencia.
// Reglas de baremo, meritos y datos de Personal permanecen en sus propietarios.
type ReferenciaPropuesta struct {
	Campo      string                                    `json:"campo"`
	Referencia bolsa.ReferenciaConfiguracionConvocatoria `json:"referencia"`
}

type Pendiente struct {
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

var camposReferencia = []string{
	"fuente_bases", "catalogos", "calendario", "reglas_baremacion", "flujo_proceso",
	"flujo_solicitud", "plantilla", "plaza", "oep", "rpt",
}

// Canonico permite ausencia y referencias parciales; limita material antes de
// copiarlo y rechaza controles, Unicode invalido y campos de dependencia ajenos.
func (m Material) Canonico() (Material, error) {
	c := m.Contenido
	if len(m.Referencias) > len(camposReferencia) || len(c.Categorias) > 100 ||
		len(c.Plazos) > 100 || len(c.Requisitos) > 200 || len(c.Documentos) > 100 || len(c.Ayuda) > 100 ||
		!textosSeguros(reflect.ValueOf(m)) {
		return Material{}, ErrMaterialInvalido
	}
	campos := make(map[string]bool, len(camposReferencia))
	for _, campo := range camposReferencia {
		campos[campo] = true
	}
	vistos := make(map[string]bool, len(m.Referencias))
	for _, ref := range m.Referencias {
		if !campos[ref.Campo] || vistos[ref.Campo] || ref.Referencia.Version < 0 || ref.Referencia.Version > 1_000_000 {
			return Material{}, ErrMaterialInvalido
		}
		vistos[ref.Campo] = true
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > MaximoBytesMaterial {
		return Material{}, ErrMaterialInvalido
	}
	var clon Material
	if json.Unmarshal(b, &clon) != nil {
		return Material{}, ErrMaterialInvalido
	}
	clon.Contenido.Categorias = append([]string{}, clon.Contenido.Categorias...)
	clon.Contenido.Plazos = append([]bolsa.PlazoConvocatoria{}, clon.Contenido.Plazos...)
	clon.Contenido.Requisitos = append([]bolsa.RequisitoConvocatoria{}, clon.Contenido.Requisitos...)
	clon.Contenido.Documentos = append([]bolsa.DocumentoPublicableConvocatoria{}, clon.Contenido.Documentos...)
	clon.Contenido.Ayuda = append([]bolsa.AyudaConvocatoria{}, clon.Contenido.Ayuda...)
	// El orden de requisitos, plazos y documentos propuestos se conserva.
	sort.Slice(clon.Referencias, func(i, j int) bool { return clon.Referencias[i].Campo < clon.Referencias[j].Campo })
	if clon.Referencias == nil {
		clon.Referencias = []ReferenciaPropuesta{}
	}
	return clon, nil
}

func (m Material) HuellaSHA256() (string, error) {
	c, err := m.Canonico()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(struct {
		Esquema  string   `json:"esquema"`
		Material Material `json:"material"`
	}{"bolsa.preparacion_bases.material.v1", c})
	if err != nil {
		return "", ErrMaterialInvalido
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Pendientes se deriva del material; el consumidor nunca declara una
// referencia como aprobada. Un baremo requiere lectura exacta con estado
// disponible_para_preparacion antes de usarlo en el siguiente circuito.
func (m Material) Pendientes() ([]Pendiente, error) {
	c, err := m.Canonico()
	if err != nil {
		return nil, err
	}
	refs := map[string]bolsa.ReferenciaConfiguracionConvocatoria{}
	for _, ref := range c.Referencias {
		refs[ref.Campo] = ref.Referencia
	}
	resultado := make([]Pendiente, 0, 20)
	for _, campo := range camposReferencia {
		ref := refs[campo]
		codigo := "referencia_no_verificada"
		if ref == (bolsa.ReferenciaConfiguracionConvocatoria{}) {
			codigo = "referencia_ausente"
		} else if ref.Validar() != nil {
			codigo = "referencia_invalida"
		}
		resultado = append(resultado, Pendiente{campo, codigo})
	}
	if c.Contenido.Validar() != nil {
		resultado = append(resultado, Pendiente{"contenido", "contenido_no_validado"})
	}
	for _, campo := range []string{"documentos_admitidos", "firma_y_custodia", "acto_aprobacion", "publicacion_oficial"} {
		resultado = append(resultado, Pendiente{campo, "circuito_pendiente"})
	}
	return resultado, nil
}

func IdentificadorValido(s string) bool {
	return len(s) > 0 && len(s) <= 180 && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789:_./-") == ""
}

func HuellaValida(s string) bool {
	return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == ""
}

func textosSeguros(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if len(s) > 12000 || !utf8.ValidString(s) {
			return false
		}
		for _, r := range s {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return false
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && !textosSeguros(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if !textosSeguros(v.Index(i)) {
				return false
			}
		}
	}
	return true
}
