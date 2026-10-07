package identidadordinaria

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrCertificadoTemporalNoDisponible = errors.New("certificado_temporal_no_disponible")

type errorCertificadoTemporal struct{ causa error }

func (e *errorCertificadoTemporal) Error() string { return ErrCertificadoTemporalNoDisponible.Error() }
func (e *errorCertificadoTemporal) Unwrap() []error {
	return []error{ErrCertificadoTemporalNoDisponible, e.causa}
}

func falloCertificadoTemporal(causa error) error {
	if causa == nil {
		return ErrCertificadoTemporalNoDisponible
	}
	return &errorCertificadoTemporal{causa: causa}
}

// PoliticaCertificadoTemporal identifica la política privada exacta que emitió
// ServicioIdentidad. La retirada limita toda sesión, aunque su certificado o
// su asignación de perfil aún sean vigentes.
type PoliticaCertificadoTemporal struct {
	Referencia   string
	HuellaSHA256 string
	RetiradaEn   time.Time
}

func (p PoliticaCertificadoTemporal) valida(ahora time.Time) bool {
	if len(p.Referencia) < 26 || len(p.Referencia) > 132 || !strings.HasPrefix(p.Referencia, "pga_") ||
		len(p.HuellaSHA256) != 64 || p.RetiradaEn.IsZero() || p.RetiradaEn.Location() != time.UTC ||
		p.RetiradaEn.Nanosecond() != 0 || !ahora.Before(p.RetiradaEn) {
		return false
	}
	for _, c := range p.Referencia[4:] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	for _, c := range p.HuellaSHA256 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return strings.Trim(p.HuellaSHA256, "0") != ""
}

type ConfiguracionFuenteCertificadoTemporal struct {
	Identidad    *httpseguridad.ServicioIdentidad
	Revalidador  core.RevalidadorAutenticacionActorV1
	Resolutor    core.ResolutorContextoActorRegistradoV2
	Autorizacion ports.FuenteAutorizacion
	Reloj        core.RelojVinculoAutenticacionActorV2
	PorCuenta    map[string]PerfilNominal
	Politica     PoliticaCertificadoTemporal
}

// FuenteCertificadoTemporal comparte las autoridades de identidad y V3 con
// otros consumidores. No conserva sesiones, actores ni decisiones entre usos.
type FuenteCertificadoTemporal struct {
	identidad    *httpseguridad.ServicioIdentidad
	revalidador  core.RevalidadorAutenticacionActorV1
	resolutor    core.ResolutorContextoActorRegistradoV2
	autorizacion ports.FuenteAutorizacion
	reloj        core.RelojVinculoAutenticacionActorV2
	porCuenta    map[string]PerfilNominal
	politica     PoliticaCertificadoTemporal
}

func NuevaFuenteCertificadoTemporal(c ConfiguracionFuenteCertificadoTemporal) (*FuenteCertificadoTemporal, error) {
	if c.Identidad == nil || dependenciaNula(c.Revalidador) || dependenciaNula(c.Resolutor) ||
		dependenciaNula(c.Autorizacion) || dependenciaNula(c.Reloj) ||
		len(c.PorCuenta) == 0 || len(c.PorCuenta) > 512 || !c.Politica.valida(c.Reloj.Ahora()) {
		return nil, ErrCertificadoTemporalNoDisponible
	}
	porCuenta := make(map[string]PerfilNominal, len(c.PorCuenta))
	for cuenta, nominal := range c.PorCuenta {
		if (core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial,
		}, PerfilActivoRef: nominal.PerfilActivoRef}).Validar() != nil ||
			!versionRolRefValida(nominal.VersionRolRef) {
			return nil, ErrCertificadoTemporalNoDisponible
		}
		porCuenta[cuenta] = nominal
	}
	return &FuenteCertificadoTemporal{c.Identidad, c.Revalidador, c.Resolutor, c.Autorizacion, c.Reloj, porCuenta, c.Politica}, nil
}

