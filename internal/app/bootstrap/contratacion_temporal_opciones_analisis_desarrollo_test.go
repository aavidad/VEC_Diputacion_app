package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojOpcionesAnalisisPrueba struct{}

func (relojOpcionesAnalisisPrueba) Ahora() time.Time {
	return time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
}

func resolutorReglasCTPrueba(t *testing.T, ruta string) *reglas.Resolutor {
	t.Helper()
	resolutor, err := nuevoResolutorReglasEjemplo(
		ruta, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, relojOpcionesAnalisisPrueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

// catalogoReglasCTModificadoPrueba copia el paquete de ejemplo y deja que la
// prueba cambie sus entradas, como haría RRHH al publicar otra versión.
func catalogoReglasCTModificadoPrueba(t *testing.T, cambiar func([]map[string]any) []map[string]any) string {
	t.Helper()
	contenido, err := os.ReadFile(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	catalogo := paquete["catalogo"].(map[string]any)
	var entradas []map[string]any
	for _, entrada := range catalogo["entradas"].([]any) {
		entradas = append(entradas, entrada.(map[string]any))
	}
	entradas = cambiar(entradas)
	lista := make([]any, 0, len(entradas))
	for _, entrada := range entradas {
		lista = append(lista, entrada)
	}
	catalogo["entradas"] = lista
	salida, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "ct_reglas.demo.json")
	if err := os.WriteFile(ruta, salida, 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func entradaReglaCTPrueba(clave, etiqueta string, atributos map[string]any) map[string]any {
	base := map[string]any{"origen": "ejemplo", "norma": "Supuesto de trabajo.", "duda": "Duda 7.", "unidad": "ninguna"}
	for clave, valor := range atributos {
		base[clave] = valor
	}
	return map[string]any{
		"clave": clave, "etiqueta": etiqueta, "vigente_desde": "2026-02-01T00:00:00Z",
		"orden": 99, "atributos": base,
	}
}

func TestOpcionesAnalisisSinCatalogoSonLasDeSiempre(t *testing.T) {
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(opciones.modalidades) != 5 || len(opciones.causas) != 1 || len(opciones.entradasRC) != 1 ||
		opciones.urgenciaDisponible {
		t.Fatalf("opciones inesperadas: %+v", opciones)
	}
	for _, modalidad := range opciones.modalidades {
		if modalidad.Duracion != nil {
			t.Fatalf("sin catálogo no hay duraciones máximas: %+v", modalidad)
		}
	}
	configuracion := nuevaConfiguracionAnalisisContratacionTemporalDesarrollo(nil)
	contenido, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]any
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatal(err)
	}
	if _, existe := campos["duraciones_maximas"]; existe {
		t.Fatal("sin catálogo la configuración no publica duraciones")
	}
	if _, existe := campos["urgencia_disponible"]; existe {
		t.Fatal("sin catálogo la configuración no ofrece urgencia")
	}
}

func TestOpcionesAnalisisDelPaqueteDeEjemploConservanLasDeSiempre(t *testing.T) {
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, rutaReglasCTEjemploPrueba))
	if err != nil {
		t.Fatal(err)
	}
	predeterminadas := opcionesAnalisisPredeterminadas()
	if len(opciones.modalidades) != len(predeterminadas.modalidades) || !opciones.urgenciaDisponible {
		t.Fatalf("opciones inesperadas: %+v", opciones)
	}
	conMaximo := map[domain.ClaveCatalogo]reglas.Unidad{
		"vacante": reglas.UnidadAnios, "acumulacion_tareas": reglas.UnidadMeses, "programa": reglas.UnidadAnios,
	}
	for indice, modalidad := range opciones.modalidades {
		if modalidad.Clave != predeterminadas.modalidades[indice].Clave ||
			modalidad.Etiqueta != predeterminadas.modalidades[indice].Etiqueta {
			t.Fatalf("modalidad %d distinta de la de siempre: %+v", indice, modalidad)
		}
		unidad, debeTener := conMaximo[modalidad.Clave]
		if debeTener != (modalidad.Duracion != nil) || (debeTener && (modalidad.Duracion.Unidad != unidad ||
			modalidad.Duracion.Bloquear || modalidad.Duracion.ReglaRef == "")) {
			t.Fatalf("duración inesperada en %s: %+v", modalidad.Clave, modalidad.Duracion)
		}
	}
	if opciones.causas[0] != predeterminadas.causas[0] ||
		opciones.entradasRC[0] != predeterminadas.entradasRC[0] {
		t.Fatalf("causa o retención de crédito distinta de la de siempre: %+v %+v", opciones.causas, opciones.entradasRC)
	}
}

