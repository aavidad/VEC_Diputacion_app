package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const maximoDocumentoCanonicoRPT = 16 << 20
const maximoRespuestaLecturaRPT = 48 << 20
const maximoVersionMaterialRPT = math.MaxInt32

const (
	accionListarCategoriasRPT = "vec.catalogos.categorias.listar_habilitadas"
	accionLeerPublicacionRPT  = "vec.catalogos.categorias.consultar_historica"
	accionConsultarUsoRPT     = "vec.catalogos.categorias.consultar_uso"
	audienciaLecturaRPT       = "vec_catalogos_configurables.lectura_categorias.v1"
	finalidadLecturaRPT       = "consultar_categorias_rpt"
	tipoCatalogoRPT           = "catalogo_configurable"
	tipoUsoRPT                = "uso_categoria"
	consultaListaRPT          = `SELECT vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaPublicacionRPT    = `SELECT vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaUsoRPT            = `SELECT vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada($1::jsonb,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaHuellaMaterialRPT = `SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex')`
	configurarLecturaRPT      = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
)

var huellaRPT = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveRPT = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)
var claveCatalogoRPT = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)

// LectorCategoriasRPTPostgreSQL se compone con un descriptor gobernado y un
// pool técnico propio. El descriptor solo restringe: la decisión V3 fresca y
// su consumo nominal siguen siendo necesarios en cada método.
type LectorCategoriasRPTPostgreSQL struct {
	pool       iniciadorLecturaRPT
	descriptor ports.DescriptorCatalogoRPT
}

type iniciadorLecturaRPT interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

var _ ports.LectorCategoriasRPT = (*LectorCategoriasRPTPostgreSQL)(nil)

func NuevoLectorCategoriasRPTPostgreSQL(pool *pgxpool.Pool, descriptor ports.DescriptorCatalogoRPT) (*LectorCategoriasRPTPostgreSQL, error) {
	return nuevoLectorCategoriasRPTPostgreSQL(pool, descriptor)
}

func nuevoLectorCategoriasRPTPostgreSQL(pool iniciadorLecturaRPT, descriptor ports.DescriptorCatalogoRPT) (*LectorCategoriasRPTPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, ports.ErrLecturaRPTNoDisponible
	}
	if !claveCatalogoRPT.MatchString(descriptor.CatalogoID) ||
		!claveCatalogoRPT.MatchString(descriptor.ModuloID) {
		return nil, ports.ErrLecturaRPTDenegada
	}
	return &LectorCategoriasRPTPostgreSQL{pool: pool, descriptor: descriptor}, nil
}

type materialListaRPT struct {
	CatalogoID        string  `json:"catalogo_id"`
	ModuloID          string  `json:"modulo_id"`
	CursorCategoriaID *string `json:"cursor_categoria_id"`
	Limite            int     `json:"limite"`
}

