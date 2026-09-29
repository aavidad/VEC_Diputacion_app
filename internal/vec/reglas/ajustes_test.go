package reglas

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ajustesMemoria guarda versiones de ajustes y devuelve la vigente en el
// instante pedido, como hará el almacén de PostgreSQL.
type ajustesMemoria struct {
	versiones []VersionAjustes
	err       error
	pedidos   []time.Time
}

func (a *ajustesMemoria) AjustesVigentesEn(_ context.Context, id string, instante time.Time) (VersionAjustes, bool, error) {
	a.pedidos = append(a.pedidos, instante)
	if a.err != nil {
		return VersionAjustes{}, false, a.err
	}
	var elegida VersionAjustes
	encontrada := false
	for _, v := range a.versiones {
		if v.CatalogoID == id && !v.VigenteDesde.After(instante) && (!encontrada || v.Version > elegida.Version) {
			elegida, encontrada = v, true
		}
	}
	return elegida, encontrada, nil
}

func versionAjustes(t *testing.T, version int, desde time.Time, ajustes map[string]map[string]string) VersionAjustes {
	t.Helper()
	huella, err := HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	return VersionAjustes{
		CatalogoID: CatalogoAjustesDe(CatalogoContratacionTemporal), Version: version, HuellaSHA256: huella,
		VigenteDesde: desde, Ajustes: ajustes,
	}
}

func resolutorCTConAjustes(t *testing.T, ajustes ConsultaAjustes, calculadora CalculadoraPlazos) *Resolutor {
	t.Helper()
	base := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, calculadora)
	cfg := base.cfg
	cfg.Ajustes = ajustes
	resolutor, err := NuevoResolutor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestReglaBaseDeclaraLoQueAdmiteAjustar(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	regla, err := resolutor.Regla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	e := regla.Edicion
	if e == nil || !slices.Equal(e.Campos, []string{CampoCantidad, CampoCantidadUrgente, CampoUnidad}) ||
		!slices.Equal(e.OpcionesUnidad, []Unidad{UnidadDiasHabiles, UnidadDiasNaturales}) ||
		e.CantidadMinima != 1 || e.CantidadMaxima != 60 || e.Admite(CampoComputo) || regla.Ajuste != nil {
		t.Fatalf("edición inesperada: %+v", e)
	}
	if jornada, err := resolutor.Regla(t.Context(), CTJornadaCompleta); err != nil || jornada.Edicion != nil {
		t.Fatalf("una regla sin «editable» no es ajustable: %+v %v", jornada.Edicion, err)
	}
}

func TestAjusteSeAplicaSobreLaReglaBaseYCitaSuVersion(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	almacen := &ajustesMemoria{versiones: []VersionAjustes{versionAjustes(t, 1, desde, map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7", CampoCantidadUrgente: "3"},
	})}}
	resolutor := resolutorCTConAjustes(t, almacen, nil)
	regla, err := resolutor.Regla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	if regla.Cantidad != 7 || regla.Atributos[AtributoCantidadUrgente] != "3" || regla.Origen != OrigenEjemplo ||
		regla.EsEjemplo() || !regla.PaqueteEjemplo || regla.Duda == "" || regla.Edicion == nil {
		t.Fatalf("regla ajustada inesperada: %+v", regla)
	}
	a := regla.Ajuste
	if a == nil || a.Version != 1 || !a.VigenteDesde.Equal(desde) ||
		a.BaseReferencia.CatalogoID != CatalogoContratacionTemporal || a.Campos[CampoCantidad] != "7" {
		t.Fatalf("ajuste inesperado: %+v", a)
	}
	if regla.Referencia != "vec.contratacion_temporal.reglas.ajustes:1:c03.plazo_fiscalizacion" ||
		regla.HuellaCatalogo != regla.ReferenciaEntrada.CatalogoHuellaSHA256 ||
		regla.HuellaCatalogo == a.BaseReferencia.CatalogoHuellaSHA256 || regla.HuellaCatalogo == a.HuellaAjustes {
		t.Fatalf("referencia de la regla ajustada: %s %s", regla.Referencia, regla.HuellaCatalogo)
	}
	// Las demás reglas siguen citando la base.
	if otra, err := resolutor.Regla(t.Context(), CTPlazoSubsanacion); err != nil || otra.Ajuste != nil ||
		otra.Referencia != "vec.contratacion_temporal.reglas:1:c04.plazo_subsanacion" || otra.Origen != OrigenEjemplo {
		t.Fatalf("regla sin ajuste: %+v %v", otra, err)
	}
}

