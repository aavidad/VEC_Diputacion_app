package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Transacciones CT115 (cese y cierre) y CT116 (modificación). La preparación
// es de solo lectura; la confirmación consume la autorización atestada y
// escribe versión, actuación y outbox en la misma transacción.
const (
	maximoCargaSeguimiento    = 3 * 1024 * 1024
	maximoIntentosSeguimiento = 3
)

type descriptorOperacionSeguimiento struct {
	preparar, confirmar, esquemaPreparar, esquemaConfirmar, esquemaResultado string
}

var operacionesSeguimientoSQL = map[string]descriptorOperacionSeguimiento{
	ports.OperacionRegistrarCese: {"vec_contratacion_temporal.preparar_cese_nombramiento_v1", "vec_contratacion_temporal.confirmar_cese_nombramiento_v1",
		"vec.contratacion-temporal.preparar-cese.v1", "vec.contratacion-temporal.confirmar-cese.v1", "vec.contratacion-temporal.resultado-cese.v1"},
	ports.OperacionCerrarExpediente: {"vec_contratacion_temporal.preparar_cierre_expediente_v1", "vec_contratacion_temporal.confirmar_cierre_expediente_v1",
		"vec.contratacion-temporal.preparar-cierre-expediente.v1", "vec.contratacion-temporal.confirmar-cierre-expediente.v1",
		"vec.contratacion-temporal.resultado-cierre-expediente.v1"},
	ports.OperacionModificarTrasNombramiento: {"vec_contratacion_temporal.preparar_modificacion_nombramiento_v1",
		"vec_contratacion_temporal.confirmar_modificacion_nombramiento_v1", "vec.contratacion-temporal.preparar-modificacion-nombramiento.v1",
		"vec.contratacion-temporal.confirmar-modificacion-nombramiento.v1", "vec.contratacion-temporal.resultado-modificacion-nombramiento.v1"},
	ports.OperacionCancelarExpediente: {"vec_contratacion_temporal.preparar_cancelacion_expediente_v1",
		"vec_contratacion_temporal.confirmar_cancelacion_expediente_v1", "vec.contratacion-temporal.preparar-cancelacion.v1",
		"vec.contratacion-temporal.confirmar-cancelacion.v1", "vec.contratacion-temporal.resultado-cancelacion.v1"},
}

type RepositorioOperacionSeguimientoPostgreSQL struct {
	pool iniciadorTransacciones
}

var (
	_ ports.RepositorioOperacionSeguimiento = (*RepositorioOperacionSeguimientoPostgreSQL)(nil)
	_ ports.LectorEstadoSeguimiento         = (*RepositorioOperacionSeguimientoPostgreSQL)(nil)
)

func NuevoRepositorioOperacionSeguimientoPostgreSQL(pool *pgxpool.Pool) (*RepositorioOperacionSeguimientoPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &RepositorioOperacionSeguimientoPostgreSQL{pool: pool}, nil
}

