package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

var ErrRepositorioNoDisponible = errors.New("documentos: repositorio PostgreSQL no disponible")

// Repositorio usa exclusivamente las fachadas nominales del esquema documental.
// La conexión debe pertenecer a un LOGIN con la única membresía técnica
// vec_documentos_ejecutor; las funciones SQL verifican esto de nuevo.
type Repositorio struct{ db *pgxpool.Pool }

func NuevoRepositorio(db *pgxpool.Pool) (*Repositorio, error) {
	if db == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &Repositorio{db: db}, nil
}

var _ ports.Repositorio = (*Repositorio)(nil)

type materialV3 struct {
	capacidad, decision, motivo, contexto []byte
	persona, perfil                       int64
	payload, sobre, evidencia, raiz       []byte
}

func validarAutorizacion(a ports.AutorizacionV3, accion string, preimagen []byte) (materialV3, error) {
	if a.ValidarPara(accion, time.Now().UTC()) != nil || len(preimagen) == 0 {
		return materialV3{}, ports.ErrSolicitudInvalida
	}
	resumen := a.Material.ResumenCapacidad()
	if resumen.Operacion() != accion || resumen.EfectoHuellaSHA256() != ports.HuellaPreimagen(preimagen) {
		return materialV3{}, ports.ErrSolicitudInvalida
	}
	return materialV3{
		capacidad: a.Material.CapacidadCanonica(), decision: a.Material.DecisionCanonica(),
		motivo: a.Material.MotivoCanonico(), contexto: a.Material.ContextoActorCanonico(),
		persona: int64(a.Material.PersonaVersion()), perfil: int64(a.Material.PerfilVersion()),
		payload: a.Material.PayloadVECAD3(), sobre: a.Material.SobreCOSESign1(),
		evidencia: a.Material.EvidenciaVerificacion(), raiz: a.Material.RaizPublicaSPKI(),
	}, nil
}

