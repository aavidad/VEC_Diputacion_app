package application

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrAdmisionGarantiaActoDenegada = errors.New("admision_garantia_acto_denegada")

type errorAdmisionGarantiaActo struct{ causa error }

func (e *errorAdmisionGarantiaActo) Error() string { return ErrAdmisionGarantiaActoDenegada.Error() }
func (e *errorAdmisionGarantiaActo) Unwrap() []error {
	return []error{ErrAdmisionGarantiaActoDenegada, e.causa}
}

func falloAdmisionGarantiaActo(causa error) error {
	if causa == nil {
		return ErrAdmisionGarantiaActoDenegada
	}
	return &errorAdmisionGarantiaActo{causa: causa}
}

const (
	EntornoAdmisionFirmaDesarrollo = "desarrollo"
	accionAdmisionFirmaVec         = "contratacion_temporal.documento.firma_vec.registrar"
	audienciaAdmisionFirmaVec      = "vec_contratacion_temporal.firma_vec.v2"
	moduloAdmisionFirmaVec         = "contratacion_temporal"
	tipoAdmisionFirmaVec           = "firma_vec_documento_contratacion_temporal"
	finalidadAdmisionFirmaVec      = "gestionar_contratacion_temporal"
)

// DescriptorAdmisionGarantiaActo identifica sólo el acto V2 de registro VEC.
// El llamador lo obtiene de su ruta montada, nunca del cuerpo HTTP.
type DescriptorAdmisionGarantiaActo struct {
	Accion, Audiencia, ModuloID, TipoRecurso, Finalidad string
	RecursoRef, MaterialHuellaSHA256                    string
}

func (d DescriptorAdmisionGarantiaActo) exactoFirmaVec() bool {
	return d.Accion == accionAdmisionFirmaVec && d.Audiencia == audienciaAdmisionFirmaVec &&
		d.ModuloID == moduloAdmisionFirmaVec && d.TipoRecurso == tipoAdmisionFirmaVec &&
		d.Finalidad == finalidadAdmisionFirmaVec &&
		referenciaAdmisionValida(d.RecursoRef, "operacion-firma-vec-ct:") &&
		huellaAdmisionValida(d.MaterialHuellaSHA256)
}

// PoliticaPrivadaAdmisionFirmaDesarrollo es una aprobación de configuración
// privada. Su referencia y huella se fijan fuera de HTTP; publicar el rol V3
// y asignarlo mediante Administración sigue siendo una dependencia separada.
type PoliticaPrivadaAdmisionFirmaDesarrollo struct {
	Referencia, HuellaSHA256                              string
	Version                                               uint64
	RetiradaEn                                            time.Time
	PoliticaAutenticacionRef, PoliticaAutenticacionSHA256 string
	RolVersionRef, RolSHA256                              string
	ControlRevision                                       uint64
	ControlSHA256                                         string
}

func (p PoliticaPrivadaAdmisionFirmaDesarrollo) valida(ahora time.Time) bool {
	if p.Version == 0 || p.ControlRevision == 0 || !referenciaAdmisionValida(p.Referencia, "politica:") ||
		!strings.HasSuffix(p.Referencia, ":v"+strconv.FormatUint(p.Version, 10)) ||
		!referenciaAdmisionValida(p.PoliticaAutenticacionRef, "pga_") ||
		!referenciaAdmisionValida(p.RolVersionRef, "rol:") ||
		!huellaAdmisionValida(p.HuellaSHA256) ||
		!huellaAdmisionValida(p.PoliticaAutenticacionSHA256) ||
		!huellaAdmisionValida(p.RolSHA256) || !huellaAdmisionValida(p.ControlSHA256) ||
		p.RetiradaEn.IsZero() || p.RetiradaEn.Location() != time.UTC ||
		p.RetiradaEn.Nanosecond() != 0 || !ahora.Before(p.RetiradaEn) {
		return false
	}
	return true
}

func referenciaAdmisionValida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+8 || len(valor) > 512 || !strings.HasPrefix(valor, prefijo) {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
			c != '_' && c != '-' && c != ':' {
			return false
		}
	}
	return !strings.Contains(valor, "*")
}

