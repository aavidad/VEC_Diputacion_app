package domain

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGINPIXSerializacionDeterministaYLigaduras(t *testing.T) {
	modeloA := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
		{Clave: "observaciones", Campo: CampoNuloGINPIX()},
		{Clave: "codigo_puesto", Campo: campoValorGINPIXPrueba(t, "PT-SINT-07")},
		{Clave: "codigo_relacion", Campo: CampoAusenteGINPIX()},
	})
	modeloB := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
		{Clave: "codigo_relacion", Campo: CampoAusenteGINPIX()},
		{Clave: "codigo_puesto", Campo: campoValorGINPIXPrueba(t, "PT-SINT-07")},
		{Clave: "observaciones", Campo: CampoNuloGINPIX()},
	})
	bytesA, errA := modeloA.SerializarCanonico()
	bytesB, errB := modeloB.SerializarCanonico()
	if errA != nil || errB != nil || !bytes.Equal(bytesA, bytesB) ||
		modeloA.HuellaSHA256() != modeloB.HuellaSHA256() {
		t.Fatalf("modelo canónico no determinista: %v / %v", errA, errB)
	}

	mapeoA := mapeoGINPIXPrueba(t, []ReglaMapeoGINPIX{
		{CampoCanonico: "observaciones", CampoDestino: "obs", PermiteNulo: true},
		{CampoCanonico: "codigo_relacion", CampoDestino: "relacion", PermiteNulo: true},
		{CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true},
	})
	mapeoB := mapeoGINPIXPrueba(t, []ReglaMapeoGINPIX{
		{CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true},
		{CampoCanonico: "codigo_relacion", CampoDestino: "relacion", PermiteNulo: true},
		{CampoCanonico: "observaciones", CampoDestino: "obs", PermiteNulo: true},
	})
	serialMapeoA, _ := mapeoA.SerializarCanonico()
	serialMapeoB, _ := mapeoB.SerializarCanonico()
	if !bytes.Equal(serialMapeoA, serialMapeoB) ||
		mapeoA.Publicacion().HuellaSHA256 != mapeoB.Publicacion().HuellaSHA256 {
		t.Fatal("mapeo versionado no determinista")
	}

	cargaA, err := AplicarMapeoGINPIX(modeloA, mapeoA)
	if err != nil {
		t.Fatalf("aplicar mapeo sintético: %v", err)
	}
	cargaB, err := AplicarMapeoGINPIX(modeloB, mapeoB)
	if err != nil {
		t.Fatalf("aplicar mapeo reordenado: %v", err)
	}
	serialCargaA, _ := cargaA.SerializarCanonico()
	serialCargaB, _ := cargaB.SerializarCanonico()
	if !bytes.Equal(serialCargaA, serialCargaB) ||
		cargaA.Datos().HuellaSHA256 != cargaB.Datos().HuellaSHA256 {
		t.Fatal("carga mapeada no determinista")
	}
	if got, want := cargaA.Datos().HuellaSHA256, "ca715942053b0234b4f4fba6b5de10921ce806bb36ae8093b7171fa0dc69ff9e"; got != want {
		t.Fatalf("vector canónico GINPIX divergente: got %s want %s", got, want)
	}

	datos := cargaA.Datos()
	if datos.ExpedienteRef != "expediente_sintetico_ginpix_01" ||
		datos.IncorporacionRef != "incorporacion_sintetica_ginpix_01" ||
		datos.ProcedenciaModeloRef != "procedencia_sintetica_modelo_ginpix_01" ||
		datos.CorrelacionRef != "correlacion_sintetica_ginpix_01" ||
		datos.IdempotenciaRef != "idempotencia_sintetica_ginpix_01" ||
		datos.MapeoRef != "mapeo_sintetico_ginpix_01" || datos.MapeoVersion != 3 ||
		datos.ProcedenciaMapeoRef != "procedencia_sintetica_mapeo_ginpix_01" {
		t.Fatalf("se perdieron ligaduras opacas/versionadas: %+v", datos)
	}
}

