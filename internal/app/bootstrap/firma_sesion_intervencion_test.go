package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// sesionConsultaFirmasIntervencionPrueba reúne la persona común con sus dos
// perfiles sintéticos. La petición conserva el canal de Intervención y la
// sesión se emite para el perfil lector, ambos elegidos por composición.
func sesionConsultaFirmasIntervencionPrueba(t *testing.T) (*entornoSesionConsultaPrueba, *soporteFiscalizacionContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, *proveedorSesionConsultaRRHHDesarrollo) {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	ahora := e.reloj.Ahora()
	lector, err := nuevoPerfilFijoCTDesarrollo(e.principal, e.soporte.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaResultadosFiscalizacion},
		func(actor, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
			plantilla, err := instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, perfil, ahora)
			if err != nil {
				return dominiovec.InstantaneaAutorizacion{}, err
			}
			// La consulta interna no reutiliza la concesión de firmar. Conserva
			// solo la proyección de lectura que exige esta dependencia.
			plantilla.VersionRol.Concesiones = plantilla.VersionRol.Concesiones[1:]
			return plantilla, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	// El resolvedor simula la fuente de contexto del perfil lector; no se
	// reutiliza el resultado del perfil fiscalizador para emitir la sesión.
	e.resolutor.base = lector.contexto.Resultado
	canal := &soporteFiscalizacionContratacionTemporalDesarrollo{
		sello: e.soporte.sello, principalID: e.soporte.principalID,
		certificadoSHA256: e.soporte.certificadoSHA256, contexto: e.soporte.contexto,
	}
	p, err := nuevaSesionConsultaFirmasIntervencionCTDesarrollo(context.Background(), e.p, canal, e.soporte, lector)
	if err != nil {
		t.Fatal(err)
	}
	return e, canal, lector, p
}

func contextoConsultaFirmasIntervencionPrueba(t *testing.T, e *entornoSesionConsultaPrueba, p *proveedorSesionConsultaRRHHDesarrollo) context.Context {
	t.Helper()
	principal := clonarPrincipalDesarrollo(e.principal)
	principal.Roles = []string{rolIntervencionContratacionTemporalDesarrollo}
	ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, principal, httpinterno.RutaResultadosFiscalizacion)
	canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	canal.metodo = http.MethodPost
	canal.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
	canal.certificadoValidoHasta = e.reloj.Ahora().Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
	d, ok := p.fronteras.resolver(http.MethodPost, httpinterno.RutaResultadosFiscalizacion)
	if !ok {
		t.Fatal("falta la frontera de resultados de fiscalización")
	}
	return context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpinterno.RutaResultadosFiscalizacion,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: p.fronteras, descriptor: d,
	})
}

