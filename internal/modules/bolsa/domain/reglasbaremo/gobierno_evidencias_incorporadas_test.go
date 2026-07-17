package reglasbaremo

import (
	"testing"
	"time"
)

func TestVersionPredicaEvidenciasIncorporadasSinConfundirOtraValida(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	aprobacion := aprobacionGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(2*time.Minute))
	publicada, err := borrador.Publicar(
		borrador.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"),
		aprobacion, instanteBaseGobiernoPrueba.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	otraAprobacion := aprobacion.clonar()
	otraAprobacion.atestacion = referenciaPrueba(t, "atestacion:aprobacion:otra", 1, '9')
	if !publicada.IncorporaAprobacionExacta(aprobacion) ||
		publicada.IncorporaAprobacionExacta(otraAprobacion) ||
		borrador.IncorporaAprobacionExacta(aprobacion) {
		t.Fatal("la publicacion no distinguio su aprobacion exacta")
	}

	dependencias := dependenciasGobiernoPrueba(t, publicada, instanteBaseGobiernoPrueba.Add(4*time.Minute))
	activa, err := publicada.Activar(
		publicada.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "activacion"),
		dependencias, instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	otrasDependencias := dependencias.clonar()
	otrasDependencias.atestacion = referenciaPrueba(t, "atestacion:dependencias:otra", 1, '8')
	if !activa.IncorporaDependenciasExactas(dependencias) ||
		activa.IncorporaDependenciasExactas(otrasDependencias) ||
		publicada.IncorporaDependenciasExactas(dependencias) {
		t.Fatal("la activacion no distinguio sus dependencias exactas")
	}

	autoridad := autoridadGobiernoPrueba(
		t, activa, AccionRetirarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	retirada, err := activa.Retirar(
		activa.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "retirada"),
		autoridad, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	otraAutoridad := autoridad.clonar()
	otraAutoridad.atestacion = referenciaPrueba(t, "atestacion:autoridad:otra", 1, '7')
	if !retirada.IncorporaAutoridadExacta(autoridad) ||
		retirada.IncorporaAutoridadExacta(otraAutoridad) ||
		activa.IncorporaAutoridadExacta(autoridad) {
		t.Fatal("la transicion terminal no distinguio su autoridad exacta")
	}
}

func TestInstanteUltimaActuacionSigueExactamenteElEstado(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	publicada := publicarGobiernoPrueba(
		t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute),
	)
	activa := activarGobiernoPrueba(
		t, publicada, instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	autoridad := autoridadGobiernoPrueba(
		t, activa, AccionRetirarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	retirada, err := activa.Retirar(
		activa.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "retirada"),
		autoridad, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		version  VersionGobernadaReglasBaremo
		esperado time.Time
	}{
		{borrador, instanteBaseGobiernoPrueba},
		{publicada, instanteBaseGobiernoPrueba.Add(3 * time.Minute)},
		{activa, instanteBaseGobiernoPrueba.Add(5 * time.Minute)},
		{retirada, instanteBaseGobiernoPrueba.Add(7 * time.Minute)},
	}
	for _, caso := range casos {
		obtenido, err := caso.version.InstanteUltimaActuacion()
		if err != nil || !obtenido.Equal(caso.esperado) ||
			obtenido.Location() != time.UTC || obtenido.Nanosecond()%1_000 != 0 {
			t.Fatalf("instante=%s esperado=%s error=%v", obtenido, caso.esperado, err)
		}
	}
	if _, err := (VersionGobernadaReglasBaremo{}).InstanteUltimaActuacion(); err == nil {
		t.Fatal("version cero devolvio instante")
	}
}
