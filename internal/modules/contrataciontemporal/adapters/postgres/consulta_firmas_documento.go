package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	application "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// LectorFirmasDocumentoAutorizadasPostgreSQL consulta CT con consumo V3,
// auditoría y resultado dentro de la misma transacción.
type LectorFirmasDocumentoAutorizadasPostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

var _ ports.LectorFirmasDocumentoAutorizadas = (*LectorFirmasDocumentoAutorizadasPostgreSQL)(nil)

func NuevoLectorFirmasDocumentoAutorizadasPostgreSQL(pool *pgxpool.Pool) (*LectorFirmasDocumentoAutorizadasPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &LectorFirmasDocumentoAutorizadasPostgreSQL{pool: pool}, nil
}

const consultarFirmasAtestadasSQL = `SELECT vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`

type respuestaFirmasAtestadas struct {
	Encontrado    *bool         `json:"Encontrado"`
	ExpedienteRef string        `json:"ExpedienteRef"`
	Firmas        []firmaSQL118 `json:"Firmas"`
}

func errorConsultaFirmasAtestadas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "42501":
			return ports.ErrFirmaDocumentoDenegada
		case "22023":
			return ports.ErrSolicitudFirmaDocumentoInvalida
		case "P1525":
			return ports.ErrRegistroFirmaDocumentoNoDisponible
		}
	}
	return ports.ErrRegistroFirmaDocumentoNoDisponible
}