// SesionRegistrada procede de la cápsula ya vinculada por la fachada interna.
// El consumidor emite y consume su decisión V3 para cada recurso y finalidad.
type SesionRegistrada struct {
	vinculo      core.VinculoAutenticacionActorV2
	resultado    core.ResultadoContextoActorRegistradoV2
	snapshot     core.InstantaneaAutorizacion
	empleadoRef  string
	canalSHA256  string
	vigenteHasta time.Time
}

const sesionRegistradaRedactada = "[SESION CERTIFICADO TEMPORAL CONFIDENCIAL]"

func (SesionRegistrada) String() string   { return sesionRegistradaRedactada }
func (SesionRegistrada) GoString() string { return sesionRegistradaRedactada }
func (SesionRegistrada) LogValue() slog.Value {
	return slog.StringValue(sesionRegistradaRedactada)
}
func (SesionRegistrada) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(sesionRegistradaRedactada))
}
func (SesionRegistrada) MarshalJSON() ([]byte, error) { return nil, ErrCertificadoTemporalNoDisponible }
func (*SesionRegistrada) UnmarshalJSON([]byte) error  { return ErrCertificadoTemporalNoDisponible }
func (SesionRegistrada) MarshalText() ([]byte, error) { return nil, ErrCertificadoTemporalNoDisponible }
func (*SesionRegistrada) UnmarshalText([]byte) error  { return ErrCertificadoTemporalNoDisponible }
func (SesionRegistrada) MarshalBinary() ([]byte, error) {
	return nil, ErrCertificadoTemporalNoDisponible
}
func (*SesionRegistrada) UnmarshalBinary([]byte) error { return ErrCertificadoTemporalNoDisponible }
func (SesionRegistrada) GobEncode() ([]byte, error)    { return nil, ErrCertificadoTemporalNoDisponible }
func (*SesionRegistrada) GobDecode([]byte) error       { return ErrCertificadoTemporalNoDisponible }
func (SesionRegistrada) MarshalCBOR() ([]byte, error)  { return nil, ErrCertificadoTemporalNoDisponible }
func (*SesionRegistrada) UnmarshalCBOR([]byte) error   { return ErrCertificadoTemporalNoDisponible }
func (SesionRegistrada) MarshalYAML() (any, error)     { return nil, ErrCertificadoTemporalNoDisponible }
func (*SesionRegistrada) UnmarshalYAML(func(any) error) error {
	return ErrCertificadoTemporalNoDisponible
}
func (SesionRegistrada) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrCertificadoTemporalNoDisponible
}
func (*SesionRegistrada) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrCertificadoTemporalNoDisponible
}

func (s SesionRegistrada) Contexto() (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, core.InstantaneaAutorizacion, error) {
	if s.vinculo.ValidarPara(s.resultado) != nil || s.snapshot.Validar() != nil ||
		s.empleadoRef == "" || s.canalSHA256 == "" || s.vigenteHasta.IsZero() {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, ErrCertificadoTemporalNoDisponible
	}
	resultado, err := s.resultado.Clonar()
	if err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, ErrCertificadoTemporalNoDisponible
	}
	return s.vinculo, resultado, clonarInstantaneaAutorizacion(s.snapshot), nil
}

func (s SesionRegistrada) EmpleadoRef() string     { return s.empleadoRef }
func (s SesionRegistrada) CanalSHA256() string     { return s.canalSHA256 }
func (s SesionRegistrada) VigenteHasta() time.Time { return s.vigenteHasta }

// EsperadosSesion se obtiene exclusivamente de una sesión abierta. Al
// revalidar, sus valores cotejan la identidad previa; no seleccionan una nueva.
type EsperadosSesion struct {
	personaRef, cuentaRef, perfilRef, autenticacionRef, sesionRef, empleadoRef, canalSHA256 string
}

const esperadosSesionRedactados = "[IDENTIDAD DE SESION ESPERADA CONFIDENCIAL]"