// Un plazo se calcula con el ajuste vigente cuando empezó a correr: un cambio
// posterior no mueve los plazos que ya corren.
func TestPlazoEnCursoConservaElValorConQueEmpezo(t *testing.T) {
	cambio := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	almacen := &ajustesMemoria{versiones: []VersionAjustes{versionAjustes(t, 1, cambio, map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})}}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)

	antes := cambio.Add(-time.Hour)
	regla, _, err := resolutor.Vencimiento(t.Context(), CTPlazoFiscalizacion, antes, "")
	if err != nil || calculadora.recibida.Cantidad != 10 || regla.Ajuste != nil {
		t.Fatalf("empezó antes del cambio: cantidad %d, %v", calculadora.recibida.Cantidad, err)
	}
	despues := cambio.Add(time.Hour)
	regla, _, err = resolutor.Vencimiento(t.Context(), CTPlazoFiscalizacion, despues, "")
	if err != nil || calculadora.recibida.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != 1 {
		t.Fatalf("empezó después del cambio: cantidad %d, %v", calculadora.recibida.Cantidad, err)
	}
	if got := almacen.pedidos[len(almacen.pedidos)-1]; !got.Equal(despues) {
		t.Fatalf("los ajustes se piden en el inicio del plazo, no ahora: %v", got)
	}
	// Urgente: sin ajuste de la cantidad urgente rige la de la base.
	if _, _, err := resolutor.VencimientoUrgente(t.Context(), CTPlazoFiscalizacion, despues, ""); err != nil || calculadora.recibida.Cantidad != 5 {
		t.Fatalf("urgente tras ajustar solo la ordinaria: %d, %v", calculadora.recibida.Cantidad, err)
	}
	enCurso, err := resolutor.ReglasEn(t.Context(), antes)
	if err != nil || len(enCurso) == 0 {
		t.Fatal(err)
	}
	for _, r := range enCurso {
		if r.Ajuste != nil {
			t.Fatalf("ReglasEn antes del cambio aplicó un ajuste: %s", r.Clave)
		}
	}
	if _, err := resolutor.ReglasEn(t.Context(), time.Time{}); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("instante vacío: %v", err)
	}
}

// Un ajuste que no encaja con su regla deja fuera de uso esa regla, no el
// catálogo: las demás siguen respondiendo y nunca se vuelve al valor base.
func TestAjusteFueraDeLoQueAdmiteLaReglaDejaSoloEsaReglaFueraDeUso(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	casos := map[string]map[string]map[string]string{
		"campo no editable":        {CTPlazoFiscalizacion: {CampoComputo: "civil"}},
		"cantidad sobre el máximo": {CTPlazoFiscalizacion: {CampoCantidad: "61"}},
		"cantidad no canónica":     {CTPlazoFiscalizacion: {CampoCantidad: "07"}},
		"unidad fuera de opciones": {CTPlazoFiscalizacion: {CampoUnidad: "meses"}},
		"urgente mayor que plazo":  {CTPlazoFiscalizacion: {CampoCantidad: "4"}},
		"regla no ajustable":       {CTJornadaCompleta: {CampoCantidad: "2200"}},
	}
	for nombre, ajustes := range casos {
		t.Run(nombre, func(t *testing.T) {
			var clave string
			for c := range ajustes {
				clave = c
			}
			v := versionAjustes(t, 1, desde, ajustes)
			calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
			resolutor := resolutorCTConAjustes(t, &ajustesMemoria{versiones: []VersionAjustes{v}}, calculadora)
			todas, err := resolutor.Reglas(t.Context())
			if err != nil {
				t.Fatalf("el catálogo entero cayó: %v", err)
			}
			for _, r := range todas {
				if r.AjusteNoAplicable != (r.Clave == clave) || r.Ajuste != nil {
					t.Fatalf("marca de ajuste no aplicable en %s: %+v", r.Clave, r)
				}
			}
			if _, err := resolutor.Regla(t.Context(), clave); !errors.Is(err, ErrAjusteInvalido) {
				t.Fatalf("la regla con ajuste roto se usó: %v", err)
			}
			if _, _, err := resolutor.Vencimiento(t.Context(), clave, desde.Add(time.Hour), ""); err == nil {
				t.Fatal("se calculó un plazo con un ajuste roto")
			}
			if _, err := resolutor.Regla(t.Context(), CTPlazoSubsanacion); err != nil {
				t.Fatalf("otra regla dejó de responder: %v", err)
			}
		})
	}
	// Un ajuste de una regla que ya no existe en la base no gobierna nada.
	huerfano := versionAjustes(t, 1, desde, map[string]map[string]string{"c99.retirada": {CampoCantidad: "3"}})
	if _, err := resolutorCTConAjustes(t, &ajustesMemoria{versiones: []VersionAjustes{huerfano}}, nil).Reglas(t.Context()); err != nil {
		t.Fatalf("ajuste huérfano: %v", err)
	}
}

