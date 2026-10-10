package bootstrap

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func perfilesB2VinculoPrueba(t *testing.T, config *archivoIncorporacionPersonalB2) *perfilesNominalesIncorporacion {
	t.Helper()
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: vinculo.PrincipalID, PerfilV3Ref: vinculo.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "unidad:desarrollo:rrhh", ActorRef: vinculo.PrincipalID}
	detalle, err := nuevoPerfilNominalIncorporacion(soporte, refs, claveIncorporacionDetalle, nil, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	p := &perfilesNominalesIncorporacion{soporte: soporte, consultas: consultas, detalle: detalle}
	if err := extenderPerfilesNominalesB2(p, refs, config, soporte.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	return p
}

// El registro CT154 tiene un perfil nominal propio, sólo con la concesión que
// AD3-127 admite (campos ["recibo"]), y no amplía el perfil de consulta.
func TestVinculoRPTB2RegistroConPerfilPropioYCamposExactos(t *testing.T) {
	config := configuracionB2PuraPrueba().PersonalB2
	p := perfilesB2VinculoPrueba(t, config)
	registro := p.b2[ct.AccionRegistrarVinculoCategoriaRPT]
	consulta := p.b2[ct.AccionConsultarVinculoCategoriaRPT]
	lecturaRPT := p.b2[ct.AccionConsultarPublicacionCategoriaRPT]
	if registro == nil || registro == consulta || registro == lecturaRPT || registro.clave != "incorporacion_b2_"+grupoRegistroVinculoRPTB2 {
		t.Fatal("el registro del vínculo no tiene perfil nominal propio")
	}
	if registro.perfilRef() == lecturaRPT.perfilRef() {
		t.Fatal("CT154 exige perfiles distintos para registrar y leer la publicación RPT")
	}
	c := registro.plantilla.VersionRol.Concesiones
	if len(c) != 1 || c[0].Accion != ct.AccionRegistrarVinculoCategoriaRPT || !slices.Equal(c[0].CamposPermitidos, []string{"recibo"}) ||
		c[0].ModuloID != ct.ModuloContratacion || c[0].TipoRecurso != "vinculo_categoria_rpt_ct" ||
		!slices.Equal(c[0].Finalidades, []string{finalidadVinculoCategoriaRPTCT}) || c[0].GarantiaMinima != core.AuthAssuranceHigh {
		t.Fatalf("concesión del registro distinta de AD3-127: %+v", c)
	}
	a := registro.plantilla.AsignacionPerfil.Ambitos
	if len(a) != 1 || a[0].Clave != "organizacion_ref" || !slices.Equal(a[0].Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		t.Fatalf("ámbitos del registro: %+v", a)
	}
	for _, concesion := range consulta.plantilla.VersionRol.Concesiones {
		if concesion.Accion == ct.AccionRegistrarVinculoCategoriaRPT {
			t.Fatal("el perfil de consulta ganó el registro")
		}
	}
}

// Sin la operación en la configuración privada (caso de un servidor ya
// desplegado) todo sigue arrancando, no se crea el perfil y el registro se deniega.
func TestVinculoRPTB2RegistroOpcionalDeniegaSinConfiguracion(t *testing.T) {
	config := configuracionB2PuraPrueba().PersonalB2
	delete(config.Operaciones, claveRegistroVinculoRPTB2)
	if err := validarConfiguracionIncorporacionB2(config); err != nil {
		t.Fatalf("configuración previa sin registro rechazada: %v", err)
	}
	p := perfilesB2VinculoPrueba(t, config)
	if p.b2[ct.AccionRegistrarVinculoCategoriaRPT] != nil || len(p.b2) != len(operacionesIncorporacionB2())-1 {
		t.Fatal("perfil de registro creado sin configuración")
	}
	for _, perfil := range p.todos() {
		if strings.HasSuffix(perfil.clave, grupoRegistroVinculoRPTB2) {
			t.Fatal("perfil de registro provisionable sin configuración")
		}
	}
	a := &autoridadIncorporacionPersonalB2{perfiles: p, catalogoRPTID: config.CatalogoRPTID, moduloRPTID: config.ModuloRPTID}
	ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: httpct.RutaVinculoCategoriaRPTB2})
	if _, err := a.RegistroCT(ctx, registroVinculoPrueba(config.CatalogoRPTID, config.ModuloRPTID)); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("registro sin perfil configurado: %v", err)
	}

	otra := configuracionB2PuraPrueba().PersonalB2
	delete(otra.Operaciones, "ct_vinculo_consultar")
	if validarConfiguracionIncorporacionB2(otra) == nil {
		t.Fatal("una operación obligatoria ausente se admitió")
	}
	extra := configuracionB2PuraPrueba().PersonalB2
	delete(extra.Operaciones, claveRegistroVinculoRPTB2)
	extra.Operaciones["ct_vinculo_otro"] = config.Operaciones["ct_vinculo_consultar"]
	if validarConfiguracionIncorporacionB2(extra) == nil {
		t.Fatal("una operación desconocida sustituyó al registro")
	}
}

