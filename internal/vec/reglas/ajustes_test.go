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

// La consulta retrospectiva es solo una vista; el cálculo operativo exige
// una instantánea que CT deberá guardar al iniciar el plazo.
func TestCalculoConAjustesExigeInstantaneaExplicita(t *testing.T) {
	cambio := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	almacen := &ajustesMemoria{versiones: []VersionAjustes{versionAjustes(t, 1, cambio, map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})}}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	antes := cambio.Add(-time.Hour)
	despues := cambio.Add(time.Hour)
	for _, inicio := range []time.Time{antes, despues} {
		if _, _, err := resolutor.Vencimiento(t.Context(), CTPlazoFiscalizacion, inicio, ""); !errors.Is(err, ErrAjustesNoDisponibles) {
			t.Fatalf("cálculo sin instantánea admitido: %v", err)
		}
		if _, _, err := resolutor.VencimientoUrgente(t.Context(), CTPlazoFiscalizacion, inicio, ""); !errors.Is(err, ErrAjustesNoDisponibles) {
			t.Fatalf("cálculo urgente sin instantánea admitido: %v", err)
		}
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
	if calculadora.recibida.Cantidad != 0 {
		t.Fatal("la calculadora recibió un plazo sin instantánea")
	}
}

func TestInstantaneaConAusenciaEstableEInmutable(t *testing.T) {
	almacen := &ajustesMemoria{}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	instantanea, err := resolutor.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := instantanea.Datos()
	huellaVacia, _ := HuellaAjustes(nil)
	if err != nil || datos.AjustesEncontrados || datos.VersionAjustes != 0 ||
		datos.HuellaAjustes != huellaVacia || string(datos.CanonicoAjustes) != "{}" ||
		datos.Base.Cantidad != 10 || datos.Efectiva.Cantidad != 10 {
		t.Fatalf("ausencia no fijada: %+v, %v", datos, err)
	}
	datos.Base.Atributos[CampoCantidad] = "999"
	datos.Efectiva.Atributos[CampoCantidad] = "999"
	datos.Efectiva.Edicion.Campos[0] = "fases"
	datos.CanonicoAjustes[0] = 'x'
	otra, err := instantanea.Datos()
	if err != nil || otra.Base.Atributos[CampoCantidad] != "10" ||
		otra.Efectiva.Atributos[CampoCantidad] != "10" || otra.Efectiva.Edicion.Campos[0] != CampoCantidad ||
		string(otra.CanonicoAjustes) != "{}" {
		t.Fatalf("la copia mutó la instantánea: %+v, %v", otra, err)
	}
	consultas := len(almacen.pedidos)
	almacen.versiones = append(almacen.versiones, versionAjustes(t, 1, diaPresentacion.Ahora().Add(time.Second),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "7"}}))
	regla, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, diaPresentacion.Ahora().Add(time.Minute), "", false)
	if err != nil || regla.Cantidad != 10 || calculadora.recibida.Cantidad != 10 || len(almacen.pedidos) != consultas {
		t.Fatalf("la ausencia se recalculó: regla=%+v, pedido=%+v, err=%v", regla, calculadora.recibida, err)
	}
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), InstantaneaRegla{}, diaPresentacion.Ahora(), "", false); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("instantánea vacía admitida: %v", err)
	}
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, time.Time{}, "", false); !errors.Is(err, ErrReglaSinPlazo) {
		t.Fatalf("inicio vacío admitido: %v", err)
	}
	// El instante SQL que identifica el inicio de una transacción puede
	// preceder al reloj Go de preparación; su vínculo lo acreditará CT110.
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, diaPresentacion.Ahora().Add(-time.Second), "", false); err != nil {
		t.Fatalf("se inventó una relación temporal entre relojes: %v", err)
	}
	sinAlmacen := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, calculadora)
	if _, err := sinAlmacen.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("instantánea sin fuente de ajustes admitida: %v", err)
	}
}

func TestInstantaneaConVersionAunqueLaReglaNoCambie(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	version := versionAjustes(t, 1, desde, map[string]map[string]string{CTPlazoSubsanacion: {CampoCantidad: "7"}})
	almacen := &ajustesMemoria{versiones: []VersionAjustes{version}}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	instantanea, err := resolutor.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := instantanea.Datos()
	if err != nil || !datos.AjustesEncontrados || datos.VersionAjustes != 1 ||
		datos.HuellaAjustes != version.HuellaSHA256 || !datos.AjustesVigenteDesde.Equal(desde) ||
		datos.Efectiva.Ajuste != nil || datos.Efectiva.Cantidad != 10 ||
		datos.Base.ReferenciaEntrada.CatalogoHuellaSHA256 != datos.Efectiva.HuellaCatalogo {
		t.Fatalf("versión sin cambio perdida: %+v, %v", datos, err)
	}
	almacen.versiones = append(almacen.versiones, versionAjustes(t, 2, diaPresentacion.Ahora().Add(time.Second),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "8"}}))
	consultas := len(almacen.pedidos)
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, diaPresentacion.Ahora().Add(time.Minute), "", false); err != nil ||
		calculadora.recibida.Cantidad != 10 || len(almacen.pedidos) != consultas {
		t.Fatalf("instantánea sin cambio reconsultada: %+v, %v", calculadora.recibida, err)
	}
}

