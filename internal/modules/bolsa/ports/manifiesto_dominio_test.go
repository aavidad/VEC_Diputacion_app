package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"
)

func manifiestoSinSelloPrueba(t *testing.T) ManifiestoProbatorioBaremacion {
	t.Helper()
	manifiesto := manifiestoProbatorioValidoPrueba(t, contenidoDecisionValidoPrueba(t)).Clonar()
	manifiesto.HuellaManifiestoSHA256 = ""
	manifiesto.SelloManifiestoHMACSHA256 = ""
	return manifiesto
}

func TestManifiestoProbatorioV3RechazaDominioAusenteOAjeno(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"sin esquema", func(m *ManifiestoProbatorioBaremacion) { m.Esquema = "" }},
		{"esquema ajeno", func(m *ManifiestoProbatorioBaremacion) { m.Esquema = "vec.otro.manifiesto" }},
		{"sin finalidad", func(m *ManifiestoProbatorioBaremacion) { m.Finalidad = "" }},
		{"finalidad ajena", func(m *ManifiestoProbatorioBaremacion) { m.Finalidad = "otra_finalidad" }},
		{"sin version", func(m *ManifiestoProbatorioBaremacion) { m.VersionEsquema = 0 }},
		{"version ajena", func(m *ManifiestoProbatorioBaremacion) { m.VersionEsquema = 1 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manifiesto := manifiestoSinSelloPrueba(t)
			caso.mutar(&manifiesto)
			if _, _, err := manifiesto.PrepararSellado(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
				t.Fatalf("el dominio ausente o ajeno fue admitido: %v", err)
			}
		})
	}
}

func TestManifiestoProbatorioV3RechazaReferenciasNoCanonicas(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"comodin", func(m *ManifiestoProbatorioBaremacion) { m.Referencia = "manifiesto:*" }},
		{"unicode", func(m *ManifiestoProbatorioBaremacion) { m.Referencia = "manifiesto:á" }},
		{"comodin en autorizacion", func(m *ManifiestoProbatorioBaremacion) {
			m.Autorizaciones[0].AutorizacionRef = "autorizacion:*"
		}},
		{"unicode en evidencia", func(m *ManifiestoProbatorioBaremacion) {
			m.Evidencias[0].Referencia = "evidencia:á"
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manifiesto := manifiestoSinSelloPrueba(t)
			caso.mutar(&manifiesto)
			if _, _, err := manifiesto.PrepararSellado(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
				t.Fatalf("la referencia no canonica fue admitida: %v", err)
			}
		})
	}
}

func TestManifiestoProbatorioV3CompartePerfilTemporalConPostgreSQL(t *testing.T) {
	rechazados := []struct {
		nombre   string
		instante time.Time
	}{
		{"valor cero", time.Time{}},
		{"anio negativo", time.Date(-1, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"anio cero", time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"anio de cinco cifras", time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"zona no UTC", time.Date(2026, time.July, 16, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))},
	}
	for _, caso := range rechazados {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manifiesto := manifiestoSinSelloPrueba(t)
			manifiesto.CreadoEn = caso.instante
			if _, _, err := manifiesto.PrepararSellado(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
				t.Fatalf("el instante no canonico fue admitido: %v", err)
			}
		})
	}

	for _, instante := range []time.Time{
		time.Date(1, time.January, 1, 0, 0, 0, 1, time.UTC),
		time.Date(9999, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	} {
		manifiesto := manifiestoSinSelloPrueba(t)
		manifiesto.CreadoEn = instante
		if _, _, err := manifiesto.PrepararSellado(); err != nil {
			t.Fatalf("el instante limite canonico fue rechazado (%s): %v", instante.Format(time.RFC3339Nano), err)
		}
	}
}