func TestGINPIXAusenteNuloYVacioSonDistintos(t *testing.T) {
	estados := []CampoGINPIX{
		CampoAusenteGINPIX(),
		CampoNuloGINPIX(),
		campoValorGINPIXPrueba(t, ""),
	}
	serializaciones := make(map[string]struct{}, len(estados))
	huellas := make(map[string]struct{}, len(estados))
	for _, estado := range estados {
		modelo := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
			{Clave: "observaciones", Campo: estado},
		})
		serial, err := modelo.SerializarCanonico()
		if err != nil {
			t.Fatalf("serializar estado %d: %v", estado.Estado, err)
		}
		serializaciones[string(serial)] = struct{}{}
		huellas[modelo.HuellaSHA256()] = struct{}{}
	}
	if len(serializaciones) != 3 || len(huellas) != 3 {
		t.Fatalf("ausente, nulo y vacío colisionaron: %d/%d", len(serializaciones), len(huellas))
	}
}

func TestGINPIXEsquemasDesconocidosYHuellasAdulteradasSeDeniegan(t *testing.T) {
	borradorModelo := borradorModeloGINPIXPrueba([]DatoCanonicoGINPIX{
		{Clave: "codigo_puesto", Campo: campoValorGINPIXPrueba(t, "PT-SINT-08")},
	})
	borradorModelo.Esquema = "vec.dipgra.contratacion-temporal.ginpix.modelo.v99"
	if _, err := NuevoModeloCanonicoGINPIX(borradorModelo); !errors.Is(err, ErrModeloGINPIXInvalido) {
		t.Fatalf("esquema de modelo desconocido aceptado: %v", err)
	}
	borradorMapeo := borradorMapeoGINPIXPrueba([]ReglaMapeoGINPIX{
		{CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true},
	})
	borradorMapeo.Esquema = "vec.dipgra.contratacion-temporal.ginpix.mapeo.v99"
	if _, err := PublicarMapeoVersionadoGINPIX(borradorMapeo); !errors.Is(err, ErrMapeoGINPIXInvalido) {
		t.Fatalf("esquema de mapeo desconocido aceptado: %v", err)
	}

	modelo := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
		{Clave: "codigo_puesto", Campo: campoValorGINPIXPrueba(t, "PT-SINT-08")},
	})
	publicacionModelo := modelo.Publicacion()
	publicacionModelo.HuellaSHA256 = strings.Repeat("a", 64)
	if _, err := RestaurarModeloCanonicoGINPIX(publicacionModelo); !errors.Is(err, ErrModeloGINPIXInvalido) {
		t.Fatalf("huella de modelo adulterada aceptada: %v", err)
	}
	mapeo := mapeoGINPIXPrueba(t, []ReglaMapeoGINPIX{
		{CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true},
	})
	publicacionMapeo := mapeo.Publicacion()
	publicacionMapeo.HuellaSHA256 = strings.Repeat("b", 64)
	if _, err := RestaurarMapeoVersionadoGINPIX(publicacionMapeo); !errors.Is(err, ErrMapeoGINPIXInvalido) {
		t.Fatalf("huella de mapeo adulterada aceptada: %v", err)
	}
}

