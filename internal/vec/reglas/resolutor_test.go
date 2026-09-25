package reglas

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	rutaReglasBolsaPrueba = "../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"
	rutaReglasCTPrueba    = "../../../data/demo/reglas/ct_reglas.ejemplo.demo.json"
)

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

var diaPresentacion = relojFijo(time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC))

type calculadoraFalsa struct {
	recibida  SolicitudVencimiento
	resultado Vencimiento
	err       error
}

func (c *calculadoraFalsa) CalcularVencimiento(_ context.Context, s SolicitudVencimiento) (Vencimiento, error) {
	c.recibida = s
	return c.resultado, c.err
}

func resolutorReal(t *testing.T, ruta, catalogo, modulo string, calculadora CalculadoraPlazos) *Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := NuevoResolutor(Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: catalogo, ModuloID: modulo,
		Reloj: diaPresentacion, Calculadora: calculadora, MunicipioSede: MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestResolutorDevuelveReglaTipadaConReferenciaYHuella(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, nil)
	regla, err := resolutor.Regla(t.Context(), BolsaPlazoRespuesta)
	if err != nil {
		t.Fatal(err)
	}
	if regla.Cantidad != 1 || regla.Unidad != UnidadDiasHabiles || regla.Computo != ComputoAdministrativo ||
		regla.Inicio != "contacto_efectivo" || regla.Origen != OrigenEjemplo || !regla.EsEjemplo() ||
		!regla.PaqueteEjemplo || regla.Paquete != MarcaPaqueteEjemplo ||
		regla.Referencia != "vec.bolsa.reglas:1:b05.plazo_respuesta" ||
		!regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(regla.HuellaCatalogo) ||
		regla.ReferenciaEntrada.CatalogoHuellaSHA256 != regla.HuellaCatalogo {
		t.Fatalf("regla inesperada: %+v", regla)
	}
	reglamento, err := resolutor.Regla(t.Context(), BolsaIntentosContacto)
	if err != nil || reglamento.Origen != OrigenReglamento || reglamento.Articulo != "art. 8.2.a" ||
		reglamento.EsEjemplo() || reglamento.Cantidad != 2 {
		t.Fatalf("regla del Reglamento inesperada: %+v %v", reglamento, err)
	}
	mixta, err := resolutor.Regla(t.Context(), BolsaSinRespuestaBaja)
	if err != nil || mixta.Origen != OrigenReglamento || !mixta.EsEjemplo() {
		t.Fatalf("una regla con parte de ejemplo debe rotularse como ejemplo: %+v %v", mixta, err)
	}
	documentos, err := resolutor.Regla(t.Context(), BolsaDocumentosIncorporacion)
	if err != nil || len(documentos.Elementos()) != 5 {
		t.Fatalf("lista inesperada: %+v %v", documentos, err)
	}
	mixta.Atributos["origen"] = "alterado"
	if otra, _ := resolutor.Regla(t.Context(), BolsaSinRespuestaBaja); otra.Atributos["origen"] != "reglamento" {
		t.Fatal("los atributos devueltos deben ser una copia")
	}
}

