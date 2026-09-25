package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// requiereRegistroContactoV2 elige la función de Bolsa 000031 solo cuando
// hace falta: control de intentos o un resultado que la v1 no admite. El
// resto de contactos siguen por la v1 sin cambios.
func requiereRegistroContactoV2(c ports.ComandoRegistrarContactoParticipacion) bool {
	return c.ControlIntentos != nil || c.Contacto.Resultado == dominiobolsa.ResultadoContactoNumeroErroneo || c.Contacto.Resultado == dominiobolsa.ResultadoContactoNoEntregado
}

// registrarContactoV2 registra el contacto y, con control, devuelve el
// resumen de intentos previos que la función calculó bajo su cerrojo.
func registrarContactoV2(ctx context.Context, tx pgx.Tx, c ports.ComandoRegistrarContactoParticipacion, out *ports.RegistroContactoParticipacion) error {
	var maximo, separacion any
	var impedir any
	var resultados any
	if p := c.ControlIntentos; p != nil {
		if p.Validar() != nil {
			return ports.ErrContactoParticipacionNoDisponible
		}
		maximo = p.MaximoIntentos()
		separacion = int(p.SeparacionMinima / time.Second)
		impedir = p.ControlSeparacion == dominiobolsa.ControlReglaImpedir
		resultados = append([]string(nil), p.ResultadosSinContacto...)
	}
	m := c.Material
	var previoSin int
	var previoContactado bool
	var previoUltimo *time.Time
	err := tx.QueryRow(ctx, `SELECT reutilizado,recibo_ref,contacto_ref,previos_sin_contacto,previo_contactado,previo_ultimo FROM vec_bolsa_llamamientos.registrar_contacto_participacion_v2($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21,$22::integer,$23::integer,$24::boolean,$25::text[])`,
		c.Contacto.ContactoRef, c.Contacto.BolsaRef, c.Contacto.ParticipacionRef, nuloTexto(c.Contacto.LlamamientoRef), c.Contacto.Canal, c.Contacto.Instante, c.Contacto.Actor, c.Contacto.Resultado, c.Contacto.Anotacion, c.ClaveIdempotencia, c.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
		maximo, separacion, impedir, resultados,
	).Scan(&out.Reutilizado, &out.ReciboRef, &out.Contacto.ContactoRef, &previoSin, &previoContactado, &previoUltimo)
	if err != nil {
		return err
	}
	if c.ControlIntentos != nil {
		resumen := dominiobolsa.ResumenIntentosTelefonicos{SinContacto: previoSin, Contactado: previoContactado}
		if previoUltimo != nil {
			resumen.UltimoIntento = previoUltimo.UTC()
		}
		out.ResumenPrevio = &resumen
	}
	return nil
}
