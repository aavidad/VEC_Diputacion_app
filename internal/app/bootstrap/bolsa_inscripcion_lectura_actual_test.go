package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteLecturaActualInscripcionPrueba struct {
	i                 vecdomain.InstantaneaAutorizacion
	err               error
	llamadas          int
	principal, perfil string
}

type ambitoLecturaRRHHInscripcionPrueba struct {
	llamadas  int
	resultado AmbitoLecturaRRHHInscripcionBolsa
	err       error
}

type ambitoRecursoRRHHInscripcionPrueba struct {
	resultado AmbitoRecursoRRHHInscripcionBolsa
	llamadas  int
}

func (f *ambitoRecursoRRHHInscripcionPrueba) ResolverAmbitoRRHH(_ context.Context, _ contextoSeguridadComunDesarrollo, _ AcreditacionSesionInscripcionBolsa, ref string, captura inscripcion.CapturaLectura) (AmbitoRecursoRRHHInscripcionBolsa, error) {
	f.llamadas++
	if captura.Accion != inscripcion.AccionDetalleRRHH || captura.RecursoRef != ref || captura.AmbitoSolicitud == nil || captura.ConjuntoGestion == nil {
		return AmbitoRecursoRRHHInscripcionBolsa{}, errors.New("captura no ligada")
	}
	return f.resultado, nil
}

func (f *ambitoLecturaRRHHInscripcionPrueba) ResolverAmbitoLecturaRRHH(context.Context, contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, string, string, inscripcion.Filtro) (AmbitoLecturaRRHHInscripcionBolsa, error) {
	f.llamadas++
	return f.resultado, f.err
}

func (f *fuenteLecturaActualInscripcionPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (vecdomain.InstantaneaAutorizacion, error) {
	f.llamadas++
	f.principal, f.perfil = principal, perfil
	return f.i, f.err
}

func descriptorLecturasInscripcionPrueba() map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion {
	resultado := make(map[ClaveOperacionInscripcionBolsa]DescriptorLecturaActualInscripcion, 12)
	for _, clave := range clavesLecturaInscripcionBolsa() {
		tipo, finalidad, ok := tipoFinalidadInscripcionBolsa(clave)
		if !ok {
			panic("clave de prueba desconocida")
		}
		d := DescriptorLecturaActualInscripcion{Accion: clave.Accion, ModuloID: "bolsa",
			TipoRecurso: tipo, Finalidad: finalidad,
			Campos: camposLecturaInscripcionBolsa(clave.Accion, accionRRHHInscripcion(clave.Accion))}
		if clave.Canal == "externa_personal" {
			d.AmbitoVinculoClave, d.AmbitoVinculoTipo = "candidato_ref", vecdomain.TipoReferenciaContextoActorCandidato
		} else if !accionRRHHInscripcion(clave.Accion) {
			d.AmbitoVinculoClave, d.AmbitoVinculoTipo = "empleado_ref", vecdomain.TipoReferenciaContextoActorEmpleado
		}
		resultado[clave] = d
	}
	return resultado
}

func hojasJSONInscripcion(t reflect.Type, prefijo string) []string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return []string{strings.TrimSuffix(prefijo, ".")}
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		return hojasJSONInscripcion(t.Elem(), strings.TrimSuffix(prefijo, ".")+"[].")
	}
	if t.Kind() != reflect.Struct {
		return []string{strings.TrimSuffix(prefijo, ".")}
	}
	var hojas []string
	for i := 0; i < t.NumField(); i++ {
		campo := t.Field(i)
		if !campo.IsExported() {
			continue
		}
		nombre := strings.Split(campo.Tag.Get("json"), ",")[0]
		if nombre == "-" {
			continue
		}
		if nombre == "" {
			nombre = campo.Name
		}
		hojas = append(hojas, hojasJSONInscripcion(campo.Type, prefijo+nombre+".")...)
	}
	return hojas
}

