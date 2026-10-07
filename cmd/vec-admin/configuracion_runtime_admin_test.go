package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeNominalExigeFuenteAprobadaYPoolSeparado(t *testing.T) {
	base := configuracionPrivadaPrueba(t)
	dir := filepath.Dir(base.Pools.CuentasADMIN)
	c := configuracionRuntimeADMIN{Version: 1, PoolContexto: filepath.Join(dir, "contexto.json"),
		FuenteIdentificadoresArchivo: filepath.Join(dir, "originales.json"),
		FuenteIdentificadoresSHA256:  strings.Repeat("a", 64), ProcesoContexto: "vec_admin"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	leida, err := cargarConfiguracionRuntimeADMIN(archivoPrivadoPrueba(t, string(b)), base)
	if err != nil || leida != c {
		t.Fatal("runtime privado no conservado")
	}
	for _, mutar := range []func(*configuracionRuntimeADMIN){
		func(x *configuracionRuntimeADMIN) { x.PoolContexto = base.Pools.CuentasADMIN },
		func(x *configuracionRuntimeADMIN) { x.PoolContexto = base.Pools.RegistroSesiones },
		func(x *configuracionRuntimeADMIN) { x.FuenteIdentificadoresSHA256 = strings.Repeat("0", 64) },
		func(x *configuracionRuntimeADMIN) { x.FuenteIdentificadoresArchivo = base.Firmante.ClavePrivadaArchivo },
		func(x *configuracionRuntimeADMIN) { x.ProcesoContexto = "" },
	} {
		alterada := c
		mutar(&alterada)
		if validarConfiguracionRuntimeADMIN(alterada, base) == nil {
			t.Fatal("runtime sin separación o fuente aprobada aceptado")
		}
	}
	for _, bruto := range []string{`{}`, `{"version":1,"version":1}`, `{"version":null}`} {
		if _, err := cargarConfiguracionRuntimeADMIN(archivoPrivadoPrueba(t, bruto), base); err == nil {
			t.Fatal("runtime ambiguo o incompleto aceptado")
		}
	}
}
