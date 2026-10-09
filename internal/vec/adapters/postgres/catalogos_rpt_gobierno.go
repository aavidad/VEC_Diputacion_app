package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	prepararPropuestaGobiernoRPTSQL = `SELECT vec_catalogos_configurables.preparar_propuesta_rpt($1::jsonb,$2::bytea,$3::jsonb)`
	prepararAvanceGobiernoRPTSQL    = `SELECT vec_catalogos_configurables.preparar_avance_rpt($1::text,$2::text,$3::bigint,$4::text)`
	proponerGobiernoRPTSQL          = `SELECT vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	aprobarGobiernoRPTSQL           = `SELECT vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	confirmarGobiernoRPTSQL         = `SELECT vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	maximoMaterialGobiernoRPT       = 18 << 20
)

var claseFuenteGobiernoRPT = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)

// FuenteGobiernoCategoriaRPT procede de configuración privada confiable, nunca
// del cuerpo HTTP. El constructor copia los bytes y coteja su huella.
type FuenteGobiernoCategoriaRPT struct {
	Bytes           []byte
	SHA256          string
	FuenteRef       string
	Clase           string
	ProcedenciaRef  string
	CustodiaRef     string
	OrganizacionRef string
	VigenteDesde    time.Time
	VigenteHasta    time.Time
}

type fuenteMetaGobiernoRPT struct {
	SHA256          string    `json:"sha256"`
	FuenteRef       string    `json:"fuente_ref"`
	Clase           string    `json:"clase"`
	ProcedenciaRef  string    `json:"procedencia_ref"`
	CustodiaRef     string    `json:"custodia_ref"`
	OrganizacionRef string    `json:"organizacion_ref"`
	VigenteDesde    time.Time `json:"vigente_desde"`
	VigenteHasta    time.Time `json:"vigente_hasta"`
}

type GestorGobiernoCategoriaRPTPostgreSQL struct {
	pool        iniciadorLecturaRPT
	descriptor  ports.DescriptorCatalogoRPT
	fuenteBytes []byte
	fuenteMeta  fuenteMetaGobiernoRPT
}

var _ ports.PreparadorGobiernoCategoriaRPT = (*GestorGobiernoCategoriaRPTPostgreSQL)(nil)
var _ ports.GestorGobiernoCategoriaRPT = (*GestorGobiernoCategoriaRPTPostgreSQL)(nil)

func NuevoGestorGobiernoCategoriaRPTPostgreSQL(pool *pgxpool.Pool, descriptor ports.DescriptorCatalogoRPT, fuente FuenteGobiernoCategoriaRPT) (*GestorGobiernoCategoriaRPTPostgreSQL, error) {
	return nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptor, fuente)
}

func nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool iniciadorLecturaRPT, descriptor ports.DescriptorCatalogoRPT, fuente FuenteGobiernoCategoriaRPT) (*GestorGobiernoCategoriaRPTPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	suma := sha256.Sum256(fuente.Bytes)
	if !claveRPT.MatchString(descriptor.CatalogoID) || !claveRPT.MatchString(descriptor.ModuloID) ||
		len(fuente.Bytes) < 1 || len(fuente.Bytes) > maximoDocumentoCanonicoRPT ||
		fuente.SHA256 != hex.EncodeToString(suma[:]) ||
		!referenciaFuenteGobiernoRPT(fuente.FuenteRef) ||
		!claseFuenteGobiernoRPT.MatchString(fuente.Clase) ||
		!referenciaFuenteGobiernoRPT(fuente.ProcedenciaRef) ||
		!referenciaFuenteGobiernoRPT(fuente.CustodiaRef) ||
		!referenciaFuenteGobiernoRPT(fuente.OrganizacionRef) ||
		!instanteRPTValido(fuente.VigenteDesde) || !instanteRPTValido(fuente.VigenteHasta) ||
		!fuente.VigenteDesde.Before(fuente.VigenteHasta) {
		return nil, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return &GestorGobiernoCategoriaRPTPostgreSQL{
		pool: pool, descriptor: descriptor, fuenteBytes: bytes.Clone(fuente.Bytes),
		fuenteMeta: fuenteMetaGobiernoRPT{fuente.SHA256, fuente.FuenteRef, fuente.Clase,
			fuente.ProcedenciaRef, fuente.CustodiaRef, fuente.OrganizacionRef,
			fuente.VigenteDesde.UTC(), fuente.VigenteHasta.UTC()},
	}, nil
}

func referenciaFuenteGobiernoRPT(s string) bool {
	return len(s) >= 3 && len(s) <= 320 && utf8.ValidString(s) && !strings.ContainsRune(s, '\x00')
}

type preparacionPropuestaGobiernoRPTWire struct {
	Contenido       domain.ContenidoGobiernoCategoriaRPT `json:"contenido"`
	HuellaSHA256    string                               `json:"huella_sha256"`
	FuenteSHA256    string                               `json:"fuente_sha256"`
	CategoriaID     string                               `json:"categoria_id"`
	OrganizacionRef string                               `json:"organizacion_ref"`
}

type preparacionAvanceGobiernoRPTWire struct {
	PropuestaRef    string `json:"propuesta_ref"`
	CatalogoID      string `json:"catalogo_id"`
	ModuloID        string `json:"modulo_id"`
	HuellaSHA256    string `json:"huella_sha256"`
	Revision        int64  `json:"revision"`
	CategoriaID     string `json:"categoria_id"`
	OrganizacionRef string `json:"organizacion_ref"`
	FuenteSHA256    string `json:"fuente_sha256"`
}

type materialGobiernoRPTWire struct {
	Accion           string                                `json:"accion"`
	PropuestaRef     string                                `json:"propuesta_ref"`
	ReciboRef        string                                `json:"recibo_ref"`
	Contenido        *domain.ContenidoGobiernoCategoriaRPT `json:"contenido,omitempty"`
	FuenteMeta       *fuenteMetaGobiernoRPT                `json:"fuente_meta,omitempty"`
	FuenteSHA256     string                                `json:"fuente_sha256,omitempty"`
	OrganizacionRef  string                                `json:"organizacion_ref,omitempty"`
	HuellaSHA256     string                                `json:"huella_sha256"`
	RevisionEsperada int64                                 `json:"revision_esperada"`
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) materialPropuesta(m ports.MaterialPropuestaGobiernoCategoriaRPT) (materialGobiernoRPTWire, error) {
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return materialGobiernoRPTWire{}, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaRecursoGobiernoRPT(m.PropuestaRef, g.descriptor.ModuloID) ||
		!referenciaUsoRPTValida(m.ReciboRef) || !huellaRPT.MatchString(m.HuellaSHA256) ||
		m.Contenido.Accion != domain.AccionGobiernoCategoriaRPTPublicar ||
		m.Contenido.CatalogoID != g.descriptor.CatalogoID || m.Contenido.ModuloID != g.descriptor.ModuloID ||
		m.Contenido.FuenteRef != g.fuenteMeta.FuenteRef || !m.Contenido.TamanoBorradorValido() {
		return materialGobiernoRPTWire{}, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return materialGobiernoRPTWire{Accion: "proponer", PropuestaRef: m.PropuestaRef, ReciboRef: m.ReciboRef,
		Contenido: &m.Contenido, FuenteMeta: &g.fuenteMeta, FuenteSHA256: g.fuenteMeta.SHA256,
		HuellaSHA256: m.HuellaSHA256, RevisionEsperada: 0}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) materialAvance(m ports.MaterialAvanceGobiernoCategoriaRPT, accion string) (materialGobiernoRPTWire, error) {
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return materialGobiernoRPTWire{}, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaRecursoGobiernoRPT(m.PropuestaRef, g.descriptor.ModuloID) || !referenciaUsoRPTValida(m.ReciboRef) ||
		!huellaRPT.MatchString(m.HuellaSHA256) || m.CatalogoID != g.descriptor.CatalogoID || m.ModuloID != g.descriptor.ModuloID ||
		(accion == "aprobar" && m.RevisionEsperada != 1) || (accion == "confirmar" && m.RevisionEsperada != 2) {
		return materialGobiernoRPTWire{}, ports.ErrGobiernoCategoriaRPTInvalido
	}
	return materialGobiernoRPTWire{Accion: accion, PropuestaRef: m.PropuestaRef, ReciboRef: m.ReciboRef,
		HuellaSHA256: m.HuellaSHA256, RevisionEsperada: m.RevisionEsperada,
		FuenteSHA256: g.fuenteMeta.SHA256, OrganizacionRef: g.fuenteMeta.OrganizacionRef}, nil
}

func referenciaRecursoGobiernoRPT(ref, modulo string) bool {
	return (domain.RecursoAutorizable{Referencia: ref, ModuloID: modulo, Tipo: ports.TipoRecursoGobiernoCategoriaRPT}).Validar() == nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararPropuestaGobiernoCategoriaRPT(ctx context.Context, b ports.BorradorPropuestaGobiernoCategoriaRPT) (ports.PreparacionPropuestaGobiernoCategoriaRPT, error) {
	var cero ports.PreparacionPropuestaGobiernoCategoriaRPT
	if ctx == nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if !referenciaRecursoGobiernoRPT(b.PropuestaRef, g.descriptor.ModuloID) || !referenciaUsoRPTValida(b.ReciboRef) ||
		b.Contenido.CatalogoID != g.descriptor.CatalogoID || b.Contenido.ModuloID != g.descriptor.ModuloID ||
		b.Contenido.FuenteRef != g.fuenteMeta.FuenteRef || b.Contenido.Accion != domain.AccionGobiernoCategoriaRPTPublicar ||
		!b.Contenido.TamanoBorradorValido() || b.Contenido.PreimagenesHuellaSHA256 != "" {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	contenido, err := json.Marshal(b.Contenido)
	if err != nil || len(contenido) > maximoMaterialGobiernoRPT {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	defer borrarPiezasRPT(contenido)
	meta, err := json.Marshal(g.fuenteMeta)
	if err != nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err = tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	var respuesta []byte
	if err = tx.QueryRow(ctx, prepararPropuestaGobiernoRPTSQL, string(contenido), g.fuenteBytes, string(meta)).Scan(&respuesta); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	defer borrarPiezasRPT(respuesta)
	var p preparacionPropuestaGobiernoRPTWire
	if len(respuesta) > maximoMaterialGobiernoRPT || numeroClavesRPT(respuesta) != 5 || decodificarRPT(respuesta, &p) != nil ||
		!huellaRPT.MatchString(p.HuellaSHA256) || p.FuenteSHA256 != g.fuenteMeta.SHA256 ||
		p.OrganizacionRef != g.fuenteMeta.OrganizacionRef || !claveRPT.MatchString(p.CategoriaID) ||
		p.Contenido.CatalogoID != b.Contenido.CatalogoID || p.Contenido.ModuloID != b.Contenido.ModuloID ||
		p.Contenido.FuenteRef != b.Contenido.FuenteRef {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	comparacion := p.Contenido
	comparacion.PreimagenesHuellaSHA256 = ""
	if !reflect.DeepEqual(comparacion, b.Contenido) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	m := ports.MaterialPropuestaGobiernoCategoriaRPT{PropuestaRef: b.PropuestaRef, Contenido: p.Contenido, HuellaSHA256: p.HuellaSHA256, ReciboRef: b.ReciboRef}
	w, err := g.materialPropuesta(m)
	if err != nil {
		return cero, err
	}
	a, err := g.autorizable(ctx, tx, ports.AccionProponerGobiernoCategoriaRPT, m.PropuestaRef, m.HuellaSHA256, w)
	if err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return ports.PreparacionPropuestaGobiernoCategoriaRPT{Material: m, Autorizable: a}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararAprobacionGobiernoCategoriaRPT(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	return g.prepararAvance(ctx, m, "aprobar", ports.AccionAprobarGobiernoCategoriaRPT)
}
func (g *GestorGobiernoCategoriaRPTPostgreSQL) PrepararConfirmacionGobiernoCategoriaRPT(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	return g.prepararAvance(ctx, m, "confirmar", ports.AccionConfirmarGobiernoCategoriaRPT)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) prepararAvance(ctx context.Context, m ports.MaterialAvanceGobiernoCategoriaRPT, acto, accion string) (ports.PreparacionGobiernoCategoriaRPT, error) {
	var cero ports.PreparacionGobiernoCategoriaRPT
	if ctx == nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	w, err := g.materialAvance(m, acto)
	if err != nil {
		return cero, err
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err = tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	var respuesta []byte
	if err = tx.QueryRow(ctx, prepararAvanceGobiernoRPTSQL, m.PropuestaRef, m.HuellaSHA256, m.RevisionEsperada, acto).Scan(&respuesta); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	defer borrarPiezasRPT(respuesta)
	var p preparacionAvanceGobiernoRPTWire
	if len(respuesta) > 4096 || numeroClavesRPT(respuesta) != 8 || decodificarRPT(respuesta, &p) != nil ||
		p.PropuestaRef != m.PropuestaRef || p.CatalogoID != m.CatalogoID || p.ModuloID != m.ModuloID ||
		p.HuellaSHA256 != m.HuellaSHA256 || !claveRPT.MatchString(p.CategoriaID) ||
		p.OrganizacionRef != g.fuenteMeta.OrganizacionRef || p.FuenteSHA256 != g.fuenteMeta.SHA256 ||
		(p.Revision < m.RevisionEsperada || p.Revision > 3) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	a, err := g.autorizable(ctx, tx, accion, m.PropuestaRef, m.HuellaSHA256, w)
	if err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return a, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) autorizable(ctx context.Context, tx pgx.Tx, accion, ref, huella string, w materialGobiernoRPTWire) (ports.PreparacionGobiernoCategoriaRPT, error) {
	var cero ports.PreparacionGobiernoCategoriaRPT
	material, err := json.Marshal(w)
	if err != nil || len(material) > maximoMaterialGobiernoRPT {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	defer borrarPiezasRPT(material)
	var h string
	if err = tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&h); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if !huellaRPT.MatchString(h) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	r := domain.RecursoAutorizable{Referencia: ref, ModuloID: g.descriptor.ModuloID, Tipo: ports.TipoRecursoGobiernoCategoriaRPT,
		Ambitos: map[string]string{"catalogo_id": g.descriptor.CatalogoID, "modulo_id": g.descriptor.ModuloID}, Atributos: map[string]string{"material_sha256": h}}
	if r.Validar() != nil {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	return ports.PreparacionGobiernoCategoriaRPT{Accion: accion, Finalidad: ports.FinalidadGobiernoCategoriaRPT,
		Audiencia: ports.AudienciaGobiernoCategoriaRPT, Recurso: r, HuellaPropuesta: huella}, nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) ProponerGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenPropuestaGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	w, err := g.materialPropuesta(o.Material)
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, w, g.fuenteBytes, ports.AccionProponerGobiernoCategoriaRPT, proponerGobiernoRPTSQL, 1, domain.EstadoGobiernoCategoriaRPTPropuesta)
}
func (g *GestorGobiernoCategoriaRPTPostgreSQL) AprobarGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	w, err := g.materialAvance(o.Material, "aprobar")
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, w, nil, ports.AccionAprobarGobiernoCategoriaRPT, aprobarGobiernoRPTSQL, 2, domain.EstadoGobiernoCategoriaRPTAprobada)
}
func (g *GestorGobiernoCategoriaRPTPostgreSQL) ConfirmarGobiernoCategoriaRPT(ctx context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	w, err := g.materialAvance(o.Material, "confirmar")
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, w, nil, ports.AccionConfirmarGobiernoCategoriaRPT, confirmarGobiernoRPTSQL, 3, domain.EstadoGobiernoCategoriaRPTConfirmada)
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) validarAutorizacion(s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, ref string) (string, error) {
	if a.ValidarEstructura() != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	d, err := s.Datos()
	if err != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	r := d.Recurso
	if d.Accion != accion || d.Finalidad != ports.FinalidadGobiernoCategoriaRPT || r.Referencia != ref || r.ModuloID != g.descriptor.ModuloID ||
		r.Tipo != ports.TipoRecursoGobiernoCategoriaRPT || len(r.Ambitos) != 2 || r.Ambitos["catalogo_id"] != g.descriptor.CatalogoID ||
		r.Ambitos["modulo_id"] != g.descriptor.ModuloID || len(r.Atributos) != 1 || !huellaRPT.MatchString(r.Atributos["material_sha256"]) || r.Validar() != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	z := a.ResumenCapacidad()
	if err != nil || z.ValidarEstructura() != nil || z.Operacion() != accion || z.AudienciaConsumo() != ports.AudienciaGobiernoCategoriaRPT ||
		z.EfectoRef() != ref || z.EfectoHuellaSHA256() != h || ligaduraSolicitudUsoRPT(d, s, a, z, h) != nil {
		return "", ports.ErrGobiernoCategoriaRPTDenegado
	}
	return r.Atributos["material_sha256"], nil
}

func (g *GestorGobiernoCategoriaRPTPostgreSQL) ejecutar(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, w materialGobiernoRPTWire, fuente []byte, accion, sql string, revision int64, estado string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	if ctx == nil {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	material, err := json.Marshal(w)
	if err != nil || len(material) > maximoMaterialGobiernoRPT {
		return cero, ports.ErrGobiernoCategoriaRPTInvalido
	}
	defer borrarPiezasRPT(material)
	h, err := g.validarAutorizacion(s, a, accion, w.PropuestaRef)
	if err != nil {
		return cero, err
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	defer revertirUsoRPT(tx)
	if _, err = tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	var actual string
	if err = tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&actual); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	if actual != h {
		return cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	piezas := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer borrarPiezasRPT(piezas...)
	var bruto []byte
	err = tx.QueryRow(ctx, sql, string(material), fuente, piezas[0], piezas[1], piezas[2], piezas[3],
		strconv.FormatUint(a.PersonaVersion(), 10), strconv.FormatUint(a.PerfilVersion(), 10), piezas[4], piezas[5], piezas[6], piezas[7]).Scan(&bruto)
	if err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	defer borrarPiezasRPT(bruto)
	r, err := decodificarResultadoGobiernoRPT(bruto, a, w, accion, revision, estado)
	if err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorGobiernoRPT(ctx, err)
	}
	return r, nil
}

type resultadoGobiernoRPTWire struct {
	ports.ResultadoGobiernoCategoriaRPT
	RegistradaEn time.Time `json:"registrada_en"`
}

func decodificarResultadoGobiernoRPT(bruto []byte, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, w materialGobiernoRPTWire, accion string, revision int64, estado string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	var x resultadoGobiernoRPTWire
	if len(bruto) < 2 || len(bruto) > 16384 || numeroClavesRPT(bruto) != 10 || decodificarRPT(bruto, &x) != nil {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	r := x.ResultadoGobiernoCategoriaRPT
	e := r.Evidencia
	z := a.ResumenCapacidad()
	if r.PropuestaRef != w.PropuestaRef || r.HuellaSHA256 != w.HuellaSHA256 || r.ReciboRef != w.ReciboRef ||
		r.Accion != w.Accion || r.Revision != revision || r.Estado != estado || r.Version < 1 || r.Version > maximoVersionMaterialRPT ||
		((accion == ports.AccionConfirmarGobiernoCategoriaRPT && r.RevisionCategoria < 1) ||
			(accion != ports.AccionConfirmarGobiernoCategoriaRPT && r.RevisionCategoria != 0)) ||
		!instanteRPTValido(x.RegistradaEn) || e.DecisionRef != z.DecisionRef() || e.EfectoRef != z.EfectoRef() ||
		e.HuellaEfectoSHA256 != z.EfectoHuellaSHA256() || !huellaRPT.MatchString(e.ConsumoHuellaSHA256) ||
		e.AuditoriaRef == "" || !e.ConsumoNuevo || !instanteRPTValido(e.ConsumidaEn) ||
		e.ConsumidaEn.Before(z.EmitidaEn()) || !e.ConsumidaEn.Before(z.ExpiraEn()) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	return r, nil
}

func errorGobiernoRPT(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		return causa
	}
	var pg *pgconn.PgError
	if errors.As(causa, &pg) {
		switch pg.Code {
		case "42501", "PDI03":
			return ports.ErrGobiernoCategoriaRPTDenegado
		case "22023":
			return ports.ErrGobiernoCategoriaRPTInvalido
		case "23505", "40001", "55P03", "55000":
			return ports.ErrGobiernoCategoriaRPTConflicto
		}
	}
	return ports.ErrGobiernoCategoriaRPTNoDisponible
}
