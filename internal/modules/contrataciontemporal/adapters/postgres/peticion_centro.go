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

const audienciaPeticionCentro = "vec_contratacion_temporal.confirmar_alta_atestada.v1"

// El proveedor aplica la identidad y política vigentes del servidor y entrega
// una autorización nueva, ligada al material exacto, para cada consulta/efecto.
type ProveedorAutorizacionPeticionCentro interface {
	AutorizarLecturaPeticionCentro(context.Context, ports.ConsultaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	AutorizarEscrituraPeticionCentro(context.Context, ports.MaterialPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type RepositorioPeticionesCentroPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorAutorizacionPeticionCentro
}

var _ ports.RepositorioPeticionesCentro = (*RepositorioPeticionesCentroPostgreSQL)(nil)
var _ ports.ConsultaBandejaPeticionesCentro = (*RepositorioPeticionesCentroPostgreSQL)(nil)

func NuevoRepositorioPeticionesCentroPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorAutorizacionPeticionCentro) (*RepositorioPeticionesCentroPostgreSQL, error) {
	if pool == nil || proveedor == nil {
		return nil, ports.ErrPeticionCentroNoDisponible
	}
	return &RepositorioPeticionesCentroPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (r *RepositorioPeticionesCentroPostgreSQL) ConsultarOperacion(ctx context.Context, actor domain.ActorPeticionCentro, clave string) (*ports.MaterialPeticionCentro, error) {
	c := ports.ConsultaPeticionCentro{Modo: "operacion", Actor: actor, Referencia: clave}
	var material *ports.MaterialPeticionCentro
	err := r.consultar(ctx, c, func(b []byte) error {
		if err := decodificarJSONEstricto(b, &material); err != nil {
			return ports.ErrPeticionCentroNoDisponible
		}
		if material != nil && (material.Validar() != nil || material.Actor != actor || material.Comando.ClaveIdempotencia != clave) {
			return domain.ErrRatificacionCentroDenegada
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return material, nil
}

func (r *RepositorioPeticionesCentroPostgreSQL) ObtenerPeticion(ctx context.Context, actor domain.ActorPeticionCentro, referencia string) (domain.DatosPeticionCentro, error) {
	c := ports.ConsultaPeticionCentro{Modo: "peticion", Actor: actor, Referencia: referencia}
	var datos domain.DatosPeticionCentro
	err := r.consultar(ctx, c, func(b []byte) error {
		if err := decodificarJSONEstricto(b, &datos); err != nil {
			return ports.ErrPeticionCentroNoDisponible
		}
		if datos.Referencia != referencia || !peticionCentroVisiblePara(datos, actor) {
			return domain.ErrRatificacionCentroDenegada
		}
		_, err := domain.RehidratarPeticionCentro(datos)
		return err
	})
	if err != nil {
		return domain.DatosPeticionCentro{}, err
	}
	return datos, nil
}

func (r *RepositorioPeticionesCentroPostgreSQL) ListarPeticiones(ctx context.Context, actor domain.ActorPeticionCentro) ([]domain.DatosPeticionCentro, error) {
	c := ports.ConsultaPeticionCentro{Modo: "bandeja", Actor: actor, Referencia: actor.CentroRef}
	var datos []domain.DatosPeticionCentro
	err := r.consultar(ctx, c, func(b []byte) error {
		if err := decodificarJSONEstricto(b, &datos); err != nil || datos == nil || len(datos) > 50 {
			return ports.ErrPeticionCentroNoDisponible
		}
		vistos := make(map[string]bool, len(datos))
		for _, p := range datos {
			if vistos[p.Referencia] || !peticionCentroVisiblePara(p, actor) {
				return domain.ErrRatificacionCentroDenegada
			}
			if _, err := domain.RehidratarPeticionCentro(p); err != nil {
				return err
			}
			vistos[p.Referencia] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return datos, nil
}

func peticionCentroVisiblePara(p domain.DatosPeticionCentro, a domain.ActorPeticionCentro) bool {
	return a.Validar() == nil && (a == p.Configuracion.Solicitante || a == p.Configuracion.Ratificador)
}

func (r *RepositorioPeticionesCentroPostgreSQL) consultar(ctx context.Context, c ports.ConsultaPeticionCentro, validar func([]byte) error) error {
	if r == nil || r.pool == nil || r.proveedor == nil || ctx == nil {
		return ports.ErrPeticionCentroNoDisponible
	}
	if err := c.Validar(); err != nil {
		return err
	}
	a, err := r.proveedor.AutorizarLecturaPeticionCentro(ctx, c)
	if err != nil {
		return err
	}
	recurso, err := RecursoConsultaPeticionCentro(c)
	if err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return ports.ErrPeticionCentroNoDisponible
	}
	return r.ejecutar(ctx, "consultar_peticiones_centro_v1", b, a, ports.AccionConsultarPeticionCentro, recurso, validar)
}

func (r *RepositorioPeticionesCentroPostgreSQL) ConfirmarPeticion(ctx context.Context, m ports.MaterialPeticionCentro) (ports.ReciboPeticionCentro, error) {
	var recibo ports.ReciboPeticionCentro
	if r == nil || r.pool == nil || r.proveedor == nil || ctx == nil {
		return recibo, ports.ErrPeticionCentroNoDisponible
	}
	if err := m.Validar(); err != nil {
		return recibo, err
	}
	a, err := r.proveedor.AutorizarEscrituraPeticionCentro(ctx, m)
	if err != nil {
		return recibo, err
	}
	recurso, err := RecursoEscrituraPeticionCentro(m)
	if err != nil {
		return recibo, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > 64*1024 {
		return recibo, domain.ErrPeticionCentroInvalida
	}
	err = r.ejecutar(ctx, "registrar_peticion_centro_v1", b, a, AccionEscrituraPeticionCentro(m), recurso, func(b []byte) error {
		if err := decodificarJSONEstricto(b, &recibo); err != nil {
			return ports.ErrReciboPeticionCentroNoConfiable
		}
		recibo.RegistradoEn = recibo.RegistradoEn.UTC()
		return recibo.ValidarPara(m)
	})
	if err != nil {
		return ports.ReciboPeticionCentro{}, err
	}
	return recibo, nil
}

func AccionEscrituraPeticionCentro(m ports.MaterialPeticionCentro) string {
	if m.Comando.Operacion == ports.OperacionPresentarPeticionCentro {
		return ports.AccionPresentarPeticionCentro
	}
	if m.Comando.Operacion == ports.OperacionRatificarPeticionCentro {
		return ports.AccionRatificarPeticionCentro
	}
	return ""
}

func RecursoEscrituraPeticionCentro(m ports.MaterialPeticionCentro) (vecdomain.RecursoAutorizable, error) {
	if err := m.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	return recursoPeticionCentro(m.Peticion.Referencia, m.Actor.CentroRef, m)
}

func RecursoConsultaPeticionCentro(c ports.ConsultaPeticionCentro) (vecdomain.RecursoAutorizable, error) {
	if err := c.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	ref := c.Referencia
	if c.Modo == "operacion" {
		ref = "operacion:peticion-centro:" + ref
	}
	if c.Modo == "bandeja" {
		ref = "peticiones:centro:" + ref
	}
	return recursoPeticionCentro(ref, c.Actor.CentroRef, c)
}

func recursoPeticionCentro(ref, centro string, material any) (vecdomain.RecursoAutorizable, error) {
	b, err := json.Marshal(material)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrPeticionCentroNoDisponible
	}
	h := sha256.Sum256(b)
	return vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: "contratacion_temporal", Tipo: ports.TipoRecursoPeticionCentro,
		Ambitos:   map[string]string{"organizacion_ref": "organizacion:desarrollo:dipgra", "centro_ref": centro},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}

func (r *RepositorioPeticionesCentroPostgreSQL) ejecutar(ctx context.Context, funcion string, contenido []byte, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string, recurso vecdomain.RecursoAutorizable, validar func([]byte) error) error {
	// Nombres cerrados; nunca se interpola SQL, identidad ni función del cliente.
	if funcion != "consultar_peticiones_centro_v1" && funcion != "registrar_peticion_centro_v1" {
		return ports.ErrPeticionCentroNoDisponible
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia ||
		resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != audienciaPeticionCentro {
		return domain.ErrRatificacionCentroDenegada
	}
	tx, err := iniciarTransaccionAltaCandidata(ctx, r.pool)
	if err != nil {
		return errorPeticionCentroSQL(ctx, err)
	}
	defer revertirTransaccion(tx)
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			clear(b)
		}
	}()
	var salida []byte
	err = tx.QueryRow(ctx, "SELECT vec_contratacion_temporal."+funcion+"($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text", string(contenido), secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return errorPeticionCentroSQL(ctx, err)
	}
	defer clear(salida)
	if len(salida) == 0 || len(salida) > 2*1024*1024 {
		return ports.ErrPeticionCentroNoDisponible
	}
	if err := validar(salida); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errorPeticionCentroSQL(ctx, err)
	}
	return nil
}

func errorPeticionCentroSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P0670":
			return domain.ErrPeticionCentroInvalida
		case "P0671":
			return ports.ErrClavePeticionCentroUsada
		case "P0672":
			return domain.ErrVersionPeticionCentroEnConflicto
		case "P0673", "42501":
			return domain.ErrRatificacionCentroDenegada
		}
	}
	// Incluye concurrencia serializable y COMMIT incierto: conserva la clave
	// para reintentar, nunca declara confirmado un resultado que no se recibió.
	return ports.ErrPeticionCentroNoDisponible
}
