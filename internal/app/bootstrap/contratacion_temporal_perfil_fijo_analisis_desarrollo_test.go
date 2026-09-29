package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// escenarioAnalisisPerfilFijoPrueba compone los perfiles fijos, da por
// publicada en la autoridad de prueba la asignación exacta de cada uno y liga
// la sesión del perfil del análisis.
func escenarioAnalisisPerfilFijoPrueba(t *testing.T) (
	*soporteAltaContratacionTemporalDesarrollo, *autorizadorAnalisisContratacionTemporalDesarrollo,
	*autoridadAsignacionesContratacionTemporalDesarrolloPrueba, *perfilFijoCTDesarrollo, dominiovec.Principal,
) {
	t.Helper()
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principal, time.Now().UTC().Truncate(time.Microsecond), nil); err != nil {
		t.Fatal(err)
	}
	autoridad := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	autoridad.asignaciones = map[string]instantaneaPublicadaDesarrollo{}
	for _, p := range s.perfilesFijosRegistrados() {
		autoridad.asignaciones[p.perfilRef()] = instantaneaPublicadaDesarrollo{
			instantanea:    clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla),
			actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo,
		}
	}
	fijo := s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH)
	if fijo == nil {
		t.Fatal("el análisis no tiene perfil fijo")
	}
	s.mu.Lock()
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.mu.Unlock()
	delegado := base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	return s, &autorizadorAnalisisContratacionTemporalDesarrollo{delegado: delegado}, autoridad, fijo, principal
}

func solicitudAnalisisPerfilFijoPrueba(
	t *testing.T, ctx context.Context, fijo *perfilFijoCTDesarrollo, accion, motivo string, ambitos map[string]string,
) dominiovec.SolicitudAutorizacionLigadaV3 {
	t.Helper()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: fijo.contexto.Vinculo,
		ReferenciaMotivo:          referenciaMotivoAutorizacionAnalisisDesarrollo(motivo),
		Accion:                    accion,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: "expediente:analisis:fijo:" + ambitos["_expediente"], ModuloID: ports.ModuloContratacion,
			Tipo: ports.TipoRecursoAnalisis, Ambitos: sinClavesInternasPrueba(ambitos),
			Atributos: map[string]string{ports.AtributoUnidadPoliticaRef: unidadCoberturaContratacionTemporalDesarrollo},
		},
		Finalidad: finalidadAnalisisContratacionTemporalDesarrollo, Correlacion: correlacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func sinClavesInternasPrueba(ambitos map[string]string) map[string]string {
	resultado := make(map[string]string, len(ambitos))
	for clave, valor := range ambitos {
		if clave != "_expediente" {
			resultado[clave] = valor
		}
	}
	return resultado
}

