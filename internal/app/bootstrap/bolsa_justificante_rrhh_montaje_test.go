package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type seleccionadorJustificanteNominalPrueba struct {
	registro *registroSesionConsultaPrueba
	perfil   string
	llamadas int
}

func (s *seleccionadorJustificanteNominalPrueba) SeleccionarPerfilJustificanteBolsaRRHH(
	_ context.Context, q vecports.SolicitudPerfilJustificanteBolsaRRHH,
) (vecports.SeleccionPerfilJustificanteBolsaRRHH, error) {
	s.llamadas++
	c := s.registro.confirmacion
	if q.AutenticacionRef != c.AutenticacionRef || q.SesionRef != c.SesionRef ||
		q.CuentaConfirmadaRef != c.CuentaRef || q.ControlSesionRef != c.ControlSesionRef ||
		q.ControlSesionRevision != c.ControlSesionRevision ||
		q.ControlSesionSHA256 != c.ControlSesionHuellaSHA256 ||
		q.AutenticacionSHA256 != c.AltaConfirmada.AutenticacionHuellaSHA256 ||
		!core.ReferenciaCorrelacionAutorizacionV2Valida(q.CorrelacionRef) {
		return vecports.SeleccionPerfilJustificanteBolsaRRHH{}, docports.ErrSubidaLocalDenegada
	}
	return vecports.SeleccionPerfilJustificanteBolsaRRHH{
		PerfilActivoRef: s.perfil, AuditoriaRef: "aud_v3_b_" + strings.Repeat("a", 32)}, nil
}

type entornoJustificanteNominalPrueba struct {
	sesion    sesionTecnicaJustificanteRenunciaRRHH
	capacidad capacidadConsultaContratacionTemporalDesarrollo
	frontera  fronteraSeguridadComunDesarrollo
	registro  *registroSesionConsultaPrueba
	selector  *seleccionadorJustificanteNominalPrueba
}

func nuevoEntornoJustificanteNominalPrueba(t *testing.T,
	principal core.Principal, base core.ResultadoContextoActorRegistradoV2,
	sujeto string) *entornoJustificanteNominalPrueba {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	reloj := &relojSesionConsultaPrueba{ahora: ahora.Add(time.Second)}
	cuentaRef := base.Contexto.Instantanea.CuentaRef
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuentaRef}
	revalidador := &revalidadorSesionConsultaPrueba{registro: registro}
	contextos := &resolutorSesionConsultaPrueba{base: base, reloj: reloj}
	usuarios := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{
		registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj,
		instancia: "instancia-justificante-prueba"}}
	selector := &seleccionadorJustificanteNominalPrueba{registro: registro,
		perfil: base.Contexto.PerfilActivoRef}
	principal.ID = sujeto
	principal.Roles = nil
	certificado := principal.Attributes["certificate_sha256"]
	cuentas := map[string]cuentaNominalJustificanteRRHH{certificado: {
		certificadoSHA256: certificado, sujeto: sujeto, cuentaRef: cuentaRef}}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{
		fronteraJustificanteBolsaRRHHDesarrollo([]string{base.Contexto.PerfilActivoRef})})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, _ := fronteras.resolver(http.MethodGet, bolsahttp.RutaJustificanteRenunciaRRHH)
	holder := &contextoOperacionCTDesarrollo{}
	return &entornoJustificanteNominalPrueba{
		sesion: sesionTecnicaJustificanteRenunciaRRHH{usuarios: usuarios, selector: selector, cuentas: cuentas},
		capacidad: capacidadConsultaContratacionTemporalDesarrollo{principal: principal,
			certificadoVerificadoEn: ahora.Add(-time.Second),
			certificadoValidoHasta:  ahora.Add(5 * time.Minute), contextoOperacion: holder},
		frontera: fronteraSeguridadComunDesarrollo{metodo: http.MethodGet,
			ruta: bolsahttp.RutaJustificanteRenunciaRRHH, catalogo: fronteras,
			descriptor: descriptor, holder: holder}, registro: registro, selector: selector,
	}
}

