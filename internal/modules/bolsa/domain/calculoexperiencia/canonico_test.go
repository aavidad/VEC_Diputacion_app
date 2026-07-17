package calculoexperiencia

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEntradaExperienciaVectorCanonicoGolden(t *testing.T) {
	entrada, err := NuevaEntradaExperiencia(
		referenciaEntrada(t, "instantanea:vacia", 1, 'a'),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.bolsa.entrada_experiencia.v1","instantanea":{"referencia":"` +
		entrada.Instantanea().Referencia() + `","version":1,"huella_sha256":"` +
		strings.Repeat("a", 64) + `"},"tramos":[]}`
	contenido, err := entrada.RepresentacionCanonica()
	if err != nil || string(contenido) != esperado {
		t.Fatalf("canonico = %s, %v; quiere %s", contenido, err, esperado)
	}
	if entrada.Validar() != nil {
		t.Fatal("la entrada golden no valida")
	}
}

func TestEntradaExperienciaCanonicaEsIndependienteDelOrdenYLasMutaciones(t *testing.T) {
	entrada := entradaCanonicaPrueba(t)
	canonico, err := entrada.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := entrada.HuellaSHA256()
	if err != nil || !huellaSHA256Canonica(huella) {
		t.Fatalf("huella = %q, %v", huella, err)
	}
	serializado, err := json.Marshal(entrada)
	if err != nil || !bytes.Equal(serializado, canonico) {
		t.Fatalf("Marshal = %s, %v", serializado, err)
	}

	tramos := entrada.Tramos()
	for izquierda, derecha := 0, len(tramos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		tramos[izquierda], tramos[derecha] = tramos[derecha], tramos[izquierda]
	}
	for indice := range tramos {
		atributos := tramos[indice].Atributos()
		for izquierda, derecha := 0, len(atributos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
			atributos[izquierda], atributos[derecha] = atributos[derecha], atributos[izquierda]
		}
		reconstruido, err := NuevoTramoExperiencia(
			tramos[indice].Referencia(),
			tramos[indice].ServicioRef(),
			tramos[indice].Periodo(),
			tramos[indice].Jornada(),
			tramos[indice].Atestacion(),
			atributos,
		)
		if err != nil {
			t.Fatal(err)
		}
		tramos[indice] = reconstruido
	}
	otra, err := NuevaEntradaExperiencia(entrada.Instantanea(), tramos)
	if err != nil {
		t.Fatal(err)
	}
	otroCanonico, _ := otra.RepresentacionCanonica()
	otraHuella, _ := otra.HuellaSHA256()
	if !bytes.Equal(canonico, otroCanonico) || huella != otraHuella {
		t.Fatalf("representaciones distintas:\n%s\n%s", canonico, otroCanonico)
	}
}

func TestRestaurarEntradaExperienciaRoundtripYHuella(t *testing.T) {
	original := entradaCanonicaPrueba(t)
	contenido, _ := original.RepresentacionCanonica()
	huella, _ := original.HuellaSHA256()

	restaurada, err := RestaurarEntradaExperiencia(contenido)
	if err != nil {
		t.Fatal(err)
	}
	reconstruido, _ := restaurada.RepresentacionCanonica()
	if !bytes.Equal(reconstruido, contenido) {
		t.Fatalf("roundtrip distinto:\n%s\n%s", contenido, reconstruido)
	}
	conHuella, err := RestaurarEntradaExperienciaConHuellaSHA256(contenido, huella)
	if err != nil || conHuella.Validar() != nil {
		t.Fatalf("restaurar con huella: %v", err)
	}
	huellaMala := strings.Repeat("f", 64)
	if huellaMala == huella {
		huellaMala = strings.Repeat("e", 64)
	}
	if _, err := RestaurarEntradaExperienciaConHuellaSHA256(contenido, huellaMala); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("huella distinta: %v", err)
	}
	if _, err := RestaurarEntradaExperienciaConHuellaSHA256(contenido, strings.ToUpper(huella)); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("huella no canonica: %v", err)
	}
}

