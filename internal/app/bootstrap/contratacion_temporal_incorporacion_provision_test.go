package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

type sesionNominalIncorporacionPrueba struct {
	contexto ct.ContextoAutorizacionAltaV3
	err      error
	llamadas int
}

func (s *sesionNominalIncorporacionPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	s.llamadas++
	return contextoSeguridadComunDesarrollo{Vinculo: s.contexto.Vinculo, Resultado: s.contexto.Resultado}, s.err
}

func escenarioNominalIncorporacion(t *testing.T) (*autoridadOperacionesIncorporacionV2, context.Context, core.DatosSolicitudAutorizacionLigadaV3, *publicadorPermisoIncorporacionPrueba) {
	t.Helper()
	a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
	nominales, err := nuevosPerfilesNominalesIncorporacion(a.soporte, a.consultas, a.referencias, []string{"centro:ejercicio:0001", "centro:ejercicio:0002"}, a.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	p.publicadas = map[string]instantaneaPublicadaDesarrollo{}
	for _, perfil := range []*perfilFijoCTDesarrollo{nominales.detalle, nominales.alta, nominales.ct} {
		perfil.contextoEsperadoRegistrado = perfil.contexto.Resultado
		perfil.sesionOperativa = &sesionNominalIncorporacionPrueba{contexto: perfil.contexto}
		p.publicadas[perfil.perfilRef()] = instantaneaPublicadaDesarrollo{instantanea: perfil.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	}
	a.nominales = nominales
	a.referencias.PerfilV3Ref = nominales.ct.perfilRef()
	ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
	return a, ctx, d, p
}

func TestIncorporacionV2TresPerfilesExactosConCuatroAcciones(t *testing.T) {
	a, ctx, base, p := escenarioNominalIncorporacion(t)
	sesiones := map[string]bool{}
	perfiles := map[string]bool{}
	for _, perfil := range []*perfilFijoCTDesarrollo{a.nominales.detalle, a.nominales.alta, a.nominales.ct} {
		v, err := perfil.contexto.Vinculo.Datos()
		if err != nil {
			t.Fatal(err)
		}
		if sesiones[v.SesionRef] || perfiles[v.PerfilActivoRef] {
			t.Fatal("sesión o perfil compartido entre geometrías")
		}
		sesiones[v.SesionRef], perfiles[v.PerfilActivoRef] = true, true
	}
	for _, operacion := range []string{"detalle", "alta", "ct", "lectura", "alta", "detalle", "lectura"} {
		d := datosOperacionPermisoIncorporacionPrueba(t, a, base, operacion)
		perfil := a.nominales.ct
		if operacion == "alta" {
			perfil = a.nominales.alta
		}
		if operacion == "detalle" {
			perfil = a.nominales.detalle
		}
		d.VinculoAutenticacionActor = perfil.contexto.Vinculo
		solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, perfil.contexto.Resultado)
		if !errors.Is(err, errPDPIncorporacionPrueba) {
			t.Fatalf("%s: %v", operacion, err)
		}
		if !perfil.plantilla.AsignacionPerfil.Cubre(d.Recurso) {
			t.Fatalf("%s no cubierto por su geometría fija", operacion)
		}
		for _, otro := range []*perfilFijoCTDesarrollo{a.nominales.detalle, a.nominales.alta, a.nominales.ct} {
			if otro != perfil && otro.plantilla.AsignacionPerfil.Cubre(d.Recurso) {
				t.Fatal("una geometría cubre otra")
			}
		}
	}
	if !reflect.DeepEqual(p.orden, []string{"pdp", "pdp", "pdp", "pdp", "pdp", "pdp", "pdp"}) {
		t.Fatal("publicó gobierno durante la petición", p.orden)
	}
	for _, perfil := range []*perfilFijoCTDesarrollo{a.nominales.detalle, a.nominales.alta, a.nominales.ct} {
		if perfil.sesionOperativa.(*sesionNominalIncorporacionPrueba).llamadas != 1 {
			t.Fatal("no conservó captura nominal única")
		}
		if !reflect.DeepEqual(p.publicadas[perfil.perfilRef()].instantanea, perfil.plantilla) {
			t.Fatal("alteró asignación publicada")
		}
	}
}

func TestIncorporacionV2NominalRevocacionYCausaDependencia(t *testing.T) {
	for _, caso := range []string{"revocada", "rol_retirado", "ambitos_restringidos", "acto_ajeno", "lector_caido", "sesion_caida", "perfil_cruzado"} {
		t.Run(caso, func(t *testing.T) {
			a, ctx, d, p := escenarioNominalIncorporacion(t)
			d = datosOperacionPermisoIncorporacionPrueba(t, a, d, "ct")
			perfil := a.nominales.ct
			d.VinculoAutenticacionActor = perfil.contexto.Vinculo
			publicada := p.publicadas[perfil.perfilRef()]
			publicada.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
			causa := errors.New("dependencia nominal de prueba no disponible")
			switch caso {
			case "revocada":
				publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
			case "rol_retirado":
				publicada.instantanea.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
			case "ambitos_restringidos":
				publicada.instantanea.AsignacionPerfil.Ambitos[1].Valores = []string{"unidad:restringida"}
			case "acto_ajeno":
				publicada.actoAsignacion = "acto:ajeno"
			case "lector_caido":
				p.errorLectura = causa
			case "sesion_caida":
				perfil.sesionOperativa.(*sesionNominalIncorporacionPrueba).err = causa
			case "perfil_cruzado":
				d.VinculoAutenticacionActor = a.nominales.alta.contexto.Vinculo
			}
			p.publicadas[perfil.perfilRef()] = publicada
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				t.Fatal(err)
			}
			resultado := perfil.contexto.Resultado
			if caso == "perfil_cruzado" {
				resultado = a.nominales.alta.contexto.Resultado
			}
			_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
			if caso == "lector_caido" {
				if !errors.Is(err, ct.ErrConsultaRRHHNoDisponible) || !errors.Is(err, causa) {
					t.Fatal("perdió causa de dependencia", err)
				}
			} else if caso == "sesion_caida" {
				if !errors.Is(err, causa) {
					t.Fatal("perdió causa de sesión", err)
				}
			} else if !errors.Is(err, ct.ErrAutorizacionDenegada) {
				t.Fatal("no denegó cruce/revocación", err)
			}
			if len(p.orden) != 0 {
				t.Fatal("negativa alcanzó publicación/PDP", p.orden)
			}
		})
	}
}

