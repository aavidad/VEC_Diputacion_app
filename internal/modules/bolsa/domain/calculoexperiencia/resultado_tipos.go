package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	esquemaResultadoExperienciaV1 = "vec.bolsa.resultado_experiencia.v1"
	esquemaPlanResultadoV1        = "vec.bolsa.plan_experiencia.v1"
	contratoMotorResultadoV1      = "vec.bolsa.motor_experiencia.v1"
	versionMotorResultadoV1       = uint64(1)

	maximoAplicacionesResultadoV1    = 100_000
	maximoDescartesResultadoV1       = 100_000
	maximoSinCoincidenciaResultadoV1 = maximoTramosEntrada
	maximoBloqueosResultadoV1        = 100_000
	maximoReglasResultadoV1          = 1_024
	maximoSeccionesResultadoV1       = 64
	maximoReferenciasBloqueoV1       = 1_024
	maximoBytesResultadoV1           = 64 * 1024 * 1024
)

type EstadoResultadoExperienciaV1 string

const (
	ResultadoExperienciaCompletado EstadoResultadoExperienciaV1 = "completado"
	ResultadoExperienciaBloqueado  EstadoResultadoExperienciaV1 = "bloqueado"
)

type FaseResultadoExperienciaV1 string

const (
	FaseResultadoSeleccion  FaseResultadoExperienciaV1 = "seleccion"
	FaseResultadoIntervalos FaseResultadoExperienciaV1 = "intervalos"
	FaseResultadoPuntuacion FaseResultadoExperienciaV1 = "puntuacion"
	FaseResultadoCompletado FaseResultadoExperienciaV1 = "completado"
)

type CodigoRazonResultadoExperienciaV1 string

const (
	RazonCoincidenciaUnica    CodigoRazonResultadoExperienciaV1 = "coincidencia_unica"
	RazonPrioridad            CodigoRazonResultadoExperienciaV1 = "prioridad"
	RazonAcumulacion          CodigoRazonResultadoExperienciaV1 = "acumulacion"
	RazonPrioridadInferior    CodigoRazonResultadoExperienciaV1 = "prioridad_inferior"
	RazonNingunaCoincidencia  CodigoRazonResultadoExperienciaV1 = "ninguna_regla_coincidente"
	RazonPosteriorCorte       CodigoRazonResultadoExperienciaV1 = "posterior_corte"
	RazonIntervaloVacio       CodigoRazonResultadoExperienciaV1 = "intervalo_vacio"
	RazonJornadaProporcional  CodigoRazonResultadoExperienciaV1 = "jornada_proporcional"
	RazonJornadaIntegra       CodigoRazonResultadoExperienciaV1 = "jornada_integra"
	RazonUmbralAlcanzado      CodigoRazonResultadoExperienciaV1 = "umbral_alcanzado"
	RazonUmbralNoAlcanzado    CodigoRazonResultadoExperienciaV1 = "umbral_no_alcanzado"
	RazonProteccionAtestada   CodigoRazonResultadoExperienciaV1 = "proteccion_atestada"
	RazonProteccionNoAtestada CodigoRazonResultadoExperienciaV1 = "proteccion_no_atestada"
)

type CodigoBloqueoResultadoExperienciaV1 string

const (
	BloqueoResultadoCatalogoIncompatible  CodigoBloqueoResultadoExperienciaV1 = "catalogo_incompatible"
	BloqueoResultadoGruposDistintos       CodigoBloqueoResultadoExperienciaV1 = "reglas_en_grupos_distintos"
	BloqueoResultadoCoincidenciaRechazada CodigoBloqueoResultadoExperienciaV1 = "coincidencia_reglas_rechazada"
	BloqueoResultadoSolape                CodigoBloqueoResultadoExperienciaV1 = "solape_tramos"
	BloqueoResultadoRedondeoNoExacto      CodigoBloqueoResultadoExperienciaV1 = "redondeo_no_exacto"
)

