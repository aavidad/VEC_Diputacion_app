package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type preparadorUsosB2Prueba struct{ llamadas int }

func (p *preparadorUsosB2Prueba) PrepararReservaUsoCategoriaRPT(context.Context, vp.MaterialReservaUsoCategoriaRPT) (vp.PreparacionAutorizacionUsoCategoriaRPT, error) {
	p.llamadas++
	return vp.PreparacionAutorizacionUsoCategoriaRPT{}, nil
}
func (p *preparadorUsosB2Prueba) PrepararConfirmacionUsoCategoriaRPT(context.Context, vp.MaterialTerminalUsoCategoriaRPT) (vp.PreparacionAutorizacionUsoCategoriaRPT, error) {
	p.llamadas++
	return vp.PreparacionAutorizacionUsoCategoriaRPT{}, nil
}
func (p *preparadorUsosB2Prueba) PrepararCancelacionUsoCategoriaRPT(context.Context, vp.MaterialTerminalUsoCategoriaRPT) (vp.PreparacionAutorizacionUsoCategoriaRPT, error) {
	p.llamadas++
	return vp.PreparacionAutorizacionUsoCategoriaRPT{}, nil
}

func TestIncorporacionB2MaterialAjenoNoPreparaNiEmiteV3(t *testing.T) {
	preparador := &preparadorUsosB2Prueba{}
	a := &autoridadIncorporacionPersonalB2{preparadorUsosRPT: preparador}
	material := vp.MaterialReservaUsoCategoriaRPT{Consumidor: "bolsa", UsoRef: "uso:ajeno"}
	if _, err := a.AutorizarReservaPlanB2(t.Context(), personal.PlanIncorporacionCT{}, material); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("reserva ajena no denegada: %v", err)
	}
	if _, err := a.AutorizarConfirmacionPlanB2(t.Context(), personal.PlanIncorporacionCT{}, vp.MaterialTerminalUsoCategoriaRPT{Reserva: material}); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("confirmación ajena no denegada: %v", err)
	}
	if preparador.llamadas != 0 {
		t.Fatalf("material ajeno alcanzó preparador RPT: %d", preparador.llamadas)
	}
}

