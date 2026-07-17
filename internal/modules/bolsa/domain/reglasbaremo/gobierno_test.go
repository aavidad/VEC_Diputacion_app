package reglasbaremo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

const actorGobiernoPrueba = "per_0123456789abcdef0123456789abcdef"

var instanteBaseGobiernoPrueba = time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC)

func TestGobiernoReglasBaremoRecorreCicloYFijaRevisiones(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	huellaBorrador := huellaGobiernoPrueba(t, borrador)
	referenciaContenido := referenciaContenidoGobiernoPrueba(t, borrador)

	publicada := publicarGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute))
	activa := activarGobiernoPrueba(t, publicada, instanteBaseGobiernoPrueba.Add(5*time.Minute))
	sucesora := referenciaPrueba(t, "reglas:convocatoria-2026:2", 2, 'a')
	autoridad := autoridadGobiernoPrueba(
		t, activa, AccionSustituirReglasBaremo, actorGobiernoPrueba, &sucesora,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	sustituida, err := activa.Sustituir(
		3, actorGobiernoPrueba, motivoGobiernoPrueba(t, "sustitucion"), sucesora,
		autoridad, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre   string
		version  VersionGobernadaReglasBaremo
		estado   EstadoGobiernoReglasBaremo
		revision uint64
	}{
		{"borrador", borrador, EstadoReglasBaremoBorrador, 1},
		{"publicada", publicada, EstadoReglasBaremoPublicada, 2},
		{"activa", activa, EstadoReglasBaremoActiva, 3},
		{"sustituida", sustituida, EstadoReglasBaremoSustituida, 4},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if caso.version.Estado() != caso.estado || caso.version.Revision() != caso.revision {
				t.Fatalf("estado/revision: %s/%d", caso.version.Estado(), caso.version.Revision())
			}
			if err := caso.version.Validar(); err != nil {
				t.Fatal(err)
			}
			if obtenida := referenciaContenidoGobiernoPrueba(t, caso.version); !referenciasVersionadasIguales(obtenida, referenciaContenido) {
				t.Fatal("la transicion cambio la referencia exacta del contenido")
			}
		})
	}
	if huellaGobiernoPrueba(t, publicada) == huellaBorrador ||
		huellaGobiernoPrueba(t, activa) == huellaGobiernoPrueba(t, publicada) ||
		huellaGobiernoPrueba(t, sustituida) == huellaGobiernoPrueba(t, activa) {
		t.Fatal("una transicion no cambio la huella del estado gobernado")
	}
}

func TestGobiernoReglasBaremoPermiteRetirarYDescartarPorRutasSeparadas(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	descarteEn := instanteBaseGobiernoPrueba.Add(2 * time.Minute)
	autoridadDescarte := autoridadGobiernoPrueba(
		t, borrador, AccionDescartarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(time.Minute),
	)
	descartada, err := borrador.Descartar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "descarte"),
		autoridadDescarte, descarteEn,
	)
	if err != nil || descartada.Estado() != EstadoReglasBaremoDescartada || descartada.Revision() != 2 {
		t.Fatalf("descarte: estado=%s revision=%d error=%v", descartada.Estado(), descartada.Revision(), err)
	}

	publicada := publicarGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute))
	activa := activarGobiernoPrueba(t, publicada, instanteBaseGobiernoPrueba.Add(5*time.Minute))
	autoridadRetirada := autoridadGobiernoPrueba(
		t, activa, AccionRetirarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(6*time.Minute),
	)
	retirada, err := activa.Retirar(
		3, actorGobiernoPrueba, motivoGobiernoPrueba(t, "retirada"),
		autoridadRetirada, instanteBaseGobiernoPrueba.Add(7*time.Minute),
	)
	if err != nil || retirada.Estado() != EstadoReglasBaremoRetirada || retirada.Revision() != 4 {
		t.Fatalf("retirada: estado=%s revision=%d error=%v", retirada.Estado(), retirada.Revision(), err)
	}
}

