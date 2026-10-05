package bootstrap

import (
	"bytes"
	"context"
	"regexp"
	"testing"
	"time"
)

// AD188 rechaza un clave_id o un secreto ya publicados («material
// incompatible»). Dos publicaciones con secuencias distintas deben derivar
// claves distintas; repetir la misma secuencia, las mismas.
func TestGobiernoUsuariosClavesNuevasEnCadaPublicacion(t *testing.T) {
	formatoAD188 := regexp.MustCompile(`^clave:capacidad:admin:usuarios:[a-z0-9:._-]{1,120}$`)
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	type clave struct {
		id       string
		material []byte
	}
	derivar := func(secuencia uint64) []clave {
		t.Helper()
		c := cfg
		usuarios, _ := AudienciasConjuntoCapacidadesAdmin(0)
		c.Entradas = descriptoresClavesUsuariosAdmin(usuarios, secuencia, 4, 9, reloj.Ahora(), time.Hour)
		m, err := PrepararMaterialUsuariosAdmin(context.Background(), c, reloj)
		if err != nil {
			t.Fatal(err)
		}
		defer m.Cerrar()
		conf, _, err := m.Configuracion()
		if err != nil || len(conf.EntradasCapacidad) != 2 {
			t.Fatal("material incompleto")
		}
		var claves []clave
		for _, e := range conf.EntradasCapacidad {
			if !formatoAD188.MatchString(e.ClaveID) {
				t.Fatalf("clave_id fuera del formato de AD188: %s", e.ClaveID)
			}
			claves = append(claves, clave{e.ClaveID, bytes.Clone(e.Material)})
		}
		return claves
	}
	hoy, manana, repetida := derivar(20261006), derivar(20261007), derivar(20261006)
	for i := range hoy {
		if hoy[i].id == manana[i].id || bytes.Equal(hoy[i].material, manana[i].material) {
			t.Fatalf("la renovación repite clave_id o secreto: %s", hoy[i].id)
		}
		if hoy[i].id != repetida[i].id || !bytes.Equal(hoy[i].material, repetida[i].material) {
			t.Fatal("la misma secuencia no reproduce sus claves")
		}
	}
}
