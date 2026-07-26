package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type escenarioMaterialConsumoV3Prueba struct {
	base      escenarioConfianzaAtestacionV3Prueba
	prueba    PruebaConfianzaAtestacionAutorizacionV3
	capacidad CapacidadBreveAtestacionAutorizacionV3
	material  MaterialConsumoAutorizacionAtestadaV3
}

func nuevoEscenarioMaterialConsumoV3Prueba(
	t *testing.T,
) escenarioMaterialConsumoV3Prueba {
	t.Helper()
	base, prueba := escenarioYPruebaConfianzaV3(t)
	clave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x61}, 32),
	)
	emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		&relojConfianzaAtestacionV3Prueba{
			ahora: base.ahora.Add(time.Microsecond),
		},
		bytes.NewReader(bytes.Repeat([]byte{0xa1}, 32)),
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(
		context.Background(),
		base.solicitud,
		base.decision,
		base.motivo,
		base.resultado,
		base.atestacion,
		prueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	material, err := NuevoMaterialConsumoAutorizacionAtestadaV3(
		base.solicitud,
		base.decision,
		base.motivo,
		base.resultado,
		base.atestacion,
		prueba,
		capacidad,
		base.raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioMaterialConsumoV3Prueba{
		base:      base,
		prueba:    prueba,
		capacidad: capacidad,
		material:  material,
	}
}

func TestMaterialConsumoV3EntregaLasDiezEntradasCoherentes(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, err := escenario.material.ExportarMaterialParaConsumidor()
	if err != nil || exportacion.ValidarEstructura() != nil {
		t.Fatalf("exportar material válido: %v", err)
	}
	capacidad, _ := escenario.capacidad.ExportacionCanonicaParaConsumidor()
	resumen, _ := escenario.capacidad.ResumenParaConsumidor()
	decision, _ := domain.RepresentacionCanonicaDecisionAutorizacionV3(
		escenario.base.decision,
	)
	motivo, _ := domain.RepresentacionCanonicaMotivoAutorizacionV2(
		escenario.base.motivo,
	)
	solicitudFirma, _ := escenario.base.atestacion.Solicitud()
	payload, _ := solicitudFirma.Mensaje()
	resultadoFirma, _ := escenario.base.atestacion.Resultado()
	sobre, _ := resultadoFirma.Firma()
	evidencia, _ := escenario.prueba.ExportacionCanonicaParaConsumidor()
	spki, _ := x509.MarshalPKIXPublicKey(escenario.base.raiz.clavePublica)
	esperadas := [][]byte{
		capacidad,
		decision,
		motivo,
		escenario.base.resultado.RepresentacionCanonica,
		payload,
		sobre,
		evidencia,
		spki,
	}
	obtenidas := [][]byte{
		exportacion.CapacidadCanonica(),
		exportacion.DecisionCanonica(),
		exportacion.MotivoCanonico(),
		exportacion.ContextoActorCanonico(),
		exportacion.PayloadVECAD3(),
		exportacion.SobreCOSESign1(),
		exportacion.EvidenciaVerificacion(),
		exportacion.RaizPublicaSPKI(),
	}
	for indice := range esperadas {
		if !bytes.Equal(esperadas[indice], obtenidas[indice]) {
			t.Fatalf("pieza binaria %d no coincide", indice)
		}
	}
	instantanea := escenario.base.resultado.Contexto.Instantanea
	resumenObtenido := exportacion.ResumenCapacidad()
	if exportacion.PersonaVersion() != instantanea.PersonaVersion ||
		exportacion.PerfilVersion() != instantanea.PerfilVersion ||
		resumenObtenido.DecisionRef() != resumen.DecisionRef() ||
		resumenObtenido.DecisionHuellaSHA256() != resumen.DecisionHuellaSHA256() ||
		resumenObtenido.MotivoHuellaSHA256() != resumen.MotivoHuellaSHA256() ||
		resumenObtenido.ContextoRef() != resumen.ContextoRef() ||
		resumenObtenido.ContextoHuellaSHA256() != resumen.ContextoHuellaSHA256() ||
		resumenObtenido.Operacion() != resumen.Operacion() ||
		resumenObtenido.EfectoRef() != resumen.EfectoRef() ||
		resumenObtenido.EfectoHuellaSHA256() != resumen.EfectoHuellaSHA256() ||
		resumenObtenido.AudienciaConsumo() != resumen.AudienciaConsumo() ||
		!resumenObtenido.EmitidaEn().Equal(resumen.EmitidaEn()) ||
		!resumenObtenido.ExpiraEn().Equal(resumen.ExpiraEn()) {
		t.Fatal("versiones o resumen no proceden del conjunto nominal")
	}
	if len(exportacion.RaizPublicaSPKI()) !=
		ports.TamanoRaizPublicaSPKIEd25519V3 {
		t.Fatal("la raíz no usa DER-SPKI Ed25519 exacto")
	}
	huella, err := exportacion.HuellaConjuntoSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella del conjunto inválida: %q, %v", huella, err)
	}
}

