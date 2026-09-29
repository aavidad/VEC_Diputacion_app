package bootstrap

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func escenarioPerfilesFijosPrueba(t *testing.T) (*soporteAltaContratacionTemporalDesarrollo, *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	origen := origenEntregaPerfilFijoPrueba(t)
	s.origen = origen
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principal, time.Now().UTC().Truncate(time.Microsecond), origen); err != nil {
		t.Fatal(err)
	}
	autoridad, ok := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if !ok {
		t.Fatal("escenario sin autoridad de prueba")
	}
	return s, autoridad
}

func TestEntregaSinCatalogoNoComponePermisoFijo(t *testing.T) {
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principal,
		time.Now().UTC().Truncate(time.Microsecond), nil); err != nil {
		t.Fatal(err)
	}
	if s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodPost) != nil ||
		s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet) != nil {
		t.Fatal("sin fuente de catálogo se perfilaron permisos de entrega")
	}
	if _, err := nuevaInstantaneaAutorizacionEntregaPeticionDesarrollo(
		s.principalID, s.contexto.Resultado.Contexto.PerfilActivoRef,
		time.Now().UTC().Truncate(time.Microsecond), nil); err == nil {
		t.Fatal("la plantilla de entrega aceptó el fallback sintético")
	}
}

func origenEntregaPerfilFijoPrueba(t *testing.T) *origenConsultasContratacionTemporalDesarrollo {
	t.Helper()
	catalogo, err := nuevoCatalogoDesarrollo(
		filepath.Join("..", "..", "..", "data", "catalogos", "estructura-organizativa", "v1.rpt-publica.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	return nuevoOrigenConsultasConCatalogoDesarrollo(catalogo)
}

func TestPerfilFijoEntregaSoloSeleccionadoPorMetodoDeCapacidad(t *testing.T) {
	s, autoridad := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	fijo := s.perfilFijoParaRuta(rutaEntregaPeticionCentro)
	lector := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if fijo == nil || len(fijo.plantilla.VersionRol.Concesiones) != 2 ||
		fijo.plantilla.VersionRol.Concesiones[0].Accion != ports.AccionEntregarPeticionRRHH ||
		fijo.plantilla.VersionRol.Concesiones[1].Accion != ports.AccionCrearSolicitud ||
		len(fijo.plantilla.AsignacionPerfil.Ambitos) != 3 ||
		lector == nil || lector == fijo || len(lector.plantilla.VersionRol.Concesiones) != 1 ||
		lector.plantilla.VersionRol.Concesiones[0].Accion != ports.AccionConsultarPeticionesRRHH ||
		len(lector.plantilla.AsignacionPerfil.Ambitos) != 1 {
		t.Fatal("perfil de entrega sin acciones o ámbitos cerrados")
	}
	recursoGET := vecdomain.RecursoAutorizable{Referencia: "peticiones:centro:rrhh",
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoEntregaPeticionCentro,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo}}
	recursoPOST := vecdomain.RecursoAutorizable{Referencia: "peticion:centro:001",
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoEntregaPeticionCentro,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"centro_ref": "centro-520", "categoria_ref": categoriaAltaContratacionTemporalDesarrollo}}
	if !lector.plantilla.AsignacionPerfil.Cubre(recursoGET) ||
		lector.plantilla.AsignacionPerfil.Cubre(recursoPOST) ||
		!fijo.plantilla.AsignacionPerfil.Cubre(recursoPOST) ||
		fijo.plantilla.AsignacionPerfil.Cubre(recursoGET) {
		t.Fatal("la geometría de ámbitos cruzó GET y POST")
	}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: s.sello, ruta: rutaEntregaPeticionCentro,
		principal: principal, metodo: http.MethodGet}
	ctxGET := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if s.perfilFijoParaContexto(ctxGET, rutaEntregaPeticionCentro) != lector {
		t.Fatal("GET tomó el perfil del POST")
	}
	ctxFalso := context.WithValue(ctxGET, claveFronteraSeguridadComunDesarrollo{},
		fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: rutaEntregaPeticionCentro})
	if s.perfilFijoParaContexto(ctxFalso, rutaEntregaPeticionCentro) != lector {
		t.Fatal("un método de contexto libre sustituyó la capacidad sellada")
	}
	capacidad.metodo = http.MethodPost
	ctxPOST := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if s.perfilFijoParaContexto(ctxPOST, rutaEntregaPeticionCentro) != fijo {
		t.Fatal("POST no eligió su perfil fijo")
	}
	if _, ok := s.instantaneaParaContexto(ctxPOST, rutaEntregaPeticionCentro); ok {
		t.Fatal("POST sin asignación publicada recibió permiso")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatal("POST preparó o publicó permisos por petición")
	}
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	fronteras := []descriptorFronteraComunDesarrollo{
		{Metodo: http.MethodGet, Ruta: rutaEntregaPeticionCentro, PerfilesActivosRef: []string{dinamico}},
		{Metodo: http.MethodPost, Ruta: rutaEntregaPeticionCentro, PerfilesActivosRef: []string{dinamico}},
	}
	asignadas, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, fronteras)
	if err != nil || asignadas[0].PerfilesActivosRef[0] != lector.perfilRef() ||
		asignadas[1].PerfilesActivosRef[0] != fijo.perfilRef() {
		t.Fatalf("GET/POST cruzaron perfiles: %v, %+v", err, asignadas)
	}
}

