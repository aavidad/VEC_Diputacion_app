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

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	consultafirmas "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type seudonimizadorCuentaIntervencionPrueba struct {
	delegado postgresidentidad.SeudonimizadorAlta
	entradas []postgresidentidad.IdentificadoresAlta
}

func (s *seudonimizadorCuentaIntervencionPrueba) SeudonimizarAlta(ctx context.Context, ids postgresidentidad.IdentificadoresAlta) (postgresidentidad.SeudonimosAlta, error) {
	s.entradas = append(s.entradas, ids)
	return s.delegado.SeudonimizarAlta(ctx, ids)
}

func TestCuentaNominalFirmasIntervencionArranqueYReplayConMismoHMAC(t *testing.T) {
	l, _ := escenarioLectorFirmasContextoCanonicoPrueba(t)
	seudonimizador := &seudonimizadorCuentaIntervencionPrueba{delegado: &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}}
	gobierno := new(baseCuentaNominalDesarrolloPrueba)
	for i := 0; i < 2; i++ {
		if err := prepararCuentaNominalFirmasIntervencionConTransaccion(context.Background(), gobierno, l.canal, seudonimizador); err != nil {
			t.Fatal(err)
		}
	}
	v, err := l.canal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	rrhh, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if v.CuentaRef == rrhh.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		gobierno.incorporaciones != 1 || gobierno.confirmaciones != 2 || len(seudonimizador.entradas) != 2 {
		t.Fatal("cuenta mezclada con RRHH o alta de cuenta duplicada")
	}
	ids := seudonimizador.entradas[0]
	if ids.CuentaID != "desarrollo:"+v.CuentaRef || ids.SujetoID != l.canal.principalOriginal.ID ||
		ids.EspacioIdentidad != espacioIdentidadSesionDesarrollo || ids.CuentaOrdinariaID != "" {
		t.Fatal("alias derivado para otro sujeto o namespace")
	}
	esperados, err := seudonimizador.delegado.SeudonimizarAlta(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range gobierno.transacciones {
		if tx.cuenta != v.CuentaRef || tx.alias[1] != v.CuentaRef || tx.alias[5] != int64(esperados.ClaveVersion) ||
			!reflect.DeepEqual(tx.alias[6], esperados.CuentaIDHMAC[:]) || !reflect.DeepEqual(tx.alias[7], esperados.SujetoIDHMAC[:]) {
			t.Fatal("registro de alias no conserva la generación/HMAC de sesión")
		}
		for _, paso := range tx.pasos {
			if paso != "configuracion" && paso != "consultar" && paso != "incorporar" && paso != "cotejar" && paso != "alias" && paso != "commit" {
				t.Fatal("el arranque registró una sesión o autenticación")
			}
		}
	}
}

func TestCuentaNominalFirmasIntervencionNoReparaRetiradaNiAdmiteContextoAjeno(t *testing.T) {
	for _, caso := range []string{"cuenta_retirada", "alias_ajeno", "perfil_lector", "cuenta_ajena", "actor_original_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			l, _ := escenarioLectorFirmasContextoCanonicoPrueba(t)
			gobierno := new(baseCuentaNominalDesarrolloPrueba)
			s := &seudonimizadorSesionDesarrollo{derivador: nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
			switch caso {
			case "cuenta_retirada":
				gobierno.existe, gobierno.incompatibilidad = true, true
			case "alias_ajeno":
				gobierno.fallo = "alias_otra_cuenta"
			case "perfil_lector":
				l.canal.contexto = l.perfil.contexto
			case "actor_original_ajeno":
				l.canal.principalOriginal.ID = "desarrollo:actor-ajeno"
			case "cuenta_ajena":
				otro := clonarPrincipalDesarrollo(l.canal.principalOriginal)
				otro.ID = "desarrollo:actor-ajeno"
				contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(otro, l.canal.reloj.Ahora())
				if err != nil {
					t.Fatal(err)
				}
				l.canal.contexto = contexto
			}
			if err := prepararCuentaNominalFirmasIntervencionConTransaccion(context.Background(), gobierno, l.canal, s); err == nil || gobierno.confirmaciones != 0 || gobierno.incorporaciones != 0 {
				t.Fatal("estado ajeno reparado o confirmado")
			}
			if caso == "perfil_lector" || caso == "cuenta_ajena" || caso == "actor_original_ajeno" {
				if gobierno.inicios != 0 {
					t.Fatal("una identidad ajena alcanzó SQL")
				}
			}
			if caso == "cuenta_retirada" && gobierno.transacciones[0].alias != nil {
				t.Fatal("se registró alias sobre cuenta retirada")
			}
		})
	}
}

