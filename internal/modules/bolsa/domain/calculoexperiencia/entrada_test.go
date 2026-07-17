package calculoexperiencia

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPeriodoServicioConservaFinSinDecidirInclusividad(t *testing.T) {
	desde := fechaEntrada(t, 2026, 1, 1)
	mismoDia := fechaEntrada(t, 2026, 1, 1)
	cerrado, err := NuevoPeriodoServicioCerrado(desde, mismoDia)
	if err != nil || cerrado.Modo() != PeriodoServicioCerrado || cerrado.EnCurso() {
		t.Fatalf("cerrado = %#v, %v", cerrado, err)
	}
	fin, existe := cerrado.FinInformado()
	if !existe || fin != mismoDia || cerrado.Desde() != desde {
		t.Fatalf("fin = %v, %t", fin, existe)
	}

	abierto, err := NuevoPeriodoServicioEnCurso(desde)
	if err != nil || abierto.Modo() != PeriodoServicioEnCurso || !abierto.EnCurso() {
		t.Fatalf("en curso = %#v, %v", abierto, err)
	}
	if _, existe := abierto.FinInformado(); existe {
		t.Fatal("un periodo en curso expuso fin")
	}

	anterior := fechaEntrada(t, 2025, 12, 31)
	if _, err := NuevoPeriodoServicioCerrado(desde, anterior); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("periodo invertido: %v", err)
	}
	if _, err := NuevoPeriodoServicioCerrado(baremacion.FechaCivil{}, mismoDia); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("inicio invalido: %v", err)
	}
	if _, err := NuevoPeriodoServicioEnCurso(baremacion.FechaCivil{}); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("abierto invalido: %v", err)
	}
}

func TestAtributoCatalogadoEsCerradoYVersionado(t *testing.T) {
	catalogo := referenciaEntrada(t, "catalogo:empleador", 3, 'a')
	atributo, err := NuevoAtributoCatalogado("empleador", catalogo, "entidad_local")
	if err != nil || atributo.Clave() != "empleador" || atributo.Catalogo() != catalogo ||
		atributo.Valor() != "entidad_local" {
		t.Fatalf("atributo = %#v, %v", atributo, err)
	}

	casosInvalidos := []struct {
		clave string
		valor string
	}{
		{clave: "", valor: "entidad_local"},
		{clave: "Empleador", valor: "entidad_local"},
		{clave: "empleador libre", valor: "entidad_local"},
		{clave: "dni", valor: "entidad_local"},
		{clave: "dni_hash", valor: "entidad_local"},
		{clave: "persona_ref", valor: "entidad_local"},
		{clave: "nombre_completo", valor: "entidad_local"},
		{clave: "contacto.email", valor: "entidad_local"},
		{clave: "dato_salud", valor: "entidad_local"},
		{clave: "motivo-protegido", valor: "entidad_local"},
		{clave: "cuenta_iban", valor: "entidad_local"},
		{clave: "numero_nss", valor: "entidad_local"},
		{clave: "empleador", valor: "12345678x"},
		{clave: "empleador", valor: "x1234567a"},
		{clave: "empleador", valor: "es1234567890123456789012"},
		{clave: "empleador", valor: ""},
		{clave: "empleador", valor: "Entidad local"},
	}
	for _, caso := range casosInvalidos {
		if _, err := NuevoAtributoCatalogado(caso.clave, catalogo, caso.valor); !errors.Is(err, ErrValorNoCanonico) {
			t.Errorf("%q/%q: %v", caso.clave, caso.valor, err)
		}
	}
	if _, err := NuevoAtributoCatalogado("empleador", reglasbaremo.ReferenciaVersionada{}, "local"); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("catalogo invalido: %v", err)
	}
}

func TestComputoIntegroAtestadoEsExplicitoYNoContieneCausa(t *testing.T) {
	ausente := SinComputoIntegroAtestado()
	if ausente.EstaAtestado() {
		t.Fatal("ausente aparece atestado")
	}
	if _, existe := ausente.Referencia(); existe {
		t.Fatal("ausente expone referencia")
	}

	referencia := referenciaEntrada(t, "atestacion:computo:001", 1, 'b')
	atestado, err := NuevoComputoIntegroAtestado(referencia)
	if err != nil || !atestado.EstaAtestado() {
		t.Fatalf("atestacion = %#v, %v", atestado, err)
	}
	obtenida, existe := atestado.Referencia()
	if !existe || obtenida != referencia {
		t.Fatalf("referencia = %#v, %t", obtenida, existe)
	}
	if _, err := NuevoComputoIntegroAtestado(reglasbaremo.ReferenciaVersionada{}); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("atestacion invalida: %v", err)
	}
}

