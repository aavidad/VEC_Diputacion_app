package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaPrepararReincorporacion  = "vec.contratacion-temporal.preparar-reincorporacion-titular.v1"
	esquemaConfirmarReincorporacion = "vec.contratacion-temporal.confirmar-reincorporacion-titular.v1"
	esquemaResultadoReincorporacion = "vec.contratacion-temporal.resultado-reincorporacion-titular.v1"
)

type RepositorioReincorporacionTitularPostgreSQL struct {
	pool iniciadorLecturaReincorporacionTitular
}

var _ ports.RepositorioReincorporacionTitular = (*RepositorioReincorporacionTitularPostgreSQL)(nil)

func NuevoRepositorioReincorporacionTitularPostgreSQL(pool *pgxpool.Pool) (*RepositorioReincorporacionTitularPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &RepositorioReincorporacionTitularPostgreSQL{pool: pool}, nil
}

type respuestaReincorporacionSQL struct {
	Esquema       string                     `json:"esquema"`
	Resultado     string                     `json:"resultado"`
	Material      map[string]any             `json:"material"`
	Expediente    *domain.Expediente         `json:"expediente"`
	Referencias   *referenciasSubsanacionSQL `json:"referencias"`
	RelacionRef   string                     `json:"relacion_ref"`
	CeseEventoRef string                     `json:"cese_evento_ref"`
	CeseReciboRef string                     `json:"cese_recibo_ref"`
	AmbitoHMAC    string                     `json:"ambito_idempotencia_hmac"`
	HuellaHMAC    string                     `json:"huella_peticion_hmac"`
	Recibo        *struct {
		Operacion         string                 `json:"operacion"`
		OrganizacionRef   string                 `json:"organizacion_ref"`
		ExpedienteRef     string                 `json:"expediente_ref"`
		RelacionRef       string                 `json:"relacion_ref"`
		FechaEfectiva     string                 `json:"fecha_efectiva"`
		CeseEventoRef     string                 `json:"cese_evento_ref"`
		CeseReciboRef     string                 `json:"cese_recibo_ref"`
		VersionAnterior   uint64                 `json:"version_anterior"`
		VersionResultante uint64                 `json:"version_resultante"`
		FaseResultante    domain.ClaveFase       `json:"fase_resultante"`
		EstadoResultante  domain.EstadoOperativo `json:"estado_resultante"`
		DocumentoRef      string                 `json:"documento_ref"`
		DocumentoSHA256   string                 `json:"documento_sha256"`
		ReciboRef         string                 `json:"recibo_ref"`
		AuditoriaRef      string                 `json:"auditoria_ref"`
		EventoRef         string                 `json:"evento_ref"`
		ActorRef          string                 `json:"actor_ref"`
		RegistradaEn      time.Time              `json:"registrada_en"`
	} `json:"recibo"`
}

func (r respuestaReincorporacionSQL) recibo() ports.ReciboReincorporacionTitular {
	if r.Recibo == nil {
		return ports.ReciboReincorporacionTitular{}
	}
	x := r.Recibo
	return ports.ReciboReincorporacionTitular{Operacion: x.Operacion, OrganizacionRef: x.OrganizacionRef,
		ExpedienteRef: x.ExpedienteRef, RelacionRef: x.RelacionRef, FechaEfectiva: x.FechaEfectiva,
		CeseEventoRef: x.CeseEventoRef, CeseReciboRef: x.CeseReciboRef, VersionAnterior: x.VersionAnterior,
		VersionResultante: x.VersionResultante, ReciboRef: x.ReciboRef, AuditoriaRef: x.AuditoriaRef,
		EventoRef: x.EventoRef, ActorRef: x.ActorRef, RegistradaEn: x.RegistradaEn.UTC()}
}

func decodificarReincorporacionSQL(b []byte) (respuestaReincorporacionSQL, error) {
	var r respuestaReincorporacionSQL
	if len(b) == 0 || len(b) > maximoCargaSeguimiento || decodificarConExpedienteSQL(b, &r, "expediente") != nil || r.Esquema != esquemaResultadoReincorporacion {
		return r, ports.ErrResultadoSeguimientoNoConfiable
	}
	// Los conflictos de CT130 son respuestas mínimas. Un campo adicional
	// convertiría un recibo o una proyección contradictorios en COMMIT.
	conflicto := func(err error) (respuestaReincorporacionSQL, error) {
		var campos map[string]json.RawMessage
		if decodificarJSONEstricto(b, &campos) != nil || len(campos) != 2 {
			return r, ports.ErrResultadoSeguimientoNoConfiable
		}
		return r, err
	}
	switch r.Resultado {
	case "preparada", "confirmada":
		return r, nil
	case "sin_cese":
		return conflicto(ports.ErrReincorporacionSinCese)
	case "cese_no_coincide":
		return conflicto(ports.ErrReincorporacionCeseNoCoincide)
	case "reincorporacion_existente":
		return conflicto(ports.ErrReincorporacionYaRegistrada)
	case "version_en_conflicto":
		return conflicto(domain.ErrVersionEnConflicto)
	case "idempotencia_reutilizada":
		return conflicto(ports.ErrClaveIdempotenciaUsada)
	default:
		return r, ports.ErrResultadoSeguimientoNoConfiable
	}
}