func (EsperadosSesion) String() string   { return esperadosSesionRedactados }
func (EsperadosSesion) GoString() string { return esperadosSesionRedactados }
func (EsperadosSesion) LogValue() slog.Value {
	return slog.StringValue(esperadosSesionRedactados)
}
func (EsperadosSesion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(esperadosSesionRedactados))
}
func (EsperadosSesion) MarshalJSON() ([]byte, error) { return nil, ErrCertificadoTemporalNoDisponible }
func (*EsperadosSesion) UnmarshalJSON([]byte) error  { return ErrCertificadoTemporalNoDisponible }
func (EsperadosSesion) MarshalText() ([]byte, error) { return nil, ErrCertificadoTemporalNoDisponible }
func (*EsperadosSesion) UnmarshalText([]byte) error  { return ErrCertificadoTemporalNoDisponible }
func (EsperadosSesion) MarshalBinary() ([]byte, error) {
	return nil, ErrCertificadoTemporalNoDisponible
}
func (*EsperadosSesion) UnmarshalBinary([]byte) error { return ErrCertificadoTemporalNoDisponible }
func (EsperadosSesion) GobEncode() ([]byte, error)    { return nil, ErrCertificadoTemporalNoDisponible }
func (*EsperadosSesion) GobDecode([]byte) error       { return ErrCertificadoTemporalNoDisponible }
func (EsperadosSesion) MarshalCBOR() ([]byte, error)  { return nil, ErrCertificadoTemporalNoDisponible }
func (*EsperadosSesion) UnmarshalCBOR([]byte) error   { return ErrCertificadoTemporalNoDisponible }
func (EsperadosSesion) MarshalYAML() (any, error)     { return nil, ErrCertificadoTemporalNoDisponible }
func (*EsperadosSesion) UnmarshalYAML(func(any) error) error {
	return ErrCertificadoTemporalNoDisponible
}
func (EsperadosSesion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrCertificadoTemporalNoDisponible
}
func (*EsperadosSesion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrCertificadoTemporalNoDisponible
}

func (s SesionRegistrada) Esperados() (EsperadosSesion, error) {
	d, err := s.vinculo.Datos()
	if err != nil || s.empleadoRef == "" || s.canalSHA256 == "" {
		return EsperadosSesion{}, ErrCertificadoTemporalNoDisponible
	}
	return EsperadosSesion{d.PrincipalID, d.CuentaRef, d.PerfilActivoRef, d.AutenticacionRef, d.SesionRef, s.empleadoRef, s.canalSHA256}, nil
}

