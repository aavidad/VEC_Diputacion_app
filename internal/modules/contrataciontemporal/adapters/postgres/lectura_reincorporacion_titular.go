package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	esquemaLecturaReincorporacionTitular         = "vec.contratacion-temporal.lectura-reincorporacion-titular.v1"
	consultaLecturaReincorporacionTitular        = `SELECT vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)::text`
	maximoRespuestaLecturaReincorporacionTitular = 16 * 1024
)

type iniciadorLecturaReincorporacionTitular interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type LectorAntecedenteReincorporacionTitularPostgreSQL struct {
	pool iniciadorLecturaReincorporacionTitular
}

var _ ports.LectorAntecedenteReincorporacionTitular = (*LectorAntecedenteReincorporacionTitularPostgreSQL)(nil)

func NuevoLectorAntecedenteReincorporacionTitularPostgreSQL(pool *pgxpool.Pool) (*LectorAntecedenteReincorporacionTitularPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &LectorAntecedenteReincorporacionTitularPostgreSQL{pool: pool}, nil
}

type respuestaLecturaReincorporacionTitular struct {
	Esquema             string    `json:"esquema"`
	Resultado           string    `json:"resultado"`
	ExpedienteRef       string    `json:"expediente_ref"`
	CeseEventoRef       string    `json:"cese_evento_ref"`
	CeseReciboRef       string    `json:"cese_recibo_ref"`
	LecturaRef          string    `json:"lectura_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	RegistradaEn        time.Time `json:"registrada_en"`
}

func decodificarLecturaReincorporacionTitular(b []byte, m ports.MaterialReincorporacionTitular) (ports.AntecedenteReincorporacionTitular, error) {
	var vacio ports.AntecedenteReincorporacionTitular
	var r respuestaLecturaReincorporacionTitular
	if len(b) == 0 || len(b) > maximoRespuestaLecturaReincorporacionTitular ||
		decodificarJSONEstricto(b, &r) != nil || r.Esquema != esquemaLecturaReincorporacionTitular {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	a := ports.AntecedenteReincorporacionTitular{Resultado: r.Resultado, ExpedienteRef: r.ExpedienteRef,
		CeseEventoRef: r.CeseEventoRef, CeseReciboRef: r.CeseReciboRef, LecturaRef: r.LecturaRef,
		AuditoriaRef: r.AuditoriaRef, ConsumoHuellaSHA256: r.ConsumoHuellaSHA256, RegistradaEn: r.RegistradaEn}
	if !a.ValidoPara(m) {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	return a, nil
}

func validarAtestacionLecturaReincorporacion(m ports.MaterialReincorporacionTitular, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) error {
	if a.ValidarEstructura() != nil {
		return ports.ErrAutorizacionDenegada
	}
	recurso := vd.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion,
		Tipo:    ports.TipoRecursoLecturaReincorporacionTitular,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"version_expediente": strconv.FormatUint(m.VersionEsperada, 10),
			"relacion_ref": m.RelacionRef, "fecha_efectiva": m.FechaEfectiva.Format(time.DateOnly),
			"documento_ref": m.DocumentoRef, "documento_sha256": m.DocumentoSHA256}}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || c.Operacion() != string(ports.AccionConsultarAntecedenteReincorporacionTitular) ||
		c.EfectoRef() != m.ExpedienteRef || c.EfectoHuellaSHA256() != huella ||
		c.AudienciaConsumo() != ports.AudienciaLecturaReincorporacionTitularV1 {
		return ports.ErrAutorizacionDenegada
	}
	decision := a.DecisionCanonica()
	defer borrarBytes(decision)
	var d struct {
		DecisionRef     string `json:"decision_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		RecursoRef      string `json:"recurso_ref"`
		Contexto        string `json:"contexto_recurso_huella_sha256"`
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
	}
	if json.Unmarshal(decision, &d) != nil || d.DecisionRef != c.DecisionRef() ||
		d.PrincipalID != m.ActorRef || d.PerfilActivoRef != m.PerfilRef ||
		d.RecursoRef != m.ExpedienteRef || d.Contexto != huella ||
		d.Accion != string(ports.AccionConsultarAntecedenteReincorporacionTitular) ||
		d.Finalidad != ports.FinalidadLecturaReincorporacionTitular {
		return ports.ErrAutorizacionDenegada
	}
	h := sha256.Sum256(decision)
	if hex.EncodeToString(h[:]) != c.DecisionHuellaSHA256() {
		return ports.ErrAutorizacionDenegada
	}
	return nil
}

func (l *LectorAntecedenteReincorporacionTitularPostgreSQL) LeerAntecedenteReincorporacionTitular(ctx context.Context,
	m ports.MaterialReincorporacionTitular, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.AntecedenteReincorporacionTitular, error) {
	var vacio ports.AntecedenteReincorporacionTitular
	if ctx == nil || !m.Valido() {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if l == nil || dependenciaNula(l.pool) {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	if err := validarAtestacionLecturaReincorporacion(m, a); err != nil {
		return vacio, err
	}
	cuerpo, err := json.Marshal(materialReincorporacionSQL(m))
	if err != nil || len(cuerpo) > 4096 {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	defer borrarBytes(cuerpo)
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var respuesta ports.AntecedenteReincorporacionTitular
	for intento := 0; intento < maximoIntentosSeguimiento; intento++ {
		err = func() error {
			tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
			if e != nil {
				return e
			}
			defer revertirTransaccion(tx)
			if e = configurarTransaccionSeguimiento(ctx, tx); e != nil {
				return e
			}
			var b []byte
			e = tx.QueryRow(ctx, consultaLecturaReincorporacionTitular, cuerpo,
				secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()),
				secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&b)
			if e != nil {
				return e
			}
			defer borrarBytes(b)
			respuesta, e = decodificarLecturaReincorporacionTitular(b, m)
			if e != nil {
				return e
			}
			return tx.Commit(ctx)
		}()
		if err == nil {
			return respuesta, nil
		}
		if ctx.Err() != nil || !errorPostgreSQLReintentable(err) {
			break
		}
	}
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		return vacio, err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return vacio, ports.ErrAutorizacionDenegada
		case "22023":
			return vacio, ports.ErrOperacionSeguimientoInvalida
		}
	}
	return vacio, ports.ErrOperacionSeguimientoNoDisponible
}