func TestGINPIXCompatibilidadFallaCerrada(t *testing.T) {
	casos := []struct {
		nombre string
		campo  CampoGINPIX
		regla  ReglaMapeoGINPIX
		valido bool
	}{
		{"ausente_opcional", CampoAusenteGINPIX(), ReglaMapeoGINPIX{}, true},
		{"ausente_obligatorio", CampoAusenteGINPIX(), ReglaMapeoGINPIX{Obligatorio: true}, false},
		{"nulo_permitido", CampoNuloGINPIX(), ReglaMapeoGINPIX{PermiteNulo: true}, true},
		{"nulo_denegado", CampoNuloGINPIX(), ReglaMapeoGINPIX{}, false},
		{"vacio_permitido", campoValorGINPIXPrueba(t, ""), ReglaMapeoGINPIX{PermiteVacio: true}, true},
		{"vacio_denegado", campoValorGINPIXPrueba(t, ""), ReglaMapeoGINPIX{}, false},
		{"valor", campoValorGINPIXPrueba(t, "SINT-01"), ReglaMapeoGINPIX{Obligatorio: true}, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			modelo := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
				{Clave: "campo_sintetico", Campo: caso.campo},
			})
			regla := caso.regla
			regla.CampoCanonico = "campo_sintetico"
			regla.CampoDestino = "destino_sintetico"
			mapeo := mapeoGINPIXPrueba(t, []ReglaMapeoGINPIX{regla})
			_, err := AplicarMapeoGINPIX(modelo, mapeo)
			if caso.valido && err != nil {
				t.Fatalf("compatibilidad válida denegada: %v", err)
			}
			if !caso.valido && !errors.Is(err, ErrCompatibilidadGINPIXDenegada) {
				t.Fatalf("incompatibilidad aceptada: %v", err)
			}
		})
	}

	modelo := modeloGINPIXPrueba(t, []DatoCanonicoGINPIX{
		{Clave: "campo_a", Campo: CampoAusenteGINPIX()},
		{Clave: "campo_b", Campo: CampoAusenteGINPIX()},
	})
	mapeo := mapeoGINPIXPrueba(t, []ReglaMapeoGINPIX{
		{CampoCanonico: "campo_a", CampoDestino: "destino_a"},
	})
	if _, err := AplicarMapeoGINPIX(modelo, mapeo); !errors.Is(err, ErrCompatibilidadGINPIXDenegada) {
		t.Fatalf("campo sin regla se omitió silenciosamente: %v", err)
	}
}

func TestGINPIXLimitesDuplicadosYReferenciasOpacas(t *testing.T) {
	datos := make([]DatoCanonicoGINPIX, maximoCamposGINPIX+1)
	for indice := range datos {
		datos[indice] = DatoCanonicoGINPIX{
			Clave: ClaveCatalogo(fmt.Sprintf("campo_%03d", indice)),
			Campo: CampoAusenteGINPIX(),
		}
	}
	if _, err := NuevoModeloCanonicoGINPIX(borradorModeloGINPIXPrueba(datos)); !errors.Is(err, ErrModeloGINPIXInvalido) {
		t.Fatalf("cardinalidad fuera de límite aceptada: %v", err)
	}
	duplicados := []DatoCanonicoGINPIX{
		{Clave: "campo_a", Campo: CampoAusenteGINPIX()},
		{Clave: "campo_a", Campo: CampoNuloGINPIX()},
	}
	if _, err := NuevoModeloCanonicoGINPIX(borradorModeloGINPIXPrueba(duplicados)); !errors.Is(err, ErrModeloGINPIXInvalido) {
		t.Fatalf("clave canónica duplicada aceptada: %v", err)
	}
	borrador := borradorModeloGINPIXPrueba([]DatoCanonicoGINPIX{
		{Clave: "campo_a", Campo: CampoAusenteGINPIX()},
	})
	borrador.CorrelacionRef = "dato personal no opaco"
	if _, err := NuevoModeloCanonicoGINPIX(borrador); !errors.Is(err, ErrModeloGINPIXInvalido) {
		t.Fatalf("correlación no opaca aceptada: %v", err)
	}

	reglas := []ReglaMapeoGINPIX{
		{CampoCanonico: "campo_a", CampoDestino: "destino_a"},
		{CampoCanonico: "campo_b", CampoDestino: "destino_a"},
	}
	if _, err := PublicarMapeoVersionadoGINPIX(borradorMapeoGINPIXPrueba(reglas)); !errors.Is(err, ErrMapeoGINPIXInvalido) {
		t.Fatalf("destino duplicado aceptado: %v", err)
	}
}

