package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"strconv"
	"time"
	administracion "vec-diputacion-granada/internal/modules/administracion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type semanticoGobiernoModulos struct {
	Esquema              string
	Accion               string
	CatalogoID           string
	Version              int
	ActorRef             string
	Cabeza               ports.CabezaCatalogoOperativo
	Contenido            json.RawMessage
	HuellaBorradorSHA256 string
	Finalidad            string
	Motivo               string
	AprobacionRef        string
}

func validarSemanticoGobiernoModulos(bruto []byte, huella string, c domain.CatalogoConfigurable, cabeza ports.CabezaCatalogoOperativo, borrador, actor, finalidad, accion string) ([]byte, error) {
	sem, err := leerSemanticoGobiernoModulos(bruto, huella)
	if err != nil || sem.Accion != accion || sem.CatalogoID != c.ID || sem.Version != c.Version || sem.ActorRef != actor || sem.Cabeza != cabeza || sem.Finalidad != finalidad {
		return nil, domain.ErrCatalogoConfigurableInvalido
	}
	if accion == ports.AccionCrearCatalogoConfigurable {
		if len(sem.Contenido) < 2 || bytes.Equal(bytes.TrimSpace(sem.Contenido), []byte("null")) || !json.Valid(sem.Contenido) || sem.HuellaBorradorSHA256 != "" || sem.Motivo != c.MotivoCreacion || sem.AprobacionRef != "" {
			return nil, domain.ErrCatalogoConfigurableInvalido
		}
	} else if !bytes.Equal(bytes.TrimSpace(sem.Contenido), []byte("null")) || sem.HuellaBorradorSHA256 != borrador || sem.Motivo != c.MotivoPublicacion || sem.AprobacionRef != c.AprobacionRef {
		return nil, domain.ErrCatalogoConfigurableInvalido
	}

	return bytes.Clone(bruto), nil
}
func leerSemanticoGobiernoModulos(bruto []byte, huella string) (semanticoGobiernoModulos, error) {
	var s semanticoGobiernoModulos
	if len(bruto) < 2 || len(bruto) > 64<<10 || !shaGobiernoModulos.MatchString(huella) {
		return s, domain.ErrCatalogoConfigurableInvalido
	}
	suma := sha256.Sum256(bruto)
	if hex.EncodeToString(suma[:]) != huella || decodificarGobiernoModulos(bruto, &s) != nil || s.Esquema != "vec.catalogos.operacion.v1" {
		return semanticoGobiernoModulos{}, domain.ErrCatalogoConfigurableInvalido
	}
	canon, err := json.Marshal(s)
	if err != nil || !bytes.Equal(canon, bruto) {
		return semanticoGobiernoModulos{}, domain.ErrCatalogoConfigurableInvalido
	}
	return s, nil
}

type recuperarGobiernoModulosWire struct {
	Clave                string                            `json:"clave_operacion"`
	HuellaMaterial       string                            `json:"huella_material_sha256"`
	MaterialSemantico    []byte                            `json:"material_semantico_base64"`
	Operacion            string                            `json:"operacion"`
	CatalogoID           string                            `json:"catalogo_id"`
	Version              int                               `json:"version"`
	Revision             int                               `json:"revision"`
	Estado               domain.EstadoCatalogoConfigurable `json:"estado"`
	ConfiguracionVersion int64                             `json:"configuracion_version"`
	ConfiguracionHuella  string                            `json:"configuracion_huella_sha256"`
	RegistroSHA256       string                            `json:"registro_sha256"`
	RegistroVersionRef   string                            `json:"registro_version_ref"`
}

