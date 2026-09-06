package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorEntregaPeticionCentro interface {
	ActorEntregaPeticionCentro(context.Context) (string, string, error)
	NuevaClaveAltaDePeticion(context.Context) (string, string, error)
	AutorizarEntregaPeticionCentro(context.Context, ports.MaterialEntregaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type RepositorioEntregasPeticionCentroPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorEntregaPeticionCentro
}

func NuevoRepositorioEntregasPeticionCentroPostgreSQL(pool *pgxpool.Pool, p ProveedorEntregaPeticionCentro) (*RepositorioEntregasPeticionCentroPostgreSQL, error) {
	if pool == nil || dependenciaNula(p) {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	return &RepositorioEntregasPeticionCentroPostgreSQL{pool, p}, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) material(ctx context.Context, modo string, c ports.ComandoEntregarPeticionCentro) (ports.MaterialEntregaPeticionCentro, error) {
	if ctx == nil || r == nil || r.pool == nil || dependenciaNula(r.proveedor) {
		return ports.MaterialEntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
	}
	a, p, err := r.proveedor.ActorEntregaPeticionCentro(ctx)
	return ports.MaterialEntregaPeticionCentro{Modo: modo, ActorRef: a, PerfilRef: p, PeticionRef: c.PeticionRef, VersionEsperada: c.VersionEsperada}, err
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
	return r.entrega(ctx, m)
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ConfirmarEntrega(ctx context.Context, c ports.ComandoEntregarPeticionCentro, alta ports.AltaDePeticionCentro) (ports.EntregaPeticionCentro, error) {
	m, err := r.material(ctx, "confirmar", c)
	if err != nil {
		return ports.EntregaPeticionCentro{}, err
	}
	m.ReciboAlta, m.AmbitoAltaHMAC = &alta.Recibo, alta.AmbitoHMAC
	return r.entrega(ctx, m)
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) entrega(ctx context.Context, m ports.MaterialEntregaPeticionCentro) (ports.EntregaPeticionCentro, error) {
	var e ports.EntregaPeticionCentro
	err := r.ejecutar(ctx, m, func(b []byte) error {
		if decodificarJSONEstricto(b, &e) != nil || e.ValidarReserva() != nil || e.Peticion.Referencia != m.PeticionRef || e.ActorRef != m.ActorRef || e.PerfilRef != m.PerfilRef {
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
	return vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoEntregaPeticionCentro,
		Ambitos: map[string]string{"organizacion_ref": "organizacion:desarrollo:dipgra"}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}

func (r *RepositorioEntregasPeticionCentroPostgreSQL) ejecutar(ctx context.Context, m ports.MaterialEntregaPeticionCentro, validar func([]byte) error) error {
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
	err = tx.QueryRow(ctx, "SELECT vec_contratacion_temporal.gestionar_entrega_peticion_centro_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", string(b), secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return errorEntregaPeticionSQL(ctx, err)
	}
	defer clear(salida)
	if len(salida) == 0 || len(salida) > 2*1024*1024 {
		return ports.ErrPeticionCentroNoDisponible
	}
	if err = validar(salida); err != nil {
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
