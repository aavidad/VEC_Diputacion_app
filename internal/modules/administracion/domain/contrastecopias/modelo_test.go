package contrastecopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func hashPrueba(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func evidencia() Snapshot {
	s := Snapshot{Version: 1, PostgreSQL: "18.4", Completo: true}
	for _, c := range clases {
		if !individual(c) {
			s.Objetos = append(s.Objetos, Objeto{c, "inventario", 1, hashPrueba(c)})
		}
	}
	s.Objetos = append(s.Objetos, Objeto{"tablas", `"prueba"."contenido"`, 2, hashPrueba("dos filas")})
	s.Objetos = append(s.Objetos, Objeto{"secuencias", `"prueba"."contador"`, 1, hashPrueba("3 true")})
	for _, c := range clases {
		if individual(c) {
			s.Objetos = append(s.Objetos, Resumir(c, s.Objetos))
		}
	}
	return s
}

func TestMismoRecuentoContenidoDistinto(t *testing.T) {
	a, b := evidencia(), evidencia()
	for i, o := range b.Objetos {
		if o.Clase == "tablas" && o.Clave != "inventario" {
			b.Objetos[i].SHA256 = hashPrueba("una celda ha cambiado")
		}
	}
	for i, o := range b.Objetos {
		if o.Clase == "tablas" && o.Clave == "inventario" {
			b.Objetos[i] = Resumir("tablas", b.Objetos)
		}
	}
	r := Comparar(a, b)
	if r.Estado != Diferente {
		t.Fatalf("mismo count no puede ocultar contenido diferente: %+v", r)
	}
	diag, _ := json.Marshal(r)
	if strings.Contains(string(diag), "contenido\"") || strings.Contains(string(diag), a.Objetos[5].SHA256) {
		t.Fatal("diagnóstico contiene datos privados")
	}
}

func TestOrdenNoFormaParteDeLaIgualdad(t *testing.T) {
	a, b := evidencia(), evidencia()
	for i, j := 0, len(b.Objetos)-1; i < j; i, j = i+1, j-1 {
		b.Objetos[i], b.Objetos[j] = b.Objetos[j], b.Objetos[i]
	}
	if r := Comparar(a, b); r.Estado != Igual || len(r.Razones) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestFallaCerradoInventario(t *testing.T) {
	casos := map[string]func(*Snapshot){
		"incompleto":    func(s *Snapshot) { s.Completo = false },
		"motivo":        func(s *Snapshot) { s.Motivos = []string{"objeto_desconocido"} },
		"clase omitida": func(s *Snapshot) { s.Objetos = s.Objetos[1:] },
		"tabla omitida": func(s *Snapshot) { s.Objetos = append(s.Objetos[:5], s.Objetos[6:]...) },
		"desconocido":   func(s *Snapshot) { s.Objetos = append(s.Objetos, Objeto{"nuevo", "inventario", 0, hashPrueba("")}) },
		"duplicado":     func(s *Snapshot) { s.Objetos = append(s.Objetos, s.Objetos[0]) },
		"hash":          func(s *Snapshot) { s.Objetos[0].SHA256 = strings.Repeat("f", 63) },
		"cantidad":      func(s *Snapshot) { s.Objetos[0].Cantidad = -1 },
		"version":       func(s *Snapshot) { s.Version = 2 },
		"postgresql":    func(s *Snapshot) { s.PostgreSQL = "17.6" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, b := evidencia(), evidencia()
			mutar(&b)
			if Comparar(a, b).Estado != NoComprobable {
				t.Fatal("inventario malformado aceptado")
			}
		})
	}
}

func TestDiferenciasPorClase(t *testing.T) {
	for _, c := range []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto", "secuencias", "objetos_grandes"} {
		t.Run(c, func(t *testing.T) {
			a, b := evidencia(), evidencia()
			if c == "objetos_grandes" {
				b.Objetos = append(b.Objetos, Objeto{c, "objeto:1", 7, hashPrueba("bytes")})
			} else {
				for i, o := range b.Objetos {
					if o.Clase == c && (o.Clave != "inventario" || !individual(c)) {
						b.Objetos[i].SHA256 = hashPrueba("distinto")
					}
				}
			}
			if individual(c) {
				for i, o := range b.Objetos {
					if o.Clase == c && o.Clave == "inventario" {
						b.Objetos[i] = Resumir(c, b.Objetos)
					}
				}
			}
			if r := Comparar(a, b); r.Estado != Diferente {
				t.Fatalf("%+v", r)
			}
		})
	}
}