func TestCantidadUrgenteNoEditableLimitaElAjusteOrdinario(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	atributos := map[string]string{
		"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles",
		"cantidad": "10", AtributoCantidadUrgente: "5", "inicio": "contacto_efectivo", "computo": "administrativo",
		AtributoEditable: CampoCantidad, AtributoCantidadMinima: "1", AtributoCantidadMaxima: "60",
	}
	ajustes := map[string]map[string]string{BolsaPlazoRespuesta: {CampoCantidad: "4"}}
	huella, err := HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	version := VersionAjustes{CatalogoID: CatalogoAjustesDe(CatalogoBolsa), Version: 1,
		HuellaSHA256: huella, VigenteDesde: desde, Ajustes: ajustes}
	resolutor, err := NuevoResolutor(Configuracion{
		Consulta:   consultaMemoria{[]domain.CatalogoConfigurable{catalogoMemoria(t, 1, desde, atributos)}},
		CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa, Reloj: diaPresentacion,
		Ajustes: consultaAjustesFija{version},
	})
	if err != nil {
		t.Fatal(err)
	}
	reglas, err := resolutor.Reglas(t.Context())
	if err != nil || len(reglas) != 1 || !reglas[0].AjusteNoAplicable {
		t.Fatalf("ajuste ordinario menor que urgencia: %+v, %v", reglas, err)
	}
	if _, err := resolutor.Regla(t.Context(), BolsaPlazoRespuesta); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("regla inválida utilizable: %v", err)
	}
}

func TestVersionDeAjustesIncoherenteNoSeSustituyePorLaBase(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	buena := versionAjustes(t, 1, desde, map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "7"}})
	alteraciones := map[string]func(*VersionAjustes){
		"huella":   func(v *VersionAjustes) { v.HuellaSHA256 = "0" + v.HuellaSHA256[1:] },
		"catálogo": func(v *VersionAjustes) { v.CatalogoID = CatalogoContratacionTemporal },
		"versión":  func(v *VersionAjustes) { v.Version = 0 },
		"futura":   func(v *VersionAjustes) { v.VigenteDesde = diaPresentacion.Ahora().Add(time.Hour) },
		"campo inventado": func(v *VersionAjustes) {
			v.Ajustes = map[string]map[string]string{CTPlazoFiscalizacion: {"fases": "x"}}
		},
	}
	for nombre, alterar := range alteraciones {
		t.Run(nombre, func(t *testing.T) {
			v := buena
			alterar(&v)
			// La memoria filtra por catálogo e instante; aquí se devuelve tal cual.
			resolutor := resolutorCTConAjustes(t, consultaAjustesFija{v}, nil)
			if _, err := resolutor.Reglas(t.Context()); !errors.Is(err, ErrAjustesNoDisponibles) {
				t.Fatalf("aceptada: %v", err)
			}
		})
	}
	caida := resolutorCTConAjustes(t, &ajustesMemoria{err: errors.New("base caída")}, nil)
	if _, err := caida.Reglas(t.Context()); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("almacén caído: %v", err)
	}
	conflicto := resolutorCTConAjustes(t, &ajustesMemoria{err: errors.Join(errors.New("PostgreSQL"), ErrAjustesConflicto)}, nil)
	if _, err := conflicto.Reglas(t.Context()); !errors.Is(err, ErrAjustesConflicto) || errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("conflicto de ajustes colapsado: %v", err)
	}
	vacio := resolutorCTConAjustes(t, &ajustesMemoria{}, nil)
	if regla, err := vacio.Regla(t.Context(), CTPlazoFiscalizacion); err != nil || regla.Cantidad != 10 || regla.Ajuste != nil {
		t.Fatalf("sin versiones rige la base: %+v %v", regla, err)
	}
}