func TestCamposLecturaInscripcionCubrenHojasDTOExactas(t *testing.T) {
	for _, caso := range []struct {
		accion string
		rrhh   bool
		tipo   reflect.Type
	}{
		{inscripcion.AccionListarAbiertas, false, reflect.TypeOf(inscripcion.PaginaAbiertas{})},
		{inscripcion.AccionDetalleAbierta, false, reflect.TypeOf(inscripcion.BolsaAbierta{})},
		{inscripcion.AccionListarPropias, false, reflect.TypeOf(inscripcion.Pagina{})},
		{inscripcion.AccionDetallePropia, false, reflect.TypeOf(inscripcion.Solicitud{})},
		{inscripcion.AccionListarRRHH, true, reflect.TypeOf(inscripcion.Pagina{})},
		{inscripcion.AccionDetalleRRHH, true, reflect.TypeOf(inscripcion.Solicitud{})},
		{inscripcion.AccionConvocatoriasRRHH, true, reflect.TypeOf(inscripcion.PaginaConvocatoriasGestion{})},
		{inscripcion.AccionMotivosRRHH, true, reflect.TypeOf(inscripcion.CatalogoMotivos{})},
	} {
		t.Run(caso.accion, func(t *testing.T) {
			actual := hojasJSONInscripcion(caso.tipo, "")
			if caso.accion == inscripcion.AccionListarPropias || caso.accion == inscripcion.AccionDetallePropia {
				actual = slices.DeleteFunc(actual, func(campo string) bool { return strings.HasSuffix(campo, "persona_resumen") })
			}
			if caso.accion == inscripcion.AccionListarPropias {
				actual = slices.DeleteFunc(actual, func(campo string) bool { return campo == "convocatoria_titulo" })
			}
			slices.Sort(actual)
			esperados := camposLecturaInscripcionBolsa(caso.accion, caso.rrhh)
			if !slices.Equal(actual, esperados) {
				t.Fatalf("DTO/catálogo divergentes: DTO=%q catálogo=%q", actual, esperados)
			}
		})
	}
}

func TestHuellaCamposLecturaInscripcionPorCanal(t *testing.T) {
	type entrada struct {
		Accion string   `json:"accion"`
		Canal  string   `json:"canal"`
		Campos []string `json:"campos"`
	}
	var entradas []entrada
	for _, clave := range clavesLecturaInscripcionBolsa() {
		entradas = append(entradas, entrada{
			Accion: clave.Accion, Canal: clave.Canal, Campos: camposLecturaInscripcionBolsa(clave.Accion, accionRRHHInscripcion(clave.Accion))})
	}
	slices.SortFunc(entradas, func(a, b entrada) int {
		if a.Accion < b.Accion {
			return -1
		}
		if a.Accion > b.Accion {
			return 1
		}
		if a.Canal < b.Canal {
			return -1
		}
		if a.Canal > b.Canal {
			return 1
		}
		return 0
	})
	canon, err := json.Marshal(entradas)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(canon)
	const esperada = "1d11d463fd7d111dea31bc1bfa98ce2be89e15c9a1b52eb6e9255f6e95bff60c"
	if hex.EncodeToString(huella[:]) != esperada {
		t.Fatalf("catálogo de campos cambió: %x", huella)
	}
}