func TestOpcionesAnalisisSeCambianEnElCatalogo(t *testing.T) {
	ruta := catalogoReglasCTModificadoPrueba(t, func(entradas []map[string]any) []map[string]any {
		for _, entrada := range entradas {
			if entrada["clave"] == "c08.acumulacion_tareas" {
				entrada["atributos"].(map[string]any)["al_superar"] = "bloquear"
			}
		}
		return append(entradas,
			entradaReglaCTPrueba("c12.modalidad.interinidad_programa", "Interinidad por programa", map[string]any{"duracion_regla": "c08.circunstancias_produccion"}),
			entradaReglaCTPrueba("c13.causa.refuerzo_verano", "Refuerzo de verano", nil),
		)
	})
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, ruta))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	catalogo.componerOpcionesAnalisis(opciones)
	configuracion := nuevaConfiguracionAnalisisContratacionTemporalDesarrollo(nil, catalogo)
	if len(configuracion.Modalidades) != 6 || configuracion.Modalidades[5].Clave != "interinidad_programa" ||
		len(configuracion.Causas) != 2 || !configuracion.UrgenciaDisponible {
		t.Fatalf("configuración inesperada: %+v", configuracion)
	}
	bloquea := false
	for _, duracion := range configuracion.DuracionesMaximas {
		if duracion.ModalidadClave == "acumulacion_tareas" {
			bloquea = duracion.Bloquear && duracion.Unidad == "meses" && duracion.Cantidad == 9
		}
	}
	if !bloquea {
		t.Fatalf("la acumulación de tareas debía bloquear: %+v", configuracion.DuracionesMaximas)
	}
	solicitud := solicitudPrepararArtefactoAnalisisDesarrolloPrueba("interinidad_programa", "expediente:ct:desarrollo:opciones")
	solicitud.DatosFuncionales.CausaClave = "refuerzo_verano"
	if !solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(solicitud, catalogo) {
		t.Fatal("la modalidad y la causa nuevas del catálogo deben admitirse")
	}
	if solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(solicitud, nil) {
		t.Fatal("sin el catálogo la modalidad nueva no existe")
	}
	larga := solicitudPrepararArtefactoAnalisisDesarrolloPrueba("acumulacion_tareas", "expediente:ct:desarrollo:opciones")
	larga.DatosFuncionales.Periodo = periodoAnalisisPrueba(t, "2026-01-01", "2026-10-01")
	if solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(larga, catalogo) {
		t.Fatal("un periodo que supera un máximo que bloquea debe rechazarse")
	}
	larga.DatosFuncionales.Periodo = periodoAnalisisPrueba(t, "2026-01-01", "2026-09-30")
	if !solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(larga, catalogo) {
		t.Fatal("nueve meses exactos no superan el máximo")
	}
	aviso := solicitudPrepararArtefactoAnalisisDesarrolloPrueba("vacante", "expediente:ct:desarrollo:opciones")
	aviso.DatosFuncionales.Periodo = periodoAnalisisPrueba(t, "2026-01-01", "2030-01-01")
	if !solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(aviso, catalogo) {
		t.Fatal("un máximo que solo avisa no impide registrar")
	}
}

func TestOpcionesAnalisisRechazanCatalogoRoto(t *testing.T) {
	casos := map[string]map[string]any{
		"duración inexistente": entradaReglaCTPrueba("c12.modalidad.rota", "Rota", map[string]any{"duracion_regla": "c08.no_existe"}),
		"duración sin plazo":   entradaReglaCTPrueba("c12.modalidad.rota", "Rota", map[string]any{"duracion_regla": "c07.jornada_completa"}),
		"clave no canónica":    entradaReglaCTPrueba("c13.causa.9_empieza_por_cifra", "Cifra", nil),
		"RC repetida": entradaReglaCTPrueba("c14.entrada_rc.repetida", "RC repetida", map[string]any{
			"referencia": "rc:desarrollo:001", "huella_sha256": "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b",
			"numero": "rc:desarrollo:numero:001", "fecha": "2026-01-02", "importe_centimos": "5000000",
			"documento_ref": "documento:rc:desarrollo:001",
		}),
		"RC sin importe": entradaReglaCTPrueba("c14.entrada_rc.rota", "RC rota", map[string]any{
			"referencia": "rc:desarrollo:002", "huella_sha256": "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b",
			"numero": "rc:desarrollo:numero:002", "fecha": "2026-01-02", "documento_ref": "documento:rc:desarrollo:002",
		}),
	}
	for nombre, extra := range casos {
		t.Run(nombre, func(t *testing.T) {
			ruta := catalogoReglasCTModificadoPrueba(t, func(entradas []map[string]any) []map[string]any {
				return append(entradas, extra)
			})
			if _, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, ruta)); err == nil {
				t.Fatal("un catálogo roto no puede componerse")
			}
		})
	}
}

