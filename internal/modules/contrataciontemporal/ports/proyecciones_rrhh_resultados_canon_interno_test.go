package ports

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncuadreCanonResultadoCuentaOctetosUTF8(t *testing.T) {
	t.Parallel()
	constructor := nuevoConstructorCanonResultadoRRHH("VECTOR-UTF8\n")
	constructor.texto("Área_Ñ")
	obtenido, err := constructor.finalizar()
	if err != nil {
		t.Fatal(err)
	}
	esperado := []byte("VECTOR-UTF8\n8:Área_Ñ\n")
	if !bytes.Equal(obtenido, esperado) {
		t.Fatalf("encuadre UTF-8 divergente: %q != %q", obtenido, esperado)
	}
}

func TestConstructorCanonResultadoRechazaAntesDeSuperarPresupuesto(
	t *testing.T,
) {
	t.Parallel()
	constructor := nuevoConstructorCanonResultadoRRHH("")
	constructor.crudo(make([]byte, LimiteMaximoCanonResultadoRRHH))
	if len(constructor.bytesCanonicos) != LimiteMaximoCanonResultadoRRHH {
		t.Fatalf("el borde exacto no fue aceptado: %d", len(constructor.bytesCanonicos))
	}
	constructor.crudo([]byte{1})
	if len(constructor.bytesCanonicos) != LimiteMaximoCanonResultadoRRHH {
		t.Fatal("el constructor anexó bytes después del límite")
	}
	if _, err := constructor.finalizar(); !errors.Is(
		err, ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("el exceso no dejó el constructor en fallo cerrado: %v", err)
	}
}

func TestEncuadreCanonResultadoNoAnexaTextoQueExcedePresupuesto(
	t *testing.T,
) {
	t.Parallel()
	constructor := nuevoConstructorCanonResultadoRRHH("BASE\n")
	longitudInicial := len(constructor.bytesCanonicos)
	constructor.texto(string(make(
		[]byte,
		LimiteMaximoCanonResultadoRRHH,
	)))
	if len(constructor.bytesCanonicos) != longitudInicial {
		t.Fatal("un texto excesivo fue anexado parcialmente")
	}
	if _, err := constructor.finalizar(); !errors.Is(
		err, ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("el texto excesivo no falló cerrado: %v", err)
	}
}
