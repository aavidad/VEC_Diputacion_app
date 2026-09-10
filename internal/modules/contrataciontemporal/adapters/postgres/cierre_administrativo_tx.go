package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

type TransaccionCierreAdministrativoPostgreSQL struct {
	pool      iniciadorRegistroIncorporacionV2
	proveedor ProveedorAutorizacionCierreAdministrativo
}

var _ ports.TransaccionCierreAdministrativo = (*TransaccionCierreAdministrativoPostgreSQL)(nil)

func NuevaTransaccionCierreAdministrativoPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorAutorizacionCierreAdministrativo) (*TransaccionCierreAdministrativoPostgreSQL, error) {
	if nuloRegistroTX(pool) || nuloRegistroTX(proveedor) {
		return nil, ports.ErrTransaccionCierreAdministrativoNoDisponible
	}
	return &TransaccionCierreAdministrativoPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

const prepararCierreSQL87 = `SELECT vec_contratacion_temporal.preparar_cierre_administrativo_sin_cese_v1($1::jsonb,$2::jsonb,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)::text`
const confirmarCierreSQL87 = `SELECT vec_contratacion_temporal.confirmar_cierre_administrativo_sin_cese_v1($1,$2::jsonb,$3,$4)::text`

type solicitudCierreSQL87 struct {
	Operacion         string `json:"operacion"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	SeguimientoRef    string `json:"seguimiento_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	TransicionClave   string `json:"transicion_clave"`
	MotivoClave       string `json:"motivo_clave"`
}

func solicitudCierre87(s ports.SolicitudTransaccionCierreAdministrativo) solicitudCierreSQL87 {
	return solicitudCierreSQL87{string(s.Operacion), s.OrganizacionRef, s.ExpedienteRef, s.SeguimientoRef, s.VersionEsperada, s.ClaveIdempotencia, string(s.TransicionClave), string(s.MotivoClave)}
}

type reciboCierreSQL87 struct {
	Version        uint64 `json:"version_resultante"`
	ActuacionRef   string `json:"actuacion_ref"`
	ReciboRef      string `json:"recibo_ref"`
	ActorRef       string `json:"actor_ref"`
	CorrelacionRef string `json:"correlacion_ref"`
}
type preparacionCierreSQL87 struct {
	Recuperado       bool                                                       `json:"recuperado"`
	Solicitud        solicitudCierreSQL87                                       `json:"solicitud"`
	VerificadaEn     time.Time                                                  `json:"verificada_en"`
	Original         domain.PublicacionDefinicionSeguimiento                    `json:"publicacion_original"`
	Sucesora         domain.PublicacionDefinicionSeguimiento                    `json:"publicacion_sucesora"`
	Anterior         domain.EstadoPersistidoSeguimiento                         `json:"estado_anterior"`
	AnteriorCanon    string                                                     `json:"anterior_canon_hex"`
	AnteriorSHA256   string                                                     `json:"anterior_sha256"`
	Inventario       ports.EntradaInventarioTareasCierreAdministrativoEjercicio `json:"inventario"`
	InventarioCanon  string                                                     `json:"inventario_canon_hex"`
	InventarioSHA256 string                                                     `json:"inventario_sha256"`
	ActuacionRef     string                                                     `json:"actuacion_ref"`
	ReciboRef        string                                                     `json:"recibo_ref"`
	RegistradaEn     time.Time                                                  `json:"registrada_en"`
	Documentos       []domain.DocumentoSeguimiento                              `json:"documentos"`
	Recibo           *reciboCierreSQL87                                         `json:"recibo"`
	Posterior        *domain.EstadoPersistidoSeguimiento                        `json:"estado_posterior"`
	PosteriorCanon   string                                                     `json:"posterior_canon_hex"`
	PosteriorSHA256  string                                                     `json:"posterior_sha256"`
}

func decodificarCierre87(ctx context.Context, b []byte, v any) error {
	if validarJSONRegistroV2(ctx, b) != nil {
		return ports.ErrResultadoCierreAdministrativoInvalido
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || exigirFinSnapshotSeguimiento(d) != nil {
		return ports.ErrResultadoCierreAdministrativoInvalido
	}
	return nil
}
func errorCierre87(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "P0871":
			return ports.ErrClaveIdempotenciaCierreAdministrativoUsada
		case "P0872":
			return domain.ErrVersionEnConflicto
		case "42501", "P0873", "P1102":
			return ports.ErrCierreAdministrativoDenegado
		case "22023", "P0870":
			return ports.ErrPreparacionCierreAdministrativoInvalida
		}
	}
	return ports.ErrTransaccionCierreAdministrativoNoDisponible
}

