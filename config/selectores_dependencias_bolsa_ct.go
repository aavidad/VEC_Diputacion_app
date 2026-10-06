package config

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
)

// Inventario versionado de las variables de entorno de Contratación temporal
// y Bolsa (y de las comunes de las que dependen), con lo que cada una
// necesita cuando está activa. Es solo diagnóstico: no decide qué se compone
// ni sustituye a las validaciones que detienen el arranque.

//go:embed selectores_dependencias_bolsa_ct_v1.json
var inventarioSelectoresBolsaCTV1 []byte

// Tipos de variable del inventario.
const (
	TipoVariableSelector = "selector"
	TipoVariableConexion = "conexion"
	TipoVariableCatalogo = "catalogo"
	TipoVariableAjuste   = "ajuste"
	TipoVariableComun    = "comun"
)

var ErrInventarioSelectoresBolsaCTInvalido = errors.New("config: inventario de selectores de Bolsa y CT invalido")

// VariableEntornoBolsaCT describe una variable del inventario.
type VariableEntornoBolsaCT struct {
	Variable string   `json:"variable"`
	Tipo     string   `json:"tipo"`
	UsoReal  bool     `json:"uso_real"`
	Requiere []string `json:"requiere,omitempty"`
	Nota     string   `json:"nota,omitempty"`
}

// InventarioSelectoresBolsaCT es el fichero de datos ya validado.
type InventarioSelectoresBolsaCT struct {
	Version     int                      `json:"version"`
	Descripcion string                   `json:"descripcion"`
	Variables   []VariableEntornoBolsaCT `json:"variables"`
}

// DependenciaSelectorAusente nombra una variable activa y la que le falta.
// Nunca lleva valores: solo nombres de variables.
type DependenciaSelectorAusente struct {
	Variable string
	Falta    string
}

// CargarInventarioSelectoresBolsaCT decodifica y valida el inventario
// embebido. Devuelve una copia: el llamante puede modificarla sin efectos.
func CargarInventarioSelectoresBolsaCT() (InventarioSelectoresBolsaCT, error) {
	return decodificarInventarioSelectoresBolsaCT(inventarioSelectoresBolsaCTV1)
}

func decodificarInventarioSelectoresBolsaCT(datos []byte) (InventarioSelectoresBolsaCT, error) {
	var inv InventarioSelectoresBolsaCT
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&inv); err != nil || inv.Version != 1 || len(inv.Variables) == 0 {
		return InventarioSelectoresBolsaCT{}, ErrInventarioSelectoresBolsaCTInvalido
	}
	vistas := make(map[string]bool, len(inv.Variables))
	for _, v := range inv.Variables {
		if !strings.HasPrefix(v.Variable, "VEC_") || vistas[v.Variable] {
			return InventarioSelectoresBolsaCT{}, ErrInventarioSelectoresBolsaCTInvalido
		}
		switch v.Tipo {
		case TipoVariableSelector, TipoVariableConexion, TipoVariableCatalogo, TipoVariableAjuste, TipoVariableComun:
		default:
			return InventarioSelectoresBolsaCT{}, ErrInventarioSelectoresBolsaCTInvalido
		}
		vistas[v.Variable] = true
	}
	for _, v := range inv.Variables {
		for _, r := range v.Requiere {
			if !vistas[r] || r == v.Variable {
				return InventarioSelectoresBolsaCT{}, ErrInventarioSelectoresBolsaCTInvalido
			}
		}
	}
	return inv, nil
}

// DependenciasAusentes recorre el inventario con el lector de entorno dado y
// devuelve, en el orden del fichero, cada dependencia que falta a una
// variable activa. Un selector cuenta como activo o satisfecho solo con
// "true"; cualquier otra variable, con un valor no vacío.
func (inv InventarioSelectoresBolsaCT) DependenciasAusentes(entorno func(string) string) []DependenciaSelectorAusente {
	if entorno == nil {
		return nil
	}
	tipos := make(map[string]string, len(inv.Variables))
	for _, v := range inv.Variables {
		tipos[v.Variable] = v.Tipo
	}
	puesta := func(nombre string) bool {
		valor := strings.TrimSpace(entorno(nombre))
		if tipos[nombre] == TipoVariableSelector {
			return valor == "true"
		}
		return valor != ""
	}
	var ausentes []DependenciaSelectorAusente
	for _, v := range inv.Variables {
		if len(v.Requiere) == 0 || !puesta(v.Variable) {
			continue
		}
		for _, r := range v.Requiere {
			if !puesta(r) {
				ausentes = append(ausentes, DependenciaSelectorAusente{Variable: v.Variable, Falta: r})
			}
		}
	}
	return ausentes
}
