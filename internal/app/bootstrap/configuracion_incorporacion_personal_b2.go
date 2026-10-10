package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"vec-diputacion-granada/config"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

// La configuración describe fuentes y material ya gobernados. No contiene
// personas, planes por expediente, permisos pedidos desde HTTP ni reglas jurídicas.
type archivoIncorporacionPersonalB2 struct {
	Protocolo     string                                     `json:"protocolo"`
	OrganismoRef  string                                     `json:"organismo_ref"`
	CatalogoRPTID string                                     `json:"catalogo_rpt_id"`
	ModuloRPTID   string                                     `json:"modulo_rpt_id"`
	Pools         map[string]string                          `json:"dsn_files"`
	Operaciones   map[string]archivoOperacionIncorporacionB2 `json:"operaciones"`
	// CeseFechaEfecto decide cómo la fecha de efectos del cese CT termina la
	// relación en Personal (duda 140, pendiente de RRHH). Sin valor no se
	// compone el fin en Personal; un valor desconocido impide arrancar.
	CeseFechaEfecto string `json:"cese_fecha_efecto,omitempty"`
}
type archivoOperacionIncorporacionB2 struct {
	Motivo    core.ReferenciaEntradaCatalogo  `json:"motivo"`
	Capacidad archivoCapacidadIncorporacionV2 `json:"capacidad"`
}
type descriptorOperacionIncorporacionB2 struct {
	clave, accion, audiencia, modulo, tipo, finalidad string
}

func operacionesIncorporacionB2() []descriptorOperacionIncorporacionB2 {
	return []descriptorOperacionIncorporacionB2{
		{"bolsa_anclaje", bolsa.AccionConsultaAnclajeAceptacionCT, bolsa.AudienciaConsultaAnclajeAceptacionCT, "bolsa", bolsa.TipoRecursoAnclajeAceptacionCT, bolsa.FinalidadConsultaAnclajeAceptacionCT},
		{"personal_clases", "personal.plan_incorporacion_ct.clases_ocupacion", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"ct_detalle", ct.AccionConsultarDetalleRRHH, ct.AudienciaConsumoConsultaDetalleRRHHV3, ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadConsultarDetalleRRHH},
		{"bolsa_persona", bolsa.AccionConsultaPersonaAceptacionCT, bolsa.AudienciaConsultaPersonaAceptacionCT, "bolsa", bolsa.TipoRecursoPersonaAceptacionCT, bolsa.FinalidadConsultaPersonaAceptacionCT},
		{"ct_vinculo_consultar", ct.AccionConsultarVinculoCategoriaRPT, ct.AudienciaConsultarVinculoCategoriaRPT, ct.ModuloContratacion, "vinculo_categoria_rpt_ct", finalidadVinculoCategoriaRPTCT},
		{"rpt_publicacion", "vec.catalogos.categorias.consultar_historica", ct.AudienciaConsultarPublicacionCategoriaRPT, "rpt", "catalogo_configurable", finalidadLecturaCategoriaRPT},
		{"rpt_uso_consultar", "vec.catalogos.categorias.consultar_uso", "vec_catalogos_configurables.lectura_categorias.v1", "rpt", "uso_categoria", "consultar_categorias_rpt"},
		{"rpt_reservar", "vec.catalogos.categorias.reservar_uso", "vec_catalogos_configurables.usos_categorias.v1", "rpt", "uso_categoria", "vincular_categoria_a_operacion"},
		{"rpt_confirmar", "vec.catalogos.categorias.confirmar_uso", "vec_catalogos_configurables.usos_categorias.v1", "rpt", "uso_categoria", "vincular_categoria_a_operacion"},

		{"ct_plan_preparar", "contratacion_temporal.incorporacion_personal.plan.registrar", "vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1", ct.ModuloContratacion, "incorporacion_personal_ct", "incorporar_personal_desde_ct"},
		{"ct_plan_consultar", "contratacion_temporal.incorporacion_personal.plan.consultar", "vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1", ct.ModuloContratacion, "incorporacion_personal_ct", "incorporar_personal_desde_ct"},
		{"ct_origen_confirmar", "contratacion_temporal.incorporacion_personal.origen.confirmar", "vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1", ct.ModuloContratacion, "incorporacion_personal_ct", "incorporar_personal_desde_ct"},
		{"personal_plan_preparar", "personal.plan_incorporacion_ct.preparar", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"personal_plan_consultar", "personal.plan_incorporacion_ct.consultar", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"personal_plan_ejecutar", "personal.plan_incorporacion_ct.ejecutar", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"personal_plan_seleccionar", "personal.plan_incorporacion_ct.seleccionar", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"personal_plan_confirmar", "personal.plan_incorporacion_ct.confirmar", personal.AudienciaPlanIncorporacionCT, "personal", "plan_incorporacion_ct", "gestionar_incorporacion_ct"},
		{"personal_vacantes", personal.AccionVacantesB2, personal.AudienciaVacantesB2, "personal", "vacantes_rrhh", "consultar_vacantes"},
		{"personal_catalogos", personal.AccionConsultarCatalogoEmpleadoB2, personal.AudienciaConsultarCatalogoEmpleadoB2, "personal", "catalogo_empleado_rrhh", "consultar_catalogo_empleado"},
		{"personal_alta", personal.AccionAltaEmpleadoB2, personal.AudienciaAltaEmpleadoB2, "personal", "alta_empleado_rrhh", "registrar_empleado"},
		{"personal_hecho", personal.AccionHechoEmpleadoB2, personal.AudienciaHechoEmpleadoB2, "personal", "hecho_empleado_rrhh", "registrar_hecho_empleado"},
		{"personal_ficha", personal.AccionFichaEmpleadoB2, personal.AudienciaFichaEmpleadoB2, "personal", "registro_empleado_rrhh", "consultar_ficha_empleado"},
		{claveRegistroVinculoRPTB2, ct.AccionRegistrarVinculoCategoriaRPT, ct.AudienciaRegistrarVinculoCategoriaRPT, ct.ModuloContratacion, "vinculo_categoria_rpt_ct", finalidadVinculoCategoriaRPTCT},
	}
}

