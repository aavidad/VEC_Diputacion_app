package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	administracion "vec-diputacion-granada/internal/modules/administracion"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	audienciaGobiernoModulos         = "vec_catalogos_configurables.gobierno_modulos.v1"
	finalidadGobiernoModulos         = "gobernar_modulos_administracion"
	parametrosGobiernoModulos        = `($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	consultaConfirmarGobiernoModulos = `SELECT catalogo_canonico,recibo_canonico,recibo_sha256,recuperada FROM vec_catalogos_configurables.confirmar_gobierno_modulos_v1` + parametrosGobiernoModulos
	consultaRecuperarGobiernoModulos = `SELECT catalogo_canonico,recibo_canonico,recibo_sha256,recuperada FROM vec_catalogos_configurables.recuperar_gobierno_modulos_v1` + parametrosGobiernoModulos
	configurarGobiernoModulos        = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
	maximoMaterialGobiernoModulos    = 2 << 20
)

var (
	errGobiernoModulos   = errors.New("vec: gobierno durable de modulos no disponible")
	shaGobiernoModulos   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	claveGobiernoModulos = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)
)

type iniciadorGobiernoModulos interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RepositorioGobiernoModulosPostgreSQL usa identidad registrada y configuración
// institucional de composición. No resuelve cuentas ni crea permisos o perfiles.
type RepositorioGobiernoModulosPostgreSQL struct {
	pool   iniciadorGobiernoModulos
	fuente ports.FuenteGobiernoModulosV3
	emisor ports.EmisorGobiernoModulosV3
}

var _ ports.RepositorioCatalogosOperativos = (*RepositorioGobiernoModulosPostgreSQL)(nil)

func NuevoRepositorioGobiernoModulosPostgreSQL(pool *pgxpool.Pool, fuente ports.FuenteGobiernoModulosV3, emisor ports.EmisorGobiernoModulosV3) (*RepositorioGobiernoModulosPostgreSQL, error) {
	return nuevoRepositorioGobiernoModulos(pool, fuente, emisor)
}
func nuevoRepositorioGobiernoModulos(pool iniciadorGobiernoModulos, fuente ports.FuenteGobiernoModulosV3, emisor ports.EmisorGobiernoModulosV3) (*RepositorioGobiernoModulosPostgreSQL, error) {
	if nuloGobiernoModulos(pool) || nuloGobiernoModulos(fuente) || nuloGobiernoModulos(emisor) {
		return nil, errGobiernoModulos
	}
	return &RepositorioGobiernoModulosPostgreSQL{pool: pool, fuente: fuente, emisor: emisor}, nil
}

func (r *RepositorioGobiernoModulosPostgreSQL) disponible(ctx context.Context) error {
	if ctx == nil || r == nil || nuloGobiernoModulos(r.pool) || nuloGobiernoModulos(r.fuente) || nuloGobiernoModulos(r.emisor) {
		return errGobiernoModulos
	}
	return ctx.Err()
}

func (r *RepositorioGobiernoModulosPostgreSQL) configuracion(ctx context.Context) (ports.ConfiguracionGobiernoModulosAprobada, error) {
	cfg, err := r.fuente.ConfiguracionAprobadaGobiernoModulos(ctx)
	cfg.Registrados = append([]string(nil), cfg.Registrados...)
	cfg.Gobernados = append([]string(nil), cfg.Gobernados...)
	if err != nil || cfg.Version < 1 || cfg.Version > 1_000_000 || cfg.CatalogoID != "administracion.modulos" || !shaGobiernoModulos.MatchString(cfg.HuellaSHA256) || !shaGobiernoModulos.MatchString(cfg.RegistroSHA256) || !referenciaGobiernoModulos(cfg.PerfilFijoRef) || !referenciaGobiernoModulos(cfg.RegistroVersionRef) || cfg.FinalidadRef != finalidadGobiernoModulos || cfg.MotivoCrear.Validar() != nil || cfg.MotivoPublicar.Validar() != nil || len(cfg.Registrados) == 0 || len(cfg.Registrados) > 128 || len(cfg.Gobernados) == 0 || len(cfg.Gobernados) > len(cfg.Registrados) {
		return ports.ConfiguracionGobiernoModulosAprobada{}, errGobiernoModulos
	}
	registrados := make(map[string]bool, len(cfg.Registrados))
	for _, id := range cfg.Registrados {
		if !claveGobiernoModulos.MatchString(id) || registrados[id] {
			return ports.ConfiguracionGobiernoModulosAprobada{}, errGobiernoModulos
		}
		registrados[id] = true
	}
	gobernados := make(map[string]bool, len(cfg.Gobernados))
	for _, id := range cfg.Gobernados {
		if !registrados[id] || gobernados[id] || id == administracion.ModuleID || id == "vec.module.usuarios" {
			return ports.ConfiguracionGobiernoModulosAprobada{}, errGobiernoModulos
		}
		gobernados[id] = true
	}
	return cfg, nil
}

type cabezaGobiernoModulosWire struct {
	Version      int                               `json:"version"`
	HuellaSHA256 string                            `json:"huella_sha256"`
	Estado       domain.EstadoCatalogoConfigurable `json:"estado"`
}
type borradorGobiernoModulosWire struct {
	Version      int    `json:"version"`
	Revision     int    `json:"revision"`
	HuellaSHA256 string `json:"huella_sha256"`
}
type materialGobiernoModulos struct {
	Clave                string                       `json:"clave_operacion"`
	Operacion            string                       `json:"operacion"`
	HuellaMaterial       string                       `json:"huella_material_sha256"`
	MaterialSemantico    []byte                       `json:"material_semantico_base64"`
	ConfiguracionVersion int64                        `json:"configuracion_version"`
	ConfiguracionHuella  string                       `json:"configuracion_huella_sha256"`
	RegistroSHA256       string                       `json:"registro_sha256"`
	RegistroVersionRef   string                       `json:"registro_version_ref"`
	Cabeza               cabezaGobiernoModulosWire    `json:"cabeza_publicada_esperada"`
	Borrador             *borradorGobiernoModulosWire `json:"borrador_esperado"`
	Catalogo             []byte                       `json:"catalogo_canonico_base64"`
	HuellaComun          string                       `json:"huella_comun"`
	Traza                domain.AuditEntry            `json:"traza"`
	Evento               domain.Event                 `json:"evento"`
}

func (r *RepositorioGobiernoModulosPostgreSQL) ConfirmarAltaBorradorCatalogoOperativo(ctx context.Context, c ports.ConfirmacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	return r.confirmar(ctx, "crear", c)
}
func (r *RepositorioGobiernoModulosPostgreSQL) ConfirmarPublicacionCatalogoOperativo(ctx context.Context, c ports.ConfirmacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	return r.confirmar(ctx, "publicar", c)
}

func (r *RepositorioGobiernoModulosPostgreSQL) confirmar(ctx context.Context, operacion string, c ports.ConfirmacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	c.MaterialCanonico = bytes.Clone(c.MaterialCanonico)
	defer clear(c.MaterialCanonico)
	c.Auditoria.ActorRoles = append([]string(nil), c.Auditoria.ActorRoles...)
	c.Auditoria.Metadata = maps.Clone(c.Auditoria.Metadata)
	c.Evento.Payload = maps.Clone(c.Evento.Payload)
	var cero ports.ResultadoOperacionCatalogoOperativo
	if err := r.disponible(ctx); err != nil {
		return cero, err
	}
	cfg, err := r.configuracion(ctx)
	if err != nil {
		return cero, err
	}
	canon, err := c.Catalogo.ClonarCanonico()
	if err != nil || canon.ID != cfg.CatalogoID || canon.ModuloID != administracion.ModuleID || canon.Revision != 1 || canon.Version > 1_000_000 || !instantesCatalogoGobiernoModulos(canon) || !claveOperacionGobiernoModulos(c.ClaveIdempotencia) || !shaGobiernoModulos.MatchString(c.HuellaMaterialSHA256) || c.CabezaEsperada.CatalogoID != cfg.CatalogoID || c.CabezaEsperada.Version < 1 || c.CabezaEsperada.Estado != domain.EstadoCatalogoPublicado || !shaGobiernoModulos.MatchString(c.CabezaEsperada.HuellaSHA256) || canon.Version != c.CabezaEsperada.Version+1 {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	if err := validarEntradasGobiernoModulos(canon, cfg); err != nil {
		return cero, err
	}
	accion, evento, estado, motivo := ports.AccionCrearCatalogoConfigurable, domain.AccionCatalogoBorradorCreado, domain.EstadoCatalogoBorrador, cfg.MotivoCrear
	if operacion == "publicar" {
		accion, evento, estado, motivo = ports.AccionPublicarCatalogoConfigurable, domain.AccionCatalogoPublicado, domain.EstadoCatalogoPublicado, cfg.MotivoPublicar
	}
	huella, err := canon.HuellaSHA256()
	if err != nil || canon.Estado != estado || c.Auditoria.AfterHash != huella || c.Auditoria.BeforeHash != c.HuellaAnteriorSHA256 || c.Auditoria.SubjectRef != canon.Referencia() || c.Auditoria.ModuleID != administracion.ModuleID || c.Auditoria.ObjectVersion != canon.Version || c.Auditoria.Action != evento || c.Auditoria.Result != "correcto" || c.Evento.Type != evento || c.Evento.SubjectRef != canon.Referencia() || c.Evento.ModuleID != administracion.ModuleID || c.Evento.ActorID != c.Auditoria.ActorID || c.Evento.Payload["huella_sha256"] != huella || !instanteGobiernoModulos(c.Auditoria.OccurredAt) || !c.Auditoria.OccurredAt.Equal(c.Evento.OccurredAt) {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	if (operacion == "crear" && (c.HuellaAnteriorSHA256 != "" || canon.CreadoPor != c.Auditoria.ActorID || !canon.CreadoEn.Equal(c.Auditoria.OccurredAt))) || (operacion == "publicar" && (!shaGobiernoModulos.MatchString(c.HuellaAnteriorSHA256) || canon.PublicadoPor != c.Auditoria.ActorID || !canon.PublicadoEn.Equal(c.Auditoria.OccurredAt))) {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	recursoV1 := domain.RecursoAutorizable{Referencia: canon.Referencia(), ModuloID: administracion.ModuleID, Tipo: "catalogo_configurable", Atributos: map[string]string{"estado": string(estado), "revision": strconv.Itoa(canon.Revision)}}
	datos, err := validarV1GobiernoModulos(c.Autorizacion, accion, recursoV1, cfg)
	if err != nil || datos.Decision.PrincipalID != c.Auditoria.ActorID || datos.Decision.PerfilActivoRef != c.Auditoria.ActorProfile || datos.Decision.DecisionRef != c.Auditoria.AuthorizationRef || datos.Decision.Finalidad != c.Auditoria.Purpose || datos.Decision.CorrelacionRef != c.Auditoria.CorrelationRef {
		return cero, domain.ErrAutorizacionDenegada
	}
	semantico, err := validarSemanticoGobiernoModulos(c.MaterialCanonico, c.HuellaMaterialSHA256, canon, c.CabezaEsperada, c.HuellaAnteriorSHA256, c.Auditoria.ActorID, cfg.FinalidadRef, accion)
	if err != nil {
		return cero, err
	}
	defer clear(semantico)
	vinculoV1, err := datos.Decision.VinculoAutenticacionActor.Datos()
	if err != nil || c.Auditoria.AuthMethod != vinculoV1.MetodoObservado || c.Auditoria.AuthAssurance != vinculoV1.GarantiaObservada {
		return cero, domain.ErrAutorizacionDenegada
	}
	bruto, err := json.Marshal(canon)
	if err != nil {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	defer clear(bruto)
	m := materialGobiernoModulos{Clave: c.ClaveIdempotencia, Operacion: operacion, HuellaMaterial: c.HuellaMaterialSHA256, MaterialSemantico: semantico, ConfiguracionVersion: cfg.Version, ConfiguracionHuella: cfg.HuellaSHA256, RegistroSHA256: cfg.RegistroSHA256, RegistroVersionRef: cfg.RegistroVersionRef, Cabeza: cabezaGobiernoModulosWire{Version: c.CabezaEsperada.Version, HuellaSHA256: c.CabezaEsperada.HuellaSHA256, Estado: c.CabezaEsperada.Estado}, Catalogo: bruto, HuellaComun: huella, Traza: c.Auditoria, Evento: c.Evento}
	if operacion == "publicar" {
		m.Borrador = &borradorGobiernoModulosWire{Version: canon.Version, Revision: canon.Revision, HuellaSHA256: c.HuellaAnteriorSHA256}
	}
	material, err := json.Marshal(m)
	if err != nil || len(material) > maximoMaterialGobiernoModulos {
		return cero, domain.ErrCatalogoConfigurableInvalido
	}
	defer clear(material)
	exportacion, actor, err := r.autorizar(ctx, c.Autorizacion, datos, cfg, motivo, recursoV1, material)
	if err != nil {
		return cero, err
	}
	return r.ejecutar(ctx, consultaConfirmarGobiernoModulos, material, exportacion, expectativaReciboGobiernoModulos{clave: c.ClaveIdempotencia, material: c.HuellaMaterialSHA256, accion: accion, id: canon.ID, version: canon.Version, actor: actor, catalogo: bruto})
}

func validarEntradasGobiernoModulos(c domain.CatalogoConfigurable, cfg ports.ConfiguracionGobiernoModulosAprobada) error {
	ids := make(map[string]bool, len(cfg.Gobernados))
	for _, id := range cfg.Gobernados {
		ids[id] = true
	}
	if len(c.Entradas) != len(ids) {
		return domain.ErrCatalogoConfigurableInvalido
	}
	for _, e := range c.Entradas {
		if !ids[e.Clave] || len(e.Atributos) != 1 || (e.Atributos["habilitado"] != "true" && e.Atributos["habilitado"] != "false") {
			return domain.ErrCatalogoConfigurableInvalido
		}
		delete(ids, e.Clave)
	}
	return nil
}

func validarV1GobiernoModulos(v ports.EvidenciaUsoDecisionAutorizacion, accion string, recurso domain.RecursoAutorizable, cfg ports.ConfiguracionGobiernoModulosAprobada) (ports.DatosEvidenciaUsoDecisionAutorizacion, error) {
	datos, err := v.Datos()
	huella, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || errHuella != nil || v.ValidarEn(time.Now().UTC()) != nil || !datos.Decision.Concedida || datos.Decision.Accion != accion || datos.Decision.RecursoRef != recurso.Referencia || datos.Decision.ModuloID != administracion.ModuleID || datos.Decision.TipoRecurso != "catalogo_configurable" || datos.Decision.ContextoRecursoHuellaSHA256 != huella || datos.Decision.PerfilActivoRef != cfg.PerfilFijoRef || datos.Decision.Finalidad != cfg.FinalidadRef || len(datos.Decision.CamposPermitidos) != 0 || len(datos.Decision.Obligaciones) != 0 {
		return ports.DatosEvidenciaUsoDecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	return datos, nil
}

func (r *RepositorioGobiernoModulosPostgreSQL) autorizar(ctx context.Context, v ports.EvidenciaUsoDecisionAutorizacion, datos ports.DatosEvidenciaUsoDecisionAutorizacion, cfg ports.ConfiguracionGobiernoModulosAprobada, motivo domain.ReferenciaEntradaCatalogo, recurso domain.RecursoAutorizable, material []byte) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, error) {
	var cero ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	id, err := r.fuente.IdentidadRegistradaGobiernoModulos(ctx, v)
	if err != nil || id.Resultado.Validar() != nil || id.Vinculo.ValidarPara(id.Resultado) != nil || id.Resultado.Contexto.Principal.ID != datos.Decision.PrincipalID || id.Resultado.Contexto.PerfilActivoRef != cfg.PerfilFijoRef || datos.Decision.VinculoAutenticacionActor.ValidarPara(id.Resultado.Contexto) != nil {
		return cero, "", domain.ErrAutorizacionDenegada
	}
	v2, err := id.Vinculo.Datos()
	v1, errV1 := datos.Decision.VinculoAutenticacionActor.Datos()
	if err != nil || errV1 != nil || v2.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 || !v2.CuentaPrivilegiada || v2.GarantiaObservada != domain.AuthAssuranceHigh || (v2.MetodoObservado != domain.AuthMethodCertificate && v2.MetodoObservado != domain.AuthMethodDNIe) || !mismaAutenticacionGobiernoModulos(v1, v2) {
		return cero, "", domain.ErrAutorizacionDenegada
	}
	suma := sha256.Sum256(material)
	recurso.Atributos["material_sha256"] = hex.EncodeToString(suma[:])
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return cero, "", errGobiernoModulos
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: id.Vinculo, ReferenciaMotivo: motivo, Accion: datos.Decision.Accion, Recurso: recurso, Finalidad: cfg.FinalidadRef, Correlacion: correlacion})
	if err != nil {
		return cero, "", domain.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := r.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, id.Resultado)
	if err != nil || nuloGobiernoModulos(exportador) || decision.ValidarPara(solicitud) != nil {
		return cero, "", domain.ErrAutorizacionDenegada
	}
	exportacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !ports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, id.Resultado, motivo, exportacion, audienciaGobiernoModulos) || !sinRestriccionesGobiernoModulos(exportacion.DecisionCanonica()) {
		return cero, "", domain.ErrAutorizacionDenegada
	}
	return exportacion, v2.PrincipalID, nil
}

func mismaAutenticacionGobiernoModulos(a domain.DatosVinculoAutenticacionActorV1, b domain.DatosVinculoAutenticacionActorV2) bool {
	return a.AutenticacionRef == b.AutenticacionRef && a.AutenticacionHuellaSHA256 == b.AutenticacionHuellaSHA256 && a.AsercionRef == b.AsercionRef && a.SesionRef == b.SesionRef && a.ControlSesionRef == b.ControlSesionRef && a.ControlSesionRevision == b.ControlSesionRevision && a.ControlSesionHuellaSHA256 == b.ControlSesionHuellaSHA256 && a.CuentaRef == b.CuentaRef && a.CuentaOrdinariaRef == b.CuentaOrdinariaRef && a.PrincipalID == b.PrincipalID && a.PerfilActivoRef == b.PerfilActivoRef && a.CuentaPrivilegiada == b.CuentaPrivilegiada && a.Superficie == b.Superficie && a.MetodoObservado == b.MetodoObservado && a.GarantiaObservada == b.GarantiaObservada && a.PoliticaGarantiaRef == b.PoliticaGarantiaRef && a.PoliticaGarantiaHuellaSHA256 == b.PoliticaGarantiaHuellaSHA256 && a.AutenticacionVerificadaEn.Equal(b.AutenticacionVerificadaEn) && a.SesionEmitidaEn.Equal(b.SesionEmitidaEn) && a.SesionValidaHasta.Equal(b.SesionValidaHasta) && a.SesionRevalidadaEn.Equal(b.SesionRevalidadaEn)
}

func nuloGobiernoModulos(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
func referenciaGobiernoModulos(s string) bool {
	if len(s) < 1 || len(s) > 512 || s != strings.TrimSpace(s) {
		return false
	}
	for _, r := range s {
		if r <= 32 || r == 127 {
			return false
		}
	}
	return true
}
func instanteGobiernoModulos(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
func decodificarGobiernoModulos(bruto []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return ports.ErrReciboCatalogoOperativoInvalido
	}
	return nil
}
func errorGobiernoModulos(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return domain.ErrAutorizacionDenegada
		case "23505", "40001", "40P01":
			return ports.ErrOperacionCatalogoOperativoEnConflicto
		case "22023", "23514":
			return domain.ErrCatalogoConfigurableInvalido
		case "54000":
			return ports.ErrLimitesConsultaCatalogosInvalidos
		case "P0002":
			return ports.ErrOperacionCatalogoOperativoNoEncontrada
		}
	}
	return errGobiernoModulos
}

// El consumidor de módulos implementa el efecto completo, sin proyecciones de
// campos ni obligaciones adicionales. Una decisión V3 restrictiva no se omite.
func sinRestriccionesGobiernoModulos(bruto []byte) bool {
	defer clear(bruto)
	var campos map[string]json.RawMessage
	return json.Unmarshal(bruto, &campos) == nil && bytes.Equal(campos["campos_permitidos"], []byte("[]")) && bytes.Equal(campos["obligaciones"], []byte("[]"))
}

func claveOperacionGobiernoModulos(s string) bool {
	return len(s) >= 3 && len(s) <= 160 && referenciaGobiernoModulos(s)
}
func instantesCatalogoGobiernoModulos(c domain.CatalogoConfigurable) bool {
	for _, t := range []time.Time{c.CreadoEn, c.UltimaModificacionEn, c.PublicadoEn, c.RetiradoEn} {
		if !t.IsZero() && !instanteGobiernoModulos(t) {
			return false
		}
	}
	for _, e := range c.Entradas {
		if !instanteGobiernoModulos(e.VigenteDesde) || !e.VigenteHasta.IsZero() && !instanteGobiernoModulos(e.VigenteHasta) {
			return false
		}
	}
	return true
}
