package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

func resolutorReglasCTEjemploPrueba(t *testing.T) *reglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojPresentacionReglasEjemplo})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// El documento que acredita la incorporación, quién la confirma y si se
// ofrece el cierre sin cese salen del catálogo de ejemplo, no del código.
func TestIncorporacionAcreditadaLeeLasReglasDelCatalogo(t *testing.T) {
	r := resolutorReglasCTEjemploPrueba(t)
	f := fuenteReglaAcreditacionDesarrollo{reglas: r}
	ctx := context.Background()
	for modalidad, esperado := range map[domain.ClaveCatalogo]domain.ClaveCatalogo{"sustitucion": "toma_posesion", "vacante": "toma_posesion", "relevo": "contrato_firmado"} {
		tipo, regla, err := f.DocumentoAcreditativo(ctx, modalidad)
		if err != nil || tipo != esperado || regla.ModalidadClave != string(modalidad) || !regla.Valida() ||
			!strings.HasSuffix(regla.Referencia, reglas.CTAcreditacionIncorporacion) {
			t.Fatalf("%s: %s %+v %v", modalidad, tipo, regla, err)
		}
	}
	if _, _, err := f.DocumentoAcreditativo(ctx, "Mal"); !errors.Is(err, ports.ErrReglaAcreditacionNoDisponible) {
		t.Fatalf("modalidad inválida: %v", err)
	}
	admitido, err := admisionCierreSinCeseDesarrollo(r)(ctx)
	if err != nil || admitido {
		t.Fatalf("c10 del ejemplo no contempla el cierre sin cese: %v %v", admitido, err)
	}
	if admisionCierreSinCeseDesarrollo(nil) != nil {
		t.Fatal("sin catálogo se conserva la conducta anterior")
	}
	politica, err := fuenteReglasSeguimientoDesarrollo{reglas: r}.PoliticaConfirmacionGINPIX(ctx, relojPresentacionReglasEjemplo.ahora)
	if err != nil || !politica.ValidaEn(relojPresentacionReglasEjemplo.ahora) || politica.MotivoAutorizacion != motivoSeguimientoCeseDesarrollo(httpinterno.RutaConfirmacionesGINPIX) {
		t.Fatalf("política de GINPIX: %+v %v", politica, err)
	}
}

func TestIncorporacionCentroConcedeSegunLaReglaYLigaElRecurso(t *testing.T) {
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, CTSeguimientoCeseEnabled: "true", CTIncorporacionAcreditadaEnabled: "true",
		CTAnalisisMotivosSourcePath: rutaMotivosCTEjemploPrueba}
	cfg.ReglasEjemplo.CTSourcePath = rutaReglasCTEjemploPrueba
	cfg.ReglasEjemplo.CausasCeseSourcePath = rutaCausasCeseEjemploPrueba
	i, err := nuevaIncorporacionCentroDesarrollo(cfg, relojContratacionTemporalDesarrollo{})
	if err != nil || !i.activo || len(i.roles) != 2 {
		t.Fatalf("composición: %+v %v", i, err)
	}
	for _, rol := range []string{"solicitante_centro", "ratificador_centro"} {
		c := i.concesiones(rol)
		if len(c) != 2 || c[0].Accion != ports.AccionConsultarIncorporacionesCentro || c[1].Accion != ports.AccionConfirmarIncorporacionCentro ||
			c[1].TipoRecurso != ports.TipoRecursoIncorporacionCentro {
			t.Fatalf("%s: %+v", rol, c)
		}
	}
	i.roles = []string{"ratificador_centro"}
	if c := i.concesiones("solicitante_centro"); len(c) != 1 {
		t.Fatalf("sin la regla el solicitante solo consulta: %+v", c)
	}
	apagada, err := nuevaIncorporacionCentroDesarrollo(config.Config{}, relojContratacionTemporalDesarrollo{})
	if err != nil || apagada.activo || apagada.concesiones("ratificador_centro") != nil {
		t.Fatalf("apagada: %+v %v", apagada, err)
	}
	if rutas, err := apagada.rutas(nil); err != nil || rutas != nil {
		t.Fatalf("apagada no monta rutas: %v %v", rutas, err)
	}
	actor := domain.ActorPeticionCentro{ActorRef: "per_c1", PerfilRef: "prf_c1", CentroRef: "centro-520", PuestoRef: "puesto-1"}
	recurso := ports.RecursoIncorporacionCentro("expediente:ct:1", organizacionAltaContratacionTemporalDesarrollo, "centro-520", []byte(`{}`))
	r := &recursoIncorporacionCentroDesarrollo{accion: ports.AccionConfirmarIncorporacionCentro, recurso: recurso, actor: actor}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionConfirmarIncorporacionCentro, Recurso: recurso}
	if !r.validaPara("per_c1", "prf_c1", d) {
		t.Fatal("recurso exacto rechazado")
	}
	if r.validaPara("per_otro", "prf_c1", d) {
		t.Fatal("otro actor aceptado")
	}
	d.Accion = ports.AccionConsultarIncorporacionesCentro
	if r.validaPara("per_c1", "prf_c1", d) {
		t.Fatal("otra acción aceptada")
	}
	otro := ports.RecursoIncorporacionCentro("expediente:ct:1", organizacionAltaContratacionTemporalDesarrollo, "centro-999", []byte(`{}`))
	if (&recursoIncorporacionCentroDesarrollo{accion: ports.AccionConfirmarIncorporacionCentro, recurso: otro, actor: actor}).validaPara("per_c1", "prf_c1",
		vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionConfirmarIncorporacionCentro, Recurso: otro}) {
		t.Fatal("recurso de otro centro aceptado")
	}
}

func TestIncorporacionAcreditadaSelectorYMigraciones(t *testing.T) {
	if err := comprobarMigracionesIncorporacionAcreditadaDesarrollo(t.Context(), nil); !errors.Is(err, ErrIncorporacionAcreditadaMigracionesNoDisponibles) {
		t.Fatalf("sin pool: %v", err)
	}
	if err := validarSelectoresDespliegueBolsaCT(config.Config{CTIncorporacionAcreditadaEnabled: "si"}); !errors.Is(err, config.ErrConfiguracionCTIncorporacionAcreditadaSelector) {
		t.Fatalf("valor inválido: %v", err)
	}
	if err := validarValorSelectoresDespliegueBolsaCT(config.Config{CTIncorporacionAcreditadaEnabled: "si"}); !errors.Is(err, config.ErrConfiguracionCTIncorporacionAcreditadaSelector) {
		t.Fatalf("valor inválido fuera de vec-server: %v", err)
	}
	if err := validarSelectoresDespliegueBolsaCT(config.Config{CTIncorporacionAcreditadaEnabled: "true"}); !errors.Is(err, config.ErrConfiguracionCTIncorporacionAcreditadaActivacion) {
		t.Fatalf("sin doble llave: %v", err)
	}
	s := seleccionMaterialCTDesarrollo{incorporacionAcreditada: true}
	d := descriptoresMaterialSeleccionadosCTDesarrollo(s)
	if len(d) == 0 || d[len(d)-1].Audiencia != ports.AudienciaConsumoConfirmacionGINPIXV1 {
		t.Fatalf("descriptor de GINPIX: %+v", d)
	}
}
