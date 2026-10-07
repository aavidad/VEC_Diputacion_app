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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	maximoRespuestaVinculoRPT     = 262144
	maximoRespuestaPublicacionRPT = 18874368
	consultaVinculoRPT            = `SELECT vec_contratacion_temporal.consultar_vinculo_categoria_rpt_v1($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`
	registroVinculoRPT            = `SELECT vec_contratacion_temporal.registrar_vinculo_categoria_rpt_v1($1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14,$15,$16::numeric,$17::numeric,$18,$19,$20,$21)::text`
	lecturaPublicacionRPT         = `SELECT vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)::text`
)

var _ ports.FuenteVinculoCategoriaRPT = (*FuenteVinculoCategoriaRPTPostgreSQL)(nil)
var _ ports.FuentePublicacionCategoriaRPT = (*FuentePublicacionCategoriaRPTPostgreSQL)(nil)

type FuenteVinculoCategoriaRPTPostgreSQL struct{ pool *pgxpool.Pool }
type FuentePublicacionCategoriaRPTPostgreSQL struct{ pool *pgxpool.Pool }

func NuevaFuenteVinculoCategoriaRPTPostgreSQL(p *pgxpool.Pool) (*FuenteVinculoCategoriaRPTPostgreSQL, error) {
	if p == nil {
		return nil, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return &FuenteVinculoCategoriaRPTPostgreSQL{p}, nil
}
func NuevaFuentePublicacionCategoriaRPTPostgreSQL(p *pgxpool.Pool) (*FuentePublicacionCategoriaRPTPostgreSQL, error) {
	if p == nil {
		return nil, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return &FuentePublicacionCategoriaRPTPostgreSQL{p}, nil
}

// MaterialPublicacionCategoriaRPTPostgreSQL usa la serializacion jsonb::text
// de PostgreSQL que exige AD3-117. El proveedor de autorizacion RPT debe
// atestar exactamente estos bytes; json.Marshal no es equivalente.
func MaterialPublicacionCategoriaRPTPostgreSQL(ctx context.Context, p *pgxpool.Pool, publicacion domain.PublicacionCategoriaRPT) ([]byte, error) {
	if ctx == nil || p == nil || publicacion.Validar() != nil {
		return nil, ports.ErrVinculoCategoriaRPTInvalido
	}
	var material string
	e := p.QueryRow(ctx, `SELECT pg_catalog.jsonb_build_object('catalogo_id',$1::text,'modulo_id',$2::text,'version',$3::integer,'huella_sha256',$4::text,'categoria_id',$5::text)::text`,
		publicacion.CatalogoID, publicacion.ModuloID, strconv.FormatUint(publicacion.CatalogoVersion, 10), publicacion.CatalogoHuella, publicacion.CategoriaID).Scan(&material)
	if e != nil || len(material) == 0 || len(material) > 4096 {
		return nil, clasificarErrorVinculoRPT(ctx, e)
	}
	return []byte(material), nil
}

func (f *FuenteVinculoCategoriaRPTPostgreSQL) Consultar(ctx context.Context, c ports.ConsultaVinculoCategoriaRPT, cap vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.LecturaVinculoCategoriaRPT, error) {
	if f == nil || f.pool == nil || ctx == nil {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	b, e := c.Canonico()
	if e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, e
	}
	if !capacidadParaVinculoRPT(cap, ports.AccionConsultarVinculoCategoriaRPT, ports.AudienciaConsultarVinculoCategoriaRPT, c.ExpedienteRef, "contratacion_temporal", "vinculo_categoria_rpt_ct", map[string]string{"organizacion_ref": c.OrganizacionRef}, b) {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	tx, e := iniciarVinculoRPT(ctx, f.pool)
	if e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	defer revertirVinculoRPT(tx)
	args := append([]any{string(b)}, argumentosCapacidadVinculoRPT(cap)...)
	var raw string
	if e = tx.QueryRow(ctx, consultaVinculoRPT, args...).Scan(&raw); e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	l, encontrado, e := decodificarConsultaVinculoRPT(raw, c)
	if e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	if !encontrado {
		return ports.LecturaVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoEncontrado
	}
	return l, nil
}

func (f *FuenteVinculoCategoriaRPTPostgreSQL) Registrar(ctx context.Context, m ports.RegistroVinculoCategoriaRPT, capCT, capRPT vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboVinculoCategoriaRPT, error) {
	if f == nil || f.pool == nil || ctx == nil {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	b, e := m.Canonico()
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	materialRPT, e := MaterialPublicacionCategoriaRPTPostgreSQL(ctx, f.pool, m.Publicacion())
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if !capacidadParaVinculoRPT(capCT, ports.AccionRegistrarVinculoCategoriaRPT, ports.AudienciaRegistrarVinculoCategoriaRPT, m.ExpedienteRef, "contratacion_temporal", "vinculo_categoria_rpt_ct", map[string]string{"organizacion_ref": m.OrganizacionRef}, b) ||
		!capacidadParaVinculoRPT(capRPT, ports.AccionConsultarPublicacionCategoriaRPT, ports.AudienciaConsultarPublicacionCategoriaRPT, m.CatalogoID, m.ModuloID, "catalogo_configurable", map[string]string{"catalogo_id": m.CatalogoID, "modulo_id": m.ModuloID}, materialRPT) {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	tx, e := iniciarVinculoRPT(ctx, f.pool)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	defer revertirVinculoRPT(tx)
	args := append([]any{string(b)}, argumentosCapacidadVinculoRPT(capCT)...)
	args = append(args, argumentosCapacidadVinculoRPT(capRPT)...)
	var raw string
	if e = tx.QueryRow(ctx, registroVinculoRPT, args...).Scan(&raw); e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	r, e := decodificarRegistroVinculoRPT(raw, m)
	if e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	return r, nil
}

func (f *FuentePublicacionCategoriaRPTPostgreSQL) ConsultarPublicacionCategoriaRPT(ctx context.Context, p domain.PublicacionCategoriaRPT, cap vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.PublicacionCategoriaRPT, error) {
	if f == nil || f.pool == nil || ctx == nil {
		return domain.PublicacionCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	b, e := MaterialPublicacionCategoriaRPTPostgreSQL(ctx, f.pool, p)
	if e != nil {
		return domain.PublicacionCategoriaRPT{}, e
	}
	if !capacidadParaVinculoRPT(cap, ports.AccionConsultarPublicacionCategoriaRPT, ports.AudienciaConsultarPublicacionCategoriaRPT, p.CatalogoID, p.ModuloID, "catalogo_configurable", map[string]string{"catalogo_id": p.CatalogoID, "modulo_id": p.ModuloID}, b) {
		return domain.PublicacionCategoriaRPT{}, ports.ErrVinculoCategoriaRPTDenegado
	}
	tx, e := iniciarVinculoRPT(ctx, f.pool)
	if e != nil {
		return domain.PublicacionCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	defer revertirVinculoRPT(tx)
	args := append([]any{string(b)}, argumentosCapacidadVinculoRPT(cap)...)
	var raw string
	if e = tx.QueryRow(ctx, lecturaPublicacionRPT, args...).Scan(&raw); e != nil {
		return domain.PublicacionCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	encontrado, e := validarLecturaPublicacionRPT(raw, p)
	if e != nil {
		return domain.PublicacionCategoriaRPT{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return domain.PublicacionCategoriaRPT{}, clasificarErrorVinculoRPT(ctx, e)
	}
	if !encontrado {
		return domain.PublicacionCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoEncontrado
	}
	return p, nil
}

func iniciarVinculoRPT(ctx context.Context, p *pgxpool.Pool) (pgx.Tx, error) {
	tx, e := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`)
	if e != nil {
		revertirVinculoRPT(tx)
		return nil, e
	}
	return tx, nil
}
func revertirVinculoRPT(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
	defer cancel()
	_ = tx.Rollback(ctx)
}

func argumentosCapacidadVinculoRPT(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []any {
	return []any{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		strconv.FormatUint(a.PersonaVersion(), 10), strconv.FormatUint(a.PerfilVersion(), 10), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
}
func capacidadParaVinculoRPT(a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, audiencia, recurso, modulo, tipo string, ambitos map[string]string, material []byte) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	r := a.ResumenCapacidad()
	contexto, e := huellaContextoVinculoRPT(recurso, modulo, tipo, ambitos, material)
	return e == nil && r.Operacion() == accion && r.AudienciaConsumo() == audiencia && r.EfectoRef() == recurso && r.EfectoHuellaSHA256() == contexto
}

func huellaContextoVinculoRPT(recurso, modulo, tipo string, ambitos map[string]string, material []byte) (string, error) {
	h := sha256.Sum256(material)
	r := vecdomain.RecursoAutorizable{Referencia: recurso, ModuloID: modulo, Tipo: tipo, Ambitos: ambitos,
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	return r.HuellaContextoAutorizacionSHA256()
}

func clasificarErrorVinculoRPT(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pe *pgconn.PgError
	if errors.As(e, &pe) {
		switch pe.Code {
		case "22023", "22P02":
			return ports.ErrVinculoCategoriaRPTInvalido
		case "42501":
			return ports.ErrVinculoCategoriaRPTDenegado
		case "23505", "23514", "55000":
			return ports.ErrVinculoCategoriaRPTConflicto
		}
	}
	return ports.ErrVinculoCategoriaRPTNoDisponible
}

func jsonVinculoRPT(raw string, out any) error {
	if len(raw) < 2 || len(raw) > maximoRespuestaVinculoRPT || !json.Valid([]byte(raw)) {
		return ports.ErrVinculoCategoriaRPTNoDisponible
	}
	if e := json.Unmarshal([]byte(raw), out); e != nil {
		return ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return nil
}

func decodificarConsultaVinculoRPT(raw string, c ports.ConsultaVinculoCategoriaRPT) (ports.LecturaVinculoCategoriaRPT, bool, error) {
	var x struct {
		Encontrado        bool   `json:"encontrado"`
		VersionExpediente uint64 `json:"version_expediente"`
		Analisis          struct {
			Version      uint64 `json:"version"`
			ReciboRef    string `json:"recibo_ref"`
			HuellaSHA256 string `json:"huella_sha256"`
			CategoriaRef string `json:"categoria_ref"`
		} `json:"analisis"`
		Vinculo *ports.EstadoVinculoCategoriaRPT `json:"vinculo"`
	}
	if e := jsonVinculoRPT(raw, &x); e != nil {
		return ports.LecturaVinculoCategoriaRPT{}, false, e
	}
	if !x.Encontrado {
		return ports.LecturaVinculoCategoriaRPT{}, false, nil
	}
	l := ports.LecturaVinculoCategoriaRPT{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		Analisis: ports.AnclajeAnalisisCategoriaRPT{VersionExpediente: x.VersionExpediente, AnalisisVersion: x.Analisis.Version, AnalisisReciboRef: x.Analisis.ReciboRef, AnalisisHuellaSHA256: x.Analisis.HuellaSHA256, CategoriaRef: x.Analisis.CategoriaRef}}
	l.Vinculo = x.Vinculo
	if l.ValidarPara(c) != nil {
		return ports.LecturaVinculoCategoriaRPT{}, false, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return l, true, nil
}

func decodificarRegistroVinculoRPT(raw string, m ports.RegistroVinculoCategoriaRPT) (ports.ReciboVinculoCategoriaRPT, error) {
	var x struct {
		Replay  bool                             `json:"replay"`
		Recibo  ports.ReciboVinculoCategoriaRPT  `json:"recibo"`
		Vinculo *ports.EstadoVinculoCategoriaRPT `json:"vinculo"`
	}
	if e := jsonVinculoRPT(raw, &x); e != nil {
		return ports.ReciboVinculoCategoriaRPT{}, e
	}
	if x.Vinculo == nil || x.Recibo.ValidarPara(m) != nil || *x.Vinculo != x.Recibo.Vinculo {
		return ports.ReciboVinculoCategoriaRPT{}, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return x.Recibo, nil
}

func validarLecturaPublicacionRPT(raw string, p domain.PublicacionCategoriaRPT) (bool, error) {
	var x struct {
		Encontrado   bool  `json:"encontrado"`
		ConsumoNuevo *bool `json:"consumo_nuevo"`
		Datos        struct {
			Publicacion struct {
				CatalogoID        string `json:"catalogo_id"`
				Version           uint64 `json:"version"`
				HuellaSHA256      string `json:"huella_sha256"`
				DocumentoCanonico string `json:"documento_canonico"`
			} `json:"publicacion"`
			Entrada struct {
				Clave string `json:"clave"`
			} `json:"entrada"`
		} `json:"datos"`
	}
	if len(raw) < 2 || len(raw) > maximoRespuestaPublicacionRPT || !json.Valid([]byte(raw)) || json.Unmarshal([]byte(raw), &x) != nil {
		return false, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	if x.ConsumoNuevo == nil || !*x.ConsumoNuevo {
		return false, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	if !x.Encontrado {
		return false, nil
	}
	d := x.Datos.Publicacion
	var documento struct {
		ID       string `json:"id"`
		ModuloID string `json:"modulo_id"`
		Version  uint64 `json:"version"`
	}
	h := sha256.Sum256([]byte(d.DocumentoCanonico))
	if len(d.DocumentoCanonico) == 0 || len(d.DocumentoCanonico) > 16777216 || json.Unmarshal([]byte(d.DocumentoCanonico), &documento) != nil ||
		d.CatalogoID != p.CatalogoID || d.Version != p.CatalogoVersion || d.HuellaSHA256 != p.CatalogoHuella ||
		p.CatalogoHuella != hex.EncodeToString(h[:]) || documento.ID != p.CatalogoID || documento.ModuloID != p.ModuloID || documento.Version != p.CatalogoVersion ||
		x.Datos.Entrada.Clave != p.CategoriaID {
		return false, ports.ErrVinculoCategoriaRPTNoDisponible
	}
	return true, nil
}
