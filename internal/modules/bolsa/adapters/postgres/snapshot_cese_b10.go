package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	publicacionb10 "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	canonicob10 "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

var ErrCapturaCeseB10NoDisponible = errors.New("bolsa: captura de cese B10 no disponible")

var documentoPublicoB10 = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)

// Los tres puertos son proveedores configurados, versionados y autorizados.
// Esta pieza no consulta Persona, staging RRHH ni tablas de otro módulo.
type ProveedorProyeccionV2CeseB10 interface {
	PrepararProyeccionV2CeseB10(context.Context, time.Time) (proyeccion, manifiesto []byte, anterior time.Time, err error)
}

type SolicitudDocumentoPublicableB10 struct {
	ParticipacionRef string
	ActaRef          string
	FilaNumero       int
}

type DocumentoPublicableB10 struct {
	ParticipacionRef string
	Documento        string
}

type ProveedorDocumentoPublicable interface {
	ResolverDocumentosPublicablesB10(context.Context, time.Time, []SolicitudDocumentoPublicableB10) ([]DocumentoPublicableB10, error)
}

type SolicitudMetadatosBolsaB10 struct {
	BolsaRef     string
	CategoriaRef string
}

type MetadatosBolsaB10 struct {
	BolsaRef       string
	Categoria      string
	CategoriaClave string
	Grupos         []string
}

type ProveedorMetadatosBolsaB10 interface {
	ResolverMetadatosBolsasB10(context.Context, time.Time, []SolicitudMetadatosBolsaB10) ([]MetadatosBolsaB10, error)
}

type ParticipacionFuenteCeseB10 struct {
	Ref                 string
	Orden               int
	Documento           string
	EstadoEfectivo      string
	FechaDisponible     *time.Time
	ParticipacionOrigen string
}

type BolsaFuenteCeseB10 struct {
	Ref             string
	Categoria       string
	CategoriaClave  string
	Grupos          []string
	TipoLista       string
	VigenteDesde    time.Time
	VigenteHasta    *time.Time
	Total           int
	Participaciones []ParticipacionFuenteCeseB10
}

type CapturaCeseB10 struct {
	OrigenPosicion        int64
	OrigenRef             string
	EventoRef             string
	Fase                  string
	BolsaRef              string
	ParticipacionOrigen   string
	Corte                 time.Time
	AnteriorActualizadaEn time.Time
	ProyeccionV2          []byte
	ManifiestoV2          []byte
	Bolsas                []BolsaFuenteCeseB10
}

// CapturadorCeseB10PostgreSQL usa únicamente las funciones de Bolsa 000052.
// El constructor falla cerrado mientras falte cualquiera de los proveedores.
type CapturadorCeseB10PostgreSQL struct {
	pool       *pgxpool.Pool
	v2         ProveedorProyeccionV2CeseB10
	documentos ProveedorDocumentoPublicable
	metadatos  ProveedorMetadatosBolsaB10
}

func NuevoCapturadorCeseB10PostgreSQL(pool *pgxpool.Pool, v2 ProveedorProyeccionV2CeseB10,
	documentos ProveedorDocumentoPublicable, metadatos ProveedorMetadatosBolsaB10,
) (*CapturadorCeseB10PostgreSQL, error) {
	if pool == nil || v2 == nil || documentos == nil || metadatos == nil {
		return nil, ErrCapturaCeseB10NoDisponible
	}
	return &CapturadorCeseB10PostgreSQL{pool: pool, v2: v2, documentos: documentos, metadatos: metadatos}, nil
}

type filaSnapshotCeseB10 struct {
	BolsaRef         string     `json:"bolsa_ref"`
	CategoriaRef     string     `json:"categoria_ref"`
	ActaRef          string     `json:"acta_ref"`
	VigenteDesde     time.Time  `json:"vigente_desde"`
	VigenteHasta     *time.Time `json:"vigente_hasta"`
	Total            int        `json:"total"`
	TipoLista        string     `json:"tipo_lista"`
	ParticipacionRef string     `json:"participacion_ref"`
	FilaNumero       int        `json:"fila_numero"`
	Orden            int        `json:"orden"`
	EstadoEfectivo   string     `json:"estado_efectivo"`
	FechaDisponible  *time.Time `json:"fecha_disponible"`
	EsOrigen         bool       `json:"es_origen"`
}