func TestDecisorLecturaActualInscripcionPostgreSQLExigePoolsYDescriptores(t *testing.T) {
	if _, err := NuevoDecisorLecturaActualInscripcionPostgreSQL(nil, nil, ConfiguracionDecisorLecturaActualInscripcion{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("pools ausentes: %v", err)
	}
	f := &fuenteLecturaActualInscripcionPrueba{}
	if _, err := nuevoDecisorLecturaActualInscripcionFuentes(f, f, ConfiguracionDecisorLecturaActualInscripcion{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("descriptores ausentes: %v", err)
	}
}

func TestDecisorLecturaActualInscripcionPermisoVigenteAntesDeBolsa(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, empleadoRef := contextoInscripcionCanalPrueba(t, ahora, false, true)
	a := acreditacionSesionInscripcionPrueba(t, s, "interna_corporativa", ahora)
	v, err := s.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	filtro := inscripcion.Filtro{Limite: 20}
	referencia, err := inscripcion.RecursoLectura(inscripcion.AccionListarPropias, v.PrincipalID, "es", filtro, "")
	if err != nil {
		t.Fatal(err)
	}
	identidad := &identidadCandidatoBolsaDesarrollo{personaRef: v.PrincipalID, perfilRef: v.PerfilActivoRef, candidatoRef: "candidato_prueba_inscripcion"}
	i, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.PublicadaPor = "seguridad:prueba-publicada"
	i.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:prueba-publicada"
	i.AsignacionPerfil.EmitidaPor = "identidad:prueba-publicada"
	i.VersionRol.Concesiones = []vecdomain.ConcesionRol{{Accion: inscripcion.AccionListarPropias,
		ModuloID: "bolsa", TipoRecurso: "solicitud_inscripcion_empleado", Finalidades: []string{"consulta_inscripcion_propia"},
		CamposPermitidos: camposLecturaInscripcionBolsa(inscripcion.AccionListarPropias, false), GarantiaMinima: vecdomain.AuthAssuranceHigh}}
	i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{{Clave: "empleado_ref", Valores: []string{empleadoRef}}}
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteLecturaActualInscripcionPrueba{i: i}
	otraFuente := &fuenteLecturaActualInscripcionPrueba{err: errors.New("fuente equivocada")}
	d, err := nuevoDecisorLecturaActualInscripcionFuentes(otraFuente, fuente,
		ConfiguracionDecisorLecturaActualInscripcion{Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
			Descriptores: descriptorLecturasInscripcionPrueba(), RRHHNominal: []identidadConsultaRRHHDesarrollo{{perfilRef: "prf_rrhh_prueba",
				identidad: identidadCertificadoDesarrollo{principal: vecdomain.Principal{Attributes: map[string]string{
					"certificate_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro)
	if err != nil || !decision.Concedida || decision.CertificadoHuellaSHA256 != a.CertificadoHuellaSHA256 ||
		decision.RevisionPermisos == 0 || !revisionHuellaLecturaInscripcionCoincide(decision.RevisionPermisos, decision.HuellaInstantaneaSHA256) ||
		decision.CorrelacionRef == "" || decision.RecursoRef != referencia ||
		decision.Filtro != filtro || !decision.ValidaHasta.After(ahora) || decision.ValidaHasta.After(ahora.Add(30*time.Second)) {
		t.Fatalf("lectura actual: %+v %v", decision, err)
	}
	if fuente.llamadas != 1 || otraFuente.llamadas != 0 || fuente.principal != v.PrincipalID || fuente.perfil != v.PerfilActivoRef {
		t.Fatalf("fuente seleccionada: interna=%d externa=%d", fuente.llamadas, otraFuente.llamadas)
	}
	getterRRHH := &ambitoLecturaRRHHInscripcionPrueba{}
	d.ambitoRRHH = getterRRHH
	refRRHH := "solicitud_inscripcion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionDetalleRRHH, refRRHH, inscripcion.Filtro{}); !errors.Is(err, inscripcion.ErrAccesoDenegado) || getterRRHH.llamadas != 0 {
		t.Fatalf("empleado accedió a getter RRHH: llamadas=%d err=%v", getterRRHH.llamadas, err)
	}
	if _, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarRRHH, "inscripciones_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", inscripcion.Filtro{Limite: 20}); !errors.Is(err, inscripcion.ErrSolicitudInvalida) || getterRRHH.llamadas != 0 {
		t.Fatalf("bandeja RRHH sin convocatoria consultó getter: llamadas=%d err=%v", getterRRHH.llamadas, err)
	}
	contenidoCambiado := i
	contenidoCambiado.VersionRol.Nombre = "Publicacion de prueba con contenido distinto"
	if contenidoCambiado.Validar() != nil {
		t.Fatal("fixture de rol cambiado invalida")
	}
	fuente.i = contenidoCambiado
	otraDecision, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro)
	if err != nil || otraDecision.HuellaInstantaneaSHA256 == decision.HuellaInstantaneaSHA256 ||
		otraDecision.RevisionPermisos == decision.RevisionPermisos {
		t.Fatalf("contenido cambiado con mismas refs/revisiones no cambio huella: %v", err)
	}
	fuente.i = i
	// La identidad inválida corta antes incluso de obtener la instantánea.
	for nombre, cambiar := range map[string]func(*AcreditacionSesionInscripcionBolsa){
		"certificado":     func(x *AcreditacionSesionInscripcionBolsa) { x.CertificadoHuellaSHA256 = "no_canonica" },
		"sesion revocada": func(x *AcreditacionSesionInscripcionBolsa) { x.ValidaHasta = ahora },
		"perfil":          func(x *AcreditacionSesionInscripcionBolsa) { x.PerfilRef = "prf_ajeno" },
		"canal":           func(x *AcreditacionSesionInscripcionBolsa) { x.Canal = "externa_personal" },
	} {
		t.Run(nombre, func(t *testing.T) {
			mutada := a
			cambiar(&mutada)
			fuente.llamadas = 0
			if _, err := d.DecidirLecturaActual(context.Background(), s, mutada, inscripcion.AccionListarPropias, referencia, filtro); err == nil || fuente.llamadas != 0 {
				t.Fatalf("identidad invalida alcanzó fuente: %d %v", fuente.llamadas, err)
			}
		})
	}
	for nombre, cambiar := range map[string]func(*vecdomain.InstantaneaAutorizacion){
		"permiso ausente": func(x *vecdomain.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones = []vecdomain.ConcesionRol{{Accion: "otra.accion", ModuloID: "bolsa", TipoRecurso: "solicitud_inscripcion_empleado", Finalidades: []string{"consulta_inscripcion_propia"}, CamposPermitidos: camposLecturaInscripcionBolsa(inscripcion.AccionListarPropias, false), GarantiaMinima: vecdomain.AuthAssuranceHigh}}
		},
		"ambito ajeno": func(x *vecdomain.InstantaneaAutorizacion) {
			x.AsignacionPerfil.Ambitos[0].Valores = []string{referenciaAltaContratacionTemporalDesarrollo("emp_", "otro-empleado")}
		},
		"campos extra": func(x *vecdomain.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[0].CamposPermitidos = append(camposLecturaInscripcionBolsa(inscripcion.AccionListarPropias, false), "dni")
		},
		"perfil revocado": func(x *vecdomain.InstantaneaAutorizacion) {
			x.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			x.AsignacionPerfil.RevocadaEn = ahora
			x.AsignacionPerfil.RevocadaPor = "revocador_prueba"
			x.AsignacionPerfil.RevocacionRef = "revocacion_prueba"
		},
		"perfil cambiado": func(x *vecdomain.InstantaneaAutorizacion) { x.AsignacionPerfil.PerfilActivoRef = "prf_ajeno" },
	} {
		t.Run(nombre, func(t *testing.T) {
			mutada := i
			mutada.VersionRol.Concesiones = append([]vecdomain.ConcesionRol(nil), i.VersionRol.Concesiones...)
			mutada.AsignacionPerfil.Ambitos = append([]vecdomain.AmbitoPerfil(nil), i.AsignacionPerfil.Ambitos...)
			cambiar(&mutada)
			fuente.i, fuente.llamadas = mutada, 0
			if _, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) || fuente.llamadas != 1 {
				t.Fatalf("concesión inválida llegó a Bolsa: fuente=%d err=%v", fuente.llamadas, err)
			}
		})
	}
}

func TestDecisorLecturaRRHHComparaConjuntoActualEHistoria(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, _ := contextoInscripcionCanalPrueba(t, ahora, false, false)
	a := acreditacionSesionInscripcionPrueba(t, s, "interna_corporativa", ahora)
	v, err := s.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ref := "solicitud_inscripcion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	unidad, ambito := "unidad_prueba", "ambito_prueba"
	identidad := &identidadCandidatoBolsaDesarrollo{personaRef: v.PrincipalID, perfilRef: v.PerfilActivoRef, candidatoRef: "candidato_prueba_inscripcion"}
	i, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.PublicadaPor = "seguridad:prueba-publicada"
	i.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:prueba-publicada"
	i.AsignacionPerfil.EmitidaPor = "identidad:prueba-publicada"
	i.VersionRol.Concesiones = []vecdomain.ConcesionRol{{Accion: inscripcion.AccionDetalleRRHH,
		ModuloID: "bolsa", TipoRecurso: "solicitud_inscripcion", Finalidades: []string{"consulta_inscripcion_rrhh"},
		CamposPermitidos: camposLecturaInscripcionBolsa(inscripcion.AccionDetalleRRHH, true), GarantiaMinima: vecdomain.AuthAssuranceHigh}}
	i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{{Clave: "ambito_ref", Valores: []string{ambito}}, {Clave: "unidad_ref", Valores: []string{unidad}}}
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	getter := &ambitoLecturaRRHHInscripcionPrueba{resultado: AmbitoLecturaRRHHInscripcionBolsa{RecursoRef: ref,
		ConjuntoGestion: ConjuntoGestionRRHHInscripcionBolsa{ConjuntoRef: "conjunto_prueba", UnidadRef: unidad, AmbitoRef: ambito,
			FuenteRef: "fuente_actual", FuenteVersion: 2, FuenteHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		AmbitoSolicitud: &AmbitoSolicitudRRHHInscripcionBolsa{SolicitudRef: ref, UnidadRef: unidad, AmbitoRef: ambito,
			FuenteRef: "fuente_historica", FuenteVersion: 1, FuenteHuellaSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}}
	fuente := &fuenteLecturaActualInscripcionPrueba{i: i}
	d, err := nuevoDecisorLecturaActualInscripcionFuentes(&fuenteLecturaActualInscripcionPrueba{}, fuente,
		ConfiguracionDecisorLecturaActualInscripcion{Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
			Descriptores: descriptorLecturasInscripcionPrueba(), AmbitoRRHH: getter,
			RRHHNominal: []identidadConsultaRRHHDesarrollo{{perfilRef: v.PerfilActivoRef,
				identidad: identidadCertificadoDesarrollo{principal: vecdomain.Principal{Attributes: map[string]string{"certificate_sha256": a.CertificadoHuellaSHA256}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionDetalleRRHH, ref, inscripcion.Filtro{})
	if err != nil || !decision.Concedida || decision.ConjuntoGestion == nil || decision.AmbitoSolicitud == nil ||
		decision.ConjuntoGestion.FuenteVersion != 2 || decision.AmbitoSolicitud.FuenteVersion != 1 ||
		getter.llamadas != 1 || fuente.llamadas != 1 {
		t.Fatalf("conjunto actual/historia: %+v getter=%d fuente=%d err=%v", decision, getter.llamadas, fuente.llamadas, err)
	}
	autoridad := &autoridadNominalInscripcionBolsa{c: ConfiguracionAutoridadInscripcionBolsa{
		Lectura: d, Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
		Lecturas: map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa{
			{inscripcion.AccionDetalleRRHH, "interna_corporativa"}: {Accion: inscripcion.AccionDetalleRRHH,
				Finalidad: "consulta_inscripcion_rrhh", Campos: camposLecturaInscripcionBolsa(inscripcion.AccionDetalleRRHH, true)},
		}}}
	captura, err := autoridad.CapturarLectura(context.Background(), s, a, inscripcion.AccionDetalleRRHH, ref, inscripcion.Filtro{})
	if err != nil || captura.ConjuntoGestion == nil || captura.AmbitoSolicitud == nil ||
		captura.ConjuntoGestion.FuenteVersion != 2 || captura.AmbitoSolicitud.FuenteVersion != 1 ||
		!slices.Equal(captura.Campos, camposLecturaInscripcionBolsa(inscripcion.AccionDetalleRRHH, true)) {
		t.Fatalf("captura perdió conjunto/historia/campos: %+v err=%v", captura, err)
	}
	getterEscritura := &ambitoRecursoRRHHInscripcionPrueba{resultado: AmbitoRecursoRRHHInscripcionBolsa{
		SolicitudRef: ref, UnidadRef: unidad, AmbitoRef: ambito, FuenteRef: "fuente_historica", FuenteVersion: 1,
		FuenteHuellaSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AuditoriaRef: "auditoria_prueba"}}
	autoridad.c.AmbitoRRHH = getterEscritura
	autoridad.rrhhNominal = []identidadRRHHInscripcionBolsa{{perfilRef: v.PerfilActivoRef, certificadoSHA256: a.CertificadoHuellaSHA256}}
	ctxRecurso, err := autoridad.ambitosEscrituraInscripcion(context.Background(), s, a, inscripcion.AccionDecidir, ref, ahora)
	if err != nil || getterEscritura.llamadas != 1 || ctxRecurso.ConjuntoGestion == nil || ctxRecurso.AmbitoSolicitud == nil ||
		ctxRecurso.ConjuntoGestion.FuenteVersion != 2 || ctxRecurso.AmbitoSolicitud.FuenteVersion != 1 ||
		ctxRecurso.Ambitos["unidad_ref"] != unidad || ctxRecurso.Ambitos["ambito_ref"] != ambito {
		t.Fatalf("fuentes de recurso V3: %+v llamadas=%d err=%v", ctxRecurso, getterEscritura.llamadas, err)
	}
	getterEscritura.resultado.FuenteHuellaSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := autoridad.ambitosEscrituraInscripcion(context.Background(), s, a, inscripcion.AccionDecidir, ref, ahora); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("fuente histórica cambiada aceptada: %v", err)
	}
	getter.resultado.AmbitoSolicitud.AmbitoRef = "otro_ambito"
	fuente.llamadas = 0
	if _, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionDetalleRRHH, ref, inscripcion.Filtro{}); !errors.Is(err, inscripcion.ErrAccesoDenegado) || fuente.llamadas != 0 {
		t.Fatalf("historia de otro ámbito llegó a AID: fuente=%d err=%v", fuente.llamadas, err)
	}
}