type materialPublicacionRPT struct {
	CatalogoID   string `json:"catalogo_id"`
	ModuloID     string `json:"modulo_id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	CategoriaID  string `json:"categoria_id"`
}

type materialUsoRPT struct {
	CatalogoID       string `json:"catalogo_id"`
	ModuloID         string `json:"modulo_id"`
	Consumidor       string `json:"consumidor"`
	UsoRef           string `json:"uso_ref"`
	ReservaReciboRef string `json:"reserva_recibo_ref"`
}

func (l *LectorCategoriasRPTPostgreSQL) ListarCategoriasHabilitadasRPT(ctx context.Context, o ports.OrdenCategoriasHabilitadasRPT) (ports.ResultadoCategoriasHabilitadasRPT, error) {
	var cero ports.ResultadoCategoriasHabilitadasRPT
	if l == nil || o.Consulta.CatalogoID != l.descriptor.CatalogoID ||
		o.Consulta.Limite < 1 || o.Consulta.Limite > 100 ||
		(o.Consulta.CursorCategoriaID != "" && !claveRPT.MatchString(o.Consulta.CursorCategoriaID)) {
		return cero, ports.ErrLecturaRPTInvalida
	}
	var cursor *string
	if o.Consulta.CursorCategoriaID != "" {
		cursor = &o.Consulta.CursorCategoriaID
	}
	material, err := json.Marshal(materialListaRPT{
		CatalogoID: l.descriptor.CatalogoID, ModuloID: l.descriptor.ModuloID,
		CursorCategoriaID: cursor, Limite: o.Consulta.Limite,
	})
	if err != nil {
		return cero, ports.ErrLecturaRPTInvalida
	}
	resultado, evidencia, err := ejecutarLecturaRPT(l, ctx, o.Solicitud, o.Autorizacion,
		accionListarCategoriasRPT, tipoCatalogoRPT, l.descriptor.CatalogoID, "", consultaListaRPT, material,
		func(datos []byte) (ports.ResultadoCategoriasHabilitadasRPT, error) {
			return decodificarListaRPT(datos, o.Consulta, l.descriptor)
		})
	resultado.Evidencia = evidencia
	return resultado, err
}

func (l *LectorCategoriasRPTPostgreSQL) LeerPublicacionCategoriaRPT(ctx context.Context, o ports.OrdenPublicacionCategoriaRPT) (ports.ResultadoPublicacionCategoriaRPT, error) {
	var cero ports.ResultadoPublicacionCategoriaRPT
	consulta := o.Consulta
	if l == nil || consulta.Referencia.CatalogoID != l.descriptor.CatalogoID ||
		consulta.Referencia.Version < 1 || consulta.Referencia.Version > maximoVersionMaterialRPT ||
		!huellaRPT.MatchString(consulta.Referencia.HuellaSHA256) ||
		!claveRPT.MatchString(consulta.CategoriaID) {
		return cero, ports.ErrLecturaRPTInvalida
	}
	material, err := json.Marshal(materialPublicacionRPT{
		CatalogoID: l.descriptor.CatalogoID, ModuloID: l.descriptor.ModuloID,
		Version: consulta.Referencia.Version, HuellaSHA256: consulta.Referencia.HuellaSHA256,
		CategoriaID: consulta.CategoriaID,
	})
	if err != nil {
		return cero, ports.ErrLecturaRPTInvalida
	}
	resultado, evidencia, err := ejecutarLecturaRPT(l, ctx, o.Solicitud, o.Autorizacion,
		accionLeerPublicacionRPT, tipoCatalogoRPT, l.descriptor.CatalogoID, "", consultaPublicacionRPT, material,
		func(datos []byte) (ports.ResultadoPublicacionCategoriaRPT, error) {
			return decodificarPublicacionHistoricaRPT(datos, consulta, l.descriptor)
		})
	resultado.Evidencia = evidencia
	return resultado, err
}

func (l *LectorCategoriasRPTPostgreSQL) ConsultarUsoCategoriaRPT(ctx context.Context, o ports.OrdenUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	var cero ports.ResultadoUsoCategoriaRPT
	consulta := o.Consulta
	if l == nil || !claveRPT.MatchString(consulta.Consumidor) ||
		len(consulta.UsoRef) < 3 || len(consulta.UsoRef) > 160 ||
		len(consulta.ReservaReciboRef) < 3 || len(consulta.ReservaReciboRef) > 160 {
		return cero, ports.ErrLecturaRPTInvalida
	}
	material, err := json.Marshal(materialUsoRPT{
		CatalogoID: l.descriptor.CatalogoID, ModuloID: l.descriptor.ModuloID,
		Consumidor: consulta.Consumidor, UsoRef: consulta.UsoRef, ReservaReciboRef: consulta.ReservaReciboRef,
	})
	if err != nil {
		return cero, ports.ErrLecturaRPTInvalida
	}
	resultado, evidencia, err := ejecutarLecturaRPT(l, ctx, o.Solicitud, o.Autorizacion,
		accionConsultarUsoRPT, tipoUsoRPT, consulta.UsoRef, consulta.Consumidor, consultaUsoRPT, material,
		func(datos []byte) (ports.ResultadoUsoCategoriaRPT, error) {
			return decodificarUsoCategoriaRPT(datos, consulta, l.descriptor)
		})
	resultado.Evidencia = evidencia
	return resultado, err
}

func (l *LectorCategoriasRPTPostgreSQL) validarAutorizacionRPT(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	accion, tipo, referencia, consumidor string,
) (string, error) {
	if l == nil || valorNuloPostgreSQL(l.pool) || autorizacion.ValidarEstructura() != nil {
		return "", ports.ErrLecturaRPTDenegada
	}
	d, err := solicitud.Datos()
	if err != nil || d.Recurso.Validar() != nil {
		return "", ports.ErrLecturaRPTDenegada
	}
	r := d.Recurso
	if d.Accion != accion || d.Finalidad != finalidadLecturaRPT ||
		r.Referencia != referencia || r.ModuloID != l.descriptor.ModuloID || r.Tipo != tipo ||
		len(r.Atributos) != 1 || !huellaRPT.MatchString(r.Atributos["material_sha256"]) ||
		r.Ambitos["catalogo_id"] != l.descriptor.CatalogoID || r.Ambitos["modulo_id"] != l.descriptor.ModuloID {
		return "", ports.ErrLecturaRPTDenegada
	}
	if tipo == tipoCatalogoRPT {
		if len(r.Ambitos) != 2 || consumidor != "" {
			return "", ports.ErrLecturaRPTDenegada
		}
	} else if len(r.Ambitos) != 3 || r.Ambitos["consumidor"] != consumidor || consumidor == "" {
		return "", ports.ErrLecturaRPTDenegada
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	z := autorizacion.ResumenCapacidad()
	if err != nil || z.Operacion() != accion || z.AudienciaConsumo() != audienciaLecturaRPT ||
		z.EfectoRef() != referencia || z.EfectoHuellaSHA256() != huella {
		return "", ports.ErrLecturaRPTDenegada
	}
	return r.Atributos["material_sha256"], nil
}

func ejecutarLecturaRPT[T any](l *LectorCategoriasRPTPostgreSQL, ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	accion, tipo, referencia, consumidor, consultaSQL string, material []byte,
	decodificar func([]byte) (T, error),
) (T, ports.EvidenciaLecturaRPT, error) {
	var cero T
	var evidenciaVacia ports.EvidenciaLecturaRPT
	defer borrarPiezasRPT(material)
	if ctx == nil || decodificar == nil {
		return cero, evidenciaVacia, ports.ErrLecturaRPTDenegada
	}
	huellaEsperada, err := l.validarAutorizacionRPT(solicitud, autorizacion, accion, tipo, referencia, consumidor)
	if err != nil {
		return cero, evidenciaVacia, err
	}
	if err := ctx.Err(); err != nil {
		return cero, evidenciaVacia, err
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, evidenciaVacia, errorLecturaRPT(ctx, err)
	}
	if valorNuloPostgreSQL(tx) {
		return cero, evidenciaVacia, ports.ErrLecturaRPTNoDisponible
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	if _, err := tx.Exec(ctx, configurarLecturaRPT); err != nil {
		return cero, evidenciaVacia, errorLecturaRPT(ctx, err)
	}
	// jsonb::text tiene la representación de PostgreSQL, distinta de
	// json.Marshal. Se coteja sin leer tablas ni consumir AD3.
	var huellaMaterial string
	if err := tx.QueryRow(ctx, consultaHuellaMaterialRPT, string(material)).Scan(&huellaMaterial); err != nil {
		return cero, evidenciaVacia, errorLecturaRPT(ctx, err)
	}
	if huellaMaterial != huellaEsperada {
		return cero, evidenciaVacia, ports.ErrLecturaRPTDenegada
	}
	piezas := [][]byte{
		autorizacion.CapacidadCanonica(), autorizacion.DecisionCanonica(), autorizacion.MotivoCanonico(),
		autorizacion.ContextoActorCanonico(), autorizacion.PayloadVECAD3(), autorizacion.SobreCOSESign1(),
		autorizacion.EvidenciaVerificacion(), autorizacion.RaizPublicaSPKI(),
	}
	defer borrarPiezasRPT(piezas...)
	var respuesta []byte
	err = tx.QueryRow(ctx, consultaSQL, string(material), piezas[0], piezas[1], piezas[2], piezas[3],
		strconv.FormatUint(autorizacion.PersonaVersion(), 10), strconv.FormatUint(autorizacion.PerfilVersion(), 10),
		piezas[4], piezas[5], piezas[6], piezas[7]).Scan(&respuesta)
	if err != nil {
		return cero, evidenciaVacia, errorLecturaRPT(ctx, err)
	}
	defer borrarPiezasRPT(respuesta)
	evidencia, encontrada, datos, err := decodificarReciboLecturaRPT(respuesta, autorizacion)
	if err != nil {
		return cero, evidenciaVacia, err
	}
	resultado := cero
	if encontrada {
		resultado, err = decodificar(datos)
		if err != nil {
			return cero, evidenciaVacia, err
		}
	}
	if err := ctx.Err(); err != nil {
		return cero, evidenciaVacia, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, evidenciaVacia, errorLecturaRPT(ctx, err)
	}
	return resultado, evidencia, nil
}

func borrarPiezasRPT(piezas ...[]byte) {
	for _, pieza := range piezas {
		clear(pieza)
	}
}

func errorLecturaRPT(ctx context.Context, causa error) error {
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
			return ports.ErrLecturaRPTDenegada
		case "22023":
			return ports.ErrLecturaRPTInvalida
		case "54000":
			return ports.ErrLecturaRPTPresupuesto
		}
	}
	return ports.ErrLecturaRPTNoDisponible
}

type publicacionRPTWire struct {
	CatalogoID        string    `json:"catalogo_id"`
	Version           int       `json:"version"`
	HuellaSHA256      string    `json:"huella_sha256"`
	DocumentoCanonico string    `json:"documento_canonico"`
	PublicadaEn       time.Time `json:"publicada_en"`
}

type reciboLecturaRPTWire struct {
	DecisionRef         string          `json:"decision_ref"`
	EfectoRef           string          `json:"efecto_ref"`
	HuellaEfectoSHA256  string          `json:"huella_efecto_sha256"`
	ConsumoHuellaSHA256 string          `json:"consumo_huella_sha256"`
	AuditoriaRef        string          `json:"auditoria_ref"`
	ConsumidaEn         time.Time       `json:"consumida_en"`
	ConsumoNuevo        bool            `json:"consumo_nuevo"`
	Encontrado          bool            `json:"encontrado"`
	Datos               json.RawMessage `json:"datos"`
}

func decodificarReciboLecturaRPT(bruto []byte, autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EvidenciaLecturaRPT, bool, []byte, error) {
	var cero ports.EvidenciaLecturaRPT
	var r reciboLecturaRPTWire
	if len(bruto) < 2 || len(bruto) > maximoRespuestaLecturaRPT ||
		numeroClavesRPT(bruto) != 9 || decodificarRPT(bruto, &r) != nil ||
		autorizacion.ValidarEstructura() != nil {
		return cero, false, nil, ports.ErrLecturaRPTNoConfiable
	}
	resumen := autorizacion.ResumenCapacidad()
	if r.DecisionRef != resumen.DecisionRef() || r.EfectoRef != resumen.EfectoRef() ||
		r.HuellaEfectoSHA256 != resumen.EfectoHuellaSHA256() ||
		!huellaRPT.MatchString(r.ConsumoHuellaSHA256) || r.AuditoriaRef == "" ||
		!r.ConsumoNuevo || !instanteRPTValido(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(resumen.EmitidaEn()) || !r.ConsumidaEn.Before(resumen.ExpiraEn()) {
		return cero, false, nil, ports.ErrLecturaRPTNoConfiable
	}
	if (r.Encontrado && (len(r.Datos) == 0 || bytes.Equal(bytes.TrimSpace(r.Datos), []byte("null")))) ||
		(!r.Encontrado && !bytes.Equal(bytes.TrimSpace(r.Datos), []byte("null"))) {
		return cero, false, nil, ports.ErrLecturaRPTNoConfiable
	}
	evidencia := ports.EvidenciaLecturaRPT{
		DecisionRef: r.DecisionRef, EfectoRef: r.EfectoRef, HuellaEfectoSHA256: r.HuellaEfectoSHA256,
		ConsumoHuellaSHA256: r.ConsumoHuellaSHA256, AuditoriaRef: r.AuditoriaRef,
		ConsumidaEn: r.ConsumidaEn.UTC(), ConsumoNuevo: true,
	}
	return evidencia, r.Encontrado, r.Datos, nil
}

func (p publicacionRPTWire) referencia() ports.ReferenciaPublicacionRPT {
	return ports.ReferenciaPublicacionRPT{CatalogoID: p.CatalogoID, Version: p.Version, HuellaSHA256: p.HuellaSHA256}
}

// comprobarPublicacionRPT verifica los bytes originales, el documento de
// dominio y cada entrada antes de entregar una publicación al consumidor.
// La huella nunca se reconstruye desde JSONB de una fila de control.
func comprobarPublicacionRPT(p publicacionRPTWire, descriptor ports.DescriptorCatalogoRPT) (ports.PublicacionRPT, map[string]domain.EntradaCatalogoConfigurable, error) {
	var cero ports.PublicacionRPT
	if p.CatalogoID != descriptor.CatalogoID || p.Version < 1 || !huellaRPT.MatchString(p.HuellaSHA256) ||
		len(p.DocumentoCanonico) < 2 || len(p.DocumentoCanonico) > maximoDocumentoCanonicoRPT {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	suma := sha256.Sum256([]byte(p.DocumentoCanonico))
	if hex.EncodeToString(suma[:]) != p.HuellaSHA256 {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	var catalogo domain.CatalogoConfigurable
	if decodificarRPT([]byte(p.DocumentoCanonico), &catalogo) != nil ||
		catalogo.ID != p.CatalogoID || catalogo.ModuloID != descriptor.ModuloID || catalogo.Version != p.Version ||
		catalogo.Estado != domain.EstadoCatalogoPublicado || catalogo.Validar() != nil {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	representacion, err := json.Marshal(canonico)
	if err != nil || !bytes.Equal(representacion, []byte(p.DocumentoCanonico)) {
		return cero, nil, ports.ErrLecturaRPTNoConfiable
	}
	entradas := make(map[string]domain.EntradaCatalogoConfigurable, len(catalogo.Entradas))
	for _, entrada := range catalogo.Entradas {
		entradas[entrada.Clave] = entrada
	}
	return ports.PublicacionRPT{Referencia: p.referencia(), DocumentoCanonico: p.DocumentoCanonico, PublicadaEn: p.PublicadaEn}, entradas, nil
}

type categoriaRPTWire struct {
	CategoriaID  string          `json:"categoria_id"`
	CatalogoID   string          `json:"catalogo_id"`
	Version      int             `json:"version"`
	HuellaSHA256 string          `json:"huella_sha256"`
	Revision     int64           `json:"revision"`
	Estado       string          `json:"estado"`
	Etiqueta     string          `json:"etiqueta"`
	Definicion   json.RawMessage `json:"definicion"`
}

func (c categoriaRPTWire) referencia() ports.ReferenciaPublicacionRPT {
	return ports.ReferenciaPublicacionRPT{CatalogoID: c.CatalogoID, Version: c.Version, HuellaSHA256: c.HuellaSHA256}
}

type listaRPTWire struct {
	AnclajePublicacion json.RawMessage   `json:"anclaje_publicacion"`
	Items              []json.RawMessage `json:"items"`
	Publicaciones      []json.RawMessage `json:"publicaciones"`
	HayMas             bool              `json:"hay_mas"`
	SiguienteCursor    *string           `json:"siguiente_cursor"`
}

func decodificarListaRPT(datos []byte, consulta ports.ConsultaCategoriasHabilitadasRPT, descriptor ports.DescriptorCatalogoRPT) (ports.ResultadoCategoriasHabilitadasRPT, error) {
	var resultado ports.ResultadoCategoriasHabilitadasRPT
	var wire listaRPTWire
	if numeroClavesRPT(datos) != 5 || decodificarRPT(datos, &wire) != nil ||
		len(wire.Items) > consulta.Limite || len(wire.Publicaciones) > len(wire.Items) ||
		numeroClavesRPT(wire.AnclajePublicacion) != 4 {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	var anclaje publicacionRPTWire
	if decodificarRPT(wire.AnclajePublicacion, &anclaje) != nil || anclaje.CatalogoID != consulta.CatalogoID {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	publicacionAncla, entradasAncla, err := comprobarPublicacionRPT(anclaje, descriptor)
	if err != nil {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	resultado.AnclajePublicacion = &publicacionAncla
	resultado.Publicaciones = append(resultado.Publicaciones, publicacionAncla)
	type publicacionCompleta struct {
		publicacion ports.PublicacionRPT
		entradas    map[string]domain.EntradaCatalogoConfigurable
	}
	publicaciones := make(map[ports.ReferenciaPublicacionRPT]publicacionCompleta, len(wire.Publicaciones)+1)
	publicaciones[anclaje.referencia()] = publicacionCompleta{publicacion: publicacionAncla, entradas: entradasAncla}
	for _, bruto := range wire.Publicaciones {
		var p publicacionRPTWire
		if numeroClavesRPT(bruto) != 4 || decodificarRPT(bruto, &p) != nil {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		publicacion, entradas, err := comprobarPublicacionRPT(p, descriptor)
		if err != nil || p.CatalogoID != consulta.CatalogoID || p.Version <= anclaje.Version {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		if _, duplicada := publicaciones[p.referencia()]; duplicada {
			return resultado, ports.ErrLecturaRPTNoConfiable
		}
		publicaciones[p.referencia()] = publicacionCompleta{publicacion: publicacion, entradas: entradas}
		resultado.Publicaciones = append(resultado.Publicaciones, publicacion)
	}
	anterior := consulta.CursorCategoriaID
	for _, bruto := range wire.Items {
		var c categoriaRPTWire
		if numeroClavesRPT(bruto) != 8 || decodificarRPT(bruto, &c) != nil ||
			c.CatalogoID != consulta.CatalogoID || c.CategoriaID <= anterior || c.Revision < 1 ||
			c.Estado != "habilitada" {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		p, existe := publicaciones[c.referencia()]
		entrada, publicada := p.entradas[c.CategoriaID]
		var recibida domain.EntradaCatalogoConfigurable
		if !existe || !publicada || decodificarRPT(c.Definicion, &recibida) != nil ||
			!reflect.DeepEqual(recibida, entrada) || c.Etiqueta != entrada.Etiqueta {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		resultado.Categorias = append(resultado.Categorias, ports.CategoriaHabilitadaRPT{
			CategoriaID: c.CategoriaID, Publicacion: c.referencia(), Revision: c.Revision,
			Estado: c.Estado, Etiqueta: c.Etiqueta, Definicion: entrada,
		})
		anterior = c.CategoriaID
	}
	if wire.HayMas {
		if len(resultado.Categorias) == 0 || wire.SiguienteCursor == nil || *wire.SiguienteCursor != anterior {
			return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
		}
	} else if wire.SiguienteCursor != nil {
		return ports.ResultadoCategoriasHabilitadasRPT{}, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado, resultado.HayMas, resultado.SiguienteCursor = true, wire.HayMas, wire.SiguienteCursor
	return resultado, nil
}

type publicacionHistoricaRPTWire struct {
	Publicacion   json.RawMessage `json:"publicacion"`
	Entrada       json.RawMessage `json:"entrada"`
	ControlActual json.RawMessage `json:"control_actual"`
}

type controlActualRPTWire struct {
	CatalogoID   string `json:"catalogo_id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Revision     int64  `json:"revision"`
	Estado       string `json:"estado"`
}

