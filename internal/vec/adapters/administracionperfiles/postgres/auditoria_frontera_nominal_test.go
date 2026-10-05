package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type revalidadorFronteraPrueba struct {
	a domain.AutenticacionRevalidadaV1
}

func (r revalidadorFronteraPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorFronteraPrueba struct {
	r domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorFronteraPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojFronteraPrueba struct{ t time.Time }

func (r relojFronteraPrueba) Ahora() time.Time { return r.t }

func sesionFronteraPrueba(t *testing.T, ahora time.Time) (domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) {
	t.Helper()
	r, anterior, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	d, err := anterior.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a := d.Autenticacion()
	a.CuentaOrdinariaRef = "cta_" + strings.Repeat("f", 24)
	a.CuentaPrivilegiada = true
	a.Superficie = domain.SuperficieAutenticacionAdministracionPrivilegiadaV1
	v, resultado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorFronteraPrueba{a},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef},
		resolutorFronteraPrueba{r}, domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: r.Contexto.Instantanea.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}, PerfilActivoRef: r.Contexto.PerfilActivoRef}, relojFronteraPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	e := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: v}
	if e.ValidarEn(resultado.Contexto, ahora) != nil {
		t.Fatal("V2 ADMIN inválido")
	}
	return resultado.Contexto, e
}

func configFronteraPrueba() ConfiguracionAuditoriaFronteraNominal {
	destinos := map[string]DestinoFronteraNominal{}
	for k := range accionesFronteraNominal {
		destinos[k] = DestinoFronteraNominal{Accion: "administracion.perfiles.frontera." + k, RecursoRef: "administracion:perfiles:" + k, FinalidadRef: "gestion_perfiles", TipoRecurso: "fijo"}
	}
	destinos["buscar_personas"] = DestinoFronteraNominal{Accion: "administracion.usuarios.listar", RecursoRef: "conjunto_admin:" + strings.Repeat("a", 32), FinalidadRef: "gestion_usuarios", TipoRecurso: "conjunto"}
	destinos["consultar_persona"] = DestinoFronteraNominal{Accion: "administracion.usuarios.consultar", FinalidadRef: "gestion_usuarios", TipoRecurso: "persona"}
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}
	return ConfiguracionAuditoriaFronteraNominal{Proceso: "vec_admin", Canal: "administracion_privilegiada", MotivoDenegado: motivo, MotivoError: motivo, Plazo: 500 * time.Millisecond, Destinos: destinos}
}

type registroFronteraPrueba struct {
	ordenes   []domain.DatosOrdenIntentoAuditoria
	llamadas  int
	errores   []error
	ahora     time.Time
	cancelado bool
	plazo     bool
}

