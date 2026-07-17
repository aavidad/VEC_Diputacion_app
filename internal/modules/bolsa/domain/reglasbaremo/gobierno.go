package reglasbaremo

import (
	"time"
)

const (
	esquemaGobiernoReglasBaremo     = "vec.bolsa.gobierno-reglas-baremo.v1"
	maximoBytesGobiernoReglasBaremo = maximoBytesRepresentacion + 1024*1024
)

// EstadoGobiernoReglasBaremo describe exclusivamente el gobierno de una
// version de contenido inmutable. No se modifica el conjunto al transicionar.
type EstadoGobiernoReglasBaremo string

const (
	EstadoReglasBaremoBorrador   EstadoGobiernoReglasBaremo = "borrador"
	EstadoReglasBaremoPublicada  EstadoGobiernoReglasBaremo = "publicada"
	EstadoReglasBaremoActiva     EstadoGobiernoReglasBaremo = "activa"
	EstadoReglasBaremoSustituida EstadoGobiernoReglasBaremo = "sustituida"
	EstadoReglasBaremoRetirada   EstadoGobiernoReglasBaremo = "retirada"
	EstadoReglasBaremoDescartada EstadoGobiernoReglasBaremo = "descartada"
)

func (e EstadoGobiernoReglasBaremo) Valido() bool {
	switch e {
	case EstadoReglasBaremoBorrador, EstadoReglasBaremoPublicada,
		EstadoReglasBaremoActiva, EstadoReglasBaremoSustituida,
		EstadoReglasBaremoRetirada, EstadoReglasBaremoDescartada:
		return true
	default:
		return false
	}
}

type actoPublicacionReglasBaremo struct {
	actorRef   string
	motivo     MotivoCatalogadoReglasBaremo
	aprobacion AtestacionAprobacionFirmadaReglasBaremo
	instante   time.Time
}

type actoActivacionReglasBaremo struct {
	actorRef     string
	motivo       MotivoCatalogadoReglasBaremo
	dependencias AtestacionDependenciasVigentesReglasBaremo
	instante     time.Time
}

type actoTerminalReglasBaremo struct {
	accion    AccionGobiernoReglasBaremo
	actorRef  string
	motivo    MotivoCatalogadoReglasBaremo
	autoridad AtestacionAutoridadReglasBaremo
	instante  time.Time
}

// VersionGobernadaReglasBaremo envuelve un contenido inmutable. Todos sus
// campos son privados; cada transicion devuelve una copia nueva y aumenta una
// sola vez la revision usada para OCC.
type VersionGobernadaReglasBaremo struct {
	conjunto       ConjuntoReglasBaremo
	revision       uint64
	estado         EstadoGobiernoReglasBaremo
	creadaPor      string
	creadaEn       time.Time
	motivoCreacion MotivoCatalogadoReglasBaremo
	publicacion    *actoPublicacionReglasBaremo
	activacion     *actoActivacionReglasBaremo
	terminal       *actoTerminalReglasBaremo
}