// materialSeguimientoSQL traduce la intención al contrato JSON de CT115/116.
func materialSeguimientoSQL(operacion string, material any) (map[string]any, string, string, error) {
	base := func(org, exp, actor, perfil string, version uint64) map[string]any {
		return map[string]any{"organizacion_ref": org, "expediente_ref": exp, "actor_ref": actor, "perfil_ref": perfil, "version_esperada": version}
	}
	switch m := material.(type) {
	case ports.MaterialCese:
		if operacion != ports.OperacionRegistrarCese || !m.Valido() {
			break
		}
		r := base(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada)
		r["causa_clave"], r["fecha_efecto"] = string(m.Datos.CausaClave), m.Datos.FechaEfecto.Format(time.DateOnly)
		r["justificante_tipo"], r["justificante_ref"] = string(m.Datos.JustificanteTipo), m.Datos.JustificanteRef
		r["justificante_sha256"], r["observaciones"] = m.Datos.JustificanteSHA256, m.Datos.Observaciones
		return r, m.OrganizacionRef, m.ExpedienteRef, nil
	case ports.MaterialCierreExpediente:
		if operacion != ports.OperacionCerrarExpediente || !m.Valido() {
			break
		}
		r := base(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada)
		fecha := ""
		if m.Datos.GINPIXConfirmadaEn != nil {
			fecha = m.Datos.GINPIXConfirmadaEn.Format(time.DateOnly)
		}
		r["condiciones"], r["ginpix_numero"], r["ginpix_confirmada_en"] = append([]string(nil), m.Datos.Condiciones...), m.Datos.GINPIXNumero, fecha
		r["observaciones"] = m.Datos.Observaciones
		return r, m.OrganizacionRef, m.ExpedienteRef, nil
	case ports.MaterialModificacionNombramiento:
		if operacion != ports.OperacionModificarTrasNombramiento || !m.Valido() {
			break
		}
		d := m.Datos
		r := base(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada)
		r["motivo_clave"], r["periodo_inicio"], r["periodo_fin"] = string(d.MotivoClave), d.Periodo.Inicio.Format(time.DateOnly), d.Periodo.Fin.Format(time.DateOnly)
		r["porcentaje_jornada"], r["coste_centimos"], r["fuente_coste_ref"] = uint16(d.Jornada), d.Coste.Centimos, d.FuenteCoste
		r["fase_retorno"], r["estado_retorno"], r["observaciones"] = string(d.FaseRetorno), string(domain.EstadoEnCurso), d.Observaciones
		return r, m.OrganizacionRef, m.ExpedienteRef, nil
	case ports.MaterialCancelacion:
		if operacion != ports.OperacionCancelarExpediente || !m.Valido() {
			break
		}
		return materialCancelacionSQL(m), m.OrganizacionRef, m.ExpedienteRef, nil
	}
	return nil, "", "", ports.ErrOperacionSeguimientoInvalida
}

type reciboSeguimientoSQL struct {
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
	CausaClave        domain.ClaveCatalogo   `json:"causa_clave,omitempty"`
	FechaEfecto       string                 `json:"fecha_efecto,omitempty"`
	CeseReciboRef     string                 `json:"cese_recibo_ref,omitempty"`
	CosteCentimos     int64                  `json:"coste_centimos,omitempty"`
	MotivoClave       domain.ClaveCatalogo   `json:"motivo_clave,omitempty"`
}

func (r reciboSeguimientoSQL) puertos() ports.ReciboOperacionSeguimiento {
	return ports.ReciboOperacionSeguimiento{Operacion: r.Operacion, OrganizacionRef: r.OrganizacionRef, ExpedienteRef: r.ExpedienteRef,
		VersionAnterior: r.VersionAnterior, VersionResultante: r.VersionResultante, FaseResultante: r.FaseResultante,
		EstadoResultante: r.EstadoResultante, ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef,
		ActorRef: r.ActorRef, RegistradaEn: r.RegistradaEn.UTC(), CausaClave: r.CausaClave, FechaEfecto: r.FechaEfecto,
		CeseReciboRef: r.CeseReciboRef, CosteCentimos: r.CosteCentimos, MotivoClave: r.MotivoClave}
}

type respuestaSeguimientoSQL struct {
	Esquema       string                     `json:"esquema"`
	Resultado     string                     `json:"resultado"`
	Material      map[string]any             `json:"material,omitempty"`
	Expediente    *domain.Expediente         `json:"expediente,omitempty"`
	Referencias   *referenciasSubsanacionSQL `json:"referencias,omitempty"`
	Incorporacion *struct {
		ReciboRef string `json:"recibo_ref"`
		Inicio    string `json:"inicio"`
	} `json:"incorporacion,omitempty"`
	CeseReciboRef string                `json:"cese_recibo_ref,omitempty"`
	AmbitoHMAC    string                `json:"ambito_idempotencia_hmac,omitempty"`
	HuellaHMAC    string                `json:"huella_peticion_hmac,omitempty"`
	Recibo        *reciboSeguimientoSQL `json:"recibo,omitempty"`
}

func errorResultadoSeguimiento(resultado string) error {
	switch resultado {
	case "preparada", "confirmada":
		return nil
	case "idempotencia_reutilizada":
		return ports.ErrClaveIdempotenciaUsada
	case "version_en_conflicto":
		return domain.ErrVersionEnConflicto
	case "sin_incorporacion":
		return ports.ErrCeseSinIncorporacion
	case "fecha_anterior_incorporacion":
		return ports.ErrCeseFechaAnteriorIncorporacion
	case "cese_existente":
		return ports.ErrCeseYaRegistrado
	case "sin_cese":
		return ports.ErrCierreSinCese
	case "cierre_existente":
		return ports.ErrCierreYaRegistrado
	case "sin_cambios":
		return ports.ErrModificacionSinCambios
	case "credito_insuficiente":
		return ports.ErrModificacionCreditoInsuficiente
	case "fase_no_admitida":
		return ports.ErrCancelacionNoAdmitida
	case "tras_fiscalizacion":
		return ports.ErrCancelacionTrasFiscalizacion
	case "cancelacion_existente":
		return ports.ErrCancelacionYaRegistrada
	}
	return ports.ErrResultadoSeguimientoNoConfiable
}