func TestMaterialConsumoV3MantieneSnapshotYCopiasDefensivas(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	primera, err := escenario.material.ExportarMaterialParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	accesores := []func() []byte{
		primera.CapacidadCanonica,
		primera.DecisionCanonica,
		primera.MotivoCanonico,
		primera.ContextoActorCanonico,
		primera.PayloadVECAD3,
		primera.SobreCOSESign1,
		primera.EvidenciaVerificacion,
		primera.RaizPublicaSPKI,
	}
	for indice, acceder := range accesores {
		antes := acceder()
		mutada := acceder()
		mutada[len(mutada)/2] ^= 0xff
		if !bytes.Equal(antes, acceder()) {
			t.Fatalf("el accesor %d expuso el buffer interno", indice)
		}
	}
	segunda, err := escenario.material.ExportarMaterialParaConsumidor()
	if err != nil || segunda.ValidarEstructura() != nil ||
		!bytes.Equal(
			primera.CapacidadCanonica(),
			segunda.CapacidadCanonica(),
		) {
		t.Fatal("la exportación no fijó una instantánea estable")
	}
	if tipo := reflect.TypeOf(primera); tipo.Kind() != reflect.Struct {
		t.Fatalf("la exportación permite typed-nil: %s", tipo.Kind())
	}
}

