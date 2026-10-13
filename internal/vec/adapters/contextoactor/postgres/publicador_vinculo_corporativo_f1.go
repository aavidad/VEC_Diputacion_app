package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SolicitudPublicacionVinculoF1 contiene referencias opacas de una fila de
// fuente previamente aprobada por DBA. La base comprueba su coincidencia exacta.
type SolicitudPublicacionVinculoF1 struct {
	OperacionRef          string
	CuentaRef             string
	PersonaRef            string
	PerfilRef             string
	VinculoContextoRef    string
	VinculoCorporativoRef string
	OrganizacionRef       string
}

func (s SolicitudPublicacionVinculoF1) valida() bool {
	return s.OperacionRef != "" && s.CuentaRef != "" && s.PersonaRef != "" &&
		s.PerfilRef != "" && s.VinculoContextoRef != "" &&
		s.VinculoCorporativoRef != "" && s.OrganizacionRef != ""
}

type ResultadoPublicacionVinculoF1 struct {
	Codigo                string `json:"codigo"`
	Estado                string `json:"estado"`
	Huella                string `json:"huella,omitempty"`
	Clave                 string `json:"clave,omitempty"`
	Observado             string `json:"observado,omitempty"`
	Esperado              string `json:"esperado,omitempty"`
	AuditoriaRef          string `json:"auditoria_ref,omitempty"`
	AuditoriaHuellaSHA256 string `json:"auditoria_huella_sha256,omitempty"`
	EventoRef             string `json:"evento_ref,omitempty"`
}

// ErrorPublicacionVinculoF1 retiene solo el código de PostgreSQL y la rutina.
// Nunca incluye parámetros, DSN ni el mensaje del servidor.
type ErrorPublicacionVinculoF1 struct {
	SQLState  string
	Routine   string
	Clave     string
	Observado string
	Esperado  string
}

func (e *ErrorPublicacionVinculoF1) Error() string {
	return fmt.Sprintf("CA42 SQLSTATE=%s routine=%s", e.SQLState, e.Routine)
}

func errorPublicacionVinculoF1(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		e := &ErrorPublicacionVinculoF1{SQLState: pgErr.SQLState(), Routine: pgErr.Routine}
		if partes := errorCA42Estructurado.FindStringSubmatch(pgErr.Message); partes != nil {
			e.Clave, e.Observado, e.Esperado = partes[1], partes[2], partes[3]
		}
		return e
	}
	return &ErrorPublicacionVinculoF1{SQLState: "08000", Routine: "conexion"}
}

var errorCA42Estructurado = regexp.MustCompile(`^CA42: PARO clave=([A-Za-z0-9_]+) actual=([A-Za-z0-9_]+) esperado=([A-Za-z0-9_]+)$`)

const consultaComprobarVinculoF1 = `SELECT vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1($1,$2,$3,$4,$5,$6,$7)`
const consultaAplicarVinculoF1 = `SELECT vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1($1,$2,$3,$4,$5,$6,$7,$8)`

// EjecutarPublicacionVinculoF1 abre una sola transacción serializable. La
// comprobación confirma su lectura auditada; la aplicación confirma el efecto.
func EjecutarPublicacionVinculoF1(ctx context.Context, pool *pgxpool.Pool, s SolicitudPublicacionVinculoF1, aplicar bool, huella string) (ResultadoPublicacionVinculoF1, error) {
	var resultado ResultadoPublicacionVinculoF1
	if pool == nil || !s.valida() || (aplicar && len(huella) != 64) || (!aplicar && huella != "") {
		return resultado, &ErrorPublicacionVinculoF1{SQLState: "22023", Routine: "entrada"}
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return resultado, errorPublicacionVinculoF1(err)
	}
	defer func() {
		rollbackCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancelar()
		if fallo := tx.Rollback(rollbackCtx); fallo != nil && !errors.Is(fallo, pgx.ErrTxClosed) {
			causa := errorPublicacionVinculoF1(fallo)
			slog.Error("CA42.rollback", "causa", causa)
		}
	}()
	for _, setting := range []string{
		`SET LOCAL timezone='UTC'`,
		`SET LOCAL lock_timeout='2s'`,
		`SET LOCAL statement_timeout='10s'`,
		`SET LOCAL transaction_timeout='15s'`,
	} {
		if _, err = tx.Exec(ctx, setting); err != nil {
			return resultado, errorPublicacionVinculoF1(err)
		}
	}
	var crudo []byte
	if aplicar {
		err = tx.QueryRow(ctx, consultaAplicarVinculoF1,
			s.OperacionRef, s.CuentaRef, s.PersonaRef, s.PerfilRef,
			s.VinculoContextoRef, s.VinculoCorporativoRef, s.OrganizacionRef, huella).Scan(&crudo)
	} else {
		err = tx.QueryRow(ctx, consultaComprobarVinculoF1,
			s.OperacionRef, s.CuentaRef, s.PersonaRef, s.PerfilRef,
			s.VinculoContextoRef, s.VinculoCorporativoRef, s.OrganizacionRef).Scan(&crudo)
	}
	if err != nil {
		return resultado, errorPublicacionVinculoF1(err)
	}
	if err = json.Unmarshal(crudo, &resultado); err != nil || resultado.Codigo != "CA42" || resultado.Estado == "" {
		return ResultadoPublicacionVinculoF1{}, &ErrorPublicacionVinculoF1{SQLState: "XX000", Routine: "respuesta"}
	}
	if err = tx.Commit(ctx); err != nil {
		return ResultadoPublicacionVinculoF1{}, errorPublicacionVinculoF1(err)
	}
	return resultado, nil
}
