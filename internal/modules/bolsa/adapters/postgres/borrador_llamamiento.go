// Package postgres contiene el adaptador durable del borrador interno de
// llamamiento. La cuenta runtime solo invoca las dos funciones nominales de
// Bolsa; no consulta tablas ni concede autoridad.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionGuardarBorradorLlamamientoInternoPostgreSQL   = "vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1"
	funcionConsultarBorradorLlamamientoInternoPostgreSQL = "vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1"
	esquemaCrearBorradorLlamamientoPostgreSQL            = "vec.bolsa.llamamiento.borrador-interno.crear.v1"
	esquemaConsultaBorradorLlamamientoPostgreSQL         = "vec.bolsa.llamamiento.borrador-interno-consulta.v1"
)

var (
	_ puertosbolsa.TransaccionBorradorLlamamiento = (*RepositorioBorradorLlamamientoPostgreSQL)(nil)
	_ puertosbolsa.LectorBorradorLlamamiento      = (*RepositorioBorradorLlamamientoPostgreSQL)(nil)
)

// RepositorioBorradorLlamamientoPostgreSQL conserva creación y lectura
// auditada en transacciones SERIALIZABLE separadas pero bajo el mismo contrato
// V3. El lector es read-write porque la función SQL registra la consulta.
type RepositorioBorradorLlamamientoPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoRepositorioBorradorLlamamientoPostgreSQL(pool *pgxpool.Pool) (*RepositorioBorradorLlamamientoPostgreSQL, error) {
	return nuevoRepositorioBorradorLlamamientoPostgreSQL(pool)
}

func nuevoRepositorioBorradorLlamamientoPostgreSQL(pool iniciadorTransacciones) (*RepositorioBorradorLlamamientoPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	return &RepositorioBorradorLlamamientoPostgreSQL{pool: pool}, nil
}

func (r *RepositorioBorradorLlamamientoPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return nil, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		revertir(tx)
		return nil, errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	return tx, nil
}

func (r *RepositorioBorradorLlamamientoPostgreSQL) CrearBorradorLlamamiento(ctx context.Context, comando puertosbolsa.ComandoCrearBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	if ctx == nil || comando.Validar() != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	comandoJSON, err := serializarCrearBorradorLlamamientoPostgreSQL(comando)
	if err != nil || !materialCrearBorradorLlamamientoExacto(comando) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	defer revertir(tx)
	var ref, reciboRef, huella string
	var reintento bool
	var registrado time.Time
	err = tx.QueryRow(ctx, `SELECT borrador_ref,recibo_ref,huella_comando_sha256,reintento_idempotente,registrado_en FROM `+funcionGuardarBorradorLlamamientoInternoPostgreSQL+`($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		comandoJSON, comando.Material.CapacidadCanonica(), comando.Material.DecisionCanonica(), comando.Material.MotivoCanonico(), comando.Material.ContextoActorCanonico(), comando.Material.PersonaVersion(), comando.Material.PerfilVersion(), comando.Material.PayloadVECAD3(), comando.Material.SobreCOSESign1(), comando.Material.EvidenciaVerificacion(), comando.Material.RaizPublicaSPKI()).Scan(&ref, &reciboRef, &huella, &reintento, &registrado)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorCrearBorradorLlamamientoPostgreSQL(ctx, err)
	}
	registrado = registrado.UTC()
	resultado := puertosbolsa.ReciboBorradorLlamamiento{Referencia: reciboRef, Borrador: comando.Borrador, HuellaComandoSHA256: huella, ReintentoIdempotente: reintento, RegistradoEn: registrado}
	if ref != comando.Borrador.Referencia() || huella != comando.HuellaComandoSHA256 || !referenciaReciboBorradorLlamamientoValida(reciboRef) || resultado.Validar() != nil || !instantePostgreSQLLlamamientoValido(registrado) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	return resultado, nil
}

func (r *RepositorioBorradorLlamamientoPostgreSQL) ObtenerBorradorLlamamiento(ctx context.Context, referencia, propietario, unidad, ambito string, solicitud dominiovec.SolicitudAutorizacionLigadaV3, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	consulta, err := serializarConsultaBorradorLlamamientoPostgreSQL(referencia, propietario, unidad, ambito, solicitud)
	if err != nil || !materialConsultaBorradorLlamamientoExacto(referencia, propietario, unidad, ambito, solicitud, decision, confirmacion, material) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, err
	}
	defer revertir(tx)
	var ref, dueño, unidadDB, ambitoDB, contenidoJSON, estado, reciboRef, huella string
	var version uint64
	var creado time.Time
	err = tx.QueryRow(ctx, `SELECT borrador_ref,propietario_ref,unidad_ref,ambito_ref,contenido_canonico,estado,version,huella_comando_sha256,recibo_ref,creada_en FROM `+funcionConsultarBorradorLlamamientoInternoPostgreSQL+`($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		consulta, material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(), material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI()).Scan(&ref, &dueño, &unidadDB, &ambitoDB, &contenidoJSON, &estado, &version, &huella, &reciboRef, &creado)
	if err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorConsultarBorradorLlamamientoPostgreSQL(ctx, err)
	}
	var contenido struct {
		Resumen string `json:"resumen"`
	}
	if !jsonExactoBorradorLlamamientoPostgreSQL([]byte(contenidoJSON), &contenido) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	borrador, err := dominiobolsa.NuevoBorradorLlamamiento(ref, dueño, unidadDB, ambitoDB, dominiobolsa.ContenidoBorradorLlamamiento{Resumen: contenido.Resumen})
	creado = creado.UTC()
	resultado := puertosbolsa.ReciboBorradorLlamamiento{Referencia: reciboRef, Borrador: borrador, HuellaComandoSHA256: huella, RegistradoEn: creado}
	if err != nil || estado != string(dominiobolsa.EstadoBorradorLlamamientoInterno) || version != 1 || ref != referencia || dueño != propietario || unidadDB != unidad || ambitoDB != ambito || ref != "borrador-llamamiento:alta:"+huella || !referenciaReciboBorradorLlamamientoValida(reciboRef) || resultado.Validar() != nil || !instantePostgreSQLLlamamientoValido(creado) {
		return puertosbolsa.ReciboBorradorLlamamiento{}, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, errorBorradorLlamamientoPostgreSQL(ctx, err)
	}
	return resultado, nil
}