func TestInstantaneaAjustadaConservaValorYUrgencia(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	version := versionAjustes(t, 1, desde, map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "7"}})
	almacen := &ajustesMemoria{versiones: []VersionAjustes{version}}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	instantanea, err := resolutor.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := instantanea.Datos()
	if err != nil || datos.Base.Cantidad != 10 || datos.Efectiva.Cantidad != 7 ||
		datos.Efectiva.Ajuste == nil || datos.HuellaAjustes != version.HuellaSHA256 {
		t.Fatalf("valor ajustado no fijado: %+v, %v", datos, err)
	}
	datos.Efectiva.Ajuste.Campos[CampoCantidad] = "999"
	almacen.versiones = append(almacen.versiones, versionAjustes(t, 2, diaPresentacion.Ahora().Add(time.Second),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "6"}}))
	consultas := len(almacen.pedidos)
	inicio := diaPresentacion.Ahora().Add(time.Minute)
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, inicio, "", false); err != nil ||
		calculadora.recibida.Cantidad != 7 || len(almacen.pedidos) != consultas {
		t.Fatalf("valor ajustado reconsultado: %+v, %v", calculadora.recibida, err)
	}
	if _, _, err := resolutor.CalcularConInstantanea(t.Context(), instantanea, inicio, "", true); err != nil ||
		calculadora.recibida.Cantidad != 5 {
		t.Fatalf("urgencia de la base perdida: %+v, %v", calculadora.recibida, err)
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
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	if _, _, err := conflicto.ajustesEn(ctx, diaPresentacion.Ahora()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación no tuvo prioridad sobre conflicto: %v", err)
	}
	ausenciaIncoherente := resolutorCTConAjustes(t, consultaAjustesAusencia{buena}, nil)
	if _, err := ausenciaIncoherente.Reglas(t.Context()); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("versión oculta detrás de encontrada=false: %v", err)
	}
	if _, err := ausenciaIncoherente.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("instantánea convirtió versión oculta en ausencia: %v", err)
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

type consultaAjustesAusencia struct{ v VersionAjustes }

func (c consultaAjustesAusencia) AjustesVigentesEn(context.Context, string, time.Time) (VersionAjustes, bool, error) {
	return c.v, false, nil
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

func TestPrepararCambioDerivaAnteriorDeBaseYCanonicoCompleto(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, instante, err := resolutor.catalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := PrepararCambioAjustes(base, instante, 0, VersionAjustes{}, false,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}})
	if err != nil {
		t.Fatal(err)
	}
	datos := preparada.Datos()
	if datos.VersionEsperada != 0 || datos.BaseVersion != base.Version ||
		len(datos.Cambios) != 1 || datos.Cambios[0].Anterior != "10" || datos.Cambios[0].Nuevo != "7" ||
		string(datos.Canonico) != `{"c03.plazo_fiscalizacion":{"cantidad":"7"}}` {
		t.Fatalf("primer cambio no deriva la base exacta: %+v", datos)
	}
	huella, err := HuellaAjustes(datos.Ajustes)
	if err != nil || datos.HuellaSHA256 != huella {
		t.Fatalf("huella no canónica: %s %v", datos.HuellaSHA256, err)
	}
	datos.Ajustes[CTPlazoFiscalizacion][CampoCantidad] = "99"
	datos.Canonico[0] = 'x'
	datos.Cambios[0].Anterior = "inventado"
	otra := preparada.Datos()
	if otra.Cambios[0].Anterior != "10" || otra.Ajustes[CTPlazoFiscalizacion][CampoCantidad] != "7" ||
		string(otra.Canonico) != `{"c03.plazo_fiscalizacion":{"cantidad":"7"}}` {
		t.Fatalf("la preparación conservó alias mutable: %+v", otra)
	}
	// Tras publicar, la cabeza ya es v1. El material de la solicitud original
	// conserva versión esperada 0; reprocesarlo desde esa cabeza sería otro
	// material y CT148 rechazaría la clave reutilizada.
	publicada := versionAjustes(t, 1, instante, preparada.Datos().Ajustes)
	repetida, err := PrepararCambioAjustes(base, instante, 0, VersionAjustes{}, false,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}})
	if err != nil || repetida.Datos().VersionEsperada != 0 ||
		string(repetida.Datos().Canonico) != string(preparada.Datos().Canonico) {
		t.Fatalf("no se conservó el material de la solicitud original: %v", err)
	}
	if _, err := PrepararCambioAjustes(base, instante, 0, publicada, true,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}}); !errors.Is(err, ErrAjustesConflicto) {
		t.Fatalf("se recalculó un replay desde la cabeza avanzada: %v", err)
	}
}

