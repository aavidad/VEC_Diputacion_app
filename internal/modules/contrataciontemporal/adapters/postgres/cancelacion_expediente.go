package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Cancelación del expediente (CT122). La preparación y la confirmación usan
// el mismo repositorio que las operaciones de seguimiento; aquí solo viven la
// traducción del material y la lectura de la cancelación registrada.

var _ ports.LectorEstadoCancelacion = (*RepositorioOperacionSeguimientoPostgreSQL)(nil)

// materialCancelacionSQL traduce la intención al contrato JSON de CT122.
func materialCancelacionSQL(m ports.MaterialCancelacion) map[string]any {
	fases := make([]string, len(m.Datos.FasesAdmitidas))
	for i, f := range m.Datos.FasesAdmitidas {
		fases[i] = string(f)
	}
	return map[string]any{"organizacion_ref": m.OrganizacionRef, "expediente_ref": m.ExpedienteRef, "actor_ref": m.ActorRef,
		"perfil_ref": m.PerfilRef, "version_esperada": m.VersionEsperada, "canal": string(m.Datos.Canal),
		"motivo_clave": string(m.Datos.MotivoClave), "fases_admitidas": fases, "observaciones": m.Datos.Observaciones}
}

// ConsultarEstadoCancelacion lee la cancelación registrada del expediente.
// La composición solo la invoca tras acreditar la lectura del expediente.
func (r *RepositorioOperacionSeguimientoPostgreSQL) ConsultarEstadoCancelacion(ctx context.Context, org, exp string) (ports.EstadoCancelacionExpediente, error) {
	var vacio ports.EstadoCancelacionExpediente
	if !domain.ReferenciaOpacaValida(org) || !domain.ReferenciaOpacaValida(exp) {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	var salida struct {
		Esquema       string `json:"esquema"`
		ExpedienteRef string `json:"expediente_ref"`
		Cancelacion   *struct {
			Canal         domain.CanalCancelacion `json:"canal"`
			MotivoClave   string                  `json:"motivo_clave"`
			FasePrevia    string                  `json:"fase_previa"`
			Observaciones string                  `json:"observaciones"`
			ReciboRef     string                  `json:"recibo_ref"`
			RegistradaEn  time.Time               `json:"registrada_en"`
		} `json:"cancelacion"`
	}
	err := r.ejecutar(ctx, true, func(tx pgx.Tx) error {
		var contenido []byte
		if err := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.consultar_cancelacion_expediente_v1($1,$2)::text", org, exp).Scan(&contenido); err != nil {
			return err
		}
		if len(contenido) > maximoCargaSeguimiento || decodificarJSONEstricto(contenido, &salida) != nil ||
			salida.Esquema != "vec.contratacion-temporal.cancelacion-expediente.v1" || salida.ExpedienteRef != exp {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	estado := ports.EstadoCancelacionExpediente{ExpedienteRef: exp}
	if c := salida.Cancelacion; c != nil {
		if !c.Canal.Valido() || !domain.ClaveCatalogo(c.MotivoClave).Valida() || !domain.ClaveFase(c.FasePrevia).Valida() ||
			!domain.ReferenciaOpacaValida(c.ReciboRef) || !domain.InstanteUTCCanonico(c.RegistradaEn.UTC()) {
			return vacio, ports.ErrResultadoSeguimientoNoConfiable
		}
		estado.Cancelacion = &ports.CancelacionRegistrada{Canal: c.Canal, MotivoClave: c.MotivoClave, FasePrevia: c.FasePrevia,
			Observaciones: c.Observaciones, ReciboRef: c.ReciboRef, RegistradaEn: c.RegistradaEn.UTC()}
	}
	return estado, nil
}