func (a AutorizacionCierreAdministrativo) validarEn(s ports.SolicitudTransaccionCierreAdministrativo, t time.Time) error {
	if a.validarPara(s) != nil || !domain.InstanteUTCCanonico(t) || !a.Confirmacion.DentroDeVentanaEn(t) || !a.Contexto.Vinculo.VigenteEn(t, a.Contexto.Resultado) {
		return ports.ErrCierreAdministrativoDenegado
	}
	v, err := a.Contexto.Vinculo.Datos()
	r := a.Exportacion.ResumenCapacidad()
	if err != nil || !vd.CumpleGarantiaAutenticacion(v.GarantiaObservada, vd.AuthAssuranceHigh) || t.Before(r.EmitidaEn()) || !t.Before(r.ExpiraEn()) {
		return ports.ErrCierreAdministrativoDenegado
	}
	return nil
}

func (t *TransaccionCierreAdministrativoPostgreSQL) EjecutarCierreAdministrativo(ctx context.Context, s ports.SolicitudTransaccionCierreAdministrativo, aplicar ports.AplicarCierreAdministrativo) (ports.ResultadoCierreAdministrativo, error) {
	cero := ports.ResultadoCierreAdministrativo{}
	if ctx == nil || t == nil || nuloRegistroTX(t.pool) || nuloRegistroTX(t.proveedor) || aplicar == nil || s.Validar() != nil || s.Operacion != ports.OperacionCerrarAdministrativamenteSinCese || s.TransicionClave != domain.TransicionCerrarAdministrativamenteSinCese {
		return cero, ports.ErrSolicitudCierreAdministrativoInvalida
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	autoridad, err := t.proveedor.AutorizarCierreAdministrativo(ctx, s)
	if err != nil || autoridad.validarPara(s) != nil {
		return cero, ports.ErrCierreAdministrativoDenegado
	}
	peticion, err := json.Marshal(solicitudCierre87(s))
	if err != nil {
		return cero, ports.ErrSolicitudCierreAdministrativoInvalida
	}
	coordenadas, err := autoridad.coordenadasSQL()
	if err != nil {
		return cero, err
	}
	parametros, err := exportacionParametrosRegistroV2(autoridad.Exportacion, true)
	if err != nil {
		return cero, ports.ErrCierreAdministrativoDenegado
	}
	defer func() {
		clear(peticion)
		clear(coordenadas)
		for _, v := range parametros {
			if b, ok := v.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorCierre87(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tx.Rollback(c)
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorCierre87(ctx, err)
	}
	var contenido []byte
	args := append([]any{peticion, coordenadas}, parametros...)
	if err = tx.QueryRow(ctx, prepararCierreSQL87, args...).Scan(&contenido); err != nil {
		return cero, errorCierre87(ctx, err)
	}
	defer func() { clear(contenido) }()
	var wire preparacionCierreSQL87
	if decodificarCierre87(ctx, contenido, &wire) != nil || wire.Solicitud != solicitudCierre87(s) || autoridad.validarEn(s, wire.VerificadaEn) != nil {
		return cero, ports.ErrResultadoCierreAdministrativoInvalido
	}
	original, err := domain.RestaurarDefinicionSeguimiento(wire.Original)
	if err != nil {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	sucesora, err := domain.RestaurarDefinicionSeguimiento(wire.Sucesora)
	if err != nil {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	canon, err := hex.DecodeString(wire.AnteriorCanon)
	if err != nil {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	estadoJSON, err := json.Marshal(wire.Anterior)
	if err != nil {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	previo, err := RestaurarSeguimientoPersistido(original, ExpectativaSeguimientoPersistido{Referencia: s.SeguimientoRef, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, RelacionRef: wire.Anterior.RelacionRef, Definicion: original.Referencia(), Version: s.VersionEsperada}, SnapshotSeguimientoPersistido{EstadoJSON: estadoJSON, EstadoCanonico: canon, HuellaCanonicaSHA256: wire.AnteriorSHA256})
	if err != nil {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	inventario, err := restaurarInventarioTareasCierre87(wire.Inventario, wire.InventarioCanon, wire.InventarioSHA256)
	if err != nil || inventario.ValidarPara(s) != nil || inventario.Pendientes != 0 {
		return cero, ports.ErrPreparacionCierreAdministrativoInvalida
	}
	var recibo reciboCierreSQL87
	if wire.Recuperado {
		if wire.Recibo == nil || wire.Posterior == nil {
			return cero, ports.ErrResultadoCierreAdministrativoInvalido
		}
		posterior, err := restaurarSnapshotCierreAdministrativo(original, sucesora, *wire.Posterior, wire.PosteriorCanon, wire.PosteriorSHA256)
		if err != nil {
			return cero, ports.ErrResultadoCierreAdministrativoInvalido
		}
		recibo = *wire.Recibo
		actos := posterior.Actuaciones()
		ultimo := actos[len(actos)-1]
		ep := posterior.Estado()
		if ep.Continuacion == nil || ep.Continuacion.HuellaEstadoAnteriorSHA256 != wire.AnteriorSHA256 || ep.HuellaRaizSHA256 != previo.Estado().HuellaRaizSHA256 || ep.Referencia != s.SeguimientoRef || ep.OrganizacionRef != s.OrganizacionRef || ep.ExpedienteRef != s.ExpedienteRef || ep.RelacionRef != previo.Estado().RelacionRef ||
			posterior.Version() != s.VersionEsperada+1 || ultimo.TransicionClave != s.TransicionClave || ultimo.MotivoClave != s.MotivoClave || ultimo.ActuacionRef != recibo.ActuacionRef || ultimo.ReciboRef != recibo.ReciboRef || ultimo.ActorRef != recibo.ActorRef || ultimo.CorrelacionRef != recibo.CorrelacionRef || ultimo.ActuacionRef != wire.ActuacionRef || ultimo.ReciboRef != wire.ReciboRef || !ultimo.RegistradaEn.Equal(wire.RegistradaEn) {
			return cero, ports.ErrResultadoCierreAdministrativoInvalido
		}
	} else {
		if wire.Recibo != nil || wire.Posterior != nil || wire.RegistradaEn != wire.VerificadaEn {
			return cero, ports.ErrPreparacionCierreAdministrativoInvalida
		}
		datos := domain.DatosTransicionSeguimiento{ActuacionRef: wire.ActuacionRef, TransicionClave: s.TransicionClave, MotivoClave: s.MotivoClave, ActorRef: autoridad.ActorRef, UnidadRef: autoridad.UnidadRef, EfectivoEn: wire.RegistradaEn, RegistradaEn: wire.RegistradaEn, Documentos: wire.Documentos, ReciboRef: wire.ReciboRef, CorrelacionRef: autoridad.CorrelacionRef}
		continuacion, err := domain.NuevaContinuacionSeguimiento(original, sucesora, previo, datos)
		if err != nil {
			return cero, err
		}
		p := ports.PreparacionTransaccionCierreAdministrativo{Solicitud: s, Definicion: original, DefinicionSucesora: &sucesora, Continuacion: &continuacion, Seguimiento: previo, Inventario: inventario, ContextoAutorizacionV3: autoridad.Contexto, SolicitudAutorizacionV3: autoridad.Solicitud, DecisionAutorizacionV3: autoridad.Decision, ConfirmacionAutorizacionV3: autoridad.Confirmacion, MotivoAutorizacionV3: autoridad.Motivo, ActorRef: autoridad.ActorRef, PerfilRef: autoridad.PerfilRef, UnidadRef: autoridad.UnidadRef, ActuacionRef: wire.ActuacionRef, ReciboRef: wire.ReciboRef, CorrelacionRef: autoridad.CorrelacionRef, Documentos: wire.Documentos, EfectivoEn: wire.RegistradaEn, RegistradaEn: wire.RegistradaEn}
		if p.ValidarPara(s) != nil {
			return cero, ports.ErrPreparacionCierreAdministrativoInvalida
		}
		siguiente, err := aplicar(p)
		if err != nil {
			return cero, err
		}
		snapshot, err := PrepararSnapshotCierreAdministrativo(original, sucesora, siguiente)
		if err != nil {
			return cero, err
		}
		var confirmadoJSON []byte
		if err = tx.QueryRow(ctx, confirmarCierreSQL87, wire.ReciboRef, snapshot.EstadoJSON, snapshot.EstadoCanonico, snapshot.HuellaCanonicaSHA256).Scan(&confirmadoJSON); err != nil {
			return cero, errorCierre87(ctx, err)
		}
		if decodificarCierre87(ctx, confirmadoJSON, &recibo) != nil {
			return cero, ports.ErrResultadoCierreAdministrativoInvalido
		}
		if recibo.ActuacionRef != wire.ActuacionRef || recibo.ReciboRef != wire.ReciboRef || recibo.ActorRef != autoridad.ActorRef || recibo.CorrelacionRef != autoridad.CorrelacionRef {
			return cero, ports.ErrResultadoCierreAdministrativoInvalido
		}
	}
	estado := ports.EstadoResultadoCierreAdministrativoConfirmado
	if wire.Recuperado {
		estado = ports.EstadoResultadoCierreAdministrativoReplayConfirmado
	}
	resultado, err := ports.NuevoResultadoCierreAdministrativo(ports.DatosResultadoCierreAdministrativo{Solicitud: s, VersionResultante: recibo.Version, ActuacionRef: recibo.ActuacionRef, ReciboRef: recibo.ReciboRef, ActorRef: recibo.ActorRef, CorrelacionRef: recibo.CorrelacionRef, Estado: estado})
	if err != nil {
		return cero, err
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorCierre87(ctx, err)
	}
	confirmado = true
	return resultado, nil
}