func TestGobiernoReglasBaremoRechazaOCCTransicionesIlegalesYReplay(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	aprobacion := aprobacionGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(time.Minute))
	publicarEn := instanteBaseGobiernoPrueba.Add(3 * time.Minute)

	_, err := borrador.Publicar(
		2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion, publicarEn,
	)
	if !errors.Is(err, ErrGobiernoRevisionConflicto) {
		t.Fatalf("OCC no rechazo revision obsoleta: %v", err)
	}

	publicada, err := borrador.Publicar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion, publicarEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = publicada.Publicar(
		2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion,
		publicarEn.Add(time.Minute),
	)
	if !errors.Is(err, ErrGobiernoTransicionProhibida) {
		t.Fatalf("doble publicacion aceptada: %v", err)
	}
	_, err = publicada.Descartar(
		2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "descarte"),
		AtestacionAutoridadReglasBaremo{}, publicarEn.Add(time.Minute),
	)
	if !errors.Is(err, ErrGobiernoTransicionProhibida) {
		t.Fatalf("descarte desde publicada aceptado: %v", err)
	}
	_, err = borrador.Retirar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "retirada"),
		AtestacionAutoridadReglasBaremo{}, publicarEn,
	)
	if !errors.Is(err, ErrGobiernoTransicionProhibida) {
		t.Fatalf("retirada desde borrador aceptada: %v", err)
	}

	otroBorrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba.Add(time.Minute))
	_, err = otroBorrador.Publicar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion,
		publicarEn.Add(time.Minute),
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("replay contra otro estado aceptado: %v", err)
	}
}

func TestGobiernoReglasBaremoExigeOrdenTemporalUTCYVigencia(t *testing.T) {
	conjunto := conjuntoPrueba(t, true)
	motivo := motivoGobiernoPrueba(t, "creacion")
	_, err := NuevaVersionGobernadaReglasBaremo(
		conjunto, actorGobiernoPrueba, motivo, instanteBaseGobiernoPrueba.In(time.FixedZone("local", 3600)),
	)
	if !errors.Is(err, ErrGobiernoValorInvalido) {
		t.Fatalf("instante no UTC aceptado: %v", err)
	}
	_, err = NuevaVersionGobernadaReglasBaremo(
		conjunto, actorGobiernoPrueba, motivo, instanteBaseGobiernoPrueba.Add(time.Nanosecond),
	)
	if !errors.Is(err, ErrGobiernoValorInvalido) {
		t.Fatalf("precision submicrosegundo aceptada: %v", err)
	}

	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	aprobacion := aprobacionGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(time.Minute))
	_, err = borrador.Publicar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion,
		instanteBaseGobiernoPrueba.Add(30*time.Second),
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("publicacion anterior a la verificacion aceptada: %v", err)
	}
	_, err = borrador.Publicar(
		1, actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion,
		instanteBaseGobiernoPrueba.Add(11*time.Minute),
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("aprobacion caducada aceptada: %v", err)
	}
}

func TestGobiernoReglasBaremoLigaActivacionADependenciasExactas(t *testing.T) {
	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	publicada := publicarGobiernoPrueba(t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute))
	dependencias := dependenciasGobiernoPrueba(t, publicada, instanteBaseGobiernoPrueba.Add(4*time.Minute))

	datos := datosDependenciasDesdeAtestacion(dependencias)
	datos.Convocatoria = referenciaPrueba(t, "convocatoria:distinta", 1, 'b')
	convocatoriaIncorrecta, err := NuevaAtestacionDependenciasVigentesReglasBaremo(datos)
	if err != nil {
		t.Fatal(err)
	}
	_, err = publicada.Activar(
		2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "activacion"),
		convocatoriaIncorrecta, instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("convocatoria inexacta aceptada: %v", err)
	}

	datos = datosDependenciasDesdeAtestacion(dependencias)
	basesOriginales := datos.Bases
	datos.Bases = referenciaPrueba(t, datos.Bases.Referencia(), datos.Bases.Version()+1, 'c')
	for indice := range datos.Dependencias {
		if referenciasVersionadasIguales(datos.Dependencias[indice], basesOriginales) {
			datos.Dependencias[indice] = datos.Bases
			break
		}
	}
	basesIncorrectas, err := NuevaAtestacionDependenciasVigentesReglasBaremo(datos)
	if err != nil {
		t.Fatal(err)
	}
	_, err = publicada.Activar(
		2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "activacion"),
		basesIncorrectas, instanteBaseGobiernoPrueba.Add(5*time.Minute),
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("bases inexactas aceptadas: %v", err)
	}

	datos = datosDependenciasDesdeAtestacion(dependencias)
	datos.Dependencias = datos.Dependencias[1:]
	sinDependencia, err := NuevaAtestacionDependenciasVigentesReglasBaremo(datos)
	if err == nil {
		_, err = publicada.Activar(
			2, actorGobiernoPrueba, motivoGobiernoPrueba(t, "activacion"),
			sinDependencia, instanteBaseGobiernoPrueba.Add(5*time.Minute),
		)
	}
	if err == nil {
		t.Fatal("conjunto incompleto de dependencias aceptado")
	}
}