func escenarioLectorFirmasContextoCanonicoPrueba(t *testing.T) (*lectorFirmasIntervencionCTDesarrollo, context.Context) {
	t.Helper()
	original := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	reloj := relojContratacionTemporalDesarrollo{}
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(original, reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	canonico := base.Resultado.Contexto.Principal
	if canonico.ID != base.Resultado.Contexto.PersonaRef || canonico.ID == original.ID || len(canonico.Roles) != 0 || len(canonico.Attributes) != 0 {
		t.Fatal("la prueba no usa el normalizador real de ContextoActor")
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	canal := &soporteFiscalizacionContratacionTemporalDesarrollo{sello: sello, principalID: original.ID, principalOriginal: clonarPrincipalDesarrollo(original),
		certificadoSHA256: original.Attributes["certificate_sha256"], contexto: base, reloj: reloj,
		fijo: &perfilFijoCTDesarrollo{contexto: base, plantilla: dominiovec.InstantaneaAutorizacion{AsignacionPerfil: dominiovec.AsignacionPerfil{PerfilActivoRef: base.Resultado.Contexto.PerfilActivoRef}}}}
	// Mutar el objeto entregado por la identidad no altera la copia privada.
	original.Roles[0], original.Attributes["certificate_sha256"] = "rol:ajeno", strings.Repeat("b", 64)
	perfil, err := prepararPerfilLectorFirmasIntervencionCTDesarrollo(canal)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadAsignacionesContratacionTemporalDesarrolloPrueba{asignaciones: map[string]instantaneaPublicadaDesarrollo{perfil.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}}
	lector := &lectorFirmasIntervencionCTDesarrollo{canal: canal, perfil: perfil, esperado: perfil.contexto.Resultado,
		sesion: proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}, puente: &soporteAltaContratacionTemporalDesarrollo{reloj: reloj, autoridadAsignaciones: autoridad}}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{
		sello: sello, ruta: httpinterno.RutaResultadosFiscalizacion, metodo: http.MethodPost, principal: clonarPrincipalDesarrollo(canal.principalOriginal)})
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:prueba"}
	r, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, claveConsultaFirmasDocumentoCTDesarrollo{}, m)
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Accion: ports.AccionConsultarFirmasDocumento, Recurso: r, Finalidad: ports.FinalidadFirmaDocumento, ReferenciaMotivo: motivoConsultaFirmasDocumentoCTDesarrollo()})
	return lector, ctx
}

func TestLectorFirmasIntervencionConContextoV3CanonicoReal(t *testing.T) {
	l, ctx := escenarioLectorFirmasContextoCanonicoPrueba(t)
	antes, err := l.canal.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	operativo, err := l.contextoOperativo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if operativo.Resultado.Contexto.Principal.ID != antes.Contexto.Principal.ID || len(operativo.Resultado.Contexto.Principal.Roles) != 0 ||
		len(operativo.Resultado.Contexto.Principal.Attributes) != 0 || !reflect.DeepEqual(antes, l.canal.contexto.Resultado) {
		t.Fatal("se alteró el principal canónico o el contexto fiscalizador")
	}
	if _, err := l.ObtenerInstantaneaAutorizacion(ctx, operativo.Resultado.Contexto.Principal.ID, l.perfil.perfilRef()); err != nil {
		t.Fatal(err)
	}
	vOrigen, _ := l.canal.contexto.Vinculo.Datos()
	vLectura, _ := operativo.Vinculo.Datos()
	if vOrigen.SesionRef == vLectura.SesionRef || vOrigen.PerfilActivoRef == vLectura.PerfilActivoRef ||
		operativo.Resultado.Contexto.PersonaRef != antes.Contexto.PersonaRef || operativo.Resultado.Contexto.Instantanea.CuentaRef != antes.Contexto.Instantanea.CuentaRef {
		t.Fatal("el lector perdió la persona/cuenta o compartió sesión/perfil de fiscalización")
	}
	if _, err := l.ObtenerInstantaneaAutorizacion(ctx, l.canal.principalID, l.perfil.perfilRef()); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatal("se aceptó el identificador HTTP como actor canónico")
	}
}