func TestTramoExperienciaOrdenaYCopiaAtributos(t *testing.T) {
	atributoB := atributoEntrada(t, "relacion", "laboral", "catalogo:relacion", 'c')
	atributoA := atributoEntrada(t, "empleador", "entidad_local", "catalogo:empleador", 'd')
	atributos := []AtributoCatalogado{atributoB, atributoA}
	tramo := tramoEntrada(t, "tramo:002", "servicio-estable-001", atributos, false)
	atributos[0] = AtributoCatalogado{}

	obtenidos := tramo.Atributos()
	if len(obtenidos) != 2 || obtenidos[0].Clave() != "empleador" || obtenidos[1].Clave() != "relacion" {
		t.Fatalf("atributos = %#v", obtenidos)
	}
	obtenidos[0] = AtributoCatalogado{}
	if tramo.Atributos()[0].Clave() != "empleador" {
		t.Fatal("el getter permitio mutar el tramo")
	}
	if tramo.ServicioRef() != servicioReferenciaPrueba("servicio-estable-001") || !tramo.Jornada().EsValida() {
		t.Fatalf("tramo inesperado: %#v", tramo)
	}
}

func TestTramoExperienciaRechazaDuplicadosYLimites(t *testing.T) {
	catalogoUno := referenciaEntrada(t, "catalogo:uno", 1, '1')
	catalogoDos := referenciaEntrada(t, "catalogo:dos", 1, '2')
	a, _ := NuevoAtributoCatalogado("eje_a", catalogoUno, "uno")
	bMismaClave, _ := NuevoAtributoCatalogado("eje_a", catalogoDos, "dos")
	bMismoCatalogo, _ := NuevoAtributoCatalogado("eje_b", catalogoUno, "dos")
	if _, err := construirTramoEntrada(t, "tramo:duplicado-clave", "servicio-1", []AtributoCatalogado{a, bMismaClave}); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("clave duplicada: %v", err)
	}
	if _, err := construirTramoEntrada(t, "tramo:catalogo-compartido", "servicio-2", []AtributoCatalogado{a, bMismoCatalogo}); err != nil {
		t.Fatalf("dos ejes pueden compartir catalogo: %v", err)
	}

	demasiados := make([]AtributoCatalogado, maximoAtributosPorTramo+1)
	for indice := range demasiados {
		catalogo := referenciaEntrada(t, fmt.Sprintf("catalogo:%03d", indice), 1, byte('a'+indice%6))
		demasiados[indice], _ = NuevoAtributoCatalogado(
			fmt.Sprintf("eje_%03d", indice), catalogo, fmt.Sprintf("valor_%03d", indice),
		)
	}
	if _, err := construirTramoEntrada(t, "tramo:demasiados", "servicio-3", demasiados); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("demasiados atributos: %v", err)
	}

	periodo, _ := NuevoPeriodoServicioEnCurso(fechaEntrada(t, 2026, 1, 1))
	jornada, _ := baremacion.NuevaFraccionJornada(1, 1)
	if _, err := NuevoTramoExperiencia(
		referenciaEntrada(t, "tramo:servicio-invalido", 1, 'e'), "servicio con espacios",
		periodo, jornada, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("servicio libre: %v", err)
	}
	if _, err := NuevoTramoExperiencia(
		referenciaEntrada(t, "tramo:servicio-dni", 1, 'e'), "12345678x",
		periodo, jornada, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("dni no opaco: %v", err)
	}
	if _, err := NuevoTramoExperiencia(
		referenciaEntrada(t, "tramo:servicio-corto", 1, 'e'), "srv_12345678x",
		periodo, jornada, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("referencia sin entropia suficiente: %v", err)
	}
	if _, err := NuevoTramoExperiencia(
		referenciaEntrada(t, "tramo:servicio-no-hex", 1, 'e'),
		"srv_12345678x_______________________1234567890123456789012345678",
		periodo, jornada, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("referencia seudonima no canonica: %v", err)
	}
	referenciaDirecta := referenciaEntrada(t, "catalogo:tramo_directo", 1, 'e')
	if _, err := NuevoTramoExperiencia(
		referenciaDirecta, servicioReferenciaPrueba("servicio-ref-directa"),
		periodo, jornada, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("referencia de tramo no opaca: %v", err)
	}
	if _, err := NuevoTramoExperiencia(
		referenciaEntrada(t, "tramo:jornada-invalida", 1, 'f'), servicioReferenciaPrueba("servicio-4"),
		periodo, baremacion.FraccionJornada{}, SinComputoIntegroAtestado(), nil,
	); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("jornada invalida: %v", err)
	}
}