func TestResolutorResuelveTodasLasClavesPublicadas(t *testing.T) {
	casos := []struct {
		ruta, catalogo, modulo string
		claves                 []string
	}{
		{rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, []string{
			BolsaOrdenPrelacion, BolsaIntentosContacto, BolsaSeparacionIntentos, BolsaProcesosSinContacto,
			BolsaFranjaLlamadas, BolsaPlazoRespuesta, BolsaCorreoNoAbrePlazo, BolsaFueraDePlazo,
			BolsaSinRespuestaBaja, BolsaSiguienteCandidato, BolsaPlazoPublicacion,
			BolsaAcreditarRenunciaJustificada, BolsaPeriodoMatrimonio, BolsaRenunciaNoJustificada,
			BolsaRenunciaNombramientoEnCurso, BolsaReposicionGeneral, BolsaReposicionAcumulacionTareas,
			BolsaRecuperaPosicion, BolsaPrestaServicios, BolsaAvisoEncadenamiento, BolsaPausaVoluntaria,
			BolsaVacanteDuracionMaxima, BolsaSAEDuracionMaxima, BolsaPlazoDocumentacion,
			BolsaDocumentosIncorporacion, BolsaPlazoIncorporacion, BolsaConsecuencias, BolsaVigencia,
			BolsaAgotamiento, BolsaCausaBajaNoAcepta, BolsaCausaBajaNoSePresenta,
			BolsaCausaBajaNoAportaDocumentacion, BolsaCausaBajaSinContacto,
			BolsaCausaBajaRenunciaTrasDisposicion, BolsaCausaBajaRenunciaNombramiento,
			BolsaTransicionesRenuncia,
			BolsaEstadosRecurso,
			BolsaPrefijoSanciones + "baja_llamamiento_directo", BolsaPrefijoSanciones + "baja_sin_contacto",
			BolsaPrefijoSanciones + "baja_publicacion", BolsaPrefijoSanciones + "baja_renuncia_nombramiento",
			BolsaPrefijoSanciones + "pasar_al_final", BolsaPrefijoSanciones + "suspension",
		}},
		{rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, []string{
			CTPlazoAnalisis, CTPlazoInformes, CTPlazoFiscalizacion, CTPlazoSubsanacion, CTMotivosRectificacion,
			CTAltaSeguridadSocial, CTJornadaCompleta, CTDuracionAcumulacionTareas, CTDuracionProgramasTemporales,
			CTDuracionVacante, CTDuracionSustitucion, CTDuracionCircunstancias,
		}},
	}
	for _, caso := range casos {
		resolutor := resolutorReal(t, caso.ruta, caso.catalogo, caso.modulo, nil)
		todas, err := resolutor.Reglas(t.Context())
		if err != nil || len(todas) != len(caso.claves) {
			t.Fatalf("%s: %d reglas, se esperaban %d (%v)", caso.catalogo, len(todas), len(caso.claves), err)
		}
		for _, clave := range caso.claves {
			if _, err := resolutor.Regla(t.Context(), clave); err != nil {
				t.Errorf("%s: %s no se resuelve: %v", caso.catalogo, clave, err)
			}
		}
	}
}

func TestResolutorCalculaVencimientoConLaCalculadora(t *testing.T) {
	calculadora := &calculadoraFalsa{resultado: Vencimiento{
		UltimoDia:    "2026-09-29",
		VenceAntesDe: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC),
		Calendarios:  []string{"cal:1"},
	}}
	resolutor := resolutorReal(t, rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, calculadora)
	contacto := time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC)
	regla, vencimiento, err := resolutor.Vencimiento(t.Context(), BolsaPlazoRespuesta, contacto, "")
	if err != nil {
		t.Fatal(err)
	}
	if calculadora.recibida != (SolicitudVencimiento{
		Inicio: contacto, Unidad: UnidadDiasHabiles, Cantidad: 1,
		Computo: ComputoAdministrativo, MunicipioSede: MunicipioSedeDiputacion,
	}) {
		t.Fatalf("solicitud inesperada: %+v", calculadora.recibida)
	}
	if regla.Clave != BolsaPlazoRespuesta || vencimiento.UltimoDia != "2026-09-29" || len(vencimiento.Calendarios) != 1 {
		t.Fatalf("vencimiento inesperado: %+v %+v", regla, vencimiento)
	}
	if _, _, err := resolutor.Vencimiento(t.Context(), BolsaReposicionGeneral, contacto, "municipio:ine:18098"); err != nil ||
		calculadora.recibida.Computo != ComputoCivil || calculadora.recibida.Unidad != UnidadMeses ||
		calculadora.recibida.Cantidad != 5 || calculadora.recibida.MunicipioSede != "municipio:ine:18098" {
		t.Fatalf("cómputo civil inesperado: %+v %v", calculadora.recibida, err)
	}
	for _, sinPlazo := range []string{BolsaIntentosContacto, BolsaAvisoEncadenamiento, BolsaFranjaLlamadas} {
		if _, _, err := resolutor.Vencimiento(t.Context(), sinPlazo, contacto, ""); !errors.Is(err, ErrReglaSinPlazo) {
			t.Errorf("%s debe rechazar el cálculo: %v", sinPlazo, err)
		}
	}
	calculadora.err = errors.New("calendario sin cobertura")
	if _, _, err := resolutor.Vencimiento(t.Context(), BolsaPlazoRespuesta, contacto, ""); !errors.Is(err, ErrCalculoNoDisponible) {
		t.Fatalf("un fallo del calendario no puede ocultarse: %v", err)
	}
	calculadora.err = nil
	calculadora.resultado = Vencimiento{UltimoDia: "2026-09-27", VenceAntesDe: contacto.Add(-time.Hour)}
	if _, _, err := resolutor.Vencimiento(t.Context(), BolsaPlazoRespuesta, contacto, ""); !errors.Is(err, ErrCalculoNoDisponible) {
		t.Fatalf("un vencimiento anterior al inicio debe rechazarse: %v", err)
	}
	sinCalculadora := resolutorReal(t, rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, nil)
	if _, _, err := sinCalculadora.Vencimiento(t.Context(), BolsaPlazoRespuesta, contacto, ""); !errors.Is(err, ErrCalculoNoDisponible) {
		t.Fatalf("sin calculadora no hay vencimiento: %v", err)
	}
}

