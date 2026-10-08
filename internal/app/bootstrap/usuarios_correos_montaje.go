package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errComposicionUsuariosCorreos = errors.New("bootstrap: correos de Usuarios no disponibles")

// dependenciasCorreosUsuariosDesarrollo agrupa lo común a las dos
// superficies: material V3 publicado, claves y transporte SMTP.
type dependenciasCorreosUsuariosDesarrollo struct {
	materiales proveedoresMaterialCorreosUsuarios
	cripto     *usuariosseguridad.AdaptadorCorreos
	transporte usuariosports.TransportadorCorreosPropios
}

// fuenteClavesCorreosDesarrollo deriva del KMS de desarrollo cuatro claves
// independientes (cifrado, igualdad, huella semántica y código). En
// producción la fuente será el KMS corporativo, con rotación y retención.
type fuenteClavesCorreosDesarrollo struct {
	claves usuariosseguridad.ClavesCorreos
}

func (f fuenteClavesCorreosDesarrollo) CargarClavesCorreos(context.Context) (usuariosseguridad.ClavesCorreos, error) {
	return f.claves, nil
}

func nuevaFuenteClavesCorreosDesarrollo(kms *emisorKMSDesarrollo) (fuenteClavesCorreosDesarrollo, error) {
	if kms == nil || claveContactoKMSCero(kms.claveEnvoltura) {
		return fuenteClavesCorreosDesarrollo{}, errComposicionUsuariosCorreos
	}
	clave := func(ambito string) usuariosseguridad.ClaveCorreo {
		return usuariosseguridad.ClaveCorreo{Ref: "clave:kms:desarrollo:usuarios-correos-" + ambito + ":v1",
			Material: derivarClaveDesarrollo(kms.claveEnvoltura, "vec.kms.desarrollo.usuarios-correos."+ambito+".v1")}
	}
	return fuenteClavesCorreosDesarrollo{claves: usuariosseguridad.ClavesCorreos{
		CifradoActivo: clave("cifrado"), Igualdad: clave("igualdad"),
		SemanticaActiva: clave("semantica"), CodigoActivo: clave("codigo"),
	}}, nil
}

// nuevasDependenciasCorreosUsuariosDesarrollo devuelve nil si «Mis correos»
// no se ha pedido. Pedirlo sin preferencias, SMTP o textos es un error.
func nuevasDependenciasCorreosUsuariosDesarrollo(cfg config.Config, kms *emisorKMSDesarrollo, materiales proveedoresMaterialCorreosUsuarios) (*dependenciasCorreosUsuariosDesarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosCorreosDesarrollo)
	if err != nil || !activo {
		return nil, err
	}
	preferencias, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !preferencias {
		return nil, errComposicionUsuariosCorreos
	}
	for _, p := range materiales.lote {
		if p == nil {
			return nil, errComposicionUsuariosCorreos
		}
	}
	fuente, err := nuevaFuenteClavesCorreosDesarrollo(kms)
	if err != nil {
		return nil, err
	}
	cripto, err := usuariosseguridad.NuevoAdaptadorCorreos(fuente, time.Now)
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	smtp, err := nuevoEnviadorCorreoLlamamientoDesarrollo(cfg)
	if err != nil || smtp == nil {
		return nil, errComposicionUsuariosCorreos
	}
	catalogo, err := i18n.Load()
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	transporte, err := nuevoTransporteCorreosPropiosDesarrollo(smtp, catalogo, cfg.SMTPFrom)
	if err != nil {
		return nil, err
	}
	return &dependenciasCorreosUsuariosDesarrollo{materiales: materiales, cripto: cripto, transporte: transporte}, nil
}

// materialesSuperficie devuelve los seis proveedores de la superficie en
// el orden de accionesCorreosUsuarios.
func (d *dependenciasCorreosUsuariosDesarrollo) materialesSuperficie(superficie core.SuperficieAutenticacionActorV1) ([6]*proveedorMaterialAltaContratacionTemporalDesarrollo, bool) {
	var r [6]*proveedorMaterialAltaContratacionTemporalDesarrollo
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
	copy(r[:], d.materiales.lote[inicio:inicio+len(r)])
	return r, true
}

func rutaCorreosSuperficie(superficie core.SuperficieAutenticacionActorV1) string {
	switch superficie {
	case core.SuperficieAutenticacionInternaCorporativaV1:
		return usuarioshttp.RutaMisCorreos
	case core.SuperficieAutenticacionExternaPersonalV1:
		return usuarioshttp.RutaMisCorreosAreaPersonal
	}
	return ""
}

