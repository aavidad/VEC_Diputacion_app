package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const consultaSaldoEfectosC3 = `SELECT vec_cronos_v1.consultar_saldo_efectos_c3_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

func NuevoRepositorioConsultaSaldoConEfectosC3(pool *pgxpool.Pool) (*RepositorioConsultaSaldo, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioConsultaSaldo{db: pool, conEfectosC3: true}, nil
}

func PreflightSaldoEfectosC3(ctx context.Context, pool *pgxpool.Pool) error {
	if ctx == nil || pool == nil {
		return ports.ErrDependenciaNoDisponible
	}
	var permitido bool
	err := pool.QueryRow(ctx, `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_cronos_v1.consultar_saldo_efectos_c3_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE'),false)
	 AND NOT has_table_privilege('vec_cronos_v1.snapshot_saldo_efectos_c3','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
	 AND NOT has_table_privilege('vec_cronos_v1.politica_efectos_saldo_c3','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`).Scan(&permitido)
	if err != nil || !permitido {
		return ports.ErrDependenciaNoDisponible
	}
	return nil
}

func decodificarFuenteSaldoEfectosC3(bruto []byte, empleado, desde, hasta, zona string) (ports.FuenteSaldo, error) {
	var vacia ports.FuenteSaldo
	if len(bruto) == 0 || len(bruto) > 8<<20 {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	var sobre struct {
		FuenteCanonica string    `json:"fuente_canonica"`
		SnapshotRef    string    `json:"snapshot_ref"`
		FuenteSHA256   string    `json:"fuente_sha256"`
		CorteUTC       time.Time `json:"corte_utc"`
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(&sobre) != nil || d.Decode(new(any)) != io.EOF || len(sobre.FuenteCanonica) == 0 || len(sobre.FuenteCanonica) > 4<<20 ||
		!domain.HuellaEfectosValida(sobre.FuenteSHA256) || sobre.SnapshotRef != "saldo:cronos:snapshot:"+sobre.FuenteSHA256 ||
		sobre.CorteUTC.IsZero() {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	suma := sha256.Sum256([]byte(sobre.FuenteCanonica))
	if hex.EncodeToString(suma[:]) != sobre.FuenteSHA256 {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	var fuente struct {
		Saldo             json.RawMessage                         `json:"saldo"`
		Programaciones    []domain.ProgramacionSaldoEfectosC3     `json:"programaciones"`
		Permisos          []domain.PermisoConcedidoSaldoEfectosC3 `json:"permisos"`
		Politicas         []domain.PoliticaSaldoEfectosC3         `json:"politicas"`
		CoberturaPermisos *bool                                   `json:"cobertura_permisos"`
	}
	d = json.NewDecoder(strings.NewReader(sobre.FuenteCanonica))
	d.DisallowUnknownFields()
	if d.Decode(&fuente) != nil || d.Decode(new(any)) != io.EOF || len(fuente.Saldo) == 0 || fuente.Programaciones == nil || fuente.Permisos == nil || fuente.Politicas == nil ||
		fuente.CoberturaPermisos == nil || !*fuente.CoberturaPermisos || len(fuente.Programaciones) > 367 || len(fuente.Permisos) > 1000 || len(fuente.Politicas) > 30 {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	base, err := decodificarFuenteSaldoInterna(fuente.Saldo)
	if err != nil || base.EmpleadoRef != empleado || base.Desde != desde || base.Hasta != hasta || base.ZonaHoraria != zona {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	for i := range fuente.Politicas {
		fuente.Politicas[i].PublicadaEn = fuente.Politicas[i].PublicadaEn.UTC()
		if fuente.Politicas[i].Validar() != nil {
			return vacia, ports.ErrDependenciaNoDisponible
		}
	}
	var raiz map[string]json.RawMessage
	var politicasBrutas []json.RawMessage
	if json.Unmarshal([]byte(sobre.FuenteCanonica), &raiz) != nil || json.Unmarshal(raiz["politicas"], &politicasBrutas) != nil || len(politicasBrutas) != len(fuente.Politicas) {
		return vacia, ports.ErrDependenciaNoDisponible
	}
	for i, bruta := range politicasBrutas {
		var campos map[string]json.RawMessage
		if json.Unmarshal(bruta, &campos) != nil || len(campos["reglas"]) == 0 {
			return vacia, ports.ErrDependenciaNoDisponible
		}
		huella := sha256.Sum256(campos["reglas"])
		if hex.EncodeToString(huella[:]) != fuente.Politicas[i].SHA256 {
			return vacia, ports.ErrDependenciaNoDisponible
		}
	}
	for i := range fuente.Programaciones {
		p := &fuente.Programaciones[i]
		p.InicioUTC = p.InicioUTC.UTC()
		p.FinUTC = p.FinUTC.UTC()
	}
	base.EfectosC3 = &ports.FuenteSaldoEfectosC3{SnapshotRef: sobre.SnapshotRef, FuenteSHA256: sobre.FuenteSHA256,
		CorteUTC: sobre.CorteUTC.UTC(), Politicas: fuente.Politicas, Programaciones: fuente.Programaciones, Permisos: fuente.Permisos, CoberturaPermisos: true}
	return base, nil
}
