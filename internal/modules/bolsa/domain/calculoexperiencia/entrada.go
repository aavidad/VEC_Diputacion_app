package calculoexperiencia

import (
	"sort"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	maximoTramosEntrada        = 10_000
	maximoAtributosPorTramo    = 64
	maximoCaracteresClave      = 128
	maximoCaracteresReferencia = 512
	caracteresCargaTokenOpaco  = 64
	prefijoInstantaneaEntrada  = "iex_"
	prefijoTramoEntrada        = "trm_"
	prefijoServicioEntrada     = "srv_"
	prefijoAtestacionEntrada   = "ati_"
)

// ModoPeriodoServicio diferencia de forma expresa un periodo cerrado de uno
// que seguia en curso al generar la instantanea.
type ModoPeriodoServicio string

const (
	PeriodoServicioCerrado ModoPeriodoServicio = "cerrado"
	PeriodoServicioEnCurso ModoPeriodoServicio = "en_curso"
)

// PeriodoServicio conserva las fechas informadas por la fuente. No decide si
// el extremo final se incluye: esa decision pertenece a la regla publicada.
type PeriodoServicio struct {
	modo         ModoPeriodoServicio
	desde        baremacion.FechaCivil
	finInformado baremacion.FechaCivil
}

// NuevoPeriodoServicioCerrado acepta fin igual a inicio porque una regla con
// extremo inclusivo puede representar asi un servicio de un dia.
func NuevoPeriodoServicioCerrado(
	desde baremacion.FechaCivil,
	finInformado baremacion.FechaCivil,
) (PeriodoServicio, error) {
	periodo := PeriodoServicio{
		modo:         PeriodoServicioCerrado,
		desde:        desde,
		finInformado: finInformado,
	}
	if err := periodo.validar(); err != nil {
		return PeriodoServicio{}, err
	}
	return periodo, nil
}

// NuevoPeriodoServicioEnCurso no inventa una fecha final provisional.
func NuevoPeriodoServicioEnCurso(desde baremacion.FechaCivil) (PeriodoServicio, error) {
	periodo := PeriodoServicio{modo: PeriodoServicioEnCurso, desde: desde}
	if err := periodo.validar(); err != nil {
		return PeriodoServicio{}, err
	}
	return periodo, nil
}

func (p PeriodoServicio) Modo() ModoPeriodoServicio    { return p.modo }
func (p PeriodoServicio) Desde() baremacion.FechaCivil { return p.desde }

// FinInformado devuelve false cuando el servicio estaba en curso.
func (p PeriodoServicio) FinInformado() (baremacion.FechaCivil, bool) {
	return p.finInformado, p.modo == PeriodoServicioCerrado
}

func (p PeriodoServicio) EnCurso() bool { return p.modo == PeriodoServicioEnCurso }

func (p PeriodoServicio) validar() error {
	if !p.desde.EsValida() {
		return nuevoError("periodo.desde", CodigoValorInvalido)
	}
	switch p.modo {
	case PeriodoServicioCerrado:
		if !p.finInformado.EsValida() {
			return nuevoError("periodo.fin_informado", CodigoValorInvalido)
		}
		comparacion, err := p.desde.Comparar(p.finInformado)
		if err != nil || comparacion > 0 {
			return nuevoError("periodo", CodigoValorInvalido)
		}
	case PeriodoServicioEnCurso:
		if p.finInformado.EsValida() {
			return nuevoError("periodo.fin_informado", CodigoValorInvalido)
		}
	default:
		return nuevoError("periodo.modo", CodigoValorInvalido)
	}
	return nil
}

// AtributoCatalogado aporta una clave normalizada y un valor gobernado por una
// version exacta de catalogo. No admite descripciones ni texto libre.
type AtributoCatalogado struct {
	clave    string
	catalogo reglasbaremo.ReferenciaVersionada
	valor    string
}

func NuevoAtributoCatalogado(
	clave string,
	catalogo reglasbaremo.ReferenciaVersionada,
	valor string,
) (AtributoCatalogado, error) {
	atributo := AtributoCatalogado{clave: clave, catalogo: catalogo, valor: valor}
	if err := atributo.validar(); err != nil {
		return AtributoCatalogado{}, err
	}
	return atributo, nil
}

