package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const referenciaLiquidacionPrueba = "dco_aaaaaaaaaaaaaaaaaaaaaa"

type preparadorLiquidacionHTTPPrueba struct {
	llamadas int
	ref      string
	version  uint64
	revisar  []domain.RevisionLineaLiquidacion
	preparar *domain.PreparacionLiquidacionPublicada
	err      error
}

func (p *preparadorLiquidacionHTTPPrueba) PrepararLiquidacionEconomica(_ context.Context, ref string, version uint64, revisar []domain.RevisionLineaLiquidacion) (*domain.PreparacionLiquidacionPublicada, error) {
	p.llamadas++
	p.ref, p.version, p.revisar = ref, version, revisar
	return p.preparar, p.err
}

type identidadLiquidacionHTTPPrueba struct {
	llamadas int
	s        ports.SolicitudLiquidacionEconomica
	err      error
}

func (i *identidadLiquidacionHTTPPrueba) ResolverRegistroLiquidacionEconomica(_ context.Context, s ports.SolicitudLiquidacionEconomica) (ports.IdentidadEfectivaCircuito, error) {
	i.llamadas++
	i.s = s
	return ports.IdentidadEfectivaCircuito{UnidadCompetenciaRef: "unidad:rrhh"}, i.err
}

type registroLiquidacionHTTPPrueba struct {
	llamadas  int
	s         ports.SolicitudLiquidacionEconomica
	resultado *domain.LiquidacionEconomica
}

func (p *registroLiquidacionHTTPPrueba) RegistrarPreparada(_ context.Context, _ ports.IdentidadEfectivaCircuito, s ports.SolicitudLiquidacionEconomica) (*domain.LiquidacionEconomica, error) {
	p.llamadas++
	p.s = s
	return p.resultado, nil
}

type capturaLiquidacionHTTPPrueba struct {
	llamadas int
	ref      string
	err      error
	contexto []byte
}

func (c *capturaLiquidacionHTTPPrueba) CapturarLecturaLiquidacionEconomica(_ context.Context, ref string) (ports.CapturaLecturaLiquidacionEconomica, error) {
	c.llamadas++
	c.ref = ref
	return ports.CapturaLecturaLiquidacionEconomica{ContextoCanonico: c.contexto}, c.err
}

type lectorLiquidacionHTTPPrueba struct {
	llamadas  int
	consulta  ports.ConsultaLiquidacionEconomica
	resultado ports.ResultadoLecturaLiquidacionEconomica
	err       error
}

func (l *lectorLiquidacionHTTPPrueba) Consultar(_ context.Context, _ ports.CapturaLecturaLiquidacionEconomica, q ports.ConsultaLiquidacionEconomica) (ports.ResultadoLecturaLiquidacionEconomica, error) {
	l.llamadas++
	l.consulta = q
	return l.resultado, l.err
}

func contextoLiquidacionHTTPPrueba(t *testing.T, letra string) []byte {
	t.Helper()
	referencia := strings.Repeat(letra, 24)
	ahora := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + referencia,
		Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + referencia,
		VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1,
		PersonaRef: "per_" + referencia, PersonaVersion: 1,
		PerfilActivoRef: "prf_" + referencia, PerfilVersion: 1,
		Estado:       vecdomain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + referencia,
			Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado,
			Referencia: "emp_" + referencia, Estado: vecdomain.EstadoVinculoContextoActorActivo,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	return canonico
}