func TestIncorporacionB2PlanDeOtroCatalogoOModuloNoLlegaAPreparadorRPT(t *testing.T) {
	d := personal.DatosPlanIncorporacionCT{
		IdempotenciaRef: "10000000-0000-4000-8000-000000000001", OrigenCTRef: "ct:plan", OrigenCTReciboRef: "ct:recibo", OrigenCTHuellaSHA256: strings.Repeat("a", 64),
		ExpedienteRef: "exp:uno", ExpedienteVersion: 7, OrganismoRef: "org:uno", UnidadRef: "uni:uno", PersonaRef: "per_" + strings.Repeat("p", 24), PersonaVersion: 1,
		FuenteBolsaRef: "bolsa:persona", FuenteBolsaVersion: 2, FuenteBolsaReciboRef: "bolsa:recibo", FuenteBolsaHuellaSHA256: strings.Repeat("b", 64),
		Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: "reg:uno", Version: 1}, Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: "mod:uno", Version: 2}, Desde: personal.FechaCivil("2026-10-01"),
		PlazaRef: "plaza:10000000-0000-4000-8000-000000000002", PuestoRef: "puesto:10000000-0000-4000-8000-000000000003", ClaseOcupacion: "temporal",
		VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 2, FuenteOrganizacionRef: "organizacion:uno", FuenteOrganizacionHuellaSHA256: strings.Repeat("c", 64),
		CatalogoRPTID: "rpt:catalogo", CatalogoRPTModulo: "personal", CatalogoRPTCategoria: "categoria:uno", CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("d", 64),
		VinculoCTReciboRef: "vinculo:recibo", Procedencia: personal.ProcedenciaActoEmpleadoB2{ActoRef: "acto:incorporacion", FuenteRef: "fuente:ct", FuenteVersion: 7, FuenteHuellaSHA256: strings.Repeat("e", 64), IdempotenciaRef: "10000000-0000-4000-8000-000000000004"},
	}
	p := personal.PlanIncorporacionCT{
		ClasesOcupacionCatalogoRef: "personal:clases_ocupacion_ct", ClasesOcupacionCatalogoVersion: 1, ClasesOcupacionCatalogoHuellaSHA256: strings.Repeat("f", 64),
		PlanRef: "perplan_" + strings.Repeat("a", 32), ReciboRef: "perplanrec_" + strings.Repeat("b", 32), Version: 1, Datos: d, Modo: "alta_empleado",
		ClaveAltaRelacion: "10000000-0000-4000-8000-000000000005", ClaveOcupacion: "10000000-0000-4000-8000-000000000006", UsoRPTRef: "uso:uno", ReservaRPTRef: "reserva:uno", ConfirmacionRPTRef: "confirmacion:uno",
	}
	for _, caso := range []struct {
		nombre  string
		cambiar func(*personal.PlanIncorporacionCT)
	}{
		{"catalogo", func(p *personal.PlanIncorporacionCT) { p.Datos.CatalogoRPTID = "rpt:otro" }},
		{"modulo", func(p *personal.PlanIncorporacionCT) { p.Datos.CatalogoRPTModulo = "bolsa" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			plan := p
			caso.cambiar(&plan)
			plan.HuellaSHA256 = plan.CalcularHuellaSHA256()
			if err := plan.Validar(); err != nil {
				t.Fatalf("plan de prueba inválido: %v", err)
			}
			m := vp.MaterialReservaUsoCategoriaRPT{Consumidor: "personal", UsoRef: plan.UsoRPTRef, CategoriaID: plan.Datos.CatalogoRPTCategoria,
				Publicacion: vp.ReferenciaPublicacionRPT{CatalogoID: plan.Datos.CatalogoRPTID, Version: int(plan.Datos.CatalogoRPTVersion), HuellaSHA256: plan.Datos.CatalogoRPTHuellaSHA256}, ReservaReciboRef: plan.ReservaRPTRef}
			if !materialReservaCorrespondePlanB2(plan, m) {
				t.Fatal("la prueba no ejercita la guarda del catálogo y módulo")
			}
			preparador := &preparadorUsosB2Prueba{}
			a := &autoridadIncorporacionPersonalB2{preparadorUsosRPT: preparador, catalogoRPTID: p.Datos.CatalogoRPTID, moduloRPTID: p.Datos.CatalogoRPTModulo}
			if _, err := a.AutorizarReservaPlanB2(t.Context(), plan, m); !errors.Is(err, ct.ErrAutorizacionDenegada) {
				t.Fatalf("reserva: %v", err)
			}
			if _, err := a.AutorizarConfirmacionPlanB2(t.Context(), plan, vp.MaterialTerminalUsoCategoriaRPT{Reserva: m, TerminalReciboRef: plan.ConfirmacionRPTRef}); !errors.Is(err, ct.ErrAutorizacionDenegada) {
				t.Fatalf("confirmación: %v", err)
			}
			if preparador.llamadas != 0 {
				t.Fatalf("plan de otro catálogo o módulo alcanzó preparador: %d", preparador.llamadas)
			}
		})
	}
}

func TestIncorporacionB2SoloEmiteParaPreparacionRPTExacta(t *testing.T) {
	const usoRef = "uso:incorporacion:prueba"
	const accion = "vec.catalogos.categorias.reservar_uso"
	d, ok := descriptorIncorporacionB2(accion)
	if !ok {
		t.Fatal("falta la operación de reserva RPT")
	}
	a := &autoridadIncorporacionPersonalB2{
		operaciones:   map[string]operacionAutorizadaIncorporacionB2{accion: {descriptor: d}},
		catalogoRPTID: "categorias_rpt", moduloRPTID: "personal",
	}
	base := vp.PreparacionAutorizacionUsoCategoriaRPT{
		Accion: accion, Finalidad: d.finalidad, AudienciaConsumo: d.audiencia,
		Recurso: core.RecursoAutorizable{
			Referencia: usoRef, ModuloID: "personal", Tipo: "uso_categoria",
			Ambitos:   map[string]string{"catalogo_id": "categorias_rpt", "modulo_id": "personal", "consumidor": "personal"},
			Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)},
		},
	}
	if !a.preparacionUsoRPTValida(accion, usoRef, base) {
		t.Fatal("la preparación exacta fue denegada")
	}
	casos := map[string]func(*vp.PreparacionAutorizacionUsoCategoriaRPT){
		"accion": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) {
			p.Accion = "vec.catalogos.categorias.confirmar_uso"
		},
		"finalidad": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Finalidad = "consultar_categorias_rpt" },
		"audiencia": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) {
			p.AudienciaConsumo = "vec_catalogos_configurables.lectura_categorias.v1"
		},
		"uso":          func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Referencia = "uso:ajeno" },
		"modulo":       func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.ModuloID = "bolsa" },
		"tipo":         func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Tipo = "categoria" },
		"catalogo":     func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Ambitos["catalogo_id"] = "otro" },
		"consumidor":   func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Ambitos["consumidor"] = "bolsa" },
		"ambito_extra": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Ambitos["organismo_ref"] = "otro" },
		"huella": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) {
			p.Recurso.Atributos["material_sha256"] = "invalida"
		},
		"atributo_extra": func(p *vp.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Atributos["otro"] = "dato" },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := base
			p.Recurso.Ambitos = map[string]string{}
			for k, v := range base.Recurso.Ambitos {
				p.Recurso.Ambitos[k] = v
			}
			p.Recurso.Atributos = map[string]string{}
			for k, v := range base.Recurso.Atributos {
				p.Recurso.Atributos[k] = v
			}
			alterar(&p)
			if a.preparacionUsoRPTValida(accion, usoRef, p) {
				t.Fatal("la preparación alterada alcanzaría emisión V3")
			}
		})
	}
}

