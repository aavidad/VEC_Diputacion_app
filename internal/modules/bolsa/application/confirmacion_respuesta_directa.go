package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ConfirmarAceptacionDirecta actúa sobre la ficha ya consultada. B30 y B17 se
// cotejan en la transacción de Bolsa; la aplicación no consulta respuestas
// privadas antes de obtener una autorización nominal.
func (s *ServicioSituacionParticipacion) ConfirmarAceptacionDirecta(ctx context.Context,
	q ports.SolicitudConfirmarAceptacionDirecta,
) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	vacio := ports.RegistroConfirmacionAceptacionDirecta{}
	if ctx == nil || s == nil || q.Validar() != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioConfirmacionAceptacionDirecta)
	if !ok || s.reloj == nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	canon, huella, err := ports.CanonConfirmacionAceptacionDirecta(q)
	if err != nil {
		return vacio, err
	}
	var contextoRecurso []byte
	auth, decision, confirmacion, material, err := s.autorizarOperacionConContrato(ctx,
		q.SolicitudCambiarSituacionParticipacion, huella, &contextoRecurso,
		contratoOperacionSituacion{
			accion: ports.AccionConfirmarAceptacionDirecta, tipoRecurso: ports.TipoRecursoConfirmarAceptacionDirecta,
			finalidad: ports.FinalidadConfirmarAceptacionDirecta, audiencia: ports.AudienciaConfirmarAceptacionDirecta,
			claveAtributo: "confirmacion_aceptacion_sha256", pertenenciaEnTransaccion: true,
			propagarInstantaneaObsoleta: true,
		})
	if err != nil {
		return vacio, err
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	h := sha256.Sum256([]byte(q.ParticipacionRef + "\x1f" + q.ClaveIdempotencia))
	comando := ports.ComandoConfirmarAceptacionDirecta{
		ComandoCambiarSituacionParticipacion: ports.ComandoCambiarSituacionParticipacion{
			Cambio: domain.CambioSituacionParticipacion{ParticipacionRef: q.ParticipacionRef,
				Destino: domain.SituacionPendienteIncorporacion,
				Desde:   ahora, RegistradaEn: ahora, Motivo: q.Motivo},
			Actor: q.ResultadoContexto.Contexto.PersonaRef, BolsaRef: q.BolsaRef,
			ClaveIdempotencia:     q.ClaveIdempotencia,
			ReciboRef:             "recibo:respuesta-revision:" + hex.EncodeToString(h[:]),
			SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material,
		}, LlamamientoRef: q.LlamamientoRef, SituacionEsperadaDesde: q.SituacionEsperadaDesde,
		Canon: canon, CanonSHA256: huella, ContextoRecursoCanonico: contextoRecurso,
	}
	registro, err := repo.ConfirmarAceptacionDirecta(ctx, comando)
	if err != nil {
		return vacio, err
	}
	if registro.ValidarPara(comando) != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	return registro, nil
}
