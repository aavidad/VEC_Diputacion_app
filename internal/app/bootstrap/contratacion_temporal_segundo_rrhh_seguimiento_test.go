package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestSeleccionSoporteSeguimientoRRHHUsaHojaVerificadaYSujeto(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	original := identidadRRHHOriginalPrueba()
	segundo := original
	segundo.huella = sha256.Sum256([]byte("hoja-segunda"))
	segundo.principal = clonarPrincipalDesarrollo(original.principal)
	segundo.principal.ID = "desarrollo:segunda-rrhh"
	segundo.principal.Attributes["certificate_sha256"] = hex.EncodeToString(segundo.huella[:])
	base := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: original.principal.ID, certificadoSHA256: original.principal.Attributes["certificate_sha256"]}
	adicional := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: segundo.principal.ID, certificadoSHA256: segundo.principal.Attributes["certificate_sha256"]}
	selector := seleccionSoporteSeguimientoRRHH{original: base, adicional: adicional}
	capacidad := func(i identidadCertificadoDesarrollo, ruta, metodo string) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: ruta, metodo: metodo, principal: i.principal})
	}
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		esperado *soporteAltaContratacionTemporalDesarrollo
	}{
		{"original", capacidad(original, httpinterno.RutaNoIncorporaciones, http.MethodPost), base},
		{"adicional", capacidad(segundo, httpinterno.RutaNoIncorporaciones, http.MethodPost), adicional},
		{"adicional en consulta", capacidad(segundo, httpinterno.RutaSeguimientoCese, http.MethodPost), adicional},
		{"adicional en consulta GET", capacidad(segundo, httpinterno.RutaSeguimientoCese, http.MethodGet), nil},
		{"adicional en cese", capacidad(segundo, httpinterno.RutaCesesNombramiento, http.MethodPost), nil},
		{"adicional en GINPIX", capacidad(segundo, httpinterno.RutaConfirmacionesGINPIX, http.MethodPost), nil},
		{"sin capacidad", context.Background(), nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			got, ok := selector.elegir(caso.ctx)
			if got != caso.esperado || ok != (caso.esperado != nil) {
				t.Fatalf("soporte=%p válido=%v", got, ok)
			}
		})
	}
	autoridadOriginal := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: base}}
	autoridadAdicional := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: adicional}}
	autoridadOriginal.adicional = autoridadAdicional
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaNoIncorporaciones, http.MethodPost)) != autoridadAdicional {
		t.Fatal("el consumidor c22 no selecciona el soporte de la segunda hoja")
	}
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaSeguimientoCese, http.MethodPost)) != autoridadAdicional {
		t.Fatal("la lectura exacta no selecciona el soporte de la segunda hoja")
	}
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaSeguimientoCese, http.MethodGet)) != autoridadOriginal {
		t.Fatal("GET de seguimiento selecciona el soporte de la segunda hoja")
	}
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaCesesNombramiento, http.MethodPost)) != autoridadOriginal {
		t.Fatal("la segunda hoja selecciona el soporte en una ruta distinta de c22")
	}
	falsa := segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.ID = original.principal.ID
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones, http.MethodPost)); ok {
		t.Fatal("sujeto original con hoja segunda aceptado")
	}
	falsa = segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.Attributes["certificate_sha256"] = original.principal.Attributes["certificate_sha256"]
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones, http.MethodPost)); ok {
		t.Fatal("hoja original con sujeto segundo aceptada")
	}
}