func TestSesionConsultaFirmasIntervencionConservaCanalYPerfilSeparados(t *testing.T) {
	e, canal, lector, p := sesionConsultaFirmasIntervencionPrueba(t)
	antesCanal, err := canal.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoConsultaFirmasIntervencionPrueba(t, e, p)
	ctxCapsula, capsula, err := p.acreditarPeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if capsula.evidencia.Ruta != httpinterno.RutaResultadosFiscalizacion ||
		capsula.evidencia.Alta.SujetoID != canal.principalID ||
		capsula.evidencia.CertificadoSHA256 != canal.certificadoSHA256 ||
		capsula.evidencia.PerfilRef != lector.perfilRef() {
		t.Fatal("la cápsula no conservó ruta, actor, certificado o perfil lector")
	}
	macOriginal := capsula.mac
	capsula.mac[0] ^= 0xff
	if _, _, err := p.registrarCapsula(ctxCapsula, capsula); !errors.Is(err, ports.ErrAutorizacionDenegada) || len(e.registro.altas) != 0 {
		t.Fatal("una cápsula con HMAC alterado alcanzó el registro")
	}
	capsula.mac = macOriginal
	if _, _, err := p.registrarCapsula(ctxCapsula, capsula); err != nil {
		t.Fatal(err)
	}
	if len(e.registro.altas) != 1 || e.registro.altas[0].SujetoID != canal.principalID ||
		e.registro.altas[0].CuentaID != "desarrollo:"+lector.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		!reflect.DeepEqual(antesCanal, canal.contexto.Resultado) {
		t.Fatal("la sesión alteró el holder de fiscalización o perdió su identidad")
	}
	concesion := lector.plantilla.VersionRol.Concesiones[0]
	if concesion.Accion != ports.AccionConsultarFirmasDocumento ||
		concesion.TipoRecurso != ports.TipoRecursoConsultaFirmasDocumento ||
		!slices.Equal(concesion.Finalidades, []string{ports.FinalidadFirmaDocumento}) ||
		!slices.Equal(concesion.CamposPermitidos, consultafirmas.CamposConsultaFirmasDocumento()) ||
		len(concesion.CamposPermitidos) != 18 {
		t.Fatal("el lector no tiene la plantilla cerrada de consulta de firmas")
	}
	resuelto, err := p.ResolverContexto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resuelto.Resultado.Contexto.PerfilActivoRef != lector.perfilRef() ||
		resuelto.Resultado.Contexto.Principal.ID != canal.contexto.Resultado.Contexto.Principal.ID ||
		resuelto.Resultado.Contexto.PersonaRef != canal.contexto.Resultado.Contexto.PersonaRef ||
		resuelto.Vinculo.ValidarPara(resuelto.Resultado) != nil ||
		len(e.registro.altas) != 2 || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 2 ||
		!reflect.DeepEqual(antesCanal, canal.contexto.Resultado) {
		t.Fatalf("sesión nominal incoherente: perfil=%t actor=%t persona=%t vínculo=%t altas=%d revalidaciones=%d resoluciones=%d holder=%t",
			resuelto.Resultado.Contexto.PerfilActivoRef == lector.perfilRef(),
			resuelto.Resultado.Contexto.Principal.ID == canal.contexto.Resultado.Contexto.Principal.ID,
			resuelto.Resultado.Contexto.PersonaRef == canal.contexto.Resultado.Contexto.PersonaRef,
			resuelto.Vinculo.ValidarPara(resuelto.Resultado) == nil,
			len(e.registro.altas), e.revalidador.llamadas, e.resolutor.llamadas,
			reflect.DeepEqual(antesCanal, canal.contexto.Resultado))
	}
}

func TestSesionConsultaFirmasIntervencionConstructorExigePuenteYConcesionExactos(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*entornoSesionConsultaPrueba, *soporteFiscalizacionContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo)
	}{
		{"sello_puente", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			e.soporte.sello = &selloConsultasContratacionTemporalDesarrollo{}
		}},
		{"actor_puente", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			e.soporte.principalID = "certificado:ajeno"
		}},
		{"certificado_puente", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo) {
			e.soporte.certificadoSHA256 = strings.Repeat("e", 64)
		}},
		{"permiso_firma_adicional", func(_ *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, lector *perfilFijoCTDesarrollo) {
			lector.plantilla.VersionRol.Concesiones = append(lector.plantilla.VersionRol.Concesiones, lector.plantilla.VersionRol.Concesiones[0])
		}},
		{"proyeccion_incompleta", func(_ *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, lector *perfilFijoCTDesarrollo) {
			c := &lector.plantilla.VersionRol.Concesiones[0]
			c.CamposPermitidos = slices.Clone(c.CamposPermitidos[:17])
		}},
		{"sesion_origen_reutilizada", func(_ *entornoSesionConsultaPrueba, canal *soporteFiscalizacionContratacionTemporalDesarrollo, lector *perfilFijoCTDesarrollo) {
			lector.contexto.Vinculo = canal.contexto.Vinculo
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, canal, lector, _ := sesionConsultaFirmasIntervencionPrueba(t)
			caso.alterar(e, canal, lector)
			if _, err := nuevaSesionConsultaFirmasIntervencionCTDesarrollo(context.Background(), e.p, canal, e.soporte, lector); !errors.Is(err, ports.ErrAutorizacionDenegada) {
				t.Fatalf("se aceptó una autoridad ajena o una concesión ampliada: %v", err)
			}
		})
	}
}

