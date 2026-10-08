package reglas

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestValidarCatalogoBaseReglasIncluyeEntradasFuturasSinCambiarHuella(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	catalogo, _, err := resolutor.catalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	antes, huellaAntes, err := CanonicoCatalogoBaseReglas(catalogo)
	if err != nil {
		t.Fatalf("catálogo real no válido: %v", err)
	}
	if err := ValidarCatalogoBaseReglas(catalogo); err != nil {
		t.Fatalf("reglas reales no válidas: %v", err)
	}
	despues, huellaDespues, err := CanonicoCatalogoBaseReglas(catalogo)
	if err != nil || !bytes.Equal(antes, despues) || huellaAntes != huellaDespues {
		t.Fatal("la validación alteró el canónico o su huella")
	}
	modificado, err := catalogo.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	for indice := range modificado.Entradas {
		if modificado.Entradas[indice].Clave == CTPlazoSubsanacion {
			modificado.Entradas[indice].VigenteDesde = diaPresentacion.Ahora().AddDate(1, 0, 0)
			modificado.Entradas[indice].Atributos["unidad"] = "unidad_desconocida"
		}
	}
	if _, err := modificado.ClonarCanonico(); err != nil {
		t.Fatalf("la mutación debe superar la estructura genérica: %v", err)
	}
	if err := ValidarCatalogoBaseReglas(modificado); !errors.Is(err, ErrReglaInvalida) {
		t.Fatalf("entrada futura mal formada admitida: %v", err)
	}
}

func instantaneaPersistidaPrueba(t *testing.T, resolutor *Resolutor) InstantaneaPersistidaRegla {
	t.Helper()
	instantanea, err := resolutor.PrepararInstantaneaRegla(t.Context(), CTPlazoFiscalizacion)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := instantanea.Datos()
	if err != nil {
		t.Fatal(err)
	}
	catalogo, _, err := resolutor.catalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	base, huella, err := CanonicoCatalogoBaseReglas(catalogo)
	if err != nil || huella != datos.Base.HuellaCatalogo {
		t.Fatalf("catálogo base distinto del preparado: %v", err)
	}
	return InstantaneaPersistidaRegla{
		CatalogoBaseID: catalogo.ID, CatalogoBaseVersion: catalogo.Version,
		CatalogoBaseHuella: huella, CatalogoBaseCanonico: base,
		CatalogoAjustesID: datos.CatalogoAjustesID, AjustesEncontrados: datos.AjustesEncontrados,
		VersionAjustes: datos.VersionAjustes, HuellaAjustes: datos.HuellaAjustes,
		CanonicoAjustes: datos.CanonicoAjustes, AjustesVigenteDesde: datos.AjustesVigenteDesde,
		PreparadaEn: datos.PreparadaEn,
		Fase:        "fiscalizacion", FaseDesde: diaPresentacion.Ahora(),
	}
}

func TestRehidratarInstantaneaPersistidaConservaAusenciaYNoConsultaLaCabeza(t *testing.T) {
	almacen := &ajustesMemoria{}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{
		UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC),
	}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	p := instantaneaPersistidaPrueba(t, resolutor)
	if p.AjustesEncontrados || string(p.CanonicoAjustes) != "{}" {
		t.Fatal("ausencia no fijada")
	}
	consultas := len(almacen.pedidos)
	almacen.versiones = append(almacen.versiones, versionAjustes(t, 1, diaPresentacion.Ahora().Add(time.Second),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "7"}}))
	regla, _, err := resolutor.CalcularConInstantaneaPersistida(t.Context(), p, MunicipioSedeDiputacion, false)
	if err != nil || regla.Cantidad != 10 || calculadora.recibida.Cantidad != 10 || len(almacen.pedidos) != consultas {
		t.Fatalf("la cabeza alteró el plazo guardado: regla=%+v, solicitud=%+v, err=%v", regla, calculadora.recibida, err)
	}
}