func TestIncorporacionV2PlantillasRechazanCentrosAmbiguos(t *testing.T) {
	a, _, _, _ := escenarioPermisoIncorporacionPrueba(t)
	for _, centros := range [][]string{nil, {"*"}, {"centro:uno", "centro:uno"}, {""}, {"centro:uno", strings.Repeat("a", 513)}} {
		if _, err := nuevoPerfilNominalIncorporacion(a.soporte, a.referencias, claveIncorporacionAlta, centros, a.reloj.Ahora()); err == nil {
			t.Fatal("admitió lista de centros ambigua")
		}
	}
}

func TestIncorporacionV2FronteraNominalUnicaConPerfilesDeterministas(t *testing.T) {
	a, _, _, _ := escenarioNominalIncorporacion(t)
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(a.soporte.contexto.Resultado.Contexto.PerfilActivoRef, []string{a.soporte.contexto.Resultado.Contexto.PerfilActivoRef})
	originales := make(map[string][]string, len(declaraciones))
	for _, d := range declaraciones {
		originales[d.Clave] = append([]string(nil), d.PerfilesActivosRef...)
	}
	resultado, err := asignarPerfilesNominalesIncorporacionEnFronteras(a.soporte, declaraciones)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(resultado)
	if err != nil {
		t.Fatal(err)
	}
	detalle, ok := catalogo.resolver("POST", httpinterno.RutaConsultaDetalleRRHH)
	if !ok {
		t.Fatal("sin nodo nominal de detalle")
	}
	for _, perfil := range []*perfilFijoCTDesarrollo{a.nominales.detalle, a.nominales.alta, a.nominales.ct} {
		if !detalle.admitePerfil(perfil.perfilRef()) {
			t.Fatal("perfil no declarado antes del catálogo")
		}
	}
	for _, d := range resultado {
		if d.Clave != detalle.Clave && !reflect.DeepEqual(d.PerfilesActivosRef, originales[d.Clave]) {
			t.Fatal("amplió otra frontera")
		}
	}
	for _, d := range declaraciones {
		if !reflect.DeepEqual(d.PerfilesActivosRef, originales[d.Clave]) {
			t.Fatal("modificó catálogo de entrada")
		}
	}
	if _, err := asignarPerfilesNominalesIncorporacionEnFronteras(a.soporte, resultado); err == nil {
		t.Fatal("admitió composición duplicada")
	}
}

