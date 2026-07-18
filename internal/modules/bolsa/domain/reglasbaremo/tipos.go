package reglasbaremo

import (
	"crypto/subtle"
	"sort"

	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	maximoVersion            uint64 = 1_000_000_000
	maximoOrden              uint32 = 1_000_000
	maximoCriteriosPorRegla         = 32
	maximoValoresPorCriterio        = 256
)

// ReferenciaVersionada fija una dependencia por referencia opaca, version y
// huella. Nunca significa "la version vigente".
type ReferenciaVersionada struct {
	referencia   string
	version      uint64
	huellaSHA256 string
}

// NuevaReferenciaVersionada construye una referencia cerrada y reproducible.
func NuevaReferenciaVersionada(referencia string, version uint64, huellaSHA256 string) (ReferenciaVersionada, error) {
	resultado := ReferenciaVersionada{
		referencia:   referencia,
		version:      version,
		huellaSHA256: huellaSHA256,
	}
	if err := resultado.validar("referencia_versionada"); err != nil {
		return ReferenciaVersionada{}, err
	}
	return resultado, nil
}

// Referencia devuelve el identificador opaco exacto.
func (r ReferenciaVersionada) Referencia() string { return r.referencia }

// Version devuelve la version positiva fijada.
func (r ReferenciaVersionada) Version() uint64 { return r.version }

// HuellaSHA256 devuelve la huella hexadecimal canonica.
func (r ReferenciaVersionada) HuellaSHA256() string { return r.huellaSHA256 }

// Validar comprueba que la referencia conserva su forma canonica cerrada.
func (r ReferenciaVersionada) Validar() error {
	return r.validar("referencia_versionada")
}

// CoincideExactamenteCon compara en tiempo constante dos referencias validas,
// sin normalizar ni omitir campos. Un valor cero o invalido nunca coincide.
func (r ReferenciaVersionada) CoincideExactamenteCon(otra ReferenciaVersionada) bool {
	if r.Validar() != nil || otra.Validar() != nil {
		return false
	}
	return r.coincidenciaExactaConstante(otra) == 1
}

func (r ReferenciaVersionada) coincidenciaExactaConstante(otra ReferenciaVersionada) int {
	coincide := textoReglasBaremoIgualConstante(r.referencia, otra.referencia)
	coincide &= numeroReglasBaremoIgualConstante(r.version, otra.version)
	coincide &= textoReglasBaremoIgualConstante(r.huellaSHA256, otra.huellaSHA256)
	return coincide
}

func textoReglasBaremoIgualConstante(izquierda, derecha string) int {
	// Las entradas validas ya respetan el limite. El relleno impide que una
	// diferencia de longitud acorte la comparacion de referencias HMAC.
	if len(izquierda) > maximoCaracteresReferencia || len(derecha) > maximoCaracteresReferencia {
		return 0
	}
	var canonicaIzquierda, canonicaDerecha [maximoCaracteresReferencia]byte
	copy(canonicaIzquierda[:], izquierda)
	copy(canonicaDerecha[:], derecha)
	coincide := subtle.ConstantTimeEq(int32(len(izquierda)), int32(len(derecha)))
	coincide &= subtle.ConstantTimeCompare(canonicaIzquierda[:], canonicaDerecha[:])
	return coincide
}

func numeroReglasBaremoIgualConstante(izquierdo, derecho uint64) int {
	// Versiones y revisiones validas no superan maximoVersion (< MaxInt32).
	return subtle.ConstantTimeEq(int32(izquierdo), int32(derecho))
}

func (r ReferenciaVersionada) validar(campo string) error {
	if !referenciaValida(r.referencia) {
		return nuevoError(campo+".referencia", CodigoValorNoCanonico)
	}
	if r.version == 0 || r.version > maximoVersion {
		return nuevoError(campo+".version", CodigoFueraDeLimites)
	}
	if !huellaSHA256Valida(r.huellaSHA256) {
		return nuevoError(campo+".huella_sha256", CodigoValorNoCanonico)
	}
	return nil
}

// IdentidadConjuntoReglasBaremo enlaza el conjunto con una convocatoria y un
// expediente concretos mediante tokens opacos de 128 bits. Etiquetas, codigos
// oficiales y datos personales permanecen fuera de esta identidad interna.
type IdentidadConjuntoReglasBaremo struct {
	referencia      string
	version         uint64
	convocatoriaRef string
	expedienteRef   string
}

