package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaIncorporacionAcreditadaSQL = "vec.contratacion-temporal.incorporacion-acreditada.v1"
	esquemaPropuestasExpedienteSQL    = "vec.contratacion-temporal.propuestas-expediente.v1"
)

var _ ports.LectorIncorporacionAcreditada = (*RepositorioOperacionSeguimientoPostgreSQL)(nil)

// ConsultarIncorporacionAcreditada lee las confirmaciones de GINPIX y del
// centro y la no incorporación (CT124), y las propuestas de nombramiento con
// su sustitución (CT128). La composición solo la invoca tras acreditar la
// lectura del detalle del mismo expediente, o dentro del cierre ya autorizado.
func (r *RepositorioOperacionSeguimientoPostgreSQL) ConsultarIncorporacionAcreditada(ctx context.Context, org, exp string) (ports.EstadoIncorporacionAcreditada, error) {
	var vacio ports.EstadoIncorporacionAcreditada
	if !domain.ReferenciaOpacaValida(org) || !domain.ReferenciaOpacaValida(exp) {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	type recibo struct {
		ReciboRef    string    `json:"recibo_ref"`
		RegistradaEn time.Time `json:"registrada_en"`
	}
	var salida struct {
		Esquema       string `json:"esquema"`
		ExpedienteRef string `json:"expediente_ref"`
		GINPIX        *struct {
			Numero       string `json:"ginpix_numero"`
			ConfirmadaEn string `json:"ginpix_confirmada_en"`
			Recibo       recibo `json:"recibo"`
		} `json:"ginpix"`
		Centro *struct {
			FechaIncorporacion string `json:"fecha_incorporacion"`
			DocumentoTipo      string `json:"documento_tipo"`
			DocumentoRef       string `json:"documento_ref"`
			DocumentoSHA256    string `json:"documento_sha256"`
			Recibo             recibo `json:"recibo"`
		} `json:"centro"`
		NoIncorporacion *struct {
			MotivoClave       string `json:"motivo_clave"`
			ConsecuenciaClave string `json:"consecuencia_clave"`
			ResolucionRef     string `json:"resolucion_ref"`
			ResolucionSHA256  string `json:"resolucion_sha256"`
			ResueltaPor       string `json:"resuelta_por"`
			FechaNotificacion string `json:"fecha_notificacion"`
			Recibo            recibo `json:"recibo"`
		} `json:"no_incorporacion"`
	}
	var propuestas struct {
		Esquema       string `json:"esquema"`
		ExpedienteRef string `json:"expediente_ref"`
		Propuestas    []struct {
			Orden             uint32    `json:"orden"`
			VersionResultante uint64    `json:"version_resultante"`
			ConfirmadaEn      time.Time `json:"confirmada_en"`
			ReciboRef         string    `json:"recibo_ref"`
			Vigente           bool      `json:"vigente"`
			Sustitucion       *struct {
				NoIncorporacionReciboRef string    `json:"no_incorporacion_recibo_ref"`
				MotivoClave              string    `json:"motivo_clave"`
				RegistradaEn             time.Time `json:"registrada_en"`
			} `json:"sustitucion"`
		} `json:"propuestas"`
	}
	err := r.ejecutar(ctx, true, func(tx pgx.Tx) error {
		var contenido []byte
		if err := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.consultar_incorporacion_acreditada_v1($1,$2)::text", org, exp).Scan(&contenido); err != nil {
			return err
		}
		if len(contenido) > maximoCargaSeguimiento || decodificarJSONEstricto(contenido, &salida) != nil ||
			salida.Esquema != esquemaIncorporacionAcreditadaSQL || salida.ExpedienteRef != exp {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		if err := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.consultar_propuestas_expediente_v1($1,$2)::text", org, exp).Scan(&contenido); err != nil {
			return err
		}
		if len(contenido) > maximoCargaSeguimiento || decodificarJSONEstricto(contenido, &propuestas) != nil ||
			propuestas.Esquema != esquemaPropuestasExpedienteSQL || propuestas.ExpedienteRef != exp {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	var estado ports.EstadoIncorporacionAcreditada
	if g := salida.GINPIX; g != nil {
		estado.GINPIX = &ports.EstadoGINPIXConfirmado{Numero: g.Numero, ConfirmadaEn: g.ConfirmadaEn,
			ReciboRef: g.Recibo.ReciboRef, RegistradaEn: g.Recibo.RegistradaEn.UTC()}
	}
	if c := salida.Centro; c != nil {
		estado.Centro = &ports.EstadoConfirmacionCentro{FechaIncorporacion: c.FechaIncorporacion, DocumentoTipo: c.DocumentoTipo,
			DocumentoRef: c.DocumentoRef, DocumentoSHA256: c.DocumentoSHA256, ReciboRef: c.Recibo.ReciboRef, RegistradaEn: c.Recibo.RegistradaEn.UTC()}
	}
	if n := salida.NoIncorporacion; n != nil {
		estado.NoIncorporacion = &ports.EstadoNoIncorporacion{MotivoClave: n.MotivoClave, ConsecuenciaClave: n.ConsecuenciaClave,
			ResolucionRef: n.ResolucionRef, ResolucionSHA256: n.ResolucionSHA256, ResueltaPor: n.ResueltaPor,
			FechaNotificacion: n.FechaNotificacion, ReciboRef: n.Recibo.ReciboRef, RegistradaEn: n.Recibo.RegistradaEn.UTC()}
	}
	for _, p := range propuestas.Propuestas {
		e := ports.EstadoPropuestaExpediente{Orden: p.Orden, VersionResultante: p.VersionResultante, ConfirmadaEn: p.ConfirmadaEn.UTC(),
			ReciboRef: p.ReciboRef, Vigente: p.Vigente}
		if s := p.Sustitucion; s != nil {
			e.Sustitucion = &ports.SustitucionPropuestaExpediente{NoIncorporacionReciboRef: s.NoIncorporacionReciboRef,
				MotivoClave: s.MotivoClave, RegistradaEn: s.RegistradaEn.UTC()}
		}
		estado.Propuestas = append(estado.Propuestas, e)
	}
	if !estado.Valido() {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	return estado, nil
}