func TestRestaurarEntradaExperienciaRechazaJSONHostilONoCanonico(t *testing.T) {
	entrada := entradaCanonicaPrueba(t)
	canonico, _ := entrada.RepresentacionCanonica()
	texto := string(canonico)

	material := materialEntradaExperiencia{
		Esquema:     esquemaEntradaExperiencia,
		Instantanea: materializarReferencia(entrada.instantanea),
		Tramos:      make([]materialTramo, len(entrada.tramos)),
	}
	for indice, tramo := range entrada.tramos {
		material.Tramos[indice] = materializarTramo(tramo)
	}
	materialInvertido := material
	materialInvertido.Tramos = append([]materialTramo(nil), material.Tramos...)
	for izquierda, derecha := 0, len(materialInvertido.Tramos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		materialInvertido.Tramos[izquierda], materialInvertido.Tramos[derecha] =
			materialInvertido.Tramos[derecha], materialInvertido.Tramos[izquierda]
	}
	tramosInvertidos, _ := json.Marshal(materialInvertido)

	atributosInvertidos := material
	atributosInvertidos.Tramos = append([]materialTramo(nil), material.Tramos...)
	indiceConVariosAtributos := indiceTramoMaterial(material, func(tramo materialTramo) bool {
		return len(tramo.Atributos) > 1
	})
	atributosInvertidos.Tramos[indiceConVariosAtributos].Atributos = append(
		[]materialAtributo(nil),
		atributosInvertidos.Tramos[indiceConVariosAtributos].Atributos...,
	)
	atributos := atributosInvertidos.Tramos[indiceConVariosAtributos].Atributos
	for izquierda, derecha := 0, len(atributos)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		atributos[izquierda], atributos[derecha] = atributos[derecha], atributos[izquierda]
	}
	atributosNoCanonicos, _ := json.Marshal(atributosInvertidos)

	casos := []struct {
		nombre    string
		contenido []byte
		error     error
	}{
		{nombre: "espacio", contenido: append([]byte(" "), canonico...), error: ErrValorNoCanonico},
		{nombre: "sobrante", contenido: append(append([]byte(nil), canonico...), []byte(`{}`)...), error: ErrValorNoCanonico},
		{
			nombre:    "campo_desconocido",
			contenido: []byte(strings.Replace(texto, `{"esquema":`, `{"desconocido":1,"esquema":`, 1)),
			error:     ErrValorNoCanonico,
		},
		{
			nombre: "clave_duplicada",
			contenido: []byte(strings.Replace(
				texto,
				`{"esquema":`,
				`{"esquema":"vec.bolsa.entrada_experiencia.v1","esquema":`,
				1,
			)),
			error: ErrValorNoCanonico,
		},
		{
			nombre:    "esquema_futuro",
			contenido: []byte(strings.Replace(texto, esquemaEntradaExperiencia, "vec.bolsa.entrada_experiencia.v2", 1)),
			error:     ErrEsquemaIncompatible,
		},
		{
			nombre:    "jornada_no_reducida",
			contenido: []byte(strings.Replace(texto, `"jornada":"1/2"`, `"jornada":"2/4"`, 1)),
			error:     ErrValorNoCanonico,
		},
		{nombre: "tramos_desordenados", contenido: tramosInvertidos, error: ErrValorNoCanonico},
		{nombre: "atributos_desordenados", contenido: atributosNoCanonicos, error: ErrValorNoCanonico},
		{nombre: "utf8_invalido", contenido: []byte{0xff, 0xfe}, error: ErrValorNoCanonico},
		{nombre: "vacio", contenido: nil, error: ErrFueraDeLimites},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := RestaurarEntradaExperiencia(caso.contenido); !errors.Is(err, caso.error) {
				t.Fatalf("error = %v; quiere %v", err, caso.error)
			}
		})
	}
}

func TestPrevueloJSONRechazaDuplicadosAnidadosYProfundidad(t *testing.T) {
	if err := comprobarJSONEntradaSinClavesDuplicadas(
		[]byte(`{"tramo":{"atributos":[],"atributos":[]}}`),
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("duplicado anidado: %v", err)
	}
	profundo := strings.Repeat("[", maximaProfundidadJSONEntrada+2) + "0" +
		strings.Repeat("]", maximaProfundidadJSONEntrada+2)
	if err := comprobarJSONEntradaSinClavesDuplicadas(
		[]byte(profundo),
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("profundidad: %v", err)
	}
	entrada := entradaCanonicaPrueba(t)
	canonico, _ := entrada.RepresentacionCanonica()
	if err := comprobarJSONEntradaSinClavesDuplicadas(canonico); err != nil {
		t.Fatalf("canonico valido rechazado: %v", err)
	}
}