func decodificarPublicacionHistoricaRPT(datos []byte, consulta ports.ConsultaPublicacionCategoriaRPT, descriptor ports.DescriptorCatalogoRPT) (ports.ResultadoPublicacionCategoriaRPT, error) {
	var resultado ports.ResultadoPublicacionCategoriaRPT
	var wire publicacionHistoricaRPTWire
	if numeroClavesRPT(datos) != 3 || decodificarRPT(datos, &wire) != nil ||
		numeroClavesRPT(wire.Publicacion) != 5 {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	var p publicacionRPTWire
	if decodificarRPT(wire.Publicacion, &p) != nil || p.referencia() != consulta.Referencia ||
		!instanteRPTValido(p.PublicadaEn) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	publicacion, entradas, err := comprobarPublicacionRPT(p, descriptor)
	if err != nil {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	var entrada domain.EntradaCatalogoConfigurable
	publicada, existe := entradas[consulta.CategoriaID]
	if !existe || decodificarRPT(wire.Entrada, &entrada) != nil ||
		!reflect.DeepEqual(entrada, publicada) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado, resultado.Publicacion, resultado.Entrada = true, &publicacion, &publicada
	if !bytes.Equal(bytes.TrimSpace(wire.ControlActual), []byte("null")) {
		var c controlActualRPTWire
		if numeroClavesRPT(wire.ControlActual) != 5 || decodificarRPT(wire.ControlActual, &c) != nil ||
			c.CatalogoID != consulta.Referencia.CatalogoID || c.Version < 1 ||
			!huellaRPT.MatchString(c.HuellaSHA256) || c.Revision < 1 ||
			(c.Estado != "habilitada" && c.Estado != "deshabilitada" && c.Estado != "tombstone") {
			return ports.ResultadoPublicacionCategoriaRPT{}, ports.ErrLecturaRPTNoConfiable
		}
		resultado.ControlActual = &ports.ControlCategoriaRPT{
			Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: c.CatalogoID, Version: c.Version, HuellaSHA256: c.HuellaSHA256},
			Revision:    c.Revision, Estado: c.Estado,
		}
	}
	return resultado, nil
}

type usoCategoriaRPTWire struct {
	Consumidor        string     `json:"consumidor"`
	UsoRef            string     `json:"uso_ref"`
	CategoriaID       string     `json:"categoria_id"`
	CatalogoID        string     `json:"catalogo_id"`
	Version           int        `json:"version"`
	HuellaSHA256      string     `json:"huella_sha256"`
	Estado            string     `json:"estado"`
	Revision          int64      `json:"revision"`
	ReservaReciboRef  string     `json:"reserva_recibo_ref"`
	TerminalReciboRef *string    `json:"terminal_recibo_ref"`
	ReservadoEn       time.Time  `json:"reservado_en"`
	TerminalEn        *time.Time `json:"terminal_en"`
}

func decodificarUsoCategoriaRPT(datos []byte, consulta ports.ConsultaUsoCategoriaRPT, descriptor ports.DescriptorCatalogoRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	var resultado ports.ResultadoUsoCategoriaRPT
	var u usoCategoriaRPTWire
	if numeroClavesRPT(datos) != 12 || decodificarRPT(datos, &u) != nil ||
		u.Consumidor != consulta.Consumidor || u.UsoRef != consulta.UsoRef ||
		u.ReservaReciboRef != consulta.ReservaReciboRef || !claveRPT.MatchString(u.CategoriaID) ||
		u.CatalogoID != descriptor.CatalogoID || u.Version < 1 || !huellaRPT.MatchString(u.HuellaSHA256) ||
		!instanteRPTValido(u.ReservadoEn) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	terminal := u.Estado == "confirmado" || u.Estado == "cancelado"
	if !(u.Estado == "reservado" || terminal) ||
		(!terminal && (u.Revision != 1 || u.TerminalReciboRef != nil || u.TerminalEn != nil)) ||
		(terminal && (u.Revision != 2 || u.TerminalReciboRef == nil || *u.TerminalReciboRef == "" ||
			u.TerminalEn == nil || !instanteRPTValido(*u.TerminalEn) || u.TerminalEn.Before(u.ReservadoEn))) {
		return resultado, ports.ErrLecturaRPTNoConfiable
	}
	resultado.Encontrado = true
	resultado.Uso = &ports.UsoCategoriaRPT{
		Consumidor: u.Consumidor, UsoRef: u.UsoRef, CategoriaID: u.CategoriaID,
		Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: u.CatalogoID, Version: u.Version, HuellaSHA256: u.HuellaSHA256},
		Estado:      u.Estado, Revision: u.Revision, ReservaReciboRef: u.ReservaReciboRef,
		TerminalReciboRef: u.TerminalReciboRef, ReservadoEn: u.ReservadoEn.UTC(), TerminalEn: u.TerminalEn,
	}
	return resultado, nil
}

func instanteRPTValido(instante time.Time) bool {
	_, desfase := instante.Zone()
	return !instante.IsZero() && desfase == 0 && instante.Nanosecond()%1_000 == 0
}

func numeroClavesRPT(bruto []byte) int {
	var campos map[string]json.RawMessage
	if json.Unmarshal(bruto, &campos) != nil {
		return -1
	}
	return len(campos)
}

func decodificarRPT(bruto []byte, destino any) error {
	decoder := json.NewDecoder(bytes.NewReader(bruto))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ports.ErrLecturaRPTNoConfiable
	}
	return nil
}