func materialReincorporacionSQL(m ports.MaterialReincorporacionTitular) map[string]any {
	return map[string]any{"organizacion_ref": m.OrganizacionRef, "expediente_ref": m.ExpedienteRef, "relacion_ref": m.RelacionRef,
		"version_esperada": m.VersionEsperada, "actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef,
		"fecha_efectiva": m.FechaEfectiva.Format(time.DateOnly), "documento_ref": m.DocumentoRef, "documento_sha256": m.DocumentoSHA256}
}

func (r *RepositorioReincorporacionTitularPostgreSQL) ejecutar(ctx context.Context, lectura bool, f func(pgx.Tx) error) error {
	if ctx == nil || r == nil || dependenciaNula(r.pool) {
		return ports.ErrOperacionSeguimientoNoDisponible
	}
	modo := pgx.ReadWrite
	if lectura {
		modo = pgx.ReadOnly
	}
	var err error
	for intento := 0; intento < maximoIntentosSeguimiento; intento++ {
		err = func() error {
			tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: modo})
			if e != nil {
				return e
			}
			defer revertirTransaccion(tx)
			if e = configurarTransaccionSeguimiento(ctx, tx); e != nil {
				return e
			}
			if e = f(tx); e != nil {
				return e
			}
			return tx.Commit(ctx)
		}()
		if err == nil || ctx.Err() != nil || !errorPostgreSQLReintentable(err) {
			break
		}
	}
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrAutorizacionDenegada
		case "22023":
			return ports.ErrOperacionSeguimientoInvalida
		}
	}
	return ports.ErrOperacionSeguimientoNoDisponible
}