func (r *RepositorioGobiernoModulosPostgreSQL) RecuperarOperacionCatalogoOperativo(ctx context.Context, c ports.RecuperacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	c.MaterialCanonico = bytes.Clone(c.MaterialCanonico)
	defer clear(c.MaterialCanonico)
	var cero ports.ResultadoOperacionCatalogoOperativo
	if err := r.disponible(ctx); err != nil {
		return cero, err
	}
	cfg, err := r.configuracion(ctx)
	if err != nil {
		return cero, err
	}
	estado, operacion, motivo := domain.EstadoCatalogoBorrador, "crear", cfg.MotivoCrear
	if c.Accion == ports.AccionPublicarCatalogoConfigurable {
		estado, operacion, motivo = domain.EstadoCatalogoPublicado, "publicar", cfg.MotivoPublicar
	} else if c.Accion != ports.AccionCrearCatalogoConfigurable {
		return cero, domain.ErrAutorizacionDenegada
	}
	sem, err := leerSemanticoGobiernoModulos(c.MaterialCanonico, c.HuellaMaterialSHA256)
	if err != nil || !claveOperacionGobiernoModulos(c.ClaveIdempotencia) || c.CatalogoID != cfg.CatalogoID || c.Version < 2 || c.Version > 1_000_000 || sem.Accion != c.Accion || sem.CatalogoID != c.CatalogoID || sem.Version != c.Version || sem.Finalidad != cfg.FinalidadRef {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	recurso := domain.RecursoAutorizable{Referencia: c.CatalogoID + ":" + strconv.Itoa(c.Version), ModuloID: administracion.ModuleID, Tipo: "catalogo_configurable", Atributos: map[string]string{"estado": string(estado), "revision": "1"}}
	datos, err := validarV1GobiernoModulos(c.Autorizacion, c.Accion, recurso, cfg)
	if err != nil || datos.Decision.PrincipalID != sem.ActorRef {
		return cero, domain.ErrAutorizacionDenegada
	}
	semBytes := bytes.Clone(c.MaterialCanonico)
	defer clear(semBytes)
	bruto, err := materialRecuperacionGobiernoModulos(c, cfg, estado, operacion, semBytes)
	if err != nil {
		return cero, errGobiernoModulos
	}
	defer clear(bruto)
	exportacion, actor, err := r.autorizar(ctx, c.Autorizacion, datos, cfg, motivo, recurso, bruto)
	if err != nil {
		return cero, err
	}
	return r.ejecutar(ctx, consultaRecuperarGobiernoModulos, bruto, exportacion, expectativaReciboGobiernoModulos{clave: c.ClaveIdempotencia, material: c.HuellaMaterialSHA256, accion: c.Accion, id: c.CatalogoID, version: c.Version, actor: actor, recuperacion: true})
}

func materialRecuperacionGobiernoModulos(c ports.RecuperacionCatalogoOperativo, cfg ports.ConfiguracionGobiernoModulosAprobada, estado domain.EstadoCatalogoConfigurable, operacion string, semantico []byte) ([]byte, error) {
	return json.Marshal(recuperarGobiernoModulosWire{Clave: c.ClaveIdempotencia, HuellaMaterial: c.HuellaMaterialSHA256, MaterialSemantico: semantico, Operacion: operacion, CatalogoID: c.CatalogoID, Version: c.Version, Revision: 1, Estado: estado, ConfiguracionVersion: cfg.Version, ConfiguracionHuella: cfg.HuellaSHA256, RegistroSHA256: cfg.RegistroSHA256, RegistroVersionRef: cfg.RegistroVersionRef})
}

type expectativaReciboGobiernoModulos struct {
	clave, material, accion, id, actor string
	version                            int
	catalogo                           []byte
	recuperacion                       bool
}
type reciboGobiernoModulosWire struct {
	Referencia string                            `json:"referencia"`
	Clave      string                            `json:"clave_idempotencia"`
	Material   string                            `json:"huella_material_sha256"`
	Accion     string                            `json:"accion"`
	CatalogoID string                            `json:"catalogo_id"`
	Version    int                               `json:"version"`
	Huella     string                            `json:"huella_sha256"`
	Estado     domain.EstadoCatalogoConfigurable `json:"estado"`
	Actor      string                            `json:"actor_ref"`
	Auditoria  string                            `json:"auditoria_ref"`
	Outbox     string                            `json:"outbox_ref"`
	Fecha      time.Time                         `json:"confirmado_en"`
}

func comprobarReciboGobiernoModulos(canon, recibo []byte, huella string, recuperada bool, x expectativaReciboGobiernoModulos) (ports.ResultadoOperacionCatalogoOperativo, error) {
	var cero ports.ResultadoOperacionCatalogoOperativo
	if len(recibo) < 2 || len(recibo) > 64<<10 || !shaGobiernoModulos.MatchString(huella) || x.recuperacion && !recuperada {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	suma := sha256.Sum256(recibo)
	var w reciboGobiernoModulosWire
	if hex.EncodeToString(suma[:]) != huella || !camposReciboGobiernoModulos(recibo) || decodificarGobiernoModulos(recibo, &w) != nil || w.Clave != x.clave || w.Material != x.material || w.Accion != x.accion || w.CatalogoID != x.id || w.Version != x.version || w.Actor != x.actor || !referenciaGobiernoModulos(w.Referencia) || !referenciaGobiernoModulos(w.Auditoria) || !referenciaGobiernoModulos(w.Outbox) || !instanteGobiernoModulos(w.Fecha) {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	c, err := comprobarCatalogoGobiernoModulos(canon, w.Huella)
	if err != nil || c.ID != x.id || c.Version != x.version || c.Estado != w.Estado || c.ModuloID != administracion.ModuleID {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	fecha, actor := c.CreadoEn, c.CreadoPor
	if x.accion == ports.AccionPublicarCatalogoConfigurable {
		fecha, actor = c.PublicadoEn, c.PublicadoPor
		if c.Estado != domain.EstadoCatalogoPublicado {
			return cero, ports.ErrReciboCatalogoOperativoInvalido
		}
	} else if x.accion != ports.AccionCrearCatalogoConfigurable || c.Estado != domain.EstadoCatalogoBorrador {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	if actor != x.actor || !fecha.Equal(w.Fecha) || (!recuperada && len(x.catalogo) > 0 && !bytes.Equal(canon, x.catalogo)) {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	return ports.ResultadoOperacionCatalogoOperativo{Catalogo: c, Recibo: ports.ReciboCatalogoOperativo{Referencia: w.Referencia, ClaveIdempotencia: w.Clave, HuellaMaterialSHA256: w.Material, Accion: w.Accion, CatalogoID: w.CatalogoID, Version: w.Version, HuellaSHA256: w.Huella, Estado: w.Estado, ActorRef: w.Actor, AuditoriaRef: w.Auditoria, OutboxRef: w.Outbox, ConfirmadoEn: w.Fecha.UTC()}, Recuperada: recuperada}, nil
}
func comprobarCatalogoGobiernoModulos(bruto []byte, huella string) (domain.CatalogoConfigurable, error) {
	var cero domain.CatalogoConfigurable
	if len(bruto) < 2 || len(bruto) > ports.MaximoBytesConsultaCatalogosAcotada || !shaGobiernoModulos.MatchString(huella) {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	suma := sha256.Sum256(bruto)
	var c domain.CatalogoConfigurable
	if hex.EncodeToString(suma[:]) != huella || decodificarGobiernoModulos(bruto, &c) != nil {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	c, err := c.ClonarCanonico()
	if err != nil || !instantesCatalogoGobiernoModulos(c) {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	canon, err := json.Marshal(c)
	if err != nil || !bytes.Equal(canon, bruto) {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	return c, nil
}

func (r *RepositorioGobiernoModulosPostgreSQL) ejecutar(ctx context.Context, consulta string, material []byte, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, x expectativaReciboGobiernoModulos) (ports.ResultadoOperacionCatalogoOperativo, error) {
	var cero ports.ResultadoOperacionCatalogoOperativo
	if a.ValidarEstructura() != nil {
		return cero, domain.ErrAutorizacionDenegada
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	if nuloGobiernoModulos(tx) {
		return cero, errGobiernoModulos
	}
	defer rollbackGobiernoModulos(tx)
	if _, err := tx.Exec(ctx, configurarGobiernoModulos); err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	piezas := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, p := range piezas {
			clear(p)
		}
	}()
	var canon, recibo []byte
	var huella string
	var recuperada bool
	err = tx.QueryRow(ctx, consulta, material, piezas[0], piezas[1], piezas[2], piezas[3], strconv.FormatUint(a.PersonaVersion(), 10), strconv.FormatUint(a.PerfilVersion(), 10), piezas[4], piezas[5], piezas[6], piezas[7]).Scan(&canon, &recibo, &huella, &recuperada)
	defer clear(canon)
	defer clear(recibo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cero, ports.ErrOperacionCatalogoOperativoNoEncontrada
		}
		return cero, errorGobiernoModulos(ctx, err)
	}
	resultado, err := comprobarReciboGobiernoModulos(canon, recibo, huella, recuperada, x)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	return resultado, nil
}
func rollbackGobiernoModulos(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func camposReciboGobiernoModulos(bruto []byte) bool {
	d := json.NewDecoder(bytes.NewReader(bruto))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return false
	}
	vistos := map[string]bool{}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return false
		}
		k, ok := t.(string)
		if !ok || vistos[k] {
			return false
		}
		vistos[k] = true
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return false
		}
	}
	t, err = d.Token()
	if err != nil || t != json.Delim('}') || len(vistos) != 12 {
		return false
	}
	return true
}