func preparacionLiquidacionHTTPPrueba(t *testing.T) *domain.PreparacionLiquidacionPublicada {
	t.Helper()
	indice := 0
	d := domain.DocumentoComision{GrupoDieta: 2, VersionTarifaAceptada: "tarifa:publicada:v1", TramosAceptados: []int{0},
		Lineas:              []domain.LineaDocumentoComision{{Tipo: "dieta", Grupo: 2, IndiceTramo: &indice, Fecha: "2026-10-01", Concepto: "manutencion", ImporteCentimos: 500, VersionTarifaRef: "tarifa:publicada:v1", Rotulo: "dietas_tarifa_publicada"}},
		ManutencionCentimos: 500, TotalOrientativoCentimos: 500}
	dh, err := domain.HuellaDatosLiquidacion(d)
	if err != nil {
		t.Fatal(err)
	}
	c := domain.CatalogoLiquidacionPublicado{Referencia: "catalogo:liquidacion:publicado", Version: "version:uno", PublicacionRef: "publicacion:acto:uno",
		FechaPublicacion: "2026-09-20", VigenteDesde: "2026-10-01", VigenteHasta: "2026-12-31", Ambito: "granada:ordinario", PaisISO2: "ES", Grupo: 2,
		Vehiculo: "sin_vehiculo_propio", VersionTarifaRef: "tarifa:publicada:v1", RotuloClave: "dietas_tarifa_publicada",
		Normas: []domain.NormaLiquidacionPublicada{{Referencia: "norma:bases", Articulo: "30", URL: "https://www.dipgra.es/bases"}, {Referencia: "norma:boe", Articulo: "10", URL: "https://www.boe.es/boe"}},
		Reglas: []domain.ReglaLiquidacionPublicada{{ReglaLiquidacionPropuesta: domain.ReglaLiquidacionPropuesta{Referencia: "regla:manutencion:dos", Tipo: "dieta", Concepto: "manutencion", Grupo: 2, TopeCentimos: 1000}, NormaRef: "norma:bases"}}}
	c.ReglaDevengo = domain.ReglaDevengoPublicada{Referencia: "regla:devengo:uno", NormaRef: "norma:bases", Configuracion: domain.ConfiguracionDevengoPublicada{
		Esquema: "nacional_ordinaria_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5,
		HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14,
		HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, PorcentajeMismoDia: 50,
		PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50,
		PorcentajeIntermedio: 100, Alojamiento: "cerrado_pendiente_rrhh"}}
	c.ReglaDevengo.HuellaSHA256, err = c.ReglaDevengo.HuellaCalculada()
	if err != nil {
		t.Fatal(err)
	}
	c.HuellaSHA256, err = c.HuellaCalculada()
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.PrepararLiquidacionPublicada(referenciaLiquidacionPrueba, 2, "2026-10-01", "08:00", "2026-10-01", "18:00", dh, d, c,
		[]domain.RevisionLineaLiquidacion{{Indice: 0, ReglaRef: "regla:manutencion:dos", ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_importe"}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func manejadorLiquidacionHTTPPrueba(t *testing.T) (*ManejadorLiquidacionEconomica, *preparadorLiquidacionHTTPPrueba, *identidadLiquidacionHTTPPrueba, *registroLiquidacionHTTPPrueba, *capturaLiquidacionHTTPPrueba, *lectorLiquidacionHTTPPrueba) {
	t.Helper()
	p := &preparadorLiquidacionHTTPPrueba{preparar: preparacionLiquidacionHTTPPrueba(t)}
	i, r, c, l := &identidadLiquidacionHTTPPrueba{}, &registroLiquidacionHTTPPrueba{},
		&capturaLiquidacionHTTPPrueba{contexto: contextoLiquidacionHTTPPrueba(t, "a")}, &lectorLiquidacionHTTPPrueba{}
	constancia := domain.ConstanciaLiquidacionEconomica{Referencia: "liquidacion:uno", ClaveIdempotencia: "clave_economia_20261014", NumeroDocumento: "VEC-D-2026-000001", VersionEntrada: 2, VersionResultante: 3,
		TitularPersonaRef: "persona:titular", DecisorPersonaRef: "per_" + strings.Repeat("a", 24), UnidadRef: "unidad:rrhh", Fecha: "2026-10-14T10:00:00Z",
		ReciboRef: "recibo:uno", AuditoriaRef: "auditoria:uno", EventoRef: "evento:uno"}
	var err error
	r.resultado, err = domain.NuevaLiquidacionEconomica(p.preparar, constancia)
	if err != nil {
		t.Fatal(err)
	}
	l.resultado.Liquidacion = r.resultado.Instantanea()
	m, err := NuevoManejadorLiquidacionEconomica(p, i, r, c, l)
	if err != nil {
		t.Fatal(err)
	}
	return m, p, i, r, c, l
}

func TestLiquidacionHTTPPreparaRegistraYLeyendoRecupera(t *testing.T) {
	m, p, i, r, c, l := manejadorLiquidacionHTTPPrueba(t)
	base := RutaCircuito + "/" + referenciaLiquidacionPrueba + "/liquidaciones"
	w := peticionCircuito(m, http.MethodGet, base+"/propuesta?version_esperada=2", "")
	if w.Code != http.StatusOK || p.llamadas != 1 || p.revisar != nil || !strings.Contains(w.Body.String(), `"propuesta"`) {
		t.Fatalf("propuesta: %d %s", w.Code, w.Body.String())
	}
	cuerpo := `{"version_esperada":2,"clave_idempotencia":"clave_economia_20261014","revisiones":[{"indice":0,"regla_ref":"regla:manutencion:dos","reconocido_propuesto_centimos":400,"motivo_codigo":"revision_importe"}]}`
	w = peticionCircuito(m, http.MethodPost, base, cuerpo)
	if w.Code != http.StatusOK || p.llamadas != 2 || i.llamadas != 1 || r.llamadas != 1 || c.llamadas != 0 || l.llamadas != 0 ||
		i.s.Decision.UnidadRef != "" || r.s.Decision.UnidadRef != "unidad:rrhh" || r.s.Preparacion != p.preparar || !strings.Contains(w.Body.String(), `"recibo_ref":"recibo:uno"`) {
		t.Fatalf("registro: %d %s", w.Code, w.Body.String())
	}
	w = peticionCircuito(m, http.MethodGet, RutaBorradores+"/"+referenciaLiquidacionPrueba+"/liquidaciones/3", "")
	if w.Code != http.StatusOK || c.llamadas != 1 || l.llamadas != 1 || l.consulta.Version != 3 || l.consulta.Referencia != referenciaLiquidacionPrueba {
		t.Fatalf("lectura historica: %d %s", w.Code, w.Body.String())
	}
	w = peticionCircuito(m, http.MethodGet, base+"/actual", "")
	if w.Code != http.StatusOK || l.consulta.Version != 0 || l.llamadas != 2 {
		t.Fatalf("lectura actual: %d %s", w.Code, w.Body.String())
	}
}

func TestLiquidacionHTTPRecuperaCommitInciertoSoloMismaOperacion(t *testing.T) {
	m, p, i, r, c, l := manejadorLiquidacionHTTPPrueba(t)
	base := RutaCircuito + "/" + referenciaLiquidacionPrueba + "/liquidaciones"
	cuerpo := `{"version_esperada":2,"clave_idempotencia":"clave_economia_20261014","revisiones":[{"indice":0,"regla_ref":"regla:manutencion:dos","reconocido_propuesto_centimos":400,"motivo_codigo":"revision_importe"}]}`
	primero := peticionCircuito(m, http.MethodPost, base, cuerpo)
	if primero.Code != http.StatusOK || r.llamadas != 1 || c.llamadas != 0 || l.llamadas != 0 {
		t.Fatalf("primer acto: %d %s", primero.Code, primero.Body.String())
	}
	p.err = ports.ErrEstadoCircuitoConflicto // La primera operación avanzó la cola.
	segundo := peticionCircuito(m, http.MethodPost, base, cuerpo)
	if segundo.Code != http.StatusOK || segundo.Body.String() != primero.Body.String() ||
		r.llamadas != 1 || i.llamadas != 1 || c.llamadas != 1 || l.llamadas != 1 ||
		l.consulta.Referencia != referenciaLiquidacionPrueba || l.consulta.Version != 3 {
		t.Fatalf("repetición exacta: %d %s", segundo.Code, segundo.Body.String())
	}
	for _, caso := range []struct {
		nombre string
		cuerpo string
	}{
		{"otra clave", strings.Replace(cuerpo, "clave_economia_20261014", "otra_clave_economia_20261014", 1)},
		{"otro importe", strings.Replace(cuerpo, `"reconocido_propuesto_centimos":400`, `"reconocido_propuesto_centimos":399`, 1)},
		{"otro motivo", strings.Replace(cuerpo, `"motivo_codigo":"revision_importe"`, `"motivo_codigo":"revision_distinta"`, 1)},
		{"otra regla", strings.Replace(cuerpo, `"regla_ref":"regla:manutencion:dos"`, `"regla_ref":"regla:distinta"`, 1)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := peticionCircuito(m, http.MethodPost, base, caso.cuerpo)
			if w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "recibo:uno") || r.llamadas != 1 || i.llamadas != 1 {
				t.Fatalf("replay distinto: %d %s", w.Code, w.Body.String())
			}
		})
	}
	c.contexto = contextoLiquidacionHTTPPrueba(t, "b")
	if w := peticionCircuito(m, http.MethodPost, base, cuerpo); w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "recibo:uno") {
		t.Fatalf("otra persona: %d %s", w.Code, w.Body.String())
	}
	c.contexto = contextoLiquidacionHTTPPrueba(t, "a")
	c.err = ports.ErrAccesoCircuitoDenegado
	if w := peticionCircuito(m, http.MethodPost, base, cuerpo); w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "recibo:uno") || l.llamadas != 6 {
		t.Fatalf("lectura denegada: %d %s", w.Code, w.Body.String())
	}
	c.err = nil
	l.err = ports.ErrAccesoCircuitoDenegado
	if w := peticionCircuito(m, http.MethodPost, base, cuerpo); w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "recibo:uno") {
		t.Fatalf("SQL denegó lectura: %d %s", w.Code, w.Body.String())
	}
	l.err = nil
	l.resultado.Liquidacion.SnapshotSHA256 = strings.Repeat("0", 64)
	if w := peticionCircuito(m, http.MethodPost, base, cuerpo); w.Code != http.StatusConflict || strings.Contains(w.Body.String(), "recibo:uno") {
		t.Fatalf("instantánea alterada: %d %s", w.Code, w.Body.String())
	}
	if !errors.Is(p.err, ports.ErrEstadoCircuitoConflicto) {
		t.Fatal("se perdió la causa original")
	}
}