// claveRegistroVinculoRPTB2 es la única operación B2 opcional: sin su entrada
// en la configuración privada no se crea su perfil nominal ni se monta su ruta.
const claveRegistroVinculoRPTB2 = "ct_vinculo_registrar"

// ClaveRegistroVinculoRPTB2 la usa la herramienta de preparación del material.
const ClaveRegistroVinculoRPTB2 = claveRegistroVinculoRPTB2

// operacionConfiguradaB2 dice si la configuración privada habilita la operación.
func operacionConfiguradaB2(c *archivoIncorporacionPersonalB2, clave string) bool {
	if clave != claveRegistroVinculoRPTB2 {
		return true
	}
	if c == nil {
		return false
	}
	_, ok := c.Operaciones[clave]
	return ok
}
func operacionesConfiguradasB2(c *archivoIncorporacionPersonalB2) []descriptorOperacionIncorporacionB2 {
	r := []descriptorOperacionIncorporacionB2{}
	for _, d := range operacionesIncorporacionB2() {
		if operacionConfiguradaB2(c, d.clave) {
			r = append(r, d)
		}
	}
	return r
}
func descriptorIncorporacionB2(accion string) (descriptorOperacionIncorporacionB2, bool) {
	for _, d := range operacionesIncorporacionB2() {
		if d.accion == accion {
			return d, true
		}
	}
	return descriptorOperacionIncorporacionB2{}, false
}
func descriptoresMaterialIncorporacionB2() []descriptorMaterialConsumidorV3Desarrollo {
	resultado := []descriptorMaterialConsumidorV3Desarrollo{}
	operaciones := operacionesIncorporacionB2()
	for _, d := range operaciones {
		if (d.modulo == "personal" && d.audiencia != personal.AudienciaPlanIncorporacionCT) || (d.modulo == "bolsa" && d.accion != bolsa.AccionConsultaAnclajeAceptacionCT) || d.modulo == "rpt" || d.accion == ct.AccionConsultarVinculoCategoriaRPT || d.accion == ct.AccionRegistrarVinculoCategoriaRPT || d.accion == ct.AccionConsultarDetalleRRHH {
			continue
		}
		existe := false
		for _, r := range resultado {
			existe = existe || r.Audiencia == d.audiencia
		}
		if existe {
			continue
		}
		resultado = append(resultado, descriptorMaterialConsumidorV3Desarrollo{Audiencia: d.audiencia, Dominio: "vec.incorporacion-b2." + d.clave + ".capacidad-v3", Prefijo: "clave:capacidad:incorporacion-b2-" + d.clave + ":", ProveedorNominal: "proveedor-material-incorporacion-b2-" + d.clave})
	}
	// Las cinco audiencias adicionales comparten el mismo publicador V3.
	// Mantener primero las cinco existentes conserva sus dominios y orden;
	// el registro del vínculo CT154 va el último por el mismo motivo.
	for _, clave := range [...]string{"bolsa_persona", "ct_vinculo_consultar", "rpt_publicacion", "rpt_reservar", claveRegistroVinculoRPTB2} {
		for _, d := range operaciones {
			if d.clave == clave {
				resultado = append(resultado, descriptorMaterialConsumidorV3Desarrollo{Audiencia: d.audiencia, Dominio: "vec.incorporacion-b2." + d.clave + ".capacidad-v3", Prefijo: "clave:capacidad:incorporacion-b2-" + d.clave + ":", ProveedorNominal: "proveedor-material-incorporacion-b2-" + d.clave})
				break
			}
		}
	}
	return resultado
}
func cargarEmisorOperacionIncorporacionB2(raiz *os.Root, c archivoCapacidadIncorporacionV2, audiencia string, reloj ct.Reloj) (*confianza.EmisorCapacidadesAtestacionAutorizacionV3, error) {
	secreto, e := leerArchivoIncorporacionV2(raiz, c.File, 256)
	if e != nil {
		return nil, e
	}
	defer borrarBytes(secreto)
	h := sha256.Sum256(secreto)
	if hex.EncodeToString(h[:]) != c.SHA256 {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	k, e := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(c.ClaveID, c.Version, secreto, c.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, c.Desde, c.Hasta, time.Time{}, c.RevisionGobierno, c.HuellaGobierno)
	if e != nil || reloj.Ahora().Before(c.Desde) || !reloj.Ahora().Before(c.Hasta) {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(k, reloj)
}

var rolesPoolsIncorporacionB2 = map[string]string{
	"personal_planes":    "vec_personal_ejecutor",
	"personal_actos":     "vec_personal_ejecutor",
	"personal_consultas": "vec_personal_ejecutor",
	"bolsa_persona":      "vec_bolsa_llamamientos_ejecutor",
	"rpt":                "vec_personal_ejecutor",
}

func validarConfiguracionIncorporacionB2(c *archivoIncorporacionPersonalB2) error {
	f := ct.ErrComposicionIncorporacionAplicacion
	if c == nil || c.Protocolo != "personal_b2_v1" || c.OrganismoRef == "" || c.CatalogoRPTID == "" || c.ModuloRPTID == "" || len(c.Pools) != len(rolesPoolsIncorporacionB2) || len(c.Operaciones) != len(operacionesConfiguradasB2(c)) ||
		(c.CeseFechaEfecto != "" && !inc.ReglaFechaCesePersonalB2(c.CeseFechaEfecto).Valida()) {
		return f
	}
	for k := range rolesPoolsIncorporacionB2 {
		if c.Pools[k] == "" {
			return f
		}
	}
	for _, d := range operacionesConfiguradasB2(c) {
		op, ok := c.Operaciones[d.clave]
		if !ok || !core.ReferenciaMotivoAutorizacionV2Valida(op.Motivo) || op.Capacidad.File == "" {
			return f
		}
	}
	return nil
}
func cargarPoolsIncorporacionB2(ctx context.Context, raiz *os.Root, c *archivoIncorporacionPersonalB2, existentes map[string]*pgxpool.Pool) (map[string]*pgxpool.Pool, func(), error) {
	if e := validarConfiguracionIncorporacionB2(c); e != nil {
		return nil, nil, e
	}
	pools := map[string]*pgxpool.Pool{}
	var once sync.Once
	cerrar := func() {
		once.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	usuarios := map[string]bool{}
	for _, p := range existentes {
		if p != nil {
			usuarios[p.Config().ConnConfig.User] = true
		}
	}
	for nombre, rol := range rolesPoolsIncorporacionB2 {
		b, e := leerArchivoIncorporacionV2(raiz, c.Pools[nombre], 16<<10)
		if e != nil {
			cerrar()
			return nil, nil, e
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		pc, e := pgxpool.ParseConfig(dsn)
		if e != nil || pc.ConnConfig.User == "" || usuarios[pc.ConnConfig.User] {
			cerrar()
			return nil, nil, ct.ErrComposicionIncorporacionAplicacion
		}
		usuarios[pc.ConnConfig.User] = true
		p, _, e := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-incorporacion-b2-"+nombre, rol)
		if e != nil {
			cerrar()
			return nil, nil, e
		}
		pools[nombre] = p
	}
	return pools, cerrar, nil
}
func cargarAutoridadIncorporacionB2(raiz *os.Root, c *archivoIncorporacionPersonalB2, p *perfilesNominalesIncorporacion, pools map[string]*pgxpool.Pool, registroPool *pgxpool.Pool, material *proveedorMaterialAltaContratacionTemporalDesarrollo, reloj ct.Reloj) (*autoridadIncorporacionPersonalB2, error) {
	if validarConfiguracionIncorporacionB2(c) != nil || p == nil || len(p.b2) != len(operacionesConfiguradasB2(c)) || material == nil || registroPool == nil {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	fuente, e := pgvec.NuevoAlmacenAutorizacion(pools["fuente_autorizacion"])
	if e != nil {
		return nil, e
	}
	registro, e := pgvec.NuevoAlmacenAutorizacion(registroPool)
	if e != nil {
		return nil, e
	}
	a := &autoridadIncorporacionPersonalB2{perfiles: p, operaciones: map[string]operacionAutorizadaIncorporacionB2{}, material: material, organismoRef: c.OrganismoRef, reloj: reloj}
	for _, d := range operacionesConfiguradasB2(c) {
		op := c.Operaciones[d.clave]
		if d.modulo == "rpt" {
			d.modulo = c.ModuloRPTID
		}
		motivos, e := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools["motivos_autorizacion"], op.Motivo.CatalogoID)
		if e != nil {
			return nil, e
		}
		pdp, e := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivos, reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
		if e != nil {
			return nil, e
		}
		emisor, e := cargarEmisorOperacionIncorporacionB2(raiz, op.Capacidad, d.audiencia, reloj)
		if e != nil {
			return nil, e
		}
		a.operaciones[d.accion] = operacionAutorizadaIncorporacionB2{descriptor: d, motivo: op.Motivo, pdp: pdp, emisor: emisor}
	}
	return a, nil
}

var rolesPoolsIncorporacionB2Pura = map[string]string{"fuente_autorizacion": "vec_autorizacion_fuente", "motivos_autorizacion": "vec_autorizacion_motivos_evaluador", "registro_ct": "vec_contratacion_temporal_ejecutor"}

func rolesPoolsConfiguracionIncorporacion(c archivoIncorporacionV2) map[string]string {
	if c.Planes == "" {
		return rolesPoolsIncorporacionB2Pura
	}
	return rolesPoolsIncorporacionV2
}
func poolsConfiguracionIncorporacionValidos(c archivoIncorporacionV2) bool {
	if c.PersonalB2 != nil && validarConfiguracionIncorporacionB2(c.PersonalB2) != nil {
		return false
	}
	if c.Planes == "" && (c.Personal != "" || c.Continuidad != nil || c.PersonalB2 == nil) {
		return false
	}
	return len(c.Pools) == len(rolesPoolsConfiguracionIncorporacion(c))
}
func cargarIncorporacionB2Pura(cfg config.Config, c archivoIncorporacionV2, raiz *os.Root, alta *dependenciasAltaContratacionTemporalDesarrollo, consultas dependenciasConsultasRRHHDesarrollo, reloj relojContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) (ConfiguracionIncorporacionDesarrollo, func(), error) {
	vacia := ConfiguracionIncorporacionDesarrollo{}
	if c.Planes != "" || c.Personal != "" || c.Continuidad != nil || validarConfiguracionIncorporacionB2(c.PersonalB2) != nil {
		return vacia, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(60*time.Second))
	defer cancelar()
	pools := map[string]*pgxpool.Pool{}
	var montaje *montajeIncorporacionPersonalB2
	var once sync.Once
	cerrar := func() {
		once.Do(func() {
			if montaje != nil {
				montaje.cerrar()
			}
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, p := range []*pgxpool.Pool{alta.postgresql.ejecucion, alta.postgresql.bolsa, alta.postgresql.gobierno, alta.postgresql.registroAutorizacion, alta.postgresql.confirmador} {
		if p != nil {
			usuarios[p.Config().ConnConfig.User] = true
		}
	}
	for nombre, rol := range rolesPoolsIncorporacionB2Pura {
		b, e := leerArchivoIncorporacionV2(raiz, c.Pools[nombre], 16<<10)
		if e != nil {
			return vacia, nil, e
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		pc, e := pgxpool.ParseConfig(dsn)
		if e != nil || pc.ConnConfig.User == "" || usuarios[pc.ConnConfig.User] {
			return vacia, nil, ct.ErrComposicionIncorporacionAplicacion
		}
		usuarios[pc.ConnConfig.User] = true
		p, _, e := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-incorporacion-b2-"+nombre, rol)
		if e != nil {
			return vacia, nil, e
		}
		pools[nombre] = p
	}
	perfilDetalle, e := nuevoPerfilNominalIncorporacion(alta.soporte, c.Referencias, claveIncorporacionDetalle, nil, reloj.Ahora())
	if e != nil {
		return vacia, nil, e
	}
	nominales := &perfilesNominalesIncorporacion{soporte: alta.soporte, consultas: consultas.autoridad, detalle: perfilDetalle}
	if e = extenderPerfilesNominalesB2(nominales, c.Referencias, c.PersonalB2, reloj.Ahora()); e != nil {
		return vacia, nil, e
	}
	descriptor, ok := fronteras.resolver(http.MethodPost, httpct.RutaConsultaDetalleRRHH)
	if !ok || !esDescriptorDetalleContratacionTemporalDesarrollo(descriptor, alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef) {
		return vacia, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	for _, p := range nominales.todos() {
		if !descriptor.admitePerfil(p.perfilRef()) {
			return vacia, nil, ct.ErrComposicionIncorporacionAplicacion
		}
	}
	motivoDetalle, e := consultas.motivos.ResolverMotivoDetalleRRHH(ctx, reloj.Ahora())
	if e != nil {
		return vacia, nil, e
	}
	if c.PersonalB2.Operaciones["ct_detalle"].Motivo != motivoDetalle {
		return vacia, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	autoridad, e := cargarAutoridadIncorporacionB2(raiz, c.PersonalB2, nominales, pools, alta.postgresql.registroAutorizacion, consultas.materialDetalle, reloj)
	if e != nil {
		return vacia, nil, e
	}
	emisorDetalle, e := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, consultas.materialDetalle)
	if e != nil {
		return vacia, nil, e
	}
	emisor, e := ct.NuevoEmisorMaterialConsultaRRHH(consultas.motivos, seg.GeneradorReferenciasCriptograficas{}, reloj, consultas.emisorCuadro, emisorDetalle)
	if e != nil {
		return vacia, nil, e
	}
	detalle, e := appct.NuevoServicioConsultaDetalleRRHH(&contextoDetalleIncorporacionB2{autoridad: autoridad, org: c.Referencias.OrganizacionRef, reloj: reloj}, emisor, consultas.sesion, reloj)
	if e != nil {
		return vacia, nil, e
	}
	if e = provisionarPerfilesNominalesIncorporacion(ctx, alta.postgresql.gobierno, nominales, aprobacionProvisionPerfilesRRHHDesdeConfig(cfg)); e != nil {
		return vacia, nil, e
	}
	if e = configurarSesionesNominalesIncorporacion(ctx, nominales, consultas.identidad); e != nil {
		return vacia, nil, e
	}
	montaje, e = cargarMontajeIncorporacionPersonalB2(ctx, raiz, c.PersonalB2, nominales, pools, alta.postgresql.registroAutorizacion, consultas.materialDetalle, detalle, c.Referencias.OrganizacionRef, reloj)
	if e != nil {
		return vacia, nil, e
	}
	nominales.montajeB2 = montaje
	completa = true
	return ConfiguracionIncorporacionDesarrollo{nominales: nominales, fronteras: fronteras, detalleNominal: detalle, Referencias: c.Referencias}, cerrar, nil
}

type contextoDetalleIncorporacionB2 struct {
	autoridad *autoridadIncorporacionPersonalB2
	org       string
	reloj     ct.Reloj
}

func (a *contextoDetalleIncorporacionB2) ResolverContextoConsultaRRHH(ctx context.Context) (ct.ContextoConsultaRRHH, error) {
	c, e := a.autoridad.contexto(ctx, ct.AccionConsultarDetalleRRHH)
	if e != nil {
		return ct.ContextoConsultaRRHH{}, e
	}
	return ct.NuevoContextoConsultaRRHH(c, a.org, a.reloj.Ahora())
}