func (a AtributoCatalogado) Clave() string                               { return a.clave }
func (a AtributoCatalogado) Catalogo() reglasbaremo.ReferenciaVersionada { return a.catalogo }
func (a AtributoCatalogado) Valor() string                               { return a.valor }

func (a AtributoCatalogado) validar() error {
	if !claveGobernadaValida(a.clave) || clavePersonalReservada(a.clave) {
		return nuevoError("atributo.clave", CodigoValorNoCanonico)
	}
	if err := validarReferenciaVersionada(a.catalogo, "atributo.catalogo"); err != nil {
		return err
	}
	if !referenciaCatalogoValida(a.catalogo.Referencia()) {
		return nuevoError("atributo.catalogo", CodigoValorNoCanonico)
	}
	if !claveGobernadaValida(a.valor) || pareceIdentificadorPersonalDirecto(a.valor) {
		return nuevoError("atributo.valor", CodigoValorNoCanonico)
	}
	return nil
}

type modoComputoIntegroAtestado string

const (
	computoIntegroAusente  modoComputoIntegroAtestado = "ausente"
	computoIntegroAtestado modoComputoIntegroAtestado = "atestado"
)

// ComputoIntegroAtestado solo informa de la consecuencia computable. Nunca
// contiene la causa medica, familiar, sindical o de otra naturaleza.
type ComputoIntegroAtestado struct {
	modo       modoComputoIntegroAtestado
	referencia reglasbaremo.ReferenciaVersionada
}

func SinComputoIntegroAtestado() ComputoIntegroAtestado {
	return ComputoIntegroAtestado{modo: computoIntegroAusente}
}

func NuevoComputoIntegroAtestado(
	referencia reglasbaremo.ReferenciaVersionada,
) (ComputoIntegroAtestado, error) {
	atestacion := ComputoIntegroAtestado{
		modo:       computoIntegroAtestado,
		referencia: referencia,
	}
	if err := atestacion.validar(); err != nil {
		return ComputoIntegroAtestado{}, err
	}
	return atestacion, nil
}

func (a ComputoIntegroAtestado) EstaAtestado() bool {
	return a.modo == computoIntegroAtestado
}

func (a ComputoIntegroAtestado) Referencia() (reglasbaremo.ReferenciaVersionada, bool) {
	return a.referencia, a.EstaAtestado()
}

func (a ComputoIntegroAtestado) validar() error {
	switch a.modo {
	case computoIntegroAusente:
		if !referenciaVersionadaVacia(a.referencia) {
			return nuevoError("computo_integro_atestado.referencia", CodigoValorInvalido)
		}
	case computoIntegroAtestado:
		if err := validarReferenciaTokenVersionada(
			a.referencia,
			prefijoAtestacionEntrada,
			"computo_integro_atestado.referencia",
		); err != nil {
			return err
		}
	default:
		return nuevoError("computo_integro_atestado.modo", CodigoValorInvalido)
	}
	return nil
}

// TramoExperiencia es un hecho temporal minimo. servicioRef permite reconocer
// tramos procedentes del mismo servicio sin revelar identidad ni empleador.
// Sigue siendo un seudonimo sujeto a las mismas medidas de proteccion que el
// resto del expediente; no convierte el dato en anonimo. Los prefijos y la
// carga hexadecimal solo fijan el formato: el adaptador de fuente debe generar
// el token en servidor con aleatoriedad o seudonimizacion institucional que no
// permita probar identificadores por diccionario.
type TramoExperiencia struct {
	referencia  reglasbaremo.ReferenciaVersionada
	servicioRef string
	periodo     PeriodoServicio
	jornada     baremacion.FraccionJornada
	atestacion  ComputoIntegroAtestado
	atributos   []AtributoCatalogado
}