func TestJustificanteRRHHResolucionNominalSeparaSujetoCertificadoDePersona(t *testing.T) {
	base, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	segundo := clonarPrincipalDesarrollo(principal)
	segundo.ID = "tecnica_rrhh_segunda"
	segundo.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
	contextoSegundo, err := nuevoContextoAltaContratacionTemporalDesarrollo(segundo,
		time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	if base.contexto.Resultado.Contexto.PerfilActivoRef == contextoSegundo.Resultado.Contexto.PerfilActivoRef ||
		base.contexto.Resultado.Contexto.Instantanea.CuentaRef == contextoSegundo.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("las técnicas no tienen perfil y cuenta nominales distintos")
	}
	casos := []struct {
		principal core.Principal
		actor     core.ResultadoContextoActorRegistradoV2
		sujeto    string
	}{
		{principal, base.contexto.Resultado, "sujeto_certificado_primera"},
		{segundo, contextoSegundo.Resultado, "sujeto_certificado_segunda"},
	}
	var personas []string
	for _, caso := range casos {
		e := nuevoEntornoJustificanteNominalPrueba(t, caso.principal, caso.actor, caso.sujeto)
		resultado, err := e.sesion.resolverNominal(context.Background(), e.capacidad, e.frontera)
		if err != nil || resultado.Resultado.Validar() != nil ||
			resultado.Vinculo.ValidarPara(resultado.Resultado) != nil ||
			resultado.Resultado.Contexto.PersonaRef == caso.sujeto ||
			resultado.Resultado.Contexto.Instantanea.CuentaRef != caso.actor.Contexto.Instantanea.CuentaRef ||
			resultado.Resultado.Contexto.PerfilActivoRef != caso.actor.Contexto.PerfilActivoRef ||
			len(e.registro.altas) != 1 || e.registro.altas[0].SujetoID != caso.sujeto || e.selector.llamadas != 1 {
			t.Fatalf("sesión nominal divergente: error=%v registro=%d selector=%d", err, len(e.registro.altas), e.selector.llamadas)
		}
		personas = append(personas, resultado.Resultado.Contexto.PersonaRef)
	}
	if personas[0] == personas[1] {
		t.Fatal("dos técnicas compartieron PersonaRef")
	}
}

func TestJustificanteRRHHResolucionNominalDeniegaVinculosAjenos(t *testing.T) {
	base, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	e := nuevoEntornoJustificanteNominalPrueba(t, principal, base.contexto.Resultado, "sujeto_certificado_primera")
	certificado := e.capacidad.principal.Attributes["certificate_sha256"]
	ajeno := e.capacidad
	ajeno.principal.ID = "sujeto_certificado_ajeno"
	if _, err := e.sesion.resolverNominal(context.Background(), ajeno, e.frontera); !errors.Is(err, docports.ErrSubidaLocalDenegada) || len(e.registro.altas) != 0 {
		t.Fatalf("sujeto ajeno llegó al registro: %v", err)
	}
	ajeno = e.capacidad
	ajeno.principal = clonarPrincipalDesarrollo(e.capacidad.principal)
	ajeno.principal.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
	if _, err := e.sesion.resolverNominal(context.Background(), ajeno, e.frontera); !errors.Is(err, docports.ErrSubidaLocalDenegada) || len(e.registro.altas) != 0 {
		t.Fatalf("huella ajena llegó al registro: %v", err)
	}
	cuenta := e.sesion.cuentas[certificado]
	cuenta.cuentaRef = referenciaAltaContratacionTemporalDesarrollo("cta_", "cuenta-ajena")
	e.sesion.cuentas[certificado] = cuenta
	if _, err := e.sesion.resolverNominal(context.Background(), e.capacidad, e.frontera); !errors.Is(err, docports.ErrSubidaLocalNoDisponible) || e.selector.llamadas != 0 {
		t.Fatalf("cuenta divergente alcanzó selector: %v", err)
	}
	cuenta.cuentaRef = base.contexto.Resultado.Contexto.Instantanea.CuentaRef
	e.sesion.cuentas[certificado] = cuenta
	e.selector.perfil = referenciaAltaContratacionTemporalDesarrollo("prf_", "perfil-ajeno")
	if _, err := e.sesion.resolverNominal(context.Background(), e.capacidad, e.frontera); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("perfil ajeno admitido: %v", err)
	}
}

type diagnosticoSelectorJustificantePrueba struct{ etapa, codigo string }

func (d diagnosticoSelectorJustificantePrueba) Error() string {
	return "postgres://clave-secreta@local"
}
func (d diagnosticoSelectorJustificantePrueba) DiagnosticoSelectorPerfilJustificante() (string, string) {
	return d.etapa, d.codigo
}