func TestLectorFirmasIntervencionDeniegaSemillaOContextoCruzados(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		alterar  func(*lectorFirmasIntervencionCTDesarrollo)
		preparar bool
	}{
		{"actor_original", func(l *lectorFirmasIntervencionCTDesarrollo) { l.canal.principalOriginal.ID = "desarrollo:otro-actor" }, true},
		{"certificado_original", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.canal.principalOriginal.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
		}, true},
		{"rol_original", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.canal.principalOriginal.Roles = []string{rolTecnicoRRHHContratacionTemporalDesarrollo}
		}, true},
		{"cuenta", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.canal.contexto.Resultado.Contexto.Instantanea.CuentaRef = "cuenta:otra"
		}, false},
		{"persona", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.canal.contexto.Resultado.Contexto.PersonaRef = "persona:otra"
		}, false},
		{"actor_canonico", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.canal.contexto.Resultado.Contexto.Principal.ID = "persona:otra"
		}, false},
		{"perfil", func(l *lectorFirmasIntervencionCTDesarrollo) {
			l.perfil.plantilla.AsignacionPerfil.PerfilActivoRef = l.canal.contexto.Resultado.Contexto.PerfilActivoRef
		}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			l, ctx := escenarioLectorFirmasContextoCanonicoPrueba(t)
			caso.alterar(l)
			if caso.preparar {
				if _, err := prepararPerfilLectorFirmasIntervencionCTDesarrollo(l.canal); err == nil {
					t.Fatal("semilla ajena admitida")
				}
			} else if _, err := l.contextoOperativo(ctx); !errors.Is(err, ports.ErrAutorizacionDenegada) {
				t.Fatalf("cruce no denegado: %v", err)
			}
		})
	}
}

type lectorAsignacionFirmasIntervencionPrueba struct{ err error }

func (f lectorAsignacionFirmasIntervencionPrueba) PrepararInstantanea(
	_ context.Context, i dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	return i, nil
}
func (f lectorAsignacionFirmasIntervencionPrueba) PublicarInstantanea(context.Context, dominiovec.InstantaneaAutorizacion) error {
	return nil
}
func (f lectorAsignacionFirmasIntervencionPrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	return instantaneaPublicadaDesarrollo{}, false, f.err
}

func TestLectorFirmasIntervencionPerfilSoloLecturaOrganizacion(t *testing.T) {
	principal := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, base, ahora, clavePerfilFijoLectorFirmasIntervencionCT,
		[]string{httpinterno.RutaResultadosFiscalizacion},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaLectorFirmasIntervencionCTDesarrollo(actor, ref, ahora)
		})
	if err != nil {
		t.Fatal(err)
	}
	baseVinculo, _ := base.Vinculo.Datos()
	lectorVinculo, _ := perfil.contexto.Vinculo.Datos()
	if perfil.perfilRef() == baseVinculo.PerfilActivoRef || lectorVinculo.SesionRef == baseVinculo.SesionRef ||
		perfil.contexto.Resultado.Contexto.PersonaRef != base.Resultado.Contexto.PersonaRef ||
		perfil.contexto.Resultado.Contexto.Instantanea.CuentaRef != base.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("el perfil lector no mantiene identidad y sesión separadas")
	}
	concesiones := perfil.plantilla.VersionRol.Concesiones
	if len(concesiones) != 1 || concesiones[0].Accion != ports.AccionConsultarFirmasDocumento ||
		concesiones[0].TipoRecurso != ports.TipoRecursoConsultaFirmasDocumento ||
		!slices.Equal(concesiones[0].CamposPermitidos, consultafirmas.CamposConsultaFirmasDocumento()) ||
		len(perfil.plantilla.AsignacionPerfil.Ambitos) != 1 ||
		perfil.plantilla.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		!slices.Equal(perfil.plantilla.AsignacionPerfil.Ambitos[0].Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		t.Fatal("el perfil lector concede más que la consulta nominal org-only")
	}
}

