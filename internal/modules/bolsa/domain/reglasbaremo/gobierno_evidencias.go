package reglasbaremo

import (
	"sort"
	"time"
)

const (
	maximoFirmantesAprobacionReglasBaremo = 64
	maximoDependenciasReglasBaremo        = 1 + maximoSeccionesPorConjunto +
		maximoGruposConcurrencia + maximoReglasExperienciaConjunto + maximoCriteriosConjunto
)

// AccionGobiernoReglasBaremo forma parte del vinculo de autoridad. No es un
// texto configurable ni permite inventar nuevas transiciones desde datos.
type AccionGobiernoReglasBaremo string

const (
	AccionPublicarReglasBaremo  AccionGobiernoReglasBaremo = "publicar"
	AccionActivarReglasBaremo   AccionGobiernoReglasBaremo = "activar"
	AccionSustituirReglasBaremo AccionGobiernoReglasBaremo = "sustituir"
	AccionRetirarReglasBaremo   AccionGobiernoReglasBaremo = "retirar"
	AccionDescartarReglasBaremo AccionGobiernoReglasBaremo = "descartar"
)

func (a AccionGobiernoReglasBaremo) valida() bool {
	switch a {
	case AccionPublicarReglasBaremo, AccionActivarReglasBaremo,
		AccionSustituirReglasBaremo, AccionRetirarReglasBaremo,
		AccionDescartarReglasBaremo:
		return true
	default:
		return false
	}
}

// MotivoCatalogadoReglasBaremo fija tanto la version y huella del catalogo
// como la clave elegida. El dominio no conserva un motivo libre en las trazas.
type MotivoCatalogadoReglasBaremo struct {
	catalogo ReferenciaVersionada
	clave    string
}

func NuevoMotivoCatalogadoReglasBaremo(
	catalogo ReferenciaVersionada,
	clave string,
) (MotivoCatalogadoReglasBaremo, error) {
	motivo := MotivoCatalogadoReglasBaremo{catalogo: catalogo, clave: clave}
	if motivo.validar() != nil {
		return MotivoCatalogadoReglasBaremo{}, ErrGobiernoValorInvalido
	}
	return motivo, nil
}

func (m MotivoCatalogadoReglasBaremo) Catalogo() ReferenciaVersionada { return m.catalogo }
func (m MotivoCatalogadoReglasBaremo) Clave() string                  { return m.clave }

func (m MotivoCatalogadoReglasBaremo) validar() error {
	if m.catalogo.validar("motivo.catalogo") != nil || !claveValida(m.clave) {
		return ErrGobiernoValorInvalido
	}
	return nil
}

// VinculoEstadoReglasBaremo impide aplicar una atestacion a otra revision,
// otro contenido o un estado materialmente distinto.
type VinculoEstadoReglasBaremo struct {
	contenido          ReferenciaVersionada
	revision           uint64
	huellaEstadoSHA256 string
}

func NuevoVinculoEstadoReglasBaremo(
	contenido ReferenciaVersionada,
	revision uint64,
	huellaEstadoSHA256 string,
) (VinculoEstadoReglasBaremo, error) {
	vinculo := VinculoEstadoReglasBaremo{
		contenido: contenido, revision: revision,
		huellaEstadoSHA256: huellaEstadoSHA256,
	}
	if vinculo.validar() != nil {
		return VinculoEstadoReglasBaremo{}, ErrGobiernoVinculoInexacto
	}
	return vinculo, nil
}

func (v VinculoEstadoReglasBaremo) Contenido() ReferenciaVersionada { return v.contenido }
func (v VinculoEstadoReglasBaremo) Revision() uint64                { return v.revision }
func (v VinculoEstadoReglasBaremo) HuellaEstadoSHA256() string      { return v.huellaEstadoSHA256 }

func (v VinculoEstadoReglasBaremo) validar() error {
	if v.contenido.validar("vinculo.contenido") != nil || v.revision == 0 ||
		v.revision > maximoVersion || !huellaSHA256Valida(v.huellaEstadoSHA256) {
		return ErrGobiernoVinculoInexacto
	}
	return nil
}

type DatosAtestacionAprobacionFirmadaReglasBaremo struct {
	Atestacion    ReferenciaVersionada
	Vinculo       VinculoEstadoReglasBaremo
	Firma         ReferenciaVersionada
	PoliticaFirma ReferenciaVersionada
	Firmantes     []string
	FirmadaEn     time.Time
	VerificadaEn  time.Time
	ValidaHasta   time.Time
}