func TestGobiernoReglasBaremoHaceCopiasYCanonizaSinMutacion(t *testing.T) {
	conjunto := conjuntoPrueba(t, true)
	borrador, err := NuevaVersionGobernadaReglasBaremo(
		conjunto, actorGobiernoPrueba, motivoGobiernoPrueba(t, "creacion"), instanteBaseGobiernoPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	antes, err := borrador.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	conjunto.secciones[0] = SeccionBaremo{}
	devuelto, err := borrador.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	devuelto.reglasExperiencia[0].criterios[0].valores[0] = "alterado"
	devuelto.secciones[0] = SeccionBaremo{}
	despues, err := borrador.RepresentacionCanonica()
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatalf("una copia externa altero el estado: %v", err)
	}

	firmantes := []string{
		"per_22222222222222222222222222222222",
		"per_11111111111111111111111111111111",
	}
	aprobacion := aprobacionGobiernoPruebaConFirmantes(
		t, borrador, instanteBaseGobiernoPrueba.Add(time.Minute), firmantes,
	)
	firmantes[0] = "dni:12345678Z"
	obtenidos := aprobacion.Firmantes()
	obtenidos[0] = "nombre:alterado"
	if aprobacion.Firmantes()[0] != "per_11111111111111111111111111111111" {
		t.Fatal("los firmantes no quedaron ordenados y aislados")
	}

	clon, err := borrador.Clonar()
	if err != nil || huellaGobiernoPrueba(t, clon) != huellaGobiernoPrueba(t, borrador) {
		t.Fatalf("clon no equivalente: %v", err)
	}
}

