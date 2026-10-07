// Package contrastecopias compara evidencia lógica; no restaura ni autoriza copias.
package contrastecopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
)

const (
	Igual         = "igual"
	Diferente     = "diferente"
	NoComprobable = "no_comprobable"
)

// Snapshot es evidencia privada. Sus nombres y huellas no son una vista ADMIN.
// Completo solo puede afirmarse después de inventariar todas las clases admitidas.
type Snapshot struct {
	Version    int      `json:"version"`
	PostgreSQL string   `json:"postgresql"`
	Completo   bool     `json:"completo"`
	Objetos    []Objeto `json:"objetos"`
	Motivos    []string `json:"motivos"`
}

type Objeto struct {
	Clase    string `json:"clase"`
	Clave    string `json:"clave"`
	Cantidad int64  `json:"cantidad"`
	SHA256   string `json:"sha256"`
}

// Razon no revela nombres privados, contenido, rutas ni huellas de celdas.
type Razon struct {
	Codigo     string `json:"codigo"`
	Clase      string `json:"clase,omitempty"`
	Referencia string `json:"referencia,omitempty"`
}

type Resultado struct {
	Estado  string  `json:"estado"`
	Razones []Razon `json:"razones"`
}

var versionPG = regexp.MustCompile(`^18\.[0-9]+$`)
var shaCanonico = regexp.MustCompile(`^[0-9a-f]{64}$`)

var clases = []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto", "tablas", "secuencias", "objetos_grandes"}

func individual(clase string) bool {
	return clase == "tablas" || clase == "secuencias" || clase == "objetos_grandes"
}

func conocida(clase string) bool {
	for _, c := range clases {
		if c == clase {
			return true
		}
	}
	return false
}

// Resumir sella las entradas individuales de una clase como conjunto ordenado.
// La entrada inventario nunca forma parte de su propia huella.
func Resumir(clase string, objetos []Objeto) Objeto {
	lista := make([]Objeto, 0)
	for _, o := range objetos {
		if o.Clase == clase && o.Clave != "inventario" {
			lista = append(lista, o)
		}
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Clave < lista[j].Clave })
	b, _ := json.Marshal(lista)
	h := sha256.Sum256(b)
	return Objeto{Clase: clase, Clave: "inventario", Cantidad: int64(len(lista)), SHA256: hex.EncodeToString(h[:])}
}

func identidad(o Objeto) string {
	b, _ := json.Marshal([]string{o.Clase, o.Clave})
	return string(b)
}

func referencia(o Objeto) string {
	h := sha256.Sum256([]byte(identidad(o)))
	return hex.EncodeToString(h[:])
}

// Validar rechaza snapshots incompletos y clases que el formato no describe.
// Una declaración offline no sustituye procedencia y autenticación externas.
func Validar(s Snapshot) []Razon {
	var rs []Razon
	if s.Version != 1 || !versionPG.MatchString(s.PostgreSQL) {
		rs = append(rs, Razon{Codigo: "formato_no_admitido"})
	}
	if !s.Completo || len(s.Motivos) != 0 {
		rs = append(rs, Razon{Codigo: "inventario_incompleto"})
	}
	if len(s.Objetos) > 100000 {
		return append(rs, Razon{Codigo: "limite_objetos"})
	}
	vistos := map[string]bool{}
	inventarios := map[string]Objeto{}
	for _, o := range s.Objetos {
		if !conocida(o.Clase) || o.Clave == "" || len(o.Clave) > 1024 || o.Cantidad < 0 || !shaCanonico.MatchString(o.SHA256) || (!individual(o.Clase) && o.Clave != "inventario") {
			rs = append(rs, Razon{Codigo: "objeto_no_admitido"})
			continue
		}
		id := identidad(o)
		if vistos[id] {
			rs = append(rs, Razon{Codigo: "objeto_duplicado", Clase: o.Clase, Referencia: referencia(o)})
		}
		vistos[id] = true
		if o.Clave == "inventario" {
			inventarios[o.Clase] = o
		}
	}
	for _, c := range clases {
		o, ok := inventarios[c]
		if !ok {
			rs = append(rs, Razon{Codigo: "clase_ausente", Clase: c})
			continue
		}
		if individual(c) && o != Resumir(c, s.Objetos) {
			rs = append(rs, Razon{Codigo: "inventario_incoherente", Clase: c})
		}
	}
	return ordenar(rs)
}

func ordenar(rs []Razon) []Razon {
	sort.Slice(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		if a.Codigo != b.Codigo {
			return a.Codigo < b.Codigo
		}
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.Referencia < b.Referencia
	})
	result := make([]Razon, 0, len(rs))
	for _, r := range rs {
		if len(result) == 0 || result[len(result)-1] != r {
			result = append(result, r)
		}
	}
	return result
}

// Comparar conserva duplicados de datos a través de la huella por tabla.
// La igualdad se limita al contenido y objetos descritos por este contrato.
func Comparar(a, b Snapshot) Resultado {
	rs := append(Validar(a), Validar(b)...)
	if len(rs) > 0 {
		return Resultado{NoComprobable, ordenar(rs)}
	}
	if a.PostgreSQL != b.PostgreSQL {
		rs = append(rs, Razon{Codigo: "version_postgresql_diferente"})
	}
	observados := make(map[string]Objeto, len(b.Objetos))
	for _, o := range b.Objetos {
		observados[identidad(o)] = o
	}
	for _, o := range a.Objetos {
		x, ok := observados[identidad(o)]
		if !ok {
			rs = append(rs, Razon{Codigo: "objeto_ausente", Clase: o.Clase, Referencia: referencia(o)})
		} else {
			if o.Cantidad != x.Cantidad {
				rs = append(rs, Razon{Codigo: "cantidad_diferente", Clase: o.Clase, Referencia: referencia(o)})
			}
			if o.SHA256 != x.SHA256 {
				rs = append(rs, Razon{Codigo: "contenido_diferente", Clase: o.Clase, Referencia: referencia(o)})
			}
		}
		delete(observados, identidad(o))
	}
	for _, o := range observados {
		rs = append(rs, Razon{Codigo: "objeto_adicional", Clase: o.Clase, Referencia: referencia(o)})
	}
	estado := Igual
	if len(rs) > 0 {
		estado = Diferente
	}
	return Resultado{estado, ordenar(rs)}
}