// validarFilasFirmasAtestadas comprueba la proyección cerrada antes del commit.
// El esquema CT145 no incluye identidad de persona ni datos del certificado.
func validarFilasFirmasAtestadas(b []byte, respuesta respuestaFirmasAtestadas, expediente string) ([]ports.FirmaRegistrada, error) {
	if respuesta.Encontrado == nil || respuesta.ExpedienteRef != expediente || respuesta.Firmas == nil || len(respuesta.Firmas) > 1000 ||
		(!*respuesta.Encontrado && len(respuesta.Firmas) != 0) {
		return nil, ports.ErrResultadoFirmaDocumentoInvalido
	}
	var crudo struct {
		Firmas []map[string]json.RawMessage `json:"Firmas"`
	}
	if json.Unmarshal(b, &crudo) != nil || len(crudo.Firmas) != len(respuesta.Firmas) {
		return nil, ports.ErrResultadoFirmaDocumentoInvalido
	}
	campos := application.CamposConsultaFirmasDocumento()
	firmas := make([]ports.FirmaRegistrada, 0, len(respuesta.Firmas))
	for i, f := range respuesta.Firmas {
		if i > 0 {
			anterior := respuesta.Firmas[i-1]
			if anterior.Documento > f.Documento || (anterior.Documento == f.Documento && anterior.Secuencia >= f.Secuencia) {
				return nil, ports.ErrResultadoFirmaDocumentoInvalido
			}
		}
		if len(crudo.Firmas[i]) != len(campos) {
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		for _, campo := range campos {
			if _, ok := crudo.Firmas[i][campo]; !ok {
				return nil, ports.ErrResultadoFirmaDocumentoInvalido
			}
		}
		if string(crudo.Firmas[i]["ConMotivoDevolucion"]) == "null" {
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		fila := ports.FirmaRegistrada{
			FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef, Documento: f.Documento,
			Secuencia: f.Secuencia, ExpedienteVersion: f.ExpedienteVersion,
			CatalogoRef: f.CatalogoRef, CatalogoHuella: f.CatalogoHuella,
			PasoRef: f.PasoRef, PasoOrden: f.PasoOrden,
			Resultado: domain.ResultadoFirmaDocumento(f.Resultado), ConMotivoDevolucion: f.ConMotivo,
			OriginalHuella: textoFirma118(f.OriginalHuella), FirmadoHuella: textoFirma118(f.FirmadoHuella),
			SelloTiempoEstado: textoFirma118(f.SelloTiempoEstado), RegistradaEn: f.RegistradaEn.UTC(),
			ClaveIdempotencia: f.ClaveIdempotencia, DocumentoCustodiaRef: textoFirma118(f.DocumentoCustodia),
			DocumentoCustodiaVersion: versionFirma118(f.VersionCustodia),
		}
		if !domain.ReferenciaOpacaValida(fila.FirmaRef) || !domain.ReferenciaOpacaValida(fila.ReciboRef) ||
			!domain.ClaveDocumentoFirmaValida(fila.Documento) || fila.Secuencia < 1 || fila.Secuencia > 100000 ||
			fila.ExpedienteVersion == 0 || fila.ExpedienteVersion > 9007199254740991 ||
			!domain.ReferenciaOpacaValida(fila.CatalogoRef) || !domain.HuellaSHA256FirmaValida(fila.CatalogoHuella) ||
			fila.PasoRef == "" || len(fila.PasoRef) > 256 || fila.PasoOrden < 1 || fila.PasoOrden > domain.MaximoPasosCircuitoFirma ||
			!ports.ClaveIdempotenciaFirmaValida(fila.ClaveIdempotencia) || f.RegistradaEn.IsZero() ||
			(f.DocumentoCustodia == nil) != (f.VersionCustodia == nil) ||
			(f.DocumentoCustodia != nil && (!domain.ReferenciaOpacaValida(fila.DocumentoCustodiaRef) || fila.DocumentoCustodiaVersion == 0 || fila.DocumentoCustodiaVersion > 9007199254740991)) {
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		switch fila.Resultado {
		case domain.ResultadoFirmaFirmado:
			if fila.ConMotivoDevolucion || !domain.HuellaSHA256FirmaValida(fila.OriginalHuella) ||
				!domain.HuellaSHA256FirmaValida(fila.FirmadoHuella) || fila.OriginalHuella == fila.FirmadoHuella ||
				(fila.SelloTiempoEstado != "no_presente" && fila.SelloTiempoEstado != "valido" && fila.SelloTiempoEstado != "no_comprobado") {
				return nil, ports.ErrResultadoFirmaDocumentoInvalido
			}
		case domain.ResultadoFirmaDevuelto:
			if !fila.ConMotivoDevolucion || fila.OriginalHuella != "" || fila.FirmadoHuella != "" ||
				fila.SelloTiempoEstado != "" || f.OriginalHuella != nil || f.FirmadoHuella != nil ||
				f.SelloTiempoEstado != nil || fila.DocumentoCustodiaRef != "" {
				return nil, ports.ErrResultadoFirmaDocumentoInvalido
			}
		default:
			return nil, ports.ErrResultadoFirmaDocumentoInvalido
		}
		firmas = append(firmas, fila)
	}
	return firmas, nil
}

func (l *LectorFirmasDocumentoAutorizadasPostgreSQL) ConsultarFirmasAutorizadas(ctx context.Context, m ports.MaterialConsultaFirmasDocumento, c ports.CapacidadConsultaFirmasDocumento) ([]ports.FirmaRegistrada, error) {
	if ctx == nil || l == nil || nuloRegistroTX(l.pool) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	material, err := application.CanonicoConsultaFirmasDocumento(m)
	if err != nil {
		return nil, err
	}
	if err := application.ValidarCapacidadConsultaFirmasDocumento(c, m); err != nil {
		return nil, err
	}
	parametros, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	defer func() {
		for _, v := range parametros {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return nil, errorConsultaFirmasAtestadas(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
			defer cancelar()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return nil, errorConsultaFirmasAtestadas(ctx, err)
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, consultarFirmasAtestadasSQL, append([]any{string(material)}, parametros...)...).Scan(&contenido); err != nil {
		return nil, errorConsultaFirmasAtestadas(ctx, err)
	}
	defer clear(contenido)
	var respuesta respuestaFirmasAtestadas
	if decodificarFirma118(contenido, &respuesta) != nil {
		return nil, ports.ErrResultadoFirmaDocumentoInvalido
	}
	firmas, err := validarFilasFirmasAtestadas(contenido, respuesta, m.ExpedienteRef)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, errorConsultaFirmasAtestadas(ctx, err)
	}
	confirmado = true
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !*respuesta.Encontrado {
		return nil, ports.ErrExpedienteConsultaFirmasNoEncontrado
	}
	return firmas, nil
}
