package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const funcionPrepararEntregaConOriginal = "vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1"
const maximoVersionEnteraPostgreSQL = uint64(1<<63 - 1)

type verificadorOriginalAltaEntrega interface {
	VerificarOriginalAltaEntrega(context.Context, ports.EntregaPeticionCentro, ports.OriginalAltaEntrega) error
}

type ProveedorEntregaPeticionCentro interface {
	ActorEntregaPeticionCentro(context.Context) (string, string, error)
	ComprobarPerfilEntregaPeticionCentro(context.Context) error
	RegistrarDenegacionEntregaPreV3(context.Context) error
	NuevaClaveAltaDePeticion(context.Context) (string, string, error)
	AutorizarEntregaPeticionCentro(context.Context, ports.MaterialEntregaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type lectorAmbitosEntregaPeticionCentro interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RepositorioEntregasPeticionCentroPostgreSQL struct {
	pool      iniciadorTransacciones
	lector    lectorAmbitosEntregaPeticionCentro
	proveedor ProveedorEntregaPeticionCentro
}

func NuevoRepositorioEntregasPeticionCentroPostgreSQL(pool *pgxpool.Pool, p ProveedorEntregaPeticionCentro) (*RepositorioEntregasPeticionCentroPostgreSQL, error) {
	if pool == nil || dependenciaNula(p) {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	return &RepositorioEntregasPeticionCentroPostgreSQL{pool: pool, lector: pool, proveedor: p}, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) material(ctx context.Context, modo string, c ports.ComandoEntregarPeticionCentro) (ports.MaterialEntregaPeticionCentro, error) {
	if ctx == nil || r == nil || r.pool == nil || r.lector == nil || dependenciaNula(r.proveedor) {
		return ports.MaterialEntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
	}
	a, p, err := r.proveedor.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		return ports.MaterialEntregaPeticionCentro{}, err
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: modo, ActorRef: a, PerfilRef: p, PeticionRef: c.PeticionRef, VersionEsperada: c.VersionEsperada}
	if modo != "bandeja" {
		if c.Validar() != nil {
			return ports.MaterialEntregaPeticionCentro{}, domain.ErrPeticionCentroInvalida
		}
		// Antes de proyectar centro/categoría de una referencia, exigir la
		// asignación publicada del perfil de POST. La existencia de la petición
		// no se consulta para un perfil revocado o ausente.
		if err := r.proveedor.ComprobarPerfilEntregaPeticionCentro(ctx); err != nil {
			if errors.Is(err, ports.ErrPeticionCentroNoDisponible) {
				return ports.MaterialEntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
			}
			return ports.MaterialEntregaPeticionCentro{}, ports.ErrAutorizacionDenegada
		}
		// La función gobernada entrega exclusivamente los ámbitos de la revisión
		// ratificada. La transacción del efecto los vuelve a cotejar antes del
		// consumo V3: esta lectura nunca concede por sí sola el permiso.
		if err := r.lector.QueryRow(ctx,
			"SELECT centro_ref,categoria_ref FROM vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1($1)",
			c.PeticionRef).Scan(&m.CentroRef, &m.CategoriaRef); err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "P0681" {
				if r.proveedor.RegistrarDenegacionEntregaPreV3(ctx) != nil {
					return ports.MaterialEntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
				}
				// La proyección no revela si una referencia existe. Solo el efecto
				// autorizado puede producir el 409 de estado/versionado.
				return ports.MaterialEntregaPeticionCentro{}, ports.ErrAutorizacionDenegada
			}
			return ports.MaterialEntregaPeticionCentro{}, errorEntregaPeticionSQL(ctx, err)
		}
	}
	return m, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ListarPeticionesRRHH(ctx context.Context) ([]ports.EntregaPeticionCentro, error) {
	m, err := r.material(ctx, "bandeja", ports.ComandoEntregarPeticionCentro{})
	if err != nil {
		return nil, err
	}
	var filas []ports.EntregaPeticionCentro
	err = r.ejecutar(ctx, m, func(b []byte) error {
		if decodificarJSONEstricto(b, &filas) != nil || filas == nil || len(filas) > 50 {
			return ports.ErrPeticionCentroNoDisponible
		}
		vistas := make(map[string]bool, len(filas))
		for _, f := range filas {
			if f.Validar() != nil || vistas[f.Peticion.Referencia] || f.ClaveAlta != "" || f.ActorRef != "" || f.PerfilRef != "" {
				return ports.ErrReciboPeticionCentroNoConfiable
			}
			vistas[f.Peticion.Referencia] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filas, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) PrepararEntrega(ctx context.Context, c ports.ComandoEntregarPeticionCentro) (ports.EntregaPeticionCentro, error) {
	m, err := r.material(ctx, "preparar", c)
	if err != nil {
		return ports.EntregaPeticionCentro{}, err
	}
	m.ClaveAltaCandidata, m.AmbitoAltaHMAC, err = r.proveedor.NuevaClaveAltaDePeticion(ctx)
	if err != nil {
		return ports.EntregaPeticionCentro{}, err
	}
	return r.entrega(ctx, m, c.NumeroExpedienteMOAD == "")
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ConfirmarEntrega(ctx context.Context, c ports.ComandoEntregarPeticionCentro, alta ports.AltaDePeticionCentro) (ports.EntregaPeticionCentro, error) {
	m, err := r.material(ctx, "confirmar", c)
	if err != nil {
		return ports.EntregaPeticionCentro{}, err
	}
	m.ReciboAlta, m.AmbitoAltaHMAC = &alta.Recibo, alta.AmbitoHMAC
	return r.entrega(ctx, m, false)
}

// Las dos señales proceden de la transacción SQL que decide cada inserción.
// Su ausencia no se interpreta como un replay: impide servir código antiguo
// junto a una migración incompleta o una respuesta no confiable.
type resultadoEntregaPeticionCentroSQL struct {
	ports.EntregaPeticionCentro
	ReservaCreadaAhora      *bool `json:"reserva_creada_ahora"`
	ConfirmacionCreadaAhora *bool `json:"confirmacion_creada_ahora"`
}

func (s resultadoEntregaPeticionCentroSQL) entregaPara(modo string, soloExistente bool) (ports.EntregaPeticionCentro, error) {
	if s.ReservaCreadaAhora == nil || s.ConfirmacionCreadaAhora == nil ||
		(*s.ReservaCreadaAhora && *s.ConfirmacionCreadaAhora) ||
		(*s.ReservaCreadaAhora && (modo != "preparar" || s.EstadoEntrega != "preparada")) ||
		(*s.ConfirmacionCreadaAhora && s.EstadoEntrega != "confirmada") ||
		(modo == "confirmar" && s.EstadoEntrega != "confirmada") {
		return ports.EntregaPeticionCentro{}, ports.ErrReciboPeticionCentroNoConfiable
	}
	if soloExistente && *s.ReservaCreadaAhora {
		return ports.EntregaPeticionCentro{}, ports.ErrNumeroMOADAusente
	}
	e := s.EntregaPeticionCentro
	e.ReservaCreadaAhora = *s.ReservaCreadaAhora
	e.ConfirmadaAhora = *s.ConfirmacionCreadaAhora
	return e, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) entrega(ctx context.Context, m ports.MaterialEntregaPeticionCentro, soloExistente bool) (ports.EntregaPeticionCentro, error) {
	var e ports.EntregaPeticionCentro
	err := r.ejecutarConOriginal(ctx, m, soloExistente, func(b, contenidoOriginal []byte) error {
		var resultado resultadoEntregaPeticionCentroSQL
		if decodificarJSONEstricto(b, &resultado) != nil {
			return ports.ErrReciboPeticionCentroNoConfiable
		}
		var err error
		e, err = resultado.entregaPara(m.Modo, soloExistente)
		if err != nil {
			return err
		}
		if soloExistente {
			if len(contenidoOriginal) == 0 || bytes.Equal(contenidoOriginal, []byte("null")) {
				if e.EstadoEntrega == "confirmada" {
					return ports.ErrReciboPeticionCentroNoConfiable
				}
				return ports.ErrNumeroMOADAusente
			}
			if len(contenidoOriginal) > 8192 {
				return ports.ErrReciboPeticionCentroNoConfiable
			}
			var original ports.OriginalAltaEntrega
			if decodificarJSONEstricto(contenidoOriginal, &original) != nil || original.Validar() != nil {
				return ports.ErrReciboPeticionCentroNoConfiable
			}
			verificador, ok := r.proveedor.(verificadorOriginalAltaEntrega)
			if !ok || dependenciaNula(verificador) {
				return ports.ErrPeticionCentroNoDisponible
			}
			if err := verificador.VerificarOriginalAltaEntrega(ctx, e, original); err != nil {
				return err
			}
			e.AltaAnterior = &ports.AltaDePeticionCentro{Recibo: original.ReciboAlta, AmbitoHMAC: e.AmbitoAltaHMAC}
		}
		// CT150 puede devolver una confirmación histórica cuyo perfil reservado
		// precede al perfil fijo. Solo preparar admite ese replay: la función SQL
		// ya ha cotejado la reserva, el alta durable y la decisión vigente.
		perfilHistoricoConfirmado := m.Modo == "preparar" && e.EstadoEntrega == "confirmada"
		if e.ValidarReserva() != nil || e.Peticion.Referencia != m.PeticionRef || e.ActorRef != m.ActorRef ||
			(e.PerfilRef != m.PerfilRef && !perfilHistoricoConfirmado) {
			return ports.ErrReciboPeticionCentroNoConfiable
		}
		return nil
	})
	if err != nil {
		return ports.EntregaPeticionCentro{}, err
	}
	return e, nil
}

func AccionEntregaPeticionCentro(m ports.MaterialEntregaPeticionCentro) string {
	if m.Modo == "bandeja" {
		return ports.AccionConsultarPeticionesRRHH
	}
	return ports.AccionEntregarPeticionRRHH
}

func RecursoEntregaPeticionCentro(m ports.MaterialEntregaPeticionCentro) (vecdomain.RecursoAutorizable, error) {
	if err := m.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	ref := m.PeticionRef
	if m.Modo == "bandeja" {
		ref = "peticiones:centro:rrhh"
	}
	ambitos := map[string]string{"organizacion_ref": "organizacion:desarrollo:dipgra"}
	if m.Modo != "bandeja" {
		ambitos["centro_ref"], ambitos["categoria_ref"] = m.CentroRef, m.CategoriaRef
	}
	return vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoEntregaPeticionCentro,
		Ambitos: ambitos, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ejecutar(ctx context.Context, m ports.MaterialEntregaPeticionCentro, validar func([]byte) error) error {
	return r.ejecutarConOriginal(ctx, m, false, func(entrega, _ []byte) error { return validar(entrega) })
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ejecutarConOriginal(
	ctx context.Context, m ports.MaterialEntregaPeticionCentro, conOriginal bool,
	validar func([]byte, []byte) error,
) error {
	recurso, err := RecursoEntregaPeticionCentro(m)
	if err != nil {
		return err
	}
	a, err := r.proveedor.AutorizarEntregaPeticionCentro(ctx, m)
	if err != nil {
		return err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || resumen.Operacion() != AccionEntregaPeticionCentro(m) || resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != audienciaPeticionCentro {
		return ports.ErrAutorizacionDenegada
	}
	if a.PersonaVersion() > maximoVersionEnteraPostgreSQL || a.PerfilVersion() > maximoVersionEnteraPostgreSQL {
		return ports.ErrPeticionCentroNoDisponible
	}
	personaVersion := int64(a.PersonaVersion()) // #nosec G115 -- comprobada frente a MaxInt64 arriba.
	perfilVersion := int64(a.PerfilVersion())   // #nosec G115 -- comprobada frente a MaxInt64 arriba.
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	defer clear(b)
	tx, err := iniciarTransaccionAltaCandidata(ctx, r.pool)
	if err != nil {
		return errorEntregaPeticionSQL(ctx, err)
	}
	defer revertirTransaccion(tx)
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			clear(b)
		}
	}()
	var salida []byte
	var original []byte
	if conOriginal {
		err = tx.QueryRow(ctx, "SELECT entrega::text,original_alta::text FROM "+funcionPrepararEntregaConOriginal+"($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)", string(b), secretos[0], secretos[1], secretos[2], secretos[3], personaVersion, perfilVersion, secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida, &original)
	} else {
		err = tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", string(b), secretos[0], secretos[1], secretos[2], secretos[3], personaVersion, perfilVersion, secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	}
	if err != nil {
		return errorEntregaPeticionSQL(ctx, err)
	}
	defer clear(salida)
	defer clear(original)
	if len(salida) == 0 || len(salida) > 2*1024*1024 {
		return ports.ErrPeticionCentroNoDisponible
	}
	if err = validar(salida, original); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return errorEntregaPeticionSQL(ctx, err)
	}
	return nil
}

func errorEntregaPeticionSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P0680":
			return domain.ErrPeticionCentroInvalida
		case "P0681":
			return ports.ErrEntregaPeticionEnConflicto
		case "P0683", "42501":
			return ports.ErrAutorizacionDenegada
		}
	}
	return ports.ErrPeticionCentroNoDisponible
}
