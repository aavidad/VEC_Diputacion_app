package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionPrepararSubsanacionReparos  = "vec_contratacion_temporal.preparar_subsanacion_reparos_v1"
	funcionConfirmarSubsanacionReparos = "vec_contratacion_temporal.confirmar_subsanacion_reparos_v1"
	esquemaPrepararSubsanacionReparos  = "vec.contratacion-temporal.preparar-subsanacion-reparos.v1"
	esquemaConfirmarSubsanacionReparos = "vec.contratacion-temporal.confirmar-subsanacion-reparos.v1"
	esquemaRespuestaSubsanacionReparos = "vec.contratacion-temporal.resultado-subsanacion-reparos.v1"
)

// La preparación solo consulta o propone referencias. Su confirmación durable
// pertenece a la misma transacción que consume la autorización y escribe el efecto.
type PreparadorSubsanacionReparosPostgreSQL struct {
	pool      iniciadorTransacciones
	generador ports.GeneradorReferenciasSubsanacionReparo
}

func NuevoPreparadorSubsanacionReparosPostgreSQL(pool *pgxpool.Pool, generador ports.GeneradorReferenciasSubsanacionReparo) (*PreparadorSubsanacionReparosPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(generador) {
		return nil, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return &PreparadorSubsanacionReparosPostgreSQL{pool: pool, generador: generador}, nil
}

type materialSubsanacionSQL struct {
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
	VersionEsperada uint64 `json:"version_esperada"`
	ActorRef        string `json:"actor_ref"`
	PerfilRef       string `json:"perfil_ref"`
	Observaciones   string `json:"observaciones"`
}

func materialSubsanacionParaSQL(m ports.MaterialSubsanacionReparo) materialSubsanacionSQL {
	return materialSubsanacionSQL{m.OrganizacionRef, m.ExpedienteRef, m.VersionEsperada, m.ActorRef, m.PerfilRef, m.Observaciones}
}

type referenciasSubsanacionSQL struct {
	ReservaRef string `json:"reserva_ref"`
	ReciboRef  string `json:"recibo_ref"`
	EventoRef  string `json:"evento_ref"`
}

type operacionPrepararSubsanacionSQL struct {
	Esquema     string                    `json:"esquema"`
	Operacion   string                    `json:"operacion"`
	Material    materialSubsanacionSQL    `json:"material"`
	SellosHMAC  sellosPrepararAltaV2      `json:"sellos_hmac"`
	Referencias referenciasSubsanacionSQL `json:"referencias_candidatas"`
}

type reciboSubsanacionSQL struct {
	Operacion         string                 `json:"operacion"`
	OrganizacionRef   string                 `json:"organizacion_ref"`
	ExpedienteRef     string                 `json:"expediente_ref"`
	VersionAnterior   uint64                 `json:"version_anterior"`
	VersionResultante uint64                 `json:"version_resultante"`
	FaseResultante    domain.ClaveFase       `json:"fase_resultante"`
	EstadoResultante  domain.EstadoOperativo `json:"estado_resultante"`
	ReciboRef         string                 `json:"recibo_ref"`
	AuditoriaRef      string                 `json:"auditoria_ref"`
	EventoRef         string                 `json:"evento_ref"`
	ActorRef          string                 `json:"actor_ref"`
	RegistradaEn      time.Time              `json:"registrada_en"`
}

func (r reciboSubsanacionSQL) puertos() ports.ReciboSubsanacionReparo {
	return ports.ReciboSubsanacionReparo{Operacion: r.Operacion, OrganizacionRef: r.OrganizacionRef, ExpedienteRef: r.ExpedienteRef, VersionAnterior: r.VersionAnterior, VersionResultante: r.VersionResultante, FaseResultante: r.FaseResultante, EstadoResultante: r.EstadoResultante, ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef, ActorRef: r.ActorRef, RegistradaEn: r.RegistradaEn}
}

type respuestaSubsanacionSQL struct {
	Esquema     string                     `json:"esquema"`
	Resultado   string                     `json:"resultado"`
	Material    *materialSubsanacionSQL    `json:"material,omitempty"`
	Expediente  *domain.Expediente         `json:"expediente,omitempty"`
	Referencias *referenciasSubsanacionSQL `json:"referencias,omitempty"`
	RetornoRef  string                     `json:"retorno_ref,omitempty"`
	AmbitoHMAC  string                     `json:"ambito_idempotencia_hmac,omitempty"`
	HuellaHMAC  string                     `json:"huella_peticion_hmac,omitempty"`
	Recibo      *reciboSubsanacionSQL      `json:"recibo,omitempty"`
}

func decodificarRespuestaSubsanacionSQL(contenido []byte) (respuestaSubsanacionSQL, error) {
	var r respuestaSubsanacionSQL
	if len(contenido) == 0 || len(contenido) > maximoCargaConfirmarFiscalizacion || decodificarJSONEstricto(contenido, &r) != nil || r.Esquema != esquemaRespuestaSubsanacionReparos {
		return r, ports.ErrResultadoFiscalizacionNoConfiable
	}
	switch r.Resultado {
	case "preparada", "confirmada":
		return r, nil
	case "idempotencia_reutilizada":
		return r, ports.ErrClaveIdempotenciaUsada
	case "version_en_conflicto":
		return r, domain.ErrVersionEnConflicto
	case "acceso_denegado":
		return r, ports.ErrAutorizacionDenegada
	default:
		return r, ports.ErrResultadoFiscalizacionNoConfiable
	}
}

func (p *PreparadorSubsanacionReparosPostgreSQL) PrepararSubsanacionReparo(ctx context.Context, s ports.SolicitudPrepararSubsanacionReparo) (ports.PreparacionSubsanacionReparo, error) {
	if p == nil || dependenciaNula(p.pool) || dependenciaNula(p.generador) || ctx == nil || !s.Material.Valido() || s.AmbitosHMAC.ValidarDominio(ports.DominioAmbitoIdempotenciaSubsanacionReparo) != nil || s.HuellasPeticionHMAC.ValidarDominio(ports.DominioHuellaPeticionSubsanacionReparo) != nil {
		return ports.PreparacionSubsanacionReparo{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	refs, err := p.generador.GenerarReferenciasSubsanacionReparo(ctx)
	if err != nil || !refs.Validas() {
		return ports.PreparacionSubsanacionReparo{}, errorDependenciaFiscalizacion(ctx)
	}
	sellos, err := nuevosSellosPrepararAltaV2(s.AmbitosHMAC, s.HuellasPeticionHMAC)
	if err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	op := operacionPrepararSubsanacionSQL{esquemaPrepararSubsanacionReparos, ports.OperacionRegistrarSubsanacionReparo, materialSubsanacionParaSQL(s.Material), sellos, referenciasSubsanacionSQL{refs.ReservaRef, refs.ReciboRef, refs.EventoRef}}
	cuerpo, err := json.Marshal(op)
	if err != nil || len(cuerpo) > maximoCargaConfirmarFiscalizacion {
		return ports.PreparacionSubsanacionReparo{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	defer borrarBytes(cuerpo)
	for intento := 0; intento < maximoIntentosPrepararFiscalizacion; intento++ {
		preparada, causa := p.prepararEnTransaccion(ctx, s, op, cuerpo)
		if causa == nil {
			return preparada, nil
		}
		if ctx.Err() != nil || !errorPostgreSQLReintentable(causa) || intento+1 == maximoIntentosPrepararFiscalizacion {
			return ports.PreparacionSubsanacionReparo{}, normalizarErrorSubsanacionSQL(ctx, causa)
		}
	}
	return ports.PreparacionSubsanacionReparo{}, ports.ErrPersistenciaFiscalizacionNoDisponible
}

func (p *PreparadorSubsanacionReparosPostgreSQL) prepararEnTransaccion(ctx context.Context, s ports.SolicitudPrepararSubsanacionReparo, op operacionPrepararSubsanacionSQL, cuerpo []byte) (ports.PreparacionSubsanacionReparo, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	defer revertirTransaccion(tx)
	if err = configurarTransaccionSubsanacion(ctx, tx); err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, "SELECT "+funcionPrepararSubsanacionReparos+"($1::jsonb)::text", cuerpo).Scan(&contenido); err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	defer borrarBytes(contenido)
	r, err := decodificarRespuestaSubsanacionSQL(contenido)
	if err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	if r.Material == nil || *r.Material != op.Material || r.Expediente == nil || r.Referencias == nil || !op.SellosHMAC.contienePar(r.AmbitoHMAC, r.HuellaHMAC) {
		return ports.PreparacionSubsanacionReparo{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	preparada := ports.PreparacionSubsanacionReparo{Material: s.Material, Expediente: *r.Expediente, ReservaRef: r.Referencias.ReservaRef, ReciboRef: r.Referencias.ReciboRef, EventoRef: r.Referencias.EventoRef, RetornoRef: r.RetornoRef, AmbitoIdempotenciaHMAC: r.AmbitoHMAC, HuellaPeticionHMAC: r.HuellaHMAC, Confirmada: r.Resultado == "confirmada"}
	if r.Recibo != nil {
		recibo := r.Recibo.puertos()
		preparada.ReciboConfirmado = &recibo
	}
	if !preparacionSubsanacionSQLValida(preparada) {
		return ports.PreparacionSubsanacionReparo{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.PreparacionSubsanacionReparo{}, err
	}
	return preparada, nil
}

func preparacionSubsanacionSQLValida(p ports.PreparacionSubsanacionReparo) bool {
	e := p.Expediente
	n := p.Material.VersionEsperada
	if p.Confirmada {
		n++
	}
	if !ports.VersionOperacionAnalisisConIncrementoValida(p.Material.VersionEsperada) || e.Validar() != nil || e.Referencia != p.Material.ExpedienteRef || e.OrganizacionRef != p.Material.OrganizacionRef || e.Version != n || e.Asignacion == nil || e.Fiscalizacion == nil || e.Fiscalizacion.Retorno == nil || e.Fiscalizacion.Resultado != domain.FiscalizacionDesfavorable || e.FaseActual != domain.FaseSubsanacionUnidad || e.EstadoActual != domain.EstadoIncidencia || e.Fiscalizacion.Retorno.RetornoRef != p.RetornoRef || !(ports.ReferenciasEfectoSubsanacionReparo{ReservaRef: p.ReservaRef, ReciboRef: p.ReciboRef, EventoRef: p.EventoRef}).Validas() {
		return false
	}
	if !p.Confirmada {
		return p.ReciboConfirmado == nil
	}
	if p.ReciboConfirmado == nil {
		return false
	}
	return p.ReciboConfirmado.ValidarPara(ports.OrdenConfirmarSubsanacionReparo{OrganizacionRef: p.Material.OrganizacionRef, Expediente: e, VersionAnterior: p.Material.VersionEsperada, ClaveIdempotencia: p.Material.ClaveIdempotencia, ActorRef: p.Material.ActorRef, UnidadRef: e.Asignacion.UnidadRef, ReciboRef: p.ReciboRef, EventoRef: p.EventoRef, RegistradaEn: p.ReciboConfirmado.RegistradaEn, Material: p.Material, Preparacion: p})
}

type proveedorMaterialConfirmacionSubsanacionReparo interface {
	ProveerMaterialConfirmacionSubsanacionReparo(context.Context, ports.OrdenConfirmarSubsanacionReparo) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type TransaccionSubsanacionReparosPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor proveedorMaterialConfirmacionSubsanacionReparo
}

func NuevaTransaccionSubsanacionReparosPostgreSQL(pool *pgxpool.Pool, proveedor proveedorMaterialConfirmacionSubsanacionReparo) (*TransaccionSubsanacionReparosPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrPersistenciaFiscalizacionNoDisponible
	}
	return &TransaccionSubsanacionReparosPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

type politicaSubsanacionSQL struct {
	DefinicionRef          string               `json:"definicion_ref"`
	DefinicionVersion      uint64               `json:"definicion_version"`
	DefinicionHuellaSHA256 string               `json:"definicion_huella_sha256"`
	Accion                 domain.ClaveCatalogo `json:"accion"`
	Finalidad              domain.ClaveCatalogo `json:"finalidad"`
	EvaluadaEn             time.Time            `json:"evaluada_en"`
	ValidaHasta            time.Time            `json:"valida_hasta"`
}

type operacionConfirmarSubsanacionSQL struct {
	Esquema             string                               `json:"esquema"`
	Operacion           string                               `json:"operacion"`
	Material            materialSubsanacionSQL               `json:"material"`
	Referencias         referenciasSubsanacionSQL            `json:"referencias"`
	AmbitoHMAC          string                               `json:"ambito_idempotencia_hmac"`
	HuellaHMAC          string                               `json:"huella_peticion_hmac"`
	RetornoRef          string                               `json:"retorno_ref"`
	ExpedienteAnterior  domain.Expediente                    `json:"expediente_anterior"`
	ExpedienteSiguiente domain.Expediente                    `json:"expediente_siguiente"`
	Actuacion           domain.Actuacion                     `json:"actuacion"`
	Politica            politicaSubsanacionSQL               `json:"politica"`
	Autorizacion        autorizacionConfirmarFiscalizacionV1 `json:"autorizacion"`
	InstanteEfecto      time.Time                            `json:"instante_efecto"`
}

func (t *TransaccionSubsanacionReparosPostgreSQL) ConfirmarSubsanacionReparo(ctx context.Context, o ports.OrdenConfirmarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error) {
	if t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) || ctx == nil || validarOrdenSubsanacionSQL(o, o.RegistradaEn) != nil {
		return ports.ReciboSubsanacionReparo{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	material, err := t.proveedor.ProveerMaterialConfirmacionSubsanacionReparo(ctx, o)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, errorDependenciaFiscalizacion(ctx)
	}
	entradas, err := prepararEntradasSubsanacionSQL(o, material)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	defer entradas.borrar()
	for intento := 0; intento < maximoIntentosConfirmarFiscalizacion; intento++ {
		recibo, causa := t.confirmarEnTransaccionSubsanacion(ctx, o, entradas)
		if causa == nil {
			return recibo, nil
		}
		if ctx.Err() != nil || !errorPostgreSQLReintentable(causa) || intento+1 == maximoIntentosConfirmarFiscalizacion {
			return ports.ReciboSubsanacionReparo{}, normalizarErrorSubsanacionSQL(ctx, causa)
		}
	}
	return ports.ReciboSubsanacionReparo{}, ports.ErrPersistenciaFiscalizacionNoDisponible
}

func prepararEntradasSubsanacionSQL(o ports.OrdenConfirmarSubsanacionReparo, m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (entradasConfirmarFiscalizacion, error) {
	if m.ValidarEstructura() != nil {
		return entradasConfirmarFiscalizacion{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	s, errS := o.Evidencia.SolicitudV3.Datos()
	c, errC := o.Evidencia.ConfirmacionV3.Datos()
	v, errV := s.VinculoAutenticacionActor.Datos()
	h, errH := s.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := m.ResumenCapacidad()
	if errS != nil || errC != nil || errV != nil || errH != nil || resumen.ValidarEstructura() != nil || resumen.DecisionRef() != c.DecisionRef || resumen.DecisionHuellaSHA256() != c.DecisionHuellaSHA256 || resumen.ContextoRef() != v.RegistroContextoRef || resumen.ContextoHuellaSHA256() != v.ContextoActorHuellaSHA256 || resumen.Operacion() != s.Accion || resumen.EfectoRef() != o.Material.ExpedienteRef || resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != audienciaConfirmarFiscalizacionV1 || !capacidadBreveContenidaEnConcesion(resumen, c) {
		return entradasConfirmarFiscalizacion{}, ports.ErrAutorizacionDenegada
	}
	// Solo se exportan representaciones canónicas permitidas; las capacidades Go
	// permanecen fuera del DTO JSON y del transporte genérico.
	a, err := nuevaAutorizacionConfirmarFiscalizacion(ports.OrdenConfirmarFiscalizacion{Evidencia: ports.EvidenciaAutorizacionFiscalizacion{Contexto: o.Evidencia.Contexto, SolicitudV3: o.Evidencia.SolicitudV3, DecisionV3: o.Evidencia.DecisionV3, ConfirmacionV3: o.Evidencia.ConfirmacionV3}})
	if err != nil {
		return entradasConfirmarFiscalizacion{}, err
	}
	decision := m.DecisionCanonica()
	motivo := m.MotivoCanonico()
	defer borrarBytes(decision)
	defer borrarBytes(motivo)
	if hex.EncodeToString(decision) != a.DecisionCanonicaHex || hex.EncodeToString(motivo) != a.MotivoCanonicoHex || m.PersonaVersion() != a.PersonaVersion || m.PerfilVersion() != a.PerfilVersion {
		return entradasConfirmarFiscalizacion{}, ports.ErrAutorizacionDenegada
	}
	p := o.Politica
	op := operacionConfirmarSubsanacionSQL{Esquema: esquemaConfirmarSubsanacionReparos, Operacion: ports.OperacionRegistrarSubsanacionReparo, Material: materialSubsanacionParaSQL(o.Material), Referencias: referenciasSubsanacionSQL{o.Preparacion.ReservaRef, o.ReciboRef, o.EventoRef}, AmbitoHMAC: o.Preparacion.AmbitoIdempotenciaHMAC, HuellaHMAC: o.Preparacion.HuellaPeticionHMAC, RetornoRef: o.Preparacion.RetornoRef, ExpedienteAnterior: o.Preparacion.Expediente, ExpedienteSiguiente: o.Expediente, Actuacion: o.Expediente.Actuaciones[len(o.Expediente.Actuaciones)-1], Politica: politicaSubsanacionSQL{p.DefinicionRef, p.DefinicionVersion, p.DefinicionHuellaSHA256, p.Accion, p.Finalidad, p.EvaluadaEn, p.ValidaHasta}, Autorizacion: a, InstanteEfecto: o.RegistradaEn}
	contenido, err := json.Marshal(op)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCargaConfirmarFiscalizacion {
		borrarBytes(contenido)
		return entradasConfirmarFiscalizacion{}, ports.ErrPreparacionFiscalizacionInvalida
	}
	return entradasConfirmarFiscalizacion{contenido: contenido, capacidad: m.CapacidadCanonica(), decision: m.DecisionCanonica(), motivo: m.MotivoCanonico(), contextoActor: m.ContextoActorCanonico(), personaVersion: int64(m.PersonaVersion()), perfilVersion: int64(m.PerfilVersion()), payloadVECAD3: m.PayloadVECAD3(), sobreCOSESign1: m.SobreCOSESign1(), evidencia: m.EvidenciaVerificacion(), raizPublicaSPKI: m.RaizPublicaSPKI()}, nil
}

func (t *TransaccionSubsanacionReparosPostgreSQL) confirmarEnTransaccionSubsanacion(ctx context.Context, o ports.OrdenConfirmarSubsanacionReparo, e entradasConfirmarFiscalizacion) (ports.ReciboSubsanacionReparo, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	defer revertirTransaccion(tx)
	if err = configurarTransaccionSubsanacion(ctx, tx); err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	var ahora time.Time
	if err = tx.QueryRow(ctx, "SELECT date_trunc('microseconds', clock_timestamp())").Scan(&ahora); err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	if err = validarOrdenSubsanacionSQL(o, normalizarInstantePostgreSQL(ahora)); err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, "SELECT "+funcionConfirmarSubsanacionReparos+"($1::jsonb,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", e.contenido, e.capacidad, e.decision, e.motivo, e.contextoActor, e.personaVersion, e.perfilVersion, e.payloadVECAD3, e.sobreCOSESign1, e.evidencia, e.raizPublicaSPKI).Scan(&contenido); err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	defer borrarBytes(contenido)
	r, err := decodificarRespuestaSubsanacionSQL(contenido)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	if r.Resultado != "confirmada" || r.Recibo == nil {
		return ports.ReciboSubsanacionReparo{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	recibo := r.Recibo.puertos()
	if !recibo.ValidarPara(o) {
		return ports.ReciboSubsanacionReparo{}, ports.ErrResultadoFiscalizacionNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	return recibo, nil
}

func configurarTransaccionSubsanacion(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`)
	return err
}
func normalizarErrorSubsanacionSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, domain.ErrVersionEnConflicto) || errors.Is(err, ports.ErrAutorizacionDenegada) {
		return err
	}
	return normalizarErrorConfirmacionFiscalizacion(ctx, err)
}

func parHMACSubsanacionValido(ambito, huella string) bool {
	a, errA := ports.NuevaColeccionSellosHMAC(ambito, nil)
	h, errH := ports.NuevaColeccionSellosHMAC(huella, nil)
	if errA != nil || errH != nil || a.ValidarDominio(ports.DominioAmbitoIdempotenciaSubsanacionReparo) != nil || h.ValidarDominio(ports.DominioHuellaPeticionSubsanacionReparo) != nil {
		return false
	}
	_, err := nuevosSellosPrepararAltaV2(a, h)
	return err == nil
}

func validarOrdenSubsanacionSQL(o ports.OrdenConfirmarSubsanacionReparo, instante time.Time) error {
	p := o.Preparacion
	if !o.Validar() || !domain.InstanteUTCCanonico(instante) || instante.Before(o.RegistradaEn) || p.Confirmada || p.ReciboConfirmado != nil || !preparacionSubsanacionSQLValida(p) || p.Material != o.Material || p.ReciboRef != o.ReciboRef || p.EventoRef != o.EventoRef || o.Material.VersionEsperada != o.VersionAnterior || o.Material.ClaveIdempotencia != o.ClaveIdempotencia || !parHMACSubsanacionValido(p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC) {
		return ports.ErrPreparacionFiscalizacionInvalida
	}
	ps := ports.SolicitudResolverPoliticaSubsanacionReparo{OrganizacionRef: o.OrganizacionRef, ExpedienteRef: o.Expediente.Referencia, ActorRef: o.ActorRef, PerfilRef: o.Material.PerfilRef, RetornoRef: p.RetornoRef, VersionEsperada: o.VersionAnterior, Instante: o.Politica.EvaluadaEn}
	if !o.Politica.ValidaPara(ps, instante) {
		return ports.ErrAutorizacionDenegada
	}
	ultima := o.Expediente.Actuaciones[len(o.Expediente.Actuaciones)-1]
	esperado, err := p.Expediente.RegistrarSubsanacionReparo(o.VersionAnterior, domain.DatosSubsanacionReparo{RetornoRef: p.RetornoRef, Observaciones: o.Material.Observaciones}, domain.DatosActuacion{AccionClave: ultima.AccionClave, ActorRef: ultima.ActorRef, UnidadRef: ultima.UnidadRef, ReciboRef: ultima.ReciboRef, RealizadaEn: ultima.RealizadaEn, FaseDestino: ultima.FaseDestino, EstadoDestino: ultima.EstadoDestino, Observaciones: ultima.Observaciones, RetornoRef: ultima.RetornoRef})
	if err != nil || !reflect.DeepEqual(esperado, o.Expediente) {
		return ports.ErrPreparacionFiscalizacionInvalida
	}
	return validarAutorizacionSubsanacionSQL(o, instante)
}

func validarAutorizacionSubsanacionSQL(o ports.OrdenConfirmarSubsanacionReparo, instante time.Time) error {
	s, es := o.Evidencia.SolicitudV3.Datos()
	v, ev := s.VinculoAutenticacionActor.Datos()
	vc, ec := o.Evidencia.Contexto.Vinculo.Datos()
	concedida, _, ed := o.Evidencia.DecisionV3.Resultado()
	h, eh := vd.HuellaSHA256DecisionAutorizacionV3(o.Evidencia.DecisionV3)
	c, ef := o.Evidencia.ConfirmacionV3.Datos()
	emitida, validaHasta, errVentana := o.Evidencia.DecisionV3.VentanaValidez()
	if es != nil || ev != nil || ec != nil || ed != nil || eh != nil || ef != nil || errVentana != nil || !c.EmitidaEn.Equal(emitida) || !c.ValidaHasta.Equal(validaHasta) || !concedida || o.Evidencia.Contexto.Resultado.Validar() != nil || o.Evidencia.Contexto.Vinculo.ValidarPara(o.Evidencia.Contexto.Resultado) != nil || !o.Evidencia.Contexto.Vinculo.VigenteEn(instante, o.Evidencia.Contexto.Resultado) || o.Evidencia.DecisionV3.ValidarPara(o.Evidencia.SolicitudV3) != nil || !reflect.DeepEqual(v, vc) || v.PrincipalID != o.ActorRef || v.PerfilActivoRef != o.Material.PerfilRef || s.ReferenciaMotivo != o.Politica.MotivoAutorizacion || s.Accion != string(domain.AccionRegistrarSubsanacionReparo) || s.Finalidad != ports.FinalidadRegistrarSubsanacionReparo || s.Recurso.Referencia != o.Material.ExpedienteRef || s.Recurso.ModuloID != ports.ModuloContratacion || s.Recurso.Tipo != ports.TipoRecursoSubsanacionReparo || c.DecisionHuellaSHA256 != h || !o.Evidencia.ConfirmacionV3.DentroDeVentanaEn(instante) {
		return ports.ErrAutorizacionDenegada
	}
	p := o.Preparacion
	huella := sha256.Sum256([]byte(o.Material.Observaciones))
	ambitos := map[string]string{"organizacion_ref": o.OrganizacionRef, "expediente_ref": o.Material.ExpedienteRef, "fase_previa": string(p.Expediente.FaseActual), "estado_previo": string(p.Expediente.EstadoActual)}
	atributos := map[string]string{"version_expediente": strconv.FormatUint(o.VersionAnterior, 10), "retorno_ref": p.RetornoRef, "observaciones_huella_sha256": hex.EncodeToString(huella[:]), "unidad_asignada_ref": p.Expediente.Asignacion.UnidadRef, "responsable_asignado_ref": p.Expediente.Asignacion.ResponsableRef, "politica_ref": o.Politica.DefinicionRef, "politica_version": strconv.FormatUint(o.Politica.DefinicionVersion, 10), "politica_huella_sha256": o.Politica.DefinicionHuellaSHA256, "ambito_idempotencia_hmac": p.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": p.HuellaPeticionHMAC}
	if !reflect.DeepEqual(s.Recurso.Ambitos, ambitos) || !reflect.DeepEqual(s.Recurso.Atributos, atributos) {
		return ports.ErrAutorizacionDenegada
	}
	// Comprobar la representación canónica confirma que no se transporta una
	// decisión diferente de la que liga la confirmación.
	canonica, err := vd.RepresentacionCanonicaDecisionAutorizacionV3(o.Evidencia.DecisionV3)
	if err != nil || bytes.Equal(canonica, nil) {
		return ports.ErrAutorizacionDenegada
	}
	defer borrarBytes(canonica)
	return nil
}

var _ ports.PreparadorSubsanacionReparo = (*PreparadorSubsanacionReparosPostgreSQL)(nil)
var _ ports.ConfirmadorSubsanacionReparo = (*TransaccionSubsanacionReparosPostgreSQL)(nil)
