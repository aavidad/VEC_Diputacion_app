package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistroOperacionContactoPostgreSQL sólo confirma un contacto si la
// intención preparada y su compromiso HMAC siguen vigentes. Contacto3 enlaza
// operación, versión, consumo V3, recibo T13 y outbox en una transacción.
type RegistroOperacionContactoPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoRegistroOperacionContactoPostgreSQL(pool *pgxpool.Pool) (*RegistroOperacionContactoPostgreSQL, error) {
	return nuevoRegistroOperacionContactoPostgreSQL(pool)
}

func nuevoRegistroOperacionContactoPostgreSQL(pool iniciadorTransacciones) (*RegistroOperacionContactoPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &RegistroOperacionContactoPostgreSQL{pool: pool}, nil
}

func (r *RegistroOperacionContactoPostgreSQL) GuardarContactoUsuario(ctx context.Context, orden ports.OrdenRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	if r == nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	return guardarContactoPostgreSQL(ctx, r.pool, orden, true)
}

func (r *RegistroOperacionContactoPostgreSQL) PrepararOperacionContacto(ctx context.Context, o ports.OrdenPrepararOperacionContacto) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	a := o.Acceso
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || !vecapp.ReferenciaOperacionContactoValida(o.OperacionRef) || !vecapp.ReferenciaOperacionContactoValida(a.Recurso.Atributos["contacto_operacion_ref"]) || a.Recurso.Atributos["contacto_operacion_ref"] != o.OperacionRef || o.SujetoRef != a.SujetoRef || a.ContextoActor.PersonaRef != o.SujetoRef || o.VersionEsperada >= 1<<53-1 || a.Material.ValidarEstructura() != nil || len(o.HuellasReplay) == 0 {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	var ref, estado, consumoRef, consumoHuella string
	var claveHMAC, valorHMAC string
	var version uint64
	var versionResultado *uint64
	var reciboRef *string
	var replay, conflictoMaterial, conflictoVersion bool
	var recibo []byte
	err = tx.QueryRow(ctx, `SELECT operacion_ref,estado,version_esperada,replay_confirmado,version_resultante,recibo_ref,conflicto_material,conflicto_version,auditoria_central,consumo_ref,consumo_huella_sha256,hmac_clave_ref,hmac_valor FROM vec_contacto_usuario_v1.preparar_operacion_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionPrepararOperacionContacto, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&ref, &estado, &version, &replay, &versionResultado, &reciboRef, &conflictoMaterial, &conflictoVersion, &recibo, &consumoRef, &consumoHuella, &claveHMAC, &valorHMAC)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	estadoAudit := estado
	if conflictoMaterial || conflictoVersion {
		estadoAudit = "conflicto"
	}
	evidencia, err := vecapp.ValidarEvidenciaCentralOperacionContacto(recibo, a, o.OperacionRef, estadoAudit, consumoRef, consumoHuella)
	if err != nil || version != o.VersionEsperada || (conflictoVersion && ref != "") || (!conflictoVersion && !vecapp.ReferenciaOperacionContactoValida(ref)) || (ref != o.OperacionRef && !replay && !conflictoMaterial) || (replay && conflictoMaterial) || (estado == string(ports.OperacionContactoConfirmada) && (!replay && !conflictoMaterial || versionResultado == nil || reciboRef == nil)) {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, vecapp.ErrContactoUsuarioCommitIncierto
	}
	if conflictoVersion {
		return vacio, vecapp.ErrContactoUsuarioConflicto
	}
	op := ports.OperacionContactoUsuario{OperacionRef: ref, Estado: ports.EstadoOperacionContactoUsuario(estado), VersionEsperada: version, ReplayConfirmado: replay, AuditoriaOperacion: evidencia, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella, HuellaOriginal: ports.HuellaSolicitudContactoUsuario{ClaveRef: claveHMAC, ValorHMACSHA256: valorHMAC}}
	if versionResultado != nil {
		op.Version = *versionResultado
	}
	if reciboRef != nil {
		op.ReciboRef = *reciboRef
	}
	if vecapp.ValidarOperacionContacto(op) != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if conflictoMaterial {
		return op, vecapp.ErrOperacionContactoPreparada
	}
	return op, nil
}

func (r *RegistroOperacionContactoPostgreSQL) CancelarOperacionContacto(ctx context.Context, o ports.OrdenCancelarOperacionContacto) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	a := o.Acceso
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || !vecapp.ReferenciaOperacionContactoValida(o.OperacionRef) || a.Recurso.Atributos["contacto_operacion_ref"] != o.OperacionRef || o.SujetoRef != a.SujetoRef || a.ContextoActor.PersonaRef != o.SujetoRef || a.Material.ValidarEstructura() != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	var ref, estado, consumoRef, consumoHuella string
	var version uint64
	var replay, encontrada, conflicto bool
	var recibo []byte
	err = tx.QueryRow(ctx, `SELECT operacion_ref,estado,version_esperada,replay_confirmado,encontrada,conflicto,auditoria_central,consumo_ref,consumo_huella_sha256 FROM vec_contacto_usuario_v1.cancelar_operacion_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionCancelarOperacionContacto, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&ref, &estado, &version, &replay, &encontrada, &conflicto, &recibo, &consumoRef, &consumoHuella)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	evidencia, err := vecapp.ValidarEvidenciaCentralOperacionContacto(recibo, a, o.OperacionRef, estado, consumoRef, consumoHuella)
	if err != nil || (encontrada && ref != o.OperacionRef) || (!encontrada && ref != "") || (conflicto && estado != "conflicto") || (!conflicto && encontrada && estado != "cancelada") {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, vecapp.ErrContactoUsuarioCommitIncierto
	}
	if !encontrada {
		return vacio, vecapp.ErrOperacionContactoNoEncontrada
	}
	if conflicto {
		return vacio, vecapp.ErrContactoUsuarioConflicto
	}
	op := ports.OperacionContactoUsuario{OperacionRef: ref, Estado: ports.OperacionContactoCancelada, VersionEsperada: version, ReplayConfirmado: replay, AuditoriaOperacion: evidencia, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella}
	if vecapp.ValidarOperacionContacto(op) != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	return op, nil
}

