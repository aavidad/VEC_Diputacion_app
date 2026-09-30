package separacionportales

import "testing"

func TestMaterialAvisosTienePropietarioExacto(t *testing.T) {
	for ruta, want := range map[string]pertenencia{"usuarios/avisos-externos.json": pertenenciaExterna, "bolsa/avisos-externos.json": pertenenciaInterna, "usuarios/avisos-internos.json": pertenenciaInterna, "usuarios/avisos-externos.key": pertenenciaProhibida} {
		if got := clasificar(ruta); got != want {
			t.Fatalf("ruta %s: %v", ruta, got)
		}
	}
}
func TestPoolAvisosNuncaEntraEnProcesoInterno(t *testing.T) {
	entorno := Entorno{Variables: []string{"VEC_EXTERNO_AVISOS_BOLSA_DATABASE_URL=postgresql://vec_externo_avisos_bolsa:sintetica@localhost/vec"}}
	if ComprobarEntorno(PortalInterno, entorno) == nil {
		t.Fatal("el interno no puede recibir la credencial del outbox consumidor")
	}
	if err := ComprobarEntorno(PortalExterno, entorno); err != nil {
		t.Fatal(err)
	}
}