func TestRestaurarEntradaRechazaUnionesDiscriminatoriasInvalidas(t *testing.T) {
	entrada := entradaCanonicaPrueba(t)
	material := materialEntradaExperiencia{
		Esquema:     esquemaEntradaExperiencia,
		Instantanea: materializarReferencia(entrada.instantanea),
		Tramos:      make([]materialTramo, len(entrada.tramos)),
	}
	for indice, tramo := range entrada.tramos {
		material.Tramos[indice] = materializarTramo(tramo)
	}
	indiceCerrado := indiceTramoMaterial(material, func(tramo materialTramo) bool {
		return tramo.Periodo.Modo == PeriodoServicioCerrado
	})
	indiceEnCurso := indiceTramoMaterial(material, func(tramo materialTramo) bool {
		return tramo.Periodo.Modo == PeriodoServicioEnCurso
	})
	indiceSinAtestacion := indiceTramoMaterial(material, func(tramo materialTramo) bool {
		return tramo.Atestacion.Modo == computoIntegroAusente
	})
	indiceConAtestacion := indiceTramoMaterial(material, func(tramo materialTramo) bool {
		return tramo.Atestacion.Modo == computoIntegroAtestado
	})

	casos := []struct {
		nombre    string
		modificar func(*materialEntradaExperiencia)
	}{
		{
			nombre: "cerrado_sin_fin",
			modificar: func(m *materialEntradaExperiencia) {
				m.Tramos[indiceCerrado].Periodo.FinInformado = nil
			},
		},
		{
			nombre: "en_curso_con_fin",
			modificar: func(m *materialEntradaExperiencia) {
				m.Tramos[indiceEnCurso].Periodo.FinInformado = &m.Tramos[indiceCerrado].Periodo.Desde
			},
		},
		{
			nombre: "atestacion_ausente_con_ref",
			modificar: func(m *materialEntradaExperiencia) {
				referencia := materializarReferencia(entrada.instantanea)
				m.Tramos[indiceSinAtestacion].Atestacion.Referencia = &referencia
			},
		},
		{
			nombre: "atestacion_sin_ref",
			modificar: func(m *materialEntradaExperiencia) {
				m.Tramos[indiceConAtestacion].Atestacion.Referencia = nil
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := material
			alterado.Tramos = append([]materialTramo(nil), material.Tramos...)
			caso.modificar(&alterado)
			contenido, _ := json.Marshal(alterado)
			if _, err := RestaurarEntradaExperiencia(contenido); err == nil {
				t.Fatal("se acepto una union discriminada incoherente")
			}
		})
	}
}

func indiceTramoMaterial(
	material materialEntradaExperiencia,
	cumple func(materialTramo) bool,
) int {
	for indice, tramo := range material.Tramos {
		if cumple(tramo) {
			return indice
		}
	}
	panic("el vector de prueba no contiene el tramo esperado")
}

func TestEntradaCompruebaLimitesAntesDeCopiarOReconstruir(t *testing.T) {
	instantanea := referenciaEntrada(t, "instantanea:limites", 1, 'd')
	atributos := make([]AtributoCatalogado, maximoAtributosPorTramo)
	catalogoCompartido := referenciaEntrada(t, "catalogo:compartido", 1, 'c')
	for indice := range atributos {
		atributos[indice], _ = NuevoAtributoCatalogado(
			"eje_"+strings.Repeat("x", 100)+string(rune('a'+indice%26))+string(rune('0'+indice%10)),
			catalogoCompartido,
			"valor_"+strings.Repeat("y", 100)+string(rune('a'+indice%26))+string(rune('0'+indice%10)),
		)
	}
	base := tramoEntrada(t, "tramo:presupuesto", "servicio-presupuesto", atributos, false)
	tramos := make([]TramoExperiencia, maximoTramosEntrada)
	for indice := range tramos {
		tramos[indice] = base
	}
	if _, err := NuevaEntradaExperiencia(instantanea, tramos); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("presupuesto previo: %v", err)
	}

	demasiadosTramos := materialEntradaExperiencia{
		Esquema:     esquemaEntradaExperiencia,
		Instantanea: materializarReferencia(instantanea),
		Tramos:      make([]materialTramo, maximoTramosEntrada+1),
	}
	contenido, _ := json.Marshal(demasiadosTramos)
	if _, err := RestaurarEntradaExperiencia(contenido); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("limite de tramos al restaurar: %v", err)
	}

	demasiadosAtributos := materialEntradaExperiencia{
		Esquema:     esquemaEntradaExperiencia,
		Instantanea: materializarReferencia(instantanea),
		Tramos: []materialTramo{{
			Atributos: make([]materialAtributo, maximoAtributosPorTramo+1),
		}},
	}
	contenido, _ = json.Marshal(demasiadosAtributos)
	if _, err := RestaurarEntradaExperiencia(contenido); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("limite de atributos al restaurar: %v", err)
	}

	excesivo := bytes.Repeat([]byte{'x'}, maximoBytesRepresentacionEntrada+1)
	if _, err := RestaurarEntradaExperiencia(excesivo); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("limite de bytes: %v", err)
	}
}