func decodificarRespuestaSeguimiento(contenido []byte, esquema string) (respuestaSeguimientoSQL, error) {
	var r respuestaSeguimientoSQL
	if len(contenido) == 0 || len(contenido) > maximoCargaSeguimiento || decodificarJSONEstricto(contenido, &r) != nil || r.Esquema != esquema {
		return r, ports.ErrResultadoSeguimientoNoConfiable
	}
	return r, errorResultadoSeguimiento(r.Resultado)
}

func configurarTransaccionSeguimiento(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`)
	return err
}

func normalizarErrorSeguimiento(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, propio := range []error{domain.ErrVersionEnConflicto, ports.ErrClaveIdempotenciaUsada, ports.ErrCeseSinIncorporacion,
		ports.ErrCeseFechaAnteriorIncorporacion, ports.ErrCeseYaRegistrado, ports.ErrCierreSinCese, ports.ErrCierreYaRegistrado,
		ports.ErrModificacionSinCambios, ports.ErrModificacionCreditoInsuficiente, ports.ErrCancelacionNoAdmitida,
		ports.ErrCancelacionTrasFiscalizacion, ports.ErrCancelacionYaRegistrada, ports.ErrResultadoSeguimientoNoConfiable,
		ports.ErrOperacionSeguimientoInvalida, ports.ErrAutorizacionDenegada} {
		if errors.Is(err, propio) {
			return propio
		}
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

// ejecutar repite la transacción ante conflictos de serialización.
func (r *RepositorioOperacionSeguimientoPostgreSQL) ejecutar(ctx context.Context, lectura bool, f func(pgx.Tx) error) error {
	if ctx == nil || r == nil || dependenciaNula(r.pool) {
		return ports.ErrOperacionSeguimientoNoDisponible
	}
	acceso := pgx.ReadWrite
	if lectura {
		acceso = pgx.ReadOnly
	}
	var err error
	for intento := 0; intento < maximoIntentosSeguimiento; intento++ {
		err = func() error {
			tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: acceso})
			if err != nil {
				return err
			}
			defer revertirTransaccion(tx)
			if err := configurarTransaccionSeguimiento(ctx, tx); err != nil {
				return err
			}
			if err := f(tx); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}()
		if err == nil || ctx.Err() != nil || !errorPostgreSQLReintentable(err) {
			break
		}
	}
	if err != nil {
		return normalizarErrorSeguimiento(ctx, err)
	}
	return nil
}

func (r *RepositorioOperacionSeguimientoPostgreSQL) PrepararOperacionSeguimiento(ctx context.Context, operacion string, material any, sellos ports.SellosOperacionSeguimiento, refs ports.ReferenciasEfectoSeguimiento) (ports.PreparacionOperacionSeguimiento, error) {
	var vacia ports.PreparacionOperacionSeguimiento
	d, ok := operacionesSeguimientoSQL[operacion]
	dominioAmbito, dominioHuella, _ := ports.DominiosHMACOperacionSeguimiento(operacion)
	if !ok || !refs.Validas() || sellos.Ambitos.ValidarDominio(dominioAmbito) != nil || sellos.Huellas.ValidarDominio(dominioHuella) != nil {
		return vacia, ports.ErrOperacionSeguimientoInvalida
	}
	m, org, exp, err := materialSeguimientoSQL(operacion, material)
	if err != nil {
		return vacia, err
	}
	pares, err := nuevosSellosPrepararAltaV2(sellos.Ambitos, sellos.Huellas)
	if err != nil {
		return vacia, ports.ErrOperacionSeguimientoInvalida
	}
	cuerpo, err := json.Marshal(map[string]any{"esquema": d.esquemaPreparar, "operacion": operacion, "material": m, "sellos_hmac": pares,
		"referencias_candidatas": referenciasSubsanacionSQL{refs.ReservaRef, refs.ReciboRef, refs.EventoRef}})
	if err != nil {
		return vacia, ports.ErrOperacionSeguimientoInvalida
	}
	defer borrarBytes(cuerpo)
	var p ports.PreparacionOperacionSeguimiento
	err = r.ejecutar(ctx, true, func(tx pgx.Tx) error {
		var contenido []byte
		if err := tx.QueryRow(ctx, "SELECT "+d.preparar+"($1::jsonb)::text", cuerpo).Scan(&contenido); err != nil {
			return err
		}
		defer borrarBytes(contenido)
		resp, err := decodificarRespuestaSeguimiento(contenido, d.esquemaResultado)
		if err != nil {
			return err
		}
		if resp.Expediente == nil || resp.Referencias == nil || !pares.contienePar(resp.AmbitoHMAC, resp.HuellaHMAC) ||
			resp.Expediente.Validar() != nil || resp.Expediente.Referencia != exp || resp.Expediente.OrganizacionRef != org {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		p = ports.PreparacionOperacionSeguimiento{Expediente: resp.Expediente.Clonar(), AmbitoIdempotenciaHMAC: resp.AmbitoHMAC, HuellaPeticionHMAC: resp.HuellaHMAC,
			Referencias:   ports.ReferenciasEfectoSeguimiento{ReservaRef: resp.Referencias.ReservaRef, ReciboRef: resp.Referencias.ReciboRef, EventoRef: resp.Referencias.EventoRef},
			CeseReciboRef: resp.CeseReciboRef, Confirmada: resp.Resultado == "confirmada"}
		if resp.Incorporacion != nil {
			inicio, err := time.Parse(time.DateOnly, resp.Incorporacion.Inicio)
			if err != nil {
				return ports.ErrResultadoSeguimientoNoConfiable
			}
			p.IncorporacionRef, p.InicioIncorporacion = resp.Incorporacion.ReciboRef, inicio.UTC()
		}
		if fuente, ok := resp.Material["fuente_coste_ref"].(string); ok {
			p.FuenteCosteRef = fuente
		}
		if p.Confirmada {
			if resp.Recibo == nil {
				return ports.ErrResultadoSeguimientoNoConfiable
			}
			recibo := resp.Recibo.puertos()
			p.Recibo = &recibo
		} else if resp.Recibo != nil {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacia, err
	}
	return p, nil
}

// autorizacionSeguimientoSQL expone solo representaciones canónicas ya
// ligadas; SQL coteja cada campo con la decisión y la capacidad.
func autorizacionSeguimientoSQL(o ports.OrdenConfirmarOperacionSeguimiento, exp string) (autorizacionConfirmarFiscalizacionV1, error) {
	var vacia autorizacionConfirmarFiscalizacionV1
	a := o.Autorizacion
	if a.ValidarEstructura() != nil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	recurso := vd.RecursoAutorizable{Referencia: exp, ModuloID: ports.ModuloContratacion, Tipo: tipoRecursoOperacionSeguimiento(o.Operacion),
		Ambitos: o.Contexto.Ambitos, Atributos: o.Contexto.Atributos}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || c.Operacion() != string(o.Accion) || c.EfectoRef() != exp || c.EfectoHuellaSHA256() != huella || c.AudienciaConsumo() != o.Audiencia {
		return vacia, ports.ErrAutorizacionDenegada
	}
	decision := a.DecisionCanonica()
	motivo := a.MotivoCanonico()
	defer borrarBytes(decision)
	defer borrarBytes(motivo)
	var dec struct {
		DecisionRef     string `json:"decision_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		RecursoRef      string `json:"recurso_ref"`
		Contexto        string `json:"contexto_recurso_huella_sha256"`
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
	}
	if json.Unmarshal(decision, &dec) != nil || dec.DecisionRef != c.DecisionRef() || dec.RecursoRef != exp || dec.Contexto != huella ||
		dec.Accion != string(o.Accion) || dec.Finalidad != o.Finalidad {
		return vacia, ports.ErrAutorizacionDenegada
	}
	h := sha256.Sum256(decision)
	if hex.EncodeToString(h[:]) != c.DecisionHuellaSHA256() {
		return vacia, ports.ErrAutorizacionDenegada
	}
	return autorizacionConfirmarFiscalizacionV1{DecisionCanonicaHex: hex.EncodeToString(decision), MotivoCanonicoHex: hex.EncodeToString(motivo),
		PersonaVersion: a.PersonaVersion(), PerfilVersion: a.PerfilVersion(), DecisionRef: dec.DecisionRef, DecisionHuellaSHA256: c.DecisionHuellaSHA256(),
		PrincipalID: dec.PrincipalID, PerfilActivoRef: dec.PerfilActivoRef, Accion: dec.Accion, RecursoRef: exp, ContextoRecursoHuellaSHA256: huella,
		Finalidad: dec.Finalidad}, nil
}