// AtestacionAprobacionFirmadaReglasBaremo es una afirmacion estructurada de un
// verificador externo. El dominio liga referencias, huellas y tiempos, pero no
// verifica certificados ni convierte una huella SHA-256 en una firma.
type AtestacionAprobacionFirmadaReglasBaremo struct {
	atestacion    ReferenciaVersionada
	vinculo       VinculoEstadoReglasBaremo
	firma         ReferenciaVersionada
	politicaFirma ReferenciaVersionada
	firmantes     []string
	firmadaEn     time.Time
	verificadaEn  time.Time
	validaHasta   time.Time
}

func NuevaAtestacionAprobacionFirmadaReglasBaremo(
	datos DatosAtestacionAprobacionFirmadaReglasBaremo,
) (AtestacionAprobacionFirmadaReglasBaremo, error) {
	if len(datos.Firmantes) == 0 || len(datos.Firmantes) > maximoFirmantesAprobacionReglasBaremo {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	atestacion := AtestacionAprobacionFirmadaReglasBaremo{
		atestacion: datos.Atestacion, vinculo: datos.Vinculo,
		firma: datos.Firma, politicaFirma: datos.PoliticaFirma,
		firmantes: append([]string(nil), datos.Firmantes...),
		firmadaEn: datos.FirmadaEn, verificadaEn: datos.VerificadaEn,
		validaHasta: datos.ValidaHasta,
	}
	if atestacion.validar() != nil {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	sort.Strings(atestacion.firmantes)
	return atestacion, nil
}

func (a AtestacionAprobacionFirmadaReglasBaremo) Atestacion() ReferenciaVersionada {
	return a.atestacion
}
func (a AtestacionAprobacionFirmadaReglasBaremo) Vinculo() VinculoEstadoReglasBaremo {
	return a.vinculo
}
func (a AtestacionAprobacionFirmadaReglasBaremo) Firma() ReferenciaVersionada {
	return a.firma
}
func (a AtestacionAprobacionFirmadaReglasBaremo) PoliticaFirma() ReferenciaVersionada {
	return a.politicaFirma
}
func (a AtestacionAprobacionFirmadaReglasBaremo) Firmantes() []string {
	return append([]string(nil), a.firmantes...)
}
func (a AtestacionAprobacionFirmadaReglasBaremo) FirmadaEn() time.Time    { return a.firmadaEn }
func (a AtestacionAprobacionFirmadaReglasBaremo) VerificadaEn() time.Time { return a.verificadaEn }
func (a AtestacionAprobacionFirmadaReglasBaremo) ValidaHasta() time.Time  { return a.validaHasta }

func (a AtestacionAprobacionFirmadaReglasBaremo) validar() error {
	if a.atestacion.validar("aprobacion.atestacion") != nil || a.vinculo.validar() != nil ||
		a.firma.validar("aprobacion.firma") != nil ||
		a.politicaFirma.validar("aprobacion.politica_firma") != nil ||
		!referenciasVersionadasDistintas(
			a.atestacion, a.vinculo.contenido, a.firma, a.politicaFirma,
		) || len(a.firmantes) == 0 || len(a.firmantes) > maximoFirmantesAprobacionReglasBaremo ||
		!referenciasOpacasUnicas(a.firmantes) ||
		!instanteGobiernoReglasBaremoValido(a.firmadaEn) ||
		!instanteGobiernoReglasBaremoValido(a.verificadaEn) ||
		!instanteGobiernoReglasBaremoValido(a.validaHasta) ||
		a.verificadaEn.Before(a.firmadaEn) || !a.validaHasta.After(a.verificadaEn) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

type DatosAtestacionDependenciasVigentesReglasBaremo struct {
	Atestacion     ReferenciaVersionada
	Vinculo        VinculoEstadoReglasBaremo
	Convocatoria   ReferenciaVersionada
	Bases          ReferenciaVersionada
	Dependencias   []ReferenciaVersionada
	VerificadorRef string
	VerificadaEn   time.Time
	ValidaHasta    time.Time
}

// AtestacionDependenciasVigentesReglasBaremo liga la activacion a la version
// exacta de convocatoria, bases y todas las referencias del contenido.
type AtestacionDependenciasVigentesReglasBaremo struct {
	atestacion     ReferenciaVersionada
	vinculo        VinculoEstadoReglasBaremo
	convocatoria   ReferenciaVersionada
	bases          ReferenciaVersionada
	dependencias   []ReferenciaVersionada
	verificadorRef string
	verificadaEn   time.Time
	validaHasta    time.Time
}

func NuevaAtestacionDependenciasVigentesReglasBaremo(
	datos DatosAtestacionDependenciasVigentesReglasBaremo,
) (AtestacionDependenciasVigentesReglasBaremo, error) {
	if len(datos.Dependencias) == 0 || len(datos.Dependencias) > maximoDependenciasReglasBaremo {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	atestacion := AtestacionDependenciasVigentesReglasBaremo{
		atestacion: datos.Atestacion, vinculo: datos.Vinculo,
		convocatoria: datos.Convocatoria, bases: datos.Bases,
		dependencias:   append([]ReferenciaVersionada(nil), datos.Dependencias...),
		verificadorRef: datos.VerificadorRef, verificadaEn: datos.VerificadaEn,
		validaHasta: datos.ValidaHasta,
	}
	sortReferenciasVersionadas(atestacion.dependencias)
	if atestacion.validar() != nil {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	return atestacion, nil
}

func (a AtestacionDependenciasVigentesReglasBaremo) Atestacion() ReferenciaVersionada {
	return a.atestacion
}
func (a AtestacionDependenciasVigentesReglasBaremo) Vinculo() VinculoEstadoReglasBaremo {
	return a.vinculo
}
func (a AtestacionDependenciasVigentesReglasBaremo) Convocatoria() ReferenciaVersionada {
	return a.convocatoria
}
func (a AtestacionDependenciasVigentesReglasBaremo) Bases() ReferenciaVersionada {
	return a.bases
}
func (a AtestacionDependenciasVigentesReglasBaremo) Dependencias() []ReferenciaVersionada {
	return append([]ReferenciaVersionada(nil), a.dependencias...)
}
func (a AtestacionDependenciasVigentesReglasBaremo) VerificadorRef() string {
	return a.verificadorRef
}
func (a AtestacionDependenciasVigentesReglasBaremo) VerificadaEn() time.Time { return a.verificadaEn }
func (a AtestacionDependenciasVigentesReglasBaremo) ValidaHasta() time.Time  { return a.validaHasta }

func (a AtestacionDependenciasVigentesReglasBaremo) validar() error {
	if a.atestacion.validar("dependencias.atestacion") != nil || a.vinculo.validar() != nil ||
		a.convocatoria.validar("dependencias.convocatoria") != nil ||
		a.bases.validar("dependencias.bases") != nil ||
		!referenciaValida(a.verificadorRef) || len(a.dependencias) == 0 ||
		!referenciasVersionadasUnicasYValidas(a.dependencias) ||
		!contieneReferenciaVersionada(a.dependencias, a.bases) ||
		!instanteGobiernoReglasBaremoValido(a.verificadaEn) ||
		!instanteGobiernoReglasBaremoValido(a.validaHasta) ||
		!a.validaHasta.After(a.verificadaEn) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

type DatosAtestacionAutoridadReglasBaremo struct {
	Atestacion   ReferenciaVersionada
	Vinculo      VinculoEstadoReglasBaremo
	Accion       AccionGobiernoReglasBaremo
	PrincipalRef string
	Relacionada  *ReferenciaVersionada
	EmitidaEn    time.Time
	ValidaHasta  time.Time
}

// AtestacionAutoridadReglasBaremo demuestra estructuralmente que una autoridad
// externa autorizo una transicion terminal exacta. Su autenticidad pertenece
// al adaptador de verificacion, no a este valor puro.
type AtestacionAutoridadReglasBaremo struct {
	atestacion   ReferenciaVersionada
	vinculo      VinculoEstadoReglasBaremo
	accion       AccionGobiernoReglasBaremo
	principalRef string
	relacionada  *ReferenciaVersionada
	emitidaEn    time.Time
	validaHasta  time.Time
}

func NuevaAtestacionAutoridadReglasBaremo(
	datos DatosAtestacionAutoridadReglasBaremo,
) (AtestacionAutoridadReglasBaremo, error) {
	atestacion := AtestacionAutoridadReglasBaremo{
		atestacion: datos.Atestacion, vinculo: datos.Vinculo,
		accion: datos.Accion, principalRef: datos.PrincipalRef,
		relacionada: clonarReferenciaVersionada(datos.Relacionada),
		emitidaEn:   datos.EmitidaEn, validaHasta: datos.ValidaHasta,
	}
	if atestacion.validar() != nil {
		return AtestacionAutoridadReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	return atestacion, nil
}

func (a AtestacionAutoridadReglasBaremo) Atestacion() ReferenciaVersionada {
	return a.atestacion
}
func (a AtestacionAutoridadReglasBaremo) Vinculo() VinculoEstadoReglasBaremo {
	return a.vinculo
}
func (a AtestacionAutoridadReglasBaremo) Accion() AccionGobiernoReglasBaremo { return a.accion }
func (a AtestacionAutoridadReglasBaremo) PrincipalRef() string               { return a.principalRef }
func (a AtestacionAutoridadReglasBaremo) Relacionada() (ReferenciaVersionada, bool) {
	if a.relacionada == nil {
		return ReferenciaVersionada{}, false
	}
	return *a.relacionada, true
}
func (a AtestacionAutoridadReglasBaremo) EmitidaEn() time.Time   { return a.emitidaEn }
func (a AtestacionAutoridadReglasBaremo) ValidaHasta() time.Time { return a.validaHasta }

func (a AtestacionAutoridadReglasBaremo) validar() error {
	relacionadaRequerida := a.accion == AccionSustituirReglasBaremo
	if a.atestacion.validar("autoridad.atestacion") != nil || a.vinculo.validar() != nil ||
		(a.accion != AccionSustituirReglasBaremo && a.accion != AccionRetirarReglasBaremo &&
			a.accion != AccionDescartarReglasBaremo) || !referenciaValida(a.principalRef) ||
		(a.relacionada != nil) != relacionadaRequerida ||
		(a.relacionada != nil && (a.relacionada.validar("autoridad.relacionada") != nil ||
			referenciasVersionadasIguales(*a.relacionada, a.vinculo.contenido))) ||
		!instanteGobiernoReglasBaremoValido(a.emitidaEn) ||
		!instanteGobiernoReglasBaremoValido(a.validaHasta) ||
		!a.validaHasta.After(a.emitidaEn) {
		return ErrGobiernoEvidenciaInvalida
	}
	return nil
}

func instanteGobiernoReglasBaremoValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func referenciasVersionadasIguales(a, b ReferenciaVersionada) bool {
	return a.referencia == b.referencia && a.version == b.version &&
		a.huellaSHA256 == b.huellaSHA256
}

func referenciasVersionadasDistintas(referencias ...ReferenciaVersionada) bool {
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		clave := referencia.referencia + "\x00" + referencia.huellaSHA256
		if _, existe := vistas[clave]; existe {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
}

func referenciasOpacasUnicas(referencias []string) bool {
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if !referenciaValida(referencia) {
			return false
		}
		if _, existe := vistas[referencia]; existe {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}

func referenciasVersionadasUnicasYValidas(referencias []ReferenciaVersionada) bool {
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if referencia.validar("dependencia") != nil {
			return false
		}
		if _, existe := vistas[referencia.referencia]; existe {
			return false
		}
		vistas[referencia.referencia] = struct{}{}
	}
	return true
}

func contieneReferenciaVersionada(referencias []ReferenciaVersionada, buscada ReferenciaVersionada) bool {
	for _, referencia := range referencias {
		if referenciasVersionadasIguales(referencia, buscada) {
			return true
		}
	}
	return false
}

func sortReferenciasVersionadas(referencias []ReferenciaVersionada) {
	sort.Slice(referencias, func(i, j int) bool {
		if referencias[i].referencia != referencias[j].referencia {
			return referencias[i].referencia < referencias[j].referencia
		}
		if referencias[i].version != referencias[j].version {
			return referencias[i].version < referencias[j].version
		}
		return referencias[i].huellaSHA256 < referencias[j].huellaSHA256
	})
}

func clonarReferenciaVersionada(origen *ReferenciaVersionada) *ReferenciaVersionada {
	if origen == nil {
		return nil
	}
	clon := *origen
	return &clon
}