func TestIncorporacionB2PuraDetalleYTresFronterasNominales(t *testing.T) {
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref: vinculo.PrincipalID, PerfilV3Ref: vinculo.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		UnidadRef:       "unidad:desarrollo:rrhh", ActorRef: vinculo.PrincipalID,
	}
	detalle, err := nuevoPerfilNominalIncorporacion(soporte, refs, claveIncorporacionDetalle, nil, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	nominales := &perfilesNominalesIncorporacion{soporte: soporte, consultas: consultas, detalle: detalle}
	if err := extenderPerfilesNominalesB2(nominales, refs, configuracionB2PuraPrueba().PersonalB2, soporte.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	ct155 := nominales.b2[ct.AccionLeerPlanNominalB2]
	esperadosCT155 := map[string][]string{
		ct.AccionLeerPlanNominalB2:      {"plan"},
		ct.AccionRegistrarPlanNominalB2: {"recibo"},
		ct.AccionConfirmarOrigenB2:      {"recibo"},
	}
	if ct155 == nil || ct155 != nominales.b2[ct.AccionRegistrarPlanNominalB2] || ct155 != nominales.b2[ct.AccionConfirmarOrigenB2] || len(ct155.plantilla.VersionRol.Concesiones) != len(esperadosCT155) {
		t.Fatal("CT155 debe conservar un único perfil con tres acciones exactas")
	}
	for _, concesion := range ct155.plantilla.VersionRol.Concesiones {
		campos, existe := esperadosCT155[concesion.Accion]
		if !existe || !slices.Equal(concesion.CamposPermitidos, campos) || len(concesion.Obligaciones) != 0 {
			t.Fatalf("concesión CT155 divergente del contrato AD3-130: %s campos=%v", concesion.Accion, concesion.CamposPermitidos)
		}
		delete(esperadosCT155, concesion.Accion)
	}
	if len(esperadosCT155) != 0 {
		t.Fatalf("faltan concesiones CT155: %v", esperadosCT155)
	}
	if nominales.legadoCompuesto || nominales.alta != nil || nominales.ct != nil || len(nominales.todos()) != 1+len(gruposPerfilesIncorporacionB2()) {
		t.Fatal("B2 puro compuso perfiles del protocolo anterior o perdió uno propio")
	}
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(vinculo.PerfilActivoRef, []string{vinculo.PerfilActivoRef})
	declaraciones = append(declaraciones, descriptoresFronterasIncorporacionB2Desarrollo()...)
	declaraciones, err = asignarPerfilesNominalesB2EnFronteras(soporte, declaraciones)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(declaraciones)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := catalogo.resolver(http.MethodPost, httpct.RutaConsultaDetalleRRHH)
	if !ok || !esDescriptorDetalleContratacionTemporalDesarrollo(d, vinculo.PerfilActivoRef) || len(d.PerfilesActivosRef) != 2+len(gruposPerfilesIncorporacionB2()) {
		t.Fatal("detalle B2 puro carece de frontera nominal exacta")
	}
	for _, perfil := range nominales.todos() {
		if !d.admitePerfil(perfil.perfilRef()) {
			t.Fatalf("detalle deniega perfil nominal %s", perfil.clave)
		}
	}
	for _, clave := range []string{claveIncorporacionAlta, claveIncorporacionCT} {
		c, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principalCanalNominalIncorporacion(soporte), soporte.reloj.Ahora(), discriminadorPerfilFijoCTDesarrollo(clave))
		if err != nil {
			t.Fatal(err)
		}
		if d.admitePerfil(c.Resultado.Contexto.PerfilActivoRef) {
			t.Fatalf("detalle B2 puro admite perfil legado %s", clave)
		}
	}
	for _, par := range []struct{ metodo, ruta string }{{http.MethodGet, httpct.RutaPlanB2}, {http.MethodPost, httpct.RutaPlanB2}, {http.MethodPost, httpct.RutaConfirmacionB2}} {
		d, ok := catalogo.resolver(par.metodo, par.ruta)
		if !ok || d.admitePerfil(detalle.perfilRef()) || d.admitePerfil(vinculo.PerfilActivoRef) || len(d.PerfilesActivosRef) != len(gruposPerfilesIncorporacionB2()) {
			t.Fatalf("ruta B2 sin perfiles propios cerrados: %s %s", par.metodo, par.ruta)
		}
	}
	rutas := []vechttp.RutaExacta{rutaCoberturaCTPrueba(httpct.RutaPlanB2), rutaCoberturaCTPrueba(httpct.RutaConfirmacionB2)}
	if err := validarCoberturaRutasCTDesarrollo(rutas, catalogo); err != nil {
		t.Fatal(err)
	}
}