func TestEntradaExperienciaOrdenaCopiaYAdmiteCeroTramos(t *testing.T) {
	instantanea := referenciaEntrada(t, "instantanea:servicios:001", 4, '9')
	vacia, err := NuevaEntradaExperiencia(instantanea, nil)
	if err != nil || len(vacia.Tramos()) != 0 {
		t.Fatalf("entrada vacia = %#v, %v", vacia, err)
	}

	tramoB := tramoEntrada(t, "tramo:002", "servicio-compartido", nil, false)
	tramoA := tramoEntrada(t, "tramo:001", "servicio-compartido", nil, true)
	origen := []TramoExperiencia{tramoB, tramoA}
	entrada, err := NuevaEntradaExperiencia(instantanea, origen)
	if err != nil {
		t.Fatal(err)
	}
	origen[0] = TramoExperiencia{}
	obtenidos := entrada.Tramos()
	primeraEsperada := tokenReferenciaPrueba(prefijoTramoEntrada, "tramo:001")
	segundaEsperada := tokenReferenciaPrueba(prefijoTramoEntrada, "tramo:002")
	if primeraEsperada > segundaEsperada {
		primeraEsperada, segundaEsperada = segundaEsperada, primeraEsperada
	}
	if obtenidos[0].Referencia().Referencia() != primeraEsperada ||
		obtenidos[1].Referencia().Referencia() != segundaEsperada {
		t.Fatalf("orden = %#v", obtenidos)
	}
	obtenidos[0] = TramoExperiencia{}
	if entrada.Tramos()[0].Referencia().Referencia() != primeraEsperada {
		t.Fatal("el getter permitio mutar la entrada")
	}
	if entrada.Instantanea() != instantanea || entrada.Validar() != nil {
		t.Fatal("instantanea o validacion inesperada")
	}
}

func TestEntradaExperienciaRechazaReferenciaRepetidaAunqueCambieVersion(t *testing.T) {
	instantanea := referenciaEntrada(t, "instantanea:duplicados", 1, '8')
	primero := tramoEntrada(t, "tramo:repetido", "servicio-1", nil, false)
	segundo := primero.clonar()
	segundo.referencia = referenciaEntrada(t, "tramo:repetido", 2, '7')
	if _, err := NuevaEntradaExperiencia(instantanea, []TramoExperiencia{primero, segundo}); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("referencia repetida: %v", err)
	}

	demasiados := make([]TramoExperiencia, maximoTramosEntrada+1)
	if _, err := NuevaEntradaExperiencia(instantanea, demasiados); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("demasiados tramos: %v", err)
	}
	if _, err := NuevaEntradaExperiencia(reglasbaremo.ReferenciaVersionada{}, nil); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("instantanea invalida: %v", err)
	}
}

func TestEntradaExperienciaNoModelaIdentidadNiCausa(t *testing.T) {
	entrada := entradaCanonicaPrueba(t)
	contenido, err := entrada.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	texto := strings.ToLower(string(contenido))
	for _, prohibido := range []string{
		`"dni"`, `"nif"`, `"nie"`, `"nombre"`, `"apellidos"`,
		`"persona"`, `"diagnostico"`, `"causa"`, `"direccion"`,
	} {
		if strings.Contains(texto, prohibido) {
			t.Errorf("la entrada contiene el campo prohibido %s", prohibido)
		}
	}
}