func TestSegundoRRHHRegistraLecturaExactaAntesDeEntregar(t *testing.T) {
	originalSoporte, _, originalPrincipal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal = clonarPrincipalDesarrollo(principal)
	principal.ID = "certificado_rrhh_segundo"
	huellaSegunda := sha256.Sum256([]byte("segunda-hoja"))
	principal.Attributes["certificate_sha256"] = hex.EncodeToString(huellaSegunda[:])
	soporte.principalID = principal.ID
	soporte.certificadoSHA256 = principal.Attributes["certificate_sha256"]
	soporte.sello = originalSoporte.sello
	contextoSegundo, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	soporte.contexto = contextoSegundo
	soporte.contextoEsperadoRegistrado = soporte.contexto.Resultado
	soporte.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: soporte.contexto}
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(vinculo.PrincipalID,
		vinculo.PerfilActivoRef, soporte.reloj.Ahora(), "seguimiento_cese_ct_desarrollo", "Seguimiento", "seguimiento-cese-ct-desarrollo",
		[]vecdomain.ConcesionRol{{Accion: accionConsultarSeguimientoCeseDesarrollo, ModuloID: ports.ModuloContratacion,
			TipoRecurso: "seguimiento_contratacion_temporal", Finalidades: []string{"gestionar_contratacion_temporal"},
			GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		t.Fatal(err)
	}
	soporte.seguimientoCese = &soporteSeguimientoCeseDesarrollo{instantanea: instantanea}
	registro := soporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	registroLectura := registroLecturaSegundoRRHH{soporte: soporte, durable: registro}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, registroLectura, registroLectura,
		soporte, soporte.reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	segundo := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{
		soporte: soporte, autorizador: servicio}}
	original := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: originalSoporte}, adicional: segundo}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaSeguimientoCese)
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.metodo = http.MethodPost
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if err := original.AutorizarLecturaSeguimiento(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:1"); err != nil {
		t.Fatalf("lectura exacta denegada: %v", err)
	}
	if registro.concesiones != 1 || soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba).publicadas != 1 {
		t.Fatal("la lectura se entregaría sin publicar el ámbito exacto y registrar su decisión")
	}
	if originalSoporte.principalID != originalPrincipal.ID ||
		originalSoporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba).concesiones != 0 {
		t.Fatal("la lectura de la segunda hoja usó el soporte original")
	}
	pendiente := ports.EstadoSeguimientoExpediente{ExpedienteRef: "expediente:1", VersionExpediente: 8,
		FaseClave: domain.FaseNombramiento, EstadoClave: domain.EstadoEnCurso,
		Acreditada: &ports.EstadoIncorporacionAcreditada{PropuestaNoIncorporacion: &ports.EstadoPropuestaNoIncorporacion{}}}
	if revision, err := original.ValidarEntregaSeguimiento(ctx, pendiente); err != nil || !revision {
		t.Fatalf("propuesta pendiente denegada: revisión=%v error=%v", revision, err)
	}
	pendiente.Acreditada.PropuestaNoIncorporacion = nil
	pendiente.Acreditada.NoIncorporacion = &ports.EstadoNoIncorporacion{ResueltaPor: vinculo.PrincipalID}
	if revision, err := original.ValidarEntregaSeguimiento(ctx, pendiente); err != nil || !revision {
		t.Fatalf("recibo propio tras reinicio denegado: revisión=%v error=%v", revision, err)
	}
	pendiente.Acreditada.NoIncorporacion.ResueltaPor = "per_ajena"
	if _, err := original.ValidarEntregaSeguimiento(ctx, pendiente); err == nil {
		t.Fatal("recibo de otra persona accesible")
	}
	pendiente.Acreditada.NoIncorporacion.ResueltaPor = vinculo.PrincipalID
	pendiente.Cese = &ports.EstadoCeseExpediente{}
	if _, err := original.ValidarEntregaSeguimiento(ctx, pendiente); err == nil {
		t.Fatal("cese expuesto en la revisión de no incorporación")
	}
	registro.errConcesion = errors.New("registro durable no disponible")
	if err := original.AutorizarLecturaSeguimiento(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:2"); err == nil {
		t.Fatal("lectura sin confirmación durable")
	}
}