func TestLiquidacionHTTPComparaCadaRevisionPorIndice(t *testing.T) {
	lineas := []domain.LineaLiquidacionPropuesta{
		{Indice: 0, ReglaRef: "regla:uno", ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_uno"},
		{Indice: 1, ReglaRef: "regla:dos", ReconocidoPropuestoCentimos: 300, MotivoCodigo: "revision_dos"},
	}
	revisiones := []domain.RevisionLineaLiquidacion{
		{Indice: 1, ReglaRef: "regla:dos", ReconocidoPropuestoCentimos: 300, MotivoCodigo: "revision_dos"},
		{Indice: 0, ReglaRef: "regla:uno", ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_uno"},
	}
	if !mismasRevisionesLiquidacion(lineas, revisiones) {
		t.Fatal("mismo contenido en orden distinto")
	}
	revisiones[1].Indice = 1
	if mismasRevisionesLiquidacion(lineas, revisiones) {
		t.Fatal("índice repetido admitido")
	}
	revisiones[1].Indice = 0
	revisiones[1].MotivoCodigo = "revision_distinta"
	if mismasRevisionesLiquidacion(lineas, revisiones) {
		t.Fatal("segunda revisión distinta admitida")
	}
}

func TestLiquidacionHTTPRechazaEntradaSinAutoridad(t *testing.T) {
	m, p, i, r, c, l := manejadorLiquidacionHTTPPrueba(t)
	base := RutaCircuito + "/" + referenciaLiquidacionPrueba + "/liquidaciones"
	valida := `{"version_esperada":2,"clave_idempotencia":"clave_economia_20261014","revisiones":[{"indice":0,"regla_ref":"regla:manutencion:dos","reconocido_propuesto_centimos":400,"motivo_codigo":"revision_importe"}]}`
	for _, cuerpo := range []string{
		strings.Replace(valida, `"version_esperada":2`, `"version_esperada":2,"actor_ref":"actor:libre"`, 1),
		strings.Replace(valida, `"version_esperada":2`, `"version_esperada":2,"unidad_ref":"unidad:libre"`, 1),
		strings.Replace(valida, `"indice":0`, `"indice":0,"catalogo_ref":"catalogo:libre"`, 1),
		valida + ` {}`,
		`{"version_esperada":2,"clave_idempotencia":"corta","revisiones":[]}`,
	} {
		if w := peticionCircuito(m, http.MethodPost, base, cuerpo); w.Code != http.StatusBadRequest {
			t.Fatalf("cuerpo aceptado: %d %s", w.Code, w.Body.String())
		}
	}
	for _, ruta := range []string{base + "/propuesta?version_esperada=2&version_esperada=3", base + "/propuesta?version_esperada=02", base + "/propuesta?version_esperada=2&unidad_ref=libre"} {
		if w := peticionCircuito(m, http.MethodGet, ruta, ""); w.Code != http.StatusBadRequest {
			t.Fatalf("consulta aceptada: %s %d", ruta, w.Code)
		}
	}
	for _, ruta := range []string{base + "/01", base + "/propuesta/extra", RutaBorradores + "/%64co_aaaaaaaaaaaaaaaaaaaaaa/liquidaciones/3"} {
		if w := peticionCircuito(m, http.MethodGet, ruta, ""); w.Code != http.StatusNotFound {
			t.Fatalf("ruta aceptada: %s %d", ruta, w.Code)
		}
	}
	if w := peticionCircuito(m, http.MethodDelete, base, ""); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("metodo: %d", w.Code)
	}
	for _, cabecera := range []string{"Cookie", "Authorization", "X-Vec-Actor", "X-Vec-Persona", "X-Vec-Perfil"} {
		peticion := httptest.NewRequest(http.MethodGet, base+"/actual", nil)
		peticion.Header.Set("Accept", "application/json")
		peticion.Header.Set(cabecera, "valor_no_fiable")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, peticion)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("cabecera %s: %d", cabecera, w.Code)
		}
	}
	if p.llamadas != 0 || i.llamadas != 0 || r.llamadas != 0 || c.llamadas != 0 || l.llamadas != 0 {
		t.Fatal("se invoco dependencia con entrada rechazada")
	}
}