func TestPrepararCambioConVersionPreviaConservaOtrosCampos(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, instante, err := resolutor.catalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	previa := versionAjustes(t, 1, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7", CampoCantidadUrgente: "3"},
		CTPlazoSubsanacion:   {CampoCantidad: "8"},
	})
	preparada, err := PrepararCambioAjustes(base, instante, 1, previa, true,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "9"}})
	if err != nil {
		t.Fatal(err)
	}
	datos := preparada.Datos()
	if datos.VersionEsperada != 1 || datos.Cambios[0].Anterior != "7" ||
		datos.Ajustes[CTPlazoFiscalizacion][CampoCantidadUrgente] != "3" ||
		datos.Ajustes[CTPlazoSubsanacion][CampoCantidad] != "8" {
		t.Fatalf("perdió la versión completa: %+v", datos)
	}
	canonico, err := CanonicoAjustes(datos.Ajustes)
	if err != nil || string(canonico) != string(datos.Canonico) {
		t.Fatalf("material completo divergente: %s, %v", datos.Canonico, err)
	}
	// CT148 rechaza una lista vacía antes de mirar la clave de idempotencia.
	if _, err := PrepararCambioAjustes(base, instante, 1, previa, true, nil); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("petición vacía admitida: %v", err)
	}
	for nombre, caso := range map[string]struct {
		version int
		previa  VersionAjustes
		con     bool
		nuevo   string
	}{
		"igual a la base":        {version: 0, nuevo: "10"},
		"igual al ajuste previo": {version: 1, previa: previa, con: true, nuevo: "7"},
	} {
		t.Run(nombre, func(t *testing.T) {
			_, err := PrepararCambioAjustes(base, instante, caso.version, caso.previa, caso.con,
				[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: caso.nuevo}})
			if !errors.Is(err, ErrAjusteInvalido) {
				t.Fatalf("valor nuevo igual al anterior admitido: %v", err)
			}
		})
	}
	corrupta := previa
	corrupta.HuellaSHA256 = strings.Repeat("0", 64)
	if _, err := PrepararCambioAjustes(base, instante, 1, corrupta, true,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "9"}}); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("versión previa sin huella aceptada: %v", err)
	}
	for _, solicitud := range [][]SolicitudCambioAjuste{
		{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "61"}},
		{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "9"},
			{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "8"}},
	} {
		if _, err := PrepararCambioAjustes(base, instante, 1, previa, true, solicitud); !errors.Is(err, ErrAjusteInvalido) {
			t.Fatalf("cambio inválido aceptado: %+v, %v", solicitud, err)
		}
	}
}

func TestPrepararAjustesSobreCabezaFuturaConservaCamposYExigeFecha(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, ahora, err := resolutor.catalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	cabeza := versionAjustes(t, 2, ahora.Add(24*time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7", CampoCantidadUrgente: "3"},
		CTPlazoSubsanacion:   {CampoCantidad: "8"},
	})
	efecto := ahora.Add(48*time.Hour + 1234*time.Nanosecond).In(time.FixedZone("CET", 3600))
	solicitud := []SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "9"}}
	preparada, err := PrepararAjustesSobreVersion(base, ahora, &efecto, 2, cabeza, true, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	d := preparada.Datos()
	if d.VersionEsperada != 2 || d.Cambios[0].Anterior != "7" ||
		d.Ajustes[CTPlazoFiscalizacion][CampoCantidadUrgente] != "3" ||
		d.Ajustes[CTPlazoSubsanacion][CampoCantidad] != "8" || d.EfectoDesde == nil ||
		d.EfectoDesde.Location() != time.UTC || !d.EfectoDesde.Equal(efecto.UTC().Truncate(time.Microsecond)) {
		t.Fatalf("cabeza futura o fecha no conservadas: %+v", d)
	}
	*d.EfectoDesde = ahora
	if otra := preparada.Datos(); !otra.EfectoDesde.Equal(efecto.UTC().Truncate(time.Microsecond)) {
		t.Fatal("la fecha compartió un puntero mutable")
	}
	if _, err := PrepararAjustesSobreVersion(base, ahora, nil, 2, cabeza, true, solicitud); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("cabeza futura adelantada por omisión: %v", err)
	}
	retroactiva := ahora.Add(-time.Hour)
	if _, err := PrepararAjustesSobreVersion(base, ahora, &retroactiva, 2, cabeza, true, solicitud); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("fecha retroactiva admitida: %v", err)
	}
	corrupta := cabeza
	corrupta.HuellaSHA256 = strings.Repeat("0", 64)
	if _, err := PrepararAjustesSobreVersion(base, ahora, &efecto, 2, corrupta, true, solicitud); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("cabeza sin integridad admitida: %v", err)
	}
	if _, err := PrepararAjustesSobreVersion(base, ahora, &efecto, 2, cabeza, true,
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "61"}}); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("ajuste inválido admitido: %v", err)
	}
	if cabeza.Ajustes[CTPlazoFiscalizacion][CampoCantidad] != "7" || cabeza.Ajustes[CTPlazoSubsanacion][CampoCantidad] != "8" {
		t.Fatal("se modificó la fuente")
	}
}