type consultaAjustesFija struct{ v VersionAjustes }

func (c consultaAjustesFija) AjustesVigentesEn(context.Context, string, time.Time) (VersionAjustes, bool, error) {
	return c.v, true, nil
}

func TestDeclaracionDeEdicionIncoherenteInvalidaLaRegla(t *testing.T) {
	base := func(extra map[string]string) map[string]string {
		a := map[string]string{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles",
			"cantidad": "5", "inicio": "contacto_efectivo", "computo": "administrativo"}
		for clave, valor := range extra {
			a[clave] = valor
		}
		return a
	}
	invalidas := map[string]map[string]string{
		"urgente mayor que ordinaria": {AtributoCantidadUrgente: "6"},
		"urgente no canónica":         {AtributoCantidadUrgente: "05"},
		"reglamento":                  {"origen": "reglamento", "articulo": "art. 1", "editable": "cantidad", "cantidad_minima": "1", "cantidad_maxima": "9"},
		"sin límites":                 {"editable": "cantidad"},
		"límites sin campo":           {"cantidad_minima": "1", "cantidad_maxima": "9"},
		"valor fuera":                 {"editable": "cantidad", "cantidad_minima": "6", "cantidad_maxima": "9"},
		"campo estructural":           {"editable": "fases"},
		"campo repetido":              {"editable": "cantidad,cantidad", "cantidad_minima": "1", "cantidad_maxima": "9"},
		"unidad sin opción":           {"editable": "unidad"},
		"unidad no plazo":             {"editable": "unidad", "opciones_unidad": "dias_habiles,horas"},
		"unidad actual fuera":         {"editable": "unidad", "opciones_unidad": "dias_naturales"},
		"cómputo inventado":           {"editable": "computo", "opciones_computo": "administrativo,lunar"},
	}
	for nombre, extra := range invalidas {
		t.Run(nombre, func(t *testing.T) {
			resolutor, err := NuevoResolutor(Configuracion{
				Consulta:   consultaMemoria{[]domain.CatalogoConfigurable{catalogoMemoria(t, 1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), base(extra))}},
				CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa, Reloj: diaPresentacion,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := resolutor.Reglas(t.Context()); !errors.Is(err, ErrReglaInvalida) {
				t.Fatalf("admitida: %v", err)
			}
		})
	}
}

func TestCanonicoDeAjustes(t *testing.T) {
	a := map[string]map[string]string{"c03.plazo_fiscalizacion": {"cantidad": "7", "unidad": "dias_habiles"}}
	canonico, err := CanonicoAjustes(a)
	if err != nil || string(canonico) != `{"c03.plazo_fiscalizacion":{"cantidad":"7","unidad":"dias_habiles"}}` {
		t.Fatalf("canónico: %s %v", canonico, err)
	}
	if vacio, err := CanonicoAjustes(nil); err != nil || string(vacio) != "{}" {
		t.Fatalf("vacío: %s %v", vacio, err)
	}
	for _, malo := range []map[string]map[string]string{
		{"C03": {"cantidad": "7"}}, {"c03": {}}, {"c03": {"fases": "x"}}, {"c03": {"cantidad": " 7"}},
		{"c": {CampoCantidad: "7"}}, {"c" + strings.Repeat("a", 80): {CampoCantidad: "7"}},
		{"c03": {CampoUnidad: "dias-habiles"}}, {"c03": {CampoUnidad: "Dias_habiles"}},
		{"c03": {CampoUnidad: "días_habiles"}}, {"c03": {CampoUnidad: "dias/habiles"}},
	} {
		if _, err := CanonicoAjustes(malo); !errors.Is(err, ErrAjusteInvalido) {
			t.Errorf("admitido %v", malo)
		}
	}
	demasiadosBytes := make(map[string]map[string]string, maximoReglasAjustadas)
	for i := 0; i < maximoReglasAjustadas; i++ {
		demasiadosBytes["c"+strconv.Itoa(i)] = map[string]string{
			CampoCantidad: strings.Repeat("1", 64), CampoCantidadUrgente: strings.Repeat("2", 64),
			CampoUnidad: strings.Repeat("a", 64), CampoComputo: strings.Repeat("b", 64),
		}
	}
	if _, err := CanonicoAjustes(demasiadosBytes); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("admitidos más de 16 KiB de ajustes: %v", err)
	}
}