func (r *RegistroOperacionContactoPostgreSQL) ListarOperacionesContacto(ctx context.Context, o ports.OrdenListarOperacionesContacto) (ports.ResultadoListaOperacionesContacto, error) {
	vacio := ports.ResultadoListaOperacionesContacto{}
	a := o.Acceso
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || o.SujetoRef != a.SujetoRef || a.ContextoActor.PersonaRef != o.SujetoRef || o.Limite < 1 || o.Limite > 50 || o.DespuesDe != "" && !vecapp.ReferenciaOperacionContactoValida(o.DespuesDe) || a.Material.ValidarEstructura() != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	var bruto, recibo []byte
	var siguiente, consumoRef, consumoHuella string
	var cursorEncontrado bool
	err = tx.QueryRow(ctx, `SELECT operaciones,siguiente_desde,cursor_encontrado,auditoria_central,consumo_ref,consumo_huella_sha256 FROM vec_contacto_usuario_v1.listar_operaciones_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionListarOperacionesContacto, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&bruto, &siguiente, &cursorEncontrado, &recibo, &consumoRef, &consumoHuella)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	evidencia, err := vecapp.ValidarEvidenciaCentralOperacionContacto(recibo, a, o.DespuesDe, "consulta", consumoRef, consumoHuella)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	operaciones, err := decodificarListaOperacionesContactoSQL(bruto)
	if err != nil || len(operaciones) > int(o.Limite) || (siguiente != "" && (len(operaciones) == 0 || siguiente != operaciones[len(operaciones)-1].OperacionRef)) || o.DespuesDe == "" && !cursorEncontrado {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, vecapp.ErrContactoUsuarioCommitIncierto
	}
	if !cursorEncontrado {
		return vacio, vecapp.ErrOperacionContactoNoEncontrada
	}
	return ports.ResultadoListaOperacionesContacto{Operaciones: operaciones, SiguienteDesde: siguiente, Auditoria: evidencia, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella}, nil
}

func (r *RegistroOperacionContactoPostgreSQL) DetalleOperacionContacto(ctx context.Context, o ports.OrdenDetalleOperacionContacto) (ports.ResultadoDetalleOperacionContacto, error) {
	vacio := ports.ResultadoDetalleOperacionContacto{}
	a := o.Acceso
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil || !vecapp.ReferenciaOperacionContactoValida(o.OperacionRef) || o.SujetoRef != a.SujetoRef || a.ContextoActor.PersonaRef != o.SujetoRef || a.Recurso.Atributos["contacto_operacion_ref"] != o.OperacionRef || a.Material.ValidarEstructura() != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(a.Recurso, a.Auditoria)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return vacio, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return vacio, contactoError(ctx, err)
	}
	var encontrada bool
	var bruto, recibo []byte
	var consumoRef, consumoHuella string
	err = tx.QueryRow(ctx, `SELECT encontrada,operacion,auditoria_central,consumo_ref,consumo_huella_sha256 FROM vec_contacto_usuario_v1.detalle_operacion_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionDetalleOperacionContacto, a.Material, a.PayloadNegocio, recurso, auditoria)...).Scan(&encontrada, &bruto, &recibo, &consumoRef, &consumoHuella)
	if err != nil {
		return vacio, contactoError(ctx, err)
	}
	estadoAudit := "ausente"
	if encontrada {
		estadoAudit = "encontrada"
	}
	evidencia, err := vecapp.ValidarEvidenciaCentralOperacionContacto(recibo, a, o.OperacionRef, estadoAudit, consumoRef, consumoHuella)
	if err != nil {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	var op ports.OperacionContactoUsuario
	if encontrada {
		op, err = decodificarOperacionContactoSQL(bruto)
		if err != nil || op.OperacionRef != o.OperacionRef {
			return vacio, vecapp.ErrContactoUsuarioNoDisponible
		}
	} else if !bytes.Equal(bytes.TrimSpace(bruto), []byte("{}")) {
		return vacio, vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, vecapp.ErrContactoUsuarioCommitIncierto
	}
	return ports.ResultadoDetalleOperacionContacto{Encontrada: encontrada, Operacion: op, Auditoria: evidencia, ConsumoRef: consumoRef, ConsumoHuellaSHA256: consumoHuella}, nil
}

