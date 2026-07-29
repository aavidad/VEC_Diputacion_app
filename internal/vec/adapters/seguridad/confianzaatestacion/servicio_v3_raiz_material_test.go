package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func pruebaRaizMaterialV3(
	t *testing.T,
	escenario escenarioConfianzaAtestacionV3Prueba,
) PruebaConfianzaAtestacionAutorizacionV3 {
	t.Helper()
	prueba, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	)
	if err != nil {
		t.Fatalf("verificar atestación V3: %v", err)
	}
	return prueba
}

func exigirMismaRaizMaterialV3(
	t *testing.T,
	obtenida RaizPublicaAtestacionAutorizacionV3,
	esperada RaizPublicaAtestacionAutorizacionV3,
) {
	t.Helper()
	if obtenida.validar() != nil ||
		obtenida.claveID != esperada.claveID ||
		obtenida.version != esperada.version ||
		!bytes.Equal(obtenida.clavePublica, esperada.clavePublica) ||
		obtenida.huellaClaveSPKISHA256 != esperada.huellaClaveSPKISHA256 ||
		obtenida.audienciaDespliegue != esperada.audienciaDespliegue ||
		obtenida.estado != esperada.estado ||
		!obtenida.validaDesde.Equal(esperada.validaDesde) ||
		!obtenida.validaHasta.Equal(esperada.validaHasta) ||
		!obtenida.revocadaEn.Equal(esperada.revocadaEn) {
		t.Fatalf("raíz nominal distinta: obtenida=%v esperada=%v", obtenida, esperada)
	}
}

func TestRaizPublicaParaPruebaV3RecuperaCopiaDefensivaExacta(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba := pruebaRaizMaterialV3(t, escenario)

	primera, err := escenario.servicio.raizPublicaParaPruebaV3(prueba)
	if err != nil {
		t.Fatalf("recuperar raíz nominal: %v", err)
	}
	exigirMismaRaizMaterialV3(t, primera, escenario.raiz)

	primera.clavePublica[0] ^= 0xff
	primera.claveID = "clave:atestacion:v3:alterada"
	segunda, err := escenario.servicio.raizPublicaParaPruebaV3(prueba)
	if err != nil {
		t.Fatalf("recuperar segunda copia: %v", err)
	}
	exigirMismaRaizMaterialV3(t, segunda, escenario.raiz)
	if bytes.Equal(primera.clavePublica, segunda.clavePublica) {
		t.Fatal("la clave pública devuelta comparte memoria con el servicio")
	}
}

func TestRaizPublicaParaPruebaV3RechazaCerosCrucesYMetadatos(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba := pruebaRaizMaterialV3(t, escenario)

	var servicioNulo *ServicioConfianzaAtestacionAutorizacionV3
	for nombre, caso := range map[string]struct {
		servicio *ServicioConfianzaAtestacionAutorizacionV3
		prueba   PruebaConfianzaAtestacionAutorizacionV3
	}{
		"servicio_nulo": {servicio: servicioNulo, prueba: prueba},
		"servicio_cero": {
			servicio: &ServicioConfianzaAtestacionAutorizacionV3{},
			prueba:   prueba,
		},
		"prueba_cero": {servicio: escenario.servicio},
		"mapa_sin_clave": {
			servicio: &ServicioConfianzaAtestacionAutorizacionV3{
				raices: map[string]raizVerificacionAtestacionV3{},
			},
			prueba: prueba,
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := caso.servicio.raizPublicaParaPruebaV3(caso.prueba); !errors.Is(
				err,
				ErrVerificacionConfianzaAtestacionV3Fallida,
			) {
				t.Fatalf("valor cero o cruce aceptado: %v", err)
			}
		})
	}

	mutaciones := map[string]func(*datosPruebaConfianzaAtestacionAutorizacionV3){
		"clave_id": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.ClaveID = "clave:atestacion:v3:ajena:2026-07"
		},
		"version_raiz": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.RaizVersion++
		},
		"huella_spki": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.HuellaClaveSPKISHA256 = strings.Repeat("9", 64)
		},
		"audiencia": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.AudienciaDespliegue = "vec-diputacion/pruebas/otra-audiencia"
		},
		"inicio_raiz": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.RaizValidaDesde = d.RaizValidaDesde.Add(-time.Minute)
		},
		"fin_raiz": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.RaizValidaHasta = d.RaizValidaHasta.Add(time.Minute)
		},
		"revision_configuracion": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.RevisionConfiguracion = "confianza:atestacion:v3:revision:ajena"
		},
		"secuencia_configuracion": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.SecuenciaConfiguracion++
		},
		"huella_configuracion": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.HuellaConfiguracionSHA256 = strings.Repeat("8", 64)
		},
		"publicacion_configuracion": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.ConfiguracionPublicadaEn = d.ConfiguracionPublicadaEn.Add(-time.Minute)
		},
		"fin_configuracion": func(d *datosPruebaConfianzaAtestacionAutorizacionV3) {
			d.ConfiguracionExpiraEn = d.ConfiguracionExpiraEn.Add(time.Minute)
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			datos := *prueba.datos
			mutar(&datos)
			ajena, err := nuevaPruebaConfianzaAtestacionAutorizacionV3(datos)
			if err != nil {
				t.Fatalf("la prueba cruzada debe ser válida por sí sola: %v", err)
			}
			if _, err := escenario.servicio.raizPublicaParaPruebaV3(ajena); !errors.Is(
				err,
				ErrVerificacionConfianzaAtestacionV3Fallida,
			) {
				t.Fatalf("metadato cruzado aceptado: %v", err)
			}
		})
	}
}

