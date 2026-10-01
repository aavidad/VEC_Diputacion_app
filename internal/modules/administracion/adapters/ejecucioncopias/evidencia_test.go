package ejecucioncopias

import (
	"strings"
	"testing"
	cs06 "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func snapshotPrueba() cs06.Snapshot {
	s := cs06.Snapshot{Version: 1, PostgreSQL: "18.4", Completo: true}
	for _, clase := range []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto", "tablas", "secuencias", "objetos_grandes"} {
		if clase == "tablas" {
			s.Objetos = append(s.Objetos, cs06.Objeto{Clase: clase, Clave: "tabla:sintetica", Cantidad: 2, SHA256: strings.Repeat("1", 64)})
		}
		o := cs06.Objeto{Clase: clase, Clave: "inventario", SHA256: strings.Repeat("2", 64)}
		if clase == "tablas" || clase == "secuencias" || clase == "objetos_grandes" {
			o = cs06.Resumir(clase, s.Objetos)
		}
		s.Objetos = append(s.Objetos, o)
	}
	return s
}
func TestEvidenciaIncluyeContenidoYPrivilegiosDefaultAunqueRecuentoIgual(t *testing.T) {
	s := snapshotPrueba()
	archivos := []copias.Artefacto{{ID: "configuracion:sintetica", Tipo: "configuracion", SHA256: strings.Repeat("3", 64), TamanoBytes: 12}}
	original, err := evidenciaSnapshot(s, archivos, "origen:sintetico", "")
	if err != nil {
		t.Fatal(err)
	}
	cambiado := s
	cambiado.Objetos = append([]cs06.Objeto(nil), s.Objetos...)
	for n, o := range cambiado.Objetos {
		if o.Clase == "tablas" && o.Clave != "inventario" {
			cambiado.Objetos[n].SHA256 = strings.Repeat("4", 64)
		}
	}
	for n, o := range cambiado.Objetos {
		if o.Clase == "tablas" && o.Clave == "inventario" {
			cambiado.Objetos[n] = cs06.Resumir("tablas", cambiado.Objetos)
		}
	}
	otra, err := evidenciaSnapshot(cambiado, archivos, "ensayo:sintetico", "")
	if err != nil {
		t.Fatal(err)
	}
	if original.RecuentosSHA256 != otra.RecuentosSHA256 || original.ContenidoSHA256 == otra.ContenidoSHA256 {
		t.Fatal("contenido distinto quedó oculto por igualdad de recuentos")
	}
	cambiado = s
	cambiado.Objetos = append([]cs06.Objeto(nil), s.Objetos...)
	for n, o := range cambiado.Objetos {
		if o.Clase == "privilegios_defecto" {
			cambiado.Objetos[n].SHA256 = strings.Repeat("5", 64)
		}
	}
	otra, err = evidenciaSnapshot(cambiado, archivos, "ensayo:acl", "")
	if err != nil {
		t.Fatal(err)
	}
	if original.ACLSHA256 == otra.ACLSHA256 {
		t.Fatal("privilegios por defecto omitidos")
	}
	for left, right := 0, len(s.Objetos)-1; left < right; left, right = left+1, right-1 {
		s.Objetos[left], s.Objetos[right] = s.Objetos[right], s.Objetos[left]
	}
	reordenada, err := evidenciaSnapshot(s, archivos, "otra:ref", "")
	if err != nil || original.ContenidoSHA256 != reordenada.ContenidoSHA256 {
		t.Fatal("orden físico afectó al contraste")
	}
}