func TestPerfilEntregaUsaCentroOriginalDeOrganizacion(t *testing.T) {
	catalogos, err := origenEntregaPerfilFijoPrueba(t).catalogosAlta()
	if err != nil {
		t.Fatal(err)
	}
	directo, original := false, false
	for _, centro := range catalogos.Centros {
		if centro.Referencia == "centro:rpt:520" {
			directo = true
		}
	}
	for _, centro := range catalogos.centrosOrganizacion {
		if centro == "centro-520" {
			original = true
		}
	}
	if !directo || !original {
		t.Fatal("catálogo de alta y organización perdieron sus referencias distintas")
	}
	s, _ := escenarioPerfilesFijosPrueba(t)
	fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodPost)
	if fijo == nil {
		t.Fatal("sin perfil POST")
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: "peticion:centro:prueba",
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoEntregaPeticionCentro,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"centro_ref": "centro-520", "categoria_ref": categoriaAltaContratacionTemporalDesarrollo}}
	if !fijo.plantilla.AsignacionPerfil.Cubre(recurso) {
		t.Fatal("centro original de organización no cubierto")
	}
	for _, ajeno := range []string{"centro:rpt:520", "centro:ajeno"} {
		recurso.Ambitos["centro_ref"] = ajeno
		if fijo.plantilla.AsignacionPerfil.Cubre(recurso) {
			t.Fatalf("perfil de entrega amplió centro a %s", ajeno)
		}
	}
}

