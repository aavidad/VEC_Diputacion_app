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
		if !campos[ref.Campo] || vistos[ref.Campo] {
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
	b, err := m.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Pendientes se deriva del material; el consumidor nunca declara una
// referencia como aprobada. Un baremo requiere lectura exacta con estado
// disponible_para_preparacion antes de usarlo en el siguiente circuito.
type EvaluacionMaterialBases struct {
	ContenidoCanonico *bolsa.ContenidoPublicableConvocatoria
	Pendientes        []Pendiente
}

func (m Material) Pendientes() ([]Pendiente, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return nil, err
	}
	evaluacion, err := EvaluarMaterialBases(canonico)
	return evaluacion.Pendientes, err
}

// EvaluarMaterialBases es la unica politica de preparacion incompleta.
// Canoniza con el dominio Bolsa existente y conserva cada dependencia pendiente.
func EvaluarMaterialBases(m Material) (EvaluacionMaterialBases, error) {
	// La evaluacion conserva referencias propuestas invalidas como pendientes.
	// Los limites del almacenamiento se aplican aparte en Material.Canonico;
	// no convierten aqui un aviso de la CLI existente en un fallo de entrada.
	c := m
	if len(c.Referencias) > len(camposReferencia) {
		return EvaluacionMaterialBases{}, ErrMaterialInvalido
	}
	refs := map[string]bolsa.ReferenciaConfiguracionConvocatoria{}
	for _, ref := range c.Referencias {
		permitido := false
		for _, campo := range camposReferencia {
			if ref.Campo == campo {
				permitido = true
				break
			}
		}
		if _, repetida := refs[ref.Campo]; !permitido || repetida {
			return EvaluacionMaterialBases{}, ErrMaterialInvalido
		}
		refs[ref.Campo] = ref.Referencia
	}
	resultado := make([]Pendiente, 0, 23)
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
	contenido := c.Contenido
	for _, campo := range []struct {
		clave   string
		ausente bool
	}{
		{"identificador_publico", contenido.IdentificadorPublico == ""}, {"tipo", contenido.Tipo == ""},
		{"titulo", contenido.Titulo == ""}, {"resumen", contenido.Resumen == ""},
		{"catalogo_categorias", contenido.CatalogoCategorias == (bolsa.ReferenciaCatalogoCategorias{})},
		{"categorias", len(contenido.Categorias) == 0}, {"plazos", len(contenido.Plazos) == 0},
		{"documentos_propuestos", len(contenido.Documentos) == 0},
	} {
		if campo.ausente {
			resultado = append(resultado, Pendiente{campo.clave, "material_ausente"})
		}
	}
	canonico, errCanonico := c.Contenido.ClonarCanonico()
	if errors.Is(errCanonico, bolsa.ErrVersionConvocatoriaGobernadaInvalida) {
		resultado = append(resultado, Pendiente{"contenido", "contenido_no_validado"})
	} else if errCanonico != nil {
		return EvaluacionMaterialBases{}, errCanonico
	}
	for _, campo := range []string{"documentos_admitidos", "firma_y_custodia", "acto_aprobacion", "publicacion_oficial"} {
		resultado = append(resultado, Pendiente{campo, "circuito_pendiente"})
	}
	evaluacion := EvaluacionMaterialBases{Pendientes: resultado}
	if errCanonico == nil {
		evaluacion.ContenidoCanonico = &canonico
	}
	return evaluacion, nil
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