func TestLectorFirmasIntervencionNoUsaOtroCanal(t *testing.T) {
	p := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	canal := &soporteFiscalizacionContratacionTemporalDesarrollo{
		sello: sello, principalID: p.ID, certificadoSHA256: p.Attributes["certificate_sha256"],
	}
	lector := &lectorFirmasIntervencionCTDesarrollo{canal: canal}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{
		sello: sello, ruta: httpinterno.RutaResultadosFiscalizacion, metodo: http.MethodPost, principal: p,
	}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if !lector.capacidadValida(ctx) {
		t.Fatal("se rechazó el canal propio de Intervención")
	}
	for _, alterar := range []func(*capacidadConsultaContratacionTemporalDesarrollo){
		func(c *capacidadConsultaContratacionTemporalDesarrollo) { c.metodo = http.MethodGet },
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.ruta = httpinterno.RutaConsultaFirmaDocumento
		},
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.principal.Roles = []string{rolTecnicoRRHHContratacionTemporalDesarrollo}
		},
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.sello = &selloConsultasContratacionTemporalDesarrollo{}
		},
	} {
		copia := capacidad
		alterar(&copia)
		if lector.capacidadValida(context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, copia)) {
			t.Fatal("se aceptó otro canal, método, rol o sello")
		}
	}
	if _, err := lector.ConsultarFirmas(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:prueba"); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("dependencia ausente debe ser indisponibilidad, recibió %v", err)
	}
}

func TestLectorFirmasIntervencionDistingueRevocacionDeCaida(t *testing.T) {
	p := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := nuevoPerfilFijoCTDesarrollo(p, base, ahora, clavePerfilFijoLectorFirmasIntervencionCT,
		[]string{httpinterno.RutaResultadosFiscalizacion},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaLectorFirmasIntervencionCTDesarrollo(actor, ref, ahora)
		})
	if err != nil {
		t.Fatal(err)
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	canal := &soporteFiscalizacionContratacionTemporalDesarrollo{sello: sello,
		principalID: p.ID, principalOriginal: clonarPrincipalDesarrollo(p), certificadoSHA256: p.Attributes["certificate_sha256"], contexto: base}
	lector := &lectorFirmasIntervencionCTDesarrollo{canal: canal, perfil: perfil,
		puente: &soporteAltaContratacionTemporalDesarrollo{
			autoridadAsignaciones: lectorAsignacionFirmasIntervencionPrueba{},
		}}
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef: "expediente:prueba"}
	recurso, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionConsultarFirmasDocumento,
		Recurso: recurso, Finalidad: ports.FinalidadFirmaDocumento,
		ReferenciaMotivo: motivoConsultaFirmasDocumentoCTDesarrollo()}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: httpinterno.RutaResultadosFiscalizacion,
			metodo: http.MethodPost, principal: p})
	ctx = context.WithValue(ctx, claveConsultaFirmasDocumentoCTDesarrollo{}, m)
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	if _, err := lector.ObtenerInstantaneaAutorizacion(ctx, base.Resultado.Contexto.Principal.ID, perfil.perfilRef()); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("asignación retirada debe denegar, recibió %v", err)
	}
	lector.puente.autoridadAsignaciones = lectorAsignacionFirmasIntervencionPrueba{err: errors.New("fuente caída")}
	if _, err := lector.ObtenerInstantaneaAutorizacion(ctx, base.Resultado.Contexto.Principal.ID, perfil.perfilRef()); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("fuente caída debe ser indisponibilidad, recibió %v", err)
	}
}