func TestMaterialConsumoV3RechazaCrucesDeCapacidadYRaiz(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	contenido, _ := escenario.capacidad.ExportacionCanonicaParaConsumidor()
	base, _ := interpretarExportacionCapacidadV3(contenido)
	mutaciones := map[string]func(*capacidadAtestacionAutorizacionV3JSON){
		"decision": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.DecisionRef = "dec_otra23456789abcdef0123456789abcdef"
		},
		"huella_decision": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaDecisionSHA256 = strings.Repeat("5", 64)
		},
		"motivo": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaMotivoSHA256 = strings.Repeat("6", 64)
		},
		"contexto_ref": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.ContextoRef = "rca_otra23456789abcdefghijklmn"
		},
		"contexto_huella": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaContextoSHA256 = strings.Repeat("7", 64)
		},
		"payload": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaPayloadVECAD3SHA256 = strings.Repeat("1", 64)
		},
		"cose": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaSobreCOSESHA256 = strings.Repeat("2", 64)
		},
		"evidencia": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaPruebaSHA256 = strings.Repeat("3", 64)
		},
		"spki": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaRaizSPKISHA256 = strings.Repeat("4", 64)
		},
		"version_raiz": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.RaizVersion++
		},
		"kid_raiz": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.RaizClaveID = "clave:atestacion:v3:otra:2026-07"
		},
		"audiencia_raiz": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.AudienciaDespliegue = "vec-diputacion/pruebas/otro-despliegue"
		},
		"vigencia_raiz": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.RaizValidaDesde = escenario.base.raiz.validaDesde.
				Add(-time.Microsecond).
				Format(time.RFC3339Nano)
		},
		"operacion": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.Operacion = "contratacion_temporal.solicitud.retirar"
		},
		"efecto_ref": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.EfectoRef = "hmac-sha256:vec.efecto.otro/v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"efecto_huella": func(d *capacidadAtestacionAutorizacionV3JSON) {
			d.HuellaEfectoSHA256 = strings.Repeat("8", 64)
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			documento := base
			mutar(&documento)
			cruzada, err := nuevaCapacidadBreveAtestacionAutorizacionV3(documento)
			if err != nil {
				t.Fatalf("preparar cruce estructural: %v", err)
			}
			_, err = NuevoMaterialConsumoAutorizacionAtestadaV3(
				escenario.base.solicitud,
				escenario.base.decision,
				escenario.base.motivo,
				escenario.base.resultado,
				escenario.base.atestacion,
				escenario.prueba,
				cruzada,
				escenario.base.raiz,
			)
			if !errors.Is(
				err,
				ErrMaterialConsumoAutorizacionAtestadaV3Invalido,
			) {
				t.Fatalf("cruce %s aceptado: %v", nombre, err)
			}
		})
	}
	otraPrivada := ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat("w", ed25519.SeedSize)),
	)
	otraRaiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		escenario.base.raiz.claveID,
		escenario.base.raiz.version,
		otraPrivada.Public().(ed25519.PublicKey),
		escenario.base.raiz.audienciaDespliegue,
		escenario.base.raiz.estado,
		escenario.base.raiz.validaDesde,
		escenario.base.raiz.validaHasta,
		time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NuevoMaterialConsumoAutorizacionAtestadaV3(
		escenario.base.solicitud,
		escenario.base.decision,
		escenario.base.motivo,
		escenario.base.resultado,
		escenario.base.atestacion,
		escenario.prueba,
		escenario.capacidad,
		otraRaiz,
	)
	if !errors.Is(err, ErrMaterialConsumoAutorizacionAtestadaV3Invalido) {
		t.Fatalf("raíz A/B aceptada: %v", err)
	}
	mutacionesRaiz := map[string]func(*RaizPublicaAtestacionAutorizacionV3){
		"kid": func(r *RaizPublicaAtestacionAutorizacionV3) {
			r.claveID = "clave:atestacion:v3:otra:2026-07"
		},
		"version": func(r *RaizPublicaAtestacionAutorizacionV3) {
			r.version++
		},
		"audiencia": func(r *RaizPublicaAtestacionAutorizacionV3) {
			r.audienciaDespliegue = "vec-diputacion/pruebas/otro-despliegue"
		},
		"ventana": func(r *RaizPublicaAtestacionAutorizacionV3) {
			r.validaDesde = r.validaDesde.Add(-time.Microsecond)
		},
		"estado": func(r *RaizPublicaAtestacionAutorizacionV3) {
			r.estado = EstadoClaveAtestacionAutorizacionV3Revocada
			r.revocadaEn = escenario.base.ahora.Add(-30 * time.Minute)
		},
	}
	for nombre, mutar := range mutacionesRaiz {
		t.Run("raiz_"+nombre, func(t *testing.T) {
			otra := escenario.base.raiz
			otra.clavePublica = bytes.Clone(escenario.base.raiz.clavePublica)
			mutar(&otra)
			if otra.validar() != nil {
				t.Fatal("la raíz cruzada debía ser nominalmente válida")
			}
			_, err := NuevoMaterialConsumoAutorizacionAtestadaV3(
				escenario.base.solicitud,
				escenario.base.decision,
				escenario.base.motivo,
				escenario.base.resultado,
				escenario.base.atestacion,
				escenario.prueba,
				escenario.capacidad,
				otra,
			)
			if !errors.Is(
				err,
				ErrMaterialConsumoAutorizacionAtestadaV3Invalido,
			) {
				t.Fatalf("ligadura de raíz %s aceptada: %v", nombre, err)
			}
		})
	}
}