func TestIncorporacionV2FuenteCapturaDosSesionesPropiasDelMismoCanal(t *testing.T) {
	a, ctx, _, _ := escenarioNominalIncorporacion(t)
	f := &fuenteAutoridadIncorporacionV2Desarrollo{soporte: a.soporte, consultas: a.consultas, motivoAlta: a.motivoAlta, motivoLectura: a.motivoLectura, referencias: a.referencias, nominales: a.nominales}
	p, err := f.PeticionVerificada(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if p.AltaNominal == nil || p.Contexto.PerfilActivoRef != a.nominales.ct.perfilRef() || p.AltaNominal.Contexto.PerfilActivoRef != a.nominales.alta.perfilRef() || p.PerfilesNominales.AltaRef != p.AltaNominal.Contexto.PerfilActivoRef || p.PerfilesNominales.ConsultaConfirmacionRef != p.Contexto.PerfilActivoRef {
		t.Fatal("captura de perfiles no es la composición fija")
	}
	if p.Autenticacion.SesionRef == p.AltaNominal.Autenticacion.SesionRef || p.Contexto.Cuenta != p.AltaNominal.Contexto.Cuenta {
		t.Fatal("no conserva sesiones propias y misma cuenta")
	}
	if p.PreparacionCT.ActorRef != a.referencias.ActorRef || p.PreparacionCT.OrganizacionRef != a.referencias.OrganizacionRef {
		t.Fatal("alteró referencias propietarias")
	}
}

func TestIncorporacionV2SesionNominalClasificaSinPerderCausa(t *testing.T) {
	causa := errors.New("resolutor temporalmente no disponible")
	if err := errorSesionNominalIncorporacion(context.Background(), causa); !errors.Is(err, ct.ErrConsultaRRHHNoDisponible) || !errors.Is(err, causa) {
		t.Fatal("dependencia degradada a denegación")
	}
	if err := errorSesionNominalIncorporacion(context.Background(), core.ErrAutenticacionRevalidadaInvalida); !errors.Is(err, ct.ErrAutorizacionDenegada) || !errors.Is(err, core.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("revocación perdió causa o clasificación")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := errorSesionNominalIncorporacion(ctx, causa); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdió precedencia")
	}
}

type registroSesionNominalCaidoPrueba struct {
	httpseguridad.RegistroSesiones
	err error
}

func (r registroSesionNominalCaidoPrueba) ConsumirAsercionYRegistrar(context.Context, httpseguridad.AltaSesionAtomica) (httpseguridad.ConfirmacionAltaSesion, error) {
	return httpseguridad.ConfirmacionAltaSesion{}, r.err
}

type resolutorNominalCaidoPrueba struct{ err error }

func (r resolutorNominalCaidoPrueba) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return core.ResultadoContextoActorRegistradoV2{}, r.err
}

func TestIncorporacionV2SesionNominalReutilizaCadenaYErrores(t *testing.T) {
	for _, caso := range []string{"valida", "revocada", "registro_caido", "resolutor_caido"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevaSesionConsultaPrueba(t)
			v, _ := e.soporte.contexto.Vinculo.Datos()
			refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "unidad:desarrollo:rrhh", ActorRef: v.PrincipalID}
			perfil, err := nuevoPerfilNominalIncorporacion(e.soporte, refs, claveIncorporacionCT, nil, e.reloj.Ahora())
			if err != nil {
				t.Fatal(err)
			}
			declaraciones, err := asignarPerfilesNominalesIncorporacionEnFronteras(e.soporte, descriptoresFronterasContratacionTemporalDesarrollo(v.PerfilActivoRef, []string{v.PerfilActivoRef}))
			if err != nil {
				t.Fatal(err)
			}
			catalogo, err := nuevoCatalogoFronterasComunDesarrollo(declaraciones)
			if err != nil {
				t.Fatal(err)
			}
			e.p.fronteras = catalogo
			fija, err := nuevaSesionReincorporacionTitularDesarrollo(e.p, perfil.contexto.Resultado)
			if err != nil {
				t.Fatal(err)
			}
			proveedor := fija.(*proveedorSesionConsultaRRHHDesarrollo)
			e.resolutor.base = perfil.contexto.Resultado
			causa := errors.New("dependencia nominal de prueba caída")
			switch caso {
			case "revocada":
				e.revalidador.err = core.ErrAutenticacionRevalidadaInvalida
			case "registro_caido":
				proveedor.registro = registroSesionNominalCaidoPrueba{RegistroSesiones: e.registro, err: causa}
			case "resolutor_caido":
				proveedor.resolutor = resolutorNominalCaidoPrueba{err: causa}
			}
			ctx := e.contexto()
			canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			canal.ruta = httpinterno.RutaIncorporacionEjercicioV2
			ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
			ctx, err = contextoDetalleIncorporacionV2Desarrollo(ctx, e.soporte, catalogo)
			if err != nil {
				t.Fatal(err)
			}
			s := &sesionNominalIncorporacion{proveedor: proveedor}
			resultado, err := s.ResolverContexto(ctx)
			switch caso {
			case "valida":
				if err != nil || resultado.Resultado.Contexto.PerfilActivoRef != perfil.perfilRef() || resultado.Vinculo.CoincideExactamenteCon(perfil.contexto.Vinculo) || len(e.registro.altas) != 1 || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
					t.Fatal("no reutilizó cadena con captura nominal fresca", err)
				}
			case "revocada":
				if !errors.Is(err, ct.ErrAutorizacionDenegada) || !errors.Is(err, core.ErrAutenticacionRevalidadaInvalida) {
					t.Fatal("revocación mal clasificada", err)
				}
			default:
				if !errors.Is(err, ct.ErrConsultaRRHHNoDisponible) || !errors.Is(err, causa) {
					t.Fatal("caída perdió su causa", err)
				}
			}
		})
	}
}