func TestDuracionMaximaCuentaDeFechaAFecha(t *testing.T) {
	casos := []struct {
		duracion duracionMaximaAnalisisCT
		inicio   string
		fin      string
		supera   bool
	}{
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadMeses, Cantidad: 9}, "2026-01-01", "2026-09-30", false},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadMeses, Cantidad: 9}, "2026-01-01", "2026-10-01", true},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadMeses, Cantidad: 1}, "2026-01-31", "2026-02-27", false},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadMeses, Cantidad: 1}, "2026-01-31", "2026-02-28", true},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadAnios, Cantidad: 3}, "2026-03-01", "2029-02-28", false},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadAnios, Cantidad: 3}, "2026-03-01", "2029-03-01", true},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadDiasNaturales, Cantidad: 10}, "2026-03-01", "2026-03-10", false},
		{duracionMaximaAnalisisCT{Unidad: reglas.UnidadDiasNaturales, Cantidad: 10}, "2026-03-01", "2026-03-11", true},
	}
	for _, caso := range casos {
		if supera := caso.duracion.superaDuracion(periodoAnalisisPrueba(t, caso.inicio, caso.fin)); supera != caso.supera {
			t.Errorf("%+v %s..%s: supera=%v", caso.duracion, caso.inicio, caso.fin, supera)
		}
	}
	var nula *duracionMaximaAnalisisCT
	if nula.superaDuracion(periodoAnalisisPrueba(t, "2026-01-01", "2099-01-01")) {
		t.Fatal("sin máximo nada lo supera")
	}
}

func periodoAnalisisPrueba(t *testing.T, inicio, fin string) domain.PeriodoPrevisto {
	t.Helper()
	desde, errDesde := time.Parse(time.DateOnly, inicio)
	hasta, errHasta := time.Parse(time.DateOnly, fin)
	if errDesde != nil || errHasta != nil {
		t.Fatal("fecha de prueba no válida")
	}
	return domain.PeriodoPrevisto{Inicio: desde.UTC(), Fin: hasta.UTC()}
}

func TestUrgenciaAnalisisSoloConLaReglaDelCatalogo(t *testing.T) {
	solicitud := solicitudPrepararArtefactoAnalisisDesarrolloPrueba("vacante", "expediente:ct:desarrollo:urgencia")
	solicitud.DatosFuncionales.MotivoUrgencia = "Cierre del centro de día si no se cubre la plaza antes del lunes."
	if solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(solicitud, nil) {
		t.Fatal("sin la regla c15 no se admite declarar la urgencia")
	}
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, rutaReglasCTEjemploPrueba))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	catalogo.componerOpcionesAnalisis(opciones)
	if !solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(solicitud, catalogo) {
		t.Fatal("con la regla c15 la urgencia se admite")
	}
}

type consultaMigracionPrueba struct {
	instalada bool
	err       error
}

func (c consultaMigracionPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaMigracionPrueba(c)
}

type filaMigracionPrueba consultaMigracionPrueba

func (f filaMigracionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.instalada
	return nil
}

func TestArranqueExigeLaMigracionDeUrgencia(t *testing.T) {
	if err := comprobarMigracionUrgenciaAnalisis(t.Context(), consultaMigracionPrueba{instalada: true}); err != nil {
		t.Fatal(err)
	}
	for _, consulta := range []consultaMigracionPrueba{{instalada: false}, {err: errors.New("sin conexión")}} {
		if err := comprobarMigracionUrgenciaAnalisis(t.Context(), consulta); !errors.Is(err, errMigracionUrgenciaAnalisisNoInstalada) {
			t.Fatalf("arrancaría sin CT-000125: %v", err)
		}
	}
}
