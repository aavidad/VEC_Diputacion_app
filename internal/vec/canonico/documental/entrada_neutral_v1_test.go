package documental

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonizarEntradaNeutralV1ConservaPreimagenHistorica(t *testing.T) {
	t.Parallel()

	canonico, valido := CanonizarEntradaNeutralV1(
		"Resolución", []string{"Uno", "Dos: con delimitador\ny unicode á"},
	)
	esperado := "62:vec.documentos.entrada-neutral.contenido-longitud-prefijada.v1\n" +
		"11:Resolución\n1:2\n3:Uno\n33:Dos: con delimitador\ny unicode á\n"
	if !valido || string(canonico) != esperado {
		t.Fatalf("preimagen neutral distinta: valida=%t canonico=%q", valido, canonico)
	}

	primero, _ := CanonizarEntradaNeutralV1("a", []string{"bc"})
	segundo, _ := CanonizarEntradaNeutralV1("ab", []string{"c"})
	if bytes.Equal(primero, segundo) {
		t.Fatal("el codec confundio particiones distintas de los mismos campos")
	}
	if vacio, valido := CanonizarEntradaNeutralV1("", []string{"parrafo"}); !valido || len(vacio) == 0 {
		t.Fatal("un titulo vacio con contenido dejo de pertenecer al dominio historico")
	}
	compuesto, _ := CanonizarEntradaNeutralV1("á", nil)
	descompuesto, _ := CanonizarEntradaNeutralV1("a\u0301", nil)
	if bytes.Equal(compuesto, descompuesto) {
		t.Fatal("el codec normalizo Unicode y confundio preimagenes distintas")
	}
}

func TestCanonizarEntradaNeutralV1RechazaFueraDelDominioCerrado(t *testing.T) {
	t.Parallel()

	for nombre, prueba := range map[string]struct {
		titulo   string
		parrafos []string
	}{
		"vacia":              {},
		"control titulo":     {titulo: "invalido\x00"},
		"control parrafo":    {titulo: "valido", parrafos: []string{"invalido\x7f"}},
		"utf8 titulo":        {titulo: string([]byte{0xff})},
		"utf8 parrafo":       {titulo: "valido", parrafos: []string{string([]byte{0xff})}},
		"numero de parrafos": {titulo: "valido", parrafos: make([]string, maximosParrafosEntradaNeutralV1+1)},
	} {
		prueba := prueba
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			if canonico, valido := CanonizarEntradaNeutralV1(prueba.titulo, prueba.parrafos); valido || canonico != nil {
				t.Fatalf("entrada invalida aceptada: valida=%t bytes=%d", valido, len(canonico))
			}
		})
	}

	if _, valido := CanonizarEntradaNeutralV1("tab\tlinea\nretorno\r", nil); !valido {
		t.Fatal("los tres controles expresamente admitidos fueron rechazados")
	}
}

func TestCanonizarEntradaNeutralV1ConservaFronterasExactas(t *testing.T) {
	// 66 bytes de esquema, 4 del contador y 10 de cabecera/cierre del titulo:
	// este contenido produce exactamente el maximo historico de 16 MiB.
	longitudTituloMaximo := maximoBytesEntradaNeutralV1 - 80
	titulo := strings.Repeat("x", longitudTituloMaximo+1)
	canonico, valido := CanonizarEntradaNeutralV1(titulo[:longitudTituloMaximo], nil)
	if !valido || len(canonico) != maximoBytesEntradaNeutralV1 ||
		!PreimagenEntradaNeutralV1Valida(titulo[:longitudTituloMaximo], nil, canonico, uint64(len(canonico))) {
		t.Fatalf("la frontera inclusiva de tamano cambio: valida=%t bytes=%d", valido, len(canonico))
	}
	canonico[len(canonico)-1] ^= 1
	if PreimagenEntradaNeutralV1Valida(titulo[:longitudTituloMaximo], nil, canonico, uint64(len(canonico))) ||
		PreimagenEntradaNeutralV1Valida(titulo[:longitudTituloMaximo], nil, canonico, uint64(len(canonico)-1)) ||
		PreimagenEntradaNeutralV1Valida(titulo[:longitudTituloMaximo], nil, nil, 0) ||
		PreimagenEntradaNeutralV1Valida(titulo[:longitudTituloMaximo], nil, canonico, ^uint64(0)) {
		t.Fatal("la validacion acepto preimagen alterada, tamano discordante o entrada vacia")
	}
	canonico[len(canonico)-1] ^= 1
	if canonico, valido = CanonizarEntradaNeutralV1(titulo, nil); valido || canonico != nil {
		t.Fatalf("se acepto un byte sobre el limite: valida=%t bytes=%d", valido, len(canonico))
	}

	parrafos := make([]string, maximosParrafosEntradaNeutralV1)
	if _, valido = CanonizarEntradaNeutralV1("titulo", parrafos); !valido {
		t.Fatal("la frontera inclusiva de numero de parrafos cambio")
	}
	parrafos = append(parrafos, "")
	if canonico, valido = CanonizarEntradaNeutralV1("titulo", parrafos); valido || canonico != nil {
		t.Fatal("se acepto un parrafo sobre el limite")
	}
}

func TestCanonizarEntradaNeutralV1ConservaPoliticaDeControlesYUTF8(t *testing.T) {
	for control := rune(0); control < 0x20; control++ {
		_, valido := CanonizarEntradaNeutralV1("x"+string(control), nil)
		esperado := control == '\t' || control == '\n' || control == '\r'
		if valido != esperado {
			t.Errorf("control U+%04X: valida=%t, esperado=%t", control, valido, esperado)
		}
	}
	if _, valido := CanonizarEntradaNeutralV1("x\x7f", nil); valido {
		t.Fatal("DEL dejo de rechazarse")
	}
	if _, valido := CanonizarEntradaNeutralV1("x\u0085", nil); !valido {
		t.Fatal("un control Unicode historicamente admitido cambio la compatibilidad")
	}
	if _, valido := CanonizarEntradaNeutralV1(string([]byte{'x', 0xff}), nil); valido {
		t.Fatal("se acepto UTF-8 invalido")
	}
}