func TestJustificanteRRHHRegistraCausaTecnicaSinExponerSecreto(t *testing.T) {
	var salida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&salida, nil)))
	defer slog.SetDefault(anterior)
	registrarFalloSelectorJustificanteRRHH(diagnosticoSelectorJustificantePrueba{"consulta", "sqlstate_42883"})
	registrarFalloSelectorJustificanteRRHH(diagnosticoSelectorJustificantePrueba{"postgres://clave-secreta@local", "postgres://clave-secreta@local"})
	causa := errors.New("postgres://clave-secreta@local")
	fallo := noDisponibleSesionNominalJustificanteRRHH("registro_sesion", causa)
	texto := salida.String()
	if !strings.Contains(texto, "sqlstate_42883") || !strings.Contains(texto, "etapa=consulta") ||
		strings.Contains(texto, "clave-secreta") || !strings.Contains(texto, "causa=fuente_autorizacion") ||
		!strings.Contains(texto, "etapa=registro_sesion") || !errors.Is(fallo, causa) ||
		!errors.Is(fallo, docports.ErrSubidaLocalNoDisponible) || strings.Contains(fallo.Error(), "clave-secreta") {
		t.Fatalf("diagnóstico no acotado: %s", texto)
	}
}

func TestJustificanteRRHHResolverNominalNoExigeRolCT(t *testing.T) {
	base, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.Roles = nil
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "bolsa-justificante-renuncia-rrhh", Metodo: http.MethodGet,
		Ruta:               bolsahttp.RutaJustificanteRenunciaRRHH,
		Superficie:         superficieInternaSeguridadComunDesarrollo,
		PerfilesActivosRef: []string{base.contexto.Resultado.Contexto.PerfilActivoRef},
		ClavePolitica:      "politica-documental-prueba", ClaveCapacidad: "capacidad-documental-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	cuentas := map[string]cuentaNominalJustificanteRRHH{
		principal.Attributes["certificate_sha256"]: {
			certificadoSHA256: principal.Attributes["certificate_sha256"],
			sujeto:            principal.ID, cuentaRef: referenciaAltaContratacionTemporalDesarrollo("cta_", "tecnica-nominal"),
		},
	}
	sesion := sesionTecnicaJustificanteRenunciaRRHH{
		autoridad: &autoridadConsultasContratacionTemporalDesarrollo{
			sello: base.sello, fronterasSeguridadComun: fronteras,
		}, cuentas: cuentas,
	}
	resolver := func(p core.Principal) error {
		ctx := contextoRutaCoberturaDesarrolloPrueba(base, p, bolsahttp.RutaJustificanteRenunciaRRHH)
		c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		c.metodo = http.MethodGet
		ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
		ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
			metodo: http.MethodGet, ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
			superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras,
		})
		_, err := sesion.ResolverContexto(ctx)
		return err
	}
	// Se invoca el resolutor de producción. La cuenta válida debe alcanzar la
	// composición nominal, aquí ausente, aunque el principal no lleve rol CT.
	if err := resolver(principal); !errors.Is(err, docports.ErrSubidaLocalNoDisponible) ||
		!errors.Is(err, errDependenciaSesionNominalJustificanteRRHH) {
		t.Fatalf("la cuenta interna no alcanzó su sesión nominal: %v", err)
	}
	aspirante := clonarPrincipalDesarrollo(principal)
	aspirante.Roles = []string{"candidato_bolsa"}
	if err := resolver(aspirante); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("aspirante admitido: %v", err)
	}
	forastera := clonarPrincipalDesarrollo(principal)
	forastera.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
	if err := resolver(forastera); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("certificado no inventariado admitido: %v", err)
	}
}

func TestJustificanteRenunciaRRHHSinConfiguracionDejaRutaCerrada(t *testing.T) {
	ruta, cerrar := nuevaRutaJustificanteRenunciaRRHH(config.Config{}, nil, nil, nil, nil)
	if ruta == nil || ruta.Ruta != bolsahttp.RutaJustificanteRenunciaRRHH || ruta.Manejador == nil || cerrar == nil {
		t.Fatal("la falta de configuración impidió montar la ruta cerrada")
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		ruta.Ruta+"?bolsa=bolsa_1&participacion=part_1", nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") == "" {
		t.Fatalf("sin configuración: estado=%d cabeceras=%v", w.Code, w.Header())
	}
	cerrar()
}

func TestConfiguracionJustificanteRenunciaRRHHNoExponeDSN(t *testing.T) {
	c := configuracionJustificanteRenunciaRRHH{DSNDocumentos: "postgres://usuario:clave-ficticia@127.0.0.1/vec"}
	b, err := json.Marshal(c)
	if err != nil || strings.Contains(string(b), "clave-ficticia") ||
		strings.Contains(fmt.Sprintf("%+v %#v", c, c), "clave-ficticia") {
		t.Fatal("la configuración expuso el DSN")
	}
}