func TestGobiernoReglasBaremoIncluyeGruposYDeduplicaCatalogosExactos(t *testing.T) {
	conjunto, catalogoCompartido, grupo := conjuntoCatalogosCompartidosGobiernoPrueba(t, false)
	version, err := NuevaVersionGobernadaReglasBaremo(
		conjunto, actorGobiernoPrueba, motivoGobiernoPrueba(t, "creacion"), instanteBaseGobiernoPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	dependencias, err := version.DependenciasContenido()
	if err != nil {
		t.Fatal(err)
	}
	aparicionesCatalogo := 0
	incluyeGrupo := false
	for _, dependencia := range dependencias {
		if referenciasVersionadasIguales(dependencia, catalogoCompartido) {
			aparicionesCatalogo++
		}
		if referenciasVersionadasIguales(dependencia, grupo) {
			incluyeGrupo = true
		}
	}
	if aparicionesCatalogo != 1 {
		t.Fatalf("catalogo exacto compartido no deduplicado: %d", aparicionesCatalogo)
	}
	if !incluyeGrupo {
		t.Fatal("falta la definicion exacta del grupo de concurrencia")
	}
}

func TestGobiernoReglasBaremoRechazaDependenciaHomónimaDivergente(t *testing.T) {
	conjunto, _, _ := conjuntoCatalogosCompartidosGobiernoPrueba(t, true)
	version, err := NuevaVersionGobernadaReglasBaremo(
		conjunto, actorGobiernoPrueba, motivoGobiernoPrueba(t, "creacion"), instanteBaseGobiernoPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = version.DependenciasContenido(); !errors.Is(err, ErrGobiernoInvarianteQuebrada) {
		t.Fatalf("referencia con version/huella divergente aceptada: %v", err)
	}
}

func TestGobiernoReglasBaremoRechazaEvidenciasSobredimensionadasAntesDeCopiar(t *testing.T) {
	demasiadosFirmantes := make([]string, maximoFirmantesAprobacionReglasBaremo+1)
	_, err := NuevaAtestacionAprobacionFirmadaReglasBaremo(
		DatosAtestacionAprobacionFirmadaReglasBaremo{Firmantes: demasiadosFirmantes},
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("lista de firmantes sobredimensionada aceptada: %v", err)
	}

	demasiadasDependencias := make([]ReferenciaVersionada, maximoDependenciasReglasBaremo+1)
	_, err = NuevaAtestacionDependenciasVigentesReglasBaremo(
		DatosAtestacionDependenciasVigentesReglasBaremo{Dependencias: demasiadasDependencias},
	)
	if !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
		t.Fatalf("lista de dependencias sobredimensionada aceptada: %v", err)
	}
}

func conjuntoCatalogosCompartidosGobiernoPrueba(
	t *testing.T,
	divergente bool,
) (ConjuntoReglasBaremo, ReferenciaVersionada, ReferenciaVersionada) {
	t.Helper()
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	catalogoCompartido := reglas[0].criterios[0].catalogo
	catalogoSegundo := catalogoCompartido
	if divergente {
		catalogoSegundo = referenciaPrueba(
			t, catalogoCompartido.Referencia(), catalogoCompartido.Version()+1, '5',
		)
	}
	reglas[1].criterios[0].catalogo = catalogoSegundo
	conjunto, err := NuevoConjuntoReglasBaremo(identidad, bases, fecha, secciones, grupos, reglas)
	if err != nil {
		t.Fatal(err)
	}
	return conjunto, catalogoCompartido, grupos[0].Definicion()
}

func nuevaVersionGobiernoPrueba(t *testing.T, instante time.Time) VersionGobernadaReglasBaremo {
	t.Helper()
	version, err := NuevaVersionGobernadaReglasBaremo(
		conjuntoPrueba(t, true), actorGobiernoPrueba, motivoGobiernoPrueba(t, "creacion"), instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func motivoGobiernoPrueba(t *testing.T, clave string) MotivoCatalogadoReglasBaremo {
	t.Helper()
	suma := sha256.Sum256([]byte(clave))
	claveOpaca := "motivo_" + hex.EncodeToString(suma[:16])
	motivo, err := NuevoMotivoCatalogadoReglasBaremo(
		referenciaPrueba(t, "motivos_autorizacion", 4, 'd'), claveOpaca,
	)
	if err != nil {
		t.Fatal(err)
	}
	return motivo
}

func aprobacionGobiernoPrueba(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
	firmadaEn time.Time,
) AtestacionAprobacionFirmadaReglasBaremo {
	t.Helper()
	return aprobacionGobiernoPruebaConFirmantes(
		t, version, firmadaEn, []string{
			"per_11111111111111111111111111111111",
			"per_22222222222222222222222222222222",
		},
	)
}

func aprobacionGobiernoPruebaConFirmantes(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
	firmadaEn time.Time,
	firmantes []string,
) AtestacionAprobacionFirmadaReglasBaremo {
	t.Helper()
	vinculo, err := version.VinculoEstado()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion, err := NuevaAtestacionAprobacionFirmadaReglasBaremo(
		DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion: referenciaPrueba(t, "atestacion:aprobacion", 1, 'e'), Vinculo: vinculo,
			Firma:         referenciaPrueba(t, "documento:firma-aprobacion", 1, 'f'),
			PoliticaFirma: referenciaPrueba(t, "politica:firma-aprobacion", 2, '0'),
			Firmantes:     firmantes, FirmadaEn: firmadaEn,
			VerificadaEn: firmadaEn.Add(time.Minute), ValidaHasta: firmadaEn.Add(9 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return aprobacion
}

func publicarGobiernoPrueba(
	t *testing.T,
	borrador VersionGobernadaReglasBaremo,
	instante time.Time,
) VersionGobernadaReglasBaremo {
	t.Helper()
	aprobacion := aprobacionGobiernoPrueba(t, borrador, instante.Add(-2*time.Minute))
	publicada, err := borrador.Publicar(
		borrador.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "publicacion"), aprobacion, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return publicada
}

func dependenciasGobiernoPrueba(
	t *testing.T,
	publicada VersionGobernadaReglasBaremo,
	verificadaEn time.Time,
) AtestacionDependenciasVigentesReglasBaremo {
	t.Helper()
	vinculo, err := publicada.VinculoEstado()
	if err != nil {
		t.Fatal(err)
	}
	dependencias, err := publicada.DependenciasContenido()
	if err != nil {
		t.Fatal(err)
	}
	convocatoria := referenciaPrueba(
		t, publicada.conjunto.Identidad().ConvocatoriaRef(), 7, '1',
	)
	atestacion, err := NuevaAtestacionDependenciasVigentesReglasBaremo(
		DatosAtestacionDependenciasVigentesReglasBaremo{
			Atestacion: referenciaPrueba(t, "atestacion:dependencias", 1, '2'), Vinculo: vinculo,
			Convocatoria: convocatoria, Bases: publicada.conjunto.Bases(), Dependencias: dependencias,
			VerificadorRef: "svc_0123456789abcdef0123456789abcdef", VerificadaEn: verificadaEn,
			ValidaHasta: verificadaEn.Add(8 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return atestacion
}

func datosDependenciasDesdeAtestacion(
	a AtestacionDependenciasVigentesReglasBaremo,
) DatosAtestacionDependenciasVigentesReglasBaremo {
	return DatosAtestacionDependenciasVigentesReglasBaremo{
		Atestacion: a.Atestacion(), Vinculo: a.Vinculo(), Convocatoria: a.Convocatoria(),
		Bases: a.Bases(), Dependencias: a.Dependencias(), VerificadorRef: a.VerificadorRef(),
		VerificadaEn: a.VerificadaEn(), ValidaHasta: a.ValidaHasta(),
	}
}

func activarGobiernoPrueba(
	t *testing.T,
	publicada VersionGobernadaReglasBaremo,
	instante time.Time,
) VersionGobernadaReglasBaremo {
	t.Helper()
	dependencias := dependenciasGobiernoPrueba(t, publicada, instante.Add(-time.Minute))
	activa, err := publicada.Activar(
		publicada.Revision(), actorGobiernoPrueba, motivoGobiernoPrueba(t, "activacion"),
		dependencias, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return activa
}

func autoridadGobiernoPrueba(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
	accion AccionGobiernoReglasBaremo,
	actor string,
	relacionada *ReferenciaVersionada,
	emitidaEn time.Time,
) AtestacionAutoridadReglasBaremo {
	t.Helper()
	vinculo, err := version.VinculoEstado()
	if err != nil {
		t.Fatal(err)
	}
	autoridad, err := NuevaAtestacionAutoridadReglasBaremo(DatosAtestacionAutoridadReglasBaremo{
		Atestacion: referenciaPrueba(t, "atestacion:autoridad:"+string(accion), 1, '3'),
		Vinculo:    vinculo, Accion: accion, PrincipalRef: actor, Relacionada: relacionada,
		EmitidaEn: emitidaEn, ValidaHasta: emitidaEn.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return autoridad
}

func referenciaContenidoGobiernoPrueba(
	t *testing.T,
	version VersionGobernadaReglasBaremo,
) ReferenciaVersionada {
	t.Helper()
	referencia, err := version.ReferenciaContenido()
	if err != nil {
		t.Fatal(err)
	}
	return referencia
}

func huellaGobiernoPrueba(t *testing.T, version VersionGobernadaReglasBaremo) string {
	t.Helper()
	huella, err := version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return huella
}