func TestGINPIXCopiasDefensivas(t *testing.T) {
	entrada := []DatoCanonicoGINPIX{
		{Clave: "campo_a", Campo: campoValorGINPIXPrueba(t, "SINT-A")},
	}
	modelo := modeloGINPIXPrueba(t, entrada)
	serialOriginal, _ := modelo.SerializarCanonico()
	entrada[0].Campo.Valor = "MUTADO"
	snapshot := modelo.Publicacion()
	snapshot.Datos[0].Campo.Valor = "MUTADO-2"
	serialPosterior, _ := modelo.SerializarCanonico()
	if !bytes.Equal(serialOriginal, serialPosterior) {
		t.Fatal("modelo expuso una colección mutable")
	}

	reglas := []ReglaMapeoGINPIX{
		{CampoCanonico: "campo_a", CampoDestino: "destino_a", Obligatorio: true},
	}
	mapeo := mapeoGINPIXPrueba(t, reglas)
	serialMapeo, _ := mapeo.SerializarCanonico()
	reglas[0].CampoDestino = "mutado"
	snapshotMapeo := mapeo.Publicacion()
	snapshotMapeo.Reglas[0].CampoDestino = "mutado_2"
	serialMapeoPosterior, _ := mapeo.SerializarCanonico()
	if !bytes.Equal(serialMapeo, serialMapeoPosterior) {
		t.Fatal("mapeo expuso una colección mutable")
	}

	carga, err := AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatalf("crear carga: %v", err)
	}
	serialCarga, _ := carga.SerializarCanonico()
	datosCarga := carga.Datos()
	datosCarga.Campos[0].Campo.Valor = "MUTADO-3"
	serialCargaPosterior, _ := carga.SerializarCanonico()
	if !bytes.Equal(serialCarga, serialCargaPosterior) {
		t.Fatal("carga expuso una colección mutable")
	}
}

func modeloGINPIXPrueba(t *testing.T, datos []DatoCanonicoGINPIX) ModeloCanonicoGINPIX {
	t.Helper()
	modelo, err := NuevoModeloCanonicoGINPIX(borradorModeloGINPIXPrueba(datos))
	if err != nil {
		t.Fatalf("crear modelo GINPIX sintético: %v", err)
	}
	return modelo
}

func borradorModeloGINPIXPrueba(datos []DatoCanonicoGINPIX) BorradorModeloCanonicoGINPIX {
	return BorradorModeloCanonicoGINPIX{
		Esquema:           EsquemaModeloCanonicoGINPIXV1,
		VersionExpediente: 7,
		ExpedienteRef:     "expediente_sintetico_ginpix_01",
		IncorporacionRef:  "incorporacion_sintetica_ginpix_01",
		ProcedenciaRef:    "procedencia_sintetica_modelo_ginpix_01",
		CorrelacionRef:    "correlacion_sintetica_ginpix_01",
		IdempotenciaRef:   "idempotencia_sintetica_ginpix_01",
		Datos:             datos,
	}
}

func mapeoGINPIXPrueba(t *testing.T, reglas []ReglaMapeoGINPIX) MapeoVersionadoGINPIX {
	t.Helper()
	mapeo, err := PublicarMapeoVersionadoGINPIX(borradorMapeoGINPIXPrueba(reglas))
	if err != nil {
		t.Fatalf("crear mapeo GINPIX sintético: %v", err)
	}
	return mapeo
}

func borradorMapeoGINPIXPrueba(reglas []ReglaMapeoGINPIX) BorradorMapeoVersionadoGINPIX {
	return BorradorMapeoVersionadoGINPIX{
		Esquema:        EsquemaMapeoGINPIXV1,
		Referencia:     "mapeo_sintetico_ginpix_01",
		Version:        3,
		ProcedenciaRef: "procedencia_sintetica_mapeo_ginpix_01",
		Reglas:         reglas,
	}
}

func campoValorGINPIXPrueba(t *testing.T, valor string) CampoGINPIX {
	t.Helper()
	campo, err := CampoValorGINPIX(valor)
	if err != nil {
		t.Fatalf("crear campo GINPIX sintético: %v", err)
	}
	return campo
}
