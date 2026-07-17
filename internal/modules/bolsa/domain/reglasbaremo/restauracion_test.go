package reglasbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRestaurarConjuntoDesdeBytesCanonicos(t *testing.T) {
	original := conjuntoPrueba(t, true)
	contenido, err := original.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := original.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}

	restaurado, err := RestaurarConjuntoReglasBaremo(contenido)
	if err != nil {
		t.Fatal(err)
	}
	restauradoConHuella, err := RestaurarConjuntoReglasBaremoConHuellaSHA256(contenido, huella)
	if err != nil {
		t.Fatal(err)
	}
	bytesRestaurados, err := restaurado.RepresentacionCanonica()
	if err != nil || !bytes.Equal(bytesRestaurados, contenido) {
		t.Fatalf("restauracion no reproduce los bytes: %v", err)
	}
	bytesConHuella, err := restauradoConHuella.RepresentacionCanonica()
	if err != nil || !bytes.Equal(bytesConHuella, contenido) {
		t.Fatalf("restauracion con huella no reproduce los bytes: %v", err)
	}

	contenido[0] = '['
	bytesTrasMutacion, err := restaurado.RepresentacionCanonica()
	if err != nil || !bytes.Equal(bytesTrasMutacion, bytesRestaurados) {
		t.Fatalf("el agregado restaurado conserva los bytes recibidos: %v", err)
	}
}

func TestRestaurarRechazaJSONMaleableYHostil(t *testing.T) {
	canonico, err := conjuntoPrueba(t, true).RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}

	var generico any
	if err := json.Unmarshal(canonico, &generico); err != nil {
		t.Fatal(err)
	}
	clavesReordenadas, err := json.Marshal(generico)
	if err != nil || bytes.Equal(clavesReordenadas, canonico) {
		t.Fatalf("no se genero una representacion alternativa: %v", err)
	}
	conSangrado := &bytes.Buffer{}
	if err := json.Indent(conSangrado, canonico, "", "  "); err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre    string
		contenido []byte
	}{
		{
			"campo_desconocido",
			insertarTrasLlave(canonico, `"secreto_super_sensible":"dato_privado",`),
		},
		{
			"clave_duplicada",
			insertarTrasLlave(canonico, `"esquema":"vec.bolsa.conjunto_reglas_baremo.v1",`),
		},
		{"espacio_inicial", append([]byte(" "), canonico...)},
		{"espacio_final", append(append([]byte(nil), canonico...), '\n')},
		{"sangrado", conSangrado.Bytes()},
		{"claves_reordenadas", clavesReordenadas},
		{
			"escape_unicode_equivalente",
			sustituirUnaVez(t, canonico, `"unidad_base":"dia"`, `"unidad_base":"\u0064ia"`),
		},
		{"segundo_valor", append(append([]byte(nil), canonico...), []byte(`{}`)...)},
		{"segundo_nulo", append(append([]byte(nil), canonico...), []byte(`null`)...)},
		{"basura_final", append(append([]byte(nil), canonico...), []byte(`dato_privado`)...)},
		{"utf8_invalido", append(append([]byte(nil), canonico...), 0xff)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := RestaurarConjuntoReglasBaremo(caso.contenido)
			if !errors.Is(err, ErrValorNoCanonico) {
				t.Fatalf("representacion hostil aceptada o error incorrecto: %v", err)
			}
			comprobarErrorSeguroYTipado(t, err)
		})
	}
}

func TestRestaurarRechazaEsquemaDistinto(t *testing.T) {
	canonico, _ := conjuntoPrueba(t, true).RepresentacionCanonica()
	alterado := sustituirUnaVez(
		t,
		canonico,
		`"esquema":"vec.bolsa.conjunto_reglas_baremo.v1"`,
		`"esquema":"vec.bolsa.conjunto_reglas_baremo.v2"`,
	)
	_, err := RestaurarConjuntoReglasBaremo(alterado)
	if !errors.Is(err, ErrEsquemaIncompatible) {
		t.Fatalf("esquema distinto aceptado: %v", err)
	}
	comprobarErrorSeguroYTipado(t, err)
}