func (c *CapturadorCeseB10PostgreSQL) CapturarCeseB10(
	ctx context.Context, evento publicacionb10.CesePendienteB10,
) (CapturaCeseB10, error) {
	if ctx == nil || c == nil || c.pool == nil || c.v2 == nil || c.documentos == nil || c.metadatos == nil ||
		evento.OrigenPosicion < 0 || evento.OrigenRef == "" || evento.EventoRef == "" || evento.BolsaRef == "" ||
		(evento.Fase != "cese" && evento.Fase != "vencimiento") {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return CapturaCeseB10{}, err
	}
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadWrite})
	if err != nil {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	defer tx.Rollback(context.Background())
	var corte time.Time
	var filasJSON, proyeccionGuardada, manifiestoGuardado, bolsasGuardadas []byte
	var anteriorGuardada *time.Time
	err = tx.QueryRow(ctx, `SELECT corte,filas,anterior_actualizada_en,proyeccion_v2,manifiesto_v2,bolsas_v1
		FROM vec_bolsa_llamamientos.capturar_cese_b10_v1($1::bigint,$2::text,$3::text,$4::text,$5::text)`,
		evento.OrigenPosicion, evento.OrigenRef, evento.EventoRef, evento.BolsaRef, evento.Fase,
	).Scan(&corte, &filasJSON, &anteriorGuardada, &proyeccionGuardada, &manifiestoGuardado, &bolsasGuardadas)
	if err != nil {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	var filas []filaSnapshotCeseB10
	if json.Unmarshal(filasJSON, &filas) != nil || len(filas) == 0 || len(filas) > 1_000_000 {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	var participacionOrigen string
	for _, fila := range filas {
		if fila.EsOrigen {
			if participacionOrigen != "" || fila.BolsaRef != evento.BolsaRef {
				return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
			}
			participacionOrigen = fila.ParticipacionRef
		}
	}
	if participacionOrigen == "" {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	var proyeccion, manifiesto, bolsasJSON []byte
	var anterior time.Time
	if len(bolsasGuardadas) > 0 {
		proyeccion, manifiesto, bolsasJSON = proyeccionGuardada, manifiestoGuardado, bolsasGuardadas
		if anteriorGuardada != nil {
			anterior = *anteriorGuardada
		}
	} else {
		proyeccion, manifiesto, anterior, err = c.v2.PrepararProyeccionV2CeseB10(ctx, corte)
		if err != nil || (!anterior.IsZero() && !anterior.Before(corte)) {
			return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
		}
		bolsas, err := c.construirBolsas(ctx, corte, filas)
		if err != nil {
			return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
		}
		bolsasJSON, err = serializarBolsasCeseB10(corte, bolsas)
		if err != nil {
			return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
		}
		if _, err = canonicob10.PrepararMaterialPublicacionV3(proyeccion, manifiesto, bolsasJSON); err != nil {
			return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
		}
		var anteriorSQL *time.Time
		if !anterior.IsZero() {
			anteriorSQL = &anterior
		}
		var reutilizada bool
		if err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.guardar_material_cese_b10_v1(
			$1::text,$2::text,$3::timestamptz,$4::timestamptz,$5::bytea,$6::bytea,$7::bytea)`,
			evento.EventoRef, evento.Fase, corte, anteriorSQL, proyeccion, manifiesto, bolsasJSON,
		).Scan(&reutilizada); err != nil || reutilizada {
			return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	bolsas, err := cotejarMaterialBolsasCeseB10(corte, filas, bolsasJSON)
	if err != nil {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	if _, err = canonicob10.PrepararMaterialPublicacionV3(proyeccion, manifiesto, bolsasJSON); err != nil {
		return CapturaCeseB10{}, ErrCapturaCeseB10NoDisponible
	}
	return CapturaCeseB10{OrigenPosicion: evento.OrigenPosicion, OrigenRef: evento.OrigenRef,
		EventoRef: evento.EventoRef, Fase: evento.Fase, BolsaRef: evento.BolsaRef,
		ParticipacionOrigen: participacionOrigen,
		Corte:               corte, AnteriorActualizadaEn: anterior, ProyeccionV2: proyeccion,
		ManifiestoV2: manifiesto, Bolsas: bolsas}, nil
}

func (c *CapturadorCeseB10PostgreSQL) construirBolsas(ctx context.Context, corte time.Time,
	filas []filaSnapshotCeseB10,
) ([]BolsaFuenteCeseB10, error) {
	solicitudesDoc := make([]SolicitudDocumentoPublicableB10, 0, len(filas))
	solicitudesMeta := make([]SolicitudMetadatosBolsaB10, 0)
	vistas := make(map[string]struct{})
	for _, fila := range filas {
		if fila.ParticipacionRef == "" || fila.BolsaRef == "" || fila.ActaRef == "" || fila.FilaNumero < 1 || fila.Orden < 1 {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		solicitudesDoc = append(solicitudesDoc, SolicitudDocumentoPublicableB10{
			ParticipacionRef: fila.ParticipacionRef, ActaRef: fila.ActaRef, FilaNumero: fila.FilaNumero,
		})
		if _, existe := vistas[fila.BolsaRef]; !existe {
			vistas[fila.BolsaRef] = struct{}{}
			solicitudesMeta = append(solicitudesMeta, SolicitudMetadatosBolsaB10{BolsaRef: fila.BolsaRef, CategoriaRef: fila.CategoriaRef})
		}
	}
	docs, err := c.documentos.ResolverDocumentosPublicablesB10(ctx, corte, solicitudesDoc)
	if err != nil || len(docs) != len(solicitudesDoc) {
		return nil, ErrCapturaCeseB10NoDisponible
	}
	metas, err := c.metadatos.ResolverMetadatosBolsasB10(ctx, corte, solicitudesMeta)
	if err != nil || len(metas) != len(solicitudesMeta) {
		return nil, ErrCapturaCeseB10NoDisponible
	}
	porRef := make(map[string]string, len(docs))
	for _, doc := range docs {
		if _, duplicado := porRef[doc.ParticipacionRef]; duplicado || !documentoPublicoB10.MatchString(doc.Documento) {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		porRef[doc.ParticipacionRef] = doc.Documento
	}
	metaPorBolsa := make(map[string]MetadatosBolsaB10, len(metas))
	for _, meta := range metas {
		if _, duplicada := metaPorBolsa[meta.BolsaRef]; duplicada || meta.Categoria == "" || meta.CategoriaClave == "" || len(meta.Grupos) == 0 {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		metaPorBolsa[meta.BolsaRef] = meta
	}
	bolsas := make([]BolsaFuenteCeseB10, 0, len(metas))
	for _, fila := range filas {
		documento, existeDoc := porRef[fila.ParticipacionRef]
		meta, existeMeta := metaPorBolsa[fila.BolsaRef]
		if !existeDoc || !existeMeta {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		if len(bolsas) == 0 || bolsas[len(bolsas)-1].Ref != fila.BolsaRef {
			bolsas = append(bolsas, BolsaFuenteCeseB10{Ref: fila.BolsaRef, Categoria: meta.Categoria,
				CategoriaClave: meta.CategoriaClave, Grupos: append([]string(nil), meta.Grupos...),
				TipoLista: fila.TipoLista, VigenteDesde: fila.VigenteDesde, VigenteHasta: fila.VigenteHasta,
				Total: fila.Total, Participaciones: make([]ParticipacionFuenteCeseB10, 0, fila.Total)})
		}
		actual := &bolsas[len(bolsas)-1]
		if actual.Total != fila.Total || actual.TipoLista != fila.TipoLista ||
			!actual.VigenteDesde.Equal(fila.VigenteDesde) || !instantesOpcionalesIguales(actual.VigenteHasta, fila.VigenteHasta) ||
			fila.Orden != len(actual.Participaciones)+1 {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		estado, err := estadoPublicoCeseB10(fila.EstadoEfectivo, fila.FechaDisponible, corte)
		if err != nil {
			return nil, err
		}
		actual.Participaciones = append(actual.Participaciones, ParticipacionFuenteCeseB10{
			Ref: fila.ParticipacionRef, Orden: fila.Orden, Documento: documento,
			EstadoEfectivo: estado, FechaDisponible: fila.FechaDisponible,
		})
	}
	for _, bolsa := range bolsas {
		if len(bolsa.Participaciones) != bolsa.Total {
			return nil, ErrCapturaCeseB10NoDisponible
		}
	}
	return bolsas, nil
}

func estadoPublicoCeseB10(efectivo string, disponible *time.Time, corte time.Time) (string, error) {
	switch efectivo {
	case "disponible":
		return "disponible", nil
	case "trabajando", "pendiente_incorporacion":
		return "ocupado", nil
	case "disponible_desde":
		if disponible == nil {
			return "", ErrCapturaCeseB10NoDisponible
		}
		if !disponible.After(corte) {
			return "disponible", nil
		}
		return "no_disponible", nil
	case "no_disponible":
		return "no_disponible", nil
	case "excluido":
		return "excluido", nil
	case "renuncia":
		return "renuncia_pendiente", nil
	default:
		return "", ErrCapturaCeseB10NoDisponible
	}
}

func serializarBolsasCeseB10(corte time.Time, bolsas []BolsaFuenteCeseB10) ([]byte, error) {
	publicas := make([]canonicob10.BolsaManifiestoV1, len(bolsas))
	for i, bolsa := range bolsas {
		posiciones := make([]canonicob10.PosicionBolsaManifiestoV1, len(bolsa.Participaciones))
		for j, p := range bolsa.Participaciones {
			posiciones[j] = canonicob10.PosicionBolsaManifiestoV1{
				Orden: p.Orden, DocumentoEnmascarado: p.Documento, EstadoClave: p.EstadoEfectivo,
			}
		}
		publicas[i] = canonicob10.BolsaManifiestoV1{BolsaRef: bolsa.Ref, Categoria: bolsa.Categoria,
			CategoriaClave: bolsa.CategoriaClave, Grupos: bolsa.Grupos, TipoLista: bolsa.TipoLista,
			VigenteDesde: bolsa.VigenteDesde, VigenteHasta: bolsa.VigenteHasta,
			Total: bolsa.Total, Posiciones: posiciones}
	}
	return json.Marshal(struct {
		GeneradoEn string                          `json:"generado_en"`
		Bolsas     []canonicob10.BolsaManifiestoV1 `json:"bolsas"`
	}{GeneradoEn: corte.UTC().Format("2006-01-02T15:04:05.000000Z"), Bolsas: publicas})
}

func cotejarMaterialBolsasCeseB10(corte time.Time, filas []filaSnapshotCeseB10,
	material []byte,
) ([]BolsaFuenteCeseB10, error) {
	var publico canonicob10.BolsasManifiestoV1
	if json.Unmarshal(material, &publico) != nil || !publico.GeneradoEn.Equal(corte) {
		return nil, ErrCapturaCeseB10NoDisponible
	}
	porRef := make(map[string]canonicob10.BolsaManifiestoV1, len(publico.Bolsas))
	for _, bolsa := range publico.Bolsas {
		if _, duplicada := porRef[bolsa.BolsaRef]; duplicada {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		porRef[bolsa.BolsaRef] = bolsa
	}
	bolsas := make([]BolsaFuenteCeseB10, 0, len(publico.Bolsas))
	for _, fila := range filas {
		b, existe := porRef[fila.BolsaRef]
		if !existe || fila.Orden < 1 || fila.Orden > len(b.Posiciones) ||
			fila.Total != b.Total || fila.TipoLista != b.TipoLista ||
			!fila.VigenteDesde.Equal(b.VigenteDesde) || !instantesOpcionalesIguales(fila.VigenteHasta, b.VigenteHasta) {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		p := b.Posiciones[fila.Orden-1]
		estado, err := estadoPublicoCeseB10(fila.EstadoEfectivo, fila.FechaDisponible, corte)
		if err != nil || p.Orden != fila.Orden || p.EstadoClave != estado || !documentoPublicoB10.MatchString(p.DocumentoEnmascarado) {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		if len(bolsas) == 0 || bolsas[len(bolsas)-1].Ref != fila.BolsaRef {
			bolsas = append(bolsas, BolsaFuenteCeseB10{Ref: b.BolsaRef, Categoria: b.Categoria,
				CategoriaClave: b.CategoriaClave, Grupos: append([]string(nil), b.Grupos...),
				TipoLista: b.TipoLista, VigenteDesde: b.VigenteDesde, VigenteHasta: b.VigenteHasta,
				Total: b.Total, Participaciones: make([]ParticipacionFuenteCeseB10, 0, b.Total)})
		}
		actual := &bolsas[len(bolsas)-1]
		if len(actual.Participaciones)+1 != fila.Orden {
			return nil, ErrCapturaCeseB10NoDisponible
		}
		actual.Participaciones = append(actual.Participaciones, ParticipacionFuenteCeseB10{
			Ref: fila.ParticipacionRef, Orden: fila.Orden, Documento: p.DocumentoEnmascarado,
			EstadoEfectivo: estado, FechaDisponible: fila.FechaDisponible,
		})
	}
	if len(bolsas) != len(publico.Bolsas) {
		return nil, ErrCapturaCeseB10NoDisponible
	}
	for _, b := range bolsas {
		if len(b.Participaciones) != b.Total {
			return nil, ErrCapturaCeseB10NoDisponible
		}
	}
	return bolsas, nil
}

func instantesOpcionalesIguales(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}