// Abrir consume una petición previamente autenticada por la misma instancia
// de ServicioIdentidad y ligada por FachadaIdentidadOffline.AutenticarYVincular.
func (f *FuenteCertificadoTemporal) Abrir(ctx context.Context) (SesionRegistrada, error) {
	var vacia SesionRegistrada
	if f == nil || f.identidad == nil || dependenciaNula(f.revalidador) ||
		dependenciaNula(f.resolutor) || dependenciaNula(f.autorizacion) ||
		dependenciaNula(f.reloj) || ctx == nil {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	ahora := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !f.politica.valida(ahora) {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	cuenta, auditoria, err := f.identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	if cuenta.Validar() != nil || cuenta.Metodo != core.AuthMethodCertificate ||
		cuenta.Garantia != core.AuthAssuranceSubstantial ||
		auditoria.CuentaRef() != cuenta.CuentaRef || auditoria.CuentaOrdinariaRef() != cuenta.CuentaRef ||
		auditoria.CuentaPrivilegiada() || auditoria.Superficie() != httpseguridad.SuperficieInternaCorporativa ||
		auditoria.ControlSesionEstado() != httpseguridad.EstadoControlSesionActiva ||
		auditoria.MetodoPrimario() != httpseguridad.MetodoCertificado ||
		auditoria.MetodoObservado() != cuenta.Metodo || auditoria.GarantiaObservada() != cuenta.Garantia ||
		auditoria.PoliticaGarantiaRef() != f.politica.Referencia ||
		auditoria.PoliticaGarantiaHuellaSHA256() != f.politica.HuellaSHA256 ||
		!factorCertificadoTemporalValido(auditoria) || auditoria.CanalVinculadoRef() == "" {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	nominal, ok := f.porCuenta[cuenta.CuentaRef]
	if !ok {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, f.revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef()},
		f.resolutor, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: nominal.PerfilActivoRef}, f.reloj)
	if err != nil || ctx.Err() != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	d, err := vinculo.Datos()
	if err != nil || d.CuentaRef != cuenta.CuentaRef || d.CuentaOrdinariaRef != cuenta.CuentaRef ||
		d.CuentaPrivilegiada || d.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		d.MetodoObservado != core.AuthMethodCertificate || d.GarantiaObservada != core.AuthAssuranceSubstantial ||
		d.AutenticacionRef != auditoria.AutenticacionRef() || d.SesionRef != auditoria.SesionRef() ||
		d.PerfilActivoRef != nominal.PerfilActivoRef || d.PoliticaGarantiaRef != f.politica.Referencia ||
		d.PoliticaGarantiaHuellaSHA256 != f.politica.HuellaSHA256 || !vinculo.VigenteEn(ahora, resultado) {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	if err := f.identidad.ExigirSujetoPersonaCertificadoTemporal(ctx, d.PrincipalID); err != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	empleadoRef, empleadoHasta, ok := empleadoAcreditado(resultado, ahora)
	if !ok {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	snapshot, err := f.autorizacion.ObtenerInstantaneaAutorizacion(ctx, d.PrincipalID, nominal.PerfilActivoRef)
	if err != nil || ctx.Err() != nil || snapshot.Validar() != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	ahora = f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !f.politica.valida(ahora) || !vinculo.VigenteEn(ahora, resultado) ||
		!resultado.Contexto.Instantanea.VigenteEn(ahora) || !empleadoVigente(resultado, empleadoRef, ahora) ||
		snapshot.AsignacionPerfil.PrincipalID != d.PrincipalID ||
		snapshot.AsignacionPerfil.PerfilActivoRef != nominal.PerfilActivoRef ||
		snapshot.AsignacionPerfil.VersionRolRef != nominal.VersionRolRef ||
		snapshot.VersionRol.Referencia() != nominal.VersionRolRef ||
		snapshot.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		snapshot.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		!snapshot.AsignacionPerfil.VigenteEn(ahora) ||
		ahora.Before(snapshot.VersionRol.PublicadaEn) || ahora.Before(snapshot.ControlVigenciaVersionRol.ActualizadoEn) ||
		ctx.Err() != nil {
		return vacia, ErrCertificadoTemporalNoDisponible
	}
	if err := f.identidad.ExigirSujetoPersonaCertificadoTemporal(ctx, d.PrincipalID); err != nil {
		return vacia, falloCertificadoTemporal(err)
	}
	canal := sha256.Sum256([]byte(auditoria.CanalVinculadoRef()))
	return SesionRegistrada{
		vinculo: vinculo, resultado: resultado, snapshot: clonarInstantaneaAutorizacion(snapshot), empleadoRef: empleadoRef,
		canalSHA256: hex.EncodeToString(canal[:]),
		vigenteHasta: minimoTiempo(d.SesionValidaHasta, resultado.Contexto.Instantanea.VigenteHasta,
			empleadoHasta, snapshot.AsignacionPerfil.VigenteHasta, f.politica.RetiradaEn),
	}, nil
}

func (f *FuenteCertificadoTemporal) Revalidar(ctx context.Context, esperado EsperadosSesion) (SesionRegistrada, error) {
	if esperado.personaRef == "" || esperado.cuentaRef == "" || esperado.perfilRef == "" ||
		esperado.autenticacionRef == "" || esperado.sesionRef == "" || esperado.empleadoRef == "" || esperado.canalSHA256 == "" {
		return SesionRegistrada{}, ErrCertificadoTemporalNoDisponible
	}
	sesion, err := f.Abrir(ctx)
	if err != nil {
		return SesionRegistrada{}, err
	}
	actual, err := sesion.Esperados()
	if err != nil || actual.personaRef != esperado.personaRef || actual.cuentaRef != esperado.cuentaRef ||
		actual.perfilRef != esperado.perfilRef || actual.autenticacionRef != esperado.autenticacionRef ||
		actual.sesionRef != esperado.sesionRef || actual.empleadoRef != esperado.empleadoRef ||
		subtle.ConstantTimeCompare([]byte(actual.canalSHA256), []byte(esperado.canalSHA256)) != 1 {
		return SesionRegistrada{}, ErrCertificadoTemporalNoDisponible
	}
	return sesion, nil
}

func factorCertificadoTemporalValido(a httpseguridad.ContextoAuditoriaAutenticada) bool {
	factores := a.Factores()
	return len(factores) == 1 && factores[0].Metodo == httpseguridad.MetodoCertificado &&
		factores[0].EvidenciaRef != "" && factores[0].GrupoCriptograficoRef != "" && !factores[0].VerificadoEn.IsZero()
}

func empleadoAcreditado(r core.ResultadoContextoActorRegistradoV2, ahora time.Time) (string, time.Time, bool) {
	if r.Validar() != nil {
		return "", time.Time{}, false
	}
	var ref string
	var hasta time.Time
	for _, v := range r.Contexto.Instantanea.Vinculos {
		if v.Tipo != core.TipoReferenciaContextoActorEmpleado {
			continue
		}
		if ref != "" || !v.VigenteEn(ahora) {
			return "", time.Time{}, false
		}
		ref, hasta = v.Referencia, v.VigenteHasta
	}
	return ref, hasta, ref != ""
}

func empleadoVigente(r core.ResultadoContextoActorRegistradoV2, ref string, ahora time.Time) bool {
	actual, _, ok := empleadoAcreditado(r, ahora)
	return ok && actual == ref
}

func minimoTiempo(primero time.Time, restantes ...time.Time) time.Time {
	for _, t := range restantes {
		if t.Before(primero) {
			primero = t
		}
	}
	return primero
}

func clonarInstantaneaAutorizacion(original core.InstantaneaAutorizacion) core.InstantaneaAutorizacion {
	copia := original
	copia.AsignacionPerfil.Ambitos = append([]core.AmbitoPerfil(nil), original.AsignacionPerfil.Ambitos...)
	for i := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[i].Valores = append([]string(nil), original.AsignacionPerfil.Ambitos[i].Valores...)
	}
	copia.VersionRol.Concesiones = append([]core.ConcesionRol(nil), original.VersionRol.Concesiones...)
	for i := range copia.VersionRol.Concesiones {
		copia.VersionRol.Concesiones[i].Finalidades = append([]string(nil), original.VersionRol.Concesiones[i].Finalidades...)
		copia.VersionRol.Concesiones[i].CamposPermitidos = append([]string(nil), original.VersionRol.Concesiones[i].CamposPermitidos...)
		copia.VersionRol.Concesiones[i].Obligaciones = append([]string(nil), original.VersionRol.Concesiones[i].Obligaciones...)
	}
	copia.Politicas = append([]core.PoliticaRestrictiva(nil), original.Politicas...)
	for i := range copia.Politicas {
		copia.Politicas[i].Acciones = append([]string(nil), original.Politicas[i].Acciones...)
		copia.Politicas[i].Modulos = append([]string(nil), original.Politicas[i].Modulos...)
		copia.Politicas[i].TiposRecurso = append([]string(nil), original.Politicas[i].TiposRecurso...)
		copia.Politicas[i].FinalidadesPermitidas = append([]string(nil), original.Politicas[i].FinalidadesPermitidas...)
		copia.Politicas[i].CamposPermitidos = append([]string(nil), original.Politicas[i].CamposPermitidos...)
		copia.Politicas[i].Obligaciones = append([]string(nil), original.Politicas[i].Obligaciones...)
		copia.Politicas[i].Restricciones = append([]core.RestriccionAtributoRecurso(nil), original.Politicas[i].Restricciones...)
		for j := range copia.Politicas[i].Restricciones {
			copia.Politicas[i].Restricciones[j].ValoresPermitidos = append([]string(nil), original.Politicas[i].Restricciones[j].ValoresPermitidos...)
		}
	}
	return copia
}
