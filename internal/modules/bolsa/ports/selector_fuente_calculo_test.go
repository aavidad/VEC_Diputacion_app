package ports

import (
	"bytes"
	"strings"
	"testing"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestSelectorFuenteExactaTieneVectorCanonicoV1Estable(t *testing.T) {
	selector := selectorFuenteCanonicoPrueba(t)
	primera, err := selector.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := selector.RepresentacionCanonicaV1()
	if err != nil || !bytes.Equal(primera, segunda) {
		t.Fatal("la representacion canonica no es determinista")
	}
	if !bytes.Contains(primera, []byte(`"esquema":"`+
		oficial.EsquemaSelectorFuenteExactaCalculoReglasBaremoV1+`"`)) {
		t.Fatal("la representacion no declara su esquema")
	}
	esperada := []byte(`{"esquema":"vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1",` +
		`"reglas":{"contenido":{"referencia":"reglas:contenido:canonico","version":3,` +
		`"huella_sha256":"` + strings.Repeat("a", 64) + `"},"revision":7,` +
		`"huella_estado_sha256":"` + strings.Repeat("d", 64) + `"},` +
		`"instantanea_entrada":{"referencia":"entrada:canonica","version":5,` +
		`"huella_sha256":"` + strings.Repeat("4", 64) + `"},` +
		`"sujeto_pseudonimo":{"referencia":"` + sujetoSelectorCanonicoPrueba("0") + `","version":11,` +
		`"huella_sha256":"` + strings.Repeat("6", 64) + `"},` +
		`"convocatoria":{"referencia":"convocatoria:canonica","version":13,` +
		`"huella_sha256":"` + strings.Repeat("8", 64) + `"}}`)
	if !bytes.Equal(primera, esperada) {
		t.Fatalf("material canonico inesperado:\n%s", primera)
	}
	huella, err := selector.HuellaSHA256V1()
	if err != nil {
		t.Fatal(err)
	}
	const vectorEsperado = "a3ae9f9109b38aaf736169c1609ecf21e8784bab15fc2aa02668b9e9875891b9"
	if huella != vectorEsperado {
		t.Fatalf("vector canonico modificado: %s", huella)
	}
	primera[0] ^= 0xff
	tercera, err := selector.RepresentacionCanonicaV1()
	if err != nil || !bytes.Equal(segunda, tercera) {
		t.Fatal("el contenido devuelto permite alterar llamadas posteriores")
	}
}

func TestSelectorFuenteExactaLigaCadaCampoSemantico(t *testing.T) {
	base := selectorFuenteCanonicoPrueba(t)
	huellaBase, err := base.HuellaSHA256V1()
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := []struct {
		nombre string
		mutar  func(SelectorFuenteExactaCalculoReglasBaremo) SelectorFuenteExactaCalculoReglasBaremo
	}{
		{"reglas.contenido.referencia", mutarEstadoSelector(t, "reglas:contenido:otro", 3, "a", 7, "d")},
		{"reglas.contenido.version", mutarEstadoSelector(t, "reglas:contenido:canonico", 4, "a", 7, "d")},
		{"reglas.contenido.huella", mutarEstadoSelector(t, "reglas:contenido:canonico", 3, "b", 7, "d")},
		{"reglas.revision", mutarEstadoSelector(t, "reglas:contenido:canonico", 3, "a", 8, "d")},
		{"reglas.huella_estado", mutarEstadoSelector(t, "reglas:contenido:canonico", 3, "a", 7, "e")},
		{"entrada.referencia", mutarReferenciaSelector(t, "entrada", "entrada:otra", 5, "4")},
		{"entrada.version", mutarReferenciaSelector(t, "entrada", "entrada:canonica", 6, "4")},
		{"entrada.huella", mutarReferenciaSelector(t, "entrada", "entrada:canonica", 5, "5")},
		{"sujeto.referencia", mutarReferenciaSelector(t, "sujeto", sujetoSelectorCanonicoPrueba("1"), 11, "6")},
		{"sujeto.version", mutarReferenciaSelector(t, "sujeto", sujetoSelectorCanonicoPrueba("0"), 12, "6")},
		{"sujeto.huella", mutarReferenciaSelector(t, "sujeto", sujetoSelectorCanonicoPrueba("0"), 11, "7")},
		{"convocatoria.referencia", mutarReferenciaSelector(t, "convocatoria", "convocatoria:otra", 13, "8")},
		{"convocatoria.version", mutarReferenciaSelector(t, "convocatoria", "convocatoria:canonica", 14, "8")},
		{"convocatoria.huella", mutarReferenciaSelector(t, "convocatoria", "convocatoria:canonica", 13, "9")},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := caso.mutar(base)
			huella, errMutada := mutado.HuellaSHA256V1()
			if errMutada != nil || huella == huellaBase {
				t.Fatalf("campo no ligado: huella=%q error=%v", huella, errMutada)
			}
		})
	}
}