func (r *RepositorioReincorporacionTitularPostgreSQL) PrepararReincorporacionTitular(ctx context.Context, m ports.MaterialReincorporacionTitular,
	lectura ports.AntecedenteReincorporacionTitular, sellos ports.SellosOperacionSeguimiento, refs ports.ReferenciasEfectoSeguimiento) (ports.PreparacionReincorporacionTitular, error) {
	var vacio ports.PreparacionReincorporacionTitular
	if !m.Valido() || !lectura.ValidoPara(m) || lectura.Resultado != ports.ResultadoAntecedenteCoincide ||
		!refs.Validas() || sellos.Ambitos.ValidarDominio(ports.DominioAmbitoReincorporacionTitular) != nil ||
		sellos.Huellas.ValidarDominio(ports.DominioHuellaReincorporacionTitular) != nil {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	pares, err := nuevosSellosPrepararAltaV2(sellos.Ambitos, sellos.Huellas)
	if err != nil {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	cuerpo, err := json.Marshal(map[string]any{"esquema": esquemaPrepararReincorporacion, "operacion": ports.OperacionRegistrarReincorporacionTitular,
		"material": materialReincorporacionSQL(m), "sellos_hmac": pares,
		"referencias_candidatas": referenciasSubsanacionSQL{refs.ReservaRef, refs.ReciboRef, refs.EventoRef}})
	if err != nil || len(cuerpo) > 65536 {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	defer borrarBytes(cuerpo)
	var preparacion ports.PreparacionReincorporacionTitular
	var resultadoError error
	// CT134 registra el uso único del recibo de lectura en la misma transacción
	// que la preparación; ambas escrituras se deshacen juntas ante cualquier fallo.
	err = r.ejecutar(ctx, false, func(tx pgx.Tx) error {
		resultadoError = nil
		var b []byte
		if e := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.preparar_reincorporacion_titular_acreditada_v1($1::jsonb,$2::text,$3::text)::text",
			cuerpo, lectura.LecturaRef, lectura.AuditoriaRef).Scan(&b); e != nil {
			return e
		}
		defer borrarBytes(b)
		var e error
		respuesta, e := decodificarReincorporacionSQL(b)
		if e != nil {
			if errors.Is(e, ports.ErrResultadoSeguimientoNoConfiable) {
				return e
			}
			resultadoError = e
			return nil
		}
		if respuesta.Referencias == nil || respuesta.RelacionRef != m.RelacionRef ||
			!ports.ColeccionesHMACContienenPar(sellos.Ambitos, ports.DominioAmbitoReincorporacionTitular, sellos.Huellas,
				ports.DominioHuellaReincorporacionTitular, respuesta.AmbitoHMAC, respuesta.HuellaHMAC) {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		p := ports.PreparacionReincorporacionTitular{Referencias: ports.ReferenciasEfectoSeguimiento{ReservaRef: respuesta.Referencias.ReservaRef,
			ReciboRef: respuesta.Referencias.ReciboRef, EventoRef: respuesta.Referencias.EventoRef},
			AmbitoIdempotenciaHMAC: respuesta.AmbitoHMAC, HuellaPeticionHMAC: respuesta.HuellaHMAC,
			CeseEventoRef: respuesta.CeseEventoRef, CeseReciboRef: respuesta.CeseReciboRef, Confirmada: respuesta.Resultado == "confirmada"}
		if !p.Referencias.Validas() || p.CeseEventoRef != lectura.CeseEventoRef || p.CeseReciboRef != lectura.CeseReciboRef {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		if p.Confirmada {
			if respuesta.Recibo == nil {
				return ports.ErrResultadoSeguimientoNoConfiable
			}
			x := respuesta.recibo()
			if !x.ValidoPara(m) || x.CeseEventoRef != p.CeseEventoRef || x.CeseReciboRef != p.CeseReciboRef ||
				x.ReciboRef != p.Referencias.ReciboRef || x.EventoRef != p.Referencias.EventoRef {
				return ports.ErrResultadoSeguimientoNoConfiable
			}
			p.Recibo = &x
		} else {
			if respuesta.Expediente == nil || respuesta.Expediente.Validar() != nil || respuesta.Recibo != nil ||
				respuesta.Expediente.Referencia != m.ExpedienteRef || respuesta.Expediente.OrganizacionRef != m.OrganizacionRef ||
				respuesta.Expediente.Version != m.VersionEsperada || respuesta.Expediente.Asignacion == nil {
				return ports.ErrResultadoSeguimientoNoConfiable
			}
			p.Expediente = respuesta.Expediente.Clonar()
		}
		preparacion = p
		return nil
	})
	if err != nil {
		return vacio, err
	}
	if resultadoError != nil {
		return vacio, resultadoError
	}
	return preparacion, nil
}

func autorizacionReincorporacionSQL(o ports.OrdenConfirmarReincorporacionTitular) (autorizacionConfirmarFiscalizacionV1, error) {
	var vacia autorizacionConfirmarFiscalizacionV1
	a := o.Autorizacion
	if a.ValidarEstructura() != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	m := o.Material
	recurso := vd.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion,
		Tipo: ports.TipoRecursoReincorporacionTitular, Ambitos: o.Contexto.Ambitos, Atributos: o.Contexto.Atributos}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || c.Operacion() != string(domain.AccionRegistrarReincorporacionTitular) || c.EfectoRef() != m.ExpedienteRef ||
		c.EfectoHuellaSHA256() != huella || c.AudienciaConsumo() != ports.AudienciaConsumoReincorporacionTitularV1 {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision, motivo := a.DecisionCanonica(), a.MotivoCanonico()
	defer borrarBytes(decision)
	defer borrarBytes(motivo)
	var cr struct {
		DecisionRef     string `json:"decision_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		RecursoRef      string `json:"recurso_ref"`
		Contexto        string `json:"contexto_recurso_huella_sha256"`
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
	}
	if json.Unmarshal(decision, &cr) != nil || cr.DecisionRef != c.DecisionRef() || cr.RecursoRef != m.ExpedienteRef || cr.Contexto != huella ||
		cr.Accion != string(domain.AccionRegistrarReincorporacionTitular) || cr.Finalidad != ports.FinalidadRegistrarReincorporacionTitular ||
		cr.PrincipalID != m.ActorRef || cr.PerfilActivoRef != m.PerfilRef {
		return vacia, ports.ErrAutorizacionDenegada
	}
	h := sha256.Sum256(decision)
	if hex.EncodeToString(h[:]) != c.DecisionHuellaSHA256() {
		return vacia, ports.ErrAutorizacionDenegada
	}
	return autorizacionConfirmarFiscalizacionV1{DecisionCanonicaHex: hex.EncodeToString(decision), MotivoCanonicoHex: hex.EncodeToString(motivo),
		PersonaVersion: a.PersonaVersion(), PerfilVersion: a.PerfilVersion(), DecisionRef: cr.DecisionRef,
		DecisionHuellaSHA256: c.DecisionHuellaSHA256(), PrincipalID: cr.PrincipalID, PerfilActivoRef: cr.PerfilActivoRef,
		Accion: cr.Accion, RecursoRef: m.ExpedienteRef, ContextoRecursoHuellaSHA256: huella, Finalidad: cr.Finalidad}, nil
}

func (r *RepositorioReincorporacionTitularPostgreSQL) ConfirmarReincorporacionTitular(ctx context.Context, o ports.OrdenConfirmarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error) {
	var vacio ports.ReciboReincorporacionTitular
	m := o.Material
	if !m.Valido() || !o.Lectura.ValidoPara(m) || o.Lectura.Resultado != ports.ResultadoAntecedenteCoincide ||
		o.Preparacion.Confirmada || !o.Preparacion.Referencias.Validas() || len(o.Siguiente.Actuaciones) == 0 ||
		o.Siguiente.Validar() != nil || !domain.InstanteUTCCanonico(o.InstanteEfecto) || !o.Politica.ValidaEn(o.InstanteEfecto) {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	aut, err := autorizacionReincorporacionSQL(o)
	if err != nil {
		return vacio, err
	}
	p := o.Politica
	cuerpo, err := json.Marshal(map[string]any{"esquema": esquemaConfirmarReincorporacion, "operacion": ports.OperacionRegistrarReincorporacionTitular,
		"material": materialReincorporacionSQL(m), "referencias": referenciasSubsanacionSQL{o.Preparacion.Referencias.ReservaRef, o.Preparacion.Referencias.ReciboRef, o.Preparacion.Referencias.EventoRef},
		"ambito_idempotencia_hmac": o.Preparacion.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": o.Preparacion.HuellaPeticionHMAC,
		"expediente_anterior": o.Preparacion.Expediente, "expediente_siguiente": o.Siguiente,
		"actuacion": o.Siguiente.Actuaciones[len(o.Siguiente.Actuaciones)-1],
		"politica": map[string]any{"definicion_ref": p.DefinicionRef, "definicion_version": p.DefinicionVersion,
			"definicion_huella_sha256": p.DefinicionHuellaSHA256, "accion": string(domain.AccionRegistrarReincorporacionTitular),
			"finalidad": ports.FinalidadRegistrarReincorporacionTitular, "evaluada_en": p.EvaluadaEn, "valida_hasta": p.ValidaHasta},
		"autorizacion": aut, "instante_efecto": o.InstanteEfecto,
		"contexto": map[string]any{"ambitos": o.Contexto.Ambitos, "atributos": o.Contexto.Atributos}})
	if err != nil || len(cuerpo) > maximoCargaSeguimiento {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	defer borrarBytes(cuerpo)
	x := o.Autorizacion
	secretos := [][]byte{x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(),
		x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var recibo ports.ReciboReincorporacionTitular
	var resultadoError error
	err = r.ejecutar(ctx, false, func(tx pgx.Tx) error {
		resultadoError = nil
		var b []byte
		if e := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.confirmar_reincorporacion_titular_acreditada_v1($1::jsonb,$2::text,$3::text,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)::text",
			cuerpo, o.Lectura.LecturaRef, o.Lectura.AuditoriaRef,
			secretos[0], secretos[1], secretos[2], secretos[3], int64(x.PersonaVersion()), int64(x.PerfilVersion()),
			secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&b); e != nil {
			return e
		}
		defer borrarBytes(b)
		var e error
		respuesta, e := decodificarReincorporacionSQL(b)
		if e != nil {
			if errors.Is(e, ports.ErrResultadoSeguimientoNoConfiable) {
				return e
			}
			resultadoError = e
			return nil
		}
		if respuesta.Resultado != "confirmada" || respuesta.Recibo == nil {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		recibo = respuesta.recibo()
		if !recibo.ValidoPara(m) || recibo.ReciboRef != o.Preparacion.Referencias.ReciboRef ||
			recibo.EventoRef != o.Preparacion.Referencias.EventoRef ||
			recibo.CeseEventoRef != o.Preparacion.CeseEventoRef || recibo.CeseReciboRef != o.Preparacion.CeseReciboRef ||
			!recibo.RegistradaEn.Equal(o.InstanteEfecto) {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	if resultadoError != nil {
		return vacio, resultadoError
	}
	return recibo, nil
}