func TestRehidratarInstantaneaPersistidaConservaAjusteYUrgencia(t *testing.T) {
	version := versionAjustes(t, 1, diaPresentacion.Ahora().Add(-time.Hour),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "7", CampoCantidadUrgente: "3"}})
	almacen := &ajustesMemoria{versiones: []VersionAjustes{version}}
	calculadora := &calculadoraFalsa{resultado: Vencimiento{
		UltimoDia: "2026-10-09", VenceAntesDe: time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC),
	}}
	resolutor := resolutorCTConAjustes(t, almacen, calculadora)
	p := instantaneaPersistidaPrueba(t, resolutor)
	almacen.versiones = append(almacen.versiones, versionAjustes(t, 2, diaPresentacion.Ahora().Add(time.Second),
		map[string]map[string]string{CTPlazoFiscalizacion: {CampoCantidad: "6", CampoCantidadUrgente: "2"}}))
	consultas := len(almacen.pedidos)
	regla, _, err := resolutor.CalcularConInstantaneaPersistida(t.Context(), p, MunicipioSedeDiputacion, true)
	if err != nil || regla.Cantidad != 7 || calculadora.recibida.Cantidad != 3 ||
		regla.Ajuste == nil || regla.Ajuste.Version != 1 || len(almacen.pedidos) != consultas {
		t.Fatalf("la versión guardada cambió: regla=%+v, solicitud=%+v, err=%v", regla, calculadora.recibida, err)
	}
}

func TestRehidratarInstantaneaPersistidaRechazaAlteraciones(t *testing.T) {
	resolutor := resolutorCTConAjustes(t, &ajustesMemoria{}, nil)
	valida := instantaneaPersistidaPrueba(t, resolutor)
	casos := map[string]func(*InstantaneaPersistidaRegla){
		"base alterada": func(p *InstantaneaPersistidaRegla) {
			p.CatalogoBaseCanonico = bytes.Replace(p.CatalogoBaseCanonico, []byte("plazo_fiscalizacion"), []byte("plazo_fiscalizacjon"), 1)
		},
		"huella base":                    func(p *InstantaneaPersistidaRegla) { p.CatalogoBaseHuella = "" },
		"version base":                   func(p *InstantaneaPersistidaRegla) { p.CatalogoBaseVersion++ },
		"huella ajustes":                 func(p *InstantaneaPersistidaRegla) { p.HuellaAjustes = "" },
		"ausencia convertida en version": func(p *InstantaneaPersistidaRegla) { p.VersionAjustes = 1 },
		"fecha perdida":                  func(p *InstantaneaPersistidaRegla) { p.FaseDesde = time.Time{} },
		"fase sin regla":                 func(p *InstantaneaPersistidaRegla) { p.Fase = "fase_inexistente" },
		"regla de otra fase":             func(p *InstantaneaPersistidaRegla) { p.ReglaClave = CTPlazoSubsanacion },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := valida
			p.CatalogoBaseCanonico = bytes.Clone(valida.CatalogoBaseCanonico)
			p.CanonicoAjustes = bytes.Clone(valida.CanonicoAjustes)
			alterar(&p)
			if _, err := RehidratarInstantaneaRegla(p); err == nil {
				t.Fatal("instantánea corrupta aceptada")
			}
		})
	}
	if _, _, err := resolutor.CalcularConInstantaneaPersistida(t.Context(), valida, "", false); err == nil {
		t.Fatal("municipio implícito admitido")
	}
}

func TestRehidratarInstantaneaPersistidaRechazaFaseAmbigua(t *testing.T) {
	resolutor := resolutorCTConAjustes(t, &ajustesMemoria{}, nil)
	p := instantaneaPersistidaPrueba(t, resolutor)
	var base domain.CatalogoConfigurable
	if err := json.Unmarshal(p.CatalogoBaseCanonico, &base); err != nil {
		t.Fatal(err)
	}
	for indice := range base.Entradas {
		if base.Entradas[indice].Clave == CTPlazoSubsanacion {
			base.Entradas[indice].Atributos["fases"] = "fiscalizacion"
		}
	}
	canonico, huella, err := CanonicoCatalogoBaseReglas(base)
	if err != nil {
		t.Fatal(err)
	}
	p.CatalogoBaseCanonico, p.CatalogoBaseHuella = canonico, huella
	if _, err := RehidratarInstantaneaRegla(p); err == nil {
		t.Fatal("dos reglas para la misma fase admitidas")
	}
}