func huellaAdmisionValida(valor string) bool {
	if len(valor) != 64 || strings.Trim(valor, "0") == "" {
		return false
	}
	for _, c := range valor {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

type ConfiguracionAdmisionGarantiaFirmaVecDesarrollo struct {
	EntornoConfiable string
	Politica         PoliticaPrivadaAdmisionFirmaDesarrollo
	Reloj            core.RelojVinculoAutenticacionActorV2
}

// AdmisionGarantiaFirmaVecDesarrollo sólo decide si la garantía REAL permite
// presentar este acto al PDP V3. No concede la acción ni registra un efecto.
type AdmisionGarantiaFirmaVecDesarrollo struct {
	politica PoliticaPrivadaAdmisionFirmaDesarrollo
	reloj    core.RelojVinculoAutenticacionActorV2
}

func NuevaAdmisionGarantiaFirmaVecDesarrollo(c ConfiguracionAdmisionGarantiaFirmaVecDesarrollo) (*AdmisionGarantiaFirmaVecDesarrollo, error) {
	if c.EntornoConfiable != EntornoAdmisionFirmaDesarrollo || nuloAdmision(c.Reloj) ||
		!c.Politica.valida(c.Reloj.Ahora().UTC().Truncate(time.Microsecond)) {
		return nil, ErrAdmisionGarantiaActoDenegada
	}
	return &AdmisionGarantiaFirmaVecDesarrollo{politica: c.Politica, reloj: c.Reloj}, nil
}

func nuloAdmision(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

// ResumenAdmisionGarantiaActo conserva la excepción y la garantía observada
// para auditoría. Es informativo: sólo EvidenciaAdmisionGarantiaActo puede
// presentarse a la autoridad consumidora junto a V3.
type ResumenAdmisionGarantiaActo struct {
	GarantiaReal                                          core.AuthAssurance
	PoliticaRef, PoliticaSHA256                           string
	PoliticaVersion                                       uint64
	PoliticaAutenticacionRef, PoliticaAutenticacionSHA256 string
	RolVersionRef, RolSHA256                              string
	RecursoRef, MaterialHuellaSHA256                      string
	ControlRevision                                       uint64
	ControlSHA256                                         string
	AdmitidaEn, RetiradaEn, VigenteHasta                  time.Time
}

type datosEvidenciaAdmisionGarantiaActo struct {
	resumen                          ResumenAdmisionGarantiaActo
	descriptor                       DescriptorAdmisionGarantiaActo
	personaRef, cuentaRef, perfilRef string
	autenticacionRef, sesionRef      string
	contextoRegistroRef, contextoSHA string
	asignacionRef, asignacionSHA     string
	catalogoRevision                 uint64
	catalogoSHA                      string
}

// EvidenciaAdmisionGarantiaActo no admite literal válido, deserialización ni
// reproducción desde HTTP. Su validez debe cotejarse con el mismo admisor y
// una captura V3 fresca antes del efecto.
type EvidenciaAdmisionGarantiaActo struct {
	datos *datosEvidenciaAdmisionGarantiaActo
}

const evidenciaAdmisionRedactada = "[EVIDENCIA ADMISION GARANTIA CONFIDENCIAL]"

func (EvidenciaAdmisionGarantiaActo) String() string   { return evidenciaAdmisionRedactada }
func (EvidenciaAdmisionGarantiaActo) GoString() string { return evidenciaAdmisionRedactada }
func (EvidenciaAdmisionGarantiaActo) LogValue() slog.Value {
	return slog.StringValue(evidenciaAdmisionRedactada)
}
func (EvidenciaAdmisionGarantiaActo) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(evidenciaAdmisionRedactada))
}
func (EvidenciaAdmisionGarantiaActo) MarshalJSON() ([]byte, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalJSON([]byte) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (EvidenciaAdmisionGarantiaActo) MarshalText() ([]byte, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalText([]byte) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (EvidenciaAdmisionGarantiaActo) MarshalBinary() ([]byte, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalBinary([]byte) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (EvidenciaAdmisionGarantiaActo) GobEncode() ([]byte, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) GobDecode([]byte) error { return ErrAdmisionGarantiaActoDenegada }
func (EvidenciaAdmisionGarantiaActo) MarshalCBOR() ([]byte, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalCBOR([]byte) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (EvidenciaAdmisionGarantiaActo) MarshalYAML() (any, error) {
	return nil, ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalYAML(func(any) error) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (EvidenciaAdmisionGarantiaActo) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrAdmisionGarantiaActoDenegada
}
func (*EvidenciaAdmisionGarantiaActo) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrAdmisionGarantiaActoDenegada
}

func (e EvidenciaAdmisionGarantiaActo) Resumen() (ResumenAdmisionGarantiaActo, error) {
	if e.datos == nil || e.datos.resumen.GarantiaReal != core.AuthAssuranceSubstantial {
		return ResumenAdmisionGarantiaActo{}, ErrAdmisionGarantiaActoDenegada
	}
	return e.datos.resumen, nil
}

// ExigirVentanaDecisionV3 impide que una decisión emitida para este acto dure
// más que la admisión temporal. V3 conserva su evaluación y CAS propios.
func (e EvidenciaAdmisionGarantiaActo) ExigirVentanaDecisionV3(emitidaEn, validaHasta time.Time) error {
	if e.datos == nil || !instanteAdmisionCanonico(emitidaEn) || !instanteAdmisionCanonico(validaHasta) ||
		emitidaEn.Before(e.datos.resumen.AdmitidaEn) || !validaHasta.After(emitidaEn) ||
		validaHasta.After(e.datos.resumen.VigenteHasta) {
		return ErrAdmisionGarantiaActoDenegada
	}
	return nil
}

func instanteAdmisionCanonico(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}

func (a *AdmisionGarantiaFirmaVecDesarrollo) Admitir(ctx context.Context,
	vinculo core.VinculoAutenticacionActorV2, resultado core.ResultadoContextoActorRegistradoV2,
	snapshot core.InstantaneaAutorizacion, descriptor DescriptorAdmisionGarantiaActo,
) (EvidenciaAdmisionGarantiaActo, error) {
	var vacia EvidenciaAdmisionGarantiaActo
	if a == nil || nuloAdmision(a.reloj) || ctx == nil || !descriptor.exactoFirmaVec() {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, falloAdmisionGarantiaActo(err)
	}
	ahora := a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !a.politica.valida(ahora) || vinculo.ValidarPara(resultado) != nil ||
		!vinculo.VigenteEn(ahora, resultado) || snapshot.Validar() != nil {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	d, err := vinculo.Datos()
	if err != nil || d.CuentaRef == "" || d.CuentaOrdinariaRef != d.CuentaRef || d.CuentaPrivilegiada ||
		d.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		d.MetodoObservado != core.AuthMethodCertificate ||
		d.GarantiaObservada != core.AuthAssuranceSubstantial ||
		d.PoliticaGarantiaRef != a.politica.PoliticaAutenticacionRef ||
		d.PoliticaGarantiaHuellaSHA256 != a.politica.PoliticaAutenticacionSHA256 ||
		snapshot.AsignacionPerfil.PrincipalID != d.PrincipalID ||
		snapshot.AsignacionPerfil.PerfilActivoRef != d.PerfilActivoRef ||
		!snapshot.AsignacionPerfil.VigenteEn(ahora) ||
		snapshot.AsignacionPerfil.VersionRolRef != a.politica.RolVersionRef ||
		snapshot.VersionRol.Referencia() != a.politica.RolVersionRef ||
		snapshot.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		ahora.Before(snapshot.VersionRol.PublicadaEn) ||
		snapshot.ControlVigenciaVersionRol.VersionRolRef != a.politica.RolVersionRef ||
		snapshot.ControlVigenciaVersionRol.Revision != a.politica.ControlRevision ||
		snapshot.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		ahora.Before(snapshot.ControlVigenciaVersionRol.ActualizadoEn) ||
		len(snapshot.VersionRol.Concesiones) != 1 {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	concesion := snapshot.VersionRol.Concesiones[0]
	if concesion.Accion != descriptor.Accion || concesion.ModuloID != descriptor.ModuloID ||
		concesion.TipoRecurso != descriptor.TipoRecurso ||
		len(concesion.Finalidades) != 1 || concesion.Finalidades[0] != descriptor.Finalidad ||
		concesion.GarantiaMinima != core.AuthAssuranceSubstantial {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	rolSHA, err := snapshot.VersionRol.HuellaSHA256()
	if err != nil || rolSHA != a.politica.RolSHA256 {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	controlSHA, err := snapshot.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil || controlSHA != a.politica.ControlSHA256 {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	asignacionSHA, err := snapshot.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return vacia, ErrAdmisionGarantiaActoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacia, falloAdmisionGarantiaActo(err)
	}
	vigenteHasta := minimoAdmision(d.SesionValidaHasta, resultado.Contexto.Instantanea.VigenteHasta,
		snapshot.AsignacionPerfil.VigenteHasta, a.politica.RetiradaEn)
	resumen := ResumenAdmisionGarantiaActo{
		GarantiaReal: core.AuthAssuranceSubstantial,
		PoliticaRef:  a.politica.Referencia, PoliticaSHA256: a.politica.HuellaSHA256,
		PoliticaVersion:             a.politica.Version,
		PoliticaAutenticacionRef:    a.politica.PoliticaAutenticacionRef,
		PoliticaAutenticacionSHA256: a.politica.PoliticaAutenticacionSHA256,
		RolVersionRef:               a.politica.RolVersionRef, RolSHA256: rolSHA,
		RecursoRef: descriptor.RecursoRef, MaterialHuellaSHA256: descriptor.MaterialHuellaSHA256,
		ControlRevision: a.politica.ControlRevision, ControlSHA256: controlSHA,
		AdmitidaEn: ahora, RetiradaEn: a.politica.RetiradaEn, VigenteHasta: vigenteHasta,
	}
	return EvidenciaAdmisionGarantiaActo{datos: &datosEvidenciaAdmisionGarantiaActo{
		resumen: resumen, descriptor: descriptor,
		personaRef: d.PrincipalID, cuentaRef: d.CuentaRef, perfilRef: d.PerfilActivoRef,
		autenticacionRef: d.AutenticacionRef, sesionRef: d.SesionRef,
		contextoRegistroRef: d.RegistroContextoRef, contextoSHA: d.ContextoActorHuellaSHA256,
		asignacionRef: snapshot.AsignacionPerfil.Referencia(), asignacionSHA: asignacionSHA,
		catalogoRevision: snapshot.RevisionCatalogoPoliticas,
		catalogoSHA:      snapshot.CatalogoPoliticasHuellaSHA256,
	}}, nil
}

// ValidarEvidencia exige otra captura viva. La comprobación no lee la base:
// el consumidor conserva la obligación de que V3/SQL revaliden antes del efecto.
func (a *AdmisionGarantiaFirmaVecDesarrollo) ValidarEvidencia(ctx context.Context, evidencia EvidenciaAdmisionGarantiaActo,
	vinculo core.VinculoAutenticacionActorV2, resultado core.ResultadoContextoActorRegistradoV2,
	snapshot core.InstantaneaAutorizacion, descriptor DescriptorAdmisionGarantiaActo,
) error {
	if evidencia.datos == nil {
		return ErrAdmisionGarantiaActoDenegada
	}
	actual, err := a.Admitir(ctx, vinculo, resultado, snapshot, descriptor)
	if err != nil {
		return err
	}
	if actual.datos == nil ||
		actual.datos.resumen.AdmitidaEn.Before(evidencia.datos.resumen.AdmitidaEn) {
		return ErrAdmisionGarantiaActoDenegada
	}
	// La admisión original conserva su instante probatorio. Al revalidar sólo
	// puede avanzar el reloj: actor, sesión, material y controles deben ser los
	// mismos, y la ventana original no se amplía con la nueva comprobación.
	refrescada := *actual.datos
	refrescada.resumen.AdmitidaEn = evidencia.datos.resumen.AdmitidaEn
	if refrescada != *evidencia.datos ||
		!actual.datos.resumen.AdmitidaEn.Before(evidencia.datos.resumen.VigenteHasta) {
		return ErrAdmisionGarantiaActoDenegada
	}
	return nil
}

func minimoAdmision(primero time.Time, resto ...time.Time) time.Time {
	for _, actual := range resto {
		if actual.Before(primero) {
			primero = actual
		}
	}
	return primero
}