func TestResolutorSinCatalogoOFueraDeVigenciaFallaCerrado(t *testing.T) {
	var nulo *Resolutor
	if nulo.Disponible() {
		t.Fatal("un resolutor nulo no está disponible")
	}
	if _, err := nulo.Regla(t.Context(), BolsaPlazoRespuesta); !errors.Is(err, ErrReglasNoConfiguradas) {
		t.Fatalf("sin catálogo: %v", err)
	}
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasBolsaPrueba)
	if err != nil {
		t.Fatal(err)
	}
	casos := []Configuracion{
		{Consulta: consulta, CatalogoID: CatalogoBolsa, ModuloID: ModuloContratacionTemporal, Reloj: diaPresentacion},
		{Consulta: consulta, CatalogoID: CatalogoContratacionTemporal, ModuloID: ModuloBolsa, Reloj: diaPresentacion},
		{Consulta: consulta, CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa,
			Reloj: relojFijo(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))},
	}
	for indice, cfg := range casos {
		resolutor, err := NuevoResolutor(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := resolutor.Regla(t.Context(), BolsaPlazoRespuesta); !errors.Is(err, ErrReglasNoDisponibles) {
			t.Errorf("caso %d: %v", indice, err)
		}
	}
	if _, err := NuevoResolutor(Configuracion{Consulta: consulta, CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa}); !errors.Is(err, ErrConfiguracion) {
		t.Fatalf("sin reloj: %v", err)
	}
	if _, err := resolutorReal(t, rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, nil).Regla(t.Context(), "b99.inexistente"); !errors.Is(err, ErrReglaNoEncontrada) {
		t.Fatalf("clave inexistente: %v", err)
	}
	cancelado, cancelar := context.WithCancel(t.Context())
	cancelar()
	if _, err := resolutorReal(t, rutaReglasBolsaPrueba, CatalogoBolsa, ModuloBolsa, nil).Reglas(cancelado); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado: %v", err)
	}
}

type consultaMemoria struct{ catalogos []domain.CatalogoConfigurable }

func (c consultaMemoria) ObtenerCatalogoAcotado(context.Context, string, int, ports.LimitesConsultaCatalogosAcotada) (ports.ResultadoConsultaCatalogoAcotado, error) {
	return ports.ResultadoConsultaCatalogoAcotado{}, ports.ErrCatalogoNoEncontrado
}