func TestSelectorFuenteExactaRechazaValoresAusentesONoCanonicos(t *testing.T) {
	base := selectorFuenteCanonicoPrueba(t)
	casos := []SelectorFuenteExactaCalculoReglasBaremo{
		{},
		{EstadoReglas: base.EstadoReglas, InstantaneaEntrada: base.InstantaneaEntrada,
			SujetoPseudonimo: base.SujetoPseudonimo},
	}
	for indice, caso := range casos {
		if caso.Validar() == nil {
			t.Fatalf("caso %d acepto un valor cero", indice)
		}
		if _, err := caso.RepresentacionCanonicaV1(); err == nil {
			t.Fatalf("caso %d produjo material para un valor cero", indice)
		}
	}
	if _, err := reglas.NuevaReferenciaVersionada(
		"referencia:no:canonica", 1, strings.Repeat("A", 64),
	); err == nil {
		t.Fatal("el dominio permitio introducir una huella no canonica")
	}
	for _, referencia := range []string{"12345678Z", "persona@example.test", "../../etc/passwd"} {
		sujeto, err := reglas.NuevaReferenciaVersionada(referencia, 1, strings.Repeat("1", 64))
		if err == nil {
			candidato := base
			candidato.SujetoPseudonimo = sujeto
			if candidato.Validar() == nil {
				t.Fatalf("selector acepto sujeto directo %q", referencia)
			}
		}
	}
	confundido := base
	confundido.Convocatoria = base.SujetoPseudonimo
	if confundido.Validar() == nil {
		t.Fatal("selector acepto sujeto y convocatoria con la misma referencia")
	}
}

func selectorFuenteCanonicoPrueba(t *testing.T) SelectorFuenteExactaCalculoReglasBaremo {
	t.Helper()
	contenido := referenciaSelectorCanonicaPrueba(t, "reglas:contenido:canonico", 3, "a")
	estado, err := reglas.NuevoVinculoEstadoReglasBaremo(contenido, 7, strings.Repeat("d", 64))
	if err != nil {
		t.Fatal(err)
	}
	return SelectorFuenteExactaCalculoReglasBaremo{
		EstadoReglas:       estado,
		InstantaneaEntrada: referenciaSelectorCanonicaPrueba(t, "entrada:canonica", 5, "4"),
		SujetoPseudonimo: referenciaSelectorCanonicaPrueba(
			t, sujetoSelectorCanonicoPrueba("0"), 11, "6",
		),
		Convocatoria: referenciaSelectorCanonicaPrueba(t, "convocatoria:canonica", 13, "8"),
	}
}

func sujetoSelectorCanonicoPrueba(marca string) string {
	return "hmac-sha256:seudonimo_selector_v1:" + strings.Repeat(marca, 64)
}

func referenciaSelectorCanonicaPrueba(
	t *testing.T, referencia string, version uint64, digito string,
) reglas.ReferenciaVersionada {
	t.Helper()
	valor, err := reglas.NuevaReferenciaVersionada(referencia, version, strings.Repeat(digito, 64))
	if err != nil {
		t.Fatal(err)
	}
	return valor
}

func mutarEstadoSelector(
	t *testing.T, ref string, version uint64, huella string, revision uint64, huellaEstado string,
) func(SelectorFuenteExactaCalculoReglasBaremo) SelectorFuenteExactaCalculoReglasBaremo {
	t.Helper()
	contenido := referenciaSelectorCanonicaPrueba(t, ref, version, huella)
	estado, err := reglas.NuevoVinculoEstadoReglasBaremo(
		contenido, revision, strings.Repeat(huellaEstado, 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	return func(selector SelectorFuenteExactaCalculoReglasBaremo) SelectorFuenteExactaCalculoReglasBaremo {
		selector.EstadoReglas = estado
		return selector
	}
}

func mutarReferenciaSelector(
	t *testing.T, campo, ref string, version uint64, huella string,
) func(SelectorFuenteExactaCalculoReglasBaremo) SelectorFuenteExactaCalculoReglasBaremo {
	t.Helper()
	valor := referenciaSelectorCanonicaPrueba(t, ref, version, huella)
	return func(selector SelectorFuenteExactaCalculoReglasBaremo) SelectorFuenteExactaCalculoReglasBaremo {
		switch campo {
		case "entrada":
			selector.InstantaneaEntrada = valor
		case "sujeto":
			selector.SujetoPseudonimo = valor
		case "convocatoria":
			selector.Convocatoria = valor
		}
		return selector
	}
}