func TestMaterialConsumoV3RechazaAtestacionYPruebaNominalesAjenas(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	otraPrivada := ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat("x", ed25519.SeedSize)),
	)
	otraRaiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		escenario.base.raiz.claveID,
		escenario.base.raiz.version,
		otraPrivada.Public().(ed25519.PublicKey),
		escenario.base.raiz.audienciaDespliegue,
		EstadoClaveAtestacionAutorizacionV3Activa,
		escenario.base.raiz.validaDesde,
		escenario.base.raiz.validaHasta,
		time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	otraConfiguracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:revision:otra",
		escenario.base.configuracion.secuencia+1,
		escenario.base.ahora.Add(-time.Minute),
		escenario.base.ahora.Add(30*time.Minute),
		otraRaiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	otroServicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		otraConfiguracion,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.base.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        otraRaiz.claveID,
		Audiencia:      otraRaiz.audienciaDespliegue,
	}
	otraAtestacion := atestacionConfianzaAtestacionV3Prueba(
		t,
		cabecera,
		escenario.base.decision,
		escenario.base.motivo,
		escenario.base.resultado,
		otraPrivada,
		escenario.base.ahora,
	)
	otraPrueba, err := otroServicio.Verificar(
		context.Background(),
		escenario.base.solicitud,
		escenario.base.decision,
		escenario.base.motivo,
		escenario.base.resultado,
		otraAtestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, caso := range map[string]struct {
		atestacion ports.AtestacionAutorizacionV3
		prueba     PruebaConfianzaAtestacionAutorizacionV3
		raiz       RaizPublicaAtestacionAutorizacionV3
	}{
		"atestacion_b_prueba_a": {
			atestacion: otraAtestacion,
			prueba:     escenario.prueba,
			raiz:       escenario.base.raiz,
		},
		"atestacion_b_prueba_b_capacidad_a": {
			atestacion: otraAtestacion,
			prueba:     otraPrueba,
			raiz:       otraRaiz,
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			_, err := NuevoMaterialConsumoAutorizacionAtestadaV3(
				escenario.base.solicitud,
				escenario.base.decision,
				escenario.base.motivo,
				escenario.base.resultado,
				caso.atestacion,
				caso.prueba,
				escenario.capacidad,
				caso.raiz,
			)
			if !errors.Is(
				err,
				ErrMaterialConsumoAutorizacionAtestadaV3Invalido,
			) {
				t.Fatalf("cruce nominal aceptado: %v", err)
			}
		})
	}
}

func TestMaterialConsumoV3DetectaAlteracionInternaDeCadaPrueba(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	resumen := escenario.material.entradas.resumenCapacidad
	otroResumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		resumen.DecisionRef(),
		resumen.DecisionHuellaSHA256(),
		resumen.MotivoHuellaSHA256(),
		resumen.ContextoRef(),
		resumen.ContextoHuellaSHA256(),
		resumen.Operacion(),
		"hmac-sha256:vec.efecto.otro/v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		resumen.EfectoHuellaSHA256(),
		resumen.AudienciaConsumo(),
		resumen.EmitidaEn(),
		resumen.ExpiraEn(),
	)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*MaterialConsumoAutorizacionAtestadaV3){
		"capacidad": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.capacidadCanonica[0] ^= 1
		},
		"decision": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.decisionCanonica[0] ^= 1
		},
		"motivo": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.motivoCanonico[0] ^= 1
		},
		"contexto": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.contextoActorCanonico[0] ^= 1
		},
		"persona_version": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.personaVersion++
		},
		"perfil_version": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.perfilVersion++
		},
		"resumen": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.resumenCapacidad = otroResumen
		},
		"payload": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.payloadVECAD3[0] ^= 1
		},
		"cose": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.sobreCOSESign1[0] ^= 1
		},
		"evidencia": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.evidenciaVerificacion[0] ^= 1
		},
		"spki": func(m *MaterialConsumoAutorizacionAtestadaV3) {
			m.entradas.raizPublicaSPKI[0] ^= 1
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			material := escenario.material
			material.entradas = clonarEntradasMaterialConsumoV3(
				escenario.material.entradas,
			)
			mutar(&material)
			if _, err := material.ExportarMaterialParaConsumidor(); !errors.Is(
				err,
				ErrMaterialConsumoAutorizacionAtestadaV3Invalido,
			) {
				t.Fatalf("alteración interna %s aceptada: %v", nombre, err)
			}
		})
	}
}