func TestConfiguracionJustificanteRRHHSoloAceptaDependenciasTecnicas(t *testing.T) {
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, rutaConfiguracionJustificanteRenunciaRRHH)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	base := `{"version":1,"superficie":"interna_corporativa","dsn_documentos":"postgres://ejemplo"`
	for _, caso := range []struct {
		nombre, contenido string
		aceptar           bool
	}{
		{"tecnica", base + `}`, true},
		{"persona_fijada", base + `,"persona_ref":"per_fijada"}`, false},
		{"perfil_fijado", base + `,"perfil_ref":"prf_fijado"}`, false},
		{"certificado_fijado", base + `,"certificado_sha256":"fijado"}`, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := os.WriteFile(ruta, []byte(caso.contenido), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := leerConfiguracionJustificanteRenunciaRRHH(config.Config{DevelopmentMaterialDir: directorio})
			if (err == nil) != caso.aceptar {
				t.Fatalf("configuracion aceptada=%t", err == nil)
			}
		})
	}
}

type consultorJustificanteRRHHPermisoPrueba struct {
	actorRefs []string
	denegar   bool
}

type claveContextoNominalJustificantePrueba struct{}
type sesionNominalJustificantePrueba struct{}

func (sesionNominalJustificantePrueba) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	z, ok := ctx.Value(claveContextoNominalJustificantePrueba{}).(contextoSeguridadComunDesarrollo)
	if !ok {
		return contextoSeguridadComunDesarrollo{}, docports.ErrSubidaLocalDenegada
	}
	return z, nil
}

func (c *consultorJustificanteRRHHPermisoPrueba) AbrirJustificanteRenunciaRRHH(_ context.Context,
	i bolsahttp.IdentidadJustificanteRenunciaRRHH, _, _ string) (io.ReadCloser, error) {
	c.actorRefs = append(c.actorRefs, i.Actor.Contexto.PersonaRef)
	if c.denegar {
		return nil, docports.ErrSubidaLocalDenegada
	}
	return io.NopCloser(strings.NewReader("%PDF-1.4\nrecibo\n%%EOF")), nil
}

