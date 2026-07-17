package calculoexperiencia

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestResultadoExperienciaV1CanonHuellaYRestauracionExactos(t *testing.T) {
	casos := []struct {
		nombre    string
		resultado ResultadoExperienciaV1
	}{
		{"completado", resultadoCompletadoPrueba(t)},
		{"bloqueado", resultadoBloqueadoSeleccionPrueba(t)},
		{"grande", resultadoGrandeRescatadoPrueba(t)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			contenido, err := caso.resultado.RepresentacionCanonica()
			if err != nil {
				t.Fatal(err)
			}
			huella, err := caso.resultado.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			restaurado, err := RestaurarResultadoExperienciaV1ConHuellaSHA256(contenido, huella)
			if err != nil {
				t.Fatal(err)
			}
			restauradoCanonico, err := restaurado.RepresentacionCanonica()
			if err != nil || !bytes.Equal(contenido, restauradoCanonico) {
				t.Fatalf("round-trip no exacto: %v", err)
			}
		})
	}
}

func TestResultadoExperienciaV1CanonNoRepiteDatosLaboralesInnecesarios(t *testing.T) {
	contenido, err := resultadoCompletadoPrueba(t).RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	texto := strings.ToLower(string(contenido))
	for _, prohibido := range []string{
		"servicio_ref", "atributos", "valor_catalogado", "dni", "nif", "nie",
		"nombre", "apellido", "correo", "diagnostico", "causa", "direccion", "iban",
	} {
		if strings.Contains(texto, prohibido) {
			t.Errorf("la salida contiene %q", prohibido)
		}
	}
	for _, obligatorio := range []string{
		`"fecha_corte":"2026-12-31"`, `"plan":`, `"conjunto":`,
		`"instantanea":`, `"huella_contenido_sha256":`, `"motor":`,
	} {
		if !strings.Contains(texto, obligatorio) {
			t.Errorf("falta %s", obligatorio)
		}
	}
}

func TestRestaurarResultadoExperienciaV1RechazaRedondeoManipulado(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t)
	material := materializarResultadoExperienciaV1(resultado)
	material.Reglas[0].Redondeo.Salida = "1000001/1"
	material.Reglas[0].TopePuntos.Antes = "1000001/1"
	material.Reglas[0].TopePuntos.Despues = "1000001/1"
	material.Reglas[0].PuntosFinales = "1000001/1"
	material.Secciones[0].AntesTope = "1000001/1"
	material.Secciones[0].Tope.Antes = "1000001/1"
	material.Secciones[0].Tope.Despues = "1000001/1"
	puntos, _ := baremacion.PuntosDesdeMicropuntos(1_000_001)
	material.Secciones[0].PuntosFinales = puntos
	material.Total = &puntos
	contenido, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarResultadoExperienciaV1(contenido); err == nil {
		t.Fatal("se acepto un redondeo aritmeticamente falso con subtotales cuadrados")
	}
}

func TestRestaurarResultadoExperienciaV1RechazaIntervaloManipulado(t *testing.T) {
	material := materializarResultadoExperienciaV1(resultadoCompletadoPrueba(t))
	material.Intervalos[0].Efectivo.Dias++
	contenido, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarResultadoExperienciaV1(contenido); err == nil {
		t.Fatal("se aceptaron dias distintos del intervalo normalizado")
	}

	material = materializarResultadoExperienciaV1(resultadoCompletadoPrueba(t))
	material.Vinculos.FechaCorte = fechaResultadoPrueba(t, 2025, 12, 31)
	contenido, err = json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarResultadoExperienciaV1(contenido); err == nil {
		t.Fatal("se acepto un intervalo incompatible con la fecha de corte")
	}
}

func TestRestaurarResultadoExperienciaV1RechazaMutacionesCanonicas(t *testing.T) {
	contenido, err := resultadoCompletadoPrueba(t).RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	casos := [][]byte{
		append([]byte(" "), contenido...),
		append(append([]byte(nil), contenido...), '\n'),
		bytes.Replace(contenido, []byte(`"estado":"completado"`), []byte(`"estado":"desconocido"`), 1),
		bytes.Replace(contenido, []byte(`"esquema":"vec.bolsa.resultado_experiencia.v1"`),
			[]byte(`"esquema":"vec.bolsa.resultado_experiencia.v2"`), 1),
		bytes.Replace(contenido, []byte(`"fase":"completado"`),
			[]byte(`"fase":"seleccion"`), 1),
		bytes.Replace(contenido, []byte(`"estado":"completado"`),
			[]byte(`"estado":"completado","estado":"completado"`), 1),
		bytes.Replace(contenido, []byte(`"fase":"completado"`),
			[]byte(`"fase":"completado","campo_desconocido":1`), 1),
	}
	for indice, caso := range casos {
		if _, err := RestaurarResultadoExperienciaV1(caso); err == nil {
			t.Errorf("mutacion %d aceptada", indice)
		}
	}
	huella, _ := resultadoCompletadoPrueba(t).HuellaSHA256()
	huellaFalsa := "0" + huella[1:]
	if huellaFalsa == huella {
		huellaFalsa = "1" + huella[1:]
	}
	if _, err := RestaurarResultadoExperienciaV1ConHuellaSHA256(contenido, huellaFalsa); err == nil {
		t.Fatal("se acepto una huella distinta")
	}
}

func TestRestaurarResultadoExperienciaV1LimitaAntesDeMaterializar(t *testing.T) {
	elementos := bytes.Repeat([]byte("{},"), maximoSeccionesResultadoV1+1)
	elementos = elementos[:len(elementos)-1]
	datos := append([]byte("["), elementos...)
	datos = append(datos, ']')
	var secciones materialesSeccionesResultadoV1
	if err := json.Unmarshal(datos, &secciones); err == nil {
		t.Fatal("se materializaron demasiadas secciones")
	}

	demasiadoGrande := make([]byte, maximoBytesResultadoV1+1)
	if _, err := RestaurarResultadoExperienciaV1(demasiadoGrande); err == nil {
		t.Fatal("se acepto contenido superior al presupuesto")
	}
}