// La asignación del segundo técnico vale tras reiniciar tanto en su forma
// sembrada (solo organización) como ceñida a un expediente por una operación
// anterior; cualquier otra dimensión, organización o concesión se rechaza.
func TestInstantaneaSegundoRRHHAceptaSemillaYAsignacionCenida(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	const principal, perfil = "per_0123456789abcdef0123456789abcdef", "prf_0123456789abcdef0123456789abcdef"
	org := vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}
	exp := vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:ct:prueba"}}
	fase := vecdomain.AmbitoPerfil{Clave: "fase_previa", Valores: []string{string(domain.FaseNombramiento)}}
	estado := vecdomain.AmbitoPerfil{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}}
	concesion := vecdomain.ConcesionRol{Accion: string(domain.AccionRegistrarNoIncorporacion), ModuloID: ports.ModuloContratacion,
		TipoRecurso: ports.TipoRecursoNoIncorporacion, Finalidades: []string{ports.FinalidadRegistrarNoIncorporacion},
		GarantiaMinima: vecdomain.AuthAssuranceHigh}
	lectura := vecdomain.ConcesionRol{Accion: accionConsultarSeguimientoCeseDesarrollo, ModuloID: ports.ModuloContratacion,
		TipoRecurso: "seguimiento_contratacion_temporal", Finalidades: []string{"gestionar_contratacion_temporal"},
		GarantiaMinima: vecdomain.AuthAssuranceHigh}
	nueva := func(ambitos []vecdomain.AmbitoPerfil, concesiones ...vecdomain.ConcesionRol) vecdomain.InstantaneaAutorizacion {
		t.Helper()
		i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
			"seguimiento_prueba", "Seguimiento de prueba", "seguimiento_prueba", concesiones, ambitos)
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	otraOrg := vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{"organizacion:otra"}}
	otraFase := vecdomain.AmbitoPerfil{Clave: "fase_previa", Valores: []string{string(domain.FaseFiscalizacion)}}
	unidad := vecdomain.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}}
	otra := concesion
	otra.Accion = string(domain.AccionRechazarNoIncorporacion)
	for _, caso := range []struct {
		nombre string
		i      vecdomain.InstantaneaAutorizacion
		valida bool
	}{
		{"semilla", nueva([]vecdomain.AmbitoPerfil{org}, concesion, lectura), true},
		{"ceñida", nueva([]vecdomain.AmbitoPerfil{org, exp, fase, estado}, concesion, lectura), true},
		{"otra organización", nueva([]vecdomain.AmbitoPerfil{otraOrg}, concesion, lectura), false},
		{"otra fase", nueva([]vecdomain.AmbitoPerfil{org, exp, otraFase, estado}, concesion, lectura), false},
		{"dimensión ajena", nueva([]vecdomain.AmbitoPerfil{org, unidad}, concesion, lectura), false},
		{"sin organización", nueva([]vecdomain.AmbitoPerfil{exp}, concesion, lectura), false},
		{"sin concesión c22", nueva([]vecdomain.AmbitoPerfil{org}, otra, lectura), false},
		{"sin concesión de lectura", nueva([]vecdomain.AmbitoPerfil{org}, concesion), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := instantaneaSegundoRRHHNoIncorporacionValida(caso.i, principal, perfil, ahora); got != caso.valida {
				t.Fatalf("válida=%v, esperada %v", got, caso.valida)
			}
		})
	}
	if instantaneaSegundoRRHHNoIncorporacionValida(nueva([]vecdomain.AmbitoPerfil{org}, concesion, lectura), "per_otro0000000000000000000000000000", perfil, ahora) {
		t.Fatal("otro actor aceptado")
	}
	base := baseSegundoRRHH(nueva([]vecdomain.AmbitoPerfil{org, exp, fase, estado}, concesion, lectura))
	if len(base.AsignacionPerfil.Ambitos) != 1 || base.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" {
		t.Fatalf("la base conserva el ámbito de una operación anterior: %+v", base.AsignacionPerfil.Ambitos)
	}
}
