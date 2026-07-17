package reglasbaremo

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRestaurarVersionGobernadaRecorreTodosLosEstados(t *testing.T) {
	for nombre, original := range versionesGobernadasRestauracionPrueba(t) {
		t.Run(nombre, func(t *testing.T) {
			canonico, err := original.RepresentacionCanonica()
			if err != nil {
				t.Fatal(err)
			}
			huella, err := original.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}

			restaurada, err := RestaurarVersionGobernadaReglasBaremo(canonico)
			if err != nil {
				t.Fatal(err)
			}
			conHuella, err := RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
				canonico, huella,
			)
			if err != nil {
				t.Fatal(err)
			}
			comprobarVersionRestauradaIgual(t, restaurada, original, canonico)
			comprobarVersionRestauradaIgual(t, conHuella, original, canonico)

			canonico[0] = '['
			reproducido, err := restaurada.RepresentacionCanonica()
			if err != nil || reproducido[0] != '{' {
				t.Fatalf("la version retuvo los bytes recibidos: %v", err)
			}
		})
	}
}

func TestRestaurarVersionGobernadaRechazaJSONNoCanonico(t *testing.T) {
	canonico, _ := nuevaVersionGobiernoPrueba(
		t, instanteBaseGobiernoPrueba,
	).RepresentacionCanonica()
	conSangrado := &bytes.Buffer{}
	if err := jsonIndentacionGobierno(conSangrado, canonico); err != nil {
		t.Fatal(err)
	}

	casos := map[string][]byte{
		"campo_desconocido": insertarTrasLlave(
			canonico, `"secreto_super_sensible":"dato_privado",`,
		),
		"clave_duplicada": insertarTrasLlave(
			canonico, `"esquema":"vec.bolsa.gobierno-reglas-baremo.v1",`,
		),
		"espacio_inicial": append([]byte(" "), canonico...),
		"espacio_final":   append(append([]byte(nil), canonico...), '\n'),
		"sangrado":        conSangrado.Bytes(),
		"campos_reordenados": sustituirUnaVez(
			t, canonico,
			`"revision":1,"estado":"borrador"`,
			`"estado":"borrador","revision":1`,
		),
		"escape_equivalente": sustituirUnaVez(
			t, canonico, `"estado":"borrador"`, `"estado":"borrad\u006fr"`,
		),
		"segundo_valor": append(append([]byte(nil), canonico...), []byte(`{}`)...),
		"basura_final":  append(append([]byte(nil), canonico...), []byte("dato_privado")...),
		"utf8_invalido": append(append([]byte(nil), canonico...), 0xff),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := RestaurarVersionGobernadaReglasBaremo(contenido)
			if !errors.Is(err, ErrValorNoCanonico) {
				t.Fatalf("representacion hostil aceptada o error incorrecto: %v", err)
			}
			comprobarErrorSeguroYTipado(t, err)
		})
	}
}

func TestRestaurarVersionGobernadaRechazaEsquemaEstadoYRevision(t *testing.T) {
	activa := versionesGobernadasRestauracionPrueba(t)["activa"]
	canonico, _ := activa.RepresentacionCanonica()

	esquema := sustituirUnaVez(
		t, canonico,
		`"esquema":"vec.bolsa.gobierno-reglas-baremo.v1"`,
		`"esquema":"vec.bolsa.gobierno-reglas-baremo.v2"`,
	)
	if _, err := RestaurarVersionGobernadaReglasBaremo(esquema); !errors.Is(
		err, ErrEsquemaIncompatible,
	) {
		t.Fatalf("esquema incompatible aceptado: %v", err)
	}

	estado := sustituirUnaVez(t, canonico, `"estado":"activa"`, `"estado":"pendiente"`)
	if _, err := RestaurarVersionGobernadaReglasBaremo(estado); !errors.Is(
		err, ErrGobiernoEstadoInvalido,
	) {
		t.Fatalf("estado desconocido aceptado: %v", err)
	}

	revision := sustituirUnaVez(t, canonico, `"revision":3`, `"revision":2`)
	if _, err := RestaurarVersionGobernadaReglasBaremo(revision); !errors.Is(
		err, ErrGobiernoInvarianteQuebrada,
	) {
		t.Fatalf("revision incoherente aceptada: %v", err)
	}
}

func TestRestaurarVersionGobernadaVerificaHuellaYLimitePrevio(t *testing.T) {
	version := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	canonico, _ := version.RepresentacionCanonica()
	huella, _ := version.HuellaSHA256()

	_, err := RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		canonico, strings.Repeat("0", 64),
	)
	if !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("huella distinta aceptada: %v", err)
	}
	_, err = RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		canonico, strings.ToUpper(huella),
	)
	if !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella no canonica aceptada: %v", err)
	}

	demasiado := bytes.Repeat(
		[]byte("dato_privado"), maximoBytesGobiernoReglasBaremo/len("dato_privado")+2,
	)
	_, err = RestaurarVersionGobernadaReglasBaremo(demasiado)
	if !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("contenido sobredimensionado aceptado: %v", err)
	}
	comprobarErrorSeguroYTipado(t, err)
}

func comprobarVersionRestauradaIgual(
	t *testing.T,
	obtenida, esperada VersionGobernadaReglasBaremo,
	canonico []byte,
) {
	t.Helper()
	bytesObtenidos, err := obtenida.RepresentacionCanonica()
	if err != nil || !bytes.Equal(bytesObtenidos, canonico) {
		t.Fatalf("restauracion no reproduce los bytes: %v", err)
	}
	if obtenida.Estado() != esperada.Estado() || obtenida.Revision() != esperada.Revision() {
		t.Fatalf("estado/revision restaurados: %s/%d", obtenida.Estado(), obtenida.Revision())
	}
	conjunto, err := obtenida.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	conjunto.secciones[0] = SeccionBaremo{}
	despues, err := obtenida.RepresentacionCanonica()
	if err != nil || !bytes.Equal(despues, canonico) {
		t.Fatalf("una copia devuelta altero la version: %v", err)
	}
}

func versionesGobernadasRestauracionPrueba(
	t *testing.T,
) map[string]VersionGobernadaReglasBaremo {
	t.Helper()
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	publicada := publicarGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute))
	activa := activarGobiernoPrueba(t, publicada, instanteBaseGobiernoPrueba.Add(5*time.Minute))

	sucesora := referenciaPrueba(t, "reglas:convocatoria-2026:2", 2, 'a')
	autoridadSustitucion := autoridadGobiernoPrueba(
		t, activa, AccionSustituirReglasBaremo, actorGobiernoPrueba, &sucesora,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	sustituida, err := activa.Sustituir(
		3, actorGobiernoPrueba, motivoGobiernoPrueba(t, "sustitucion"), sucesora,
		autoridadSustitucion, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	autoridadRetirada := autoridadGobiernoPrueba(
		t, activa, AccionRetirarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	retirada, err := activa.Retirar(
		3, actorGobiernoPrueba, motivoGobiernoPrueba(t, "retirada"),
		autoridadRetirada, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	autoridadDescarte := autoridadGobiernoPrueba(
		t, borrador, AccionDescartarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(time.Minute),
	)
	descartada, err := borrador.Descartar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "descarte"),
		autoridadDescarte, instanteBaseGobiernoPrueba.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]VersionGobernadaReglasBaremo{
		"borrador": borrador, "publicada": publicada, "activa": activa,
		"sustituida": sustituida, "retirada": retirada, "descartada": descartada,
	}
}