func TestLiquidacionHTTPSinSesionNoPreparaNiLee(t *testing.T) {
	m, p, i, r, c, l := manejadorLiquidacionHTTPPrueba(t)
	p.err = ports.ErrAccesoCircuitoDenegado
	c.err = ports.ErrAccesoCircuitoDenegado
	base := RutaCircuito + "/" + referenciaLiquidacionPrueba + "/liquidaciones"
	w := peticionCircuito(m, http.MethodGet, base+"/propuesta?version_esperada=2", "")
	if w.Code != http.StatusForbidden || i.llamadas != 0 || r.llamadas != 0 {
		t.Fatalf("propuesta sin permiso: %d", w.Code)
	}
	cuerpo := `{"version_esperada":2,"clave_idempotencia":"clave_economia_20261014","revisiones":[{"indice":0,"regla_ref":"regla:manutencion:dos","reconocido_propuesto_centimos":400,"motivo_codigo":"revision_importe"}]}`
	w = peticionCircuito(m, http.MethodPost, base, cuerpo)
	if w.Code != http.StatusForbidden || i.llamadas != 0 || r.llamadas != 0 {
		t.Fatalf("registro sin permiso: %d", w.Code)
	}
	c.err = ports.ErrAccesoCircuitoDenegado
	w = peticionCircuito(m, http.MethodGet, base+"/actual", "")
	if w.Code != http.StatusForbidden || l.llamadas != 0 {
		t.Fatalf("lectura sin permiso: %d", w.Code)
	}
}