func TestRaizPublicaParaPruebaV3SeleccionaRotacionExacta(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	privadaRotada := ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat("m", ed25519.SeedSize)),
	)
	raizRotada, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		"clave:atestacion:v3:material:rotada",
		2,
		privadaRotada.Public().(ed25519.PublicKey),
		audienciaConfianzaAtestacionV3Prueba,
		EstadoClaveAtestacionAutorizacionV3Activa,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:material:rotacion",
		escenario.configuracion.secuencia+1,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(30*time.Minute),
		escenario.raiz,
		raizRotada,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}

	atestacionRotada := atestacionConfianzaAtestacionV3Prueba(
		t,
		domain.CabeceraAtestacionAutorizacionV3{
			FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
			Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
			ClaveID:        raizRotada.claveID,
			Audiencia:      audienciaConfianzaAtestacionV3Prueba,
		},
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		privadaRotada,
		escenario.ahora,
	)
	pruebaOriginal, err := servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	pruebaRotada, err := servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		atestacionRotada,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenidaOriginal, err := servicio.raizPublicaParaPruebaV3(pruebaOriginal)
	if err != nil {
		t.Fatal(err)
	}
	obtenidaRotada, err := servicio.raizPublicaParaPruebaV3(pruebaRotada)
	if err != nil {
		t.Fatal(err)
	}
	exigirMismaRaizMaterialV3(t, obtenidaOriginal, escenario.raiz)
	exigirMismaRaizMaterialV3(t, obtenidaRotada, raizRotada)
}

func TestRaizPublicaParaPruebaV3RechazaRaizRevocada(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba := pruebaRaizMaterialV3(t, escenario)
	revocada, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		escenario.raiz.claveID,
		escenario.raiz.version,
		escenario.raiz.clavePublica,
		escenario.raiz.audienciaDespliegue,
		EstadoClaveAtestacionAutorizacionV3Revocada,
		escenario.raiz.validaDesde,
		escenario.raiz.validaHasta,
		escenario.ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:material:revocada",
		escenario.configuracion.secuencia+1,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(30*time.Minute),
		revocada,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	datos := *prueba.datos
	datos.RevisionConfiguracion = configuracion.revision
	datos.SecuenciaConfiguracion = configuracion.secuencia
	datos.HuellaConfiguracionSHA256 = configuracion.huellaSHA256
	datos.ConfiguracionPublicadaEn = configuracion.publicadaEn
	datos.ConfiguracionExpiraEn = configuracion.expiraEn
	pruebaMismaConfiguracion, err :=
		nuevaPruebaConfianzaAtestacionAutorizacionV3(datos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.raizPublicaParaPruebaV3(
		pruebaMismaConfiguracion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV3Fallida) {
		t.Fatalf("raíz revocada recuperada: %v", err)
	}
}