type FronteraRestosResultadoExperienciaV1 string

const (
	FronteraRestosResultadoExacta  FronteraRestosResultadoExperienciaV1 = "exacta"
	FronteraRestosResultadoPeriodo FronteraRestosResultadoExperienciaV1 = "periodo"
	FronteraRestosResultadoRegla   FronteraRestosResultadoExperienciaV1 = "regla"
)

type VinculoMotorResultadoExperienciaV1 struct {
	contrato             string
	version              uint64
	huellaContratoSHA256 string
}

func (v VinculoMotorResultadoExperienciaV1) Contrato() string { return v.contrato }
func (v VinculoMotorResultadoExperienciaV1) Version() uint64  { return v.version }
func (v VinculoMotorResultadoExperienciaV1) HuellaContratoSHA256() string {
	return v.huellaContratoSHA256
}

type VinculoPlanResultadoExperienciaV1 struct {
	esquema      string
	huellaSHA256 string
}

func (v VinculoPlanResultadoExperienciaV1) Esquema() string      { return v.esquema }
func (v VinculoPlanResultadoExperienciaV1) HuellaSHA256() string { return v.huellaSHA256 }

type VinculoEntradaResultadoExperienciaV1 struct {
	instantanea           reglasbaremo.ReferenciaVersionada
	huellaContenidoSHA256 string
}

func (v VinculoEntradaResultadoExperienciaV1) Instantanea() reglasbaremo.ReferenciaVersionada {
	return v.instantanea
}
func (v VinculoEntradaResultadoExperienciaV1) HuellaContenidoSHA256() string {
	return v.huellaContenidoSHA256
}

type VinculosResultadoExperienciaV1 struct {
	motor      VinculoMotorResultadoExperienciaV1
	plan       VinculoPlanResultadoExperienciaV1
	conjunto   reglasbaremo.ReferenciaVersionada
	entrada    VinculoEntradaResultadoExperienciaV1
	fechaCorte baremacion.FechaCivil
}

func (v VinculosResultadoExperienciaV1) Motor() VinculoMotorResultadoExperienciaV1 { return v.motor }
func (v VinculosResultadoExperienciaV1) Plan() VinculoPlanResultadoExperienciaV1   { return v.plan }
func (v VinculosResultadoExperienciaV1) Conjunto() reglasbaremo.ReferenciaVersionada {
	return v.conjunto
}
func (v VinculosResultadoExperienciaV1) Entrada() VinculoEntradaResultadoExperienciaV1 {
	return v.entrada
}
func (v VinculosResultadoExperienciaV1) FechaCorte() baremacion.FechaCivil { return v.fechaCorte }

// AplicacionSeleccionResultadoExperienciaV1 afirma que todos los criterios
// gobernados de la regla ligada por el plan resultaron verdaderos. No repite
// claves ni valores catalogados: el plan exacto conserva las primeras y la
// huella de entrada liga los segundos sin ampliar datos laborales en la salida.
type AplicacionSeleccionResultadoExperienciaV1 struct {
	tramo        reglasbaremo.ReferenciaVersionada
	reglaClave   string
	grupoClave   string
	seccionClave string
	prioridad    uint32
	razon        CodigoRazonResultadoExperienciaV1
}