func registroVinculoPrueba(catalogo, modulo string) ct.RegistroVinculoCategoriaRPT {
	return ct.RegistroVinculoCategoriaRPT{Esquema: ct.EsquemaRegistroVinculoCategoriaRPT, OrganizacionRef: "organizacion:prueba",
		ExpedienteRef: "expediente:prueba-1", VersionExpedienteEsperada: 3, AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis",
		AnalisisHuellaSHA256: strings.Repeat("a", 64), CategoriaRef: "auxiliar_administrativo", CatalogoID: catalogo, ModuloID: modulo,
		CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "auxiliar_administrativo",
		FuenteRef: "fuente:rpt-ejemplo", MotivoRef: "motivo:vinculo", AprobacionRef: "aprobacion:rrhh",
		ClaveIdempotencia: "11111111-2222-4333-8444-555555555555"}
}

// Cada método de la ruta nueva sólo admite sus acciones; la confirmación B2
// no puede usarse para registrar el vínculo.
func TestVinculoRPTB2AccionesPermitidasPorRuta(t *testing.T) {
	en := func(metodo, ruta string) context.Context {
		return context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: metodo, ruta: ruta})
	}
	casos := []struct {
		ctx    context.Context
		accion string
		ok     bool
	}{
		{en("GET", httpct.RutaVinculoCategoriaRPTB2), ct.AccionConsultarVinculoCategoriaRPT, true},
		{en("GET", httpct.RutaVinculoCategoriaRPTB2), ct.AccionRegistrarVinculoCategoriaRPT, false},
		{en("GET", httpct.RutaVinculoCategoriaRPTB2), ct.AccionConsultarPublicacionCategoriaRPT, false},
		{en("POST", httpct.RutaVinculoCategoriaRPTB2), ct.AccionRegistrarVinculoCategoriaRPT, true},
		{en("POST", httpct.RutaVinculoCategoriaRPTB2), ct.AccionConsultarPublicacionCategoriaRPT, true},
		{en("POST", httpct.RutaVinculoCategoriaRPTB2), ct.AccionRegistrarPlanNominalB2, false},
		{en("POST", httpct.RutaVinculoCategoriaRPTB2), ct.AccionConsultarDetalleRRHH, false},
		{en("PUT", httpct.RutaVinculoCategoriaRPTB2), ct.AccionRegistrarVinculoCategoriaRPT, false},
		{en("POST", httpct.RutaConfirmacionB2), ct.AccionRegistrarVinculoCategoriaRPT, false},
		{en("POST", httpct.RutaPlanB2), ct.AccionRegistrarVinculoCategoriaRPT, false},
	}
	for _, c := range casos {
		if operacionPermitidaEnRutaIncorporacionB2(c.ctx, c.accion) != c.ok {
			t.Errorf("%v %s: se esperaba %v", c.ctx.Value(claveRutaPeticionIncorporacionB2{}), c.accion, c.ok)
		}
	}
}

func TestVinculoRPTB2RegistroCTRechazaCatalogoAjeno(t *testing.T) {
	a := &autoridadIncorporacionPersonalB2{catalogoRPTID: "categorias_rpt", moduloRPTID: "personal"}
	for _, m := range []ct.RegistroVinculoCategoriaRPT{registroVinculoPrueba("otro_catalogo", "personal"), registroVinculoPrueba("categorias_rpt", "otro_modulo")} {
		if _, err := a.RegistroCT(context.Background(), m); !errors.Is(err, ct.ErrAutorizacionDenegada) {
			t.Fatalf("catálogo ajeno admitido: %v", err)
		}
	}
}