func serializarCrearBorradorLlamamientoPostgreSQL(c puertosbolsa.ComandoCrearBorradorLlamamiento) ([]byte, error) {
	if c.Validar() != nil {
		return nil, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida
	}
	b := c.Borrador
	return dominiobolsa.RepresentacionCanonicaComandoCrearBorradorLlamamiento(
		b.PropietarioRef(), b.UnidadRef(), b.AmbitoRef(), c.ClaveIdempotencia, b.Contenido(),
	)
}

func serializarConsultaBorradorLlamamientoPostgreSQL(ref, propietario, unidad, ambito string, solicitud dominiovec.SolicitudAutorizacionLigadaV3) ([]byte, error) {
	datos, err := solicitud.Datos()
	if err != nil || datos.Recurso.Referencia != ref {
		return nil, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil || vinculo.PrincipalID != propietario {
		return nil, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida
	}
	return json.Marshal(struct {
		Esquema        string `json:"esquema"`
		BorradorRef    string `json:"borrador_ref"`
		PropietarioRef string `json:"propietario_ref"`
		UnidadRef      string `json:"unidad_ref"`
		AmbitoRef      string `json:"ambito_ref"`
	}{esquemaConsultaBorradorLlamamientoPostgreSQL, ref, propietario, unidad, ambito})
}

func materialCrearBorradorLlamamientoExacto(c puertosbolsa.ComandoCrearBorradorLlamamiento) bool {
	datos, err := c.SolicitudAutorizacion.Datos()
	if err != nil || c.Decision.ValidarPara(c.SolicitudAutorizacion) != nil || c.Confirmacion.Validar() != nil || c.Material.ValidarEstructura() != nil {
		return false
	}
	huella, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := c.Material.ResumenCapacidad()
	return err == nil && datos.Accion == puertosbolsa.AccionCrearBorradorLlamamientoInterno && datos.Finalidad == puertosbolsa.FinalidadCrearBorradorLlamamientoInterno && datos.Recurso.Referencia == c.Borrador.Referencia() && datos.Recurso.ModuloID == puertosbolsa.ModuloBorradorLlamamiento && datos.Recurso.Tipo == puertosbolsa.TipoRecursoBorradorLlamamiento && resumen.Operacion() == datos.Accion && resumen.EfectoRef() == datos.Recurso.Referencia && resumen.EfectoHuellaSHA256() == huella && resumen.AudienciaConsumo() == puertosbolsa.AudienciaCrearBorradorLlamamientoInterno
}

func materialConsultaBorradorLlamamientoExacto(ref, propietario, unidad, ambito string, solicitud dominiovec.SolicitudAutorizacionLigadaV3, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	datos, err := solicitud.Datos()
	if err != nil || decision.ValidarPara(solicitud) != nil || confirmacion.Validar() != nil || material.ValidarEstructura() != nil {
		return false
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	huella, huellaErr := datos.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	return err == nil && huellaErr == nil && vinculo.PrincipalID == propietario && datos.Accion == puertosbolsa.AccionConsultarBorradorLlamamientoInterno && datos.Finalidad == puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno && datos.Recurso.Referencia == ref && datos.Recurso.ModuloID == puertosbolsa.ModuloBorradorLlamamiento && datos.Recurso.Tipo == puertosbolsa.TipoRecursoBorradorLlamamiento && datos.Recurso.Ambitos["unidad_ref"] == unidad && datos.Recurso.Ambitos["ambito_ref"] == ambito && len(datos.Recurso.Ambitos) == 2 && resumen.Operacion() == datos.Accion && resumen.EfectoRef() == ref && resumen.EfectoHuellaSHA256() == huella && resumen.AudienciaConsumo() == puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno
}

func jsonExactoBorradorLlamamientoPostgreSQL(b []byte, destino any) bool {
	if len(b) == 0 || len(b) > 16384 || decodificarJSONExactoLlamamiento(b, destino) != nil {
		return false
	}
	return true
}

func referenciaReciboBorradorLlamamientoValida(ref string) bool {
	if len(ref) != len("recibo:")+64 || ref[:len("recibo:")] != "recibo:" {
		return false
	}
	for _, c := range ref[len("recibo:"):] {
		if c < 'a' || c > 'p' {
			return false
		}
	}
	return true
}

func errorBorradorLlamamientoPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
}
func errorCrearBorradorLlamamientoPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VBL01":
			return puertosbolsa.ErrClaveBorradorLlamamientoReutilizada
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		}
	}
	return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
}
func errorConsultarBorradorLlamamientoPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible
}