func TestManifiestoProbatorioV3AutenticaDominioComoPrimerosCampos(t *testing.T) {
	manifiesto := manifiestoSinSelloPrueba(t)
	_, carga, err := manifiesto.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto v3: %v", err)
	}
	material := carga.Revelar()
	leer := func() string {
		t.Helper()
		if len(material) < 8 {
			t.Fatal("material canónico truncado antes de la longitud")
		}
		longitud := binary.BigEndian.Uint64(material[:8])
		material = material[8:]
		if longitud > uint64(len(material)) {
			t.Fatal("material canónico truncado dentro de un campo")
		}
		valor := string(material[:longitud])
		material = material[longitud:]
		return valor
	}
	if finalidad := leer(); finalidad != string(FinalidadSelloManifiestoProbatorioBaremacionV3) {
		t.Fatalf("finalidad criptografica ausente: %q", finalidad)
	}
	interior := []byte(leer())
	if len(material) != 0 {
		t.Fatal("la envoltura criptografica contiene campos no definidos")
	}
	material = interior
	esperados := []string{
		EsquemaManifiestoProbatorioBaremacion,
		FinalidadManifiestoProbatorioBaremacion,
		strconv.Itoa(VersionManifiestoProbatorioBaremacion),
	}
	for indice, esperado := range esperados {
		if obtenido := leer(); obtenido != esperado {
			t.Fatalf("campo de dominio %d fuera de orden: obtenido=%q esperado=%q", indice, obtenido, esperado)
		}
	}
}

func TestRepresentacionCanonicaManifiestoProbatorioV3EsPublicaReproducibleYEstable(t *testing.T) {
	base := manifiestoSinSelloPrueba(t)
	preparado, producida, err := base.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto: %v", err)
	}
	reconstruidaPreparada, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(preparado)
	if err != nil {
		t.Fatalf("reconstruir manifiesto preparado: %v", err)
	}
	sellado, err := preparado.IncorporarSello("hmac-sha256:vector_1:" + huellaPruebaPuertos("9"))
	if err != nil {
		t.Fatalf("incorporar sello de vector: %v", err)
	}
	reconstruidaSellada, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(sellado)
	if err != nil {
		t.Fatalf("reconstruir manifiesto sellado: %v", err)
	}
	if !bytes.Equal(producida.Revelar(), reconstruidaPreparada.Revelar()) ||
		!bytes.Equal(producida.Revelar(), reconstruidaSellada.Revelar()) {
		t.Fatal("productor y verificador no reconstruyen los mismos bytes")
	}
	suma := sha256.Sum256(producida.Revelar())
	const vectorSHA256 = "c8fb92a5f3b3dae996fed88b507fb1cd454d877ce08698e84edba1d50e3608af"
	const longitudVector = 6393
	if obtenida := hex.EncodeToString(suma[:]); obtenida != vectorSHA256 || len(producida.Revelar()) != longitudVector {
		t.Fatalf("vector canonico alterado: obtenido=%s esperado=%s longitud=%d", obtenida, vectorSHA256, len(producida.Revelar()))
	}
}

func TestRepresentacionCanonicaManifiestoProbatorioV3RechazaMutacionesYNoAdmiteIntercambio(t *testing.T) {
	base := manifiestoSinSelloPrueba(t)
	preparado, representacionBase, err := base.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	cambios := []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"esquema", func(m *ManifiestoProbatorioBaremacion) { m.Esquema = "vec.otro.manifiesto" }},
		{"finalidad", func(m *ManifiestoProbatorioBaremacion) { m.Finalidad = "otra_finalidad" }},
		{"version", func(m *ManifiestoProbatorioBaremacion) { m.VersionEsquema++ }},
		{"referencia", func(m *ManifiestoProbatorioBaremacion) { m.Referencia += "-alterada" }},
		{"autorizacion", func(m *ManifiestoProbatorioBaremacion) { m.Autorizaciones[0].AutorizacionRef += "-alterada" }},
		{"evidencia", func(m *ManifiestoProbatorioBaremacion) {
			m.Evidencias[0].HuellaEvidenciaSHA256 = huellaPruebaPuertos("8")
		}},
		{"huella", func(m *ManifiestoProbatorioBaremacion) { m.HuellaManifiestoSHA256 = huellaPruebaPuertos("7") }},
	}
	for _, cambio := range cambios {
		t.Run(cambio.nombre, func(t *testing.T) {
			mutado := preparado.Clonar()
			cambio.mutar(&mutado)
			if _, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(mutado); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
				t.Fatalf("mutacion admitida: %v", err)
			}
		})
	}

	alternativo := base.Clonar()
	alternativo.Referencia += "-alternativo"
	alternativo.CreadoEn = alternativo.CreadoEn.Add(time.Second)
	_, representacionAlternativa, err := alternativo.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto alternativo: %v", err)
	}
	if bytes.Equal(representacionBase.Revelar(), representacionAlternativa.Revelar()) {
		t.Fatal("dos manifiestos distintos comparten representacion canonica")
	}
}