// montarCorreosUsuariosSuperficie crea la autoridad hermana de preferencias
// para la ruta de correos: misma identidad, pools y registrador de frontera;
// su propio manejador, proveedor V3 y ruta exacta. No posee pools.
func montarCorreosUsuariosSuperficie(ctx context.Context, preferencias *autoridadPreferenciasUsuariosDesarrollo, ejecutor *pgxpool.Pool,
	autorizador vecports.AutorizadorSolicitudLigadaV3, c configuracionUsuariosPreferenciasDesarrollo, d *dependenciasCorreosUsuariosDesarrollo,
	intentos *registroIntentosConsultaCorreos,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	if ctx == nil || preferencias == nil || ejecutor == nil || autorizador == nil || d == nil || d.cripto == nil || d.transporte == nil {
		return nil, errComposicionUsuariosCorreos
	}
	if preferencias.superficie == core.SuperficieAutenticacionInternaCorporativaV1 && intentos == nil ||
		preferencias.superficie != core.SuperficieAutenticacionInternaCorporativaV1 && intentos != nil {
		return nil, errComposicionUsuariosCorreos
	}
	ruta := rutaCorreosSuperficie(preferencias.superficie)
	materiales, ok := d.materialesSuperficie(preferencias.superficie)
	if ruta == "" || !ok {
		return nil, errComposicionUsuariosCorreos
	}
	registro, err := usuariospg.NuevoRegistroCorreosPostgreSQL(ctx, ejecutor, d.cripto, preferencias.superficie)
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	emisores := make(map[string]emisorPreferenciasUsuarios, len(materiales))
	for i, material := range materiales {
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, material)
		if err != nil {
			return nil, errComposicionUsuariosCorreos
		}
		emisores[accionesCorreosUsuarios[i].accion] = emisor
	}
	servicio, err := usuariosapp.NuevoServicioCorreos(usuariosapp.DependenciasCorreos{
		Registro: registro, Protector: d.cripto, Sellador: d.cripto, Desafios: d.cripto, Validador: d.cripto,
		Transporte: d.transporte, AhoraUTC: time.Now,
	})
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	correos := *preferencias
	correos.ruta, correos.cerrar, correos.manejador, correos.proveedor, correos.correos = ruta, nil, nil, nil, nil
	correos.intentosConsultaCorreos = intentos
	correos.proveedorCorreos = &proveedorCorreosUsuarios{autoridad: &correos, emisores: emisores,
		motivoConsulta: c.MotivoConsulta, motivoActualizacion: c.MotivoActualizacion}
	var manejador *usuarioshttp.ManejadorCorreos
	if intentos != nil {
		manejador, err = usuarioshttp.NuevoManejadorConsultaCorreosInternaConIntentos(servicio, &correos, &correos, &correos)
	} else {
		manejador, err = usuarioshttp.NuevoManejadorCorreosEnRuta(servicio, &correos, &correos, ruta)
	}
	if err != nil {
		return nil, errComposicionUsuariosCorreos
	}
	correos.manejador = manejador
	return &correos, nil
}

// ResolverOrdenCorreos sólo acepta el contexto que esta misma autoridad fijó
// en la frontera para la petición en curso.
func (a *autoridadPreferenciasUsuariosDesarrollo) ResolverOrdenCorreos(ctx context.Context) (usuariosports.OrdenCorreos, error) {
	if a == nil || ctx == nil || a.proveedorCorreos == nil {
		return usuariosports.OrdenCorreos{}, usuariosports.ErrCorreosNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
		return usuariosports.OrdenCorreos{}, usuariosports.ErrCorreosNoAutenticado
	}
	return usuariosapp.NuevaOrdenCorreos(c.resultado.Contexto, c.vinculo, a.superficie, a.proveedorCorreos)
}

// descifradorPreflightCorreos sólo sirve para la sonda de ACL: nunca descifra.
type descifradorPreflightCorreos struct{}

func (descifradorPreflightCorreos) ConDireccionCorreoDescifrada(context.Context, string, usuariosports.SobreDireccionCorreo, func([]byte) error) error {
	return errComposicionUsuariosCorreos
}

// La sonda comprueba, con el mismo LOGIN ejecutor de cada superficie, que las
// funciones de 000004 y la frontera de 000005 existen antes de publicar las
// doce audiencias V3.
const sondaFronteraCorreosSQL = `SELECT count(*)=1 FROM pg_catalog.pg_constraint
 WHERE conname='denegacion_frontera_preferencias_ruta_check'
 AND conrelid=pg_catalog.to_regclass('vec_usuarios.denegacion_frontera_preferencias')
 AND position('/api/vec/usuarios/area-personal/mis-correos' IN pg_catalog.pg_get_constraintdef(oid))>0`

func preflightSQLCorreosUsuariosDesarrollo(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancel()
	for _, superficie := range superficiesUsuariosEnProceso(cfg) {
		c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, superficie)
		if err != nil {
			return errComposicionUsuariosCorreos
		}
		pool, _, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(superficie)))
		if err != nil {
			return errComposicionUsuariosCorreos
		}
		_, errRegistro := usuariospg.NuevoRegistroCorreosPostgreSQL(ctx, pool, descifradorPreflightCorreos{}, superficie)
		var frontera bool
		errFrontera := pool.QueryRow(ctx, sondaFronteraCorreosSQL).Scan(&frontera)
		pool.Close()
		if errRegistro != nil || errFrontera != nil || !frontera {
			return errComposicionUsuariosCorreos
		}
	}
	return nil
}

func dominioRemitente(remitente string) string {
	_, dominio, ok := strings.Cut(strings.TrimSpace(remitente), "@")
	dominio = strings.TrimSuffix(dominio, ">")
	if !ok || dominio == "" || strings.ContainsAny(dominio, " <>@\r\n") {
		return ""
	}
	return dominio
}
