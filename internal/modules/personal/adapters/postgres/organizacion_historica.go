package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const consultaOrganizacionHistorica = `SELECT vec_personal.consultar_organizacion_historica_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const ajustesOrganizacionHistorica = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
const maxRespuestaOrganizacionHistorica = 512 << 10

type iniciadorOrganizacionHistorica interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioOrganizacionHistoricaPostgreSQL struct {
	pool iniciadorOrganizacionHistorica
}

var _ ports.RepositorioOrganizacionHistorica = (*RepositorioOrganizacionHistoricaPostgreSQL)(nil)

func NuevoRepositorioOrganizacionHistoricaPostgreSQL(pool *pgxpool.Pool) (*RepositorioOrganizacionHistoricaPostgreSQL, error) {
	return nuevoRepositorioOrganizacionHistoricaPostgreSQL(pool)
}

func nuevoRepositorioOrganizacionHistoricaPostgreSQL(pool iniciadorOrganizacionHistorica) (*RepositorioOrganizacionHistoricaPostgreSQL, error) {
	if nuloOrganizacionHistorica(pool) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return &RepositorioOrganizacionHistoricaPostgreSQL{pool: pool}, nil
}

// La función nominal consume AD3, registra auditoría y lee la foto en la misma
// transacción. Este adaptador nunca consulta las tablas ni establece identidad
// mediante cabeceras, GUC o datos del navegador.
func (r *RepositorioOrganizacionHistoricaPostgreSQL) ConsultarOrganizacionHistorica(ctx context.Context, o ports.OrdenConsultaOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio ports.ResultadoConsultaOrganizacionHistorica
	if r == nil || ctx == nil || nuloOrganizacionHistorica(r.pool) {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	solicitud := o.Material.Solicitud()
	reconstruido, err := domain.NuevoMaterialConsultaOrganizacionHistorica(solicitud)
	if err != nil || !bytes.Equal(reconstruido.Canonico(), o.Material.Canonico()) || !autorizacionOrganizacionHistoricaLigada(o.Material, o.Autorizacion) {
		return vacio, domain.ErrConsultaOrganizacionHistoricaInvalida
	}
	a := o.Autorizacion
	parametros := []any{string(o.Material.Canonico()), a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), int64(a.PersonaVersion()), int64(a.PerfilVersion()), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, p := range parametros {
			if b, ok := p.([]byte); ok {
				for i := range b {
					b[i] = 0
				}
			}
		}
	}()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorOrganizacionHistorica(ctx, err)
	}
	if tx == nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesOrganizacionHistorica); err != nil {
		return vacio, errorOrganizacionHistorica(ctx, err)
	}
	var bruto []byte
	if err = tx.QueryRow(ctx, consultaOrganizacionHistorica, parametros...).Scan(&bruto); err != nil {
		return vacio, errorOrganizacionHistorica(ctx, err)
	}
	resultado, err := decodificarOrganizacionHistorica(bruto, o)
	if err != nil {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorOrganizacionHistorica(ctx, err)
	}
	confirmada = true
	return resultado, nil
}

func autorizacionOrganizacionHistoricaLigada(m domain.MaterialConsultaOrganizacionHistorica, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	if a.ValidarEstructura() != nil {
		return false
	}
	s, x := m.Solicitud(), a.ResumenCapacidad()
	h, err := m.HuellaSHA256()
	return err == nil && a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion && a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionConsultaOrganizacionHistorica && x.AudienciaConsumo() == domain.AudienciaConsultaOrganizacionHistorica &&
		x.EfectoRef() == m.Recurso().Referencia && x.EfectoHuellaSHA256() == h
}

var referenciaOH = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
var huellaOH = regexp.MustCompile(`^[a-f0-9]{64}$`)

func decodificarOrganizacionHistorica(bruto []byte, o ports.OrdenConsultaOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio ports.ResultadoConsultaOrganizacionHistorica
	if len(bruto) == 0 || len(bruto) > maxRespuestaOrganizacionHistorica || bytes.Equal(bytes.TrimSpace(bruto), []byte("null")) {
		return vacio, errors.New("respuesta inválida")
	}
	if err := verificarJSONOrganizacionHistorica(bruto); err != nil {
		return vacio, err
	}
	if err := verificarFormaOrganizacionHistorica(bruto); err != nil {
		return vacio, err
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	var r ports.ResultadoConsultaOrganizacionHistorica
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return vacio, errors.New("respuesta inválida")
	}
	s, p, e := o.Material.Solicitud().Selector, r.Pagina, r.Evidencia
	x := o.Autorizacion.ResumenCapacidad()
	if !selectorOHIgual(p.Selector, s) || !referenciaOpcionalOH(p.VersionRPTRef) || !referenciaOpcionalOH(p.VersionPlantillaRef) ||
		(s.VersionRPTRef != "" && p.VersionRPTRef != s.VersionRPTRef) || (s.VersionPlantillaRef != "" && p.VersionPlantillaRef != s.VersionPlantillaRef) ||
		!cursorOH(p.CursorSiguiente) || (p.CursorSiguiente != "" && p.CursorSiguiente == s.Cursor) ||
		!coberturaOH(p.Cobertura.Unidades, len(p.Unidades)) || !coberturaOH(p.Cobertura.PuestosTipo, len(p.PuestosTipo)) ||
		!coberturaOH(p.Cobertura.Dotaciones, len(p.Dotaciones)) || !coberturaOH(p.Cobertura.Plazas, len(p.Plazas)) ||
		!coberturaOH(p.Cobertura.PuestosIndividuales, len(p.PuestosIndividuales)) || !coberturaOH(p.Cobertura.Vinculos, len(p.Vinculos)) ||
		len(p.Unidades)+len(p.PuestosTipo)+len(p.Dotaciones)+len(p.Plazas)+len(p.PuestosIndividuales)+len(p.Vinculos) > s.Limite ||
		!referenciaOH.MatchString(e.ReciboRef) || !referenciaOH.MatchString(e.AuditoriaRef) || !huellaOH.MatchString(e.ConsumoHuellaSHA256) ||
		e.DecisionRef != x.DecisionRef() || e.EfectoRef != x.EfectoRef() || !instanteOH(e.ConsultadaEn) ||
		e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) {
		return vacio, errors.New("respuesta inválida")
	}
	ids := map[string]bool{}
	traza := func(t domain.TrazaOrganizacionHistorica) bool {
		if t.ValidarEn(s) != nil || ids[t.ID] {
			return false
		}
		ids[t.ID] = true
		return true
	}
	for _, v := range p.Unidades {
		if !traza(v.Traza) || v.CatalogoID != ports.IDCatalogoOrganizacion || v.CatalogoVersion < 1 || v.CatalogoRevision < 1 || !referenciaOH.MatchString(v.ClaveCatalogo) || !referenciaOpcionalOH(v.PadreID) || !unoDeOH(v.Tipo, "delegacion", "centro", "puesto_responsabilidad") || !textoOH(v.Etiqueta) {
			return vacio, errors.New("unidad inválida")
		}
	}
	for _, v := range p.PuestosTipo {
		if !traza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !textoOH(v.CodigoFuente) || !referenciaOH.MatchString(v.UnidadID) || !textoOH(v.Denominacion) || !referenciaOH.MatchString(v.ClasificacionRef) {
			return vacio, errors.New("puesto tipo inválido")
		}
	}
	for _, v := range p.Dotaciones {
		if !traza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !referenciaOH.MatchString(v.PuestoTipoID) || v.Cantidad < 1 || v.Cantidad > 100000 {
			return vacio, errors.New("dotación inválida")
		}
	}
	for _, v := range p.Plazas {
		if !traza(v.Traza) || v.VersionPlantillaRef != p.VersionPlantillaRef || !textoOH(v.CodigoFuente) || !referenciaOH.MatchString(v.ClasificacionRef) || !referenciaOH.MatchString(v.UnidadID) || !unoDeOH(v.EstadoEstructural, "vigente", "amortizada") {
			return vacio, errors.New("plaza inválida")
		}
	}
	for _, v := range p.PuestosIndividuales {
		if !traza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !textoOH(v.CodigoFuente) || !referenciaOH.MatchString(v.PuestoTipoID) || !referenciaOH.MatchString(v.UnidadID) || !unoDeOH(v.EstadoEstructural, "vigente", "suprimido") {
			return vacio, errors.New("puesto inválido")
		}
	}
	for _, v := range p.Vinculos {
		if !traza(v.Traza) || !referenciaOH.MatchString(v.PlazaID) || !referenciaOH.MatchString(v.PuestoID) {
			return vacio, errors.New("vínculo inválido")
		}
	}
	return r, nil
}

func verificarJSONOrganizacionHistorica(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		if profundidad > 12 {
			return errors.New("profundidad JSON")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		sep, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch sep {
		case '{':
			visto := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				clave, ok := k.(string)
				if !ok || visto[clave] {
					return errors.New("clave JSON duplicada")
				}
				visto[clave] = true
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("JSON inválido")
		}
		_, err = d.Token()
		return err
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("JSON adicional")
	}
	return nil
}

// Un campo ausente o un array nulo no puede convertirse en una cobertura
// aparentemente válida por los valores cero de encoding/json.
func verificarFormaOrganizacionHistorica(b []byte) error {
	var raiz map[string]json.RawMessage
	if json.Unmarshal(b, &raiz) != nil || !clavesOH(raiz, []string{"pagina", "evidencia"}, nil) {
		return errors.New("forma JSON")
	}
	var pagina, evidencia map[string]json.RawMessage
	if json.Unmarshal(raiz["pagina"], &pagina) != nil || json.Unmarshal(raiz["evidencia"], &evidencia) != nil {
		return errors.New("forma JSON")
	}
	colecciones := []string{"unidades", "puestos_tipo", "dotaciones", "plazas", "puestos_individuales", "vinculos"}
	if !clavesOH(pagina, []string{"selector", "version_rpt_ref", "version_plantilla_ref", "cobertura", "unidades", "puestos_tipo", "dotaciones", "plazas", "puestos_individuales", "vinculos"}, []string{"cursor_siguiente"}) ||
		!clavesOH(evidencia, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) {
		return errors.New("forma JSON")
	}
	for _, nombre := range colecciones {
		if len(pagina[nombre]) == 0 || pagina[nombre][0] != '[' {
			return errors.New("colección JSON")
		}
	}
	var selector, cobertura map[string]json.RawMessage
	if json.Unmarshal(pagina["selector"], &selector) != nil || json.Unmarshal(pagina["cobertura"], &cobertura) != nil ||
		!clavesOH(selector, []string{"organismo_ref", "unidad_clave", "vigente_en", "conocido_en", "version_rpt_ref", "version_plantilla_ref", "limite", "cursor"}, nil) ||
		!clavesOH(cobertura, colecciones, nil) {
		return errors.New("forma JSON")
	}
	return nil
}

func clavesOH(campos map[string]json.RawMessage, obligatorias, opcionales []string) bool {
	if campos == nil {
		return false
	}
	permitidas := make(map[string]bool, len(obligatorias)+len(opcionales))
	for _, clave := range obligatorias {
		permitidas[clave] = true
		if _, ok := campos[clave]; !ok {
			return false
		}
	}
	for _, clave := range opcionales {
		permitidas[clave] = true
	}
	for clave := range campos {
		if !permitidas[clave] {
			return false
		}
	}
	return true
}

func coberturaOH(v string, n int) bool {
	return (v == "completa" || v == "parcial" || v == "sin_datos") && (v != "sin_datos" || n == 0)
}
func referenciaOpcionalOH(v string) bool { return v == "" || referenciaOH.MatchString(v) }
func cursorOH(v string) bool {
	if len(v) > 256 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func unoDeOH(v string, opciones ...string) bool {
	for _, x := range opciones {
		if v == x {
			return true
		}
	}
	return false
}
func textoOH(v string) bool {
	if v == "" || len([]rune(v)) > 512 || v[0] == ' ' || v[len(v)-1] == ' ' {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func instanteOH(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
func selectorOHIgual(a, b domain.SelectorOrganizacionHistorica) bool {
	return a.OrganismoRef == b.OrganismoRef && a.UnidadClave == b.UnidadClave && a.VigenteEn == b.VigenteEn &&
		instanteOH(a.ConocidoEn) && a.ConocidoEn.Equal(b.ConocidoEn) &&
		a.VersionRPTRef == b.VersionRPTRef && a.VersionPlantillaRef == b.VersionPlantillaRef &&
		a.Limite == b.Limite && a.Cursor == b.Cursor
}
func nuloOrganizacionHistorica(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
func errorOrganizacionHistorica(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code == "42501" {
			return domain.ErrConsultaOrganizacionHistoricaDenegada
		}
		if pg.Code == "22023" {
			return domain.ErrConsultaOrganizacionHistoricaInvalida
		}
	}
	return domain.ErrOrganizacionHistoricaNoDisponible
}