func TestRepresentacionCanonicaManifiestoProbatorioV3CodificaConteosYParticionInequivoca(t *testing.T) {
	base := manifiestoSinSelloPrueba(t)
	preparado, representacion, err := base.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	envoltura := representacion.Revelar()
	if finalidad := leerCampoCanonicoManifiestoPrueba(t, &envoltura); finalidad !=
		string(FinalidadSelloManifiestoProbatorioBaremacionV3) {
		t.Fatalf("finalidad criptografica inesperada: %q", finalidad)
	}
	material := []byte(leerCampoCanonicoManifiestoPrueba(t, &envoltura))
	if len(envoltura) != 0 {
		t.Fatal("campos fuera de la envoltura definida")
	}

	// Doce campos forman la cabecera fija. Cada autorizacion ocupa cinco y
	// cada evidencia cuatro; ambos limites se declaran antes de su seccion.
	for campo := 0; campo < 12; campo++ {
		_ = leerCampoCanonicoManifiestoPrueba(t, &material)
	}
	if cantidad := leerCampoCanonicoManifiestoPrueba(t, &material); cantidad !=
		strconv.Itoa(len(preparado.Autorizaciones)) {
		t.Fatalf("cantidad de autorizaciones ausente: %q", cantidad)
	}
	for range preparado.Autorizaciones {
		for campo := 0; campo < 5; campo++ {
			_ = leerCampoCanonicoManifiestoPrueba(t, &material)
		}
	}
	if cantidad := leerCampoCanonicoManifiestoPrueba(t, &material); cantidad !=
		strconv.Itoa(len(preparado.Evidencias)) {
		t.Fatalf("cantidad de evidencias ausente: %q", cantidad)
	}
	for range preparado.Evidencias {
		for campo := 0; campo < 4; campo++ {
			_ = leerCampoCanonicoManifiestoPrueba(t, &material)
		}
	}
	if huella := leerCampoCanonicoManifiestoPrueba(t, &material); huella != preparado.HuellaManifiestoSHA256 {
		t.Fatalf("particion desplazo la huella final: %q", huella)
	}
	if len(material) != 0 {
		t.Fatal("material restante tras consumir las secciones declaradas")
	}

	for _, caso := range []struct {
		nombre string
		mutar  func(*ManifiestoProbatorioBaremacion)
	}{
		{"menos autorizaciones", func(m *ManifiestoProbatorioBaremacion) {
			m.Autorizaciones = m.Autorizaciones[:len(m.Autorizaciones)-1]
		}},
		{"menos evidencias", func(m *ManifiestoProbatorioBaremacion) {
			m.Evidencias = m.Evidencias[:len(m.Evidencias)-1]
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := base.Clonar()
			caso.mutar(&mutado)
			if _, _, err := mutado.PrepararSellado(); err == nil {
				t.Fatal("se admitio cambiar la particion declarada")
			}
		})
	}
}

func leerCampoCanonicoManifiestoPrueba(t *testing.T, material *[]byte) string {
	t.Helper()
	if len(*material) < 8 {
		t.Fatal("material canonico truncado antes de una longitud")
	}
	longitud := binary.BigEndian.Uint64((*material)[:8])
	*material = (*material)[8:]
	if longitud > uint64(len(*material)) {
		t.Fatal("material canonico truncado dentro de un campo")
	}
	valor := string((*material)[:longitud])
	*material = (*material)[longitud:]
	return valor
}
