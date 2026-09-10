package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

// ConfiguracionIncorporacionDesarrollo es opt-in del mismo servidor privado.
// El operador entrega fuentes selladas, pools propietarios ya comprobados y
// cadena nominal gobernada. No abre conexiones, instala esquemas, crea roles,
// publica capacidades ni transforma el rol técnico RRHH en permiso Personal.
// Autoridad/Detalle/Reloj de Preparacion se ligan aquí, no desde configuración.
type ConfiguracionIncorporacionDesarrollo struct {
	// Sólo la carga nominal de arranque compone este detalle con el PDP real.
	detalleNominal            *appct.ServicioConsultaDetalleRRHH
	Referencias               ReferenciasCTIncorporacionDesarrollo
	Preparacion               inc.ConfiguracionPreparacionDurableV2PostgreSQL
	Cadena                    *inc.CadenaAutorizacionAplicacion
	MotivoAlta, MotivoLectura core.ReferenciaEntradaCatalogo
	AltaPersonal, RegistroCT  *pgxpool.Pool
}

// Correspondencia explícita del servidor, ligada al actor/perfil nominales.
// No es una concesión ni deriva referencias CT de identificadores V3.
type ReferenciasCTIncorporacionDesarrollo struct {
	PrincipalV3Ref, PerfilV3Ref          string
	OrganizacionRef, UnidadRef, ActorRef string
}

func (r ReferenciasCTIncorporacionDesarrollo) valida() bool {
	if r.PrincipalV3Ref == "" || r.PerfilV3Ref == "" {
		return false
	}
	if !domct.UnidadSeguimientoValida(r.UnidadRef) || !domct.ActorSeguimientoValido(r.ActorRef) {
		return false
	}
	var refs []string
	// CT82 conserva la organización original: nunca normalizarla a un hash.
	if strings.HasPrefix(r.OrganizacionRef, "organizacion:") && len(r.OrganizacionRef) > len("organizacion:") && domct.ReferenciaOpacaValida(r.OrganizacionRef) {
		// La consulta y el PDP cotejan el ámbito exacto del expediente.
	} else {
		refs = append(refs, r.OrganizacionRef)
	}
	for _, ref := range refs {
		if len(ref) != 68 || !strings.HasPrefix(ref, "ref:") || strings.ToLower(ref) != ref || ref == "ref:"+strings.Repeat("0", 64) {
			return false
		}
		if _, err := hex.DecodeString(ref[4:]); err != nil {
			return false
		}
	}
	return true
}

type claveIncorporacionV2Desarrollo struct{}
type fuenteAutoridadIncorporacionV2Desarrollo struct {
	soporte                   *soporteAltaContratacionTemporalDesarrollo
	consultas                 *autoridadConsultasRRHHDesarrollo
	motivoAlta, motivoLectura core.ReferenciaEntradaCatalogo
	referencias               ReferenciasCTIncorporacionDesarrollo
}

func (f *fuenteAutoridadIncorporacionV2Desarrollo) PeticionVerificada(ctx context.Context) (inc.PeticionAutoridad, error) {
	var cero inc.PeticionAutoridad
	if ctx == nil || f == nil || f.soporte == nil || f.consultas == nil {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if ctx.Value(claveIncorporacionV2Desarrollo{}) != f.soporte.sello {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	c, err := f.consultas.contextoConsultaRRHHDesarrollo(ctx)
	if err != nil {
		return cero, err
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	if !f.referencias.valida() || f.referencias.PrincipalV3Ref != v.PrincipalID || f.referencias.PerfilV3Ref != v.PerfilActivoRef {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	// Correlación CT independiente, no alias de una correlación/autenticación V3.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	return inc.PeticionAutoridad{
		Autenticacion: core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef},
		Contexto:      core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: v.CuentaRef, Metodo: v.MetodoObservado, Garantia: v.GarantiaObservada}, PerfilActivoRef: v.PerfilActivoRef},
		PreparacionCT: ct.PreparacionSeguimientoConfirmacionIncorporacion{OrganizacionRef: f.referencias.OrganizacionRef, UnidadRef: f.referencias.UnidadRef, ActorRef: f.referencias.ActorRef, CorrelacionRef: "ref:" + hex.EncodeToString(nonce[:])},
		MotivoAlta:    f.motivoAlta, MotivoLectura: f.motivoLectura,
	}, nil
}

func nuevasDependenciasIncorporacionV2Desarrollo(c ConfiguracionIncorporacionDesarrollo, alta *dependenciasAltaContratacionTemporalDesarrollo,
	consultas dependenciasConsultasRRHHDesarrollo, reloj relojContratacionTemporalDesarrollo) (*inc.ServidorV2PostgreSQL, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	if !c.Referencias.valida() || alta == nil || alta.soporte == nil || consultas.identidad == nil || consultas.autoridad == nil ||
		c.Preparacion.Autoridad != nil || c.Preparacion.Detalle != nil || c.Preparacion.Reloj != nil {
		return nil, f
	}
	detalle, ok := consultas.detalle.(*appct.ServicioConsultaDetalleRRHH)
	if c.detalleNominal != nil {
		detalle, ok = c.detalleNominal, true
	}
	if !ok || detalle == nil {
		return nil, f
	}
	for _, m := range []core.ReferenciaEntradaCatalogo{c.MotivoAlta, c.MotivoLectura} {
		if _, err := core.HuellaSHA256MotivoAutorizacionV2(m); err != nil {
			return nil, f
		}
	}
	c.Preparacion.Detalle, c.Preparacion.Reloj = detalle, reloj
	return inc.NuevoServidorV2PostgreSQL(inc.ConfiguracionServidorV2PostgreSQL{
		FuenteAutoridad: &fuenteAutoridadIncorporacionV2Desarrollo{alta.soporte, consultas.autoridad, c.MotivoAlta, c.MotivoLectura, c.Referencias},
		Revalidador:     consultas.identidad.revalidador, Resolutor: consultas.identidad.resolutor, Cadena: c.Cadena,
		Correlador: seguridad.GeneradorReferenciasCriptograficas{}, Preparacion: c.Preparacion, AltaPersonal: c.AltaPersonal, RegistroCT: c.RegistroCT,
	})
}

// Hijo de la petición mTLS existente para la consulta RRHH. Conserva la misma
// identidad y ventana del certificado. El sello privado no concede permisos:
// el detalle y cada consumidor vuelven a exigir su autorización nominal.
func contextoDetalleIncorporacionV2Desarrollo(ctx context.Context, soporte *soporteAltaContratacionTemporalDesarrollo) (context.Context, error) {
	if ctx == nil || soporte == nil {
		return nil, ct.ErrDenegadaIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c, ok := soporte.capacidadValida(ctx)
	if !ok || (c.ruta != httpinterno.RutaIncorporacionEjercicioV2 && c.ruta != httpinterno.RutaFichaGINPIXV2) {
		return nil, ct.ErrDenegadaIncorporacionAplicacion
	}
	c.ruta = httpinterno.RutaConsultaDetalleRRHH
	c.consultaRRHH = &contextoConsultaRRHHPeticionDesarrollo{}
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	return context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, soporte.sello), nil
}

func ligarContextoIncorporacionV2Desarrollo(h http.Handler, soporte *soporteAltaContratacionTemporalDesarrollo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := contextoDetalleIncorporacionV2Desarrollo(r.Context(), soporte)
		if err == nil {
			r = r.WithContext(ctx)
		}
		// Sin sello, el mismo handler nominal devuelve denegación; no se crea
		// otra respuesta ni se omite la validación del contrato HTTP existente.
		h.ServeHTTP(w, r)
	})
}