func NuevoTramoExperiencia(
	referencia reglasbaremo.ReferenciaVersionada,
	servicioRef string,
	periodo PeriodoServicio,
	jornada baremacion.FraccionJornada,
	atestacion ComputoIntegroAtestado,
	atributos []AtributoCatalogado,
) (TramoExperiencia, error) {
	if len(atributos) > maximoAtributosPorTramo {
		return TramoExperiencia{}, nuevoError("tramo.atributos", CodigoFueraDeLimites)
	}
	tramo := TramoExperiencia{
		referencia:  referencia,
		servicioRef: servicioRef,
		periodo:     periodo,
		jornada:     jornada,
		atestacion:  atestacion,
		atributos:   append([]AtributoCatalogado(nil), atributos...),
	}
	if err := tramo.validar(false); err != nil {
		return TramoExperiencia{}, err
	}
	sort.Slice(tramo.atributos, func(i, j int) bool {
		return compararAtributos(tramo.atributos[i], tramo.atributos[j]) < 0
	})
	if err := tramo.validar(true); err != nil {
		return TramoExperiencia{}, err
	}
	return tramo, nil
}

func (t TramoExperiencia) Referencia() reglasbaremo.ReferenciaVersionada { return t.referencia }
func (t TramoExperiencia) ServicioRef() string                           { return t.servicioRef }
func (t TramoExperiencia) Periodo() PeriodoServicio                      { return t.periodo }
func (t TramoExperiencia) Jornada() baremacion.FraccionJornada           { return t.jornada }
func (t TramoExperiencia) Atestacion() ComputoIntegroAtestado            { return t.atestacion }
func (t TramoExperiencia) Atributos() []AtributoCatalogado {
	return append([]AtributoCatalogado(nil), t.atributos...)
}

func (t TramoExperiencia) clonar() TramoExperiencia {
	t.atributos = append([]AtributoCatalogado(nil), t.atributos...)
	return t
}

func (t TramoExperiencia) validar(exigirOrden bool) error {
	if err := validarReferenciaTokenVersionada(
		t.referencia,
		prefijoTramoEntrada,
		"tramo.referencia",
	); err != nil {
		return err
	}
	if !referenciaOpacaValida(t.servicioRef) {
		return nuevoError("tramo.servicio_ref", CodigoValorNoCanonico)
	}
	if err := t.periodo.validar(); err != nil {
		return err
	}
	if !t.jornada.EsValida() {
		return nuevoError("tramo.jornada", CodigoValorInvalido)
	}
	if err := t.atestacion.validar(); err != nil {
		return err
	}
	if len(t.atributos) > maximoAtributosPorTramo {
		return nuevoError("tramo.atributos", CodigoFueraDeLimites)
	}
	claves := make(map[string]struct{}, len(t.atributos))
	for indice, atributo := range t.atributos {
		if err := atributo.validar(); err != nil {
			return err
		}
		if _, existe := claves[atributo.clave]; existe {
			return nuevoError("tramo.atributo.clave", CodigoValorDuplicado)
		}
		claves[atributo.clave] = struct{}{}
		if exigirOrden && indice > 0 && compararAtributos(t.atributos[indice-1], atributo) >= 0 {
			return nuevoError("tramo.atributos", CodigoValorNoCanonico)
		}
	}
	return nil
}

// EntradaExperiencia es la instantanea inmutable y minimizada que consume el
// futuro calculador. Una entrada vacia es valida y representa cero hechos.
type EntradaExperiencia struct {
	instantanea reglasbaremo.ReferenciaVersionada
	tramos      []TramoExperiencia
}

func NuevaEntradaExperiencia(
	instantanea reglasbaremo.ReferenciaVersionada,
	tramos []TramoExperiencia,
) (EntradaExperiencia, error) {
	if len(tramos) > maximoTramosEntrada {
		return EntradaExperiencia{}, nuevoError("entrada.tramos", CodigoFueraDeLimites)
	}
	if err := comprobarPresupuestoCanonicoEntrada(instantanea, tramos); err != nil {
		return EntradaExperiencia{}, err
	}
	entrada := EntradaExperiencia{
		instantanea: instantanea,
		tramos:      clonarTramos(tramos),
	}
	if err := entrada.validar(false); err != nil {
		return EntradaExperiencia{}, err
	}
	sort.Slice(entrada.tramos, func(i, j int) bool {
		return compararTramos(entrada.tramos[i], entrada.tramos[j]) < 0
	})
	if err := entrada.validar(true); err != nil {
		return EntradaExperiencia{}, err
	}
	if _, err := entrada.RepresentacionCanonica(); err != nil {
		return EntradaExperiencia{}, err
	}
	return entrada, nil
}

func (e EntradaExperiencia) Instantanea() reglasbaremo.ReferenciaVersionada {
	return e.instantanea
}