type autoridadVinculoCapturaPrueba struct{ publicacion domct.PublicacionCategoriaRPT }

func (a *autoridadVinculoCapturaPrueba) ConsultaCT(context.Context, ct.ConsultaVinculoCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrVinculoCategoriaRPTDenegado
}
func (a *autoridadVinculoCapturaPrueba) RegistroCT(context.Context, ct.RegistroVinculoCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrVinculoCategoriaRPTDenegado
}
func (a *autoridadVinculoCapturaPrueba) LecturaRPT(_ context.Context, p domct.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.publicacion = p
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ct.ErrVinculoCategoriaRPTDenegado
}

type fuentesVinculoNoUsadasPrueba struct{}

func (fuentesVinculoNoUsadasPrueba) Consultar(context.Context, ct.ConsultaVinculoCategoriaRPT, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ct.LecturaVinculoCategoriaRPT, error) {
	return ct.LecturaVinculoCategoriaRPT{}, ct.ErrVinculoCategoriaRPTNoDisponible
}
func (fuentesVinculoNoUsadasPrueba) Registrar(context.Context, ct.RegistroVinculoCategoriaRPT, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ct.ReciboVinculoCategoriaRPT, error) {
	return ct.ReciboVinculoCategoriaRPT{}, ct.ErrVinculoCategoriaRPTNoDisponible
}
func (fuentesVinculoNoUsadasPrueba) ConsultarPublicacionCategoriaRPT(context.Context, domct.PublicacionCategoriaRPT, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domct.PublicacionCategoriaRPT, error) {
	return domct.PublicacionCategoriaRPT{}, ct.ErrVinculoCategoriaRPTNoDisponible
}

// La fachada pone organización y catálogo del servidor (el canal no los trae)
// y traduce la negativa del servicio existente a 403 sin crear nada.
func TestVinculoRPTB2FachadaCompletaIntencionYTraduceErrores(t *testing.T) {
	autoridad := &autoridadVinculoCapturaPrueba{}
	servicio, err := appct.NuevoServicioVinculoCategoriaRPT(autoridad, fuentesVinculoNoUsadasPrueba{}, fuentesVinculoNoUsadasPrueba{}, relojFijoVinculoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fachadaVinculoCategoriaRPTB2{organizacionRef: "organizacion:prueba", catalogoID: "categorias_rpt", moduloID: "personal", servicio: servicio}
	m := registroVinculoPrueba("x", "y")
	e := httpct.EntradaVinculoCategoriaRPTB2{ExpedienteRef: m.ExpedienteRef, VersionExpedienteEsperada: m.VersionExpedienteEsperada,
		AnalisisVersion: m.AnalisisVersion, AnalisisReciboRef: m.AnalisisReciboRef, AnalisisHuellaSHA256: m.AnalisisHuellaSHA256,
		CategoriaRef: m.CategoriaRef, CatalogoVersion: m.CatalogoVersion, CatalogoHuellaSHA256: m.CatalogoHuellaSHA256, CategoriaID: m.CategoriaID,
		FuenteRef: m.FuenteRef, MotivoRef: m.MotivoRef, AprobacionRef: m.AprobacionRef, ClaveIdempotencia: m.ClaveIdempotencia}
	if _, err := f.RegistrarVinculo(context.Background(), e); !errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) {
		t.Fatalf("negativa no traducida a 403: %v", err)
	}
	if autoridad.publicacion.CatalogoID != "categorias_rpt" || autoridad.publicacion.ModuloID != "personal" || autoridad.publicacion.CategoriaID != m.CategoriaID {
		t.Fatalf("la publicación no usa el catálogo del servidor: %+v", autoridad.publicacion)
	}
	e.ExpedienteRef = "no-es-un-expediente"
	if _, err := f.RegistrarVinculo(context.Background(), e); !errors.Is(err, httpct.ErrPeticionIncorporacionPersonalB2) {
		t.Fatalf("intención inválida no traducida a 422: %v", err)
	}
	if _, err := f.ConsultarVinculo(context.Background(), "expediente:prueba-1"); !errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) {
		t.Fatalf("consulta denegada no traducida a 403: %v", err)
	}
}

type relojFijoVinculoPrueba struct{}

func (relojFijoVinculoPrueba) Ahora() time.Time { return time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC) }
