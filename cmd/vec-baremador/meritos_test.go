package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
)

func TestCLIMeritosDosReglasYRechazo(t *testing.T) {
	for _, caso := range []struct{ reglas, entrada, estado, total string }{{"meritos_reglas_a", "meritos_entrada", "completado", "4000000"}, {"meritos_reglas_b", "meritos_entrada", "completado", "6000000"}, {"meritos_reglas_a", "meritos_bloqueada", "bloqueado", ""}} {
		args := append(argsPrueba(t, caso.reglas, caso.entrada), "--modo", "meritos")
		var salida, diagnostico bytes.Buffer
		if code := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); code != 0 || diagnostico.Len() != 0 {
			t.Fatalf("%d %s", code, diagnostico.String())
		}
		var v struct {
			Resultado struct{ Estado, Total string }
		}
		if err := json.Unmarshal(salida.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v.Resultado.Estado != caso.estado || v.Resultado.Total != caso.total {
			t.Fatal("salida equivoca")
		}
	}
	for _, modo := range []string{"desconocido", "meritos", "experiencia"} {
		args := append(argsPrueba(t, "reglas_a", "entrada"), "--modo", modo)
		if modo == "experiencia" {
			args = append(argsPrueba(t, "meritos_reglas_a", "meritos_entrada"), "--modo", modo)
		}
		var salida, diagnostico bytes.Buffer
		if code := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); code != 2 || salida.Len() != 0 {
			t.Fatal("modo o esquema cruzado admitido")
		}
	}
}