func TestSesionConsultaFirmasIntervencionDeniegaCanalYContextosCruzados(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*entornoSesionConsultaPrueba, *soporteFiscalizacionContratacionTemporalDesarrollo, *perfilFijoCTDesarrollo, *proveedorSesionConsultaRRHHDesarrollo, context.Context) context.Context
	}{
		{"metodo", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			c.metodo = http.MethodGet
			return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
		}},
		{"ruta", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			c.ruta = httpinterno.RutaConsultaCuadroRRHH
			return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
		}},
		{"otro_actor", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			c.principal.ID = "certificado:ajeno"
			return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
		}},
		{"otro_certificado", func(e *entornoSesionConsultaPrueba, _ *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			c.principal.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
			return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
		}},
		{"sello", func(e *entornoSesionConsultaPrueba, canal *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			canal.sello = &selloConsultasContratacionTemporalDesarrollo{}
			return ctx
		}},
		{"perfil_lector", func(_ *entornoSesionConsultaPrueba, canal *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, p *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			p.base.Contexto.PerfilActivoRef = canal.contexto.Resultado.Contexto.PerfilActivoRef
			return ctx
		}},
		{"contexto_canal", func(_ *entornoSesionConsultaPrueba, canal *soporteFiscalizacionContratacionTemporalDesarrollo, _ *perfilFijoCTDesarrollo, _ *proveedorSesionConsultaRRHHDesarrollo, ctx context.Context) context.Context {
			canal.contexto.Resultado.Contexto.Instantanea.CuentaRef = "cuenta:ajena"
			return ctx
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, canal, lector, p := sesionConsultaFirmasIntervencionPrueba(t)
			ctx := caso.cambiar(e, canal, lector, p, contextoConsultaFirmasIntervencionPrueba(t, e, p))
			if _, err := p.ResolverContexto(ctx); err == nil || len(e.registro.altas) != 0 || e.revalidador.llamadas != 0 {
				t.Fatal("un canal cruzado alcanzó la sesión nominal o recurrió a RRHH")
			}
		})
	}

	t.Run("perfil_origen", func(t *testing.T) {
		e, canal, lector, _ := sesionConsultaFirmasIntervencionPrueba(t)
		lector.contexto = canal.contexto
		if _, err := nuevaSesionConsultaFirmasIntervencionCTDesarrollo(context.Background(), e.p, canal, e.soporte, lector); !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatalf("se aceptó el perfil de origen como lector: %v", err)
		}
	})
}

func TestSesionConsultaFirmasIntervencionFallaCerradaAnteRevocacionYFuenteCaida(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*entornoSesionConsultaPrueba, *proveedorSesionConsultaRRHHDesarrollo)
	}{
		{"sesion_revocada", func(e *entornoSesionConsultaPrueba, _ *proveedorSesionConsultaRRHHDesarrollo) {
			e.revalidador.err = errors.New("sesión revocada")
		}},
		{"contexto_historico", func(e *entornoSesionConsultaPrueba, _ *proveedorSesionConsultaRRHHDesarrollo) {
			e.resolutor.historico = true
		}},
		{"fuente_contexto_caida", func(e *entornoSesionConsultaPrueba, _ *proveedorSesionConsultaRRHHDesarrollo) {
			e.resolutor.base = dominiovec.ResultadoContextoActorRegistradoV2{}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, _, _, p := sesionConsultaFirmasIntervencionPrueba(t)
			caso.alterar(e, p)
			if _, err := p.ResolverContexto(contextoConsultaFirmasIntervencionPrueba(t, e, p)); err == nil {
				t.Fatal("se resolvió una sesión revocada o una fuente no actual")
			}
		})
	}
}
