package bootstrap

import (
	"context"
	"reflect"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// HuellasProvisionLecturasNominalesRRHHBolsa prepara las dos huellas de una
// asignación ya publicada. Las versiones 21..32 replican exactamente la
// versión base 5..16 y añaden sólo las tres consultas nominales; por ejemplo,
// la v6 pasa a v22 sin conceder por accidente documental o datos de contacto.
func HuellasProvisionLecturasNominalesRRHHBolsa(ctx context.Context, p *politicaBorradorLlamamientoBolsaDesarrollo) (string, string, error) {
	if p == nil {
		return "", "", errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	actual, objetivo, err := prepararProvisionLecturasNominalesRRHHBolsa(ctx, p)
	if err != nil {
		return "", "", err
	}
	pre, e1 := actual.AsignacionPerfil.HuellaSHA256()
	post, e2 := objetivo.AsignacionPerfil.HuellaSHA256()
	if e1 != nil || e2 != nil {
		return "", "", errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return pre, post, nil
}

// ProvisionarLecturasNominalesRRHHBolsa usa la autoridad central existente y
// su CAS. La referencia de aprobación y ambas huellas llegan de la provisión
// gobernada fuera de la petición HTTP; ninguna ausencia crea una concesión.
// La referencia es una entrada del operador: este método comprueba formato,
// huellas y CAS, pero la aprobación humana se verifica en el canal invocante.
func ProvisionarLecturasNominalesRRHHBolsa(ctx context.Context, p *politicaBorradorLlamamientoBolsaDesarrollo,
	aprobacionRef, preimagenSHA256, objetivoSHA256 string) error {
	if p == nil || !aprobacionDocumentalBolsaValida.MatchString(aprobacionRef) ||
		!huellaDocumentalBolsaValida.MatchString(preimagenSHA256) || !huellaDocumentalBolsaValida.MatchString(objetivoSHA256) {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	actual, objetivo, err := prepararProvisionLecturasNominalesRRHHBolsa(ctx, p)
	if err != nil {
		return err
	}
	pre, e1 := actual.AsignacionPerfil.HuellaSHA256()
	post, e2 := objetivo.AsignacionPerfil.HuellaSHA256()
	if e1 != nil || e2 != nil || pre != preimagenSHA256 || post != objetivoSHA256 {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preparada, err := p.autoridad.prepararInstantanea(ctx, objetivo, false)
	if err != nil || !reflect.DeepEqual(preparada, objetivo) ||
		p.autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, actual) != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preparada)
	p.publicada = true
	return nil
}

func prepararProvisionLecturasNominalesRRHHBolsa(ctx context.Context, p *politicaBorradorLlamamientoBolsaDesarrollo) (dominiovec.InstantaneaAutorizacion, dominiovec.InstantaneaAutorizacion, error) {
	var vacio dominiovec.InstantaneaAutorizacion
	if ctx == nil || ctx.Err() != nil || p == nil || p.soporte == nil || p.soporte.soporteCanal == nil ||
		p.autoridad == nil {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	lector, ok := p.autoridad.(lectorAsignacionPublicadaCTDesarrollo)
	if !ok {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	vinculo, err := p.soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil || vinculo.PrincipalID == "" || vinculo.PerfilActivoRef == "" {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	publicada, existe, err := lector.leerAsignacionPublicada(ctx, vinculo.PerfilActivoRef)
	if err != nil || !existe || publicada.instantanea.Validar() != nil {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	actual := publicada.instantanea
	version := actual.VersionRol.Version
	if version < 5 || version > 16 || actual.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		actual.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef ||
		!actual.AsignacionPerfil.VigenteEn(p.reloj.Ahora()) ||
		actual.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	base, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, p.reloj.Ahora(), version)
	if err != nil {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	base = instantaneaMiBolsaEsperada(base, actual)
	base.AsignacionPerfil.AsignacionID = actual.AsignacionPerfil.AsignacionID
	base.AsignacionPerfil.Version = actual.AsignacionPerfil.Version
	if !reflect.DeepEqual(base, actual) {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	objetivo, err := objetivoProvisionRRHHBolsa(actual, vinculo, p.soporte, p.reloj.Ahora().UTC().Truncate(time.Microsecond),
		version+saltoProvisionLecturasNominalesBolsa)
	if err != nil || objetivo.Validar() != nil {
		return vacio, vacio, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return actual, objetivo, nil
}
