package main

import (
	"path/filepath"
	"testing"
)

func TestOverlayExigeDecimoPoolTecnicoPrivadoDistinto(t *testing.T) {
	base := configuracionPrivadaPrueba(t)
	for _, ruta := range []string{"", base.Pools.CuentasADMIN, "relativo.json"} {
		u := usuariosMetadatosPrueba(t, base)
		u.PoolFronteraTecnica = ruta
		if validarConfiguracionUsuariosMetadatosPrivada(u, base) == nil {
			t.Fatal("pool_tecnico_no_exclusivo")
		}
	}
	u := usuariosMetadatosPrueba(t, base)
	u.PoolFronteraTecnica = filepath.Join(t.TempDir(), "frontera.json")
	if validarConfiguracionUsuariosMetadatosPrivada(u, base) != nil {
		t.Fatal("pool_tecnico_privado_rechazado")
	}
}