func (e EntradaExperiencia) Tramos() []TramoExperiencia { return clonarTramos(e.tramos) }

func (e EntradaExperiencia) Validar() error { return e.validar(true) }

func (e EntradaExperiencia) validar(exigirOrden bool) error {
	if err := validarReferenciaTokenVersionada(
		e.instantanea,
		prefijoInstantaneaEntrada,
		"entrada.instantanea",
	); err != nil {
		return err
	}
	if len(e.tramos) > maximoTramosEntrada {
		return nuevoError("entrada.tramos", CodigoFueraDeLimites)
	}
	referencias := make(map[string]struct{}, len(e.tramos))
	for indice, tramo := range e.tramos {
		if err := tramo.validar(true); err != nil {
			return err
		}
		claveReferencia := tramo.referencia.Referencia()
		if _, existe := referencias[claveReferencia]; existe {
			return nuevoError("entrada.tramo.referencia", CodigoValorDuplicado)
		}
		referencias[claveReferencia] = struct{}{}
		if exigirOrden && indice > 0 && compararTramos(e.tramos[indice-1], tramo) >= 0 {
			return nuevoError("entrada.tramos", CodigoValorNoCanonico)
		}
	}
	return nil
}

func clonarTramos(origen []TramoExperiencia) []TramoExperiencia {
	clon := make([]TramoExperiencia, len(origen))
	for indice := range origen {
		clon[indice] = origen[indice].clonar()
	}
	return clon
}

func compararTramos(izquierda, derecha TramoExperiencia) int {
	if izquierda.referencia.Referencia() < derecha.referencia.Referencia() {
		return -1
	}
	if izquierda.referencia.Referencia() > derecha.referencia.Referencia() {
		return 1
	}
	if izquierda.referencia.Version() < derecha.referencia.Version() {
		return -1
	}
	if izquierda.referencia.Version() > derecha.referencia.Version() {
		return 1
	}
	if izquierda.referencia.HuellaSHA256() < derecha.referencia.HuellaSHA256() {
		return -1
	}
	if izquierda.referencia.HuellaSHA256() > derecha.referencia.HuellaSHA256() {
		return 1
	}
	return 0
}

func compararAtributos(izquierda, derecha AtributoCatalogado) int {
	if izquierda.clave < derecha.clave {
		return -1
	}
	if izquierda.clave > derecha.clave {
		return 1
	}
	if izquierda.catalogo.Referencia() < derecha.catalogo.Referencia() {
		return -1
	}
	if izquierda.catalogo.Referencia() > derecha.catalogo.Referencia() {
		return 1
	}
	if izquierda.valor < derecha.valor {
		return -1
	}
	if izquierda.valor > derecha.valor {
		return 1
	}
	return 0
}

func validarReferenciaVersionada(
	referencia reglasbaremo.ReferenciaVersionada,
	campo string,
) error {
	reconstruida, err := reglasbaremo.NuevaReferenciaVersionada(
		referencia.Referencia(),
		referencia.Version(),
		referencia.HuellaSHA256(),
	)
	if err != nil || reconstruida.Referencia() != referencia.Referencia() ||
		reconstruida.Version() != referencia.Version() ||
		reconstruida.HuellaSHA256() != referencia.HuellaSHA256() {
		return nuevoError(campo, CodigoValorNoCanonico)
	}
	return nil
}

func validarReferenciaTokenVersionada(
	referencia reglasbaremo.ReferenciaVersionada,
	prefijo string,
	campo string,
) error {
	if err := validarReferenciaVersionada(referencia, campo); err != nil {
		return err
	}
	if !tokenOpacoValido(referencia.Referencia(), prefijo) {
		return nuevoError(campo, CodigoValorNoCanonico)
	}
	return nil
}

func referenciaVersionadaVacia(referencia reglasbaremo.ReferenciaVersionada) bool {
	return referencia.Referencia() == "" && referencia.Version() == 0 && referencia.HuellaSHA256() == ""
}

func claveGobernadaValida(valor string) bool {
	if len(valor) == 0 || len(valor) > maximoCaracteresClave {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			(indice > 0 && (caracter == '.' || caracter == '_' || caracter == '-')) {
			continue
		}
		return false
	}
	return true
}