func TestIncorporacionB2PerfilPersonalConservaActorYRevocaLectura(t *testing.T) {
	base, ctx, _, publicador := escenarioNominalIncorporacion(t)
	refs := base.referencias
	refs.PerfilV3Ref = base.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	c := &archivoIncorporacionPersonalB2{Protocolo: "personal_b2_v1", OrganismoRef: "organismo:prueba", CatalogoRPTID: "categorias_rpt", ModuloRPTID: "personal"}
	if e := extenderPerfilesNominalesB2(base.nominales, refs, c, base.reloj.Ahora()); e != nil {
		t.Fatal(e)
	}
	for _, p := range base.nominales.todos() {
		p.contextoEsperadoRegistrado = p.contexto.Resultado
		p.sesionOperativa = &sesionNominalIncorporacionPrueba{contexto: p.contexto}
		publicador.publicadas[p.perfilRef()] = instantaneaPublicadaDesarrollo{instantanea: p.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	}
	autoridad := &autoridadIncorporacionPersonalB2{perfiles: base.nominales, reloj: base.reloj}
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1"})
	primero, e := autoridad.ActorPreparacionPlanB2(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, accion := range []string{"personal.plan_incorporacion_ct.consultar", "personal.plan_incorporacion_ct.ejecutar", "personal.plan_incorporacion_ct.confirmar", personal.AccionAltaEmpleadoB2, personal.AccionHechoEmpleadoB2, personal.AccionFichaEmpleadoB2} {
		actual, e := autoridad.actor(ctx, accion)
		if e != nil || !reflect.DeepEqual(actual, primero) {
			t.Fatalf("%s sustituyó actor/cuenta: %v", accion, e)
		}
	}
	perfil := base.nominales.b2[personal.AccionFichaEmpleadoB2]
	revocada := publicador.publicadas[perfil.perfilRef()]
	revocada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	publicador.publicadas[perfil.perfilRef()] = revocada
	if _, e = autoridad.ActorLecturaHechosB2(ctx); !errors.Is(e, ct.ErrAutorizacionDenegada) {
		t.Fatalf("contexto capturado conservó concesión revocada: %v", e)
	}
}
func TestIncorporacionB2GETYPrepararNoAutorizanEfectoPersonal(t *testing.T) {
	for _, metodo := range []string{"GET", "POST"} {
		ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: metodo, ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/plan/v1"})
		for _, accion := range []string{"personal.plan_incorporacion_ct.preparar", "personal.plan_incorporacion_ct.ejecutar", personal.AccionAltaEmpleadoB2, personal.AccionHechoEmpleadoB2, "vec.catalogos.categorias.reservar_uso", "vec.catalogos.categorias.confirmar_uso", ct.AccionConfirmarOrigenB2} {
			if operacionPermitidaEnRutaIncorporacionB2(ctx, accion) {
				t.Fatalf("%s permitió efecto %s", metodo, accion)
			}
		}
		if operacionPermitidaEnRutaIncorporacionB2(ctx, ct.AccionRegistrarPlanNominalB2) != (metodo == "POST") {
			t.Fatal("preparación CT no está ligada al POST")
		}
	}
	if operacionPermitidaEnRutaIncorporacionB2(context.Background(), ct.AccionLeerPlanNominalB2) {
		t.Fatal("falta de frontera concedió lectura")
	}
	if operacionPermitidaEnRutaIncorporacionB2(nil, ct.AccionLeerPlanNominalB2) {
		t.Fatal("contexto ausente concedió lectura")
	}
	ctx, cancelar := context.WithCancel(context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1"}))
	cancelar()
	if operacionPermitidaEnRutaIncorporacionB2(ctx, ct.AccionConfirmarOrigenB2) {
		t.Fatal("petición cancelada conservó autorización de efecto")
	}
}
