package postgres

import (
	"context"
	"encoding/json"
	"errors"
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
	accionReservarUsoRPT    = "vec.catalogos.categorias.reservar_uso"
	accionConfirmarUsoRPT   = "vec.catalogos.categorias.confirmar_uso"
	accionCancelarUsoRPT    = "vec.catalogos.categorias.cancelar_uso"
	audienciaUsosRPT        = "vec_catalogos_configurables.usos_categorias.v1"
	finalidadUsosRPT        = "vincular_categoria_a_operacion"
	consultaReservarUsoRPT  = `SELECT vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaConfirmarUsoRPT = `SELECT vec_autorizacion_atestada_v3.confirmar_uso_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaCancelarUsoRPT  = `SELECT vec_autorizacion_atestada_v3.cancelar_uso_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
)

// GestorUsosCategoriaRPTPostgreSQL usa un único LOGIN ejecutor de CT, Bolsa o
// Personal. La fachada SQL vuelve a comprobar esa familia y consume V3 antes
// del efecto; este adaptador nunca elige el consumidor desde JSON del cliente.
type GestorUsosCategoriaRPTPostgreSQL struct {
	pool       iniciadorLecturaRPT
	descriptor ports.DescriptorCatalogoRPT
	consumidor string
}

var _ ports.GestorUsosCategoriaRPT = (*GestorUsosCategoriaRPTPostgreSQL)(nil)

func NuevoGestorUsosCategoriaRPTPostgreSQL(pool *pgxpool.Pool, descriptor ports.DescriptorCatalogoRPT, consumidor string) (*GestorUsosCategoriaRPTPostgreSQL, error) {
	return nuevoGestorUsosCategoriaRPTPostgreSQL(pool, descriptor, consumidor)
}

func nuevoGestorUsosCategoriaRPTPostgreSQL(pool iniciadorLecturaRPT, descriptor ports.DescriptorCatalogoRPT, consumidor string) (*GestorUsosCategoriaRPTPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ports.ErrUsoCategoriaRPTNoDisponible
	}
	if !claveCatalogoRPT.MatchString(descriptor.CatalogoID) ||
		!claveCatalogoRPT.MatchString(descriptor.ModuloID) ||
		(consumidor != "contratacion_temporal" && consumidor != "bolsa" && consumidor != "personal") {
		return nil, ports.ErrUsoCategoriaRPTDenegado
	}
	return &GestorUsosCategoriaRPTPostgreSQL{pool: pool, descriptor: descriptor, consumidor: consumidor}, nil
}

type materialReservaUsoRPTWire struct {
	CatalogoID       string `json:"catalogo_id"`
	ModuloID         string `json:"modulo_id"`
	Consumidor       string `json:"consumidor"`
	UsoRef           string `json:"uso_ref"`
	CategoriaID      string `json:"categoria_id"`
	Version          int    `json:"version"`
	HuellaSHA256     string `json:"huella_sha256"`
	ReservaReciboRef string `json:"reserva_recibo_ref"`
}

type materialTerminalUsoRPTWire struct {
	CatalogoID        string `json:"catalogo_id"`
	ModuloID          string `json:"modulo_id"`
	Consumidor        string `json:"consumidor"`
	UsoRef            string `json:"uso_ref"`
	CategoriaID       string `json:"categoria_id"`
	Version           int    `json:"version"`
	HuellaSHA256      string `json:"huella_sha256"`
	ReservaReciboRef  string `json:"reserva_recibo_ref"`
	TerminalReciboRef string `json:"terminal_recibo_ref"`
	EvidenciaRef      string `json:"evidencia_ref"`
	EvidenciaSHA256   string `json:"evidencia_sha256"`
}

func (g *GestorUsosCategoriaRPTPostgreSQL) materialReserva(m ports.MaterialReservaUsoCategoriaRPT) (materialReservaUsoRPTWire, error) {
	var cero materialReservaUsoRPTWire
	if g == nil || valorNuloPostgreSQL(g.pool) {
		return cero, ports.ErrUsoCategoriaRPTNoDisponible
	}
	if m.Consumidor != g.consumidor || !g.referenciaRecursoUsoRPTValida(m.UsoRef) ||
		!claveRPT.MatchString(m.CategoriaID) || m.Publicacion.CatalogoID != g.descriptor.CatalogoID ||
		m.Publicacion.Version < 1 || m.Publicacion.Version > maximoVersionMaterialRPT ||
		!huellaRPT.MatchString(m.Publicacion.HuellaSHA256) || !referenciaUsoRPTValida(m.ReservaReciboRef) {
		return cero, ports.ErrUsoCategoriaRPTInvalido
	}
	return materialReservaUsoRPTWire{
		CatalogoID: g.descriptor.CatalogoID, ModuloID: g.descriptor.ModuloID,
		Consumidor: g.consumidor, UsoRef: m.UsoRef, CategoriaID: m.CategoriaID,
		Version: m.Publicacion.Version, HuellaSHA256: m.Publicacion.HuellaSHA256,
		ReservaReciboRef: m.ReservaReciboRef,
	}, nil
}

func (g *GestorUsosCategoriaRPTPostgreSQL) materialTerminal(m ports.MaterialTerminalUsoCategoriaRPT) (materialTerminalUsoRPTWire, error) {
	var cero materialTerminalUsoRPTWire
	base, err := g.materialReserva(m.Reserva)
	if err != nil {
		return cero, err
	}
	if !referenciaUsoRPTValida(m.TerminalReciboRef) || m.TerminalReciboRef == m.Reserva.ReservaReciboRef ||
		!referenciaUsoRPTValida(m.EvidenciaRef) || !huellaRPT.MatchString(m.EvidenciaSHA256) {
		return cero, ports.ErrUsoCategoriaRPTInvalido
	}
	return materialTerminalUsoRPTWire{
		CatalogoID: base.CatalogoID, ModuloID: base.ModuloID, Consumidor: base.Consumidor,
		UsoRef: base.UsoRef, CategoriaID: base.CategoriaID, Version: base.Version,
		HuellaSHA256: base.HuellaSHA256, ReservaReciboRef: base.ReservaReciboRef,
		TerminalReciboRef: m.TerminalReciboRef, EvidenciaRef: m.EvidenciaRef,
		EvidenciaSHA256: m.EvidenciaSHA256,
	}, nil
}

func referenciaUsoRPTValida(valor string) bool {
	return len(valor) >= 3 && len(valor) <= 160 && utf8.ValidString(valor) &&
		!strings.ContainsRune(valor, '\x00')
}

// uso_ref también es la referencia exacta del recurso V3. Un uso histórico
// legible con referencia Unicode no puede iniciar otra operación V3; no se
// reescribe su historia. Recibos y evidencia siguen siendo UTF-8 opaco.
func (g *GestorUsosCategoriaRPTPostgreSQL) referenciaRecursoUsoRPTValida(valor string) bool {
	return referenciaUsoRPTValida(valor) && (domain.RecursoAutorizable{
		Referencia: valor, ModuloID: g.descriptor.ModuloID, Tipo: tipoUsoRPT,
	}).Validar() == nil
}

func (g *GestorUsosCategoriaRPTPostgreSQL) ReservarUsoCategoriaRPT(ctx context.Context, o ports.OrdenReservaUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	if err := comprobarContextoUsoRPT(ctx); err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	m, err := g.materialReserva(o.Material)
	if err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, o.Material, "", "", "",
		accionReservarUsoRPT, consultaReservarUsoRPT, m)
}

func (g *GestorUsosCategoriaRPTPostgreSQL) ConfirmarUsoCategoriaRPT(ctx context.Context, o ports.OrdenConfirmacionUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	if err := comprobarContextoUsoRPT(ctx); err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	m, err := g.materialTerminal(o.Material)
	if err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, o.Material.Reserva,
		o.Material.TerminalReciboRef, o.Material.EvidenciaRef, o.Material.EvidenciaSHA256,
		accionConfirmarUsoRPT, consultaConfirmarUsoRPT, m)
}

func (g *GestorUsosCategoriaRPTPostgreSQL) CancelarUsoCategoriaRPT(ctx context.Context, o ports.OrdenCancelacionUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	if err := comprobarContextoUsoRPT(ctx); err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	m, err := g.materialTerminal(o.Material)
	if err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, err
	}
	return g.ejecutar(ctx, o.Solicitud, o.Autorizacion, o.Material.Reserva,
		o.Material.TerminalReciboRef, o.Material.EvidenciaRef, o.Material.EvidenciaSHA256,
		accionCancelarUsoRPT, consultaCancelarUsoRPT, m)
}

func comprobarContextoUsoRPT(ctx context.Context) error {
	if ctx == nil {
		return ports.ErrUsoCategoriaRPTInvalido
	}
	return ctx.Err()
}

func (g *GestorUsosCategoriaRPTPostgreSQL) validarAutorizacion(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	accion, usoRef string,
) (string, error) {
	if g == nil || valorNuloPostgreSQL(g.pool) || autorizacion.ValidarEstructura() != nil {
		return "", ports.ErrUsoCategoriaRPTDenegado
	}
	d, err := solicitud.Datos()
	if err != nil || d.Recurso.Validar() != nil {
		return "", ports.ErrUsoCategoriaRPTDenegado
	}
	r := d.Recurso
	if d.Accion != accion || d.Finalidad != finalidadUsosRPT ||
		r.Referencia != usoRef || r.ModuloID != g.descriptor.ModuloID || r.Tipo != tipoUsoRPT ||
		len(r.Ambitos) != 3 || r.Ambitos["catalogo_id"] != g.descriptor.CatalogoID ||
		r.Ambitos["modulo_id"] != g.descriptor.ModuloID || r.Ambitos["consumidor"] != g.consumidor ||
		len(r.Atributos) != 1 || !huellaRPT.MatchString(r.Atributos["material_sha256"]) {
		return "", ports.ErrUsoCategoriaRPTDenegado
	}
	huellaRecurso, err := r.HuellaContextoAutorizacionSHA256()
	resumen := autorizacion.ResumenCapacidad()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaUsosRPT ||
		resumen.EfectoRef() != usoRef || resumen.EfectoHuellaSHA256() != huellaRecurso {
		return "", ports.ErrUsoCategoriaRPTDenegado
	}
	return r.Atributos["material_sha256"], nil
}

func (g *GestorUsosCategoriaRPTPostgreSQL) ejecutar(
	ctx context.Context, solicitud domain.SolicitudAutorizacionLigadaV3,
	autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	base ports.MaterialReservaUsoCategoriaRPT,
	terminalReciboRef, evidenciaRef, evidenciaSHA256, accion, funcion string,
	materialWire any,
) (ports.ResultadoUsoCategoriaRPT, error) {
	var cero ports.ResultadoUsoCategoriaRPT
	if ctx == nil {
		return cero, ports.ErrUsoCategoriaRPTInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	material, err := json.Marshal(materialWire)
	if err != nil || len(material) == 0 || len(material) > 4096 {
		return cero, ports.ErrUsoCategoriaRPTInvalido
	}
	defer borrarPiezasRPT(material)
	huellaEsperada, err := g.validarAutorizacion(solicitud, autorizacion, accion, base.UsoRef)
	if err != nil {
		return cero, err
	}
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, errorUsoCategoriaRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, ports.ErrUsoCategoriaRPTNoDisponible
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	if _, err := tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, errorUsoCategoriaRPT(ctx, err)
	}
	// La huella de jsonb::text procede de PostgreSQL en esta misma
	// transacción. No se sustituye por SHA de json.Marshal de Go.
	var huellaMaterial string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&huellaMaterial); err != nil {
		return cero, errorUsoCategoriaRPT(ctx, err)
	}
	if !huellaRPT.MatchString(huellaMaterial) || huellaMaterial != huellaEsperada {
		return cero, ports.ErrUsoCategoriaRPTDenegado
	}
	piezas := [][]byte{
		autorizacion.CapacidadCanonica(), autorizacion.DecisionCanonica(), autorizacion.MotivoCanonico(),
		autorizacion.ContextoActorCanonico(), autorizacion.PayloadVECAD3(), autorizacion.SobreCOSESign1(),
		autorizacion.EvidenciaVerificacion(), autorizacion.RaizPublicaSPKI(),
	}
	defer borrarPiezasRPT(piezas...)
	var respuesta []byte
	err = tx.QueryRow(ctx, funcion, string(material), piezas[0], piezas[1], piezas[2], piezas[3],
		strconv.FormatUint(autorizacion.PersonaVersion(), 10), strconv.FormatUint(autorizacion.PerfilVersion(), 10),
		piezas[4], piezas[5], piezas[6], piezas[7]).Scan(&respuesta)
	if err != nil {
		return cero, errorUsoCategoriaRPT(ctx, err)
	}
	defer borrarPiezasRPT(respuesta)
	resultado, err := g.decRPTUso(respuesta, autorizacion, base, terminalReciboRef,
		evidenciaRef, evidenciaSHA256, accion)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorUsoCategoriaRPT(ctx, err)
	}
	return resultado, nil
}

type reciboUsoRPTWire struct {
	DecisionRef         string          `json:"decision_ref"`
	EfectoRef           string          `json:"efecto_ref"`
	HuellaEfectoSHA256  string          `json:"huella_efecto_sha256"`
	ConsumoHuellaSHA256 string          `json:"consumo_huella_sha256"`
	AuditoriaRef        string          `json:"auditoria_ref"`
	ConsumidaEn         time.Time       `json:"consumida_en"`
	ConsumoNuevo        bool            `json:"consumo_nuevo"`
	ReciboRef           string          `json:"recibo_ref"`
	Uso                 json.RawMessage `json:"uso"`
}

func (g *GestorUsosCategoriaRPTPostgreSQL) decRPTUso(
	bruto []byte, autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	base ports.MaterialReservaUsoCategoriaRPT,
	terminalReciboRef, evidenciaRef, evidenciaSHA256, accion string,
) (ports.ResultadoUsoCategoriaRPT, error) {
	var cero ports.ResultadoUsoCategoriaRPT
	var recibo reciboUsoRPTWire
	if len(bruto) < 2 || len(bruto) > maximoRespuestaLecturaRPT ||
		numeroClavesRPT(bruto) != 9 || decodificarRPT(bruto, &recibo) != nil ||
		autorizacion.ValidarEstructura() != nil {
		return cero, ports.ErrUsoCategoriaRPTNoConfiable
	}
	resumen := autorizacion.ResumenCapacidad()
	esperado := base.ReservaReciboRef
	if accion != accionReservarUsoRPT {
		esperado = terminalReciboRef
	}
	if recibo.DecisionRef != resumen.DecisionRef() || recibo.EfectoRef != resumen.EfectoRef() ||
		recibo.HuellaEfectoSHA256 != resumen.EfectoHuellaSHA256() ||
		!huellaRPT.MatchString(recibo.ConsumoHuellaSHA256) || recibo.AuditoriaRef == "" ||
		!recibo.ConsumoNuevo || !instanteRPTValido(recibo.ConsumidaEn) ||
		recibo.ConsumidaEn.Before(resumen.EmitidaEn()) || !recibo.ConsumidaEn.Before(resumen.ExpiraEn()) ||
		recibo.ReciboRef != esperado || numeroClavesRPT(recibo.Uso) != 12 {
		return cero, ports.ErrUsoCategoriaRPTNoConfiable
	}
	consulta := ports.ConsultaUsoCategoriaRPT{
		Consumidor: base.Consumidor, UsoRef: base.UsoRef, ReservaReciboRef: base.ReservaReciboRef,
	}
	uso, err := decodificarUsoCategoriaRPT(recibo.Uso, consulta, g.descriptor)
	if err != nil || !uso.Encontrado || uso.Uso == nil ||
		uso.Uso.CategoriaID != base.CategoriaID || uso.Uso.Publicacion != base.Publicacion {
		return cero, ports.ErrUsoCategoriaRPTNoConfiable
	}
	u := uso.Uso
	if accion == accionReservarUsoRPT {
		if terminalReciboRef != "" || evidenciaRef != "" || evidenciaSHA256 != "" {
			return cero, ports.ErrUsoCategoriaRPTNoConfiable
		}
	} else {
		estado := "confirmado"
		if accion == accionCancelarUsoRPT {
			estado = "cancelado"
		}
		if u.Estado != estado || u.TerminalReciboRef == nil || *u.TerminalReciboRef != terminalReciboRef ||
			!referenciaUsoRPTValida(evidenciaRef) || !huellaRPT.MatchString(evidenciaSHA256) {
			return cero, ports.ErrUsoCategoriaRPTNoConfiable
		}
	}
	uso.Evidencia = ports.EvidenciaLecturaRPT{
		DecisionRef: recibo.DecisionRef, EfectoRef: recibo.EfectoRef,
		HuellaEfectoSHA256:  recibo.HuellaEfectoSHA256,
		ConsumoHuellaSHA256: recibo.ConsumoHuellaSHA256,
		AuditoriaRef:        recibo.AuditoriaRef, ConsumidaEn: recibo.ConsumidaEn.UTC(), ConsumoNuevo: true,
	}
	return uso, nil
}

func errorUsoCategoriaRPT(ctx context.Context, causa error) error {
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
			return ports.ErrUsoCategoriaRPTDenegado
		case "22023":
			return ports.ErrUsoCategoriaRPTInvalido
		case "23505", "40001", "55P03", "55000":
			return ports.ErrUsoCategoriaRPTConflicto
		}
	}
	return ports.ErrUsoCategoriaRPTNoDisponible
}