func tipoRecursoOperacionSeguimiento(operacion string) string {
	switch operacion {
	case ports.OperacionRegistrarCese:
		return ports.TipoRecursoCese
	case ports.OperacionCerrarExpediente:
		return ports.TipoRecursoCierreExpediente
	case ports.OperacionModificarTrasNombramiento:
		return ports.TipoRecursoModificacionNombramiento
	case ports.OperacionCancelarExpediente:
		return ports.TipoRecursoCancelacion
	}
	return ""
}

func (r *RepositorioOperacionSeguimientoPostgreSQL) ConfirmarOperacionSeguimiento(ctx context.Context, o ports.OrdenConfirmarOperacionSeguimiento) (ports.ReciboOperacionSeguimiento, error) {
	var vacio ports.ReciboOperacionSeguimiento
	d, ok := operacionesSeguimientoSQL[o.Operacion]
	if !ok || o.Preparacion.Confirmada || o.Siguiente.Validar() != nil || len(o.Siguiente.Actuaciones) == 0 || !domain.InstanteUTCCanonico(o.InstanteEfecto) {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	m, org, exp, err := materialSeguimientoSQL(o.Operacion, o.Material)
	if err != nil {
		return vacio, err
	}
	a, err := autorizacionSeguimientoSQL(o, exp)
	if err != nil {
		return vacio, err
	}
	if !o.Politica.ValidaEn(o.InstanteEfecto) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	p := o.Politica
	cuerpo, err := json.Marshal(map[string]any{
		"esquema": d.esquemaConfirmar, "operacion": o.Operacion, "material": m,
		"referencias":              referenciasSubsanacionSQL{o.Preparacion.Referencias.ReservaRef, o.Preparacion.Referencias.ReciboRef, o.Preparacion.Referencias.EventoRef},
		"ambito_idempotencia_hmac": o.Preparacion.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": o.Preparacion.HuellaPeticionHMAC,
		"expediente_anterior": o.Preparacion.Expediente, "expediente_siguiente": o.Siguiente,
		"actuacion": o.Siguiente.Actuaciones[len(o.Siguiente.Actuaciones)-1],
		"politica": map[string]any{"definicion_ref": p.DefinicionRef, "definicion_version": p.DefinicionVersion, "definicion_huella_sha256": p.DefinicionHuellaSHA256,
			"accion": string(o.Accion), "finalidad": o.Finalidad, "evaluada_en": p.EvaluadaEn, "valida_hasta": p.ValidaHasta},
		"autorizacion": a, "instante_efecto": o.InstanteEfecto,
		"contexto": map[string]any{"ambitos": o.Contexto.Ambitos, "atributos": o.Contexto.Atributos},
	})
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
	var recibo ports.ReciboOperacionSeguimiento
	err = r.ejecutar(ctx, false, func(tx pgx.Tx) error {
		var contenido []byte
		if err := tx.QueryRow(ctx, "SELECT "+d.confirmar+"($1::jsonb,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", cuerpo,
			secretos[0], secretos[1], secretos[2], secretos[3], int64(x.PersonaVersion()), int64(x.PerfilVersion()),
			secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&contenido); err != nil {
			return err
		}
		defer borrarBytes(contenido)
		resp, err := decodificarRespuestaSeguimiento(contenido, d.esquemaResultado)
		if err != nil {
			return err
		}
		if resp.Resultado != "confirmada" || resp.Recibo == nil {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		recibo = resp.Recibo.puertos()
		if !recibo.ValidoPara(o.Operacion, org, exp, o.Preparacion.Expediente.Version) ||
			recibo.ReciboRef != o.Preparacion.Referencias.ReciboRef || recibo.EventoRef != o.Preparacion.Referencias.EventoRef {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	return recibo, nil
}

// ConsultarEstadoSeguimiento lee cese y cierre registrados. La composición
// solo la invoca tras acreditar la lectura del detalle del expediente.
func (r *RepositorioOperacionSeguimientoPostgreSQL) ConsultarEstadoSeguimiento(ctx context.Context, org, exp string) (ports.EstadoSeguimientoExpediente, error) {
	var vacio ports.EstadoSeguimientoExpediente
	if !domain.ReferenciaOpacaValida(org) || !domain.ReferenciaOpacaValida(exp) {
		return vacio, ports.ErrOperacionSeguimientoInvalida
	}
	type reciboFecha struct {
		ReciboRef    string    `json:"recibo_ref"`
		RegistradaEn time.Time `json:"registrada_en"`
	}
	var salida struct {
		Esquema       string `json:"esquema"`
		ExpedienteRef string `json:"expediente_ref"`
		Incorporacion *struct {
			ReciboRef string `json:"recibo_ref"`
			Inicio    string `json:"inicio"`
		} `json:"incorporacion"`
		Cese *struct {
			CausaClave         string          `json:"causa_clave"`
			FechaEfecto        string          `json:"fecha_efecto"`
			JustificanteTipo   string          `json:"justificante_tipo"`
			JustificanteRef    string          `json:"justificante_ref"`
			JustificanteSHA256 string          `json:"justificante_sha256"`
			Observaciones      string          `json:"observaciones"`
			Recibo             json.RawMessage `json:"recibo"`
		} `json:"cese"`
		Cierre *struct {
			Condiciones        []string        `json:"condiciones"`
			GINPIXNumero       string          `json:"ginpix_numero"`
			GINPIXConfirmadaEn string          `json:"ginpix_confirmada_en"`
			Observaciones      string          `json:"observaciones"`
			Recibo             json.RawMessage `json:"recibo"`
		} `json:"cierre"`
	}
	err := r.ejecutar(ctx, true, func(tx pgx.Tx) error {
		var contenido []byte
		if err := tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.consultar_cese_cierre_expediente_v1($1,$2)::text", org, exp).Scan(&contenido); err != nil {
			return err
		}
		if len(contenido) > maximoCargaSeguimiento || json.Unmarshal(contenido, &salida) != nil ||
			salida.Esquema != "vec.contratacion-temporal.cese-cierre-expediente.v1" || salida.ExpedienteRef != exp {
			return ports.ErrResultadoSeguimientoNoConfiable
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	estado := ports.EstadoSeguimientoExpediente{ExpedienteRef: exp}
	if i := salida.Incorporacion; i != nil {
		estado.IncorporacionRef, estado.InicioIncorporacion = i.ReciboRef, i.Inicio
	}
	if c := salida.Cese; c != nil {
		var rf reciboFecha
		if json.Unmarshal(c.Recibo, &rf) != nil {
			return vacio, ports.ErrResultadoSeguimientoNoConfiable
		}
		estado.Cese = &ports.EstadoCeseExpediente{CausaClave: c.CausaClave, FechaEfecto: c.FechaEfecto, JustificanteTipo: c.JustificanteTipo,
			JustificanteRef: c.JustificanteRef, JustificanteSHA256: c.JustificanteSHA256, Observaciones: c.Observaciones,
			ReciboRef: rf.ReciboRef, RegistradaEn: rf.RegistradaEn.UTC()}
	}
	if c := salida.Cierre; c != nil {
		var rf reciboFecha
		if json.Unmarshal(c.Recibo, &rf) != nil || !domain.CondicionesCierreValidas(c.Condiciones) {
			return vacio, ports.ErrResultadoSeguimientoNoConfiable
		}
		estado.Cierre = &ports.EstadoCierreExpediente{Condiciones: append([]string(nil), c.Condiciones...), GINPIXNumero: c.GINPIXNumero,
			GINPIXConfirmadaEn: c.GINPIXConfirmadaEn, Observaciones: c.Observaciones, ReciboRef: rf.ReciboRef, RegistradaEn: rf.RegistradaEn.UTC()}
	}
	if (estado.Cese == nil && estado.Cierre != nil) || strings.TrimSpace(estado.ExpedienteRef) == "" {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	return estado, nil
}
