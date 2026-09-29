package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// soporteFiscalizacionPerfilFijoPrueba compone el soporte de Intervención con
// su perfil fijo y una autoridad de prueba que ya tiene publicada la plantilla.
func soporteFiscalizacionPerfilFijoPrueba(t *testing.T) (
	*soporteFiscalizacionContratacionTemporalDesarrollo, *autoridadAsignacionesContratacionTemporalDesarrolloPrueba,
) {
	t.Helper()
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v, err := base.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	plantilla, err := nuevaInstantaneaAutorizacionFiscalizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef,
		time.Now().UTC().Truncate(time.Microsecond), faseFiscalizacionPrueba)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadAsignacionesContratacionTemporalDesarrolloPrueba{asignaciones: map[string]instantaneaPublicadaDesarrollo{
		v.PerfilActivoRef: {instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(plantilla),
			actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo},
	}}
	puente := &soporteAltaContratacionTemporalDesarrollo{contexto: base.contexto, reloj: base.reloj, autoridadAsignaciones: autoridad}
	fijo := &perfilFijoCTDesarrollo{clave: operacionFaseFiscalizacionCT, contexto: base.contexto, plantilla: plantilla,
		rutas: map[string]struct{}{httpinterno.RutaResultadosFiscalizacion: {}}, propioDelSoporte: true}
	return &soporteFiscalizacionContratacionTemporalDesarrollo{fijo: fijo, puente: puente, fase: faseFiscalizacionPrueba,
		reloj: base.reloj, contexto: base.contexto}, autoridad
}

// La fiscalización consume el perfil fijo de Intervención para cualquier
// expediente en una fase del catálogo, sin preparar ni publicar; deniega
// fuera del catálogo, con el expediente en los ámbitos o si la asignación
// está revocada.
func TestFiscalizacionPerfilFijoConsumeSinPublicar(t *testing.T) {
	s, autoridad := soporteFiscalizacionPerfilFijoPrueba(t)
	pedir := func(datos dominiovec.DatosSolicitudAutorizacionLigadaV3) (dominiovec.InstantaneaAutorizacion, bool) {
		return s.instantaneaParaContexto(context.WithValue(context.Background(),
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos))
	}
	for _, expediente := range []string{"expediente:fiscalizacion:uno", "expediente:fiscalizacion:dos"} {
		datos := datosAutorizacionFiscalizacionDesarrollo(domain.FaseInformeJuridico, domain.EstadoEnCurso, map[string]string{})
		datos.Recurso.Referencia = expediente
		consumida, ok := pedir(datos)
		if !ok || !consumida.AsignacionPerfil.Cubre(datos.Recurso) || consumida.AsignacionPerfil.PerfilActivoRef != s.fijo.perfilRef() {
			t.Fatalf("%s: no se consumió el perfil fijo de Intervención", expediente)
		}
	}
	ajena := datosAutorizacionFiscalizacionDesarrollo(domain.FaseNombramiento, domain.EstadoEnCurso, map[string]string{})
	conExpediente := datosAutorizacionFiscalizacionDesarrollo(domain.FaseInformeJuridico, domain.EstadoEnCurso, map[string]string{})
	conExpediente.Recurso.Ambitos["expediente_ref"] = conExpediente.Recurso.Referencia
	for _, datos := range []dominiovec.DatosSolicitudAutorizacionLigadaV3{ajena, conExpediente} {
		if _, ok := pedir(datos); ok {
			t.Fatalf("concedida fuera del permiso fijo: %+v", datos.Recurso.Ambitos)
		}
	}
	publicada := autoridad.asignaciones[s.fijo.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	autoridad.asignaciones[s.fijo.perfilRef()] = publicada
	if _, ok := pedir(datosAutorizacionFiscalizacionDesarrollo(domain.FaseInformeJuridico, domain.EstadoEnCurso, map[string]string{})); ok {
		t.Fatal("se consumió una asignación revocada")
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("se preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
	}
	// La plantilla no lleva expediente y cubre los tres orígenes.
	ambitos := map[string]int{}
	for _, a := range s.fijo.plantilla.AsignacionPerfil.Ambitos {
		ambitos[a.Clave] = len(a.Valores)
	}
	if len(ambitos) != 3 || ambitos["fase_previa"] != 3 || ambitos["estado_previo"] != 2 {
		t.Fatalf("ámbitos del perfil de Intervención inesperados: %+v", ambitos)
	}
}