// El análisis se autoriza con su perfil fijo para cualquier expediente de la
// organización en la fase y el estado del catálogo, sin preparar ni publicar
// nada; con el expediente en los ámbitos, fuera de fase, con otra acción o
// con otro motivo se deniega.
func TestAnalisisPerfilFijoConcedeSinExpedienteNiPublicar(t *testing.T) {
	s, autorizador, autoridad, fijo, principal := escenarioAnalisisPerfilFijoPrueba(t)
	valido := func(expediente string) map[string]string {
		return map[string]string{"_expediente": expediente, "organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"fase_previa": "solicitud", "estado_previo": string(domain.EstadoEnCurso)}
	}
	con := func(expediente, clave, valor string) map[string]string {
		a := valido(expediente)
		a[clave] = valor
		return a
	}
	casos := []struct {
		nombre, ruta, accion, motivo string
		ambitos                      map[string]string
		concedida                    bool
	}{
		{"registro_uno", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro", valido("uno"), true},
		{"registro_otro_expediente", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro", valido("dos"), true},
		{"rectificacion", httpinterno.RutaRectificacionAnalisisRRHH, ports.AccionRectificarAnalisis, "rectificacion", valido("uno"), true},
		{"expediente_en_ambitos", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro",
			con("tres", "expediente_ref", "expediente:analisis:fijo:tres"), false},
		{"fase_fuera_de_catalogo", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro",
			con("cuatro", "fase_previa", "asignacion_unidad"), false},
		{"estado_distinto", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro",
			con("cinco", "estado_previo", string(domain.EstadoIncidencia)), false},
		{"otra_organizacion", httpinterno.RutaRegistroAnalisisRRHH, ports.AccionRegistrarAnalisis, "registro",
			con("seis", "organizacion_ref", "organizacion:ajena"), false},
		{"accion_de_registro_en_rectificacion", httpinterno.RutaRectificacionAnalisisRRHH, ports.AccionRegistrarAnalisis, "rectificacion", valido("uno"), false},
		{"motivo_de_registro_en_rectificacion", httpinterno.RutaRectificacionAnalisisRRHH, ports.AccionRectificarAnalisis, "registro", valido("uno"), false},
	}
	concedidas := 0
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, caso.ruta)
			solicitud := solicitudAnalisisPerfilFijoPrueba(t, ctx, fijo, caso.accion, caso.motivo, caso.ambitos)
			decision, _, err := autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, fijo.contexto.Resultado)
			if !caso.concedida {
				if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
					t.Fatalf("no denegada fuera del permiso fijo del análisis: %v", err)
				}
				return
			}
			concedida, _, errResultado := decision.Resultado()
			if err != nil || errResultado != nil || !concedida {
				t.Fatalf("no concedida: %v %v", err, errResultado)
			}
			concedidas++
		})
	}
	registro := s.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	if concedidas != 3 || registro.concesiones != 3 {
		t.Fatalf("concesiones: %d concedidas, %d registradas", concedidas, registro.concesiones)
	}
	// Estas peticiones se deniegan antes de llegar al PDP (la solicitud no
	// casa con el permiso fijo): no hay decisión que registrar.
	if registro.denegaciones != 0 {
		t.Fatalf("%d denegaciones registradas sin decisión del PDP", registro.denegaciones)
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("el análisis preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
	}
	// El canal del análisis es el del perfil fijo, no el dinámico.
	canal, err := s.ResolverContextoCanalAnalisisRRHH(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaRectificacionAnalisisRRHH))
	if err != nil || canal.PerfilRef != fijo.perfilRef() {
		t.Fatalf("el canal del análisis no usa su perfil fijo: %+v %v", canal, err)
	}
	// Otra ruta no obtiene el canal del análisis.
	if _, err := s.ResolverContextoCanalAnalisisRRHH(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes)); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("el alta obtuvo el canal del análisis: %v", err)
	}
}

// Una asignación revocada, o publicada por otro acto, no se consume: el
// análisis se deniega y nada se publica.
func TestAnalisisPerfilFijoNoConsumeAsignacionRevocadaNiAjena(t *testing.T) {
	for _, alterar := range []func(*instantaneaPublicadaDesarrollo){
		func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
		},
		func(p *instantaneaPublicadaDesarrollo) { p.actoAsignacion = actoAsignacionCTDesarrollo },
	} {
		s, autorizador, autoridad, fijo, principal := escenarioAnalisisPerfilFijoPrueba(t)
		publicada := autoridad.asignaciones[fijo.perfilRef()]
		alterar(&publicada)
		autoridad.asignaciones[fijo.perfilRef()] = publicada
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaRegistroAnalisisRRHH)
		solicitud := solicitudAnalisisPerfilFijoPrueba(t, ctx, fijo, ports.AccionRegistrarAnalisis, "registro",
			map[string]string{"_expediente": "uno", "organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"fase_previa": "solicitud", "estado_previo": string(domain.EstadoEnCurso)})
		if _, _, err := autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, fijo.contexto.Resultado); err == nil {
			t.Fatal("concedido con una asignación no consumible")
		}
		if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
			t.Fatalf("preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
		}
	}
}

