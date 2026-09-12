package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const consultaConfiguracionCorreoSQL = `SELECT vec_administracion.consultar_configuracion_correo_v2($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`

var (
	patronHuellaConsultaCorreo = regexp.MustCompile(`^[0-9a-f]{64}$`)
	patronReciboConsultaCorreo = regexp.MustCompile(`^acc_[0-9a-f]{40}$`)
)

// ConsultaConfiguracionCorreoPostgreSQL es la única frontera que entrega la
// vista SMTP. PostgreSQL consume V3 y añade el recibo T13 en esta transacción.
type ConsultaConfiguracionCorreoPostgreSQL struct{ pool iniciadorConfiguracionCorreo }

var _ adminports.RegistroConsultaConfiguracionCorreo = (*ConsultaConfiguracionCorreoPostgreSQL)(nil)

func NuevaConsultaConfiguracionCorreoPostgreSQL(pool *pgxpool.Pool) (*ConsultaConfiguracionCorreoPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ErrConfiguracionCorreoNoDisponible
	}
	return &ConsultaConfiguracionCorreoPostgreSQL{pool: pool}, nil
}

func (r *ConsultaConfiguracionCorreoPostgreSQL) ConsultarConfiguracionCorreoAuditada(ctx context.Context, orden adminports.OrdenConsultaConfiguracionCorreoAutorizada) (adminports.ResultadoConsultaConfiguracionCorreo, error) {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.pool) || validarOrdenConsultaConfiguracionCorreo(orden) != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	payload := bytes.Clone(orden.Preparacion.PayloadNegocio)
	defer borrar(payload)
	tx, err := iniciarConsultaConfiguracionCorreo(ctx, r.pool)
	if err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	capacidad := orden.Material.CapacidadCanonica()
	decision := orden.Material.DecisionCanonica()
	motivo := orden.Material.MotivoCanonico()
	contexto := orden.Material.ContextoActorCanonico()
	ad3 := orden.Material.PayloadVECAD3()
	cose := orden.Material.SobreCOSESign1()
	evidencia := orden.Material.EvidenciaVerificacion()
	raiz := orden.Material.RaizPublicaSPKI()
	defer borrar(capacidad)
	defer borrar(decision)
	defer borrar(motivo)
	defer borrar(contexto)
	defer borrar(ad3)
	defer borrar(cose)
	defer borrar(evidencia)
	defer borrar(raiz)
	var respuesta []byte
	err = tx.QueryRow(ctx, consultaConfiguracionCorreoSQL, payload, capacidad, decision, motivo, contexto,
		int64(orden.Material.PersonaVersion()), int64(orden.Material.PerfilVersion()), ad3, cose, evidencia, raiz).Scan(&respuesta)
	if err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer borrar(respuesta)
	resultado, err := resultadoConsultaConfiguracionCorreoDesdeJSON(respuesta, orden.Preparacion.Auditoria)
	if err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	return resultado, nil
}

func iniciarConsultaConfiguracionCorreo(ctx context.Context, pool iniciadorConfiguracionCorreo) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, err
	}
	for _, sentencia := range [...]string{"SET LOCAL search_path = pg_catalog", "SET LOCAL row_security = on", "SET LOCAL TIME ZONE 'UTC'", "SET LOCAL lock_timeout = '3s'", "SET LOCAL statement_timeout = '15s'", "SET LOCAL idle_in_transaction_session_timeout = '20s'"} {
		if _, err = tx.Exec(ctx, sentencia); err != nil {
			_ = tx.Rollback(context.Background())
			return nil, err
		}
	}
	return tx, nil
}

func validarOrdenConsultaConfiguracionCorreo(orden adminports.OrdenConsultaConfiguracionCorreoAutorizada) error {
	if len(orden.Preparacion.PayloadNegocio) == 0 || len(orden.Preparacion.PayloadNegocio) > 32768 || orden.Material.ValidarEstructura() != nil {
		return errors.New("orden de consulta invalida")
	}
	canonico, err := adminapp.PayloadConsultaConfiguracionCorreo(orden.Preparacion.Auditoria)
	if err != nil {
		return errors.New("payload de consulta invalido")
	}
	defer borrar(canonico)
	if !bytes.Equal(orden.Preparacion.PayloadNegocio, canonico) {
		return errors.New("payload de consulta invalido")
	}
	return nil
}

func resultadoConsultaConfiguracionCorreoDesdeJSON(datos []byte, preparada vecdomain.AuditEntry) (adminports.ResultadoConsultaConfiguracionCorreo, error) {
	if len(datos) == 0 || len(datos) > 32768 {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, errors.New("respuesta invalida")
	}
	var sobre struct {
		Configuracion json.RawMessage `json:"configuracion"`
		Auditoria     json.RawMessage `json:"auditoria"`
	}
	if decodificarEstricto(datos, &sobre) != nil || len(sobre.Configuracion) == 0 || len(sobre.Auditoria) == 0 {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, errors.New("sobre invalido")
	}
	var vista admindomain.VistaConfiguracionCorreo
	var recibo vecdomain.AuditEntry
	if decodificarEstricto(sobre.Configuracion, &vista) != nil || vista.Validar() != nil || decodificarEstricto(sobre.Auditoria, &recibo) != nil || !reciboConsultaCompletoValido(recibo, preparada, vista.Version) {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, errors.New("resultado no acreditado")
	}
	return adminports.ResultadoConsultaConfiguracionCorreo{Vista: vista, ReciboAuditoria: recibo}, nil
}

func decodificarEstricto(datos []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("json con sufijo")
	}
	return nil
}

func reciboConsultaCompletoValido(recibo, preparada vecdomain.AuditEntry, version uint64) bool {
	if version > uint64(^uint(0)>>1) || !patronReciboConsultaCorreo.MatchString(recibo.ID) || recibo.Seq <= 0 || recibo.IntegrityAlgorithm != "sha256-chain-v1" || !patronHuellaConsultaCorreo.MatchString(recibo.PrevSignature) || !patronHuellaConsultaCorreo.MatchString(recibo.Signature) || recibo.AuthorizationRef == "" || recibo.ObjectVersion != int(version) || len(recibo.Metadata) != 1 || recibo.Metadata["consumo_ref"] == "" || recibo.RepresentedSubjectID != "" || recibo.ExpedienteRef != "" || recibo.DocumentRef != "" || recibo.RuleRef != "" || recibo.Reason != "" || recibo.BeforeHash != "" || recibo.AfterHash != "" || recibo.ActorID != preparada.ActorID || recibo.ActorProfile != preparada.ActorProfile || !reflect.DeepEqual(recibo.ActorRoles, preparada.ActorRoles) || recibo.AuthMethod != preparada.AuthMethod || recibo.AuthAssurance != preparada.AuthAssurance || recibo.Purpose != preparada.Purpose || recibo.Action != preparada.Action || recibo.ModuleID != preparada.ModuleID || recibo.SubjectRef != preparada.SubjectRef || recibo.Result != preparada.Result || recibo.CorrelationRef != preparada.CorrelationRef || recibo.OccurredAt.IsZero() || !recibo.OccurredAt.Equal(preparada.OccurredAt) {
		return false
	}
	return true
}