func autorizacionJSON(a ports.AutorizacionV3) ([]byte, error) {
	return json.Marshal(struct {
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
		RecursoRef      string `json:"recurso_ref"`
		AmbitoRef       string `json:"ambito_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		CorrelacionRef  string `json:"correlacion_ref"`
	}{a.Accion, a.Finalidad, a.RecursoRef, a.AmbitoRef, a.PrincipalID, a.PerfilActivoRef, a.CorrelacionRef})
}

func (r *Repositorio) transaccion(ctx context.Context, funcion string, args ...any) ([]byte, error) {
	if r == nil || r.db == nil || ctx == nil {
		return nil, ErrRepositorioNoDisponible
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, ErrRepositorioNoDisponible
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	for _, ajuste := range []string{"SET LOCAL timezone='UTC'", "SET LOCAL statement_timeout='10s'", "SET LOCAL lock_timeout='2s'"} {
		if _, err = tx.Exec(ctx, ajuste); err != nil {
			return nil, ErrRepositorioNoDisponible
		}
	}
	var resultado []byte
	if err = tx.QueryRow(ctx, funcion, args...).Scan(&resultado); err != nil {
		return nil, ErrRepositorioNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, ErrRepositorioNoDisponible
	}
	return resultado, nil
}

type documentoJSON struct {
	ID                   string    `json:"id"`
	NumeroVEC            string    `json:"numero_vec"`
	ModuloID             string    `json:"modulo_id"`
	ExpedienteRef        string    `json:"expediente_ref"`
	TipoRef              string    `json:"tipo_ref"`
	Version              uint64    `json:"version"`
	MIME                 string    `json:"mime"`
	HuellaSHA256         string    `json:"huella_sha256"`
	Tamano               int64     `json:"tamano"`
	ObjetoRef            string    `json:"objeto_ref"`
	ObjetoVersion        string    `json:"objeto_version"`
	PoliticaRef          string    `json:"politica_ref"`
	VersionPolitica      uint64    `json:"version_politica"`
	HuellaPoliticaSHA256 string    `json:"huella_politica_sha256"`
	ConservacionHasta    time.Time `json:"conservacion_hasta"`
	Proteccion           string    `json:"proteccion"`
	EstadoFirma          string    `json:"estado_firma"`
	CreadoEn             time.Time `json:"creado_en"`
	// Las proyecciones v1 de originales no llevan custodia; la lista v2 y el
	// registro externo la declaran siempre.
	Custodia    string `json:"custodia"`
	CustodioID  string `json:"custodio_id"`
	CustodiaRef string `json:"custodia_ref"`
}

func decodificarDocumento(raw []byte) (domain.Documento, error) {
	var v documentoJSON
	if err := json.Unmarshal(raw, &v); err != nil {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	d := domain.Documento{
		ID: v.ID, NumeroVEC: v.NumeroVEC, ModuloID: v.ModuloID, ExpedienteRef: v.ExpedienteRef,
		TipoRef: v.TipoRef, Version: v.Version, MIME: v.MIME, HuellaSHA256: v.HuellaSHA256,
		Tamano: v.Tamano, ObjetoRef: v.ObjetoRef, ObjetoVersion: v.ObjetoVersion,
		PoliticaRef: v.PoliticaRef, VersionPolitica: v.VersionPolitica,
		HuellaPoliticaSHA256: v.HuellaPoliticaSHA256, ConservacionHasta: v.ConservacionHasta,
		Proteccion: v.Proteccion, EstadoFirma: v.EstadoFirma, CreadoEn: v.CreadoEn,
		Custodia: v.Custodia,
	}
	switch v.Custodia {
	case "":
		d.Custodia = domain.CustodiaVEC
	case domain.CustodiaExterna:
		d.CustodiaExternaRef = domain.ReferenciaCustodiaExterna{CustodioID: v.CustodioID, Referencia: v.CustodiaRef, HuellaSHA256: v.HuellaSHA256}
	}
	if d.Validar() != nil {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	return d, nil
}

func (r *Repositorio) ConfirmarAlta(ctx context.Context, a ports.AltaPersistente) (domain.Documento, error) {
	preimagen, err := a.PreimagenAlta()
	if err != nil || a.Objeto.Validar() != nil || a.Objeto.Objeto.HuellaSHA256 != a.HuellaSHA256 ||
		a.Objeto.Objeto.MIME != a.MIME || a.Objeto.Objeto.Tamano != a.Tamano ||
		a.Objeto.Evidencia.Objeto != a.Objeto.Objeto.Objeto || a.Autorizacion.RecursoRef != a.ID ||
		a.Autorizacion.AmbitoRef != a.ExpedienteRef ||
		a.Objeto.Objeto.RetenidoHasta.Before(a.Politica.Politica().ConservacionHasta()) ||
		(a.Politica.Politica().Proteccion() == "bloqueo" && !a.Objeto.Objeto.Inmovilizado) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	material, err := validarAutorizacion(a.Autorizacion, ports.AccionAlta, preimagen)
	if err != nil {
		return domain.Documento{}, err
	}
	auth, _ := autorizacionJSON(a.Autorizacion)
	recibo, err := json.Marshal(a.Objeto.Evidencia)
	if err != nil {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	suma := sha256.Sum256(recibo)
	objeto, _ := json.Marshal(struct {
		ObjetoRef                string    `json:"objeto_ref"`
		ObjetoVersion            string    `json:"objeto_version"`
		ConectorRef              string    `json:"conector_ref"`
		ReciboObjetoRef          string    `json:"recibo_objeto_ref"`
		ReciboObjetoHuellaSHA256 string    `json:"recibo_objeto_huella_sha256"`
		RetenidoHasta            time.Time `json:"retenido_hasta"`
		Inmovilizado             bool      `json:"inmovilizado"`
		MIME                     string    `json:"mime"`
		Tamano                   int64     `json:"tamano"`
		HuellaSHA256             string    `json:"huella_sha256"`
	}{a.Objeto.Objeto.Objeto.Referencia, a.Objeto.Objeto.Objeto.Version,
		a.Objeto.Objeto.ConectorID, a.Objeto.Evidencia.OperacionRef, hex.EncodeToString(suma[:]),
		a.Objeto.Objeto.RetenidoHasta, a.Objeto.Objeto.Inmovilizado,
		a.MIME, a.Tamano, a.HuellaSHA256})
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.confirmar_alta_v2($1,$2::jsonb,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		preimagen, string(objeto), string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return domain.Documento{}, err
	}
	return decodificarDocumento(raw)
}

// ConfirmarReferenciaExterna registra la referencia y huella de un original
// custodiado fuera de VEC. No hay objeto ni recibo de almacén que cotejar.
func (r *Repositorio) ConfirmarReferenciaExterna(ctx context.Context, a ports.AltaExternaPersistente) (domain.Documento, error) {
	preimagen, err := a.PreimagenExterna()
	if err != nil || a.Autorizacion.RecursoRef != a.ID || a.Autorizacion.AmbitoRef != a.ExpedienteRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	material, err := validarAutorizacion(a.Autorizacion, ports.AccionRegistrarExterno, preimagen)
	if err != nil {
		return domain.Documento{}, err
	}
	auth, _ := autorizacionJSON(a.Autorizacion)
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.registrar_referencia_externa_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return domain.Documento{}, err
	}
	d, err := decodificarDocumento(raw)
	if err != nil || d.Custodia != domain.CustodiaExterna {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	return d, nil
}

func (r *Repositorio) ListarExpediente(ctx context.Context, c ports.ConsultaExpediente) (ports.PaginaDocumentos, error) {
	if !domain.ReferenciaOpacaValida(c.ExpedienteRef) || c.Autorizacion.RecursoRef != c.ExpedienteRef ||
		c.Autorizacion.AmbitoRef != c.ExpedienteRef || c.Limite < 1 || c.Limite > 100 ||
		(c.Cursor != "" && !domain.ReferenciaOpacaValida(c.Cursor)) {
		return ports.PaginaDocumentos{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := c.PreimagenListar()
	if err != nil {
		return ports.PaginaDocumentos{}, err
	}
	material, err := validarAutorizacion(c.Autorizacion, ports.AccionListar, preimagen)
	if err != nil {
		return ports.PaginaDocumentos{}, err
	}
	auth, _ := autorizacionJSON(c.Autorizacion)
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.listar_expediente_v2($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return ports.PaginaDocumentos{}, err
	}
	var pagina struct {
		Items           []json.RawMessage `json:"items"`
		SiguienteCursor string            `json:"siguiente_cursor"`
	}
	if json.Unmarshal(raw, &pagina) != nil || len(pagina.Items) > int(c.Limite) {
		return ports.PaginaDocumentos{}, ErrRepositorioNoDisponible
	}
	resultado := ports.PaginaDocumentos{SiguienteCursor: pagina.SiguienteCursor}
	for _, item := range pagina.Items {
		d, err := decodificarDocumento(item)
		if err != nil {
			return ports.PaginaDocumentos{}, err
		}
		resultado.Items = append(resultado.Items, d)
	}
	if resultado.SiguienteCursor != "" && !domain.ReferenciaOpacaValida(resultado.SiguienteCursor) {
		return ports.PaginaDocumentos{}, ErrRepositorioNoDisponible
	}
	return resultado, nil
}

func (r *Repositorio) Obtener(ctx context.Context, c ports.ConsultaDocumento) (domain.Documento, error) {
	if !domain.ReferenciaOpacaValida(c.DocumentoID) || c.Version == 0 ||
		c.Autorizacion.RecursoRef != c.DocumentoID || !domain.ReferenciaOpacaValida(c.Autorizacion.AmbitoRef) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := c.PreimagenDescargar()
	if err != nil {
		return domain.Documento{}, err
	}
	material, err := validarAutorizacion(c.Autorizacion, ports.AccionDescargar, preimagen)
	if err != nil {
		return domain.Documento{}, err
	}
	auth, _ := autorizacionJSON(c.Autorizacion)
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.obtener_original_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return domain.Documento{}, err
	}
	return decodificarDocumento(raw)
}

func (r *Repositorio) ConfirmarPreparacion(ctx context.Context, p ports.PreparacionNotificacion) (domain.NotificacionPreparada, error) {
	if !domain.ReferenciaOpacaValida(p.ID) || !domain.ReferenciaOpacaValida(p.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(p.DocumentoID) ||
		!domain.ReferenciaOpacaValida(p.DestinatarioRef) || !domain.IdentificadorTecnicoValido(p.Canal) ||
		p.Version == 0 || p.Autorizacion.RecursoRef != p.ID || !domain.ReferenciaOpacaValida(p.Autorizacion.AmbitoRef) {
		return domain.NotificacionPreparada{}, ports.ErrSolicitudInvalida
	}
	preimagen, err := p.PreimagenPreparar()
	if err != nil {
		return domain.NotificacionPreparada{}, err
	}
	material, err := validarAutorizacion(p.Autorizacion, ports.AccionPrepararNotificacion, preimagen)
	if err != nil {
		return domain.NotificacionPreparada{}, err
	}
	auth, _ := autorizacionJSON(p.Autorizacion)
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.preparar_notificacion_v2($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return domain.NotificacionPreparada{}, err
	}
	var n struct {
		ID              string    `json:"id"`
		DocumentoID     string    `json:"documento_id"`
		Version         uint64    `json:"version"`
		DestinatarioRef string    `json:"destinatario_ref"`
		Canal           string    `json:"canal"`
		PreparadaEn     time.Time `json:"preparada_en"`
	}
	if json.Unmarshal(raw, &n) != nil {
		return domain.NotificacionPreparada{}, ErrRepositorioNoDisponible
	}
	resultado := domain.NotificacionPreparada{ID: n.ID, DocumentoID: n.DocumentoID,
		Version: n.Version, DestinatarioRef: n.DestinatarioRef, Canal: n.Canal, PreparadaEn: n.PreparadaEn}
	if resultado.Validar() != nil {
		return domain.NotificacionPreparada{}, fmt.Errorf("%w: resultado", ErrRepositorioNoDisponible)
	}
	return resultado, nil
}