// Las fases del análisis salen del catálogo (c23); sin entrada rigen las de
// siempre y una entrada mal formada impide arrancar.
func TestFasesOperacionAnalisisDesdeCatalogo(t *testing.T) {
	regla := func(valor, estado string) reglas.Regla {
		return reglas.Regla{Clave: reglas.CTPrefijoFaseOperacion + "analisis", Unidad: reglas.UnidadLista, Valor: valor,
			Atributos: map[string]string{"estado": estado}}
	}
	sin, err := fasesOperacionDesdeReglasCT(nil)
	if err != nil || !sin[operacionFaseAnalisisCT].admite("solicitud", domain.EstadoEnCurso) {
		t.Fatalf("sin catálogo no rige la fase de siempre: %v", err)
	}
	dos, err := fasesOperacionDesdeReglasCT([]reglas.Regla{regla("solicitud,analisis", "en_curso")})
	f := dos[operacionFaseAnalisisCT]
	if err != nil || !f.admite("analisis", domain.EstadoEnCurso) || !f.admite("solicitud", domain.EstadoEnCurso) ||
		f.admite("fiscalizacion", domain.EstadoEnCurso) || f.admite("solicitud", domain.EstadoIncidencia) {
		t.Fatalf("fases del catálogo mal leídas: %+v %v", f, err)
	}
	ambitos := f.ambitosPerfil(organizacionAltaContratacionTemporalDesarrollo)
	if len(ambitos) != 3 || ambitos[1].Clave != "fase_previa" || len(ambitos[1].Valores) != 2 || ambitos[1].Valores[0] != "analisis" {
		t.Fatalf("ámbitos del permiso fijo inesperados: %+v", ambitos)
	}
	for _, mala := range []reglas.Regla{
		regla("", "en_curso"), regla("solicitud", "desconocido"), regla("solicitud,solicitud", "en_curso"),
		regla("Solicitud", "en_curso"), {Clave: reglas.CTPrefijoFaseOperacion + "analisis", Unidad: reglas.UnidadMeses, Valor: "solicitud",
			Atributos: map[string]string{"estado": "en_curso"}},
	} {
		if _, err := fasesOperacionDesdeReglasCT([]reglas.Regla{mala}); err == nil {
			t.Fatalf("regla mal formada admitida: %+v", mala)
		}
	}
	if _, err := fasesOperacionDesdeReglasCT([]reglas.Regla{regla("solicitud", "en_curso"), regla("analisis", "en_curso")}); err == nil {
		t.Fatal("dos entradas para la misma operación admitidas")
	}
	// Un estado propio por fase («estado_<fase>») forma pares exactos.
	pares := reglas.Regla{Clave: reglas.CTPrefijoFaseOperacion + "informe_juridico", Unidad: reglas.UnidadLista,
		Valor: "asignacion_unidad,subsanacion_unidad", Atributos: map[string]string{"estado": "en_curso", "estado_subsanacion_unidad": "incidencia"}}
	conPares, err := fasesOperacionDesdeReglasCT([]reglas.Regla{pares})
	fi := conPares[operacionFaseInformeJuridicoCT]
	if err != nil || !fi.admite("asignacion_unidad", domain.EstadoEnCurso) || !fi.admite("subsanacion_unidad", domain.EstadoIncidencia) ||
		fi.admite("asignacion_unidad", domain.EstadoIncidencia) || fi.admite("subsanacion_unidad", domain.EstadoEnCurso) {
		t.Fatalf("pares del catálogo mal leídos: %+v %v", fi, err)
	}
	huerfano := pares
	huerfano.Atributos = map[string]string{"estado": "en_curso", "estado_fiscalizacion": "incidencia"}
	if _, err := fasesOperacionDesdeReglasCT([]reglas.Regla{huerfano}); err == nil {
		t.Fatal("estado de una fase que no está en la lista admitido")
	}
	var nulas *opcionesAnalisisCTDesarrollo
	if f, ok := nulas.faseOperacionVigente(operacionFaseAnalisisCT); !ok || !f.admite("solicitud", domain.EstadoEnCurso) {
		t.Fatal("sin opciones no rige la fase de siempre")
	}
	if _, ok := nulas.faseOperacionVigente("desconocida"); ok {
		t.Fatal("una operación sin fase quedó admitida")
	}
}
