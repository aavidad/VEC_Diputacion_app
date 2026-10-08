package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// registrarContactoTelefonoServidorV4 usa el reloj PostgreSQL dentro de la
// misma función que serializa clave, reglas, consumo V3 e inserción.
func registrarContactoTelefonoServidorV4(ctx context.Context, tx pgx.Tx, c ports.ComandoRegistrarContactoParticipacion, out *ports.RegistroContactoParticipacion) error {
	if !c.InstanteServidor || !c.Contacto.Instante.IsZero() || c.Contacto.Canal != dominiobolsa.CanalContactoTelefono || c.Contacto.LlamamientoRef == "" || c.Contacto.OfertaRef != "" {
		return ports.ErrContactoParticipacionNoDisponible
	}
	var maximo, separacion, impedir, resultados, zona, desde, hasta, soloHabiles, control, fecha, habil any
	if p := c.ControlIntentos; p != nil {
		if p.Validar() != nil {
			return ports.ErrContactoParticipacionNoDisponible
		}
		maximo = p.MaximoIntentos()
		separacion = int(p.SeparacionMinima / time.Second)
		impedir = p.ControlSeparacion == dominiobolsa.ControlReglaImpedir
		resultados = append([]string(nil), p.ResultadosSinContacto...)
		if p.Franja.Zona != nil {
			zona = p.Franja.Zona.String()
			desde, hasta = p.Franja.DesdeMinuto, p.Franja.HastaMinuto
			soloHabiles = p.Franja.SoloDiasHabiles
			control = p.Franja.Control
			if p.Franja.SoloDiasHabiles {
				if c.FechaDiaHabil.IsZero() {
					return ports.ErrContactoParticipacionNoDisponible
				}
				fecha, habil = c.FechaDiaHabil, c.DiaHabil
			}
		}
	}
	m := c.Material
	var previoSin int
	var previoContactado bool
	var previoUltimo *time.Time
	err := tx.QueryRow(ctx, `SELECT reutilizado,recibo_ref,contacto_ref,instante,previos_sin_contacto,previo_contactado,previo_ultimo FROM vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::numeric,$15::numeric,$16,$17,$18,$19,$20::integer,$21::integer,$22::boolean,$23::text[],$24::text,$25::integer,$26::integer,$27::boolean,$28::text,$29::date,$30::boolean)`,
		c.Contacto.ContactoRef, c.Contacto.BolsaRef, c.Contacto.ParticipacionRef, c.Contacto.LlamamientoRef, c.Contacto.Actor, c.Contacto.Resultado, c.Contacto.Anotacion, c.ClaveIdempotencia, c.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
		maximo, separacion, impedir, resultados, zona, desde, hasta, soloHabiles, control, fecha, habil,
	).Scan(&out.Reutilizado, &out.ReciboRef, &out.Contacto.ContactoRef, &out.Contacto.Instante, &previoSin, &previoContactado, &previoUltimo)
	if err != nil {
		return err
	}
	out.Contacto.Instante = out.Contacto.Instante.UTC()
	if out.Contacto.Validar() != nil {
		return ports.ErrContactoParticipacionNoDisponible
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
