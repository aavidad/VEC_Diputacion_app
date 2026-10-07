package httpseguridad

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type claveCapsulaPresentacionCertificado struct{}

type capsulaPresentacionCertificadoVinculada struct {
	capsula  CapsulaPresentacionCertificado
	canalRef string
}

// VincularCapsulaPresentacion liga una confirmacion durable ya cotejada al
// contexto exacto de C4. No admite otra capsula ni reutiliza una anterior.
func (s *ServicioPresentacionCertificado) VincularCapsulaPresentacion(
	ctx context.Context,
	capsula CapsulaPresentacionCertificado,
	canal CanalProxyAutenticado,
) (context.Context, error) {
	if s == nil || s.identidad == nil || ctx == nil || capsula.datos == nil ||
		capsula.datos.servicio != s || canal.validar(s.identidad) != nil ||
		ctx.Value(clavePeticionCertificadoActual{}) != capsula.datos.marcador ||
		ctx.Value(claveCapsulaIdentidad{}) != nil ||
		ctx.Value(claveCapsulaPresentacionCertificado{}) != nil ||
		subtle.ConstantTimeCompare([]byte(canal.ReferenciaVinculacion()), []byte(capsula.datos.canalRef)) != 1 {
		return nil, ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ahora := s.identidad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if validarResultadoPresentacionCertificado(
		capsula.datos.resultado, capsula.datos.presentacion,
		capsula.datos.estadoActual, true, ahora,
	) != nil || !capsula.datos.consumida.CompareAndSwap(false, true) {
		return nil, ErrPresentacionCertificadoNoValida
	}
	return context.WithValue(ctx, claveCapsulaPresentacionCertificado{},
		capsulaPresentacionCertificadoVinculada{capsula: capsula, canalRef: canal.ReferenciaVinculacion()}), nil
}

// datosCapsulaPresentacion obtiene los atributos nominales de la misma
// presentacion confirmada. El actor F1 y cada acto V3 se resuelven después.
// No afirma que la sesion siga vigente más allá de esta petición.
func (s *ServicioIdentidad) datosCapsulaPresentacion(
	ctx context.Context,
) (domain.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if s == nil || ctx == nil {
		return domain.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return domain.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, err
	}
	vinculada, ok := ctx.Value(claveCapsulaPresentacionCertificado{}).(capsulaPresentacionCertificadoVinculada)
	if !ok || vinculada.capsula.datos == nil || vinculada.capsula.datos.servicio == nil ||
		vinculada.capsula.datos.servicio.identidad != s ||
		ctx.Value(clavePeticionCertificadoActual{}) != vinculada.capsula.datos.marcador ||
		!vinculada.capsula.datos.consumida.Load() ||
		subtle.ConstantTimeCompare([]byte(vinculada.canalRef), []byte(vinculada.capsula.datos.canalRef)) != 1 ||
		ctx.Value(claveCapsulaIdentidad{}) != nil {
		return domain.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrPresentacionCertificadoNoValida
	}
	c := vinculada.capsula.datos
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if validarResultadoPresentacionCertificado(c.resultado, c.presentacion, c.estadoActual, true, ahora) != nil ||
		ctx.Err() != nil {
		return domain.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrPresentacionCertificadoNoValida
	}
	original, recibo := c.resultado.SesionOriginal, c.resultado.Recibo
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: original.CuentaRef, Metodo: original.MetodoObservado, Garantia: original.GarantiaObservada,
	}
	if cuenta.Validar() != nil {
		return domain.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrPresentacionCertificadoNoValida
	}
	factores := make([]ResumenFactorAuditoria, 0, len(c.estadoActual.factores))
	for _, factor := range c.estadoActual.factores {
		factores = append(factores, ResumenFactorAuditoria{
			Metodo: factor.Metodo, EvidenciaRef: factor.EvidenciaRef,
			GrupoCriptograficoRef: factor.GrupoCriptograficoRef, VerificadoEn: factor.VerificadoEn,
		})
	}
	auditoria := ContextoAuditoriaAutenticada{
		autenticacionRef: original.AutenticacionRef,
		asercionRef:      original.AsercionRef, sesionRef: original.SesionRef,
		controlSesionRef: original.ControlSesionRef,
		cuentaRef:        original.CuentaRef, cuentaOrdinariaRef: original.CuentaOrdinariaRef,
		autenticacionHuellaSHA256: original.AutenticacionHuellaSHA256,
		controlSesionRevision:     original.ControlSesionRevision,
		controlSesionEstado:       EstadoControlSesionActiva,
		controlSesionHuellaSHA256: original.ControlSesionHuellaSHA256,
		sesionRevalidadaEn:        original.SesionRevalidadaEn,
		sesionValidaHasta:         original.SesionValidaHasta,
		emisor:                    c.estadoActual.emisor, audiencia: c.estadoActual.audiencia,
		cuentaPrivilegiada: original.CuentaPrivilegiada,
		superficie:         c.estadoActual.superficie, metodoPrimario: c.estadoActual.metodoPrimario,
		metodoObservado:           original.MetodoObservado,
		autenticacionVerificadaEn: original.AutenticacionVerificadaEn,
		garantia:                  original.GarantiaObservada,
		emitidaEn:                 original.SesionEmitidaEn,
		noAntesDe:                 original.SesionEmitidaEn, expiraEn: original.SesionValidaHasta,
		politicaGarantiaRef: original.PoliticaGarantiaRef,
		huellaPolitica:      "sha256:" + original.PoliticaGarantiaHuellaSHA256,
		huellaConfiguracion: "sha256:" + hex.EncodeToString(s.huellaConfiguracion[:]),
		canalVinculadoRef:   c.canalRef, factores: factores,
		presentacionRef:                  recibo.PresentacionRef,
		presentacionOperacionRef:         recibo.OperacionRef,
		presentacionGeneracion:           recibo.Generacion,
		presentacionSesionGeneracion:     recibo.SesionGeneracion,
		presentacionReciboHuellaSHA256:   recibo.HuellaSHA256,
		presentacionAsercionHuellaSHA256: recibo.AsercionActualHuellaSHA256,
		presentacionCanalHuellaSHA256:    recibo.CanalSHA256,
		presentacionEmitidaEn:            c.presentacion.AsercionActualEmitidaEn,
		presentacionValidaHasta:          recibo.ValidaHasta,
	}
	return cuenta, auditoria, nil
}