func TestRestaurarRechazaModosValoresYPunterosIncoherentes(t *testing.T) {
	canonico, _ := conjuntoPrueba(t, true).RepresentacionCanonica()
	casos := []struct {
		nombre   string
		esperado error
		buscar   string
		poner    string
	}{
		{
			"modo_jornada_desconocido", ErrPoliticaIncompleta,
			`"jornada":{"modo":"proporcional"}`,
			`"jornada":{"modo":"variable"}`,
		},
		{
			"umbral_con_jornada_sin_umbral", ErrPoliticaIncompleta,
			`"jornada":{"modo":"proporcional"}`,
			`"jornada":{"modo":"proporcional","umbral":"1/2"}`,
		},
		{
			"limite_con_solape_no_acumulable", ErrPoliticaIncompleta,
			`"solape":{"modo":"elegir_mayor_puntuacion"}`,
			`"solape":{"modo":"elegir_mayor_puntuacion","limite":"1/1"}`,
		},
		{
			"limite_declarado_sin_valor", ErrPoliticaIncompleta,
			`"maximo_puntos":{"modo":"limitado","valor":"10000000"}`,
			`"maximo_puntos":{"modo":"limitado"}`,
		},
		{
			"sin_limite_con_valor", ErrPoliticaIncompleta,
			`"maximo_puntos":{"modo":"limitado","valor":"10000000"}`,
			`"maximo_puntos":{"modo":"sin_limite","valor":"10000000"}`,
		},
		{
			"puntos_como_numero_json", ErrValorNoCanonico,
			`"puntos_por_unidad":"550000"`,
			`"puntos_por_unidad":550000`,
		},
		{
			"racional_no_reducido", ErrValorNoCanonico,
			`"unidades_base_por_unidad":"30/1"`,
			`"unidades_base_por_unidad":"60/2"`,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := sustituirUnaVez(t, canonico, caso.buscar, caso.poner)
			_, err := RestaurarConjuntoReglasBaremo(alterado)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("valor incoherente aceptado o error incorrecto: %v", err)
			}
			comprobarErrorSeguroYTipado(t, err)
		})
	}
}

func TestRestaurarVerificaHuellaEsperada(t *testing.T) {
	conjunto := conjuntoPrueba(t, true)
	canonico, _ := conjunto.RepresentacionCanonica()
	huella, _ := conjunto.HuellaSHA256()

	_, err := RestaurarConjuntoReglasBaremoConHuellaSHA256(canonico, strings.Repeat("0", 64))
	if !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("huella distinta aceptada: %v", err)
	}
	comprobarErrorSeguroYTipado(t, err)

	_, err = RestaurarConjuntoReglasBaremoConHuellaSHA256(canonico, strings.ToUpper(huella))
	if !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella no canonica aceptada: %v", err)
	}
	comprobarErrorSeguroYTipado(t, err)
}

func TestRestaurarCompruebaLimiteAntesDeDecodificar(t *testing.T) {
	contenido := bytes.Repeat([]byte("dato_privado"), maximoBytesRepresentacion/len("dato_privado")+2)
	if len(contenido) <= maximoBytesRepresentacion {
		t.Fatal("la prueba no supera el limite")
	}
	_, err := RestaurarConjuntoReglasBaremo(contenido)
	if !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("contenido sobredimensionado no rechazado por limite: %v", err)
	}
	comprobarErrorSeguroYTipado(t, err)
}

func insertarTrasLlave(contenido []byte, insercion string) []byte {
	resultado := make([]byte, 0, len(contenido)+len(insercion))
	resultado = append(resultado, '{')
	resultado = append(resultado, insercion...)
	resultado = append(resultado, contenido[1:]...)
	return resultado
}

func sustituirUnaVez(t *testing.T, contenido []byte, buscar, poner string) []byte {
	t.Helper()
	if bytes.Count(contenido, []byte(buscar)) == 0 {
		t.Fatalf("no se encontro el fragmento de prueba: %s", buscar)
	}
	return bytes.Replace(contenido, []byte(buscar), []byte(poner), 1)
}

func comprobarErrorSeguroYTipado(t *testing.T, err error) {
	t.Helper()
	var errorModelo *ErrorModelo
	if !errors.As(err, &errorModelo) {
		t.Fatalf("error no tipado: %T: %v", err, err)
	}
	if strings.Contains(err.Error(), "dato_privado") ||
		strings.Contains(err.Error(), "secreto_super_sensible") {
		t.Fatalf("el error expone la entrada: %v", err)
	}
}