func (a AplicacionSeleccionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada {
	return a.tramo
}
func (a AplicacionSeleccionResultadoExperienciaV1) ReglaClave() string   { return a.reglaClave }
func (a AplicacionSeleccionResultadoExperienciaV1) GrupoClave() string   { return a.grupoClave }
func (a AplicacionSeleccionResultadoExperienciaV1) SeccionClave() string { return a.seccionClave }
func (a AplicacionSeleccionResultadoExperienciaV1) Prioridad() uint32    { return a.prioridad }
func (a AplicacionSeleccionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1 {
	return a.razon
}

type DescarteSeleccionResultadoExperienciaV1 struct {
	tramo             reglasbaremo.ReferenciaVersionada
	reglaClave        string
	grupoClave        string
	reglaSeleccionada string
	razon             CodigoRazonResultadoExperienciaV1
}

func (d DescarteSeleccionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada {
	return d.tramo
}
func (d DescarteSeleccionResultadoExperienciaV1) ReglaClave() string { return d.reglaClave }
func (d DescarteSeleccionResultadoExperienciaV1) GrupoClave() string { return d.grupoClave }
func (d DescarteSeleccionResultadoExperienciaV1) ReglaSeleccionada() string {
	return d.reglaSeleccionada
}
func (d DescarteSeleccionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1 {
	return d.razon
}

type SinCoincidenciaResultadoExperienciaV1 struct {
	tramo reglasbaremo.ReferenciaVersionada
	razon CodigoRazonResultadoExperienciaV1
}

func (s SinCoincidenciaResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada {
	return s.tramo
}
func (s SinCoincidenciaResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1 {
	return s.razon
}

type SeleccionResultadoExperienciaV1 struct {
	aplicaciones    []AplicacionSeleccionResultadoExperienciaV1
	descartes       []DescarteSeleccionResultadoExperienciaV1
	sinCoincidencia []SinCoincidenciaResultadoExperienciaV1
	evaluaciones    uint64
}

func (s SeleccionResultadoExperienciaV1) Aplicaciones() []AplicacionSeleccionResultadoExperienciaV1 {
	return append([]AplicacionSeleccionResultadoExperienciaV1(nil), s.aplicaciones...)
}
func (s SeleccionResultadoExperienciaV1) Descartes() []DescarteSeleccionResultadoExperienciaV1 {
	return append([]DescarteSeleccionResultadoExperienciaV1(nil), s.descartes...)
}
func (s SeleccionResultadoExperienciaV1) SinCoincidencia() []SinCoincidenciaResultadoExperienciaV1 {
	return append([]SinCoincidenciaResultadoExperienciaV1(nil), s.sinCoincidencia...)
}
func (s SeleccionResultadoExperienciaV1) Evaluaciones() uint64 { return s.evaluaciones }

type IntervaloAplicacionResultadoExperienciaV1 struct {
	tramo         reglasbaremo.ReferenciaVersionada
	reglaClave    string
	periodo       PeriodoServicio
	extremo       reglasbaremo.TratamientoExtremoFinal
	efectivo      baremacion.IntervaloCivil
	tieneEfectivo bool
	dias          uint64
	razon         CodigoRazonResultadoExperienciaV1
}

func (i IntervaloAplicacionResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada {
	return i.tramo
}
func (i IntervaloAplicacionResultadoExperienciaV1) ReglaClave() string       { return i.reglaClave }
func (i IntervaloAplicacionResultadoExperienciaV1) Periodo() PeriodoServicio { return i.periodo }
func (i IntervaloAplicacionResultadoExperienciaV1) Extremo() reglasbaremo.TratamientoExtremoFinal {
	return i.extremo
}
func (i IntervaloAplicacionResultadoExperienciaV1) Efectivo() (baremacion.IntervaloCivil, bool) {
	return i.efectivo, i.tieneEfectivo
}
func (i IntervaloAplicacionResultadoExperienciaV1) Dias() uint64 { return i.dias }
func (i IntervaloAplicacionResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1 {
	return i.razon
}

type JornadaResultadoExperienciaV1 struct {
	origen             baremacion.FraccionJornada
	modo               reglasbaremo.ModoJornada
	factor             exactoResultadoV1
	atestacionPresente bool
	atestacionUsada    bool
	razon              CodigoRazonResultadoExperienciaV1
}

func (j JornadaResultadoExperienciaV1) Origen() baremacion.FraccionJornada       { return j.origen }
func (j JornadaResultadoExperienciaV1) Modo() reglasbaremo.ModoJornada           { return j.modo }
func (j JornadaResultadoExperienciaV1) FactorExacto() string                     { return j.factor.texto() }
func (j JornadaResultadoExperienciaV1) AtestacionPresente() bool                 { return j.atestacionPresente }
func (j JornadaResultadoExperienciaV1) AtestacionUsada() bool                    { return j.atestacionUsada }
func (j JornadaResultadoExperienciaV1) Razon() CodigoRazonResultadoExperienciaV1 { return j.razon }

type UnidadesAplicacionResultadoExperienciaV1 struct {
	exactas   exactoResultadoV1
	aportadas exactoResultadoV1
	resto     exactoResultadoV1
	frontera  FronteraRestosResultadoExperienciaV1
}

func (u UnidadesAplicacionResultadoExperienciaV1) Exactas() string   { return u.exactas.texto() }
func (u UnidadesAplicacionResultadoExperienciaV1) Aportadas() string { return u.aportadas.texto() }
func (u UnidadesAplicacionResultadoExperienciaV1) Resto() string     { return u.resto.texto() }
func (u UnidadesAplicacionResultadoExperienciaV1) Frontera() FronteraRestosResultadoExperienciaV1 {
	return u.frontera
}

type PuntuacionPeriodoResultadoExperienciaV1 struct {
	bruto           exactoResultadoV1
	redondeado      exactoResultadoV1
	tieneRedondeado bool
}

func (p PuntuacionPeriodoResultadoExperienciaV1) BrutoExacto() string { return p.bruto.texto() }
func (p PuntuacionPeriodoResultadoExperienciaV1) RedondeadoExacto() (string, bool) {
	return p.redondeado.texto(), p.tieneRedondeado
}

type AplicacionCalculadaResultadoExperienciaV1 struct {
	tramo      reglasbaremo.ReferenciaVersionada
	reglaClave string
	jornada    JornadaResultadoExperienciaV1
	unidades   UnidadesAplicacionResultadoExperienciaV1
	puntuacion PuntuacionPeriodoResultadoExperienciaV1
}

func (a AplicacionCalculadaResultadoExperienciaV1) Tramo() reglasbaremo.ReferenciaVersionada {
	return a.tramo
}
func (a AplicacionCalculadaResultadoExperienciaV1) ReglaClave() string { return a.reglaClave }
func (a AplicacionCalculadaResultadoExperienciaV1) Jornada() JornadaResultadoExperienciaV1 {
	return a.jornada
}
func (a AplicacionCalculadaResultadoExperienciaV1) Unidades() UnidadesAplicacionResultadoExperienciaV1 {
	return a.unidades
}
func (a AplicacionCalculadaResultadoExperienciaV1) Puntuacion() PuntuacionPeriodoResultadoExperienciaV1 {
	return a.puntuacion
}

type TopeResultadoExperienciaV1 struct {
	antes    exactoResultadoV1
	limitado bool
	limite   exactoResultadoV1
	despues  exactoResultadoV1
	aplicado bool
}

func (t TopeResultadoExperienciaV1) Antes() string          { return t.antes.texto() }
func (t TopeResultadoExperienciaV1) Limite() (string, bool) { return t.limite.texto(), t.limitado }
func (t TopeResultadoExperienciaV1) Despues() string        { return t.despues.texto() }
func (t TopeResultadoExperienciaV1) Aplicado() bool         { return t.aplicado }

type RedondeoResultadoExperienciaV1 struct {
	momento reglasbaremo.MomentoRedondeo
	modo    baremacion.ModoRedondeo
	entrada exactoResultadoV1
	salida  exactoResultadoV1
}

func (r RedondeoResultadoExperienciaV1) Momento() reglasbaremo.MomentoRedondeo { return r.momento }
func (r RedondeoResultadoExperienciaV1) Modo() baremacion.ModoRedondeo         { return r.modo }
func (r RedondeoResultadoExperienciaV1) EntradaExacta() string                 { return r.entrada.texto() }
func (r RedondeoResultadoExperienciaV1) SalidaExacta() string                  { return r.salida.texto() }

type ResultadoReglaExperienciaV1 struct {
	seccionClave       string
	reglaClave         string
	unidadesAgregadas  exactoResultadoV1
	unidadesTrasRestos exactoResultadoV1
	restoRegla         exactoResultadoV1
	topeUnidades       TopeResultadoExperienciaV1
	coeficiente        baremacion.Puntos
	bruto              exactoResultadoV1
	redondeo           RedondeoResultadoExperienciaV1
	topePuntos         TopeResultadoExperienciaV1
	puntosFinales      exactoResultadoV1
}

func (r ResultadoReglaExperienciaV1) SeccionClave() string                     { return r.seccionClave }
func (r ResultadoReglaExperienciaV1) ReglaClave() string                       { return r.reglaClave }
func (r ResultadoReglaExperienciaV1) UnidadesAgregadas() string                { return r.unidadesAgregadas.texto() }
func (r ResultadoReglaExperienciaV1) UnidadesTrasRestos() string               { return r.unidadesTrasRestos.texto() }
func (r ResultadoReglaExperienciaV1) RestoRegla() string                       { return r.restoRegla.texto() }
func (r ResultadoReglaExperienciaV1) TopeUnidades() TopeResultadoExperienciaV1 { return r.topeUnidades }
func (r ResultadoReglaExperienciaV1) Coeficiente() baremacion.Puntos           { return r.coeficiente }
func (r ResultadoReglaExperienciaV1) BrutoExacto() string                      { return r.bruto.texto() }
func (r ResultadoReglaExperienciaV1) Redondeo() RedondeoResultadoExperienciaV1 { return r.redondeo }
func (r ResultadoReglaExperienciaV1) TopePuntos() TopeResultadoExperienciaV1   { return r.topePuntos }
func (r ResultadoReglaExperienciaV1) PuntosFinalesExactos() string             { return r.puntosFinales.texto() }

type SubtotalSeccionResultadoExperienciaV1 struct {
	seccionClave  string
	antesTope     exactoResultadoV1
	tope          TopeResultadoExperienciaV1
	puntosFinales baremacion.Puntos
}

func (s SubtotalSeccionResultadoExperienciaV1) SeccionClave() string             { return s.seccionClave }
func (s SubtotalSeccionResultadoExperienciaV1) AntesTopeExacto() string          { return s.antesTope.texto() }
func (s SubtotalSeccionResultadoExperienciaV1) Tope() TopeResultadoExperienciaV1 { return s.tope }
func (s SubtotalSeccionResultadoExperienciaV1) PuntosFinales() baremacion.Puntos {
	return s.puntosFinales
}

type BloqueoResultadoExperienciaV1 struct {
	codigo           CodigoBloqueoResultadoExperienciaV1
	tramos           []reglasbaremo.ReferenciaVersionada
	reglas           []string
	grupoClave       string
	seccionClave     string
	claveGobernada   string
	valorExacto      exactoResultadoV1
	tieneValorExacto bool
}

func (b BloqueoResultadoExperienciaV1) Codigo() CodigoBloqueoResultadoExperienciaV1 { return b.codigo }
func (b BloqueoResultadoExperienciaV1) Tramos() []reglasbaremo.ReferenciaVersionada {
	return append([]reglasbaremo.ReferenciaVersionada(nil), b.tramos...)
}
func (b BloqueoResultadoExperienciaV1) Reglas() []string       { return append([]string(nil), b.reglas...) }
func (b BloqueoResultadoExperienciaV1) GrupoClave() string     { return b.grupoClave }
func (b BloqueoResultadoExperienciaV1) SeccionClave() string   { return b.seccionClave }
func (b BloqueoResultadoExperienciaV1) ClaveGobernada() string { return b.claveGobernada }
func (b BloqueoResultadoExperienciaV1) ValorExacto() (string, bool) {
	return b.valorExacto.texto(), b.tieneValorExacto
}