func TestCacheInstantaneasPersistidasValidaCadaTramoYSeparaPares(t *testing.T) {
	resolutor := resolutorCTConAjustes(t, &ajustesMemoria{}, nil)
	primera := instantaneaPersistidaPrueba(t, resolutor)
	cache := NuevaCacheInstantaneasPersistidas()
	uno, err := cache.Rehidratar(primera)
	if err != nil || uno.datos.Efectiva.Cantidad != 10 {
		t.Fatalf("primera captura: %+v, %v", uno.datos.Efectiva, err)
	}
	segunda := primera
	var base domain.CatalogoConfigurable
	if err := json.Unmarshal(primera.CatalogoBaseCanonico, &base); err != nil {
		t.Fatal(err)
	}
	for i := range base.Entradas {
		if base.Entradas[i].Clave == CTPlazoFiscalizacion {
			base.Entradas[i].Atributos[CampoCantidad] = "7"
		}
	}
	segunda.CatalogoBaseCanonico, segunda.CatalogoBaseHuella, err = CanonicoCatalogoBaseReglas(base)
	if err != nil {
		t.Fatal(err)
	}
	dos, err := cache.Rehidratar(segunda)
	if err != nil || dos.datos.Efectiva.Cantidad != 7 || len(cache.entradas) != 2 {
		t.Fatalf("segundo par no aislado: %+v, %v, entradas=%d", dos.datos.Efectiva, err, len(cache.entradas))
	}
	alterada := primera
	alterada.CatalogoBaseCanonico = bytes.Clone(primera.CatalogoBaseCanonico)
	alterada.CatalogoBaseCanonico[0] ^= 1
	if _, err := cache.Rehidratar(alterada); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("bytes base alterados admitidos en acierto: %v", err)
	}
	alterada = primera
	alterada.CanonicoAjustes = []byte("{ }")
	if _, err := cache.Rehidratar(alterada); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("bytes de ajustes alterados admitidos en acierto: %v", err)
	}
	alterada = primera
	alterada.CatalogoBaseVersion++
	if _, err := cache.Rehidratar(alterada); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("versión alterada admitida en acierto: %v", err)
	}
	alterada = primera
	alterada.PreparadaEn = base.PublicadoEn.Add(-time.Second)
	if _, err := cache.Rehidratar(alterada); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("fecha anterior a publicación admitida: %v", err)
	}
	alterada = primera
	alterada.Fase = "subsanacion_unidad"
	if _, err := cache.Rehidratar(alterada); err != nil {
		t.Fatalf("otra fase del mismo catálogo no resuelta: %v", err)
	}
	var grupo sync.WaitGroup
	for i := 0; i < 16; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			if _, err := cache.Rehidratar(primera); err != nil {
				t.Errorf("captura concurrente: %v", err)
			}
		}()
	}
	grupo.Wait()
}

func TestCacheInstantaneasPersistidasConservaParSinReglaYNoConfiaEnMetadatos(t *testing.T) {
	resolutor := resolutorCTConAjustes(t, &ajustesMemoria{}, nil)
	valida := instantaneaPersistidaPrueba(t, resolutor)
	cache := NuevaCacheInstantaneasPersistidas()
	sinRegla := valida
	sinRegla.Fase = "fase_inexistente"
	for i := 0; i < 2; i++ {
		if _, err := cache.Rehidratar(sinRegla); !errors.Is(err, ErrReglaNoEncontrada) {
			t.Fatalf("fase sin regla %d: %v", i, err)
		}
		if len(cache.entradas) != 1 {
			t.Fatalf("la fase sin regla no conservó el par validado: %d", len(cache.entradas))
		}
	}
	// Un fallo de metadatos se resuelve por tramo. No invalida ni autoriza
	// una captura posterior que traiga el mismo par íntegro.
	invalida := valida
	invalida.CatalogoBaseVersion++
	if _, err := cache.Rehidratar(invalida); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("versión ajena admitida: %v", err)
	}
	if _, err := cache.Rehidratar(valida); err != nil {
		t.Fatalf("metadato ajeno contaminó la fase válida: %v", err)
	}
	// También al entrar por primera vez con metadatos erróneos se conserva
	// únicamente el par íntegro; la decisión del tramo vuelve a comprobarse.
	cache = NuevaCacheInstantaneasPersistidas()
	if _, err := cache.Rehidratar(invalida); !errors.Is(err, ErrReglasNoDisponibles) || len(cache.entradas) != 1 {
		t.Fatalf("primera captura errónea: %v, pares=%d", err, len(cache.entradas))
	}
	if _, err := cache.Rehidratar(valida); err != nil {
		t.Fatalf("par íntegro rechazado tras metadato erróneo: %v", err)
	}
}