func entradaCanonicaPrueba(t *testing.T) EntradaExperiencia {
	t.Helper()
	atributoRelacion := atributoEntrada(t, "relacion", "funcionario_interino", "catalogo:relacion", '3')
	atributoEmpleador := atributoEntrada(t, "empleador", "entidad_local", "catalogo:empleador", '4')
	tramoDos := tramoEntrada(t, "tramo:002", "servicio-002", []AtributoCatalogado{atributoRelacion}, true)
	tramoUno := tramoEntrada(t, "tramo:001", "servicio-001", []AtributoCatalogado{atributoRelacion, atributoEmpleador}, false)
	entrada, err := NuevaEntradaExperiencia(
		referenciaEntrada(t, "instantanea:servicios:2026", 7, '0'),
		[]TramoExperiencia{tramoDos, tramoUno},
	)
	if err != nil {
		t.Fatalf("entrada canonica: %v", err)
	}
	return entrada
}

func tramoEntrada(
	t *testing.T,
	referencia string,
	servicio string,
	atributos []AtributoCatalogado,
	enCurso bool,
) TramoExperiencia {
	t.Helper()
	tramo, err := construirTramoEntradaConModo(t, referencia, servicio, atributos, enCurso)
	if err != nil {
		t.Fatalf("tramo %s: %v", referencia, err)
	}
	return tramo
}

func construirTramoEntrada(
	t *testing.T,
	referencia string,
	servicio string,
	atributos []AtributoCatalogado,
) (TramoExperiencia, error) {
	t.Helper()
	return construirTramoEntradaConModo(t, referencia, servicio, atributos, false)
}

func construirTramoEntradaConModo(
	t *testing.T,
	referencia string,
	servicio string,
	atributos []AtributoCatalogado,
	enCurso bool,
) (TramoExperiencia, error) {
	t.Helper()
	desde := fechaEntrada(t, 2025, 1, 1)
	var periodo PeriodoServicio
	var err error
	if enCurso {
		periodo, err = NuevoPeriodoServicioEnCurso(desde)
	} else {
		periodo, err = NuevoPeriodoServicioCerrado(desde, fechaEntrada(t, 2025, 12, 31))
	}
	if err != nil {
		t.Fatal(err)
	}
	jornada, _ := baremacion.NuevaFraccionJornada(1, 2)
	atestacion := SinComputoIntegroAtestado()
	if enCurso {
		atestacion, err = NuevoComputoIntegroAtestado(
			referenciaEntrada(t, "atestacion:"+strings.ReplaceAll(referencia, ":", "-"), 1, '6'),
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	return NuevoTramoExperiencia(
		referenciaEntrada(t, referencia, 1, '5'),
		servicioReferenciaPrueba(servicio),
		periodo,
		jornada,
		atestacion,
		atributos,
	)
}

func servicioReferenciaPrueba(etiqueta string) string {
	return tokenReferenciaPrueba(prefijoServicioEntrada, etiqueta)
}

func tokenReferenciaPrueba(prefijo string, etiqueta string) string {
	huella := sha256.Sum256([]byte(etiqueta))
	return prefijo + hex.EncodeToString(huella[:])
}

func atributoEntrada(
	t *testing.T,
	clave string,
	valor string,
	catalogo string,
	huella byte,
) AtributoCatalogado {
	t.Helper()
	atributo, err := NuevoAtributoCatalogado(
		clave,
		referenciaEntrada(t, catalogo, 1, huella),
		valor,
	)
	if err != nil {
		t.Fatalf("atributo %s: %v", clave, err)
	}
	return atributo
}

func referenciaEntrada(
	t *testing.T,
	referencia string,
	version uint64,
	huella byte,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	switch {
	case strings.HasPrefix(referencia, "instantanea:"):
		referencia = tokenReferenciaPrueba(prefijoInstantaneaEntrada, referencia)
	case strings.HasPrefix(referencia, "tramo:"):
		referencia = tokenReferenciaPrueba(prefijoTramoEntrada, referencia)
	case strings.HasPrefix(referencia, "atestacion:"):
		referencia = tokenReferenciaPrueba(prefijoAtestacionEntrada, referencia)
	}
	resultado, err := reglasbaremo.NuevaReferenciaVersionada(
		referencia,
		version,
		strings.Repeat(string(huella), 64),
	)
	if err != nil {
		t.Fatalf("referencia %s: %v", referencia, err)
	}
	return resultado
}

func fechaEntrada(t *testing.T, anio, mes, dia int) baremacion.FechaCivil {
	t.Helper()
	fecha, err := baremacion.NuevaFechaCivil(anio, mes, dia)
	if err != nil {
		t.Fatalf("fecha %04d-%02d-%02d: %v", anio, mes, dia, err)
	}
	return fecha
}