// NuevaIdentidadConjuntoReglasBaremo valida la identidad sin normalizar
// silenciosamente ninguna referencia.
func NuevaIdentidadConjuntoReglasBaremo(
	referencia string,
	version uint64,
	convocatoriaRef string,
	expedienteRef string,
) (IdentidadConjuntoReglasBaremo, error) {
	identidad := IdentidadConjuntoReglasBaremo{
		referencia:      referencia,
		version:         version,
		convocatoriaRef: convocatoriaRef,
		expedienteRef:   expedienteRef,
	}
	if err := identidad.validar(); err != nil {
		return IdentidadConjuntoReglasBaremo{}, err
	}
	return identidad, nil
}

// Referencia devuelve la referencia inmutable del conjunto.
func (i IdentidadConjuntoReglasBaremo) Referencia() string { return i.referencia }

// Version devuelve la version semantica del conjunto.
func (i IdentidadConjuntoReglasBaremo) Version() uint64 { return i.version }

// ConvocatoriaRef devuelve la convocatoria a la que pertenece.
func (i IdentidadConjuntoReglasBaremo) ConvocatoriaRef() string { return i.convocatoriaRef }

// ExpedienteRef devuelve el expediente administrativo enlazado.
func (i IdentidadConjuntoReglasBaremo) ExpedienteRef() string { return i.expedienteRef }

func (i IdentidadConjuntoReglasBaremo) validar() error {
	if !referenciaIdentidadOpaca128Valida(i.referencia, "rgl_") {
		return nuevoError("identidad.referencia", CodigoValorNoCanonico)
	}
	if i.version == 0 || i.version > maximoVersion {
		return nuevoError("identidad.version", CodigoFueraDeLimites)
	}
	if !referenciaIdentidadOpaca128Valida(i.convocatoriaRef, "con_") {
		return nuevoError("identidad.convocatoria_ref", CodigoValorNoCanonico)
	}
	if !referenciaIdentidadOpaca128Valida(i.expedienteRef, "exp_") {
		return nuevoError("identidad.expediente_ref", CodigoValorNoCanonico)
	}
	if i.referencia == i.convocatoriaRef || i.referencia == i.expedienteRef ||
		i.convocatoriaRef == i.expedienteRef {
		return nuevoError("identidad.referencias", CodigoValorDuplicado)
	}
	return nil
}

