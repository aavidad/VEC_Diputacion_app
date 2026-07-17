package reglasbaremo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	if err != nil || !referenciaValida(actorRef) || motivo.validar() != nil ||
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
	if !referenciaValida(actorRef) || motivo.validar() != nil {
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

func (v VersionGobernadaReglasBaremo) Validar() error {
	if v.conjunto.Validar() != nil || v.revision == 0 || v.revision > maximoVersion ||
		!v.estado.Valido() || !referenciaValida(v.creadaPor) ||
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

func actoComunValido(
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	instante, noAntes time.Time,
) bool {
	return referenciaValida(actorRef) && motivo.validar() == nil &&
		instanteGobiernoReglasBaremoValido(instante) && !instante.Before(noAntes)
}

func (v VersionGobernadaReglasBaremo) anteriorBorrador() VersionGobernadaReglasBaremo {
	v.revision = 1
	v.estado = EstadoReglasBaremoBorrador
	v.publicacion = nil
	v.activacion = nil
	v.terminal = nil
	return v
}

func (v VersionGobernadaReglasBaremo) anteriorPublicada() VersionGobernadaReglasBaremo {
	v.revision = 2
	v.estado = EstadoReglasBaremoPublicada
	v.activacion = nil
	v.terminal = nil
	return v
}

func (v VersionGobernadaReglasBaremo) anteriorActiva() VersionGobernadaReglasBaremo {
	v.revision = 3
	v.estado = EstadoReglasBaremoActiva
	v.terminal = nil
	return v
}

func (a actoPublicacionReglasBaremo) validarPara(
	version VersionGobernadaReglasBaremo,
	instante time.Time,
) error {
	if !a.instante.Equal(instante) {
		return ErrGobiernoEvidenciaInvalida
	}
	return a.aprobacion.validarPara(version, instante)
}

func (a AtestacionAprobacionFirmadaReglasBaremo) validarPara(
	version VersionGobernadaReglasBaremo,
	instante time.Time,
) error {
	vinculo, err := version.VinculoEstado()
	if err != nil || a.validar() != nil || !vinculosEstadoIguales(a.vinculo, vinculo) ||
		a.firmadaEn.Before(version.creadaEn) || a.verificadaEn.After(instante) ||
		instante.Before(a.verificadaEn) || !instante.Before(a.validaHasta) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

func (a AtestacionDependenciasVigentesReglasBaremo) validarPara(
	version VersionGobernadaReglasBaremo,
	instante time.Time,
) error {
	vinculo, errVinculo := version.VinculoEstado()
	dependencias, errDependencias := version.DependenciasContenido()
	identidad := version.conjunto.Identidad()
	// El conjunto solo conoce la familia opaca de convocatoria para evitar un
	// ciclo de huellas. La atestacion conserva su referencia, version y huella
	// exactas; el verificador externo certifica que esa convocatoria enlaza
	// precisamente este VinculoEstado de reglas.
	if errVinculo != nil || errDependencias != nil || a.validar() != nil ||
		!vinculosEstadoIguales(a.vinculo, vinculo) ||
		a.convocatoria.referencia != identidad.ConvocatoriaRef() ||
		!referenciasVersionadasIguales(a.bases, version.conjunto.Bases()) ||
		!listasReferenciasVersionadasIguales(a.dependencias, dependencias) ||
		a.verificadaEn.Before(version.ultimaActuacionEn()) || a.verificadaEn.After(instante) ||
		!instante.Before(a.validaHasta) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

func (a AtestacionAutoridadReglasBaremo) validarPara(
	version VersionGobernadaReglasBaremo,
	accion AccionGobiernoReglasBaremo,
	actorRef string,
	relacionada *ReferenciaVersionada,
	instante time.Time,
) error {
	vinculo, err := version.VinculoEstado()
	if err != nil || a.validar() != nil || a.accion != accion ||
		a.principalRef != actorRef || !vinculosEstadoIguales(a.vinculo, vinculo) ||
		!referenciasOpcionalesIguales(a.relacionada, relacionada) ||
		a.emitidaEn.Before(version.ultimaActuacionEn()) || a.emitidaEn.After(instante) ||
		!instante.Before(a.validaHasta) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

func vinculosEstadoIguales(a, b VinculoEstadoReglasBaremo) bool {
	return referenciasVersionadasIguales(a.contenido, b.contenido) &&
		a.revision == b.revision && a.huellaEstadoSHA256 == b.huellaEstadoSHA256
}

func referenciasOpcionalesIguales(a, b *ReferenciaVersionada) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return referenciasVersionadasIguales(*a, *b)
}

func listasReferenciasVersionadasIguales(a, b []ReferenciaVersionada) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if !referenciasVersionadasIguales(a[indice], b[indice]) {
			return false
		}
	}
	return true
}

func (a AtestacionAprobacionFirmadaReglasBaremo) clonar() AtestacionAprobacionFirmadaReglasBaremo {
	a.firmantes = append([]string(nil), a.firmantes...)
	return a
}

func (a AtestacionDependenciasVigentesReglasBaremo) clonar() AtestacionDependenciasVigentesReglasBaremo {
	a.dependencias = append([]ReferenciaVersionada(nil), a.dependencias...)
	return a
}

func (a AtestacionAutoridadReglasBaremo) clonar() AtestacionAutoridadReglasBaremo {
	a.relacionada = clonarReferenciaVersionada(a.relacionada)
	return a
}

func (v VersionGobernadaReglasBaremo) Clonar() (VersionGobernadaReglasBaremo, error) {
	if v.Validar() != nil {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	conjunto, err := clonarConjuntoGobiernoReglasBaremo(v.conjunto)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, ErrGobiernoInvarianteQuebrada
	}
	v.conjunto = conjunto
	if v.publicacion != nil {
		clon := *v.publicacion
		clon.aprobacion = clon.aprobacion.clonar()
		v.publicacion = &clon
	}
	if v.activacion != nil {
		clon := *v.activacion
		clon.dependencias = clon.dependencias.clonar()
		v.activacion = &clon
	}
	if v.terminal != nil {
		clon := *v.terminal
		clon.autoridad = clon.autoridad.clonar()
		v.terminal = &clon
	}
	return v, nil
}

func clonarConjuntoGobiernoReglasBaremo(
	conjunto ConjuntoReglasBaremo,
) (ConjuntoReglasBaremo, error) {
	contenido, err := conjunto.RepresentacionCanonica()
	if err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	return RestaurarConjuntoReglasBaremo(contenido)
}

func (v VersionGobernadaReglasBaremo) RepresentacionCanonica() ([]byte, error) {
	if v.Validar() != nil {
		return nil, ErrGobiernoInvarianteQuebrada
	}
	referencia, err := v.ReferenciaContenido()
	if err != nil {
		return nil, err
	}
	material := materialGobiernoReglasBaremo{
		Esquema: esquemaGobiernoReglasBaremo, Contenido: v.conjunto,
		ReferenciaContenido: materialReferenciaGobierno(referencia),
		Revision:            v.revision, Estado: v.estado, CreadaPor: v.creadaPor,
		CreadaEn: v.creadaEn, MotivoCreacion: materialMotivoGobierno(v.motivoCreacion),
	}
	if v.publicacion != nil {
		material.Publicacion = materialPublicacionGobierno(v.publicacion)
	}
	if v.activacion != nil {
		material.Activacion = materialActivacionGobierno(v.activacion)
	}
	if v.terminal != nil {
		material.Terminal = materialTerminalGobierno(v.terminal)
	}
	contenido, err := json.Marshal(material)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesGobiernoReglasBaremo {
		return nil, ErrGobiernoInvarianteQuebrada
	}
	return append([]byte(nil), contenido...), nil
}

func (v VersionGobernadaReglasBaremo) HuellaSHA256() (string, error) {
	contenido, err := v.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

type materialGobiernoReglasBaremo struct {
	Esquema             string                             `json:"esquema"`
	Contenido           ConjuntoReglasBaremo               `json:"contenido"`
	ReferenciaContenido materialReferenciaGobiernoReglas   `json:"referencia_contenido"`
	Revision            uint64                             `json:"revision"`
	Estado              EstadoGobiernoReglasBaremo         `json:"estado"`
	CreadaPor           string                             `json:"creada_por"`
	CreadaEn            time.Time                          `json:"creada_en"`
	MotivoCreacion      materialMotivoGobiernoReglas       `json:"motivo_creacion"`
	Publicacion         *materialPublicacionGobiernoReglas `json:"publicacion,omitempty"`
	Activacion          *materialActivacionGobiernoReglas  `json:"activacion,omitempty"`
	Terminal            *materialTerminalGobiernoReglas    `json:"terminal,omitempty"`
}

type materialReferenciaGobiernoReglas struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialMotivoGobiernoReglas struct {
	Catalogo materialReferenciaGobiernoReglas `json:"catalogo"`
	Clave    string                           `json:"clave"`
}

type materialVinculoGobiernoReglas struct {
	Contenido          materialReferenciaGobiernoReglas `json:"contenido"`
	Revision           uint64                           `json:"revision"`
	HuellaEstadoSHA256 string                           `json:"huella_estado_sha256"`
}

type materialAprobacionGobiernoReglas struct {
	Atestacion    materialReferenciaGobiernoReglas `json:"atestacion"`
	Vinculo       materialVinculoGobiernoReglas    `json:"vinculo"`
	Firma         materialReferenciaGobiernoReglas `json:"firma"`
	PoliticaFirma materialReferenciaGobiernoReglas `json:"politica_firma"`
	Firmantes     []string                         `json:"firmantes"`
	FirmadaEn     time.Time                        `json:"firmada_en"`
	VerificadaEn  time.Time                        `json:"verificada_en"`
	ValidaHasta   time.Time                        `json:"valida_hasta"`
}

type materialDependenciasGobiernoReglas struct {
	Atestacion     materialReferenciaGobiernoReglas   `json:"atestacion"`
	Vinculo        materialVinculoGobiernoReglas      `json:"vinculo"`
	Convocatoria   materialReferenciaGobiernoReglas   `json:"convocatoria"`
	Bases          materialReferenciaGobiernoReglas   `json:"bases"`
	Dependencias   []materialReferenciaGobiernoReglas `json:"dependencias"`
	VerificadorRef string                             `json:"verificador_ref"`
	VerificadaEn   time.Time                          `json:"verificada_en"`
	ValidaHasta    time.Time                          `json:"valida_hasta"`
}

type materialAutoridadGobiernoReglas struct {
	Atestacion   materialReferenciaGobiernoReglas  `json:"atestacion"`
	Vinculo      materialVinculoGobiernoReglas     `json:"vinculo"`
	Accion       AccionGobiernoReglasBaremo        `json:"accion"`
	PrincipalRef string                            `json:"principal_ref"`
	Relacionada  *materialReferenciaGobiernoReglas `json:"relacionada,omitempty"`
	EmitidaEn    time.Time                         `json:"emitida_en"`
	ValidaHasta  time.Time                         `json:"valida_hasta"`
}

type materialPublicacionGobiernoReglas struct {
	ActorRef   string                           `json:"actor_ref"`
	Motivo     materialMotivoGobiernoReglas     `json:"motivo"`
	Aprobacion materialAprobacionGobiernoReglas `json:"aprobacion"`
	Instante   time.Time                        `json:"instante"`
}

type materialActivacionGobiernoReglas struct {
	ActorRef     string                             `json:"actor_ref"`
	Motivo       materialMotivoGobiernoReglas       `json:"motivo"`
	Dependencias materialDependenciasGobiernoReglas `json:"dependencias"`
	Instante     time.Time                          `json:"instante"`
}

type materialTerminalGobiernoReglas struct {
	Accion    AccionGobiernoReglasBaremo      `json:"accion"`
	ActorRef  string                          `json:"actor_ref"`
	Motivo    materialMotivoGobiernoReglas    `json:"motivo"`
	Autoridad materialAutoridadGobiernoReglas `json:"autoridad"`
	Instante  time.Time                       `json:"instante"`
}

func materialReferenciaGobierno(r ReferenciaVersionada) materialReferenciaGobiernoReglas {
	return materialReferenciaGobiernoReglas{r.referencia, r.version, r.huellaSHA256}
}

func materialMotivoGobierno(m MotivoCatalogadoReglasBaremo) materialMotivoGobiernoReglas {
	return materialMotivoGobiernoReglas{materialReferenciaGobierno(m.catalogo), m.clave}
}

func materialVinculoGobierno(v VinculoEstadoReglasBaremo) materialVinculoGobiernoReglas {
	return materialVinculoGobiernoReglas{
		materialReferenciaGobierno(v.contenido), v.revision, v.huellaEstadoSHA256,
	}
}

func materialAprobacionGobierno(a AtestacionAprobacionFirmadaReglasBaremo) materialAprobacionGobiernoReglas {
	return materialAprobacionGobiernoReglas{
		Atestacion: materialReferenciaGobierno(a.atestacion), Vinculo: materialVinculoGobierno(a.vinculo),
		Firma: materialReferenciaGobierno(a.firma), PoliticaFirma: materialReferenciaGobierno(a.politicaFirma),
		Firmantes: append([]string(nil), a.firmantes...), FirmadaEn: a.firmadaEn,
		VerificadaEn: a.verificadaEn, ValidaHasta: a.validaHasta,
	}
}

func materialDependenciasGobierno(a AtestacionDependenciasVigentesReglasBaremo) materialDependenciasGobiernoReglas {
	dependencias := make([]materialReferenciaGobiernoReglas, len(a.dependencias))
	for indice := range a.dependencias {
		dependencias[indice] = materialReferenciaGobierno(a.dependencias[indice])
	}
	return materialDependenciasGobiernoReglas{
		Atestacion: materialReferenciaGobierno(a.atestacion), Vinculo: materialVinculoGobierno(a.vinculo),
		Convocatoria: materialReferenciaGobierno(a.convocatoria), Bases: materialReferenciaGobierno(a.bases),
		Dependencias: dependencias, VerificadorRef: a.verificadorRef,
		VerificadaEn: a.verificadaEn, ValidaHasta: a.validaHasta,
	}
}

func materialAutoridadGobierno(a AtestacionAutoridadReglasBaremo) materialAutoridadGobiernoReglas {
	material := materialAutoridadGobiernoReglas{
		Atestacion: materialReferenciaGobierno(a.atestacion), Vinculo: materialVinculoGobierno(a.vinculo),
		Accion: a.accion, PrincipalRef: a.principalRef, EmitidaEn: a.emitidaEn,
		ValidaHasta: a.validaHasta,
	}
	if a.relacionada != nil {
		relacionada := materialReferenciaGobierno(*a.relacionada)
		material.Relacionada = &relacionada
	}
	return material
}

func materialPublicacionGobierno(a *actoPublicacionReglasBaremo) *materialPublicacionGobiernoReglas {
	if a == nil {
		return nil
	}
	return &materialPublicacionGobiernoReglas{
		ActorRef: a.actorRef, Motivo: materialMotivoGobierno(a.motivo),
		Aprobacion: materialAprobacionGobierno(a.aprobacion), Instante: a.instante,
	}
}

func materialActivacionGobierno(a *actoActivacionReglasBaremo) *materialActivacionGobiernoReglas {
	if a == nil {
		return nil
	}
	return &materialActivacionGobiernoReglas{
		ActorRef: a.actorRef, Motivo: materialMotivoGobierno(a.motivo),
		Dependencias: materialDependenciasGobierno(a.dependencias), Instante: a.instante,
	}
}

func materialTerminalGobierno(a *actoTerminalReglasBaremo) *materialTerminalGobiernoReglas {
	if a == nil {
		return nil
	}
	return &materialTerminalGobiernoReglas{
		Accion: a.accion, ActorRef: a.actorRef, Motivo: materialMotivoGobierno(a.motivo),
		Autoridad: materialAutoridadGobierno(a.autoridad), Instante: a.instante,
	}
}