func decodificarListaOperacionesContactoSQL(raw []byte) ([]ports.OperacionContactoUsuario, error) {
	var filas []json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&filas) != nil || dec.Decode(new(any)) != io.EOF || filas == nil || len(filas) > 50 {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	salida := make([]ports.OperacionContactoUsuario, 0, len(filas))
	for _, fila := range filas {
		op, err := decodificarOperacionContactoSQL(fila)
		if err != nil {
			return nil, err
		}
		salida = append(salida, op)
	}
	return salida, nil
}

func decodificarOperacionContactoSQL(raw []byte) (ports.OperacionContactoUsuario, error) {
	var fila struct {
		OperacionRef    string  `json:"operacion_ref"`
		Estado          string  `json:"estado"`
		VersionEsperada uint64  `json:"version_esperada"`
		Version         *uint64 `json:"version"`
		ReciboRef       *string `json:"recibo_ref"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&fila) != nil || dec.Decode(new(any)) != io.EOF {
		return ports.OperacionContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	op := ports.OperacionContactoUsuario{OperacionRef: fila.OperacionRef, Estado: ports.EstadoOperacionContactoUsuario(fila.Estado), VersionEsperada: fila.VersionEsperada}
	if fila.Version != nil {
		op.Version = *fila.Version
	}
	if fila.ReciboRef != nil {
		op.ReciboRef = *fila.ReciboRef
	}
	if vecapp.ValidarOperacionContacto(op) != nil || op.Estado != ports.OperacionContactoConfirmada && (fila.Version != nil || fila.ReciboRef != nil) || op.Estado == ports.OperacionContactoConfirmada && (fila.Version == nil || fila.ReciboRef == nil) {
		return ports.OperacionContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	return op, nil
}

var _ ports.RegistroContactoUsuario = (*RegistroOperacionContactoPostgreSQL)(nil)
var _ ports.RepositorioOperacionesContactoUsuario = (*RegistroOperacionContactoPostgreSQL)(nil)
