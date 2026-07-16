package memory

import (
	"context"
	"reflect"
	"strconv"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func claveVersionManifiesto(baremacionRef string, numeroVersion uint64) string {
	return baremacionRef + "\x00" + strconv.FormatUint(numeroVersion, 10)
}

func prepararManifiestoPersistido(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	numeroVersion uint64,
) (*manifiestoBaremacionPersistido, error) {
	switch solicitud.Clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		if numeroVersion != 1 || solicitud.Manifiesto != nil || len(solicitud.Agregado.Decisiones) != 0 {
			return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		return nil, nil
	case puertosbolsa.ClaseCambioIncorporarDecision:
		if solicitud.Manifiesto == nil || solicitud.VersionEsperada == nil ||
			numeroVersion != solicitud.VersionEsperada.Numero+1 {
			return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		decision, existe := solicitud.Agregado.UltimaDecision()
		if !existe {
			return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		manifiesto := solicitud.Manifiesto.Clonar()
		if manifiesto.ValidarCoberturaFirmaPara(
			*solicitud.VersionEsperada, decision.Contenido, decision.Firma,
		) != nil || manifiesto.BaremacionMeritoRef != solicitud.Agregado.ID ||
			manifiesto.DecisionRef != decision.Contenido.ID ||
			manifiesto.VersionBase+1 != numeroVersion ||
			decision.Contenido.VersionBaremacion != numeroVersion {
			return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		return &manifiestoBaremacionPersistido{
			Manifiesto:          manifiesto,
			BaremacionMeritoRef: solicitud.Agregado.ID,
			NumeroVersion:       numeroVersion,
			DecisionRef:         decision.Contenido.ID,
		}, nil
	default:
		return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
}

// instantaneasManifiestosVersionBloqueada reconstruye, bajo el bloqueo del
// repositorio, todos los manifiestos historicos que sostienen el agregado. No
// llama a conectores externos: devuelve copias profundas para verificarlas sin
// mantener el mutex durante una operacion criptografica.
func (r *RepositorioBaremaciones) instantaneasManifiestosVersionBloqueada(
	version puertosbolsa.VersionBaremacion,
) ([]puertosbolsa.ManifiestoProbatorioBaremacion, error) {
	if r == nil || version.Validar() != nil {
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versiones := r.versionesPorBaremacion[version.Referencia.BaremacionMeritoRef]
	if version.Referencia.Numero > uint64(len(versiones)) ||
		!reflect.DeepEqual(versiones[version.Referencia.Numero-1], version) {
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	instantaneas := make([]puertosbolsa.ManifiestoProbatorioBaremacion, 0, len(version.Agregado.Decisiones))
	for indice := range version.Agregado.Decisiones {
		decision := version.Agregado.Decisiones[indice]
		numeroVersion := uint64(indice + 2)
		if decision.Contenido.VersionBaremacion != numeroVersion ||
			decision.Contenido.VersionAnteriorBaremacion+1 != numeroVersion {
			return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		referencia, existe := r.manifiestoRefPorVersion[claveVersionManifiesto(
			version.Referencia.BaremacionMeritoRef, numeroVersion,
		)]
		if !existe {
			return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		persistido, existe := r.manifiestosPorReferencia[referencia]
		if !existe || persistido.BaremacionMeritoRef != version.Referencia.BaremacionMeritoRef ||
			persistido.NumeroVersion != numeroVersion || persistido.DecisionRef != decision.Contenido.ID ||
			persistido.Manifiesto.Referencia != referencia {
			return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		manifiesto := persistido.Manifiesto.Clonar()
		if manifiesto.VersionBase < 1 || manifiesto.VersionBase >= numeroVersion ||
			manifiesto.VersionBase > uint64(len(versiones)) {
			return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		base := versiones[manifiesto.VersionBase-1]
		if base.Validar() != nil || base.Referencia.Numero != manifiesto.VersionBase ||
			base.Referencia.HuellaEstadoSHA256 != manifiesto.HuellaVersionBaseSHA256 ||
			manifiesto.ValidarCoberturaFirmaPara(base.Referencia, decision.Contenido, decision.Firma) != nil ||
			decision.Firma.ManifiestoProbatorioRef != manifiesto.Referencia ||
			decision.Firma.HuellaManifiestoProbatorioSHA256 != manifiesto.HuellaManifiestoSHA256 ||
			decision.Firma.SelloManifiestoProbatorioHMACSHA256 != manifiesto.SelloManifiestoHMACSHA256 {
			return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		instantaneas = append(instantaneas, manifiesto)
	}
	return instantaneas, nil
}

func (r *RepositorioBaremaciones) instantaneasHistoricasParaConfirmacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) ([]puertosbolsa.ManifiestoProbatorioBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return nil, err
	}
	if !r.cadenasIntegrasBloqueadas() {
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if solicitud.Clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return nil, nil
	}
	if solicitud.Clase != puertosbolsa.ClaseCambioIncorporarDecision || solicitud.VersionEsperada == nil {
		return nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	versiones := r.versionesPorBaremacion[solicitud.Agregado.ID]
	numeroEsperado := solicitud.VersionEsperada.Numero
	if len(versiones) == 0 || numeroEsperado == 0 || numeroEsperado > uint64(len(versiones)) ||
		uint64(len(versiones)) > numeroEsperado+1 {
		return nil, puertosbolsa.ErrVersionBaremacionConflicto
	}
	base := versiones[numeroEsperado-1]
	if base.Referencia != *solicitud.VersionEsperada {
		return nil, puertosbolsa.ErrVersionBaremacionConflicto
	}
	// Una unica version posterior puede corresponder al reintento exacto de una
	// confirmacion ya materializada. Se fotografia su resultado; la fase bajo el
	// mutex comprobara despues que la huella de confirmacion coincide.
	versionObjetivo := base
	if uint64(len(versiones)) == numeroEsperado+1 {
		versionObjetivo = versiones[numeroEsperado]
	}
	return r.instantaneasManifiestosVersionBloqueada(versionObjetivo)
}

func (r *RepositorioBaremaciones) verificarInstantaneasManifiestos(
	ctx context.Context,
	instantaneas []puertosbolsa.ManifiestoProbatorioBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	for indice := range instantaneas {
		if err := r.verificarSelloManifiesto(ctx, instantaneas[indice]); err != nil {
			return err
		}
	}
	return validarContextoEjecucion(ctx)
}

func (r *RepositorioBaremaciones) capacidadConfirmacionDisponibleBloqueada(baremacionRef string) bool {
	if r == nil || len(r.auditorias) != len(r.eventosOutbox) ||
		len(r.auditorias) >= maximoTransaccionesMemoria ||
		len(r.referenciasTransaccion) != len(r.auditorias)*2 || !r.cadenasIntegrasBloqueadas() {
		return false
	}
	versiones := r.versionesPorBaremacion[baremacionRef]
	if len(versiones) >= maximoVersionesPorBaremacionMemoria {
		return false
	}
	if len(versiones) == 0 && len(r.versionesPorBaremacion) >= maximoBaremacionesMemoria {
		return false
	}
	return true
}

func (r *RepositorioBaremaciones) cadenasIntegrasBloqueadas() bool {
	if r == nil || len(r.auditorias) != len(r.eventosOutbox) ||
		len(r.referenciasTransaccion) != len(r.auditorias)*2 {
		return false
	}
	huellaAuditoriaAnterior, huellaEventoAnterior := "", ""
	referencias := make(map[string]struct{}, len(r.auditorias)*2)
	referenciasManifiestos := make(map[string]struct{}, len(r.manifiestosPorReferencia))
	for indice := range r.auditorias {
		auditoria, evento := r.auditorias[indice], r.eventosOutbox[indice]
		secuencia := uint64(indice + 1)
		if auditoria.Validar() != nil || evento.Validar() != nil || auditoria.Secuencia != secuencia ||
			evento.Secuencia != secuencia || auditoria.HuellaAnteriorAuditoriaSHA256 != huellaAuditoriaAnterior ||
			evento.HuellaEventoAnteriorSHA256 != huellaEventoAnterior ||
			auditoria.HuellaRegistroSHA256 != huellaAuditoria(auditoria) ||
			evento.HuellaRegistroSHA256 != huellaEvento(evento) || evento.AuditoriaRef != auditoria.Referencia ||
			evento.HuellaAuditoriaSHA256 != auditoria.HuellaRegistroSHA256 ||
			evento.BaremacionMeritoRef != auditoria.BaremacionMeritoRef || evento.VersionNueva != auditoria.VersionNueva ||
			evento.HuellaNuevaSHA256 != auditoria.HuellaNuevaSHA256 || evento.SujetoRef != auditoria.SujetoRef ||
			evento.PrincipalRef != auditoria.PrincipalRef || evento.CorrelacionRef != auditoria.CorrelacionRef ||
			!evento.RegistradoEn.Equal(auditoria.RegistradaEn) {
			return false
		}
		versiones := r.versionesPorBaremacion[auditoria.BaremacionMeritoRef]
		if auditoria.VersionNueva > uint64(len(versiones)) {
			return false
		}
		version := versiones[auditoria.VersionNueva-1]
		if version.Validar() != nil || version.Referencia.Numero != auditoria.VersionNueva ||
			version.Referencia.HuellaEstadoSHA256 != auditoria.HuellaNuevaSHA256 ||
			version.Agregado.SujetoRef != auditoria.SujetoRef || !version.ConfirmadaEn.Equal(auditoria.RegistradaEn) {
			return false
		}
		manifiestos, err := r.instantaneasManifiestosVersionBloqueada(version)
		if err != nil {
			return false
		}
		for actual := range manifiestos {
			referenciasManifiestos[manifiestos[actual].Referencia] = struct{}{}
		}
		for _, referencia := range []string{auditoria.Referencia, evento.Referencia} {
			if _, repetida := referencias[referencia]; repetida {
				return false
			}
			referencias[referencia] = struct{}{}
			if _, reservada := r.referenciasTransaccion[referencia]; !reservada {
				return false
			}
		}
		huellaAuditoriaAnterior = auditoria.HuellaRegistroSHA256
		huellaEventoAnterior = evento.HuellaRegistroSHA256
	}
	if len(referenciasManifiestos) != len(r.manifiestosPorReferencia) ||
		len(referenciasManifiestos) != len(r.manifiestoRefPorVersion) {
		return false
	}
	for referencia, persistido := range r.manifiestosPorReferencia {
		if _, existe := referenciasManifiestos[referencia]; !existe ||
			r.manifiestoRefPorVersion[claveVersionManifiesto(
				persistido.BaremacionMeritoRef, persistido.NumeroVersion,
			)] != referencia {
			return false
		}
	}
	return true
}

func (r *RepositorioBaremaciones) ahora() (time.Time, error) {
	if r == nil || interfazNula(r.reloj) {
		return time.Time{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora := r.reloj.Ahora()
	if ahora.IsZero() {
		return time.Time{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ahora.UTC(), nil
}

func (r *RepositorioBaremaciones) verificarSelloReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	if r == nil || interfazNula(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaReservaBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad: puertosbolsa.FinalidadSelloReservaBaremacion, RepresentacionCanonica: representacion,
		SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	return validarContextoEjecucion(ctx)
}

func (r *RepositorioBaremaciones) verificarSelloConfirmacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	if r == nil || interfazNula(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaConfirmacionBaremacion(solicitud)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad: puertosbolsa.FinalidadSelloConfirmacionBaremacionV2, RepresentacionCanonica: representacion,
		SelloHMAC: solicitud.HuellaSolicitudHMAC,
	}
	if peticion.Validar() != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	return validarContextoEjecucion(ctx)
}

func (r *RepositorioBaremaciones) verificarSelloManifiesto(
	ctx context.Context,
	manifiesto puertosbolsa.ManifiestoProbatorioBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	if r == nil || interfazNula(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaManifiestoProbatorioBaremacion(manifiesto)
	if err != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
		SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
	}
	if peticion.Validar() != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		return errorVerificacionConContexto(ctx, puertosbolsa.ErrSelloBaremacionNoAutentico)
	}
	return validarContextoEjecucion(ctx)
}