func referenciaIdentidadOpaca128Valida(valor, prefijo string) bool {
	const longitudHex128 = 32
	if len(prefijo) == 0 || len(valor) != len(prefijo)+longitudHex128 ||
		valor[:len(prefijo)] != prefijo {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

// LimitePuntos distingue de forma expresa la ausencia de limite de un valor
// cero. Su valor cero es invalido y no selecciona una politica implicita.
type LimitePuntos struct {
	modo  modoLimite
	valor baremacion.Puntos
}

type modoLimite string

const (
	modoSinLimite modoLimite = "sin_limite"
	modoLimitado  modoLimite = "limitado"
)

// SinLimitePuntos declara explicitamente que la regla no tiene tope propio.
func SinLimitePuntos() LimitePuntos { return LimitePuntos{modo: modoSinLimite} }

// NuevoLimitePuntos declara un tope positivo exacto.
func NuevoLimitePuntos(valor baremacion.Puntos) (LimitePuntos, error) {
	limite := LimitePuntos{modo: modoLimitado, valor: valor}
	if err := limite.validar("limite_puntos"); err != nil {
		return LimitePuntos{}, err
	}
	return limite, nil
}

// EstaLimitado indica si existe un maximo propio.
func (l LimitePuntos) EstaLimitado() bool { return l.modo == modoLimitado }

// Valor devuelve el limite y si este fue configurado.
func (l LimitePuntos) Valor() (baremacion.Puntos, bool) {
	return l.valor, l.EstaLimitado()
}

func (l LimitePuntos) validar(campo string) error {
	switch l.modo {
	case modoSinLimite:
		if l.valor.Micropuntos() != 0 {
			return nuevoError(campo, CodigoPoliticaIncompleta)
		}
	case modoLimitado:
		if !l.valor.EsValido() || l.valor.Micropuntos() <= 0 {
			return nuevoError(campo, CodigoFueraDeLimites)
		}
	default:
		return nuevoError(campo, CodigoPoliticaIncompleta)
	}
	return nil
}

// LimiteUnidades distingue un limite racional positivo de la ausencia expresa
// de tope temporal.
type LimiteUnidades struct {
	modo  modoLimite
	valor baremacion.Racional
}

// SinLimiteUnidades declara explicitamente que no hay tope temporal propio.
func SinLimiteUnidades() LimiteUnidades { return LimiteUnidades{modo: modoSinLimite} }

// NuevoLimiteUnidades construye un tope racional positivo y exacto.
func NuevoLimiteUnidades(valor baremacion.Racional) (LimiteUnidades, error) {
	limite := LimiteUnidades{modo: modoLimitado, valor: valor}
	if err := limite.validar("limite_unidades"); err != nil {
		return LimiteUnidades{}, err
	}
	return limite, nil
}

// EstaLimitado indica si hay un maximo de unidades.
func (l LimiteUnidades) EstaLimitado() bool { return l.modo == modoLimitado }

// Valor devuelve el limite y si este fue configurado.
func (l LimiteUnidades) Valor() (baremacion.Racional, bool) {
	return l.valor, l.EstaLimitado()
}

func (l LimiteUnidades) validar(campo string) error {
	switch l.modo {
	case modoSinLimite:
		if l.valor.EsValido() {
			return nuevoError(campo, CodigoPoliticaIncompleta)
		}
	case modoLimitado:
		if !l.valor.EsValido() || l.valor.Numerador() <= 0 {
			return nuevoError(campo, CodigoFueraDeLimites)
		}
	default:
		return nuevoError(campo, CodigoPoliticaIncompleta)
	}
	return nil
}

// UnidadTemporal identifica la unidad de entrada o la unidad puntuable.
type UnidadTemporal string

const (
	UnidadTemporalDia  UnidadTemporal = "dia"
	UnidadTemporalMes  UnidadTemporal = "mes"
	UnidadTemporalAnio UnidadTemporal = "anio"
	UnidadTemporalHora UnidadTemporal = "hora"
)

func (u UnidadTemporal) valida() bool {
	switch u {
	case UnidadTemporalDia, UnidadTemporalMes, UnidadTemporalAnio, UnidadTemporalHora:
		return true
	default:
		return false
	}
}

// TratamientoExtremoFinal fija si el ultimo dia u hora se incorpora antes de
// convertir unidades.
type TratamientoExtremoFinal string

const (
	ExtremoFinalExclusivo TratamientoExtremoFinal = "exclusivo"
	ExtremoFinalInclusivo TratamientoExtremoFinal = "inclusivo"
)

func (t TratamientoExtremoFinal) valido() bool {
	return t == ExtremoFinalExclusivo || t == ExtremoFinalInclusivo
}

// PoliticaUnidadTemporal expresa una conversion exacta. Por ejemplo, una
// regla por meses convencionales puede fijar dia -> mes y 30/1 unidades base.
type PoliticaUnidadTemporal struct {
	unidadBase            UnidadTemporal
	unidadPuntuable       UnidadTemporal
	unidadesBasePorUnidad baremacion.Racional
	extremoFinal          TratamientoExtremoFinal
}

// NuevaPoliticaUnidadTemporal exige todos sus parametros; no presupone 30
// dias por mes, 365 por anio ni la inclusion del extremo final.
func NuevaPoliticaUnidadTemporal(
	unidadBase UnidadTemporal,
	unidadPuntuable UnidadTemporal,
	unidadesBasePorUnidad baremacion.Racional,
	extremoFinal TratamientoExtremoFinal,
) (PoliticaUnidadTemporal, error) {
	politica := PoliticaUnidadTemporal{
		unidadBase:            unidadBase,
		unidadPuntuable:       unidadPuntuable,
		unidadesBasePorUnidad: unidadesBasePorUnidad,
		extremoFinal:          extremoFinal,
	}
	if err := politica.validar(); err != nil {
		return PoliticaUnidadTemporal{}, err
	}
	return politica, nil
}

func (p PoliticaUnidadTemporal) UnidadBase() UnidadTemporal      { return p.unidadBase }
func (p PoliticaUnidadTemporal) UnidadPuntuable() UnidadTemporal { return p.unidadPuntuable }
func (p PoliticaUnidadTemporal) UnidadesBasePorUnidad() baremacion.Racional {
	return p.unidadesBasePorUnidad
}
func (p PoliticaUnidadTemporal) ExtremoFinal() TratamientoExtremoFinal { return p.extremoFinal }

func (p PoliticaUnidadTemporal) validar() error {
	if !p.unidadBase.valida() || !p.unidadPuntuable.valida() ||
		!p.unidadesBasePorUnidad.EsValido() || p.unidadesBasePorUnidad.Numerador() <= 0 ||
		!p.extremoFinal.valido() {
		return nuevoError("politica_unidad_temporal", CodigoPoliticaIncompleta)
	}
	uno, _ := baremacion.NuevoRacional(1, 1)
	if p.unidadBase == p.unidadPuntuable {
		comparacion, err := p.unidadesBasePorUnidad.Comparar(uno)
		if err != nil || comparacion != 0 {
			return nuevoError("politica_unidad_temporal.conversion", CodigoValorInvalido)
		}
	}
	return nil
}

// ModoJornada selecciona una semantica revisada por el dominio.
type ModoJornada string

const (
	JornadaProporcional       ModoJornada = "proporcional"
	JornadaIntegra            ModoJornada = "integra"
	JornadaIntegraDesdeUmbral ModoJornada = "integra_desde_umbral"
	JornadaProtegidaIntegra   ModoJornada = "protegida_integra"
	JornadaPorHoras           ModoJornada = "por_horas"
)

func (m ModoJornada) valido() bool {
	switch m {
	case JornadaProporcional, JornadaIntegra, JornadaIntegraDesdeUmbral,
		JornadaProtegidaIntegra, JornadaPorHoras:
		return true
	default:
		return false
	}
}

// PoliticaJornada conserva el modo y, solo cuando corresponde, el umbral
// exacto. El valor cero no representa ninguna politica.
type PoliticaJornada struct {
	modo        ModoJornada
	tieneUmbral bool
	umbral      baremacion.FraccionJornada
}

// NuevaPoliticaJornada construye una politica sin umbral.
func NuevaPoliticaJornada(modo ModoJornada) (PoliticaJornada, error) {
	politica := PoliticaJornada{modo: modo}
	if err := politica.validar(); err != nil {
		return PoliticaJornada{}, err
	}
	return politica, nil
}

// NuevaPoliticaJornadaDesdeUmbral construye exclusivamente la politica de
// computo integro a partir de una fraccion publicada.
func NuevaPoliticaJornadaDesdeUmbral(umbral baremacion.FraccionJornada) (PoliticaJornada, error) {
	politica := PoliticaJornada{
		modo:        JornadaIntegraDesdeUmbral,
		tieneUmbral: true,
		umbral:      umbral,
	}
	if err := politica.validar(); err != nil {
		return PoliticaJornada{}, err
	}
	return politica, nil
}

func (p PoliticaJornada) Modo() ModoJornada { return p.modo }
func (p PoliticaJornada) Umbral() (baremacion.FraccionJornada, bool) {
	return p.umbral, p.tieneUmbral
}

func (p PoliticaJornada) validar() error {
	if !p.modo.valido() {
		return nuevoError("politica_jornada.modo", CodigoPoliticaIncompleta)
	}
	if p.modo == JornadaIntegraDesdeUmbral {
		if !p.tieneUmbral || !p.umbral.EsValida() {
			return nuevoError("politica_jornada.umbral", CodigoPoliticaIncompleta)
		}
		return nil
	}
	if p.tieneUmbral || p.umbral.EsValida() {
		return nuevoError("politica_jornada.umbral", CodigoPoliticaIncompleta)
	}
	return nil
}

// ModoRestos fija en que frontera se conserva o descarta una fraccion temporal.
type ModoRestos string

const (
	RestosConservarExactos    ModoRestos = "conservar_exactos"
	RestosAcumularPorRegla    ModoRestos = "acumular_por_regla"
	RestosDescartarPorPeriodo ModoRestos = "descartar_por_periodo"
	RestosDescartarPorRegla   ModoRestos = "descartar_por_regla"
)

func (m ModoRestos) valido() bool {
	switch m {
	case RestosConservarExactos, RestosAcumularPorRegla,
		RestosDescartarPorPeriodo, RestosDescartarPorRegla:
		return true
	default:
		return false
	}
}

// PoliticaRestos exige una eleccion expresa.
type PoliticaRestos struct{ modo ModoRestos }

// NuevaPoliticaRestos valida un modo conocido.
func NuevaPoliticaRestos(modo ModoRestos) (PoliticaRestos, error) {
	politica := PoliticaRestos{modo: modo}
	if err := politica.validar(); err != nil {
		return PoliticaRestos{}, err
	}
	return politica, nil
}

func (p PoliticaRestos) Modo() ModoRestos { return p.modo }

func (p PoliticaRestos) validar() error {
	if !p.modo.valido() {
		return nuevoError("politica_restos.modo", CodigoPoliticaIncompleta)
	}
	return nil
}

// MomentoRedondeo fija la unica frontera en la que se redondea.
type MomentoRedondeo string

const (
	RedondearPorPeriodo MomentoRedondeo = "periodo"
	RedondearPorRegla   MomentoRedondeo = "regla"
	RedondearPorSeccion MomentoRedondeo = "seccion"
	RedondearEnTotal    MomentoRedondeo = "total"
)

func (m MomentoRedondeo) valido() bool {
	switch m {
	case RedondearPorPeriodo, RedondearPorRegla, RedondearPorSeccion, RedondearEnTotal:
		return true
	default:
		return false
	}
}

// PoliticaRedondeo combina un momento explicito con uno de los modos exactos
// del fundamento comun de baremacion.
type PoliticaRedondeo struct {
	momento MomentoRedondeo
	modo    baremacion.ModoRedondeo
}

// NuevaPoliticaRedondeo no aplica ningun modo por defecto.
func NuevaPoliticaRedondeo(momento MomentoRedondeo, modo baremacion.ModoRedondeo) (PoliticaRedondeo, error) {
	politica := PoliticaRedondeo{momento: momento, modo: modo}
	if err := politica.validar(); err != nil {
		return PoliticaRedondeo{}, err
	}
	return politica, nil
}

func (p PoliticaRedondeo) Momento() MomentoRedondeo      { return p.momento }
func (p PoliticaRedondeo) Modo() baremacion.ModoRedondeo { return p.modo }

func (p PoliticaRedondeo) validar() error {
	if !p.momento.valido() || !p.modo.EsValido() {
		return nuevoError("politica_redondeo", CodigoPoliticaIncompleta)
	}
	return nil
}

// CriterioExperiencia enlaza un eje configurable con un catalogo versionado y
// un conjunto cerrado de claves admitidas. No ejecuta expresiones libres.
type CriterioExperiencia struct {
	clave    string
	catalogo ReferenciaVersionada
	valores  []string
}

// NuevoCriterioExperiencia valida, deduplica mediante rechazo y ordena los
// valores para que el mismo significado produzca los mismos bytes.
func NuevoCriterioExperiencia(
	clave string,
	catalogo ReferenciaVersionada,
	valores []string,
) (CriterioExperiencia, error) {
	criterio := CriterioExperiencia{
		clave:    clave,
		catalogo: catalogo,
		valores:  append([]string(nil), valores...),
	}
	if err := criterio.validar(); err != nil {
		return CriterioExperiencia{}, err
	}
	sort.Strings(criterio.valores)
	return criterio, nil
}

func (c CriterioExperiencia) Clave() string                  { return c.clave }
func (c CriterioExperiencia) Catalogo() ReferenciaVersionada { return c.catalogo }
func (c CriterioExperiencia) Valores() []string              { return append([]string(nil), c.valores...) }

func (c CriterioExperiencia) clonar() CriterioExperiencia {
	c.valores = append([]string(nil), c.valores...)
	return c
}

func (c CriterioExperiencia) validar() error {
	if !claveValida(c.clave) {
		return nuevoError("criterio.clave", CodigoValorNoCanonico)
	}
	if err := c.catalogo.validar("criterio.catalogo"); err != nil {
		return err
	}
	if len(c.valores) == 0 || len(c.valores) > maximoValoresPorCriterio {
		return nuevoError("criterio.valores", CodigoFueraDeLimites)
	}
	vistos := make(map[string]struct{}, len(c.valores))
	for _, valor := range c.valores {
		if !claveValida(valor) {
			return nuevoError("criterio.valor", CodigoValorNoCanonico)
		}
		if _, existe := vistos[valor]; existe {
			return nuevoError("criterio.valor", CodigoValorDuplicado)
		}
		vistos[valor] = struct{}{}
	}
	return nil
}

// SeccionBaremo es una seccion ordenada y acotada del baremo.
type SeccionBaremo struct {
	clave         string
	definicion    ReferenciaVersionada
	orden         uint32
	puntosMinimos baremacion.Puntos
	puntosMaximos baremacion.Puntos
}

// NuevaSeccionBaremo construye una seccion sin inferir sus limites.
func NuevaSeccionBaremo(
	clave string,
	definicion ReferenciaVersionada,
	orden uint32,
	puntosMinimos baremacion.Puntos,
	puntosMaximos baremacion.Puntos,
) (SeccionBaremo, error) {
	seccion := SeccionBaremo{
		clave:         clave,
		definicion:    definicion,
		orden:         orden,
		puntosMinimos: puntosMinimos,
		puntosMaximos: puntosMaximos,
	}
	if err := seccion.validar(); err != nil {
		return SeccionBaremo{}, err
	}
	return seccion, nil
}

func (s SeccionBaremo) Clave() string                    { return s.clave }
func (s SeccionBaremo) Definicion() ReferenciaVersionada { return s.definicion }
func (s SeccionBaremo) Orden() uint32                    { return s.orden }
func (s SeccionBaremo) PuntosMinimos() baremacion.Puntos { return s.puntosMinimos }
func (s SeccionBaremo) PuntosMaximos() baremacion.Puntos { return s.puntosMaximos }

func (s SeccionBaremo) validar() error {
	if !claveValida(s.clave) {
		return nuevoError("seccion.clave", CodigoValorNoCanonico)
	}
	if err := s.definicion.validar("seccion.definicion"); err != nil {
		return err
	}
	if s.orden == 0 || s.orden > maximoOrden {
		return nuevoError("seccion.orden", CodigoFueraDeLimites)
	}
	if !s.puntosMinimos.EsValido() || !s.puntosMaximos.EsValido() ||
		s.puntosMaximos.Micropuntos() <= 0 ||
		s.puntosMinimos.Micropuntos() > s.puntosMaximos.Micropuntos() {
		return nuevoError("seccion.puntos", CodigoFueraDeLimites)
	}
	return nil
}

// ReglaExperiencia configura como transformar experiencia elegible en puntos.
// Es solo modelo: no contiene ni ejecuta un calculador.
type ReglaExperiencia struct {
	clave                  string
	definicion             ReferenciaVersionada
	seccionClave           string
	orden                  uint32
	criterios              []CriterioExperiencia
	grupoConcurrenciaClave string
	prioridadConcurrencia  uint32
	unidadTemporal         PoliticaUnidadTemporal
	jornada                PoliticaJornada
	restos                 PoliticaRestos
	redondeo               PoliticaRedondeo
	puntosPorUnidad        baremacion.Puntos
	maximoUnidades         LimiteUnidades
	maximoPuntos           LimitePuntos
}

// NuevaReglaExperiencia exige coeficiente y politicas completos. No hay una
// jornada, conversion, solape, resto o redondeo implicitos.
func NuevaReglaExperiencia(
	clave string,
	definicion ReferenciaVersionada,
	seccionClave string,
	orden uint32,
	criterios []CriterioExperiencia,
	grupoConcurrenciaClave string,
	prioridadConcurrencia uint32,
	unidadTemporal PoliticaUnidadTemporal,
	jornada PoliticaJornada,
	restos PoliticaRestos,
	redondeo PoliticaRedondeo,
	puntosPorUnidad baremacion.Puntos,
	maximoUnidades LimiteUnidades,
	maximoPuntos LimitePuntos,
) (ReglaExperiencia, error) {
	regla := ReglaExperiencia{
		clave:                  clave,
		definicion:             definicion,
		seccionClave:           seccionClave,
		orden:                  orden,
		criterios:              clonarCriterios(criterios),
		grupoConcurrenciaClave: grupoConcurrenciaClave,
		prioridadConcurrencia:  prioridadConcurrencia,
		unidadTemporal:         unidadTemporal,
		jornada:                jornada,
		restos:                 restos,
		redondeo:               redondeo,
		puntosPorUnidad:        puntosPorUnidad,
		maximoUnidades:         maximoUnidades,
		maximoPuntos:           maximoPuntos,
	}
	if err := regla.validar(); err != nil {
		return ReglaExperiencia{}, err
	}
	sort.Slice(regla.criterios, func(i, j int) bool {
		return regla.criterios[i].clave < regla.criterios[j].clave
	})
	return regla, nil
}

func (r ReglaExperiencia) Clave() string                          { return r.clave }
func (r ReglaExperiencia) Definicion() ReferenciaVersionada       { return r.definicion }
func (r ReglaExperiencia) SeccionClave() string                   { return r.seccionClave }
func (r ReglaExperiencia) Orden() uint32                          { return r.orden }
func (r ReglaExperiencia) Criterios() []CriterioExperiencia       { return clonarCriterios(r.criterios) }
func (r ReglaExperiencia) GrupoConcurrenciaClave() string         { return r.grupoConcurrenciaClave }
func (r ReglaExperiencia) PrioridadConcurrencia() uint32          { return r.prioridadConcurrencia }
func (r ReglaExperiencia) UnidadTemporal() PoliticaUnidadTemporal { return r.unidadTemporal }
func (r ReglaExperiencia) Jornada() PoliticaJornada               { return r.jornada }
func (r ReglaExperiencia) Restos() PoliticaRestos                 { return r.restos }
func (r ReglaExperiencia) Redondeo() PoliticaRedondeo             { return r.redondeo }
func (r ReglaExperiencia) PuntosPorUnidad() baremacion.Puntos     { return r.puntosPorUnidad }
func (r ReglaExperiencia) MaximoUnidades() LimiteUnidades         { return r.maximoUnidades }
func (r ReglaExperiencia) MaximoPuntos() LimitePuntos             { return r.maximoPuntos }

func (r ReglaExperiencia) clonar() ReglaExperiencia {
	r.criterios = clonarCriterios(r.criterios)
	return r
}

func (r ReglaExperiencia) validar() error {
	if !claveValida(r.clave) {
		return nuevoError("regla.clave", CodigoValorNoCanonico)
	}
	if err := r.definicion.validar("regla.definicion"); err != nil {
		return err
	}
	if !claveValida(r.seccionClave) {
		return nuevoError("regla.seccion_clave", CodigoValorNoCanonico)
	}
	if r.orden == 0 || r.orden > maximoOrden {
		return nuevoError("regla.orden", CodigoFueraDeLimites)
	}
	if len(r.criterios) == 0 || len(r.criterios) > maximoCriteriosPorRegla {
		return nuevoError("regla.criterios", CodigoFueraDeLimites)
	}
	if !claveValida(r.grupoConcurrenciaClave) {
		return nuevoError("regla.grupo_concurrencia_clave", CodigoValorNoCanonico)
	}
	if r.prioridadConcurrencia == 0 || r.prioridadConcurrencia > maximoOrden {
		return nuevoError("regla.prioridad_concurrencia", CodigoFueraDeLimites)
	}
	clavesCriterio := make(map[string]struct{}, len(r.criterios))
	for _, criterio := range r.criterios {
		if err := criterio.validar(); err != nil {
			return err
		}
		if _, existe := clavesCriterio[criterio.clave]; existe {
			return nuevoError("regla.criterio", CodigoValorDuplicado)
		}
		clavesCriterio[criterio.clave] = struct{}{}
	}
	if err := r.unidadTemporal.validar(); err != nil {
		return err
	}
	if err := r.jornada.validar(); err != nil {
		return err
	}
	if r.jornada.modo == JornadaPorHoras && r.unidadTemporal.unidadBase != UnidadTemporalHora {
		return nuevoError("regla.jornada_por_horas", CodigoPoliticaIncompleta)
	}
	if err := r.restos.validar(); err != nil {
		return err
	}
	if err := r.redondeo.validar(); err != nil {
		return err
	}
	if !r.puntosPorUnidad.EsValido() || r.puntosPorUnidad.Micropuntos() <= 0 {
		return nuevoError("regla.puntos_por_unidad", CodigoCoeficienteAusente)
	}
	if err := r.maximoUnidades.validar("regla.maximo_unidades"); err != nil {
		return err
	}
	if err := r.maximoPuntos.validar("regla.maximo_puntos"); err != nil {
		return err
	}
	return nil
}

func clonarCriterios(origen []CriterioExperiencia) []CriterioExperiencia {
	clon := make([]CriterioExperiencia, len(origen))
	for indice := range origen {
		clon[indice] = origen[indice].clonar()
	}
	return clon
}
