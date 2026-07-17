package calculoexperienciaoficial

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestIntencionLigaClaveResultadoEstadoYFase(t *testing.T) {
	base := intencionPrueba(t)
	huellaBase, _ := base.HuellaSHA256()
	claveDistintaDatos := datosClavePrueba()
	claveDistintaDatos.Convocatoria.Version++
	claveDistinta, _ := NuevaClaveEfectoV1(claveDistintaDatos)
	variantes := []IntencionResultadoV1{
		crearIntencion(t, claveDistinta, huellaPrueba("a"), ResultadoCompletado, FaseCompletado),
		crearIntencion(t, clavePrueba(t), huellaPrueba("b"), ResultadoCompletado, FaseCompletado),
		crearIntencion(t, clavePrueba(t), huellaPrueba("a"), ResultadoBloqueado, FaseSeleccion),
		crearIntencion(t, clavePrueba(t), huellaPrueba("a"), ResultadoBloqueado, FaseIntervalos),
	}
	for indice, variante := range variantes {
		huella, _ := variante.HuellaSHA256()
		if huella == huellaBase {
			t.Fatalf("variante semántica %d comparte intención", indice)
		}
	}
	if err := base.ValidarPara(
		base.Clave(), base.HuellaResultadoSHA256(), base.Estado(), base.Fase(),
	); err != nil {
		t.Fatalf("la intención no coincide consigo misma: %v", err)
	}
	if err := base.ValidarPara(
		base.Clave(), huellaPrueba("b"), base.Estado(), base.Fase(),
	); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("resultado ajeno aceptado: %v", err)
	}
}

func TestIntencionEstadoFaseSonCoherentes(t *testing.T) {
	casos := []struct {
		estado EstadoResultadoV1
		fase   FaseResultadoV1
	}{
		{ResultadoCompletado, FaseSeleccion},
		{ResultadoBloqueado, FaseCompletado},
		{"desconocido", FasePuntuacion},
		{ResultadoBloqueado, "desconocida"},
	}
	for _, caso := range casos {
		_, err := NuevaIntencionResultadoV1(
			clavePrueba(t), huellaPrueba("a"), caso.estado, caso.fase,
		)
		if !errors.Is(err, ErrEstadoIncompatible) {
			t.Fatalf("estado/fase incompatibles aceptados: %q/%q: %v", caso.estado, caso.fase, err)
		}
	}
}

func TestIntencionCanonicaRestaurableYJSONEstrictoRecursivo(t *testing.T) {
	intencion := intencionPrueba(t)
	canonico, _ := intencion.RepresentacionCanonica()
	huella, _ := intencion.HuellaSHA256()
	restaurada, err := RestaurarIntencionResultadoV1ConHuellaSHA256(canonico, huella)
	if err != nil {
		t.Fatalf("restaurar intención: %v", err)
	}
	if huellaRestaurada, _ := restaurada.HuellaSHA256(); huellaRestaurada != huella {
		t.Fatal("restauración cambió intención")
	}
	duplicada := bytes.Replace(
		canonico,
		[]byte(`"referencia":"SujetoPseudonimo/ABC#1","version":1`),
		[]byte(`"referencia":"SujetoPseudonimo/ABC#1","version":1,"version":1`),
		1,
	)
	if bytes.Equal(duplicada, canonico) {
		t.Fatal("la prueba no insertó la clave duplicada")
	}
	if _, err := RestaurarIntencionResultadoV1(duplicada); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("clave anidada duplicada aceptada: %v", err)
	}
}

func TestReciboMinimoValidaIntencionYProducePredecesor(t *testing.T) {
	intencion := intencionPrueba(t)
	indice, _ := CalcularIndiceHMACSHA256(intencion.Clave(), []byte(strings.Repeat("k", 32)))
	recibo, err := NuevoReciboV1("ReciboCalculo/ABC#1", 7, indice, intencion)
	if err != nil {
		t.Fatal(err)
	}
	if err := recibo.ValidarPara(indice, intencion); err != nil {
		t.Fatalf("recibo no ligado a su intención: %v", err)
	}
	ajena := crearIntencion(
		t, intencion.Clave(), huellaPrueba("b"), ResultadoCompletado, FaseCompletado,
	)
	if err := recibo.ValidarPara(indice, ajena); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("recibo aceptó intención ajena: %v", err)
	}
	if err := recibo.ValidarPara(huellaPrueba("c"), intencion); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("recibo aceptó índice ajeno: %v", err)
	}
	vinculo, err := recibo.VinculoPredecesor()
	if err != nil || vinculo.ReferenciaRecibo != recibo.Referencia() {
		t.Fatalf("vínculo predecesor inválido: %+v / %v", vinculo, err)
	}
	huellaRecibo, _ := recibo.HuellaSHA256()
	if vinculo.HuellaReciboSHA256 != huellaRecibo {
		t.Fatal("el predecesor no fija el recibo exacto")
	}
}

func TestReciboCanonicoRestaurableYDetectaManipulacion(t *testing.T) {
	recibo := reciboPrueba(t)
	canonico, _ := recibo.RepresentacionCanonica()
	huella, _ := recibo.HuellaSHA256()
	restaurado, err := RestaurarReciboV1ConHuellaSHA256(canonico, huella)
	if err != nil {
		t.Fatalf("restaurar recibo: %v", err)
	}
	if err := restaurado.Validar(); err != nil {
		t.Fatalf("recibo restaurado inválido: %v", err)
	}
	manipulado := bytes.Replace(canonico, []byte(huellaPrueba("a")), []byte(huellaPrueba("f")), 1)
	if bytes.Equal(manipulado, canonico) {
		t.Fatal("la prueba no manipuló el recibo")
	}
	if _, err := RestaurarReciboV1ConHuellaSHA256(manipulado, huella); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("manipulación de recibo no detectada: %v", err)
	}
}

func crearIntencion(
	t *testing.T,
	clave ClaveEfectoV1,
	huella string,
	estado EstadoResultadoV1,
	fase FaseResultadoV1,
) IntencionResultadoV1 {
	t.Helper()
	intencion, err := NuevaIntencionResultadoV1(clave, huella, estado, fase)
	if err != nil {
		t.Fatalf("crear intención: %v", err)
	}
	return intencion
}