func TestLiquidacionHTTPErroresNominales(t *testing.T) {
	m, p, i, _, _, _ := manejadorLiquidacionHTTPPrueba(t)
	base := RutaCircuito + "/" + referenciaLiquidacionPrueba + "/liquidaciones"
	casos := []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrCompetenciaCircuitoSinFuente, http.StatusForbidden, "competencia_sin_fuente"},
		{ports.ErrConflictoIdempotencia, http.StatusConflict, "conflicto_idempotencia"},
		{ports.ErrVersionComisionConflicto, http.StatusConflict, "conflicto_version"},
	}
	for _, caso := range casos {
		p.err = nil
		i.err = caso.err
		cuerpo := `{"version_esperada":2,"clave_idempotencia":"clave_economia_20261014","revisiones":[{"indice":0,"regla_ref":"regla:manutencion:dos","reconocido_propuesto_centimos":400,"motivo_codigo":"revision_importe"}]}`
		w := peticionCircuito(m, http.MethodPost, base, cuerpo)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), "dietas.error."+caso.codigo) {
			t.Fatalf("error %v: %d %s", caso.err, w.Code, w.Body.String())
		}
	}
}

func TestLiquidacionHTTPRespuestaSoloInstantanea(t *testing.T) {
	m, _, _, _, _, _ := manejadorLiquidacionHTTPPrueba(t)
	w := peticionCircuito(m, http.MethodGet, RutaCircuito+"/"+referenciaLiquidacionPrueba+"/liquidaciones/actual", "")
	var cuerpo map[string]json.RawMessage
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || len(cuerpo) != 1 || len(cuerpo["liquidacion"]) == 0 {
		t.Fatalf("salida: %d %s", w.Code, w.Body.String())
	}
}
