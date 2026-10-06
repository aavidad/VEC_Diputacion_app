package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	usuariosimagen "vec-diputacion-granada/internal/modules/usuarios/adapters/imagen"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errComposicionUsuariosImagen = errors.New("bootstrap: imagen de Usuarios no disponible")

// Decodificaciones de fotos simultáneas en todo el proceso: cada una puede
// ocupar decenas de MB durante unos cientos de milisegundos.
const decodificacionesFotoSimultaneas = 2

// dependenciasImagenUsuariosDesarrollo agrupa lo común a las dos superficies:
// material V3 publicado y un único transformador (con su límite de turnos).
type dependenciasImagenUsuariosDesarrollo struct {
	materiales    proveedoresMaterialImagenUsuarios
	transformador usuariosports.TransformadorFoto
}

// nuevasDependenciasImagenUsuariosDesarrollo devuelve nil si «Mi imagen» no
// se ha pedido. Pedirla sin preferencias o sin su material es un error.
func nuevasDependenciasImagenUsuariosDesarrollo(cfg config.Config, materiales proveedoresMaterialImagenUsuarios) (*dependenciasImagenUsuariosDesarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosImagenDesarrollo)
	if err != nil || !activo {
		return nil, err
	}
	preferencias, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !preferencias {
		return nil, errComposicionUsuariosImagen
	}
	for _, p := range materiales {
		if p == nil {
			return nil, errComposicionUsuariosImagen
		}
	}
	return &dependenciasImagenUsuariosDesarrollo{materiales: materiales, transformador: usuariosimagen.Nuevo(decodificacionesFotoSimultaneas)}, nil
}

func (d *dependenciasImagenUsuariosDesarrollo) materialesSuperficie(superficie core.SuperficieAutenticacionActorV1) ([2]*proveedorMaterialAltaContratacionTemporalDesarrollo, bool) {
	var r [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	if d == nil {
		return r, false
	}
	inicio := 0
	switch superficie {
	case core.SuperficieAutenticacionInternaCorporativaV1:
	case core.SuperficieAutenticacionExternaPersonalV1:
		inicio = len(r)
	default:
		return r, false
	}
	copy(r[:], d.materiales[inicio:inicio+len(r)])
	return r, true
}

func rutaImagenSuperficie(superficie core.SuperficieAutenticacionActorV1) string {
	switch superficie {
	case core.SuperficieAutenticacionInternaCorporativaV1:
		return usuarioshttp.RutaMiImagen
	case core.SuperficieAutenticacionExternaPersonalV1:
		return usuarioshttp.RutaMiImagenAreaPersonal
	}
	return ""
}

// montarImagenUsuariosSuperficie crea la autoridad hermana de preferencias
// para la ruta de «Mi imagen»: misma identidad, pools y registrador de
// frontera; su propio manejador, proveedor V3 y ruta exacta. No posee pools.
func montarImagenUsuariosSuperficie(ctx context.Context, preferencias *autoridadPreferenciasUsuariosDesarrollo, ejecutor *pgxpool.Pool,
	autorizador vecports.AutorizadorSolicitudLigadaV3, c configuracionUsuariosPreferenciasDesarrollo, d *dependenciasImagenUsuariosDesarrollo,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	if ctx == nil || preferencias == nil || ejecutor == nil || autorizador == nil || d == nil || d.transformador == nil {
		return nil, errComposicionUsuariosImagen
	}
	ruta := rutaImagenSuperficie(preferencias.superficie)
	materiales, ok := d.materialesSuperficie(preferencias.superficie)
	if ruta == "" || !ok {
		return nil, errComposicionUsuariosImagen
	}
	registro, err := usuariospg.NuevoRegistroImagenPostgreSQL(ctx, ejecutor, preferencias.superficie)
	if err != nil {
		return nil, errComposicionUsuariosImagen
	}
	emisores := make(map[string]emisorPreferenciasUsuarios, len(materiales))
	for i, material := range materiales {
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, material)
		if err != nil {
			return nil, errComposicionUsuariosImagen
		}
		emisores[accionesImagenUsuarios[i].accion] = emisor
	}
	servicio, err := usuariosapp.NuevoServicioImagen(registro, d.transformador, time.Now)
	if err != nil {
		return nil, errComposicionUsuariosImagen
	}
	imagen := *preferencias
	imagen.ruta, imagen.cerrar, imagen.manejador, imagen.proveedor = ruta, nil, nil, nil
	imagen.proveedorCorreos, imagen.correos, imagen.imagen = nil, nil, nil
	imagen.prefijoError, imagen.metodoEscritura = "api.usuarios.imagen.error.", "POST"
	imagen.proveedorImagen = &proveedorImagenUsuarios{autoridad: &imagen, emisores: emisores,
		motivoConsulta: c.MotivoConsulta, motivoActualizacion: c.MotivoActualizacion}
	manejador, err := usuarioshttp.NuevoManejadorImagenEnRuta(servicio, &imagen, &imagen, ruta)
	if err != nil {
		return nil, errComposicionUsuariosImagen
	}
	imagen.manejador = manejador
	return &imagen, nil
}

// ResolverOrdenImagen sólo acepta el contexto que esta misma autoridad fijó
// en la frontera para la petición en curso.
func (a *autoridadPreferenciasUsuariosDesarrollo) ResolverOrdenImagen(ctx context.Context) (usuariosports.OrdenImagen, error) {
	if a == nil || ctx == nil || a.proveedorImagen == nil {
		return usuariosports.OrdenImagen{}, usuariosports.ErrImagenNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
		return usuariosports.OrdenImagen{}, usuariosports.ErrImagenNoAutenticado
	}
	return usuariosapp.NuevaOrdenImagen(c.resultado.Contexto, c.vinculo, a.superficie, a.proveedorImagen)
}

// La sonda comprueba, con el LOGIN ejecutor de cada superficie, que las
// fachadas de 000006 y la frontera de 000007 existen antes de publicar las
// cuatro audiencias V3.
const sondaFronteraImagenSQL = `SELECT count(*)=1 FROM pg_catalog.pg_constraint
 WHERE conname='denegacion_frontera_preferencias_ruta_check'
 AND conrelid=pg_catalog.to_regclass('vec_usuarios.denegacion_frontera_preferencias')
 AND position('/api/vec/usuarios/area-personal/mi-imagen' IN pg_catalog.pg_get_constraintdef(oid))>0`

func preflightSQLImagenUsuariosDesarrollo(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancel()
	for _, superficie := range superficiesUsuariosEnProceso(cfg) {
		c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, superficie)
		if err != nil {
			return errComposicionUsuariosImagen
		}
		pool, _, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(superficie)))
		if err != nil {
			return errComposicionUsuariosImagen
		}
		_, errRegistro := usuariospg.NuevoRegistroImagenPostgreSQL(ctx, pool, superficie)
		var frontera bool
		errFrontera := pool.QueryRow(ctx, sondaFronteraImagenSQL).Scan(&frontera)
		pool.Close()
		if errRegistro != nil || errFrontera != nil || !frontera {
			return errComposicionUsuariosImagen
		}
	}
	return nil
}
