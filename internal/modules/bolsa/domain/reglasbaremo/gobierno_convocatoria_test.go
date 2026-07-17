package reglasbaremo

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestConvocatoriaActivacionSigueCicloCompletoEHistorico(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	comprobarConvocatoriaActivacionAusente(t, borrador)

	publicada := publicarGobiernoPrueba(
		t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute),
	)
	comprobarConvocatoriaActivacionAusente(t, publicada)

	dependencias := dependenciasGobiernoPrueba(
		t, publicada, instanteBaseGobiernoPrueba.Add(4*time.Minute),
	)
	esperada := dependencias.Convocatoria()
	activa, err := publicada.Activar(
		publicada.Revision(), actorGobiernoPrueba,
		motivoGobiernoPrueba(t, "activacion"), dependencias,
		instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	comprobarConvocatoriaActivacionExacta(t, activa, esperada)

	sucesora := referenciaPrueba(t, "reglas:convocatoria-2026:2", 2, 'a')
	autoridadSustitucion := autoridadGobiernoPrueba(
		t, activa, AccionSustituirReglasBaremo, actorGobiernoPrueba, &sucesora,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	sustituida, err := activa.Sustituir(
		activa.Revision(), actorGobiernoPrueba,
		motivoGobiernoPrueba(t, "sustitucion"), sucesora,
		autoridadSustitucion, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	comprobarConvocatoriaActivacionExacta(t, sustituida, esperada)

	autoridadRetirada := autoridadGobiernoPrueba(
		t, activa, AccionRetirarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	retirada, err := activa.Retirar(
		activa.Revision(), actorGobiernoPrueba,
		motivoGobiernoPrueba(t, "retirada"), autoridadRetirada,
		instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	comprobarConvocatoriaActivacionExacta(t, retirada, esperada)

	autoridadDescarte := autoridadGobiernoPrueba(
		t, borrador, AccionDescartarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(time.Minute),
	)
	descartada, err := borrador.Descartar(
		borrador.Revision(), actorGobiernoPrueba,
		motivoGobiernoPrueba(t, "descarte"), autoridadDescarte,
		instanteBaseGobiernoPrueba.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	comprobarConvocatoriaActivacionAusente(t, descartada)
}

func TestConvocatoriaActivacionSobreClonConRestauracionCanonica(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	publicada := publicarGobiernoPrueba(
		t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute),
	)
	dependencias := dependenciasGobiernoPrueba(
		t, publicada, instanteBaseGobiernoPrueba.Add(4*time.Minute),
	)
	activa, err := publicada.Activar(
		publicada.Revision(), actorGobiernoPrueba,
		motivoGobiernoPrueba(t, "activacion"), dependencias,
		instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	antes, err := activa.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}

	// Clonar restaura el conjunto desde su representacion canonica y copia los
	// actos gobernados sin compartir sus colecciones internas.
	clon, err := activa.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	despues, err := clon.RepresentacionCanonica()
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatalf("el clon canonico cambio la version: %v", err)
	}
	comprobarConvocatoriaActivacionExacta(t, clon, dependencias.Convocatoria())

	devuelta, presente, err := clon.ConvocatoriaActivacion()
	if err != nil || !presente || !referenciasVersionadasIguales(devuelta, dependencias.Convocatoria()) {
		t.Fatalf("convocatoria no disponible en el clon: %v", err)
	}
	devuelta = ReferenciaVersionada{}
	comprobarConvocatoriaActivacionExacta(t, clon, dependencias.Convocatoria())
}

func TestConvocatoriaActivacionFallaCerradoAnteVersionInvalida(t *testing.T) {
	_, presente, err := (VersionGobernadaReglasBaremo{}).ConvocatoriaActivacion()
	if presente || !errors.Is(err, ErrGobiernoInvarianteQuebrada) {
		t.Fatalf("version invalida aceptada: presente=%v error=%v", presente, err)
	}
}

func comprobarConvocatoriaActivacionAusente(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
) {
	t.Helper()
	convocatoria, presente, err := version.ConvocatoriaActivacion()
	if err != nil || presente || convocatoria != (ReferenciaVersionada{}) {
		t.Fatalf(
			"se invento convocatoria sin activacion: presente=%v convocatoria=%v error=%v",
			presente, convocatoria, err,
		)
	}
}

func comprobarConvocatoriaActivacionExacta(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
	esperada ReferenciaVersionada,
) {
	t.Helper()
	obtenida, presente, err := version.ConvocatoriaActivacion()
	if err != nil || !presente || !referenciasVersionadasIguales(obtenida, esperada) {
		t.Fatalf(
			"convocatoria de activacion inexacta: presente=%v obtenida=%v error=%v",
			presente, obtenida, err,
		)
	}
}