func referenciaOpacaValida(valor string) bool {
	return tokenOpacoValido(valor, prefijoServicioEntrada)
}

func tokenOpacoValido(valor string, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) ||
		len(valor) != len(prefijo)+caracteresCargaTokenOpaco {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= '0' && caracter <= '9') || (caracter >= 'a' && caracter <= 'f') {
			continue
		}
		return false
	}
	return true
}

func referenciaCatalogoValida(referencia string) bool {
	const prefijo = "catalogo:"
	if !strings.HasPrefix(referencia, prefijo) || len(referencia) <= len(prefijo) {
		return false
	}
	contenido := strings.ToLower(referencia[len(prefijo):])
	return !clavePersonalReservada(contenido) && !pareceIdentificadorPersonalDirecto(contenido)
}

func pareceIdentificadorPersonalDirecto(valor string) bool {
	longitud := len(valor)
	if longitud == 9 {
		// DNI: ocho cifras y letra. NIE: X/Y/Z, siete cifras y letra.
		dni := esRangoDigitos(valor, 0, 8) && esLetraASCII(valor[8])
		nie := (valor[0] == 'x' || valor[0] == 'y' || valor[0] == 'z') &&
			esRangoDigitos(valor, 1, 8) && esLetraASCII(valor[8])
		return dni || nie
	}
	if longitud == 24 && strings.HasPrefix(valor, "es") {
		return esRangoDigitos(valor, 2, longitud)
	}
	return false
}

func esRangoDigitos(valor string, desde, hasta int) bool {
	if desde < 0 || hasta > len(valor) || desde >= hasta {
		return false
	}
	for indice := desde; indice < hasta; indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
	}
	return true
}

func esLetraASCII(valor byte) bool {
	return valor >= 'a' && valor <= 'z'
}

func clavePersonalReservada(clave string) bool {
	for _, prefijo := range []string{
		"dni", "nif", "nie", "nombre", "apellido", "persona", "correo", "email",
		"salud", "diagnostico", "causa", "motivo", "telefono", "direccion",
		"nacimiento", "iban", "nss", "naf", "seguridad_social",
	} {
		if clave == prefijo || strings.HasPrefix(clave, prefijo+"_") ||
			strings.HasPrefix(clave, prefijo+".") || strings.HasPrefix(clave, prefijo+"-") {
			return true
		}
	}
	for _, token := range strings.FieldsFunc(clave, func(caracter rune) bool {
		return caracter == '_' || caracter == '.' || caracter == '-'
	}) {
		switch token {
		case "dni", "nif", "nie", "nombre", "apellido", "apellidos", "persona",
			"correo", "email", "salud", "diagnostico", "causa", "motivo",
			"telefono", "direccion", "nacimiento", "iban", "ss", "nss", "naf":
			return true
		}
	}
	return false
}

// comprobarPresupuestoCanonicoEntrada usa una cota superior deliberadamente
// conservadora antes de reservar copias. La comprobacion exacta se repite tras
// serializar. El adaptador y los catalogos gobernados siguen siendo la
// autoridad que garantiza que una clave permitida no codifique datos personales.
func comprobarPresupuestoCanonicoEntrada(
	instantanea reglasbaremo.ReferenciaVersionada,
	tramos []TramoExperiencia,
) error {
	total := 512 + tamanoReferenciaEstimado(instantanea)
	for _, tramo := range tramos {
		if len(tramo.atributos) > maximoAtributosPorTramo {
			return nuevoError("tramo.atributos", CodigoFueraDeLimites)
		}
		total += 768 + tamanoReferenciaEstimado(tramo.referencia) + len(tramo.servicioRef)
		if tramo.atestacion.EstaAtestado() {
			total += tamanoReferenciaEstimado(tramo.atestacion.referencia)
		}
		for _, atributo := range tramo.atributos {
			total += 256 + len(atributo.clave) + len(atributo.valor) +
				tamanoReferenciaEstimado(atributo.catalogo)
		}
		if total > maximoBytesRepresentacionEntrada {
			return nuevoError("representacion_canonica", CodigoFueraDeLimites)
		}
	}
	return nil
}

func tamanoReferenciaEstimado(referencia reglasbaremo.ReferenciaVersionada) int {
	return 160 + len(referencia.Referencia()) + len(referencia.HuellaSHA256())
}