func TestJustificanteRRHHResuelveDosTecnicasYDeniegaSinPermiso(t *testing.T) {
	base, _, principalBase := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principalLector := clonarPrincipalDesarrollo(principalBase)
	principalLector.ID = "tecnica_rrhh_segunda"
	principalLector.Attributes["certificate_sha256"] = strings.Repeat("e", 64)
	contextoLector, err := nuevoContextoAltaContratacionTemporalDesarrollo(principalLector,
		time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	lector := &soporteAltaContratacionTemporalDesarrollo{sello: base.sello,
		principalID: principalLector.ID, certificadoSHA256: principalLector.Attributes["certificate_sha256"],
		lectorConsultasRRHH: true, contexto: contextoLector,
		contextoEsperadoRegistrado: contextoLector.Resultado, reloj: relojContratacionTemporalDesarrollo{},
		sesionOperativa: proveedorSesionOperativaCTPrueba{contexto: contextoLector}}
	perfiles := []string{base.contexto.Resultado.Contexto.PerfilActivoRef,
		lector.contexto.Resultado.Contexto.PerfilActivoRef}
	if perfiles[0] == perfiles[1] {
		t.Fatal("las tecnicas comparten un perfil")
	}
	// Ambas técnicas tienen también Preferencias, cada una con un perfil
	// personal distinto. Ninguna referencia de ese catálogo entra en la
	// sesión técnica que se entrega al justificante.
	perfilesUsuarios := map[string]string{
		principalBase.ID:   referenciaAltaContratacionTemporalDesarrollo("prf_", "preferencias-primera"),
		principalLector.ID: referenciaAltaContratacionTemporalDesarrollo("prf_", "preferencias-segunda"),
	}
	if perfilesUsuarios[principalBase.ID] == perfilesUsuarios[principalLector.ID] ||
		perfilesUsuarios[principalBase.ID] == perfiles[0] || perfilesUsuarios[principalLector.ID] == perfiles[1] {
		t.Fatal("la prueba no separa perfiles personales y técnicos")
	}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "bolsa-justificante-renuncia-rrhh", Metodo: http.MethodGet,
		Ruta:               bolsahttp.RutaJustificanteRenunciaRRHH,
		Superficie:         superficieInternaSeguridadComunDesarrollo,
		PerfilesActivosRef: perfiles[:1], ClavePolitica: "politica-documental-prueba",
		ClaveCapacidad: "capacidad-documental-prueba",
	}})
	if err != nil {
		t.Fatal(err)
	}
	cuentas := map[string]cuentaNominalJustificanteRRHH{}
	for indice, principal := range []core.Principal{principalBase, principalLector} {
		cuentas[principal.Attributes["certificate_sha256"]] = cuentaNominalJustificanteRRHH{
			certificadoSHA256: principal.Attributes["certificate_sha256"],
			sujeto:            principal.ID, cuentaRef: referenciaAltaContratacionTemporalDesarrollo("cta_", fmt.Sprint(indice)),
		}
		if !principalNominalJustificanteRRHHValido(principal, cuentas) {
			t.Fatal("cuenta técnica no admitida")
		}
	}
	otroRol := clonarPrincipalDesarrollo(principalLector)
	otroRol.Roles = []string{"rol_bolsa_publicado_distinto"}
	if !principalNominalJustificanteRRHHValido(otroRol, cuentas) {
		t.Fatal("se exigió un nombre fijo de rol")
	}
	aspirante := clonarPrincipalDesarrollo(principalLector)
	aspirante.Roles = []string{"candidato_bolsa"}
	if principalNominalJustificanteRRHHValido(aspirante, cuentas) {
		t.Fatal("el aspirante cruzó la frontera interna")
	}
	forastera := clonarPrincipalDesarrollo(principalLector)
	forastera.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
	if principalNominalJustificanteRRHHValido(forastera, cuentas) ||
		principalNominalJustificanteRRHHValido(principalLector, nil) {
		t.Fatal("se admitió certificado sin cuenta interna propia")
	}
	consulta := &consultorJustificanteRRHHPermisoPrueba{}
	h, err := bolsahttp.NuevoHandlerJustificanteRenunciaRRHH(
		proveedorIdentidadJustificanteRenunciaRRHH{sesion: sesionNominalJustificantePrueba{}}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	peticion := func(soporte *soporteAltaContratacionTemporalDesarrollo, principal core.Principal) *http.Request {
		r := httptest.NewRequest(http.MethodGet,
			bolsahttp.RutaJustificanteRenunciaRRHH+"?bolsa=bolsa_1&participacion=part_1", nil)
		ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, bolsahttp.RutaJustificanteRenunciaRRHH)
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		capacidad.metodo = http.MethodGet
		ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
		ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
			metodo: http.MethodGet, ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
			superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras})
		f, ok := fronteraSeguridadComunDesdeContexto(ctx)
		if !ok {
			t.Fatal("frontera sin registrar")
		}
		z := contextoSeguridadComunDesarrollo{Vinculo: soporte.contexto.Vinculo,
			Resultado: soporte.contexto.Resultado}
		if perfilJustificanteAdmitidoPorFrontera(f, z) {
			t.Fatal("perfil estático sin acuse nominal")
		}
		z.seleccionJustificanteRRHH = &seleccionNominalJustificanteRRHH{
			perfil:    z.Resultado.Contexto.PerfilActivoRef,
			auditoria: "aud_v3_b_" + strings.Repeat("a", 32),
			ruta:      bolsahttp.RutaJustificanteRenunciaRRHH, metodo: http.MethodGet,
			catalogo: fronteras.identidad, holder: f.holder,
		}
		if !perfilJustificanteAdmitidoPorFrontera(f, z) {
			t.Fatal("acuse propio rechazado")
		}
		ctx = context.WithValue(ctx, claveContextoNominalJustificantePrueba{}, z)
		return r.WithContext(ctx)
	}
	peticiones := []*http.Request{peticion(base, principalBase), peticion(lector, principalLector)}
	contextos := []core.ResultadoContextoActorRegistradoV2{
		base.contexto.Resultado, lector.contexto.Resultado}
	for indice, r := range peticiones {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || len(consulta.actorRefs) != indice+1 ||
			consulta.actorRefs[indice] != contextos[indice].Contexto.PersonaRef {
			t.Fatalf("tecnica %d: estado=%d actor=%v", indice, w.Code, consulta.actorRefs)
		}
	}
	if consulta.actorRefs[0] == consulta.actorRefs[1] {
		t.Fatal("las tecnicas compartieron identidad")
	}
	consulta.denegar = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticiones[1])
	if w.Code != http.StatusForbidden || len(consulta.actorRefs) != 3 ||
		consulta.actorRefs[2] != contextos[1].Contexto.PersonaRef {
		t.Fatalf("permiso retirado: estado=%d actor=%v", w.Code, consulta.actorRefs)
	}
}