func (r *registroFronteraPrueba) AppendIntentoAuditoria(ctx context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	r.llamadas++
	d, err := o.Datos()
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	r.ordenes = append(r.ordenes, d)
	r.cancelado = ctx.Err() != nil
	_, r.plazo = ctx.Deadline()
	if len(r.errores) >= r.llamadas && r.errores[r.llamadas-1] != nil {
		return ports.AcuseIntentoAuditoria{}, r.errores[r.llamadas-1]
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_" + strings.Repeat("c", 32), Secuencia: 1, HuellaSHA256: strings.Repeat("d", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: r.ahora}, nil
}

func denegacionFronteraPrueba(t *testing.T, accion, codigo, recurso string) (context.Context, api.DenegacionADMIN) {
	t.Helper()
	actor, e := sesionFronteraPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// El proveedor de sesión usa su propia referencia criptográfica, distinta
	// de la correlación técnica ya presente en ctx.
	c, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := c.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	return ctx, api.DenegacionADMIN{Codigo: codigo, Accion: accion, RecursoRef: recurso, Actor: actor, Evidencia: e, CorrelacionRef: ref,
		ActorPersonaRef: "per_" + strings.Repeat("z", 22), PerfilActivoRef: "prf_" + strings.Repeat("z", 22)}
}

func TestFronteraNominalReintentaMismaOrdenTrasCancelacion(t *testing.T) {
	cfg := configFronteraPrueba()
	espia := &registroFronteraPrueba{ahora: time.Now().UTC(), errores: []error{ports.ErrIntentoAuditoriaNoDisponible}}
	a, err := NuevaAuditorFronteraNominal(espia, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Destinos["buscar_personas"] = DestinoFronteraNominal{Accion: "administracion.otro"}
	ctx, d := denegacionFronteraPrueba(t, "buscar_personas", "solicitud_invalida", "SECRET/ruta?cursor=privado")
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := a.RegistrarDenegacionADMIN(ctx, d); err != nil {
		t.Fatal(err)
	}
	if espia.llamadas != 2 || espia.ordenes[0].IntentoRef != espia.ordenes[1].IntentoRef || espia.cancelado || !espia.plazo {
		t.Fatal("reintento cambió orden o perdió plazo propio")
	}
	datos := espia.ordenes[0].Datos
	if datos.Resultado != domain.ResultadoIntentoAuditoriaDenegado || datos.RecursoRef != "conjunto_admin:"+strings.Repeat("a", 32) ||
		strings.Contains(datos.RecursoRef, "SECRET") || datos.Accion != "administracion.usuarios.listar" ||
		espia.ordenes[0].ResultadoContexto.Contexto.PersonaRef != d.Actor.PersonaRef {
		t.Fatalf("autoridad de strings HTTP infiltrada: %+v", datos)
	}
}

func TestFronteraNominalPersonaExactaYFallbackOpaco(t *testing.T) {
	a, err := NuevaAuditorFronteraNominal(&registroFronteraPrueba{}, configFronteraPrueba())
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ref    string
		exacta bool
	}{{"per_" + strings.Repeat("p", 22), true}, {"../../dato-personal?x", false}} {
		espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
		a.registrador = espia
		ctx, d := denegacionFronteraPrueba(t, "consultar_persona", "solicitud_invalida", caso.ref)
		if err := a.RegistrarDenegacionADMIN(ctx, d); err != nil {
			t.Fatal(err)
		}
		got := espia.ordenes[0].Datos.RecursoRef
		if caso.exacta && got != caso.ref || !caso.exacta && (!strings.HasPrefix(got, "solicitud_admin:") || strings.Contains(got, caso.ref)) {
			t.Fatalf("recurso persona=%q exacta=%t", got, caso.exacta)
		}
	}
}

func TestFronteraNominalSinV2NoFabricaActor(t *testing.T) {
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a, err := NuevaAuditorFronteraNominal(espia, configFronteraPrueba())
	if err != nil {
		t.Fatal(err)
	}
	ctx, d := denegacionFronteraPrueba(t, "consultar", "servicio_no_disponible", "")
	d.Actor = domain.ContextoActor{}
	d.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
	if err := a.RegistrarDenegacionADMIN(ctx, d); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || espia.llamadas != 0 {
		t.Fatalf("actor fabricado: %v", err)
	}
}

func TestFronteraNominalErrorDeRespuestaUsaMotivoErrorYRecursoPrivado(t *testing.T) {
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a, err := NuevaAuditorFronteraNominal(espia, configFronteraPrueba())
	if err != nil {
		t.Fatal(err)
	}
	ctx, d := denegacionFronteraPrueba(t, "consultar", "servicio_no_disponible", "SECRET/archivo?token")
	if err := a.RegistrarDenegacionADMIN(ctx, d); err != nil {
		t.Fatal(err)
	}
	x := espia.ordenes[0].Datos
	if x.Resultado != domain.ResultadoIntentoAuditoriaError || x.Motivo.Referencia() != a.config.MotivoError.Referencia() || x.RecursoRef != "administracion:perfiles:consultar" || strings.Contains(x.RecursoRef, "SECRET") {
		t.Fatalf("error de frontera filtró datos: %+v", x)
	}
}

func TestFronteraNominalRespuestaIncompatibleDePersonaAuditaError(t *testing.T) {
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a, err := NuevaAuditorFronteraNominal(espia, configFronteraPrueba())
	if err != nil {
		t.Fatal(err)
	}
	persona := "per_" + strings.Repeat("p", 22)
	ctx, d := denegacionFronteraPrueba(t, "consultar_persona", "respuesta_incompatible", persona)
	d.ActorPersonaRef = "SECRETcuerpoRRHH"
	d.PerfilActivoRef = "SECRETcabecera"
	if err := a.RegistrarDenegacionADMIN(ctx, d); err != nil {
		t.Fatal(err)
	}
	if espia.llamadas != 1 || len(espia.ordenes) != 1 {
		t.Fatal("respuesta incompatible sin acuse común")
	}
	datos := espia.ordenes[0].Datos
	if datos.Resultado != domain.ResultadoIntentoAuditoriaError || datos.RecursoRef != persona || datos.Accion != "administracion.usuarios.consultar" ||
		strings.Contains(datos.RecursoRef, "SECRET") || espia.ordenes[0].ResultadoContexto.Contexto.PersonaRef != d.Actor.PersonaRef {
		t.Fatalf("error de ficha mal registrado: %+v", datos)
	}
}

// El destino de la preparación es opcional (configuraciones anteriores al
// lote) y, si está, se valida como los demás. Una clave desconocida no entra.
func TestFronteraNominalDestinoOpcionalDePreparacion(t *testing.T) {
	reg := &registroFronteraPrueba{ahora: time.Now()}
	if _, err := NuevaAuditorFronteraNominal(reg, configFronteraPrueba()); err != nil {
		t.Fatal("sin destino de preparación:", err)
	}
	c := configFronteraPrueba()
	c.Destinos["preparar_lote_ordinario"] = DestinoFronteraNominal{Accion: "administracion.perfiles.preparar_lote_ordinario",
		RecursoRef: "administracion:perfiles:preparar_lote_ordinario", FinalidadRef: "gestion_perfiles", TipoRecurso: "fijo"}
	if _, err := NuevaAuditorFronteraNominal(reg, c); err != nil {
		t.Fatal("con destino de preparación:", err)
	}
	c.Destinos["preparar_lote_ordinario"] = DestinoFronteraNominal{Accion: "otra.accion", RecursoRef: "administracion:perfiles:x", FinalidadRef: "gestion_perfiles", TipoRecurso: "fijo"}
	if _, err := NuevaAuditorFronteraNominal(reg, c); err == nil {
		t.Fatal("destino de preparación inválido aceptado")
	}
	c = configFronteraPrueba()
	c.Destinos["desconocida"] = c.Destinos["escribir"]
	if _, err := NuevaAuditorFronteraNominal(reg, c); err == nil {
		t.Fatal("destino desconocido aceptado")
	}
}
