package reglasbaremo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

func actoComunValido(
	actorRef string,
	motivo MotivoCatalogadoReglasBaremo,
	instante, noAntes time.Time,
) bool {
	return referenciaPersonaOpacaValida(actorRef) && motivo.validar() == nil &&
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
	return a.CoincideExactamenteCon(b)
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