// Las rutas con perfil fijo nunca preparan ni publican; sin la autoridad
// PostgreSQL que consume la asignación publicada, se deniegan.
func TestRutasPerfilFijoNuncaPreparanNiPublican(t *testing.T) {
	s, autoridad := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	for _, ruta := range []string{httpinterno.RutaAltaSolicitudes, httpinterno.RutaPropuestaCobertura,
		httpinterno.RutaRegistroAnalisisRRHH, httpinterno.RutaRectificacionAnalisisRRHH,
		httpinterno.RutaAsignaciones, httpinterno.RutaPreparacionesInformeJuridico,
		httpinterno.RutaDecisionCobertura, httpinterno.RutaRectificacionCobertura, httpinterno.RutaResultadoCobertura} {
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
		if _, ok := s.instantaneaParaContexto(ctx, ruta); ok {
			t.Fatalf("%s: concedida sin asignación publicada consumible", ruta)
		}
		if ruta == httpinterno.RutaDecisionCobertura || ruta == httpinterno.RutaRectificacionCobertura {
			_ = s.publicarInstantaneaDecisionCobertura(ctx, ruta)
		}
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("un perfil fijo preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
	}
}

func TestPerfilesFijosSeparanPerfilYRutas(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	alta, cobertura := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes), s.perfilFijoParaRuta(httpinterno.RutaPropuestaCobertura)
	if alta == nil || cobertura == nil || alta == cobertura ||
		alta.perfilRef() == dinamico || cobertura.perfilRef() == dinamico ||
		alta.contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef ||
		alta.contexto.Resultado.Contexto.Instantanea.CuentaRef != s.contexto.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("perfiles fijos sin separar o de otra persona")
	}
	analisis := s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH)
	if analisis == nil || analisis != s.perfilFijoParaRuta(httpinterno.RutaRectificacionAnalisisRRHH) ||
		analisis == alta || analisis == cobertura || analisis.perfilRef() == dinamico {
		t.Fatal("el análisis no tiene su propio perfil fijo")
	}
	asignacion, informe := s.perfilFijoParaRuta(httpinterno.RutaAsignaciones), s.perfilFijoParaRuta(httpinterno.RutaPreparacionesInformeJuridico)
	vistos := map[*perfilFijoCTDesarrollo]bool{alta: true, cobertura: true, analisis: true}
	for _, p := range []*perfilFijoCTDesarrollo{asignacion, informe} {
		if p == nil || vistos[p] || p.perfilRef() == dinamico {
			t.Fatal("la asignación y el informe no tienen cada uno su perfil fijo")
		}
		vistos[p] = true
	}
	for _, ruta := range []string{httpinterno.RutaSubsanacionReparos,
		httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		if s.perfilFijoParaRuta(ruta) != nil {
			t.Fatalf("%s no debe usar un perfil fijo", ruta)
		}
	}
	if s.perfilFijoParaRuta(rutaEntregaPeticionCentro) == nil {
		t.Fatal("POST de entrega sin perfil fijo")
	}
	// Una ruta ya asignada no puede pasar a otro perfil.
	if err := s.registrarPerfilFijoCTDesarrollo(&perfilFijoCTDesarrollo{clave: "otro", plantilla: alta.plantilla,
		rutas: map[string]struct{}{httpinterno.RutaAltaSolicitudes: {}}}); err == nil {
		t.Fatal("se registró otro perfil para una ruta ya asignada")
	}
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo(dinamico, []string{dinamico})
	asignadas, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range asignadas {
		fijo := s.perfilFijoParaRuta(d.Ruta)
		switch {
		case fijo != nil && (len(d.PerfilesActivosRef) != 1 || d.PerfilesActivosRef[0] != fijo.perfilRef()):
			t.Fatalf("%s: la frontera no admite solo su perfil fijo", d.Ruta)
		case fijo == nil && len(d.PerfilesActivosRef) == 1 && d.PerfilesActivosRef[0] != dinamico:
			t.Fatalf("%s: una ruta dinámica cambió de perfil", d.Ruta)
		}
	}
	// Una frontera que no esté en el perfil dinámico no se reasigna.
	fronteras[0].PerfilesActivosRef = []string{"prf_ajeno"}
	for i := range fronteras {
		if fronteras[i].Ruta == httpinterno.RutaAltaSolicitudes {
			fronteras[i].PerfilesActivosRef = []string{"prf_ajeno"}
		}
	}
	if _, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, fronteras); err == nil {
		t.Fatal("se reasignó una frontera que no era del perfil dinámico")
	}
}

// El contexto operativo de una ruta fija usa la sesión de su perfil; sin
// ella, deniega (nunca cae a la sesión del perfil dinámico).
func TestContextoOperativoPerfilFijoNoCaeAlDinamico(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes)
	if _, err := s.contextoOperativoDesarrollo(ctx); err == nil {
		t.Fatal("el alta resolvió contexto sin la sesión de su perfil fijo")
	}
	fijo := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	s.mu.Lock()
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.mu.Unlock()
	contexto, err := s.contextoOperativoDesarrollo(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes))
	if err != nil || contexto.Resultado.Contexto.PerfilActivoRef != fijo.perfilRef() {
		t.Fatalf("el alta no usa el contexto de su perfil fijo: %v", err)
	}
	_ = context.Background()
}