func NuevaVersionGobernadaReglasBaremo(
	conjunto ConjuntoReglasBaremo,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	clon, err := clonarConjuntoGobiernoReglasBaremo(conjunto)
	if err != nil || !referenciaPersonaOpacaValida(actorRef) || motivo.validar() != nil ||
		!instanteGobiernoReglasBaremoValido(instante) {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoValorInvalido
	}
	version := VersionGobernadaReglasBaremo{
		conjunto: clon, revision: 1, estado: EstadoReglasBaremoBorrador,
		creadaPor: actorRef, creadaEn: instante, motivoCreacion: motivo,
	}
	if version.Validar() != nil {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	return version, nil
}

func (v VersionGobernadaReglasBaremo) Revision() uint64 { return v.revision }
func (v VersionGobernadaReglasBaremo) Estado() EstadoGobiernoReglasBaremo {
	return v.estado
}
func (v VersionGobernadaReglasBaremo) CreadaPor() string   { return v.creadaPor }
func (v VersionGobernadaReglasBaremo) CreadaEn() time.Time { return v.creadaEn }
func (v VersionGobernadaReglasBaremo) MotivoCreacion() MotivoCatalogadoReglasBaremo {
	return v.motivoCreacion
}

// InstanteUltimaActuacion devuelve el instante de negocio que forma parte del
// estado actual: creacion para el borrador o el acto de su ultima transicion.
// No representa el reloj de persistencia ni el momento de registro durable.
func (v VersionGobernadaReglasBaremo) InstanteUltimaActuacion() (time.Time, error) {
	if v.Validar() != nil {
		return time.Time{}, ErrGobiernoInvarianteQuebrada
	}
	instante := v.ultimaActuacionEn()
	if !instanteGobiernoReglasBaremoValido(instante) {
		return time.Time{}, ErrGobiernoInvarianteQuebrada
	}
	return instante, nil
}

// ActorUltimaActuacion devuelve la referencia de persona declarada opaca que
// produjo el estado actual. Su formato no acredita por si solo procedencia ni
// minimizacion; esa garantia corresponde a la frontera de identidad.
func (v VersionGobernadaReglasBaremo) ActorUltimaActuacion() (string, error) {
	if v.Validar() != nil {
		return "", ErrGobiernoInvarianteQuebrada
	}
	actor, _ := v.actorYMotivoUltimaActuacion()
	if !referenciaPersonaOpacaValida(actor) {
		return "", ErrGobiernoInvarianteQuebrada
	}
	return actor, nil
}

// MotivoUltimaActuacion devuelve la entrada exacta del catalogo incorporada al
// estado actual. Su etiqueta humana permanece fuera del agregado.
func (v VersionGobernadaReglasBaremo) MotivoUltimaActuacion() (
	MotivoCatalogadoReglasBaremo,
	error,
) {
	if v.Validar() != nil {
		return MotivoCatalogadoReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	_, motivo := v.actorYMotivoUltimaActuacion()
	if motivo.validar() != nil {
		return MotivoCatalogadoReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	return motivo, nil
}

func (v VersionGobernadaReglasBaremo) Conjunto() (ConjuntoReglasBaremo, error) {
	if v.Validar() != nil {
		return ConjuntoReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	return clonarConjuntoGobiernoReglasBaremo(v.conjunto)
}

func (v VersionGobernadaReglasBaremo) ReferenciaContenido() (ReferenciaVersionada, error) {
	if v.conjunto.Validar() != nil {
		return ReferenciaVersionada{}, ErrGobiernoInvarianteQuebrada
	}
	referencia, err := v.conjunto.ReferenciaVersionada()
	if err != nil {
		return ReferenciaVersionada{}, ErrGobiernoInvarianteQuebrada
	}
	return referencia, nil
}

func (v VersionGobernadaReglasBaremo) VinculoEstado() (VinculoEstadoReglasBaremo, error) {
	referencia, errReferencia := v.ReferenciaContenido()
	huella, errHuella := v.HuellaSHA256()
	if errReferencia != nil || errHuella != nil {
		return VinculoEstadoReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	return NuevoVinculoEstadoReglasBaremo(referencia, v.revision, huella)
}

// ConvocatoriaActivacion devuelve la referencia exacta que el verificador de
// dependencias ligo al activar la version. El segundo resultado distingue una
// ausencia legitima de activacion en borradores, versiones publicadas y
// descartadas. Las versiones sustituidas o retiradas conservan el vinculo para
// permitir la reproduccion historica.
//
// No expone el acto ni la atestacion internos. La referencia se devuelve por
// valor y no comparte estado mutable con la version gobernada.
func (v VersionGobernadaReglasBaremo) ConvocatoriaActivacion() (
	ReferenciaVersionada,
	bool,
	error,
) {
	if v.Validar() != nil {
		return ReferenciaVersionada{}, false, ErrGobiernoInvarianteQuebrada
	}
	if v.activacion == nil {
		return ReferenciaVersionada{}, false, nil
	}
	return v.activacion.dependencias.Convocatoria(), true, nil
}

// DependenciasContenido devuelve bases, definiciones y catalogos exactos en
// orden canonico para que el verificador externo pueda atestarlos.
func (v VersionGobernadaReglasBaremo) DependenciasContenido() ([]ReferenciaVersionada, error) {
	if v.conjunto.Validar() != nil {
		return nil, ErrGobiernoInvarianteQuebrada
	}
	referencias := make([]ReferenciaVersionada, 0)
	porReferencia := make(map[string]ReferenciaVersionada)
	agregar := func(referencia ReferenciaVersionada) bool {
		if referencia.validar("dependencia") != nil {
			return false
		}
		previa, existe := porReferencia[referencia.referencia]
		if existe {
			return referenciasVersionadasIguales(previa, referencia)
		}
		porReferencia[referencia.referencia] = referencia
		referencias = append(referencias, referencia)
		return true
	}
	if !agregar(v.conjunto.Bases()) {
		return nil, ErrGobiernoInvarianteQuebrada
	}
	for _, seccion := range v.conjunto.Secciones() {
		if !agregar(seccion.Definicion()) {
			return nil, ErrGobiernoInvarianteQuebrada
		}
	}
	for _, grupo := range v.conjunto.GruposConcurrenciaExperiencia() {
		if !agregar(grupo.Definicion()) {
			return nil, ErrGobiernoInvarianteQuebrada
		}
	}
	for _, regla := range v.conjunto.ReglasExperiencia() {
		if !agregar(regla.Definicion()) {
			return nil, ErrGobiernoInvarianteQuebrada
		}
		for _, criterio := range regla.Criterios() {
			if !agregar(criterio.Catalogo()) {
				return nil, ErrGobiernoInvarianteQuebrada
			}
		}
	}
	sortReferenciasVersionadas(referencias)
	return append([]ReferenciaVersionada(nil), referencias...), nil
}

func (v VersionGobernadaReglasBaremo) Publicar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	aprobacion AtestacionAprobacionFirmadaReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	if err := v.prepararTransicion(revisionEsperada, EstadoReglasBaremoBorrador, actorRef, motivo, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if err := aprobacion.validarPara(v, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	siguiente, _ := v.Clonar()
	siguiente.revision++
	siguiente.estado = EstadoReglasBaremoPublicada
	siguiente.publicacion = &actoPublicacionReglasBaremo{
		actorRef: actorRef, motivo: motivo, aprobacion: aprobacion.clonar(), instante: instante,
	}
	return siguiente.validadaTrasTransicion()
}

func (v VersionGobernadaReglasBaremo) Activar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	dependencias AtestacionDependenciasVigentesReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	if err := v.prepararTransicion(revisionEsperada, EstadoReglasBaremoPublicada, actorRef, motivo, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if err := dependencias.validarPara(v, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	siguiente, _ := v.Clonar()
	siguiente.revision++
	siguiente.estado = EstadoReglasBaremoActiva
	siguiente.activacion = &actoActivacionReglasBaremo{
		actorRef: actorRef, motivo: motivo, dependencias: dependencias.clonar(), instante: instante,
	}
	return siguiente.validadaTrasTransicion()
}

func (v VersionGobernadaReglasBaremo) Sustituir(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	sucesora ReferenciaVersionada,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	return v.aplicarTerminal(
		revisionEsperada, EstadoReglasBaremoSustituida, AccionSustituirReglasBaremo,
		actorRef, motivo, &sucesora, autoridad, instante,
	)
}

func (v VersionGobernadaReglasBaremo) Retirar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	return v.aplicarTerminal(
		revisionEsperada, EstadoReglasBaremoRetirada, AccionRetirarReglasBaremo,
		actorRef, motivo, nil, autoridad, instante,
	)
}

func (v VersionGobernadaReglasBaremo) Descartar(
	revisionEsperada uint64,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	if err := v.prepararTransicion(revisionEsperada, EstadoReglasBaremoBorrador, actorRef, motivo, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if err := autoridad.validarPara(v, AccionDescartarReglasBaremo, actorRef, nil, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	siguiente, _ := v.Clonar()
	siguiente.revision++
	siguiente.estado = EstadoReglasBaremoDescartada
	siguiente.terminal = &actoTerminalReglasBaremo{
		accion: AccionDescartarReglasBaremo,
	}
	// Se asigna por campos para evitar que un literal incompleto pueda olvidar
	// una evidencia al evolucionar el tipo.
	siguiente.terminal.accion = AccionDescartarReglasBaremo
	siguiente.terminal.actorRef = actorRef
	siguiente.terminal.motivo = motivo
	siguiente.terminal.autoridad = autoridad.clonar()
	siguiente.terminal.instante = instante
	return siguiente.validadaTrasTransicion()
}

func (v VersionGobernadaReglasBaremo) aplicarTerminal(
	revisionEsperada uint64,
	estadoResultado EstadoGobiernoReglasBaremo,
	accion AccionGobiernoReglasBaremo,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	relacionada *ReferenciaVersionada,
	autoridad AtestacionAutoridadReglasBaremo,
	instante time.Time,
) (VersionGobernadaReglasBaremo, error) {
	if err := v.prepararTransicion(revisionEsperada, EstadoReglasBaremoActiva, actorRef, motivo, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if err := autoridad.validarPara(v, accion, actorRef, relacionada, instante); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	siguiente, _ := v.Clonar()
	siguiente.revision++
	siguiente.estado = estadoResultado
	siguiente.terminal = &actoTerminalReglasBaremo{
		accion: accion, actorRef: actorRef, motivo: motivo,
		autoridad: autoridad.clonar(), instante: instante,
	}
	return siguiente.validadaTrasTransicion()
}

func (v VersionGobernadaReglasBaremo) prepararTransicion(
	revisionEsperada uint64,
	estadoEsperado EstadoGobiernoReglasBaremo,
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	instante time.Time,
) error {
	if v.Validar() != nil {
		return ErrGobiernoInvarianteQuebrada
	}
	if revisionEsperada != v.revision {
		return ErrGobiernoRevisionConflicto
	}
	if v.estado != estadoEsperado {
		return ErrGobiernoTransicionProhibida
	}
	if !referenciaPersonaOpacaValida(actorRef) || motivo.validar() != nil {
		return ErrGobiernoValorInvalido
	}
	if !instanteGobiernoReglasBaremoValido(instante) || instante.Before(v.ultimaActuacionEn()) {
		return ErrGobiernoInstanteInvalido
	}
	return nil
}

func (v VersionGobernadaReglasBaremo) validadaTrasTransicion() (VersionGobernadaReglasBaremo, error) {
	if v.Validar() != nil {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	return v.Clonar()
}

func (v VersionGobernadaReglasBaremo) ultimaActuacionEn() time.Time {
	if v.terminal != nil {
		return v.terminal.instante
	}
	if v.activacion != nil {
		return v.activacion.instante
	}
	if v.publicacion != nil {
		return v.publicacion.instante
	}
	return v.creadaEn
}

func (v VersionGobernadaReglasBaremo) actorYMotivoUltimaActuacion() (
	string,
	MotivoCatalogadoReglasBaremo,
) {
	if v.terminal != nil {
		return v.terminal.actorRef, v.terminal.motivo
	}
	if v.activacion != nil {
		return v.activacion.actorRef, v.activacion.motivo
	}
	if v.publicacion != nil {
		return v.publicacion.actorRef, v.publicacion.motivo
	}
	return v.creadaPor, v.motivoCreacion
}

func (v VersionGobernadaReglasBaremo) Validar() error {
	if v.conjunto.Validar() != nil || v.revision == 0 || v.revision > maximoVersion ||
		!v.estado.Valido() || !referenciaPersonaOpacaValida(v.creadaPor) ||
		!instanteGobiernoReglasBaremoValido(v.creadaEn) || v.motivoCreacion.validar() != nil {
		return ErrGobiernoInvarianteQuebrada
	}
	switch v.estado {
	case EstadoReglasBaremoBorrador:
		if v.revision != 1 || v.publicacion != nil || v.activacion != nil || v.terminal != nil {
			return ErrGobiernoInvarianteQuebrada
		}
	case EstadoReglasBaremoPublicada:
		if v.revision != 2 || v.publicacion == nil || v.activacion != nil || v.terminal != nil {
			return ErrGobiernoInvarianteQuebrada
		}
		anterior := v.anteriorBorrador()
		if v.publicacion.validarPara(anterior, v.publicacion.instante) != nil ||
			!actoComunValido(v.publicacion.actorRef, v.publicacion.motivo, v.publicacion.instante, v.creadaEn) {
			return ErrGobiernoInvarianteQuebrada
		}
	case EstadoReglasBaremoActiva:
		if v.revision != 3 || v.publicacion == nil || v.activacion == nil || v.terminal != nil {
			return ErrGobiernoInvarianteQuebrada
		}
		anterior := v.anteriorPublicada()
		if anterior.Validar() != nil || v.activacion.dependencias.validarPara(anterior, v.activacion.instante) != nil ||
			!actoComunValido(v.activacion.actorRef, v.activacion.motivo, v.activacion.instante, v.publicacion.instante) {
			return ErrGobiernoInvarianteQuebrada
		}
	case EstadoReglasBaremoSustituida, EstadoReglasBaremoRetirada:
		accion := AccionRetirarReglasBaremo
		if v.estado == EstadoReglasBaremoSustituida {
			accion = AccionSustituirReglasBaremo
		}
		if v.revision != 4 || v.publicacion == nil || v.activacion == nil || v.terminal == nil ||
			v.terminal.accion != accion {
			return ErrGobiernoInvarianteQuebrada
		}
		anterior := v.anteriorActiva()
		relacionada := v.terminal.autoridad.relacionada
		if anterior.Validar() != nil ||
			v.terminal.autoridad.validarPara(anterior, accion, v.terminal.actorRef, relacionada, v.terminal.instante) != nil ||
			!actoComunValido(v.terminal.actorRef, v.terminal.motivo, v.terminal.instante, v.activacion.instante) {
			return ErrGobiernoInvarianteQuebrada
		}
	case EstadoReglasBaremoDescartada:
		if v.revision != 2 || v.publicacion != nil || v.activacion != nil || v.terminal == nil ||
			v.terminal.accion != AccionDescartarReglasBaremo {
			return ErrGobiernoInvarianteQuebrada
		}
		anterior := v.anteriorBorrador()
		if v.terminal.autoridad.validarPara(
			anterior, AccionDescartarReglasBaremo, v.terminal.actorRef, nil, v.terminal.instante,
		) != nil || !actoComunValido(
			v.terminal.actorRef, v.terminal.motivo, v.terminal.instante, v.creadaEn,
		) {
			return ErrGobiernoInvarianteQuebrada
		}
	}
	return nil
}