func (c consultaMemoria) ListarVersionesCatalogoAcotado(context.Context, string, ports.LimitesConsultaCatalogosAcotada) (ports.ResultadoConsultaCatalogosAcotada, error) {
	return ports.ResultadoConsultaCatalogosAcotada{Catalogos: c.catalogos}, nil
}

func catalogoMemoria(t *testing.T, version int, publicado time.Time, atributos map[string]string) domain.CatalogoConfigurable {
	t.Helper()
	referenciaAnterior := ""
	if version > 1 {
		referenciaAnterior = "vec.bolsa.reglas:" + strconv.Itoa(version-1)
	}
	borrador := domain.CatalogoConfigurable{
		ID: CatalogoBolsa, Version: version, Revision: 1, VersionAnteriorRef: referenciaAnterior,
		ModuloID: ModuloBolsa, Nombre: "Reglas", FuenteRef: MarcaPaqueteEjemplo, MotivoCreacion: "Prueba.",
		Entradas: []domain.EntradaCatalogoConfigurable{{
			Clave: BolsaPlazoRespuesta, Etiqueta: "Plazo", VigenteDesde: publicado, Atributos: atributos,
		}},
		Estado: domain.EstadoCatalogoBorrador, CreadoPor: "prueba", CreadoEn: publicado,
	}
	publicadoCat, err := borrador.Publicar("revisor", "sin_aprobacion", "Prueba.", publicado)
	if err != nil {
		t.Fatal(err)
	}
	return publicadoCat
}

func TestResolutorEligeLaVersionVigenteYRechazaAtributosInvalidos(t *testing.T) {
	base := map[string]string{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles",
		"cantidad": "1", "inicio": "contacto_efectivo", "computo": "administrativo"}
	segunda := map[string]string{}
	for clave, valor := range base {
		segunda[clave] = valor
	}
	segunda["cantidad"] = "2"
	futura := map[string]string{}
	for clave, valor := range base {
		futura[clave] = valor
	}
	futura["cantidad"] = "3"
	catalogos := []domain.CatalogoConfigurable{
		catalogoMemoria(t, 3, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), futura),
		catalogoMemoria(t, 1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), base),
		catalogoMemoria(t, 2, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), segunda),
	}
	resolutor, err := NuevoResolutor(Configuracion{Consulta: consultaMemoria{catalogos}, CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa, Reloj: diaPresentacion})
	if err != nil {
		t.Fatal(err)
	}
	regla, err := resolutor.Regla(t.Context(), BolsaPlazoRespuesta)
	if err != nil || regla.Cantidad != 2 || regla.Referencia != "vec.bolsa.reglas:2:b05.plazo_respuesta" || !regla.PaqueteEjemplo {
		t.Fatalf("versión vigente inesperada: %+v %v", regla, err)
	}
	invalidos := []map[string]string{
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "semanas", "cantidad": "1"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles", "cantidad": "01"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles", "cantidad": "0"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "ninguna", "cantidad": "1"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "franja_horaria", "valor": "14:00-09:00"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "dias_habiles", "cantidad": "1", "inicio": "x"},
		{"origen": "ejemplo", "norma": "N", "duda": "D", "unidad": "horas", "cantidad": "1", "inicio": "x", "computo": "civil"},
		{"origen": "reglamento", "norma": "N", "duda": "D", "unidad": "ninguna"},
		{"origen": "ejemplo", "articulo": "art. 1", "norma": "N", "duda": "D", "unidad": "ninguna"},
		{"origen": "otro", "norma": "N", "duda": "D", "unidad": "ninguna"},
		{"origen": "ejemplo", "duda": "D", "unidad": "ninguna"},
	}
	for indice, atributos := range invalidos {
		resolutor, err := NuevoResolutor(Configuracion{
			Consulta:   consultaMemoria{[]domain.CatalogoConfigurable{catalogoMemoria(t, 1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), atributos)}},
			CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa, Reloj: diaPresentacion,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := resolutor.Reglas(t.Context()); !errors.Is(err, ErrReglaInvalida) {
			t.Errorf("caso %d aceptado: %v", indice, err)
		}
	}
}