func TestMaterialConsumoV3BloqueaCodecsFormatosYLogs(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, _ := escenario.material.ExportarMaterialParaConsumidor()
	secreto := string(exportacion.CapacidadCanonica()[:32])
	for nombre, valor := range map[string]any{
		"material":    escenario.material,
		"exportacion": exportacion,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); err == nil {
				t.Fatal("JSON aceptó material opaco")
			}
			if _, err := xml.Marshal(valor); err == nil {
				t.Fatal("XML aceptó material opaco")
			}
			for _, formato := range []string{
				"%v", "%+v", "%#v", "%s", "%q", "%x", "%X",
			} {
				if salida := fmt.Sprintf(formato, valor); strings.Contains(
					salida,
					secreto,
				) {
					t.Fatalf("formato %s filtró material", formato)
				}
			}
			var registro bytes.Buffer
			slog.New(slog.NewJSONHandler(&registro, nil)).Info(
				"material",
				"valor",
				valor,
			)
			if strings.Contains(registro.String(), secreto) {
				t.Fatal("slog filtró material")
			}
		})
	}
	errores := []error{}
	_, errTexto := exportacion.MarshalText()
	errores = append(errores, errTexto)
	_, errBinario := exportacion.MarshalBinary()
	errores = append(errores, errBinario)
	_, errGob := exportacion.GobEncode()
	errores = append(errores, errGob)
	_, errCBOR := exportacion.MarshalCBOR()
	errores = append(errores, errCBOR)
	_, errYAML := exportacion.MarshalYAML()
	errores = append(errores, errYAML)
	for indice, err := range errores {
		if err == nil {
			t.Fatalf("codec %d aceptó material", indice)
		}
	}
}

func TestMaterialConsumoV3RechazaCerosYLimitesFueraDeSQL(
	t *testing.T,
) {
	t.Parallel()
	var material MaterialConsumoAutorizacionAtestadaV3
	if _, err := material.ExportarMaterialParaConsumidor(); !errors.Is(
		err,
		ErrMaterialConsumoAutorizacionAtestadaV3Invalido,
	) {
		t.Fatalf("material cero aceptado: %v", err)
	}
	var exportacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if exportacion.ValidarEstructura() == nil {
		t.Fatal("exportación cero aceptada")
	}
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	valida, _ := escenario.material.ExportarMaterialParaConsumidor()
	_, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		valida.CapacidadCanonica(),
		valida.ResumenCapacidad(),
		valida.DecisionCanonica(),
		valida.MotivoCanonico(),
		valida.ContextoActorCanonico(),
		ports.VersionMaximaExactaMaterialConsumoV3+1,
		valida.PerfilVersion(),
		valida.PayloadVECAD3(),
		valida.SobreCOSESign1(),
		valida.EvidenciaVerificacion(),
		valida.RaizPublicaSPKI(),
	)
	if !errors.Is(
		err,
		ports.ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida,
	) {
		t.Fatalf("versión fuera de SQL aceptada: %v", err)
	}
}
